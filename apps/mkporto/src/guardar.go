package main

import (
	"encoding/json"
	"io"
	"os"

	"mkporto/internal/base"
	"mkporto/internal/sshcfg"
)

func init() {
	registrar("guardar", guardar)
	registrar("remover", remover)
}

// entrada é de onde o `guardar` lê o pedido. Os testes trocam.
var entrada io.Reader = os.Stdin

// Um pedido não passa disto (um host são poucas linhas).
const pedidoMaximo = 64 * 1024

// pedidoDeGuardar é o JSON que a janela manda.
type pedidoDeGuardar struct {
	Apelido  string `json:"apelido"`
	Host     string `json:"host"`
	Usuario  string `json:"usuario"`
	Porta    *int   `json:"porta"`
	Chave    string `json:"chave"`
	Salto    string `json:"salto"`
	Anterior string `json:"anterior"`
}

// lePedido lê um objeto JSON só, sem campo desconhecido e sem sobra.
func lePedido() (*pedidoDeGuardar, bool) {
	var p pedidoDeGuardar
	dec := json.NewDecoder(io.LimitReader(entrada, pedidoMaximo))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&p); err != nil {
		return nil, false
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, false
	}
	return &p, true
}

// doConfigDaPessoa é a frase de quando o apelido não é do app.
func doConfigDaPessoa(apelido, arquivo string) int {
	return base.Falha(base.Recusado, "o host %s é do seu ~/.ssh/config (está em %s): o MkPorto não mexe no que não é dele",
		apelido, base.Til(arquivo))
}

// guardar: grava um host (novo ou editado) no arquivo do app e garante o
// Include no topo do config.
func guardar(args []string) int {
	if pos, err := posicionais(args); err != nil {
		return base.Falha(base.Erro, "%s", err)
	} else if len(pos) != 0 {
		return base.Falha(base.Erro, "uso: mkporto-motor guardar (com o pedido em JSON na entrada padrão)")
	}
	p, ok := lePedido()
	if !ok {
		return base.Falha(base.Erro, "o pedido não é o JSON que o guardar espera")
	}
	nova := sshcfg.Entrada{Apelido: p.Apelido, Host: p.Host, Usuario: p.Usuario, Porta: 22, Chave: p.Chave, Salto: p.Salto}
	if p.Porta != nil {
		nova.Porta = *p.Porta
	}
	if err := sshcfg.Valida(nova); err != nil {
		return base.FalhaDeCampo(err.Campo, err.Frase)
	}
	if p.Anterior != "" && !sshcfg.ApelidoSeguro(p.Anterior) {
		return base.FalhaDeCampo("anterior", "o apelido anterior não é um apelido válido")
	}

	dir := base.PastaDoSsh()
	app := sshcfg.ArquivoDoApp(dir)
	doUsuario := sshcfg.Le(dir, base.Home())
	for _, apelido := range []string{p.Anterior, nova.Apelido} {
		if apelido == "" {
			continue
		}
		if arquivo := doUsuario.DeclaradoFora(apelido, app); arquivo != "" {
			return doConfigDaPessoa(apelido, arquivo)
		}
	}

	entradas := sshcfg.DoApp(app)
	indice := func(apelido string) int {
		for i, e := range entradas {
			if e.Apelido == apelido {
				return i
			}
		}
		return -1
	}
	switch {
	case p.Anterior != "" && p.Anterior != nova.Apelido:
		i := indice(p.Anterior)
		if i < 0 {
			return base.Falha(base.Erro, "não há um host %s guardado pelo MkPorto para editar", p.Anterior)
		}
		if indice(nova.Apelido) >= 0 {
			return base.FalhaDeCampo("apelido", "já existe um host com o apelido "+nova.Apelido)
		}
		entradas[i] = nova
	case indice(nova.Apelido) >= 0:
		entradas[indice(nova.Apelido)] = nova
	default:
		entradas = append(entradas, nova)
	}

	if err := sshcfg.Grava(dir, entradas); err != nil {
		return base.Falha(base.Erro, "%s", err)
	}
	posto, err := sshcfg.GaranteInclude(dir, base.Home())
	if err != nil {
		return base.Falha(base.Erro, "o host foi guardado, mas o ssh ainda não o vê: %s", err)
	}
	include := "ja_havia"
	if posto {
		include = "posto"
	}
	_ = base.Escreve(map[string]any{"ok": true, "apelido": nova.Apelido, "arquivo": base.Til(app), "include": include})
	return base.Certo
}

// remover: tira um host do arquivo do app. O que é do config da pessoa fica.
func remover(args []string) int {
	pos, err := posicionais(args)
	if err != nil {
		return base.Falha(base.Erro, "%s", err)
	}
	if len(pos) != 1 {
		return base.Falha(base.Erro, "uso: mkporto-motor remover [--] <apelido>")
	}
	apelido := pos[0]
	dir := base.PastaDoSsh()
	app := sshcfg.ArquivoDoApp(dir)
	entradas := sshcfg.DoApp(app)
	ficam := make([]sshcfg.Entrada, 0, len(entradas))
	for _, e := range entradas {
		if e.Apelido != apelido {
			ficam = append(ficam, e)
		}
	}
	if len(ficam) == len(entradas) {
		if arquivo := sshcfg.Le(dir, base.Home()).DeclaradoFora(apelido, app); arquivo != "" {
			return doConfigDaPessoa(apelido, arquivo)
		}
		return base.Falha(base.Erro, "não conheço o host %s", apelido)
	}
	if err := sshcfg.Grava(dir, ficam); err != nil {
		return base.Falha(base.Erro, "%s", err)
	}
	_ = base.Escreve(map[string]any{"ok": true, "apelido": apelido})
	return base.Certo
}
