// Package props atende a folha de propriedades: o que se sabe de um ou vários
// itens, a contagem do conteúdo das pastas (em fluxo, cancelável, sem seguir
// links) e a mudança de permissões e de grupo. Veja ../../../CONTRATO.md.
package props

import (
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"mkfiles/internal/base"
	"mkfiles/internal/item"
)

// De quanto em quanto tempo a contagem parcial sai (os testes encurtam).
var passoDaContagem = 200 * time.Millisecond

// Ligar põe na sessão os pedidos propriedades e permissoes.
func Ligar(s *base.Sessao) {
	s.Ligar("propriedades", propriedades)
	s.Ligar("permissoes", permissoes)
}

type contagem struct {
	Arquivos int64 `json:"arquivos"`
	Pastas   int64 `json:"pastas"`
	Tamanho  int64 `json:"tamanho"`
}

func propriedades(p *base.Pedido) {
	var q struct {
		Caminhos []string `json:"caminhos"`
	}
	_ = p.Ler(&q)
	if err := base.Caminhos(q.Caminhos); err != nil {
		p.Falha(err.Error())
		return
	}
	itens := make([]item.Item, 0, len(q.Caminhos))
	var onde any = filepath.Dir(q.Caminhos[0])
	souDono := true
	eu := uint32(os.Getuid())
	var conta contagem
	var pastas []string
	for _, c := range q.Caminhos {
		info, err := os.Lstat(c)
		if err != nil {
			motivo, frase := item.Motivo(err)
			if motivo == "nao-existe" {
				frase = "«" + filepath.Base(c) + "» não existe mais"
			}
			p.Falha(frase, base.Campos{"motivo": motivo})
			return
		}
		it, err := item.De(c, true)
		if err != nil {
			p.Falha(err.Error())
			return
		}
		itens = append(itens, it)
		if filepath.Dir(c) != onde {
			onde = nil
		}
		if st, ok := info.Sys().(*syscall.Stat_t); !ok || st.Uid != eu {
			souDono = false
		}
		if info.IsDir() {
			pastas = append(pastas, c)
		} else {
			conta.Arquivos++
			conta.Tamanho += tamanhoDe(info)
		}
	}
	p.Fluxo(func(cancelado func() bool) {
		p.Responde(base.Campos{"itens": itens, "onde": onde, "sou_dono": souDono,
			"grupos": item.GruposDe(eu, uint32(os.Getgid())), "contagem": conta, "fim": false})
		ultima := time.Now()
		parou := false
		for _, pasta := range pastas {
			if parou {
				break
			}
			_ = filepath.WalkDir(pasta, func(c string, d fs.DirEntry, err error) error {
				if cancelado() {
					parou = true
					return filepath.SkipAll
				}
				if err != nil || c == pasta {
					return nil // uma pasta que não abre fica fora da conta
				}
				if d.IsDir() {
					conta.Pastas++
				} else {
					conta.Arquivos++
					if info, err := d.Info(); err == nil {
						conta.Tamanho += tamanhoDe(info)
					}
				}
				if time.Since(ultima) >= passoDaContagem {
					ultima = time.Now()
					p.Responde(base.Campos{"contagem": conta, "fim": false})
				}
				return nil
			})
		}
		fim := base.Campos{"contagem": conta, "fim": true}
		if parou {
			fim["cancelada"] = true
		}
		p.Responde(fim)
	})
}

// tamanhoDe: o de um arquivo comum; link e especial contam zero (o link não é
// seguido).
func tamanhoDe(info fs.FileInfo) int64 {
	if info.Mode().IsRegular() {
		return info.Size()
	}
	return 0
}

// paraModo passa os 12 bits do chmod para o FileMode do Go.
func paraModo(bits uint32) fs.FileMode {
	m := fs.FileMode(bits & 0o777)
	if bits&0o4000 != 0 {
		m |= fs.ModeSetuid
	}
	if bits&0o2000 != 0 {
		m |= fs.ModeSetgid
	}
	if bits&0o1000 != 0 {
		m |= fs.ModeSticky
	}
	return m
}

