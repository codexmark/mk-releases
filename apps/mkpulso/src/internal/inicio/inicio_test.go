package inicio

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"mkpulso/internal/acoes"
	"mkpulso/internal/servicos/dubles"
)

// maquina é a home e o /etc falsos de um teste, com o dublê do systemctl.
type maquina struct {
	t          *testing.T
	home, etc  string
	usuario    string // ~/.config/autostart
	sistema    string // /etc/xdg/autostart
	mesa       *dubles.Mesa
	habilitada string // ~/.config/systemd/user
}

func novaMaquina(t *testing.T) *maquina {
	t.Helper()
	m := &maquina{t: t, home: t.TempDir(), etc: t.TempDir()}
	m.usuario = filepath.Join(m.home, ".config", "autostart")
	m.sistema = filepath.Join(m.etc, "xdg", "autostart")
	m.habilitada = filepath.Join(m.home, ".config", "systemd", "user")
	t.Setenv("MKPULSO_HOME", m.home)
	t.Setenv("MKPULSO_ETC", m.etc)
	t.Setenv("XDG_CURRENT_DESKTOP", "Hyprland")
	m.mesa = dubles.Nova(t)
	m.mesa.Comando("systemctl",
		dubles.Regra{Contem: " --user list-unit-files --state=enabled ", Texto: "[]\n"},
		dubles.Regra{Contem: ""},
	)
	return m
}

// comUnidades troca o dublê por um com as unidades gravadas em testdata.
func (m *maquina) comUnidades() {
	m.t.Helper()
	m.mesa.Comando("systemctl",
		dubles.Regra{Contem: " --user list-unit-files --state=enabled ", Arquivo: "testdata/habilitadas.json"},
		dubles.Regra{Contem: " --user show ", Arquivo: "testdata/habilitadas-show.txt"},
		dubles.Regra{Contem: ""},
	)
	m.escreve(filepath.Join(m.habilitada, "default.target.wants", "syncthing.service"), "")
	m.escreve(filepath.Join(m.habilitada, "timers.target.wants", "backup.timer"), "")
}

func (m *maquina) escreve(caminho, conteudo string) {
	m.t.Helper()
	if err := os.MkdirAll(filepath.Dir(caminho), 0o755); err != nil {
		m.t.Fatal(err)
	}
	if err := os.WriteFile(caminho, []byte(conteudo), 0o644); err != nil {
		m.t.Fatal(err)
	}
}

func (m *maquina) le(caminho string) string {
	m.t.Helper()
	b, err := os.ReadFile(caminho)
	if err != nil {
		m.t.Fatal(err)
	}
	return string(b)
}

func (m *maquina) existe(caminho string) bool {
	_, err := os.Lstat(caminho)
	return err == nil
}

func (m *maquina) item(id string) Item {
	m.t.Helper()
	itens, err := Lista()
	if err != nil {
		m.t.Fatal(err)
	}
	for _, i := range itens {
		if i.ID == id {
			return i
		}
	}
	m.t.Fatalf("%s não está na lista: %+v", id, itens)
	return Item{}
}

// semSobras confere que a escrita atômica não largou arquivo temporário.
func (m *maquina) semSobras() {
	m.t.Helper()
	entradas, _ := os.ReadDir(m.usuario)
	for _, e := range entradas {
		if !strings.HasSuffix(e.Name(), ".desktop") || strings.HasPrefix(e.Name(), ".") {
			m.t.Errorf("sobrou %s em ~/.config/autostart", e.Name())
		}
	}
}

const foo = `[Desktop Entry]
Type=Application
Name=Foo
Name[pt_BR]=Fu
Exec=foo --tray %U
# um comentário que fica

[Desktop Action x]
Name=X
Exec=foo -x
`

const fooDesligado = `[Desktop Entry]
Type=Application
Name=Foo
Name[pt_BR]=Fu
Exec=foo --tray %U
# um comentário que fica
Hidden=true

[Desktop Action x]
Name=X
Exec=foo -x
`

