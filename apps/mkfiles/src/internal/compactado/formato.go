// Package compactado cuida dos arquivos compactados: dizer o que há dentro
// (sem extrair), abrir um arquivo de dentro, extrair e compactar. zip, tar,
// gzip e bzip2 são da biblioteca padrão; xz, zstd, 7z e rar passam por um
// comando de fora, quando existe. Extrair e compactar são tarefas do trilho
// (../operar). Veja ../../../CONTRATO.md, "Compactados".
package compactado

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"strings"

	"mkfiles/internal/base"
)

// Os formatos que o contrato nomeia.
var deExtrair = []string{"zip", "tar", "tar.gz", "tar.bz2", "tar.xz", "tar.zst", "gz", "bz2", "xz", "zst", "7z", "rar"}
var deCompactar = []string{"zip", "tar.gz", "tar.zst", "tar.xz", "7z"}

// As extensões de cada formato, da mais longa para a mais curta.
var extensoes = []struct{ ext, formato string }{
	{".tar.gz", "tar.gz"}, {".tar.bz2", "tar.bz2"}, {".tar.xz", "tar.xz"}, {".tar.zst", "tar.zst"},
	{".tgz", "tar.gz"}, {".tbz2", "tar.bz2"}, {".tbz", "tar.bz2"}, {".txz", "tar.xz"}, {".tzst", "tar.zst"},
	{".zip", "zip"}, {".tar", "tar"}, {".gz", "gz"}, {".bz2", "bz2"}, {".xz", "xz"}, {".zst", "zst"},
	{".7z", "7z"}, {".rar", "rar"},
}

// peloNome diz o formato que o nome promete e o nome sem a extensão.
func peloNome(nome string) (formato, semExtensao string) {
	minusculo := strings.ToLower(nome)
	for _, e := range extensoes {
		if strings.HasSuffix(minusculo, e.ext) && len(nome) > len(e.ext) {
			return e.formato, nome[:len(nome)-len(e.ext)]
		}
	}
	return "", nome
}

// pelosBytes diz o que o começo do arquivo é: zip, gz, bz2, xz, zst, 7z, rar,
// tar ou "".
func pelosBytes(b []byte) string {
	tem := func(m string) bool { return bytes.HasPrefix(b, []byte(m)) }
	switch {
	case tem("PK\x03\x04"), tem("PK\x05\x06"):
		return "zip"
	case tem("\x1f\x8b"):
		return "gz"
	case tem("BZh"):
		return "bz2"
	case tem("\xfd7zXZ\x00"):
		return "xz"
	case tem("\x28\xb5\x2f\xfd"):
		return "zst"
	case tem("7z\xbc\xaf\x27\x1c"):
		return "7z"
	case tem("Rar!\x1a\x07"):
		return "rar"
	case len(b) >= 262 && string(b[257:262]) == "ustar":
		return "tar"
	}
	return ""
}

// formatoDe decide o formato de um arquivo: os bytes dizem a casca, o nome diz
// se há um tar dentro. "" quando não é um compactado que se conheça (ou o
// nome promete uma coisa e os bytes são outra).
func formatoDe(caminho string) string {
	nome, _ := peloNome(caminho[strings.LastIndexByte(caminho, '/')+1:])
	f, err := os.Open(caminho)
	if err != nil {
		return ""
	}
	defer f.Close()
	cabeca := make([]byte, 512)
	n, _ := io.ReadFull(f, cabeca)
	casca := pelosBytes(cabeca[:n])
	switch casca {
	case "":
		if nome == "tar" && n == 512 {
			return "tar" // tar antigo, sem "ustar"
		}
		return ""
	case "gz", "bz2", "xz", "zst":
		if nome == "tar."+casca {
			return nome
		}
	}
	return casca
}

// ferramenta acha o comando de fora: o da variável (se está posta, só ele), ou
// o primeiro dos candidatos que houver. "" se não há.
func ferramenta(variavel string, candidatos ...string) string {
	if v := os.Getenv(variavel); v != "" {
		candidatos = []string{v}
	}
	for _, c := range candidatos {
		if achado, err := exec.LookPath(c); err == nil {
			return achado
		}
	}
	return ""
}

func oXz() string     { return ferramenta("MKFILES_XZ", "xz") }
func oZstd() string   { return ferramenta("MKFILES_ZSTD", "zstd") }
func o7z() string     { return ferramenta("MKFILES_7Z", "7z", "7zz", "7za") }
func oUnrar() string  { return ferramenta("MKFILES_UNRAR", "unrar") }
func oBsdtar() string { return ferramenta("MKFILES_BSDTAR", "bsdtar") }

// falta diz o pacote que falta para ler (ou gravar) um formato; "" se dá.
func falta(formato string) string {
	switch formato {
	case "tar.xz", "xz":
		if oXz() == "" {
			return "xz"
		}
	case "tar.zst", "zst":
		if oZstd() == "" {
			return "zstd"
		}
	case "7z":
		if o7z() == "" {
			return "7zip"
		}
	case "rar":
		if o7z() == "" && oUnrar() == "" && oBsdtar() == "" {
			return "unrar"
		}
	}
	return ""
}

func disponibilidade(formatos []string) []base.Campos {
	lista := make([]base.Campos, 0, len(formatos))
	for _, f := range formatos {
		pacote := falta(f)
		c := base.Campos{"formato": f, "disponivel": pacote == "", "falta": nil}
		if pacote != "" {
			c["falta"] = pacote
		}
		lista = append(lista, c)
	}
	return lista
}

// erroDe é um erro com o "motivo" do contrato (senha, falta, formato).
type erroDe struct {
	motivo, frase, pacote string
}

func (e *erroDe) Error() string { return e.frase }

func semFerramenta(pacote string) error {
	return &erroDe{motivo: "falta", pacote: pacote, frase: "para isso falta o pacote " + pacote}
}

var (
	erroDeFormato  = &erroDe{motivo: "formato", frase: "não é um compactado que eu conheça"}
	erroPedeSenha  = &erroDe{motivo: "senha", frase: "este arquivo pede uma senha"}
	erroSenhaRuim  = &erroDe{motivo: "senha", frase: "a senha não confere"}
	erroCifradoZip = &erroDe{motivo: "falta", pacote: "7zip", frase: "zip com senha precisa do 7zip"}
)

// camposDoErro monta os campos a mais da resposta de erro.
func camposDoErro(err error) base.Campos {
	if e, ok := err.(*erroDe); ok {
		c := base.Campos{"motivo": e.motivo}
		if e.pacote != "" {
			c["pacote"] = e.pacote
		}
		return c
	}
	return base.Campos{"motivo": "erro"}
}
