package main

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"mksom/internal/base"
	"mksom/internal/nivel"
	"mksom/internal/pw"
)

// O medidor: a taxa pedida ao pw-record e quantas linhas por segundo.
const (
	taxaDoMedidor      = 16000
	linhasPorSegundo   = 15
	taxaDaGravacao     = 48000
	nomeDaGravacao     = "teste.wav"
	limiteDeReclamacao = 4 << 10
)

// A gravação de teste: quanto dura, de quanto em quanto a contagem anda, e
// quanto um filho tem para sair depois do sinal antes de ser morto.
var (
	duracaoDaGravacao = 5 * time.Second
	passoDaContagem   = time.Second
	prazoDeMorrer     = 2 * time.Second
)

// ateParar devolve um contexto que acaba quando a janela manda parar: a
// entrada padrão fecha (a janela fechou ou largou o fluxo), chega a linha
// "parar", ou vem um SIGTERM/SIGINT.
func ateParar() (context.Context, context.CancelFunc) {
	ctx, cancela := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	fio := entrada
	go func() {
		leitor := bufio.NewScanner(fio)
		for leitor.Scan() {
			if strings.TrimSpace(leitor.Text()) == "parar" {
				break
			}
		}
		cancela()
	}()
	return ctx, cancela
}

// alvoDe é o que vai no --target do pw-record: o object.serial do nó (é o
// que o pw-record de hoje entende por um número), ou o id se o dump não diz.
func alvoDe(no pw.No) string {
	if no.Serie >= 0 {
		return strconv.Itoa(no.Serie)
	}
	return strconv.Itoa(no.ID)
}

// filho prepara um comando que morre com o contexto (pelo sinal dado, e à
// força se demorar) e junto com o motor.
func filho(ctx context.Context, caminho string, sinal syscall.Signal, args ...string) (*exec.Cmd, *bytes.Buffer) {
	cmd := exec.CommandContext(ctx, caminho, args...)
	cmd.Cancel = func() error { return cmd.Process.Signal(sinal) }
	cmd.WaitDelay = prazoDeMorrer
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGTERM}
	erros := &bytes.Buffer{}
	cmd.Stderr = &limitado{b: erros, resta: limiteDeReclamacao}
	return cmd, erros
}

// limitado guarda só o começo do que um filho reclama.
type limitado struct {
	b     *bytes.Buffer
	resta int
}

func (l *limitado) Write(p []byte) (int, error) {
	if n := min(len(p), l.resta); n > 0 {
		l.b.Write(p[:n])
		l.resta -= n
	}
	return len(p), nil
}

// porque monta o fim de uma frase de erro com o que o comando reclamou.
func porque(erros *bytes.Buffer) string {
	if frase := primeiraLinha(erros.Bytes()); frase != "" {
		return ": " + frase
	}
	return ""
}

func fim() int {
	_ = base.Escreve(map[string]bool{"fim": true})
	return base.Certo
}

func micNivel(id string) int {
	no, codigo := entradaDe(id)
	if codigo != base.Certo {
		return codigo
	}
	caminho, err := oPwRecord.caminho()
	if err != nil {
		return base.Falha(base.Erro, "%v", err)
	}
	ctx, cancela := ateParar()
	defer cancela()
	// --raw: PCM cru. Sem ele o pw-record manda um contêiner AU (cabeçalho e
	// big-endian); o leitor entende os dois, mas cru não há o que errar.
	cmd, erros := filho(ctx, caminho, syscall.SIGTERM,
		"--raw", "--target", alvoDe(no), "--format", "s16", "--rate", strconv.Itoa(taxaDoMedidor), "--channels", "1", "-")
	saida, err := cmd.StdoutPipe()
	if err == nil {
		err = cmd.Start()
	}
	if err != nil {
		return base.Falha(base.Erro, "não consegui ouvir a entrada: %v", err)
	}
	// Se a janela some (escrever falha), o medidor para.
	if nivel.Medir(saida, taxaDoMedidor/linhasPorSegundo, func(n nivel.Nivel) error { return base.Escreve(n) }) != nil {
		cancela()
	}
	_, _ = io.Copy(io.Discard, saida)
	err = cmd.Wait()
	if ctx.Err() != nil {
		return fim()
	}
	if err != nil {
		return base.Falha(base.Erro, "a entrada parou de responder%s", porque(erros))
	}
	return fim()
}

