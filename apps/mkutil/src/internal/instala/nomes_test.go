package instala

import "testing"

func TestIdentificaTiraVersaoEArquiteturaDoNome(t *testing.T) {
	casos := []struct{ arquivo, id, nome, versao string }{
		{"Krita-5.2.2-x86_64.AppImage", "krita", "Krita", "5.2.2"},
		{"krita-5.2.2-x86_64.appimage", "krita", "krita", "5.2.2"},
		{"Obsidian-1.5.3.AppImage", "obsidian", "Obsidian", "1.5.3"},
		{"balenaEtcher-1.18.11-x64.AppImage", "balenaetcher", "balenaEtcher", "1.18.11"},
		{"firefox-128.0b3.tar.xz", "firefox", "firefox", "128.0b3"},
		{"GitHub Desktop-linux-3.3.6.AppImage", "github-desktop", "GitHub Desktop", "3.3.6"},
		{"tor-browser-linux-x86_64-13.0.9.tar.xz", "tor-browser", "tor browser", "13.0.9"},
		{"my_app_v2.1.zip", "my-app", "my app", "2.1"},
		{"7zip-linux.tar.xz", "7zip", "7zip", ""},
		{"2048.AppImage", "2048", "2048", ""},
		{"godot-4.tar.gz", "godot", "godot", "4"},
		{"blender-3d.tgz", "blender-3d", "blender 3d", ""},
		{"app-x86_64.AppImage", "app", "app", ""},
		{"Só Ação!.zip", "s-a-o", "Só Ação!", ""},
		{"htop-3.3.0-3-x86_64.pkg.tar.zst", "htop", "htop", "3.3.0"},
		{"semextensao", "semextensao", "semextensao", ""},
		{"---.zip", "app", "app", ""},
		{".AppImage", "appimage", ".AppImage", ""},
	}
	for _, c := range casos {
		id, nome, versao := Identifica(c.arquivo)
		v := ""
		if versao != nil {
			v = *versao
		}
		if id != c.id || nome != c.nome || v != c.versao {
			t.Errorf("%q: veio (%q, %q, %q), queria (%q, %q, %q)", c.arquivo, id, nome, v, c.id, c.nome, c.versao)
		}
		if !IdValido(id) {
			t.Errorf("%q: o id %q não é válido", c.arquivo, id)
		}
	}
}

func TestExtensaoDeConheceAsCompostasPrimeiro(t *testing.T) {
	casos := []struct{ nome, sem, tipo, formato string }{
		{"a.pkg.tar.zst", "a", "pacote", "tar.zst"},
		{"a.pkg.tar.xz", "a", "pacote", "tar.xz"},
		{"a.tar.zst", "a", "compactado", "tar.zst"},
		{"a.TAR.GZ", "a", "compactado", "tar.gz"},
		{"a.tgz", "a", "compactado", "tar.gz"},
		{"a.tar.bz2", "a", "compactado", "tar.bz2"},
		{"a.b.zip", "a.b", "compactado", "zip"},
		{"a.AppImage", "a", "appimage", ""},
		{"a.deb", "a", "deb", ""},
		{"a.flatpakref", "a", "flatpak", ""},
		{"a.txt", "a.txt", "", ""},
		{".zip", ".zip", "", ""},
	}
	for _, c := range casos {
		sem, tipo, formato := extensaoDe(c.nome)
		if sem != c.sem || tipo != c.tipo || formato != c.formato {
			t.Errorf("%q: veio (%q, %q, %q)", c.nome, sem, tipo, formato)
		}
	}
}

func TestIdValidoRecusaOQueViraCaminhoOuOpcao(t *testing.T) {
	for _, id := range []string{"krita", "7zip", "a.b_c+d-e", "a"} {
		if !IdValido(id) {
			t.Errorf("%q devia valer", id)
		}
	}
	for _, id := range []string{"", "-rf", ".oculto", "a/b", "../a", "a..b", "A", "a b", "a\n", "ação", "a*"} {
		if IdValido(id) {
			t.Errorf("%q não devia valer", id)
		}
	}
	if got := FazId("  --Olá/../Mundo--  "); got != "ol-.-mundo" {
		t.Errorf("FazId: %q", got)
	}
	if !IdValido(FazId("a/../b")) || !IdValido(FazId("....")) {
		t.Errorf("FazId tem de dar sempre um id válido: %q, %q", FazId("a/../b"), FazId("...."))
	}
}
