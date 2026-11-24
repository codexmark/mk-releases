package midia

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"mkcorte/internal/base"
)

// Tira é a resposta do `miniaturas`: os quadros pequenos que a linha do
// tempo mostra em cima de cada clipe.
type Tira struct {
	Arquivo string  `json:"arquivo"`
	Duracao float64 `json:"duracao_s"`
	Altura  int     `json:"altura"`
	Pasta   string  `json:"pasta"`
	DoCache bool    `json:"do_cache"`
	Quadros []Mini  `json:"quadros"`
}

// Mini é um quadro da tira: de que instante do vídeo e onde está o JPEG.
type Mini struct {
	Tempo   float64 `json:"tempo_s"`
	Arquivo string  `json:"arquivo"`
}

// Os padrões e os limites da tira.
const (
	QuantasPadrao = 12
	AlturaPadrao  = 54
	maximoDeMinis = 60
	nomeDaTira    = "tira.json"
	validadeCache = 30 * 24 * time.Hour
)

// Quantos ffmpeg rodam ao mesmo tempo para tirar os quadros (os testes põem 1).
var emParalelo = 4

// chaveDaTira identifica a tira de um arquivo: muda se o caminho, o tamanho,
// a data de modificação ou o pedido (quantas, altura) mudarem.
func chaveDaTira(caminho string, tamanho int64, modificado time.Time, quantas, altura int) string {
	soma := sha256.Sum256([]byte(caminho + "\x00" + strconv.FormatInt(tamanho, 10) + "\x00" + strconv.FormatInt(modificado.UnixNano(), 10) +
		"\x00" + strconv.Itoa(quantas) + "\x00" + strconv.Itoa(altura)))
	return hex.EncodeToString(soma[:16])
}

