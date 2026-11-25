package main

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"mkporto/internal/base"
	"mkporto/internal/sshcfg"
)

func init() { registrar("entrar", entrar) }

// Quanto se espera pelo teste de entrada inteiro (o ssh tem 5 s para conectar).
var prazoDaEntrada = 10 * time.Second

const motivoMaximo = 160

// entrar: testa se a pessoa entra na máquina com o que já tem (agente,
// chaves), sem abrir sessão e sem conversa: o ssh roda `true` lá e sai. Com
// BatchMode ele não pergunta nada, e nunca aceita sozinho a chave de um host.
func entrar(args []string) int {
	pos, err := posicionais(args)
	if err != nil {
		return base.Falha(base.Erro, "%s", err)
	}
	if len(pos) != 1 {
		return base.Falha(base.Erro, "uso: mkporto-motor entrar [--] <apelido>")
	}
	apelido := pos[0]
	if codigo := confereApelido(apelido); codigo != base.Certo {
		return codigo
	}
	ssh, tem := oSsh()
	if !tem {
		return base.Falha(base.Erro, "não achei o ssh nesta máquina")
	}

	ctx, cancela := context.WithTimeout(context.Background(), prazoDaEntrada)
	defer cancela()
	cmd := exec.CommandContext(ctx, ssh, "-o", "BatchMode=yes", "-o", "ConnectTimeout=5", "-o", "NumberOfPasswordPrompts=0", "--", apelido, "true")
	cmd.WaitDelay = 200 * time.Millisecond
	var reclamou bytes.Buffer
	cmd.Stderr = &limitado{b: &reclamou, max: 16 * 1024}
	inicio := time.Now()
	err = cmd.Run()
	ms := time.Since(inicio).Milliseconds()

	resposta := map[string]any{"apelido": apelido, "entra": err == nil, "motivo": nil, "ms": ms}
	if err != nil {
		codigo := -1
		var saiu *exec.ExitError
		if errors.As(err, &saiu) {
			codigo = saiu.ExitCode()
		}
		resposta["motivo"] = motivoDaEntrada(reclamou.String(), codigo, ctx.Err() != nil)
	}
	_ = base.Escreve(resposta)
	return base.Certo
}

// confereApelido escreve o erro e devolve o código se o apelido não existe
// (1) ou não pode ir para uma linha de comando (2).
func confereApelido(apelido string) int {
	existe := false
	for _, h := range catalogo() {
		if h.Apelido == apelido {
			existe = true
			break
		}
	}
	if !existe {
		return base.Falha(base.Erro, "não conheço o host %s", apelido)
	}
	// Um Host do config da pessoa pode ter qualquer coisa no nome; para uma
	// linha de comando só vai o que não tem como virar opção nem shell.
	if !sshcfg.ApelidoSeguro(apelido) {
		return base.Falha(base.Recusado, "o apelido %s tem caracteres que o MkPorto não põe numa linha de comando", apelido)
	}
	return base.Certo
}

// motivoDaEntrada traduz o que o ssh reclamou numa frase para a pessoa.
func motivoDaEntrada(reclamou string, codigo int, estourou bool) string {
	tem := func(trecho string) bool { return strings.Contains(reclamou, trecho) }
	switch {
	case tem("REMOTE HOST IDENTIFICATION HAS CHANGED"):
		return "a chave do host mudou: confira antes de conectar"
	case tem("Host key verification failed"):
		return "a chave do host ainda não foi aceita: conecte uma vez pelo terminal"
	case tem("Permission denied"):
		return "a máquina não aceitou nenhuma das suas chaves (envie uma com enviar chave)"
	case estourou, tem("timed out"):
		return "a máquina não respondeu a tempo"
	case tem("Connection refused"):
		return "a máquina recusou a conexão"
	case tem("Could not resolve hostname"):
		return "não achei esse nome"
	case tem("No route to host"), tem("Network is unreachable"):
		return "sem rota até lá"
	}
	if ultima := ultimaLinha(reclamou, motivoMaximo); ultima != "" {
		return ultima
	}
	return "o ssh saiu com o código " + strconv.Itoa(codigo)
}
