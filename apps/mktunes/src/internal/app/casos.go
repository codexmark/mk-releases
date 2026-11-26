// SPDX-License-Identifier: GPL-3.0-or-later

package app

import (
	"context"
	"strings"
	"sync"
	"time"

	"mktunes/internal/dominio"
	"mktunes/internal/laco"
)

// Desfecho é como uma busca terminou.
type Desfecho int

const (
	Achou Desfecho = iota
	NadaAchado
	ConsultaInvalida
	Indisponivel
)

// BuscaMusicas procura músicas num catálogo.
func BuscaMusicas(ctx context.Context, c Catalogo, bruto string) ([]dominio.Faixa, Desfecho) {
	consulta, ok := dominio.Consulta(bruto)
	if !ok {
		return nil, ConsultaInvalida
	}
	faixas, erro := c.BuscaMusicas(ctx, consulta)
	switch {
	case erro != nil:
		return nil, Indisponivel
	case len(faixas) == 0:
		return nil, NadaAchado
	}
	return faixas, Achou
}

// BuscaPaginas procura álbuns, artistas ou playlists.
func BuscaPaginas(ctx context.Context, b BuscaDePaginas, bruto string, filtro dominio.Filtro) ([]dominio.Item, Desfecho) {
	consulta, ok := dominio.Consulta(bruto)
	if !ok {
		return nil, ConsultaInvalida
	}
	itens, erro := b.BuscaPaginas(ctx, consulta, filtro)
	switch {
	case erro != nil:
		return nil, Indisponivel
	case len(itens) == 0:
		return nil, NadaAchado
	}
	return itens, Achou
}

// Sugere dá as sugestões para o que foi digitado: a partir de duas letras, e
// nada (sem erro) quando o catálogo não responde.
func Sugere(ctx context.Context, f FonteDeSugestoes, texto string) []string {
	digitado := strings.TrimSpace(texto)
	if len([]rune(digitado)) < 2 {
		return nil
	}
	sugestoes, erro := f.Sugere(ctx, digitado)
	if erro != nil {
		return nil
	}
	return sugestoes
}

// Estacoes lista as rádios mais votadas, ou as que casam com o nome.
func Estacoes(ctx context.Context, d DiretorioDeRadio, consulta string) ([]dominio.Faixa, error) {
	const limite, maxConsulta = 50, 128
	nome := []rune(strings.TrimSpace(consulta))
	if len(nome) > maxConsulta {
		nome = nome[:maxConsulta]
	}
	if len(nome) == 0 {
		return d.Topo(ctx, limite)
	}
	return d.Busca(ctx, string(nome), limite)
}

// Inicio é o feed da tela inicial, página a página, com filtro de clima.
type Inicio struct {
	Fonte  FonteDoInicio
	Guarda GuardaDoInicio // pode ser nil
}

// Guardado é o Início como estava da última vez, para mostrar enquanto o novo
// carrega.
func (i Inicio) Guardado() (dominio.PaginaFeed, bool) {
	if i.Guarda == nil {
		return dominio.PaginaFeed{}, false
	}
	return i.Guarda.Carrega()
}

// Primeira carrega a primeira página; false quando a fonte não respondeu.
func (i Inicio) Primeira(ctx context.Context, chip *dominio.Chip) (dominio.PaginaFeed, bool) {
	pagina, erro := i.Fonte.Inicio(ctx, chip)
	if erro != nil {
		return dominio.PaginaFeed{}, false
	}
	pagina = semVazias(pagina)
	// Os tokens de continuação expiram: o Início guardado é uma página só.
	if chip == nil && i.Guarda != nil {
		guardada := pagina
		guardada.Proxima = ""
		i.Guarda.Guarda(guardada)
	}
	return pagina, true
}

// Proxima carrega a página seguinte pelo token.
func (i Inicio) Proxima(ctx context.Context, token string) (dominio.PaginaFeed, bool) {
	pagina, erro := i.Fonte.Mais(ctx, token)
	if erro != nil {
		return dominio.PaginaFeed{}, false
	}
	return semVazias(pagina), true
}

