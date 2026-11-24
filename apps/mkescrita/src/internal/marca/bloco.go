// Package marca é o Markdown do MkEscrita: um conversor próprio para HTML (só
// com a biblioteca padrão, para o subconjunto que importa ao escrever), a
// contagem de palavras e a lista de títulos. Nada do que vem do texto vira
// marcação por conta própria: HTML digitado sai escapado, e endereço que não
// é http, https, mailto ou relativo não vira link.
package marca

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	reCerca     = regexp.MustCompile("^( {0,3})(`{3,}|~{3,})[ \t]*([^`]*)$")
	reTitulo    = regexp.MustCompile(`^ {0,3}(#{1,6})(?:[ \t]+(.*?))??(?:[ \t]+#+)?[ \t]*$`)
	reRegua     = regexp.MustCompile(`^ {0,3}(?:(?:\*[ \t]*){3,}|(?:-[ \t]*){3,}|(?:_[ \t]*){3,})$`)
	reCitacao   = regexp.MustCompile(`^ {0,3}> ?`)
	reItem      = regexp.MustCompile(`^( *)([-*+]|[0-9]{1,9}[.)])( +|$)`)
	reTarefa    = regexp.MustCompile(`^\[([ xX])\](?: +|$)`)
	reSeparador = regexp.MustCompile(`^ {0,3}\|?[ \t]*:?-+:?[ \t]*(?:\|[ \t]*:?-+:?[ \t]*)*\|?[ \t]*$`)
	reLinguagem = regexp.MustCompile(`^[A-Za-z0-9_+#.-]{1,40}`)
)

// Normaliza deixa o texto do jeito que o conversor espera: quebras "\n", sem
// byte nulo, tabulação do começo da linha como quatro espaços.
func Normaliza(md string) []string {
	md = strings.ReplaceAll(md, "\r\n", "\n")
	md = strings.ReplaceAll(md, "\r", "\n")
	md = strings.ReplaceAll(md, "\x00", "�")
	linhas := strings.Split(md, "\n")
	for i, l := range linhas {
		if strings.HasPrefix(strings.TrimLeft(l, " "), "\t") {
			n := 0
			for n < len(l) && (l[n] == ' ' || l[n] == '\t') {
				n++
			}
			linhas[i] = strings.ReplaceAll(l[:n], "\t", "    ") + l[n:]
		}
	}
	return linhas
}

func vazia(l string) bool { return strings.TrimSpace(l) == "" }

func recuo(l string) int {
	n := 0
	for n < len(l) && l[n] == ' ' {
		n++
	}
	return n
}

// cerca reconhece a linha que abre um bloco de código cercado.
func cerca(l string) (marca string, recuoDaCerca int, linguagem string, ok bool) {
	m := reCerca.FindStringSubmatch(l)
	if m == nil {
		return "", 0, "", false
	}
	return m[2], len(m[1]), reLinguagem.FindString(strings.TrimSpace(m[3])), true
}

// fechaCerca diz se a linha fecha a cerca aberta com `marca`.
func fechaCerca(l, marca string) bool {
	t := strings.TrimSpace(l)
	return recuo(l) <= 3 && len(t) >= len(marca) && strings.Trim(t, marca[:1]) == ""
}

// titulo reconhece "## Título" (o "#" precisa de um espaço depois).
func titulo(l string) (nivel int, texto string, ok bool) {
	m := reTitulo.FindStringSubmatch(l)
	if m == nil {
		return 0, "", false
	}
	resto := strings.TrimLeft(l, " ")[len(m[1]):]
	if resto != "" && resto[0] != ' ' && resto[0] != '\t' {
		return 0, "", false
	}
	return len(m[1]), strings.TrimSpace(m[2]), true
}

// item reconhece o começo de um item de lista.
type marcaDeItem struct {
	recuo    int  // espaços antes da marca
	conteudo int  // coluna em que o texto do item começa
	ordenada bool // "1." em vez de "-"
	numero   int
	resto    string
}

func item(l string) (marcaDeItem, bool) {
	m := reItem.FindStringSubmatch(l)
	if m == nil || reRegua.MatchString(l) {
		return marcaDeItem{}, false
	}
	it := marcaDeItem{recuo: len(m[1])}
	espacos := len(m[3])
	if espacos == 0 || espacos > 4 {
		espacos = 1
	}
	it.conteudo = len(m[1]) + len(m[2]) + espacos
	if len(m[2]) > 1 {
		it.ordenada = true
		it.numero, _ = strconv.Atoi(m[2][:len(m[2])-1])
	}
	if len(l) > len(m[0]) {
		it.resto = l[len(m[1])+len(m[2])+espacos:]
	}
	return it, true
}

