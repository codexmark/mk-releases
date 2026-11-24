package main

import (
	"fmt"
	"strings"

	"mkpacotes/internal/acoes"
	"mkpacotes/internal/alpm"
	"mkpacotes/internal/aur"
	"mkpacotes/internal/base"
	"mkpacotes/internal/pacman"
)

func init() { registrar("detalhe", cmdDetalhe) }

type dependencia struct {
	Nome      string `json:"nome"`
	Pede      string `json:"pede"`
	Instalado bool   `json:"instalado"`
}

type opcional struct {
	Nome      string `json:"nome"`
	Para      string `json:"para"`
	Instalado bool   `json:"instalado"`
}

type arquivos struct {
	Total       int      `json:"total"`
	Executaveis []string `json:"executaveis"`
}

type noAur struct {
	Versao        string  `json:"versao"`
	Votos         int     `json:"votos"`
	Popularidade  float64 `json:"popularidade"`
	Desatualizado *int64  `json:"desatualizado_desde_s"`
	Mantenedor    *string `json:"mantenedor"`
	ModificadoEm  *int64  `json:"modificado_em_s"`
}

type ficha struct {
	Nome         string        `json:"nome"`
	Versao       string        `json:"versao"`
	Descricao    string        `json:"descricao"`
	Origem       *string       `json:"origem"`
	Instalado    bool          `json:"instalado"`
	Motivo       *string       `json:"motivo"`
	Orfao        bool          `json:"orfao"`
	Protegido    bool          `json:"protegido"`
	Atualizacao  *string       `json:"atualizacao"`
	URL          string        `json:"url"`
	Licencas     []string      `json:"licencas"`
	Mantenedor   *string       `json:"mantenedor"`
	Tamanho      *int64        `json:"tamanho_bytes"`
	Baixar       *int64        `json:"baixar_bytes"`
	InstaladoEm  *int64        `json:"instalado_em_s"`
	ConstruidoEm *int64        `json:"construido_em_s"`
	Depende      []dependencia `json:"depende"`
	Opcionais    []opcional    `json:"opcionais"`
	Fornece      []string      `json:"fornece"`
	Conflita     []string      `json:"conflita"`
	RequeridoPor []string      `json:"requerido_por"`
	OpcionalPara []string      `json:"opcional_para"`
	Arquivos     *arquivos     `json:"arquivos"`
	Aur          *noAur        `json:"aur"`
	ErroAur      *string       `json:"erro_aur"`
}

// cheia: uma lista que sai como [] no JSON, nunca null.
func cheia(l []string) []string {
	if l == nil {
		return []string{}
	}
	return l
}

