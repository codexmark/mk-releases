// SPDX-License-Identifier: GPL-3.0-or-later

package laco

import (
	"context"
	"sync"
	"time"
)

// Teste é o laço dos testes: o relógio só anda com Avanca, e Roda executa tudo
// que está pendente até o daemon sossegar (o que foi para fora com Vai é
// esperado, a não ser que esteja parado num Portao). Quem dirige é a goroutine
// do teste: é ela que faz o papel do laço.
type Teste struct {
	mu     sync.Mutex
	mudou  *sync.Cond
	fila   []func()
	ativos int
	agora  time.Time
	prazos []*prazo
	seq    int
}

type prazo struct {
	tarefa
	quando time.Time
	seq    int
	f      func()
}

// NovoTeste começa em 22/11/2026, meio-dia (UTC), como os testes da fonte.
func NovoTeste() *Teste {
	l := &Teste{agora: time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)}
	l.mudou = sync.NewCond(&l.mu)
	return l
}

func (l *Teste) Posta(f func()) {
	l.mu.Lock()
	l.fila = append(l.fila, f)
	l.mu.Unlock()
	l.mudou.Broadcast()
}

func (l *Teste) Agora() time.Time {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.agora
}

func (l *Teste) Depois(d time.Duration, f func()) Tarefa {
	l.seq++
	p := &prazo{quando: l.Agora().Add(d), seq: l.seq, f: f}
	l.prazos = append(l.prazos, p)
	return p
}

func (l *Teste) Vai(trabalho func(ctx context.Context) func()) Tarefa {
	ctx, cancela := context.WithCancel(context.Background())
	t := &tarefa{aoCancela: cancela}
	l.mu.Lock()
	l.ativos++
	l.mu.Unlock()
	go func() {
		depois := trabalho(ctx)
		l.mu.Lock()
		l.fila = append(l.fila, func() {
			cancela()
			if t.Ativa() {
				t.feita = true
				if depois != nil {
					depois()
				}
			}
		})
		l.ativos--
		l.mu.Unlock()
		l.mudou.Broadcast()
	}()
	return t
}

// Roda executa o que está na fila e espera os trabalhos de fora, até não
// sobrar nada a fazer agora.
func (l *Teste) Roda() {
	for {
		l.mu.Lock()
		for len(l.fila) == 0 && l.ativos > 0 {
			l.mudou.Wait()
		}
		if len(l.fila) == 0 {
			l.mu.Unlock()
			return
		}
		f := l.fila[0]
		l.fila = l.fila[1:]
		l.mu.Unlock()
		f()
	}
}

// Avanca anda o relógio, rodando cada prazo na sua hora.
func (l *Teste) Avanca(d time.Duration) {
	alvo := l.Agora().Add(d)
	for {
		l.Roda()
		var proximo *prazo
		vivos := l.prazos[:0]
		for _, p := range l.prazos {
			if !p.Ativa() {
				continue
			}
			vivos = append(vivos, p)
			if p.quando.After(alvo) {
				continue
			}
			if proximo == nil || p.quando.Before(proximo.quando) || (p.quando.Equal(proximo.quando) && p.seq < proximo.seq) {
				proximo = p
			}
		}
		l.prazos = vivos
		if proximo == nil {
			break
		}
		l.mu.Lock()
		if proximo.quando.After(l.agora) {
			l.agora = proximo.quando
		}
		l.mu.Unlock()
		proximo.feita = true
		proximo.f()
	}
	l.mu.Lock()
	l.agora = alvo
	l.mu.Unlock()
	l.Roda()
}

// Portao segura um trabalho de fora até o teste abrir: é a resposta que
// demora. Enquanto espera, o trabalho não conta para o Roda.
type Portao struct {
	l         *Teste
	abriu     chan struct{}
	aberto    bool
	esperando bool
	desistiu  bool
}

// Portao cria um portão fechado.
func (l *Teste) Portao() *Portao { return &Portao{l: l, abriu: make(chan struct{})} }

// Espera bloqueia até Abre, ou até o trabalho ser cancelado (devolve o erro).
func (p *Portao) Espera(ctx context.Context) error {
	p.l.mu.Lock()
	if p.aberto {
		p.l.mu.Unlock()
		return nil
	}
	p.esperando = true
	p.l.ativos--
	p.l.mu.Unlock()
	p.l.mudou.Broadcast()
	select {
	case <-p.abriu:
		return nil
	case <-ctx.Done():
		p.l.mu.Lock()
		defer p.l.mu.Unlock()
		if !p.aberto {
			p.desistiu = true
			p.l.ativos++
		}
		return ctx.Err()
	}
}

// Abre solta quem espera (e quem vier a esperar).
func (p *Portao) Abre() {
	p.l.mu.Lock()
	if !p.aberto {
		p.aberto = true
		if p.esperando && !p.desistiu {
			p.l.ativos++
		}
		close(p.abriu)
	}
	p.l.mu.Unlock()
}
