// Package prefs guarda as preferências (~/.config/mk/mkfiles/preferencias.json)
// e o estado da janela (~/.local/state/mk/mkfiles/estado.json). Tudo que entra
// é conferido: chave desconhecida ou valor fora do lugar é ignorado, e um
// arquivo corrompido volta ao padrão. Veja ../../../CONTRATO.md.
package prefs

import (
	"encoding/json"
	"os"
	"sort"
	"sync"

	"mkfiles/internal/base"
)

// Os limites do contrato.
const (
	LimiteDePastas = 500 // visões lembradas por pasta
	LimiteDeAbas   = 30  // abas por painel
	// Lixeira é o "caminho" da aba que mostra a lixeira (não é uma pasta).
	Lixeira = "lixeira:"
)

type ordem struct {
	Campo   string `json:"campo"`
	Inversa bool   `json:"inversa"`
}

// preferencias é o arquivo de preferências. A ordem das pastas lembradas (a
// mais antiga primeiro) fica numa chave à parte, que a janela não vê.
type preferencias struct {
	Visao            string            `json:"visao"`
	VisaoPorPasta    map[string]string `json:"visao_por_pasta"`
	Ordem            ordem             `json:"ordem"`
	Ocultos          bool              `json:"ocultos"`
	IconePx          int               `json:"icone_px"`
	PastasAntes      bool              `json:"pastas_antes"`
	ConfirmarLixeira bool              `json:"confirmar_lixeira"`
	Clique           string            `json:"clique"`
	Colunas          map[string]int    `json:"colunas"`
	// As duas de baixo só são guardadas: quem aplica é a janela.
	MiniaturasRemotas bool `json:"miniaturas_remotas"`
	MiniaturaRemotaMB int  `json:"miniatura_remota_mb"`
	idade             []string
}

type painel struct {
	Abas  []string `json:"abas"`
	Atual int      `json:"atual"`
}

type estado struct {
	Paineis   []painel `json:"paineis"`
	Ativo     int      `json:"ativo"`
	Duplo     bool     `json:"duplo"`
	LateralPx int      `json:"lateral_px"`
}

func padrao() *preferencias {
	return &preferencias{Visao: "icones", VisaoPorPasta: map[string]string{}, Ordem: ordem{Campo: "nome"}, IconePx: 64,
		PastasAntes: true, Clique: "duplo", Colunas: map[string]int{"tamanho": 90, "tipo": 150, "mtime": 150, "permissoes": 100},
		MiniaturasRemotas: true, MiniaturaRemotaMB: 5}
}

func estadoPadrao() *estado {
	return &estado{Paineis: []painel{}, LateralPx: 200}
}

func um(valor string, de ...string) bool {
	for _, v := range de {
		if v == valor {
			return true
		}
	}
	return false
}

func visaoValida(v string) bool { return um(v, "icones", "lista", "compacta") }

// le decodifica um valor só se ele tem exatamente o tipo esperado.
func le[T any](cru json.RawMessage, serve func(T) bool, poe func(T)) {
	var v T
	if string(cru) == "null" || json.Unmarshal(cru, &v) != nil {
		return
	}
	if serve == nil || serve(v) {
		poe(v)
	}
}

// aplica põe em p o que há de válido em campos; o resto fica como estava.
func (p *preferencias) aplica(campos map[string]json.RawMessage) {
	le(campos["visao"], visaoValida, func(v string) { p.Visao = v })
	le(campos["ocultos"], nil, func(v bool) { p.Ocultos = v })
	le(campos["pastas_antes"], nil, func(v bool) { p.PastasAntes = v })
	le(campos["confirmar_lixeira"], nil, func(v bool) { p.ConfirmarLixeira = v })
	le(campos["miniaturas_remotas"], nil, func(v bool) { p.MiniaturasRemotas = v })
	le(campos["miniatura_remota_mb"], func(v int) bool { return v >= 1 && v <= 100 }, func(v int) { p.MiniaturaRemotaMB = v })
	le(campos["clique"], func(v string) bool { return um(v, "duplo", "simples") }, func(v string) { p.Clique = v })
	le(campos["icone_px"], func(v int) bool { return v >= 32 && v <= 256 }, func(v int) { p.IconePx = v })
	le(campos["ordem"], nil, func(o map[string]json.RawMessage) {
		le(o["campo"], func(v string) bool { return um(v, "nome", "tamanho", "tipo", "mtime") }, func(v string) { p.Ordem.Campo = v })
		le(o["inversa"], nil, func(v bool) { p.Ordem.Inversa = v })
	})
	le(campos["colunas"], nil, func(c map[string]json.RawMessage) {
		for _, coluna := range []string{"tamanho", "tipo", "mtime", "permissoes"} {
			le(c[coluna], func(v int) bool { return v >= 40 && v <= 600 }, func(v int) { p.Colunas[coluna] = v })
		}
	})
	// As pastas na ordem em que foram lembradas (só o arquivo traz).
	var idade []string
	le(campos["visao_por_pasta_ordem"], nil, func(v []string) { idade = v })
	le(campos["visao_por_pasta"], nil, func(pastas map[string]json.RawMessage) {
		vistas := map[string]bool{}
		lembra := func(pasta string) {
			cru, tem := pastas[pasta]
			if !tem || vistas[pasta] || base.Caminho(pasta) != nil {
				return
			}
			vistas[pasta] = true
			var visao string
			if json.Unmarshal(cru, &visao) != nil || !visaoValida(visao) {
				// null (ou um valor que não serve) esquece a pasta.
				if string(cru) == "null" {
					p.esquece(pasta)
				}
				return
			}
			p.esquece(pasta)
			p.VisaoPorPasta[pasta] = visao
			p.idade = append(p.idade, pasta)
		}
		for _, pasta := range idade {
			lembra(pasta)
		}
		for _, pasta := range chaves(pastas) {
			lembra(pasta)
		}
	})
	for len(p.idade) > LimiteDePastas {
		delete(p.VisaoPorPasta, p.idade[0])
		p.idade = p.idade[1:]
	}
}

