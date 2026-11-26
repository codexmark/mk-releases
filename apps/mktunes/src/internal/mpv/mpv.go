// SPDX-License-Identifier: GPL-3.0-or-later

// Pacote mpv: a saída de áudio do daemon é um mpv sem janela, falado por JSON
// no socket de --input-ipc-server. O mpv também é quem baixa o stream.
package mpv

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"mktunes/internal/app"
	"mktunes/internal/dominio"
)

// Argumentos é a linha de comando do mpv (sem o nome do programa). O mpv.conf
// e os scripts do ouvinte ficam de fora: o mpv do daemon não é dele para
// configurar, e o plugin de MPRIS do sistema mostraria um segundo player, meio
// quebrado, ao lado do do daemon.
func Argumentos(socket string, extras []string) []string {
	return append([]string{
		"--config=no", "--load-scripts=no", "--idle=yes", "--no-video", "--no-terminal",
		"--audio-display=no", "--input-ipc-server=" + socket, "--keep-open=no",
	}, extras...)
}

const (
	tentativas = 100
	passo      = 50 * time.Millisecond
)

// Processo é um mpv rodando e a conversa com ele, uma linha JSON de cada vez.
type Processo struct {
	comando *exec.Cmd
	conexao net.Conn
	escrita sync.Mutex
	saiu    chan struct{}
}

// Inicia sobe o mpv e espera o socket dele responder. Cada linha que o mpv
// escreve (respostas e eventos) vai para aoLinha; quando a conversa acaba (o
// mpv morreu, ou Fecha), aoFim é chamada uma vez. As duas rodam fora do laço.
func Inicia(programa, socket string, extras []string, aoLinha func(linha string), aoFim func()) (*Processo, error) {
	_ = os.Remove(socket)
	comando := exec.Command(programa, Argumentos(socket, extras)...)
	// Se o daemon morrer sem se despedir, o mpv vai junto.
	comando.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGTERM}
	if erro := comando.Start(); erro != nil {
		return nil, erro
	}
	p := &Processo{comando: comando, saiu: make(chan struct{})}
	go func() {
		_ = comando.Wait()
		close(p.saiu)
	}()
	for range tentativas {
		select {
		case <-p.saiu:
			return nil, fmt.Errorf("o mpv saiu logo ao subir (%s)", comando.ProcessState)
		default:
		}
		if conexao, erro := net.Dial("unix", socket); erro == nil {
			p.conexao = conexao
			go p.le(aoLinha, aoFim)
			return p, nil
		}
		time.Sleep(passo)
	}
	_ = comando.Process.Kill()
	return nil, errors.New("o mpv não abriu " + socket)
}

func (p *Processo) le(aoLinha func(string), aoFim func()) {
	linhas := bufio.NewScanner(p.conexao)
	linhas.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for linhas.Scan() {
		aoLinha(linhas.Text())
	}
	if aoFim != nil {
		aoFim()
	}
}

// Envia manda uma linha ao mpv. Se ele foi embora, a linha se perde: o tocador
// percebe pelos eventos que não chegam.
func (p *Processo) Envia(linha string) {
	p.escrita.Lock()
	defer p.escrita.Unlock()
	_ = p.conexao.SetWriteDeadline(time.Now().Add(2 * time.Second))
	_, _ = p.conexao.Write([]byte(linha + "\n"))
}

// Fecha encerra a conversa e o mpv.
func (p *Processo) Fecha() {
	_ = p.conexao.Close()
	_ = p.comando.Process.Signal(syscall.SIGTERM)
	select {
	case <-p.saiu:
	case <-time.After(2 * time.Second):
		_ = p.comando.Process.Kill()
	}
}

var observadas = []string{"time-pos", "duration", "pause"}

// Tocador traduz os pedidos da sessão em comandos do mpv e os eventos do mpv
// em eventos da sessão. Tudo nele roda no laço.
type Tocador struct {
	envia    func(linha string)
	aoEvento func(e app.Evento)

	posicao, duracao float64 // segundos
	// O mpv descarta um pulo pedido antes de o arquivo carregar; um pedido
	// nesse meio tempo espera aqui.
	carregando   bool
	puloPendente *float64
	// O nível do stream (a normalização) e o do ouvinte, multiplicados no
	// volume do mpv.
	volumeDoStream, volumeDoOuvinte float64
	velocidade                      float64
}

// NovoTocador começa a observar posição, duração e pausa.
func NovoTocador(envia func(linha string), aoEvento func(e app.Evento)) *Tocador {
	t := &Tocador{aoEvento: aoEvento, volumeDoStream: 1, volumeDoOuvinte: 1, velocidade: 1}
	t.Religa(envia)
	return t
}

