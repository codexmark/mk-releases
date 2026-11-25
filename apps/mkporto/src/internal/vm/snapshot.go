package vm

import (
	"regexp"
	"strings"
)

// Snapshot é uma linha do `qemu-img snapshot -l`.
type Snapshot struct {
	ID      string `json:"id"`
	Rotulo  string `json:"rotulo"`
	Data    string `json:"data"`
	Tamanho string `json:"tamanho"`
}

var (
	dataDeSnapshot = regexp.MustCompile(`\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}`)
	unidade        = regexp.MustCompile(`^(B|[KMGTPE]i?B?)$`)
)

// LeSnapshots lê a saída do `qemu-img snapshot -l`:
//
//	Snapshot list:
//	ID        TAG               VM SIZE                DATE     VM CLOCK     ICOUNT
//	1         antes                 0 B 2026-11-24 10:00:00 00:00:00.000          0
//
// O formato mudou entre versões (o tamanho já foi "0" e "1.1G", hoje é
// "0 B" e "1.1 GiB"): a âncora de cada linha é a data. Antes dela vêm o
// número, o rótulo (que pode ter espaço) e o tamanho; o que vem depois não
// interessa. Linha sem data (os cabeçalhos) é pulada.
func LeSnapshots(saida string) []Snapshot {
	lista := []Snapshot{}
	for _, linha := range strings.Split(saida, "\n") {
		onde := dataDeSnapshot.FindStringIndex(linha)
		if onde == nil {
			continue
		}
		antes := strings.Fields(linha[:onde[0]])
		if len(antes) < 3 {
			continue
		}
		s := Snapshot{ID: antes[0], Data: linha[onde[0]:onde[1]]}
		resto := antes[1:]
		if n := len(resto); n >= 3 && unidade.MatchString(resto[n-1]) {
			s.Tamanho = resto[n-2] + " " + resto[n-1]
			resto = resto[:n-2]
		} else {
			s.Tamanho = resto[n-1]
			resto = resto[:n-1]
		}
		s.Rotulo = strings.Join(resto, " ")
		lista = append(lista, s)
	}
	return lista
}
