package grupos

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mkpulso/internal/coleta"
	"mkpulso/internal/coleta/falso"
	"mkpulso/internal/janelas"
)

// naArvore aponta as raízes para testdata/desktop.
func naArvore(t *testing.T) {
	t.Helper()
	falso.Raizes(t)
	raiz, err := filepath.Abs("testdata/desktop")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("MKPULSO_PROC", filepath.Join(raiz, "proc"))
	t.Setenv("MKPULSO_ETC", filepath.Join(raiz, "etc"))
	t.Setenv("MKPULSO_HOME", filepath.Join(raiz, "home"))
	t.Setenv("MKPULSO_APPS", filepath.Join(raiz, "apps"))
	t.Setenv("XDG_DATA_DIRS", "/nao/existe:"+filepath.Join(raiz, "xdg"))
}

var janelasDaArvore = []janelas.Janela{
	{Titulo: "Mk — Firefox", Workspace: "2", Endereco: "0x55d0", PID: 1234, Classe: "firefox"},
	{Titulo: "Janela privativa", Workspace: "3", Endereco: "0x55d1", PID: 1250, Classe: "firefox"},
	{Titulo: "Meu app", Workspace: "1", Endereco: "0x55d2", PID: 1500, Classe: "MeuApp"},
	{Titulo: "notas.org", Workspace: "1", Endereco: "0x55d3", PID: 1800, Classe: "emacs"},
	{Titulo: "de ninguém", Workspace: "1", Endereco: "0x55d4", PID: 99999, Classe: "sumiu"},
}

func agrupaArvore(t *testing.T) []Grupo {
	t.Helper()
	naArvore(t)
	return Agrupar(coleta.MedirProcessos(nil, coleta.Colher(0)), janelasDaArvore, 4, NovosAtalhos())
}

// grupoDoPid acha o grupo em que um processo caiu.
func grupoDoPid(t *testing.T, gs []Grupo, pid int) Grupo {
	t.Helper()
	var achado *Grupo
	for i := range gs {
		for _, p := range gs[i].Processos {
			if p.PID == pid {
				if achado != nil {
					t.Fatalf("o processo %d está em dois grupos: %s e %s", pid, achado.ID, gs[i].ID)
				}
				achado = &gs[i]
			}
		}
	}
	if achado == nil {
		t.Fatalf("o processo %d não está em grupo nenhum", pid)
	}
	return *achado
}

