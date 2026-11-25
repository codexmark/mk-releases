package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"mksom/internal/base"
	"mksom/internal/eq"
)

// --- o mundo de mentira ---------------------------------------------------------

// casa é o mundo de um teste: as pastas do app, a pasta dos dublês e o
// arquivo em que cada dublê anota a sua chamada.
type casa struct {
	t        *testing.T
	raiz     string
	config   string
	unidades string
	run      string
	bin      string
}

// todos os comandos externos do motor, pela variável que os troca.
var externos = map[string]string{
	"systemctl": "MKSOM_SYSTEMCTL", "pw-dump": "MKSOM_PW_DUMP", "pw-cli": "MKSOM_PW_CLI", "wpctl": "MKSOM_WPCTL",
	"pipewire": "MKSOM_PIPEWIRE", "pw-record": "MKSOM_PW_RECORD", "pw-play": "MKSOM_PW_PLAY",
}

// mundo monta uma casa de mentira: HOME e as pastas do app em pastas
// temporárias, um PATH vazio (nenhum comando de verdade existe) e cada
// comando externo apontando para um dublê que ainda não existe. O PipeWire
// de mentira começa com os alto-falantes como saída padrão e um microfone.
func mundo(t *testing.T) *casa {
	t.Helper()
	raiz := t.TempDir()
	c := &casa{t: t, raiz: raiz, config: filepath.Join(raiz, "config", "mk", "mksom"), unidades: filepath.Join(raiz, "config", "systemd", "user"),
		run: filepath.Join(raiz, "run"), bin: filepath.Join(raiz, "bin")}
	for _, p := range []string{c.bin, c.run} {
		if err := os.MkdirAll(p, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("HOME", raiz)
	t.Setenv("PATH", c.bin)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(raiz, "config"))
	t.Setenv("XDG_RUNTIME_DIR", c.run)
	t.Setenv("MKSOM_CONFIG", c.config)
	t.Setenv("MKSOM_UNIDADES", c.unidades)
	for nome, variavel := range externos {
		t.Setenv(variavel, filepath.Join(c.bin, nome))
	}
	c.dump("dump.json", dumpDe(false, "alsa_output.exemplo.analog-stereo"))
	c.falso("pw-dump", imprime(filepath.Join(raiz, "dump.json")))
	c.falso("systemctl", "")
	c.falso("wpctl", "")
	c.falso("pw-cli", "")
	return c
}

// falso põe um comando de mentira (um script de /bin/sh que só usa o que o
// shell tem dentro) na pasta dos dublês. Cada chamada vira uma linha no
// registro comum, na ordem em que aconteceu.
func (c *casa) falso(nome, corpo string) {
	c.t.Helper()
	script := "#!/bin/sh\nsep=$(printf '\\037'); chamada=" + nome + "; for a in \"$@\"; do chamada=\"$chamada$sep$a\"; done\n" +
		"printf '%s\\n' \"$chamada\" >>'" + filepath.Join(c.raiz, "chamadas") + "'\n" + corpo + "\n"
	if err := os.WriteFile(filepath.Join(c.bin, nome), []byte(script), 0o755); err != nil {
		c.t.Fatal(err)
	}
}

// some tira um dublê: o comando "não está instalado".
func (c *casa) some(nome string) {
	_ = os.Remove(filepath.Join(c.bin, nome))
}

// chamadas devolve o que os dublês anotaram, na ordem: "nome arg arg…".
func (c *casa) chamadas() []string {
	c.t.Helper()
	b, err := os.ReadFile(filepath.Join(c.raiz, "chamadas"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		c.t.Fatal(err)
	}
	var out []string
	for _, linha := range strings.Split(strings.TrimRight(string(b), "\n"), "\n") {
		out = append(out, strings.ReplaceAll(linha, "\037", " "))
	}
	return out
}

// semDump tira do registro as consultas (pw-dump): sobram as ações.
func semDump(chamadas []string) []string {
	out := []string{}
	for _, c := range chamadas {
		if c != "pw-dump" {
			out = append(out, c)
		}
	}
	return out
}

func (c *casa) esquece() {
	_ = os.Remove(filepath.Join(c.raiz, "chamadas"))
}

func (c *casa) dump(nome, texto string) {
	c.t.Helper()
	if err := os.WriteFile(filepath.Join(c.raiz, nome), []byte(texto), 0o600); err != nil {
		c.t.Fatal(err)
	}
}

// imprime é o corpo de um dublê que escreve um arquivo no stdout (um `cat`
// só com o que o shell tem dentro).
func imprime(arquivo string) string {
	return "while IFS= read -r l || [ -n \"$l\" ]; do printf '%s\\n' \"$l\"; done <'" + arquivo + "'"
}

// copia é o trecho de shell que troca o dump de agora por outro.
func (c *casa) copia(de string) string {
	return "{ " + imprime(filepath.Join(c.raiz, de)) + "; } >'" + filepath.Join(c.raiz, "dump.json") + "'"
}

// dumpDe monta um pw-dump de mentira: os alto-falantes (51), um fone USB
// (58), o microfone (52, serial 311), um microfone USB (53, sem serial), o
// monitor (54) e, se pedido, o sink do equalizador (71).
func dumpDe(comEq bool, saidaPadrao string, semNos ...int) string {
	fora := map[int]bool{}
	for _, id := range semNos {
		fora[id] = true
	}
	nos := []struct {
		id    int
		texto string
	}{
		{51, `{"id": 51, "type": "PipeWire:Interface:Node", "info": {"props": {"node.name": "alsa_output.exemplo.analog-stereo", "node.description": "Alto-falantes", "media.class": "Audio/Sink", "object.serial": 310}, "params": {"Props": [{"mute": false, "channelVolumes": [0.064, 0.064]}]}}}`},
		{58, `{"id": 58, "type": "PipeWire:Interface:Node", "info": {"props": {"node.name": "alsa_output.usb-exemplo.analog-stereo", "node.description": "Fone USB", "media.class": "Audio/Sink"}}}`},
		{52, `{"id": 52, "type": "PipeWire:Interface:Node", "info": {"props": {"node.name": "alsa_input.exemplo.analog-stereo", "node.description": "Microfone interno", "media.class": "Audio/Source", "object.serial": 311}, "params": {"Props": [{"mute": false, "channelVolumes": [0.405224, 0.405224]}]}}}`},
		{53, `{"id": 53, "type": "PipeWire:Interface:Node", "info": {"props": {"node.name": "alsa_input.usb-exemplo.mono", "node.description": "Microfone USB", "media.class": "Audio/Source"}, "params": {"Props": [{"mute": true, "channelVolumes": [1.0]}]}}}`},
		{54, `{"id": 54, "type": "PipeWire:Interface:Node", "info": {"props": {"node.name": "alsa_output.exemplo.analog-stereo.monitor", "media.class": "Audio/Source"}}}`},
	}
	var itens []string
	for _, n := range nos {
		if !fora[n.id] {
			itens = append(itens, n.texto)
		}
	}
	if comEq {
		itens = append(itens, `{"id": 71, "type": "PipeWire:Interface:Node", "info": {"props": {"node.name": "mksom.eq", "node.description": "Mk Equalizador", "media.class": "Audio/Sink"}}}`,
			`{"id": 72, "type": "PipeWire:Interface:Node", "info": {"props": {"node.name": "mksom.eq.saida", "media.class": "Stream/Output/Audio"}}}`)
	}
	meta := `{"id": 36, "type": "PipeWire:Interface:Metadata", "props": {"metadata.name": "default"}, "metadata": [`
	if saidaPadrao != "" {
		meta += `{"subject": 0, "key": "default.audio.sink", "type": "Spa:String:JSON", "value": {"name": "` + saidaPadrao + `"}}, `
	}
	meta += `{"subject": 0, "key": "default.audio.source", "type": "Spa:String:JSON", "value": {"name": "alsa_input.exemplo.analog-stereo"}}]}`
	return "[\n" + strings.Join(append(itens, meta), ",\n") + "\n]\n"
}

const altoFalantes = "alsa_output.exemplo.analog-stereo"

// sobeComOSystemctl faz o systemctl de mentira se comportar como o de
// verdade para o PipeWire: `enable --now` faz o sink aparecer, `disable
// --now` o tira.
func (c *casa) sobeComOSystemctl(padraoDepois string) {
	c.dump("ligado.json", dumpDe(true, padraoDepois))
	c.dump("desligado.json", dumpDe(false, altoFalantes))
	c.falso("systemctl", `case "$*" in
  *"enable --now"*) `+c.copia("ligado.json")+` ;;
  *"disable --now"*) `+c.copia("desligado.json")+` ;;
esac`)
}

// saidaSegura é um buffer que aguenta ser escrito por uma goroutine e lido por outra.
type saidaSegura struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *saidaSegura) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *saidaSegura) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// roda executa um subcomando do motor, com `pedido` na entrada padrão, e
// devolve o código e as linhas JSON que ele escreveu.
func roda(t *testing.T, pedido string, args ...string) (int, []map[string]any) {
	t.Helper()
	var b saidaSegura
	antes, entradaAntes := base.Saida, entrada
	base.Saida, entrada = &b, strings.NewReader(pedido)
	defer func() { base.Saida, entrada = antes, entradaAntes }()
	codigo := rodar(args)
	return codigo, linhas(t, b.String())
}

