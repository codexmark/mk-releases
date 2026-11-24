package conta

import (
	"encoding/json"
	"errors"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"
)

// conta é um caso: a expressão, o que se mostra e se é aproximado. O cru é o
// mostrado sem os pontos de milhar (conferido à parte em TestCruEBonito).
type caso struct {
	expr, resultado string
	aprox           bool
}

func confere(t *testing.T, casos []caso) {
	t.Helper()
	for _, c := range casos {
		r, err := Calcular(c.expr)
		if err != nil {
			t.Errorf("%q: erro %v", c.expr, err)
			continue
		}
		if r.Resultado != c.resultado || r.Aproximado != c.aprox {
			t.Errorf("%q: obtido %q (aproximado %v), esperado %q (aproximado %v)", c.expr, r.Resultado, r.Aproximado, c.resultado, c.aprox)
		}
	}
}

func TestAritmeticaExata(t *testing.T) {
	confere(t, []caso{
		{"0,1 + 0,2", "0,3", false},
		{"0.1 + 0.2", "0,3", false},
		{"1 - 0,9", "0,1", false},
		{"2 + 2", "4", false},
		{"7 - 10", "-3", false},
		{"3 × 4", "12", false},
		{"10 ÷ 4", "2,5", false},
		{"1 ÷ 3 × 3", "1", false},
		{"0,1 × 3", "0,3", false},
		{"1,1 × 1,1", "1,21", false},
		{"12345678901234567890 × 98765432109876543210", "1.219.326.311.370.217.952.237.463.801.111.263.526.900", false},
		{"99999999999999999999 + 1", "100.000.000.000.000.000.000", false},
		{"2^100", "1.267.650.600.228.229.401.496.703.205.376", false},
		{"0,1^20", "0,00000000000000000001", false},
		{"1 ÷ 1024", "0,0009765625", false},
		{"1234567,891 + 0", "1.234.567,891", false},
		{"0 × -5", "0", false},
		{"5 - 5", "0", false},
	})
}

func TestPrecedenciaParentesesEMenos(t *testing.T) {
	confere(t, []caso{
		{"2 + 3 × 4", "14", false},
		{"(2 + 3) × 4", "20", false},
		{"2 × 3 + 4", "10", false},
		{"10 - 4 - 3", "3", false},
		{"100 ÷ 10 ÷ 5", "2", false},
		{"2 × 3^2", "18", false},
		{"2^3^2", "512", false},
		{"(2^3)^2", "64", false},
		{"-2^2", "-4", false},
		{"(-2)^2", "4", false},
		{"2^-2", "0,25", false},
		{"2^-1^2", "0,5", false},
		{"-3 + 5", "2", false},
		{"5 - -3", "8", false},
		{"5 + -3", "2", false},
		{"5 × -3", "-15", false},
		{"--3", "3", false},
		{"+3", "3", false},
		{"-(2 + 3)", "-5", false},
		{"((((7))))", "7", false},
		{"2(3 + 4)", "14", false},
		{"(1 + 2)(3 + 4)", "21", false},
		{"2√9", "6", false},
		{"0^0", "1", false},
		{"0^5", "0", false},
		{"1^999999999999999999999999", "1", false},
		{"(-1)^999999999999999999999999", "-1", false},
		{"(-1)^999999999999999999999998", "1", false},
	})
}

func TestSinonimosEEspacos(t *testing.T) {
	confere(t, []caso{
		{"6*7", "42", false},
		{"6x7", "42", false},
		{"6 X 7", "42", false},
		{"6·7", "42", false},
		{"84/2", "42", false},
		{"84:2", "42", false},
		{"2**5", "32", false},
		{"50 − 8", "42", false},
		{"50 – 8", "42", false},
		{"raiz(81)", "9", false},
		{"RAIZ 81", "9", false},
		{"sqrt(81)", "9", false},
		{"  6\t× 7 ", "42", false},
	})
}

