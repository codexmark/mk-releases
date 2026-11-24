// Package grupos junta os processos por identidade: é o coração do MkPulso.
// A pessoa não quer ver 327 processos, quer ver "Firefox", "pipewire" e "o
// resto do sistema". Aqui também moram a regra da energia (energia.go), as
// médias de 1 e 3 minutos (medias.go) e a leitura dos atalhos de menu
// (atalhos.go).
//
// # A regra de agrupamento
//
// Quem diz de quem é um processo é o cgroup dele (a linha "0::" de
// /proc/<pid>/cgroup), porque é ali que o systemd anota a unidade. Na ordem:
//
//  1. Thread do kernel → o grupo "sistema:kernel".
//
//  2. Dentro de user@<uid>.service (o systemd do usuário), vale a primeira
//     unidade do caminho depois das fatias (.slice):
//
//     - app-<nome>-<algo>.scope, app-<nome>@<algo>.service ou
//     app-<nome>.service → o app "app:<nome>". Os escapes do systemd
//     (\x2d é um hífen) são desfeitos. Se o lançador pôs um prefixo
//     (app-gnome-<nome>-…, app-flatpak-<nome>-…), fica o maior final do
//     nome que tem atalho de menu; sem atalho nenhum, o nome inteiro.
//     - init.scope, ou nenhuma unidade e nenhuma fatia → é o próprio
//     systemd do usuário: "servico:user@<uid>.service", escopo "sistema".
//     - qualquer outra unidade → "servico:<unidade>", escopo "usuario".
//
//  3. Fora de user@<uid>.service, a primeira unidade depois das fatias:
//
//     - session-N.scope (um login num terminal ou por ssh, onde tudo o que a
//     pessoa abre cai no mesmo cgroup) → pelo dono, veja 4.
//     - qualquer outra (.service, .scope, .mount, .socket, .swap) →
//     "servico:<unidade>", escopo "sistema".
//
//  4. O que sobra (sessão, um cgroup que só tem fatias, um cgroup que não é
//     do systemd) é agrupado pelo dono: sobe-se pelos pais enquanto o pai
//     está no mesmo cgroup, e o topo da subida dá o nome ao grupo,
//     "app:<executável>". Assim o vim aberto num terminal fica com o
//     terminal, e dois terminais iguais são um grupo só. Esses grupos não têm
//     unidade nem escopo.
//
//  5. Sem cgroup, ou no cgroup raiz → "sistema:outros".
//
// O tipo: "sistema" para os dois grupos de sistema; "app" se o grupo tem
// janela, ou se tem nome de app (regras 2 e 4) com atalho de menu; senão
// "servico". O id não muda quando uma janela aparece: só o tipo.
//
// O id "servico:<unidade>" pode se repetir: o dbus-broker.service do sistema
// e o do usuário são grupos separados, que só o escopo distingue.
//
// O processo principal (pid) é o mais antigo do grupo; a unidade e o escopo
// do grupo são os dele. O nome e o ícone vêm do atalho (.desktop) do nome do
// app ou, na falta, da classe de uma janela do grupo.
package grupos

import (
	"regexp"
	"sort"
	"strings"

	"mkpulso/internal/coleta"
	"mkpulso/internal/janelas"
)

// Processo é um item de grupos[].processos do retrato.
type Processo struct {
	PID      int      `json:"pid"`
	PPID     int      `json:"ppid"`
	Nome     string   `json:"nome"`
	Cmd      string   `json:"cmd"`
	Usuario  string   `json:"usuario"`
	Estado   string   `json:"estado"`
	CPUPct   float64  `json:"cpu_pct"`
	MemKB    int64    `json:"mem_kb"`
	DiscoBps *float64 `json:"disco_bps"`
	GPUPct   *float64 `json:"gpu_pct"`
	Threads  int      `json:"threads"`
	Nice     int      `json:"nice"`
	InicioS  *int64   `json:"inicio_s"`
}

// Media são as médias de 1 e 3 minutos de um grupo.
type Media struct {
	CPUPct1m   float64 `json:"cpu_pct_1m"`
	CPUPct3m   float64 `json:"cpu_pct_3m"`
	MemKB1m    int64   `json:"mem_kb_1m"`
	MemKB3m    int64   `json:"mem_kb_3m"`
	DiscoBps1m float64 `json:"disco_bps_1m"`
	DiscoBps3m float64 `json:"disco_bps_3m"`
}

