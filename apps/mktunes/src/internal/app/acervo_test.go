// SPDX-License-Identifier: GPL-3.0-or-later

package app_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"mktunes/internal/app"
	"mktunes/internal/dominio"
	"mktunes/internal/laco"
	"mktunes/internal/prova"
)

var meioDia = time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)

// passaDireto guarda o backup como está: estes testes são da junção, não do JSON.
type passaDireto struct{ ultimo *dominio.Backup }

func (p *passaDireto) Codifica(b dominio.Backup) string { p.ultimo = &b; return "backup" }
func (p *passaDireto) Decodifica(texto string) (dominio.Backup, bool) {
	if texto != "backup" || p.ultimo == nil {
		return dominio.Backup{}, false
	}
	return *p.ultimo, true
}

func novoAcervo(codec app.Codec) (*app.Acervo, *prova.Memoria) {
	m := prova.NovaMemoria()
	n := 0
	a := app.NovoAcervo(m, codec, func() time.Time { return meioDia })
	a.NovoID = func() string { n++; return "p" + string(rune('0'+n)) }
	return a, m
}

func TestCurtidasEListas(t *testing.T) {
	a, m := novoAcervo(nil)
	musica := prova.Faixa("song")
	mudou := 0
	a.Mudou.Ouve(func() { mudou++ })
	igual(t, false, a.AlternaCurtida(musica))
	igual(t, []string{"song"}, prova.IDs(a.Curtidas()))
	igual(t, true, a.Curtida("song"))
	igual(t, true, a.AlternaCurtida(musica))
	igual(t, false, a.Curtida("song"))
	igual(t, 2, mudou)

	m.Tocadas = []dominio.Tocada{
		{Faixa: musica, Em: meioDia.Add(-time.Hour)},
		{Faixa: musica, Em: meioDia.Add(-2 * time.Hour)},
		{Faixa: prova.Faixa("old"), Em: meioDia.Add(-20 * 24 * time.Hour)},
	}
	listas := a.Listas()
	igual(t, []string{"song", "old"}, prova.IDs(listas.NoRepeat))
	igual(t, []string{"song"}, prova.IDs(listas.Recentes))
	igual(t, []string{"old"}, prova.IDs(listas.Redescobrir))
	igual(t, 2, a.Resumo(dominio.PeriodoDia).Tocadas)
}

func TestPlaylistsDoAcervo(t *testing.T) {
	a, _ := novoAcervo(nil)
	id, ok := a.CriaPlaylist("  Estrada ", nil)
	igual(t, "p1", id)
	igual(t, true, ok)
	igual(t, "Estrada", a.Playlists()[0].Titulo)
	_, ok = a.CriaPlaylist("   ", nil)
	igual(t, false, ok)
	igual(t, 1, len(a.Playlists()))

	igual(t, true, a.AdicionaAPlaylist(id, prova.Faixa("1")))
	igual(t, false, a.AdicionaAPlaylist(id, prova.Faixa("1")))
	igual(t, false, a.AdicionaAPlaylist("não existe", prova.Faixa("1")))
	a.AdicionaAPlaylist(id, prova.Faixa("2"))
	a.EditaPlaylist(id, func(p dominio.PlaylistLocal) dominio.PlaylistLocal { return p.MoveFaixa(1, 0) })
	igual(t, []string{"2", "1"}, prova.IDs(a.Playlists()[0].Faixas))
	a.EditaPlaylist(id, func(p dominio.PlaylistLocal) dominio.PlaylistLocal { return p.Remove(0).Renomeia("Viagem") })
	igual(t, []string{"1"}, prova.IDs(a.Playlists()[0].Faixas))
	igual(t, "Viagem", a.Playlists()[0].Titulo)
	a.ApagaPlaylist(id)
	igual(t, 0, len(a.Playlists()))

	faixa := prova.Faixa("1")
	a.CriaPlaylist("Nova", &faixa)
	igual(t, []string{"1"}, prova.IDs(a.Playlists()[0].Faixas))
}

