// SPDX-License-Identifier: GPL-3.0-or-later

package daemon

import (
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"mktunes/internal/app"
	"mktunes/internal/dominio"
	"mktunes/internal/prova"
)

func TestPlayer(t *testing.T) {
	t.Run("toca qualquer lista que a janela mandar, a partir da música escolhida", func(t *testing.T) {
		b := novaBancada(t, "", nil)
		b.toca(1, b.c.musicas...)
		igual(t, "2", b.p.Tocando.ID)
		igual(t, "playing", b.estado("now")["status"])
		igual(t, "Halo", b.estado("now")["title"])
		igual(t, "Halo", b.estado("now")["track"].(map[string]any)["title"])
		igual(t, false, b.estado("now")["live"])
	})
	t.Run("o modo de repetir pode ser posto direto, como os controles do sistema pedem", func(t *testing.T) {
		b := novaBancada(t, "", nil)
		b.manda(`{"cmd":"repeat","mode":"one"}`)
		igual(t, "one", b.estado("now")["repeat"])
		b.manda(`{"cmd":"repeat","mode":"off"}`)
		igual(t, "off", b.estado("now")["repeat"])
	})
	t.Run("o transporte, as edições da fila e o repetir chegam à sessão", func(t *testing.T) {
		b := novaBancada(t, "", nil)
		b.toca(0, append(b.c.musicas, prova.Faixa("3", prova.Titulo("Diamonds")))...)
		b.manda(`{"cmd":"removeUpNext","index":0}`)
		igual(t, []string{"Diamonds"}, titulosDe(b.estado("now"), "upNext"))
		b.manda(`{"cmd":"cycleRepeat"}`)
		igual(t, "all", b.estado("now")["repeat"])
		b.manda(`{"cmd":"toggle"}`)
		igual(t, "paused", b.estado("now")["status"])
		b.manda(`{"cmd":"next"}`)
		igual(t, "Diamonds", b.p.Tocando.Titulo)
		b.manda(`{"cmd":"stopAfterCurrent"}`)
		igual(t, true, b.estado("now")["stopAfterCurrent"])
		b.manda(`{"cmd":"speed","value":1.25}`)
		igual(t, 1.25, b.estado("now")["speed"])
		b.manda(`{"cmd":"seek","seconds":61.5}`)
		igual(t, "seek 1m1.5s", b.p.Chamadas[len(b.p.Chamadas)-1])
	})
	t.Run("a posição viaja na sua mensagem pequena enquanto toca", func(t *testing.T) {
		b := novaBancada(t, "", nil)
		b.toca(0, b.c.musicas...)
		b.p.Pos = 12 * time.Second
		b.l.Avanca(600 * time.Millisecond)
		igual(t, 12.0, b.estado("progress")["position"])
		igual(t, 30.0, b.estado("progress")["duration"])
		_, tem := b.estado("now")["position"]
		igual(t, false, tem)
	})
	t.Run("um salto com a música pausada aparece na hora, sem esperar o tique", func(t *testing.T) {
		b := novaBancada(t, "", nil)
		b.toca(0, b.c.musicas...)
		b.manda(`{"cmd":"toggle"}`)
		igual(t, "paused", b.estado("now")["status"])
		b.manda(`{"cmd":"seek","seconds":20}`)
		igual(t, 20.0, b.estado("progress")["position"])
	})
	t.Run("tocar a seguir e pôr na fila dizem o que fizeram", func(t *testing.T) {
		b := novaBancada(t, "", nil)
		b.toca(0, b.c.musicas[0])
		b.manda(msg{"cmd": "addToQueue", "track": faixaNoFio(prova.Faixa("9", prova.Titulo("Work")))})
		b.manda(msg{"cmd": "playNext", "track": faixaNoFio(prova.Faixa("8", prova.Titulo("Stay")))})
		igual(t, []string{"Stay", "Work"}, titulosDe(b.estado("now"), "upNext"))
		igual(t, []string{"Na fila: Work", "Toca a seguir: Stay"}, b.avisos)
	})
	t.Run("embaralhar toca a lista toda fora de ordem", func(t *testing.T) {
		b := novaBancada(t, "", nil)
		var muitas []dominio.Faixa
		for i := range 30 {
			muitas = append(muitas, prova.Faixa(string(rune('A'+i))))
		}
		b.manda(msg{"cmd": "play", "tracks": faixasNoFio(muitas), "index": 3, "shuffle": true})
		igual(t, 29, len(lista(b.estado("now"), "upNext")))
		b.manda(`{"cmd":"shuffleUpNext"}`)
		b.manda(`{"cmd":"clearUpNext"}`)
		igual(t, 0, len(lista(b.estado("now"), "upNext")))
	})
	t.Run("o timer de dormir mostra o que falta", func(t *testing.T) {
		b := novaBancada(t, "", nil)
		b.toca(0, b.c.musicas[0])
		b.manda(`{"cmd":"sleep","minutes":15}`)
		igual(t, "15:00", b.estado("now")["sleep"])
		b.l.Avanca(61 * time.Second)
		igual(t, "13:59", b.estado("now")["sleep"])
		b.manda(`{"cmd":"sleep","cancel":true}`)
		igual(t, "", b.estado("now")["sleep"])
		b.manda(`{"cmd":"sleep","endOfTrack":true}`)
		igual(t, "fim da faixa", b.estado("now")["sleep"])
		igual(t, true, b.estado("now")["stopAfterCurrent"])
	})
	t.Run("tocar uma coleção inteira, e avisar quando não dá", func(t *testing.T) {
		b := novaBancada(t, "", nil)
		b.manda(`{"cmd":"playCollection","playlistId":"OLAK1"}`)
		igual(t, "Umbrella", b.estado("now")["title"])
		b.c.derruba(true)
		b.manda(`{"cmd":"playCollection","playlistId":"OLAK1"}`)
		igual(t, "Não deu para abrir esta coleção agora.", b.ultimoAviso())
	})
	t.Run("comandos quebrados, desconhecidos e fora de alcance não mudam nada", func(t *testing.T) {
		b := novaBancada(t, "", nil)
		for _, ruim := range []string{"not json", `{"cmd":"fly"}`, `{"cmd":7}`, `[1,2]`, `{"cmd":"play","tracks":[{"id":""}],"index":0}`, `{"cmd":"play","tracks":"x","index":0}`, `{"cmd":"seek","seconds":"NaN"}`, `{"cmd":"speed"}`, `{"cmd":"sleep","minutes":-5}`, `{"cmd":"moveUpNext","from":"a"}`, `{"cmd":"playNext","track":{"id":"x","provider":"MYSPACE"}}`} {
			b.manda(ruim)
		}
		b.manda(msg{"cmd": "play", "tracks": faixasNoFio(b.c.musicas[:1]), "index": 5})
		igual(t, "idle", b.estado("now")["status"])
		igual(t, msg{}, b.estado("now")["track"])
		igual(t, 0, len(b.avisos))
	})
	t.Run("o comando interno quit pede ao daemon para sair", func(t *testing.T) {
		b := novaBancada(t, "", nil)
		b.manda(`{"cmd":"quit"}`)
		igual(t, 1, b.saiu)
	})
}

