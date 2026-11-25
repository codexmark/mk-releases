package vm

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// LePid lê o arquivo de pid que o QEMU escreveu. 0 se não há ou não é um pid.
func LePid(arquivo string) int {
	b, err := os.ReadFile(arquivo)
	if err != nil {
		return 0
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil || pid <= 1 {
		return 0
	}
	return pid
}

// Cmdline devolve os argumentos de um processo (do /proc/<pid>/cmdline). nil
// se o processo não existe.
func Cmdline(proc string, pid int) []string {
	b, err := os.ReadFile(filepath.Join(proc, strconv.Itoa(pid), "cmdline"))
	if err != nil || len(b) == 0 {
		return nil
	}
	partes := bytes.Split(bytes.TrimRight(b, "\x00"), []byte{0})
	args := make([]string, len(partes))
	for i, p := range partes {
		args[i] = string(p)
	}
	return args
}

// EDaMaquina diz se o processo é o QEMU daquela máquina: na linha de comando
// dele há `-name <nome>` e o `-pidfile` dela. É o que se confere antes de
// mandar um sinal: o pid de um arquivo velho pode ser hoje de outro programa.
func EDaMaquina(proc string, pid int, nome, pidfile string) bool {
	args := Cmdline(proc, pid)
	temNome, temPid := false, false
	for i := 0; i+1 < len(args); i++ {
		if args[i] == "-name" && args[i+1] == nome {
			temNome = true
		}
		if args[i] == "-pidfile" && args[i+1] == pidfile {
			temPid = true
		}
	}
	return temNome && temPid
}

// Uso é o que o /proc diz do processo: o tempo de CPU gasto (em ticks, modo
// usuário mais sistema) e a memória residente.
type Uso struct {
	Ticks int64
	RssKB int64
}

// LeUso lê /proc/<pid>/stat e /proc/<pid>/status.
func LeUso(proc string, pid int) (Uso, bool) {
	var u Uso
	stat, err := os.ReadFile(filepath.Join(proc, strconv.Itoa(pid), "stat"))
	if err != nil {
		return u, false
	}
	// O nome do programa vem entre parênteses e pode ter espaço e parêntese:
	// os campos se contam depois do último ")". utime e stime são o 14º e o
	// 15º da linha, isto é, o 12º e o 13º depois do nome.
	fim := bytes.LastIndexByte(stat, ')')
	if fim < 0 {
		return u, false
	}
	campos := strings.Fields(string(stat[fim+1:]))
	if len(campos) < 13 {
		return u, false
	}
	utime, err1 := strconv.ParseInt(campos[11], 10, 64)
	stime, err2 := strconv.ParseInt(campos[12], 10, 64)
	if err1 != nil || err2 != nil {
		return u, false
	}
	u.Ticks = utime + stime
	status, err := os.ReadFile(filepath.Join(proc, strconv.Itoa(pid), "status"))
	if err == nil {
		for _, linha := range strings.Split(string(status), "\n") {
			if strings.HasPrefix(linha, "VmRSS:") {
				partes := strings.Fields(linha)
				if len(partes) >= 2 {
					u.RssKB, _ = strconv.ParseInt(partes[1], 10, 64)
				}
			}
		}
	}
	return u, true
}
