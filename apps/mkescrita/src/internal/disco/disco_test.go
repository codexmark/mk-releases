package disco

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"mkescrita/internal/base"
)

func escreve(t *testing.T, caminho, texto string, modo os.FileMode) {
	t.Helper()
	if err := os.WriteFile(caminho, []byte(texto), modo); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(caminho, modo); err != nil {
		t.Fatal(err)
	}
}

func conteudo(t *testing.T, caminho string) string {
	t.Helper()
	b, err := os.ReadFile(caminho)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// semSobras confere que nenhum temporário ficou na pasta.
func semSobras(t *testing.T, pasta string) {
	t.Helper()
	entradas, _ := os.ReadDir(pasta)
	for _, e := range entradas {
		if strings.Contains(e.Name(), ".mkescrita-") {
			t.Errorf("sobrou um temporário: %s", e.Name())
		}
	}
}

func TestAbrirDevolveOTextoEAImpressao(t *testing.T) {
	pasta := t.TempDir()
	arq := filepath.Join(pasta, "notas.md")
	escreve(t, arq, "# Olá\n\nmundo, ação\n", 0o644)
	a, err := Abrir(arq)
	if err != nil {
		t.Fatal(err)
	}
	if a.Texto != "# Olá\n\nmundo, ação\n" || a.Nome != "notas.md" || a.Arquivo != arq || a.Novo {
		t.Errorf("aberto: %+v", a)
	}
	if a.FimDeLinha != "lf" || !a.QuebraFinal || a.SoLeitura {
		t.Errorf("fim de linha/quebra/leitura: %+v", a)
	}
	if a.Impressao.Tamanho != int64(len("# Olá\n\nmundo, ação\n")) || a.Impressao.Sha256 != Resumo([]byte("# Olá\n\nmundo, ação\n")) || a.Impressao.Mtime == 0 {
		t.Errorf("impressão: %+v", a.Impressao)
	}
	if len(a.Impressao.Sha256) != 64 {
		t.Errorf("sha256 devia ter 64 letras: %s", a.Impressao.Sha256)
	}
}

func TestAbrirFimDeLinhaEQuebraFinal(t *testing.T) {
	pasta := t.TempDir()
	casos := []struct {
		disco, texto, fim string
		quebra            bool
	}{
		{"a\r\nb\r\n", "a\nb\n", "crlf", true},
		{"a\r\nb", "a\nb", "crlf", false},
		{"a\nb", "a\nb", "lf", false},
		{"a\r\nb\nc\n", "a\r\nb\nc\n", "misto", true},
		{"a\rb\r\n", "a\rb\r\n", "misto", true},
		{"", "", "lf", true},
	}
	for _, c := range casos {
		arq := filepath.Join(pasta, "t.txt")
		escreve(t, arq, c.disco, 0o644)
		a, err := Abrir(arq)
		if err != nil {
			t.Fatal(err)
		}
		if a.Texto != c.texto || a.FimDeLinha != c.fim || a.QuebraFinal != c.quebra {
			t.Errorf("%q: texto %q fim %s quebra %v", c.disco, a.Texto, a.FimDeLinha, a.QuebraFinal)
		}
		// A impressão é do que está no disco, não do texto convertido.
		if a.Impressao.Sha256 != Resumo([]byte(c.disco)) {
			t.Errorf("%q: a impressão não é a do disco", c.disco)
		}
	}
}

func TestAbrirRecusaOQueNaoETexto(t *testing.T) {
	pasta := t.TempDir()
	t.Setenv("MKESCRITA_LIMITE_MB", "1")
	binario := filepath.Join(pasta, "foto.png")
	escreve(t, binario, "\x89PNG\x00\x01\x02", 0o644)
	latin := filepath.Join(pasta, "velho.txt")
	escreve(t, latin, "a\xe7\xe3o", 0o644)
	grande := filepath.Join(pasta, "grande.md")
	escreve(t, grande, strings.Repeat("a", 1<<20+1), 0o644)
	for arq, trecho := range map[string]string{binario: "binário", latin: "UTF-8", grande: "grande demais", pasta: "pasta"} {
		_, err := Abrir(arq)
		if !base.FoiRecusa(err) || !strings.Contains(err.Error(), trecho) {
			t.Errorf("%s: queria recusa com %q, veio %v", filepath.Base(arq), trecho, err)
		}
	}
	// No limite exato, abre.
	escreve(t, grande, strings.Repeat("a", 1<<20), 0o644)
	if _, err := Abrir(grande); err != nil {
		t.Errorf("no limite devia abrir: %v", err)
	}
}

func TestAbrirArquivoQueAindaNaoExiste(t *testing.T) {
	pasta := t.TempDir()
	a, err := Abrir(filepath.Join(pasta, "novo.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !a.Novo || a.Texto != "" || a.Impressao != nil || a.Nome != "novo.md" || !a.QuebraFinal {
		t.Errorf("novo: %+v", a)
	}
	if _, err := Abrir(filepath.Join(pasta, "nao", "existe", "x.md")); err == nil || !strings.Contains(err.Error(), "pasta") {
		t.Errorf("pasta que não existe devia ser erro: %v", err)
	}
}

func TestAbrirSoLeitura(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root escreve em tudo")
	}
	arq := filepath.Join(t.TempDir(), "ro.md")
	escreve(t, arq, "x\n", 0o444)
	a, err := Abrir(arq)
	if err != nil || !a.SoLeitura {
		t.Errorf("só leitura: %+v %v", a, err)
	}
	if _, err := Salvar(Pedido{Caminho: arq, Texto: []byte("y\n"), Esperado: a.Impressao.Sha256}); err == nil || !strings.Contains(err.Error(), "só para leitura") {
		t.Errorf("salvar em arquivo só de leitura devia falhar: %v", err)
	}
	if conteudo(t, arq) != "x\n" {
		t.Error("o arquivo só de leitura foi trocado")
	}
}

func TestSalvarGravaEPreservaPermissoes(t *testing.T) {
	pasta := t.TempDir()
	arq := filepath.Join(pasta, "notas.md")
	escreve(t, arq, "antes\n", 0o640)
	a, _ := Abrir(arq)
	s, err := Salvar(Pedido{Caminho: arq, Texto: []byte("depois, ação\n"), Esperado: a.Impressao.Sha256})
	if err != nil {
		t.Fatal(err)
	}
	if conteudo(t, arq) != "depois, ação\n" {
		t.Errorf("conteúdo: %q", conteudo(t, arq))
	}
	info, _ := os.Stat(arq)
	if info.Mode().Perm() != 0o640 {
		t.Errorf("permissões: %o", info.Mode().Perm())
	}
	if s.Impressao.Sha256 != Resumo([]byte("depois, ação\n")) || s.Impressao.Tamanho != info.Size() || s.Impressao.Mtime != info.ModTime().UnixNano() {
		t.Errorf("impressão depois de salvar: %+v", s.Impressao)
	}
	semSobras(t, pasta)
	// Salvar de novo com a impressão nova funciona; com a velha, é conflito... não:
	// o conteúdo no disco é o que eu mesmo gravei, então a impressão nova vale.
	if _, err := Salvar(Pedido{Caminho: arq, Texto: []byte("mais\n"), Esperado: s.Impressao.Sha256}); err != nil {
		t.Errorf("segundo salvar: %v", err)
	}
}

func TestSalvarArquivoNovoRespeitaAUmask(t *testing.T) {
	pasta := t.TempDir()
	arq := filepath.Join(pasta, "novo.md")
	antiga := syscall.Umask(0o027)
	defer syscall.Umask(antiga)
	if _, err := Salvar(Pedido{Caminho: arq, Texto: []byte("oi"), Esperado: Ausente, QuebraFinal: true}); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(arq)
	if info.Mode().Perm() != 0o640 {
		t.Errorf("permissões do arquivo novo: %o", info.Mode().Perm())
	}
	if conteudo(t, arq) != "oi\n" {
		t.Errorf("a quebra final não entrou: %q", conteudo(t, arq))
	}
	semSobras(t, pasta)
}

func TestSalvarConflitoNaoGravaNada(t *testing.T) {
	pasta := t.TempDir()
	arq := filepath.Join(pasta, "notas.md")
	escreve(t, arq, "aberto\n", 0o644)
	a, _ := Abrir(arq)
	escreve(t, arq, "mudado por fora\n", 0o644)

	_, err := Salvar(Pedido{Caminho: arq, Texto: []byte("o meu\n"), Esperado: a.Impressao.Sha256})
	var c *Conflito
	if !errors.As(err, &c) || c.Motivo != "mudou" || c.Impressao.Sha256 != Resumo([]byte("mudado por fora\n")) {
		t.Fatalf("queria conflito 'mudou' com a impressão do disco: %v %+v", err, c)
	}
	if conteudo(t, arq) != "mudado por fora\n" {
		t.Error("gravou apesar do conflito")
	}
	semSobras(t, pasta)

	// Sumiu.
	os.Remove(arq)
	_, err = Salvar(Pedido{Caminho: arq, Texto: []byte("o meu\n"), Esperado: a.Impressao.Sha256})
	if !errors.As(err, &c) || c.Motivo != "sumiu" || c.Impressao != nil {
		t.Errorf("queria conflito 'sumiu': %v", err)
	}
	if _, e := os.Stat(arq); e == nil {
		t.Error("recriou o arquivo apesar do conflito")
	}

	// Apareceu: eu achava que não existia.
	escreve(t, arq, "de outra pessoa\n", 0o644)
	_, err = Salvar(Pedido{Caminho: arq, Texto: []byte("o meu\n"), Esperado: Ausente})
	if !errors.As(err, &c) || c.Motivo != "apareceu" {
		t.Errorf("queria conflito 'apareceu': %v", err)
	}
	if conteudo(t, arq) != "de outra pessoa\n" {
		t.Error("gravou por cima do arquivo que apareceu")
	}

	// Forçar grava assim mesmo.
	s, err := Salvar(Pedido{Caminho: arq, Texto: []byte("o meu\n"), Esperado: a.Impressao.Sha256, Forcar: true})
	if err != nil || conteudo(t, arq) != "o meu\n" || s.Impressao.Sha256 != Resumo([]byte("o meu\n")) {
		t.Errorf("forçar: %v %q", err, conteudo(t, arq))
	}
}

func TestSalvarMudancaSoDeDataNaoEConflito(t *testing.T) {
	arq := filepath.Join(t.TempDir(), "notas.md")
	escreve(t, arq, "igual\n", 0o644)
	a, _ := Abrir(arq)
	depois := time.Now().Add(time.Hour)
	os.Chtimes(arq, depois, depois) // um `touch`: o conteúdo é o mesmo
	if _, err := Salvar(Pedido{Caminho: arq, Texto: []byte("novo\n"), Esperado: a.Impressao.Sha256}); err != nil {
		t.Errorf("mesmo conteúdo com outra data não é conflito: %v", err)
	}
}

func TestSalvarPedeImpressaoOuForcar(t *testing.T) {
	arq := filepath.Join(t.TempDir(), "notas.md")
	escreve(t, arq, "x\n", 0o644)
	if _, err := Salvar(Pedido{Caminho: arq, Texto: []byte("y\n")}); err == nil {
		t.Error("salvar sem --esperado e sem --forcar devia falhar")
	}
	if conteudo(t, arq) != "x\n" {
		t.Error("gravou sem saber o que esperar")
	}
	if _, err := Salvar(Pedido{Texto: []byte("y\n"), Forcar: true}); err == nil {
		t.Error("salvar sem caminho devia falhar")
	}
}

func TestSalvarSegueLinkSimbolico(t *testing.T) {
	pasta := t.TempDir()
	os.Mkdir(filepath.Join(pasta, "real"), 0o755)
	alvo := filepath.Join(pasta, "real", "alvo.md")
	link := filepath.Join(pasta, "atalho.md")
	escreve(t, alvo, "antes\n", 0o600)
	if err := os.Symlink(filepath.Join("real", "alvo.md"), link); err != nil { // relativo, de propósito
		t.Fatal(err)
	}
	a, err := Abrir(link)
	if err != nil || a.Arquivo != link {
		t.Fatalf("abrir pelo link: %+v %v", a, err)
	}
	s, err := Salvar(Pedido{Caminho: link, Texto: []byte("depois\n"), Esperado: a.Impressao.Sha256})
	if err != nil {
		t.Fatal(err)
	}
	if info, _ := os.Lstat(link); info.Mode()&os.ModeSymlink == 0 {
		t.Error("o link simbólico virou arquivo")
	}
	if conteudo(t, alvo) != "depois\n" {
		t.Errorf("o alvo não recebeu o texto: %q", conteudo(t, alvo))
	}
	if info, _ := os.Stat(alvo); info.Mode().Perm() != 0o600 {
		t.Errorf("permissões do alvo: %o", info.Mode().Perm())
	}
	if s.Arquivo != link {
		t.Errorf("a resposta devia falar do caminho que a pessoa usou: %s", s.Arquivo)
	}
	semSobras(t, filepath.Join(pasta, "real"))
	semSobras(t, pasta)

	// Link para um arquivo que ainda não existe: cria o alvo, o link fica.
	solto := filepath.Join(pasta, "solto.md")
	os.Symlink(filepath.Join(pasta, "real", "ainda-nao.md"), solto)
	if _, err := Salvar(Pedido{Caminho: solto, Texto: []byte("oi\n"), Esperado: Ausente}); err != nil {
		t.Fatal(err)
	}
	if info, _ := os.Lstat(solto); info.Mode()&os.ModeSymlink == 0 {
		t.Error("o link solto virou arquivo")
	}
	if conteudo(t, filepath.Join(pasta, "real", "ainda-nao.md")) != "oi\n" {
		t.Error("o alvo do link solto não foi criado")
	}

	// Link em círculo: erro, não laço.
	os.Symlink(filepath.Join(pasta, "b"), filepath.Join(pasta, "a"))
	os.Symlink(filepath.Join(pasta, "a"), filepath.Join(pasta, "b"))
	if _, err := Salvar(Pedido{Caminho: filepath.Join(pasta, "a"), Texto: []byte("x"), Forcar: true}); err == nil {
		t.Error("link em círculo devia ser erro")
	}
}

func TestSalvarFimDeLinha(t *testing.T) {
	pasta := t.TempDir()
	arq := filepath.Join(pasta, "win.txt")
	escreve(t, arq, "a\r\nb\r\n", 0o644)
	a, _ := Abrir(arq)
	// Abrir e salvar sem mexer devolve o arquivo byte a byte.
	s, err := Salvar(Pedido{Caminho: arq, Texto: []byte(a.Texto), Esperado: a.Impressao.Sha256, FimDeLinha: a.FimDeLinha, QuebraFinal: a.QuebraFinal})
	if err != nil || conteudo(t, arq) != "a\r\nb\r\n" || s.Impressao.Sha256 != a.Impressao.Sha256 {
		t.Errorf("ida e volta CRLF: %q %v", conteudo(t, arq), err)
	}
	for _, c := range []struct {
		texto, fim string
		quebra     bool
		quer       string
	}{
		{"a\nb", "crlf", true, "a\r\nb\r\n"},
		{"a\nb", "crlf", false, "a\r\nb"},
		{"a\r\nb\n", "crlf", false, "a\r\nb\r\n"}, // não dobra o \r
		{"a\nb", "lf", false, "a\nb"},
		{"a\nb", "lf", true, "a\nb\n"},
		{"a\nb\n\n", "lf", true, "a\nb\n\n"}, // não mexe nas quebras que já existem
		{"", "lf", true, ""},
		{"a\r\nb\n", "misto", false, "a\r\nb\n"},
	} {
		if g := string(Preparar([]byte(c.texto), c.fim, c.quebra)); g != c.quer {
			t.Errorf("Preparar(%q, %s, %v) = %q, queria %q", c.texto, c.fim, c.quebra, g, c.quer)
		}
	}
	// Preparar não estraga o texto de quem chamou.
	original := []byte("abc")
	Preparar(original[:2], "lf", true)
	if string(original) != "abc" {
		t.Error("Preparar escreveu no texto de quem chamou")
	}
}

func TestSalvarFalhaLimpaQuandoAPastaNaoDeixa(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root escreve em tudo")
	}
	pasta := t.TempDir()
	arq := filepath.Join(pasta, "notas.md")
	escreve(t, arq, "antes\n", 0o644)
	os.Chmod(pasta, 0o555)
	defer os.Chmod(pasta, 0o755)
	_, err := Salvar(Pedido{Caminho: arq, Texto: []byte("depois\n"), Forcar: true})
	if err == nil || !strings.Contains(err.Error(), "sem permissão") {
		t.Errorf("queria erro de permissão: %v", err)
	}
	if conteudo(t, arq) != "antes\n" {
		t.Error("o arquivo antigo foi estragado")
	}
	if _, err := Salvar(Pedido{Caminho: filepath.Join(pasta, "nao", "x.md"), Texto: []byte("x"), Esperado: Ausente}); err == nil {
		t.Error("pasta que não existe devia ser erro")
	}
	if _, err := Salvar(Pedido{Caminho: pasta, Texto: []byte("x"), Forcar: true}); !base.FoiRecusa(err) {
		t.Errorf("salvar numa pasta devia ser recusado: %v", err)
	}
}

