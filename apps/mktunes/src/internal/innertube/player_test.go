// SPDX-License-Identifier: GPL-3.0-or-later

package innertube

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"mktunes/internal/app"
	"mktunes/internal/dominio"
)

var _ app.Resolvedor = (*Player)(nil)

var agora = time.Unix(1795348800, 0)

const video = "n8X9_MgEdCg"

func faixa(id string) dominio.Faixa { return dominio.Faixa{ID: id, Provedor: dominio.YouTube} }

func novoPlayer(s *servidor) *Player {
	visitante := &Visitante{}
	visitante.guarda("convidado")
	return &Player{Cliente: s.cliente(VisionOS, visitante), Agora: func() time.Time { return agora }}
}

// resposta monta um "player" com um formato de áudio.
func resposta(endereco string) string {
	return `{"playabilityStatus":{"status":"OK"},"streamingData":{"expiresInSeconds":"21540","adaptiveFormats":[{"itag":140,"mimeType":"audio/mp4; codecs=\"mp4a.40.2\"","bitrate":131206,"url":"` + endereco + `"}]}}`
}

func TestPlayerResolvePeloVisionOS(t *testing.T) {
	s := novoServidor(t)
	s.gravada(t, "player-visionos.json")
	db := 6.68
	p := novoPlayer(s)
	p.Volume = func(_ context.Context, id string) *float64 {
		igual(t, video, id)
		return &db
	}
	stream, erro := p.Resolve(context.Background(), faixa(video))
	if erro != nil {
		t.Fatal(erro)
	}
	// O M4A mais rico que vem com endereço direto: nem o cifrado (141), nem o
	// comprimido (DRC), nem a faixa dublada.
	igual(t, "https://rr1---sn-teste.googlevideo.com/videoplayback?itag=140&expire=1795370340", stream.URL)
	igual(t, "140", stream.Variante)
	igual(t, 6.68, *stream.LoudnessDb)

	// O pedido é o do app do visionOS, com o visitante de convidado.
	pedido := s.ultimo()
	igual(t, "/youtubei/v1/player?prettyPrint=false", pedido.caminho)
	igual(t, []string{"101", "0.1", "convidado", "com.apple.visionos.youtube/1.0 (Vision Pro; visionOS 2.0; pt_BR)"},
		[]string{pedido.cabecalho.Get("X-YouTube-Client-Name"), pedido.cabecalho.Get("X-YouTube-Client-Version"), pedido.cabecalho.Get("X-Goog-Visitor-Id"), pedido.cabecalho.Get("User-Agent")})
	cliente := doCliente(pedido)
	igual(t, []any{"VISIONOS", "0.1", "convidado", "Apple", "RealityDevice14,1", "visionOS", "2.0", "pt", "BR"},
		[]any{cliente["clientName"], cliente["clientVersion"], cliente["visitorData"], cliente["deviceMake"], cliente["deviceModel"], cliente["osName"], cliente["osVersion"], cliente["hl"], cliente["gl"]})
	igual(t, []any{video, true, true}, []any{pedido.corpo["videoId"], pedido.corpo["contentCheckOk"], pedido.corpo["racyCheckOk"]})

	// O endereço fica guardado enquanto vale: a mesma faixa não pergunta de novo.
	de, _ := p.Resolve(context.Background(), faixa(video))
	igual(t, stream, de)
	igual(t, 1, len(s.pedidos))
	// A dois minutos de vencer (expiresInSeconds), pergunta de novo.
	agora = agora.Add(21540*time.Second - time.Minute)
	defer func() { agora = time.Unix(1795348800, 0) }()
	s.gravada(t, "player-visionos.json")
	p.Resolve(context.Background(), faixa(video))
	igual(t, 2, len(s.pedidos))
}

func TestEscolheOAudio(t *testing.T) {
	audio := func(mime string, bps int) formato {
		return formato{itag: bps / 1000, m4a: mime == "m4a", bps: bps, endereco: fmt.Sprintf("https://rr1.googlevideo.com/%d", bps/1000)}
	}
	escolhido := func(q dominio.Qualidade, formatos ...formato) string {
		f, tem := escolheAudio(formatos, q)
		if !tem {
			return ""
		}
		return f.endereco
	}
	alta, economia := dominio.QualidadeAlta, dominio.QualidadeEconomia
	// Os casos da fonte em Kotlin (pickAudio).
	igual(t, "https://rr1.googlevideo.com/128", escolhido(alta, audio("m4a", 48000), audio("opus", 160000), audio("m4a", 128000)))
	igual(t, "https://rr1.googlevideo.com/160", escolhido(alta, audio("opus", 70000), audio("opus", 160000)))
	igual(t, "", escolhido(alta))
	igual(t, "https://rr1.googlevideo.com/128", escolhido(economia, audio("m4a", 48000), audio("m4a", 128000), audio("opus", 160000)))
	igual(t, "https://rr1.googlevideo.com/150", escolhido(economia, audio("opus", 160000), audio("opus", 150000)))
	// O "128 kbps" do YouTube mede uns 130: ainda é economia.
	igual(t, "https://rr1.googlevideo.com/129", escolhido(economia, audio("m4a", 48826), audio("m4a", 129530), audio("m4a", 255000)))

	// Da resposta gravada: a economia fica no 140, e só as faixas originais contam.
	no := gravada(t, "player-visionos.json")
	f, _ := escolheAudio(leFormatos(no), economia)
	igual(t, 140, f.itag)
	igual(t, 4, len(leFormatos(no)))
	// Só com faixas comprimidas ou dubladas, elas servem.
	so, _ := LeJSON(strings.NewReader(`{"streamingData":{"adaptiveFormats":[{"itag":140,"mimeType":"audio/mp4","bitrate":130000,"isDrc":true,"url":"https://a.googlevideo.com/drc"}]}}`))
	igual(t, 1, len(leFormatos(so)))
}

