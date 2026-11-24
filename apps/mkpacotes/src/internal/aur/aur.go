// Package aur fala com o AUR pela API RPC dele (v5), por HTTPS, só com a
// biblioteca padrão: a busca e a ficha de pacotes. O endereço vem de
// base.AUR() (MKPACOTES_AUR), para os testes apontarem um servidor de
// mentira. Todo erro vira frase: sem rede, o resto do app continua.
package aur

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"mkpacotes/internal/alpm"
	"mkpacotes/internal/base"
)

// Prazo é quanto esperar por uma resposta do AUR.
var Prazo = 12 * time.Second

// Quantos nomes cabem num pedido de ficha e o tamanho máximo de uma resposta.
const (
	lote       = 100
	maxDeBytes = 16 << 20
)

// Pacote é um pacote do AUR.
type Pacote struct {
	Nome, Versao, Descricao, URL string
	Votos                        int
	Popularidade                 float64
	Mantenedor                   string // "" é pacote sem mantenedor
	DesatualizadoDesde           *int64 // nil: não está marcado
	ModificadoEm                 *int64
	Depende, Construcao          []string
	Opcionais, Licencas          []string
	Fornece, Conflita            []string
}

type resposta struct {
	Type    string `json:"type"`
	Error   string `json:"error"`
	Results []struct {
		Name, Version           string
		Description, URL        *string
		NumVotes                int
		Popularity              float64
		Maintainer              *string
		OutOfDate, LastModified *int64
		Depends, MakeDepends    []string
		OptDepends, License     []string
		Provides, Conflicts     []string
	} `json:"results"`
}

func texto(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func pede(endereco string) ([]Pacote, error) {
	cliente := &http.Client{Timeout: Prazo}
	req, err := http.NewRequest(http.MethodGet, endereco, nil)
	if err != nil {
		return nil, errors.New("o endereço do AUR não é válido")
	}
	req.Header.Set("User-Agent", "mkpacotes")
	r, err := cliente.Do(req)
	if err != nil {
		return nil, errors.New("não consegui falar com o AUR (sem rede?)")
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("o AUR respondeu com o código %d", r.StatusCode)
	}
	corpo, err := io.ReadAll(io.LimitReader(r.Body, maxDeBytes))
	if err != nil {
		return nil, errors.New("não consegui falar com o AUR (sem rede?)")
	}
	var resp resposta
	if err := json.Unmarshal(corpo, &resp); err != nil {
		return nil, errors.New("o AUR respondeu algo que não entendi")
	}
	if resp.Type == "error" {
		return nil, fmt.Errorf("o AUR recusou a busca: %s", resp.Error)
	}
	lista := make([]Pacote, 0, len(resp.Results))
	for _, x := range resp.Results {
		lista = append(lista, Pacote{Nome: x.Name, Versao: x.Version, Descricao: texto(x.Description), URL: texto(x.URL),
			Votos: x.NumVotes, Popularidade: x.Popularity, Mantenedor: texto(x.Maintainer), DesatualizadoDesde: x.OutOfDate,
			ModificadoEm: x.LastModified, Depende: x.Depends, Construcao: x.MakeDepends, Opcionais: x.OptDepends,
			Licencas: x.License, Fornece: x.Provides, Conflita: x.Conflicts})
	}
	return lista, nil
}

// Busca procura no AUR pelo nome e pela descrição.
func Busca(texto string) ([]Pacote, error) {
	return pede(base.AUR() + "/rpc/v5/search/" + url.PathEscape(texto) + "?by=name-desc")
}

// Info devolve a ficha dos pacotes pedidos (os que o AUR não tem não vêm).
// Os nomes são validados antes de sair da máquina.
func Info(nomes []string) ([]Pacote, error) {
	for _, n := range nomes {
		if err := alpm.ValidaNome(n); err != nil {
			return nil, err
		}
	}
	var todos []Pacote
	for i := 0; i < len(nomes); i += lote {
		consulta := url.Values{}
		for _, n := range nomes[i:min(i+lote, len(nomes))] {
			consulta.Add("arg[]", n)
		}
		lista, err := pede(base.AUR() + "/rpc/v5/info?" + consulta.Encode())
		if err != nil {
			return nil, err
		}
		todos = append(todos, lista...)
	}
	return todos, nil
}
