// SPDX-License-Identifier: GPL-3.0-or-later

package app_test

import (
	"context"
	"errors"
	"math"
	"math/rand/v2"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"

	"mktunes/internal/app"
	"mktunes/internal/dominio"
	"mktunes/internal/laco"
	"mktunes/internal/prova"
)

func igual(t *testing.T, esperado, obtido any) {
	t.Helper()
	if !reflect.DeepEqual(esperado, obtido) {
		t.Fatalf("esperado %#v\n  obtido %#v", esperado, obtido)
	}
}

// radio é o rádio de mentira do autoplay.
type radio struct {
	mu       sync.Mutex
	sementes []string
	resposta func(ctx context.Context, semente dominio.Faixa) ([]dominio.Faixa, error)
	playlist func() ([]dominio.Faixa, error)
}

func (r *radio) Radio(ctx context.Context, semente dominio.Faixa) ([]dominio.Faixa, error) {
	r.mu.Lock()
	r.sementes = append(r.sementes, semente.ID)
	resposta := r.resposta
	r.mu.Unlock()
	if resposta == nil {
		var faixas []dominio.Faixa
		for _, n := range []string{"1", "2", "3"} {
			faixas = append(faixas, prova.Faixa(semente.ID+"-r"+n))
		}
		return faixas, nil
	}
	return resposta(ctx, semente)
}

func (r *radio) Playlist(context.Context, string) ([]dominio.Faixa, error) { return r.playlist() }

func (r *radio) pedidas() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string{}, r.sementes...)
}

// bancada é uma sessão com tudo de mentira em volta.
type bancada struct {
	l *laco.Teste
	r *prova.Resolvedor
	p *prova.Tocador
	s *app.Sessao
}

func novaBancada(ajusta func(o *app.OpcoesDaSessao)) *bancada {
	b := &bancada{l: laco.NovoTeste(), r: &prova.Resolvedor{Pendentes: map[string]*laco.Portao{}, Indisponiveis: map[string]bool{}}, p: &prova.Tocador{}}
	o := app.OpcoesDaSessao{Resolvedor: b.r, Tocador: b.p, Laco: b.l}
	if ajusta != nil {
		ajusta(&o)
	}
	b.s = app.NovaSessao(o)
	b.p.AoEvento = b.s.AoEvento
	return b
}

func (b *bancada) atual() string {
	f, _ := b.s.Estado().Faixa()
	return f.ID
}

func (b *bancada) naFila() []string {
	ids := []string{}
	if f, tem := b.s.Estado().Faixa(); tem {
		ids = append(ids, f.ID)
	}
	return append(ids, prova.IDs(b.s.Estado().Fila.ASeguir())...)
}

func (b *bancada) toca(indice int, ids ...string) {
	var faixas []dominio.Faixa
	for _, id := range ids {
		faixas = append(faixas, prova.Faixa(id, prova.SemStream))
	}
	b.s.TocaDe(faixas, indice)
	b.l.Roda()
}

func (b *bancada) pendente(id string) *laco.Portao {
	portao := b.l.Portao()
	b.r.Muda(func(r *prova.Resolvedor) { r.Pendentes[id] = portao })
	return portao
}

func (b *bancada) evento(e app.Evento) {
	b.s.AoEvento(e)
	b.l.Roda()
}

