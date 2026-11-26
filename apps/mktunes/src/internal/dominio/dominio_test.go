// SPDX-License-Identifier: GPL-3.0-or-later

package dominio

import (
	"math"
	"math/rand/v2"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func umaFaixa(id string) Faixa {
	return Faixa{ID: id, Provedor: Deezer, Titulo: "Track " + id, Artista: "Artist " + id, Duracao: 30 * time.Second, Stream: "https://cdn.example/" + id + ".mp3"}
}

func igual(t *testing.T, esperado, obtido any) {
	t.Helper()
	if !reflect.DeepEqual(esperado, obtido) {
		t.Fatalf("esperado %#v\n  obtido %#v", esperado, obtido)
	}
}

func ids(faixas []Faixa) []string {
	lista := []string{}
	for _, f := range faixas {
		lista = append(lista, f.ID)
	}
	return lista
}

func atual(t *testing.T, q Fila) string {
	t.Helper()
	f, ok := q.Atual()
	if !ok {
		return ""
	}
	return f.ID
}

func TestFila(t *testing.T) {
	um, dois, tres := umaFaixa("1"), umaFaixa("2"), umaFaixa("3")
	t.Run("vazia não toca nada nem tem a seguir", func(t *testing.T) {
		q := FilaVazia()
		igual(t, "", atual(t, q))
		igual(t, 0, len(q.ASeguir()))
		_, tem := q.Ultima()
		igual(t, false, tem)
	})
	t.Run("começa na faixa escolhida e lista o resto", func(t *testing.T) {
		q := NovaFila([]Faixa{um, dois, tres}, 1)
		igual(t, "2", atual(t, q))
		igual(t, []string{"3"}, ids(q.ASeguir()))
	})
	t.Run("próxima anda, e no fim acaba ou dá a volta", func(t *testing.T) {
		igual(t, "2", atual(t, NovaFila([]Faixa{um, dois, tres}, 0).Proxima()))
		igual(t, "", atual(t, NovaFila([]Faixa{um, dois}, 1).Proxima()))
		igual(t, "1", atual(t, NovaFila([]Faixa{um, dois}, 1).ComRepetir(RepetirTudo).Proxima()))
	})
	t.Run("repetir uma toca de novo ao terminar, mas próxima sai dela", func(t *testing.T) {
		q := NovaFila([]Faixa{um, dois}, 0).ComRepetir(RepetirUma)
		igual(t, "1", atual(t, q.AoTerminar()))
		igual(t, "2", atual(t, q.Proxima()))
	})
	t.Run("parar depois desta acaba a fila e desliga a marca", func(t *testing.T) {
		q := NovaFila([]Faixa{um, dois}, 0).ComPararAposAtual(true).AoTerminar()
		igual(t, "", atual(t, q))
		igual(t, false, q.PararAposAtual())
	})
	t.Run("anterior volta, e na primeira fica", func(t *testing.T) {
		igual(t, "1", atual(t, NovaFila([]Faixa{um, dois}, 0).Anterior()))
		igual(t, "1", atual(t, NovaFila([]Faixa{um, dois}, 1).Anterior()))
	})
	t.Run("tocar a seguir, inserir e enfileirar", func(t *testing.T) {
		base := NovaFila([]Faixa{um, dois}, 0)
		igual(t, []string{"3", "2"}, ids(base.TocaASeguir(tres).ASeguir()))
		igual(t, []string{"2", "3"}, ids(base.InsereASeguir(1, tres).ASeguir()))
		igual(t, []string{"2", "3"}, ids(base.Enfileira(tres).ASeguir()))
		igual(t, []string{"2"}, ids(base.ASeguir())) // a original não muda
		igual(t, "1", atual(t, FilaVazia().Enfileira(um)))
	})
	t.Run("pular para uma faixa a seguir guarda o que ficou para trás", func(t *testing.T) {
		q := NovaFila([]Faixa{um, dois, tres}, 0).PulaPara(1)
		igual(t, "3", atual(t, q))
		igual(t, 0, len(q.ASeguir()))
		igual(t, "2", atual(t, q.Anterior()))
		igual(t, "1", atual(t, NovaFila([]Faixa{um, dois}, 0).PulaPara(7)))
	})
	t.Run("o ciclo de repetir", func(t *testing.T) {
		q := NovaFila([]Faixa{um}, 0)
		var modos []Repetir
		for range 3 {
			q = q.CicloRepetir()
			modos = append(modos, q.Repetir())
		}
		igual(t, []Repetir{RepetirTudo, RepetirUma, RepetirNao}, modos)
	})
	t.Run("enfileirar várias", func(t *testing.T) {
		igual(t, []string{"2", "3"}, ids(NovaFila([]Faixa{um}, 0).EnfileiraTodas([]Faixa{dois, tres}).ASeguir()))
		q := FilaVazia().EnfileiraTodas([]Faixa{um, dois})
		igual(t, "1", atual(t, q))
		igual(t, []string{"2"}, ids(q.ASeguir()))
	})
	t.Run("sabe o que já tem e qual é a última", func(t *testing.T) {
		q := NovaFila([]Faixa{um, dois}, 1)
		igual(t, true, q.Contem("1"))
		igual(t, false, q.Contem("3"))
		ultima, _ := NovaFila([]Faixa{um, dois, tres}, 0).Ultima()
		igual(t, "3", ultima.ID)
	})
	t.Run("remover, mover e limpar o que vem a seguir não mexe na atual", func(t *testing.T) {
		base := NovaFila([]Faixa{um, dois, tres}, 0)
		igual(t, []string{"3"}, ids(base.RemoveASeguir(0).ASeguir()))
		igual(t, []string{"3", "2"}, ids(base.MoveASeguir(1, 0).ASeguir()))
		igual(t, 0, len(base.LimpaASeguir().ASeguir()))
		igual(t, "1", atual(t, base.LimpaASeguir()))
		igual(t, []string{"2", "3"}, ids(base.RemoveASeguir(5).MoveASeguir(0, 3).MoveASeguir(4, 0).ASeguir()))
	})
	t.Run("embaralhar mantém a atual e todas as outras", func(t *testing.T) {
		var faixas []Faixa
		for i := range 20 {
			faixas = append(faixas, umaFaixa(string(rune('a'+i))))
		}
		acaso := rand.New(rand.NewPCG(7, 7))
		q := NovaFila(faixas, 5).Embaralhada(acaso.Shuffle)
		igual(t, "f", atual(t, q))
		seguir := ids(q.ASeguir())
		igual(t, 19, len(seguir))
		sort.Strings(seguir)
		igual(t, strings.Split("abcdeghijklmnopqrst", ""), seguir)
		repetida := ids(NovaFila([]Faixa{um, dois, um}, 0).Embaralhada(acaso.Shuffle).ASeguir())
		sort.Strings(repetida)
		igual(t, []string{"1", "2"}, repetida)
	})
	t.Run("o retrato refaz a fila, e um retrato estragado volta vazio", func(t *testing.T) {
		q := NovaFila([]Faixa{um, dois, tres}, 1).ComRepetir(RepetirTudo)
		de := Restaura(q.Retrato())
		igual(t, "2", atual(t, de))
		igual(t, []string{"3"}, ids(de.ASeguir()))
		igual(t, RepetirTudo, de.Repetir())
		igual(t, "1", atual(t, de.Anterior()))
		igual(t, "", atual(t, Restaura(FilaVazia().Retrato())))
		igual(t, "", atual(t, Restaura(Retrato{Faixas: []Faixa{um}, Indice: 7, Repetir: RepetirNao})))
	})
}

func TestConsultaEBuscasRecentes(t *testing.T) {
	c, ok := Consulta("  lilriog \n")
	igual(t, "lilriog", c)
	igual(t, true, ok)
	_, ok = Consulta("   ")
	igual(t, false, ok)
	_, ok = Consulta(strings.Repeat("a", TamanhoMaxConsulta+1))
	igual(t, false, ok)
	c, _ = Consulta(strings.Repeat("a", TamanhoMaxConsulta))
	igual(t, TamanhoMaxConsulta, len(c))

	historico := []string{"b", "Daft Punk", "c", "d", "e", "f"}
	igual(t, []string{"daft punk", "b", "c", "d", "e", "f"}, LembraBusca(historico, "  daft punk "))
	igual(t, []string{"new", "b", "Daft Punk", "c", "d", "e"}, LembraBusca(historico, "new"))
	igual(t, []string{"a"}, LembraBusca([]string{"a"}, "   "))
}

func TestOuvida(t *testing.T) {
	igual(t, false, Ouvida(29*time.Second, 4*time.Minute))
	igual(t, true, Ouvida(30*time.Second, 4*time.Minute))
	igual(t, false, Ouvida(19*time.Second, 40*time.Second))
	igual(t, true, Ouvida(20*time.Second, 40*time.Second))
	igual(t, true, Ouvida(30*time.Second, 0))
	igual(t, false, Ouvida(0, 0))
}

var agora = time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)

