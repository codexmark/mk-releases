package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"mkpacotes/internal/base"
	"mkpacotes/internal/dubles"
)

// roda um subcomando com a saída num buffer e devolve o código e as linhas.
func roda(t *testing.T, args ...string) (int, []string) {
	t.Helper()
	antes := base.Saida
	b := &bytes.Buffer{}
	base.Saida = b
	defer func() { base.Saida = antes }()
	codigo := rodar(args)
	texto := strings.TrimRight(b.String(), "\n")
	if texto == "" {
		return codigo, nil
	}
	return codigo, strings.Split(texto, "\n")
}

// quer confere o código e a resposta (uma linha só), comparando como JSON.
func quer(t *testing.T, codigo int, esperado string, args ...string) {
	t.Helper()
	veio, linhas := roda(t, args...)
	if veio != codigo {
		t.Errorf("%v: código %d, queria %d (%v)", args, veio, codigo, linhas)
	}
	if len(linhas) != 1 {
		t.Fatalf("%v: devia responder uma linha, vieram %d: %v", args, len(linhas), linhas)
	}
	var a, b any
	if err := json.Unmarshal([]byte(linhas[0]), &a); err != nil {
		t.Fatalf("%v: a resposta não é JSON: %s", args, linhas[0])
	}
	if err := json.Unmarshal([]byte(esperado), &b); err != nil {
		t.Fatalf("o esperado não é JSON: %v\n%s", err, esperado)
	}
	if !reflect.DeepEqual(a, b) {
		t.Errorf("%v:\n veio %s\n quer %s", args, linhas[0], esperado)
	}
}

// querErro confere o código e um trecho da frase de erro.
func querErro(t *testing.T, codigo int, trecho string, args ...string) {
	t.Helper()
	veio, linhas := roda(t, args...)
	if veio != codigo || len(linhas) == 0 {
		t.Errorf("%v: código %d, queria %d (%v)", args, veio, codigo, linhas)
		return
	}
	var r map[string]any
	if err := json.Unmarshal([]byte(linhas[len(linhas)-1]), &r); err != nil {
		t.Fatalf("%v: a resposta não é JSON: %s", args, linhas[len(linhas)-1])
	}
	frase, _ := r["erro"].(string)
	if len(r) != 1 || !strings.Contains(frase, trecho) {
		t.Errorf("%v: queria um erro com %q, veio %s", args, trecho, linhas[len(linhas)-1])
	}
}

// campo lê um valor de dentro de uma resposta: campo(r, "totais", "aur").
func campo(v any, caminho ...any) any {
	for _, c := range caminho {
		switch k := c.(type) {
		case string:
			m, _ := v.(map[string]any)
			v = m[k]
		case int:
			l, _ := v.([]any)
			if k >= len(l) {
				return nil
			}
			v = l[k]
		}
	}
	return v
}

func objeto(t *testing.T, linha string) map[string]any {
	t.Helper()
	var r map[string]any
	if err := json.Unmarshal([]byte(linha), &r); err != nil {
		t.Fatalf("não é JSON: %s", linha)
	}
	return r
}

const repositorios = "core glibc 2.42-1 [installed]\ncore bash 5.3.21-1 [installed: 5.3.20-1]\ncore libnl 3.11.0-1 [installed]\n" +
	"extra htop 3.5.3-1 [installed]\nextra fd 10.2.0-1\ncore mkinitcpio 40-1 [installed]\n"

