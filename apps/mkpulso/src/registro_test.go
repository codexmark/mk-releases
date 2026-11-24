package main

import (
	"reflect"
	"strings"
	"testing"

	"mkpulso/internal/servicos/dubles"
)

func TestRegistro_Contrato(t *testing.T) {
	m := dubles.Nova(t)
	m.Comando("journalctl", dubles.Regra{Contem: "", Arquivo: "internal/servicos/testdata/journal.jsonl"})
	codigo, saida := rodaAcao(t, "registro", "sshd.service")
	if codigo != 0 || strings.Count(saida, "\n") != 1 {
		t.Fatalf("veio %d %s", codigo, saida)
	}
	comeco := `{"linhas":[{"t_s":1795470000,"nivel":"info","texto":"Server listening on 0.0.0.0 port 22."},` +
		`{"t_s":1795470060,"nivel":"erro","texto":"error: kex_exchange_identification: Connection closed by remote host"},` +
		`{"t_s":1795470061,"nivel":"aviso","texto":"spa.alsa: front:0p: (0 suppressed) snd_pcm_avail after recover: Pipe quebrado"},` +
		`{"t_s":1795470062,"nivel":"depuração","texto":"debug1: Bind to port 22 on ::."},` +
		`{"t_s":1795470063,"nivel":"info","texto":"Ol` + "\ufffd" + ` mundo"},`
	if !strings.HasPrefix(saida, comeco) {
		t.Errorf("veio %s\nqueria começar por %s", saida, comeco)
	}
	if !strings.HasSuffix(saida, `{"t_s":null,"nivel":"info","texto":"duas\nlinhas"}]}`+"\n") {
		t.Errorf("fim: %s", saida)
	}
	rodaAcao(t, "registro", "pipewire.service", "--usuario", "--linhas", "5")
	rodaAcao(t, "registro", "--linhas=500", "cups.socket")
	quer := []string{"-u sshd.service -n 80 -o json --no-pager", "--user -u pipewire.service -n 5 -o json --no-pager",
		"-u cups.socket -n 500 -o json --no-pager"}
	if !reflect.DeepEqual(m.Chamadas("journalctl"), quer) {
		t.Errorf("chamadas %v", m.Chamadas("journalctl"))
	}

	m.Comando("journalctl", dubles.Regra{Contem: ""})
	querAcao(t, 0, `{"linhas":[]}`, "registro", "sshd.service")
	antes := len(m.Chamadas("journalctl"))
	querErroDeAcao(t, 1, "1 a 500", "registro", "sshd.service", "--linhas", "0")
	querErroDeAcao(t, 1, "1 a 500", "registro", "sshd.service", "--linhas", "501")
	querErroDeAcao(t, 1, "1 a 500", "registro", "sshd.service", "--linhas", "-3")
	querErroDeAcao(t, 1, "não é um número", "registro", "sshd.service", "--linhas", "1e2")
	querErroDeAcao(t, 1, "não é um número", "registro", "sshd.service", "--linhas", "muitas")
	querErroDeAcao(t, 1, "--linhas", "registro", "sshd.service", "--linhas")
	querErroDeAcao(t, 1, "não conheço a opção", "registro", "--now")
	querErroDeAcao(t, 1, "não conheço a opção", "registro", "sshd.service", "--file=/etc/shadow")
	querErroDeAcao(t, 1, "não é nome de unidade", "registro", "a.service;rm -rf")
	querErroDeAcao(t, 1, "não é uma unidade", "registro", "sshd")
	querErroDeAcao(t, 1, "uso:", "registro")
	querErroDeAcao(t, 1, "uso:", "registro", "a.service", "b.service")
	if len(m.Chamadas("journalctl")) != antes {
		t.Errorf("chegou ao journalctl: %v", m.Chamadas("journalctl")[antes:])
	}

	m.Comando("journalctl", dubles.Regra{Contem: "", Codigo: 1, Erro: "Failed to open journal: Permission denied\n"})
	querErroDeAcao(t, 1, "Permission denied", "registro", "sshd.service")
	dubles.Nova(t)
	querErroDeAcao(t, 1, "não achei o journalctl", "registro", "sshd.service")
}