func tocada(f Faixa, dias int) Tocada { return Tocada{f, agora.Add(-time.Duration(dias) * dia)} }

func TestListasInteligentes(t *testing.T) {
	a, b, c := umaFaixa("a"), umaFaixa("b"), umaFaixa("c")
	t.Run("no repeat: as mais tocadas de trinta dias, o empate pela mais recente", func(t *testing.T) {
		igual(t, []string{"b", "a"}, ids(NoRepeat([]Tocada{tocada(a, 1), tocada(b, 2), tocada(b, 3), tocada(c, 40), tocada(c, 41), tocada(c, 42)}, agora)))
		igual(t, []string{"b", "a"}, ids(NoRepeat([]Tocada{tocada(a, 5), tocada(b, 1)}, agora)))
		var muitas []Tocada
		for i := range 30 {
			muitas = append(muitas, tocada(umaFaixa(string(rune('A'+i))), 1))
		}
		igual(t, 20, len(NoRepeat(muitas, agora)))
	})
	t.Run("tocadas recentemente: uma vez cada, a mais nova primeiro, numa semana", func(t *testing.T) {
		igual(t, []string{"a", "b"}, ids(TocadasRecentemente([]Tocada{tocada(a, 3), tocada(b, 1), tocada(a, 0), tocada(c, 8)}, agora)))
	})
	t.Run("redescobrir: ouvidas pela última vez há duas a quatro semanas", func(t *testing.T) {
		tocadas := []Tocada{tocada(a, 20), tocada(b, 25), tocada(c, 20), tocada(c, 2), tocada(umaFaixa("old"), 31)}
		igual(t, []string{"b", "a"}, ids(Redescobrir(tocadas, agora)))
	})
	t.Run("a faixa fica com os detalhes da última tocada", func(t *testing.T) {
		renomeada := a
		renomeada.Titulo = "Novo título"
		igual(t, "Novo título", TocadasRecentemente([]Tocada{tocada(a, 3), tocada(renomeada, 1)}, agora)[0].Titulo)
	})
}

