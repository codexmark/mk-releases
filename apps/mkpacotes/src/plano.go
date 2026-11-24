package main

import (
	"strings"

	"mkpacotes/internal/acoes"
	"mkpacotes/internal/base"
)

func init() {
	registrar("plano", cmdPlano)
	registrar("agir", cmdAgir)
}

const usoDeAcao = "<" + "instalar|remover|orfaos|cache|atualizar" + "> [pacotes…] [--mesmo-assim]"

// mkpacotes-coletor plano <acao> [pacotes…] [--mesmo-assim]
func cmdPlano(args []string) int {
	pos, ops, err := opcoes(args, []string{"--mesmo-assim"}, nil)
	if err != nil {
		return falha(err)
	}
	if len(pos) == 0 {
		return base.Falha(base.Erro, "uso: mkpacotes-coletor plano %s", usoDeAcao)
	}
	p, err := acoes.Planeja(pos[0], pos[1:], ops["--mesmo-assim"] != "")
	if err != nil {
		return falha(err)
	}
	_ = base.Escreve(p)
	return base.Certo
}

// mkpacotes-coletor agir <acao> [pacotes…] [--mesmo-assim]
//
// Transmite {"linha": …} a cada linha dos comandos e termina com {"ok": true,
// …} ou {"erro": …}.
func cmdAgir(args []string) int {
	pos, ops, err := opcoes(args, []string{"--mesmo-assim"}, nil)
	if err != nil {
		return falha(err)
	}
	if len(pos) == 0 {
		return base.Falha(base.Erro, "uso: mkpacotes-coletor agir %s", usoDeAcao)
	}
	emite := func(linha string) {
		_ = base.Escreve(map[string]string{"linha": strings.ToValidUTF8(linha, "?")})
	}
	p, err := acoes.Age(pos[0], pos[1:], ops["--mesmo-assim"] != "", emite)
	if err != nil {
		return falha(err)
	}
	_ = base.Escreve(struct {
		Ok      bool     `json:"ok"`
		Acao    string   `json:"acao"`
		Pacotes []string `json:"pacotes"`
	}{true, p.Acao, p.Pedidos})
	return base.Certo
}
