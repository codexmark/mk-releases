// Package falso monta uma imagem squashfs 4.0 pequena, na memória, para os
// testes: o suficiente para um AppImage de mentira (um atalho, um ícone, uns
// links). Cada tabela cabe num bloco de metadados só, o que basta para uma
// dúzia de arquivos. Não é um mksquashfs: é o formato escrito à mão, para o
// leitor (../) ser testado sem ferramenta nenhuma.
package falso

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"sort"
	"strings"
)

// Arquivo é uma entrada da imagem. Com Link, é um link simbólico; com Pasta,
// uma pasta vazia; senão um arquivo comum com o Conteudo. As pastas do
// caminho são criadas sozinhas.
type Arquivo struct {
	Caminho  string
	Conteudo []byte
	Link     string
	Pasta    bool
}

// Opcoes da imagem.
type Opcoes struct {
	// Compressor é o número que vai no superbloco: 1 gzip, 4 xz, 6 zstd.
	// Zero vale 1.
	Compressor uint16
	// Comprime, se não for nil, comprime cada bloco (de dados e de
	// metadados). Com nil, os blocos vão sem compressão. Zlib é o do gzip.
	Comprime func([]byte) []byte
	// Bloco é o tamanho do bloco de dados (padrão 4096).
	Bloco uint32
	// Fragmentos põe o resto de cada arquivo (o que não enche um bloco) num
	// bloco de fragmentos, como o mksquashfs faz.
	Fragmentos bool
}

// Zlib comprime como o squashfs "gzip" espera.
func Zlib(b []byte) []byte {
	var out bytes.Buffer
	w := zlib.NewWriter(&out)
	_, _ = w.Write(b)
	_ = w.Close()
	return out.Bytes()
}

type no struct {
	nome    string
	arq     *Arquivo
	filhos  map[string]*no
	pasta   bool
	numero  uint32
	posicao int // onde o inode começa na tabela de inodes
	tamanho int // do inode
	// arquivo
	inicio  uint64
	blocos  []uint32
	frag    uint32
	fragPos uint32
	// pasta
	listaPos int
	listaTam int
}

const semFragmento = 0xFFFFFFFF

