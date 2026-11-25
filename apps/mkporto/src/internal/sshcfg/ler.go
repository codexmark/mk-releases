// Package sshcfg lê e escreve o ~/.ssh/config do jeito que o MkPorto precisa
// (../../../CONTRATO.md): um leitor próprio do formato (blocos Host, Include
// com glob, Match encerrando o bloco), a resolução de cada apelido com a regra
// do ssh (o primeiro valor achado ganha) e a gravação do arquivo do app. Nada
// aqui roda o ssh nem abre arquivo que não seja o config e os incluídos.
package sshcfg

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Até onde um Include pode descer.
const profundidadeMaxima = 16

// Linhas maiores que isto não são de um config de verdade.
const linhaMaxima = 64 * 1024

// Opcao é uma linha "Chave valor" de um bloco: a chave em minúsculas.
type Opcao struct {
	Chave string
	Args  []string
}

// Bloco é um trecho do config que vale para os mesmos padrões: um "Host …"
// (Declarado), a continuação dele depois de um Include, o que vem antes do
// primeiro Host (Global) ou o que está sob um Match (que não vale para
// ninguém: o motor não avalia condições).
type Bloco struct {
	Padroes   []string
	Arquivo   string
	Declarado bool
	Global    bool
	Match     bool
	Opcoes    []Opcao
}

// Leitura é o config inteiro, com os incluídos, na ordem em que o ssh o lê.
type Leitura struct {
	Blocos []Bloco
}

// Host é uma máquina do catálogo. Texto vazio quer dizer "não tem".
type Host struct {
	Apelido string
	Host    string
	Usuario string
	Porta   int
	Chave   string
	Salto   string
	Origem  string // "mkporto" ou "config"
	Arquivo string
}

// ArquivoDoApp é o único arquivo em que o MkPorto escreve.
func ArquivoDoApp(dir string) string {
	return filepath.Join(dir, "config.d", "mkporto.conf")
}

// Le lê dir/config e tudo o que ele inclui. `casa` é a home, para o "~/" dos
// Include. Arquivo que não existe ou não abre é como se estivesse vazio.
func Le(dir, casa string) *Leitura {
	l := &Leitura{}
	l.arquivo(filepath.Join(dir, "config"), dir, casa, Bloco{Global: true}, nil)
	return l
}

// LeSo lê um arquivo só, sem seguir os Include dele.
func LeSo(caminho string) *Leitura {
	l := &Leitura{}
	l.arquivo(caminho, "", "", Bloco{Global: true}, nil)
	return l
}

// arquivo lê um arquivo. `herdado` diz para quem valem as linhas antes do
// primeiro Host dele (num Include dentro de um Host, só para aquele host);
// `pilha` são os arquivos abertos no caminho até aqui, para não andar em laço.
// Com dir vazio os Include não são seguidos.
func (l *Leitura) arquivo(caminho, dir, casa string, herdado Bloco, pilha []string) {
	if len(pilha) >= profundidadeMaxima {
		return
	}
	real := caminho
	if r, err := filepath.EvalSymlinks(caminho); err == nil {
		real = r
	}
	for _, p := range pilha {
		if p == real {
			return
		}
	}
	info, err := os.Stat(caminho)
	if err != nil || !info.Mode().IsRegular() {
		return
	}
	f, err := os.Open(caminho)
	if err != nil {
		return
	}
	defer f.Close()
	pilha = append(pilha[:len(pilha):len(pilha)], real)

	continua := func(modelo Bloco) *Bloco {
		l.Blocos = append(l.Blocos, Bloco{Padroes: modelo.Padroes, Arquivo: caminho, Global: modelo.Global, Match: modelo.Match})
		return &l.Blocos[len(l.Blocos)-1]
	}
	atual := continua(herdado)

	leitor := bufio.NewScanner(f)
	leitor.Buffer(make([]byte, 0, 4096), linhaMaxima)
	for leitor.Scan() {
		chave, args := palavras(leitor.Text())
		switch chave {
		case "":
			continue
		case "host":
			l.Blocos = append(l.Blocos, Bloco{Padroes: args, Arquivo: caminho, Declarado: true})
			atual = &l.Blocos[len(l.Blocos)-1]
		case "match":
			l.Blocos = append(l.Blocos, Bloco{Arquivo: caminho, Match: true})
			atual = &l.Blocos[len(l.Blocos)-1]
		case "include":
			if atual.Match || dir == "" {
				continue
			}
			modelo := *atual
			for _, padrao := range args {
				for _, incluido := range expande(padrao, dir, casa) {
					l.arquivo(incluido, dir, casa, modelo, pilha)
				}
			}
			// O que vem depois do Include continua valendo para os mesmos
			// padrões, mas depois do que o incluído disse.
			atual = continua(modelo)
		default:
			atual.Opcoes = append(atual.Opcoes, Opcao{Chave: chave, Args: args})
		}
	}
}

// expande resolve o caminho de um Include: "~/" é a home, o relativo é a
// partir da pasta do ssh, e o glob vira os arquivos que existem, em ordem.
func expande(padrao, dir, casa string) []string {
	padrao = Resolve(padrao, dir, casa)
	if padrao == "" {
		return nil
	}
	achados, err := filepath.Glob(padrao)
	if err != nil {
		return nil
	}
	return achados
}

