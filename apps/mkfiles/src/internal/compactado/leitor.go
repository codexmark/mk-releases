package compactado

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// Os tipos de uma entrada de compactado.
const (
	eArquivo  = iota
	ePasta    // pasta
	eLink     // link simbólico
	eDuro     // hardlink do tar (alvo é outra entrada)
	eEspecial // dispositivo, fifo, soquete: nunca é criado
)

// entrada é um item de dentro de um compactado.
type entrada struct {
	nome    string // o caminho de dentro, limpo ("pasta/a.txt"); vale se ruim == ""
	cru     string // como está escrito no arquivo
	ruim    string // por que o nome não serve ("" se serve)
	tipo    int
	tamanho int64 // -1 se o formato não diz
	mtime   time.Time
	modo    uint32 // os 12 bits
	alvo    string
	cifrada bool
}

// nomeLimpo confere o caminho de uma entrada: nada de byte nulo, barra
// invertida, caminho absoluto ou "..". Devolve o caminho limpo, ou o motivo.
func nomeLimpo(cru string) (limpo, ruim string) {
	switch {
	case strings.ContainsRune(cru, 0):
		return "", "nome com byte nulo"
	case strings.ContainsRune(cru, '\\'):
		return "", "nome com barra invertida"
	case strings.HasPrefix(cru, "/"):
		return "", "caminho absoluto"
	}
	var partes []string
	for _, p := range strings.Split(cru, "/") {
		switch p {
		case "", ".":
		case "..":
			return "", "caminho que sai da pasta"
		default:
			if len(p) > 255 {
				return "", "nome comprido demais"
			}
			partes = append(partes, p)
		}
	}
	if len(partes) == 0 {
		return "", "nome vazio"
	}
	return strings.Join(partes, "/"), ""
}

func novaEntrada(cru string) entrada {
	e := entrada{cru: cru, tamanho: -1}
	e.nome, e.ruim = nomeLimpo(cru)
	return e
}

// bitsDe traduz um fs.FileMode nos 12 bits de permissão.
func bitsDe(m fs.FileMode) uint32 {
	bits := uint32(m.Perm())
	if m&fs.ModeSetuid != 0 {
		bits |= 0o4000
	}
	if m&fs.ModeSetgid != 0 {
		bits |= 0o2000
	}
	if m&fs.ModeSticky != 0 {
		bits |= 0o1000
	}
	return bits
}

// visita recebe cada entrada; abre dá o conteúdo (só de eArquivo). Devolver
// um erro para a leitura.
type visita func(e entrada, abre func() (io.ReadCloser, error)) error

// leitura percorre um compactado dos formatos que o motor lê sozinho (zip,
// tar e as cascas gz/bz2, mais xz/zst por um descompressor em pipe).
type leitura struct {
	ctx     context.Context
	caminho string
	formato string
	lido    atomic.Int64 // bytes compactados já lidos (o andamento)
}

type contador struct {
	r io.Reader
	n *atomic.Int64
}

func (c contador) Read(b []byte) (int, error) {
	n, err := c.r.Read(b)
	c.n.Add(int64(n))
	return n, err
}

// O prazo de um comando de fora que só lista, e o de um que extrai ou
// compacta (que também para quando a tarefa é cancelada).
var (
	prazoDeListar  = 60 * time.Second
	prazoDeExtrair = 6 * time.Hour
)

