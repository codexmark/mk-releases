package remoto

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"mkfiles/internal/base"
)

// area é o estado dos remotos numa sessão: as montagens em andamento (para o
// remoto-responder achar a pergunta) e o vigia da lista de montados.
type area struct {
	s        *base.Sessao
	trava    sync.Mutex
	montando map[string]*montagem // id do pedido -> montagem
	vigiando bool
	para     chan struct{}
}

// Ligar põe na sessão os pedidos dos remotos.
func Ligar(s *base.Sessao) {
	a := &area{s: s, montando: map[string]*montagem{}, para: make(chan struct{})}
	s.Ligar("remoto-hosts", func(p *base.Pedido) { p.Responde(base.Campos{"hosts": Hosts()}) })
	s.Ligar("remoto-montar", a.montar)
	s.Ligar("remoto-responder", a.responder)
	s.Ligar("remotos", a.listar)
	s.Ligar("remoto-desmontar", a.desmontar)
	s.AoFechar(func() { close(a.para) })
}

// chave escreve o id de um pedido como texto, para servir de chave.
func chave(id any) string { return fmt.Sprint(id) }

func (a *area) montar(p *base.Pedido) {
	var q struct {
		URI     string `json:"uri"`
		Anonimo bool   `json:"anonimo"`
	}
	_ = p.Ler(&q)
	e, err := Abre(q.URI)
	if err != nil {
		p.Falha(err.Error(), base.Campos{"motivo": "endereco", "saida": nil})
		return
	}
	if base.GVFS() == "" {
		p.Falha("não há onde o gvfs montar os remotos (falta XDG_RUNTIME_DIR)", base.Campos{"motivo": "falta", "saida": nil})
		return
	}
	uri := e.URI()
	pronto := func(saida string) bool {
		raiz, caminho, achou := Onde(e)
		if !achou {
			return false
		}
		// O caminho pedido pode não existir lá: então fica a raiz.
		if _, err := os.Stat(caminho); err != nil {
			caminho = raiz
		}
		p.Responde(base.Campos{"uri": uri, "caminho": caminho, "raiz": raiz, "nome": base.NomeDoRemoto(path.Base(raiz))})
		a.avisa()
		return true
	}
	if pronto("") {
		return // já estava montado
	}
	m := &montagem{}
	id := chave(p.ID)
	a.trava.Lock()
	a.montando[id] = m
	a.trava.Unlock()
	p.Fluxo(func(func() bool) {
		defer func() {
			a.trava.Lock()
			delete(a.montando, id)
			a.trava.Unlock()
		}()
		r := m.monta(uri, q.Anonimo, p.Parou(), func(numero int, tipo, texto, padrao string, deNovo bool) {
			campos := base.Campos{"pedido": p.ID, "pergunta": numero, "tipo": tipo, "texto": texto, "padrao": nil, "de_novo": deNovo}
			if padrao != "" {
				campos["padrao"] = padrao
			}
			p.S.Evento("pergunta-remota", campos)
		})
		// O gio pode sair com erro dizendo que já estava montado: vale o que
		// está na pasta do gvfs.
		if r.motivo == "" || r.motivo == "erro" {
			// O FUSE pode levar um instante para mostrar a pasta.
			for tentativa := 0; tentativa < 20; tentativa++ {
				if pronto(r.saida) {
					return
				}
				if r.motivo != "" {
					break
				}
				select {
				case <-p.Parou():
					tentativa = 20
				case <-time.After(100 * time.Millisecond):
				}
			}
			if r.motivo == "" {
				r.motivo, r.frase = "erro", "o gio disse que montou, mas o remoto não apareceu em "+base.GVFS()
			}
		}
		var saida any
		if r.saida != "" {
			saida = r.saida
		}
		p.Falha(r.frase, base.Campos{"motivo": r.motivo, "saida": saida})
	})
}

