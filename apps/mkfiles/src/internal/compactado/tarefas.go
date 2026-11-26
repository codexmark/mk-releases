package compactado

import (
	"archive/tar"
	"archive/zip"
	"compress/flate"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"mkfiles/internal/base"
	"mkfiles/internal/lixeira"
	"mkfiles/internal/operar"
)

func init() {
	operar.Registrar("extrair", novaExtracao)
	operar.Registrar("compactar", novaCompactacao)
}

const (
	marcaDeExtrair   = ".mkfiles-extrai-"
	marcaDeCompactar = ".mkfiles-compacta-"
)

var serie atomic.Int64

func temporario(pasta, marca, fim string) string {
	return filepath.Join(pasta, marca+strconv.Itoa(os.Getpid())+"-"+strconv.FormatInt(serie.Add(1), 10)+fim)
}

// anota diz ao trilho o que a tarefa criou, para o desfazer (apaga o arquivo
// que não mudou, a pasta que ficou vazia). O trilho só aceita a anotação se
// tiver FezArquivo/FezPasta exportados.
func anota(t *operar.Tarefa, caminho string, pasta bool) {
	if pasta {
		if f, ok := any(t).(interface{ FezPasta(string) }); ok {
			f.FezPasta(caminho)
		}
	} else if f, ok := any(t).(interface{ FezArquivo(string) }); ok {
		f.FezArquivo(caminho)
	}
}

// anotaArvore anota uma árvore recém-posta no lugar: as pastas antes do que
// têm dentro (o desfazer anda de trás para a frente).
func anotaArvore(t *operar.Tarefa, raiz string) {
	_ = filepath.WalkDir(raiz, func(c string, d fs.DirEntry, err error) error {
		if err == nil {
			anota(t, c, d.IsDir())
		}
		return nil
	})
}

// comCancelamento dá um contexto que acaba quando a tarefa é cancelada.
func comCancelamento(t *operar.Tarefa) (context.Context, func()) {
	ctx, cancela := context.WithCancel(context.Background())
	go func() {
		relogio := time.NewTicker(100 * time.Millisecond)
		defer relogio.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-relogio.C:
				if t.Cancelada() {
					cancela()
					return
				}
			}
		}
	}()
	return ctx, cancela
}

func ehPasta(caminho string) bool {
	info, err := os.Stat(caminho)
	return err == nil && info.IsDir()
}

func existe(caminho string) bool {
	_, err := os.Lstat(caminho)
	return err == nil
}

func livreEm(pasta, nome string, dePasta bool) string {
	return lixeira.Numerado(nome, dePasta, func(n string) bool { return existe(filepath.Join(pasta, n)) })
}

// --- extrair ---

type extracao struct {
	caminho   string
	destino   string
	emPasta   bool
	internos  []string
	senha     *string
	semLimite bool
	formato   string
}

