package operar

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"mkfiles/internal/base"
	"mkfiles/internal/lixeira"
)

func init() {
	Registrar("copiar", func(p *base.Pedido) (Trabalho, error) {
		q, err := comDestino(p, false)
		if err != nil {
			return nil, err
		}
		return &copia{origens: q.Origens, destino: q.Destino}, nil
	})
	Registrar("mover", func(p *base.Pedido) (Trabalho, error) {
		q, err := comDestino(p, true)
		if err != nil {
			return nil, err
		}
		return &mudanca{origens: q.Origens, destino: q.Destino}, nil
	})
	Registrar("ligar", func(p *base.Pedido) (Trabalho, error) {
		q := ler(p)
		if err := base.Caminhos(q.Origens); err != nil {
			return nil, err
		}
		if err := pasta(q.Destino); err != nil {
			return nil, err
		}
		return &ligacao{origens: q.Origens, destino: q.Destino}, nil
	})
	Registrar("criar-pasta", func(p *base.Pedido) (Trabalho, error) {
		q := ler(p)
		if err := nomeNovo(q.Destino, q.Nome); err != nil {
			return nil, err
		}
		return &criacao{destino: q.Destino, nome: q.Nome, pasta: true}, nil
	})
	Registrar("criar-arquivo", func(p *base.Pedido) (Trabalho, error) {
		q := ler(p)
		if err := nomeNovo(q.Destino, q.Nome); err != nil {
			return nil, err
		}
		c := &criacao{destino: q.Destino, nome: q.Nome}
		if q.Modelo != nil && *q.Modelo != "" {
			if err := base.Caminho(*q.Modelo); err != nil {
				return nil, err
			}
			if info, err := os.Stat(*q.Modelo); err != nil || !info.Mode().IsRegular() {
				return nil, errors.New("o modelo não é um arquivo")
			}
			c.modelo = *q.Modelo
		}
		return c, nil
	})
	Registrar("renomear", func(p *base.Pedido) (Trabalho, error) {
		q := ler(p)
		if err := mexivel(q.Caminho); err != nil {
			return nil, err
		}
		if err := base.Nome(q.Nome); err != nil {
			return nil, &DeCampo{"nome", err.Error()}
		}
		if !existe(q.Caminho) {
			return nil, errors.New("isso não existe mais")
		}
		novo := filepath.Join(filepath.Dir(q.Caminho), q.Nome)
		if novo == q.Caminho {
			return nil, &DeCampo{"nome", "o nome é o mesmo"}
		}
		if existe(novo) {
			return nil, &DeCampo{"nome", "já existe um item com esse nome"}
		}
		return &renomeacao{de: q.Caminho, para: novo}, nil
	})
	Registrar("lixeira", func(p *base.Pedido) (Trabalho, error) {
		q := ler(p)
		if err := mexiveis(q.Origens); err != nil {
			return nil, err
		}
		return &lixada{origens: q.Origens}, nil
	})
	Registrar("apagar", func(p *base.Pedido) (Trabalho, error) {
		q := ler(p)
		if err := mexiveis(q.Origens); err != nil {
			return nil, err
		}
		return &apagada{origens: q.Origens}, nil
	})
	daLixeira := func(p *base.Pedido) ([]lixeira.Entrada, error) {
		q := ler(p)
		if len(q.Itens) == 0 {
			return nil, errors.New("nenhum item")
		}
		var es []lixeira.Entrada
		for _, c := range q.Itens {
			e, err := lixeira.Ler(c)
			if err != nil {
				return nil, err
			}
			es = append(es, e)
		}
		return es, nil
	}
	Registrar("restaurar", func(p *base.Pedido) (Trabalho, error) {
		es, err := daLixeira(p)
		return &restauro{itens: es}, err
	})
	Registrar("lixeira-apagar", func(p *base.Pedido) (Trabalho, error) {
		es, err := daLixeira(p)
		return &limpeza{itens: es}, err
	})
	Registrar("lixeira-esvaziar", func(p *base.Pedido) (Trabalho, error) {
		return &limpeza{tudo: true}, nil
	})
}

