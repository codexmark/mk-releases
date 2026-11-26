// SPDX-License-Identifier: GPL-3.0-or-later

package dbus

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	nomeDoBarramento    = "org.freedesktop.DBus"
	caminhoDoBarramento = Caminho("/org/freedesktop/DBus")
	// prazoDaConversa vale para o aperto de mão e para cada chamada ao barramento.
	prazoDaConversa = 5 * time.Second
)

// ErrSemBarramento: não há barramento de sessão (a variável não existe).
var ErrSemBarramento = errors.New("dbus: sem barramento de sessão")

// EnderecosDe lê DBUS_SESSION_BUS_ADDRESS e devolve os sockets a tentar, na
// ordem. Só o transporte unix é conhecido (path, abstract e runtime).
func EnderecosDe(variavel, runtime string) []string {
	var sockets []string
	for _, endereco := range strings.Split(variavel, ";") {
		transporte, resto, tem := strings.Cut(endereco, ":")
		if !tem || transporte != "unix" {
			continue
		}
		for _, parte := range strings.Split(resto, ",") {
			chave, valor, _ := strings.Cut(parte, "=")
			valor, erro := url.PathUnescape(valor)
			if erro != nil {
				continue
			}
			switch {
			case chave == "path" && valor != "":
				sockets = append(sockets, valor)
			case chave == "abstract" && valor != "":
				sockets = append(sockets, "@"+valor)
			case chave == "runtime" && valor == "yes" && runtime != "":
				sockets = append(sockets, runtime+"/bus")
			}
		}
	}
	return sockets
}

// Conexao é uma conexão com o barramento. Envia pode ser chamada de qualquer
// goroutine; as mensagens que chegam vão para a função dada em Atende.
type Conexao struct {
	fio     net.Conn
	leitura *bufio.Reader
	escrita sync.Mutex
	serial  uint32
	// Nome é o nome único que o barramento deu (":1.42").
	Nome string

	mu       sync.Mutex
	esperas  map[uint32]chan *Mensagem
	fechada  bool
	terminou chan struct{}
}

// Liga conecta ao barramento de sessão, se apresenta (Hello) e começa a
// atender: as chamadas de método que chegarem vão para aoChamar (fora do
// laço; quem recebe decide onde tratar).
func Liga(aoChamar func(m *Mensagem)) (*Conexao, error) {
	sockets := EnderecosDe(os.Getenv("DBUS_SESSION_BUS_ADDRESS"), os.Getenv("XDG_RUNTIME_DIR"))
	if len(sockets) == 0 {
		return nil, ErrSemBarramento
	}
	var ultimo error
	for _, socket := range sockets {
		c, erro := LigaEm(socket, aoChamar)
		if erro == nil {
			return c, nil
		}
		ultimo = erro
	}
	return nil, ultimo
}

// LigaEm conecta a um socket Unix (um caminho, ou "@nome" para os abstratos).
func LigaEm(socket string, aoChamar func(m *Mensagem)) (*Conexao, error) {
	fio, erro := net.DialTimeout("unix", socket, prazoDaConversa)
	if erro != nil {
		return nil, erro
	}
	c := &Conexao{fio: fio, leitura: bufio.NewReader(fio), esperas: map[uint32]chan *Mensagem{}, terminou: make(chan struct{})}
	if erro := c.autentica(); erro != nil {
		_ = fio.Close()
		return nil, erro
	}
	go c.le(aoChamar)
	ola, erro := c.Chama(nomeDoBarramento, caminhoDoBarramento, nomeDoBarramento, "Hello", "")
	if erro != nil {
		c.Fecha()
		return nil, erro
	}
	if len(ola.Corpo) == 1 {
		c.Nome, _ = ola.Corpo[0].(string)
	}
	return c, nil
}

// autentica faz o aperto de mão: um byte nulo, AUTH EXTERNAL com o uid em
// hexadecimal, e BEGIN depois do OK do servidor.
func (c *Conexao) autentica() error {
	_ = c.fio.SetDeadline(time.Now().Add(prazoDaConversa))
	defer c.fio.SetDeadline(time.Time{})
	uid := hex.EncodeToString([]byte(strconv.Itoa(os.Getuid())))
	if _, erro := c.fio.Write([]byte("\x00AUTH EXTERNAL " + uid + "\r\n")); erro != nil {
		return erro
	}
	linha, erro := c.leitura.ReadString('\n')
	if erro != nil {
		return erro
	}
	if !strings.HasPrefix(linha, "OK ") && strings.TrimSpace(linha) != "OK" {
		return fmt.Errorf("dbus: o barramento recusou a autenticação: %s", strings.TrimSpace(linha))
	}
	_, erro = c.fio.Write([]byte("BEGIN\r\n"))
	return erro
}

