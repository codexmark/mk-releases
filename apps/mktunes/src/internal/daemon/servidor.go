// SPDX-License-Identifier: GPL-3.0-or-later

package daemon

import (
	"bufio"
	"errors"
	"net"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"mktunes/internal/laco"
)

// ErrJaRodando: outro mktunesd já responde no socket.
var ErrJaRodando = errors.New("o mktunesd já está rodando")

// linhaMax é a maior linha aceita de uma janela. (A versão em Kotlin cortava
// em 64 KiB, o que estragava um "play" com uma lista grande; aqui cabe uma
// biblioteca inteira e ainda há teto.)
const linhaMax = 8 << 20

// Responde diz se alguém atende no socket.
func Responde(socket string) bool {
	conexao, erro := net.DialTimeout("unix", socket, time.Second)
	if erro != nil {
		return false
	}
	_ = conexao.Close()
	return true
}

// Servidor é a porta do daemon para as janelas: um objeto JSON por linha em
// cada sentido, num socket Unix que só o usuário abre.
type Servidor struct {
	socket  string
	escuta  net.Listener
	mu      sync.Mutex
	abertas map[net.Conn]bool
	fechado bool
}

// Serve abre o socket e passa a atender. Cada linha recebida vai para
// aoComando, no laço. Um arquivo de socket deixado por um daemon que morreu é
// substituído; se alguém responde nele, é ErrJaRodando.
func Serve(socket string, quadro *Quadro, l laco.Laco, aoComando func(linha string)) (*Servidor, error) {
	if _, erro := os.Lstat(socket); erro == nil {
		if Responde(socket) {
			return nil, ErrJaRodando
		}
		if erro := os.Remove(socket); erro != nil {
			return nil, erro
		}
	}
	if erro := os.MkdirAll(filepath.Dir(socket), 0o700); erro != nil {
		return nil, erro
	}
	// O socket já nasce fechado para os outros usuários.
	antiga := syscall.Umask(0o177)
	escuta, erro := net.Listen("unix", socket)
	syscall.Umask(antiga)
	if erro != nil {
		return nil, erro
	}
	if erro := os.Chmod(socket, 0o600); erro != nil {
		_ = escuta.Close()
		return nil, erro
	}
	s := &Servidor{socket: socket, escuta: escuta, abertas: map[net.Conn]bool{}}
	go s.aceita(quadro, l, aoComando)
	return s, nil
}

func (s *Servidor) aceita(quadro *Quadro, l laco.Laco, aoComando func(string)) {
	for {
		conexao, erro := s.escuta.Accept()
		if erro != nil {
			return
		}
		s.mu.Lock()
		if s.fechado {
			s.mu.Unlock()
			_ = conexao.Close()
			return
		}
		s.abertas[conexao] = true
		s.mu.Unlock()
		go s.atende(conexao, quadro, l, aoComando)
	}
}

func (s *Servidor) atende(conexao net.Conn, quadro *Quadro, l laco.Laco, aoComando func(string)) {
	pronto := make(chan *Assinante, 1)
	l.Posta(func() { pronto <- quadro.Assina() })
	assinante := <-pronto
	fim := make(chan struct{})
	go func() {
		// Quem escreve: tudo que o quadro puser na fila desta janela.
		for {
			select {
			case <-assinante.Chegou():
				for _, linha := range assinante.Tira() {
					_ = conexao.SetWriteDeadline(time.Now().Add(10 * time.Second))
					if _, erro := conexao.Write([]byte(linha + "\n")); erro != nil {
						// Uma janela travada (não lê há 10 s) é solta. Uma que já foi
						// embora pode ter deixado um comando por ler: quem lê decide.
						if errors.Is(erro, os.ErrDeadlineExceeded) {
							_ = conexao.Close()
						}
						return
					}
				}
			case <-fim:
				return
			}
		}
	}()
	linhas := bufio.NewScanner(conexao)
	linhas.Buffer(make([]byte, 64*1024), linhaMax)
	for linhas.Scan() {
		linha := linhas.Text()
		l.Posta(func() { aoComando(linha) })
	}
	// A janela foi embora (ou mandou uma linha grande demais).
	close(fim)
	_ = conexao.Close()
	l.Posta(func() { quadro.Desliga(assinante) })
	s.mu.Lock()
	delete(s.abertas, conexao)
	s.mu.Unlock()
}

// Fecha para de atender, solta as janelas e apaga o socket.
func (s *Servidor) Fecha() {
	s.mu.Lock()
	s.fechado = true
	for conexao := range s.abertas {
		_ = conexao.Close()
	}
	s.mu.Unlock()
	_ = s.escuta.Close()
	_ = os.Remove(s.socket)
}
