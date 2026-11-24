package main

import (
	"mkpulso/internal/base"
	"mkpulso/internal/inicio"
)

func init() { registrar("inicio", cmdInicio) }

// mkpulso-coletor inicio
// mkpulso-coletor inicio <ligar|desligar> <id>
func cmdInicio(args []string) int {
	pos, _, err := opcoesDeAcao(args, nil, nil)
	if err != nil {
		return falhaDeAcao(err)
	}
	if len(pos) == 0 {
		itens, err := inicio.Lista()
		if err != nil {
			return falhaDeAcao(err)
		}
		_ = base.Escreve(struct {
			Itens []inicio.Item `json:"itens"`
		}{itens})
		return base.Certo
	}
	if len(pos) != 2 || (pos[0] != "ligar" && pos[0] != "desligar") {
		return base.Falha(base.Erro, "uso: mkpulso-coletor inicio [<ligar|desligar> <id>]")
	}
	ligar := pos[0] == "ligar"
	if err := inicio.Muda(pos[1], ligar); err != nil {
		return falhaDeAcao(err)
	}
	_ = base.Escreve(struct {
		Ok     bool   `json:"ok"`
		ID     string `json:"id"`
		Ligado bool   `json:"ligado"`
	}{true, pos[1], ligar})
	return base.Certo
}
