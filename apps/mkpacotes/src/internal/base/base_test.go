package base

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

var variaveis = []string{"MKPACOTES_PACMAN_DB", "MKPACOTES_LOG", "MKPACOTES_MK", "MKPACOTES_CACHE", "MKPACOTES_AUR", "MKPACOTES_TRAVA"}

func TestRaizesPadrao(t *testing.T) {
	for _, v := range variaveis {
		t.Setenv(v, "")
	}
	casos := map[string]string{
		PacmanDB():                       "/var/lib/pacman",
		PacmanDB("local"):                "/var/lib/pacman/local",
		PacmanDB("db.lck"):               "/var/lib/pacman/db.lck",
		PacmanDB("a", "..", "b"):         "/var/lib/pacman/b",
		Mk("system", "packages", "base"): "/usr/share/mk/system/packages/base",
		Log():                            "/var/log/pacman.log",
		Cache():                          "/var/cache/pacman/pkg",
		AUR():                            "https://aur.archlinux.org",
	}
	for obtido, esperado := range casos {
		if obtido != esperado {
			t.Errorf("esperava %q, veio %q", esperado, obtido)
		}
	}
}

func TestRaizesTrocadasPelasVariaveis(t *testing.T) {
	t.Setenv("MKPACOTES_PACMAN_DB", "/tmp/falso/pacman")
	t.Setenv("MKPACOTES_LOG", "/tmp/falso/pacman.log")
	t.Setenv("MKPACOTES_MK", "/tmp/falso/mk")
	t.Setenv("MKPACOTES_CACHE", "/tmp/falso/cache")
	t.Setenv("MKPACOTES_AUR", "http://127.0.0.1:9/")
	t.Setenv("MKPACOTES_TRAVA", "/tmp/falso/trava")
	casos := map[string]string{
		PacmanDB("local", "x"): "/tmp/falso/pacman/local/x",
		Log():                  "/tmp/falso/pacman.log",
		Mk("bin", "mk-update"): "/tmp/falso/mk/bin/mk-update",
		Cache():                "/tmp/falso/cache",
		AUR():                  "http://127.0.0.1:9",
		Trava():                "/tmp/falso/trava",
	}
	for obtido, esperado := range casos {
		if obtido != esperado {
			t.Errorf("esperava %q, veio %q", esperado, obtido)
		}
	}
}

func TestTravaFicaNaPastaDaSessaoOuNaTemporariaComOUid(t *testing.T) {
	t.Setenv("MKPACOTES_TRAVA", "")
	t.Setenv("XDG_RUNTIME_DIR", "/run/user/1000")
	if got := Trava(); got != "/run/user/1000/mkpacotes.trava" {
		t.Errorf("Trava: %q", got)
	}
	t.Setenv("XDG_RUNTIME_DIR", "")
	t.Setenv("TMPDIR", "/tmp/x")
	if got := Trava(); !strings.HasPrefix(got, "/tmp/x/mkpacotes-") || !strings.HasSuffix(got, ".trava") {
		t.Errorf("Trava: %q", got)
	}
}

func TestConfigSegueOXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/cfg")
	if got := Config("mk", "ajustes.json"); got != "/tmp/cfg/mk/ajustes.json" {
		t.Errorf("Config: %q", got)
	}
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "/home/alguem")
	if got := Config("mk"); got != "/home/alguem/.config/mk" {
		t.Errorf("Config: %q", got)
	}
}

// troca a saída por um buffer durante o teste.
func captura(t *testing.T) *bytes.Buffer {
	t.Helper()
	antes := Saida
	b := &bytes.Buffer{}
	Saida = b
	t.Cleanup(func() { Saida = antes })
	return b
}

func TestEscreveUmaLinhaSoSemEscaparAcentosNemSinais(t *testing.T) {
	b := captura(t)
	if err := Escreve(map[string]any{"nome": "ação <rápida> & \"boa\"", "n": nil}); err != nil {
		t.Fatal(err)
	}
	saida := b.String()
	if strings.Count(saida, "\n") != 1 || !strings.HasSuffix(saida, "\n") {
		t.Errorf("devia ser uma linha só, terminada em quebra: %q", saida)
	}
	for _, pedaco := range []string{"ação", "<rápida>", "&", `"n":null`} {
		if !strings.Contains(saida, pedaco) {
			t.Errorf("faltou %q em %q", pedaco, saida)
		}
	}
	var volta map[string]any
	if err := json.Unmarshal(b.Bytes(), &volta); err != nil {
		t.Fatalf("não é JSON: %v", err)
	}
}

func TestEscreveTextoComQuebraDeLinhaContinuaNumaLinha(t *testing.T) {
	b := captura(t)
	_ = Escreve(map[string]string{"linha": "uma\nduas"})
	if strings.Count(b.String(), "\n") != 1 {
		t.Errorf("a quebra de dentro do texto tem de sair escapada: %q", b.String())
	}
}

func TestFalhaEscreveOErroEDevolveOCodigo(t *testing.T) {
	b := captura(t)
	if c := Falha(Recusado, "não removo o %s", "glibc"); c != Recusado {
		t.Errorf("código %d", c)
	}
	var r map[string]string
	if err := json.Unmarshal(b.Bytes(), &r); err != nil {
		t.Fatalf("não é JSON: %v", err)
	}
	if r["erro"] != "não removo o glibc" || len(r) != 1 {
		t.Errorf("resposta %v", r)
	}
}

func TestFalhaComPorcentagemNoTextoNaoViraFormato(t *testing.T) {
	b := captura(t)
	Falha(Erro, "%s", "disco 100% cheio")
	if !strings.Contains(b.String(), "disco 100% cheio") {
		t.Errorf("veio %q", b.String())
	}
}

func TestCodigosDoContrato(t *testing.T) {
	if Certo != 0 || Erro != 1 || Recusado != 2 {
		t.Errorf("os códigos de saída são do contrato: 0, 1 e 2")
	}
}
