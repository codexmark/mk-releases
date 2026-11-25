// mksom-motor: a metade do MkSom (o equalizador e o teste de microfone do Mk)
// que fala com o PipeWire: escreve o arquivo do filter-chain e a unidade do
// systemd, liga e desliga, aplica os ganhos ao vivo, lê o pw-dump e mede o
// microfone. A janela (../qml) só mostra o que sai daqui. O contrato entre as
// duas está em ../CONTRATO.md: mude lá primeiro.
//
// Cada subcomando mora no seu arquivo e se registra em init() (veja
// registrar). Este arquivo só escolhe quem roda.
package main

import (
	"io"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"

	"mksom/internal/base"
)

// Um subcomando: recebe os argumentos depois do nome e devolve o código de
// saída (base.Certo, base.Erro ou base.Recusado). Escreve o JSON ele mesmo.
type comando func(args []string) int

var comandos = map[string]comando{}

// entrada é a entrada padrão do motor: o pedido do `eq ajustar`, e o fio que
// a janela segura enquanto um fluxo roda (fechou, o fluxo para). Os testes
// trocam.
var entrada io.Reader = os.Stdin

// registrar põe um subcomando na mesa. Chamado dos init() de cada arquivo.
func registrar(nome string, c comando) {
	comandos[nome] = c
}

func main() {
	// Se a janela morre no meio de um fluxo, escrever na saída dá erro em vez
	// de matar o motor: ele ainda tem de parar o pw-record e apagar a gravação.
	signal.Ignore(syscall.SIGPIPE)
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
	return base.Falha(base.Erro, "uso: mksom-motor <%s>", strings.Join(nomes, " | "))
}
