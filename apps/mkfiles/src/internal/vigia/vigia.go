// Package vigia avisa a janela do que muda nas pastas abertas, por inotify
// (pacote syscall, sem nada de fora). As mudanças de cada pasta saem juntas,
// no máximo um evento "pasta" a cada 150 ms. Veja ../../../CONTRATO.md.
package vigia

import (
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"mkfiles/internal/base"
	"mkfiles/internal/item"
)

// Os números do contrato (os testes encurtam o intervalo).
var (
	Intervalo     = 150 * time.Millisecond
	LimiteDoLote  = 2000
	oQueInteressa = uint32(syscall.IN_CREATE | syscall.IN_DELETE | syscall.IN_MOVED_FROM | syscall.IN_MOVED_TO |
		syscall.IN_CLOSE_WRITE | syscall.IN_MODIFY | syscall.IN_ATTRIB | syscall.IN_DELETE_SELF | syscall.IN_MOVE_SELF |
		syscall.IN_ONLYDIR)
)

// pasta é uma pasta vigiada e o que mudou nela desde o último evento.
type pasta struct {
	caminho    string
	wd         int32
	refs       int
	mudou      map[string]bool
	saiu       map[string]bool
	sumiu      bool
	recarregar bool
	marcado    bool // já há um evento a caminho
	// Num remoto (gvfs) o inotify não vê nada: a pasta é relida de tempos em
	// tempos. para fecha quando ela deixa de ser vigiada.
	para chan struct{}
}

type vigia struct {
	s       *base.Sessao
	trava   sync.Mutex
	arquivo *os.File // o descritor do inotify
	fechado bool
	pastas  map[string]*pasta
	porWd   map[int32][]*pasta
}

// Ligar põe na sessão os pedidos vigiar e desvigiar.
func Ligar(s *base.Sessao) {
	v := &vigia{s: s, pastas: map[string]*pasta{}, porWd: map[int32][]*pasta{}}
	s.Ligar("vigiar", v.vigiar)
	s.Ligar("desvigiar", v.desvigiar)
	s.AoFechar(v.fecha)
}

func (v *vigia) fecha() {
	v.trava.Lock()
	v.fechado = true
	arq := v.arquivo
	v.arquivo = nil
	for _, ps := range v.pastas {
		if ps.para != nil {
			close(ps.para)
			ps.para = nil
		}
	}
	v.pastas, v.porWd = map[string]*pasta{}, map[int32][]*pasta{}
	v.trava.Unlock()
	if arq != nil {
		arq.Close() // solta a leitura
	}
}

// abre cria o inotify na primeira pasta vigiada. Quem chama tem a trava.
func (v *vigia) abre() error {
	if v.arquivo != nil {
		return nil
	}
	if v.fechado {
		return errors.New("o motor está fechando")
	}
	// Sem bloquear: assim o descritor entra no vigia de rede do Go e fechar o
	// arquivo acorda a leitura.
	fd, err := syscall.InotifyInit1(syscall.IN_CLOEXEC | syscall.IN_NONBLOCK)
	if err != nil {
		return err
	}
	v.arquivo = os.NewFile(uintptr(fd), "inotify")
	go v.le(v.arquivo)
	return nil
}

func (v *vigia) vigiar(p *base.Pedido) {
	var q struct {
		Caminho string `json:"caminho"`
	}
	_ = p.Ler(&q)
	if err := base.Caminho(q.Caminho); err != nil {
		p.Falha(err.Error())
		return
	}
	v.trava.Lock()
	defer v.trava.Unlock()
	if ja := v.pastas[q.Caminho]; ja != nil {
		ja.refs++
		p.Responde(base.Campos{"vigiando": true})
		return
	}
	if _, _, remoto := base.Remoto(q.Caminho); remoto {
		nova := &pasta{caminho: q.Caminho, wd: semInotify, refs: 1, mudou: map[string]bool{}, saiu: map[string]bool{}, para: make(chan struct{})}
		v.pastas[q.Caminho] = nova
		go v.rele(nova)
		p.Responde(base.Campos{"vigiando": true})
		return
	}
	if err := v.abre(); err != nil {
		p.Falha("não deu para vigiar pastas: " + err.Error())
		return
	}
	wd, err := syscall.InotifyAddWatch(int(v.arquivo.Fd()), q.Caminho, oQueInteressa)
	if err != nil {
		frase := "não deu para vigiar essa pasta: " + err.Error()
		switch {
		case errors.Is(err, syscall.ENOSPC):
			frase = "o sistema chegou ao limite de pastas vigiadas"
		case errors.Is(err, syscall.ENOENT):
			frase = "essa pasta não existe mais"
		case errors.Is(err, syscall.ENOTDIR):
			frase = "isso não é uma pasta"
		case errors.Is(err, syscall.EACCES):
			frase = "você não tem permissão para abrir essa pasta"
		}
		p.Falha(frase)
		return
	}
	nova := &pasta{caminho: q.Caminho, wd: int32(wd), refs: 1, mudou: map[string]bool{}, saiu: map[string]bool{}}
	v.pastas[q.Caminho] = nova
	v.porWd[nova.wd] = append(v.porWd[nova.wd], nova)
	p.Responde(base.Campos{"vigiando": true})
}

