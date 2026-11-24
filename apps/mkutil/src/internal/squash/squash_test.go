package squash

import (
	"bytes"
	"encoding/binary"
	"errors"
	"reflect"
	"strings"
	"testing"

	"mkutil/internal/dubles"
	"mkutil/internal/squash/falso"
)

var grande = bytes.Repeat([]byte("0123456789abcdef"), 700) // 11200 bytes: dois blocos e um resto

func arvore() []falso.Arquivo {
	return []falso.Arquivo{
		{Caminho: "AppRun", Conteudo: []byte("#!/bin/sh\n")},
		{Caminho: "krita.desktop", Conteudo: []byte("[Desktop Entry]\nName=Krita\n")},
		{Caminho: ".DirIcon", Link: "krita.png"},
		{Caminho: "krita.png", Link: "usr/share/icons/krita.png"},
		{Caminho: "usr/share/icons/krita.png", Conteudo: grande},
		{Caminho: "usr/share/absoluto", Link: "/krita.desktop"},
		{Caminho: "usr/share/sobe", Link: "../../AppRun"},
		{Caminho: "usr/foge", Link: "../../../../etc/passwd"},
		{Caminho: "ciclo-a", Link: "ciclo-b"},
		{Caminho: "ciclo-b", Link: "ciclo-a"},
		{Caminho: "vazio", Conteudo: nil},
		{Caminho: "oca", Pasta: true},
	}
}

func abre(t *testing.T, img []byte, deslocamento int64) *FS {
	t.Helper()
	fs, err := Abre(bytes.NewReader(img), deslocamento)
	if err != nil {
		t.Fatal(err)
	}
	return fs
}

func confere(t *testing.T, fs *FS) {
	t.Helper()
	nomes, err := fs.Lista("/")
	if err != nil {
		t.Fatal(err)
	}
	var vistos []string
	for _, e := range nomes {
		vistos = append(vistos, e.Nome+":"+e.Tipo)
	}
	quer := []string{".DirIcon:link", "AppRun:arquivo", "ciclo-a:link", "ciclo-b:link", "krita.desktop:arquivo",
		"krita.png:link", "oca:pasta", "usr:pasta", "vazio:arquivo"}
	if !reflect.DeepEqual(vistos, quer) {
		t.Errorf("raiz:\n veio %v\n quer %v", vistos, quer)
	}
	le := func(caminho, quer string) {
		t.Helper()
		b, err := fs.Le(caminho, 1<<20)
		if err != nil {
			t.Errorf("%s: %v", caminho, err)
		} else if string(b) != quer {
			t.Errorf("%s: vieram %d bytes, queria %d", caminho, len(b), len(quer))
		}
	}
	le("krita.desktop", "[Desktop Entry]\nName=Krita\n")
	le("/usr/share/icons/krita.png", string(grande))
	le(".DirIcon", string(grande))                            // dois links seguidos
	le("usr/share/absoluto", "[Desktop Entry]\nName=Krita\n") // link absoluto: a raiz é a da imagem
	le("usr/share/sobe", "#!/bin/sh\n")
	le("usr/../AppRun", "#!/bin/sh\n")
	le("vazio", "")
	if l, err := fs.Lista("oca"); err != nil || len(l) != 0 {
		t.Errorf("pasta vazia: %v, %v", l, err)
	}
	if l, err := fs.Lista("usr/share"); err != nil || len(l) != 3 {
		t.Errorf("usr/share: %v, %v", l, err)
	}
}

func TestLeUmaImagemSemCompressao(t *testing.T) {
	confere(t, abre(t, falso.Monta(arvore(), falso.Opcoes{}), 0))
}

func TestLeUmaImagemGzipComFragmentos(t *testing.T) {
	confere(t, abre(t, falso.Monta(arvore(), falso.Opcoes{Comprime: falso.Zlib, Fragmentos: true}), 0))
}

func TestLeUmaImagemGzipSemFragmentos(t *testing.T) {
	confere(t, abre(t, falso.Monta(arvore(), falso.Opcoes{Comprime: falso.Zlib}), 0))
}

func TestAImagemPodeComecarNoMeioDoArquivo(t *testing.T) {
	// Num AppImage o squashfs vem depois do ELF.
	img := append(bytes.Repeat([]byte{0x7f}, 5000), falso.Monta(arvore(), falso.Opcoes{Comprime: falso.Zlib, Fragmentos: true})...)
	confere(t, abre(t, img, 5000))
}

