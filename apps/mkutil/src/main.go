// mkutil-motor: a metade do MkUtil (MkInstalar e MkComprimir) que olha os
// arquivos e faz o trabalho. As janelas (../qml) só mostram o que sai daqui.
// O contrato entre as duas está em ../CONTRATO.md: mude lá primeiro.
//
// Cada subcomando mora no seu arquivo e se registra em init() (veja
// registrar). Este arquivo só escolhe quem roda.
package main

import (
	"os"
	"sort"
	"strings"

	"mkutil/internal/base"
)

// Um subcomando: recebe os argumentos depois do nome e devolve o código de
// saída (base.Certo, base.Erro ou base.Recusado). Escreve o JSON ele mesmo.
type comando func(args []string) int

var comandos = map[string]comando{}

// registrar põe um subcomando na mesa. Chamado dos init() de cada arquivo.
func registrar(nome string, c comando) {
	comandos[nome] = c
}

func main() {
	os.Exit(rodar(os.Args[1:]))
}

func rodar(args []string) int {
	if len(args) == 0 {
		return uso()
	}
	c, ok := comandos[args[0]]
	if !ok {
		return uso()
	}
	return c(args[1:])
}

func uso() int {
	nomes := make([]string, 0, len(comandos))
	for n := range comandos {
		nomes = append(nomes, n)
	}
	sort.Strings(nomes)
	return base.Falha(base.Erro, "uso: mkutil-motor <%s>", strings.Join(nomes, " | "))
}
