// SPDX-License-Identifier: GPL-3.0-or-later

package mpris

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"mktunes/internal/dbus"
	"mktunes/internal/laco"
)

func igual(t *testing.T, esperado, obtido any) {
	t.Helper()
	if !reflect.DeepEqual(esperado, obtido) {
		t.Fatalf("esperado %#v\n  obtido %#v", esperado, obtido)
	}
}

// igualJSON compara dois comandos pelo valor (90 e 90.0 são o mesmo número).
func igualJSON(t *testing.T, esperado, obtido string) {
	t.Helper()
	var a, b any
	if json.Unmarshal([]byte(esperado), &a) != nil || json.Unmarshal([]byte(obtido), &b) != nil || !reflect.DeepEqual(a, b) {
		t.Fatalf("esperado %s\n  obtido %s", esperado, obtido)
	}
}

const musica = `{"id":"qUYep-Q_1","provider":"YOUTUBE","title":"Sonho de Amor","artist":"Natanzinho Lima","durationMs":194000,"length":"3:14","artwork":"https://img/1.jpg"}`

func agora(situacao, repete string, aoVivo bool) string {
	faixa := musica
	if situacao == "idle" {
		faixa = "{}"
	}
	vivo := "false"
	if aoVivo {
		vivo = "true"
	}
	return `{"type":"now","status":"` + situacao + `","track":` + faixa + `,"title":"Sonho de Amor","artist":"Natanzinho Lima","artwork":"https://img/1.jpg","live":` + vivo + `,"upNext":[],"repeat":"` + repete + `","speed":1.25}`
}

const (
	progresso = `{"type":"progress","position":61.5,"duration":194.0}`
	volume    = `{"type":"volume","level":40}`
)

