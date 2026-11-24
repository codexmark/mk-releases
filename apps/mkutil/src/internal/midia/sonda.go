// Package midia é o MkComprimir: pergunta ao ffprobe o que um vídeo ou uma
// imagem é, monta os argumentos do ffmpeg para cada predefinição, roda,
// acompanha o progresso e cuida da saída (nunca em cima de um arquivo que
// existe). O contrato está em ../../../CONTRATO.md.
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

	"mkutil/internal/base"
)

// Midia é a resposta do `midia`.
type Midia struct {
	Arquivo string   `json:"arquivo"`
	Tipo    string   `json:"tipo"`
	Tamanho int64    `json:"tamanho_bytes"`
	Duracao *float64 `json:"duracao_s"`
	Largura int      `json:"largura"`
	Altura  int      `json:"altura"`
	FPS     *float64 `json:"fps"`
	Codec   string   `json:"codec"`
	Taxa    *int64   `json:"taxa_bps"`
	Audio   *Audio   `json:"audio"`
}

// Audio é a primeira trilha de áudio.
type Audio struct {
	Codec  string `json:"codec"`
	Taxa   *int64 `json:"taxa_bps"`
	Canais int    `json:"canais"`
}

var extensaoSimples = regexp.MustCompile(`^\.[a-z0-9]{1,8}$`)

// comAtalho chama f com um link simbólico para o arquivo, de nome neutro
// ("entrada.mp4"), numa pasta temporária. É esse o nome que o ffmpeg e o
// ffprobe recebem: o de verdade pode começar com "-", ter "%" ou "*" (que
// eles leem como padrão de sequência de imagens) ou parecer um protocolo
// ("concat:…"). A pasta serve também para os arquivos da primeira passada.
func comAtalho(arquivo string, f func(link, pasta string) error) error {
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
	pasta, err := os.MkdirTemp("", "mkcomprimir-")
	if err != nil {
		return fmt.Errorf("não consegui criar uma pasta temporária: %v", err)
	}
	defer os.RemoveAll(pasta)
	ext := strings.ToLower(filepath.Ext(abs))
	if !extensaoSimples.MatchString(ext) {
		ext = ""
	}
	link := filepath.Join(pasta, "entrada"+ext)
	if err := os.Symlink(abs, link); err != nil {
		return fmt.Errorf("não consegui preparar o arquivo: %v", err)
	}
	return f(link, pasta)
}

// Sonda pergunta ao ffprobe o que o arquivo é.
func Sonda(arquivo string) (*Midia, error) {
	var m *Midia
	err := comAtalho(arquivo, func(link, _ string) error {
		var err error
		m, err = sonda(arquivo, link)
		return err
	})
	return m, err
}

func sonda(arquivo, link string) (*Midia, error) {
	abs, _ := filepath.Abs(arquivo)
	fi, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("não achei o arquivo %s", base.Til(abs))
	}
	if !base.Tem("ffprobe") {
		return nil, errors.New("não achei o ffprobe nesta máquina: o MkComprimir precisa do pacote ffmpeg")
	}
	saida, err := base.Roda("ffprobe", 30*time.Second, nil, "-v", "error", "-of", "json", "-show_format", "-show_streams", "-i", link)
	if err != nil {
		return nil, fmt.Errorf("o ffprobe não entendeu o arquivo: não parece vídeo nem imagem (%v)", err)
	}
	m, err := interpreta(saida)
	if err != nil {
		return nil, err
	}
	m.Arquivo, m.Tamanho = abs, fi.Size()
	if m.Tipo == "video" && m.Taxa == nil && m.Duracao != nil && *m.Duracao > 0 {
		t := int64(float64(m.Tamanho) * 8 / *m.Duracao)
		if m.Audio != nil && m.Audio.Taxa != nil && *m.Audio.Taxa < t {
			t -= *m.Audio.Taxa
		}
		m.Taxa = &t
	}
	return m, nil
}

type trilhaDoProbe struct {
	Tipo       string `json:"codec_type"`
	Codec      string `json:"codec_name"`
	Largura    int    `json:"width"`
	Altura     int    `json:"height"`
	TaxaMedia  string `json:"avg_frame_rate"`
	TaxaReal   string `json:"r_frame_rate"`
	Quadros    string `json:"nb_frames"`
	Duracao    string `json:"duration"`
	Taxa       string `json:"bit_rate"`
	Canais     int    `json:"channels"`
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

// fracao lê "30000/1001".
func fracao(s string) *float64 {
	a, b, ok := strings.Cut(s, "/")
	if !ok {
		return numero(s)
	}
	n, d := numero(a), numero(b)
	if n == nil || d == nil {
		return nil
	}
	v := math.Round(*n / *d * 1000) / 1000
	if v <= 0 || v > 1000 {
		return nil
	}
	return &v
}

// interpreta lê o JSON do ffprobe.
func interpreta(saida []byte) (*Midia, error) {
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
			return nil, errors.New("este arquivo só tem áudio: o MkComprimir encolhe vídeos e imagens")
		}
		return nil, errors.New("este arquivo não tem imagem: não parece vídeo nem foto")
	}
	m := &Midia{Largura: video.Largura, Altura: video.Altura, Codec: video.Codec}
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
	if int(math.Abs(math.Round(rot)))%180 == 90 {
		m.Largura, m.Altura = m.Altura, m.Largura
	}
	dur := numero(r.Formato.Duracao)
	if dur == nil {
		dur = numero(video.Duracao)
	}
	nome := r.Formato.Nome
	parado := nome == "image2" || strings.HasSuffix(nome, "_pipe") || dur == nil || strings.TrimSpace(video.Quadros) == "1"
	if parado {
		m.Tipo = "imagem"
		return m, nil
	}
	m.Tipo, m.Duracao = "video", dur
	if m.FPS = fracao(video.TaxaMedia); m.FPS == nil {
		m.FPS = fracao(video.TaxaReal)
	}
	if audio != nil {
		m.Audio = &Audio{Codec: audio.Codec, Taxa: inteiro(audio.Taxa), Canais: audio.Canais}
	}
	m.Taxa = inteiro(video.Taxa)
	if m.Taxa == nil {
		if total := inteiro(r.Formato.Taxa); total != nil {
			t := *total
			if m.Audio != nil && m.Audio.Taxa != nil && *m.Audio.Taxa < t {
				t -= *m.Audio.Taxa
			}
			m.Taxa = &t
		}
	}
	return m, nil
}
