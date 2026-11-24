package alpm

import "strings"

// Vercmp compara duas versões como o pacman (alpm_pkg_vercmp): -1 se a é mais
// velha, 0 se iguais, 1 se a é mais nova. Uma versão é
// [época:]versão[-lançamento]; o lançamento só entra na conta se as duas têm.
func Vercmp(a, b string) int {
	if a == b {
		return 0
	}
	ea, va, ra := partes(a)
	eb, vb, rb := partes(b)
	if c := trecho(ea, eb); c != 0 {
		return c
	}
	if c := trecho(va, vb); c != 0 {
		return c
	}
	if ra != "" && rb != "" {
		return trecho(ra, rb)
	}
	return 0
}

func partes(v string) (epoca, versao, lancamento string) {
	epoca = "0"
	if i := strings.IndexByte(v, ':'); i >= 0 && soDigitos(v[:i]) {
		if i > 0 {
			epoca = v[:i]
		}
		v = v[i+1:]
	}
	if i := strings.LastIndexByte(v, '-'); i >= 0 {
		return epoca, v[:i], v[i+1:]
	}
	return epoca, v, ""
}

func soDigitos(s string) bool {
	for i := 0; i < len(s); i++ {
		if !digito(s[i]) {
			return false
		}
	}
	return true
}

func digito(c byte) bool { return c >= '0' && c <= '9' }
func letra(c byte) bool  { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }

// trecho é o rpmvercmp: anda pelos dois textos em pedaços só de números ou só
// de letras, separados por qualquer outra coisa.
func trecho(a, b string) int {
	if a == b {
		return 0
	}
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		si, sj := i, j
		for i < len(a) && !digito(a[i]) && !letra(a[i]) {
			i++
		}
		for j < len(b) && !digito(b[j]) && !letra(b[j]) {
			j++
		}
		if i >= len(a) || j >= len(b) {
			break
		}
		// Separadores de tamanhos diferentes: o mais longo é mais novo.
		if i-si != j-sj {
			if i-si < j-sj {
				return -1
			}
			return 1
		}
		pi, pj := i, j
		numero := digito(a[i])
		if numero {
			for i < len(a) && digito(a[i]) {
				i++
			}
			for j < len(b) && digito(b[j]) {
				j++
			}
		} else {
			for i < len(a) && letra(a[i]) {
				i++
			}
			for j < len(b) && letra(b[j]) {
				j++
			}
		}
		um, dois := a[pi:i], b[pj:j]
		// Tipos diferentes: número ganha de letra.
		if dois == "" {
			if numero {
				return 1
			}
			return -1
		}
		if numero {
			um, dois = strings.TrimLeft(um, "0"), strings.TrimLeft(dois, "0")
			if len(um) != len(dois) {
				if len(um) > len(dois) {
					return 1
				}
				return -1
			}
		}
		if c := strings.Compare(um, dois); c != 0 {
			return c
		}
	}
	if i >= len(a) && j >= len(b) {
		return 0
	}
	// O que sobrou: um resto de letras nunca ganha de nada ("1.5b" < "1.5").
	if (i >= len(a) && !letra(b[j])) || (i < len(a) && letra(a[i])) {
		return -1
	}
	return 1
}
