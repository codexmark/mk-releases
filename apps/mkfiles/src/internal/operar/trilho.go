// Package operar é o trilho das tarefas do motor: copiar, mover, criar,
// renomear, apagar, a lixeira, desfazer e refazer. Toda tarefa passa pela
// mesma fila (uma por vez), com progresso, pausa, cancelamento e conflitos
// que perguntam à janela. Uma fase nova (renomear em lote, compactar) é só
// mais um Trabalho registrado com Registrar. Veja ../../../CONTRATO.md.
package operar

import (
	"sort"
	"sync"
	"time"

	"mkfiles/internal/base"
	"mkfiles/internal/item"
	"mkfiles/internal/lixeira"
)

// Trabalho é o que uma tarefa faz. Prepara conta o que há para fazer
// (Tarefa.Total); Roda faz, perguntando a Tarefa.Segue entre um passo e outro
// e contando com Tarefa.Avanca.
type Trabalho interface {
	Titulo() string  // "Copiando 3 itens para Documentos"
	Destino() string // a pasta de destino ("" se não há)
	Prepara(t *Tarefa)
	Roda(t *Tarefa)
}

// Refazivel é o trabalho que entra no histórico: tem um nome para o desfazer
// e sabe se montar de novo para o refazer.
type Refazivel interface {
	Descricao() string // "Copiar 3 itens"
	Refaz() Trabalho
}

// Fabrica monta o trabalho de um pedido, ou diz por que não (um *Recusa, um
// *DeCampo ou um erro comum).
type Fabrica func(p *base.Pedido) (Trabalho, error)

// Recusa é a recusa de segurança ("recusado": true).
type Recusa struct{ Frase string }

func (r *Recusa) Error() string { return r.Frase }

// DeCampo é o erro de formulário (diz o campo).
type DeCampo struct{ Campo, Frase string }

func (e *DeCampo) Error() string { return e.Frase }

// ComCampos é o erro que leva mais campos na resposta (os conflitos de um
// renomear em lote).
type ComCampos struct {
	Frase string
	Mais  base.Campos
}

func (e *ComCampos) Error() string { return e.Frase }

var fabricas = map[string]Fabrica{}

// Registrar liga uma ação (o nome do pedido) à fábrica do trabalho dela. É
// chamado nos init: o Ligar põe na sessão tudo que estiver registrado.
func Registrar(acao string, f Fabrica) {
	if _, ja := fabricas[acao]; ja {
		panic("operar: ação registrada duas vezes: " + acao)
	}
	fabricas[acao] = f
}

// O intervalo mínimo entre dois eventos de andamento de uma tarefa.
var intervalo = 100 * time.Millisecond

// Os modos de uma tarefa.
const (
	normal = iota
	desfazendo
	refazendo
)

type registro struct {
	descricao string
	feitos    []feito
	refaz     func() Trabalho
}

// Trilho é a fila de tarefas de uma sessão, e o histórico dela.
type Trilho struct {
	s *base.Sessao

	trava    sync.Mutex
	fila     []*Tarefa
	vivas    map[int]*Tarefa // as que ainda não acabaram
	proxima  int
	chegou   chan struct{}
	fechou   chan struct{}
	fechado  bool
	desfazer []*registro
	refazer  []*registro
}

// Resposta é o que a janela disse num conflito.
type Resposta struct {
	Qual  string // substituir, pular, manter, renomear, cancelar
	Nome  string
	todos bool
}

// Tarefa é uma operação na fila. Os métodos exportados são o que um Trabalho
// usa.
type Tarefa struct {
	ID   int
	Acao string

	tr       *Trilho
	trabalho Trabalho
	modo     int
	reg      *registro // o que está sendo desfeito ou refeito

	trava      sync.Mutex
	estado     string
	itens      int
	itensTotal int
	bytes      int64
	bytesTotal int64
	atual      string
	criados    []string
	erros      []base.Campos
	semLixeira []string
	feitos     []feito
	naoDesfaz  bool
	desfazivel bool
	final      bool

	pausada bool
	retoma  chan struct{}
	parar   chan struct{}
	umaVez  sync.Once

	conflito  int
	esperando bool
	respostas chan Resposta
	paraTodos string

	ultimoEvento time.Time
	marca        time.Time
	bytesNaMarca int64
	inicio       time.Time
	velocidade   int64
}

