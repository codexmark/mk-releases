// SPDX-License-Identifier: GPL-3.0-or-later

package guarda

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"mktunes/internal/app"
	"mktunes/internal/dominio"
	"mktunes/internal/prova"
)

var _ app.Guarda = (*Guarda)(nil)
var _ app.Codec = Codec{}
var _ app.GuardaDoInicio = FeedGuardado{}

var meioDia = time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)

func igual(t *testing.T, esperado, obtido any) {
	t.Helper()
	if !reflect.DeepEqual(esperado, obtido) {
		t.Fatalf("esperado %#v\n  obtido %#v", esperado, obtido)
	}
}

func abre(t *testing.T, pasta string) *Guarda {
	t.Helper()
	g, erro := Abre(pasta)
	if erro != nil {
		t.Fatal(erro)
	}
	g.AoErrar = func(erro error) { t.Errorf("guarda: %v", erro) }
	return g
}

func musica() dominio.Faixa {
	return dominio.Faixa{ID: "W40W7L1fhUM", Provedor: dominio.YouTube, Titulo: "Até Quando", Artista: "Natanzinho Lima", Duracao: 144 * time.Second, Capa: "https://art/1"}
}

func TestBibliotecaVoltaDepoisDeReabrir(t *testing.T) {
	pasta := t.TempDir()
	g := abre(t, pasta)
	igual(t, dominio.AjustesPadrao(), g.Ajustes())
	_, tem := g.CarregaSessao()
	igual(t, false, tem)

	g.Curte(prova.Faixa("a"), true, meioDia)
	g.Curte(musica(), true, meioDia.Add(time.Second))
	g.Curte(prova.Faixa("b"), true, meioDia.Add(2*time.Second))
	g.Curte(musica(), true, meioDia.Add(3*time.Second)) // curtir de novo mantém uma entrada, no topo
	g.Curte(prova.Faixa("b"), false, meioDia)
	g.SalvaPlaylist(dominio.PlaylistLocal{ID: "p1", Titulo: "Estrada", Faixas: []dominio.Faixa{musica(), prova.Faixa("b")}}, meioDia)
	g.SalvaPlaylist(dominio.PlaylistLocal{ID: "p2", Titulo: "Treino"}, meioDia.Add(time.Second))
	g.SalvaPlaylist(dominio.PlaylistLocal{ID: "p1", Titulo: "Viagem", Faixas: []dominio.Faixa{prova.Faixa("b")}}, meioDia.Add(time.Hour))
	g.SalvaPlaylist(dominio.PlaylistLocal{ID: "p3", Titulo: "Some"}, meioDia)
	g.ApagaPlaylist("p3")
	album := dominio.ColecaoSalva{ID: "MPRE1", Tipo: dominio.Album, Titulo: "Sobrevivendo no Inferno", Dono: "Racionais MC's", Capa: "https://art/1", SalvaEm: meioDia}
	g.SalvaColecao(album)
	g.SalvaColecao(dominio.ColecaoSalva{ID: "VL2", Tipo: dominio.Playlist, Titulo: "Topnejo", Dono: "Playlist", SalvaEm: meioDia.Add(time.Minute)})
	g.SalvaColecao(dominio.ColecaoSalva{ID: "VL3", Tipo: dominio.Playlist, Titulo: "Fora", SalvaEm: meioDia})
	g.SalvaColecao(album)
	g.EsqueceColecao("VL3")
	ajustes := dominio.Ajustes{Qualidade: dominio.QualidadeEconomia, Trabalhadores: 2, PularSilencio: true}
	g.SalvaAjustes(ajustes)
	g.SalvaBuscas([]string{"lilriog", "racionais"})
	sessao := app.SessaoSalva{Fila: dominio.NovaFila([]dominio.Faixa{musica(), prova.Faixa("b")}, 1).ComRepetir(dominio.RepetirTudo).Retrato(), Posicao: 42 * time.Second}
	g.SalvaSessao(sessao)

	de := abre(t, pasta)
	igual(t, []string{"W40W7L1fhUM", "a"}, prova.IDs(de.Curtidas()))
	igual(t, musica(), de.Curtidas()[0])
	igual(t, []dominio.PlaylistLocal{{ID: "p1", Titulo: "Viagem", Faixas: []dominio.Faixa{prova.Faixa("b")}}, {ID: "p2", Titulo: "Treino", Faixas: []dominio.Faixa{}}}, de.Playlists())
	igual(t, []string{"VL2", "MPRE1"}, []string{de.Salvos()[0].ID, de.Salvos()[1].ID})
	igual(t, album, de.Salvos()[1])
	igual(t, ajustes, de.Ajustes())
	igual(t, []string{"lilriog", "racionais"}, de.Buscas())
	voltou, tem := de.CarregaSessao()
	igual(t, true, tem)
	igual(t, sessao, voltou)

	// Os arquivos são só do usuário e trazem a versão.
	info, _ := os.Stat(filepath.Join(pasta, arqBiblioteca))
	igual(t, os.FileMode(0o600), info.Mode().Perm())
	bruto, _ := os.ReadFile(filepath.Join(pasta, arqBiblioteca))
	igual(t, true, strings.Contains(string(bruto), `"versao":1`))
	sobras, _ := filepath.Glob(filepath.Join(pasta, ".*.tmp"))
	igual(t, 0, len(sobras))
}

