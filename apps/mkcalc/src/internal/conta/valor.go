package conta

import (
	"math"
	"math/big"
)

// Um valor no meio da conta: a fração exata e se ela já passou por alguma
// aproximação (uma raiz que não fecha).
type valor struct {
	r     *big.Rat
	aprox bool
}

// Um nó da expressão lida.
type no interface {
	valer() (valor, error)
}

type (
	numero   struct{ r *big.Rat }
	menos    struct{ de no }
	porcento struct{ de no }
	raiz     struct{ de no }
	grupo    struct{ de no } // um parêntese: (10%) não é "10% do total"
	dupla    struct {
		op       rune
		esq, dir no
	}
)

var (
	um  = big.NewRat(1, 1)
	cem = big.NewRat(100, 1)
)

// cabe recusa um número que passou do limite do contrato.
func cabe(v valor) (valor, error) {
	if v.r.Num().BitLen() > LimiteDeBits || v.r.Denom().BitLen() > LimiteDeBits {
		return valor{}, erro("número grande demais")
	}
	return v, nil
}

func (n *numero) valer() (valor, error) { return cabe(valor{r: n.r}) }

func (n *grupo) valer() (valor, error) { return n.de.valer() }

func (n *menos) valer() (valor, error) {
	v, err := n.de.valer()
	if err != nil {
		return valor{}, err
	}
	return valor{r: new(big.Rat).Neg(v.r), aprox: v.aprox}, nil
}

// Sozinho, o % divide por 100. O "10% do total" é da soma (veja dupla.valer).
func (n *porcento) valer() (valor, error) {
	v, err := n.de.valer()
	if err != nil {
		return valor{}, err
	}
	return cabe(valor{r: new(big.Rat).Quo(v.r, cem), aprox: v.aprox})
}

func (n *raiz) valer() (valor, error) {
	v, err := n.de.valer()
	if err != nil {
		return valor{}, err
	}
	if v.r.Sign() < 0 {
		return valor{}, erro("raiz de número negativo")
	}
	return raizDe(v, 2), nil
}

func (n *dupla) valer() (valor, error) {
	e, err := n.esq.valer()
	if err != nil {
		return valor{}, err
	}
	var d valor
	if p, ok := n.dir.(*porcento); ok && (n.op == '+' || n.op == '-') {
		// 200 + 10%: a porcentagem do total até aqui.
		x, err := p.de.valer()
		if err != nil {
			return valor{}, err
		}
		d = valor{r: new(big.Rat).Mul(e.r, new(big.Rat).Quo(x.r, cem)), aprox: e.aprox || x.aprox}
	} else if d, err = n.dir.valer(); err != nil {
		return valor{}, err
	}
	v := valor{r: new(big.Rat), aprox: e.aprox || d.aprox}
	switch n.op {
	case '+':
		v.r.Add(e.r, d.r)
	case '-':
		v.r.Sub(e.r, d.r)
	case '×':
		v.r.Mul(e.r, d.r)
	case '÷':
		if d.r.Sign() == 0 {
			return valor{}, erro("divisão por zero")
		}
		v.r.Quo(e.r, d.r)
	case '^':
		return elevar(e, d)
	}
	return cabe(v)
}

// elevar faz base^expoente. O expoente inteiro é exato; o fracionário p/q é a
// raiz de índice q de base^p (exata quando fecha). O tamanho do resultado é
// conferido antes de qualquer conta: 9^9^9 é recusado na hora.
func elevar(base, expoente valor) (valor, error) {
	aprox := base.aprox || expoente.aprox
	p, q := expoente.r.Num(), expoente.r.Denom()
	if !q.IsInt64() || q.Int64() > LimiteDaRaiz {
		return valor{}, erro("expoente com casas demais")
	}
	indice := int(q.Int64())
	negativo := false
	if base.r.Sign() < 0 && indice > 1 {
		// A raiz de índice ímpar de um negativo existe: (-8)^(1/3) = -2.
		if indice%2 == 0 {
			return valor{}, erro("raiz de número negativo")
		}
		negativo = p.Bit(0) == 1
		base = valor{r: new(big.Rat).Abs(base.r)}
	}
	v, err := elevarInteiro(base.r, p)
	if err != nil {
		return valor{}, err
	}
	v.aprox = aprox
	if indice > 1 {
		v = raizDe(v, indice)
	}
	if negativo {
		v.r.Neg(v.r)
	}
	return cabe(v)
}

