// SPDX-License-Identifier: GPL-3.0-or-later

package innertube

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"mktunes/internal/app"
	"mktunes/internal/dominio"
)

// O Feed cumpre as portas de catálogo do app.
var (
	_ app.Catalogo         = Feed{}
	_ app.BuscaDePaginas   = Feed{}
	_ app.FonteDeSugestoes = Feed{}
	_ app.FonteDoInicio    = Feed{}
	_ app.FonteDeFilas     = Feed{}
	_ app.Navegador        = Feed{}
)

func igual(t *testing.T, esperado, obtido any) {
	t.Helper()
	if !reflect.DeepEqual(esperado, obtido) {
		t.Fatalf("esperado %#v\n  obtido %#v", esperado, obtido)
	}
}

// As respostas gravadas são respostas de verdade do WEB_REMIX, aparadas (as
// mesmas dos testes da fonte em Kotlin); a de busca de músicas é deduzida.
func gravada(t *testing.T, nome string) *No {
	t.Helper()
	arquivo, erro := os.Open("testdata/" + nome)
	if erro != nil {
		t.Fatal(erro)
	}
	defer arquivo.Close()
	no, erro := LeJSON(arquivo)
	if erro != nil {
		t.Fatal(erro)
	}
	return no
}

func titulos(prateleiras []dominio.Prateleira) []string {
	var lista []string
	for _, p := range prateleiras {
		lista = append(lista, p.Titulo)
	}
	return lista
}

func TestArvoreLembraAOrdem(t *testing.T) {
	no, erro := LeJSON(strings.NewReader(`{"z":{"alvo":"primeiro"},"a":[{"alvo":"segundo","n":"12","x":3.5,"v":null}],"runs":[{"text":"A"},{"text":" • "},{"text":"B"}]}`))
	if erro != nil {
		t.Fatal(erro)
	}
	igual(t, "primeiro", no.Primeiro("alvo").Texto())
	igual(t, 2, len(no.Todos("alvo")))
	igual(t, 12, no.Lista("a")[0].Int("n"))
	x, ok := no.Lista("a")[0].Num("x")
	igual(t, 3.5, x)
	igual(t, true, ok)
	igual(t, "A • B", no.Runs())
	igual(t, "A", no.RunsIniciais())
	// Um caminho que não existe não quebra: devolve vazio.
	var nulo *No
	igual(t, "", nulo.Obj("x").Obj("y").Str("z")+no.Obj("nada").Runs()+no.Obj("a").Str("x"))
	igual(t, 0, len(nulo.Todos("x"))+len(no.Lista("z")))
	for _, ruim := range []string{"<html>", `{"a":1} sobra`, `{"a":`, ""} {
		if _, erro := LeJSON(strings.NewReader(ruim)); erro == nil {
			t.Fatalf("devia recusar %q", ruim)
		}
	}
}