// comExecutar liga (ou desliga) o x de quem já tem o r.
func comExecutar(bits uint32, liga bool) uint32 {
	if !liga {
		return bits &^ 0o111
	}
	return bits | (bits&0o444)>>2
}

type erroDe struct {
	Caminho string `json:"caminho"`
	Erro    string `json:"erro"`
}

func permissoes(p *base.Pedido) {
	var q struct {
		Caminhos   []string `json:"caminhos"`
		Modo       *int64   `json:"modo"`
		Executavel *bool    `json:"executavel"`
		Grupo      *string  `json:"grupo"`
		Recursivo  bool     `json:"recursivo"`
	}
	if err := p.Ler(&q); err != nil {
		p.Falha("pedido de permissões mal formado")
		return
	}
	if err := base.Caminhos(q.Caminhos); err != nil {
		p.Falha(err.Error())
		return
	}
	if q.Modo != nil && (*q.Modo < 0 || *q.Modo > 0o7777) {
		p.FalhaDeCampo("modo", "modo de permissão inválido")
		return
	}
	if q.Modo == nil && q.Executavel == nil && q.Grupo == nil {
		p.Falha("nada para mudar")
		return
	}
	gid := -1
	if q.Grupo != nil {
		n, ok := item.GidDe(*q.Grupo)
		if !ok {
			p.FalhaDeCampo("grupo", "não conheço o grupo «"+*q.Grupo+"»")
			return
		}
		gid = int(n)
	}
	if q.Recursivo {
		for _, c := range q.Caminhos {
			if err := base.Protegido(c); err != nil {
				p.Recusa(err.Error())
				return
			}
		}
	}
	erros := []erroDe{}
	// muda aplica o pedido a um caminho. dentro: é conteúdo de uma pasta do
	// pedido (as pastas de dentro ganham o x de quem tem o r).
	muda := func(c string, info fs.FileInfo, dentro bool) {
		if info.Mode()&fs.ModeSymlink != 0 {
			if !dentro {
				erros = append(erros, erroDe{c, "é um link: as permissões são as do alvo"})
			}
			return
		}
		var err error
		if gid >= 0 {
			err = os.Lchown(c, -1, gid)
		}
		bits, mudou := item.Modo(info.Mode()), false
		if q.Modo != nil {
			antes := bits
			bits, mudou = uint32(*q.Modo), true
			if dentro && info.IsDir() {
				bits = comExecutar(bits, true)
			} else if dentro {
				// Um arquivo de dentro fica com o x que já tinha: o modo de uma
				// pasta (que tem x) não faz de cada arquivo um executável.
				bits = bits&^0o7111 | antes&0o111
			}
		}
		if q.Executavel != nil && !(dentro && info.IsDir()) {
			bits, mudou = comExecutar(bits, *q.Executavel), true
		}
		if err == nil && mudou {
			err = os.Chmod(c, paraModo(bits))
		}
		if err != nil {
			frase := err.Error()
			if os.IsPermission(err) {
				frase = "você não tem permissão para mudar isso"
			}
			erros = append(erros, erroDe{c, frase})
		}
	}
	for _, c := range q.Caminhos {
		info, err := os.Lstat(c)
		if err != nil {
			_, frase := item.Motivo(err)
			if os.IsNotExist(err) {
				frase = "não existe mais"
			}
			erros = append(erros, erroDe{c, frase})
			continue
		}
		if q.Recursivo && info.IsDir() {
			// O conteúdo primeiro: tirar o x da pasta de cima antes fecharia a
			// porta. O WalkDir não segue links.
			_ = filepath.WalkDir(c, func(dentro string, d fs.DirEntry, err error) error {
				if err != nil || dentro == c {
					return nil
				}
				if i, err := d.Info(); err == nil {
					muda(dentro, i, true)
				}
				return nil
			})
		}
		muda(c, info, false)
	}
	itens := []item.Item{}
	for _, c := range q.Caminhos {
		if it, err := item.De(c, true); err == nil {
			itens = append(itens, it)
		}
	}
	p.Responde(base.Campos{"itens": itens, "erros": erros})
}
