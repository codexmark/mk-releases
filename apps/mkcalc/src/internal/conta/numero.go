package conta

import (
	"math/big"
	"strings"
)

// LerNumero lê um número escrito com algarismos, pontos e vírgulas, pela regra
// do contrato ("Vírgula, ponto e separador de milhar"):
//
//  1. um separador só, uma vez só, é o decimal (1,5 e 1.5), menos o ponto com
//     três algarismos depois e de um a três antes, que é milhar (1.234);
//  2. o mesmo separador repetido é milhar, com os grupos certos (1.234.567);
//  3. com os dois, o último é o decimal e o outro é milhar (1.234,56).
//
// Devolve false quando o texto não é um número. A mesma regra está em
// qml/Calc.js (normalizarNumero); as duas passam por testdata/numeros.json.
func LerNumero(s string) (*big.Rat, bool) {
	pontos, virgulas := strings.Count(s, "."), strings.Count(s, ",")
	var inteiro, frac string
	switch {
	case pontos+virgulas == 0:
		inteiro = s
	case pontos+virgulas == 1:
		i := strings.IndexAny(s, ".,")
		inteiro, frac = s[:i], s[i+1:]
		if s[i] == '.' && len(frac) == 3 && len(inteiro) >= 1 && len(inteiro) <= 3 && inteiro[0] != '0' {
			// 1.234 é mil duzentos e trinta e quatro, como se escreve em português.
			inteiro, frac = inteiro+frac, ""
		}
	case pontos == 0 || virgulas == 0:
		sep := "."
		if pontos == 0 {
			sep = ","
		}
		if !emGrupos(s, sep) {
			return nil, false
		}
		inteiro = strings.ReplaceAll(s, sep, "")
	default:
		i := strings.LastIndexAny(s, ".,")
		decimal, milhar := s[i:i+1], "."
		if decimal == "." {
			milhar = ","
		}
		if strings.Count(s, decimal) != 1 || !emGrupos(s[:i], milhar) {
			return nil, false
		}
		inteiro, frac = strings.ReplaceAll(s[:i], milhar, ""), s[i+1:]
	}
	if inteiro+frac == "" || !soAlgarismos(inteiro) || !soAlgarismos(frac) {
		return nil, false
	}
	n, ok := new(big.Int).SetString(inteiro+frac, 10)
	if !ok {
		return nil, false
	}
	return new(big.Rat).SetFrac(n, dezElevado(len(frac))), true
}

// emGrupos diz se s são grupos de milhar bem feitos: um a três algarismos e
// depois sempre três, separados por sep.
func emGrupos(s, sep string) bool {
	partes := strings.Split(s, sep)
	if len(partes) < 2 || len(partes[0]) < 1 || len(partes[0]) > 3 {
		return false
	}
	for i, p := range partes {
		if !soAlgarismos(p) || (i > 0 && len(p) != 3) {
			return false
		}
	}
	return true
}

func soAlgarismos(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func dezElevado(n int) *big.Int {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n)), nil)
}
