// SPDX-License-Identifier: GPL-3.0-or-later

package guarda

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// BancoAntigo é o SQLite do daemon em Kotlin, na mesma pasta de dados.
const BancoAntigo = "mktunes.db"

// ErrSemSqlite: o comando sqlite3 não está instalado; a migração fica para a
// próxima subida.
var ErrSemSqlite = errors.New("falta o sqlite3")

const prazoDoSqlite = 30 * time.Second

// Migrado conta o que veio do banco antigo.
type Migrado struct{ Curtidas, Playlists, Tocadas, Salvos, Downloads int }

type linhaFaixa struct {
	TrackID    string  `json:"track_id"`
	Provider   string  `json:"provider"`
	Title      string  `json:"title"`
	Artist     string  `json:"artist"`
	DurationMs int64   `json:"duration_ms"`
	ArtworkURL *string `json:"artwork_url"`
	StreamURL  *string `json:"stream_url"`
}

func (l linhaFaixa) dto() faixaDTO {
	return faixaDTO{l.TrackID, l.Provider, l.Title, l.Artist, l.DurationMs, vazioENulo(l.ArtworkURL), vazioENulo(l.StreamURL)}
}

func vazioENulo(p *string) *string {
	if p == nil || *p == "" {
		return nil
	}
	return p
}

// As consultas seguem o esquema dos .sq do MkTunes em Kotlin (versão 5 do banco).
const (
	sqlCurtidas  = "SELECT track_id, provider, title, artist, duration_ms, artwork_url, stream_url, liked_at FROM liked_track ORDER BY liked_at DESC, rowid DESC"
	sqlPlaylists = "SELECT id, title, created_at FROM local_playlist ORDER BY created_at, rowid"
	sqlFaixas    = "SELECT playlist_id, track_id, provider, title, artist, duration_ms, artwork_url, stream_url FROM local_playlist_track ORDER BY playlist_id, position"
	sqlTocadas   = "SELECT track_id, provider, title, artist, duration_ms, artwork_url, stream_url, played_at FROM play ORDER BY played_at, id"
	sqlSalvos    = "SELECT id, kind, title, owner, artwork_url, saved_at FROM saved_collection ORDER BY saved_at DESC, rowid DESC"
	sqlDownloads = "SELECT track_id, provider, title, artist, duration_ms, artwork_url, stream_url, status, progress, path, reason, created_at FROM download ORDER BY created_at DESC"
	sqlPrefs     = "SELECT key, value FROM preference"
)