func texto(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func dependencias(local *alpm.Base, pedem []string) []dependencia {
	lista := make([]dependencia, 0, len(pedem))
	for _, d := range pedem {
		lista = append(lista, dependencia{Nome: alpm.NomeDe(d), Pede: d, Instalado: local.Satisfeito(d)})
	}
	return lista
}

func opcionais(local *alpm.Base, pedem []string) []opcional {
	lista := make([]opcional, 0, len(pedem))
	for _, o := range pedem {
		_, para, _ := strings.Cut(o, ":")
		lista = append(lista, opcional{Nome: alpm.NomeDe(o), Para: strings.TrimSpace(para), Instalado: local.Satisfeito(alpm.NomeDe(o))})
	}
	return lista
}

func doAur(p aur.Pacote) *noAur {
	return &noAur{Versao: p.Versao, Votos: p.Votos, Popularidade: p.Popularidade, Desatualizado: p.DesatualizadoDesde,
		Mantenedor: texto(p.Mantenedor), ModificadoEm: p.ModificadoEm}
}

// perguntaAoAur devolve a ficha de um pacote no AUR: (nil, nil) se ele não
// está lá.
func perguntaAoAur(nome string) (*aur.Pacote, error) {
	lista, err := aur.Info([]string{nome})
	if err != nil {
		return nil, err
	}
	for i := range lista {
		if lista[i].Nome == nome {
			return &lista[i], nil
		}
	}
	return nil, nil
}

// mkpacotes-coletor detalhe <nome>
func cmdDetalhe(args []string) int {
	pos, _, err := opcoes(args, nil, nil)
	if err != nil {
		return falha(err)
	}
	if len(pos) != 1 {
		return base.Falha(base.Erro, "uso: mkpacotes-coletor detalhe <nome>")
	}
	nome := pos[0]
	if err := alpm.ValidaNome(nome); err != nil {
		return falha(err)
	}
	local, err := alpm.Le()
	if err != nil {
		return falha(err)
	}
	f := ficha{Nome: nome, Protegido: acoes.Protegidos()[nome]}

	if p, ok := local.Pacotes[nome]; ok {
		m := motivo(p)
		f.Versao, f.Descricao, f.Instalado, f.Motivo, f.Orfao = p.Versao, p.Descricao, true, &m, local.Orfao(nome)
		f.URL, f.Licencas, f.Mantenedor = p.URL, cheia(p.Licencas), texto(p.Empacotador)
		f.Tamanho, f.InstaladoEm, f.ConstruidoEm = &p.Tamanho, p.InstaladoEm, p.ConstruidoEm
		f.Depende, f.Opcionais = dependencias(local, p.Depende), opcionais(local, p.Opcionais)
		f.Fornece, f.Conflita = cheia(p.Fornece), cheia(p.Conflita)
		f.RequeridoPor, f.OpcionalPara = cheia(local.RequeridoPor(nome)), cheia(local.OpcionalPara(nome))
		if total, execs, err := local.Arquivos(nome); err == nil {
			f.Arquivos = &arquivos{Total: total, Executaveis: cheia(execs)}
		}
		if repos, err := pacman.Repos(); err == nil {
			f.Origem, f.Atualizacao = origemE(p, repos)
		}
		if f.Origem != nil && *f.Origem == "aur" {
			la, err := perguntaAoAur(nome)
			switch {
			case err != nil:
				f.ErroAur = texto(err.Error())
			case la != nil:
				f.Aur = doAur(*la)
			}
		}
		_ = base.Escreve(f)
		return base.Certo
	}

	f.RequeridoPor, f.OpcionalPara = []string{}, []string{}
	doRepo, erroDoPacman := pacman.Info(nome)
	if doRepo != nil {
		f.Versao, f.Descricao, f.Origem = doRepo.Versao, doRepo.Descricao, &doRepo.Repo
		f.URL, f.Licencas, f.Mantenedor = doRepo.URL, cheia(doRepo.Licencas), texto(doRepo.Empacotador)
		f.Tamanho, f.Baixar, f.ConstruidoEm = doRepo.Tamanho, doRepo.Baixar, doRepo.ConstruidoEm
		f.Depende, f.Opcionais = dependencias(local, doRepo.Depende), opcionais(local, doRepo.Opcionais)
		f.Fornece, f.Conflita = cheia(doRepo.Fornece), cheia(doRepo.Conflita)
		_ = base.Escreve(f)
		return base.Certo
	}

	la, err := perguntaAoAur(nome)
	if err != nil {
		return base.Falha(base.Erro, "%s não está instalado nem nos repositórios, e não deu para procurar no AUR: %v", nome, err)
	}
	if la == nil {
		if erroDoPacman != nil {
			return base.Falha(base.Erro, "não achei %s no AUR, e o pacman não respondeu: %v", nome, erroDoPacman)
		}
		return base.Falha(base.Erro, "%s", fmt.Sprintf("não achei %s nos repositórios nem no AUR", nome))
	}
	origem := "aur"
	f.Versao, f.Descricao, f.Origem = la.Versao, la.Descricao, &origem
	f.URL, f.Licencas, f.Mantenedor = la.URL, cheia(la.Licencas), texto(la.Mantenedor)
	f.ConstruidoEm = la.ModificadoEm
	f.Depende, f.Opcionais = dependencias(local, la.Depende), opcionais(local, la.Opcionais)
	f.Fornece, f.Conflita = cheia(la.Fornece), cheia(la.Conflita)
	f.Aur = doAur(*la)
	_ = base.Escreve(f)
	return base.Certo
}