func TestSessaoToca(t *testing.T) {
	t.Run("começa ociosa", func(t *testing.T) {
		b := novaBancada(nil)
		igual(t, "", b.atual())
		igual(t, app.Ocioso, b.s.Estado().Situacao)
	})
	t.Run("escolher uma faixa resolve o stream e toca", func(t *testing.T) {
		b := novaBancada(nil)
		b.toca(1, "1", "2")
		igual(t, "2", b.atual())
		igual(t, app.Tocando, b.s.Estado().Situacao)
		igual(t, "play https://stream.example/2", b.p.Chamadas[len(b.p.Chamadas)-1])
	})
	t.Run("mostra carregando enquanto o stream não vem, com o progresso em zero", func(t *testing.T) {
		b := novaBancada(nil)
		b.toca(0, "1", "2")
		b.p.Pos = 200 * time.Second
		b.pendente("2")
		b.s.Proxima()
		b.l.Roda()
		igual(t, app.Carregando, b.s.Estado().Situacao)
		igual(t, "2", b.atual())
		igual(t, time.Duration(0), b.s.Progresso().Posicao)
	})
	t.Run("uma escolha mais nova ganha de um stream que ainda resolve", func(t *testing.T) {
		b := novaBancada(nil)
		lento := b.pendente("1")
		b.toca(0, "1", "2")
		b.toca(1, "1", "2")
		lento.Abre()
		b.l.Roda()
		igual(t, "2", b.atual())
		igual(t, []string{"play https://stream.example/2"}, b.p.Com("play"))
	})
	t.Run("avisa da faixa sem stream, e tocar de novo tenta outra vez", func(t *testing.T) {
		b := novaBancada(nil)
		b.r.Muda(func(r *prova.Resolvedor) { r.Indisponiveis["1"] = true })
		b.toca(0, "1")
		igual(t, app.FalhouStream, b.s.Estado().Situacao)
		b.r.Muda(func(r *prova.Resolvedor) { r.Indisponiveis = map[string]bool{} })
		b.s.AlternaPausa()
		b.l.Roda()
		igual(t, app.Tocando, b.s.Estado().Situacao)
	})
	t.Run("um erro inesperado ao achar o stream marca a faixa como indisponível", func(t *testing.T) {
		b := novaBancada(nil)
		b.r.Muda(func(r *prova.Resolvedor) { r.Falha = errors.New("o extrator mudou") })
		b.toca(0, "1")
		igual(t, app.FalhouStream, b.s.Estado().Situacao)
	})
	t.Run("pausar e continuar", func(t *testing.T) {
		b := novaBancada(nil)
		b.toca(0, "1")
		b.s.AlternaPausa()
		igual(t, app.Pausado, b.s.Estado().Situacao)
		b.s.AlternaPausa()
		igual(t, app.Tocando, b.s.Estado().Situacao)
		igual(t, []string{"pause", "resume"}, b.p.Chamadas[len(b.p.Chamadas)-2:])
	})
	t.Run("quando a faixa acaba a próxima começa; a última deixa a sessão ociosa", func(t *testing.T) {
		b := novaBancada(nil)
		b.toca(0, "1", "2")
		b.evento(app.Evento{Tipo: app.Terminou})
		igual(t, "2", b.atual())
		igual(t, "play https://stream.example/2", b.p.Chamadas[len(b.p.Chamadas)-1])
		b.evento(app.Evento{Tipo: app.Terminou})
		igual(t, "", b.atual())
		igual(t, app.Ocioso, b.s.Estado().Situacao)
	})
	t.Run("repetir uma toca a mesma de novo", func(t *testing.T) {
		b := novaBancada(nil)
		b.toca(0, "1", "2")
		b.s.DefineRepetir(dominio.RepetirUma)
		b.evento(app.Evento{Tipo: app.Terminou})
		igual(t, "1", b.atual())
		igual(t, 2, len(b.p.Com("play https://stream.example/1")))
	})
	t.Run("pausa e retomada vindas de fora aparecem, mas não escondem o carregando", func(t *testing.T) {
		b := novaBancada(nil)
		b.toca(0, "1", "2")
		b.evento(app.Evento{Tipo: app.MudouPausa, Tocando: false})
		igual(t, app.Pausado, b.s.Estado().Situacao)
		b.evento(app.Evento{Tipo: app.MudouPausa, Tocando: true})
		igual(t, app.Tocando, b.s.Estado().Situacao)
		b.pendente("2")
		b.s.Proxima()
		b.evento(app.Evento{Tipo: app.MudouPausa, Tocando: true})
		igual(t, app.Carregando, b.s.Estado().Situacao)
	})
	t.Run("um erro do tocador aparece", func(t *testing.T) {
		b := novaBancada(nil)
		b.toca(0, "1")
		b.evento(app.Evento{Tipo: app.FalhouAoTocar, Motivo: "decoder"})
		igual(t, app.FalhouTocador, b.s.Estado().Situacao)
	})
	t.Run("trocar de faixa segura a antiga em vez de parar", func(t *testing.T) {
		b := novaBancada(nil)
		b.toca(0, "1", "2", "3")
		b.p.Chamadas = nil
		b.s.Proxima()
		b.l.Roda()
		igual(t, "2", b.atual())
		igual(t, 0, len(b.p.Com("stop")))
		igual(t, "hold 2", b.p.Chamadas[0])
	})
	t.Run("anterior enquanto a próxima carrega volta na hora", func(t *testing.T) {
		b := novaBancada(nil)
		b.toca(0, "1", "2")
		b.p.Pos = 90 * time.Second
		b.pendente("2")
		b.s.Proxima()
		b.l.Roda()
		b.s.Anterior()
		b.l.Roda()
		igual(t, "1", b.atual())
	})
	t.Run("um fim avisado enquanto a próxima carrega não a pula", func(t *testing.T) {
		b := novaBancada(nil)
		b.toca(0, "1", "2", "3")
		b.pendente("2")
		b.s.Proxima()
		b.l.Roda()
		b.evento(app.Evento{Tipo: app.Terminou})
		igual(t, "2", b.atual())
	})
	t.Run("próxima na última para de tocar", func(t *testing.T) {
		b := novaBancada(nil)
		b.toca(0, "1")
		b.s.Proxima()
		b.l.Roda()
		igual(t, app.Ocioso, b.s.Estado().Situacao)
		igual(t, "stop", b.p.Chamadas[len(b.p.Chamadas)-1])
	})
	t.Run("anterior no começo volta uma faixa; mais adiante recomeça a atual", func(t *testing.T) {
		b := novaBancada(nil)
		b.toca(1, "1", "2")
		b.p.Pos = 40 * time.Second
		b.s.Anterior()
		b.l.Roda()
		igual(t, "2", b.atual())
		igual(t, "seek 0s", b.p.Chamadas[len(b.p.Chamadas)-1])
		b.p.Pos = 2 * time.Second
		b.s.Anterior()
		b.l.Roda()
		igual(t, "1", b.atual())
	})
}

