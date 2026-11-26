// SPDX-License-Identifier: GPL-3.0-or-later

// Pacote ytdlp: acha o endereço do áudio de uma faixa do YouTube chamando o
// yt-dlp do sistema (no lugar do NewPipe da versão em Kotlin). Nunca passa por
// shell; o id do vídeo é conferido antes de ir para a linha de comando; só um
// yt-dlp por faixa de cada vez; e o endereço fica guardado pelo tempo que o
// próprio YouTube diz que ele vale.
package ytdlp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"mktunes/internal/app"
	"mktunes/internal/dominio"
)

// Seletores de formato (-f): só HTTP direto (sem manifestos), M4A primeiro
// (é a extensão com que os downloads são salvos), depois a maior taxa; na
// economia, até 128 kbps, ou a menor que houver.
const (
	SeletorAlta     = "bestaudio[ext=m4a][protocol^=http]/bestaudio[protocol^=http]/bestaudio"
	SeletorEconomia = "bestaudio[ext=m4a][abr<=128][protocol^=http]/bestaudio[abr<=128][protocol^=http]/worstaudio[protocol^=http]/worstaudio"
)

// AvisoSemYtdlp é o que a janela mostra quando o comando não está instalado.
const AvisoSemYtdlp = "O MkTunes precisa do yt-dlp para tocar do YouTube. Instale o pacote yt-dlp (sudo pacman -S yt-dlp)."

const (
	prazoPadrao = 45 * time.Second
	// folga é quanto antes do vencimento um endereço guardado deixa de servir.
	folga = 2 * time.Minute
	// guardados é o teto do cache de endereços.
	guardados = 32
	// juntos é quantos yt-dlp rodam ao mesmo tempo, no máximo.
	juntos   = 3
	saidaMax = 64 * 1024
)

var (
	idDeVideo = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)
	vence     = regexp.MustCompile(`[?&/]expire[=/](\d{9,11})`)
	itag      = regexp.MustCompile(`[?&/]itag[=/](\d+)`)
)

// IDValido diz se o texto é um id de vídeo do YouTube.
func IDValido(id string) bool { return idDeVideo.MatchString(id) }

// hostsDoYouTube são os únicos de onde um endereço de áudio é aceito.
var hostsDoYouTube = []string{"googlevideo.com", "youtube.com"}

