package midia

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"mkutil/internal/base"
)

// Pedido é o que veio na linha de comando do `comprimir`.
type Pedido struct {
	Predefinicao string
	AlvoMB       float64
	Lado         int
	Codec        string // h264, h265, av1
	Acelerar     string // "", vaapi, nvenc
	SemAudio     bool
	Formato      string // jpeg, webp, avif
	Qualidade    int
	Largura      int
	FPS          int
}

// Plano é o que o ffmpeg vai fazer: uma lista de argumentos por passada.
// Neles, "{entrada}", "{saida}" e "{passada}" são trocados na hora de rodar.
type Plano struct {
	Sufixo   string
	Ext      string
	Passadas [][]string
	VideoBps int64
	AudioBps int64
	Largura  int
	Altura   int
}

// Codificadores lista os codificadores que o ffmpeg desta máquina tem.
func Codificadores() (map[string]bool, error) {
	if !base.Tem("ffmpeg") {
		return nil, errors.New("não achei o ffmpeg nesta máquina: instale o pacote ffmpeg")
	}
	saida, err := base.Roda("ffmpeg", 20*time.Second, nil, "-hide_banner", "-encoders")
	if err != nil {
		return nil, fmt.Errorf("o ffmpeg não respondeu: %v", err)
	}
	tem := map[string]bool{}
	depois := false
	for _, linha := range strings.Split(string(saida), "\n") {
		campos := strings.Fields(linha)
		if strings.HasPrefix(strings.TrimSpace(linha), "------") {
			depois = true
			continue
		}
		if depois && len(campos) >= 2 && len(campos[0]) == 6 {
			tem[campos[1]] = true
		}
	}
	return tem, nil
}

// Capacidades é a resposta do `codificadores`.
type Capacidades struct {
	Ffmpeg     bool     `json:"ffmpeg"`
	Codecs     []string `json:"codecs"`
	Aceleracao []string `json:"aceleracao"`
	Imagens    []string `json:"imagens"`
	Animados   []string `json:"animados"`
}

// Resume diz, do que o ffmpeg tem, o que o MkComprimir oferece.
func Resume(tem map[string]bool) Capacidades {
	c := Capacidades{Ffmpeg: tem != nil, Codecs: []string{}, Aceleracao: []string{}, Imagens: []string{}, Animados: []string{}}
	poe := func(lista *[]string, nome string, algum ...string) {
		for _, a := range algum {
			if tem[a] {
				*lista = append(*lista, nome)
				return
			}
		}
	}
	poe(&c.Codecs, "h264", "libx264")
	poe(&c.Codecs, "h265", "libx265")
	poe(&c.Codecs, "av1", "libsvtav1", "libaom-av1")
	poe(&c.Aceleracao, "vaapi", "h264_vaapi", "hevc_vaapi", "av1_vaapi")
	poe(&c.Aceleracao, "nvenc", "h264_nvenc", "hevc_nvenc", "av1_nvenc")
	poe(&c.Imagens, "jpeg", "mjpeg")
	poe(&c.Imagens, "webp", "libwebp")
	poe(&c.Imagens, "avif", "libaom-av1", "libsvtav1")
	poe(&c.Animados, "gif", "gif")
	poe(&c.Animados, "webp", "libwebp_anim", "libwebp")
	return c
}

func par(v float64) int { return int(math.Floor(v/2+0.5)) * 2 }

// Dimensoes limita o lado menor do quadro a `lado`, sem ampliar e com
// dimensões pares (os codificadores pedem). lado 0 só deixa par.
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

// porLargura limita a largura, sem ampliar, com dimensões pares.
func porLargura(largura, altura, alvo int) (int, int) {
	if alvo <= 0 || largura <= alvo {
		return max(2, largura/2*2), max(2, altura/2*2)
	}
	alvo = max(2, alvo/2*2)
	return alvo, max(2, par(float64(altura)*float64(alvo)/float64(largura)))
}

// Os lados que o `caber` tenta, do maior para o menor.
var ladosDoCaber = []int{1080, 720, 480, 360, 240}