func TestColecoesSalvas(t *testing.T) {
	a, _ := novoAcervo(nil)
	pagina := func(id string, tipo dominio.TipoColecao, dono string) dominio.PaginaColecao {
		return dominio.PaginaColecao{ID: id, Tipo: tipo, Titulo: "Sobrevivendo no Inferno", Subtitulo: "Álbum • 1997", Dono: dono, Capa: "https://art/" + id, PlaylistID: "OLAK" + id}
	}
	igual(t, false, a.AlternaSalvo(pagina("MPRE1", dominio.Album, "Racionais MC's")))
	igual(t, []dominio.ColecaoSalva{{ID: "MPRE1", Tipo: dominio.Album, Titulo: "Sobrevivendo no Inferno", Dono: "Racionais MC's", Capa: "https://art/MPRE1", SalvaEm: meioDia}}, a.Salvos())
	igual(t, true, a.Salvo("MPRE1"))
	a.AlternaSalvo(pagina("VLPL2", dominio.Playlist, ""))
	igual(t, "VLPL2", a.Salvos()[0].ID)
	igual(t, "Playlist", a.Salvos()[0].Dono)
	igual(t, true, a.AlternaSalvo(pagina("MPRE1", dominio.Album, "x")))
	igual(t, false, a.Salvo("MPRE1"))
}

func TestBuscasRecentesEAjustes(t *testing.T) {
	a, m := novoAcervo(nil)
	a.LembraBusca("daft punk")
	igual(t, []string{"lilriog", "daft punk"}, a.LembraBusca("lilriog"))
	a.LimpaBuscas()
	igual(t, 0, len(a.Buscas()))

	a.MudaAjustes(func(s dominio.Ajustes) dominio.Ajustes {
		s.Autoplay, s.Qualidade, s.Normalizar = false, dominio.QualidadeEconomia, false
		return s.ComTrabalhadores(7)
	})
	igual(t, dominio.Ajustes{Qualidade: dominio.QualidadeEconomia, Trabalhadores: 4, OndaSonora: true}, m.Prefs)
}

func TestBackup(t *testing.T) {
	album := dominio.ColecaoSalva{ID: "MPRE1", Tipo: dominio.Album, Titulo: "Sobrevivendo no Inferno", Dono: "Racionais MC's", SalvaEm: meioDia}
	tocada := dominio.Tocada{Faixa: prova.Faixa("p"), Em: meioDia.Add(-time.Minute)}
	t.Run("exporta tudo que o ouvinte montou", func(t *testing.T) {
		codec := &passaDireto{}
		a, m := novoAcervo(codec)
		a.AlternaCurtida(prova.Faixa("liked"))
		m.RegistraTocada(tocada)
		m.SalvaPlaylist(dominio.PlaylistLocal{ID: "1", Titulo: "Estrada", Faixas: []dominio.Faixa{prova.Faixa("a")}}, meioDia)
		m.Prefs.Autoplay = false
		m.Recentes = []string{"lilriog"}
		m.SalvaColecao(album)
		a.Exporta()
		b := codec.ultimo
		igual(t, meioDia, b.CriadoEm)
		igual(t, []string{"liked"}, prova.IDs(b.Curtidas))
		igual(t, "Estrada", b.Playlists[0].Titulo)
		igual(t, 1, len(b.Tocadas))
		igual(t, false, b.Ajustes.Autoplay)
		igual(t, []string{"lilriog"}, b.Buscas)
		igual(t, []dominio.ColecaoSalva{album}, b.Salvos)
	})
	t.Run("restaurar junta ao que o computador novo já tem, sem repetir", func(t *testing.T) {
		codec := &passaDireto{}
		velho, mv := novoAcervo(codec)
		velho.AlternaCurtida(prova.Faixa("both"))
		velho.AlternaCurtida(prova.Faixa("old"))
		mv.RegistraTocada(tocada)
		mv.SalvaPlaylist(dominio.PlaylistLocal{ID: "1", Titulo: "Estrada", Faixas: []dominio.Faixa{prova.Faixa("a")}}, meioDia)
		mv.Recentes = []string{"old search"}
		mv.SalvaColecao(album)
		texto := velho.Exporta()

		novo, mn := novoAcervo(codec)
		novo.AlternaCurtida(prova.Faixa("both"))
		mn.RegistraTocada(tocada)
		mn.Recentes = []string{"new search"}
		jaAqui := album
		jaAqui.Titulo = "already here"
		mn.SalvaColecao(jaAqui)
		r, ok := novo.Restaura(texto)
		igual(t, true, ok)
		igual(t, app.Restaurado{Curtidas: 1, Playlists: 1, Tocadas: 0}, r)
		igual(t, []string{"old", "both"}, prova.IDs(novo.Curtidas()))
		igual(t, 1, len(mn.Tocadas))
		igual(t, "Estrada", mn.Listas[0].Titulo)
		igual(t, []string{"new search", "old search"}, mn.Recentes)
		igual(t, "already here", mn.Guardados[0].Titulo)
	})
	t.Run("um backup antigo acrescenta a uma playlist editada depois, em vez de desfazer", func(t *testing.T) {
		codec := &passaDireto{}
		velho, mv := novoAcervo(codec)
		mv.SalvaPlaylist(dominio.PlaylistLocal{ID: "1", Titulo: "Estrada", Faixas: []dominio.Faixa{prova.Faixa("a"), prova.Faixa("gone")}}, meioDia)
		texto := velho.Exporta()
		aqui, ma := novoAcervo(codec)
		ma.SalvaPlaylist(dominio.PlaylistLocal{ID: "1", Titulo: "Estrada nova", Faixas: []dominio.Faixa{prova.Faixa("a"), prova.Faixa("b")}}, meioDia)
		aqui.Restaura(texto)
		igual(t, "Estrada nova", ma.Listas[0].Titulo)
		igual(t, []string{"a", "b", "gone"}, prova.IDs(ma.Listas[0].Faixas))
	})
	t.Run("recusa um arquivo que não é backup", func(t *testing.T) {
		a, _ := novoAcervo(&passaDireto{})
		_, ok := a.Restaura("{ hello }")
		igual(t, false, ok)
	})
}

