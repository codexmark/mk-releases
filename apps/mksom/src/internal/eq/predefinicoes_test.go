package eq

import (
	"strings"
	"testing"
)

func TestFabrica_AsSeisDoContratoTodasValidas(t *testing.T) {
	quer := []string{"plano", "graves", "agudos", "voz", "noite", "fone pequeno"}
	if len(Fabrica) != len(quer) {
		t.Fatalf("%d predefinições", len(Fabrica))
	}
	for i, p := range Fabrica {
		if p.Nome != quer[i] || !p.Fabrica {
			t.Errorf("%d: %q fábrica=%v", i, p.Nome, p.Fabrica)
		}
		if err := p.Ajuste().Valida(); err != nil {
			t.Errorf("%s: %v", p.Nome, err)
		}
		if !DeFabrica(strings.ToUpper(p.Nome)) {
			t.Errorf("%s: DeFabrica não vê maiúsculas", p.Nome)
		}
	}
	plano := Fabrica[0].Ajuste()
	if !plano.Igual(Plano()) {
		t.Errorf("plano: %v", plano)
	}
	// A noite corta os graves e os agudos e não levanta nada.
	noite := Fabrica[4]
	if noite.Ganhos[0] >= 0 || noite.Ganhos[9] >= 0 {
		t.Errorf("noite: %v", noite.Ganhos)
	}
	for _, g := range noite.Ganhos {
		if g > 0 {
			t.Errorf("noite levanta: %v", noite.Ganhos)
		}
	}
	// Quem levanta alguma banda vem com pré-ganho para não estourar.
	for _, p := range Fabrica {
		maior := 0.0
		for _, g := range p.Ganhos {
			if g > maior {
				maior = g
			}
		}
		if maior > 0 && p.Pre > -maior {
			t.Errorf("%s: levanta %v dB com pré %v", p.Nome, maior, p.Pre)
		}
	}
	if DeFabrica("minha") {
		t.Error("minha não é de fábrica")
	}
}

func TestValidaNome(t *testing.T) {
	for _, bom := range []string{"minha", "Fone do trabalho", "rock & jazz (2)", "água", strings.Repeat("é", 40), "a"} {
		if _, err := ValidaNome(bom); err != nil {
			t.Errorf("%q: %v", bom, err)
		}
	}
	if n, err := ValidaNome("  com espaços  "); err != nil || n != "com espaços" {
		t.Errorf("apara: %q %v", n, err)
	}
	for _, ruim := range []string{"", "   ", "a/b", "/", "..", ".", `a\b`, "a\nb", "a\tb", "a\x00b", "a\x7fb", "a\u0085b", "\xff\xfe", strings.Repeat("a", 41), strings.Repeat("é", 41), "-rf", "--"} {
		if _, err := ValidaNome(ruim); err == nil {
			t.Errorf("%q: tinha de recusar", ruim)
		}
	}
}

func TestAcha_FabricaPrimeiroSemOlharMaiusculas(t *testing.T) {
	daPessoa := []Predefinicao{{Nome: "Minha", Ganhos: []float64{1, 0, 0, 0, 0, 0, 0, 0, 0, 0}, Pre: -1}}
	todas := Todas(daPessoa)
	if len(todas) != len(Fabrica)+1 || todas[0].Nome != "plano" || todas[len(todas)-1].Nome != "Minha" || todas[len(todas)-1].Fabrica {
		t.Errorf("todas: %v", todas)
	}
	if p, ok := Acha(todas, "minha"); !ok || p.Pre != -1 {
		t.Errorf("minha: %v %v", p, ok)
	}
	if p, ok := Acha(todas, "GRAVES"); !ok || !p.Fabrica {
		t.Errorf("graves: %v %v", p, ok)
	}
	if _, ok := Acha(todas, "outra"); ok {
		t.Error("outra não existe")
	}
	// Qual tem exatamente estes valores.
	if n := QualE(todas, Plano()); n != "plano" {
		t.Errorf("plano: %q", n)
	}
	if n := QualE(todas, daPessoa[0].Ajuste()); n != "Minha" {
		t.Errorf("minha: %q", n)
	}
	if n := QualE(todas, Ajuste{Ganhos: []float64{0.5, 0, 0, 0, 0, 0, 0, 0, 0, 0}}); n != "" {
		t.Errorf("nenhuma: %q", n)
	}
}
