// SPDX-License-Identifier: GPL-3.0-or-later

package app

import (
	"context"
	"time"

	"mktunes/internal/dominio"
	"mktunes/internal/laco"
)

// Situacao é em que pé a reprodução está.
type Situacao int

const (
	Ocioso Situacao = iota
	Carregando
	Tocando
	Pausado
	// FalhouStream: não deu para achar o áudio da faixa.
	FalhouStream
	// FalhouTocador: o tocador desistiu no meio.
	FalhouTocador
)

// Falhou junta os dois jeitos de falhar.
func (s Situacao) Falhou() bool { return s == FalhouStream || s == FalhouTocador }

// Estado é a fila e o que acontece com a faixa atual. TocaID muda cada vez que
// uma faixa começa a carregar, mesmo que seja a mesma de novo.
type Estado struct {
	Fila     dominio.Fila
	Situacao Situacao
	TocaID   int64
}

// Faixa é a que toca, se há uma.
func (e Estado) Faixa() (dominio.Faixa, bool) { return e.Fila.Atual() }

// Progresso é onde a faixa atual está.
type Progresso struct{ Posicao, Duracao time.Duration }

const (
	reiniciaDepoisDe = 3 * time.Second
	poucasRestantes  = 2
	loteDoAutoplay   = 20
	velocidadeMin    = 0.5
	velocidadeMax    = 2.0
)

// OpcoesDaSessao são as peças da sessão; só Resolvedor, Tocador e Laco são
// obrigatórios.
type OpcoesDaSessao struct {
	Resolvedor Resolvedor
	Tocador    Tocador
	Laco       laco.Laco
	// Autoplay é de onde vem o rádio que continua a fila.
	Autoplay       FonteDeFilas
	AutoplayLigado func() bool
	Normalizar     func() bool
	Mistura        dominio.Embaralha
	// Local dá a cópia baixada de uma faixa, quando existe: toca sem rede.
	Local func(f dominio.Faixa) (Stream, bool)
	// AoFalhar recebe o erro de um stream que não veio (para o registro).
	AoFalhar func(f dominio.Faixa, erro error)
	// AoRecusar recebe o motivo de o tocador não ter aberto a faixa (para o registro).
	AoRecusar func(f dominio.Faixa, motivo string)
}

// Sessao é dona da fila: pede os streams ao resolvedor e diz ao tocador o que
// fazer. Tudo nela roda no laço.
type Sessao struct {
	o OpcoesDaSessao
	// Mudou avisa a cada mudança do estado ou da velocidade.
	Mudou Sinal

	estado     Estado
	velocidade float64
	carregando laco.Tarefa
	estendendo laco.Tarefa
	// tocaASeguir conta os "tocar a seguir" desde que a faixa atual começou: o
	// próximo entra depois deles.
	tocaASeguir int
	// retomarEm é onde começar quando uma sessão restaurada for tocada: até lá
	// nada é carregado.
	retomarEm *time.Duration
	// esperaTocador: o tocador já recebeu a faixa e ainda não disse que a abriu
	// (a situação continua Carregando até ele dizer, ou falhar).
	esperaTocador bool
	// pausaAoCarregar: o ouvinte pausou nessa espera; a faixa abre pausada.
	pausaAoCarregar bool
	// geracao sobe quando o ouvinte troca a fila: respostas atrasadas do
	// autoplay da fila antiga são descartadas.
	geracao int
}

// NovaSessao começa ociosa, com a fila vazia.
func NovaSessao(o OpcoesDaSessao) *Sessao {
	if o.AutoplayLigado == nil {
		o.AutoplayLigado = func() bool { return true }
	}
	if o.Normalizar == nil {
		o.Normalizar = func() bool { return true }
	}
	return &Sessao{o: o, estado: Estado{Fila: dominio.FilaVazia()}, velocidade: 1}
}

// Estado é o de agora.
func (s *Sessao) Estado() Estado { return s.estado }

// Velocidade é a de agora (1 é a normal).
func (s *Sessao) Velocidade() float64 { return s.velocidade }

// DefineVelocidade muda a velocidade, de metade ao dobro.
func (s *Sessao) DefineVelocidade(v float64) {
	s.velocidade = min(max(v, velocidadeMin), velocidadeMax)
	s.o.Tocador.Velocidade(s.velocidade)
	s.Mudou.Avisa()
}

