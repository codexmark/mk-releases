package marca

import (
	"strings"
	"testing"
)

type caso struct{ md, quer string }

func confere(t *testing.T, casos []caso) {
	t.Helper()
	for _, c := range casos {
		if g := HTML(c.md); g != c.quer {
			t.Errorf("HTML(%q)\n  obtido:   %q\n  esperado: %q", c.md, g, c.quer)
		}
	}
}

func TestTitulosEParagrafos(t *testing.T) {
	confere(t, []caso{
		{"", ""},
		{"\n\n", ""},
		{"olá", "<p>olá</p>\n"},
		{"uma\nduas\n\noutro", "<p>uma\nduas</p>\n<p>outro</p>\n"},
		{"# Um", "<h1>Um</h1>\n"},
		{"###### Seis ######", "<h6>Seis</h6>\n"},
		{"## Com *ênfase*", "<h2>Com <em>ênfase</em></h2>\n"},
		{"####### sete não é título", "<p>####### sete não é título</p>\n"},
		{"#semespaço", "<p>#semespaço</p>\n"},
		{"#", "<h1></h1>\n"},
		{"texto\n# Título\nmais", "<p>texto</p>\n<h1>Título</h1>\n<p>mais</p>\n"},
		{"a\r\nb\r\n", "<p>a\nb</p>\n"},
	})
}

func TestQuebrasDeLinha(t *testing.T) {
	confere(t, []caso{
		{"um  \ndois", "<p>um<br>\ndois</p>\n"},
		{"um\\\ndois", "<p>um<br>\ndois</p>\n"},
		{"um \ndois", "<p>um\ndois</p>\n"},
		{"fim com espaços  ", "<p>fim com espaços</p>\n"},
		{"**negrito que  \natravessa**", "<p><strong>negrito que<br>\natravessa</strong></p>\n"},
	})
}

func TestEnfase(t *testing.T) {
	confere(t, []caso{
		{"**negrito**", "<p><strong>negrito</strong></p>\n"},
		{"__negrito__", "<p><strong>negrito</strong></p>\n"},
		{"*itálico*", "<p><em>itálico</em></p>\n"},
		{"_itálico_", "<p><em>itálico</em></p>\n"},
		{"~~tachado~~", "<p><s>tachado</s></p>\n"},
		{"***os dois***", "<p><em><strong>os dois</strong></em></p>\n"},
		{"**a *b* c**", "<p><strong>a <em>b</em> c</strong></p>\n"},
		{"*a **b** c*", "<p><em>a <strong>b</strong> c</em></p>\n"},
		{"**a *b***", "<p><strong>a <em>b</em></strong></p>\n"},
		{"*a **b***", "<p><em>a <strong>b</strong></em></p>\n"},
		{"~~a **b**~~", "<p><s>a <strong>b</strong></s></p>\n"},
		// O que não é ênfase fica como está.
		{"nome_de_arquivo e outro_nome", "<p>nome_de_arquivo e outro_nome</p>\n"},
		{"2 * 3 * 4", "<p>2 * 3 * 4</p>\n"},
		{"**sem fechar", "<p>**sem fechar</p>\n"},
		{"*sem fechar", "<p>*sem fechar</p>\n"},
		{"** não abre **", "<p>** não abre **</p>\n"},
		{"um ~ til ~~ dois", "<p>um ~ til ~~ dois</p>\n"},
		{"\\*escapado\\*", "<p>*escapado*</p>\n"},
		{"\\# não é título", "<p># não é título</p>\n"},
		{"ação_ção", "<p>ação_ção</p>\n"},
	})
}