func TestExportarNuncaPassaPorCimaSemForcar(t *testing.T) {
	pasta := t.TempDir()
	saida := filepath.Join(pasta, "doc.html")
	if _, err := Exportar(saida, []byte("<p>um</p>"), false); err != nil {
		t.Fatal(err)
	}
	_, err := Exportar(saida, []byte("<p>dois</p>"), false)
	if !base.FoiRecusa(err) || !strings.Contains(err.Error(), "já existe") {
		t.Errorf("queria recusa: %v", err)
	}
	if conteudo(t, saida) != "<p>um</p>" {
		t.Error("passou por cima sem --forcar")
	}
	if _, err := Exportar(saida, []byte("<p>dois</p>"), true); err != nil || conteudo(t, saida) != "<p>dois</p>" {
		t.Errorf("forçar: %v", err)
	}
	if _, err := Exportar(pasta, []byte("x"), true); !base.FoiRecusa(err) {
		t.Errorf("exportar para uma pasta devia ser recusado: %v", err)
	}
	if _, err := Exportar("", []byte("x"), true); err == nil {
		t.Error("exportar sem saída devia falhar")
	}
	semSobras(t, pasta)
}

func TestVigiarAvisaQuandoMudaESome(t *testing.T) {
	pasta := t.TempDir()
	arq := filepath.Join(pasta, "notas.md")
	escreve(t, arq, "um\n", 0o644)
	var trava sync.Mutex
	var eventos []Evento
	parar := make(chan struct{})
	fim := make(chan struct{})
	go func() {
		Vigiar(arq, "", 5*time.Millisecond, func(e Evento) {
			trava.Lock()
			eventos = append(eventos, e)
			trava.Unlock()
		}, func() bool {
			select {
			case <-parar:
				return false
			default:
				return true
			}
		})
		close(fim)
	}()
	espera := func(n int) []Evento {
		t.Helper()
		for i := 0; i < 400; i++ {
			trava.Lock()
			tem := len(eventos)
			copia := append([]Evento{}, eventos...)
			trava.Unlock()
			if tem >= n {
				return copia
			}
			time.Sleep(5 * time.Millisecond)
		}
		t.Fatalf("esperava %d eventos, vieram %d", n, len(eventos))
		return nil
	}
	time.Sleep(30 * time.Millisecond)
	if trava.Lock(); len(eventos) != 0 {
		t.Errorf("sem mudança não devia haver evento: %+v", eventos)
	}
	trava.Unlock()

	escreve(t, arq, "dois!\n", 0o644)
	ev := espera(1)
	if ev[0].Evento != "mudou" || ev[0].Impressao.Sha256 != Resumo([]byte("dois!\n")) {
		t.Errorf("primeiro evento: %+v", ev[0])
	}
	os.Remove(arq)
	ev = espera(2)
	if ev[1].Evento != "sumiu" || ev[1].Impressao != nil {
		t.Errorf("segundo evento: %+v", ev[1])
	}
	escreve(t, arq, "voltou\n", 0o644)
	ev = espera(3)
	if ev[2].Evento != "mudou" {
		t.Errorf("terceiro evento: %+v", ev[2])
	}
	close(parar)
	select {
	case <-fim:
	case <-time.After(2 * time.Second):
		t.Fatal("o vigia não parou")
	}
}

