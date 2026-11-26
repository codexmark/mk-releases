// SPDX-License-Identifier: GPL-3.0-or-later

package daemon

import (
	"bufio"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"mktunes/internal/dominio"
	"mktunes/internal/laco"
	"mktunes/internal/prova"
)

func TestFaixaVaiEVolta(t *testing.T) {
	faixa := dominio.Faixa{ID: "abc", Provedor: dominio.YouTube, Titulo: "Umbrella", Artista: "Rihanna", Duracao: 275 * time.Second, Capa: "https://img/a.jpg"}
	noFio := faixaNoFio(faixa)
	igual(t, "4:35", noFio["length"])
	igual(t, int64(275000), noFio["durationMs"])
	_, temStream := noFio["streamUrl"]
	igual(t, false, temStream)
	var lida map[string]any
	b := novaBancada(t, "", nil)
	b.app.Quadro.Estado("teste", msg{"track": noFio})
	lida = b.estado("teste")["track"].(map[string]any)
	de, ok := faixaDoFio(lida)
	igual(t, true, ok)
	igual(t, faixa, de)

	// Uma música sem duração conhecida não mostra "0:00".
	igual(t, "", faixaNoFio(prova.Faixa("1", prova.Duracao(0)))["length"])
	igual(t, "https://cdn.example/1.mp3", faixaNoFio(prova.Faixa("1"))["streamUrl"])
	// Uma faixa que a janela estragou é recusada em vez de adivinhada.
	for _, ruim := range []any{nil, "texto", map[string]any{}, map[string]any{"id": " ", "provider": "YOUTUBE"}, map[string]any{"id": "x", "provider": "MYSPACE"}, map[string]any{"id": 7, "provider": "YOUTUBE"}} {
		if _, ok := faixaDoFio(ruim); ok {
			t.Fatalf("devia recusar %#v", ruim)
		}
	}
	igual(t, 1, len(faixasDoFio([]any{map[string]any{"id": "a", "provider": "LOCAL"}, "lixo", map[string]any{}})))
	igual(t, 0, len(faixasDoFio("não é lista")))
}

func TestItensEDownloadsNoFio(t *testing.T) {
	album := dominio.Item{Tipo: dominio.ItemColecao, ID: "MPRE1", Colecao: dominio.Album, Titulo: "Anti", Subtitulo: "Rihanna", PlaylistID: "OLAK1"}
	igual(t, msg{"kind": "album", "id": "MPRE1", "title": "Anti", "subtitle": "Rihanna", "artwork": "", "playlistId": "OLAK1"}, itemNoFio(album))
	igual(t, "playlist", itemNoFio(dominio.Item{Tipo: dominio.ItemColecao, Colecao: dominio.Playlist})["kind"])
	igual(t, "artist", itemNoFio(dominio.Item{Tipo: dominio.ItemArtista, ID: "UC1"})["kind"])
	musica := itemNoFio(dominio.ItemDeMusica(prova.Faixa("1")))
	igual(t, []any{"song", "Artist 1", "Track 1"}, []any{musica["kind"], musica["subtitle"], musica["title"]})

	job := dominio.Download{Faixa: prova.Faixa("1"), Estado: dominio.Baixando, Progresso: 0.42}
	igual(t, []any{"downloading", 42}, []any{downloadNoFio(job)["status"], downloadNoFio(job)["percent"]})
	job.Estado, job.Motivo = dominio.Falhou, "Sem rede"
	igual(t, []any{"failed", "Sem rede"}, []any{downloadNoFio(job)["status"], downloadNoFio(job)["reason"]})
	igual(t, "queued", downloadNoFio(dominio.Download{Estado: dominio.NaFila})["status"])
	igual(t, "cancelled", downloadNoFio(dominio.Download{Estado: dominio.Cancelado})["status"])
	igual(t, "/m/a.m4a", downloadNoFio(dominio.Download{Estado: dominio.Concluido, Caminho: "/m/a.m4a"})["path"])
	igual(t, "2:05", relogio(125*time.Second))
}

