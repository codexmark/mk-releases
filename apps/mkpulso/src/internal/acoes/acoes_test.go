package acoes

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"testing"
)

// processo é uma linha do /proc falso.
type processo struct {
	pid, ppid int
	nome      string
	uid       int // -1: o de quem roda o teste
}

// procFalso monta um /proc só com os processos dados e o põe em MKPULSO_PROC.
func procFalso(t *testing.T, ps ...processo) {
	t.Helper()
	raiz := t.TempDir()
	for _, p := range ps {
		uid := p.uid
		if uid < 0 {
			uid = os.Getuid()
		}
		dir := filepath.Join(raiz, strconv.Itoa(p.pid))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		status := fmt.Sprintf("Name:\t%s\nUmask:\t0022\nState:\tS (sleeping)\nTgid:\t%d\nPid:\t%d\nPPid:\t%d\nTracerPid:\t0\nUid:\t%d\t%d\t%d\t%d\nGid:\t1000\t1000\t1000\t1000\n",
			p.nome, p.pid, p.pid, p.ppid, uid, uid, uid, uid)
		if err := os.WriteFile(filepath.Join(dir, "status"), []byte(status), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("MKPULSO_PROC", raiz)
}

// envio é um sinal que o código pediu (e que o teste não deixou sair).
type envio struct {
	pid int
	sig syscall.Signal
}

// semSinais troca a função que manda sinal por uma que só anota.
func semSinais(t *testing.T, erro error) *[]envio {
	t.Helper()
	var enviados []envio
	antes := Mata
	Mata = func(pid int, sig syscall.Signal) error {
		enviados = append(enviados, envio{pid, sig})
		return erro
	}
	t.Cleanup(func() { Mata = antes })
	return &enviados
}

// nice é uma mudança de prioridade que o código pediu.
type nice struct{ pid, valor int }

func semPrioridade(t *testing.T, erro error) *[]nice {
	t.Helper()
	var pedidos []nice
	antes := MudaNice
	MudaNice = func(pid, valor int) error {
		pedidos = append(pedidos, nice{pid, valor})
		return erro
	}
	t.Cleanup(func() { MudaNice = antes })
	return &pedidos
}

func TestSinal_CadaNomeDoContratoViraOSinalCerto(t *testing.T) {
	casos := map[string]syscall.Signal{
		"encerrar":    syscall.SIGTERM,
		"matar":       syscall.SIGKILL,
		"pausar":      syscall.SIGSTOP,
		"continuar":   syscall.SIGCONT,
		"recarregar":  syscall.SIGHUP,
		"interromper": syscall.SIGINT,
	}
	for nome, sig := range casos {
		procFalso(t, processo{4321, 900, "firefox", -1})
		enviados := semSinais(t, nil)
		if err := Sinal(4321, nome, false); err != nil {
			t.Fatalf("%s: %v", nome, err)
		}
		if !reflect.DeepEqual(*enviados, []envio{{4321, sig}}) {
			t.Errorf("%s: mandou %v, queria %v ao 4321", nome, *enviados, sig)
		}
	}
	if len(Sinais()) != len(casos) {
		t.Errorf("Sinais() tem %d nomes, o contrato tem %d", len(Sinais()), len(casos))
	}
}

func TestSinal_NomeDesconhecidoEErroDeUsoENaoMandaNada(t *testing.T) {
	procFalso(t, processo{4321, 900, "firefox", -1})
	enviados := semSinais(t, nil)
	for _, nome := range []string{"", "KILL", "9", "SIGTERM", "explodir"} {
		err := Sinal(4321, nome, false)
		if err == nil || EhRecusa(err) {
			t.Errorf("sinal %q: queria erro de uso, veio %v", nome, err)
		}
	}
	if len(*enviados) != 0 {
		t.Errorf("mandou %v", *enviados)
	}
}

func TestSinal_RecusasDeSeguranca(t *testing.T) {
	outro := os.Getuid() + 1
	procFalso(t,
		processo{1, 0, "systemd", 0},
		processo{os.Getpid(), os.Getppid(), "mkpulso-coletor", -1},
		processo{os.Getppid(), 1, "bash", -1},
		processo{500, 1, "sshd", outro},
		processo{600, 1, "Hyprland", -1},
		processo{601, 600, "quickshell", -1},
		processo{602, 600, "qs", -1},
		processo{603, 1, "sddm", -1},
		processo{604, 1, "systemd", -1},
		processo{605, 1, "sddm-helper", -1},
	)
	casos := []struct {
		nome  string
		pid   int
		trava string // um trecho da frase
	}{
		{"o pid 1", 1, "processo 1"},
		{"o pid 0", 0, "não é um processo"},
		{"um pid negativo (seria um grupo inteiro)", -1, "não é um processo"},
		{"um pid negativo qualquer", -4321, "não é um processo"},
		{"o próprio coletor", os.Getpid(), "próprio coletor"},
		{"o pai do coletor", os.Getppid(), "quem chamou"},
		{"processo de outro usuário", 500, "outro usuário"},
		{"o Hyprland", 600, "sessão"},
		{"o quickshell do Maré", 601, "sessão"},
		{"o qs do Maré", 602, "sessão"},
		{"o sddm", 603, "sessão"},
		{"o systemd do usuário", 604, "sessão"},
		{"o ajudante do sddm", 605, "sessão"},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			enviados := semSinais(t, nil)
			pedidos := semPrioridade(t, nil)
			for _, err := range []error{Sinal(c.pid, "encerrar", false), Prioridade(c.pid, 5, false)} {
				if !EhRecusa(err) {
					t.Fatalf("queria recusa, veio %v", err)
				}
				if !strings.Contains(err.Error(), c.trava) {
					t.Errorf("a frase %q não fala de %q", err.Error(), c.trava)
				}
				if Codigo(err) != 2 {
					t.Errorf("código %d, queria 2", Codigo(err))
				}
			}
			if len(*enviados) != 0 || len(*pedidos) != 0 {
				t.Errorf("chegou a agir: %v %v", *enviados, *pedidos)
			}
		})
	}
}