func TestCadaRamoDaRegraNaArvore(t *testing.T) {
	gs := agrupaArvore(t)
	casos := []struct {
		pid                              int
		id, tipo, escopo, unidade, oCaso string
	}{
		{2, "sistema:kernel", "sistema", "", "", "o kthreadd"},
		{30, "sistema:kernel", "sistema", "", "", "filho do kthreadd"},
		{1, "servico:init.scope", "servico", "sistema", "init.scope", "o pid 1"},
		{300, "servico:sshd.service", "servico", "sistema", "sshd.service", "serviço do sistema"},
		{311, "servico:dbus-broker.service", "servico", "sistema", "dbus-broker.service", "serviço do sistema com filho"},
		{321, "servico:docker.service", "servico", "sistema", "docker.service", "subgrupo dentro de um serviço"},
		{800, "servico:user@1000.service", "servico", "sistema", "user@1000.service", "o systemd do usuário"},
		{810, "servico:dbus-broker.service", "servico", "usuario", "dbus-broker.service", "serviço de usuário com o mesmo nome de um do sistema"},
		{820, "servico:pipewire.service", "servico", "usuario", "pipewire.service", "serviço de usuário"},
		{830, "servico:wayland-wm@hyprland.desktop.service", "servico", "usuario", "wayland-wm@hyprland.desktop.service", "o compositor"},
		{1234, "app:firefox", "app", "usuario", "app-firefox-1234.scope", "app-<nome>-<algo>.scope"},
		{1240, "app:firefox", "app", "usuario", "app-firefox-1234.scope", "filho no mesmo scope"},
		{1250, "app:firefox", "app", "usuario", "app-firefox-1234.scope", "outro scope do mesmo app"},
		{1300, "app:claude-desktop", "servico", "usuario", `app-claude\x2ddesktop@autostart.service`, "app-<nome>@<algo>.service com \\x2d, sem janela nem atalho"},
		{1400, "app:org.gnome.Nautilus", "app", "usuario", "app-gnome-org.gnome.Nautilus-1400.scope", "prefixo do lançador some quando o resto tem atalho"},
		{1500, "app:meu-app", "app", "usuario", "app-meu-app-1500.scope", "sem atalho mas com janela"},
		{1700, "servico:ryoku-session-guard@9.service", "servico", "usuario", "ryoku-session-guard@9.service", "serviço dentro de uma fatia app-*.slice"},
		{1800, "servico:emacs.service", "app", "usuario", "emacs.service", "serviço de usuário com janela"},
		{2000, "app:login", "servico", "", "", "raiz da sessão"},
		{2001, "app:login", "servico", "", "", "o shell do login"},
		{2100, "app:kitty", "app", "", "", "solto na sessão, com atalho"},
		{2102, "app:kitty", "app", "", "", "neto do terminal"},
		{2201, "app:kitty", "app", "", "", "filho do segundo terminal"},
		{3100, "app:solto", "servico", "", "", "numa fatia, sem unidade"},
		{3200, "sistema:outros", "sistema", "", "", "cgroup raiz"},
		{3201, "sistema:outros", "sistema", "", "", "sem cgroup"},
		{3301, "app:nginx", "servico", "", "", "cgroup que não é do systemd"},
	}
	for _, c := range casos {
		g := grupoDoPid(t, gs, c.pid)
		if g.ID != c.id || g.Tipo != c.tipo || g.Escopo != c.escopo || g.Unidade != c.unidade {
			t.Errorf("%s (pid %d): veio id=%q tipo=%q escopo=%q unidade=%q; esperava id=%q tipo=%q escopo=%q unidade=%q",
				c.oCaso, c.pid, g.ID, g.Tipo, g.Escopo, g.Unidade, c.id, c.tipo, c.escopo, c.unidade)
		}
	}
	total := 0
	for _, g := range gs {
		total += len(g.Processos)
		if g.N != len(g.Processos) {
			t.Errorf("%s: n=%d com %d processos", g.ID, g.N, len(g.Processos))
		}
	}
	if total != len(arvore) {
		t.Errorf("cada processo num grupo só: %d processos, %d agrupados", len(arvore), total)
	}
}

func TestNomeIconeEJanelasNaArvore(t *testing.T) {
	gs := agrupaArvore(t)
	f := grupoDoPid(t, gs, 1234)
	if f.Nome != "Navegador Firefox" || f.Icone != "firefox" {
		t.Errorf("firefox: o atalho da home ganha e Name[pt_BR] ganha de Name[pt] e de Name: %q / %q", f.Nome, f.Icone)
	}
	if f.N != 3 || f.PID != 1234 || f.MemKB != 700000 {
		t.Errorf("firefox: n=%d pid=%d mem=%d", f.N, f.PID, f.MemKB)
	}
	if len(f.Janelas) != 2 || f.Janelas[0].Titulo != "Mk — Firefox" || f.Janelas[1].Workspace != "3" {
		t.Errorf("firefox: janelas = %+v", f.Janelas)
	}
	if n := grupoDoPid(t, gs, 1400); n.Nome != "Ficheiros" || n.Icone != "org.gnome.Nautilus" {
		t.Errorf("nautilus: atalho achado pelo XDG_DATA_DIRS, Name[pt] ganha de Name: %q / %q", n.Nome, n.Icone)
	}
	if k := grupoDoPid(t, gs, 2100); k.Nome != "kitty" || k.Icone != "kitty" || len(k.Janelas) != 0 || k.Janelas == nil || k.N != 5 || k.PID != 2100 {
		t.Errorf("kitty = %+v", k)
	}
	if m := grupoDoPid(t, gs, 1500); m.Nome != "meu-app" || m.Icone != "" || len(m.Janelas) != 1 {
		t.Errorf("meu-app: sem atalho o nome é o do app e o ícone é vazio: %+v", m)
	}
	if e := grupoDoPid(t, gs, 1800); e.Nome != "Emacs" || e.Icone != "emacs" {
		t.Errorf("emacs: serviço com janela acha o atalho pela classe da janela: %q / %q", e.Nome, e.Icone)
	}
	if c := grupoDoPid(t, gs, 1300); c.Nome != "claude-desktop" {
		t.Errorf("o \\x2d do systemd é um hífen: %q", c.Nome)
	}
	if s := grupoDoPid(t, gs, 300); s.Nome != "sshd.service" || s.Icone != "" {
		t.Errorf("serviço: o nome é a unidade: %+v", s)
	}
	if k := grupoDoPid(t, gs, 2); k.Nome != "Kernel" || k.N != 2 {
		t.Errorf("kernel = %q com %d", k.Nome, k.N)
	}
	if o := grupoDoPid(t, gs, 3200); o.Nome != "Outros processos" || o.N != 2 {
		t.Errorf("outros = %+v", o)
	}
	for _, g := range gs {
		for _, j := range g.Janelas {
			if j.Endereco == "0x55d4" {
				t.Errorf("janela de um pid que não existe foi parar em %s", g.ID)
			}
		}
	}
}

