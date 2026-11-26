// Package mime descobre o tipo de um arquivo: pelo nome, com o banco
// shared-mime-info (globs2), e por um farejo pequeno de conteúdo quando o nome
// não diz nada. Também diz o nome do ícone do tema para cada tipo.
package mime

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"mkfiles/internal/base"
)

// Os tipos que não saem do banco.
const (
	Pasta    = "inode/directory"
	Link     = "inode/symlink"
	Binario  = "application/octet-stream"
	Vazio    = "application/x-zerosize"
	Texto    = "text/plain"
	farejado = 512
)

type padrao struct {
	peso    int
	mime    string
	glob    string
	exato   bool // o padrão diferencia maiúsculas ("cs")
	tamanho int
}

type banco struct {
	literais  map[string][]padrao // nome inteiro (como está e em minúsculas)
	sufixos   map[string][]padrao // ".ext", em minúsculas
	outros    []padrao            // padrões com curinga no meio
	genericos map[string]string   // mime -> ícone genérico
}

var (
	trava  sync.Mutex
	aberto *banco
)

// Zerar esquece o banco lido (os testes trocam MKFILES_MIME).
func Zerar() {
	trava.Lock()
	aberto = nil
	trava.Unlock()
}

func oBanco() *banco {
	trava.Lock()
	defer trava.Unlock()
	if aberto == nil {
		aberto = ler(base.Raiz("MKFILES_MIME", "/usr/share/mime"))
	}
	return aberto
}

func ler(raiz string) *banco {
	b := &banco{literais: map[string][]padrao{}, sufixos: map[string][]padrao{}, genericos: map[string]string{}}
	if f, err := os.Open(filepath.Join(raiz, "globs2")); err == nil {
		linhas := bufio.NewScanner(f)
		for linhas.Scan() {
			linha := linhas.Text()
			if linha == "" || linha[0] == '#' {
				continue
			}
			campos := strings.SplitN(linha, ":", 4)
			if len(campos) < 3 || campos[2] == "" {
				continue
			}
			peso, err := strconv.Atoi(campos[0])
			if err != nil {
				continue
			}
			p := padrao{peso: peso, mime: campos[1], glob: campos[2], tamanho: len(campos[2])}
			if len(campos) == 4 {
				for _, marca := range strings.Split(campos[3], ",") {
					if marca == "cs" {
						p.exato = true
					}
				}
			}
			switch {
			case !strings.ContainsAny(p.glob, "*?["):
				chave := p.glob
				if !p.exato {
					chave = strings.ToLower(chave)
				}
				b.literais[chave] = append(b.literais[chave], p)
			case strings.HasPrefix(p.glob, "*.") && !strings.ContainsAny(p.glob[1:], "*?["):
				chave := strings.ToLower(p.glob[1:])
				b.sufixos[chave] = append(b.sufixos[chave], p)
			default:
				b.outros = append(b.outros, p)
			}
		}
		f.Close()
	}
	if dados, err := os.ReadFile(filepath.Join(raiz, "generic-icons")); err == nil {
		for _, linha := range strings.Split(string(dados), "\n") {
			if m, icone, ok := strings.Cut(linha, ":"); ok && icone != "" {
				b.genericos[m] = icone
			}
		}
	}
	return b
}

// DoNome diz o tipo pelo nome do arquivo. achou é false quando nenhum padrão
// do banco casa.
func DoNome(nome string) (mime string, achou bool) {
	b := oBanco()
	minusculo := strings.ToLower(nome)
	var melhor padrao
	ve := func(p padrao, casaExato bool) {
		if p.exato && !casaExato {
			return
		}
		if !achou || p.peso > melhor.peso || (p.peso == melhor.peso && (p.tamanho > melhor.tamanho || (p.tamanho == melhor.tamanho && p.exato && !melhor.exato))) {
			melhor, achou = p, true
		}
	}
	for _, p := range b.literais[nome] {
		ve(p, true)
	}
	if minusculo != nome {
		for _, p := range b.literais[minusculo] {
			ve(p, false)
		}
	}
	// Cada ponto do nome abre um sufixo: "a.tar.gz" tenta ".tar.gz" e ".gz".
	for i := 0; i < len(nome); i++ {
		if nome[i] != '.' || i == 0 {
			continue
		}
		for _, p := range b.sufixos[minusculo[i:]] {
			ve(p, strings.HasSuffix(nome, p.glob[1:]))
		}
	}
	for _, p := range b.outros {
		if ok, _ := filepath.Match(p.glob, nome); ok {
			ve(p, true)
		} else if !p.exato {
			if ok, _ := filepath.Match(strings.ToLower(p.glob), minusculo); ok {
				ve(p, false)
			}
		}
	}
	return melhor.mime, achou
}

