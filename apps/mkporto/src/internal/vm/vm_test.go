package vm

import (
	"bufio"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func loja(t *testing.T) Loja {
	t.Helper()
	base := t.TempDir()
	return Loja{Raiz: filepath.Join(base, "maquinas"), Execucao: filepath.Join(base, "execucao")}
}

func boa() Maquina {
	return Maquina{Nome: "oficina", MemoriaMB: 4096, CPUs: 2, DiscoGB: 40, PortaSSH: 2222, CriadaEm: 1795000000}
}

// --- validação ------------------------------------------------------------------

func TestNomeERotulo(t *testing.T) {
	for _, bom := range []string{"a", "oficina", "win-11", "teste_2.novo", "0", strings.Repeat("a", 40)} {
		if !NomeBom(bom) || ErroDoNome(bom) != "" {
			t.Errorf("nome %q devia servir", bom)
		}
	}
	for _, ruim := range []string{"", "-x", ".x", "_x", "Oficina", "a b", "a/b", "..", "máquina", "a\nb", "a,b", "a;b", strings.Repeat("a", 41)} {
		if NomeBom(ruim) || ErroDoNome(ruim) == "" {
			t.Errorf("nome %q não devia servir", ruim)
		}
	}
	for _, bom := range []string{"antes", "Antes-de-atualizar_2.0", "1", strings.Repeat("a", 40)} {
		if !RotuloBom(bom) {
			t.Errorf("rótulo %q devia servir", bom)
		}
	}
	for _, ruim := range []string{"", "-a", "a b", "a/b", "ação", "a\nb", strings.Repeat("a", 41)} {
		if RotuloBom(ruim) {
			t.Errorf("rótulo %q não devia servir", ruim)
		}
	}
}

func TestValida_OsLimitesDeCadaCampo(t *testing.T) {
	com := func(muda func(*Maquina)) Maquina {
		m := boa()
		muda(&m)
		return m
	}
	if err := Valida(boa()); err != nil {
		t.Fatalf("a boa: %v", err)
	}
	for _, m := range []Maquina{
		com(func(m *Maquina) { m.MemoriaMB = 512 }), com(func(m *Maquina) { m.MemoriaMB = 65536 }),
		com(func(m *Maquina) { m.CPUs = 1 }), com(func(m *Maquina) { m.CPUs = 32 }),
		com(func(m *Maquina) { m.DiscoGB = 1 }), com(func(m *Maquina) { m.DiscoGB = 2000 }),
		com(func(m *Maquina) { m.ISO = "/isos/um sistema (novo).iso" }),
	} {
		if err := Valida(m); err != nil {
			t.Errorf("%+v: %v", m, err)
		}
	}
	casos := []struct {
		m     Maquina
		campo string
	}{
		{com(func(m *Maquina) { m.Nome = "" }), "nome"},
		{com(func(m *Maquina) { m.Nome = "-name" }), "nome"},
		{com(func(m *Maquina) { m.Nome = "../fora" }), "nome"},
		{com(func(m *Maquina) { m.MemoriaMB = 511 }), "memoria_mb"},
		{com(func(m *Maquina) { m.MemoriaMB = 65537 }), "memoria_mb"},
		{com(func(m *Maquina) { m.CPUs = 0 }), "cpus"},
		{com(func(m *Maquina) { m.CPUs = 33 }), "cpus"},
		{com(func(m *Maquina) { m.DiscoGB = 0 }), "disco_gb"},
		{com(func(m *Maquina) { m.DiscoGB = 2001 }), "disco_gb"},
		{com(func(m *Maquina) { m.ISO = "relativo.iso" }), "iso"},
		{com(func(m *Maquina) { m.ISO = "/isos/a,b.iso" }), "iso"},
		{com(func(m *Maquina) { m.ISO = "/isos/a.iso,readonly=off" }), "iso"},
		{com(func(m *Maquina) { m.ISO = "/isos/a\n.iso" }), "iso"},
		{com(func(m *Maquina) { m.ISO = "/isos/a\x07.iso" }), "iso"},
		{com(func(m *Maquina) { m.ISO = "-cdrom" }), "iso"},
	}
	for _, c := range casos {
		err := Valida(c.m)
		if err == nil || err.Campo != c.campo || err.Frase == "" {
			t.Errorf("%+v: %v (esperado campo %s)", c.m, err, c.campo)
		}
	}
	if f := ErroDaISO("/a,b.iso"); !strings.Contains(f, "vírgula") {
		t.Errorf("a frase da vírgula tem de dizer por quê: %q", f)
	}
}

// --- a loja ---------------------------------------------------------------------

func TestLoja_CriaGravaLeEListaEmOrdem(t *testing.T) {
	l := loja(t)
	if len(l.Todas()) != 0 {
		t.Fatalf("sem a pasta, nenhuma máquina")
	}
	for _, nome := range []string{"zebra", "oficina"} {
		m := boa()
		m.Nome = nome
		if err := l.CriaPasta(nome); err != nil {
			t.Fatal(err)
		}
		if err := l.Grava(m); err != nil {
			t.Fatal(err)
		}
	}
	info, _ := os.Stat(l.Pasta("oficina"))
	if info.Mode().Perm() != 0o700 {
		t.Errorf("pasta: %o", info.Mode().Perm())
	}
	// Uma pasta sem maquina.json, um arquivo solto e um json estragado não entram.
	_ = os.Mkdir(l.Pasta("vazia"), 0o700)
	_ = os.WriteFile(filepath.Join(l.Raiz, "solto"), []byte("x"), 0o600)
	_ = os.Mkdir(l.Pasta("estragada"), 0o700)
	_ = os.WriteFile(l.Config("estragada"), []byte("{"), 0o600)
	// Um maquina.json que diz outro nome (copiado à mão) também não.
	_ = os.Mkdir(l.Pasta("copia"), 0o700)
	b, _ := json.Marshal(boa())
	_ = os.WriteFile(l.Config("copia"), b, 0o600)

	todas := l.Todas()
	if len(todas) != 2 || todas[0].Nome != "oficina" || todas[1].Nome != "zebra" {
		t.Fatalf("todas: %+v", todas)
	}
	m, err := l.Le("oficina")
	if err != nil || !reflect.DeepEqual(m, boa()) {
		t.Errorf("leu: %+v %v", m, err)
	}
	if _, err := l.Le("fantasma"); err == nil || !strings.Contains(err.Error(), "não conheço") {
		t.Errorf("fantasma: %v", err)
	}
	if _, err := l.Le("../oficina"); err == nil {
		t.Errorf("nome com barra tem de ser recusado antes de abrir")
	}
	if !l.Existe("vazia") || l.Existe("fantasma") {
		t.Errorf("Existe")
	}
	if err := l.CriaPasta("oficina"); err == nil {
		t.Errorf("criar por cima devia falhar")
	}
	// A gravação não deixa temporário.
	itens, _ := os.ReadDir(l.Pasta("oficina"))
	if len(itens) != 1 {
		t.Errorf("sobrou: %v", itens)
	}
}

func TestApaga_SoAPastaDaMaquinaENadaFora(t *testing.T) {
	l := loja(t)
	_ = l.CriaPasta("oficina")
	_ = l.Grava(boa())
	_ = l.PreparaCorrida("oficina")
	_ = os.WriteFile(l.Log("oficina"), []byte("x"), 0o600)
	fora := filepath.Join(filepath.Dir(l.Raiz), "fora")
	_ = os.MkdirAll(fora, 0o700)
	_ = os.WriteFile(filepath.Join(fora, "precioso"), []byte("x"), 0o600)
	// Um atalho, dentro das máquinas, para uma pasta de fora.
	if err := os.Symlink(fora, l.Pasta("atalho")); err != nil {
		t.Fatal(err)
	}

	for _, nome := range []string{"", ".", "..", "../fora", "/", "oficina/..", "atalho", "fantasma"} {
		if err := l.Apaga(nome); err == nil {
			t.Errorf("Apaga(%q) devia recusar", nome)
		}
	}
	if _, err := os.Stat(filepath.Join(fora, "precioso")); err != nil {
		t.Fatalf("apagou fora das máquinas")
	}
	if _, err := os.Stat(l.Config("oficina")); err != nil {
		t.Fatalf("apagou a máquina numa recusa")
	}
	// Loja sem raiz nunca apaga nada.
	if err := (Loja{}).Apaga("oficina"); err == nil {
		t.Errorf("raiz vazia devia recusar")
	}

	if err := l.Apaga("oficina"); err != nil {
		t.Fatal(err)
	}
	for _, sumiu := range []string{l.Pasta("oficina"), l.Corrida("oficina")} {
		if _, err := os.Stat(sumiu); err == nil {
			t.Errorf("ficou: %s", sumiu)
		}
	}
	for _, ficou := range []string{l.Raiz, l.Execucao, fora} {
		if _, err := os.Stat(ficou); err != nil {
			t.Errorf("sumiu: %s", ficou)
		}
	}
}

func TestCorrida_PreparaLimpaEOFimDoLog(t *testing.T) {
	l := loja(t)
	if err := l.PreparaCorrida("oficina"); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(l.Corrida("oficina"))
	if info.Mode().Perm() != 0o700 {
		t.Errorf("pasta de execução: %o", info.Mode().Perm())
	}
	_ = os.WriteFile(l.Pid("oficina"), []byte("123\n"), 0o600)
	_ = os.WriteFile(l.Socket("oficina"), nil, 0o600)
	if err := l.PreparaCorrida("oficina"); err != nil {
		t.Fatal(err)
	}
	for _, velho := range []string{l.Pid("oficina"), l.Socket("oficina")} {
		if _, err := os.Stat(velho); err == nil {
			t.Errorf("sobrou de uma corrida anterior: %s", velho)
		}
	}
	if err := (Loja{Raiz: l.Raiz}).PreparaCorrida("oficina"); err == nil || !strings.Contains(err.Error(), "XDG_RUNTIME_DIR") {
		t.Errorf("sem pasta de execução: %v", err)
	}

	f, err := l.AbreLog("oficina")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString("primeira\n\nqemu: aviso \x1b[31mcolorido\n" + strings.Repeat("x", 300) + "\nqemu-system-x86_64: -cdrom /x.iso: Could not open\n\n")
	f.Close()
	fim := l.FimDoLog("oficina", 2)
	if !strings.HasSuffix(fim, " / qemu-system-x86_64: -cdrom /x.iso: Could not open") || strings.Contains(fim, "primeira") || strings.Contains(fim, "\x1b") {
		t.Errorf("fim do log: %q", fim)
	}
	if !strings.HasPrefix(fim, strings.Repeat("x", 200)+"…") {
		t.Errorf("linha comprida não foi cortada: %q", fim[:30])
	}
	if l.FimDoLog("fantasma", 3) != "" {
		t.Errorf("sem log, nada")
	}
}

func TestCopiaVarsENoDisco(t *testing.T) {
	l := loja(t)
	_ = l.CriaPasta("oficina")
	modelo := filepath.Join(t.TempDir(), OvmfVars)
	_ = os.WriteFile(modelo, []byte("variáveis"), 0o644)
	if err := l.CopiaVars("oficina", modelo); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(l.Vars("oficina"))
	info, _ := os.Stat(l.Vars("oficina"))
	if string(b) != "variáveis" || info.Mode().Perm() != 0o600 {
		t.Errorf("vars: %q %o", b, info.Mode().Perm())
	}
	if err := l.CopiaVars("oficina", modelo); err == nil {
		t.Errorf("nunca por cima do vars de uma máquina")
	}
	if err := l.CopiaVars("oficina", modelo+".nao"); err == nil {
		t.Errorf("sem o modelo é erro")
	}
	if n := NoDisco(l.Vars("oficina")); n < 0 {
		t.Errorf("NoDisco: %d", n)
	}
	if NoDisco(filepath.Join(l.Raiz, "nada")) != -1 {
		t.Errorf("arquivo que não existe")
	}
}

// --- a linha de comando do QEMU ---------------------------------------------------

// par acha "-opcao valor" na lista.
func par(argv []string, opcao, valor string) bool {
	for i := 0; i+1 < len(argv); i++ {
		if argv[i] == opcao && argv[i+1] == valor {
			return true
		}
	}
	return false
}

func tem(argv []string, item string) bool {
	for _, a := range argv {
		if a == item {
			return true
		}
	}
	return false
}

func TestArgv_AMaquinaInteira(t *testing.T) {
	l := Loja{Raiz: "/m", Execucao: "/e"}
	m := boa()
	m.ISO, m.UEFI = "/isos/sistema novo.iso", true
	argv, avisos := l.Argv(m, Ambiente{Qemu: "qemu-system-x86_64", KVM: true, OVMF: "/ovmf", Display: "gtk"})
	esperado := []string{"qemu-system-x86_64",
		"-name", "oficina",
		"-machine", "q35",
		"-enable-kvm", "-cpu", "host",
		"-smp", "2",
		"-m", "4096",
		"-drive", "if=pflash,format=raw,readonly=on,file=/ovmf/OVMF_CODE.4m.fd",
		"-drive", "if=pflash,format=raw,file=/m/oficina/vars.fd",
		"-drive", "file=/m/oficina/disco.qcow2,if=virtio,format=qcow2",
		"-cdrom", "/isos/sistema novo.iso",
		"-boot", "order=cd,menu=on",
		"-nic", "user,model=virtio-net-pci,hostfwd=tcp:127.0.0.1:2222-:22",
		"-device", "qemu-xhci", "-device", "usb-tablet",
		"-audiodev", "pipewire,id=som", "-device", "intel-hda", "-device", "hda-duplex,audiodev=som",
		"-vga", "none",
		"-device", "virtio-vga", "-display", "gtk",
		"-qmp", "unix:/e/oficina/qmp.sock,server=on,wait=off",
		"-pidfile", "/e/oficina/pid",
		"-D", "/e/oficina/qemu-depuracao.log"}
	if !reflect.DeepEqual(argv, esperado) {
		t.Errorf("argv:\n%q\nesperado:\n%q", argv, esperado)
	}
	if len(avisos) != 0 {
		t.Errorf("avisos: %v", avisos)
	}
}

func TestArgv_AsVariacoes(t *testing.T) {
	l := Loja{Raiz: "/m", Execucao: "/e"}
	amb := Ambiente{Qemu: "q", KVM: true, OVMF: "/ovmf", Display: "gtk"}

	// Sem KVM: emulada, e a pessoa é avisada.
	semKvm := amb
	semKvm.KVM = false
	argv, avisos := l.Argv(boa(), semKvm)
	if tem(argv, "-enable-kvm") || !par(argv, "-cpu", "max") || len(avisos) != 1 || !strings.Contains(avisos[0], "lenta") {
		t.Errorf("sem kvm: %q %v", argv, avisos)
	}
	// Sem UEFI: nenhum pflash. Sem ISO: nenhum -cdrom, e o disco na frente.
	argv, _ = l.Argv(boa(), amb)
	for _, a := range argv {
		if strings.Contains(a, "pflash") || a == "-cdrom" {
			t.Errorf("sobrou %q", a)
		}
	}
	if !par(argv, "-boot", "order=c,menu=on") {
		t.Errorf("boot sem ISO: %q", argv)
	}
	// --sem-iso numa máquina que tem ISO.
	comIso := boa()
	comIso.ISO = "/x.iso"
	semIso := amb
	semIso.SemISO = true
	if argv, _ = l.Argv(comIso, semIso); tem(argv, "-cdrom") || !par(argv, "-boot", "order=c,menu=on") {
		t.Errorf("--sem-iso: %q", argv)
	}
	// 3D: o vídeo acelerado e o gl na janela.
	tresD := boa()
	tresD.TresD = true
	if argv, _ = l.Argv(tresD, amb); !par(argv, "-device", "virtio-vga-gl") || !par(argv, "-display", "gtk,gl=on") || par(argv, "-device", "virtio-vga") {
		t.Errorf("3d: %q", argv)
	}
	// Sem janela (display none): sem som, e nunca o vídeo acelerado, nem com 3D.
	semTela := amb
	semTela.Display = "none"
	argv, _ = l.Argv(tresD, semTela)
	if !par(argv, "-display", "none") || !par(argv, "-device", "virtio-vga") || tem(argv, "-audiodev") || par(argv, "-device", "intel-hda") {
		t.Errorf("display none: %q", argv)
	}
	for _, a := range argv {
		if strings.Contains(a, "gl") || strings.Contains(a, "gtk") {
			t.Errorf("display none com %q", a)
		}
	}
	// Em todas: uma tela só, o QMP num socket, o pid, e nada de daemonize.
	for _, a := range [][]string{argv} {
		if !par(a, "-vga", "none") || tem(a, "-daemonize") || !par(a, "-pidfile", "/e/oficina/pid") {
			t.Errorf("%q", a)
		}
	}
}

func TestArgv_VirgulaNoCaminhoVaiDobrada(t *testing.T) {
	l := Loja{Raiz: "/casa/ana, a boa/m", Execucao: "/run/u,1/e"}
	m := boa()
	m.UEFI = true
	argv, _ := l.Argv(m, Ambiente{Qemu: "q", OVMF: "/ovmf,x", Display: "gtk"})
	for _, quer := range [][2]string{
		{"-drive", "if=pflash,format=raw,readonly=on,file=/ovmf,,x/OVMF_CODE.4m.fd"},
		{"-drive", "if=pflash,format=raw,file=/casa/ana,, a boa/m/oficina/vars.fd"},
		{"-drive", "file=/casa/ana,, a boa/m/oficina/disco.qcow2,if=virtio,format=qcow2"},
		{"-qmp", "unix:/run/u,,1/e/oficina/qmp.sock,server=on,wait=off"},
		// O -pidfile e o -D são um arquivo só, não uma lista de opções.
		{"-pidfile", "/run/u,1/e/oficina/pid"},
	} {
		if !par(argv, quer[0], quer[1]) {
			t.Errorf("falta %s %s em %q", quer[0], quer[1], argv)
		}
	}
}

// --- snapshots ------------------------------------------------------------------

func TestLeSnapshots_AsSaidasDoQemuImg(t *testing.T) {
	nova := `Snapshot list:
ID        TAG               VM SIZE                DATE     VM CLOCK     ICOUNT
1         antes                 0 B 2026-11-24 10:00:00 00:00:00.000          0
2         antes-de-atualizar 1.21 GiB 2026-11-25 08:30:15 01:02:03.456         --
3         com espaço no nome  512 MiB 2026-11-26 09:00:00 00:10:00.000
`
	esperado := []Snapshot{
		{"1", "antes", "2026-11-24 10:00:00", "0 B"},
		{"2", "antes-de-atualizar", "2026-11-25 08:30:15", "1.21 GiB"},
		{"3", "com espaço no nome", "2026-11-26 09:00:00", "512 MiB"},
	}
	if obtido := LeSnapshots(nova); !reflect.DeepEqual(obtido, esperado) {
		t.Errorf("nova:\n%+v", obtido)
	}
	// O formato antigo: o tamanho colado na unidade, ou só "0".
	antiga := "Snapshot list:\nID        TAG                 VM SIZE                DATE       VM CLOCK\n" +
		"1         antes                     0 2019-03-01 12:00:00   00:00:00.000\n" +
		"2         depois                 1.1G 2019-03-02 12:00:00   00:01:00.000\n"
	esperado = []Snapshot{{"1", "antes", "2019-03-01 12:00:00", "0"}, {"2", "depois", "2019-03-02 12:00:00", "1.1G"}}
	if obtido := LeSnapshots(antiga); !reflect.DeepEqual(obtido, esperado) {
		t.Errorf("antiga:\n%+v", obtido)
	}
	for _, vazia := range []string{"", "\n", "Snapshot list:\nID TAG VM SIZE DATE VM CLOCK\n", "lixo sem data\n", "2026-11-24 10:00:00\n"} {
		if obtido := LeSnapshots(vazia); len(obtido) != 0 {
			t.Errorf("%q: %+v", vazia, obtido)
		}
	}
	if b, _ := json.Marshal(LeSnapshots("")); string(b) != "[]" {
		t.Errorf("sem snapshot é [], não null: %s", b)
	}
}

// --- QMP ------------------------------------------------------------------------

// pastaCurta é uma pasta temporária de caminho curto: o caminho de um socket
// unix não passa de uns 100 caracteres.
func pastaCurta(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "mkp")
	if err != nil || len(dir) > 60 {
		if err == nil {
			os.RemoveAll(dir)
		}
		if dir, err = os.MkdirTemp("/tmp", "mkp"); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

// servidorQMP é um QEMU de mentira num socket unix: saúda, aceita o
// qmp_capabilities e responde a cada comando com o que `responde` disser
// (várias linhas, se quiser: eventos antes da resposta). `recebidos` anota os
// comandos que chegaram.
func servidorQMP(t *testing.T, saudacao string, responde func(comando string) []string) (socket string, recebidos chan string) {
	t.Helper()
	socket = filepath.Join(pastaCurta(t), "qmp.sock")
	l, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	recebidos = make(chan string, 32)
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			go func() {
				defer c.Close()
				if saudacao != "" {
					_, _ = c.Write([]byte(saudacao + "\r\n"))
				}
				leitor := bufio.NewScanner(c)
				for leitor.Scan() {
					var pedido struct {
						Execute string `json:"execute"`
					}
					if json.Unmarshal(leitor.Bytes(), &pedido) != nil {
						continue
					}
					recebidos <- pedido.Execute
					for _, linha := range responde(pedido.Execute) {
						_, _ = c.Write([]byte(linha + "\r\n"))
					}
				}
			}()
		}
	}()
	return socket, recebidos
}

