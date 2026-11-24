package coleta

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"mkpulso/internal/coleta/falso"
)

func acha(t *testing.T, ps []Processo, pid int) Processo {
	t.Helper()
	for _, p := range ps {
		if p.PID == pid {
			return p
		}
	}
	t.Fatalf("o processo %d não apareceu em %d lidos", pid, len(ps))
	return Processo{}
}

func tem(ps []Processo, pid int) bool {
	for _, p := range ps {
		if p.PID == pid {
			return true
		}
	}
	return false
}

func TestProcessoComum(t *testing.T) {
	r := falso.Raizes(t)
	falso.Escreve(t, r, "etc/passwd", "root:x:0:0::/root:/bin/bash\nmark:x:1000:1000:Mark:/home/mark:/bin/bash\n")
	falso.Proc{PID: 1234, PPID: 900, Comm: "firefox", Estado: "R", Cmd: []string{"/usr/lib/firefox/firefox", "--new-window"},
		UID: 1000, Utime: 300, Stime: 50, Threads: 80, Nice: 5, Inicio: 7000, RSS: 400000,
		Cgroup: "/user.slice/user-1000.slice/user@1000.service/app.slice/app-firefox-1234.scope", Leu: 4096, Gravou: 8192}.Grava(t, r)

	p := acha(t, LerProcessos(), 1234)
	if p.PPID != 900 || p.Nome != "firefox" || p.Estado != "rodando" {
		t.Errorf("ppid, nome ou estado errados: %+v", p)
	}
	if p.Cmd != "/usr/lib/firefox/firefox --new-window" {
		t.Errorf("cmd = %q", p.Cmd)
	}
	if p.Usuario != "mark" || p.UID != 1000 {
		t.Errorf("usuário = %q (%d)", p.Usuario, p.UID)
	}
	if p.Ticks != 350 || p.Threads != 80 || p.Nice != 5 || p.InicioTicks != 7000 || p.MemKB != 400000 {
		t.Errorf("números errados: %+v", p)
	}
	if !p.TemIO || p.Leitura != 4096 || p.Escrita != 8192 {
		t.Errorf("io errado: %+v", p)
	}
	if p.Cgroup != "/user.slice/user-1000.slice/user@1000.service/app.slice/app-firefox-1234.scope" || p.Kernel {
		t.Errorf("cgroup ou kernel errados: %+v", p)
	}
}

func TestEstadosEmPortugues(t *testing.T) {
	r := falso.Raizes(t)
	casos := map[string]string{"R": "rodando", "S": "dormindo", "D": "esperando disco", "T": "parado", "t": "parado", "Z": "zumbi", "I": "ocioso"}
	pid := 100
	esperado := map[int]string{}
	for letra, nome := range casos {
		falso.Proc{PID: pid, PPID: 1, Comm: "x", Estado: letra, Cmd: []string{"x"}, Cgroup: "/a.service"}.Grava(t, r)
		esperado[pid] = nome
		pid++
	}
	ps := LerProcessos()
	for pid, nome := range esperado {
		if p := acha(t, ps, pid); p.Estado != nome {
			t.Errorf("pid %d: estado %q, esperava %q", pid, p.Estado, nome)
		}
	}
}

func TestCommComEspacoEParenteses(t *testing.T) {
	r := falso.Raizes(t)
	falso.Proc{PID: 50, PPID: 7, Comm: "Web (Content) ) R 9", Cmd: []string{"web"}, Utime: 11, Threads: 3, Cgroup: "/a.service"}.Grava(t, r)
	p := acha(t, LerProcessos(), 50)
	if p.Nome != "Web (Content) ) R 9" || p.PPID != 7 || p.Ticks != 11 || p.Threads != 3 || p.Estado != "dormindo" {
		t.Errorf("comm com parênteses confundiu o stat: %+v", p)
	}
}

func TestCmdlineComBytesInvalidos(t *testing.T) {
	r := falso.Raizes(t)
	falso.Proc{PID: 51, PPID: 1, Comm: "torto", Cmd: []string{"/bin/torto", "\xff\xfe arquivo\xc3"}, Cgroup: "/a.service"}.Grava(t, r)
	p := acha(t, LerProcessos(), 51)
	if !utf8.ValidString(p.Cmd) || !strings.HasPrefix(p.Cmd, "/bin/torto ") || !strings.Contains(p.Cmd, "arquivo") {
		t.Errorf("cmd = %q", p.Cmd)
	}
}

