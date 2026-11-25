package eq

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func ajuste(pre float64, ganhos ...float64) Ajuste {
	return Ajuste{Ganhos: ganhos, Pre: pre}
}

var deTeste = ajuste(-4.5, 3, 2.5, 0, -1, 0, 0.5, 0, -12, 12, 4)

func TestValida(t *testing.T) {
	bons := []Ajuste{Plano(), deTeste, ajuste(-12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12), ajuste(0, -12, 0, 0, 0, 0, 0, 0, 0, 0, 0)}
	for _, a := range bons {
		if err := a.Valida(); err != nil {
			t.Errorf("%v: %v", a, err)
		}
	}
	nan := 0.0
	nan = nan / nan
	ruins := map[string]Ajuste{
		"nove bandas":        ajuste(0, 0, 0, 0, 0, 0, 0, 0, 0, 0),
		"onze bandas":        ajuste(0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0),
		"sem ganhos":         {Pre: 0},
		"acima de 12":        ajuste(0, 12.5, 0, 0, 0, 0, 0, 0, 0, 0, 0),
		"abaixo de -12":      ajuste(0, 0, 0, 0, 0, 0, 0, 0, 0, 0, -12.5),
		"fora do passo":      ajuste(0, 0.3, 0, 0, 0, 0, 0, 0, 0, 0, 0),
		"NaN":                ajuste(0, nan, 0, 0, 0, 0, 0, 0, 0, 0, 0),
		"pré positivo":       ajuste(0.5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0),
		"pré abaixo de -12":  ajuste(-12.5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0),
		"pré fora do passo":  ajuste(-0.2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0),
		"pré que não é nada": ajuste(nan, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0),
	}
	for nome, a := range ruins {
		if err := a.Valida(); err == nil {
			t.Errorf("%s: tinha de recusar", nome)
		}
	}
}

func TestBandas_PrimeiraEUltimaSaoPrateleiras(t *testing.T) {
	if !reflect.DeepEqual(Frequencias, []float64{31, 62, 125, 250, 500, 1000, 2000, 4000, 8000, 16000}) {
		t.Errorf("frequências: %v", Frequencias)
	}
	quer := []string{"bq_lowshelf", "bq_peaking", "bq_peaking", "bq_peaking", "bq_peaking", "bq_peaking", "bq_peaking", "bq_peaking", "bq_peaking", "bq_highshelf"}
	for i, q := range quer {
		if Filtro(i) != q {
			t.Errorf("banda %d: %s", i, Filtro(i))
		}
	}
}

func TestControles_NomesEstaveis(t *testing.T) {
	if Controle("L", 3) != "eq_L_3:Gain" || Controle("R", 9) != "eq_R_9:Gain" || ControleDoPre("L") != "pre_L:Mult" {
		t.Errorf("%s %s %s", Controle("L", 3), Controle("R", 9), ControleDoPre("L"))
	}
	if Mult(0) != 1 || fmt.Sprintf("%.6f", Mult(-6)) != "0.501187" || fmt.Sprintf("%.6f", Mult(-12)) != "0.251189" {
		t.Errorf("Mult: %v %v %v", Mult(0), Mult(-6), Mult(-12))
	}
}

// bloco devolve a linha do conf que declara o nó com esse nome.
func linhaDoNo(t *testing.T, conf, nome string) string {
	t.Helper()
	for _, l := range strings.Split(conf, "\n") {
		if strings.Contains(l, "name = "+nome+" ") {
			return strings.TrimSpace(l)
		}
	}
	t.Fatalf("o conf não tem o nó %s", nome)
	return ""
}

