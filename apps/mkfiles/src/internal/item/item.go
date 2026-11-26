// Package item monta o "item" do contrato (o que a janela sabe de cada
// arquivo) e atende os pedidos listar e item.
package item

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"mkfiles/internal/base"
	"mkfiles/internal/mime"
)

// Item é um arquivo, pasta, link ou especial, como o contrato descreve.
type Item struct {
	Nome         string  `json:"nome"`
	Caminho      string  `json:"caminho"`
	Tipo         string  `json:"tipo"`
	Tamanho      int64   `json:"tamanho"`
	Mtime        int64   `json:"mtime"`
	Modo         uint32  `json:"modo"`
	Permissoes   string  `json:"permissoes"`
	Dono         string  `json:"dono"`
	Grupo        string  `json:"grupo"`
	Oculto       bool    `json:"oculto"`
	Mime         string  `json:"mime"`
	Icone        string  `json:"icone"`
	IconeReserva string  `json:"icone_reserva"`
	Alvo         *string `json:"alvo"`
	Quebrado     bool    `json:"quebrado"`
	AlvoTipo     *string `json:"alvo_tipo"`
}

// Os tipos de item.
const (
	Pasta    = "pasta"
	Arquivo  = "arquivo"
	Link     = "link"
	Especial = "especial"
)

// Modo são os 12 bits de permissão (rwx de dono, grupo e outros, mais setuid,
// setgid e sticky), como o chmod os entende.
func Modo(m fs.FileMode) uint32 {
	bits := uint32(m.Perm())
	if m&fs.ModeSetuid != 0 {
		bits |= 0o4000
	}
	if m&fs.ModeSetgid != 0 {
		bits |= 0o2000
	}
	if m&fs.ModeSticky != 0 {
		bits |= 0o1000
	}
	return bits
}

// Permissoes escreve os 12 bits como as 9 letras do ls ("rwxr-xr-x").
func Permissoes(bits uint32) string {
	const letras = "rwxrwxrwx"
	b := []byte(letras)
	for i := range b {
		if bits&(1<<uint(8-i)) == 0 {
			b[i] = '-'
		}
	}
	especial := func(bit uint32, pos int, com, sem byte) {
		if bits&bit == 0 {
			return
		}
		if b[pos] == 'x' {
			b[pos] = com
		} else {
			b[pos] = sem
		}
	}
	especial(0o4000, 2, 's', 'S')
	especial(0o2000, 5, 's', 'S')
	especial(0o1000, 8, 't', 'T')
	return string(b)
}

func tipoDe(m fs.FileMode) (tipo, mimeDoTipo string) {
	switch {
	case m.IsDir():
		return Pasta, mime.Pasta
	case m&fs.ModeSymlink != 0:
		return Link, mime.Link
	case m&fs.ModeSocket != 0:
		return Especial, "inode/socket"
	case m&fs.ModeNamedPipe != 0:
		return Especial, "inode/fifo"
	case m&fs.ModeCharDevice != 0:
		return Especial, "inode/chardevice"
	case m&fs.ModeDevice != 0:
		return Especial, "inode/blockdevice"
	case m.IsRegular():
		return Arquivo, ""
	}
	return Especial, mime.Binario
}

// DeInfo monta o item de um caminho de que já se tem o Lstat. oculto vem de
// quem lista (o nome com ponto ou o .hidden da pasta); sempre manda farejar o
// conteúdo mesmo de um arquivo com extensão.
func DeInfo(caminho string, info fs.FileInfo, oculto, sempre bool) Item {
	it := Item{Nome: filepath.Base(caminho), Caminho: caminho, Oculto: oculto,
		Tamanho: info.Size(), Mtime: info.ModTime().Unix(), Modo: Modo(info.Mode())}
	it.Permissoes = Permissoes(it.Modo)
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		it.Dono, it.Grupo = nomeDoDono(st.Uid), nomeDoGrupo(st.Gid)
	}
	it.Tipo, it.Mime = tipoDe(info.Mode())
	doOutroLado := info
	if it.Tipo == Link {
		alvo, _ := os.Readlink(caminho)
		it.Alvo = &alvo
		la, err := os.Stat(caminho)
		if err != nil {
			it.Quebrado = true
			it.Tamanho = 0
		} else {
			doOutroLado = la
			t, m := tipoDe(la.Mode())
			it.AlvoTipo = &t
			it.Tamanho = la.Size()
			it.Mime = m
		}
	} else if it.Tipo == Pasta {
		it.Tamanho = 0
	}
	if it.Mime == "" {
		it.Mime = mime.DoArquivo(caminho, doOutroLado.Size(), sempre)
	}
	it.Icone, it.IconeReserva = mime.Icone(it.Mime)
	return it
}

