package compactado

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	errCancelada = errors.New("cancelada")
	errBomba     = errors.New("parece uma bomba de compressão: o conteúdo é grande demais para o tamanho do arquivo")
	errGrande    = errors.New("o arquivo é grande demais para abrir de dentro do compactado: extraia antes")
)

// A taxa só conta depois de tanto escrito: um zip de 1 KiB que vira 1 MiB não
// é uma bomba. Os testes diminuem.
var pisoDaTaxa int64 = 100 << 20

// limites são as travas contra bomba: o total extraído (MKFILES_EXTRAIR_MAX,
// bytes; padrão 20 GiB) e a taxa de compressão (MKFILES_EXTRAIR_TAXA; padrão
// 200 para 1).
func limites() (maximo, taxa int64) {
	maximo, taxa = 20<<30, 200
	if n, err := strconv.ParseInt(os.Getenv("MKFILES_EXTRAIR_MAX"), 10, 64); err == nil && n > 0 {
		maximo = n
	}
	if n, err := strconv.ParseInt(os.Getenv("MKFILES_EXTRAIR_TAXA"), 10, 64); err == nil && n > 0 {
		taxa = n
	}
	return maximo, taxa
}

// passaDoLimite diz se tantos bytes extraídos de um arquivo de tal tamanho
// parecem uma bomba.
func passaDoLimite(escritos, doArquivo int64) bool {
	maximo, taxa := limites()
	return escritos > maximo || (escritos > pisoDaTaxa && escritos > taxa*max(doArquivo, 1))
}

// andamento é o que o extrator precisa de uma tarefa (a *operar.Tarefa serve).
type andamento interface {
	Segue() bool
	Erro(caminho, frase string)
	Avanca(itens int, bytes int64, atual string)
}

// mudo é o andamento de quem não é tarefa (abrir um arquivo de dentro).
type mudo struct {
	parou  func() bool
	avisos []string
}

func (m *mudo) Segue() bool                { return m.parou == nil || !m.parou() }
func (m *mudo) Erro(caminho, frase string) { m.avisos = append(m.avisos, caminho+": "+frase) }
func (m *mudo) Avanca(int, int64, string)  {}
func (m *mudo) primeiro() string           { return strings.Join(m.avisos[:min(1, len(m.avisos))], "") }
func (m *mudo) temAviso() bool             { return len(m.avisos) > 0 }
func semAndamento(parou func() bool) *mudo { return &mudo{parou: parou} }

type pendente struct {
	rel  string
	e    entrada
	modo uint32
}

// extrator escreve as entradas de um compactado dentro de uma pasta que é só
// dele (raiz, criada vazia pelo motor), sem nunca sair de lá: cada pedaço do
// caminho é conferido (pasta de verdade, nunca link), o arquivo nasce com
// O_EXCL e O_NOFOLLOW, e os links só são criados no fim, quando nada mais vai
// ser escrito.
type extrator struct {
	raiz      string
	arquivo   string // o compactado (para os avisos)
	doArquivo int64  // o tamanho dele
	semLimite bool
	teto      int64 // se > 0, o máximo que se escreve (abrir um arquivo de dentro)
	a         andamento
	// quer diz se uma entrada entra e com que caminho (relativo à raiz). nil:
	// todas, com o próprio nome.
	quer func(nome string) (rel string, ok bool)

	escritos int64
	feitos   int
	pastas   []pendente
	links    []pendente
	duros    []pendente
}

func (x *extrator) aviso(e entrada, frase string) {
	nome := e.cru
	if len(nome) > 200 {
		nome = nome[:200]
	}
	x.a.Erro(x.arquivo, "«"+strings.ToValidUTF8(strings.ReplaceAll(nome, "\x00", "?"), "?")+"»: "+frase)
}

// lugar confere e prepara o caminho de uma entrada dentro da raiz: cria as
// pastas de cima que faltam e recusa passar por qualquer coisa que não seja
// uma pasta de verdade. Devolve o caminho absoluto.
func (x *extrator) lugar(rel string) (string, error) {
	partes := strings.Split(rel, "/")
	atual := x.raiz
	for _, p := range partes[:len(partes)-1] {
		atual = filepath.Join(atual, p)
		info, err := os.Lstat(atual)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			if err := os.Mkdir(atual, 0o700); err != nil {
				return "", err
			}
		case err != nil:
			return "", err
		case !info.IsDir():
			return "", errors.New("o caminho passa por algo que não é uma pasta")
		}
	}
	return filepath.Join(atual, partes[len(partes)-1]), nil
}

// livra tira o que já ocupa o lugar de uma entrada repetida (a última vale),
// a não ser que seja uma pasta.
func livra(destino string) error {
	info, err := os.Lstat(destino)
	if err != nil {
		return nil
	}
	if info.IsDir() {
		return errors.New("já há uma pasta com esse nome no arquivo")
	}
	return os.Remove(destino)
}