func TestModelo(t *testing.T) {
	t.Run("sem nada para tocar está parado, com a faixa vazia e nada a fazer", func(t *testing.T) {
		p := Jogador(agora("idle", "off", false), progresso, volume)
		igual(t, "Stopped", p["PlaybackStatus"])
		igual(t, map[string]any{"mpris:trackid": SemFaixa}, p["Metadata"])
		igual(t, []any{false, false, false, true}, []any{p["CanPlay"], p["CanGoNext"], p["CanSeek"], p["CanControl"]})
		// Antes de o daemon publicar qualquer coisa também.
		igual(t, "Stopped", Jogador("", "", "")["PlaybackStatus"])
		igual(t, 1.0, Jogador("", "", "")["Volume"])
	})
	t.Run("uma música tocando diz à ilha o título, o artista, a capa, a duração e onde está", func(t *testing.T) {
		p := Jogador(agora("playing", "off", false), progresso, volume)
		igual(t, "Playing", p["PlaybackStatus"])
		igual(t, map[string]any{
			"xesam:title": "Sonho de Amor", "xesam:artist": []string{"Natanzinho Lima"}, "mpris:artUrl": "https://img/1.jpg",
			"mpris:length": int64(194_000_000),
			// Os caminhos de objeto só aceitam letras, números e _: o resto vira _XX.
			"mpris:trackid": dbus.Caminho("/dev/mk/mktunes/track/qUYep_2DQ_5F1"),
		}, p["Metadata"])
		igual(t, []any{int64(61_500_000), 0.4, 1.25, true, true}, []any{p["Position"], p["Volume"], p["Rate"], p["CanSeek"], p["CanGoNext"]})
		igual(t, true, dbus.CaminhoValido(string(p["Metadata"].(map[string]any)["mpris:trackid"].(dbus.Caminho))))
	})
	t.Run("pausado, carregando e falhou, e o modo de repetir", func(t *testing.T) {
		de := func(situacao string) any {
			return Jogador(agora(situacao, "off", false), progresso, volume)["PlaybackStatus"]
		}
		igual(t, []any{"Paused", "Playing", "Stopped"}, []any{de("paused"), de("loading"), de("failed")})
		laco := func(repete string) any {
			return Jogador(agora("playing", repete, false), progresso, volume)["LoopStatus"]
		}
		igual(t, []any{"None", "Track", "Playlist"}, []any{laco("off"), laco("one"), laco("all")})
	})
	t.Run("uma rádio não tem duração nem dá para pular", func(t *testing.T) {
		p := Jogador(agora("playing", "off", true), progresso, volume)
		_, tem := p["Metadata"].(map[string]any)["mpris:length"]
		igual(t, []any{false, false}, []any{tem, p["CanSeek"]})
	})
	t.Run("os botões da ilha viram os comandos da janela", func(t *testing.T) {
		tocando, pausado := agora("playing", "off", false), agora("paused", "off", false)
		igualJSON(t, `{"cmd":"toggle"}`, TocaPausa())
		igualJSON(t, `{"cmd":"next"}`, Proxima())
		igualJSON(t, `{"cmd":"previous"}`, Anterior())
		linha, vale := Toca(pausado)
		igualJSON(t, `{"cmd":"toggle"}`, linha)
		igual(t, true, vale)
		_, vale = Toca(tocando)
		igual(t, false, vale)
		_, vale = Pausa(tocando)
		igual(t, true, vale)
		_, vale = Pausa(pausado)
		igual(t, false, vale)
	})
	t.Run("pular, volume, repetir e embaralhar vindos de fora", func(t *testing.T) {
		igualJSON(t, `{"cmd":"seek","seconds":90.0}`, PoePosicao(90_000_000))
		igualJSON(t, `{"cmd":"seek","seconds":71.5}`, Pula(10_000_000, progresso))
		igualJSON(t, `{"cmd":"seek","seconds":0.0}`, Pula(-100_000_000, progresso))
		igualJSON(t, `{"cmd":"volume","level":37}`, Volume(0.37))
		linha, vale := Repete("Track")
		igualJSON(t, `{"cmd":"repeat","mode":"one"}`, linha)
		igual(t, true, vale)
		linha, _ = Repete("Playlist")
		igualJSON(t, `{"cmd":"repeat","mode":"all"}`, linha)
		_, vale = Repete("Talvez")
		igual(t, false, vale)
		linha, vale = Embaralha(true)
		igualJSON(t, `{"cmd":"shuffleUpNext"}`, linha)
		igual(t, true, vale)
		_, vale = Embaralha(false)
		igual(t, false, vale)
	})
}

// fio é o barramento de mentira do serviço: guarda o que foi respondido e emitido.
type fio struct {
	respostas []any
	erros     []string
	sinais    []string
	corpos    [][]any
}

func (f *fio) Responde(_ *dbus.Mensagem, assinatura string, valores ...any) {
	// Tudo que o serviço responde tem de caber no fio de verdade.
	if _, erro := dbus.Codifica(assinatura, valores...); erro != nil {
		panic(erro)
	}
	f.respostas = append(f.respostas, valores)
}
func (f *fio) RespondeErro(_ *dbus.Mensagem, nome, _ string) { f.erros = append(f.erros, nome) }
func (f *fio) EmiteSinal(_ dbus.Caminho, iface, membro, assinatura string, valores ...any) error {
	if _, erro := dbus.Codifica(assinatura, valores...); erro != nil {
		panic(erro)
	}
	f.sinais = append(f.sinais, iface+"."+membro)
	f.corpos = append(f.corpos, valores)
	return nil
}

type bancada struct {
	f        *fio
	l        *laco.Teste
	s        *Servico
	estados  map[string]string
	mandados []string
	erguida  int
}

func nova() *bancada {
	b := &bancada{f: &fio{}, l: laco.NovoTeste(), estados: map[string]string{
		"now":      `{"type":"now","status":"playing","track":{"id":"abc","provider":"YOUTUBE","title":"Envolver","artist":"Anitta","durationMs":194000,"artwork":""},"live":false,"repeat":"off","speed":1.0,"upNext":[]}`,
		"progress": `{"type":"progress","position":12.0,"duration":194.0}`,
		"volume":   `{"type":"volume","level":80}`,
	}}
	b.s = NovoServico(b.f, b.l, func(tipo string) string { return b.estados[tipo] }, func(linha string) { b.mandados = append(b.mandados, linha) }, func() { b.erguida++ })
	return b
}

