package sshcfg

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
)

func boa() Entrada {
	return Entrada{Apelido: "atlas", Host: "atlas.oficina.example", Usuario: "marina", Porta: 22}
}

func modo(t *testing.T, caminho string) os.FileMode {
	t.Helper()
	info, err := os.Stat(caminho)
	if err != nil {
		t.Fatal(err)
	}
	return info.Mode().Perm()
}

func le(t *testing.T, caminho string) string {
	t.Helper()
	b, err := os.ReadFile(caminho)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestValida_OQuePassa(t *testing.T) {
	casos := []Entrada{
		boa(),
		{Apelido: "a", Host: "192.0.2.1", Porta: 1},
		{Apelido: "Web_1.prod-2", Host: "2001:db8::1", Porta: 65535},
		{Apelido: "x", Host: "x.test", Usuario: "ana.silva", Porta: 22, Chave: "~/.ssh/minha chave", Salto: "ana@farol:2222,outro"},
	}
	for _, e := range casos {
		if err := Valida(e); err != nil {
			t.Errorf("%+v: %v", e, err)
		}
	}
}

func TestValida_RecusaInjecaoDeOpcaoEDeLinha(t *testing.T) {
	com := func(muda func(*Entrada)) Entrada {
		e := boa()
		muda(&e)
		return e
	}
	casos := []struct {
		nome  string
		e     Entrada
		campo string
	}{
		{"apelido vazio", com(func(e *Entrada) { e.Apelido = "" }), "apelido"},
		{"apelido começando com -", com(func(e *Entrada) { e.Apelido = "-oProxyCommand=x" }), "apelido"},
		{"apelido com espaço", com(func(e *Entrada) { e.Apelido = "a b" }), "apelido"},
		{"apelido com curinga", com(func(e *Entrada) { e.Apelido = "web*" }), "apelido"},
		{"apelido com quebra de linha", com(func(e *Entrada) { e.Apelido = "a\nHost *" }), "apelido"},
		{"apelido com acento", com(func(e *Entrada) { e.Apelido = "máquina" }), "apelido"},
		{"apelido com barra", com(func(e *Entrada) { e.Apelido = "a/b" }), "apelido"},
		{"apelido comprido", com(func(e *Entrada) { e.Apelido = strings.Repeat("a", 65) }), "apelido"},
		{"host vazio", com(func(e *Entrada) { e.Host = "" }), "host"},
		{"host começando com -", com(func(e *Entrada) { e.Host = "-oProxyCommand=touch pwn" }), "host"},
		{"host com espaço", com(func(e *Entrada) { e.Host = "a b" }), "host"},
		{"host com quebra de linha", com(func(e *Entrada) { e.Host = "a\nProxyCommand touch pwn" }), "host"},
		{"host com retorno de carro", com(func(e *Entrada) { e.Host = "a\rb" }), "host"},
		{"host com tab", com(func(e *Entrada) { e.Host = "a\tb" }), "host"},
		{"host com controle", com(func(e *Entrada) { e.Host = "a\x00b" }), "host"},
		{"host com aspas", com(func(e *Entrada) { e.Host = `a"b` }), "host"},
		{"host com #", com(func(e *Entrada) { e.Host = "#a" }), "host"},
		{"host comprido", com(func(e *Entrada) { e.Host = strings.Repeat("a", 254) }), "host"},
		{"usuario começando com -", com(func(e *Entrada) { e.Usuario = "-l" }), "usuario"},
		{"usuario com espaço", com(func(e *Entrada) { e.Usuario = "ana silva" }), "usuario"},
		{"usuario com quebra de linha", com(func(e *Entrada) { e.Usuario = "ana\nPort 1" }), "usuario"},
		{"porta zero", com(func(e *Entrada) { e.Porta = 0 }), "porta"},
		{"porta negativa", com(func(e *Entrada) { e.Porta = -1 }), "porta"},
		{"porta grande", com(func(e *Entrada) { e.Porta = 65536 }), "porta"},
		{"chave com quebra de linha", com(func(e *Entrada) { e.Chave = "~/.ssh/k\nProxyCommand x" }), "chave"},
		{"chave com aspas", com(func(e *Entrada) { e.Chave = `~/.ssh/k" ProxyCommand "x` }), "chave"},
		{"chave com barra invertida", com(func(e *Entrada) { e.Chave = `~/.ssh/k\` }), "chave"},
		{"chave começando com -", com(func(e *Entrada) { e.Chave = "-x" }), "chave"},
		{"salto começando com -", com(func(e *Entrada) { e.Salto = "-oProxyCommand=x" }), "salto"},
		{"salto com espaço", com(func(e *Entrada) { e.Salto = "a b" }), "salto"},
		{"salto com quebra de linha", com(func(e *Entrada) { e.Salto = "a\nb" }), "salto"},
	}
	for _, c := range casos {
		err := Valida(c.e)
		if err == nil {
			t.Errorf("%s: devia recusar", c.nome)
			continue
		}
		if err.Campo != c.campo {
			t.Errorf("%s: campo %q, esperado %q (%s)", c.nome, err.Campo, c.campo, err.Frase)
		}
		if err.Frase == "" {
			t.Errorf("%s: sem frase", c.nome)
		}
	}
}

func TestTexto_OQueEscreveOLeitorLeDeVolta(t *testing.T) {
	es := []Entrada{
		{Apelido: "atlas", Host: "atlas.oficina.example", Usuario: "marina", Porta: 2222, Chave: "~/.ssh/minha chave", Salto: "farol"},
		{Apelido: "farol", Host: "192.0.2.10", Porta: 22},
	}
	dir, _ := pasta(t, nil)
	if err := Grava(dir, es); err != nil {
		t.Fatal(err)
	}
	if de := DoApp(ArquivoDoApp(dir)); !reflect.DeepEqual(de, es) {
		t.Errorf("leu de volta:\n%+v\nesperado:\n%+v", de, es)
	}
	texto := le(t, ArquivoDoApp(dir))
	if !strings.HasPrefix(texto, "# ") || !strings.Contains(texto, "MkPorto") {
		t.Errorf("sem o cabeçalho que avisa de quem é o arquivo:\n%s", texto)
	}
	if !strings.Contains(texto, `IdentityFile "~/.ssh/minha chave"`) {
		t.Errorf("a chave vai entre aspas:\n%s", texto)
	}
	if strings.Contains(texto, "User \n") || strings.Count(texto, "User ") != 1 {
		t.Errorf("campo vazio não vira linha:\n%s", texto)
	}
}

func TestGrava_Permissoes600E700(t *testing.T) {
	casa := t.TempDir()
	dir := filepath.Join(casa, ".ssh") // ainda não existe
	antiga := mascara(0)
	defer mascara(antiga)
	if err := Grava(dir, []Entrada{boa()}); err != nil {
		t.Fatal(err)
	}
	if m := modo(t, ArquivoDoApp(dir)); m != 0o600 {
		t.Errorf("arquivo do app: %o", m)
	}
	if m := modo(t, filepath.Join(dir, "config.d")); m != 0o700 {
		t.Errorf("config.d: %o", m)
	}
	if m := modo(t, dir); m != 0o700 {
		t.Errorf(".ssh: %o", m)
	}
}

func TestGrava_ReescreveInteiroSemDeixarTemporario(t *testing.T) {
	dir, _ := pasta(t, nil)
	if err := Grava(dir, []Entrada{boa(), {Apelido: "b", Host: "b.test", Porta: 22}}); err != nil {
		t.Fatal(err)
	}
	if err := Grava(dir, []Entrada{{Apelido: "b", Host: "b.test", Porta: 22}}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(le(t, ArquivoDoApp(dir)), "atlas") {
		t.Errorf("o que saiu da lista sai do arquivo")
	}
	itens, _ := os.ReadDir(filepath.Join(dir, "config.d"))
	if len(itens) != 1 {
		t.Errorf("sobrou arquivo temporário: %v", itens)
	}
}

func TestGrava_NaoEscreveEntradaInvalida(t *testing.T) {
	dir, _ := pasta(t, nil)
	if err := Grava(dir, []Entrada{{Apelido: "a", Host: "x\nProxyCommand y", Porta: 22}}); err == nil {
		t.Fatal("devia recusar")
	}
	if _, err := os.Stat(ArquivoDoApp(dir)); err == nil {
		t.Errorf("gravou mesmo assim")
	}
}

func TestDoApp_ArquivoQueNaoExisteEVazio(t *testing.T) {
	if es := DoApp(filepath.Join(t.TempDir(), "nada.conf")); len(es) != 0 {
		t.Errorf("%+v", es)
	}
}

func TestGaranteInclude_CriaOConfigCom600(t *testing.T) {
	dir, casa := pasta(t, nil)
	antiga := mascara(0)
	defer mascara(antiga)
	posto, err := GaranteInclude(dir, casa)
	if err != nil || !posto {
		t.Fatalf("posto=%v err=%v", posto, err)
	}
	if texto := le(t, filepath.Join(dir, "config")); texto != "Include config.d/mkporto.conf\n" {
		t.Errorf("config: %q", texto)
	}
	if m := modo(t, filepath.Join(dir, "config")); m != 0o600 {
		t.Errorf("modo: %o", m)
	}
}

func TestGaranteInclude_NoTopoPreservandoORestoByteAByte(t *testing.T) {
	// De propósito: tabs, CRLF, espaço no fim, sem quebra na última linha.
	antes := "# o meu config\r\n\tHost  antigo   \n    HostName 192.0.2.5\t\n\n\nHost *\n  User eu"
	dir, casa := pasta(t, map[string]string{"config": antes})
	if err := os.Chmod(filepath.Join(dir, "config"), 0o640); err != nil {
		t.Fatal(err)
	}
	posto, err := GaranteInclude(dir, casa)
	if err != nil || !posto {
		t.Fatalf("posto=%v err=%v", posto, err)
	}
	depois := le(t, filepath.Join(dir, "config"))
	if depois != "Include config.d/mkporto.conf\n\n"+antes {
		t.Errorf("config:\n%q", depois)
	}
	if m := modo(t, filepath.Join(dir, "config")); m != 0o640 {
		t.Errorf("a permissão que a pessoa pôs fica: %o", m)
	}
}

func TestGaranteInclude_NaoDuplica(t *testing.T) {
	for nome, antes := range map[string]string{
		"o próprio":            "Include config.d/mkporto.conf\nHost a\n",
		"segunda vez":          "Include config.d/mkporto.conf\n\nHost a\n",
		"glob":                 "# meus\nInclude config.d/*\nHost a\n",
		"glob com extensão":    "include   config.d/*.conf\n",
		"com igual e aspas":    "Include=\"config.d/mkporto.conf\"\n",
		"entre outros":         "Include outra/* config.d/mk*.conf\n",
		"depois de uma opção":  "ServerAliveInterval 30\nInclude config.d/*\n",
		"maiúsculas e o til":   "INCLUDE ~/.ssh/config.d/*\n",
		"absoluto":             "", // preenchido abaixo
		"glob em duas pastas?": "Include config.?/mkporto.conf\n",
	} {
		dir, casa := pasta(t, nil)
		if nome == "absoluto" {
			antes = "Include " + ArquivoDoApp(dir) + "\n"
		}
		if err := os.WriteFile(filepath.Join(dir, "config"), []byte(antes), 0o600); err != nil {
			t.Fatal(err)
		}
		posto, err := GaranteInclude(dir, casa)
		if err != nil || posto {
			t.Errorf("%s: posto=%v err=%v", nome, posto, err)
		}
		if depois := le(t, filepath.Join(dir, "config")); depois != antes {
			t.Errorf("%s: o config mudou:\n%q", nome, depois)
		}
	}
}

func TestGaranteInclude_IncludeQueNaoCobreOuQueSoValeParaUmHostNaoConta(t *testing.T) {
	for nome, antes := range map[string]string{
		"outro arquivo":      "Include config.d/outro.conf\nHost a\n",
		"outra pasta":        "Include outra/*\n",
		"dentro de um Host":  "Host a\n  Include config.d/*\n",
		"depois de um Host":  "Host a\n  User x\nInclude config.d/mkporto.conf\n",
		"dentro de um Match": "Match all\nInclude config.d/*\n",
		"comentado":          "# Include config.d/mkporto.conf\nHost a\n",
		"subpasta":           "Include config.d/sub/*\n",
	} {
		dir, casa := pasta(t, map[string]string{"config": antes})
		posto, err := GaranteInclude(dir, casa)
		if err != nil || !posto {
			t.Errorf("%s: posto=%v err=%v", nome, posto, err)
			continue
		}
		if depois := le(t, filepath.Join(dir, "config")); depois != "Include config.d/mkporto.conf\n\n"+antes {
			t.Errorf("%s:\n%q", nome, depois)
		}
		// E a segunda chamada não põe outra.
		if posto, _ := GaranteInclude(dir, casa); posto {
			t.Errorf("%s: duplicou", nome)
		}
	}
}

func TestGaranteInclude_ConfigQueEAtalhoContinuaAtalho(t *testing.T) {
	dir, casa := pasta(t, nil)
	alvo := filepath.Join(casa, "dotfiles", "ssh-config")
	if err := os.MkdirAll(filepath.Dir(alvo), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(alvo, []byte("Host a\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(alvo, filepath.Join(dir, "config")); err != nil {
		t.Fatal(err)
	}
	if posto, err := GaranteInclude(dir, casa); err != nil || !posto {
		t.Fatalf("posto=%v err=%v", posto, err)
	}
	info, err := os.Lstat(filepath.Join(dir, "config"))
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("o atalho virou arquivo")
	}
	if texto := le(t, alvo); texto != "Include config.d/mkporto.conf\n\nHost a\n" {
		t.Errorf("alvo: %q", texto)
	}
}

func TestGaranteInclude_OSshPassaAVerOArquivoDoApp(t *testing.T) {
	dir, casa := pasta(t, map[string]string{"config": "Host meu\n  User eu\nHost *\n  User todos\n"})
	if err := Grava(dir, []Entrada{boa()}); err != nil {
		t.Fatal(err)
	}
	if _, err := GaranteInclude(dir, casa); err != nil {
		t.Fatal(err)
	}
	hs := Le(dir, casa).Hosts(ArquivoDoApp(dir))
	if !reflect.DeepEqual(apelidos(hs), []string{"atlas", "meu"}) {
		t.Fatalf("apelidos: %v", apelidos(hs))
	}
	if a := acha(t, hs, "atlas"); a.Origem != "mkporto" || a.Usuario != "marina" {
		t.Errorf("atlas: %+v", a)
	}
}

// mascara troca a umask do processo e devolve a anterior.
func mascara(m int) int { return syscall.Umask(m) }

func TestPalavraSegura(t *testing.T) {
	for _, boa := range []string{"farol", "ana@farol:2222", "192.0.2.1", "2001:db8::1"} {
		if !PalavraSegura(boa) {
			t.Errorf("%q devia servir", boa)
		}
	}
	for _, ruim := range []string{"", "-oProxyCommand=x", "a b", "a\nb", "a\"b", "#a", "a\x00b"} {
		if PalavraSegura(ruim) {
			t.Errorf("%q não devia servir", ruim)
		}
	}
}