func (v *vigia) desvigiar(p *base.Pedido) {
	var q struct {
		Caminho string `json:"caminho"`
	}
	_ = p.Ler(&q)
	if err := base.Caminho(q.Caminho); err != nil {
		p.Falha(err.Error())
		return
	}
	v.trava.Lock()
	defer v.trava.Unlock()
	if ja := v.pastas[q.Caminho]; ja != nil {
		ja.refs--
		if ja.refs > 0 {
			p.Responde(base.Campos{"vigiando": true})
			return
		}
		v.solta(ja)
	}
	p.Responde(base.Campos{"vigiando": false})
}

// solta deixa de vigiar uma pasta. Quem chama tem a trava.
func (v *vigia) solta(ps *pasta) {
	delete(v.pastas, ps.caminho)
	if ps.para != nil {
		close(ps.para)
		ps.para = nil
		return
	}
	resto := v.porWd[ps.wd][:0]
	for _, outra := range v.porWd[ps.wd] {
		if outra != ps {
			resto = append(resto, outra)
		}
	}
	if len(resto) > 0 {
		v.porWd[ps.wd] = resto // o mesmo lugar por outro caminho (um link)
		return
	}
	delete(v.porWd, ps.wd)
	if v.arquivo != nil && !ps.sumiu {
		_, _ = syscall.InotifyRmWatch(int(v.arquivo.Fd()), uint32(ps.wd))
	}
}

// le é a goroutine que recebe os eventos do kernel até o arquivo fechar.
func (v *vigia) le(arq *os.File) {
	buf := make([]byte, 64<<10)
	for {
		n, err := arq.Read(buf)
		if err != nil || n < syscall.SizeofInotifyEvent {
			return
		}
		v.trava.Lock()
		for i := 0; i+syscall.SizeofInotifyEvent <= n; {
			wd := int32(binary.NativeEndian.Uint32(buf[i:]))
			mascara := binary.NativeEndian.Uint32(buf[i+4:])
			tamanho := int(binary.NativeEndian.Uint32(buf[i+12:]))
			inicio := i + syscall.SizeofInotifyEvent
			if inicio+tamanho > n {
				break
			}
			nome := buf[inicio : inicio+tamanho]
			for len(nome) > 0 && nome[len(nome)-1] == 0 {
				nome = nome[:len(nome)-1]
			}
			v.anota(wd, mascara, string(nome))
			i = inicio + tamanho
		}
		v.trava.Unlock()
	}
}

// anota guarda um evento do kernel na pasta dele e marca a saída do lote.
// Quem chama tem a trava.
func (v *vigia) anota(wd int32, mascara uint32, nome string) {
	alvos := v.porWd[wd]
	if mascara&syscall.IN_Q_OVERFLOW != 0 {
		// A fila estourou: não se sabe o que se perdeu, em pasta nenhuma.
		alvos = alvos[:0]
		for _, ps := range v.pastas {
			ps.recarregar = true
			alvos = append(alvos, ps)
		}
	}
	for _, ps := range alvos {
		switch {
		case mascara&syscall.IN_Q_OVERFLOW != 0:
		case mascara&(syscall.IN_DELETE_SELF|syscall.IN_MOVE_SELF|syscall.IN_UNMOUNT|syscall.IN_IGNORED) != 0:
			ps.sumiu = true
		case nome == "":
			continue // mudou a própria pasta (permissão, data): nada a listar
		case nome == ".hidden":
			ps.recarregar = true // muda quem é oculto na pasta inteira
		case mascara&(syscall.IN_DELETE|syscall.IN_MOVED_FROM) != 0:
			ps.saiu[nome] = true
			delete(ps.mudou, nome)
		default:
			ps.mudou[nome] = true
			delete(ps.saiu, nome)
		}
		if !ps.marcado {
			ps.marcado = true
			deQuem := ps
			time.AfterFunc(Intervalo, func() { v.soltaLote(deQuem) })
		}
	}
}

