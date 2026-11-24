// Package alpm lê a base local do pacman direto dos arquivos
// (/var/lib/pacman/local/<nome>-<versão>/desc e files), sem chamar o pacman:
// é só leitura de arquivo (milissegundos para mil e tantos pacotes), não
// depende do idioma da saída do `pacman -Qi` e funciona com o pacman travado.
// Também tem o que é regra do pacman e não precisa dele: nome de pacote
// válido, comparação de versões (vercmp), quem depende de quem e os órfãos.
package alpm

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"mkpacotes/internal/base"
)

// Pacote é um pacote instalado, como o desc dele diz.
type Pacote struct {
	Nome, Versao, Descricao string
	URL, Empacotador        string
	Tamanho                 int64  // bytes instalados
	InstaladoEm             *int64 // segundos da época; nil se o desc não diz
	ConstruidoEm            *int64
	Pedido                  bool // a pessoa pediu (false: veio como dependência)
	Licencas                []string
	Depende                 []string // como no desc: "glibc>=2.38"
	Opcionais               []string // "nome: para quê"
	Fornece                 []string
	Conflita                []string

	pasta string
}

// Base é o conjunto dos instalados, com os índices de quem depende de quem.
type Base struct {
	Pacotes map[string]*Pacote

	fornecedores map[string][]string // nome fornecido -> quem fornece
	pedidoPor    map[string][]string // pacote -> quem depende dele
	opcionalDe   map[string][]string // pacote -> de quem ele é opcional
}

// Le lê a base local inteira. Pasta de pacote sem desc legível é ignorada.
func Le() (*Base, error) {
	raiz := base.PacmanDB("local")
	entradas, err := os.ReadDir(raiz)
	if err != nil {
		return nil, fmt.Errorf("não achei a base do pacman em %s", raiz)
	}
	b := &Base{Pacotes: map[string]*Pacote{}}
	for _, e := range entradas {
		if !e.IsDir() {
			continue
		}
		pasta := filepath.Join(raiz, e.Name())
		p, err := leDesc(filepath.Join(pasta, "desc"))
		if err != nil || p.Nome == "" {
			continue
		}
		p.pasta = pasta
		b.Pacotes[p.Nome] = p
	}
	b.indexa()
	return b, nil
}

// leDesc lê um desc: blocos "%CHAVE%", uma linha por valor, linha vazia no fim.
func leDesc(caminho string) (*Pacote, error) {
	campos, err := leBlocos(caminho, nil)
	if err != nil {
		return nil, err
	}
	um := func(chave string) string {
		if v := campos[chave]; len(v) > 0 {
			return v[0]
		}
		return ""
	}
	data := func(chave string) *int64 {
		n, err := strconv.ParseInt(um(chave), 10, 64)
		if err != nil || n <= 0 {
			return nil
		}
		return &n
	}
	p := &Pacote{
		Nome: um("NAME"), Versao: um("VERSION"), Descricao: um("DESC"), URL: um("URL"), Empacotador: um("PACKAGER"),
		InstaladoEm: data("INSTALLDATE"), ConstruidoEm: data("BUILDDATE"), Pedido: um("REASON") != "1",
		Licencas: campos["LICENSE"], Depende: campos["DEPENDS"], Opcionais: campos["OPTDEPENDS"],
		Fornece: campos["PROVIDES"], Conflita: campos["CONFLICTS"],
	}
	p.Tamanho, _ = strconv.ParseInt(um("SIZE"), 10, 64)
	return p, nil
}

// leBlocos lê um arquivo no formato do pacman (desc, files). Com `so`, guarda
// só essas chaves.
func leBlocos(caminho string, so map[string]bool) (map[string][]string, error) {
	f, err := os.Open(caminho)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	campos := map[string][]string{}
	chave := ""
	linhas := bufio.NewScanner(f)
	linhas.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for linhas.Scan() {
		l := linhas.Text()
		switch {
		case l == "":
			chave = ""
		case chave == "" && len(l) > 2 && l[0] == '%' && l[len(l)-1] == '%':
			chave = l[1 : len(l)-1]
		case chave != "" && (so == nil || so[chave]):
			campos[chave] = append(campos[chave], l)
		}
	}
	return campos, linhas.Err()
}

