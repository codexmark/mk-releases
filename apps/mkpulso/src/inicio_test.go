package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"mkpulso/internal/servicos/dubles"
)

// casaDeInicio monta a home e o /etc falsos com um item de cada origem.
func casaDeInicio(t *testing.T) (home, etc string, m *dubles.Mesa) {
	t.Helper()
	home, etc = t.TempDir(), t.TempDir()
	t.Setenv("MKPULSO_HOME", home)
	t.Setenv("MKPULSO_ETC", etc)
	t.Setenv("XDG_CURRENT_DESKTOP", "Hyprland")
	escreve := func(caminho, conteudo string) {
		if err := os.MkdirAll(filepath.Dir(caminho), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(caminho, []byte(conteudo), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	escreve(filepath.Join(etc, "xdg", "autostart", "foo.desktop"), "[Desktop Entry]\nName=Foo\nExec=foo --tray\n")
	escreve(filepath.Join(home, ".config", "systemd", "user", "default.target.wants", "syncthing.service"), "")
	escreve(filepath.Join(home, ".config", "mk", "hypr", "mare", "autostart.lua"),
		"hl.on(\"hyprland.start\", function()\n    hl.exec_cmd(\"bash '\" .. paths.shell .. \"/scripts/mare-start'\")\nend)\n")
	m = dubles.Nova(t)
	m.Comando("systemctl",
		dubles.Regra{Contem: " --user list-unit-files --state=enabled ", Texto: `[{"unit_file":"syncthing.service","state":"enabled","preset":"enabled"}]` + "\n"},
		dubles.Regra{Contem: " --user show ", Texto: "Id=syncthing.service\nDescription=Syncthing\nExecStart={ path=/usr/bin/syncthing ; argv[]=syncthing ; ignore_errors=no ; start_time=[n/a] ; stop_time=[n/a] ; pid=0 ; code=(null) ; status=0/0 }\n"},
		dubles.Regra{Contem: ""},
	)
	return home, etc, m
}

func TestInicio_Contrato(t *testing.T) {
	home, etc, m := casaDeInicio(t)
	atalho := "desktop:" + filepath.Join(etc, "xdg", "autostart", "foo.desktop")
	// O exemplo do contrato, com o caminho da raiz falsa no lugar de /etc.
	lista := func(ligado string) string {
		return `{"itens":[` +
			`{"id":"` + atalho + `","origem":"atalho","nome":"Foo","comando":"foo --tray","ligado":` + ligado + `,"do_sistema":true,"mudavel":true},` +
			`{"id":"unidade:syncthing.service","origem":"serviço de usuário","nome":"Syncthing","comando":"syncthing","ligado":true,"do_sistema":false,"mudavel":true},` +
			`{"id":"mk:mare","origem":"Mk","nome":"Maré (a ilha e os painéis)","comando":"mare-start","ligado":true,"do_sistema":true,"mudavel":false}]}`
	}
	querAcao(t, 0, lista("true"), "inicio")

	copia := filepath.Join(home, ".config", "autostart", "foo.desktop")
	querAcao(t, 0, `{"ok":true,"id":"`+atalho+`","ligado":false}`, "inicio", "desligar", atalho)
	if b, err := os.ReadFile(copia); err != nil || string(b) != "[Desktop Entry]\nName=Foo\nExec=foo --tray\nHidden=true\n" {
		t.Errorf("a cópia de desligar: %q %v", b, err)
	}
	querAcao(t, 0, lista("false"), "inicio")
	querAcao(t, 0, `{"ok":true,"id":"`+atalho+`","ligado":true}`, "inicio", "ligar", atalho)
	if _, err := os.Stat(copia); !os.IsNotExist(err) {
		t.Errorf("ligar de novo devia apagar a cópia: %v", err)
	}
	querAcao(t, 0, lista("true"), "inicio")

	querAcao(t, 0, `{"ok":true,"id":"unidade:syncthing.service","ligado":false}`, "inicio", "desligar", "unidade:syncthing.service")
	querAcao(t, 0, `{"ok":true,"id":"unidade:syncthing.service","ligado":true}`, "inicio", "ligar", "unidade:syncthing.service")
	var acoes []string
	for _, c := range m.Chamadas("systemctl") {
		if strings.Contains(c, "able ") {
			acoes = append(acoes, c)
		}
	}
	if quer := []string{"--user disable -- syncthing.service", "--user enable -- syncthing.service"}; !reflect.DeepEqual(acoes, quer) {
		t.Errorf("chamadas %v", acoes)
	}

	// O que não é mudável e as entradas hostis: erro, e nada muda.
	antes := len(m.Chamadas("systemctl"))
	querErroDeAcao(t, 1, "faz parte do Mk", "inicio", "desligar", "mk:mare")
	querErroDeAcao(t, 1, "faz parte do Mk", "inicio", "ligar", "mk:mare")
	querErroDeAcao(t, 1, "não é um atalho", "inicio", "desligar", "desktop:../../etc/passwd")
	querErroDeAcao(t, 1, "não é um atalho", "inicio", "desligar", "desktop:"+filepath.Join(etc, "xdg", "autostart")+"/../../../etc/passwd")
	querErroDeAcao(t, 1, "não é um atalho", "inicio", "ligar", "desktop:/etc/passwd")
	querErroDeAcao(t, 1, "não existe", "inicio", "desligar", "desktop:"+filepath.Join(etc, "xdg", "autostart", "sumiu.desktop"))
	querErroDeAcao(t, 1, "não é nome de unidade", "inicio", "desligar", "unidade:../x.service")
	querErroDeAcao(t, 1, "não é nome de unidade", "inicio", "ligar", "unidade:--now")
	querErroDeAcao(t, 1, "não é nome de unidade", "inicio", "ligar", "unidade:a.service;rm -rf")
	querErroDeAcao(t, 1, "sistema", "inicio", "desligar", "unidade:pipewire.socket")
	querErroDeAcao(t, 1, "não conheço o item", "inicio", "ligar", "foo")
	querErroDeAcao(t, 1, "não conheço a opção", "inicio", "ligar", "--now")
	querErroDeAcao(t, 1, "uso:", "inicio", "apagar", atalho)
	querErroDeAcao(t, 1, "uso:", "inicio", "ligar")
	querErroDeAcao(t, 1, "uso:", "inicio", "ligar", atalho, "outro")
	if len(m.Chamadas("systemctl")) != antes {
		t.Errorf("chegou ao systemctl: %v", m.Chamadas("systemctl")[antes:])
	}
	if entradas, _ := os.ReadDir(filepath.Join(home, ".config", "autostart")); len(entradas) != 0 {
		t.Errorf("sobrou coisa em ~/.config/autostart: %v", entradas)
	}
}

func TestInicio_SemNadaEListaVazia(t *testing.T) {
	t.Setenv("MKPULSO_HOME", t.TempDir())
	t.Setenv("MKPULSO_ETC", t.TempDir())
	dubles.Nova(t)
	querAcao(t, 0, `{"itens":[]}`, "inicio")
}