func TestPlayerRecusas(t *testing.T) {
	ctx := context.Background()
	falha := func(p *Player, f dominio.Faixa, trecho string) {
		t.Helper()
		_, erro := p.Resolve(ctx, f)
		if !errors.Is(erro, app.ErrStream) || !strings.Contains(erro.Error(), trecho) {
			t.Fatalf("erro: %v (esperava %q)", erro, trecho)
		}
	}
	s := novoServidor(t)
	p := novoPlayer(s)
	// Um id que não é de vídeo nem chega ao YouTube; outro provedor também não.
	for _, id := range []string{"", "curto", "--exec=rm -rf", "n8X9_MgEdCg;x", "../../etc/pw", "n8X9_MgEdCg\n"} {
		falha(p, faixa(id), "id de vídeo inválido")
	}
	falha(p, dominio.Faixa{ID: video, Provedor: dominio.Deezer}, "não é uma faixa do YouTube")
	igual(t, 0, len(s.pedidos))

	// O YouTube diz por que não toca.
	s.texto(`{"playabilityStatus":{"status":"UNPLAYABLE","reason":"Vídeo indisponível"}}`)
	falha(p, faixa(video), "UNPLAYABLE: Vídeo indisponível")
	s.respostas = append(s.respostas, func(w http.ResponseWriter) { w.WriteHeader(500) })
	falha(p, faixa(video), "HTTP 500")
	// Sem áudio com endereço direto (só cifrado), não há o que tocar.
	s.texto(`{"playabilityStatus":{"status":"OK"},"streamingData":{"adaptiveFormats":[{"itag":140,"mimeType":"audio/mp4","signatureCipher":"s=A&url=https%3A%2F%2Fx.googlevideo.com"}]}}`)
	falha(p, faixa(video), "sem áudio direto")
	// Só endereços https do googlevideo.com são aceitos.
	for _, ruim := range []string{"http://x.googlevideo.com/v", "https://evil.example/googlevideo.com", "https://googlevideo.com.evil.example/v", "file:///etc/passwd", "https://user@x.googlevideo.com/v", "https://www.youtube.com/x"} {
		s.texto(resposta(ruim))
		falha(p, faixa(video), "sem áudio direto")
		igual(t, false, EnderecoAceito(ruim))
	}
	igual(t, true, EnderecoAceito("https://rr1---sn-x.googlevideo.com/videoplayback?a=b"))
	// Uma falha não fica guardada.
	s.texto(resposta("https://x.googlevideo.com/v"))
	if _, erro := p.Resolve(ctx, faixa(video)); erro != nil {
		t.Fatal(erro)
	}

	// Quem demora demais é cortado.
	solta := make(chan struct{})
	defer close(solta)
	s.respostas = append(s.respostas, func(w http.ResponseWriter) { <-solta })
	lento := novoPlayer(s)
	lento.Prazo = 100 * time.Millisecond
	falha(lento, faixa(video), "não respondeu")
}