// arquivoDaGravacao é onde a gravação de teste mora: na pasta em memória.
func arquivoDaGravacao() (string, error) {
	pasta, err := base.PastaTemporaria()
	if err != nil {
		return "", err
	}
	return filepath.Join(pasta, nomeDaGravacao), nil
}

func micGravar(id string) int {
	arquivo, err := arquivoDaGravacao()
	if err != nil {
		return base.Falha(base.Recusado, "%v", err)
	}
	no, codigo := entradaDe(id)
	if codigo != base.Certo {
		return codigo
	}
	caminho, err := oPwRecord.caminho()
	if err != nil {
		return base.Falha(base.Erro, "%v", err)
	}
	pasta := filepath.Dir(arquivo)
	if err := os.MkdirAll(pasta, 0o700); err == nil {
		err = os.Chmod(pasta, 0o700)
	}
	if err != nil {
		return base.Falha(base.Erro, "não consegui preparar a pasta da gravação: %v", err)
	}
	_ = os.Remove(arquivo)

	ctx, cancela := ateParar()
	defer cancela()
	cmd, erros := filho(ctx, caminho, syscall.SIGINT,
		"--target", alvoDe(no), "--format", "s16", "--rate", strconv.Itoa(taxaDaGravacao), "--channels", "1", arquivo)
	if err := cmd.Start(); err != nil {
		return base.Falha(base.Erro, "não consegui gravar: %v", err)
	}
	acabou := make(chan error, 1)
	go func() { acabou <- cmd.Wait() }()

	for falta := int(duracaoDaGravacao / passoDaContagem); falta >= 1; falta-- {
		_ = base.Escreve(map[string]int{"falta": falta})
		select {
		case <-time.After(passoDaContagem):
		case <-ctx.Done():
			<-acabou
			_ = os.Remove(arquivo)
			return fim()
		case <-acabou:
			_ = os.Remove(arquivo)
			return base.Falha(base.Erro, "a gravação parou antes da hora%s", porque(erros))
		}
	}
	// Deu o tempo: o mesmo sinal de um Ctrl+C, para o pw-record fechar o arquivo.
	_ = cmd.Process.Signal(syscall.SIGINT)
	select {
	case <-acabou:
	case <-time.After(prazoDeMorrer):
		_ = cmd.Process.Kill()
		<-acabou
	}
	if i, err := os.Stat(arquivo); err != nil || i.Size() == 0 {
		_ = os.Remove(arquivo)
		return base.Falha(base.Erro, "a gravação saiu vazia%s", porque(erros))
	}
	_ = base.Escreve(map[string]any{"ok": true, "segundos": int(duracaoDaGravacao / time.Second)})
	return base.Certo
}

func micOuvir() int {
	arquivo, err := arquivoDaGravacao()
	if err != nil {
		return base.Falha(base.Recusado, "%v", err)
	}
	if !existe(arquivo) {
		return base.Falha(base.Recusado, "não há gravação para ouvir: grave primeiro")
	}
	// Tocou (ou não deu): a gravação some de qualquer jeito.
	defer os.Remove(arquivo)
	caminho, err := oPwPlay.caminho()
	if err != nil {
		return base.Falha(base.Erro, "%v", err)
	}
	ate, cancela := ateParar()
	defer cancela()
	ctx, solta := context.WithTimeout(ate, duracaoDaGravacao+10*time.Second)
	defer solta()
	cmd, erros := filho(ctx, caminho, syscall.SIGTERM, arquivo)
	err = cmd.Run()
	if ate.Err() != nil {
		return fim()
	}
	if err != nil {
		return base.Falha(base.Erro, "não consegui tocar a gravação%s", porque(erros))
	}
	_ = base.Escreve(map[string]bool{"ok": true})
	return base.Certo
}

func micLimpar() int {
	if arquivo, err := arquivoDaGravacao(); err == nil {
		_ = os.Remove(arquivo)
	}
	_ = base.Escreve(map[string]bool{"ok": true})
	return base.Certo
}
