// SPDX-License-Identifier: GPL-3.0-or-later

// Pacote guarda: o que o ouvinte montou, em arquivos JSON na pasta de dados
// (~/.local/share/mktunes), escritos de forma atômica e com número de versão;
// o histórico de escuta é um arquivo de linhas. Também o backup (no mesmo
// formato do MkTunes em Kotlin), o Início guardado e a migração única do
// banco SQLite antigo.
package guarda

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"mktunes/internal/dominio"
)

// faixaDTO é uma faixa como fica no JSON: na sessão guardada, na biblioteca e
// nos backups (os nomes são os do backup da versão em Kotlin).
type faixaDTO struct {
	ID         string  `json:"id"`
	Provider   string  `json:"provider"`
	Title      string  `json:"title"`
	Artist     string  `json:"artist"`
	DurationMs int64   `json:"durationMs"`
	ArtworkURL *string `json:"artworkUrl"`
	StreamURL  *string `json:"streamUrl"`
}

func opcional(texto string) *string {
	if texto == "" {
		return nil
	}
	return &texto
}

func texto(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func paraDTO(f dominio.Faixa) faixaDTO {
	return faixaDTO{f.ID, string(f.Provedor), f.Titulo, f.Artista, f.Duracao.Milliseconds(), opcional(f.Capa), opcional(f.Stream)}
}

// faixa devolve false para um provedor que esta versão não conhece: a faixa é
// pulada, não um erro.
func (d faixaDTO) faixa() (dominio.Faixa, bool) {
	provedor, ok := dominio.ProvedorDe(d.Provider)
	if !ok || d.ID == "" {
		return dominio.Faixa{}, false
	}
	return dominio.Faixa{ID: d.ID, Provedor: provedor, Titulo: d.Title, Artista: d.Artist, Duracao: time.Duration(d.DurationMs) * time.Millisecond, Capa: texto(d.ArtworkURL), Stream: texto(d.StreamURL)}, true
}

func paraDTOs(faixas []dominio.Faixa) []faixaDTO {
	dtos := make([]faixaDTO, 0, len(faixas))
	for _, f := range faixas {
		dtos = append(dtos, paraDTO(f))
	}
	return dtos
}

func faixasDe(dtos []faixaDTO) []dominio.Faixa {
	faixas := make([]dominio.Faixa, 0, len(dtos))
	for _, d := range dtos {
		if f, ok := d.faixa(); ok {
			faixas = append(faixas, f)
		}
	}
	return faixas
}

type ajustesDTO struct {
	Autoplay        bool   `json:"autoplay"`
	AudioQuality    string `json:"audioQuality"`
	DownloadWorkers int    `json:"downloadWorkers"`
	NormalizeVolume bool   `json:"normalizeVolume"`
	SkipSilence     bool   `json:"skipSilence"`
	SoundWave       *bool  `json:"soundWave"`
}

func ajustesParaDTO(a dominio.Ajustes) *ajustesDTO {
	onda := a.OndaSonora
	return &ajustesDTO{a.Autoplay, string(a.Qualidade), a.Trabalhadores, a.Normalizar, a.PularSilencio, &onda}
}

func (d *ajustesDTO) ajustes() dominio.Ajustes {
	if d == nil {
		return dominio.AjustesPadrao()
	}
	qualidade, ok := dominio.QualidadeDe(d.AudioQuality)
	if !ok || d.AudioQuality != string(qualidade) {
		qualidade = dominio.QualidadeAlta
	}
	a := dominio.Ajustes{Autoplay: d.Autoplay, Qualidade: qualidade, Normalizar: d.NormalizeVolume, PularSilencio: d.SkipSilence, OndaSonora: d.SoundWave == nil || *d.SoundWave}
	return a.ComTrabalhadores(d.DownloadWorkers)
}

type salvoDTO struct {
	ID         string  `json:"id"`
	Kind       string  `json:"kind"`
	Title      string  `json:"title"`
	Owner      string  `json:"owner"`
	ArtworkURL *string `json:"artworkUrl"`
	SavedAtMs  int64   `json:"savedAtMs"`
}

func salvoParaDTO(c dominio.ColecaoSalva) salvoDTO {
	return salvoDTO{c.ID, string(c.Tipo), c.Titulo, c.Dono, opcional(c.Capa), c.SalvaEm.UnixMilli()}
}

func (d salvoDTO) salvo() (dominio.ColecaoSalva, bool) {
	tipo, ok := dominio.TipoColecaoDe(d.Kind)
	if !ok {
		return dominio.ColecaoSalva{}, false
	}
	return dominio.ColecaoSalva{ID: d.ID, Tipo: tipo, Titulo: d.Title, Dono: d.Owner, Capa: texto(d.ArtworkURL), SalvaEm: time.UnixMilli(d.SavedAtMs).UTC()}, true
}

type tocadaDTO struct {
	Track faixaDTO `json:"track"`
	AtMs  int64    `json:"atMs"`
}

type playlistDTO struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	CreatedAtMs int64      `json:"createdAtMs,omitempty"`
	Tracks      []faixaDTO `json:"tracks"`
}