func elevarInteiro(base *big.Rat, n *big.Int) (valor, error) {
	num, den := base.Num(), base.Denom()
	absNum := new(big.Int).Abs(num)
	switch {
	case n.Sign() == 0:
		return valor{r: new(big.Rat).Set(um)}, nil // inclusive 0^0
	case num.Sign() == 0:
		if n.Sign() < 0 {
			return valor{}, erro("divisão por zero")
		}
		return valor{r: new(big.Rat)}, nil
	case base.IsInt() && absNum.IsInt64() && absNum.Int64() == 1:
		// 1 e -1 aguentam qualquer expoente.
		if num.Sign() < 0 && n.Bit(0) == 1 {
			return valor{r: big.NewRat(-1, 1)}, nil
		}
		return valor{r: new(big.Rat).Set(um)}, nil
	}
	// Quantos bits o resultado teria, no mínimo: cada multiplicação soma
	// pelo menos (bits - 1) do maior dos dois lados da fração.
	bits := int64(max(absNum.BitLen(), den.BitLen()) - 1)
	vezes := new(big.Int).Abs(n)
	if !vezes.IsInt64() || vezes.Int64() > LimiteDeBits || vezes.Int64()*bits > LimiteDeBits {
		return valor{}, erro("expoente grande demais")
	}
	a := new(big.Int).Exp(num, vezes, nil)
	b := new(big.Int).Exp(den, vezes, nil)
	if n.Sign() < 0 {
		a, b = b, a
		if b.Sign() < 0 {
			a.Neg(a)
			b.Neg(b)
		}
	}
	return cabe(valor{r: new(big.Rat).SetFrac(a, b)})
}

// raizDe tira a raiz de índice k de um valor que não é negativo. É exata
// quando o numerador e o denominador são potências perfeitas; senão sai com
// casasDeTrabalho casas e o valor passa a ser aproximado.
func raizDe(v valor, k int) valor {
	num, den := v.r.Num(), v.r.Denom()
	indice := big.NewInt(int64(k))
	a, b := raizInteira(num, k), raizInteira(den, k)
	if new(big.Int).Exp(a, indice, nil).Cmp(num) == 0 && new(big.Int).Exp(b, indice, nil).Cmp(den) == 0 {
		return valor{r: new(big.Rat).SetFrac(a, b), aprox: v.aprox}
	}
	// raiz(num/den) com T casas = raiz(num × 10^(kT) / den) / 10^T.
	escala := new(big.Int).Mul(num, dezElevado(k*casasDeTrabalho))
	escala.Quo(escala, den)
	return valor{r: new(big.Rat).SetFrac(raizInteira(escala, k), dezElevado(casasDeTrabalho)), aprox: true}
}

// raizInteira devolve o maior inteiro r com r^k <= n (n >= 0, k >= 1), pelo
// método de Newton partindo de cima.
func raizInteira(n *big.Int, k int) *big.Int {
	switch {
	case n.Sign() <= 0:
		return new(big.Int)
	case k == 1:
		return new(big.Int).Set(n)
	case k == 2:
		return new(big.Int).Sqrt(n)
	}
	// Um chute acima da raiz: 2^teto(bits/k) sempre serve; quando cabe num
	// float64, o chute já sai quase em cima e o Newton fecha em poucos passos.
	x := new(big.Int).Lsh(big.NewInt(1), uint((n.BitLen()+k-1)/k))
	if n.BitLen()/k < 1000 {
		mantissa := new(big.Float)
		expoente := new(big.Float).SetInt(n).MantExp(mantissa)
		m, _ := mantissa.Float64()
		chute := math.Pow(2, (math.Log2(m)+float64(expoente))/float64(k)) * (1 + 1e-9)
		if c, _ := new(big.Float).SetFloat64(chute).Int(nil); c != nil {
			if c.Add(c, big.NewInt(2)); c.Cmp(x) < 0 {
				x = c
			}
		}
	}
	k1, kk := big.NewInt(int64(k-1)), big.NewInt(int64(k))
	y, t := new(big.Int), new(big.Int)
	for {
		// y = ((k-1)·x + n / x^(k-1)) / k
		t.Exp(x, k1, nil)
		y.Quo(n, t)
		y.Add(y, t.Mul(x, k1))
		y.Quo(y, kk)
		if y.Cmp(x) >= 0 {
			return x
		}
		x.Set(y)
	}
}