func TestPercentualDeCalculadoraDeMesa(t *testing.T) {
	confere(t, []caso{
		{"200 + 10%", "220", false},
		{"200 - 10%", "180", false},
		{"100 + 50 + 10%", "165", false},
		{"200 + 10% + 5%", "231", false},
		{"200 + (5 + 5)%", "220", false},
		{"80 - 12,5%", "70", false},
		{"(200 + 10%) × 2", "440", false},
		// Com ×, ÷, ^ ou sozinho: divide por 100.
		{"200 × 10%", "20", false},
		{"200 ÷ 10%", "2.000", false},
		{"50%", "0,5", false},
		{"10% + 200", "200,1", false},
		{"200 + 10% × 2", "200,2", false},
		{"200 + (10%)", "200,1", false},
		{"2^200%", "4", false},
		{"√16%", "0,04", false},
		{"50%%", "0,005", false},
	})
}

func TestRaizEExpoenteFracionario(t *testing.T) {
	confere(t, []caso{
		{"√16", "4", false},
		{"√0,25", "0,5", false},
		{"√(4 ÷ 9) × 3", "2", false},
		{"√√81", "3", false},
		{"√0", "0", false},
		{"√2", "1,414213562373", true},
		{"√2 × √2", "2", true},
		{"√2 × √2 - 2", "0", true},
		{"√1000000000000000000000000000000000000000000", "1.000.000.000.000.000.000.000", false},
		{"8^(1 ÷ 3)", "2", false},
		{"(-8)^(1 ÷ 3)", "-2", false},
		{"(-8)^(2 ÷ 3)", "4", false},
		{"32^0,2", "2", false},
		{"4^1,5", "8", false},
		{"4^-0,5", "0,5", false},
		{"2^0,5", "1,414213562373", true},
		{"2^(1 ÷ 3)", "1,259921049895", true},
		{"2^0,001", "1,000693387463", true},
		{"10^0,5", "3,162277660168", true},
	})
}

func TestArredondamentoEAproximado(t *testing.T) {
	confere(t, []caso{
		{"1 ÷ 3", "0,333333333333", true},
		{"2 ÷ 3", "0,666666666667", true},
		{"-2 ÷ 3", "-0,666666666667", true},
		{"10 ÷ 3", "3,333333333333", true},
		{"1000000 ÷ 7", "142.857,142857142857", true},
		// Abaixo de 1 as 12 casas contam do primeiro algarismo que não é zero.
		{"1 ÷ 3000", "0,000333333333333", true},
		{"1 ÷ 30000000000000000000", "0,0000000000000000000333333333333", true},
		{"2 ÷ 3 - 1", "-0,333333333333", true},
		// 0,9999… arredonda para cima e muda de casa.
		{"1 - 1 ÷ 3000000000000000", "1", true},
		// Decimal finito: inteiro até 40 casas, arredondado depois disso.
		{"1 ÷ 2^40", "0,0000000000009094947017729282379150390625", false},
		{"1 ÷ 2^50", "0,0000000000000008881784197", true},
	})
}

func TestCruEBonito(t *testing.T) {
	for _, c := range []struct{ expr, bonito, cru string }{
		{"1234567,891 + 0", "1.234.567,891", "1234567,891"},
		{"-1234,5 × 1000", "-1.234.500", "-1234500"},
		{"999 + 1", "1.000", "1000"},
		{"100 × 1", "100", "100"},
		{"1 ÷ 8", "0,125", "0,125"},
		{"0,0001234567 + 0", "0,0001234567", "0,0001234567"},
		{"-0,5 × 1", "-0,5", "-0,5"},
		{"0 × -1", "0", "0"},
	} {
		r, err := Calcular(c.expr)
		if err != nil || r.Resultado != c.bonito || r.Cru != c.cru {
			t.Errorf("%q: obtido %q / %q (%v), esperado %q / %q", c.expr, r.Resultado, r.Cru, err, c.bonito, c.cru)
		}
	}
}