func TestSessaoControles(t *testing.T) {
	comecada := func() *bancada {
		acaso := rand.New(rand.NewPCG(3, 3))
		b := novaBancada(func(o *app.OpcoesDaSessao) { o.Mistura = acaso.Shuffle })
		b.p.Pos, b.p.Dur = 42*time.Second, 3*time.Minute
		b.toca(0, "1", "2", "3", "4")
		b.p.Pos = 42 * time.Second
		b.p.Chamadas = nil
		return b
	}
	seguir := func(b *bancada) []string { return prova.IDs(b.s.Estado().Fila.ASeguir()) }
	t.Run("pula para uma faixa da fila", func(t *testing.T) {
		b := comecada()
		b.s.TocaDaFila(1)
		b.l.Roda()
		igual(t, "3", b.atual())
		igual(t, []string{"hold 3", "play https://stream.example/3"}, b.p.Chamadas)
	})
	t.Run("tocar a seguir entra logo depois da atual, na ordem em que foram escolhidas", func(t *testing.T) {
		b := comecada()
		b.s.TocaASeguir(prova.Faixa("a"))
		b.s.TocaASeguir(prova.Faixa("b"))
		igual(t, []string{"a", "b", "2", "3", "4"}, seguir(b))
		igual(t, 0, len(b.p.Chamadas))
	})
	t.Run("tocar a seguir continua sendo a seguir depois de editar a fila", func(t *testing.T) {
		b := novaBancada(nil)
		b.toca(0, "cur", "x", "y")
		b.s.TocaASeguir(prova.Faixa("a"))
		b.s.RemoveDaFila(0)
		b.s.TocaASeguir(prova.Faixa("b"))
		igual(t, []string{"cur", "b", "x", "y"}, b.naFila())
	})
	t.Run("pôr na fila vai para o fim; numa sessão vazia começa a tocar", func(t *testing.T) {
		b := comecada()
		b.s.PoeNaFila(prova.Faixa("z"))
		igual(t, []string{"2", "3", "4", "z"}, seguir(b))
		vazia := novaBancada(nil)
		vazia.s.PoeNaFila(prova.Faixa("z", prova.SemStream))
		vazia.l.Roda()
		igual(t, "z", vazia.atual())
		igual(t, []string{"hold z", "play https://stream.example/z"}, vazia.p.Chamadas)
	})
	t.Run("edita a fila sem interromper a música, e ignora linhas que já sumiram", func(t *testing.T) {
		b := comecada()
		b.s.MoveNaFila(2, 0)
		b.s.RemoveDaFila(1)
		igual(t, []string{"4", "3"}, seguir(b))
		b.s.RemoveDaFila(5)
		b.s.MoveNaFila(0, 3)
		b.s.MoveNaFila(4, 0)
		igual(t, []string{"4", "3"}, seguir(b))
		b.s.LimpaFila()
		igual(t, 0, len(seguir(b)))
		igual(t, 0, len(b.p.Chamadas))
	})
	t.Run("embaralha o que vem a seguir, mantendo a atual", func(t *testing.T) {
		b := comecada()
		b.s.EmbaralhaFila()
		igual(t, "1", b.atual())
		ids := seguir(b)
		sort.Strings(ids)
		igual(t, []string{"2", "3", "4"}, ids)
		igual(t, 0, len(b.p.Chamadas))
	})
	t.Run("repetir e parar depois desta", func(t *testing.T) {
		b := comecada()
		b.s.CicloRepetir()
		igual(t, dominio.RepetirTudo, b.s.Estado().Fila.Repetir())
		b.s.AlternaPararAposAtual()
		igual(t, true, b.s.Estado().Fila.PararAposAtual())
	})
	t.Run("pula dentro da faixa e informa o progresso", func(t *testing.T) {
		b := comecada()
		igual(t, app.Progresso{Posicao: 42 * time.Second, Duracao: 3 * time.Minute}, b.s.Progresso())
		b.s.VaiPara(90 * time.Second)
		igual(t, []string{"seek 1m30s"}, b.p.Chamadas)
	})
	t.Run("segue o próxima e o anterior apertados fora do app", func(t *testing.T) {
		b := comecada()
		b.p.Pos = time.Second
		b.evento(app.Evento{Tipo: app.PediuProxima})
		igual(t, "2", b.atual())
		b.evento(app.Evento{Tipo: app.PediuAnterior})
		igual(t, "1", b.atual())
	})
}

