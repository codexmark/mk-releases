package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// roda chama o motor como a linha de comando chamaria, com o histórico numa
// pasta do teste, e devolve a saída e o código.
func roda(t *testing.T, entrada string, args ...string) (string, int) {
	t.Helper()
	var saida bytes.Buffer
	codigo := rodar(args, strings.NewReader(entrada), &saida)
	return saida.String(), codigo
}

func estado(t *testing.T) string {
	t.Helper()
	pasta := t.TempDir()
	t.Setenv("XDG_STATE_HOME", pasta)
	t.Setenv("MKCALC_HISTORICO_LIMITE", "")
	return filepath.Join(pasta, "mk", "mkcalc", "historico.json")
}

// linhas lê a saída: um objeto JSON por linha, sem nada além disso.
func linhas(t *testing.T, saida string) []map[string]any {
	t.Helper()
	var objetos []map[string]any
	for _, l := range strings.Split(strings.TrimRight(saida, "\n"), "\n") {
		var o map[string]any
		if err := json.Unmarshal([]byte(l), &o); err != nil {
			t.Fatalf("linha que não é um objeto JSON: %q (%v)", l, err)
		}
		objetos = append(objetos, o)
	}
	return objetos
}

func TestCalcularRespondeOContrato(t *testing.T) {
	arq := estado(t)
	saida, codigo := roda(t, "", "calcular", "1.234,5 * 2")
	if codigo != certo || saida != `{"aproximado":false,"cru":"2469","expressao":"1234,5 × 2","resultado":"2.469"}`+"\n" {
		t.Errorf("%d %s", codigo, saida)
	}
	// Os pedaços da linha de comando se juntam; o "-" na frente é conta.
	if saida, codigo = roda(t, "", "calcular", "-2", "+", "3"); codigo != certo || linhas(t, saida)[0]["cru"] != "1" {
		t.Errorf("%d %s", codigo, saida)
	}
	if saida, codigo = roda(t, "", "calcular", "--", "-2+3"); codigo != certo || linhas(t, saida)[0]["cru"] != "1" {
		t.Errorf("%d %s", codigo, saida)
	}
	if saida, _ = roda(t, "", "calcular", "2/3"); linhas(t, saida)[0]["aproximado"] != true {
		t.Errorf("%s", saida)
	}
	// Sem --guardar nada vai para o disco.
	if _, err := os.Stat(arq); err == nil {
		t.Error("calcular sem --guardar escreveu o histórico")
	}
}

func TestContaRecusadaSaiCom2EDizSeEstaIncompleta(t *testing.T) {
	estado(t)
	for _, c := range []struct {
		expr, frase string
		incompleta  bool
	}{
		{"1/0", "divisão por zero", false},
		{"2+", "expressão incompleta", true},
		{"(2", "parêntese sem par", true},
		{"9^9^9", "expoente grande demais", false},
	} {
		saida, codigo := roda(t, "", "calcular", c.expr)
		o := linhas(t, saida)[0]
		if codigo != recusada || o["erro"] != c.frase || o["incompleta"] != c.incompleta || len(o) != 2 {
			t.Errorf("%q: %d %s", c.expr, codigo, saida)
		}
	}
}

func TestUsoErradoSaiCom1(t *testing.T) {
	estado(t)
	for _, args := range [][]string{{}, {"nada"}, {"calcular"}, {"calcular", "--guardar"}, {"historico", "x"}, {"apagar"}, {"apagar", "x"}, {"apagar", "1", "2"}, {"limpar", "x"}, {"servir", "x"}} {
		saida, codigo := roda(t, "", args...)
		if erro, _ := linhas(t, saida)[0]["erro"].(string); codigo != falhou || !strings.HasPrefix(erro, "uso: mkcalc-motor <") {
			t.Errorf("%v: %d %s", args, codigo, saida)
		}
	}
}

func TestGuardarHistoricoApagarLimpar(t *testing.T) {
	arq := estado(t)
	saida, codigo := roda(t, "", "historico")
	if codigo != certo || saida != `{"itens":[]}`+"\n" {
		t.Errorf("histórico vazio: %d %s", codigo, saida)
	}
	roda(t, "", "calcular", "--guardar", "1+1")
	saida, codigo = roda(t, "", "calcular", "200+10%", "--guardar")
	o := linhas(t, saida)[0]
	itens, _ := o["itens"].([]any)
	if codigo != certo || o["cru"] != "220" || len(itens) != 2 {
		t.Fatalf("%d %s", codigo, saida)
	}
	primeiro := itens[0].(map[string]any)
	if primeiro["expressao"] != "200 + 10%" || primeiro["resultado"] != "220" || primeiro["cru"] != "220" || primeiro["aproximado"] != false || primeiro["id"] != float64(2) || primeiro["quando_s"].(float64) < 1e9 {
		t.Errorf("o item guardado: %v", primeiro)
	}
	// Uma conta recusada não entra.
	if saida, codigo = roda(t, "", "calcular", "--guardar", "1/0"); codigo != recusada || strings.Contains(saida, "itens") {
		t.Errorf("%d %s", codigo, saida)
	}
	if saida, _ = roda(t, "", "historico"); len(linhas(t, saida)[0]["itens"].([]any)) != 2 {
		t.Errorf("%s", saida)
	}
	if saida, codigo = roda(t, "", "apagar", "1"); codigo != certo || len(linhas(t, saida)[0]["itens"].([]any)) != 1 {
		t.Errorf("apagar: %d %s", codigo, saida)
	}
	if saida, codigo = roda(t, "", "apagar", "1"); codigo != falhou || linhas(t, saida)[0]["erro"] != "essa conta não está mais no histórico" {
		t.Errorf("apagar de novo: %d %s", codigo, saida)
	}
	if saida, codigo = roda(t, "", "limpar"); codigo != certo || saida != `{"itens":[]}`+"\n" {
		t.Errorf("limpar: %d %s", codigo, saida)
	}
	// Histórico que não dá para gravar: a conta sai do mesmo jeito, com o aviso.
	os.Remove(arq)
	os.Mkdir(arq, 0o755)
	saida, codigo = roda(t, "", "calcular", "--guardar", "2+2")
	o = linhas(t, saida)[0]
	aviso, _ := o["aviso"].(string)
	if _, tem := o["itens"]; codigo != certo || o["cru"] != "4" || tem || !strings.HasPrefix(aviso, "não deu para guardar no histórico") {
		t.Errorf("%d %s", codigo, saida)
	}
	if saida, codigo = roda(t, "", "limpar"); codigo != falhou || !strings.HasPrefix(linhas(t, saida)[0]["erro"].(string), "não deu para gravar o histórico") {
		t.Errorf("limpar sem poder gravar: %d %s", codigo, saida)
	}
}