func TestLista_AtalhosDoSistemaEDoUsuario(t *testing.T) {
	m := novaMaquina(t)
	m.escreve(filepath.Join(m.sistema, "foo.desktop"), foo)
	m.escreve(filepath.Join(m.usuario, "bar.desktop"), "[Desktop Entry]\nName=Bar\nExec=\"/opt/bar/bar\" --minimizado %f %%\n")
	m.escreve(filepath.Join(m.sistema, "leiame.txt"), "não é atalho")
	m.escreve(filepath.Join(m.sistema, "pasta.desktop", "x"), "uma pasta não é atalho")
	m.escreve(filepath.Join(m.sistema, "semnome.desktop"), "[Desktop Entry]\nExec=semnome\n")
	itens, err := Lista()
	if err != nil {
		t.Fatal(err)
	}
	quer := []Item{
		{ID: "desktop:" + filepath.Join(m.usuario, "bar.desktop"), Origem: "atalho", Nome: "Bar", Comando: `"/opt/bar/bar" --minimizado %`,
			Ligado: true, DoSistema: false, Mudavel: true},
		{ID: "desktop:" + filepath.Join(m.sistema, "foo.desktop"), Origem: "atalho", Nome: "Fu", Comando: "foo --tray",
			Ligado: true, DoSistema: true, Mudavel: true},
		{ID: "desktop:" + filepath.Join(m.sistema, "semnome.desktop"), Origem: "atalho", Nome: "semnome", Comando: "semnome",
			Ligado: true, DoSistema: true, Mudavel: true},
	}
	if !reflect.DeepEqual(itens, quer) {
		t.Errorf("veio %+v\nquer %+v", itens, quer)
	}
}

func TestLista_SemNadaEListaVaziaNoJSON(t *testing.T) {
	novaMaquina(t)
	itens, err := Lista()
	if err != nil || itens == nil || len(itens) != 0 {
		t.Fatalf("veio %v %v", itens, err)
	}
}

func TestLista_CamposDoContratoNoJSON(t *testing.T) {
	m := novaMaquina(t)
	m.escreve(filepath.Join(m.sistema, "foo.desktop"), "[Desktop Entry]\nName=Foo\nExec=foo --tray\n")
	b, _ := json.Marshal(m.item("desktop:" + filepath.Join(m.sistema, "foo.desktop")))
	quer := `{"id":"desktop:` + filepath.Join(m.sistema, "foo.desktop") + `","origem":"atalho","nome":"Foo","comando":"foo --tray","ligado":true,"do_sistema":true,"mudavel":true}`
	if string(b) != quer {
		t.Errorf("veio %s\nquer %s", b, quer)
	}
}

func TestLista_RespeitaHiddenEOAmbiente(t *testing.T) {
	m := novaMaquina(t)
	casos := []struct {
		arquivo, extra string
		aparece        bool
		ligado         bool
	}{
		{"normal", "", true, true},
		{"escondido", "Hidden=true\n", true, false},
		{"escondido-falso", "Hidden=false\n", true, true},
		{"gnome-desligado", "X-GNOME-Autostart-enabled=false\n", true, false},
		{"gnome-ligado", "X-GNOME-Autostart-enabled=true\n", true, true},
		{"so-gnome", "OnlyShowIn=GNOME;\n", false, false},
		{"so-kde-ou-gnome", "OnlyShowIn=KDE;GNOME;\n", false, false},
		{"tambem-hyprland", "OnlyShowIn=GNOME;Hyprland;\n", true, true},
		{"menos-hyprland", "NotShowIn=KDE;Hyprland\n", false, false},
		{"menos-kde", "NotShowIn=KDE;\n", true, true},
		{"sem-o-programa", "TryExec=/nao/existe/programa\n", false, false},
		{"sem-o-programa-no-path", "TryExec=programa-que-nao-existe\n", false, false},
		{"com-o-programa", "TryExec=systemctl\n", true, true},
	}
	for _, c := range casos {
		m.escreve(filepath.Join(m.sistema, c.arquivo+".desktop"), "[Desktop Entry]\nName="+c.arquivo+"\nExec=x\n"+c.extra+"\n[Outro Grupo]\nHidden=true\nOnlyShowIn=GNOME;\n")
	}
	itens, err := Lista()
	if err != nil {
		t.Fatal(err)
	}
	veio := map[string]Item{}
	for _, i := range itens {
		veio[i.Nome] = i
	}
	for _, c := range casos {
		i, aparece := veio[c.arquivo]
		if aparece != c.aparece || (aparece && i.Ligado != c.ligado) {
			t.Errorf("%s: aparece=%v ligado=%v, queria aparece=%v ligado=%v", c.arquivo, aparece, i.Ligado, c.aparece, c.ligado)
		}
	}
	// O ambiente é uma lista: "Mk:Hyprland" também é Hyprland.
	t.Setenv("XDG_CURRENT_DESKTOP", "Mk:Hyprland")
	if itens, _ := Lista(); len(itens) != len(veio) {
		t.Errorf("com XDG_CURRENT_DESKTOP=Mk:Hyprland vieram %d, queria %d", len(itens), len(veio))
	}
	// Sem a variável, o Mk é Hyprland do mesmo jeito.
	t.Setenv("XDG_CURRENT_DESKTOP", "")
	if itens, _ := Lista(); len(itens) != len(veio) {
		t.Errorf("sem XDG_CURRENT_DESKTOP vieram %d, queria %d", len(itens), len(veio))
	}
}

