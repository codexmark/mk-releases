package props

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"mkfiles/internal/prova"
)

func motor(t *testing.T) *prova.Motor {
	t.Helper()
	m := prova.Novo(t, Ligar)
	m.Grava(".etc/passwd", fmt.Sprintf("ana:x:%d:%d::/home/ana:/bin/sh\n", os.Getuid(), os.Getgid()))
	m.Grava(".etc/group", fmt.Sprintf("turma:x:%d:\nroda:x:4242:ana\n", os.Getgid()))
	t.Setenv("MKFILES_ETC", m.Casa+"/.etc")
	t.Setenv("MKFILES_MIME", m.Casa+"/.mime")
	return m
}

func modo(t *testing.T, c string) os.FileMode {
	t.Helper()
	info, err := os.Lstat(c)
	if err != nil {
		t.Fatal(err)
	}
	return info.Mode().Perm()
}

func TestPropriedades(t *testing.T) {
	m := motor(t)
	fora := m.Grava("fora/grande.bin", "0123456789")
	pasta := m.Pasta("p")
	m.Grava("p/a.txt", "abc")
	m.Grava("p/sub/b.txt", "de")
	m.Grava("p/sub/fundo/c.txt", "f")
	// Um link para fora não é seguido: não conta o que há do outro lado.
	os.Symlink(filepath.Dir(fora), pasta+"/atalho")
	solto := m.Grava("solto.txt", "1234")

	linhas := m.PedeFluxo(`{"op":"propriedades","caminhos":` + prova.J([]string{pasta, solto}) + `}`)
	primeira, fim := linhas[0], linhas[len(linhas)-1]
	itens := primeira["itens"].([]any)
	if len(itens) != 2 || itens[0].(map[string]any)["tipo"] != "pasta" || itens[1].(map[string]any)["nome"] != "solto.txt" ||
		primeira["onde"] != m.Casa || primeira["sou_dono"] != true || primeira["fim"] != false {
		t.Errorf("a primeira linha: %v", primeira)
	}
	if g := primeira["grupos"].([]any); len(g) != 2 || g[0] != "turma" || g[1] != "roda" {
		t.Errorf("grupos: %v", g)
	}
	// Na primeira linha já vai o que não precisa de contagem: o arquivo solto.
	if c := primeira["contagem"].(map[string]any); c["arquivos"] != float64(1) || c["tamanho"] != float64(4) || c["pastas"] != float64(0) {
		t.Errorf("contagem inicial: %v", c)
	}
	c := fim["contagem"].(map[string]any)
	if fim["fim"] != true || c["arquivos"] != float64(5) || c["pastas"] != float64(2) || c["tamanho"] != float64(10) {
		t.Errorf("contagem final (3 arquivos + o link + o solto; 2 pastas; 3+2+1+4 bytes): %v", fim)
	}

	// Itens de pastas diferentes: não há um "onde".
	linhas = m.PedeFluxo(`{"op":"propriedades","caminhos":` + prova.J([]string{fora, solto}) + `}`)
	if linhas[0]["onde"] != nil || len(linhas) != 2 || linhas[1]["contagem"].(map[string]any)["tamanho"] != float64(14) {
		t.Errorf("vários de pastas diferentes: %v", linhas)
	}
	// Um link: é do link que se fala.
	linhas = m.PedeFluxo(`{"op":"propriedades","caminhos":` + prova.J([]string{pasta + "/atalho"}) + `}`)
	if it := linhas[0]["itens"].([]any)[0].(map[string]any); it["tipo"] != "link" || it["alvo"] != filepath.Dir(fora) ||
		linhas[len(linhas)-1]["contagem"].(map[string]any)["arquivos"] != float64(1) {
		t.Errorf("link: %v", linhas)
	}
	for _, ruim := range []string{`[]`, `["relativo"]`, `["/a/../b"]`, prova.J([]string{m.Casa + "/nao-ha"})} {
		if r := m.Pede(`{"op":"propriedades","caminhos":` + ruim + `}`); r["erro"] == nil {
			t.Errorf("devia falhar: %s", ruim)
		}
	}
}

