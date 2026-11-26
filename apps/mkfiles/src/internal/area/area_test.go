package area

import (
	"os"
	"strings"
	"testing"
	"time"

	"mkfiles/internal/prova"
)

// O wl-copy de mentira guarda o tipo e o texto em arquivos da casa; o
// wl-paste de mentira responde com o que há em .area/<tipo> (a "/" vira "_").
func motor(t *testing.T) *prova.Motor {
	t.Helper()
	m := prova.Novo(t, Ligar)
	t.Setenv("MKFILES_MIME", m.Casa+"/.mime")
	copia := m.Grava(".bin/wl-copy", `#!/bin/sh
echo "$*" >"$HOME/.copiado-args"
cat >"$HOME/.copiado"
[ -f "$HOME/.copia-falha" ] && { echo "sem compositor" >&2; exit 1; }
exit 0
`)
	cola := m.Grava(".bin/wl-paste", `#!/bin/sh
echo "$*" >>"$HOME/.cola-args"
[ -d "$HOME/.area" ] || { echo "Nothing is copied" >&2; exit 1; }
if [ "$1" = "--list-types" ]; then cat "$HOME/.area/tipos"; exit 0; fi
tipo=""
while [ $# -gt 0 ]; do [ "$1" = "--type" ] && tipo=$2; shift; done
[ -f "$HOME/.dorme" ] && sleep 5
cat "$HOME/.area/$(echo "$tipo" | tr '/;=' '___')"
`)
	os.Chmod(copia, 0o755)
	os.Chmod(cola, 0o755)
	t.Setenv("MKFILES_WLCOPY", copia)
	t.Setenv("MKFILES_WLPASTE", cola)
	return m
}

func le(m *prova.Motor, nome string) string {
	b, _ := os.ReadFile(m.Casa + "/" + nome)
	return string(b)
}

func TestPor(t *testing.T) {
	m := motor(t)
	texto := "cut\nfile:///home/ana/a%20b.txt\nfile:///home/ana/$(x);`y`"
	r := m.Pede(`{"op":"area-por","tipo":"x-special/gnome-copied-files","texto":` + prova.J(texto) + `}`)
	if r["posto"] != true || le(m, ".copiado") != texto || le(m, ".copiado-args") != "--type x-special/gnome-copied-files\n" {
		t.Errorf("por: %v, args %q, texto %q", r, le(m, ".copiado-args"), le(m, ".copiado"))
	}
	for _, tipo := range []string{"text/uri-list", "text/plain"} {
		if r := m.Pede(`{"op":"area-por","tipo":"` + tipo + `","texto":"x"}`); r["posto"] != true {
			t.Errorf("%s: %v", tipo, r)
		}
	}
	os.Remove(m.Casa + "/.copiado")
	for _, ruim := range []string{"image/png", "", "text/plain --foreground", "-o"} {
		if r := m.Pede(`{"op":"area-por","tipo":` + prova.J(ruim) + `,"texto":"x"}`); r["campo"] != "tipo" {
			t.Errorf("tipo %q devia ser recusado: %v", ruim, r)
		}
	}
	if r := m.Pede(`{"op":"area-por","tipo":"text/plain","texto":` + prova.J(strings.Repeat("x", LimiteDoTexto+1)) + `}`); r["erro"] == nil {
		t.Error("texto grande demais")
	}
	if le(m, ".copiado") != "" {
		t.Error("um pedido recusado chegou ao wl-copy")
	}
	m.Grava(".copia-falha", "")
	if r := m.Pede(`{"op":"area-por","tipo":"text/plain","texto":"x"}`); r["erro"] != "o wl-copy falhou: sem compositor" {
		t.Errorf("wl-copy que falha: %v", r)
	}
	t.Setenv("MKFILES_WLCOPY", m.Casa+"/.bin/nao-ha")
	if r := m.Pede(`{"op":"area-por","tipo":"text/plain","texto":"x"}`); r["erro"] == nil {
		t.Error("sem wl-copy")
	}
}