func TestIdsNaoSeRepetemSalvoPeloEscopo(t *testing.T) {
	gs := agrupaArvore(t)
	visto := map[string]bool{}
	for _, g := range gs {
		chave := g.ID + "|" + g.Escopo
		if visto[chave] {
			t.Errorf("grupo repetido: %s", chave)
		}
		visto[chave] = true
	}
	if !visto["servico:dbus-broker.service|sistema"] || !visto["servico:dbus-broker.service|usuario"] {
		t.Errorf("o dbus-broker do sistema e o do usuário são grupos separados: %v", visto)
	}
}

func TestIdentificar(t *testing.T) {
	temAtalho := func(nome string) bool { return nome == "firefox-esr" || nome == "org.kde.dolphin" || nome == "code" }
	u := "/user.slice/user-1000.slice/user@1000.service"
	casos := []struct{ cgroup, id, escopo string }{
		{u + "/app.slice/app-firefox-1234.scope", "app:firefox", "usuario"},
		{u + "/app.slice/app-firefox.scope", "app:firefox", "usuario"},
		{u + `/app.slice/app-firefox\x2desr-999.scope`, "app:firefox-esr", "usuario"},
		{u + "/app.slice/app-firefox-esr-999.scope", "app:firefox-esr", "usuario"},
		{u + "/app.slice/app-flatpak-org.kde.dolphin-4242.scope", "app:org.kde.dolphin", "usuario"},
		{u + "/app.slice/app-niri-code-77.scope/sub", "app:code", "usuario"},
		{u + "/app.slice/app-foot@abc123.service", "app:foot", "usuario"},
		{u + "/app.slice/app-foot.service", "app:foot", "usuario"},
		{u + `/app.slice/app-um\x20nome\xc3\xa7@1.service`, "app:um nomeç", "usuario"},
		{u + `/app.slice/app-torto\xZZ\x-9.scope`, `app:torto\xZZ\x`, "usuario"},
		{u + "/app.slice/app--5.scope", "servico:app--5.scope", "usuario"},
		{u + "/app.slice/app-.scope", "servico:app-.scope", "usuario"},
		{u + "/session.slice/pipewire.service", "servico:pipewire.service", "usuario"},
		{u + "/app.slice/dconf.service", "servico:dconf.service", "usuario"},
		{u + "/init.scope", "servico:user@1000.service", "sistema"},
		{u, "servico:user@1000.service", "sistema"},
		{u + "/app.slice", "", ""},
		{"/system.slice/sshd.service", "servico:sshd.service", "sistema"},
		{"/system.slice/system-getty.slice/getty@tty1.service", "servico:getty@tty1.service", "sistema"},
		{"/system.slice/boot.mount", "servico:boot.mount", "sistema"},
		{"/machine.slice/libpod-abc.scope/container", "servico:libpod-abc.scope", "sistema"},
		{"/init.scope", "servico:init.scope", "sistema"},
		{"/user.slice/user-1000.slice/session-3.scope", "", ""},
		{"/user.slice/user-1000.slice/session-c1.scope", "", ""},
		{"/docker/abc", "", ""},
		{"/user.slice", "", ""},
		{"/", "sistema:outros", ""},
		{"", "sistema:outros", ""},
		{"///", "sistema:outros", ""},
	}
	for _, c := range casos {
		i := Identificar(c.cgroup, false, temAtalho)
		if i.ID != c.id || i.Escopo != c.escopo {
			t.Errorf("%q: veio id=%q escopo=%q; esperava id=%q escopo=%q", c.cgroup, i.ID, i.Escopo, c.id, c.escopo)
		}
		if (c.id == "") != i.PeloDono {
			t.Errorf("%q: PeloDono=%v", c.cgroup, i.PeloDono)
		}
	}
	if i := Identificar("/system.slice/sshd.service", true, nil); i.ID != "sistema:kernel" {
		t.Errorf("thread do kernel: %+v", i)
	}
	if i := Identificar(u+"/app.slice/app-gnome-foo-1.scope", false, nil); i.ID != "app:gnome-foo" {
		t.Errorf("sem atalho nenhum vale o nome inteiro: %+v", i)
	}
}

