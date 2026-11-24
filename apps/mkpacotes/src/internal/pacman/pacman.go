// Package pacman fala com o pacman (e com o checkupdates) para o que é dos
// repositórios: de onde é cada pacote, a busca, a ficha, o ensaio de uma
// instalação ou remoção e o que há para atualizar. Só leitura, sem root. Os
// comandos são achados no PATH (os testes põem dublês) e rodam com LC_ALL=C,
// para a saída ser sempre a mesma.
package pacman

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"mkpacotes/internal/alpm"
)

// Quanto esperar por uma leitura.
const (
	prazoDeLeitura = 30 * time.Second
	prazoDaRede    = 90 * time.Second // o checkupdates baixa as bases
)

// Falhou é um comando que saiu com erro: a frase e o código de saída.
type Falhou struct {
	Frase string
	Cod   int
}

func (f *Falhou) Error() string { return f.Frase }

// Codigo devolve o código de saída guardado num erro do Roda (-1 se o erro
// não é de um comando que rodou).
func Codigo(err error) int {
	var f *Falhou
	if errors.As(err, &f) {
		return f.Cod
	}
	return -1
}

// Roda executa um comando externo com LC_ALL=C e devolve o stdout. O erro já
// vem como frase: o comando não existe, demorou demais, ou o que ele disse no
// stderr (a primeira linha).
func Roda(nome string, prazo time.Duration, args ...string) ([]byte, error) {
	caminho, err := exec.LookPath(nome)
	if err != nil {
		return nil, fmt.Errorf("não achei o %s nesta máquina", nome)
	}
	ctx, cancela := context.WithTimeout(context.Background(), prazo)
	defer cancela()
	var saida, erros bytes.Buffer
	cmd := exec.CommandContext(ctx, caminho, args...)
	cmd.Env = AmbienteC()
	cmd.Stdout, cmd.Stderr = &saida, &erros
	cmd.WaitDelay = time.Second
	err = cmd.Run()
	if err == nil {
		return saida.Bytes(), nil
	}
	if ctx.Err() != nil {
		return saida.Bytes(), fmt.Errorf("o %s demorou demais e foi interrompido", nome)
	}
	var saiu *exec.ExitError
	if !errors.As(err, &saiu) {
		return saida.Bytes(), fmt.Errorf("não consegui rodar o %s: %v", nome, err)
	}
	frase := Resumo(erros.String())
	if frase == "" {
		frase = fmt.Sprintf("o %s saiu com o código %d", nome, saiu.ExitCode())
	}
	return saida.Bytes(), &Falhou{Frase: frase, Cod: saiu.ExitCode()}
}

// AmbienteC é o ambiente de quem roda, sem as variáveis de idioma e com
// LC_ALL=C.
func AmbienteC() []string {
	var amb []string
	for _, v := range os.Environ() {
		if strings.HasPrefix(v, "LC_") || strings.HasPrefix(v, "LANG=") || strings.HasPrefix(v, "LANGUAGE=") {
			continue
		}
		amb = append(amb, v)
	}
	return append(amb, "LC_ALL=C")
}

// Resumo fica com a primeira linha não vazia de um stderr, sem espaços
// repetidos e sem passar de 300 letras.
func Resumo(stderr string) string {
	for _, linha := range strings.Split(stderr, "\n") {
		linha = strings.Join(strings.Fields(linha), " ")
		if linha == "" {
			continue
		}
		if letras := []rune(linha); len(letras) > 300 {
			linha = string(letras[:300]) + "…"
		}
		return linha
	}
	return ""
}

// Sinc é um pacote numa base sincronizada: em que repositório e em que versão.
type Sinc struct {
	Repo, Versao string
}

// Repos devolve tudo o que os repositórios sincronizados têm (`pacman -Sl`).
// Com o mesmo nome em dois, vale o primeiro (a ordem do pacman.conf).
func Repos() (map[string]Sinc, error) {
	saida, err := Roda("pacman", prazoDeLeitura, "-Sl")
	if err != nil {
		return nil, err
	}
	repos := map[string]Sinc{}
	for _, linha := range strings.Split(string(saida), "\n") {
		c := strings.Fields(linha)
		if len(c) < 3 {
			continue
		}
		if _, ja := repos[c[1]]; !ja {
			repos[c[1]] = Sinc{Repo: c[0], Versao: c[2]}
		}
	}
	return repos, nil
}