func TestSinal_MesmoAssimSoPassaASessao(t *testing.T) {
	outro := os.Getuid() + 1
	procFalso(t,
		processo{1, 0, "systemd", 0},
		processo{os.Getpid(), os.Getppid(), "mkpulso-coletor", -1},
		processo{os.Getppid(), 1, "qs", -1},
		processo{500, 1, "sshd", outro},
		processo{600, 1, "Hyprland", -1},
	)
	enviados := semSinais(t, nil)
	if err := Sinal(600, "recarregar", true); err != nil {
		t.Fatalf("a sessão com --mesmo-assim: %v", err)
	}
	if !reflect.DeepEqual(*enviados, []envio{{600, syscall.SIGHUP}}) {
		t.Errorf("mandou %v", *enviados)
	}
	for _, pid := range []int{1, 0, -1, os.Getpid(), os.Getppid(), 500} {
		if err := Sinal(pid, "encerrar", true); !EhRecusa(err) {
			t.Errorf("pid %d com --mesmo-assim: queria recusa, veio %v", pid, err)
		}
	}
	if len(*enviados) != 1 {
		t.Errorf("mandou %v", *enviados)
	}
}

func TestSinal_ProcessoQueJaSaiuEErroComum(t *testing.T) {
	procFalso(t)
	enviados := semSinais(t, nil)
	err := Sinal(4321, "encerrar", false)
	if err == nil || EhRecusa(err) || !strings.Contains(err.Error(), "já saiu") {
		t.Fatalf("queria o erro de que já saiu, veio %v", err)
	}
	if Codigo(err) != 1 {
		t.Errorf("código %d, queria 1", Codigo(err))
	}
	if len(*enviados) != 0 {
		t.Errorf("mandou %v", *enviados)
	}
	// Some entre a conferência e o sinal: o mesmo erro.
	procFalso(t, processo{4321, 900, "firefox", -1})
	semSinais(t, syscall.ESRCH)
	err = Sinal(4321, "encerrar", false)
	if err == nil || !strings.Contains(err.Error(), "já saiu") {
		t.Errorf("ESRCH: veio %v", err)
	}
	semSinais(t, syscall.EPERM)
	err = Sinal(4321, "encerrar", false)
	if err == nil || EhRecusa(err) || !strings.Contains(err.Error(), "não deixou") {
		t.Errorf("EPERM: veio %v", err)
	}
}