func TestInicioEContinuacao(t *testing.T) {
	inicio := lePaginaFeed(gravada(t, "home.json"))
	var chips []string
	for _, c := range inicio.Chips {
		chips = append(chips, c.Titulo)
	}
	igual(t, []string{"Energia", "Positividade", "Festa"}, chips) // sem os podcasts
	igual(t, true, strings.HasPrefix(inicio.Chips[0].Token, "ggNCSgQIDBAB"))
	igual(t, []string{"Músicas para churrasco", "Mixes de músicas"}, titulos(inicio.Prateleiras))
	cartao := inicio.Prateleiras[0].Itens[0]
	igual(t, true, strings.Contains(cartao.Capa, "w544-h544"))
	igual(t, dominio.Item{
		Tipo: dominio.ItemColecao, ID: "VLRDCLAK5uy_k5faEHND0iXJljeASESqJ3A8UtRr2FL00", Colecao: dominio.Playlist,
		Titulo: "Topnejo", Subtitulo: "Panda, MJ Records, Zé Neto & Cristiano, Gusttavo Lima",
		Capa: cartao.Capa, PlaylistID: "RDCLAK5uy_k5faEHND0iXJljeASESqJ3A8UtRr2FL00",
	}, cartao)
	musica := inicio.Prateleiras[1].Itens[0]
	igual(t, dominio.ItemMusica, musica.Tipo)
	igual(t, "W40W7L1fhUM", musica.Faixa.ID)
	igual(t, dominio.YouTube, musica.Faixa.Provedor)
	igual(t, "Até Quando", musica.Faixa.Titulo)
	igual(t, "Natanzinho Lima, Ronaldinho Gaúcho e Tu Musica", musica.Faixa.Artista)
	igual(t, "", musica.Faixa.Stream)
	igual(t, true, strings.Contains(musica.Capa, "w544-h544"))
	igual(t, true, strings.HasPrefix(inicio.Proxima, "4qmFsgKhAhIM"))

	mais := lePaginaFeed(gravada(t, "home-continuation.json"))
	igual(t, []string{"Playlists da comunidade em alta", "Hits de hoje", "Lançamentos"}, titulos(mais.Prateleiras))
	album := mais.Prateleiras[2].Itens[0]
	igual(t, dominio.Album, album.Colecao)
	igual(t, "MPREb_ecy08n3bd9O", album.ID)
	igual(t, "OLAK5uy_lzUOzSNayT3L2yLO8-aP8DTluf0woE0GI", album.PlaylistID)
	igual(t, true, mais.Proxima != "")
}

func TestFilaERelogio(t *testing.T) {
	faixas := leFila(gravada(t, "next.json"))
	var ids []string
	for _, f := range faixas {
		ids = append(ids, f.ID)
	}
	igual(t, []string{"n8X9_MgEdCg", "kL8CyVqzmkc", "8ElfdgoIwZU", "ALZHF5UqnU4", "6FNHe3kf8_s"}, ids)
	igual(t, "Jackpot", faixas[1].Titulo)
	igual(t, "TheFatRat", faixas[1].Artista)
	igual(t, 196*time.Second, faixas[1].Duracao)
	igual(t, 0, len(leFila(gravada(t, "home.json"))))

	igual(t, 249*time.Second, leRelogio("4:09"))
	igual(t, 3723*time.Second, leRelogio("1:02:03"))
	igual(t, time.Duration(0), leRelogio("ao vivo"))
	igual(t, time.Duration(0), leRelogio(""))
}

func TestPaginas(t *testing.T) {
	album := lePaginaColecao("MPREb_P7h02JyMUPK", dominio.Album, gravada(t, "album.json"))
	igual(t, "Manuscritos, Capítulo II", album.Titulo)
	igual(t, "Álbum • 2026", album.Subtitulo)
	igual(t, "Marília Mendonça", album.Dono)
	igual(t, "12 músicas • 38 minutos", album.Detalhe)
	igual(t, "OLAK5uy_nyNq0kfDUYmMad7yTE9ugxSnvcuh2ecRA", album.PlaylistID)
	igual(t, true, strings.Contains(album.Capa, "w544-h544"))
	igual(t, 4, len(album.Faixas))
	// As faixas de um álbum levam o artista e a capa do álbum.
	igual(t, dominio.Faixa{ID: "V36Vt07DKzc", Provedor: dominio.YouTube, Titulo: "No Dia Do Seu Casamento", Artista: "Marília Mendonça", Duracao: 172 * time.Second, Capa: album.Capa}, album.Faixas[0])

	playlist := lePaginaColecao("VLRDCLAK5uy_x", dominio.Playlist, gravada(t, "playlist.json"))
	igual(t, "Topnejo", playlist.Titulo)
	igual(t, "47 músicas • 2 horas e 16 minutos", playlist.Detalhe)
	igual(t, []string{"Hugo & Guilherme", "Zé Neto & Cristiano"}, []string{playlist.Faixas[0].Artista, playlist.Faixas[1].Artista})
	igual(t, 135*time.Second, playlist.Faixas[0].Duracao)

	artista := lePaginaArtista("UCJqd5Ejxi_fPcttEQ-E7hFw", gravada(t, "artist.json"))
	igual(t, "Natanzinho Lima", artista.Nome)
	igual(t, true, strings.HasPrefix(artista.Descricao, "Natanzinho Lima é um artista"))
	igual(t, true, strings.HasPrefix(artista.Capa, "https://"))
	igual(t, []string{"Sonho de Amor (Ao Vivo)", "Até Quando", "Me Apaixonei Nessa Morena (Ao Vivo)"}, []string{artista.Principais[0].Titulo, artista.Principais[1].Titulo, artista.Principais[2].Titulo})
	igual(t, []string{"Álbuns", "Singles e EPs"}, titulos(artista.Prateleiras))
	igual(t, dominio.ItemColecao, artista.Prateleiras[0].Itens[0].Tipo)
	igual(t, "22,2 mi ouvintes mensais", artista.Audiencia)
	igual(t, "RDEMoTXFe6mUAQDy4k8p2M9BhA", artista.RadioID)

	// Uma resposta inesperada dá uma página vazia, não um erro.
	vazio, _ := LeJSON(strings.NewReader("{}"))
	igual(t, dominio.PaginaColecao{ID: "x", Tipo: dominio.Album}, lePaginaColecao("x", dominio.Album, vazio))
	igual(t, dominio.PaginaArtista{ID: "x"}, lePaginaArtista("x", vazio))
	igual(t, dominio.PaginaFeed{}, lePaginaFeed(vazio))
}

