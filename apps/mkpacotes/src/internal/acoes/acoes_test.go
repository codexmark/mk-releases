package acoes

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"mkpacotes/internal/dubles"
)

func num(v int64) *int64   { return &v }
func txt(v string) *string { return &v }

// maquina monta uma máquina de mentira inteira: a base do pacman, os
// repositórios (pacman -Sl), o Mk com a lista de base, o cache, a trava do
// coletor e um AUR que conhece o "aurzinho". Nenhum comando de verdade existe.
func maquina(t *testing.T) *dubles.Mesa {
	t.Helper()
	m := dubles.Nova(t)
	dubles.Base(t,
		dubles.Pacote{Nome: "glibc", Versao: "2.42-1", Tamanho: 50000000, Dependencia: true},
		dubles.Pacote{Nome: "bash", Versao: "5.3.20-1", Tamanho: 10000000, Depende: []string{"glibc"}, Fornece: []string{"sh"}},
		dubles.Pacote{Nome: "libnl", Versao: "3.11.0-1", Tamanho: 1500000, Dependencia: true, Depende: []string{"glibc"}},
		dubles.Pacote{Nome: "htop", Versao: "3.5.3-1", Tamanho: 400000, Depende: []string{"libnl", "glibc"}},
		dubles.Pacote{Nome: "libvelha", Versao: "0.9-2", Tamanho: 1000, Dependencia: true},
		dubles.Pacote{Nome: "mkinitcpio", Versao: "40-1", Tamanho: 2000, Dependencia: true},
		dubles.Pacote{Nome: "do-aur", Versao: "1.0-1", Tamanho: 3000},
		dubles.Pacote{Nome: "usa-htop", Versao: "1-1", Depende: []string{"htop"}},
	)
	mk := t.TempDir()
	if err := os.MkdirAll(filepath.Join(mk, "system", "packages"), 0o755); err != nil {
		t.Fatal(err)
	}
	lista := "# Sistema base\nbase\nmkinitcpio   # gera o initramfs\n\n  man-db\n-Syu\n"
	if err := os.WriteFile(filepath.Join(mk, "system", "packages", "base.packages"), []byte(lista), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MKPACOTES_MK", mk)
	t.Setenv("MKPACOTES_CACHE", t.TempDir())
	t.Setenv("MKPACOTES_TRAVA", filepath.Join(t.TempDir(), "trava"))
	t.Setenv("MKPACOTES_TERMINAL", "")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m.Comando("pacman",
		dubles.Regra{Contem: " -Sl ", Texto: "core glibc 2.42-1 [installed]\ncore bash 5.3.20-1 [installed]\ncore libnl 3.11.0-1 [installed]\n" +
			"extra htop 3.5.3-1 [installed]\nextra fd 10.2.0-1\nextra ripgrep 15.0-1\ncore mkinitcpio 40-1\n"},
		dubles.Regra{Contem: " -S --needed --noconfirm --print --print-format %n %v %r %s -- fd ripgrep ", Texto: "oniguruma 6.9-1 extra 300000\nfd 10.2.0-1 extra 1200000\nripgrep 15.0-1 extra 1500000\n"},
		dubles.Regra{Contem: " -S --needed --noconfirm --print --print-format %n %v %r %s -- fd ", Texto: "fd 10.2.0-1 extra 1200000\n"},
		dubles.Regra{Contem: " -S --needed --noconfirm --print --print-format %n %v %r %s -- bash ", Texto: " there is nothing to do\n"},
		dubles.Regra{Contem: " -Rs --noconfirm --print --print-format %n %v -- usa-htop htop ", Texto: "usa-htop 1-1\nhtop 3.5.3-1\nlibnl 3.11.0-1\n"},
		dubles.Regra{Contem: " -Rs --noconfirm --print --print-format %n %v -- do-aur ", Texto: "do-aur 1.0-1\n"},
		dubles.Regra{Contem: " -Rs --noconfirm --print --print-format %n %v -- libvelha ", Texto: "libvelha 0.9-2\n"},
		dubles.Regra{Contem: " -Rs --noconfirm --print --print-format %n %v -- mkinitcpio ", Texto: "mkinitcpio 40-1\n"},
		dubles.Regra{Contem: " -Rs --noconfirm --print --print-format %n %v -- usa-htop ", Texto: "usa-htop 1-1\nhtop 3.5.3-1\nlibnl 3.11.0-1\nglibc 2.42-1\n"},
	)
	aurFalso(t, `{"type":"multiinfo","results":[{"Name":"aurzinho","Version":"2.0-1","Depends":["glibc","libfalta>=2"],"MakeDepends":["go"],"NumVotes":7}]}`)
	return m
}

func aurFalso(t *testing.T, corpo string) {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.RawQuery, "aurzinho") || strings.Contains(r.URL.RawQuery, "do-aur") {
			fmt.Fprint(w, corpo)
			return
		}
		fmt.Fprint(w, `{"type":"multiinfo","results":[]}`)
	}))
	t.Cleanup(s.Close)
	t.Setenv("MKPACOTES_AUR", s.URL)
}