func TestCodigo(t *testing.T) {
	confere(t, []caso{
		{"`a < b`", "<p><code>a &lt; b</code></p>\n"},
		{"`` tem ` crase ``", "<p><code>tem ` crase</code></p>\n"},
		{"`**não é negrito**`", "<p><code>**não é negrito**</code></p>\n"},
		{"`sem fechar", "<p>`sem fechar</p>\n"},
		{"*a `*` b*", "<p><em>a <code>*</code> b</em></p>\n"},
		{"```\n<b>cru</b>\n  recuado\n```", "<pre><code>&lt;b&gt;cru&lt;/b&gt;\n  recuado\n</code></pre>\n"},
		{"```go\nx := 1\n```\ndepois", "<pre><code class=\"language-go\">x := 1\n</code></pre>\n<p>depois</p>\n"},
		{"~~~ c++ titulo\n# não é título\n~~~", "<pre><code class=\"language-c++\"># não é título\n</code></pre>\n"},
		{"```\nsem fechar\n\n- fica código", "<pre><code>sem fechar\n\n- fica código\n</code></pre>\n"},
		{"````\n```\ndentro\n```\n````", "<pre><code>```\ndentro\n```\n</code></pre>\n"},
		// A linguagem vem do texto: nada além de letras, números e + # . - _ entra na classe.
		{"```\"><script>x\ny\n```", "<pre><code>y\n</code></pre>\n"},
		{"```js\"onload=\"x\ny\n```", "<pre><code class=\"language-js\">y\n</code></pre>\n"},
	})
}

func TestCitacoes(t *testing.T) {
	confere(t, []caso{
		{"> uma\n> duas", "<blockquote>\n<p>uma\nduas</p>\n</blockquote>\n"},
		{"> uma\ncontinua", "<blockquote>\n<p>uma\ncontinua</p>\n</blockquote>\n"},
		{">colada", "<blockquote>\n<p>colada</p>\n</blockquote>\n"},
		{"> # Título\n>\n> - item", "<blockquote>\n<h1>Título</h1>\n<ul>\n<li>item</li>\n</ul>\n</blockquote>\n"},
		{"> fora\n> > dentro", "<blockquote>\n<p>fora</p>\n<blockquote>\n<p>dentro</p>\n</blockquote>\n</blockquote>\n"},
		{"> a\n\nb", "<blockquote>\n<p>a</p>\n</blockquote>\n<p>b</p>\n"},
	})
}

