// SPDX-License-Identifier: GPL-3.0-or-later

// Pacote prova: os dublês que os testes do daemon dividem (uma faixa de
// exemplo, um tocador e um resolvedor de mentira, uma guarda na memória).
// Nada aqui entra no caminho de quem usa o app.
package prova

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"mktunes/internal/app"
	"mktunes/internal/dominio"
	"mktunes/internal/laco"
)

// Faixa é a faixa de exemplo dos testes; cada função em muda ajusta um campo.
func Faixa(id string, muda ...func(*dominio.Faixa)) dominio.Faixa {
	f := dominio.Faixa{ID: id, Provedor: dominio.Deezer, Titulo: "Track " + id, Artista: "Artist " + id, Duracao: 30 * time.Second, Stream: "https://cdn.example/" + id + ".mp3"}
	for _, m := range muda {
		m(&f)
	}
	return f
}

// Titulo, Artista e companhia são os ajustes mais comuns de Faixa.
func Titulo(t string) func(*dominio.Faixa)  { return func(f *dominio.Faixa) { f.Titulo = t } }
func Artista(a string) func(*dominio.Faixa) { return func(f *dominio.Faixa) { f.Artista = a } }
func SemStream(f *dominio.Faixa)            { f.Stream = "" }
func Provedor(p dominio.Provedor) func(*dominio.Faixa) {
	return func(f *dominio.Faixa) { f.Provedor = p }
}
func Duracao(d time.Duration) func(*dominio.Faixa) { return func(f *dominio.Faixa) { f.Duracao = d } }

// IDs lista os ids das faixas.
func IDs(faixas []dominio.Faixa) []string {
	ids := []string{}
	for _, f := range faixas {
		ids = append(ids, f.ID)
	}
	return ids
}

// Tocador é um tocador de mentira que anota o que pediram. Só o laço o usa.
// Com AoEvento ligado à sessão, cada faixa "carrega" na hora em que mandam
// tocar; com Demora, só quando o teste chama Carrega.
type Tocador struct {
	AoEvento func(e app.Evento)
	Demora   bool
	Chamadas []string
	Pos, Dur time.Duration
	Tocando  *dominio.Faixa
	Stream   app.Stream
	Tocados  []app.Stream
}

func (t *Tocador) anota(formato string, a ...any) {
	t.Chamadas = append(t.Chamadas, fmt.Sprintf(formato, a...))
}
func (t *Tocador) Posicao() time.Duration { return t.Pos }
func (t *Tocador) Duracao() time.Duration { return t.Dur }
func (t *Tocador) Toca(s app.Stream, f dominio.Faixa) {
	t.anota("play %s", s.URL)
	t.Tocando, t.Stream = &f, s
	t.Tocados = append(t.Tocados, s)
	if !t.Demora {
		t.Carrega()
	}
}

// Carrega conta à sessão que a faixa abriu.
func (t *Tocador) Carrega() {
	if t.AoEvento != nil {
		t.AoEvento(app.Evento{Tipo: app.Carregou})
	}
}
func (t *Tocador) Pausa()                    { t.anota("pause") }
func (t *Tocador) Segura(f dominio.Faixa)    { t.anota("hold %s", f.ID); t.Pos = 0 }
func (t *Tocador) Continua()                 { t.anota("resume") }
func (t *Tocador) VaiPara(p time.Duration)   { t.anota("seek %s", p); t.Pos = p }
func (t *Tocador) Para()                     { t.anota("stop"); t.Tocando = nil }
func (t *Tocador) Velocidade(v float64)      { t.anota("speed %v", v) }
func (t *Tocador) PularSilencio(ligado bool) { t.anota("skipSilence %v", ligado) }

// Com devolve só as chamadas que começam com o prefixo.
func (t *Tocador) Com(prefixo string) []string {
	achadas := []string{}
	for _, c := range t.Chamadas {
		if len(c) >= len(prefixo) && c[:len(prefixo)] == prefixo {
			achadas = append(achadas, c)
		}
	}
	return achadas
}

// Resolvedor responde na hora com https://stream.example/<id>, a não ser que a
// faixa esteja em Indisponiveis, tenha um portão em Pendentes, ou Falha diga
// outra coisa.
type Resolvedor struct {
	mu            sync.Mutex
	Indisponiveis map[string]bool
	Pendentes     map[string]*laco.Portao
	Falha         error
	LoudnessDb    *float64
	Base          string
}

func (r *Resolvedor) Resolve(ctx context.Context, f dominio.Faixa) (app.Stream, error) {
	r.mu.Lock()
	indisponivel, portao, falha, base := r.Indisponiveis[f.ID], r.Pendentes[f.ID], r.Falha, r.Base
	r.mu.Unlock()
	if falha != nil {
		return app.Stream{}, falha
	}
	if indisponivel {
		return app.Stream{}, fmt.Errorf("%w: bloqueada", app.ErrStream)
	}
	if portao != nil {
		if erro := portao.Espera(ctx); erro != nil {
			return app.Stream{}, erro
		}
	}
	if base == "" {
		base = "https://stream.example/"
	}
	return app.Stream{URL: base + f.ID, LoudnessDb: r.LoudnessDb, Volume: 1}, nil
}