func recusado(err error) bool {
	var r *Recusa
	return errors.As(err, &r)
}

// ── a segurança ────────────────────────────────────────────────────────

func TestProtegidos_OsFixosEOQueOMkDeclaraComoBase(t *testing.T) {
	maquina(t)
	p := Protegidos()
	for _, n := range []string{"base", "linux", "linux-firmware", "systemd", "pacman", "glibc", "sudo", "limine", "hyprland", "quickshell", "sddm",
		"mkinitcpio", "man-db"} {
		if !p[n] {
			t.Errorf("%s devia ser protegido", n)
		}
	}
	for _, n := range []string{"htop", "bash", "-Syu", "#", ""} {
		if p[n] {
			t.Errorf("%q não devia ser protegido", n)
		}
	}
	// Sem o Mk instalado (ou sem a lista), ficam os fixos.
	t.Setenv("MKPACOTES_MK", filepath.Join(t.TempDir(), "nada"))
	if p := Protegidos(); !p["glibc"] || p["mkinitcpio"] || len(p) != 11 {
		t.Errorf("sem a lista do Mk: %v", p)
	}
}

func TestRemover_RecusaOQueDerrubaOSistema(t *testing.T) {
	m := maquina(t)
	for _, nomes := range [][]string{{"mkinitcpio"}} {
		_, err := Planeja("remover", nomes, false)
		if !recusado(err) || !strings.Contains(err.Error(), "mkinitcpio") || !strings.Contains(err.Error(), "--mesmo-assim") {
			t.Errorf("remover %v: %v", nomes, err)
		}
		if _, err := Age("remover", nomes, false, func(string) {}); !recusado(err) {
			t.Errorf("agir remover %v: %v", nomes, err)
		}
	}
	if c := m.Chamadas("pkexec"); len(c) != 0 {
		t.Errorf("chegou ao pkexec: %v", c)
	}
}

func TestRemover_RecusaTambemQuandoOProtegidoSaiDeCarona(t *testing.T) {
	m := maquina(t)
	// Tirando o htop de quem o usa, o ensaio leva junto a glibc (protegida).
	_, err := Planeja("remover", []string{"usa-htop"}, false)
	if !recusado(err) || !strings.Contains(err.Error(), "glibc") {
		t.Errorf("devia recusar pela glibc: %v", err)
	}
	if _, err := Age("remover", []string{"usa-htop"}, false, func(string) {}); !recusado(err) {
		t.Errorf("agir: %v", err)
	}
	if len(m.Chamadas("pkexec")) != 0 {
		t.Errorf("chegou ao pkexec")
	}
}

