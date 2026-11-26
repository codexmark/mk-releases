// SPDX-License-Identifier: GPL-3.0-or-later

package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"mktunes/internal/app"
	"mktunes/internal/dominio"
	"mktunes/internal/guarda"
	"mktunes/internal/laco"
	"mktunes/internal/prova"
)

func igual(t *testing.T, esperado, obtido any) {
	t.Helper()
	if !reflect.DeepEqual(esperado, obtido) {
		t.Fatalf("esperado %#v\n  obtido %#v", esperado, obtido)
	}
}

// catalogoFalso é a rede inteira de mentira: o catálogo, o Início, as filas,
// as páginas, as sugestões, as letras e as rádios.
type catalogoFalso struct {
	mu          sync.Mutex
	musicas     []dominio.Faixa
	fora        bool
	paginasFora bool
	pedidos     []string
	letra       *dominio.LetraDaFaixa
	album       dominio.Item
	artista     dominio.Item
	estacoes    []dominio.Faixa
}

func novoCatalogo() *catalogoFalso {
	return &catalogoFalso{
		musicas:  []dominio.Faixa{prova.Faixa("1", prova.Titulo("Umbrella"), prova.Artista("Rihanna")), prova.Faixa("2", prova.Titulo("Halo"), prova.Artista("Beyoncé"))},
		album:    dominio.Item{Tipo: dominio.ItemColecao, ID: "MPRE1", Colecao: dominio.Album, Titulo: "Anti", Subtitulo: "Rihanna", PlaylistID: "OLAK1"},
		artista:  dominio.Item{Tipo: dominio.ItemArtista, ID: "UC1", Titulo: "Rihanna", Subtitulo: "Artista"},
		estacoes: []dominio.Faixa{prova.Faixa("r1", prova.Titulo("Rádio Um"), prova.Provedor(dominio.Radio), prova.Duracao(0))},
	}
}

func (c *catalogoFalso) confere() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.fora {
		return app.ErrCatalogo
	}
	return nil
}

func (c *catalogoFalso) derruba(fora bool) {
	c.mu.Lock()
	c.fora = fora
	c.mu.Unlock()
}

func (c *catalogoFalso) BuscaMusicas(context.Context, string) ([]dominio.Faixa, error) {
	return c.musicas, c.confere()
}

func (c *catalogoFalso) Inicio(_ context.Context, chip *dominio.Chip) (dominio.PaginaFeed, error) {
	if erro := c.confere(); erro != nil {
		return dominio.PaginaFeed{}, erro
	}
	titulo := "Escolha a dedo"
	if chip != nil {
		titulo = chip.Titulo
	}
	return dominio.PaginaFeed{
		Chips:       []dominio.Chip{{Titulo: "Relax", Token: "t-relax"}, {Titulo: "Festa", Token: "t-festa"}},
		Prateleiras: []dominio.Prateleira{{Titulo: titulo, Itens: []dominio.Item{dominio.ItemDeMusica(c.musicas[0]), dominio.ItemDeMusica(c.musicas[1]), c.album}}},
		Proxima:     "p2",
	}, nil
}

func (c *catalogoFalso) Mais(context.Context, string) (dominio.PaginaFeed, error) {
	return dominio.PaginaFeed{Prateleiras: []dominio.Prateleira{{Titulo: "Mais", Itens: []dominio.Item{c.album}}}}, c.confere()
}

func (c *catalogoFalso) Radio(context.Context, dominio.Faixa) ([]dominio.Faixa, error) {
	return nil, nil
}
func (c *catalogoFalso) Playlist(context.Context, string) ([]dominio.Faixa, error) {
	return c.musicas, c.confere()
}

func (c *catalogoFalso) Colecao(_ context.Context, id string) (dominio.PaginaColecao, error) {
	return dominio.PaginaColecao{ID: id, Tipo: dominio.Album, Titulo: "Anti", Subtitulo: "Álbum • 2016", Dono: "Rihanna", Detalhe: "2 músicas", PlaylistID: "OLAK1", Faixas: c.musicas}, c.confere()
}

