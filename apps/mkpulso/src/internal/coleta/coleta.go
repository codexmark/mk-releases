// Package coleta lê o sistema: os processos de /proc, a CPU, a memória, os
// discos, a rede, as GPUs, a temperatura e a bateria. Tudo passa pelas raízes
// de base (MKPULSO_PROC, MKPULSO_SYS, MKPULSO_ETC), então os testes rodam em
// árvores de mentira.
//
// Há dois tipos de leitura. Os contadores (ticks de CPU, bytes de disco e de
// rede, nanossegundos de GPU) só viram taxa com duas leituras: Colher tira
// uma Amostra, e as funções Medir* comparam duas amostras sobre o tempo que
// passou entre elas (o relógio monotônico de quem chamou). Os valores de
// agora (memória, temperatura, bateria, uso da GPU) são lidos na hora.
//
// Regra da casa: o que não deu para ler é nil (null no JSON), nunca zero; e
// nenhuma entrada torta derruba a leitura do resto.
package coleta

import (
	"math"
	"os"
	"strconv"
	"strings"

	"mkpulso/internal/base"
)

// TicksPorSegundo é o CLK_TCK do Linux: a unidade de utime e stime.
const TicksPorSegundo = 100

// Ptr devolve o endereço de um valor: é como os campos que podem ser null
// são preenchidos.
func Ptr[T any](v T) *T { return &v }

// Arred arredonda para um número de casas depois da vírgula.
func Arred(x float64, casas int) float64 {
	if math.IsNaN(x) || math.IsInf(x, 0) {
		return 0
	}
	p := math.Pow(10, float64(casas))
	return math.Round(x*p) / p
}

func lerTexto(caminho string) (string, bool) {
	b, err := os.ReadFile(caminho)
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(b)), true
}

func lerNumero(caminho string) (uint64, bool) {
	s, ok := lerTexto(caminho)
	if !ok {
		return 0, false
	}
	n, err := strconv.ParseUint(s, 10, 64)
	return n, err == nil
}

func lerInteiro(caminho string) (int64, bool) {
	s, ok := lerTexto(caminho)
	if !ok {
		return 0, false
	}
	n, err := strconv.ParseInt(s, 10, 64)
	return n, err == nil
}

// numero lê um inteiro sem sinal; lixo ou número grande demais vale 0.
func numero(s string) uint64 {
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// subiu devolve quanto um contador andou; se voltou (reinício), 0.
func subiu(antes, depois uint64) float64 {
	if depois < antes {
		return 0
	}
	return float64(depois - antes)
}

func limita(x, min, max float64) float64 {
	if math.IsNaN(x) || x < min {
		return min
	}
	if x > max {
		return max
	}
	return x
}

// Maquina é o nome da máquina: o do kernel, senão o de /etc/hostname, senão
// vazio.
func Maquina() string {
	if s, ok := lerTexto(base.Proc("sys", "kernel", "hostname")); ok && s != "" {
		return s
	}
	if s, ok := lerTexto(base.Etc("hostname")); ok {
		return s
	}
	return ""
}
