// Package eq é o equalizador no papel: as bandas, a validação dos ganhos, o
// texto do arquivo do filter-chain, o da unidade do systemd e as mudanças que
// viram chamadas ao vivo. Não roda nada nem escreve em disco.
package eq

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// Os limites e os nomes do contrato (../../../CONTRATO.md).
const (
	NumBandas = 10
	GanhoMin  = -12.0
	GanhoMax  = 12.0
	PreMin    = -12.0
	PreMax    = 0.0
	Passo     = 0.5
	Q         = 1.0

	NoDoSink      = "mksom.eq"
	NoDaSaida     = "mksom.eq.saida"
	Descricao     = "Mk Equalizador"
	NomeDaUnidade = "mksom-eq.service"
)

// Frequencias são os centros das bandas, em Hz.
var Frequencias = []float64{31, 62, 125, 250, 500, 1000, 2000, 4000, 8000, 16000}

// Canais do grafo: cada um tem a sua corrente de filtros.
var Canais = []string{"L", "R"}

// Ajuste é o que a pessoa escolheu: o ganho de cada banda e o pré-ganho, em dB.
type Ajuste struct {
	Ganhos []float64 `json:"ganhos"`
	Pre    float64   `json:"pre"`
}

// Plano é tudo em zero.
func Plano() Ajuste {
	return Ajuste{Ganhos: make([]float64, NumBandas)}
}

func noPasso(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0) && v*2 == math.Trunc(v*2)
}

// Valida confere o número de bandas, os limites e o passo de meio dB.
func (a Ajuste) Valida() error {
	if len(a.Ganhos) != NumBandas {
		return fmt.Errorf("o equalizador tem %d bandas (vieram %d)", NumBandas, len(a.Ganhos))
	}
	for i, g := range a.Ganhos {
		if !noPasso(g) || g < GanhoMin || g > GanhoMax {
			return fmt.Errorf("o ganho da banda %d tem de ir de −12 a +12 dB, em passos de 0,5", i+1)
		}
	}
	if !noPasso(a.Pre) || a.Pre < PreMin || a.Pre > PreMax {
		return errors.New("o pré-ganho tem de ir de −12 a 0 dB, em passos de 0,5")
	}
	return nil
}

// Igual diz se dois ajustes têm os mesmos valores.
func (a Ajuste) Igual(b Ajuste) bool {
	if len(a.Ganhos) != len(b.Ganhos) || a.Pre != b.Pre {
		return false
	}
	for i := range a.Ganhos {
		if a.Ganhos[i] != b.Ganhos[i] {
			return false
		}
	}
	return true
}

// Copia devolve um ajuste que não divide a lista de ganhos com este.
func (a Ajuste) Copia() Ajuste {
	return Ajuste{Ganhos: append([]float64{}, a.Ganhos...), Pre: a.Pre}
}

// Filtro é o filtro builtin de cada banda: prateleira nas pontas, sino no meio.
func Filtro(banda int) string {
	switch banda {
	case 0:
		return "bq_lowshelf"
	case NumBandas - 1:
		return "bq_highshelf"
	}
	return "bq_peaking"
}

func noDaBanda(canal string, banda int) string {
	return "eq_" + canal + "_" + strconv.Itoa(banda)
}

func noDoPre(canal string) string {
	return "pre_" + canal
}

// Controle é o nome do controle de ganho de uma banda num canal, como o
// filter-chain o publica: "<nó>:<controle>".
func Controle(canal string, banda int) string {
	return noDaBanda(canal, banda) + ":Gain"
}

// ControleDoPre é o nome do controle do pré-ganho de um canal.
func ControleDoPre(canal string) string {
	return noDoPre(canal) + ":Mult"
}

// Mult é o pré-ganho como fator (o filtro linear multiplica, não soma dB).
func Mult(pre float64) float64 {
	return math.Pow(10, pre/20)
}

// db escreve um número com ponto e pelo menos uma casa ("3.0", "-2.5").
func db(v float64) string {
	s := strconv.FormatFloat(v, 'f', -1, 64)
	if !strings.Contains(s, ".") {
		s += ".0"
	}
	return s
}

func fator(v float64) string {
	return strconv.FormatFloat(v, 'f', 6, 64)
}

var alvoSeguro = regexp.MustCompile(`^[A-Za-z0-9._:+-]{1,200}$`)

// AlvoSeguro diz se um node.name pode ir, entre aspas, para o conf: só o que
// um nome de saída de verdade tem, e nunca o próprio equalizador.
func AlvoSeguro(nome string) bool {
	return alvoSeguro.MatchString(nome) && nome != NoDoSink && nome != NoDaSaida
}

