// SPDX-License-Identifier: GPL-3.0-or-later

package guarda

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"

	"mktunes/internal/app"
	"mktunes/internal/dominio"
)

// Versao é a versão do formato dos arquivos da guarda.
const Versao = 1

const (
	arqBiblioteca   = "biblioteca.json"
	arqPreferencias = "preferencias.json"
	arqSessao       = "sessao.json"
	arqDownloads    = "downloads.json"
	arqHistorico    = "historico.jsonl"
)

type curtidaDTO struct {
	faixaDTO
	LikedAtMs int64 `json:"likedAtMs"`
}

type bibliotecaDTO struct {
	Versao    int           `json:"versao"`
	Liked     []curtidaDTO  `json:"liked"`
	Playlists []playlistDTO `json:"playlists"`
	Saved     []salvoDTO    `json:"saved"`
}

type preferenciasDTO struct {
	Versao   int         `json:"versao"`
	Settings *ajustesDTO `json:"settings"`
	Searches []string    `json:"searches"`
}

type sessaoDTO struct {
	Versao     int        `json:"versao"`
	Tracks     []faixaDTO `json:"tracks"`
	Index      *int       `json:"index"`
	Repeat     string     `json:"repeat"`
	PositionMs int64      `json:"positionMs"`
}

type downloadDTO struct {
	Track       faixaDTO `json:"track"`
	Status      string   `json:"status"`
	Progress    float64  `json:"progress"`
	Path        string   `json:"path,omitempty"`
	Reason      string   `json:"reason,omitempty"`
	CreatedAtMs int64    `json:"createdAtMs"`
}

type downloadsDTO struct {
	Versao int           `json:"versao"`
	Jobs   []downloadDTO `json:"jobs"`
}

// Guarda é a biblioteca do ouvinte no disco. A biblioteca e as preferências
// ficam também na memória (a janela as vê inteiras a cada mudança); a sessão,
// os downloads e o histórico só no disco. Só o laço a chama.
type Guarda struct {
	pasta string
	// AoErrar recebe as falhas de escrita (disco cheio, pasta sem permissão).
	AoErrar func(erro error)

	biblioteca   bibliotecaDTO
	preferencias preferenciasDTO
}

// Abre lê a guarda da pasta (criando-a). Um arquivo estragado é posto de lado
// (".danificado") e a parte dele começa vazia; um arquivo de uma versão mais
// nova é um erro: nada é sobrescrito.
func Abre(pasta string) (*Guarda, error) {
	if erro := os.MkdirAll(pasta, 0o700); erro != nil {
		return nil, erro
	}
	g := &Guarda{pasta: pasta, AoErrar: func(erro error) { fmt.Fprintln(os.Stderr, "mktunesd: guarda:", erro) }}
	if erro := g.le(arqBiblioteca, &g.biblioteca, &g.biblioteca.Versao); erro != nil {
		return nil, erro
	}
	if erro := g.le(arqPreferencias, &g.preferencias, &g.preferencias.Versao); erro != nil {
		return nil, erro
	}
	for _, arquivo := range []string{arqSessao, arqDownloads} {
		var cabeca struct {
			Versao int `json:"versao"`
		}
		if erro := g.le(arquivo, &cabeca, &cabeca.Versao); erro != nil {
			return nil, erro
		}
	}
	return g, nil
}

func (g *Guarda) caminho(nome string) string { return filepath.Join(g.pasta, nome) }

// le carrega um arquivo em alvo. Não existir é normal.
func (g *Guarda) le(nome string, alvo any, versao *int) error {
	bruto, erro := os.ReadFile(g.caminho(nome))
	if errors.Is(erro, fs.ErrNotExist) {
		return nil
	}
	if erro != nil {
		return erro
	}
	if json.Unmarshal(bruto, alvo) != nil {
		lado := fmt.Sprintf("%s.danificado-%d", g.caminho(nome), time.Now().Unix())
		g.AoErrar(fmt.Errorf("%s está estragado; ficou em %s e esta parte começa vazia", nome, lado))
		return os.Rename(g.caminho(nome), lado)
	}
	if *versao > Versao {
		return fmt.Errorf("%s é de uma versão mais nova do MkTunes (formato %d; este lê até %d)", g.caminho(nome), *versao, Versao)
	}
	return nil
}

func (g *Guarda) escreve(nome string, valor any, firme bool) {
	if erro := escreveAtomico(g.caminho(nome), emJSON(valor), firme); erro != nil {
		g.AoErrar(erro)
	}
}

func (g *Guarda) gravaBiblioteca() {
	g.biblioteca.Versao = Versao
	g.escreve(arqBiblioteca, g.biblioteca, true)
}

func (g *Guarda) gravaPreferencias() {
	g.preferencias.Versao = Versao
	g.escreve(arqPreferencias, g.preferencias, true)
}

func (g *Guarda) Curtidas() []dominio.Faixa {
	faixas := make([]dominio.Faixa, 0, len(g.biblioteca.Liked))
	for _, c := range g.biblioteca.Liked {
		if f, ok := c.faixa(); ok {
			faixas = append(faixas, f)
		}
	}
	return faixas
}