func TestSom(t *testing.T) {
	alto := 6.68
	sessao := func(normaliza bool) *bancada {
		b := novaBancada(func(o *app.OpcoesDaSessao) { o.Normalizar = func() bool { return normaliza } })
		b.r.Muda(func(r *prova.Resolvedor) { r.LoudnessDb = &alto })
		return b
	}
	b := sessao(true)
	b.toca(0, "1")
	if v := b.p.Tocados[0].Volume; math.Abs(v-0.463) > 0.001 {
		t.Fatalf("volume normalizado: %v", v)
	}
	b = sessao(false)
	b.toca(0, "1")
	igual(t, 1.0, b.p.Tocados[0].Volume)
	b.s.DefineVelocidade(1.5)
	b.s.DefineVelocidade(9)
	igual(t, []string{"speed 1.5", "speed 2"}, b.p.Com("speed"))
	igual(t, 2.0, b.s.Velocidade())
}

func TestAutoplay(t *testing.T) {
	com := func(r *radio, ligado bool) *bancada {
		return novaBancada(func(o *app.OpcoesDaSessao) {
			o.Autoplay = r
			o.AutoplayLigado = func() bool { return ligado }
		})
	}
	t.Run("uma música sozinha é seguida pelo rádio dela", func(t *testing.T) {
		r := &radio{}
		b := com(r, true)
		b.toca(0, "seed")
		igual(t, []string{"seed"}, r.pedidas())
		igual(t, []string{"seed", "seed-r1", "seed-r2", "seed-r3"}, b.naFila())
	})
	t.Run("o rádio deixa de fora o que já está na fila, e põe no máximo vinte", func(t *testing.T) {
		r := &radio{resposta: func(context.Context, dominio.Faixa) ([]dominio.Faixa, error) {
			return []dominio.Faixa{prova.Faixa("seed"), prova.Faixa("x"), prova.Faixa("x")}, nil
		}}
		b := com(r, true)
		b.toca(0, "seed")
		igual(t, []string{"seed", "x"}, b.naFila())
		r = &radio{resposta: func(context.Context, dominio.Faixa) ([]dominio.Faixa, error) {
			var muitas []dominio.Faixa
			for i := range 50 {
				muitas = append(muitas, prova.Faixa(string(rune('A'+i))))
			}
			return muitas, nil
		}}
		b = com(r, true)
		b.toca(0, "seed")
		igual(t, 20, len(b.s.Estado().Fila.ASeguir()))
	})
	t.Run("espera enquanto sobram mais de duas, e semeia pelo fim da fila", func(t *testing.T) {
		r := &radio{}
		b := com(r, true)
		b.toca(0, "1", "2", "3", "4")
		igual(t, 0, len(r.pedidas()))
		b.s.Proxima()
		b.l.Roda()
		igual(t, []string{"4"}, r.pedidas())
	})
	t.Run("fica quieto repetindo, desligado ou depois de uma rádio", func(t *testing.T) {
		r := &radio{}
		b := com(r, true)
		b.s.DefineRepetir(dominio.RepetirTudo)
		b.toca(0, "seed")
		com(r, false).toca(0, "seed")
		estacao := com(r, true)
		estacao.s.TocaDe([]dominio.Faixa{prova.Faixa("station", prova.Provedor(dominio.Radio))}, 0)
		estacao.l.Roda()
		igual(t, 0, len(r.pedidas()))
	})
	t.Run("descarta a resposta atrasada quando o ouvinte já escolheu outra coisa", func(t *testing.T) {
		var b *bancada
		var atrasada *laco.Portao
		r := &radio{resposta: func(ctx context.Context, s dominio.Faixa) ([]dominio.Faixa, error) {
			if s.ID == "seed" {
				if erro := atrasada.Espera(ctx); erro != nil {
					return nil, erro
				}
				return []dominio.Faixa{prova.Faixa("stale")}, nil
			}
			return nil, nil
		}}
		b = novaBancada(func(o *app.OpcoesDaSessao) { o.Autoplay = r })
		atrasada = b.l.Portao()
		b.toca(0, "seed")
		b.toca(0, "other")
		atrasada.Abre()
		b.l.Roda()
		igual(t, []string{"other"}, b.naFila())
	})
	t.Run("uma falha do rádio (esperada ou não) deixa a fila como estava", func(t *testing.T) {
		r := &radio{resposta: func(context.Context, dominio.Faixa) ([]dominio.Faixa, error) { return nil, app.ErrCatalogo }}
		b := com(r, true)
		b.toca(0, "seed")
		igual(t, []string{"seed"}, b.naFila())
		igual(t, app.Tocando, b.s.Estado().Situacao)
	})
	t.Run("a resposta que chega depois de a fila acabar não deixa uma faixa morta", func(t *testing.T) {
		var b *bancada
		var atrasada *laco.Portao
		r := &radio{resposta: func(ctx context.Context, s dominio.Faixa) ([]dominio.Faixa, error) {
			if s.ID != "seed" {
				return []dominio.Faixa{prova.Faixa("x-r1")}, nil
			}
			if erro := atrasada.Espera(ctx); erro != nil {
				return nil, erro
			}
			return []dominio.Faixa{prova.Faixa("stale")}, nil
		}}
		b = novaBancada(func(o *app.OpcoesDaSessao) { o.Autoplay = r })
		atrasada = b.l.Portao()
		b.toca(0, "seed")
		b.s.Proxima()
		b.l.Roda()
		igual(t, "", b.atual())
		// Pôr na fila vazia começa de novo, com o rádio próprio.
		b.s.PoeNaFila(prova.Faixa("x"))
		b.l.Roda()
		atrasada.Abre()
		b.l.Roda()
		igual(t, []string{"x", "x-r1"}, b.naFila())
	})
}

