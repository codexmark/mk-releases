// Package miniatura faz e guarda as miniaturas conforme a especificação do
// freedesktop: ~/.cache/thumbnails/{normal,large}/<md5 do URI>.png, com os
// blocos tEXt Thumb::URI e Thumb::MTime. PNG, JPEG e GIF o motor reduz
// sozinho; o resto vai para o miniaturizador do sistema (.thumbnailer). Veja
// ../../../CONTRATO.md.
package miniatura

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/png"
)

const assinaturaPNG = "\x89PNG\r\n\x1a\n"

// O bloco IHDR vem sempre primeiro e tem tamanho fixo: assinatura (8) +
// comprimento (4) + tipo (4) + dados (13) + crc (4).
const fimDoIHDR = 8 + 4 + 4 + 13 + 4

// blocoDeTexto monta um bloco tEXt (chave, byte nulo, valor) com o crc.
func blocoDeTexto(chave, valor string) []byte {
	dados := append(append([]byte(chave), 0), valor...)
	b := make([]byte, 0, 12+len(dados))
	b = binary.BigEndian.AppendUint32(b, uint32(len(dados)))
	b = append(b, "tEXt"...)
	b = append(b, dados...)
	return binary.BigEndian.AppendUint32(b, crc32.ChecksumIEEE(b[4:]))
}

// blocos percorre os blocos de um PNG: vê(tipo, dados, o bloco inteiro). Para
// quando vê devolve false. Erro se o arquivo não é um PNG inteiro até ali.
func blocos(dados []byte, ve func(tipo string, corpo, inteiro []byte) bool) error {
	if !bytes.HasPrefix(dados, []byte(assinaturaPNG)) {
		return errors.New("não é um PNG")
	}
	for i := 8; i < len(dados); {
		if i+12 > len(dados) {
			return errors.New("PNG cortado")
		}
		n := int(binary.BigEndian.Uint32(dados[i:]))
		if n < 0 || i+12+n > len(dados) {
			return errors.New("PNG cortado")
		}
		tipo := string(dados[i+4 : i+8])
		if !ve(tipo, dados[i+8:i+8+n], dados[i:i+12+n]) || tipo == "IEND" {
			return nil
		}
		i += 12 + n
	}
	return nil
}

// textosDe lê os blocos tEXt que vêm antes da imagem.
func textosDe(dados []byte) map[string]string {
	textos := map[string]string{}
	_ = blocos(dados, func(tipo string, corpo, _ []byte) bool {
		if tipo == "IDAT" {
			return false
		}
		if tipo == "tEXt" {
			if chave, valor, ok := bytes.Cut(corpo, []byte{0}); ok {
				textos[string(chave)] = string(valor)
			}
		}
		return true
	})
	return textos
}

// comTextos devolve o PNG com esses blocos tEXt logo depois do IHDR, no lugar
// dos que ele tivesse com as mesmas chaves.
func comTextos(dados []byte, textos [][2]string) ([]byte, error) {
	if len(dados) < fimDoIHDR || string(dados[12:16]) != "IHDR" {
		return nil, errors.New("não é um PNG")
	}
	nossas := map[string]bool{}
	for _, t := range textos {
		nossas[t[0]] = true
	}
	saida := make([]byte, 0, len(dados)+256)
	saida = append(saida, dados[:fimDoIHDR]...)
	for _, t := range textos {
		saida = append(saida, blocoDeTexto(t[0], t[1])...)
	}
	temFim := false
	err := blocos(dados, func(tipo string, corpo, inteiro []byte) bool {
		switch tipo {
		case "IHDR":
			return true
		case "tEXt":
			if chave, _, ok := bytes.Cut(corpo, []byte{0}); ok && nossas[string(chave)] {
				return true
			}
		case "IEND":
			temFim = true
		}
		saida = append(saida, inteiro...)
		return true
	})
	if err != nil {
		return nil, err
	}
	if !temFim {
		return nil, errors.New("PNG cortado")
	}
	return saida, nil
}

// codifica escreve uma imagem como PNG com os blocos tEXt.
func codifica(img image.Image, textos [][2]string) ([]byte, error) {
	var b bytes.Buffer
	if err := (&png.Encoder{CompressionLevel: png.BestSpeed}).Encode(&b, img); err != nil {
		return nil, err
	}
	return comTextos(b.Bytes(), textos)
}