func TestListas(t *testing.T) {
	confere(t, []caso{
		{"- a\n- b", "<ul>\n<li>a</li>\n<li>b</li>\n</ul>\n"},
		{"* a\n+ b", "<ul>\n<li>a</li>\n<li>b</li>\n</ul>\n"},
		{"1. a\n2. b", "<ol>\n<li>a</li>\n<li>b</li>\n</ol>\n"},
		{"3) a\n4) b", "<ol start=\"3\">\n<li>a</li>\n<li>b</li>\n</ol>\n"},
		// Aninhadas, com dois e com quatro espaços, e misturando tipos.
		{"- a\n  - b\n    - c\n- d", "<ul>\n<li>a\n<ul>\n<li>b\n<ul>\n<li>c</li>\n</ul></li>\n</ul></li>\n<li>d</li>\n</ul>\n"},
		{"- a\n    - b", "<ul>\n<li>a\n<ul>\n<li>b</li>\n</ul></li>\n</ul>\n"},
		{"1. a\n   - b\n2. c", "<ol>\n<li>a\n<ul>\n<li>b</li>\n</ul></li>\n<li>c</li>\n</ol>\n"},
		{"1. a\n  - b", "<ol>\n<li>a\n<ul>\n<li>b</li>\n</ul></li>\n</ol>\n"},
		{"- a\n  1. b\n  2. c", "<ul>\n<li>a\n<ol>\n<li>b</li>\n<li>c</li>\n</ol></li>\n</ul>\n"},
		// De tarefas.
		{"- [ ] fazer\n- [x] feito\n- [X] também\n- [] não é tarefa",
			"<ul>\n<li class=\"tarefa\">☐ fazer</li>\n<li class=\"tarefa feita\">☑ feito</li>\n<li class=\"tarefa feita\">☑ também</li>\n<li>[] não é tarefa</li>\n</ul>\n"},
		{"- [ ] a\n  - [x] b", "<ul>\n<li class=\"tarefa\">☐ a\n<ul>\n<li class=\"tarefa feita\">☑ b</li>\n</ul></li>\n</ul>\n"},
		// Com linha em branco entre os itens, cada um ganha parágrafo.
		{"- a\n\n- b", "<ul>\n<li><p>a</p></li>\n<li><p>b</p></li>\n</ul>\n"},
		{"- a\n\n  segundo parágrafo\n- b", "<ul>\n<li><p>a</p>\n<p>segundo parágrafo</p></li>\n<li><p>b</p></li>\n</ul>\n"},
		// A frase do item continua na linha de baixo, com ou sem recuo.
		{"- a\n  continua\n- b", "<ul>\n<li>a\ncontinua</li>\n<li>b</li>\n</ul>\n"},
		{"- a\ncontinua", "<ul>\n<li>a\ncontinua</li>\n</ul>\n"},
		// Onde a lista acaba.
		{"- a\n\nparágrafo", "<ul>\n<li>a</li>\n</ul>\n<p>parágrafo</p>\n"},
		{"- a\n# Título", "<ul>\n<li>a</li>\n</ul>\n<h1>Título</h1>\n"},
		{"- a\n1. b", "<ul>\n<li>a</li>\n</ul>\n<ol>\n<li>b</li>\n</ol>\n"},
		{"texto\n- a", "<p>texto</p>\n<ul>\n<li>a</li>\n</ul>\n"},
		// O que parece lista e não é.
		{"em\n1986. Foi um ano", "<p>em\n1986. Foi um ano</p>\n"},
		{"-sem espaço", "<p>-sem espaço</p>\n"},
		{"- **negrito** no item", "<ul>\n<li><strong>negrito</strong> no item</li>\n</ul>\n"},
		{"-", "<ul>\n<li></li>\n</ul>\n"},
	})
}

func TestLinhasHorizontais(t *testing.T) {
	confere(t, []caso{
		{"---", "<hr>\n"},
		{"***", "<hr>\n"},
		{"___", "<hr>\n"},
		{"- - -", "<hr>\n"},
		{"a\n\n-----\n\nb", "<p>a</p>\n<hr>\n<p>b</p>\n"},
		{"--", "<p>--</p>\n"},
	})
}

func TestTabelas(t *testing.T) {
	confere(t, []caso{
		{"| a | b |\n|---|---|\n| 1 | 2 |",
			"<table>\n<thead>\n<tr><th>a</th><th>b</th></tr>\n</thead>\n<tbody>\n<tr><td>1</td><td>2</td></tr>\n</tbody>\n</table>\n"},
		{"a | b\n:-- | :-:\n**1** | 2\n\ndepois",
			"<table>\n<thead>\n<tr><th align=\"left\">a</th><th align=\"center\">b</th></tr>\n</thead>\n<tbody>\n<tr><td align=\"left\"><strong>1</strong></td><td align=\"center\">2</td></tr>\n</tbody>\n</table>\n<p>depois</p>\n"},
		// Linha com células a menos ou a mais: a tabela tem as colunas do cabeçalho.
		{"|a|b|\n|-|-|\n|1|\n|1|2|3|",
			"<table>\n<thead>\n<tr><th>a</th><th>b</th></tr>\n</thead>\n<tbody>\n<tr><td>1</td><td></td></tr>\n<tr><td>1</td><td>2</td></tr>\n</tbody>\n</table>\n"},
		{"|a|\n|-|\n|x \\| y|", "<table>\n<thead>\n<tr><th>a</th></tr>\n</thead>\n<tbody>\n<tr><td>x | y</td></tr>\n</tbody>\n</table>\n"},
		{"|a|\n|-|\n|<b>x</b>|", "<table>\n<thead>\n<tr><th>a</th></tr>\n</thead>\n<tbody>\n<tr><td>&lt;b&gt;x&lt;/b&gt;</td></tr>\n</tbody>\n</table>\n"},
		// Sem a linha de traços, ou com outra quantidade de colunas, não é tabela.
		{"a | b\nc | d", "<p>a | b\nc | d</p>\n"},
		{"a | b\n---", "<p>a | b</p>\n<hr>\n"},
	})
}

