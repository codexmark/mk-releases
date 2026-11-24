package main

import (
	"fmt"
	"strings"
)

// opcoes separa os argumentos de um subcomando: os posicionais e as opções.
// `comValor` são as opções que levam um valor ("--lado 720"); `semValor`, as
// que só ligam algo. Depois de "--" tudo é posicional: é como se passa um
// arquivo cujo nome começa com "-". Opção desconhecida ou repetida é erro.
func opcoes(args, comValor, semValor []string) (pos []string, ops map[string]string, err error) {
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
		case tem(comValor, a):
			if i+1 >= len(args) {
				return nil, nil, fmt.Errorf("a opção %s pede um valor", a)
			}
			if _, repetida := ops[a]; repetida {
				return nil, nil, fmt.Errorf("a opção %s veio duas vezes", a)
			}
			i++
			ops[a] = args[i]
		case tem(semValor, a):
			ops[a] = "sim"
		case strings.HasPrefix(a, "-") && a != "-":
			return nil, nil, fmt.Errorf("não conheço a opção %s (para um arquivo cujo nome começa com \"-\", ponha -- antes)", a)
		default:
			pos = append(pos, a)
		}
	}
	return pos, ops, nil
}