func TestPropriedadesEmFluxoECancelavel(t *testing.T) {
	m := motor(t)
	passoDaContagem = 0 // uma linha por arquivo: dá para ver o fluxo e cancelar no meio
	t.Cleanup(func() { passoDaContagem = 200 * time.Millisecond })
	pasta := m.Pasta("muitos")
	const total = 3000
	for i := 0; i < total; i++ {
		m.Grava(fmt.Sprintf("muitos/%04d", i), "x")
	}
	id := m.Manda(`{"op":"propriedades","caminhos":` + prova.J([]string{pasta}) + `}`)
	m.Resposta(id)
	parcial := m.Resposta(id)
	if parcial["fim"] != false || parcial["contagem"] == nil {
		t.Fatalf("contagem parcial: %v", parcial)
	}
	m.Pede(fmt.Sprintf(`{"op":"cancelar","pedido":%d}`, id))
	resto := m.Fluxo(id)
	fim := resto[len(resto)-1]
	if fim["cancelada"] != true || fim["contagem"].(map[string]any)["arquivos"].(float64) >= total {
		t.Errorf("cancelar no meio: %v", fim)
	}
}

func TestPermissoes(t *testing.T) {
	m := motor(t)
	a := m.Grava("a.sh", "#!/bin/sh\n")
	r := m.Pede(`{"op":"permissoes","caminhos":` + prova.J([]string{a}) + `,"modo":384,"executavel":null,"grupo":null,"recursivo":false}`)
	if modo(t, a) != 0o600 || r["itens"].([]any)[0].(map[string]any)["permissoes"] != "rw-------" || len(r["erros"].([]any)) != 0 {
		t.Errorf("modo 0600: %v (%v)", modo(t, a), r)
	}
	// Executável: o x de quem já tem o r.
	m.Pede(`{"op":"permissoes","caminhos":` + prova.J([]string{a}) + `,"executavel":true}`)
	if modo(t, a) != 0o700 {
		t.Errorf("executável em 0600: %v", modo(t, a))
	}
	os.Chmod(a, 0o640)
	m.Pede(`{"op":"permissoes","caminhos":` + prova.J([]string{a}) + `,"executavel":true}`)
	if modo(t, a) != 0o750 {
		t.Errorf("executável em 0640: %v", modo(t, a))
	}
	m.Pede(`{"op":"permissoes","caminhos":` + prova.J([]string{a}) + `,"executavel":false}`)
	if modo(t, a) != 0o640 {
		t.Errorf("tirar o executável: %v", modo(t, a))
	}
	// Setgid e sticky chegam ao disco.
	pasta := m.Pasta("compartilhada")
	m.Pede(`{"op":"permissoes","caminhos":` + prova.J([]string{pasta}) + `,"modo":1021}`) // 01775
	if info, _ := os.Stat(pasta); info.Mode()&os.ModeSticky == 0 || info.Mode().Perm() != 0o775 {
		t.Errorf("sticky: %v", info.Mode())
	}
	// O grupo: o próprio (o dono sempre pode); um desconhecido é recusado.
	if r := m.Pede(`{"op":"permissoes","caminhos":` + prova.J([]string{a}) + `,"grupo":"turma"}`); len(r["erros"].([]any)) != 0 ||
		r["itens"].([]any)[0].(map[string]any)["grupo"] != "turma" {
		t.Errorf("grupo: %v", r)
	}
	if r := m.Pede(`{"op":"permissoes","caminhos":` + prova.J([]string{a}) + `,"grupo":"nao-ha"}`); r["campo"] != "grupo" {
		t.Errorf("grupo desconhecido: %v", r)
	}
	// Um link nunca é seguido: o alvo fica como estava.
	link := m.Casa + "/link"
	os.Symlink(a, link)
	r = m.Pede(`{"op":"permissoes","caminhos":` + prova.J([]string{link}) + `,"modo":511}`)
	if modo(t, a) != 0o640 || len(r["erros"].([]any)) != 1 {
		t.Errorf("link: alvo com %v, %v", modo(t, a), r)
	}
	// Um que não existe vira erro do item; os outros seguem.
	r = m.Pede(`{"op":"permissoes","caminhos":` + prova.J([]string{m.Casa + "/nao-ha", a}) + `,"modo":420}`)
	if modo(t, a) != 0o644 || len(r["erros"].([]any)) != 1 || len(r["itens"].([]any)) != 1 {
		t.Errorf("um que não existe: %v", r)
	}
	for _, ruim := range []string{
		`{"op":"permissoes","caminhos":[],"modo":420}`,
		`{"op":"permissoes","caminhos":["relativo"],"modo":420}`,
		`{"op":"permissoes","caminhos":` + prova.J([]string{a}) + `,"modo":4096}`,
		`{"op":"permissoes","caminhos":` + prova.J([]string{a}) + `,"modo":-1}`,
		`{"op":"permissoes","caminhos":` + prova.J([]string{a}) + `,"modo":"0644"}`,
		`{"op":"permissoes","caminhos":` + prova.J([]string{a}) + `}`,
	} {
		if r := m.Pede(ruim); r["erro"] == nil {
			t.Errorf("devia falhar: %s -> %v", ruim, r)
		}
	}
	if modo(t, a) != 0o644 {
		t.Errorf("um pedido recusado mexeu no arquivo: %v", modo(t, a))
	}
}

