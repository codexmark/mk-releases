package abrir

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"mkfiles/internal/mime"
	"mkfiles/internal/prova"
)

// motor monta a casa de mentira com um banco de tipos, apps do "sistema" e da
// pessoa, e um PATH só com os terminais pedidos.
func motor(t *testing.T, terminaisNoPath ...string) *prova.Motor {
	t.Helper()
	m := prova.Novo(t, Ligar)
	m.Grava(".mime/globs2", "50:text/plain:*.txt\n50:image/png:*.png\n50:application/pdf:*.pdf\n")
	t.Setenv("MKFILES_MIME", m.Casa+"/.mime")
	mime.Zerar()
	t.Cleanup(mime.Zerar)
	bin := m.Pasta(".bin")
	for _, nome := range terminaisNoPath {
		m.Grava(".bin/"+nome, "#!/bin/sh\n")
		os.Chmod(filepath.Join(bin, nome), 0o755)
	}
	t.Setenv("PATH", bin)
	t.Setenv("MKFILES_TERMINAL", "")
	sis := ".sistema/share/applications/"
	m.Grava(sis+"visor.desktop", "[Desktop Entry]\nType=Application\nName=Viewer\nName[pt_BR]=Visor\nExec=visor %F\nIcon=visor\nMimeType=image/png;image/jpeg;\n")
	m.Grava(sis+"editor.desktop", "[Desktop Entry]\nType=Application\nName=Editor\nExec=editor --novo \"%f\"\nMimeType=text/plain;\n")
	m.Grava(sis+"pintor.desktop", "[Desktop Entry]\nType=Application\nName=Pintor\nExec=pintor %U\nMimeType=image/png;\n")
	m.Grava(sis+"kde/gwen.desktop", "[Desktop Entry]\nType=Application\nName=Gwen\nExec=gwen %u\nMimeType=image/png;\n")
	m.Grava(sis+"escondido.desktop", "[Desktop Entry]\nType=Application\nName=Escondido\nExec=x %f\nNoDisplay=true\nMimeType=image/png;\n")
	m.Grava(sis+"semexec.desktop", "[Desktop Entry]\nType=Application\nName=Sem Exec\nMimeType=image/png;\n")
	m.Grava(sis+"link.desktop", "[Desktop Entry]\nType=Link\nName=Um link\nURL=http://x\n")
	m.Grava(sis+"vim.desktop", "[Desktop Entry]\nType=Application\nName=Vim\nExec=vim %F\nTerminal=true\nMimeType=text/plain;\n")
	// O da pessoa esconde o do sistema com o mesmo id.
	m.Grava(".local/share/applications/pintor.desktop", "[Desktop Entry]\nType=Application\nName=Pintor da Ana\nExec=pintor-da-ana %U\nMimeType=image/png;\n")
	return m
}

func ids(r prova.Linha) string {
	var lista []string
	for _, a := range r["apps"].([]any) {
		lista = append(lista, a.(map[string]any)["id"].(string))
	}
	return strings.Join(lista, " ")
}