func TestBusca(t *testing.T) {
	t.Run("tudo por padrão, páginas quando um filtro é escolhido, e a busca fica lembrada", func(t *testing.T) {
		b := novaBancada(t, "", nil)
		b.manda(`{"cmd":"search","query":"rihanna"}`)
		igual(t, "results", b.estado("search")["status"])
		igual(t, "all", b.estado("search")["filter"])
		igual(t, []string{"Umbrella", "Halo"}, titulosDe(b.estado("search"), "tracks"))
		b.manda(`{"cmd":"search","filter":"songs"}`)
		igual(t, []string{"Umbrella", "Halo"}, titulosDe(b.estado("search"), "tracks"))
		igual(t, []any{}, b.estado("search")["sections"])
		b.manda(`{"cmd":"search","filter":"albums"}`)
		igual(t, []string{"Anti"}, titulosDe(b.estado("search"), "items"))
		igual(t, "albums", b.estado("search")["filter"])
		igual(t, []any{"rihanna"}, b.estado("search")["recent"])
		b.manda(`{"cmd":"clearSearch"}`)
		igual(t, "idle", b.estado("search")["status"])
		igual(t, "", b.estado("search")["query"])
		igual(t, []any{}, b.estado("search")["items"])
		b.manda(`{"cmd":"clearRecent"}`)
		igual(t, []any{}, b.estado("search")["recent"])
	})
	t.Run("tudo de uma vez põe o melhor resultado em cima e uma seção por tipo que achou algo", func(t *testing.T) {
		b := novaBancada(t, "", nil)
		b.manda(`{"cmd":"search","query":"Rihanna"}`)
		busca := b.estado("search")
		igual(t, "artist", busca["top"].(map[string]any)["kind"])
		igual(t, "Rihanna", busca["top"].(map[string]any)["title"])
		igual(t, []string{"Artistas", "Álbuns"}, titulosDe(busca, "sections"))
		igual(t, []string{"Anti"}, titulosDe(lista(busca, "sections")[1], "items"))
		igual(t, "albums", lista(busca, "sections")[1]["filter"])
	})
	t.Run("tudo de uma vez ainda mostra as músicas quando as páginas falham", func(t *testing.T) {
		b := novaBancada(t, "", nil)
		b.c.paginasFora = true
		b.manda(`{"cmd":"search","query":"umbrella"}`)
		igual(t, "results", b.estado("search")["status"])
		igual(t, []string{"Umbrella", "Halo"}, titulosDe(b.estado("search"), "tracks"))
		igual(t, "song", b.estado("search")["top"].(map[string]any)["kind"])
	})
	t.Run("diz quando o catálogo está fora do ar, e quando não achou nada", func(t *testing.T) {
		b := novaBancada(t, "", nil)
		b.c.derruba(true)
		b.manda(`{"cmd":"search","query":"rihanna"}`)
		igual(t, "unavailable", b.estado("search")["status"])
		igual(t, nil, b.estado("search")["top"])
		b.c.derruba(false)
		b.c.musicas = nil
		b.manda(`{"cmd":"search","query":"zzz","filter":"songs"}`)
		igual(t, "nothing", b.estado("search")["status"])
	})
	t.Run("sugere quando a digitação pausa", func(t *testing.T) {
		b := novaBancada(t, "", nil)
		b.manda(`{"cmd":"suggest","text":"ri"}`)
		b.l.Avanca(200 * time.Millisecond)
		b.manda(`{"cmd":"suggest","text":"riha"}`)
		igual(t, []any{}, b.estado("search")["suggestions"])
		b.l.Avanca(400 * time.Millisecond)
		igual(t, []any{"riha remix"}, b.estado("search")["suggestions"])
	})
}