// Ligar põe na sessão os pedidos das tarefas registradas e os de controle.
func Ligar(s *base.Sessao) {
	tr := &Trilho{s: s, vivas: map[int]*Tarefa{}, chegou: make(chan struct{}, 1), fechou: make(chan struct{})}
	acoes := make([]string, 0, len(fabricas))
	for acao := range fabricas {
		acoes = append(acoes, acao)
	}
	sort.Strings(acoes)
	for _, acao := range acoes {
		fabrica := fabricas[acao]
		s.Ligar(acao, func(p *base.Pedido) {
			trabalho, err := fabrica(p)
			if err != nil {
				falha(p, err)
				return
			}
			t := tr.enfileira(p.Op, trabalho, normal, nil)
			if t == nil {
				p.Falha("o motor está fechando")
				return
			}
			p.Responde(base.Campos{"tarefa": t.ID})
		})
	}
	s.Ligar("desfazer", func(p *base.Pedido) { tr.volta(p, desfazendo) })
	s.Ligar("refazer", func(p *base.Pedido) { tr.volta(p, refazendo) })
	s.Ligar("responder", tr.responder)
	s.Ligar("tarefa-pausar", func(p *base.Pedido) {
		var q struct {
			Tarefa int  `json:"tarefa"`
			Pausar bool `json:"pausar"`
		}
		_ = p.Ler(&q)
		t := tr.viva(q.Tarefa)
		if t == nil {
			p.Falha("essa tarefa já acabou")
			return
		}
		t.pausa(q.Pausar)
		p.Responde(base.Campos{"tarefa": t.ID})
	})
	s.Ligar("tarefa-cancelar", func(p *base.Pedido) {
		var q struct {
			Tarefa int `json:"tarefa"`
		}
		_ = p.Ler(&q)
		t := tr.viva(q.Tarefa)
		if t == nil {
			p.Falha("essa tarefa já acabou")
			return
		}
		t.cancela()
		p.Responde(base.Campos{"tarefa": t.ID})
	})
	s.Ligar("tarefas", func(p *base.Pedido) {
		tr.trava.Lock()
		vivas := make([]*Tarefa, 0, len(tr.vivas))
		for _, t := range tr.vivas {
			vivas = append(vivas, t)
		}
		tr.trava.Unlock()
		sort.Slice(vivas, func(i, j int) bool { return vivas[i].ID < vivas[j].ID })
		lista := []base.Campos{}
		for _, t := range vivas {
			t.trava.Lock()
			lista = append(lista, t.campos())
			t.trava.Unlock()
		}
		p.Responde(base.Campos{"tarefas": lista})
	})
	s.AoFechar(tr.fecha)
	go tr.laco()
}

func falha(p *base.Pedido, err error) {
	switch e := err.(type) {
	case *Recusa:
		p.Recusa(e.Frase)
	case *DeCampo:
		p.FalhaDeCampo(e.Campo, e.Frase)
	case *ComCampos:
		p.Falha(e.Frase, e.Mais)
	default:
		p.Falha(err.Error())
	}
}

func (tr *Trilho) viva(id int) *Tarefa {
	tr.trava.Lock()
	defer tr.trava.Unlock()
	return tr.vivas[id]
}

func (tr *Trilho) enfileira(acao string, trabalho Trabalho, modo int, reg *registro) *Tarefa {
	tr.trava.Lock()
	if tr.fechado {
		tr.trava.Unlock()
		return nil
	}
	tr.proxima++
	t := &Tarefa{ID: tr.proxima, Acao: acao, tr: tr, trabalho: trabalho, modo: modo, reg: reg, estado: "fila",
		parar: make(chan struct{}), respostas: make(chan Resposta, 1)}
	tr.vivas[t.ID] = t
	tr.fila = append(tr.fila, t)
	tr.s.Conta()
	tr.trava.Unlock()
	t.avisa()
	select {
	case tr.chegou <- struct{}{}:
	default:
	}
	return t
}

// laco roda as tarefas, uma por vez, até a sessão fechar.
func (tr *Trilho) laco() {
	for {
		tr.trava.Lock()
		var t *Tarefa
		if len(tr.fila) > 0 {
			t, tr.fila = tr.fila[0], tr.fila[1:]
		}
		tr.trava.Unlock()
		if t == nil {
			select {
			case <-tr.chegou:
				continue
			case <-tr.fechou:
				return
			}
		}
		tr.roda(t)
	}
}

