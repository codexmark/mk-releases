package lugares

import (
	"os"
	"path/filepath"
	"strings"

	"mkfiles/internal/base"
)

func arquivoDeFavoritos() string { return base.Config("gtk-3.0", "bookmarks") }

// lerFavoritos devolve as linhas do arquivo, como estão.
func lerFavoritos() []string {
	b, err := os.ReadFile(arquivoDeFavoritos())
	if err != nil || len(b) == 0 || len(b) > 1<<20 {
		return nil
	}
	return strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
}

func gravarFavoritos(linhas []string) error {
	texto := ""
	if len(linhas) > 0 {
		texto = strings.Join(linhas, "\n") + "\n"
	}
	return base.GravaAtomico(arquivoDeFavoritos(), []byte(texto), 0o644)
}

// partes separa uma linha em URI e nome ("" se a linha não dá nome).
func partes(linha string) (uri, nome string) {
	uri, nome, _ = strings.Cut(strings.TrimSpace(linha), " ")
	return uri, strings.TrimSpace(nome)
}

// posicoes diz em que linhas do arquivo estão os favoritos (as linhas em
// branco não contam).
func posicoes(linhas []string) []int {
	var ps []int
	for i, l := range linhas {
		if strings.TrimSpace(l) != "" {
			ps = append(ps, i)
		}
	}
	return ps
}

func favoritos(linhas []string) []base.Campos {
	lista := []base.Campos{}
	for _, i := range posicoes(linhas) {
		uri, nome := partes(linhas[i])
		f := base.Campos{"uri": uri, "caminho": nil, "nome": nome, "existe": false, "remoto": true}
		if c, ok := base.DeURI(uri); ok {
			f["caminho"], f["remoto"] = c, false
			_, err := os.Stat(c)
			f["existe"] = err == nil
			if nome == "" {
				f["nome"] = filepath.Base(c)
			}
		} else if nome == "" {
			f["nome"] = uri
		}
		lista = append(lista, f)
	}
	return lista
}

func nomeDeFavorito(n string) (string, bool) {
	n = strings.TrimSpace(n)
	return n, len(n) <= 200 && !strings.ContainsAny(n, "\r\n\x00")
}

// muda lê o arquivo, aplica fn às linhas e grava o resultado. fn devolve uma
// frase de erro ("" se deu certo).
func (a *area) muda(p *base.Pedido, fn func(linhas []string) ([]string, string)) {
	a.trava.Lock()
	defer a.trava.Unlock()
	linhas, erro := fn(lerFavoritos())
	if erro != "" {
		p.Falha(erro)
		return
	}
	if err := gravarFavoritos(linhas); err != nil {
		p.Falha("não deu para gravar os favoritos: " + err.Error())
		return
	}
	p.Responde(base.Campos{"favoritos": favoritos(linhas)})
}

func acha(linhas []string, uri string) int {
	for _, i := range posicoes(linhas) {
		if u, _ := partes(linhas[i]); u == uri {
			return i
		}
	}
	return -1
}

func (a *area) favoritoPor(p *base.Pedido) {
	var q struct {
		Caminho string  `json:"caminho"`
		Nome    *string `json:"nome"`
		Posicao *int    `json:"posicao"`
	}
	_ = p.Ler(&q)
	if err := base.Caminho(q.Caminho); err != nil {
		p.Falha(err.Error())
		return
	}
	if !ePasta(q.Caminho) {
		p.Falha("só uma pasta pode ser favorita")
		return
	}
	nome := ""
	if q.Nome != nil {
		var ok bool
		if nome, ok = nomeDeFavorito(*q.Nome); !ok {
			p.FalhaDeCampo("nome", "esse nome não serve")
			return
		}
	}
	a.muda(p, func(linhas []string) ([]string, string) {
		uri := base.URI(q.Caminho)
		// Já é favorita (com o URI escrito de qualquer jeito): fica como está.
		for _, i := range posicoes(linhas) {
			u, _ := partes(linhas[i])
			if c, ok := base.DeURI(u); ok && c == q.Caminho {
				return linhas, ""
			}
		}
		nova := uri
		if nome != "" {
			nova += " " + nome
		}
		ps := posicoes(linhas)
		onde := len(linhas)
		if q.Posicao != nil && *q.Posicao >= 0 && *q.Posicao < len(ps) {
			onde = ps[*q.Posicao]
		}
		return append(linhas[:onde:onde], append([]string{nova}, linhas[onde:]...)...), ""
	})
}

func (a *area) favoritoTirar(p *base.Pedido) {
	var q struct {
		URI string `json:"uri"`
	}
	_ = p.Ler(&q)
	a.muda(p, func(linhas []string) ([]string, string) {
		i := acha(linhas, q.URI)
		if i < 0 {
			return nil, "esse favorito não existe mais"
		}
		return append(linhas[:i:i], linhas[i+1:]...), ""
	})
}

func (a *area) favoritoMover(p *base.Pedido) {
	var q struct {
		De   *int `json:"de"`
		Para *int `json:"para"`
	}
	_ = p.Ler(&q)
	a.muda(p, func(linhas []string) ([]string, string) {
		ps := posicoes(linhas)
		if q.De == nil || q.Para == nil || *q.De < 0 || *q.De >= len(ps) || *q.Para < 0 || *q.Para >= len(ps) {
			return nil, "posição de favorito que não existe"
		}
		// Só os favoritos trocam de lugar: as linhas em branco ficam onde estão.
		ordem := make([]string, len(ps))
		for n, i := range ps {
			ordem[n] = linhas[i]
		}
		movido := ordem[*q.De]
		ordem = append(ordem[:*q.De], ordem[*q.De+1:]...)
		ordem = append(ordem[:*q.Para:*q.Para], append([]string{movido}, ordem[*q.Para:]...)...)
		for n, i := range ps {
			linhas[i] = ordem[n]
		}
		return linhas, ""
	})
}

func (a *area) favoritoRenomear(p *base.Pedido) {
	var q struct {
		URI  string `json:"uri"`
		Nome string `json:"nome"`
	}
	_ = p.Ler(&q)
	nome, ok := nomeDeFavorito(q.Nome)
	if !ok {
		p.FalhaDeCampo("nome", "esse nome não serve")
		return
	}
	a.muda(p, func(linhas []string) ([]string, string) {
		i := acha(linhas, q.URI)
		if i < 0 {
			return nil, "esse favorito não existe mais"
		}
		linhas[i] = q.URI
		if nome != "" {
			linhas[i] += " " + nome
		}
		return linhas, ""
	})
}