type campos struct {
	Origens  []string `json:"origens"`
	Itens    []string `json:"itens"`
	Caminhos []string `json:"caminhos"`
	Destino  string   `json:"destino"`
	Caminho  string   `json:"caminho"`
	Nome     string   `json:"nome"`
	Modelo   *string  `json:"modelo"`
}

// ler lê os campos de um pedido. Na chamada de um subcomando só os caminhos
// soltos chegam em "caminhos": valem de "origens" e de "itens".
func ler(p *base.Pedido) campos {
	var q campos
	_ = p.Ler(&q)
	if q.Origens == nil {
		q.Origens = q.Caminhos
	}
	if q.Itens == nil {
		q.Itens = q.Caminhos
	}
	return q
}

// mexivel: o caminho é válido e não é dos que o motor nunca mexe.
func mexivel(caminho string) error {
	if err := base.Caminho(caminho); err != nil {
		return err
	}
	if err := base.Protegido(caminho); err != nil {
		return &Recusa{err.Error()}
	}
	return nil
}

func mexiveis(caminhos []string) error {
	if len(caminhos) == 0 {
		return errors.New("nenhum caminho")
	}
	// Primeiro a forma de todos, depois a segurança: um pedido torto é erro,
	// não recusa.
	if err := base.Caminhos(caminhos); err != nil {
		return err
	}
	for _, c := range caminhos {
		if err := mexivel(c); err != nil {
			return err
		}
	}
	return nil
}

func pasta(caminho string) error {
	if err := base.Caminho(caminho); err != nil {
		return err
	}
	if info, err := os.Stat(caminho); err != nil || !info.IsDir() {
		if f, caiu := base.ConexaoCaiu(caminho, err); caiu {
			return errors.New(f)
		}
		return errors.New("o destino não é uma pasta: «" + caminho + "»")
	}
	return nil
}

func nomeNovo(destino, nome string) error {
	if err := pasta(destino); err != nil {
		return err
	}
	if err := base.Nome(nome); err != nil {
		return &DeCampo{"nome", err.Error()}
	}
	if existe(filepath.Join(destino, nome)) {
		return &DeCampo{"nome", "já existe um item com esse nome"}
	}
	return nil
}

// dentroDeSi: a origem é uma pasta e o destino é ela mesma ou fica dentro
// dela. Compara os caminhos de verdade: um link no meio não engana.
func dentroDeSi(origem, destino string) bool {
	info, err := os.Lstat(origem)
	if err != nil || !info.IsDir() {
		return false
	}
	real := func(c string) string {
		if r, err := filepath.EvalSymlinks(c); err == nil {
			return r
		}
		return c
	}
	return base.Dentro(real(origem), real(destino))
}

// comDestino lê e confere um copiar ou mover.
func comDestino(p *base.Pedido, movendo bool) (campos, error) {
	q := ler(p)
	if err := base.Caminhos(q.Origens); err != nil {
		return q, err
	}
	if err := pasta(q.Destino); err != nil {
		return q, err
	}
	verbo := "copiar"
	if movendo {
		verbo = "mover"
	}
	for _, o := range q.Origens {
		if movendo {
			if err := mexivel(o); err != nil {
				return q, err
			}
			if filepath.Dir(o) == q.Destino {
				return q, errors.New("«" + filepath.Base(o) + "» já está nessa pasta")
			}
		}
		if dentroDeSi(o, q.Destino) {
			return q, errors.New("não dá para " + verbo + " uma pasta para dentro dela mesma")
		}
	}
	return q, nil
}

// quantos escreve "«a.txt»" ou "3 itens".
func quantos(caminhos []string) string {
	if len(caminhos) == 1 {
		return "«" + filepath.Base(caminhos[0]) + "»"
	}
	return strconv.Itoa(len(caminhos)) + " itens"
}

func nomeDaPasta(caminho string) string {
	if caminho == "/" {
		return "a raiz"
	}
	return filepath.Base(caminho)
}

func (t *Tarefa) totalDe(caminhos []string) {
	for _, c := range caminhos {
		t.Total(conta(c))
	}
}

// --- copiar ---

type copia struct {
	origens []string
	destino string
}