// soltaLote manda o evento "pasta" com tudo que juntou.
func (v *vigia) soltaLote(ps *pasta) {
	v.trava.Lock()
	if v.fechado || v.pastas[ps.caminho] != ps {
		v.trava.Unlock()
		return
	}
	mudou, saiu := ps.mudou, ps.saiu
	sumiu, recarregar := ps.sumiu, ps.recarregar
	ps.mudou, ps.saiu, ps.recarregar, ps.marcado = map[string]bool{}, map[string]bool{}, false, false
	if sumiu {
		v.solta(ps) // não há mais o que vigiar ali
	}
	v.trava.Unlock()

	itens := []item.Item{}
	nomes := []string{}
	if len(mudou)+len(saiu) > LimiteDoLote {
		recarregar = true
	}
	if !recarregar && !sumiu {
		for _, nome := range ordenados(mudou) {
			it, err := item.De(filepath.Join(ps.caminho, nome), false)
			if err != nil {
				saiu[nome] = true // já não está mais lá
				continue
			}
			itens = append(itens, it)
		}
		nomes = append(nomes, ordenados(saiu)...)
	}
	v.s.Evento("pasta", base.Campos{"caminho": ps.caminho, "mudou": itens, "saiu": nomes, "sumiu": sumiu, "recarregar": recarregar})
}

func ordenados(m map[string]bool) []string {
	lista := make([]string, 0, len(m))
	for n := range m {
		lista = append(lista, n)
	}
	sort.Strings(lista)
	return lista
}

// semInotify é o "wd" de uma pasta que não é vigiada pelo kernel.
const semInotify = -1

// Releitura é de quanto em quanto tempo uma pasta remota é relida
// (MKFILES_RELER_MS; padrão 5 s).
func Releitura() time.Duration {
	if n, err := strconv.Atoi(os.Getenv("MKFILES_RELER_MS")); err == nil && n >= 10 {
		return time.Duration(n) * time.Millisecond
	}
	return 5 * time.Second
}

// retrato resume uma pasta: os nomes com tamanho e data. existe é false se
// ela não abre mais.
func retrato(caminho string) (resumo string, existe bool) {
	entradas, err := os.ReadDir(caminho)
	if err != nil {
		return "", false
	}
	var b strings.Builder
	for _, e := range entradas {
		b.WriteString(e.Name())
		if info, err := e.Info(); err == nil {
			b.WriteString("\x00" + strconv.FormatInt(info.Size(), 10) + "\x00" + strconv.FormatInt(info.ModTime().UnixNano(), 10) + "\x00" + info.Mode().String())
		}
		b.WriteByte('\n')
	}
	return b.String(), true
}

// rele é a reserva de onde o inotify não chega (o FUSE do gvfs): relê a pasta
// de tempos em tempos e, se algo mudou, pede à janela que liste de novo.
func (v *vigia) rele(ps *pasta) {
	v.trava.Lock()
	para := ps.para
	v.trava.Unlock()
	if para == nil {
		return
	}
	antes, _ := retrato(ps.caminho)
	relogio := time.NewTicker(Releitura())
	defer relogio.Stop()
	for {
		select {
		case <-para:
			return
		case <-relogio.C:
		}
		agora, existe := retrato(ps.caminho)
		if existe && agora == antes {
			continue
		}
		antes = agora
		v.trava.Lock()
		vale := !v.fechado && v.pastas[ps.caminho] == ps
		if vale && !existe {
			v.solta(ps) // não há mais o que vigiar ali
		}
		v.trava.Unlock()
		if !vale {
			return
		}
		v.s.Evento("pasta", base.Campos{"caminho": ps.caminho, "mudou": []item.Item{}, "saiu": []string{}, "sumiu": !existe, "recarregar": existe})
		if !existe {
			return
		}
	}
}
