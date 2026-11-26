// SPDX-License-Identifier: GPL-3.0-or-later

package dominio

import (
	"math"
	"sort"
	"strings"
	"time"
)

// Tocada é uma vez em que o ouvinte ouviu mesmo uma faixa.
type Tocada struct {
	Faixa Faixa
	Em    time.Time
}

// Ouvida diz quando uma tocada conta: 30 s de música, ou a metade de uma
// faixa mais curta.
func Ouvida(posicao, duracao time.Duration) bool {
	preciso := 30 * time.Second
	if duracao > 0 && duracao/2 < preciso {
		preciso = duracao / 2
	}
	return posicao >= preciso
}

const dia = 24 * time.Hour

// JanelaHistorico é até onde o histórico é guardado e o "No repeat" olha.
const JanelaHistorico = 30 * dia

const (
	janelaRecentes    = 7 * dia
	redescobrirDepois = 14 * dia
	tamanhoNoRepeat   = 20
	tamanhoLista      = 40
)

type resumo struct {
	faixa   Faixa
	vezes   int
	ultima  time.Time
	chegada int
}

// resume dá uma entrada por faixa, com os detalhes da tocada mais recente, na
// ordem em que cada faixa apareceu pela primeira vez.
func resume(tocadas []Tocada) []resumo {
	onde := map[string]int{}
	var lista []resumo
	for _, t := range tocadas {
		i, visto := onde[t.Faixa.ID]
		if !visto {
			onde[t.Faixa.ID] = len(lista)
			lista = append(lista, resumo{faixa: t.Faixa, vezes: 1, ultima: t.Em, chegada: len(lista)})
			continue
		}
		lista[i].vezes++
		if t.Em.After(lista[i].ultima) {
			lista[i].ultima, lista[i].faixa = t.Em, t.Faixa
		}
	}
	return lista
}

func desde(tocadas []Tocada, inicio time.Time) []Tocada {
	var dentro []Tocada
	for _, t := range tocadas {
		if !t.Em.Before(inicio) {
			dentro = append(dentro, t)
		}
	}
	return dentro
}

func faixasDe(lista []resumo, limite int) []Faixa {
	if len(lista) > limite {
		lista = lista[:limite]
	}
	faixas := make([]Faixa, 0, len(lista))
	for _, r := range lista {
		faixas = append(faixas, r.faixa)
	}
	return faixas
}

func porVezes(lista []resumo) {
	sort.SliceStable(lista, func(i, j int) bool {
		if lista[i].vezes != lista[j].vezes {
			return lista[i].vezes > lista[j].vezes
		}
		return lista[i].ultima.After(lista[j].ultima)
	})
}

// NoRepeat: as mais tocadas dos últimos trinta dias; no empate, a mais recente.
func NoRepeat(tocadas []Tocada, agora time.Time) []Faixa {
	lista := resume(desde(tocadas, agora.Add(-JanelaHistorico)))
	porVezes(lista)
	return faixasDe(lista, tamanhoNoRepeat)
}

// TocadasRecentemente: cada faixa uma vez, a mais nova primeiro, numa semana.
func TocadasRecentemente(tocadas []Tocada, agora time.Time) []Faixa {
	lista := resume(desde(tocadas, agora.Add(-janelaRecentes)))
	sort.SliceStable(lista, func(i, j int) bool { return lista[i].ultima.After(lista[j].ultima) })
	return faixasDe(lista, tamanhoLista)
}

// Redescobrir: faixas ouvidas pela última vez há duas a quatro semanas, a
// mais antiga primeiro.
func Redescobrir(tocadas []Tocada, agora time.Time) []Faixa {
	corte := agora.Add(-redescobrirDepois)
	var lista []resumo
	for _, r := range resume(desde(tocadas, agora.Add(-JanelaHistorico))) {
		if !r.ultima.After(corte) {
			lista = append(lista, r)
		}
	}
	sort.SliceStable(lista, func(i, j int) bool { return lista[i].ultima.Before(lista[j].ultima) })
	return faixasDe(lista, tamanhoLista)
}

// Periodo é a janela do resumo de escuta.
type Periodo string

const (
	PeriodoDia    Periodo = "DAY"
	PeriodoSemana Periodo = "WEEK"
	PeriodoMes    Periodo = "MONTH"
)