// TocaDe troca a fila e toca a partir de indice.
func (s *Sessao) TocaDe(faixas []dominio.Faixa, indice int) {
	s.novaFila()
	s.carrega(dominio.NovaFila(faixas, indice).ComRepetir(s.estado.Fila.Repetir()), nil)
}

// Restaura traz de volta uma fila guardada, pausada, sem tocar no tocador até
// o ouvinte apertar tocar.
func (s *Sessao) Restaura(salva SessaoSalva) {
	fila := dominio.Restaura(salva.Fila)
	if _, tem := fila.Atual(); !tem {
		return
	}
	posicao := salva.Posicao
	s.retomarEm = &posicao
	s.define(Estado{Fila: fila, Situacao: Pausado, TocaID: s.estado.TocaID})
}

// AlternaPausa toca ou pausa.
func (s *Sessao) AlternaPausa() {
	if s.retomarEm != nil {
		posicao := *s.retomarEm
		s.carrega(s.estado.Fila, &posicao)
		return
	}
	switch s.estado.Situacao {
	case Carregando:
		// Antes de o tocador ter a faixa não há o que pausar.
		if !s.esperaTocador {
			return
		}
		if s.pausaAoCarregar = !s.pausaAoCarregar; s.pausaAoCarregar {
			s.o.Tocador.Pausa()
		} else {
			s.o.Tocador.Continua()
		}
	case Tocando:
		s.o.Tocador.Pausa()
		s.situacao(Pausado)
	case Pausado:
		s.o.Tocador.Continua()
		s.situacao(Tocando)
	case FalhouStream, FalhouTocador:
		s.carrega(s.estado.Fila, nil)
	}
}

// Proxima pula para a faixa seguinte.
func (s *Sessao) Proxima() { s.carrega(s.estado.Fila.Proxima(), nil) }

// Anterior volta ao começo da faixa, ou à faixa anterior se ela mal começou.
func (s *Sessao) Anterior() {
	// Uma faixa restaurada e ainda não tocada: "recomeçar" só muda onde ela retoma.
	if s.retomarEm != nil {
		if *s.retomarEm > reiniciaDepoisDe {
			zero := time.Duration(0)
			s.retomarEm = &zero
			s.Mudou.Avisa()
			return
		}
		s.carrega(s.estado.Fila.Anterior(), nil)
		return
	}
	if s.Progresso().Posicao > reiniciaDepoisDe {
		s.o.Tocador.VaiPara(0)
		return
	}
	s.carrega(s.estado.Fila.Anterior(), nil)
}

// DefineRepetir escolhe o modo de repetição.
func (s *Sessao) DefineRepetir(modo dominio.Repetir) {
	s.editaFila(func(q dominio.Fila) dominio.Fila { return q.ComRepetir(modo) })
}

// CicloRepetir passa ao próximo modo de repetição.
func (s *Sessao) CicloRepetir() { s.editaFila(dominio.Fila.CicloRepetir) }

// AlternaPararAposAtual liga ou desliga o "parar depois desta".
func (s *Sessao) AlternaPararAposAtual() {
	s.editaFila(func(q dominio.Fila) dominio.Fila { return q.ComPararAposAtual(!q.PararAposAtual()) })
}

// TocaASeguir põe a faixa logo depois da atual, depois dos "tocar a seguir"
// anteriores.
func (s *Sessao) TocaASeguir(f dominio.Faixa) {
	if _, tem := s.estado.Faixa(); !tem {
		s.PoeNaFila(f)
		return
	}
	lugar := s.tocaASeguir
	s.editaFila(func(q dominio.Fila) dominio.Fila { return q.InsereASeguir(lugar, f) })
	s.tocaASeguir++
}

// PoeNaFila põe a faixa no fim; sem nada tocando, começa por ela.
func (s *Sessao) PoeNaFila(f dominio.Faixa) {
	if _, tem := s.estado.Faixa(); !tem {
		s.novaFila()
		s.carrega(dominio.NovaFila([]dominio.Faixa{f}, 0), nil)
		return
	}
	s.editaFila(func(q dominio.Fila) dominio.Fila { return q.Enfileira(f) })
}

// TocaDaFila pula para uma faixa da lista "a seguir".
func (s *Sessao) TocaDaFila(indice int) { s.carrega(s.estado.Fila.PulaPara(indice), nil) }