const saudacaoDoQemu = `{"QMP": {"version": {"qemu": {"micro": 0, "minor": 1, "major": 10}, "package": ""}, "capabilities": ["oob"]}}`

func normal(estado string) func(string) []string {
	return func(comando string) []string {
		switch comando {
		case "qmp_capabilities", "stop", "cont", "system_powerdown", "quit":
			return []string{`{"return": {}}`}
		case "query-status":
			return []string{estado}
		}
		return []string{`{"error": {"class": "CommandNotFound", "desc": "The command ` + comando + ` has not been found"}}`}
	}
}

func encurta(t *testing.T, d time.Duration) {
	t.Helper()
	antigo := PrazoDoQMP
	PrazoDoQMP = d
	t.Cleanup(func() { PrazoDoQMP = antigo })
}

func TestQMP_SaudacaoCapacidadesComandoEResposta(t *testing.T) {
	socket, recebidos := servidorQMP(t, saudacaoDoQemu, normal(`{"return": {"status": "running", "singlestep": false, "running": true}}`))
	estado, err := Estado(socket)
	if err != nil || estado != "ligada" {
		t.Fatalf("estado: %q %v", estado, err)
	}
	// A ordem da conversa: primeiro as capacidades, depois o comando.
	if a, b := <-recebidos, <-recebidos; a != "qmp_capabilities" || b != "query-status" {
		t.Errorf("conversa: %s, %s", a, b)
	}
	if _, err := Pergunta(socket, "system_powerdown"); err != nil {
		t.Errorf("powerdown: %v", err)
	}
	if _, err := Pergunta(socket, "inventado"); err == nil || !strings.Contains(err.Error(), "has not been found") {
		t.Errorf("o erro do QEMU tem de chegar: %v", err)
	}
}

