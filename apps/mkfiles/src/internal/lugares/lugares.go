// Package lugares cuida da barra lateral: os lugares fixos (user-dirs.dirs),
// os favoritos (o arquivo de bookmarks do GTK, o mesmo do Nautilus e dos
// seletores de arquivo) e os discos (lsblk e udisksctl). Veja
// ../../../CONTRATO.md.
package lugares

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"mkfiles/internal/base"
)

// pastaXDG lê uma pasta do user-dirs.dirs (XDG_DOCUMENTS_DIR="$HOME/Documentos").
// "" se o arquivo não a tem ou se o valor não é um caminho absoluto.
func pastaXDG(chave string) string {
	b, err := os.ReadFile(base.Config("user-dirs.dirs"))
	if err != nil {
		return ""
	}
	for _, linha := range strings.Split(string(b), "\n") {
		linha = strings.TrimSpace(linha)
		nome, valor, ok := strings.Cut(linha, "=")
		if !ok || strings.TrimSpace(nome) != chave {
			continue
		}
		valor = strings.TrimSpace(valor)
		if len(valor) < 2 || valor[0] != '"' || valor[len(valor)-1] != '"' {
			return ""
		}
		valor = strings.ReplaceAll(valor[1:len(valor)-1], `\"`, `"`)
		if resto, tem := strings.CutPrefix(valor, "$HOME"); tem {
			if resto != "" && resto[0] != '/' {
				return ""
			}
			valor = base.Home() + resto
		}
		if !filepath.IsAbs(valor) || strings.ContainsRune(valor, 0) {
			return ""
		}
		return filepath.Clean(valor)
	}
	return ""
}

// PastaDeModelos é a pasta de modelos: XDG_TEMPLATES_DIR do user-dirs.dirs
// (se não é a própria pasta pessoal), senão ~/Templates. "" se não há pasta
// pessoal.
func PastaDeModelos() string {
	h := base.Home()
	if h == "" {
		return ""
	}
	if p := pastaXDG("XDG_TEMPLATES_DIR"); p != "" && p != h {
		return p
	}
	return filepath.Join(h, "Templates")
}

func ePasta(c string) bool {
	info, err := os.Stat(c)
	return err == nil && info.IsDir()
}

func fixos() []base.Campos {
	h := base.Home()
	lista := []base.Campos{}
	if h != "" {
		lista = append(lista, base.Campos{"id": "pessoal", "nome": "Pasta pessoal", "caminho": h, "icone": "user-home"})
	}
	for _, l := range []struct{ id, nome, chave, icone string }{
		{"documentos", "Documentos", "XDG_DOCUMENTS_DIR", "folder-documents"},
		{"downloads", "Downloads", "XDG_DOWNLOAD_DIR", "folder-download"},
		{"imagens", "Imagens", "XDG_PICTURES_DIR", "folder-pictures"},
		{"musicas", "Músicas", "XDG_MUSIC_DIR", "folder-music"},
		{"videos", "Vídeos", "XDG_VIDEOS_DIR", "folder-videos"},
	} {
		if c := pastaXDG(l.chave); c != "" && c != h && ePasta(c) {
			lista = append(lista, base.Campos{"id": l.id, "nome": l.nome, "caminho": c, "icone": l.icone})
		}
	}
	return append(lista,
		base.Campos{"id": "lixeira", "nome": "Lixeira", "caminho": nil, "icone": "user-trash"},
		base.Campos{"id": "raiz", "nome": "Raiz", "caminho": "/", "icone": "drive-harddisk"})
}

// area é o estado desta área numa sessão: a trava do arquivo de favoritos e a
// vigia dos discos.
type area struct {
	s        *base.Sessao
	trava    sync.Mutex // favoritos e a vigia
	parar    chan struct{}
	montando sync.Mutex // um udisksctl por vez
}

// Ligar põe na sessão os pedidos dos lugares, dos favoritos e dos discos.
func Ligar(s *base.Sessao) {
	a := &area{s: s}
	s.Ligar("lugares", func(p *base.Pedido) {
		a.trava.Lock()
		defer a.trava.Unlock()
		p.Responde(base.Campos{"lugares": fixos(), "favoritos": favoritos(lerFavoritos())})
	})
	s.Ligar("favorito-por", a.favoritoPor)
	s.Ligar("favorito-tirar", a.favoritoTirar)
	s.Ligar("favorito-mover", a.favoritoMover)
	s.Ligar("favorito-renomear", a.favoritoRenomear)
	s.Ligar("discos", a.pedirDiscos)
	s.Ligar("montar", func(p *base.Pedido) { a.udisks(p, "montar") })
	s.Ligar("desmontar", func(p *base.Pedido) { a.udisks(p, "desmontar") })
	s.Ligar("ejetar", func(p *base.Pedido) { a.udisks(p, "ejetar") })
	s.Ligar("vigiar-discos", a.vigiarDiscos)
	s.Ligar("espaco", func(p *base.Pedido) {
		var q struct {
			Caminho string `json:"caminho"`
		}
		_ = p.Ler(&q)
		if err := base.Caminho(q.Caminho); err != nil {
			p.Falha(err.Error())
			return
		}
		livre, total, ok := espaco(q.Caminho)
		if !ok {
			p.Falha("não deu para medir o espaço de «" + q.Caminho + "»")
			return
		}
		p.Responde(base.Campos{"livre": livre, "total": total})
	})
	s.AoFechar(a.pararVigia)
}

// espaco diz quanto há livre (para quem não é root) e no total, em bytes.
func espaco(caminho string) (livre, total int64, ok bool) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(caminho, &st); err != nil {
		return 0, 0, false
	}
	return int64(st.Bavail) * int64(st.Bsize), int64(st.Blocks) * int64(st.Bsize), true
}
