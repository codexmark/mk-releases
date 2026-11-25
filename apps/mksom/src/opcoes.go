package main

import (
	"fmt"
	"strings"
)

// posicionais separa os argumentos de um subcomando. O motor não tem opção
// nenhuma: o que começa com "-" antes de um "--" é erro, e depois de "--"
// tudo é nome ou número (é assim que a janela manda o que a
// pessoa escolheu, para nada virar opção por acidente).
func posicionais(args []string) ([]string, error) {
	var pos []string
	for i, a := range args {
		switch {
		case a == "--":
			return append(pos, args[i+1:]...), nil
		case strings.HasPrefix(a, "-"):
			return nil, fmt.Errorf("não conheço a opção %s", a)
		default:
			pos = append(pos, a)
		}
	}
	return pos, nil
}