func TestQMP_Pausada(t *testing.T) {
	socket, _ := servidorQMP(t, saudacaoDoQemu, normal(`{"return": {"status": "paused", "running": false}}`))
	if estado, err := Estado(socket); err != nil || estado != "pausada" {
		t.Errorf("estado: %q %v", estado, err)
	}
	// Qualquer outro estado de uma máquina viva conta como ligada.
	socket, _ = servidorQMP(t, saudacaoDoQemu, normal(`{"return": {"status": "suspended", "running": false}}`))
	if estado, _ := Estado(socket); estado != "ligada" {
		t.Errorf("suspensa: %q", estado)
	}
}

func TestQMP_EventosNoMeioSaoIgnorados(t *testing.T) {
	evento := `{"timestamp": {"seconds": 1795000000, "microseconds": 1}, "event": "RTC_CHANGE", "data": {"offset": 0}}`
	socket, _ := servidorQMP(t, saudacaoDoQemu, func(comando string) []string {
		if comando == "query-status" {
			return []string{evento, evento, `{"return": {"status": "running", "running": true}}`, evento}
		}
		return []string{evento, `{"return": {}}`}
	})
	if estado, err := Estado(socket); err != nil || estado != "ligada" {
		t.Errorf("estado: %q %v", estado, err)
	}
}