func (p *preferencias) esquece(pasta string) {
	if _, tem := p.VisaoPorPasta[pasta]; !tem {
		return
	}
	delete(p.VisaoPorPasta, pasta)
	for i, c := range p.idade {
		if c == pasta {
			p.idade = append(p.idade[:i], p.idade[i+1:]...)
			break
		}
	}
}

// chaves devolve as chaves em ordem alfabética: a ordem em que vieram escritas
// o mapa já perdeu, e assim ao menos é sempre a mesma.
func chaves(m map[string]json.RawMessage) []string {
	lista := make([]string, 0, len(m))
	for c := range m {
		lista = append(lista, c)
	}
	sort.Strings(lista)
	return lista
}

func (e *estado) aplica(campos map[string]json.RawMessage) {
	le(campos["duplo"], nil, func(v bool) { e.Duplo = v })
	le(campos["ativo"], func(v int) bool { return v == 0 || v == 1 }, func(v int) { e.Ativo = v })
	le(campos["lateral_px"], func(v int) bool { return v >= 120 && v <= 480 }, func(v int) { e.LateralPx = v })
	le(campos["paineis"], func(v []json.RawMessage) bool { return len(v) <= 2 }, func(crus []json.RawMessage) {
		paineis := []painel{}
		for _, cru := range crus {
			var lido struct {
				Abas  []any `json:"abas"`
				Atual any   `json:"atual"`
			}
			if json.Unmarshal(cru, &lido) != nil {
				return // um painel torto: a lista inteira é ignorada
			}
			p := painel{Abas: []string{}}
			for _, a := range lido.Abas {
				if c, ok := a.(string); ok && (c == Lixeira || base.Caminho(c) == nil) && len(p.Abas) < LimiteDeAbas {
					p.Abas = append(p.Abas, c)
				}
			}
			if n, ok := lido.Atual.(float64); ok && n == float64(int(n)) && int(n) >= 0 && int(n) < len(p.Abas) {
				p.Atual = int(n)
			}
			paineis = append(paineis, p)
		}
		e.Paineis = paineis
	})
}

func arquivoDePrefs() string  { return base.Config("mk", "mkfiles", "preferencias.json") }
func arquivoDeEstado() string { return base.Estado("mk", "mkfiles", "estado.json") }

func lerCampos(arquivo string) map[string]json.RawMessage {
	b, err := os.ReadFile(arquivo)
	if err != nil || len(b) > 4<<20 {
		return nil
	}
	var campos map[string]json.RawMessage
	if json.Unmarshal(b, &campos) != nil {
		return nil
	}
	return campos
}

func lerPrefs() *preferencias {
	p := padrao()
	p.aplica(lerCampos(arquivoDePrefs()))
	return p
}

func lerEstado() *estado {
	e := estadoPadrao()
	e.aplica(lerCampos(arquivoDeEstado()))
	return e
}

func grava(arquivo string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return base.GravaAtomico(arquivo, append(b, '\n'), 0o644)
}

func (p *preferencias) grava() error {
	// O arquivo leva também a idade das pastas lembradas.
	type cru preferencias
	return grava(arquivoDePrefs(), struct {
		*cru
		Idade []string `json:"visao_por_pasta_ordem"`
	}{(*cru)(p), append([]string{}, p.idade...)})
}

// Ligar põe na sessão os pedidos prefs, prefs-gravar e estado-gravar.
func Ligar(s *base.Sessao) {
	var trava sync.Mutex
	s.Ligar("prefs", func(p *base.Pedido) {
		trava.Lock()
		defer trava.Unlock()
		p.Responde(base.Campos{"prefs": lerPrefs(), "estado": lerEstado()})
	})
	s.Ligar("prefs-gravar", func(p *base.Pedido) {
		var q struct {
			Prefs map[string]json.RawMessage `json:"prefs"`
		}
		_ = p.Ler(&q)
		trava.Lock()
		defer trava.Unlock()
		prefs := lerPrefs()
		delete(q.Prefs, "visao_por_pasta_ordem") // é do arquivo, não da janela
		prefs.aplica(q.Prefs)
		if err := prefs.grava(); err != nil {
			p.Falha("não deu para gravar as preferências: " + err.Error())
			return
		}
		p.Responde(base.Campos{"prefs": prefs})
	})
	s.Ligar("estado-gravar", func(p *base.Pedido) {
		var q struct {
			Estado map[string]json.RawMessage `json:"estado"`
		}
		_ = p.Ler(&q)
		trava.Lock()
		defer trava.Unlock()
		e := lerEstado()
		e.aplica(q.Estado)
		if err := grava(arquivoDeEstado(), e); err != nil {
			p.Falha("não deu para gravar o estado: " + err.Error())
			return
		}
		p.Responde(base.Campos{"estado": e})
	})
}