// Resolve transforma o caminho de um Include num caminho absoluto (ainda com
// o glob, se houver).
func Resolve(caminho, dir, casa string) string {
	switch {
	case caminho == "":
		return ""
	case strings.HasPrefix(caminho, "~/"):
		if casa == "" {
			return ""
		}
		return filepath.Join(casa, caminho[2:])
	case filepath.IsAbs(caminho):
		return filepath.Clean(caminho)
	}
	return filepath.Join(dir, caminho)
}

// palavras separa uma linha do config na palavra-chave (em minúsculas) e nos
// argumentos. Aceita "Chave valor" e "Chave=valor", aspas duplas em volta de
// um argumento e um comentário (#) no começo da linha ou de um argumento.
// Linha vazia ou só de comentário devolve a chave vazia.
func palavras(linha string) (chave string, args []string) {
	linha = strings.TrimSpace(linha)
	if linha == "" || linha[0] == '#' {
		return "", nil
	}
	fim := strings.IndexAny(linha, " \t=")
	if fim < 0 {
		return strings.ToLower(linha), nil
	}
	chave = strings.ToLower(linha[:fim])
	resto := strings.TrimLeft(linha[fim:], " \t")
	if strings.HasPrefix(resto, "=") {
		resto = resto[1:]
	}
	var atual strings.Builder
	dentro, tem := false, false
	solta := func() {
		if tem {
			args = append(args, atual.String())
		}
		atual.Reset()
		tem = false
	}
	for _, c := range resto {
		switch {
		case c == '"':
			dentro = !dentro
			tem = true
		case dentro:
			atual.WriteRune(c)
		case c == ' ' || c == '\t':
			solta()
		case c == '#' && !tem:
			return chave, args
		default:
			atual.WriteRune(c)
			tem = true
		}
	}
	solta()
	return chave, args
}

// Curinga diz se um padrão de Host não é o nome de uma máquina.
func Curinga(padrao string) bool {
	return strings.ContainsAny(padrao, "*?!")
}

// casa diz se um nome casa com os padrões de um Host, como no ssh: basta um
// padrão casar, e um negado (!x) que casa derruba todos.
func casa(padroes []string, nome string) bool {
	achou := false
	for _, p := range padroes {
		if strings.HasPrefix(p, "!") {
			if casaUm(p[1:], nome) {
				return false
			}
			continue
		}
		if casaUm(p, nome) {
			achou = true
		}
	}
	return achou
}

// casaUm compara um nome com um padrão: * é qualquer trecho, ? uma letra.
func casaUm(padrao, nome string) bool {
	p, n := []rune(padrao), []rune(nome)
	pi, ni, estrela, marca := 0, 0, -1, 0
	for ni < len(n) {
		switch {
		case pi < len(p) && p[pi] == '*':
			estrela, marca = pi, ni
			pi++
		case pi < len(p) && (p[pi] == '?' || p[pi] == n[ni]):
			pi++
			ni++
		case estrela >= 0:
			pi = estrela + 1
			marca++
			ni = marca
		default:
			return false
		}
	}
	for pi < len(p) && p[pi] == '*' {
		pi++
	}
	return pi == len(p)
}

// vale diz se o bloco se aplica a um apelido.
func (b *Bloco) vale(apelido string) bool {
	if b.Match {
		return false
	}
	return b.Global || casa(b.Padroes, apelido)
}

// Hosts devolve as máquinas: cada apelido sem curinga declarado num Host, uma
// vez só, na ordem em que aparecem, com os valores resolvidos.
func (l *Leitura) Hosts(arquivoDoApp string) []Host {
	hosts := []Host{}
	visto := map[string]bool{}
	for i := range l.Blocos {
		b := &l.Blocos[i]
		if !b.Declarado {
			continue
		}
		for _, apelido := range b.Padroes {
			if Curinga(apelido) || visto[apelido] {
				continue
			}
			visto[apelido] = true
			h := l.resolve(apelido)
			h.Arquivo = b.Arquivo
			h.Origem = "config"
			if b.Arquivo == arquivoDoApp {
				h.Origem = "mkporto"
			}
			hosts = append(hosts, h)
		}
	}
	return hosts
}

// resolve junta as opções de todos os blocos que valem para o apelido: o
// primeiro valor achado ganha.
func (l *Leitura) resolve(apelido string) Host {
	h := Host{Apelido: apelido}
	achado := map[string]string{}
	for i := range l.Blocos {
		b := &l.Blocos[i]
		if !b.vale(apelido) {
			continue
		}
		for _, o := range b.Opcoes {
			if _, ja := achado[o.Chave]; ja || len(o.Args) == 0 {
				continue
			}
			achado[o.Chave] = o.Args[0]
		}
	}
	h.Host = apelido
	if nome := achado["hostname"]; nome != "" {
		h.Host = strings.NewReplacer("%%", "%", "%h", apelido).Replace(nome)
	}
	h.Usuario = achado["user"]
	h.Porta = 22
	if p, err := strconv.Atoi(achado["port"]); err == nil && p >= 1 && p <= 65535 {
		h.Porta = p
	}
	h.Chave = achado["identityfile"]
	if s := achado["proxyjump"]; !strings.EqualFold(s, "none") {
		h.Salto = s
	}
	return h
}

// DeclaradoFora diz em que arquivo (que não é o do app) o apelido está
// declarado num Host; vazio se em nenhum.
func (l *Leitura) DeclaradoFora(apelido, arquivoDoApp string) string {
	for i := range l.Blocos {
		b := &l.Blocos[i]
		if !b.Declarado || b.Arquivo == arquivoDoApp {
			continue
		}
		for _, p := range b.Padroes {
			if p == apelido {
				return b.Arquivo
			}
		}
	}
	return ""
}