func linhas(t *testing.T, texto string) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, l := range strings.Split(strings.TrimSpace(texto), "\n") {
		if l == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			t.Fatalf("linha que não é JSON: %q", l)
		}
		out = append(out, m)
	}
	return out
}

// quer roda um subcomando que tem de dar certo e devolve a resposta (a última linha).
func quer(t *testing.T, pedido string, args ...string) map[string]any {
	t.Helper()
	codigo, ls := roda(t, pedido, args...)
	if codigo != 0 || len(ls) == 0 {
		t.Fatalf("%v: código %d, saída %v", args, codigo, ls)
	}
	if _, tem := ls[len(ls)-1]["erro"]; tem {
		t.Fatalf("%v: erro com código 0: %v", args, ls)
	}
	return ls[len(ls)-1]
}

// querErro roda um subcomando que tem de falhar com esse código e devolve a frase.
func querErro(t *testing.T, codigoEsperado int, pedido string, args ...string) string {
	t.Helper()
	codigo, ls := roda(t, pedido, args...)
	if codigo != codigoEsperado || len(ls) == 0 {
		t.Fatalf("%v: código %d (esperado %d), saída %v", args, codigo, codigoEsperado, ls)
	}
	frase, _ := ls[len(ls)-1]["erro"].(string)
	if frase == "" {
		t.Fatalf("%v: sem {\"erro\"}: %v", args, ls)
	}
	return frase
}

