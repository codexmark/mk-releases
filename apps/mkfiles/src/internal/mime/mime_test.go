package mime

import (
	"os"
	"path/filepath"
	"testing"
)

// Um banco de mentira, pequeno, com os casos que importam.
const globs = `# comentário
50:text/plain:*.txt
50:image/jpeg:*.jpg
50:image/jpeg:*.jpeg
50:image/png:*.png
50:application/gzip:*.gz
50:application/x-compressed-tar:*.tar.gz
50:text/x-makefile:makefile
50:text/x-csrc:*.c
50:text/x-c++src:*.C:cs
60:text/x-readme:README*
50:application/x-trash:*~
10:text/x-fraco:*.dup
80:text/x-forte:*.dup
`

func comBanco(t *testing.T) string {
	t.Helper()
	raiz := t.TempDir()
	os.WriteFile(filepath.Join(raiz, "globs2"), []byte(globs), 0o644)
	os.WriteFile(filepath.Join(raiz, "generic-icons"), []byte("text/x-csrc:text-x-script\napplication/gzip:package-x-generic\n"), 0o644)
	t.Setenv("MKFILES_MIME", raiz)
	Zerar()
	t.Cleanup(Zerar)
	return raiz
}

func TestDoNome(t *testing.T) {
	comBanco(t)
	casos := []struct{ nome, mime string }{
		{"a.txt", "text/plain"},
		{"FOTO.JPG", "image/jpeg"},
		{"x.tar.gz", "application/x-compressed-tar"}, // o padrão mais longo ganha
		{"x.gz", "application/gzip"},
		{"Makefile", "text/x-makefile"},
		{"makefile", "text/x-makefile"},
		{"a.c", "text/x-csrc"},
		{"a.C", "text/x-c++src"}, // o padrão exato ("cs") só casa na caixa certa
		{"README.md", "text/x-readme"},
		{"velho~", "application/x-trash"},
		{"a.dup", "text/x-forte"}, // o peso maior ganha
		{".txt", ""},              // um arquivo oculto sem extensão não é ".txt"
		{"sem", ""},
		{"a.desconhecido", ""},
	}
	for _, c := range casos {
		m, achou := DoNome(c.nome)
		if m != c.mime || achou != (c.mime != "") {
			t.Errorf("DoNome(%q) = %q, %v; esperado %q", c.nome, m, achou, c.mime)
		}
	}
}

func TestDoConteudo(t *testing.T) {
	casos := []struct{ dados, mime string }{
		{"", Vazio},
		{"\x89PNG\r\n\x1a\n....", "image/png"},
		{"\xff\xd8\xff\xe0..", "image/jpeg"},
		{"GIF89a..", "image/gif"},
		{"%PDF-1.7", "application/pdf"},
		{"PK\x03\x04..", "application/zip"},
		{"\x1f\x8b\x08", "application/gzip"},
		{"\x7fELF\x02", "application/x-executable"},
		{"#!/bin/sh\necho", "application/x-shellscript"},
		{"olá, mundo\n\tação", Texto},
		{"abc\x00def", Binario},
		{"\xff\xfe\x00\x01", Binario},
	}
	for _, c := range casos {
		if m := DoConteudo([]byte(c.dados)); m != c.mime {
			t.Errorf("DoConteudo(%q) = %q; esperado %q", c.dados, m, c.mime)
		}
	}
	// Um caractere de vários bytes cortado no limite do farejo ainda é texto.
	if m := DoConteudo([]byte("aç")[:2]); m != Texto {
		t.Errorf("caractere cortado no fim: %q", m)
	}
}

func TestDoArquivo(t *testing.T) {
	comBanco(t)
	pasta := t.TempDir()
	grava := func(nome, dados string) string {
		c := filepath.Join(pasta, nome)
		os.WriteFile(c, []byte(dados), 0o644)
		return c
	}
	if m := DoArquivo(grava("nota.txt", "\x89PNG\r\n\x1a\n"), 8, true); m != "text/plain" {
		t.Errorf("o nome vem antes do conteúdo: %q", m)
	}
	if m := DoArquivo(grava("script", "#!/bin/sh\n"), 10, false); m != "application/x-shellscript" {
		t.Errorf("sem extensão, fareja: %q", m)
	}
	if m := DoArquivo(grava("foto.xyz", "\x89PNG\r\n\x1a\n"), 8, false); m != Binario {
		t.Errorf("com extensão desconhecida o listar não abre o arquivo: %q", m)
	}
	if m := DoArquivo(grava("foto.xyz", "\x89PNG\r\n\x1a\n"), 8, true); m != "image/png" {
		t.Errorf("com sempre, fareja: %q", m)
	}
	if m := DoArquivo(grava("vazio", ""), 0, true); m != Vazio {
		t.Errorf("vazio: %q", m)
	}
	if m := DoArquivo(filepath.Join(pasta, "nao-existe"), 5, true); m != Binario {
		t.Errorf("arquivo que não abre: %q", m)
	}
}

func TestIcone(t *testing.T) {
	comBanco(t)
	casos := []struct{ mime, icone, reserva string }{
		{"text/plain", "text-plain", "text-x-generic"},
		{"text/x-csrc", "text-x-csrc", "text-x-script"},
		{"application/gzip", "application-gzip", "package-x-generic"},
		{"image/png", "image-png", "image-x-generic"},
		{Pasta, "folder", "folder"},
		{"torto", "application-octet-stream", "application-x-generic"},
	}
	for _, c := range casos {
		i, r := Icone(c.mime)
		if i != c.icone || r != c.reserva {
			t.Errorf("Icone(%q) = %q, %q; esperado %q, %q", c.mime, i, r, c.icone, c.reserva)
		}
	}
}

func TestFamilia(t *testing.T) {
	casos := map[string]string{
		"image/png": "imagem", "video/mp4": "video", "audio/flac": "audio", "text/x-csrc": "texto",
		"application/pdf": "documento", "application/vnd.oasis.opendocument.text": "documento",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": "documento",
		"application/zip": "", Pasta: "",
	}
	for m, f := range casos {
		if Familia(m) != f {
			t.Errorf("Familia(%q) = %q; esperado %q", m, Familia(m), f)
		}
	}
}

func TestSemBanco(t *testing.T) {
	t.Setenv("MKFILES_MIME", filepath.Join(t.TempDir(), "nao-ha"))
	Zerar()
	t.Cleanup(Zerar)
	if _, achou := DoNome("a.txt"); achou {
		t.Error("sem banco nada casa pelo nome")
	}
	if i, r := Icone("text/plain"); i != "text-plain" || r != "text-x-generic" {
		t.Errorf("sem banco o ícone ainda sai: %q %q", i, r)
	}
}
