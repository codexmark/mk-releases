// Package conta é a calculadora do MkCalc: lê uma expressão, faz a conta com
// frações exatas (math/big) e escreve o resultado em pt-BR. O que ela aceita,
// o percentual, a regra da vírgula e do ponto, o arredondamento e os limites
// estão em ../../../CONTRATO.md: mude lá primeiro.
package conta

import (
	"math/big"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Os números do contrato.
const (
	// CasasExatas: um decimal finito com até tantas casas sai inteiro.
	CasasExatas = 40
	// CasasAproximadas: o resto é arredondado para tantas casas.
	CasasAproximadas = 12
	// LimiteDeBits: nenhum número da conta passa disso (cerca de 3.000 algarismos).
	LimiteDeBits = 10240
	// LimiteDoTexto: o tamanho máximo de uma expressão, em caracteres.
	LimiteDoTexto = 5000
	// LimiteDaRaiz: o maior denominador de um expoente fracionário.
	LimiteDaRaiz = 1000

	// Quantas casas decimais cabem em LimiteDeBits (10^casas < 2^bits).
	casasMaximas = LimiteDeBits * 30102 / 100000

	casasDeTrabalho = 40 // com quantas casas sai uma raiz que não é exata
	casasDoRuido    = 30 // abaixo de 10^-30, numa conta aproximada, é zero
)

// Erro é uma conta que não saiu. Incompleta diz que a expressão só não
// terminou de ser escrita: a janela espera a próxima tecla em vez de reclamar.
type Erro struct {
	Frase      string
	Incompleta bool
}

func (e *Erro) Error() string { return e.Frase }

func erro(frase string) error       { return &Erro{Frase: frase} }
func incompleta(frase string) error { return &Erro{Frase: frase, Incompleta: true} }

// Resultado é a resposta de uma conta, com os nomes do contrato.
type Resultado struct {
	Expressao  string `json:"expressao"`
	Resultado  string `json:"resultado"`
	Cru        string `json:"cru"`
	Aproximado bool   `json:"aproximado"`
}

// Calcular lê a expressão e devolve o resultado, ou um *Erro.
func Calcular(texto string) (Resultado, error) {
	if utf8.RuneCountInString(texto) > LimiteDoTexto {
		return Resultado{}, erro("expressão longa demais")
	}
	fichas, err := separar(texto)
	if err != nil {
		return Resultado{}, err
	}
	if len(fichas) == 0 {
		return Resultado{}, incompleta("expressão vazia")
	}
	arvore, err := (&leitor{fichas: fichas}).tudo()
	if err != nil {
		return Resultado{}, err
	}
	v, err := arvore.valer()
	if err != nil {
		return Resultado{}, err
	}
	bonito, cru, aproximado := formatar(v)
	return Resultado{Expressao: arrumar(fichas), Resultado: bonito, Cru: cru, Aproximado: aproximado}, nil
}

// --- as fichas ---------------------------------------------------------------

type tipo int

const (
	fNumero tipo = iota
	fOperador
	fAbre
	fFecha
	fPorcento
	fRaiz
)

// Uma ficha da expressão. op é um de + - × ÷ ^ (já sem os sinônimos).
type ficha struct {
	tipo   tipo
	op     rune
	numero *big.Rat
	texto  string // como apareceu, para as frases de erro
}

var sinonimos = map[rune]rune{
	'+': '+', '-': '-', '−': '-', '–': '-',
	'×': '×', '*': '×', '·': '×',
	'÷': '÷', '/': '÷', ':': '÷',
	'^': '^',
}

func separar(texto string) ([]ficha, error) {
	var fichas []ficha
	rs := []rune(texto)
	for i := 0; i < len(rs); {
		c := rs[i]
		switch {
		case unicode.IsSpace(c):
			i++
		case (c >= '0' && c <= '9') || c == '.' || c == ',':
			j := i
			for j < len(rs) && ((rs[j] >= '0' && rs[j] <= '9') || rs[j] == '.' || rs[j] == ',') {
				j++
			}
			escrito := string(rs[i:j])
			n, ok := LerNumero(escrito)
			if !ok {
				return nil, erro("número mal formado: «" + escrito + "»")
			}
			fichas = append(fichas, ficha{tipo: fNumero, numero: n, texto: escrito})
			i = j
		case c == '*' && i+1 < len(rs) && rs[i+1] == '*':
			fichas = append(fichas, ficha{tipo: fOperador, op: '^', texto: "^"})
			i += 2
		case sinonimos[c] != 0:
			fichas = append(fichas, ficha{tipo: fOperador, op: sinonimos[c], texto: string(sinonimos[c])})
			i++
		case c == '(':
			fichas = append(fichas, ficha{tipo: fAbre, texto: "("})
			i++
		case c == ')':
			fichas = append(fichas, ficha{tipo: fFecha, texto: ")"})
			i++
		case c == '%':
			fichas = append(fichas, ficha{tipo: fPorcento, texto: "%"})
			i++
		case c == '√':
			fichas = append(fichas, ficha{tipo: fRaiz, texto: "√"})
			i++
		case unicode.IsLetter(c):
			j := i
			for j < len(rs) && unicode.IsLetter(rs[j]) {
				j++
			}
			palavra := string(rs[i:j])
			switch strings.ToLower(palavra) {
			case "x":
				fichas = append(fichas, ficha{tipo: fOperador, op: '×', texto: "×"})
			case "raiz", "sqrt":
				fichas = append(fichas, ficha{tipo: fRaiz, texto: "√"})
			default:
				return nil, erro("não entendi «" + palavra + "»")
			}
			i = j
		default:
			return nil, erro("não entendi «" + string(c) + "»")
		}
	}
	return fichas, nil
}

// naFrente diz se o + ou - da posição i é um sinal (vem na frente de um
// número) e não uma conta entre dois.
func naFrente(fichas []ficha, i int) bool {
	if i == 0 {
		return true
	}
	switch fichas[i-1].tipo {
	case fOperador, fAbre, fRaiz:
		return true
	}
	return false
}

// arrumar reescreve a expressão do jeito do contrato: um espaço de cada lado
// de + - × ÷ (o ^ fica colado), vírgula decimal, sem milhar, e o × que estava
// subentendido (2(3) vira 2 × (3)).
func arrumar(fichas []ficha) string {
	var b strings.Builder
	for i, f := range fichas {
		switch f.tipo {
		case fNumero:
			b.WriteString(exato(f.numero))
		case fOperador:
			switch {
			case f.op == '^':
				b.WriteString("^")
			case !naFrente(fichas, i):
				b.WriteString(" " + string(f.op) + " ")
			case f.op == '-':
				b.WriteString("-")
			}
		case fAbre, fRaiz:
			if i > 0 && comecaOutro(fichas[i-1]) {
				b.WriteString(" × ")
			}
			b.WriteString(f.texto)
		default:
			b.WriteString(f.texto)
		}
	}
	return b.String()
}

// comecaOutro: depois desta ficha, um ( ou um √ é uma multiplicação.
func comecaOutro(f ficha) bool {
	return f.tipo == fNumero || f.tipo == fFecha || f.tipo == fPorcento
}

// --- a leitura ---------------------------------------------------------------

type leitor struct {
	fichas  []ficha
	i       int
	abertos int // quantos ( ainda esperam o )
}

func (l *leitor) olha() *ficha {
	if l.i < len(l.fichas) {
		return &l.fichas[l.i]
	}
	return nil
}

func (l *leitor) eOperador(ops string) bool {
	f := l.olha()
	return f != nil && f.tipo == fOperador && strings.ContainsRune(ops, f.op)
}

func (l *leitor) tudo() (no, error) {
	n, err := l.soma()
	if err != nil {
		return nil, err
	}
	if f := l.olha(); f != nil {
		if f.tipo == fFecha {
			return nil, erro("parêntese sem par")
		}
		return nil, erro("falta um operador antes de «" + f.texto + "»")
	}
	return n, nil
}

func (l *leitor) soma() (no, error) {
	esq, err := l.produto()
	if err != nil {
		return nil, err
	}
	for l.eOperador("+-") {
		op := l.olha().op
		l.i++
		dir, err := l.produto()
		if err != nil {
			return nil, err
		}
		esq = &dupla{op: op, esq: esq, dir: dir}
	}
	return esq, nil
}

func (l *leitor) produto() (no, error) {
	esq, err := l.sinal()
	if err != nil {
		return nil, err
	}
	for {
		op := '×'
		switch f := l.olha(); {
		case l.eOperador("×÷"):
			op = f.op
			l.i++
		case f != nil && (f.tipo == fAbre || f.tipo == fRaiz):
			// 2(3 + 4) e 2√9: o × está subentendido.
		default:
			return esq, nil
		}
		dir, err := l.sinal()
		if err != nil {
			return nil, err
		}
		esq = &dupla{op: op, esq: esq, dir: dir}
	}
}

func (l *leitor) sinal() (no, error) {
	if l.eOperador("+-") {
		op := l.olha().op
		l.i++
		n, err := l.sinal()
		if err != nil {
			return nil, err
		}
		if op == '-' {
			return &menos{n}, nil
		}
		return n, nil
	}
	return l.potencia()
}

func (l *leitor) potencia() (no, error) {
	base, err := l.sufixo()
	if err != nil {
		return nil, err
	}
	if !l.eOperador("^") {
		return base, nil
	}
	l.i++
	// O expoente lê o próprio sinal e a própria potência: 2^-3 e 2^3^2 = 2^9.
	expoente, err := l.sinal()
	if err != nil {
		return nil, err
	}
	return &dupla{op: '^', esq: base, dir: expoente}, nil
}

func (l *leitor) sufixo() (no, error) {
	n, err := l.simples()
	if err != nil {
		return nil, err
	}
	for f := l.olha(); f != nil && f.tipo == fPorcento; f = l.olha() {
		l.i++
		n = &porcento{n}
	}
	return n, nil
}

func (l *leitor) simples() (no, error) {
	f := l.olha()
	if f == nil {
		return nil, incompleta("expressão incompleta")
	}
	switch f.tipo {
	case fNumero:
		l.i++
		return &numero{f.numero}, nil
	case fAbre:
		l.i++
		l.abertos++
		dentro, err := l.soma()
		if err != nil {
			return nil, err
		}
		fim := l.olha()
		if fim == nil {
			return nil, incompleta("parêntese sem par")
		}
		if fim.tipo != fFecha {
			return nil, erro("falta um operador antes de «" + fim.texto + "»")
		}
		l.i++
		l.abertos--
		return &grupo{dentro}, nil
	case fRaiz:
		l.i++
		// √-4 é lido (para dizer "raiz de número negativo"), mas só o sinal.
		if l.eOperador("+-") {
			op := l.olha().op
			l.i++
			n, err := l.simples()
			if err != nil {
				return nil, err
			}
			if op == '-' {
				n = &menos{n}
			}
			return &raiz{n}, nil
		}
		n, err := l.simples()
		if err != nil {
			return nil, err
		}
		return &raiz{n}, nil
	case fFecha:
		if l.abertos == 0 {
			return nil, erro("parêntese sem par")
		}
		// "()" e "(2 +)": o que a janela manda enquanto a pessoa ainda escreve.
		return nil, incompleta("expressão incompleta")
	}
	return nil, erro("operador fora de lugar: «" + f.texto + "»")
}
