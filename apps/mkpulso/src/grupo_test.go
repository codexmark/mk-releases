package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"mkpulso/internal/servicos/dubles"
)

// cgroupDeAcao põe em MKPULSO_SYS o cgroup de uma unidade com os pids.
func cgroupDeAcao(t *testing.T, caminho string, pids ...int) {
	t.Helper()
	sys := os.Getenv("MKPULSO_SYS")
	if sys == "" {
		sys = t.TempDir()
		t.Setenv("MKPULSO_SYS", sys)
	}
	dir := filepath.Join(sys, "fs", "cgroup", caminho)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	var s strings.Builder
	for _, p := range pids {
		s.WriteString(strconv.Itoa(p) + "\n")
	}
	if err := os.WriteFile(filepath.Join(dir, "cgroup.procs"), []byte(s.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

// systemctlDeGrupo é um dublê que conhece os apps de testdata e o cgroup de
// cada unidade usada nos testes de grupo.
func systemctlDeGrupo(t *testing.T) *dubles.Mesa {
	t.Helper()
	m := dubles.Nova(t)
	cg := func(unidade, cgroup string) dubles.Regra {
		return dubles.Regra{Contem: " show --timestamp=unix --property=ControlGroup,MainPID -- " + unidade + " ", Texto: "ControlGroup=" + cgroup + "\nMainPID=0\n"}
	}
	m.Comando("systemctl",
		dubles.Regra{Contem: " --user list-units --type=scope,service ", Arquivo: "internal/servicos/testdata/usuario-list-units-app.json"},
		cg("app-firefox-1234.scope", "/user.slice/app.slice/app-firefox-1234.scope"),
		cg("app-firefox-5678.scope", "/user.slice/app.slice/app-firefox-5678.scope"),
		cg("sshd.service", "/system.slice/sshd.service"),
		cg("mare.service", "/user.slice/app.slice/mare.service"),
		dubles.Regra{Contem: " show ", Texto: "ControlGroup=\nMainPID=0\n"},
		dubles.Regra{Contem: ""},
	)
	cgroupDeAcao(t, "/user.slice/app.slice/app-firefox-1234.scope", 1234, 1240)
	cgroupDeAcao(t, "/user.slice/app.slice/app-firefox-5678.scope", 5678)
	cgroupDeAcao(t, "/system.slice/sshd.service", 812)
	cgroupDeAcao(t, "/user.slice/app.slice/mare.service", 601, 602)
	return m
}

// acoesNoSystemctl devolve só as chamadas que agem (não as de leitura).
func acoesNoSystemctl(m *dubles.Mesa) []string {
	var acoes []string
	for _, c := range m.Chamadas("systemctl") {
		if !strings.Contains(" "+c+" ", " show ") && !strings.Contains(" "+c+" ", " list-units ") {
			acoes = append(acoes, c)
		}
	}
	return acoes
}

func TestGrupo_Contrato(t *testing.T) {
	procDeAcao(t, map[int]string{1234: "firefox", 1240: "Web Content", 5678: "firefox", 2000: "kitty", 2001: "bash", 601: "quickshell", 602: "mare-start"}, 1, 812)
	feitos := semSinaisDeAcao(t)

	// Um app com unidades: o systemd cuida do grupo inteiro.
	m := systemctlDeGrupo(t)
	querAcao(t, 0, `{"ok":true,"id":"app:firefox","sinal":"encerrar","pids":[1234,1240,5678]}`, "grupo", "app:firefox", "encerrar")
	if quer := []string{"--user stop -- app-firefox-1234.scope", "--user stop -- app-firefox-5678.scope"}; !reflect.DeepEqual(acoesNoSystemctl(m), quer) {
		t.Errorf("encerrar app:firefox: %v", acoesNoSystemctl(m))
	}
	m = systemctlDeGrupo(t)
	querAcao(t, 0, `{"ok":true,"id":"app:firefox","sinal":"matar","pids":[1234,1240,5678]}`, "grupo", "app:firefox", "matar", "--pids", "1234,1240,5678")
	if quer := []string{"--user kill --signal=SIGKILL -- app-firefox-1234.scope", "--user kill --signal=SIGKILL -- app-firefox-5678.scope"}; !reflect.DeepEqual(acoesNoSystemctl(m), quer) {
		t.Errorf("matar app:firefox: %v", acoesNoSystemctl(m))
	}
	if len(*feitos) != 0 {
		t.Errorf("com unidade, o sinal é do systemd: %v", *feitos)
	}

	// Um app sem unidade: os pids vêm da janela.
	m = systemctlDeGrupo(t)
	querAcao(t, 0, `{"ok":true,"id":"app:kitty","sinal":"encerrar","pids":[2000,2001]}`, "grupo", "app:kitty", "encerrar", "--pids", "2000,2001,4242")
	if len(acoesNoSystemctl(m)) != 0 || fmt.Sprint(*feitos) != "[sinal 15 ao 2001 sinal 15 ao 2000]" {
		t.Errorf("app:kitty: systemctl %v, sinais %v", acoesNoSystemctl(m), *feitos)
	}
	*feitos = nil
	// Processos do app que ficaram fora da unidade também recebem.
	querAcao(t, 0, `{"ok":true,"id":"app:firefox","sinal":"pausar","pids":[1234,1240,2000,5678]}`, "grupo", "app:firefox", "pausar", "--pids=1234,2000")
	if fmt.Sprint(*feitos) != "[sinal 19 ao 2000]" {
		t.Errorf("fora da unidade: %v", *feitos)
	}
	*feitos = nil

	// Um serviço: de sistema sem opção, de usuário com --usuario.
	m = systemctlDeGrupo(t)
	querAcao(t, 0, `{"ok":true,"id":"servico:sshd.service","sinal":"encerrar","pids":[812]}`, "grupo", "servico:sshd.service", "encerrar")
	querAcao(t, 0, `{"ok":true,"id":"servico:sshd.service","sinal":"recarregar","pids":[812]}`, "grupo", "servico:sshd.service", "recarregar")
	querAcao(t, 0, `{"ok":true,"id":"servico:syncthing.service","sinal":"matar","pids":[]}`, "grupo", "servico:syncthing.service", "matar", "--usuario")
	if quer := []string{"stop -- sshd.service", "kill --signal=SIGHUP -- sshd.service", "--user kill --signal=SIGKILL -- syncthing.service"}; !reflect.DeepEqual(acoesNoSystemctl(m), quer) {
		t.Errorf("serviços: %v", acoesNoSystemctl(m))
	}

	// Recusas: código 2, e nada acontece.
	m = systemctlDeGrupo(t)
	querErroDeAcao(t, 2, "sessão", "grupo", "servico:mare.service", "encerrar", "--usuario")
	querErroDeAcao(t, 2, "--mesmo-assim", "grupo", "servico:sddm.service", "matar")
	querErroDeAcao(t, 2, "--mesmo-assim", "grupo", "servico:user@1000.service", "encerrar")
	querErroDeAcao(t, 2, "sessão", "grupo", "app:mare", "encerrar", "--pids", "2000,601")
	querErroDeAcao(t, 2, "processo 1", "grupo", "app:kitty", "encerrar", "--pids", "2000,1")
	querErroDeAcao(t, 2, "não é um processo", "grupo", "app:kitty", "matar", "--pids", "2000,-1")
	querErroDeAcao(t, 2, "outro usuário", "grupo", "app:kitty", "matar", "--pids", "812,2000")
	querErroDeAcao(t, 2, "outro usuário", "grupo", "app:firefox", "matar", "--pids", "812")
	querErroDeAcao(t, 2, "sistema", "grupo", "sistema:kernel", "matar", "--pids", "2000")
	querErroDeAcao(t, 2, "sistema", "grupo", "sistema:outros", "encerrar")
	// Erros de uso: código 1.
	querErroDeAcao(t, 1, "--pids", "grupo", "app:kitty", "encerrar")
	querErroDeAcao(t, 1, "não é um número", "grupo", "app:kitty", "encerrar", "--pids", "2000,1e9")
	querErroDeAcao(t, 1, "não é um número", "grupo", "app:kitty", "encerrar", "--pids", "2000;rm -rf")
	querErroDeAcao(t, 1, "não conheço o sinal", "grupo", "app:firefox", "explodir")
	querErroDeAcao(t, 1, "não é nome de unidade", "grupo", "servico:a.service;rm -rf", "encerrar")
	querErroDeAcao(t, 1, "não é uma unidade", "grupo", "servico:tmp.mount", "encerrar")
	querErroDeAcao(t, 1, "não conheço o grupo", "grupo", "firefox", "encerrar")
	querErroDeAcao(t, 1, "não é nome de unidade", "grupo", "servico:--now", "encerrar")
	querErroDeAcao(t, 1, "não conheço a opção", "grupo", "app:firefox", "encerrar", "--now")
	querErroDeAcao(t, 1, "uso:", "grupo", "app:firefox")
	if len(acoesNoSystemctl(m)) != 0 || len(*feitos) != 0 {
		t.Errorf("agiu apesar da recusa: systemctl %v, sinais %v", acoesNoSystemctl(m), *feitos)
	}
	// A sessão, só insistindo.
	querAcao(t, 0, `{"ok":true,"id":"servico:mare.service","sinal":"encerrar","pids":[601,602]}`, "grupo", "servico:mare.service", "encerrar", "--usuario", "--mesmo-assim")
}

func TestGrupo_SemSystemctlOsPidsAindaValem(t *testing.T) {
	dubles.Nova(t)
	procDeAcao(t, map[int]string{2000: "kitty"})
	feitos := semSinaisDeAcao(t)
	querAcao(t, 0, `{"ok":true,"id":"app:kitty","sinal":"interromper","pids":[2000]}`, "grupo", "app:kitty", "interromper", "--pids", "2000")
	if fmt.Sprint(*feitos) != "[sinal 2 ao 2000]" {
		t.Errorf("sinais: %v", *feitos)
	}
	querErroDeAcao(t, 1, "systemctl", "grupo", "app:kitty", "encerrar")
	querErroDeAcao(t, 1, "systemctl", "grupo", "servico:sshd.service", "encerrar")
}