func TestAExpressaoArrumadaELidaDeVoltaIgual(t *testing.T) {
	for _, c := range []struct{ expr, arrumada string }{
		{"2+2", "2 + 2"},
		{"1.234,50*2", "1234,5 × 2"},
		{"1,234,567.8/2", "1234567,8 ÷ 2"},
		{"-3--4", "-3 - -4"},
		{"+3", "3"},
		{"2^-3", "2^-3"},
		{"2**3**2", "2^3^2"},
		{"raiz(9)x2", "√(9) × 2"},
		{"2(3+4)%", "2 × (3 + 4)%"},
		{"(1+2)√9", "(1 + 2) × √9"},
		{"200 + 10 %", "200 + 10%"},
		{"√ - 4 + 5", "√-4 + 5"},
		{"007 : ,5", "7 ÷ 0,5"},
	} {
		r, err := Calcular(c.expr)
		var e *Erro
		if err != nil && !errors.As(err, &e) {
			t.Fatalf("%q: %v", c.expr, err)
		}
		if err != nil {
			// A raiz de negativo não sai, mas a arrumação é conferida pelas fichas.
			fichas, _ := separar(c.expr)
			if got := arrumar(fichas); got != c.arrumada {
				t.Errorf("%q: arrumada %q, esperado %q", c.expr, got, c.arrumada)
			}
			continue
		}
		if r.Expressao != c.arrumada {
			t.Errorf("%q: arrumada %q, esperado %q", c.expr, r.Expressao, c.arrumada)
		}
		de, err := Calcular(r.Expressao)
		if err != nil || de != r {
			t.Errorf("%q: a arrumada %q não dá o mesmo: %+v (%v), esperado %+v", c.expr, r.Expressao, de, err, r)
		}
	}
}

func TestErros(t *testing.T) {
	for _, c := range []struct {
		expr, frase string
		incompleta  bool
	}{
		{"", "expressão vazia", true},
		{"   ", "expressão vazia", true},
		{"2 +", "expressão incompleta", true},
		{"2 ×", "expressão incompleta", true},
		{"-", "expressão incompleta", true},
		{"√", "expressão incompleta", true},
		{"2^", "expressão incompleta", true},
		{"()", "expressão incompleta", true},
		{"(2 +)", "expressão incompleta", true},
		{"(", "expressão incompleta", true},
		{"(3 × 4", "parêntese sem par", true},
		{"((1 + 2)", "parêntese sem par", true},
		{"3 × 4)", "parêntese sem par", false},
		{")", "parêntese sem par", false},
		{"2 + )", "parêntese sem par", false},
		{"1 ÷ 0", "divisão por zero", false},
		{"1 ÷ (2 - 2)", "divisão por zero", false},
		{"5 ÷ 0%", "divisão por zero", false},
		{"0^-1", "divisão por zero", false},
		{"0^-0,5", "divisão por zero", false},
		{"√-4", "raiz de número negativo", false},
		{"√(1 - 2)", "raiz de número negativo", false},
		{"(-8)^0,5", "raiz de número negativo", false},
		{"1.2.3", "número mal formado: «1.2.3»", false},
		{"2 + 1,,5", "número mal formado: «1,,5»", false},
		{"2 + × 3", "operador fora de lugar: «×»", false},
		{"× 3", "operador fora de lugar: «×»", false},
		{"%", "operador fora de lugar: «%»", false},
		{"2 3", "falta um operador antes de «3»", false},
		{"(1 + 2) 3", "falta um operador antes de «3»", false},
		{"(2 3)", "falta um operador antes de «3»", false},
		{"50% 2", "falta um operador antes de «2»", false},
		{"abc", "não entendi «abc»", false},
		{"2 $ 3", "não entendi «$»", false},
		{"2xx3", "não entendi «xx»", false},
		{"9^9^9", "expoente grande demais", false},
		{"2^99999999999999999999999999", "expoente grande demais", false},
		{"0,5^99999999", "expoente grande demais", false},
		{"2^-99999999", "expoente grande demais", false},
		{"10^10^10", "expoente grande demais", false},
		{"2^10240", "número grande demais", false},
		{"2^0,0001", "expoente com casas demais", false},
		{"2^(1 ÷ 1001)", "expoente com casas demais", false},
		{strings.Repeat("1+", 2500) + "1", "expressão longa demais", false},
	} {
		_, err := Calcular(c.expr)
		var e *Erro
		if !errors.As(err, &e) {
			t.Errorf("%.40q: devia ser um *Erro, veio %v", c.expr, err)
			continue
		}
		if e.Frase != c.frase || e.Incompleta != c.incompleta {
			t.Errorf("%.40q: obtido %q (incompleta %v), esperado %q (incompleta %v)", c.expr, e.Frase, e.Incompleta, c.frase, c.incompleta)
		}
	}
}

