package base

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"sync"
)

// Campos é uma resposta ou um evento: os campos do contrato.
type Campos map[string]any

// Sessao é uma conversa com a janela (o modo servir) ou uma chamada só. Cada
// área do motor se liga a ela (Ligar) com os pedidos que atende, escreve por
// ela (Responde, Evento) e pendura nela o que roda por fora (Vai).
type Sessao struct {
	// SemID: as linhas saem sem "id" (a chamada de um subcomando só).
	SemID bool

	saida   io.Writer
	trava   sync.Mutex // da saída e do estado abaixo
	ops     map[string]func(*Pedido)
	fluxos  map[string]*Pedido
	fechar  []func()
	codigo  int
	rodando sync.WaitGroup
	fechada bool
}

// NovaSessao cria a sessão que escreve em saida. O pedido "cancelar" já vem.
func NovaSessao(saida io.Writer) *Sessao {
	s := &Sessao{saida: saida, ops: map[string]func(*Pedido){}, fluxos: map[string]*Pedido{}}
	s.Ligar("cancelar", func(p *Pedido) {
		var q struct {
			Pedido json.RawMessage `json:"pedido"`
		}
		_ = p.Ler(&q)
		s.trava.Lock()
		alvo := s.fluxos[string(bytes.TrimSpace(q.Pedido))]
		s.trava.Unlock()
		if alvo != nil {
			alvo.cancelar()
		}
		p.Responde(Campos{"cancelado": alvo != nil})
	})
	return s
}

// Ligar registra quem atende um pedido. Ligar duas vezes o mesmo é erro de
// programação.
func (s *Sessao) Ligar(op string, fn func(*Pedido)) {
	if _, ja := s.ops[op]; ja {
		panic("base: pedido ligado duas vezes: " + op)
	}
	s.ops[op] = fn
}

// Ops são os pedidos ligados, em ordem.
func (s *Sessao) Ops() []string {
	var ops []string
	for op := range s.ops {
		ops = append(ops, op)
	}
	sort.Strings(ops)
	return ops
}

// AoFechar pendura o que roda quando a sessão acaba (parar tarefas, vigias).
func (s *Sessao) AoFechar(fn func()) {
	s.trava.Lock()
	s.fechar = append(s.fechar, fn)
	s.trava.Unlock()
}

// Fecha avisa todo mundo que acabou (cancela os fluxos, roda os AoFechar) e
// espera o que estava rodando.
func (s *Sessao) Fecha() {
	s.trava.Lock()
	if s.fechada {
		s.trava.Unlock()
		return
	}
	s.fechada = true
	fechar := s.fechar
	var abertos []*Pedido
	for _, p := range s.fluxos {
		abertos = append(abertos, p)
	}
	s.trava.Unlock()
	for _, p := range abertos {
		p.cancelar()
	}
	for _, fn := range fechar {
		fn()
	}
	s.rodando.Wait()
}

// Vai roda fn por fora, contada: Espera e Fecha aguardam.
func (s *Sessao) Vai(fn func()) {
	s.rodando.Add(1)
	go func() {
		defer s.rodando.Done()
		fn()
	}()
}

// Conta e Descontou são para quem cuida da própria goroutine de vida longa mas
// quer que a chamada de um subcomando só espere um trabalho acabar (a fila de
// tarefas): Conta ao enfileirar, Descontou ao terminar.
func (s *Sessao) Conta()     { s.rodando.Add(1) }
func (s *Sessao) Descontou() { s.rodando.Done() }

// Espera aguarda os fluxos e os trabalhos contados acabarem.
func (s *Sessao) Espera() { s.rodando.Wait() }

// Codigo é o pior código de saída visto: 2 se houve recusa, 1 se houve erro.
func (s *Sessao) Codigo() int {
	s.trava.Lock()
	defer s.trava.Unlock()
	return s.codigo
}

// Marca anota um código de saída (uma tarefa que acabou em erro, por exemplo).
func (s *Sessao) Marca(codigo int) {
	s.trava.Lock()
	if codigo > s.codigo {
		s.codigo = codigo
	}
	s.trava.Unlock()
}

// escreve solta uma linha de JSON, inteira, sem escapar acentos nem <>.
func (s *Sessao) escreve(v any) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		b.Reset()
		fmt.Fprintf(&b, "{\"erro\":%q}\n", "resposta que não coube em JSON: "+err.Error())
	}
	s.trava.Lock()
	defer s.trava.Unlock()
	_, _ = s.saida.Write(b.Bytes())
}

