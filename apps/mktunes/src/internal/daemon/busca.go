// SPDX-License-Identifier: GPL-3.0-or-later

package daemon

import (
	"context"
	"strings"
	"sync"
	"time"
	"unicode"

	"mktunes/internal/app"
	"mktunes/internal/dominio"
	"mktunes/internal/laco"
)

// busca é a aba Buscar: tudo de uma vez (o melhor resultado, as músicas e uma
// seção por tipo de página), ou um tipo só pelo filtro; as buscas recentes e
// as sugestões enquanto se digita.
type busca struct {
	quadro    *Quadro
	laco      laco.Laco
	acervo    *app.Acervo
	catalogo  app.Catalogo
	paginas   app.BuscaDePaginas
	sugestoes app.FonteDeSugestoes

	situacao  string
	consulta  string
	filtro    dominio.Filtro // "": tudo de uma vez
	faixas    []msg
	itens     []msg
	topo      any
	secoes    []msg
	sugeridas []string

	buscando, sugerindo, esperando laco.Tarefa
}

// sugereDepois: só pergunta quando a digitação pausa, para cada tecla não
// virar um pedido.
const sugereDepois = 300 * time.Millisecond

// secoes, nesta ordem: o melhor resultado olha os artistas antes dos álbuns.
var secoesDaBusca = []struct {
	filtro dominio.Filtro
	titulo string
}{{dominio.FiltroArtistas, "Artistas"}, {dominio.FiltroAlbuns, "Álbuns"}, {dominio.FiltroPlaylists, "Playlists"}}

func novaBusca(q *Quadro, l laco.Laco, a *app.Acervo, c app.Catalogo, p app.BuscaDePaginas, s app.FonteDeSugestoes) *busca {
	b := &busca{quadro: q, laco: l, acervo: a, catalogo: c, paginas: p, sugestoes: s, situacao: "idle"}
	b.publica()
	return b
}

func (b *busca) publica() {
	filtro := "all"
	if b.filtro != "" {
		filtro = strings.ToLower(string(b.filtro))
	}
	b.quadro.Estado("search", msg{
		"status": b.situacao, "query": b.consulta, "filter": filtro,
		"tracks": naoNula(b.faixas), "items": naoNula(b.itens), "top": b.topo, "sections": naoNula(b.secoes),
		"recent": textos(b.acervo.Buscas()), "suggestions": textos(b.sugeridas),
	})
}

func naoNula(lista []msg) []msg {
	if lista == nil {
		return []msg{}
	}
	return lista
}

func textos(lista []string) []string {
	if lista == nil {
		return []string{}
	}
	return lista
}

func (b *busca) limpa() { b.faixas, b.itens, b.topo, b.secoes = nil, nil, nil, nil }

func (b *busca) trata(cmd string, a args) bool {
	switch cmd {
	case "search":
		filtro := b.filtro
		if nome, ok := a.texto("filter"); ok {
			if nome == "all" {
				filtro = ""
			} else if lido, ok := dominio.FiltroDe(nome); ok {
				filtro = lido
			}
		}
		b.roda(a.textoOu("query", b.consulta), filtro)
	case "suggest":
		b.sugere(a.textoOu("text", ""))
	case "clearSearch":
		b.paraDeSugerir()
		laco.Cancela(b.buscando)
		b.situacao, b.consulta, b.sugeridas = "idle", "", nil
		b.limpa()
		b.publica()
	case "clearRecent":
		b.acervo.LimpaBuscas()
		b.publica()
	default:
		return false
	}
	return true
}

func (b *busca) paraDeSugerir() {
	laco.Cancela(b.esperando)
	laco.Cancela(b.sugerindo)
}

func (b *busca) roda(consulta string, filtro dominio.Filtro) {
	b.paraDeSugerir()
	b.consulta, b.filtro, b.sugeridas = consulta, filtro, nil
	if strings.TrimSpace(consulta) == "" {
		b.publica()
		return
	}
	laco.Cancela(b.buscando)
	b.situacao = "loading"
	b.acervo.LembraBusca(consulta)
	b.publica()
	b.buscando = b.laco.Vai(func(ctx context.Context) func() {
		switch filtro {
		case "":
			return b.tudo(ctx, consulta)
		case dominio.FiltroMusicas:
			faixas, desfecho := app.BuscaMusicas(ctx, b.catalogo, consulta)
			return func() {
				if b.encerra(desfecho) {
					b.faixas = faixasNoFio(faixas)
				}
				b.publica()
			}
		}
		itens, desfecho := app.BuscaPaginas(ctx, b.paginas, consulta, filtro)
		return func() {
			if b.encerra(desfecho) {
				b.itens = itensNoFio(itens)
			}
			b.publica()
		}
	})
}