func TestResumoDeEscuta(t *testing.T) {
	um, dois := umaFaixa("1"), umaFaixa("2")
	um.Artista, um.Duracao = "LilRioG", 2*time.Minute
	dois.Artista, dois.Duracao = "One Saint", 0
	horas := func(h int) time.Time { return agora.Add(-time.Duration(h) * time.Hour) }
	r := ResumoDe([]Tocada{{um, horas(1)}, {um, horas(2)}, {dois, horas(3)}, {dois, horas(30)}}, PeriodoDia, agora)
	igual(t, 3, r.Tocadas)
	igual(t, 4*time.Minute, r.Duracao)
	igual(t, []Contagem{{Nome: "LilRioG", Vezes: 2}, {Nome: "One Saint", Vezes: 1}}, r.Artistas)
	igual(t, []string{"1", "2"}, []string{r.Faixas[0].Faixa.ID, r.Faixas[1].Faixa.ID})
	igual(t, []int{2, 1}, []int{r.Faixas[0].Vezes, r.Faixas[1].Vezes})
	p, ok := PeriodoDe("month")
	igual(t, true, ok)
	igual(t, 30*dia, p.Janela())
}

func TestPlaylistLocal(t *testing.T) {
	a, b, c := umaFaixa("a"), umaFaixa("b"), umaFaixa("c")
	a.Capa, b.Capa, c.Capa = "https://art/a", "https://art/b", "https://art/a"
	vazia := PlaylistLocal{ID: "p1", Titulo: "Estrada"}
	igual(t, []string{"a", "b"}, ids(vazia.Adiciona(a).Adiciona(b).Adiciona(a).Faixas))
	igual(t, true, vazia.Adiciona(a).Contem("a"))
	igual(t, false, vazia.Adiciona(a).Contem("b"))
	cheia := vazia.Adiciona(a).Adiciona(b).Adiciona(c)
	igual(t, []string{"a", "c"}, ids(cheia.Remove(1).Faixas))
	igual(t, []string{"c", "a", "b"}, ids(cheia.MoveFaixa(2, 0).Faixas))
	igual(t, []string{"a", "b", "c"}, ids(cheia.MoveFaixa(0, 9).Faixas))
	igual(t, []string{"https://art/a", "https://art/b"}, cheia.Capas())
	igual(t, "Viagem", cheia.Renomeia(" Viagem ").Titulo)
	igual(t, "Estrada", cheia.Renomeia("  ").Titulo)

	titulo, ok := TituloDePlaylist("  Estrada ")
	igual(t, "Estrada", titulo)
	igual(t, true, ok)
	_, ok = TituloDePlaylist("   ")
	igual(t, false, ok)
	titulo, _ = TituloDePlaylist(strings.Repeat("x", 300))
	igual(t, TamanhoMaxTitulo, len(titulo))
}