func TestBuscas(t *testing.T) {
	albuns := leBuscaDePaginas(gravada(t, "search-albums.json"))
	igual(t, 3, len(albuns))
	igual(t, true, strings.HasPrefix(albuns[0].Capa, "https://"))
	igual(t, dominio.Item{Tipo: dominio.ItemColecao, ID: "MPREb_nEt6ccyx0NC", Colecao: dominio.Album, Titulo: "Marília Mendonça (Ao Vivo)", Subtitulo: "Álbum • Marília Mendonça • 2016", Capa: albuns[0].Capa, PlaylistID: "OLAK5uy_koW6YsUXP8k3vlC744FBGobryatX3FtvU"}, albuns[0])

	artista := leBuscaDePaginas(gravada(t, "search-artists.json"))[0]
	igual(t, dominio.ItemArtista, artista.Tipo)
	igual(t, "UCj14PDgkcQQMz6v-g2eruBA", artista.ID)
	igual(t, "Marília Mendonça", artista.Titulo)
	igual(t, true, strings.HasPrefix(artista.Subtitulo, "Artista"))

	playlist := leBuscaDePaginas(gravada(t, "search-playlists.json"))[0]
	igual(t, "VLPLvgcQr40gK92qvJdfq11CIIIosGUPGBMk", playlist.ID)
	igual(t, dominio.Playlist, playlist.Colecao)

	sugestoes := leSugestoes(gravada(t, "search-suggestions.json"))
	igual(t, []string{"racionais", "racionais mc's", "racionais vida loka"}, sugestoes[:3])
	igual(t, true, len(sugestoes) <= 7)
	vazio, _ := LeJSON(strings.NewReader("{}"))
	igual(t, 0, len(leSugestoes(vazio))+len(leBuscaDePaginas(vazio))+len(leBuscaDeMusicas(vazio)))
}

// A forma desta resposta é deduzida (não há gravação de verdade): linhas como
// as de playlist, com a duração no fim do subtítulo.
func TestBuscaDeMusicasDeduzida(t *testing.T) {
	musicas := leBuscaDeMusicas(gravada(t, "search-songs.DEDUZIDO.json"))
	igual(t, []dominio.Faixa{
		{ID: "W40W7L1fhUM", Provedor: dominio.YouTube, Titulo: "Até Quando", Artista: "Natanzinho Lima e Ronaldinho Gaúcho", Duracao: 144 * time.Second, Capa: "https://lh3.googleusercontent.com/capa-um=w544-h544-l90-rj"},
		// Sem link no título, o id vem de playlistItemData; o rótulo "Música" não é o artista.
		{ID: "qUYepQ_1abc", Provedor: dominio.YouTube, Titulo: "Sonho de Amor (Ao Vivo)", Artista: "Natanzinho Lima", Duracao: 3794 * time.Second, Capa: "https://lh3.googleusercontent.com/capa-dois=w544-h544-l90-rj"},
	}, musicas)
	// As linhas dos álbuns (sem vídeo) não viram músicas.
	igual(t, 0, len(leBuscaDeMusicas(gravada(t, "search-albums.json"))))
}