// Evento solta {"evento": nome, …campos}.
func (s *Sessao) Evento(nome string, campos Campos) {
	linha := Campos{"evento": nome}
	for k, v := range campos {
		linha[k] = v
	}
	s.escreve(linha)
}

// Atender recebe uma linha de pedido e chama quem a atende, na hora (quem
// demora roda por fora, com Fluxo ou Vai).
func (s *Sessao) Atender(linha []byte) {
	var cabeca struct {
		ID json.RawMessage `json:"id"`
		Op string          `json:"op"`
	}
	if err := json.Unmarshal(linha, &cabeca); err != nil {
		s.Marca(Erro)
		linha := Campos{"erro": "pedido que não é JSON"}
		if !s.SemID {
			linha["id"] = nil
		}
		s.escreve(linha)
		return
	}
	p := &Pedido{S: s, Op: cabeca.Op, Cru: append([]byte(nil), linha...), parar: make(chan struct{})}
	p.ID, p.chave = idDe(cabeca.ID)
	fn := s.ops[cabeca.Op]
	if fn == nil {
		p.Falha("não conheço o pedido «" + cabeca.Op + "»")
		return
	}
	fn(p)
}

// idDe devolve o id como veio, se é um número ou um texto; o resto vira null.
func idDe(cru json.RawMessage) (any, string) {
	var v any
	d := json.NewDecoder(bytes.NewReader(cru))
	d.UseNumber()
	if len(cru) == 0 || d.Decode(&v) != nil {
		return nil, ""
	}
	switch v.(type) {
	case json.Number, string:
		return v, string(bytes.TrimSpace(cru))
	}
	return nil, ""
}

// Pedido é um pedido da janela: quem o atende lê os campos (Ler) e responde
// (Responde, Falha, Recusa), uma vez ou em fluxo.
type Pedido struct {
	S   *Sessao
	ID  any
	Op  string
	Cru []byte

	chave string
	parar chan struct{}
	uma   sync.Once
}

// Ler põe os campos do pedido em v (uma struct com as etiquetas json).
func (p *Pedido) Ler(v any) error {
	return json.Unmarshal(p.Cru, v)
}

// Responde solta uma linha de resposta, com o id do pedido.
func (p *Pedido) Responde(campos Campos) {
	linha := Campos{}
	for k, v := range campos {
		linha[k] = v
	}
	if !p.S.SemID {
		linha["id"] = p.ID
	}
	p.S.escreve(linha)
}

// Falha responde {"erro": frase} (e mais campos, se vierem).
func (p *Pedido) Falha(frase string, mais ...Campos) {
	p.S.Marca(Erro)
	linha := Campos{"erro": frase}
	for _, m := range mais {
		for k, v := range m {
			linha[k] = v
		}
	}
	p.Responde(linha)
}

// FalhaDeCampo é o erro de formulário: diz qual campo está errado.
func (p *Pedido) FalhaDeCampo(campo, frase string) {
	p.Falha(frase, Campos{"campo": campo})
}

// Recusa é a recusa de segurança: {"erro": frase, "recusado": true}.
func (p *Pedido) Recusa(frase string) {
	p.S.Marca(Recusado)
	p.Responde(Campos{"erro": frase, "recusado": true})
}

// Fluxo roda fn por fora (a resposta sai em várias linhas) e deixa o pedido
// cancelável: fn consulta cancelado() entre um passo e outro, e é ela quem
// solta a última linha ("fim": true).
func (p *Pedido) Fluxo(fn func(cancelado func() bool)) {
	s := p.S
	s.trava.Lock()
	if s.fechada {
		s.trava.Unlock()
		p.Responde(Campos{"fim": true, "cancelada": true})
		return
	}
	if p.chave != "" {
		s.fluxos[p.chave] = p
	}
	s.trava.Unlock()
	s.Vai(func() {
		defer func() {
			s.trava.Lock()
			if s.fluxos[p.chave] == p {
				delete(s.fluxos, p.chave)
			}
			s.trava.Unlock()
		}()
		fn(p.Cancelado)
	})
}

// Cancelado diz se pediram para parar este fluxo.
func (p *Pedido) Cancelado() bool {
	select {
	case <-p.parar:
		return true
	default:
		return false
	}
}

// Parou é o canal que fecha quando o fluxo é cancelado (para um select).
func (p *Pedido) Parou() <-chan struct{} { return p.parar }

func (p *Pedido) cancelar() { p.uma.Do(func() { close(p.parar) }) }
