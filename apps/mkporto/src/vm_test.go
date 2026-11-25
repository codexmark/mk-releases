package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"mkporto/internal/vm"
)

// O binário dos testes também faz as vezes de QEMU: quando o motor o roda com
// MKPORTO_QEMU_FALSO no ambiente, ele não testa nada, só finge ser a máquina
// (abre o socket do QMP, escreve o pid e responde). Nenhum QEMU de verdade
// roda nos testes.
func TestMain(m *testing.M) {
	if modo := os.Getenv("MKPORTO_QEMU_FALSO"); modo != "" {
		qemuFalso(modo, os.Args[1:])
		return
	}
	os.Exit(m.Run())
}

// qemuFalso lê o -qmp e o -pidfile da linha de comando e se comporta como o
// modo manda: "normal" (responde ao QMP; powerdown e quit o encerram),
// "morre" (reclama e sai, como o QEMU com um argumento errado) ou "mudo"
// (escreve o pid, abre o socket e nunca responde: um QEMU travado).
func qemuFalso(modo string, args []string) {
	var socket, pidfile string
	for i := 0; i+1 < len(args); i++ {
		switch args[i] {
		case "-qmp":
			socket = strings.TrimSuffix(strings.TrimPrefix(args[i+1], "unix:"), ",server=on,wait=off")
		case "-pidfile":
			pidfile = args[i+1]
		}
	}
	if modo == "morre" {
		fmt.Fprintln(os.Stderr, "qemu-system-x86_64: -cdrom /x.iso: Could not open '/x.iso': No such file or directory")
		os.Exit(1)
	}
	escuta, err := net.Listen("unix", socket)
	if err != nil {
		fmt.Fprintln(os.Stderr, "qemu falso:", err)
		os.Exit(1)
	}
	_ = os.WriteFile(pidfile, []byte(strconv.Itoa(os.Getpid())+"\n"), 0o600)
	pausada := false
	for {
		c, err := escuta.Accept()
		if err != nil {
			os.Exit(1)
		}
		if modo == "mudo" {
			continue // fica com a conexão aberta e calado
		}
		fmt.Fprintf(c, "%s\r\n", `{"QMP": {"version": {"qemu": {"micro": 0, "minor": 1, "major": 10}}, "capabilities": []}}`)
		leitor := bufio.NewScanner(c)
		for leitor.Scan() {
			var pedido struct {
				Execute string `json:"execute"`
			}
			_ = json.Unmarshal(leitor.Bytes(), &pedido)
			switch pedido.Execute {
			case "query-status":
				if pausada {
					fmt.Fprintf(c, "%s\r\n", `{"return": {"status": "paused", "running": false}}`)
				} else {
					fmt.Fprintf(c, "%s\r\n", `{"return": {"status": "running", "running": true}}`)
				}
			case "stop":
				pausada = true
				fmt.Fprintf(c, "%s\r\n%s\r\n", `{"event": "STOP", "timestamp": {"seconds": 1, "microseconds": 0}}`, `{"return": {}}`)
			case "cont":
				pausada = false
				fmt.Fprintf(c, "%s\r\n", `{"return": {}}`)
			case "system_powerdown", "quit":
				fmt.Fprintf(c, "%s\r\n", `{"return": {}}`)
				c.Close()
				_ = os.Remove(pidfile)
				os.Exit(0)
			default:
				fmt.Fprintf(c, "%s\r\n", `{"return": {}}`)
			}
		}
		c.Close()
	}
}

