package item

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"mkfiles/internal/mime"
	"mkfiles/internal/prova"
)

func motor(t *testing.T) *prova.Motor {
	t.Helper()
	m := prova.Novo(t, Ligar)
	banco := m.Pasta(".mime")
	m.Grava(".mime/globs2", "50:text/plain:*.txt\n50:image/png:*.png\n")
	m.Grava(".mime/generic-icons", "")
	t.Setenv("MKFILES_MIME", banco)
	mime.Zerar()
	t.Cleanup(mime.Zerar)
	etc := m.Pasta(".etc")
	m.Grava(".etc/passwd", fmt.Sprintf("root:x:0:0::/root:/bin/sh\nana:x:%d:%d::/home/ana:/bin/sh\n", os.Getuid(), os.Getgid()))
	m.Grava(".etc/group", fmt.Sprintf("root:x:0:\nturma:x:%d:\nroda:x:4242:ana,bia\n", os.Getgid()))
	t.Setenv("MKFILES_ETC", etc)
	return m
}

func itens(linhas []prova.Linha) map[string]map[string]any {
	todos := map[string]map[string]any{}
	for _, l := range linhas {
		lista, _ := l["itens"].([]any)
		for _, i := range lista {
			it := i.(map[string]any)
			todos[it["nome"].(string)] = it
		}
	}
	return todos
}

func TestPermissoes(t *testing.T) {
	casos := map[uint32]string{0o644: "rw-r--r--", 0o755: "rwxr-xr-x", 0: "---------", 0o4755: "rwsr-xr-x",
		0o2644: "rw-r-Sr--", 0o1777: "rwxrwxrwt", 0o1776: "rwxrwxrwT"}
	for bits, texto := range casos {
		if Permissoes(bits) != texto {
			t.Errorf("Permissoes(%o) = %s; esperado %s", bits, Permissoes(bits), texto)
		}
	}
	if Modo(os.ModeSetuid|os.ModeSticky|0o750) != 0o5750 {
		t.Errorf("Modo: %o", Modo(os.ModeSetuid|os.ModeSticky|0o750))
	}
}

func TestListar(t *testing.T) {
	m := motor(t)
	pasta := m.Pasta("p")
	m.Grava("p/nota.txt", "olá")
	m.Grava("p/.oculto", "x")
	m.Grava("p/escondido.txt", "x")
	m.Grava("p/.hidden", "escondido.txt\n\nnao-existe\n")
	m.Grava("p/script", "#!/bin/sh\n")
	m.Grava("p/vazio", "")
	m.Pasta("p/sub")
	os.Chmod(filepath.Join(pasta, "script"), 0o755)
	os.Symlink("nota.txt", filepath.Join(pasta, "atalho"))
	os.Symlink("sub", filepath.Join(pasta, "para-pasta"))
	os.Symlink("/nao/existe", filepath.Join(pasta, "quebrado"))
	if err := syscall.Mkfifo(filepath.Join(pasta, "cano"), 0o600); err != nil {
		t.Fatal(err)
	}

	linhas := m.PedeFluxo(`{"op":"listar","caminho":` + prova.J(pasta) + `}`)
	fim := linhas[len(linhas)-1]
	if fim["fim"] != true || fim["total"] != float64(11) || fim["gravavel"] != true {
		t.Fatalf("a última linha: %v", fim)
	}
	is := itens(linhas)
	nota := is["nota.txt"]
	if nota["tipo"] != "arquivo" || nota["tamanho"] != float64(4) || nota["mime"] != "text/plain" ||
		nota["icone"] != "text-plain" || nota["icone_reserva"] != "text-x-generic" || nota["permissoes"] != "rw-r--r--" ||
		nota["modo"] != float64(0o644) || nota["dono"] != "ana" || nota["grupo"] != "turma" || nota["oculto"] != false ||
		nota["caminho"] != pasta+"/nota.txt" || nota["alvo"] != nil || nota["alvo_tipo"] != nil || nota["quebrado"] != false ||
		nota["mtime"].(float64) < 1e9 {
		t.Errorf("nota.txt: %v", nota)
	}
	if is[".oculto"]["oculto"] != true || is["escondido.txt"]["oculto"] != true || is[".hidden"]["oculto"] != true {
		t.Error("oculto: o ponto e o .hidden")
	}
	if is["script"]["mime"] != "application/x-shellscript" || is["script"]["permissoes"] != "rwxr-xr-x" {
		t.Errorf("script: %v", is["script"])
	}
	if is["vazio"]["mime"] != "application/x-zerosize" {
		t.Errorf("vazio: %v", is["vazio"])
	}
	if s := is["sub"]; s["tipo"] != "pasta" || s["mime"] != "inode/directory" || s["icone"] != "folder" || s["tamanho"] != float64(0) {
		t.Errorf("sub: %v", s)
	}
	if a := is["atalho"]; a["tipo"] != "link" || a["alvo"] != "nota.txt" || a["quebrado"] != false || a["alvo_tipo"] != "arquivo" ||
		a["mime"] != "text/plain" || a["tamanho"] != float64(4) {
		t.Errorf("atalho: %v", a)
	}
	if a := is["para-pasta"]; a["tipo"] != "link" || a["alvo_tipo"] != "pasta" || a["mime"] != "inode/directory" {
		t.Errorf("para-pasta: %v", a)
	}
	if q := is["quebrado"]; q["tipo"] != "link" || q["quebrado"] != true || q["alvo"] != "/nao/existe" || q["alvo_tipo"] != nil ||
		q["mime"] != "inode/symlink" || q["tamanho"] != float64(0) {
		t.Errorf("quebrado: %v", q)
	}
	if c := is["cano"]; c["tipo"] != "especial" || c["mime"] != "inode/fifo" {
		t.Errorf("cano: %v", c)
	}
}

