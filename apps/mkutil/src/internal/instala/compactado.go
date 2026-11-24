package instala

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"

	"mkutil/internal/base"
)

// Ler e extrair um arquivo compactado que veio da internet, desconfiando de
// tudo que está nele. As regras (as mesmas na inspeção e na extração):
//
//   - nome de entrada: relativo, sem "..", sem "\" e sem byte nulo;
//   - link simbólico: alvo relativo que, resolvido a partir da pasta do link,
//     fica dentro; link físico: para um arquivo de dentro;
//   - dispositivo, fifo, soquete: recusado;
//   - mais entradas ou mais bytes que o limite: recusado (bomba);
//   - a extração escreve por um os.Root preso à pasta de destino: mesmo que
//     uma regra daqui falhasse, o sistema não deixa o caminho sair dela;
//   - no fim, todo link simbólico é resolvido de verdade e tem de cair dentro.
//
// Uma recusa vale para o arquivo inteiro: nada fica extraído pela metade.

// FichaCompactado é o bloco "compactado" do contrato.
type FichaCompactado struct {
	Formato            string   `json:"formato"`
	Entradas           int64    `json:"entradas"`
	TamanhoTotal       int64    `json:"tamanho_total_bytes"`
	PastaUnica         *string  `json:"pasta_unica"`
	Executaveis        []string `json:"executaveis"`
	Atalhos            []string `json:"atalhos"`
	ExecutavelSugerido *string  `json:"executavel_sugerido"`
}

// entrada é um item de um tar ou de um zip, já num formato só.
type entrada struct {
	nome    string
	tipo    byte // 'a' arquivo, 'd' pasta, 'l' link simbólico, 'h' link físico, 'o' outro
	modo    fs.FileMode
	unix    bool // o arquivo tem permissões do Unix (um zip feito no Windows não tem)
	tamanho int64
	alvo    string
	abre    func() (io.ReadCloser, error)
}

// fluxo é o conteúdo descomprimido de um .tar.*: da biblioteca padrão (gzip,
// bzip2) ou de um comando do PATH (xz, zstd) que recebe o arquivo pelo stdin
// e nunca vê o nome dele.
type fluxo struct {
	io.Reader
	arquivo *os.File
	cmd     *exec.Cmd
	erros   *bytes.Buffer
	lidos   *atomic.Int64
	total   int64
	acabou  bool
}

type contador struct {
	r io.Reader
	n *atomic.Int64
}

func (c contador) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n.Add(int64(n))
	return n, err
}

func abreFluxo(arquivo, formato string) (*fluxo, error) {
	f, err := os.Open(arquivo)
	if err != nil {
		return nil, fmt.Errorf("não consegui abrir o arquivo: %v", err)
	}
	fl := &fluxo{arquivo: f, lidos: &atomic.Int64{}}
	if fi, err := f.Stat(); err == nil {
		fl.total = fi.Size()
	}
	contado := contador{f, fl.lidos}
	switch formato {
	case "tar.gz":
		z, err := gzip.NewReader(bufio.NewReader(contado))
		if err != nil {
			f.Close()
			return nil, errors.New("o arquivo não é um gzip de verdade (download incompleto?)")
		}
		fl.Reader = z
	case "tar.bz2":
		fl.Reader = bzip2.NewReader(bufio.NewReader(contado))
	case "tar.xz", "tar.zst":
		nome := map[string]string{"tar.xz": "xz", "tar.zst": "zstd"}[formato]
		caminho, err := exec.LookPath(nome)
		if err != nil {
			f.Close()
			return nil, fmt.Errorf("para abrir um .%s é preciso o %s, que não achei nesta máquina", formato, nome)
		}
		fl.cmd = exec.Command(caminho, "-dc")
		fl.cmd.Stdin = contado
		fl.erros = &bytes.Buffer{}
		fl.cmd.Stderr = fl.erros
		saida, err := fl.cmd.StdoutPipe()
		if err == nil {
			err = fl.cmd.Start()
		}
		if err != nil {
			f.Close()
			return nil, fmt.Errorf("não consegui rodar o %s: %v", nome, err)
		}
		fl.Reader = saida
	default:
		f.Close()
		return nil, fmt.Errorf("não sei abrir o formato %s", formato)
	}
	return fl, nil
}

// Close larga o arquivo e, se há um descompressor rodando, o encerra.
func (f *fluxo) Close() error {
	if f.cmd != nil && f.cmd.Process != nil && !f.acabou {
		f.acabou = true
		_ = f.cmd.Process.Kill()
		_ = f.cmd.Wait()
	}
	return f.arquivo.Close()
}