func (a *area) responder(p *base.Pedido) {
	var q struct {
		Pedido   json.RawMessage `json:"pedido"`
		Pergunta int             `json:"pergunta"`
		Resposta *string         `json:"resposta"`
	}
	if err := p.Ler(&q); err != nil {
		p.Falha("resposta mal formada")
		return
	}
	var id any
	d := json.NewDecoder(bytes.NewReader(q.Pedido))
	d.UseNumber()
	_ = d.Decode(&id)
	a.trava.Lock()
	m := a.montando[chave(id)]
	a.trava.Unlock()
	if m == nil || !m.responde(q.Pergunta, q.Resposta) {
		p.Falha("essa pergunta não está mais esperando resposta")
		return
	}
	p.Responde(base.Campos{"respondida": true})
}

// rede pergunta ao gvfs o que há na rede local: `gio list -u network:///`
// (uma URI por linha). Se não der, lista vazia.
func rede() []base.Campos {
	lista := []base.Campos{}
	saida, err := roda(4*time.Second, "list", "-u", "network:///")
	if err != nil {
		return lista
	}
	for _, linha := range strings.Split(saida, "\n") {
		uri := strings.TrimSpace(linha)
		if !strings.Contains(uri, "://") || strings.ContainsAny(uri, " \t") {
			continue
		}
		nome := strings.TrimRight(uri, "/")
		nome = nome[strings.LastIndexByte(nome, '/')+1:]
		if n, err := url.PathUnescape(nome); err == nil {
			nome = n
		}
		lista = append(lista, base.Campos{"uri": uri, "nome": nome})
	}
	return lista
}

func (a *area) listar(p *base.Pedido) {
	var q struct {
		Rede bool `json:"rede"`
	}
	_ = p.Ler(&q)
	a.vigia()
	if !q.Rede {
		p.Responde(base.Campos{"remotos": Montados(), "rede": nil})
		return
	}
	// Perguntar à rede pode demorar: por fora.
	p.S.Vai(func() { p.Responde(base.Campos{"remotos": Montados(), "rede": rede()}) })
}

func (a *area) desmontar(p *base.Pedido) {
	var q struct {
		URI string `json:"uri"`
	}
	_ = p.Ler(&q)
	// Só se desmonta o que está na lista dos montados: o texto recebido nunca
	// vai direto para o comando.
	var alvo string
	for _, m := range Montados() {
		if m.URI != nil && *m.URI == q.URI {
			alvo = *m.URI
		}
	}
	if alvo == "" {
		p.Falha("esse remoto não está montado")
		return
	}
	p.S.Vai(func() {
		if saida, err := roda(20*time.Second, "mount", "-u", alvo); err != nil {
			frase := "não deu para desmontar"
			if saida != "" {
				frase += ": " + saida[strings.LastIndexByte(saida, '\n')+1:]
			}
			p.Falha(frase)
			return
		}
		// A pasta pode levar um instante para sumir.
		for i := 0; i < 20; i++ {
			if _, _, ainda := Onde(mustAbre(alvo)); !ainda {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		p.Responde(base.Campos{"remotos": Montados()})
		a.avisa()
	})
}

func mustAbre(uri string) Endereco {
	e, _ := Abre(uri)
	return e
}

// avisa solta o evento com a lista dos montados.
func (a *area) avisa() {
	a.s.Evento("remotos", base.Campos{"remotos": Montados()})
}

// intervalo é o relógio dos discos (MKFILES_DISCOS_MS): os remotos montados
// são conferidos no mesmo passo.
func intervalo() time.Duration {
	if n, err := strconv.Atoi(os.Getenv("MKFILES_DISCOS_MS")); err == nil && n >= 10 {
		return time.Duration(n) * time.Millisecond
	}
	return 2 * time.Second
}

// vigia começa (uma vez, e só no modo servir) a conferir a pasta do gvfs: um
// celular ligado, um remoto montado por outro programa ou que caiu mudam a
// lista sem passar por aqui.
func (a *area) vigia() {
	a.trava.Lock()
	ja := a.vigiando
	a.vigiando = true
	a.trava.Unlock()
	if ja || a.s.SemID {
		return
	}
	retrato := func() string {
		b, _ := json.Marshal(Montados())
		return string(b)
	}
	go func() {
		antes := retrato()
		relogio := time.NewTicker(intervalo())
		defer relogio.Stop()
		for {
			select {
			case <-a.para:
				return
			case <-relogio.C:
			}
			if agora := retrato(); agora != antes {
				antes = agora
				a.avisa()
			}
		}
	}()
}