// recebe trata uma entrada.
func (x *extrator) recebe(e entrada, abre func() (io.ReadCloser, error)) error {
	if !x.a.Segue() {
		return errCancelada
	}
	if e.ruim != "" {
		x.aviso(e, e.ruim+": ficou de fora")
		return nil
	}
	rel := e.nome
	if x.quer != nil {
		var ok bool
		if rel, ok = x.quer(e.nome); !ok {
			return nil
		}
	}
	switch e.tipo {
	case eEspecial:
		x.aviso(e, "é um dispositivo ou um cano: ficou de fora")
		return nil
	case eLink:
		x.links = append(x.links, pendente{rel: rel, e: e})
		return nil
	case eDuro:
		x.duros = append(x.duros, pendente{rel: rel, e: e})
		return nil
	}
	destino, err := x.lugar(rel)
	if err != nil {
		x.aviso(e, err.Error()+": ficou de fora")
		return nil
	}
	if e.tipo == ePasta {
		if info, err := os.Lstat(destino); err == nil && !info.IsDir() {
			x.aviso(e, "já há um arquivo com esse nome: a pasta ficou de fora")
			return nil
		} else if err != nil {
			if err := os.Mkdir(destino, 0o700); err != nil {
				x.aviso(e, err.Error())
				return nil
			}
		}
		x.pastas = append(x.pastas, pendente{rel: rel, e: e})
		x.feitos++
		x.a.Avanca(1, 0, filepath.Join(x.arquivo, e.nome))
		return nil
	}
	if err := livra(destino); err != nil {
		x.aviso(e, err.Error()+": ficou de fora")
		return nil
	}
	if e.cifrada {
		return erroCifradoZip
	}
	corpo, err := abre()
	if err != nil {
		if e.cifrada {
			return erroCifradoZip
		}
		x.aviso(e, "não deu para ler: "+err.Error())
		return nil
	}
	defer corpo.Close()
	f, err := os.OpenFile(destino, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		x.aviso(e, err.Error())
		return nil
	}
	err = x.copia(f, corpo, filepath.Join(x.arquivo, e.nome))
	if e2 := f.Close(); err == nil {
		err = e2
	}
	if err != nil {
		os.Remove(destino)
		if err == errCancelada || err == errBomba || err == errGrande {
			return err
		}
		if errors.Is(err, syscall.ENOSPC) {
			return errors.New("o disco encheu")
		}
		x.aviso(e, "não deu para extrair: "+err.Error())
		return nil
	}
	// O dono sempre lê e escreve; setuid, setgid e sticky nunca passam.
	_ = os.Chmod(destino, fs.FileMode(e.modo&0o777|0o600))
	if !e.mtime.IsZero() {
		_ = os.Chtimes(destino, time.Now(), e.mtime)
	}
	x.feitos++
	return nil
}

// O bloco de cópia (os testes diminuem, para cancelar no meio).
var tamanhoDoBloco = 128 << 10

// aCadaBloco é um gancho de teste.
var aCadaBloco func()

func (x *extrator) copia(para io.Writer, de io.Reader, atual string) error {
	bloco := make([]byte, tamanhoDoBloco)
	primeiro := true
	for {
		if !x.a.Segue() {
			return errCancelada
		}
		n, err := de.Read(bloco)
		if n > 0 {
			x.escritos += int64(n)
			if x.teto > 0 && x.escritos > x.teto {
				return errGrande
			}
			if !x.semLimite && passaDoLimite(x.escritos, x.doArquivo) {
				return errBomba
			}
			if _, e := para.Write(bloco[:n]); e != nil {
				return e
			}
			itens := 0
			if primeiro {
				itens, primeiro = 1, false
			}
			x.a.Avanca(itens, int64(n), atual)
			if aCadaBloco != nil {
				aCadaBloco()
			}
		}
		if err == io.EOF {
			if primeiro {
				x.a.Avanca(1, 0, atual)
			}
			return nil
		}
		if err != nil {
			return err
		}
	}
}

// alvoDentro diz se o alvo de um link, lido a partir da pasta do link, fica
// dentro da raiz. Um alvo absoluto nunca fica.
func alvoDentro(rel, alvo string) bool {
	if alvo == "" || strings.HasPrefix(alvo, "/") || strings.ContainsRune(alvo, 0) {
		return false
	}
	fundo := strings.Count(rel, "/") // quantas pastas há acima do link, dentro da raiz
	for _, p := range strings.Split(alvo, "/") {
		switch p {
		case "", ".":
		case "..":
			fundo--
			if fundo < 0 {
				return false
			}
		default:
			fundo++
		}
	}
	return true
}