// Cabe faz a conta do `caber` (está no contrato): a taxa de vídeo e de áudio
// para o arquivo sair com até alvoMB, e as dimensões em que essa taxa ainda
// dá imagem que preste.
func Cabe(alvoMB, duracao float64, temAudio bool, largura, altura int, fps *float64) (video, audio int64, l, a int, err error) {
	if !(alvoMB > 0 && duracao > 0) {
		return 0, 0, 0, 0, errors.New("para caber num tamanho é preciso o tamanho (em MB) e a duração do vídeo")
	}
	total := int64(math.Floor(alvoMB * 8_000_000 * 0.97 / duracao))
	switch {
	case !temAudio:
	case total >= 500_000:
		audio = 96_000
	case total >= 200_000:
		audio = 64_000
	default:
		audio = 32_000
	}
	video = total - audio
	if video < 50_000 {
		return 0, 0, 0, 0, fmt.Errorf("não cabe: o vídeo é longo demais para %s MB (sobraria menos de 50 kb/s para a imagem)", numeroCurto(alvoMB))
	}
	f := 30.0
	if fps != nil && *fps > 0 {
		f = *fps
	}
	menor := min(largura, altura)
	lados := []int{menor}
	for _, x := range ladosDoCaber {
		if x < menor {
			lados = append(lados, x)
		}
	}
	for _, lado := range lados {
		l, a = Dimensoes(largura, altura, lado)
		if float64(video)/(float64(l)*float64(a)*f) >= 0.045 {
			return video, audio, l, a, nil
		}
	}
	l, a = Dimensoes(largura, altura, min(240, menor))
	return video, audio, l, a, nil
}

func numeroCurto(v float64) string {
	return strings.TrimSuffix(strings.TrimSuffix(strconv.FormatFloat(v, 'f', 1, 64), "0"), ".")
}

// O que cada predefinição de vídeo por qualidade pede.
var porQualidade = map[string]struct {
	crf, ladoMaximo, audioK int
	sufixo                  string
}{
	"menor":      {30, 720, 64, "-menor"},
	"equilibrio": {24, 1080, 128, "-equilibrio"},
	"alta":       {19, 0, 160, "-alta"},
	"resolucao":  {22, 0, 128, ""},
}

// O começo de toda chamada: sem conversa, sem ler o teclado, progresso em
// linhas no stdout.
var comum = []string{"-hide_banner", "-nostdin", "-loglevel", "error", "-y", "-progress", "pipe:1", "-nostats"}

// codificadorDe escolhe o codificador do ffmpeg para o codec e a aceleração
// pedidos. Aceleração só se a pessoa pediu, e só se o ffmpeg lista.
func codificadorDe(codec, acelerar string, tem map[string]bool) (string, error) {
	if codec == "" {
		codec = "h264"
	}
	nomes := map[string]map[string][]string{
		"h264": {"": {"libx264"}, "vaapi": {"h264_vaapi"}, "nvenc": {"h264_nvenc"}},
		"h265": {"": {"libx265"}, "vaapi": {"hevc_vaapi"}, "nvenc": {"hevc_nvenc"}},
		"av1":  {"": {"libsvtav1", "libaom-av1"}, "vaapi": {"av1_vaapi"}, "nvenc": {"av1_nvenc"}},
	}
	porCodec, ok := nomes[codec]
	if !ok {
		return "", fmt.Errorf("não conheço o codec %q: use h264, h265 ou av1", codec)
	}
	candidatos, ok := porCodec[acelerar]
	if !ok {
		return "", fmt.Errorf("não conheço a aceleração %q: use vaapi ou nvenc", acelerar)
	}
	for _, c := range candidatos {
		if tem[c] {
			return c, nil
		}
	}
	if acelerar != "" {
		return "", fmt.Errorf("o ffmpeg desta máquina não tem o codificador %s: tire a aceleração", candidatos[0])
	}
	return "", fmt.Errorf("o ffmpeg desta máquina não tem o codificador %s", candidatos[0])
}