func TestTocaColecaoEPorProvedor(t *testing.T) {
	album := []dominio.Faixa{prova.Faixa("a1"), prova.Faixa("a2")}
	tenta := func(resposta func() ([]dominio.Faixa, error)) (*bancada, bool) {
		b := novaBancada(nil)
		var deu bool
		b.s.TocaColecao(&radio{playlist: resposta}, "OLAK5uy_album", func(ok bool) { deu = ok })
		b.l.Roda()
		return b, deu
	}
	b, deu := tenta(func() ([]dominio.Faixa, error) { return album, nil })
	igual(t, true, deu)
	igual(t, []string{"a1", "a2"}, b.naFila())
	_, deu = tenta(func() ([]dominio.Faixa, error) { return nil, app.ErrCatalogo })
	igual(t, false, deu)
	_, deu = tenta(func() ([]dominio.Faixa, error) { return nil, nil })
	igual(t, false, deu)

	r := app.PorProvedor{dominio.YouTube: &prova.Resolvedor{Base: "https://yt.example/"}}
	s, _ := r.Resolve(context.Background(), prova.Faixa("1"))
	igual(t, "https://cdn.example/1.mp3", s.URL)
	s, _ = r.Resolve(context.Background(), prova.Faixa("abc", prova.SemStream, prova.Provedor(dominio.YouTube)))
	igual(t, "https://yt.example/abc", s.URL)
	_, erro := r.Resolve(context.Background(), prova.Faixa("abc", prova.SemStream, prova.Provedor(dominio.Spotify)))
	igual(t, true, errors.Is(erro, app.ErrStream))
}