func TestLista_OAtalhoDoUsuarioSobrepoeODoSistema(t *testing.T) {
	m := novaMaquina(t)
	id := "desktop:" + filepath.Join(m.sistema, "foo.desktop")
	m.escreve(filepath.Join(m.sistema, "foo.desktop"), foo)
	m.escreve(filepath.Join(m.usuario, "foo.desktop"), "[Desktop Entry]\nName=Meu Foo\nExec=foo --do-meu-jeito\n")
	itens, _ := Lista()
	if len(itens) != 1 {
		t.Fatalf("queria um item só, vieram %+v", itens)
	}
	quer := Item{ID: id, Origem: "atalho", Nome: "Meu Foo", Comando: "foo --do-meu-jeito", Ligado: true, DoSistema: true, Mudavel: true}
	if itens[0] != quer {
		t.Errorf("veio %+v\nquer %+v", itens[0], quer)
	}
	// A cópia mínima que só desliga: o nome e o comando são os do sistema.
	m.escreve(filepath.Join(m.usuario, "foo.desktop"), "[Desktop Entry]\nHidden=true\n")
	quer = Item{ID: id, Origem: "atalho", Nome: "Fu", Comando: "foo --tray", Ligado: false, DoSistema: true, Mudavel: true}
	if veio := m.item(id); veio != quer {
		t.Errorf("veio %+v\nquer %+v", veio, quer)
	}
}

func TestMuda_DesligarAtalhoDoSistemaECopiaComHiddenELigarApagaACopia(t *testing.T) {
	m := novaMaquina(t)
	id := "desktop:" + filepath.Join(m.sistema, "foo.desktop")
	m.escreve(filepath.Join(m.sistema, "foo.desktop"), foo)
	copia := filepath.Join(m.usuario, "foo.desktop")

	if err := Muda(id, false); err != nil {
		t.Fatal(err)
	}
	if veio := m.le(copia); veio != fooDesligado {
		t.Errorf("a cópia:\n%s\nqueria:\n%s", veio, fooDesligado)
	}
	if m.le(filepath.Join(m.sistema, "foo.desktop")) != foo {
		t.Error("mexeu no arquivo do sistema")
	}
	if i := m.item(id); i.Ligado || !i.DoSistema || i.Nome != "Fu" {
		t.Errorf("depois de desligar: %+v", i)
	}
	m.semSobras()
	// Desligar de novo não muda nada.
	if err := Muda(id, false); err != nil || m.le(copia) != fooDesligado {
		t.Errorf("desligar duas vezes: %v", err)
	}

	if err := Muda(id, true); err != nil {
		t.Fatal(err)
	}
	if m.existe(copia) {
		t.Errorf("a cópia devia ter sido apagada:\n%s", m.le(copia))
	}
	if i := m.item(id); !i.Ligado {
		t.Errorf("depois de ligar: %+v", i)
	}
	// Ligar o que já está ligado não cria nada.
	if err := Muda(id, true); err != nil || m.existe(copia) {
		t.Errorf("ligar duas vezes: %v", err)
	}
	m.semSobras()
}