// descompressor abre a casca de um arquivo: devolve o fluxo descompactado e
// uma função que termina (e diz se o comando de fora reclamou).
func (l *leitura) descompressor(f *os.File, casca string) (io.Reader, func() error, error) {
	origem := contador{f, &l.lido}
	nada := func() error { return nil }
	switch casca {
	case "":
		return origem, nada, nil
	case "gz":
		z, err := gzip.NewReader(bufio.NewReaderSize(origem, 64<<10))
		if err != nil {
			return nil, nil, erroDeFormato
		}
		return z, z.Close, nil
	case "bz2":
		return bzip2.NewReader(bufio.NewReaderSize(origem, 64<<10)), nada, nil
	}
	comando, pacote := oXz(), "xz"
	if casca == "zst" {
		comando, pacote = oZstd(), "zstd"
	}
	if comando == "" {
		return nil, nil, semFerramenta(pacote)
	}
	ctx, cancela := context.WithTimeout(l.ctx, prazoDeExtrair)
	cmd := exec.CommandContext(ctx, comando, "-dc")
	cmd.Stdin = origem
	var reclamacao bytes.Buffer
	cmd.Stderr = &limitado{b: &reclamacao, resta: 4 << 10}
	cmd.WaitDelay = 2 * time.Second
	saida, err := cmd.StdoutPipe()
	if err == nil {
		err = cmd.Start()
	}
	if err != nil {
		cancela()
		return nil, nil, errors.New("o " + pacote + " não rodou: " + err.Error())
	}
	fecha := func() error {
		// Quem parou antes do fim (ou acabou): o resto do fluxo não interessa.
		sobra, _ := io.Copy(io.Discard, io.LimitReader(saida, 1<<20))
		if sobra >= 1<<20 {
			cancela() // ainda tinha muito a dizer: não se espera
		}
		defer cancela()
		err := cmd.Wait()
		if err != nil && l.ctx.Err() == nil && reclamacao.Len() > 0 {
			return errors.New("o " + pacote + " disse: " + strings.TrimSpace(reclamacao.String()))
		}
		return nil
	}
	return saida, fecha, nil
}

// limitado guarda só o começo do que um comando escreve no stderr.
type limitado struct {
	b     *bytes.Buffer
	resta int
}

func (l *limitado) Write(p []byte) (int, error) {
	if l.resta > 0 {
		n := min(len(p), l.resta)
		l.b.Write(p[:n])
		l.resta -= n
	}
	return len(p), nil
}

// percorre chama fn para cada entrada, na ordem do arquivo.
func (l *leitura) percorre(fn visita) error {
	if l.formato == "zip" {
		return l.percorreZip(fn)
	}
	f, err := os.Open(l.caminho)
	if err != nil {
		return err
	}
	defer f.Close()
	casca := ""
	switch l.formato {
	case "tar":
	case "tar.gz", "tar.bz2", "tar.xz", "tar.zst":
		casca = strings.TrimPrefix(l.formato, "tar.")
	case "gz", "bz2", "xz", "zst":
		casca = l.formato
	default:
		return erroDeFormato
	}
	fluxo, fecha, err := l.descompressor(f, casca)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(l.formato, "tar") {
		// Um arquivo só, com o nome do compactado sem a extensão.
		_, nome := peloNome(filepath.Base(l.caminho))
		if nome == filepath.Base(l.caminho) {
			nome += ".extraído"
		}
		e := novaEntrada(nome)
		e.modo = 0o644
		if info, err := f.Stat(); err == nil {
			e.mtime = info.ModTime()
		}
		err = fn(e, func() (io.ReadCloser, error) { return io.NopCloser(fluxo), nil })
		if e2 := fecha(); err == nil {
			err = e2
		}
		return err
	}
	tr := tar.NewReader(fluxo)
	for {
		if l.ctx.Err() != nil {
			_ = fecha()
			return errCancelada
		}
		h, err := tr.Next()
		if err == io.EOF {
			return fecha()
		}
		if err != nil {
			if e2 := fecha(); e2 != nil {
				return e2
			}
			return errors.New("o arquivo está corrompido ou cortado")
		}
		e := novaEntrada(h.Name)
		e.tamanho, e.mtime, e.modo = h.Size, h.ModTime, bitsDe(h.FileInfo().Mode())
		switch h.Typeflag {
		case tar.TypeReg, tar.TypeRegA, tar.TypeGNUSparse, tar.TypeCont:
			e.tipo = eArquivo
		case tar.TypeDir:
			e.tipo, e.tamanho = ePasta, 0
		case tar.TypeSymlink:
			e.tipo, e.alvo, e.tamanho = eLink, h.Linkname, 0
		case tar.TypeLink:
			e.tipo, e.alvo, e.tamanho = eDuro, h.Linkname, 0
		case tar.TypeXGlobalHeader, tar.TypeXHeader, tar.TypeGNULongName, tar.TypeGNULongLink:
			continue
		default:
			e.tipo, e.tamanho = eEspecial, 0
		}
		if err := fn(e, func() (io.ReadCloser, error) { return io.NopCloser(tr), nil }); err != nil {
			_ = fecha()
			return err
		}
	}
}

