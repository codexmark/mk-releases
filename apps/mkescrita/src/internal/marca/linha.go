package marca

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// A marca de quebra forçada dentro de um parágrafo (o texto de entrada nunca
// tem byte nulo: Normaliza troca).
const quebra = '\x00'

// escapa escreve um texto como texto de HTML: nada do que a pessoa digitou
// vira marcação.
func escapa(w *strings.Builder, s string) {
	for i := 0; i < len(s); i++ {
		escapaByte(w, s[i])
	}
}

func escapaByte(w *strings.Builder, c byte) {
	switch c {
	case '&':
		w.WriteString("&amp;")
	case '<':
		w.WriteString("&lt;")
	case '>':
		w.WriteString("&gt;")
	case '"':
		w.WriteString("&quot;")
	case '\'':
		w.WriteString("&#39;")
	case quebra:
		w.WriteString("<br>\n")
	default:
		w.WriteByte(c)
	}
}

func pontuacao(c byte) bool {
	return c < utf8.RuneSelf && strings.IndexByte("!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~", c) >= 0
}

func espaco(c byte) bool { return c == ' ' || c == '\n' || c == '\t' || c == quebra }

// letraAntes e letraDepois dizem se há letra ou algarismo colado na posição:
// é o que impede o "_" de nome_de_arquivo de virar itálico.
func letraAntes(s string, i int) bool {
	if i <= 0 {
		return false
	}
	r, _ := utf8.DecodeLastRuneInString(s[:i])
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

func letraDepois(s string, i int) bool {
	if i >= len(s) {
		return false
	}
	r, _ := utf8.DecodeRuneInString(s[i:])
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

// corrida conta quantos `c` seguidos há a partir de i.
func corrida(s string, i int, c byte) int {
	n := 0
	for i+n < len(s) && s[i+n] == c {
		n++
	}
	return n
}

// fimDoCodigo: s[i] é uma crase. Devolve onde começa o conteúdo, onde ele
// acaba e onde o trecho de código termina (ok=false se a corrida não fecha).
func fimDoCodigo(s string, i int) (ini, fim, depois int, ok bool) {
	n := corrida(s, i, '`')
	for j := i + n; j < len(s); {
		if s[j] != '`' {
			j++
			continue
		}
		m := corrida(s, j, '`')
		if m == n {
			return i + n, j, j + m, true
		}
		j += m
	}
	return 0, 0, i + n, false
}

// pula passa por cima do que não pode conter um fechamento: um caractere
// escapado ou um trecho de código. Devolve i se não há o que pular.
func pula(s string, i int) int {
	if s[i] == '\\' && i+1 < len(s) && pontuacao(s[i+1]) {
		return i + 2
	}
	if s[i] == '`' {
		_, _, depois, _ := fimDoCodigo(s, i)
		return depois
	}
	return i
}

// fechaEnfase acha onde fecha uma ênfase de `n` marcas `c` (1 = itálico,
// 2 = negrito) aberta antes de `de`. Devolve o índice da marca que fecha, -1
// se não fecha.
func fechaEnfase(s string, de int, c byte, n int) int {
	for j := de; j < len(s); {
		if k := pula(s, j); k != j {
			j = k
			continue
		}
		if s[j] != c {
			j++
			continue
		}
		m := corrida(s, j, c)
		podeFechar := j > de && !espaco(s[j-1]) && (c != '_' || !letraDepois(s, j+m))
		switch {
		case !podeFechar:
		case n == 1 && m == 1:
			return j
		case n == 1 && m >= 3:
			return j + m - 1 // "*a **b***": o negrito de dentro fecha antes
		case n == 2 && m >= 2:
			return j + m - 2 // "**a *b***": o itálico de dentro fecha antes
		case n == 3 && m >= 3:
			return j
		}
		j += m
	}
	return -1
}

// fechaColchete: s[i] é "[". Devolve o índice do "]" que casa, -1 se não há.
func fechaColchete(s string, i int) int {
	fundo := 0
	for j := i; j < len(s); {
		if k := pula(s, j); k != j {
			j = k
			continue
		}
		switch s[j] {
		case '[':
			fundo++
		case ']':
			fundo--
			if fundo == 0 {
				return j
			}
		}
		j++
	}
	return -1
}

// destino lê "(url "título")" a partir de s[i] == '('. Devolve a url, o
// título e o índice depois do ")".
func destino(s string, i int) (url, titulo string, depois int, ok bool) {
	j := i + 1
	for j < len(s) && espaco(s[j]) {
		j++
	}
	if j < len(s) && s[j] == '<' {
		k := strings.IndexAny(s[j:], ">\n")
		if k < 0 || s[j+k] != '>' {
			return "", "", 0, false
		}
		url = s[j+1 : j+k]
		j += k + 1
	} else {
		ini, fundo := j, 0
		for j < len(s) && !espaco(s[j]) {
			if s[j] == '\\' && j+1 < len(s) && pontuacao(s[j+1]) {
				j += 2
				continue
			}
			if s[j] == '(' {
				fundo++
			}
			if s[j] == ')' {
				if fundo == 0 {
					break
				}
				fundo--
			}
			j++
		}
		url = s[ini:j]
	}
	for j < len(s) && espaco(s[j]) {
		j++
	}
	if j < len(s) && (s[j] == '"' || s[j] == '\'') {
		aspa := s[j]
		k := strings.IndexByte(s[j+1:], aspa)
		if k < 0 {
			return "", "", 0, false
		}
		titulo = s[j+1 : j+1+k]
		j += k + 2
		for j < len(s) && espaco(s[j]) {
			j++
		}
	}
	if j >= len(s) || s[j] != ')' {
		return "", "", 0, false
	}
	return semEscapes(url), semEscapes(titulo), j + 1, true
}

// semEscapes tira a barra de "\)" e companhia.
func semEscapes(s string) string {
	if !strings.Contains(s, "\\") {
		return s
	}
	var w strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) && pontuacao(s[i+1]) {
			i++
		}
		w.WriteByte(s[i])
	}
	return w.String()
}