// servidor é um YouTube de mentira: responde as gravações na ordem e guarda os pedidos.
type servidor struct {
	*httptest.Server
	respostas []func(w http.ResponseWriter)
	pedidos   []pedido
}

type pedido struct {
	caminho   string
	cabecalho http.Header
	corpo     map[string]any
}

func novoServidor(t *testing.T) *servidor {
	s := &servidor{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bruto, _ := io.ReadAll(r.Body)
		p := pedido{caminho: r.URL.Path + "?" + r.URL.RawQuery, cabecalho: r.Header}
		_ = json.Unmarshal(bruto, &p.corpo)
		s.pedidos = append(s.pedidos, p)
		if len(s.respostas) == 0 {
			w.Write([]byte("{}"))
			return
		}
		responde := s.respostas[0]
		s.respostas = s.respostas[1:]
		responde(w)
	}))
	t.Cleanup(s.Close)
	return s
}

func (s *servidor) texto(corpo string) {
	s.respostas = append(s.respostas, func(w http.ResponseWriter) { w.Write([]byte(corpo)) })
}

func (s *servidor) gravada(t *testing.T, nome string) {
	bruto, erro := os.ReadFile("testdata/" + nome)
	if erro != nil {
		t.Fatal(erro)
	}
	s.texto(string(bruto))
}

func (s *servidor) cliente(perfil Perfil, visitante *Visitante) *Cliente {
	return &Cliente{HTTP: s.Client(), Perfil: perfil, Base: s.URL + "/youtubei/v1/", Idioma: "pt", Regiao: "BR", Visitante: visitante}
}

func (s *servidor) ultimo() pedido { return s.pedidos[len(s.pedidos)-1] }

func doCliente(p pedido) map[string]any {
	return p.corpo["context"].(map[string]any)["client"].(map[string]any)
}

func TestCliente(t *testing.T) {
	ctx := context.Background()
	t.Run("posta no endpoint como o YouTube Music da web", func(t *testing.T) {
		s := novoServidor(t)
		if _, erro := s.cliente(WebRemix, &Visitante{}).Posta(ctx, "browse", map[string]any{"browseId": "FEmusic_home"}); erro != nil {
			t.Fatal(erro)
		}
		p := s.ultimo()
		igual(t, "/youtubei/v1/browse?prettyPrint=false", p.caminho)
		igual(t, "67", p.cabecalho.Get("X-YouTube-Client-Name"))
		igual(t, "https://music.youtube.com", p.cabecalho.Get("Origin"))
		igual(t, WebRemix.UserAgent, p.cabecalho.Get("User-Agent"))
		igual(t, "WEB_REMIX", doCliente(p)["clientName"])
		igual(t, "pt", doCliente(p)["hl"])
		igual(t, "BR", doCliente(p)["gl"])
		igual(t, "FEmusic_home", p.corpo["browseId"])
		_, tem := doCliente(p)["visitorData"]
		igual(t, false, tem) // começa sem identidade de visitante
	})
	t.Run("guarda a identidade que o YouTube entrega, que as continuações exigem, e a divide com o cliente do headset", func(t *testing.T) {
		s := novoServidor(t)
		s.texto(`{"responseContext":{"visitorData":"CgtWb"}}`)
		visitante := &Visitante{}
		web := s.cliente(WebRemix, visitante)
		web.Posta(ctx, "browse", nil)
		web.Posta(ctx, "browse", map[string]any{"continuation": "next-page"})
		igual(t, "CgtWb", s.ultimo().cabecalho.Get("X-Goog-Visitor-Id"))
		igual(t, "CgtWb", doCliente(s.ultimo())["visitorData"])

		s.cliente(AndroidVR, visitante).Posta(ctx, "player", map[string]any{"videoId": "x"})
		p := s.ultimo()
		igual(t, "28", p.cabecalho.Get("X-YouTube-Client-Name"))
		igual(t, "", p.cabecalho.Get("Origin"))
		igual(t, "CgtWb", p.cabecalho.Get("X-Goog-Visitor-Id"))
		igual(t, "ANDROID_VR", doCliente(p)["clientName"])
		igual(t, "32", doCliente(p)["androidSdkVersion"])
	})
	t.Run("um erro HTTP, uma resposta que não se lê ou a rede fora são o catálogo indisponível", func(t *testing.T) {
		s := novoServidor(t)
		s.respostas = append(s.respostas, func(w http.ResponseWriter) { w.WriteHeader(500) })
		s.texto("<html>")
		s.texto("[1]")
		c := s.cliente(WebRemix, nil)
		for range 3 {
			if _, erro := c.Posta(ctx, "browse", nil); !errors.Is(erro, app.ErrCatalogo) {
				t.Fatalf("erro: %v", erro)
			}
		}
		s.Close()
		if _, erro := c.Posta(ctx, "browse", nil); !errors.Is(erro, app.ErrCatalogo) {
			t.Fatalf("erro: %v", erro)
		}
	})
}

