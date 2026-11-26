// Package lixeira é a lixeira do freedesktop: a da pessoa
// ($XDG_DATA_HOME/Trash) e a de cada outro sistema de arquivos montado
// (<ponto>/.Trash-<uid>), com um .trashinfo por item. Atende os pedidos
// lixeira-listar e lixeira-estado; mandar, restaurar e apagar são tarefas do
// pacote operar, que usa o que este exporta. Veja ../../../CONTRATO.md.
package lixeira

import (
	"encoding/json"
	"errors"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"mkfiles/internal/base"
	"mkfiles/internal/item"
)

// ErroSemLixeira: não há lixeira possível no disco daquele arquivo.
var ErroSemLixeira = errors.New("não há lixeira nesse disco")

// Lixeira é uma pasta de lixeira: Raiz tem files/ e info/. Ponto é o ponto de
// montagem a que ela pertence ("" na lixeira da pessoa: lá o Path é absoluto).
type Lixeira struct {
	Raiz  string
	Ponto string
}

// Entrada é um item que está numa lixeira.
type Entrada struct {
	Arquivo  string // o arquivo dentro de files/
	Info     string // o .trashinfo dele
	Origem   string // de onde veio
	ApagadoS int64  // quando (0 se o .trashinfo não diz)
	Lixeira  string // a raiz da lixeira
}

const formatoDaData = "2006-01-02T15:04:05"

// DaCasa é a lixeira da pessoa.
func DaCasa() Lixeira { return Lixeira{Raiz: base.Dados("Trash")} }

func doPonto(ponto string) Lixeira {
	return Lixeira{Raiz: filepath.Join(ponto, ".Trash-"+strconv.Itoa(os.Getuid())), Ponto: ponto}
}

// ePasta: uma pasta de verdade (um link no lugar não serve).
func ePasta(caminho string) bool {
	info, err := os.Lstat(caminho)
	return err == nil && info.IsDir()
}

// Para escolhe a lixeira de um arquivo e garante que ela existe: a da pessoa,
// se o arquivo está no mesmo ponto de montagem que ela; senão a do ponto dele.
// Sem como criar, ErroSemLixeira.
func Para(caminho string) (Lixeira, error) {
	if base.Home() == "" {
		return Lixeira{}, ErroSemLixeira
	}
	// Num remoto (gvfs) não há lixeira: a janela pergunta e apaga de vez.
	if _, _, remoto := base.Remoto(caminho); remoto {
		return Lixeira{}, ErroSemLixeira
	}
	l := DaCasa()
	if ponto := base.PontoDe(caminho); ponto != base.PontoDe(l.Raiz) {
		l = doPonto(ponto)
		if info, err := os.Lstat(l.Raiz); err == nil && !info.IsDir() {
			return Lixeira{}, ErroSemLixeira
		}
	}
	for _, sub := range []string{"files", "info"} {
		if err := os.MkdirAll(filepath.Join(l.Raiz, sub), 0o700); err != nil {
			return Lixeira{}, ErroSemLixeira
		}
	}
	return l, nil
}

// Todas são as lixeiras da pessoa que existem: a de casa e as dos pontos de
// montagem.
func Todas() []Lixeira {
	var todas []Lixeira
	vistas := map[string]bool{}
	junta := func(l Lixeira) {
		if !vistas[l.Raiz] && ePasta(l.Raiz) {
			vistas[l.Raiz] = true
			todas = append(todas, l)
		}
	}
	if base.Home() != "" {
		junta(DaCasa())
	}
	gvfs := base.GVFS()
	for _, m := range base.Montagens() {
		if gvfs != "" && base.Dentro(gvfs, m.Ponto) {
			continue // remotos não têm lixeira
		}
		junta(doPonto(m.Ponto))
	}
	return todas
}

var numerado = regexp.MustCompile(`^(.*) \((\d{1,9})\)$`)