func TestPlayerPedeVisitante(t *testing.T) {
	ctx := context.Background()
	t.Run("sem visitante, consegue um antes de pedir o player", func(t *testing.T) {
		s := novoServidor(t)
		visitante := &Visitante{}
		visitas := 0
		p := &Player{Cliente: s.cliente(VisionOS, visitante), Visita: func(context.Context) error {
			visitas++
			visitante.guarda("novo")
			return nil
		}}
		// Quem mede o volume também só é perguntado depois de haver visitante.
		comQuem := ""
		p.Volume = func(context.Context, string) *float64 {
			comQuem = visitante.le()
			return nil
		}
		s.texto(resposta("https://x.googlevideo.com/v"))
		if _, erro := p.Resolve(ctx, faixa(video)); erro != nil {
			t.Fatal(erro)
		}
		igual(t, 1, visitas)
		igual(t, "novo", s.ultimo().cabecalho.Get("X-Goog-Visitor-Id"))
		igual(t, "novo", comQuem)
	})
	t.Run("um visitante recusado vale uma nova tentativa com outro, e só uma", func(t *testing.T) {
		s := novoServidor(t)
		visitante := &Visitante{}
		visitante.guarda("vencido")
		visitas := 0
		p := &Player{Cliente: s.cliente(VisionOS, visitante), Visita: func(context.Context) error {
			visitas++
			igual(t, "", visitante.le())
			visitante.guarda("novo")
			return nil
		}}
		s.texto(`{"playabilityStatus":{"status":"LOGIN_REQUIRED","reason":"Faça login para confirmar que você não é um bot"}}`)
		s.texto(resposta("https://x.googlevideo.com/v"))
		stream, erro := p.Resolve(ctx, faixa(video))
		if erro != nil {
			t.Fatal(erro)
		}
		igual(t, "https://x.googlevideo.com/v", stream.URL)
		igual(t, 1, visitas)
		igual(t, []string{"vencido", "novo"}, []string{s.pedidos[0].cabecalho.Get("X-Goog-Visitor-Id"), s.pedidos[1].cabecalho.Get("X-Goog-Visitor-Id")})

		s.texto(`{"playabilityStatus":{"status":"LOGIN_REQUIRED","reason":"não é um bot"}}`)
		s.texto(`{"playabilityStatus":{"status":"LOGIN_REQUIRED","reason":"não é um bot"}}`)
		_, erro = p.Resolve(ctx, faixa("outra_faixa"))
		igual(t, true, errors.Is(erro, app.ErrStream) && strings.Contains(erro.Error(), "LOGIN_REQUIRED"))
		igual(t, 4, len(s.pedidos))
	})
}

func TestPlayerQualidadeECache(t *testing.T) {
	ctx := context.Background()
	s := novoServidor(t)
	p := novoPlayer(s)
	p.DefineQualidade(dominio.QualidadeEconomia)
	s.gravada(t, "player-visionos.json")
	economia, erro := p.Resolve(ctx, faixa(video))
	if erro != nil {
		t.Fatal(erro)
	}
	igual(t, "140", economia.Variante)
	// Sem quem meça o volume, vale o que o próprio formato informa.
	igual(t, -1.79, *economia.LoudnessDb)
	// A outra qualidade pode ser outro endereço: não reaproveita o guardado.
	p.DefineQualidade(dominio.QualidadeAlta)
	s.gravada(t, "player-visionos.json")
	p.Resolve(ctx, faixa(video))
	igual(t, 2, len(s.pedidos))

	// O cache tem teto.
	for i := range guardados + 8 {
		s.texto(resposta("https://x.googlevideo.com/v"))
		if _, erro := p.Resolve(ctx, faixa(fmt.Sprintf("video%06d", i))); erro != nil {
			t.Fatal(erro)
		}
	}
	igual(t, guardados, len(p.cache))
	// Sem validade na resposta, nada é guardado.
	antes := len(s.pedidos)
	semValidade := `{"playabilityStatus":{"status":"OK"},"streamingData":{"adaptiveFormats":[{"itag":140,"mimeType":"audio/mp4","bitrate":1,"url":"https://x.googlevideo.com/v"}]}}`
	s.texto(semValidade)
	s.texto(semValidade)
	p.Resolve(ctx, faixa("sem_validad"))
	p.Resolve(ctx, faixa("sem_validad"))
	igual(t, antes+2, len(s.pedidos))
}

func TestSoUmPedidoPorFaixaDeCadaVez(t *testing.T) {
	s := novoServidor(t)
	s.respostas = append(s.respostas, func(w http.ResponseWriter) {
		time.Sleep(300 * time.Millisecond)
		w.Write([]byte(resposta("https://x.googlevideo.com/v")))
	})
	p := novoPlayer(s)
	var espera sync.WaitGroup
	for range 4 {
		espera.Add(1)
		go func() {
			defer espera.Done()
			if _, erro := p.Resolve(context.Background(), faixa(video)); erro != nil {
				t.Error(erro)
			}
		}()
	}
	espera.Wait()
	igual(t, 1, len(s.pedidos))

	// Quem desiste não espera, mas o endereço fica guardado para quando a faixa voltar.
	s.respostas = append(s.respostas, func(w http.ResponseWriter) {
		time.Sleep(300 * time.Millisecond)
		w.Write([]byte(resposta("https://x.googlevideo.com/outra")))
	})
	ctx, cancela := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancela()
	_, erro := p.Resolve(ctx, faixa("outra_faixa"))
	igual(t, true, errors.Is(erro, app.ErrStream))
	stream, erro := p.Resolve(context.Background(), faixa("outra_faixa"))
	igual(t, nil, erro)
	igual(t, "https://x.googlevideo.com/outra", stream.URL)
	igual(t, 2, len(s.pedidos))
}

func TestVisitaConsegueOVisitante(t *testing.T) {
	s := novoServidor(t)
	visitante := &Visitante{}
	s.texto(`{"responseContext":{"visitorData":"recém-chegado"}}`)
	if erro := (Feed{Cliente: s.cliente(WebRemix, visitante)}).Visita(context.Background()); erro != nil {
		t.Fatal(erro)
	}
	igual(t, "recém-chegado", visitante.le())
	igual(t, "FEmusic_home", s.ultimo().corpo["browseId"])
}
