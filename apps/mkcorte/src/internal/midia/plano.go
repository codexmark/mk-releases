package midia

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"strconv"
	"strings"

	"mkcorte/internal/base"
)

// Edicao é o que a janela manda para o `plano` e o `exportar`: os vídeos de
// origem e, na ordem em que vão sair, os clipes (um trecho de uma fonte).
type Edicao struct {
	Fontes    []string `json:"fontes"`
	Clipes    []Clipe  `json:"clipes"`
	Qualidade string   `json:"qualidade"` // "original" (ou ""), "1080", "720"
	Modo      string   `json:"modo"`      // "exato" (ou ""), "rapido"
}

// Clipe é um trecho [inicio, fim), em segundos, da fonte de índice Fonte.
type Clipe struct {
	Fonte  int     `json:"fonte"`
	Inicio float64 `json:"inicio"`
	Fim    float64 `json:"fim"`
}

// Os limites de uma edição: o ffmpeg abre uma entrada por clipe.
const (
	maximoDeFontes = 64
	maximoDeClipes = 400
	menorClipe     = 0.02 // segundos
)

// LeEdicao lê o JSON de uma edição e confere o que não depende dos arquivos.
func LeEdicao(texto []byte) (*Edicao, error) {
	var e Edicao
	if err := json.Unmarshal(texto, &e); err != nil {
		return nil, errors.New("a edição não é um JSON que eu entenda")
	}
	if len(e.Fontes) == 0 || len(e.Clipes) == 0 {
		return nil, errors.New("a edição está vazia: não há o que exportar")
	}
	if len(e.Fontes) > maximoDeFontes {
		return nil, fmt.Errorf("a edição tem vídeos demais (%d; o máximo é %d)", len(e.Fontes), maximoDeFontes)
	}
	if len(e.Clipes) > maximoDeClipes {
		return nil, fmt.Errorf("a edição tem clipes demais (%d; o máximo é %d)", len(e.Clipes), maximoDeClipes)
	}
	for i, f := range e.Fontes {
		if !filepath.IsAbs(f) {
			return nil, fmt.Errorf("o vídeo %d da edição não tem caminho absoluto: %q", i+1, f)
		}
	}
	for i, c := range e.Clipes {
		if c.Fonte < 0 || c.Fonte >= len(e.Fontes) {
			return nil, fmt.Errorf("o clipe %d aponta para um vídeo que não está na edição", i+1)
		}
		if math.IsNaN(c.Inicio) || math.IsNaN(c.Fim) || math.IsInf(c.Inicio, 0) || math.IsInf(c.Fim, 0) || c.Inicio < 0 || c.Fim-c.Inicio < menorClipe {
			return nil, fmt.Errorf("o clipe %d tem um trecho que não existe (%g a %g)", i+1, c.Inicio, c.Fim)
		}
	}
	switch e.Qualidade {
	case "":
		e.Qualidade = "original"
	case "original", "1080", "720":
	default:
		return nil, fmt.Errorf("não conheço a qualidade %q (original, 1080 ou 720)", e.Qualidade)
	}
	switch e.Modo {
	case "":
		e.Modo = "exato"
	case "exato", "rapido":
	default:
		return nil, fmt.Errorf("não conheço o modo %q (exato ou rapido)", e.Modo)
	}
	return &e, nil
}

// Rapido diz se a edição pode sair sem recodificar, e por que não.
type Rapido struct {
	Pode   bool   `json:"pode"`
	Motivo string `json:"motivo"`
	Aviso  string `json:"aviso"`
}

// AvisoDoRapido é a franqueza do modo sem recodificar.
const AvisoDoRapido = "Sem recodificar, o corte cai no quadro-chave anterior: cada clipe pode começar um pouco antes do ponto marcado " +
	"(até alguns segundos, conforme o vídeo) e a duração final pode passar da prevista. Para o corte exato ao quadro, use o modo exato."

