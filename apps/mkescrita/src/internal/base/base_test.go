package base

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func comEntrada(t *testing.T, texto string) {
	t.Helper()
	antes := Entrada
	Entrada = strings.NewReader(texto)
	t.Cleanup(func() { Entrada = antes })
}

func TestPastasSeguemOXDG(t *testing.T) {
	t.Setenv("HOME", "/casa")
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	if g := Rascunhos(); g != "/casa/.local/state/mk/mkescrita/rascunhos" {
		t.Errorf("rascunhos: %s", g)
	}
	if g := Preferencias(); g != "/casa/.config/mk/mkescrita.json" {
		t.Errorf("preferências: %s", g)
	}
	t.Setenv("XDG_STATE_HOME", "/e")
	t.Setenv("XDG_CONFIG_HOME", "/c")
	if g := Rascunhos(); g != "/e/mk/mkescrita/rascunhos" {
		t.Errorf("rascunhos com XDG: %s", g)
	}
	if g := Preferencias(); g != "/c/mk/mkescrita.json" {
		t.Errorf("preferências com XDG: %s", g)
	}
	// Um XDG relativo não vale (é o que a especificação manda ignorar).
	t.Setenv("XDG_STATE_HOME", "relativo")
	if g := Rascunhos(); g != "/casa/.local/state/mk/mkescrita/rascunhos" {
		t.Errorf("XDG relativo devia ser ignorado: %s", g)
	}
}

func TestLimite(t *testing.T) {
	for valor, quer := range map[string]int64{"": 4 << 20, "1": 1 << 20, "x": 4 << 20, "-3": 4 << 20, "999999": 4 << 20} {
		t.Setenv("MKESCRITA_LIMITE_MB", valor)
		if g := LimiteBytes(); g != quer {
			t.Errorf("MKESCRITA_LIMITE_MB=%q: %d, queria %d", valor, g, quer)
		}
	}
}

func TestEscreveNaoEscapaAcentoNemSinais(t *testing.T) {
	var b bytes.Buffer
	antes := Saida
	Saida = &b
	defer func() { Saida = antes }()
	if c := Falha(Recusado, "não <abro> %s", "isso"); c != Recusado {
		t.Errorf("código %d", c)
	}
	if g := b.String(); g != "{\"erro\":\"não <abro> isso\"}\n" {
		t.Errorf("saída: %q", g)
	}
	b.Reset()
	if c := FalhaDe(fmt.Errorf("embrulhado: %w", Recusa("não"))); c != Recusado {
		t.Errorf("recusa embrulhada devia sair com 2, saiu %d", c)
	}
	if c := FalhaDe(errors.New("quebrou")); c != Erro {
		t.Errorf("erro comum devia sair com 1, saiu %d", c)
	}
}

func TestLeTexto(t *testing.T) {
	comEntrada(t, "olá\n")
	if g, err := LeTexto("5"); err != nil || string(g) != "olá\n" {
		t.Errorf("texto inteiro: %q %v", g, err)
	}
	comEntrada(t, "olá\n")
	if g, err := LeTexto(""); err != nil || string(g) != "olá\n" {
		t.Errorf("sem --bytes: %q %v", g, err)
	}
	comEntrada(t, "ol")
	if _, err := LeTexto("5"); err == nil || !strings.Contains(err.Error(), "cortado") {
		t.Errorf("texto cortado devia falhar: %v", err)
	}
	comEntrada(t, "")
	if _, err := LeTexto("5"); err == nil {
		t.Error("stdin vazio com --bytes 5 devia falhar")
	}
	comEntrada(t, "")
	if g, err := LeTexto("0"); err != nil || len(g) != 0 {
		t.Errorf("texto vazio de propósito: %q %v", g, err)
	}
	comEntrada(t, "x")
	if _, err := LeTexto("um"); err == nil {
		t.Error("--bytes torto devia falhar")
	}
	comEntrada(t, "a\xffb")
	if _, err := LeTexto(""); !FoiRecusa(err) {
		t.Errorf("texto que não é UTF-8 devia ser recusado: %v", err)
	}
	t.Setenv("MKESCRITA_LIMITE_MB", "1")
	comEntrada(t, strings.Repeat("a", 1<<20+1))
	if _, err := LeTexto(""); !FoiRecusa(err) {
		t.Errorf("texto acima do limite devia ser recusado: %v", err)
	}
}

func TestNome(t *testing.T) {
	if Nome("/a/b/notas.md") != "notas.md" || Nome("") != "" {
		t.Error("nome")
	}
}