func TestSinal_StatusIlegivelERecusa(t *testing.T) {
	raiz := t.TempDir()
	if err := os.MkdirAll(filepath.Join(raiz, "4321"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(raiz, "4321", "status"), []byte("Name:\tfoo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MKPULSO_PROC", raiz)
	enviados := semSinais(t, nil)
	if err := Sinal(4321, "encerrar", false); !EhRecusa(err) {
		t.Errorf("sem saber o dono, queria recusa; veio %v", err)
	}
	if len(*enviados) != 0 {
		t.Errorf("mandou %v", *enviados)
	}
}

func TestSinal_DePontaAPontaNumFilhoDoTeste(t *testing.T) {
	t.Setenv("MKPULSO_PROC", "")
	filho := exec.Command("sleep", "30")
	if err := filho.Start(); err != nil {
		t.Skipf("sem sleep para servir de filho: %v", err)
	}
	if err := Sinal(filho.Process.Pid, "encerrar", false); err != nil {
		_ = filho.Process.Kill()
		t.Fatal(err)
	}
	err := filho.Wait()
	var saiu *exec.ExitError
	if !errors.As(err, &saiu) {
		t.Fatalf("o filho não morreu de sinal: %v", err)
	}
	if st := saiu.Sys().(syscall.WaitStatus); !st.Signaled() || st.Signal() != syscall.SIGTERM {
		t.Errorf("o filho saiu com %v, queria SIGTERM", st)
	}
	// Depois de colhido, o pid não existe mais.
	if err := Sinal(filho.Process.Pid, "encerrar", false); err == nil || !strings.Contains(err.Error(), "já saiu") {
		t.Errorf("depois de morto: veio %v", err)
	}
}

func TestPrioridade_MandaONiceAoProcesso(t *testing.T) {
	procFalso(t, processo{4321, 900, "firefox", -1})
	pedidos := semPrioridade(t, nil)
	if err := Prioridade(4321, 10, false); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(*pedidos, []nice{{4321, 10}}) {
		t.Errorf("pediu %v", *pedidos)
	}
}

func TestPrioridade_VaiEmTodasAsThreads(t *testing.T) {
	procFalso(t, processo{4321, 900, "firefox", -1})
	for _, tid := range []string{"4321", "4330", "4331"} {
		if err := os.MkdirAll(filepath.Join(os.Getenv("MKPULSO_PROC"), "4321", "task", tid), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	pedidos := semPrioridade(t, nil)
	if err := Prioridade(4321, 3, false); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(*pedidos, []nice{{4321, 3}, {4330, 3}, {4331, 3}}) {
		t.Errorf("pediu %v", *pedidos)
	}
}

func TestPrioridade_ForaDaFaixaEErroDeUso(t *testing.T) {
	procFalso(t, processo{4321, 900, "firefox", -1})
	pedidos := semPrioridade(t, nil)
	for _, n := range []int{-21, 20, 99, -1000} {
		err := Prioridade(4321, n, false)
		if err == nil || EhRecusa(err) || !strings.Contains(err.Error(), "-20 a 19") {
			t.Errorf("nice %d: veio %v", n, err)
		}
	}
	for _, n := range []int{-20, 0, 19} {
		if err := Prioridade(4321, n, false); err != nil {
			t.Errorf("nice %d: %v", n, err)
		}
	}
	if len(*pedidos) != 3 {
		t.Errorf("pediu %v", *pedidos)
	}
}

func TestPrioridade_SistemaRecusaSubirDaAFraseDoContrato(t *testing.T) {
	procFalso(t, processo{4321, 900, "firefox", -1})
	semPrioridade(t, syscall.EACCES)
	err := Prioridade(4321, -5, false)
	if err == nil || err.Error() != "só dá para baixar a prioridade; para subir é preciso ser administrador" {
		t.Fatalf("veio %v", err)
	}
	if Codigo(err) != 1 {
		t.Errorf("código %d, queria 1", Codigo(err))
	}
	semPrioridade(t, syscall.ESRCH)
	if err := Prioridade(4321, 5, false); err == nil || !strings.Contains(err.Error(), "já saiu") {
		t.Errorf("ESRCH: veio %v", err)
	}
}

func TestPrioridade_DePontaAPontaNumFilhoDoTeste(t *testing.T) {
	t.Setenv("MKPULSO_PROC", "")
	filho := exec.Command("sleep", "30")
	if err := filho.Start(); err != nil {
		t.Skipf("sem sleep para servir de filho: %v", err)
	}
	defer func() {
		_ = filho.Process.Kill()
		_ = filho.Wait()
	}()
	if err := Prioridade(filho.Process.Pid, 7, false); err != nil {
		t.Fatal(err)
	}
	stat, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", filho.Process.Pid))
	if err != nil {
		t.Fatal(err)
	}
	// Depois do ")" do nome, o nice é o 17º campo.
	campos := strings.Fields(string(stat[strings.LastIndexByte(string(stat), ')')+1:]))
	if campos[16] != "7" {
		t.Errorf("nice do filho: %s, queria 7", campos[16])
	}
}

func TestGrupo_DosFilhosParaOsPais(t *testing.T) {
	procFalso(t,
		processo{100, 1, "firefox", -1},
		processo{110, 100, "Web Content", -1},
		processo{111, 110, "neto", -1},
		processo{120, 100, "Socket Process", -1},
	)
	enviados := semSinais(t, nil)
	pids, err := Grupo([]int{100, 110, 111, 120, 110}, "encerrar", false)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(pids, []int{100, 110, 111, 120}) {
		t.Errorf("devolveu %v", pids)
	}
	ordem := []int{}
	for _, e := range *enviados {
		ordem = append(ordem, e.pid)
		if e.sig != syscall.SIGTERM {
			t.Errorf("sinal %v ao %d", e.sig, e.pid)
		}
	}
	if !reflect.DeepEqual(ordem, []int{111, 120, 110, 100}) {
		t.Errorf("ordem %v, queria o neto, os filhos e por fim o pai", ordem)
	}
}

func TestGrupo_UmaRecusaParaTudoAntesDoPrimeiroSinal(t *testing.T) {
	procFalso(t,
		processo{100, 1, "firefox", -1},
		processo{110, 100, "Web Content", -1},
		processo{600, 1, "Hyprland", -1},
	)
	enviados := semSinais(t, nil)
	for _, pids := range [][]int{{100, 110, 600}, {100, 1}, {100, -1}, {os.Getpid(), 100}} {
		if _, err := Grupo(pids, "matar", false); !EhRecusa(err) {
			t.Errorf("%v: queria recusa, veio %v", pids, err)
		}
	}
	if len(*enviados) != 0 {
		t.Errorf("mandou %v", *enviados)
	}
	if _, err := Grupo([]int{100, 110, 600}, "matar", true); err != nil {
		t.Errorf("com --mesmo-assim: %v", err)
	}
	if len(*enviados) != 3 {
		t.Errorf("mandou %v", *enviados)
	}
}

func TestGrupo_QuemJaSaiuFicaDeFora(t *testing.T) {
	procFalso(t, processo{100, 1, "firefox", -1})
	enviados := semSinais(t, nil)
	pids, err := Grupo([]int{100, 110}, "encerrar", false)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(pids, []int{100}) || len(*enviados) != 1 {
		t.Errorf("devolveu %v, mandou %v", pids, *enviados)
	}
	if _, err := Grupo(nil, "encerrar", false); err == nil {
		t.Error("grupo vazio devia ser erro")
	}
	pids, err = Grupo([]int{110, 111}, "encerrar", false)
	if err != nil || len(pids) != 0 || pids == nil {
		t.Errorf("todos já saíram: queria lista vazia sem erro, veio %v %v", pids, err)
	}
}

func TestLePid_SoAceitaNumeroInteiro(t *testing.T) {
	for _, s := range []string{"", "1e9", "12a", "0x10", " 12", "1.5", "+5", "99999999999999999999", "--pids"} {
		if _, err := LeNumero(s); err == nil {
			t.Errorf("%q passou", s)
		}
	}
	for s, n := range map[string]int{"1234": 1234, "-1": -1, "0": 0, "-20": -20} {
		v, err := LeNumero(s)
		if err != nil || v != n {
			t.Errorf("%q: %d %v", s, v, err)
		}
	}
}