// Religa passa a falar com outro mpv (o anterior morreu): observa de novo e
// repõe o volume e a velocidade.
func (t *Tocador) Religa(envia func(linha string)) {
	t.envia = envia
	t.carregando, t.puloPendente = false, nil
	for i, nome := range observadas {
		t.comando("observe_property", i+1, nome)
	}
	if t.velocidade != 1 {
		t.comando("set_property", "speed", t.velocidade)
	}
}

func (t *Tocador) comando(args ...any) {
	// Sem trocar & por um escape: o endereço do stream vai como é.
	var linha bytes.Buffer
	codificador := json.NewEncoder(&linha)
	codificador.SetEscapeHTML(false)
	_ = codificador.Encode(map[string]any{"command": args})
	t.envia(strings.TrimRight(linha.String(), "\n"))
}

func segundos(s float64) time.Duration { return time.Duration(s * float64(time.Second)) }

func (t *Tocador) Posicao() time.Duration { return segundos(t.posicao) }
func (t *Tocador) Duracao() time.Duration { return segundos(t.duracao) }

func (t *Tocador) Toca(s app.Stream, f dominio.Faixa) {
	t.posicao, t.duracao = 0, 0
	t.carregando, t.puloPendente = true, nil
	t.comando("set_property", "force-media-title", f.Artista+" — "+f.Titulo)
	t.volumeDoStream = s.Volume
	t.aplicaVolume()
	t.comando("loadfile", s.URL, "replace")
	t.comando("set_property", "pause", false)
}

func (t *Tocador) Segura(dominio.Faixa) {
	t.Pausa()
	t.posicao, t.duracao = 0, 0
}

func (t *Tocador) Pausa()    { t.comando("set_property", "pause", true) }
func (t *Tocador) Continua() { t.comando("set_property", "pause", false) }

func (t *Tocador) VaiPara(posicao time.Duration) {
	s := float64(posicao.Milliseconds()) / 1000
	t.posicao = s
	if t.carregando {
		t.puloPendente = &s
		return
	}
	t.comando("seek", s, "absolute")
}

func (t *Tocador) Para() { t.comando("stop") }

func (t *Tocador) Velocidade(v float64) {
	t.velocidade = v
	t.comando("set_property", "speed", v)
}

// PularSilencio: o mpv não tem um equivalente barato; o ajuste é ignorado aqui.
func (t *Tocador) PularSilencio(bool) {}

// VolumeDoOuvinte é o volume do ouvinte, de 0 a 1, por cima do do stream.
func (t *Tocador) VolumeDoOuvinte(nivel float64) {
	t.volumeDoOuvinte = math.Min(math.Max(nivel, 0), 1)
	t.aplicaVolume()
}

func (t *Tocador) aplicaVolume() {
	t.comando("set_property", "volume", int(math.Round(t.volumeDoStream*t.volumeDoOuvinte*100)))
}

// AoLinha recebe cada linha que o mpv escreveu.
func (t *Tocador) AoLinha(linha string) {
	var m struct {
		Event     string          `json:"event"`
		Name      string          `json:"name"`
		Data      json.RawMessage `json:"data"`
		Reason    string          `json:"reason"`
		FileError string          `json:"file_error"`
	}
	if json.Unmarshal([]byte(linha), &m) != nil {
		return
	}
	switch m.Event {
	case "file-loaded":
		t.carregando = false
		if t.puloPendente != nil {
			t.comando("seek", *t.puloPendente, "absolute")
			t.puloPendente = nil
		}
	case "property-change":
		var numero float64
		var ligado bool
		switch m.Name {
		case "time-pos":
			if !t.carregando && t.puloPendente == nil && json.Unmarshal(m.Data, &numero) == nil && string(m.Data) != "null" && len(m.Data) > 0 {
				t.posicao = numero
			}
		case "duration":
			if len(m.Data) > 0 && string(m.Data) != "null" && json.Unmarshal(m.Data, &numero) == nil {
				t.duracao = numero
			}
		case "pause":
			if len(m.Data) > 0 && string(m.Data) != "null" && json.Unmarshal(m.Data, &ligado) == nil {
				t.aoEvento(app.Evento{Tipo: app.MudouPausa, Tocando: !ligado})
			}
		}
	case "end-file":
		switch m.Reason {
		case "eof":
			t.aoEvento(app.Evento{Tipo: app.Terminou})
		case "error":
			t.carregando, t.puloPendente = false, nil
			motivo := m.FileError
			if motivo == "" {
				motivo = "erro do mpv"
			}
			t.aoEvento(app.Evento{Tipo: app.FalhouAoTocar, Motivo: motivo})
		}
	}
}
