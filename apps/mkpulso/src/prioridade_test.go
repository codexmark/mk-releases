package main

import (
	"fmt"
	"testing"

	"mkpulso/internal/servicos/dubles"
)

func TestPrioridade_Contrato(t *testing.T) {
	dubles.Nova(t)
	procDeAcao(t, map[int]string{1234: "firefox", 600: "Hyprland"}, 1, 500)
	feitos := semSinaisDeAcao(t)

	querAcao(t, 0, `{"ok":true,"pid":1234,"nice":10}`, "prioridade", "1234", "10")
	querAcao(t, 0, `{"ok":true,"pid":1234,"nice":19}`, "prioridade", "1234", "19")
	querAcao(t, 0, `{"ok":true,"pid":1234,"nice":0}`, "prioridade", "1234", "0")
	querAcao(t, 0, `{"ok":true,"pid":600,"nice":5}`, "prioridade", "600", "5", "--mesmo-assim")
	// O sistema recusa subir a prioridade: a frase do contrato, código 1.
	querAcao(t, 1, `{"erro":"só dá para baixar a prioridade; para subir é preciso ser administrador"}`, "prioridade", "1234", "-5")
	querAcao(t, 1, `{"erro":"só dá para baixar a prioridade; para subir é preciso ser administrador"}`, "prioridade", "1234", "-20")
	if quer := []string{"nice 10 ao 1234", "nice 19 ao 1234", "nice 0 ao 1234", "nice 5 ao 600", "nice -5 ao 1234", "nice -20 ao 1234"}; fmt.Sprint(*feitos) != fmt.Sprint(quer) {
		t.Errorf("pedidos: %v, queria %v", *feitos, quer)
	}
	*feitos = nil

	querErroDeAcao(t, 2, "processo 1", "prioridade", "1", "5")
	querErroDeAcao(t, 2, "não é um processo", "prioridade", "-1", "5")
	querErroDeAcao(t, 2, "outro usuário", "prioridade", "500", "5")
	querErroDeAcao(t, 2, "sessão", "prioridade", "600", "5")
	querErroDeAcao(t, 1, "-20 a 19", "prioridade", "1234", "99")
	querErroDeAcao(t, 1, "-20 a 19", "prioridade", "1234", "20")
	querErroDeAcao(t, 1, "-20 a 19", "prioridade", "1234", "-21")
	querErroDeAcao(t, 1, "não é um número", "prioridade", "1234", "alta")
	querErroDeAcao(t, 1, "não é um número", "prioridade", "1e9", "5")
	querErroDeAcao(t, 1, "já saiu", "prioridade", "4242", "5")
	querErroDeAcao(t, 1, "uso:", "prioridade", "1234")
	if len(*feitos) != 0 {
		t.Errorf("mudou prioridade apesar da recusa: %v", *feitos)
	}
}