func TestMelhorResultado(t *testing.T) {
	artista := dominio.Item{Tipo: dominio.ItemArtista, ID: "UC1", Titulo: "Natanzinho Lima", Subtitulo: "Artista"}
	album := dominio.Item{Tipo: dominio.ItemColecao, ID: "MPRE1", Colecao: dominio.Album, Titulo: "Na Favela", Subtitulo: "Natanzinho Lima", PlaylistID: "OLAK1"}
	musica := prova.Faixa("1", prova.Titulo("Sonho de Amor"), prova.Artista("Natanzinho Lima"))
	pega := func(consulta string, artistas, albuns []dominio.Item, musicas []dominio.Faixa) any {
		item, tem := MelhorResultado(consulta, artistas, albuns, musicas)
		if !tem {
			return nil
		}
		return item
	}
	// O artista quando a busca é o nome dele, em qualquer caixa e com ou sem acento.
	igual(t, any(artista), pega("natanzinho  LIMA", []dominio.Item{artista}, []dominio.Item{album}, []dominio.Faixa{musica}))
	joao := dominio.Item{Tipo: dominio.ItemArtista, ID: "UC2", Titulo: "João Gomes"}
	igual(t, any(joao), pega("joao gomes", []dominio.Item{joao}, nil, []dominio.Faixa{musica}))
	igual(t, any(joao), pega("joão gomes", []dominio.Item{joao}, nil, []dominio.Faixa{musica}))
	// O álbum quando a busca é o título dele.
	igual(t, any(album), pega("na favela", []dominio.Item{artista}, []dominio.Item{album}, []dominio.Faixa{musica}))
	// Senão a primeira música, que é o melhor palpite do catálogo.
	igual(t, any(dominio.ItemDeMusica(musica)), pega("sonho de amor natanzinho", []dominio.Item{artista}, []dominio.Item{album}, []dominio.Faixa{musica}))
	// Sem músicas, a primeira página achada, ou nada.
	igual(t, any(artista), pega("x", []dominio.Item{artista}, []dominio.Item{album}, nil))
	igual(t, any(album), pega("x", nil, []dominio.Item{album}, nil))
	igual(t, nil, pega("x", nil, nil, nil))
}

func TestQuadroEntregaOUltimoEstadoEAsAvulsas(t *testing.T) {
	q := NovoQuadro()
	q.Estado("now", msg{"n": 1})
	q.Estado("volume", msg{"level": 100})
	a := q.Assina()
	igual(t, []string{`{"n":1,"type":"now"}`, `{"level":100,"type":"volume"}`}, a.Tira())
	// Um estado igual ao anterior não é repassado; um novo toma o lugar do que a
	// janela ainda não leu.
	q.Estado("now", msg{"n": 1})
	igual(t, 0, len(a.Tira()))
	q.Estado("now", msg{"n": 2})
	q.Aviso("Curtida <3 & cia")
	q.Estado("now", msg{"n": 3})
	igual(t, []string{`{"n":3,"type":"now"}`, `{"text":"Curtida <3 & cia","type":"notice"}`}, a.Tira())
	igual(t, []string{`{"n":3,"type":"now"}`, `{"level":100,"type":"volume"}`}, q.Linhas())
	// Uma janela lenta não acumula sem fim: as avulsas mais velhas caem.
	for i := range 200 {
		q.Avulsa(msg{"i": i})
	}
	sobra := a.Tira()
	igual(t, avulsasNaFila, len(sobra))
	igual(t, `{"i":199}`, sobra[len(sobra)-1])
	q.Desliga(a)
	q.Estado("now", msg{"n": 4})
	igual(t, 0, len(a.Tira()))
}

// pastaCurta dá uma pasta temporária de nome curto: o caminho de um socket Unix
// tem limite de tamanho, e o nome do teste não cabe nele.
func pastaCurta(t *testing.T) string {
	t.Helper()
	pasta, erro := os.MkdirTemp("", "mkt")
	if erro != nil {
		t.Fatal(erro)
	}
	t.Cleanup(func() { os.RemoveAll(pasta) })
	return pasta
}

// lacoRodando roda um laço de verdade numa goroutine, para o servidor.
func lacoRodando(t *testing.T) *laco.Real {
	l := laco.Novo()
	go l.Roda()
	t.Cleanup(l.Fecha)
	return l
}

func leLinha(t *testing.T, leitor *bufio.Reader, conexao net.Conn) string {
	t.Helper()
	conexao.SetReadDeadline(time.Now().Add(5 * time.Second))
	linha, erro := leitor.ReadString('\n')
	if erro != nil {
		t.Fatalf("a janela não recebeu nada: %v", erro)
	}
	return strings.TrimSuffix(linha, "\n")
}