// buscadorFalso é o disco de mentira dos downloads.
type buscadorFalso struct {
	mu        sync.Mutex
	portoes   map[string]*laco.Portao
	comecados []string
	arquivos  map[string]bool
	falhando  map[string]bool
	quebrado  map[string]bool
}

func (b *buscadorFalso) Busca(ctx context.Context, url, relativo string, progresso func(float64)) (string, error) {
	b.mu.Lock()
	b.comecados = append(b.comecados, relativo)
	portao, falha, quebra := b.portoes[url], b.falhando[url], b.quebrado[url]
	b.mu.Unlock()
	progresso(0.5)
	if portao != nil {
		if erro := portao.Espera(ctx); erro != nil {
			return "", erro
		}
	}
	if falha {
		return "", &app.ErroDownload{Motivo: "HTTP 403"}
	}
	if quebra {
		return "", errors.New("No space left on device")
	}
	progresso(1)
	b.mu.Lock()
	defer b.mu.Unlock()
	b.arquivos["/music/"+relativo] = true
	return "/music/" + relativo, nil
}
func (b *buscadorFalso) Existe(caminho string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.arquivos[caminho]
}
func (b *buscadorFalso) Apaga(caminho string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.arquivos, caminho)
}
func (b *buscadorFalso) quantosComecaram() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.comecados)
}

type bancadaDeDownloads struct {
	l *laco.Teste
	r *prova.Resolvedor
	b *buscadorFalso
	m *prova.Memoria
	d *app.Downloads
}

func novosDownloads(trabalhadores int) *bancadaDeDownloads {
	x := &bancadaDeDownloads{
		l: laco.NovoTeste(),
		r: &prova.Resolvedor{Indisponiveis: map[string]bool{}, Base: "https://media/"},
		b: &buscadorFalso{portoes: map[string]*laco.Portao{}, arquivos: map[string]bool{}, falhando: map[string]bool{}, quebrado: map[string]bool{}},
		m: prova.NovaMemoria(),
	}
	x.d = app.NovosDownloads(x.r, x.b, x.m, x.l, func() int { return trabalhadores })
	return x
}