func (c *copia) Titulo() string {
	return "Copiando " + quantos(c.origens) + " para " + nomeDaPasta(c.destino)
}
func (c *copia) Destino() string   { return c.destino }
func (c *copia) Descricao() string { return "Copiar " + quantos(c.origens) }
func (c *copia) Refaz() Trabalho   { n := *c; return &n }
func (c *copia) Prepara(t *Tarefa) { t.totalDe(c.origens) }

func (c *copia) Roda(t *Tarefa) {
	for _, o := range c.origens {
		if !t.Segue() {
			return
		}
		para := filepath.Join(c.destino, filepath.Base(o))
		if filepath.Dir(o) == c.destino {
			// Copiar para a própria pasta é duplicar: nome novo, sem perguntar.
			info, err := os.Lstat(o)
			if err != nil {
				t.Erro(o, frase(err))
				continue
			}
			para = filepath.Join(c.destino, livreEm(c.destino, filepath.Base(o), info.IsDir()))
		}
		t.copiaItem(o, para, true)
	}
}

// copiaItem copia um item para um destino que pode já existir (aí pergunta).
func (t *Tarefa) copiaItem(de, para string, topo bool) {
	info, err := os.Lstat(de)
	if err != nil {
		t.Erro(de, frase(err))
		return
	}
	para, como := t.resolve(de, para, info, true)
	switch como {
	case pula:
		t.pulou(de, de)
		return
	case mescla:
		entradas, err := os.ReadDir(de)
		if err != nil {
			t.Erro(de, frase(err))
		}
		for _, e := range entradas {
			if !t.Segue() {
				return
			}
			t.copiaItem(filepath.Join(de, e.Name()), filepath.Join(para, e.Name()), false)
		}
		t.Avanca(1, 0, de)
	case substitui:
		t.NaoDesfaz()
		tmp := aoLado(para)
		antes := t.nErros()
		if t.copiaArvore(de, tmp, false) || t.nErros() > antes {
			removeArvore(tmp, nil)
			return
		}
		if err := troca(tmp, para); err != nil {
			removeArvore(tmp, nil)
			t.Erro(para, frase(err))
			return
		}
	default:
		antes := t.nErros()
		if t.copiaArvore(de, para, true) || (t.nErros() > antes && !existe(para)) {
			return
		}
	}
	if topo {
		t.Criou(para)
	}
}

// pulou conta como feito o que não vai ser feito (um item pulado).
func (t *Tarefa) pulou(arvore, atual string) {
	itens, bytes := conta(arvore)
	t.Avanca(itens, bytes, atual)
}

// --- mover ---

type mudanca struct {
	origens []string
	destino string
}

func (m *mudanca) Titulo() string {
	return "Movendo " + quantos(m.origens) + " para " + nomeDaPasta(m.destino)
}
func (m *mudanca) Destino() string   { return m.destino }
func (m *mudanca) Descricao() string { return "Mover " + quantos(m.origens) }
func (m *mudanca) Refaz() Trabalho   { n := *m; return &n }
func (m *mudanca) Prepara(t *Tarefa) { t.totalDe(m.origens) }

func (m *mudanca) Roda(t *Tarefa) {
	for _, o := range m.origens {
		if !t.Segue() {
			return
		}
		t.moveItem(o, filepath.Join(m.destino, filepath.Base(o)), true)
	}
}

func (t *Tarefa) moveItem(de, para string, topo bool) {
	info, err := os.Lstat(de)
	if err != nil {
		t.Erro(de, frase(err))
		return
	}
	if err := base.Protegido(de); err != nil {
		t.Erro(de, err.Error())
		return
	}
	para, como := t.resolve(de, para, info, true)
	switch como {
	case pula:
		t.pulou(de, de)
		return
	case mescla:
		entradas, err := os.ReadDir(de)
		if err != nil {
			t.Erro(de, frase(err))
		}
		for _, e := range entradas {
			if !t.Segue() {
				return
			}
			t.moveItem(filepath.Join(de, e.Name()), filepath.Join(para, e.Name()), false)
		}
		_ = os.Remove(de) // sai se ficou vazia; o que foi pulado continua nela
		t.Avanca(1, 0, de)
	case substitui:
		t.NaoDesfaz()
		if !t.move(de, para, true) {
			return
		}
	default:
		if !t.move(de, para, false) {
			return
		}
		t.fez(feito{tipo: fMovido, de: de, caminho: para})
	}
	if topo {
		t.Criou(para)
	}
}

