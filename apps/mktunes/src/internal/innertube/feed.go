// SPDX-License-Identifier: GPL-3.0-or-later

package innertube

import (
	"context"
	"strings"
	"time"

	"mktunes/internal/dominio"
)

const (
	inicio         = "FEmusic_home"
	prefixoRadio   = "RDAMVM"
	prefixoNavega  = "VL"
	prefixoDeAlbum = "MPRE"
)

// filtros são os tokens de filtro da busca do YouTube Music da web (o campo
// "params"). Os de álbuns, artistas e playlists vêm da fonte em Kotlin; o de
// músicas segue o mesmo padrão (muda só o campo que escolhe o tipo).
var filtros = map[dominio.Filtro]string{
	dominio.FiltroMusicas:   "EgWKAQIIAWoMEA4QChADEAQQCRAF",
	dominio.FiltroAlbuns:    "EgWKAQIYAWoMEA4QChADEAQQCRAF",
	dominio.FiltroArtistas:  "EgWKAQIgAWoMEA4QChADEAQQCRAF",
	dominio.FiltroPlaylists: "EgeKAQQoAEABagwQDhAKEAMQBBAJEAU%3D",
}

// Feed é o YouTube Music de convidado: o Início, as buscas, as páginas e as
// filas. Cumpre as portas de catálogo do app.
type Feed struct{ Cliente *Cliente }

func (f Feed) Inicio(ctx context.Context, chip *dominio.Chip) (dominio.PaginaFeed, error) {
	campos := map[string]any{"browseId": inicio}
	if chip != nil {
		campos["params"] = chip.Token
	}
	resposta, erro := f.Cliente.Posta(ctx, "browse", campos)
	if erro != nil {
		return dominio.PaginaFeed{}, erro
	}
	return lePaginaFeed(resposta), nil
}

func (f Feed) Mais(ctx context.Context, token string) (dominio.PaginaFeed, error) {
	resposta, erro := f.Cliente.Posta(ctx, "browse", map[string]any{"continuation": token})
	if erro != nil {
		return dominio.PaginaFeed{}, erro
	}
	return lePaginaFeed(resposta), nil
}

// Radio é o mix de uma faixa, sem ela mesma.
func (f Feed) Radio(ctx context.Context, semente dominio.Faixa) ([]dominio.Faixa, error) {
	mix, erro := f.Mix(ctx, semente.ID)
	var outras []dominio.Faixa
	for _, faixa := range mix {
		if faixa.ID != semente.ID {
			outras = append(outras, faixa)
		}
	}
	return outras, erro
}

// Mix é uma música seguida do rádio dela.
func (f Feed) Mix(ctx context.Context, id string) ([]dominio.Faixa, error) {
	return f.fila(ctx, map[string]any{"videoId": id, "playlistId": prefixoRadio + id})
}

func (f Feed) Playlist(ctx context.Context, playlistID string) ([]dominio.Faixa, error) {
	return f.fila(ctx, map[string]any{"playlistId": strings.TrimPrefix(playlistID, prefixoNavega)})
}

func (f Feed) fila(ctx context.Context, campos map[string]any) ([]dominio.Faixa, error) {
	campos["isAudioOnly"] = true
	resposta, erro := f.Cliente.Posta(ctx, "next", campos)
	if erro != nil {
		return nil, erro
	}
	return leFila(resposta), nil
}

func (f Feed) Colecao(ctx context.Context, id string) (dominio.PaginaColecao, error) {
	resposta, erro := f.Cliente.Posta(ctx, "browse", map[string]any{"browseId": id})
	if erro != nil {
		return dominio.PaginaColecao{}, erro
	}
	tipo := dominio.Playlist
	if strings.HasPrefix(id, prefixoDeAlbum) {
		tipo = dominio.Album
	}
	return lePaginaColecao(id, tipo, resposta), nil
}

func (f Feed) Artista(ctx context.Context, id string) (dominio.PaginaArtista, error) {
	resposta, erro := f.Cliente.Posta(ctx, "browse", map[string]any{"browseId": id})
	if erro != nil {
		return dominio.PaginaArtista{}, erro
	}
	return lePaginaArtista(id, resposta), nil
}

func (f Feed) BuscaPaginas(ctx context.Context, consulta string, filtro dominio.Filtro) ([]dominio.Item, error) {
	if filtro == dominio.FiltroMusicas {
		return nil, nil
	}
	resposta, erro := f.busca(ctx, consulta, filtro)
	if erro != nil {
		return nil, erro
	}
	return leBuscaDePaginas(resposta), nil
}

// BuscaMusicas busca músicas pelo mesmo cliente (na fonte em Kotlin ia pelo
// NewPipe).
func (f Feed) BuscaMusicas(ctx context.Context, consulta string) ([]dominio.Faixa, error) {
	resposta, erro := f.busca(ctx, consulta, dominio.FiltroMusicas)
	if erro != nil {
		return nil, erro
	}
	return leBuscaDeMusicas(resposta), nil
}

func (f Feed) busca(ctx context.Context, consulta string, filtro dominio.Filtro) (*No, error) {
	return f.Cliente.Posta(ctx, "search", map[string]any{"query": consulta, "params": filtros[filtro]})
}

func (f Feed) Sugere(ctx context.Context, texto string) ([]string, error) {
	resposta, erro := f.Cliente.Posta(ctx, "music/get_search_suggestions", map[string]any{"input": texto})
	if erro != nil {
		return nil, erro
	}
	return leSugestoes(resposta), nil
}

// Volume pergunta ao YouTube quão alta uma faixa foi medida
// (playerConfig.audioConfig.loudnessDb). É pedido junto com o stream, então
// tem um prazo curto: a música nunca espera por ele.
type Volume struct {
	// Cliente tem de usar o perfil AndroidVR.
	Cliente *Cliente
	Prazo   time.Duration
}

// LoudnessDb devolve nil quando o YouTube não diz a tempo; a normalização é
// então só pulada.
func (v Volume) LoudnessDb(ctx context.Context, id string) *float64 {
	prazo := v.Prazo
	if prazo <= 0 {
		prazo = 1500 * time.Millisecond
	}
	ctx, cancela := context.WithTimeout(ctx, prazo)
	defer cancela()
	resposta, erro := v.Cliente.Posta(ctx, "player", map[string]any{"videoId": id, "contentCheckOk": true, "racyCheckOk": true})
	if erro != nil {
		return nil
	}
	if db, ok := resposta.Obj("playerConfig").Obj("audioConfig").Num("loudnessDb"); ok {
		return &db
	}
	return nil
}