// Plano é a resposta do `plano`: como a edição seria exportada.
type Plano struct {
	Modo           string   `json:"modo"`
	Duracao        float64  `json:"duracao_s"`
	Clipes         int      `json:"clipes"`
	Largura        int      `json:"largura"`
	Altura         int      `json:"altura"`
	FPS            *float64 `json:"fps"`
	TemAudio       bool     `json:"tem_audio"`
	Tamanho        int64    `json:"tamanho_estimado_bytes"`
	SemRecodificar Rapido   `json:"sem_recodificar"`
	Saida          string   `json:"saida"`
	Comando        []string `json:"comando"`
	Avisos         []string `json:"avisos"`

	// Os argumentos do ffmpeg, com "{fonteN}", "{lista}" e "{saida}" no lugar
	// dos caminhos (quem roda troca), e o texto da lista do modo rápido.
	Args  []string `json:"-"`
	Lista string   `json:"-"`
}

func par(v float64) int { return int(math.Floor(v/2+0.5)) * 2 }

// Dimensoes limita o lado menor do quadro a `lado`, sem ampliar e com
// dimensões pares (o H.264 pede). lado 0 só deixa par.
func Dimensoes(largura, altura, lado int) (int, int) {
	menor := min(largura, altura)
	if lado <= 0 || menor <= lado {
		return max(2, largura/2*2), max(2, altura/2*2)
	}
	lado = lado / 2 * 2
	maior := max(2, par(float64(max(largura, altura))*float64(lado)/float64(menor)))
	if largura >= altura {
		return maior, lado
	}
	return lado, maior
}

// encaixe diz como um quadro de l×a entra num de L×A sem deformar: o tamanho
// depois da escala (par, nunca maior que o quadro) e a margem das faixas.
func encaixe(l, a, L, A int) (sl, sa, x, y int) {
	razao := math.Min(float64(L)/float64(l), float64(A)/float64(a))
	sl = min(L, max(2, par(float64(l)*razao)))
	sa = min(A, max(2, par(float64(a)*razao)))
	return sl, sa, (L - sl) / 2, (A - sa) / 2
}

// seg escreve um tempo para o ffmpeg: segundos com três casas.
func seg(v float64) string { return strconv.FormatFloat(v, 'f', 3, 64) }

// O começo de toda chamada: sem conversa, sem ler o teclado, progresso em
// linhas no stdout.
var comum = []string{"-hide_banner", "-nostdin", "-loglevel", "error", "-y", "-progress", "pipe:1", "-nostats"}

// Os codecs que entram num MP4 sem recodificar.
var (
	videoDeMP4 = map[string]bool{"h264": true, "hevc": true, "av1": true, "mpeg4": true, "vp9": true}
	audioDeMP4 = map[string]bool{"aac": true, "mp3": true, "ac3": true, "eac3": true, "opus": true, "alac": true, "flac": true}
)

// mesmosParametros diz se dois vídeos podem ser emendados sem recodificar.
func mesmosParametros(a, b *Fonte) bool {
	if a == b {
		return true
	}
	if a.Codec != b.Codec || a.Perfil != b.Perfil || a.nivel != b.nivel || a.gravado != b.gravado || a.Largura != b.Largura || a.Altura != b.Altura ||
		a.Pixel != b.Pixel || a.SAR != b.SAR || a.Fracao != b.Fracao || a.Rotacao != b.Rotacao || a.Trilha != b.Trilha || a.TemAudio != b.TemAudio {
		return false
	}
	if a.Audio == nil || b.Audio == nil {
		return a.Audio == nil && b.Audio == nil
	}
	return a.Audio.Codec == b.Audio.Codec && a.Audio.Canais == b.Audio.Canais && a.Audio.Amostragem == b.Audio.Amostragem && a.Audio.Trilha == b.Audio.Trilha
}

