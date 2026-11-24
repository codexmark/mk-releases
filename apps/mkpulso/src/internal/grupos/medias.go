package grupos

import (
	"math"
	"time"

	"mkpulso/internal/coleta"
)

// As duas janelas das médias.
const (
	umMinuto    = time.Minute
	tresMinutos = 3 * time.Minute
)

type ponto struct {
	quando time.Duration
	cpu    float64
	mem    float64
	disco  float64
}

// Historico guarda, em memória, os últimos três minutos de cada grupo. As
// janelas são por tempo, não por contagem: com o watch a 250 ms ou a 5 s, um
// minuto é um minuto. Quem chama dá o relógio (monotônico).
type Historico struct {
	pontos map[string][]ponto
}

// NovoHistorico começa vazio.
func NovoHistorico() *Historico {
	return &Historico{pontos: map[string][]ponto{}}
}

func chaveDoGrupo(g *Grupo) string { return g.Escopo + "|" + g.ID }

// Registrar anota o retrato de agora e preenche a média de cada grupo: a
// média simples dos pontos do último minuto e dos últimos três (o de agora
// incluído; antes de completar o tempo, vale a média do que há). O que tem
// três minutos ou mais é jogado fora, e um grupo que não veio neste retrato
// é esquecido: se voltar, começa do zero.
func (h *Historico) Registrar(agora time.Duration, gs []Grupo) {
	novos := make(map[string][]ponto, len(gs))
	for i := range gs {
		g := &gs[i]
		chave := chaveDoGrupo(g)
		p := ponto{quando: agora, cpu: g.CPUPct, mem: float64(g.MemKB)}
		if g.DiscoBps != nil {
			p.disco = *g.DiscoBps
		}
		// novos[chave] primeiro: dois grupos com a mesma chave no mesmo
		// retrato não perdem o histórico um do outro
		serie, ja := novos[chave]
		if !ja {
			serie = h.pontos[chave]
		}
		corte := 0
		for corte < len(serie) && (agora-serie[corte].quando >= tresMinutos || serie[corte].quando > agora) {
			corte++
		}
		serie = append(serie[corte:], p)
		novos[chave] = serie

		var m1, m3 ponto
		var n1, n3 float64
		for _, q := range serie {
			m3.cpu, m3.mem, m3.disco = m3.cpu+q.cpu, m3.mem+q.mem, m3.disco+q.disco
			n3++
			if agora-q.quando < umMinuto {
				m1.cpu, m1.mem, m1.disco = m1.cpu+q.cpu, m1.mem+q.mem, m1.disco+q.disco
				n1++
			}
		}
		g.Media = Media{
			CPUPct1m: coleta.Arred(m1.cpu/n1, 1), CPUPct3m: coleta.Arred(m3.cpu/n3, 1),
			MemKB1m: int64(math.Round(m1.mem / n1)), MemKB3m: int64(math.Round(m3.mem / n3)),
			DiscoBps1m: coleta.Arred(m1.disco/n1, 0), DiscoBps3m: coleta.Arred(m3.disco/n3, 0),
		}
	}
	h.pontos = novos
}

// Pontos diz quantos pontos estão guardados para um id (somando os escopos).
func (h *Historico) Pontos(id string) int {
	n := 0
	for chave, serie := range h.pontos {
		if len(chave) > len(id) && chave[len(chave)-len(id):] == id && chave[len(chave)-len(id)-1] == '|' {
			n += len(serie)
		}
	}
	return n
}
