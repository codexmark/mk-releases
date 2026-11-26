package abrir

import (
	"errors"
	"strings"

	"mkfiles/internal/base"
)

// palavras separa a linha Exec em argumentos, como a especificação de Desktop
// Entry manda: espaços separam; entre aspas duplas vale tudo, e a barra
// invertida protege " ` $ e \. Nada passa por shell.
func palavras(linha string) ([]string, error) {
	var args []string
	var atual strings.Builder
	tem, aspas := false, false
	for i := 0; i < len(linha); i++ {
		c := linha[i]
		switch {
		case aspas && c == '\\' && i+1 < len(linha) && strings.IndexByte("\"`$\\", linha[i+1]) >= 0:
			i++
			atual.WriteByte(linha[i])
		case c == '"':
			aspas = !aspas
			tem = true
		case !aspas && (c == ' ' || c == '\t'):
			if tem {
				args = append(args, atual.String())
				atual.Reset()
				tem = false
			}
		default:
			atual.WriteByte(c)
			tem = true
		}
	}
	if aspas {
		return nil, errors.New("aspas sem par no Exec")
	}
	if tem {
		args = append(args, atual.String())
	}
	if len(args) == 0 || args[0] == "" {
		return nil, errors.New("Exec vazio")
	}
	return args, nil
}

// argvs monta as linhas de comando que abrem os arquivos com um app: uma só
// se o Exec aceita vários (%F, %U), uma por arquivo se não.
func argvs(a app, arquivos []string) ([][]string, error) {
	molde, err := palavras(a.Exec)
	if err != nil {
		return nil, err
	}
	varios, algum := false, false
	for _, p := range molde {
		for i := 0; i+1 < len(p); i++ {
			if p[i] != '%' {
				continue
			}
			switch p[i+1] {
			case 'F', 'U':
				varios, algum = true, true
			case 'f', 'u':
				algum = true
			}
			i++
		}
	}
	if varios {
		return [][]string{monta(a, molde, arquivos)}, nil
	}
	var todas [][]string
	for _, arq := range arquivos {
		argv := monta(a, molde, []string{arq})
		if !algum {
			argv = append(argv, arq)
		}
		todas = append(todas, argv)
	}
	return todas, nil
}

func monta(a app, molde, arquivos []string) []string {
	var argv []string
	for n, p := range molde {
		// Um código de campo sozinho vira um argumento por arquivo (ou some).
		switch p {
		case "%f", "%F":
			argv = append(argv, arquivos...)
			continue
		case "%u", "%U":
			for _, arq := range arquivos {
				argv = append(argv, base.URI(arq))
			}
			continue
		case "%i":
			if a.Icone != "" {
				argv = append(argv, "--icon", a.Icone)
			}
			continue
		}
		var b strings.Builder
		for i := 0; i < len(p); i++ {
			if p[i] != '%' || i+1 >= len(p) {
				b.WriteByte(p[i])
				continue
			}
			i++
			switch p[i] {
			case '%':
				b.WriteByte('%')
			case 'f', 'F':
				if len(arquivos) > 0 {
					b.WriteString(arquivos[0])
				}
			case 'u', 'U':
				if len(arquivos) > 0 {
					b.WriteString(base.URI(arquivos[0]))
				}
			case 'c':
				b.WriteString(a.Nome)
			case 'k':
				b.WriteString(a.Arquivo)
			}
			// O resto (%i no meio, %d %D %n %N %v %m, desconhecido) some.
		}
		if b.Len() > 0 || n == 0 {
			argv = append(argv, b.String())
		}
	}
	return argv
}
