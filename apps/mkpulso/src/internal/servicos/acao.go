package servicos

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"mkpulso/internal/acoes"
	"mkpulso/internal/base"
)

// As ações do contrato e o verbo do systemctl de cada uma.
var verbos = []struct{ acao, verbo string }{
	{"iniciar", "start"},
	{"parar", "stop"},
	{"reiniciar", "restart"},
	{"habilitar", "enable"},
	{"desabilitar", "disable"},
	{"recarregar", "reload"},
}

// Acoes devolve os nomes de ação que o contrato aceita.
func Acoes() []string {
	nomes := make([]string, len(verbos))
	for i, v := range verbos {
		nomes[i] = v.acao
	}
	return nomes
}

// derruba diz se parar a unidade derruba a sessão ou a máquina: o sddm, o
// D-Bus, o que é do próprio systemd, o gerente de um usuário e a rede.
func derruba(unidade string) bool {
	nome := unidade[:strings.LastIndexByte(unidade, '.')]
	switch {
	case nome == "sddm", nome == "NetworkManager":
		return true
	case nome == "dbus", strings.HasPrefix(nome, "dbus-"), strings.HasPrefix(nome, "dbus@"):
		return true
	case strings.HasPrefix(nome, "systemd-"), strings.HasPrefix(nome, "user@"):
		return true
	}
	return false
}

// moraASessao recusa mexer numa unidade que tem dentro o Hyprland, o Maré
// ou o próprio coletor: pará-la é derrubar a sessão por outro caminho.
func moraASessao(unidade string, usuario bool) error {
	for _, pid := range PidsDaUnidade(unidade, usuario) {
		if quem, sim := acoes.DaSessao(pid); sim {
			return acoes.Recusar("%s tem dentro %s (processo %d): mexer nela derruba a sessão (--mesmo-assim para insistir)", unidade, quem, pid)
		}
	}
	return nil
}

// Age executa uma ação do contrato numa unidade. Não passa
// --no-ask-password: se precisar de senha, é o polkit que pergunta (o Maré é
// o agente). Parar ou desabilitar o que derruba a sessão ou a máquina só
// com mesmoAssim; reiniciar também, menos a rede (que volta sozinha).
func Age(acao, unidade string, usuario, mesmoAssim bool) error {
	verbo := ""
	for _, v := range verbos {
		if v.acao == acao {
			verbo = v.verbo
		}
	}
	if verbo == "" {
		return fmt.Errorf("não conheço a ação %q: use %s", acao, strings.Join(Acoes(), ", "))
	}
	if err := ValidaUnidade(unidade); err != nil {
		return err
	}
	if !mesmoAssim {
		para := acao == "parar" || acao == "reiniciar"
		if derruba(unidade) && (acao == "parar" || acao == "desabilitar" || (acao == "reiniciar" && unidade != "NetworkManager.service")) {
			return acoes.Recusar("%s %s pode derrubar a sessão ou a máquina (--mesmo-assim para insistir)", acao, unidade)
		}
		if para {
			if err := moraASessao(unidade, usuario); err != nil {
				return err
			}
		}
	}
	if _, err := systemctl(usuario, 0, verbo, "--", unidade); err != nil {
		return fmt.Errorf("não deu para %s %s: %v", acao, unidade, err)
	}
	return nil
}

// PidsDaUnidade devolve os processos de uma unidade, em ordem: os do cgroup
// dela (e dos que estão dentro) ou, sem cgroup legível, o processo
// principal. Lista vazia quando não há nenhum ou não deu para saber.
func PidsDaUnidade(unidade string, usuario bool) []int {
	pids := []int{}
	fichas, err := Mostra(usuario, []string{"ControlGroup", "MainPID"}, []string{unidade})
	if err != nil || len(fichas) == 0 {
		return pids
	}
	visto := map[int]bool{}
	if cg := fichas[0]["ControlGroup"]; strings.HasPrefix(cg, "/") && len(cg) > 1 && !saiDaArvore(cg) {
		_ = filepath.WalkDir(base.Sys("fs", "cgroup", cg), func(caminho string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || d.Name() != "cgroup.procs" {
				return nil
			}
			b, _ := os.ReadFile(caminho)
			for _, campo := range strings.Fields(string(b)) {
				if pid, err := strconv.Atoi(campo); err == nil && pid > 0 && !visto[pid] {
					visto[pid] = true
					pids = append(pids, pid)
				}
			}
			return nil
		})
	}
	if len(pids) == 0 {
		if pid, err := strconv.Atoi(fichas[0]["MainPID"]); err == nil && pid > 0 {
			pids = append(pids, pid)
		}
	}
	sort.Ints(pids)
	return pids
}