func (x *bancadaDeDownloads) de(id string) dominio.Download {
	for _, j := range x.d.Lista() {
		if j.Faixa.ID == id {
			return j
		}
	}
	return dominio.Download{}
}

func (x *bancadaDeDownloads) portao(url string) *laco.Portao {
	p := x.l.Portao()
	x.b.mu.Lock()
	x.b.portoes[url] = p
	x.b.mu.Unlock()
	return p
}

func TestDownloads(t *testing.T) {
	um := prova.Faixa("one", prova.Titulo("Ignored"), prova.Artista("LilRioG"), prova.SemStream)
	dois := prova.Faixa("two", prova.SemStream)
	t.Run("baixa a faixa para a pasta do artista", func(t *testing.T) {
		x := novosDownloads(1)
		x.d.Enfileira(um)
		x.l.Roda()
		igual(t, dominio.Concluido, x.de("one").Estado)
		igual(t, "/music/LilRioG/Ignored [one].m4a", x.de("one").Caminho)
		igual(t, "/music/LilRioG/Ignored [one].m4a", x.d.CaminhoLocal("one"))
		igual(t, x.d.Lista(), x.m.Baixados)
		// E toca do arquivo em vez da rede.
		s, tem := x.d.StreamLocal(um)
		igual(t, true, tem)
		igual(t, "file:///music/LilRioG/Ignored%20%5Bone%5D.m4a", s.URL)
		_, tem = x.d.StreamLocal(dois)
		igual(t, false, tem)
		// Um arquivo apagado por fora não é oferecido.
		x.b.Apaga("/music/LilRioG/Ignored [one].m4a")
		igual(t, "", x.d.CaminhoLocal("one"))
	})
	t.Run("informa o progresso, um de cada vez por padrão", func(t *testing.T) {
		x := novosDownloads(1)
		portao := x.portao("https://media/one")
		x.d.Enfileira(um, dois)
		x.l.Roda()
		igual(t, dominio.Baixando, x.de("one").Estado)
		igual(t, 0.5, x.de("one").Progresso)
		igual(t, "", x.d.CaminhoLocal("one"))
		igual(t, dominio.NaFila, x.de("two").Estado)
		portao.Abre()
		x.l.Roda()
		igual(t, dominio.Concluido, x.de("two").Estado)
	})
	t.Run("vários de uma vez quando o ajuste deixa", func(t *testing.T) {
		x := novosDownloads(2)
		x.portao("https://media/one")
		x.portao("https://media/two")
		x.d.Enfileira(um, dois)
		x.l.Roda()
		igual(t, 2, x.b.quantosComecaram())
	})
	t.Run("um álbum fica na sua ordem depois de reiniciar", func(t *testing.T) {
		x := novosDownloads(0)
		x.d.Enfileira(prova.Faixa("t1"), prova.Faixa("t2"), prova.Faixa("t3"))
		x.l.Roda()
		igual(t, 3, len(x.m.Baixados))
		igual(t, true, x.m.Baixados[0].CriadoEm.Before(x.m.Baixados[1].CriadoEm) && x.m.Baixados[1].CriadoEm.Before(x.m.Baixados[2].CriadoEm))
	})
	t.Run("explica as falhas e tenta de novo a pedido", func(t *testing.T) {
		x := novosDownloads(1)
		x.b.quebrado["https://media/one"] = true
		x.d.Enfileira(um)
		x.l.Roda()
		igual(t, "Não foi possível salvar a música", x.de("one").Motivo)

		x = novosDownloads(1)
		x.b.falhando["https://media/one"] = true
		x.d.Enfileira(um)
		x.l.Roda()
		igual(t, "HTTP 403", x.de("one").Motivo)
		// E a causa vai para quem anota (o log do daemon); um download que deu certo não.
		var anotadas []string
		x = novosDownloads(1)
		x.d.AoFalhar = func(f dominio.Faixa, erro error) { anotadas = append(anotadas, f.ID+": "+erro.Error()) }
		x.b.falhando["https://media/one"] = true
		x.d.Enfileira(um, dois)
		x.l.Roda()
		igual(t, []string{"one: HTTP 403"}, anotadas)

		x = novosDownloads(1)
		x.r.Muda(func(r *prova.Resolvedor) { r.Indisponiveis["one"] = true })
		x.d.Enfileira(um)
		x.l.Roda()
		igual(t, dominio.Falhou, x.de("one").Estado)
		igual(t, "Faixa indisponível para download", x.de("one").Motivo)
		x.r.Muda(func(r *prova.Resolvedor) { r.Indisponiveis = map[string]bool{} })
		x.d.TentaDeNovo("one")
		x.l.Roda()
		igual(t, dominio.Concluido, x.de("one").Estado)
	})
	t.Run("nunca baixa rádio ao vivo, nem o que já baixou ou está a caminho", func(t *testing.T) {
		x := novosDownloads(1)
		x.d.Enfileira(prova.Faixa("radio:1", prova.Provedor(dominio.Radio)))
		x.l.Roda()
		igual(t, 0, len(x.d.Lista()))
		x.d.Enfileira(um)
		x.l.Roda()
		x.d.Enfileira(um)
		x.l.Roda()
		igual(t, 1, x.b.quantosComecaram())
	})
	t.Run("cancelar para a transferência, e o progresso atrasado não a ressuscita", func(t *testing.T) {
		x := novosDownloads(1)
		x.portao("https://media/one")
		x.d.Enfileira(um)
		x.l.Roda()
		x.d.Cancela("one")
		x.l.Roda()
		igual(t, dominio.Cancelado, x.de("one").Estado)
		igual(t, 0, len(x.b.arquivos))
		igual(t, dominio.Cancelado, x.m.Baixados[0].Estado)
	})
	t.Run("remover apaga o arquivo e o histórico", func(t *testing.T) {
		x := novosDownloads(1)
		x.d.Enfileira(um)
		x.l.Roda()
		x.d.Remove("one")
		x.l.Roda()
		igual(t, 0, len(x.b.arquivos)+len(x.d.Lista())+len(x.m.Baixados))
	})
	t.Run("depois de reiniciar, o que foi interrompido aparece como falha", func(t *testing.T) {
		x := novosDownloads(1)
		x.m.Baixados = []dominio.Download{
			{Faixa: um, Estado: dominio.Baixando, Progresso: 0.3, CriadoEm: meioDia},
			{Faixa: dois, Estado: dominio.Concluido, Caminho: "/music/x.m4a", CriadoEm: meioDia},
		}
		x.b.arquivos["/music/x.m4a"] = true
		x.d.Restaura()
		igual(t, "Interrompido", x.de("one").Motivo)
		igual(t, dominio.Falhou, x.m.Baixados[0].Estado)
		igual(t, "/music/x.m4a", x.d.CaminhoLocal("two"))
	})
	t.Run("um download começado antes de o histórico carregar é mantido", func(t *testing.T) {
		x := novosDownloads(1)
		x.m.Baixados = []dominio.Download{{Faixa: dois, Estado: dominio.Concluido, Caminho: "/music/x.m4a", CriadoEm: meioDia}}
		x.portao("https://media/one")
		x.d.Enfileira(um)
		x.l.Roda()
		x.d.Restaura()
		igual(t, 2, len(x.d.Lista()))
		igual(t, dominio.Baixando, x.de("one").Estado)
	})
}

