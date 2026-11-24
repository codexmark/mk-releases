package main

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"mkpulso/internal/acoes"
	"mkpulso/internal/base"
	"mkpulso/internal/servicos"
)

func init() { registrar("grupo", cmdGrupo) }

// mkpulso-coletor grupo <id> <sinal> [--pids 1,2,3] [--usuario] [--mesmo-assim]
//
// `servico:<unidade>` vai pelo systemd (de usuário com --usuario).
// `app:<nome>` vai pelo systemd nas unidades `app-<nome>-*.scope` e
// `app-<nome>@*.service` que houver; os processos de --pids que ficaram fora
// delas (ou todos, quando o app não tem unidade) recebem o sinal um a um,
// dos filhos para os pais. É a janela que conhece os pids, pelo retrato.
func cmdGrupo(args []string) int {
	pos, ops, err := opcoesDeAcao(args, []string{"--usuario", "--mesmo-assim"}, []string{"--pids"})
	if err != nil {
		return falhaDeAcao(err)
	}
	if len(pos) != 2 {
		return base.Falha(base.Erro, "uso: mkpulso-coletor grupo <app:nome|servico:unidade> <%s> [--pids 1,2,3] [--usuario] [--mesmo-assim]", strings.Join(acoes.Sinais(), "|"))
	}
	id, sinal := pos[0], pos[1]
	mesmoAssim := ops["--mesmo-assim"] != ""
	if _, err := acoes.NumeroDoSinal(sinal); err != nil {
		return falhaDeAcao(err)
	}
	var dados []int
	if lista, tem := ops["--pids"]; tem {
		for _, campo := range strings.Split(lista, ",") {
			pid, err := acoes.LeNumero(campo)
			if err != nil {
				return falhaDeAcao(fmt.Errorf("em --pids, %v", err))
			}
			dados = append(dados, pid)
		}
	}

	var pids []int
	switch {
	case strings.HasPrefix(id, "sistema:"):
		err = acoes.Recusar("não mando sinal a um grupo do sistema inteiro: escolha o processo")
	case strings.HasPrefix(id, "servico:"):
		pids, err = servicos.SinalNaUnidade(strings.TrimPrefix(id, "servico:"), ops["--usuario"] != "", sinal, mesmoAssim)
	case strings.HasPrefix(id, "app:"):
		pids, err = sinalNoApp(strings.TrimPrefix(id, "app:"), dados, sinal, mesmoAssim)
	default:
		err = fmt.Errorf("não conheço o grupo %q: o id começa por app: ou servico:", id)
	}
	if err != nil {
		return falhaDeAcao(err)
	}
	if pids == nil {
		pids = []int{}
	}
	_ = base.Escreve(struct {
		Ok    bool   `json:"ok"`
		ID    string `json:"id"`
		Sinal string `json:"sinal"`
		Pids  []int  `json:"pids"`
	}{true, id, sinal, pids})
	return base.Certo
}

// sinalNoApp manda o sinal a um app: pelas unidades dele e, para o que ficou
// fora delas, pelos pids dados. As recusas dos pids valem antes de qualquer
// coisa acontecer.
func sinalNoApp(nome string, dados []int, sinal string, mesmoAssim bool) ([]int, error) {
	for _, pid := range dados {
		if err := acoes.Confere(pid, mesmoAssim); acoes.EhRecusa(err) {
			return nil, err
		}
	}
	unidades, errUnidades := servicos.UnidadesDoApp(nome)
	if len(unidades) == 0 && len(dados) == 0 {
		if errUnidades != nil {
			return nil, fmt.Errorf("não deu para achar as unidades de app:%s (%v) e não vieram os processos em --pids", nome, errUnidades)
		}
		return nil, errors.New("app:" + nome + " não tem unidade do systemd: passe os processos dele com --pids")
	}
	feitos := map[int]bool{}
	for _, u := range unidades {
		daUnidade, err := servicos.SinalNaUnidade(u, true, sinal, mesmoAssim)
		if err != nil {
			return nil, err
		}
		for _, pid := range daUnidade {
			feitos[pid] = true
		}
	}
	var resto []int
	for _, pid := range dados {
		if !feitos[pid] {
			resto = append(resto, pid)
		}
	}
	if len(resto) > 0 {
		avulsos, err := acoes.Grupo(resto, sinal, mesmoAssim)
		if err != nil {
			return nil, err
		}
		for _, pid := range avulsos {
			feitos[pid] = true
		}
	}
	pids := make([]int, 0, len(feitos))
	for pid := range feitos {
		pids = append(pids, pid)
	}
	sort.Ints(pids)
	return pids, nil
}