const formatoBackup = "mktunes-backup"

type backupDTO struct {
	Format      string        `json:"format"`
	Version     int           `json:"version"`
	CreatedAtMs int64         `json:"createdAtMs"`
	Liked       []faixaDTO    `json:"liked"`
	Playlists   []playlistDTO `json:"playlists"`
	Plays       []tocadaDTO   `json:"plays"`
	Settings    *ajustesDTO   `json:"settings"`
	Searches    []string      `json:"searches"`
	Saved       []salvoDTO    `json:"saved"`
}

// Codec lê e escreve o backup: JSON legível, marcado com o nome do formato e a
// versão, igual ao da versão em Kotlin (um backup de lá entra aqui e vice-versa).
type Codec struct{}

func (Codec) Codifica(b dominio.Backup) string {
	dto := backupDTO{
		Format: formatoBackup, Version: dominio.VersaoBackup, CreatedAtMs: b.CriadoEm.UnixMilli(),
		Liked: paraDTOs(b.Curtidas), Playlists: []playlistDTO{}, Plays: []tocadaDTO{},
		Settings: ajustesParaDTO(b.Ajustes), Searches: append([]string{}, b.Buscas...), Saved: []salvoDTO{},
	}
	for _, p := range b.Playlists {
		dto.Playlists = append(dto.Playlists, playlistDTO{ID: p.ID, Title: p.Titulo, Tracks: paraDTOs(p.Faixas)})
	}
	for _, t := range b.Tocadas {
		dto.Plays = append(dto.Plays, tocadaDTO{paraDTO(t.Faixa), t.Em.UnixMilli()})
	}
	for _, c := range b.Salvos {
		dto.Saved = append(dto.Saved, salvoParaDTO(c))
	}
	return string(emJSON(dto))
}

func (Codec) Decodifica(bruto string) (dominio.Backup, bool) {
	var dto backupDTO
	if json.Unmarshal([]byte(bruto), &dto) != nil || dto.Format != formatoBackup || dto.Version > dominio.VersaoBackup {
		return dominio.Backup{}, false
	}
	b := dominio.Backup{CriadoEm: time.UnixMilli(dto.CreatedAtMs).UTC(), Curtidas: faixasDe(dto.Liked), Ajustes: dto.Settings.ajustes(), Buscas: dto.Searches}
	for _, p := range dto.Playlists {
		b.Playlists = append(b.Playlists, dominio.PlaylistLocal{ID: p.ID, Titulo: p.Title, Faixas: faixasDe(p.Tracks)})
	}
	for _, t := range dto.Plays {
		if f, ok := t.Track.faixa(); ok {
			b.Tocadas = append(b.Tocadas, dominio.Tocada{Faixa: f, Em: time.UnixMilli(t.AtMs).UTC()})
		}
	}
	for _, s := range dto.Saved {
		if c, ok := s.salvo(); ok {
			b.Salvos = append(b.Salvos, c)
		}
	}
	return b, true
}

// emJSON codifica sem trocar <, > e & por escapes: títulos ficam legíveis.
func emJSON(valor any) []byte {
	var saida bytes.Buffer
	codificador := json.NewEncoder(&saida)
	codificador.SetEscapeHTML(false)
	_ = codificador.Encode(valor)
	return bytes.TrimRight(saida.Bytes(), "\n")
}

// escreveAtomico grava num temporário ao lado e troca de nome: quem lê vê o
// arquivo velho inteiro ou o novo inteiro, nunca a metade. Com firme, espera o
// disco confirmar antes da troca.
func escreveAtomico(caminho string, dados []byte, firme bool) error {
	pasta := filepath.Dir(caminho)
	if erro := os.MkdirAll(pasta, 0o700); erro != nil {
		return erro
	}
	tmp, erro := os.CreateTemp(pasta, "."+filepath.Base(caminho)+".*.tmp")
	if erro != nil {
		return erro
	}
	nome := tmp.Name()
	_, erro = tmp.Write(dados)
	if erro == nil && firme {
		erro = tmp.Sync()
	}
	if fecha := tmp.Close(); erro == nil {
		erro = fecha
	}
	if erro == nil {
		erro = os.Rename(nome, caminho)
	}
	if erro != nil {
		_ = os.Remove(nome)
	}
	return erro
}