// Migra traz o banco SQLite antigo para a guarda, uma vez: exporta as tabelas
// com `sqlite3 -json`, junta ao que a guarda já tem (sem repetir) e renomeia o
// banco para mktunes.db.migrado (não apaga). Sem banco, não faz nada (false).
// Se algo falha, o banco fica onde está e a guarda como estava no disco.
func (g *Guarda) Migra(ctx context.Context, sqlite3 string) (Migrado, bool, error) {
	banco := g.caminho(BancoAntigo)
	if _, erro := os.Stat(banco); errors.Is(erro, fs.ErrNotExist) {
		return Migrado{}, false, nil
	}
	if _, erro := exec.LookPath(sqlite3); erro != nil {
		return Migrado{}, false, ErrSemSqlite
	}
	var (
		curtidas []struct {
			linhaFaixa
			LikedAt int64 `json:"liked_at"`
		}
		playlists []struct {
			ID        string `json:"id"`
			Title     string `json:"title"`
			CreatedAt int64  `json:"created_at"`
		}
		faixas []struct {
			linhaFaixa
			PlaylistID string `json:"playlist_id"`
		}
		tocadas []struct {
			linhaFaixa
			PlayedAt int64 `json:"played_at"`
		}
		salvos []struct {
			ID         string  `json:"id"`
			Kind       string  `json:"kind"`
			Title      string  `json:"title"`
			Owner      string  `json:"owner"`
			ArtworkURL *string `json:"artwork_url"`
			SavedAt    int64   `json:"saved_at"`
		}
		downloads []struct {
			linhaFaixa
			Status    string  `json:"status"`
			Progress  float64 `json:"progress"`
			Path      *string `json:"path"`
			Reason    *string `json:"reason"`
			CreatedAt int64   `json:"created_at"`
		}
		prefs []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}
	)
	for _, consulta := range []struct {
		sql  string
		alvo any
	}{{sqlCurtidas, &curtidas}, {sqlPlaylists, &playlists}, {sqlFaixas, &faixas}, {sqlTocadas, &tocadas}, {sqlSalvos, &salvos}, {sqlDownloads, &downloads}, {sqlPrefs, &prefs}} {
		if erro := consultaSqlite(ctx, sqlite3, banco, consulta.sql, consulta.alvo); erro != nil {
			return Migrado{}, false, erro
		}
	}

	// Monta tudo numa cópia: a guarda só muda se o disco aceitar.
	var feito Migrado
	nova := g.biblioteca
	tem := map[string]bool{}
	for _, c := range nova.Liked {
		tem[c.ID] = true
	}
	nova.Liked = append([]curtidaDTO(nil), nova.Liked...)
	for _, c := range curtidas {
		if !tem[c.TrackID] {
			tem[c.TrackID] = true
			nova.Liked = append(nova.Liked, curtidaDTO{c.dto(), c.LikedAt})
			feito.Curtidas++
		}
	}
	sort.SliceStable(nova.Liked, func(i, j int) bool { return nova.Liked[i].LikedAtMs > nova.Liked[j].LikedAtMs })

	tem = map[string]bool{}
	for _, p := range nova.Playlists {
		tem[p.ID] = true
	}
	nova.Playlists = append([]playlistDTO(nil), nova.Playlists...)
	for _, p := range playlists {
		if tem[p.ID] {
			continue
		}
		dto := playlistDTO{ID: p.ID, Title: p.Title, CreatedAtMs: p.CreatedAt, Tracks: []faixaDTO{}}
		for _, f := range faixas {
			if f.PlaylistID == p.ID {
				dto.Tracks = append(dto.Tracks, f.dto())
			}
		}
		nova.Playlists = append(nova.Playlists, dto)
		feito.Playlists++
	}

	tem = map[string]bool{}
	for _, s := range nova.Saved {
		tem[s.ID] = true
	}
	nova.Saved = append([]salvoDTO(nil), nova.Saved...)
	for _, s := range salvos {
		if !tem[s.ID] {
			nova.Saved = append(nova.Saved, salvoDTO{s.ID, s.Kind, s.Title, s.Owner, vazioENulo(s.ArtworkURL), s.SavedAt})
			feito.Salvos++
		}
	}
	sort.SliceStable(nova.Saved, func(i, j int) bool { return nova.Saved[i].SavedAtMs > nova.Saved[j].SavedAtMs })

	// As preferências antigas só entram onde a guarda nova ainda não tem as suas.
	novasPrefs := g.preferencias
	var sessao *sessaoDTO
	for _, p := range prefs {
		switch p.Key {
		case "settings":
			var a ajustesDTO
			if novasPrefs.Settings == nil && json.Unmarshal([]byte(p.Value), &a) == nil {
				novasPrefs.Settings = ajustesParaDTO((&a).ajustes())
			}
		case "search_history":
			var buscas []string
			if novasPrefs.Searches == nil && json.Unmarshal([]byte(p.Value), &buscas) == nil {
				novasPrefs.Searches = buscas
			}
		case "session":
			var s sessaoDTO
			if _, erro := os.Stat(g.caminho(arqSessao)); errors.Is(erro, fs.ErrNotExist) && json.Unmarshal([]byte(p.Value), &s) == nil {
				s.Versao = Versao
				sessao = &s
			}
		}
	}

	baixados := g.leDownloads()
	tem = map[string]bool{}
	for _, j := range baixados.Jobs {
		tem[j.Track.ID] = true
	}
	for _, d := range downloads {
		if !tem[d.TrackID] {
			baixados.Jobs = append(baixados.Jobs, downloadDTO{d.dto(), d.Status, d.Progress, texto(d.Path), texto(d.Reason), d.CreatedAt})
			feito.Downloads++
		}
	}
	baixados.Versao = Versao

	type chave struct {
		id string
		em int64
	}
	ouvidas := map[chave]bool{}
	g.leHistorico(func(t tocadaDTO) { ouvidas[chave{t.Track.ID, t.AtMs}] = true })
	var novasTocadas []tocadaDTO
	for _, t := range tocadas {
		if c := (chave{t.TrackID, t.PlayedAt}); !ouvidas[c] {
			ouvidas[c] = true
			novasTocadas = append(novasTocadas, tocadaDTO{t.dto(), t.PlayedAt})
		}
	}
	feito.Tocadas = len(novasTocadas)

	nova.Versao, novasPrefs.Versao = Versao, Versao
	if erro := escreveAtomico(g.caminho(arqBiblioteca), emJSON(nova), true); erro != nil {
		return Migrado{}, false, erro
	}
	g.biblioteca = nova
	if erro := escreveAtomico(g.caminho(arqPreferencias), emJSON(novasPrefs), true); erro != nil {
		return Migrado{}, false, erro
	}
	g.preferencias = novasPrefs
	if erro := escreveAtomico(g.caminho(arqDownloads), emJSON(baixados), true); erro != nil {
		return Migrado{}, false, erro
	}
	if sessao != nil {
		if erro := escreveAtomico(g.caminho(arqSessao), emJSON(sessao), true); erro != nil {
			return Migrado{}, false, erro
		}
	}
	var falha error
	antes := g.AoErrar
	g.AoErrar = func(erro error) { falha = erro }
	g.acrescentaTocadas(novasTocadas)
	g.AoErrar = antes
	if falha != nil {
		return Migrado{}, false, falha
	}
	// Só agora o banco sai do caminho: daqui em diante a guarda nova é a que vale.
	destino := banco + ".migrado"
	if _, erro := os.Lstat(destino); erro == nil {
		destino = fmt.Sprintf("%s.%d", destino, time.Now().Unix())
	}
	if erro := os.Rename(banco, destino); erro != nil {
		return feito, false, erro
	}
	return feito, true, nil
}

