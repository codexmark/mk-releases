// Package squash lê um squashfs 4.0 só com a biblioteca padrão: o bastante
// para tirar de dentro de um AppImage o atalho (.desktop) e o ícone SEM
// executar o arquivo (nada de `--appimage-extract`) e sem montar nada.
//
// É só leitura e desconfia de tudo: todo tamanho que vem da imagem tem teto
// (um bloco de metadados tem 8 KiB, um bloco de dados tem o tamanho do
// superbloco e no máximo 1 MiB, uma listagem 1 MiB, um arquivo o teto que
// quem chama der) e um link simbólico é resolvido dentro da imagem, com
// limite de saltos. Uma imagem estragada dá erro, nunca pânico.
//
// A compressão gzip (zlib) é da biblioteca padrão. zstd e xz não são: cada
// bloco passa pelo comando `zstd -dc` ou `xz -dc` do PATH, pelo stdin (o
// comando só vê bytes, nunca um nome de arquivo). Sem o comando, erro, e
// quem chama segue sem o que queria ler.
package squash

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
	"time"

	"mkutil/internal/base"
)

// ErrNaoExiste é o erro de um caminho que a imagem não tem.
var ErrNaoExiste = errors.New("não existe na imagem")

const (
	magia          = 0x73717368 // "hsqs"
	tamanhoDeMeta  = 8192
	blocoMaximo    = 1 << 20
	listagemMaxima = 1 << 20
	saltosDeLink   = 16
	semFragmento   = 0xFFFFFFFF
	naoComprimido  = 1 << 24
)

// FS é uma imagem aberta.
type FS struct {
	r          io.ReaderAt
	base       int64
	bloco      uint32
	compressor uint16
	fragmentos uint32
	raiz       uint64
	inodes     int64
	pastas     int64
	fragTab    int64
	// Os blocos de metadados já abertos (poucos: cada um tem até 8 KiB).
	guardados map[int64]metaGuardada
}

type metaGuardada struct {
	dados []byte
	prox  int64
}

// Entrada é um nome de uma pasta. Tipo: "pasta", "arquivo", "link" ou "outro".
type Entrada struct {
	Nome string
	Tipo string
	ref  uint64
}

type inode struct {
	tipo string
	// pasta
	listaBloco uint32
	listaPos   uint16
	listaTam   uint32
	// arquivo
	inicio  uint64
	tamanho uint64
	frag    uint32
	fragPos uint32
	blocos  []uint32
	// link
	alvo string
}

// Abre lê o superbloco de uma imagem que começa em `deslocamento` dentro de r.
func Abre(r io.ReaderAt, deslocamento int64) (*FS, error) {
	sb := make([]byte, 96)
	if _, err := r.ReadAt(sb, deslocamento); err != nil {
		return nil, errors.New("não é um squashfs: curto demais")
	}
	le := binary.LittleEndian
	if le.Uint32(sb[0:]) != magia {
		return nil, errors.New("não é um squashfs")
	}
	if maior, menor := le.Uint16(sb[28:]), le.Uint16(sb[30:]); maior != 4 {
		return nil, fmt.Errorf("squashfs de versão %d.%d: só leio a 4", maior, menor)
	}
	fs := &FS{
		r:          r,
		base:       deslocamento,
		bloco:      le.Uint32(sb[12:]),
		fragmentos: le.Uint32(sb[16:]),
		compressor: le.Uint16(sb[20:]),
		raiz:       le.Uint64(sb[32:]),
		inodes:     int64(le.Uint64(sb[64:])),
		pastas:     int64(le.Uint64(sb[72:])),
		fragTab:    int64(le.Uint64(sb[80:])),
	}
	if fs.bloco < 4096 || fs.bloco > blocoMaximo || fs.bloco&(fs.bloco-1) != 0 {
		return nil, fmt.Errorf("squashfs com bloco de %d bytes: não leio", fs.bloco)
	}
	if fs.inodes < 96 || fs.pastas < 96 {
		return nil, errors.New("squashfs estragado: tabelas fora do lugar")
	}
	return fs, nil
}

// Lista devolve os nomes de uma pasta, em ordem.
func (fs *FS) Lista(caminho string) ([]Entrada, error) {
	in, err := fs.acha(caminho, saltosDeLink)
	if err != nil {
		return nil, err
	}
	if in.tipo != "pasta" {
		return nil, fmt.Errorf("%s não é uma pasta", caminho)
	}
	return fs.lista(in)
}

