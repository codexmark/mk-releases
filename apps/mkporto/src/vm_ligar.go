package main

import (
	"os"
	"os/exec"
	"syscall"
	"time"

	"mkporto/internal/base"
	"mkporto/internal/vm"
)

// Quanto se espera o QEMU atender no QMP depois de ligar, e sumir depois de
// mandado embora.
var (
	prazoDeLigar      = 10 * time.Second
	prazoDeMorrer     = 5 * time.Second
	passoDeEspera     = 100 * time.Millisecond
	linhasDoLogNoErro = 4
)

// vmLigar: sobe o QEMU da máquina e volta quando ele atende no QMP (ou
// morre). O processo nasce num grupo próprio, sem entrada nem saída herdadas,
// e continua depois que o motor sai. Nunca por shell.
func vmLigar(args []string) int {
	l, m, ops, codigo := umaMaquina(args, "ligar [--sem-iso] [--] <nome>", "--sem-iso")
	if codigo != base.Certo {
		return codigo
	}
	if s := situacaoDe(l, m.Nome); s.viva() {
		return base.Falha(base.Recusado, "a %s já está %s", m.Nome, s.Estado)
	}
	if l.Execucao == "" {
		return base.Falha(base.Recusado, "não há onde pôr o socket da máquina: falta o XDG_RUNTIME_DIR nesta sessão")
	}
	tela := base.Comando("MKPORTO_QEMU_DISPLAY", "gtk")
	if tela != "gtk" && tela != "none" {
		return base.Falha(base.Erro, "MKPORTO_QEMU_DISPLAY só aceita gtk ou none")
	}
	qemu := comandoDe("MKPORTO_QEMU", "qemu-system-x86_64")
	if !qemu.Tem {
		return base.Falha(base.Erro, "falta o QEMU: instale o pacote qemu-desktop")
	}
	ovmf := oOvmf()
	if m.UEFI {
		if !ovmf.Tem {
			return base.Falha(base.Erro, "esta máquina é UEFI e falta o OVMF: instale o pacote edk2-ovmf")
		}
		if _, err := os.Stat(l.Vars(m.Nome)); err != nil {
			return base.Falha(base.Erro, "a máquina perdeu o vars.fd dela (%s)", base.Til(l.Vars(m.Nome)))
		}
	}
	semIso := ops["--sem-iso"] != ""
	if m.ISO != "" && !semIso {
		if info, err := os.Stat(m.ISO); err != nil || !info.Mode().IsRegular() {
			return base.Falha(base.Erro, "a ISO não está mais em %s: ligue sem ela", m.ISO)
		}
	}
	if _, err := os.Stat(l.Disco(m.Nome)); err != nil {
		return base.Falha(base.Erro, "a máquina perdeu o disco dela (%s)", base.Til(l.Disco(m.Nome)))
	}
	if len(l.Socket(m.Nome)) > 100 {
		return base.Falha(base.Erro, "o caminho do socket da máquina é comprido demais (%s)", l.Socket(m.Nome))
	}

	argv, avisos := l.Argv(m, vm.Ambiente{Qemu: qemu.Caminho, KVM: oKvm().Tem, OVMF: ovmf.Caminho, Display: tela, SemISO: semIso})
	if err := l.PreparaCorrida(m.Nome); err != nil {
		return base.Falha(base.Erro, "%s", err)
	}
	registro, err := l.AbreLog(m.Nome)
	if err != nil {
		return base.Falha(base.Erro, "não consegui abrir o registro da máquina: %v", err)
	}
	// Sem contexto de propósito: a máquina continua depois que o motor sai.
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdout, cmd.Stderr = registro, registro
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	err = cmd.Start()
	registro.Close()
	if err != nil {
		return base.Falha(base.Erro, "não consegui rodar o QEMU: %v", err)
	}
	morreu := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(morreu)
	}()

	limite := time.After(prazoDeLigar)
	for {
		select {
		case <-morreu:
			l.LimpaCorrida(m.Nome)
			if fim := l.FimDoLog(m.Nome, linhasDoLogNoErro); fim != "" {
				return base.Falha(base.Erro, "o QEMU não subiu: %s", fim)
			}
			return base.Falha(base.Erro, "o QEMU não subiu e não disse por quê")
		case <-limite:
			return base.Falha(base.Erro, "o QEMU subiu (pid %d) mas não atendeu no QMP em %d s: veja %s",
				cmd.Process.Pid, int(prazoDeLigar/time.Second), base.Til(l.Log(m.Nome)))
		case <-time.After(passoDeEspera):
		}
		if _, err := os.Stat(l.Socket(m.Nome)); err != nil {
			continue
		}
		if estado, err := vm.Estado(l.Socket(m.Nome)); err == nil {
			_ = base.Escreve(map[string]any{"ok": true, "nome": m.Nome, "estado": estado, "pid": cmd.Process.Pid,
				"porta_ssh": m.PortaSSH, "avisos": avisos, "argv": argv})
			return base.Certo
		}
	}
}

