// SPDX-License-Identifier: GPL-3.0-or-later

package rede

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"mktunes/internal/app"
	"mktunes/internal/dominio"
)

var (
	_ app.FonteDeLetras    = Lrclib{}
	_ app.DiretorioDeRadio = RadioBrowser{}
	_ app.Buscador         = Arquivos{}
)

func igual(t *testing.T, esperado, obtido any) {
	t.Helper()
	if !reflect.DeepEqual(esperado, obtido) {
		t.Fatalf("esperado %#v\n  obtido %#v", esperado, obtido)
	}
}

// fila é um servidor que responde o que foi enfileirado, na ordem, e guarda os pedidos.
type fila struct {
	*httptest.Server
	mu        sync.Mutex
	respostas []http.HandlerFunc
	pedidos   []*http.Request
}

func novaFila(t *testing.T) *fila {
	f := &fila{}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.pedidos = append(f.pedidos, r)
		var responde http.HandlerFunc
		if len(f.respostas) > 0 {
			responde, f.respostas = f.respostas[0], f.respostas[1:]
		}
		f.mu.Unlock()
		if responde == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		responde(w, r)
	}))
	t.Cleanup(f.Close)
	return f
}

func (f *fila) responde(codigo int, corpo string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.respostas = append(f.respostas, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(codigo)
		w.Write([]byte(corpo))
	})
}

func (f *fila) pedido(i int) *http.Request {
	f.mu.Lock()
	defer f.mu.Unlock()
	if i < 0 {
		i += len(f.pedidos)
	}
	return f.pedidos[i]
}

func TestLetrasDoLrclib(t *testing.T) {
	ctx := context.Background()
	musica := dominio.Faixa{ID: "1", Titulo: "Canção", Artista: "Banda", Duracao: 200 * time.Second}
	t.Run("acha a letra sincronizada por título, artista e duração", func(t *testing.T) {
		s := novaFila(t)
		s.responde(200, `{"duration":200,"syncedLyrics":"[00:01.00] linha um\n[00:05.00] linha dois","plainLyrics":"linha um\nlinha dois","extra":1}`)
		letra, erro := Lrclib{HTTP: s.Client(), Base: s.URL}.Acha(ctx, musica)
		igual(t, nil, erro)
		igual(t, dominio.Letra{{Tempo: time.Second, Texto: "linha um"}, {Tempo: 5 * time.Second, Texto: "linha dois"}}, letra.Sincronizada)
		p := s.pedido(0)
		igual(t, "/api/get", p.URL.Path)
		igual(t, []string{"Canção", "Banda", "200"}, []string{p.URL.Query().Get("track_name"), p.URL.Query().Get("artist_name"), p.URL.Query().Get("duration")})
		igual(t, true, strings.HasPrefix(p.Header.Get("User-Agent"), "MkTunes"))
	})
	t.Run("fica com a letra simples quando nada tem tempo", func(t *testing.T) {
		s := novaFila(t)
		s.responde(200, `{"duration":200,"syncedLyrics":null,"plainLyrics":"só texto"}`)
		letra, _ := Lrclib{HTTP: s.Client(), Base: s.URL}.Acha(ctx, musica)
		igual(t, &dominio.LetraDaFaixa{Texto: "só texto"}, letra)
	})
	t.Run("sem resultado exato, procura e aceita a sincronizada a até cinco segundos", func(t *testing.T) {
		s := novaFila(t)
		s.responde(404, `{"code":404}`)
		s.responde(200, `[{"duration":260,"syncedLyrics":"[00:01.00] longe demais"},{"duration":203,"plainLyrics":"simples"},{"duration":197,"syncedLyrics":"[00:02.00] perto o bastante"}]`)
		letra, _ := Lrclib{HTTP: s.Client(), Base: s.URL + "/"}.Acha(ctx, musica)
		igual(t, "perto o bastante", letra.Sincronizada[0].Texto)
		igual(t, "/api/search", s.pedido(-1).URL.Path)
		igual(t, "", s.pedido(-1).URL.Query().Get("duration"))
	})
	t.Run("não acha nada quando nenhuma das duas tem letra", func(t *testing.T) {
		s := novaFila(t)
		s.responde(404, "")
		s.responde(200, "[]")
		letra, erro := Lrclib{HTTP: s.Client(), Base: s.URL}.Acha(ctx, musica)
		igual(t, true, letra == nil && erro == nil)
	})
	t.Run("um erro do servidor ou uma resposta que não se lê são o serviço fora do ar", func(t *testing.T) {
		s := novaFila(t)
		s.responde(500, "")
		_, erro := Lrclib{HTTP: s.Client(), Base: s.URL}.Acha(ctx, musica)
		igual(t, true, errors.Is(erro, app.ErrCatalogo))
		s.responde(200, "<html>")
		_, erro = Lrclib{HTTP: s.Client(), Base: s.URL}.Acha(ctx, musica)
		igual(t, true, errors.Is(erro, app.ErrCatalogo))
	})
	t.Run("pular a faixa cancela na hora uma busca lenta", func(t *testing.T) {
		s := novaFila(t)
		solta := make(chan struct{})
		defer close(solta)
		s.respostas = append(s.respostas, func(http.ResponseWriter, *http.Request) { <-solta })
		ctx, cancela := context.WithTimeout(ctx, 200*time.Millisecond)
		defer cancela()
		inicio := time.Now()
		_, erro := Lrclib{HTTP: s.Client(), Base: s.URL}.Acha(ctx, musica)
		igual(t, true, erro != nil && time.Since(inicio) < 3*time.Second)
	})
}