// esquema devolve o esquema de uma URL ("" se ela é relativa), olhando-a do
// jeito que um navegador olha: sem espaços nem caracteres de controle no
// meio ("java\tscript:" é "javascript:") e em minúsculas.
func esquema(url string) string {
	limpa := strings.Map(func(r rune) rune {
		if r <= ' ' || r == 0x7f || unicode.IsSpace(r) || unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return -1
		}
		return unicode.ToLower(r)
	}, url)
	for i := 0; i < len(limpa); i++ {
		c := limpa[i]
		switch {
		case c == ':':
			if i == 0 {
				return ":"
			}
			return limpa[:i]
		case c >= 'a' && c <= 'z':
		case i > 0 && (c >= '0' && c <= '9' || c == '+' || c == '.' || c == '-'):
		default:
			return ""
		}
	}
	return ""
}

// URLSegura diz se uma URL vinda do texto pode ir para um href (ou src, com
// imagem=true). Só passam as relativas e http, https e (em link) mailto:
// javascript:, data:, vbscript:, file: e o resto ficam de fora.
func URLSegura(url string, imagem bool) bool {
	if strings.TrimSpace(url) == "" {
		return false
	}
	switch esquema(url) {
	case "", "http", "https":
		return true
	case "mailto":
		return !imagem
	}
	return false
}

// atributo escreve uma URL num atributo: escapada e sem espaços soltos.
func atributo(w *strings.Builder, url string) {
	url = strings.TrimSpace(url)
	for i := 0; i < len(url); i++ {
		switch c := url[i]; {
		case c == ' ':
			w.WriteString("%20")
		case c < ' ' || c == 0x7f:
		default:
			escapaByte(w, c)
		}
	}
}

// urlSolta: s[i:] começa com http:// ou https://. Devolve onde a URL acaba
// (sem a pontuação final que é da frase, não do endereço).
func urlSolta(s string, i int) int {
	j := i
	for j < len(s) && !espaco(s[j]) && s[j] != '<' && s[j] != '>' && s[j] != '"' && s[j] != '`' {
		j++
	}
	for j > i {
		c := s[j-1]
		if strings.IndexByte(".,;:!?'*_~", c) >= 0 ||
			(c == ')' && strings.Count(s[i:j], ")") > strings.Count(s[i:j], "(")) {
			j--
			continue
		}
		break
	}
	return j
}

// maxFundo limita a recursão (negrito dentro de link dentro de itálico...).
const maxFundo = 12

// EmLinha converte a marcação que vive dentro de uma linha: ênfase, código,
// links, imagens. Todo o resto sai escapado.
func EmLinha(s string) string {
	var w strings.Builder
	emLinha(&w, s, 0)
	return w.String()
}

