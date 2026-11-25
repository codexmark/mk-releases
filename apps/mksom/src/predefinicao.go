package main

import (
	"mksom/internal/base"
	"mksom/internal/eq"
)

type predefinicaoJSON struct {
	Ok            bool              `json:"ok"`
	Predefinicao  *string           `json:"predefinicao"`
	Predefinicoes []eq.Predefinicao `json:"predefinicoes"`
	Ganhos        []float64         `json:"ganhos"`
	Pre           float64           `json:"pre"`
	AoVivo        bool              `json:"ao_vivo"`
	Aplicados     []string          `json:"aplicados"`
}

func respondePredefinicao(st guardado, daPessoa []eq.Predefinicao, aoVivo bool, aplicados []string) int {
	if aplicados == nil {
		aplicados = []string{}
	}
	_ = base.Escreve(predefinicaoJSON{true, ouNulo(st.Predefinicao), eq.Todas(daPessoa), st.Ganhos, st.Pre, aoVivo, aplicados})
	return base.Certo
}

// posicao acha uma predefinição da pessoa pelo nome (sem olhar maiúsculas).
func posicao(lista []eq.Predefinicao, nome string) int {
	for i, p := range lista {
		if mesmoNome(p.Nome, nome) {
			return i
		}
	}
	return -1
}

func eqPredefinicao(args []string) int {
	if len(args) == 0 {
		return base.Falha(base.Erro, usoDoEq)
	}
	nomes, err := posicionais(args[1:])
	if err != nil {
		return base.Falha(base.Erro, "%v", err)
	}
	quantos := map[string]int{"aplicar": 1, "guardar": 1, "renomear": 2, "apagar": 1}
	if n, ok := quantos[args[0]]; !ok || len(nomes) != n {
		return base.Falha(base.Erro, usoDoEq)
	}
	daPessoa := lePredefinicoes()
	switch args[0] {
	case "aplicar":
		p, ok := eq.Acha(eq.Todas(daPessoa), nomes[0])
		if !ok {
			return base.Falha(base.Erro, "não há predefinição com esse nome")
		}
		aoVivo, aplicados, st, err := aplica(p.Ajuste(), p.Nome)
		if err != nil {
			return base.Falha(base.Erro, "%v", err)
		}
		return respondePredefinicao(st, daPessoa, aoVivo, aplicados)

	case "guardar":
		nome, err := eq.ValidaNome(nomes[0])
		if err != nil {
			return base.Falha(base.Erro, "%v", err)
		}
		if eq.DeFabrica(nome) {
			return base.Falha(base.Recusado, "“%s” é uma predefinição de fábrica: escolha outro nome", nome)
		}
		st := leEstado()
		nova := eq.Predefinicao{Nome: nome, Ganhos: st.Ganhos, Pre: st.Pre}
		if i := posicao(daPessoa, nome); i >= 0 {
			daPessoa[i] = nova
		} else if len(daPessoa) >= eq.PredefinicoesMax {
			return base.Falha(base.Erro, "já há %d predefinições guardadas: apague uma antes", eq.PredefinicoesMax)
		} else {
			daPessoa = append(daPessoa, nova)
		}
		if err := gravaPredefinicoes(daPessoa); err != nil {
			return base.Falha(base.Erro, "não consegui guardar a predefinição: %v", err)
		}
		st.Predefinicao = nome
		if err := gravaEstado(st); err != nil {
			return base.Falha(base.Erro, "não consegui gravar o estado: %v", err)
		}
		return respondePredefinicao(st, daPessoa, false, nil)

	case "renomear":
		if eq.DeFabrica(nomes[0]) {
			return base.Falha(base.Recusado, "as predefinições de fábrica não mudam de nome")
		}
		i := posicao(daPessoa, nomes[0])
		if i < 0 {
			return base.Falha(base.Erro, "não há predefinição com esse nome")
		}
		novo, err := eq.ValidaNome(nomes[1])
		if err != nil {
			return base.Falha(base.Erro, "%v", err)
		}
		if eq.DeFabrica(novo) {
			return base.Falha(base.Recusado, "“%s” é uma predefinição de fábrica: escolha outro nome", novo)
		}
		if j := posicao(daPessoa, novo); j >= 0 && j != i {
			return base.Falha(base.Erro, "já há uma predefinição com esse nome")
		}
		st := leEstado()
		if mesmoNome(st.Predefinicao, daPessoa[i].Nome) {
			st.Predefinicao = novo
		}
		daPessoa[i].Nome = novo
		if err := gravaPredefinicoes(daPessoa); err != nil {
			return base.Falha(base.Erro, "não consegui guardar a predefinição: %v", err)
		}
		if err := gravaEstado(st); err != nil {
			return base.Falha(base.Erro, "não consegui gravar o estado: %v", err)
		}
		return respondePredefinicao(st, daPessoa, false, nil)
	}

	// apagar
	if eq.DeFabrica(nomes[0]) {
		return base.Falha(base.Recusado, "as predefinições de fábrica não se apagam")
	}
	i := posicao(daPessoa, nomes[0])
	if i < 0 {
		return base.Falha(base.Erro, "não há predefinição com esse nome")
	}
	st := leEstado()
	if mesmoNome(st.Predefinicao, daPessoa[i].Nome) {
		st.Predefinicao = ""
	}
	daPessoa = append(daPessoa[:i], daPessoa[i+1:]...)
	if err := gravaPredefinicoes(daPessoa); err != nil {
		return base.Falha(base.Erro, "não consegui guardar as predefinições: %v", err)
	}
	if err := gravaEstado(st); err != nil {
		return base.Falha(base.Erro, "não consegui gravar o estado: %v", err)
	}
	return respondePredefinicao(st, daPessoa, false, nil)
}
