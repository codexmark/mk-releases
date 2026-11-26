// SPDX-License-Identifier: GPL-3.0-or-later

package innertube

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"mktunes/internal/app"
	"mktunes/internal/dominio"
)

// VisionOS é o app do Vision Pro: o "player" dele entrega o endereço do áudio
// pronto (sem cifra para desfazer) e a CDN serve esse endereço inteiro, do
// começo ao fim. Só responde a quem traz um visitante de convidado.
var VisionOS = Perfil{
	Nome:       "VISIONOS",
	ID:         "101",
	Versao:     "0.1",
	Base:       "https://www.youtube.com/youtubei/v1/",
	UserAgent:  "com.apple.visionos.youtube/1.0 (Vision Pro; visionOS 2.0; pt_BR)",
	MaisCampos: map[string]string{"deviceMake": "Apple", "deviceModel": "RealityDevice14,1", "osName": "visionOS", "osVersion": "2.0"},
}

const (
	prazoDoPlayer = 20 * time.Second
	// folga é quanto antes do vencimento um endereço guardado deixa de servir.
	folga = 2 * time.Minute
	// guardados é o teto do cache de endereços.
	guardados = 32
	// juntos é quantos pedidos de "player" andam ao mesmo tempo, no máximo.
	juntos = 3
	// tetoDaEconomia: até "128 kbps", que o YouTube mede em uns 130.
	tetoDaEconomia = 140_000
)

var idDeVideo = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)

// IDValido diz se o texto é um id de vídeo do YouTube.
func IDValido(id string) bool { return idDeVideo.MatchString(id) }

// hostDoAudio é o único de onde um endereço de áudio é aceito.
const hostDoAudio = "googlevideo.com"

// EnderecoAceito confere que o endereço do áudio é https e da CDN do YouTube.
func EnderecoAceito(bruto string) bool {
	u, erro := url.Parse(bruto)
	if erro != nil || u.Scheme != "https" || u.User != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host == hostDoAudio || strings.HasSuffix(host, "."+hostDoAudio)
}

// formato é uma codificação de áudio que o "player" ofereceu com endereço direto.
type formato struct {
	itag     int
	m4a      bool
	bps      int
	endereco string
	// loudnessDb é o volume medido que o formato informa, quando informa.
	loudnessDb *float64
}

// leFormatos tira de streamingData.adaptiveFormats os áudios que dá para abrir
// direto: com "url" (os de signatureCipher pediriam desfazer a cifra) num
// endereço aceito. A faixa original, sem compressão de volume (DRC), vem antes
// das outras: estas só servem quando não há mais nada.
func leFormatos(resposta *No) []formato {
	var originais, outros []formato
	for _, no := range resposta.Obj("streamingData").Lista("adaptiveFormats") {
		mime := no.Str("mimeType")
		endereco := no.Str("url")
		if !strings.HasPrefix(mime, "audio/") || !EnderecoAceito(endereco) {
			continue
		}
		bps := no.Int("averageBitrate")
		if bps <= 0 {
			bps = no.Int("bitrate")
		}
		f := formato{itag: no.Int("itag"), m4a: strings.HasPrefix(mime, "audio/mp4"), bps: bps, endereco: endereco}
		if db, tem := no.Num("loudnessDb"); tem {
			f.loudnessDb = &db
		}
		faixa := no.Obj("audioTrack")
		if no.Bool("isDrc") || (faixa != nil && !faixa.Bool("audioIsDefault")) {
			outros = append(outros, f)
			continue
		}
		originais = append(originais, f)
	}
	if len(originais) == 0 {
		return outros
	}
	return originais
}

