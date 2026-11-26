// SPDX-License-Identifier: GPL-3.0-or-later

package app

import (
	"crypto/rand"
	"encoding/hex"
	"sort"
	"strings"
	"time"

	"mktunes/internal/dominio"
)

// Acervo é a biblioteca do ouvinte neste computador: curtidas, listas
// inteligentes, playlists, coleções salvas, buscas recentes, ajustes e o
// backup. Tudo passa pela guarda; Mudou avisa a cada alteração.
type Acervo struct {
	guarda Guarda
	codec  Codec
	agora  func() time.Time
	// NovoID dá o id de uma playlist nova (os testes trocam).
	NovoID func() string
	Mudou  Sinal
}

// NovoAcervo liga a biblioteca à guarda.
func NovoAcervo(g Guarda, c Codec, agora func() time.Time) *Acervo {
	return &Acervo{guarda: g, codec: c, agora: agora, NovoID: idAleatorio}
}

func idAleatorio() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b[:4]) + "-" + hex.EncodeToString(b[4:6]) + "-" + hex.EncodeToString(b[6:8]) + "-" + hex.EncodeToString(b[8:10]) + "-" + hex.EncodeToString(b[10:])
}

// Curtidas, a mais recente primeiro.
func (a *Acervo) Curtidas() []dominio.Faixa { return a.guarda.Curtidas() }

// Curtida diz se a faixa está entre as curtidas.
func (a *Acervo) Curtida(id string) bool {
	for _, f := range a.guarda.Curtidas() {
		if f.ID == id {
			return true
		}
	}
	return false
}

// AlternaCurtida curte ou descurte; devolve se a faixa era curtida.
func (a *Acervo) AlternaCurtida(f dominio.Faixa) bool {
	era := a.Curtida(f.ID)
	a.guarda.Curte(f, !era, a.agora())
	a.Mudou.Avisa()
	return era
}

// ListasInteligentes são as três listas feitas do histórico.
type ListasInteligentes struct{ NoRepeat, Recentes, Redescobrir []dominio.Faixa }

// Listas monta as listas inteligentes de agora.
func (a *Acervo) Listas() ListasInteligentes {
	agora := a.agora()
	tocadas := a.guarda.TocadasDesde(agora.Add(-dominio.JanelaHistorico))
	return ListasInteligentes{
		NoRepeat:    dominio.NoRepeat(tocadas, agora),
		Recentes:    dominio.TocadasRecentemente(tocadas, agora),
		Redescobrir: dominio.Redescobrir(tocadas, agora),
	}
}

// Resumo é o que foi ouvido no período.
func (a *Acervo) Resumo(p dominio.Periodo) dominio.Resumo {
	agora := a.agora()
	return dominio.ResumoDe(a.guarda.TocadasDesde(agora.Add(-p.Janela())), p, agora)
}

// Playlists são as do ouvinte, na ordem em que foram criadas.
func (a *Acervo) Playlists() []dominio.PlaylistLocal { return a.guarda.Playlists() }

func (a *Acervo) playlist(id string) (dominio.PlaylistLocal, bool) {
	for _, p := range a.guarda.Playlists() {
		if p.ID == id {
			return p, true
		}
	}
	return dominio.PlaylistLocal{}, false
}

func (a *Acervo) salvaPlaylist(p dominio.PlaylistLocal) {
	a.guarda.SalvaPlaylist(p, a.agora())
	a.Mudou.Avisa()
}

// CriaPlaylist cria uma playlist (já com uma faixa, se vier); devolve o id, ou
// false quando o título está em branco.
func (a *Acervo) CriaPlaylist(titulo string, com *dominio.Faixa) (string, bool) {
	limpo, ok := dominio.TituloDePlaylist(titulo)
	if !ok {
		return "", false
	}
	p := dominio.PlaylistLocal{ID: a.NovoID(), Titulo: limpo}
	if com != nil {
		p = p.Adiciona(*com)
	}
	a.salvaPlaylist(p)
	return p.ID, true
}

// AdicionaAPlaylist devolve false quando a faixa já estava lá (ou a playlist
// não existe).
func (a *Acervo) AdicionaAPlaylist(id string, f dominio.Faixa) bool {
	p, tem := a.playlist(id)
	if !tem || p.Contem(f.ID) {
		return false
	}
	a.salvaPlaylist(p.Adiciona(f))
	return true
}

// EditaPlaylist aplica uma mudança a uma playlist que existe.
func (a *Acervo) EditaPlaylist(id string, muda func(dominio.PlaylistLocal) dominio.PlaylistLocal) {
	if p, tem := a.playlist(id); tem {
		a.salvaPlaylist(muda(p))
	}
}

// ApagaPlaylist apaga a playlist.
func (a *Acervo) ApagaPlaylist(id string) {
	a.guarda.ApagaPlaylist(id)
	a.Mudou.Avisa()
}

// Salvos são os álbuns e playlists do catálogo guardados, o mais novo primeiro.
func (a *Acervo) Salvos() []dominio.ColecaoSalva { return a.guarda.Salvos() }

