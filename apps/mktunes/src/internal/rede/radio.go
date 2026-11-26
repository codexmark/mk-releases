// SPDX-License-Identifier: GPL-3.0-or-later

package rede

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"mktunes/internal/app"
	"mktunes/internal/dominio"
)

// Espelhos são os servidores do radio-browser.info, tentados em ordem.
var Espelhos = []string{
	"https://de1.api.radio-browser.info/",
	"https://nl1.api.radio-browser.info/",
	"https://at1.api.radio-browser.info/",
}

// RadioBrowser é o diretório de rádios da internet. Só oferece streams HTTPS.
type RadioBrowser struct {
	HTTP *http.Client
	// Servidores troca os espelhos (os testes apontam para servidores locais).
	Servidores []string
	// Pais é o código de duas letras cujas estações vêm primeiro; vazio lista
	// o mundo todo.
	Pais string
}

type estacao struct {
	UUID        string `json:"stationuuid"`
	Name        string `json:"name"`
	Stream      string `json:"url_resolved"`
	Favicon     string `json:"favicon"`
	Tags        string `json:"tags"`
	Country     string `json:"country"`
	CountryCode string `json:"countrycode"`
}

// paises em português, pelos códigos mais comuns no diretório; para os outros
// vale o nome que o diretório escreveu.
var paises = map[string]string{
	"AR": "Argentina", "AT": "Áustria", "AU": "Austrália", "BE": "Bélgica", "BO": "Bolívia", "BR": "Brasil",
	"CA": "Canadá", "CH": "Suíça", "CL": "Chile", "CN": "China", "CO": "Colômbia", "CR": "Costa Rica",
	"CU": "Cuba", "CZ": "Tchéquia", "DE": "Alemanha", "DK": "Dinamarca", "DO": "República Dominicana",
	"EC": "Equador", "EG": "Egito", "ES": "Espanha", "FI": "Finlândia", "FR": "França", "GB": "Reino Unido",
	"GR": "Grécia", "HR": "Croácia", "HU": "Hungria", "IE": "Irlanda", "IL": "Israel", "IN": "Índia",
	"IT": "Itália", "JP": "Japão", "KR": "Coreia do Sul", "MX": "México", "NL": "Países Baixos",
	"NO": "Noruega", "NZ": "Nova Zelândia", "PE": "Peru", "PL": "Polônia", "PT": "Portugal", "PY": "Paraguai",
	"RO": "Romênia", "RS": "Sérvia", "RU": "Rússia", "SE": "Suécia", "TR": "Turquia", "UA": "Ucrânia",
	"US": "Estados Unidos", "UY": "Uruguai", "VE": "Venezuela", "ZA": "África do Sul", "AO": "Angola",
	"MZ": "Moçambique", "CV": "Cabo Verde",
}

func soHTTPS(endereco string) string {
	u, erro := url.Parse(strings.TrimSpace(endereco))
	if erro != nil || u.Scheme != "https" || u.Host == "" {
		return ""
	}
	return u.String()
}

func (e estacao) faixa() (dominio.Faixa, bool) {
	stream := soHTTPS(e.Stream)
	if stream == "" || e.UUID == "" {
		return dominio.Faixa{}, false
	}
	var descricao []string
	for _, tag := range strings.Split(e.Tags, ",") {
		if tag = strings.TrimSpace(tag); tag != "" && len(descricao) < 3 {
			descricao = append(descricao, tag)
		}
	}
	pais := paises[strings.ToUpper(strings.TrimSpace(e.CountryCode))]
	if pais == "" {
		pais = strings.TrimSpace(e.Country)
	}
	if pais != "" {
		descricao = append(descricao, pais)
	}
	return dominio.Faixa{
		ID: "radio:" + e.UUID, Provedor: dominio.Radio,
		Titulo: strings.TrimSpace(e.Name), Artista: strings.Join(descricao, " · "),
		Capa: soHTTPS(e.Favicon), Stream: stream,
	}, true
}

// Topo lista as estações mais votadas do país do ouvinte, ou do mundo quando o
// país não tem nenhuma.
func (r RadioBrowser) Topo(ctx context.Context, limite int) ([]dominio.Faixa, error) {
	if len(r.Pais) == 2 {
		daqui, erro := r.consulta(ctx, "json/stations/search", url.Values{
			"countrycode": {r.Pais}, "order": {"votes"}, "reverse": {"true"}, "hidebroken": {"true"},
			"offset": {"0"}, "limit": {strconv.Itoa(limite)},
		})
		if erro != nil {
			return nil, erro
		}
		if len(daqui) > 0 {
			return daqui, nil
		}
	}
	return r.consulta(ctx, "json/stations/topvote", url.Values{"offset": {"0"}, "limit": {strconv.Itoa(limite)}, "hidebroken": {"true"}})
}

// Busca procura estações pelo nome, as mais votadas primeiro.
func (r RadioBrowser) Busca(ctx context.Context, nome string, limite int) ([]dominio.Faixa, error) {
	return r.consulta(ctx, "json/stations/search", url.Values{
		"name": {nome}, "order": {"votes"}, "reverse": {"true"}, "hidebroken": {"true"},
		"offset": {"0"}, "limit": {strconv.Itoa(limite)},
	})
}

func (r RadioBrowser) consulta(ctx context.Context, caminho string, parametros url.Values) ([]dominio.Faixa, error) {
	servidores := r.Servidores
	if len(servidores) == 0 {
		servidores = Espelhos
	}
	var ultimo error
	for _, servidor := range servidores {
		corpo, achou, erro := pega(ctx, r.HTTP, strings.TrimSuffix(servidor, "/")+"/"+caminho+"?"+parametros.Encode())
		if erro == nil && !achou {
			erro = errors.New("HTTP 404")
		}
		var estacoes []estacao
		if erro == nil {
			erro = json.Unmarshal(corpo, &estacoes)
		}
		if erro != nil {
			ultimo = erro
			if ctx.Err() != nil {
				break
			}
			continue
		}
		faixas := []dominio.Faixa{}
		for _, e := range estacoes {
			if f, ok := e.faixa(); ok {
				faixas = append(faixas, f)
			}
		}
		return faixas, nil
	}
	return nil, fmt.Errorf("%w: nenhum espelho do radio-browser respondeu: %v", app.ErrCatalogo, ultimo)
}
