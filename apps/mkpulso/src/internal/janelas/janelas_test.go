package janelas

import (
	"testing"
	"time"

	"mkpulso/internal/coleta/falso"
)

const clientes = `[{
    "address": "0x55d0",
    "mapped": true,
    "hidden": false,
    "at": [55, 62],
    "workspace": { "id": 2, "name": "2" },
    "class": "firefox",
    "title": "Mk — Firefox",
    "pid": 1234,
    "xwayland": false
},{
    "address": "0x55e0",
    "mapped": true,
    "workspace": { "id": -98, "name": "special:rascunho" },
    "class": "kitty",
    "title": "",
    "pid": 2000
},{
    "address": "0x55f0",
    "mapped": false,
    "workspace": { "id": 1, "name": "1" },
    "class": "fantasma",
    "title": "não mapeada",
    "pid": 2001
},{
    "address": "0x5600",
    "mapped": true,
    "workspace": { "id": 1, "name": "1" },
    "class": "sem-dono",
    "title": "sem pid",
    "pid": -1
}]`

// dubla põe um hyprctl de mentira que responde a cada subcomando.
func dubla(t *testing.T, r, clients, monitors, version string) {
	t.Helper()
	falso.Escreve(t, r, "resp/clients", clients)
	falso.Escreve(t, r, "resp/monitors", monitors)
	falso.Escreve(t, r, "resp/version", version)
	falso.Comando(t, r, "hyprctl", `[ "$2" = "-j" ] || exit 9
case "$1" in clients|monitors|version) exec /bin/cat "`+r+`/resp/$1" ;; esac
exit 1`)
}

func TestListarJanelas(t *testing.T) {
	r := falso.Raizes(t)
	dubla(t, r, clientes, "[]", "{}")
	js := Listar()
	if len(js) != 2 {
		t.Fatalf("esperava as duas janelas mapeadas e com dono, vieram %+v", js)
	}
	if js[0].Titulo != "Mk — Firefox" || js[0].Workspace != "2" || js[0].Endereco != "0x55d0" || js[0].PID != 1234 || js[0].Classe != "firefox" {
		t.Errorf("janela = %+v", js[0])
	}
	if js[1].Workspace != "special:rascunho" || js[1].PID != 2000 {
		t.Errorf("janela = %+v", js[1])
	}
}

func TestSemHyprctlOuComEleQuebrado(t *testing.T) {
	r := falso.Raizes(t)
	vazio := func(caso string) {
		t.Helper()
		if js := Listar(); js == nil || len(js) != 0 {
			t.Errorf("%s: janelas deveria ser uma lista vazia, veio %#v", caso, js)
		}
		if ts := Telas(); ts == nil || len(ts) != 0 {
			t.Errorf("%s: telas deveria ser uma lista vazia, veio %#v", caso, ts)
		}
		if c := Compositor(); c != nil {
			t.Errorf("%s: compositor deveria ser null, veio %q", caso, *c)
		}
	}
	vazio("sem hyprctl no PATH")
	falso.Comando(t, r, "hyprctl", "echo 'HYPRLAND_INSTANCE_SIGNATURE not set!' >&2\nexit 1")
	vazio("hyprctl que falha")
	falso.Comando(t, r, "hyprctl", `echo '[{"address": "0x1", "pid": 5, "mapped": true}]'; exit 3`)
	vazio("hyprctl que responde mas sai com erro")
	for _, lixo := range []string{"", "isto não é json", `{"um": "objeto"}`, `[{"address": "0x1", "pid": "texto"`, `[1, 2, 3]`, "null"} {
		dubla(t, r, lixo, lixo, lixo)
		vazio("lixo " + lixo)
	}
}

func TestHyprctlTravadoNaoTravaORetrato(t *testing.T) {
	r := falso.Raizes(t)
	falso.Comando(t, r, "hyprctl", "exec /bin/sleep 20")
	velho := LimiteDeTempo
	LimiteDeTempo = 80 * time.Millisecond
	t.Cleanup(func() { LimiteDeTempo = velho })
	inicio := time.Now()
	if js := Listar(); len(js) != 0 {
		t.Errorf("hyprctl travado: sem janelas, veio %+v", js)
	}
	if levou := time.Since(inicio); levou > 2*time.Second {
		t.Errorf("deveria desistir no limite de tempo, levou %v", levou)
	}
}

func TestLimiteDeTempoPadrao(t *testing.T) {
	if LimiteDeTempo != time.Second {
		t.Errorf("o limite do hyprctl é de 1 s, está em %v", LimiteDeTempo)
	}
}

func TestTelas(t *testing.T) {
	r := falso.Raizes(t)
	dubla(t, r, "[]", `[{"id": 0, "name": "eDP-1", "description": "BOE 0x0697", "width": 1366, "height": 768,
		"refreshRate": 60.01200, "scale": 1, "disabled": false},
		{"id": 1, "name": "DP-1", "description": "Dell U2720Q", "width": 3840, "height": 2160, "refreshRate": 59.997, "scale": 1.5}]`, "{}")
	ts := Telas()
	if len(ts) != 2 {
		t.Fatalf("telas = %+v", ts)
	}
	if ts[0].Nome != "eDP-1" || ts[0].Descricao != "BOE 0x0697" || ts[0].Largura != 1366 || ts[0].Altura != 768 || ts[0].Hz != 60.01 || ts[0].Escala != 1 {
		t.Errorf("tela = %+v", ts[0])
	}
	if ts[1].Hz != 60 || ts[1].Escala != 1.5 {
		t.Errorf("tela = %+v", ts[1])
	}
}

func TestCompositor(t *testing.T) {
	r := falso.Raizes(t)
	dubla(t, r, "[]", "[]", `{"branch": "main", "commit": "11cbc6c5", "version": "0.57.2", "tag": "v0.57.2", "dirty": false}`)
	if c := Compositor(); c == nil || *c != "Hyprland 0.57.2" {
		t.Errorf("compositor = %v", c)
	}
	dubla(t, r, "[]", "[]", `{"tag": "v0.41.0-12-gabc"}`)
	if c := Compositor(); c == nil || *c != "Hyprland 0.41.0-12-gabc" {
		t.Errorf("compositor só com tag = %v", c)
	}
	dubla(t, r, "[]", "[]", `{"commit": "abc"}`)
	if c := Compositor(); c != nil {
		t.Errorf("resposta sem versão não diz nada: null, veio %q", *c)
	}
}
