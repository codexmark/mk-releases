// SPDX-License-Identifier: GPL-3.0-or-later

package dominio

import "math/rand/v2"

// Repetir é o modo de repetição da fila.
type Repetir string

const (
	RepetirNao  Repetir = "OFF"
	RepetirTudo Repetir = "ALL"
	RepetirUma  Repetir = "ONE"
)

// RepetirDe lê o nome guardado de um modo de repetição.
func RepetirDe(nome string) (Repetir, bool) {
	switch r := Repetir(nome); r {
	case RepetirNao, RepetirTudo, RepetirUma:
		return r, true
	}
	return "", false
}

// Retrato é o que basta para refazer uma fila depois de reiniciar. Indice
// negativo quer dizer "nada tocando".
type Retrato struct {
	Faixas  []Faixa
	Indice  int
	Repetir Repetir
}

// Fila é a ordem de tocar, imutável: cada operação devolve uma fila nova e
// nunca mexe nas faixas da antiga.
type Fila struct {
	faixas         []Faixa
	pos            int // -1: nada tocando
	repetir        Repetir
	pararAposAtual bool
}

// FilaVazia não tem nada tocando nem a seguir.
func FilaVazia() Fila { return Fila{pos: -1, repetir: RepetirNao} }

// NovaFila começa a tocar em comecaEm; fora da lista, a fila vem vazia.
func NovaFila(faixas []Faixa, comecaEm int) Fila {
	if len(faixas) == 0 || comecaEm < 0 || comecaEm >= len(faixas) {
		return FilaVazia()
	}
	return Fila{faixas: copia(faixas), pos: comecaEm, repetir: RepetirNao}
}

// Restaura refaz uma fila guardada; o que estiver inconsistente volta vazio.
func Restaura(r Retrato) Fila {
	if r.Indice < 0 || r.Indice >= len(r.Faixas) {
		return FilaVazia()
	}
	rep := r.Repetir
	if _, ok := RepetirDe(string(rep)); !ok {
		rep = RepetirNao
	}
	return Fila{faixas: copia(r.Faixas), pos: r.Indice, repetir: rep}
}

func copia(f []Faixa) []Faixa { return append([]Faixa(nil), f...) }

// Atual é a faixa que toca, se há uma.
func (q Fila) Atual() (Faixa, bool) {
	if q.pos < 0 {
		return Faixa{}, false
	}
	return q.faixas[q.pos], true
}

// ASeguir são as faixas depois da atual.
func (q Fila) ASeguir() []Faixa {
	if q.pos < 0 {
		return nil
	}
	return copia(q.faixas[q.pos+1:])
}

// Repetir é o modo de repetição.
func (q Fila) Repetir() Repetir { return q.repetir }

// PararAposAtual diz se a fila acaba quando a faixa atual terminar.
func (q Fila) PararAposAtual() bool { return q.pararAposAtual }

// Ultima é a faixa do fim da fila: a semente natural do autoplay.
func (q Fila) Ultima() (Faixa, bool) {
	if len(q.faixas) == 0 {
		return Faixa{}, false
	}
	return q.faixas[len(q.faixas)-1], true
}

// Contem diz se a fila já tem essa faixa, tocada ou não.
func (q Fila) Contem(id string) bool {
	for _, f := range q.faixas {
		if f.ID == id {
			return true
		}
	}
	return false
}

// Retrato tira a foto que o Restaura entende.
func (q Fila) Retrato() Retrato {
	return Retrato{Faixas: copia(q.faixas), Indice: q.pos, Repetir: q.repetir}
}

// Proxima: o ouvinte apertou "próxima"; sempre sai da faixa atual.
func (q Fila) Proxima() Fila { return q.avanca() }

// AoTerminar: a faixa atual acabou sozinha.
func (q Fila) AoTerminar() Fila {
	switch {
	case q.pos < 0:
		return q
	case q.pararAposAtual:
		q.pos, q.pararAposAtual = -1, false
		return q
	case q.repetir == RepetirUma:
		return q
	}
	return q.avanca()
}

// Anterior volta uma faixa; na primeira, fica nela.
func (q Fila) Anterior() Fila {
	if q.pos > 0 {
		q.pos--
	}
	return q
}

// TocaASeguir põe a faixa logo depois da atual.
func (q Fila) TocaASeguir(f Faixa) Fila { return q.InsereASeguir(0, f) }

// InsereASeguir põe a faixa na posição indice da lista "a seguir".
func (q Fila) InsereASeguir(indice int, f Faixa) Fila {
	if q.pos < 0 {
		return q.comecaCom(f)
	}
	lugar := min(q.pos+1+max(indice, 0), len(q.faixas))
	novo := make([]Faixa, 0, len(q.faixas)+1)
	novo = append(novo, q.faixas[:lugar]...)
	novo = append(novo, f)
	q.faixas = append(novo, q.faixas[lugar:]...)
	return q
}

