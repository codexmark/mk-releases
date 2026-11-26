// SPDX-License-Identifier: GPL-3.0-or-later

package daemon

import (
	"sync"
)

// avulsasNaFila é quantas mensagens avulsas esperam por uma janela lenta; além
// disso as mais velhas se perdem (os estados nunca: fica sempre o último).
const avulsasNaFila = 64

// Quadro guarda o último estado de cada área (uma linha JSON por "type") e o
// repassa, junto com as mensagens avulsas, a quem está conectado. Só o laço o
// usa; a entrega a cada janela corre por conta do Assinante.
type Quadro struct {
	ordem      []string
	linhas     map[string]string
	assinantes map[*Assinante]bool
	ouvintes   []func(tipo, linha string)
	// guardados são avisos dados antes de haver janela (os da subida): saem quando
	// a primeira janela mandar um comando (quem só bate no socket para ver se há
	// daemon, como o lançador, não os leva embora).
	guardados []string
}

// NovoQuadro começa vazio.
func NovoQuadro() *Quadro {
	return &Quadro{linhas: map[string]string{}, assinantes: map[*Assinante]bool{}}
}

// Estado troca o estado de uma área; o "type" da mensagem é o nome dela. Um
// estado igual ao anterior não é repassado.
func (q *Quadro) Estado(tipo string, m msg) {
	m["type"] = tipo
	linha := codifica(m)
	anterior, existia := q.linhas[tipo]
	if existia && anterior == linha {
		return
	}
	if !existia {
		q.ordem = append(q.ordem, tipo)
	}
	q.linhas[tipo] = linha
	for a := range q.assinantes {
		a.poe(tipo, linha)
	}
	for _, ouve := range q.ouvintes {
		ouve(tipo, linha)
	}
}

// Avulsa manda uma mensagem só a quem está conectado agora (uma página
// pedida, um aviso).
func (q *Quadro) Avulsa(m msg) {
	linha := codifica(m)
	for a := range q.assinantes {
		a.poe("", linha)
	}
	for _, ouve := range q.ouvintes {
		ouve("", linha)
	}
}

// Aviso é uma linha curta que a janela mostra por um instante ("Curtida").
func (q *Quadro) Aviso(texto string) {
	if len(q.assinantes) == 0 && len(q.guardados) < avulsasNaFila {
		q.guardados = append(q.guardados, texto)
		for _, ouve := range q.ouvintes {
			ouve("", codifica(msg{"type": "notice", "text": texto}))
		}
		return
	}
	q.Avulsa(msg{"type": "notice", "text": texto})
}

// SoltaGuardados entrega os avisos da subida a quem está conectado.
func (q *Quadro) SoltaGuardados() {
	if len(q.guardados) == 0 || len(q.assinantes) == 0 {
		return
	}
	guardados := q.guardados
	q.guardados = nil
	for _, texto := range guardados {
		for a := range q.assinantes {
			a.poe("", codifica(msg{"type": "notice", "text": texto}))
		}
	}
}

// Linha é o estado atual de uma área ("" se ela ainda não publicou).
func (q *Quadro) Linha(tipo string) string { return q.linhas[tipo] }

// Linhas são todos os estados, na ordem em que as áreas nasceram.
func (q *Quadro) Linhas() []string {
	linhas := make([]string, 0, len(q.ordem))
	for _, tipo := range q.ordem {
		linhas = append(linhas, q.linhas[tipo])
	}
	return linhas
}

// Ouve registra quem quer ver cada mudança de dentro do daemon (o MPRIS); as
// avulsas chegam com o tipo vazio.
func (q *Quadro) Ouve(f func(tipo, linha string)) { q.ouvintes = append(q.ouvintes, f) }

// Assina liga uma janela: ela recebe primeiro todos os estados como estão e
// depois cada mudança e cada avulsa.
func (q *Quadro) Assina() *Assinante {
	a := &Assinante{chegou: make(chan struct{}, 1)}
	for _, tipo := range q.ordem {
		a.poe(tipo, q.linhas[tipo])
	}
	q.assinantes[a] = true
	return a
}

// Desliga para de mandar a essa janela.
func (q *Quadro) Desliga(a *Assinante) { delete(q.assinantes, a) }

type entrega struct{ tipo, linha string }

// Assinante é a fila de uma janela. O laço põe; a goroutine que escreve no
// socket tira. A fila tem teto: um estado novo toma o lugar do antigo da mesma
// área, e as avulsas mais velhas caem quando passam do limite.
type Assinante struct {
	mu     sync.Mutex
	fila   []entrega
	chegou chan struct{}
}

func (a *Assinante) poe(tipo, linha string) {
	a.mu.Lock()
	trocou := false
	if tipo != "" {
		for i := range a.fila {
			if a.fila[i].tipo == tipo {
				a.fila[i].linha = linha
				trocou = true
				break
			}
		}
	}
	if !trocou {
		a.fila = append(a.fila, entrega{tipo, linha})
		avulsas := 0
		for _, e := range a.fila {
			if e.tipo == "" {
				avulsas++
			}
		}
		for i := 0; avulsas > avulsasNaFila && i < len(a.fila); i++ {
			if a.fila[i].tipo == "" {
				a.fila = append(a.fila[:i], a.fila[i+1:]...)
				avulsas--
				i--
			}
		}
	}
	a.mu.Unlock()
	select {
	case a.chegou <- struct{}{}:
	default:
	}
}

// Chegou avisa quando há linhas a tirar.
func (a *Assinante) Chegou() <-chan struct{} { return a.chegou }

// Tira esvazia a fila e devolve as linhas, na ordem.
func (a *Assinante) Tira() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	linhas := make([]string, 0, len(a.fila))
	for _, e := range a.fila {
		linhas = append(linhas, e.linha)
	}
	a.fila = nil
	return linhas
}