func TestRetomar(t *testing.T) {
	duas := []dominio.Faixa{prova.Faixa("1"), prova.Faixa("2")}
	salva := func(indice int, posicao time.Duration) app.SessaoSalva {
		return app.SessaoSalva{Fila: dominio.NovaFila(duas, indice).Retrato(), Posicao: posicao}
	}
	plays := func(b *bancada) []string { return append(b.p.Com("play"), b.p.Com("seek")...) }
	t.Run("a sessão restaurada espera pausada e retoma de onde parou", func(t *testing.T) {
		b := novaBancada(nil)
		b.s.Restaura(salva(1, 42*time.Second))
		igual(t, app.Pausado, b.s.Estado().Situacao)
		igual(t, "2", b.atual())
		igual(t, 42*time.Second, b.s.Progresso().Posicao)
		igual(t, 0, len(b.p.Chamadas))
		b.s.AlternaPausa()
		b.l.Roda()
		igual(t, []string{"play https://stream.example/2", "seek 42s"}, plays(b))
		igual(t, app.Tocando, b.s.Estado().Situacao)
	})
	t.Run("tocar vindo de fora retoma; com a faixa carregando não muda nada", func(t *testing.T) {
		b := novaBancada(nil)
		b.s.Restaura(salva(1, 42*time.Second))
		b.s.AoEvento(app.Evento{Tipo: app.PediuTocar})
		b.s.AoEvento(app.Evento{Tipo: app.PediuTocar})
		b.l.Roda()
		igual(t, []string{"play https://stream.example/2", "seek 42s"}, plays(b))
	})
	t.Run("anterior numa faixa restaurada a recomeça, e pular muda onde ela retoma", func(t *testing.T) {
		b := novaBancada(nil)
		b.s.Restaura(salva(1, 90*time.Second))
		b.s.Anterior()
		igual(t, time.Duration(0), b.s.Progresso().Posicao)
		b.s.VaiPara(20 * time.Second)
		b.s.AlternaPausa()
		b.l.Roda()
		igual(t, []string{"play https://stream.example/2", "seek 20s"}, plays(b))
	})
	t.Run("o guardião restaura, guarda a fila quando muda e a posição a cada dez segundos", func(t *testing.T) {
		m := prova.NovaMemoria()
		antes := salva(0, 2*time.Minute)
		m.Sessao = &antes
		b := novaBancada(nil)
		app.GuardaSessao(b.s, m, b.l)
		igual(t, "1", b.atual())
		// Abrir o app sem apertar tocar mantém a posição guardada.
		igual(t, 2*time.Minute, m.Sessao.Posicao)
		b.s.TocaDe(duas, 0)
		b.l.Roda()
		b.s.Proxima()
		b.l.Roda()
		igual(t, 1, m.Sessao.Fila.Indice)
		b.p.Pos = 25 * time.Second
		guardadas := m.SessoesSal
		b.l.Avanca(11 * time.Second)
		igual(t, 25*time.Second, m.Sessao.Posicao)
		igual(t, guardadas+1, m.SessoesSal)
		b.s.AlternaPausa()
		b.l.Avanca(30 * time.Second)
		igual(t, guardadas+1, m.SessoesSal)
	})
	t.Run("sem nada guardado o tocador fica ocioso", func(t *testing.T) {
		b := novaBancada(nil)
		app.GuardaSessao(b.s, prova.NovaMemoria(), b.l)
		igual(t, "", b.atual())
	})
}

func TestSoneca(t *testing.T) {
	tocando := func() (*bancada, *app.Soneca) {
		b := novaBancada(nil)
		b.toca(0, "1", "2")
		return b, app.NovaSoneca(b.s, b.l)
	}
	t.Run("pausa a música quando o tempo acaba", func(t *testing.T) {
		b, soneca := tocando()
		soneca.Comeca(15 * time.Minute)
		b.l.Avanca(14 * time.Minute)
		igual(t, app.Tocando, b.s.Estado().Situacao)
		estado, resta := soneca.Estado()
		igual(t, app.SonecaContando, estado)
		igual(t, time.Minute, resta)
		b.l.Avanca(time.Minute + time.Second)
		igual(t, app.Pausado, b.s.Estado().Situacao)
		estado, _ = soneca.Estado()
		igual(t, app.SonecaDesligada, estado)
	})
	t.Run("cancelar deixa a música tocando", func(t *testing.T) {
		b, soneca := tocando()
		soneca.Comeca(5 * time.Minute)
		soneca.Cancela()
		b.l.Avanca(10 * time.Minute)
		igual(t, app.Tocando, b.s.Estado().Situacao)
	})
	t.Run("fim da faixa para depois da atual, e se desliga quando a fila é trocada", func(t *testing.T) {
		b, soneca := tocando()
		soneca.FimDaFaixa()
		igual(t, true, b.s.Estado().Fila.PararAposAtual())
		estado, _ := soneca.Estado()
		igual(t, app.SonecaFimDaFaixa, estado)
		b.toca(0, "album")
		estado, _ = soneca.Estado()
		igual(t, app.SonecaDesligada, estado)

		soneca.FimDaFaixa()
		soneca.Cancela()
		igual(t, false, b.s.Estado().Fila.PararAposAtual())
	})
	t.Run("o tempo acabando enquanto a próxima carrega ainda termina em silêncio", func(t *testing.T) {
		b, soneca := tocando()
		portao := b.pendente("2")
		soneca.Comeca(time.Minute)
		b.l.Avanca(59 * time.Second)
		b.s.Proxima()
		b.l.Avanca(5 * time.Second)
		igual(t, app.Carregando, b.s.Estado().Situacao)
		portao.Abre()
		b.l.Roda()
		igual(t, app.Pausado, b.s.Estado().Situacao)
	})
}