// Enfileira põe a faixa no fim; numa fila parada, começa a tocar por ela.
func (q Fila) Enfileira(f Faixa) Fila {
	if q.pos < 0 {
		return q.comecaCom(f)
	}
	q.faixas = append(copia(q.faixas), f)
	return q
}

// EnfileiraTodas põe várias no fim, na ordem.
func (q Fila) EnfileiraTodas(mais []Faixa) Fila {
	if len(mais) == 0 {
		return q
	}
	if q.pos < 0 {
		q.faixas, q.pos = copia(mais), 0
		return q
	}
	q.faixas = append(copia(q.faixas), mais...)
	return q
}

// PulaPara: o ouvinte escolheu uma faixa da lista "a seguir".
func (q Fila) PulaPara(indice int) Fila {
	if q.pos < 0 {
		return q
	}
	if alvo := q.pos + 1 + indice; indice >= 0 && alvo < len(q.faixas) {
		q.pos = alvo
	}
	return q
}

// CicloRepetir passa de não repetir para tudo, uma, e de volta.
func (q Fila) CicloRepetir() Fila {
	switch q.repetir {
	case RepetirNao:
		q.repetir = RepetirTudo
	case RepetirTudo:
		q.repetir = RepetirUma
	default:
		q.repetir = RepetirNao
	}
	return q
}

// ComRepetir troca o modo de repetição.
func (q Fila) ComRepetir(modo Repetir) Fila { q.repetir = modo; return q }

// ComPararAposAtual liga ou desliga o "parar depois desta".
func (q Fila) ComPararAposAtual(ligado bool) Fila { q.pararAposAtual = ligado; return q }

// RemoveASeguir tira uma faixa da lista "a seguir".
func (q Fila) RemoveASeguir(indice int) Fila {
	if q.pos < 0 || indice < 0 || q.pos+1+indice >= len(q.faixas) {
		return q
	}
	i := q.pos + 1 + indice
	novo := copia(q.faixas[:i])
	q.faixas = append(novo, q.faixas[i+1:]...)
	return q
}

// MoveASeguir troca uma faixa de lugar dentro da lista "a seguir".
func (q Fila) MoveASeguir(de, para int) Fila {
	seguir := q.ASeguir()
	if q.pos < 0 || de < 0 || para < 0 || de >= len(seguir) || para >= len(seguir) {
		return q
	}
	seguir = Move(seguir, de, para)
	q.faixas = append(copia(q.faixas[:q.pos+1]), seguir...)
	return q
}

// LimpaASeguir deixa só o que já tocou e a faixa atual.
func (q Fila) LimpaASeguir() Fila {
	if q.pos < 0 {
		return q
	}
	q.faixas = copia(q.faixas[:q.pos+1])
	return q
}

// Embaralha é quem mistura: recebe quantos itens e a função que troca dois.
type Embaralha func(n int, troca func(i, j int))

// Embaralhada mistura a fila mantendo a faixa atual tocando, na frente.
// Com mistura nula usa o acaso de verdade.
func (q Fila) Embaralhada(mistura Embaralha) Fila {
	if mistura == nil {
		mistura = rand.Shuffle
	}
	if q.pos < 0 {
		novo := copia(q.faixas)
		mistura(len(novo), func(i, j int) { novo[i], novo[j] = novo[j], novo[i] })
		q.faixas = novo
		return q
	}
	outras := make([]Faixa, 0, len(q.faixas))
	for i, f := range q.faixas {
		if i != q.pos {
			outras = append(outras, f)
		}
	}
	mistura(len(outras), func(i, j int) { outras[i], outras[j] = outras[j], outras[i] })
	q.faixas = append([]Faixa{q.faixas[q.pos]}, outras...)
	q.pos = 0
	return q
}

func (q Fila) avanca() Fila {
	if q.pos < 0 {
		return q
	}
	switch prox := q.pos + 1; {
	case prox < len(q.faixas):
		q.pos = prox
	case q.repetir == RepetirTudo && len(q.faixas) > 0:
		q.pos = 0
	default:
		q.pos = -1
	}
	return q
}

// comecaCom só é chamado com a fila parada: a faixa vira a fila inteira.
func (q Fila) comecaCom(f Faixa) Fila {
	q.faixas, q.pos = []Faixa{f}, 0
	return q
}

// Move tira o item de "de" e o põe em "para", numa cópia.
func Move[T any](lista []T, de, para int) []T {
	novo := append([]T(nil), lista...)
	item := novo[de]
	novo = append(novo[:de], novo[de+1:]...)
	novo = append(novo[:para], append([]T{item}, novo[para:]...)...)
	return novo
}