func (b *Base) indexa() {
	b.fornecedores = map[string][]string{}
	b.pedidoPor = map[string][]string{}
	b.opcionalDe = map[string][]string{}
	for _, p := range b.Pacotes {
		for _, f := range p.Fornece {
			b.fornecedores[NomeDe(f)] = append(b.fornecedores[NomeDe(f)], p.Nome)
		}
	}
	junta := func(indice map[string][]string, alvo, quem string) {
		if alvo == quem {
			return
		}
		for _, q := range indice[alvo] {
			if q == quem {
				return
			}
		}
		indice[alvo] = append(indice[alvo], quem)
	}
	for _, p := range b.Pacotes {
		for _, d := range p.Depende {
			for _, alvo := range b.quemAtende(NomeDe(d)) {
				junta(b.pedidoPor, alvo, p.Nome)
			}
		}
		for _, d := range p.Opcionais {
			for _, alvo := range b.quemAtende(NomeDe(d)) {
				junta(b.opcionalDe, alvo, p.Nome)
			}
		}
	}
	for _, indice := range []map[string][]string{b.pedidoPor, b.opcionalDe} {
		for _, lista := range indice {
			sort.Strings(lista)
		}
	}
}

// quemAtende: os instalados que satisfazem um nome: o próprio pacote e os
// que o fornecem.
func (b *Base) quemAtende(nome string) []string {
	var quem []string
	if _, ok := b.Pacotes[nome]; ok {
		quem = append(quem, nome)
	}
	return append(quem, b.fornecedores[nome]...)
}

// Nomes devolve os nomes dos instalados, em ordem.
func (b *Base) Nomes() []string {
	nomes := make([]string, 0, len(b.Pacotes))
	for n := range b.Pacotes {
		nomes = append(nomes, n)
	}
	sort.Strings(nomes)
	return nomes
}

// Satisfeito diz se há um instalado com o nome da dependência ou que o
// fornece. (A restrição de versão não é conferida: quem instala é o pacman.)
func (b *Base) Satisfeito(dep string) bool {
	return len(b.quemAtende(NomeDe(dep))) > 0
}

// RequeridoPor devolve os instalados que dependem do pacote, em ordem.
func (b *Base) RequeridoPor(nome string) []string { return b.pedidoPor[nome] }

// OpcionalPara devolve os instalados que têm o pacote como dependência opcional.
func (b *Base) OpcionalPara(nome string) []string { return b.opcionalDe[nome] }

// Orfao: veio como dependência e nenhum instalado depende dele, nem como
// opcional. É a conta do `pacman -Qdt` (conferida com ele numa máquina de
// verdade): o que algum pacote ainda lista como opcional não é sobra.
func (b *Base) Orfao(nome string) bool {
	p, ok := b.Pacotes[nome]
	return ok && !p.Pedido && len(b.pedidoPor[nome]) == 0 && len(b.opcionalDe[nome]) == 0
}

// Orfaos devolve os nomes de todos os órfãos, em ordem.
func (b *Base) Orfaos() []string {
	var nomes []string
	for _, n := range b.Nomes() {
		if b.Orfao(n) {
			nomes = append(nomes, n)
		}
	}
	return nomes
}

// Arquivos conta os arquivos de um instalado (pastas não contam) e devolve
// os que ficam em usr/bin/, com a barra na frente.
func (b *Base) Arquivos(nome string) (total int, executaveis []string, err error) {
	p, ok := b.Pacotes[nome]
	if !ok {
		return 0, nil, fmt.Errorf("%s não está instalado", nome)
	}
	campos, err := leBlocos(filepath.Join(p.pasta, "files"), map[string]bool{"FILES": true})
	if err != nil {
		return 0, nil, fmt.Errorf("não consegui ler a lista de arquivos de %s", nome)
	}
	for _, a := range campos["FILES"] {
		if strings.HasSuffix(a, "/") {
			continue
		}
		total++
		if strings.HasPrefix(a, "usr/bin/") {
			executaveis = append(executaveis, "/"+a)
		}
	}
	sort.Strings(executaveis)
	return total, executaveis, nil
}

// ValidaNome recusa o que não é nome de pacote antes de ele chegar a um
// comando: só minúsculas, números e @ . _ + -, sem começar com - (viraria
// opção) nem com . (viraria caminho).
func ValidaNome(nome string) error {
	ruim := nome == "" || len(nome) > 256 || nome[0] == '-' || nome[0] == '.'
	for i := 0; i < len(nome) && !ruim; i++ {
		c := nome[i]
		ruim = !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || strings.IndexByte("@._+-", c) >= 0)
	}
	if ruim {
		return fmt.Errorf("%q não é nome de pacote", nome)
	}
	return nil
}

// NomeDe tira de uma dependência a restrição de versão ("glibc>=2.38") e a
// explicação de uma opcional ("nome: para quê").
func NomeDe(dep string) string {
	if i := strings.IndexAny(dep, "<>=:"); i >= 0 {
		dep = dep[:i]
	}
	return strings.TrimSpace(dep)
}