// As estações aqui são inventadas.
const estacoes = `[
 {"stationuuid":"aaa-1","name":"  Rádio Cobre  ","url_resolved":"https://stream.example/cobre.mp3","favicon":"https://example/cobre.png","tags":"mpb,samba,bossa nova,jazz","country":"Brazil","countrycode":"BR"},
 {"stationuuid":"bbb-2","name":"Plain HTTP","url_resolved":"http://stream.example/plain.mp3","favicon":"","tags":"","country":""},
 {"stationuuid":"ccc-3","name":"Sem stream","url_resolved":"","favicon":"http://example/x.png","tags":"rock","country":"Portugal"},
 {"stationuuid":"ddd-4","name":"Longe","url_resolved":"https://stream.example/longe","favicon":"http://example/inseguro.png","tags":"","country":"Wakanda","countrycode":"WK"}]`

func TestRadioBrowser(t *testing.T) {
	ctx := context.Background()
	t.Run("lista as mais votadas como faixas ao vivo, só com stream HTTPS", func(t *testing.T) {
		s := novaFila(t)
		s.responde(200, estacoes)
		achadas, erro := RadioBrowser{HTTP: s.Client(), Servidores: []string{s.URL + "/"}}.Topo(ctx, 20)
		igual(t, nil, erro)
		p := s.pedido(0)
		igual(t, "/json/stations/topvote", p.URL.Path)
		igual(t, []string{"20", "true"}, []string{p.URL.Query().Get("limit"), p.URL.Query().Get("hidebroken")})
		igual(t, []dominio.Faixa{
			{ID: "radio:aaa-1", Provedor: dominio.Radio, Titulo: "Rádio Cobre", Artista: "mpb · samba · bossa nova · Brasil", Capa: "https://example/cobre.png", Stream: "https://stream.example/cobre.mp3"},
			{ID: "radio:ddd-4", Provedor: dominio.Radio, Titulo: "Longe", Artista: "Wakanda", Stream: "https://stream.example/longe"},
		}, achadas)
	})
	t.Run("começa pelas mais votadas do país do ouvinte, e cai no mundo quando o país não tem", func(t *testing.T) {
		s := novaFila(t)
		s.responde(200, estacoes)
		achadas, _ := RadioBrowser{HTTP: s.Client(), Servidores: []string{s.URL}, Pais: "BR"}.Topo(ctx, 20)
		q := s.pedido(0).URL.Query()
		igual(t, "/json/stations/search", s.pedido(0).URL.Path)
		igual(t, []string{"BR", "votes", "true"}, []string{q.Get("countrycode"), q.Get("order"), q.Get("reverse")})
		igual(t, 2, len(achadas))
		s.responde(200, "[]")
		s.responde(200, estacoes)
		achadas, _ = RadioBrowser{HTTP: s.Client(), Servidores: []string{s.URL}, Pais: "AQ"}.Topo(ctx, 20)
		igual(t, "/json/stations/topvote", s.pedido(-1).URL.Path)
		igual(t, 2, len(achadas))
	})
	t.Run("procura pelo nome, por votos", func(t *testing.T) {
		s := novaFila(t)
		s.responde(200, "[]")
		achadas, erro := RadioBrowser{HTTP: s.Client(), Servidores: []string{s.URL}}.Busca(ctx, "cobre", 10)
		q := s.pedido(0).URL.Query()
		igual(t, []string{"cobre", "votes", "true", "10"}, []string{q.Get("name"), q.Get("order"), q.Get("reverse"), q.Get("limit")})
		igual(t, true, erro == nil && len(achadas) == 0)
	})
	t.Run("passa ao próximo espelho quando um cai; todos fora é o diretório indisponível", func(t *testing.T) {
		um, dois := novaFila(t), novaFila(t)
		um.responde(503, "oops")
		dois.responde(200, estacoes)
		r := RadioBrowser{HTTP: um.Client(), Servidores: []string{um.URL, dois.URL}}
		achadas, _ := r.Topo(ctx, 20)
		igual(t, 2, len(achadas))
		um.responde(503, "oops")
		dois.responde(200, "<html>")
		_, erro := r.Topo(ctx, 20)
		igual(t, true, errors.Is(erro, app.ErrCatalogo))
	})
}