func TestInicioDoApp(t *testing.T) {
	t.Run("carrega os chips e as prateleiras ao aparecer, e mais a pedido", func(t *testing.T) {
		b := novaBancada(t, "", nil)
		igual(t, "loading", b.estado("home")["status"])
		b.manda(`{"cmd":"home"}`)
		igual(t, "ready", b.estado("home")["status"])
		igual(t, []string{"Relax", "Festa"}, titulosDe(b.estado("home"), "chips"))
		igual(t, []string{"Escolha a dedo"}, titulosDe(b.estado("home"), "shelves"))
		igual(t, true, b.estado("home")["hasMore"])
		itens := lista(lista(b.estado("home"), "shelves")[0], "items")
		igual(t, []any{"song", "song", "album"}, []any{itens[0]["kind"], itens[1]["kind"], itens[2]["kind"]})
		b.manda(`{"cmd":"homeMore"}`)
		igual(t, []string{"Escolha a dedo", "Mais"}, titulosDe(b.estado("home"), "shelves"))
		igual(t, false, b.estado("home")["hasMore"])
	})
	t.Run("um chip filtra o Início e tocar nele de novo o limpa", func(t *testing.T) {
		b := novaBancada(t, "", nil)
		b.manda(`{"cmd":"home"}`)
		b.manda(`{"cmd":"homeChip","token":"t-festa"}`)
		igual(t, "t-festa", b.estado("home")["chip"])
		igual(t, []string{"Festa"}, titulosDe(b.estado("home"), "shelves"))
		igual(t, []string{"Relax", "Festa"}, titulosDe(b.estado("home"), "chips"))
		b.manda(`{"cmd":"homeChip","token":"t-festa"}`)
		igual(t, "", b.estado("home")["chip"])
	})
	t.Run("fora do ar ele diz, e ainda oferece o que tocou há pouco; tentar de novo recarrega", func(t *testing.T) {
		b := novaBancada(t, "", nil)
		b.toca(0, b.c.musicas[0])
		b.p.Pos = 40 * time.Second
		b.l.Avanca(45 * time.Second)
		b.c.derruba(true)
		b.manda(`{"cmd":"home"}`)
		igual(t, "unavailable", b.estado("home")["status"])
		igual(t, []string{"Umbrella"}, titulosDe(b.estado("home"), "recent"))
		b.c.derruba(false)
		b.manda(`{"cmd":"homeRetry"}`)
		igual(t, "ready", b.estado("home")["status"])
	})
}

