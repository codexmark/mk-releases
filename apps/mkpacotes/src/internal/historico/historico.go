// Package historico lê o registro do pacman (/var/log/pacman.log) e fica só
// com o que mudou nos pacotes: instalado, atualizado, removido, reinstalado
// ou rebaixado, com a data. O resto (ganchos, avisos, a linha de comando) é
// ignorado.
package historico

import (
	"bufio"
	"errors"
	"io"
	"os"
	"strings"
	"time"

	"mkpacotes/internal/base"
)

// Evento é uma mudança num pacote.
type Evento struct {
	Quando *int64  `json:"t_s"` // nil se a data da linha não deu para ler
	Acao   string  `json:"acao"`
	Nome   string  `json:"nome"`
	De     *string `json:"de"`
	Para   *string `json:"para"`
}

var acoes = map[string]string{"installed": "instalado", "upgraded": "atualizado", "removed": "removido",
	"reinstalled": "reinstalado", "downgraded": "rebaixado"}

// Le devolve os eventos do mais novo para o mais velho: no máximo `limite`,
// só os de pacote cujo nome contém `busca` (sem diferenciar maiúsculas).
// `total` conta os que casam, antes do corte.
func Le(busca string, limite int) (eventos []Evento, total int, err error) {
	f, err := os.Open(base.Log())
	if err != nil {
		return nil, 0, errors.New("não consegui ler o registro do pacman (" + base.Log() + ")")
	}
	defer f.Close()
	busca = strings.ToLower(busca)
	var todos []Evento
	leitor := bufio.NewReaderSize(f, 256*1024)
	for {
		linha, err := leLinha(leitor)
		if e, ok := evento(linha); ok && (busca == "" || strings.Contains(strings.ToLower(e.Nome), busca)) {
			todos = append(todos, e)
		}
		if err != nil {
			break
		}
	}
	total = len(todos)
	eventos = make([]Evento, 0, min(total, limite))
	for i := total - 1; i >= 0 && len(eventos) < limite; i-- {
		eventos = append(eventos, todos[i])
	}
	return eventos, total, nil
}

// leLinha lê uma linha de qualquer tamanho, mas só guarda o começo dela (uma
// linha de scriptlet pode ter megabytes; a de um evento é curta).
func leLinha(r *bufio.Reader) (string, error) {
	var b strings.Builder
	for {
		pedaco, err := r.ReadSlice('\n')
		if b.Len() < 4096 {
			b.Write(pedaco)
		}
		if err == bufio.ErrBufferFull {
			continue
		}
		if err != nil && err != io.EOF {
			return b.String(), err
		}
		return strings.TrimRight(b.String(), "\r\n"), err
	}
}

// evento entende uma linha "[data] [ALPM] verbo nome (versão)".
func evento(linha string) (Evento, bool) {
	if len(linha) < 2 || linha[0] != '[' {
		return Evento{}, false
	}
	data, resto, ok := strings.Cut(linha[1:], "] [ALPM] ")
	if !ok {
		return Evento{}, false
	}
	c := strings.SplitN(resto, " ", 3)
	if len(c) != 3 || acoes[c[0]] == "" || !strings.HasPrefix(c[2], "(") || !strings.HasSuffix(c[2], ")") {
		return Evento{}, false
	}
	e := Evento{Quando: instante(data), Acao: acoes[c[0]], Nome: c[1]}
	versoes := c[2][1 : len(c[2])-1]
	switch c[0] {
	case "installed":
		e.Para = &versoes
	case "removed":
		e.De = &versoes
	case "reinstalled":
		e.De, e.Para = &versoes, &versoes
	default:
		de, para, ok := strings.Cut(versoes, " -> ")
		if !ok {
			return Evento{}, false
		}
		e.De, e.Para = &de, &para
	}
	return e, true
}

// instante lê a data do pacman: a de hoje ("2026-11-20T23:44:58-0400") ou a
// antiga ("2019-03-01 10:00", na hora local).
func instante(data string) *int64 {
	if q, err := time.Parse("2006-01-02T15:04:05-0700", data); err == nil {
		s := q.Unix()
		return &s
	}
	if q, err := time.ParseInLocation("2006-01-02 15:04", data, time.Local); err == nil {
		s := q.Unix()
		return &s
	}
	return nil
}
