// SPDX-License-Identifier: GPL-3.0-or-later

package rede

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"mktunes/internal/app"
)

const (
	pastaDeEspera = ".mktunes-incomplete"
	blocoPadrao   = 1 << 20
	passoDeAviso  = 0.01
	// parado é quanto tempo sem chegar um byte até a transferência ser cortada.
	parado = 30 * time.Second
	// fresco: o que foi escrito há menos que isso pode ser um download em andamento.
	fresco = time.Minute
)

// Arquivos baixa para uma pasta de espera escondida e só move o arquivo para o
// lugar quando ele está inteiro.
type Arquivos struct {
	HTTP *http.Client
	// Raiz é a pasta de música do MkTunes; nada é escrito fora dela.
	Raiz string
	// Bloco é o tamanho de cada pedido por faixa de bytes (padrão: 1 MB).
	Bloco int64
}

func interrompida(causa error) error {
	return fmt.Errorf("%w: %v", &app.ErroDownload{Motivo: "Conexão interrompida"}, causa)
}

// dentro devolve o caminho absoluto de relativo, se ele fica mesmo sob a raiz.
func (a Arquivos) dentro(relativo string) (string, bool) {
	raiz := filepath.Clean(a.Raiz)
	alvo := filepath.Join(raiz, relativo)
	if raiz == "" || raiz == "/" || relativo == "" || !strings.HasPrefix(alvo, raiz+string(filepath.Separator)) {
		return "", false
	}
	return alvo, true
}

// Busca salva o endereço em relativo, sob a raiz, e devolve o caminho absoluto.
func (a Arquivos) Busca(ctx context.Context, endereco, relativo string, progresso func(float64)) (string, error) {
	alvo, ok := a.dentro(relativo)
	if !ok {
		return "", &app.ErroDownload{Motivo: "Nome de arquivo inválido"}
	}
	if u, erro := url.Parse(endereco); erro != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return "", &app.ErroDownload{Motivo: "Endereço inválido"}
	}
	espera := filepath.Join(a.Raiz, pastaDeEspera)
	if erro := os.MkdirAll(espera, 0o755); erro != nil {
		return "", erro
	}
	nome := make([]byte, 12)
	_, _ = rand.Read(nome)
	parcial := filepath.Join(espera, hex.EncodeToString(nome)+".part")
	defer os.Remove(parcial)
	if erro := a.baixa(ctx, endereco, parcial, progresso); erro != nil {
		return "", erro
	}
	if erro := os.MkdirAll(filepath.Dir(alvo), 0o755); erro != nil {
		return "", erro
	}
	if erro := os.Rename(parcial, alvo); erro != nil {
		return "", erro
	}
	return alvo, nil
}

// LimpaIncompletos apaga, na subida, os pedaços de downloads que o sistema
// cortou; o que foi escrito no último minuto fica (pode já ser um download novo).
func (a Arquivos) LimpaIncompletos() {
	espera := filepath.Join(a.Raiz, pastaDeEspera)
	entradas, _ := os.ReadDir(espera)
	corte := time.Now().Add(-fresco)
	for _, e := range entradas {
		if info, erro := e.Info(); erro == nil && info.Mode().IsRegular() && info.ModTime().Before(corte) {
			_ = os.Remove(filepath.Join(espera, e.Name()))
		}
	}
}

func (a Arquivos) Existe(caminho string) bool {
	info, erro := os.Stat(caminho)
	return erro == nil && info.Mode().IsRegular()
}

// Apaga remove um arquivo baixado; só o que está sob a raiz.
func (a Arquivos) Apaga(caminho string) {
	raiz := filepath.Clean(a.Raiz)
	if limpo := filepath.Clean(caminho); raiz != "" && raiz != "/" && strings.HasPrefix(limpo, raiz+string(filepath.Separator)) {
		_ = os.Remove(limpo)
	}
}

// avisos informa o progresso em passos de pelo menos 1 %, sem chegar a 1 antes
// de o arquivo estar completo.
type avisos struct {
	avisa   func(float64)
	total   int64 // -1: não se sabe
	avisado float64
}