func TestQMP_MudoSemSaudacaoOuSemSocket(t *testing.T) {
	encurta(t, 250*time.Millisecond)
	// Atende mas não diz nada (um QEMU travado).
	socket, _ := servidorQMP(t, "", func(string) []string { return nil })
	inicio := time.Now()
	if _, err := Estado(socket); err == nil {
		t.Errorf("mudo devia falhar")
	}
	if time.Since(inicio) > 3*time.Second {
		t.Errorf("demorou %v", time.Since(inicio))
	}
	// Saúda, aceita as capacidades e cala no comando.
	socket, _ = servidorQMP(t, saudacaoDoQemu, func(comando string) []string {
		if comando == "qmp_capabilities" {
			return []string{`{"return": {}}`}
		}
		return nil
	})
	if _, err := Estado(socket); err == nil || !strings.Contains(err.Error(), "não respondeu") {
		t.Errorf("calado no comando: %v", err)
	}
	// Quem atende não é um QEMU.
	socket, _ = servidorQMP(t, `{"oi": 1}`, normal(""))
	if _, err := Estado(socket); err == nil {
		t.Errorf("saudação errada devia falhar")
	}
	socket, _ = servidorQMP(t, "isto não é json", normal(""))
	if _, err := Estado(socket); err == nil {
		t.Errorf("lixo devia falhar")
	}
	if _, err := Estado(filepath.Join(pastaCurta(t), "nada.sock")); err == nil {
		t.Errorf("sem socket devia falhar")
	}
}

