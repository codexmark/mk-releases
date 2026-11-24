package base

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"mkutil/internal/dubles"
)

func TestPastasDaPessoaSaemDeMKUTIL_HOME(t *testing.T) {
	t.Setenv("MKUTIL_HOME", "/tmp/falso/casa")
	casos := map[string]string{
		Home(".config", "mk"):           "/tmp/falso/casa/.config/mk",
		Aplicativos():                   "/tmp/falso/casa/Aplicativos",
		Aplicativos("krita.AppImage"):   "/tmp/falso/casa/Aplicativos/krita.AppImage",
		Icones("krita.png"):             "/tmp/falso/casa/Aplicativos/.icones/krita.png",
		Atalhos("mkinstalar-a.desktop"): "/tmp/falso/casa/.local/share/applications/mkinstalar-a.desktop",
		Registro():                      "/tmp/falso/casa/.local/state/mk/instalados.json",
	}
	for obtido, esperado := range casos {
		if obtido != esperado {
			t.Errorf("esperava %q, veio %q", esperado, obtido)
		}
	}
}

func TestHomeSemVariavelEADeQuemRoda(t *testing.T) {
	t.Setenv("MKUTIL_HOME", "")
	t.Setenv("HOME", "/home/alguem")
	if got := Home(".config"); got != "/home/alguem/.config" {
		t.Errorf("Home: %q", got)
	}
}

func TestTilEscreveAHomeComoAPessoaLe(t *testing.T) {
	t.Setenv("MKUTIL_HOME", "/home/ana")
	casos := map[string]string{
		"/home/ana":                 "~",
		"/home/ana/Aplicativos/a b": "~/Aplicativos/a b",
		"/home/anabela/x":           "/home/anabela/x",
		"/opt/x":                    "/opt/x",
	}
	for entrada, esperado := range casos {
		if got := Til(entrada); got != esperado {
			t.Errorf("Til(%q) = %q, queria %q", entrada, got, esperado)
		}
	}
}

func TestLimitesTemPadraoEVemDoAmbiente(t *testing.T) {
	t.Setenv("MKUTIL_LIMITE_MB", "")
	t.Setenv("MKUTIL_LIMITE_ENTRADAS", "")
	if LimiteBytes() != 8192<<20 || LimiteEntradas() != 200000 {
		t.Errorf("padrões: %d bytes, %d entradas", LimiteBytes(), LimiteEntradas())
	}
	t.Setenv("MKUTIL_LIMITE_MB", "3")
	t.Setenv("MKUTIL_LIMITE_ENTRADAS", "10")
	if LimiteBytes() != 3<<20 || LimiteEntradas() != 10 {
		t.Errorf("do ambiente: %d bytes, %d entradas", LimiteBytes(), LimiteEntradas())
	}
	// Lixo, zero e negativo não desligam o limite: vale o padrão.
	for _, v := range []string{"abc", "0", "-5", "1e9"} {
		t.Setenv("MKUTIL_LIMITE_MB", v)
		t.Setenv("MKUTIL_LIMITE_ENTRADAS", v)
		if LimiteBytes() != 8192<<20 || LimiteEntradas() != 200000 {
			t.Errorf("%q devia cair no padrão", v)
		}
	}
}

