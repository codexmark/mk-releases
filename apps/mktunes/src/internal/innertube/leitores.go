// SPDX-License-Identifier: GPL-3.0-or-later

package innertube

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"mktunes/internal/dominio"
)

const tamanhoDaCapa = "544"

var (
	tamanhoNaURL = regexp.MustCompile(`=w\d+-h\d+`)
	relogio      = regexp.MustCompile(`^\d{1,3}(:\d{1,2}){1,2}$`)
)

// lePaginaFeed lê uma página do Início ou uma continuação dela.
func lePaginaFeed(resposta *No) dominio.PaginaFeed {
	var pagina dominio.PaginaFeed
	for _, chip := range resposta.Todos("chipCloudChipRenderer") {
		titulo := chip.Obj("text").Runs()
		token := chip.Obj("navigationEndpoint").Obj("browseEndpoint").Str("params")
		// Podcasts ficam de fora.
		if minusculo := strings.ToLower(titulo); token != "" && minusculo != "podcasts" && minusculo != "podcast" {
			pagina.Chips = append(pagina.Chips, dominio.Chip{Titulo: titulo, Token: token})
		}
	}
	for _, prateleira := range resposta.Todos("musicCarouselShelfRenderer") {
		if lida, ok := lePrateleira(prateleira); ok {
			pagina.Prateleiras = append(pagina.Prateleiras, lida)
		}
	}
	pagina.Proxima = resposta.Primeiro("nextContinuationData").Str("continuation")
	return pagina
}

// leFila lê as faixas de uma resposta de "next": um rádio, um álbum ou uma
// playlist como fila.
func leFila(resposta *No) []dominio.Faixa {
	var faixas []dominio.Faixa
	for _, linha := range resposta.Primeiro("playlistPanelRenderer").Lista("contents") {
		video := linha.Obj("playlistPanelVideoRenderer")
		if video == nil {
			video = linha.Obj("playlistPanelVideoWrapperRenderer").Obj("primaryRenderer").Obj("playlistPanelVideoRenderer")
		}
		id := video.Str("videoId")
		if id == "" {
			continue
		}
		faixas = append(faixas, dominio.Faixa{
			ID: id, Provedor: dominio.YouTube,
			Titulo:  video.Obj("title").Runs(),
			Artista: video.Obj("shortBylineText").Runs(),
			Duracao: leRelogio(video.Obj("lengthText").Runs()),
			Capa:    maiorCapa(video),
		})
	}
	return faixas
}

// leRelogio lê "4:09" ou "1:02:03"; zero quando não é uma duração.
func leRelogio(texto string) time.Duration {
	partes := strings.Split(strings.TrimSpace(texto), ":")
	if len(partes) > 3 {
		return 0
	}
	total := 0
	for _, parte := range partes {
		numero, erro := strconv.Atoi(parte)
		if erro != nil {
			return 0
		}
		total = total*60 + numero
	}
	return time.Duration(total) * time.Second
}

func lePrateleira(prateleira *No) (dominio.Prateleira, bool) {
	titulo := prateleira.Obj("header").Obj("musicCarouselShelfBasicHeaderRenderer").Obj("title").Runs()
	if titulo == "" {
		return dominio.Prateleira{}, false
	}
	lida := dominio.Prateleira{Titulo: titulo}
	for _, item := range prateleira.Lista("contents") {
		if cartao := item.Obj("musicTwoRowItemRenderer"); cartao != nil {
			// Um cartão quadrado: playlist, álbum ou artista. Os de vídeo ficam de fora.
			navega := cartao.Obj("navigationEndpoint").Obj("browseEndpoint")
			if lido, ok := itemDePagina(navega, cartao.Obj("title").Runs(), cartao.Obj("subtitle").Runs(), cartao); ok {
				lida.Itens = append(lida.Itens, lido)
			}
		} else if linha := item.Obj("musicResponsiveListItemRenderer"); linha != nil {
			if faixa, ok := leLinhaDeFaixa(linha, "", ""); ok {
				lida.Itens = append(lida.Itens, dominio.ItemDeMusica(faixa))
			}
		}
	}
	return lida, true
}