func TestListarErros(t *testing.T) {
	m := motor(t)
	arquivo := m.Grava("a.txt", "x")
	fechada := m.Pasta("fechada")
	os.Chmod(fechada, 0)
	t.Cleanup(func() { os.Chmod(fechada, 0o755) })
	casos := []struct{ caminho, motivo string }{
		{m.Casa + "/nao-ha", "nao-existe"},
		{arquivo, "nao-e-pasta"},
		{"relativo", "erro"},
		{m.Casa + "/../x", "erro"},
		{"/a\x00b", "erro"},
	}
	if os.Getuid() != 0 {
		casos = append(casos, struct{ caminho, motivo string }{fechada, "sem-permissao"})
	}
	for _, c := range casos {
		r := m.Pede(`{"op":"listar","caminho":` + prova.J(c.caminho) + `}`)
		if r["erro"] == nil || r["motivo"] != c.motivo {
			t.Errorf("listar %q: %v (esperado o motivo %s)", c.caminho, r, c.motivo)
		}
	}
	// Uma pasta só de leitura lista, e diz que não é gravável.
	leitura := m.Pasta("leitura")
	os.Chmod(leitura, 0o555)
	t.Cleanup(func() { os.Chmod(leitura, 0o755) })
	if r := m.PedeFluxo(`{"op":"listar","caminho":` + prova.J(leitura) + `}`); os.Getuid() != 0 && r[0]["gravavel"] != false {
		t.Errorf("pasta só de leitura: %v", r)
	}
}

func TestListarVinteMilEmLotes(t *testing.T) {
	m := motor(t)
	pasta := m.Pasta("muitos")
	const total = 20000
	for i := 0; i < total; i++ {
		f, err := os.Create(filepath.Join(pasta, fmt.Sprintf("arquivo-%05d.txt", i)))
		if err != nil {
			t.Fatal(err)
		}
		f.Close()
	}
	linhas := m.PedeFluxo(`{"op":"listar","caminho":` + prova.J(pasta) + `}`)
	if len(linhas) < total/500 {
		t.Fatalf("esperava pelo menos %d linhas, vieram %d", total/500, len(linhas))
	}
	vistos := 0
	for i, l := range linhas {
		n := len(l["itens"].([]any))
		if n > 500 {
			t.Fatalf("lote %d com %d itens", i, n)
		}
		if (l["fim"] == true) != (i == len(linhas)-1) {
			t.Fatalf("só a última linha tem fim: true (linha %d)", i)
		}
		vistos += n
	}
	if vistos != total || linhas[len(linhas)-1]["total"] != float64(total) {
		t.Errorf("vistos %d, total %v", vistos, linhas[len(linhas)-1]["total"])
	}

	// O tamanho do lote é trocável, e o fluxo é cancelável.
	t.Setenv("MKFILES_LOTE", "7")
	id := m.Manda(`{"op":"listar","caminho":` + prova.J(pasta) + `}`)
	if n := len(m.Resposta(id)["itens"].([]any)); n != 7 {
		t.Errorf("lote de 7: %d", n)
	}
	m.Pede(fmt.Sprintf(`{"op":"cancelar","pedido":%d}`, id))
	resto := m.Fluxo(id)
	if ultimo := resto[len(resto)-1]; ultimo["cancelada"] != true || len(resto) > total/7-10 {
		t.Errorf("cancelar no meio: %d linhas, a última %v", len(resto), ultimo)
	}
}

func TestItem(t *testing.T) {
	m := motor(t)
	// O pedido item fareja sempre: uma extensão desconhecida não esconde um PNG.
	c := m.Grava("foto.xyz", "\x89PNG\r\n\x1a\n....")
	r := m.Pede(`{"op":"item","caminho":` + prova.J(c) + `}`)
	if it, _ := r["item"].(map[string]any); it == nil || it["mime"] != "image/png" || it["nome"] != "foto.xyz" {
		t.Errorf("item: %v", r)
	}
	if r := m.Pede(`{"op":"item","caminho":` + prova.J(m.Casa+"/nao-ha") + `}`); r["motivo"] != "nao-existe" {
		t.Errorf("item que não existe: %v", r)
	}
	if r := m.Pede(`{"op":"item","caminho":"a/b"}`); r["erro"] == nil {
		t.Errorf("caminho relativo: %v", r)
	}
	m.Grava("x/.hidden", "b\n")
	m.Grava("x/b", "1")
	if it, _ := De(m.Casa+"/x/b", false); !it.Oculto {
		t.Error("De: o .hidden da pasta vale")
	}
}

func TestDonos(t *testing.T) {
	motor(t)
	if nomeDoDono(0) != "root" || nomeDoDono(987654) != "987654" || nomeDoGrupo(4242) != "roda" || nomeDoGrupo(987654) != "987654" {
		t.Error("nomes de dono e grupo")
	}
	if g := GruposDe(uint32(os.Getuid()), uint32(os.Getgid())); len(g) != 2 || g[0] != "turma" || g[1] != "roda" {
		t.Errorf("GruposDe: %v", g)
	}
	if n, ok := GidDe("roda"); !ok || n != 4242 {
		t.Error("GidDe")
	}
	if _, ok := GidDe("nao-ha"); ok {
		t.Error("GidDe de grupo que não existe")
	}
}
