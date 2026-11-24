package instala

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"mkutil/internal/base"
	"mkutil/internal/instala/falso"
)

const binario = "\x7fELF\x02\x01\x01 um binário de mentira"

func TestNomeSeguro(t *testing.T) {
	bons := map[string]string{
		"a": "a", "a/b/c": "a/b/c", "./a/b": "a/b", "a//b/": "a/b", "./": "", ".": "", "a/./b": "a/b",
		"ação/é assim.txt": "ação/é assim.txt", "..a/b..": "..a/b..", "a/...": "a/...", "-rf": "-rf",
	}
	for nome, quer := range bons {
		if got, err := nomeSeguro(nome); err != nil || got != quer {
			t.Errorf("nomeSeguro(%q) = %q, %v; queria %q", nome, got, err, quer)
		}
	}
	ruins := []string{"", "/etc/passwd", "../a", "a/../../b", "a/..", "..", "a/b/../c", "a\\..\\b", "a\x00b", "//a",
		strings.Repeat("a", 256), strings.Repeat("a/", 2100)}
	for _, nome := range ruins {
		if _, err := nomeSeguro(nome); err == nil || !base.FoiRecusa(err) {
			t.Errorf("nomeSeguro(%q) devia recusar: %v", nome[:min(len(nome), 30)], err)
		}
	}
}

func TestAlvoSeguro(t *testing.T) {
	bons := [][2]string{{"a", "b"}, {"a/b", "../c"}, {"a/b/c", "../../d"}, {"lib/x.so", "x.so.1"}, {"a/b", "./c/d"}, {"a/b", ".."}, {"a", "."}}
	for _, c := range bons {
		if err := alvoSeguro(c[0], c[1]); err != nil {
			t.Errorf("alvoSeguro(%q, %q): %v", c[0], c[1], err)
		}
	}
	ruins := [][2]string{{"a", ".."}, {"a", "../b"}, {"a/b", "../../c"}, {"a", "/etc/passwd"}, {"a", ""}, {"a/b", "c/../../../d"},
		{"a", "b\x00"}, {"a", "/"}, {"a", strings.Repeat("b", 5000)}}
	for _, c := range ruins {
		if err := alvoSeguro(c[0], c[1]); err == nil || !base.FoiRecusa(err) {
			t.Errorf("alvoSeguro(%q, %q) devia recusar", c[0], c[1][:min(len(c[1]), 20)])
		}
	}
}

func firefox() []falso.Entrada {
	return []falso.Entrada{
		{Nome: "firefox/", Tipo: 'd'},
		{Nome: "firefox/firefox", Conteudo: binario, Modo: 0o755},
		{Nome: "firefox/firefox-bin", Conteudo: binario, Modo: 0o4755}, // setuid: não pode passar
		{Nome: "firefox/updater.sh", Conteudo: "#!/bin/sh\n", Modo: 0o755},
		{Nome: "firefox/libxul.so", Conteudo: binario, Modo: 0o755},
		{Nome: "firefox/libnss.so.3", Conteudo: binario, Modo: 0o755},
		{Nome: "firefox/leia.txt", Conteudo: "texto com x", Modo: 0o755},
		{Nome: "firefox/dados.bin", Conteudo: binario, Modo: 0o644},
		{Nome: "firefox/browser/", Tipo: 'd', Modo: 0o500},
		{Nome: "firefox/browser/chrome/icons/default/default128.png", Conteudo: "png"},
		{Nome: "firefox/browser/firefox.png", Conteudo: "png do app"},
		{Nome: "firefox/firefox.desktop", Conteudo: "[Desktop Entry]\nName=Firefox\nExec=/bin/rm -rf ~\nIcon=firefox\nCategories=Network;\n"},
		{Nome: "firefox/atalho", Tipo: 'l', Alvo: "browser/firefox.png"},
		{Nome: "firefox/browser/sobe", Tipo: 'l', Alvo: "../firefox"},
		{Nome: "firefox/solto", Tipo: 'l', Alvo: "nao-existe"},
		{Nome: "firefox/copia", Tipo: 'h', Alvo: "firefox/leia.txt"},
	}
}

