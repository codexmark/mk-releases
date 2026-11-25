package main

import (
	"mkporto/internal/base"
	"mkporto/internal/sshcfg"
)

func init() { registrar("listar", listar) }

// hostJSON é um host do jeito que o contrato o descreve: o que ele não tem
// sai como null.
type hostJSON struct {
	Apelido string       `json:"apelido"`
	Host    string       `json:"host"`
	Usuario *string      `json:"usuario"`
	Porta   int          `json:"porta"`
	Chave   *string      `json:"chave"`
	Salto   *string      `json:"salto"`
	Origem  string       `json:"origem"`
	Arquivo string       `json:"arquivo"`
	Efetivo bool         `json:"efetivo"`
	Escrito *escritoJSON `json:"escrito"`
}

// escritoJSON é o que está escrito no arquivo do app para um host, sem
// completar com nada: é daí que o formulário de edição parte.
type escritoJSON struct {
	Host    string  `json:"host"`
	Usuario *string `json:"usuario"`
	Porta   int     `json:"porta"`
	Chave   *string `json:"chave"`
	Salto   *string `json:"salto"`
}

// ouNada troca o texto vazio por null.
func ouNada(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// catalogo lê o config e devolve os hosts, com os valores do leitor próprio.
func catalogo() []sshcfg.Host {
	dir := base.PastaDoSsh()
	return sshcfg.Le(dir, base.Home()).Hosts(sshcfg.ArquivoDoApp(dir))
}

// listar: os hosts do ~/.ssh/config e dos arquivos que ele inclui, com os
// valores como o ssh os vê.
func listar(args []string) int {
	if pos, err := posicionais(args); err != nil {
		return base.Falha(base.Erro, "%s", err)
	} else if len(pos) != 0 {
		return base.Falha(base.Erro, "uso: mkporto-motor listar")
	}
	dir := base.PastaDoSsh()
	escritos := map[string]sshcfg.Entrada{}
	for _, e := range sshcfg.DoApp(sshcfg.ArquivoDoApp(dir)) {
		escritos[e.Apelido] = e
	}
	lidos := catalogo()
	respondeu := efetiva(lidos)
	hosts := []hostJSON{}
	for i, h := range lidos {
		j := hostJSON{Apelido: h.Apelido, Host: h.Host, Usuario: ouNada(h.Usuario), Porta: h.Porta,
			Chave: ouNada(h.Chave), Salto: ouNada(h.Salto), Origem: h.Origem, Arquivo: base.Til(h.Arquivo), Efetivo: respondeu[i]}
		if e, tem := escritos[h.Apelido]; tem && h.Origem == "mkporto" {
			j.Escrito = &escritoJSON{Host: e.Host, Usuario: ouNada(e.Usuario), Porta: e.Porta, Chave: ouNada(e.Chave), Salto: ouNada(e.Salto)}
		}
		hosts = append(hosts, j)
	}
	_ = base.Escreve(map[string]any{
		"pasta":          base.Til(dir),
		"arquivo_do_app": base.Til(sshcfg.ArquivoDoApp(dir)),
		"hosts":          hosts,
	})
	return base.Certo
}
