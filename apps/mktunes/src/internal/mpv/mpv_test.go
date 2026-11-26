// SPDX-License-Identifier: GPL-3.0-or-later

package mpv

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"mktunes/internal/app"
	"mktunes/internal/prova"
)

var _ app.Tocador = (*Tocador)(nil)

// O próprio binário de teste faz o papel do mpv quando MKTUNES_MPV_DE_MENTIRA
// está ligada: abre o socket pedido, responde a cada comando e "toca" o que
// mandarem carregar até o fim.
func TestMain(m *testing.M) {
	if os.Getenv("MKTUNES_MPV_DE_MENTIRA") == "1" {
		mpvDeMentira()
		return
	}
	os.Exit(m.Run())
}

func mpvDeMentira() {
	var socket string
	for _, a := range os.Args[1:] {
		if depois, ok := strings.CutPrefix(a, "--input-ipc-server="); ok {
			socket = depois
		}
	}
	_ = os.WriteFile(os.Getenv("MKTUNES_MPV_ARGS_VISTOS"), []byte(strings.Join(os.Args[1:], "\n")), 0o600)
	escuta, erro := net.Listen("unix", socket)
	if erro != nil {
		os.Exit(3)
	}
	conexao, erro := escuta.Accept()
	if erro != nil {
		os.Exit(4)
	}
	linhas := bufio.NewScanner(conexao)
	for linhas.Scan() {
		var pedido struct {
			Command []any `json:"command"`
		}
		_ = json.Unmarshal(linhas.Bytes(), &pedido)
		fmt.Fprintln(conexao, `{"error":"success"}`)
		if len(pedido.Command) > 0 && pedido.Command[0] == "loadfile" {
			fmt.Fprintln(conexao, `{"event":"file-loaded"}`)
			fmt.Fprintln(conexao, `{"event":"property-change","id":2,"name":"duration","data":1.0}`)
			fmt.Fprintln(conexao, `{"event":"end-file","reason":"eof"}`)
		}
	}
}

func igual(t *testing.T, esperado, obtido any) {
	t.Helper()
	if !reflect.DeepEqual(esperado, obtido) {
		t.Fatalf("esperado %#v\n  obtido %#v", esperado, obtido)
	}
}

func TestArgumentos(t *testing.T) {
	args := Argumentos("/run/mpv.sock", []string{"--ao=null"})
	for _, preciso := range []string{"--input-ipc-server=/run/mpv.sock", "--no-video", "--idle=yes", "--no-terminal",
		// Fora o mpv.conf e os scripts do ouvinte: o plugin de MPRIS dele mostraria um segundo player.
		"--config=no", "--load-scripts=no"} {
		if !slices.Contains(args, preciso) {
			t.Fatalf("falta %s em %v", preciso, args)
		}
	}
	igual(t, "--ao=null", args[len(args)-1])
}

// fio guarda o que o tocador mandou ao mpv, cada comando como texto.
type fio struct {
	enviadas [][]string
	eventos  []app.Evento
}

func (f *fio) envia(linha string) {
	var pedido struct {
		Command []any `json:"command"`
	}
	_ = json.Unmarshal([]byte(linha), &pedido)
	var comando []string
	for _, a := range pedido.Command {
		comando = append(comando, fmt.Sprint(a))
	}
	f.enviadas = append(f.enviadas, comando)
}

func (f *fio) tem(comando ...string) bool {
	return slices.ContainsFunc(f.enviadas, func(c []string) bool { return slices.Equal(c, comando) })
}

func (f *fio) ultima() []string { return f.enviadas[len(f.enviadas)-1] }

func novoTocador() (*Tocador, *fio) {
	f := &fio{}
	return NovoTocador(f.envia, func(e app.Evento) { f.eventos = append(f.eventos, e) }), f
}