func TestHistoricoEmLinhas(t *testing.T) {
	g := abre(t, t.TempDir())
	g.RegistraTocada(dominio.Tocada{Faixa: musica(), Em: meioDia})
	g.RegistraTocada(dominio.Tocada{Faixa: prova.Faixa("velha"), Em: meioDia.Add(-40 * 24 * time.Hour)})
	g.RegistraTocada(dominio.Tocada{Faixa: prova.Faixa("ontem"), Em: meioDia.Add(-24 * time.Hour)})
	todas := g.TocadasDesde(time.Unix(0, 0))
	igual(t, []string{"velha", "ontem", "W40W7L1fhUM"}, []string{todas[0].Faixa.ID, todas[1].Faixa.ID, todas[2].Faixa.ID})
	igual(t, dominio.Tocada{Faixa: musica(), Em: meioDia}, todas[2])
	igual(t, 1, len(g.TocadasDesde(meioDia.Add(-time.Hour))))
	g.EsqueceTocadasAntesDe(meioDia.Add(-30 * 24 * time.Hour))
	igual(t, 2, len(g.TocadasDesde(time.Unix(0, 0))))
	// Uma linha estragada não derruba o resto.
	arquivo := filepath.Join(g.pasta, arqHistorico)
	bruto, _ := os.ReadFile(arquivo)
	os.WriteFile(arquivo, append([]byte("lixo\n"), bruto...), 0o600)
	igual(t, 2, len(g.TocadasDesde(time.Unix(0, 0))))
}

func TestDownloadsGuardados(t *testing.T) {
	g := abre(t, t.TempDir())
	um, dois := prova.Faixa("1"), prova.Faixa("2")
	g.SalvaDownload(dominio.Download{Faixa: um, Estado: dominio.NaFila, CriadoEm: meioDia})
	g.SalvaDownload(dominio.Download{Faixa: dois, Estado: dominio.Falhou, Motivo: "HTTP 403", CriadoEm: meioDia.Add(time.Millisecond)})
	g.SalvaDownload(dominio.Download{Faixa: um, Estado: dominio.Concluido, Progresso: 1, Caminho: "/m/1.m4a", CriadoEm: meioDia})
	igual(t, []dominio.Download{
		{Faixa: dois, Estado: dominio.Falhou, Motivo: "HTTP 403", CriadoEm: meioDia.Add(time.Millisecond)},
		{Faixa: um, Estado: dominio.Concluido, Progresso: 1, Caminho: "/m/1.m4a", CriadoEm: meioDia},
	}, g.Downloads())
	g.RemoveDownload("2")
	igual(t, 1, len(g.Downloads()))
}

func TestArquivoEstragadoOuNovoDemais(t *testing.T) {
	pasta := t.TempDir()
	os.WriteFile(filepath.Join(pasta, arqBiblioteca), []byte("{ metade"), 0o600)
	g, erro := Abre(pasta)
	if erro != nil {
		t.Fatal(erro)
	}
	igual(t, 0, len(g.Curtidas()))
	postos, _ := filepath.Glob(filepath.Join(pasta, arqBiblioteca+".danificado-*"))
	igual(t, 1, len(postos))

	os.WriteFile(filepath.Join(pasta, arqPreferencias), []byte(`{"versao":99}`), 0o600)
	_, erro = Abre(pasta)
	if erro == nil || !strings.Contains(erro.Error(), "versão mais nova") {
		t.Fatalf("erro: %v", erro)
	}
	bruto, _ := os.ReadFile(filepath.Join(pasta, arqPreferencias))
	igual(t, `{"versao":99}`, string(bruto))
}

