// Package inicio cuida do que sobe quando a pessoa entra na sessão, nos
// três tipos do contrato (../../../CONTRATO.md): os atalhos de autostart do
// padrão XDG, as unidades de usuário habilitadas e o que o autostart.lua do
// Mk sobe (este, só para ler).
package inicio

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"mkpulso/internal/base"
	"mkpulso/internal/servicos"
)

// Item é uma linha de `mkpulso-coletor inicio`.
type Item struct {
	ID        string `json:"id"`
	Origem    string `json:"origem"`
	Nome      string `json:"nome"`
	Comando   string `json:"comando"`
	Ligado    bool   `json:"ligado"`
	DoSistema bool   `json:"do_sistema"`
	Mudavel   bool   `json:"mudavel"`
}

// Os prefixos de id e os textos de origem do contrato.
const (
	prefixoAtalho  = "desktop:"
	prefixoUnidade = "unidade:"
	prefixoMk      = "mk:"

	origemAtalho  = "atalho"
	origemUnidade = "serviço de usuário"
	origemMk      = "Mk"
)

// Lista devolve tudo que sobe no login: os atalhos, as unidades de usuário
// e o que é do Mk, nessa ordem. Sem systemctl (ou sem a sessão do systemd),
// as unidades ficam de fora e o resto aparece do mesmo jeito.
func Lista() ([]Item, error) {
	itens := []Item{}
	atalhos, err := listaAtalhos()
	if err != nil {
		return nil, err
	}
	itens = append(itens, atalhos...)
	itens = append(itens, listaUnidades()...)
	itens = append(itens, listaMk()...)
	return itens, nil
}

// Muda liga ou desliga um item pelo id. O que é do Mk não muda por aqui.
func Muda(id string, ligar bool) error {
	switch {
	case strings.HasPrefix(id, prefixoAtalho):
		return mudaAtalho(strings.TrimPrefix(id, prefixoAtalho), ligar)
	case strings.HasPrefix(id, prefixoUnidade):
		return mudaUnidade(strings.TrimPrefix(id, prefixoUnidade), ligar)
	case strings.HasPrefix(id, prefixoMk):
		return fmt.Errorf("%s faz parte do Mk e não dá para mudar por aqui", id)
	}
	return fmt.Errorf("não conheço o item %q: o id começa por %s, %s ou %s", id, prefixoAtalho, prefixoUnidade, prefixoMk)
}

// ---- unidades de usuário ----

// pastaDasUnidades é onde o `systemctl --user enable` põe os atalhos de
// quem a própria pessoa ligou.
func pastaDasUnidades() string { return base.Home(".config", "systemd", "user") }

// ligadaPeloUsuario diz se a unidade foi habilitada por esta conta (há um
// link dela numa pasta .wants ou .requires da home). As outras habilitadas
// foram ligadas para todos pelo sistema, e o `--user disable` não as alcança.
func ligadaPeloUsuario(unidade string) bool {
	pastas, _ := os.ReadDir(pastaDasUnidades())
	for _, p := range pastas {
		nome := p.Name()
		if !strings.HasSuffix(nome, ".wants") && !strings.HasSuffix(nome, ".requires") && !strings.HasSuffix(nome, ".upholds") {
			continue
		}
		if _, err := os.Lstat(filepath.Join(pastaDasUnidades(), nome, unidade)); err == nil {
			return true
		}
	}
	return false
}

func listaUnidades() []Item {
	saida, err := servicos.Systemctl(true, "list-unit-files", "--state=enabled", "--type=service,socket,timer", "--output=json", "--no-pager")
	if err != nil {
		return nil
	}
	var arquivos []struct {
		UnitFile string `json:"unit_file"`
	}
	if json.Unmarshal(saida, &arquivos) != nil {
		return nil
	}
	var nomes []string
	for _, a := range arquivos {
		if servicos.ValidaUnidade(a.UnitFile) == nil && !strings.Contains(a.UnitFile, "@.") {
			nomes = append(nomes, a.UnitFile)
		}
	}
	sort.Strings(nomes)
	fichas, err := servicos.Mostra(true, []string{"Id", "Description", "ExecStart"}, nomes)
	if err != nil {
		fichas = make([]map[string]string, len(nomes))
	}
	itens := make([]Item, 0, len(nomes))
	for i, nome := range nomes {
		doUsuario := ligadaPeloUsuario(nome)
		item := Item{ID: prefixoUnidade + nome, Origem: origemUnidade, Nome: fichas[i]["Description"],
			Comando: comandoDoExecStart(fichas[i]["ExecStart"]), Ligado: true, DoSistema: !doUsuario, Mudavel: doUsuario}
		if item.Nome == "" {
			item.Nome = nome
		}
		itens = append(itens, item)
	}
	return itens
}

// comandoDoExecStart tira a linha de comando do que o `systemctl show`
// devolve: "{ path=/usr/bin/x ; argv[]=/usr/bin/x serve ; ignore_errors=no ; … }".
func comandoDoExecStart(valor string) string {
	_, resto, ok := strings.Cut(valor, "argv[]=")
	if !ok {
		return ""
	}
	comando, _, _ := strings.Cut(resto, " ; ")
	return strings.TrimSpace(comando)
}