func semVazias(p dominio.PaginaFeed) dominio.PaginaFeed {
	cheias := make([]dominio.Prateleira, 0, len(p.Prateleiras))
	for _, s := range p.Prateleiras {
		if len(s.Itens) > 0 {
			cheias = append(cheias, s)
		}
	}
	p.Prateleiras = cheias
	return p
}

// DesfechoLetra é como a busca de uma letra terminou.
type DesfechoLetra struct {
	Desfecho Desfecho // Achou, NadaAchado ou Indisponivel
	Letra    dominio.LetraDaFaixa
}

// lembradas é quantas letras ficam na memória: as das últimas faixas tocadas.
const lembradas = 16

// Letras acha a letra da faixa que toca e lembra das últimas, para não pedir
// de novo ao voltar uma faixa. Pode ser chamada de fora do laço.
type Letras struct {
	fonte FonteDeLetras
	mu    sync.Mutex
	ordem []string
	sabe  map[string]DesfechoLetra
}

// NovasLetras liga a busca de letras à fonte.
func NovasLetras(f FonteDeLetras) *Letras { return &Letras{fonte: f, sabe: map[string]DesfechoLetra{}} }

// Acha devolve a letra; uma fonte fora do ar não é lembrada (pergunta de novo).
func (l *Letras) Acha(ctx context.Context, f dominio.Faixa) DesfechoLetra {
	l.mu.Lock()
	sabido, tem := l.sabe[f.ID]
	l.mu.Unlock()
	if tem {
		return sabido
	}
	achada, erro := l.fonte.Acha(ctx, f)
	if erro != nil {
		return DesfechoLetra{Desfecho: Indisponivel}
	}
	desfecho := DesfechoLetra{Desfecho: NadaAchado}
	if achada != nil {
		desfecho = DesfechoLetra{Desfecho: Achou, Letra: *achada}
	}
	l.mu.Lock()
	if _, tem := l.sabe[f.ID]; !tem {
		l.ordem = append(l.ordem, f.ID)
		if len(l.ordem) > lembradas {
			delete(l.sabe, l.ordem[0])
			l.ordem = l.ordem[1:]
		}
	}
	l.sabe[f.ID] = desfecho
	l.mu.Unlock()
	return desfecho
}

// SonecaEstado é o que o timer de dormir está fazendo.
type SonecaEstado int

const (
	SonecaDesligada SonecaEstado = iota
	SonecaContando
	SonecaFimDaFaixa
)

// Soneca pausa a música depois de um tempo, ou quando a faixa atual acabar.
type Soneca struct {
	sessao *Sessao
	laco   laco.Laco
	Mudou  Sinal

	estado    SonecaEstado
	resta     time.Duration
	contagem  laco.Tarefa
	esperando bool // o tempo acabou com uma faixa carregando
}

// NovaSoneca liga o timer à sessão.
func NovaSoneca(s *Sessao, l laco.Laco) *Soneca {
	t := &Soneca{sessao: s, laco: l}
	s.Mudou.Ouve(t.aoMudarSessao)
	return t
}

// Estado é o de agora; Resta vale enquanto conta.
func (t *Soneca) Estado() (SonecaEstado, time.Duration) { return t.estado, t.resta }

// Comeca conta o tempo e pausa no fim.
func (t *Soneca) Comeca(depois time.Duration) {
	t.Cancela()
	t.conta(depois)
}

func (t *Soneca) conta(resta time.Duration) {
	if resta > 0 {
		t.define(SonecaContando, resta)
		t.contagem = t.laco.Depois(time.Second, func() { t.conta(resta - time.Second) })
		return
	}
	// Uma faixa pode estar carregando agora: espera começar, e então pausa.
	t.esperando = true
	t.aoMudarSessao()
}

// FimDaFaixa para quando a faixa atual terminar.
func (t *Soneca) FimDaFaixa() {
	t.Cancela()
	if !t.sessao.Estado().Fila.PararAposAtual() {
		t.sessao.AlternaPararAposAtual()
	}
	t.define(SonecaFimDaFaixa, 0)
}

