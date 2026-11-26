// SPDX-License-Identifier: GPL-3.0-or-later

// Pacote app: os casos de uso do MkTunes (a sessão que toca, os downloads, a
// biblioteca, as buscas) e as portas por onde eles falam com o mundo (o
// catálogo, o tocador, a guarda). Porte do core/application do MkTunes em
// Kotlin. O que tem estado roda no laço; o que só pergunta à rede pode rodar
// fora dele.
package app

import (
	"context"
	"errors"
	"time"

	"mktunes/internal/dominio"
)

// ErrCatalogo: a fonte (YouTube Music, LRCLIB, radio-browser) não respondeu.
var ErrCatalogo = errors.New("catálogo indisponível")

// ErrStream: a fonte não entrega o áudio desta faixa.
var ErrStream = errors.New("stream indisponível")

// ErroDownload é uma falha de download com um motivo que a janela mostra.
type ErroDownload struct{ Motivo string }

func (e *ErroDownload) Error() string { return e.Motivo }

// Stream é um endereço de áudio pronto para tocar.
type Stream struct {
	URL string
	// LoudnessDb é quanto a faixa passa do alvo de volume do YouTube, se ele disse.
	LoudnessDb *float64
	// Volume é como o tocador deve tocá-la (a sessão define, pela normalização).
	Volume float64
	// Variante é a codificação (o itag do YouTube).
	Variante string
}

// Resolvedor transforma uma faixa do catálogo em algo que o tocador abre.
type Resolvedor interface {
	Resolve(ctx context.Context, f dominio.Faixa) (Stream, error)
}

// TipoEvento é o que o tocador conta à sessão.
type TipoEvento int

const (
	Terminou TipoEvento = iota
	FalhouAoTocar
	// MudouPausa: tocar ou pausar pedido por qualquer um (o mpv, as teclas de mídia).
	MudouPausa
	PediuProxima
	PediuAnterior
	// PediuTocar: "tocar" de fora; a sessão decide se é retomar, tentar de novo
	// ou carregar a fila restaurada.
	PediuTocar
	// Carregou: o tocador abriu a faixa que mandaram tocar e o som começa.
	Carregou
)

// Evento do tocador. Tocando vale para MudouPausa; Motivo para FalhouAoTocar.
type Evento struct {
	Tipo    TipoEvento
	Tocando bool
	Motivo  string
}

// Tocador é a saída de áudio. Só o laço o chama.
type Tocador interface {
	Posicao() time.Duration
	// Duracao é a que o stream informa; zero até saber.
	Duracao() time.Duration
	Toca(s Stream, f dominio.Faixa)
	Pausa()
	// Segura cala a faixa antiga enquanto a próxima é resolvida.
	Segura(f dominio.Faixa)
	Continua()
	VaiPara(posicao time.Duration)
	Para()
	Velocidade(v float64)
	PularSilencio(ligado bool)
}

// Catalogo busca músicas.
type Catalogo interface {
	BuscaMusicas(ctx context.Context, consulta string) ([]dominio.Faixa, error)
}

// BuscaDePaginas busca álbuns, artistas ou playlists.
type BuscaDePaginas interface {
	BuscaPaginas(ctx context.Context, consulta string, filtro dominio.Filtro) ([]dominio.Item, error)
}

// FonteDeSugestoes completa o texto que está sendo digitado na busca.
type FonteDeSugestoes interface {
	Sugere(ctx context.Context, texto string) ([]string, error)
}

// FonteDoInicio é o feed editorial (o "Início" do YouTube Music).
type FonteDoInicio interface {
	Inicio(ctx context.Context, chip *dominio.Chip) (dominio.PaginaFeed, error)
	Mais(ctx context.Context, token string) (dominio.PaginaFeed, error)
}

// GuardaDoInicio guarda o último feed, para abrir já com música na tela.
type GuardaDoInicio interface {
	Carrega() (dominio.PaginaFeed, bool)
	Guarda(p dominio.PaginaFeed)
}