func TestPermissoesRecursivas(t *testing.T) {
	m := motor(t)
	fora := m.Grava("fora/intocado.txt", "x")
	pasta := m.Pasta("p")
	arq := m.Grava("p/a.txt", "x")
	fundo := m.Grava("p/sub/fundo/b.txt", "x")
	os.Symlink(fora, pasta+"/atalho")
	os.Symlink(filepath.Dir(fora), pasta+"/atalho-pasta")

	// Sem "recursivo" só a pasta muda.
	m.Pede(`{"op":"permissoes","caminhos":` + prova.J([]string{pasta}) + `,"modo":488}`) // 0750
	if modo(t, pasta) != 0o750 || modo(t, arq) != 0o644 {
		t.Errorf("sem recursivo: %v %v", modo(t, pasta), modo(t, arq))
	}
	// Com: o modo vale para o conteúdo; nas pastas de dentro, quem tem r ganha x.
	r := m.Pede(`{"op":"permissoes","caminhos":` + prova.J([]string{pasta}) + `,"modo":416,"recursivo":true}`) // 0640
	if len(r["erros"].([]any)) != 0 {
		t.Errorf("erros: %v", r["erros"])
	}
	os.Chmod(pasta, 0o750) // para poder olhar dentro
	if modo(t, arq) != 0o640 || modo(t, fundo) != 0o640 || modo(t, pasta+"/sub") != 0o750 || modo(t, pasta+"/sub/fundo") != 0o750 {
		t.Errorf("recursivo: %v %v %v %v", modo(t, arq), modo(t, fundo), modo(t, pasta+"/sub"), modo(t, pasta+"/sub/fundo"))
	}
	if info, _ := os.Stat(fora); info.Mode().Perm() != 0o644 {
		t.Errorf("seguiu um link para fora: %v", info.Mode())
	}
	if info, _ := os.Stat(filepath.Dir(fora)); info.Mode().Perm() != 0o755 {
		t.Errorf("seguiu um link de pasta para fora: %v", info.Mode())
	}
	// Executável recursivo: só os arquivos.
	m.Pede(`{"op":"permissoes","caminhos":` + prova.J([]string{pasta}) + `,"executavel":true,"recursivo":true}`)
	if modo(t, fundo) != 0o750 || modo(t, pasta+"/sub") != 0o750 {
		t.Errorf("executável recursivo: %v %v", modo(t, fundo), modo(t, pasta+"/sub"))
	}
}

func TestPermissoesRecursivasRecusadas(t *testing.T) {
	m := motor(t)
	antes := modo(t, m.Casa)
	for _, c := range []string{"/", m.Casa, filepath.Dir(m.Casa)} {
		r := m.Pede(`{"op":"permissoes","caminhos":` + prova.J([]string{c}) + `,"modo":448,"recursivo":true}`)
		if r["recusado"] != true {
			t.Errorf("recursivo em %q devia ser recusado: %v", c, r)
		}
	}
	if modo(t, m.Casa) != antes {
		t.Error("a pasta pessoal foi mexida")
	}
}

func TestRecursivoNaoFazExecutavelOQueNaoEra(t *testing.T) {
	m := motor(t)
	pasta := m.Pasta("q")
	texto := m.Grava("q/sub/nota.txt", "x")
	programa := m.Grava("q/sub/roda.sh", "#!/bin/sh\n")
	os.Chmod(programa, 0o700)
	// O modo de uma pasta tem x: aplicado ao conteúdo, as pastas de dentro
	// ficam com ele, e cada arquivo fica com o x que já tinha (ou não tinha).
	m.Pede(`{"op":"permissoes","caminhos":` + prova.J([]string{pasta}) + `,"modo":493,"recursivo":true}`) // 0755
	if modo(t, pasta) != 0o755 || modo(t, pasta+"/sub") != 0o755 || modo(t, texto) != 0o644 || modo(t, programa) != 0o744 {
		t.Errorf("0755 no conteúdo: pasta %v, sub %v, texto %v, programa %v", modo(t, pasta), modo(t, pasta+"/sub"), modo(t, texto), modo(t, programa))
	}
}
