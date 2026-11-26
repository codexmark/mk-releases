// SPDX-License-Identifier: GPL-3.0-or-later

// Pacote innertube: o cliente próprio da API do YouTube (InnerTube), como os
// apps dele a chamam, sem conta: o Início, as buscas, as sugestões, as páginas
// de álbum, playlist e artista, as filas (rádio) e o volume medido de cada
// faixa. Porte do adapter/youtube/innertube do MkTunes em Kotlin, com a busca
// de músicas acrescentada (lá ela ia pelo NewPipe).
package innertube

import (
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"
)

// No é um valor JSON que lembra a ordem das chaves: os leitores procuram "o
// primeiro X do documento", e um mapa do Go perderia essa ordem. Todos os
// métodos aceitam um nó nulo, para descer pelos renderizadores aninhados do
// InnerTube sem conferir cada passo.
type No struct {
	tipo   byte // 'o' objeto, 'a' lista, 's' texto, 'n' número, 'b' booleano, 'z' nulo
	chaves []string
	filhos []*No
	texto  string
}

// LeJSON monta a árvore de um documento JSON.
func LeJSON(r io.Reader) (*No, error) {
	leitor := json.NewDecoder(r)
	leitor.UseNumber()
	no, erro := leNo(leitor)
	if erro != nil {
		return nil, erro
	}
	if _, erro := leitor.Token(); erro != io.EOF {
		return nil, errors.New("sobra texto depois do JSON")
	}
	return no, nil
}

func leNo(leitor *json.Decoder) (*No, error) {
	token, erro := leitor.Token()
	if erro != nil {
		return nil, erro
	}
	switch valor := token.(type) {
	case json.Delim:
		switch valor {
		case '{':
			no := &No{tipo: 'o'}
			for leitor.More() {
				chave, erro := leitor.Token()
				if erro != nil {
					return nil, erro
				}
				filho, erro := leNo(leitor)
				if erro != nil {
					return nil, erro
				}
				no.chaves = append(no.chaves, chave.(string))
				no.filhos = append(no.filhos, filho)
			}
			_, erro := leitor.Token()
			return no, erro
		case '[':
			no := &No{tipo: 'a'}
			for leitor.More() {
				filho, erro := leNo(leitor)
				if erro != nil {
					return nil, erro
				}
				no.filhos = append(no.filhos, filho)
			}
			_, erro := leitor.Token()
			return no, erro
		}
		return nil, errors.New("JSON fora de ordem")
	case string:
		return &No{tipo: 's', texto: valor}, nil
	case json.Number:
		return &No{tipo: 'n', texto: valor.String()}, nil
	case bool:
		return &No{tipo: 'b', texto: strconv.FormatBool(valor)}, nil
	}
	return &No{tipo: 'z'}, nil
}

func (n *No) filho(chave string) *No {
	if n == nil || n.tipo != 'o' {
		return nil
	}
	for i, c := range n.chaves {
		if c == chave {
			return n.filhos[i]
		}
	}
	return nil
}

// Obj é o objeto guardado na chave, ou nulo.
func (n *No) Obj(chave string) *No {
	if f := n.filho(chave); f != nil && f.tipo == 'o' {
		return f
	}
	return nil
}

// Lista são os itens da lista guardada na chave.
func (n *No) Lista(chave string) []*No {
	if f := n.filho(chave); f != nil && f.tipo == 'a' {
		return f.filhos
	}
	return nil
}

// Str é o texto guardado na chave ("" quando não é um texto).
func (n *No) Str(chave string) string { return n.filho(chave).Texto() }

// Texto é o conteúdo de um nó de texto.
func (n *No) Texto() string {
	if n == nil || n.tipo != 's' {
		return ""
	}
	return n.texto
}

// Int lê um número inteiro (ou um texto que é um), 0 se não for.
func (n *No) Int(chave string) int {
	f := n.filho(chave)
	if f == nil || (f.tipo != 'n' && f.tipo != 's') {
		return 0
	}
	valor, _ := strconv.Atoi(f.texto)
	return valor
}

// Num lê um número com vírgula.
func (n *No) Num(chave string) (float64, bool) {
	f := n.filho(chave)
	if f == nil || f.tipo != 'n' {
		return 0, false
	}
	valor, erro := strconv.ParseFloat(f.texto, 64)
	return valor, erro == nil
}

// Todos são todos os valores guardados sob a chave, em qualquer profundidade,
// na ordem do documento.
func (n *No) Todos(chave string) []*No {
	var achados []*No
	n.anda(chave, func(valor *No) bool {
		achados = append(achados, valor)
		return true
	})
	return achados
}

// Primeiro é o primeiro valor sob a chave, na ordem do documento.
func (n *No) Primeiro(chave string) *No {
	var achado *No
	n.anda(chave, func(valor *No) bool {
		achado = valor
		return false
	})
	return achado
}

func (n *No) anda(chave string, visita func(*No) bool) bool {
	if n == nil {
		return true
	}
	for i, filho := range n.filhos {
		if n.tipo == 'o' && n.chaves[i] == chave && !visita(filho) {
			return false
		}
		if !filho.anda(chave, visita) {
			return false
		}
	}
	return true
}

// Runs é o texto corrido de um bloco {runs:[{text}]}.
func (n *No) Runs() string {
	var texto strings.Builder
	for _, run := range n.Lista("runs") {
		texto.WriteString(run.Str("text"))
	}
	return texto.String()
}

// trechos divide os runs nos pedaços separados por " • ".
func (n *No) trechos() []string {
	var pedacos []string
	var atual strings.Builder
	for _, run := range n.Lista("runs") {
		if texto := run.Str("text"); strings.TrimSpace(texto) == "•" {
			pedacos = append(pedacos, strings.TrimSpace(atual.String()))
			atual.Reset()
		} else {
			atual.WriteString(texto)
		}
	}
	return append(pedacos, strings.TrimSpace(atual.String()))
}

// RunsIniciais são os runs antes do primeiro " • ": os artistas de um subtítulo.
func (n *No) RunsIniciais() string { return n.trechos()[0] }