func TestLinksEImagens(t *testing.T) {
	confere(t, []caso{
		{"[texto](https://exemplo.org)", "<p><a href=\"https://exemplo.org\">texto</a></p>\n"},
		{"[t](https://e.org/a?b=1&c=2 \"um título\")", "<p><a href=\"https://e.org/a?b=1&amp;c=2\" title=\"um título\">t</a></p>\n"},
		{"[**forte**](pasta/arquivo.md)", "<p><a href=\"pasta/arquivo.md\"><strong>forte</strong></a></p>\n"},
		{"[âncora](#secao) e [mail](mailto:a@b.org)", "<p><a href=\"#secao\">âncora</a> e <a href=\"mailto:a@b.org\">mail</a></p>\n"},
		{"[t](<com espaço.md>)", "<p><a href=\"com%20espaço.md\">t</a></p>\n"},
		{"[t](https://e.org/a_(b))", "<p><a href=\"https://e.org/a_(b)\">t</a></p>\n"},
		{"![uma *foto*](img/foto.png \"legenda\")", "<p><img src=\"img/foto.png\" alt=\"uma foto\" title=\"legenda\"></p>\n"},
		{"[![alt](a.png)](https://e.org)", "<p><a href=\"https://e.org\"><img src=\"a.png\" alt=\"alt\"></a></p>\n"},
		{"<https://e.org/x>", "<p><a href=\"https://e.org/x\">https://e.org/x</a></p>\n"},
		{"<mailto:a@b.org>", "<p><a href=\"mailto:a@b.org\">a@b.org</a></p>\n"},
		{"veja https://e.org/a_b. E (https://e.org/c).", "<p>veja <a href=\"https://e.org/a_b\">https://e.org/a_b</a>. E (<a href=\"https://e.org/c\">https://e.org/c</a>).</p>\n"},
		// O que não fecha fica como texto.
		{"[texto](sem fechar", "<p>[texto](sem fechar</p>\n"},
		{"[só colchetes]", "<p>[só colchetes]</p>\n"},
		{"a < b > c", "<p>a &lt; b &gt; c</p>\n"},
	})
}

