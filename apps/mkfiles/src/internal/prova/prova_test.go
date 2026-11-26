package prova

import (
	"os"
	"strings"
	"testing"

	"mkfiles/internal/base"
)

func TestOArnes(t *testing.T) {
	m := Novo(t, func(s *base.Sessao) {
		s.Ligar("eco", func(p *base.Pedido) {
			p.S.Evento("ouvi", base.Campos{"op": p.Op})
			p.Responde(base.Campos{"fim": false})
			p.Responde(base.Campos{"fim": true})
		})
	})
	if !strings.HasPrefix(os.Getenv("XDG_CONFIG_HOME"), m.Casa) || base.Home() != m.Casa {
		t.Fatal("a casa de mentira não é a pasta pessoal")
	}
	if linhas := m.PedeFluxo(`{"op":"eco"}`); len(linhas) != 2 {
		t.Fatalf("fluxo: %v", linhas)
	}
	if e := m.Evento("ouvi", nil); e["op"] != "eco" {
		t.Fatalf("evento: %v", e)
	}
	c := m.Grava("a/b.txt", "x")
	if b, _ := os.ReadFile(c); string(b) != "x" {
		t.Fatal("Grava")
	}
	m.Montagem(m.Pasta("disco"), "/dev/x")
	if base.PontoDe(m.Casa+"/disco/a") != m.Casa+"/disco" {
		t.Fatalf("PontoDe: %s", base.PontoDe(m.Casa+"/disco/a"))
	}
	if J([]string{"a"}) != `["a"]` {
		t.Fatal("J")
	}
}