func TestMuda_LigarPeloIdDaCopiaTambemVale(t *testing.T) {
	m := novaMaquina(t)
	m.escreve(filepath.Join(m.sistema, "foo.desktop"), foo)
	m.escreve(filepath.Join(m.usuario, "foo.desktop"), fooDesligado)
	if err := Muda("desktop:"+filepath.Join(m.usuario, "foo.desktop"), true); err != nil {
		t.Fatal(err)
	}
	if m.existe(filepath.Join(m.usuario, "foo.desktop")) {
		t.Error("a cópia devia ter sido apagada")
	}
}

func TestMuda_LigarACopiaMinimaApaga(t *testing.T) {
	m := novaMaquina(t)
	id := "desktop:" + filepath.Join(m.sistema, "foo.desktop")
	m.escreve(filepath.Join(m.sistema, "foo.desktop"), foo)
	m.escreve(filepath.Join(m.usuario, "foo.desktop"), "[Desktop Entry]\nHidden=true\n")
	if err := Muda(id, true); err != nil {
		t.Fatal(err)
	}
	if m.existe(filepath.Join(m.usuario, "foo.desktop")) {
		t.Error("a cópia mínima devia ter sido apagada")
	}
	if !m.item(id).Ligado {
		t.Error("não ligou")
	}
}

func TestMuda_LigarCopiaQueMudouOutraCoisaSoTiraOHidden(t *testing.T) {
	m := novaMaquina(t)
	id := "desktop:" + filepath.Join(m.sistema, "foo.desktop")
	m.escreve(filepath.Join(m.sistema, "foo.desktop"), foo)
	copia := filepath.Join(m.usuario, "foo.desktop")
	m.escreve(copia, "[Desktop Entry]\nName=Meu Foo\nHidden=true\nExec=foo --do-meu-jeito\n")
	if err := Muda(id, true); err != nil {
		t.Fatal(err)
	}
	if veio := m.le(copia); veio != "[Desktop Entry]\nName=Meu Foo\nExec=foo --do-meu-jeito\n" {
		t.Errorf("a cópia ficou:\n%s", veio)
	}
	if i := m.item(id); !i.Ligado || i.Comando != "foo --do-meu-jeito" {
		t.Errorf("%+v", i)
	}
	m.semSobras()
}

func TestMuda_AtalhoSoDoUsuarioMudaNoLugar(t *testing.T) {
	m := novaMaquina(t)
	caminho := filepath.Join(m.usuario, "bar.desktop")
	id := "desktop:" + caminho
	original := "[Desktop Entry]\nName=Bar\nExec=bar\n\n[Desktop Action y]\nName=Y\n"
	m.escreve(caminho, original)
	if err := Muda(id, false); err != nil {
		t.Fatal(err)
	}
	if veio := m.le(caminho); veio != "[Desktop Entry]\nName=Bar\nExec=bar\nHidden=true\n\n[Desktop Action y]\nName=Y\n" {
		t.Errorf("desligado:\n%s", veio)
	}
	if m.item(id).Ligado {
		t.Error("não desligou")
	}
	if err := Muda(id, true); err != nil {
		t.Fatal(err)
	}
	if veio := m.le(caminho); veio != original {
		t.Errorf("religado:\n%s", veio)
	}
	m.semSobras()
}

func TestMuda_LigarOQueOGnomeDesligou(t *testing.T) {
	m := novaMaquina(t)
	caminho := filepath.Join(m.usuario, "bar.desktop")
	m.escreve(caminho, "[Desktop Entry]\nName=Bar\nX-GNOME-Autostart-enabled=false\nExec=bar\nHidden=true\n")
	if err := Muda("desktop:"+caminho, true); err != nil {
		t.Fatal(err)
	}
	if veio := m.le(caminho); veio != "[Desktop Entry]\nName=Bar\nX-GNOME-Autostart-enabled=true\nExec=bar\n" {
		t.Errorf("ficou:\n%s", veio)
	}
}