// --- criar link ---

type ligacao struct {
	origens []string
	destino string
}

func (l *ligacao) Titulo() string    { return "Criando link para " + quantos(l.origens) }
func (l *ligacao) Destino() string   { return l.destino }
func (l *ligacao) Descricao() string { return "Criar link para " + quantos(l.origens) }
func (l *ligacao) Refaz() Trabalho   { n := *l; return &n }
func (l *ligacao) Prepara(t *Tarefa) { t.Total(len(l.origens), 0) }

func (l *ligacao) Roda(t *Tarefa) {
	for _, o := range l.origens {
		if !t.Segue() {
			return
		}
		t.ligaItem(o, l.destino)
		t.Avanca(1, 0, o)
	}
}

func (t *Tarefa) ligaItem(o, destino string) {
	info, err := os.Lstat(o)
	if err != nil {
		t.Erro(o, frase(err))
		return
	}
	para := filepath.Join(destino, filepath.Base(o))
	if filepath.Dir(o) == destino {
		para = filepath.Join(destino, livreEm(destino, filepath.Base(o), info.IsDir()))
	}
	para, como := t.resolve(o, para, info, false)
	switch como {
	case pula:
		return
	case substitui:
		t.NaoDesfaz()
		tmp := aoLado(para)
		err := os.Symlink(o, tmp)
		if err == nil {
			err = troca(tmp, para)
		}
		if err != nil {
			os.Remove(tmp)
			t.Erro(para, frase(err))
			return
		}
	default:
		if err := os.Symlink(o, para); err != nil {
			t.Erro(para, frase(err))
			return
		}
		t.fezArquivo(para)
	}
	t.Criou(para)
}

// --- criar pasta e arquivo ---

type criacao struct {
	destino, nome string
	pasta         bool
	modelo        string
}

func (c *criacao) Titulo() string {
	if c.pasta {
		return "Criando a pasta «" + c.nome + "»"
	}
	return "Criando «" + c.nome + "»"
}
func (c *criacao) Destino() string { return c.destino }
func (c *criacao) Descricao() string {
	if c.pasta {
		return "Criar a pasta «" + c.nome + "»"
	}
	return "Criar «" + c.nome + "»"
}
func (c *criacao) Refaz() Trabalho   { n := *c; return &n }
func (c *criacao) Prepara(t *Tarefa) { t.Total(1, 0) }

func (c *criacao) Roda(t *Tarefa) {
	para := filepath.Join(c.destino, c.nome)
	defer t.Avanca(1, 0, para)
	switch {
	case c.pasta:
		if err := os.Mkdir(para, 0o777); err != nil {
			t.Erro(para, frase(err))
			return
		}
		t.fez(feito{tipo: fPasta, caminho: para})
	case c.modelo != "":
		info, err := os.Stat(c.modelo)
		if err == nil {
			err = t.copiaArquivo(c.modelo, para, info)
		}
		if err == errCancelada {
			return
		}
		if err != nil {
			t.Erro(para, frase(err))
			return
		}
		agora := time.Now()
		_ = os.Chtimes(para, agora, agora)
		t.fezArquivo(para)
	default:
		f, err := os.OpenFile(para, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o666)
		if err != nil {
			t.Erro(para, frase(err))
			return
		}
		f.Close()
		t.fezArquivo(para)
	}
	t.Criou(para)
}

// --- renomear ---

type renomeacao struct{ de, para string }

func (r *renomeacao) Titulo() string {
	return "Renomeando «" + filepath.Base(r.de) + "» para «" + filepath.Base(r.para) + "»"
}
func (r *renomeacao) Destino() string   { return filepath.Dir(r.para) }
func (r *renomeacao) Descricao() string { return "Renomear «" + filepath.Base(r.de) + "»" }
func (r *renomeacao) Refaz() Trabalho   { n := *r; return &n }
func (r *renomeacao) Prepara(t *Tarefa) { t.Total(1, 0) }

