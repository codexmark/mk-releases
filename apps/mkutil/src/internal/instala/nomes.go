// Package instala é o MkInstalar: diz o que um arquivo baixado é (AppImage,
// pacote do Arch, arquivo compactado), instala em ~/Aplicativos (ou pelo
// pacman), cria o atalho de menu, anota no registro e desinstala o que ele
// mesmo instalou. O contrato está em ../../../CONTRATO.md.
package instala

import (
	"regexp"
	"strings"
)

// As extensões que o MkInstalar conhece, as compostas antes.
var extensoes = []struct{ ext, tipo, formato string }{
	{".pkg.tar.zst", "pacote", "tar.zst"},
	{".pkg.tar.xz", "pacote", "tar.xz"},
	{".tar.gz", "compactado", "tar.gz"},
	{".tgz", "compactado", "tar.gz"},
	{".tar.xz", "compactado", "tar.xz"},
	{".txz", "compactado", "tar.xz"},
	{".tar.zst", "compactado", "tar.zst"},
	{".tar.bz2", "compactado", "tar.bz2"},
	{".tbz2", "compactado", "tar.bz2"},
	{".zip", "compactado", "zip"},
	{".appimage", "appimage", ""},
	{".deb", "deb", ""},
	{".rpm", "rpm", ""},
	{".flatpakref", "flatpak", ""},
	{".flatpak", "flatpak", ""},
}

// extensaoDe separa o nome do arquivo da extensão conhecida. Sem extensão
// conhecida, tipo é "".
func extensaoDe(nome string) (semExt, tipo, formato string) {
	baixo := strings.ToLower(nome)
	for _, e := range extensoes {
		if strings.HasSuffix(baixo, e.ext) && len(nome) > len(e.ext) {
			return nome[:len(nome)-len(e.ext)], e.tipo, e.formato
		}
	}
	return nome, "", ""
}

var (
	// "5.2.2", "v1.0", "128.0b3", "2024.1", "3" ou uma data "20240131".
	pareceVersao = regexp.MustCompile(`^[vV]?\d+(\.\d+)+[0-9A-Za-z.+~]*$|^[vV]?\d+$`)
	naoEDoId     = regexp.MustCompile(`[^a-z0-9._+-]+`)
	hifens       = regexp.MustCompile(`-{2,}`)
	pontos       = regexp.MustCompile(`\.{2,}`)
)

// O que vem depois do nome e não é dele: arquitetura, sistema, empacotamento.
var ruido = map[string]bool{
	"x86": true, "x64": true, "amd64": true, "aarch64": true, "arm64": true, "armhf": true, "i386": true, "i686": true,
	"linux": true, "linux64": true, "gnu": true, "glibc": true, "appimage": true, "any": true, "portable": true,
	"latest": true, "stable": true, "release": true,
}

// Identifica tira do nome de um arquivo o nome do app, o id (o nome no disco e
// a chave no registro) e a versão, quando dá: "Krita-5.2.2-x86_64.AppImage"
// vira ("krita", "Krita", "5.2.2"). O primeiro pedaço é sempre do nome, mesmo
// parecendo versão ("7zip").
func Identifica(nomeDoArquivo string) (id, nome string, versao *string) {
	semExt, _, _ := extensaoDe(nomeDoArquivo)
	pedacos := strings.FieldsFunc(semExt, func(r rune) bool { return r == '-' || r == '_' || r == ' ' })
	corte := len(pedacos)
	for i, p := range pedacos {
		if i == 0 {
			continue
		}
		if pareceVersao.MatchString(p) || ruido[strings.ToLower(p)] {
			corte = i
			break
		}
	}
	// A versão é o primeiro pedaço depois do nome com cara de versão. Um
	// número solto mais adiante ("64" de "x86_64") não é.
	for i := corte; i < len(pedacos); i++ {
		if p := pedacos[i]; pareceVersao.MatchString(p) && (i == corte || strings.Contains(p, ".")) {
			v := strings.TrimLeft(p, "vV")
			versao = &v
			break
		}
	}
	nome = strings.Join(pedacos[:corte], " ")
	id = FazId(strings.Join(pedacos[:corte], "-"))
	if nome == "" {
		nome = id
	}
	return id, nome, versao
}

// FazId reduz um texto ao alfabeto do id: a-z 0-9 . _ + -, começando por
// letra ou número. O que sobra vazio vira "app".
func FazId(texto string) string {
	id := naoEDoId.ReplaceAllString(strings.ToLower(texto), "-")
	id = hifens.ReplaceAllString(id, "-")
	id = pontos.ReplaceAllString(id, ".")
	id = strings.Trim(id, "-._+")
	if len(id) > 64 {
		id = strings.Trim(id[:64], "-._+")
	}
	if id == "" {
		return "app"
	}
	return id
}

var idValido = regexp.MustCompile(`^[a-z0-9][a-z0-9._+-]{0,63}$`)

// IdValido diz se um id pode virar nome de arquivo e de atalho.
func IdValido(id string) bool {
	return idValido.MatchString(id) && !strings.Contains(id, "..")
}