func TestOsLimitesNaoDeixamTravar(t *testing.T) {
	// Nenhuma destas pode demorar: a janela pede a prévia a cada tecla.
	for _, expr := range []string{
		"9^9^9", "9^9^9^9^9", "(9^9)^(9^9)", "2^2^2^2^2^2", "0,1^9^9", "7^(999 ÷ 1000)", "2^10000 × 2^10000",
		"(1 ÷ 3)^20000", "√(2^10001)", "(2^10000)^(1 ÷ 999)", strings.Repeat("(", 400) + "1" + strings.Repeat(")", 400),
		strings.Repeat("9", 4999), strings.Repeat("-", 4999) + "1", strings.Repeat("√", 1500) + "2", strings.Repeat("2^0,001+", 499) + "1",
		strings.Repeat("(", 1900) + "1" + strings.Repeat(")", 1900), strings.Repeat("7^(999÷1000)×", 300) + "1",
		"1" + strings.Repeat("×99999999", 110),
	} {
		inicio := time.Now()
		Calcular(expr)
		if d := time.Since(inicio); d > 2*time.Second {
			t.Errorf("%.30q levou %v", expr, d)
		}
	}
	// O tamanho do que cabe: 2^10000 sai inteiro, com os 3011 algarismos.
	r, err := Calcular("2^10000")
	if err != nil || r.Aproximado || len(r.Cru) != 3011 || !strings.HasPrefix(r.Cru, "19950631168807583848837421626835850838") {
		t.Errorf("2^10000: %v, %d algarismos", err, len(r.Cru))
	}
	if _, err := Calcular(strings.Repeat("1+", LimiteDoTexto/2-1) + "1"); err != nil {
		t.Errorf("uma expressão no limite de tamanho devia passar: %v", err)
	}
}

func TestLerNumeroPelosCasosDoContrato(t *testing.T) {
	dados, err := os.ReadFile("testdata/numeros.json")
	if err != nil {
		t.Fatal(err)
	}
	var casos []struct {
		Texto string  `json:"texto"`
		Le    *string `json:"le"`
	}
	if err := json.Unmarshal(dados, &casos); err != nil {
		t.Fatal(err)
	}
	if len(casos) < 30 {
		t.Fatalf("só %d casos", len(casos))
	}
	for _, c := range casos {
		n, ok := LerNumero(c.Texto)
		switch {
		case c.Le == nil && ok:
			t.Errorf("%q devia ser recusado, leu %s", c.Texto, exato(n))
		case c.Le != nil && !ok:
			t.Errorf("%q devia ler %s, foi recusado", c.Texto, *c.Le)
		case c.Le != nil && exato(n) != *c.Le:
			t.Errorf("%q: leu %s, esperado %s", c.Texto, exato(n), *c.Le)
		}
	}
}

func TestRaizInteira(t *testing.T) {
	semente := uint64(12345)
	proximo := func() uint64 {
		semente = semente*6364136223846793005 + 1442695040888963407
		return semente >> 11
	}
	for i := 0; i < 400; i++ {
		k := int(proximo()%60) + 1
		n := new(big.Int).SetUint64(proximo())
		n.Exp(n, big.NewInt(int64(proximo()%9)), nil)
		n.Add(n, new(big.Int).SetUint64(proximo()%1000))
		r := raizInteira(n, k)
		kk := big.NewInt(int64(k))
		baixo := new(big.Int).Exp(r, kk, nil)
		cima := new(big.Int).Exp(new(big.Int).Add(r, big.NewInt(1)), kk, nil)
		if baixo.Cmp(n) > 0 || cima.Cmp(n) <= 0 {
			t.Fatalf("raizInteira(%v, %d) = %v", n, k, r)
		}
	}
	for _, c := range []struct {
		n    string
		k    int
		raiz string
	}{{"0", 5, "0"}, {"1", 1000, "1"}, {"1024", 10, "2"}, {"1023", 10, "1"}, {"59049", 10, "3"}, {"1000000000000000000000000000000", 3, "10000000000"}} {
		n, _ := new(big.Int).SetString(c.n, 10)
		if r := raizInteira(n, c.k).String(); r != c.raiz {
			t.Errorf("raizInteira(%s, %d) = %s, esperado %s", c.n, c.k, r, c.raiz)
		}
	}
	// Um número enorme, de raiz que não cabe num float64.
	n := new(big.Int).Exp(big.NewInt(7), big.NewInt(9000), nil)
	if r := raizInteira(n, 3); r.Cmp(new(big.Int).Exp(big.NewInt(7), big.NewInt(3000), nil)) != 0 {
		t.Errorf("raiz cúbica de 7^9000 errada")
	}
}

