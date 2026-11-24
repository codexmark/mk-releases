package pacman

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"mkpacotes/internal/dubles"
)

func num(v int64) *int64 { return &v }

const saidaSl = `core bash 5.3.20-1 [installed]
core glibc 2.42-1 [installed: 2.41-1]
extra htop 3.5.3-1
extra fd 10.2.0-1
multilib bash 5.0-1
linha torta
`

func TestRepos_OndeEstaCadaPacoteEAVersaoDeLa(t *testing.T) {
	m := dubles.Nova(t)
	m.Comando("pacman", dubles.Regra{Contem: " -Sl ", Texto: saidaSl})
	repos, err := Repos()
	if err != nil {
		t.Fatal(err)
	}
	quer := map[string]Sinc{"bash": {"core", "5.3.20-1"}, "glibc": {"core", "2.42-1"}, "htop": {"extra", "3.5.3-1"}, "fd": {"extra", "10.2.0-1"}}
	if !reflect.DeepEqual(repos, quer) {
		t.Errorf("veio %v\nquer %v (com o mesmo nome em dois repositórios vale o primeiro)", repos, quer)
	}
	if c := m.Chamadas("pacman"); !reflect.DeepEqual(c, []string{"-Sl"}) {
		t.Errorf("chamou %v", c)
	}
}

func TestRepos_SemOPacmanEErroEmFrase(t *testing.T) {
	dubles.Nova(t)
	if _, err := Repos(); err == nil || !strings.Contains(err.Error(), "não achei o pacman") {
		t.Errorf("erro: %v", err)
	}
}

func TestRoda_SempreEmInglesParaASaidaSerLida(t *testing.T) {
	m := dubles.Nova(t)
	t.Setenv("LC_ALL", "pt_BR.UTF-8")
	t.Setenv("LANG", "pt_BR.UTF-8")
	m.Comando("pacman", dubles.Regra{Contem: ""})
	if _, err := Roda("pacman", time.Second, "-Sl"); err != nil {
		t.Fatal(err)
	}
	if amb := m.Ambiente("pacman"); !strings.Contains(amb, "LC_ALL=C\n") || strings.Contains(amb, "pt_BR") {
		t.Errorf("o pacman tem de rodar com LC_ALL=C e sem o idioma da sessão:\n%s", amb)
	}
}

func TestRoda_ErroVemComAPrimeiraLinhaDoStderrEOCodigo(t *testing.T) {
	m := dubles.Nova(t)
	m.Comando("pacman", dubles.Regra{Contem: "", Codigo: 1, Erro: "error:   target not found: nada\noutra linha\n"})
	_, err := Roda("pacman", time.Second, "-S", "--print", "--", "nada")
	if err == nil || err.Error() != "error: target not found: nada" || Codigo(err) != 1 {
		t.Errorf("erro %v, código %d", err, Codigo(err))
	}
	m.Comando("pacman", dubles.Regra{Contem: "", Codigo: 3})
	if _, err := Roda("pacman", time.Second, "-x"); err == nil || !strings.Contains(err.Error(), "saiu com o código 3") {
		t.Errorf("erro %v", err)
	}
}

func TestBusca_DuasLinhasPorPacote(t *testing.T) {
	m := dubles.Nova(t)
	m.Comando("pacman", dubles.Regra{Contem: " -Ss ", Texto: `extra/htop 3.5.3-1 [installed]
    Interactive process viewer
extra/bashtop 0.9.25-3 (grupo-x) [installed: 0.9-1]
    Linux resource monitor
core/sem-descricao 1-1
extra/btop 1.4.5-1
    A monitor of resources
`})
	achados, err := Busca([]string{"top", "monitor"})
	if err != nil {
		t.Fatal(err)
	}
	quer := []Achado{{"htop", "3.5.3-1", "Interactive process viewer", "extra"}, {"bashtop", "0.9.25-3", "Linux resource monitor", "extra"},
		{"sem-descricao", "1-1", "", "core"}, {"btop", "1.4.5-1", "A monitor of resources", "extra"}}
	if !reflect.DeepEqual(achados, quer) {
		t.Errorf("veio %+v", achados)
	}
	if c := m.Chamadas("pacman"); !reflect.DeepEqual(c, []string{"-Ss -- top monitor"}) {
		t.Errorf("as palavras vão depois de --: %v", c)
	}
}