// Le devolve o conteúdo de um arquivo (seguindo links), se ele tiver até
// `max` bytes.
func (fs *FS) Le(caminho string, max int64) ([]byte, error) {
	in, err := fs.acha(caminho, saltosDeLink)
	if err != nil {
		return nil, err
	}
	if in.tipo != "arquivo" {
		return nil, fmt.Errorf("%s não é um arquivo", caminho)
	}
	if max < 0 || in.tamanho > uint64(max) {
		return nil, fmt.Errorf("%s é grande demais (%d bytes)", caminho, in.tamanho)
	}
	return fs.conteudo(in)
}

// acha resolve um caminho a partir da raiz da imagem. "." e ".." são
// resolvidos antes (nunca sobem além da raiz) e um link no meio ou no fim é
// seguido dentro da imagem.
func (fs *FS) acha(caminho string, saltos int) (*inode, error) {
	limpo := path.Clean("/" + caminho)
	atual, err := fs.inode(fs.raiz)
	if err != nil {
		return nil, err
	}
	if limpo == "/" {
		return atual, nil
	}
	partes := strings.Split(limpo[1:], "/")
	for i, p := range partes {
		if atual.tipo != "pasta" {
			return nil, ErrNaoExiste
		}
		lista, err := fs.lista(atual)
		if err != nil {
			return nil, err
		}
		j := sort.Search(len(lista), func(k int) bool { return lista[k].Nome >= p })
		if j >= len(lista) || lista[j].Nome != p {
			// Uma imagem fora de ordem: procura do jeito lento.
			j = -1
			for k := range lista {
				if lista[k].Nome == p {
					j = k
					break
				}
			}
			if j < 0 {
				return nil, ErrNaoExiste
			}
		}
		in, err := fs.inode(lista[j].ref)
		if err != nil {
			return nil, err
		}
		if in.tipo == "link" {
			if saltos <= 0 {
				return nil, errors.New("links demais no caminho")
			}
			novo := in.alvo
			if !strings.HasPrefix(novo, "/") {
				novo = "/" + strings.Join(partes[:i], "/") + "/" + novo
			}
			return fs.acha(novo+"/"+strings.Join(partes[i+1:], "/"), saltos-1)
		}
		atual = in
	}
	return atual, nil
}

// cursor lê uma tabela de metadados em sequência, bloco a bloco.
type cursor struct {
	fs   *FS
	prox int64 // onde começa o próximo bloco, no arquivo
	buf  []byte
}

func (fs *FS) cursor(tabela int64, bloco uint32, pos uint16) (*cursor, error) {
	c := &cursor{fs: fs, prox: fs.base + tabela + int64(bloco)}
	if err := c.carrega(); err != nil {
		return nil, err
	}
	if int(pos) > len(c.buf) {
		return nil, errors.New("squashfs estragado: posição fora do bloco")
	}
	c.buf = c.buf[pos:]
	return c, nil
}

func (c *cursor) carrega() error {
	dados, prox, err := c.fs.meta(c.prox)
	if err != nil {
		return err
	}
	c.buf, c.prox = dados, prox
	return nil
}

func (c *cursor) le(n int) ([]byte, error) {
	out := make([]byte, 0, n)
	for len(out) < n {
		if len(c.buf) == 0 {
			if err := c.carrega(); err != nil {
				return nil, err
			}
			if len(c.buf) == 0 {
				return nil, errors.New("squashfs estragado: bloco de metadados vazio")
			}
		}
		k := min(n-len(out), len(c.buf))
		out = append(out, c.buf[:k]...)
		c.buf = c.buf[k:]
	}
	return out, nil
}

// meta lê um bloco de metadados que começa em pos (no arquivo) e devolve o
// conteúdo e onde começa o seguinte.
func (fs *FS) meta(pos int64) ([]byte, int64, error) {
	if g, ok := fs.guardados[pos]; ok {
		return g.dados, g.prox, nil
	}
	dados, prox, err := fs.metaDoDisco(pos)
	if err != nil {
		return nil, 0, err
	}
	if fs.guardados == nil || len(fs.guardados) >= 256 {
		fs.guardados = map[int64]metaGuardada{}
	}
	fs.guardados[pos] = metaGuardada{dados, prox}
	return dados, prox, nil
}

func (fs *FS) metaDoDisco(pos int64) ([]byte, int64, error) {
	var cab [2]byte
	if _, err := fs.r.ReadAt(cab[:], pos); err != nil {
		return nil, 0, errors.New("squashfs estragado: metadados além do fim")
	}
	h := binary.LittleEndian.Uint16(cab[:])
	tam := int(h & 0x7FFF)
	if tam == 0 || tam > tamanhoDeMeta+512 {
		return nil, 0, errors.New("squashfs estragado: bloco de metadados de tamanho impossível")
	}
	dados := make([]byte, tam)
	if _, err := fs.r.ReadAt(dados, pos+2); err != nil {
		return nil, 0, errors.New("squashfs estragado: metadados além do fim")
	}
	if h&0x8000 == 0 {
		var err error
		if dados, err = fs.descomprime(dados, tamanhoDeMeta); err != nil {
			return nil, 0, err
		}
	} else if tam > tamanhoDeMeta {
		return nil, 0, errors.New("squashfs estragado: bloco de metadados grande demais")
	}
	return dados, pos + 2 + int64(tam), nil
}