func TestCompactado_InspecionarAchaAPastaUnicaOExecutavelEOAtalho(t *testing.T) {
	_, baixados := casa(t)
	arq := grava(t, filepath.Join(baixados, "firefox-128.0.tar.gz"), falso.TarGz(firefox()))
	f, err := Inspeciona(arq)
	if err != nil {
		t.Fatal(err)
	}
	c := f.Compactado
	if f.Tipo != "compactado" || f.ID != "firefox" || *f.Versao != "128.0" || *f.Destino != base.Aplicativos("firefox") || c == nil {
		t.Fatalf("ficha: %+v", f)
	}
	if c.Formato != "tar.gz" || c.Entradas != 16 || *c.PastaUnica != "firefox" || *c.ExecutavelSugerido != "firefox" {
		t.Errorf("compactado: %+v", c)
	}
	// Só o que tem o bit de execução e é programa (ELF ou script); biblioteca não.
	if quer := []string{"firefox", "firefox-bin", "updater.sh"}; !reflect.DeepEqual(c.Executaveis, quer) {
		t.Errorf("executáveis: %q", c.Executaveis)
	}
	if !reflect.DeepEqual(c.Atalhos, []string{"firefox.desktop"}) {
		t.Errorf("atalhos: %q", c.Atalhos)
	}
	if quer := []string{"extrair 16 itens para ~/Aplicativos/firefox", "criar o atalho no menu para firefox"}; !reflect.DeepEqual(f.Plano, quer) {
		t.Errorf("plano: %q", f.Plano)
	}
	if existe(base.Aplicativos()) {
		t.Error("inspecionar não extrai")
	}
}

func TestCompactado_InstalarExtraiSemAPastaDoTopoCriaOAtalhoEDesinstalaTudo(t *testing.T) {
	_, baixados := casa(t)
	arq := grava(t, filepath.Join(baixados, "firefox-128.0.tar.gz"), falso.TarGz(firefox()))
	fim, fases, err := instala(t, arq, Opcoes{})
	if err != nil {
		t.Fatal(err)
	}
	destino := base.Aplicativos("firefox")
	quer := []string{"atalho", "browser/", "browser/chrome/", "browser/chrome/icons/", "browser/chrome/icons/default/",
		"browser/chrome/icons/default/default128.png", "browser/firefox.png", "browser/sobe", "copia", "dados.bin", "firefox",
		"firefox-bin", "firefox.desktop", "leia.txt", "libnss.so.3", "libxul.so", "solto", "updater.sh"}
	if got := tudo(t, destino); !reflect.DeepEqual(got, quer) {
		t.Errorf("extraído:\n %v\nqueria:\n %v", got, quer)
	}
	modo := func(rel string) os.FileMode {
		fi, err := os.Lstat(filepath.Join(destino, rel))
		if err != nil {
			t.Fatal(err)
		}
		return fi.Mode()
	}
	if modo("firefox") != 0o755 || modo("dados.bin") != 0o644 || modo("firefox-bin") != 0o755 {
		t.Errorf("modos: %v %v %v", modo("firefox"), modo("dados.bin"), modo("firefox-bin"))
	}
	if modo("firefox-bin")&os.ModeSetuid != 0 {
		t.Error("o setuid passou")
	}
	if modo(".").Perm() != 0o755 || modo("browser").Perm() != 0o755 {
		t.Errorf("pastas: %v %v", modo("."), modo("browser"))
	}
	if alvo, _ := os.Readlink(filepath.Join(destino, "atalho")); alvo != "browser/firefox.png" || modo("atalho")&os.ModeSymlink == 0 {
		t.Errorf("link: %q", alvo)
	}
	if le(t, filepath.Join(destino, "copia")) != "texto com x" || le(t, filepath.Join(destino, "browser", "sobe")) != binario {
		t.Error("link físico ou link que sobe dentro da pasta")
	}
	// O atalho: o Exec é o caminho novo, nunca o que o .desktop de dentro dizia.
	atalho := le(t, base.Atalhos("mkinstalar-firefox.desktop"))
	querAtalho := "[Desktop Entry]\nType=Application\nName=Firefox\nCategories=Network;\nExec=\"" + destino + "/firefox\" -rf ~\nTryExec=" +
		destino + "/firefox\nIcon=" + destino + "/browser/firefox.png\nX-MkInstalar=firefox\n"
	if atalho != querAtalho {
		t.Errorf("atalho:\n%s\nqueria:\n%s", atalho, querAtalho)
	}
	if strings.Contains(atalho, "/bin/rm") {
		t.Error("o programa do Exec de dentro passou")
	}
	if !fim.Ok || fim.Tipo != "compactado" || fim.Nome != "Firefox" || *fim.Atalho != base.Atalhos("mkinstalar-firefox.desktop") {
		t.Errorf("fim: %+v", fim)
	}
	if len(fases) != 2 || !strings.Contains(fases[0], `"fase":"extraindo"`) || fases[1] != `{"fase":"atalho"}` {
		t.Errorf("fases: %v", fases)
	}
	if got := tudo(t, base.Aplicativos()); got[0] != "firefox/" {
		t.Errorf("sobrou algo pela metade em ~/Aplicativos: %v", got[:3])
	}
	itens, _ := LeRegistro()
	if len(itens) != 1 || itens[0].Tipo != "compactado" || itens[0].Destino != destino || itens[0].Icone != nil || itens[0].Tamanho == 0 {
		t.Errorf("registro: %+v", itens)
	}
	// E sai inteiro, mesmo com uma subpasta que veio sem permissão de escrita.
	_ = os.Chmod(filepath.Join(destino, "browser"), 0o500)
	if _, err := Desinstala("firefox"); err != nil {
		t.Fatal(err)
	}
	if existe(destino) || existe(base.Atalhos("mkinstalar-firefox.desktop")) {
		t.Error("desinstalar deixou resto")
	}
}

