// SPDX-License-Identifier: GPL-3.0-or-later

package dbus

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

func igual(t *testing.T, esperado, obtido any) {
	t.Helper()
	if !reflect.DeepEqual(esperado, obtido) {
		t.Fatalf("esperado %#v\n  obtido %#v", esperado, obtido)
	}
}

func hexa(t *testing.T, assinatura string, valores ...any) string {
	t.Helper()
	dados, erro := Codifica(assinatura, valores...)
	if erro != nil {
		t.Fatal(erro)
	}
	return hex.EncodeToString(dados)
}

// Os bytes aqui foram calculados à mão pela especificação: tamanhos em 32 bits
// little-endian, textos com NUL no fim, e cada tipo alinhado ao seu tamanho.
func TestBytesPelaEspecificacao(t *testing.T) {
	igual(t, "03000000"+"666f6f00", hexa(t, "s", "foo"))
	igual(t, "0173"+"00", hexa(t, "g", Assinatura("s")))
	// Um byte, três de enchimento, e o uint32 alinhado em 4.
	igual(t, "01"+"000000"+"02000000", hexa(t, "yu", byte(1), uint32(2)))
	igual(t, "01000000"+"0000", hexa(t, "bn", true, int16(0)))
	igual(t, "08000000"+"01000000"+"02000000", hexa(t, "ai", []any{int32(1), int32(2)}))
	// O tamanho de uma lista não conta o enchimento antes do primeiro elemento.
	igual(t, "08000000"+"00000000"+"0500000000000000", hexa(t, "ax", []any{int64(5)}))
	igual(t, "00000000", hexa(t, "as", []string{}))
	// Uma variante: a assinatura (1 byte de tamanho, o tipo, NUL) e o valor alinhado.
	igual(t, "017500"+"00"+"07000000", hexa(t, "v", Variante{"u", uint32(7)}))
	// Um dicionário a{sv} com {"a": true}: cada entrada alinhada em 8.
	igual(t, "10000000"+"00000000"+"01000000"+"6100"+"016200"+"000000"+"01000000", hexa(t, "a{sv}", map[string]Variante{"a": {"b", true}}))
	// Um struct (ys) depois de um byte: alinhado em 8.
	igual(t, "ff"+"00000000000000"+"07"+"000000"+"01000000"+"7a00", hexa(t, "y(ys)", byte(255), []any{byte(7), "z"}))
	igual(t, "000000000000f03f", hexa(t, "d", 1.0))
	igual(t, "01000000"+"2f00", hexa(t, "o", Caminho("/")))
}

func TestTodosOsTiposVaoEVoltam(t *testing.T) {
	assinatura := "ybnqiuxtdsogvasa{sv}a{us}(sx)aay"
	valores := []any{
		byte(200), true, int16(-2), uint16(65000), int32(-70000), uint32(4000000000), int64(-1 << 40), uint64(1 << 63), 0.4,
		"Até Quando — ♫", Caminho("/org/mpris/MediaPlayer2"), Assinatura("a{sv}"),
		Variante{"a{sv}", map[string]Variante{
			"mpris:trackid": {"o", Caminho("/dev/mk/mktunes/track/abc")},
			"xesam:artist":  {"as", []string{"Natanzinho Lima"}},
			"mpris:length":  {"x", int64(194000000)},
		}},
		[]string{"um", "", "três"},
		map[string]Variante{"Volume": {"d", 0.8}, "CanSeek": {"b", false}, "Aninhado": {"v", Variante{"s", "dentro"}}},
		[]Par{{uint32(1), "um"}, {uint32(2), "dois"}},
		[]any{"struct", int64(-1)},
		[]any{[]byte{1, 2, 3}, []byte{}},
	}
	dados, erro := Codifica(assinatura, valores...)
	if erro != nil {
		t.Fatal(erro)
	}
	lidos, erro := Decodifica(assinatura, dados)
	if erro != nil {
		t.Fatal(erro)
	}
	igual(t, valores, lidos)
	// Codificar de novo dá os mesmos bytes (os dicionários saem em ordem de chave).
	outra, _ := Codifica(assinatura, lidos...)
	igual(t, true, bytes.Equal(dados, outra))
}

