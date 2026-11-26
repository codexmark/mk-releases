package lixeira

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"mkfiles/internal/prova"
)

func existe(c string) bool {
	_, err := os.Lstat(c)
	return err == nil
}

func TestNumerado(t *testing.T) {
	ocupados := map[string]bool{}
	oc := func(n string) bool { return ocupados[n] }
	casos := []struct {
		nome  string
		pasta bool
		quer  string
	}{
		{"foto.jpg", false, "foto (2).jpg"},
		{"foto (2).jpg", false, "foto (3).jpg"},
		{"pacote.tar.gz", false, "pacote (2).tar.gz"},
		{"PACOTE.TAR.XZ", false, "PACOTE (2).TAR.XZ"},
		{"sem-extensao", false, "sem-extensao (2)"},
		{".bashrc", false, ".bashrc (2)"},
		{"pasta.v2", true, "pasta.v2 (2)"},
		{"a (1).txt", false, "a (1) (2).txt"},
		{".tar.gz", false, ".tar (2).gz"},
	}
	for _, c := range casos {
		if n := Numerado(c.nome, c.pasta, oc); n != c.quer {
			t.Errorf("Numerado(%q) = %q; esperado %q", c.nome, n, c.quer)
		}
	}
	ocupados["foto (2).jpg"], ocupados["foto (3).jpg"] = true, true
	if n := Numerado("foto.jpg", false, oc); n != "foto (4).jpg" {
		t.Errorf("pula os ocupados: %q", n)
	}
	longo := strings.Repeat("ç", 126) + ".txt"
	if n := Numerado(longo, false, oc); len(n) > 255 || !strings.HasSuffix(n, " (2).txt") {
		t.Errorf("nome comprido: %d bytes, %q", len(n), n)
	}
}