func TestCompactado_SemPastaUnicaVaiTudoParaAPastaDoApp(t *testing.T) {
	_, baixados := casa(t)
	arq := grava(t, filepath.Join(baixados, "Ferramenta_v2.1-linux.zip"), falso.Zip([]falso.Entrada{
		{Nome: "ferramenta", Conteudo: binario, Modo: 0o755},
		{Nome: "docs/leia.txt", Conteudo: "oi"},
		{Nome: "outra/coisa", Conteudo: "#!/bin/sh\n", Modo: 0o755},
	}, false))
	f, err := Inspeciona(arq)
	if err != nil {
		t.Fatal(err)
	}
	if f.ID != "ferramenta" || f.Compactado.PastaUnica != nil || *f.Compactado.ExecutavelSugerido != "ferramenta" || f.Compactado.Formato != "zip" {
		t.Errorf("ficha: %+v %+v", f, f.Compactado)
	}
	if _, _, err := instala(t, arq, Opcoes{}); err != nil {
		t.Fatal(err)
	}
	if quer := []string{"docs/", "docs/leia.txt", "ferramenta", "outra/", "outra/coisa"}; !reflect.DeepEqual(tudo(t, base.Aplicativos("ferramenta")), quer) {
		t.Errorf("extraído: %v", tudo(t, base.Aplicativos("ferramenta")))
	}
	// Um arquivo só no topo não é "pasta única".
	so := grava(t, filepath.Join(baixados, "solo.tar.gz"), falso.TarGz([]falso.Entrada{{Nome: "solo", Conteudo: binario, Modo: 0o755}}))
	if f, err := Inspeciona(so); err != nil || f.Compactado.PastaUnica != nil || *f.Compactado.ExecutavelSugerido != "solo" {
		t.Errorf("arquivo só: %+v, %v", f, err)
	}
}

func TestCompactado_ZipFeitoForaDoUnixAchaOProgramaPeloConteudoEDaExecucao(t *testing.T) {
	_, baixados := casa(t)
	arq := grava(t, filepath.Join(baixados, "jogo.zip"), falso.Zip([]falso.Entrada{
		{Nome: "jogo/jogo.x86_64", Conteudo: binario},
		{Nome: "jogo/dados.pck", Conteudo: "dados e mais dados"},
		{Nome: "jogo/lib/libsteam.so", Conteudo: binario},
	}, true))
	f, err := Inspeciona(arq)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(f.Compactado.Executaveis, []string{"jogo.x86_64"}) {
		t.Errorf("executáveis: %q", f.Compactado.Executaveis)
	}
	fim, _, err := instala(t, arq, Opcoes{})
	if err != nil {
		t.Fatal(err)
	}
	if fi, _ := os.Stat(base.Aplicativos("jogo", "jogo.x86_64")); fi.Mode().Perm()&0o100 == 0 {
		t.Errorf("o executável escolhido ficou sem +x: %v", fi.Mode())
	}
	if fi, _ := os.Stat(base.Aplicativos("jogo", "dados.pck")); fi.Mode().Perm() != 0o644 {
		t.Errorf("dados: %v", fi.Mode())
	}
	if !strings.Contains(le(t, *fim.Atalho), "Icon=application-x-executable\n") {
		t.Error("sem ícone na pasta: o genérico")
	}
}

