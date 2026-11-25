package main

import (
	"reflect"
	"testing"
)

// Enviar chave e gerar chave terminam sozinhos: o terminal tem de ficar aberto
// para a pessoa ler o que o comando disse. Conectar não espera.
func TestOTerminalEsperaDepoisDeEnviarEDeGerar(t *testing.T) {
	argv := []string{"ssh-copy-id", "-i", "/c/id.pub", "--", "atlas"}
	casos := map[string]string{"ghostty": "--wait-after-command=true", "kitty": "--hold", "foot": "--hold", "alacritty": "--hold"}
	for nome, opcao := range casos {
		com := linhaDoTerminal(nome, "t", argv, true)
		sem := linhaDoTerminal(nome, "t", argv, false)
		if !tem(com, opcao) {
			t.Errorf("%s: esperando, falta %s em %v", nome, opcao, com)
		}
		if tem(sem, opcao) {
			t.Errorf("%s: sem esperar, sobra %s em %v", nome, opcao, sem)
		}
		// A opção vem antes do comando, que continua inteiro no fim.
		if !reflect.DeepEqual(com[len(com)-len(argv):], argv) {
			t.Errorf("%s: o comando não está inteiro no fim: %v", nome, com)
		}
	}
}

func tem(lista []string, item string) bool {
	for _, x := range lista {
		if x == item {
			return true
		}
	}
	return false
}
