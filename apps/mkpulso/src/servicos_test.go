package main

import (
	"encoding/json"
	"strings"
	"testing"

	"mkpulso/internal/servicos/dubles"
)

// systemctlDeServicos é o dublê com as saídas gravadas do systemctl.
func systemctlDeServicos(t *testing.T) *dubles.Mesa {
	t.Helper()
	dados := "internal/servicos/testdata/"
	m := dubles.Nova(t)
	m.Comando("systemctl",
		dubles.Regra{Contem: " --user list-units ", Arquivo: dados + "usuario-list-units.json"},
		dubles.Regra{Contem: " --user list-unit-files ", Arquivo: dados + "usuario-list-unit-files.json"},
		dubles.Regra{Contem: " --user show --timestamp=unix --property=Id,", Arquivo: dados + "usuario-show.txt"},
		dubles.Regra{Contem: " list-units ", Arquivo: dados + "sistema-list-units.json"},
		dubles.Regra{Contem: " list-unit-files ", Arquivo: dados + "sistema-list-unit-files.json"},
		dubles.Regra{Contem: " show --timestamp=unix --property=Id,", Arquivo: dados + "sistema-show.txt"},
		dubles.Regra{Contem: " show ", Texto: "ControlGroup=\nMainPID=0\n"},
		dubles.Regra{Contem: ""},
	)
	return m
}

func TestServicos_Contrato(t *testing.T) {
	m := systemctlDeServicos(t)
	codigo, saida := rodaAcao(t, "servicos", "--sistema")
	if codigo != 0 || strings.Count(saida, "\n") != 1 {
		t.Fatalf("veio %d %s", codigo, saida)
	}
	// A linha do exemplo do contrato, campo por campo e na ordem.
	if quer := `{"unidade":"sshd.service","escopo":"sistema","descricao":"OpenSSH Daemon","estado":"ativo","sub":"running","habilitado":"sim","pid":812,"mem_kb":5400,"desde_s":1795470000}`; !strings.Contains(saida, quer) {
		t.Errorf("faltou a linha do sshd:\n%s", saida)
	}
	if !strings.HasPrefix(saida, `{"servicos":[{"unidade":"NetworkManager.service",`) {
		t.Errorf("começo: %.80s", saida)
	}
	var resposta struct {
		Servicos []map[string]any `json:"servicos"`
	}
	if err := json.Unmarshal([]byte(saida), &resposta); err != nil {
		t.Fatal(err)
	}
	estados := map[string]bool{"ativo": true, "inativo": true, "falhou": true, "ativando": true, "desativando": true}
	habilitados := map[string]bool{"sim": true, "não": true, "estático": true, "mascarado": true, "gerado": true}
	vistos := map[string]bool{}
	for _, s := range resposta.Servicos {
		if len(s) != 9 {
			t.Errorf("%v: %d campos, o contrato tem 9", s["unidade"], len(s))
		}
		estado, habilitado, unidade := s["estado"].(string), s["habilitado"].(string), s["unidade"].(string)
		if !estados[estado] || !habilitados[habilitado] || s["escopo"] != "sistema" {
			t.Errorf("%s: estado %q, habilitado %q, escopo %v", unidade, estado, habilitado, s["escopo"])
		}
		if !strings.HasSuffix(unidade, ".service") && !strings.HasSuffix(unidade, ".socket") && !strings.HasSuffix(unidade, ".timer") {
			t.Errorf("%s não é .service, .socket nem .timer", unidade)
		}
		vistos[estado], vistos[habilitado] = true, true
	}
	for _, texto := range []string{"ativo", "inativo", "falhou", "ativando", "desativando", "sim", "não", "estático", "mascarado"} {
		if !vistos[texto] {
			t.Errorf("o testdata não exercitou %q", texto)
		}
	}
	if len(resposta.Servicos) != 14 {
		t.Errorf("%d serviços de sistema, queria 14", len(resposta.Servicos))
	}
	for _, c := range m.Chamadas("systemctl") {
		if strings.Contains(c, "--user") {
			t.Errorf("--sistema chamou %q", c)
		}
	}

	conta := func(args ...string) (sistema, usuario int) {
		t.Helper()
		codigo, saida := rodaAcao(t, args...)
		if codigo != 0 {
			t.Fatalf("%v: %d %s", args, codigo, saida)
		}
		return strings.Count(saida, `"escopo":"sistema"`), strings.Count(saida, `"escopo":"usuario"`)
	}
	if s, u := conta("servicos", "--usuario"); s != 0 || u != 5 {
		t.Errorf("--usuario: %d de sistema e %d de usuário", s, u)
	}
	if s, u := conta("servicos"); s != 14 || u != 5 {
		t.Errorf("sem opção: %d de sistema e %d de usuário", s, u)
	}
	_, saida = rodaAcao(t, "servicos", "--usuario")
	if !strings.Contains(saida, `"habilitado":"gerado"`) || !strings.Contains(saida, `{"unidade":"app-claude\\x2ddesktop@autostart.service"`) {
		t.Errorf("usuário: %s", saida)
	}

	querErroDeAcao(t, 1, "uso:", "servicos", "sshd.service")
	querErroDeAcao(t, 1, "não conheço a opção", "servicos", "--tudo")
}

func TestServicos_SemSystemctlOuSemBarramentoEUmaFrase(t *testing.T) {
	m := dubles.Nova(t)
	querErroDeAcao(t, 1, "não achei o systemctl", "servicos")
	m.Comando("systemctl", dubles.Regra{Contem: " --user ", Codigo: 1,
		Erro: "Failed to connect to user scope bus via local transport: $DBUS_SESSION_BUS_ADDRESS and $XDG_RUNTIME_DIR not defined\n"})
	querErroDeAcao(t, 1, "systemd do usuário", "servicos", "--usuario")
	m.Comando("systemctl", dubles.Regra{Contem: "", Texto: "<html>não é json</html>\n"})
	querErroDeAcao(t, 1, "não entendi", "servicos", "--sistema")
}