func TestRemover_ComMesmoAssimOPlanoSaiComOAviso(t *testing.T) {
	maquina(t)
	p, err := Planeja("remover", []string{"mkinitcpio"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Avisos) != 1 || !strings.Contains(p.Avisos[0], "mkinitcpio") || len(p.Sai) != 1 {
		t.Errorf("plano: %+v", p)
	}
}

func TestNomes_SoNomeDePacoteChegaAUmComando(t *testing.T) {
	m := maquina(t)
	m.Comando("pkexec", dubles.Regra{Contem: ""})
	m.Comando("yay", dubles.Regra{Contem: ""})
	antes := len(m.Chamadas("pacman"))
	for _, acao := range []string{"instalar", "remover"} {
		for _, nomes := range [][]string{{"-Syu"}, {"--noconfirm"}, {"htop", "-R"}, {"a b"}, {"a;reboot"}, {"$(reboot)"}, {"../../etc"}, {"Htop"}, {""}, {"--", "htop"}} {
			_, err := Planeja(acao, nomes, false)
			if err == nil || recusado(err) || !strings.Contains(err.Error(), "não é nome de pacote") {
				t.Errorf("plano %s %q: %v", acao, nomes, err)
			}
			if _, err := Age(acao, nomes, true, func(string) {}); err == nil {
				t.Errorf("agir %s %q devia falhar", acao, nomes)
			}
		}
	}
	if len(m.Chamadas("pacman")) != antes || len(m.Chamadas("pkexec")) != 0 || len(m.Chamadas("yay")) != 0 {
		t.Errorf("algum comando rodou: pacman %v, pkexec %v, yay %v", m.Chamadas("pacman"), m.Chamadas("pkexec"), m.Chamadas("yay"))
	}
}

func TestAcao_DesconhecidaOuComArgumentoErrado(t *testing.T) {
	maquina(t)
	casos := []struct {
		acao  string
		nomes []string
		frase string
	}{
		{"explodir", nil, "não conheço a ação"}, {"instalar", nil, "faltou dizer o pacote"}, {"remover", nil, "faltou dizer o pacote"},
		{"orfaos", []string{"htop"}, "não leva pacote"}, {"cache", []string{"htop"}, "não leva pacote"}, {"atualizar", []string{"htop"}, "não leva pacote"},
	}
	for _, c := range casos {
		if _, err := Planeja(c.acao, c.nomes, false); err == nil || !strings.Contains(err.Error(), c.frase) {
			t.Errorf("%s %v: %v", c.acao, c.nomes, err)
		}
	}
}

func TestAgir_RecusaComOPacmanTravado(t *testing.T) {
	m := maquina(t)
	m.Comando("pkexec", dubles.Regra{Contem: ""})
	trava := filepath.Join(os.Getenv("MKPACOTES_PACMAN_DB"), "db.lck")
	if err := os.WriteFile(trava, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if !Travado() {
		t.Fatalf("Travado devia ver o db.lck")
	}
	for _, acao := range []string{"instalar", "remover", "orfaos", "cache"} {
		nomes := map[string][]string{"instalar": {"fd"}, "remover": {"do-aur"}}[acao]
		_, err := Age(acao, nomes, false, func(string) {})
		if !recusado(err) || !strings.Contains(err.Error(), "o pacman está travado") || !strings.Contains(err.Error(), "db.lck") {
			t.Errorf("%s: %v", acao, err)
		}
	}
	if len(m.Chamadas("pkexec")) != 0 {
		t.Errorf("chegou ao pkexec com o pacman travado")
	}
	// A atualização é do mk-update, que sabe lidar com a trava: essa passa.
	m.Comando("mk-update", dubles.Regra{Contem: ""})
	m.Comando("foot", dubles.Regra{Contem: ""})
	if _, err := Age("atualizar", nil, false, func(string) {}); err != nil {
		t.Errorf("atualizar com a trava: %v", err)
	}
	// O plano sai, avisando.
	p, err := Planeja("remover", []string{"do-aur"}, false)
	if err != nil || !p.Travado {
		t.Errorf("o plano sai e diz que está travado: %+v, %v", p, err)
	}
	os.Remove(trava)
	if Travado() {
		t.Errorf("sem o db.lck não está travado")
	}
}

func TestAgir_UmaAcaoDeCadaVez(t *testing.T) {
	m := maquina(t)
	m.Comando("pkexec", dubles.Regra{Contem: ""})
	solta, err := Tranca()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Tranca(); !recusado(err) {
		t.Errorf("a segunda tranca devia ser recusada: %v", err)
	}
	_, err = Age("remover", []string{"do-aur"}, false, func(string) {})
	if !recusado(err) || !strings.Contains(err.Error(), "já há uma ação") {
		t.Errorf("devia recusar a segunda ação: %v", err)
	}
	if len(m.Chamadas("pkexec")) != 0 {
		t.Errorf("a segunda ação chegou ao pkexec")
	}
	solta()
	if _, err := Age("remover", []string{"do-aur"}, false, func(string) {}); err != nil {
		t.Errorf("depois de solta, a ação roda: %v", err)
	}
	// E a ação que terminou soltou a trava dela.
	if solta, err := Tranca(); err != nil {
		t.Errorf("a ação não soltou a trava: %v", err)
	} else {
		solta()
	}
}

func TestAgir_NuncaComoRoot(t *testing.T) {
	m := maquina(t)
	m.Comando("pkexec", dubles.Regra{Contem: ""})
	m.Comando("yay", dubles.Regra{Contem: ""})
	antes := Euid
	Euid = func() int { return 0 }
	t.Cleanup(func() { Euid = antes })
	_, err := Age("instalar", []string{"aurzinho"}, false, func(string) {})
	if !recusado(err) || !strings.Contains(err.Error(), "root") {
		t.Errorf("devia recusar como root: %v", err)
	}
	if len(m.Chamadas("yay")) != 0 || len(m.Chamadas("pkexec")) != 0 {
		t.Errorf("rodou algo como root")
	}
}

// ── o plano ────────────────────────────────────────────────────────────

func TestPlano_InstalarDosRepositorios(t *testing.T) {
	maquina(t)
	p, err := Planeja("instalar", []string{"fd", "ripgrep", "fd"}, false)
	if err != nil {
		t.Fatal(err)
	}
	quer := &Plano{Acao: "instalar", Pedidos: []string{"fd", "ripgrep"},
		Entra: []Entra{{"oniguruma", txt("6.9-1"), txt("extra"), num(300000), false}, {"fd", txt("10.2.0-1"), txt("extra"), num(1200000), true},
			{"ripgrep", txt("15.0-1"), txt("extra"), num(1500000), true}},
		Sai: []Sai{}, Baixar: num(3000000),
		Comandos: [][]string{{"pkexec", "pacman", "-S", "--needed", "--noconfirm", "--", "fd", "ripgrep"}}, Avisos: []string{}}
	if !reflect.DeepEqual(p, quer) {
		t.Errorf("veio %s\nquer %s", mostra(p), mostra(quer))
	}
}

func mostra(p *Plano) string {
	s := fmt.Sprintf("%+v", *p)
	for _, e := range p.Entra {
		v, o, b := "nil", "nil", "nil"
		if e.Versao != nil {
			v = *e.Versao
		}
		if e.Origem != nil {
			o = *e.Origem
		}
		if e.Baixar != nil {
			b = fmt.Sprint(*e.Baixar)
		}
		s += fmt.Sprintf("\n  entra %s %s %s %s %v", e.Nome, v, o, b, e.Pedido)
	}
	return s
}

func TestPlano_InstalarOQueJaEstaEmDiaENadaAFazer(t *testing.T) {
	maquina(t)
	p, err := Planeja("instalar", []string{"bash"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !p.NadaAFazer || len(p.Entra) != 0 || len(p.Comandos) != 0 || len(p.Avisos) != 1 || !strings.Contains(p.Avisos[0], "bash já está instalado") {
		t.Errorf("plano: %s", mostra(p))
	}
}

func TestPlano_InstalarDoAURVaiPeloYayEMostraOQueFalta(t *testing.T) {
	m := maquina(t)
	m.Comando("yay", dubles.Regra{Contem: ""})
	p, err := Planeja("instalar", []string{"aurzinho", "fd"}, false)
	if err != nil {
		t.Fatal(err)
	}
	querEntra := []Entra{{"fd", txt("10.2.0-1"), txt("extra"), num(1200000), true}, {"aurzinho", txt("2.0-1"), txt("aur"), nil, true},
		{"libfalta", nil, nil, nil, false}, {"go", nil, nil, nil, false}}
	querComandos := [][]string{{"pkexec", "pacman", "-S", "--needed", "--noconfirm", "--", "fd"},
		{"yay", "-S", "--needed", "--noconfirm", "--sudo", "pkexec", "--", "aurzinho"}}
	if !reflect.DeepEqual(p.Entra, querEntra) || !reflect.DeepEqual(p.Comandos, querComandos) || p.Baixar != nil {
		t.Errorf("plano: %s", mostra(p))
	}
	if len(p.Avisos) != 1 || !strings.Contains(p.Avisos[0], "AUR") {
		t.Errorf("devia avisar que o AUR compila aqui: %v", p.Avisos)
	}
	if len(m.Chamadas("yay")) != 0 {
		t.Errorf("o plano não roda o yay")
	}
}

func TestPlano_InstalarDoAURSemOYayAvisaEOAgirFalha(t *testing.T) {
	m := maquina(t)
	m.Comando("pkexec", dubles.Regra{Contem: ""})
	p, err := Planeja("instalar", []string{"aurzinho"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Avisos) != 2 || !strings.Contains(p.Avisos[1], "yay") {
		t.Errorf("avisos: %v", p.Avisos)
	}
	if _, err := Age("instalar", []string{"aurzinho"}, false, func(string) {}); err == nil || recusado(err) || !strings.Contains(err.Error(), "não achei o yay") {
		t.Errorf("agir sem yay: %v", err)
	}
	if len(m.Chamadas("pkexec")) != 0 {
		t.Errorf("nada devia rodar")
	}
}

func TestPlano_InstalarOQueNaoExisteEmLugarNenhum(t *testing.T) {
	maquina(t)
	if _, err := Planeja("instalar", []string{"nao-existe"}, false); err == nil || !strings.Contains(err.Error(), "não achei nao-existe nos repositórios nem no AUR") {
		t.Errorf("erro: %v", err)
	}
	// Sem rede não dá para saber se é do AUR: o erro diz isso.
	s := httptest.NewServer(http.NotFoundHandler())
	s.Close()
	t.Setenv("MKPACOTES_AUR", s.URL)
	if _, err := Planeja("instalar", []string{"aurzinho"}, false); err == nil || !strings.Contains(err.Error(), "aurzinho não está nos repositórios") ||
		!strings.Contains(err.Error(), "sem rede") {
		t.Errorf("erro sem rede: %v", err)
	}
	// E os oficiais continuam sem rede.
	if _, err := Planeja("instalar", []string{"fd"}, false); err != nil {
		t.Errorf("oficial sem rede: %v", err)
	}
}

func TestPlano_RemoverMostraAntesOQueMaisSai(t *testing.T) {
	maquina(t)
	p, err := Planeja("remover", []string{"usa-htop", "htop"}, false)
	if err != nil {
		t.Fatal(err)
	}
	quer := &Plano{Acao: "remover", Pedidos: []string{"usa-htop", "htop"}, Entra: []Entra{},
		Sai:     []Sai{{"usa-htop", "1-1", 0, true}, {"htop", "3.5.3-1", 400000, true}, {"libnl", "3.11.0-1", 1500000, false}},
		Liberar: num(1900000), Comandos: [][]string{{"pkexec", "pacman", "-Rns", "--noconfirm", "--", "usa-htop", "htop"}}, Avisos: []string{}}
	if !reflect.DeepEqual(p, quer) {
		t.Errorf("veio %+v\nquer %+v", p, quer)
	}
}

func TestPlano_RemoverOQueNaoEstaInstaladoOuDeQueOutroDepende(t *testing.T) {
	m := maquina(t)
	antes := len(m.Chamadas("pacman"))
	if _, err := Planeja("remover", []string{"fd"}, false); err == nil || !strings.Contains(err.Error(), "fd não está instalado") {
		t.Errorf("erro: %v", err)
	}
	_, err := Planeja("remover", []string{"htop"}, false)
	if err == nil || recusado(err) || !strings.Contains(err.Error(), "usa-htop depende dele") {
		t.Errorf("erro: %v", err)
	}
	_, err = Planeja("remover", []string{"bash", "libnl"}, false)
	if err == nil || !strings.Contains(err.Error(), "htop depende") {
		t.Errorf("erro: %v", err)
	}
	if len(m.Chamadas("pacman")) != antes {
		t.Errorf("essas contas não precisam do pacman")
	}
}

func TestPlano_OrfaosDeixaOsProtegidosDeFora(t *testing.T) {
	maquina(t)
	p, err := Planeja("orfaos", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	// libvelha e mkinitcpio são órfãos; o mkinitcpio é da base do Mk e fica.
	if !reflect.DeepEqual(p.Pedidos, []string{"libvelha"}) || len(p.Sai) != 1 || *p.Liberar != 1000 ||
		!reflect.DeepEqual(p.Comandos, [][]string{{"pkexec", "pacman", "-Rns", "--noconfirm", "--", "libvelha"}}) {
		t.Errorf("plano: %+v", p)
	}
	if len(p.Avisos) != 1 || !strings.Contains(p.Avisos[0], "mkinitcpio") {
		t.Errorf("avisos: %v", p.Avisos)
	}
}

func TestPlano_OrfaosSemNenhum(t *testing.T) {
	dubles.Nova(t)
	dubles.Base(t, dubles.Pacote{Nome: "bash"})
	if _, err := Planeja("orfaos", nil, false); err == nil || recusado(err) || !strings.Contains(err.Error(), "não há pacote órfão") {
		t.Errorf("erro: %v", err)
	}
}

func encheCache(t *testing.T) {
	t.Helper()
	arquivos := map[string]int{
		"bash-5.3.20-1-x86_64.pkg.tar.zst": 1000, "bash-5.3.20-1-x86_64.pkg.tar.zst.sig": 10, // instalado: fica
		"bash-5.3.19-1-x86_64.pkg.tar.zst": 900,                                          // versão velha: sai
		"fd-10.2.0-1-x86_64.pkg.tar.zst":   500, "fd-10.2.0-1-x86_64.pkg.tar.zst.sig": 5, // não instalado: sai
		"do-aur-1.0-1-any.pkg.tar.xz": 300, // instalado: fica
		"anotacao.txt":                77,  // não é pacote: não conta
	}
	for nome, tamanho := range arquivos {
		if err := os.WriteFile(filepath.Join(os.Getenv("MKPACOTES_CACHE"), nome), make([]byte, tamanho), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(os.Getenv("MKPACOTES_CACHE"), "download-abc"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestPlano_CacheDizQuantoSolta(t *testing.T) {
	maquina(t)
	encheCache(t)
	p, err := Planeja("cache", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(p.Cache, &Cache{Arquivos: 6, Bytes: 2715, Sobram: 3}) || *p.Liberar != 1405 || p.NadaAFazer ||
		!reflect.DeepEqual(p.Comandos, [][]string{{"pkexec", "pacman", "-Sc", "--noconfirm"}}) {
		t.Errorf("plano: %+v cache %+v liberar %d", p, p.Cache, *p.Liberar)
	}
}

func TestPlano_CacheVazioOuSemAPastaENadaAFazer(t *testing.T) {
	maquina(t)
	p, err := Planeja("cache", nil, false)
	if err != nil || !p.NadaAFazer || *p.Liberar != 0 || len(p.Comandos) != 0 {
		t.Errorf("cache vazio: %+v, %v", p, err)
	}
	t.Setenv("MKPACOTES_CACHE", filepath.Join(t.TempDir(), "nada"))
	if _, err := Planeja("cache", nil, false); err == nil || !strings.Contains(err.Error(), "não consegui ler o cache") {
		t.Errorf("sem a pasta: %v", err)
	}
}

func TestPlano_AtualizarEOMkUpdateNumTerminal(t *testing.T) {
	m := maquina(t)
	if _, err := Planeja("atualizar", nil, false); err == nil || !strings.Contains(err.Error(), "não achei o mk-update") {
		t.Errorf("sem mk-update: %v", err)
	}
	m.Comando("mk-update", dubles.Regra{Contem: ""})
	if _, err := Planeja("atualizar", nil, false); err == nil || !strings.Contains(err.Error(), "não achei um terminal") {
		t.Errorf("sem terminal: %v", err)
	}
	m.Comando("kitty", dubles.Regra{Contem: ""})
	m.Comando("alacritty", dubles.Regra{Contem: ""})
	mk := filepath.Join(m.Dir(), "mk-update")
	p, err := Planeja("atualizar", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if quer := [][]string{{"kitty", "--title=Mk · atualizações", mk}}; !reflect.DeepEqual(p.Comandos, quer) {
		t.Errorf("comandos: %v", p.Comandos)
	}
	// O terminal escolhido nos Ajustes do Mk ganha; o do MKPACOTES_TERMINAL ganha de todos.
	if err := os.MkdirAll(filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "mk"), 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "mk", "ajustes.json"), []byte(`{"apps.terminal":"alacritty"}`), 0o644)
	p, _ = Planeja("atualizar", nil, false)
	if quer := [][]string{{"alacritty", "--title", "Mk · atualizações", "-e", mk}}; !reflect.DeepEqual(p.Comandos, quer) {
		t.Errorf("comandos com os Ajustes: %v", p.Comandos)
	}
	m.Comando("ghostty", dubles.Regra{Contem: ""})
	t.Setenv("MKPACOTES_TERMINAL", "ghostty")
	p, _ = Planeja("atualizar", nil, false)
	cfg := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "mk", "ghostty", "config")
	if quer := [][]string{{"ghostty", "--config-default-files=false", "--config-file=?" + cfg, "--title=Mk · atualizações", "-e", mk}}; !reflect.DeepEqual(p.Comandos, quer) {
		t.Errorf("comandos com MKPACOTES_TERMINAL: %v", p.Comandos)
	}
	// Um terminal que o Mk não conhece não vira comando.
	t.Setenv("MKPACOTES_TERMINAL", "rm")
	m.Comando("rm", dubles.Regra{Contem: ""})
	p, _ = Planeja("atualizar", nil, false)
	if p.Comandos[0][0] != "alacritty" {
		t.Errorf("MKPACOTES_TERMINAL desconhecido tem de ser ignorado: %v", p.Comandos)
	}
}

func TestPlano_AtualizarSemMkUpdateNoPathUsaODoMk(t *testing.T) {
	m := maquina(t)
	m.Comando("foot", dubles.Regra{Contem: ""})
	mk := filepath.Join(os.Getenv("MKPACOTES_MK"), "bin", "mk-update")
	os.MkdirAll(filepath.Dir(mk), 0o755)
	os.WriteFile(mk, []byte("#!/bin/sh\n"), 0o755)
	p, err := Planeja("atualizar", nil, false)
	if err != nil || !reflect.DeepEqual(p.Comandos, [][]string{{"foot", "--title=Mk · atualizações", mk}}) {
		t.Errorf("veio %+v, %v", p, err)
	}
}

// ── a ação ─────────────────────────────────────────────────────────────

func coleta() (*[]string, func(string)) {
	var linhas []string
	return &linhas, func(l string) { linhas = append(linhas, l) }
}

func TestAgir_InstalarOficialVaiPeloPkexecETransmiteAsLinhas(t *testing.T) {
	m := maquina(t)
	m.Comando("pkexec", dubles.Regra{Contem: "", Texto: "resolvendo dependências...\n\x1b[1minstalando fd...\x1b[0m\rinstalando fd... 100%\n\n", Erro: "aviso no stderr\n"})
	linhas, emite := coleta()
	p, err := Age("instalar", []string{"fd"}, false, emite)
	if err != nil {
		t.Fatal(err)
	}
	if quer := []string{"pacman -S --needed --noconfirm -- fd"}; !reflect.DeepEqual(m.Chamadas("pkexec"), quer) {
		t.Errorf("pkexec chamado com %v", m.Chamadas("pkexec"))
	}
	if !reflect.DeepEqual(p.Pedidos, []string{"fd"}) {
		t.Errorf("plano: %+v", p)
	}
	if (*linhas)[0] != "$ pkexec pacman -S --needed --noconfirm -- fd" {
		t.Errorf("a primeira linha é o comando: %q", *linhas)
	}
	resto := append([]string{}, (*linhas)[1:]...)
	// O stderr vem junto; as cores somem; o \r de uma barra vira linha.
	for _, quer := range []string{"resolvendo dependências...", "instalando fd...", "instalando fd... 100%", "aviso no stderr"} {
		achou := false
		for _, l := range resto {
			achou = achou || l == quer
		}
		if !achou {
			t.Errorf("faltou a linha %q em %q", quer, resto)
		}
	}
	if len(resto) != 4 {
		t.Errorf("linha vazia não é transmitida: %q", resto)
	}
}

func TestAgir_AURVaiPeloYayComoAPessoaEOficiaisAntes(t *testing.T) {
	m := maquina(t)
	m.Comando("pkexec", dubles.Regra{Contem: "", Texto: "ok pacman\n"})
	m.Comando("yay", dubles.Regra{Contem: "", Texto: "ok yay\n"})
	linhas, emite := coleta()
	if _, err := Age("instalar", []string{"aurzinho", "fd"}, false, emite); err != nil {
		t.Fatal(err)
	}
	if quer := []string{"pacman -S --needed --noconfirm -- fd"}; !reflect.DeepEqual(m.Chamadas("pkexec"), quer) {
		t.Errorf("pkexec: %v", m.Chamadas("pkexec"))
	}
	// O yay roda direto (nunca por baixo do pkexec) e é ele quem chama o pkexec.
	if quer := []string{"-S --needed --noconfirm --sudo pkexec -- aurzinho"}; !reflect.DeepEqual(m.Chamadas("yay"), quer) {
		t.Errorf("yay: %v", m.Chamadas("yay"))
	}
	quer := []string{"$ pkexec pacman -S --needed --noconfirm -- fd", "ok pacman", "$ yay -S --needed --noconfirm --sudo pkexec -- aurzinho", "ok yay"}
	if !reflect.DeepEqual(*linhas, quer) {
		t.Errorf("linhas: %q", *linhas)
	}
	for _, c := range append(m.Chamadas("pkexec"), m.Chamadas("yay")...) {
		if strings.Contains(c, "sudo ") && !strings.Contains(c, "--sudo pkexec") {
			t.Errorf("sudo não entra em comando nenhum: %s", c)
		}
	}
}

func TestAgir_RemoverEOrfaosUsamRns(t *testing.T) {
	m := maquina(t)
	m.Comando("pkexec", dubles.Regra{Contem: ""})
	if _, err := Age("remover", []string{"usa-htop", "htop"}, false, func(string) {}); err != nil {
		t.Fatal(err)
	}
	if _, err := Age("orfaos", nil, false, func(string) {}); err != nil {
		t.Fatal(err)
	}
	if _, err := Age("remover", []string{"mkinitcpio"}, true, func(string) {}); err != nil {
		t.Fatal(err)
	}
	quer := []string{"pacman -Rns --noconfirm -- usa-htop htop", "pacman -Rns --noconfirm -- libvelha", "pacman -Rns --noconfirm -- mkinitcpio"}
	if !reflect.DeepEqual(m.Chamadas("pkexec"), quer) {
		t.Errorf("pkexec: %v", m.Chamadas("pkexec"))
	}
}

func TestAgir_Cache(t *testing.T) {
	m := maquina(t)
	encheCache(t)
	m.Comando("pkexec", dubles.Regra{Contem: ""})
	if _, err := Age("cache", nil, false, func(string) {}); err != nil {
		t.Fatal(err)
	}
	if quer := []string{"pacman -Sc --noconfirm"}; !reflect.DeepEqual(m.Chamadas("pkexec"), quer) {
		t.Errorf("pkexec: %v", m.Chamadas("pkexec"))
	}
}

func TestAgir_NadaAFazerNaoPedeSenha(t *testing.T) {
	m := maquina(t)
	m.Comando("pkexec", dubles.Regra{Contem: ""})
	linhas, emite := coleta()
	if _, err := Age("instalar", []string{"bash"}, false, emite); err != nil {
		t.Fatal(err)
	}
	if _, err := Age("cache", nil, false, emite); err != nil {
		t.Fatal(err)
	}
	if len(m.Chamadas("pkexec")) != 0 {
		t.Errorf("sem nada a fazer o pkexec não é chamado: %v", m.Chamadas("pkexec"))
	}
	if len(*linhas) != 3 || !strings.Contains((*linhas)[0], "bash já está instalado") || (*linhas)[1] != "nada a fazer" {
		t.Errorf("linhas: %q", *linhas)
	}
}

func TestAgir_SenhaRecusadaViraFrase(t *testing.T) {
	m := maquina(t)
	for _, codigo := range []int{126, 127} {
		m.Comando("pkexec", dubles.Regra{Contem: "", Codigo: codigo, Erro: "Error executing command as another user: Request dismissed\n"})
		_, err := Age("remover", []string{"do-aur"}, false, func(string) {})
		if err == nil || recusado(err) || err.Error() != "a senha não foi dada: nada foi mudado" {
			t.Errorf("código %d: %v", codigo, err)
		}
	}
}

func TestAgir_ErroDoPacmanVemNaFrase(t *testing.T) {
	m := maquina(t)
	m.Comando("pkexec", dubles.Regra{Contem: "", Codigo: 1, Texto: "resolving dependencies...\n", Erro: "error: failed to commit transaction (conflicting files)\nfd: /usr/bin/fd exists in filesystem\nErrors occurred, no packages were upgraded.\n"})
	m.Comando("yay", dubles.Regra{Contem: ""})
	_, err := Age("instalar", []string{"fd", "aurzinho"}, false, func(string) {})
	if err == nil || err.Error() != "não deu para instalar: error: failed to commit transaction (conflicting files)" {
		t.Errorf("erro: %v", err)
	}
	if len(m.Chamadas("yay")) != 0 {
		t.Errorf("depois de um comando falhar, o seguinte não roda")
	}
	encheCache(t)
	m.Comando("pkexec", dubles.Regra{Contem: "", Codigo: 4})
	if _, err := Age("cache", nil, false, func(string) {}); err == nil || err.Error() != "não deu para limpar o cache: o pkexec saiu com o código 4" {
		t.Errorf("sem linha nenhuma, a frase traz o código: %v", err)
	}
}

func TestAgir_SemOPkexec(t *testing.T) {
	maquina(t)
	if _, err := Age("remover", []string{"do-aur"}, false, func(string) {}); err == nil || !strings.Contains(err.Error(), "não achei o pkexec") {
		t.Errorf("erro: %v", err)
	}
}

func TestAgir_AtualizarAbreOMkUpdateNoTerminalENaoRefazAAtualizacao(t *testing.T) {
	m := maquina(t)
	m.Comando("mk-update", dubles.Regra{Contem: ""})
	m.Comando("foot", dubles.Regra{Contem: ""})
	m.Comando("pkexec", dubles.Regra{Contem: ""})
	linhas, emite := coleta()
	antes := len(m.Chamadas("pacman"))
	if _, err := Age("atualizar", nil, false, emite); err != nil {
		t.Fatal(err)
	}
	if quer := []string{"--title=Mk · atualizações " + filepath.Join(m.Dir(), "mk-update")}; !reflect.DeepEqual(m.Chamadas("foot"), quer) {
		t.Errorf("foot: %v", m.Chamadas("foot"))
	}
	if len(m.Chamadas("pkexec")) != 0 || len(m.Chamadas("pacman")) != antes {
		t.Errorf("a atualização é do mk-update: o coletor não chama pacman nem pkexec")
	}
	if len(*linhas) < 2 || !strings.HasPrefix((*linhas)[0], "$ foot ") {
		t.Errorf("linhas: %q", *linhas)
	}
	m.Comando("foot", dubles.Regra{Contem: "", Codigo: 1})
	if _, err := Age("atualizar", nil, false, func(string) {}); err == nil {
		t.Errorf("terminal que falha tem de virar erro")
	}
}
