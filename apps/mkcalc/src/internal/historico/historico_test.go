package historico

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// casa aponta o histórico para uma pasta do teste e fixa o relógio.
func casa(t *testing.T) string {
	t.Helper()
	estado := t.TempDir()
	t.Setenv("XDG_STATE_HOME", estado)
	t.Setenv("MKCALC_HISTORICO_LIMITE", "")
	hora := int64(1795000000)
	Agora = func() time.Time { hora++; return time.Unix(hora, 0) }
	t.Cleanup(func() { Agora = time.Now })
	return filepath.Join(estado, "mk", "mkcalc", "historico.json")
}

func guarda(t *testing.T, expr, res string) []Item {
	t.Helper()
	itens, err := Guardar(expr, res, res, false)
	if err != nil {
		t.Fatal(err)
	}
	return itens
}

func TestOCaminhoSegueOXDG(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/x/estado")
	if c := Caminho(); c != "/x/estado/mk/mkcalc/historico.json" {
		t.Errorf("com XDG_STATE_HOME: %s", c)
	}
	// Vazio ou relativo não vale (é o que a especificação do XDG manda): cai na home.
	for _, v := range []string{"", "relativo/estado"} {
		t.Setenv("XDG_STATE_HOME", v)
		t.Setenv("HOME", "/casa/ana")
		if c := Caminho(); c != "/casa/ana/.local/state/mk/mkcalc/historico.json" {
			t.Errorf("XDG_STATE_HOME=%q: %s", v, c)
		}
	}
}

func TestSemArquivoOHistoricoEVazioENaoCriaNada(t *testing.T) {
	arq := casa(t)
	if itens := Ler(); itens == nil || len(itens) != 0 {
		t.Errorf("devia ser uma lista vazia (e não nil, que vira null no JSON): %#v", itens)
	}
	if _, err := os.Stat(filepath.Dir(arq)); err == nil {
		t.Error("ler não devia criar a pasta")
	}
}

func TestGuardarPoeNaFrenteComIdQueSoCresce(t *testing.T) {
	arq := casa(t)
	guarda(t, "1 + 1", "2")
	guarda(t, "2 + 2", "4")
	itens := guarda(t, "3 + 3", "6")
	if len(itens) != 3 || itens[0].Expressao != "3 + 3" || itens[0].ID != 3 || itens[2].ID != 1 {
		t.Fatalf("%+v", itens)
	}
	if itens[0].QuandoS <= itens[1].QuandoS {
		t.Errorf("a hora não andou: %+v", itens)
	}
	// O que está no disco é o que o Ler devolve, no formato do contrato.
	var a struct {
		Versao int
		Itens  []map[string]any
	}
	dados, _ := os.ReadFile(arq)
	if err := json.Unmarshal(dados, &a); err != nil || a.Versao != 1 || len(a.Itens) != 3 {
		t.Fatalf("arquivo: %v %s", err, dados)
	}
	for _, campo := range []string{"id", "expressao", "resultado", "cru", "aproximado", "quando_s"} {
		if _, tem := a.Itens[0][campo]; !tem {
			t.Errorf("falta o campo %s no arquivo", campo)
		}
	}
	// Apagar o do meio e guardar outro: o id não é reaproveitado para baixo.
	if _, err := Apagar(3); err != nil {
		t.Fatal(err)
	}
	if itens = guarda(t, "4 + 4", "8"); itens[0].ID != 3 || len(itens) != 3 {
		t.Errorf("depois de apagar o mais novo: %+v", itens)
	}
	if info, _ := os.Stat(arq); info.Mode().Perm() != 0o600 {
		t.Errorf("o histórico é só de quem fez as contas: %v", info.Mode().Perm())
	}
}

func TestAMesmaContaDeNovoSoAtualizaAHora(t *testing.T) {
	casa(t)
	guarda(t, "1 + 1", "2")
	antes := guarda(t, "2 + 2", "4")
	depois := guarda(t, "2 + 2", "4")
	if len(depois) != 2 || depois[0].ID != antes[0].ID || depois[0].QuandoS <= antes[0].QuandoS {
		t.Errorf("antes %+v depois %+v", antes, depois)
	}
	// Mas a mesma conta mais para trás entra de novo.
	if itens := guarda(t, "1 + 1", "2"); len(itens) != 3 {
		t.Errorf("%+v", itens)
	}
}

func TestOLimiteTiraAMaisVelha(t *testing.T) {
	casa(t)
	t.Setenv("MKCALC_HISTORICO_LIMITE", "3")
	var itens []Item
	for i := 1; i <= 5; i++ {
		itens = guarda(t, fmt.Sprintf("%d + 0", i), fmt.Sprint(i))
	}
	if len(itens) != 3 || itens[0].Cru != "5" || itens[2].Cru != "3" || len(Ler()) != 3 {
		t.Errorf("%+v", itens)
	}
	for _, torto := range []string{"0", "-4", "muitos"} {
		t.Setenv("MKCALC_HISTORICO_LIMITE", torto)
		if Limite() != LimitePadrao {
			t.Errorf("MKCALC_HISTORICO_LIMITE=%q devia valer o padrão", torto)
		}
	}
}

