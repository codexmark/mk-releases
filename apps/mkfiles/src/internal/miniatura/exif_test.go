package miniatura

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"testing"

	"mkfiles/internal/prova"
)

// exif monta o segmento APP1/Exif com a tag de orientação (e uma outra antes,
// para a busca ter de andar), na ordem de bytes pedida.
func exif(ordem binary.ByteOrder, orientacao uint16) []byte {
	var tiff bytes.Buffer
	if ordem == binary.LittleEndian {
		tiff.WriteString("II")
	} else {
		tiff.WriteString("MM")
	}
	p := func(v any) { binary.Write(&tiff, ordem, v) }
	p(uint16(42))
	p(uint32(8))
	p(uint16(2)) // duas entradas
	p(uint16(0x010f))
	p(uint16(2))
	p(uint32(1))
	p(uint32(0)) // fabricante (qualquer)
	p(uint16(0x0112))
	p(uint16(3))
	p(uint32(1))
	p(orientacao)
	p(uint16(0))
	p(uint32(0))
	corpo := append([]byte("Exif\x00\x00"), tiff.Bytes()...)
	seg := []byte{0xff, 0xe1, byte((len(corpo) + 2) >> 8), byte(len(corpo) + 2)}
	return append(seg, corpo...)
}

// jpegCom codifica a imagem e enfia os segmentos logo depois do SOI.
func jpegCom(t *testing.T, img image.Image, segmentos ...[]byte) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := jpeg.Encode(&b, img, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}
	cru := b.Bytes()
	saida := append([]byte{}, cru[:2]...)
	for _, s := range segmentos {
		saida = append(saida, s...)
	}
	return append(saida, cru[2:]...)
}

func TestOrientacaoDoExif(t *testing.T) {
	img := imagem(8, 8)
	for _, ordem := range []binary.ByteOrder{binary.LittleEndian, binary.BigEndian} {
		for o := uint16(1); o <= 8; o++ {
			if lida := orientacao(jpegCom(t, img, exif(ordem, o))); lida != int(o) {
				t.Errorf("%v, orientação %d: li %d", ordem, o, lida)
			}
		}
	}
	// Um APP0 e um comentário antes do Exif não atrapalham.
	app0 := []byte{0xff, 0xe0, 0, 4, 'x', 'y'}
	if o := orientacao(jpegCom(t, img, app0, exif(binary.BigEndian, 6))); o != 6 {
		t.Errorf("com APP0 antes: %d", o)
	}
	// Sem Exif, valor fora de 1..8, ou bytes tortos: 1 (como está), sem estourar.
	torto := exif(binary.LittleEndian, 6)
	casos := [][]byte{
		jpegCom(t, img),
		jpegCom(t, img, exif(binary.LittleEndian, 9)),
		jpegCom(t, img, exif(binary.LittleEndian, 0)),
		{}, {0xff}, {0xff, 0xd8}, {0xff, 0xd8, 0xff, 0xe1}, {0xff, 0xd8, 0xff, 0xe1, 0xff, 0xff, 'E'},
		append([]byte{0xff, 0xd8}, torto[:len(torto)-9]...),
		[]byte("\x89PNG\r\n\x1a\n"),
	}
	// O deslocamento do IFD apontando para fora.
	fora := append([]byte{}, exif(binary.LittleEndian, 6)...)
	binary.LittleEndian.PutUint32(fora[14:], 0xfffffff0)
	casos = append(casos, append([]byte{0xff, 0xd8}, fora...))
	for i, c := range casos {
		if o := orientacao(c); o != 1 {
			t.Errorf("caso %d: orientação %d, esperado 1", i, o)
		}
	}
}

