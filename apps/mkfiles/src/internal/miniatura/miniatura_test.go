package miniatura

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mkfiles/internal/base"
	"mkfiles/internal/mime"
	"mkfiles/internal/prova"
)

func motor(t *testing.T) *prova.Motor {
	t.Helper()
	m := prova.Novo(t, Ligar)
	m.Grava(".mime/globs2", "50:image/png:*.png\n50:image/jpeg:*.jpg\n50:image/gif:*.gif\n50:video/mp4:*.mp4\n50:application/pdf:*.pdf\n50:text/plain:*.txt\n")
	t.Setenv("MKFILES_MIME", m.Casa+"/.mime")
	mime.Zerar()
	t.Cleanup(mime.Zerar)
	t.Setenv("MKFILES_MINIATURIZADORES", m.Pasta(".miniaturizadores"))
	return m
}

// imagem pinta um retângulo: a metade esquerda vermelha, a direita azul.
func imagem(l, a int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, l, a))
	for y := 0; y < a; y++ {
		for x := 0; x < l; x++ {
			if x < l/2 {
				img.SetNRGBA(x, y, color.NRGBA{255, 0, 0, 255})
			} else {
				img.SetNRGBA(x, y, color.NRGBA{0, 0, 255, 255})
			}
		}
	}
	return img
}