func TestServirUmaLinhaPorPedidoComOMesmoId(t *testing.T) {
	estado(t)
	pedidos := strings.Join([]string{
		`{"id": 1, "op": "calcular", "expr": "0,1+0,2"}`,
		`{"id": "prévia-7", "op": "calcular", "expr": "2+"}`,
		``,
		`isto não é json`,
		`{"id": 3, "op": "calcular", "expr": "6x7", "guardar": true}`,
		`{"id": 4, "op": "historico"}`,
		`{"id": 5, "op": "dançar"}`,
		`{"op": "calcular", "expr": "1/0"}`,
		`{"id": {"a": 1}, "op": "calcular", "expr": "1"}`,
		`{"id": 7, "op": "apagar"}`,
		`{"id": 8, "op": "apagar", "item": 99}`,
		`{"id": 9, "op": "apagar", "item": 1}`,
		`{"id": 10, "op": "calcular", "expr": "9^9^9", "guardar": true}`,
		`{"id": 11, "op": "limpar"}`,
		`{"id": 12345678901234567890, "op": "calcular", "expr": "1"}`,
	}, "\n") + "\n"
	saida, codigo := roda(t, pedidos, "servir")
	if codigo != certo {
		t.Fatalf("código %d", codigo)
	}
	r := linhas(t, saida)
	if len(r) != 14 {
		t.Fatalf("%d respostas para 14 pedidos:\n%s", len(r), saida)
	}
	confere := func(i int, id any, campo string, valor any) {
		t.Helper()
		if r[i]["id"] != id || r[i][campo] != valor {
			t.Errorf("resposta %d: esperado id %v e %s=%v, veio %v", i, id, campo, valor, r[i])
		}
	}
	confere(0, float64(1), "resultado", "0,3")
	confere(1, "prévia-7", "incompleta", true)
	confere(2, nil, "erro", "pedido que não é JSON")
	confere(3, float64(3), "expressao", "6 × 7")
	if itens, _ := r[3]["itens"].([]any); len(itens) != 1 {
		t.Errorf("guardar devia devolver o histórico: %v", r[3])
	}
	if itens, _ := r[4]["itens"].([]any); r[4]["id"] != float64(4) || len(itens) != 1 {
		t.Errorf("historico: %v", r[4])
	}
	confere(5, float64(5), "erro", "não conheço o pedido «dançar»")
	confere(6, nil, "erro", "divisão por zero")
	confere(7, nil, "cru", "1")
	confere(8, float64(7), "erro", `o pedido apagar precisa de "item"`)
	confere(9, float64(8), "erro", "essa conta não está mais no histórico")
	if itens, tem := r[10]["itens"].([]any); r[10]["id"] != float64(9) || !tem || len(itens) != 0 {
		t.Errorf("apagar: %v", r[10])
	}
	confere(11, float64(10), "erro", "expoente grande demais")
	if itens, tem := r[12]["itens"].([]any); r[12]["id"] != float64(11) || !tem || len(itens) != 0 {
		t.Errorf("limpar: %v", r[12])
	}
	// Um id que não cabe num float volta com todos os algarismos.
	if !strings.Contains(saida, `"id":12345678901234567890`) {
		t.Errorf("o id grande mudou:\n%s", saida)
	}
	// Todo "id" aparece, mesmo null: é por ele que a janela casa as respostas.
	for i, o := range r {
		if _, tem := o["id"]; !tem {
			t.Errorf("resposta %d sem id: %v", i, o)
		}
	}
}

func TestServirSemPedidoEComLinhaGrandeDemais(t *testing.T) {
	estado(t)
	if saida, codigo := roda(t, "", "servir"); codigo != certo || saida != "" {
		t.Errorf("entrada vazia: %d %q", codigo, saida)
	}
	// O último pedido sem quebra de linha também é atendido.
	if saida, codigo := roda(t, `{"id":1,"op":"calcular","expr":"1+1"}`, "servir"); codigo != certo || linhas(t, saida)[0]["cru"] != "2" {
		t.Errorf("%d %q", codigo, saida)
	}
	grande := `{"id":1,"op":"calcular","expr":"` + strings.Repeat("1", limiteDaLinha) + `"}` + "\n"
	saida, codigo := roda(t, `{"id":0,"op":"calcular","expr":"1"}`+"\n"+grande, "servir")
	r := linhas(t, saida)
	if codigo != falhou || len(r) != 2 || r[1]["erro"] != "pedido grande demais" {
		t.Errorf("%d %q", codigo, saida)
	}
}
