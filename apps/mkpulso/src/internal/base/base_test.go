package base

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestRaizesPadrao(t *testing.T) {
	for _, v := range []string{"MKPULSO_PROC", "MKPULSO_SYS", "MKPULSO_ETC", "MKPULSO_RUN"} {
		t.Setenv(v, "")
	}
	casos := map[string]string{
		Proc("1", "stat"):       "/proc/1/stat",
		Sys("class", "drm"):     "/sys/class/drm",
		Etc("passwd"):           "/etc/passwd",
		Run("user", "1000"):     "/run/user/1000",
		Proc():                  "/proc",
		Sys("a", "..", "b"):     "/sys/b",
		Etc("xdg", "autostart"): "/etc/xdg/autostart",
	}
	for obtido, esperado := range casos {
		if obtido != esperado {
			t.Errorf("esperava %q, veio %q", esperado, obtido)
		}
	}
}

func TestRaizesTrocadasPelasVariaveis(t *testing.T) {
	t.Setenv("MKPULSO_PROC", "/tmp/falso/proc")
	t.Setenv("MKPULSO_SYS", "/tmp/falso/sys")
	t.Setenv("MKPULSO_ETC", "/tmp/falso/etc")
	t.Setenv("MKPULSO_RUN", "/tmp/falso/run")
	t.Setenv("MKPULSO_HOME", "/tmp/falso/casa")
	if got := Proc("self", "cgroup"); got != "/tmp/falso/proc/self/cgroup" {
		t.Errorf("Proc: %q", got)
	}
	if got := Sys("power"); got != "/tmp/falso/sys/power" {
		t.Errorf("Sys: %q", got)
	}
	if got := Etc("passwd"); got != "/tmp/falso/etc/passwd" {
		t.Errorf("Etc: %q", got)
	}
	if got := Run("x"); got != "/tmp/falso/run/x" {
		t.Errorf("Run: %q", got)
	}
	if got := Home(".config", "mk"); got != "/tmp/falso/casa/.config/mk" {
		t.Errorf("Home: %q", got)
	}
}

func TestHomeSemVariavelEADeQuemRoda(t *testing.T) {
	t.Setenv("MKPULSO_HOME", "")
	t.Setenv("HOME", "/home/alguem")
	if got := Home(".config"); got != "/home/alguem/.config" {
		t.Errorf("Home: %q", got)
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
	_ = Escreve(map[string]string{"texto": "uma\nduas"})
	if strings.Count(b.String(), "\n") != 1 {
		t.Errorf("a quebra de dentro do texto tem de sair escapada: %q", b.String())
	}
}

func TestFalhaEscreveOErroEDevolveOCodigo(t *testing.T) {
	b := captura(t)
	if c := Falha(Recusado, "não mexo no processo %d", 1); c != Recusado {
		t.Errorf("código %d", c)
	}
	var r map[string]string
	if err := json.Unmarshal(b.Bytes(), &r); err != nil {
		t.Fatalf("não é JSON: %v", err)
	}
	if r["erro"] != "não mexo no processo 1" || len(r) != 1 {
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