func TestPaginas(t *testing.T) {
	t.Run("um álbum abre como página respondida a quem pediu", func(t *testing.T) {
		b := novaBancada(t, "", nil)
		b.manda(`{"cmd":"open","kind":"album","id":"MPRE1","ref":"p7"}`)
		pagina := b.pagina()
		igual(t, []any{"p7", "ready", "album", "Anti", "Rihanna", "2 músicas", "OLAK1", false}, []any{pagina["ref"], pagina["status"], pagina["kind"], pagina["title"], pagina["owner"], pagina["detail"], pagina["playlistId"], pagina["saved"]})
		igual(t, []string{"Umbrella", "Halo"}, titulosDe(pagina, "tracks"))
	})
	t.Run("a página de artista traz as músicas principais e as prateleiras", func(t *testing.T) {
		b := novaBancada(t, "", nil)
		b.manda(`{"cmd":"open","kind":"artist","id":"UC1","ref":"p8"}`)
		pagina := b.pagina()
		igual(t, []any{"artist", "Rihanna", "Cantora", "12 mi ouvintes mensais", "RDEM1"}, []any{pagina["kind"], pagina["title"], pagina["subtitle"], pagina["audience"], pagina["radioPlaylistId"]})
		igual(t, []string{"Umbrella", "Halo"}, titulosDe(pagina, "tracks"))
		igual(t, []string{"Álbuns"}, titulosDe(pagina, "shelves"))
	})
	t.Run("uma página que não carrega avisa", func(t *testing.T) {
		b := novaBancada(t, "", nil)
		b.c.derruba(true)
		b.manda(`{"cmd":"open","kind":"album","id":"MPRE1","ref":"p9"}`)
		igual(t, "unavailable", b.pagina()["status"])
		igual(t, "p9", b.pagina()["ref"])
	})
	t.Run("salvar um álbum o põe na biblioteca, e de novo o tira", func(t *testing.T) {
		b := novaBancada(t, "", nil)
		b.manda(`{"cmd":"toggleSaved","id":"MPRE1"}`) // ainda não foi aberto: nada acontece
		igual(t, 0, len(b.avisos))
		b.manda(`{"cmd":"open","kind":"album","id":"MPRE1","ref":"p1"}`)
		b.manda(`{"cmd":"toggleSaved","id":"MPRE1"}`)
		igual(t, []string{"Anti"}, titulosDe(b.estado("library"), "saved"))
		igual(t, "Álbum salvo na biblioteca", b.ultimoAviso())
		igual(t, map[string]any{"type": "saved", "id": "MPRE1", "saved": true}, b.avulsas[len(b.avulsas)-1])
		b.manda(`{"cmd":"open","kind":"album","id":"MPRE1","ref":"p2"}`)
		igual(t, true, b.pagina()["saved"])
		b.manda(`{"cmd":"toggleSaved","id":"MPRE1"}`)
		igual(t, "Álbum removido da biblioteca", b.ultimoAviso())
		igual(t, 0, len(lista(b.estado("library"), "saved")))
	})
}

