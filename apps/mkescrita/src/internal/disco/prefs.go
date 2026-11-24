package disco

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"mkescrita/internal/base"
)

// Prefs são as preferências lembradas entre sessões.
type Prefs struct {
	// Fonte: a família escolhida ("" = a monoespaçada do tema).
	Fonte string `json:"fonte"`
	// Zoom: o tamanho do texto, em porcento (50 a 300).
	Zoom int `json:"zoom"`
	// Previa: a pré-visualização fica aberta.
	Previa bool `json:"previa"`
}

// Os limites do zoom.
const (
	ZoomMinimo = 50
	ZoomMaximo = 300
)

func padrao() Prefs { return Prefs{Fonte: "", Zoom: 100, Previa: false} }

func (p *Prefs) acerta() {
	if p.Zoom < ZoomMinimo {
		p.Zoom = ZoomMinimo
	}
	if p.Zoom > ZoomMaximo {
		p.Zoom = ZoomMaximo
	}
	limpa := strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, p.Fonte))
	if letras := []rune(limpa); len(letras) > 120 {
		limpa = string(letras[:120])
	}
	p.Fonte = limpa
}

// parcial é o que pode vir num `preferencias gravar`: só o que mudou.
type parcial struct {
	Fonte  *string  `json:"fonte"`
	Zoom   *float64 `json:"zoom"`
	Previa *bool    `json:"previa"`
}

func (p *Prefs) aplica(dados []byte) error {
	var m parcial
	dec := json.NewDecoder(bytes.NewReader(dados))
	if err := dec.Decode(&m); err != nil {
		return err
	}
	if m.Fonte != nil {
		p.Fonte = *m.Fonte
	}
	if m.Zoom != nil {
		p.Zoom = int(*m.Zoom + 0.5)
	}
	if m.Previa != nil {
		p.Previa = *m.Previa
	}
	p.acerta()
	return nil
}

// LerPrefs lê as preferências. Arquivo ausente ou estragado: os padrões (um
// arquivo de preferências ruim nunca impede ninguém de escrever).
func LerPrefs() Prefs {
	p := padrao()
	dados, err := os.ReadFile(base.Preferencias())
	if err != nil {
		return p
	}
	lidas := p
	if lidas.aplica(dados) != nil {
		return p
	}
	return lidas
}

// GravarPrefs junta `mudancas` (um objeto JSON, só com o que mudou) às
// preferências que já existem, acerta os limites e grava.
func GravarPrefs(mudancas []byte) (Prefs, error) {
	p := LerPrefs()
	if err := p.aplica(mudancas); err != nil {
		return p, fmt.Errorf("preferências inválidas: espero um objeto JSON com fonte, zoom e previa")
	}
	// Um link simbólico (as preferências num repositório de dotfiles) continua link.
	caminho, err := destinoReal(base.Preferencias())
	if err != nil {
		return p, err
	}
	if err := os.MkdirAll(filepath.Dir(caminho), 0o755); err != nil {
		return p, fmt.Errorf("não consegui criar a pasta das preferências: %v", frase(err))
	}
	dados, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return p, err
	}
	return p, gravaAtomico(caminho, append(dados, '\n'), 0o644, nil)
}