// FonteDeFilas dá filas prontas: o rádio de uma faixa, ou todas as faixas de
// uma playlist ou álbum.
type FonteDeFilas interface {
	Radio(ctx context.Context, semente dominio.Faixa) ([]dominio.Faixa, error)
	Playlist(ctx context.Context, playlistID string) ([]dominio.Faixa, error)
}

// Navegador abre páginas de álbum, playlist e artista.
type Navegador interface {
	Colecao(ctx context.Context, id string) (dominio.PaginaColecao, error)
	Artista(ctx context.Context, id string) (dominio.PaginaArtista, error)
}

// FonteDeLetras acha a letra de uma faixa; nil quando a fonte não tem.
type FonteDeLetras interface {
	Acha(ctx context.Context, f dominio.Faixa) (*dominio.LetraDaFaixa, error)
}

// DiretorioDeRadio lista estações de rádio da internet, como faixas ao vivo.
type DiretorioDeRadio interface {
	Topo(ctx context.Context, limite int) ([]dominio.Faixa, error)
	Busca(ctx context.Context, nome string, limite int) ([]dominio.Faixa, error)
}

// Buscador copia um arquivo remoto para a pasta de música.
type Buscador interface {
	// Busca salva a URL em caminhoRelativo, dentro da pasta de downloads, e
	// devolve o caminho absoluto. Cancelado, não deixa arquivo pela metade.
	Busca(ctx context.Context, url, caminhoRelativo string, progresso func(float64)) (string, error)
	Existe(caminho string) bool
	Apaga(caminho string)
}

// SessaoSalva é a fila e a posição que voltam depois de reiniciar.
type SessaoSalva struct {
	Fila    dominio.Retrato
	Posicao time.Duration
}

// Guarda é tudo que fica no disco do ouvinte. Só o laço a chama.
type Guarda interface {
	RegistraTocada(t dominio.Tocada)
	// TocadasDesde devolve as tocadas a partir de inicio, em ordem de tempo.
	TocadasDesde(inicio time.Time) []dominio.Tocada
	EsqueceTocadasAntesDe(corte time.Time)

	// Curtidas, a mais recente primeiro.
	Curtidas() []dominio.Faixa
	Curte(f dominio.Faixa, curtida bool, em time.Time)

	Playlists() []dominio.PlaylistLocal
	SalvaPlaylist(p dominio.PlaylistLocal, em time.Time)
	ApagaPlaylist(id string)

	// Salvos, o mais recente primeiro.
	Salvos() []dominio.ColecaoSalva
	SalvaColecao(c dominio.ColecaoSalva)
	EsqueceColecao(id string)

	Ajustes() dominio.Ajustes
	SalvaAjustes(a dominio.Ajustes)

	Buscas() []string
	SalvaBuscas(b []string)

	SalvaSessao(s SessaoSalva)
	CarregaSessao() (SessaoSalva, bool)

	SalvaDownload(d dominio.Download)
	Downloads() []dominio.Download
	RemoveDownload(id string)
}

// Codec transforma um backup no texto de um arquivo e de volta.
type Codec interface {
	Codifica(b dominio.Backup) string
	// Decodifica devolve false quando o texto não é um backup que esta versão lê.
	Decodifica(texto string) (dominio.Backup, bool)
}

// Sinal avisa quem quiser saber que algo mudou. Só o laço o usa.
type Sinal struct{ ouvintes []func() }

// Ouve registra quem quer ser avisado.
func (s *Sinal) Ouve(f func()) { s.ouvintes = append(s.ouvintes, f) }

// Avisa chama todos, na ordem em que se registraram.
func (s *Sinal) Avisa() {
	for _, f := range s.ouvintes {
		f()
	}
}

// PorProvedor manda cada faixa ao resolvedor do seu provedor, ou toca o link
// que o catálogo já deu.
type PorProvedor map[dominio.Provedor]Resolvedor

func (p PorProvedor) Resolve(ctx context.Context, f dominio.Faixa) (Stream, error) {
	if f.Stream != "" {
		return Stream{URL: f.Stream, Volume: 1}, nil
	}
	r, ok := p[f.Provedor]
	if !ok {
		return Stream{}, errors.Join(ErrStream, errors.New("sem resolvedor para "+string(f.Provedor)))
	}
	return r.Resolve(ctx, f)
}
