package main

import (
	"fmt"

	"mkpulso/internal/acoes"
	"mkpulso/internal/base"
)

func init() { registrar("prioridade", cmdPrioridade) }

// mkpulso-coletor prioridade <pid> <nice> [--mesmo-assim]
func cmdPrioridade(args []string) int {
	pos, ops, err := opcoesDeAcao(args, []string{"--mesmo-assim"}, nil)
	if err != nil {
		return falhaDeAcao(err)
	}
	if len(pos) != 2 {
		return base.Falha(base.Erro, "uso: mkpulso-coletor prioridade <pid> <nice de -20 a 19> [--mesmo-assim]")
	}
	pid, err := acoes.LeNumero(pos[0])
	if err != nil {
		return falhaDeAcao(fmt.Errorf("o pid %v", err))
	}
	nice, err := acoes.LeNumero(pos[1])
	if err != nil {
		return falhaDeAcao(fmt.Errorf("a prioridade (nice) vai de -20 a 19: %v", err))
	}
	if err := acoes.Prioridade(pid, nice, ops["--mesmo-assim"] != ""); err != nil {
		return falhaDeAcao(err)
	}
	_ = base.Escreve(struct {
		Ok   bool `json:"ok"`
		Pid  int  `json:"pid"`
		Nice int  `json:"nice"`
	}{true, pid, nice})
	return base.Certo
}