func TestCmdCompridoECortado(t *testing.T) {
	r := falso.Raizes(t)
	falso.Proc{PID: 52, PPID: 1, Comm: "longo", Cmd: []string{"/bin/longo", strings.Repeat("ção", 2000)}, Cgroup: "/a.service"}.Grava(t, r)
	p := acha(t, LerProcessos(), 52)
	if len(p.Cmd) > LimiteCmd+4 || !utf8.ValidString(p.Cmd) || !strings.HasSuffix(p.Cmd, "…") {
		t.Errorf("cmd com %d bytes, final %q", len(p.Cmd), p.Cmd[len(p.Cmd)-8:])
	}
}

func TestNomeDestruncado(t *testing.T) {
	r := falso.Raizes(t)
	falso.Proc{PID: 53, PPID: 1, Comm: "xdg-desktop-por", Cmd: []string{"/usr/lib/xdg-desktop-portal-hyprland"}, Cgroup: "/a.service"}.Grava(t, r)
	falso.Proc{PID: 54, PPID: 1, Comm: "Isolated Web Co", Cmd: []string{"/usr/lib/firefox/firefox", "-contentproc"}, Cgroup: "/a.service"}.Grava(t, r)
	falso.Proc{PID: 55, PPID: 1, Comm: "bash", Cmd: []string{"-bash"}, Cgroup: "/a.service"}.Grava(t, r)
	ps := LerProcessos()
	if n := acha(t, ps, 53).Nome; n != "xdg-desktop-portal-hyprland" {
		t.Errorf("nome cortado em 15 letras não foi completado: %q", n)
	}
	if n := acha(t, ps, 54).Nome; n != "Isolated Web Co" {
		t.Errorf("nome que o processo escolheu foi trocado: %q", n)
	}
	if n := acha(t, ps, 55).Nome; n != "bash" {
		t.Errorf("nome = %q", n)
	}
}

func TestIoIlegivel(t *testing.T) {
	r := falso.Raizes(t)
	falso.Proc{PID: 60, PPID: 1, Comm: "alheio", Cmd: []string{"alheio"}, SemIO: true, Cgroup: "/a.service"}.Grava(t, r)
	if p := acha(t, LerProcessos(), 60); p.TemIO {
		t.Errorf("io ilegível deveria ficar sem leitura: %+v", p)
	}
}

func TestUsuarioSemPasswd(t *testing.T) {
	r := falso.Raizes(t)
	falso.Escreve(t, r, "etc/passwd", "linha torta\n::\nmark:x:abc:1\n")
	falso.Proc{PID: 61, PPID: 1, Comm: "x", Cmd: []string{"x"}, UID: 4242, Cgroup: "/a.service"}.Grava(t, r)
	if p := acha(t, LerProcessos(), 61); p.Usuario != "4242" {
		t.Errorf("usuário desconhecido deveria sair como o número: %q", p.Usuario)
	}
}

func TestProcessoQueSomeEIgnorado(t *testing.T) {
	r := falso.Raizes(t)
	falso.Proc{PID: 70, PPID: 1, Comm: "fica", Cmd: []string{"fica"}, Cgroup: "/a.service"}.Grava(t, r)
	falso.Proc{PID: 71, PPID: 1, Comm: "some", Cmd: []string{"some"}, Cgroup: "/a.service"}.Grava(t, r)
	falso.Apaga(t, r, "proc/71/stat")
	falso.Proc{PID: 72, PPID: 1, Comm: "some2", Cmd: []string{"some"}, Cgroup: "/a.service"}.Grava(t, r)
	falso.Apaga(t, r, "proc/72/status")
	falso.Escreve(t, r, "proc/73/outra-coisa", "x")
	falso.Escreve(t, r, "proc/nao-e-pid/stat", "x")
	ps := LerProcessos()
	if !tem(ps, 70) || tem(ps, 71) || tem(ps, 72) || tem(ps, 73) || len(ps) != 1 {
		t.Errorf("esperava só o 70, vieram %d: %+v", len(ps), ps)
	}
}