func TestUmLinkNuncaSaiDaImagemNemGiraParaSempre(t *testing.T) {
	fs := abre(t, falso.Monta(arvore(), falso.Opcoes{}), 0)
	// "../../../../etc/passwd" a partir de usr/ para na raiz da imagem: lá não há etc.
	if _, err := fs.Le("usr/foge", 1<<20); !errors.Is(err, ErrNaoExiste) {
		t.Errorf("link que sobe demais: %v", err)
	}
	if _, err := fs.Le("ciclo-a", 1<<20); err == nil || !strings.Contains(err.Error(), "links demais") {
		t.Errorf("ciclo: %v", err)
	}
	if _, err := fs.Le("nao-existe", 1<<20); !errors.Is(err, ErrNaoExiste) {
		t.Errorf("ausente: %v", err)
	}
	if _, err := fs.Le("usr", 1<<20); err == nil {
		t.Error("ler uma pasta como arquivo devia falhar")
	}
	if _, err := fs.Lista("AppRun"); err == nil {
		t.Error("listar um arquivo devia falhar")
	}
}

func TestLeRecusaArquivoMaiorQueOTeto(t *testing.T) {
	fs := abre(t, falso.Monta(arvore(), falso.Opcoes{}), 0)
	if _, err := fs.Le("usr/share/icons/krita.png", 100); err == nil || !strings.Contains(err.Error(), "grande demais") {
		t.Errorf("veio %v", err)
	}
}

func TestZstdEXzPassamPeloComandoDoPATH(t *testing.T) {
	// O dublê "descomprime" devolvendo o que recebeu: os blocos da imagem de
	// teste vão marcados como comprimidos, mas são o próprio conteúdo.
	identidade := func(b []byte) []byte { return b }
	for nome, id := range map[string]uint16{"zstd": 6, "xz": 4} {
		m := dubles.Nova(t)
		img := falso.Monta(arvore(), falso.Opcoes{Compressor: id, Comprime: identidade, Fragmentos: true})
		fs := abre(t, img, 0)
		if _, err := fs.Le("krita.desktop", 1<<20); err == nil || !strings.Contains(err.Error(), "não achei o "+nome) {
			t.Errorf("%s ausente: %v", nome, err)
		}
		m.Comando(nome, `[ "$1" = -dc ] || exit 9; exec /bin/cat`)
		confere(t, fs)
		if c := m.Chamadas(nome); len(c) == 0 || !reflect.DeepEqual(c[0], []string{"-dc"}) {
			t.Errorf("%s chamado com %q", nome, c)
		}
	}
}

func TestCompressorQueNaoSeiLerEErroClaro(t *testing.T) {
	dubles.Nova(t)
	img := falso.Monta(arvore(), falso.Opcoes{Compressor: 3, Comprime: func(b []byte) []byte { return b }})
	fs, err := Abre(bytes.NewReader(img), 0)
	if err == nil {
		_, err = fs.Lista("/")
	}
	if err == nil || !strings.Contains(err.Error(), "compressão") {
		t.Errorf("veio %v", err)
	}
}

func TestImagemEstragadaNaoDerrubaNemPedeMemoriaSemFim(t *testing.T) {
	boa := falso.Monta(arvore(), falso.Opcoes{Comprime: falso.Zlib, Fragmentos: true})
	if _, err := Abre(bytes.NewReader([]byte("curto")), 0); err == nil {
		t.Error("arquivo curto devia falhar")
	}
	if _, err := Abre(bytes.NewReader(bytes.Repeat([]byte{0}, 200)), 0); err == nil {
		t.Error("sem a magia devia falhar")
	}
	velha := append([]byte{}, boa...)
	binary.LittleEndian.PutUint16(velha[28:], 3)
	if _, err := Abre(bytes.NewReader(velha), 0); err == nil || !strings.Contains(err.Error(), "versão") {
		t.Errorf("versão 3: %v", err)
	}
	blocao := append([]byte{}, boa...)
	binary.LittleEndian.PutUint32(blocao[12:], 1<<30)
	if _, err := Abre(bytes.NewReader(blocao), 0); err == nil {
		t.Error("bloco de 1 GB devia falhar")
	}
	// Cada byte trocado, um por vez, e cada corte: nunca um pânico. O que
	// vier (erro ou conteúdo errado) tanto faz.
	olha := func(img []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("pânico: %v", r)
			}
		}()
		fs, err := Abre(bytes.NewReader(img), 0)
		if err != nil {
			return
		}
		_, _ = fs.Lista("/")
		_, _ = fs.Lista("usr/share")
		_, _ = fs.Le(".DirIcon", 1<<20)
		_, _ = fs.Le("krita.desktop", 1<<20)
	}
	for i := 0; i < len(boa); i++ {
		for _, v := range []byte{0xff, boa[i] ^ 0x55} {
			ruim := append([]byte{}, boa...)
			ruim[i] = v
			olha(ruim)
		}
	}
	for i := 0; i < len(boa); i += 7 {
		olha(boa[:i])
	}
}
