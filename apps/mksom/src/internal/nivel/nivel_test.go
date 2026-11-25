package nivel

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"testing"
)

func pcm(amostras []int16) []byte {
	b := make([]byte, 2*len(amostras))
	for i, a := range amostras {
		binary.LittleEndian.PutUint16(b[2*i:], uint16(a))
	}
	return b
}

// senoide devolve n amostras de uma senoide com o pico em dBFS dado, com um
// número inteiro de ciclos a cada 16 amostras.
func senoide(n int, dbfs float64) []int16 {
	amp := 32768 * math.Pow(10, dbfs/20)
	out := make([]int16, n)
	for i := range out {
		out[i] = int16(math.Round(amp * math.Sin(2*math.Pi*float64(i)/16)))
	}
	return out
}

func constante(n int, v int16) []int16 {
	out := make([]int16, n)
	for i := range out {
		out[i] = v
	}
	return out
}

func TestDe_SilencioSenoideESaturado(t *testing.T) {
	if n := De(constante(1000, 0)); n.Pico != -90 || n.Rms != -90 {
		t.Errorf("silêncio: %+v", n)
	}
	if n := De(nil); n.Pico != Piso || n.Rms != Piso {
		t.Errorf("sem amostras: %+v", n)
	}
	// Senoide com pico a −6 dBFS: o RMS fica 3 dB abaixo.
	if n := De(senoide(1600, -6)); n.Pico != -6 || n.Rms != -9 {
		t.Errorf("senoide a −6: %+v", n)
	}
	if n := De(senoide(1600, -40)); n.Pico != -40 || n.Rms != -43 {
		t.Errorf("senoide a −40: %+v", n)
	}
	// Saturado: tudo no teto (dos dois lados).
	if n := De(constante(500, 32767)); n.Pico != 0 || n.Rms != 0 {
		t.Errorf("saturado em cima: %+v", n)
	}
	if n := De(constante(500, -32768)); n.Pico != 0 || n.Rms != 0 {
		t.Errorf("saturado embaixo: %+v", n)
	}
	// Um estalo só: o pico vê, o RMS quase não.
	estalo := constante(1000, 0)
	estalo[500] = 32767
	if n := De(estalo); n.Pico != 0 || n.Rms != -30 {
		t.Errorf("estalo: %+v", n)
	}
	// Bem baixo: abaixo do piso fica no piso.
	if n := De(constante(100, 1)); n.Pico != -90 || n.Rms != -90 {
		t.Errorf("um bit: %+v", n)
	}
}

func coleta(t *testing.T, r io.Reader, porBloco int) []Nivel {
	t.Helper()
	var out []Nivel
	if err := Medir(r, porBloco, func(n Nivel) error { out = append(out, n); return nil }); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestMedir_UmNivelPorBloco(t *testing.T) {
	var fluxo []int16
	// Blocos de 1072 amostras: 67 ciclos inteiros da senoide.
	fluxo = append(fluxo, constante(1072, 0)...)
	fluxo = append(fluxo, senoide(1072, -6)...)
	fluxo = append(fluxo, constante(1072, 32767)...)
	fluxo = append(fluxo, constante(100, 32767)...) // um resto que não fecha bloco
	niveis := coleta(t, bytes.NewReader(pcm(fluxo)), 1072)
	if len(niveis) != 3 {
		t.Fatalf("%d níveis: %+v", len(niveis), niveis)
	}
	if niveis[0] != (Nivel{-90, -90}) || niveis[1].Pico != -6 || niveis[1].Rms != -9 || niveis[2] != (Nivel{0, 0}) {
		t.Errorf("%+v", niveis)
	}
}

// aosPoucos entrega o que tem de um em um byte ímpar, como um pipe faria.
type aosPoucos struct {
	r io.Reader
	n int
}

func (a *aosPoucos) Read(p []byte) (int, error) {
	a.n++
	tam := 1 + a.n%7
	if tam > len(p) {
		tam = len(p)
	}
	return a.r.Read(p[:tam])
}

func TestMedir_LeiturasPartidasNaoDesalinhamAsAmostras(t *testing.T) {
	fluxo := pcm(append(senoide(1600, -6), senoide(1600, -20)...))
	niveis := coleta(t, &aosPoucos{r: bytes.NewReader(fluxo)}, 1600)
	if len(niveis) != 2 || niveis[0] != (Nivel{-6, -9}) || niveis[1] != (Nivel{-20, -23}) {
		t.Errorf("%+v", niveis)
	}
}

func TestMedir_PulaUmCabecalhoWav(t *testing.T) {
	cabecalho := make([]byte, 44)
	copy(cabecalho, "RIFF")
	copy(cabecalho[8:], "WAVEfmt ")
	for i := 16; i < 44; i++ {
		cabecalho[i] = 0xff // lido como amostras, pareceria saturado
	}
	niveis := coleta(t, bytes.NewReader(append(cabecalho, pcm(constante(200, 0))...)), 100)
	if len(niveis) != 2 || niveis[0] != (Nivel{-90, -90}) || niveis[1] != (Nivel{-90, -90}) {
		t.Errorf("%+v", niveis)
	}
}

func TestMedir_ParaQuandoQuemRecebeDesiste(t *testing.T) {
	basta := errors.New("basta")
	n := 0
	err := Medir(bytes.NewReader(pcm(constante(1000, 0))), 100, func(Nivel) error {
		n++
		if n == 2 {
			return basta
		}
		return nil
	})
	if !errors.Is(err, basta) || n != 2 {
		t.Errorf("%v depois de %d", err, n)
	}
}

// O pw-record sem --raw manda o som num contêiner AU (".snd"): cabeçalho de
// tamanho variável e amostras em big-endian. Visto numa VM: lido como PCM
// cru, o cabeçalho virava um pico de 0 dBFS no primeiro bloco.
func TestMedir_EntendeOContainerAu(t *testing.T) {
	cabecalho := make([]byte, 28)
	copy(cabecalho, ".snd")
	binary.BigEndian.PutUint32(cabecalho[4:], 28) // onde o som começa
	binary.BigEndian.PutUint32(cabecalho[8:], 0xffffffff)
	binary.BigEndian.PutUint32(cabecalho[12:], 3) // 16 bits linear
	amostras := constante(200, 16384)             // -6 dBFS
	corpo := make([]byte, 2*len(amostras))
	for i, a := range amostras {
		binary.BigEndian.PutUint16(corpo[2*i:], uint16(a))
	}
	niveis := coleta(t, bytes.NewReader(append(cabecalho, corpo...)), 100)
	if len(niveis) != 2 {
		t.Fatalf("%+v", niveis)
	}
	for _, n := range niveis {
		if math.Abs(n.Pico+6.0) > 0.1 || math.Abs(n.Rms+6.0) > 0.1 {
			t.Errorf("esperava -6 dBFS, veio %+v", n)
		}
	}
}
