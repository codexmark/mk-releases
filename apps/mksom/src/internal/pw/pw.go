// Package pw lê a saída do `pw-dump` (uma lista JSON com tudo que o PipeWire
// tem) e tira dela só o que o MkSom usa: os nós de saída e de entrada de
// áudio, com volume e mudo, e quem é o padrão. O leitor é tolerante: item que
// não entende, pula; campo que falta, fica desconhecido.
package pw

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"strconv"
	"strings"
)

// No é um nó de áudio do PipeWire.
type No struct {
	ID        int
	Serie     int // object.serial, ou -1
	Nome      string
	Descricao string
	Classe    string
	Volume    int // em %, na escala do wpctl; -1 se o dump não diz
	Mudo      bool
}

// Grafo é o que foi lido de um pw-dump.
type Grafo struct {
	Nos           []No
	SaidaPadrao   string // node.name
	EntradaPadrao string
}

// Porcento converte o volume linear do PipeWire para a escala do wpctl (raiz
// cúbica), em %.
func Porcento(linear float64) int {
	if !(linear > 0) {
		return 0
	}
	return int(math.Round(math.Cbrt(linear) * 100))
}

// texto lê um valor JSON como texto: uma string, ou um número escrito.
func texto(bruto json.RawMessage) string {
	var s string
	if json.Unmarshal(bruto, &s) == nil {
		return s
	}
	var n json.Number
	if json.Unmarshal(bruto, &n) == nil {
		return n.String()
	}
	return ""
}

// inteiro lê um valor JSON como inteiro (número ou texto com algarismos).
func inteiro(bruto json.RawMessage) (int, bool) {
	n, err := strconv.Atoi(strings.TrimSpace(texto(bruto)))
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}

func objeto(bruto json.RawMessage) map[string]json.RawMessage {
	var m map[string]json.RawMessage
	if json.Unmarshal(bruto, &m) != nil {
		return nil
	}
	return m
}

// Le interpreta a saída do pw-dump. Só é erro o que não é uma lista JSON.
func Le(dados []byte) (Grafo, error) {
	var g Grafo
	if len(bytes.TrimSpace(dados)) == 0 {
		return g, errors.New("o pw-dump não disse nada")
	}
	var itens []json.RawMessage
	if err := json.Unmarshal(dados, &itens); err != nil || itens == nil {
		return g, errors.New("não entendi a resposta do pw-dump")
	}
	efetivo := map[string]string{}
	configurado := map[string]string{}
	for _, bruto := range itens {
		item := objeto(bruto)
		if item == nil {
			continue
		}
		tipo := texto(item["type"])
		switch {
		case strings.HasSuffix(tipo, ":Node"):
			if no, ok := leNo(item); ok {
				g.Nos = append(g.Nos, no)
			}
		case strings.HasSuffix(tipo, ":Metadata"):
			leMetadados(item, efetivo, configurado)
		}
	}
	escolhe := func(qual string) string {
		if v := efetivo[qual]; v != "" {
			return v
		}
		return configurado[qual]
	}
	g.SaidaPadrao = escolhe("sink")
	g.EntradaPadrao = escolhe("source")
	return g, nil
}

func leNo(item map[string]json.RawMessage) (No, bool) {
	no := No{Serie: -1, Volume: -1}
	id, ok := inteiro(item["id"])
	if !ok {
		return no, false
	}
	no.ID = id
	info := objeto(item["info"])
	props := objeto(info["props"])
	if props == nil {
		return no, false
	}
	no.Nome = texto(props["node.name"])
	no.Classe = texto(props["media.class"])
	if no.Nome == "" || no.Classe == "" {
		return no, false
	}
	for _, chave := range []string{"node.description", "node.nick"} {
		if no.Descricao == "" {
			no.Descricao = strings.TrimSpace(texto(props[chave]))
		}
	}
	if no.Descricao == "" {
		no.Descricao = no.Nome
	}
	if s, ok := inteiro(props["object.serial"]); ok {
		no.Serie = s
	}
	// info.params.Props é uma lista; o volume está no item que tem
	// channelVolumes (ou, na falta, "volume").
	var lista []json.RawMessage
	_ = json.Unmarshal(objeto(info["params"])["Props"], &lista)
	for _, bruto := range lista {
		p := objeto(bruto)
		if p == nil {
			continue
		}
		var canais []float64
		if json.Unmarshal(p["channelVolumes"], &canais) == nil && len(canais) > 0 {
			soma := 0.0
			for _, c := range canais {
				soma += c
			}
			no.Volume = Porcento(soma / float64(len(canais)))
		} else if no.Volume < 0 {
			var v float64
			if _, tem := p["volume"]; tem && json.Unmarshal(p["volume"], &v) == nil {
				no.Volume = Porcento(v)
			}
		}
		if m, tem := p["mute"]; tem {
			var b bool
			if json.Unmarshal(m, &b) == nil {
				no.Mudo = b
			}
		}
		if _, tem := p["channelVolumes"]; tem {
			break
		}
	}
	return no, true
}

// leMetadados tira do objeto "default" quem é a saída e a entrada padrão. O
// valor é {"name": "..."}; aceita também esse JSON dentro de um texto, ou o
// nome solto.
func leMetadados(item map[string]json.RawMessage, efetivo, configurado map[string]string) {
	if nome := texto(objeto(item["props"])["metadata.name"]); nome != "" && nome != "default" {
		return
	}
	var entradas []json.RawMessage
	if json.Unmarshal(item["metadata"], &entradas) != nil {
		return
	}
	for _, bruto := range entradas {
		e := objeto(bruto)
		if e == nil {
			continue
		}
		chave := texto(e["key"])
		var destino map[string]string
		var qual string
		switch chave {
		case "default.audio.sink":
			destino, qual = efetivo, "sink"
		case "default.audio.source":
			destino, qual = efetivo, "source"
		case "default.configured.audio.sink":
			destino, qual = configurado, "sink"
		case "default.configured.audio.source":
			destino, qual = configurado, "source"
		default:
			continue
		}
		valor := e["value"]
		nome := texto(objeto(valor)["name"])
		if nome == "" {
			if s := strings.TrimSpace(texto(valor)); strings.HasPrefix(s, "{") {
				nome = texto(objeto(json.RawMessage(s))["name"])
			} else {
				nome = s
			}
		}
		if nome != "" {
			destino[qual] = nome
		}
	}
}

func eEntrada(no No) bool {
	return (no.Classe == "Audio/Source" || no.Classe == "Audio/Source/Virtual") && !strings.HasSuffix(no.Nome, ".monitor")
}

// Saidas são os nós Audio/Sink, na ordem do dump.
func (g Grafo) Saidas() []No {
	var out []No
	for _, n := range g.Nos {
		if n.Classe == "Audio/Sink" {
			out = append(out, n)
		}
	}
	return out
}

// Entradas são os nós Audio/Source (e os virtuais), sem os monitores.
func (g Grafo) Entradas() []No {
	var out []No
	for _, n := range g.Nos {
		if eEntrada(n) {
			out = append(out, n)
		}
	}
	return out
}

// Saida procura uma saída pelo node.name.
func (g Grafo) Saida(nome string) (No, bool) {
	if nome == "" {
		return No{}, false
	}
	for _, n := range g.Saidas() {
		if n.Nome == nome {
			return n, true
		}
	}
	return No{}, false
}

// Entrada procura uma entrada pelo id.
func (g Grafo) Entrada(id int) (No, bool) {
	for _, n := range g.Entradas() {
		if n.ID == id {
			return n, true
		}
	}
	return No{}, false
}