// Achado é um resultado da busca nos repositórios.
type Achado struct {
	Nome, Versao, Descricao, Repo string
}

// Busca procura nos repositórios (`pacman -Ss`): todas as palavras têm de
// aparecer no nome ou na descrição. Nada achado é lista vazia, sem erro.
func Busca(palavras []string) ([]Achado, error) {
	saida, err := Roda("pacman", prazoDeLeitura, append([]string{"-Ss", "--"}, palavras...)...)
	if err != nil && !(Codigo(err) == 1 && strings.Contains(err.Error(), "saiu com o código")) {
		return nil, err
	}
	var achados []Achado
	for _, linha := range strings.Split(string(saida), "\n") {
		if linha == "" {
			continue
		}
		if linha[0] == ' ' || linha[0] == '\t' {
			if n := len(achados); n > 0 && achados[n-1].Descricao == "" {
				achados[n-1].Descricao = strings.TrimSpace(linha)
			}
			continue
		}
		c := strings.Fields(linha)
		repo, nome, ok := strings.Cut(c[0], "/")
		if !ok || len(c) < 2 {
			continue
		}
		achados = append(achados, Achado{Nome: nome, Versao: c[1], Repo: repo})
	}
	return achados, nil
}

// Ficha é o que o `pacman -Si` diz de um pacote do repositório.
type Ficha struct {
	Repo, Nome, Versao, Descricao string
	URL, Empacotador              string
	Licencas, Depende, Opcionais  []string
	Fornece, Conflita             []string
	Baixar, Tamanho               *int64 // bytes
	ConstruidoEm                  *int64
}

// Info devolve a ficha de um pacote dos repositórios, ou nil se nenhum tem.
func Info(nome string) (*Ficha, error) {
	if err := alpm.ValidaNome(nome); err != nil {
		return nil, err
	}
	saida, err := Roda("pacman", prazoDeLeitura, "-Si", "--", nome)
	if err != nil {
		if strings.Contains(err.Error(), "was not found") {
			return nil, nil
		}
		return nil, err
	}
	campos := map[string][]string{}
	chave := ""
	for _, linha := range strings.Split(string(saida), "\n") {
		if strings.TrimSpace(linha) == "" {
			if len(campos) > 0 {
				break // o mesmo pacote em outro repositório: vale o primeiro
			}
			continue
		}
		if linha[0] == ' ' {
			if chave != "" {
				campos[chave] = append(campos[chave], strings.TrimSpace(linha))
			}
			continue
		}
		k, v, ok := strings.Cut(linha, " : ")
		if !ok {
			continue
		}
		chave = strings.TrimSpace(k)
		campos[chave] = []string{strings.TrimSpace(v)}
	}
	um := func(k string) string {
		if v := campos[k]; len(v) > 0 && v[0] != "None" {
			return v[0]
		}
		return ""
	}
	lista := func(k string) []string {
		if um(k) == "" {
			return nil
		}
		return strings.Fields(um(k))
	}
	f := &Ficha{Repo: um("Repository"), Nome: um("Name"), Versao: um("Version"), Descricao: um("Description"),
		URL: um("URL"), Empacotador: um("Packager"), Licencas: lista("Licenses"), Depende: lista("Depends On"),
		Fornece: lista("Provides"), Conflita: lista("Conflicts With"),
		Baixar: bytesDe(um("Download Size")), Tamanho: bytesDe(um("Installed Size"))}
	if um("Optional Deps") != "" {
		for _, o := range campos["Optional Deps"] {
			f.Opcionais = append(f.Opcionais, strings.TrimSpace(strings.TrimSuffix(o, "[installed]")))
		}
	}
	if quando, err := time.ParseInLocation("Mon Jan _2 15:04:05 2006", um("Build Date"), time.Local); err == nil {
		s := quando.Unix()
		f.ConstruidoEm = &s
	}
	if f.Nome == "" {
		return nil, nil
	}
	return f, nil
}