func TestMuda_LigarOQueOSistemaTrazDesligadoCriaACopia(t *testing.T) {
	m := novaMaquina(t)
	id := "desktop:" + filepath.Join(m.sistema, "foo.desktop")
	m.escreve(filepath.Join(m.sistema, "foo.desktop"), "[Desktop Entry]\nName=Foo\nExec=foo\nHidden=true\n")
	if m.item(id).Ligado {
		t.Fatal("devia vir desligado")
	}
	if err := Muda(id, true); err != nil {
		t.Fatal(err)
	}
	copia := filepath.Join(m.usuario, "foo.desktop")
	if veio := m.le(copia); veio != "[Desktop Entry]\nName=Foo\nExec=foo\n" {
		t.Errorf("a cópia:\n%s", veio)
	}
	if !m.item(id).Ligado {
		t.Error("não ligou")
	}
	if err := Muda(id, false); err != nil {
		t.Fatal(err)
	}
	if m.item(id).Ligado || !m.existe(copia) {
		t.Error("não desligou de novo")
	}
	// A cópia mínima por cima de um atalho que o sistema traz desligado.
	m.escreve(copia, "[Desktop Entry]\nHidden=true\n")
	if err := Muda(id, true); err != nil {
		t.Fatal(err)
	}
	if !m.item(id).Ligado {
		t.Errorf("não ligou a partir da cópia mínima:\n%s", m.le(copia))
	}
}

func TestMuda_EntradaHostilNaoTocaEmNada(t *testing.T) {
	m := novaMaquina(t)
	m.comUnidades()
	m.escreve(filepath.Join(m.sistema, "foo.desktop"), foo)
	m.escreve(filepath.Join(m.etc, "segredo.desktop"), "[Desktop Entry]\nName=Fora\n")
	m.escreve(filepath.Join(m.sistema, "sub", "x.desktop"), "[Desktop Entry]\nName=Sub\n")
	m.escreve(filepath.Join(m.home, ".config", "mk", "hypr", "mare", "autostart.lua"), `hl.exec_cmd("udiskie")`+"\n")
	ids := []string{
		"", "desktop:", "coisa:x", "foo.desktop", "mk:udiskie", "mk:mare", "mk:",
		"desktop:../../etc/passwd",
		"desktop:" + filepath.Join(m.sistema, "..", "..", "segredo.desktop"),
		"desktop:" + m.sistema + "/../../segredo.desktop",
		"desktop:" + m.sistema + "/../autostart/foo.desktop",
		"desktop:" + filepath.Join(m.etc, "segredo.desktop"),
		"desktop:" + filepath.Join(m.sistema, "sub", "x.desktop"),
		"desktop:" + filepath.Join(m.sistema, "naoexiste.desktop"),
		"desktop:" + filepath.Join(m.sistema, "leiame.txt"),
		"desktop:" + m.usuario + "/../autostart/foo.desktop",
		"desktop:xdg/autostart/foo.desktop",
		"desktop:/etc/passwd",
		"unidade:", "unidade:--now", "unidade:a.service;rm -rf", "unidade:../x.service", "unidade:syncthing", "unidade:tmp.mount",
		"unidade:pipewire.socket\n",
	}
	for _, id := range ids {
		for _, ligar := range []bool{true, false} {
			if err := Muda(id, ligar); err == nil {
				t.Errorf("%q (ligar=%v) passou", id, ligar)
			}
		}
	}
	if c := m.mesa.Chamadas("systemctl"); len(c) != 0 {
		t.Errorf("chegou ao systemctl: %v", c)
	}
	if m.existe(m.usuario) {
		t.Error("criou ~/.config/autostart")
	}
	if m.le(filepath.Join(m.etc, "segredo.desktop")) != "[Desktop Entry]\nName=Fora\n" {
		t.Error("mexeu num arquivo de fora")
	}
}

