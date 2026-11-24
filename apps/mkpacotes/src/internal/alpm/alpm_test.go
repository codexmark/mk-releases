package alpm

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"mkpacotes/internal/dubles"
)

func num(v int64) *int64 { return &v }

// Uma máquina pequena: o bash pedido, a glibc e a readline como dependências
// dele, um órfão (libvelha), um fornecedor (dash fornece sh) e um do AUR.
func baseFalsa(t *testing.T) string {
	t.Helper()
	return dubles.Base(t,
		dubles.Pacote{Nome: "glibc", Versao: "2.42-1", Desc: "GNU C Library", Tamanho: 50000000, InstaladoEm: 1792000000, Dependencia: true},
		dubles.Pacote{Nome: "readline", Versao: "8.3-1", Desc: "GNU readline", Tamanho: 800000, InstaladoEm: 1792000001, Dependencia: true,
			Depende: []string{"glibc", "libncursesw.so=6-64"}, Fornece: []string{"libreadline.so=8-64"}},
		dubles.Pacote{Nome: "bash", Versao: "5.3.20-1", Desc: "The GNU Bourne Again shell", URL: "https://www.gnu.org/software/bash/",
			Empacotador: "Fulano <fulano@archlinux.org>", Tamanho: 10055840, InstaladoEm: 1792797774, ConstruidoEm: 1789501639,
			Licencas: []string{"GPL-3.0-or-later"}, Depende: []string{"readline", "libreadline.so=8-64", "glibc>=2.38"},
			Opcionais: []string{"bash-completion: for tab completion"}, Fornece: []string{"sh"},
			Arquivos: []string{"etc/", "etc/bash.bashrc", "usr/", "usr/bin/", "usr/bin/bash", "usr/bin/bashbug", "usr/share/", "usr/share/doc/bash/README"}},
		dubles.Pacote{Nome: "libvelha", Versao: "1:0.9-2", Desc: "Sobrou", Tamanho: 1000, Dependencia: true, Depende: []string{"glibc"}},
		dubles.Pacote{Nome: "bash-completion", Versao: "2.16-1", Desc: "Completar", Dependencia: true, Depende: []string{"bash"}},
		dubles.Pacote{Nome: "script-x", Versao: "1-1", Desc: "Usa um sh qualquer", Depende: []string{"sh"}},
	)
}

func TestLe_UmPacoteInteiro(t *testing.T) {
	baseFalsa(t)
	b, err := Le()
	if err != nil {
		t.Fatal(err)
	}
	quer := &Pacote{Nome: "bash", Versao: "5.3.20-1", Descricao: "The GNU Bourne Again shell", URL: "https://www.gnu.org/software/bash/",
		Empacotador: "Fulano <fulano@archlinux.org>", Tamanho: 10055840, InstaladoEm: num(1792797774), ConstruidoEm: num(1789501639),
		Pedido: true, Licencas: []string{"GPL-3.0-or-later"}, Depende: []string{"readline", "libreadline.so=8-64", "glibc>=2.38"},
		Opcionais: []string{"bash-completion: for tab completion"}, Fornece: []string{"sh"}}
	veio := *b.Pacotes["bash"]
	veio.pasta = ""
	if !reflect.DeepEqual(&veio, quer) {
		t.Errorf("bash:\n veio %+v\n quer %+v", &veio, quer)
	}
	if b.Pacotes["glibc"].Pedido || b.Pacotes["glibc"].InstaladoEm == nil {
		t.Errorf("glibc veio como dependência e tem data: %+v", b.Pacotes["glibc"])
	}
	if d := b.Pacotes["libvelha"]; d.InstaladoEm != nil || d.ConstruidoEm != nil {
		t.Errorf("data que o desc não tem é nil, não zero: %+v", d)
	}
	if quer := []string{"bash", "bash-completion", "glibc", "libvelha", "readline", "script-x"}; !reflect.DeepEqual(b.Nomes(), quer) {
		t.Errorf("Nomes: %v", b.Nomes())
	}
}

func TestLe_SemABaseEErroEmFrase(t *testing.T) {
	t.Setenv("MKPACOTES_PACMAN_DB", filepath.Join(t.TempDir(), "nada"))
	if _, err := Le(); err == nil || !strings.Contains(err.Error(), "não achei a base do pacman") {
		t.Errorf("erro: %v", err)
	}
}

