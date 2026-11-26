package miniatura

import (
	"encoding/binary"
	"image"
	"image/color"
	"io"
	"os"
)

// Quanto do começo de um JPEG é lido atrás da orientação: o Exif vem logo
// depois do início do arquivo.
const comecoDoJPEG = 128 << 10

// orientacaoDe lê a orientação do Exif de um arquivo JPEG (1 a 8; 1 se não
// há, ou se não deu para ler).
func orientacaoDe(caminho string) int {
	f, err := os.Open(caminho)
	if err != nil {
		return 1
	}
	defer f.Close()
	b := make([]byte, comecoDoJPEG)
	n, _ := io.ReadFull(f, b)
	return orientacao(b[:n])
}

// orientacao acha a tag de orientação (0x0112) no segmento APP1/Exif de um
// JPEG. Só essa tag é lida, à mão; qualquer coisa fora do lugar dá 1 (a
// imagem como está).
func orientacao(b []byte) int {
	if len(b) < 4 || b[0] != 0xff || b[1] != 0xd8 {
		return 1
	}
	for i := 2; i+4 <= len(b); {
		if b[i] != 0xff {
			return 1
		}
		marca := b[i+1]
		if marca == 0xff { // enchimento
			i++
			continue
		}
		if marca == 0xda || marca == 0xd9 { // começou a imagem: não há Exif
			return 1
		}
		tamanho := int(b[i+2])<<8 | int(b[i+3])
		if tamanho < 2 {
			return 1
		}
		fim := i + 2 + tamanho
		if marca == 0xe1 && fim <= len(b) && tamanho >= 8 && string(b[i+4:i+10]) == "Exif\x00\x00" {
			return orientacaoDoTIFF(b[i+10 : fim])
		}
		i = fim
	}
	return 1
}

func orientacaoDoTIFF(t []byte) int {
	if len(t) < 8 {
		return 1
	}
	var ordem binary.ByteOrder
	switch string(t[:2]) {
	case "II":
		ordem = binary.LittleEndian
	case "MM":
		ordem = binary.BigEndian
	default:
		return 1
	}
	if ordem.Uint16(t[2:]) != 42 {
		return 1
	}
	ifd := uint64(ordem.Uint32(t[4:]))
	if ifd+2 > uint64(len(t)) {
		return 1
	}
	entradas := uint64(ordem.Uint16(t[ifd:]))
	for e := uint64(0); e < entradas; e++ {
		p := ifd + 2 + e*12
		if p+12 > uint64(len(t)) {
			return 1
		}
		if ordem.Uint16(t[p:]) != 0x0112 {
			continue
		}
		// Um SHORT só: o valor mora nos dois primeiros bytes do campo.
		if ordem.Uint16(t[p+2:]) != 3 || ordem.Uint32(t[p+4:]) != 1 {
			return 1
		}
		if v := int(ordem.Uint16(t[p+8:])); v >= 1 && v <= 8 {
			return v
		}
		return 1
	}
	return 1
}

// orienta devolve a imagem como a orientação do Exif manda mostrar (2 a 8:
// espelhada, girada, transposta).
func orienta(img image.Image, o int) image.Image {
	if o < 2 || o > 8 {
		return img
	}
	area := img.Bounds()
	l, a := area.Dx(), area.Dy()
	ld, ad := l, a
	if o >= 5 {
		ld, ad = a, l
	}
	saida := image.NewRGBA(image.Rect(0, 0, ld, ad))
	for y := 0; y < ad; y++ {
		for x := 0; x < ld; x++ {
			var sx, sy int
			switch o {
			case 2:
				sx, sy = l-1-x, y
			case 3:
				sx, sy = l-1-x, a-1-y
			case 4:
				sx, sy = x, a-1-y
			case 5:
				sx, sy = y, x
			case 6:
				sx, sy = y, a-1-x
			case 7:
				sx, sy = l-1-y, a-1-x
			case 8:
				sx, sy = l-1-y, x
			}
			saida.Set(x, y, color.RGBAModel.Convert(img.At(area.Min.X+sx, area.Min.Y+sy)))
		}
	}
	return saida
}