// video monta os argumentos do codificador de vídeo. Com taxa > 0 é por
// taxa (o `caber`); senão por qualidade (crf, na escala do H.264).
func argsDeVideo(cod string, crf int, taxa int64, passada int) []string {
	k := strconv.FormatInt(taxa/1000, 10) + "k"
	teto := strconv.FormatInt(taxa*3/2000, 10) + "k"
	switch {
	case cod == "libx264":
		a := []string{"-c:v", cod, "-preset", "medium", "-pix_fmt", "yuv420p"}
		if taxa > 0 {
			return append(a, "-b:v", k, "-pass", strconv.Itoa(passada), "-passlogfile", "{passada}")
		}
		return append(a, "-crf", strconv.Itoa(crf))
	case cod == "libx265":
		a := []string{"-c:v", cod, "-preset", "medium", "-pix_fmt", "yuv420p", "-tag:v", "hvc1"}
		if taxa > 0 {
			return append(a, "-b:v", k, "-x265-params", "log-level=error:pass="+strconv.Itoa(passada)+":stats={passada}.x265")
		}
		return append(a, "-crf", strconv.Itoa(crf+4), "-x265-params", "log-level=error")
	case cod == "libsvtav1":
		a := []string{"-c:v", cod, "-preset", "8", "-pix_fmt", "yuv420p"}
		if taxa > 0 {
			return append(a, "-b:v", k)
		}
		return append(a, "-crf", strconv.Itoa(crf+8))
	case cod == "libaom-av1":
		a := []string{"-c:v", cod, "-cpu-used", "6", "-row-mt", "1", "-pix_fmt", "yuv420p"}
		if taxa > 0 {
			return append(a, "-b:v", k)
		}
		return append(a, "-crf", strconv.Itoa(crf+8), "-b:v", "0")
	case strings.HasSuffix(cod, "_vaapi"):
		if taxa > 0 {
			return []string{"-c:v", cod, "-b:v", k, "-maxrate", teto}
		}
		return []string{"-c:v", cod, "-qp", strconv.Itoa(crf)}
	default: // nvenc
		if taxa > 0 {
			return []string{"-c:v", cod, "-preset", "p5", "-rc", "vbr", "-b:v", k, "-maxrate", teto}
		}
		return []string{"-c:v", cod, "-preset", "p5", "-rc", "vbr", "-cq", strconv.Itoa(crf), "-b:v", "0"}
	}
}

// duasPassadas: só os codificadores por software que sabem fazer direito.
func duasPassadas(cod string) bool { return cod == "libx264" || cod == "libx265" }