// fonteFalsa responde a tudo que os casos de busca perguntam.
type fonteFalsa struct {
	erro      error
	faixas    []dominio.Faixa
	itens     []dominio.Item
	pedidas   []string
	pagina    func(chip *dominio.Chip, token string) dominio.PaginaFeed
	guardada  *dominio.PaginaFeed
	letra     *dominio.LetraDaFaixa
	perguntas int
}

func (f *fonteFalsa) BuscaMusicas(_ context.Context, c string) ([]dominio.Faixa, error) {
	f.pedidas = append(f.pedidas, c)
	return f.faixas, f.erro
}
func (f *fonteFalsa) BuscaPaginas(_ context.Context, c string, filtro dominio.Filtro) ([]dominio.Item, error) {
	f.pedidas = append(f.pedidas, c+"|"+string(filtro))
	return f.itens, f.erro
}
func (f *fonteFalsa) Sugere(_ context.Context, texto string) ([]string, error) {
	f.pedidas = append(f.pedidas, texto)
	return []string{texto + " mc's", texto + " vida loka"}, f.erro
}
func (f *fonteFalsa) Topo(context.Context, int) ([]dominio.Faixa, error) {
	return []dominio.Faixa{prova.Faixa("top")}, f.erro
}
func (f *fonteFalsa) Busca(_ context.Context, nome string, _ int) ([]dominio.Faixa, error) {
	f.pedidas = append(f.pedidas, nome)
	return []dominio.Faixa{prova.Faixa("found")}, f.erro
}
func (f *fonteFalsa) Inicio(_ context.Context, chip *dominio.Chip) (dominio.PaginaFeed, error) {
	if f.erro != nil {
		return dominio.PaginaFeed{}, f.erro
	}
	return f.pagina(chip, ""), nil
}
func (f *fonteFalsa) Mais(_ context.Context, token string) (dominio.PaginaFeed, error) {
	return f.pagina(nil, token), f.erro
}
func (f *fonteFalsa) Carrega() (dominio.PaginaFeed, bool) {
	if f.guardada == nil {
		return dominio.PaginaFeed{}, false
	}
	return *f.guardada, true
}
func (f *fonteFalsa) Guarda(p dominio.PaginaFeed) { f.guardada = &p }
func (f *fonteFalsa) Acha(context.Context, dominio.Faixa) (*dominio.LetraDaFaixa, error) {
	f.perguntas++
	return f.letra, f.erro
}

