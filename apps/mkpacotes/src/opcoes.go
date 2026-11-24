package main

import (
	"errors"
	"fmt"

	"mkpacotes/internal/acoes"
	"mkpacotes/internal/base"
)

// opcoes separa os argumentos de um subcomando: os de posição, as bandeiras
// (viram "sim") e as opções com valor. Opção desconhecida é erro: nada que
// começa com "-" passa adiante sem ter sido previsto. Depois de "--", tudo é
// de posição (e quem recebe valida).
func opcoes(args, bandeiras, comValor []string) (pos []string, ops map[string]string, err error) {
	ops = map[string]string{}
	tem := func(lista []string, a string) bool {
		for _, x := range lista {
			if x == a {
				return true
			}
		}
		return false
	}
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			return append(pos, args[i+1:]...), ops, nil
		case tem(bandeiras, a):
			ops[a] = "sim"
		case tem(comValor, a):
			if i+1 >= len(args) {
				return nil, nil, fmt.Errorf("faltou o valor de %s", a)
			}
			i++
			ops[a] = args[i]
		case len(a) > 0 && a[0] == '-':
			return nil, nil, fmt.Errorf("não conheço a opção %s", a)
		default:
			pos = append(pos, a)
		}
	}
	return pos, ops, nil
}

// falha responde um erro com o código do contrato: 2 para uma recusa por
// segurança, 1 para o resto.
func falha(err error) int {
	var r *acoes.Recusa
	if errors.As(err, &r) {
		return base.Falha(base.Recusado, "%s", err.Error())
	}
	return base.Falha(base.Erro, "%s", err.Error())
}