// comecaBloco diz se a linha interrompe um parágrafo.
func comecaBloco(l string) bool {
	if _, _, _, ok := cerca(l); ok {
		return true
	}
	if _, _, ok := titulo(l); ok {
		return true
	}
	if reRegua.MatchString(l) || reCitacao.MatchString(l) {
		return true
	}
	// Uma lista só interrompe um parágrafo se o item tem texto e, sendo
	// numerada, começa do 1 ("em 1986. Foi…" não vira lista).
	if it, ok := item(l); ok && it.recuo <= 3 {
		return strings.TrimSpace(it.resto) != "" && (!it.ordenada || it.numero == 1)
	}
	return false
}

// HTML converte um texto em Markdown num fragmento de HTML.
func HTML(md string) string {
	var w strings.Builder
	blocos(&w, Normaliza(md), false, 0)
	return w.String()
}

// blocos converte uma sequência de linhas. `justa` é a lista sem linhas em
// branco entre os itens: os parágrafos dela saem sem <p>.
func blocos(w *strings.Builder, linhas []string, justa bool, fundo int) {
	if fundo > maxFundo {
		w.WriteString("<p>")
		escapa(w, strings.Join(linhas, "\n"))
		w.WriteString("</p>\n")
		return
	}
	for i := 0; i < len(linhas); {
		l := linhas[i]
		if vazia(l) {
			i++
			continue
		}
		if marca, rec, linguagem, ok := cerca(l); ok {
			i++
			w.WriteString("<pre><code")
			if linguagem != "" {
				w.WriteString(` class="language-`)
				escapa(w, linguagem)
				w.WriteString(`"`)
			}
			w.WriteString(">")
			for ; i < len(linhas); i++ {
				if fechaCerca(linhas[i], marca) {
					i++
					break
				}
				c := linhas[i]
				if r := recuo(c); r > rec {
					c = c[rec:]
				} else {
					c = c[r:]
				}
				escapa(w, c)
				w.WriteByte('\n')
			}
			w.WriteString("</code></pre>\n")
			continue
		}
		if nivel, texto, ok := titulo(l); ok {
			n := strconv.Itoa(nivel)
			w.WriteString("<h" + n + ">")
			emLinha(w, texto, 0)
			w.WriteString("</h" + n + ">\n")
			i++
			continue
		}
		if reRegua.MatchString(l) {
			w.WriteString("<hr>\n")
			i++
			continue
		}
		if reCitacao.MatchString(l) {
			var dentro []string
			for ; i < len(linhas); i++ {
				if loc := reCitacao.FindStringIndex(linhas[i]); loc != nil {
					dentro = append(dentro, linhas[i][loc[1]:])
				} else if !vazia(linhas[i]) && !comecaBloco(linhas[i]) && len(dentro) > 0 && !vazia(dentro[len(dentro)-1]) {
					dentro = append(dentro, linhas[i]) // a frase continua sem o ">"
				} else {
					break
				}
			}
			w.WriteString("<blockquote>\n")
			blocos(w, dentro, false, fundo+1)
			w.WriteString("</blockquote>\n")
			continue
		}
		if it, ok := item(l); ok && it.recuo <= 3 {
			i = lista(w, linhas, i, it, fundo)
			continue
		}
		if fim, ok := tabela(w, linhas, i); ok {
			i = fim
			continue
		}
		// Parágrafo: até a linha em branco ou o começo de outro bloco.
		var texto strings.Builder
		for ; i < len(linhas); i++ {
			p := linhas[i]
			if vazia(p) || (texto.Len() > 0 && comecaBloco(p)) {
				break
			}
			if texto.Len() > 0 {
				texto.WriteByte('\n')
			}
			// Dois espaços no fim (ou uma barra) pedem quebra de linha.
			limpa := strings.TrimLeft(p, " ")
			forcada := strings.HasSuffix(limpa, "  ") || (strings.HasSuffix(limpa, "\\") && !strings.HasSuffix(limpa, "\\\\"))
			limpa = strings.TrimRight(limpa, " ")
			if forcada {
				limpa = strings.TrimSuffix(limpa, "\\")
				texto.WriteString(limpa)
				texto.WriteByte(quebra)
			} else {
				texto.WriteString(limpa)
			}
		}
		corpo := strings.TrimRight(texto.String(), string(rune(quebra))+"\n")
		if !justa {
			w.WriteString("<p>")
		}
		emLinha(w, strings.ReplaceAll(corpo, string(rune(quebra))+"\n", string(rune(quebra))), 0)
		if !justa {
			w.WriteString("</p>")
		}
		w.WriteByte('\n')
	}
}