func TestBibliotecaDoApp(t *testing.T) {
	t.Run("curtir uma música a lista e marca o que está tocando", func(t *testing.T) {
		b := novaBancada(t, "", nil)
		musica := b.c.musicas[0]
		b.toca(0, musica)
		b.manda(msg{"cmd": "like", "track": faixaNoFio(musica)})
		igual(t, []string{"Umbrella"}, titulosDe(b.estado("library"), "liked"))
		igual(t, []any{"1"}, b.estado("library")["likedIds"])
		igual(t, true, b.estado("now")["liked"])
		igual(t, "Curtida", b.ultimoAviso())
		b.manda(msg{"cmd": "like", "track": faixaNoFio(musica)})
		igual(t, 0, len(lista(b.estado("library"), "liked")))
		igual(t, "Removida das curtidas", b.ultimoAviso())
	})
	t.Run("playlists são criadas, preenchidas, editadas e apagadas", func(t *testing.T) {
		b := novaBancada(t, "", nil)
		b.manda(msg{"cmd": "createPlaylist", "title": " Treino ", "track": faixaNoFio(b.c.musicas[0])})
		igual(t, `Playlist "Treino" criada`, b.ultimoAviso())
		id := lista(b.estado("library"), "playlists")[0]["id"].(string)
		b.manda(msg{"cmd": "addToPlaylist", "id": id, "track": faixaNoFio(b.c.musicas[1])})
		igual(t, "Adicionada a Treino", b.ultimoAviso())
		b.manda(msg{"cmd": "addToPlaylist", "id": id, "track": faixaNoFio(b.c.musicas[1])})
		igual(t, "Já está em Treino", b.ultimoAviso())
		igual(t, []string{"Umbrella", "Halo"}, titulosDe(lista(b.estado("library"), "playlists")[0], "tracks"))
		b.manda(msg{"cmd": "movePlaylistTrack", "id": id, "from": 1, "to": 0})
		b.manda(msg{"cmd": "removePlaylistTrack", "id": id, "index": 1})
		b.manda(msg{"cmd": "renamePlaylist", "id": id, "title": "Corrida"})
		playlist := lista(b.estado("library"), "playlists")[0]
		igual(t, "Corrida", playlist["title"])
		igual(t, []string{"Halo"}, titulosDe(playlist, "tracks"))
		b.manda(msg{"cmd": "deletePlaylist", "id": id})
		igual(t, 0, len(lista(b.estado("library"), "playlists")))
		b.manda(`{"cmd":"createPlaylist","title":"  "}`)
		igual(t, "Dê um nome à playlist.", b.ultimoAviso())
	})
	t.Run("as listas inteligentes e o resumo vêm do que foi ouvido", func(t *testing.T) {
		b := novaBancada(t, "", nil)
		b.toca(0, b.c.musicas[0])
		b.p.Pos = 40 * time.Second
		b.l.Avanca(45 * time.Second)
		b.manda(`{"cmd":"library"}`)
		biblioteca := b.estado("library")
		igual(t, []string{"No repeat", "Tocadas recentemente"}, titulosDe(biblioteca, "smart"))
		resumo := biblioteca["stats"].(map[string]any)
		igual(t, 1.0, resumo["plays"])
		igual(t, "Rihanna", lista(resumo, "artists")[0]["name"])
		igual(t, 1.0, lista(resumo, "tracks")[0]["plays"])
		igual(t, "week", biblioteca["period"])
		b.manda(`{"cmd":"statsPeriod","period":"month"}`)
		igual(t, "month", b.estado("library")["period"])
	})
}