// fim é chamado quando o conteúdo acabou: se o descompressor saiu com erro, o
// que veio dele não é o arquivo inteiro.
func (f *fluxo) fim() error {
	if f.cmd == nil || f.acabou {
		return nil
	}
	f.acabou = true
	if err := f.cmd.Wait(); err != nil {
		return f.erro(fmt.Errorf("o descompressor falhou: %v", err))
	}
	return nil
}

// erro troca um erro de leitura pela frase do descompressor, se ele disse algo.
func (f *fluxo) erro(err error) error {
	if f.erros != nil {
		if frase := base.Resumo(f.erros.String()); frase != "" {
			return errors.New(frase)
		}
	}
	return err
}

// fracao é quanto do arquivo já foi lido, de 0 a 1.
func (f *fluxo) fracao() float64 {
	if f.total <= 0 {
		return 0
	}
	return min(1, float64(f.lidos.Load())/float64(f.total))
}

// percorre chama visita para cada entrada do arquivo, na ordem. `andou` (pode
// ser nil) recebe quanto já foi, de 0 a 1.
func percorre(arquivo, formato string, andou func(float64), visita func(e *entrada) error) error {
	if formato == "zip" {
		z, err := zip.OpenReader(arquivo)
		if err != nil {
			return fmt.Errorf("não consegui ler o zip: %v", err)
		}
		defer z.Close()
		for i, f := range z.File {
			modo := f.Mode()
			e := &entrada{nome: f.Name, modo: modo.Perm(), unix: f.CreatorVersion>>8 == 3, tamanho: int64(f.UncompressedSize64), abre: f.Open}
			if f.UncompressedSize64 > 1<<62 {
				return base.Recusa("o zip declara uma entrada de tamanho impossível: não extraio")
			}
			switch {
			case modo.IsDir() || strings.HasSuffix(f.Name, "/"):
				e.tipo = 'd'
			case modo&fs.ModeSymlink != 0:
				e.tipo = 'l'
				r, err := f.Open()
				if err != nil {
					return fmt.Errorf("não consegui ler o zip: %v", err)
				}
				alvo, err := io.ReadAll(io.LimitReader(r, 4097))
				r.Close()
				if err != nil {
					return fmt.Errorf("não consegui ler o zip: %v", err)
				}
				e.alvo, e.tamanho = string(alvo), 0
			case modo&(fs.ModeDevice|fs.ModeCharDevice|fs.ModeNamedPipe|fs.ModeSocket|fs.ModeIrregular) != 0:
				e.tipo = 'o'
			default:
				e.tipo = 'a'
			}
			if err := visita(e); err != nil {
				return err
			}
			if andou != nil {
				andou(float64(i+1) / float64(len(z.File)))
			}
		}
		return nil
	}

	fl, err := abreFluxo(arquivo, formato)
	if err != nil {
		return err
	}
	defer fl.Close()
	tr := tar.NewReader(bufio.NewReaderSize(fl, 1<<16))
	for {
		h, err := tr.Next()
		if err == io.EOF {
			if err := fl.fim(); err != nil {
				return fmt.Errorf("não consegui ler o arquivo (está inteiro?): %v", err)
			}
			return nil
		}
		if err != nil {
			return fmt.Errorf("não consegui ler o arquivo (está inteiro?): %v", fl.erro(err))
		}
		e := &entrada{nome: h.Name, modo: fs.FileMode(h.Mode).Perm(), unix: true, tamanho: h.Size, alvo: h.Linkname,
			abre: func() (io.ReadCloser, error) { return io.NopCloser(tr), nil }}
		switch h.Typeflag {
		case tar.TypeReg, tar.TypeRegA, tar.TypeGNUSparse: //nolint:staticcheck // TypeRegA ainda aparece em arquivos antigos
			e.tipo = 'a'
		case tar.TypeDir:
			e.tipo, e.tamanho = 'd', 0
		case tar.TypeSymlink:
			e.tipo, e.tamanho = 'l', 0
		case tar.TypeLink:
			e.tipo, e.tamanho = 'h', 0
		case tar.TypeXGlobalHeader:
			continue
		default:
			e.tipo = 'o'
		}
		if err := visita(e); err != nil {
			return err
		}
		if andou != nil {
			andou(fl.fracao())
		}
	}
}

