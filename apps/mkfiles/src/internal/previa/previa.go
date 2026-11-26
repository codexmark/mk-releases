// Package previa atende o pedido previa: o que a janela mostra na prévia
// rápida (Espaço) de um item. Diz a classe do item e, para o que é barato e
// seguro de ler aqui (um texto, o começo de uma pasta), já manda o conteúdo.
// Imagem, áudio, vídeo e PDF a janela mostra por conta dela (a miniatura
// x-large, no caso do PDF); o compactado, pelo pacote dos compactados.
package previa

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unicode/utf8"

	"mkfiles/internal/base"
	"mkfiles/internal/item"
	"mkfiles/internal/mime"
)

// Os limites do contrato.
const (
	Limite    = 200 << 10 // bytes de texto
	Primeiros = 50        // itens de uma pasta
)

// Os tipos que são texto sem ser text/*.
var textos = map[string]bool{
	"application/json": true, "application/xml": true, "application/x-shellscript": true,
	"application/javascript": true, "application/x-javascript": true, "application/toml": true,
	"application/yaml": true, "application/x-yaml": true, "application/x-desktop": true,
	"application/sql": true, "application/x-php": true, "application/x-perl": true, "application/x-ruby": true,
	"application/x-awk": true, "application/x-wine-extension-ini": true, "application/x-subrip": true,
	"application/x-zerosize": true, "application/ld+json": true, "application/x-ndjson": true,
	"application/x-gettext-translation-template": true, "application/x-troff-man": true,
}

// Os tipos de compactado que a janela sabe mostrar por dentro.
var compactados = map[string]bool{
	"application/zip": true, "application/x-tar": true, "application/x-compressed-tar": true,
	"application/gzip": true, "application/x-gzip": true, "application/x-bzip2": true, "application/x-bzip": true,
	"application/x-bzip-compressed-tar": true, "application/x-bzip2-compressed-tar": true,
	"application/x-xz": true, "application/x-xz-compressed-tar": true,
	"application/zstd": true, "application/x-zstd-compressed-tar": true,
	"application/x-7z-compressed": true, "application/vnd.rar": true, "application/x-rar": true,
	"application/x-rar-compressed": true, "application/java-archive": true,
}

// Classe diz como a prévia mostra um tipo: "imagem", "texto", "audio",
// "video", "pdf", "compactado" ou "outro" (a pasta é decidida por quem chama).
func Classe(tipo string) string {
	familia, sub, _ := strings.Cut(tipo, "/")
	switch {
	case familia == "image":
		return "imagem"
	case familia == "audio":
		return "audio"
	case familia == "video":
		return "video"
	case tipo == "application/pdf":
		return "pdf"
	case compactados[tipo]:
		return "compactado"
	case familia == "text", textos[tipo], strings.HasSuffix(sub, "+xml"), strings.HasSuffix(sub, "+json"):
		return "texto"
	}
	return "outro"
}

// Ligar põe na sessão o pedido previa.
func Ligar(s *base.Sessao) {
	s.Ligar("previa", func(p *base.Pedido) {
		var q struct {
			Caminho string `json:"caminho"`
		}
		_ = p.Ler(&q)
		if err := base.Caminho(q.Caminho); err != nil {
			p.Falha(err.Error())
			return
		}
		it, err := item.De(q.Caminho, true)
		if err != nil {
			motivo, frase := item.Motivo(err)
			if motivo == "nao-existe" {
				frase = "isso não existe mais"
			}
			p.Falha(frase, base.Campos{"motivo": motivo})
			return
		}
		r := base.Campos{"item": it, "classe": "outro", "texto": nil, "cortado": false, "binario": false,
			"itens": nil, "total": nil}
		// O que há de verdade nesse caminho (do outro lado, se é um link).
		tipo := it.Tipo
		if tipo == item.Link && it.AlvoTipo != nil {
			tipo = *it.AlvoTipo
		}
		switch tipo {
		case item.Pasta:
			r["classe"] = "pasta"
			if itens, total, ok := comeco(q.Caminho); ok {
				r["itens"], r["total"] = itens, total
			}
		case item.Arquivo:
			r["classe"] = Classe(it.Mime)
			if r["classe"] == "texto" {
				texto, cortado, binario, err := le(q.Caminho)
				switch {
				case err != nil:
					r["classe"] = "outro"
				case binario:
					r["binario"] = true
				default:
					r["texto"], r["cortado"] = texto, cortado
				}
			}
		}
		p.Responde(r)
	})
}

// comeco lê os primeiros itens de uma pasta (na ordem do disco) e conta todos.
func comeco(pasta string) ([]item.Item, int, bool) {
	f, err := os.Open(pasta)
	if err != nil {
		return nil, 0, false
	}
	defer f.Close()
	ocultos := item.Ocultos(pasta)
	itens := []item.Item{}
	total := 0
	for {
		nomes, err := f.Readdirnames(1024)
		for _, nome := range nomes {
			total++
			if len(itens) >= Primeiros {
				continue
			}
			caminho := filepath.Join(pasta, nome)
			if info, err := os.Lstat(caminho); err == nil {
				itens = append(itens, item.DeInfo(caminho, info, strings.HasPrefix(nome, ".") || ocultos[nome], false))
			}
		}
		if err != nil || len(nomes) == 0 {
			return itens, total, true
		}
	}
}

// le devolve o começo de um arquivo de texto. Só abre arquivo comum (o
// O_NONBLOCK é para um fifo que tomasse o lugar não prender o motor), e diz
// que é binário o que tem byte nulo ou não é UTF-8.
func le(caminho string) (texto string, cortado, binario bool, err error) {
	f, err := os.OpenFile(caminho, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return "", false, false, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return "", false, false, os.ErrInvalid
	}
	b := make([]byte, Limite+1)
	n, err := io.ReadFull(f, b)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return "", false, false, err
	}
	b = b[:n]
	if cortado = n > Limite; cortado {
		b = b[:Limite]
		// Volta até o começo de um caractere: o último pode ter ficado pela metade.
		for i := 0; i < utf8.UTFMax && len(b) > 0 && !utf8.Valid(b[len(b)-min(len(b), utf8.UTFMax):]); i++ {
			if r, tam := utf8.DecodeLastRune(b); r != utf8.RuneError || tam != 1 {
				break
			}
			b = b[:len(b)-1]
		}
	}
	if bytes.IndexByte(b, 0) >= 0 || !utf8.Valid(b) || mime.DoConteudo(b[:min(len(b), 512)]) == mime.Binario {
		return "", false, true, nil
	}
	return string(b), cortado, false, nil
}
