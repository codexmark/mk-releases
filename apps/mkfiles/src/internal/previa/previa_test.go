package previa

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"testing"
	"unicode/utf8"

	"mkfiles/internal/mime"
	"mkfiles/internal/prova"
)

func motor(t *testing.T) *prova.Motor {
	t.Helper()
	m := prova.Novo(t, Ligar)
	m.Grava(".mime/globs2", strings.Join([]string{
		"50:text/plain:*.txt", "50:text/markdown:*.md", "50:application/json:*.json", "50:image/png:*.png",
		"50:image/svg+xml:*.svg", "50:audio/flac:*.flac", "50:video/mp4:*.mp4", "50:application/pdf:*.pdf",
		"50:application/zip:*.zip", "50:application/x-compressed-tar:*.tar.gz", "50:application/x-7z-compressed:*.7z",
		"50:application/x-shellscript:*.sh", "50:application/vnd.oasis.opendocument.text:*.odt", ""}, "\n"))
	t.Setenv("MKFILES_MIME", m.Casa+"/.mime")
	mime.Zerar()
	t.Cleanup(mime.Zerar)
	return m
}

func previa(m *prova.Motor, caminho string) prova.Linha {
	return m.Pede(`{"op":"previa","caminho":` + prova.J(caminho) + `}`)
}

func TestClasses(t *testing.T) {
	m := motor(t)
	casos := map[string]string{
		"foto.png": "imagem", "desenho.svg": "imagem", "som.flac": "audio", "filme.mp4": "video", "doc.pdf": "pdf",
		"pacote.zip": "compactado", "fontes.tar.gz": "compactado", "x.7z": "compactado",
		"nota.txt": "texto", "leia.md": "texto", "dados.json": "texto", "roda.sh": "texto", "carta.odt": "outro",
	}
	for nome, classe := range casos {
		r := previa(m, m.Grava("c/"+nome, "conteúdo"))
		it, _ := r["item"].(map[string]any)
		if r["classe"] != classe || it == nil || it["nome"] != nome {
			t.Errorf("%s: classe %v (esperada %s), item %v", nome, r["classe"], classe, it)
		}
		// Os campos do contrato estão sempre lá.
		for _, campo := range []string{"item", "classe", "texto", "cortado", "binario", "itens", "total"} {
			if _, tem := r[campo]; !tem {
				t.Errorf("%s: falta o campo %s", nome, campo)
			}
		}
		if (classe == "texto") != (r["texto"] == "conteúdo") {
			t.Errorf("%s: texto %v", nome, r["texto"])
		}
	}
}

func TestTexto(t *testing.T) {
	m := motor(t)
	// Sem extensão: o farejo decide.
	if r := previa(m, m.Grava("LEIAME", "olá\n")); r["classe"] != "texto" || r["texto"] != "olá\n" || r["cortado"] != false || r["binario"] != false {
		t.Errorf("texto sem extensão: %v", r)
	}
	// Binário com nome de texto: diz que é binário e não manda o conteúdo.
	if r := previa(m, m.Grava("falso.txt", "abc\x00def")); r["classe"] != "texto" || r["binario"] != true || r["texto"] != nil {
		t.Errorf("binário com nome de texto: %v", r)
	}
	if r := previa(m, m.Grava("latin1.txt", "a\xe7\xe3o")); r["binario"] != true || r["texto"] != nil {
		t.Errorf("texto que não é UTF-8: %v", r)
	}
	if r := previa(m, m.Grava("programa", "\x7fELF\x02\x01\x01\x00")); r["classe"] != "outro" || r["texto"] != nil {
		t.Errorf("binário sem extensão: %v", r)
	}
	if r := previa(m, m.Grava("vazio.txt", "")); r["classe"] != "texto" || r["texto"] != "" {
		t.Errorf("texto vazio: %v", r)
	}
	// Grande: só os primeiros 200 KiB, cortados numa fronteira de caractere.
	grande := strings.Repeat("ação ", Limite/6+1000) // "ação " tem 7 bytes: o corte cai no meio de um "ç"
	r := previa(m, m.Grava("grande.txt", grande))
	texto, _ := r["texto"].(string)
	if r["cortado"] != true || len(texto) > Limite || len(texto) < Limite-4 || !utf8.ValidString(texto) || !strings.HasPrefix(grande, texto) {
		t.Errorf("grande: cortado %v, %d bytes, válido %v", r["cortado"], len(texto), utf8.ValidString(texto))
	}
	// Exatamente no limite não é cortado.
	if r := previa(m, m.Grava("justo.txt", strings.Repeat("x", Limite))); r["cortado"] != false || len(r["texto"].(string)) != Limite {
		t.Errorf("no limite: %v", r["cortado"])
	}
}

