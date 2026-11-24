package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"

	"mkcalc/internal/historico"
)

// O tamanho máximo de uma linha de pedido.
const limiteDaLinha = 64 << 10

// Um pedido do modo servidor (uma linha JSON).
type pedido struct {
	ID      json.RawMessage `json:"id"`
	Op      string          `json:"op"`
	Expr    string          `json:"expr"`
	Guardar bool            `json:"guardar"`
	Item    *int64          `json:"item"`
}

// servir lê um pedido por linha e responde uma linha por pedido, com o mesmo
// id, até a entrada fechar. Pedido torto vira uma resposta de erro; o
// servidor continua.
func servir(entrada io.Reader, saida io.Writer) int {
	linhas := bufio.NewScanner(entrada)
	linhas.Buffer(make([]byte, 0, 4096), limiteDaLinha)
	for linhas.Scan() {
		linha := bytes.TrimSpace(linhas.Bytes())
		if len(linha) == 0 {
			continue
		}
		r := atender(linha)
		if escrever(saida, r, certo) != certo {
			return falhou // ninguém mais lê: a janela fechou
		}
	}
	if linhas.Err() != nil {
		r := falha("pedido grande demais")
		r["id"] = nil
		return escrever(saida, r, falhou)
	}
	return certo
}

func atender(linha []byte) resposta {
	var p pedido
	if err := json.Unmarshal(linha, &p); err != nil {
		r := falha("pedido que não é JSON")
		r["id"] = nil
		return r
	}
	var r resposta
	switch p.Op {
	case "calcular":
		r, _ = calcular(p.Expr, p.Guardar)
	case "historico":
		r = resposta{"itens": historico.Ler()}
	case "apagar":
		if p.Item == nil {
			r = falha("o pedido apagar precisa de \"item\"")
		} else {
			r, _ = apagar(*p.Item)
		}
	case "limpar":
		r, _ = limpar()
	default:
		r = falha("não conheço o pedido «" + p.Op + "»")
	}
	r["id"] = idDe(p.ID)
	return r
}

// idDe devolve o id do pedido como veio, se é um número ou um texto; o resto
// (ausente, objeto, lista) vira null.
func idDe(cru json.RawMessage) any {
	var v any
	d := json.NewDecoder(bytes.NewReader(cru))
	d.UseNumber()
	if len(cru) == 0 || d.Decode(&v) != nil {
		return nil
	}
	switch v.(type) {
	case json.Number, string:
		return v
	}
	return nil
}
