// SPDX-License-Identifier: GPL-3.0-or-later

// Pacote dominio: as regras do MkTunes que não dependem de nada de fora (a
// fila, o que conta como "ouvida", as listas inteligentes, os nomes dos
// arquivos baixados, as letras). Porte do core/domain do MkTunes em Kotlin.
package dominio

import (
	"strings"
	"time"
)

// Provedor diz de onde uma faixa vem. Os nomes são os do protocolo e do backup.
type Provedor string

const (
	Deezer     Provedor = "DEEZER"
	Audius     Provedor = "AUDIUS"
	YouTube    Provedor = "YOUTUBE"
	SoundCloud Provedor = "SOUNDCLOUD"
	Spotify    Provedor = "SPOTIFY"
	Local      Provedor = "LOCAL"
	Radio      Provedor = "RADIO"
)

// ProvedorDe devolve o provedor com esse nome, ou false quando esta versão não
// o conhece (um backup de uma versão mais nova): a faixa é pulada, não um erro.
func ProvedorDe(nome string) (Provedor, bool) {
	switch p := Provedor(nome); p {
	case Deezer, Audius, YouTube, SoundCloud, Spotify, Local, Radio:
		return p, true
	}
	return "", false
}

// Faixa é uma música (ou uma estação de rádio). Capa e Stream vazios querem
// dizer "não tem": o endereço do áudio é resolvido na hora de tocar.
type Faixa struct {
	ID       string
	Provedor Provedor
	Titulo   string
	Artista  string
	Duracao  time.Duration
	Capa     string
	Stream   string
}

// AoVivo: rádio da internet não tem duração, não entra no histórico e não
// puxa autoplay depois dela.
func (f Faixa) AoVivo() bool { return f.Provedor == Radio }

// TamanhoMaxConsulta é o maior texto de busca aceito.
const TamanhoMaxConsulta = 200

// Consulta limpa um texto de busca; false quando fica vazio ou longo demais.
func Consulta(bruto string) (string, bool) {
	limpo := strings.TrimSpace(bruto)
	if limpo == "" || len([]rune(limpo)) > TamanhoMaxConsulta {
		return "", false
	}
	return limpo, true
}

// BuscasRecentes é quantas buscas ficam guardadas.
const BuscasRecentes = 6

// LembraBusca põe a busca no topo das recentes: as seis últimas, a mais nova
// primeiro, sem repetir (ignorando maiúsculas).
func LembraBusca(historico []string, consulta string) []string {
	limpo := strings.TrimSpace(consulta)
	if limpo == "" {
		return historico
	}
	novo := []string{limpo}
	for _, h := range historico {
		if !strings.EqualFold(h, limpo) {
			novo = append(novo, h)
		}
	}
	if len(novo) > BuscasRecentes {
		novo = novo[:BuscasRecentes]
	}
	return novo
}

// Filtro é o que uma busca procura; músicas tocam, o resto abre uma página.
type Filtro string

const (
	FiltroMusicas   Filtro = "SONGS"
	FiltroAlbuns    Filtro = "ALBUMS"
	FiltroArtistas  Filtro = "ARTISTS"
	FiltroPlaylists Filtro = "PLAYLISTS"
)

// FiltroDe lê o nome de um filtro (em qualquer caixa).
func FiltroDe(nome string) (Filtro, bool) {
	switch f := Filtro(strings.ToUpper(nome)); f {
	case FiltroMusicas, FiltroAlbuns, FiltroArtistas, FiltroPlaylists:
		return f, true
	}
	return "", false
}

// TipoColecao separa álbum de playlist.
type TipoColecao string

const (
	Album    TipoColecao = "ALBUM"
	Playlist TipoColecao = "PLAYLIST"
)

// TipoColecaoDe lê o nome guardado de um tipo de coleção.
func TipoColecaoDe(nome string) (TipoColecao, bool) {
	switch t := TipoColecao(nome); t {
	case Album, Playlist:
		return t, true
	}
	return "", false
}

// TipoItem diz o que um cartão do feed é.
type TipoItem int

const (
	ItemMusica TipoItem = iota
	ItemColecao
	ItemArtista
)

// Item é um cartão do feed: uma música, uma coleção (álbum ou playlist) ou um
// artista. Só os campos do seu tipo valem.
type Item struct {
	Tipo      TipoItem
	Faixa     Faixa       // ItemMusica
	ID        string      // coleção e artista
	Colecao   TipoColecao // ItemColecao
	Titulo    string
	Subtitulo string
	Capa      string
	// PlaylistID é a fila que toca a coleção inteira, quando a fonte dá uma.
	PlaylistID string
}

// ItemDeMusica embrulha uma faixa como cartão.
func ItemDeMusica(f Faixa) Item {
	return Item{Tipo: ItemMusica, Faixa: f, Titulo: f.Titulo, Subtitulo: f.Artista, Capa: f.Capa}
}

// Chip é um filtro de clima do Início ("Energia", "Relax"…).
type Chip struct {
	Titulo string
	Token  string
}

// Prateleira é uma fileira de cartões com título.
type Prateleira struct {
	Titulo string
	Itens  []Item
}

// PaginaFeed é uma página do Início; Proxima vazia quer dizer que acabou.
type PaginaFeed struct {
	Chips       []Chip
	Prateleiras []Prateleira
	Proxima     string
}

// PaginaColecao é um álbum ou playlist aberto a partir de um cartão.
type PaginaColecao struct {
	ID         string
	Tipo       TipoColecao
	Titulo     string
	Subtitulo  string // "Álbum • 2026"
	Dono       string // o artista de um álbum; vazio na maioria das playlists
	Detalhe    string // "12 músicas • 38 minutos"
	Capa       string
	PlaylistID string
	Faixas     []Faixa
}

// PaginaArtista é a página de um artista.
type PaginaArtista struct {
	ID          string
	Nome        string
	Descricao   string
	Capa        string
	Principais  []Faixa
	Prateleiras []Prateleira
	// Audiencia como o catálogo escreve ("22,2 mi ouvintes mensais").
	Audiencia string
	// RadioID é a fila de músicas parecidas, quando o catálogo oferece.
	RadioID string
}

// TipoAlvo é para onde um link compartilhado aponta.
type TipoAlvo int

const (
	AlvoMusica TipoAlvo = iota
	AlvoPlaylist
	AlvoAlbum
	AlvoArtista
)

// Alvo é o destino de um link: o tipo e o id (de vídeo ou de navegação).
type Alvo struct {
	Tipo TipoAlvo
	ID   string
}
