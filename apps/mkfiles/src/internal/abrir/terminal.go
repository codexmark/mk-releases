package abrir

import (
	"encoding/json"
	"os"
	"os/exec"

	"mkfiles/internal/base"
)

// Os terminais que o Mk conhece, na ordem de preferência.
var terminais = []string{"ghostty", "kitty", "foot", "alacritty", "wezterm"}

const semTerminal = "não achei um terminal para abrir (ghostty, kitty, foot, alacritty ou wezterm)"

// terminal escolhe o terminal: o de MKFILES_TERMINAL, o dos Ajustes do Mk
// ("apps.terminal") ou o primeiro que houver. Só entram os que o Mk conhece.
func terminal() string {
	var ajustes map[string]any
	if b, err := os.ReadFile(base.Config("mk", "ajustes.json")); err == nil {
		_ = json.Unmarshal(b, &ajustes)
	}
	doAjuste, _ := ajustes["apps.terminal"].(string)
	for _, pedido := range append([]string{os.Getenv("MKFILES_TERMINAL"), doAjuste}, terminais...) {
		for _, t := range terminais {
			if t != pedido {
				continue
			}
			if _, err := exec.LookPath(t); err == nil {
				return t
			}
		}
	}
	return ""
}

// linhaDoTerminal monta o argv de cada terminal: como ele recebe o título, a
// pasta de trabalho e o comando. O comando vai como lista, um argumento por
// item, e os cinco terminais o executam direto, sem shell. Sem comando, abre o
// shell da pessoa na pasta.
func linhaDoTerminal(nome, titulo, pasta string, argv []string) []string {
	var linha []string
	se := func(valor string, partes ...string) []string {
		if valor == "" {
			return nil
		}
		return partes
	}
	switch nome {
	case "kitty":
		linha = append(append([]string{nome}, se(titulo, "--title="+titulo)...), se(pasta, "--directory="+pasta)...)
	case "foot":
		linha = append(append([]string{nome}, se(titulo, "--title="+titulo)...), se(pasta, "--working-directory="+pasta)...)
	case "alacritty":
		linha = append(append([]string{nome}, se(titulo, "--title", titulo)...), se(pasta, "--working-directory", pasta)...)
		if len(argv) > 0 {
			linha = append(linha, "-e")
		}
	case "wezterm":
		linha = append([]string{nome, "start", "--always-new-process"}, se(pasta, "--cwd", pasta)...)
		if len(argv) > 0 {
			linha = append(linha, "--")
		}
	default:
		// O ghostty abre com a config do Mk, como no Maré.
		linha = []string{"ghostty", "--config-default-files=false", "--config-file=?" + base.Config("mk", "ghostty", "config")}
		linha = append(append(linha, se(titulo, "--title="+titulo)...), se(pasta, "--working-directory="+pasta)...)
		if len(argv) > 0 {
			linha = append(linha, "-e")
		}
	}
	return append(linha, argv...)
}
