package coleta

import (
	"os"
	"strconv"
	"strings"

	"mkpulso/internal/base"
)

// Memoria é o bloco "mem" do retrato, em kB.
type Memoria struct {
	TotalKB      int64 `json:"total_kb"`
	UsadoKB      int64 `json:"usado_kb"`
	DisponivelKB int64 `json:"disponivel_kb"`
	CacheKB      int64 `json:"cache_kb"`
	SwapTotalKB  int64 `json:"swap_total_kb"`
	SwapUsadoKB  int64 `json:"swap_usado_kb"`
}

// LerMemoria lê /proc/meminfo. Usado é o total menos o disponível (o que o
// kernel diz que dá para pegar sem ir ao swap); cache é Cached + Buffers +
// SReclaimable. Sem MemTotal, devolve nil.
func LerMemoria() *Memoria {
	b, err := os.ReadFile(base.Proc("meminfo"))
	if err != nil {
		return nil
	}
	v := map[string]int64{}
	for _, linha := range strings.Split(string(b), "\n") {
		chave, valor, ok := strings.Cut(linha, ":")
		if !ok {
			continue
		}
		c := strings.Fields(valor)
		if len(c) == 0 {
			continue
		}
		if n, err := strconv.ParseInt(c[0], 10, 64); err == nil && n >= 0 {
			v[chave] = n
		}
	}
	total, ok := v["MemTotal"]
	if !ok || total <= 0 {
		return nil
	}
	m := &Memoria{TotalKB: total}
	m.CacheKB = v["Cached"] + v["Buffers"] + v["SReclaimable"]
	if disp, ok := v["MemAvailable"]; ok {
		m.DisponivelKB = disp
	} else {
		m.DisponivelKB = v["MemFree"] + v["Cached"] + v["Buffers"]
	}
	if m.DisponivelKB > total {
		m.DisponivelKB = total
	}
	m.UsadoKB = total - m.DisponivelKB
	m.SwapTotalKB = v["SwapTotal"]
	if livre := v["SwapFree"]; livre < m.SwapTotalKB {
		m.SwapUsadoKB = m.SwapTotalKB - livre
	}
	return m
}