// encerra põe a situação pelo desfecho; com resultados (true), o que estava na
// tela é limpo para receber os novos.
func (b *busca) encerra(d app.Desfecho) bool {
	switch d {
	case app.Achou:
		b.situacao = "results"
		b.limpa()
		return true
	case app.NadaAchado:
		b.situacao = "nothing"
	case app.Indisponivel:
		b.situacao = "unavailable"
	default:
		b.situacao = "idle"
	}
	return false
}

// tudo pede as músicas e cada tipo de página juntos; o que responder é mostrado.
func (b *busca) tudo(ctx context.Context, consulta string) func() {
	var (
		espera     sync.WaitGroup
		faixas     []dominio.Faixa
		dasMusicas app.Desfecho
		achados    = make([][]dominio.Item, len(secoesDaBusca))
		dasPaginas = make([]app.Desfecho, len(secoesDaBusca))
	)
	espera.Add(1 + len(secoesDaBusca))
	go func() {
		defer espera.Done()
		faixas, dasMusicas = app.BuscaMusicas(ctx, b.catalogo, consulta)
	}()
	for i, secao := range secoesDaBusca {
		go func() {
			defer espera.Done()
			achados[i], dasPaginas[i] = app.BuscaPaginas(ctx, b.paginas, consulta, secao.filtro)
		}()
	}
	espera.Wait()
	return func() {
		b.limpa()
		foraDoAr := dasMusicas == app.Indisponivel
		var secoes []msg
		for i, secao := range secoesDaBusca {
			foraDoAr = foraDoAr && dasPaginas[i] == app.Indisponivel
			if len(achados[i]) > 0 {
				secoes = append(secoes, msg{"title": secao.titulo, "filter": strings.ToLower(string(secao.filtro)), "items": itensNoFio(achados[i])})
			}
		}
		switch {
		case dasMusicas == app.ConsultaInvalida:
			b.situacao = "idle"
		case foraDoAr:
			b.situacao = "unavailable"
		case len(faixas) == 0 && len(secoes) == 0:
			b.situacao = "nothing"
		default:
			b.situacao, b.faixas, b.secoes = "results", faixasNoFio(faixas), secoes
			if melhor, tem := MelhorResultado(consulta, achados[0], achados[1], faixas); tem {
				b.topo = itemNoFio(melhor)
			}
		}
		b.publica()
	}
}

func (b *busca) sugere(texto string) {
	b.paraDeSugerir()
	b.esperando = b.laco.Depois(sugereDepois, func() {
		b.sugerindo = b.laco.Vai(func(ctx context.Context) func() {
			achadas := app.Sugere(ctx, b.sugestoes, texto)
			return func() {
				b.sugeridas = achadas
				b.publica()
			}
		})
	})
}

// MelhorResultado é o resultado mostrado grande em cima de uma busca por tudo.
// A primeira música do catálogo é o melhor palpite dele, a não ser que a busca
// diga exatamente o nome de um artista ou de um álbum: aí é essa página que se
// queria.
func MelhorResultado(consulta string, artistas, albuns []dominio.Item, musicas []dominio.Faixa) (dominio.Item, bool) {
	procurado := plano(consulta)
	if len(artistas) > 0 && plano(artistas[0].Titulo) == procurado {
		return artistas[0], true
	}
	if len(albuns) > 0 && plano(albuns[0].Titulo) == procurado {
		return albuns[0], true
	}
	switch {
	case len(musicas) > 0:
		return dominio.ItemDeMusica(musicas[0]), true
	case len(artistas) > 0:
		return artistas[0], true
	case len(albuns) > 0:
		return albuns[0], true
	}
	return dominio.Item{}, false
}

// semAcento troca as letras acentuadas do português (e vizinhos) pela letra
// pura; a biblioteca padrão não traz a decomposição do Unicode.
var semAcento = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a", "å", "a", "é", "e", "è", "e", "ê", "e", "ë", "e",
	"í", "i", "ì", "i", "î", "i", "ï", "i", "ó", "o", "ò", "o", "ô", "o", "õ", "o", "ö", "o",
	"ú", "u", "ù", "u", "û", "u", "ü", "u", "ç", "c", "ñ", "n", "ý", "y", "ÿ", "y",
)

// plano tira caixa, acentos e espaços repetidos de um texto, para comparar.
func plano(texto string) string {
	minusculo := semAcento.Replace(strings.ToLower(texto))
	// Acentos que vieram soltos (texto já decomposto) saem também.
	minusculo = strings.Map(func(letra rune) rune {
		if unicode.Is(unicode.Mn, letra) {
			return -1
		}
		return letra
	}, minusculo)
	return strings.Join(strings.Fields(minusculo), " ")
}