// vmDesligar: pede o desligamento (o botão de desligar, pelo QMP) e volta
// logo; com --forcar, manda o QEMU embora (QMP `quit`; se ele não atende,
// SIGTERM no processo, depois de conferir que é o QEMU daquela máquina).
func vmDesligar(args []string) int {
	l, m, ops, codigo := umaMaquina(args, "desligar [--forcar] [--] <nome>", "--forcar")
	if codigo != base.Certo {
		return codigo
	}
	s := situacaoDe(l, m.Nome)
	if !s.viva() {
		return base.Falha(base.Erro, "a %s já está desligada", m.Nome)
	}
	if ops["--forcar"] == "" {
		if _, err := vm.Pergunta(l.Socket(m.Nome), "system_powerdown"); err != nil {
			return base.Falha(base.Erro, "%s: a máquina não recebeu o pedido (dá para forçar)", err)
		}
		_ = base.Escreve(map[string]any{"ok": true, "nome": m.Nome, "pedido": true, "forcado": false})
		return base.Certo
	}
	if _, err := vm.Pergunta(l.Socket(m.Nome), "quit"); err != nil {
		// O QMP não atende: sinal no processo, e só se ele ainda é o QEMU
		// desta máquina (o pid de um arquivo velho pode ser de outro programa).
		if !vm.EDaMaquina(oProc(), s.Pid, m.Nome, l.Pid(m.Nome)) {
			l.LimpaCorrida(m.Nome)
			return base.Falha(base.Erro, "o processo da %s já não é o QEMU dela: nada foi sinalizado", m.Nome)
		}
		if err := syscall.Kill(s.Pid, syscall.SIGTERM); err != nil {
			return base.Falha(base.Erro, "não consegui parar o QEMU (pid %d): %v", s.Pid, err)
		}
	}
	limite := time.Now().Add(prazoDeMorrer)
	for vm.EDaMaquina(oProc(), s.Pid, m.Nome, l.Pid(m.Nome)) {
		if time.Now().After(limite) {
			return base.Falha(base.Erro, "o QEMU da %s (pid %d) não saiu em %d s", m.Nome, s.Pid, int(prazoDeMorrer/time.Second))
		}
		time.Sleep(passoDeEspera)
	}
	l.LimpaCorrida(m.Nome)
	_ = base.Escreve(map[string]any{"ok": true, "nome": m.Nome, "pedido": true, "forcado": true})
	return base.Certo
}

// vmPausa: pausar (QMP `stop`) e continuar (`cont`).
func vmPausa(args []string, acao, comando, estado string) int {
	l, m, _, codigo := umaMaquina(args, acao+" [--] <nome>")
	if codigo != base.Certo {
		return codigo
	}
	if !situacaoDe(l, m.Nome).viva() {
		return base.Falha(base.Erro, "a %s está desligada", m.Nome)
	}
	if _, err := vm.Pergunta(l.Socket(m.Nome), comando); err != nil {
		return base.Falha(base.Erro, "%s", err)
	}
	_ = base.Escreve(map[string]any{"ok": true, "nome": m.Nome, "estado": estado})
	return base.Certo
}

// vmUso: quanto de CPU (em ticks, para a janela fazer a diferença) e de
// memória o QEMU da máquina está usando, pelo /proc.
func vmUso(args []string) int {
	l, m, _, codigo := umaMaquina(args, "uso [--] <nome>")
	if codigo != base.Certo {
		return codigo
	}
	s := situacaoDe(l, m.Nome)
	if !s.viva() {
		return base.Falha(base.Erro, "a %s está desligada", m.Nome)
	}
	uso, ok := vm.LeUso(oProc(), s.Pid)
	if !ok {
		return base.Falha(base.Erro, "não consegui ler o uso do processo %d", s.Pid)
	}
	_ = base.Escreve(map[string]any{"nome": m.Nome, "pid": s.Pid, "ticks": uso.Ticks, "rss_kb": uso.RssKB,
		"hz": 100, "agora_ms": time.Now().UnixMilli()})
	return base.Certo
}