func TestStatTorto(t *testing.T) {
	r := falso.Raizes(t)
	falso.Proc{PID: 80, PPID: 1, Comm: "vazio", Cmd: []string{"x"}, Cgroup: "/a.service"}.Grava(t, r)
	falso.Escreve(t, r, "proc/80/stat", "")
	falso.Proc{PID: 81, PPID: 1, Comm: "cortado", Cmd: []string{"x"}, Cgroup: "/a.service"}.Grava(t, r)
	falso.Escreve(t, r, "proc/81/stat", "81 (cortado) S 1 81")
	falso.Proc{PID: 82, PPID: 1, Comm: "semfim", Cmd: []string{"x"}, Cgroup: "/a.service"}.Grava(t, r)
	falso.Escreve(t, r, "proc/82/stat", "82 (semfim S 1 2 3")
	falso.Proc{PID: 83, PPID: 1, Comm: "gigante", Cmd: []string{"x"}, Cgroup: "/a.service"}.Grava(t, r)
	falso.Escreve(t, r, "proc/83/stat", "83 (gigante) S 1 83 83 0 -1 0 0 0 0 0 99999999999999999999999999 18446744073709551615 0 0 20 0 99999999999999999999 0 5 0 0\n")
	falso.Escreve(t, r, "proc/83/status", "Uid:\t99999999999999999999\nVmRSS:\t99999999999999999999999 kB\n")
	falso.Escreve(t, r, "proc/83/io", "read_bytes: 99999999999999999999999\nwrite_bytes: -5\n")

	ps := LerProcessos()
	if tem(ps, 80) || tem(ps, 81) || tem(ps, 82) {
		t.Errorf("stat vazio ou cortado deveria ser ignorado: %+v", ps)
	}
	g := acha(t, ps, 83)
	if g.PPID != 1 || g.InicioTicks != 5 || g.MemKB < 0 {
		t.Errorf("número gigante derrubou a leitura: %+v", g)
	}
}

func TestThreadsDoKernel(t *testing.T) {
	r := falso.Raizes(t)
	falso.Proc{PID: 2, PPID: 0, Comm: "kthreadd"}.Grava(t, r)
	falso.Proc{PID: 15, PPID: 2, Comm: "kworker/0:1-events", Estado: "I"}.Grava(t, r)
	falso.Proc{PID: 16, PPID: 1, Comm: "semcmd", Bandeiras: 0x00200000}.Grava(t, r)
	falso.Proc{PID: 17, PPID: 1, Comm: "zumbi", Estado: "Z", Cgroup: "/a.service"}.Grava(t, r)
	falso.Proc{PID: 18, PPID: 1, Comm: "gente", Cmd: []string{"gente"}, RSS: 10, Cgroup: "/a.service"}.Grava(t, r)
	falso.Proc{PID: 19, PPID: 1, Comm: "semcmd2"}.Grava(t, r)
	ps := LerProcessos()
	for _, pid := range []int{2, 15, 16, 19} {
		if p := acha(t, ps, pid); !p.Kernel {
			t.Errorf("pid %d deveria ser thread do kernel: %+v", pid, p)
		}
	}
	if acha(t, ps, 15).Cmd != "[kworker/0:1-events]" {
		t.Errorf("cmd de thread do kernel = %q", acha(t, ps, 15).Cmd)
	}
	if acha(t, ps, 17).Kernel || acha(t, ps, 18).Kernel {
		t.Errorf("zumbi e processo comum não são do kernel")
	}
}

func TestSemProcNaoQuebra(t *testing.T) {
	r := falso.Raizes(t)
	falso.Apaga(t, r, "proc")
	if ps := LerProcessos(); len(ps) != 0 {
		t.Errorf("sem /proc deveria vir vazio: %+v", ps)
	}
	a := Colher(0)
	if a == nil || len(a.Processos) != 0 || a.TemCPU {
		t.Errorf("amostra sem /proc: %+v", a)
	}
}

// --- taxas ---

func medido(t *testing.T, ms []Medido, pid int) Medido {
	t.Helper()
	for _, m := range ms {
		if m.PID == pid {
			return m
		}
	}
	t.Fatalf("o processo %d não foi medido", pid)
	return Medido{}
}

func TestCpuEDiscoPorProcesso(t *testing.T) {
	r := falso.Raizes(t)
	falso.Stat(t, r, [2]uint64{100, 900}, [2]uint64{100, 900})
	falso.Proc{PID: 10, PPID: 1, Comm: "a", Cmd: []string{"a"}, Utime: 100, Stime: 100, Inicio: 500, Cgroup: "/a.service", Leu: 1000, Gravou: 1000}.Grava(t, r)
	falso.Proc{PID: 11, PPID: 1, Comm: "velho", Cmd: []string{"velho"}, Utime: 900, Inicio: 600, Cgroup: "/a.service", SemIO: true}.Grava(t, r)
	antes := Colher(10 * time.Second)

	falso.Stat(t, r, [2]uint64{150, 950}, [2]uint64{200, 900})
	falso.Proc{PID: 10, PPID: 1, Comm: "a", Cmd: []string{"a"}, Utime: 250, Stime: 150, Inicio: 500, Cgroup: "/a.service", Leu: 3000, Gravou: 5000}.Grava(t, r)
	// o pid 11 foi reaproveitado por outro processo (outro início)
	falso.Proc{PID: 11, PPID: 1, Comm: "novo", Cmd: []string{"novo"}, Utime: 5000, Inicio: 99000, Cgroup: "/a.service", SemIO: true}.Grava(t, r)
	falso.Proc{PID: 12, PPID: 1, Comm: "nasceu", Cmd: []string{"nasceu"}, Utime: 700, Inicio: 99100, Cgroup: "/a.service", Leu: 99999}.Grava(t, r)
	depois := Colher(12 * time.Second)

	ms := MedirProcessos(antes, depois)
	a := medido(t, ms, 10)
	if a.CPUPct != 100 { // 200 ticks em 2 s a 100 ticks/s
		t.Errorf("cpu de quem gastou um núcleo inteiro = %v", a.CPUPct)
	}
	if a.DiscoBps == nil || *a.DiscoBps != 3000 { // (2000+4000)/2 s
		t.Errorf("disco = %v", a.DiscoBps)
	}
	if a.InicioS == nil || *a.InicioS != 1795000005 {
		t.Errorf("início = %v", a.InicioS)
	}
	if m := medido(t, ms, 11); m.CPUPct != 0 || m.DiscoBps != nil {
		t.Errorf("pid reaproveitado deveria começar do zero e sem disco: %+v", m)
	}
	if m := medido(t, ms, 12); m.CPUPct != 0 || m.DiscoBps == nil || *m.DiscoBps != 0 {
		t.Errorf("processo novo tem 0 no primeiro retrato: cpu %v disco %v", m.CPUPct, m.DiscoBps)
	}
}