func novaExtracao(p *base.Pedido) (operar.Trabalho, error) {
	var q struct {
		Caminho   string   `json:"caminho"`
		Destino   string   `json:"destino"`
		Modo      string   `json:"modo"`
		Internos  []string `json:"internos"`
		Senha     *string  `json:"senha"`
		SemLimite bool     `json:"sem_limite"`
	}
	if err := p.Ler(&q); err != nil {
		return nil, errors.New("pedido de extrair mal formado")
	}
	if err := base.Caminho(q.Caminho); err != nil {
		return nil, err
	}
	if q.Destino == "" {
		q.Destino = filepath.Dir(q.Caminho)
	}
	if err := base.Caminho(q.Destino); err != nil {
		return nil, err
	}
	if !ehPasta(q.Destino) {
		return nil, errors.New("o destino não é uma pasta")
	}
	if q.Modo == "" {
		q.Modo = "pasta"
	}
	if q.Modo != "aqui" && q.Modo != "pasta" {
		return nil, &operar.DeCampo{Campo: "modo", Frase: "modo de extrair desconhecido: «" + q.Modo + "»"}
	}
	if q.Senha != nil && (*q.Senha == "" || strings.ContainsAny(*q.Senha, "\n\r\x00")) {
		return nil, &operar.DeCampo{Campo: "senha", Frase: "a senha não pode ser vazia nem ter quebra de linha"}
	}
	e := &extracao{caminho: q.Caminho, destino: q.Destino, emPasta: q.Modo == "pasta" && len(q.Internos) == 0,
		senha: q.Senha, semLimite: q.SemLimite}
	for _, i := range q.Internos {
		limpo, ruim := nomeLimpo(i)
		if ruim != "" {
			return nil, errors.New("item de dentro inválido: " + ruim)
		}
		e.internos = append(e.internos, limpo)
	}
	if info, err := os.Stat(q.Caminho); err != nil || info.IsDir() {
		return nil, errors.New("o arquivo não existe mais")
	}
	if e.formato = formatoDe(q.Caminho); e.formato == "" {
		return nil, erroDeFormato
	}
	if pacote := falta(e.formato); pacote != "" {
		return nil, semFerramenta(pacote)
	}
	return e, nil
}

func (e *extracao) Titulo() string         { return "Extraindo «" + filepath.Base(e.caminho) + "»" }
func (e *extracao) Destino() string        { return e.destino }
func (e *extracao) Descricao() string      { return "Extrair «" + filepath.Base(e.caminho) + "»" }
func (e *extracao) Refaz() operar.Trabalho { n := *e; return &n }
func (e *extracao) Prepara(t *operar.Tarefa) {
	// Só o zip diz de graça quanto há: o resto conta enquanto anda.
	if e.formato != "zip" {
		return
	}
	if z, err := zip.OpenReader(e.caminho); err == nil {
		var bytes int64
		itens := 0
		for _, f := range z.File {
			if nome, ruim := nomeLimpo(f.Name); ruim == "" {
				if _, quer := e.filtro()(nome); quer {
					itens++
					bytes += int64(f.UncompressedSize64)
				}
			}
		}
		z.Close()
		t.Total(itens, bytes)
	}
}

// filtro diz que entradas entram e onde: com `internos`, só o pedido, e cada
// um pelo último nome (a pasta "a/b" vira "b/…").
func (e *extracao) filtro() func(string) (string, bool) {
	if len(e.internos) == 0 {
		return func(n string) (string, bool) { return n, true }
	}
	return func(n string) (string, bool) {
		for _, i := range e.internos {
			corte := strings.LastIndexByte(i, '/') + 1
			if n == i || strings.HasPrefix(n, i+"/") {
				return n[corte:], true
			}
		}
		return "", false
	}
}

func (e *extracao) Roda(t *operar.Tarefa) {
	info, err := os.Stat(e.caminho)
	if err != nil {
		t.Erro(e.caminho, "o arquivo não existe mais")
		return
	}
	tmp := temporario(e.destino, marcaDeExtrair, "")
	if err := os.Mkdir(tmp, 0o700); err != nil {
		t.Erro(e.destino, "não deu para criar a pasta temporária: "+err.Error())
		return
	}
	defer limpa(tmp, e.destino, marcaDeExtrair)
	ctx, cancela := comCancelamento(t)
	defer cancela()

	feitos, err := e.extrai(ctx, t, tmp, info.Size())
	switch {
	case t.Cancelada() || err == errCancelada:
		return
	case err == errBomba:
		t.Erro(e.caminho, errBomba.Error()+". Nada foi extraído")
		return
	case err != nil:
		t.Erro(e.caminho, err.Error())
		return
	case feitos == 0:
		t.Erro(e.caminho, "nada foi extraído")
		return
	}

	if e.emPasta {
		_, nome := peloNome(filepath.Base(e.caminho))
		if nome == filepath.Base(e.caminho) || base.Nome(nome) != nil {
			nome = filepath.Base(e.caminho) + " (extraído)"
		}
		if existe(filepath.Join(e.destino, nome)) {
			nome = livreEm(e.destino, nome, true)
		}
		final := filepath.Join(e.destino, nome)
		_ = os.Chmod(tmp, 0o755)
		if err := os.Rename(tmp, final); err != nil {
			t.Erro(final, err.Error())
			return
		}
		t.Criou(final)
		anotaArvore(t, final)
		return
	}
	filhos, _ := os.ReadDir(tmp)
	for _, f := range filhos {
		if !t.Segue() {
			return
		}
		e.coloca(t, filepath.Join(tmp, f.Name()), filepath.Join(e.destino, f.Name()), true)
	}
}

