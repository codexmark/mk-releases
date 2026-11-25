package main

import (
	"mksom/internal/base"
	"mksom/internal/eq"
	"mksom/internal/pw"
)

func init() {
	registrar("estado", cmdEstado)
}

type somJSON struct {
	Responde bool    `json:"responde"`
	Erro     *string `json:"erro"`
}

type saidaJSON struct {
	ID        int    `json:"id"`
	Nome      string `json:"nome"`
	Descricao string `json:"descricao"`
}

type anteriorJSON struct {
	Nome      string  `json:"nome"`
	Descricao *string `json:"descricao"`
	Existe    bool    `json:"existe"`
}

type entradaJSON struct {
	ID        int    `json:"id"`
	Nome      string `json:"nome"`
	Descricao string `json:"descricao"`
	Padrao    bool   `json:"padrao"`
	Volume    *int   `json:"volume"`
	Mudo      bool   `json:"mudo"`
}

type eqJSON struct {
	Ligado        bool              `json:"ligado"`
	NaSaida       bool              `json:"na_saida"`
	Bandas        []float64         `json:"bandas"`
	Ganhos        []float64         `json:"ganhos"`
	Pre           float64           `json:"pre"`
	Predefinicao  *string           `json:"predefinicao"`
	SaidaAnterior *anteriorJSON     `json:"saida_anterior"`
	Predefinicoes []eq.Predefinicao `json:"predefinicoes"`
}

type estadoJSON struct {
	Som         somJSON       `json:"som"`
	Eq          eqJSON        `json:"eq"`
	SaidaPadrao *saidaJSON    `json:"saida_padrao"`
	Entradas    []entradaJSON `json:"entradas"`
}

// ouNulo devolve nil para o texto vazio: no contrato, o que não há é null.
func ouNulo(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// anteriorDe descreve a saída que era a padrão antes de ligar.
func anteriorDe(nome string, g pw.Grafo) *anteriorJSON {
	if nome == "" {
		return nil
	}
	a := &anteriorJSON{Nome: nome}
	if no, ok := g.Saida(nome); ok {
		a.Existe = true
		a.Descricao = &no.Descricao
	}
	return a
}

// entradasDe monta a lista de entradas como a janela a mostra.
func entradasDe(g pw.Grafo) []entradaJSON {
	out := []entradaJSON{}
	for _, n := range g.Entradas() {
		e := entradaJSON{ID: n.ID, Nome: n.Nome, Descricao: n.Descricao, Padrao: n.Nome == g.EntradaPadrao, Mudo: n.Mudo}
		if n.Volume >= 0 {
			v := n.Volume
			e.Volume = &v
		}
		out = append(out, e)
	}
	return out
}

func cmdEstado(args []string) int {
	if len(args) != 0 {
		return base.Falha(base.Erro, "uso: mksom-motor estado")
	}
	st := leEstado()
	g, err := leSom()
	r := estadoJSON{Som: somJSON{Responde: err == nil}, Entradas: entradasDe(g)}
	if err != nil {
		r.Som.Erro = ouNulo(err.Error())
	}
	_, noAr := g.Saida(eq.NoDoSink)
	r.Eq = eqJSON{
		Ligado:        noAr,
		NaSaida:       noAr && g.SaidaPadrao == eq.NoDoSink,
		Bandas:        eq.Frequencias,
		Ganhos:        st.Ganhos,
		Pre:           st.Pre,
		Predefinicao:  ouNulo(st.Predefinicao),
		SaidaAnterior: anteriorDe(st.SaidaAnterior, g),
		Predefinicoes: eq.Todas(lePredefinicoes()),
	}
	if no, ok := g.Saida(g.SaidaPadrao); ok {
		r.SaidaPadrao = &saidaJSON{no.ID, no.Nome, no.Descricao}
	}
	_ = base.Escreve(r)
	return base.Certo
}
