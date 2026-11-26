// SPDX-License-Identifier: GPL-3.0-or-later

package prova

import (
	"context"
	"errors"
	"testing"
	"time"

	"mktunes/internal/app"
	"mktunes/internal/dominio"
)

// Os dublês têm de cumprir as portas que fingem ser.
var (
	_ app.Tocador    = (*Tocador)(nil)
	_ app.Resolvedor = (*Resolvedor)(nil)
	_ app.Guarda     = (*Memoria)(nil)
)

func TestDubles(t *testing.T) {
	f := Faixa("1", Titulo("Umbrella"), SemStream)
	if f.Titulo != "Umbrella" || f.Stream != "" || f.Duracao != 30*time.Second {
		t.Fatalf("faixa: %+v", f)
	}
	r := &Resolvedor{Indisponiveis: map[string]bool{"x": true}}
	if s, _ := r.Resolve(context.Background(), f); s.URL != "https://stream.example/1" {
		t.Fatalf("stream: %+v", s)
	}
	if _, erro := r.Resolve(context.Background(), Faixa("x")); !errors.Is(erro, app.ErrStream) {
		t.Fatalf("erro: %v", erro)
	}
	m := NovaMemoria()
	m.Curte(Faixa("a"), true, time.Time{})
	m.Curte(Faixa("b"), true, time.Time{})
	m.Curte(Faixa("a"), false, time.Time{})
	if ids := IDs(m.Curtidas()); len(ids) != 1 || ids[0] != "b" {
		t.Fatalf("curtidas: %v", ids)
	}
	m.SalvaColecao(dominio.ColecaoSalva{ID: "1"})
	m.SalvaColecao(dominio.ColecaoSalva{ID: "2"})
	if m.Salvos()[0].ID != "2" {
		t.Fatal("o salvo mais novo vem primeiro")
	}
}
