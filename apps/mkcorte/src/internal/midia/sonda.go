// Package midia é o motor do MkCorte: pergunta ao ffprobe o que um vídeo é,
// gera a tira de quadros da linha do tempo, monta os argumentos do ffmpeg
// para exportar uma edição (função pura, em plano.go), roda, acompanha o
// progresso e cuida da saída (nunca em cima de um arquivo que existe). O
// contrato está em ../../../CONTRATO.md.
package midia

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"mkcorte/internal/base"
)

// Fonte é a resposta do `sondar`: um vídeo de origem.
type Fonte struct {
	Arquivo string  `json:"arquivo"`
	Tamanho int64   `json:"tamanho_bytes"`
	Duracao float64 `json:"duracao_s"`
	// Como o vídeo aparece na tela: já com a rotação e o pixel não quadrado
	// aplicados.
	Largura int      `json:"largura"`
	Altura  int      `json:"altura"`
	FPS     *float64 `json:"fps"`
	// A taxa de quadros como o arquivo a escreve ("30000/1001"), ou "".
	Fracao   string `json:"fps_fracao"`
	Codec    string `json:"codec"`
	Perfil   string `json:"perfil"`
	Pixel    string `json:"formato_de_pixel"`
	SAR      string `json:"sar"`
	Rotacao  int    `json:"rotacao"`
	Taxa     *int64 `json:"taxa_bps"`
	Trilha   int    `json:"trilha"`
	TemAudio bool   `json:"tem_audio"`
	Audio    *Audio `json:"audio"`
	// Só com --quadros-chave.
	QuadrosChave []float64 `json:"quadros_chave,omitempty"`

	// Os pixels gravados (antes da rotação e do SAR) e o nível: só para saber
	// se duas fontes têm os mesmos parâmetros.
	gravado [2]int
	nivel   int
}

// Audio é a primeira trilha de áudio.
type Audio struct {
	Codec      string `json:"codec"`
	Taxa       *int64 `json:"taxa_bps"`
	Canais     int    `json:"canais"`
	Amostragem int    `json:"amostragem_hz"`
	Trilha     int    `json:"trilha"`
}

var extensaoSimples = regexp.MustCompile(`^\.[a-z0-9]{1,8}$`)

// nomeDoAtalho é o nome neutro que o ffmpeg e o ffprobe veem no lugar do
// arquivo k: "fonte0.mp4". O nome de verdade pode começar com "-", ter aspas,
// ":" (que eles leem como protocolo) ou "%".
func nomeDoAtalho(k int, arquivo string) string {
	ext := strings.ToLower(filepath.Ext(arquivo))
	if !extensaoSimples.MatchString(ext) {
		ext = ""
	}
	return "fonte" + strconv.Itoa(k) + ext
}

// comAtalhos chama f com uma pasta temporária em que cada arquivo tem um link
// simbólico de nome neutro (veja nomeDoAtalho). `links` vem na ordem dos
// arquivos, com o caminho absoluto de cada link.
func comAtalhos(arquivos []string, f func(pasta string, links []string) error) error {
	absolutos := make([]string, len(arquivos))
	for i, arquivo := range arquivos {
		abs, err := filepath.Abs(arquivo)
		if err != nil {
			return fmt.Errorf("não entendi o caminho %s", arquivo)
		}
		fi, err := os.Stat(abs)
		if err != nil {
			return fmt.Errorf("não achei o arquivo %s", base.Til(abs))
		}
		if !fi.Mode().IsRegular() {
			return fmt.Errorf("%s não é um arquivo", base.Til(abs))
		}
		absolutos[i] = abs
	}
	pasta, err := os.MkdirTemp("", "mkcorte-")
	if err != nil {
		return fmt.Errorf("não consegui criar uma pasta temporária: %v", err)
	}
	defer os.RemoveAll(pasta)
	links := make([]string, len(absolutos))
	for i, abs := range absolutos {
		links[i] = filepath.Join(pasta, nomeDoAtalho(i, abs))
		if err := os.Symlink(abs, links[i]); err != nil {
			return fmt.Errorf("não consegui preparar o arquivo: %v", err)
		}
	}
	return f(pasta, links)
}