func (b *bancada) chama(iface, membro string, corpo ...any) any {
	antes := len(b.f.respostas)
	b.s.Chamada(&dbus.Mensagem{Tipo: dbus.Chamada, Caminho: caminho, Interface: iface, Membro: membro, Corpo: corpo})
	if len(b.f.respostas) == antes {
		return nil
	}
	if resposta := b.f.respostas[len(b.f.respostas)-1].([]any); len(resposta) > 0 {
		return resposta[0]
	}
	return "ok"
}

func TestServicoRespondeComoUmPlayer(t *testing.T) {
	b := nova()
	igual(t, any(dbus.Variante{Assinatura: "s", Valor: "MkTunes"}), b.chama(props, "Get", raiz, "Identity"))
	igual(t, any(dbus.Variante{Assinatura: "s", Valor: "Playing"}), b.chama(props, "Get", player, "PlaybackStatus"))
	igual(t, any(dbus.Variante{Assinatura: "x", Valor: int64(12000000)}), b.chama(props, "Get", player, "Position"))
	igual(t, any(dbus.Variante{Assinatura: "d", Valor: 0.8}), b.chama(props, "Get", player, "Volume"))
	metadados := b.chama(props, "Get", player, "Metadata").(dbus.Variante)
	igual(t, "a{sv}", metadados.Assinatura)
	igual(t, map[string]dbus.Variante{
		"mpris:trackid": {Assinatura: "o", Valor: dbus.Caminho("/dev/mk/mktunes/track/abc")},
		"xesam:title":   {Assinatura: "s", Valor: "Envolver"},
		"xesam:artist":  {Assinatura: "as", Valor: []string{"Anitta"}},
		"mpris:length":  {Assinatura: "x", Valor: int64(194000000)},
	}, metadados.Valor)

	todas := b.chama(props, "GetAll", raiz).(map[string]dbus.Variante)
	igual(t, []any{false, true, "mktunes", []string{}}, []any{todas["CanQuit"].Valor, todas["CanRaise"].Valor, todas["DesktopEntry"].Valor, todas["SupportedUriSchemes"].Valor})
	igual(t, 15, len(b.chama(props, "GetAll", player).(map[string]dbus.Variante)))
	igual(t, 0, len(b.chama(props, "GetAll", "outra.interface").(map[string]dbus.Variante)))

	xml := b.chama("org.freedesktop.DBus.Introspectable", "Introspect").(string)
	for _, trecho := range []string{`<interface name="org.mpris.MediaPlayer2.Player">`, `<method name="SetPosition">`, `<property name="Metadata" type="a{sv}" access="read"/>`, `<signal name="Seeked">`} {
		if !strings.Contains(xml, trecho) {
			t.Fatalf("falta %s na introspecção", trecho)
		}
	}
	b.s.Chamada(&dbus.Mensagem{Caminho: "/org", Membro: "Introspect"})
	igual(t, []any{`<node><node name="mpris"/></node>`}, b.f.respostas[len(b.f.respostas)-1])
	igual(t, "ok", b.chama("org.freedesktop.DBus.Peer", "Ping"))

	// O que não existe é um erro com nome, não silêncio.
	b.chama(props, "Get", player, "Inventada")
	b.chama(props, "Get", "outra.interface", "X")
	b.chama(props, "Get", player)
	b.chama(player, "Voa")
	b.chama("outra.interface", "Next")
	b.s.Chamada(&dbus.Mensagem{Caminho: "/outro/objeto", Interface: player, Membro: "Next"})
	igual(t, []string{erroDePropriedade, erroDeInterface, erroDeArgumentos, erroDeMetodo, erroDeMetodo, erroDeObjeto}, b.f.erros)
	igual(t, 0, len(b.mandados))
}

