package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"mkescrita/internal/base"
	"mkescrita/internal/disco"
)

// mundo aponta o estado, as preferências e o /proc para pastas do teste.
func mundo(t *testing.T) string {
	t.Helper()
	raiz := t.TempDir()
	t.Setenv("HOME", filepath.Join(raiz, "casa"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(raiz, "estado"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(raiz, "config"))
	t.Setenv("MKESCRITA_PROC", filepath.Join(raiz, "proc"))
	t.Setenv("MKESCRITA_LIMITE_MB", "")
	textos := filepath.Join(raiz, "textos")
	os.MkdirAll(textos, 0o755)
	return textos
}

// roda chama o motor com `entrada` no stdin e devolve o código e a resposta.
func roda(t *testing.T, entrada string, args ...string) (int, map[string]any) {
	t.Helper()
	var saida bytes.Buffer
	base.Saida, base.Entrada = &saida, strings.NewReader(entrada)
	defer func() { base.Saida, base.Entrada = os.Stdout, os.Stdin }()
	codigo := rodar(args)
	linhas := strings.Split(strings.TrimSpace(saida.String()), "\n")
	if len(linhas) != 1 {
		t.Fatalf("%v: esperava uma linha de JSON, veio %q", args, saida.String())
	}
	var r map[string]any
	if err := json.Unmarshal([]byte(linhas[0]), &r); err != nil {
		t.Fatalf("%v: a resposta não é JSON: %q", args, linhas[0])
	}
	return codigo, r
}

func bytesDe(s string) string { return strconv.Itoa(len(s)) }

func TestSemSubcomandoMostraOUso(t *testing.T) {
	for _, args := range [][]string{{}, {"nada"}, {"--help"}} {
		codigo, r := roda(t, "", args...)
		erro, _ := r["erro"].(string)
		if codigo != base.Erro || !strings.HasPrefix(erro, "uso: mkescrita-motor <abrir | contar | estrutura | exportar | html | preferencias | rascunho | salvar | vigiar>") {
			t.Errorf("%v: %d %q", args, codigo, erro)
		}
	}
}

func TestAbrirESalvarDePontaAPonta(t *testing.T) {
	textos := mundo(t)
	arq := filepath.Join(textos, "-começa com traço.md")
	os.WriteFile(arq, []byte("# Um\r\ntexto\r\n"), 0o644)

	codigo, a := roda(t, "", "abrir", "--", arq)
	if codigo != base.Certo || a["texto"] != "# Um\ntexto\n" || a["fim_de_linha"] != "crlf" || a["quebra_final"] != true ||
		a["novo"] != false || a["so_leitura"] != false || a["nome"] != "-começa com traço.md" || a["arquivo"] != arq {
		t.Fatalf("abrir: %d %v", codigo, a)
	}
	imp := a["impressao"].(map[string]any)
	sha := imp["sha256"].(string)
	if imp["tamanho_bytes"] != float64(13) || imp["mtime_ns"].(float64) <= 0 || len(sha) != 64 {
		t.Errorf("impressão: %v", imp)
	}

	novo := "# Um\ntexto, ação\n"
	codigo, s := roda(t, novo, "salvar", "--esperado", sha, "--fim-de-linha", "crlf", "--quebra-final", "--bytes", bytesDe(novo), "--", arq)
	if codigo != base.Certo || s["arquivo"] != arq || s["nome"] != "-começa com traço.md" {
		t.Fatalf("salvar: %d %v", codigo, s)
	}
	if b, _ := os.ReadFile(arq); string(b) != "# Um\r\ntexto, ação\r\n" {
		t.Errorf("no disco: %q", b)
	}
	sha2 := s["impressao"].(map[string]any)["sha256"].(string)
	if sha2 != disco.Resumo([]byte("# Um\r\ntexto, ação\r\n")) {
		t.Error("a impressão devolvida não é a do disco")
	}

	// Mudou por fora: conflito, saída 2, nada gravado, e a impressão do que está lá.
	os.WriteFile(arq, []byte("de fora\n"), 0o644)
	codigo, c := roda(t, "o meu\n", "salvar", "--esperado", sha2, "--", arq)
	if codigo != base.Recusado || c["conflito"] != "mudou" || c["erro"] != nil || c["frase"] == "" ||
		c["impressao"].(map[string]any)["sha256"] != disco.Resumo([]byte("de fora\n")) {
		t.Errorf("conflito: %d %v", codigo, c)
	}
	if b, _ := os.ReadFile(arq); string(b) != "de fora\n" {
		t.Errorf("gravou apesar do conflito: %q", b)
	}
	// Sumiu.
	os.Remove(arq)
	codigo, c = roda(t, "o meu\n", "salvar", "--esperado", sha2, "--", arq)
	if codigo != base.Recusado || c["conflito"] != "sumiu" || c["impressao"] != nil {
		t.Errorf("sumiu: %d %v", codigo, c)
	}
	if _, has := c["impressao"]; !has {
		t.Error("o que o motor não sabe é null, não ausente")
	}
	// --forcar grava assim mesmo.
	codigo, _ = roda(t, "o meu\n", "salvar", "--esperado", sha2, "--forcar", "--", arq)
	if b, _ := os.ReadFile(arq); codigo != base.Certo || string(b) != "o meu\n" {
		t.Errorf("forçar: %d %q", codigo, b)
	}
}

func TestSalvarNaoGravaTextoCortadoNemSemSaberOQueEspera(t *testing.T) {
	textos := mundo(t)
	arq := filepath.Join(textos, "notas.md")
	os.WriteFile(arq, []byte("inteiro\n"), 0o644)
	sha := disco.Resumo([]byte("inteiro\n"))
	for nome, args := range map[string][]string{
		"stdin cortado":       {"salvar", "--esperado", sha, "--bytes", "500", arq},
		"sem esperado":        {"salvar", arq},
		"sem arquivo":         {"salvar", "--forcar"},
		"dois arquivos":       {"salvar", "--forcar", arq, arq + "2"},
		"fim de linha torto":  {"salvar", "--forcar", "--fim-de-linha", "cr", arq},
		"opção desconhecida":  {"salvar", "--forcar", "--apagar", arq},
		"bytes não é número":  {"salvar", "--forcar", "--bytes", "muitos", arq},
		"esperado sem valor":  {"salvar", arq, "--esperado"},
		"esperado duas vezes": {"salvar", "--esperado", sha, "--esperado", sha, arq},
	} {
		codigo, r := roda(t, "meio tex", args...)
		if codigo != base.Erro || r["erro"] == nil {
			t.Errorf("%s: devia ser erro (1), veio %d %v", nome, codigo, r)
		}
		if b, _ := os.ReadFile(arq); string(b) != "inteiro\n" {
			t.Fatalf("%s: o arquivo foi trocado: %q", nome, b)
		}
	}
	// Texto vazio só passa quando a janela diz que é vazio de propósito.
	codigo, _ := roda(t, "", "salvar", "--esperado", sha, "--bytes", "0", arq)
	if b, _ := os.ReadFile(arq); codigo != base.Certo || len(b) != 0 {
		t.Errorf("vazio de propósito: %d %q", codigo, b)
	}
}

func TestAbrirRecusaComSaida2(t *testing.T) {
	textos := mundo(t)
	binario := filepath.Join(textos, "a.bin")
	os.WriteFile(binario, []byte{0x7f, 'E', 'L', 'F', 0, 1}, 0o644)
	codigo, r := roda(t, "", "abrir", binario)
	if codigo != base.Recusado || !strings.Contains(r["erro"].(string), "binário") {
		t.Errorf("binário: %d %v", codigo, r)
	}
	t.Setenv("MKESCRITA_LIMITE_MB", "1")
	grande := filepath.Join(textos, "g.md")
	os.WriteFile(grande, bytes.Repeat([]byte("ab"), 1<<20), 0o644)
	codigo, r = roda(t, "", "abrir", grande)
	if codigo != base.Recusado || !strings.Contains(r["erro"].(string), "limite é 1 MB") {
		t.Errorf("grande: %d %v", codigo, r)
	}
	codigo, r = roda(t, "", "abrir")
	if codigo != base.Erro || !strings.HasPrefix(r["erro"].(string), "uso:") {
		t.Errorf("sem arquivo: %d %v", codigo, r)
	}
	codigo, r = roda(t, "", "abrir", filepath.Join(textos, "novo.md"))
	if codigo != base.Certo || r["novo"] != true || r["impressao"] != nil || r["texto"] != "" {
		t.Errorf("arquivo que ainda não existe: %d %v", codigo, r)
	}
}

func TestRascunhosDePontaAPonta(t *testing.T) {
	mundo(t)
	codigo, r := roda(t, "", "rascunho", "listar")
	if codigo != base.Certo || len(r["itens"].([]any)) != 0 {
		t.Fatalf("lista vazia: %d %v", codigo, r)
	}
	texto := "ainda não salvei\nisto\n"
	codigo, r = roda(t, texto, "rascunho", "guardar", "j-1", "--origem", "/textos/carta.md", "--base", "abc", "--bytes", bytesDe(texto))
	if codigo != base.Certo || r["id"] != "j-1" || r["nome"] != "carta.md" || r["trecho"] != "ainda não salvei" || r["tamanho_bytes"] != float64(len(texto)) {
		t.Fatalf("guardar: %d %v", codigo, r)
	}
	if _, tem := r["texto"]; tem {
		t.Error("guardar não devolve o texto de volta")
	}
	roda(t, "sem nome", "rascunho", "guardar", "j-2")
	_, r = roda(t, "", "rascunho", "listar")
	itens := r["itens"].([]any)
	if len(itens) != 2 {
		t.Fatalf("listar: %v", r)
	}
	for _, it := range itens {
		m := it.(map[string]any)
		for _, campo := range []string{"id", "origem", "nome", "guardado_em_s", "tamanho_bytes", "trecho", "vivo"} {
			if _, tem := m[campo]; !tem {
				t.Errorf("item sem %s: %v", campo, m)
			}
		}
		// O /proc do teste está vazio: nenhuma janela viva é dona de nada.
		if m["vivo"] != false {
			t.Errorf("vivo: %v", m)
		}
	}
	codigo, r = roda(t, "", "rascunho", "ler", "j-1")
	if codigo != base.Certo || r["texto"] != texto || r["origem"] != "/textos/carta.md" || r["base_sha256"] != "abc" {
		t.Errorf("ler: %d %v", codigo, r)
	}
	codigo, r = roda(t, "", "rascunho", "apagar", "j-1")
	if codigo != base.Certo || r["ok"] != true {
		t.Errorf("apagar: %d %v", codigo, r)
	}
	if codigo, _ = roda(t, "", "rascunho", "ler", "j-1"); codigo != base.Erro {
		t.Error("ler o apagado devia ser erro")
	}
	// Texto cortado não vira rascunho por cima do bom.
	codigo, _ = roda(t, "se", "rascunho", "guardar", "j-2", "--bytes", "99")
	_, r = roda(t, "", "rascunho", "ler", "j-2")
	if codigo != base.Erro || r["texto"] != "sem nome" {
		t.Errorf("rascunho cortado: %d %v", codigo, r)
	}
	for _, args := range [][]string{{"rascunho"}, {"rascunho", "queimar"}, {"rascunho", "ler"}, {"rascunho", "ler", "../x"}, {"rascunho", "guardar"}, {"rascunho", "apagar", "a", "b"}} {
		if codigo, r := roda(t, "x", args...); codigo != base.Erro || r["erro"] == nil {
			t.Errorf("%v devia ser erro: %d %v", args, codigo, r)
		}
	}
}

func TestHtmlContarEstrutura(t *testing.T) {
	mundo(t)
	md := "# Olá\n\n<script>x</script> [a](javascript:x) **forte**\n\n## Dois\n"
	codigo, r := roda(t, md, "html", "--bytes", bytesDe(md))
	if codigo != base.Certo || r["html"] != "<h1>Olá</h1>\n<p>&lt;script&gt;x&lt;/script&gt; a <strong>forte</strong></p>\n<h2>Dois</h2>\n" {
		t.Errorf("html: %d %q", codigo, r["html"])
	}
	codigo, r = roda(t, md, "html", "--pagina", "--titulo", "Meu título")
	pagina, _ := r["html"].(string)
	if codigo != base.Certo || !strings.HasPrefix(pagina, "<!doctype html>") || !strings.Contains(pagina, "<title>Meu título</title>") || !strings.Contains(pagina, "<h1>Olá</h1>") {
		t.Errorf("página: %d %.200q", codigo, pagina)
	}
	codigo, r = roda(t, md, "contar")
	if codigo != base.Certo || r["palavras"] != float64(5) || r["leitura_s"] != float64(2) || r["linhas"] != float64(5) {
		t.Errorf("contar: %d %v", codigo, r)
	}
	for _, campo := range []string{"palavras", "caracteres", "caracteres_sem_espacos", "linhas", "paragrafos", "leitura_s"} {
		if _, tem := r[campo]; !tem {
			t.Errorf("contar sem %s", campo)
		}
	}
	codigo, r = roda(t, md, "estrutura")
	ts := r["titulos"].([]any)
	if codigo != base.Certo || len(ts) != 2 || ts[1].(map[string]any)["texto"] != "Dois" || ts[1].(map[string]any)["nivel"] != float64(2) || ts[1].(map[string]any)["linha"] != float64(5) {
		t.Errorf("estrutura: %d %v", codigo, r)
	}
	_, r = roda(t, "", "estrutura")
	if ts, ok := r["titulos"].([]any); !ok || len(ts) != 0 {
		t.Errorf("sem títulos é lista vazia: %v", r)
	}
	for _, args := range [][]string{{"html", "arquivo.md"}, {"contar", "x"}, {"estrutura", "--pagina"}, {"html", "--bytes", "3"}} {
		if codigo, r := roda(t, "texto", args...); codigo != base.Erro || r["erro"] == nil {
			t.Errorf("%v devia ser erro: %d %v", args, codigo, r)
		}
	}
}

func TestExportar(t *testing.T) {
	textos := mundo(t)
	saida := filepath.Join(textos, "doc.html")
	codigo, r := roda(t, "# Título\n\ntexto", "exportar", "html", saida)
	if codigo != base.Certo || r["arquivo"] != saida || r["nome"] != "doc.html" {
		t.Fatalf("exportar: %d %v", codigo, r)
	}
	b, _ := os.ReadFile(saida)
	if !strings.HasPrefix(string(b), "<!doctype html>") || !strings.Contains(string(b), "<title>Título</title>") {
		t.Errorf("o exportado: %.120q", b)
	}
	codigo, r = roda(t, "outro", "exportar", "html", saida)
	if codigo != base.Recusado || !strings.Contains(r["erro"].(string), "já existe") {
		t.Errorf("não sobrescreve sem --forcar: %d %v", codigo, r)
	}
	if depois, _ := os.ReadFile(saida); !bytes.Equal(b, depois) {
		t.Error("sobrescreveu sem --forcar")
	}
	codigo, _ = roda(t, "outro", "exportar", "html", "--forcar", "--titulo", "T", saida)
	if depois, _ := os.ReadFile(saida); codigo != base.Certo || !strings.Contains(string(depois), "<p>outro</p>") || !strings.Contains(string(depois), "<title>T</title>") {
		t.Errorf("--forcar: %d", codigo)
	}
	for _, args := range [][]string{{"exportar"}, {"exportar", "pdf", saida}, {"exportar", "html"}, {"exportar", "html", saida, "mais"}} {
		if codigo, r := roda(t, "x", args...); codigo != base.Erro || r["erro"] == nil {
			t.Errorf("%v devia ser erro: %d %v", args, codigo, r)
		}
	}
}

func TestPreferenciasDePontaAPonta(t *testing.T) {
	mundo(t)
	codigo, r := roda(t, "", "preferencias", "ler")
	if codigo != base.Certo || r["fonte"] != "" || r["zoom"] != float64(100) || r["previa"] != false {
		t.Errorf("padrões: %d %v", codigo, r)
	}
	codigo, r = roda(t, `{"fonte":"Iosevka","zoom":900}`, "preferencias", "gravar")
	if codigo != base.Certo || r["fonte"] != "Iosevka" || r["zoom"] != float64(300) {
		t.Errorf("gravar: %d %v", codigo, r)
	}
	_, r = roda(t, "", "preferencias", "ler")
	if r["fonte"] != "Iosevka" || r["zoom"] != float64(300) {
		t.Errorf("ler depois: %v", r)
	}
	if _, err := os.Stat(base.Preferencias()); err != nil {
		t.Errorf("o arquivo de preferências devia estar em %s", base.Preferencias())
	}
	for _, args := range [][]string{{"preferencias"}, {"preferencias", "apagar"}, {"preferencias", "ler", "x"}} {
		if codigo, r := roda(t, "{}", args...); codigo != base.Erro || r["erro"] == nil {
			t.Errorf("%v devia ser erro: %d %v", args, codigo, r)
		}
	}
	if codigo, _ := roda(t, "não é json", "preferencias", "gravar"); codigo != base.Erro {
		t.Error("JSON torto devia ser erro")
	}
}

func TestVigiarValidaOsArgumentos(t *testing.T) {
	mundo(t)
	for _, args := range [][]string{{"vigiar"}, {"vigiar", "a", "b"}, {"vigiar", "--intervalo-ms", "1", "a"}, {"vigiar", "--intervalo-ms", "x", "a"}} {
		if codigo, r := roda(t, "", args...); codigo != base.Erro || r["erro"] == nil {
			t.Errorf("%v devia ser erro: %d %v", args, codigo, r)
		}
	}
}