func TestApagarELimpar(t *testing.T) {
	arq := casa(t)
	guarda(t, "1 + 1", "2")
	guarda(t, "2 + 2", "4")
	itens, err := Apagar(1)
	if err != nil || len(itens) != 1 || itens[0].ID != 2 {
		t.Fatalf("%+v %v", itens, err)
	}
	if itens, err = Apagar(77); err != ErroSemItem || len(itens) != 1 {
		t.Errorf("apagar o que não existe: %+v %v", itens, err)
	}
	itens, err = Limpar()
	if err != nil || itens == nil || len(itens) != 0 || len(Ler()) != 0 {
		t.Errorf("limpar: %#v %v", itens, err)
	}
	if dados, _ := os.ReadFile(arq); !strings.Contains(string(dados), `"itens": []`) {
		t.Errorf("o arquivo limpo: %s", dados)
	}
}

func TestArquivoCorrompidoNaoDerruba(t *testing.T) {
	arq := casa(t)
	os.MkdirAll(filepath.Dir(arq), 0o755)
	for nome, conteudo := range map[string]string{
		"lixo":           "\x00\x01 isto não é json",
		"pela metade":    `{"versao": 1, "itens": [{"id": 1, "expr`,
		"vazio":          "",
		"outro formato":  `[1, 2, 3]`,
		"outra versão":   `{"versao": 9, "itens": [{"id": 1, "expressao": "1", "resultado": "1", "cru": "1"}]}`,
		"itens trocados": `{"versao": 1, "itens": "nada"}`,
	} {
		if err := os.WriteFile(arq, []byte(conteudo), 0o600); err != nil {
			t.Fatal(err)
		}
		if itens := Ler(); len(itens) != 0 {
			t.Errorf("%s: devia começar vazio, veio %+v", nome, itens)
		}
		// E a próxima conta regrava um arquivo bom.
		if itens := guarda(t, "1 + 1", "2"); len(itens) != 1 || itens[0].ID != 1 || len(Ler()) != 1 {
			t.Errorf("%s: depois de guardar: %+v", nome, itens)
		}
	}
	// Itens tortos no meio de um arquivo bom saem; os bons ficam.
	os.WriteFile(arq, []byte(`{"versao": 1, "itens": [
		{"id": 5, "expressao": "2 + 2", "resultado": "4", "cru": "4", "aproximado": false, "quando_s": 1},
		{"id": 5, "expressao": "repetido", "resultado": "4", "cru": "4"},
		{"id": 0, "expressao": "sem id", "resultado": "4", "cru": "4"},
		{"id": 3, "expressao": "", "resultado": "4", "cru": "4"},
		{"id": 2, "expressao": "1 + 1", "resultado": "2", "cru": "2", "aproximado": false, "quando_s": 1}]}`), 0o600)
	if itens := Ler(); len(itens) != 2 || itens[0].ID != 5 || itens[1].ID != 2 {
		t.Errorf("itens tortos: %+v", itens)
	}
	// Uma pasta no lugar do arquivo: vazio ao ler, erro (sem pânico) ao gravar.
	os.Remove(arq)
	os.Mkdir(arq, 0o755)
	if len(Ler()) != 0 {
		t.Error("pasta no lugar do arquivo devia dar histórico vazio")
	}
	if _, err := Guardar("1 + 1", "2", "2", false); err == nil {
		t.Error("gravar por cima de uma pasta devia dar erro")
	}
}

func TestAEscritaEAtomicaENaoDeixaTemporario(t *testing.T) {
	arq := casa(t)
	for i := 0; i < 20; i++ {
		guarda(t, fmt.Sprintf("%d + 0", i), fmt.Sprint(i))
	}
	entradas, _ := os.ReadDir(filepath.Dir(arq))
	if len(entradas) != 1 || entradas[0].Name() != "historico.json" {
		t.Errorf("sobrou coisa na pasta: %v", entradas)
	}
	// Sem poder escrever na pasta: erro, e o arquivo de antes continua inteiro.
	if os.Getuid() == 0 {
		return
	}
	antes, _ := os.ReadFile(arq)
	os.Chmod(filepath.Dir(arq), 0o500)
	defer os.Chmod(filepath.Dir(arq), 0o700)
	if _, err := Guardar("9 + 9", "18", "18", false); err == nil {
		t.Error("devia falhar numa pasta só de leitura")
	}
	if depois, _ := os.ReadFile(arq); string(depois) != string(antes) {
		t.Error("a escrita que falhou mexeu no arquivo")
	}
}