func TestMuda_OQueEDoMkNaoMuda(t *testing.T) {
	m := novaMaquina(t)
	m.escreve(filepath.Join(m.home, ".config", "mk", "hypr", "mare", "autostart.lua"), `hl.exec_cmd("udiskie")`+"\n")
	err := Muda("mk:udiskie", false)
	if err == nil || !strings.Contains(err.Error(), "Mk") || acoes.EhRecusa(err) {
		t.Errorf("veio %v", err)
	}
}

func TestLista_ServicosDeUsuarioHabilitados(t *testing.T) {
	m := novaMaquina(t)
	m.comUnidades()
	itens, err := Lista()
	if err != nil {
		t.Fatal(err)
	}
	quer := []Item{
		{ID: "unidade:backup.timer", Origem: "serviço de usuário", Nome: "Cópia de segurança diária", Comando: "",
			Ligado: true, DoSistema: false, Mudavel: true},
		{ID: "unidade:pipewire.socket", Origem: "serviço de usuário", Nome: "PipeWire Multimedia System Sockets", Comando: "",
			Ligado: true, DoSistema: true, Mudavel: false}, // ligada para todos pelo sistema
		{ID: "unidade:syncthing.service", Origem: "serviço de usuário", Nome: "Syncthing", Comando: "/usr/bin/syncthing serve --no-browser",
			Ligado: true, DoSistema: false, Mudavel: true},
	}
	if !reflect.DeepEqual(itens, quer) {
		t.Errorf("veio %+v\nquer %+v", itens, quer)
	}
	c := m.mesa.Chamadas("systemctl")
	if c[0] != "--user list-unit-files --state=enabled --type=service,socket,timer --output=json --no-pager" {
		t.Errorf("chamada: %q", c[0])
	}
}

func TestMuda_ServicoDeUsuarioEEnableEDisable(t *testing.T) {
	m := novaMaquina(t)
	m.comUnidades()
	if err := Muda("unidade:syncthing.service", false); err != nil {
		t.Fatal(err)
	}
	if err := Muda("unidade:syncthing.service", true); err != nil {
		t.Fatal(err)
	}
	// Ligar o que ainda não está ligado por ninguém também vale.
	if err := Muda("unidade:novo.service", true); err != nil {
		t.Fatal(err)
	}
	quer := []string{"--user disable -- syncthing.service", "--user enable -- syncthing.service", "--user enable -- novo.service"}
	if c := m.mesa.Chamadas("systemctl"); !reflect.DeepEqual(c, quer) {
		t.Errorf("chamadas %v\nqueria %v", c, quer)
	}
	// O que o sistema liga para todos não se desliga daqui.
	err := Muda("unidade:pipewire.socket", false)
	if err == nil || !strings.Contains(err.Error(), "sistema") {
		t.Errorf("pipewire.socket: veio %v", err)
	}
	if c := m.mesa.Chamadas("systemctl"); len(c) != 3 {
		t.Errorf("chegou ao systemctl: %v", c)
	}
}

func TestMuda_ErroDoSystemctlVoltaNaFrase(t *testing.T) {
	m := novaMaquina(t)
	m.mesa.Comando("systemctl", dubles.Regra{Contem: "", Codigo: 1, Erro: "Failed to enable unit: Unit novo.service does not exist\n"})
	err := Muda("unidade:novo.service", true)
	if err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("veio %v", err)
	}
}

