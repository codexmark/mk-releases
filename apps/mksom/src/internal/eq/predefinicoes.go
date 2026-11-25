package eq

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Limites das predefinições da pessoa.
const (
	NomeMax          = 40
	PredefinicoesMax = 50
)

// Predefinicao é um ajuste com nome. As de fábrica não mudam.
type Predefinicao struct {
	Nome    string    `json:"nome"`
	Fabrica bool      `json:"fabrica"`
	Ganhos  []float64 `json:"ganhos"`
	Pre     float64   `json:"pre"`
}

// Ajuste devolve os valores da predefinição (uma cópia).
func (p Predefinicao) Ajuste() Ajuste {
	return Ajuste{Ganhos: append([]float64{}, p.Ganhos...), Pre: p.Pre}
}

// Fabrica são as predefinições que vêm com o app, na ordem em que aparecem.
// O pré-ganho de cada uma cobre o pico da curva (veja Som.js: preSugerido).
//
//	31   62   125  250  500  1k   2k   4k   8k   16k
var Fabrica = []Predefinicao{
	{"plano", true, []float64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, 0},
	{"graves", true, []float64{4, 3.5, 2.5, 1, 0, 0, 0, 0, 0, 0}, -5},
	{"agudos", true, []float64{0, 0, 0, 0, 0, 0, 1, 2.5, 3.5, 4}, -4.5},
	{"voz", true, []float64{-4, -3, -1, 0, 1, 2.5, 3, 2, 0, -1}, -4.5},
	{"noite", true, []float64{-6, -4, -2, 0, 0, 0, 0, -1, -3, -5}, 0},
	{"fone pequeno", true, []float64{4, 3.5, 2.5, 1, 0, 0, 0, 1, 1.5, 1.5}, -5},
}

// DeFabrica diz se o nome é de uma predefinição de fábrica (sem olhar maiúsculas).
func DeFabrica(nome string) bool {
	for _, p := range Fabrica {
		if strings.EqualFold(p.Nome, nome) {
			return true
		}
	}
	return false
}

// ValidaNome apara o nome de uma predefinição da pessoa e o recusa se estiver
// vazio, comprido, com barra, com caractere de controle ou com cara de opção.
func ValidaNome(nome string) (string, error) {
	if !utf8.ValidString(nome) {
		return "", errors.New("o nome tem caracteres inválidos")
	}
	// Controle primeiro (uma quebra de linha no fim não é "espaço em volta");
	// depois só os espaços das pontas saem. A janela faz igual (Som.erroDoNome).
	for _, r := range nome {
		if unicode.IsControl(r) {
			return "", errors.New("o nome não pode ter caractere de controle")
		}
	}
	nome = strings.Trim(nome, " ")
	switch {
	case nome == "":
		return "", errors.New("dê um nome à predefinição")
	case utf8.RuneCountInString(nome) > NomeMax:
		return "", errors.New("o nome pode ter até 40 caracteres")
	case strings.ContainsAny(nome, `/\`):
		return "", errors.New("o nome não pode ter barra")
	case strings.HasPrefix(nome, "-") || nome == "." || nome == "..":
		return "", errors.New("o nome não pode começar com - nem ser só pontos")
	}
	return nome, nil
}

// Todas junta as de fábrica (primeiro) com as da pessoa.
func Todas(daPessoa []Predefinicao) []Predefinicao {
	out := make([]Predefinicao, 0, len(Fabrica)+len(daPessoa))
	for _, p := range Fabrica {
		out = append(out, Predefinicao{p.Nome, true, append([]float64{}, p.Ganhos...), p.Pre})
	}
	for _, p := range daPessoa {
		out = append(out, Predefinicao{p.Nome, false, append([]float64{}, p.Ganhos...), p.Pre})
	}
	return out
}

// Acha procura uma predefinição pelo nome, sem olhar maiúsculas.
func Acha(lista []Predefinicao, nome string) (Predefinicao, bool) {
	for _, p := range lista {
		if strings.EqualFold(p.Nome, nome) {
			return p, true
		}
	}
	return Predefinicao{}, false
}

// QualE devolve o nome da primeira predefinição com exatamente estes valores,
// ou "".
func QualE(lista []Predefinicao, a Ajuste) string {
	for _, p := range lista {
		if p.Ajuste().Igual(a) {
			return p.Nome
		}
	}
	return ""
}