// Numerado acha um nome livre a partir de nome: "foto (2).jpg", "foto
// (3).jpg"… O número vai antes da extensão (em ".tar.gz" e parentes, antes do
// ".tar"); numa pasta, no fim. ocupado diz se um nome já existe.
func Numerado(nome string, pasta bool, ocupado func(string) bool) string {
	raiz, ext := nome, ""
	if !pasta {
		if i := strings.LastIndexByte(nome, '.'); i > 0 {
			raiz, ext = nome[:i], nome[i:]
			if j := len(raiz) - 4; j > 0 && strings.EqualFold(raiz[j:], ".tar") {
				raiz, ext = raiz[:j], raiz[j:]+ext
			}
		}
	}
	n := 2
	if m := numerado.FindStringSubmatch(raiz); m != nil {
		if v, err := strconv.Atoi(m[2]); err == nil && v >= 2 {
			raiz, n = m[1], v+1
		}
	}
	for ; ; n++ {
		sufixo := " (" + strconv.Itoa(n) + ")" + ext
		if sobra := 255 - len(sufixo); len(raiz) > sobra {
			// Sem espaço para o número: encurta a raiz.
			raiz = strings.ToValidUTF8(raiz[:max(sobra, 0)], "")
		}
		candidato := raiz + sufixo
		if !ocupado(candidato) {
			return candidato
		}
	}
}