func (c *catalogoFalso) Artista(_ context.Context, id string) (dominio.PaginaArtista, error) {
	return dominio.PaginaArtista{ID: id, Nome: "Rihanna", Descricao: "Cantora", Principais: c.musicas, Prateleiras: []dominio.Prateleira{{Titulo: "Álbuns", Itens: []dominio.Item{c.album}}}, Audiencia: "12 mi ouvintes mensais", RadioID: "RDEM1"}, c.confere()
}

func (c *catalogoFalso) BuscaPaginas(_ context.Context, _ string, filtro dominio.Filtro) ([]dominio.Item, error) {
	if erro := c.confere(); erro != nil {
		return nil, erro
	}
	c.mu.Lock()
	fora := c.paginasFora
	c.mu.Unlock()
	switch {
	case fora:
		return nil, app.ErrCatalogo
	case filtro == dominio.FiltroArtistas:
		return []dominio.Item{c.artista}, nil
	case filtro == dominio.FiltroPlaylists:
		return nil, nil
	}
	return []dominio.Item{c.album}, nil
}

func (c *catalogoFalso) Sugere(_ context.Context, texto string) ([]string, error) {
	return []string{texto + " remix"}, nil
}

func (c *catalogoFalso) Acha(context.Context, dominio.Faixa) (*dominio.LetraDaFaixa, error) {
	return c.letra, c.confere()
}

func (c *catalogoFalso) Topo(context.Context, int) ([]dominio.Faixa, error) {
	return c.estacoes, c.confere()
}

func (c *catalogoFalso) Busca(_ context.Context, nome string, _ int) ([]dominio.Faixa, error) {
	var achadas []dominio.Faixa
	for _, e := range c.estacoes {
		if strings.Contains(strings.ToLower(e.Titulo), strings.ToLower(nome)) {
			achadas = append(achadas, e)
		}
	}
	return achadas, c.confere()
}

// discoFalso é a pasta de música de mentira.
type discoFalso struct {
	mu       sync.Mutex
	salvos   map[string]string
	quebrado bool
	// falha, quando não é nil, é o erro que o download devolve.
	falha error
}

func (d *discoFalso) Busca(_ context.Context, url, relativo string, progresso func(float64)) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.falha != nil {
		return "", d.falha
	}
	if d.quebrado {
		return "", errors.New("disco cheio")
	}
	progresso(1)
	d.salvos["/music/"+relativo] = url
	return "/music/" + relativo, nil
}

func (d *discoFalso) Existe(caminho string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, tem := d.salvos[caminho]
	return tem
}

func (d *discoFalso) Apaga(caminho string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.salvos, caminho)
}

type volumeNaMemoria struct{ nivel *int }

func (v *volumeNaMemoria) Carrega() (int, bool) {
	if v.nivel == nil {
		return 0, false
	}
	return *v.nivel, true
}
func (v *volumeNaMemoria) Salva(nivel int) { v.nivel = &nivel }

// bancada é o daemon inteiro com a guarda de verdade numa pasta temporária e a
// rede trocada por dublês.
type bancada struct {
	t       *testing.T
	l       *laco.Teste
	c       *catalogoFalso
	p       *prova.Tocador
	d       *discoFalso
	app     *App
	volumes []float64
	locais  []dominio.Faixa
	avisos  []string
	// registro é o que o daemon anotou no log.
	registro []string
	avulsas  []map[string]any
	ajustes  []dominio.Ajustes
	saiu     int
	pasta    string
	backups  string
}