// Mexer à mão na lista "a seguir" encerra a série de "tocar a seguir".

// RemoveDaFila tira uma faixa da lista "a seguir".
func (s *Sessao) RemoveDaFila(indice int) {
	s.reordena(func(q dominio.Fila) dominio.Fila { return q.RemoveASeguir(indice) })
}

// MoveNaFila troca uma faixa de lugar na lista "a seguir".
func (s *Sessao) MoveNaFila(de, para int) {
	s.reordena(func(q dominio.Fila) dominio.Fila { return q.MoveASeguir(de, para) })
}

// LimpaFila esvazia a lista "a seguir".
func (s *Sessao) LimpaFila() { s.reordena(dominio.Fila.LimpaASeguir) }

// EmbaralhaFila mistura a lista "a seguir".
func (s *Sessao) EmbaralhaFila() {
	s.reordena(func(q dominio.Fila) dominio.Fila { return q.Embaralhada(s.o.Mistura) })
}

// VaiPara pula para uma posição da faixa.
func (s *Sessao) VaiPara(posicao time.Duration) {
	if s.retomarEm != nil {
		s.retomarEm = &posicao
		s.Mudou.Avisa()
		return
	}
	// Enquanto o próximo stream é resolvido o tocador segura a faixa antiga.
	if s.estado.Situacao == Carregando && !s.esperaTocador {
		return
	}
	s.o.Tocador.VaiPara(posicao)
}

// Progresso diz onde a faixa está; a duração do stream vale mais que a do
// catálogo, quando conhecida.
func (s *Sessao) Progresso() Progresso {
	var doCatalogo time.Duration
	if f, tem := s.estado.Faixa(); tem {
		doCatalogo = f.Duracao
	}
	if s.retomarEm != nil {
		return Progresso{*s.retomarEm, doCatalogo}
	}
	if s.estado.Situacao == Carregando && !s.esperaTocador {
		return Progresso{0, doCatalogo}
	}
	duracao := s.o.Tocador.Duracao()
	if duracao <= 0 {
		duracao = doCatalogo
	}
	return Progresso{s.o.Tocador.Posicao(), duracao}
}

// AoEvento recebe o que o tocador conta.
func (s *Sessao) AoEvento(e Evento) {
	switch e.Tipo {
	case Terminou:
		// O que o tocador segura enquanto carrega pode "acabar": não é a música acabando.
		if s.estado.Situacao != Carregando {
			s.carrega(s.estado.Fila.AoTerminar(), nil)
		}
	case Carregou:
		if !s.esperaTocador {
			return
		}
		s.esperaTocador = false
		if s.pausaAoCarregar {
			s.situacao(Pausado)
		} else {
			s.situacao(Tocando)
		}
	case FalhouAoTocar:
		// O que o tocador segura enquanto a próxima é resolvida também pode
		// falhar: não é a faixa nova falhando.
		if s.estado.Situacao == Carregando && !s.esperaTocador {
			return
		}
		s.esperaTocador = false
		if faixa, tem := s.estado.Faixa(); tem && s.o.AoRecusar != nil {
			s.o.AoRecusar(faixa, e.Motivo)
		}
		s.situacao(FalhouTocador)
	case MudouPausa:
		switch {
		case s.estado.Situacao == Tocando && !e.Tocando:
			s.situacao(Pausado)
		case s.estado.Situacao == Pausado && e.Tocando && s.retomarEm == nil:
			s.situacao(Tocando)
		}
	case PediuProxima:
		s.Proxima()
	case PediuAnterior:
		s.Anterior()
	case PediuTocar:
		// Só "tocar", nunca pausar.
		if s.retomarEm != nil || s.estado.Situacao == Pausado || s.estado.Situacao.Falhou() || (s.esperaTocador && s.pausaAoCarregar) {
			s.AlternaPausa()
		}
	}
}

// TocaColecao toca um álbum ou playlist inteiro, do começo; avisa em resultado
// se deu (false: não carregou, ou veio vazio).
func (s *Sessao) TocaColecao(fonte FonteDeFilas, playlistID string, resultado func(deu bool)) {
	s.o.Laco.Vai(func(ctx context.Context) func() {
		faixas, erro := fonte.Playlist(ctx, playlistID)
		return func() {
			deu := erro == nil && len(faixas) > 0
			if deu {
				s.TocaDe(faixas, 0)
			}
			if resultado != nil {
				resultado(deu)
			}
		}
	})
}

