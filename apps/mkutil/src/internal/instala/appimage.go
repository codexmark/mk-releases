package instala

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"mkutil/internal/base"
	"mkutil/internal/squash"
)

// FichaAppImage é o bloco "appimage" do contrato.
type FichaAppImage struct {
	Tipo       int  `json:"tipo"`
	TemDesktop bool `json:"tem_desktop"`
	TemIcone   bool `json:"tem_icone"`
	Fuse       bool `json:"fuse"`
}

// tipoDeAppImage devolve 1 ou 2 se o arquivo é um ELF com a magia do AppImage
// ("AI" e o tipo, no deslocamento 8), ou 0.
func tipoDeAppImage(r io.ReaderAt) int {
	var cab [11]byte
	if _, err := r.ReadAt(cab[:], 0); err != nil {
		return 0
	}
	if !bytes.Equal(cab[:4], []byte{0x7f, 'E', 'L', 'F'}) || cab[8] != 'A' || cab[9] != 'I' {
		return 0
	}
	if cab[10] == 1 || cab[10] == 2 {
		return int(cab[10])
	}
	return 0
}

// fimDoELF diz onde o ELF acaba: é onde começa o squashfs de um AppImage do
// tipo 2 (o fim da tabela de seções).
func fimDoELF(r io.ReaderAt) (int64, error) {
	var c [64]byte
	if _, err := r.ReadAt(c[:], 0); err != nil {
		return 0, errors.New("ELF curto demais")
	}
	var ordem binary.ByteOrder = binary.LittleEndian
	if c[5] == 2 {
		ordem = binary.BigEndian
	}
	var shoff, tam, n uint64
	switch c[4] {
	case 1:
		shoff, tam, n = uint64(ordem.Uint32(c[0x20:])), uint64(ordem.Uint16(c[0x2E:])), uint64(ordem.Uint16(c[0x30:]))
	case 2:
		shoff, tam, n = ordem.Uint64(c[0x28:]), uint64(ordem.Uint16(c[0x3A:])), uint64(ordem.Uint16(c[0x3C:]))
	default:
		return 0, errors.New("ELF de classe desconhecida")
	}
	fim := shoff + tam*n
	if shoff == 0 || fim > 1<<40 {
		return 0, errors.New("ELF sem tabela de seções")
	}
	return int64(fim), nil
}

// precisaDeFuse2 diz se o runtime do AppImage é dos que carregam a
// libfuse.so.2: os ligados dinamicamente (têm PT_INTERP). O runtime novo, estático,
// traz o FUSE dentro. Na dúvida, precisa.
func precisaDeFuse2(r io.ReaderAt) bool {
	f, err := elf.NewFile(r)
	if err != nil {
		return true
	}
	for _, p := range f.Progs {
		if p.Type == elf.PT_INTERP {
			return true
		}
	}
	return false
}

// temFuse2 procura a libfuse.so.2 nas pastas de bibliotecas do sistema.
func temFuse2() bool {
	for _, pasta := range base.LibsDoSistema() {
		if achados, _ := filepath.Glob(filepath.Join(pasta, "libfuse.so.2*")); len(achados) > 0 {
			return true
		}
	}
	return false
}

// deDentro é o que deu para ler do squashfs de um AppImage, sem executá-lo.
type deDentro struct {
	desktop   map[string]string
	icone     []byte
	extensao  string // ".png", ".svg" ou ".xpm"
	motivo    string // por que não deu, para o aviso
	temAtalho bool
}

const (
	maximoDoAtalho = 256 << 10
	maximoDoIcone  = 4 << 20
)

// extensaoDeIcone reconhece o formato de um ícone pelo começo dele.
func extensaoDeIcone(b []byte) string {
	comeco := b
	if len(comeco) > 512 {
		comeco = comeco[:512]
	}
	switch {
	case bytes.HasPrefix(b, []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}):
		return ".png"
	case bytes.Contains(comeco, []byte("<svg")):
		return ".svg"
	case bytes.HasPrefix(bytes.TrimSpace(comeco), []byte("/* XPM */")):
		return ".xpm"
	}
	return ""
}

// leDeDentro abre o squashfs de um AppImage do tipo 2 e tira o atalho e o
// ícone. Nunca executa nada do arquivo.
func leDeDentro(f *os.File) deDentro {
	fim, err := fimDoELF(f)
	if err != nil {
		return deDentro{motivo: err.Error()}
	}
	fs, err := squash.Abre(f, fim)
	if err != nil {
		return deDentro{motivo: err.Error()}
	}
	raiz, err := fs.Lista("/")
	if err != nil {
		return deDentro{motivo: err.Error()}
	}
	var d deDentro
	for _, e := range raiz {
		if !strings.HasSuffix(e.Nome, ".desktop") || e.Tipo == "pasta" {
			continue
		}
		if b, err := fs.Le(e.Nome, maximoDoAtalho); err == nil {
			d.desktop, d.temAtalho = leDesktop(b), true
			break
		}
	}
	candidatos := []string{".DirIcon"}
	if nome := d.desktop["Icon"]; nome != "" && !strings.ContainsAny(nome, "/\\") && nome != "." && nome != ".." {
		for _, ext := range []string{".png", ".svg", ".xpm"} {
			candidatos = append(candidatos, nome+ext)
		}
		for _, tam := range []string{"256x256", "512x512", "128x128", "scalable", "96x96", "64x64", "48x48"} {
			for _, ext := range []string{".png", ".svg"} {
				candidatos = append(candidatos, "usr/share/icons/hicolor/"+tam+"/apps/"+nome+ext)
			}
		}
		candidatos = append(candidatos, "usr/share/pixmaps/"+nome+".png")
	}
	for _, c := range candidatos {
		b, err := fs.Le(c, maximoDoIcone)
		if err != nil {
			continue
		}
		if ext := extensaoDeIcone(b); ext != "" {
			d.icone, d.extensao = b, ext
			break
		}
	}
	return d
}
