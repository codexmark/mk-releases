// SPDX-License-Identifier: GPL-3.0-or-later

package mpris

import (
	"reflect"
	"time"

	"mktunes/internal/dbus"
	"mktunes/internal/laco"
)

const (
	// Nome é o nome bem conhecido do player no barramento.
	Nome    = "org.mpris.MediaPlayer2.mktunes"
	caminho = dbus.Caminho("/org/mpris/MediaPlayer2")
	raiz    = "org.mpris.MediaPlayer2"
	player  = "org.mpris.MediaPlayer2.Player"
	props   = "org.freedesktop.DBus.Properties"
	// folgaDoPulo: um salto de posição que o relógio não explica é um pulo.
	folgaDoPulo = 1_500_000
)

const introspeccao = `<!DOCTYPE node PUBLIC "-//freedesktop//DTD D-BUS Object Introspection 1.0//EN" "http://www.freedesktop.org/standards/dbus/1.0/introspect.dtd">
<node>
 <interface name="org.freedesktop.DBus.Introspectable">
  <method name="Introspect"><arg name="xml" type="s" direction="out"/></method>
 </interface>
 <interface name="org.freedesktop.DBus.Peer">
  <method name="Ping"/>
 </interface>
 <interface name="org.freedesktop.DBus.Properties">
  <method name="Get"><arg name="interface" type="s" direction="in"/><arg name="property" type="s" direction="in"/><arg name="value" type="v" direction="out"/></method>
  <method name="GetAll"><arg name="interface" type="s" direction="in"/><arg name="properties" type="a{sv}" direction="out"/></method>
  <method name="Set"><arg name="interface" type="s" direction="in"/><arg name="property" type="s" direction="in"/><arg name="value" type="v" direction="in"/></method>
  <signal name="PropertiesChanged"><arg name="interface" type="s"/><arg name="changed" type="a{sv}"/><arg name="invalidated" type="as"/></signal>
 </interface>
 <interface name="org.mpris.MediaPlayer2">
  <method name="Raise"/>
  <method name="Quit"/>
  <property name="CanQuit" type="b" access="read"/>
  <property name="CanRaise" type="b" access="read"/>
  <property name="HasTrackList" type="b" access="read"/>
  <property name="Identity" type="s" access="read"/>
  <property name="DesktopEntry" type="s" access="read"/>
  <property name="SupportedUriSchemes" type="as" access="read"/>
  <property name="SupportedMimeTypes" type="as" access="read"/>
 </interface>
 <interface name="org.mpris.MediaPlayer2.Player">
  <method name="Next"/>
  <method name="Previous"/>
  <method name="Pause"/>
  <method name="PlayPause"/>
  <method name="Stop"/>
  <method name="Play"/>
  <method name="Seek"><arg name="Offset" type="x" direction="in"/></method>
  <method name="SetPosition"><arg name="TrackId" type="o" direction="in"/><arg name="Position" type="x" direction="in"/></method>
  <method name="OpenUri"><arg name="Uri" type="s" direction="in"/></method>
  <signal name="Seeked"><arg name="Position" type="x"/></signal>
  <property name="PlaybackStatus" type="s" access="read"/>
  <property name="LoopStatus" type="s" access="readwrite"/>
  <property name="Rate" type="d" access="readwrite"/>
  <property name="Shuffle" type="b" access="readwrite"/>
  <property name="Metadata" type="a{sv}" access="read"/>
  <property name="Volume" type="d" access="readwrite"/>
  <property name="Position" type="x" access="read"/>
  <property name="MinimumRate" type="d" access="read"/>
  <property name="MaximumRate" type="d" access="read"/>
  <property name="CanGoNext" type="b" access="read"/>
  <property name="CanGoPrevious" type="b" access="read"/>
  <property name="CanPlay" type="b" access="read"/>
  <property name="CanPause" type="b" access="read"/>
  <property name="CanSeek" type="b" access="read"/>
  <property name="CanControl" type="b" access="read"/>
 </interface>
</node>`

// Os nós acima do objeto só dizem qual é o filho.
var pais = map[dbus.Caminho]string{"/": "org", "/org": "mpris", "/org/mpris": "MediaPlayer2"}

// Barramento é o que o serviço usa da conexão (os testes põem um dublê).
type Barramento interface {
	Responde(a *dbus.Mensagem, assinatura string, valores ...any)
	RespondeErro(a *dbus.Mensagem, nome, texto string)
	EmiteSinal(caminho dbus.Caminho, iface, membro, assinatura string, valores ...any) error
}