func (c *Conexao) le(aoChamar func(m *Mensagem)) {
	defer close(c.terminou)
	for {
		m, erro := LeMensagem(c.leitura)
		if erro != nil {
			c.mu.Lock()
			c.fechada = true
			for serial, espera := range c.esperas {
				close(espera)
				delete(c.esperas, serial)
			}
			c.mu.Unlock()
			return
		}
		switch m.Tipo {
		case Resposta, Erro:
			c.mu.Lock()
			espera, tem := c.esperas[m.RespostaA]
			delete(c.esperas, m.RespostaA)
			c.mu.Unlock()
			if tem {
				espera <- m
			}
		case Chamada:
			if aoChamar != nil {
				aoChamar(m)
			}
		}
	}
}

// Terminou fecha quando a conversa acaba (o barramento caiu, ou Fecha).
func (c *Conexao) Terminou() <-chan struct{} { return c.terminou }

// Envia põe uma mensagem no fio, dando a ela o próximo número de série.
func (c *Conexao) Envia(m *Mensagem) (uint32, error) { return c.envia(m, nil) }

// envia chama registra (com o número de série) antes de a mensagem sair: quem
// espera a resposta já está na lista quando ela chega.
func (c *Conexao) envia(m *Mensagem, registra func(serial uint32) error) (uint32, error) {
	c.escrita.Lock()
	defer c.escrita.Unlock()
	m.Serial = c.serial + 1
	dados, erro := m.Codifica()
	if erro != nil {
		return 0, erro
	}
	c.serial++
	if registra != nil {
		if erro := registra(m.Serial); erro != nil {
			return 0, erro
		}
	}
	_ = c.fio.SetWriteDeadline(time.Now().Add(prazoDaConversa))
	_, erro = c.fio.Write(dados)
	return m.Serial, erro
}

// Chama faz uma chamada de método e espera a resposta. Não pode ser usada de
// dentro de aoChamar (é a mesma goroutine que leria a resposta).
func (c *Conexao) Chama(destino string, caminho Caminho, iface, membro, assinatura string, valores ...any) (*Mensagem, error) {
	espera := make(chan *Mensagem, 1)
	pedido := &Mensagem{Tipo: Chamada, Destino: destino, Caminho: caminho, Interface: iface, Membro: membro, Assinatura: assinatura, Corpo: valores}
	proximo, erro := c.envia(pedido, func(serial uint32) error {
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.fechada {
			return errors.New("dbus: conexão fechada")
		}
		c.esperas[serial] = espera
		return nil
	})
	if erro != nil {
		c.mu.Lock()
		delete(c.esperas, pedido.Serial)
		c.mu.Unlock()
		return nil, erro
	}
	select {
	case resposta, veio := <-espera:
		if !veio {
			return nil, errors.New("dbus: o barramento fechou a conexão")
		}
		if resposta.Tipo == Erro {
			texto := resposta.NomeDoErro
			if len(resposta.Corpo) > 0 {
				if detalhe, ok := resposta.Corpo[0].(string); ok {
					texto += ": " + detalhe
				}
			}
			return nil, errors.New("dbus: " + texto)
		}
		return resposta, nil
	case <-time.After(prazoDaConversa):
		c.mu.Lock()
		delete(c.esperas, proximo)
		c.mu.Unlock()
		return nil, errors.New("dbus: o barramento não respondeu a " + membro)
	}
}

// Marcas de RequestName e a resposta que diz "o nome é seu".
const (
	naoEntraNaFila = 4
	donoDoNome     = 1
)

// PedeNome pede um nome bem conhecido; false quando outro processo já o tem.
func (c *Conexao) PedeNome(nome string) (bool, error) {
	resposta, erro := c.Chama(nomeDoBarramento, caminhoDoBarramento, nomeDoBarramento, "RequestName", "su", nome, uint32(naoEntraNaFila))
	if erro != nil {
		return false, erro
	}
	codigo, _ := resposta.Corpo[0].(uint32)
	return len(resposta.Corpo) == 1 && codigo == donoDoNome, nil
}

// Responde devolve o resultado de uma chamada (nada, se quem chamou não quer
// resposta).
func (c *Conexao) Responde(a *Mensagem, assinatura string, valores ...any) {
	if a.Marcas&SemResposta != 0 {
		return
	}
	_, _ = c.Envia(&Mensagem{Tipo: Resposta, Destino: a.Remetente, RespostaA: a.Serial, Assinatura: assinatura, Corpo: valores})
}

// RespondeErro devolve um erro a uma chamada.
func (c *Conexao) RespondeErro(a *Mensagem, nome, texto string) {
	if a.Marcas&SemResposta != 0 {
		return
	}
	_, _ = c.Envia(&Mensagem{Tipo: Erro, Destino: a.Remetente, RespostaA: a.Serial, NomeDoErro: nome, Assinatura: "s", Corpo: []any{texto}})
}

// EmiteSinal manda um sinal a quem estiver ouvindo.
func (c *Conexao) EmiteSinal(caminho Caminho, iface, membro, assinatura string, valores ...any) error {
	_, erro := c.Envia(&Mensagem{Tipo: Sinal, Caminho: caminho, Interface: iface, Membro: membro, Assinatura: assinatura, Corpo: valores})
	return erro
}

// Fecha encerra a conexão (o barramento solta os nomes que ela tinha).
func (c *Conexao) Fecha() {
	_ = c.fio.Close()
	<-c.terminou
}