// extrai põe o conteúdo na pasta temporária, com todas as travas. Devolve
// quantos itens saíram.
func (e *extracao) extrai(ctx context.Context, t *operar.Tarefa, tmp string, doArquivo int64) (int, error) {
	deFora := e.formato == "7z" || e.formato == "rar"
	if e.formato == "zip" {
		// Um zip com senha a biblioteca padrão não abre: vai pelo 7z.
		if z, err := zip.OpenReader(e.caminho); err == nil {
			for _, f := range z.File {
				deFora = deFora || f.Flags&1 != 0
			}
			z.Close()
		}
		if deFora && o7z() == "" {
			return 0, erroCifradoZip
		}
	}
	if !deFora {
		x := &extrator{raiz: tmp, arquivo: e.caminho, doArquivo: doArquivo, semLimite: e.semLimite, a: t}
		if len(e.internos) > 0 {
			x.quer = e.filtro()
		}
		l := &leitura{ctx: ctx, caminho: e.caminho, formato: e.formato}
		if err := l.percorre(x.recebe); err != nil {
			return 0, err
		}
		x.termina()
		return x.feitos, nil
	}
	// A ferramenta de fora extrai tudo numa pasta só dela; o que saiu passa
	// pela varredura e só então vai para o lugar.
	if !e.semLimite {
		var es []entrada
		var err error
		if o7z() != "" {
			es, err = lista7z(ctx, e.caminho, e.senha)
		} else {
			es, err = listaRar(ctx, e.caminho)
		}
		if err != nil {
			return 0, err
		}
		ind := indice{es: es}
		if itens, tamanho, sabe := ind.totais(); sabe {
			if passaDoLimite(tamanho, doArquivo) {
				return 0, errBomba
			}
			t.Total(itens, tamanho)
		}
	}
	bruto := tmp
	if len(e.internos) > 0 {
		bruto = filepath.Join(tmp, ".tudo")
		if err := os.Mkdir(bruto, 0o700); err != nil {
			return 0, err
		}
	}
	if err := extraiDeFora(ctx, e.caminho, e.formato, e.senha, bruto); err != nil {
		return 0, err
	}
	if err := varre(bruto, t, e.caminho, doArquivo, e.semLimite); err != nil {
		return 0, err
	}
	if len(e.internos) > 0 {
		for _, i := range e.internos {
			origem := filepath.Join(bruto, filepath.FromSlash(i))
			if !existe(origem) {
				t.Erro(e.caminho, "«"+i+"» não está no arquivo")
				continue
			}
			_ = os.Rename(origem, filepath.Join(tmp, filepath.Base(origem)))
		}
		apagaArvore(bruto)
	}
	filhos, _ := os.ReadDir(tmp)
	return len(filhos), nil
}