func TestDownloadsDoApp(t *testing.T) {
	t.Run("uma música baixa e depois toca do arquivo", func(t *testing.T) {
		b := novaBancada(t, "", nil)
		musica := prova.Faixa("1", prova.Titulo("Umbrella"), prova.Artista("Rihanna"), prova.SemStream)
		igual(t, "/home/ouvinte/Música/MkTunes", b.estado("downloads")["folder"])
		b.manda(msg{"cmd": "download", "track": faixaNoFio(musica)})
		igual(t, "Baixando Umbrella", b.ultimoAviso())
		b.l.Avanca(300 * time.Millisecond)
		job := lista(b.estado("downloads"), "jobs")[0]
		igual(t, "completed", job["status"])
		igual(t, "/music/Rihanna/Umbrella [1].m4a", job["path"])
		b.toca(0, musica)
		tocado, _ := url.Parse(b.p.Stream.URL)
		igual(t, job["path"], tocado.Path)
		igual(t, "completed", b.estado("now")["download"])
	})
	t.Run("um álbum inteiro baixa de uma vez e diz quantas; rádio não baixa", func(t *testing.T) {
		b := novaBancada(t, "", nil)
		b.manda(msg{"cmd": "downloadAll", "tracks": faixasNoFio(append(b.c.musicas, b.c.estacoes...))})
		b.l.Avanca(300 * time.Millisecond)
		igual(t, 2, len(lista(b.estado("downloads"), "jobs")))
		igual(t, "Baixando 2 músicas", b.ultimoAviso())
		b.manda(msg{"cmd": "download", "track": faixaNoFio(b.c.estacoes[0])})
		igual(t, 2, len(lista(b.estado("downloads"), "jobs")))
	})
	t.Run("remover um download apaga o arquivo", func(t *testing.T) {
		b := novaBancada(t, "", nil)
		b.manda(msg{"cmd": "download", "track": faixaNoFio(b.c.musicas[0])})
		b.l.Avanca(300 * time.Millisecond)
		b.manda(`{"cmd":"removeDownload","id":"1"}`)
		b.l.Avanca(300 * time.Millisecond)
		igual(t, 0, len(lista(b.estado("downloads"), "jobs"))+len(b.d.salvos))
	})
	t.Run("limpar o histórico tira falhas e cancelados mas fica com as músicas no disco", func(t *testing.T) {
		b := novaBancada(t, "", nil)
		b.manda(msg{"cmd": "download", "track": faixaNoFio(b.c.musicas[0])})
		b.l.Avanca(300 * time.Millisecond)
		b.d.quebrado = true
		b.manda(msg{"cmd": "download", "track": faixaNoFio(b.c.musicas[1])})
		b.l.Avanca(300 * time.Millisecond)
		jobs := lista(b.estado("downloads"), "jobs")
		igual(t, []any{"failed", "completed"}, []any{jobs[0]["status"], jobs[1]["status"]})
		igual(t, "Não foi possível salvar a música", jobs[0]["reason"])
		b.manda(`{"cmd":"retryDownload","id":"2"}`)
		b.manda(`{"cmd":"cancelDownload","id":"2"}`)
		b.manda(`{"cmd":"clearDownloadHistory"}`)
		b.l.Avanca(300 * time.Millisecond)
		jobs = lista(b.estado("downloads"), "jobs")
		igual(t, 1, len(jobs))
		igual(t, "Umbrella", jobs[0]["track"].(map[string]any)["title"])
		igual(t, 1, len(b.d.salvos))
	})
}

func TestCompasso(t *testing.T) {
	b := novaBancada(t, "", nil)
	var visto []int
	valor := 0
	compasso := NovoCompasso(b.l, 250*time.Millisecond, func() { visto = append(visto, valor) })
	compasso.Pede()
	// Uma rajada de mudanças chega no máximo uma vez por janela, terminando na última.
	for i := 1; i <= 100; i++ {
		valor = i
		compasso.Pede()
		b.l.Avanca(10 * time.Millisecond)
	}
	b.l.Avanca(time.Second)
	igual(t, 0, visto[0])
	igual(t, 100, visto[len(visto)-1])
	if len(visto) > 6 {
		t.Fatalf("saiu %d vezes: %v", len(visto), visto)
	}
	// Uma mudança sozinha sai na hora.
	antes := len(visto)
	valor = 7
	compasso.Pede()
	igual(t, []int{7}, visto[antes:])
}

