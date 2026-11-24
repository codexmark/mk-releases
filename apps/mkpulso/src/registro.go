package main

import (
	"fmt"

	"mkpulso/internal/acoes"
	"mkpulso/internal/base"
	"mkpulso/internal/servicos"
)

func init() { registrar("registro", cmdRegistro) }

// mkpulso-coletor registro <unidade> [--usuario] [--linhas N]
func cmdRegistro(args []string) int {
	pos, ops, err := opcoesDeAcao(args, []string{"--usuario"}, []string{"--linhas"})
	if err != nil {
		return falhaDeAcao(err)
	}
	if len(pos) != 1 {
		return base.Falha(base.Erro, "uso: mkpulso-coletor registro <unidade> [--usuario] [--linhas N]")
	}
	n := servicos.PadraoDeLinhas
	if texto, tem := ops["--linhas"]; tem {
		if n, err = acoes.LeNumero(texto); err != nil {
			return falhaDeAcao(fmt.Errorf("em --linhas, %v", err))
		}
	}
	linhas, err := servicos.Registro(pos[0], ops["--usuario"] != "", n)
	if err != nil {
		return falhaDeAcao(err)
	}
	_ = base.Escreve(struct {
		Linhas []servicos.Linha `json:"linhas"`
	}{linhas})
	return base.Certo
}