// Planeja monta o que o ffmpeg vai rodar para o pedido.
func Planeja(m *Midia, p Pedido, tem map[string]bool) (*Plano, error) {
	if m.Tipo == "imagem" {
		if p.Predefinicao != "imagem" {
			return nil, fmt.Errorf("%q é predefinição de vídeo, e isto é uma imagem: use --predefinicao imagem", p.Predefinicao)
		}
		return planejaImagem(m, p, tem)
	}
	if m.Duracao == nil || *m.Duracao <= 0 {
		return nil, errors.New("não sei a duração deste vídeo: não dá para comprimir")
	}
	semAudio := p.SemAudio || m.Audio == nil
	entrada := func(antes ...string) []string {
		a := append([]string{}, comum...)
		a = append(a, antes...)
		return append(a, "-i", "{entrada}", "-map", "0:v:0")
	}
	sufixoDoAudio := ""
	if p.SemAudio && m.Audio != nil && p.Predefinicao != "sem-audio" {
		sufixoDoAudio = "-sem-audio"
	}

	switch p.Predefinicao {
	case "sem-audio":
		if m.Audio == nil {
			return nil, errors.New("este vídeo já não tem áudio")
		}
		ext := strings.ToLower(filepath.Ext(m.Arquivo))
		if !extensaoSimples.MatchString(ext) || ext == ".gif" {
			ext = ".mkv"
		}
		args := append(entrada(), "-c:v", "copy", "-an", "-sn", "-dn", "{saida}")
		return &Plano{Sufixo: "-sem-audio", Ext: ext, Passadas: [][]string{args}, Largura: m.Largura, Altura: m.Altura}, nil

	case "gif", "webp":
		fps, largura := p.FPS, p.Largura
		if fps == 0 {
			fps = map[string]int{"gif": 12, "webp": 15}[p.Predefinicao]
		}
		if largura == 0 {
			largura = map[string]int{"gif": 480, "webp": 640}[p.Predefinicao]
		}
		if fps < 1 || fps > 60 || largura < 16 || largura > 7680 {
			return nil, errors.New("--fps vai de 1 a 60 e --largura de 16 a 7680")
		}
		l, a := porLargura(m.Largura, m.Altura, largura)
		filtro := fmt.Sprintf("fps=%d,scale=%d:%d:flags=lanczos", fps, l, a)
		args := entrada()
		if p.Predefinicao == "gif" {
			if !tem["gif"] {
				return nil, errors.New("o ffmpeg desta máquina não tem o codificador gif")
			}
			args = append(args, "-vf", filtro+",split[a][b];[a]palettegen=stats_mode=diff[p];[b][p]paletteuse=dither=bayer:bayer_scale=5",
				"-c:v", "gif", "-loop", "0")
		} else {
			cod := "libwebp_anim"
			if !tem[cod] {
				cod = "libwebp"
			}
			if !tem[cod] {
				return nil, errors.New("o ffmpeg desta máquina não tem o codificador libwebp")
			}
			args = append(args, "-vf", filtro, "-c:v", cod, "-lossless", "0", "-q:v", "60", "-compression_level", "4", "-loop", "0")
		}
		args = append(args, "-an", "-sn", "-dn", "{saida}")
		return &Plano{Ext: "." + p.Predefinicao, Passadas: [][]string{args}, Largura: l, Altura: a}, nil

	case "menor", "equilibrio", "alta", "resolucao", "caber":
	case "imagem":
		return nil, errors.New("\"imagem\" é predefinição de imagem, e isto é um vídeo")
	default:
		return nil, fmt.Errorf("não conheço a predefinição %q", p.Predefinicao)
	}

	cod, err := codificadorDe(p.Codec, p.Acelerar, tem)
	if err != nil {
		return nil, err
	}
	plano := &Plano{Ext: ".mp4"}
	var crf int
	var audioBps int64
	if p.Predefinicao == "caber" {
		fps := m.FPS
		plano.VideoBps, plano.AudioBps, plano.Largura, plano.Altura, err = Cabe(p.AlvoMB, *m.Duracao, !semAudio, m.Largura, m.Altura, fps)
		if err != nil {
			return nil, err
		}
		audioBps = plano.AudioBps
		plano.Sufixo = "-" + numeroCurto(p.AlvoMB) + "mb"
	} else {
		q := porQualidade[p.Predefinicao]
		lado := q.ladoMaximo
		plano.Sufixo = q.sufixo
		if p.Predefinicao == "resolucao" {
			if p.Lado != 1080 && p.Lado != 720 && p.Lado != 480 {
				return nil, errors.New("--lado tem de ser 1080, 720 ou 480")
			}
			lado = p.Lado
			plano.Sufixo = fmt.Sprintf("-%dp", p.Lado)
		}
		crf, audioBps = q.crf, int64(q.audioK)*1000
		plano.Largura, plano.Altura = Dimensoes(m.Largura, m.Altura, lado)
	}
	plano.Sufixo += sufixoDoAudio

	// Os filtros: reduzir só se as dimensões mudam; na VAAPI o quadro sobe
	// para a placa depois de reduzido.
	var filtros []string
	if plano.Largura != m.Largura || plano.Altura != m.Altura {
		filtros = append(filtros, fmt.Sprintf("scale=%d:%d:flags=lanczos", plano.Largura, plano.Altura))
	}
	var antes []string
	if strings.HasSuffix(cod, "_vaapi") {
		antes = []string{"-vaapi_device", "/dev/dri/renderD128"}
		filtros = append(filtros, "format=nv12", "hwupload")
	}
	audio := []string{"-an"}
	if !semAudio {
		audio = []string{"-map", "0:a:0", "-c:a", "aac", "-b:a", strconv.FormatInt(audioBps/1000, 10) + "k"}
		if p.Predefinicao == "menor" || p.Predefinicao == "caber" {
			audio = append(audio, "-ac", "2")
		}
	}
	monta := func(passada, de int) []string {
		args := entrada(antes...)
		if len(filtros) > 0 {
			args = append(args, "-vf", strings.Join(filtros, ","))
		}
		args = append(args, argsDeVideo(cod, crf, plano.VideoBps, passada)...)
		if passada == 1 && de == 2 {
			// A primeira passada só mede: sem áudio, sem arquivo.
			return append(args, "-an", "-sn", "-dn", "-f", "null", "/dev/null")
		}
		args = append(args, audio...)
		return append(args, "-sn", "-dn", "-movflags", "+faststart", "{saida}")
	}
	if p.Predefinicao == "caber" && duasPassadas(cod) {
		plano.Passadas = [][]string{monta(1, 2), monta(2, 2)}
	} else {
		plano.Passadas = [][]string{monta(1, 1)}
	}
	return plano, nil
}

