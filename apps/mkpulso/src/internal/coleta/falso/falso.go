// Package falso monta raízes de mentira (/proc, /sys, /etc, /run, a home e
// um PATH só com dublês) para os testes do coletor. Nada aqui toca na
// máquina: tudo fica dentro de t.TempDir().
package falso

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// Raizes cria a pasta do teste e aponta para dentro dela todas as raízes
// trocáveis do contrato. O PATH fica só com <raiz>/bin: um comando externo
// só existe se o teste puser um dublê (veja Comando).
func Raizes(t testing.TB) string {
	t.Helper()
	raiz := t.TempDir()
	for _, d := range []string{"proc", "sys", "etc", "run", "home", "bin", "apps", "pacman"} {
		if err := os.MkdirAll(filepath.Join(raiz, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("MKPULSO_PROC", filepath.Join(raiz, "proc"))
	t.Setenv("MKPULSO_SYS", filepath.Join(raiz, "sys"))
	t.Setenv("MKPULSO_ETC", filepath.Join(raiz, "etc"))
	t.Setenv("MKPULSO_RUN", filepath.Join(raiz, "run"))
	t.Setenv("MKPULSO_HOME", filepath.Join(raiz, "home"))
	t.Setenv("MKPULSO_APPS", filepath.Join(raiz, "apps"))
	t.Setenv("MKPULSO_PACMAN", filepath.Join(raiz, "pacman"))
	t.Setenv("MKPULSO_PCIIDS", filepath.Join(raiz, "pci.ids"))
	t.Setenv("MKPULSO_MK", filepath.Join(raiz, "mk"))
	t.Setenv("XDG_DATA_DIRS", "")
	t.Setenv("PATH", filepath.Join(raiz, "bin"))
	return raiz
}

// Escreve grava um arquivo dentro da raiz, criando as pastas do caminho.
func Escreve(t testing.TB, raiz, caminho, conteudo string) {
	t.Helper()
	inteiro := filepath.Join(raiz, caminho)
	if err := os.MkdirAll(filepath.Dir(inteiro), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inteiro, []byte(conteudo), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Apaga tira um arquivo ou uma pasta inteira de dentro da raiz.
func Apaga(t testing.TB, raiz, caminho string) {
	t.Helper()
	if err := os.RemoveAll(filepath.Join(raiz, caminho)); err != nil {
		t.Fatal(err)
	}
}

// Comando põe um dublê no PATH do teste: um script de /bin/sh com o corpo
// dado. Use só comandos internos do shell (echo, printf, exit) ou caminhos
// inteiros (/bin/sleep), porque o PATH do teste não tem /usr/bin.
func Comando(t testing.TB, raiz, nome, corpo string) {
	t.Helper()
	inteiro := filepath.Join(raiz, "bin", nome)
	if err := os.WriteFile(inteiro, []byte("#!/bin/sh\n"+corpo+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}

// Proc descreve um processo de mentira. Os campos vazios ganham um valor
// comum (estado "S", um thread).
type Proc struct {
	PID, PPID    int
	Comm         string
	Estado       string
	Cmd          []string
	UID          int
	Utime, Stime uint64
	Threads      int
	Nice         int
	Inicio       uint64
	RSS          int64 // kB; 0 não escreve a linha VmRSS
	Cgroup       string
	Leu, Gravou  uint64
	SemIO        bool
	Bandeiras    uint64
}

// Grava escreve o processo em <raiz>/proc/<pid>/ (stat, status, cmdline,
// comm, io e cgroup), no formato do kernel.
func (p Proc) Grava(t testing.TB, raiz string) {
	t.Helper()
	if p.Estado == "" {
		p.Estado = "S"
	}
	if p.Threads == 0 {
		p.Threads = 1
	}
	dir := fmt.Sprintf("proc/%d/", p.PID)
	stat := fmt.Sprintf("%d (%s) %s %d %d %d 0 -1 %d 100 0 0 0 %d %d 0 0 20 %d %d 0 %d 1000000 %d 18446744073709551615 0 0 0 0 0 0 0 0 0 0 0 0 17 1 0 0 0 0 0\n",
		p.PID, p.Comm, p.Estado, p.PPID, p.PID, p.PID, p.Bandeiras, p.Utime, p.Stime, p.Nice, p.Threads, p.Inicio, p.RSS/4)
	Escreve(t, raiz, dir+"stat", stat)
	status := fmt.Sprintf("Name:\t%s\nUmask:\t0022\nState:\t%s\nPid:\t%d\nPPid:\t%d\nUid:\t%d\t%d\t%d\t%d\nGid:\t%d\t%d\t%d\t%d\n",
		p.Comm, p.Estado, p.PID, p.PPID, p.UID, p.UID, p.UID, p.UID, p.UID, p.UID, p.UID, p.UID)
	if p.RSS > 0 {
		status += fmt.Sprintf("VmPeak:\t%8d kB\nVmRSS:\t%8d kB\n", p.RSS*2, p.RSS)
	}
	status += fmt.Sprintf("Threads:\t%d\n", p.Threads)
	Escreve(t, raiz, dir+"status", status)
	cmd := ""
	for _, a := range p.Cmd {
		cmd += a + "\x00"
	}
	Escreve(t, raiz, dir+"cmdline", cmd)
	Escreve(t, raiz, dir+"comm", p.Comm+"\n")
	if !p.SemIO {
		Escreve(t, raiz, dir+"io", fmt.Sprintf("rchar: 1\nwchar: 1\nsyscr: 1\nsyscw: 1\nread_bytes: %d\nwrite_bytes: %d\ncancelled_write_bytes: 0\n", p.Leu, p.Gravou))
	}
	if p.Cgroup != "" {
		Escreve(t, raiz, dir+"cgroup", "0::"+p.Cgroup+"\n")
	}
}

// Stat escreve um /proc/stat com os ticks dados por núcleo: cada item é
// {ocupado, ocioso}. O btime é fixo (1795000000).
func Stat(t testing.TB, raiz string, nucleos ...[2]uint64) {
	t.Helper()
	var ocupado, ocioso uint64
	linhas := ""
	for i, n := range nucleos {
		ocupado += n[0]
		ocioso += n[1]
		linhas += fmt.Sprintf("cpu%d %d 0 0 %d 0 0 0 0 0 0\n", i, n[0], n[1])
	}
	Escreve(t, raiz, "proc/stat", fmt.Sprintf("cpu  %d 0 0 %d 0 0 0 0 0 0\n%sintr 1 2 3\nctxt 5\nbtime 1795000000\nprocesses 9\n", ocupado, ocioso, linhas))
}