// maquina monta a máquina de mentira dos testes de contrato: a base do
// pacman, o pacman e o AUR falsos, o Mk, o cache e o registro.
func maquina(t *testing.T) *dubles.Mesa {
	t.Helper()
	m := dubles.Nova(t)
	raiz := dubles.Base(t,
		dubles.Pacote{Nome: "glibc", Versao: "2.42-1", Desc: "GNU C Library", Tamanho: 50000000, InstaladoEm: 1792000000, Dependencia: true},
		dubles.Pacote{Nome: "bash", Versao: "5.3.20-1", Desc: "The GNU Bourne Again shell", URL: "https://www.gnu.org/software/bash/",
			Empacotador: "Fulano <fulano@archlinux.org>", Tamanho: 10000000, InstaladoEm: 1792000100, ConstruidoEm: 1789000000,
			Licencas: []string{"GPL-3.0-or-later"}, Depende: []string{"glibc>=2.38", "ncurses"}, Opcionais: []string{"bash-completion: for tab completion"},
			Fornece: []string{"sh"}, Arquivos: []string{"usr/", "usr/bin/", "usr/bin/bash", "usr/share/", "usr/share/man/man1/bash.1.gz"}},
		dubles.Pacote{Nome: "libnl", Versao: "3.11.0-1", Desc: "Netlink", Tamanho: 1500000, InstaladoEm: 1792000200, Dependencia: true, Depende: []string{"glibc"}},
		dubles.Pacote{Nome: "htop", Versao: "3.5.3-1", Desc: "Interactive process viewer", Tamanho: 400000, InstaladoEm: 1792000300, Depende: []string{"libnl", "glibc"}},
		dubles.Pacote{Nome: "libvelha", Versao: "0.9-2", Desc: "Sobrou", Tamanho: 1000, InstaladoEm: 1792000400, Dependencia: true},
		dubles.Pacote{Nome: "mkinitcpio", Versao: "40-1", Desc: "Initramfs", Tamanho: 2000, InstaladoEm: 1792000500, Depende: []string{"bash"}},
		dubles.Pacote{Nome: "do-aur", Versao: "1.0-1", Desc: "Veio do AUR", Tamanho: 3000, InstaladoEm: 1792000600, Depende: []string{"sh"}},
	)
	if err := os.MkdirAll(filepath.Join(raiz, "sync"), 0o755); err != nil {
		t.Fatal(err)
	}
	mk := t.TempDir()
	os.MkdirAll(filepath.Join(mk, "system", "packages"), 0o755)
	os.WriteFile(filepath.Join(mk, "system", "packages", "base.packages"), []byte("base\nmkinitcpio\n"), 0o644)
	t.Setenv("MKPACOTES_MK", mk)
	t.Setenv("MKPACOTES_CACHE", t.TempDir())
	t.Setenv("MKPACOTES_TRAVA", filepath.Join(t.TempDir(), "trava"))
	t.Setenv("MKPACOTES_TERMINAL", "")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	registro := filepath.Join(t.TempDir(), "pacman.log")
	os.WriteFile(registro, []byte("[2026-11-20T10:00:00+0000] [ALPM] installed fd (10.1-1)\n[2026-11-20T10:00:01+0000] [ALPM] transaction completed\n"+
		"[2026-11-21T10:00:00+0000] [ALPM] upgraded htop (3.5.2-1 -> 3.5.3-1)\n[2026-11-22T10:00:00+0000] [ALPM] removed fd (10.1-1)\n"), 0o644)
	t.Setenv("MKPACOTES_LOG", registro)
	m.Comando("pacman",
		dubles.Regra{Contem: " -Sl ", Texto: repositorios},
		dubles.Regra{Contem: " -Ss -- top ", Texto: "extra/htop 3.5.3-1 [installed]\n    Interactive process viewer\nextra/btop 1.4.5-1\n    A monitor of resources on top\n"},
		dubles.Regra{Contem: " -Ss ", Codigo: 1},
		dubles.Regra{Contem: " -Si -- fd ", Texto: "Repository      : extra\nName            : fd\nVersion         : 10.2.0-1\nDescription     : Simple, fast alternative to find\n" +
			"URL             : https://github.com/sharkdp/fd\nLicenses        : MIT  Apache-2.0\nProvides        : None\nDepends On      : gcc-libs  glibc\n" +
			"Optional Deps   : None\nConflicts With  : None\nDownload Size   : 1.00 MiB\nInstalled Size  : 4.00 MiB\nPackager        : Beltrano <b@archlinux.org>\n"},
		dubles.Regra{Contem: " -Si ", Codigo: 1, Erro: "error: package 'x' was not found\n"},
		dubles.Regra{Contem: " -Qu ", Texto: "bash 5.3.20-1 -> 5.3.21-1\n"},
		dubles.Regra{Contem: " -S --needed --noconfirm --print --print-format %n %v %r %s -- fd ", Texto: "fd 10.2.0-1 extra 1048576\n"},
		dubles.Regra{Contem: " -Rs --noconfirm --print --print-format %n %v -- htop ", Texto: "htop 3.5.3-1\nlibnl 3.11.0-1\n"},
		dubles.Regra{Contem: " -Rs --noconfirm --print --print-format %n %v -- libvelha ", Texto: "libvelha 0.9-2\n"},
		dubles.Regra{Contem: " -Rs --noconfirm --print --print-format %n %v -- mkinitcpio ", Texto: "mkinitcpio 40-1\n"},
	)
	aurFalso(t)
	return m
}

