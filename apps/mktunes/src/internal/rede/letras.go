// SPDX-License-Identifier: GPL-3.0-or-later

// Pacote rede: os serviços usados sem conta além do YouTube — as letras do
// LRCLIB, as estações do radio-browser.info — e o download de arquivos para a
// pasta de música. Tudo por net/http, com o cliente (e o prazo dele) dado por
// quem monta o daemon.
package rede

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"mktunes/internal/app"
	"mktunes/internal/dominio"
)

// Quem os pedidos dizem ser (o LRCLIB e o radio-browser pedem um nome de app).
const agente = "MkTunes (https://github.com/codexmark/mktunes)"

// corpoMax é o maior corpo JSON aceito dos serviços.
const corpoMax = 8 << 20

// pega faz um GET e devolve o corpo; achou=false para 404.
func pega(ctx context.Context, cliente *http.Client, endereco string) (corpo []byte, achou bool, erro error) {
	pedido, erro := http.NewRequestWithContext(ctx, http.MethodGet, endereco, nil)
	if erro != nil {
		return nil, false, erro
	}
	pedido.Header.Set("User-Agent", agente)
	resposta, erro := cliente.Do(pedido)
	if erro != nil {
		return nil, false, erro
	}
	defer resposta.Body.Close()
	if resposta.StatusCode == http.StatusNotFound {
		return nil, false, nil
	}
	if resposta.StatusCode < 200 || resposta.StatusCode > 299 {
		return nil, false, fmt.Errorf("HTTP %d", resposta.StatusCode)
	}
	corpo, erro = io.ReadAll(io.LimitReader(resposta.Body, corpoMax))
	return corpo, erro == nil, erro
}

// Lrclib é o LRCLIB, o banco aberto de letras: primeiro a busca exata, depois
// uma procura que só aceita letras a até cinco segundos da duração da faixa.
type Lrclib struct {
	HTTP *http.Client
	// Base é "https://lrclib.net/" (os testes trocam).
	Base string
}

type entradaLrclib struct {
	Duration     *float64 `json:"duration"`
	SyncedLyrics *string  `json:"syncedLyrics"`
	PlainLyrics  *string  `json:"plainLyrics"`
}

func (e entradaLrclib) sincronizada() *dominio.LetraDaFaixa {
	if e.SyncedLyrics == nil {
		return nil
	}
	if letra := dominio.LeLRC(*e.SyncedLyrics); len(letra) > 0 {
		return &dominio.LetraDaFaixa{Sincronizada: letra}
	}
	return nil
}

func (e entradaLrclib) simples() *dominio.LetraDaFaixa {
	if e.PlainLyrics == nil || strings.TrimSpace(*e.PlainLyrics) == "" {
		return nil
	}
	return &dominio.LetraDaFaixa{Texto: *e.PlainLyrics}
}

func (l Lrclib) consulta(ctx context.Context, caminho string, parametros url.Values, alvo any) (bool, error) {
	base := l.Base
	if base == "" {
		base = "https://lrclib.net/"
	}
	corpo, achou, erro := pega(ctx, l.HTTP, strings.TrimSuffix(base, "/")+"/"+caminho+"?"+parametros.Encode())
	if erro != nil {
		return false, fmt.Errorf("%w: o LRCLIB não respondeu: %v", app.ErrCatalogo, erro)
	}
	if !achou {
		return false, nil
	}
	if json.Unmarshal(corpo, alvo) != nil {
		return false, fmt.Errorf("%w: o LRCLIB mandou uma resposta que não se lê", app.ErrCatalogo)
	}
	return true, nil
}

// Acha devolve a letra da faixa, sincronizada quando há; nil quando o LRCLIB
// não tem.
func (l Lrclib) Acha(ctx context.Context, f dominio.Faixa) (*dominio.LetraDaFaixa, error) {
	segundos := int64(f.Duracao.Seconds())
	parametros := url.Values{"track_name": {f.Titulo}, "artist_name": {f.Artista}}
	exata := url.Values{"track_name": {f.Titulo}, "artist_name": {f.Artista}}
	if segundos > 0 {
		exata.Set("duration", strconv.FormatInt(segundos, 10))
	}
	var entrada entradaLrclib
	achou, erro := l.consulta(ctx, "api/get", exata, &entrada)
	if erro != nil {
		return nil, erro
	}
	if achou {
		if letra := entrada.sincronizada(); letra != nil {
			return letra, nil
		}
		if letra := entrada.simples(); letra != nil {
			return letra, nil
		}
	}
	var resultados []entradaLrclib
	if _, erro := l.consulta(ctx, "api/search", parametros, &resultados); erro != nil {
		return nil, erro
	}
	var perto []entradaLrclib
	for _, r := range resultados {
		if segundos <= 0 || r.Duration == nil || math.Abs(*r.Duration-float64(segundos)) <= 5 {
			perto = append(perto, r)
		}
	}
	for _, r := range perto {
		if letra := r.sincronizada(); letra != nil {
			return letra, nil
		}
	}
	for _, r := range perto {
		if letra := r.simples(); letra != nil {
			return letra, nil
		}
	}
	return nil, nil
}