// coloca leva um item da pasta temporária para o lugar, perguntando quando o
// lugar está ocupado. Pasta sobre pasta, com "substituir", mescla.
func (e *extracao) coloca(t *operar.Tarefa, de, para string, topo bool) {
	info, err := os.Lstat(de)
	if err != nil {
		return
	}
	pasta := filepath.Dir(para)
	for {
		la, err := os.Lstat(para)
		if err != nil {
			break
		}
		sugestao := livreEm(pasta, filepath.Base(para), info.IsDir())
		r := t.Conflito(de, para, sugestao)
		switch r.Qual {
		case "manter":
			para = filepath.Join(pasta, sugestao)
		case "renomear":
			if err := base.Nome(r.Nome); err != nil {
				t.Erro(para, err.Error())
				return
			}
			para = filepath.Join(pasta, r.Nome)
		case "substituir":
			switch {
			case info.IsDir() && la.IsDir():
				filhos, _ := os.ReadDir(de)
				for _, f := range filhos {
					if !t.Segue() {
						return
					}
					e.coloca(t, filepath.Join(de, f.Name()), filepath.Join(para, f.Name()), false)
				}
				return
			case la.IsDir():
				t.Erro(para, "já há uma pasta com esse nome: uma pasta não é trocada por um arquivo")
				return
			case info.IsDir():
				if err := os.Remove(para); err != nil {
					t.Erro(para, err.Error())
					return
				}
			}
			t.NaoDesfaz()
			if err := os.Rename(de, para); err != nil {
				t.Erro(para, err.Error())
			} else if topo {
				t.Criou(para)
			}
			return
		default: // pular e cancelar
			return
		}
	}
	if err := os.Rename(de, para); err != nil {
		t.Erro(para, err.Error())
		return
	}
	if topo {
		t.Criou(para)
	}
	anotaArvore(t, para)
}

// --- compactar ---

type compactacao struct {
	origens []string
	destino string
	nome    string
	formato string
	nivel   int // -1: o padrão do formato
	senha   *string
}

func novaCompactacao(p *base.Pedido) (operar.Trabalho, error) {
	var q struct {
		Origens  []string `json:"origens"`
		Caminhos []string `json:"caminhos"`
		Destino  string   `json:"destino"`
		Nome     string   `json:"nome"`
		Formato  string   `json:"formato"`
		Nivel    *int     `json:"nivel"`
		Senha    *string  `json:"senha"`
	}
	if err := p.Ler(&q); err != nil {
		return nil, errors.New("pedido de compactar mal formado")
	}
	if len(q.Origens) == 0 {
		q.Origens = q.Caminhos
	}
	if err := base.Caminhos(q.Origens); err != nil {
		return nil, err
	}
	if err := base.Caminho(q.Destino); err != nil {
		return nil, err
	}
	if !ehPasta(q.Destino) {
		return nil, errors.New("o destino não é uma pasta")
	}
	if err := base.Nome(q.Nome); err != nil {
		return nil, &operar.DeCampo{Campo: "nome", Frase: err.Error()}
	}
	conhecido := false
	for _, f := range deCompactar {
		conhecido = conhecido || f == q.Formato
	}
	if !conhecido {
		return nil, &operar.DeCampo{Campo: "formato", Frase: "não sei compactar em «" + q.Formato + "»"}
	}
	c := &compactacao{origens: q.Origens, destino: q.Destino, nome: q.Nome, formato: q.Formato, nivel: -1, senha: q.Senha}
	if q.Nivel != nil {
		if *q.Nivel < 0 || *q.Nivel > 9 {
			return nil, &operar.DeCampo{Campo: "nivel", Frase: "o nível vai de 0 a 9"}
		}
		c.nivel = *q.Nivel
	}
	if q.Senha != nil {
		switch {
		case *q.Senha == "" || strings.ContainsAny(*q.Senha, "\n\r\x00"):
			return nil, &operar.DeCampo{Campo: "senha", Frase: "a senha não pode ser vazia nem ter quebra de linha"}
		case q.Formato != "zip" && q.Formato != "7z":
			return nil, &operar.DeCampo{Campo: "senha", Frase: "só zip e 7z aceitam senha"}
		case o7z() == "":
			return nil, erroCifradoZip
		}
	}
	if pacote := falta(q.Formato); pacote != "" {
		return nil, semFerramenta(pacote)
	}
	nomes := map[string]bool{}
	for _, o := range q.Origens {
		if o == "/" {
			return nil, errors.New("a raiz do sistema não se compacta")
		}
		if !existe(o) {
			return nil, errors.New("«" + filepath.Base(o) + "» não existe mais")
		}
		if nomes[filepath.Base(o)] {
			return nil, errors.New("há dois itens chamados «" + filepath.Base(o) + "»: compacte um de cada vez")
		}
		nomes[filepath.Base(o)] = true
	}
	if existe(filepath.Join(q.Destino, q.Nome)) {
		return nil, &operar.DeCampo{Campo: "nome", Frase: "já existe um item com esse nome"}
	}
	return c, nil
}