func mudaUnidade(unidade string, ligar bool) error {
	if err := servicos.ValidaUnidade(unidade); err != nil {
		return err
	}
	verbo, oque := "enable", "ligar"
	if !ligar {
		verbo, oque = "disable", "desligar"
		if !ligadaPeloUsuario(unidade) {
			return fmt.Errorf("%s não foi ligada por esta conta (é o sistema que a liga para todos): não dá para desligar por aqui", unidade)
		}
	}
	if _, err := servicos.Systemctl(true, verbo, "--", unidade); err != nil {
		return fmt.Errorf("não deu para %s %s: %v", oque, unidade, err)
	}
	return nil
}

// ---- o que o Mk sobe ----

// Os processos do autostart.lua que o MkPulso sabe apresentar. Vale o
// primeiro cujo trecho aparece na linha de comando.
var conhecidosDoMk = []struct{ trecho, id, nome string }{
	{"mare-start", "mare", "Maré (a ilha e os painéis)"},
	{"hypridle", "hypridle", "Hypridle (apaga a tela e suspende sozinho)"},
	{"wl-clip-persist", "wl-clip-persist", "Área de transferência (guarda o que foi copiado)"},
	{"--type text --watch cliphist", "cliphist-texto", "Histórico da área de transferência (texto)"},
	{"--type image --watch cliphist", "cliphist-imagem", "Histórico da área de transferência (imagens)"},
	{"mare-look", "mare-look", "Aparência dos apps (ícones, cursor e fontes)"},
	{"udiskie", "udiskie", "Udiskie (monta pendrives e discos)"},
	{"polkitagent", "polkit", "Agente de senhas (polkit)"},
	{"gnome-keyring-daemon", "chaveiro", "Chaveiro (senhas guardadas)"},
}

var (
	chamadaDeExec = regexp.MustCompile(`\bhl\.exec\w*\(`)
	scriptNoBash  = regexp.MustCompile(`^(?:ba)?sh '([^']+)'(.*)$`)
	foraDoId      = regexp.MustCompile(`[^a-z0-9._-]+`)
)

// listaMk lê (só lê) o autostart.lua do Mk e devolve uma linha por
// hl.exec_cmd(...). Sem o arquivo, nada.
func listaMk() []Item {
	b, err := os.ReadFile(base.Home(".config", "mk", "hypr", "mare", "autostart.lua"))
	if err != nil {
		return nil
	}
	var itens []Item
	usados := map[string]int{}
	for _, linha := range strings.Split(string(b), "\n") {
		linha = strings.TrimSpace(linha)
		if strings.HasPrefix(linha, "--") {
			continue
		}
		onde := chamadaDeExec.FindStringIndex(linha)
		if onde == nil {
			continue
		}
		comando, ok := textoDoLua(linha[onde[1]:])
		if !ok {
			continue
		}
		id, nome := "", ""
		for _, c := range conhecidosDoMk {
			if strings.Contains(comando, c.trecho) {
				id, nome = c.id, c.nome
				break
			}
		}
		// "bash '…/scripts/mare-start'" é, para quem lê, "mare-start".
		if m := scriptNoBash.FindStringSubmatch(comando); m != nil {
			comando = filepath.Base(m[1]) + m[2]
		}
		if id == "" {
			programa := comando
			if campos := strings.Fields(comando); len(campos) > 0 {
				programa = filepath.Base(strings.Trim(campos[0], `'"`))
			}
			nome = programa
			id = strings.Trim(foraDoId.ReplaceAllString(strings.ToLower(programa), "-"), "-")
			if id == "" {
				id = "processo"
			}
		}
		usados[id]++
		if n := usados[id]; n > 1 {
			id = fmt.Sprintf("%s-%d", id, n)
		}
		itens = append(itens, Item{ID: prefixoMk + id, Origem: origemMk, Nome: nome, Comando: comando,
			Ligado: true, DoSistema: true, Mudavel: false})
	}
	return itens
}

// textoDoLua lê o argumento de uma chamada Lua (o que vem depois do "(") e
// junta os pedaços de texto; o que é variável vira "…". Devolve false se a
// chamada não fecha na linha ou não tem texto nenhum.
func textoDoLua(s string) (string, bool) {
	var texto strings.Builder
	temTexto := false
	variavel := func() {
		if !strings.HasSuffix(texto.String(), "…") {
			texto.WriteString("…")
		}
	}
	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c == ' ' || c == '\t':
			i++
		case c == ')':
			return texto.String(), temTexto
		case c == '"' || c == '\'':
			fim := -1
			for j := i + 1; j < len(s); j++ {
				if s[j] == '\\' && j+1 < len(s) {
					j++
					texto.WriteByte(s[j])
					continue
				}
				if s[j] == c {
					fim = j
					break
				}
				texto.WriteByte(s[j])
			}
			if fim < 0 {
				return "", false
			}
			temTexto = true
			i = fim + 1
		case strings.HasPrefix(s[i:], ".."):
			i += 2
		default:
			// Uma variável ou uma conta: pula até o próximo "..", texto ou
			// o ")" que fecha a chamada.
			fundo := 0
			for i < len(s) {
				if s[i] == '(' {
					fundo++
				} else if s[i] == ')' {
					if fundo == 0 {
						break
					}
					fundo--
				} else if fundo == 0 && (strings.HasPrefix(s[i:], "..") || s[i] == '"' || s[i] == '\'') {
					break
				}
				i++
			}
			variavel()
		}
	}
	return "", false
}

var errNaoExiste = errors.New("não existe")
