package coleta

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"mkpulso/internal/base"
)

// Tempos são os ticks de uma linha "cpu" de /proc/stat.
type Tempos struct {
	Total  uint64
	Ocioso uint64 // idle + iowait
}

// EstadoCPU é /proc/stat: a linha geral, uma por núcleo e a hora do boot.
type EstadoCPU struct {
	Geral   Tempos
	Nucleos []Tempos
	Boot    int64 // btime, em segundos desde a época; 0 se não veio
}

// LerEstadoCPU lê /proc/stat. O segundo valor diz se a linha geral veio.
func LerEstadoCPU() (EstadoCPU, bool) {
	var e EstadoCPU
	b, err := os.ReadFile(base.Proc("stat"))
	if err != nil {
		return e, false
	}
	tem := false
	for _, linha := range strings.Split(string(b), "\n") {
		c := strings.Fields(linha)
		if len(c) < 2 {
			continue
		}
		switch {
		case c[0] == "btime":
			e.Boot, _ = strconv.ParseInt(c[1], 10, 64)
		case strings.HasPrefix(c[0], "cpu"):
			t, ok := temposDaLinha(c[1:])
			if !ok {
				continue
			}
			if c[0] == "cpu" {
				e.Geral = t
				tem = true
			} else if _, err := strconv.Atoi(c[0][3:]); err == nil {
				e.Nucleos = append(e.Nucleos, t)
			}
		}
	}
	return e, tem
}

// user nice system idle iowait irq softirq steal (guest já está em user).
func temposDaLinha(c []string) (Tempos, bool) {
	var t Tempos
	if len(c) < 4 {
		return t, false
	}
	if len(c) > 8 {
		c = c[:8]
	}
	for i, s := range c {
		n, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return Tempos{}, false
		}
		t.Total += n
		if i == 3 || i == 4 {
			t.Ocioso += n
		}
	}
	return t, true
}

// CPU é o bloco "cpu" do retrato.
type CPU struct {
	Pct       *float64  `json:"pct"`
	PorNucleo []float64 `json:"por_nucleo"`
	FreqMHz   *float64  `json:"freq_mhz"`
	TempC     *float64  `json:"temp_c"`
	Carga     []float64 `json:"carga"`
}

func ocupacao(antes, depois Tempos) float64 {
	total := subiu(antes.Total, depois.Total)
	if total <= 0 {
		return 0
	}
	ocioso := subiu(antes.Ocioso, depois.Ocioso)
	return Arred(limita(100*(1-ocioso/total), 0, 100), 1)
}

// MedirCPU dá a ocupação total (0 a 100, a máquina inteira) e a de cada
// núcleo entre duas amostras. Só preenche pct e por_nucleo; frequência,
// temperatura e carga são de quem monta o retrato.
func MedirCPU(antes, depois *Amostra) CPU {
	c := CPU{PorNucleo: []float64{}}
	if antes == nil || depois == nil || !antes.TemCPU || !depois.TemCPU {
		return c
	}
	c.Pct = Ptr(ocupacao(antes.CPU.Geral, depois.CPU.Geral))
	for i := range depois.CPU.Nucleos {
		if i >= len(antes.CPU.Nucleos) {
			break
		}
		c.PorNucleo = append(c.PorNucleo, ocupacao(antes.CPU.Nucleos[i], depois.CPU.Nucleos[i]))
	}
	return c
}

// Frequencia é a média, em MHz, da frequência atual dos núcleos (cpufreq;
// sem ele, o "cpu MHz" de /proc/cpuinfo).
func Frequencia() *float64 {
	var soma float64
	var n int
	arquivos, _ := filepath.Glob(base.Sys("devices", "system", "cpu", "cpu[0-9]*", "cpufreq", "scaling_cur_freq"))
	for _, a := range arquivos {
		if khz, ok := lerNumero(a); ok && khz > 0 {
			soma += float64(khz) / 1000
			n++
		}
	}
	if n == 0 {
		b, err := os.ReadFile(base.Proc("cpuinfo"))
		if err != nil {
			return nil
		}
		for _, linha := range strings.Split(string(b), "\n") {
			chave, valor, ok := strings.Cut(linha, ":")
			if !ok || strings.TrimSpace(chave) != "cpu MHz" {
				continue
			}
			if mhz, err := strconv.ParseFloat(strings.TrimSpace(valor), 64); err == nil && mhz > 0 {
				soma += mhz
				n++
			}
		}
	}
	if n == 0 {
		return nil
	}
	return Ptr(Arred(soma/float64(n), 0))
}

// Carga são as três médias de /proc/loadavg (1, 5 e 15 minutos).
func Carga() []float64 {
	s, ok := lerTexto(base.Proc("loadavg"))
	if !ok {
		return nil
	}
	c := strings.Fields(s)
	if len(c) < 3 {
		return nil
	}
	carga := make([]float64, 3)
	for i := range carga {
		v, err := strconv.ParseFloat(c[i], 64)
		if err != nil {
			return nil
		}
		carga[i] = Arred(v, 2)
	}
	return carga
}

// Temperatura é a da CPU, em graus: o sensor do processador no hwmon
// (coretemp na Intel, k10temp ou zenpower na AMD), de preferência a leitura
// do pacote inteiro; sem ele, uma zona térmica (a x86_pkg_temp, senão a
// primeira que responder).
func Temperatura() *float64 {
	pastas, _ := filepath.Glob(base.Sys("class", "hwmon", "hwmon*"))
	for _, p := range pastas {
		nome, _ := lerTexto(filepath.Join(p, "name"))
		if nome != "coretemp" && nome != "k10temp" && nome != "zenpower" {
			continue
		}
		escolhido := filepath.Join(p, "temp1_input")
		melhor := 99
		rotulos, _ := filepath.Glob(filepath.Join(p, "temp*_label"))
		for _, r := range rotulos {
			texto, _ := lerTexto(r)
			nota := 99
			switch {
			case strings.HasPrefix(texto, "Package id"):
				nota = 0
			case texto == "Tdie":
				nota = 1
			case texto == "Tctl":
				nota = 2
			}
			if nota < melhor {
				melhor = nota
				escolhido = strings.TrimSuffix(r, "_label") + "_input"
			}
		}
		if v := miligraus(escolhido); v != nil {
			return v
		}
	}
	zonas, _ := filepath.Glob(base.Sys("class", "thermal", "thermal_zone*"))
	var primeira *float64
	for _, z := range zonas {
		v := miligraus(filepath.Join(z, "temp"))
		if v == nil {
			continue
		}
		if tipo, _ := lerTexto(filepath.Join(z, "type")); tipo == "x86_pkg_temp" {
			return v
		}
		if primeira == nil {
			primeira = v
		}
	}
	return primeira
}

func miligraus(arquivo string) *float64 {
	m, ok := lerInteiro(arquivo)
	if !ok || m < -100000 || m > 200000 {
		return nil
	}
	return Ptr(Arred(float64(m)/1000, 1))
}