func conteudo(t *testing.T, caminho string) string {
	t.Helper()
	b, err := os.ReadFile(caminho)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func estadoGuardado(t *testing.T, c *casa) guardado {
	t.Helper()
	var g guardado
	if err := json.Unmarshal([]byte(conteudo(t, filepath.Join(c.config, "estado.json"))), &g); err != nil {
		t.Fatal(err)
	}
	return g
}

func encurta(t *testing.T, prazo *time.Duration, novo time.Duration) {
	t.Helper()
	antigo := *prazo
	*prazo = novo
	t.Cleanup(func() { *prazo = antigo })
}

func igual(t *testing.T, oQue string, obtido, esperado any) {
	t.Helper()
	if !reflect.DeepEqual(obtido, esperado) {
		t.Errorf("%s:\n obtido:   %#v\n esperado: %#v", oQue, obtido, esperado)
	}
}

func pedidoDe(pre float64, ganhos ...float64) string {
	b, _ := json.Marshal(map[string]any{"ganhos": ganhos, "pre": pre})
	return string(b)
}

// --- uso ------------------------------------------------------------------------

func TestSemSubcomandoOuComUmDesconhecido_UsoEmJSON(t *testing.T) {
	mundo(t)
	for _, args := range [][]string{nil, {"dancar"}} {
		if frase := querErro(t, 1, "", args...); frase != "uso: mksom-motor <eq | estado | mic>" {
			t.Errorf("%v: %s", args, frase)
		}
	}
	for _, args := range [][]string{{"eq"}, {"eq", "voar"}, {"eq", "ligar", "ja"}, {"eq", "predefinicao"}, {"eq", "predefinicao", "aplicar"},
		{"eq", "predefinicao", "renomear", "--", "a"}, {"mic"}, {"mic", "cantar"}, {"mic", "padrao"}, {"mic", "volume", "52"}, {"mic", "listar", "tudo"},
		{"estado", "x"}, {"mic", "padrao", "-9"}, {"eq", "predefinicao", "apagar", "-x"}} {
		if frase := querErro(t, 1, "", args...); !strings.HasPrefix(frase, "uso: mksom-motor ") && !strings.HasPrefix(frase, "não conheço a opção") {
			t.Errorf("%v: %s", args, frase)
		}
	}
}

// --- estado ---------------------------------------------------------------------

func TestEstado_Contrato(t *testing.T) {
	c := mundo(t)
	r := quer(t, "", "estado")
	igual(t, "som", r["som"], map[string]any{"responde": true, "erro": nil})
	e := r["eq"].(map[string]any)
	igual(t, "ligado", e["ligado"], false)
	igual(t, "na_saida", e["na_saida"], false)
	igual(t, "bandas", e["bandas"], []any{31.0, 62.0, 125.0, 250.0, 500.0, 1000.0, 2000.0, 4000.0, 8000.0, 16000.0})
	igual(t, "ganhos", e["ganhos"], []any{0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0})
	igual(t, "pre", e["pre"], 0.0)
	igual(t, "predefinicao", e["predefinicao"], nil)
	igual(t, "saida_anterior", e["saida_anterior"], nil)
	predefinicoes := e["predefinicoes"].([]any)
	if len(predefinicoes) != 6 {
		t.Fatalf("predefinições: %v", predefinicoes)
	}
	igual(t, "a primeira", predefinicoes[0], map[string]any{"nome": "plano", "fabrica": true, "pre": 0.0, "ganhos": e["ganhos"]})
	igual(t, "saida_padrao", r["saida_padrao"], map[string]any{"id": 51.0, "nome": altoFalantes, "descricao": "Alto-falantes"})
	igual(t, "entradas", r["entradas"], []any{
		map[string]any{"id": 52.0, "nome": "alsa_input.exemplo.analog-stereo", "descricao": "Microfone interno", "padrao": true, "volume": 74.0, "mudo": false},
		map[string]any{"id": 53.0, "nome": "alsa_input.usb-exemplo.mono", "descricao": "Microfone USB", "padrao": false, "volume": 100.0, "mudo": true},
	})
	// Só consulta: nenhuma ação, nenhum arquivo.
	igual(t, "chamadas", c.chamadas(), []string{"pw-dump"})
	if existe(c.config) || existe(c.unidades) {
		t.Error("o estado não escreve nada")
	}

	// Com o equalizador no ar e um estado guardado.
	c.dump("dump.json", dumpDe(true, "mksom.eq"))
	if err := gravaEstado(guardado{Ligado: true, Predefinicao: "graves", Ganhos: eq.Fabrica[1].Ganhos, Pre: eq.Fabrica[1].Pre, SaidaAnterior: altoFalantes}); err != nil {
		t.Fatal(err)
	}
	r = quer(t, "", "estado")
	e = r["eq"].(map[string]any)
	if e["ligado"] != true || e["na_saida"] != true || e["predefinicao"] != "graves" || e["pre"] != eq.Fabrica[1].Pre {
		t.Errorf("eq: %v", e)
	}
	igual(t, "saida_anterior", e["saida_anterior"], map[string]any{"nome": altoFalantes, "descricao": "Alto-falantes", "existe": true})
	igual(t, "saida_padrao", r["saida_padrao"], map[string]any{"id": 71.0, "nome": "mksom.eq", "descricao": "Mk Equalizador"})
}

func TestEstado_LigadoEODoPipewireNaoODoArquivo(t *testing.T) {
	c := mundo(t)
	// O arquivo diz ligado, mas o sink não existe (a unidade caiu).
	if err := gravaEstado(guardado{Ligado: true, Ganhos: eq.Plano().Ganhos, SaidaAnterior: "alsa_output.sumiu"}); err != nil {
		t.Fatal(err)
	}
	e := quer(t, "", "estado")["eq"].(map[string]any)
	if e["ligado"] != false || e["na_saida"] != false {
		t.Errorf("%v", e)
	}
	igual(t, "anterior que sumiu", e["saida_anterior"], map[string]any{"nome": "alsa_output.sumiu", "descricao": nil, "existe": false})
	// O sink existe mas a saída padrão é outra.
	c.dump("dump.json", dumpDe(true, altoFalantes))
	e = quer(t, "", "estado")["eq"].(map[string]any)
	if e["ligado"] != true || e["na_saida"] != false {
		t.Errorf("%v", e)
	}
}

func TestEstado_SemPipewireAindaAbre(t *testing.T) {
	c := mundo(t)
	for nome, prepara := range map[string]func(){
		"pw-dump que falha":       func() { c.falso("pw-dump", "echo 'can not connect' >&2; exit 1") },
		"pw-dump que diz bobagem": func() { c.falso("pw-dump", "echo 'Error: host is down'") },
		"sem pw-dump":             func() { c.some("pw-dump") },
	} {
		prepara()
		r := quer(t, "", "estado")
		som := r["som"].(map[string]any)
		frase, _ := som["erro"].(string)
		if som["responde"] != false || !strings.HasPrefix(frase, "o PipeWire não respondeu (") {
			t.Errorf("%s: %v", nome, som)
		}
		e := r["eq"].(map[string]any)
		if e["ligado"] != false || len(e["predefinicoes"].([]any)) != 6 || r["saida_padrao"] != nil {
			t.Errorf("%s: %v", nome, r)
		}
		igual(t, nome+": entradas", r["entradas"], []any{})
	}
}

func TestEstado_ArquivosTortosValemPlano(t *testing.T) {
	c := mundo(t)
	if err := os.MkdirAll(c.config, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, texto := range []string{"", "{", `{"ganhos": [1, 2]}`, `{"ganhos": [99, 0, 0, 0, 0, 0, 0, 0, 0, 0], "pre": 0}`, `{"ganhos": "x", "pre": []}`, `[1]`} {
		if err := os.WriteFile(filepath.Join(c.config, "estado.json"), []byte(texto), 0o600); err != nil {
			t.Fatal(err)
		}
		e := quer(t, "", "estado")["eq"].(map[string]any)
		igual(t, texto, e["ganhos"], []any{0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0})
	}
	// Predefinições: só as boas entram.
	tortas := `[{"nome": "boa", "ganhos": [1, 0, 0, 0, 0, 0, 0, 0, 0, 0], "pre": -1},
	  {"nome": "a/b", "ganhos": [0, 0, 0, 0, 0, 0, 0, 0, 0, 0], "pre": 0},
	  {"nome": "plano", "ganhos": [0, 0, 0, 0, 0, 0, 0, 0, 0, 0], "pre": 0},
	  {"nome": "BOA", "ganhos": [0, 0, 0, 0, 0, 0, 0, 0, 0, 0], "pre": 0},
	  {"nome": "alta", "ganhos": [40, 0, 0, 0, 0, 0, 0, 0, 0, 0], "pre": 0},
	  {"nome": "curta", "ganhos": [0], "pre": 0}, 7, "x", {"nome": 3}]`
	if err := os.WriteFile(filepath.Join(c.config, "predefinicoes.json"), []byte(tortas), 0o600); err != nil {
		t.Fatal(err)
	}
	ps := quer(t, "", "estado")["eq"].(map[string]any)["predefinicoes"].([]any)
	if len(ps) != 7 || ps[6].(map[string]any)["nome"] != "boa" || ps[6].(map[string]any)["fabrica"] != false {
		t.Errorf("%v", ps)
	}
}

// --- eq ligar / desligar --------------------------------------------------------

func TestEqLigar_Contrato(t *testing.T) {
	c := mundo(t)
	c.sobeComOSystemctl("mksom.eq")
	if err := gravaEstado(guardado{Ganhos: []float64{3, 2.5, 0, -1, 0, 0.5, 0, -12, 12, 4}, Pre: -4.5}); err != nil {
		t.Fatal(err)
	}
	r := quer(t, "", "eq", "ligar")
	igual(t, "resposta", r, map[string]any{"ok": true, "ligado": true, "na_saida": true, "aviso": nil,
		"saida_anterior": map[string]any{"nome": altoFalantes, "descricao": "Alto-falantes", "existe": true}})

	// A ordem: consulta, recarrega, sobe, espera o sink, torna padrão e confere que virou.
	igual(t, "chamadas", c.chamadas(), []string{
		"pw-dump",
		"systemctl --user daemon-reload",
		"systemctl --user enable --now mksom-eq.service",
		"pw-dump",
		"wpctl set-default 71",
		"pw-dump",
	})
	// O conf e a unidade foram escritos antes de qualquer systemctl, com os
	// valores guardados e a saída de verdade como alvo.
	conf := conteudo(t, filepath.Join(c.config, "eq.conf"))
	if conf != eq.Conf(eq.Ajuste{Ganhos: []float64{3, 2.5, 0, -1, 0, 0.5, 0, -12, 12, 4}, Pre: -4.5}, altoFalantes) {
		t.Errorf("conf:\n%s", conf)
	}
	if !strings.Contains(conf, `target.object = "`+altoFalantes+`"`) || !strings.Contains(conf, `"Gain" = -12.0`) {
		t.Errorf("conf sem o alvo ou sem os ganhos")
	}
	unidade := conteudo(t, filepath.Join(c.unidades, "mksom-eq.service"))
	if !strings.Contains(unidade, "ExecStart="+filepath.Join(c.bin, "pipewire")+" -c \""+filepath.Join(c.config, "eq.conf")+"\"\n") {
		t.Errorf("unidade:\n%s", unidade)
	}
	for _, f := range []string{filepath.Join(c.config, "eq.conf"), filepath.Join(c.config, "estado.json"), filepath.Join(c.unidades, "mksom-eq.service")} {
		if i, _ := os.Stat(f); i.Mode().Perm() != 0o600 {
			t.Errorf("%s: modo %o", f, i.Mode().Perm())
		}
	}
	st := estadoGuardado(t, c)
	if !st.Ligado || st.SaidaAnterior != altoFalantes || st.Pre != -4.5 {
		t.Errorf("estado: %+v", st)
	}
	// O motor não roda o pipewire: quem roda é o systemd.
	for _, ch := range c.chamadas() {
		if strings.HasPrefix(ch, "pipewire") {
			t.Errorf("o motor rodou o pipewire: %s", ch)
		}
	}
}

func TestEqLigar_EsperaOSinkAparecer(t *testing.T) {
	c := mundo(t)
	encurta(t, &passoDaEspera, 5*time.Millisecond)
	// O sink só aparece na quarta consulta depois de subir.
	c.dump("ligado.json", dumpDe(true, "mksom.eq"))
	contador := filepath.Join(c.raiz, "consultas")
	c.falso("pw-dump", `n=0; [ -f '`+contador+`' ] && read -r n <'`+contador+`'; n=$((n + 1)); printf '%s\n' "$n" >'`+contador+`'
if [ "$n" -ge 5 ]; then `+imprime(filepath.Join(c.raiz, "ligado.json"))+`; else `+imprime(filepath.Join(c.raiz, "dump.json"))+`; fi`)
	quer(t, "", "eq", "ligar")
	ch := c.chamadas()
	if len(ch) != 9 || ch[len(ch)-2] != "wpctl set-default 71" || ch[len(ch)-3] != "pw-dump" {
		t.Errorf("chamadas: %v", ch)
	}
}

func TestEqLigar_DesfazQuandoOSinkNaoAparece(t *testing.T) {
	c := mundo(t)
	encurta(t, &esperaDoSink, 60*time.Millisecond)
	encurta(t, &passoDaEspera, 10*time.Millisecond)
	frase := querErro(t, 1, "", "eq", "ligar")
	if frase != "o equalizador não ligou: o equalizador não apareceu no PipeWire" {
		t.Errorf("frase: %s", frase)
	}
	acoes := semDump(c.chamadas())
	igual(t, "ações", acoes, []string{
		"systemctl --user daemon-reload",
		"systemctl --user enable --now mksom-eq.service",
		"systemctl --user disable --now mksom-eq.service",
	})
	// A saída padrão não foi tocada, e o estado diz desligado.
	if st := estadoGuardado(t, c); st.Ligado {
		t.Errorf("estado: %+v", st)
	}
}

func TestEqLigar_DesfazQuandoOEnableOuOSetDefaultFalham(t *testing.T) {
	c := mundo(t)
	c.falso("systemctl", `case "$*" in *"enable --now"*) echo "Failed to enable unit: nope" >&2; exit 1 ;; esac`)
	frase := querErro(t, 1, "", "eq", "ligar")
	if frase != "o equalizador não ligou: o systemctl falhou: Failed to enable unit: nope" {
		t.Errorf("frase: %s", frase)
	}
	igual(t, "enable que falha", semDump(c.chamadas()), []string{
		"systemctl --user daemon-reload", "systemctl --user enable --now mksom-eq.service", "systemctl --user disable --now mksom-eq.service"})

	c.esquece()
	c.sobeComOSystemctl("mksom.eq")
	c.falso("wpctl", "echo 'Object not found' >&2; exit 2")
	frase = querErro(t, 1, "", "eq", "ligar")
	if !strings.Contains(frase, "não virou a saída padrão") || !strings.Contains(frase, "Object not found") {
		t.Errorf("frase: %s", frase)
	}
	igual(t, "set-default que falha", semDump(c.chamadas()), []string{
		"systemctl --user daemon-reload", "systemctl --user enable --now mksom-eq.service", "wpctl set-default 71",
		"systemctl --user disable --now mksom-eq.service"})
	if st := estadoGuardado(t, c); st.Ligado {
		t.Errorf("estado: %+v", st)
	}

	// O daemon-reload que falha: nada subiu, nada a desfazer.
	c.esquece()
	c.falso("systemctl", "echo 'Failed to connect to bus' >&2; exit 1")
	querErro(t, 1, "", "eq", "ligar")
	igual(t, "daemon-reload que falha", semDump(c.chamadas()), []string{"systemctl --user daemon-reload"})
}

func TestEqLigar_SemPipewireNaoEscreveNadaNemChamaNinguem(t *testing.T) {
	c := mundo(t)
	c.falso("pw-dump", "echo 'pw-dump: can not connect: Host is down' >&2; exit 1")
	frase := querErro(t, 1, "", "eq", "ligar")
	if frase != "o PipeWire não respondeu (o pw-dump falhou: pw-dump: can not connect: Host is down)" {
		t.Errorf("frase: %s", frase)
	}
	igual(t, "chamadas", c.chamadas(), []string{"pw-dump"})
	if existe(c.config) || existe(c.unidades) {
		t.Error("escreveu arquivo sem o PipeWire responder")
	}
	// Um pw-dump que trava é morto no prazo.
	encurta(t, &prazoDoComando, 80*time.Millisecond)
	c.falso("pw-dump", "while :; do :; done")
	inicio := time.Now()
	frase = querErro(t, 1, "", "eq", "ligar")
	if frase != "o PipeWire não respondeu (o pw-dump não respondeu a tempo)" || time.Since(inicio) > 3*time.Second {
		t.Errorf("frase: %s (%v)", frase, time.Since(inicio))
	}
}

func TestEqLigar_JaNoArSoTornaPadraoENaoLembraOProprioEqualizador(t *testing.T) {
	c := mundo(t)
	// O sink está no ar (a unidade subiu no login) e a saída padrão é o fone.
	c.dump("dump.json", dumpDe(true, "alsa_output.usb-exemplo.analog-stereo"))
	// (o wpctl de mentira faz o que o de verdade faria: a padrão vira o equalizador)
	c.dump("padrao-eq.json", dumpDe(true, "mksom.eq"))
	c.falso("wpctl", c.copia("padrao-eq.json"))
	quer(t, "", "eq", "ligar")
	igual(t, "chamadas", c.chamadas(), []string{"pw-dump", "wpctl set-default 71", "pw-dump"})
	if st := estadoGuardado(t, c); !st.Ligado || st.SaidaAnterior != "alsa_output.usb-exemplo.analog-stereo" {
		t.Errorf("estado: %+v", st)
	}
	if existe(filepath.Join(c.unidades, "mksom-eq.service")) {
		t.Error("com o sink no ar a unidade não é reescrita")
	}
	// Ligar de novo com o equalizador já como padrão: a anterior continua a mesma.
	c.esquece()
	c.dump("dump.json", dumpDe(true, "mksom.eq"))
	r := quer(t, "", "eq", "ligar")
	if st := estadoGuardado(t, c); st.SaidaAnterior != "alsa_output.usb-exemplo.analog-stereo" {
		t.Errorf("lembrou o próprio equalizador: %+v", st)
	}
	igual(t, "anterior", r["saida_anterior"], map[string]any{"nome": "alsa_output.usb-exemplo.analog-stereo", "descricao": "Fone USB", "existe": true})
}

func TestEqLigar_SemSaidaPadraoOConfNaoTemAlvo(t *testing.T) {
	c := mundo(t)
	c.dump("dump.json", dumpDe(false, ""))
	encurta(t, &esperaDoPadrao, 30*time.Millisecond)
	encurta(t, &passoDaEspera, 10*time.Millisecond)
	c.dump("ligado.json", dumpDe(true, ""))
	c.falso("systemctl", `case "$*" in *"enable --now"*) `+c.copia("ligado.json")+` ;; esac`)
	r := quer(t, "", "eq", "ligar")
	igual(t, "anterior", r["saida_anterior"], nil)
	if strings.Contains(conteudo(t, filepath.Join(c.config, "eq.conf")), "target.object") {
		t.Error("sem saída conhecida não há target.object")
	}
}

func TestEqDesligar_Contrato(t *testing.T) {
	c := mundo(t)
	c.sobeComOSystemctl("mksom.eq")
	quer(t, "", "eq", "ligar")
	c.esquece()

	r := quer(t, "", "eq", "desligar")
	igual(t, "resposta", r, map[string]any{"ok": true, "ligado": false, "na_saida": false, "aviso": nil,
		"saida_anterior": map[string]any{"nome": altoFalantes, "descricao": "Alto-falantes", "existe": true}})
	// Primeiro devolve a saída de antes, depois derruba a unidade.
	igual(t, "chamadas", c.chamadas(), []string{"pw-dump", "wpctl set-default 51", "systemctl --user disable --now mksom-eq.service"})
	if st := estadoGuardado(t, c); st.Ligado || st.SaidaAnterior != altoFalantes {
		t.Errorf("estado: %+v", st)
	}
}

func TestEqDesligar_AnteriorSumiuNaoInventa(t *testing.T) {
	c := mundo(t)
	if err := gravaEstado(guardado{Ligado: true, Ganhos: eq.Plano().Ganhos, SaidaAnterior: "alsa_output.usb-exemplo.analog-stereo"}); err != nil {
		t.Fatal(err)
	}
	// O fone USB (58) foi tirado: não há para quem devolver.
	c.dump("dump.json", dumpDe(true, "mksom.eq", 58))
	r := quer(t, "", "eq", "desligar")
	igual(t, "chamadas", c.chamadas(), []string{"pw-dump", "systemctl --user disable --now mksom-eq.service"})
	igual(t, "anterior", r["saida_anterior"], map[string]any{"nome": "alsa_output.usb-exemplo.analog-stereo", "descricao": nil, "existe": false})
}

func TestEqDesligar_RespeitaQuemTrocouASaidaEDesligaMesmoSemPipewire(t *testing.T) {
	c := mundo(t)
	if err := gravaEstado(guardado{Ligado: true, Ganhos: eq.Plano().Ganhos, SaidaAnterior: altoFalantes}); err != nil {
		t.Fatal(err)
	}
	// A pessoa já escolheu o fone como saída: o motor não troca de volta.
	c.dump("dump.json", dumpDe(true, "alsa_output.usb-exemplo.analog-stereo"))
	quer(t, "", "eq", "desligar")
	igual(t, "chamadas", c.chamadas(), []string{"pw-dump", "systemctl --user disable --now mksom-eq.service"})

	// Sem PipeWire: a unidade é desligada do mesmo jeito, e o aviso conta.
	c.esquece()
	c.falso("pw-dump", "exit 1")
	r := quer(t, "", "eq", "desligar")
	if aviso, _ := r["aviso"].(string); !strings.HasPrefix(aviso, "o PipeWire não respondeu") {
		t.Errorf("aviso: %v", r)
	}
	igual(t, "chamadas", c.chamadas(), []string{"pw-dump", "systemctl --user disable --now mksom-eq.service"})

	// O systemctl que falha é erro.
	c.falso("systemctl", "echo 'Failed to disable' >&2; exit 1")
	if frase := querErro(t, 1, "", "eq", "desligar"); !strings.HasPrefix(frase, "o equalizador não desligou: ") {
		t.Errorf("frase: %s", frase)
	}
}

// --- eq ajustar -----------------------------------------------------------------

func TestEqAjustar_Contrato(t *testing.T) {
	c := mundo(t)
	c.sobeComOSystemctl("mksom.eq")
	quer(t, "", "eq", "ligar")
	c.esquece()

	// Do plano para: banda 3 em +4, banda 9 em −2,5 e pré em −6.
	r := quer(t, pedidoDe(-6, 0, 0, 0, 4, 0, 0, 0, 0, 0, -2.5), "eq", "ajustar")
	igual(t, "resposta", r, map[string]any{"ok": true, "ao_vivo": true, "predefinicao": nil,
		"aplicados": []any{"pre_L:Mult", "pre_R:Mult", "eq_L_3:Gain", "eq_R_3:Gain", "eq_L_9:Gain", "eq_R_9:Gain"}})
	// Uma chamada por banda que mudou (os dois canais juntos), no nó do sink; nada reinicia.
	igual(t, "chamadas", c.chamadas(), []string{
		"pw-dump",
		`pw-cli set-param 71 Props { params = [ "pre_L:Mult" 0.501187 "pre_R:Mult" 0.501187 ] }`,
		`pw-cli set-param 71 Props { params = [ "eq_L_3:Gain" 4.0 "eq_R_3:Gain" 4.0 ] }`,
		`pw-cli set-param 71 Props { params = [ "eq_L_9:Gain" -2.5 "eq_R_9:Gain" -2.5 ] }`,
	})
	st := estadoGuardado(t, c)
	igual(t, "ganhos guardados", st.Ganhos, []float64{0, 0, 0, 4, 0, 0, 0, 0, 0, -2.5})
	if st.Pre != -6 || !st.Ligado || st.Predefinicao != "" {
		t.Errorf("estado: %+v", st)
	}
	// O conf foi regravado para a próxima subida, com o mesmo alvo.
	igual(t, "conf", conteudo(t, filepath.Join(c.config, "eq.conf")), eq.Conf(st.ajuste(), altoFalantes))

	// Só uma banda mudou: só ela é aplicada.
	c.esquece()
	r = quer(t, pedidoDe(-6, 0, 0, 0, 4.5, 0, 0, 0, 0, 0, -2.5), "eq", "ajustar")
	igual(t, "aplicados", r["aplicados"], []any{"eq_L_3:Gain", "eq_R_3:Gain"})
	igual(t, "chamadas", c.chamadas(), []string{"pw-dump", `pw-cli set-param 71 Props { params = [ "eq_L_3:Gain" 4.5 "eq_R_3:Gain" 4.5 ] }`})

	// Nada mudou: nenhuma chamada ao pw-cli.
	c.esquece()
	r = quer(t, pedidoDe(-6, 0, 0, 0, 4.5, 0, 0, 0, 0, 0, -2.5), "eq", "ajustar")
	igual(t, "aplicados", r["aplicados"], []any{})
	igual(t, "chamadas", c.chamadas(), []string{"pw-dump"})

	// Voltar aos valores de uma predefinição diz o nome dela.
	r = quer(t, pedidoDe(0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0), "eq", "ajustar")
	igual(t, "predefinição", r["predefinicao"], "plano")
	for _, ch := range c.chamadas() {
		if strings.HasPrefix(ch, "systemctl") || strings.HasPrefix(ch, "wpctl") {
			t.Errorf("ajustar não reinicia nem troca saída: %s", ch)
		}
	}
}

func TestEqAjustar_DesligadoSoGrava(t *testing.T) {
	c := mundo(t)
	r := quer(t, pedidoDe(-3, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0), "eq", "ajustar")
	igual(t, "resposta", r, map[string]any{"ok": true, "ao_vivo": false, "predefinicao": nil, "aplicados": []any{}})
	igual(t, "chamadas", c.chamadas(), []string{"pw-dump"})
	if st := estadoGuardado(t, c); st.Ganhos[0] != 3 || st.Pre != -3 || st.Ligado {
		t.Errorf("estado: %+v", st)
	}
	// Nunca ligado: ainda não há conf para regravar.
	if existe(filepath.Join(c.config, "eq.conf")) {
		t.Error("escreveu o conf sem nunca ter ligado")
	}
	// Sem PipeWire também grava.
	c.falso("pw-dump", "exit 1")
	r = quer(t, pedidoDe(0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0), "eq", "ajustar")
	if r["ao_vivo"] != false || estadoGuardado(t, c).Ganhos[0] != 1 {
		t.Errorf("%v", r)
	}
}

func TestEqAjustar_Validacoes(t *testing.T) {
	c := mundo(t)
	c.dump("dump.json", dumpDe(true, "mksom.eq"))
	for _, pedido := range []string{
		"", "x", "[]", `{"ganhos": [0, 0, 0, 0, 0, 0, 0, 0, 0, 0]}`, `{"pre": 0}`,
		pedidoDe(0, 0, 0, 0), pedidoDe(0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0),
		pedidoDe(0, 12.5, 0, 0, 0, 0, 0, 0, 0, 0, 0), pedidoDe(0, -13, 0, 0, 0, 0, 0, 0, 0, 0, 0),
		pedidoDe(0, 0.25, 0, 0, 0, 0, 0, 0, 0, 0, 0), pedidoDe(0.5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0),
		pedidoDe(-12.5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0), pedidoDe(-0.1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0),
		`{"ganhos": ["3; rm -rf", 0, 0, 0, 0, 0, 0, 0, 0, 0], "pre": 0}`,
		`{"ganhos": [1e400, 0, 0, 0, 0, 0, 0, 0, 0, 0], "pre": 0}`,
		`{"ganhos": [null, 0, 0, 0, 0, 0, 0, 0, 0, 0], "pre": "0"}`,
	} {
		querErro(t, 1, pedido, "eq", "ajustar")
	}
	// Nada foi aplicado nem gravado.
	for _, ch := range c.chamadas() {
		if ch != "pw-dump" {
			t.Errorf("pedido inválido chegou a: %s", ch)
		}
	}
	if existe(filepath.Join(c.config, "estado.json")) {
		t.Error("pedido inválido gravou estado")
	}
}

func TestEqAjustar_PwCliQueFalhaNaoGravaOEstado(t *testing.T) {
	c := mundo(t)
	c.dump("dump.json", dumpDe(true, "mksom.eq"))
	for nome, corpo := range map[string]string{
		"sai com erro":       "echo 'Error: \"Unknown object\"' >&2; exit 1",
		"diz Error e sai 0":  `echo 'Error: "id 71 unknown"'`,
		"não está instalado": "",
	} {
		if corpo == "" {
			c.some("pw-cli")
		} else {
			c.falso("pw-cli", corpo)
		}
		querErro(t, 1, pedidoDe(0, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0), "eq", "ajustar")
		if existe(filepath.Join(c.config, "estado.json")) {
			t.Errorf("%s: gravou o estado sem aplicar", nome)
		}
	}
}

// --- eq predefinicao ------------------------------------------------------------

func nomesDe(t *testing.T, r map[string]any) []string {
	t.Helper()
	var out []string
	for _, p := range r["predefinicoes"].([]any) {
		out = append(out, p.(map[string]any)["nome"].(string))
	}
	return out
}

func TestEqPredefinicao_Contrato(t *testing.T) {
	c := mundo(t)
	c.dump("dump.json", dumpDe(true, "mksom.eq"))

	// Aplicar uma de fábrica: ao vivo, só o que muda em relação ao plano.
	r := quer(t, "", "eq", "predefinicao", "aplicar", "--", "graves")
	graves := eq.Fabrica[1]
	if r["ok"] != true || r["predefinicao"] != "graves" || r["ao_vivo"] != true || r["pre"] != graves.Pre {
		t.Errorf("aplicar: %v", r)
	}
	mudam := 1 // o pré
	for _, g := range graves.Ganhos {
		if g != 0 {
			mudam++
		}
	}
	if n := len(c.chamadas()) - 1; n != mudam {
		t.Errorf("%d chamadas ao pw-cli, esperava %d: %v", n, mudam, c.chamadas())
	}
	if len(r["aplicados"].([]any)) != 2*mudam || len(nomesDe(t, r)) != 6 {
		t.Errorf("aplicar: %v", r)
	}
	if st := estadoGuardado(t, c); st.Predefinicao != "graves" || !reflect.DeepEqual(st.Ganhos, graves.Ganhos) {
		t.Errorf("estado: %+v", st)
	}

	// Mexer e guardar como "minha".
	quer(t, pedidoDe(-2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0), "eq", "ajustar")
	c.esquece()
	r = quer(t, "", "eq", "predefinicao", "guardar", "--", "  Minha  ")
	if r["predefinicao"] != "Minha" || r["ao_vivo"] != false {
		t.Errorf("guardar: %v", r)
	}
	igual(t, "nomes", nomesDe(t, r), []string{"plano", "graves", "agudos", "voz", "noite", "fone pequeno", "Minha"})
	igual(t, "a guardada", r["predefinicoes"].([]any)[6], map[string]any{"nome": "Minha", "fabrica": false, "pre": -2.0,
		"ganhos": []any{2.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0}})
	if len(c.chamadas()) != 0 {
		t.Errorf("guardar não fala com o PipeWire: %v", c.chamadas())
	}
	if i, _ := os.Stat(filepath.Join(c.config, "predefinicoes.json")); i.Mode().Perm() != 0o600 {
		t.Errorf("modo: %o", i.Mode().Perm())
	}

	// Guardar de novo com o mesmo nome (sem olhar maiúsculas) troca, não duplica.
	quer(t, pedidoDe(-3, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0), "eq", "ajustar")
	r = quer(t, "", "eq", "predefinicao", "guardar", "--", "minha")
	if len(nomesDe(t, r)) != 7 || r["predefinicoes"].([]any)[6].(map[string]any)["pre"] != -3.0 {
		t.Errorf("guardar por cima: %v", r)
	}

	// Aplicar a da pessoa depois de outra.
	quer(t, "", "eq", "predefinicao", "aplicar", "--", "plano")
	c.esquece()
	r = quer(t, "", "eq", "predefinicao", "aplicar", "--", "MINHA")
	if r["predefinicao"] != "minha" || r["pre"] != -3.0 {
		t.Errorf("aplicar a minha: %v", r)
	}
	igual(t, "chamadas", c.chamadas(), []string{"pw-dump",
		`pw-cli set-param 71 Props { params = [ "pre_L:Mult" 0.707946 "pre_R:Mult" 0.707946 ] }`,
		`pw-cli set-param 71 Props { params = [ "eq_L_0:Gain" 3.0 "eq_R_0:Gain" 3.0 ] }`})

	// Renomear: a atual acompanha o nome novo.
	r = quer(t, "", "eq", "predefinicao", "renomear", "--", "minha", "Fone do trabalho")
	if r["predefinicao"] != "Fone do trabalho" || nomesDe(t, r)[6] != "Fone do trabalho" {
		t.Errorf("renomear: %v", r)
	}
	// Apagar: some da lista e deixa de ser a atual.
	r = quer(t, "", "eq", "predefinicao", "apagar", "--", "fone do trabalho")
	if r["predefinicao"] != nil || len(nomesDe(t, r)) != 6 {
		t.Errorf("apagar: %v", r)
	}
	if st := estadoGuardado(t, c); st.Predefinicao != "" || st.Ganhos[0] != 3 {
		t.Errorf("apagar não mexe nos valores: %+v", st)
	}
}

func TestEqPredefinicao_AsDeFabricaSaoRecusadas(t *testing.T) {
	c := mundo(t)
	for _, args := range [][]string{
		{"guardar", "--", "plano"}, {"guardar", "--", "GRAVES"}, {"renomear", "--", "voz", "fala"}, {"apagar", "--", "noite"},
	} {
		querErro(t, 2, "", append([]string{"eq", "predefinicao"}, args...)...)
	}
	quer(t, "", "eq", "predefinicao", "guardar", "--", "minha")
	querErro(t, 2, "", "eq", "predefinicao", "renomear", "--", "minha", "Fone Pequeno")
	if existe(filepath.Join(c.config, "predefinicoes.json")) {
		if strings.Contains(conteudo(t, filepath.Join(c.config, "predefinicoes.json")), "equeno") {
			t.Error("gravou um nome de fábrica")
		}
	}
}

func TestEqPredefinicao_NomesInvalidosEQueNaoExistem(t *testing.T) {
	c := mundo(t)
	for _, nome := range []string{"", "   ", "a/b", `a\b`, "a\nb", "a\tb", "a\x1bb", "..", strings.Repeat("x", 41), "\xff"} {
		querErro(t, 1, "", "eq", "predefinicao", "guardar", "--", nome)
	}
	// Com cara de opção, só depois de --; e mesmo assim o nome é recusado.
	querErro(t, 1, "", "eq", "predefinicao", "guardar", "-rf")
	querErro(t, 1, "", "eq", "predefinicao", "guardar", "--", "-rf")
	if existe(filepath.Join(c.config, "predefinicoes.json")) {
		t.Error("um nome inválido foi gravado")
	}
	querErro(t, 1, "", "eq", "predefinicao", "aplicar", "--", "fantasma")
	querErro(t, 1, "", "eq", "predefinicao", "apagar", "--", "fantasma")
	querErro(t, 1, "", "eq", "predefinicao", "renomear", "--", "fantasma", "outra")
	quer(t, "", "eq", "predefinicao", "guardar", "--", "uma")
	quer(t, "", "eq", "predefinicao", "guardar", "--", "duas")
	if frase := querErro(t, 1, "", "eq", "predefinicao", "renomear", "--", "uma", "DUAS"); frase != "já há uma predefinição com esse nome" {
		t.Errorf("frase: %s", frase)
	}
	querErro(t, 1, "", "eq", "predefinicao", "renomear", "--", "uma", "a/b")
	// Trocar só as maiúsculas do próprio nome pode.
	r := quer(t, "", "eq", "predefinicao", "renomear", "--", "uma", "Uma")
	igual(t, "nomes", nomesDe(t, r)[6:], []string{"Uma", "duas"})
	// O limite de predefinições guardadas.
	for i := 0; i < eq.PredefinicoesMax-2; i++ {
		quer(t, "", "eq", "predefinicao", "guardar", "--", "p"+strconv.Itoa(i))
	}
	if frase := querErro(t, 1, "", "eq", "predefinicao", "guardar", "--", "demais"); !strings.Contains(frase, "apague uma") {
		t.Errorf("frase: %s", frase)
	}
}

// --- mic: listar, padrão, volume, mudo -------------------------------------------

func TestMicListar_Contrato(t *testing.T) {
	c := mundo(t)
	r := quer(t, "", "mic", "listar")
	igual(t, "entradas", r, map[string]any{"entradas": []any{
		map[string]any{"id": 52.0, "nome": "alsa_input.exemplo.analog-stereo", "descricao": "Microfone interno", "padrao": true, "volume": 74.0, "mudo": false},
		map[string]any{"id": 53.0, "nome": "alsa_input.usb-exemplo.mono", "descricao": "Microfone USB", "padrao": false, "volume": 100.0, "mudo": true},
	}})
	// Sem entrada nenhuma: lista vazia, não null.
	c.dump("dump.json", dumpDe(false, altoFalantes, 52, 53))
	igual(t, "nenhuma", quer(t, "", "mic", "listar"), map[string]any{"entradas": []any{}})
	c.falso("pw-dump", "exit 1")
	if frase := querErro(t, 1, "", "mic", "listar"); !strings.HasPrefix(frase, "o PipeWire não respondeu") {
		t.Errorf("frase: %s", frase)
	}
}

func TestMicPadraoVolumeMudo_Contrato(t *testing.T) {
	c := mundo(t)
	casos := []struct {
		args    []string
		chamada string
	}{
		{[]string{"padrao", "53"}, "wpctl set-default 53"},
		{[]string{"volume", "52", "0"}, "wpctl set-volume 52 0.00"},
		{[]string{"volume", "52", "7"}, "wpctl set-volume 52 0.07"},
		{[]string{"volume", "52", "74"}, "wpctl set-volume 52 0.74"},
		{[]string{"volume", "52", "100"}, "wpctl set-volume 52 1.00"},
		{[]string{"volume", "52", "150"}, "wpctl set-volume 52 1.50"},
		{[]string{"mudo", "52", "sim"}, "wpctl set-mute 52 1"},
		{[]string{"mudo", "53", "nao"}, "wpctl set-mute 53 0"},
	}
	for _, caso := range casos {
		c.esquece()
		r := quer(t, "", append([]string{"mic"}, caso.args...)...)
		igual(t, caso.chamada, r, map[string]any{"ok": true})
		igual(t, caso.chamada, c.chamadas(), []string{"pw-dump", caso.chamada})
	}
	c.falso("wpctl", "echo 'Could not connect to PipeWire' >&2; exit 1")
	if frase := querErro(t, 1, "", "mic", "mudo", "52", "sim"); frase != "o wpctl falhou: Could not connect to PipeWire" {
		t.Errorf("frase: %s", frase)
	}
}

func TestMic_Validacoes(t *testing.T) {
	c := mundo(t)
	// O que não é número não chega a comando nenhum (código 1).
	for _, args := range [][]string{
		{"padrao", "x"}, {"padrao", "5 2"}, {"padrao", "52; reboot"}, {"padrao", "$(reboot)"}, {"padrao", "5.2"}, {"padrao", ""},
		{"padrao", "12345678"}, {"padrao", "@DEFAULT_SOURCE@"}, {"padrao", "--", "-1"}, {"padrao", "٥٢"},
		{"volume", "52", "151"}, {"volume", "52", "-1"}, {"volume", "52", "x"}, {"volume", "52", "50%"}, {"volume", "52", "1.0"}, {"volume", "52", "1000"},
		{"volume", "x", "50"}, {"mudo", "52", "talvez"}, {"mudo", "52", "1"}, {"mudo", "52", "SIM"}, {"mudo", "x", "sim"},
		{"nivel", "x"}, {"gravar", "5 2"},
	} {
		querErro(t, 1, "", append([]string{"mic"}, args...)...)
	}
	for _, ch := range c.chamadas() {
		if ch != "pw-dump" {
			t.Errorf("argumento inválido chegou a: %s", ch)
		}
	}
	// Um id que é número mas não é de uma entrada: recusado (código 2). 51 é
	// uma saída, 54 o monitor, 71 não existe.
	c.falso("pw-record", "")
	for _, args := range [][]string{{"padrao", "51"}, {"volume", "54", "50"}, {"mudo", "71", "sim"}, {"nivel", "51"}, {"gravar", "999"}} {
		if frase := querErro(t, 2, "", append([]string{"mic"}, args...)...); frase != "essa entrada não existe mais" {
			t.Errorf("%v: %s", args, frase)
		}
	}
	for _, ch := range c.chamadas() {
		if ch != "pw-dump" {
			t.Errorf("id de fora chegou a: %s", ch)
		}
	}
}

// --- mic nivel ------------------------------------------------------------------

// octal escreve bytes como escapes para o printf do shell.
func octal(dados []byte) string {
	var s strings.Builder
	for _, b := range dados {
		fmt.Fprintf(&s, "\\%03o", b)
	}
	return s.String()
}

// bloco devolve um bloco do medidor (1066 amostras s16le) com o que a função diz.
func bloco(amostra func(i int) int16) []byte {
	out := make([]byte, 0, 2*taxaDoMedidor/linhasPorSegundo)
	for i := 0; i < taxaDoMedidor/linhasPorSegundo; i++ {
		a := uint16(amostra(i))
		out = append(out, byte(a), byte(a>>8))
	}
	return out
}

var (
	silencio = bloco(func(int) int16 { return 0 })
	saturado = bloco(func(int) int16 { return 32767 })
	// Senoide com pico a −6 dBFS, 41 ciclos inteiros no bloco, com uma amostra no pico.
	senoide = bloco(func(i int) int16 {
		return int16(math.Round(32768 * math.Pow(10, -6.0/20) * math.Sin(2*math.Pi*(float64(i)+0.5)/26)))
	})
)

func TestMicNivel_Contrato(t *testing.T) {
	c := mundo(t)
	// O pw-record de mentira solta PCM conhecido e termina.
	c.falso("pw-record", "printf '"+octal(silencio)+"'\nprintf '"+octal(senoide)+"'\nprintf '"+octal(saturado)+"'\nprintf '"+octal(silencio[:500])+"'")
	// A entrada padrão fica aberta enquanto o fluxo roda, como a janela faz.
	fio, ponta := io.Pipe()
	defer ponta.Close()
	var b saidaSegura
	antes, entradaAntes := base.Saida, entrada
	base.Saida, entrada = &b, fio
	defer func() { base.Saida, entrada = antes, entradaAntes }()
	if codigo := rodar([]string{"mic", "nivel", "52"}); codigo != 0 {
		t.Fatalf("código %d: %s", codigo, b.String())
	}
	if got, quer := b.String(), "{\"pico\":-90,\"rms\":-90}\n{\"pico\":-6,\"rms\":-9}\n{\"pico\":0,\"rms\":0}\n{\"fim\":true}\n"; got != quer {
		t.Errorf("níveis:\n%s", got)
	}
	// O alvo é o object.serial do nó (311), não o id (52).
	igual(t, "chamadas", c.chamadas(), []string{"pw-dump", "pw-record --raw --target 311 --format s16 --rate 16000 --channels 1 -"})

	// Sem serial no dump (o 53), vale o id.
	c.esquece()
	b = saidaSegura{}
	rodar([]string{"mic", "nivel", "53"})
	igual(t, "sem serial", c.chamadas()[1], "pw-record --raw --target 53 --format s16 --rate 16000 --channels 1 -")
}

func TestMicNivel_PwRecordQueMorreComErroOuNaoExiste(t *testing.T) {
	c := mundo(t)
	c.falso("pw-record", "echo 'error: stream error: no target node' >&2; exit 1")
	fio, ponta := io.Pipe()
	defer ponta.Close()
	var b saidaSegura
	antes, entradaAntes := base.Saida, entrada
	base.Saida, entrada = &b, fio
	defer func() { base.Saida, entrada = antes, entradaAntes }()
	if codigo := rodar([]string{"mic", "nivel", "52"}); codigo != 1 {
		t.Errorf("código %d", codigo)
	}
	if got := b.String(); got != "{\"erro\":\"a entrada parou de responder: error: stream error: no target node\"}\n" {
		t.Errorf("%s", got)
	}
	c.some("pw-record")
	b = saidaSegura{}
	if codigo := rodar([]string{"mic", "nivel", "52"}); codigo != 1 || !strings.Contains(b.String(), "falta o pw-record") {
		t.Errorf("sem pw-record: %s", b.String())
	}
}

// vivo diz se o processo ainda existe.
func vivo(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}

// pidDe lê o pid que um dublê anotou.
func pidDe(t *testing.T, arquivo string) int {
	t.Helper()
	for i := 0; i < 200; i++ {
		if b, err := os.ReadFile(arquivo); err == nil {
			if pid, err := strconv.Atoi(strings.TrimSpace(string(b))); err == nil {
				return pid
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("o dublê não anotou o pid")
	return 0
}

// fluxo roda um subcomando de fluxo numa goroutine, com a entrada padrão num
// pipe que o teste segura. Devolve a saída, a ponta de escrita e o canal por
// onde o código de saída chega.
func fluxo(t *testing.T, args ...string) (*saidaSegura, *io.PipeWriter, chan int) {
	t.Helper()
	fio, ponta := io.Pipe()
	b := &saidaSegura{}
	antes, entradaAntes := base.Saida, entrada
	base.Saida, entrada = b, fio
	acabou := make(chan int, 1)
	go func() { acabou <- rodar(args) }()
	t.Cleanup(func() {
		ponta.Close()
		select {
		case <-acabou:
		case <-time.After(5 * time.Second):
		}
		base.Saida, entrada = antes, entradaAntes
	})
	return b, ponta, acabou
}

func espera(t *testing.T, acabou chan int, prazo time.Duration) int {
	t.Helper()
	select {
	case codigo := <-acabou:
		acabou <- codigo
		return codigo
	case <-time.After(prazo):
		t.Fatal("o motor não terminou no prazo")
		return -1
	}
}

func TestMicNivel_ParaQuandoAEntradaFechaEMataOFilho(t *testing.T) {
	for nome, comoPara := range map[string]func(*io.PipeWriter){
		"a entrada fecha": func(p *io.PipeWriter) { p.Close() },
		"a linha parar":   func(p *io.PipeWriter) { _, _ = p.Write([]byte("parar\n")) },
	} {
		c := mundo(t)
		pid := filepath.Join(c.raiz, "pid")
		// Um pw-record que nunca para: PCM sem fim, e ignora o primeiro aviso… não: sai no TERM.
		c.falso("pw-record", "printf '%s\\n' \"$$\" >'"+pid+"'\nwhile :; do printf '"+octal(senoide)+"' || exit 0; done")
		b, ponta, acabou := fluxo(t, "mic", "nivel", "52")
		filho := pidDe(t, pid)
		// Espera chegarem uns níveis.
		for i := 0; strings.Count(b.String(), "\n") < 3; i++ {
			if i > 400 {
				t.Fatalf("%s: os níveis não chegaram: %q", nome, b.String())
			}
			time.Sleep(5 * time.Millisecond)
		}
		if !vivo(filho) {
			t.Fatalf("%s: o pw-record morreu cedo", nome)
		}
		comoPara(ponta)
		if codigo := espera(t, acabou, 4*time.Second); codigo != 0 {
			t.Errorf("%s: código %d", nome, codigo)
		}
		if vivo(filho) {
			t.Errorf("%s: o pw-record continua vivo", nome)
		}
		ls := linhas(t, b.String())
		igual(t, nome+": a última linha", ls[len(ls)-1], map[string]any{"fim": true})
		igual(t, nome+": um nível", ls[0], map[string]any{"pico": -6.0, "rms": -9.0})
	}
}

func TestMicNivel_FilhoQueIgnoraOSinalEMortoAForca(t *testing.T) {
	c := mundo(t)
	encurta(t, &prazoDeMorrer, 100*time.Millisecond)
	pid := filepath.Join(c.raiz, "pid")
	c.falso("pw-record", "trap '' TERM INT PIPE\nprintf '%s\\n' \"$$\" >'"+pid+"'\nprintf '"+octal(silencio)+"'\nwhile :; do :; done")
	_, ponta, acabou := fluxo(t, "mic", "nivel", "52")
	filho := pidDe(t, pid)
	ponta.Close()
	if codigo := espera(t, acabou, 4*time.Second); codigo != 0 {
		t.Errorf("código %d", codigo)
	}
	if vivo(filho) {
		t.Error("o filho teimoso continua vivo")
	}
}

// --- mic gravar / ouvir / limpar ------------------------------------------------

// gravadorFalso é um pw-record de mentira para a gravação: escreve no arquivo
// pedido (o último argumento) e fica até o sinal, como o de verdade.
func gravadorFalso(pid string) string {
	return "for a in \"$@\"; do arquivo=$a; done\nprintf 'RIFFxxxxWAVEdados' >\"$arquivo\"\nprintf '%s\\n' \"$$\" >'" + pid + "'\n" +
		"trap 'exit 0' INT TERM\nwhile :; do :; done"
}

func TestMicGravar_Contrato(t *testing.T) {
	c := mundo(t)
	encurta(t, &duracaoDaGravacao, 150*time.Millisecond)
	encurta(t, &passoDaContagem, 50*time.Millisecond)
	pid := filepath.Join(c.raiz, "pid")
	c.falso("pw-record", gravadorFalso(pid))
	b, _, acabou := fluxo(t, "mic", "gravar", "52")
	if codigo := espera(t, acabou, 5*time.Second); codigo != 0 {
		t.Fatalf("código %d: %s", codigo, b.String())
	}
	if got := b.String(); got != "{\"falta\":3}\n{\"falta\":2}\n{\"falta\":1}\n{\"ok\":true,\"segundos\":0}\n" {
		t.Errorf("saída:\n%s", got)
	}
	// O arquivo fica na pasta em memória, e só lá.
	arquivo := filepath.Join(c.run, "mksom", "teste.wav")
	igual(t, "chamadas", c.chamadas(), []string{"pw-dump", "pw-record --target 311 --format s16 --rate 48000 --channels 1 " + arquivo})
	if conteudo(t, arquivo) != "RIFFxxxxWAVEdados" {
		t.Error("a gravação não está onde devia")
	}
	if i, _ := os.Stat(filepath.Dir(arquivo)); i.Mode().Perm() != 0o700 {
		t.Errorf("modo da pasta: %o", i.Mode().Perm())
	}
	if vivo(pidDe(t, pid)) {
		t.Error("o pw-record continua vivo depois da gravação")
	}
	if existe(c.config) {
		t.Error("gravar não escreve na pasta do app")
	}
}

func TestMicGravar_InterrompidoApagaOArquivo(t *testing.T) {
	c := mundo(t)
	pid := filepath.Join(c.raiz, "pid")
	c.falso("pw-record", gravadorFalso(pid))
	b, ponta, acabou := fluxo(t, "mic", "gravar", "52")
	filho := pidDe(t, pid)
	ponta.Close()
	if codigo := espera(t, acabou, 4*time.Second); codigo != 0 {
		t.Errorf("código %d", codigo)
	}
	ls := linhas(t, b.String())
	igual(t, "fim", ls[len(ls)-1], map[string]any{"fim": true})
	if existe(filepath.Join(c.run, "mksom", "teste.wav")) || vivo(filho) {
		t.Error("interrompido: sobrou o arquivo ou o filho")
	}
}

func TestMicGravar_SemPastaEmMemoriaRecusaEErros(t *testing.T) {
	c := mundo(t)
	c.falso("pw-record", "exit 0")
	t.Setenv("XDG_RUNTIME_DIR", "")
	if frase := querErro(t, 2, "", "mic", "gravar", "52"); !strings.Contains(frase, "não vai para disco") {
		t.Errorf("frase: %s", frase)
	}
	querErro(t, 2, "", "mic", "ouvir")
	igual(t, "nada rodou", len(c.chamadas()), 0)
	t.Setenv("XDG_RUNTIME_DIR", c.run)

	// O pw-record que morre na hora, e o que não grava nada.
	encurta(t, &duracaoDaGravacao, 60*time.Millisecond)
	encurta(t, &passoDaContagem, 30*time.Millisecond)
	c.falso("pw-record", "echo 'error: no target' >&2; exit 1")
	b, _, acabou := fluxo(t, "mic", "gravar", "52")
	if codigo := espera(t, acabou, 4*time.Second); codigo != 1 || !strings.Contains(b.String(), "a gravação parou antes da hora: error: no target") {
		t.Errorf("morreu na hora: %s", b.String())
	}
}

func TestMicGravar_GravacaoVaziaEErro(t *testing.T) {
	c := mundo(t)
	encurta(t, &duracaoDaGravacao, 60*time.Millisecond)
	encurta(t, &passoDaContagem, 30*time.Millisecond)
	c.falso("pw-record", "trap 'exit 0' INT TERM\nwhile :; do :; done")
	b, _, acabou := fluxo(t, "mic", "gravar", "52")
	if codigo := espera(t, acabou, 4*time.Second); codigo != 1 || !strings.Contains(b.String(), "a gravação saiu vazia") {
		t.Errorf("vazia: %s", b.String())
	}
}

func TestMicOuvir_Contrato(t *testing.T) {
	c := mundo(t)
	arquivo := filepath.Join(c.run, "mksom", "teste.wav")
	grava := func() {
		if err := os.MkdirAll(filepath.Dir(arquivo), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(arquivo, []byte("RIFF"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// Sem gravação: recusado, e o pw-play não roda.
	c.falso("pw-play", "")
	if frase := querErro(t, 2, "", "mic", "ouvir"); !strings.Contains(frase, "grave primeiro") {
		t.Errorf("frase: %s", frase)
	}
	igual(t, "nada rodou", len(c.chamadas()), 0)

	// Toca e apaga.
	grava()
	b, _, acabou := fluxo(t, "mic", "ouvir")
	if codigo := espera(t, acabou, 4*time.Second); codigo != 0 || b.String() != "{\"ok\":true}\n" {
		t.Errorf("ouvir: %s", b.String())
	}
	igual(t, "chamadas", c.chamadas(), []string{"pw-play " + arquivo})
	if existe(arquivo) {
		t.Error("a gravação não foi apagada depois de tocar")
	}
}

func TestMicOuvir_ErroEInterrupcaoTambemApagam(t *testing.T) {
	c := mundo(t)
	arquivo := filepath.Join(c.run, "mksom", "teste.wav")
	grava := func() {
		_ = os.MkdirAll(filepath.Dir(arquivo), 0o700)
		if err := os.WriteFile(arquivo, []byte("RIFF"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	grava()
	c.falso("pw-play", "echo 'error: sem saída' >&2; exit 1")
	b, _, acabou := fluxo(t, "mic", "ouvir")
	if codigo := espera(t, acabou, 4*time.Second); codigo != 1 || !strings.Contains(b.String(), "não consegui tocar a gravação: error: sem saída") {
		t.Errorf("erro: %s", b.String())
	}
	if existe(arquivo) {
		t.Error("com erro a gravação também some")
	}

	grava()
	pid := filepath.Join(c.raiz, "pid")
	c.falso("pw-play", "printf '%s\\n' \"$$\" >'"+pid+"'\nwhile :; do :; done")
	b2, ponta, acabou2 := fluxo(t, "mic", "ouvir")
	filho := pidDe(t, pid)
	ponta.Close()
	if codigo := espera(t, acabou2, 4*time.Second); codigo != 0 || b2.String() != "{\"fim\":true}\n" {
		t.Errorf("interrompido: %s", b2.String())
	}
	if existe(arquivo) || vivo(filho) {
		t.Error("interrompido: sobrou o arquivo ou o pw-play")
	}
}

func TestMicLimpar(t *testing.T) {
	c := mundo(t)
	arquivo := filepath.Join(c.run, "mksom", "teste.wav")
	_ = os.MkdirAll(filepath.Dir(arquivo), 0o700)
	if err := os.WriteFile(arquivo, []byte("RIFF"), 0o600); err != nil {
		t.Fatal(err)
	}
	igual(t, "limpar", quer(t, "", "mic", "limpar"), map[string]any{"ok": true})
	if existe(arquivo) {
		t.Error("não apagou")
	}
	// Sem nada para apagar (e sem pasta em memória) continua certo.
	quer(t, "", "mic", "limpar")
	t.Setenv("XDG_RUNTIME_DIR", "")
	quer(t, "", "mic", "limpar")
	igual(t, "nada rodou", len(c.chamadas()), 0)
}

// --- o que o motor nunca faz ----------------------------------------------------

func TestNadaEscreveNaConfiguracaoDoPipewire(t *testing.T) {
	c := mundo(t)
	// Mesmo com as variáveis apontando para lá, o motor recusa.
	t.Setenv("MKSOM_CONFIG", filepath.Join(c.raiz, "config", "pipewire", "pipewire.conf.d"))
	c.sobeComOSystemctl("mksom.eq")
	if frase := querErro(t, 1, "", "eq", "ligar"); !strings.Contains(frase, "não escreve na configuração do PipeWire") {
		t.Errorf("frase: %s", frase)
	}
	querErro(t, 1, pedidoDe(0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0), "eq", "ajustar")
	if existe(filepath.Join(c.raiz, "config", "pipewire")) {
		t.Error("criou a pasta do PipeWire")
	}
	for _, ch := range c.chamadas() {
		if strings.HasPrefix(ch, "systemctl") {
			t.Errorf("subiu a unidade sem conf: %s", ch)
		}
	}
}

func TestUmCicloInteiroSoTocaNasPastasDoApp(t *testing.T) {
	c := mundo(t)
	c.sobeComOSystemctl("mksom.eq")
	quer(t, "", "eq", "ligar")
	quer(t, pedidoDe(-2, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0), "eq", "ajustar")
	quer(t, "", "eq", "predefinicao", "guardar", "--", "minha")
	quer(t, "", "eq", "desligar")
	var achados []string
	_ = filepath.Walk(filepath.Join(c.raiz, "config"), func(p string, i os.FileInfo, err error) error {
		if err == nil && !i.IsDir() {
			rel, _ := filepath.Rel(c.raiz, p)
			achados = append(achados, rel)
		}
		return nil
	})
	igual(t, "arquivos", achados, []string{
		"config/mk/mksom/eq.conf", "config/mk/mksom/estado.json", "config/mk/mksom/predefinicoes.json", "config/systemd/user/mksom-eq.service"})
}

func TestEqLigar_SeOWirePlumberNaoTrocaAPadraoAvisaEFicaNoAr(t *testing.T) {
	c := mundo(t)
	encurta(t, &esperaDoPadrao, 40*time.Millisecond)
	encurta(t, &passoDaEspera, 10*time.Millisecond)
	// O sink sobe e o set-default é aceito, mas a saída padrão continua a mesma.
	c.sobeComOSystemctl(altoFalantes)
	r := quer(t, "", "eq", "ligar")
	if r["ligado"] != true || r["na_saida"] != false {
		t.Errorf("resposta: %v", r)
	}
	if aviso, _ := r["aviso"].(string); !strings.Contains(aviso, "não o tornou a saída padrão") {
		t.Errorf("aviso: %v", r["aviso"])
	}
	// Nada é desfeito: o equalizador está no ar, só não é a padrão.
	for _, ch := range c.chamadas() {
		if strings.Contains(ch, "disable") {
			t.Errorf("desfez sem motivo: %s", ch)
		}
	}
	if st := estadoGuardado(t, c); !st.Ligado {
		t.Errorf("estado: %+v", st)
	}
}