// Muda troca os campos do resolvedor com segurança (ele é lido fora do laço).
func (r *Resolvedor) Muda(f func(r *Resolvedor)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	f(r)
}

// Memoria é uma guarda que só existe na memória.
type Memoria struct {
	Tocadas    []dominio.Tocada
	Gostei     []dominio.Faixa
	Listas     []dominio.PlaylistLocal
	Guardados  []dominio.ColecaoSalva
	Prefs      dominio.Ajustes
	Recentes   []string
	Sessao     *app.SessaoSalva
	SessoesSal int
	Baixados   []dominio.Download
}

// NovaMemoria começa com os ajustes de fábrica.
func NovaMemoria() *Memoria { return &Memoria{Prefs: dominio.AjustesPadrao()} }

func (m *Memoria) RegistraTocada(t dominio.Tocada) { m.Tocadas = append(m.Tocadas, t) }
func (m *Memoria) TocadasDesde(inicio time.Time) []dominio.Tocada {
	var dentro []dominio.Tocada
	for _, t := range m.Tocadas {
		if !t.Em.Before(inicio) {
			dentro = append(dentro, t)
		}
	}
	sort.SliceStable(dentro, func(i, j int) bool { return dentro[i].Em.Before(dentro[j].Em) })
	return dentro
}
func (m *Memoria) EsqueceTocadasAntesDe(corte time.Time) { m.Tocadas = m.TocadasDesde(corte) }
func (m *Memoria) Curtidas() []dominio.Faixa             { return append([]dominio.Faixa(nil), m.Gostei...) }
func (m *Memoria) Curte(f dominio.Faixa, curtida bool, _ time.Time) {
	var outras []dominio.Faixa
	for _, g := range m.Gostei {
		if g.ID != f.ID {
			outras = append(outras, g)
		}
	}
	if curtida {
		outras = append([]dominio.Faixa{f}, outras...)
	}
	m.Gostei = outras
}
func (m *Memoria) Playlists() []dominio.PlaylistLocal {
	return append([]dominio.PlaylistLocal(nil), m.Listas...)
}
func (m *Memoria) SalvaPlaylist(p dominio.PlaylistLocal, _ time.Time) {
	for i := range m.Listas {
		if m.Listas[i].ID == p.ID {
			m.Listas[i] = p
			return
		}
	}
	m.Listas = append(m.Listas, p)
}
func (m *Memoria) ApagaPlaylist(id string) {
	var resto []dominio.PlaylistLocal
	for _, p := range m.Listas {
		if p.ID != id {
			resto = append(resto, p)
		}
	}
	m.Listas = resto
}
func (m *Memoria) Salvos() []dominio.ColecaoSalva {
	return append([]dominio.ColecaoSalva(nil), m.Guardados...)
}
func (m *Memoria) SalvaColecao(c dominio.ColecaoSalva) {
	m.EsqueceColecao(c.ID)
	m.Guardados = append([]dominio.ColecaoSalva{c}, m.Guardados...)
}
func (m *Memoria) EsqueceColecao(id string) {
	var resto []dominio.ColecaoSalva
	for _, c := range m.Guardados {
		if c.ID != id {
			resto = append(resto, c)
		}
	}
	m.Guardados = resto
}
func (m *Memoria) Ajustes() dominio.Ajustes       { return m.Prefs }
func (m *Memoria) SalvaAjustes(a dominio.Ajustes) { m.Prefs = a }
func (m *Memoria) Buscas() []string               { return append([]string(nil), m.Recentes...) }
func (m *Memoria) SalvaBuscas(b []string)         { m.Recentes = append([]string(nil), b...) }
func (m *Memoria) SalvaSessao(s app.SessaoSalva)  { m.Sessao = &s; m.SessoesSal++ }
func (m *Memoria) CarregaSessao() (app.SessaoSalva, bool) {
	if m.Sessao == nil {
		return app.SessaoSalva{}, false
	}
	return *m.Sessao, true
}
func (m *Memoria) SalvaDownload(d dominio.Download) {
	for i := range m.Baixados {
		if m.Baixados[i].Faixa.ID == d.Faixa.ID {
			m.Baixados[i] = d
			return
		}
	}
	m.Baixados = append(m.Baixados, d)
}
func (m *Memoria) Downloads() []dominio.Download {
	return append([]dominio.Download(nil), m.Baixados...)
}
func (m *Memoria) RemoveDownload(id string) {
	var resto []dominio.Download
	for _, d := range m.Baixados {
		if d.Faixa.ID != id {
			resto = append(resto, d)
		}
	}
	m.Baixados = resto
}