func TestLe_PastaQuebradaOuSemNomeEIgnorada(t *testing.T) {
	raiz := baseFalsa(t)
	for _, d := range []string{"torto-1-1", "vazio-1-1"} {
		if err := os.MkdirAll(filepath.Join(raiz, "local", d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(raiz, "local", "vazio-1-1", "desc"), []byte("%DESC%\nsem nome\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	b, err := Le()
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Pacotes) != 6 {
		t.Errorf("devia ter 6 pacotes, tem %d", len(b.Pacotes))
	}
}

func TestSatisfeito_PeloNomeOuPorQuemFornece(t *testing.T) {
	baseFalsa(t)
	b, _ := Le()
	for dep, quer := range map[string]bool{"glibc": true, "glibc>=2.38": true, "sh": true, "libreadline.so=8-64": true,
		"libreadline.so": true, "bash-completion": true, "zsh": false, "libncursesw.so=6-64": false} {
		if b.Satisfeito(dep) != quer {
			t.Errorf("Satisfeito(%q) devia ser %v", dep, quer)
		}
	}
}

func TestRequeridoPor_ContaONomeEOQueOPacoteFornece(t *testing.T) {
	baseFalsa(t)
	b, _ := Le()
	casos := map[string][]string{
		"glibc":           {"bash", "libvelha", "readline"},
		"readline":        {"bash"},
		"bash":            {"bash-completion", "script-x"}, // script-x pede "sh", que o bash fornece
		"libvelha":        nil,
		"bash-completion": nil,
		"nao-existe":      nil,
	}
	for nome, quer := range casos {
		if veio := b.RequeridoPor(nome); !reflect.DeepEqual(veio, quer) {
			t.Errorf("RequeridoPor(%s): veio %v, queria %v", nome, veio, quer)
		}
	}
	if veio := b.OpcionalPara("bash-completion"); !reflect.DeepEqual(veio, []string{"bash"}) {
		t.Errorf("OpcionalPara: %v", veio)
	}
}

func TestOrfaos_DependenciaQueNinguemPede(t *testing.T) {
	baseFalsa(t)
	b, _ := Le()
	// bash-completion é dependência e ninguém depende dela, mas o bash a lista
	// como opcional: não é órfã, como no pacman -Qdt.
	if quer := []string{"libvelha"}; !reflect.DeepEqual(b.Orfaos(), quer) {
		t.Errorf("Orfaos: %v", b.Orfaos())
	}
	if b.Orfao("glibc") || b.Orfao("bash") || b.Orfao("nao-existe") || b.Orfao("bash-completion") || !b.Orfao("libvelha") {
		t.Errorf("Orfao: glibc é pedida pelo bash, bash foi pedido pela pessoa")
	}
}

func TestOrfaos_QuemDependeDeSiMesmoNaoSeSalva(t *testing.T) {
	dubles.Base(t, dubles.Pacote{Nome: "umbigo", Dependencia: true, Depende: []string{"umbigo"}})
	b, _ := Le()
	if !b.Orfao("umbigo") {
		t.Errorf("um pacote não é dependência de si mesmo")
	}
}

func TestArquivos_ContaSoArquivosEAchaOsExecutaveis(t *testing.T) {
	baseFalsa(t)
	b, _ := Le()
	total, execs, err := b.Arquivos("bash")
	if err != nil {
		t.Fatal(err)
	}
	if total != 4 || !reflect.DeepEqual(execs, []string{"/usr/bin/bash", "/usr/bin/bashbug"}) {
		t.Errorf("total %d, executáveis %v", total, execs)
	}
	if total, execs, err := b.Arquivos("glibc"); err != nil || total != 0 || len(execs) != 0 {
		t.Errorf("pacote sem arquivo: %d %v %v", total, execs, err)
	}
	if _, _, err := b.Arquivos("nao-existe"); err == nil {
		t.Errorf("pacote que não existe devia dar erro")
	}
}

func TestValidaNome(t *testing.T) {
	for _, n := range []string{"bash", "gtk4", "libc++", "python-pip", "r8168-dkms", "lib32-glibc", "kde.x", "a@b", "9base", "x"} {
		if err := ValidaNome(n); err != nil {
			t.Errorf("%q devia passar: %v", n, err)
		}
	}
	for _, n := range []string{"", "-Syu", "--noconfirm", "-", ".", "..", ".oculto", "a b", "a;b", "a/b", "../x", "Bash", "a$b", "a\nb",
		"a|b", "a>b", "a=1", "ação", "a\x00b", "a`b`", "$(x)", "a,b", strings.Repeat("a", 257)} {
		if err := ValidaNome(n); err == nil {
			t.Errorf("%q devia ser recusado", n)
		} else if !strings.Contains(err.Error(), "não é nome de pacote") {
			t.Errorf("%q: frase %q", n, err)
		}
	}
}

func TestNomeDe_TiraARestricaoDeVersao(t *testing.T) {
	for dep, quer := range map[string]string{"glibc": "glibc", "glibc>=2.38": "glibc", "sh=1": "sh", "a<2": "a", "a<=2": "a",
		"libreadline.so=8-64": "libreadline.so", "bash-completion: for tab": "bash-completion", "": ""} {
		if veio := NomeDe(dep); veio != quer {
			t.Errorf("NomeDe(%q) = %q, queria %q", dep, veio, quer)
		}
	}
}

func TestVercmp_ComoODoPacman(t *testing.T) {
	// Casos da suíte do próprio pacman (test/util/vercmptest.sh).
	casos := []struct {
		a, b string
		quer int
	}{
		{"1.5.0", "1.5.0", 0}, {"1.5.1", "1.5.0", 1}, {"1.5.1", "1.5", 1}, {"1.5.0-1", "1.5.0-1", 0},
		{"1.5.0-1", "1.5.0-2", -1}, {"1.5.0-1", "1.5.1-1", -1}, {"1.5.0-2", "1.5.1-1", -1},
		{"1.5-1", "1.5.1-1", -1}, {"1.5-2", "1.5.1-1", -1}, {"1.5-2", "1.5.1-2", -1},
		{"1.5", "1.5-1", 0}, {"1.5-1", "1.5", 0}, {"1.1-1", "1.1", 0}, {"1.0-1", "1.1", -1}, {"1.1-1", "1.0", 1},
		{"1.5b-1", "1.5-1", -1}, {"1.5b", "1.5", -1}, {"1.5b-1", "1.5", -1}, {"1.5b", "1.5.1", -1},
		{"1.0a", "1.0alpha", -1}, {"1.0alpha", "1.0b", -1}, {"1.0b", "1.0beta", -1}, {"1.0beta", "1.0rc", -1}, {"1.0rc", "1.0", -1},
		{"1.5.a", "1.5", 1}, {"1.5.b", "1.5.a", 1}, {"1.5.1", "1.5.b", 1},
		{"1.5.b-1", "1.5.b", 0}, {"1.5-1", "1.5.b", -1},
		{"2.0", "2_0", 0}, {"2.0_a", "2_0.a", 0}, {"2.0a", "2.0.a", -1}, {"2___a", "2_a", 1},
		{"0:1.0", "0:1.0", 0}, {"0:1.0", "0:1.1", -1}, {"1:1.0", "0:1.0", 1}, {"1:1.0", "0:1.1", 1}, {"1:1.0", "2:1.1", -1},
		{"1:1.0", "0:1.0-1", 1}, {"1:1.0-1", "0:1.1-1", 1},
		{"0:1.0", "1.0", 0}, {"0:1.0", "1.1", -1}, {"0:1.1", "1.0", 1}, {"1:1.0", "1.0", 1}, {"1:1.0", "1.1", 1}, {"1:1.1", "1.1", 1},
		{"1.010", "1.9", 1}, {"1.09", "1.9", 0}, {"20261019-1", "20260909-1", 1}, {"7.2.8-9", "7.2.8-10", -1},
	}
	for _, c := range casos {
		if veio := Vercmp(c.a, c.b); veio != c.quer {
			t.Errorf("Vercmp(%q, %q) = %d, queria %d", c.a, c.b, veio, c.quer)
		}
		if veio := Vercmp(c.b, c.a); veio != -c.quer {
			t.Errorf("Vercmp(%q, %q) = %d, queria %d (o inverso)", c.b, c.a, veio, -c.quer)
		}
	}
}