func TestCompactado_SemExecutavelExtraiSemAtalhoEAvisa(t *testing.T) {
	_, baixados := casa(t)
	arq := grava(t, filepath.Join(baixados, "fontes.tar.gz"), falso.TarGz([]falso.Entrada{{Nome: "fontes/a.ttf", Conteudo: "ttf!"}, {Nome: "fontes/b.ttf", Conteudo: "ttf!"}}))
	f, err := Inspeciona(arq)
	if err != nil || f.Atalho != nil || f.Compactado.ExecutavelSugerido != nil || len(f.Avisos) != 1 || !strings.Contains(f.Avisos[0], "sem atalho") {
		t.Fatalf("ficha: %+v, %v", f, err)
	}
	fim, _, err := instala(t, arq, Opcoes{})
	if err != nil || fim.Atalho != nil || existe(base.Atalhos()) || !existe(base.Aplicativos("fontes", "a.ttf")) {
		t.Errorf("fim: %+v, %v", fim, err)
	}
	if len(fim.Avisos) != 1 {
		t.Errorf("avisos: %q", fim.Avisos)
	}
}

func TestCompactado_EscolherOExecutavelSoDeDentro(t *testing.T) {
	_, baixados := casa(t)
	arq := grava(t, filepath.Join(baixados, "firefox.tar.gz"), falso.TarGz(firefox()))
	fora := grava(t, base.Home("fora.sh"), []byte("#!/bin/sh\n"))
	for _, ruim := range []string{"../../fora.sh", fora, "nao-existe", "browser", "a\\b", "solto"} {
		_, _, err := instala(t, arq, Opcoes{Executavel: ruim})
		if err == nil {
			t.Fatalf("%q devia falhar", ruim)
		}
		if existe(base.Aplicativos("firefox")) || existe(base.Atalhos()) || len(tudo(t, base.Aplicativos())) != 0 {
			t.Fatalf("%q: sobrou algo: %v", ruim, tudo(t, base.Aplicativos()))
		}
	}
	if fi, _ := os.Stat(fora); fi.Mode().Perm() != 0o644 {
		t.Error("mexeu no arquivo de fora")
	}
	fim, _, err := instala(t, arq, Opcoes{Executavel: "./updater.sh"})
	if err != nil || !strings.Contains(le(t, *fim.Atalho), "Exec=\""+base.Aplicativos("firefox", "updater.sh")+"\"") {
		t.Errorf("escolhido: %v", err)
	}
	// Um link para um arquivo de dentro vale: o atalho aponta para o link.
	if _, err := Desinstala("firefox"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := instala(t, arq, Opcoes{Executavel: "browser/sobe"}); err != nil {
		t.Errorf("link de dentro: %v", err)
	}
}

func TestCompactado_SubstituirTrocaAPastaInteira(t *testing.T) {
	_, baixados := casa(t)
	v1 := grava(t, filepath.Join(baixados, "app-1.0.tar.gz"), falso.TarGz([]falso.Entrada{{Nome: "app/app", Conteudo: binario, Modo: 0o755}, {Nome: "app/velho.txt", Conteudo: "v1"}}))
	v2 := grava(t, filepath.Join(baixados, "app-2.0.tar.gz"), falso.TarGz([]falso.Entrada{{Nome: "app/app", Conteudo: binario, Modo: 0o755}, {Nome: "app/novo.txt", Conteudo: "v2"}}))
	if _, _, err := instala(t, v1, Opcoes{}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := instala(t, v2, Opcoes{}); err == nil || !strings.Contains(err.Error(), "já está instalado") {
		t.Errorf("sem substituir: %v", err)
	}
	if _, _, err := instala(t, v2, Opcoes{Substituir: true}); err != nil {
		t.Fatal(err)
	}
	if quer := []string{"app", "novo.txt"}; !reflect.DeepEqual(tudo(t, base.Aplicativos("app")), quer) {
		t.Errorf("depois de substituir: %v", tudo(t, base.Aplicativos("app")))
	}
	// Uma pasta que a pessoa pôs lá não é trocada.
	grava(t, base.Aplicativos("meu", "coisa.txt"), []byte("da pessoa"))
	meu := grava(t, filepath.Join(baixados, "meu.tar.gz"), falso.TarGz([]falso.Entrada{{Nome: "meu/meu", Conteudo: binario, Modo: 0o755}}))
	if _, _, err := instala(t, meu, Opcoes{Substituir: true}); err == nil || !strings.Contains(err.Error(), "já existe") {
		t.Errorf("pasta da pessoa: %v", err)
	}
	if le(t, base.Aplicativos("meu", "coisa.txt")) != "da pessoa" {
		t.Error("trocou a pasta da pessoa")
	}
}

func TestCompactado_XzEZstdPassamPeloComandoQueSoVeOStdin(t *testing.T) {
	m, baixados := casa(t)
	puro := falso.Tar([]falso.Entrada{{Nome: "app/app", Conteudo: binario, Modo: 0o755}})
	for nome, comando := range map[string]string{"-app um.tar.xz": "xz", "-app dois.tar.zst": "zstd", "-app tres.txz": "xz"} {
		arq := grava(t, filepath.Join(baixados, nome), puro)
		if _, err := Inspeciona(arq); err == nil || !strings.Contains(err.Error(), "é preciso o "+comando) {
			t.Errorf("%s sem o %s: %v", nome, comando, err)
		}
		m.Comando(comando, `[ "$#" = 1 ] && [ "$1" = -dc ] || { echo "argumentos: $*" >&2; exit 9; }; exec /bin/cat`)
		fim, _, err := instala(t, arq, Opcoes{})
		if err != nil {
			t.Fatalf("%s: %v", nome, err)
		}
		if !existe(filepath.Join(*fim.Destino, "app")) {
			t.Errorf("%s: não extraiu", nome)
		}
		for _, c := range m.Chamadas(comando) {
			if !reflect.DeepEqual(c, []string{"-dc"}) {
				t.Errorf("%s viu mais que -dc: %q", comando, c)
			}
		}
		os.Remove(filepath.Join(m.Dir, comando))
	}
	// O descompressor reclama: a frase dele chega.
	m.Comando("xz", `printf 'xz: (stdin): File format not recognized\n' >&2; exit 1`)
	arq := grava(t, filepath.Join(baixados, "ruim.tar.xz"), []byte("lixo"))
	if _, err := Inspeciona(arq); err == nil || !strings.Contains(err.Error(), "File format not recognized") {
		t.Errorf("erro do xz: %v", err)
	}
	// Um .tar.gz que não é gzip, um zip que não é zip, um arquivo vazio.
	for nome, conteudo := range map[string][]byte{"a.tar.gz": []byte("isto não é gzip"), "b.zip": []byte("nem isto é zip"), "c.tar.gz": falso.TarGz(nil), "d.tar.bz2": []byte("BZnão")} {
		if _, err := Inspeciona(grava(t, filepath.Join(baixados, nome), conteudo)); err == nil || base.FoiRecusa(err) {
			t.Errorf("%s: devia ser erro comum, veio %v", nome, err)
		}
	}
}

// ---- os maliciosos ------------------------------------------------------

// Cada caso é um arquivo que tenta sair da pasta, ou encher o disco. Vale
// para tar e para zip. `soNaExtracao` são os que só se revelam ao extrair
// (cada link, sozinho, parece ficar dentro).
func TestCompactado_RecusaOsMaliciososENaoDeixaNadaParaTras(t *testing.T) {
	app := falso.Entrada{Nome: "app/app", Conteudo: binario, Modo: 0o755}
	casos := []struct {
		nome         string
		entradas     []falso.Entrada
		trecho       string
		soTar        bool
		soNaExtracao bool
		limiteMB     string
		limiteN      string
	}{
		{nome: "zip slip com ../", entradas: []falso.Entrada{app, {Nome: "app/../../fora/pwn", Conteudo: "x"}}, trecho: "sai da pasta"},
		{nome: "../ logo no começo", entradas: []falso.Entrada{{Nome: "../fora/pwn", Conteudo: "x"}}, trecho: "sai da pasta"},
		{nome: "caminho absoluto", entradas: []falso.Entrada{app, {Nome: "/tmp/mkutil-pwn", Conteudo: "x"}}, trecho: "absoluto"},
		{nome: "barra invertida", entradas: []falso.Entrada{app, {Nome: "app\\..\\..\\fora", Conteudo: "x"}}, trecho: "nome estranho"},
		{nome: "pasta com ..", entradas: []falso.Entrada{app, {Nome: "app/../../fora/", Tipo: 'd'}}, trecho: "sai da pasta"},
		{nome: "link absoluto", entradas: []falso.Entrada{app, {Nome: "app/link", Tipo: 'l', Alvo: "/etc/passwd"}}, trecho: "aponta para fora"},
		{nome: "link que sobe", entradas: []falso.Entrada{app, {Nome: "app/link", Tipo: 'l', Alvo: "../../fora"}}, trecho: "aponta para fora"},
		{nome: "link para o topo, com a pasta única tirada", entradas: []falso.Entrada{app, {Nome: "app/link", Tipo: 'l', Alvo: "../fora"}}, trecho: "aponta para fora"},
		{nome: "link para fora e escrita por ele", entradas: []falso.Entrada{app, {Nome: "app/link", Tipo: 'l', Alvo: "../../fora"}, {Nome: "app/link/pwn", Conteudo: "x"}}, trecho: "aponta para fora"},
		{nome: "dois links que juntos saem", entradas: []falso.Entrada{app, {Nome: "app/d/", Tipo: 'd'}, {Nome: "app/d/l1", Tipo: 'l', Alvo: ".."},
			{Nome: "app/l2", Tipo: 'l', Alvo: "d/l1/.."}}, trecho: "aponta para fora", soNaExtracao: true},
		{nome: "dois links que juntos saem e escrita por eles", entradas: []falso.Entrada{app, {Nome: "app/d/", Tipo: 'd'}, {Nome: "app/d/l1", Tipo: 'l', Alvo: ".."},
			{Nome: "app/l2", Tipo: 'l', Alvo: "d/l1/../.."}, {Nome: "app/l2/fora/pwn", Conteudo: "x"}}, trecho: "fora da pasta", soNaExtracao: true},
		{nome: "link solto com .. por dentro de outro link", entradas: []falso.Entrada{app, {Nome: "app/d/", Tipo: 'd'}, {Nome: "app/d/l1", Tipo: 'l', Alvo: ".."},
			{Nome: "app/l2", Tipo: 'l', Alvo: "d/l1/../fora/ainda-nao-existe"}}, trecho: "aponta para fora", soNaExtracao: true},
		{nome: "link físico para fora", entradas: []falso.Entrada{app, {Nome: "app/duro", Tipo: 'h', Alvo: "../../fora/segredo"}}, trecho: "link físico", soTar: true},
		{nome: "link físico absoluto", entradas: []falso.Entrada{app, {Nome: "app/duro", Tipo: 'h', Alvo: "/etc/passwd"}}, trecho: "link físico", soTar: true},
		{nome: "dispositivo", entradas: []falso.Entrada{app, {Nome: "app/sda", Tipo: 'c'}}, trecho: "arquivo especial"},
		{nome: "fifo", entradas: []falso.Entrada{app, {Nome: "app/cano", Tipo: 'f'}}, trecho: "arquivo especial"},
		{nome: "arquivo trocando pasta", entradas: []falso.Entrada{app, {Nome: "app/d/x", Conteudo: "x"}, {Nome: "app/d", Conteudo: "x"}}, trecho: "troca uma pasta", soNaExtracao: true},
		{nome: "bomba de tamanho", entradas: []falso.Entrada{app, {Nome: "app/zeros", Tamanho: 3 << 20}}, trecho: "bomba", limiteMB: "2"},
		{nome: "bomba em muitos pedaços", entradas: []falso.Entrada{app, {Nome: "app/a", Tamanho: 1 << 20}, {Nome: "app/b", Tamanho: 1 << 20}, {Nome: "app/c", Tamanho: 1 << 20}}, trecho: "bomba", limiteMB: "2"},
		{nome: "bomba de entradas", entradas: []falso.Entrada{app, {Nome: "app/1"}, {Nome: "app/2"}, {Nome: "app/3"}, {Nome: "app/4"}, {Nome: "app/5"}}, trecho: "bomba", limiteN: "5"},
	}
	for _, c := range casos {
		formatos := map[string][]byte{"tar.gz": falso.TarGz(c.entradas)}
		if !c.soTar {
			formatos["zip"] = falso.Zip(c.entradas, false)
		}
		for formato, conteudo := range formatos {
			t.Run(c.nome+" ("+formato+")", func(t *testing.T) {
				_, baixados := casa(t)
				t.Setenv("MKUTIL_LIMITE_MB", c.limiteMB)
				t.Setenv("MKUTIL_LIMITE_ENTRADAS", c.limiteN)
				segredo := grava(t, base.Home("fora", "segredo"), []byte("segredo"))
				arq := grava(t, filepath.Join(baixados, "app."+formato), conteudo)
				antes := tudo(t, filepath.Dir(baixados))
				_, err := Inspeciona(arq)
				if !c.soNaExtracao {
					querRecusa(t, err, c.trecho)
				} else if err != nil {
					t.Errorf("a inspeção devia passar (cada link sozinho fica dentro): %v", err)
				}
				_, _, err = instala(t, arq, Opcoes{})
				querRecusa(t, err, c.trecho)
				depois := tudo(t, filepath.Dir(baixados))
				// Só pode ter aparecido a pasta ~/Aplicativos, vazia.
				sobra := []string{}
				for _, p := range depois {
					if p != "casa/Aplicativos/" {
						sobra = append(sobra, p)
					}
				}
				if !reflect.DeepEqual(antes, sobra) {
					t.Errorf("mudou o disco:\n antes  %v\n depois %v", antes, depois)
				}
				if le(t, segredo) != "segredo" || existe("/tmp/mkutil-pwn") {
					t.Error("escreveu fora da pasta")
				}
			})
		}
	}
}

func TestCompactado_BombaDeTamanhoDeclaradoERecusadaSemLerOsBytes(t *testing.T) {
	_, baixados := casa(t)
	// Um tar cujo cabeçalho declara 1 TB (e não traz os bytes): com o limite
	// padrão, de 8 GB, é recusado no cabeçalho, sem ler nem escrever nada.
	var b bytes.Buffer
	w := tar.NewWriter(&b)
	if err := w.WriteHeader(&tar.Header{Name: "app/zeros", Mode: 0o644, Size: 1 << 40, Typeflag: tar.TypeReg, Format: tar.FormatPAX}); err != nil {
		t.Fatal(err)
	}
	arq := grava(t, filepath.Join(baixados, "bomba.tar.gz"), gz(t, b.Bytes()))
	_, err := Inspeciona(arq)
	querRecusa(t, err, "bomba")
	_, _, err = instala(t, arq, Opcoes{})
	querRecusa(t, err, "bomba")
	if len(tudo(t, base.Aplicativos())) != 0 {
		t.Error("sobrou algo")
	}
}

func TestCompactado_OLimiteEConfiguravel(t *testing.T) {
	_, baixados := casa(t)
	arq := grava(t, filepath.Join(baixados, "app.tar.gz"), falso.TarGz([]falso.Entrada{{Nome: "app/app", Conteudo: binario, Modo: 0o755}, {Nome: "app/dados", Tamanho: 3 << 20}}))
	t.Setenv("MKUTIL_LIMITE_MB", "2")
	if _, err := Inspeciona(arq); err == nil || !strings.Contains(err.Error(), "passa de 2 MB") {
		t.Errorf("limite de 2 MB: %v", err)
	}
	t.Setenv("MKUTIL_LIMITE_MB", "4")
	if _, _, err := instala(t, arq, Opcoes{}); err != nil {
		t.Errorf("limite de 4 MB: %v", err)
	}
}

func gz(t *testing.T, b []byte) []byte {
	t.Helper()
	var out bytes.Buffer
	w := gzip.NewWriter(&out)
	_, _ = w.Write(b)
	_ = w.Close()
	return out.Bytes()
}
