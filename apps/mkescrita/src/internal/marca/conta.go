package marca

import (
	"strings"
	"unicode"
)

// PalavrasPorMinuto é o ritmo de leitura usado no tempo de leitura.
const PalavrasPorMinuto = 200

// Contagem é a resposta de `contar`.
type Contagem struct {
	Palavras   int `json:"palavras"`
	Caracteres int `json:"caracteres"`
	// SemEspacos: os caracteres que não são espaço nem quebra de linha.
	SemEspacos int `json:"caracteres_sem_espacos"`
	Linhas     int `json:"linhas"`
	Paragrafos int `json:"paragrafos"`
	// Leitura: o tempo de leitura, em segundos, a 200 palavras por minuto.
	Leitura int `json:"leitura_s"`
}

// Contar conta o que a pessoa escreveu. Palavra é um trecho entre espaços que
// tem ao menos uma letra ou um algarismo: "#", "-" e "**" soltos não contam.
func Contar(texto string) Contagem {
	texto = strings.ReplaceAll(texto, "\r\n", "\n")
	var c Contagem
	for _, pedaco := range strings.Fields(texto) {
		if strings.IndexFunc(pedaco, func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }) >= 0 {
			c.Palavras++
		}
	}
	for _, r := range texto {
		c.Caracteres++
		if !unicode.IsSpace(r) {
			c.SemEspacos++
		}
	}
	if texto != "" {
		c.Linhas = strings.Count(texto, "\n") + 1
		if strings.HasSuffix(texto, "\n") {
			c.Linhas--
		}
	}
	dentro := false
	for _, l := range strings.Split(texto, "\n") {
		if strings.TrimSpace(l) == "" {
			dentro = false
		} else if !dentro {
			dentro = true
			c.Paragrafos++
		}
	}
	c.Leitura = (c.Palavras*60 + PalavrasPorMinuto - 1) / PalavrasPorMinuto
	return c
}

// Titulo é um título do texto, para o sumário.
type Titulo struct {
	Nivel int    `json:"nivel"`
	Texto string `json:"texto"`
	// Linha: a linha do texto em que ele está, contando do 1.
	Linha int `json:"linha"`
}

// Estrutura lista os títulos ("# …") do texto, na ordem. O que está dentro de
// um bloco de código cercado não é título.
func Estrutura(md string) []Titulo {
	titulos := []Titulo{}
	aberta := ""
	for n, l := range Normaliza(md) {
		if aberta != "" {
			if fechaCerca(l, aberta) {
				aberta = ""
			}
			continue
		}
		if marca, _, _, ok := cerca(l); ok {
			aberta = marca
			continue
		}
		if nivel, texto, ok := titulo(l); ok {
			titulos = append(titulos, Titulo{Nivel: nivel, Texto: TextoPuro(texto), Linha: n + 1})
		}
	}
	return titulos
}