func backupDeExemplo() dominio.Backup {
	return dominio.Backup{
		CriadoEm:  meioDia,
		Curtidas:  []dominio.Faixa{musica()},
		Playlists: []dominio.PlaylistLocal{{ID: "p1", Titulo: "Estrada", Faixas: []dominio.Faixa{musica(), prova.Faixa("b")}}},
		Tocadas:   []dominio.Tocada{{Faixa: musica(), Em: meioDia.Add(-time.Hour)}},
		Ajustes:   dominio.Ajustes{Qualidade: dominio.QualidadeEconomia, Trabalhadores: 2, Normalizar: true, PularSilencio: true},
		Buscas:    []string{"lilriog"},
		Salvos:    []dominio.ColecaoSalva{{ID: "MPRE1", Tipo: dominio.Album, Titulo: "Sobrevivendo no Inferno", Dono: "Racionais MC's", Capa: "https://art/1", SalvaEm: meioDia}},
	}
}

func TestBackupNoFormatoDaFonte(t *testing.T) {
	codec := Codec{}
	escrito := codec.Codifica(backupDeExemplo())
	for _, trecho := range []string{`"format":"mktunes-backup"`, `"version":1`, `"createdAtMs":1795348800000`, `"streamUrl":null`, `"artworkUrl":"https://art/1"`, `"atMs":`, `"savedAtMs":`, `"audioQuality":"ECONOMY"`, `"provider":"YOUTUBE"`} {
		if !strings.Contains(escrito, trecho) {
			t.Fatalf("falta %s em %s", trecho, escrito)
		}
	}
	lido, ok := codec.Decodifica(escrito)
	igual(t, true, ok)
	igual(t, backupDeExemplo(), lido)

	_, ok = codec.Decodifica("not json")
	igual(t, false, ok)
	_, ok = codec.Decodifica(`{"some":"other app"}`)
	igual(t, false, ok)
	_, ok = codec.Decodifica(strings.Replace(escrito, `"version":1`, `"version":99`, 1))
	igual(t, false, ok)
	// Um backup de antes de existirem os álbuns salvos.
	antigo, ok := codec.Decodifica(regexp.MustCompile(`,"saved":\[.*?\]`).ReplaceAllString(escrito, ""))
	igual(t, true, ok)
	igual(t, 0, len(antigo.Salvos))
	// Uma fonte que esta versão não conhece: a faixa é pulada.
	outro, ok := codec.Decodifica(strings.Replace(escrito, `"provider":"YOUTUBE"`, `"provider":"TIDAL"`, 1))
	igual(t, true, ok)
	igual(t, 0, len(outro.Curtidas))
	// Um backup escrito pela versão em Kotlin (os campos na ordem e com os nulos dela).
	deKotlin := `{"format":"mktunes-backup","version":1,"createdAtMs":1795348800000,"liked":[{"id":"abc","provider":"YOUTUBE","title":"T","artist":"A","durationMs":1000,"artworkUrl":null,"streamUrl":null}],"playlists":[],"plays":[],"settings":{"autoplay":false,"audioQuality":"HIGH","downloadWorkers":9,"normalizeVolume":true,"skipSilence":false,"soundWave":true},"searches":["x"],"saved":[]}`
	lido, ok = codec.Decodifica(deKotlin)
	igual(t, true, ok)
	igual(t, dominio.Ajustes{Qualidade: dominio.QualidadeAlta, Trabalhadores: 4, Normalizar: true, OndaSonora: true}, lido.Ajustes)
	igual(t, "abc", lido.Curtidas[0].ID)
}

