// Package acoes é a metade do coletor que muda a máquina: instalar, remover,
// tirar os órfãos, limpar o cache e abrir a atualização. Cada ação tem um
// ensaio (Planeja: o que vai entrar, sair e rodar, sem root) e a execução
// (Age: refaz o ensaio e as recusas, e transmite o que os comandos dizem).
//
// As regras de segurança moram neste arquivo:
//   - nome de pacote só chega a um comando validado e depois de "--";
//   - remover o que derruba o sistema é recusado sem --mesmo-assim;
//   - com o pacman travado (db.lck), nada roda;
//   - uma ação de cada vez (flock);
//   - nunca como root, nunca sudo: o root vem do pkexec, e o yay roda como a
//     pessoa.
package acoes

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"
	"syscall"

	"mkpacotes/internal/alpm"
	"mkpacotes/internal/base"
)

// Recusa é um não por segurança (código de saída 2), diferente de um erro.
type Recusa struct{ Frase string }

func (r *Recusa) Error() string { return r.Frase }

func recusa(formato string, args ...any) error {
	return &Recusa{Frase: fmt.Sprintf(formato, args...)}
}

// Euid é o usuário efetivo de quem roda. Os testes trocam.
var Euid = os.Geteuid

// O que nunca sai sem --mesmo-assim: sem um deles a máquina não liga, não
// entra na sessão ou não consegue mais instalar nada.
var fixos = []string{"base", "linux", "linux-firmware", "systemd", "pacman", "glibc", "sudo", "limine", "hyprland", "quickshell", "sddm"}

// Protegidos devolve os pacotes que o `remover` recusa: os fixos e o que o
// Mk declara como base (system/packages/base.packages, um por linha, com #
// para comentário).
func Protegidos() map[string]bool {
	p := map[string]bool{}
	for _, n := range fixos {
		p[n] = true
	}
	f, err := os.Open(base.Mk("system", "packages", "base.packages"))
	if err != nil {
		return p
	}
	defer f.Close()
	linhas := bufio.NewScanner(f)
	for linhas.Scan() {
		nome, _, _ := strings.Cut(linhas.Text(), "#")
		nome = strings.TrimSpace(nome)
		if alpm.ValidaNome(nome) == nil {
			p[nome] = true
		}
	}
	return p
}

// Travado diz se a trava do pacman existe: outro pacman está rodando, ou um
// morreu no meio e a deixou.
func Travado() bool {
	_, err := os.Lstat(base.PacmanDB("db.lck"))
	return err == nil
}

func recusaTravado() error {
	return recusa("o pacman está travado (%s existe): outro pacman está rodando, ou um morreu no meio. Espere ele terminar; se não há nenhum, rode mk atualizar, que oferece tirar a trava",
		base.PacmanDB("db.lck"))
}

// Tranca segura a trava do coletor (flock, sem esperar): só uma ação do
// MkPacotes roda por vez. Devolve como soltar. Quem morre solta sozinho.
func Tranca() (solta func(), err error) {
	f, err := os.OpenFile(base.Trava(), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("não consegui criar a trava do MkPacotes (%s)", base.Trava())
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		return nil, recusa("já há uma ação do MkPacotes em andamento: espere ela terminar")
	}
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		f.Close()
	}, nil
}

// limpa valida os nomes e tira os repetidos, guardando a ordem.
func limpa(nomes []string) ([]string, error) {
	var unicos []string
	visto := map[string]bool{}
	for _, n := range nomes {
		if err := alpm.ValidaNome(n); err != nil {
			return nil, err
		}
		if !visto[n] {
			visto[n] = true
			unicos = append(unicos, n)
		}
	}
	return unicos, nil
}

// protegidosEm devolve, em ordem, os nomes da lista que são protegidos.
func protegidosEm(nomes []string) []string {
	p := Protegidos()
	var quais []string
	for _, n := range nomes {
		if p[n] {
			quais = append(quais, n)
		}
	}
	sort.Strings(quais)
	return quais
}

// lista escreve nomes como numa frase: "a", "a e b", "a, b e c".
func lista(nomes []string) string {
	switch len(nomes) {
	case 0:
		return ""
	case 1:
		return nomes[0]
	}
	return strings.Join(nomes[:len(nomes)-1], ", ") + " e " + nomes[len(nomes)-1]
}