func (g *Guarda) Curte(f dominio.Faixa, curtida bool, em time.Time) {
	outras := make([]curtidaDTO, 0, len(g.biblioteca.Liked)+1)
	if curtida {
		outras = append(outras, curtidaDTO{paraDTO(f), em.UnixMilli()})
	}
	for _, c := range g.biblioteca.Liked {
		if c.ID != f.ID {
			outras = append(outras, c)
		}
	}
	g.biblioteca.Liked = outras
	g.gravaBiblioteca()
}

func (g *Guarda) Playlists() []dominio.PlaylistLocal {
	listas := make([]dominio.PlaylistLocal, 0, len(g.biblioteca.Playlists))
	for _, p := range g.biblioteca.Playlists {
		listas = append(listas, dominio.PlaylistLocal{ID: p.ID, Titulo: p.Title, Faixas: faixasDe(p.Tracks)})
	}
	return listas
}

func (g *Guarda) SalvaPlaylist(p dominio.PlaylistLocal, em time.Time) {
	dto := playlistDTO{ID: p.ID, Title: p.Titulo, CreatedAtMs: em.UnixMilli(), Tracks: paraDTOs(p.Faixas)}
	achou := false
	for i := range g.biblioteca.Playlists {
		if g.biblioteca.Playlists[i].ID == p.ID {
			dto.CreatedAtMs = g.biblioteca.Playlists[i].CreatedAtMs
			g.biblioteca.Playlists[i] = dto
			achou = true
		}
	}
	if !achou {
		g.biblioteca.Playlists = append(g.biblioteca.Playlists, dto)
	}
	g.gravaBiblioteca()
}

func (g *Guarda) ApagaPlaylist(id string) {
	resto := g.biblioteca.Playlists[:0:0]
	for _, p := range g.biblioteca.Playlists {
		if p.ID != id {
			resto = append(resto, p)
		}
	}
	g.biblioteca.Playlists = resto
	g.gravaBiblioteca()
}

func (g *Guarda) Salvos() []dominio.ColecaoSalva {
	salvos := make([]dominio.ColecaoSalva, 0, len(g.biblioteca.Saved))
	for _, s := range g.biblioteca.Saved {
		if c, ok := s.salvo(); ok {
			salvos = append(salvos, c)
		}
	}
	return salvos
}

func (g *Guarda) SalvaColecao(c dominio.ColecaoSalva) {
	g.poeSalvo(salvoParaDTO(c))
	g.gravaBiblioteca()
}

// poeSalvo mantém a ordem: o salvo mais recente primeiro; no empate, o que
// entrou por último.
func (g *Guarda) poeSalvo(novo salvoDTO) {
	lista := []salvoDTO{novo}
	for _, s := range g.biblioteca.Saved {
		if s.ID != novo.ID {
			lista = append(lista, s)
		}
	}
	sort.SliceStable(lista, func(i, j int) bool { return lista[i].SavedAtMs > lista[j].SavedAtMs })
	g.biblioteca.Saved = lista
}

func (g *Guarda) EsqueceColecao(id string) {
	resto := g.biblioteca.Saved[:0:0]
	for _, s := range g.biblioteca.Saved {
		if s.ID != id {
			resto = append(resto, s)
		}
	}
	g.biblioteca.Saved = resto
	g.gravaBiblioteca()
}

func (g *Guarda) Ajustes() dominio.Ajustes { return g.preferencias.Settings.ajustes() }

func (g *Guarda) SalvaAjustes(a dominio.Ajustes) {
	g.preferencias.Settings = ajustesParaDTO(a)
	g.gravaPreferencias()
}

func (g *Guarda) Buscas() []string { return append([]string(nil), g.preferencias.Searches...) }

func (g *Guarda) SalvaBuscas(b []string) {
	g.preferencias.Searches = append([]string{}, b...)
	g.gravaPreferencias()
}

func (g *Guarda) SalvaSessao(s app.SessaoSalva) {
	dto := sessaoDTO{Versao: Versao, Tracks: paraDTOs(s.Fila.Faixas), Repeat: string(s.Fila.Repetir), PositionMs: s.Posicao.Milliseconds()}
	if s.Fila.Indice >= 0 {
		indice := s.Fila.Indice
		dto.Index = &indice
	}
	g.escreve(arqSessao, dto, false)
}

func (g *Guarda) CarregaSessao() (app.SessaoSalva, bool) {
	bruto, erro := os.ReadFile(g.caminho(arqSessao))
	var dto sessaoDTO
	if erro != nil || json.Unmarshal(bruto, &dto) != nil {
		return app.SessaoSalva{}, false
	}
	return dto.sessao(), true
}

func (d sessaoDTO) sessao() app.SessaoSalva {
	// Uma faixa de provedor desconhecido some da lista; o índice só vale se a
	// lista veio inteira.
	faixas := faixasDe(d.Tracks)
	indice := -1
	if d.Index != nil && len(faixas) == len(d.Tracks) {
		indice = *d.Index
	}
	repetir, ok := dominio.RepetirDe(d.Repeat)
	if !ok {
		repetir = dominio.RepetirNao
	}
	return app.SessaoSalva{Fila: dominio.Retrato{Faixas: faixas, Indice: indice, Repetir: repetir}, Posicao: time.Duration(d.PositionMs) * time.Millisecond}
}