func novaBancada(t *testing.T, pasta string, volume GuardaDeVolume) *bancada {
	t.Helper()
	if pasta == "" {
		pasta = t.TempDir()
	}
	if volume == nil {
		volume = &volumeNaMemoria{}
	}
	g, erro := guarda.Abre(pasta)
	if erro != nil {
		t.Fatal(erro)
	}
	b := &bancada{t: t, l: laco.NovoTeste(), c: novoCatalogo(), p: &prova.Tocador{}, d: &discoFalso{salvos: map[string]string{}}, pasta: pasta, backups: t.TempDir()}
	b.p.AoEvento = func(e app.Evento) { b.app.Sessao.AoEvento(e) }
	b.locais = []dominio.Faixa{prova.Faixa("l1", prova.Titulo("Local Um"), prova.Provedor(dominio.Local), func(f *dominio.Faixa) { f.Stream = "file:///m/um.mp3" })}
	b.app = Monta(Portas{
		Catalogo: b.c, Inicio: b.c, Filas: b.c, Navegador: b.c, Paginas: b.c, Sugestoes: b.c,
		Streams: &prova.Resolvedor{Base: "https://stream/"}, Tocador: b.p, Letras: b.c, Radio: b.c, Arquivos: b.d,
		Volume: func(nivel float64) { b.volumes = append(b.volumes, nivel) },
		Local:  func(context.Context) []dominio.Faixa { return b.locais },
	}, Config{
		Guarda: g, Codec: guarda.Codec{}, Laco: b.l, PastaDeBackups: b.backups, PastaDeDownloads: "/home/ouvinte/Música/MkTunes",
		Versao: "9.9.9", Volume: volume,
		AoMudarAjustes: func(a dominio.Ajustes) { b.ajustes = append(b.ajustes, a) },
		AoSair:         func() { b.saiu++ },
		Registro:       func(texto string) { b.registro = append(b.registro, texto) },
	})
	b.app.Quadro.Ouve(func(tipo, linha string) {
		if tipo != "" {
			return
		}
		var m map[string]any
		_ = json.Unmarshal([]byte(linha), &m)
		b.avulsas = append(b.avulsas, m)
		if m["type"] == "notice" {
			b.avisos = append(b.avisos, m["text"].(string))
		}
	})
	b.l.Roda()
	return b
}

// manda entrega um comando e deixa o daemon sossegar.
func (b *bancada) manda(comando any) {
	b.t.Helper()
	linha, ok := comando.(string)
	if !ok {
		linha = codifica(comando)
	}
	b.app.Trata(linha)
	b.l.Roda()
}

func (b *bancada) toca(indice int, faixas ...dominio.Faixa) {
	b.t.Helper()
	b.manda(msg{"cmd": "play", "tracks": faixasNoFio(faixas), "index": indice})
}

// estado lê o estado de uma área como a janela o veria.
func (b *bancada) estado(tipo string) map[string]any {
	b.t.Helper()
	var m map[string]any
	if erro := json.Unmarshal([]byte(b.app.Quadro.Linha(tipo)), &m); erro != nil {
		b.t.Fatalf("o estado %q não é JSON: %v", tipo, erro)
	}
	return m
}

func (b *bancada) pagina() map[string]any {
	b.t.Helper()
	for i := len(b.avulsas) - 1; i >= 0; i-- {
		if b.avulsas[i]["type"] == "page" {
			return b.avulsas[i]
		}
	}
	b.t.Fatal("nenhuma página chegou")
	return nil
}

func (b *bancada) ultimoAviso() string {
	if len(b.avisos) == 0 {
		return ""
	}
	return b.avisos[len(b.avisos)-1]
}

func lista(m map[string]any, chave string) []map[string]any {
	var itens []map[string]any
	bruto, _ := m[chave].([]any)
	for _, item := range bruto {
		itens = append(itens, item.(map[string]any))
	}
	return itens
}

func titulosDe(m map[string]any, chave string) []string {
	titulos := []string{}
	for _, item := range lista(m, chave) {
		titulos = append(titulos, item["title"].(string))
	}
	return titulos
}

var _ = time.Second