// Grupo é um item de "grupos" do retrato.
type Grupo struct {
	ID        string           `json:"id"`
	Tipo      string           `json:"tipo"`
	Nome      string           `json:"nome"`
	Icone     string           `json:"icone"`
	Unidade   string           `json:"unidade"`
	Escopo    string           `json:"escopo"`
	Janelas   []janelas.Janela `json:"janelas"`
	N         int              `json:"n"`
	PID       int              `json:"pid"`
	CPUPct    float64          `json:"cpu_pct"`
	MemKB     int64            `json:"mem_kb"`
	DiscoBps  *float64         `json:"disco_bps"`
	GPUPct    *float64         `json:"gpu_pct"`
	Energia   string           `json:"energia"`
	Media     Media            `json:"media"`
	Processos []Processo       `json:"processos"`
}

// Identidade é o que o cgroup diz de um processo.
type Identidade struct {
	ID       string // "" quando PeloDono
	NomeApp  string // o <nome> de um app-<nome>-…; "" se não é app
	Unidade  string // como o systemd escreve (com os escapes)
	Escopo   string // "usuario", "sistema" ou ""
	PeloDono bool   // o cgroup não diz: agrupar pelo ancestral (regra 4)
}

var (
	doUsuario = regexp.MustCompile(`^user@\d+\.service$`)
	deSessao  = regexp.MustCompile(`^session-[^/]*\.scope$`)
)

func eUnidade(s string) bool {
	for _, sufixo := range []string{".service", ".scope", ".mount", ".socket", ".swap"} {
		if strings.HasSuffix(s, sufixo) && len(s) > len(sufixo) {
			return true
		}
	}
	return false
}

// Identificar aplica as regras 1, 2, 3 e 5 a um caminho de cgroup. temAtalho
// diz se um nome de app tem atalho de menu (pode ser nil: nenhum tem).
func Identificar(cgroup string, kernel bool, temAtalho func(string) bool) Identidade {
	if kernel {
		return Identidade{ID: "sistema:kernel"}
	}
	var partes []string
	for _, p := range strings.Split(cgroup, "/") {
		if p != "" {
			partes = append(partes, p)
		}
	}
	if len(partes) == 0 {
		return Identidade{ID: "sistema:outros"}
	}
	escopo := "sistema"
	gerente := ""
	for i, p := range partes {
		if doUsuario.MatchString(p) {
			escopo, gerente = "usuario", p
			partes = partes[i+1:]
			break
		}
	}
	unidade := ""
	for _, p := range partes {
		if strings.HasSuffix(p, ".slice") {
			continue
		}
		if eUnidade(p) {
			unidade = p
		}
		break
	}
	if gerente != "" {
		if unidade == "init.scope" || len(partes) == 0 {
			return Identidade{ID: "servico:" + gerente, Unidade: gerente, Escopo: "sistema"}
		}
		if nome := nomeDoApp(unidade, temAtalho); nome != "" {
			return Identidade{ID: "app:" + nome, NomeApp: nome, Unidade: unidade, Escopo: escopo}
		}
	}
	if unidade == "" || (gerente == "" && deSessao.MatchString(unidade)) {
		return Identidade{PeloDono: true}
	}
	return Identidade{ID: "servico:" + unidade, Unidade: unidade, Escopo: escopo}
}

// nomeDoApp tira o <nome> de app-<nome>-<algo>.scope, app-<nome>@<algo>.service
// ou app-<nome>.service; "" se a unidade não é de app.
func nomeDoApp(unidade string, temAtalho func(string) bool) string {
	resto, ok := strings.CutPrefix(unidade, "app-")
	if !ok {
		return ""
	}
	var pedacos []string
	if miolo, ok := strings.CutSuffix(resto, ".scope"); ok {
		pedacos = strings.Split(miolo, "-")
		if len(pedacos) > 1 {
			pedacos = pedacos[:len(pedacos)-1] // o <algo> do fim
		}
	} else if miolo, ok := strings.CutSuffix(resto, ".service"); ok {
		miolo, _, _ = strings.Cut(miolo, "@")
		pedacos = strings.Split(miolo, "-")
	} else {
		return ""
	}
	inteiro := semEscape(strings.Join(pedacos, "-"))
	if strings.Trim(inteiro, "- ") == "" {
		return ""
	}
	if temAtalho != nil {
		for i := range pedacos {
			if candidato := semEscape(strings.Join(pedacos[i:], "-")); candidato != "" && temAtalho(candidato) {
				return candidato
			}
		}
	}
	return inteiro
}

