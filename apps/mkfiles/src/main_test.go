package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// casa monta a pasta pessoal de mentira: nenhum teste daqui toca fora dela.
func casa(t *testing.T) string {
	t.Helper()
	c, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", c)
	for _, v := range []string{"CONFIG", "DATA", "CACHE", "STATE"} {
		t.Setenv("XDG_"+v+"_HOME", filepath.Join(c, ".xdg", strings.ToLower(v)))
	}
	t.Setenv("XDG_DATA_DIRS", filepath.Join(c, ".sistema"))
	t.Setenv("XDG_CONFIG_DIRS", filepath.Join(c, ".sistema"))
	montagens := filepath.Join(c, ".mountinfo")
	os.WriteFile(montagens, []byte("1 0 8:1 / / rw - ext4 /dev/raiz rw\n"), 0o644)
	t.Setenv("MKFILES_MOUNTINFO", montagens)
	t.Setenv("MKFILES_MIME", filepath.Join(c, ".mime"))
	t.Setenv("MKFILES_GVFS", filepath.Join(c, ".gvfs"))
	for _, comando := range []string{"LSBLK", "UDISKSCTL", "WLCOPY", "WLPASTE", "GIO"} {
		t.Setenv("MKFILES_"+comando, filepath.Join(c, "nao-ha"))
	}
	return c
}

func roda(t *testing.T, entrada string, args ...string) (int, []map[string]any) {
	t.Helper()
	var saida bytes.Buffer
	codigo := rodar(args, strings.NewReader(entrada), &saida)
	var linhas []map[string]any
	for _, l := range strings.Split(strings.TrimSpace(saida.String()), "\n") {
		var m map[string]any
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			t.Fatalf("linha que não é JSON: %q", l)
		}
		linhas = append(linhas, m)
	}
	return codigo, linhas
}

func TestUso(t *testing.T) {
	casa(t)
	for _, args := range [][]string{nil, {"--ajuda"}, {"servir", "a-mais"}} {
		codigo, linhas := roda(t, "", args...)
		erro, _ := linhas[0]["erro"].(string)
		if codigo != 1 || !strings.HasPrefix(erro, "uso: mkfiles-motor <") {
			t.Errorf("%v: código %d, %v", args, codigo, linhas)
		}
	}
	codigo, linhas := roda(t, "", "dançar")
	if codigo != 1 || linhas[0]["erro"] != "não conheço o pedido «dançar»" {
		t.Errorf("op desconhecido: %d %v", codigo, linhas)
	}
}

func TestSobreDizTodosOsPedidos(t *testing.T) {
	casa(t)
	codigo, linhas := roda(t, "", "sobre")
	if codigo != 0 || linhas[0]["versao"] != float64(1) {
		t.Fatalf("sobre: %d %v", codigo, linhas)
	}
	var ops []string
	for _, o := range linhas[0]["ops"].([]any) {
		ops = append(ops, o.(string))
	}
	esperados := "abrir apagar apps area-gravar area-ler area-por buscar cancelar compactado-abrir compactado-info compactado-listar compactar copiar criar-arquivo criar-pasta " +
		"definir-padrao desfazer desmontar desvigiar discos ejetar espaco estado-gravar extrair favorito-mover favorito-por " +
		"favorito-renomear favorito-tirar formatos item ligar listar lixeira lixeira-apagar lixeira-estado lixeira-esvaziar " +
		"lixeira-listar lugares miniatura miniatura-cancelar modelos montar mover permissoes prefs prefs-gravar previa " +
		"propriedades refazer remoto-desmontar remoto-hosts remoto-montar remoto-responder remotos renomear renomear-lote responder restaurar sobre tarefa-cancelar tarefa-pausar tarefas terminal " +
		"vigiar vigiar-discos"
	if strings.Join(ops, " ") != esperados {
		t.Errorf("os pedidos ligados:\n%s\nesperados:\n%s", strings.Join(ops, " "), esperados)
	}
	ext := linhas[0]["extensoes"].(map[string]any)
	for e, ligada := range map[string]bool{"lote": true, "compactados": true, "previa": true, "remotos": true} {
		if ext[e] != ligada {
			t.Errorf("extensão %s: %v", e, ext[e])
		}
	}
	// O sobre diz onde os remotos aparecem (a pasta do gvfs).
	if linhas[0]["gvfs"] != os.Getenv("MKFILES_GVFS") {
		t.Errorf("gvfs: %v", linhas[0]["gvfs"])
	}
	// Um pedido que não existe continua sendo erro.
	if c, l := roda(t, "", "remoto-inventado"); c != 1 || !strings.Contains(l[0]["erro"].(string), "não conheço") {
		t.Errorf("pedido inventado: %v", l)
	}
}