// semRede é um transporte que reprova o teste que tentar sair para a rede.
type semRede struct{}

func (semRede) RoundTrip(r *http.Request) (*http.Response, error) {
	return nil, errors.New("um teste tentou falar com " + r.URL.Host)
}

func audio(n int) []byte {
	dados := make([]byte, n)
	for i := range dados {
		dados[i] = byte(i * 7)
	}
	return dados
}

// comFaixas serve os dados respeitando o cabeçalho Range (como a CDN do YouTube).
func comFaixas(dados []byte, pedidos *int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		*pedidos++
		var de, ate int
		fmt.Sscanf(r.Header.Get("Range"), "bytes=%d-%d", &de, &ate)
		if de >= len(dados) {
			w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
			return
		}
		ate = min(ate, len(dados)-1)
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", de, ate, len(dados)))
		w.WriteHeader(http.StatusPartialContent)
		w.Write(dados[de : ate+1])
	}
}

func arquivosEm(raiz string) []string {
	var achados []string
	filepath.WalkDir(raiz, func(caminho string, e fs.DirEntry, _ error) error {
		if e != nil && e.Type().IsRegular() {
			achados = append(achados, caminho)
		}
		return nil
	})
	return achados
}

func TestArquivos(t *testing.T) {
	ctx := context.Background()
	nada := func(float64) {}
	t.Run("pede em faixas de bytes, salva na pasta de música e informa o progresso", func(t *testing.T) {
		dados, pedidos := audio(3500), 0
		s := httptest.NewServer(comFaixas(dados, &pedidos))
		defer s.Close()
		raiz := t.TempDir()
		a := Arquivos{HTTP: s.Client(), Raiz: raiz, Bloco: 1000}
		var visto []float64
		caminho, erro := a.Busca(ctx, s.URL+"/a", "LilRioG/Ignored [x].m4a", func(p float64) { visto = append(visto, p) })
		igual(t, nil, erro)
		igual(t, filepath.Join(raiz, "LilRioG/Ignored [x].m4a"), caminho)
		salvo, _ := os.ReadFile(caminho)
		igual(t, true, bytes.Equal(dados, salvo))
		igual(t, 4, pedidos)
		igual(t, true, a.Existe(caminho))
		igual(t, true, len(visto) > 1 && visto[len(visto)-1] == 1 && sort.Float64sAreSorted(visto))
		igual(t, []string{caminho}, arquivosEm(raiz))
		a.Apaga(caminho)
		igual(t, false, a.Existe(caminho))
	})
	t.Run("um erro HTTP falha sem deixar nada para trás", func(t *testing.T) {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(403) }))
		defer s.Close()
		raiz := t.TempDir()
		_, erro := Arquivos{HTTP: s.Client(), Raiz: raiz}.Busca(ctx, s.URL+"/a", "a.m4a", nada)
		var falha *app.ErroDownload
		igual(t, true, errors.As(erro, &falha))
		igual(t, "HTTP 403", falha.Motivo)
		igual(t, 0, len(arquivosEm(raiz)))
	})
	t.Run("um download cancelado não deixa arquivo pela metade", func(t *testing.T) {
		solta := make(chan struct{})
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Length", "100000")
			w.Write(audio(1000))
			w.(http.Flusher).Flush()
			<-solta
		}))
		defer s.Close()
		defer close(solta)
		raiz := t.TempDir()
		ctx, cancela := context.WithTimeout(ctx, 300*time.Millisecond)
		defer cancela()
		_, erro := Arquivos{HTTP: s.Client(), Raiz: raiz}.Busca(ctx, s.URL+"/a", "a.m4a", nada)
		igual(t, true, errors.Is(erro, context.DeadlineExceeded))
		igual(t, 0, len(arquivosEm(raiz)))
	})
	t.Run("um servidor sem faixas manda o arquivo inteiro num pedido só", func(t *testing.T) {
		dados := audio(5000)
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.Write(dados) }))
		defer s.Close()
		caminho, erro := Arquivos{HTTP: s.Client(), Raiz: t.TempDir(), Bloco: 1000}.Busca(ctx, s.URL, "a.m4a", nada)
		igual(t, nil, erro)
		salvo, _ := os.ReadFile(caminho)
		igual(t, true, bytes.Equal(dados, salvo))
	})
	t.Run("um servidor que para de respeitar as faixas no meio ainda dá o arquivo exato, e o progresso recomeça", func(t *testing.T) {
		dados, pedidos := audio(300000), 0
		faixas := comFaixas(dados, &pedidos)
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if pedidos >= 2 {
				w.Header().Set("Content-Length", strconv.Itoa(len(dados)))
				w.Write(dados)
				return
			}
			faixas(w, r)
		}))
		defer s.Close()
		var visto []float64
		caminho, erro := Arquivos{HTTP: s.Client(), Raiz: t.TempDir(), Bloco: 100000}.Busca(ctx, s.URL, "a.m4a", func(p float64) { visto = append(visto, p) })
		igual(t, nil, erro)
		salvo, _ := os.ReadFile(caminho)
		igual(t, true, bytes.Equal(dados, salvo))
		igual(t, false, sort.Float64sAreSorted(visto))
	})
	t.Run("faixas vazias falham em vez de rodar para sempre", func(t *testing.T) {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusPartialContent) }))
		defer s.Close()
		_, erro := Arquivos{HTTP: s.Client(), Raiz: t.TempDir(), Bloco: 1000}.Busca(ctx, s.URL, "a.m4a", nada)
		var falha *app.ErroDownload
		igual(t, true, errors.As(erro, &falha))
	})
	t.Run("um tamanho desconhecido que é múltiplo exato do bloco termina limpo", func(t *testing.T) {
		dados := audio(2000)
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var de, ate int
			fmt.Sscanf(r.Header.Get("Range"), "bytes=%d-%d", &de, &ate)
			if de >= len(dados) {
				w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
				return
			}
			w.WriteHeader(http.StatusPartialContent)
			w.Write(dados[de:min(ate+1, len(dados))])
		}))
		defer s.Close()
		caminho, erro := Arquivos{HTTP: s.Client(), Raiz: t.TempDir(), Bloco: 1000}.Busca(ctx, s.URL, "a.m4a", nada)
		igual(t, nil, erro)
		salvo, _ := os.ReadFile(caminho)
		igual(t, true, bytes.Equal(dados, salvo))
	})
	t.Run("a rede caindo é uma conexão interrompida", func(t *testing.T) {
		s := httptest.NewServer(nil)
		cliente, endereco := s.Client(), s.URL
		s.Close()
		_, erro := Arquivos{HTTP: cliente, Raiz: t.TempDir()}.Busca(ctx, endereco, "a.m4a", nada)
		var falha *app.ErroDownload
		igual(t, true, errors.As(erro, &falha))
		igual(t, "Conexão interrompida", falha.Motivo)
	})
	t.Run("nada é escrito nem apagado fora da pasta de música", func(t *testing.T) {
		raiz := t.TempDir()
		fora := filepath.Join(t.TempDir(), "fora.txt")
		os.WriteFile(fora, []byte("x"), 0o600)
		a := Arquivos{HTTP: &http.Client{Transport: semRede{}}, Raiz: raiz}
		for _, relativo := range []string{"../fora.m4a", "a/../../fora.m4a", "", "/etc/x"[0:0]} {
			if _, erro := a.Busca(ctx, "https://x.invalid/a", relativo, nada); erro == nil {
				t.Fatalf("devia recusar %q", relativo)
			}
		}
		_, erro := a.Busca(ctx, "file:///etc/passwd", "a.m4a", nada)
		igual(t, true, erro != nil)
		a.Apaga(fora)
		a.Apaga(raiz + "/../" + filepath.Base(filepath.Dir(fora)) + "/fora.txt")
		if _, erro := os.Stat(fora); erro != nil {
			t.Fatal("apagou um arquivo de fora da pasta de música")
		}
		Arquivos{Raiz: ""}.Apaga(fora)
		Arquivos{Raiz: "/"}.Apaga(fora)
		if _, erro := os.Stat(fora); erro != nil {
			t.Fatal("uma raiz vazia ou / não pode apagar nada")
		}
	})
	t.Run("a limpeza da subida tira os pedaços velhos e deixa o que ainda está sendo escrito", func(t *testing.T) {
		raiz := t.TempDir()
		espera := filepath.Join(raiz, pastaDeEspera)
		os.MkdirAll(espera, 0o755)
		velho, novo := filepath.Join(espera, "velho.part"), filepath.Join(espera, "novo.part")
		os.WriteFile(velho, []byte("x"), 0o600)
		os.WriteFile(novo, []byte("x"), 0o600)
		os.Chtimes(velho, time.Now().Add(-time.Hour), time.Now().Add(-time.Hour))
		Arquivos{Raiz: raiz}.LimpaIncompletos()
		igual(t, []string{novo}, arquivosEm(raiz))
	})
}