// semEscape desfaz os \xNN que o systemd põe no nome das unidades. Um
// escape torto fica como veio.
func semEscape(s string) string {
	if !strings.Contains(s, `\x`) {
		return s
	}
	hexa := func(c byte) int {
		switch {
		case c >= '0' && c <= '9':
			return int(c - '0')
		case c >= 'a' && c <= 'f':
			return int(c-'a') + 10
		case c >= 'A' && c <= 'F':
			return int(c-'A') + 10
		}
		return -1
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+3 < len(s) && s[i+1] == 'x' {
			if alto, baixo := hexa(s[i+2]), hexa(s[i+3]); alto >= 0 && baixo >= 0 {
				b.WriteByte(byte(alto*16 + baixo))
				i += 3
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return strings.ToValidUTF8(b.String(), "�")
}

// dono sobe pelos pais de um processo enquanto o pai está no mesmo cgroup
// (regra 4) e devolve o topo da subida.
func dono(m *coleta.Medido, porPID map[int]*coleta.Medido) *coleta.Medido {
	visto := map[int]bool{m.PID: true}
	atual := m
	for {
		pai, ok := porPID[atual.PPID]
		if !ok || pai.PID <= 1 || pai.Kernel || pai.Cgroup != m.Cgroup || visto[pai.PID] {
			return atual
		}
		visto[pai.PID] = true
		atual = pai
	}
}

type montagem struct {
	grupo     *Grupo
	nomeApp   string
	principal *coleta.Medido
	idents    map[int]Identidade
}

// Agrupar junta os processos medidos em grupos, liga cada janela ao grupo do
// processo dono, calcula as somas e a energia, e ordena por CPU decrescente
// (empate: id). As médias saem iguais ao valor de agora: quem tem histórico
// (o watch) passa depois por Historico.Registrar. Nunca devolve nil.
func Agrupar(ms []coleta.Medido, js []janelas.Janela, nucleos int, atalhos *Atalhos) []Grupo {
	temAtalho := func(nome string) bool {
		_, ok := atalhos.Procurar(nome)
		return ok
	}
	porPID := make(map[int]*coleta.Medido, len(ms))
	for i := range ms {
		porPID[ms[i].PID] = &ms[i]
	}
	porCgroup := map[string]Identidade{}
	montagens := map[string]*montagem{}
	grupoDoPID := map[int]*montagem{}
	var ordem []*montagem

	for i := range ms {
		m := &ms[i]
		var ident Identidade
		if m.Kernel {
			ident = Identificar("", true, nil)
		} else if guardada, ok := porCgroup[m.Cgroup]; ok {
			ident = guardada
		} else {
			ident = Identificar(m.Cgroup, false, temAtalho)
			porCgroup[m.Cgroup] = ident
		}
		if ident.PeloDono {
			if nome := dono(m, porPID).Nome; nome != "" {
				ident = Identidade{ID: "app:" + nome, NomeApp: nome}
			} else {
				ident = Identidade{ID: "sistema:outros"}
			}
		}
		chave := ident.ID
		if ident.NomeApp == "" {
			chave += "|" + ident.Escopo
		}
		mt, ok := montagens[chave]
		if !ok {
			mt = &montagem{grupo: &Grupo{ID: ident.ID, Janelas: []janelas.Janela{}}, nomeApp: ident.NomeApp, idents: map[int]Identidade{}}
			montagens[chave] = mt
			ordem = append(ordem, mt)
		}
		mt.idents[m.PID] = ident
		grupoDoPID[m.PID] = mt
		g := mt.grupo
		g.Processos = append(g.Processos, Processo{PID: m.PID, PPID: m.PPID, Nome: m.Nome, Cmd: m.Cmd, Usuario: m.Usuario, Estado: m.Estado,
			CPUPct: m.CPUPct, MemKB: m.MemKB, DiscoBps: m.DiscoBps, GPUPct: m.GPUPct, Threads: m.Threads, Nice: m.Nice, InicioS: m.InicioS})
		g.CPUPct += m.CPUPct
		g.MemKB += m.MemKB
		if m.DiscoBps != nil {
			if g.DiscoBps == nil {
				g.DiscoBps = coleta.Ptr(0.0)
			}
			*g.DiscoBps += *m.DiscoBps
		}
		if m.GPUPct != nil {
			if g.GPUPct == nil {
				g.GPUPct = coleta.Ptr(0.0)
			}
			*g.GPUPct += *m.GPUPct
		}
		if p := mt.principal; p == nil || m.InicioTicks < p.InicioTicks || (m.InicioTicks == p.InicioTicks && m.PID < p.PID) {
			mt.principal = m
		}
	}

	for _, j := range js {
		if mt, ok := grupoDoPID[j.PID]; ok {
			mt.grupo.Janelas = append(mt.grupo.Janelas, j)
		}
	}

	gs := make([]Grupo, 0, len(ordem))
	for _, mt := range ordem {
		g := mt.grupo
		ident := mt.idents[mt.principal.PID]
		g.N = len(g.Processos)
		g.PID = mt.principal.PID
		g.Unidade, g.Escopo = ident.Unidade, ident.Escopo
		g.CPUPct = coleta.Arred(g.CPUPct, 1)
		if g.GPUPct != nil && *g.GPUPct > 100 {
			*g.GPUPct = 100
		}
		nomeia(mt, atalhos)
		g.Energia = Energia(g.CPUPct, nucleos, g.DiscoBps, g.GPUPct)
		g.Media = Media{CPUPct1m: g.CPUPct, CPUPct3m: g.CPUPct, MemKB1m: g.MemKB, MemKB3m: g.MemKB}
		if g.DiscoBps != nil {
			g.Media.DiscoBps1m, g.Media.DiscoBps3m = *g.DiscoBps, *g.DiscoBps
		}
		sort.SliceStable(g.Processos, func(i, j int) bool {
			if g.Processos[i].CPUPct != g.Processos[j].CPUPct {
				return g.Processos[i].CPUPct > g.Processos[j].CPUPct
			}
			return g.Processos[i].PID < g.Processos[j].PID
		})
		gs = append(gs, *g)
	}
	sort.SliceStable(gs, func(i, j int) bool {
		if gs[i].CPUPct != gs[j].CPUPct {
			return gs[i].CPUPct > gs[j].CPUPct
		}
		if gs[i].ID != gs[j].ID {
			return gs[i].ID < gs[j].ID
		}
		return gs[i].Escopo < gs[j].Escopo
	})
	return gs
}

// nomeia decide o tipo, o nome e o ícone de um grupo já montado.
func nomeia(mt *montagem, atalhos *Atalhos) {
	g := mt.grupo
	switch g.ID {
	case "sistema:kernel":
		g.Tipo, g.Nome = "sistema", "Kernel"
		return
	case "sistema:outros":
		g.Tipo, g.Nome = "sistema", "Outros processos"
		return
	}
	g.Tipo, g.Nome = "servico", g.Unidade
	var atalho Atalho
	achou := false
	if mt.nomeApp != "" {
		g.Nome = mt.nomeApp
		atalho, achou = atalhos.Procurar(mt.nomeApp)
		if achou {
			g.Tipo = "app"
		}
	}
	if len(g.Janelas) > 0 {
		g.Tipo = "app"
		for i := 0; i < len(g.Janelas) && !achou; i++ {
			atalho, achou = atalhos.Procurar(g.Janelas[i].Classe)
		}
	}
	if achou {
		g.Nome, g.Icone = atalho.Nome, atalho.Icone
	}
}

// Agora lê a máquina uma vez e devolve os grupos, sem taxas (a CPU e o disco
// saem zerados): serve para quem só precisa saber quem está em qual grupo.
func Agora() []Grupo {
	a := coleta.Colher(0)
	return Agrupar(coleta.MedirProcessos(nil, a), janelas.Listar(), a.Nucleos(), NovosAtalhos())
}

// PIDs devolve os processos dos grupos com esse id, em ordem de pid. Com
// escopo vazio vale qualquer escopo; com "usuario" ou "sistema", só o grupo
// daquele escopo (o mesmo "servico:<unidade>" pode existir nos dois).
func PIDs(gs []Grupo, id, escopo string) []int {
	var pids []int
	for _, g := range gs {
		if g.ID != id || (escopo != "" && g.Escopo != escopo) {
			continue
		}
		for _, p := range g.Processos {
			pids = append(pids, p.PID)
		}
	}
	sort.Ints(pids)
	return pids
}