func TestPalavras(t *testing.T) {
	casos := []struct {
		linha string
		args  []string
	}{
		{`visor %F`, []string{"visor", "%F"}},
		{`"/opt/meu app/bin" --x   "a b" c`, []string{"/opt/meu app/bin", "--x", "a b", "c"}},
		{`sh -c "echo \"oi\" \$HOME \\ \` + "`" + `"`, []string{"sh", "-c", "echo \"oi\" $HOME \\ `"}},
		{`a ""`, []string{"a", ""}},
		{`a b;c $(d) | e`, []string{"a", "b;c", "$(d)", "|", "e"}},
	}
	for _, c := range casos {
		args, err := palavras(c.linha)
		if err != nil || !reflect.DeepEqual(args, c.args) {
			t.Errorf("palavras(%q) = %q, %v; esperado %q", c.linha, args, err, c.args)
		}
	}
	for _, ruim := range []string{``, `   `, `"aberta`, `""`} {
		if _, err := palavras(ruim); err == nil {
			t.Errorf("palavras(%q) devia falhar", ruim)
		}
	}
}

func TestArgvs(t *testing.T) {
	a := app{ID: "x.desktop", Arquivo: "/apps/x.desktop", Nome: "Meu App", Icone: "icone-x"}
	arqs := []string{"/p/a b.txt", "/p/$(rm -rf);`x`.txt"}
	casos := []struct {
		exec string
		quer [][]string
	}{
		{"x %F", [][]string{{"x", arqs[0], arqs[1]}}},
		{"x %f", [][]string{{"x", arqs[0]}, {"x", arqs[1]}}},
		{"x %U", [][]string{{"x", "file:///p/a%20b.txt", "file:///p/$%28rm%20-rf%29%3B%60x%60.txt"}}},
		{"x %u", [][]string{{"x", "file:///p/a%20b.txt"}, {"x", "file:///p/$%28rm%20-rf%29%3B%60x%60.txt"}}},
		{"x", [][]string{{"x", arqs[0]}, {"x", arqs[1]}}},
		{"x --arquivo=%f 100%% %i %c %k %d %v", [][]string{
			{"x", "--arquivo=" + arqs[0], "100%", "--icon", "icone-x", "Meu App", "/apps/x.desktop"},
			{"x", "--arquivo=" + arqs[1], "100%", "--icon", "icone-x", "Meu App", "/apps/x.desktop"}}},
	}
	for _, c := range casos {
		a.Exec = c.exec
		tem, err := argvs(a, arqs)
		if err != nil {
			t.Errorf("argvs(%q): %v", c.exec, err)
			continue
		}
		// O URI codifica o que for preciso; o que importa é voltar ao caminho.
		if strings.Contains(c.exec, "%U") || strings.Contains(c.exec, "%u") {
			if len(tem) != len(c.quer) || len(tem[0]) != len(c.quer[0]) || !strings.HasPrefix(tem[0][1], "file:///p/a%20b.txt") {
				t.Errorf("argvs(%q) = %q", c.exec, tem)
			}
			continue
		}
		if !reflect.DeepEqual(tem, c.quer) {
			t.Errorf("argvs(%q) = %q; esperado %q", c.exec, tem, c.quer)
		}
	}
	a.Exec, a.Icone = "x %i %f", ""
	if tem, _ := argvs(a, arqs[:1]); !reflect.DeepEqual(tem, [][]string{{"x", arqs[0]}}) {
		t.Errorf("sem ícone o %%i some: %q", tem)
	}
	a.Exec = `"aberta`
	if _, err := argvs(a, arqs); err == nil {
		t.Error("Exec com aspas sem par")
	}
}

func TestApps(t *testing.T) {
	m := motor(t)
	// Sem mimeapps.list: os que declaram o tipo, o da pessoa na frente (a ordem
	// das pastas), e o primeiro é o padrão.
	r := m.Pede(`{"op":"apps","mime":"image/png","todos":false}`)
	if ids(r) != "pintor.desktop kde-gwen.desktop visor.desktop" || r["padrao"] != "pintor.desktop" || r["mime"] != "image/png" {
		t.Errorf("sem mimeapps: %s, padrão %v", ids(r), r["padrao"])
	}
	primeiro := r["apps"].([]any)[0].(map[string]any)
	if primeiro["nome"] != "Pintor da Ana" || primeiro["padrao"] != true || primeiro["icone"] != nil || primeiro["terminal"] != false {
		t.Errorf("o app da pessoa esconde o do sistema: %v", primeiro)
	}
	if v := r["apps"].([]any)[2].(map[string]any); v["nome"] != "Visor" || v["icone"] != "visor" || v["padrao"] != false {
		t.Errorf("nome em português e ícone: %v", v)
	}

	// A ordem do XDG: o da pessoa em ~/.config ganha do do sistema.
	m.Grava(".sistema/xdg/mimeapps.list", "[Default Applications]\nimage/png=visor.desktop\n[Added Associations]\nimage/png=editor.desktop;\n")
	r = m.Pede(`{"op":"apps","mime":"image/png"}`)
	if r["padrao"] != "visor.desktop" || ids(r) != "visor.desktop editor.desktop pintor.desktop kde-gwen.desktop" {
		t.Errorf("mimeapps do sistema: %s, padrão %v", ids(r), r["padrao"])
	}
	m.Grava(".config/mimeapps.list", "[Default Applications]\nimage/png=nao-instalado.desktop;kde-gwen.desktop;\n\n[Removed Associations]\nimage/png=pintor.desktop;\n")
	r = m.Pede(`{"op":"apps","mime":"image/png"}`)
	if r["padrao"] != "kde-gwen.desktop" || ids(r) != "kde-gwen.desktop editor.desktop visor.desktop" {
		t.Errorf("mimeapps da pessoa (padrão que não existe é pulado; Removed tira): %s, padrão %v", ids(r), r["padrao"])
	}
	// $XDG_DATA_HOME/applications/mimeapps.list vem depois dos de configuração.
	os.Remove(m.Casa + "/.config/mimeapps.list")
	os.Remove(m.Casa + "/.sistema/xdg/mimeapps.list")
	m.Grava(".local/share/applications/mimeapps.list", "[Default Applications]\nimage/png=visor.desktop\n")
	if r = m.Pede(`{"op":"apps","mime":"image/png"}`); r["padrao"] != "visor.desktop" {
		t.Errorf("mimeapps em applications/: %v", r["padrao"])
	}

	// Tipo sem app nenhum.
	r = m.Pede(`{"op":"apps","mime":"application/x-nada"}`)
	if r["padrao"] != nil || len(r["apps"].([]any)) != 0 {
		t.Errorf("sem app: %v", r)
	}
	// Todos: os visíveis com Exec, por nome.
	r = m.Pede(`{"op":"apps","mime":"application/x-nada","todos":true}`)
	if ids(r) != "editor.desktop kde-gwen.desktop pintor.desktop vim.desktop visor.desktop" {
		t.Errorf("todos: %s", ids(r))
	}
	for _, ruim := range []string{"", "semBarra", "a/b c", "a/b\n[x]"} {
		if r := m.Pede(`{"op":"apps","mime":` + prova.J(ruim) + `}`); r["erro"] == nil {
			t.Errorf("mime %q devia ser recusado", ruim)
		}
	}
}

func TestDefinirPadrao(t *testing.T) {
	m := motor(t)
	arq := m.Casa + "/.config/mimeapps.list"
	// Sem arquivo: cria.
	if r := m.Pede(`{"op":"definir-padrao","mime":"image/png","app":"visor.desktop"}`); r["padrao"] != "visor.desktop" {
		t.Fatalf("definir: %v", r)
	}
	if b, _ := os.ReadFile(arq); string(b) != "[Default Applications]\nimage/png=visor.desktop;\n" {
		t.Errorf("arquivo novo: %q", b)
	}
	// Com arquivo: troca a linha, acrescenta no grupo certo e preserva o resto.
	antes := "# meu comentário\n[Added Associations]\nimage/png=editor.desktop;\n\n[Default Applications]\ntext/plain=editor.desktop\nimage/png=visor.desktop;\n\n[Removed Associations]\nx/y=z.desktop\n"
	m.Grava(".config/mimeapps.list", antes)
	m.Pede(`{"op":"definir-padrao","mime":"image/png","app":"kde-gwen.desktop"}`)
	m.Pede(`{"op":"definir-padrao","mime":"application/pdf","app":"visor.desktop"}`)
	quer := "# meu comentário\n[Added Associations]\nimage/png=editor.desktop;\n\n[Default Applications]\ntext/plain=editor.desktop\nimage/png=kde-gwen.desktop;\napplication/pdf=visor.desktop;\n\n[Removed Associations]\nx/y=z.desktop\n"
	if b, _ := os.ReadFile(arq); string(b) != quer {
		t.Errorf("preservar o resto:\n%s\nesperado:\n%s", b, quer)
	}
	if r := m.Pede(`{"op":"apps","mime":"image/png"}`); r["padrao"] != "kde-gwen.desktop" {
		t.Errorf("o padrão novo vale: %v", r["padrao"])
	}
	// Não sobra temporário, e só esse arquivo foi escrito.
	if e, _ := os.ReadDir(m.Casa + "/.config"); len(e) != 1 {
		t.Errorf("sobrou arquivo em .config: %v", e)
	}
	// Sem o grupo: acrescenta no fim.
	m.Grava(".config/mimeapps.list", "[Added Associations]\na/b=c.desktop")
	m.Pede(`{"op":"definir-padrao","mime":"image/png","app":"visor.desktop"}`)
	if b, _ := os.ReadFile(arq); string(b) != "[Added Associations]\na/b=c.desktop\n\n[Default Applications]\nimage/png=visor.desktop;\n" {
		t.Errorf("sem o grupo: %q", b)
	}
	for _, ruim := range []string{"nao-ha.desktop", "escondido.desktop", "semexec.desktop", "", "visor.desktop;x"} {
		if r := m.Pede(`{"op":"definir-padrao","mime":"image/png","app":` + prova.J(ruim) + `}`); r["campo"] != "app" {
			t.Errorf("app %q devia ser recusado: %v", ruim, r)
		}
	}
	if r := m.Pede(`{"op":"definir-padrao","mime":"x\n[y]/z","app":"visor.desktop"}`); r["campo"] != "mime" {
		t.Errorf("mime com quebra de linha: %v", r)
	}
}

func lancados(r prova.Linha) [][]string {
	var todos [][]string
	for _, l := range r["lancar"].([]any) {
		var argv []string
		for _, a := range l.(map[string]any)["argv"].([]any) {
			argv = append(argv, a.(string))
		}
		todos = append(todos, argv)
	}
	return todos
}

func TestAbrir(t *testing.T) {
	m := motor(t, "kitty")
	a := m.Grava("p/a b.png", "x")
	b := m.Grava("p/$(touch pwn);`x`.png", "x")
	nota := m.Grava("p/nota.txt", "x")
	pdf := m.Grava("p/doc.pdf", "x")
	m.Grava(".config/mimeapps.list", "[Default Applications]\nimage/png=visor.desktop\ntext/plain=editor.desktop\n")

	r := m.Pede(`{"op":"abrir","caminhos":` + prova.J([]string{a, nota, b, pdf}) + `,"app":null}`)
	quer := [][]string{{"visor", a, b}, {"editor", "--novo", nota}}
	if tem := lancados(r); !reflect.DeepEqual(tem, quer) {
		t.Errorf("abrir = %q; esperado %q", tem, quer)
	}
	if l := r["lancar"].([]any)[0].(map[string]any); l["app"] != "visor.desktop" || l["nome"] != "Visor" {
		t.Errorf("de quem é o argv: %v", l)
	}
	sem := r["sem_app"].([]any)
	if len(sem) != 1 || sem[0].(map[string]any)["caminho"] != pdf || sem[0].(map[string]any)["mime"] != "application/pdf" {
		t.Errorf("sem_app: %v", sem)
	}

	// Com o app dado: todos vão para ele, um argv por arquivo se o Exec é %u.
	r = m.Pede(`{"op":"abrir","caminhos":` + prova.J([]string{a, pdf}) + `,"app":"kde-gwen.desktop"}`)
	if tem := lancados(r); len(tem) != 2 || tem[0][0] != "gwen" || tem[0][1] != "file://"+m.Casa+"/p/a%20b.png" || len(r["sem_app"].([]any)) != 0 {
		t.Errorf("com o app dado: %q", tem)
	}

	// Terminal=true: a linha do terminal do Mk, com o comando inteiro depois.
	r = m.Pede(`{"op":"abrir","caminhos":` + prova.J([]string{nota}) + `,"app":"vim.desktop"}`)
	if tem := lancados(r); !reflect.DeepEqual(tem, [][]string{{"kitty", "--title=Vim", "vim", nota}}) {
		t.Errorf("Terminal=true: %q", tem)
	}

	for _, ruim := range []string{`{"op":"abrir","caminhos":[]}`, `{"op":"abrir","caminhos":["relativo.txt"]}`,
		`{"op":"abrir","caminhos":["/a/../b"]}`, `{"op":"abrir","caminhos":` + prova.J([]string{m.Casa + "/nao-ha.txt"}) + `}`,
		`{"op":"abrir","caminhos":` + prova.J([]string{a}) + `,"app":"nao-ha.desktop"}`} {
		if r := m.Pede(ruim); r["erro"] == nil {
			t.Errorf("devia falhar: %s -> %v", ruim, r)
		}
	}
	if _, err := os.Stat(m.Casa + "/pwn"); err == nil {
		t.Error("o nome do arquivo foi executado")
	}
}

func TestAbrirSemTerminal(t *testing.T) {
	m := motor(t) // PATH sem terminal nenhum
	nota := m.Grava("nota.txt", "x")
	r := m.Pede(`{"op":"abrir","caminhos":` + prova.J([]string{nota}) + `,"app":"vim.desktop"}`)
	l := r["lancar"].([]any)[0].(map[string]any)
	if l["aviso"] == nil || !reflect.DeepEqual(lancados(r), [][]string{{"vim", nota}}) {
		t.Errorf("sem terminal: %v", l)
	}
	if r := m.Pede(`{"op":"terminal","caminho":` + prova.J(m.Casa) + `}`); r["erro"] == nil || !strings.Contains(r["erro"].(string), "não achei um terminal") {
		t.Errorf("terminal sem terminal: %v", r)
	}
}

func TestTerminal(t *testing.T) {
	m := motor(t, "kitty", "foot", "alacritty", "wezterm", "ghostty")
	pasta := m.Pasta("minha pasta")
	arquivo := m.Grava("minha pasta/a.txt", "x")
	argv := func(caminho string) []string {
		r := m.Pede(`{"op":"terminal","caminho":` + prova.J(caminho) + `}`)
		var a []string
		lista, _ := r["argv"].([]any)
		for _, x := range lista {
			a = append(a, x.(string))
		}
		return a
	}
	casos := map[string][]string{
		"kitty":     {"kitty", "--directory=" + pasta},
		"foot":      {"foot", "--working-directory=" + pasta},
		"alacritty": {"alacritty", "--working-directory", pasta},
		"wezterm":   {"wezterm", "start", "--always-new-process", "--cwd", pasta},
		"ghostty":   {"ghostty", "--config-default-files=false", "--config-file=?" + m.Casa + "/.config/mk/ghostty/config", "--working-directory=" + pasta},
	}
	for nome, quer := range casos {
		t.Setenv("MKFILES_TERMINAL", nome)
		if tem := argv(pasta); !reflect.DeepEqual(tem, quer) {
			t.Errorf("%s: %q; esperado %q", nome, tem, quer)
		}
	}
	// De um arquivo, a pasta dele. Sem a variável, o dos Ajustes; sem os dois,
	// o primeiro da lista; um terminal desconhecido não entra.
	t.Setenv("MKFILES_TERMINAL", "xterm; rm -rf")
	m.Grava(".config/mk/ajustes.json", `{"apps.terminal":"foot"}`)
	if tem := argv(arquivo); !reflect.DeepEqual(tem, casos["foot"]) {
		t.Errorf("do ajuste, na pasta do arquivo: %q", tem)
	}
	os.Remove(m.Casa + "/.config/mk/ajustes.json")
	if tem := argv(pasta); tem[0] != "ghostty" {
		t.Errorf("o primeiro que houver: %q", tem)
	}
	if r := m.Pede(`{"op":"terminal","caminho":"relativa"}`); r["erro"] == nil {
		t.Error("caminho relativo")
	}
	if r := m.Pede(`{"op":"terminal","caminho":` + prova.J(m.Casa+"/nao-ha") + `}`); r["erro"] == nil {
		t.Error("pasta que não existe")
	}
	// Com comando: cada terminal recebe o argv inteiro, um argumento por item.
	cmd := []string{"vim", "a b"}
	if l := linhaDoTerminal("alacritty", "T", "", cmd); !reflect.DeepEqual(l, []string{"alacritty", "--title", "T", "-e", "vim", "a b"}) {
		t.Errorf("alacritty: %q", l)
	}
	if l := linhaDoTerminal("wezterm", "T", "", cmd); !reflect.DeepEqual(l, []string{"wezterm", "start", "--always-new-process", "--", "vim", "a b"}) {
		t.Errorf("wezterm: %q", l)
	}
	if l := linhaDoTerminal("ghostty", "T", "", cmd); l[len(l)-3] != "-e" || l[3] != "--title=T" {
		t.Errorf("ghostty: %q", l)
	}
	if l := linhaDoTerminal("foot", "T", "", cmd); !reflect.DeepEqual(l, []string{"foot", "--title=T", "vim", "a b"}) {
		t.Errorf("foot: %q", l)
	}
}

func TestModelos(t *testing.T) {
	m := motor(t)
	if r := m.Pede(`{"op":"modelos"}`); r["pasta"] != nil || len(r["modelos"].([]any)) != 0 {
		t.Errorf("sem pasta de modelos: %v", r)
	}
	m.Grava("Templates/Carta.txt", "Prezado")
	m.Grava("Templates/.oculto", "x")
	m.Pasta("Templates/subpasta")
	r := m.Pede(`{"op":"modelos"}`)
	lista := r["modelos"].([]any)
	if r["pasta"] != m.Casa+"/Templates" || len(lista) != 1 || lista[0].(map[string]any)["nome"] != "Carta.txt" ||
		lista[0].(map[string]any)["mime"] != "text/plain" {
		t.Errorf("~/Templates: %v", r)
	}
	// A pasta do user-dirs.dirs vale mais.
	m.Grava("Modelos/Planilha.txt", "x")
	m.Grava(".config/user-dirs.dirs", "# gerado\nXDG_TEMPLATES_DIR=\"$HOME/Modelos\"\n")
	r = m.Pede(`{"op":"modelos"}`)
	if r["pasta"] != m.Casa+"/Modelos" || r["modelos"].([]any)[0].(map[string]any)["nome"] != "Planilha.txt" {
		t.Errorf("XDG_TEMPLATES_DIR: %v", r)
	}
}
