package pw

import (
	"testing"
)

// Um pw-dump pequeno com a forma do de verdade: um nó de saída, um de
// entrada, um cliente, um fluxo, um dispositivo e os metadados "default".
const dump = `[
  {"id": 0, "type": "PipeWire:Interface:Core", "version": 4, "info": {"name": "pipewire-0", "props": {"core.name": "pipewire-0"}}},
  {"id": 40, "type": "PipeWire:Interface:Device", "info": {"props": {"device.name": "alsa_card.exemplo", "media.class": "Audio/Device"}}},
  {"id": 51, "type": "PipeWire:Interface:Node", "version": 3, "info": {
     "state": "suspended",
     "props": {"node.name": "alsa_output.exemplo.analog-stereo", "node.description": "Alto-falantes", "media.class": "Audio/Sink", "object.serial": 310},
     "params": {"Props": [{"volume": 1.0, "mute": false, "channelVolumes": [0.064, 0.064]}, {"params": []}]}}},
  {"id": 52, "type": "PipeWire:Interface:Node", "version": 3, "info": {
     "props": {"node.name": "alsa_input.exemplo.analog-stereo", "node.description": "Microfone interno", "node.nick": "Mic", "media.class": "Audio/Source", "object.serial": 311},
     "params": {"Props": [{"volume": 1.0, "mute": true, "channelVolumes": [0.125, 0.125], "softMute": false}]}}},
  {"id": 53, "type": "PipeWire:Interface:Node", "info": {
     "props": {"node.name": "alsa_input.usb-exemplo.mono", "node.nick": "USB", "media.class": "Audio/Source", "object.serial": "312"},
     "params": {"Props": [{"channelVolumes": [1.0]}]}}},
  {"id": 54, "type": "PipeWire:Interface:Node", "info": {"props": {"node.name": "alsa_output.exemplo.analog-stereo.monitor", "media.class": "Audio/Source"}}},
  {"id": 55, "type": "PipeWire:Interface:Node", "info": {"props": {"node.name": "navegador", "media.class": "Stream/Output/Audio"}}},
  {"id": 56, "type": "PipeWire:Interface:Node", "info": {"props": {"node.name": "camera", "media.class": "Video/Source"}}},
  {"id": 57, "type": "PipeWire:Interface:Node", "info": {"props": {"node.name": "filtro.voz", "node.description": "Voz limpa", "media.class": "Audio/Source/Virtual"}}},
  {"id": 71, "type": "PipeWire:Interface:Node", "info": {"props": {"node.name": "mksom.eq", "node.description": "Mk Equalizador", "media.class": "Audio/Sink"}}},
  {"id": 36, "type": "PipeWire:Interface:Metadata", "props": {"metadata.name": "default"}, "metadata": [
     {"subject": 0, "key": "default.configured.audio.sink", "type": "Spa:String:JSON", "value": {"name": "outra.coisa"}},
     {"subject": 0, "key": "default.audio.sink", "type": "Spa:String:JSON", "value": {"name": "mksom.eq"}},
     {"subject": 0, "key": "default.audio.source", "type": "Spa:String:JSON", "value": {"name": "alsa_input.exemplo.analog-stereo"}}
  ]},
  {"id": 37, "type": "PipeWire:Interface:Metadata", "props": {"metadata.name": "settings"}, "metadata": [
     {"subject": 0, "key": "default.audio.sink", "value": {"name": "nao.e.daqui"}}
  ]}
]`

