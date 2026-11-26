// Package busca procura dentro de uma pasta, descendo nas de dentro, e vai
// mandando o que acha: por nome (sem diferenciar maiúsculas nem acentos, com
// * e ?), com filtros de tipo, tamanho e data, e dentro dos arquivos de
// texto. Não segue links nem sai do sistema de arquivos sem pedir. Veja
// ../../../CONTRATO.md.
package busca

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unicode"

	"mkfiles/internal/base"
	"mkfiles/internal/item"
	"mkfiles/internal/mime"
)

// Os limites da busca no conteúdo, e de quanto em quanto sai um lote.
const (
	limiteDoConteudo = 8 << 20
	farejoDeBinario  = 8 << 10
	achadosPorLote   = 100
)

var esperaDoLote = 200 * time.Millisecond

// As letras com acento do latim e a letra sem ele. O que não está aqui fica
// como está.
var semAcento = func() map[rune]string {
	grupos := map[string]string{
		"a": "àáâãäåāăą", "c": "çćĉċč", "d": "ďđ", "e": "èéêëēĕėęě", "g": "ĝğġģ", "h": "ĥħ", "i": "ìíîïĩīĭįı",
		"j": "ĵ", "k": "ķ", "l": "ĺļľŀł", "n": "ñńņňŉ", "o": "òóôõöøōŏő", "r": "ŕŗř", "s": "śŝşšſ", "t": "ţťŧ",
		"u": "ùúûüũūŭůűų", "w": "ŵ", "y": "ýÿŷ", "z": "źżž", "ae": "æ", "oe": "œ", "ss": "ß", "th": "þ",
	}
	t := map[rune]string{}
	for sem, com := range grupos {
		for _, r := range com {
			t[r] = sem
		}
	}
	return t
}()