// itemDePagina monta o cartão de uma página navegável, pelo tipo de página do
// YouTube Music; false para o que não é artista, álbum nem playlist.
func itemDePagina(navega *No, titulo, subtitulo string, dono *No) (dominio.Item, bool) {
	id := navega.Str("browseId")
	if id == "" {
		return dominio.Item{}, false
	}
	item := dominio.Item{ID: id, Titulo: titulo, Subtitulo: subtitulo, Capa: maiorCapa(dono)}
	switch navega.Primeiro("pageType").Texto() {
	case "MUSIC_PAGE_TYPE_ARTIST":
		item.Tipo = dominio.ItemArtista
	case "MUSIC_PAGE_TYPE_ALBUM":
		item.Tipo, item.Colecao = dominio.ItemColecao, dominio.Album
	case "MUSIC_PAGE_TYPE_PLAYLIST":
		item.Tipo, item.Colecao = dominio.ItemColecao, dominio.Playlist
	default:
		return dominio.Item{}, false
	}
	if item.Tipo == dominio.ItemColecao {
		item.PlaylistID = dono.Primeiro("watchPlaylistEndpoint").Str("playlistId")
	}
	return item, true
}

func colunas(linha *No) []*No {
	var textos []*No
	for _, coluna := range linha.Lista("flexColumns") {
		textos = append(textos, coluna.Obj("musicResponsiveListItemFlexColumnRenderer").Obj("text"))
	}
	return textos
}

// leLinhaDeFaixa lê uma linha de música como os álbuns, as playlists e as
// páginas de artista as listam. As linhas de álbum deixam o artista e a capa
// para o cabeçalho da página: daí as reservas.
func leLinhaDeFaixa(linha *No, artistaReserva, capaReserva string) (dominio.Faixa, bool) {
	textos := colunas(linha)
	var primeiro, segunda *No
	if len(textos) > 0 {
		if runs := textos[0].Lista("runs"); len(runs) > 0 {
			primeiro = runs[0]
		}
	}
	if len(textos) > 1 {
		segunda = textos[1]
	}
	id := primeiro.Obj("navigationEndpoint").Obj("watchEndpoint").Str("videoId")
	if id == "" {
		id = linha.Obj("playlistItemData").Str("videoId")
	}
	if id == "" {
		return dominio.Faixa{}, false
	}
	var fixa *No
	if fixas := linha.Lista("fixedColumns"); len(fixas) > 0 {
		fixa = fixas[0]
	}
	faixa := dominio.Faixa{
		ID: id, Provedor: dominio.YouTube,
		Titulo:  primeiro.Str("text"),
		Artista: segunda.RunsIniciais(),
		Duracao: leRelogio(fixa.Obj("musicResponsiveListItemFixedColumnRenderer").Obj("text").Runs()),
		Capa:    maiorCapa(linha),
	}
	if faixa.Artista == "" {
		faixa.Artista = artistaReserva
	}
	if faixa.Capa == "" {
		faixa.Capa = capaReserva
	}
	return faixa, true
}

// maiorCapa pega a maior miniatura do primeiro bloco de miniaturas e pede a
// versão nítida dela.
func maiorCapa(dono *No) string {
	miniaturas := dono.Primeiro("thumbnails")
	if miniaturas == nil || miniaturas.tipo != 'a' {
		return ""
	}
	var melhor *No
	for _, m := range miniaturas.filhos {
		if melhor == nil || m.Int("width") > melhor.Int("width") {
			melhor = m
		}
	}
	return tamanhoNaURL.ReplaceAllString(melhor.Str("url"), "=w"+tamanhoDaCapa+"-h"+tamanhoDaCapa)
}

// lePaginaColecao lê uma página de álbum (MPRE…) ou de playlist (VL…).
func lePaginaColecao(id string, tipo dominio.TipoColecao, resposta *No) dominio.PaginaColecao {
	cabeca := resposta.Primeiro("musicResponsiveHeaderRenderer")
	pagina := dominio.PaginaColecao{
		ID: id, Tipo: tipo,
		Titulo:     cabeca.Obj("title").Runs(),
		Subtitulo:  cabeca.Obj("subtitle").Runs(),
		Dono:       cabeca.Obj("straplineTextOne").Runs(),
		Detalhe:    cabeca.Obj("secondSubtitle").Runs(),
		Capa:       maiorCapa(cabeca),
		PlaylistID: cabeca.Primeiro("watchPlaylistEndpoint").Str("playlistId"),
	}
	for _, linha := range resposta.Primeiro("secondaryContents").Todos("musicResponsiveListItemRenderer") {
		if faixa, ok := leLinhaDeFaixa(linha, pagina.Dono, pagina.Capa); ok {
			pagina.Faixas = append(pagina.Faixas, faixa)
		}
	}
	return pagina
}