func emLinha(w *strings.Builder, s string, fundo int) {
	if fundo > maxFundo {
		escapa(w, s)
		return
	}
	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c == '\\' && i+1 < len(s) && pontuacao(s[i+1]):
			escapaByte(w, s[i+1])
			i += 2
			continue

		case c == '`':
			ini, fim, depois, ok := fimDoCodigo(s, i)
			if ok {
				codigo := strings.Map(func(r rune) rune {
					if r == '\n' || r == quebra {
						return ' '
					}
					return r
				}, s[ini:fim])
				if len(codigo) > 2 && codigo[0] == ' ' && codigo[len(codigo)-1] == ' ' && strings.TrimSpace(codigo) != "" {
					codigo = codigo[1 : len(codigo)-1]
				}
				w.WriteString("<code>")
				escapa(w, codigo)
				w.WriteString("</code>")
			} else {
				escapa(w, s[i:depois])
			}
			i = depois
			continue

		case c == '!' && i+1 < len(s) && s[i+1] == '[', c == '[':
			imagem := c == '!'
			abre := i
			if imagem {
				abre++
			}
			if fecha := fechaColchete(s, abre); fecha > 0 && fecha+1 < len(s) && s[fecha+1] == '(' {
				if url, titulo, depois, ok := destino(s, fecha+1); ok {
					rotulo := s[abre+1 : fecha]
					switch {
					case !URLSegura(url, imagem):
						// Endereço que não passa: fica só o texto, sem link.
						emLinha(w, rotulo, fundo+1)
					case imagem:
						w.WriteString(`<img src="`)
						atributo(w, url)
						w.WriteString(`" alt="`)
						escapa(w, TextoPuro(rotulo))
						w.WriteString(`"`)
						if titulo != "" {
							w.WriteString(` title="`)
							escapa(w, titulo)
							w.WriteString(`"`)
						}
						w.WriteString(">")
					default:
						w.WriteString(`<a href="`)
						atributo(w, url)
						w.WriteString(`"`)
						if titulo != "" {
							w.WriteString(` title="`)
							escapa(w, titulo)
							w.WriteString(`"`)
						}
						w.WriteString(">")
						emLinha(w, rotulo, fundo+1)
						w.WriteString("</a>")
					}
					i = depois
					continue
				}
			}

		case c == '<':
			if k := strings.IndexAny(s[i+1:], "> \n<"); k > 0 && s[i+1+k] == '>' {
				url := s[i+1 : i+1+k]
				e := esquema(url)
				if (e == "http" || e == "https" || e == "mailto") && URLSegura(url, false) {
					w.WriteString(`<a href="`)
					atributo(w, url)
					w.WriteString(`">`)
					escapa(w, strings.TrimPrefix(url, "mailto:"))
					w.WriteString("</a>")
					i += k + 2
					continue
				}
			}

		case (c == 'h') && !letraAntes(s, i) && (strings.HasPrefix(s[i:], "http://") || strings.HasPrefix(s[i:], "https://")):
			if fim := urlSolta(s, i); fim > i+8 {
				w.WriteString(`<a href="`)
				atributo(w, s[i:fim])
				w.WriteString(`">`)
				escapa(w, s[i:fim])
				w.WriteString("</a>")
				i = fim
				continue
			}

		case c == '*' || c == '_':
			n := corrida(s, i, c)
			abre := i+n < len(s) && !espaco(s[i+n]) && (c != '_' || !letraAntes(s, i))
			if abre {
				if n > 3 {
					n = 3
				}
				// Tenta a marca mais forte primeiro e vai afrouxando.
				feito := false
				for m := n; m >= 1 && !feito; m-- {
					fecha := fechaEnfase(s, i+m, c, m)
					if fecha < 0 {
						continue
					}
					abreTag, fechaTag := "<em>", "</em>"
					switch m {
					case 2:
						abreTag, fechaTag = "<strong>", "</strong>"
					case 3:
						abreTag, fechaTag = "<em><strong>", "</strong></em>"
					}
					w.WriteString(abreTag)
					emLinha(w, s[i+m:fecha], fundo+1)
					w.WriteString(fechaTag)
					i = fecha + m
					feito = true
				}
				if feito {
					continue
				}
			}
			// Não abriu nada: a corrida inteira é texto.
			escapa(w, s[i:i+corrida(s, i, c)])
			i += corrida(s, i, c)
			continue

		case c == '~' && corrida(s, i, '~') == 2 && i+2 < len(s) && !espaco(s[i+2]):
			fecha := -1
			for j := i + 2; j < len(s); {
				if k := pula(s, j); k != j {
					j = k
					continue
				}
				if s[j] == '~' {
					m := corrida(s, j, '~')
					if m == 2 && j > i+2 && !espaco(s[j-1]) {
						fecha = j
						break
					}
					j += m
					continue
				}
				j++
			}
			if fecha > 0 {
				w.WriteString("<s>")
				emLinha(w, s[i+2:fecha], fundo+1)
				w.WriteString("</s>")
				i = fecha + 2
				continue
			}
		}
		escapaByte(w, c)
		i++
	}
}

// TextoPuro devolve o que se lê de um trecho em linha, sem a marcação: o
// título "## Um *bom* `dia`" vira "Um bom dia".
func TextoPuro(s string) string {
	html := EmLinha(s)
	var w strings.Builder
	dentro := false
	for i := 0; i < len(html); i++ {
		switch c := html[i]; {
		case c == '<':
			dentro = true
		case c == '>' && dentro:
			dentro = false
		case !dentro:
			w.WriteByte(c)
		}
	}
	r := strings.NewReplacer("&lt;", "<", "&gt;", ">", "&quot;", `"`, "&#39;", "'", "&amp;", "&")
	return strings.Join(strings.Fields(r.Replace(w.String())), " ")
}