// PeriodoDe lê o nome de um período (em qualquer caixa).
func PeriodoDe(nome string) (Periodo, bool) {
	switch p := Periodo(strings.ToUpper(nome)); p {
	case PeriodoDia, PeriodoSemana, PeriodoMes:
		return p, true
	}
	return "", false
}

// Janela é quanto tempo o período cobre.
func (p Periodo) Janela() time.Duration {
	switch p {
	case PeriodoDia:
		return dia
	case PeriodoMes:
		return 30 * dia
	}
	return 7 * dia
}

// Contagem é um nome (ou uma faixa) e quantas vezes tocou.
type Contagem struct {
	Nome  string
	Faixa Faixa
	Vezes int
}

// Resumo é o que foi ouvido num período.
type Resumo struct {
	Tocadas  int
	Duracao  time.Duration // só das faixas de duração conhecida
	Artistas []Contagem
	Faixas   []Contagem
}

const topo = 8

// ResumoDe soma as tocadas do período, o tempo e os artistas e faixas do topo.
func ResumoDe(tocadas []Tocada, periodo Periodo, agora time.Time) Resumo {
	dentro := desde(tocadas, agora.Add(-periodo.Janela()))
	faixas := resume(dentro)
	porVezes(faixas)
	onde := map[string]int{}
	var artistas []Contagem
	r := Resumo{Tocadas: len(dentro)}
	for _, t := range dentro {
		r.Duracao += t.Faixa.Duracao
		if strings.TrimSpace(t.Faixa.Artista) == "" {
			continue
		}
		i, visto := onde[t.Faixa.Artista]
		if !visto {
			i = len(artistas)
			onde[t.Faixa.Artista] = i
			artistas = append(artistas, Contagem{Nome: t.Faixa.Artista})
		}
		artistas[i].Vezes++
	}
	sort.SliceStable(artistas, func(i, j int) bool { return artistas[i].Vezes > artistas[j].Vezes })
	r.Artistas = artistas[:min(len(artistas), topo)]
	for _, f := range faixas[:min(len(faixas), topo)] {
		r.Faixas = append(r.Faixas, Contagem{Nome: f.faixa.Titulo, Faixa: f.faixa, Vezes: f.vezes})
	}
	return r
}

// VolumePara é a normalização pelo loudnessDb do YouTube (quanto a faixa passa
// do alvo de −14 LKFS): só abaixa as altas, nunca aumenta, e corta no máximo
// 24 dB. Sem medida (nil), o volume fica como está.
func VolumePara(loudnessDb *float64) float64 {
	if loudnessDb == nil {
		return 1
	}
	corte := math.Min(math.Max(*loudnessDb, 0), 24)
	return math.Pow(10, -corte/20)
}

// Qualidade do áudio: a mais rica, ou até 128 kbps para economizar.
type Qualidade string

const (
	QualidadeAlta     Qualidade = "HIGH"
	QualidadeEconomia Qualidade = "ECONOMY"
)

// QualidadeDe lê o nome de uma qualidade (em qualquer caixa).
func QualidadeDe(nome string) (Qualidade, bool) {
	switch q := Qualidade(strings.ToUpper(nome)); q {
	case QualidadeAlta, QualidadeEconomia:
		return q, true
	}
	return "", false
}

// Downloads simultâneos permitidos.
const (
	MinTrabalhadores = 1
	MaxTrabalhadores = 4
)

// Ajustes são as preferências do ouvinte.
type Ajustes struct {
	Autoplay      bool
	Qualidade     Qualidade
	Trabalhadores int
	Normalizar    bool
	PularSilencio bool
	OndaSonora    bool
}

// AjustesPadrao é como o app vem.
func AjustesPadrao() Ajustes {
	return Ajustes{Autoplay: true, Qualidade: QualidadeAlta, Trabalhadores: 1, Normalizar: true, OndaSonora: true}
}

// ComTrabalhadores mantém os downloads simultâneos entre um e quatro.
func (a Ajustes) ComTrabalhadores(n int) Ajustes {
	a.Trabalhadores = min(max(n, MinTrabalhadores), MaxTrabalhadores)
	return a
}