func TestFeed(t *testing.T) {
	ctx := context.Background()
	s := novoServidor(t)
	feed := Feed{Cliente: s.cliente(WebRemix, &Visitante{})}

	s.gravada(t, "home.json")
	pagina, erro := feed.Inicio(ctx, nil)
	igual(t, nil, erro)
	igual(t, 2, len(pagina.Prateleiras))
	igual(t, "/youtubei/v1/browse?prettyPrint=false", s.ultimo().caminho)
	igual(t, "FEmusic_home", s.ultimo().corpo["browseId"])

	feed.Inicio(ctx, &dominio.Chip{Titulo: "Energia", Token: "mood-token"})
	igual(t, "mood-token", s.ultimo().corpo["params"])

	s.gravada(t, "home-continuation.json")
	pagina, _ = feed.Mais(ctx, "page-2")
	igual(t, "page-2", s.ultimo().corpo["continuation"])
	igual(t, 3, len(pagina.Prateleiras))

	// O rádio de uma música é o mix RDAMVM dela, sem ela mesma.
	s.gravada(t, "next.json")
	radio, _ := feed.Radio(ctx, dominio.Faixa{ID: "n8X9_MgEdCg"})
	igual(t, "/youtubei/v1/next?prettyPrint=false", s.ultimo().caminho)
	igual(t, "n8X9_MgEdCg", s.ultimo().corpo["videoId"])
	igual(t, "RDAMVMn8X9_MgEdCg", s.ultimo().corpo["playlistId"])
	igual(t, true, s.ultimo().corpo["isAudioOnly"])
	igual(t, 4, len(radio))
	s.gravada(t, "next.json")
	mix, _ := feed.Mix(ctx, "n8X9_MgEdCg")
	igual(t, "Unity", mix[0].Titulo)
	igual(t, 5, len(mix))

	// Uma playlist toca pela fila dela, sem o prefixo de navegação.
	s.gravada(t, "next.json")
	faixas, _ := feed.Playlist(ctx, "VLRDCLAK5uy_k5faEHND0iXJljeASESqJ3A8UtRr2FL00")
	igual(t, "RDCLAK5uy_k5faEHND0iXJljeASESqJ3A8UtRr2FL00", s.ultimo().corpo["playlistId"])
	igual(t, 5, len(faixas))

	s.gravada(t, "album.json")
	album, _ := feed.Colecao(ctx, "MPREb_P7h02JyMUPK")
	igual(t, "MPREb_P7h02JyMUPK", s.ultimo().corpo["browseId"])
	igual(t, dominio.Album, album.Tipo)
	igual(t, "Manuscritos, Capítulo II", album.Titulo)
	s.gravada(t, "playlist.json")
	playlist, _ := feed.Colecao(ctx, "VLRDCLAK5uy_x")
	igual(t, dominio.Playlist, playlist.Tipo)
	s.gravada(t, "artist.json")
	artista, _ := feed.Artista(ctx, "UCJqd5Ejxi_fPcttEQ-E7hFw")
	igual(t, "UCJqd5Ejxi_fPcttEQ-E7hFw", s.ultimo().corpo["browseId"])
	igual(t, "Natanzinho Lima", artista.Nome)

	// As buscas usam os filtros do YouTube Music.
	for filtro, esperado := range map[dominio.Filtro]string{
		dominio.FiltroAlbuns:    "EgWKAQIYAWoMEA4QChADEAQQCRAF",
		dominio.FiltroArtistas:  "EgWKAQIgAWoMEA4QChADEAQQCRAF",
		dominio.FiltroPlaylists: "EgeKAQQoAEABagwQDhAKEAMQBBAJEAU%3D",
	} {
		s.gravada(t, "search-albums.json")
		itens, _ := feed.BuscaPaginas(ctx, "marilia mendonca", filtro)
		igual(t, 3, len(itens))
		igual(t, "/youtubei/v1/search?prettyPrint=false", s.ultimo().caminho)
		igual(t, "marilia mendonca", s.ultimo().corpo["query"])
		igual(t, esperado, s.ultimo().corpo["params"])
	}
	s.gravada(t, "search-songs.DEDUZIDO.json")
	musicas, _ := feed.BuscaMusicas(ctx, "ate quando")
	igual(t, 2, len(musicas))
	igual(t, "EgWKAQIIAWoMEA4QChADEAQQCRAF", s.ultimo().corpo["params"])
	igual(t, "ate quando", s.ultimo().corpo["query"])
	antes := len(s.pedidos)
	nada, erro := feed.BuscaPaginas(ctx, "x", dominio.FiltroMusicas)
	igual(t, 0, len(nada))
	igual(t, nil, erro)
	igual(t, antes, len(s.pedidos))

	s.gravada(t, "search-suggestions.json")
	sugestoes, _ := feed.Sugere(ctx, "racion")
	igual(t, "/youtubei/v1/music/get_search_suggestions?prettyPrint=false", s.ultimo().caminho)
	igual(t, "racion", s.ultimo().corpo["input"])
	igual(t, "racionais", sugestoes[0])

	s.Close()
	_, erro = feed.Inicio(ctx, nil)
	igual(t, true, errors.Is(erro, app.ErrCatalogo))
}