func TestLacoDePaisNaoTrava(t *testing.T) {
	ms := []coleta.Medido{
		{Processo: coleta.Processo{PID: 10, PPID: 11, Nome: "a", Cgroup: "/x"}},
		{Processo: coleta.Processo{PID: 11, PPID: 10, Nome: "b", Cgroup: "/x"}},
		{Processo: coleta.Processo{PID: 12, PPID: 12, Nome: "c", Cgroup: "/x"}},
	}
	feito := make(chan []Grupo, 1)
	go func() { feito <- Agrupar(ms, nil, 1, nil) }()
	select {
	case gs := <-feito:
		n := 0
		for _, g := range gs {
			n += g.N
		}
		if n != 3 {
			t.Errorf("os três processos deveriam estar agrupados: %+v", gs)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("um laço de pais travou o agrupamento")
	}
}

func med(pid int, nome, cgroup string, cpu float64, mem int64, disco, gpu *float64) coleta.Medido {
	return coleta.Medido{Processo: coleta.Processo{PID: pid, PPID: 1, Nome: nome, Cmd: nome, Cgroup: cgroup, MemKB: mem, InicioTicks: uint64(pid)}, CPUPct: cpu, DiscoBps: disco, GPUPct: gpu}
}

func TestSomasOrdemENulos(t *testing.T) {
	f := coleta.Ptr[float64]
	ms := []coleta.Medido{
		med(10, "a", "/system.slice/a.service", 1.5, 100, f(1000), nil),
		med(11, "a2", "/system.slice/a.service", 30.25, 200, nil, f(60)),
		med(12, "a3", "/system.slice/a.service", 30.25, 300, f(500), f(70)),
		med(20, "b", "/system.slice/b.service", 90, 50, nil, nil),
		med(30, "c", "/system.slice/c.service", 0, 10, f(0), nil),
		med(31, "d", "/system.slice/d.service", 0, 10, f(0), nil),
		{Processo: coleta.Processo{PID: 5, PPID: 2, Nome: "kworker", Kernel: true}, CPUPct: 0},
	}
	gs := Agrupar(ms, nil, 4, nil)
	var ids []string
	for _, g := range gs {
		ids = append(ids, g.ID)
	}
	quer := "servico:b.service servico:a.service servico:c.service servico:d.service sistema:kernel"
	if strings.Join(ids, " ") != quer {
		t.Fatalf("ordem por cpu decrescente e, no empate, por id:\n veio    %v\n esperava %s", ids, quer)
	}
	a := gs[1]
	if a.CPUPct != 62 || a.MemKB != 600 || a.DiscoBps == nil || *a.DiscoBps != 1500 || a.GPUPct == nil || *a.GPUPct != 100 {
		t.Errorf("somas do grupo (a gpu para em 100): cpu=%v mem=%v disco=%v gpu=%v", a.CPUPct, a.MemKB, a.DiscoBps, a.GPUPct)
	}
	if a.Processos[0].PID != 11 || a.Processos[1].PID != 12 || a.Processos[2].PID != 10 {
		t.Errorf("processos por cpu decrescente e, no empate, por pid: %+v", a.Processos)
	}
	if a.PID != 10 {
		t.Errorf("o principal é o mais antigo: %d", a.PID)
	}
	b := gs[0]
	if b.DiscoBps != nil || b.GPUPct != nil {
		t.Errorf("grupo em que ninguém tem disco nem gpu legível: null, veio %v %v", b.DiscoBps, b.GPUPct)
	}
	if a.Media.CPUPct1m != 62 || a.Media.CPUPct3m != 62 || a.Media.MemKB1m != 600 || a.Media.DiscoBps3m != 1500 || b.Media.DiscoBps1m != 0 {
		t.Errorf("sem histórico a média é o valor de agora: %+v / %+v", a.Media, b.Media)
	}
	if gs := Agrupar(nil, nil, 0, nil); gs == nil || len(gs) != 0 {
		t.Errorf("sem processos: lista vazia, não null: %#v", gs)
	}
}

func TestJsonDoGrupoTemOsNomesDoContrato(t *testing.T) {
	gs := Agrupar([]coleta.Medido{med(10, "a", "/system.slice/a.service", 1, 1, nil, nil)}, nil, 1, nil)
	b, err := json.Marshal(gs[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, campo := range []string{`"id":"servico:a.service"`, `"tipo":"servico"`, `"nome":"a.service"`, `"icone":""`, `"unidade":"a.service"`, `"escopo":"sistema"`,
		`"janelas":[]`, `"n":1`, `"pid":10`, `"cpu_pct":1`, `"mem_kb":1`, `"disco_bps":null`, `"gpu_pct":null`, `"energia":"`, `"media":{"cpu_pct_1m":1,"cpu_pct_3m":1,"mem_kb_1m":1,"mem_kb_3m":1,"disco_bps_1m":0,"disco_bps_3m":0}`,
		`"processos":[{"pid":10,"ppid":1,"nome":"a","cmd":"a","usuario":"","estado":"","cpu_pct":1,"mem_kb":1,"disco_bps":null,"gpu_pct":null,"threads":0,"nice":0,"inicio_s":null}]`} {
		if !strings.Contains(string(b), campo) {
			t.Errorf("falta %s em %s", campo, b)
		}
	}
}

// --- energia ---

func TestFronteirasDaEnergia(t *testing.T) {
	f := coleta.Ptr[float64]
	casos := []struct {
		cpu     float64
		nucleos int
		disco   *float64
		gpu     *float64
		quer    string
	}{
		{0, 4, nil, nil, "nenhuma"},
		{0.3, 4, nil, nil, "nenhuma"},         // 0,075
		{0.4, 4, nil, nil, "muito baixa"},     // 0,1
		{7.9, 4, nil, nil, "muito baixa"},     // 1,975
		{8, 4, nil, nil, "baixa"},             // 2
		{31.9, 4, nil, nil, "baixa"},          // 7,975
		{32, 4, nil, nil, "moderada"},         // 8
		{99.9, 4, nil, nil, "moderada"},       // 24,975
		{100, 4, nil, nil, "alta"},            // 25: um núcleo inteiro de quatro
		{239.9, 4, nil, nil, "alta"},          // 59,975
		{240, 4, nil, nil, "muito alta"},      // 60
		{400, 4, nil, nil, "muito alta"},      //
		{100, 1, nil, nil, "muito alta"},      // o mesmo processo numa máquina de um núcleo
		{100, 16, nil, nil, "baixa"},          // 6,25: e numa de dezesseis
		{0, 4, nil, f(0), "nenhuma"},          //
		{0, 4, nil, f(4), "baixa"},            // gpu vale metade: 2
		{0, 4, nil, f(3.9), "muito baixa"},    //
		{0, 4, nil, f(50), "alta"},            // 25
		{0, 4, nil, f(100), "alta"},           // 50
		{40, 4, nil, f(100), "muito alta"},    // 10 + 50
		{0, 4, f(0), nil, "nenhuma"},          //
		{0, 4, f(104857), nil, "nenhuma"},     // um pouco menos de 0,1 MiB/s
		{0, 4, f(104858), nil, "muito baixa"}, //
		{0, 4, f(2 << 20), nil, "baixa"},      // 2 MiB/s
		{0, 4, f(8 << 20), nil, "moderada"},   // 8 MiB/s
		{0, 4, f(900 << 20), nil, "moderada"}, // o disco para em 10 pontos
		{60, 4, f(900 << 20), nil, "alta"},    // 15 + 10
		{0, 0, nil, nil, "nenhuma"},           // zero núcleos não divide por zero
		{50, 0, nil, nil, "alta"},             // vale como um núcleo
		{-5, 4, f(-1), f(-1), "nenhuma"},      // número torto não vira energia negativa
	}
	for _, c := range casos {
		if veio := Energia(c.cpu, c.nucleos, c.disco, c.gpu); veio != c.quer {
			t.Errorf("cpu=%v núcleos=%d disco=%v gpu=%v: veio %q, esperava %q (impacto %v)", c.cpu, c.nucleos, val(c.disco), val(c.gpu), veio, c.quer, Impacto(c.cpu, c.nucleos, c.disco, c.gpu))
		}
	}
}

func val(p *float64) any {
	if p == nil {
		return nil
	}
	return *p
}

func TestEnergiaNoGrupo(t *testing.T) {
	gs := Agrupar([]coleta.Medido{med(10, "a", "/system.slice/a.service", 100, 1, nil, nil), med(20, "b", "/system.slice/b.service", 0, 1, nil, nil)}, nil, 4, nil)
	if gs[0].Energia != "alta" || gs[1].Energia != "nenhuma" {
		t.Errorf("energia dos grupos: %q e %q", gs[0].Energia, gs[1].Energia)
	}
}

// --- médias ---

func grupoDeTeste(id string, cpu float64, mem int64, disco float64) Grupo {
	return Grupo{ID: id, CPUPct: cpu, MemKB: mem, DiscoBps: coleta.Ptr(disco)}
}

func TestMediasDeUmETresMinutos(t *testing.T) {
	h := NovoHistorico()
	seg := func(s int) time.Duration { return time.Duration(s) * time.Second }
	// três minutos e meio a 10 % de CPU, depois meio minuto a 70 %
	var gs []Grupo
	for s := 0; s <= 240; s += 10 {
		cpu, mem, disco := 10.0, int64(1000), 0.0
		if s > 210 {
			cpu, mem, disco = 70, 4000, 6000
		}
		gs = []Grupo{grupoDeTeste("app:a", cpu, mem, disco)}
		h.Registrar(seg(s), gs)
		if s == 0 && (gs[0].Media.CPUPct1m != 10 || gs[0].Media.CPUPct3m != 10) {
			t.Errorf("no primeiro retrato a média é o valor de agora: %+v", gs[0].Media)
		}
	}
	m := gs[0].Media
	// último minuto: os pontos de 190 a 240 (o de 180 já tem 60 s): 10 10 10 70 70 70
	if m.CPUPct1m != 40 || m.MemKB1m != 2500 || m.DiscoBps1m != 3000 {
		t.Errorf("média de 1 minuto: %+v", m)
	}
	// três minutos: de 70 a 240, 18 pontos, 15 a 10 e 3 a 70
	if m.CPUPct3m != 20 || m.MemKB3m != 1500 || m.DiscoBps3m != 1000 {
		t.Errorf("média de 3 minutos: %+v", m)
	}
	if n := h.Pontos("app:a"); n != 18 {
		t.Errorf("o que passou de três minutos é jogado fora: ficaram %d pontos", n)
	}
}

func TestMediaEPorTempoNaoPorContagem(t *testing.T) {
	h := NovoHistorico()
	// retratos a cada 250 ms: um minuto tem 240 pontos, não 60
	var gs []Grupo
	for i := 1; i <= 480; i++ {
		cpu := 0.0
		if i > 240 {
			cpu = 50
		}
		gs = []Grupo{grupoDeTeste("app:a", cpu, 1, 0)}
		h.Registrar(time.Duration(i)*250*time.Millisecond, gs)
	}
	if m := gs[0].Media; m.CPUPct1m != 50 || m.CPUPct3m != 25 {
		t.Errorf("com intervalo de 250 ms: %+v", m)
	}
	// um buraco de dez minutos (a máquina dormiu): o que é velho não entra
	gs = []Grupo{grupoDeTeste("app:a", 8, 1, 0)}
	h.Registrar(20*time.Minute, gs)
	if m := gs[0].Media; m.CPUPct1m != 8 || m.CPUPct3m != 8 || h.Pontos("app:a") != 1 {
		t.Errorf("depois de um buraco: %+v com %d pontos", m, h.Pontos("app:a"))
	}
}

func TestGrupoQueSomeEEsquecido(t *testing.T) {
	h := NovoHistorico()
	h.Registrar(0, []Grupo{grupoDeTeste("app:a", 100, 1, 0), grupoDeTeste("app:b", 1, 1, 0)})
	h.Registrar(time.Second, []Grupo{grupoDeTeste("app:b", 1, 1, 0)})
	if h.Pontos("app:a") != 0 || h.Pontos("app:b") != 2 {
		t.Errorf("quem sumiu é esquecido: a=%d b=%d", h.Pontos("app:a"), h.Pontos("app:b"))
	}
	gs := []Grupo{grupoDeTeste("app:a", 2, 1, 0), grupoDeTeste("app:b", 1, 1, 0)}
	h.Registrar(2*time.Second, gs)
	if gs[0].Media.CPUPct1m != 2 {
		t.Errorf("quem volta começa do zero: %+v", gs[0].Media)
	}
	// o mesmo id em escopos diferentes não se mistura
	h = NovoHistorico()
	a, b := grupoDeTeste("servico:x.service", 10, 1, 0), grupoDeTeste("servico:x.service", 90, 1, 0)
	a.Escopo, b.Escopo = "sistema", "usuario"
	gs = []Grupo{a, b}
	h.Registrar(0, gs)
	h.Registrar(time.Second, gs)
	if gs[0].Media.CPUPct1m != 10 || gs[1].Media.CPUPct1m != 90 {
		t.Errorf("escopos misturados: %+v %+v", gs[0].Media, gs[1].Media)
	}
	// disco null conta como zero na média
	g := Grupo{ID: "app:c", CPUPct: 1}
	gs = []Grupo{g}
	h.Registrar(2*time.Second, gs)
	if gs[0].Media.DiscoBps1m != 0 {
		t.Errorf("disco null: %+v", gs[0].Media)
	}
}

// --- atalhos ---

func TestAtalhos(t *testing.T) {
	r := falso.Raizes(t)
	a := NovosAtalhos()
	if _, ok := a.Procurar("firefox"); ok {
		t.Error("sem arquivo não há atalho")
	}
	falso.Escreve(t, r, "apps/firefox.desktop", "[Desktop Entry]\nName = Firefox \nIcon=firefox\n")
	if at, ok := a.Procurar("firefox"); !ok || at.Nome != "Firefox" || at.Icone != "firefox" {
		t.Errorf("atalho novo aparece sem reiniciar: %+v %v", at, ok)
	}
	falso.Escreve(t, r, "apps/Maiusculo.desktop", "[Desktop Entry]\nIcon=m\n")
	if at, ok := a.Procurar("Maiusculo"); !ok || at.Nome != "Maiusculo" || at.Icone != "m" {
		t.Errorf("sem Name o nome é o do arquivo: %+v %v", at, ok)
	}
	falso.Escreve(t, r, "apps/org.exemplo.app.desktop", "[Desktop Entry]\nName=Exemplo\n")
	if at, ok := a.Procurar("Org.Exemplo.App"); !ok || at.Nome != "Exemplo" {
		t.Errorf("a classe da janela às vezes vem com outra caixa: %+v %v", at, ok)
	}
	for _, torto := range []string{"", "../etc/passwd", "a/b", ".", "nao-existe", "com\x00nulo"} {
		if _, ok := a.Procurar(torto); ok {
			t.Errorf("%q não é nome de atalho", torto)
		}
	}
	falso.Escreve(t, r, "apps/binario.desktop", "\x00\xff\xfe[Desktop Entry\nName\n=\n[")
	if at, ok := a.Procurar("binario"); !ok || at.Nome != "binario" {
		t.Errorf("arquivo torto: existe, com o nome do arquivo: %+v %v", at, ok)
	}
	var nulo *Atalhos
	if _, ok := nulo.Procurar("firefox"); ok {
		t.Error("sem catálogo não há atalho")
	}
}

func TestAgoraLeAMaquinaDeMentira(t *testing.T) {
	naArvore(t)
	gs := Agora()
	if g := grupoDoPid(t, gs, 1240); g.ID != "app:firefox" {
		t.Errorf("Agora() agrupa o que está em /proc: %+v", g.ID)
	}
	if pids := PIDs(gs, "app:kitty", ""); len(pids) != 5 {
		t.Errorf("pids do grupo = %v", pids)
	}
	if pids := PIDs(gs, "servico:dbus-broker.service", "usuario"); len(pids) != 1 || pids[0] != 810 {
		t.Errorf("pids do serviço de usuário = %v", pids)
	}
}