func (p *avisos) em(bytes int64) {
	if p.total <= 0 {
		return
	}
	if fracao := float64(bytes) / float64(p.total); fracao-p.avisado >= passoDeAviso && fracao < 1 {
		p.avisado = fracao
		p.avisa(fracao)
	}
}

// baixa pede o arquivo em faixas de bytes: a CDN do YouTube segura as
// transferências inteiras na velocidade de tocar, mas serve as faixas a toda.
// Um servidor sem suporte a faixas recebe um pedido só.
func (a Arquivos) baixa(ctx context.Context, endereco, em string, progresso func(float64)) error {
	bloco := a.Bloco
	if bloco <= 0 {
		bloco = blocoPadrao
	}
	saida, erro := os.OpenFile(em, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if erro != nil {
		return erro
	}
	defer saida.Close()
	avisos := &avisos{avisa: progresso, total: -1}
	var posicao int64
	for {
		acabou, erro := a.pedaco(ctx, endereco, saida, &posicao, bloco, avisos)
		if erro != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return erro
		}
		if acabou {
			break
		}
	}
	if erro := saida.Close(); erro != nil {
		return erro
	}
	progresso(1)
	return nil
}

func (a Arquivos) pedaco(ctx context.Context, endereco string, saida *os.File, posicao *int64, bloco int64, avisos *avisos) (bool, error) {
	ctx, cancela := context.WithCancel(ctx)
	defer cancela()
	vigia := time.AfterFunc(parado, cancela)
	defer vigia.Stop()
	pedido, erro := http.NewRequestWithContext(ctx, http.MethodGet, endereco, nil)
	if erro != nil {
		return false, &app.ErroDownload{Motivo: "Endereço inválido"}
	}
	pedido.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", *posicao, *posicao+bloco-1))
	resposta, erro := a.HTTP.Do(pedido)
	if erro != nil {
		return false, interrompida(erro)
	}
	defer resposta.Body.Close()
	switch resposta.StatusCode {
	case http.StatusPartialContent:
		if avisos.total < 0 {
			if _, total, tem := strings.Cut(resposta.Header.Get("Content-Range"), "/"); tem {
				if n, erro := strconv.ParseInt(total, 10, 64); erro == nil {
					avisos.total = n
				}
			}
		}
	case http.StatusOK:
		// O arquivo inteiro, mesmo tendo pedido uma faixa adiante: recomeça do byte 0.
		if *posicao > 0 {
			if erro := saida.Truncate(0); erro != nil {
				return false, erro
			}
			if _, erro := saida.Seek(0, io.SeekStart); erro != nil {
				return false, erro
			}
		}
		*posicao = 0
		avisos.avisado = 0
		avisos.total = resposta.ContentLength
	case http.StatusRequestedRangeNotSatisfiable:
		if *posicao > 0 && avisos.total < 0 {
			return true, nil
		}
		return false, &app.ErroDownload{Motivo: "HTTP 416"}
	default:
		return false, &app.ErroDownload{Motivo: "HTTP " + strconv.Itoa(resposta.StatusCode)}
	}
	var lidos int64
	memoria := make([]byte, 64*1024)
	for {
		n, erroDeLeitura := resposta.Body.Read(memoria)
		if n > 0 {
			vigia.Reset(parado)
			if _, erro := saida.Write(memoria[:n]); erro != nil {
				return false, erro
			}
			lidos += int64(n)
			*posicao += int64(n)
			avisos.em(*posicao)
		}
		if errors.Is(erroDeLeitura, io.EOF) {
			break
		}
		if erroDeLeitura != nil {
			return false, interrompida(erroDeLeitura)
		}
	}
	if resposta.StatusCode == http.StatusPartialContent && lidos == 0 {
		return false, &app.ErroDownload{Motivo: "O servidor parou de enviar a música"}
	}
	inteiro := resposta.StatusCode == http.StatusOK
	return inteiro || (avisos.total < 0 && lidos < bloco) || (avisos.total >= 0 && *posicao >= avisos.total), nil
}
