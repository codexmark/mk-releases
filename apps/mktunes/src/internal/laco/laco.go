// SPDX-License-Identifier: GPL-3.0-or-later

// Pacote laco: o laço único do daemon. Todo o estado (a sessão, as áreas da
// janela, a guarda) só é tocado por funções que rodam nele, uma de cada vez;
// o que demora (rede, disco grande) roda fora, em Vai, e volta ao laço
// com o resultado. É o que troca as corrotinas do daemon em Kotlin.
package laco

import (
	"context"
	"sync"
	"time"
)

// Tarefa é algo marcado no laço (um prazo, um trabalho de fora) que ainda pode
// ser cancelado. Só o laço chama Cancela e Ativa.
type Tarefa interface {
	// Cancela desiste: o que ainda não rodou no laço não roda mais.
	Cancela()
	// Ativa diz se ainda falta rodar.
	Ativa() bool
}

// Laco é o que o resto do daemon usa; Real é o de verdade e Teste o de mentira,
// com relógio virtual.
type Laco interface {
	// Posta roda f no laço. Pode ser chamada de qualquer goroutine.
	Posta(f func())
	// Depois roda f no laço daqui a d.
	Depois(d time.Duration, f func()) Tarefa
	// Vai roda trabalho fora do laço e, quando ele acaba, roda no laço a função
	// que ele devolveu (se não foi cancelado nesse meio tempo).
	Vai(trabalho func(ctx context.Context) func()) Tarefa
	// Agora é o relógio.
	Agora() time.Time
}

// Ativa trata tarefa nula como inativa.
func Ativa(t Tarefa) bool { return t != nil && t.Ativa() }

// Cancela trata tarefa nula como já cancelada.
func Cancela(t Tarefa) {
	if t != nil {
		t.Cancela()
	}
}

type tarefa struct {
	cancelada bool
	feita     bool
	aoCancela func()
}

func (t *tarefa) Cancela() {
	if t.cancelada || t.feita {
		return
	}
	t.cancelada = true
	if t.aoCancela != nil {
		t.aoCancela()
	}
}

func (t *tarefa) Ativa() bool { return !t.cancelada && !t.feita }

// Real é o laço de verdade: uma goroutine (quem chama Roda) executa tudo.
type Real struct {
	mu      sync.Mutex
	chegou  *sync.Cond
	fila    []func()
	fechado bool
}

// Novo cria o laço; ele só anda depois de Roda.
func Novo() *Real {
	l := &Real{}
	l.chegou = sync.NewCond(&l.mu)
	return l
}

// Roda executa o que for postado até Fecha.
func (l *Real) Roda() {
	for {
		l.mu.Lock()
		for len(l.fila) == 0 && !l.fechado {
			l.chegou.Wait()
		}
		if len(l.fila) == 0 {
			l.mu.Unlock()
			return
		}
		f := l.fila[0]
		l.fila[0] = nil
		l.fila = l.fila[1:]
		l.mu.Unlock()
		f()
	}
}

// Fecha para o laço depois do que já está na fila.
func (l *Real) Fecha() {
	l.mu.Lock()
	l.fechado = true
	l.mu.Unlock()
	l.chegou.Broadcast()
}

func (l *Real) Posta(f func()) {
	l.mu.Lock()
	if !l.fechado {
		l.fila = append(l.fila, f)
	}
	l.mu.Unlock()
	l.chegou.Signal()
}

func (l *Real) Depois(d time.Duration, f func()) Tarefa {
	t := &tarefa{}
	relogio := time.AfterFunc(d, func() {
		l.Posta(func() {
			if t.Ativa() {
				t.feita = true
				f()
			}
		})
	})
	t.aoCancela = func() { relogio.Stop() }
	return t
}

func (l *Real) Vai(trabalho func(ctx context.Context) func()) Tarefa {
	ctx, cancela := context.WithCancel(context.Background())
	t := &tarefa{aoCancela: cancela}
	go func() {
		depois := trabalho(ctx)
		l.Posta(func() {
			cancela()
			if t.Ativa() {
				t.feita = true
				if depois != nil {
					depois()
				}
			}
		})
	}()
	return t
}

func (l *Real) Agora() time.Time { return time.Now() }