// termina cria o que ficou para o fim (hardlinks, depois links) e acerta as
// pastas. Depois disto nada mais é escrito na raiz.
func (x *extrator) termina() {
	for _, d := range x.duros {
		alvo, ruim := nomeLimpo(d.e.alvo)
		if ruim == "" && x.quer != nil {
			var ok bool
			if alvo, ok = x.quer(alvo); !ok {
				ruim = "o arquivo de origem não foi extraído"
			}
		}
		if ruim != "" {
			x.aviso(d.e, "hardlink para fora ("+ruim+"): ficou de fora")
			continue
		}
		origem, err := x.lugar(alvo)
		var destino string
		if err == nil {
			destino, err = x.lugar(d.rel)
		}
		if err == nil {
			var info fs.FileInfo
			if info, err = os.Lstat(origem); err == nil && !info.Mode().IsRegular() {
				err = errors.New("a origem do hardlink não é um arquivo")
			}
		}
		if err == nil {
			err = livra(destino)
		}
		if err == nil {
			err = os.Link(origem, destino)
		}
		if err != nil {
			x.aviso(d.e, "hardlink não criado: "+err.Error())
			continue
		}
		x.feitos++
	}
	for _, l := range x.links {
		if !alvoDentro(l.rel, l.e.alvo) {
			x.aviso(l.e, "link que aponta para fora da pasta: ficou de fora")
			continue
		}
		destino, err := x.lugar(l.rel)
		if err == nil {
			err = livra(destino)
		}
		if err == nil {
			err = os.Symlink(l.e.alvo, destino)
		}
		if err != nil {
			x.aviso(l.e, "link não criado: "+err.Error())
			continue
		}
		x.feitos++
	}
	// As pastas de dentro para fora: a mtime de cima não muda ao mexer embaixo.
	for i := len(x.pastas) - 1; i >= 0; i-- {
		p := x.pastas[i]
		destino := filepath.Join(x.raiz, filepath.FromSlash(p.rel))
		if info, err := os.Lstat(destino); err != nil || !info.IsDir() {
			continue
		}
		_ = os.Chmod(destino, fs.FileMode(p.e.modo&0o777|0o700))
		if !p.e.mtime.IsZero() {
			_ = os.Chtimes(destino, time.Now(), p.e.mtime)
		}
	}
	// As pastas que só existiram por causa de um arquivo de dentro ficam 0755.
	_ = filepath.WalkDir(x.raiz, func(c string, d fs.DirEntry, err error) error {
		if err == nil && d.IsDir() && c != x.raiz {
			if info, err := d.Info(); err == nil && info.Mode().Perm() == 0o700 && !x.declarada(c) {
				_ = os.Chmod(c, 0o755)
			}
		}
		return nil
	})
}

func (x *extrator) declarada(caminho string) bool {
	for _, p := range x.pastas {
		if filepath.Join(x.raiz, filepath.FromSlash(p.rel)) == caminho {
			return true
		}
	}
	return false
}

// varre passa a limpo o que uma ferramenta de fora extraiu numa pasta: tira
// os links que apontam para fora e os especiais, tira setuid/setgid/sticky,
// garante que o dono lê e escreve, e soma o tamanho (a trava da bomba).
func varre(raiz string, a andamento, arquivo string, doArquivo int64, semLimite bool) error {
	var total int64
	var bomba bool
	err := filepath.WalkDir(raiz, func(c string, d fs.DirEntry, err error) error {
		if err != nil || c == raiz {
			return nil
		}
		if !a.Segue() {
			return errCancelada
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(raiz, c)
		modo := info.Mode()
		switch {
		case modo&fs.ModeSymlink != 0:
			alvo, _ := os.Readlink(c)
			if !alvoDentro(filepath.ToSlash(rel), alvo) {
				os.Remove(c)
				a.Erro(arquivo, "«"+rel+"»: link que aponta para fora da pasta: ficou de fora")
			}
		case modo.IsDir():
			_ = os.Chmod(c, modo.Perm()|0o700)
		case modo.IsRegular():
			_ = os.Chmod(c, modo.Perm()|0o600)
			total += info.Size()
			if !semLimite && passaDoLimite(total, doArquivo) {
				bomba = true
				return fs.SkipAll
			}
			a.Avanca(1, info.Size(), filepath.Join(arquivo, rel))
		default:
			os.Remove(c)
			a.Erro(arquivo, "«"+rel+"»: é um dispositivo ou um cano: ficou de fora")
		}
		return nil
	})
	if bomba {
		return errBomba
	}
	return err
}

// limpa apaga uma pasta temporária do motor com tudo dentro, sem seguir
// links. Só mexe no que tem o nome que o motor dá aos temporários (a marca) e
// está dentro da pasta dada: nunca apaga um caminho que não conferiu.
func limpa(pasta, dentroDe, marca string) {
	if pasta == "" || dentroDe == "" || marca == "" || filepath.Dir(pasta) != dentroDe || !strings.HasPrefix(filepath.Base(pasta), marca) {
		return
	}
	apagaArvore(pasta)
}

func apagaArvore(caminho string) {
	info, err := os.Lstat(caminho)
	if err != nil {
		return
	}
	if info.IsDir() {
		_ = os.Chmod(caminho, 0o700)
		entradas, _ := os.ReadDir(caminho)
		for _, e := range entradas {
			apagaArvore(filepath.Join(caminho, e.Name()))
		}
	}
	_ = os.Remove(caminho)
}
