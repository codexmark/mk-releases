// Package acoes manda sinais e muda a prioridade de processos, e guarda as
// recusas de segurança do contrato (../../../CONTRATO.md): o processo 1, o
// próprio coletor e quem o chamou, processos de outro usuário e a sessão
// gráfica (esta última passa com --mesmo-assim).
//
// As chamadas do sistema ficam em variáveis (Mata, MudaNice) para os testes
// trocarem: teste nenhum manda sinal a processo que não seja filho dele.
package acoes

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"mkpulso/internal/base"
)

// Mata manda um sinal a um processo. Os testes trocam.
var Mata = syscall.Kill

// MudaNice muda o nice de um processo (ou de uma thread, pelo tid). Os
// testes trocam.
var MudaNice = func(pid, nice int) error {
	return syscall.Setpriority(syscall.PRIO_PROCESS, pid, nice)
}

// Recusa é um erro de segurança: o coletor não faz o que foi pedido, e sai
// com base.Recusado em vez de base.Erro.
type Recusa struct{ Frase string }

func (r *Recusa) Error() string { return r.Frase }

// Recusar monta uma Recusa com a frase dada.
func Recusar(formato string, args ...any) error {
	return &Recusa{Frase: fmt.Sprintf(formato, args...)}
}

// EhRecusa diz se o erro é (ou embrulha) uma Recusa.
func EhRecusa(err error) bool {
	var r *Recusa
	return errors.As(err, &r)
}

// Codigo é o código de saída de um erro: base.Recusado para uma Recusa,
// base.Erro para o resto, base.Certo sem erro.
func Codigo(err error) int {
	switch {
	case err == nil:
		return base.Certo
	case EhRecusa(err):
		return base.Recusado
	}
	return base.Erro
}

// Os sinais do contrato, na ordem em que ele os apresenta.
var sinais = []struct {
	nome string
	sig  syscall.Signal
}{
	{"encerrar", syscall.SIGTERM},
	{"matar", syscall.SIGKILL},
	{"pausar", syscall.SIGSTOP},
	{"continuar", syscall.SIGCONT},
	{"recarregar", syscall.SIGHUP},
	{"interromper", syscall.SIGINT},
}

// Sinais devolve os nomes de sinal que o contrato aceita.
func Sinais() []string {
	nomes := make([]string, len(sinais))
	for i, s := range sinais {
		nomes[i] = s.nome
	}
	return nomes
}

// NumeroDoSinal traduz o nome do contrato ("encerrar") para o sinal.
func NumeroDoSinal(nome string) (syscall.Signal, error) {
	for _, s := range sinais {
		if s.nome == nome {
			return s.sig, nil
		}
	}
	return 0, fmt.Errorf("não conheço o sinal %q: use %s", nome, strings.Join(Sinais(), ", "))
}

var inteiro = regexp.MustCompile(`^-?[0-9]+$`)

// LeNumero lê um pid ou um nice da linha de comando: só dígitos, com um
// menos opcional na frente. Nada de "1e9", "0x10" nem espaço.
func LeNumero(texto string) (int, error) {
	if !inteiro.MatchString(texto) {
		return 0, fmt.Errorf("%q não é um número inteiro", texto)
	}
	n, err := strconv.Atoi(texto)
	if err != nil {
		return 0, fmt.Errorf("%q não é um número que caiba", texto)
	}
	return n, nil
}

// Os processos que são a sessão: sem eles, tudo que está aberto cai. O nome
// é o comm do kernel (até 15 letras), lido de /proc/<pid>/status.
var daSessao = map[string]bool{
	"Hyprland":       true,
	"start-hyprland": true,
	"quickshell":     true, // o Maré
	"qs":             true, // o Maré, pelo nome curto
	"sddm":           true,
	"sddm-helper":    true,
	"systemd":        true, // o systemd do usuário (o da máquina é de outro usuário)
}

