package main

import (
	"strconv"

	"mkpacotes/internal/base"
	"mkpacotes/internal/historico"
)

func init() { registrar("historico", cmdHistorico) }

// mkpacotes-coletor historico [--linhas N] [--busca texto]
func cmdHistorico(args []string) int {
	pos, ops, err := opcoes(args, nil, []string{"--linhas", "--busca"})
	if err != nil {
		return falha(err)
	}
	if len(pos) != 0 {
		return base.Falha(base.Erro, "uso: mkpacotes-coletor historico [--linhas N] [--busca texto]")
	}
	limite := 300
	if v, dado := ops["--linhas"]; dado {
		limite, err = strconv.Atoi(v)
		if err != nil || limite < 1 || limite > 5000 {
			return base.Falha(base.Erro, "--linhas vai de 1 a 5000")
		}
	}
	eventos, total, err := historico.Le(ops["--busca"], limite)
	if err != nil {
		return falha(err)
	}
	_ = base.Escreve(map[string]any{"eventos": eventos, "total": total})
	return base.Certo
}