func TestBusca_NadaAchadoNaoEErro(t *testing.T) {
	m := dubles.Nova(t)
	m.Comando("pacman", dubles.Regra{Contem: " -Ss ", Codigo: 1})
	achados, err := Busca([]string{"xyzzy"})
	if err != nil || len(achados) != 0 {
		t.Errorf("veio %v, %v", achados, err)
	}
	m.Comando("pacman", dubles.Regra{Contem: " -Ss ", Codigo: 1, Erro: "error: failed to initialize alpm library\n"})
	if _, err := Busca([]string{"x"}); err == nil {
		t.Errorf("erro de verdade tem de aparecer")
	}
}

const saidaSi = `Repository      : core
Name            : bash
Version         : 5.3.20-1
Description     : The GNU Bourne Again shell
Architecture    : x86_64
URL             : https://www.gnu.org/software/bash/bash.html
Licenses        : GPL-3.0-or-later  MIT
Groups          : None
Provides        : sh
Depends On      : readline  libreadline.so=8-64  glibc
Optional Deps   : bash-completion: for tab completion
                  outro: mais uma [installed]
Conflicts With  : None
Replaces        : None
Download Size   : 1954.93 KiB
Installed Size  : 9.59 MiB
Packager        : Tobias Powalowski <tpowa@archlinux.org>
Build Date      : Tue Sep 15 15:47:19 2026
Validated By    : SHA-256 Sum  Signature

Repository      : multilib
Name            : bash
Version         : 5.0-1
`

func TestInfo_AFichaDeUmPacoteDoRepositorio(t *testing.T) {
	m := dubles.Nova(t)
	t.Setenv("TZ", "UTC")
	m.Comando("pacman", dubles.Regra{Contem: " -Si -- bash ", Texto: saidaSi},
		dubles.Regra{Contem: " -Si ", Codigo: 1, Erro: "error: package 'nada' was not found\n"})
	f, err := Info("bash")
	if err != nil {
		t.Fatal(err)
	}
	quando := time.Date(2026, 9, 15, 15, 47, 19, 0, time.Local).Unix()
	quer := &Ficha{Repo: "core", Nome: "bash", Versao: "5.3.20-1", Descricao: "The GNU Bourne Again shell",
		URL: "https://www.gnu.org/software/bash/bash.html", Empacotador: "Tobias Powalowski <tpowa@archlinux.org>",
		Licencas: []string{"GPL-3.0-or-later", "MIT"}, Depende: []string{"readline", "libreadline.so=8-64", "glibc"},
		Opcionais: []string{"bash-completion: for tab completion", "outro: mais uma"}, Fornece: []string{"sh"},
		Baixar: num(2001848), Tamanho: num(10055844), ConstruidoEm: &quando}
	if !reflect.DeepEqual(f, quer) {
		t.Errorf("veio %+v\nquer %+v", f, quer)
	}
	if f, err := Info("nada"); f != nil || err != nil {
		t.Errorf("pacote que o repositório não tem: nil sem erro; veio %v, %v", f, err)
	}
}

func TestEnsaioInstalar_OQueEntraSemInstalarNada(t *testing.T) {
	m := dubles.Nova(t)
	m.Comando("pacman", dubles.Regra{Contem: " -S ", Texto: "libnl 3.11.0-1 core 380000\nhtop 3.5.3-1 extra 184319\n there is nothing to do\n",
		Erro: "warning: bash-5.3.20-1 is up to date -- skipping\n"})
	itens, err := EnsaioInstalar([]string{"htop", "bash"})
	if err != nil {
		t.Fatal(err)
	}
	if quer := []Item{{"libnl", "3.11.0-1", "core", 380000}, {"htop", "3.5.3-1", "extra", 184319}}; !reflect.DeepEqual(itens, quer) {
		t.Errorf("veio %+v", itens)
	}
	if c := m.Chamadas("pacman"); !reflect.DeepEqual(c, []string{"-S --needed --noconfirm --print --print-format %n %v %r %s -- htop bash"}) {
		t.Errorf("chamou %v", c)
	}
}

func TestEnsaioRemover_OQueSaiComAsDependenciasQueSobram(t *testing.T) {
	m := dubles.Nova(t)
	m.Comando("pacman", dubles.Regra{Contem: " -Rs ", Texto: "libnl 3.11.0-1\nhtop 3.5.3-1\n"})
	itens, err := EnsaioRemover([]string{"htop"})
	if err != nil {
		t.Fatal(err)
	}
	if quer := []Item{{Nome: "libnl", Versao: "3.11.0-1"}, {Nome: "htop", Versao: "3.5.3-1"}}; !reflect.DeepEqual(itens, quer) {
		t.Errorf("veio %+v", itens)
	}
	// -Rns não aceita --print (o pacman recusa --nosave com --print): o ensaio é -Rs.
	if c := m.Chamadas("pacman"); !reflect.DeepEqual(c, []string{"-Rs --noconfirm --print --print-format %n %v -- htop"}) {
		t.Errorf("chamou %v", c)
	}
}