func (tr *Trilho) roda(t *Tarefa) {
	defer tr.s.Descontou()
	if t.Segue() {
		t.muda("rodando")
		t.trava.Lock()
		t.inicio = time.Now()
		t.marca = t.inicio
		t.trava.Unlock()
		t.trabalho.Prepara(t)
		t.avisa()
		t.trabalho.Roda(t)
	}
	t.trava.Lock()
	cancelada := t.Cancelada()
	comErro := len(t.erros) > 0
	switch {
	case cancelada:
		t.estado = "cancelada"
	case comErro:
		t.estado = "erro"
	default:
		t.estado = "feita"
	}
	t.final = true
	feitos := t.feitos
	refazivel, eRefazivel := t.trabalho.(Refazivel)
	t.desfazivel = t.modo != desfazendo && eRefazivel && len(feitos) > 0 && !t.naoDesfaz
	desfazivel := t.desfazivel
	t.trava.Unlock()

	tr.trava.Lock()
	delete(tr.vivas, t.ID)
	mudou := false
	switch {
	case t.modo == desfazendo:
		// O que foi desfeito inteiro pode ser refeito.
		if !comErro && !cancelada && t.reg != nil && t.reg.refaz != nil {
			tr.refazer = append(tr.refazer, t.reg)
			mudou = true
		}
	case desfazivel:
		tr.desfazer = append(tr.desfazer, &registro{descricao: refazivel.Descricao(), feitos: feitos, refaz: refazivel.Refaz})
		if t.modo == normal {
			tr.refazer = nil
		}
		mudou = true
	}
	tr.trava.Unlock()
	if comErro {
		tr.s.Marca(base.Erro)
	}
	t.avisa()
	if mudou {
		tr.avisaHistorico()
	}
}

func (tr *Trilho) fecha() {
	tr.trava.Lock()
	if tr.fechado {
		tr.trava.Unlock()
		return
	}
	tr.fechado = true
	vivas := make([]*Tarefa, 0, len(tr.vivas))
	for _, t := range tr.vivas {
		vivas = append(vivas, t)
	}
	tr.trava.Unlock()
	for _, t := range vivas {
		t.cancela()
	}
	// O laço sai depois de esvaziar a fila (cada tarefa cancelada acaba na hora).
	go func() {
		tr.s.Espera()
		close(tr.fechou)
	}()
}

// avisaHistorico solta o evento com o topo das duas pilhas. A leitura e a
// escrita são uma coisa só (debaixo da trava): dois avisos quase juntos (o do
// pedido de desfazer e o da tarefa dele que já acabou) saem na ordem do que
// aconteceu, e o último evento é sempre o estado de agora.
func (tr *Trilho) avisaHistorico() {
	tr.trava.Lock()
	defer tr.trava.Unlock()
	topo := func(pilha []*registro) any {
		if len(pilha) == 0 {
			return nil
		}
		return pilha[len(pilha)-1].descricao
	}
	tr.s.Evento("historico", base.Campos{"desfazer": topo(tr.desfazer), "refazer": topo(tr.refazer)})
}

func (tr *Trilho) avisaLixeira() {
	tr.s.Evento("lixeira", base.Campos{"itens": lixeira.Contar()})
}

// volta atende desfazer e refazer: tira o topo da pilha e cria a tarefa.
func (tr *Trilho) volta(p *base.Pedido, modo int) {
	tr.trava.Lock()
	pilha := &tr.desfazer
	if modo == refazendo {
		pilha = &tr.refazer
	}
	if len(*pilha) == 0 {
		tr.trava.Unlock()
		if modo == refazendo {
			p.Falha("nada para refazer")
		} else {
			p.Falha("nada para desfazer")
		}
		return
	}
	reg := (*pilha)[len(*pilha)-1]
	*pilha = (*pilha)[:len(*pilha)-1]
	tr.trava.Unlock()
	var trabalho Trabalho
	if modo == refazendo {
		trabalho = reg.refaz()
	} else {
		trabalho = &desfaz{reg: reg}
	}
	t := tr.enfileira(p.Op, trabalho, modo, reg)
	if t == nil {
		p.Falha("o motor está fechando")
		return
	}
	tr.avisaHistorico()
	p.Responde(base.Campos{"tarefa": t.ID, "descricao": reg.descricao})
}