func TestMandarListarLer(t *testing.T) {
	m := prova.Novo(t, Ligar)
	raiz := filepath.Join(m.Casa, ".local", "share", "Trash")
	if n := m.Pede(`{"op":"lixeira-estado"}`)["itens"]; n != float64(0) {
		t.Fatalf("lixeira vazia: %v", n)
	}
	a := m.Grava("Meus Documentos/ação #1.txt", "um")
	antes := time.Now().Unix()
	e, err := Mandar(a, nil)
	if err != nil {
		t.Fatal(err)
	}
	if existe(a) || e.Arquivo != raiz+"/files/ação #1.txt" || e.Origem != a || e.Lixeira != raiz {
		t.Fatalf("Mandar: %+v", e)
	}
	ficha, _ := os.ReadFile(raiz + "/info/ação #1.txt.trashinfo")
	linhas := strings.Split(string(ficha), "\n")
	if linhas[0] != "[Trash Info]" || linhas[1] != "Path="+m.Casa+"/Meus%20Documentos/a%C3%A7%C3%A3o%20%231.txt" ||
		!strings.HasPrefix(linhas[2], "DeletionDate=") {
		t.Fatalf("o .trashinfo: %q", ficha)
	}
	quando, err := time.ParseInLocation("2006-01-02T15:04:05", strings.TrimPrefix(linhas[2], "DeletionDate="), time.Local)
	if err != nil || quando.Unix() < antes-1 || quando.Unix() > time.Now().Unix()+1 {
		t.Errorf("DeletionDate na hora local: %q", linhas[2])
	}
	if info, _ := os.Stat(raiz + "/info/ação #1.txt.trashinfo"); info.Mode().Perm() != 0o600 {
		t.Errorf("modo da ficha: %v", info.Mode())
	}

	// Outro com o mesmo nome ganha " (2)", antes da extensão; e uma pasta, no fim.
	m.Grava("Meus Documentos/ação #1.txt", "dois")
	e2, err := Mandar(a, nil)
	if err != nil || e2.Arquivo != raiz+"/files/ação #1 (2).txt" || !existe(e2.Info) {
		t.Fatalf("nome ocupado: %+v %v", e2, err)
	}
	m.Pasta("p.d")
	m.Pasta("outra/p.d")
	Mandar(m.Casa+"/p.d", nil)
	if e3, _ := Mandar(m.Casa+"/outra/p.d", nil); e3.Arquivo != raiz+"/files/p.d (2)" {
		t.Errorf("pasta com nome ocupado: %+v", e3)
	}

	// Um link vai como link: o alvo fica onde está.
	alvo := m.Grava("alvo.txt", "x")
	os.Symlink(alvo, m.Casa+"/atalho")
	if _, err := Mandar(m.Casa+"/atalho", nil); err != nil || !existe(alvo) || existe(m.Casa+"/atalho") {
		t.Errorf("link: %v", err)
	}

	if n := Contar(); n != 5 {
		t.Errorf("Contar: %d", n)
	}
	linhasDoMotor := m.PedeFluxo(`{"op":"lixeira-listar"}`)
	fim := linhasDoMotor[len(linhasDoMotor)-1]
	itens := fim["itens"].([]any)
	if fim["fim"] != true || fim["total"] != float64(5) || len(itens) != 5 {
		t.Fatalf("lixeira-listar: %v", fim)
	}
	achou := false
	for _, i := range itens {
		it := i.(map[string]any)
		if it["caminho"] == e2.Arquivo {
			achou = true
			if it["nome_original"] != "ação #1.txt" || it["origem"] != a || it["lixeira"] != raiz || it["nome"] != "ação #1 (2).txt" ||
				it["apagado_s"].(float64) < float64(antes-1) || it["tipo"] != "arquivo" || it["tamanho"] != float64(4) {
				t.Errorf("item da lixeira: %v", it)
			}
		}
	}
	if !achou {
		t.Error("o item não veio na lista")
	}

	// Ler: só o que está em files/ de uma lixeira.
	if lido, err := Ler(e.Arquivo); err != nil || lido.Origem != a || lido.Info != e.Info {
		t.Errorf("Ler: %+v %v", lido, err)
	}
	for _, ruim := range []string{alvo, raiz + "/files/nao-ha", raiz + "/info/ação #1.txt.trashinfo", "relativo", raiz + "/files"} {
		if _, err := Ler(ruim); err == nil {
			t.Errorf("Ler(%q) devia falhar", ruim)
		}
	}

	// Ficha sem arquivo e arquivo sem ficha não aparecem, mas são do Conteudo.
	m.Grava(".local/share/Trash/files/orfao", "x")
	m.Grava(".local/share/Trash/info/fantasma.trashinfo", "[Trash Info]\nPath=/x\n")
	if Contar() != 5 {
		t.Error("órfãos não contam")
	}
	arqs, fichas := Conteudo(DaCasa())
	if len(arqs) != 6 || len(fichas) != 6 {
		t.Errorf("Conteudo: %d %d", len(arqs), len(fichas))
	}
	// Um arquivo sem ficha com o nome não é sobrescrito.
	m.Grava("orfao", "novo")
	if _, err := Mandar(m.Casa+"/orfao", nil); err == nil || !existe(m.Casa+"/orfao") || existe(raiz+"/info/orfao.trashinfo") {
		t.Errorf("arquivo sem ficha no caminho: %v", err)
	}
}

func TestMandarFalhas(t *testing.T) {
	m := prova.Novo(t, Ligar)
	for _, ruim := range []string{"", "/", m.Casa, filepath.Dir(m.Casa), "relativo", m.Casa + "/nao-ha"} {
		if _, err := Mandar(ruim, nil); err == nil {
			t.Errorf("Mandar(%q) devia falhar", ruim)
		}
	}
	// A própria lixeira, o que está nela e o que a contém não vão para a lixeira.
	a := m.Grava("a", "x")
	e, _ := Mandar(a, nil)
	for _, ruim := range []string{e.Arquivo, e.Lixeira, filepath.Dir(e.Lixeira)} {
		if _, err := Mandar(ruim, nil); err == nil || !existe(ruim) {
			t.Errorf("Mandar(%q) devia falhar", ruim)
		}
	}
	// Se mover falha, a ficha não fica.
	b := m.Grava("b", "x")
	if _, err := Mandar(b, func(de, para string) error { return errors.New("não deu") }); err == nil || !existe(b) ||
		existe(e.Lixeira+"/info/b.trashinfo") {
		t.Errorf("mover que falha: %v", err)
	}
}

