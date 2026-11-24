package base

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"mkcorte/internal/dubles"
)

func TestHomeCacheETil(t *testing.T) {
	t.Setenv("MKCORTE_HOME", "/tmp/falso/casa")
	t.Setenv("XDG_CACHE_HOME", "")
	casos := map[string]string{
		Home("Vídeos", "a.mp4"):         "/tmp/falso/casa/Vídeos/a.mp4",
		Cache():                         "/tmp/falso/casa/.cache/mk/mkcorte",
		Cache("abc", "q01.jpg"):         "/tmp/falso/casa/.cache/mk/mkcorte/abc/q01.jpg",
		Til("/tmp/falso/casa"):          "~",
		Til("/tmp/falso/casa/Vídeos/a"): "~/Vídeos/a",
		Til("/tmp/falso/casamento/a"):   "/tmp/falso/casamento/a",
		Til("/etc/hosts"):               "/etc/hosts",
	}
	for obtido, esperado := range casos {
		if obtido != esperado {
			t.Errorf("esperava %q, veio %q", esperado, obtido)
		}
	}
	t.Setenv("XDG_CACHE_HOME", "/tmp/falso/cache")
	if got := Cache("x"); got != "/tmp/falso/cache/mk/mkcorte/x" {
		t.Errorf("com XDG_CACHE_HOME: %q", got)
	}
	// Um XDG_CACHE_HOME relativo não vale: cairia na pasta de onde o motor rodou.
	t.Setenv("XDG_CACHE_HOME", "cache")
	if got := Cache(); got != "/tmp/falso/casa/.cache/mk/mkcorte" {
		t.Errorf("XDG_CACHE_HOME relativo: %q", got)
	}
}

func TestEscreveUmaLinhaPorValorSemEscaparAcentos(t *testing.T) {
	var b bytes.Buffer
	antes := Saida
	Saida = &b
	defer func() { Saida = antes }()
	_ = Escreve(map[string]string{"nome": "férias <1> & cia"})
	if got := b.String(); got != "{\"nome\":\"férias <1> & cia\"}\n" {
		t.Errorf("veio %q", got)
	}
	b.Reset()
	if c := Falha(Erro, "não achei %s", "a.mp4"); c != 1 || b.String() != "{\"erro\":\"não achei a.mp4\"}\n" {
		t.Errorf("Falha: %d %q", c, b.String())
	}
	b.Reset()
	if c := FalhaDe(Recusa("não dá")); c != Recusado || !strings.Contains(b.String(), "não dá") {
		t.Errorf("recusa: %d %q", c, b.String())
	}
	if c := FalhaDe(errors.New("quebrou")); c != Erro {
		t.Errorf("erro comum: %d", c)
	}
	if !FoiRecusa(Recusa("x")) || FoiRecusa(errors.New("x")) {
		t.Error("FoiRecusa")
	}
}

func TestRoda_DevolveASaidaEFazDoErroUmaFrase(t *testing.T) {
	m := dubles.Nova(t)
	m.Responde("certo", "olá", 0)
	out, err := Roda("certo", time.Second, "-a", "b c")
	if err != nil || string(out) != "olá" {
		t.Errorf("veio %q, %v", out, err)
	}
	if c := m.Chamadas("certo"); len(c) != 1 || len(c[0]) != 2 || c[0][1] != "b c" {
		t.Errorf("chamadas: %q", c)
	}
	if !Tem("certo") || Tem("nao-existe") {
		t.Error("Tem")
	}
	m.Comando("errado", `printf 'linha um\n  o   que deu   errado  \n\n' >&2; exit 3`)
	if _, err := Roda("errado", time.Second); err == nil || err.Error() != "o que deu errado" {
		t.Errorf("erro: %v", err)
	}
	m.Responde("mudo", "", 4)
	if _, err := Roda("mudo", time.Second); err == nil || !strings.Contains(err.Error(), "código 4") {
		t.Errorf("mudo: %v", err)
	}
	if _, err := Roda("nao-existe", time.Second); err == nil || !strings.Contains(err.Error(), "não achei o nao-existe") {
		t.Errorf("ausente: %v", err)
	}
	m.Comando("lento", `while :; do :; done`)
	if _, err := Roda("lento", 150*time.Millisecond); err == nil || !strings.Contains(err.Error(), "demorou demais") {
		t.Errorf("lento: %v", err)
	}
}

func TestResumo(t *testing.T) {
	if got := Resumo("a\n\n  b   c \n\n"); got != "b c" {
		t.Errorf("veio %q", got)
	}
	if got := Resumo(strings.Repeat("é", 400)); len([]rune(got)) != 301 || !strings.HasSuffix(got, "…") {
		t.Errorf("longo: %d", len([]rune(got)))
	}
	if Resumo("\n \n") != "" {
		t.Error("vazio")
	}
}
