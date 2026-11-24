package servicos

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"mkpulso/internal/acoes"
	"mkpulso/internal/servicos/dubles"
)

// systemctlGravado põe o dublê do systemctl com as saídas de testdata.
func systemctlGravado(t *testing.T) *dubles.Mesa {
	t.Helper()
	m := dubles.Nova(t)
	m.Comando("systemctl",
		dubles.Regra{Contem: " show --timestamp=unix --property=ControlGroup,MainPID ", Texto: "ControlGroup=\nMainPID=0\n"},
		dubles.Regra{Contem: " --user list-units --type=scope,service ", Arquivo: "testdata/usuario-list-units-app.json"},
		dubles.Regra{Contem: " --user list-units ", Arquivo: "testdata/usuario-list-units.json"},
		dubles.Regra{Contem: " --user list-unit-files ", Arquivo: "testdata/usuario-list-unit-files.json"},
		dubles.Regra{Contem: " --user show --timestamp=unix ", Arquivo: "testdata/usuario-show.txt"},
		dubles.Regra{Contem: " list-units ", Arquivo: "testdata/sistema-list-units.json"},
		dubles.Regra{Contem: " list-unit-files ", Arquivo: "testdata/sistema-list-unit-files.json"},
		dubles.Regra{Contem: " show --timestamp=unix ", Arquivo: "testdata/sistema-show.txt"},
		dubles.Regra{Contem: ""},
	)
	return m
}

func num[T any](v T) *T { return &v }

func achar(t *testing.T, lista []Servico, unidade string) Servico {
	t.Helper()
	for _, s := range lista {
		if s.Unidade == unidade {
			return s
		}
	}
	t.Fatalf("%s não está na lista", unidade)
	return Servico{}
}

func TestLista_SistemaTraduzOsEstadosDoContrato(t *testing.T) {
	m := systemctlGravado(t)
	lista, err := Lista(false, true)
	if err != nil {
		t.Fatal(err)
	}
	quer := Servico{Unidade: "sshd.service", Escopo: "sistema", Descricao: "OpenSSH Daemon", Estado: "ativo",
		Sub: "running", Habilitado: "sim", Pid: num(812), MemKb: num(int64(5400)), DesdeS: num(int64(1795470000))}
	if veio := achar(t, lista, "sshd.service"); !reflect.DeepEqual(veio, quer) {
		t.Errorf("sshd.service:\n veio %+v\n quer %+v", veio, quer)
	}
	casos := []struct{ unidade, estado, sub, habilitado string }{
		{"NetworkManager.service", "ativo", "running", "sim"},
		{"bluetooth.service", "desativando", "stop-sigterm", "não"},
		{"cups.service", "falhou", "failed", "não"},
		{"cups.socket", "ativo", "listening", "sim"},
		{"fstrim.timer", "ativo", "waiting", "sim"},
		{"getty@tty1.service", "inativo", "dead", "sim"}, // pelo modelo getty@.service
		{"lvm2-monitor.service", "inativo", "dead", "mascarado"},
		{"man-db.service", "ativando", "start", "estático"},
		{"systemd-boot-update.service", "inativo", "dead", "não"}, // só o arquivo, nunca carregada
		{"user@1000.service", "ativo", "running", "estático"},
	}
	for _, c := range casos {
		s := achar(t, lista, c.unidade)
		if s.Estado != c.estado || s.Sub != c.sub || s.Habilitado != c.habilitado || s.Escopo != "sistema" {
			t.Errorf("%s: veio %s/%s/%s (%s), queria %s/%s/%s", c.unidade, s.Estado, s.Sub, s.Habilitado, s.Escopo, c.estado, c.sub, c.habilitado)
		}
	}
	// A descrição de quem só tem arquivo vem do show.
	if d := achar(t, lista, "systemd-boot-update.service").Descricao; d != "Automatic Boot Loader Update" {
		t.Errorf("descrição do que não está carregado: %q", d)
	}
	// Sem pid, sem memória e parado: null, nunca zero.
	cups := achar(t, lista, "cups.service")
	if cups.Pid != nil || cups.MemKb != nil || cups.DesdeS != nil {
		t.Errorf("cups.service parado devia ter pid, mem_kb e desde_s nulos: %+v", cups)
	}
	soquete := achar(t, lista, "cups.socket")
	if soquete.Pid != nil || soquete.MemKb == nil || *soquete.MemKb != 0 || *soquete.DesdeS != 1795470002 {
		t.Errorf("cups.socket: %+v", soquete)
	}
	// Fora: o que não existe (not-found), apelidos e modelos sem instância.
	for _, s := range lista {
		switch s.Unidade {
		case "auditd.service", "dbus.service", "getty@.service", "serial-getty@.service", "user@.service":
			t.Errorf("%s não devia estar na lista", s.Unidade)
		}
	}
	if len(lista) != 14 {
		t.Errorf("%d unidades, queria 14", len(lista))
	}
	for i := 1; i < len(lista); i++ {
		if lista[i-1].Unidade >= lista[i].Unidade {
			t.Errorf("fora de ordem: %s antes de %s", lista[i-1].Unidade, lista[i].Unidade)
		}
	}
	chamadas := m.Chamadas("systemctl")
	if len(chamadas) != 3 {
		t.Fatalf("queria 3 chamadas (uma só de show), vieram %d:\n%s", len(chamadas), strings.Join(chamadas, "\n"))
	}
	if chamadas[0] != "list-units --type=service,socket,timer --all --output=json --no-pager" {
		t.Errorf("list-units: %q", chamadas[0])
	}
	if chamadas[1] != "list-unit-files --type=service,socket,timer --output=json --no-pager" {
		t.Errorf("list-unit-files: %q", chamadas[1])
	}
	if !strings.HasPrefix(chamadas[2], "show --timestamp=unix --property=Id,Description,MainPID,MemoryCurrent,ActiveEnterTimestamp -- NetworkManager.service bluetooth.service ") {
		t.Errorf("show: %q", chamadas[2])
	}
}