func TestRecusaOQueNaoCabe(t *testing.T) {
	for _, caso := range []struct {
		assinatura string
		valores    []any
	}{
		{"s", []any{7}}, {"u", []any{int32(1)}}, {"o", []any{Caminho("sem/barra")}}, {"o", []any{Caminho("/tem espaço")}},
		{"s", []any{"nulo\x00no meio"}}, {"s", []any{"\xff\xfe"}}, {"ss", []any{"só um"}}, {"s", []any{"a", "b"}},
		{"v", []any{Variante{"ss", "dois tipos"}}}, {"a{sv}", []any{[]string{"x"}}}, {"(s", []any{"x"}}, {"z", []any{1}},
		{"ay", []any{"texto"}}, {"()", []any{[]any{}}},
	} {
		if _, erro := Codifica(caso.assinatura, caso.valores...); erro == nil {
			t.Fatalf("devia recusar %q com %#v", caso.assinatura, caso.valores)
		}
	}
	// Bytes cortados, tamanhos mentirosos e texto sem NUL não derrubam o leitor.
	inteiro, _ := Codifica("sa{sv}", "abc", map[string]Variante{"k": {"as", []string{"x", "y"}}})
	for corte := range len(inteiro) {
		if _, erro := Decodifica("sa{sv}", inteiro[:corte]); erro == nil {
			t.Fatalf("devia recusar a mensagem cortada em %d", corte)
		}
	}
	for _, ruim := range []string{"ffffffff", "0300000066", "03000000666f6f41", "05000000" + "00000000"} {
		dados, _ := hex.DecodeString(ruim)
		if _, erro := Decodifica("s", dados); erro == nil {
			t.Fatalf("devia recusar %s", ruim)
		}
	}
	// Variantes dentro de variantes sem fim são cortadas.
	fundo := bytes.Repeat([]byte{1, 0x76, 0}, 200)
	if _, erro := Decodifica("v", fundo); erro == nil {
		t.Fatal("devia recusar um valor aninhado demais")
	}
	igual(t, []bool{true, true, false, false, false, false}, []bool{CaminhoValido("/"), CaminhoValido("/a/b_1"), CaminhoValido(""), CaminhoValido("/a/"), CaminhoValido("//"), CaminhoValido("/a-b")})
}

func TestMensagemOla(t *testing.T) {
	ola := &Mensagem{Tipo: Chamada, Serial: 1, Destino: "org.freedesktop.DBus", Caminho: "/org/freedesktop/DBus", Interface: "org.freedesktop.DBus", Membro: "Hello"}
	dados, erro := ola.Codifica()
	if erro != nil {
		t.Fatal(erro)
	}
	// Cabeçalho fixo: little-endian, chamada, sem marcas, protocolo 1, corpo
	// vazio, série 1, e 109 bytes de campos (caminho, interface, membro e
	// destino, cada um num struct alinhado em 8); o total é múltiplo de 8.
	igual(t, "6c010001"+"00000000"+"01000000"+"6d000000", hex.EncodeToString(dados[:16]))
	igual(t, "01016f00"+"15000000"+hex.EncodeToString([]byte("/org/freedesktop/DBus"))+"00"+"0000", hex.EncodeToString(dados[16:48]))
	igual(t, "03017300"+"05000000"+hex.EncodeToString([]byte("Hello"))+"00"+"0000", hex.EncodeToString(dados[80:96]))
	igual(t, 128, len(dados))
	lida, erro := LeMensagem(bytes.NewReader(dados))
	if erro != nil {
		t.Fatal(erro)
	}
	ola.Corpo = []any{}
	igual(t, ola, lida)

	// Uma resposta com corpo, e uma mensagem big-endian (de outra arquitetura).
	resposta := &Mensagem{Tipo: Resposta, Serial: 7, RespostaA: 3, Destino: ":1.9", Remetente: "org.freedesktop.DBus", Assinatura: "su", Corpo: []any{":1.9", uint32(1)}}
	dados, _ = resposta.Codifica()
	lida, erro = LeMensagem(bufio.NewReader(bytes.NewReader(append(dados, 0xff))))
	if erro != nil {
		t.Fatal(erro)
	}
	igual(t, resposta, lida)
	grande, _ := hex.DecodeString("42040101" + "00000004" + "00000005" + "0000000a" + "03017300" + "00000001" + "5800" + "000000000000" + "0000002a")
	lida, erro = LeMensagem(bytes.NewReader(grande))
	if erro != nil {
		t.Fatal(erro)
	}
	igual(t, &Mensagem{Tipo: Sinal, Marcas: 1, Serial: 5, Membro: "X", Corpo: []any{}}, lida)

	for _, ruim := range []string{"", "6c0100", "7a010001000000000100000000000000", "6c010002000000000100000000000000", "6c010001ffffffff0100000000000000", "6c010001000000000100000008000000" + "ffffffffffffffff"} {
		bruto, _ := hex.DecodeString(ruim)
		if _, erro := LeMensagem(bytes.NewReader(bruto)); erro == nil {
			t.Fatalf("devia recusar %q", ruim)
		}
	}
}