func (r *renomeacao) Roda(t *Tarefa) {
	defer t.Avanca(1, 0, r.para)
	if existe(r.para) {
		t.Erro(r.para, "já existe um item com esse nome")
		return
	}
	if err := renomeia(r.de, r.para); err != nil {
		t.Erro(r.de, frase(err))
		return
	}
	t.fez(feito{tipo: fMovido, de: r.de, caminho: r.para})
	t.Criou(r.para)
}

// --- mandar para a lixeira ---

type lixada struct{ origens []string }

func (l *lixada) Titulo() string    { return "Mandando " + quantos(l.origens) + " para a lixeira" }
func (l *lixada) Destino() string   { return "" }
func (l *lixada) Descricao() string { return "Mandar " + quantos(l.origens) + " para a lixeira" }
func (l *lixada) Refaz() Trabalho   { n := *l; return &n }
func (l *lixada) Prepara(t *Tarefa) { t.Total(len(l.origens), 0) }

func (l *lixada) Roda(t *Tarefa) {
	defer t.tr.avisaLixeira()
	for _, o := range l.origens {
		if !t.Segue() {
			return
		}
		// Uma pasta com um disco montado dentro não vai para a lixeira (e
		// apagar de vez também não pode): é erro, não "sem lixeira".
		if ponto := montadoDentro(o); ponto != "" {
			t.Erro(o, outroDiscoEm(ponto).Error())
			t.Avanca(1, 0, o)
			continue
		}
		e, err := lixeira.Mandar(o, renomeia)
		switch {
		case errors.Is(err, lixeira.ErroSemLixeira), errors.Is(err, syscall.EXDEV):
			t.SemLixeira(o)
		case err != nil:
			t.Erro(o, frase(err))
		default:
			t.fez(feito{tipo: fLixado, de: o, caminho: e.Arquivo})
		}
		t.Avanca(1, 0, o)
	}
}

// --- apagar de vez ---

type apagada struct{ origens []string }

func (a *apagada) Titulo() string    { return "Apagando " + quantos(a.origens) }
func (a *apagada) Destino() string   { return "" }
func (a *apagada) Prepara(t *Tarefa) { t.totalDe(a.origens) }

func (a *apagada) Roda(t *Tarefa) {
	for _, o := range a.origens {
		if !t.Segue() || !t.apaga(o, true) {
			return
		}
	}
}

// --- restaurar da lixeira ---

type restauro struct{ itens []lixeira.Entrada }

func (r *restauro) Titulo() string {
	if len(r.itens) == 1 {
		return "Restaurando «" + filepath.Base(r.itens[0].Origem) + "»"
	}
	return "Restaurando " + strconv.Itoa(len(r.itens)) + " itens"
}
func (r *restauro) Destino() string { return "" }
func (r *restauro) Prepara(t *Tarefa) {
	for _, e := range r.itens {
		t.Total(conta(e.Arquivo))
	}
}

func (r *restauro) Roda(t *Tarefa) {
	defer t.tr.avisaLixeira()
	for _, e := range r.itens {
		if !t.Segue() {
			return
		}
		t.restaura(e, true)
	}
}

// restaura devolve um item da lixeira ao lugar de origem. perguntando: lugar
// ocupado é conflito; sem perguntar (o desfazer), é erro.
func (t *Tarefa) restaura(e lixeira.Entrada, perguntando bool) bool {
	info, err := os.Lstat(e.Arquivo)
	if err != nil {
		t.Erro(e.Origem, "não está mais na lixeira")
		return false
	}
	para, como := e.Origem, livre
	if perguntando {
		para, como = t.resolve(e.Arquivo, e.Origem, info, false)
	} else if existe(para) {
		t.Erro(para, "o lugar de origem voltou a ser ocupado")
		return false
	}
	if como == pula {
		t.pulou(e.Arquivo, e.Origem)
		return false
	}
	if err := os.MkdirAll(filepath.Dir(para), 0o777); err != nil {
		t.Erro(para, frase(err))
		return false
	}
	if !t.move(e.Arquivo, para, como == substitui) {
		return false
	}
	os.Remove(e.Info)
	t.Criou(para)
	return true
}

// --- apagar da lixeira e esvaziar ---

