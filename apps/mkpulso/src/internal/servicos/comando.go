// Package servicos fala com o systemd pelo systemctl e pelo journalctl:
// lista as unidades, age nelas e lê o journal, nos termos do contrato
// (../../../CONTRATO.md). Os dois comandos são achados no PATH, então os
// testes põem dublês na frente (veja dubles/).
package servicos

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Quanto esperar por uma leitura. Uma ação não tem prazo: o polkit pode
// estar com o pedido de senha na tela.
const prazoDeLeitura = 20 * time.Second

// roda executa um comando externo e devolve o stdout. O erro já vem como
// frase: o comando não existe, demorou demais, ou o que ele disse no stderr
// (resumido numa linha).
func roda(nome string, prazo time.Duration, args ...string) ([]byte, error) {
	caminho, err := exec.LookPath(nome)
	if err != nil {
		return nil, fmt.Errorf("não achei o %s nesta máquina", nome)
	}
	ctx := context.Background()
	if prazo > 0 {
		var cancela context.CancelFunc
		ctx, cancela = context.WithTimeout(ctx, prazo)
		defer cancela()
	}
	var saida, erros bytes.Buffer
	cmd := exec.CommandContext(ctx, caminho, args...)
	cmd.Stdout, cmd.Stderr = &saida, &erros
	cmd.WaitDelay = time.Second
	err = cmd.Run()
	if err == nil {
		return saida.Bytes(), nil
	}
	if ctx.Err() != nil {
		return nil, fmt.Errorf("o %s demorou demais e foi interrompido", nome)
	}
	if frase := resumo(erros.String()); frase != "" {
		return nil, errors.New(frase)
	}
	var saiu *exec.ExitError
	if errors.As(err, &saiu) {
		return nil, fmt.Errorf("o %s saiu com o código %d", nome, saiu.ExitCode())
	}
	return nil, fmt.Errorf("não consegui rodar o %s: %v", nome, err)
}

// resumo fica com a primeira linha não vazia de um stderr, sem espaços
// repetidos e sem passar de 300 letras.
func resumo(stderr string) string {
	for _, linha := range strings.Split(stderr, "\n") {
		linha = strings.Join(strings.Fields(linha), " ")
		if linha == "" {
			continue
		}
		if letras := []rune(linha); len(letras) > 300 {
			linha = string(letras[:300]) + "…"
		}
		return linha
	}
	return ""
}

func systemctl(usuario bool, prazo time.Duration, args ...string) ([]byte, error) {
	if usuario {
		args = append([]string{"--user"}, args...)
	}
	saida, err := roda("systemctl", prazo, args...)
	if err != nil && usuario && strings.Contains(err.Error(), "Failed to connect") {
		return nil, fmt.Errorf("não consegui falar com o systemd do usuário (sem sessão aberta?): %v", err)
	}
	return saida, err
}

// Systemctl roda o systemctl (com --user, se usuario) para uma leitura ou
// uma ação rápida, e devolve o que ele escreveu. Quem chama já validou os
// argumentos: nome de unidade só passa por ValidaUnidade e depois de "--".
func Systemctl(usuario bool, args ...string) ([]byte, error) {
	return systemctl(usuario, prazoDeLeitura, args...)
}

// Mostra pede ao `systemctl show` as propriedades de várias unidades numa
// chamada só, e devolve um mapa por unidade, na ordem pedida. As datas vêm
// em segundos desde a época ("@1795470000").
func Mostra(usuario bool, propriedades, unidades []string) ([]map[string]string, error) {
	if len(unidades) == 0 {
		return nil, nil
	}
	args := append([]string{"show", "--timestamp=unix", "--property=" + strings.Join(propriedades, ","), "--"}, unidades...)
	saida, err := Systemctl(usuario, args...)
	if err != nil {
		return nil, err
	}
	var blocos []map[string]string
	for _, bloco := range strings.Split(strings.Trim(string(saida), "\n"), "\n\n") {
		m := map[string]string{}
		for _, linha := range strings.Split(bloco, "\n") {
			if chave, valor, ok := strings.Cut(linha, "="); ok {
				m[chave] = valor
			}
		}
		blocos = append(blocos, m)
	}
	// O systemd responde um bloco por unidade, na ordem. Se a conta não
	// fechar, casa pelo Id o que der.
	fichas := make([]map[string]string, len(unidades))
	if len(blocos) == len(unidades) {
		copy(fichas, blocos)
		return fichas, nil
	}
	porId := map[string]map[string]string{}
	for _, b := range blocos {
		porId[b["Id"]] = b
	}
	for i, u := range unidades {
		if b, ok := porId[u]; ok {
			fichas[i] = b
		} else {
			fichas[i] = map[string]string{}
		}
	}
	return fichas, nil
}