// semRecodificar decide se a edição pode sair por cópia de fluxo.
func semRecodificar(e *Edicao, fontes []*Fonte, largura, altura int) Rapido {
	primeira := fontes[e.Clipes[0].Fonte]
	nao := func(formato string, args ...any) Rapido { return Rapido{Motivo: fmt.Sprintf(formato, args...)} }
	if largura != primeira.Largura || altura != primeira.Altura {
		if e.Qualidade != "original" {
			return nao("reduzir para %sp pede recodificar", e.Qualidade)
		}
		return nao("o vídeo tem dimensão ímpar (%d×%d), que o MP4 em H.264 não aceita: precisa recodificar", primeira.Largura, primeira.Altura)
	}
	for _, c := range e.Clipes {
		if !mesmosParametros(primeira, fontes[c.Fonte]) {
			return nao("os clipes vêm de vídeos com parâmetros diferentes (resolução, codec, taxa de quadros ou áudio): juntá-los pede recodificar")
		}
	}
	if !videoDeMP4[primeira.Codec] {
		return nao("o vídeo está em %s, que não entra num MP4 sem recodificar", primeira.Codec)
	}
	if primeira.Audio != nil && !audioDeMP4[primeira.Audio.Codec] {
		return nao("o áudio está em %s, que não entra num MP4 sem recodificar", primeira.Audio.Codec)
	}
	return Rapido{Pode: true, Aviso: AvisoDoRapido}
}

// Planeja monta o plano de uma edição: função pura (não lê disco nem roda
// nada). `fontes` são as sondagens, na ordem de e.Fontes; `saida` é o nome
// que a saída teria. Corrige o fim de um clipe que passa do fim do vídeo.
func Planeja(e *Edicao, fontes []*Fonte, saida string) (*Plano, error) {
	if len(fontes) != len(e.Fontes) {
		return nil, errors.New("faltou sondar algum vídeo da edição")
	}
	clipes := make([]Clipe, len(e.Clipes))
	for i, c := range e.Clipes {
		f := fontes[c.Fonte]
		// Uma folga pequena: a janela arredonda os tempos.
		if c.Inicio >= f.Duracao || c.Fim > f.Duracao+0.5 {
			return nil, fmt.Errorf("o clipe %d (%s a %s s) passa do fim de %s, que tem %s s", i+1, seg(c.Inicio), seg(c.Fim), filepath.Base(f.Arquivo), seg(f.Duracao))
		}
		c.Fim = math.Min(c.Fim, f.Duracao)
		if c.Fim-c.Inicio < menorClipe {
			return nil, fmt.Errorf("o clipe %d ficou curto demais para exportar", i+1)
		}
		clipes[i] = c
	}
	e = &Edicao{Fontes: e.Fontes, Clipes: clipes, Qualidade: e.Qualidade, Modo: e.Modo}

	primeira := fontes[clipes[0].Fonte]
	lado := 0
	if e.Qualidade == "1080" || e.Qualidade == "720" {
		lado, _ = strconv.Atoi(e.Qualidade)
	}
	p := &Plano{Modo: "exato", Clipes: len(clipes), FPS: primeira.FPS, Saida: saida, Avisos: []string{}}
	p.Largura, p.Altura = Dimensoes(primeira.Largura, primeira.Altura, lado)
	for _, c := range clipes {
		p.Duracao += c.Fim - c.Inicio
		p.TemAudio = p.TemAudio || fontes[c.Fonte].Audio != nil
	}
	p.Duracao = math.Round(p.Duracao*1000) / 1000
	p.SemRecodificar = semRecodificar(e, fontes, p.Largura, p.Altura)

	if e.Modo == "rapido" {
		if !p.SemRecodificar.Pode {
			return nil, base.Recusa("não dá para exportar sem recodificar: %s", p.SemRecodificar.Motivo)
		}
		p.Modo = "rapido"
		p.TemAudio = primeira.Audio != nil
		copia(p, e, fontes)
	} else {
		recodifica(p, e, fontes)
	}

	// A linha de comando como a pessoa a leria: os atalhos pelo nome, a
	// lista pelo nome e a saída por extenso.
	troca := []string{"{saida}", saida, "{lista}", nomeDaLista}
	for k, arquivo := range e.Fontes {
		troca = append(troca, "{fonte"+strconv.Itoa(k)+"}", nomeDoAtalho(k, arquivo))
	}
	r := strings.NewReplacer(troca...)
	p.Comando = []string{"ffmpeg"}
	for _, a := range p.Args {
		p.Comando = append(p.Comando, r.Replace(a))
	}
	return p, nil
}

