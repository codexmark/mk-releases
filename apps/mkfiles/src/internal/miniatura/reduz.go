package miniatura

import (
	"errors"
	"image"
	"image/color"
	_ "image/gif" // os decodificadores da biblioteca padrão
	_ "image/jpeg"
	_ "image/png"
	"os"
)

// Os limites do que o motor abre sozinho.
const (
	LimiteDoArquivo  = 64 << 20  // bytes
	LimiteDePixels   = 100000000 // 100 megapixels
	amostrasPorCaixa = 8         // por eixo: a caixa de uma imagem enorme é amostrada
)

// abre decodifica uma imagem, recusando antes a que é grande demais.
func abre(caminho string) (image.Image, error) {
	f, err := os.Open(caminho)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if info, err := f.Stat(); err != nil || info.Size() > LimiteDoArquivo {
		return nil, errors.New("imagem grande demais")
	}
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return nil, err
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > LimiteDePixels {
		return nil, errors.New("imagem grande demais")
	}
	if _, err := f.Seek(0, 0); err != nil {
		return nil, err
	}
	img, _, err := image.Decode(f)
	return img, err
}

// medidas diz o tamanho da miniatura: cabe num quadrado de lado px, na mesma
// proporção, sem ampliar.
func medidas(l, a, px int) (int, int) {
	if l <= px && a <= px {
		return l, a
	}
	if l >= a {
		a = (a*px + l/2) / l
		l = px
	} else {
		l = (l*px + a/2) / a
		a = px
	}
	return max(l, 1), max(a, 1)
}

// reduz encolhe a imagem por caixa: cada ponto da miniatura é a média dos
// pontos da origem que ele cobre.
func reduz(img image.Image, px int) image.Image {
	area := img.Bounds()
	lo, ao := area.Dx(), area.Dy()
	ld, ad := medidas(lo, ao, px)
	if ld == lo && ad == ao {
		return img
	}
	// O acesso direto aos tipos mais comuns: sem uma chamada de interface por ponto.
	var cor func(x, y int) (r, g, b, a uint32)
	switch o := img.(type) {
	case *image.YCbCr:
		cor = func(x, y int) (uint32, uint32, uint32, uint32) {
			c := o.YCbCrAt(x, y)
			r, g, b := color.YCbCrToRGB(c.Y, c.Cb, c.Cr)
			return uint32(r) * 0x101, uint32(g) * 0x101, uint32(b) * 0x101, 0xffff
		}
	case *image.NRGBA:
		cor = func(x, y int) (uint32, uint32, uint32, uint32) { return o.NRGBAAt(x, y).RGBA() }
	case *image.RGBA:
		cor = func(x, y int) (uint32, uint32, uint32, uint32) { return o.RGBAAt(x, y).RGBA() }
	case *image.Gray:
		cor = func(x, y int) (uint32, uint32, uint32, uint32) {
			v := uint32(o.GrayAt(x, y).Y) * 0x101
			return v, v, v, 0xffff
		}
	default:
		cor = func(x, y int) (uint32, uint32, uint32, uint32) { return img.At(x, y).RGBA() }
	}
	saida := image.NewRGBA(image.Rect(0, 0, ld, ad))
	for y := 0; y < ad; y++ {
		y0, y1 := area.Min.Y+y*ao/ad, area.Min.Y+(y+1)*ao/ad
		y1 = max(y1, y0+1)
		py := max((y1-y0)/amostrasPorCaixa, 1)
		for x := 0; x < ld; x++ {
			x0, x1 := area.Min.X+x*lo/ld, area.Min.X+(x+1)*lo/ld
			x1 = max(x1, x0+1)
			pxs := max((x1-x0)/amostrasPorCaixa, 1)
			var r, g, b, a, n uint64
			for sy := y0; sy < y1; sy += py {
				for sx := x0; sx < x1; sx += pxs {
					cr, cg, cb, ca := cor(sx, sy)
					r, g, b, a, n = r+uint64(cr), g+uint64(cg), b+uint64(cb), a+uint64(ca), n+1
				}
			}
			// De 16 bits (já multiplicados pelo alfa) para os 8 do RGBA.
			saida.SetRGBA(x, y, color.RGBA{uint8(r / n >> 8), uint8(g / n >> 8), uint8(b / n >> 8), uint8(a / n >> 8)})
		}
	}
	return saida
}