// --- /proc ----------------------------------------------------------------------

func procFalso(t *testing.T, pid string, arquivos map[string]string) string {
	t.Helper()
	proc := t.TempDir()
	_ = os.MkdirAll(filepath.Join(proc, pid), 0o755)
	for nome, texto := range arquivos {
		_ = os.WriteFile(filepath.Join(proc, pid, nome), []byte(texto), 0o644)
	}
	return proc
}

func TestLePid(t *testing.T) {
	dir := t.TempDir()
	for texto, quer := range map[string]int{"4242\n": 4242, " 77 ": 77, "": 0, "abc": 0, "0": 0, "1": 0, "-5": 0} {
		arquivo := filepath.Join(dir, "pid")
		_ = os.WriteFile(arquivo, []byte(texto), 0o600)
		if obtido := LePid(arquivo); obtido != quer {
			t.Errorf("LePid(%q) = %d", texto, obtido)
		}
	}
	if LePid(filepath.Join(dir, "nada")) != 0 {
		t.Errorf("sem arquivo")
	}
}

func TestEDaMaquina_SoOQemuDaquelaMaquina(t *testing.T) {
	cmd := func(args ...string) string { return strings.Join(args, "\x00") + "\x00" }
	proc := procFalso(t, "4242", map[string]string{"cmdline": cmd("qemu-system-x86_64", "-name", "oficina", "-m", "4096", "-pidfile", "/e/oficina/pid")})
	if !EDaMaquina(proc, 4242, "oficina", "/e/oficina/pid") {
		t.Errorf("é o QEMU da oficina")
	}
	if EDaMaquina(proc, 4242, "outra", "/e/outra/pid") || EDaMaquina(proc, 4242, "oficina", "/e/outra/pid") {
		t.Errorf("não é o de outra máquina")
	}
	if EDaMaquina(proc, 999, "oficina", "/e/oficina/pid") {
		t.Errorf("processo que não existe")
	}
	// O pid foi reusado por outro programa: nunca é da máquina.
	proc = procFalso(t, "4242", map[string]string{"cmdline": cmd("bash", "-c", "echo -name oficina -pidfile /e/oficina/pid")})
	if EDaMaquina(proc, 4242, "oficina", "/e/oficina/pid") {
		t.Errorf("um texto parecido num argumento só não conta")
	}
	proc = procFalso(t, "4242", map[string]string{"cmdline": ""})
	if EDaMaquina(proc, 4242, "oficina", "/e/oficina/pid") {
		t.Errorf("cmdline vazia (processo do núcleo)")
	}
}

