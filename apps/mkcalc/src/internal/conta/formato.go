package conta

import (
	"math/big"
	"strings"
)

// casasFinitas diz com quantas casas decimais r se escreve inteiro, se for um
// decimal finito (o denominador só tem os fatores 2 e 5).
func casasFinitas(r *big.Rat) (int, bool) {
	den := new(big.Int).Set(r.Denom())
	conta := func(fator int64) int {
		f, resto, n := big.NewInt(fator), new(big.Int), 0
		for {
			q, m := new(big.Int).QuoRem(den, f, resto)
			if m.Sign() != 0 {
				return n
			}
			den, n = q, n+1
		}
	}
	dois, cinco := conta(2), conta(5)
	if den.Cmp(big.NewInt(1)) != 0 {
		return 0, false
	}
	return max(dois, cinco), true
}

// zerosNaFrente conta os zeros entre a vírgula e o primeiro algarismo que não
// é zero de um número menor que 1 em módulo (0,00033 tem 3). Zero para o resto.
func zerosNaFrente(r *big.Rat) int {
	num := new(big.Int).Abs(r.Num())
	if num.Sign() == 0 || num.Cmp(r.Denom()) >= 0 {
		return 0
	}
	// Pelo tamanho dos dois dá para chegar a um do valor; daí é só acertar.
	k := max(len(r.Denom().String())-len(num.String())-1, 0)
	for new(big.Int).Mul(num, dezElevado(k+1)).Cmp(r.Denom()) < 0 {
		k++
	}
	return k
}

// decimal escreve r com exatamente `casas` casas decimais, arredondando o meio
// para longe do zero, e tira os zeros do fim. Devolve o sinal, a parte inteira
// e a decimal (sem a vírgula; vazia quando não há).
func decimal(r *big.Rat, casas int) (negativo bool, inteiro, frac string) {
	num := new(big.Int).Abs(r.Num())
	num.Mul(num, dezElevado(casas))
	q, resto := new(big.Int).QuoRem(num, r.Denom(), new(big.Int))
	if resto.Lsh(resto, 1).Cmp(r.Denom()) >= 0 {
		q.Add(q, big.NewInt(1))
	}
	s := q.String()
	if len(s) <= casas {
		s = strings.Repeat("0", casas-len(s)+1) + s
	}
	inteiro, frac = s[:len(s)-casas], strings.TrimRight(s[len(s)-casas:], "0")
	return r.Sign() < 0 && q.Sign() != 0, inteiro, frac
}

// agrupar põe o ponto de milhar numa sequência de algarismos.
func agrupar(inteiro string) string {
	if len(inteiro) <= 3 {
		return inteiro
	}
	var b strings.Builder
	cabeca := len(inteiro) % 3
	b.WriteString(inteiro[:cabeca])
	for i := cabeca; i < len(inteiro); i += 3 {
		if b.Len() > 0 {
			b.WriteByte('.')
		}
		b.WriteString(inteiro[i : i+3])
	}
	return b.String()
}

func montar(negativo bool, inteiro, frac string) string {
	s := inteiro
	if frac != "" {
		s += "," + frac
	}
	if negativo {
		s = "-" + s
	}
	return s
}

// exato escreve um decimal finito inteiro, sem separador de milhar (é como um
// número aparece na expressão arrumada).
func exato(r *big.Rat) string {
	casas, _ := casasFinitas(r)
	return montar(decimal(r, casas))
}

// formatar escreve o valor de uma conta pela regra do contrato ("Exato e
// aproximado"): bonito é para mostrar (milhar com ponto), cru é para copiar.
func formatar(v valor) (bonito, cru string, aproximado bool) {
	r := v.r
	if v.aprox {
		// O que sobra abaixo de 10^-30 numa conta aproximada é ruído.
		if new(big.Rat).Abs(r).Cmp(new(big.Rat).SetFrac(big.NewInt(1), dezElevado(casasDoRuido))) < 0 {
			r = new(big.Rat)
		}
	}
	casas, finito := casasFinitas(r)
	aproximado = v.aprox || !finito || casas > CasasExatas
	if aproximado {
		// Nunca mais casas do que cabem num número da conta: o resultado tem de
		// poder voltar para a expressão.
		casas = min(CasasAproximadas+zerosNaFrente(r), casasMaximas)
	}
	negativo, inteiro, frac := decimal(r, casas)
	return montar(negativo, agrupar(inteiro), frac), montar(negativo, inteiro, frac), aproximado
}