func (tr *Trilho) responder(p *base.Pedido) {
	var q struct {
		Tarefa   int     `json:"tarefa"`
		Conflito int     `json:"conflito"`
		Resposta string  `json:"resposta"`
		Nome     *string `json:"nome"`
		Todos    bool    `json:"todos"`
	}
	_ = p.Ler(&q)
	r := Resposta{Qual: q.Resposta, todos: q.Todos}
	switch q.Resposta {
	case "substituir", "pular", "manter", "cancelar":
	case "renomear":
		if q.Nome == nil {
			p.FalhaDeCampo("nome", "falta o nome novo")
			return
		}
		if err := base.Nome(*q.Nome); err != nil {
			p.FalhaDeCampo("nome", err.Error())
			return
		}
		r.Nome = *q.Nome
	default:
		p.Falha("não conheço a resposta «" + q.Resposta + "»")
		return
	}
	t := tr.viva(q.Tarefa)
	if t == nil {
		p.Falha("essa pergunta não está mais de pé")
		return
	}
	t.trava.Lock()
	serve := t.esperando && t.conflito == q.Conflito
	if serve {
		t.esperando = false
	}
	t.trava.Unlock()
	if !serve {
		p.Falha("essa pergunta não está mais de pé")
		return
	}
	t.respostas <- r
	p.Responde(base.Campos{"tarefa": t.ID})
}

// campos monta o evento "tarefa" (com a trava da tarefa na mão).
func (t *Tarefa) campos() base.Campos {
	lista := func(v []string) []string {
		if v == nil {
			return []string{}
		}
		return append([]string(nil), v...)
	}
	erros := append([]base.Campos{}, t.erros...)
	var destino any
	if d := t.trabalho.Destino(); d != "" {
		destino = d
	}
	var atual any
	if t.atual != "" {
		atual = t.atual
	}
	estado := t.estado
	if t.pausada && !t.final && estado == "rodando" {
		estado = "pausada"
	}
	return base.Campos{"tarefa": t.ID, "acao": t.Acao, "estado": estado, "titulo": t.trabalho.Titulo(),
		"itens": t.itens, "itens_total": t.itensTotal, "bytes": t.bytes, "bytes_total": t.bytesTotal,
		"atual": atual, "velocidade": t.velocidade, "destino": destino,
		"criados": lista(t.criados), "erros": erros, "sem_lixeira": lista(t.semLixeira), "desfazivel": t.desfazivel}
}

// avisa solta o evento "tarefa" agora.
func (t *Tarefa) avisa() {
	t.trava.Lock()
	t.mede(time.Now())
	t.ultimoEvento = time.Now()
	campos := t.campos()
	t.trava.Unlock()
	t.tr.s.Evento("tarefa", campos)
}

// mede atualiza a velocidade: a média desde a última marca, que anda a cada
// segundo.
func (t *Tarefa) mede(agora time.Time) {
	if t.marca.IsZero() {
		return
	}
	passou := agora.Sub(t.marca)
	if passou >= time.Second || (t.velocidade == 0 && passou >= intervalo) {
		t.velocidade = int64(float64(t.bytes-t.bytesNaMarca) / passou.Seconds())
		if passou >= time.Second {
			t.marca, t.bytesNaMarca = agora, t.bytes
		}
	}
	if t.final {
		t.velocidade = 0
	}
}

func (t *Tarefa) muda(estado string) {
	t.trava.Lock()
	t.estado = estado
	t.trava.Unlock()
	t.avisa()
}

func (t *Tarefa) pausa(pausar bool) {
	t.trava.Lock()
	if t.pausada == pausar || t.final {
		t.trava.Unlock()
		return
	}
	t.pausada = pausar
	if pausar {
		t.retoma = make(chan struct{})
	} else {
		close(t.retoma)
	}
	t.trava.Unlock()
	t.avisa()
}

func (t *Tarefa) cancela() {
	t.umaVez.Do(func() { close(t.parar) })
}

// Cancelada diz se a tarefa foi cancelada.
func (t *Tarefa) Cancelada() bool {
	select {
	case <-t.parar:
		return true
	default:
		return false
	}
}