func TestLeUso_DoStatEDoStatus(t *testing.T) {
	// O nome do programa com espaço e parêntese, para o campo não escorregar.
	stat := "4242 (qemu (sys) x86) S 1 4242 4242 0 -1 4194560 100 0 0 0 1500 700 0 0 20 0 5 0 12345 8000000000 524288 18446744073709551615 1 1 0 0 0 0 0 0 0 0 0 0 17 3 0 0 0 0 0\n"
	status := "Name:\tqemu\nVmPeak:\t 9000000 kB\nVmRSS:\t 2097152 kB\nThreads:\t5\n"
	proc := procFalso(t, "4242", map[string]string{"stat": stat, "status": status})
	u, ok := LeUso(proc, 4242)
	if !ok || u.Ticks != 2200 || u.RssKB != 2097152 {
		t.Errorf("uso: %+v %v", u, ok)
	}
	if _, ok := LeUso(proc, 1); ok {
		t.Errorf("processo que não existe")
	}
	proc = procFalso(t, "4242", map[string]string{"stat": "lixo"})
	if _, ok := LeUso(proc, 4242); ok {
		t.Errorf("stat estragado")
	}
}

func TestAbreParaEscrita_SemCriarNada(t *testing.T) {
	dir := t.TempDir()
	arquivo := filepath.Join(dir, "kvm")
	if AbreParaEscrita(arquivo) {
		t.Errorf("arquivo que não existe")
	}
	if _, err := os.Stat(arquivo); err == nil {
		t.Errorf("perguntar não cria o arquivo")
	}
	_ = os.WriteFile(arquivo, nil, 0o600)
	if !AbreParaEscrita(arquivo) {
		t.Errorf("devia abrir")
	}
	_ = os.Chmod(arquivo, 0o400)
	if os.Getuid() != 0 && AbreParaEscrita(arquivo) {
		t.Errorf("só leitura não serve")
	}
}