func saiDaArvore(cgroup string) bool {
	for _, parte := range strings.Split(cgroup, "/") {
		if parte == ".." {
			return true
		}
	}
	return false
}

// UnidadesDoApp acha as unidades de usuário de um app: os escopos
// `app-<nome>-*.scope` e os serviços `app-<nome>@*.service`. O systemd
// escreve o "-" do nome do app como `\x2d`, então procura dos dois jeitos.
func UnidadesDoApp(nome string) ([]string, error) {
	for _, r := range nome {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '_' || r == '-') {
			return nil, nil // um nome assim não vira unidade
		}
	}
	if nome == "" {
		return nil, nil
	}
	saida, err := Systemctl(true, "list-units", "--type=scope,service", "--output=json", "--no-pager", "app-*")
	if err != nil {
		return nil, err
	}
	var carregadas []struct{ Unit string }
	if err := leJSON(saida, &carregadas, "list-units"); err != nil {
		return nil, err
	}
	jeitos := []string{nome, strings.ReplaceAll(nome, "-", `\x2d`)}
	var unidades []string
	for _, c := range carregadas {
		for _, j := range jeitos {
			escopo := strings.HasPrefix(c.Unit, "app-"+j+"-") && strings.HasSuffix(c.Unit, ".scope")
			servico := strings.HasPrefix(c.Unit, "app-"+j+"@") && strings.HasSuffix(c.Unit, ".service")
			if (escopo || servico) && validaNome(c.Unit, ".scope", ".service") == nil {
				unidades = append(unidades, c.Unit)
				break
			}
		}
	}
	sort.Strings(unidades)
	return unidades, nil
}

// SinalNaUnidade manda um sinal do contrato a todos os processos de uma
// unidade (serviço, soquete, timer ou escopo): `systemctl stop` para
// "encerrar", `systemctl kill` para o resto. Devolve os pids que a unidade
// tinha. As recusas são as de Age, mais a de quem mora na unidade.
func SinalNaUnidade(unidade string, usuario bool, sinal string, mesmoAssim bool) ([]int, error) {
	sig, err := acoes.NumeroDoSinal(sinal)
	if err != nil {
		return nil, err
	}
	if err := validaNome(unidade, ".service", ".socket", ".timer", ".scope"); err != nil {
		return nil, err
	}
	if !mesmoAssim && sinal != "continuar" && derruba(unidade) {
		return nil, acoes.Recusar("%s %s pode derrubar a sessão ou a máquina (--mesmo-assim para insistir)", sinal, unidade)
	}
	pids := PidsDaUnidade(unidade, usuario)
	if !mesmoAssim && sinal != "continuar" {
		for _, pid := range pids {
			if quem, sim := acoes.DaSessao(pid); sim {
				return nil, acoes.Recusar("%s tem dentro %s (processo %d): mexer nela derruba a sessão (--mesmo-assim para insistir)", unidade, quem, pid)
			}
		}
	}
	args := []string{"kill", "--signal=" + nomeDoSinal[sig], "--", unidade}
	if sinal == "encerrar" {
		args = []string{"stop", "--", unidade}
	}
	if _, err := systemctl(usuario, 0, args...); err != nil {
		return nil, fmt.Errorf("não deu para %s %s: %v", sinal, unidade, err)
	}
	return pids, nil
}