func TestPasta(t *testing.T) {
	m := motor(t)
	p := m.Pasta("muitos")
	for i := 0; i < 120; i++ {
		m.Grava(fmt.Sprintf("muitos/a%03d.txt", i), "x")
	}
	r := previa(m, p)
	if r["classe"] != "pasta" || r["total"] != float64(120) || len(r["itens"].([]any)) != Primeiros || r["texto"] != nil {
		t.Errorf("pasta: classe %v, total %v, %d itens", r["classe"], r["total"], len(r["itens"].([]any)))
	}
	if it := r["itens"].([]any)[0].(map[string]any); it["mime"] != "text/plain" || !strings.HasPrefix(it["caminho"].(string), p+"/") {
		t.Errorf("o item de dentro: %v", it)
	}
	r = previa(m, m.Pasta("vazia"))
	if r["total"] != float64(0) || len(r["itens"].([]any)) != 0 {
		t.Errorf("pasta vazia: %v", r)
	}
	// Um link para pasta mostra a pasta do outro lado.
	os.Symlink(p, m.Casa+"/atalho")
	if r := previa(m, m.Casa+"/atalho"); r["classe"] != "pasta" || r["total"] != float64(120) {
		t.Errorf("link para pasta: %v %v", r["classe"], r["total"])
	}
	if os.Getuid() != 0 {
		fechada := m.Pasta("fechada")
		os.Chmod(fechada, 0)
		t.Cleanup(func() { os.Chmod(fechada, 0o755) })
		if r := previa(m, fechada); r["classe"] != "pasta" || r["total"] != nil || r["itens"] != nil {
			t.Errorf("pasta sem permissão: %v", r)
		}
	}
}

func TestOQueNaoSeLe(t *testing.T) {
	m := motor(t)
	// Um fifo com nome de texto travaria quem o abrisse: não é aberto.
	cano := m.Casa + "/cano.txt"
	if err := syscall.Mkfifo(cano, 0o600); err != nil {
		t.Fatal(err)
	}
	if r := previa(m, cano); r["classe"] != "outro" || r["texto"] != nil {
		t.Errorf("fifo: %v", r)
	}
	os.Symlink(cano, m.Casa+"/para-o-cano.txt")
	if r := previa(m, m.Casa+"/para-o-cano.txt"); r["classe"] != "outro" || r["texto"] != nil {
		t.Errorf("link para fifo: %v", r)
	}
	os.Symlink("/nao/existe", m.Casa+"/quebrado.txt")
	if r := previa(m, m.Casa+"/quebrado.txt"); r["classe"] != "outro" || r["item"].(map[string]any)["quebrado"] != true {
		t.Errorf("link quebrado: %v", r)
	}
	// Um link para um texto mostra o texto.
	os.Symlink(m.Grava("alvo.txt", "do outro lado"), m.Casa+"/link.txt")
	if r := previa(m, m.Casa+"/link.txt"); r["texto"] != "do outro lado" {
		t.Errorf("link para texto: %v", r)
	}
	for _, ruim := range []string{"relativo", m.Casa + "/../x", "/a\x00b", ""} {
		if r := previa(m, ruim); r["erro"] == nil {
			t.Errorf("caminho %q devia ser erro: %v", ruim, r)
		}
	}
	if r := previa(m, m.Casa+"/nao-ha"); r["motivo"] != "nao-existe" {
		t.Errorf("o que não existe: %v", r)
	}
}