func TestOsBotoesDoSistemaChegamComoComandos(t *testing.T) {
	b := nova()
	b.chama(player, "Next")
	b.chama(player, "PlayPause")
	b.chama(player, "SetPosition", dbus.Caminho("/dev/mk/mktunes/track/abc"), int64(30000000))
	b.chama(props, "Set", player, "Volume", dbus.Variante{Assinatura: "d", Valor: 0.5})
	b.chama(raiz, "Raise")
	b.chama(player, "Seek", int64(-2000000))
	b.chama(player, "Pause")
	b.chama(player, "Play") // já toca: não muda nada
	b.chama(props, "Set", player, "LoopStatus", dbus.Variante{Assinatura: "s", Valor: "Playlist"})
	b.chama(props, "Set", player, "Shuffle", dbus.Variante{Assinatura: "b", Valor: true})
	b.chama(props, "Set", player, "Shuffle", dbus.Variante{Assinatura: "b", Valor: false})
	b.chama(props, "Set", player, "Volume", dbus.Variante{Assinatura: "s", Valor: "alto"})
	b.chama(player, "Previous")
	b.chama(player, "Stop")
	b.chama(player, "OpenUri", "https://x")
	b.chama(raiz, "Quit")
	b.chama(player, "Seek", "não é número")
	esperados := []string{`{"cmd":"next"}`, `{"cmd":"toggle"}`, `{"cmd":"seek","seconds":30}`, `{"cmd":"volume","level":50}`, `{"cmd":"seek","seconds":10}`, `{"cmd":"toggle"}`, `{"cmd":"repeat","mode":"all"}`, `{"cmd":"shuffleUpNext"}`, `{"cmd":"previous"}`, `{"cmd":"toggle"}`}
	igual(t, len(esperados), len(b.mandados))
	for i := range esperados {
		igualJSON(t, esperados[i], b.mandados[i])
	}
	igual(t, 1, b.erguida)
	igual(t, 0, len(b.f.erros))
	igual(t, 17, len(b.f.respostas))
}

func TestContaAsMudancasEOsPulos(t *testing.T) {
	b := nova()
	b.s.Mudou()
	igual(t, 0, len(b.f.sinais))
	// A música pausou: só o que mudou vai no sinal.
	b.estados["now"] = strings.Replace(b.estados["now"], `"playing"`, `"paused"`, 1)
	b.s.Mudou()
	igual(t, []string{props + ".PropertiesChanged"}, b.f.sinais)
	igual(t, []any{player, map[string]dbus.Variante{"PlaybackStatus": {Assinatura: "s", Valor: "Paused"}}, []string{}}, b.f.corpos[0])
	igual(t, any(dbus.Variante{Assinatura: "s", Valor: "Paused"}), b.chama(props, "Get", player, "PlaybackStatus"))

	// Tocando, a posição que anda com o relógio não é um pulo.
	b.estados["now"] = strings.Replace(b.estados["now"], `"paused"`, `"playing"`, 1)
	b.s.Mudou()
	b.l.Avanca(500 * time.Millisecond)
	b.estados["progress"] = `{"type":"progress","position":12.5,"duration":194.0}`
	b.s.Mudou()
	igual(t, 2, len(b.f.sinais))
	// Um salto que o relógio não explica é.
	b.l.Avanca(500 * time.Millisecond)
	b.estados["progress"] = `{"type":"progress","position":90.0,"duration":194.0}`
	b.s.Mudou()
	igual(t, player+".Seeked", b.f.sinais[2])
	igual(t, []any{int64(90000000)}, b.f.corpos[2])
	// Volume e faixa nova também são contados.
	b.estados["volume"] = `{"type":"volume","level":30}`
	b.s.Mudou()
	igual(t, map[string]dbus.Variante{"Volume": {Assinatura: "d", Valor: 0.3}}, b.f.corpos[3][1])
}