// Servico atende as chamadas MPRIS e conta à ilha cada mudança. Tudo nele roda
// no laço.
type Servico struct {
	barramento Barramento
	laco       laco.Laco
	// Estado lê a mensagem atual de uma área do daemon ("now", "progress", "volume").
	estado func(tipo string) string
	// Manda entrega um comando ao daemon, como a janela faria.
	manda func(linha string)
	// Levanta abre (ou traz para a frente) a janela.
	levanta func()

	mostrado      map[string]any
	ultimaPosicao int64
	ultimaVez     time.Time
}

// NovoServico liga o MPRIS ao daemon; Mudou tem de ser chamada quando "now",
// "progress" ou "volume" mudam, e Chamada para cada chamada que chega.
func NovoServico(b Barramento, l laco.Laco, estado func(tipo string) string, manda func(linha string), levanta func()) *Servico {
	s := &Servico{barramento: b, laco: l, estado: estado, manda: manda, levanta: levanta}
	s.mostrado = s.jogador()
	return s
}

func (s *Servico) jogador() map[string]any {
	return Jogador(s.estado("now"), s.estado("progress"), s.estado("volume"))
}

// variante embrulha um valor simples do modelo com o tipo D-Bus dele.
func variante(valor any) dbus.Variante {
	switch v := valor.(type) {
	case string:
		return dbus.Variante{Assinatura: "s", Valor: v}
	case bool:
		return dbus.Variante{Assinatura: "b", Valor: v}
	case float64:
		return dbus.Variante{Assinatura: "d", Valor: v}
	case int64:
		return dbus.Variante{Assinatura: "x", Valor: v}
	case dbus.Caminho:
		return dbus.Variante{Assinatura: "o", Valor: v}
	case []string:
		return dbus.Variante{Assinatura: "as", Valor: v}
	case map[string]any:
		return dbus.Variante{Assinatura: "a{sv}", Valor: variantes(v)}
	}
	return dbus.Variante{Assinatura: "s", Valor: ""}
}

func variantes(valores map[string]any) map[string]dbus.Variante {
	embrulhados := make(map[string]dbus.Variante, len(valores))
	for chave, valor := range valores {
		embrulhados[chave] = variante(valor)
	}
	return embrulhados
}

func (s *Servico) propriedades(iface string) (map[string]dbus.Variante, bool) {
	switch iface {
	case raiz:
		return variantes(map[string]any{
			"CanQuit": false, "CanRaise": true, "HasTrackList": false, "Identity": "MkTunes", "DesktopEntry": "mktunes",
			"SupportedUriSchemes": []string{}, "SupportedMimeTypes": []string{},
		}), true
	case player:
		return variantes(s.jogador()), true
	}
	return nil, false
}

func (s *Servico) despacha(linha string, vale bool) {
	if vale {
		s.manda(linha)
	}
}

const (
	erroDeMetodo      = "org.freedesktop.DBus.Error.UnknownMethod"
	erroDeObjeto      = "org.freedesktop.DBus.Error.UnknownObject"
	erroDeInterface   = "org.freedesktop.DBus.Error.UnknownInterface"
	erroDePropriedade = "org.freedesktop.DBus.Error.UnknownProperty"
	erroDeArgumentos  = "org.freedesktop.DBus.Error.InvalidArgs"
)