func TestBuscas(t *testing.T) {
	ctx := context.Background()
	duas := []dominio.Faixa{prova.Faixa("1"), prova.Faixa("2")}
	f := &fonteFalsa{faixas: duas}
	faixas, desfecho := app.BuscaMusicas(ctx, f, "  lilriog  ")
	igual(t, duas, faixas)
	igual(t, app.Achou, desfecho)
	igual(t, []string{"lilriog"}, f.pedidas)
	_, desfecho = app.BuscaMusicas(ctx, &fonteFalsa{}, "zzz")
	igual(t, app.NadaAchado, desfecho)
	f = &fonteFalsa{}
	_, desfecho = app.BuscaMusicas(ctx, f, "   ")
	igual(t, app.ConsultaInvalida, desfecho)
	igual(t, 0, len(f.pedidas))
	_, desfecho = app.BuscaMusicas(ctx, &fonteFalsa{erro: app.ErrCatalogo}, "x")
	igual(t, app.Indisponivel, desfecho)

	artista := dominio.Item{Tipo: dominio.ItemArtista, ID: "UC1", Titulo: "Marília Mendonça", Subtitulo: "Artista"}
	f = &fonteFalsa{itens: []dominio.Item{artista}}
	itens, desfecho := app.BuscaPaginas(ctx, f, " marilia ", dominio.FiltroArtistas)
	igual(t, []dominio.Item{artista}, itens)
	igual(t, app.Achou, desfecho)
	igual(t, []string{"marilia|ARTISTS"}, f.pedidas)
	_, desfecho = app.BuscaPaginas(ctx, &fonteFalsa{}, "x", dominio.FiltroAlbuns)
	igual(t, app.NadaAchado, desfecho)
	_, desfecho = app.BuscaPaginas(ctx, &fonteFalsa{}, "  ", dominio.FiltroAlbuns)
	igual(t, app.ConsultaInvalida, desfecho)
	_, desfecho = app.BuscaPaginas(ctx, &fonteFalsa{erro: app.ErrCatalogo}, "x", dominio.FiltroAlbuns)
	igual(t, app.Indisponivel, desfecho)

	f = &fonteFalsa{}
	igual(t, []string{"racion mc's", "racion vida loka"}, app.Sugere(ctx, f, "  racion "))
	igual(t, []string{"racion"}, f.pedidas)
	igual(t, 0, len(app.Sugere(ctx, f, " r ")))
	igual(t, 1, len(f.pedidas))
	igual(t, 0, len(app.Sugere(ctx, &fonteFalsa{erro: app.ErrCatalogo}, "racion")))
}

