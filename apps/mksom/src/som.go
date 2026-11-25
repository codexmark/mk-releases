package main

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"mksom/internal/base"
	"mksom/internal/pw"
)

// Quanto um comando externo de uma resposta só pode demorar.
var prazoDoComando = 4 * time.Second

// Os comandos externos do motor: a variável que os troca e o nome padrão.
type externo struct{ variavel, padrao string }

var (
	oSystemctl = externo{"MKSOM_SYSTEMCTL", "systemctl"}
	oPwDump    = externo{"MKSOM_PW_DUMP", "pw-dump"}
	oPwCli     = externo{"MKSOM_PW_CLI", "pw-cli"}
	oWpctl     = externo{"MKSOM_WPCTL", "wpctl"}
	oPipewire  = externo{"MKSOM_PIPEWIRE", "pipewire"}
	oPwRecord  = externo{"MKSOM_PW_RECORD", "pw-record"}
	oPwPlay    = externo{"MKSOM_PW_PLAY", "pw-play"}
)

func (e externo) nome() string {
	return base.Comando(e.variavel, e.padrao)
}

// caminho acha o executável, ou diz que falta.
func (e externo) caminho() (string, error) {
	c, err := exec.LookPath(e.nome())
	if err != nil {
		return "", fmt.Errorf("falta o %s (vem com o PipeWire do Mk)", e.padrao)
	}
	return c, nil
}

// primeiraLinha é o começo do que um comando reclamou, para a frase de erro.
func primeiraLinha(b []byte) string {
	s := strings.TrimSpace(string(b))
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if len(s) > 160 {
		s = s[:160]
	}
	return s
}

// roda executa um comando externo de uma resposta só, sem shell e com prazo,
// e devolve o que ele escreveu.
func (e externo) roda(args ...string) ([]byte, error) {
	caminho, err := e.caminho()
	if err != nil {
		return nil, err
	}
	ctx, cancela := context.WithTimeout(context.Background(), prazoDoComando)
	defer cancela()
	cmd := exec.CommandContext(ctx, caminho, args...)
	var saida, erros bytes.Buffer
	cmd.Stdout = &saida
	cmd.Stderr = &erros
	cmd.WaitDelay = time.Second
	err = cmd.Run()
	if ctx.Err() != nil {
		return nil, fmt.Errorf("o %s não respondeu a tempo", e.padrao)
	}
	if err != nil {
		if frase := primeiraLinha(erros.Bytes()); frase != "" {
			return nil, fmt.Errorf("o %s falhou: %s", e.padrao, frase)
		}
		return nil, fmt.Errorf("o %s falhou", e.padrao)
	}
	return saida.Bytes(), nil
}

// leSom pergunta ao PipeWire o que existe agora (pw-dump).
func leSom() (pw.Grafo, error) {
	dados, err := oPwDump.roda()
	if err != nil {
		return pw.Grafo{}, fmt.Errorf("o PipeWire não respondeu (%v)", err)
	}
	g, err := pw.Le(dados)
	if err != nil {
		return pw.Grafo{}, fmt.Errorf("o PipeWire não respondeu (%v)", err)
	}
	return g, nil
}