func (c *compactacao) quantos() string {
	if len(c.origens) == 1 {
		return "«" + filepath.Base(c.origens[0]) + "»"
	}
	return strconv.Itoa(len(c.origens)) + " itens"
}
func (c *compactacao) Titulo() string         { return "Compactando " + c.quantos() + " em «" + c.nome + "»" }
func (c *compactacao) Destino() string        { return c.destino }
func (c *compactacao) Descricao() string      { return "Compactar " + c.quantos() }
func (c *compactacao) Refaz() operar.Trabalho { n := *c; return &n }

// anda percorre as origens sem seguir links, pulando o próprio arquivo que
// está sendo escrito. dentro é o nome com que o item entra no compactado.
func (c *compactacao) anda(pular string, fn func(caminho, dentro string, info fs.FileInfo) error) error {
	final := filepath.Join(c.destino, c.nome)
	for _, o := range c.origens {
		topo := filepath.Dir(o)
		err := filepath.WalkDir(o, func(caminho string, d fs.DirEntry, err error) error {
			if err != nil {
				return fn(caminho, "", nil)
			}
			if caminho == pular || caminho == final {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return fn(caminho, "", nil)
			}
			dentro, _ := filepath.Rel(topo, caminho)
			return fn(caminho, filepath.ToSlash(dentro), info)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *compactacao) Prepara(t *operar.Tarefa) {
	itens, bytes := 0, int64(0)
	_ = c.anda("", func(_, _ string, info fs.FileInfo) error {
		if info != nil {
			itens++
			if info.Mode().IsRegular() {
				bytes += info.Size()
			}
		}
		return nil
	})
	t.Total(itens, bytes)
}

func (c *compactacao) Roda(t *operar.Tarefa) {
	final := filepath.Join(c.destino, c.nome)
	// O ".tmp" no fim: uma ferramenta de fora não inventa outra extensão.
	tmp := temporario(c.destino, marcaDeCompactar, "-"+c.nome+".tmp")
	if len(filepath.Base(tmp)) > 255 {
		tmp = temporario(c.destino, marcaDeCompactar, ".tmp")
	}
	defer os.Remove(tmp)
	ctx, cancela := comCancelamento(t)
	defer cancela()
	var err error
	if c.formato == "7z" || c.senha != nil {
		err = c.deFora(ctx, t, tmp)
	} else {
		err = c.grava(ctx, t, tmp)
	}
	switch {
	case t.Cancelada() || err == errCancelada:
		return
	case err != nil:
		t.Erro(final, err.Error())
		return
	}
	_ = os.Chmod(tmp, 0o644)
	// Sem passar por cima de nada: o nome pode ter sido ocupado enquanto isso.
	if err := os.Link(tmp, final); err != nil {
		if existe(final) {
			t.Erro(final, "já existe um item com esse nome")
			return
		}
		if err := os.Rename(tmp, final); err != nil {
			t.Erro(final, err.Error())
			return
		}
	}
	t.Criou(final)
	anota(t, final, false)
}

// deFora compacta com o 7z (7z, ou zip com senha). A senha vai pelo stdin.
func (c *compactacao) deFora(ctx context.Context, t *operar.Tarefa, tmp string) error {
	argv := []string{o7z(), "a", "-t" + c.formato, "-y", "-bd", "-snl", "-sccUTF-8"}
	if c.nivel >= 0 {
		argv = append(argv, "-mx="+strconv.Itoa(c.nivel))
	}
	if c.senha != nil {
		argv = append(argv, "-p")
		if c.formato == "7z" {
			argv = append(argv, "-mhe=on")
		}
	}
	argv = append(append(argv, "--", tmp), c.origens...)
	saida, err := roda(ctx, prazoDeExtrair, c.senha, c.destino, argv...)
	if ctx.Err() != nil {
		return errCancelada
	}
	if err != nil {
		linhas := strings.Split(strings.TrimSpace(saida), "\n")
		return errors.New("o 7z falhou: " + strings.TrimSpace(linhas[len(linhas)-1]))
	}
	if !existe(tmp) {
		return errors.New("o 7z não gravou o arquivo")
	}
	t.Avanca(len(c.origens), 0, "")
	return nil
}

// copiaContada copia o conteúdo de um arquivo para dentro do compactado,
// parando se a tarefa for cancelada. Devolve quantos bytes foram.
func copiaContada(t *operar.Tarefa, para io.Writer, de io.Reader, atual string) (int64, error) {
	bloco := make([]byte, tamanhoDoBloco)
	var total int64
	for {
		if !t.Segue() {
			return total, errCancelada
		}
		n, err := de.Read(bloco)
		if n > 0 {
			if _, e := para.Write(bloco[:n]); e != nil {
				return total, e
			}
			total += int64(n)
			t.Avanca(0, int64(n), atual)
			if aCadaBloco != nil {
				aCadaBloco()
			}
		}
		if err == io.EOF {
			return total, nil
		}
		if err != nil {
			return total, err
		}
	}
}

// grava escreve o zip ou o tar (com a casca dele) no temporário.
func (c *compactacao) grava(ctx context.Context, t *operar.Tarefa, tmp string) (err error) {
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		if e := f.Close(); err == nil {
			err = e
		}
	}()
	if c.formato == "zip" {
		return c.gravaZip(t, f, tmp)
	}
	// A casca do tar: gzip daqui, ou zstd/xz de fora, em pipe.
	var casca io.WriteCloser
	espera := func() error { return nil }
	switch c.formato {
	case "tar.gz":
		nivel := gzip.DefaultCompression
		if c.nivel >= 0 {
			nivel = c.nivel
		}
		casca, _ = gzip.NewWriterLevel(f, nivel)
	default:
		comando, pacote := oZstd(), "zstd"
		if c.formato == "tar.xz" {
			comando, pacote = oXz(), "xz"
		}
		argv := []string{"-c"}
		if c.nivel >= 0 {
			nivel := c.nivel
			if pacote == "zstd" {
				nivel = max(nivel, 1)
			}
			argv = append(argv, "-"+strconv.Itoa(nivel))
		}
		cmd := exec.CommandContext(ctx, comando, argv...)
		cmd.Stdout = f
		cmd.WaitDelay = 2 * time.Second
		entrada, e := cmd.StdinPipe()
		if e == nil {
			e = cmd.Start()
		}
		if e != nil {
			return errors.New("o " + pacote + " não rodou: " + e.Error())
		}
		casca = entrada
		espera = func() error {
			if e := cmd.Wait(); e != nil && ctx.Err() == nil {
				return errors.New("o " + pacote + " falhou: " + e.Error())
			}
			return nil
		}
	}
	tw := tar.NewWriter(casca)
	err = c.anda(tmp, func(caminho, dentro string, info fs.FileInfo) error {
		if !t.Segue() {
			return errCancelada
		}
		if info == nil {
			t.Erro(caminho, "não deu para ler: ficou de fora")
			return nil
		}
		modo := info.Mode()
		alvo := ""
		switch {
		case modo&fs.ModeSymlink != 0:
			alvo, _ = os.Readlink(caminho)
		case !modo.IsDir() && !modo.IsRegular():
			t.Erro(caminho, "é um dispositivo ou um cano: ficou de fora")
			t.Avanca(1, 0, caminho)
			return nil
		}
		h, e := tar.FileInfoHeader(info, alvo)
		if e != nil {
			t.Erro(caminho, e.Error())
			return nil
		}
		h.Name = dentro
		if modo.IsDir() {
			h.Name += "/"
		}
		var arq *os.File
		if modo.IsRegular() {
			if arq, e = os.Open(caminho); e != nil {
				t.Erro(caminho, "não deu para ler: ficou de fora")
				t.Avanca(1, 0, caminho)
				return nil
			}
			defer arq.Close()
		}
		if e := tw.WriteHeader(h); e != nil {
			return e
		}
		if arq != nil {
			n, e := copiaContada(t, tw, io.LimitReader(arq, h.Size), caminho)
			if e != nil {
				return e
			}
			if n < h.Size {
				// O arquivo encolheu no meio: o tar precisa do tamanho prometido.
				if _, e := io.CopyN(tw, zeros{}, h.Size-n); e != nil {
					return e
				}
				t.Erro(caminho, "mudou durante a leitura: pode ter ficado incompleto")
			}
		}
		t.Avanca(1, 0, caminho)
		return nil
	})
	if e := tw.Close(); err == nil {
		err = e
	}
	if e := casca.Close(); err == nil {
		err = e
	}
	if e := espera(); err == nil {
		err = e
	}
	return err
}

type zeros struct{}

func (zeros) Read(b []byte) (int, error) {
	clear(b)
	return len(b), nil
}

func (c *compactacao) gravaZip(t *operar.Tarefa, f *os.File, tmp string) error {
	zw := zip.NewWriter(f)
	metodo := zip.Deflate
	if c.nivel == 0 {
		metodo = zip.Store
	} else if c.nivel > 0 {
		nivel := c.nivel
		zw.RegisterCompressor(zip.Deflate, func(w io.Writer) (io.WriteCloser, error) { return flate.NewWriter(w, nivel) })
	}
	err := c.anda(tmp, func(caminho, dentro string, info fs.FileInfo) error {
		if !t.Segue() {
			return errCancelada
		}
		if info == nil {
			t.Erro(caminho, "não deu para ler: ficou de fora")
			return nil
		}
		modo := info.Mode()
		if !modo.IsDir() && !modo.IsRegular() && modo&fs.ModeSymlink == 0 {
			t.Erro(caminho, "é um dispositivo ou um cano: ficou de fora")
			t.Avanca(1, 0, caminho)
			return nil
		}
		h, e := zip.FileInfoHeader(info)
		if e != nil {
			t.Erro(caminho, e.Error())
			return nil
		}
		h.Name, h.Method = dentro, metodo
		switch {
		case modo.IsDir():
			h.Name += "/"
			h.Method = zip.Store
			_, e = zw.CreateHeader(h)
		case modo&fs.ModeSymlink != 0:
			// Um link é guardado como link: o conteúdo é o alvo.
			alvo, _ := os.Readlink(caminho)
			h.Method = zip.Store
			var w io.Writer
			if w, e = zw.CreateHeader(h); e == nil {
				_, e = io.WriteString(w, alvo)
			}
		default:
			arq, erro := os.Open(caminho)
			if erro != nil {
				t.Erro(caminho, "não deu para ler: ficou de fora")
				t.Avanca(1, 0, caminho)
				return nil
			}
			defer arq.Close()
			var w io.Writer
			if w, e = zw.CreateHeader(h); e == nil {
				_, e = copiaContada(t, w, arq, caminho)
			}
		}
		if e != nil {
			return e
		}
		t.Avanca(1, 0, caminho)
		return nil
	})
	if e := zw.Close(); err == nil {
		err = e
	}
	return err
}