func TestOutroDisco(t *testing.T) {
	m := prova.Novo(t, Ligar)
	disco := m.Pasta("disco")
	m.Montagem(disco, "/dev/sdz1")
	uid := strconv.Itoa(os.Getuid())
	a := m.Grava("disco/sub dir/a.txt", "x")
	e, err := Mandar(a, nil)
	if err != nil {
		t.Fatal(err)
	}
	raiz := disco + "/.Trash-" + uid
	if e.Arquivo != raiz+"/files/a.txt" || e.Lixeira != raiz {
		t.Fatalf("a lixeira do outro disco: %+v", e)
	}
	ficha, _ := os.ReadFile(e.Info)
	if !strings.Contains(string(ficha), "\nPath=sub%20dir/a.txt\n") {
		t.Errorf("o Path é relativo ao ponto de montagem: %q", ficha)
	}
	if info, _ := os.Stat(raiz); info.Mode().Perm() != 0o700 {
		t.Errorf("modo da lixeira do disco: %v", info.Mode())
	}
	if lido, err := Ler(e.Arquivo); err != nil || lido.Origem != a {
		t.Errorf("Ler devolve o caminho inteiro: %+v %v", lido, err)
	}
	if ts := Todas(); len(ts) != 1 || ts[0].Ponto != disco {
		t.Errorf("Todas (a de casa ainda não existe): %+v", ts)
	}
	Mandar(m.Grava("c", "x"), nil)
	if len(Todas()) != 2 || Contar() != 2 {
		t.Errorf("as duas lixeiras: %+v", Todas())
	}

	// Disco sem como criar a lixeira: ErroSemLixeira, e o arquivo fica.
	if os.Getuid() == 0 {
		return
	}
	fechado := m.Pasta("fechado")
	m.Montagem(fechado, "/dev/sdz2")
	b := m.Grava("fechado/b.txt", "x")
	os.Chmod(fechado, 0o555)
	t.Cleanup(func() { os.Chmod(fechado, 0o755) })
	if _, err := Mandar(b, nil); !errors.Is(err, ErroSemLixeira) || !existe(b) {
		t.Errorf("disco só de leitura: %v", err)
	}
	// Um link no lugar da .Trash-uid não é seguido.
	os.Chmod(fechado, 0o755)
	os.Symlink(m.Casa, fechado+"/.Trash-"+uid)
	if _, err := Mandar(b, nil); !errors.Is(err, ErroSemLixeira) || !existe(b) {
		t.Errorf("link no lugar da lixeira: %v", err)
	}
}

func TestListarEmLotesECancelar(t *testing.T) {
	m := prova.Novo(t, Ligar)
	for i := 0; i < 7; i++ {
		Mandar(m.Grava("f"+strconv.Itoa(i), "x"), nil)
	}
	t.Setenv("MKFILES_LOTE", "3")
	linhas := m.PedeFluxo(`{"op":"lixeira-listar"}`)
	if len(linhas) != 3 || len(linhas[0]["itens"].([]any)) != 3 || linhas[0]["fim"] != false || linhas[2]["total"] != float64(7) {
		t.Errorf("lotes: %v", linhas)
	}
	// Ficha torta (sem Path, ou relativa na lixeira de casa) não aparece.
	m.Grava(".local/share/Trash/files/t1", "x")
	m.Grava(".local/share/Trash/info/t1.trashinfo", "[Trash Info]\nDeletionDate=2020-01-01T00:00:00\n")
	m.Grava(".local/share/Trash/files/t2", "x")
	m.Grava(".local/share/Trash/info/t2.trashinfo", "[Trash Info]\nPath=relativo/t2\n")
	if Contar() != 7 {
		t.Errorf("fichas tortas: %d", Contar())
	}
}
