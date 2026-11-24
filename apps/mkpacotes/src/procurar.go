package main

import (
	"sort"
	"strings"
	"unicode/utf8"

	"mkpacotes/internal/alpm"
	"mkpacotes/internal/aur"
	"mkpacotes/internal/base"
	"mkpacotes/internal/pacman"
)

func init() { registrar("procurar", cmdProcurar) }

// Quantos resultados no máximo.
const maxDeResultados = 200

type achado struct {
	Nome            string  `json:"nome"`
	Versao          string  `json:"versao"`
	Descricao       string  `json:"descricao"`
	Origem          string  `json:"origem"`
	Instalado       bool    `json:"instalado"`
	VersaoInstalada *string `json:"versao_instalada"`
	Votos           *int    `json:"votos"`
	Desatualizado   *bool   `json:"desatualizado"`
}

// ordena põe na frente o nome igual ao texto, depois os que começam com ele,
// depois o resto; em cada faixa, os oficiais (por nome) antes do AUR (por
// votos).
func ordena(lista []achado, texto string) {
	faixa := func(a achado) int {
		switch {
		case a.Nome == texto:
			return 0
		case strings.HasPrefix(a.Nome, texto):
			return 1
		}
		return 2
	}
	sort.SliceStable(lista, func(i, j int) bool {
		a, b := lista[i], lista[j]
		if fa, fb := faixa(a), faixa(b); fa != fb {
			return fa < fb
		}
		if (a.Origem == "aur") != (b.Origem == "aur") {
			return b.Origem == "aur"
		}
		if a.Origem == "aur" && *a.Votos != *b.Votos {
			return *a.Votos > *b.Votos
		}
		return a.Nome < b.Nome
	})
}

// mkpacotes-coletor procurar <texto…> [--sem-aur]
func cmdProcurar(args []string) int {
	palavras, ops, err := opcoes(args, []string{"--sem-aur"}, nil)
	if err != nil {
		return falha(err)
	}
	var limpas []string
	for _, p := range palavras {
		limpas = append(limpas, strings.Fields(p)...)
	}
	texto := strings.ToLower(strings.Join(limpas, " "))
	if utf8.RuneCountInString(texto) < 2 {
		return base.Falha(base.Erro, "escreva pelo menos 2 letras para procurar")
	}
	if len(texto) > 200 || !utf8.ValidString(texto) || strings.ContainsRune(texto, 0) {
		return base.Falha(base.Erro, "a busca é comprida demais ou tem letras que não entendo")
	}
	instalados := map[string]*alpm.Pacote{}
	if local, err := alpm.Le(); err == nil {
		instalados = local.Pacotes
	}
	novo := func(nome, versao, descricao, origem string) achado {
		a := achado{Nome: nome, Versao: versao, Descricao: descricao, Origem: origem}
		if p, ok := instalados[nome]; ok {
			a.Instalado, a.VersaoInstalada = true, &p.Versao
		}
		return a
	}
	lista := []achado{}
	var erroOficial, erroAur *string
	oficiais, err := pacman.Busca(limpas)
	if err != nil {
		frase := err.Error()
		erroOficial = &frase
	}
	visto := map[string]bool{}
	for _, o := range oficiais {
		if !visto[o.Nome] {
			visto[o.Nome] = true
			lista = append(lista, novo(o.Nome, o.Versao, o.Descricao, o.Repo))
		}
	}
	if ops["--sem-aur"] == "" {
		// O AUR procura um trecho só: vai a palavra mais comprida, e as outras
		// filtram aqui.
		maior := limpas[0]
		for _, p := range limpas {
			if len(p) > len(maior) {
				maior = p
			}
		}
		doAur, err := aur.Busca(maior)
		if err != nil {
			frase := err.Error()
			erroAur = &frase
		}
		for _, p := range doAur {
			onde := strings.ToLower(p.Nome + " " + p.Descricao)
			casa := !visto[p.Nome]
			for _, palavra := range limpas {
				casa = casa && strings.Contains(onde, strings.ToLower(palavra))
			}
			if !casa {
				continue
			}
			a := novo(p.Nome, p.Versao, p.Descricao, "aur")
			votos, velho := p.Votos, p.DesatualizadoDesde != nil
			a.Votos, a.Desatualizado = &votos, &velho
			lista = append(lista, a)
		}
	}
	if erroOficial != nil && (erroAur != nil || ops["--sem-aur"] != "") {
		return base.Falha(base.Erro, "%s", *erroOficial)
	}
	ordena(lista, texto)
	cortado := len(lista) > maxDeResultados
	if cortado {
		lista = lista[:maxDeResultados]
	}
	_ = base.Escreve(map[string]any{"resultados": lista, "cortado": cortado, "erro_oficial": erroOficial, "erro_aur": erroAur})
	return base.Certo
}
