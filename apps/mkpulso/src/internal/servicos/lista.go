package servicos

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Servico é uma linha de `mkpulso-coletor servicos`. O que não deu para ler
// é null.
type Servico struct {
	Unidade    string `json:"unidade"`
	Escopo     string `json:"escopo"`
	Descricao  string `json:"descricao"`
	Estado     string `json:"estado"`
	Sub        string `json:"sub"`
	Habilitado string `json:"habilitado"`
	Pid        *int   `json:"pid"`
	MemKb      *int64 `json:"mem_kb"`
	DesdeS     *int64 `json:"desde_s"`
}

// Os tipos de unidade que o contrato mostra.
const tipos = "--type=service,socket,timer"

// traduzEstado leva o ActiveState do systemd para o `estado` do contrato.
func traduzEstado(active string) string {
	switch active {
	case "active", "reloading", "refreshing":
		return "ativo"
	case "failed":
		return "falhou"
	case "activating":
		return "ativando"
	case "deactivating":
		return "desativando"
	}
	return "inativo"
}

// traduzHabilitado leva o estado do arquivo da unidade para o `habilitado`
// do contrato. Sem arquivo (ou sem [Install]) é "estático".
func traduzHabilitado(estado string) string {
	switch estado {
	case "enabled", "enabled-runtime", "linked", "linked-runtime", "alias":
		return "sim"
	case "disabled", "indirect", "bad":
		return "não"
	case "masked", "masked-runtime":
		return "mascarado"
	case "generated", "transient":
		return "gerado"
	}
	return "estático"
}

// modelo devolve "getty@.service" para "getty@tty1.service" ("" se o nome
// não é instância de um modelo).
func modelo(unidade string) string {
	arroba := strings.IndexByte(unidade, '@')
	ponto := strings.LastIndexByte(unidade, '.')
	if arroba < 0 || ponto < arroba+2 {
		return ""
	}
	return unidade[:arroba+1] + unidade[ponto:]
}

func ehModelo(unidade string) bool { return strings.Contains(unidade, "@.") }

// Lista devolve as unidades .service, .socket e .timer: as de sistema
// primeiro, depois as de usuário, cada parte em ordem de nome.
func Lista(usuario, sistema bool) ([]Servico, error) {
	lista := []Servico{}
	if sistema {
		parte, err := listaEscopo(false)
		if err != nil {
			return nil, err
		}
		lista = append(lista, parte...)
	}
	if usuario {
		parte, err := listaEscopo(true)
		if err != nil {
			return nil, err
		}
		lista = append(lista, parte...)
	}
	return lista, nil
}

func leJSON(saida []byte, destino any, oque string) error {
	if err := json.Unmarshal(saida, destino); err != nil {
		return fmt.Errorf("não entendi a resposta do systemctl (%s)", oque)
	}
	return nil
}

func listaEscopo(usuario bool) ([]Servico, error) {
	escopo := "sistema"
	if usuario {
		escopo = "usuario"
	}
	saida, err := Systemctl(usuario, "list-units", tipos, "--all", "--output=json", "--no-pager")
	if err != nil {
		return nil, err
	}
	var carregadas []struct {
		Unit, Load, Active, Sub, Description string
	}
	if err := leJSON(saida, &carregadas, "list-units"); err != nil {
		return nil, err
	}
	saida, err = Systemctl(usuario, "list-unit-files", tipos, "--output=json", "--no-pager")
	if err != nil {
		return nil, err
	}
	var arquivos []struct {
		UnitFile string `json:"unit_file"`
		State    string `json:"state"`
	}
	if err := leJSON(saida, &arquivos, "list-unit-files"); err != nil {
		return nil, err
	}
	estadoDoArquivo := map[string]string{}
	for _, a := range arquivos {
		estadoDoArquivo[a.UnitFile] = a.State
	}

	servicos := map[string]*Servico{}
	for _, c := range carregadas {
		if c.Load == "not-found" || ehModelo(c.Unit) {
			continue // citada por alguém, mas não existe
		}
		servicos[c.Unit] = &Servico{Unidade: c.Unit, Escopo: escopo, Descricao: c.Description,
			Estado: traduzEstado(c.Active), Sub: c.Sub}
	}
	// O que tem arquivo mas nunca foi carregado também aparece (parado),
	// menos os apelidos e os modelos sem instância.
	for _, a := range arquivos {
		if _, ja := servicos[a.UnitFile]; ja || a.State == "alias" || ehModelo(a.UnitFile) {
			continue
		}
		servicos[a.UnitFile] = &Servico{Unidade: a.UnitFile, Escopo: escopo, Estado: "inativo", Sub: "dead"}
	}
	nomes := make([]string, 0, len(servicos))
	for nome := range servicos {
		nomes = append(nomes, nome)
	}
	sort.Strings(nomes)

	fichas, err := Mostra(usuario, []string{"Id", "Description", "MainPID", "MemoryCurrent", "ActiveEnterTimestamp"}, nomes)
	if err != nil {
		return nil, err
	}
	lista := make([]Servico, 0, len(nomes))
	for i, nome := range nomes {
		s := servicos[nome]
		estado, tem := estadoDoArquivo[nome]
		if !tem {
			estado = estadoDoArquivo[modelo(nome)]
		}
		s.Habilitado = traduzHabilitado(estado)
		ficha := fichas[i]
		if s.Descricao == "" {
			s.Descricao = ficha["Description"]
		}
		if pid, err := strconv.Atoi(ficha["MainPID"]); err == nil && pid > 0 {
			s.Pid = &pid
		}
		if bytes, err := strconv.ParseInt(ficha["MemoryCurrent"], 10, 64); err == nil && bytes >= 0 {
			kb := bytes / 1024
			s.MemKb = &kb
		}
		if s.Estado == "ativo" || s.Estado == "desativando" {
			s.DesdeS = epoca(ficha["ActiveEnterTimestamp"])
		}
		lista = append(lista, *s)
	}
	return lista, nil
}

// epoca lê "@1795470000" (o formato de --timestamp=unix).
func epoca(texto string) *int64 {
	if !strings.HasPrefix(texto, "@") {
		return nil
	}
	s, err := strconv.ParseInt(texto[1:], 10, 64)
	if err != nil || s <= 0 {
		return nil
	}
	return &s
}

// Os nomes de unidade que o coletor aceita: as letras que o systemd permite
// (inclusive a barra invertida dos escapes, como em `app-x\x2dy@z.service`),
// sem começar por "-" e com um dos sufixos dados.
var nomeDeUnidade = regexp.MustCompile(`^[A-Za-z0-9:_.@\\][A-Za-z0-9:_.@\\-]*$`)

func validaNome(nome string, sufixos ...string) error {
	if nome == "" {
		return fmt.Errorf("faltou o nome da unidade")
	}
	if len(nome) > 255 || !nomeDeUnidade.MatchString(nome) {
		return fmt.Errorf("%q não é nome de unidade", nome)
	}
	for _, s := range sufixos {
		if strings.HasSuffix(nome, s) && len(nome) > len(s) {
			return nil
		}
	}
	return fmt.Errorf("%q não é uma unidade %s", nome, strings.Join(sufixos, ", "))
}

// ValidaUnidade só deixa passar nome de .service, .socket ou .timer feito
// das letras de um nome de unidade: nada de opção ("--now"), espaço, barra
// ou ponto e vírgula chega ao systemctl.
func ValidaUnidade(nome string) error {
	return validaNome(nome, ".service", ".socket", ".timer")
}