func TestEnderecos(t *testing.T) {
	igual(t, []string{"/run/user/1000/bus"}, EnderecosDe("unix:path=/run/user/1000/bus", ""))
	igual(t, []string{"@/tmp/dbus-AbC", "/com espaço/bus", "/run/user/7/bus"}, EnderecosDe("tcp:host=x,port=1;unix:abstract=/tmp/dbus-AbC,guid=1234;unix:path=/com%20espa%c3%a7o/bus;unix:runtime=yes", "/run/user/7"))
	igual(t, 0, len(EnderecosDe("", "/run")))
	igual(t, 0, len(EnderecosDe("lixo;unix:;unix:path=;unix:path=%zz", "")))
}

// pastaCurta dá uma pasta temporária de nome curto: o caminho de um socket Unix
// tem limite de tamanho.
func pastaCurta(t *testing.T) string {
	t.Helper()
	pasta, erro := os.MkdirTemp("", "mkt")
	if erro != nil {
		t.Fatal(erro)
	}
	t.Cleanup(func() { os.RemoveAll(pasta) })
	return pasta
}

// barramento é um barramento de mentira num socket temporário: faz o aperto de
// mão, responde ao Hello e ao RequestName, e deixa o teste trocar mensagens.
type barramento struct {
	t       *testing.T
	socket  string
	escuta  net.Listener
	recusa  bool
	nomes   chan string
	conexao chan net.Conn
}

func novoBarramento(t *testing.T) *barramento {
	b := &barramento{t: t, socket: filepath.Join(pastaCurta(t), "bus"), nomes: make(chan string, 4), conexao: make(chan net.Conn, 1)}
	escuta, erro := net.Listen("unix", b.socket)
	if erro != nil {
		t.Fatal(erro)
	}
	b.escuta = escuta
	t.Cleanup(func() { escuta.Close() })
	go b.atende()
	return b
}

func (b *barramento) atende() {
	fio, erro := b.escuta.Accept()
	if erro != nil {
		return
	}
	leitura := bufio.NewReader(fio)
	nulo, _ := leitura.ReadByte()
	linha, _ := leitura.ReadString('\n')
	uid := hex.EncodeToString([]byte(strconv.Itoa(os.Getuid())))
	if nulo != 0 || linha != "AUTH EXTERNAL "+uid+"\r\n" || b.recusa {
		fio.Write([]byte("REJECTED EXTERNAL\r\n"))
		fio.Close()
		return
	}
	fio.Write([]byte("OK 1234deadbeef\r\n"))
	if comeca, _ := leitura.ReadString('\n'); comeca != "BEGIN\r\n" {
		fio.Close()
		return
	}
	responde := func(a *Mensagem, assinatura string, valores ...any) {
		dados, _ := (&Mensagem{Tipo: Resposta, Serial: 100 + a.Serial, RespostaA: a.Serial, Remetente: "org.freedesktop.DBus", Destino: ":1.42", Assinatura: assinatura, Corpo: valores}).Codifica()
		fio.Write(dados)
	}
	ola, erro := LeMensagem(leitura)
	if erro != nil || ola.Membro != "Hello" || ola.Destino != "org.freedesktop.DBus" || ola.Caminho != "/org/freedesktop/DBus" {
		fio.Close()
		return
	}
	responde(ola, "s", ":1.42")
	// Um sinal que não foi pedido (NameAcquired) é só ignorado pelo cliente.
	sinal, _ := (&Mensagem{Tipo: Sinal, Serial: 2, Caminho: "/org/freedesktop/DBus", Interface: "org.freedesktop.DBus", Membro: "NameAcquired", Assinatura: "s", Corpo: []any{":1.42"}}).Codifica()
	fio.Write(sinal)
	b.conexao <- fio
	for {
		m, erro := LeMensagem(leitura)
		if erro != nil {
			return
		}
		switch {
		case m.Tipo == Chamada && m.Membro == "RequestName":
			nome, _ := m.Corpo[0].(string)
			b.nomes <- nome
			codigo := uint32(1)
			if nome == "nome.ja.tomado" {
				codigo = 3
			}
			responde(m, "u", codigo)
		case m.Tipo == Chamada && m.Membro == "Falha":
			dados, _ := (&Mensagem{Tipo: Erro, Serial: 9, RespostaA: m.Serial, NomeDoErro: "org.freedesktop.DBus.Error.Failed", Assinatura: "s", Corpo: []any{"não deu"}}).Codifica()
			fio.Write(dados)
		default:
			b.t.Log("o barramento recebeu", m.Tipo, m.Membro)
			recebidas <- m
		}
	}
}

var recebidas = make(chan *Mensagem, 16)