func TestOrienta(t *testing.T) {
	// Uma imagem 3x2 com um valor por ponto:
	//   1 2 3
	//   4 5 6
	img := image.NewGray(image.Rect(0, 0, 3, 2))
	for i := 0; i < 6; i++ {
		img.SetGray(i%3, i/3, color.Gray{uint8(i + 1)})
	}
	quer := map[int][][]uint8{
		1: {{1, 2, 3}, {4, 5, 6}},
		2: {{3, 2, 1}, {6, 5, 4}},   // espelhada
		3: {{6, 5, 4}, {3, 2, 1}},   // de cabeça para baixo
		4: {{4, 5, 6}, {1, 2, 3}},   // espelhada na vertical
		5: {{1, 4}, {2, 5}, {3, 6}}, // transposta
		6: {{4, 1}, {5, 2}, {6, 3}}, // girada 90° para a direita
		7: {{6, 3}, {5, 2}, {4, 1}}, // transversa
		8: {{3, 6}, {2, 5}, {1, 4}}, // girada 90° para a esquerda
	}
	for o, linhas := range quer {
		saida := orienta(img, o)
		if saida.Bounds().Dx() != len(linhas[0]) || saida.Bounds().Dy() != len(linhas) {
			t.Errorf("orientação %d: medidas %v", o, saida.Bounds())
			continue
		}
		for y, linha := range linhas {
			for x, v := range linha {
				r, _, _, _ := saida.At(saida.Bounds().Min.X+x, saida.Bounds().Min.Y+y).RGBA()
				if uint8(r>>8) != v {
					t.Errorf("orientação %d, ponto (%d,%d): %d, esperado %d", o, x, y, r>>8, v)
				}
			}
		}
	}
}

func TestMiniaturaDeFotoEmPe(t *testing.T) {
	m := motor(t)
	// A foto está gravada deitada (400x200, vermelho à esquerda) e o Exif diz
	// "gire 90° para a direita": a miniatura sai em pé, vermelho em cima.
	foto := m.Casa + "/fotos/em-pe.jpg"
	os.MkdirAll(m.Casa+"/fotos", 0o755)
	os.WriteFile(foto, jpegCom(t, imagem(400, 200), exif(binary.BigEndian, 6)), 0o644)
	pede(m, "normal", "g", foto)
	arquivo, _ := m.Evento("miniatura", de(foto))["arquivo"].(string)
	f, err := os.Open(arquivo)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 64 || img.Bounds().Dy() != 128 {
		t.Fatalf("medidas: %v", img.Bounds())
	}
	cima, _, _, _ := img.At(32, 10).RGBA()
	_, _, baixo, _ := img.At(32, 118).RGBA()
	if cima>>8 < 200 || baixo>>8 < 200 {
		t.Errorf("vermelho em cima (%d) e azul embaixo (%d)", cima>>8, baixo>>8)
	}
}

func TestTamanhoXLarge(t *testing.T) {
	m := motor(t)
	foto := m.Casa + "/fotos/grande.png"
	gravaPNG(t, foto, 1000, 500)
	pede(m, "x-large", "g", foto)
	arquivo, _ := m.Evento("miniatura", de(foto))["arquivo"].(string)
	if arquivo != m.Casa+"/.cache/thumbnails/x-large/"+md5De(foto)+".png" {
		t.Fatalf("x-large: %q", arquivo)
	}
	f, _ := os.Open(arquivo)
	defer f.Close()
	if cfg, err := png.DecodeConfig(f); err != nil || cfg.Width != 512 || cfg.Height != 256 {
		t.Errorf("x-large mede 512: %v %v", cfg, err)
	}
	// O miniaturizador de fora recebe 512 no %s (é o que a prévia de PDF usa).
	miniaturizador(t, m, "pdf", "application/pdf", `echo "$2" >"`+m.Casa+`/lado"; cp "`+arquivo+`" "$5"`)
	pdf := m.Grava("doc.pdf", "%PDF-1.4")
	pede(m, "x-large", "g", pdf)
	if a, _ := m.Evento("miniatura", de(pdf))["arquivo"].(string); a == "" {
		t.Fatal("o miniaturizador de fora não fez o x-large")
	}
	if b, _ := os.ReadFile(m.Casa + "/lado"); string(bytes.TrimSpace(b)) != "512" {
		t.Errorf("o lado passado ao miniaturizador: %q", b)
	}
	if r := m.Pede(`{"op":"miniatura","caminhos":` + prova.J([]string{foto}) + `,"tamanho":"xx-large"}`); r["campo"] != "tamanho" {
		t.Errorf("tamanho desconhecido: %v", r)
	}
}