// escolheAudio é o pickAudio da fonte em Kotlin: M4A primeiro (é a extensão
// com que os downloads são salvos), depois a maior taxa; na economia, até
// 128 kbps, ou a mais leve que houver.
func escolheAudio(formatos []formato, q dominio.Qualidade) (formato, bool) {
	permitidos := formatos
	if q == dominio.QualidadeEconomia {
		permitidos = nil
		var leve *formato
		for i, f := range formatos {
			if f.bps > 0 && f.bps <= tetoDaEconomia {
				permitidos = append(permitidos, f)
			}
			if leve == nil || f.bps < leve.bps {
				leve = &formatos[i]
			}
		}
		if len(permitidos) == 0 && leve != nil {
			permitidos = []formato{*leve}
		}
	}
	var melhor *formato
	for i, f := range permitidos {
		if melhor == nil || (f.m4a && !melhor.m4a) || (f.m4a == melhor.m4a && f.bps > melhor.bps) {
			melhor = &permitidos[i]
		}
	}
	if melhor == nil {
		return formato{}, false
	}
	return *melhor, true
}

type guardado struct {
	stream app.Stream
	vence  time.Time
}

type chamada struct {
	pronta chan struct{}
	stream app.Stream
	erro   error
}

// Player acha o endereço do áudio de uma faixa do YouTube pelo "player" do
// InnerTube (no lugar do NewPipe da versão em Kotlin). O id do vídeo é
// conferido antes de qualquer pedido; só um pedido por faixa de cada vez; e o
// endereço fica guardado pelo tempo que o próprio YouTube diz que ele vale.
// Pode ser chamado de várias goroutines.
type Player struct {
	// Cliente tem de usar o perfil VisionOS, com o visitante que o WebRemix guarda.
	Cliente *Cliente
	// Volume pergunta o loudness da faixa, em paralelo (pode ser nil).
	Volume func(ctx context.Context, id string) *float64
	// Visita consegue um visitante de convidado quando não há um, ou quando o
	// que há foi recusado: uma chamada qualquer do WebRemix (pode ser nil).
	Visita func(ctx context.Context) error
	Prazo  time.Duration
	Agora  func() time.Time

	mu        sync.Mutex
	qualidade dominio.Qualidade
	cache     map[string]guardado
	emVoo     map[string]*chamada
	vagas     chan struct{}
}

// DefineQualidade troca a qualidade pedida (o ajuste do ouvinte).
func (p *Player) DefineQualidade(q dominio.Qualidade) {
	p.mu.Lock()
	p.qualidade = q
	p.mu.Unlock()
}

func (p *Player) agora() time.Time {
	if p.Agora != nil {
		return p.Agora()
	}
	return time.Now()
}

// Resolve devolve o endereço do áudio. Qualquer falha é "stream indisponível",
// com o motivo que o YouTube deu.
func (p *Player) Resolve(ctx context.Context, f dominio.Faixa) (app.Stream, error) {
	if f.Provedor != dominio.YouTube {
		return app.Stream{}, fmt.Errorf("%w: não é uma faixa do YouTube: %s", app.ErrStream, f.Provedor)
	}
	if !IDValido(f.ID) {
		return app.Stream{}, fmt.Errorf("%w: id de vídeo inválido", app.ErrStream)
	}
	p.mu.Lock()
	if p.cache == nil {
		p.cache, p.emVoo, p.vagas = map[string]guardado{}, map[string]*chamada{}, make(chan struct{}, juntos)
	}
	qualidade := p.qualidade
	chave := f.ID + "|" + string(qualidade)
	if g, tem := p.cache[chave]; tem && p.agora().Before(g.vence) {
		p.mu.Unlock()
		return g.stream, nil
	}
	c, voando := p.emVoo[chave]
	if !voando {
		c = &chamada{pronta: make(chan struct{})}
		p.emVoo[chave] = c
	}
	p.mu.Unlock()
	if !voando {
		// A vaga é esperada por quem pediu: se desistir antes, o pedido nem sai.
		select {
		case p.vagas <- struct{}{}:
			go p.busca(chave, qualidade, f.ID, c)
		case <-ctx.Done():
			p.encerra(chave, c, app.Stream{}, fmt.Errorf("%w: %v", app.ErrStream, ctx.Err()), time.Time{})
		}
	}
	select {
	case <-c.pronta:
		return c.stream, c.erro
	case <-ctx.Done():
		return app.Stream{}, fmt.Errorf("%w: %v", app.ErrStream, ctx.Err())
	}
}