func downloadParaDTO(d dominio.Download) downloadDTO {
	return downloadDTO{paraDTO(d.Faixa), string(d.Estado), d.Progresso, d.Caminho, d.Motivo, d.CriadoEm.UnixMilli()}
}

func (g *Guarda) leDownloads() downloadsDTO {
	var dto downloadsDTO
	if bruto, erro := os.ReadFile(g.caminho(arqDownloads)); erro == nil {
		_ = json.Unmarshal(bruto, &dto)
	}
	return dto
}

func (g *Guarda) gravaDownloads(dto downloadsDTO) {
	dto.Versao = Versao
	g.escreve(arqDownloads, dto, false)
}

func (g *Guarda) SalvaDownload(d dominio.Download) {
	dto := g.leDownloads()
	novo := downloadParaDTO(d)
	achou := false
	for i := range dto.Jobs {
		if dto.Jobs[i].Track.ID == d.Faixa.ID {
			dto.Jobs[i] = novo
			achou = true
		}
	}
	if !achou {
		dto.Jobs = append(dto.Jobs, novo)
	}
	g.gravaDownloads(dto)
}

func (g *Guarda) Downloads() []dominio.Download {
	var lista []dominio.Download
	for _, j := range g.leDownloads().Jobs {
		f, ok := j.Track.faixa()
		if !ok {
			continue
		}
		d := dominio.Download{Faixa: f, Estado: dominio.EstadoDownload(j.Status), Progresso: j.Progress, Caminho: j.Path, Motivo: j.Reason, CriadoEm: time.UnixMilli(j.CreatedAtMs).UTC()}
		switch d.Estado {
		case dominio.NaFila, dominio.Baixando, dominio.Concluido, dominio.Falhou:
		default:
			d.Estado = dominio.Cancelado
		}
		lista = append(lista, d)
	}
	sort.SliceStable(lista, func(i, j int) bool { return lista[i].CriadoEm.After(lista[j].CriadoEm) })
	return lista
}

func (g *Guarda) RemoveDownload(id string) {
	dto := g.leDownloads()
	resto := dto.Jobs[:0:0]
	for _, j := range dto.Jobs {
		if j.Track.ID != id {
			resto = append(resto, j)
		}
	}
	dto.Jobs = resto
	g.gravaDownloads(dto)
}

func (g *Guarda) RegistraTocada(t dominio.Tocada) {
	g.acrescentaTocadas([]tocadaDTO{{paraDTO(t.Faixa), t.Em.UnixMilli()}})
}

func (g *Guarda) acrescentaTocadas(tocadas []tocadaDTO) {
	if len(tocadas) == 0 {
		return
	}
	var linhas bytes.Buffer
	for _, t := range tocadas {
		linhas.Write(emJSON(t))
		linhas.WriteByte('\n')
	}
	arquivo, erro := os.OpenFile(g.caminho(arqHistorico), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if erro == nil {
		_, erro = arquivo.Write(linhas.Bytes())
		if fecha := arquivo.Close(); erro == nil {
			erro = fecha
		}
	}
	if erro != nil {
		g.AoErrar(erro)
	}
}

// leHistorico passa por cada tocada do arquivo; uma linha estragada é pulada.
func (g *Guarda) leHistorico(cada func(t tocadaDTO)) {
	arquivo, erro := os.Open(g.caminho(arqHistorico))
	if erro != nil {
		return
	}
	defer arquivo.Close()
	linhas := bufio.NewScanner(arquivo)
	linhas.Buffer(make([]byte, 64*1024), 1024*1024)
	for linhas.Scan() {
		var t tocadaDTO
		if json.Unmarshal(linhas.Bytes(), &t) == nil && t.Track.ID != "" {
			cada(t)
		}
	}
}

func (g *Guarda) TocadasDesde(inicio time.Time) []dominio.Tocada {
	corte := inicio.UnixMilli()
	var tocadas []dominio.Tocada
	g.leHistorico(func(t tocadaDTO) {
		if t.AtMs < corte {
			return
		}
		if f, ok := t.Track.faixa(); ok {
			tocadas = append(tocadas, dominio.Tocada{Faixa: f, Em: time.UnixMilli(t.AtMs).UTC()})
		}
	})
	sort.SliceStable(tocadas, func(i, j int) bool { return tocadas[i].Em.Before(tocadas[j].Em) })
	return tocadas
}

func (g *Guarda) EsqueceTocadasAntesDe(corte time.Time) {
	limite := corte.UnixMilli()
	var ficam bytes.Buffer
	saiu := false
	g.leHistorico(func(t tocadaDTO) {
		if t.AtMs < limite {
			saiu = true
			return
		}
		ficam.Write(emJSON(t))
		ficam.WriteByte('\n')
	})
	if saiu {
		if erro := escreveAtomico(g.caminho(arqHistorico), ficam.Bytes(), false); erro != nil {
			g.AoErrar(erro)
		}
	}
}