func TestEnsaio_NuncaLevaOpcaoNoLugarDeNome(t *testing.T) {
	m := dubles.Nova(t)
	m.Comando("pacman", dubles.Regra{Contem: ""})
	for _, nomes := range [][]string{{"-Syu"}, {"htop", "--noconfirm"}, {"a b"}, {}} {
		if _, err := EnsaioInstalar(nomes); err == nil {
			t.Errorf("EnsaioInstalar(%q) devia recusar", nomes)
		}
		if _, err := EnsaioRemover(nomes); err == nil {
			t.Errorf("EnsaioRemover(%q) devia recusar", nomes)
		}
	}
	if _, err := Info("--help"); err == nil {
		t.Errorf("Info devia recusar")
	}
	if c := m.Chamadas("pacman"); len(c) != 0 {
		t.Errorf("chegou ao pacman: %v", c)
	}
}

func TestAtualizacoes_PeloCheckupdatesQuandoHa(t *testing.T) {
	m := dubles.Nova(t)
	m.Comando("checkupdates", dubles.Regra{Contem: "", Texto: "htop 3.5.2-1 -> 3.5.3-1\nglibc 2.41-1 -> 2.42-1\nlinha torta\n"})
	m.Comando("pacman", dubles.Regra{Contem: ""})
	mud, fonte, err := Atualizacoes()
	if err != nil {
		t.Fatal(err)
	}
	if quer := []Mudanca{{"htop", "3.5.2-1", "3.5.3-1"}, {"glibc", "2.41-1", "2.42-1"}}; !reflect.DeepEqual(mud, quer) || fonte != "checkupdates" {
		t.Errorf("veio %+v de %s", mud, fonte)
	}
	if len(m.Chamadas("pacman")) != 0 {
		t.Errorf("com o checkupdates respondendo, o pacman não é chamado")
	}
}

func TestAtualizacoes_CheckupdatesSemNovidadeSaiCom2(t *testing.T) {
	m := dubles.Nova(t)
	m.Comando("checkupdates", dubles.Regra{Contem: "", Codigo: 2})
	mud, fonte, err := Atualizacoes()
	if err != nil || len(mud) != 0 || fonte != "checkupdates" {
		t.Errorf("veio %v, %s, %v", mud, fonte, err)
	}
}

func TestAtualizacoes_SemCheckupdatesOuComEleFalhandoValeOPacman(t *testing.T) {
	m := dubles.Nova(t)
	m.Comando("pacman", dubles.Regra{Contem: " -Qu ", Texto: "htop 3.5.2-1 -> 3.5.3-1\nfd 10.1-1 -> 10.2.0-1 [ignored]\n"})
	quer := []Mudanca{{"htop", "3.5.2-1", "3.5.3-1"}, {"fd", "10.1-1", "10.2.0-1"}}
	mud, fonte, err := Atualizacoes()
	if err != nil || !reflect.DeepEqual(mud, quer) || fonte != "pacman" {
		t.Errorf("sem checkupdates: %v, %s, %v", mud, fonte, err)
	}
	m.Comando("checkupdates", dubles.Regra{Contem: "", Codigo: 1, Erro: "==> ERROR: Cannot fetch updates\n"})
	mud, fonte, err = Atualizacoes()
	if err != nil || !reflect.DeepEqual(mud, quer) || fonte != "pacman" {
		t.Errorf("checkupdates sem rede: %v, %s, %v", mud, fonte, err)
	}
	// O pacman -Qu sai com 1 quando não há nada.
	m.Comando("pacman", dubles.Regra{Contem: " -Qu ", Codigo: 1})
	mud, _, err = Atualizacoes()
	if err != nil || len(mud) != 0 {
		t.Errorf("nada para atualizar: %v, %v", mud, err)
	}
}

func TestBytesDe(t *testing.T) {
	casos := map[string]*int64{"1954.93 KiB": num(2001848), "9.59 MiB": num(10055844), "12 B": num(12), "1.50 GiB": num(1610612736),
		"": nil, "None": nil, "muito": nil, "3 XiB": nil}
	for texto, quer := range casos {
		veio := bytesDe(texto)
		if (veio == nil) != (quer == nil) || (veio != nil && *veio != *quer) {
			t.Errorf("bytesDe(%q): veio %v, queria %v", texto, veio, quer)
		}
	}
}