func TestNenhumaExpressaoDerrubaEAArrumadaDaOMesmo(t *testing.T) {
	// Expressões ao acaso, com o alfabeto da calculadora: nenhuma pode dar
	// pânico nem demorar, e a que sai tem de sair igual quando o motor lê de
	// volta a expressão que ele mesmo arrumou.
	alfabeto := []rune("0123456789012345,.+-×÷^%()√ x/*")
	semente := uint64(2026)
	proximo := func(n int) int {
		semente = semente*6364136223846793005 + 1442695040888963407
		return int((semente >> 33) % uint64(n))
	}
	inicio := time.Now()
	certas := 0
	for i := 0; i < 30000; i++ {
		rs := make([]rune, 1+proximo(14))
		for j := range rs {
			rs[j] = alfabeto[proximo(len(alfabeto))]
		}
		expr := string(rs)
		r, err := Calcular(expr)
		if err != nil {
			var e *Erro
			if !errors.As(err, &e) || e.Frase == "" {
				t.Fatalf("%q: erro que não é da conta: %v", expr, err)
			}
			continue
		}
		certas++
		if r.Resultado == "" || r.Cru == "" || strings.ContainsAny(r.Cru, ". ") || strings.Contains(r.Expressao, ".") {
			t.Fatalf("%q: resposta torta: %+v", expr, r)
		}
		if de, err := Calcular(r.Expressao); err != nil || de != r {
			t.Fatalf("%q: a arrumada %q deu %+v (%v), esperado %+v", expr, r.Expressao, de, err, r)
		}
		// O cru, lido de volta, é o mesmo número.
		if de, err := Calcular(r.Cru); err != nil || de.Cru != r.Cru {
			t.Fatalf("%q: o cru %q lido de volta deu %+v (%v)", expr, r.Cru, de, err)
		}
	}
	if certas < 3000 {
		t.Errorf("só %d expressões válidas em 30000: o teste não está testando", certas)
	}
	if d := time.Since(inicio); d > 20*time.Second {
		t.Errorf("30000 expressões levaram %v", d)
	}
}

func TestOMaiorEOMenorResultadoVoltamParaAExpressao(t *testing.T) {
	// O que o motor responde tem de poder ser lido de volta (é como a janela
	// continua a conta), inclusive nos extremos do que cabe.
	for _, expr := range []string{"2^10239", "0 - 2^10239", "1 ÷ 2^10239", "1 ÷ 3^6460", "(2^10239 - 1) ÷ 2^40", "5^4409", "1 ÷ 5^4409", "(1 ÷ 3)^6000 × 2"} {
		r, err := Calcular(expr)
		if err != nil {
			t.Errorf("%s: %v", expr, err)
			continue
		}
		if len(r.Cru) < 1000 || len(r.Cru) > 3200 {
			t.Errorf("%s: o cru tem %d caracteres", expr, len(r.Cru))
		}
		de, err := Calcular(r.Cru + " + 0")
		if err != nil || de.Cru != r.Cru {
			t.Errorf("%s: o cru (%d caracteres) não volta: %v", expr, len(r.Cru), err)
		}
		if de, err = Calcular(r.Resultado); err != nil || de.Cru != r.Cru {
			t.Errorf("%s: o resultado mostrado não volta: %v", expr, err)
		}
	}
	if casasMaximas != 3082 {
		t.Errorf("casasMaximas = %d", casasMaximas)
	}
}
