package abrir

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"mkfiles/internal/base"
	"mkfiles/internal/item"
	"mkfiles/internal/lugares"
)

// Ligar põe na sessão os pedidos apps, definir-padrao, abrir, terminal e
// modelos.
func Ligar(s *base.Sessao) {
	s.Ligar("apps", pedirApps)
	s.Ligar("definir-padrao", pedirPadrao)
	s.Ligar("abrir", pedirAbrir)
	s.Ligar("terminal", pedirTerminal)
	s.Ligar("modelos", pedirModelos)
}

func mimeValido(m string) bool {
	tipo, sub, ok := strings.Cut(m, "/")
	return ok && tipo != "" && sub != "" && len(m) <= 200 && !strings.ContainsAny(m, " \t\r\n=;[]\x00")
}

func nulo(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func pedirApps(p *base.Pedido) {
	var q struct {
		Mime  string `json:"mime"`
		Todos bool   `json:"todos"`
	}
	_ = p.Ler(&q)
	if !mimeValido(q.Mime) {
		p.FalhaDeCampo("mime", "tipo de arquivo inválido")
		return
	}
	apps, ordem := osApps()
	padrao, lista := associados(q.Mime, apps, ordem)
	if q.Todos {
		lista = nil
		for _, id := range ordem {
			if apps[id].serve() {
				lista = append(lista, id)
			}
		}
		sort.SliceStable(lista, func(i, j int) bool {
			return strings.ToLower(apps[lista[i]].Nome) < strings.ToLower(apps[lista[j]].Nome)
		})
	}
	saida := []base.Campos{}
	for _, id := range lista {
		a := apps[id]
		saida = append(saida, base.Campos{"id": id, "nome": a.Nome, "icone": nulo(a.Icone), "terminal": a.Terminal, "padrao": id == padrao})
	}
	p.Responde(base.Campos{"mime": q.Mime, "padrao": nulo(padrao), "apps": saida})
}

func pedirPadrao(p *base.Pedido) {
	var q struct {
		Mime string `json:"mime"`
		App  string `json:"app"`
	}
	_ = p.Ler(&q)
	if !mimeValido(q.Mime) {
		p.FalhaDeCampo("mime", "tipo de arquivo inválido")
		return
	}
	apps, _ := osApps()
	if a, existe := apps[q.App]; !existe || !a.serve() || strings.ContainsAny(q.App, ";=\r\n\x00/") {
		p.FalhaDeCampo("app", "esse app não está instalado")
		return
	}
	if err := definirPadrao(q.Mime, q.App); err != nil {
		p.Falha("não deu para gravar o mimeapps.list: " + err.Error())
		return
	}
	p.Responde(base.Campos{"padrao": q.App})
}

func pedirAbrir(p *base.Pedido) {
	var q struct {
		Caminhos []string `json:"caminhos"`
		App      *string  `json:"app"`
	}
	_ = p.Ler(&q)
	if err := base.Caminhos(q.Caminhos); err != nil {
		p.Falha(err.Error())
		return
	}
	apps, ordem := osApps()
	if q.App != nil {
		if a, existe := apps[*q.App]; !existe || !a.serve() {
			p.FalhaDeCampo("app", "esse app não está instalado")
			return
		}
	}
	// Os arquivos de cada app, na ordem em que o app apareceu.
	var fila []string
	deCada := map[string][]string{}
	padraoDe := map[string]string{}
	semApp := []base.Campos{}
	for _, c := range q.Caminhos {
		it, err := item.De(c, true)
		if err != nil {
			_, frase := item.Motivo(err)
			p.Falha("«" + filepath.Base(c) + "»: " + strings.Replace(frase, "essa pasta", "esse arquivo", 1))
			return
		}
		id := ""
		if q.App != nil {
			id = *q.App
		} else {
			var visto bool
			if id, visto = padraoDe[it.Mime]; !visto {
				id, _ = associados(it.Mime, apps, ordem)
				padraoDe[it.Mime] = id
			}
		}
		if id == "" {
			semApp = append(semApp, base.Campos{"caminho": c, "mime": it.Mime})
			continue
		}
		if _, ja := deCada[id]; !ja {
			fila = append(fila, id)
		}
		deCada[id] = append(deCada[id], c)
	}
	lancar := []base.Campos{}
	for _, id := range fila {
		a := apps[id]
		linhas, err := argvs(a, deCada[id])
		if err != nil {
			p.Falha("o atalho de «" + a.Nome + "» está quebrado: " + err.Error())
			return
		}
		for _, argv := range linhas {
			entrada := base.Campos{"app": id, "nome": a.Nome, "argv": argv}
			if a.Terminal {
				if t := terminal(); t != "" {
					entrada["argv"] = linhaDoTerminal(t, a.Nome, "", argv)
				} else {
					entrada["aviso"] = semTerminal
				}
			}
			lancar = append(lancar, entrada)
		}
	}
	p.Responde(base.Campos{"lancar": lancar, "sem_app": semApp})
}

func pedirTerminal(p *base.Pedido) {
	var q struct {
		Caminho string `json:"caminho"`
	}
	_ = p.Ler(&q)
	if err := base.Caminho(q.Caminho); err != nil {
		p.Falha(err.Error())
		return
	}
	pasta := q.Caminho
	info, err := os.Stat(pasta)
	if err != nil {
		_, frase := item.Motivo(err)
		p.Falha(frase)
		return
	}
	if !info.IsDir() {
		pasta = filepath.Dir(pasta)
	}
	t := terminal()
	if t == "" {
		p.Falha(semTerminal)
		return
	}
	p.Responde(base.Campos{"argv": linhaDoTerminal(t, "", pasta, nil)})
}

func pedirModelos(p *base.Pedido) {
	pasta := lugares.PastaDeModelos()
	modelos := []item.Item{}
	entradas, err := os.ReadDir(pasta)
	if pasta == "" || err != nil {
		p.Responde(base.Campos{"pasta": nil, "modelos": modelos})
		return
	}
	ocultos := item.Ocultos(pasta)
	for _, e := range entradas {
		nome := e.Name()
		if strings.HasPrefix(nome, ".") || ocultos[nome] {
			continue
		}
		c := filepath.Join(pasta, nome)
		if info, err := os.Stat(c); err != nil || !info.Mode().IsRegular() {
			continue
		}
		if it, err := item.De(c, true); err == nil {
			modelos = append(modelos, it)
		}
	}
	p.Responde(base.Campos{"pasta": pasta, "modelos": modelos})
}
