package dubles

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Pacote é um pacote instalado de mentira, para a base falsa do pacman.
type Pacote struct {
	Nome, Versao, Desc string
	URL, Empacotador   string
	Tamanho            int64
	InstaladoEm        int64
	ConstruidoEm       int64
	Dependencia        bool // veio como dependência (%REASON% 1)
	Depende            []string
	Opcionais          []string
	Fornece            []string
	Conflita           []string
	Licencas           []string
	Arquivos           []string // como no arquivo files: "usr/", "usr/bin/x"
}

// Base monta uma base do pacman falsa (local/ com um desc e um files por
// pacote) numa pasta temporária, faz dela o MKPACOTES_PACMAN_DB do teste e
// devolve a pasta.
func Base(t testing.TB, pacotes ...Pacote) string {
	t.Helper()
	raiz := t.TempDir()
	if err := os.MkdirAll(filepath.Join(raiz, "local"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(raiz, "local", "ALPM_DB_VERSION"), []byte("9\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, p := range pacotes {
		Instala(t, raiz, p)
	}
	t.Setenv("MKPACOTES_PACMAN_DB", raiz)
	return raiz
}

// Instala acrescenta um pacote a uma base falsa já montada.
func Instala(t testing.TB, raiz string, p Pacote) {
	t.Helper()
	if p.Versao == "" {
		p.Versao = "1.0-1"
	}
	dir := filepath.Join(raiz, "local", p.Nome+"-"+p.Versao)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	var s strings.Builder
	campo := func(chave string, valores ...string) {
		if len(valores) == 0 || (len(valores) == 1 && valores[0] == "") {
			return
		}
		fmt.Fprintf(&s, "%%%s%%\n%s\n\n", chave, strings.Join(valores, "\n"))
	}
	numero := func(chave string, v int64) {
		if v != 0 {
			campo(chave, fmt.Sprint(v))
		}
	}
	campo("NAME", p.Nome)
	campo("VERSION", p.Versao)
	campo("BASE", p.Nome)
	campo("DESC", p.Desc)
	campo("URL", p.URL)
	campo("ARCH", "x86_64")
	numero("BUILDDATE", p.ConstruidoEm)
	numero("INSTALLDATE", p.InstaladoEm)
	campo("PACKAGER", p.Empacotador)
	numero("SIZE", p.Tamanho)
	if p.Dependencia {
		campo("REASON", "1")
	}
	campo("LICENSE", p.Licencas...)
	campo("VALIDATION", "pgp")
	campo("DEPENDS", p.Depende...)
	campo("OPTDEPENDS", p.Opcionais...)
	campo("CONFLICTS", p.Conflita...)
	campo("PROVIDES", p.Fornece...)
	if err := os.WriteFile(filepath.Join(dir, "desc"), []byte(s.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	arquivos := "%FILES%\n" + strings.Join(p.Arquivos, "\n") + "\n\n%BACKUP%\netc/x.conf\td41d8cd98f00b204e9800998ecf8427e\n"
	if err := os.WriteFile(filepath.Join(dir, "files"), []byte(arquivos), 0o644); err != nil {
		t.Fatal(err)
	}
}