// Chamada trata uma chamada de método que chegou pelo barramento.
func (s *Servico) Chamada(m *dbus.Mensagem) {
	b := s.barramento
	if filho, ePai := pais[m.Caminho]; ePai {
		if m.Membro == "Introspect" {
			b.Responde(m, "s", `<node><node name="`+filho+`"/></node>`)
			return
		}
		if m.Interface == "org.freedesktop.DBus.Peer" && m.Membro == "Ping" {
			b.Responde(m, "")
			return
		}
	}
	if m.Caminho != caminho {
		b.RespondeErro(m, erroDeObjeto, "não há objeto em "+string(m.Caminho))
		return
	}
	textos := func(n int) ([]string, bool) {
		if len(m.Corpo) < n {
			return nil, false
		}
		lidos := make([]string, n)
		for i := range n {
			texto, ok := m.Corpo[i].(string)
			if !ok {
				return nil, false
			}
			lidos[i] = texto
		}
		return lidos, true
	}
	// A interface pode vir vazia: o membro decide.
	switch membro := m.Membro; {
	case membro == "Introspect" && (m.Interface == "" || m.Interface == "org.freedesktop.DBus.Introspectable"):
		b.Responde(m, "s", introspeccao)
	case membro == "Ping" && (m.Interface == "" || m.Interface == "org.freedesktop.DBus.Peer"):
		b.Responde(m, "")
	case membro == "GetAll" && (m.Interface == "" || m.Interface == props):
		argumentos, ok := textos(1)
		if !ok {
			b.RespondeErro(m, erroDeArgumentos, "GetAll pede o nome da interface")
			return
		}
		todas, _ := s.propriedades(argumentos[0])
		if todas == nil {
			todas = map[string]dbus.Variante{}
		}
		b.Responde(m, "a{sv}", todas)
	case membro == "Get" && (m.Interface == "" || m.Interface == props):
		argumentos, ok := textos(2)
		if !ok {
			b.RespondeErro(m, erroDeArgumentos, "Get pede a interface e a propriedade")
			return
		}
		todas, conhecida := s.propriedades(argumentos[0])
		if !conhecida {
			b.RespondeErro(m, erroDeInterface, "interface desconhecida: "+argumentos[0])
			return
		}
		valor, tem := todas[argumentos[1]]
		if !tem {
			b.RespondeErro(m, erroDePropriedade, "propriedade desconhecida: "+argumentos[1])
			return
		}
		b.Responde(m, "v", valor)
	case membro == "Set" && (m.Interface == "" || m.Interface == props):
		argumentos, ok := textos(2)
		if !ok || len(m.Corpo) < 3 {
			b.RespondeErro(m, erroDeArgumentos, "Set pede a interface, a propriedade e o valor")
			return
		}
		valor, _ := m.Corpo[2].(dbus.Variante)
		if argumentos[0] == player {
			switch argumentos[1] {
			case "Volume":
				if nivel, ok := valor.Valor.(float64); ok {
					s.manda(Volume(nivel))
				}
			case "LoopStatus":
				if situacao, ok := valor.Valor.(string); ok {
					s.despacha(Repete(situacao))
				}
			case "Shuffle":
				if ligado, ok := valor.Valor.(bool); ok {
					s.despacha(Embaralha(ligado))
				}
			}
		}
		b.Responde(m, "")
	case m.Interface == raiz || (m.Interface == "" && (membro == "Raise" || membro == "Quit")):
		switch membro {
		case "Raise":
			s.levanta()
		case "Quit":
		default:
			b.RespondeErro(m, erroDeMetodo, "método desconhecido: "+membro)
			return
		}
		b.Responde(m, "")
	case m.Interface == player || m.Interface == "":
		switch membro {
		case "Next":
			s.manda(Proxima())
		case "Previous":
			s.manda(Anterior())
		case "PlayPause":
			s.manda(TocaPausa())
		case "Pause", "Stop":
			s.despacha(Pausa(s.estado("now")))
		case "Play":
			s.despacha(Toca(s.estado("now")))
		case "Seek":
			if deslocamento, ok := primeiro[int64](m.Corpo, 0); ok {
				s.manda(Pula(deslocamento, s.estado("progress")))
			}
		case "SetPosition":
			if posicao, ok := primeiro[int64](m.Corpo, 1); ok {
				s.manda(PoePosicao(posicao))
			}
		case "OpenUri":
		default:
			b.RespondeErro(m, erroDeMetodo, "método desconhecido: "+membro)
			return
		}
		b.Responde(m, "")
	default:
		b.RespondeErro(m, erroDeMetodo, "método desconhecido: "+m.Interface+"."+membro)
	}
}

func primeiro[T any](corpo []any, indice int) (T, bool) {
	var zero T
	if indice >= len(corpo) {
		return zero, false
	}
	valor, ok := corpo[indice].(T)
	return valor, ok
}

// Mudou compara o que a ilha já sabe com o estado de agora e conta a
// diferença: PropertiesChanged para o que mudou (a posição não entra: quem
// quer a lê) e Seeked quando a posição saltou.
func (s *Servico) Mudou() {
	atual := s.jogador()
	mudou := map[string]any{}
	for chave, valor := range atual {
		if chave != "Position" && !reflect.DeepEqual(s.mostrado[chave], valor) {
			mudou[chave] = valor
		}
	}
	s.mostrado = atual
	if len(mudou) > 0 {
		_ = s.barramento.EmiteSinal(caminho, props, "PropertiesChanged", "sa{sv}as", player, variantes(mudou), []string{})
	}
	posicao, _ := atual["Position"].(int64)
	agora := s.laco.Agora()
	esperada := s.ultimaPosicao
	if atual["PlaybackStatus"] == "Playing" {
		esperada += agora.Sub(s.ultimaVez).Microseconds()
	}
	if diferenca := posicao - esperada; !s.ultimaVez.IsZero() && (diferenca > folgaDoPulo || diferenca < -folgaDoPulo) {
		_ = s.barramento.EmiteSinal(caminho, player, "Seeked", "x", posicao)
	}
	s.ultimaPosicao, s.ultimaVez = posicao, agora
}