func TestLibsDoSistema(t *testing.T) {
	t.Setenv("MKUTIL_LIBS", "")
	if got := strings.Join(LibsDoSistema(), ":"); got != "/usr/lib:/usr/lib64:/lib" {
		t.Errorf("padrão: %q", got)
	}
	t.Setenv("MKUTIL_LIBS", "/a:/b")
	if got := LibsDoSistema(); len(got) != 2 || got[0] != "/a" || got[1] != "/b" {
		t.Errorf("do ambiente: %v", got)
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
	if err := Escreve(map[string]any{"nome": "ação <rápida> & \"boa\"\nduas", "n": nil}); err != nil {
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

func TestFalhaEscreveOErroEDevolveOCodigo(t *testing.T) {
	b := captura(t)
	if c := Falha(Recusado, "não extraio %d", 1); c != Recusado {
		t.Errorf("código %d", c)
	}
	var r map[string]string
	if err := json.Unmarshal(b.Bytes(), &r); err != nil {
		t.Fatalf("não é JSON: %v", err)
	}
	if r["erro"] != "não extraio 1" || len(r) != 1 {
		t.Errorf("resposta %v", r)
	}
}

func TestFalhaDeDaCodigo2ParaRecusaE1ParaOResto(t *testing.T) {
	b := captura(t)
	if c := FalhaDe(Recusa("o caminho %q sai da pasta", "../x")); c != Recusado {
		t.Errorf("recusa: código %d", c)
	}
	if c := FalhaDe(fmt.Errorf("lendo: %w", Recusa("bomba"))); c != Recusado {
		t.Errorf("recusa embrulhada: código %d", c)
	}
	if c := FalhaDe(errors.New("disco 100% cheio")); c != Erro {
		t.Errorf("erro comum: código %d", c)
	}
	if !strings.Contains(b.String(), `sai da pasta`) || !strings.Contains(b.String(), "disco 100% cheio") {
		t.Errorf("veio %q", b.String())
	}
	if FoiRecusa(errors.New("x")) || FoiRecusa(nil) {
		t.Error("erro comum não é recusa")
	}
}

func TestCodigosDoContrato(t *testing.T) {
	if Certo != 0 || Erro != 1 || Recusado != 2 {
		t.Errorf("os códigos de saída são do contrato: 0, 1 e 2")
	}
}

func TestRodaPassaOsArgumentosComoListaEDevolveOStdout(t *testing.T) {
	m := dubles.Nova(t)
	m.Comando("eco", `printf '%s|' "$@"`)
	saida, err := Roda("eco", time.Second, nil, "--", "-nome com espaço", `aspas "duplas" e 'simples'`, "$(rm -rf x); `y`")
	if err != nil {
		t.Fatal(err)
	}
	if quer := "--|-nome com espaço|aspas \"duplas\" e 'simples'|$(rm -rf x); `y`|"; string(saida) != quer {
		t.Errorf("veio %q", saida)
	}
	if c := m.Chamadas("eco"); len(c) != 1 || len(c[0]) != 4 || c[0][1] != "-nome com espaço" {
		t.Errorf("chamadas: %q", c)
	}
}

func TestRodaErrosViramFrase(t *testing.T) {
	m := dubles.Nova(t)
	if _, err := Roda("nada", time.Second, nil); err == nil || !strings.Contains(err.Error(), "não achei o nada") {
		t.Errorf("comando ausente: %v", err)
	}
	m.Comando("ruim", "printf 'aviso qualquer\\n  o   que deu errado  \\n\\n' >&2; exit 3")
	if _, err := Roda("ruim", time.Second, nil); err == nil || err.Error() != "o que deu errado" {
		t.Errorf("stderr: %v", err)
	}
	m.Comando("mudo", "exit 4")
	if _, err := Roda("mudo", time.Second, nil); err == nil || !strings.Contains(err.Error(), "código 4") {
		t.Errorf("sem stderr: %v", err)
	}
	m.Comando("lento", "while :; do :; done")
	if _, err := Roda("lento", 150*time.Millisecond, nil); err == nil || !strings.Contains(err.Error(), "demorou demais") {
		t.Errorf("prazo: %v", err)
	}
}

func TestRodaMandaAEntradaPeloStdin(t *testing.T) {
	m := dubles.Nova(t)
	m.Comando("le", `IFS= read -r l; printf 'li:%s' "$l"`)
	saida, err := Roda("le", time.Second, []byte("olá\n"))
	if err != nil || string(saida) != "li:olá" {
		t.Errorf("veio %q, %v", saida, err)
	}
	if !Tem("le") || Tem("nao-existe") {
		t.Error("Tem")
	}
}

func TestResumoFicaComAUltimaLinhaECorta(t *testing.T) {
	if got := Resumo("a\n\n  b   c \n\n"); got != "b c" {
		t.Errorf("veio %q", got)
	}
	if got := Resumo("\n \n"); got != "" {
		t.Errorf("vazio: %q", got)
	}
	if got := []rune(Resumo(strings.Repeat("ã", 500))); len(got) != 301 || got[300] != '…' {
		t.Errorf("corte: %d letras", len(got))
	}
}