func TestMusicasLocaisVolumeRadioELetras(t *testing.T) {
	t.Run("lista as músicas do computador quando pedem, e diz quando não há", func(t *testing.T) {
		b := novaBancada(t, "", nil)
		igual(t, "idle", b.estado("local")["status"])
		b.manda(`{"cmd":"local"}`)
		igual(t, "ready", b.estado("local")["status"])
		igual(t, []string{"Local Um"}, titulosDe(b.estado("local"), "tracks"))
		igual(t, "file:///m/um.mp3", lista(b.estado("local"), "tracks")[0]["streamUrl"])
		b.locais = nil
		b.manda(`{"cmd":"local"}`)
		igual(t, "empty", b.estado("local")["status"])
	})
	t.Run("o volume começa cheio e segue o ouvinte, entre o silêncio e o cheio, e volta depois de reiniciar", func(t *testing.T) {
		guardado := &volumeNaMemoria{}
		b := novaBancada(t, "", guardado)
		igual(t, 100.0, b.estado("volume")["level"])
		igual(t, 1.0, b.volumes[len(b.volumes)-1])
		b.manda(`{"cmd":"volume","level":40}`)
		igual(t, 40.0, b.estado("volume")["level"])
		igual(t, 0.4, b.volumes[len(b.volumes)-1])
		b.manda(`{"cmd":"volume","level":180}`)
		igual(t, 100.0, b.estado("volume")["level"])
		b.manda(`{"cmd":"volume","level":35}`)
		de := novaBancada(t, "", guardado)
		igual(t, 35.0, de.estado("volume")["level"])
		igual(t, 0.35, de.volumes[len(de.volumes)-1])

		arquivo := VolumeEmArquivo{Arquivo: t.TempDir() + "/state/volume"}
		_, tem := arquivo.Carrega()
		igual(t, false, tem)
		arquivo.Salva(42)
		nivel, tem := arquivo.Carrega()
		igual(t, []any{42, true}, []any{nivel, tem})
		os.WriteFile(arquivo.Arquivo, []byte("alto"), 0o600)
		_, tem = arquivo.Carrega()
		igual(t, false, tem)
	})
	t.Run("lista as estações e procura pelo nome", func(t *testing.T) {
		b := novaBancada(t, "", nil)
		igual(t, "idle", b.estado("radio")["status"])
		b.manda(`{"cmd":"radio"}`)
		igual(t, "ready", b.estado("radio")["status"])
		igual(t, []string{"Rádio Um"}, titulosDe(b.estado("radio"), "stations"))
		igual(t, "AO VIVO", lista(b.estado("radio"), "stations")[0]["length"])
		b.manda(`{"cmd":"radio","query":"nada"}`)
		igual(t, "nothing", b.estado("radio")["status"])
		b.c.derruba(true)
		b.manda(`{"cmd":"radio","query":"um"}`)
		igual(t, "unavailable", b.estado("radio")["status"])
		// Uma estação toca ao vivo, sem letra.
		b.toca(0, b.c.estacoes[0])
		igual(t, true, b.estado("now")["live"])
		igual(t, "none", b.estado("lyrics")["status"])
	})
	t.Run("a letra segue a música que toca", func(t *testing.T) {
		b := novaBancada(t, "", nil)
		igual(t, "none", b.estado("lyrics")["status"])
		b.c.letra = &dominio.LetraDaFaixa{Sincronizada: dominio.Letra{{Tempo: time.Second, Texto: "ela, ela"}}}
		b.toca(0, b.c.musicas...)
		letra := b.estado("lyrics")
		igual(t, "synced", letra["status"])
		igual(t, []map[string]any{{"ms": 1000.0, "text": "ela, ela"}}, lista(letra, "lines"))
		b.c.letra = &dominio.LetraDaFaixa{Texto: "só o texto"}
		b.manda(`{"cmd":"next"}`)
		igual(t, "plain", b.estado("lyrics")["status"])
		igual(t, "só o texto", b.estado("lyrics")["text"])
		b.c.derruba(true)
		b.toca(0, prova.Faixa("outra"))
		igual(t, "unavailable", b.estado("lyrics")["status"])
	})
}

