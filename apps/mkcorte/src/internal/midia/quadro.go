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

	"mkcorte/internal/base"
)

// Foto é a resposta do `quadro`.
type Foto struct {
	Ok      bool    `json:"ok"`
	Saida   string  `json:"saida"`
	Tempo   float64 `json:"tempo_s"`
	Tamanho int64   `json:"tamanho_bytes"`
}

// carimbo escreve um tempo para um nome de arquivo: "0-01-23.450".
func carimbo(t float64) string {
	ms := int64(math.Round(t * 1000))
	return fmt.Sprintf("%d-%02d-%02d.%03d", ms/3600000, ms/60000%60, ms/1000%60, ms%1000)
}

// NomeDoQuadro é o nome que o quadro ganha quando ninguém escolheu um: ao
// lado do vídeo, "nome (quadro 0-01-23.450).png".
func NomeDoQuadro(video string, tempo float64) string {
	pasta, nome := filepath.Split(video)
	raiz := strings.TrimSuffix(nome, filepath.Ext(nome))
	if raiz == "" {
		raiz = nome
	}
	return filepath.Join(pasta, raiz+" (quadro "+carimbo(tempo)+").png")
}

// Quadro salva o quadro do vídeo no instante dado como PNG, ao lado do vídeo
// (ou em `saida`), sem nunca escrever em cima de um arquivo que existe.
func Quadro(arquivo string, tempo float64, saida string) (*Foto, error) {
	if math.IsNaN(tempo) || math.IsInf(tempo, 0) || tempo < 0 {
		return nil, errors.New("o tempo do quadro tem de ser um número de segundos, de zero para cima")
	}
	abs, err := filepath.Abs(arquivo)
	if err != nil {
		return nil, fmt.Errorf("não entendi o caminho %s", arquivo)
	}
	var foto *Foto
	err = comAtalhos([]string{abs}, func(_ string, links []string) error {
		fonte, err := sonda(abs, links[0])
		if err != nil {
			return err
		}
		if tempo >= fonte.Duracao {
			return fmt.Errorf("o vídeo acaba em %s s: não há quadro em %s s", seg(fonte.Duracao), seg(tempo))
		}
		tempo = math.Round(tempo*1000) / 1000
		desejado := NomeDoQuadro(abs, tempo)
		if saida != "" {
			if desejado, err = filepath.Abs(saida); err != nil {
				return fmt.Errorf("não entendi o caminho %s", saida)
			}
			desejado = comExtensao(desejado, ".png")
		}
		destino := filepath.Dir(desejado)
		if !base.Tem("ffmpeg") {
			return errors.New("não achei o ffmpeg nesta máquina: instale o pacote ffmpeg")
		}
		parcial, err := os.CreateTemp(destino, ".mkcorte-*.parcial")
		if err != nil {
			return fmt.Errorf("não dá para escrever em %s: %v", base.Til(destino), err)
		}
		parcial.Close()
		defer os.Remove(parcial.Name())
		// -ss antes da entrada, decodificando: o ffmpeg pula para o
		// quadro-chave anterior e anda até o quadro pedido.
		if _, err := base.Roda("ffmpeg", 2*time.Minute, "-hide_banner", "-nostdin", "-loglevel", "error", "-y", "-ss", seg(tempo), "-i", links[0],
			"-map", "0:"+strconv.Itoa(fonte.Trilha), "-frames:v", "1", "-c:v", "png", "-f", "image2", "-update", "1", parcial.Name()); err != nil {
			return fmt.Errorf("o ffmpeg não conseguiu tirar o quadro: %v", err)
		}
		fi, err := os.Stat(parcial.Name())
		if err != nil || fi.Size() == 0 {
			return errors.New("não há quadro nesse ponto do vídeo")
		}
		_ = os.Chmod(parcial.Name(), 0o644)
		final, err := daNome(parcial.Name(), desejado)
		if err != nil {
			return err
		}
		foto = &Foto{Ok: true, Saida: final, Tempo: tempo, Tamanho: fi.Size()}
		return nil
	})
	return foto, err
}
