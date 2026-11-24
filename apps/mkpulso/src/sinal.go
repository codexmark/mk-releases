package main

import (
	"fmt"
	"strings"

	"mkpulso/internal/acoes"
	"mkpulso/internal/base"
)

func init() { registrar("sinal", cmdSinal) }

// opcoesDeAcao separa os argumentos de um subcomando de ação em posicionais
// e opções. Uma opção começa por "--" e tem de estar na lista: `simples` são
// as que só ligam algo, `comValor` as que pedem um valor ("--linhas 5" ou
// "--linhas=5"). Um número negativo ("-5") é posicional.
func opcoesDeAcao(args, simples, comValor []string) (posicionais []string, opcoes map[string]string, err error) {
	tem := func(lista []string, nome string) bool {
		for _, l := range lista {
			if l == nome {
				return true
			}
		}
		return false
	}
	opcoes = map[string]string{}
	for i := 0; i < len(args); i++ {
		if !strings.HasPrefix(args[i], "--") {
			posicionais = append(posicionais, args[i])
			continue
		}
		nome, valor, comIgual := strings.Cut(args[i], "=")
		switch {
		case tem(simples, nome) && !comIgual:
			opcoes[nome] = "sim"
		case tem(comValor, nome):
			if !comIgual {
				if i+1 >= len(args) {
					return nil, nil, fmt.Errorf("a opção %s pede um valor", nome)
				}
				i++
				valor = args[i]
			}
			opcoes[nome] = valor
		default:
			return nil, nil, fmt.Errorf("não conheço a opção %s", nome)
		}
	}
	return posicionais, opcoes, nil
}

// falhaDeAcao escreve o erro e escolhe o código: 2 para uma recusa de
// segurança, 1 para o resto.
func falhaDeAcao(err error) int {
	return base.Falha(acoes.Codigo(err), "%s", err)
}

// mkpulso-coletor sinal <pid> <sinal> [--mesmo-assim]
func cmdSinal(args []string) int {
	pos, ops, err := opcoesDeAcao(args, []string{"--mesmo-assim"}, nil)
	if err != nil {
		return falhaDeAcao(err)
	}
	if len(pos) != 2 {
		return base.Falha(base.Erro, "uso: mkpulso-coletor sinal <pid> <%s> [--mesmo-assim]", strings.Join(acoes.Sinais(), "|"))
	}
	pid, err := acoes.LeNumero(pos[0])
	if err != nil {
		return falhaDeAcao(fmt.Errorf("o pid %v", err))
	}
	if err := acoes.Sinal(pid, pos[1], ops["--mesmo-assim"] != ""); err != nil {
		return falhaDeAcao(err)
	}
	_ = base.Escreve(struct {
		Ok    bool   `json:"ok"`
		Pid   int    `json:"pid"`
		Sinal string `json:"sinal"`
	}{true, pid, pos[1]})
	return base.Certo
}