// Salvo diz se a coleção está guardada.
func (a *Acervo) Salvo(id string) bool {
	for _, c := range a.guarda.Salvos() {
		if c.ID == id {
			return true
		}
	}
	return false
}

// AlternaSalvo guarda ou tira a coleção da biblioteca; devolve se estava salva.
func (a *Acervo) AlternaSalvo(p dominio.PaginaColecao) bool {
	estava := a.Salvo(p.ID)
	if estava {
		a.guarda.EsqueceColecao(p.ID)
	} else {
		dono := p.Dono
		if strings.TrimSpace(dono) == "" {
			dono = "Playlist"
			if p.Tipo == dominio.Album {
				dono = "Álbum"
			}
		}
		a.guarda.SalvaColecao(dominio.ColecaoSalva{ID: p.ID, Tipo: p.Tipo, Titulo: p.Titulo, Dono: dono, Capa: p.Capa, SalvaEm: a.agora()})
	}
	a.Mudou.Avisa()
	return estava
}

// Buscas são as buscas recentes.
func (a *Acervo) Buscas() []string { return a.guarda.Buscas() }

// LembraBusca põe a busca no topo das recentes e devolve a lista nova.
func (a *Acervo) LembraBusca(consulta string) []string {
	novo := dominio.LembraBusca(a.guarda.Buscas(), consulta)
	a.guarda.SalvaBuscas(novo)
	return novo
}

// LimpaBuscas esquece as buscas recentes.
func (a *Acervo) LimpaBuscas() { a.guarda.SalvaBuscas(nil) }

// Ajustes são as preferências de agora.
func (a *Acervo) Ajustes() dominio.Ajustes { return a.guarda.Ajustes() }

// MudaAjustes aplica e guarda uma mudança nas preferências.
func (a *Acervo) MudaAjustes(muda func(dominio.Ajustes) dominio.Ajustes) {
	a.guarda.SalvaAjustes(muda(a.guarda.Ajustes()))
	a.Mudou.Avisa()
}

// Exporta escreve a biblioteca inteira como texto de backup.
func (a *Acervo) Exporta() string {
	return a.codec.Codifica(dominio.Backup{
		CriadoEm:  a.agora(),
		Curtidas:  a.guarda.Curtidas(),
		Playlists: a.guarda.Playlists(),
		Tocadas:   a.guarda.TocadasDesde(time.Unix(0, 0)),
		Ajustes:   a.guarda.Ajustes(),
		Buscas:    a.guarda.Buscas(),
		Salvos:    a.guarda.Salvos(),
	})
}

// Restaurado conta o que era novidade para este computador.
type Restaurado struct{ Curtidas, Playlists, Tocadas int }

// Restaura traz um backup de volta, juntando ao que já existe em vez de
// substituir. Devolve false quando o texto não é um backup.
func (a *Acervo) Restaura(texto string) (Restaurado, bool) {
	backup, ok := a.codec.Decodifica(texto)
	if !ok {
		return Restaurado{}, false
	}
	var r Restaurado
	agora := a.agora()
	// De trás para a frente: a ordem do backup fica no topo das curtidas.
	for i := len(backup.Curtidas) - 1; i >= 0; i-- {
		if f := backup.Curtidas[i]; !a.Curtida(f.ID) {
			a.guarda.Curte(f, true, agora)
			r.Curtidas++
		}
	}
	type chave struct {
		id string
		em int64
	}
	conhecidas := map[chave]bool{}
	for _, t := range a.guarda.TocadasDesde(time.Unix(0, 0)) {
		conhecidas[chave{t.Faixa.ID, t.Em.UnixMilli()}] = true
	}
	for _, t := range backup.Tocadas {
		if c := (chave{t.Faixa.ID, t.Em.UnixMilli()}); !conhecidas[c] {
			conhecidas[c] = true
			a.guarda.RegistraTocada(t)
			r.Tocadas++
		}
	}
	for _, salva := range backup.Playlists {
		// Uma playlist que já está aqui fica com o nome e a ordem; entram só as
		// faixas que só o backup tem.
		aqui, tem := a.playlist(salva.ID)
		if !tem {
			r.Playlists++
			a.guarda.SalvaPlaylist(salva, agora)
			continue
		}
		for _, f := range salva.Faixas {
			aqui = aqui.Adiciona(f)
		}
		a.guarda.SalvaPlaylist(aqui, agora)
	}
	novos := append([]dominio.ColecaoSalva(nil), backup.Salvos...)
	sort.SliceStable(novos, func(i, j int) bool { return novos[i].SalvaEm.Before(novos[j].SalvaEm) })
	for _, c := range novos {
		if !a.Salvo(c.ID) {
			a.guarda.SalvaColecao(c)
		}
	}
	a.guarda.SalvaAjustes(backup.Ajustes)
	var buscas []string
	vistas := map[string]bool{}
	for _, b := range append(a.guarda.Buscas(), backup.Buscas...) {
		if c := strings.ToLower(b); !vistas[c] && len(buscas) < dominio.BuscasRecentes {
			vistas[c] = true
			buscas = append(buscas, b)
		}
	}
	a.guarda.SalvaBuscas(buscas)
	a.Mudou.Avisa()
	return r, true
}