func TestConf_EstruturaNomesEValores(t *testing.T) {
	conf := Conf(deTeste, "alsa_output.pci-0000_00_1f.3.analog-stereo")

	// É uma configuração inteira para `pipewire -c`: a base e o filter-chain.
	for _, trecho := range []string{
		"context.properties = {", "context.spa-libs = {", "context.modules = [",
		"audio.convert.* = audioconvert/libspa-audioconvert", "support.* = support/libspa-support",
		"{ name = libpipewire-module-rt", "flags = [ ifexists nofail ]",
		"{ name = libpipewire-module-protocol-native }", "{ name = libpipewire-module-client-node }",
		"{ name = libpipewire-module-adapter }", "{ name = libpipewire-module-filter-chain",
		`node.description = "Mk Equalizador"`, `media.name = "Mk Equalizador"`,
		"audio.channels = 2", "audio.position = [ FL FR ]",
		`node.name = "mksom.eq"`, "media.class = Audio/Sink",
		`node.name = "mksom.eq.saida"`, "node.passive = true",
		`target.object = "alsa_output.pci-0000_00_1f.3.analog-stereo"`,
		`inputs = [ "pre_L:In" "pre_R:In" ]`, `outputs = [ "eq_L_9:Out" "eq_R_9:Out" ]`,
	} {
		if !strings.Contains(conf, trecho) {
			t.Errorf("falta no conf: %s", trecho)
		}
	}
	// Chaves e colchetes fecham.
	for _, par := range []string{"{}", "[]"} {
		if a, f := strings.Count(conf, par[:1]), strings.Count(conf, par[1:]); a != f {
			t.Errorf("%s: %d abrem, %d fecham", par, a, f)
		}
	}
	// 22 nós: por canal, o pré e as 10 bandas. Todos builtin.
	if n := strings.Count(conf, "{ type = builtin name = "); n != 22 {
		t.Errorf("%d nós", n)
	}
	if n := len(regexp.MustCompile(`label = (bq_lowshelf|bq_peaking|bq_highshelf|linear) `).FindAllString(conf, -1)); n != 22 {
		t.Errorf("%d rótulos conhecidos", n)
	}
	if strings.Contains(conf, "ladspa") || strings.Contains(conf, "lv2") {
		t.Error("o conf só pode usar filtros builtin")
	}
	for _, c := range []string{"L", "R"} {
		if got, quer := linhaDoNo(t, conf, "pre_"+c), `{ type = builtin name = pre_`+c+` label = linear control = { "Mult" = 0.595662 "Add" = 0.0 } }`; got != quer {
			t.Errorf("pré %s:\n %s\n %s", c, got, quer)
		}
		quer := []string{
			`label = bq_lowshelf control = { "Freq" = 31.0 "Q" = 1.0 "Gain" = 3.0 } }`,
			`label = bq_peaking control = { "Freq" = 62.0 "Q" = 1.0 "Gain" = 2.5 } }`,
			`label = bq_peaking control = { "Freq" = 125.0 "Q" = 1.0 "Gain" = 0.0 } }`,
			`label = bq_peaking control = { "Freq" = 250.0 "Q" = 1.0 "Gain" = -1.0 } }`,
			`label = bq_peaking control = { "Freq" = 500.0 "Q" = 1.0 "Gain" = 0.0 } }`,
			`label = bq_peaking control = { "Freq" = 1000.0 "Q" = 1.0 "Gain" = 0.5 } }`,
			`label = bq_peaking control = { "Freq" = 2000.0 "Q" = 1.0 "Gain" = 0.0 } }`,
			`label = bq_peaking control = { "Freq" = 4000.0 "Q" = 1.0 "Gain" = -12.0 } }`,
			`label = bq_peaking control = { "Freq" = 8000.0 "Q" = 1.0 "Gain" = 12.0 } }`,
			`label = bq_highshelf control = { "Freq" = 16000.0 "Q" = 1.0 "Gain" = 4.0 } }`,
		}
		for i, q := range quer {
			nome := fmt.Sprintf("eq_%s_%d", c, i)
			if got := linhaDoNo(t, conf, nome); got != "{ type = builtin name = "+nome+" "+q {
				t.Errorf("%s:\n %s", nome, got)
			}
		}
		// A corrente: pré → 0 → 1 → … → 9.
		elos := []string{`{ output = "pre_` + c + `:Out" input = "eq_` + c + `_0:In" }`}
		for i := 0; i < 9; i++ {
			elos = append(elos, fmt.Sprintf(`{ output = "eq_%s_%d:Out" input = "eq_%s_%d:In" }`, c, i, c, i+1))
		}
		for _, e := range elos {
			if strings.Count(conf, e) != 1 {
				t.Errorf("elo: %s", e)
			}
		}
	}
	if n := strings.Count(conf, "{ output = "); n != 20 {
		t.Errorf("%d elos", n)
	}
	// O nome de cada controle do ajuste ao vivo existe no conf: nó + ":" + controle.
	for _, c := range []string{"L", "R"} {
		for i := 0; i < NumBandas; i++ {
			no, controle, _ := strings.Cut(Controle(c, i), ":")
			if !strings.Contains(linhaDoNo(t, conf, no), `"`+controle+`" = `) {
				t.Errorf("o controle %s não está no conf", Controle(c, i))
			}
		}
		no, controle, _ := strings.Cut(ControleDoPre(c), ":")
		if !strings.Contains(linhaDoNo(t, conf, no), `"`+controle+`" = `) {
			t.Errorf("o controle %s não está no conf", ControleDoPre(c))
		}
	}
	// O mesmo ajuste dá o mesmo arquivo.
	if conf != Conf(deTeste, "alsa_output.pci-0000_00_1f.3.analog-stereo") {
		t.Error("o conf não é determinístico")
	}
}