// Sonda pergunta ao ffprobe o que o arquivo é. Com quadrosChave, lista também
// os tempos dos quadros-chave (lê o arquivo inteiro: demora num vídeo longo).
func Sonda(arquivo string, quadrosChave bool) (*Fonte, error) {
	var fonte *Fonte
	err := comAtalhos([]string{arquivo}, func(_ string, links []string) error {
		var err error
		if fonte, err = sonda(arquivo, links[0]); err != nil || !quadrosChave {
			return err
		}
		fonte.QuadrosChave, err = chaves(links[0], fonte.Trilha)
		return err
	})
	return fonte, err
}

func sonda(arquivo, link string) (*Fonte, error) {
	abs, _ := filepath.Abs(arquivo)
	fi, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("não achei o arquivo %s", base.Til(abs))
	}
	if !base.Tem("ffprobe") {
		return nil, errors.New("não achei o ffprobe nesta máquina: o MkCorte precisa do pacote ffmpeg")
	}
	saida, err := base.Roda("ffprobe", 30*time.Second, "-v", "error", "-of", "json", "-show_format", "-show_streams", "-i", link)
	if err != nil {
		return nil, fmt.Errorf("o ffprobe não entendeu %s: não parece um vídeo (%v)", filepath.Base(abs), err)
	}
	f, err := interpreta(saida)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", filepath.Base(abs), err)
	}
	f.Arquivo, f.Tamanho = abs, fi.Size()
	if f.Taxa == nil && f.Duracao > 0 {
		t := int64(float64(f.Tamanho) * 8 / f.Duracao)
		f.Taxa = &t
	}
	return f, nil
}

// chaves lista os tempos dos quadros-chave da trilha de vídeo, lendo só os
// pacotes (sem decodificar).
func chaves(link string, trilha int) ([]float64, error) {
	saida, err := base.Roda("ffprobe", 5*time.Minute, "-v", "error", "-select_streams", strconv.Itoa(trilha),
		"-show_entries", "packet=pts_time,flags", "-of", "csv=p=0", "-i", link)
	if err != nil {
		return nil, fmt.Errorf("não consegui listar os quadros-chave: %v", err)
	}
	tempos := []float64{}
	for _, linha := range strings.Split(string(saida), "\n") {
		tempo, marcas, ok := strings.Cut(strings.TrimSpace(linha), ",")
		if !ok || !strings.HasPrefix(marcas, "K") {
			continue
		}
		v, err := strconv.ParseFloat(tempo, 64)
		if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
			continue
		}
		tempos = append(tempos, math.Round(math.Max(0, v)*1000)/1000)
	}
	return tempos, nil
}

type trilhaDoProbe struct {
	Indice     int    `json:"index"`
	Tipo       string `json:"codec_type"`
	Codec      string `json:"codec_name"`
	Perfil     string `json:"profile"`
	Nivel      int    `json:"level"`
	Largura    int    `json:"width"`
	Altura     int    `json:"height"`
	Pixel      string `json:"pix_fmt"`
	SAR        string `json:"sample_aspect_ratio"`
	TaxaMedia  string `json:"avg_frame_rate"`
	TaxaReal   string `json:"r_frame_rate"`
	Quadros    string `json:"nb_frames"`
	Duracao    string `json:"duration"`
	Taxa       string `json:"bit_rate"`
	Canais     int    `json:"channels"`
	Amostragem string `json:"sample_rate"`
	Disposicao struct {
		Capa int `json:"attached_pic"`
	} `json:"disposition"`
	Laterais []struct {
		Rotacao *float64 `json:"rotation"`
	} `json:"side_data_list"`
	Marcas map[string]string `json:"tags"`
}

type respostaDoProbe struct {
	Trilhas []trilhaDoProbe `json:"streams"`
	Formato struct {
		Nome    string `json:"format_name"`
		Duracao string `json:"duration"`
		Taxa    string `json:"bit_rate"`
	} `json:"format"`
}

func numero(s string) *float64 {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 {
		return nil
	}
	return &v
}

func inteiro(s string) *int64 {
	v := numero(s)
	if v == nil {
		return nil
	}
	n := int64(*v)
	return &n
}

var fracaoInteira = regexp.MustCompile(`^[0-9]{1,9}/[0-9]{1,9}$`)