func TestFeedGuardado(t *testing.T) {
	arquivo := filepath.Join(t.TempDir(), "cache", "home-feed.json")
	guardado := FeedGuardado{Arquivo: arquivo}
	_, tem := guardado.Carrega()
	igual(t, false, tem)
	comLink := musica()
	comLink.Stream = "https://googlevideo/expira"
	album := dominio.Item{Tipo: dominio.ItemColecao, ID: "MPRE1", Colecao: dominio.Album, Titulo: "Anti", Subtitulo: "Rihanna", PlaylistID: "OLAK1"}
	artista := dominio.Item{Tipo: dominio.ItemArtista, ID: "UC1", Titulo: "Rihanna", Subtitulo: "Artista", Capa: "https://art/2"}
	guardado.Guarda(dominio.PaginaFeed{
		Chips:       []dominio.Chip{{Titulo: "Relax", Token: "t-relax"}},
		Prateleiras: []dominio.Prateleira{{Titulo: "Mixes", Itens: []dominio.Item{dominio.ItemDeMusica(comLink), album, artista}}},
		Proxima:     "p2",
	})
	pagina, tem := guardado.Carrega()
	igual(t, true, tem)
	// Volta sem o link de stream (expira) e sem o token da próxima página.
	igual(t, dominio.PaginaFeed{
		Chips:       []dominio.Chip{{Titulo: "Relax", Token: "t-relax"}},
		Prateleiras: []dominio.Prateleira{{Titulo: "Mixes", Itens: []dominio.Item{dominio.ItemDeMusica(musica()), album, artista}}},
	}, pagina)
	os.WriteFile(arquivo, []byte("{ estragado"), 0o600)
	_, tem = guardado.Carrega()
	igual(t, false, tem)
	// O arquivo que a versão em Kotlin deixou (sem os campos com valor padrão).
	os.WriteFile(arquivo, []byte(`{"chips":[],"shelves":[{"title":"Mixes","items":[{"type":"song","track":{"id":"a","provider":"YOUTUBE","title":"T","artist":"A","durationMs":1000}},{"type":"collection","id":"VL1","kind":"PLAYLIST","title":"P"}]}]}`), 0o600)
	pagina, tem = guardado.Carrega()
	igual(t, true, tem)
	igual(t, 2, len(pagina.Prateleiras[0].Itens))
}

// sqliteDeMentira responde a cada consulta com o JSON que o sqlite3 -json daria.
func sqliteDeMentira(t *testing.T, falha string) string {
	t.Helper()
	comando := filepath.Join(t.TempDir(), "sqlite3")
	corpo := `#!/bin/sh
[ "$1" = "-readonly" ] && [ "$2" = "-json" ] && [ -f "$3" ] || { echo "uso errado: $*" >&2; exit 2; }
case "$4" in
  *"` + falha + `"*) echo "Error: in prepare, database disk image is malformed" >&2; exit 11 ;;
  *"FROM liked_track"*) echo '[{"track_id":"l2","provider":"YOUTUBE","title":"Halo","artist":"Beyoncé","duration_ms":261000,"artwork_url":"https://art/2","stream_url":null,"liked_at":2000},
{"track_id":"l1","provider":"YOUTUBE","title":"Umbrella","artist":"Rihanna","duration_ms":275000,"artwork_url":null,"stream_url":null,"liked_at":1000}]' ;;
  *"FROM local_playlist_track"*) echo '[{"playlist_id":"p1","track_id":"l1","provider":"YOUTUBE","title":"Umbrella","artist":"Rihanna","duration_ms":275000,"artwork_url":null,"stream_url":null},
{"playlist_id":"p1","track_id":"l2","provider":"YOUTUBE","title":"Halo","artist":"Beyoncé","duration_ms":261000,"artwork_url":null,"stream_url":null}]' ;;
  *"FROM local_playlist"*) echo '[{"id":"p1","title":"Treino","created_at":500}]' ;;
  *"FROM play"*) echo '[{"track_id":"l1","provider":"YOUTUBE","title":"Umbrella","artist":"Rihanna","duration_ms":275000,"artwork_url":null,"stream_url":null,"played_at":1795348800000},
{"track_id":"l1","provider":"YOUTUBE","title":"Umbrella","artist":"Rihanna","duration_ms":275000,"artwork_url":null,"stream_url":null,"played_at":1795348900000}]' ;;
  *"FROM saved_collection"*) ;;
  *"FROM download"*) echo "Error: in prepare, no such table: download" >&2; exit 1 ;;
  *"FROM preference"*) cat <<'FIM'
[{"key":"settings","value":"{\"autoplay\":false,\"audioQuality\":\"ECONOMY\",\"downloadWorkers\":3,\"normalizeVolume\":true,\"skipSilence\":false,\"soundWave\":true}"},
{"key":"session","value":"{\"tracks\":[{\"id\":\"l1\",\"provider\":\"YOUTUBE\",\"title\":\"Umbrella\",\"artist\":\"Rihanna\",\"durationMs\":275000}],\"index\":0,\"repeat\":\"ALL\",\"positionMs\":30000}"},
{"key":"search_history","value":"[\"rihanna\"]"}]
FIM
  ;;
  *) echo "consulta inesperada: $4" >&2; exit 3 ;;
esac
`
	if erro := os.WriteFile(comando, []byte(corpo), 0o755); erro != nil {
		t.Fatal(erro)
	}
	return comando
}

