package base

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func casa(t *testing.T) string {
	t.Helper()
	c := t.TempDir()
	t.Setenv("HOME", c)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_RUNTIME_DIR", "")
	t.Setenv("MKSOM_CONFIG", "")
	t.Setenv("MKSOM_UNIDADES", "")
	return c
}

func TestPastas_PadraoEVariavel(t *testing.T) {
	c := casa(t)
	if got := PastaDoApp(); got != filepath.Join(c, ".config", "mk", "mksom") {
		t.Errorf("pasta do app: %s", got)
	}
	if got := PastaDasUnidades(); got != filepath.Join(c, ".config", "systemd", "user") {
		t.Errorf("unidades: %s", got)
	}
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(c, "cfg"))
	if got := PastaDoApp(); got != filepath.Join(c, "cfg", "mk", "mksom") {
		t.Errorf("com XDG_CONFIG_HOME: %s", got)
	}
	t.Setenv("MKSOM_CONFIG", filepath.Join(c, "a"))
	t.Setenv("MKSOM_UNIDADES", filepath.Join(c, "u"))
	if PastaDoApp() != filepath.Join(c, "a") || PastaDasUnidades() != filepath.Join(c, "u") {
		t.Errorf("as variáveis não valeram: %s %s", PastaDoApp(), PastaDasUnidades())
	}
}

func TestPastaTemporaria_SoNaMemoria(t *testing.T) {
	c := casa(t)
	if _, err := PastaTemporaria(); err == nil {
		t.Error("sem XDG_RUNTIME_DIR tinha de recusar: a gravação não vai para disco")
	}
	t.Setenv("XDG_RUNTIME_DIR", "relativo")
	if _, err := PastaTemporaria(); err == nil {
		t.Error("um caminho relativo tinha de ser recusado")
	}
	t.Setenv("XDG_RUNTIME_DIR", filepath.Join(c, "run"))
	p, err := PastaTemporaria()
	if err != nil || p != filepath.Join(c, "run", "mksom") {
		t.Errorf("%s %v", p, err)
	}
}

func TestComando(t *testing.T) {
	t.Setenv("MKSOM_WPCTL", "")
	if Comando("MKSOM_WPCTL", "wpctl") != "wpctl" {
		t.Error("padrão")
	}
	t.Setenv("MKSOM_WPCTL", "/x/wpctl")
	if Comando("MKSOM_WPCTL", "wpctl") != "/x/wpctl" {
		t.Error("variável")
	}
}

func TestEscreveAtomico_ModoPastaESemSobra(t *testing.T) {
	c := casa(t)
	alvo := filepath.Join(c, "a", "b", "estado.json")
	if err := EscreveAtomico(alvo, []byte("um")); err != nil {
		t.Fatal(err)
	}
	if err := EscreveAtomico(alvo, []byte("dois")); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(alvo)
	if string(b) != "dois" {
		t.Errorf("conteúdo: %q", b)
	}
	if i, _ := os.Stat(alvo); i.Mode().Perm() != 0o600 {
		t.Errorf("modo do arquivo: %o", i.Mode().Perm())
	}
	if i, _ := os.Stat(filepath.Dir(alvo)); i.Mode().Perm() != 0o700 {
		t.Errorf("modo da pasta: %o", i.Mode().Perm())
	}
	sobras, _ := os.ReadDir(filepath.Dir(alvo))
	if len(sobras) != 1 {
		t.Errorf("sobrou temporário: %v", sobras)
	}
}

func TestEscreveAtomico_NuncaNaPastaDoPipewire(t *testing.T) {
	c := casa(t)
	for _, alvo := range []string{
		filepath.Join(c, ".config", "pipewire", "pipewire.conf.d", "eq.conf"),
		filepath.Join(c, ".config", "pipewire", "eq.conf"),
		filepath.Join(c, ".config", "wireplumber", "x.conf"),
	} {
		err := EscreveAtomico(alvo, []byte("x"))
		if err == nil || !strings.Contains(err.Error(), "PipeWire") {
			t.Errorf("%s: tinha de recusar, veio %v", alvo, err)
		}
		if _, e := os.Stat(filepath.Join(c, ".config")); e == nil {
			t.Errorf("%s: criou pasta", alvo)
		}
	}
	// E com a variável apontando para lá também.
	t.Setenv("MKSOM_CONFIG", filepath.Join(c, ".config", "pipewire"))
	if err := EscreveAtomico(filepath.Join(PastaDoApp(), "eq.conf"), []byte("x")); err == nil {
		t.Error("MKSOM_CONFIG dentro de ~/.config/pipewire tinha de ser recusado")
	}
}

func TestEscreveEFalha_UmaLinhaDeJSONSemEscaparAcento(t *testing.T) {
	var b bytes.Buffer
	antes := Saida
	Saida = &b
	defer func() { Saida = antes }()
	if err := Escreve(map[string]any{"a": "não <b>"}); err != nil {
		t.Fatal(err)
	}
	if c := Falha(Recusado, "não deu: %d", 3); c != 2 {
		t.Errorf("código %d", c)
	}
	if got := b.String(); got != "{\"a\":\"não <b>\"}\n{\"erro\":\"não deu: 3\"}\n" {
		t.Errorf("%q", got)
	}
}