// nomeSeguro confere o nome de uma entrada e o devolve limpo (sem "./", sem
// barras repetidas). "" é a própria raiz ("./"), que não vira nada.
func nomeSeguro(nome string) (string, error) {
	if nome == "" {
		return "", base.Recusa("o arquivo tem uma entrada sem nome: não extraio")
	}
	if strings.ContainsRune(nome, 0) || strings.ContainsRune(nome, '\\') {
		return "", base.Recusa("o arquivo tem uma entrada com nome estranho (%q): não extraio", nome)
	}
	if strings.HasPrefix(nome, "/") {
		return "", base.Recusa("o arquivo tem um caminho absoluto (%s), que sairia da pasta: não extraio", nome)
	}
	if len(nome) > 4096 {
		return "", base.Recusa("o arquivo tem um caminho comprido demais: não extraio")
	}
	var partes []string
	for _, p := range strings.Split(nome, "/") {
		switch {
		case p == "" || p == ".":
		case p == "..":
			return "", base.Recusa("o arquivo tem um caminho que sai da pasta (%s): não extraio", nome)
		case len(p) > 255:
			return "", base.Recusa("o arquivo tem um nome comprido demais: não extraio")
		default:
			partes = append(partes, p)
		}
	}
	return strings.Join(partes, "/"), nil
}

// alvoSeguro confere o alvo de um link simbólico que fica em `rel`.
func alvoSeguro(rel, alvo string) error {
	if alvo == "" || len(alvo) > 4096 || strings.ContainsRune(alvo, 0) {
		return base.Recusa("o arquivo tem um link (%s) com alvo estranho: não extraio", rel)
	}
	if strings.HasPrefix(alvo, "/") {
		return base.Recusa("o arquivo tem um link (%s) que aponta para fora da pasta (%s): não extraio", rel, alvo)
	}
	resolvido := path.Join(path.Dir(rel), alvo)
	if resolvido == ".." || strings.HasPrefix(resolvido, "../") {
		return base.Recusa("o arquivo tem um link (%s) que aponta para fora da pasta (%s): não extraio", rel, alvo)
	}
	return nil
}

// fiscal aplica as regras a cada entrada e faz a conta dos limites.
type fiscal struct {
	maxBytes, maxEntradas int64
	entradas, bytes       int64
}

func novoFiscal() *fiscal {
	return &fiscal{maxBytes: base.LimiteBytes(), maxEntradas: base.LimiteEntradas()}
}

func (f *fiscal) bomba() error {
	return base.Recusa("o arquivo é grande demais depois de extraído (passa de %d MB ou de %d entradas): pode ser uma bomba, não extraio",
		f.maxBytes>>20, f.maxEntradas)
}

// confere valida a entrada e devolve o nome limpo. Conta a entrada e o
// tamanho que ela declara.
func (f *fiscal) confere(e *entrada) (string, error) {
	rel, err := nomeSeguro(e.nome)
	if err != nil {
		return "", err
	}
	f.entradas++
	if f.entradas > f.maxEntradas {
		return "", f.bomba()
	}
	switch e.tipo {
	case 'a':
		if e.tamanho < 0 || e.tamanho > f.maxBytes-f.bytes {
			return "", f.bomba()
		}
		f.bytes += e.tamanho
		if rel == "" {
			return "", base.Recusa("o arquivo tem uma entrada sem nome: não extraio")
		}
	case 'd':
	case 'l':
		if rel == "" {
			return "", base.Recusa("o arquivo tem uma entrada sem nome: não extraio")
		}
		if err := alvoSeguro(rel, e.alvo); err != nil {
			return "", err
		}
	case 'h':
		alvo, err := nomeSeguro(e.alvo)
		if err != nil || alvo == "" || rel == "" {
			return "", base.Recusa("o arquivo tem um link físico (%s) que aponta para fora da pasta: não extraio", e.nome)
		}
	default:
		return "", base.Recusa("o arquivo tem um arquivo especial (%s: dispositivo, fifo ou soquete): não extraio", e.nome)
	}
	return rel, nil
}

type candidato struct {
	rel string
	elf bool
}

func ehBiblioteca(nome string) bool {
	b := strings.ToLower(path.Base(nome))
	return strings.HasSuffix(b, ".so") || strings.Contains(b, ".so.")
}

