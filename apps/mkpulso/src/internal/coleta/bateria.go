package coleta

import (
	"path/filepath"

	"mkpulso/internal/base"
)

// Bateria é o bloco "bateria" do retrato.
type Bateria struct {
	Pct        *int     `json:"pct"`
	Carregando bool     `json:"carregando"`
	Watts      *float64 `json:"watts"`
}

// PastaDaBateria acha a primeira /sys/class/power_supply/BAT*; "" se a
// máquina não tem bateria.
func PastaDaBateria() string {
	pastas, _ := filepath.Glob(base.Sys("class", "power_supply", "BAT*"))
	if len(pastas) == 0 {
		return ""
	}
	return pastas[0]
}

// LerBateria lê a carga, se está carregando e a potência (power_now; sem
// ele, corrente vezes tensão). Sem bateria, nil.
func LerBateria() *Bateria {
	pasta := PastaDaBateria()
	if pasta == "" {
		return nil
	}
	b := &Bateria{}
	if pct, ok := lerNumero(filepath.Join(pasta, "capacity")); ok && pct <= 100 {
		b.Pct = Ptr(int(pct))
	}
	estado, _ := lerTexto(filepath.Join(pasta, "status"))
	b.Carregando = estado == "Charging"
	if uw, ok := lerInteiro(filepath.Join(pasta, "power_now")); ok {
		b.Watts = Ptr(Arred(modulo(float64(uw))/1e6, 1))
	} else {
		ua, okA := lerInteiro(filepath.Join(pasta, "current_now"))
		uv, okV := lerInteiro(filepath.Join(pasta, "voltage_now"))
		if okA && okV {
			b.Watts = Ptr(Arred(modulo(float64(ua))*modulo(float64(uv))/1e12, 1))
		}
	}
	return b
}

func modulo(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
