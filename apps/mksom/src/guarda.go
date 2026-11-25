package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"mksom/internal/base"
	"mksom/internal/eq"
)

// guardado é o estado.json: o que o equalizador tem de valores, se a pessoa o
// deixou ligado e qual era a saída padrão antes de ligar.
type guardado struct {
	Ligado        bool      `json:"ligado"`
	Predefinicao  string    `json:"predefinicao"`
	Ganhos        []float64 `json:"ganhos"`
	Pre           float64   `json:"pre"`
	SaidaAnterior string    `json:"saida_anterior"`
}

func arquivoDoEstado() string { return filepath.Join(base.PastaDoApp(), "estado.json") }
func arquivoDasPredefinicoes() string {
	return filepath.Join(base.PastaDoApp(), "predefinicoes.json")
}
func arquivoDoConf() string    { return filepath.Join(base.PastaDoApp(), "eq.conf") }
func arquivoDaUnidade() string { return filepath.Join(base.PastaDasUnidades(), eq.NomeDaUnidade) }

func (g guardado) ajuste() eq.Ajuste {
	return eq.Ajuste{Ganhos: append([]float64{}, g.Ganhos...), Pre: g.Pre}
}

// leEstado lê o estado.json. Ausente, ilegível ou com valores fora do
// contrato: plano (o resto do que deu para ler continua valendo).
func leEstado() guardado {
	var g guardado
	if b, err := os.ReadFile(arquivoDoEstado()); err == nil {
		_ = json.Unmarshal(b, &g)
	}
	if g.ajuste().Valida() != nil {
		plano := eq.Plano()
		g.Ganhos, g.Pre, g.Predefinicao = plano.Ganhos, plano.Pre, ""
	}
	if !eq.AlvoSeguro(g.SaidaAnterior) {
		g.SaidaAnterior = ""
	}
	return g
}

func gravaJSON(caminho string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return base.EscreveAtomico(caminho, append(b, '\n'))
}

func gravaEstado(g guardado) error {
	return gravaJSON(arquivoDoEstado(), g)
}

// predefinicaoGuardada é uma predefinição da pessoa no predefinicoes.json.
type predefinicaoGuardada struct {
	Nome   string    `json:"nome"`
	Ganhos []float64 `json:"ganhos"`
	Pre    float64   `json:"pre"`
}

// lePredefinicoes lê as predefinições da pessoa. Entrada torta (nome
// inválido, de fábrica ou repetido, valores fora do contrato) é pulada.
func lePredefinicoes() []eq.Predefinicao {
	var brutas []json.RawMessage
	b, err := os.ReadFile(arquivoDasPredefinicoes())
	if err != nil || json.Unmarshal(b, &brutas) != nil {
		return nil
	}
	var out []eq.Predefinicao
	for _, bruta := range brutas {
		var p predefinicaoGuardada
		if json.Unmarshal(bruta, &p) != nil {
			continue
		}
		nome, err := eq.ValidaNome(p.Nome)
		if err != nil || eq.DeFabrica(nome) {
			continue
		}
		if _, repetida := eq.Acha(out, nome); repetida {
			continue
		}
		if (eq.Ajuste{Ganhos: p.Ganhos, Pre: p.Pre}).Valida() != nil {
			continue
		}
		out = append(out, eq.Predefinicao{Nome: nome, Ganhos: p.Ganhos, Pre: p.Pre})
		if len(out) == eq.PredefinicoesMax {
			break
		}
	}
	return out
}

func gravaPredefinicoes(lista []eq.Predefinicao) error {
	out := make([]predefinicaoGuardada, 0, len(lista))
	for _, p := range lista {
		out = append(out, predefinicaoGuardada{p.Nome, p.Ganhos, p.Pre})
	}
	return gravaJSON(arquivoDasPredefinicoes(), out)
}

// gravaConf reescreve o eq.conf inteiro.
func gravaConf(a eq.Ajuste, alvo string) error {
	return base.EscreveAtomico(arquivoDoConf(), []byte(eq.Conf(a, alvo)))
}

// existe diz se o arquivo está lá.
func existe(caminho string) bool {
	_, err := os.Stat(caminho)
	return err == nil
}

// mesmoNome compara nomes de predefinição sem olhar maiúsculas.
func mesmoNome(a, b string) bool {
	return strings.EqualFold(a, b)
}