func TestLista_OQueOMkSobe(t *testing.T) {
	m := novaMaquina(t)
	b, err := os.ReadFile("testdata/autostart.lua")
	if err != nil {
		t.Fatal(err)
	}
	m.escreve(filepath.Join(m.home, ".config", "mk", "hypr", "mare", "autostart.lua"), string(b))
	itens, err := Lista()
	if err != nil {
		t.Fatal(err)
	}
	quer := []struct{ id, nome, comando string }{
		{"mk:mare", "Maré (a ilha e os painéis)", "mare-start"},
		{"mk:hypridle", "Hypridle (apaga a tela e suspende sozinho)", "hypridle -c '…'"},
		{"mk:wl-clip-persist", "Área de transferência (guarda o que foi copiado)", "wl-clip-persist --clipboard regular"},
		{"mk:cliphist-texto", "Histórico da área de transferência (texto)", "wl-paste --type text --watch cliphist store"},
		{"mk:cliphist-imagem", "Histórico da área de transferência (imagens)", "wl-paste --type image --watch cliphist store"},
		{"mk:mare-look", "Aparência dos apps (ícones, cursor e fontes)", "mare-look"},
		{"mk:udiskie", "Udiskie (monta pendrives e discos)", "udiskie --automount --no-tray"},
	}
	if len(itens) != len(quer) {
		t.Fatalf("vieram %d itens, queria %d: %+v", len(itens), len(quer), itens)
	}
	for i, q := range quer {
		veio := itens[i]
		if veio.ID != q.id || veio.Nome != q.nome || veio.Comando != q.comando {
			t.Errorf("item %d: veio %q %q %q\n        queria %q %q %q", i, veio.ID, veio.Nome, veio.Comando, q.id, q.nome, q.comando)
		}
		if veio.Origem != "Mk" || !veio.Ligado || !veio.DoSistema || veio.Mudavel {
			t.Errorf("item %d: %+v", i, veio)
		}
	}
}

func TestLista_LuaDesconhecidoGanhaNomePeloPrograma(t *testing.T) {
	m := novaMaquina(t)
	m.escreve(filepath.Join(m.home, ".config", "mk", "hypr", "mare", "autostart.lua"), strings.Join([]string{
		`-- hl.exec_cmd("comentado")`,
		`    hl.exec_cmd('/usr/bin/Coisa Nova --x') -- com comentário no fim`,
		`hl.exec_cmd("coisa-nova")`,
		`hl.exec_cmd("coisa-nova --de-novo")`,
		`hl.exec_cmd(variavel)`,
		`hl.exec_cmd("aspas \"dentro\" dela")`,
		`hl.exec_cmd("sem fechar`,
		`print("hl.exec_cmd não é chamada aqui")`,
	}, "\n"))
	itens, err := Lista()
	if err != nil {
		t.Fatal(err)
	}
	quer := [][3]string{
		{"mk:coisa", "Coisa", "/usr/bin/Coisa Nova --x"},
		{"mk:coisa-nova", "coisa-nova", "coisa-nova"},
		{"mk:coisa-nova-2", "coisa-nova", "coisa-nova --de-novo"},
		{"mk:aspas", "aspas", `aspas "dentro" dela`},
	}
	if len(itens) != len(quer) {
		t.Fatalf("vieram %+v", itens)
	}
	for i, q := range quer {
		if itens[i].ID != q[0] || itens[i].Nome != q[1] || itens[i].Comando != q[2] {
			t.Errorf("item %d: veio %+v, queria %v", i, itens[i], q)
		}
	}
}

func TestLista_OAgenteDeSenhasEOChaveiroDoMkTemNome(t *testing.T) {
	m := novaMaquina(t)
	m.escreve(filepath.Join(m.home, ".config", "mk", "hypr", "mare", "autostart.lua"),
		`hl.exec_cmd("systemctl --user start hyprpolkitagent")`+"\n"+`hl.exec_cmd("gnome-keyring-daemon --start --components=secrets,pkcs11")`+"\n")
	itens, _ := Lista()
	if len(itens) != 2 || itens[0].ID != "mk:polkit" || itens[0].Nome != "Agente de senhas (polkit)" ||
		itens[1].ID != "mk:chaveiro" || itens[1].Nome != "Chaveiro (senhas guardadas)" {
		t.Errorf("vieram %+v", itens)
	}
}

func TestLista_SemSystemctlAindaMostraOResto(t *testing.T) {
	m := novaMaquina(t)
	dubles.Nova(t)
	m.escreve(filepath.Join(m.sistema, "foo.desktop"), foo)
	itens, err := Lista()
	if err != nil || len(itens) != 1 {
		t.Fatalf("veio %+v %v", itens, err)
	}
}