// ficha é o que as recusas precisam saber de um processo.
type ficha struct {
	nome string
	uid  int
	ppid int
}

var errSaiu = errors.New("o processo já saiu")

// le lê a ficha de /proc/<pid>/status. Um processo que não existe dá errSaiu.
func le(pid int) (ficha, error) {
	f := ficha{uid: -1, ppid: -1}
	b, err := os.ReadFile(base.Proc(strconv.Itoa(pid), "status"))
	if err != nil {
		if os.IsNotExist(err) || errors.Is(err, syscall.ESRCH) {
			return f, errSaiu
		}
		return f, err
	}
	for _, linha := range strings.Split(string(b), "\n") {
		chave, valor, ok := strings.Cut(linha, ":")
		if !ok {
			continue
		}
		valor = strings.TrimSpace(valor)
		switch chave {
		case "Name":
			f.nome = valor
		case "PPid":
			if n, err := strconv.Atoi(valor); err == nil {
				f.ppid = n
			}
		case "Uid":
			// Real, efetivo, salvo e de arquivos: vale o real.
			if campos := strings.Fields(valor); len(campos) > 0 {
				if n, err := strconv.Atoi(campos[0]); err == nil {
					f.uid = n
				}
			}
		}
	}
	return f, nil
}

// confere aplica as recusas a um pid e devolve a ficha dele. Um processo que
// não existe mais é um erro comum ("já saiu"), não uma recusa.
func confere(pid int, mesmoAssim bool) (ficha, error) {
	switch {
	case pid <= 0:
		return ficha{}, Recusar("%d não é um processo: não mando sinal a um grupo inteiro nem a todos de uma vez", pid)
	case pid == 1:
		return ficha{}, Recusar("não mexo no processo 1: é o systemd da máquina")
	case pid == os.Getpid():
		return ficha{}, Recusar("não mexo no próprio coletor do MkPulso")
	case pid == os.Getppid():
		return ficha{}, Recusar("não mexo em quem chamou o coletor (o processo %d)", pid)
	}
	f, err := le(pid)
	if errors.Is(err, errSaiu) {
		return f, fmt.Errorf("o processo %d já saiu", pid)
	}
	if err != nil || f.uid < 0 {
		return f, Recusar("não consegui ver de quem é o processo %d: não mexo nele", pid)
	}
	if f.uid != os.Getuid() {
		return f, Recusar("o processo %d é de outro usuário: não mexo nele", pid)
	}
	if daSessao[f.nome] && !mesmoAssim {
		return f, Recusar("o processo %d (%s) é a sessão: sem ele cai tudo que está aberto (--mesmo-assim para insistir)", pid, f.nome)
	}
	return f, nil
}

// Confere diz se o coletor pode mexer no processo: nil, uma Recusa, ou o
// erro de que ele já saiu.
func Confere(pid int, mesmoAssim bool) error {
	_, err := confere(pid, mesmoAssim)
	return err
}

// erroDoSistema traduz o que o kernel respondeu.
func erroDoSistema(err error, pid int, oque string) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, syscall.ESRCH):
		return fmt.Errorf("o processo %d já saiu", pid)
	case errors.Is(err, syscall.EPERM):
		return fmt.Errorf("o sistema não deixou %s (processo %d)", oque, pid)
	}
	return fmt.Errorf("não consegui %s (processo %d): %v", oque, pid, err)
}

// Sinal manda a um processo um dos sinais do contrato, depois das recusas.
func Sinal(pid int, nome string, mesmoAssim bool) error {
	sig, err := NumeroDoSinal(nome)
	if err != nil {
		return err
	}
	if _, err := confere(pid, mesmoAssim); err != nil {
		return err
	}
	return erroDoSistema(Mata(pid, sig), pid, "mandar o sinal")
}