// descomprime abre um bloco, recusando o que passar de max bytes.
func (fs *FS) descomprime(dados []byte, max int) ([]byte, error) {
	var out []byte
	switch fs.compressor {
	case 1:
		z, err := zlib.NewReader(bytes.NewReader(dados))
		if err != nil {
			return nil, errors.New("squashfs estragado: bloco gzip ilegível")
		}
		defer z.Close()
		out, err = io.ReadAll(io.LimitReader(z, int64(max)+1))
		if err != nil {
			return nil, errors.New("squashfs estragado: bloco gzip ilegível")
		}
	case 4, 6:
		nome := map[uint16]string{4: "xz", 6: "zstd"}[fs.compressor]
		var err error
		if out, err = base.Roda(nome, 20*time.Second, dados, "-dc"); err != nil {
			return nil, fmt.Errorf("não consegui abrir a compressão %s da imagem: %v", nome, err)
		}
	default:
		nomes := map[uint16]string{2: "lzma", 3: "lzo", 5: "lz4"}
		nome := nomes[fs.compressor]
		if nome == "" {
			nome = fmt.Sprintf("nº %d", fs.compressor)
		}
		return nil, fmt.Errorf("não sei ler a compressão %s desta imagem", nome)
	}
	if len(out) > max {
		return nil, errors.New("squashfs estragado: bloco maior que o permitido")
	}
	return out, nil
}

func (fs *FS) inode(ref uint64) (*inode, error) {
	if ref>>48 != 0 {
		return nil, errors.New("squashfs estragado: referência de inode impossível")
	}
	c, err := fs.cursor(fs.inodes, uint32(ref>>16), uint16(ref&0xFFFF))
	if err != nil {
		return nil, err
	}
	le := binary.LittleEndian
	cab, err := c.le(16)
	if err != nil {
		return nil, err
	}
	in := &inode{}
	switch tipo := le.Uint16(cab); tipo {
	case 1: // pasta
		b, err := c.le(16)
		if err != nil {
			return nil, err
		}
		in.tipo = "pasta"
		in.listaBloco, in.listaTam, in.listaPos = le.Uint32(b[0:]), uint32(le.Uint16(b[8:])), le.Uint16(b[10:])
	case 8: // pasta estendida
		b, err := c.le(24)
		if err != nil {
			return nil, err
		}
		in.tipo = "pasta"
		in.listaTam, in.listaBloco, in.listaPos = le.Uint32(b[4:]), le.Uint32(b[8:]), le.Uint16(b[18:])
	case 2, 9: // arquivo, arquivo estendido
		in.tipo = "arquivo"
		if tipo == 2 {
			b, err := c.le(16)
			if err != nil {
				return nil, err
			}
			in.inicio, in.frag, in.fragPos, in.tamanho = uint64(le.Uint32(b[0:])), le.Uint32(b[4:]), le.Uint32(b[8:]), uint64(le.Uint32(b[12:]))
		} else {
			b, err := c.le(40)
			if err != nil {
				return nil, err
			}
			in.inicio, in.tamanho, in.frag, in.fragPos = le.Uint64(b[0:]), le.Uint64(b[8:]), le.Uint32(b[28:]), le.Uint32(b[32:])
		}
		n := in.tamanho / uint64(fs.bloco)
		if in.frag == semFragmento && in.tamanho%uint64(fs.bloco) != 0 {
			n++
		}
		// A lista de blocos só é lida para arquivos que cabem na memória de
		// quem pede (ícones, atalhos): 64 MiB é teto folgado.
		if in.tamanho <= 64<<20 {
			b, err := c.le(int(n) * 4)
			if err != nil {
				return nil, err
			}
			in.blocos = make([]uint32, n)
			for i := range in.blocos {
				in.blocos[i] = le.Uint32(b[i*4:])
			}
		}
	case 3, 10: // link
		b, err := c.le(8)
		if err != nil {
			return nil, err
		}
		n := le.Uint32(b[4:])
		if n == 0 || n > 4096 {
			return nil, errors.New("squashfs estragado: link de tamanho impossível")
		}
		alvo, err := c.le(int(n))
		if err != nil {
			return nil, err
		}
		in.tipo, in.alvo = "link", string(alvo)
	default:
		in.tipo = "outro"
	}
	return in, nil
}

