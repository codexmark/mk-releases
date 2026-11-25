package sshcfg

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// pasta monta um ~/.ssh de mentira: nome do arquivo (relativo) -> conteúdo.
func pasta(t *testing.T, arquivos map[string]string) (dir, casa string) {
	t.Helper()
	casa = t.TempDir()
	dir = filepath.Join(casa, ".ssh")
	for nome, texto := range arquivos {
		caminho := filepath.Join(dir, nome)
		if err := os.MkdirAll(filepath.Dir(caminho), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(caminho, []byte(texto), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	return dir, casa
}

func hosts(t *testing.T, arquivos map[string]string) []Host {
	t.Helper()
	dir, casa := pasta(t, arquivos)
	return Le(dir, casa).Hosts(ArquivoDoApp(dir))
}

func apelidos(hs []Host) []string {
	out := []string{}
	for _, h := range hs {
		out = append(out, h.Apelido)
	}
	return out
}

func acha(t *testing.T, hs []Host, apelido string) Host {
	t.Helper()
	for _, h := range hs {
		if h.Apelido == apelido {
			return h
		}
	}
	t.Fatalf("não achei o host %q em %v", apelido, apelidos(hs))
	return Host{}
}

func TestPalavras_AspasIgualEComentario(t *testing.T) {
	casos := []struct {
		linha string
		chave string
		args  []string
	}{
		{"Host a b c", "host", []string{"a", "b", "c"}},
		{"  HostName   exemplo.test  ", "hostname", []string{"exemplo.test"}},
		{"HOSTNAME=exemplo.test", "hostname", []string{"exemplo.test"}},
		{"Port = 2222", "port", []string{"2222"}},
		{"User\tmarina", "user", []string{"marina"}},
		{`IdentityFile "~/.ssh/minha chave"`, "identityfile", []string{"~/.ssh/minha chave"}},
		{`User "ana" # quem entra`, "user", []string{"ana"}},
		{"Port 22 # padrão", "port", []string{"22"}},
		{"# só comentário", "", nil},
		{"", "", nil},
		{"   ", "", nil},
		{"Host", "host", nil},
		{"Host a\r", "host", []string{"a"}},
	}
	for _, c := range casos {
		chave, args := palavras(c.linha)
		if chave != c.chave || !reflect.DeepEqual(args, c.args) {
			t.Errorf("palavras(%q) = %q %q; esperado %q %q", c.linha, chave, args, c.chave, c.args)
		}
	}
}

func TestCasa_PadroesDoSsh(t *testing.T) {
	casos := []struct {
		padroes []string
		nome    string
		quer    bool
	}{
		{[]string{"*"}, "atlas", true},
		{[]string{"*.oficina.example"}, "atlas.oficina.example", true},
		{[]string{"*.oficina.example"}, "atlas", false},
		{[]string{"web?"}, "web1", true},
		{[]string{"web?"}, "web10", false},
		{[]string{"atlas", "farol"}, "farol", true},
		{[]string{"*", "!farol"}, "farol", false},
		{[]string{"*", "!farol"}, "atlas", true},
		{[]string{"!farol"}, "atlas", false}, // só negação não casa com ninguém
		{[]string{"Atlas"}, "atlas", false},
		{[]string{"a*c*e"}, "abcde", true},
		{[]string{"a*c*e"}, "abcd", false},
	}
	for _, c := range casos {
		if obtido := casa(c.padroes, c.nome); obtido != c.quer {
			t.Errorf("casa(%q, %q) = %v", c.padroes, c.nome, obtido)
		}
	}
}

func TestHosts_OBasico(t *testing.T) {
	hs := hosts(t, map[string]string{"config": `
# o meu config
Host atlas
    HostName atlas.oficina.example
    User marina
    Port 2222
    IdentityFile ~/.ssh/id_ed25519
    ProxyJump farol

host farol
  hostname=192.0.2.10
`})
	if !reflect.DeepEqual(apelidos(hs), []string{"atlas", "farol"}) {
		t.Fatalf("apelidos: %v", apelidos(hs))
	}
	a := acha(t, hs, "atlas")
	if a.Host != "atlas.oficina.example" || a.Usuario != "marina" || a.Porta != 2222 || a.Chave != "~/.ssh/id_ed25519" || a.Salto != "farol" {
		t.Errorf("atlas: %+v", a)
	}
	if a.Origem != "config" {
		t.Errorf("origem: %q", a.Origem)
	}
	f := acha(t, hs, "farol")
	if f.Host != "192.0.2.10" || f.Usuario != "" || f.Porta != 22 || f.Chave != "" || f.Salto != "" {
		t.Errorf("farol: %+v", f)
	}
}

func TestHosts_SemHostNameOHostEOApelido(t *testing.T) {
	hs := hosts(t, map[string]string{"config": "Host servidor.test\n  User ana\n"})
	if h := acha(t, hs, "servidor.test"); h.Host != "servidor.test" {
		t.Errorf("host: %q", h.Host)
	}
}

func TestHosts_VariosApelidosNumaLinha(t *testing.T) {
	hs := hosts(t, map[string]string{"config": "Host a b c\n  User ana\n  Port 2200\n"})
	if !reflect.DeepEqual(apelidos(hs), []string{"a", "b", "c"}) {
		t.Fatalf("apelidos: %v", apelidos(hs))
	}
	for _, h := range hs {
		if h.Usuario != "ana" || h.Porta != 2200 || h.Host != h.Apelido {
			t.Errorf("%+v", h)
		}
	}
}

func TestHosts_PadroesComCuringaNaoSaoMaquinas(t *testing.T) {
	hs := hosts(t, map[string]string{"config": `
Host *
  User todos
Host *.oficina.example web? !bastiao
  Port 2222
Host atlas !outro
  HostName 192.0.2.1
`})
	if !reflect.DeepEqual(apelidos(hs), []string{"atlas"}) {
		t.Fatalf("só o atlas é máquina: %v", apelidos(hs))
	}
}

func TestHosts_OCuringaCompletaOHostEOPrimeiroValorGanha(t *testing.T) {
	hs := hosts(t, map[string]string{"config": `
Host atlas.oficina.example
  User marina
Host *.oficina.example
  User deploy
  Port 2222
Host farol
  HostName farol.test
Host *
  User todos
  IdentityFile ~/.ssh/geral
`})
	a := acha(t, hs, "atlas.oficina.example")
	if a.Usuario != "marina" || a.Porta != 2222 || a.Chave != "~/.ssh/geral" {
		t.Errorf("atlas: %+v", a)
	}
	f := acha(t, hs, "farol")
	if f.Usuario != "todos" || f.Porta != 22 {
		t.Errorf("farol: %+v", f)
	}
}

func TestHosts_OpcoesAntesDoPrimeiroHostValemParaTodos(t *testing.T) {
	hs := hosts(t, map[string]string{"config": "User global\nHost a\n  User meu\nHost b\n"})
	if acha(t, hs, "a").Usuario != "global" {
		t.Errorf("o primeiro valor ganha: o global vem antes")
	}
	if acha(t, hs, "b").Usuario != "global" {
		t.Errorf("b devia herdar o global")
	}
}

func TestHosts_MatchEncerraOBloco(t *testing.T) {
	hs := hosts(t, map[string]string{"config": `
Host atlas
  HostName 192.0.2.1
Match host atlas exec "true"
  User intruso
  Port 9
Host farol
  User ana
`})
	a := acha(t, hs, "atlas")
	if a.Usuario != "" || a.Porta != 22 {
		t.Errorf("o que está no Match não é do atlas: %+v", a)
	}
	if acha(t, hs, "farol").Usuario != "ana" {
		t.Errorf("o Host depois do Match volta a valer")
	}
	if len(hs) != 2 {
		t.Errorf("hosts: %v", apelidos(hs))
	}
}

func TestHosts_IncludeComGlobRelativoAPastaDoSsh(t *testing.T) {
	hs := hosts(t, map[string]string{
		"config":          "Include config.d/*.conf\nHost local\n  HostName 192.0.2.9\n",
		"config.d/b.conf": "Host do-b\n  HostName b.test\n",
		"config.d/a.conf": "Host do-a\n  HostName a.test\n",
		"config.d/c.txt":  "Host nao-entra\n",
	})
	if !reflect.DeepEqual(apelidos(hs), []string{"do-a", "do-b", "local"}) {
		t.Fatalf("apelidos (em ordem de inclusão): %v", apelidos(hs))
	}
	if h := acha(t, hs, "do-a"); filepath.Base(h.Arquivo) != "a.conf" || h.Origem != "config" {
		t.Errorf("do-a: %+v", h)
	}
	if h := acha(t, hs, "local"); filepath.Base(h.Arquivo) != "config" {
		t.Errorf("local: %+v", h)
	}
}

func TestHosts_IncludeAbsolutoComTilVariosNumaLinhaEAninhado(t *testing.T) {
	dir, casa := pasta(t, map[string]string{
		"config":      "Include ~/extra/um.conf \"dois.conf\"\n",
		"dois.conf":   "Include tres.conf\nHost dois\n",
		"tres.conf":   "Host tres\n",
		"sobrou.conf": "Host sobrou\n",
	})
	if err := os.MkdirAll(filepath.Join(casa, "extra"), 0o755); err != nil {
		t.Fatal(err)
	}
	abs := filepath.Join(casa, "extra", "um.conf")
	if err := os.WriteFile(abs, []byte("Host um\nInclude "+filepath.Join(dir, "sobrou.conf")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	hs := Le(dir, casa).Hosts(ArquivoDoApp(dir))
	if !reflect.DeepEqual(apelidos(hs), []string{"um", "sobrou", "tres", "dois"}) {
		t.Fatalf("apelidos: %v", apelidos(hs))
	}
}

func TestHosts_IncludeEmLacoNaoPrende(t *testing.T) {
	hs := hosts(t, map[string]string{
		"config": "Include outro\nHost a\n",
		"outro":  "Include config\nHost b\n",
	})
	if !reflect.DeepEqual(apelidos(hs), []string{"b", "a"}) {
		t.Fatalf("apelidos: %v", apelidos(hs))
	}
}

func TestHosts_IncludeQueNaoExisteOuSemConfigNaoEErro(t *testing.T) {
	if hs := hosts(t, map[string]string{"config": "Include nada/*.conf\nHost a\n"}); len(hs) != 1 {
		t.Errorf("hosts: %v", apelidos(hs))
	}
	if hs := hosts(t, map[string]string{}); len(hs) != 0 {
		t.Errorf("sem config: %v", apelidos(hs))
	}
}

func TestHosts_IncludeDentroDeUmHostSoValeParaEle(t *testing.T) {
	hs := hosts(t, map[string]string{
		"config":    "Host atlas\n  Include so-dele\n  Port 2222\nHost farol\n",
		"so-dele":   "User marina\n",
		"nao-usado": "",
	})
	if a := acha(t, hs, "atlas"); a.Usuario != "marina" || a.Porta != 2222 {
		t.Errorf("atlas: %+v", a)
	}
	if f := acha(t, hs, "farol"); f.Usuario != "" {
		t.Errorf("farol não herda o Include do atlas: %+v", f)
	}
}

func TestHosts_IncludeDentroDeMatchNaoELido(t *testing.T) {
	hs := hosts(t, map[string]string{
		"config": "Match all\n  Include la\nHost a\n",
		"la":     "Host escondido\n",
	})
	if !reflect.DeepEqual(apelidos(hs), []string{"a"}) {
		t.Fatalf("apelidos: %v", apelidos(hs))
	}
}

func TestHosts_OrigemMkportoSeVemDoArquivoDoApp(t *testing.T) {
	hs := hosts(t, map[string]string{
		"config":                "Include config.d/mkporto.conf\nHost meu\n",
		"config.d/mkporto.conf": "Host do-app\n  HostName 192.0.2.7\n",
	})
	if h := acha(t, hs, "do-app"); h.Origem != "mkporto" {
		t.Errorf("do-app: %+v", h)
	}
	if h := acha(t, hs, "meu"); h.Origem != "config" {
		t.Errorf("meu: %+v", h)
	}
}

func TestHosts_ApelidoRepetidoApareceUmaVezSoComOPrimeiroArquivo(t *testing.T) {
	hs := hosts(t, map[string]string{"config": "Host a\n  User um\nHost a b\n  User dois\n  Port 2222\n"})
	if !reflect.DeepEqual(apelidos(hs), []string{"a", "b"}) {
		t.Fatalf("apelidos: %v", apelidos(hs))
	}
	if a := acha(t, hs, "a"); a.Usuario != "um" || a.Porta != 2222 {
		t.Errorf("a: %+v", a)
	}
}

func TestHosts_PortaTortaVira22EPercentHViraOApelido(t *testing.T) {
	hs := hosts(t, map[string]string{"config": "Host a\n  Port banana\n  HostName %h.oficina.example\nHost b\n  Port 99999\n  ProxyJump none\n"})
	if a := acha(t, hs, "a"); a.Porta != 22 || a.Host != "a.oficina.example" {
		t.Errorf("a: %+v", a)
	}
	if b := acha(t, hs, "b"); b.Porta != 22 || b.Salto != "" {
		t.Errorf("b: %+v", b)
	}
}

func TestDeclaradoFora_SoContaOQueNaoEDoApp(t *testing.T) {
	dir, casa := pasta(t, map[string]string{
		"config":                "Include config.d/*\nHost meu outro\n",
		"config.d/mkporto.conf": "Host do-app\n",
		"config.d/trab.conf":    "Host trabalho\n",
	})
	l := Le(dir, casa)
	app := ArquivoDoApp(dir)
	for apelido, quer := range map[string]string{"meu": "config", "outro": "config", "trabalho": "trab.conf", "do-app": "", "ninguem": ""} {
		obtido := l.DeclaradoFora(apelido, app)
		if quer == "" && obtido != "" || quer != "" && filepath.Base(obtido) != quer {
			t.Errorf("DeclaradoFora(%q) = %q; esperado %q", apelido, obtido, quer)
		}
	}
}
