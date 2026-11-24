package coleta

import (
	"os"
	"sort"
	"strings"

	"mkpulso/internal/base"
)

// ContadorRede são os bytes recebidos e enviados por uma interface.
type ContadorRede struct {
	Nome   string
	Rx, Tx uint64
}

// Rede é um item de "rede" do retrato.
type Rede struct {
	Nome  string  `json:"nome"`
	RxBps float64 `json:"rx_bps"`
	TxBps float64 `json:"tx_bps"`
}

// LerRede lê /proc/net/dev, sem a lo, em ordem de nome. O nome vai até o
// ':' (quando é comprido, o primeiro número vem colado nele).
func LerRede() []ContadorRede {
	b, err := os.ReadFile(base.Proc("net", "dev"))
	if err != nil {
		return nil
	}
	var rs []ContadorRede
	for _, linha := range strings.Split(string(b), "\n") {
		nome, resto, ok := strings.Cut(linha, ":")
		nome = strings.TrimSpace(nome)
		if !ok || nome == "" || nome == "lo" || strings.ContainsAny(nome, " |\t") {
			continue
		}
		c := strings.Fields(resto)
		if len(c) < 9 {
			continue
		}
		rs = append(rs, ContadorRede{Nome: nome, Rx: numero(c[0]), Tx: numero(c[8])})
	}
	sort.Slice(rs, func(i, j int) bool { return rs[i].Nome < rs[j].Nome })
	return rs
}

// MedirRede dá a taxa de cada interface entre duas amostras.
func MedirRede(antes, depois *Amostra) []Rede {
	rs := []Rede{}
	if depois == nil {
		return rs
	}
	velhas := map[string]ContadorRede{}
	dt := 0.0
	if antes != nil {
		dt = (depois.Quando - antes.Quando).Seconds()
		for _, r := range antes.Rede {
			velhas[r.Nome] = r
		}
	}
	for _, r := range depois.Rede {
		item := Rede{Nome: r.Nome}
		if v, ok := velhas[r.Nome]; ok && dt > 0 {
			item.RxBps = Arred(subiu(v.Rx, r.Rx)/dt, 0)
			item.TxBps = Arred(subiu(v.Tx, r.Tx)/dt, 0)
		}
		rs = append(rs, item)
	}
	return rs
}