// O nome da lista do modo rápido, na pasta temporária dos atalhos.
const nomeDaLista = "lista.ffconcat"

// copia monta o modo rápido: o demuxer concat lê uma lista com o trecho de
// cada clipe e os pacotes vão para a saída como estão.
func copia(p *Plano, e *Edicao, fontes []*Fonte) {
	primeira := fontes[e.Clipes[0].Fonte]
	var lista strings.Builder
	lista.WriteString("ffconcat version 1.0\n")
	var tamanho float64
	varias := false
	for _, c := range e.Clipes {
		f := fontes[c.Fonte]
		varias = varias || c.Fonte != e.Clipes[0].Fonte
		// O nome do atalho só tem letras, algarismos e ponto: não precisa de
		// escape dentro das aspas.
		lista.WriteString("file '" + nomeDoAtalho(c.Fonte, e.Fontes[c.Fonte]) + "'\n")
		if c.Inicio > 0 {
			lista.WriteString("inpoint " + seg(c.Inicio) + "\n")
		}
		lista.WriteString("outpoint " + seg(c.Fim) + "\n")
		if f.Duracao > 0 {
			tamanho += float64(f.Tamanho) * (c.Fim - c.Inicio) / f.Duracao
		}
	}
	p.Lista = lista.String()
	p.Tamanho = int64(tamanho)
	p.Avisos = append(p.Avisos, AvisoDoRapido)
	if varias {
		p.Avisos = append(p.Avisos, "Os clipes vêm de arquivos diferentes com os mesmos parâmetros. Se a emenda sair com defeito, exporte no modo exato.")
	}

	args := append([]string{}, comum...)
	args = append(args, "-f", "concat", "-safe", "0", "-i", "{lista}", "-map", "0:"+strconv.Itoa(primeira.Trilha))
	if primeira.Audio != nil {
		args = append(args, "-map", "0:"+strconv.Itoa(primeira.Audio.Trilha))
	}
	args = append(args, "-codec", "copy")
	if primeira.Codec == "hevc" {
		args = append(args, "-tag:v", "hvc1") // sem isto os aparelhos da Apple não abrem
	}
	p.Args = append(args, "-avoid_negative_ts", "make_zero", "-movflags", "+faststart", "-f", "mp4", "{saida}")
}

// O que o modo exato grava.
const (
	audioBps  = 160_000
	bitsPorPx = 0.1 // H.264 em CRF 20, por pixel e por quadro: só para a estimativa
)