func TestAjustesDoApp(t *testing.T) {
	t.Run("as mudanças ficam guardadas e chegam a quem resolve os streams", func(t *testing.T) {
		pasta := t.TempDir()
		b := novaBancada(t, pasta, nil)
		igual(t, map[string]any{"type": "settings", "autoplay": true, "normalizeVolume": true, "audioQuality": "high", "downloadWorkers": 1.0, "maxWorkers": 4.0, "backupsDir": b.backups, "version": "9.9.9"}, b.estado("settings"))
		b.manda(`{"cmd":"setting","key":"autoplay","value":false}`)
		b.manda(`{"cmd":"setting","key":"audioQuality","value":"economy"}`)
		b.manda(`{"cmd":"setting","key":"downloadWorkers","value":3}`)
		b.manda(`{"cmd":"setting","key":"normalizeVolume","value":false}`)
		b.manda(`{"cmd":"setting","key":"inventado","value":true}`)
		b.manda(`{"cmd":"setting","key":"autoplay","value":"sim"}`)
		ajustes := b.estado("settings")
		igual(t, []any{false, "economy", 3.0, false}, []any{ajustes["autoplay"], ajustes["audioQuality"], ajustes["downloadWorkers"], ajustes["normalizeVolume"]})
		igual(t, dominio.QualidadeEconomia, b.ajustes[len(b.ajustes)-1].Qualidade)
		igual(t, 5, len(b.ajustes)) // o da subida e uma vez por mudança de verdade
		de := novaBancada(t, pasta, nil)
		igual(t, "economy", de.estado("settings")["audioQuality"])
		igual(t, dominio.QualidadeEconomia, de.ajustes[0].Qualidade)
	})
	t.Run("um backup escrito num arquivo restaura a biblioteca", func(t *testing.T) {
		b := novaBancada(t, "", nil)
		b.manda(msg{"cmd": "like", "track": faixaNoFio(b.c.musicas[0])})
		b.manda(`{"cmd":"exportBackup"}`)
		caminho := strings.TrimPrefix(b.ultimoAviso(), "Backup salvo em ")
		igual(t, b.backups+"/mktunes-backup-2026-11-22.json", caminho)
		b.manda(msg{"cmd": "like", "track": faixaNoFio(b.c.musicas[0])})
		igual(t, 0, len(lista(b.estado("library"), "liked")))
		b.manda(msg{"cmd": "importBackup", "path": caminho})
		igual(t, []string{"Umbrella"}, titulosDe(b.estado("library"), "liked"))
		igual(t, "Restaurado: 1 curtidas, 0 playlists", b.ultimoAviso())
		os.WriteFile(caminho, []byte("{}"), 0o600)
		b.manda(msg{"cmd": "importBackup", "path": caminho})
		igual(t, "Este arquivo não é um backup do MkTunes.", b.ultimoAviso())
		b.manda(msg{"cmd": "importBackup", "path": caminho + ".não-existe"})
		igual(t, "Não deu para ler "+caminho+".não-existe.", b.ultimoAviso())
	})
}

func TestAFilaEAPosicaoVoltamDepoisDeReiniciar(t *testing.T) {
	pasta := t.TempDir()
	primeiro := novaBancada(t, pasta, nil)
	primeiro.toca(0, primeiro.c.musicas...)
	primeiro.manda(`{"cmd":"cycleRepeat"}`)
	primeiro.p.Pos = 30 * time.Second
	primeiro.l.Avanca(11 * time.Second)

	segundo := novaBancada(t, pasta, nil)
	igual(t, "Umbrella", segundo.estado("now")["title"])
	igual(t, "paused", segundo.estado("now")["status"])
	igual(t, "all", segundo.estado("now")["repeat"])
	igual(t, []string{"Halo"}, titulosDe(segundo.estado("now"), "upNext"))
	igual(t, 30.0, segundo.estado("progress")["position"])
	igual(t, 0, len(segundo.p.Chamadas)-len(segundo.p.Com("skipSilence")))
	segundo.manda(`{"cmd":"toggle"}`)
	igual(t, "playing", segundo.estado("now")["status"])
	igual(t, "seek 30s", segundo.p.Chamadas[len(segundo.p.Chamadas)-1])

	// Ao sair, o daemon guarda onde parou.
	segundo.p.Pos = 77 * time.Second
	segundo.app.SalvaSessao()
	terceiro := novaBancada(t, pasta, nil)
	igual(t, 77.0, terceiro.estado("progress")["position"])
	_ = app.Tocando
}
