package grupos

// A regra da energia. É uma estimativa para a pessoa bater o olho, não uma
// medida em watts: o computador não diz quanto cada programa gasta.
//
//	impacto = cpu_pct ÷ núcleos          (0 a 100: a fatia da máquina inteira)
//	        + gpu_pct ÷ 2                (a GPU cheia vale meia máquina)
//	        + disco em MiB/s, até 10     (ler e gravar acorda o disco, mas pouco)
//
//	impacto < 0,1  → "nenhuma"
//	impacto < 2    → "muito baixa"
//	impacto < 8    → "baixa"
//	impacto < 25   → "moderada"
//	impacto < 60   → "alta"
//	o resto        → "muito alta"
//
// Exemplos numa máquina de 4 núcleos: um núcleo inteiro ocupado (cpu_pct 100)
// dá 25, "alta"; dois núcleos e meio, "muito alta"; um player usando 4 % da
// GPU, "baixa". O que não deu para ler (disco ou GPU null) conta como zero.
const (
	limiteNenhuma    = 0.1
	limiteMuitoBaixa = 2
	limiteBaixa      = 8
	limiteModerada   = 25
	limiteAlta       = 60

	pesoDaGPU     = 0.5
	tetoDoDisco   = 10.0
	bytesPorPonto = 1 << 20 // 1 MiB/s = 1 ponto
)

// Impacto é o número por trás da energia (veja a regra acima).
func Impacto(cpuPct float64, nucleos int, discoBps, gpuPct *float64) float64 {
	if nucleos < 1 {
		nucleos = 1
	}
	impacto := 0.0
	if cpuPct > 0 {
		impacto += cpuPct / float64(nucleos)
	}
	if gpuPct != nil && *gpuPct > 0 {
		impacto += *gpuPct * pesoDaGPU
	}
	if discoBps != nil && *discoBps > 0 {
		disco := *discoBps / bytesPorPonto
		if disco > tetoDoDisco {
			disco = tetoDoDisco
		}
		impacto += disco
	}
	return impacto
}

// Energia dá o estado do contrato para um grupo.
func Energia(cpuPct float64, nucleos int, discoBps, gpuPct *float64) string {
	switch i := Impacto(cpuPct, nucleos, discoBps, gpuPct); {
	case i < limiteNenhuma:
		return "nenhuma"
	case i < limiteMuitoBaixa:
		return "muito baixa"
	case i < limiteBaixa:
		return "baixa"
	case i < limiteModerada:
		return "moderada"
	case i < limiteAlta:
		return "alta"
	}
	return "muito alta"
}
