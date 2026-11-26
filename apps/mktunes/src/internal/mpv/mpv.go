// SPDX-License-Identifier: GPL-3.0-or-later

// Pacote mpv: a saída de áudio do daemon é um mpv sem janela, falado por JSON
// no socket de --input-ipc-server. O mpv também é quem baixa o stream, do
// endereço que o daemon achou: ele nunca procura um por conta própria
// (--ytdl=no).
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
	"runtime"
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
		"--ytdl=no",
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
	p := &Processo{comando: comando, saiu: make(chan struct{})}
	subiu := make(chan error, 1)
	go func() {
		// O Pdeathsig vale para a thread que criou o processo, não para o daemon
		// inteiro: se o Go encerrasse essa thread, o mpv seria derrubado por
		// engano. Esta goroutine fica presa a uma thread só dela, que vive
		// enquanto o mpv vive.
		runtime.LockOSThread()
		erro := comando.Start()
		subiu <- erro
		if erro != nil {
			runtime.UnlockOSThread()
			return
		}
		_ = comando.Wait()
		close(p.saiu)
	}()
	if erro := <-subiu; erro != nil {
		return nil, erro
	}
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

// Saida diz como o mpv terminou ("exit status 2", "signal: killed"), para o
// registro; espera um instante por ele, e diz se ainda está de pé.
func (p *Processo) Saida() string {
	select {
	case <-p.saiu:
		return p.comando.ProcessState.String()
	case <-time.After(time.Second):
		return "ainda rodando"
	}
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
	// pausaPedida é o último pedido de pausa feito ao mpv: um eco que diz outra
	// coisa é de um pedido mais antigo, já desfeito.
	pausaPedida *bool
	// O mpv numera cada arquivo que mandam abrir (playlist_entry_id): pedido é
	// o request_id do último loadfile; entrada, o número que o mpv deu a ele
	// (0: ainda não respondeu; -1: este mpv não numera); comecou, o do último
	// arquivo que ele começou a abrir. Assim o que chega atrasado da faixa
	// antiga não passa por notícia da nova.
	pedido           int
	entrada, comecou int64
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
	t.carregando, t.puloPendente, t.pausaPedida = false, nil, nil
	t.pedido, t.entrada, t.comecou = 0, 0, 0
	for i, nome := range observadas {
		t.comando("observe_property", i+1, nome)
	}
	if t.velocidade != 1 {
		t.comando("set_property", "speed", t.velocidade)
	}
}

func (t *Tocador) comando(args ...any) { t.manda(map[string]any{"command": args}) }

func (t *Tocador) manda(pedido map[string]any) {
	// Sem trocar & por um escape: o endereço do stream vai como é.
	var linha bytes.Buffer
	codificador := json.NewEncoder(&linha)
	codificador.SetEscapeHTML(false)
	_ = codificador.Encode(pedido)
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
	t.pedido++
	t.entrada = 0
	t.manda(map[string]any{"command": []any{"loadfile", s.URL, "replace"}, "request_id": t.pedido})
	t.pausa(false)
}

func (t *Tocador) Segura(dominio.Faixa) {
	t.Pausa()
	t.posicao, t.duracao = 0, 0
}

func (t *Tocador) Pausa()    { t.pausa(true) }
func (t *Tocador) Continua() { t.pausa(false) }

func (t *Tocador) pausa(ligada bool) {
	t.pausaPedida = &ligada
	t.comando("set_property", "pause", ligada)
}

// daAtual diz se um evento com este playlist_entry_id é da faixa pedida por último.
func (t *Tocador) daAtual(id int64) bool {
	if t.pedido == 0 || t.entrada < 0 || id == 0 {
		return true // nada pedido ainda, ou um mpv que não numera
	}
	return id == t.entrada
}

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
		Entrada   int64           `json:"playlist_entry_id"`
		Pedido    int             `json:"request_id"`
		Erro      string          `json:"error"`
	}
	if json.Unmarshal([]byte(linha), &m) != nil {
		return
	}
	switch m.Event {
	case "":
		// A resposta ao último loadfile: o número que o mpv deu à faixa, ou a recusa.
		if t.pedido == 0 || m.Pedido != t.pedido || t.entrada != 0 {
			return
		}
		if m.Erro != "" && m.Erro != "success" {
			t.entrada = -1
			t.falhou(m.Erro)
			return
		}
		var dados struct {
			Entrada int64 `json:"playlist_entry_id"`
		}
		if _ = json.Unmarshal(m.Data, &dados); dados.Entrada > 0 {
			t.entrada = dados.Entrada
		} else {
			t.entrada = -1
		}
	case "start-file":
		t.comecou = m.Entrada
	case "file-loaded":
		if !t.daAtual(t.comecou) {
			return
		}
		eraNova := t.carregando
		t.carregando = false
		if t.puloPendente != nil {
			t.comando("seek", *t.puloPendente, "absolute")
			t.puloPendente = nil
		}
		if eraNova {
			t.aoEvento(app.Evento{Tipo: app.Carregou})
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
				// O eco de um pedido que o daemon já desfez (o "segura" de antes
				// de tocar) não é o estado de agora.
				if t.pausaPedida != nil && *t.pausaPedida != ligado {
					return
				}
				t.aoEvento(app.Evento{Tipo: app.MudouPausa, Tocando: !ligado})
			}
		}
	case "end-file":
		if !t.daAtual(m.Entrada) {
			return
		}
		switch m.Reason {
		case "eof":
			t.aoEvento(app.Evento{Tipo: app.Terminou})
		case "error":
			t.falhou(m.FileError)
		}
	}
}

func (t *Tocador) falhou(motivo string) {
	t.carregando, t.puloPendente = false, nil
	if motivo == "" {
		motivo = "erro do mpv"
	}
	t.aoEvento(app.Evento{Tipo: app.FalhouAoTocar, Motivo: motivo})
}