func TestMedirSemLeituraAnteriorOuSemTempo(t *testing.T) {
	r := falso.Raizes(t)
	falso.Stat(t, r, [2]uint64{100, 900})
	falso.Proc{PID: 10, PPID: 1, Comm: "a", Cmd: []string{"a"}, Utime: 100, Cgroup: "/a.service"}.Grava(t, r)
	a := Colher(time.Second)
	if m := medido(t, MedirProcessos(nil, a), 10); m.CPUPct != 0 {
		t.Errorf("sem leitura anterior a cpu é 0: %v", m.CPUPct)
	}
	falso.Proc{PID: 10, PPID: 1, Comm: "a", Cmd: []string{"a"}, Utime: 900, Cgroup: "/a.service"}.Grava(t, r)
	b := Colher(time.Second) // o relógio não andou
	if m := medido(t, MedirProcessos(a, b), 10); m.CPUPct != 0 {
		t.Errorf("sem tempo decorrido a cpu é 0, não infinito: %v", m.CPUPct)
	}
	// contador que anda para trás não vira número negativo
	falso.Proc{PID: 10, PPID: 1, Comm: "a", Cmd: []string{"a"}, Utime: 1, Cgroup: "/a.service"}.Grava(t, r)
	c := Colher(3 * time.Second)
	if m := medido(t, MedirProcessos(b, c), 10); m.CPUPct != 0 {
		t.Errorf("contador que voltou deveria dar 0: %v", m.CPUPct)
	}
}

func TestCpuTotalEPorNucleo(t *testing.T) {
	r := falso.Raizes(t)
	falso.Stat(t, r, [2]uint64{100, 900}, [2]uint64{100, 900})
	antes := Colher(0)
	falso.Stat(t, r, [2]uint64{150, 950}, [2]uint64{200, 900})
	depois := Colher(time.Second)
	c := MedirCPU(antes, depois)
	if c.Pct == nil || *c.Pct != 75 { // 150 ocupados em 200
		t.Errorf("cpu total = %v", c.Pct)
	}
	if len(c.PorNucleo) != 2 || c.PorNucleo[0] != 50 || c.PorNucleo[1] != 100 {
		t.Errorf("por núcleo = %v", c.PorNucleo)
	}
	if depois.Nucleos() != 2 {
		t.Errorf("núcleos = %d", depois.Nucleos())
	}
	if c := MedirCPU(nil, depois); c.Pct != nil || c.PorNucleo == nil || len(c.PorNucleo) != 0 {
		t.Errorf("sem leitura anterior: pct null e lista vazia, veio %+v", c)
	}
}

func TestProcStatTorto(t *testing.T) {
	r := falso.Raizes(t)
	for _, texto := range []string{"", "cpu\n", "cpu  1 2\ncpu0 x y z w\n", "lixo\x00\xff", "cpu  99999999999999999999999 1 1 1\n"} {
		falso.Escreve(t, r, "proc/stat", texto)
		a := Colher(0)
		b := Colher(time.Second)
		c := MedirCPU(a, b)
		for _, v := range c.PorNucleo {
			if v < 0 || v > 100 {
				t.Errorf("%q: núcleo fora de 0..100: %v", texto, v)
			}
		}
		if c.Pct != nil && (*c.Pct < 0 || *c.Pct > 100) {
			t.Errorf("%q: total fora de 0..100: %v", texto, *c.Pct)
		}
	}
}