func TestVolumeMedido(t *testing.T) {
	ctx := context.Background()
	s := novoServidor(t)
	volume := Volume{Cliente: s.cliente(AndroidVR, nil)}
	s.gravada(t, "player-vr.json")
	db := volume.LoudnessDb(ctx, "n8X9_MgEdCg")
	if db == nil || *db != 6.68 {
		t.Fatalf("loudness: %v", db)
	}
	igual(t, "n8X9_MgEdCg", s.ultimo().corpo["videoId"])
	// Desiste em silêncio quando o YouTube não diz.
	s.texto(`{"playabilityStatus":{"status":"LOGIN_REQUIRED"}}`)
	s.respostas = append(s.respostas, func(w http.ResponseWriter) { w.WriteHeader(500) })
	igual(t, true, volume.LoudnessDb(ctx, "a") == nil && volume.LoudnessDb(ctx, "b") == nil)
	// E nunca segura a música por muito tempo.
	solta := make(chan struct{})
	s.respostas = append(s.respostas, func(w http.ResponseWriter) { <-solta })
	rapido := Volume{Cliente: s.cliente(AndroidVR, nil), Prazo: 200 * time.Millisecond}
	inicio := time.Now()
	igual(t, true, rapido.LoudnessDb(ctx, "slow") == nil)
	close(solta)
	if time.Since(inicio) > 2*time.Second {
		t.Fatal("esperou demais")
	}
}