// recodifica monta o modo exato: uma entrada por clipe (já posicionada no
// início dele), cada uma aparada e posta no quadro e no formato da saída, e
// tudo emendado pelo filtro concat.
func recodifica(p *Plano, e *Edicao, fontes []*Fonte) {
	primeira := fontes[e.Clipes[0].Fonte]
	taxa := primeira.Fracao
	if taxa == "" {
		taxa = "30"
	}
	fps := 30.0
	if primeira.FPS != nil {
		fps = *primeira.FPS
	}
	// Com um vídeo só, a taxa de quadros já é uma; com mais de um, todos
	// passam para a do primeiro.
	variasFontes := false
	for _, c := range e.Clipes {
		variasFontes = variasFontes || c.Fonte != e.Clipes[0].Fonte
	}

	args := append([]string{}, comum...)
	var grafo []string
	var juncao strings.Builder
	var bits float64
	faixas, silencio, outraTaxa := false, false, false
	n := len(e.Clipes)
	for k, c := range e.Clipes {
		f := fontes[c.Fonte]
		d := seg(c.Fim - c.Inicio)
		outraTaxa = outraTaxa || f.Fracao != primeira.Fracao
		args = append(args, "-ss", seg(c.Inicio), "-t", d, "-i", "{fonte"+strconv.Itoa(c.Fonte)+"}")

		v, a := "[v"+strconv.Itoa(k)+"]", "[a"+strconv.Itoa(k)+"]"
		if n == 1 {
			v, a = "[v]", "[a]"
		}
		video := []string{"trim=duration=" + d, "setpts=PTS-STARTPTS"}
		if f.Largura != p.Largura || f.Altura != p.Altura || (f.SAR != "" && f.SAR != "1:1") {
			sl, sa, x, y := encaixe(f.Largura, f.Altura, p.Largura, p.Altura)
			video = append(video, fmt.Sprintf("scale=%d:%d", sl, sa))
			if sl != p.Largura || sa != p.Altura {
				video = append(video, fmt.Sprintf("pad=%d:%d:%d:%d:color=black", p.Largura, p.Altura, x, y))
				faixas = true
			}
		}
		video = append(video, "setsar=1")
		if variasFontes {
			video = append(video, "fps="+taxa)
		}
		video = append(video, "format=yuv420p")
		grafo = append(grafo, fmt.Sprintf("[%d:%d]%s%s", k, f.Trilha, strings.Join(video, ","), v))
		juncao.WriteString(v)

		if p.TemAudio {
			if f.Audio != nil {
				grafo = append(grafo, fmt.Sprintf("[%d:%d]atrim=duration=%s,asetpts=PTS-STARTPTS,aresample=48000,aformat=sample_fmts=fltp:channel_layouts=stereo%s",
					k, f.Audio.Trilha, d, a))
			} else {
				grafo = append(grafo, "anullsrc=channel_layout=stereo:sample_rate=48000,atrim=duration="+d+a)
				silencio = true
			}
			juncao.WriteString(a)
		}

		// A estimativa: o que o CRF costuma gastar neste quadro, sem passar
		// do que o próprio vídeo gasta.
		bps := float64(p.Largura) * float64(p.Altura) * fps * bitsPorPx
		if f.Taxa != nil && float64(*f.Taxa) < bps {
			bps = float64(*f.Taxa)
		}
		bits += bps * (c.Fim - c.Inicio)
	}
	if n > 1 {
		temAudio := 0
		saidas := "[v]"
		if p.TemAudio {
			temAudio, saidas = 1, "[v][a]"
		}
		grafo = append(grafo, fmt.Sprintf("%sconcat=n=%d:v=1:a=%d%s", juncao.String(), n, temAudio, saidas))
	}
	args = append(args, "-filter_complex", strings.Join(grafo, ";"), "-map", "[v]")
	if p.TemAudio {
		args = append(args, "-map", "[a]")
		bits += audioBps * p.Duracao
	}
	args = append(args, "-c:v", "libx264", "-preset", "medium", "-crf", "20", "-pix_fmt", "yuv420p")
	if p.TemAudio {
		args = append(args, "-c:a", "aac", "-b:a", "160k", "-ar", "48000", "-ac", "2")
	}
	p.Args = append(args, "-movflags", "+faststart", "-f", "mp4", "{saida}")
	p.Tamanho = int64(bits / 8)
	if faixas {
		p.Avisos = append(p.Avisos, "Há clipes com outra proporção: entram com faixas pretas, no quadro do primeiro clipe.")
	}
	if silencio {
		p.Avisos = append(p.Avisos, "Há clipes sem áudio: ficam em silêncio.")
	}
	if outraTaxa {
		p.Avisos = append(p.Avisos, "Todos os clipes saem na taxa de quadros do primeiro.")
	}
}