// pastaCurtaDeTeste: o caminho de um socket unix não passa de uns 100 caracteres.
func pastaCurtaDeTeste(t *testing.T) string {
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

const imgFalso = `case "$1 $2" in
  "create -f") : >"$4" ;;
  "snapshot -l") printf 'Snapshot list:\nID        TAG               VM SIZE                DATE     VM CLOCK     ICOUNT\n1         antes                 0 B 2026-11-24 10:00:00 00:00:00.000          0\n2         depois            1.2 GiB 2026-11-25 11:30:00 00:10:00.000          0\n' ;;
  "snapshot -c"|"snapshot -a"|"snapshot -d") ;;
  *) exit 9 ;;
esac`

// mundoDasMaquinas monta tudo de mentira: as pastas, o QEMU (este binário),
// o qemu-img (um script), o OVMF e o KVM (arquivos). Devolve a pasta das
// máquinas, a de execução e a dos comandos falsos.
func mundoDasMaquinas(t *testing.T) (raiz, execucao, bin string) {
	t.Helper()
	_, bin = mundo(t, map[string]string{"config": "Host meu\n  HostName 192.0.2.5\n"})
	casa := filepath.Dir(bin)
	raiz, execucao = filepath.Join(casa, "maquinas"), pastaCurtaDeTeste(t)
	eu, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ovmf := filepath.Join(casa, "ovmf")
	escreve(t, filepath.Join(ovmf, vm.OvmfCode), "código")
	escreve(t, filepath.Join(ovmf, vm.OvmfVars), "variáveis de fábrica")
	escreve(t, filepath.Join(casa, "kvm"), "")
	t.Setenv("MKPORTO_MAQUINAS", raiz)
	t.Setenv("MKPORTO_EXECUCAO", execucao)
	t.Setenv("XDG_RUNTIME_DIR", "")
	t.Setenv("MKPORTO_QEMU", eu)
	t.Setenv("MKPORTO_QEMU_FALSO", "normal")
	t.Setenv("MKPORTO_QEMU_IMG", falso(t, bin, "qemu-img", imgFalso))
	t.Setenv("MKPORTO_OVMF", ovmf)
	t.Setenv("MKPORTO_KVM", filepath.Join(casa, "kvm"))
	t.Setenv("MKPORTO_PROC", "")
	t.Setenv("MKPORTO_QEMU_DISPLAY", "none")
	// Nenhum QEMU de mentira sobrevive ao teste.
	t.Cleanup(func() {
		itens, _ := os.ReadDir(execucao)
		for _, item := range itens {
			pidfile := filepath.Join(execucao, item.Name(), "pid")
			if pid := vm.LePid(pidfile); pid != 0 && vm.EDaMaquina("/proc", pid, item.Name(), pidfile) {
				_ = syscall.Kill(pid, syscall.SIGKILL)
			}
		}
	})
	return raiz, execucao, bin
}

func maquinaNova(nome string) map[string]any {
	return map[string]any{"nome": nome, "memoria_mb": 2048, "cpus": 2, "disco_gb": 20, "iso": "", "uefi": false, "tres_d": false}
}

func maquinaDe(t *testing.T, nome string) map[string]any {
	t.Helper()
	for _, m := range quer(t, "", "vm", "listar")["maquinas"].([]any) {
		if m.(map[string]any)["nome"] == nome {
			return m.(map[string]any)
		}
	}
	t.Fatalf("sem a máquina %s", nome)
	return nil
}

// ateFicar espera a máquina chegar a um estado (a janela faz o mesmo, pelo listar).
func ateFicar(t *testing.T, nome, estado string) map[string]any {
	t.Helper()
	limite := time.Now().Add(5 * time.Second)
	for {
		m := maquinaDe(t, nome)
		if m["estado"] == estado {
			return m
		}
		if time.Now().After(limite) {
			t.Fatalf("a %s não ficou %s: %v", nome, estado, m)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func TestVm_SemSubcomandoOUsoEmJSON(t *testing.T) {
	mundoDasMaquinas(t)
	for _, args := range [][]string{{"vm"}, {"vm", "banana"}} {
		linha := querErro(t, 1, "uso: mkporto-motor vm <", "", args...)
		for _, sub := range []string{"requisitos", "listar", "criar", "ligar", "desligar", "pausar", "continuar", "uso", "snapshot", "remover", "host"} {
			if !strings.Contains(linha["erro"].(string), sub) {
				t.Errorf("o uso não cita %s", sub)
			}
		}
	}
}

func TestBandeiras(t *testing.T) {
	pos, ops, err := bandeiras([]string{"--forcar", "a", "--confirmo", "x", "--", "-b", "--forcar"}, []string{"--forcar"}, []string{"--confirmo"})
	if err != nil || !reflect.DeepEqual(pos, []string{"a", "-b", "--forcar"}) || ops["--forcar"] == "" || ops["--confirmo"] != "x" {
		t.Errorf("%v %v %v", pos, ops, err)
	}
	for _, ruins := range [][]string{{"--banana"}, {"-x"}, {"--confirmo"}, {"--forcar", "--forcar"}} {
		if _, _, err := bandeiras(ruins, []string{"--forcar"}, []string{"--confirmo"}); err == nil {
			t.Errorf("%v devia ser erro", ruins)
		}
	}
}

// --- requisitos -------------------------------------------------------------------

func TestVmRequisitos_Contrato(t *testing.T) {
	_, _, bin := mundoDasMaquinas(t)
	for _, nome := range []string{"foot", "sudo", "pacman"} {
		falso(t, bin, nome, "")
	}
	t.Setenv("MKPORTO_TERMINAL", "foot")
	r := quer(t, "", "vm", "requisitos")
	if !reflect.DeepEqual(campos(r), []string{"avisos", "instalar", "kvm", "ovmf", "pronto", "qemu", "qemu_img"}) {
		t.Fatalf("campos: %v", campos(r))
	}
	for _, c := range []string{"qemu", "qemu_img", "ovmf", "kvm"} {
		item := r[c].(map[string]any)
		if item["tem"] != true || item["caminho"] == "" {
			t.Errorf("%s: %v", c, item)
		}
	}
	if r["pronto"] != true || len(r["avisos"].([]any)) != 0 {
		t.Errorf("%v", r)
	}
	instalar := r["instalar"].(map[string]any)
	argv := []string{"sudo", "pacman", "-S", "--needed", "qemu-desktop", "edk2-ovmf"}
	if obtido := lista(instalar["argv"]); !reflect.DeepEqual(obtido, argv) {
		t.Errorf("argv: %q", obtido)
	}
	// Num terminal que espera no fim, para dar para ler o que o pacman disse.
	if term := lista(instalar["terminal"]); !reflect.DeepEqual(term, append([]string{"foot", "--title=instalar o QEMU", "--hold"}, argv...)) {
		t.Errorf("terminal: %q", term)
	}
	// O motor só disse o que abrir: nem o sudo nem o pacman rodaram.
	for _, nome := range []string{"foot", "sudo", "pacman"} {
		if chamadas(t, bin, nome) != nil {
			t.Errorf("o motor rodou o %s", nome)
		}
	}
}

func TestVmRequisitos_OQueFalta(t *testing.T) {
	mundoDasMaquinas(t)
	t.Setenv("MKPORTO_QEMU", "qemu-que-nao-existe")
	t.Setenv("MKPORTO_OVMF", "/nao/existe")
	t.Setenv("MKPORTO_KVM", "/nao/existe/kvm")
	t.Setenv("MKPORTO_EXECUCAO", "")
	r := quer(t, "", "vm", "requisitos")
	if r["pronto"] != false || r["qemu"].(map[string]any)["tem"] != false || r["ovmf"].(map[string]any)["tem"] != false ||
		r["kvm"].(map[string]any)["tem"] != false || r["qemu_img"].(map[string]any)["tem"] != true {
		t.Errorf("%v", r)
	}
	avisos := strings.Join(lista(r["avisos"]), " | ")
	for _, trecho := range []string{"qemu-desktop", "edk2-ovmf", "lentas", "XDG_RUNTIME_DIR"} {
		if !strings.Contains(avisos, trecho) {
			t.Errorf("falta o aviso de %s: %s", trecho, avisos)
		}
	}
	// Sem terminal: o argv continua lá, com o aviso.
	instalar := r["instalar"].(map[string]any)
	if instalar["terminal"] != nil || instalar["aviso"] == nil {
		t.Errorf("instalar: %v", instalar)
	}
	// Sem máquinas e sem QEMU, listar é uma lista vazia, não um erro.
	if m, ok := quer(t, "", "vm", "listar")["maquinas"].([]any); !ok || len(m) != 0 {
		t.Errorf("listar: %#v", m)
	}
}

// --- criar ------------------------------------------------------------------------

func TestVmCriar_Contrato(t *testing.T) {
	raiz, _, bin := mundoDasMaquinas(t)
	iso := filepath.Join(filepath.Dir(bin), "um sistema (novo).iso")
	escreve(t, iso, "iso")
	p := maquinaNova("oficina")
	p["iso"], p["uefi"], p["tres_d"] = iso, true, true
	r := quer(t, pedido(p), "vm", "criar")
	if !reflect.DeepEqual(campos(r), []string{"nome", "ok", "pasta", "porta_ssh"}) || r["nome"] != "oficina" || r["ok"] != true {
		t.Fatalf("%v", r)
	}
	porta := int(r["porta_ssh"].(float64))
	if porta < 2222 {
		t.Errorf("porta: %d", porta)
	}
	pasta := filepath.Join(raiz, "oficina")
	if info, _ := os.Stat(pasta); info.Mode().Perm() != 0o700 {
		t.Errorf("pasta: %o", info.Mode().Perm())
	}
	var gravada vm.Maquina
	if err := json.Unmarshal([]byte(conteudo(t, filepath.Join(pasta, "maquina.json"))), &gravada); err != nil {
		t.Fatal(err)
	}
	if gravada.Nome != "oficina" || gravada.MemoriaMB != 2048 || gravada.CPUs != 2 || gravada.DiscoGB != 20 || gravada.ISO != iso ||
		!gravada.UEFI || !gravada.TresD || gravada.PortaSSH != porta || gravada.CriadaEm < time.Now().Unix()-60 {
		t.Errorf("maquina.json: %+v", gravada)
	}
	if conteudo(t, filepath.Join(pasta, "vars.fd")) != "variáveis de fábrica" {
		t.Errorf("o vars.fd é a cópia do OVMF_VARS")
	}
	c := chamadas(t, bin, "qemu-img")
	if len(c) != 1 || !reflect.DeepEqual(c[0], []string{"create", "-f", "qcow2", filepath.Join(pasta, "disco.qcow2"), "20G"}) {
		t.Errorf("qemu-img: %q", c)
	}
	itens, _ := os.ReadDir(pasta)
	if len(itens) != 3 {
		t.Errorf("na pasta: %v", itens)
	}

	m := maquinaDe(t, "oficina")
	esperado := []string{"cpus", "criada_em", "desde", "disco_bytes", "disco_gb", "estado", "iso", "memoria_mb", "nome", "pid", "porta_ssh", "tres_d", "uefi"}
	if !reflect.DeepEqual(campos(m), esperado) {
		t.Errorf("campos do listar: %v", campos(m))
	}
	if m["estado"] != "desligada" || m["pid"] != nil || m["desde"] != nil || m["iso"] != iso || m["disco_bytes"] == nil {
		t.Errorf("listar: %v", m)
	}

	// A segunda máquina ganha outra porta, e uma porta ocupada por outro
	// programa é pulada.
	ocupada, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ocupada.Close()
	segunda := int(quer(t, pedido(maquinaNova("b")), "vm", "criar")["porta_ssh"].(float64))
	if segunda == porta || segunda < 2222 {
		t.Errorf("portas: %d e %d", porta, segunda)
	}
	if m := maquinaDe(t, "b"); m["iso"] != nil || m["uefi"] != false {
		t.Errorf("b: %v", m)
	}
}

func TestPortaLivre_PulaADeOutraMaquinaEAQueEstaEmUso(t *testing.T) {
	raiz, _, _ := mundoDasMaquinas(t)
	l := aLoja()
	primeira, ok := portaLivre(l)
	if !ok || primeira < vm.PrimeiraPorta {
		t.Fatalf("%d %v", primeira, ok)
	}
	// Outro programa escutando nela.
	escuta, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(primeira))
	if err != nil {
		t.Skip("a porta foi tomada no meio do teste")
	}
	segunda, _ := portaLivre(l)
	escuta.Close()
	if segunda <= primeira {
		t.Errorf("devia pular a %d: %d", primeira, segunda)
	}
	// Uma máquina (desligada) que já tem a porta.
	m := vm.Maquina{Nome: "dona", MemoriaMB: 512, CPUs: 1, DiscoGB: 1, PortaSSH: primeira}
	_ = l.CriaPasta("dona")
	_ = l.Grava(m)
	if terceira, _ := portaLivre(l); terceira == primeira {
		t.Errorf("devia pular a porta da outra máquina")
	}
	_ = raiz
}

func TestVmCriar_ValidacaoSemCriarNada(t *testing.T) {
	raiz, _, bin := mundoDasMaquinas(t)
	pasta := filepath.Dir(bin)
	escreve(t, filepath.Join(pasta, "boa.iso"), "iso")
	escreve(t, filepath.Join(pasta, "com,virgula.iso"), "iso")
	com := func(campo string, valor any) string {
		m := maquinaNova("oficina")
		m[campo] = valor
		return pedido(m)
	}
	casos := []struct{ campo, stdin string }{
		{"nome", com("nome", "")},
		{"nome", com("nome", "-name")},
		{"nome", com("nome", "Oficina")},
		{"nome", com("nome", "../fora")},
		{"nome", com("nome", "a b")},
		{"nome", com("nome", "a\nb")},
		{"nome", com("nome", strings.Repeat("a", 41))},
		{"memoria_mb", com("memoria_mb", 511)},
		{"memoria_mb", com("memoria_mb", 65537)},
		{"cpus", com("cpus", 0)},
		{"cpus", com("cpus", 33)},
		{"disco_gb", com("disco_gb", 0)},
		{"disco_gb", com("disco_gb", 2001)},
		{"iso", com("iso", "boa.iso")},
		{"iso", com("iso", filepath.Join(pasta, "com,virgula.iso"))},
		{"iso", com("iso", filepath.Join(pasta, "boa.iso")+",readonly=off")},
		{"iso", com("iso", filepath.Join(pasta, "boa.iso")+"\n-device x")},
		{"iso", com("iso", filepath.Join(pasta, "nao-existe.iso"))},
		{"iso", com("iso", pasta)},
	}
	for _, c := range casos {
		codigo, linhas := roda(t, c.stdin, "vm", "criar")
		if codigo != 1 || len(linhas) != 1 || linhas[0]["campo"] != c.campo || linhas[0]["erro"] == nil {
			t.Errorf("%s: código %d, %v", c.stdin, codigo, linhas)
		}
	}
	for _, torto := range []string{"", "banana", "[]", `{"nome":"a","memoria_mb":"2048"}`, com("comando", "rm"), pedido(maquinaNova("a")) + " {}"} {
		querErro(t, 1, "pedido", torto, "vm", "criar")
	}
	// UEFI sem o OVMF: recusa dizendo o pacote.
	t.Setenv("MKPORTO_OVMF", "/nao/existe")
	linha := querErro(t, 1, "edk2-ovmf", com("uefi", true), "vm", "criar")
	if linha["campo"] != "uefi" {
		t.Errorf("campo: %v", linha)
	}
	if itens, _ := os.ReadDir(raiz); len(itens) != 0 {
		t.Errorf("uma recusa não cria nada: %v", itens)
	}
	if chamadas(t, bin, "qemu-img") != nil {
		t.Errorf("uma recusa não roda o qemu-img")
	}
	// O nome que já existe.
	quer(t, pedido(maquinaNova("oficina")), "vm", "criar")
	linha = querErro(t, 1, "já existe", pedido(maquinaNova("oficina")), "vm", "criar")
	if linha["campo"] != "nome" {
		t.Errorf("campo: %v", linha)
	}
}

func TestVmCriar_SeAlgoFalhaNoMeioAPastaEApagada(t *testing.T) {
	raiz, _, bin := mundoDasMaquinas(t)
	t.Setenv("MKPORTO_QEMU_IMG", falso(t, bin, "qemu-img", "printf 'qemu-img: disco.qcow2: Could not create: No space left on device\\n' >&2; exit 1"))
	querErro(t, 1, "No space left on device", pedido(maquinaNova("oficina")), "vm", "criar")
	if _, err := os.Stat(filepath.Join(raiz, "oficina")); err == nil {
		t.Errorf("a pasta da máquina que não nasceu ficou")
	}
	// O qemu-img criou o disco, mas o vars do UEFI não pôde ser copiado.
	t.Setenv("MKPORTO_QEMU_IMG", falso(t, bin, "qemu-img", imgFalso))
	if err := os.Chmod(filepath.Join(os.Getenv("MKPORTO_OVMF"), vm.OvmfVars), 0); err != nil {
		t.Fatal(err)
	}
	p := maquinaNova("oficina")
	p["uefi"] = true
	if os.Getuid() != 0 {
		querErro(t, 1, "OVMF_VARS", pedido(p), "vm", "criar")
		if _, err := os.Stat(filepath.Join(raiz, "oficina")); err == nil {
			t.Errorf("a pasta ficou depois da falha do vars")
		}
	}
	// Sem o qemu-img: erro dizendo o pacote, e nada criado.
	t.Setenv("MKPORTO_QEMU_IMG", "qemu-img-que-nao-existe")
	querErro(t, 1, "qemu-desktop", pedido(maquinaNova("oficina")), "vm", "criar")
	if itens, _ := os.ReadDir(raiz); len(itens) != 0 {
		t.Errorf("sobrou: %v", itens)
	}
}

// --- ligar, pausar, uso, desligar ---------------------------------------------------

func TestVmLigar_Contrato(t *testing.T) {
	raiz, execucao, bin := mundoDasMaquinas(t)
	quer(t, pedido(maquinaNova("oficina")), "vm", "criar")
	r := quer(t, "", "vm", "ligar", "--", "oficina")
	if !reflect.DeepEqual(campos(r), []string{"argv", "avisos", "estado", "nome", "ok", "pid", "porta_ssh"}) {
		t.Fatalf("campos: %v", campos(r))
	}
	pid := int(r["pid"].(float64))
	if r["estado"] != "ligada" || r["ok"] != true || pid <= 1 || len(r["avisos"].([]any)) != 0 {
		t.Errorf("%v", r)
	}
	// A linha de comando que o motor diz ter rodado é a que o processo tem.
	argv := lista(r["argv"])
	if !reflect.DeepEqual(vm.Cmdline("/proc", pid), argv) {
		t.Errorf("argv dito:\n%q\nargv do processo:\n%q", argv, vm.Cmdline("/proc", pid))
	}
	juntos := strings.Join(argv, " ")
	for _, trecho := range []string{"-name oficina", "-machine q35", "-enable-kvm -cpu host", "-smp 2", "-m 2048",
		"-drive file=" + filepath.Join(raiz, "oficina", "disco.qcow2") + ",if=virtio,format=qcow2",
		"-qmp unix:" + filepath.Join(execucao, "oficina", "qmp.sock") + ",server=on,wait=off",
		"-pidfile " + filepath.Join(execucao, "oficina", "pid"), "-display none"} {
		if !strings.Contains(juntos, trecho) {
			t.Errorf("falta %q em %s", trecho, juntos)
		}
	}
	// Num grupo de processos próprio: fechar a janela do MkPorto não a leva junto.
	if sid, err := syscall.Getpgid(pid); err != nil || sid != pid {
		t.Errorf("grupo do QEMU: %d %v (devia ser o dele, %d)", sid, err, pid)
	}

	m := maquinaDe(t, "oficina")
	if m["estado"] != "ligada" || m["pid"] != float64(pid) || m["desde"] == nil {
		t.Errorf("listar: %v", m)
	}
	// Ligada, não liga de novo, não aceita snapshot nem remoção.
	querErro(t, 2, "já está ligada", "", "vm", "ligar", "oficina")
	querErro(t, 2, "desligada", "", "vm", "snapshot", "listar", "oficina")
	querErro(t, 2, "desligada", "", "vm", "snapshot", "criar", "oficina", "antes")
	querErro(t, 2, "desligue", "", "vm", "remover", "oficina", "--confirmo", "oficina")
	if c := chamadas(t, bin, "qemu-img"); len(c) != 1 {
		t.Errorf("o qemu-img não toca no disco de uma máquina ligada: %q", c)
	}

	uso := quer(t, "", "vm", "uso", "oficina")
	if !reflect.DeepEqual(campos(uso), []string{"agora_ms", "hz", "nome", "pid", "rss_kb", "ticks"}) {
		t.Fatalf("uso: %v", uso)
	}
	if uso["hz"] != 100.0 || uso["pid"] != float64(pid) || uso["rss_kb"].(float64) <= 0 || uso["ticks"].(float64) < 0 ||
		uso["agora_ms"].(float64) < float64(time.Now().UnixMilli()-60000) {
		t.Errorf("uso: %v", uso)
	}

	if r := quer(t, "", "vm", "pausar", "oficina"); r["estado"] != "pausada" {
		t.Errorf("pausar: %v", r)
	}
	if m := maquinaDe(t, "oficina"); m["estado"] != "pausada" {
		t.Errorf("o listar pergunta ao QMP: %v", m)
	}
	querErro(t, 2, "já está pausada", "", "vm", "ligar", "oficina")
	if r := quer(t, "", "vm", "continuar", "oficina"); r["estado"] != "ligada" {
		t.Errorf("continuar: %v", r)
	}
	if m := maquinaDe(t, "oficina"); m["estado"] != "ligada" {
		t.Errorf("%v", m)
	}

	// Desligar sem forçar: o pedido vai e o motor volta logo.
	d := quer(t, "", "vm", "desligar", "oficina")
	if !reflect.DeepEqual(d, map[string]any{"ok": true, "nome": "oficina", "pedido": true, "forcado": false}) {
		t.Errorf("desligar: %v", d)
	}
	if m := ateFicar(t, "oficina", "desligada"); m["pid"] != nil || m["desde"] != nil {
		t.Errorf("desligada: %v", m)
	}
	for _, acao := range []string{"desligar", "pausar", "continuar", "uso"} {
		querErro(t, 1, "desligada", "", "vm", acao, "oficina")
	}
	// E liga de novo, e o --forcar a tira do ar pelo QMP.
	quer(t, "", "vm", "ligar", "oficina")
	f := quer(t, "", "vm", "desligar", "--forcar", "oficina")
	if f["forcado"] != true {
		t.Errorf("forçar: %v", f)
	}
	if m := maquinaDe(t, "oficina"); m["estado"] != "desligada" {
		t.Errorf("depois de forçar: %v", m)
	}
}

func TestVmLigar_OQueMudaALinhaDeComando(t *testing.T) {
	_, _, bin := mundoDasMaquinas(t)
	iso := filepath.Join(filepath.Dir(bin), "sistema.iso")
	escreve(t, iso, "iso")
	p := maquinaNova("oficina")
	p["iso"], p["uefi"] = iso, true
	quer(t, pedido(p), "vm", "criar")

	// Sem acesso ao KVM: liga emulada, e avisa.
	t.Setenv("MKPORTO_KVM", "/nao/existe/kvm")
	r := quer(t, "", "vm", "ligar", "oficina")
	juntos := strings.Join(lista(r["argv"]), " ")
	if strings.Contains(juntos, "-enable-kvm") || !strings.Contains(juntos, "-cpu max") || len(r["avisos"].([]any)) != 1 {
		t.Errorf("sem kvm: %s %v", juntos, r["avisos"])
	}
	if !strings.Contains(juntos, "-cdrom "+iso+" -boot order=cd,menu=on") || !strings.Contains(juntos, "if=pflash,format=raw,readonly=on,file=") {
		t.Errorf("com ISO e UEFI: %s", juntos)
	}
	quer(t, "", "vm", "desligar", "--forcar", "oficina")

	// A ISO sumiu: erro que diz o que fazer; com --sem-iso, liga.
	if err := os.Remove(iso); err != nil {
		t.Fatal(err)
	}
	querErro(t, 1, "ligue sem ela", "", "vm", "ligar", "oficina")
	r = quer(t, "", "vm", "ligar", "--sem-iso", "oficina")
	if juntos := strings.Join(lista(r["argv"]), " "); strings.Contains(juntos, "-cdrom") {
		t.Errorf("--sem-iso: %s", juntos)
	}
	quer(t, "", "vm", "desligar", "--forcar", "--", "oficina")

	// Recusas e erros de antes de rodar qualquer coisa.
	t.Setenv("MKPORTO_QEMU_DISPLAY", "sdl,gl=on")
	querErro(t, 1, "MKPORTO_QEMU_DISPLAY", "", "vm", "ligar", "--sem-iso", "oficina")
	t.Setenv("MKPORTO_QEMU_DISPLAY", "none")
	t.Setenv("MKPORTO_OVMF", "/nao/existe")
	querErro(t, 1, "edk2-ovmf", "", "vm", "ligar", "--sem-iso", "oficina")
	t.Setenv("MKPORTO_QEMU", "qemu-que-nao-existe")
	querErro(t, 1, "qemu-desktop", "", "vm", "ligar", "--sem-iso", "oficina")
	t.Setenv("MKPORTO_EXECUCAO", "")
	querErro(t, 2, "XDG_RUNTIME_DIR", "", "vm", "ligar", "--sem-iso", "oficina")
	querErro(t, 1, "fantasma", "", "vm", "ligar", "fantasma")
	querErro(t, 1, "uso:", "", "vm", "ligar")
	querErro(t, 1, "não conheço a opção", "", "vm", "ligar", "-enable-kvm")
	querErro(t, 1, "não é nome de máquina", "", "vm", "ligar", "--", "-enable-kvm")
}

func TestVmLigar_QemuQueMorreDevolveOFimDoLog(t *testing.T) {
	_, execucao, _ := mundoDasMaquinas(t)
	quer(t, pedido(maquinaNova("oficina")), "vm", "criar")
	t.Setenv("MKPORTO_QEMU_FALSO", "morre")
	linha := querErro(t, 1, "o QEMU não subiu", "", "vm", "ligar", "oficina")
	if !strings.Contains(linha["erro"].(string), "Could not open '/x.iso'") {
		t.Errorf("o erro tem de trazer o que o QEMU reclamou: %v", linha["erro"])
	}
	if m := maquinaDe(t, "oficina"); m["estado"] != "desligada" {
		t.Errorf("%v", m)
	}
	if _, err := os.Stat(filepath.Join(execucao, "oficina", "pid")); err == nil {
		t.Errorf("sobrou o pid")
	}
}

func TestVmDesligar_ComOQmpMudoForcarMandaSigtermSoAoQemuDaMaquina(t *testing.T) {
	_, execucao, _ := mundoDasMaquinas(t)
	quer(t, pedido(maquinaNova("oficina")), "vm", "criar")
	t.Setenv("MKPORTO_QEMU_FALSO", "mudo")
	encurta(t, &prazoDeLigar, 600*time.Millisecond)
	encurta(t, &vm.PrazoDoQMP, 150*time.Millisecond)
	// O QEMU sobe mas não atende: o ligar desiste de esperar e diz onde olhar.
	querErro(t, 1, "não atendeu no QMP", "", "vm", "ligar", "oficina")
	m := maquinaDe(t, "oficina")
	if m["estado"] != "travada" || m["pid"] == nil {
		t.Fatalf("travada: %v", m)
	}
	pid := int(m["pid"].(float64))
	querErro(t, 2, "já está travada", "", "vm", "ligar", "oficina")
	// Sem forçar não há como pedir: erro que diz que dá para forçar.
	querErro(t, 1, "forçar", "", "vm", "desligar", "oficina")
	if !vm.EDaMaquina("/proc", pid, "oficina", filepath.Join(execucao, "oficina", "pid")) {
		t.Fatalf("o processo devia continuar vivo")
	}
	f := quer(t, "", "vm", "desligar", "--forcar", "oficina")
	if f["forcado"] != true {
		t.Errorf("%v", f)
	}
	if vm.EDaMaquina("/proc", pid, "oficina", filepath.Join(execucao, "oficina", "pid")) {
		t.Errorf("o QEMU travado continua vivo")
	}
	if m := maquinaDe(t, "oficina"); m["estado"] != "desligada" {
		t.Errorf("%v", m)
	}
}

func TestSituacao_PidQueSobrouOuQueEDeOutroProgramaELimpoSemSinalizarNinguem(t *testing.T) {
	_, execucao, _ := mundoDasMaquinas(t)
	quer(t, pedido(maquinaNova("oficina")), "vm", "criar")
	pidfile := filepath.Join(execucao, "oficina", "pid")
	// O pid deste teste: um processo vivo, que não é o QEMU da máquina.
	for _, texto := range []string{strconv.Itoa(os.Getpid()), "999999999", "lixo"} {
		escreve(t, pidfile, texto+"\n")
		escreve(t, filepath.Join(execucao, "oficina", "qmp.sock"), "")
		if m := maquinaDe(t, "oficina"); m["estado"] != "desligada" || m["pid"] != nil {
			t.Errorf("pid %q: %v", texto, m)
		}
		// Forçar o desligamento de uma máquina assim não manda sinal a ninguém.
		querErro(t, 1, "já está desligada", "", "vm", "desligar", "--forcar", "oficina")
	}
	if _, err := os.Stat(pidfile); err == nil && vm.LePid(pidfile) != 0 {
		t.Errorf("o pid que sobrou devia ter sido limpo")
	}
	// Com um /proc de mentira em que o pid é de outro programa.
	proc := t.TempDir()
	escreve(t, filepath.Join(proc, "4242", "cmdline"), "bash\x00-c\x00sleep 1\x00")
	t.Setenv("MKPORTO_PROC", proc)
	escreve(t, pidfile, "4242\n")
	if m := maquinaDe(t, "oficina"); m["estado"] != "desligada" {
		t.Errorf("%v", m)
	}
}

// --- snapshots --------------------------------------------------------------------

func TestVmSnapshot_Contrato(t *testing.T) {
	raiz, _, bin := mundoDasMaquinas(t)
	quer(t, pedido(maquinaNova("oficina")), "vm", "criar")
	disco := filepath.Join(raiz, "oficina", "disco.qcow2")
	r := quer(t, "", "vm", "snapshot", "listar", "--", "oficina")
	esperado := []any{
		map[string]any{"id": "1", "rotulo": "antes", "data": "2026-11-24 10:00:00", "tamanho": "0 B"},
		map[string]any{"id": "2", "rotulo": "depois", "data": "2026-11-25 11:30:00", "tamanho": "1.2 GiB"},
	}
	if r["nome"] != "oficina" || !reflect.DeepEqual(r["snapshots"], esperado) {
		t.Errorf("listar: %v", r)
	}
	for acao, opcao := range map[string]string{"criar": "-c", "restaurar": "-a", "apagar": "-d"} {
		r := quer(t, "", "vm", "snapshot", acao, "--", "oficina", "antes-de-atualizar_2.0")
		if !reflect.DeepEqual(r, map[string]any{"ok": true, "nome": "oficina", "rotulo": "antes-de-atualizar_2.0"}) {
			t.Errorf("%s: %v", acao, r)
		}
		c := chamadas(t, bin, "qemu-img")
		if ultima := c[len(c)-1]; !reflect.DeepEqual(ultima, []string{"snapshot", opcao, "antes-de-atualizar_2.0", disco}) {
			t.Errorf("%s: %q", acao, ultima)
		}
	}
	antes := len(chamadas(t, bin, "qemu-img"))
	for _, rotulo := range []string{"", "-a", "a b", "a/b", "ação", "a\nb", strings.Repeat("a", 41)} {
		codigo, linhas := roda(t, "", "vm", "snapshot", "criar", "--", "oficina", rotulo)
		if codigo != 1 || linhas[0]["campo"] != "rotulo" {
			t.Errorf("rótulo %q: %d %v", rotulo, codigo, linhas)
		}
	}
	querErro(t, 1, "uso:", "", "vm", "snapshot")
	querErro(t, 1, "uso:", "", "vm", "snapshot", "listar")
	querErro(t, 1, "uso:", "", "vm", "snapshot", "listar", "oficina", "sobra")
	querErro(t, 1, "uso:", "", "vm", "snapshot", "criar", "oficina")
	querErro(t, 1, "uso:", "", "vm", "snapshot", "converter", "oficina", "x")
	querErro(t, 1, "fantasma", "", "vm", "snapshot", "listar", "fantasma")
	if len(chamadas(t, bin, "qemu-img")) != antes {
		t.Errorf("uma recusa não roda o qemu-img")
	}
	// O que o qemu-img reclamar chega à pessoa.
	t.Setenv("MKPORTO_QEMU_IMG", falso(t, bin, "qemu-img", "printf \"qemu-img: Could not find snapshot 'x'\\n\" >&2; exit 1"))
	querErro(t, 1, "Could not find snapshot", "", "vm", "snapshot", "restaurar", "oficina", "x")
}

// --- remover e host ---------------------------------------------------------------

func TestVmRemover_SoComONomeRepetidoESoAPastaDela(t *testing.T) {
	raiz, execucao, _ := mundoDasMaquinas(t)
	for _, nome := range []string{"oficina", "fica"} {
		quer(t, pedido(maquinaNova(nome)), "vm", "criar")
	}
	escreve(t, filepath.Join(execucao, "oficina", "qemu.log"), "velho")
	escreve(t, filepath.Join(execucao, "fica", "qemu.log"), "velho")
	querErro(t, 2, "repita o nome", "", "vm", "remover", "oficina")
	querErro(t, 2, "repita o nome", "", "vm", "remover", "oficina", "--confirmo", "fica")
	querErro(t, 2, "repita o nome", "", "vm", "remover", "oficina", "--confirmo", "")
	querErro(t, 1, "fantasma", "", "vm", "remover", "fantasma", "--confirmo", "fantasma")
	querErro(t, 1, "não é nome de máquina", "", "vm", "remover", "--confirmo", "..", "--", "..")
	querErro(t, 1, "uso:", "", "vm", "remover", "--confirmo", "oficina")
	if _, err := os.Stat(filepath.Join(raiz, "oficina", "maquina.json")); err != nil {
		t.Fatalf("uma recusa não apaga nada")
	}
	r := quer(t, "", "vm", "remover", "--confirmo", "oficina", "--", "oficina")
	if !reflect.DeepEqual(r, map[string]any{"ok": true, "nome": "oficina"}) {
		t.Errorf("%v", r)
	}
	for _, sumiu := range []string{filepath.Join(raiz, "oficina"), filepath.Join(execucao, "oficina")} {
		if _, err := os.Stat(sumiu); err == nil {
			t.Errorf("ficou: %s", sumiu)
		}
	}
	for _, ficou := range []string{filepath.Join(raiz, "fica", "maquina.json"), filepath.Join(execucao, "fica", "qemu.log")} {
		if _, err := os.Stat(ficou); err != nil {
			t.Errorf("sumiu o que era de outra máquina: %s", ficou)
		}
	}
}

func TestVmHost_Contrato(t *testing.T) {
	_, _, _ = mundoDasMaquinas(t)
	porta := quer(t, pedido(maquinaNova("oficina")), "vm", "criar")["porta_ssh"].(float64)
	r := quer(t, "", "vm", "host", "--", "oficina")
	if r["ok"] != true || r["apelido"] != "oficina" || r["include"] != "posto" {
		t.Errorf("%v", r)
	}
	h := hostDe(t, quer(t, "", "listar"), "oficina")
	if h["origem"] != "mkporto" || h["host"] != "127.0.0.1" || h["porta"] != porta || h["usuario"] != nil {
		t.Errorf("host: %v", h)
	}
	// A pessoa pôs o usuário e a chave pelo formulário: atualizar a porta não os apaga.
	quer(t, pedido(map[string]any{"apelido": "oficina", "host": "127.0.0.1", "porta": 1, "usuario": "ana", "chave": "~/.ssh/k"}), "guardar")
	quer(t, "", "vm", "host", "oficina")
	h = hostDe(t, quer(t, "", "listar"), "oficina")
	if h["porta"] != porta || h["usuario"] != "ana" || h["chave"] != "~/.ssh/k" {
		t.Errorf("atualizado: %v", h)
	}
	// Um apelido que já é do config da pessoa não é tomado.
	quer(t, pedido(maquinaNova("meu")), "vm", "criar")
	querErro(t, 2, "~/.ssh/config", "", "vm", "host", "meu")
	querErro(t, 1, "fantasma", "", "vm", "host", "fantasma")
}