func TestLista_UsuarioEOsDoisJuntos(t *testing.T) {
	m := systemctlGravado(t)
	lista, err := Lista(true, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(lista) != 5 {
		t.Fatalf("%d unidades de usuário, queria 5", len(lista))
	}
	claude := achar(t, lista, `app-claude\x2ddesktop@autostart.service`)
	if claude.Escopo != "usuario" || claude.Habilitado != "gerado" || *claude.Pid != 25127 || *claude.MemKb != 402432 {
		t.Errorf("claude: %+v", claude)
	}
	if s := achar(t, lista, "syncthing.service"); s.Estado != "inativo" || s.DesdeS != nil || s.Pid != nil {
		t.Errorf("syncthing parado: %+v", s)
	}
	for _, c := range m.Chamadas("systemctl") {
		if !strings.HasPrefix(c, "--user ") {
			t.Errorf("chamada sem --user: %q", c)
		}
	}
	todos, err := Lista(true, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(todos) != 19 || todos[0].Escopo != "sistema" || todos[18].Escopo != "usuario" {
		t.Errorf("os dois: %d unidades, primeiro %s, último %s", len(todos), todos[0].Escopo, todos[len(todos)-1].Escopo)
	}
}

func TestLista_CamposDoContratoNoJSON(t *testing.T) {
	systemctlGravado(t)
	lista, err := Lista(false, true)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(achar(t, lista, "sshd.service"))
	quer := `{"unidade":"sshd.service","escopo":"sistema","descricao":"OpenSSH Daemon","estado":"ativo","sub":"running","habilitado":"sim","pid":812,"mem_kb":5400,"desde_s":1795470000}`
	if string(b) != quer {
		t.Errorf("veio %s\nquer %s", b, quer)
	}
	b, _ = json.Marshal(achar(t, lista, "cups.service"))
	if !strings.Contains(string(b), `"pid":null,"mem_kb":null,"desde_s":null`) {
		t.Errorf("nulos: %s", b)
	}
}

func TestTraducao_TodosOsEstadosDoSystemd(t *testing.T) {
	estados := map[string]string{"active": "ativo", "reloading": "ativo", "refreshing": "ativo", "inactive": "inativo",
		"failed": "falhou", "activating": "ativando", "deactivating": "desativando", "maintenance": "inativo", "": "inativo"}
	for de, para := range estados {
		if v := traduzEstado(de); v != para {
			t.Errorf("estado %q: %q, queria %q", de, v, para)
		}
	}
	habilitados := map[string]string{"enabled": "sim", "enabled-runtime": "sim", "linked": "sim", "linked-runtime": "sim",
		"alias": "sim", "disabled": "não", "indirect": "não", "bad": "não", "static": "estático", "": "estático",
		"masked": "mascarado", "masked-runtime": "mascarado", "generated": "gerado", "transient": "gerado"}
	for de, para := range habilitados {
		if v := traduzHabilitado(de); v != para {
			t.Errorf("habilitado %q: %q, queria %q", de, v, para)
		}
	}
}

func TestLista_SemSystemctlEUmaFrase(t *testing.T) {
	dubles.Nova(t)
	_, err := Lista(true, true)
	if err == nil || !strings.Contains(err.Error(), "não achei o systemctl") {
		t.Fatalf("veio %v", err)
	}
}

func TestLista_SemOBarramentoDoUsuarioEUmaFrase(t *testing.T) {
	m := dubles.Nova(t)
	m.Comando("systemctl", dubles.Regra{Contem: " --user ", Codigo: 1,
		Erro: "Failed to connect to user scope bus via local transport: $DBUS_SESSION_BUS_ADDRESS and $XDG_RUNTIME_DIR not defined (consider using --machine=<user>@.host --user to connect to bus of other user)\n"})
	_, err := Lista(true, false)
	if err == nil || !strings.Contains(err.Error(), "systemd do usuário") || strings.Contains(err.Error(), "\n") {
		t.Fatalf("veio %v", err)
	}
}

func TestLista_SaidaQueNaoEJSONEUmaFrase(t *testing.T) {
	m := dubles.Nova(t)
	m.Comando("systemctl", dubles.Regra{Contem: "", Texto: "isto não é json\n"})
	if _, err := Lista(false, true); err == nil || !strings.Contains(err.Error(), "não entendi") {
		t.Fatalf("veio %v", err)
	}
}

func TestValidaUnidade(t *testing.T) {
	boas := []string{"sshd.service", "cups.socket", "fstrim.timer", "getty@tty1.service", `app-claude\x2ddesktop@autostart.service`,
		"org.cups.cupsd.service", "dbus-:1.2-org.x@0.service"}
	for _, u := range boas {
		if err := ValidaUnidade(u); err != nil {
			t.Errorf("%q devia passar: %v", u, err)
		}
	}
	mas := []string{"", "--now", "-x.service", "--user.service", "a.service;rm -rf", "a b.service", "a.service ", "../x.service",
		"/etc/x.service", "sshd", "sshd.mount", "sshd.service\n", "$(reboot).service", "a*.service", ".service", "sshd.service.bak",
		"é.service", strings.Repeat("a", 260) + ".service"}
	for _, u := range mas {
		if err := ValidaUnidade(u); err == nil {
			t.Errorf("%q passou", u)
		} else if acoes.EhRecusa(err) {
			t.Errorf("%q: nome inválido é erro de uso, não recusa", u)
		}
	}
}

func TestAge_CadaAcaoDoContratoViraOVerboCerto(t *testing.T) {
	casos := map[string]string{"iniciar": "start", "parar": "stop", "reiniciar": "restart", "habilitar": "enable",
		"desabilitar": "disable", "recarregar": "reload"}
	for acao, verbo := range casos {
		m := systemctlGravado(t)
		if err := Age(acao, "sshd.service", false, false); err != nil {
			t.Fatalf("%s: %v", acao, err)
		}
		chamadas := m.Chamadas("systemctl")
		if ultima := chamadas[len(chamadas)-1]; ultima != verbo+" -- sshd.service" {
			t.Errorf("%s: chamou %q", acao, ultima)
		}
		m = systemctlGravado(t)
		if err := Age(acao, "syncthing.service", true, false); err != nil {
			t.Fatalf("%s --usuario: %v", acao, err)
		}
		chamadas = m.Chamadas("systemctl")
		if ultima := chamadas[len(chamadas)-1]; ultima != "--user "+verbo+" -- syncthing.service" {
			t.Errorf("%s --usuario: chamou %q", acao, ultima)
		}
		for _, c := range chamadas {
			if strings.Contains(c, "--no-ask-password") {
				t.Errorf("%s: a senha é com o polkit, mas veio %q", acao, c)
			}
		}
	}
	if len(Acoes()) != len(casos) {
		t.Errorf("Acoes() tem %d, o contrato tem %d", len(Acoes()), len(casos))
	}
}

func TestAge_EntradaHostilNaoChegaAoSystemctl(t *testing.T) {
	m := systemctlGravado(t)
	for _, u := range []string{"--now", "a.service;rm -rf", "a b.service", "", "../../etc/x.service", "-f.service", "sshd"} {
		for _, acao := range Acoes() {
			if err := Age(acao, u, false, true); err == nil {
				t.Errorf("%s %q passou", acao, u)
			}
		}
	}
	for _, acao := range []string{"", "start", "kill", "--now", "apagar"} {
		err := Age(acao, "sshd.service", false, true)
		if err == nil || acoes.EhRecusa(err) {
			t.Errorf("ação %q: queria erro de uso, veio %v", acao, err)
		}
	}
	if c := m.Chamadas("systemctl"); len(c) != 0 {
		t.Errorf("chegou ao systemctl: %v", c)
	}
}

func TestAge_RecusaOQueDerrubaASessaoOuAMaquina(t *testing.T) {
	recusados := []struct{ acao, unidade string }{
		{"parar", "sddm.service"}, {"desabilitar", "sddm.service"}, {"reiniciar", "sddm.service"},
		{"parar", "dbus.service"}, {"parar", "dbus-broker.service"}, {"desabilitar", "dbus-broker.service"}, {"parar", "dbus.socket"},
		{"parar", "systemd-logind.service"}, {"desabilitar", "systemd-journald.service"}, {"parar", "systemd-journald.socket"},
		{"reiniciar", "systemd-logind.service"},
		{"parar", "user@1000.service"}, {"reiniciar", "user@1000.service"}, {"desabilitar", "user@1000.service"},
		{"parar", "NetworkManager.service"}, {"desabilitar", "NetworkManager.service"},
	}
	for _, usuario := range []bool{false, true} {
		for _, c := range recusados {
			m := systemctlGravado(t)
			err := Age(c.acao, c.unidade, usuario, false)
			if !acoes.EhRecusa(err) || !strings.Contains(err.Error(), "--mesmo-assim") {
				t.Errorf("%s %s: queria recusa, veio %v", c.acao, c.unidade, err)
			}
			if ch := m.Chamadas("systemctl"); len(ch) != 0 {
				t.Errorf("%s %s: chegou ao systemctl: %v", c.acao, c.unidade, ch)
			}
			if err := Age(c.acao, c.unidade, usuario, true); err != nil {
				t.Errorf("%s %s --mesmo-assim: %v", c.acao, c.unidade, err)
			}
			if ch := m.Chamadas("systemctl"); len(ch) != 1 {
				t.Errorf("%s %s --mesmo-assim: chamadas %v", c.acao, c.unidade, ch)
			}
		}
	}
	livres := []struct{ acao, unidade string }{
		{"iniciar", "sddm.service"}, {"habilitar", "NetworkManager.service"}, {"recarregar", "dbus-broker.service"},
		{"reiniciar", "NetworkManager.service"}, {"parar", "sshd.service"}, {"desabilitar", "cups.socket"},
		{"parar", "sddm-coisa.service"}, {"parar", "meu-dbus.service"},
	}
	for _, c := range livres {
		systemctlGravado(t)
		if err := Age(c.acao, c.unidade, false, false); err != nil {
			t.Errorf("%s %s devia passar: %v", c.acao, c.unidade, err)
		}
	}
}

// grupoFalso monta um cgroup em MKPULSO_SYS com os pids dados.
func grupoFalso(t *testing.T, raiz, caminho string, pids ...int) {
	t.Helper()
	dir := filepath.Join(raiz, "fs", "cgroup", caminho)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	var s strings.Builder
	for _, p := range pids {
		fmt.Fprintln(&s, p)
	}
	if err := os.WriteFile(filepath.Join(dir, "cgroup.procs"), []byte(s.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

// procFalso põe em MKPULSO_PROC processos de quem roda o teste, pelo nome.
func procFalso(t *testing.T, nomes map[int]string) {
	t.Helper()
	raiz := t.TempDir()
	for pid, nome := range nomes {
		dir := filepath.Join(raiz, strconv.Itoa(pid))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		status := fmt.Sprintf("Name:\t%s\nPid:\t%d\nPPid:\t1\nUid:\t%d\t%d\t%d\t%d\n", nome, pid, os.Getuid(), os.Getuid(), os.Getuid(), os.Getuid())
		if err := os.WriteFile(filepath.Join(dir, "status"), []byte(status), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("MKPULSO_PROC", raiz)
}

// unidadeComProcessos põe um dublê que diz o cgroup da unidade, e o cgroup.
func unidadeComProcessos(t *testing.T, cgroup string, principal int, pids ...int) *dubles.Mesa {
	t.Helper()
	sys := t.TempDir()
	t.Setenv("MKPULSO_SYS", sys)
	grupoFalso(t, sys, cgroup, pids...)
	m := dubles.Nova(t)
	m.Comando("systemctl",
		dubles.Regra{Contem: " show --timestamp=unix --property=ControlGroup,MainPID ", Texto: fmt.Sprintf("ControlGroup=%s\nMainPID=%d\n", cgroup, principal)},
		dubles.Regra{Contem: " list-units --type=scope,service ", Arquivo: "testdata/usuario-list-units-app.json"},
		dubles.Regra{Contem: ""},
	)
	return m
}

func TestAge_RecusaPararAUnidadeOndeMoraASessao(t *testing.T) {
	procFalso(t, map[int]string{700: "bash", 701: "quickshell", 702: "mkpulso-coletor"})
	m := unidadeComProcessos(t, "/user.slice/user-1000.slice/user@1000.service/app.slice/mare.service", 700, 700, 701, 702)
	for _, acao := range []string{"parar", "reiniciar"} {
		err := Age(acao, "mare.service", true, false)
		if !acoes.EhRecusa(err) || !strings.Contains(err.Error(), "quickshell") {
			t.Errorf("%s: queria recusa por causa do quickshell, veio %v", acao, err)
		}
	}
	for _, c := range m.Chamadas("systemctl") {
		if !strings.Contains(" "+c, " show ") {
			t.Errorf("chegou a agir: %q", c)
		}
	}
	if err := Age("parar", "mare.service", true, true); err != nil {
		t.Errorf("--mesmo-assim: %v", err)
	}
	// Habilitar e iniciar não derrubam nada: nem olham.
	if err := Age("habilitar", "mare.service", true, false); err != nil {
		t.Errorf("habilitar: %v", err)
	}
}

func TestAge_RecusaPararAUnidadeOndeMoraOProprioColetor(t *testing.T) {
	procFalso(t, map[int]string{os.Getpid(): "mkpulso-coletor"})
	unidadeComProcessos(t, "/user.slice/x/app.slice/coisa.service", 700, 700, os.Getpid())
	if err := Age("parar", "coisa.service", true, false); !acoes.EhRecusa(err) {
		t.Errorf("queria recusa, veio %v", err)
	}
}

func TestAge_ErroDoSystemctlVoltaNumaLinha(t *testing.T) {
	m := dubles.Nova(t)
	m.Comando("systemctl", dubles.Regra{Contem: "", Codigo: 1,
		Erro: "\nFailed to start sshd.service: Access denied\nSee system logs and 'systemctl status sshd.service' for details.\n"})
	err := Age("iniciar", "sshd.service", false, false)
	if err == nil || acoes.EhRecusa(err) {
		t.Fatalf("veio %v", err)
	}
	if !strings.Contains(err.Error(), "Failed to start sshd.service: Access denied") || strings.Contains(err.Error(), "\n") ||
		strings.Contains(err.Error(), "See system logs") {
		t.Errorf("frase: %q", err.Error())
	}
	m.Comando("systemctl", dubles.Regra{Contem: "", Codigo: 5})
	if err := Age("iniciar", "sshd.service", false, false); err == nil || !strings.Contains(err.Error(), "5") {
		t.Errorf("sem stderr: %v", err)
	}
	dubles.Nova(t)
	if err := Age("iniciar", "sshd.service", false, false); err == nil || !strings.Contains(err.Error(), "não achei o systemctl") {
		t.Errorf("sem systemctl: %v", err)
	}
}

func TestRegistro_TraduzNivelTempoETexto(t *testing.T) {
	m := dubles.Nova(t)
	m.Comando("journalctl", dubles.Regra{Contem: "", Arquivo: "testdata/journal.jsonl"})
	linhas, err := Registro("sshd.service", false, 80)
	if err != nil {
		t.Fatal(err)
	}
	quer := []Linha{
		{num(int64(1795470000)), "info", "Server listening on 0.0.0.0 port 22."},
		{num(int64(1795470060)), "erro", "error: kex_exchange_identification: Connection closed by remote host"},
		{num(int64(1795470061)), "aviso", "spa.alsa: front:0p: (0 suppressed) snd_pcm_avail after recover: Pipe quebrado"},
		{num(int64(1795470062)), "depuração", "debug1: Bind to port 22 on ::."},
		{num(int64(1795470063)), "info", "Ol\ufffd mundo"},         // bytes em latin-1
		{num(int64(1795470064)), "info", "\x1b[31matenção\x1b[0m"}, // bytes em UTF-8, com escapes de cor
		{num(int64(1795470065)), "erro", ""},                       // MESSAGE nulo
		{num(int64(1795470066)), "info", "sem prioridade"},         // sem PRIORITY
		{num(int64(1795470067)), "erro", "primeira"},               // campo repetido
		{nil, "info", "duas\nlinhas"},                              // sem relógio
	}
	if !reflect.DeepEqual(linhas, quer) {
		for i := range linhas {
			if i < len(quer) && !reflect.DeepEqual(linhas[i], quer[i]) {
				t.Errorf("linha %d: veio %+v, queria %+v", i, linhas[i], quer[i])
			}
		}
		t.Errorf("vieram %d linhas, queria %d", len(linhas), len(quer))
	}
	if c := m.Chamadas("journalctl"); len(c) != 1 || c[0] != "-u sshd.service -n 80 -o json --no-pager" {
		t.Errorf("chamada: %v", c)
	}
	b, _ := json.Marshal(linhas[0])
	if string(b) != `{"t_s":1795470000,"nivel":"info","texto":"Server listening on 0.0.0.0 port 22."}` {
		t.Errorf("JSON: %s", b)
	}
}

func TestRegistro_UsuarioELinhas(t *testing.T) {
	m := dubles.Nova(t)
	m.Comando("journalctl", dubles.Regra{Contem: ""})
	linhas, err := Registro("pipewire.service", true, 5)
	if err != nil || linhas == nil || len(linhas) != 0 {
		t.Fatalf("journal vazio: queria lista vazia, veio %v %v", linhas, err)
	}
	if c := m.Chamadas("journalctl"); len(c) != 1 || c[0] != "--user -u pipewire.service -n 5 -o json --no-pager" {
		t.Errorf("chamada: %v", c)
	}
}

func TestRegistro_EntradaHostilNaoChegaAoJournalctl(t *testing.T) {
	m := dubles.Nova(t)
	m.Comando("journalctl", dubles.Regra{Contem: ""})
	for _, n := range []int{0, -1, 501, 100000} {
		if _, err := Registro("sshd.service", false, n); err == nil || !strings.Contains(err.Error(), "1 a 500") {
			t.Errorf("%d linhas: veio %v", n, err)
		}
	}
	for _, u := range []string{"--now", "a.service;rm -rf", "--file=/etc/shadow", "-u.service", ""} {
		if _, err := Registro(u, false, 10); err == nil {
			t.Errorf("unidade %q passou", u)
		}
	}
	if c := m.Chamadas("journalctl"); len(c) != 0 {
		t.Errorf("chegou ao journalctl: %v", c)
	}
	for _, n := range []int{1, 500} {
		if _, err := Registro("sshd.service", false, n); err != nil {
			t.Errorf("%d linhas: %v", n, err)
		}
	}
}

func TestRegistro_ErroESemJournalctl(t *testing.T) {
	m := dubles.Nova(t)
	m.Comando("journalctl", dubles.Regra{Contem: "", Codigo: 1, Erro: "Failed to open journal: Permission denied\n"})
	if _, err := Registro("sshd.service", false, 10); err == nil || !strings.Contains(err.Error(), "Permission denied") {
		t.Errorf("veio %v", err)
	}
	dubles.Nova(t)
	if _, err := Registro("sshd.service", false, 10); err == nil || !strings.Contains(err.Error(), "não achei o journalctl") {
		t.Errorf("veio %v", err)
	}
}

func TestUnidadesDoApp_AchaEscoposEServicosDoApp(t *testing.T) {
	m := systemctlGravado(t)
	casos := map[string][]string{
		"firefox":                   {"app-firefox-1234.scope", "app-firefox-5678.scope"},
		"claude-desktop":            {`app-claude\x2ddesktop@autostart.service`},
		"com.anthropic.Claude":      {"app-com.anthropic.Claude-1031441.scope"},
		"org.kde.kdeconnect.daemon": {"app-org.kde.kdeconnect.daemon@autostart.service"},
		"fire":                      nil,
		"kitty":                     nil,
		"nome com espaço":           nil,
		"*":                         nil,
	}
	for nome, quer := range casos {
		veio, err := UnidadesDoApp(nome)
		if err != nil || !reflect.DeepEqual(veio, quer) {
			t.Errorf("%q: veio %v (%v), queria %v", nome, veio, err, quer)
		}
	}
	if c := m.Chamadas("systemctl"); c[0] != "--user list-units --type=scope,service --output=json --no-pager app-*" {
		t.Errorf("chamada: %q", c[0])
	}
}

func TestPidsDaUnidade_LeOCgroupInteiro(t *testing.T) {
	unidadeComProcessos(t, "/system.slice/sshd.service", 812, 812, 900)
	grupoFalso(t, os.Getenv("MKPULSO_SYS"), "/system.slice/sshd.service/filho", 950, 812)
	if pids := PidsDaUnidade("sshd.service", false); !reflect.DeepEqual(pids, []int{812, 900, 950}) {
		t.Errorf("veio %v", pids)
	}
	// Sem cgroup legível, sobra o processo principal; sem nada, lista vazia.
	t.Setenv("MKPULSO_SYS", t.TempDir())
	if pids := PidsDaUnidade("sshd.service", false); !reflect.DeepEqual(pids, []int{812}) {
		t.Errorf("só o principal: veio %v", pids)
	}
	systemctlGravado(t)
	if pids := PidsDaUnidade("sshd.service", false); pids == nil || len(pids) != 0 {
		t.Errorf("sem nada: veio %v", pids)
	}
	// Um cgroup que tenta sair da árvore não é lido.
	sys := t.TempDir()
	t.Setenv("MKPULSO_SYS", filepath.Join(sys, "sys"))
	grupoFalso(t, sys, "../../fora", 4242)
	m := dubles.Nova(t)
	m.Comando("systemctl", dubles.Regra{Contem: "", Texto: "ControlGroup=/../../../fora\nMainPID=0\n"})
	if pids := PidsDaUnidade("sshd.service", false); len(pids) != 0 {
		t.Errorf("saiu da árvore: %v", pids)
	}
}

func TestSinalNaUnidade_EncerrarEStopORestoEKill(t *testing.T) {
	procFalso(t, map[int]string{1234: "firefox", 1240: "Web Content"})
	casos := map[string]string{
		"encerrar":    "--user stop -- app-firefox-1234.scope",
		"matar":       "--user kill --signal=SIGKILL -- app-firefox-1234.scope",
		"pausar":      "--user kill --signal=SIGSTOP -- app-firefox-1234.scope",
		"continuar":   "--user kill --signal=SIGCONT -- app-firefox-1234.scope",
		"recarregar":  "--user kill --signal=SIGHUP -- app-firefox-1234.scope",
		"interromper": "--user kill --signal=SIGINT -- app-firefox-1234.scope",
	}
	for sinal, quer := range casos {
		m := unidadeComProcessos(t, "/user.slice/x/app.slice/app-firefox-1234.scope", 0, 1240, 1234)
		pids, err := SinalNaUnidade("app-firefox-1234.scope", true, sinal, false)
		if err != nil {
			t.Fatalf("%s: %v", sinal, err)
		}
		if !reflect.DeepEqual(pids, []int{1234, 1240}) {
			t.Errorf("%s: pids %v", sinal, pids)
		}
		c := m.Chamadas("systemctl")
		if c[len(c)-1] != quer {
			t.Errorf("%s: chamou %q, queria %q", sinal, c[len(c)-1], quer)
		}
	}
	m := unidadeComProcessos(t, "/system.slice/sshd.service", 812, 812)
	if _, err := SinalNaUnidade("sshd.service", false, "matar", false); err != nil {
		t.Fatal(err)
	}
	if c := m.Chamadas("systemctl"); c[len(c)-1] != "kill --signal=SIGKILL -- sshd.service" {
		t.Errorf("sistema: %q", c[len(c)-1])
	}
}

func TestSinalNaUnidade_Recusas(t *testing.T) {
	procFalso(t, map[int]string{600: "Hyprland", 1234: "firefox"})
	m := unidadeComProcessos(t, "/user.slice/user-1000.slice/session-1.scope", 0, 600, 1234)
	semChegar := func(oque string) {
		t.Helper()
		for _, c := range m.Chamadas("systemctl") {
			if !strings.Contains(" "+c, " show ") {
				t.Errorf("%s: chegou a agir: %q", oque, c)
			}
		}
	}
	// A unidade onde mora o Hyprland.
	if _, err := SinalNaUnidade("session-1.scope", false, "encerrar", false); !acoes.EhRecusa(err) {
		t.Errorf("session-1.scope: queria recusa, veio %v", err)
	}
	// O que derruba a máquina, pelo nome.
	for _, u := range []string{"sddm.service", "dbus-broker.service", "systemd-logind.service", "user@1000.service", "NetworkManager.service"} {
		for _, s := range []string{"encerrar", "matar", "pausar", "interromper", "recarregar"} {
			if _, err := SinalNaUnidade(u, false, s, false); !acoes.EhRecusa(err) {
				t.Errorf("%s %s: queria recusa, veio %v", s, u, err)
			}
		}
	}
	// Nome hostil, tipo errado, sinal desconhecido: erro de uso.
	for _, u := range []string{"--now", "a.scope;rm -rf", "tmp.mount", "user.slice", ""} {
		if _, err := SinalNaUnidade(u, false, "encerrar", true); err == nil || acoes.EhRecusa(err) {
			t.Errorf("%q: queria erro de uso, veio %v", u, err)
		}
	}
	if _, err := SinalNaUnidade("sshd.service", false, "explodir", true); err == nil {
		t.Error("sinal desconhecido passou")
	}
	semChegar("recusas")
	if _, err := SinalNaUnidade("session-1.scope", false, "encerrar", true); err != nil {
		t.Errorf("--mesmo-assim: %v", err)
	}
}