func TestVigiarComImpressaoConhecida(t *testing.T) {
	pasta := t.TempDir()
	arq := filepath.Join(pasta, "notas.md")
	escreve(t, arq, "no disco\n", 0o644)
	roda := func(conhecido string, voltas int) []Evento {
		var eventos []Evento
		n := 0
		Vigiar(arq, conhecido, time.Millisecond, func(e Evento) { eventos = append(eventos, e) }, func() bool { n++; return n <= voltas })
		return eventos
	}
	// A janela conhece outro conteúdo: avisa logo na primeira olhada, uma vez só.
	if ev := roda(Resumo([]byte("o que a janela tem\n")), 4); len(ev) != 1 || ev[0].Evento != "mudou" {
		t.Errorf("impressão diferente: %+v", ev)
	}
	// A janela conhece este mesmo conteúdo: silêncio.
	if ev := roda(Resumo([]byte("no disco\n")), 4); len(ev) != 0 {
		t.Errorf("mesma impressão: %+v", ev)
	}
	// A janela achava que não existia.
	if ev := roda(Ausente, 3); len(ev) != 1 || ev[0].Evento != "mudou" {
		t.Errorf("ausente e existe: %+v", ev)
	}
	os.Remove(arq)
	if ev := roda(Ausente, 3); len(ev) != 0 {
		t.Errorf("ausente e continua ausente: %+v", ev)
	}
	if ev := roda("", 3); len(ev) != 0 {
		t.Errorf("sem impressão e sem arquivo: %+v", ev)
	}
}