// Grupo manda o sinal a vários processos de uma vez, dos filhos para os
// pais. Uma recusa em qualquer um para tudo antes do primeiro sinal; quem já
// saiu fica de fora. Devolve os pids que receberam o sinal, em ordem.
func Grupo(pids []int, nome string, mesmoAssim bool) ([]int, error) {
	sig, err := NumeroDoSinal(nome)
	if err != nil {
		return nil, err
	}
	if len(pids) == 0 {
		return nil, errors.New("o grupo não tem processo nenhum")
	}
	fichas := map[int]ficha{}
	for _, pid := range pids {
		if _, visto := fichas[pid]; visto {
			continue
		}
		f, err := confere(pid, mesmoAssim)
		if EhRecusa(err) {
			return nil, err
		}
		if err != nil {
			continue // já saiu
		}
		fichas[pid] = f
	}
	// Fundura: quantos antepassados o processo tem dentro do próprio grupo.
	fundura := func(pid int) int {
		n := 0
		for visto := map[int]bool{pid: true}; ; n++ {
			pai := fichas[pid].ppid
			if _, dentro := fichas[pai]; !dentro || visto[pai] {
				return n
			}
			visto[pai] = true
			pid = pai
		}
	}
	ordem := make([]int, 0, len(fichas))
	for pid := range fichas {
		ordem = append(ordem, pid)
	}
	sort.Slice(ordem, func(i, j int) bool {
		fi, fj := fundura(ordem[i]), fundura(ordem[j])
		if fi != fj {
			return fi > fj
		}
		return ordem[i] > ordem[j]
	})
	feitos := make([]int, 0, len(ordem))
	for _, pid := range ordem {
		err := Mata(pid, sig)
		if errors.Is(err, syscall.ESRCH) {
			continue
		}
		if err != nil {
			return nil, erroDoSistema(err, pid, "mandar o sinal")
		}
		feitos = append(feitos, pid)
	}
	sort.Ints(feitos)
	return feitos, nil
}

// ErroDeSubir é a frase do contrato para quando o sistema não deixa dar mais
// prioridade (baixar o nice) a um processo.
const ErroDeSubir = "só dá para baixar a prioridade; para subir é preciso ser administrador"

// Prioridade muda o nice de um processo (-20 a 19), depois das recusas. No
// Linux o nice é de cada thread, então vai em todas as de /proc/<pid>/task.
func Prioridade(pid, nice int, mesmoAssim bool) error {
	if nice < -20 || nice > 19 {
		return fmt.Errorf("a prioridade (nice) vai de -20 a 19, e veio %d", nice)
	}
	if _, err := confere(pid, mesmoAssim); err != nil {
		return err
	}
	if err := erroDeNice(MudaNice(pid, nice), pid); err != nil {
		return err
	}
	entradas, _ := os.ReadDir(base.Proc(strconv.Itoa(pid), "task"))
	for _, e := range entradas {
		tid, err := strconv.Atoi(e.Name())
		if err != nil || tid == pid {
			continue
		}
		// Uma thread que acabou no meio do caminho não é erro.
		if err := MudaNice(tid, nice); err != nil && !errors.Is(err, syscall.ESRCH) {
			return erroDeNice(err, pid)
		}
	}
	return nil
}

func erroDeNice(err error, pid int) error {
	if errors.Is(err, syscall.EACCES) {
		return errors.New(ErroDeSubir)
	}
	return erroDoSistema(err, pid, "mudar a prioridade")
}

// DaSessao diz se um processo é dos que seguram a sessão (ou é o próprio
// coletor, ou quem o chamou) e com que nome mostrá-lo. Serve para recusar
// parar uma unidade inteira por causa de quem mora nela.
func DaSessao(pid int) (string, bool) {
	switch pid {
	case os.Getpid():
		return "o próprio coletor do MkPulso", true
	case os.Getppid():
		return "quem chamou o coletor", true
	}
	if f, err := le(pid); err == nil && daSessao[f.nome] {
		return f.nome, true
	}
	return "", false
}
