// SPDX-License-Identifier: GPL-3.0-or-later

package innertube

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"mktunes/internal/app"
)

// Perfil é de quem os pedidos dizem vir.
type Perfil struct {
	Nome       string
	ID         string
	Versao     string
	Base       string
	UserAgent  string
	Origem     string
	MaisCampos map[string]string
}

// WebRemix é o YouTube Music da web: navegação, busca, filas.
var WebRemix = Perfil{
	Nome:      "WEB_REMIX",
	ID:        "67",
	Versao:    "1.20260213.01.00",
	Base:      "https://music.youtube.com/youtubei/v1/",
	UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:140.0) Gecko/20100101 Firefox/140.0",
	Origem:    "https://music.youtube.com",
}

// AndroidVR é o app do headset: a resposta de "player" dele traz o volume
// medido de cada faixa sem pedir login.
var AndroidVR = Perfil{
	Nome:       "ANDROID_VR",
	ID:         "28",
	Versao:     "1.65.10",
	Base:       "https://www.youtube.com/youtubei/v1/",
	UserAgent:  "com.google.android.apps.youtube.vr.oculus/1.65.10 (Linux; U; Android 12L) gzip",
	MaisCampos: map[string]string{"androidSdkVersion": "32", "osName": "Android", "osVersion": "12L"},
}

// Visitante é a identidade de convidado que o YouTube entrega; clientes que a
// dividem parecem um visitante só.
type Visitante struct {
	mu    sync.Mutex
	valor string
}

func (v *Visitante) le() string {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.valor
}

func (v *Visitante) guarda(valor string) {
	v.mu.Lock()
	v.valor = valor
	v.mu.Unlock()
}

// respostaMax é o maior corpo aceito: uma página do InnerTube passa pouco de 1 MB.
const respostaMax = 32 << 20

// Cliente faz as chamadas de convidado ao InnerTube.
type Cliente struct {
	HTTP   *http.Client
	Perfil Perfil
	// Base troca o endereço do perfil (os testes apontam para um servidor local).
	Base string
	// Idioma e Regiao vão como hl e gl ("pt", "BR").
	Idioma, Regiao string
	Visitante      *Visitante
}

// Posta chama um endpoint e devolve a resposta como árvore. Qualquer falha
// (rede, HTTP, JSON que não se lê) é o catálogo indisponível.
func (c *Cliente) Posta(ctx context.Context, endpoint string, campos map[string]any) (*No, error) {
	visitante := ""
	if c.Visitante != nil {
		visitante = c.Visitante.le()
	}
	cliente := map[string]any{"clientName": c.Perfil.Nome, "clientVersion": c.Perfil.Versao, "hl": ou(c.Idioma, "en"), "gl": ou(c.Regiao, "US")}
	for chave, valor := range c.Perfil.MaisCampos {
		cliente[chave] = valor
	}
	if visitante != "" {
		cliente["visitorData"] = visitante
	}
	corpo := map[string]any{"context": map[string]any{"client": cliente, "user": map[string]any{"lockedSafetyMode": false}}}
	for chave, valor := range campos {
		corpo[chave] = valor
	}
	bruto, _ := json.Marshal(corpo)
	base := ou(c.Base, c.Perfil.Base)
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	pedido, erro := http.NewRequestWithContext(ctx, http.MethodPost, base+endpoint+"?prettyPrint=false", bytes.NewReader(bruto))
	if erro != nil {
		return nil, fmt.Errorf("%w: %v", app.ErrCatalogo, erro)
	}
	pedido.Header.Set("Content-Type", "application/json")
	pedido.Header.Set("User-Agent", c.Perfil.UserAgent)
	pedido.Header.Set("X-YouTube-Client-Name", c.Perfil.ID)
	pedido.Header.Set("X-YouTube-Client-Version", c.Perfil.Versao)
	if c.Perfil.Origem != "" {
		pedido.Header.Set("Origin", c.Perfil.Origem)
	}
	if visitante != "" {
		pedido.Header.Set("X-Goog-Visitor-Id", visitante)
	}
	resposta, erro := c.HTTP.Do(pedido)
	if erro != nil {
		return nil, fmt.Errorf("%w: o YouTube não respondeu: %v", app.ErrCatalogo, erro)
	}
	defer resposta.Body.Close()
	if resposta.StatusCode < 200 || resposta.StatusCode > 299 {
		return nil, fmt.Errorf("%w: o YouTube respondeu HTTP %d a %s", app.ErrCatalogo, resposta.StatusCode, endpoint)
	}
	no, erro := LeJSON(io.LimitReader(resposta.Body, respostaMax))
	if erro != nil || no.tipo != 'o' {
		return nil, fmt.Errorf("%w: o YouTube Music mandou uma resposta que não se lê", app.ErrCatalogo)
	}
	if novo := no.Obj("responseContext").Str("visitorData"); novo != "" && c.Visitante != nil {
		c.Visitante.guarda(novo)
	}
	return no, nil
}

func ou(valor, reserva string) string {
	if valor == "" {
		return reserva
	}
	return valor
}
