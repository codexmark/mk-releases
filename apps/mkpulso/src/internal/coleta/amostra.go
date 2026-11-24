package coleta

import (
	"runtime"
	"time"
)

// Amostra é uma leitura dos contadores num instante. Duas delas dão as
// taxas (veja MedirProcessos, MedirCPU, MedirDiscos e MedirRede).
type Amostra struct {
	Quando    time.Duration // o relógio monotônico de quem colheu
	Processos []Processo
	CPU       EstadoCPU
	TemCPU    bool // /proc/stat veio
	Discos    []ContadorDisco
	Rede      []ContadorRede
	GPUProc   map[int]map[string]uint64 // pid → motor → ns
}

// Colher lê todos os contadores. quando é o instante no relógio de quem
// chama: só a diferença entre duas amostras importa.
func Colher(quando time.Duration) *Amostra {
	a := &Amostra{Quando: quando, GPUProc: map[int]map[string]uint64{}}
	a.CPU, a.TemCPU = LerEstadoCPU()
	a.Processos = LerProcessos()
	for _, p := range a.Processos {
		if p.Kernel {
			continue
		}
		if motores, ok := LerGPUProcesso(p.PID); ok {
			a.GPUProc[p.PID] = motores
		}
	}
	a.Discos = LerDiscos()
	a.Rede = LerRede()
	return a
}

// Nucleos é o número de núcleos lógicos (as linhas cpuN de /proc/stat).
// Nunca é zero: sem /proc/stat, vale o que o Go enxerga.
func (a *Amostra) Nucleos() int {
	if n := len(a.CPU.Nucleos); n > 0 {
		return n
	}
	if a.TemCPU {
		return 1
	}
	return runtime.NumCPU()
}

// Medido é um processo com as taxas calculadas entre duas amostras.
type Medido struct {
	Processo
	CPUPct   float64  // 100 = um núcleo inteiro
	DiscoBps *float64 // nil: o io do processo é ilegível
	GPUPct   *float64 // nil: o processo não usa GPU ou não dá para saber
	InicioS  *int64   // segundos desde a época; nil sem a hora do boot
}

// MedirProcessos calcula as taxas de cada processo da amostra nova. O par é
// feito por pid e hora de início (um pid reaproveitado é outro processo).
// Processo sem leitura anterior tem 0 de CPU e de disco no primeiro retrato.
func MedirProcessos(antes, depois *Amostra) []Medido {
	if depois == nil {
		return nil
	}
	velhos := map[int]*Processo{}
	dt := 0.0
	if antes != nil {
		dt = (depois.Quando - antes.Quando).Seconds()
		for i := range antes.Processos {
			velhos[antes.Processos[i].PID] = &antes.Processos[i]
		}
	}
	ms := make([]Medido, 0, len(depois.Processos))
	for _, p := range depois.Processos {
		m := Medido{Processo: p}
		if depois.TemCPU && depois.CPU.Boot > 0 {
			m.InicioS = Ptr(depois.CPU.Boot + int64(p.InicioTicks/TicksPorSegundo))
		}
		v := velhos[p.PID]
		if v != nil && v.InicioTicks != p.InicioTicks {
			v = nil
		}
		if p.TemIO {
			m.DiscoBps = Ptr(0.0)
		}
		motores, temGPU := depois.GPUProc[p.PID]
		if temGPU {
			m.GPUPct = Ptr(0.0)
		}
		if v != nil && dt > 0 {
			m.CPUPct = Arred(subiu(v.Ticks, p.Ticks)/TicksPorSegundo/dt*100, 1)
			if p.TemIO && v.TemIO {
				m.DiscoBps = Ptr(Arred((subiu(v.Leitura, p.Leitura)+subiu(v.Escrita, p.Escrita))/dt, 0))
			}
			if velhosMotores, ok := antes.GPUProc[p.PID]; ok && temGPU {
				// o motor mais ocupado: somar gfx e vídeo passaria de 100
				maior := 0.0
				for motor, ns := range motores {
					if antigo, ok := velhosMotores[motor]; ok {
						if d := subiu(antigo, ns); d > maior {
							maior = d
						}
					}
				}
				m.GPUPct = Ptr(Arred(limita(maior/(dt*1e9)*100, 0, 100), 1))
			}
		}
		ms = append(ms, m)
	}
	return ms
}