func TestHistoricoDeEscuta(t *testing.T) {
	musica := prova.Faixa("song", prova.Duracao(3*time.Minute))
	comeca := func() (*bancada, *prova.Memoria) {
		b := novaBancada(nil)
		m := prova.NovaMemoria()
		app.OuveHistorico(b.s, m, b.l)
		return b, m
	}
	t.Run("registra a tocada depois de trinta segundos ouvindo, e só uma vez", func(t *testing.T) {
		b, m := comeca()
		inicio := b.l.Agora()
		b.s.TocaDe([]dominio.Faixa{musica}, 0)
		b.l.Roda()
		b.l.Avanca(29 * time.Second)
		igual(t, 0, len(m.Tocadas))
		b.l.Avanca(3 * time.Second)
		igual(t, []dominio.Tocada{{Faixa: musica, Em: inicio.Add(30 * time.Second)}}, m.Tocadas)
		b.l.Avanca(3 * time.Minute)
		igual(t, 1, len(m.Tocadas))
	})
	t.Run("não registra a faixa pulada cedo, nem tempo pausado, nem pulo para a frente", func(t *testing.T) {
		b, m := comeca()
		b.s.TocaDe([]dominio.Faixa{musica, prova.Faixa("next", prova.Duracao(3*time.Minute))}, 0)
		b.l.Roda()
		b.l.Avanca(10 * time.Second)
		b.s.Proxima()
		b.l.Roda()
		b.s.VaiPara(2 * time.Minute)
		b.l.Avanca(2 * time.Second)
		b.s.AlternaPausa()
		b.l.Avanca(5 * time.Minute)
		igual(t, 0, len(m.Tocadas))
	})
	t.Run("rádio ao vivo não conta, e tocadas de mais de trinta dias são esquecidas", func(t *testing.T) {
		b, m := comeca()
		b.s.TocaDe([]dominio.Faixa{prova.Faixa("station", prova.Provedor(dominio.Radio), prova.Duracao(0))}, 0)
		b.l.Roda()
		b.l.Avanca(5 * time.Minute)
		igual(t, 0, len(m.Tocadas))
		m.Tocadas = append(m.Tocadas, dominio.Tocada{Faixa: prova.Faixa("ancient"), Em: b.l.Agora().Add(-31 * 24 * time.Hour)})
		b.s.TocaDe([]dominio.Faixa{musica}, 0)
		b.l.Roda()
		b.l.Avanca(31 * time.Second)
		igual(t, 1, len(m.Tocadas))
		igual(t, "song", m.Tocadas[0].Faixa.ID)
	})
}