func TestPedidoDe(t *testing.T) {
	linha, err := pedidoDe([]string{"buscar", "/a b", "--nome=*.jpg", "--ocultos=true", "--tamanho-min=10", "--conteudo", "--", "--x"})
	if err != nil {
		t.Fatal(err)
	}
	var p map[string]any
	json.Unmarshal(linha, &p)
	aqui, _ := os.Getwd()
	if p["op"] != "buscar" || p["caminho"] != "/a b" || p["nome"] != "*.jpg" || p["ocultos"] != true ||
		p["tamanho_min"] != float64(10) || p["conteudo"] != "" || p["caminhos"].([]any)[1] != filepath.Join(aqui, "--x") {
		t.Errorf("pedidoDe: %s", linha)
	}
}

func TestUmaChamadaSo(t *testing.T) {
	c := casa(t)
	pasta := filepath.Join(c, "p")
	os.MkdirAll(pasta, 0o755)
	os.WriteFile(filepath.Join(pasta, "a.txt"), []byte("oi"), 0o644)
	codigo, linhas := roda(t, "", "listar", pasta)
	if codigo != 0 || len(linhas) != 1 || linhas[0]["total"] != float64(1) || linhas[0]["fim"] != true {
		t.Fatalf("listar: %d %v", codigo, linhas)
	}
	if _, temID := linhas[0]["id"]; temID {
		t.Error("a chamada só não tem id")
	}
	// A chamada espera a tarefa acabar: quando volta, a pasta já existe.
	codigo, linhas = roda(t, "", "criar-pasta", "--destino="+pasta, "--nome=nova")
	if _, err := os.Stat(filepath.Join(pasta, "nova")); codigo != 0 || err != nil {
		t.Fatalf("criar-pasta: %d %v %v", codigo, linhas, err)
	}
	// Erro sai com 1; recusa de segurança, com 2, e nada é tocado.
	if codigo, _ = roda(t, "", "listar", filepath.Join(c, "nao-ha")); codigo != 1 {
		t.Errorf("listar o que não existe: %d", codigo)
	}
	for _, alvo := range []string{"/", c, filepath.Dir(c)} {
		for _, op := range []string{"apagar", "lixeira"} {
			codigo, linhas = roda(t, "", op, alvo)
			if codigo != 2 || linhas[0]["recusado"] != true {
				t.Errorf("%s %s: código %d, %v", op, alvo, codigo, linhas)
			}
		}
	}
	if codigo, linhas = roda(t, "", "apagar"); codigo == 0 || linhas[0]["erro"] == nil {
		t.Errorf("apagar sem caminho: %d %v", codigo, linhas)
	}
	if _, err := os.Stat(filepath.Join(pasta, "a.txt")); err != nil {
		t.Error("as recusas não podem tocar em nada")
	}
}

func TestServir(t *testing.T) {
	c := casa(t)
	pasta := filepath.Join(c, "p")
	os.MkdirAll(pasta, 0o755)
	entrada := `{"id":1,"op":"listar","caminho":"` + pasta + `"}
lixo

{"id":"b","op":"dançar"}
{"id":3,"op":"listar","caminho":"relativo"}
{"id":4,"op":"apagar","origens":["/"]}
`
	codigo, linhas := roda(t, entrada, "servir")
	if codigo != 0 || len(linhas) != 5 {
		t.Fatalf("servir: código %d, %d linhas: %v", codigo, len(linhas), linhas)
	}
	porID := map[any]map[string]any{}
	for _, l := range linhas {
		porID[l["id"]] = l
	}
	if porID[float64(1)]["fim"] != true || porID[nil]["erro"] != "pedido que não é JSON" ||
		porID["b"]["erro"] != "não conheço o pedido «dançar»" || porID[float64(3)]["erro"] == nil ||
		porID[float64(4)]["recusado"] != true {
		t.Errorf("servir: %v", linhas)
	}
}
