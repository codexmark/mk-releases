// SPDX-License-Identifier: GPL-3.0-or-later

package daemon

import (
	"context"

	"mktunes/internal/app"
	"mktunes/internal/dominio"
	"mktunes/internal/laco"
)

// recentesNoInicio é quantas "tocadas recentemente" a fileira do Início mostra.
const recentesNoInicio = 12

// inicio é a aba Início: o feed da última vez na hora, o novo quando chega, os
// chips de clima e mais prateleiras ao rolar.
type inicio struct {
	quadro *Quadro
	laco   laco.Laco
	acervo *app.Acervo
	feed   app.Inicio

	situacao    string
	chips       []dominio.Chip
	chip        *dominio.Chip
	prateleiras []dominio.Prateleira
	proxima     string
	carregaMais bool
	maisFalhou  bool
	recentes    []dominio.Faixa
	carregado   bool
	carregando  laco.Tarefa
	paginando   laco.Tarefa
}

func novoInicio(q *Quadro, l laco.Laco, a *app.Acervo, feed app.Inicio) *inicio {
	i := &inicio{quadro: q, laco: l, acervo: a, feed: feed, situacao: "loading"}
	i.publica()
	return i
}

func (i *inicio) publica() {
	chips := make([]msg, 0, len(i.chips))
	for _, c := range i.chips {
		chips = append(chips, msg{"title": c.Titulo, "token": c.Token})
	}
	escolhido := ""
	if i.chip != nil {
		escolhido = i.chip.Token
	}
	i.quadro.Estado("home", msg{
		"status": i.situacao, "chips": chips, "chip": escolhido, "shelves": prateleirasNoFio(i.prateleiras),
		"hasMore": i.proxima != "", "loadingMore": i.carregaMais, "moreFailed": i.maisFalhou, "recent": faixasNoFio(i.recentes),
	})
}

func (i *inicio) trata(cmd string, a args) bool {
	switch cmd {
	case "home":
		// Pedido cada vez que o Início aparece: o histórico anda enquanto a música toca.
		i.recentes = i.acervo.Listas().Recentes
		if len(i.recentes) > recentesNoInicio {
			i.recentes = i.recentes[:recentesNoInicio]
		}
		i.publica()
		if !i.carregado || i.situacao == "unavailable" {
			i.recarrega(!i.carregado)
		}
	case "homeRetry":
		i.recarrega(false)
	case "homeChip":
		token, _ := a.texto("token")
		var escolhido *dominio.Chip
		for _, c := range i.chips {
			// Tocar de novo no chip escolhido o desmarca.
			if c.Token == token && (i.chip == nil || *i.chip != c) {
				escolhido = &c
			}
		}
		i.chip = escolhido
		i.publica()
		i.recarrega(false)
	case "homeMore":
		i.mais()
	default:
		return false
	}
	return true
}

func (i *inicio) recarrega(mostraGuardado bool) {
	i.carregado = true
	laco.Cancela(i.carregando)
	laco.Cancela(i.paginando)
	chip := i.chip
	busca := func() {
		if len(i.prateleiras) == 0 {
			i.situacao = "loading"
			i.publica()
		}
		i.carregando = i.laco.Vai(func(ctx context.Context) func() {
			pagina, ok := i.feed.Primeira(ctx, chip)
			return func() {
				switch {
				case ok:
					i.mostra(pagina)
				case len(i.prateleiras) == 0:
					i.situacao = "unavailable"
				default:
					// Fica o que está na tela (um Início guardado).
					i.situacao = "ready"
				}
				i.publica()
			}
		})
	}
	if !mostraGuardado || chip != nil {
		busca()
		return
	}
	i.carregando = i.laco.Vai(func(context.Context) func() {
		guardado, tem := i.feed.Guardado()
		return func() {
			if tem {
				i.mostra(guardado)
				i.publica()
			}
			busca()
		}
	})
}

func (i *inicio) mostra(p dominio.PaginaFeed) {
	// A página de um chip não traz chips: a fileira fica, para dar para voltar.
	if len(p.Chips) > 0 {
		i.chips = p.Chips
	}
	i.situacao, i.prateleiras, i.proxima, i.maisFalhou = "ready", p.Prateleiras, p.Proxima, false
}