func (fs *FS) lista(in *inode) ([]Entrada, error) {
	if in.listaTam < 3 || in.listaTam > listagemMaxima {
		return nil, errors.New("squashfs estragado: pasta de tamanho impossível")
	}
	resta := int(in.listaTam) - 3
	if resta == 0 {
		return nil, nil
	}
	c, err := fs.cursor(fs.pastas, in.listaBloco, in.listaPos)
	if err != nil {
		return nil, err
	}
	le := binary.LittleEndian
	var out []Entrada
	for resta > 0 {
		cab, err := c.le(12)
		if err != nil {
			return nil, err
		}
		resta -= 12
		quantos, inicio := int(le.Uint32(cab[0:]))+1, uint64(le.Uint32(cab[4:]))
		if quantos > 256 {
			return nil, errors.New("squashfs estragado: cabeçalho de pasta impossível")
		}
		for i := 0; i < quantos; i++ {
			e, err := c.le(8)
			if err != nil {
				return nil, err
			}
			nome, err := c.le(int(le.Uint16(e[6:])) + 1)
			if err != nil {
				return nil, err
			}
			resta -= 8 + len(nome)
			if resta < 0 {
				return nil, errors.New("squashfs estragado: pasta maior que o tamanho dela")
			}
			tipo := map[uint16]string{1: "pasta", 2: "arquivo", 3: "link"}[le.Uint16(e[4:])]
			if tipo == "" {
				tipo = "outro"
			}
			out = append(out, Entrada{Nome: string(nome), Tipo: tipo, ref: inicio<<16 | uint64(le.Uint16(e[0:]))})
		}
	}
	return out, nil
}

// dado lê um bloco de dados (ou de fragmentos) que ocupa `entrada` no disco.
func (fs *FS) dado(pos int64, entrada uint32) ([]byte, error) {
	tam := entrada &^ naoComprimido
	if tam > fs.bloco {
		return nil, errors.New("squashfs estragado: bloco de dados grande demais")
	}
	dados := make([]byte, tam)
	if _, err := fs.r.ReadAt(dados, fs.base+pos); err != nil {
		return nil, errors.New("squashfs estragado: dados além do fim")
	}
	if entrada&naoComprimido != 0 {
		return dados, nil
	}
	return fs.descomprime(dados, int(fs.bloco))
}

func (fs *FS) conteudo(in *inode) ([]byte, error) {
	if in.blocos == nil && in.tamanho > 0 && in.tamanho > 64<<20 {
		return nil, errors.New("arquivo grande demais")
	}
	out := make([]byte, 0, in.tamanho)
	pos := int64(in.inicio)
	for _, e := range in.blocos {
		falta := in.tamanho - uint64(len(out))
		if falta == 0 {
			break
		}
		if e&^naoComprimido == 0 { // um buraco: zeros
			out = append(out, make([]byte, min(falta, uint64(fs.bloco)))...)
			continue
		}
		b, err := fs.dado(pos, e)
		if err != nil {
			return nil, err
		}
		pos += int64(e &^ naoComprimido)
		if uint64(len(b)) > falta {
			b = b[:falta]
		}
		out = append(out, b...)
	}
	if falta := in.tamanho - uint64(len(out)); falta > 0 {
		if in.frag == semFragmento {
			return nil, errors.New("squashfs estragado: arquivo menor que o tamanho dele")
		}
		b, err := fs.fragmento(in.frag)
		if err != nil {
			return nil, err
		}
		ini := uint64(in.fragPos)
		if ini+falta > uint64(len(b)) {
			return nil, errors.New("squashfs estragado: fragmento curto")
		}
		out = append(out, b[ini:ini+falta]...)
	}
	return out, nil
}

func (fs *FS) fragmento(indice uint32) ([]byte, error) {
	if indice >= fs.fragmentos || fs.fragTab < 96 {
		return nil, errors.New("squashfs estragado: fragmento que não existe")
	}
	var p [8]byte
	if _, err := fs.r.ReadAt(p[:], fs.base+fs.fragTab+int64(indice/512)*8); err != nil {
		return nil, errors.New("squashfs estragado: tabela de fragmentos além do fim")
	}
	onde := int64(binary.LittleEndian.Uint64(p[:]))
	if onde < 96 {
		return nil, errors.New("squashfs estragado: tabela de fragmentos fora do lugar")
	}
	bloco, _, err := fs.meta(fs.base + onde)
	if err != nil {
		return nil, err
	}
	i := int(indice%512) * 16
	if i+16 > len(bloco) {
		return nil, errors.New("squashfs estragado: tabela de fragmentos curta")
	}
	inicio := int64(binary.LittleEndian.Uint64(bloco[i:]))
	if inicio < 96 {
		return nil, errors.New("squashfs estragado: fragmento fora do lugar")
	}
	return fs.dado(inicio, binary.LittleEndian.Uint32(bloco[i+8:]))
}