type limpeza struct {
	itens []lixeira.Entrada
	tudo  bool
	// O que o Prepara achou para o esvaziar.
	arquivos, fichas []string
}

func (l *limpeza) Titulo() string {
	if l.tudo {
		return "Esvaziando a lixeira"
	}
	return "Apagando " + strconv.Itoa(len(l.itens)) + " da lixeira"
}
func (l *limpeza) Destino() string { return "" }

func (l *limpeza) Prepara(t *Tarefa) {
	if l.tudo {
		for _, lx := range lixeira.Todas() {
			arquivos, fichas := lixeira.Conteudo(lx)
			l.arquivos = append(l.arquivos, arquivos...)
			l.fichas = append(l.fichas, fichas...)
		}
	}
	for _, e := range l.itens {
		l.arquivos = append(l.arquivos, e.Arquivo)
		l.fichas = append(l.fichas, e.Info)
	}
	t.totalDe(l.arquivos)
}

func (l *limpeza) Roda(t *Tarefa) {
	defer t.tr.avisaLixeira()
	for _, a := range l.arquivos {
		if !t.Segue() || !t.apaga(a, true) {
			return
		}
	}
	for _, f := range l.fichas {
		// A ficha de um arquivo que não saiu fica, para ele continuar à vista.
		arquivo := filepath.Join(filepath.Dir(filepath.Dir(f)), "files", filepath.Base(f[:len(f)-len(filepath.Ext(f))]))
		if !existe(arquivo) {
			os.Remove(f)
		}
	}
}

// --- desfazer ---

type desfaz struct{ reg *registro }

func (d *desfaz) Titulo() string  { return "Desfazendo: " + d.reg.descricao }
func (d *desfaz) Destino() string { return "" }
func (d *desfaz) Prepara(t *Tarefa) {
	for _, f := range d.reg.feitos {
		if f.tipo == fLote {
			t.Total(len(f.pares), 0)
		} else {
			t.Total(1, 0)
		}
	}
}

func (d *desfaz) Roda(t *Tarefa) {
	mexeuNaLixeira := false
	for i := len(d.reg.feitos) - 1; i >= 0; i-- {
		if !t.Segue() {
			break
		}
		f := d.reg.feitos[i]
		switch f.tipo {
		case fArquivo:
			info, err := os.Lstat(f.caminho)
			switch {
			case err != nil: // já não está lá
			case info.IsDir() || info.Size() != f.tamanho || info.ModTime().UnixNano() != f.mtime:
				t.Erro(f.caminho, "mudou desde então: ficou onde está")
			default:
				if err := os.Remove(f.caminho); err != nil {
					t.Erro(f.caminho, frase(err))
				}
			}
		case fPasta:
			if info, err := os.Lstat(f.caminho); err == nil && info.IsDir() {
				if err := os.Remove(f.caminho); err != nil {
					t.Erro(f.caminho, "a pasta não está mais vazia: ficou onde está")
				}
			}
		case fMovido:
			switch {
			case !existe(f.caminho):
				t.Erro(f.caminho, "não está mais onde foi posto")
			case base.Protegido(f.caminho) != nil:
				t.Erro(f.caminho, "não pode ser mexido")
			case existe(f.de):
				t.Erro(f.de, "o lugar de origem voltou a ser ocupado")
			default:
				if err := os.MkdirAll(filepath.Dir(f.de), 0o777); err != nil {
					t.Erro(f.de, frase(err))
				} else if t.move(f.caminho, f.de, false) {
					t.Criou(f.de)
				}
				continue // o move já contou
			}
		case fLote:
			// O lote volta inteiro, como foi: os pares ao contrário.
			volta := make([][2]string, len(f.pares))
			for i, par := range f.pares {
				volta[i] = [2]string{par[1], par[0]}
			}
			t.renomeiaLote(volta)
			continue // o lote já contou
		case fLixado:
			mexeuNaLixeira = true
			e, err := lixeira.Ler(f.caminho)
			if err != nil {
				t.Erro(f.de, "não está mais na lixeira")
			} else if t.restaura(e, false) {
				continue // o move já contou
			}
		}
		t.Avanca(1, 0, f.caminho)
	}
	if mexeuNaLixeira {
		t.tr.avisaLixeira()
	}
}