func TestServidorDoSocket(t *testing.T) {
	t.Run("a janela recebe o estado atual, depois cada mudança, e os comandos dela são tratados", func(t *testing.T) {
		l := lacoRodando(t)
		q := NovoQuadro()
		recebidos := make(chan string, 4)
		socket := filepath.Join(pastaCurta(t), "mktunes", "mktunes.sock")
		l.Posta(func() { q.Estado("now", msg{"n": 1}) })
		s, erro := Serve(socket, q, l, func(linha string) { recebidos <- linha })
		if erro != nil {
			t.Fatal(erro)
		}
		defer s.Fecha()
		conexao, erro := net.Dial("unix", socket)
		if erro != nil {
			t.Fatal(erro)
		}
		defer conexao.Close()
		leitor := bufio.NewReader(conexao)
		igual(t, `{"n":1,"type":"now"}`, leLinha(t, leitor, conexao))
		l.Posta(func() { q.Estado("now", msg{"n": 2}) })
		igual(t, `{"n":2,"type":"now"}`, leLinha(t, leitor, conexao))
		l.Posta(func() { q.Aviso("Curtida") })
		igual(t, `{"text":"Curtida","type":"notice"}`, leLinha(t, leitor, conexao))
		conexao.Write([]byte(`{"cmd":"toggle"}` + "\n"))
		// Uma linha grande (uma biblioteca inteira num "play") também passa.
		grande := `{"cmd":"play","pad":"` + strings.Repeat("x", 300*1024) + `"}`
		conexao.Write([]byte(grande + "\n"))
		select {
		case linha := <-recebidos:
			igual(t, `{"cmd":"toggle"}`, linha)
		case <-time.After(5 * time.Second):
			t.Fatal("o comando não chegou")
		}
		igual(t, len(grande), len(<-recebidos))
		// Só o usuário que subiu o daemon abre o socket.
		info, _ := os.Stat(socket)
		igual(t, os.FileMode(0o600), info.Mode().Perm())
		pasta, _ := os.Stat(filepath.Dir(socket))
		igual(t, os.FileMode(0o700), pasta.Mode().Perm())
		igual(t, true, Responde(socket))

		// Fechar solta a janela e apaga o socket.
		s.Fecha()
		conexao.SetReadDeadline(time.Now().Add(5 * time.Second))
		if _, erro := leitor.ReadString('\n'); erro == nil {
			t.Fatal("a conexão devia ter sido fechada")
		}
		igual(t, false, Responde(socket))
		if _, erro := os.Lstat(socket); erro == nil {
			t.Fatal("o socket devia ter sido apagado")
		}
	})
	t.Run("recusa subir quando outro daemon já responde no socket", func(t *testing.T) {
		l := lacoRodando(t)
		socket := filepath.Join(pastaCurta(t), "mktunes.sock")
		primeiro, erro := Serve(socket, NovoQuadro(), l, func(string) {})
		if erro != nil {
			t.Fatal(erro)
		}
		defer primeiro.Fecha()
		_, erro = Serve(socket, NovoQuadro(), l, func(string) {})
		igual(t, true, errors.Is(erro, ErrJaRodando))
		igual(t, true, Responde(socket))
	})
	t.Run("toma um arquivo de socket deixado por um daemon que morreu", func(t *testing.T) {
		l := lacoRodando(t)
		socket := filepath.Join(pastaCurta(t), "mktunes.sock")
		os.WriteFile(socket, nil, 0o600)
		igual(t, false, Responde(socket))
		s, erro := Serve(socket, NovoQuadro(), l, func(string) {})
		if erro != nil {
			t.Fatal(erro)
		}
		defer s.Fecha()
		igual(t, true, Responde(socket))
	})
}

func TestAvisoDaSubidaEsperaAPrimeiraJanelaQueFala(t *testing.T) {
	q := NovoQuadro()
	q.Estado("now", msg{"n": 1})
	q.Aviso("Biblioteca trazida da versão anterior")
	q.SoltaGuardados() // ninguém conectado: o aviso continua guardado
	// Quem só bate no socket (o lançador) não leva o aviso embora.
	curioso := q.Assina()
	igual(t, []string{`{"n":1,"type":"now"}`}, curioso.Tira())
	q.Desliga(curioso)
	janela := q.Assina()
	janela.Tira()
	q.SoltaGuardados()
	igual(t, []string{`{"text":"Biblioteca trazida da versão anterior","type":"notice"}`}, janela.Tira())
	q.SoltaGuardados()
	igual(t, 0, len(janela.Tira()))
}

func TestComandoDeQuemJaFoiEmboraAindaChega(t *testing.T) {
	// É o que o "mktunesd --encerra" faz: escreve o comando e fecha, sem ler nada.
	l := lacoRodando(t)
	q := NovoQuadro()
	l.Posta(func() {
		for i := range 200 {
			q.Estado("area"+strconv.Itoa(i), msg{"enchimento": strings.Repeat("x", 4096)})
		}
	})
	recebidos := make(chan string, 1)
	socket := filepath.Join(pastaCurta(t), "mktunes.sock")
	s, erro := Serve(socket, q, l, func(linha string) { recebidos <- linha })
	if erro != nil {
		t.Fatal(erro)
	}
	defer s.Fecha()
	for range 20 {
		conexao, erro := net.Dial("unix", socket)
		if erro != nil {
			t.Fatal(erro)
		}
		conexao.Write([]byte(`{"cmd":"quit"}` + "\n"))
		conexao.Close()
		select {
		case linha := <-recebidos:
			igual(t, `{"cmd":"quit"}`, linha)
		case <-time.After(5 * time.Second):
			t.Fatal("o comando se perdeu")
		}
	}
}