func TestMigraOBancoAntigoUmaVez(t *testing.T) {
	pasta := t.TempDir()
	banco := filepath.Join(pasta, BancoAntigo)
	os.WriteFile(banco, []byte("SQLite format 3\x00"), 0o600)
	g := abre(t, pasta)
	// O que já está na guarda nova fica; o banco só acrescenta.
	g.Curte(prova.Faixa("nova"), true, time.UnixMilli(5000))
	g.Curte(dominio.Faixa{ID: "l1", Provedor: dominio.YouTube, Titulo: "Umbrella (daqui)"}, true, time.UnixMilli(3000))
	g.RegistraTocada(dominio.Tocada{Faixa: prova.Faixa("l1"), Em: time.UnixMilli(1795348800000)})

	feito, migrou, erro := g.Migra(context.Background(), sqliteDeMentira(t, "nada falha"))
	if erro != nil || !migrou {
		t.Fatalf("migrou=%v erro=%v", migrou, erro)
	}
	igual(t, Migrado{Curtidas: 1, Playlists: 1, Tocadas: 1}, feito)
	if _, erro := os.Stat(banco); !errors.Is(erro, os.ErrNotExist) {
		t.Fatal("o banco devia ter saído do caminho")
	}
	if _, erro := os.Stat(banco + ".migrado"); erro != nil {
		t.Fatal("o banco devia ter virado mktunes.db.migrado")
	}

	de := abre(t, pasta)
	igual(t, []string{"nova", "l1", "l2"}, prova.IDs(de.Curtidas()))
	igual(t, "Umbrella (daqui)", de.Curtidas()[1].Titulo)
	igual(t, "https://art/2", de.Curtidas()[2].Capa)
	igual(t, []string{"l1", "l2"}, prova.IDs(de.Playlists()[0].Faixas))
	igual(t, "Treino", de.Playlists()[0].Titulo)
	igual(t, 2, len(de.TocadasDesde(time.Unix(0, 0))))
	igual(t, dominio.Ajustes{Qualidade: dominio.QualidadeEconomia, Trabalhadores: 3, Normalizar: true, OndaSonora: true}, de.Ajustes())
	igual(t, []string{"rihanna"}, de.Buscas())
	sessao, tem := de.CarregaSessao()
	igual(t, true, tem)
	igual(t, 30*time.Second, sessao.Posicao)
	igual(t, dominio.RepetirTudo, sessao.Fila.Repetir)
	igual(t, []string{"l1"}, prova.IDs(sessao.Fila.Faixas))

	// Sem banco não há o que migrar.
	_, migrou, erro = de.Migra(context.Background(), "/não/existe/sqlite3")
	igual(t, false, migrou)
	igual(t, nil, erro)
}

func TestMigracaoQueNaoDaCertoDeixaTudoComoEstava(t *testing.T) {
	pasta := t.TempDir()
	banco := filepath.Join(pasta, BancoAntigo)
	os.WriteFile(banco, []byte("SQLite format 3\x00"), 0o600)
	g := abre(t, pasta)

	_, migrou, erro := g.Migra(context.Background(), filepath.Join(pasta, "sqlite3-que-falta"))
	igual(t, false, migrou)
	igual(t, true, errors.Is(erro, ErrSemSqlite))

	_, migrou, erro = g.Migra(context.Background(), sqliteDeMentira(t, "FROM play"))
	igual(t, false, migrou)
	if erro == nil || !strings.Contains(erro.Error(), "malformed") {
		t.Fatalf("erro: %v", erro)
	}
	if _, erro := os.Stat(banco); erro != nil {
		t.Fatal("o banco tem de continuar no lugar para a próxima tentativa")
	}
	igual(t, 0, len(g.Curtidas()))
	if _, erro := os.Stat(filepath.Join(pasta, arqBiblioteca)); !errors.Is(erro, os.ErrNotExist) {
		t.Fatal("nada devia ter sido gravado")
	}
}