// DoConteudo fareja o começo de um arquivo (até 512 bytes).
func DoConteudo(b []byte) string {
	tem := func(magico string) bool { return bytes.HasPrefix(b, []byte(magico)) }
	switch {
	case len(b) == 0:
		return Vazio
	case tem("\x89PNG\r\n\x1a\n"):
		return "image/png"
	case tem("\xff\xd8\xff"):
		return "image/jpeg"
	case tem("GIF87a"), tem("GIF89a"):
		return "image/gif"
	case tem("%PDF-"):
		return "application/pdf"
	case tem("PK\x03\x04"), tem("PK\x05\x06"):
		return "application/zip"
	case tem("\x1f\x8b"):
		return "application/gzip"
	case tem("\x7fELF"):
		return "application/x-executable"
	case tem("#!"):
		return "application/x-shellscript"
	}
	if eTexto(b) {
		return Texto
	}
	return Binario
}

// eTexto: UTF-8 válido (a não ser por um caractere cortado no fim) e sem
// bytes de controle além de tab, quebra de linha, avanço de página e escape.
func eTexto(b []byte) bool {
	for len(b) > 0 {
		r, n := utf8.DecodeRune(b)
		if r == utf8.RuneError && n == 1 {
			return len(b) < utf8.UTFMax && !utf8.FullRune(b)
		}
		if r < 0x20 && r != '\t' && r != '\n' && r != '\r' && r != '\f' && r != 0x1b {
			return false
		}
		b = b[n:]
	}
	return true
}

// TemExtensao diz se o nome tem uma extensão (um ponto que não é o primeiro
// caractere, com algo depois).
func TemExtensao(nome string) bool {
	i := strings.LastIndexByte(nome, '.')
	return i > 0 && i < len(nome)-1
}

// DoArquivo diz o tipo de um arquivo comum: pelo nome e, quando o nome não
// casa com nada, pelo conteúdo. Com sempre=false o conteúdo só é lido se o
// nome não tem extensão (é o que o listar faz, para não abrir a pasta toda).
func DoArquivo(caminho string, tamanho int64, sempre bool) string {
	nome := filepath.Base(caminho)
	if m, achou := DoNome(nome); achou {
		return m
	}
	if tamanho == 0 {
		return Vazio
	}
	if !sempre && TemExtensao(nome) {
		return Binario
	}
	f, err := os.Open(caminho)
	if err != nil {
		return Binario
	}
	defer f.Close()
	b := make([]byte, farejado)
	n, _ := io.ReadFull(f, b)
	return DoConteudo(b[:n])
}

// Icone diz o nome do ícone do tema para um tipo, e o genérico que serve de
// reserva.
func Icone(mime string) (icone, reserva string) {
	if mime == Pasta {
		return "folder", "folder"
	}
	tipo, _, ok := strings.Cut(mime, "/")
	if !ok || tipo == "" {
		return "application-octet-stream", "application-x-generic"
	}
	reserva = oBanco().genericos[mime]
	if reserva == "" {
		reserva = tipo + "-x-generic"
	}
	return strings.ReplaceAll(mime, "/", "-"), reserva
}

// Familia agrupa os tipos para o filtro da busca: "imagem", "video", "audio",
// "texto", "documento" ou "".
func Familia(mime string) string {
	tipo, sub, _ := strings.Cut(mime, "/")
	switch tipo {
	case "image":
		return "imagem"
	case "video":
		return "video"
	case "audio":
		return "audio"
	case "text":
		return "texto"
	}
	if mime == "application/pdf" || strings.Contains(sub, "opendocument") || strings.Contains(sub, "officedocument") ||
		strings.HasPrefix(sub, "msword") || strings.HasPrefix(sub, "vnd.ms-") || sub == "rtf" || sub == "epub+zip" {
		return "documento"
	}
	return ""
}