func preparaEstado(t *testing.T) (proc string) {
	t.Helper()
	t.Setenv("XDG_STATE_HOME", filepath.Join(t.TempDir(), "estado"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "config"))
	proc = filepath.Join(t.TempDir(), "proc")
	os.MkdirAll(proc, 0o755)
	t.Setenv("MKESCRITA_PROC", proc)
	return proc
}

func TestRascunhos(t *testing.T) {
	proc := preparaEstado(t)
	relogio := int64(1000)
	Agora = func() int64 { relogio++; return relogio }
	defer func() { Agora = func() int64 { return time.Now().Unix() } }()

	if itens, err := ListarRascunhos(); err != nil || len(itens) != 0 || itens == nil {
		t.Fatalf("sem pasta: lista vazia, não erro nem nulo: %v %v", itens, err)
	}
	// O dono 4242 é uma janela viva chamada "qs".
	os.MkdirAll(filepath.Join(proc, "4242"), 0o755)
	os.WriteFile(filepath.Join(proc, "4242", "comm"), []byte("qs\n"), 0o644)

	if _, err := GuardarRascunho("r-um", "", "", []byte("\n\n  primeira   linha  \nsegunda\n"), 4242); err != nil {
		t.Fatal(err)
	}
	g, err := GuardarRascunho("r-dois", "/textos/carta.md", "abc123", []byte(strings.Repeat("é", 100)), 77)
	if err != nil {
		t.Fatal(err)
	}
	if g.Nome != "carta.md" || g.Bytes != 200 || g.Trecho != strings.Repeat("é", 80)+"…" || g.Vivo {
		t.Errorf("resumo do guardado: %+v", g)
	}
	itens, _ := ListarRascunhos()
	if len(itens) != 2 || itens[0].ID != "r-dois" || itens[1].ID != "r-um" {
		t.Fatalf("o mais recente primeiro: %+v", itens)
	}
	if !itens[1].Vivo || itens[0].Vivo {
		t.Errorf("vivo: só o da janela que existe: %+v", itens)
	}
	if itens[1].Trecho != "primeira linha" || itens[1].Origem != "" || itens[1].Nome != "" {
		t.Errorf("trecho e origem: %+v", itens[1])
	}
	// A janela morreu e o número foi reaproveitado por outro programa: não está vivo.
	os.WriteFile(filepath.Join(proc, "4242", "comm"), []byte("firefox\n"), 0o644)
	if itens, _ := ListarRascunhos(); itens[1].Vivo {
		t.Error("pid reaproveitado por outro programa não é a janela")
	}

	r, err := LerRascunho("r-dois")
	if err != nil || r.Texto != strings.Repeat("é", 100) || r.Origem != "/textos/carta.md" || r.Base != "abc123" || r.GuardadoEm != 1002 {
		t.Errorf("ler: %+v %v", r, err)
	}
	// Regravar troca o texto, não cria outro.
	GuardarRascunho("r-dois", "/textos/carta.md", "abc123", []byte("novo"), 77)
	if r, _ := LerRascunho("r-dois"); r.Texto != "novo" {
		t.Error("regravar não trocou o texto")
	}
	if itens, _ := ListarRascunhos(); len(itens) != 2 {
		t.Errorf("regravar criou outro: %d", len(itens))
	}
	// As permissões: só da pessoa.
	info, _ := os.Stat(filepath.Join(base.Rascunhos(), "r-dois.json"))
	if info.Mode().Perm() != 0o600 {
		t.Errorf("rascunho devia ser 0600: %o", info.Mode().Perm())
	}
	if pasta, _ := os.Stat(base.Rascunhos()); pasta.Mode().Perm() != 0o700 {
		t.Errorf("a pasta dos rascunhos devia ser 0700: %o", pasta.Mode().Perm())
	}
	semSobras(t, base.Rascunhos())

	// Um arquivo estragado e um estranho na pasta não derrubam a lista.
	os.WriteFile(filepath.Join(base.Rascunhos(), "ruim.json"), []byte("{não é json"), 0o600)
	os.WriteFile(filepath.Join(base.Rascunhos(), "LEIAME.txt"), []byte("oi"), 0o600)
	if itens, err := ListarRascunhos(); err != nil || len(itens) != 2 {
		t.Errorf("com arquivo estragado: %v %v", itens, err)
	}
	if _, err := LerRascunho("ruim"); err == nil || !strings.Contains(err.Error(), "estragado") {
		t.Errorf("ler o estragado: %v", err)
	}

	if err := ApagarRascunho("r-um"); err != nil {
		t.Fatal(err)
	}
	if err := ApagarRascunho("r-um"); err != nil {
		t.Errorf("apagar duas vezes não é erro: %v", err)
	}
	if _, err := LerRascunho("r-um"); err == nil {
		t.Error("o apagado ainda está lá")
	}
}

