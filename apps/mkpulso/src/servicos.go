package main

import (
	"mkpulso/internal/base"
	"mkpulso/internal/servicos"
)

func init() { registrar("servicos", cmdServicos) }

// mkpulso-coletor servicos [--usuario|--sistema]
func cmdServicos(args []string) int {
	pos, ops, err := opcoesDeAcao(args, []string{"--usuario", "--sistema"}, nil)
	if err != nil {
		return falhaDeAcao(err)
	}
	if len(pos) != 0 {
		return base.Falha(base.Erro, "uso: mkpulso-coletor servicos [--usuario|--sistema]")
	}
	usuario, sistema := ops["--usuario"] != "", ops["--sistema"] != ""
	if !usuario && !sistema {
		usuario, sistema = true, true // sem opção, os dois
	}
	lista, err := servicos.Lista(usuario, sistema)
	if err != nil {
		return falhaDeAcao(err)
	}
	_ = base.Escreve(struct {
		Servicos []servicos.Servico `json:"servicos"`
	}{lista})
	return base.Certo
}