func espera[T any](t *testing.T, canal <-chan T) T {
	t.Helper()
	select {
	case valor := <-canal:
		return valor
	case <-time.After(5 * time.Second):
		t.Fatal("nada chegou")
		panic("nunca")
	}
}

func TestConversaComUmBarramentoDeMentira(t *testing.T) {
	b := novoBarramento(t)
	chamadas := make(chan *Mensagem, 4)
	c, erro := LigaEm(b.socket, func(m *Mensagem) { chamadas <- m })
	if erro != nil {
		t.Fatal(erro)
	}
	defer c.Fecha()
	igual(t, ":1.42", c.Nome)
	dono, erro := c.PedeNome("org.mpris.MediaPlayer2.mktunes")
	igual(t, []any{true, nil}, []any{dono, erro})
	igual(t, "org.mpris.MediaPlayer2.mktunes", espera(t, b.nomes))
	dono, _ = c.PedeNome("nome.ja.tomado")
	igual(t, false, dono)
	_, erro = c.Chama("x.y", "/", "x.y", "Falha", "")
	if erro == nil || !strings.Contains(erro.Error(), "org.freedesktop.DBus.Error.Failed: não deu") {
		t.Fatalf("erro: %v", erro)
	}

	// O barramento entrega uma chamada de outro cliente; a resposta volta para ele.
	fio := espera(t, b.conexao)
	pedido, _ := (&Mensagem{Tipo: Chamada, Serial: 77, Remetente: ":1.7", Destino: ":1.42", Caminho: "/org/mpris/MediaPlayer2", Interface: "org.freedesktop.DBus.Properties", Membro: "Get", Assinatura: "ss", Corpo: []any{"org.mpris.MediaPlayer2", "Identity"}}).Codifica()
	fio.Write(pedido)
	chegou := espera(t, chamadas)
	igual(t, []any{"Get", ":1.7", []any{"org.mpris.MediaPlayer2", "Identity"}}, []any{chegou.Membro, chegou.Remetente, chegou.Corpo})
	c.Responde(chegou, "v", Variante{"s", "MkTunes"})
	resposta := espera(t, recebidas)
	igual(t, []any{Resposta, uint32(77), ":1.7", "v", []any{Variante{"s", "MkTunes"}}}, []any{resposta.Tipo, resposta.RespostaA, resposta.Destino, resposta.Assinatura, resposta.Corpo})

	c.RespondeErro(chegou, "org.freedesktop.DBus.Error.UnknownMethod", "não sei")
	erroVolta := espera(t, recebidas)
	igual(t, []any{Erro, "org.freedesktop.DBus.Error.UnknownMethod", []any{"não sei"}}, []any{erroVolta.Tipo, erroVolta.NomeDoErro, erroVolta.Corpo})
	// Quem não quer resposta não recebe nenhuma.
	chegou.Marcas = SemResposta
	c.Responde(chegou, "")
	c.RespondeErro(chegou, "x.y", "z")

	if erro := c.EmiteSinal("/org/mpris/MediaPlayer2", "org.mpris.MediaPlayer2.Player", "Seeked", "x", int64(30000000)); erro != nil {
		t.Fatal(erro)
	}
	sinal := espera(t, recebidas)
	igual(t, []any{Sinal, "Seeked", Caminho("/org/mpris/MediaPlayer2"), []any{int64(30000000)}}, []any{sinal.Tipo, sinal.Membro, sinal.Caminho, sinal.Corpo})
	igual(t, true, sinal.Serial > resposta.Serial)

	// O barramento caindo acaba a conversa sem travar ninguém.
	fio.Close()
	espera(t, c.Terminou())
	if _, erro := c.Chama("x.y", "/", "x.y", "Depois", ""); erro == nil {
		t.Fatal("chamar numa conexão fechada é um erro")
	}
}

func TestSemBarramentoOuRecusado(t *testing.T) {
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "")
	t.Setenv("XDG_RUNTIME_DIR", "")
	if _, erro := Liga(nil); erro != ErrSemBarramento {
		t.Fatalf("erro: %v", erro)
	}
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path="+filepath.Join(t.TempDir(), "não-existe"))
	if _, erro := Liga(nil); erro == nil || erro == ErrSemBarramento {
		t.Fatalf("erro: %v", erro)
	}
	b := novoBarramento(t)
	b.recusa = true
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path="+b.socket+",guid=abc")
	if _, erro := Liga(nil); erro == nil || !strings.Contains(erro.Error(), "recusou") {
		t.Fatalf("erro: %v", erro)
	}
	outro := novoBarramento(t)
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path=/não/existe;unix:path="+outro.socket)
	c, erro := Liga(nil)
	if erro != nil {
		t.Fatal(erro)
	}
	c.Fecha()
}
