// mkcalc-motor: a metade do MkCalc que faz a conta e guarda o histórico. A
// janela (../qml) só mostra o que sai daqui. O contrato entre as duas está em
// ../CONTRATO.md: mude lá primeiro.
package main

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"strconv"
	"strings"

	"mkcalc/internal/conta"
	"mkcalc/internal/historico"
)

// Códigos de saída do contrato.
const (
	certo    = 0
	falhou   = 1 // uso errado ou disco
	recusada = 2 // a conta não saiu
)

const uso = "uso: mkcalc-motor <calcular [--guardar] expressão | historico | apagar id | limpar | servir>"

func main() {
	os.Exit(rodar(os.Args[1:], os.Stdin, os.Stdout))
}

func rodar(args []string, entrada io.Reader, saida io.Writer) int {
	if len(args) == 0 {
		return escrever(saida, falha(uso), falhou)
	}
	resto := args[1:]
	switch args[0] {
	case "calcular":
		guardar := false
		var partes []string
		for i, a := range resto {
			if a == "--" {
				partes = append(partes, resto[i+1:]...)
				break
			}
			if a == "--guardar" {
				guardar = true
				continue
			}
			// O resto é conta, mesmo começando com "-" (-2 + 3).
			partes = append(partes, a)
		}
		if len(partes) == 0 {
			return escrever(saida, falha(uso), falhou)
		}
		r, codigo := calcular(strings.Join(partes, " "), guardar)
		return escrever(saida, r, codigo)
	case "historico":
		if len(resto) == 0 {
			return escrever(saida, resposta{"itens": historico.Ler()}, certo)
		}
	case "apagar":
		if len(resto) == 1 {
			if id, err := strconv.ParseInt(resto[0], 10, 64); err == nil {
				r, codigo := apagar(id)
				return escrever(saida, r, codigo)
			}
		}
	case "limpar":
		if len(resto) == 0 {
			r, codigo := limpar()
			return escrever(saida, r, codigo)
		}
	case "servir":
		if len(resto) == 0 {
			return servir(entrada, saida)
		}
	}
	return escrever(saida, falha(uso), falhou)
}

// Uma resposta: os campos do contrato.
type resposta map[string]any

func falha(frase string) resposta { return resposta{"erro": frase} }

// escrever solta a resposta numa linha e devolve o código de saída.
func escrever(saida io.Writer, r resposta, codigo int) int {
	e := json.NewEncoder(saida)
	e.SetEscapeHTML(false)
	if e.Encode(r) != nil && codigo == certo {
		return falhou
	}
	return codigo
}

func calcular(expressao string, guardar bool) (resposta, int) {
	r, err := conta.Calcular(expressao)
	if err != nil {
		var e *conta.Erro
		if errors.As(err, &e) {
			return resposta{"erro": e.Frase, "incompleta": e.Incompleta}, recusada
		}
		return falha(err.Error()), recusada
	}
	resp := resposta{"expressao": r.Expressao, "resultado": r.Resultado, "cru": r.Cru, "aproximado": r.Aproximado}
	if guardar {
		itens, err := historico.Guardar(r.Expressao, r.Resultado, r.Cru, r.Aproximado)
		if err != nil {
			resp["aviso"] = "não deu para guardar no histórico: " + err.Error()
		} else {
			resp["itens"] = itens
		}
	}
	return resp, certo
}

func apagar(id int64) (resposta, int) {
	itens, err := historico.Apagar(id)
	if err != nil {
		return falha(doHistorico(err)), falhou
	}
	return resposta{"itens": itens}, certo
}

func limpar() (resposta, int) {
	itens, err := historico.Limpar()
	if err != nil {
		return falha(doHistorico(err)), falhou
	}
	return resposta{"itens": itens}, certo
}

func doHistorico(err error) string {
	if errors.Is(err, historico.ErroSemItem) {
		return err.Error()
	}
	return "não deu para gravar o histórico: " + err.Error()
}
