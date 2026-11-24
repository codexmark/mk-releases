package main

import (
	"mkpacotes/internal/acoes"
	"mkpacotes/internal/alpm"
	"mkpacotes/internal/base"
	"mkpacotes/internal/pacman"
)

func init() { registrar("instalados", cmdInstalados) }

type instalado struct {
	Nome        string  `json:"nome"`
	Versao      string  `json:"versao"`
	Descricao   string  `json:"descricao"`
	Tamanho     int64   `json:"tamanho_bytes"`
	InstaladoEm *int64  `json:"instalado_em_s"`
	Motivo      string  `json:"motivo"`
	Origem      *string `json:"origem"`
	Orfao       bool    `json:"orfao"`
	Atualizacao *string `json:"atualizacao"`
	Protegido   bool    `json:"protegido"`
}

func motivo(p *alpm.Pacote) string {
	if p.Pedido {
		return "pedido"
	}
	return "dependencia"
}

// origemE diz de onde é um instalado e a versão nova, se o repositório tem
// uma: (nil, nil) quando não se sabe (o pacman não respondeu).
func origemE(p *alpm.Pacote, repos map[string]pacman.Sinc) (origem, atualizacao *string) {
	if repos == nil {
		return nil, nil
	}
	s, ok := repos[p.Nome]
	if !ok {
		o := "aur"
		return &o, nil
	}
	if alpm.Vercmp(s.Versao, p.Versao) > 0 {
		atualizacao = &s.Versao
	}
	return &s.Repo, atualizacao
}

// mkpacotes-coletor instalados
func cmdInstalados(args []string) int {
	pos, _, err := opcoes(args, nil, nil)
	if err != nil {
		return falha(err)
	}
	if len(pos) != 0 {
		return base.Falha(base.Erro, "uso: mkpacotes-coletor instalados")
	}
	local, err := alpm.Le()
	if err != nil {
		return falha(err)
	}
	var aviso *string
	repos, err := pacman.Repos()
	if err != nil {
		frase := "não deu para saber de que repositório é cada pacote nem o que há para atualizar: " + err.Error()
		aviso, repos = &frase, nil
	}
	protegidos := acoes.Protegidos()
	var totais struct {
		Pacotes      int   `json:"pacotes"`
		Pedidos      int   `json:"pedidos"`
		Aur          int   `json:"aur"`
		Orfaos       int   `json:"orfaos"`
		Atualizacoes int   `json:"atualizacoes"`
		Tamanho      int64 `json:"tamanho_bytes"`
	}
	lista := make([]instalado, 0, len(local.Pacotes))
	for _, nome := range local.Nomes() {
		p := local.Pacotes[nome]
		i := instalado{Nome: p.Nome, Versao: p.Versao, Descricao: p.Descricao, Tamanho: p.Tamanho, InstaladoEm: p.InstaladoEm,
			Motivo: motivo(p), Orfao: local.Orfao(nome), Protegido: protegidos[nome]}
		i.Origem, i.Atualizacao = origemE(p, repos)
		totais.Pacotes++
		totais.Tamanho += p.Tamanho
		if p.Pedido {
			totais.Pedidos++
		}
		if i.Origem != nil && *i.Origem == "aur" {
			totais.Aur++
		}
		if i.Orfao {
			totais.Orfaos++
		}
		if i.Atualizacao != nil {
			totais.Atualizacoes++
		}
		lista = append(lista, i)
	}
	_ = base.Escreve(map[string]any{"pacotes": lista, "totais": totais, "aviso": aviso})
	return base.Certo
}
