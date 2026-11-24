package main

import (
	"sort"

	"mkpacotes/internal/alpm"
	"mkpacotes/internal/aur"
	"mkpacotes/internal/base"
	"mkpacotes/internal/pacman"
)

func init() { registrar("atualizacoes", cmdAtualizacoes) }

type atualizacao struct {
	Nome   string  `json:"nome"`
	De     string  `json:"de"`
	Para   string  `json:"para"`
	Origem *string `json:"origem"`
}

// mkpacotes-coletor atualizacoes [--sem-aur]
func cmdAtualizacoes(args []string) int {
	pos, ops, err := opcoes(args, []string{"--sem-aur"}, nil)
	if err != nil {
		return falha(err)
	}
	if len(pos) != 0 {
		return base.Falha(base.Erro, "uso: mkpacotes-coletor atualizacoes [--sem-aur]")
	}
	local, err := alpm.Le()
	if err != nil {
		return falha(err)
	}
	lista := []atualizacao{}
	var erroOficial, erroAur *string
	repos, erroDosRepos := pacman.Repos()
	mudancas, fonte, err := pacman.Atualizacoes()
	if err != nil {
		erroOficial = texto(err.Error())
	}
	for _, m := range mudancas {
		a := atualizacao{Nome: m.Nome, De: m.De, Para: m.Para}
		if s, ok := repos[m.Nome]; ok {
			a.Origem = &s.Repo
		}
		lista = append(lista, a)
	}
	oficiais := len(lista)

	if ops["--sem-aur"] == "" {
		switch {
		case erroDosRepos != nil:
			erroAur = texto("sem o pacman não dá para saber quais instalados são do AUR")
		default:
			var estrangeiros []string
			for _, n := range local.Nomes() {
				if _, ok := repos[n]; !ok && alpm.ValidaNome(n) == nil {
					estrangeiros = append(estrangeiros, n)
				}
			}
			fichas, err := aur.Info(estrangeiros)
			if err != nil {
				erroAur = texto(err.Error())
			}
			sort.Slice(fichas, func(i, j int) bool { return fichas[i].Nome < fichas[j].Nome })
			origem := "aur"
			for _, f := range fichas {
				if p, ok := local.Pacotes[f.Nome]; ok && alpm.Vercmp(f.Versao, p.Versao) > 0 {
					lista = append(lista, atualizacao{Nome: f.Nome, De: p.Versao, Para: f.Versao, Origem: &origem})
				}
			}
		}
	}
	_ = base.Escreve(map[string]any{
		"atualizacoes": lista,
		"totais":       map[string]int{"oficiais": oficiais, "aur": len(lista) - oficiais},
		"fonte":        fonte, "erro_oficial": erroOficial, "erro_aur": erroAur,
	})
	return base.Certo
}