func TestTocador(t *testing.T) {
	t.Run("observa posição, duração e pausa assim que começa", func(t *testing.T) {
		_, f := novoTocador()
		igual(t, [][]string{{"observe_property", "1", "time-pos"}, {"observe_property", "2", "duration"}, {"observe_property", "3", "pause"}}, f.enviadas)
	})
	t.Run("toca um stream no volume dele, com a faixa como título", func(t *testing.T) {
		p, f := novoTocador()
		p.Toca(app.Stream{URL: "https://a/song.webm", Volume: 0.5}, prova.Faixa("1", prova.Titulo("Umbrella"), prova.Artista("Rihanna")))
		igual(t, true, f.tem("set_property", "force-media-title", "Rihanna — Umbrella"))
		igual(t, true, f.tem("set_property", "volume", "50"))
		igual(t, true, f.tem("loadfile", "https://a/song.webm", "replace"))
		igual(t, []string{"set_property", "pause", "false"}, f.ultima())
	})
	t.Run("o volume do ouvinte escala o do stream, agora e nas próximas, entre o silêncio e o cheio", func(t *testing.T) {
		p, f := novoTocador()
		p.Toca(app.Stream{URL: "https://a/1.webm", Volume: 0.8}, prova.Faixa("1"))
		p.VolumeDoOuvinte(0.5)
		igual(t, []string{"set_property", "volume", "40"}, f.ultima())
		p.Toca(app.Stream{URL: "https://a/2.webm", Volume: 1}, prova.Faixa("2"))
		igual(t, true, f.tem("set_property", "volume", "50"))
		p.VolumeDoOuvinte(3)
		igual(t, []string{"set_property", "volume", "100"}, f.ultima())
		p.VolumeDoOuvinte(-1)
		igual(t, []string{"set_property", "volume", "0"}, f.ultima())
	})
	t.Run("segue o que o mpv informa de posição e duração", func(t *testing.T) {
		p, _ := novoTocador()
		p.AoLinha(`{"event":"property-change","id":1,"name":"time-pos","data":12.5}`)
		p.AoLinha(`{"event":"property-change","id":2,"name":"duration","data":200.0}`)
		igual(t, 12500*time.Millisecond, p.Posicao())
		igual(t, 200*time.Second, p.Duracao())
	})
	t.Run("avisa do fim da música, mas não da música trocada ou parada", func(t *testing.T) {
		p, f := novoTocador()
		p.AoLinha(`{"event":"end-file","reason":"stop"}`)
		p.AoLinha(`{"event":"end-file","reason":"eof"}`)
		p.AoLinha(`{"event":"end-file","reason":"error","file_error":"loading failed"}`)
		igual(t, []app.Evento{{Tipo: app.Terminou}, {Tipo: app.FalhouAoTocar, Motivo: "loading failed"}}, f.eventos)
	})
	t.Run("a pausa apertada em qualquer lugar chega à sessão", func(t *testing.T) {
		p, f := novoTocador()
		p.AoLinha(`{"event":"property-change","id":3,"name":"pause","data":true}`)
		igual(t, []app.Evento{{Tipo: app.MudouPausa, Tocando: false}}, f.eventos)
	})
	t.Run("pula, pausa, continua, para e muda a velocidade", func(t *testing.T) {
		p, f := novoTocador()
		f.enviadas = nil
		p.VaiPara(90 * time.Second)
		p.Pausa()
		p.Continua()
		p.Velocidade(1.25)
		p.Para()
		igual(t, [][]string{{"seek", "90", "absolute"}, {"set_property", "pause", "true"}, {"set_property", "pause", "false"}, {"set_property", "speed", "1.25"}, {"stop"}}, f.enviadas)
	})
	t.Run("segurar a próxima cala a antiga e começa do zero", func(t *testing.T) {
		p, f := novoTocador()
		p.AoLinha(`{"event":"property-change","id":1,"name":"time-pos","data":80.0}`)
		p.Segura(prova.Faixa("2"))
		igual(t, time.Duration(0), p.Posicao())
		igual(t, true, f.tem("set_property", "pause", "true"))
	})
	t.Run("um pulo pedido enquanto a música carrega espera ela carregar", func(t *testing.T) {
		p, f := novoTocador()
		p.Toca(app.Stream{URL: "https://a/song.webm", Volume: 1}, prova.Faixa("1"))
		f.enviadas = nil
		p.VaiPara(82 * time.Second)
		igual(t, 0, len(f.enviadas))
		igual(t, 82*time.Second, p.Posicao())
		p.AoLinha(`{"event":"property-change","id":1,"name":"time-pos","data":0.0}`)
		igual(t, 82*time.Second, p.Posicao())
		p.AoLinha(`{"event":"file-loaded"}`)
		igual(t, [][]string{{"seek", "82", "absolute"}}, f.enviadas)
	})
	t.Run("ignora o que não é evento, como as respostas aos comandos", func(t *testing.T) {
		p, f := novoTocador()
		p.AoLinha(`{"error":"success","request_id":0}`)
		p.AoLinha("not json")
		p.AoLinha(`{"event":"property-change","id":1,"name":"time-pos","data":null}`)
		igual(t, time.Duration(0), p.Posicao())
		igual(t, 0, len(f.eventos))
	})
	t.Run("religar a outro mpv observa de novo e repõe a velocidade", func(t *testing.T) {
		p, _ := novoTocador()
		p.Velocidade(1.5)
		outro := &fio{}
		p.Religa(outro.envia)
		igual(t, 4, len(outro.enviadas))
		igual(t, []string{"set_property", "speed", "1.5"}, outro.ultima())
	})
}

