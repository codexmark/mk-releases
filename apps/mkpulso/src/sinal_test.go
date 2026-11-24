package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"mkpulso/internal/acoes"
	"mkpulso/internal/base"
	"mkpulso/internal/servicos/dubles"
)

// rodaAcao chama o coletor como a janela chama e devolve o código de saída
// e o que saiu no stdout.
func rodaAcao(t *testing.T, args ...string) (int, string) {
	t.Helper()
	var saida bytes.Buffer
	antes := base.Saida
	base.Saida = &saida
	defer func() { base.Saida = antes }()
	codigo := rodar(args)
	return codigo, saida.String()
}

// querAcao confere o código e a linha de JSON, ao pé da letra.
func querAcao(t *testing.T, codigo int, saida string, args ...string) {
	t.Helper()
	veioCodigo, veioSaida := rodaAcao(t, args...)
	if veioCodigo != codigo || veioSaida != saida+"\n" {
		t.Errorf("%v:\n veio %d %s quer %d %s", args, veioCodigo, veioSaida, codigo, saida)
	}
}

// querErroDeAcao confere o código e que saiu {"erro": "..."} com o trecho.
func querErroDeAcao(t *testing.T, codigo int, trecho string, args ...string) {
	t.Helper()
	veioCodigo, saida := rodaAcao(t, args...)
	if veioCodigo != codigo || !strings.HasPrefix(saida, `{"erro":"`) || !strings.HasSuffix(saida, "\"}\n") ||
		strings.Count(saida, "\n") != 1 || !strings.Contains(saida, trecho) {
		t.Errorf("%v: veio %d %s queria %d e um erro com %q", args, veioCodigo, saida, codigo, trecho)
	}
}

// procDeAcao monta um /proc falso com processos de quem roda o teste
// (nome → pid e pai), mais os que são de outro usuário.
func procDeAcao(t *testing.T, meus map[int]string, alheios ...int) {
	t.Helper()
	raiz := t.TempDir()
	escreve := func(pid, uid int, nome string) {
		dir := filepath.Join(raiz, strconv.Itoa(pid))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		status := fmt.Sprintf("Name:\t%s\nPid:\t%d\nPPid:\t1\nUid:\t%d\t%d\t%d\t%d\n", nome, pid, uid, uid, uid, uid)
		if err := os.WriteFile(filepath.Join(dir, "status"), []byte(status), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for pid, nome := range meus {
		escreve(pid, os.Getuid(), nome)
	}
	for _, pid := range alheios {
		escreve(pid, os.Getuid()+1, "alheio")
	}
	t.Setenv("MKPULSO_PROC", raiz)
}

// semSinaisDeAcao troca o envio de sinal e a mudança de prioridade por
// anotações: nada sai do teste.
func semSinaisDeAcao(t *testing.T) *[]string {
	t.Helper()
	var feitos []string
	mata, nice := acoes.Mata, acoes.MudaNice
	acoes.Mata = func(pid int, sig syscall.Signal) error {
		feitos = append(feitos, fmt.Sprintf("sinal %d ao %d", int(sig), pid))
		return nil
	}
	acoes.MudaNice = func(pid, valor int) error {
		feitos = append(feitos, fmt.Sprintf("nice %d ao %d", valor, pid))
		if valor < 0 {
			return syscall.EACCES
		}
		return nil
	}
	t.Cleanup(func() { acoes.Mata, acoes.MudaNice = mata, nice })
	return &feitos
}

func TestSinal_Contrato(t *testing.T) {
	dubles.Nova(t) // nenhum comando externo existe
	procDeAcao(t, map[int]string{1234: "firefox", 600: "Hyprland", 601: "quickshell"}, 1, 500)
	feitos := semSinaisDeAcao(t)

	querAcao(t, 0, `{"ok":true,"pid":1234,"sinal":"encerrar"}`, "sinal", "1234", "encerrar")
	querAcao(t, 0, `{"ok":true,"pid":1234,"sinal":"matar"}`, "sinal", "1234", "matar")
	querAcao(t, 0, `{"ok":true,"pid":600,"sinal":"recarregar"}`, "sinal", "600", "recarregar", "--mesmo-assim")
	querAcao(t, 0, `{"ok":true,"pid":601,"sinal":"pausar"}`, "sinal", "--mesmo-assim", "601", "pausar")
	if quer := []string{"sinal 15 ao 1234", "sinal 9 ao 1234", "sinal 1 ao 600", "sinal 19 ao 601"}; fmt.Sprint(*feitos) != fmt.Sprint(quer) {
		t.Errorf("sinais: %v, queria %v", *feitos, quer)
	}
	*feitos = nil

	// Recusado por segurança: código 2.
	querErroDeAcao(t, 2, "processo 1", "sinal", "1", "encerrar")
	querErroDeAcao(t, 2, "processo 1", "sinal", "1", "encerrar", "--mesmo-assim")
	querErroDeAcao(t, 2, "não é um processo", "sinal", "-1", "matar")
	querErroDeAcao(t, 2, "não é um processo", "sinal", "0", "matar")
	querErroDeAcao(t, 2, "próprio coletor", "sinal", strconv.Itoa(os.Getpid()), "encerrar")
	querErroDeAcao(t, 2, "quem chamou", "sinal", strconv.Itoa(os.Getppid()), "encerrar", "--mesmo-assim")
	querErroDeAcao(t, 2, "outro usuário", "sinal", "500", "encerrar")
	querErroDeAcao(t, 2, "sessão", "sinal", "600", "encerrar")
	querErroDeAcao(t, 2, "sessão", "sinal", "601", "matar")
	// Erro de uso ou de sistema: código 1.
	querErroDeAcao(t, 1, "já saiu", "sinal", "4242", "encerrar")
	querErroDeAcao(t, 1, "não é um número", "sinal", "1e9", "encerrar")
	querErroDeAcao(t, 1, "não é um número", "sinal", "firefox", "encerrar")
	querErroDeAcao(t, 1, "não é um número", "sinal", "99999999999999999999", "encerrar")
	querErroDeAcao(t, 1, "não conheço o sinal", "sinal", "1234", "SIGKILL")
	querErroDeAcao(t, 1, "não conheço o sinal", "sinal", "1234", "9")
	querErroDeAcao(t, 1, "uso:", "sinal")
	querErroDeAcao(t, 1, "uso:", "sinal", "1234")
	querErroDeAcao(t, 1, "uso:", "sinal", "1234", "encerrar", "matar")
	querErroDeAcao(t, 1, "não conheço a opção", "sinal", "1234", "encerrar", "--force")
	if len(*feitos) != 0 {
		t.Errorf("mandou sinal apesar da recusa: %v", *feitos)
	}
}