func (i *inicio) mais() {
	token := i.proxima
	if token == "" || laco.Ativa(i.paginando) {
		return
	}
	i.carregaMais, i.maisFalhou = true, false
	i.publica()
	i.paginando = i.laco.Vai(func(ctx context.Context) func() {
		pagina, ok := i.feed.Proxima(ctx, token)
		return func() {
			i.carregaMais = false
			if ok {
				i.prateleiras = append(append([]dominio.Prateleira(nil), i.prateleiras...), pagina.Prateleiras...)
				i.proxima = pagina.Proxima
			} else {
				i.maisFalhou = true
			}
			i.publica()
		}
	})
}

// ---- Páginas ----------------------------------------------------------------

// abertasGuardadas é quantas coleções abertas ficam lembradas para o "salvar".
const abertasGuardadas = 32

// paginas abre álbuns, playlists e artistas. Cada resposta sai como avulsa com
// o "ref" de quem pediu: uma página que chega atrasada cai onde foi pedida,
// não na página que está na tela agora.
type paginas struct {
	quadro    *Quadro
	laco      laco.Laco
	acervo    *app.Acervo
	navegador app.Navegador
	// abertas são as coleções abertas há pouco, para "salvar" não pedir de novo
	// ao catálogo; ordem diz qual sai quando passa do teto.
	abertas map[string]dominio.PaginaColecao
	ordem   []string
}

func (p *paginas) trata(cmd string, a args) bool {
	switch cmd {
	case "open":
		id, ok := a.texto("id")
		if !ok {
			break
		}
		ref := a.textoOu("ref", "")
		if a.textoOu("kind", "") == "artist" {
			p.abreArtista(id, ref)
		} else {
			p.abreColecao(id, ref)
		}
	case "toggleSaved":
		id, _ := a.texto("id")
		pagina, aberta := p.abertas[id]
		if !aberta {
			break
		}
		estava := p.acervo.AlternaSalvo(pagina)
		coisa := "Playlist"
		if pagina.Tipo == dominio.Album {
			coisa = "Álbum"
		}
		// "Álbum salvo", "Playlist salva": a concordância vem junto.
		acao := map[string]string{"Álbumtrue": "removido da", "Álbumfalse": "salvo na", "Playlisttrue": "removida da", "Playlistfalse": "salva na"}
		chave := coisa + "false"
		if estava {
			chave = coisa + "true"
		}
		p.quadro.Aviso(coisa + " " + acao[chave] + " biblioteca")
		p.quadro.Avulsa(msg{"type": "saved", "id": pagina.ID, "saved": !estava})
	default:
		return false
	}
	return true
}

func (p *paginas) lembra(pagina dominio.PaginaColecao) {
	if _, tem := p.abertas[pagina.ID]; !tem {
		p.ordem = append(p.ordem, pagina.ID)
		if len(p.ordem) > abertasGuardadas {
			delete(p.abertas, p.ordem[0])
			p.ordem = p.ordem[1:]
		}
	}
	p.abertas[pagina.ID] = pagina
}

func (p *paginas) foraDoAr(ref string) {
	p.quadro.Avulsa(msg{"type": "page", "ref": ref, "status": "unavailable"})
}

func (p *paginas) abreColecao(id, ref string) {
	p.laco.Vai(func(ctx context.Context) func() {
		pagina, erro := p.navegador.Colecao(ctx, id)
		return func() {
			if erro != nil {
				p.foraDoAr(ref)
				return
			}
			p.lembra(pagina)
			p.quadro.Avulsa(msg{
				"type": "page", "ref": ref, "status": "ready", "kind": tipoNoFio(pagina.Tipo), "id": pagina.ID,
				"title": pagina.Titulo, "subtitle": pagina.Subtitulo, "owner": pagina.Dono, "detail": pagina.Detalhe,
				"artwork": pagina.Capa, "playlistId": pagina.PlaylistID, "saved": p.acervo.Salvo(id), "tracks": faixasNoFio(pagina.Faixas),
			})
		}
	})
}

func (p *paginas) abreArtista(id, ref string) {
	p.laco.Vai(func(ctx context.Context) func() {
		pagina, erro := p.navegador.Artista(ctx, id)
		return func() {
			if erro != nil {
				p.foraDoAr(ref)
				return
			}
			p.quadro.Avulsa(msg{
				"type": "page", "ref": ref, "status": "ready", "kind": "artist", "id": pagina.ID,
				"title": pagina.Nome, "subtitle": pagina.Descricao, "audience": pagina.Audiencia, "radioPlaylistId": pagina.RadioID,
				"artwork": pagina.Capa, "tracks": faixasNoFio(pagina.Principais), "shelves": prateleirasNoFio(pagina.Prateleiras),
			})
		}
	})
}
