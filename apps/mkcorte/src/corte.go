package main

import (
	"errors"
	"io"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"mkcorte/internal/base"
	"mkcorte/internal/midia"
)

func init() {
	registrar("sondar", cmdSondar)
	registrar("miniaturas", cmdMiniaturas)
	registrar("plano", cmdPlano)
	registrar("exportar", cmdExportar)
	registrar("quadro", cmdQuadro)
}

// mkcorte-motor sondar <arquivo> [--quadros-chave]
func cmdSondar(args []string) int {
	pos, ops, err := opcoes(args, nil, []string{"--quadros-chave"})
	if err != nil {
		return base.Falha(base.Erro, "%s", err.Error())
	}
	if len(pos) != 1 {
		return base.Falha(base.Erro, "uso: mkcorte-motor sondar <arquivo> [--quadros-chave]")
	}
	f, err := midia.Sonda(pos[0], ops["--quadros-chave"] != "")
	if err != nil {
		return base.FalhaDe(err)
	}
	_ = base.Escreve(f)
	return base.Certo
}

// mkcorte-motor miniaturas <arquivo> [--quantas N] [--altura N]
func cmdMiniaturas(args []string) int {
	pos, ops, err := opcoes(args, []string{"--quantas", "--altura"}, nil)
	if err != nil {
		return base.Falha(base.Erro, "%s", err.Error())
	}
	if len(pos) != 1 {
		return base.Falha(base.Erro, "uso: mkcorte-motor miniaturas <arquivo> [--quantas N] [--altura N]")
	}
	quantas, altura := 0, 0
	for opcao, onde := range map[string]*int{"--quantas": &quantas, "--altura": &altura} {
		if v, veio := ops[opcao]; veio {
			n, err := strconv.Atoi(v)
			if err != nil || n <= 0 {
				return base.Falha(base.Erro, "%s pede um número inteiro maior que zero, e veio %q", opcao, v)
			}
			*onde = n
		}
	}
	t, err := midia.Miniaturas(pos[0], quantas, altura)
	if err != nil {
		return base.FalhaDe(err)
	}
	_ = base.Escreve(t)
	return base.Certo
}

// De onde a edição vem; os testes trocam.
var entrada io.Reader = os.Stdin

// edicaoDe lê a edição de um subcomando: do arquivo dado ("-" é o stdin) ou
// do texto de --json. Devolve também o que sobrou dos posicionais (a saída).
func edicaoDe(nome string, args []string) (*midia.Edicao, string, int) {
	uso := "uso: mkcorte-motor " + nome + " (<edicao.json> | - | --json <texto>) [saida]"
	pos, ops, err := opcoes(args, []string{"--json"}, nil)
	if err != nil {
		return nil, "", base.Falha(base.Erro, "%s", err.Error())
	}
	var texto []byte
	if json, veio := ops["--json"]; veio {
		texto = []byte(json)
	} else {
		if len(pos) == 0 {
			return nil, "", base.Falha(base.Erro, "%s", uso)
		}
		if pos[0] == "-" {
			texto, err = io.ReadAll(io.LimitReader(entrada, 4<<20))
		} else {
			texto, err = os.ReadFile(pos[0])
		}
		if err != nil {
			return nil, "", base.Falha(base.Erro, "não consegui ler a edição (%s)", base.Til(pos[0]))
		}
		pos = pos[1:]
	}
	if len(pos) > 1 {
		return nil, "", base.Falha(base.Erro, "%s", uso)
	}
	e, err := midia.LeEdicao(texto)
	if err != nil {
		return nil, "", base.FalhaDe(err)
	}
	saida := ""
	if len(pos) == 1 {
		saida = pos[0]
	}
	return e, saida, base.Certo
}

// mkcorte-motor plano (<edicao.json> | - | --json <texto>) [saida]
func cmdPlano(args []string) int {
	e, saida, codigo := edicaoDe("plano", args)
	if e == nil {
		return codigo
	}
	p, err := midia.Planejar(e, saida)
	if err != nil {
		return base.FalhaDe(err)
	}
	_ = base.Escreve(p)
	return base.Certo
}

// mkcorte-motor exportar (<edicao.json> | - | --json <texto>) [saida]
func cmdExportar(args []string) int {
	e, saida, codigo := edicaoDe("exportar", args)
	if e == nil {
		return codigo
	}
	// Cancelar é mandar SIGTERM (a janela faz isso ao parar o processo).
	cancela := make(chan struct{})
	sinais := make(chan os.Signal, 1)
	signal.Notify(sinais, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	signal.Ignore(syscall.SIGPIPE) // a janela fechou: a escrita falha e isso também cancela
	go func() {
		<-sinais
		close(cancela)
	}()
	defer signal.Stop(sinais)

	fim, err := midia.Exporta(e, saida, base.Escreve, cancela)
	if errors.Is(err, midia.ErrCancelado) {
		_ = base.Escreve(map[string]string{"fase": "cancelado"})
		return base.Certo
	}
	if err != nil {
		return base.FalhaDe(err)
	}
	_ = base.Escreve(fim)
	return base.Certo
}

// mkcorte-motor quadro <arquivo> <tempo> [saida]
func cmdQuadro(args []string) int {
	pos, _, err := opcoes(args, nil, nil)
	if err != nil {
		return base.Falha(base.Erro, "%s", err.Error())
	}
	if len(pos) < 2 || len(pos) > 3 {
		return base.Falha(base.Erro, "uso: mkcorte-motor quadro <arquivo> <tempo em segundos> [saida]")
	}
	tempo, err := strconv.ParseFloat(pos[1], 64)
	if err != nil {
		return base.Falha(base.Erro, "o tempo do quadro tem de ser um número de segundos, e veio %q", pos[1])
	}
	saida := ""
	if len(pos) == 3 {
		saida = pos[2]
	}
	foto, err := midia.Quadro(pos[0], tempo, saida)
	if err != nil {
		return base.FalhaDe(err)
	}
	_ = base.Escreve(foto)
	return base.Certo
}