type itemFeedDTO struct {
	Type       string    `json:"type"`
	Track      *faixaDTO `json:"track,omitempty"`
	ID         string    `json:"id,omitempty"`
	Kind       *string   `json:"kind,omitempty"`
	Title      string    `json:"title,omitempty"`
	Subtitle   string    `json:"subtitle,omitempty"`
	ArtworkURL *string   `json:"artworkUrl,omitempty"`
	PlaylistID *string   `json:"playlistId,omitempty"`
}

type chipDTO struct {
	Title string `json:"title"`
	Token string `json:"token"`
}

type prateleiraDTO struct {
	Title string        `json:"title"`
	Items []itemFeedDTO `json:"items"`
}

type feedDTO struct {
	Chips   []chipDTO       `json:"chips"`
	Shelves []prateleiraDTO `json:"shelves"`
}

// FeedGuardado é o último Início num arquivo JSON (o mesmo home-feed.json da
// versão em Kotlin); um arquivo que falta ou está estragado é só "sem nada
// guardado". Pode ser usado fora do laço.
type FeedGuardado struct{ Arquivo string }

func (g FeedGuardado) Carrega() (dominio.PaginaFeed, bool) {
	bruto, erro := os.ReadFile(g.Arquivo)
	var dto feedDTO
	if erro != nil || json.Unmarshal(bruto, &dto) != nil || dto.Shelves == nil {
		return dominio.PaginaFeed{}, false
	}
	var pagina dominio.PaginaFeed
	for _, c := range dto.Chips {
		pagina.Chips = append(pagina.Chips, dominio.Chip{Titulo: c.Title, Token: c.Token})
	}
	for _, s := range dto.Shelves {
		prateleira := dominio.Prateleira{Titulo: s.Title}
		for _, i := range s.Items {
			switch i.Type {
			case "song":
				if i.Track != nil {
					if f, ok := i.Track.faixa(); ok {
						prateleira.Itens = append(prateleira.Itens, dominio.ItemDeMusica(f))
					}
				}
			case "collection":
				if tipo, ok := dominio.TipoColecaoDe(texto(i.Kind)); ok {
					prateleira.Itens = append(prateleira.Itens, dominio.Item{Tipo: dominio.ItemColecao, ID: i.ID, Colecao: tipo, Titulo: i.Title, Subtitulo: i.Subtitle, Capa: texto(i.ArtworkURL), PlaylistID: texto(i.PlaylistID)})
				}
			case "artist":
				prateleira.Itens = append(prateleira.Itens, dominio.Item{Tipo: dominio.ItemArtista, ID: i.ID, Titulo: i.Title, Subtitulo: i.Subtitle, Capa: texto(i.ArtworkURL)})
			}
		}
		pagina.Prateleiras = append(pagina.Prateleiras, prateleira)
	}
	return pagina, true
}

func (g FeedGuardado) Guarda(p dominio.PaginaFeed) {
	dto := feedDTO{Chips: []chipDTO{}, Shelves: []prateleiraDTO{}}
	for _, c := range p.Chips {
		dto.Chips = append(dto.Chips, chipDTO{c.Titulo, c.Token})
	}
	for _, s := range p.Prateleiras {
		prateleira := prateleiraDTO{Title: s.Titulo, Items: []itemFeedDTO{}}
		for _, i := range s.Itens {
			switch i.Tipo {
			case dominio.ItemMusica:
				faixa := i.Faixa
				faixa.Stream = "" // os links de stream expiram
				dto := paraDTO(faixa)
				prateleira.Items = append(prateleira.Items, itemFeedDTO{Type: "song", Track: &dto})
			case dominio.ItemColecao:
				tipo := string(i.Colecao)
				prateleira.Items = append(prateleira.Items, itemFeedDTO{Type: "collection", ID: i.ID, Kind: &tipo, Title: i.Titulo, Subtitle: i.Subtitulo, ArtworkURL: opcional(i.Capa), PlaylistID: opcional(i.PlaylistID)})
			case dominio.ItemArtista:
				prateleira.Items = append(prateleira.Items, itemFeedDTO{Type: "artist", ID: i.ID, Title: i.Titulo, Subtitle: i.Subtitulo, ArtworkURL: opcional(i.Capa)})
			}
		}
		dto.Shelves = append(dto.Shelves, prateleira)
	}
	_ = escreveAtomico(g.Arquivo, emJSON(dto), false)
}
