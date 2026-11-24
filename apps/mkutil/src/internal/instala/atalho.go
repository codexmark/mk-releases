package instala

import (
	"errors"
	"regexp"
	"sort"
	"strings"
)

// O atalho de menu (.desktop) que o MkInstalar grava. O que vem de dentro de
// um AppImage ou de um arquivo compactado é texto de quem fez o app, então só
// passam as chaves que descrevem (nome, comentário, categorias…): nada que
// diga o que executar. O Exec é sempre o caminho que o MkInstalar escolheu.

// As chaves do atalho original que passam para o novo.
var chavesQuePassam = map[string]bool{
	"Name": true, "GenericName": true, "Comment": true, "Categories": true, "MimeType": true, "Keywords": true,
	"StartupWMClass": true, "Terminal": true, "StartupNotify": true,
}

// As que têm tradução (Name[pt_BR]).
var chavesTraduzidas = map[string]bool{"Name": true, "GenericName": true, "Comment": true, "Keywords": true}

var chaveDeDesktop = regexp.MustCompile(`^([A-Za-z0-9-]+)(\[[A-Za-z0-9_@.-]+\])?$`)

// leDesktop devolve as chaves do grupo [Desktop Entry] de um atalho.
func leDesktop(b []byte) map[string]string {
	out := map[string]string{}
	dentro := false
	for _, linha := range strings.Split(string(b), "\n") {
		linha = strings.TrimRight(linha, "\r")
		aparada := strings.TrimSpace(linha)
		if aparada == "" || strings.HasPrefix(aparada, "#") {
			continue
		}
		if strings.HasPrefix(aparada, "[") {
			dentro = aparada == "[Desktop Entry]"
			continue
		}
		if !dentro {
			continue
		}
		chave, valor, ok := strings.Cut(linha, "=")
		if !ok {
			continue
		}
		chave = strings.TrimSpace(chave)
		if _, repetida := out[chave]; !repetida && chaveDeDesktop.MatchString(chave) {
			out[chave] = limpaValor(valor)
		}
	}
	return out
}

// limpaValor tira os caracteres de controle e corta um valor sem fim.
func limpaValor(v string) string {
	v = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, strings.TrimSpace(v))
	if letras := []rune(v); len(letras) > 1000 {
		v = string(letras[:1000])
	}
	return v
}

// argumentosDoExec devolve o que vem depois do programa num Exec ("krita %U"
// → "%U"): os argumentos e os códigos de campo de quem fez o app valem para o
// mesmo programa, no caminho novo.
func argumentosDoExec(exec string) string {
	exec = strings.TrimSpace(exec)
	if strings.HasPrefix(exec, `"`) {
		for i := 1; i < len(exec); i++ {
			if exec[i] == '\\' {
				i++
			} else if exec[i] == '"' {
				return strings.TrimSpace(exec[i+1:])
			}
		}
		return ""
	}
	_, resto, _ := strings.Cut(exec, " ")
	return strings.TrimSpace(resto)
}

// citaExec escreve um caminho como o programa de um Exec: entre aspas, com as
// duas camadas de escape da especificação (a do argumento entre aspas e a do
// valor de texto) e o % dobrado.
func citaExec(caminho string) (string, error) {
	if strings.ContainsAny(caminho, "\n\r\x00") {
		return "", errors.New("o caminho tem quebra de linha: não dá para pôr num atalho")
	}
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range caminho {
		switch r {
		case '"', '`', '$':
			b.WriteString(`\\`)
			b.WriteRune(r)
		case '\\':
			b.WriteString(`\\\\`)
		case '%':
			b.WriteString("%%")
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String(), nil
}

// atalhoNovo descreve o atalho a gravar.
type atalhoNovo struct {
	ID         string
	Nome       string            // vale se o original não tiver Name
	Executavel string            // caminho absoluto
	Icone      string            // caminho absoluto ou nome de ícone do tema
	Original   map[string]string // as chaves do atalho de dentro (pode ser nil)
}

// fazAtalho escreve o conteúdo do .desktop.
func fazAtalho(a atalhoNovo) ([]byte, error) {
	prog, err := citaExec(a.Executavel)
	if err != nil {
		return nil, err
	}
	if strings.ContainsAny(a.Icone, "\n\r\x00") {
		return nil, errors.New("o caminho do ícone tem quebra de linha")
	}
	var b strings.Builder
	b.WriteString("[Desktop Entry]\nType=Application\n")
	nome := limpaValor(a.Original["Name"])
	if nome == "" {
		nome = limpaValor(a.Nome)
	}
	if nome == "" {
		nome = a.ID
	}
	b.WriteString("Name=" + nome + "\n")
	chaves := make([]string, 0, len(a.Original))
	for k := range a.Original {
		chaves = append(chaves, k)
	}
	sort.Strings(chaves)
	for _, k := range chaves {
		m := chaveDeDesktop.FindStringSubmatch(k)
		if m == nil || k == "Name" || !chavesQuePassam[m[1]] || m[2] != "" && !chavesTraduzidas[m[1]] {
			continue
		}
		if v := limpaValor(a.Original[k]); v != "" {
			b.WriteString(k + "=" + v + "\n")
		}
	}
	exec := prog
	if args := limpaValor(argumentosDoExec(a.Original["Exec"])); args != "" {
		exec += " " + args
	}
	b.WriteString("Exec=" + exec + "\n")
	// O TryExec não tem aspas nem códigos de campo: só o escape de texto.
	b.WriteString("TryExec=" + strings.ReplaceAll(a.Executavel, `\`, `\\`) + "\n")
	icone := a.Icone
	if icone == "" {
		icone = "application-x-executable"
	}
	b.WriteString("Icon=" + strings.ReplaceAll(icone, `\`, `\\`) + "\n")
	b.WriteString("X-MkInstalar=" + a.ID + "\n")
	return []byte(b.String()), nil
}