func TestLe_NosPadraoVolumeEMudo(t *testing.T) {
	g, err := Le([]byte(dump))
	if err != nil {
		t.Fatal(err)
	}
	if g.SaidaPadrao != "mksom.eq" || g.EntradaPadrao != "alsa_input.exemplo.analog-stereo" {
		t.Errorf("padrões: %q %q", g.SaidaPadrao, g.EntradaPadrao)
	}
	saidas := g.Saidas()
	if len(saidas) != 2 || saidas[0].ID != 51 || saidas[1].Nome != "mksom.eq" {
		t.Errorf("saídas: %+v", saidas)
	}
	if s := saidas[0]; s.Descricao != "Alto-falantes" || s.Serie != 310 || s.Volume != 40 || s.Mudo {
		t.Errorf("alto-falantes: %+v", s)
	}
	// As entradas: Audio/Source (e a virtual), sem o monitor, sem vídeo, sem fluxo.
	entradas := g.Entradas()
	if len(entradas) != 3 {
		t.Fatalf("entradas: %+v", entradas)
	}
	mic := entradas[0]
	if mic.ID != 52 || mic.Descricao != "Microfone interno" || mic.Volume != 50 || !mic.Mudo || mic.Serie != 311 {
		t.Errorf("mic: %+v", mic)
	}
	// Sem node.description vale o nick; o serial veio como texto.
	if usb := entradas[1]; usb.ID != 53 || usb.Descricao != "USB" || usb.Volume != 100 || usb.Serie != 312 {
		t.Errorf("usb: %+v", usb)
	}
	// Sem Props: volume desconhecido, sem serial.
	if v := entradas[2]; v.ID != 57 || v.Volume != -1 || v.Serie != -1 {
		t.Errorf("virtual: %+v", v)
	}
	if n, ok := g.Saida("mksom.eq"); !ok || n.ID != 71 {
		t.Errorf("saída por nome: %+v %v", n, ok)
	}
	if _, ok := g.Saida("alsa_input.exemplo.analog-stereo"); ok {
		t.Error("uma entrada não é saída")
	}
	if n, ok := g.Entrada(53); !ok || n.Nome != "alsa_input.usb-exemplo.mono" {
		t.Errorf("entrada por id: %+v %v", n, ok)
	}
	if _, ok := g.Entrada(51); ok {
		t.Error("uma saída não é entrada")
	}
	if _, ok := g.Entrada(54); ok {
		t.Error("o monitor não é entrada")
	}
}

func TestLe_Tolerante(t *testing.T) {
	casos := map[string]string{
		"vazio":                       `[]`,
		"itens tortos":                `[1, "x", null, [], {}, {"id": "x", "type": 3}, {"id": 9, "type": "PipeWire:Interface:Node"}, {"id": 9, "type": "PipeWire:Interface:Node", "info": 4}, {"id": 9, "type": "PipeWire:Interface:Node", "info": {"props": []}}]`,
		"metadado sem valor":          `[{"id": 1, "type": "PipeWire:Interface:Metadata", "props": {"metadata.name": "default"}, "metadata": [{"key": "default.audio.sink"}, 7, {"key": 3, "value": 1}]}]`,
		"metadado que não é lista":    `[{"id": 1, "type": "PipeWire:Interface:Metadata", "metadata": {"a": 1}}]`,
		"props com valores estranhos": `[{"id": 3, "type": "PipeWire:Interface:Node", "info": {"props": {"node.name": 5, "media.class": ["x"]}, "params": {"Props": "x"}}}]`,
	}
	for nome, texto := range casos {
		g, err := Le([]byte(texto))
		if err != nil {
			t.Errorf("%s: %v", nome, err)
		}
		if len(g.Saidas()) != 0 || len(g.Entradas()) != 0 || g.SaidaPadrao != "" {
			t.Errorf("%s: %+v", nome, g)
		}
	}
	// O que não é uma lista JSON é erro: o pw-dump não respondeu direito.
	for _, texto := range []string{"", "   ", "{}", "não é json", `[{"id": 1}`, "null"} {
		if _, err := Le([]byte(texto)); err == nil {
			t.Errorf("%q: tinha de dar erro", texto)
		}
	}
}

func TestLe_PadraoComoTextoEConfiguradoQuandoFaltaOEfetivo(t *testing.T) {
	g, err := Le([]byte(`[
	  {"id": "51", "type": "PipeWire:Interface:Node", "info": {"props": {"node.name": "a", "media.class": "Audio/Sink"}, "params": {"Props": [{"volume": 0.5, "mute": 1}]}}},
	  {"id": 36, "type": "PipeWire:Interface:Metadata", "props": {"metadata.name": "default"}, "metadata": [
	    {"key": "default.configured.audio.sink", "value": "{ \"name\": \"a\" }"},
	    {"key": "default.audio.source", "value": "b"}
	  ]}]`))
	if err != nil {
		t.Fatal(err)
	}
	if g.SaidaPadrao != "a" || g.EntradaPadrao != "b" {
		t.Errorf("padrões: %q %q", g.SaidaPadrao, g.EntradaPadrao)
	}
	// O id veio como texto; sem channelVolumes vale o "volume"; sem descrição, o nome.
	if s := g.Saidas(); len(s) != 1 || s[0].ID != 51 || s[0].Volume != 79 || s[0].Descricao != "a" || s[0].Mudo {
		t.Errorf("%+v", s)
	}
}

func TestPorcento_EscalaCubicaDoWpctl(t *testing.T) {
	for linear, quer := range map[float64]int{0: 0, 1: 100, 0.125: 50, 0.001: 10, 3.375: 150, 0.421875: 75} {
		if got := Porcento(linear); got != quer {
			t.Errorf("%v: %d (quer %d)", linear, got, quer)
		}
	}
	if Porcento(-1) != 0 {
		t.Error("negativo")
	}
}