// De monta o item de um caminho (com o farejo de conteúdo, se sempre).
func De(caminho string, sempre bool) (Item, error) {
	info, err := os.Lstat(caminho)
	if err != nil {
		return Item{}, err
	}
	nome := filepath.Base(caminho)
	oculto := strings.HasPrefix(nome, ".") || Ocultos(filepath.Dir(caminho))[nome]
	return DeInfo(caminho, info, oculto, sempre), nil
}

// Ocultos lê o arquivo .hidden de uma pasta: um nome por linha.
func Ocultos(pasta string) map[string]bool {
	b, err := os.ReadFile(filepath.Join(pasta, ".hidden"))
	if err != nil || len(b) > 1<<20 {
		return nil
	}
	nomes := map[string]bool{}
	for _, linha := range strings.Split(string(b), "\n") {
		if n := strings.TrimSpace(linha); n != "" {
			nomes[n] = true
		}
	}
	return nomes
}

// Os nomes de quem é dono: /etc/passwd e /etc/group (ou MKFILES_ETC), lidos
// uma vez.
var (
	travaDosNomes sync.Mutex
	donos, grupos map[string]string
	nomesDe       string
)

func tabela(arquivo string) map[string]string {
	t := map[string]string{}
	b, err := os.ReadFile(arquivo)
	if err != nil {
		return t
	}
	for _, linha := range strings.Split(string(b), "\n") {
		campos := strings.Split(linha, ":")
		if len(campos) >= 3 && campos[0] != "" && !strings.HasPrefix(campos[0], "#") {
			if _, ja := t[campos[2]]; !ja {
				t[campos[2]] = campos[0]
			}
		}
	}
	return t
}

func nomes() (map[string]string, map[string]string) {
	etc := base.Raiz("MKFILES_ETC", "/etc")
	travaDosNomes.Lock()
	defer travaDosNomes.Unlock()
	if donos == nil || nomesDe != etc {
		donos, grupos, nomesDe = tabela(filepath.Join(etc, "passwd")), tabela(filepath.Join(etc, "group")), etc
	}
	return donos, grupos
}

func nomeDoDono(uid uint32) string {
	d, _ := nomes()
	n := strconv.FormatUint(uint64(uid), 10)
	if nome := d[n]; nome != "" {
		return nome
	}
	return n
}

func nomeDoGrupo(gid uint32) string {
	_, g := nomes()
	n := strconv.FormatUint(uint64(gid), 10)
	if nome := g[n]; nome != "" {
		return nome
	}
	return n
}

// GruposDe diz os nomes dos grupos de que um usuário faz parte (o principal e
// os de /etc/group), para a folha de permissões.
func GruposDe(uid, gidPrincipal uint32) []string {
	etc := base.Raiz("MKFILES_ETC", "/etc")
	usuario := nomeDoDono(uid)
	lista := []string{nomeDoGrupo(gidPrincipal)}
	b, _ := os.ReadFile(filepath.Join(etc, "group"))
	for _, linha := range strings.Split(string(b), "\n") {
		campos := strings.Split(linha, ":")
		if len(campos) < 4 {
			continue
		}
		for _, membro := range strings.Split(campos[3], ",") {
			if membro == usuario && campos[0] != lista[0] {
				lista = append(lista, campos[0])
			}
		}
	}
	return lista
}

// GidDe acha o número de um grupo pelo nome.
func GidDe(nome string) (uint32, bool) {
	_, g := nomes()
	for numero, n := range g {
		if n == nome {
			if v, err := strconv.ParseUint(numero, 10, 32); err == nil {
				return uint32(v), true
			}
		}
	}
	return 0, false
}

