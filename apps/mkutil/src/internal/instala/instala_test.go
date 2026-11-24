package instala

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"mkutil/internal/base"
	"mkutil/internal/dubles"
	"mkutil/internal/instala/falso"
	sq "mkutil/internal/squash/falso"
)

// casa monta uma home falsa, sem biblioteca nenhuma e sem comando nenhum no
// PATH. Devolve a mesa dos dublês e a pasta de "downloads".
func casa(t *testing.T) (*dubles.Mesa, string) {
	t.Helper()
	raiz := t.TempDir()
	t.Setenv("MKUTIL_HOME", filepath.Join(raiz, "casa"))
	t.Setenv("MKUTIL_LIBS", filepath.Join(raiz, "libs"))
	t.Setenv("MKUTIL_LIMITE_MB", "")
	t.Setenv("MKUTIL_LIMITE_ENTRADAS", "")
	baixados := filepath.Join(raiz, "baixados")
	for _, p := range []string{filepath.Join(raiz, "casa"), filepath.Join(raiz, "libs"), baixados} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return dubles.Nova(t), baixados
}

func grava(t *testing.T, caminho string, conteudo []byte) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(caminho), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(caminho, conteudo, 0o644); err != nil {
		t.Fatal(err)
	}
	return caminho
}

func le(t *testing.T, caminho string) string {
	t.Helper()
	b, err := os.ReadFile(caminho)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func existe(caminho string) bool {
	_, err := os.Lstat(caminho)
	return err == nil
}

// tudo lista o que há debaixo de uma pasta, em ordem (pastas com "/").
func tudo(t *testing.T, pasta string) []string {
	t.Helper()
	out := []string{}
	_ = filepath.Walk(pasta, func(p string, fi os.FileInfo, err error) error {
		if err != nil || p == pasta {
			return nil
		}
		rel, _ := filepath.Rel(pasta, p)
		if fi.IsDir() {
			rel += "/"
		}
		out = append(out, rel)
		return nil
	})
	sort.Strings(out)
	return out
}

func instala(t *testing.T, arquivo string, op Opcoes) (*Fim, []string, error) {
	t.Helper()
	var fases []string
	fim, err := Instala(arquivo, op, func(v any) {
		b, _ := json.Marshal(v)
		fases = append(fases, string(b))
	})
	return fim, fases, err
}

func querRecusa(t *testing.T, err error, trecho string) {
	t.Helper()
	if err == nil {
		t.Fatalf("devia recusar (%s)", trecho)
	}
	if !base.FoiRecusa(err) {
		t.Errorf("devia ser recusa (código 2): %v", err)
	}
	if !strings.Contains(err.Error(), trecho) {
		t.Errorf("a frase %q não tem %q", err.Error(), trecho)
	}
}

func kritaPorDentro() []sq.Arquivo {
	return []sq.Arquivo{
		{Caminho: "AppRun", Conteudo: []byte("#!/bin/sh\n")},
		{Caminho: "org.kde.krita.desktop", Conteudo: []byte("[Desktop Entry]\nName=Krita\nComment=Pintura digital\nExec=krita %F\nIcon=krita\nCategories=Graphics;\nX-AppImage-Version=5.2.2\n")},
		{Caminho: ".DirIcon", Link: "krita.png"},
		{Caminho: "krita.png", Link: "usr/share/icons/hicolor/256x256/apps/krita.png"},
		{Caminho: "usr/share/icons/hicolor/256x256/apps/krita.png", Conteudo: falso.PNG},
	}
}

// ---- AppImage -----------------------------------------------------------

func TestAppImage_InspecionarDizOQueEOPlanoSemTocarNoDisco(t *testing.T) {
	_, baixados := casa(t)
	arq := grava(t, filepath.Join(baixados, "Krita-5.2.2-x86_64.AppImage"), falso.AppImage(false, kritaPorDentro()))
	f, err := Inspeciona(arq)
	if err != nil {
		t.Fatal(err)
	}
	if f.Tipo != "appimage" || f.ID != "krita" || f.Nome != "Krita" || *f.Versao != "5.2.2" || *f.Descricao != "Pintura digital" {
		t.Errorf("ficha: %+v", f)
	}
	if *f.Destino != base.Aplicativos("krita.AppImage") || *f.Atalho != base.Atalhos("mkinstalar-krita.desktop") {
		t.Errorf("destino %q, atalho %q", *f.Destino, *f.Atalho)
	}
	if ai := f.AppImage; ai == nil || ai.Tipo != 2 || !ai.TemDesktop || !ai.TemIcone || !ai.Fuse {
		t.Errorf("appimage: %+v", f.AppImage)
	}
	if f.JaInstalado || f.PrecisaSenha || len(f.Avisos) != 0 || f.Pacote != nil || f.Compactado != nil {
		t.Errorf("ficha: %+v", f)
	}
	if quer := []string{"copiar para ~/Aplicativos/krita.AppImage", "marcar como executável", "criar o atalho no menu"}; !reflect.DeepEqual(f.Plano, quer) {
		t.Errorf("plano: %q", f.Plano)
	}
	if existe(base.Aplicativos()) || existe(base.Atalhos()) || existe(base.Registro()) {
		t.Error("inspecionar não pode criar nada")
	}
}

func TestAppImage_InstalarCopiaDaExecucaoGravaOAtalhoEAnota(t *testing.T) {
	m, baixados := casa(t)
	m.Comando("update-desktop-database", "exit 0")
	conteudo := falso.AppImage(false, kritaPorDentro())
	arq := grava(t, filepath.Join(baixados, "Krita-5.2.2-x86_64.AppImage"), conteudo)
	fim, fases, err := instala(t, arq, Opcoes{})
	if err != nil {
		t.Fatal(err)
	}
	destino := base.Aplicativos("krita.AppImage")
	if !fim.Ok || fim.Fase != "fim" || fim.ID != "krita" || fim.Nome != "Krita" || *fim.Destino != destino {
		t.Errorf("fim: %+v", fim)
	}
	if le(t, destino) != string(conteudo) {
		t.Error("a cópia não é igual ao original")
	}
	if fi, _ := os.Stat(destino); fi.Mode().Perm() != 0o755 {
		t.Errorf("modo %v", fi.Mode())
	}
	if le(t, arq) != string(conteudo) {
		t.Error("o original foi tocado")
	}
	icone := base.Icones("krita.png")
	if le(t, icone) != string(falso.PNG) {
		t.Error("o ícone não foi tirado de dentro")
	}
	quer := "[Desktop Entry]\nType=Application\nName=Krita\nCategories=Graphics;\nComment=Pintura digital\n" +
		"Exec=\"" + destino + "\" %F\nTryExec=" + destino + "\nIcon=" + icone + "\nX-MkInstalar=krita\n"
	if got := le(t, base.Atalhos("mkinstalar-krita.desktop")); got != quer {
		t.Errorf("atalho:\n%s\nqueria:\n%s", got, quer)
	}
	itens, _ := LeRegistro()
	if len(itens) != 1 || itens[0].ID != "krita" || itens[0].Tipo != "appimage" || itens[0].Destino != destino ||
		*itens[0].Atalho != base.Atalhos("mkinstalar-krita.desktop") || *itens[0].Icone != icone ||
		itens[0].Origem != "Krita-5.2.2-x86_64.AppImage" || itens[0].Tamanho != int64(len(conteudo)) || itens[0].InstaladoEm == 0 {
		t.Errorf("registro: %+v", itens)
	}
	if c := m.Chamadas("update-desktop-database"); len(c) != 1 || !reflect.DeepEqual(c[0], []string{"-q", "--", base.Atalhos()}) {
		t.Errorf("update-desktop-database: %q", c)
	}
	if len(fases) < 2 || !strings.Contains(fases[0], `"fase":"copiando"`) || fases[len(fases)-1] != `{"fase":"atalho"}` ||
		!strings.Contains(fases[len(fases)-2], `"pct":100`) {
		t.Errorf("fases: %v", fases)
	}
	// Nada pela metade ficou em ~/Aplicativos.
	if quer := []string{".icones/", ".icones/krita.png", "krita.AppImage"}; !reflect.DeepEqual(tudo(t, base.Aplicativos()), quer) {
		t.Errorf("~/Aplicativos: %v", tudo(t, base.Aplicativos()))
	}
	// Agora a ficha diz que já está instalado.
	if f, _ := Inspeciona(arq); f == nil || !f.JaInstalado || !strings.Contains(f.Plano[0], "tirar a versão") {
		t.Errorf("depois de instalar: %+v", f)
	}
}

func TestAppImage_SemUpdateDesktopDatabaseInstalaDoMesmoJeito(t *testing.T) {
	_, baixados := casa(t)
	arq := grava(t, filepath.Join(baixados, "a.AppImage"), falso.AppImage(false, kritaPorDentro()))
	if _, _, err := instala(t, arq, Opcoes{}); err != nil {
		t.Fatal(err)
	}
}

func TestAppImage_SemAtalhoDeDentroSaiUmMinimoComONomeDoArquivo(t *testing.T) {
	_, baixados := casa(t)
	casos := map[string][]byte{
		"Meu App-1.0.AppImage":  append(falso.ELF(false, 2), bytes.Repeat([]byte("lixo"), 300)...),               // squashfs ilegível
		"Tipo Um-2.0.AppImage":  append(falso.ELF(false, 1), bytes.Repeat([]byte{0}, 300)...),                    // tipo 1
		"Sem Nada-3.0.AppImage": falso.AppImage(false, []sq.Arquivo{{Caminho: "AppRun", Conteudo: []byte("x")}}), // sem .desktop
	}
	for nome, conteudo := range casos {
		arq := grava(t, filepath.Join(baixados, nome), conteudo)
		f, err := Inspeciona(arq)
		if err != nil {
			t.Fatalf("%s: %v", nome, err)
		}
		if f.AppImage.TemDesktop || f.AppImage.TemIcone || len(f.Avisos) != 1 || !strings.Contains(f.Avisos[0], "só com o nome do arquivo") {
			t.Errorf("%s: %+v, avisos %q", nome, f.AppImage, f.Avisos)
		}
		fim, _, err := instala(t, arq, Opcoes{})
		if err != nil {
			t.Fatalf("%s: %v", nome, err)
		}
		_, nomeBonito, _ := Identifica(nome)
		destino := base.Aplicativos(fim.ID + ".AppImage")
		quer := "[Desktop Entry]\nType=Application\nName=" + nomeBonito + "\nExec=\"" + destino + "\"\nTryExec=" + destino +
			"\nIcon=application-x-executable\nX-MkInstalar=" + fim.ID + "\n"
		if got := le(t, *fim.Atalho); got != quer {
			t.Errorf("%s: atalho\n%s", nome, got)
		}
	}
}

func TestAppImage_SquashfsZstdSemOComandoZstdCaiNoAtalhoMinimo(t *testing.T) {
	m, baixados := casa(t)
	identidade := func(b []byte) []byte { return b }
	img := append(falso.ELF(false, 2), sq.Monta(kritaPorDentro(), sq.Opcoes{Compressor: 6, Comprime: identidade, Fragmentos: true})...)
	arq := grava(t, filepath.Join(baixados, "krita.AppImage"), img)
	f, err := Inspeciona(arq)
	if err != nil {
		t.Fatal(err)
	}
	if f.AppImage.TemDesktop || len(f.Avisos) != 1 || !strings.Contains(f.Avisos[0], "zstd") {
		t.Errorf("sem zstd: %+v %q", f.AppImage, f.Avisos)
	}
	// Com o zstd no PATH, lê.
	m.Comando("zstd", `exec /bin/cat`)
	if f, _ = Inspeciona(arq); f == nil || !f.AppImage.TemDesktop || !f.AppImage.TemIcone || f.Nome != "Krita" {
		t.Errorf("com zstd: %+v", f)
	}
}

func TestAppImage_AvisaQuandoFaltaOFuse2EDizOPacote(t *testing.T) {
	_, baixados := casa(t)
	dinamico := grava(t, filepath.Join(baixados, "din.AppImage"), falso.AppImage(true, kritaPorDentro()))
	estatico := grava(t, filepath.Join(baixados, "est.AppImage"), falso.AppImage(false, kritaPorDentro()))
	f, err := Inspeciona(dinamico)
	if err != nil {
		t.Fatal(err)
	}
	if f.AppImage.Fuse || len(f.Avisos) != 1 || !strings.Contains(f.Avisos[0], "libfuse.so.2") || !strings.Contains(f.Avisos[0], "pacote fuse2") {
		t.Errorf("dinâmico sem a lib: fuse=%v avisos=%q", f.AppImage.Fuse, f.Avisos)
	}
	// O runtime estático traz o FUSE dentro: sem aviso.
	if f, _ := Inspeciona(estatico); !f.AppImage.Fuse || len(f.Avisos) != 0 {
		t.Errorf("estático: %+v %q", f.AppImage, f.Avisos)
	}
	// Com a biblioteca na máquina, sem aviso.
	grava(t, filepath.Join(os.Getenv("MKUTIL_LIBS"), "libfuse.so.2.9.9"), []byte("x"))
	if f, _ := Inspeciona(dinamico); !f.AppImage.Fuse || len(f.Avisos) != 0 {
		t.Errorf("com a lib: %+v %q", f.AppImage, f.Avisos)
	}
	// O aviso vai também no fim da instalação.
	t.Setenv("MKUTIL_LIBS", filepath.Join(baixados, "vazio"))
	fim, _, err := instala(t, dinamico, Opcoes{})
	if err != nil || len(fim.Avisos) != 1 || !strings.Contains(fim.Avisos[0], "fuse2") {
		t.Errorf("fim: %+v, %v", fim, err)
	}
}

func TestAppImage_ValeAMagiaNaoONome(t *testing.T) {
	_, baixados := casa(t)
	falsoNome := grava(t, filepath.Join(baixados, "virus.AppImage"), []byte("#!/bin/sh\nrm -rf ~\n"))
	if _, err := Inspeciona(falsoNome); err == nil || !strings.Contains(err.Error(), "não é um") {
		t.Errorf("script com nome de AppImage: %v", err)
	}
	elfComum := grava(t, filepath.Join(baixados, "comum.AppImage"), append(falso.ELF(false, 0), make([]byte, 100)...))
	if _, err := Inspeciona(elfComum); err == nil {
		t.Error("ELF sem a magia não é AppImage")
	}
	semExtensao := grava(t, filepath.Join(baixados, "krita-5"), falso.AppImage(false, kritaPorDentro()))
	if f, err := Inspeciona(semExtensao); err != nil || f.Tipo != "appimage" || f.ID != "krita" {
		t.Errorf("AppImage sem extensão: %+v, %v", f, err)
	}
	for nome, trecho := range map[string]string{"a.deb": "Debian", "a.rpm": "Fedora", "a.flatpakref": "Flatpak", "a.txt": "não sei instalar", "a.exe": "não sei instalar"} {
		if _, err := Inspeciona(grava(t, filepath.Join(baixados, nome), []byte("xxxxxxxxxxxxxxxx"))); err == nil || !strings.Contains(err.Error(), trecho) {
			t.Errorf("%s: %v", nome, err)
		}
	}
	if _, err := Inspeciona(filepath.Join(baixados, "nao-existe.AppImage")); err == nil || !strings.Contains(err.Error(), "não achei") {
		t.Errorf("ausente: %v", err)
	}
	if _, err := Inspeciona(baixados); err == nil || !strings.Contains(err.Error(), "não é um arquivo") {
		t.Errorf("pasta: %v", err)
	}
	if existe(base.Aplicativos()) {
		t.Error("nada devia ter sido criado")
	}
}

func TestAppImage_JaInstaladoSoTrocaComSubstituirENuncaOQueNaoEDele(t *testing.T) {
	_, baixados := casa(t)
	v1 := grava(t, filepath.Join(baixados, "Krita-5.2.2.AppImage"), falso.AppImage(false, kritaPorDentro()))
	if _, _, err := instala(t, v1, Opcoes{}); err != nil {
		t.Fatal(err)
	}
	novo := append(falso.AppImage(false, kritaPorDentro()), []byte("versão nova")...)
	v2 := grava(t, filepath.Join(baixados, "Krita-5.3.0.AppImage"), novo)
	if _, _, err := instala(t, v2, Opcoes{}); err == nil || !strings.Contains(err.Error(), "já está instalado") {
		t.Errorf("sem --substituir: %v", err)
	}
	if strings.HasSuffix(le(t, base.Aplicativos("krita.AppImage")), "versão nova") {
		t.Error("trocou sem pedir")
	}
	if _, _, err := instala(t, v2, Opcoes{Substituir: true}); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(le(t, base.Aplicativos("krita.AppImage")), "versão nova") {
		t.Error("não trocou")
	}
	if itens, _ := LeRegistro(); len(itens) != 1 || *itens[0].Versao != "5.3.0" {
		t.Errorf("registro: %+v", itens)
	}

	// Um arquivo que a pessoa pôs lá à mão nunca é trocado, nem com --substituir.
	grava(t, base.Aplicativos("gimp.AppImage"), []byte("o da pessoa"))
	gimp := grava(t, filepath.Join(baixados, "gimp-3.0.AppImage"), falso.AppImage(false, kritaPorDentro()))
	f, _ := Inspeciona(gimp)
	if len(f.Avisos) != 1 || !strings.Contains(f.Avisos[0], "já existe ~/Aplicativos/gimp.AppImage") {
		t.Errorf("avisos: %q", f.Avisos)
	}
	for _, op := range []Opcoes{{}, {Substituir: true}} {
		if _, _, err := instala(t, gimp, op); err == nil || !strings.Contains(err.Error(), "já existe") {
			t.Errorf("%+v: %v", op, err)
		}
	}
	if le(t, base.Aplicativos("gimp.AppImage")) != "o da pessoa" {
		t.Error("trocou o arquivo da pessoa")
	}
}

func TestAppImage_SemAtalhoENomeHostil(t *testing.T) {
	_, baixados := casa(t)
	arq := grava(t, filepath.Join(baixados, "-rf $(touch x) 'a\" b`c`.AppImage"), falso.AppImage(false, kritaPorDentro()))
	fim, _, err := instala(t, arq, Opcoes{SemAtalho: true})
	if err != nil {
		t.Fatal(err)
	}
	if !IdValido(fim.ID) || fim.Atalho != nil || existe(base.Atalhos()) {
		t.Errorf("fim: %+v", fim)
	}
	if filepath.Dir(*fim.Destino) != base.Aplicativos() || !existe(*fim.Destino) {
		t.Errorf("destino: %q", *fim.Destino)
	}
	if itens, _ := LeRegistro(); len(itens) != 1 || itens[0].Atalho != nil {
		t.Errorf("registro: %+v", itens)
	}
}

func TestAppImage_HomeComEspacoEAspasViraExecCitado(t *testing.T) {
	_, baixados := casa(t)
	t.Setenv("MKUTIL_HOME", filepath.Join(baixados, `ana "maria" $HOME`))
	arq := grava(t, filepath.Join(baixados, "krita.AppImage"), falso.AppImage(false, kritaPorDentro()))
	fim, _, err := instala(t, arq, Opcoes{})
	if err != nil {
		t.Fatal(err)
	}
	if got := le(t, *fim.Atalho); !strings.Contains(got, `Exec="`+filepath.Join(baixados, `ana \\"maria\\" \\$HOME`)+`/Aplicativos/krita.AppImage" %F`+"\n") {
		t.Errorf("atalho:\n%s", got)
	}
}

// ---- pacote -------------------------------------------------------------

func TestPacote_InterpretaOPKGINFO(t *testing.T) {
	p, err := interpretaPKGINFO([]byte(falso.PKGINFO))
	if err != nil {
		t.Fatal(err)
	}
	if p.Nome != "htop" || p.Versao != "3.3.0-3" || *p.Descricao != "Interactive process viewer" || *p.Arquitetura != "x86_64" ||
		*p.TamanhoInstalado != 412345 || !reflect.DeepEqual(p.Dependencias, []string{"libcap", "ncurses", "libnl>=3"}) {
		t.Errorf("veio %+v", p)
	}
	for _, ruim := range []string{"", "pkgver = 1\n", "pkgname = htop\n", "pkgname = -rf\npkgver = 1\n", "pkgname = a b\npkgver = 1\n",
		"pkgname = ../x\npkgver = 1\n", "pkgname = a\npkgver = 1 2\n", "pkgname = A\npkgver = 1\n"} {
		if _, err := interpretaPKGINFO([]byte(ruim)); err == nil {
			t.Errorf("devia recusar %q", ruim)
		}
	}
}

func TestPacote_InspecionarLeComOBsdtarEPerguntaAoPacman(t *testing.T) {
	m, baixados := casa(t)
	arq := grava(t, filepath.Join(baixados, "-htop 3.3.0-3-x86_64.pkg.tar.zst"), []byte("não importa: o dublê responde"))
	m.Responde("bsdtar", falso.PKGINFO, 0)
	m.Responde("pacman", "htop 3.2.2-1\n", 0)
	f, err := Inspeciona(arq)
	if err != nil {
		t.Fatal(err)
	}
	if f.Tipo != "pacote" || f.ID != "htop" || f.Nome != "htop" || *f.Versao != "3.3.0-3" || !f.PrecisaSenha || f.Destino != nil || f.Atalho != nil {
		t.Errorf("ficha: %+v", f)
	}
	if f.Pacote == nil || *f.Pacote.Instalado != "3.2.2-1" || len(f.Pacote.Dependencias) != 3 {
		t.Errorf("pacote: %+v", f.Pacote)
	}
	if quer := []string{"trocar htop 3.2.2-1 por 3.3.0-3 com o pacman (pede a senha de administrador)", "o pacman baixa o que faltar das 3 dependências"}; !reflect.DeepEqual(f.Plano, quer) {
		t.Errorf("plano: %q", f.Plano)
	}
	if len(f.Avisos) != 1 || !strings.Contains(f.Avisos[0], "administrador") {
		t.Errorf("avisos: %q", f.Avisos)
	}
	// O nome do arquivo (com espaço, começando com "-") vai inteiro, absoluto, num argumento só.
	if c := m.Chamadas("bsdtar"); len(c) != 1 || !reflect.DeepEqual(c[0], []string{"-xOqf", arq, ".PKGINFO"}) {
		t.Errorf("bsdtar: %q", c)
	}
	if c := m.Chamadas("pacman"); len(c) != 1 || !reflect.DeepEqual(c[0], []string{"-Q", "--", "htop"}) {
		t.Errorf("pacman: %q", c)
	}
	// Não instalado: o pacman -Q falha.
	m.Comando("pacman", "exit 1")
	if f, _ := Inspeciona(arq); f.Pacote.Instalado != nil || !strings.HasPrefix(f.Plano[0], "instalar htop 3.3.0-3 com o pacman") {
		t.Errorf("não instalado: %+v %q", f.Pacote, f.Plano)
	}
	// Já na mesma versão.
	m.Responde("pacman", "htop 3.3.0-3\n", 0)
	if f, _ := Inspeciona(arq); !strings.Contains(f.Plano[0], "já está instalado") {
		t.Errorf("mesma versão: %q", f.Plano)
	}
}

func TestPacote_SemOBsdtarLeComOZstdOuOXz(t *testing.T) {
	m, baixados := casa(t)
	// O dublê do zstd devolve o que recebe: o "pacote" do teste é um tar puro.
	pacote := falso.Tar([]falso.Entrada{{Nome: ".BUILDINFO", Conteudo: "x"}, {Nome: ".PKGINFO", Conteudo: falso.PKGINFO}, {Nome: "usr/bin/htop", Conteudo: "elf"}})
	zst := grava(t, filepath.Join(baixados, "htop.pkg.tar.zst"), pacote)
	xz := grava(t, filepath.Join(baixados, "htop.pkg.tar.xz"), pacote)
	if _, err := Inspeciona(zst); err == nil || !strings.Contains(err.Error(), "é preciso o zstd") {
		t.Errorf("sem bsdtar nem zstd: %v", err)
	}
	m.Comando("zstd", `[ "$#" = 1 ] && [ "$1" = -dc ] || exit 9; exec /bin/cat`)
	m.Comando("xz", `[ "$#" = 1 ] && [ "$1" = -dc ] || exit 9; exec /bin/cat`)
	for _, arq := range []string{zst, xz} {
		f, err := Inspeciona(arq)
		if err != nil || f.Pacote.Nome != "htop" || f.Pacote.Instalado != nil {
			t.Errorf("%s: %+v, %v", filepath.Base(arq), f, err)
		}
	}
	semInfo := grava(t, filepath.Join(baixados, "x.pkg.tar.zst"), falso.Tar([]falso.Entrada{{Nome: "usr/bin/x", Conteudo: "x"}}))
	if _, err := Inspeciona(semInfo); err == nil || !strings.Contains(err.Error(), "não tem .PKGINFO") {
		t.Errorf("sem .PKGINFO: %v", err)
	}
	m.Responde("bsdtar", "", 0)
	if _, err := Inspeciona(semInfo); err == nil || !strings.Contains(err.Error(), "não tem .PKGINFO") {
		t.Errorf("bsdtar sem .PKGINFO: %v", err)
	}
	m.Comando("bsdtar", "printf 'bsdtar: Error opening archive: Unrecognized archive format\\n' >&2; exit 1")
	if _, err := Inspeciona(semInfo); err == nil || !strings.Contains(err.Error(), "Unrecognized archive format") {
		t.Errorf("bsdtar com erro: %v", err)
	}
}

func TestPacote_InstalarVaiPeloPkexecComOPacmanPorExtenso(t *testing.T) {
	m, baixados := casa(t)
	arq := grava(t, filepath.Join(baixados, "-htop 3.3.0-3-x86_64.pkg.tar.zst"), []byte("x"))
	m.Responde("bsdtar", falso.PKGINFO, 0)
	m.Comando("pacman", "exit 1")
	m.Comando("sudo", "exit 0")
	m.Comando("su", "exit 0")
	m.Comando("pkexec", `printf 'carregando pacotes...\n\n(1/1) instalando htop\n'; printf 'aviso no stderr\n' >&2`)
	fim, fases, err := instala(t, arq, Opcoes{})
	if err != nil {
		t.Fatal(err)
	}
	if !fim.Ok || fim.Tipo != "pacote" || fim.ID != "htop" || fim.Destino != nil || fim.Atalho != nil {
		t.Errorf("fim: %+v", fim)
	}
	if c := m.Chamadas("pkexec"); len(c) != 1 || !reflect.DeepEqual(c[0], []string{"/usr/bin/pacman", "-U", "--needed", "--noconfirm", "--", arq}) {
		t.Errorf("pkexec: %q", c)
	}
	if len(m.Chamadas("sudo")) != 0 || len(m.Chamadas("su")) != 0 {
		t.Error("o motor nunca chama o sudo")
	}
	junto := strings.Join(fases, "\n")
	for _, linha := range []string{`{"fase":"pacman","linha":"carregando pacotes..."}`, `{"fase":"pacman","linha":"(1/1) instalando htop"}`, `{"fase":"pacman","linha":"aviso no stderr"}`} {
		if !strings.Contains(junto, linha) {
			t.Errorf("faltou %s em %v", linha, fases)
		}
	}
	// Pacote é do pacman: não entra no registro nem em ~/Aplicativos.
	if existe(base.Registro()) || existe(base.Aplicativos()) || existe(base.Atalhos()) {
		t.Error("um pacote não mexe nas pastas do MkInstalar")
	}
}

func TestPacote_SenhaRecusadaErroDoPacmanEFaltaDoPkexec(t *testing.T) {
	m, baixados := casa(t)
	arq := grava(t, filepath.Join(baixados, "htop.pkg.tar.zst"), []byte("x"))
	m.Responde("bsdtar", falso.PKGINFO, 0)
	if _, _, err := instala(t, arq, Opcoes{}); err == nil || !strings.Contains(err.Error(), "não achei o pkexec") {
		t.Errorf("sem pkexec: %v", err)
	}
	for _, codigo := range []string{"126", "127"} {
		m.Comando("pkexec", "printf 'Error executing command as another user: Request dismissed\\n' >&2; exit "+codigo)
		if _, _, err := instala(t, arq, Opcoes{}); err == nil || err.Error() != "a senha não foi dada: nada foi instalado" {
			t.Errorf("código %s: %v", codigo, err)
		}
	}
	m.Comando("pkexec", "printf 'carregando...\\nerro: htop: assinatura inválida\\n'; exit 1")
	if _, _, err := instala(t, arq, Opcoes{}); err == nil || err.Error() != "o pacman não instalou: erro: htop: assinatura inválida" {
		t.Errorf("erro do pacman: %v", err)
	}
	m.Comando("pkexec", "exit 3")
	if _, _, err := instala(t, arq, Opcoes{}); err == nil || !strings.Contains(err.Error(), "código 3") {
		t.Errorf("pacman mudo: %v", err)
	}
	// Um arquivo que não é pacote nem chega ao pkexec.
	m.Comando("pkexec", "exit 0")
	m.Esquece("pkexec")
	m.Responde("bsdtar", "pkgname = $(rm -rf ~)\npkgver = 1\n", 0)
	if _, _, err := instala(t, arq, Opcoes{}); err == nil || !strings.Contains(err.Error(), "não é um pacote do Arch") {
		t.Errorf("PKGINFO torto: %v", err)
	}
	if len(m.Chamadas("pkexec")) != 0 {
		t.Error("chegou ao pkexec com um pacote inválido")
	}
}

// ---- registro e desinstalar --------------------------------------------

func TestDesinstalar_ApagaOQueInstalouESoIsso(t *testing.T) {
	m, baixados := casa(t)
	m.Comando("update-desktop-database", "exit 0")
	arq := grava(t, filepath.Join(baixados, "krita.AppImage"), falso.AppImage(false, kritaPorDentro()))
	outro := grava(t, filepath.Join(baixados, "gimp.AppImage"), falso.AppImage(false, kritaPorDentro()))
	for _, a := range []string{arq, outro} {
		if _, _, err := instala(t, a, Opcoes{}); err != nil {
			t.Fatal(err)
		}
	}
	grava(t, base.Aplicativos("da-pessoa.txt"), []byte("x"))
	grava(t, base.Atalhos("firefox.desktop"), []byte("x"))
	removidos, err := Desinstala("krita")
	if err != nil {
		t.Fatal(err)
	}
	quer := []string{base.Aplicativos("krita.AppImage"), base.Atalhos("mkinstalar-krita.desktop"), base.Icones("krita.png")}
	if !reflect.DeepEqual(removidos, quer) {
		t.Errorf("removidos: %q", removidos)
	}
	if quer := []string{".icones/", ".icones/gimp.png", "da-pessoa.txt", "gimp.AppImage"}; !reflect.DeepEqual(tudo(t, base.Aplicativos()), quer) {
		t.Errorf("~/Aplicativos: %v", tudo(t, base.Aplicativos()))
	}
	if quer := []string{"firefox.desktop", "mkinstalar-gimp.desktop"}; !reflect.DeepEqual(tudo(t, base.Atalhos()), quer) {
		t.Errorf("atalhos: %v", tudo(t, base.Atalhos()))
	}
	if itens, _ := LeRegistro(); len(itens) != 1 || itens[0].ID != "gimp" {
		t.Errorf("registro: %+v", itens)
	}
	if len(m.Chamadas("update-desktop-database")) != 3 {
		t.Errorf("update-desktop-database: %q", m.Chamadas("update-desktop-database"))
	}
	if _, err := Desinstala("krita"); err == nil || !strings.Contains(err.Error(), "não foi instalado pelo MkInstalar") {
		t.Errorf("de novo: %v", err)
	}
	for _, id := range []string{"", "../gimp", "-rf", "a/b", "GIMP"} {
		if _, err := Desinstala(id); err == nil {
			t.Errorf("id %q devia falhar", id)
		}
	}
	// O destino sumiu (a pessoa apagou à mão): desinstalar ainda limpa o atalho.
	os.Remove(base.Aplicativos("gimp.AppImage"))
	removidos, err = Desinstala("gimp")
	if err != nil || len(removidos) != 2 || existe(base.Atalhos("mkinstalar-gimp.desktop")) {
		t.Errorf("destino sumido: %q, %v", removidos, err)
	}
}

func TestDesinstalar_RegistroAdulteradoNaoMandaApagarForaDasPastasDoMkInstalar(t *testing.T) {
	_, _ = casa(t)
	s := func(v string) *string { return &v }
	documentos := grava(t, base.Home("Documentos", "tese.odt"), []byte("anos de trabalho"))
	grava(t, base.Aplicativos("bom"), []byte("x"))
	grava(t, base.Aplicativos(".icones", "bom.png"), []byte("x"))
	grava(t, base.Atalhos("firefox.desktop"), []byte("x"))
	grava(t, base.Atalhos("mkinstalar-bom.desktop"), []byte("x"))
	grava(t, base.Atalhos("sub", "mkinstalar-x.desktop"), []byte("x"))
	bom := Item{ID: "bom", Tipo: "appimage", Destino: base.Aplicativos("bom"), Atalho: s(base.Atalhos("mkinstalar-bom.desktop")), Icone: s(base.Icones("bom.png"))}
	casos := map[string]func(i *Item){
		"destino em Documentos":          func(i *Item) { i.Destino = documentos },
		"destino é a home":               func(i *Item) { i.Destino = base.Home() },
		"destino é ~/Aplicativos":        func(i *Item) { i.Destino = base.Aplicativos() },
		"destino com ..":                 func(i *Item) { i.Destino = base.Aplicativos() + "/../Aplicativos/bom" },
		"destino com barra no fim":       func(i *Item) { i.Destino = base.Aplicativos("bom") + "/" },
		"destino escondido":              func(i *Item) { i.Destino = base.Aplicativos(".icones") },
		"destino numa subpasta":          func(i *Item) { i.Destino = base.Aplicativos("bom", "x") },
		"destino relativo":               func(i *Item) { i.Destino = "Aplicativos/bom" },
		"destino vazio":                  func(i *Item) { i.Destino = "" },
		"destino na raiz":                func(i *Item) { i.Destino = "/" },
		"atalho de outro app":            func(i *Item) { i.Atalho = s(base.Atalhos("firefox.desktop")) },
		"atalho fora da pasta":           func(i *Item) { i.Atalho = s(documentos) },
		"atalho numa subpasta":           func(i *Item) { i.Atalho = s(base.Atalhos("sub", "mkinstalar-x.desktop")) },
		"atalho com ..":                  func(i *Item) { i.Atalho = s(base.Atalhos() + "/../applications/mkinstalar-bom.desktop") },
		"atalho sem .desktop":            func(i *Item) { i.Atalho = s(base.Atalhos("mkinstalar-bom")) },
		"ícone fora de .icones":          func(i *Item) { i.Icone = s(documentos) },
		"ícone é a pasta de aplicativos": func(i *Item) { i.Icone = s(base.Aplicativos("bom")) },
		"id que não é id":                func(i *Item) { i.ID = "../bom" },
	}
	_ = gravaRegistro([]Item{bom})
	antes := tudo(t, base.Home())
	for nome, estraga := range casos {
		item := bom
		estraga(&item)
		if item.ID == "../bom" {
			// O id inválido não passa nem pela busca: confere direto.
			querRecusa(t, confere(item), "não mexo")
			continue
		}
		if err := gravaRegistro([]Item{item}); err != nil {
			t.Fatal(err)
		}
		_, err := Desinstala("bom")
		if err == nil || !base.FoiRecusa(err) {
			t.Errorf("%s: devia recusar, veio %v", nome, err)
		}
		if depois := tudo(t, base.Home()); !reflect.DeepEqual(antes, depois) {
			t.Fatalf("%s: apagou algo:\n antes  %v\n depois %v", nome, antes, depois)
		}
		if itens, _ := LeRegistro(); len(itens) != 1 {
			t.Errorf("%s: o registro mudou", nome)
		}
	}
	// O item bom, sem adulteração, sai.
	_ = gravaRegistro([]Item{bom})
	if _, err := Desinstala("bom"); err != nil {
		t.Fatal(err)
	}
	if le(t, documentos) != "anos de trabalho" || !existe(base.Atalhos("firefox.desktop")) || existe(base.Aplicativos("bom")) {
		t.Error("o que não era dele tem de ficar; o que era, sair")
	}
}

func TestDesinstalar_LinkSimbolicoEApagadoComoLinkNuncaSeguido(t *testing.T) {
	_, _ = casa(t)
	documentos := grava(t, base.Home("Documentos", "tese.odt"), []byte("anos de trabalho"))
	if err := os.MkdirAll(base.Aplicativos(), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Dir(documentos), base.Aplicativos("esperto")); err != nil {
		t.Fatal(err)
	}
	_ = gravaRegistro([]Item{{ID: "esperto", Tipo: "compactado", Destino: base.Aplicativos("esperto")}})
	if _, err := Desinstala("esperto"); err != nil {
		t.Fatal(err)
	}
	if existe(base.Aplicativos("esperto")) || le(t, documentos) != "anos de trabalho" {
		t.Error("o link sai; o que ele apontava fica")
	}
}

func TestRegistro_IlegivelNaoEApagadoNemIgnorado(t *testing.T) {
	_, baixados := casa(t)
	grava(t, base.Registro(), []byte("{ isto não é json"))
	if _, err := LeRegistro(); err == nil || !strings.Contains(err.Error(), "ilegível") {
		t.Errorf("veio %v", err)
	}
	arq := grava(t, filepath.Join(baixados, "krita.AppImage"), falso.AppImage(false, kritaPorDentro()))
	f, err := Inspeciona(arq)
	if err != nil || len(f.Avisos) != 1 || !strings.Contains(f.Avisos[0], "ilegível") {
		t.Errorf("inspecionar avisa: %+v, %v", f, err)
	}
	if _, _, err := instala(t, arq, Opcoes{}); err == nil || !strings.Contains(err.Error(), "ilegível") {
		t.Errorf("instalar: %v", err)
	}
	if existe(base.Aplicativos()) || le(t, base.Registro()) != "{ isto não é json" {
		t.Error("não devia instalar nem mexer no registro")
	}
	if _, err := Desinstala("krita"); err == nil {
		t.Error("desinstalar devia falhar")
	}
	os.Remove(base.Registro())
	if itens, err := LeRegistro(); err != nil || len(itens) != 0 || itens == nil {
		t.Errorf("sem arquivo: %v, %v", itens, err)
	}
}
