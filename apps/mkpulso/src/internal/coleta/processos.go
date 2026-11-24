package coleta

import (
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"mkpulso/internal/base"
)

// LimiteCmd é o tamanho máximo, em bytes, da linha de comando que sai no
// retrato: um navegador tem centenas de processos com linhas de 4 kB.
const LimiteCmd = 300

// A marca de thread do kernel no campo flags de /proc/<pid>/stat.
const pfKthread = 0x00200000

// Processo é a leitura crua de um /proc/<pid>, com os contadores como estão.
type Processo struct {
	PID, PPID   int
	Nome        string // o comm, completado pelo executável quando veio cortado
	Cmd         string // a linha de comando, com espaços
	UID         int    // -1 se não deu para ler
	Usuario     string
	Estado      string // já em português, como no contrato
	Ticks       uint64 // utime + stime
	Threads     int
	Nice        int
	InicioTicks uint64 // desde o boot
	MemKB       int64  // VmRSS
	Leitura     uint64 // bytes, de /proc/<pid>/io
	Escrita     uint64
	TemIO       bool   // o io costuma ser ilegível para processo de outro usuário
	Cgroup      string // o caminho da linha "0::" (cgroup v2)
	Kernel      bool   // thread do kernel
}

type statProc struct {
	comm      string
	estado    string
	ppid      int
	bandeiras uint64
	ticks     uint64
	nice      int
	threads   int
	inicio    uint64
}

// analisaStat lê a linha de /proc/<pid>/stat. O comm fica entre o primeiro
// "(" e o último ")", porque o próprio nome pode ter espaços e parênteses.
func analisaStat(texto string) (statProc, bool) {
	var s statProc
	abre := strings.IndexByte(texto, '(')
	fecha := strings.LastIndexByte(texto, ')')
	if abre < 0 || fecha < abre {
		return s, false
	}
	s.comm = texto[abre+1 : fecha]
	c := strings.Fields(texto[fecha+1:])
	if len(c) < 20 {
		return s, false
	}
	s.estado = c[0]
	ppid, err := strconv.Atoi(c[1])
	if err != nil {
		return s, false
	}
	s.ppid = ppid
	s.bandeiras = numero(c[6])
	s.ticks = numero(c[11]) + numero(c[12])
	s.nice, _ = strconv.Atoi(c[16])
	s.threads, _ = strconv.Atoi(c[17])
	s.inicio = numero(c[19])
	return s, true
}

func estadoEmPortugues(letra string) string {
	switch letra {
	case "R":
		return "rodando"
	case "D":
		return "esperando disco"
	case "T", "t":
		return "parado"
	case "Z", "X", "x":
		return "zumbi"
	case "I":
		return "ocioso"
	}
	return "dormindo"
}

// Usuarios lê /etc/passwd: uid → nome. Linha torta é pulada.
func Usuarios() map[int]string {
	nomes := map[int]string{}
	b, err := os.ReadFile(base.Etc("passwd"))
	if err != nil {
		return nomes
	}
	for _, linha := range strings.Split(string(b), "\n") {
		c := strings.Split(linha, ":")
		if len(c) < 3 || c[0] == "" {
			continue
		}
		uid, err := strconv.Atoi(c[2])
		if err != nil {
			continue
		}
		if _, ja := nomes[uid]; !ja {
			nomes[uid] = c[0]
		}
	}
	return nomes
}

// LerProcessos lê todos os processos de /proc, em ordem de pid. Um processo
// que some no meio da leitura (ou cujo stat veio torto) fica de fora, sem
// erro.
func LerProcessos() []Processo {
	entradas, err := os.ReadDir(base.Proc())
	if err != nil {
		return nil
	}
	usuarios := Usuarios()
	ps := make([]Processo, 0, len(entradas))
	for _, e := range entradas {
		pid, err := strconv.Atoi(e.Name())
		if err != nil || pid <= 0 {
			continue
		}
		if p, ok := lerProcesso(pid, usuarios); ok {
			ps = append(ps, p)
		}
	}
	sort.Slice(ps, func(i, j int) bool { return ps[i].PID < ps[j].PID })
	return ps
}