// Tudo que vem do texto sai escapado: nenhum caso aqui pode produzir uma tag
// ou um atributo que a pessoa escreveu.
func TestNadaDoTextoViraMarcacao(t *testing.T) {
	confere(t, []caso{
		{"<script>alert(1)</script>", "<p>&lt;script&gt;alert(1)&lt;/script&gt;</p>\n"},
		{"# <img src=x onerror=alert(1)>", "<h1>&lt;img src=x onerror=alert(1)&gt;</h1>\n"},
		{"- <b>negrito</b>", "<ul>\n<li>&lt;b&gt;negrito&lt;/b&gt;</li>\n</ul>\n"},
		{"> </blockquote><script>", "<blockquote>\n<p>&lt;/blockquote&gt;&lt;script&gt;</p>\n</blockquote>\n"},
		{"a & b &amp; \"c\" 'd'", "<p>a &amp; b &amp;amp; &quot;c&quot; &#39;d&#39;</p>\n"},
		{"<!-- comentário -->", "<p>&lt;!-- comentário --&gt;</p>\n"},
		// Links com endereço perigoso: fica só o texto.
		{"[clique](javascript:alert(1))", "<p>clique</p>\n"},
		{"[clique](JaVaScRiPt:alert(1))", "<p>clique</p>\n"},
		{"[clique](<java\tscript:alert(1)>)", "<p>clique</p>\n"},
		{"[clique](<  javascript:alert(1)>)", "<p>clique</p>\n"},
		{"[clique](data:text/html;base64,PHNjcmlwdD4=)", "<p>clique</p>\n"},
		{"[clique](vbscript:x) [arquivo](file:///etc/passwd)", "<p>clique arquivo</p>\n"},
		{"![x](javascript:alert(1))", "<p>x</p>\n"},
		{"![x](data:image/svg+xml,<svg/onload=alert(1)>)", "<p>x</p>\n"},
		{"<javascript:alert(1)>", "<p>&lt;javascript:alert(1)&gt;</p>\n"},
		// Aspas no endereço e no título não fecham o atributo.
		{"[t](https://e.org/\"onmouseover=\"x)", "<p><a href=\"https://e.org/&quot;onmouseover=&quot;x\">t</a></p>\n"},
		{"[t](a.md 'x\" onclick=\"y')", "<p><a href=\"a.md\" title=\"x&quot; onclick=&quot;y\">t</a></p>\n"},
		{"![a\" onerror=\"x](a.png)", "<p><img src=\"a.png\" alt=\"a&quot; onerror=&quot;x\"></p>\n"},
		{"[t](&#106;avascript:x)", "<p><a href=\"&amp;#106;avascript:x\">t</a></p>\n"},
	})
	// De ponta a ponta: num texto cheio de tentativas, nenhuma tag fora das do conversor.
	saida := HTML("# <script>\n\n| <i> |\n|---|\n| <u> |\n\n```<x>\n</code></pre><script>\n```\n\n1. <ol>\n\n[<b>](http://a/<i>)\n")
	permitidas := map[string]bool{"h1": true, "p": true, "table": true, "thead": true, "tbody": true, "tr": true, "th": true, "td": true,
		"pre": true, "code": true, "ol": true, "li": true, "a": true}
	for _, pedaco := range strings.Split(saida, "<")[1:] {
		nome := strings.TrimPrefix(pedaco, "/")
		nome = nome[:strings.IndexAny(nome+" ", " >")]
		if !permitidas[nome] {
			t.Errorf("tag que o conversor não escreve: <%s em %q", nome, saida)
		}
	}
}

func TestURLSegura(t *testing.T) {
	for url, quer := range map[string]bool{
		"https://e.org": true, "http://e.org": true, "HTTPS://E.ORG": true, "mailto:a@b": true,
		"arquivo.md": true, "../pasta/arquivo.md": true, "/abs/x.md": true, "#ancora": true, "?q=1": true, "//e.org/x": true,
		"pasta/a:b.md": true,
		"":             false, "   ": false, "javascript:x": false, "java\nscript:x": false, "\x01javascript:x": false,
		"java​script:x": false, "data:text/html,x": false, "vbscript:x": false, "file:///x": false, ":x": false, "ftp://x": false,
	} {
		if g := URLSegura(url, false); g != quer {
			t.Errorf("URLSegura(%q) = %v, queria %v", url, g, quer)
		}
	}
	if URLSegura("mailto:a@b", true) {
		t.Error("mailto não é endereço de imagem")
	}
}

func TestEntradaEstranhaNaoDerruba(t *testing.T) {
	for _, md := range []string{
		strings.Repeat("*", 5000), strings.Repeat("[", 5000), strings.Repeat("> ", 500) + "x", strings.Repeat("  ", 300) + "- x",
		strings.Repeat("- a\n  ", 200), strings.Repeat("**a ", 2000), strings.Repeat("`", 999), "\x00\x00", "|\n|-|", "|-|\n|-|",
		strings.Repeat("[a](", 300), "![](", "[]()", "*_*_*_", "~~~~", "\\", "```", "1.", "- [ ]", "<", "\xff\xfe",
	} {
		_ = HTML(md)
		_ = Estrutura(md)
		_ = Contar(md)
		_ = Pagina(md, "")
	}
	if g := HTML("a\x00b"); strings.Contains(g, "\x00") || strings.Contains(g, "<br>") {
		t.Errorf("byte nulo do texto não pode virar quebra: %q", g)
	}
}