// inspecionaCompactado lê o arquivo inteiro uma vez, sem extrair: aplica as
// regras e junta o que a ficha mostra.
func inspecionaCompactado(arquivo, formato, id string) (*FichaCompactado, error) {
	fisc := novoFiscal()
	topos := map[string]bool{} // o nome do topo -> é (ou tem) pasta
	var execs []candidato
	var atalhos []string
	var links [][2]string
	abertosSemModo := 0
	err := percorre(arquivo, formato, nil, func(e *entrada) error {
		rel, err := fisc.confere(e)
		if err != nil {
			return err
		}
		if rel == "" {
			return nil
		}
		topo, resto, temResto := strings.Cut(rel, "/")
		topos[topo] = topos[topo] || temResto && resto != "" || e.tipo == 'd'
		if e.tipo == 'l' {
			links = append(links, [2]string{rel, e.alvo})
		}
		if e.tipo != 'a' {
			return nil
		}
		if strings.HasSuffix(rel, ".desktop") && len(atalhos) < 50 {
			atalhos = append(atalhos, rel)
		}
		if e.tamanho < 4 || ehBiblioteca(rel) || len(execs) >= 2000 {
			return nil
		}
		if e.unix && e.modo&0o111 == 0 {
			return nil
		}
		if !e.unix {
			if abertosSemModo++; abertosSemModo > 5000 {
				return nil
			}
		}
		r, err := e.abre()
		if err != nil {
			return nil
		}
		var magia [4]byte
		_, err = io.ReadFull(r, magia[:])
		r.Close()
		if err != nil {
			return nil
		}
		ehElf := bytes.Equal(magia[:], []byte{0x7f, 'E', 'L', 'F'})
		if ehElf || magia[0] == '#' && magia[1] == '!' {
			execs = append(execs, candidato{rel, ehElf})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if fisc.entradas == 0 || len(topos) == 0 {
		return nil, errors.New("o arquivo está vazio")
	}
	ficha := &FichaCompactado{Formato: formato, Entradas: fisc.entradas, TamanhoTotal: fisc.bytes,
		Executaveis: []string{}, Atalhos: []string{}}
	tira := ""
	if len(topos) == 1 {
		for topo, ehPasta := range topos {
			if ehPasta {
				tira = topo
				ficha.PastaUnica = &topo
			}
		}
	}
	corta := func(rel string) string {
		if tira == "" {
			return rel
		}
		return strings.TrimPrefix(rel, tira+"/")
	}
	// Sem a pasta do topo, um link que subia até a raiz do arquivo passa a
	// sair da pasta de destino: confere de novo, como a extração vai ver.
	for _, l := range links {
		if rel := corta(l[0]); rel != tira {
			if err := alvoSeguro(rel, l[1]); err != nil {
				return nil, err
			}
		}
	}
	bate := func(rel string) bool {
		b := path.Base(rel)
		for _, ext := range []string{".sh", ".bin", ".run", ".x86_64", ".AppImage"} {
			b = strings.TrimSuffix(b, ext)
		}
		return FazId(b) == id
	}
	sort.SliceStable(execs, func(i, j int) bool {
		a, b := corta(execs[i].rel), corta(execs[j].rel)
		if x, y := bate(a), bate(b); x != y {
			return x
		}
		if x, y := strings.Count(a, "/"), strings.Count(b, "/"); x != y {
			return x < y
		}
		if execs[i].elf != execs[j].elf {
			return execs[i].elf
		}
		return a < b
	})
	for i, c := range execs {
		if i >= 40 {
			break
		}
		ficha.Executaveis = append(ficha.Executaveis, corta(c.rel))
	}
	if len(ficha.Executaveis) > 0 {
		ficha.ExecutavelSugerido = &ficha.Executaveis[0]
	}
	sort.Strings(atalhos)
	for i, a := range atalhos {
		if i >= 10 {
			break
		}
		ficha.Atalhos = append(ficha.Atalhos, corta(a))
	}
	return ficha, nil
}

// extrai escreve o conteúdo do arquivo em destino (uma pasta que já existe e
// está vazia), sem a pasta do topo `tira`. Devolve quantos bytes escreveu.
func extrai(arquivo, formato, destino, tira string, andou func(pct float64, entrada string)) (int64, error) {
	raiz, err := os.OpenRoot(destino)
	if err != nil {
		return 0, err
	}
	defer raiz.Close()
	fisc := novoFiscal()
	var escritos int64
	atual := ""
	preso := func(err error) error {
		if err != nil && strings.Contains(err.Error(), "path escapes") {
			return base.Recusa("o arquivo tenta escrever fora da pasta (%s): não extraio", atual)
		}
		return err
	}
	// Tira o que já existe no lugar (uma entrada repetida), a não ser pasta.
	limpa := func(rel string) error {
		fi, err := raiz.Lstat(rel)
		if err != nil {
			return nil
		}
		if fi.IsDir() {
			return base.Recusa("o arquivo tem uma entrada (%s) que troca uma pasta por outra coisa: não extraio", rel)
		}
		return preso(raiz.Remove(rel))
	}
	corta := func(rel string) (string, error) {
		if tira == "" {
			return rel, nil
		}
		if rel == tira {
			return "", nil
		}
		if !strings.HasPrefix(rel, tira+"/") {
			return "", errors.New("o arquivo mudou enquanto era lido: tente de novo")
		}
		return rel[len(tira)+1:], nil
	}
	err = percorre(arquivo, formato, func(fr float64) {
		if andou != nil {
			andou(fr*100, atual)
		}
	}, func(e *entrada) error {
		rel, err := fisc.confere(e)
		if err != nil {
			return err
		}
		if rel, err = corta(rel); err != nil || rel == "" {
			return err
		}
		atual = rel
		if e.tipo != 'd' {
			if err := preso(raiz.MkdirAll(path.Dir(rel), 0o755)); err != nil {
				return err
			}
		}
		switch e.tipo {
		case 'd':
			return preso(raiz.MkdirAll(rel, 0o755))
		case 'l':
			// O alvo é conferido de novo já sem a pasta do topo.
			if err := alvoSeguro(rel, e.alvo); err != nil {
				return err
			}
			if err := limpa(rel); err != nil {
				return err
			}
			return preso(raiz.Symlink(e.alvo, rel))
		case 'h':
			alvo, err := nomeSeguro(e.alvo)
			if err == nil {
				alvo, err = corta(alvo)
			}
			if err != nil || alvo == "" {
				return base.Recusa("o arquivo tem um link físico (%s) que aponta para fora da pasta: não extraio", rel)
			}
			fi, err := raiz.Lstat(alvo)
			if err != nil || !fi.Mode().IsRegular() {
				return base.Recusa("o arquivo tem um link físico (%s) para algo que não é um arquivo de dentro: não extraio", rel)
			}
			if err := limpa(rel); err != nil {
				return err
			}
			return preso(raiz.Link(alvo, rel))
		}
		// Um arquivo comum. Só as permissões simples: nada de setuid.
		modo := fs.FileMode(0o644)
		if e.unix {
			modo = e.modo&0o755 | 0o600
		}
		if err := limpa(rel); err != nil {
			return err
		}
		r, err := e.abre()
		if err != nil {
			return fmt.Errorf("não consegui ler %s de dentro do arquivo: %v", rel, err)
		}
		defer r.Close()
		f, err := raiz.OpenFile(rel, os.O_WRONLY|os.O_CREATE|os.O_EXCL, modo)
		if err != nil {
			return preso(err)
		}
		// Os bytes de verdade, não os declarados: um a mais que o limite é bomba.
		n, err := io.Copy(f, io.LimitReader(r, fisc.maxBytes-escritos+1))
		escritos += n
		if e := f.Close(); err == nil {
			err = e
		}
		if escritos > fisc.maxBytes {
			return fisc.bomba()
		}
		if err != nil {
			return fmt.Errorf("não consegui extrair %s: %v", rel, err)
		}
		return nil
	})
	if err != nil {
		return escritos, err
	}
	return escritos, confereLinks(destino)
}

// confereLinks resolve de verdade cada link simbólico da pasta extraída: o
// que existe tem de cair dentro dela. Um link para algo que não existe só
// passa se não tem "..": aí ele só desce por pastas e links já conferidos.
func confereLinks(destino string) error {
	real, err := filepath.EvalSymlinks(destino)
	if err != nil {
		return err
	}
	return filepath.WalkDir(destino, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&fs.ModeSymlink == 0 {
			return nil
		}
		rel, _ := filepath.Rel(destino, p)
		onde, err := filepath.EvalSymlinks(p)
		if err != nil {
			alvo, _ := os.Readlink(p)
			for _, parte := range strings.Split(alvo, "/") {
				if parte == ".." {
					return base.Recusa("o arquivo tem um link (%s) que aponta para fora da pasta (%s): não extraio", rel, alvo)
				}
			}
			return nil
		}
		if onde != real && !strings.HasPrefix(onde, real+string(filepath.Separator)) {
			return base.Recusa("o arquivo tem um link (%s) que aponta para fora da pasta: não extraio", rel)
		}
		return nil
	})
}
