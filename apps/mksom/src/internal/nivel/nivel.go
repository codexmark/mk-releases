// Package nivel mede o volume de um fluxo de PCM (16 bits com sinal, little
// endian, um canal): o pico e o RMS de cada bloco, em dBFS.
package nivel

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"math"
)

// Piso é o menor nível que o medidor diz, em dBFS.
const Piso = -90.0

// Nivel é o pico e o RMS de um bloco, em dBFS, com uma casa.
type Nivel struct {
	Pico float64 `json:"pico"`
	Rms  float64 `json:"rms"`
}

// emDB converte uma amplitude (1 = fundo de escala) para dBFS, com uma casa
// e o piso.
func emDB(amplitude float64) float64 {
	if amplitude <= 0 {
		return Piso
	}
	d := math.Round(20*math.Log10(amplitude)*10) / 10
	if d < Piso {
		return Piso
	}
	if d >= 0 {
		return 0 // sem "-0" nem acima do teto
	}
	return d
}

// De mede um bloco de amostras.
func De(amostras []int16) Nivel {
	if len(amostras) == 0 {
		return Nivel{Piso, Piso}
	}
	pico, soma := 0.0, 0.0
	for _, a := range amostras {
		v := math.Abs(float64(a))
		// O teto positivo (32767) conta como fundo de escala, como o negativo.
		if v >= 32767 {
			v = 32768
		}
		if v > pico {
			pico = v
		}
		soma += v * v
	}
	return Nivel{emDB(pico / 32768), emDB(math.Sqrt(soma/float64(len(amostras))) / 32768)}
}

// Medir lê o fluxo até o fim e chama emite com o nível de cada bloco de
// `porBloco` amostras (o resto que não fecha um bloco é descartado). Se o
// fluxo começa com um cabeçalho WAV (44 bytes), ele é pulado; se começa com um
// AU (".snd", o que o pw-record manda sem --raw), pula o cabeçalho do tamanho
// que ele diz e lê as amostras em big-endian. Para no primeiro erro de emite.
func Medir(r io.Reader, porBloco int, emite func(Nivel) error) error {
	if porBloco < 1 {
		porBloco = 1
	}
	br := bufio.NewReaderSize(r, 2*porBloco)
	if inicio, _ := br.Peek(12); len(inicio) == 12 && bytes.Equal(inicio[:4], []byte("RIFF")) && bytes.Equal(inicio[8:], []byte("WAVE")) {
		if _, err := br.Discard(44); err != nil {
			return nil
		}
	}
	var ordem binary.ByteOrder = binary.LittleEndian
	if inicio, _ := br.Peek(8); len(inicio) == 8 && bytes.Equal(inicio[:4], []byte(".snd")) {
		comeco := int(binary.BigEndian.Uint32(inicio[4:]))
		if comeco < 24 || comeco > 4096 {
			comeco = 24
		}
		if _, err := br.Discard(comeco); err != nil {
			return nil
		}
		ordem = binary.BigEndian
	}
	bruto := make([]byte, 2*porBloco)
	amostras := make([]int16, porBloco)
	for {
		if _, err := io.ReadFull(br, bruto); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return nil
			}
			return err
		}
		for i := range amostras {
			amostras[i] = int16(ordem.Uint16(bruto[2*i:]))
		}
		if err := emite(De(amostras)); err != nil {
			return err
		}
	}
}