func (l *leitura) percorreZip(fn visita) error {
	z, err := zip.OpenReader(l.caminho)
	if err != nil {
		return erroDeFormato
	}
	defer z.Close()
	for _, f := range z.File {
		if l.ctx.Err() != nil {
			return errCancelada
		}
		e := novaEntrada(f.Name)
		modo := f.Mode()
		e.tamanho, e.mtime, e.modo = int64(f.UncompressedSize64), f.Modified, bitsDe(modo)
		e.cifrada = f.Flags&1 != 0
		switch {
		case modo.IsDir() || strings.HasSuffix(f.Name, "/"):
			e.tipo, e.tamanho = ePasta, 0
		case modo&fs.ModeSymlink != 0:
			e.tipo = eLink
			if !e.cifrada && f.UncompressedSize64 <= 4096 {
				if r, err := f.Open(); err == nil {
					alvo, _ := io.ReadAll(io.LimitReader(r, 4096))
					r.Close()
					e.alvo = string(alvo)
				}
			}
			e.tamanho = 0
		case modo&(fs.ModeDevice|fs.ModeNamedPipe|fs.ModeSocket|fs.ModeCharDevice) != 0:
			e.tipo, e.tamanho = eEspecial, 0
		}
		arq := f
		l.lido.Add(int64(arq.CompressedSize64))
		if err := fn(e, func() (io.ReadCloser, error) { return arq.Open() }); err != nil {
			return err
		}
	}
	return nil
}

// --- 7z e rar: pela ferramenta de fora ---

// roda executa um comando de fora, com a senha (se há) no stdin, e devolve o
// que ele escreveu. A senha nunca vai na linha de comando nem no ambiente.
func roda(ctx context.Context, prazo time.Duration, senha *string, pasta string, argv ...string) (saida string, err error) {
	ctx, cancela := context.WithTimeout(ctx, prazo)
	defer cancela()
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Dir = pasta
	if senha != nil {
		// Duas vezes: ao cifrar, a ferramenta pede e confirma.
		cmd.Stdin = strings.NewReader(*senha + "\n" + *senha + "\n")
	}
	var tudo bytes.Buffer
	cmd.Stdout = &limitado{b: &tudo, resta: 64 << 20}
	cmd.Stderr = cmd.Stdout
	cmd.WaitDelay = 2 * time.Second
	err = cmd.Run()
	return tudo.String(), err
}

// erroDaFerramenta traduz a falha de um 7z/unrar: senha, ou o que ele disse.
func erroDaFerramenta(saida string, senha *string, err error) error {
	minusculo := strings.ToLower(saida)
	if strings.Contains(minusculo, "wrong password") || strings.Contains(minusculo, "enter password") ||
		strings.Contains(minusculo, "incorrect password") || strings.Contains(minusculo, "encrypted archive") {
		if senha == nil {
			return erroPedeSenha
		}
		return erroSenhaRuim
	}
	linhas := strings.Split(strings.TrimSpace(saida), "\n")
	ultima := strings.TrimSpace(linhas[len(linhas)-1])
	if ultima == "" {
		ultima = err.Error()
	}
	if len(ultima) > 300 {
		ultima = ultima[:300]
	}
	return errors.New("a ferramenta de extração falhou: " + ultima)
}