func TestContar(t *testing.T) {
	c := Contar("# Título\n\nUma frase com **cinco** palavras.\n- item\n")
	if c.Palavras != 7 || c.Linhas != 4 || c.Paragrafos != 2 {
		t.Errorf("contagem: %+v", c)
	}
	c = Contar("ação é")
	if c.Palavras != 2 || c.Caracteres != 6 || c.SemEspacos != 5 || c.Linhas != 1 || c.Leitura != 1 {
		t.Errorf("acentos contam como uma letra: %+v", c)
	}
	if c := Contar(""); c != (Contagem{}) {
		t.Errorf("vazio: %+v", c)
	}
	if c := Contar("- [ ] ** --- #"); c.Palavras != 0 {
		t.Errorf("marcação solta não é palavra: %+v", c)
	}
	if c := Contar("a\r\nb\r\n"); c.Caracteres != 4 || c.Linhas != 2 {
		t.Errorf("CRLF conta como uma quebra: %+v", c)
	}
	for palavras, quer := range map[int]int{1: 1, 200: 60, 201: 61, 400: 120, 1000: 300} {
		if g := Contar(strings.Repeat("palavra ", palavras)).Leitura; g != quer {
			t.Errorf("leitura de %d palavras: %d s, queria %d", palavras, g, quer)
		}
	}
}

func TestEstrutura(t *testing.T) {
	ts := Estrutura("# Um\ntexto\n\n## Dois *com* `marca` [e link](x.md)\n```\n# não é título\n```\n### Três ###\n#semespaco\n~~~\n# nem este\n")
	quer := []Titulo{{1, "Um", 1}, {2, "Dois com marca e link", 4}, {3, "Três", 8}}
	if len(ts) != len(quer) {
		t.Fatalf("títulos: %+v", ts)
	}
	for i := range quer {
		if ts[i] != quer[i] {
			t.Errorf("título %d: %+v, queria %+v", i, ts[i], quer[i])
		}
	}
	if ts := Estrutura("sem títulos"); ts == nil || len(ts) != 0 {
		t.Errorf("sem títulos: lista vazia, não nula: %#v", ts)
	}
	if ts := Estrutura("# a < b & c"); ts[0].Texto != "a < b & c" {
		t.Errorf("o texto do título vem sem escape: %q", ts[0].Texto)
	}
}

func TestPagina(t *testing.T) {
	p := Pagina("# Meu <texto>\n\nolá", "")
	for _, trecho := range []string{"<!doctype html>", `<html lang="pt-BR">`, `<meta charset="utf-8">`, "<title>Meu &lt;texto&gt;</title>",
		"<style>", "@media print", "<main>\n<h1>Meu &lt;texto&gt;</h1>\n<p>olá</p>\n</main>", "</html>\n"} {
		if !strings.Contains(p, trecho) {
			t.Errorf("a página não tem %q", trecho)
		}
	}
	// Nada vem de fora: sem script, sem folha de estilo nem fonte da rede.
	for _, trecho := range []string{"<script", "<link", "@import", "url(", "http://", "https://"} {
		if strings.Contains(p, trecho) {
			t.Errorf("a página não devia ter %q", trecho)
		}
	}
	if p := Pagina("sem título", "  Dado </title><script> "); !strings.Contains(p, "<title>Dado &lt;/title&gt;&lt;script&gt;</title>") {
		t.Errorf("título dado: %q", p[:200])
	}
	if p := Pagina("sem título", ""); !strings.Contains(p, "<title>Documento</title>") {
		t.Error("sem título nenhum: Documento")
	}
}

func TestTextoPuro(t *testing.T) {
	for md, quer := range map[string]string{
		"Um *bom* `dia`": "Um bom dia", "[link](x.md) e ![img](y.png)": "link e", "a  <  b": "a < b", "": "",
	} {
		if g := TextoPuro(md); g != quer {
			t.Errorf("TextoPuro(%q) = %q, queria %q", md, g, quer)
		}
	}
}
