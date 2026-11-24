package main

import (
	"errors"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"mkutil/internal/base"
	"mkutil/internal/midia"
)

func init() {
	registrar("codificadores", cmdCodificadores)
	registrar("midia", cmdMidia)
	registrar("comprimir", cmdComprimir)
	registrar("descartar", cmdDescartar)
}

// mkutil-motor codificadores
func cmdCodificadores(args []string) int {
	if len(args) != 0 {
		return base.Falha(base.Erro, "uso: mkutil-motor codificadores")
	}
	// Sem o ffmpeg não é erro: a janela mostra que falta.
	tem, _ := midia.Codificadores()
	_ = base.Escreve(midia.Resume(tem))
	return base.Certo
}

// mkutil-motor midia <arquivo>
func cmdMidia(args []string) int {
	pos, _, err := opcoes(args, nil, nil)
	if err != nil {
		return base.Falha(base.Erro, "%s", err.Error())
	}
	if len(pos) != 1 {
		return base.Falha(base.Erro, "uso: mkutil-motor midia <arquivo>")
	}
	m, err := midia.Sonda(pos[0])
	if err != nil {
		return base.FalhaDe(err)
	}
	_ = base.Escreve(m)
	return base.Certo
}

const usoDoComprimir = "uso: mkutil-motor comprimir <arquivo> --predefinicao <menor|equilibrio|alta|caber|resolucao|sem-audio|gif|webp|imagem> " +
	"[--alvo-mb N] [--lado 1080|720|480] [--codec h264|h265|av1] [--acelerar vaapi|nvenc] [--sem-audio] " +
	"[--formato jpeg|webp|avif] [--qualidade 1-100] [--largura N] [--fps N]"

// mkutil-motor comprimir <arquivo> --predefinicao <qual> [opções]
func cmdComprimir(args []string) int {
	pos, ops, err := opcoes(args, []string{"--predefinicao", "--alvo-mb", "--lado", "--codec", "--acelerar", "--formato", "--qualidade", "--largura", "--fps"},
		[]string{"--sem-audio"})
	if err != nil {
		return base.Falha(base.Erro, "%s", err.Error())
	}
	if len(pos) != 1 || ops["--predefinicao"] == "" {
		return base.Falha(base.Erro, "%s", usoDoComprimir)
	}
	pedido := midia.Pedido{Predefinicao: ops["--predefinicao"], Codec: ops["--codec"], Acelerar: ops["--acelerar"], Formato: ops["--formato"],
		SemAudio: ops["--sem-audio"] != ""}
	for opcao, onde := range map[string]*int{"--lado": &pedido.Lado, "--qualidade": &pedido.Qualidade, "--largura": &pedido.Largura, "--fps": &pedido.FPS} {
		if v, veio := ops[opcao]; veio {
			n, err := strconv.Atoi(v)
			if err != nil || n <= 0 {
				return base.Falha(base.Erro, "%s pede um número inteiro maior que zero, e veio %q", opcao, v)
			}
			*onde = n
		}
	}
	if v, veio := ops["--alvo-mb"]; veio {
		n, err := strconv.ParseFloat(v, 64)
		if err != nil || !(n > 0 && n <= 1e6) {
			return base.Falha(base.Erro, "--alvo-mb pede um número maior que zero, e veio %q", v)
		}
		pedido.AlvoMB = n
	}

	// Cancelar é mandar SIGTERM (a janela faz isso ao parar o processo).
	cancela := make(chan struct{})
	sinais := make(chan os.Signal, 1)
	signal.Notify(sinais, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	signal.Ignore(syscall.SIGPIPE) // a janela fechou: a escrita falha e isso também cancela
	go func() {
		<-sinais
		close(cancela)
	}()
	defer signal.Stop(sinais)

	fim, err := midia.Comprime(pos[0], pedido, base.Escreve, cancela)
	if errors.Is(err, midia.ErrCancelado) {
		_ = base.Escreve(map[string]string{"fase": "cancelado"})
		return base.Certo
	}
	if err != nil {
		return base.FalhaDe(err)
	}
	_ = base.Escreve(fim)
	return base.Certo
}

// mkutil-motor descartar <saida> --original <arquivo>
func cmdDescartar(args []string) int {
	pos, ops, err := opcoes(args, []string{"--original"}, nil)
	if err != nil {
		return base.Falha(base.Erro, "%s", err.Error())
	}
	if len(pos) != 1 || ops["--original"] == "" {
		return base.Falha(base.Erro, "uso: mkutil-motor descartar <saida> --original <arquivo>")
	}
	apagado, err := midia.Descarta(pos[0], ops["--original"])
	if err != nil {
		return base.FalhaDe(err)
	}
	_ = base.Escreve(struct {
		Ok      bool   `json:"ok"`
		Apagado string `json:"apagado"`
	}{true, apagado})
	return base.Certo
}