func lerProcesso(pid int, usuarios map[int]string) (Processo, bool) {
	p := Processo{PID: pid, UID: -1}
	n := strconv.Itoa(pid)
	bruto, err := os.ReadFile(base.Proc(n, "stat"))
	if err != nil {
		return p, false
	}
	st, ok := analisaStat(string(bruto))
	if !ok {
		return p, false
	}
	status, err := os.ReadFile(base.Proc(n, "status"))
	if err != nil {
		return p, false
	}
	temRSS := false
	for _, linha := range strings.Split(string(status), "\n") {
		chave, valor, ok := strings.Cut(linha, ":")
		if !ok {
			continue
		}
		c := strings.Fields(valor)
		if len(c) == 0 {
			continue
		}
		switch chave {
		case "Uid":
			if uid, err := strconv.Atoi(c[0]); err == nil && uid >= 0 {
				p.UID = uid
			}
		case "VmRSS":
			if kb, err := strconv.ParseInt(c[0], 10, 64); err == nil && kb >= 0 {
				p.MemKB = kb
				temRSS = true
			}
		}
	}
	if p.UID >= 0 {
		if nome, ok := usuarios[p.UID]; ok {
			p.Usuario = nome
		} else {
			p.Usuario = strconv.Itoa(p.UID)
		}
	}

	p.PPID = st.ppid
	p.Estado = estadoEmPortugues(st.estado)
	p.Ticks = st.ticks
	p.Threads = st.threads
	p.Nice = st.nice
	p.InicioTicks = st.inicio

	cmdline, _ := os.ReadFile(base.Proc(n, "cmdline"))
	args := strings.Split(strings.TrimRight(string(cmdline), "\x00"), "\x00")
	semCmd := len(args) == 1 && strings.TrimSpace(args[0]) == ""

	p.Kernel = pid == 2 || st.ppid == 2 || st.bandeiras&pfKthread != 0 ||
		(semCmd && st.estado != "Z" && !temRSS)

	p.Nome = strings.ToValidUTF8(st.comm, "�")
	if !semCmd && len(st.comm) == 15 {
		// o kernel corta o comm em 15 letras; se o executável começa
		// igual, o nome inteiro é o dele
		if exe := path.Base(strings.TrimLeft(args[0], "-")); strings.HasPrefix(exe, st.comm) && utf8.ValidString(exe) {
			p.Nome = exe
		}
	}
	switch {
	case !semCmd:
		p.Cmd = cortaCmd(strings.Join(args, " "))
	case p.Kernel:
		p.Cmd = "[" + p.Nome + "]"
	default:
		p.Cmd = p.Nome
	}

	if io, err := os.ReadFile(base.Proc(n, "io")); err == nil {
		for _, linha := range strings.Split(string(io), "\n") {
			chave, valor, ok := strings.Cut(linha, ":")
			if !ok {
				continue
			}
			switch chave {
			case "read_bytes":
				p.Leitura = numero(strings.TrimSpace(valor))
				p.TemIO = true
			case "write_bytes":
				p.Escrita = numero(strings.TrimSpace(valor))
				p.TemIO = true
			}
		}
	}

	if cg, err := os.ReadFile(base.Proc(n, "cgroup")); err == nil {
		for _, linha := range strings.Split(string(cg), "\n") {
			if resto, ok := strings.CutPrefix(linha, "0::"); ok {
				p.Cgroup = strings.TrimSpace(resto)
				break
			}
		}
	}
	return p, true
}

// cortaCmd limpa a linha de comando (bytes que não são UTF-8, caracteres de
// controle) e corta no LimiteCmd, sem partir uma letra no meio.
func cortaCmd(s string) string {
	s = strings.ToValidUTF8(s, "�")
	s = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return ' '
		}
		return r
	}, s)
	s = strings.TrimSpace(s)
	if len(s) <= LimiteCmd {
		return s
	}
	corte := LimiteCmd
	for corte > 0 && !utf8.RuneStart(s[corte]) {
		corte--
	}
	return s[:corte] + "…"
}