// Miniaturas gera (ou acha no cache) a tira de quadros de um vídeo. O cache
// fica em base.Cache(), uma pasta por chave.
func Miniaturas(arquivo string, quantas, altura int) (*Tira, error) {
	if quantas <= 0 {
		quantas = QuantasPadrao
	}
	if altura <= 0 {
		altura = AlturaPadrao
	}
	if quantas > maximoDeMinis {
		return nil, fmt.Errorf("miniaturas demais (%d; o máximo é %d)", quantas, maximoDeMinis)
	}
	if altura < 16 || altura > 360 {
		return nil, fmt.Errorf("a altura das miniaturas vai de 16 a 360, e veio %d", altura)
	}
	altura = altura / 2 * 2
	abs, err := filepath.Abs(arquivo)
	if err != nil {
		return nil, fmt.Errorf("não entendi o caminho %s", arquivo)
	}
	fi, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("não achei o arquivo %s", base.Til(abs))
	}
	pasta := base.Cache(chaveDaTira(abs, fi.Size(), fi.ModTime(), quantas, altura))
	if tira := tiraPronta(pasta, abs); tira != nil {
		agora := time.Now()
		_ = os.Chtimes(pasta, agora, agora) // usada há pouco: fica
		return tira, nil
	}

	var tira *Tira
	err = comAtalhos([]string{abs}, func(_ string, links []string) error {
		fonte, err := sonda(abs, links[0])
		if err != nil {
			return err
		}
		if !base.Tem("ffmpeg") {
			return errors.New("não achei o ffmpeg nesta máquina: instale o pacote ffmpeg")
		}
		if err := os.MkdirAll(base.Cache(), 0o755); err != nil {
			return fmt.Errorf("não consegui criar o cache (%s): %v", base.Til(base.Cache()), err)
		}
		limpaVelhas(base.Cache(), time.Now().Add(-validadeCache))
		fazendo, err := os.MkdirTemp(base.Cache(), ".fazendo-")
		if err != nil {
			return fmt.Errorf("não consegui escrever no cache (%s): %v", base.Til(base.Cache()), err)
		}
		defer os.RemoveAll(fazendo)

		// Um quadro do meio de cada fatia do vídeo. Cada um é uma chamada
		// curta do ffmpeg, que pula direto para o ponto.
		tempos := make([]float64, quantas)
		feito := make([]bool, quantas)
		vagas := make(chan struct{}, max(1, emParalelo))
		var espera sync.WaitGroup
		for i := range tempos {
			tempos[i] = float64(int((float64(i)+0.5)*fonte.Duracao/float64(quantas)*1000)) / 1000
			espera.Add(1)
			vagas <- struct{}{}
			go func(i int) {
				defer espera.Done()
				defer func() { <-vagas }()
				destino := filepath.Join(fazendo, nomeDaMini(i))
				_, err := base.Roda("ffmpeg", 30*time.Second, "-hide_banner", "-nostdin", "-loglevel", "error", "-y", "-ss", seg(tempos[i]), "-i", links[0],
					"-map", "0:"+strconv.Itoa(fonte.Trilha), "-frames:v", "1", "-vf", "scale=-2:"+strconv.Itoa(altura), "-c:v", "mjpeg", "-q:v", "5",
					"-f", "image2", "-update", "1", destino)
				if fi, e := os.Stat(destino); err == nil && e == nil && fi.Size() > 0 {
					feito[i] = true
				}
			}(i)
		}
		espera.Wait()

		tira = &Tira{Arquivo: abs, Duracao: fonte.Duracao, Altura: altura, Pasta: pasta, Quadros: []Mini{}}
		for i, ok := range feito {
			if ok {
				tira.Quadros = append(tira.Quadros, Mini{Tempo: tempos[i], Arquivo: filepath.Join(pasta, nomeDaMini(i))})
			}
		}
		if len(tira.Quadros) == 0 {
			return errors.New("o ffmpeg não conseguiu tirar nenhum quadro deste vídeo")
		}
		ficha, _ := json.Marshal(tira)
		if err := os.WriteFile(filepath.Join(fazendo, nomeDaTira), ficha, 0o644); err != nil {
			return fmt.Errorf("não consegui escrever no cache: %v", err)
		}
		// A pasta entra no lugar de uma vez. Se outra janela chegou antes, a
		// dela serve.
		if err := os.Rename(fazendo, pasta); err != nil {
			if pronta := tiraPronta(pasta, abs); pronta != nil {
				tira = pronta
				return nil
			}
			_ = os.RemoveAll(pasta)
			if err := os.Rename(fazendo, pasta); err != nil {
				return fmt.Errorf("não consegui guardar as miniaturas no cache: %v", err)
			}
		}
		return nil
	})
	return tira, err
}

func nomeDaMini(i int) string { return fmt.Sprintf("q%02d.jpg", i+1) }

// tiraPronta lê uma tira do cache, se está inteira: a ficha é desse arquivo
// e todos os quadros dela existem.
func tiraPronta(pasta, arquivo string) *Tira {
	b, err := os.ReadFile(filepath.Join(pasta, nomeDaTira))
	if err != nil {
		return nil
	}
	var t Tira
	if json.Unmarshal(b, &t) != nil || t.Arquivo != arquivo || len(t.Quadros) == 0 {
		return nil
	}
	for i := range t.Quadros {
		// O caminho é refeito a partir da pasta: a ficha não manda ler fora dela.
		t.Quadros[i].Arquivo = filepath.Join(pasta, filepath.Base(t.Quadros[i].Arquivo))
		if fi, err := os.Stat(t.Quadros[i].Arquivo); err != nil || !fi.Mode().IsRegular() || fi.Size() == 0 {
			return nil
		}
	}
	t.Pasta, t.DoCache = pasta, true
	return &t
}

// limpaVelhas apaga do cache as tiras que ninguém usa desde `antes` (e os
// restos de uma geração interrompida).
func limpaVelhas(cache string, antes time.Time) {
	entradas, err := os.ReadDir(cache)
	if err != nil {
		return
	}
	for _, e := range entradas {
		fi, err := e.Info()
		if err != nil || !e.IsDir() || fi.ModTime().After(antes) {
			continue
		}
		_ = os.RemoveAll(filepath.Join(cache, e.Name()))
	}
}