const doAURNoAUR = `{"Name":"do-aur","Version":"1.1-1","Description":"Veio do AUR","URL":"https://exemplo.org/do-aur","NumVotes":12,"Popularity":0.4,
 "OutOfDate":1794000000,"Maintainer":"fulano","LastModified":1795000000,"License":["MIT"],"Depends":["sh"]}`
const topNoAUR = `{"Name":"top-do-aur","Version":"2-1","Description":"Outro top","NumVotes":3,"Popularity":0.1,"OutOfDate":null,"Maintainer":null,"LastModified":1795000001,
 "Depends":["glibc","libfalta"]}`

// aurFalso sobe um AUR que conhece o do-aur e o top-do-aur.
func aurFalso(t *testing.T) {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var achados []string
		if strings.HasPrefix(r.URL.Path, "/rpc/v5/search/") {
			if strings.Contains(r.URL.Path, "top") || strings.Contains(r.URL.Path, "outro") || strings.Contains(r.URL.Path, "banana") {
				achados = append(achados, topNoAUR)
			}
		} else {
			for _, n := range r.URL.Query()["arg[]"] {
				switch n {
				case "do-aur":
					achados = append(achados, doAURNoAUR)
				case "top-do-aur":
					achados = append(achados, topNoAUR)
				}
			}
		}
		fmt.Fprintf(w, `{"type":"multiinfo","results":[%s]}`, strings.Join(achados, ","))
	}))
	t.Cleanup(s.Close)
	t.Setenv("MKPACOTES_AUR", s.URL)
}

// semRede aponta o AUR para uma porta que recusa a conexão.
func semRede(t *testing.T) {
	t.Helper()
	s := httptest.NewServer(http.NotFoundHandler())
	s.Close()
	t.Setenv("MKPACOTES_AUR", s.URL)
}

func TestSemSubcomandoOuComUmDesconhecido_UsoEmJSON(t *testing.T) {
	for _, args := range [][]string{{}, {"banana"}} {
		codigo, linhas := roda(t, args...)
		if codigo != 1 || len(linhas) != 1 {
			t.Fatalf("%v: código %d, %v", args, codigo, linhas)
		}
		uso, _ := objeto(t, linhas[0])["erro"].(string)
		for _, sub := range []string{"uso: mkpacotes-coletor <", "instalados", "procurar", "detalhe", "atualizacoes", "historico", "estado", "plano", "agir"} {
			if !strings.Contains(uso, sub) {
				t.Errorf("o uso devia citar %q: %s", sub, uso)
			}
		}
	}
}

func TestOpcoes(t *testing.T) {
	pos, ops, err := opcoes([]string{"remover", "--mesmo-assim", "htop", "--linhas", "5", "--", "-x"}, []string{"--mesmo-assim"}, []string{"--linhas"})
	if err != nil || !reflect.DeepEqual(pos, []string{"remover", "htop", "-x"}) || ops["--mesmo-assim"] != "sim" || ops["--linhas"] != "5" {
		t.Errorf("veio %v %v %v", pos, ops, err)
	}
	for _, args := range [][]string{{"--banana"}, {"-S"}, {"--linhas"}, {"htop", "-Rns"}} {
		if _, _, err := opcoes(args, []string{"--mesmo-assim"}, []string{"--linhas"}); err == nil {
			t.Errorf("%v devia dar erro", args)
		}
	}
}
