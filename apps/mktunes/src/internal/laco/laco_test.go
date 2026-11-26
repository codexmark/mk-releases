// SPDX-License-Identifier: GPL-3.0-or-later

package laco

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func TestRealRodaNaOrdemEVoltaDeFora(t *testing.T) {
	l := Novo()
	var visto []string
	l.Posta(func() { visto = append(visto, "um") })
	l.Vai(func(context.Context) func() {
		return func() {
			visto = append(visto, "de fora")
			l.Depois(time.Millisecond, func() {
				visto = append(visto, "prazo")
				l.Fecha()
			})
		}
	})
	cancelada := l.Depois(time.Hour, func() { visto = append(visto, "nunca") })
	l.Posta(func() { cancelada.Cancela() })
	l.Roda()
	if !reflect.DeepEqual(visto, []string{"um", "de fora", "prazo"}) {
		t.Fatalf("visto: %v", visto)
	}
	if cancelada.Ativa() || Ativa(nil) {
		t.Fatal("tarefa cancelada (ou nula) não é ativa")
	}
	Cancela(nil)
}

func TestRealNaoEntregaTrabalhoCancelado(t *testing.T) {
	l := Novo()
	entregue := false
	solta := make(chan struct{})
	tarefa := l.Vai(func(ctx context.Context) func() {
		<-ctx.Done()
		close(solta)
		return func() { entregue = true }
	})
	l.Posta(func() { tarefa.Cancela() })
	go func() {
		<-solta
		l.Posta(func() { l.Fecha() })
	}()
	l.Roda()
	if entregue {
		t.Fatal("um trabalho cancelado não volta ao laço")
	}
}

func TestTesteAndaORelogioNaMao(t *testing.T) {
	l := NovoTeste()
	inicio := l.Agora()
	var visto []string
	l.Depois(2*time.Second, func() { visto = append(visto, "dois") })
	l.Depois(time.Second, func() {
		visto = append(visto, "um")
		l.Depois(500*time.Millisecond, func() { visto = append(visto, "um e meio") })
	})
	fora := l.Depois(time.Second, func() { visto = append(visto, "cancelado") })
	fora.Cancela()
	l.Avanca(1900 * time.Millisecond)
	if !reflect.DeepEqual(visto, []string{"um", "um e meio"}) {
		t.Fatalf("visto: %v", visto)
	}
	l.Avanca(time.Second)
	if len(visto) != 3 || l.Agora().Sub(inicio) != 2900*time.Millisecond {
		t.Fatalf("visto: %v em %v", visto, l.Agora().Sub(inicio))
	}
}

func TestTesteEsperaOQueFoiParaForaMenosOQueEstaNoPortao(t *testing.T) {
	l := NovoTeste()
	portao := l.Portao()
	var visto []string
	l.Vai(func(context.Context) func() { return func() { visto = append(visto, "rápido") } })
	l.Vai(func(ctx context.Context) func() {
		if portao.Espera(ctx) != nil {
			return nil
		}
		return func() { visto = append(visto, "lento") }
	})
	l.Roda()
	if !reflect.DeepEqual(visto, []string{"rápido"}) {
		t.Fatalf("visto: %v", visto)
	}
	portao.Abre()
	l.Roda()
	if !reflect.DeepEqual(visto, []string{"rápido", "lento"}) {
		t.Fatalf("visto: %v", visto)
	}

	preso := l.Portao()
	tarefa := l.Vai(func(ctx context.Context) func() {
		_ = preso.Espera(ctx)
		return func() { visto = append(visto, "cancelado") }
	})
	l.Roda()
	tarefa.Cancela()
	preso.Abre()
	l.Roda()
	if len(visto) != 2 {
		t.Fatalf("visto: %v", visto)
	}
}