// Com o sqlite3 de verdade (quando a máquina tem): o esquema dos .sq da fonte,
// num banco criado aqui, numa pasta temporária.
func TestMigraComOSqliteDeVerdade(t *testing.T) {
	sqlite3, erro := exec.LookPath("sqlite3")
	if erro != nil {
		t.Skip("sem sqlite3 nesta máquina")
	}
	pasta := t.TempDir()
	esquema := `
CREATE TABLE play (id INTEGER PRIMARY KEY AUTOINCREMENT, track_id TEXT NOT NULL, provider TEXT NOT NULL, title TEXT NOT NULL, artist TEXT NOT NULL, duration_ms INTEGER NOT NULL, artwork_url TEXT, stream_url TEXT, played_at INTEGER NOT NULL);
CREATE TABLE liked_track (track_id TEXT NOT NULL PRIMARY KEY, provider TEXT NOT NULL, title TEXT NOT NULL, artist TEXT NOT NULL, duration_ms INTEGER NOT NULL, artwork_url TEXT, stream_url TEXT, liked_at INTEGER NOT NULL);
CREATE TABLE download (track_id TEXT NOT NULL PRIMARY KEY, provider TEXT NOT NULL, title TEXT NOT NULL, artist TEXT NOT NULL, duration_ms INTEGER NOT NULL, artwork_url TEXT, stream_url TEXT, status TEXT NOT NULL, progress REAL NOT NULL, path TEXT, reason TEXT, created_at INTEGER NOT NULL);
CREATE TABLE preference (key TEXT NOT NULL PRIMARY KEY, value TEXT NOT NULL);
CREATE TABLE local_playlist (id TEXT NOT NULL PRIMARY KEY, title TEXT NOT NULL, created_at INTEGER NOT NULL);
CREATE TABLE local_playlist_track (playlist_id TEXT NOT NULL, position INTEGER NOT NULL, track_id TEXT NOT NULL, provider TEXT NOT NULL, title TEXT NOT NULL, artist TEXT NOT NULL, duration_ms INTEGER NOT NULL, artwork_url TEXT, stream_url TEXT, PRIMARY KEY (playlist_id, position));
CREATE TABLE saved_collection (id TEXT NOT NULL PRIMARY KEY, kind TEXT NOT NULL, title TEXT NOT NULL, owner TEXT NOT NULL, artwork_url TEXT, saved_at INTEGER NOT NULL);
INSERT INTO liked_track VALUES ('l1','YOUTUBE','Até "Quando"','Natanzinho Lima',144000,'https://art/1',NULL,1000);
INSERT INTO liked_track VALUES ('l2','YOUTUBE','Halo','Beyoncé',261000,NULL,NULL,2000);
INSERT INTO play(track_id,provider,title,artist,duration_ms,artwork_url,stream_url,played_at) VALUES ('l1','YOUTUBE','Até "Quando"','Natanzinho Lima',144000,NULL,NULL,1795348800000);
INSERT INTO local_playlist VALUES ('p1','Treino',500);
INSERT INTO local_playlist_track VALUES ('p1',1,'l2','YOUTUBE','Halo','Beyoncé',261000,NULL,NULL);
INSERT INTO local_playlist_track VALUES ('p1',0,'l1','YOUTUBE','Até "Quando"','Natanzinho Lima',144000,NULL,NULL);
INSERT INTO saved_collection VALUES ('MPRE1','ALBUM','Anti','Rihanna',NULL,3000);
INSERT INTO download VALUES ('l1','YOUTUBE','Até "Quando"','Natanzinho Lima',144000,NULL,NULL,'completed',1.0,'/m/a.m4a',NULL,4000);
INSERT INTO preference VALUES ('search_history','["rihanna"]');
INSERT INTO preference VALUES ('settings','{"autoplay":false,"audioQuality":"HIGH","downloadWorkers":2,"normalizeVolume":false,"skipSilence":false,"soundWave":true}');
`
	cria := exec.Command(sqlite3, filepath.Join(pasta, BancoAntigo), esquema)
	if saida, erro := cria.CombinedOutput(); erro != nil {
		t.Skipf("não deu para criar o banco de teste: %v %s", erro, saida)
	}
	g := abre(t, pasta)
	feito, migrou, erro := g.Migra(context.Background(), sqlite3)
	if erro != nil || !migrou {
		t.Fatalf("migrou=%v erro=%v", migrou, erro)
	}
	igual(t, Migrado{Curtidas: 2, Playlists: 1, Tocadas: 1, Salvos: 1, Downloads: 1}, feito)
	igual(t, []string{"l2", "l1"}, prova.IDs(g.Curtidas()))
	igual(t, `Até "Quando"`, g.Curtidas()[1].Titulo)
	igual(t, []string{"l1", "l2"}, prova.IDs(g.Playlists()[0].Faixas))
	igual(t, "Anti", g.Salvos()[0].Titulo)
	igual(t, "/m/a.m4a", g.Downloads()[0].Caminho)
	igual(t, []string{"rihanna"}, g.Buscas())
	igual(t, false, g.Ajustes().Autoplay)
	igual(t, 2, g.Ajustes().Trabalhadores)
}