func planejaImagem(m *Midia, p Pedido, tem map[string]bool) (*Plano, error) {
	q := p.Qualidade
	if q == 0 {
		q = 80
	}
	if q < 1 || q > 100 {
		return nil, errors.New("--qualidade vai de 1 a 100")
	}
	if p.Largura != 0 && (p.Largura < 16 || p.Largura > 30000) {
		return nil, errors.New("--largura vai de 16 a 30000")
	}
	l, a := m.Largura, m.Altura
	args := append(append([]string{}, comum...), "-i", "{entrada}", "-map", "0:v:0", "-frames:v", "1")
	plano := &Plano{Sufixo: "-menor"}
	formato := p.Formato
	if formato == "" {
		formato = "jpeg"
	}
	// O AVIF pede dimensões pares; os outros só mudam se a pessoa pediu.
	if p.Largura > 0 && p.Largura < l {
		l, a = porLargura(m.Largura, m.Altura, p.Largura)
	} else if formato == "avif" {
		l, a = porLargura(m.Largura, m.Altura, 0)
	}
	if l != m.Largura || a != m.Altura {
		args = append(args, "-vf", fmt.Sprintf("scale=%d:%d:flags=lanczos", l, a))
	}
	plano.Largura, plano.Altura = l, a
	switch formato {
	case "jpeg":
		if !tem["mjpeg"] {
			return nil, errors.New("o ffmpeg desta máquina não tem o codificador mjpeg")
		}
		// A escala do JPEG no ffmpeg é ao contrário: 2 é o melhor, 31 o pior.
		escala := int(math.Round(31 - float64(q-1)*29/99))
		args = append(args, "-c:v", "mjpeg", "-q:v", strconv.Itoa(escala), "-pix_fmt", "yuvj420p", "-update", "1")
		plano.Ext = ".jpg"
	case "webp":
		if !tem["libwebp"] {
			return nil, errors.New("o ffmpeg desta máquina não tem o codificador libwebp")
		}
		args = append(args, "-c:v", "libwebp", "-quality", strconv.Itoa(q), "-compression_level", "6")
		plano.Ext = ".webp"
	case "avif":
		crf := strconv.Itoa(int(math.Round(63 - float64(q)*0.6)))
		switch {
		case tem["libaom-av1"]:
			args = append(args, "-c:v", "libaom-av1", "-still-picture", "1", "-crf", crf, "-b:v", "0", "-cpu-used", "6")
		case tem["libsvtav1"]:
			args = append(args, "-c:v", "libsvtav1", "-crf", crf, "-preset", "6")
		default:
			return nil, errors.New("o ffmpeg desta máquina não tem codificador de AV1 para fazer AVIF")
		}
		plano.Ext = ".avif"
	default:
		return nil, fmt.Errorf("não conheço o formato %q: use jpeg, webp ou avif", formato)
	}
	plano.Passadas = [][]string{append(args, "{saida}")}
	return plano, nil
}

// nomeLivre devolve o caminho da saída: ao lado do original, com o sufixo e a
// extensão, e um número se já existe ("férias-menor-2.mp4").
func nomeLivre(original, sufixo, ext string) string {
	pasta, nome := filepath.Split(original)
	raiz := strings.TrimSuffix(nome, filepath.Ext(nome))
	if raiz == "" {
		raiz = nome
	}
	for n := 1; ; n++ {
		candidato := filepath.Join(pasta, raiz+sufixo+ext)
		if n > 1 {
			candidato = filepath.Join(pasta, raiz+sufixo+"-"+strconv.Itoa(n)+ext)
		}
		if _, err := os.Lstat(candidato); err != nil {
			return candidato
		}
	}
}