// lista converte uma lista que começa em linhas[i] e devolve onde ela acaba.
func lista(w *strings.Builder, linhas []string, i int, primeiro marcaDeItem, fundo int) int {
	type umItem struct {
		linhas []string
		tarefa string // "", " " ou "x"
	}
	var itens []umItem
	solta := false // há linha em branco entre itens ou dentro de um
	atual := primeiro
	for {
		// O texto do item: a primeira linha e o que vem recuado embaixo dela.
		it := umItem{}
		resto := atual.resto
		if m := reTarefa.FindStringSubmatch(resto); m != nil {
			it.tarefa = strings.ToLower(m[1])
			resto = resto[len(m[0]):]
		}
		it.linhas = append(it.linhas, resto)
		// Quem aninha com dois espaços embaixo de "1. " também é atendido.
		limiar := atual.conteudo
		if atual.recuo+2 < limiar {
			limiar = atual.recuo + 2
		}
		i++
		brancos := 0
		acabou := true
		for ; i < len(linhas); i++ {
			l := linhas[i]
			if vazia(l) {
				brancos++
				continue
			}
			r := recuo(l)
			if r >= limiar {
				// Pertence a este item.
				if brancos > 0 {
					solta = true
					for ; brancos > 0; brancos-- {
						it.linhas = append(it.linhas, "")
					}
				}
				corta := atual.conteudo
				if r < corta {
					corta = r
				}
				it.linhas = append(it.linhas, l[corta:])
				continue
			}
			if prox, ok := item(l); ok && prox.ordenada == primeiro.ordenada {
				// O próximo item da mesma lista.
				if brancos > 0 {
					solta = true
				}
				atual = prox
				acabou = false
				break
			}
			if brancos == 0 && !comecaBloco(l) {
				it.linhas = append(it.linhas, strings.TrimLeft(l, " ")) // a frase continua sem recuo
				continue
			}
			i -= brancos // as linhas em branco não são da lista
			break
		}
		itens = append(itens, it)
		if acabou {
			break
		}
	}
	if primeiro.ordenada {
		if primeiro.numero != 1 {
			w.WriteString(`<ol start="` + strconv.Itoa(primeiro.numero) + `">` + "\n")
		} else {
			w.WriteString("<ol>\n")
		}
	} else {
		w.WriteString("<ul>\n")
	}
	for _, it := range itens {
		switch it.tarefa {
		case " ":
			w.WriteString(`<li class="tarefa">☐ `)
		case "x":
			w.WriteString(`<li class="tarefa feita">☑ `)
		default:
			w.WriteString("<li>")
		}
		var dentro strings.Builder
		blocos(&dentro, it.linhas, !solta, fundo+1)
		w.WriteString(strings.TrimRight(dentro.String(), "\n"))
		w.WriteString("</li>\n")
	}
	if primeiro.ordenada {
		w.WriteString("</ol>\n")
	} else {
		w.WriteString("</ul>\n")
	}
	return i
}

// celulas parte uma linha de tabela nas barras (as escapadas e as de dentro
// de um trecho de código não contam).
func celulas(l string) []string {
	l = strings.TrimSpace(l)
	l = strings.TrimPrefix(l, "|")
	var cs []string
	ini := 0
	for j := 0; j < len(l); {
		if l[j] == '\\' && j+1 < len(l) && l[j+1] == '|' {
			j += 2
			continue
		}
		if l[j] == '`' {
			if _, _, depois, ok := fimDoCodigo(l, j); ok {
				j = depois
				continue
			}
		}
		if l[j] == '|' {
			cs = append(cs, l[ini:j])
			ini = j + 1
		}
		j++
	}
	if ini < len(l) || len(cs) == 0 {
		cs = append(cs, l[ini:])
	}
	for k, c := range cs {
		cs[k] = strings.ReplaceAll(strings.TrimSpace(c), `\|`, "|")
	}
	return cs
}

// tabela converte uma tabela simples (cabeçalho, linha de traços, linhas).
func tabela(w *strings.Builder, linhas []string, i int) (int, bool) {
	if i+1 >= len(linhas) || !strings.Contains(linhas[i], "|") || !reSeparador.MatchString(linhas[i+1]) ||
		!strings.Contains(linhas[i+1], "-") {
		return i, false
	}
	cabeca := celulas(linhas[i])
	tracos := celulas(linhas[i+1])
	if len(cabeca) != len(tracos) {
		return i, false
	}
	lado := make([]string, len(tracos))
	for k, t := range tracos {
		switch {
		case strings.HasPrefix(t, ":") && strings.HasSuffix(t, ":"):
			lado[k] = ` align="center"`
		case strings.HasSuffix(t, ":"):
			lado[k] = ` align="right"`
		case strings.HasPrefix(t, ":"):
			lado[k] = ` align="left"`
		}
	}
	w.WriteString("<table>\n<thead>\n<tr>")
	for k, c := range cabeca {
		w.WriteString("<th" + lado[k] + ">")
		emLinha(w, c, 0)
		w.WriteString("</th>")
	}
	w.WriteString("</tr>\n</thead>\n<tbody>\n")
	i += 2
	for ; i < len(linhas) && !vazia(linhas[i]) && strings.Contains(linhas[i], "|"); i++ {
		w.WriteString("<tr>")
		cs := celulas(linhas[i])
		for k := range cabeca {
			w.WriteString("<td" + lado[k] + ">")
			if k < len(cs) {
				emLinha(w, cs[k], 0)
			}
			w.WriteString("</td>")
		}
		w.WriteString("</tr>\n")
	}
	w.WriteString("</tbody>\n</table>\n")
	return i, true
}
