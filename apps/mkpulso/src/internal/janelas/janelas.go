// Package janelas pergunta ao Hyprland o que só ele sabe: as janelas abertas
// (hyprctl clients -j), as telas (monitors -j) e a versão (version -j).
//
// O hyprctl é achado no PATH. Sem ele, com ele falhando, travado ou
// devolvendo lixo, a resposta é vazia e quem chamou segue em frente: o
// retrato não depende do compositor. Cada chamada tem um limite de tempo
// curto (LimiteDeTempo), para um Hyprland travado não travar o coletor.
package janelas

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"os/exec"
	"strings"
	"time"
)

// LimiteDeTempo é quanto se espera por uma resposta do hyprctl.
var LimiteDeTempo = time.Second

// Janela é uma janela do Hyprland, como sai em grupos[].janelas. O pid e a
// classe não vão para o JSON: servem para achar o grupo dono e o atalho.
type Janela struct {
	Titulo    string `json:"titulo"`
	Workspace string `json:"workspace"`
	Endereco  string `json:"endereco"`
	PID       int    `json:"-"`
	Classe    string `json:"-"`
}

// Tela é um item de "telas" da ficha.
type Tela struct {
	Nome      string  `json:"nome"`
	Descricao string  `json:"descricao"`
	Largura   int     `json:"largura"`
	Altura    int     `json:"altura"`
	Hz        float64 `json:"hz"`
	Escala    float64 `json:"escala"`
}

// pergunta roda `hyprctl <o que> -j` e decodifica a resposta em destino.
func pergunta(oQue string, destino any) bool {
	caminho, err := exec.LookPath("hyprctl")
	if err != nil {
		return false
	}
	ctx, cancela := context.WithTimeout(context.Background(), LimiteDeTempo)
	defer cancela()
	cmd := exec.CommandContext(ctx, caminho, oQue, "-j")
	var saida bytes.Buffer
	cmd.Stdout = &saida
	// se o hyprctl deixar um filho segurando a saída, não espera por ele
	cmd.WaitDelay = 100 * time.Millisecond
	if err := cmd.Run(); err != nil {
		return false
	}
	return json.Unmarshal(saida.Bytes(), destino) == nil
}

// Listar devolve as janelas mapeadas que têm um processo dono. Nunca é nil.
func Listar() []Janela {
	js := []Janela{}
	var clientes []struct {
		Address   string `json:"address"`
		Mapped    *bool  `json:"mapped"`
		Title     string `json:"title"`
		Class     string `json:"class"`
		PID       int    `json:"pid"`
		Workspace struct {
			Name string `json:"name"`
		} `json:"workspace"`
	}
	if !pergunta("clients", &clientes) {
		return js
	}
	for _, c := range clientes {
		if c.PID <= 0 || (c.Mapped != nil && !*c.Mapped) {
			continue
		}
		js = append(js, Janela{Titulo: c.Title, Workspace: c.Workspace.Name, Endereco: c.Address, PID: c.PID, Classe: c.Class})
	}
	return js
}

// Telas devolve os monitores ligados. Nunca é nil.
func Telas() []Tela {
	ts := []Tela{}
	var monitores []struct {
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Width       int     `json:"width"`
		Height      int     `json:"height"`
		RefreshRate float64 `json:"refreshRate"`
		Scale       float64 `json:"scale"`
	}
	if !pergunta("monitors", &monitores) {
		return ts
	}
	for _, m := range monitores {
		ts = append(ts, Tela{Nome: m.Name, Descricao: m.Description, Largura: m.Width, Altura: m.Height,
			Hz: math.Round(m.RefreshRate*100) / 100, Escala: m.Scale})
	}
	return ts
}

// Compositor devolve "Hyprland <versão>", ou nil se o Hyprland não responde
// ou responde sem versão.
func Compositor() *string {
	var v struct {
		Version string `json:"version"`
		Tag     string `json:"tag"`
	}
	if !pergunta("version", &v) {
		return nil
	}
	versao := v.Version
	if versao == "" {
		versao = strings.TrimPrefix(v.Tag, "v")
	}
	if versao == "" {
		return nil
	}
	nome := "Hyprland " + versao
	return &nome
}