func TestLer(t *testing.T) {
	m := motor(t)
	// Nada copiado.
	r := m.Pede(`{"op":"area-ler"}`)
	if r["tipo"] != nil || r["texto"] != nil || len(r["tipos"].([]any)) != 0 {
		t.Errorf("vazia: %v", r)
	}
	// A ordem de preferência: gnome-copied-files, uri-list, imagem, texto.
	m.Grava(".area/tipos", "text/plain;charset=utf-8\ntext/uri-list\nx-special/gnome-copied-files\nimage/png\n")
	m.Grava(".area/x-special_gnome-copied-files", "cut\nfile:///a")
	m.Grava(".area/text_uri-list", "file:///a\r\n")
	m.Grava(".area/text_plain_charset_utf-8", "olá")
	r = m.Pede(`{"op":"area-ler"}`)
	if r["tipo"] != "x-special/gnome-copied-files" || r["texto"] != "cut\nfile:///a" || len(r["tipos"].([]any)) != 4 {
		t.Errorf("gnome-copied-files primeiro: %v", r)
	}
	if !strings.Contains(le(m, ".cola-args"), "--no-newline --type x-special/gnome-copied-files\n") {
		t.Errorf("a linha do wl-paste: %q", le(m, ".cola-args"))
	}
	m.Grava(".area/tipos", "text/plain;charset=utf-8\ntext/uri-list\nimage/png\n")
	if r = m.Pede(`{"op":"area-ler"}`); r["tipo"] != "text/uri-list" || r["texto"] != "file:///a\r\n" {
		t.Errorf("uri-list: %v", r)
	}
	// De imagem não vem texto (e o wl-paste nem é chamado para ela).
	m.Grava(".area/tipos", "image/png\ntext/plain\n")
	os.Remove(m.Casa + "/.cola-args")
	if r = m.Pede(`{"op":"area-ler"}`); r["tipo"] != "image/png" || r["texto"] != nil || strings.Contains(le(m, ".cola-args"), "--type") {
		t.Errorf("imagem: %v / %q", r, le(m, ".cola-args"))
	}
	// Texto, com o nome que ele tiver; o tipo que a janela vê é sempre text/plain.
	for _, nome := range []string{"text/plain;charset=utf-8", "UTF8_STRING"} {
		m.Grava(".area/tipos", "TARGETS\n"+nome+"\n")
		m.Grava(".area/UTF8_STRING", "olá")
		if r = m.Pede(`{"op":"area-ler"}`); r["tipo"] != "text/plain" || r["texto"] != "olá" {
			t.Errorf("texto como %s: %v", nome, r)
		}
	}
	// Um tipo que não interessa.
	m.Grava(".area/tipos", "application/x-coisa\n")
	if r = m.Pede(`{"op":"area-ler"}`); r["tipo"] != nil || r["tipos"].([]any)[0] != "application/x-coisa" {
		t.Errorf("tipo desconhecido: %v", r)
	}
	// Texto grande demais.
	m.Grava(".area/tipos", "text/plain\n")
	m.Grava(".area/text_plain", strings.Repeat("x", LimiteDoTexto+10))
	if r = m.Pede(`{"op":"area-ler"}`); r["erro"] == nil {
		t.Error("texto grande demais")
	}
	// O wl-paste que não volta é morto no prazo.
	m.Grava(".area/text_plain", "x")
	m.Grava(".dorme", "")
	prazoDeLer = 300 * time.Millisecond
	t.Cleanup(func() { prazoDeLer = 5 * time.Second })
	antes := time.Now()
	if r = m.Pede(`{"op":"area-ler"}`); r["erro"] == nil || time.Since(antes) > 3*time.Second {
		t.Errorf("prazo: %v em %v", r, time.Since(antes))
	}
}

func TestGravar(t *testing.T) {
	m := motor(t)
	pasta := m.Pasta("p")
	png := "\x89PNG\r\n\x1a\n\x00\x01binário"
	m.Grava(".area/tipos", "image/png\ntext/plain;charset=utf-8\n")
	m.Grava(".area/image_png", png)
	m.Grava(".area/text_plain_charset_utf-8", "um texto\ncolado")

	r := m.Pede(`{"op":"area-gravar","destino":` + prova.J(pasta) + `,"nome":"Imagem colada.png","tipo":"image/png"}`)
	it, _ := r["item"].(map[string]any)
	if it == nil || it["caminho"] != pasta+"/Imagem colada.png" || le(m, "p/Imagem colada.png") != png || it["mime"] != "image/png" {
		t.Fatalf("gravar imagem: %v", r)
	}
	r = m.Pede(`{"op":"area-gravar","destino":` + prova.J(pasta) + `,"nome":"nota","tipo":"text/plain"}`)
	if r["item"] == nil || le(m, "p/nota") != "um texto\ncolado" {
		t.Errorf("gravar texto: %v", r)
	}
	// Nome ocupado é erro do campo, e o arquivo que estava lá fica intacto.
	r = m.Pede(`{"op":"area-gravar","destino":` + prova.J(pasta) + `,"nome":"nota","tipo":"image/png"}`)
	if r["campo"] != "nome" || le(m, "p/nota") != "um texto\ncolado" {
		t.Errorf("nome ocupado: %v", r)
	}
	// Área vazia para aquele tipo: nada é criado.
	m.Grava(".area/tipos", "text/plain\n")
	m.Grava(".area/text_plain", "")
	for _, pedido := range []string{
		`"nome":"x.png","tipo":"image/png"`,  // não há imagem
		`"nome":"x.txt","tipo":"text/plain"`, // há, mas vazia
		`"nome":"x.bin","tipo":"application/zip"`,
		`"nome":"a/b","tipo":"text/plain"`,
		`"nome":"..","tipo":"text/plain"`,
		`"nome":"","tipo":"text/plain"`,
	} {
		if r := m.Pede(`{"op":"area-gravar","destino":` + prova.J(pasta) + `,` + pedido + `}`); r["erro"] == nil {
			t.Errorf("devia falhar: %s -> %v", pedido, r)
		}
	}
	for _, destino := range []string{"relativa", m.Casa + "/nao-ha", pasta + "/nota", pasta + "/../p"} {
		if r := m.Pede(`{"op":"area-gravar","destino":` + prova.J(destino) + `,"nome":"y","tipo":"text/plain"}`); r["erro"] == nil {
			t.Errorf("destino %q devia falhar", destino)
		}
	}
	if e, _ := os.ReadDir(pasta); len(e) != 2 {
		t.Errorf("um pedido que falhou deixou arquivo: %v", e)
	}
}