// Cancela desliga o timer.
func (t *Soneca) Cancela() {
	laco.Cancela(t.contagem)
	t.esperando = false
	era := t.estado
	t.define(SonecaDesligada, 0)
	if era == SonecaFimDaFaixa && t.sessao.Estado().Fila.PararAposAtual() {
		t.sessao.AlternaPararAposAtual()
	}
}

func (t *Soneca) aoMudarSessao() {
	estado := t.sessao.Estado()
	if t.esperando && estado.Situacao != Carregando {
		t.esperando = false
		t.define(SonecaDesligada, 0)
		if estado.Situacao == Tocando {
			t.sessao.AlternaPausa()
		}
		return
	}
	// Uma fila nova, ou a faixa acabando, desliga o "parar depois desta": o
	// timer acabou junto.
	if t.estado == SonecaFimDaFaixa && !estado.Fila.PararAposAtual() {
		t.define(SonecaDesligada, 0)
	}
}

func (t *Soneca) define(e SonecaEstado, resta time.Duration) {
	if t.estado == e && t.resta == resta {
		return
	}
	t.estado, t.resta = e, resta
	t.Mudou.Avisa()
}

// OuveHistorico observa a sessão e registra uma tocada quando a faixa foi
// ouvida de verdade (ver dominio.Ouvida): conta o tempo tocando, não onde o
// cursor está — pular para a frente, retomar ou pausar não contam. Guarda só
// os últimos trinta dias.
func OuveHistorico(s *Sessao, g Guarda, l laco.Laco) {
	var (
		ultimoID int64 = -1
		ultima   dominio.Faixa
		contando laco.Tarefa
	)
	var conta func(f dominio.Faixa, ouvido time.Duration)
	conta = func(f dominio.Faixa, ouvido time.Duration) {
		if dominio.Ouvida(ouvido, f.Duracao) {
			agora := l.Agora()
			g.RegistraTocada(dominio.Tocada{Faixa: f, Em: agora})
			g.EsqueceTocadasAntesDe(agora.Add(-dominio.JanelaHistorico))
			return
		}
		contando = l.Depois(time.Second, func() {
			if s.Estado().Situacao == Tocando {
				ouvido += time.Second
			}
			conta(f, ouvido)
		})
	}
	s.Mudou.Ouve(func() {
		estado := s.Estado()
		faixa, tem := estado.Faixa()
		if estado.TocaID == ultimoID && faixa == ultima {
			return
		}
		ultimoID, ultima = estado.TocaID, faixa
		laco.Cancela(contando)
		if tem && !faixa.AoVivo() {
			conta(faixa, 0)
		}
	})
}

// intervaloDaSessao é de quanto em quanto a posição é guardada enquanto toca.
const intervaloDaSessao = 10 * time.Second

// GuardaSessao traz a última fila de volta, pausada, guarda a fila sempre que
// ela muda e a posição de tempos em tempos enquanto toca. Devolve a função
// que guarda agora (para a saída do daemon).
func GuardaSessao(s *Sessao, g Guarda, l laco.Laco) (salvaJa func()) {
	if salva, tem := g.CarregaSessao(); tem {
		s.Restaura(salva)
	}
	salvaJa = func() { g.SalvaSessao(SessaoSalva{Fila: s.Estado().Fila.Retrato(), Posicao: s.Progresso().Posicao}) }
	var ultimo dominio.Retrato
	primeira := true
	aoMudar := func() {
		retrato := s.Estado().Fila.Retrato()
		if !primeira && mesmoRetrato(ultimo, retrato) {
			return
		}
		primeira, ultimo = false, retrato
		salvaJa()
	}
	aoMudar()
	s.Mudou.Ouve(aoMudar)
	var bate func()
	bate = func() {
		if s.Estado().Situacao == Tocando {
			salvaJa()
		}
		l.Depois(intervaloDaSessao, bate)
	}
	l.Depois(intervaloDaSessao, bate)
	return salvaJa
}

func mesmoRetrato(a, b dominio.Retrato) bool {
	if a.Indice != b.Indice || a.Repetir != b.Repetir || len(a.Faixas) != len(b.Faixas) {
		return false
	}
	for i := range a.Faixas {
		if a.Faixas[i] != b.Faixas[i] {
			return false
		}
	}
	return true
}