func (s *Sessao) editaFila(muda func(dominio.Fila) dominio.Fila) {
	s.estado.Fila = muda(s.estado.Fila)
	s.Mudou.Avisa()
}

func (s *Sessao) reordena(muda func(dominio.Fila) dominio.Fila) {
	s.tocaASeguir = 0
	s.editaFila(muda)
}

func (s *Sessao) define(e Estado) {
	s.estado = e
	s.Mudou.Avisa()
}

func (s *Sessao) situacao(sit Situacao) {
	s.estado.Situacao = sit
	s.Mudou.Avisa()
}

func (s *Sessao) carrega(fila dominio.Fila, comecaEm *time.Duration) {
	s.retomarEm = nil
	s.esperaTocador, s.pausaAoCarregar = false, false
	s.tocaASeguir = 0
	laco.Cancela(s.carregando)
	faixa, tem := fila.Atual()
	if !tem {
		s.o.Tocador.Para()
		s.define(Estado{Fila: fila, Situacao: Ocioso, TocaID: s.estado.TocaID})
		return
	}
	// Segura a faixa antiga enquanto a próxima é resolvida.
	s.o.Tocador.Segura(faixa)
	s.define(Estado{Fila: fila, Situacao: Carregando, TocaID: s.estado.TocaID + 1})
	if s.o.Local != nil {
		if stream, tem := s.o.Local(faixa); tem {
			s.carregando = nil
			s.comeca(faixa, stream, comecaEm)
			return
		}
	}
	s.carregando = s.o.Laco.Vai(func(ctx context.Context) func() {
		stream, erro := s.o.Resolvedor.Resolve(ctx, faixa)
		return func() {
			if erro != nil {
				if s.o.AoFalhar != nil {
					s.o.AoFalhar(faixa, erro)
				}
				s.situacao(FalhouStream)
				return
			}
			s.comeca(faixa, stream, comecaEm)
		}
	})
}

func (s *Sessao) comeca(faixa dominio.Faixa, stream Stream, comecaEm *time.Duration) {
	stream.Volume = 1
	if s.o.Normalizar() {
		stream.Volume = dominio.VolumePara(stream.LoudnessDb)
	}
	// A situação só vira Tocando quando o tocador conta que abriu a faixa
	// (Carregou): até lá a janela mostra "carregando", não um 0:00 parado.
	s.esperaTocador = true
	s.o.Tocador.Toca(stream, faixa)
	if comecaEm != nil {
		s.o.Tocador.VaiPara(*comecaEm)
	}
	s.estendeComRadio()
}

// estendeComRadio: com o autoplay ligado e sem repetir, completa a fila com o
// rádio da última faixa quando sobram duas ou menos.
func (s *Sessao) estendeComRadio() {
	if s.o.Autoplay == nil {
		return
	}
	fila := s.estado.Fila
	semente, tem := fila.Ultima()
	if !tem || semente.AoVivo() {
		return
	}
	if !s.o.AutoplayLigado() || fila.Repetir() != dominio.RepetirNao || len(fila.ASeguir()) > poucasRestantes || laco.Ativa(s.estendendo) {
		return
	}
	pedidaPara := s.geracao
	s.estendendo = s.o.Laco.Vai(func(ctx context.Context) func() {
		achadas, erro := s.o.Autoplay.Radio(ctx, semente)
		return func() {
			if erro != nil || pedidaPara != s.geracao {
				return
			}
			// A fila acabou antes de a resposta chegar: nada a estender.
			if _, tem := s.estado.Faixa(); !tem {
				return
			}
			var novas []dominio.Faixa
			vistas := map[string]bool{}
			for _, f := range achadas {
				if vistas[f.ID] || s.estado.Fila.Contem(f.ID) {
					continue
				}
				vistas[f.ID] = true
				if novas = append(novas, f); len(novas) == loteDoAutoplay {
					break
				}
			}
			s.editaFila(func(q dominio.Fila) dominio.Fila { return q.EnfileiraTodas(novas) })
		}
	})
}

// novaFila: uma fila que o ouvinte escolheu; o que ainda vinha para a antiga é
// descartado.
func (s *Sessao) novaFila() {
	s.geracao++
	laco.Cancela(s.estendendo)
}