// Normal põe um texto em minúsculas e sem acentos (os acentos soltos, de um
// nome decomposto, somem): é como a busca compara.
func Normal(s string) string {
	simples := true
	for i := 0; i < len(s); i++ {
		if c := s[i]; c >= 0x80 || (c >= 'A' && c <= 'Z') {
			simples = false
			break
		}
	}
	if simples {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r >= 0x0300 && r <= 0x036f {
			continue
		}
		r = unicode.ToLower(r)
		if sem, tem := semAcento[r]; tem {
			b.WriteString(sem)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Casa diz se um nome (já normalizado) casa com um padrão (já normalizado)
// em que * vale qualquer trecho e ? vale um caractere.
func Casa(padrao, nome string) bool {
	p, n := []rune(padrao), []rune(nome)
	ip, in := 0, 0
	estrela, volta := -1, 0
	for in < len(n) {
		switch {
		case ip < len(p) && (p[ip] == '?' || p[ip] == n[in]):
			ip++
			in++
		case ip < len(p) && p[ip] == '*':
			estrela, volta = ip, in
			ip++
		case estrela >= 0:
			volta++
			ip, in = estrela+1, volta
		default:
			return false
		}
	}
	for ip < len(p) && p[ip] == '*' {
		ip++
	}
	return ip == len(p)
}

type pedido struct {
	Caminho      string  `json:"caminho"`
	Nome         string  `json:"nome"`
	Tipo         string  `json:"tipo"`
	TamanhoMin   *int64  `json:"tamanho_min"`
	TamanhoMax   *int64  `json:"tamanho_max"`
	DepoisS      *int64  `json:"depois_s"`
	AntesS       *int64  `json:"antes_s"`
	Conteudo     *string `json:"conteudo"`
	Ocultos      bool    `json:"ocultos"`
	OutrosDiscos bool    `json:"outros_discos"`
}

type filtro struct {
	pedido
	padrao   string // o nome normalizado
	curinga  bool
	conteudo []byte // normalizado; nil se não procura no conteúdo
}

// discoDe diz em que sistema de arquivos uma pasta está (o st_dev). Os testes
// trocam, para simular outro disco.
var discoDe = func(caminho string, info fs.FileInfo) uint64 {
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		return uint64(st.Dev)
	}
	return 0
}

// Ligar põe na sessão o pedido buscar.
func Ligar(s *base.Sessao) {
	s.Ligar("buscar", buscar)
}

func buscar(p *base.Pedido) {
	var f filtro
	_ = p.Ler(&f.pedido)
	if err := base.Caminho(f.Caminho); err != nil {
		p.Falha(err.Error())
		return
	}
	raiz, err := os.Stat(f.Caminho)
	if err == nil && !raiz.IsDir() {
		err = syscall.ENOTDIR
	}
	if err != nil {
		motivo, frase := item.Motivo(err)
		p.Falha(frase, base.Campos{"motivo": motivo})
		return
	}
	switch f.Tipo {
	case "":
		f.Tipo = "qualquer"
	case "qualquer", "pasta", "arquivo", "imagem", "video", "audio", "texto", "documento":
	default:
		p.FalhaDeCampo("tipo", "não conheço o tipo «"+f.Tipo+"»")
		return
	}
	f.padrao = Normal(strings.TrimSpace(f.Nome))
	f.curinga = strings.ContainsAny(f.padrao, "*?")
	if f.Conteudo != nil && *f.Conteudo != "" {
		f.conteudo = []byte(Normal(*f.Conteudo))
	}
	disco := discoDe(f.Caminho, raiz)

	p.Fluxo(func(cancelado func() bool) {
		itens := []item.Item{}
		total, vistos := 0, 0
		ultimo := time.Now()
		solta := func(forcar bool) {
			if len(itens) > 0 && (forcar || len(itens) >= achadosPorLote || time.Since(ultimo) >= esperaDoLote) {
				p.Responde(base.Campos{"itens": itens, "fim": false})
				itens = []item.Item{}
				ultimo = time.Now()
			}
		}
		// Uma pilha de pastas por visitar: a busca desce sem recursão.
		pastas := []string{f.Caminho}
		for len(pastas) > 0 && !cancelado() {
			pasta := pastas[len(pastas)-1]
			pastas = pastas[:len(pastas)-1]
			entradas, err := os.ReadDir(pasta)
			if err != nil {
				continue // sem permissão, ou sumiu no meio
			}
			escondidos := item.Ocultos(pasta)
			var dentro []string
			for _, e := range entradas {
				if cancelado() {
					break
				}
				nome := e.Name()
				oculto := strings.HasPrefix(nome, ".") || escondidos[nome]
				if oculto && !f.Ocultos {
					continue
				}
				info, err := e.Info()
				if err != nil {
					continue
				}
				vistos++
				caminho := filepath.Join(pasta, nome)
				if info.IsDir() { // um link para pasta não é pasta aqui: não é seguido
					if f.OutrosDiscos || discoDe(caminho, info) == disco {
						dentro = append(dentro, caminho)
					}
				}
				if it, serve := f.serve(caminho, nome, info, oculto); serve {
					itens = append(itens, it)
					total++
				}
				solta(false)
			}
			// Na ordem da pasta: as de dentro entram ao contrário na pilha.
			for i := len(dentro) - 1; i >= 0; i-- {
				pastas = append(pastas, dentro[i])
			}
			solta(false)
		}
		fim := base.Campos{"itens": itens, "fim": true, "total": total, "vistos": vistos}
		if cancelado() {
			fim["cancelada"] = true
		}
		p.Responde(fim)
	})
}

// serve diz se um arquivo passa por todos os filtros, e monta o item dele.
// Os filtros baratos vão primeiro; o conteúdo, por último.
func (f *filtro) serve(caminho, nome string, info fs.FileInfo, oculto bool) (item.Item, bool) {
	var nada item.Item
	if f.padrao != "" {
		n := Normal(nome)
		if f.curinga && !Casa(f.padrao, n) || !f.curinga && !strings.Contains(n, f.padrao) {
			return nada, false
		}
	}
	comum := info.Mode().IsRegular()
	if f.TamanhoMin != nil || f.TamanhoMax != nil {
		if !comum || (f.TamanhoMin != nil && info.Size() < *f.TamanhoMin) || (f.TamanhoMax != nil && info.Size() > *f.TamanhoMax) {
			return nada, false
		}
	}
	quando := info.ModTime().Unix()
	if (f.DepoisS != nil && quando < *f.DepoisS) || (f.AntesS != nil && quando > *f.AntesS) {
		return nada, false
	}
	switch f.Tipo {
	case "pasta":
		if !info.IsDir() {
			return nada, false
		}
	case "arquivo":
		if !comum {
			return nada, false
		}
	}
	if f.conteudo != nil && (!comum || info.Size() > limiteDoConteudo) {
		return nada, false
	}
	it := item.DeInfo(caminho, info, oculto, false)
	switch f.Tipo {
	case "imagem", "video", "audio", "texto":
		if mime.Familia(it.Mime) != f.Tipo {
			return nada, false
		}
	case "documento":
		if familia := mime.Familia(it.Mime); familia != "documento" && familia != "texto" {
			return nada, false
		}
	}
	if f.conteudo != nil && !temDentro(caminho, f.conteudo) {
		return nada, false
	}
	return it, true
}

// temDentro procura o texto (normalizado) num arquivo de texto. Um arquivo
// com byte nulo no começo é binário: pulado.
func temDentro(caminho string, texto []byte) bool {
	arq, err := os.Open(caminho)
	if err != nil {
		return false
	}
	defer arq.Close()
	dados, err := io.ReadAll(io.LimitReader(arq, limiteDoConteudo+1))
	if err != nil || len(dados) > limiteDoConteudo {
		return false
	}
	if bytes.IndexByte(dados[:min(len(dados), farejoDeBinario)], 0) >= 0 {
		return false
	}
	return bytes.Contains([]byte(Normal(string(dados))), texto)
}