func TestEstacoes(t *testing.T) {
	ctx := context.Background()
	f := &fonteFalsa{}
	estacoes, _ := app.Estacoes(ctx, f, "")
	igual(t, []string{"top"}, prova.IDs(estacoes))
	app.Estacoes(ctx, f, "  "+strings.Repeat("a", 200)+" ")
	igual(t, 128, len(f.pedidas[0]))
	_, erro := app.Estacoes(ctx, &fonteFalsa{erro: app.ErrCatalogo}, "")
	igual(t, true, erro != nil)
}

func TestInicio(t *testing.T) {
	ctx := context.Background()
	musicas := dominio.Prateleira{Titulo: "Mixes", Itens: []dominio.Item{dominio.ItemDeMusica(prova.Faixa("1"))}}
	vazia := dominio.Prateleira{Titulo: "Nada"}
	f := &fonteFalsa{pagina: func(chip *dominio.Chip, token string) dominio.PaginaFeed {
		titulo := "Mixes"
		if chip != nil {
			titulo = chip.Titulo
		}
		if token != "" {
			titulo = token
		}
		return dominio.PaginaFeed{Prateleiras: []dominio.Prateleira{{Titulo: titulo, Itens: musicas.Itens}, vazia}, Proxima: "p2"}
	}}
	inicio := app.Inicio{Fonte: f, Guarda: f}
	_, tem := inicio.Guardado()
	igual(t, false, tem)
	_, tem = app.Inicio{Fonte: f}.Guardado()
	igual(t, false, tem)

	// Um filtro de clima não é lembrado.
	pagina, ok := inicio.Primeira(ctx, &dominio.Chip{Titulo: "Energia", Token: "t"})
	igual(t, true, ok)
	igual(t, "Energia", pagina.Prateleiras[0].Titulo)
	igual(t, true, f.guardada == nil)

	// A primeira página perde as prateleiras vazias e fica guardada sem o token.
	pagina, _ = inicio.Primeira(ctx, nil)
	igual(t, dominio.PaginaFeed{Prateleiras: []dominio.Prateleira{musicas}, Proxima: "p2"}, pagina)
	guardada, tem := inicio.Guardado()
	igual(t, true, tem)
	igual(t, dominio.PaginaFeed{Prateleiras: []dominio.Prateleira{musicas}}, guardada)

	pagina, _ = inicio.Proxima(ctx, "p2")
	igual(t, "p2", pagina.Prateleiras[0].Titulo)
	f.erro = app.ErrCatalogo
	_, ok = inicio.Primeira(ctx, nil)
	igual(t, false, ok)
}

func TestLetrasLembradas(t *testing.T) {
	ctx := context.Background()
	versos := &dominio.LetraDaFaixa{Texto: "versos"}
	f := &fonteFalsa{letra: versos}
	letras := app.NovasLetras(f)
	igual(t, app.DesfechoLetra{Desfecho: app.Achou, Letra: *versos}, letras.Acha(ctx, prova.Faixa("1")))
	letras.Acha(ctx, prova.Faixa("1"))
	igual(t, 1, f.perguntas)

	f = &fonteFalsa{}
	letras = app.NovasLetras(f)
	igual(t, app.NadaAchado, letras.Acha(ctx, prova.Faixa("1")).Desfecho)
	letras.Acha(ctx, prova.Faixa("1"))
	igual(t, 1, f.perguntas)

	// Fora do ar não é lembrado: pergunta de novo.
	f = &fonteFalsa{erro: app.ErrCatalogo}
	letras = app.NovasLetras(f)
	igual(t, app.Indisponivel, letras.Acha(ctx, prova.Faixa("1")).Desfecho)
	f.erro, f.letra = nil, versos
	igual(t, app.Achou, letras.Acha(ctx, prova.Faixa("1")).Desfecho)

	// A memória tem teto: só as últimas faixas ficam.
	for i := range 40 {
		letras.Acha(ctx, prova.Faixa(string(rune('a'+i))))
	}
	antes := f.perguntas
	letras.Acha(ctx, prova.Faixa("1"))
	igual(t, antes+1, f.perguntas)
}