// Segue diz se o trabalho continua: espera enquanto a tarefa está pausada e
// devolve false quando ela foi cancelada. O trabalho pergunta entre um passo
// e outro.
func (t *Tarefa) Segue() bool {
	for {
		t.trava.Lock()
		pausada, retoma := t.pausada, t.retoma
		t.trava.Unlock()
		if t.Cancelada() {
			return false
		}
		if !pausada {
			return true
		}
		select {
		case <-retoma:
		case <-t.parar:
			return false
		}
	}
}

// Total diz quanto há para fazer (soma ao que já foi dito).
func (t *Tarefa) Total(itens int, bytes int64) {
	t.trava.Lock()
	t.itensTotal += itens
	t.bytesTotal += bytes
	t.trava.Unlock()
}

// Avanca conta itens e bytes feitos e diz em que arquivo a tarefa está; o
// evento de andamento sai no máximo a cada 100 ms.
func (t *Tarefa) Avanca(itens int, bytes int64, atual string) {
	t.trava.Lock()
	t.itens += itens
	t.bytes += bytes
	if atual != "" {
		t.atual = atual
	}
	sai := time.Since(t.ultimoEvento) >= intervalo
	t.trava.Unlock()
	if sai {
		t.avisa()
	}
}

// Criou anota um caminho de topo que a tarefa criou (a janela o seleciona).
func (t *Tarefa) Criou(caminho string) {
	t.trava.Lock()
	t.criados = append(t.criados, caminho)
	t.trava.Unlock()
}

// Erro anota que um caminho falhou; a tarefa segue com os outros.
func (t *Tarefa) Erro(caminho string, frase string) {
	t.trava.Lock()
	t.erros = append(t.erros, base.Campos{"caminho": caminho, "erro": frase})
	t.trava.Unlock()
}

func (t *Tarefa) nErros() int {
	t.trava.Lock()
	defer t.trava.Unlock()
	return len(t.erros)
}

// SemLixeira anota um caminho que não tem lixeira possível.
func (t *Tarefa) SemLixeira(caminho string) {
	t.trava.Lock()
	t.semLixeira = append(t.semLixeira, caminho)
	t.trava.Unlock()
}

// NaoDesfaz marca que a tarefa fez algo sem volta (substituiu um arquivo).
func (t *Tarefa) NaoDesfaz() {
	t.trava.Lock()
	t.naoDesfaz = true
	t.trava.Unlock()
}

func (t *Tarefa) fez(f feito) {
	t.trava.Lock()
	t.feitos = append(t.feitos, f)
	t.trava.Unlock()
}

// Conflito pergunta à janela o que fazer com um destino que já existe, e
// espera a resposta (ou o cancelamento). Na chamada de um subcomando só não
// há quem responda: o item é pulado e entra nos erros.
func (t *Tarefa) Conflito(origem, destino string, sugestao string) Resposta {
	if t.tr.s.SemID {
		t.Erro(destino, "já existe um item com esse nome")
		return Resposta{Qual: "pular"}
	}
	t.trava.Lock()
	if t.paraTodos != "" {
		r := Resposta{Qual: t.paraTodos}
		t.trava.Unlock()
		return r
	}
	t.conflito++
	numero := t.conflito
	t.esperando = true
	t.estado = "conflito"
	restam := t.itensTotal - t.itens
	t.trava.Unlock()
	t.avisa()
	campos := base.Campos{"tarefa": t.ID, "conflito": numero, "acao": t.Acao, "sugestao": sugestao, "restam": max(restam, 0),
		"origem": nil, "destino": nil}
	if it, err := item.De(origem, false); err == nil {
		campos["origem"] = it
	}
	if it, err := item.De(destino, false); err == nil {
		campos["destino"] = it
	}
	t.tr.s.Evento("conflito", campos)
	var r Resposta
	select {
	case r = <-t.respostas:
	case <-t.parar:
		r = Resposta{Qual: "cancelar"}
	}
	t.trava.Lock()
	t.esperando = false
	t.estado = "rodando"
	if r.todos && r.Qual != "renomear" && r.Qual != "cancelar" {
		t.paraTodos = r.Qual
	}
	t.trava.Unlock()
	if r.Qual == "cancelar" {
		t.cancela()
	}
	t.avisa()
	return r
}