// Monta devolve os bytes da imagem.
func Monta(arquivos []Arquivo, op Opcoes) []byte {
	if op.Compressor == 0 {
		op.Compressor = 1
	}
	if op.Bloco == 0 {
		op.Bloco = 4096
	}
	raiz := &no{pasta: true, filhos: map[string]*no{}}
	for i := range arquivos {
		a := &arquivos[i]
		partes := strings.Split(strings.Trim(a.Caminho, "/"), "/")
		atual := raiz
		for j, p := range partes {
			f, ok := atual.filhos[p]
			if !ok {
				f = &no{nome: p, pasta: true, filhos: map[string]*no{}}
				atual.filhos[p] = f
			}
			if j == len(partes)-1 && !a.Pasta {
				f.pasta, f.arq, f.filhos = false, a, nil
			}
			atual = f
		}
	}

	le := binary.LittleEndian
	img := &bytes.Buffer{}
	img.Write(make([]byte, 96)) // o superbloco entra no fim

	// Os dados: os blocos de cada arquivo e, no fim, o bloco de fragmentos.
	var fragmentos bytes.Buffer
	var ordem []*no
	var anda func(n *no)
	anda = func(n *no) {
		nomes := make([]string, 0, len(n.filhos))
		for k := range n.filhos {
			nomes = append(nomes, k)
		}
		sort.Strings(nomes)
		for _, k := range nomes {
			anda(n.filhos[k])
		}
		ordem = append(ordem, n) // filhos antes do pai; a raiz por último
	}
	anda(raiz)
	for i, n := range ordem {
		n.numero = uint32(i + 1)
		if n.pasta || n.arq.Link != "" {
			continue
		}
		dados := n.arq.Conteudo
		n.inicio = uint64(img.Len())
		n.frag = semFragmento
		for len(dados) > 0 {
			pedaco := dados
			if len(pedaco) > int(op.Bloco) {
				pedaco = pedaco[:op.Bloco]
			}
			if op.Fragmentos && len(pedaco) < int(op.Bloco) {
				n.frag, n.fragPos = 0, uint32(fragmentos.Len())
				fragmentos.Write(pedaco)
				break
			}
			gravado, marca := pedaco, uint32(1<<24)
			if op.Comprime != nil {
				gravado, marca = op.Comprime(pedaco), 0
			}
			img.Write(gravado)
			n.blocos = append(n.blocos, uint32(len(gravado))|marca)
			dados = dados[len(pedaco):]
		}
	}
	fragInicio, fragTam := uint64(img.Len()), uint32(0)
	if fragmentos.Len() > 0 {
		if fragmentos.Len() > int(op.Bloco) {
			panic("falso: fragmentos demais para um bloco")
		}
		gravado, marca := fragmentos.Bytes(), uint32(1<<24)
		if op.Comprime != nil {
			gravado, marca = op.Comprime(gravado), 0
		}
		img.Write(gravado)
		fragTam = uint32(len(gravado)) | marca
	}

	// O tamanho de cada inode e de cada listagem, para saber as posições.
	pos := 0
	for _, n := range ordem {
		switch {
		case n.pasta:
			n.tamanho = 32
		case n.arq.Link != "":
			n.tamanho = 24 + len(n.arq.Link)
		default:
			n.tamanho = 32 + 4*len(n.blocos)
		}
		n.posicao = pos
		pos += n.tamanho
	}
	lpos := 0
	nomesDe := func(n *no) []string {
		nomes := make([]string, 0, len(n.filhos))
		for k := range n.filhos {
			nomes = append(nomes, k)
		}
		sort.Strings(nomes)
		return nomes
	}
	for _, n := range ordem {
		if !n.pasta {
			continue
		}
		n.listaPos = lpos
		if len(n.filhos) > 0 {
			n.listaTam = 12
			for k := range n.filhos {
				n.listaTam += 8 + len(k)
			}
		}
		lpos += n.listaTam
	}
	if pos > 8192 || lpos > 8192 {
		panic("falso: arquivos demais para um bloco de metadados")
	}

	// A tabela de inodes.
	inodes := &bytes.Buffer{}
	u16 := func(b *bytes.Buffer, v uint16) { _ = binary.Write(b, le, v) }
	u32 := func(b *bytes.Buffer, v uint32) { _ = binary.Write(b, le, v) }
	cabeca := func(tipo, modo uint16, n *no) {
		u16(inodes, tipo)
		u16(inodes, modo)
		u16(inodes, 0)
		u16(inodes, 0)
		u32(inodes, 0)
		u32(inodes, n.numero)
	}
	for _, n := range ordem {
		switch {
		case n.pasta:
			cabeca(1, 0o755, n)
			u32(inodes, 0) // o bloco da listagem (um só)
			u32(inodes, 2)
			u16(inodes, uint16(n.listaTam+3))
			u16(inodes, uint16(n.listaPos))
			u32(inodes, raiz.numero)
		case n.arq.Link != "":
			cabeca(3, 0o777, n)
			u32(inodes, 1)
			u32(inodes, uint32(len(n.arq.Link)))
			inodes.WriteString(n.arq.Link)
		default:
			cabeca(2, 0o644, n)
			u32(inodes, uint32(n.inicio))
			u32(inodes, n.frag)
			u32(inodes, n.fragPos)
			u32(inodes, uint32(len(n.arq.Conteudo)))
			for _, b := range n.blocos {
				u32(inodes, b)
			}
		}
	}

	// A tabela de pastas.
	pastas := &bytes.Buffer{}
	for _, n := range ordem {
		if !n.pasta || len(n.filhos) == 0 {
			continue
		}
		nomes := nomesDe(n)
		primeiro := n.filhos[nomes[0]].numero
		u32(pastas, uint32(len(nomes)-1))
		u32(pastas, 0)
		u32(pastas, primeiro)
		for _, k := range nomes {
			f := n.filhos[k]
			tipo := uint16(2)
			if f.pasta {
				tipo = 1
			} else if f.arq.Link != "" {
				tipo = 3
			}
			u16(pastas, uint16(f.posicao))
			u16(pastas, uint16(int16(int(f.numero)-int(primeiro))))
			u16(pastas, tipo)
			u16(pastas, uint16(len(k)-1))
			pastas.WriteString(k)
		}
	}

	meta := func(b []byte) {
		gravado, marca := b, uint16(0x8000)
		if op.Comprime != nil {
			gravado, marca = op.Comprime(b), 0
		}
		u16(img, uint16(len(gravado))|marca)
		img.Write(gravado)
	}
	inodeTab := uint64(img.Len())
	meta(inodes.Bytes())
	dirTab := uint64(img.Len())
	meta(pastas.Bytes())

	fragTab := uint64(0xFFFFFFFFFFFFFFFF)
	nFrag := uint32(0)
	if fragTam != 0 {
		nFrag = 1
		ondeEntradas := uint64(img.Len())
		e := &bytes.Buffer{}
		_ = binary.Write(e, le, fragInicio)
		u32(e, fragTam)
		u32(e, 0)
		meta(e.Bytes())
		fragTab = uint64(img.Len())
		_ = binary.Write(img, le, ondeEntradas)
	}
	ondeIds := uint64(img.Len())
	meta([]byte{0, 0, 0, 0})
	idTab := uint64(img.Len())
	_ = binary.Write(img, le, ondeIds)

	log := uint16(0)
	for v := op.Bloco; v > 1; v >>= 1 {
		log++
	}
	if 1<<log != op.Bloco {
		panic(fmt.Sprintf("falso: bloco %d não é potência de 2", op.Bloco))
	}
	flags := uint16(0)
	if op.Comprime == nil {
		flags |= 0x0001 | 0x0002 | 0x0008
	}
	if !op.Fragmentos {
		flags |= 0x0010
	}
	sb := &bytes.Buffer{}
	u32(sb, 0x73717368)
	u32(sb, uint32(len(ordem)))
	u32(sb, 0)
	u32(sb, op.Bloco)
	u32(sb, nFrag)
	u16(sb, op.Compressor)
	u16(sb, log)
	u16(sb, flags)
	u16(sb, 1)
	u16(sb, 4)
	u16(sb, 0)
	_ = binary.Write(sb, le, uint64(raiz.posicao)) // bloco 0, deslocamento = posição
	_ = binary.Write(sb, le, uint64(img.Len()))
	_ = binary.Write(sb, le, idTab)
	_ = binary.Write(sb, le, uint64(0xFFFFFFFFFFFFFFFF))
	_ = binary.Write(sb, le, inodeTab)
	_ = binary.Write(sb, le, dirTab)
	_ = binary.Write(sb, le, fragTab)
	_ = binary.Write(sb, le, uint64(0xFFFFFFFFFFFFFFFF))
	out := img.Bytes()
	copy(out, sb.Bytes())
	return out
}