// busca vai até o fim mesmo que quem pediu desista: o endereço fica guardado
// para quando a faixa voltar.
func (p *Player) busca(chave string, q dominio.Qualidade, id string, c *chamada) {
	defer func() { <-p.vagas }()
	prazo := p.Prazo
	if prazo <= 0 {
		prazo = prazoDoPlayer
	}
	ctx, cancela := context.WithTimeout(context.Background(), prazo)
	defer cancela()
	// Sem visitante o YouTube responde LOGIN_REQUIRED ("confirme que você não é
	// um bot"), ao player e a quem pergunta o volume: consegue um antes.
	if visitante := p.Cliente.Visitante; visitante != nil && visitante.le() == "" {
		p.visita(ctx)
	}
	volume := make(chan *float64, 1)
	go func() {
		if p.Volume == nil {
			volume <- nil
			return
		}
		volume <- p.Volume(ctx, id)
	}()
	resposta, erro := p.pede(ctx, id)
	if erro != nil {
		p.encerra(chave, c, app.Stream{}, erro, time.Time{})
		return
	}
	escolhido, tem := escolheAudio(leFormatos(resposta), q)
	if !tem {
		p.encerra(chave, c, app.Stream{}, fmt.Errorf("%w: o YouTube respondeu sem áudio direto para %s", app.ErrStream, id), time.Time{})
		return
	}
	stream := app.Stream{URL: escolhido.endereco, LoudnessDb: <-volume, Volume: 1, Variante: strconv.Itoa(escolhido.itag)}
	if stream.LoudnessDb == nil {
		// Quem mede não respondeu a tempo: fica o que o próprio formato informa.
		stream.LoudnessDb = escolhido.loudnessDb
	}
	var validade time.Time
	if segundos := resposta.Obj("streamingData").Int("expiresInSeconds"); segundos > 0 {
		validade = p.agora().Add(time.Duration(segundos)*time.Second - folga)
	}
	p.encerra(chave, c, stream, nil, validade)
}

// pede chama o "player"; se o visitante que há for recusado (LOGIN_REQUIRED),
// tenta mais uma vez com um novo.
func (p *Player) pede(ctx context.Context, id string) (*No, error) {
	visitante := p.Cliente.Visitante
	for tentativa := 0; ; tentativa++ {
		resposta, erro := p.Cliente.Posta(ctx, "player", map[string]any{"videoId": id, "contentCheckOk": true, "racyCheckOk": true})
		if erro != nil {
			return nil, fmt.Errorf("%w: %v", app.ErrStream, erro)
		}
		situacao := resposta.Obj("playabilityStatus")
		switch status := situacao.Str("status"); {
		case status == "OK":
			return resposta, nil
		case status == "LOGIN_REQUIRED" && tentativa == 0 && visitante != nil:
			visitante.guarda("")
			p.visita(ctx)
		default:
			return nil, fmt.Errorf("%w: o YouTube recusou %s: %s: %s", app.ErrStream, id, ou(status, "sem situação"), ou(situacao.Str("reason"), "sem motivo"))
		}
	}
}

func (p *Player) visita(ctx context.Context) {
	if p.Visita != nil {
		_ = p.Visita(ctx)
	}
}

func (p *Player) encerra(chave string, c *chamada, stream app.Stream, erro error, validade time.Time) {
	p.mu.Lock()
	delete(p.emVoo, chave)
	agora := p.agora()
	if erro == nil && validade.After(agora) {
		for outra, g := range p.cache {
			if !agora.Before(g.vence) {
				delete(p.cache, outra)
			}
		}
		for len(p.cache) >= guardados {
			primeira := ""
			for outra, g := range p.cache {
				if primeira == "" || g.vence.Before(p.cache[primeira].vence) {
					primeira = outra
				}
			}
			delete(p.cache, primeira)
		}
		p.cache[chave] = guardado{stream, validade}
	}
	p.mu.Unlock()
	c.stream, c.erro = stream, erro
	close(c.pronta)
}