// bytesDe lê um tamanho do pacman ("1954.93 KiB") em bytes; nil se não der.
func bytesDe(texto string) *int64 {
	c := strings.Fields(texto)
	if len(c) != 2 {
		return nil
	}
	v, err := strconv.ParseFloat(c[0], 64)
	fator, ok := map[string]float64{"B": 1, "KiB": 1 << 10, "MiB": 1 << 20, "GiB": 1 << 30, "TiB": 1 << 40}[c[1]]
	if err != nil || !ok || v < 0 {
		return nil
	}
	b := int64(v*fator + 0.5)
	return &b
}

// Item é uma linha de um ensaio: o pacote que entraria ou sairia.
type Item struct {
	Nome, Versao, Repo string
	Bytes              int64 // a baixar (instalação)
}

func validaTodos(nomes []string) error {
	if len(nomes) == 0 {
		return errors.New("faltou dizer o pacote")
	}
	for _, n := range nomes {
		if err := alpm.ValidaNome(n); err != nil {
			return err
		}
	}
	return nil
}

// EnsaioInstalar pergunta ao pacman o que uma instalação traria (os pedidos
// e as dependências), sem instalar: `pacman -S --needed --print`. O que já
// está instalado e em dia não vem.
func EnsaioInstalar(nomes []string) ([]Item, error) {
	if err := validaTodos(nomes); err != nil {
		return nil, err
	}
	args := append([]string{"-S", "--needed", "--noconfirm", "--print", "--print-format", "%n %v %r %s", "--"}, nomes...)
	saida, err := Roda("pacman", prazoDeLeitura, args...)
	if err != nil {
		return nil, err
	}
	var itens []Item
	for _, linha := range strings.Split(string(saida), "\n") {
		c := strings.Fields(linha)
		if len(c) != 4 || alpm.ValidaNome(c[0]) != nil {
			continue
		}
		b, err := strconv.ParseInt(c[3], 10, 64)
		if err != nil {
			continue
		}
		itens = append(itens, Item{Nome: c[0], Versao: c[1], Repo: c[2], Bytes: b})
	}
	return itens, nil
}

// EnsaioRemover pergunta ao pacman o que uma remoção levaria: os pedidos e as
// dependências que ficariam sem dono (`pacman -Rs --print`; o -n da remoção de
// verdade não muda a lista e o pacman não o aceita com --print).
func EnsaioRemover(nomes []string) ([]Item, error) {
	if err := validaTodos(nomes); err != nil {
		return nil, err
	}
	args := append([]string{"-Rs", "--noconfirm", "--print", "--print-format", "%n %v", "--"}, nomes...)
	saida, err := Roda("pacman", prazoDeLeitura, args...)
	if err != nil {
		return nil, err
	}
	var itens []Item
	for _, linha := range strings.Split(string(saida), "\n") {
		c := strings.Fields(linha)
		if len(c) != 2 || alpm.ValidaNome(c[0]) != nil {
			continue
		}
		itens = append(itens, Item{Nome: c[0], Versao: c[1]})
	}
	return itens, nil
}

// Mudanca é uma atualização à espera: de que versão para qual.
type Mudanca struct {
	Nome, De, Para string
}

// Atualizacoes devolve o que há para atualizar dos repositórios e de onde
// veio a resposta: "checkupdates" (sincroniza numa base à parte, sem root) ou
// "pacman" (`pacman -Qu`: o que a última sincronização já sabia), quando o
// checkupdates não existe ou falhou.
func Atualizacoes() ([]Mudanca, string, error) {
	if _, err := exec.LookPath("checkupdates"); err == nil {
		saida, err := Roda("checkupdates", prazoDaRede)
		if err == nil || Codigo(err) == 2 { // 2: nada para atualizar
			return mudancas(saida), "checkupdates", nil
		}
	}
	saida, err := Roda("pacman", prazoDeLeitura, "-Qu")
	if err != nil && !(Codigo(err) == 1 && strings.Contains(err.Error(), "saiu com o código")) {
		return nil, "pacman", err
	}
	return mudancas(saida), "pacman", nil
}

func mudancas(saida []byte) []Mudanca {
	var lista []Mudanca
	for _, linha := range strings.Split(string(saida), "\n") {
		c := strings.Fields(linha)
		if len(c) < 4 || c[2] != "->" {
			continue
		}
		lista = append(lista, Mudanca{Nome: c[0], De: c[1], Para: c[3]})
	}
	return lista
}
