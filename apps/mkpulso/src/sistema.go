// O subcomando `sistema`: a ficha da máquina, uma vez só (o formato está em
// ../CONTRATO.md; quem lê cada pedaço é internal/maquina).
package main

import (
	"mkpulso/internal/base"
	"mkpulso/internal/maquina"
)

func init() {
	registrar("sistema", sistema)
}

func sistema(args []string) int {
	if len(args) > 0 {
		return base.Falha(base.Erro, "o sistema não tem opções; uso: mkpulso-coletor sistema")
	}
	if err := base.Escreve(maquina.Ler()); err != nil {
		return base.Erro
	}
	return base.Certo
}