// lePaginaArtista lê uma página de artista (UC…): o cabeçalho, as músicas
// principais e as prateleiras de lançamentos e artistas parecidos.
func lePaginaArtista(id string, resposta *No) dominio.PaginaArtista {
	cabeca := resposta.Primeiro("musicImmersiveHeaderRenderer")
	if cabeca == nil {
		cabeca = resposta.Primeiro("musicVisualHeaderRenderer")
	}
	pagina := dominio.PaginaArtista{
		ID:        id,
		Nome:      cabeca.Obj("title").Runs(),
		Descricao: cabeca.Obj("description").Runs(),
		Capa:      maiorCapa(cabeca),
		Audiencia: strings.ReplaceAll(cabeca.Obj("monthlyListenerCount").Runs(), " ", " "),
		RadioID:   cabeca.Obj("startRadioButton").Primeiro("watchEndpoint").Str("playlistId"),
	}
	for _, linha := range resposta.Primeiro("musicShelfRenderer").Todos("musicResponsiveListItemRenderer") {
		if faixa, ok := leLinhaDeFaixa(linha, pagina.Nome, ""); ok {
			pagina.Principais = append(pagina.Principais, faixa)
		}
	}
	for _, prateleira := range resposta.Todos("musicCarouselShelfRenderer") {
		if lida, ok := lePrateleira(prateleira); ok && len(lida.Itens) > 0 {
			pagina.Prateleiras = append(pagina.Prateleiras, lida)
		}
	}
	return pagina
}

// leBuscaDePaginas lê as linhas de álbum, artista e playlist de uma busca
// filtrada.
func leBuscaDePaginas(resposta *No) []dominio.Item {
	var itens []dominio.Item
	for _, linha := range resposta.Todos("musicResponsiveListItemRenderer") {
		textos := colunas(linha)
		titulo, subtitulo := "", ""
		if len(textos) > 0 {
			titulo = textos[0].Runs()
		}
		if len(textos) > 1 {
			subtitulo = textos[1].Runs()
		}
		if item, ok := itemDePagina(linha.Obj("navigationEndpoint").Obj("browseEndpoint"), titulo, subtitulo, linha); ok {
			itens = append(itens, item)
		}
	}
	return itens
}

// rotulosDeMusica são os nomes de tipo que a busca sem filtro põe antes do
// artista ("Música • Fulano • Álbum • 3:45").
var rotulosDeMusica = map[string]bool{"música": true, "musica": true, "song": true, "vídeo": true, "video": true}

// leBuscaDeMusicas lê as músicas de uma busca filtrada por músicas. As linhas
// são como as de uma playlist, com a duração no fim do subtítulo ("Fulano •
// Álbum • 3:45") em vez de numa coluna própria.
func leBuscaDeMusicas(resposta *No) []dominio.Faixa {
	var faixas []dominio.Faixa
	vistas := map[string]bool{}
	for _, linha := range resposta.Todos("musicResponsiveListItemRenderer") {
		faixa, ok := leLinhaDeFaixa(linha, "", "")
		if !ok || vistas[faixa.ID] {
			continue
		}
		vistas[faixa.ID] = true
		if textos := colunas(linha); len(textos) > 1 {
			trechos := textos[1].trechos()
			if len(trechos) > 1 && rotulosDeMusica[strings.ToLower(trechos[0])] {
				trechos = trechos[1:]
				faixa.Artista = trechos[0]
			}
			if ultimo := trechos[len(trechos)-1]; faixa.Duracao == 0 && len(trechos) > 1 && relogio.MatchString(ultimo) {
				faixa.Duracao = leRelogio(ultimo)
			}
		}
		faixas = append(faixas, faixa)
	}
	return faixas
}

const maxSugestoes = 7

// leSugestoes lê o que o YouTube Music sugere para o texto digitado.
func leSugestoes(resposta *No) []string {
	var sugestoes []string
	vistas := map[string]bool{}
	for _, s := range resposta.Todos("searchSuggestionRenderer") {
		texto := strings.TrimSpace(s.Obj("suggestion").Runs())
		if texto == "" || vistas[texto] {
			continue
		}
		vistas[texto] = true
		if sugestoes = append(sugestoes, texto); len(sugestoes) == maxSugestoes {
			break
		}
	}
	return sugestoes
}
