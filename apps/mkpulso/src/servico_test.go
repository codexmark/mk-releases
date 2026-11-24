package main

import (
	"reflect"
	"testing"

	"mkpulso/internal/servicos/dubles"
)

func TestServico_Contrato(t *testing.T) {
	verbos := [][2]string{{"iniciar", "start"}, {"parar", "stop"}, {"reiniciar", "restart"}, {"habilitar", "enable"},
		{"desabilitar", "disable"}, {"recarregar", "reload"}}
	for _, v := range verbos {
		m := systemctlDeServicos(t)
		querAcao(t, 0, `{"ok":true,"unidade":"sshd.service","acao":"`+v[0]+`"}`, "servico", v[0], "sshd.service")
		querAcao(t, 0, `{"ok":true,"unidade":"syncthing.service","acao":"`+v[0]+`"}`, "servico", v[0], "syncthing.service", "--usuario")
		if quer := []string{v[1] + " -- sshd.service", "--user " + v[1] + " -- syncthing.service"}; !reflect.DeepEqual(acoesNoSystemctl(m), quer) {
			t.Errorf("%s: chamou %v, queria %v", v[0], acoesNoSystemctl(m), quer)
		}
	}

	m := systemctlDeServicos(t)
	// Recusado por segurança: código 2.
	querErroDeAcao(t, 2, "--mesmo-assim", "servico", "parar", "sddm.service")
	querErroDeAcao(t, 2, "--mesmo-assim", "servico", "desabilitar", "dbus-broker.service")
	querErroDeAcao(t, 2, "--mesmo-assim", "servico", "parar", "systemd-logind.service")
	querErroDeAcao(t, 2, "--mesmo-assim", "servico", "desabilitar", "user@1000.service")
	querErroDeAcao(t, 2, "--mesmo-assim", "servico", "parar", "NetworkManager.service")
	querErroDeAcao(t, 2, "--mesmo-assim", "servico", "parar", "dbus.service", "--usuario")
	// Entrada hostil e erro de uso: código 1.
	querErroDeAcao(t, 1, "não conheço a opção", "servico", "parar", "--now")
	querErroDeAcao(t, 1, "não conheço a opção", "servico", "parar", "sshd.service", "--now")
	querErroDeAcao(t, 1, "não é nome de unidade", "servico", "parar", "a.service;rm -rf")
	querErroDeAcao(t, 1, "não é nome de unidade", "servico", "iniciar", "a b.service")
	querErroDeAcao(t, 1, "não é nome de unidade", "servico", "iniciar", "-f.service")
	querErroDeAcao(t, 1, "não é nome de unidade", "servico", "iniciar", "../x.service")
	querErroDeAcao(t, 1, "não é uma unidade", "servico", "iniciar", "sshd")
	querErroDeAcao(t, 1, "não é uma unidade", "servico", "parar", "home.mount")
	querErroDeAcao(t, 1, "não conheço a ação", "servico", "kill", "sshd.service")
	querErroDeAcao(t, 1, "uso:", "servico", "parar")
	querErroDeAcao(t, 1, "uso:", "servico")
	if c := m.Chamadas("systemctl"); len(c) != 0 {
		t.Errorf("chegou ao systemctl: %v", c)
	}
	querAcao(t, 0, `{"ok":true,"unidade":"sddm.service","acao":"parar"}`, "servico", "parar", "sddm.service", "--mesmo-assim")
	if quer := []string{"stop -- sddm.service"}; !reflect.DeepEqual(m.Chamadas("systemctl"), quer) {
		t.Errorf("--mesmo-assim: %v", m.Chamadas("systemctl"))
	}
}

func TestServico_ErroDoSystemctlNaFrase(t *testing.T) {
	m := dubles.Nova(t)
	querErroDeAcao(t, 1, "não achei o systemctl", "servico", "iniciar", "sshd.service")
	m.Comando("systemctl", dubles.Regra{Contem: "", Codigo: 1,
		Erro: "Failed to start cups.service: Access denied\nSee system logs and 'systemctl status cups.service' for details.\n"})
	querAcao(t, 1, `{"erro":"não deu para iniciar cups.service: Failed to start cups.service: Access denied"}`, "servico", "iniciar", "cups.service")
}
