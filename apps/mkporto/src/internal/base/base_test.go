package base

import (
	"bytes"
	"path/filepath"
	"testing"
)

func TestPastaDoSsh_PadraoETroca(t *testing.T) {
	casa := t.TempDir()
	t.Setenv("HOME", casa)
	t.Setenv("MKPORTO_SSH_DIR", "")
	if d := PastaDoSsh(); d != filepath.Join(casa, ".ssh") {
		t.Errorf("padrão: %q", d)
	}
	t.Setenv("MKPORTO_SSH_DIR", "/outra/pasta/")
	if d := PastaDoSsh(); d != "/outra/pasta" {
		t.Errorf("trocada: %q", d)
	}
}

func TestTil_AbreviaAHomeESoEla(t *testing.T) {
	t.Setenv("HOME", "/casa/ana")
	for caminho, quer := range map[string]string{
		"/casa/ana":              "~",
		"/casa/ana/.ssh/config":  "~/.ssh/config",
		"/casa/anabela/.ssh":     "/casa/anabela/.ssh",
		"/etc/ssh":               "/etc/ssh",
		"":                       "",
		"/casa/ana/.ssh/a b.pub": "~/.ssh/a b.pub",
	} {
		if obtido := Til(caminho); obtido != quer {
			t.Errorf("Til(%q) = %q; esperado %q", caminho, obtido, quer)
		}
	}
}

func TestComando_PadraoETroca(t *testing.T) {
	t.Setenv("MKPORTO_SSH", "")
	if c := Comando("MKPORTO_SSH", "ssh"); c != "ssh" {
		t.Errorf("padrão: %q", c)
	}
	t.Setenv("MKPORTO_SSH", "/opt/ssh-de-teste")
	if c := Comando("MKPORTO_SSH", "ssh"); c != "/opt/ssh-de-teste" {
		t.Errorf("trocado: %q", c)
	}
}

func TestEscreveEFalha_UmaLinhaDeJSONSemEscaparAcento(t *testing.T) {
	var b bytes.Buffer
	antiga := Saida
	Saida = &b
	defer func() { Saida = antiga }()

	if err := Escreve(map[string]any{"a": "ação <b>"}); err != nil {
		t.Fatal(err)
	}
	if b.String() != "{\"a\":\"ação <b>\"}\n" {
		t.Errorf("Escreve: %q", b.String())
	}
	b.Reset()
	if c := Falha(Recusado, "não mexo em %s", "x"); c != 2 {
		t.Errorf("código: %d", c)
	}
	if b.String() != "{\"erro\":\"não mexo em x\"}\n" {
		t.Errorf("Falha: %q", b.String())
	}
	b.Reset()
	if c := FalhaDeCampo("porta", "a porta vai de 1 a 65535"); c != 1 {
		t.Errorf("código: %d", c)
	}
	if b.String() != "{\"campo\":\"porta\",\"erro\":\"a porta vai de 1 a 65535\"}\n" {
		t.Errorf("FalhaDeCampo: %q", b.String())
	}
}