// EnderecoAceito confere que a saída do yt-dlp é um endereço https do YouTube.
func EnderecoAceito(bruto string) bool {
	u, erro := url.Parse(bruto)
	if erro != nil || u.Scheme != "https" || u.User != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	for _, h := range hostsDoYouTube {
		if host == h || strings.HasSuffix(host, "."+h) {
			return true
		}
	}
	return false
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

// Resolvedor resolve faixas do YouTube. Pode ser chamado de várias goroutines.
type Resolvedor struct {
	// Comando é o yt-dlp (MKTUNES_YTDLP).
	Comando string
	// Volume pergunta o loudness da faixa ao InnerTube, em paralelo (pode ser nil).
	Volume func(ctx context.Context, id string) *float64
	// Aviso recebe o texto para a janela quando falta o yt-dlp.
	Aviso func(texto string)
	Prazo time.Duration
	Agora func() time.Time

	mu        sync.Mutex
	qualidade dominio.Qualidade
	cache     map[string]guardado
	emVoo     map[string]*chamada
	vagas     chan struct{}
	avisouEm  time.Time
}

// DefineQualidade troca a qualidade pedida (o ajuste do ouvinte).
func (r *Resolvedor) DefineQualidade(q dominio.Qualidade) {
	r.mu.Lock()
	r.qualidade = q
	r.mu.Unlock()
}

func (r *Resolvedor) agora() time.Time {
	if r.Agora != nil {
		return r.Agora()
	}
	return time.Now()
}

// Resolve devolve o endereço do áudio. Qualquer falha é "stream indisponível",
// com a última linha do que o yt-dlp reclamou.
func (r *Resolvedor) Resolve(ctx context.Context, f dominio.Faixa) (app.Stream, error) {
	if f.Provedor != dominio.YouTube {
		return app.Stream{}, fmt.Errorf("%w: não é uma faixa do YouTube: %s", app.ErrStream, f.Provedor)
	}
	if !IDValido(f.ID) {
		return app.Stream{}, fmt.Errorf("%w: id de vídeo inválido", app.ErrStream)
	}
	r.mu.Lock()
	if r.cache == nil {
		r.cache, r.emVoo, r.vagas = map[string]guardado{}, map[string]*chamada{}, make(chan struct{}, juntos)
	}
	seletor, chave := SeletorAlta, f.ID+"|alta"
	if r.qualidade == dominio.QualidadeEconomia {
		seletor, chave = SeletorEconomia, f.ID+"|economia"
	}
	if g, tem := r.cache[chave]; tem && r.agora().Before(g.vence) {
		r.mu.Unlock()
		return g.stream, nil
	}
	c, voando := r.emVoo[chave]
	if !voando {
		c = &chamada{pronta: make(chan struct{})}
		r.emVoo[chave] = c
	}
	r.mu.Unlock()
	if !voando {
		// A vaga é esperada por quem pediu: se desistir antes, o yt-dlp nem sobe.
		select {
		case r.vagas <- struct{}{}:
			go r.busca(chave, seletor, f.ID, c)
		case <-ctx.Done():
			r.encerra(chave, c, app.Stream{}, fmt.Errorf("%w: %v", app.ErrStream, ctx.Err()), time.Time{})
		}
	}
	select {
	case <-c.pronta:
		return c.stream, c.erro
	case <-ctx.Done():
		return app.Stream{}, fmt.Errorf("%w: %v", app.ErrStream, ctx.Err())
	}
}

// busca roda o yt-dlp até o fim mesmo que quem pediu desista: o endereço fica
// guardado para quando a faixa voltar.
func (r *Resolvedor) busca(chave, seletor, id string, c *chamada) {
	defer func() { <-r.vagas }()
	prazo := r.Prazo
	if prazo <= 0 {
		prazo = prazoPadrao
	}
	ctx, cancela := context.WithTimeout(context.Background(), prazo)
	defer cancela()
	volume := make(chan *float64, 1)
	go func() {
		if r.Volume == nil {
			volume <- nil
			return
		}
		volume <- r.Volume(ctx, id)
	}()
	endereco, erro := r.roda(ctx, seletor, id)
	if erro != nil {
		r.encerra(chave, c, app.Stream{}, erro, time.Time{})
		return
	}
	stream := app.Stream{URL: endereco, LoudnessDb: <-volume, Volume: 1, Variante: "0"}
	if m := itag.FindStringSubmatch(endereco); m != nil {
		stream.Variante = m[1]
	}
	var validade time.Time
	if m := vence.FindStringSubmatch(endereco); m != nil {
		segundos, _ := strconv.ParseInt(m[1], 10, 64)
		validade = time.Unix(segundos, 0).Add(-folga)
	}
	r.encerra(chave, c, stream, nil, validade)
}

func (r *Resolvedor) encerra(chave string, c *chamada, stream app.Stream, erro error, validade time.Time) {
	r.mu.Lock()
	delete(r.emVoo, chave)
	agora := r.agora()
	if erro == nil && validade.After(agora) {
		for outra, g := range r.cache {
			if !agora.Before(g.vence) {
				delete(r.cache, outra)
			}
		}
		for len(r.cache) >= guardados {
			primeira := ""
			for outra, g := range r.cache {
				if primeira == "" || g.vence.Before(r.cache[primeira].vence) {
					primeira = outra
				}
			}
			delete(r.cache, primeira)
		}
		r.cache[chave] = guardado{stream, validade}
	}
	r.mu.Unlock()
	c.stream, c.erro = stream, erro
	close(c.pronta)
}

// Argumentos é a linha de comando do yt-dlp (sem o nome do programa).
func Argumentos(seletor, id string) []string {
	return []string{"-f", seletor, "-g", "--no-playlist", "--no-warnings", "--", "https://music.youtube.com/watch?v=" + id}
}

func (r *Resolvedor) roda(ctx context.Context, seletor, id string) (string, error) {
	comando := exec.CommandContext(ctx, r.Comando, Argumentos(seletor, id)...)
	saida, erros := &limitado{max: saidaMax}, &limitado{max: saidaMax}
	comando.Stdout, comando.Stderr = saida, erros
	comando.WaitDelay = 2 * time.Second
	if erro := comando.Run(); erro != nil {
		if errors.Is(erro, exec.ErrNotFound) || errors.Is(erro, fs.ErrNotExist) {
			r.avisa()
			return "", fmt.Errorf("%w: falta o yt-dlp", app.ErrStream)
		}
		if ctx.Err() != nil {
			return "", fmt.Errorf("%w: o yt-dlp demorou demais", app.ErrStream)
		}
		return "", fmt.Errorf("%w: %s", app.ErrStream, ultimaLinha(erros.String(), erro.Error()))
	}
	for _, linha := range strings.Split(saida.String(), "\n") {
		if linha = strings.TrimSpace(linha); linha != "" {
			if !EnderecoAceito(linha) {
				return "", fmt.Errorf("%w: o yt-dlp devolveu um endereço que não é do YouTube", app.ErrStream)
			}
			return linha, nil
		}
	}
	return "", fmt.Errorf("%w: o yt-dlp não devolveu endereço nenhum", app.ErrStream)
}

// avisa a janela de que falta o yt-dlp, no máximo uma vez por minuto (o
// autoplay tentaria a cada faixa).
func (r *Resolvedor) avisa() {
	r.mu.Lock()
	agora := r.agora()
	calado := !r.avisouEm.IsZero() && agora.Sub(r.avisouEm) < time.Minute
	if !calado {
		r.avisouEm = agora
	}
	r.mu.Unlock()
	if !calado && r.Aviso != nil {
		r.Aviso(AvisoSemYtdlp)
	}
}

func ultimaLinha(texto, reserva string) string {
	linhas := strings.Split(strings.TrimSpace(texto), "\n")
	if ultima := strings.TrimSpace(linhas[len(linhas)-1]); ultima != "" {
		return ultima
	}
	return reserva
}

// limitado guarda só o começo do que escrevem nele.
type limitado struct {
	bytes.Buffer
	max int
}

func (l *limitado) Write(p []byte) (int, error) {
	if sobra := l.max - l.Len(); sobra > 0 {
		l.Buffer.Write(p[:min(len(p), sobra)])
	}
	return len(p), nil
}