// lista7z lê o índice de um 7z (ou rar) com o 7z.
func lista7z(ctx context.Context, caminho string, senha *string) ([]entrada, error) {
	saida, err := roda(ctx, prazoDeListar, senha, "", o7z(), "l", "-slt", "-ba", "-sccUTF-8", "--", caminho)
	if err != nil {
		return nil, erroDaFerramenta(saida, senha, err)
	}
	var es []entrada
	for _, bloco := range strings.Split(strings.ReplaceAll(saida, "\r\n", "\n"), "\n\n") {
		campos := map[string]string{}
		for _, linha := range strings.Split(bloco, "\n") {
			if k, v, ok := strings.Cut(linha, " = "); ok {
				campos[strings.TrimSpace(k)] = v
			}
		}
		if campos["Path"] == "" {
			continue
		}
		e := novaEntrada(campos["Path"])
		e.modo = 0o644
		if campos["Folder"] == "+" || strings.HasPrefix(campos["Attributes"], "D") {
			e.tipo, e.modo, e.tamanho = ePasta, 0o755, 0
		} else if n, err := strconv.ParseInt(campos["Size"], 10, 64); err == nil {
			e.tamanho = n
		}
		if m := campos["Modified"]; len(m) >= 19 {
			if quando, err := time.ParseInLocation("2006-01-02 15:04:05", m[:19], time.Local); err == nil {
				e.mtime = quando
			}
		}
		e.cifrada = campos["Encrypted"] == "+"
		es = append(es, e)
	}
	return es, nil
}

// listaRar lê os nomes de um rar sem o 7z: unrar ou bsdtar (só os nomes).
func listaRar(ctx context.Context, caminho string) ([]entrada, error) {
	var saida string
	var err error
	if u := oUnrar(); u != "" {
		saida, err = roda(ctx, prazoDeListar, nil, "", u, "lb", "-p-", "--", caminho)
	} else {
		saida, err = roda(ctx, prazoDeListar, nil, "", oBsdtar(), "-tf", caminho)
	}
	if err != nil {
		return nil, erroDaFerramenta(saida, nil, err)
	}
	var es []entrada
	for _, linha := range strings.Split(saida, "\n") {
		linha = strings.TrimRight(linha, "\r")
		if linha == "" {
			continue
		}
		e := novaEntrada(linha)
		e.modo = 0o644
		if strings.HasSuffix(linha, "/") {
			e.tipo, e.modo, e.tamanho = ePasta, 0o755, 0
		}
		es = append(es, e)
	}
	// Quem é prefixo de outro é pasta (o unrar lb não diz).
	sort.SliceStable(es, func(i, j int) bool { return es[i].nome < es[j].nome })
	for i := range es {
		if i+1 < len(es) && es[i].ruim == "" && strings.HasPrefix(es[i+1].nome, es[i].nome+"/") {
			es[i].tipo, es[i].modo, es[i].tamanho = ePasta, 0o755, 0
		}
	}
	return es, nil
}

// extraiDeFora manda a ferramenta extrair um 7z, um rar ou um zip cifrado
// inteiro para dentro de uma pasta (que o motor acabou de criar, vazia). O
// que saiu de lá ainda passa pela varredura (varre) antes de ir para o lugar.
func extraiDeFora(ctx context.Context, caminho, formato string, senha *string, pasta string) error {
	var saida string
	var err error
	switch {
	case o7z() != "":
		saida, err = roda(ctx, prazoDeExtrair, senha, pasta, o7z(), "x", "-y", "-bd", "-sccUTF-8", "-o"+pasta, "--", caminho)
	case formato == "rar" && oUnrar() != "":
		argv := []string{oUnrar(), "x", "-o+", "-y"}
		if senha == nil {
			argv = append(argv, "-p-")
		}
		saida, err = roda(ctx, prazoDeExtrair, senha, pasta, append(argv, "--", caminho, pasta+"/")...)
	case formato == "rar" && oBsdtar() != "":
		saida, err = roda(ctx, prazoDeExtrair, nil, pasta, oBsdtar(), "-xf", caminho, "-C", pasta)
	default:
		return semFerramenta(map[bool]string{true: "unrar", false: "7zip"}[formato == "rar"])
	}
	if ctx.Err() != nil {
		return errCancelada
	}
	if err != nil {
		return erroDaFerramenta(saida, senha, err)
	}
	return nil
}