func TestConf_SemAlvoOuComAlvoEstranhoNaoEscreveTargetObject(t *testing.T) {
	for _, alvo := range []string{"", "mksom.eq", "mksom.eq.saida", "a\"\n}\ncontext.exec = [ { path = \"x\" } ]", "a b", "a\"b", "a\\b", "a}", strings.Repeat("a", 300)} {
		conf := Conf(Plano(), alvo)
		if strings.Contains(conf, "target.object") || strings.Contains(conf, "context.exec") {
			t.Errorf("alvo %q entrou no conf", alvo)
		}
	}
	for _, alvo := range []string{"alsa_output.usb-Marca_Modelo-00.analog-stereo", "bluez_output.AA_BB_CC_DD_EE_FF.1", "alsa_output.pci-0000_00_1f.3.hdmi-stereo-extra1", "saida:1+2"} {
		if !strings.Contains(Conf(Plano(), alvo), `target.object = "`+alvo+`"`) {
			t.Errorf("alvo %q não entrou", alvo)
		}
	}
}

func TestMudancas_SoOQueMudou(t *testing.T) {
	antes := deTeste
	depois := ajuste(-4.5, 3, 2.5, 0, 4, 0, 0.5, 0, -12, 12, 3.5)
	m := Mudancas(antes, depois)
	if len(m) != 2 {
		t.Fatalf("%v", m)
	}
	if got := m[0].Props(); got != `{ params = [ "eq_L_3:Gain" 4.0 "eq_R_3:Gain" 4.0 ] }` {
		t.Errorf("banda 3: %s", got)
	}
	if got := m[1].Props(); got != `{ params = [ "eq_L_9:Gain" 3.5 "eq_R_9:Gain" 3.5 ] }` {
		t.Errorf("banda 9: %s", got)
	}
	if !reflect.DeepEqual(m[0].Nomes(), []string{"eq_L_3:Gain", "eq_R_3:Gain"}) {
		t.Errorf("nomes: %v", m[0].Nomes())
	}
	// Só o pré.
	m = Mudancas(antes, ajuste(-6, deTeste.Ganhos...))
	if len(m) != 1 || m[0].Props() != `{ params = [ "pre_L:Mult" 0.501187 "pre_R:Mult" 0.501187 ] }` {
		t.Errorf("pré: %v", m)
	}
	// Nada mudou: nenhuma chamada.
	if m := Mudancas(antes, antes); len(m) != 0 {
		t.Errorf("sem mudança: %v", m)
	}
	// Tudo: o pré primeiro, depois as bandas em ordem.
	m = Mudancas(Plano(), ajuste(-1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1))
	if len(m) != 11 || m[0].Nomes()[0] != "pre_L:Mult" || m[1].Nomes()[0] != "eq_L_0:Gain" || m[10].Nomes()[1] != "eq_R_9:Gain" {
		t.Errorf("tudo: %v", m)
	}
}

func TestUnidade(t *testing.T) {
	u := Unidade("pipewire", "/home/ana/.config/mk/mksom/eq.conf")
	for _, trecho := range []string{
		"[Unit]\n", "[Service]\n", "[Install]\n",
		"After=pipewire.service wireplumber.service\n", "BindsTo=pipewire.service\n",
		"ExecStart=pipewire -c \"/home/ana/.config/mk/mksom/eq.conf\"\n",
		"Restart=on-failure\n", "WantedBy=default.target\n",
	} {
		if !strings.Contains(u, trecho) {
			t.Errorf("falta na unidade: %q\n%s", trecho, u)
		}
	}
	// Um caminho com espaço, aspas, % e $ chega inteiro ao pipewire.
	u = Unidade("/opt/pw/bin/pipewire", `/home/ana maria/50% "meu" $HOME\x/eq.conf`)
	if !strings.Contains(u, `ExecStart=/opt/pw/bin/pipewire -c "/home/ana maria/50%% \"meu\" $$HOME\\x/eq.conf"`+"\n") {
		t.Errorf("escape:\n%s", u)
	}
	if NomeDaUnidade != "mksom-eq.service" || NoDoSink != "mksom.eq" {
		t.Error("nomes do contrato")
	}
}