// Motivo traduz um erro de disco no "motivo" do contrato e numa frase.
func Motivo(err error) (motivo, frase string) {
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return "nao-existe", "essa pasta não existe mais"
	case errors.Is(err, fs.ErrPermission):
		return "sem-permissao", "você não tem permissão para abrir essa pasta"
	case errors.Is(err, syscall.ENOTDIR):
		return "nao-e-pasta", "isso não é uma pasta"
	}
	return "erro", err.Error()
}

// Lote é quantos itens vão em cada linha de um fluxo (MKFILES_LOTE).
func Lote() int {
	if n, err := strconv.Atoi(os.Getenv("MKFILES_LOTE")); err == nil && n > 0 && n <= 5000 {
		return n
	}
	return 500
}

// Ligar põe na sessão os pedidos listar e item.
func Ligar(s *base.Sessao) {
	s.Ligar("listar", listar)
	s.Ligar("item", func(p *base.Pedido) {
		var q struct {
			Caminho string `json:"caminho"`
		}
		_ = p.Ler(&q)
		if err := base.Caminho(q.Caminho); err != nil {
			p.Falha(err.Error())
			return
		}
		// Por fora e com prazo: num remoto lento, a janela não espera para
		// sempre nem os outros pedidos ficam na fila.
		p.S.Vai(func() {
			it, err, noPrazo := comPrazo(p.Parou(), func() (Item, error) { return De(q.Caminho, true) })
			if !noPrazo {
				p.Falha(fraseDoPrazo(q.Caminho), base.Campos{"motivo": "tempo"})
				return
			}
			if err != nil {
				motivo, frase := MotivoEm(q.Caminho, err)
				if motivo == "nao-existe" {
					frase = "isso não existe mais"
				}
				p.Falha(frase, base.Campos{"motivo": motivo})
				return
			}
			p.Responde(base.Campos{"item": it})
		})
	})
}

func listar(p *base.Pedido) {
	var q struct {
		Caminho string `json:"caminho"`
	}
	_ = p.Ler(&q)
	if err := base.Caminho(q.Caminho); err != nil {
		p.Falha(err.Error(), base.Campos{"motivo": "erro"})
		return
	}
	p.Fluxo(func(cancelado func() bool) {
		// Abrir é por fora e com prazo: num remoto que não responde, a janela
		// recebe um erro em vez de esperar para sempre.
		pasta, err, noPrazo := comPrazo(p.Parou(), func() (*os.File, error) { return abrePasta(q.Caminho) })
		if !noPrazo {
			if cancelado() {
				p.Responde(base.Campos{"itens": []Item{}, "fim": true, "cancelada": true, "total": 0})
				return
			}
			p.Falha(fraseDoPrazo(q.Caminho), base.Campos{"motivo": "tempo"})
			return
		}
		if err != nil {
			motivo, frase := MotivoEm(q.Caminho, err)
			p.Falha(frase, base.Campos{"motivo": motivo})
			return
		}
		defer pasta.Close()
		ocultos := Ocultos(q.Caminho)
		lote, total := Lote(), 0
		// Um lote fica na mão até se saber se há mais: assim a última linha
		// (a de "fim": true) leva os últimos itens, e não uma lista vazia.
		var naMao []Item
		for {
			entradas, err := pasta.ReadDir(lote)
			itens := make([]Item, 0, len(entradas))
			for _, e := range entradas {
				info, erro := e.Info()
				if erro != nil {
					continue // sumiu entre o ReadDir e o Lstat
				}
				nome := e.Name()
				itens = append(itens, DeInfo(filepath.Join(q.Caminho, nome), info,
					strings.HasPrefix(nome, ".") || ocultos[nome], false))
			}
			total += len(itens)
			if cancelado() {
				p.Responde(base.Campos{"itens": []Item{}, "fim": true, "cancelada": true, "total": total})
				return
			}
			acabou := err != nil || len(entradas) == 0
			if naMao != nil && (!acabou || len(naMao)+len(itens) > lote) {
				p.Responde(base.Campos{"itens": naMao, "fim": false})
				naMao = nil
			}
			if !acabou {
				naMao = itens
				continue
			}
			fim := base.Campos{"itens": append(naMao, itens...), "fim": true, "total": total,
				"gravavel": syscall.Access(q.Caminho, 2) == nil}
			if err != nil && err != io.EOF {
				fim["aviso"] = "a pasta não foi lida até o fim: " + err.Error()
			}
			p.Responde(fim)
			return
		}
	})
}