// Conf é o arquivo inteiro do filter-chain para `pipewire -c`: a base de um
// processo cliente do PipeWire e o módulo que cria o sink "Mk Equalizador",
// com o pré-ganho e as 10 bandas por canal. `alvo` é o node.name da saída de
// verdade (vazio ou estranho: sem target.object, o WirePlumber escolhe).
func Conf(a Ajuste, alvo string) string {
	var s strings.Builder
	s.WriteString("# Gerado pelo MkSom (mksom-motor). Não edite: é reescrito inteiro a cada mudança.\n")
	s.WriteString("context.properties = {\n    log.level = 2\n}\n")
	s.WriteString("context.spa-libs = {\n    audio.convert.* = audioconvert/libspa-audioconvert\n    support.* = support/libspa-support\n}\n")
	s.WriteString("context.modules = [\n")
	s.WriteString("    { name = libpipewire-module-rt\n        args = { nice.level = -11 }\n        flags = [ ifexists nofail ]\n    }\n")
	s.WriteString("    { name = libpipewire-module-protocol-native }\n")
	s.WriteString("    { name = libpipewire-module-client-node }\n")
	s.WriteString("    { name = libpipewire-module-adapter }\n")
	s.WriteString("    { name = libpipewire-module-filter-chain\n        args = {\n")
	s.WriteString("            node.description = \"" + Descricao + "\"\n")
	s.WriteString("            media.name = \"" + Descricao + "\"\n")
	s.WriteString("            filter.graph = {\n                nodes = [\n")
	for _, c := range Canais {
		s.WriteString("                    { type = builtin name = " + noDoPre(c) + " label = linear control = { \"Mult\" = " + fator(Mult(a.Pre)) + " \"Add\" = 0.0 } }\n")
		for i := 0; i < NumBandas; i++ {
			g := 0.0
			if i < len(a.Ganhos) {
				g = a.Ganhos[i]
			}
			s.WriteString("                    { type = builtin name = " + noDaBanda(c, i) + " label = " + Filtro(i) +
				" control = { \"Freq\" = " + db(Frequencias[i]) + " \"Q\" = " + db(Q) + " \"Gain\" = " + db(g) + " } }\n")
		}
	}
	s.WriteString("                ]\n                links = [\n")
	for _, c := range Canais {
		s.WriteString("                    { output = \"" + noDoPre(c) + ":Out\" input = \"" + noDaBanda(c, 0) + ":In\" }\n")
		for i := 0; i < NumBandas-1; i++ {
			s.WriteString("                    { output = \"" + noDaBanda(c, i) + ":Out\" input = \"" + noDaBanda(c, i+1) + ":In\" }\n")
		}
	}
	s.WriteString("                ]\n")
	s.WriteString("                inputs = [ \"" + noDoPre("L") + ":In\" \"" + noDoPre("R") + ":In\" ]\n")
	s.WriteString("                outputs = [ \"" + noDaBanda("L", NumBandas-1) + ":Out\" \"" + noDaBanda("R", NumBandas-1) + ":Out\" ]\n")
	s.WriteString("            }\n")
	s.WriteString("            audio.channels = 2\n            audio.position = [ FL FR ]\n")
	s.WriteString("            capture.props = {\n                node.name = \"" + NoDoSink + "\"\n                media.class = Audio/Sink\n            }\n")
	s.WriteString("            playback.props = {\n                node.name = \"" + NoDaSaida + "\"\n                node.passive = true\n")
	if AlvoSeguro(alvo) {
		s.WriteString("                target.object = \"" + alvo + "\"\n")
	}
	s.WriteString("            }\n        }\n    }\n]\n")
	return s.String()
}

// Par é um controle e o valor que ele passa a ter.
type Par struct {
	Nome  string
	Valor string
}

// Mudanca é uma chamada ao vivo: os controles de uma banda (ou do pré-ganho)
// nos dois canais.
type Mudanca struct {
	Pares []Par
}

// Props é o valor do parâmetro Props para o `pw-cli set-param`.
func (m Mudanca) Props() string {
	var s strings.Builder
	s.WriteString("{ params = [")
	for _, p := range m.Pares {
		s.WriteString(" \"" + p.Nome + "\" " + p.Valor)
	}
	s.WriteString(" ] }")
	return s.String()
}

// Nomes são os controles que esta mudança toca.
func (m Mudanca) Nomes() []string {
	out := make([]string, 0, len(m.Pares))
	for _, p := range m.Pares {
		out = append(out, p.Nome)
	}
	return out
}

// Mudancas compara dois ajustes válidos e devolve só o que mudou: o
// pré-ganho primeiro, depois as bandas em ordem.
func Mudancas(antes, depois Ajuste) []Mudanca {
	var out []Mudanca
	if antes.Pre != depois.Pre {
		var m Mudanca
		for _, c := range Canais {
			m.Pares = append(m.Pares, Par{ControleDoPre(c), fator(Mult(depois.Pre))})
		}
		out = append(out, m)
	}
	for i := 0; i < NumBandas && i < len(depois.Ganhos); i++ {
		if i < len(antes.Ganhos) && antes.Ganhos[i] == depois.Ganhos[i] {
			continue
		}
		var m Mudanca
		for _, c := range Canais {
			m.Pares = append(m.Pares, Par{Controle(c, i), db(depois.Ganhos[i])})
		}
		out = append(out, m)
	}
	return out
}

// Unidade é o texto do mksom-eq.service: um pipewire à parte, só com o
// filter-chain, que sobe depois do PipeWire e do WirePlumber e cai com o
// PipeWire. O caminho do conf vai entre aspas, com o que o systemd expande
// (% e $) dobrado.
func Unidade(pipewire, conf string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, `%`, `%%`, `$`, `$$`)
	return "# Gerado pelo MkSom (mksom-motor). Não edite: é reescrito ao ligar o equalizador.\n" +
		"[Unit]\n" +
		"Description=" + Descricao + " (filter-chain do PipeWire)\n" +
		"After=pipewire.service wireplumber.service\n" +
		"BindsTo=pipewire.service\n" +
		"\n[Service]\n" +
		"ExecStart=" + pipewire + " -c \"" + r.Replace(conf) + "\"\n" +
		"Restart=on-failure\n" +
		"RestartSec=2\n" +
		"\n[Install]\n" +
		"WantedBy=default.target\n"
}
