package prefs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"mkfiles/internal/prova"
)

func j(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

const prefsPadrao = `{"clique":"duplo","colunas":{"mtime":150,"permissoes":100,"tamanho":90,"tipo":150},"confirmar_lixeira":false,` +
	`"icone_px":64,"miniatura_remota_mb":5,"miniaturas_remotas":true,"ocultos":false,"ordem":{"campo":"nome","inversa":false},"pastas_antes":true,"visao":"icones","visao_por_pasta":{}}`
const estadoPadraoJSON = `{"ativo":0,"duplo":false,"lateral_px":200,"paineis":[]}`

func TestPadroes(t *testing.T) {
	m := prova.Novo(t, Ligar)
	r := m.Pede(`{"op":"prefs"}`)
	if j(r["prefs"]) != prefsPadrao {
		t.Errorf("prefs padrão:\n%s", j(r["prefs"]))
	}
	if j(r["estado"]) != estadoPadraoJSON {
		t.Errorf("estado padrão:\n%s", j(r["estado"]))
	}
	// Ler não cria arquivo.
	if _, err := os.Stat(m.Casa + "/.config/mk"); err == nil {
		t.Error("ler as preferências não devia gravar nada")
	}
}

func TestGravarPrefs(t *testing.T) {
	m := prova.Novo(t, Ligar)
	r := m.Pede(`{"op":"prefs-gravar","prefs":{"visao":"lista","ocultos":true,"icone_px":96,"clique":"simples",
		"ordem":{"campo":"mtime"},"colunas":{"tipo":220},"visao_por_pasta":{"/home/ana/Fotos":"icones"},"pastas_antes":false,"confirmar_lixeira":true}}`)
	quer := `{"clique":"simples","colunas":{"mtime":150,"permissoes":100,"tamanho":90,"tipo":220},"confirmar_lixeira":true,` +
		`"icone_px":96,"miniatura_remota_mb":5,"miniaturas_remotas":true,"ocultos":true,"ordem":{"campo":"mtime","inversa":false},"pastas_antes":false,"visao":"lista","visao_por_pasta":{"/home/ana/Fotos":"icones"}}`
	if j(r["prefs"]) != quer {
		t.Fatalf("gravar:\n%s\nesperado:\n%s", j(r["prefs"]), quer)
	}
	arq := m.Casa + "/.config/mk/mkfiles/preferencias.json"
	if b, err := os.ReadFile(arq); err != nil || !json.Valid(b) {
		t.Fatalf("o arquivo: %v", err)
	}
	if e, _ := os.ReadDir(filepath.Dir(arq)); len(e) != 1 {
		t.Errorf("sobrou temporário: %v", e)
	}
	// Só o que mudou: o resto fica; e o que foi gravado volta ao ler.
	r = m.Pede(`{"op":"prefs-gravar","prefs":{"ordem":{"inversa":true}}}`)
	if o := r["prefs"].(map[string]any)["ordem"]; j(o) != `{"campo":"mtime","inversa":true}` {
		t.Errorf("ordem parcial: %s", j(o))
	}
	if lido := m.Pede(`{"op":"prefs"}`)["prefs"]; !reflect.DeepEqual(lido, r["prefs"]) {
		t.Errorf("o que foi gravado não é o que se lê: %s", j(lido))
	}

	// Chave desconhecida e valor fora do lugar: ignorados, um por um.
	antes := j(r["prefs"])
	r = m.Pede(`{"op":"prefs-gravar","prefs":{"visao":"mosaico","ocultos":"sim","icone_px":1000,"clique":"triplo","inventada":1,
		"ordem":{"campo":"cor","inversa":"x"},"colunas":{"tipo":5,"nome":300,"tamanho":"larga","mtime":64.5},"pastas_antes":null,
		"visao_por_pasta":{"relativa":"lista","/a/../b":"lista","/home/ana/Musicas":"mosaico"},"visao_por_pasta_ordem":["/x"]}}`)
	if j(r["prefs"]) != antes {
		t.Errorf("valores tortos mudaram algo:\n%s\nantes:\n%s", j(r["prefs"]), antes)
	}
	if r = m.Pede(`{"op":"prefs-gravar","prefs":"torto"}`); j(r["prefs"]) != antes {
		t.Errorf("prefs que não é objeto: %v", r)
	}
	// icone_px nos limites.
	for px, vale := range map[int]bool{31: false, 32: true, 256: true, 257: false} {
		r = m.Pede(fmt.Sprintf(`{"op":"prefs-gravar","prefs":{"icone_px":%d}}`, px))
		if (r["prefs"].(map[string]any)["icone_px"] == float64(px)) != vale {
			t.Errorf("icone_px %d: %v", px, r["prefs"].(map[string]any)["icone_px"])
		}
	}
}

func TestVisaoPorPasta(t *testing.T) {
	m := prova.Novo(t, Ligar)
	m.Pede(`{"op":"prefs-gravar","prefs":{"visao_por_pasta":{"/a":"lista","/b":"compacta"}}}`)
	// null esquece uma pasta; as outras ficam.
	r := m.Pede(`{"op":"prefs-gravar","prefs":{"visao_por_pasta":{"/a":null,"/c":"icones"}}}`)
	if j(r["prefs"].(map[string]any)["visao_por_pasta"]) != `{"/b":"compacta","/c":"icones"}` {
		t.Errorf("esquecer: %s", j(r["prefs"]))
	}
	// Até 500: a mais antiga sai. Regravar uma pasta a torna a mais nova.
	for i := 0; i < LimiteDePastas-2; i++ {
		m.Pede(fmt.Sprintf(`{"op":"prefs-gravar","prefs":{"visao_por_pasta":{"/p/%d":"lista"}}}`, i))
	}
	m.Pede(`{"op":"prefs-gravar","prefs":{"visao_por_pasta":{"/b":"lista"}}}`)
	r = m.Pede(`{"op":"prefs-gravar","prefs":{"visao_por_pasta":{"/nova":"lista"}}}`)
	pastas := r["prefs"].(map[string]any)["visao_por_pasta"].(map[string]any)
	if len(pastas) != LimiteDePastas || pastas["/c"] != nil || pastas["/b"] != "lista" || pastas["/nova"] != "lista" || pastas["/p/0"] != "lista" {
		t.Errorf("limite de pastas: %d, /c=%v /b=%v", len(pastas), pastas["/c"], pastas["/b"])
	}
	// A idade sobrevive ao arquivo, e a janela não a vê.
	if strings.Contains(j(r), "visao_por_pasta_ordem") {
		t.Error("a ordem das pastas é só do arquivo")
	}
	r = m.Pede(`{"op":"prefs-gravar","prefs":{"visao_por_pasta":{"/mais":"lista"}}}`)
	pastas = r["prefs"].(map[string]any)["visao_por_pasta"].(map[string]any)
	if len(pastas) != LimiteDePastas || pastas["/p/0"] != nil || pastas["/p/1"] != "lista" {
		t.Errorf("a mais antiga sai: /p/0=%v /p/1=%v", pastas["/p/0"], pastas["/p/1"])
	}
}

func TestEstado(t *testing.T) {
	m := prova.Novo(t, Ligar)
	r := m.Pede(`{"op":"estado-gravar","estado":{"paineis":[{"abas":["/home/ana","/tmp"],"atual":1},{"abas":["/"],"atual":0}],"ativo":1,"duplo":true,"lateral_px":260}}`)
	quer := `{"ativo":1,"duplo":true,"lateral_px":260,"paineis":[{"abas":["/home/ana","/tmp"],"atual":1},{"abas":["/"],"atual":0}]}`
	if j(r["estado"]) != quer {
		t.Fatalf("estado:\n%s", j(r["estado"]))
	}
	if b, err := os.ReadFile(m.Casa + "/.local/state/mk/mkfiles/estado.json"); err != nil || !json.Valid(b) {
		t.Fatalf("o arquivo do estado: %v", err)
	}
	// Parcial, e o que não serve é ignorado.
	r = m.Pede(`{"op":"estado-gravar","estado":{"lateral_px":50,"ativo":2,"duplo":"x","paineis":[{},{},{}],"outra":1}}`)
	if j(r["estado"]) != quer {
		t.Errorf("valores tortos mudaram o estado: %s", j(r["estado"]))
	}
	if r = m.Pede(`{"op":"estado-gravar","estado":{"paineis":[{"abas":"x"}]}}`); j(r["estado"]) != quer {
		t.Errorf("painel torto: %s", j(r["estado"]))
	}
	// Abas: só caminhos absolutos e limpos, até 30; "atual" fora da lista vira 0.
	abas := []string{"relativa", "/a/../b"}
	for i := 0; i < 40; i++ {
		abas = append(abas, fmt.Sprintf("/p/%d", i))
	}
	r = m.Pede(`{"op":"estado-gravar","estado":{"paineis":[{"abas":` + j(abas) + `,"atual":35}]}}`)
	p := r["estado"].(map[string]any)["paineis"].([]any)[0].(map[string]any)
	if lista := p["abas"].([]any); len(lista) != LimiteDeAbas || lista[0] != "/p/0" || p["atual"] != float64(0) {
		t.Errorf("abas: %d, a primeira %v, atual %v", len(lista), lista[0], p["atual"])
	}
	if lido := m.Pede(`{"op":"prefs"}`)["estado"]; !reflect.DeepEqual(lido, r["estado"]) {
		t.Errorf("o estado gravado não é o que se lê: %s", j(lido))
	}
}

func TestArquivoCorrompido(t *testing.T) {
	m := prova.Novo(t, Ligar)
	m.Grava(".config/mk/mkfiles/preferencias.json", `{"visao": "lista", "icone_px`)
	m.Grava(".local/state/mk/mkfiles/estado.json", `[1, 2]`)
	r := m.Pede(`{"op":"prefs"}`)
	if j(r["prefs"]) != prefsPadrao || j(r["estado"]) != estadoPadraoJSON {
		t.Errorf("arquivo corrompido volta ao padrão: %s / %s", j(r["prefs"]), j(r["estado"]))
	}
	// Um arquivo válido com valores tortos: fica o que serve.
	m.Grava(".config/mk/mkfiles/preferencias.json", `{"visao": "compacta", "icone_px": 9999, "colunas": {"tipo": 200, "mtime": -1}}`)
	p := m.Pede(`{"op":"prefs"}`)["prefs"].(map[string]any)
	if p["visao"] != "compacta" || p["icone_px"] != float64(64) || j(p["colunas"]) != `{"mtime":150,"permissoes":100,"tamanho":90,"tipo":200}` {
		t.Errorf("arquivo com valores tortos: %s", j(p))
	}
	// E a próxima gravação o regrava inteiro.
	m.Grava(".config/mk/mkfiles/preferencias.json", `lixo`)
	m.Pede(`{"op":"prefs-gravar","prefs":{"ocultos":true}}`)
	if p := m.Pede(`{"op":"prefs"}`)["prefs"].(map[string]any); p["ocultos"] != true || p["visao"] != "icones" {
		t.Errorf("regravar por cima do lixo: %s", j(p))
	}
}

func TestAAbaDaLixeiraVolta(t *testing.T) {
	m := prova.Novo(t, Ligar)
	r := m.Pede(`{"op":"estado-gravar","estado":{"paineis":[{"abas":["/tmp","lixeira:","lixeira:/x","outra:"],"atual":1}]}}`)
	if got := j(r["estado"].(map[string]any)["paineis"]); got != `[{"abas":["/tmp","lixeira:"],"atual":1}]` {
		t.Errorf("a aba da lixeira é guardada, e só ela além dos caminhos: %s", got)
	}
}

func TestPrefsDosRemotos(t *testing.T) {
	m := prova.Novo(t, Ligar)
	r := m.Pede(`{"op":"prefs-gravar","prefs":{"miniaturas_remotas":false,"miniatura_remota_mb":20}}`)
	p := r["prefs"].(map[string]any)
	if p["miniaturas_remotas"] != false || p["miniatura_remota_mb"] != float64(20) {
		t.Errorf("gravar: %v", p)
	}
	// Fora de 1 a 100 não vale.
	for _, torto := range []string{"0", "101", `"x"`} {
		r = m.Pede(`{"op":"prefs-gravar","prefs":{"miniatura_remota_mb":` + torto + `}}`)
		if r["prefs"].(map[string]any)["miniatura_remota_mb"] != float64(20) {
			t.Errorf("valor torto %s entrou: %v", torto, r["prefs"])
		}
	}
}
