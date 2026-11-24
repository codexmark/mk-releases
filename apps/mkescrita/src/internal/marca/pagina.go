package marca

import "strings"

// O estilo da página exportada: sóbrio, uma coluna, pronto para imprimir.
const estilo = `
:root { color-scheme: light; }
* { box-sizing: border-box; }
body { margin: 0; background: #fbfaf8; color: #1d1b19; font: 17px/1.65 Georgia, "Noto Serif", "Times New Roman", serif; }
main { max-width: 40rem; margin: 0 auto; padding: 4rem 1.5rem 6rem; }
h1, h2, h3, h4, h5, h6 { line-height: 1.25; margin: 2.2em 0 .6em; font-weight: 600; }
h1 { font-size: 2em; margin-top: 0; }
h2 { font-size: 1.5em; }
h3 { font-size: 1.2em; }
h4, h5, h6 { font-size: 1em; }
p, ul, ol, blockquote, pre, table { margin: 0 0 1.2em; }
a { color: #8a3d1c; text-underline-offset: .15em; }
img { max-width: 100%; height: auto; }
hr { border: 0; border-top: 1px solid #cfc8bf; margin: 2.5em 0; }
blockquote { margin-left: 0; padding-left: 1.1em; border-left: 3px solid #cfc8bf; color: #55504a; }
code, pre { font-family: "JetBrains Mono", "DejaVu Sans Mono", Consolas, monospace; font-size: .86em; }
code { background: #efebe5; padding: .1em .3em; border-radius: 3px; }
pre { background: #efebe5; padding: 1em 1.2em; overflow-x: auto; border-radius: 4px; line-height: 1.5; }
pre code { background: none; padding: 0; font-size: 1em; }
ul, ol { padding-left: 1.6em; }
li { margin: .25em 0; }
li.tarefa { list-style: none; margin-left: -1.4em; }
li.feita { color: #77716a; }
table { border-collapse: collapse; width: 100%; font-size: .95em; }
th, td { border-bottom: 1px solid #cfc8bf; padding: .45em .7em; text-align: left; vertical-align: top; }
th { border-bottom-width: 2px; }
@media print {
  body { background: none; font-size: 11.5pt; }
  main { max-width: none; padding: 0; }
  a { color: inherit; }
  pre, blockquote, table, img { break-inside: avoid; }
  h1, h2, h3, h4 { break-after: avoid; }
}
`

// Pagina converte o texto num documento HTML completo, com o estilo embutido
// (nada vem de fora: abre sem rede e imprime igual). Sem `tituloDado`, o
// título é o primeiro título do texto.
func Pagina(md, tituloDado string) string {
	t := strings.TrimSpace(tituloDado)
	if t == "" {
		if ts := Estrutura(md); len(ts) > 0 {
			t = ts[0].Texto
		}
	}
	if t == "" {
		t = "Documento"
	}
	var w strings.Builder
	w.WriteString("<!doctype html>\n<html lang=\"pt-BR\">\n<head>\n<meta charset=\"utf-8\">\n")
	w.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n<title>")
	escapa(&w, t)
	w.WriteString("</title>\n<style>")
	w.WriteString(estilo)
	w.WriteString("</style>\n</head>\n<body>\n<main>\n")
	w.WriteString(HTML(md))
	w.WriteString("</main>\n</body>\n</html>\n")
	return w.String()
}