// fracao lê "30000/1001": o valor (com três casas) e o texto, se for uma
// fração de inteiros que dá uma taxa de quadros plausível.
func fracao(s string) (*float64, string) {
	s = strings.TrimSpace(s)
	a, b, ok := strings.Cut(s, "/")
	if !ok {
		return nil, ""
	}
	n, d := numero(a), numero(b)
	if n == nil || d == nil {
		return nil, ""
	}
	v := math.Round(*n / *d * 1000) / 1000
	if v <= 0 || v > 1000 {
		return nil, ""
	}
	if !fracaoInteira.MatchString(s) {
		s = ""
	}
	return &v, s
}

// proporcaoDoPixel lê o SAR ("4:3"). O que não dá para ler, ou "0:1", é pixel
// quadrado.
func proporcaoDoPixel(s string) (float64, string) {
	a, b, ok := strings.Cut(strings.TrimSpace(s), ":")
	n, d := numero(a), numero(b)
	if !ok || n == nil || d == nil || *n == *d || *n / *d > 10 || *d / *n > 10 {
		return 1, "1:1"
	}
	return *n / *d, strings.TrimSpace(s)
}

// interpreta lê o JSON do ffprobe.
func interpreta(saida []byte) (*Fonte, error) {
	var r respostaDoProbe
	if err := json.Unmarshal(saida, &r); err != nil {
		return nil, errors.New("o ffprobe respondeu algo que não entendi")
	}
	var video, audio *trilhaDoProbe
	for i := range r.Trilhas {
		t := &r.Trilhas[i]
		switch {
		case t.Tipo == "video" && video == nil && t.Disposicao.Capa == 0 && t.Largura > 0 && t.Altura > 0:
			video = t
		case t.Tipo == "audio" && audio == nil:
			audio = t
		}
	}
	if video == nil {
		if audio != nil {
			return nil, errors.New("este arquivo só tem áudio: o MkCorte corta vídeos")
		}
		return nil, errors.New("este arquivo não tem imagem: não parece um vídeo")
	}
	dur := numero(r.Formato.Duracao)
	if dur == nil {
		dur = numero(video.Duracao)
	}
	nome := r.Formato.Nome
	if nome == "image2" || strings.HasSuffix(nome, "_pipe") || dur == nil || strings.TrimSpace(video.Quadros) == "1" {
		return nil, errors.New("isto é uma imagem parada, não um vídeo")
	}
	f := &Fonte{Duracao: math.Round(*dur*1000) / 1000, Codec: video.Codec, Perfil: video.Perfil, Pixel: video.Pixel, Trilha: video.Indice,
		gravado: [2]int{video.Largura, video.Altura}, nivel: video.Nivel}
	// O pixel que não é quadrado (DVD, câmeras antigas): a largura na tela é
	// a gravada vezes o SAR.
	var sar float64
	sar, f.SAR = proporcaoDoPixel(video.SAR)
	f.Largura, f.Altura = int(math.Round(float64(video.Largura)*sar)), video.Altura
	// Vídeo de celular em pé: gravado deitado, com a rotação anotada.
	rot := 0.0
	for _, l := range video.Laterais {
		if l.Rotacao != nil {
			rot = *l.Rotacao
		}
	}
	if v := numero(video.Marcas["rotate"]); v != nil && rot == 0 {
		rot = *v
	}
	if !math.IsNaN(rot) && !math.IsInf(rot, 0) {
		f.Rotacao = ((int(math.Round(rot)) % 360) + 360) % 360
	}
	if f.Rotacao%180 == 90 {
		f.Largura, f.Altura = f.Altura, f.Largura
	}
	if f.FPS, f.Fracao = fracao(video.TaxaMedia); f.FPS == nil {
		f.FPS, f.Fracao = fracao(video.TaxaReal)
	}
	if audio != nil {
		f.TemAudio = true
		f.Audio = &Audio{Codec: audio.Codec, Taxa: inteiro(audio.Taxa), Canais: audio.Canais, Trilha: audio.Indice}
		if hz := inteiro(audio.Amostragem); hz != nil {
			f.Audio.Amostragem = int(*hz)
		}
	}
	f.Taxa = inteiro(r.Formato.Taxa)
	return f, nil
}