func TestProcessoFalaComUmMpv(t *testing.T) {
	pasta, _ := os.MkdirTemp("", "mkt")
	t.Cleanup(func() { os.RemoveAll(pasta) })
	vistos := filepath.Join(pasta, "args")
	t.Setenv("MKTUNES_MPV_DE_MENTIRA", "1")
	t.Setenv("MKTUNES_MPV_ARGS_VISTOS", vistos)
	socket := filepath.Join(pasta, "mpv.sock")
	_ = os.WriteFile(socket, nil, 0o600) // um socket velho, de um mpv que morreu
	linhas := make(chan string, 64)
	fim := make(chan struct{})
	processo, erro := Inicia(os.Args[0], socket, []string{"--ao=null"}, func(l string) { linhas <- l }, func() { close(fim) })
	if erro != nil {
		t.Fatal(erro)
	}
	eventos := make(chan app.Evento, 8)
	tocador := NovoTocador(processo.Envia, func(e app.Evento) { eventos <- e })
	tocador.Toca(app.Stream{URL: "/tmp/silencio.wav", Volume: 1}, prova.Faixa("1"))
	prazo := time.After(10 * time.Second)
	for terminou := false; !terminou; {
		select {
		case linha := <-linhas:
			tocador.AoLinha(linha)
		case e := <-eventos:
			igual(t, app.Terminou, e.Tipo)
			terminou = true
		case <-prazo:
			t.Fatal("o mpv de mentira não tocou até o fim")
		}
	}
	igual(t, time.Second, tocador.Duracao())
	argumentos, _ := os.ReadFile(vistos)
	igual(t, strings.Join(Argumentos(socket, []string{"--ao=null"}), "\n"), string(argumentos))
	processo.Fecha()
	select {
	case <-fim:
	case <-time.After(5 * time.Second):
		t.Fatal("o fim da conversa não foi avisado")
	}
}

func TestMpvQueNaoSobe(t *testing.T) {
	pasta := t.TempDir()
	falso := filepath.Join(pasta, "mpv")
	_ = os.WriteFile(falso, []byte("#!/bin/sh\nexit 7\n"), 0o755)
	if _, erro := Inicia(falso, filepath.Join(pasta, "mpv.sock"), nil, func(string) {}, nil); erro == nil || !strings.Contains(erro.Error(), "saiu logo") {
		t.Fatalf("erro: %v", erro)
	}
	if _, erro := Inicia(filepath.Join(pasta, "não-existe"), filepath.Join(pasta, "mpv.sock"), nil, func(string) {}, nil); erro == nil {
		t.Fatal("um mpv que não existe é um erro")
	}
}