func TestCaminhoDoDownload(t *testing.T) {
	musica := umaFaixa("W40W7L1fhUM")
	musica.Titulo, musica.Artista = "Até Quando", "Natanzinho Lima"
	com := func(titulo, artista string) Faixa {
		f := musica
		f.Titulo, f.Artista = titulo, artista
		return f
	}
	nome := func(f Faixa) string {
		caminho := CaminhoDoDownload(f, "m4a")
		caminho = caminho[strings.Index(caminho, "/")+1:]
		return caminho[:strings.Index(caminho, " [")]
	}
	igual(t, "Natanzinho Lima/Até Quando [W40W7L1fhUM].m4a", CaminhoDoDownload(musica, "m4a"))
	igual(t, "Kai_Drako/AC_DC_ _Live___ [W40W7L1fhUM].m4a", CaminhoDoDownload(com(`AC/DC: "Live"?*`, "Kai/Drako"), "m4a"))
	igual(t, "Artista desconhecido/Até Quando [W40W7L1fhUM].m4a", CaminhoDoDownload(com("Até Quando", " "), "m4a"))
	longo := nome(com(strings.Repeat("é", 100), "x"))
	igual(t, 120, len(longo))
	igual(t, 60, utf8.RuneCountInString(longo))
	igual(t, 30, utf8.RuneCountInString(nome(com(strings.Repeat("😀", 60), "x"))))
	igual(t, "_/_ [W40W7L1fhUM].m4a", CaminhoDoDownload(com(".", ".."), "m4a"))
	igual(t, true, strings.HasPrefix(CaminhoDoDownload(com("x", ".hack"), "m4a"), "_hack/"))
	// O id vem da janela: não pode levar o arquivo para outra pasta.
	fuga := musica
	fuga.ID = "../../x"
	igual(t, 1, strings.Count(CaminhoDoDownload(fuga, "m4a"), "/"))
}

func TestLetras(t *testing.T) {
	lrc := "[ar:Ninguém]\n[00:01.50] primeira linha\n[00:04.00] segunda linha\n[00:04.00]\n[01:02.345] terceira linha\ntexto sem tempo"
	letra := LeLRC(lrc)
	igual(t, Letra{
		{1500 * time.Millisecond, "primeira linha"},
		{4 * time.Second, "segunda linha"},
		{4 * time.Second, ""},
		{62345 * time.Millisecond, "terceira linha"},
	}, letra)
	varias := LeLRC("[00:10.00][00:30.00] refrão")
	igual(t, []time.Duration{10 * time.Second, 30 * time.Second}, []time.Duration{varias[0].Tempo, varias[1].Tempo})
	igual(t, 0, len(LeLRC("só texto\nsem tempo")))
	igual(t, -1, letra.VersoAtivo(time.Second))
	igual(t, 0, letra.VersoAtivo(2*time.Second))
	igual(t, 2, letra.VersoAtivo(4*time.Second))
	igual(t, 3, letra.VersoAtivo(5*time.Minute))
}

func TestNormalizacaoEAjustes(t *testing.T) {
	db := func(v float64) *float64 { return &v }
	perto := func(esperado, obtido float64) {
		t.Helper()
		if math.Abs(esperado-obtido) > 0.001 {
			t.Fatalf("esperado %v, obtido %v", esperado, obtido)
		}
	}
	perto(0.463, VolumePara(db(6.68)))
	perto(1, VolumePara(db(-4)))
	perto(0.063, VolumePara(db(40)))
	perto(1, VolumePara(nil))

	igual(t, Ajustes{Autoplay: true, Qualidade: QualidadeAlta, Trabalhadores: 1, Normalizar: true, OndaSonora: true}, AjustesPadrao())
	igual(t, 4, AjustesPadrao().ComTrabalhadores(9).Trabalhadores)
	igual(t, 1, AjustesPadrao().ComTrabalhadores(0).Trabalhadores)
	igual(t, 3, AjustesPadrao().ComTrabalhadores(3).Trabalhadores)
	q, ok := QualidadeDe("economy")
	igual(t, QualidadeEconomia, q)
	igual(t, true, ok)
	_, ok = ProvedorDe("MYSPACE")
	igual(t, false, ok)
	igual(t, true, Faixa{Provedor: Radio}.AoVivo())
}