func TestRascunhoRecusaIdentificadorQueSaiDaPasta(t *testing.T) {
	preparaEstado(t)
	for _, id := range []string{"", "../fora", "a/b", "A", "-x", ".", "a.json", strings.Repeat("a", 65), "a b"} {
		if _, err := GuardarRascunho(id, "", "", []byte("x"), 0); err == nil {
			t.Errorf("guardar aceitou o id %q", id)
		}
		if _, err := LerRascunho(id); err == nil {
			t.Errorf("ler aceitou o id %q", id)
		}
		if err := ApagarRascunho(id); err == nil {
			t.Errorf("apagar aceitou o id %q", id)
		}
	}
	if _, err := GuardarRascunho("ok", "relativo.md", "", []byte("x"), 0); err == nil {
		t.Error("origem relativa devia ser erro")
	}
	if entradas, _ := os.ReadDir(filepath.Dir(base.Rascunhos())); len(entradas) > 1 {
		t.Errorf("algo foi escrito fora da pasta dos rascunhos: %v", entradas)
	}
}

func TestPreferencias(t *testing.T) {
	preparaEstado(t)
	if p := LerPrefs(); p != (Prefs{Fonte: "", Zoom: 100, Previa: false}) {
		t.Errorf("padrões: %+v", p)
	}
	p, err := GravarPrefs([]byte(`{"fonte":"  Fira Code\n ","zoom":125}`))
	if err != nil || p != (Prefs{Fonte: "Fira Code", Zoom: 125, Previa: false}) {
		t.Errorf("gravar: %+v %v", p, err)
	}
	// Só o que mudou: o resto fica.
	p, _ = GravarPrefs([]byte(`{"previa":true}`))
	if p != (Prefs{Fonte: "Fira Code", Zoom: 125, Previa: true}) {
		t.Errorf("gravar parcial: %+v", p)
	}
	if LerPrefs() != p {
		t.Errorf("ler depois de gravar: %+v", LerPrefs())
	}
	for zoom, quer := range map[string]int{"10": 50, "5000": 300, "-1": 50, "149.6": 150} {
		if p, _ := GravarPrefs([]byte(`{"zoom":` + zoom + `}`)); p.Zoom != quer {
			t.Errorf("zoom %s virou %d, queria %d", zoom, p.Zoom, quer)
		}
	}
	if _, err := GravarPrefs([]byte(`[1,2]`)); err == nil {
		t.Error("lista no lugar de objeto devia ser erro")
	}
	if _, err := GravarPrefs([]byte(`{"zoom":"grande"}`)); err == nil {
		t.Error("zoom que não é número devia ser erro")
	}
	// O arquivo é JSON legível, com quebra no fim.
	if c := conteudo(t, base.Preferencias()); !strings.HasSuffix(c, "}\n") || !strings.Contains(c, "\"fonte\": \"Fira Code\"") {
		t.Errorf("arquivo: %q", c)
	}
	// Estragado: volta aos padrões, sem erro.
	os.WriteFile(base.Preferencias(), []byte("{{{"), 0o644)
	if p := LerPrefs(); p.Zoom != 100 || p.Fonte != "" {
		t.Errorf("arquivo estragado: %+v", p)
	}
	// Zoom fora da faixa escrito à mão é acertado na leitura.
	os.WriteFile(base.Preferencias(), []byte(`{"zoom": 9999, "fonte": "X"}`), 0o644)
	if p := LerPrefs(); p.Zoom != 300 || p.Fonte != "X" {
		t.Errorf("zoom escrito à mão: %+v", p)
	}
}

func TestPreferenciasNumLinkSimbolicoContinuamLink(t *testing.T) {
	preparaEstado(t)
	real := filepath.Join(t.TempDir(), "dotfiles.json")
	os.MkdirAll(filepath.Dir(base.Preferencias()), 0o755)
	os.WriteFile(real, []byte(`{"zoom":110}`), 0o644)
	os.Symlink(real, base.Preferencias())
	if _, err := GravarPrefs([]byte(`{"zoom":120}`)); err != nil {
		t.Fatal(err)
	}
	if info, _ := os.Lstat(base.Preferencias()); info.Mode()&os.ModeSymlink == 0 {
		t.Error("o link das preferências virou arquivo")
	}
	if !strings.Contains(conteudo(t, real), "120") {
		t.Error("o alvo do link não recebeu as preferências")
	}
}