// O tocador pode levar uns segundos para abrir uma faixa (a rede, a CDN):
// até ele dizer que carregou, a sessão continua em "carregando".
func TestSessaoEsperaOTocadorCarregar(t *testing.T) {
	lenta := func() *bancada {
		b := novaBancada(nil)
		b.p.Demora = true
		return b
	}
	t.Run("fica carregando até o tocador abrir a faixa, e só então toca", func(t *testing.T) {
		b := lenta()
		b.toca(0, "1", "2")
		igual(t, app.Carregando, b.s.Estado().Situacao)
		igual(t, "play https://stream.example/1", b.p.Chamadas[len(b.p.Chamadas)-1])
		b.p.Carrega()
		igual(t, app.Tocando, b.s.Estado().Situacao)
		// Um segundo "carregou" (atrasado) não muda mais nada.
		b.s.AlternaPausa()
		b.p.Carrega()
		igual(t, app.Pausado, b.s.Estado().Situacao)
	})
	t.Run("a faixa que o tocador não abre vira falha, com o motivo anotado", func(t *testing.T) {
		var recusas []string
		b := novaBancada(func(o *app.OpcoesDaSessao) {
			o.AoRecusar = func(f dominio.Faixa, motivo string) { recusas = append(recusas, f.ID+": "+motivo) }
		})
		b.p.Demora = true
		b.toca(0, "1")
		b.evento(app.Evento{Tipo: app.FalhouAoTocar, Motivo: "loading failed"})
		igual(t, app.FalhouTocador, b.s.Estado().Situacao)
		igual(t, []string{"1: loading failed"}, recusas)
		// Tocar de novo tenta outra vez.
		b.s.AlternaPausa()
		b.l.Roda()
		igual(t, app.Carregando, b.s.Estado().Situacao)
		b.p.Carrega()
		igual(t, app.Tocando, b.s.Estado().Situacao)
	})
	t.Run("o começo pedido e um pulo durante o carregamento chegam ao tocador", func(t *testing.T) {
		b := lenta()
		b.s.Restaura(app.SessaoSalva{Fila: dominio.NovaFila([]dominio.Faixa{prova.Faixa("1", prova.SemStream)}, 0).Retrato(), Posicao: 42 * time.Second})
		b.s.AlternaPausa()
		b.l.Roda()
		igual(t, app.Carregando, b.s.Estado().Situacao)
		igual(t, []string{"seek 42s"}, b.p.Com("seek"))
		// A posição pedida já aparece, em vez de um zero parado.
		igual(t, 42*time.Second, b.s.Progresso().Posicao)
		b.s.VaiPara(50 * time.Second)
		igual(t, []string{"seek 42s", "seek 50s"}, b.p.Com("seek"))
		b.p.Carrega()
		igual(t, app.Tocando, b.s.Estado().Situacao)
	})
	t.Run("pausar durante o carregamento vale para quando a faixa abrir", func(t *testing.T) {
		b := lenta()
		b.toca(0, "1")
		b.s.AlternaPausa()
		igual(t, app.Carregando, b.s.Estado().Situacao)
		igual(t, "pause", b.p.Chamadas[len(b.p.Chamadas)-1])
		b.p.Carrega()
		igual(t, app.Pausado, b.s.Estado().Situacao)
		b.s.AlternaPausa()
		igual(t, app.Tocando, b.s.Estado().Situacao)

		// Pausar e desistir da pausa antes de abrir: toca.
		b = lenta()
		b.toca(0, "1")
		b.s.AlternaPausa()
		b.s.AlternaPausa()
		igual(t, []string{"pause", "resume"}, b.p.Chamadas[len(b.p.Chamadas)-2:])
		b.p.Carrega()
		igual(t, app.Tocando, b.s.Estado().Situacao)

		// A pausa de uma faixa não passa para a seguinte.
		b = lenta()
		b.toca(0, "1", "2")
		b.s.AlternaPausa()
		b.s.Proxima()
		b.l.Roda()
		b.p.Carrega()
		igual(t, "2", b.atual())
		igual(t, app.Tocando, b.s.Estado().Situacao)
	})
	t.Run("trocar de faixa durante o carregamento não deixa nada preso", func(t *testing.T) {
		b := lenta()
		b.toca(0, "1", "2")
		portao := b.pendente("2")
		b.s.Proxima()
		b.l.Roda()
		// O que chega da faixa antiga (abriu, falhou, acabou) enquanto a nova é
		// resolvida não é da nova.
		b.p.Carrega()
		igual(t, app.Carregando, b.s.Estado().Situacao)
		b.evento(app.Evento{Tipo: app.FalhouAoTocar, Motivo: "loading failed"})
		b.evento(app.Evento{Tipo: app.Terminou})
		igual(t, app.Carregando, b.s.Estado().Situacao)
		igual(t, "2", b.atual())
		portao.Abre()
		b.l.Roda()
		igual(t, app.Carregando, b.s.Estado().Situacao)
		b.p.Carrega()
		igual(t, app.Tocando, b.s.Estado().Situacao)
		igual(t, "2", b.atual())
	})
	t.Run("o eco da pausa que segura a faixa antiga não aparece como pausado", func(t *testing.T) {
		b := lenta()
		b.toca(0, "1")
		b.evento(app.Evento{Tipo: app.MudouPausa, Tocando: false})
		igual(t, app.Carregando, b.s.Estado().Situacao)
		b.p.Carrega()
		igual(t, app.Tocando, b.s.Estado().Situacao)
	})
}
