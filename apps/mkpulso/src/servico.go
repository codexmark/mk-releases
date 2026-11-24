package main

import (
	"strings"

	"mkpulso/internal/base"
	"mkpulso/internal/servicos"
)

func init() { registrar("servico", cmdServico) }

// mkpulso-coletor servico <acao> <unidade> [--usuario] [--mesmo-assim]
func cmdServico(args []string) int {
	pos, ops, err := opcoesDeAcao(args, []string{"--usuario", "--mesmo-assim"}, nil)
	if err != nil {
		return falhaDeAcao(err)
	}
	if len(pos) != 2 {
		return base.Falha(base.Erro, "uso: mkpulso-coletor servico <%s> <unidade> [--usuario] [--mesmo-assim]", strings.Join(servicos.Acoes(), "|"))
	}
	acao, unidade := pos[0], pos[1]
	if err := servicos.Age(acao, unidade, ops["--usuario"] != "", ops["--mesmo-assim"] != ""); err != nil {
		return falhaDeAcao(err)
	}
	_ = base.Escreve(struct {
		Ok      bool   `json:"ok"`
		Unidade string `json:"unidade"`
		Acao    string `json:"acao"`
	}{true, unidade, acao})
	return base.Certo
}