func gravaPNG(t *testing.T, caminho string, l, a int) {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, imagem(l, a)); err != nil {
		t.Fatal(err)
	}
	os.MkdirAll(filepath.Dir(caminho), 0o755)
	if err := os.WriteFile(caminho, b.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func pede(m *prova.Motor, tamanho, grupo string, caminhos ...string) {
	r := m.Pede(`{"op":"miniatura","caminhos":` + prova.J(caminhos) + `,"tamanho":"` + tamanho + `","grupo":"` + grupo + `","prioridade":0}`)
	if r["aceitos"] != float64(len(caminhos)) {
		panic(fmt.Sprintf("miniatura: %v", r))
	}
}

func de(caminho string) func(prova.Linha) bool {
	return func(l prova.Linha) bool { return l["caminho"] == caminho }
}

func md5De(caminho string) string {
	s := md5.Sum([]byte(base.URI(caminho)))
	return hex.EncodeToString(s[:])
}

func TestGeraPNGJPEGeGIF(t *testing.T) {
	m := motor(t)
	foto := m.Casa + "/Imagens/minha foto.png"
	gravaPNG(t, foto, 400, 200)
	jpg := m.Casa + "/Imagens/b.jpg"
	var b bytes.Buffer
	jpeg.Encode(&b, imagem(300, 600), nil)
	os.WriteFile(jpg, b.Bytes(), 0o644)
	gifa := m.Casa + "/Imagens/c.gif"
	b.Reset()
	gif.Encode(&b, imagem(64, 32), nil)
	os.WriteFile(gifa, b.Bytes(), 0o644)

	pede(m, "normal", "g", foto, jpg, gifa)
	e := m.Evento("miniatura", de(foto))
	arq, _ := e["arquivo"].(string)
	quer := m.Casa + "/.cache/thumbnails/normal/" + md5De(foto) + ".png"
	if arq != quer || e["tamanho"] != "normal" {
		t.Fatalf("o arquivo é o md5 do URI: %v (esperado %s)", e, quer)
	}
	dados, _ := os.ReadFile(arq)
	info, _ := os.Stat(foto)
	textos := textosDe(dados)
	if textos["Thumb::URI"] != "file://"+m.Casa+"/Imagens/minha%20foto.png" || textos["Thumb::MTime"] != fmt.Sprint(info.ModTime().Unix()) {
		t.Errorf("blocos tEXt: %v", textos)
	}
	// É um PNG que qualquer um abre, na proporção, com a caixa de 128.
	mini, err := png.Decode(bytes.NewReader(dados))
	if err != nil {
		t.Fatalf("a miniatura não é um PNG válido: %v", err)
	}
	if mini.Bounds().Dx() != 128 || mini.Bounds().Dy() != 64 {
		t.Errorf("medidas: %v", mini.Bounds())
	}
	if r, _, bl, _ := mini.At(10, 30).RGBA(); r>>8 < 250 || bl>>8 > 5 {
		t.Errorf("a metade esquerda é vermelha: %v", mini.At(10, 30))
	}
	if r, _, bl, _ := mini.At(120, 30).RGBA(); bl>>8 < 250 || r>>8 > 5 {
		t.Errorf("a metade direita é azul: %v", mini.At(120, 30))
	}
	// Arquivo 0600 em pasta 0700, e sem temporário sobrando.
	ai, _ := os.Stat(arq)
	pi, _ := os.Stat(filepath.Dir(arq))
	if ai.Mode().Perm() != 0o600 || pi.Mode().Perm() != 0o700 {
		t.Errorf("modos: arquivo %v, pasta %v", ai.Mode(), pi.Mode())
	}

	e = m.Evento("miniatura", de(jpg))
	dados, _ = os.ReadFile(e["arquivo"].(string))
	if mini, err = png.Decode(bytes.NewReader(dados)); err != nil || mini.Bounds().Dx() != 64 || mini.Bounds().Dy() != 128 {
		t.Errorf("JPEG em pé: %v %v", err, mini.Bounds())
	}
	// Uma imagem menor que a caixa não é ampliada.
	e = m.Evento("miniatura", de(gifa))
	dados, _ = os.ReadFile(e["arquivo"].(string))
	if mini, err = png.Decode(bytes.NewReader(dados)); err != nil || mini.Bounds().Dx() != 64 || mini.Bounds().Dy() != 32 {
		t.Errorf("GIF pequeno: %v %v", err, mini.Bounds())
	}
	if e, _ := os.ReadDir(m.Casa + "/.cache/thumbnails/normal"); len(e) != 3 {
		t.Errorf("sobrou temporário: %v", e)
	}

	// O tamanho large vai para a outra pasta, com 256.
	pede(m, "large", "g", foto)
	e = m.Evento("miniatura", de(foto))
	if e["arquivo"] != m.Casa+"/.cache/thumbnails/large/"+md5De(foto)+".png" || e["tamanho"] != "large" {
		t.Errorf("large: %v", e)
	}
	dados, _ = os.ReadFile(e["arquivo"].(string))
	if mini, _ = png.Decode(bytes.NewReader(dados)); mini.Bounds().Dx() != 256 {
		t.Errorf("large: %v", mini.Bounds())
	}
}

func TestReusaPelaMtime(t *testing.T) {
	m := motor(t)
	foto := m.Casa + "/a.png"
	gravaPNG(t, foto, 300, 300)
	antiga := time.Now().Add(-time.Hour).Truncate(time.Second)
	os.Chtimes(foto, antiga, antiga)
	pede(m, "normal", "g", foto)
	arq := m.Evento("miniatura", de(foto))["arquivo"].(string)
	// Marca a miniatura: se for refeita, a marca some.
	marca := time.Now().Add(-30 * time.Minute)
	os.Chtimes(arq, marca, marca)
	pede(m, "normal", "g", foto)
	m.Evento("miniatura", de(foto))
	if info, _ := os.Stat(arq); !info.ModTime().Equal(marca) {
		t.Error("a miniatura que valia foi refeita")
	}
	// O arquivo mudou (a mtime é outra): refaz.
	gravaPNG(t, foto, 100, 300)
	nova := antiga.Add(10 * time.Minute)
	os.Chtimes(foto, nova, nova)
	pede(m, "normal", "g", foto)
	m.Evento("miniatura", de(foto))
	dados, _ := os.ReadFile(arq)
	if textosDe(dados)["Thumb::MTime"] != fmt.Sprint(nova.Unix()) {
		t.Errorf("não refez: %v", textosDe(dados))
	}
	if mini, _ := png.Decode(bytes.NewReader(dados)); mini.Bounds().Dx() != 43 {
		t.Errorf("a miniatura nova tem a imagem nova: %v", mini.Bounds())
	}
	// Uma miniatura de outro arquivo no mesmo lugar (URI diferente) não vale.
	outro, _ := codifica(imagem(5, 5), textosPara("file:///outro.png", nova.Unix()))
	os.WriteFile(arq, outro, 0o600)
	pede(m, "normal", "g", foto)
	m.Evento("miniatura", de(foto))
	dados, _ = os.ReadFile(arq)
	if textosDe(dados)["Thumb::URI"] != base.URI(foto) {
		t.Error("aceitou a miniatura de outro URI")
	}
}

func TestFalha(t *testing.T) {
	m := motor(t)
	// Um "PNG" que não abre: a falha é anotada em fail/mkfiles e respeitada.
	ruim := m.Grava("quebrada.png", "\x89PNG\r\n\x1a\nisto não é imagem")
	pede(m, "normal", "g", ruim)
	if e := m.Evento("miniatura", de(ruim)); e["arquivo"] != nil {
		t.Fatalf("imagem quebrada: %v", e)
	}
	marca := m.Casa + "/.cache/thumbnails/fail/mkfiles/" + md5De(ruim) + ".png"
	dados, err := os.ReadFile(marca)
	if err != nil || textosDe(dados)["Thumb::URI"] != base.URI(ruim) {
		t.Fatalf("a falha não foi anotada: %v", err)
	}
	if _, err := os.Stat(m.Casa + "/.cache/thumbnails/normal/" + md5De(ruim) + ".png"); err == nil {
		t.Error("uma falha deixou miniatura")
	}

	// Falha anotada por outro programa, com a mtime de agora: nem tenta.
	boa := m.Casa + "/boa.png"
	gravaPNG(t, boa, 200, 200)
	info, _ := os.Stat(boa)
	alheia, _ := codifica(imagem(1, 1), textosPara(base.URI(boa), info.ModTime().Unix()))
	m.Grava(".cache/thumbnails/fail/gnome-thumbnail-factory/"+md5De(boa)+".png", string(alheia))
	pede(m, "normal", "g", boa)
	if e := m.Evento("miniatura", de(boa)); e["arquivo"] != nil {
		t.Errorf("a falha anotada não foi respeitada: %v", e)
	}
	// O arquivo mudou: a falha velha não vale mais.
	depois := info.ModTime().Add(time.Minute)
	os.Chtimes(boa, depois, depois)
	pede(m, "normal", "g", boa)
	if e := m.Evento("miniatura", de(boa)); e["arquivo"] == nil {
		t.Errorf("com a mtime nova tenta de novo: %v", e)
	}

	// O que não tem como: pasta, arquivo que não existe, vazio, tipo sem miniaturizador.
	texto := m.Grava("nota.txt", "olá")
	for _, c := range []string{m.Pasta("pasta"), m.Casa + "/nao-ha.png", m.Grava("vazio.png", ""), texto} {
		pede(m, "normal", "g", c)
		if e := m.Evento("miniatura", de(c)); e["arquivo"] != nil {
			t.Errorf("%s não devia ter miniatura: %v", c, e)
		}
	}
	// Sem miniaturizador não é uma falha a anotar.
	if _, err := os.Stat(m.Casa + "/.cache/thumbnails/fail/mkfiles/" + md5De(texto) + ".png"); err == nil {
		t.Error("tipo sem miniaturizador foi anotado como falha")
	}
	for _, ruim := range []string{
		`{"op":"miniatura","caminhos":["relativo.png"],"tamanho":"normal"}`,
		`{"op":"miniatura","caminhos":[],"tamanho":"normal"}`,
		`{"op":"miniatura","caminhos":["/a/../b.png"]}`,
		`{"op":"miniatura","caminhos":["/a.png"],"tamanho":"enorme"}`,
	} {
		if r := m.Pede(ruim); r["erro"] == nil {
			t.Errorf("devia falhar: %s", ruim)
		}
	}
}

// miniaturizador escreve um .thumbnailer que roda um script da casa.
func miniaturizador(t *testing.T, m *prova.Motor, nome, tipos, corpo string) {
	t.Helper()
	script := m.Grava(".bin/"+nome, "#!/bin/sh\necho \"$*\" >>\"$HOME/.chamadas\"\n"+corpo+"\n")
	os.Chmod(script, 0o755)
	m.Grava(".miniaturizadores/"+nome+".thumbnailer",
		"[Thumbnailer Entry]\nTryExec="+script+"\nExec=\""+script+"\" -s %s %u %i %o\nMimeType="+tipos+"\n")
}

func TestMiniaturizadorDeFora(t *testing.T) {
	m := motor(t)
	gravaPNG(t, m.Casa+"/.modelo.png", 128, 72)
	miniaturizador(t, m, "video", "video/mp4;video/x-outro;", `cp "$HOME/.modelo.png" "$5"`)
	// Um cujo programa não existe (TryExec) fica de fora.
	m.Grava(".miniaturizadores/aaa.thumbnailer", "[Thumbnailer Entry]\nTryExec=/nao/ha/programa\nExec=/nao/ha/programa %i %o\nMimeType=video/mp4;\n")
	video := m.Grava("Vídeos/meu vídeo $(x);`y`.mp4", "dados de vídeo")

	pede(m, "normal", "g", video)
	e := m.Evento("miniatura", de(video))
	arq, _ := e["arquivo"].(string)
	if arq != m.Casa+"/.cache/thumbnails/normal/"+md5De(video)+".png" {
		t.Fatalf("miniatura de fora: %v", e)
	}
	dados, _ := os.ReadFile(arq)
	info, _ := os.Stat(video)
	if tx := textosDe(dados); tx["Thumb::URI"] != base.URI(video) || tx["Thumb::MTime"] != fmt.Sprint(info.ModTime().Unix()) {
		t.Errorf("o motor põe os blocos tEXt no PNG de fora: %v", tx)
	}
	if mini, err := png.Decode(bytes.NewReader(dados)); err != nil || mini.Bounds().Dx() != 128 {
		t.Errorf("o PNG continua válido: %v", err)
	}
	// O argv: cada código vira um argumento inteiro, sem shell.
	chamadas, _ := os.ReadFile(m.Casa + "/.chamadas")
	linha := strings.TrimSpace(string(chamadas))
	if !strings.HasPrefix(linha, "-s 128 "+base.URI(video)+" "+video+" "+m.Casa+"/.cache/thumbnails/normal/.mkfiles-") {
		t.Errorf("a chamada: %q", linha)
	}
	// O temporário não fica.
	if e, _ := os.ReadDir(m.Casa + "/.cache/thumbnails/normal"); len(e) != 1 {
		t.Errorf("sobrou temporário: %v", e)
	}
	// De novo: reusa, sem chamar o programa.
	pede(m, "normal", "g", video)
	m.Evento("miniatura", de(video))
	if depois, _ := os.ReadFile(m.Casa + "/.chamadas"); string(depois) != string(chamadas) {
		t.Error("chamou o miniaturizador para uma miniatura que valia")
	}
	// Large passa 256.
	pede(m, "large", "g", video)
	m.Evento("miniatura", de(video))
	if depois, _ := os.ReadFile(m.Casa + "/.chamadas"); !strings.Contains(string(depois), "-s 256 ") {
		t.Errorf("large: %s", depois)
	}
}

func TestMiniaturizadorQueFalhaOuDemora(t *testing.T) {
	m := motor(t)
	Prazo = 400 * time.Millisecond
	t.Cleanup(func() { Prazo = 10 * time.Second })
	miniaturizador(t, m, "lento", "video/mp4;", `sleep 30 & wait`)
	miniaturizador(t, m, "falha", "application/pdf;", `exit 3`)
	miniaturizador(t, m, "lixo", "text/plain;", `echo "isto não é png" >"$5"`)
	video := m.Grava("a.mp4", "x")
	pdf := m.Grava("a.pdf", "x")
	antes := time.Now()
	pede(m, "normal", "g", video, pdf)
	if e := m.Evento("miniatura", de(video)); e["arquivo"] != nil {
		t.Errorf("o que estoura o tempo: %v", e)
	}
	if d := time.Since(antes); d > 5*time.Second {
		t.Errorf("o prazo não valeu: %v", d)
	}
	if e := m.Evento("miniatura", de(pdf)); e["arquivo"] != nil {
		t.Errorf("o que sai com erro: %v", e)
	}
	// As duas falhas ficam anotadas, e o temporário não fica.
	for _, c := range []string{video, pdf} {
		if _, err := os.Stat(m.Casa + "/.cache/thumbnails/fail/mkfiles/" + md5De(c) + ".png"); err != nil {
			t.Errorf("falha de %s não anotada", c)
		}
	}
	if e, _ := os.ReadDir(m.Casa + "/.cache/thumbnails/normal"); len(e) != 0 {
		t.Errorf("sobrou arquivo: %v", e)
	}
	// O que "dá certo" mas não escreve um PNG também é falha.
	nota := m.Grava("nota.txt", "x")
	pede(m, "normal", "g", nota)
	if e := m.Evento("miniatura", de(nota)); e["arquivo"] != nil {
		t.Errorf("saída que não é PNG: %v", e)
	}
}

func TestDoisPorVezECancelarPorGrupo(t *testing.T) {
	m := motor(t)
	gravaPNG(t, m.Casa+"/.modelo.png", 16, 16)
	// Cada chamada anota quando começa e quando acaba.
	miniaturizador(t, m, "video", "video/mp4;", `echo comeca >>"$HOME/.linha"; sleep 0.3; cp "$HOME/.modelo.png" "$5"; echo acaba >>"$HOME/.linha"`)
	var daPasta, daOutra []string
	for i := 0; i < 4; i++ {
		daPasta = append(daPasta, m.Grava(fmt.Sprintf("a/%d.mp4", i), "x"))
		daOutra = append(daOutra, m.Grava(fmt.Sprintf("b/%d.mp4", i), "x"))
	}
	pede(m, "normal", "pasta-a", daPasta...)
	time.Sleep(100 * time.Millisecond) // os dois primeiros já estão rodando
	pede(m, "normal", "pasta-b", daOutra...)
	// Saiu da pasta b: o que não começou sai da fila.
	if r := m.Pede(`{"op":"miniatura-cancelar","grupo":"pasta-b"}`); r["cancelados"] != float64(4) {
		t.Errorf("cancelar por grupo: %v", r)
	}
	for _, c := range daPasta {
		if e := m.Evento("miniatura", de(c)); e["arquivo"] == nil {
			t.Errorf("a pasta a continua: %v", e)
		}
	}
	m.SemEvento("miniatura", 200*time.Millisecond)
	// Nunca mais de dois ao mesmo tempo.
	linha, _ := os.ReadFile(m.Casa + "/.linha")
	rodando, pico := 0, 0
	for _, l := range strings.Fields(string(linha)) {
		if l == "comeca" {
			rodando++
		} else {
			rodando--
		}
		pico = max(pico, rodando)
	}
	if pico != 2 {
		t.Errorf("processos ao mesmo tempo: %d (%s)", pico, strings.Fields(string(linha)))
	}
	if r := m.Pede(`{"op":"miniatura-cancelar","grupo":"pasta-b"}`); r["cancelados"] != float64(0) {
		t.Errorf("cancelar de novo: %v", r)
	}
}

func TestFilaPorPrioridade(t *testing.T) {
	f := &fabrica{ativos: 10}
	poe := func(caminho string, prioridade int) {
		f.ordem++
		f.fila = append(f.fila, &pedido{caminho: caminho, prioridade: prioridade, ordem: f.ordem})
	}
	poe("velho", 0)
	poe("novo", 0)
	poe("urgente", 5)
	poe("fundo", -1)
	var ordem []string
	for p := f.proximo(); p != nil; p = f.proximo() {
		ordem = append(ordem, p.caminho)
	}
	if strings.Join(ordem, " ") != "urgente novo velho fundo" {
		t.Errorf("prioridade maior primeiro; no empate, o mais novo: %v", ordem)
	}
}

func TestPedidoRepetidoSobeNaFila(t *testing.T) {
	m := motor(t)
	gravaPNG(t, m.Casa+"/.modelo.png", 16, 16)
	miniaturizador(t, m, "video", "video/mp4;", `sleep 0.2; cp "$HOME/.modelo.png" "$5"`)
	var todos []string
	for i := 0; i < 5; i++ {
		todos = append(todos, m.Grava(fmt.Sprintf("v/%d.mp4", i), "x"))
	}
	pede(m, "normal", "g", todos...)
	pede(m, "normal", "g", todos[3], todos[4]) // de novo: não duplica
	for _, c := range todos {
		m.Evento("miniatura", de(c))
	}
	m.SemEvento("miniatura", 300*time.Millisecond)
}

func TestMiniaturaDeMiniatura(t *testing.T) {
	m := motor(t)
	dentro := m.Casa + "/.cache/thumbnails/normal/abc.png"
	gravaPNG(t, dentro, 300, 300)
	pede(m, "normal", "g", dentro)
	if e := m.Evento("miniatura", de(dentro)); e["arquivo"] != dentro {
		t.Errorf("uma miniatura serve de miniatura de si mesma: %v", e)
	}
	if e, _ := os.ReadDir(m.Casa + "/.cache/thumbnails/normal"); len(e) != 1 {
		t.Errorf("fez miniatura de miniatura: %v", e)
	}
}

func TestBlocosDeTexto(t *testing.T) {
	var b bytes.Buffer
	png.Encode(&b, imagem(4, 4))
	com, err := comTextos(b.Bytes(), [][2]string{{"Thumb::URI", "file:///a"}, {"Thumb::MTime", "1"}})
	if err != nil {
		t.Fatal(err)
	}
	// Trocar: as chaves antigas saem, as outras ficam.
	com, _ = comTextos(com, [][2]string{{"Outra", "x"}})
	com, err = comTextos(com, [][2]string{{"Thumb::URI", "file:///b"}, {"Thumb::MTime", "2"}})
	if err != nil {
		t.Fatal(err)
	}
	tx := textosDe(com)
	if len(tx) != 3 || tx["Thumb::URI"] != "file:///b" || tx["Thumb::MTime"] != "2" || tx["Outra"] != "x" {
		t.Errorf("textos: %v", tx)
	}
	if bytes.Count(com, []byte("Thumb::URI")) != 1 {
		t.Error("a chave antiga ficou duplicada")
	}
	if _, err := png.Decode(bytes.NewReader(com)); err != nil {
		t.Errorf("o crc dos blocos está certo: o PNG abre: %v", err)
	}
	for _, ruim := range [][]byte{nil, []byte("texto"), com[:40], com[:len(com)-6]} {
		if _, err := comTextos(ruim, nil); err == nil {
			t.Errorf("PNG torto (%d bytes) devia falhar", len(ruim))
		}
	}
	if len(textosDe([]byte("lixo"))) != 0 {
		t.Error("textosDe de lixo")
	}
}

func TestMedidasEPalavras(t *testing.T) {
	casos := [][5]int{{400, 200, 128, 128, 64}, {200, 400, 128, 64, 128}, {100, 50, 128, 100, 50}, {10000, 3, 128, 128, 1}, {128, 128, 128, 128, 128}}
	for _, c := range casos {
		if l, a := medidas(c[0], c[1], c[2]); l != c[3] || a != c[4] {
			t.Errorf("medidas(%d, %d, %d) = %d, %d", c[0], c[1], c[2], l, a)
		}
	}
	if p := palavras(`"/opt/meu app/x" -s %s  %i`); len(p) != 4 || p[0] != "/opt/meu app/x" {
		t.Errorf("palavras: %q", p)
	}
	if palavras(`"aberta`) != nil || palavras("") != nil {
		t.Error("Exec torto")
	}
	// Uma imagem enorme é recusada antes de ser decodificada.
	if _, err := abre("/nao/ha"); err == nil {
		t.Error("abre de arquivo que não existe")
	}
}