// consultaSqlite roda uma consulta só de leitura e lê o JSON que o sqlite3
// imprime (nada, quando a tabela está vazia). Uma tabela que o banco ainda
// não tinha (um banco de versão antiga) conta como vazia.
func consultaSqlite(ctx context.Context, sqlite3, banco, sql string, alvo any) error {
	ctx, cancela := context.WithTimeout(ctx, prazoDoSqlite)
	defer cancela()
	comando := exec.CommandContext(ctx, sqlite3, "-readonly", "-json", filepath.Clean(banco), sql)
	var saida, erros bytes.Buffer
	comando.Stdout, comando.Stderr = &saida, &erros
	if erro := comando.Run(); erro != nil {
		if strings.Contains(erros.String(), "no such table") {
			return nil
		}
		return fmt.Errorf("sqlite3 falhou: %s", ultimaLinha(erros.String(), erro.Error()))
	}
	if len(bytes.TrimSpace(saida.Bytes())) == 0 {
		return nil
	}
	if erro := json.Unmarshal(saida.Bytes(), alvo); erro != nil {
		return fmt.Errorf("o sqlite3 respondeu algo que não é JSON: %w", erro)
	}
	return nil
}

func ultimaLinha(texto, reserva string) string {
	linhas := strings.Split(strings.TrimSpace(texto), "\n")
	if ultima := strings.TrimSpace(linhas[len(linhas)-1]); ultima != "" {
		return ultima
	}
	return reserva
}