// Mandar põe um arquivo na lixeira dele: cria o .trashinfo (O_EXCL; nome
// ocupado ganha " (2)") e só então move. mover é quem move (nil: os.Rename).
func Mandar(caminho string, mover func(de, para string) error) (Entrada, error) {
	if err := base.Protegido(caminho); err != nil {
		return Entrada{}, err
	}
	info, err := os.Lstat(caminho)
	if err != nil {
		return Entrada{}, err
	}
	l, err := Para(caminho)
	if err != nil {
		return Entrada{}, err
	}
	if base.Dentro(caminho, l.Raiz) || base.Dentro(l.Raiz, caminho) {
		return Entrada{}, errors.New("isso é a lixeira, ou já está nela")
	}
	if mover == nil {
		mover = os.Rename
	}
	origem := caminho
	if l.Ponto != "" {
		origem = strings.TrimPrefix(strings.TrimPrefix(caminho, l.Ponto), "/")
	}
	agora := time.Now()
	texto := "[Trash Info]\nPath=" + base.CodificaCaminho(origem) + "\nDeletionDate=" + agora.Format(formatoDaData) + "\n"
	nome := filepath.Base(caminho)
	for {
		e := Entrada{Arquivo: filepath.Join(l.Raiz, "files", nome), Info: filepath.Join(l.Raiz, "info", nome+".trashinfo"),
			Origem: caminho, ApagadoS: agora.Unix(), Lixeira: l.Raiz}
		f, err := os.OpenFile(e.Info, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if errors.Is(err, fs.ErrExist) {
			nome = Numerado(nome, info.IsDir(), func(n string) bool {
				_, err := os.Lstat(filepath.Join(l.Raiz, "info", n+".trashinfo"))
				return err == nil
			})
			continue
		}
		if err != nil {
			return Entrada{}, err
		}
		_, err = f.WriteString(texto)
		if e2 := f.Close(); err == nil {
			err = e2
		}
		if err == nil {
			if _, e3 := os.Lstat(e.Arquivo); e3 == nil {
				err = errors.New("a lixeira tem um arquivo sem ficha com esse nome")
			} else {
				err = mover(caminho, e.Arquivo)
			}
		}
		if err != nil {
			os.Remove(e.Info)
			return Entrada{}, err
		}
		return e, nil
	}
}

// lerInfo lê um .trashinfo: de onde veio e quando.
func lerInfo(l Lixeira, info string) (origem string, apagado int64, ok bool) {
	b, err := os.ReadFile(info)
	if err != nil || len(b) > 1<<20 {
		return "", 0, false
	}
	for _, linha := range strings.Split(string(b), "\n") {
		linha = strings.TrimRight(linha, "\r")
		if v, tem := strings.CutPrefix(linha, "Path="); tem && origem == "" {
			c, err := url.PathUnescape(v)
			if err != nil || c == "" || strings.ContainsRune(c, 0) {
				return "", 0, false
			}
			if !filepath.IsAbs(c) {
				ponto := l.Ponto
				if ponto == "" {
					return "", 0, false
				}
				c = filepath.Join(ponto, c)
			}
			origem = filepath.Clean(c)
		}
		if v, tem := strings.CutPrefix(linha, "DeletionDate="); tem {
			if t, err := time.ParseInLocation(formatoDaData, strings.TrimSpace(v), time.Local); err == nil {
				apagado = t.Unix()
			}
		}
	}
	return origem, apagado, origem != ""
}

func entradas(l Lixeira) []Entrada {
	fichas, _ := os.ReadDir(filepath.Join(l.Raiz, "info"))
	var es []Entrada
	for _, f := range fichas {
		nome, eFicha := strings.CutSuffix(f.Name(), ".trashinfo")
		if !eFicha || nome == "" {
			continue
		}
		e := Entrada{Arquivo: filepath.Join(l.Raiz, "files", nome), Info: filepath.Join(l.Raiz, "info", f.Name()), Lixeira: l.Raiz}
		if _, err := os.Lstat(e.Arquivo); err != nil {
			continue
		}
		var ok bool
		if e.Origem, e.ApagadoS, ok = lerInfo(l, e.Info); ok {
			es = append(es, e)
		}
	}
	return es
}

// Listar junta os itens de todas as lixeiras da pessoa.
func Listar() []Entrada {
	var es []Entrada
	for _, l := range Todas() {
		es = append(es, entradas(l)...)
	}
	return es
}

// Contar diz quantos itens há nas lixeiras.
func Contar() int { return len(Listar()) }

// Ler acha a entrada de um arquivo que está em files/ de uma lixeira da
// pessoa. Um caminho de fora de uma lixeira é erro: é a trava de restaurar e
// de apagar da lixeira.
func Ler(arquivo string) (Entrada, error) {
	if err := base.Caminho(arquivo); err != nil {
		return Entrada{}, err
	}
	for _, l := range Todas() {
		if filepath.Dir(arquivo) != filepath.Join(l.Raiz, "files") {
			continue
		}
		e := Entrada{Arquivo: arquivo, Info: filepath.Join(l.Raiz, "info", filepath.Base(arquivo)+".trashinfo"), Lixeira: l.Raiz}
		if _, err := os.Lstat(arquivo); err != nil {
			return Entrada{}, errors.New("esse item não está mais na lixeira")
		}
		var ok bool
		if e.Origem, e.ApagadoS, ok = lerInfo(l, e.Info); !ok {
			return Entrada{}, errors.New("a lixeira não sabe de onde esse item veio")
		}
		return e, nil
	}
	return Entrada{}, errors.New("isso não está na lixeira: «" + arquivo + "»")
}

// Conteudo é tudo que há direto em files/ e em info/ de uma lixeira (com os
// arquivos sem ficha e as fichas sem arquivo): é o que o esvaziar apaga.
func Conteudo(l Lixeira) (arquivos, fichas []string) {
	junta := func(sub string) []string {
		var cs []string
		es, _ := os.ReadDir(filepath.Join(l.Raiz, sub))
		for _, e := range es {
			cs = append(cs, filepath.Join(l.Raiz, sub, e.Name()))
		}
		return cs
	}
	return junta("files"), junta("info")
}

// Campos monta o item do contrato de uma entrada: o item do arquivo mais
// nome_original, origem, apagado_s e lixeira.
func Campos(e Entrada) (base.Campos, bool) {
	it, err := item.De(e.Arquivo, false)
	if err != nil {
		return nil, false
	}
	it.Oculto = false
	campos := base.Campos{}
	b, _ := json.Marshal(it)
	_ = json.Unmarshal(b, &campos)
	campos["nome_original"] = filepath.Base(e.Origem)
	campos["origem"] = e.Origem
	campos["apagado_s"] = e.ApagadoS
	campos["lixeira"] = e.Lixeira
	return campos, true
}

// Ligar põe na sessão os pedidos lixeira-listar e lixeira-estado.
func Ligar(s *base.Sessao) {
	s.Ligar("lixeira-estado", func(p *base.Pedido) {
		p.Responde(base.Campos{"itens": Contar()})
	})
	s.Ligar("lixeira-listar", func(p *base.Pedido) {
		p.Fluxo(func(cancelado func() bool) {
			lote, total := item.Lote(), 0
			itens := []base.Campos{}
			for _, e := range Listar() {
				if cancelado() {
					p.Responde(base.Campos{"itens": []base.Campos{}, "fim": true, "cancelada": true, "total": total})
					return
				}
				if c, ok := Campos(e); ok {
					itens = append(itens, c)
					total++
				}
				if len(itens) >= lote {
					p.Responde(base.Campos{"itens": itens, "fim": false})
					itens = []base.Campos{}
				}
			}
			p.Responde(base.Campos{"itens": itens, "fim": true, "total": total})
		})
	})
}
