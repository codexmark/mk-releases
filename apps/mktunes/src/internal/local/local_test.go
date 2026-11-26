// SPDX-License-Identifier: GPL-3.0-or-later

package local

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"mktunes/internal/dominio"
)

func igual(t *testing.T, esperado, obtido any) {
	t.Helper()
	if !reflect.DeepEqual(esperado, obtido) {
		t.Fatalf("esperado %#v\n  obtido %#v", esperado, obtido)
	}
}

type bancada struct {
	raiz       string
	mu         sync.Mutex
	perguntado []string
	etiquetas  map[string]Etiquetas
}

func nova(t *testing.T) *bancada {
	return &bancada{raiz: t.TempDir(), etiquetas: map[string]Etiquetas{}}
}

func (b *bancada) le(_ context.Context, arquivo string) (Etiquetas, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.perguntado = append(b.perguntado, filepath.Base(arquivo))
	e, tem := b.etiquetas[filepath.Base(arquivo)]
	return e, tem
}

func (b *bancada) arquivo(t *testing.T, caminho, conteudo string) string {
	t.Helper()
	inteiro := filepath.Join(b.raiz, caminho)
	os.MkdirAll(filepath.Dir(inteiro), 0o755)
	if erro := os.WriteFile(inteiro, []byte(conteudo), 0o644); erro != nil {
		t.Fatal(erro)
	}
	return inteiro
}

func titulos(faixas []dominio.Faixa) []string {
	var lista []string
	for _, f := range faixas {
		lista = append(lista, f.Titulo)
	}
	return lista
}

func TestPastaDeMusica(t *testing.T) {
	ctx := context.Background()
	t.Run("acha as músicas de todas as pastas, mas não outros arquivos nem os downloads do próprio app", func(t *testing.T) {
		b := nova(t)
		b.arquivo(t, "Forró/Natanzinho Lima - Sonho de Amor.mp3", "x")
		b.arquivo(t, "Forró/capa.jpg", "x")
		solta := b.arquivo(t, "solta.flac", "x")
		b.arquivo(t, "MkTunes/Baixada.webm", "x")
		faixas := (&Pasta{Raiz: b.raiz, Pula: filepath.Join(b.raiz, "MkTunes"), Le: b.le}).Varre(ctx)
		igual(t, []string{"Sonho de Amor", "solta"}, titulos(faixas))
		igual(t, dominio.Local, faixas[0].Provedor)
		igual(t, "local:"+solta, faixas[1].ID)
		// O mpv quer "file:///" e o caminho com acentos e espaços escapados.
		igual(t, true, strings.HasPrefix(faixas[1].Stream, "file:///") && strings.HasSuffix(faixas[1].Stream, "/solta.flac"))
		igual(t, true, strings.HasSuffix(faixas[0].Stream, "/Forr%C3%B3/Natanzinho%20Lima%20-%20Sonho%20de%20Amor.mp3"))
	})
	t.Run("as etiquetas ganham; sem elas o nome diz artista - título, ou a pasta é o artista", func(t *testing.T) {
		b := nova(t)
		b.arquivo(t, "Pagode/Ferrugem - Pirata e Tesouro.mp3", "x")
		b.arquivo(t, "Pagode/sem traço.ogg", "x")
		b.arquivo(t, "tagueada.m4a", "x")
		b.etiquetas["tagueada.m4a"] = Etiquetas{Titulo: "Envolver", Artista: "Anitta", Duracao: 194 * time.Second}
		por := map[string]dominio.Faixa{}
		for _, f := range (&Pasta{Raiz: b.raiz, Le: b.le}).Varre(ctx) {
			por[f.Titulo] = f
		}
		igual(t, "Ferrugem", por["Pirata e Tesouro"].Artista)
		igual(t, "Pagode", por["sem traço"].Artista)
		igual(t, "Anitta", por["Envolver"].Artista)
		igual(t, 194*time.Second, por["Envolver"].Duracao)
	})
	t.Run("por artista e depois título, sem olhar a caixa; sem artista conhecido por último", func(t *testing.T) {
		b := nova(t)
		for _, nome := range []string{"b - Zeta.mp3", "A - beta.mp3", "a - Alfa.mp3", "perdida.mp3"} {
			b.arquivo(t, nome, "x")
		}
		igual(t, []string{"Alfa", "beta", "Zeta", "perdida"}, titulos((&Pasta{Raiz: b.raiz, Le: b.le}).Varre(ctx)))
	})
	t.Run("a segunda olhada só lê as etiquetas do que mudou, e esquece o que saiu", func(t *testing.T) {
		b := nova(t)
		um := b.arquivo(t, "um.mp3", "x")
		b.arquivo(t, "dois.mp3", "x")
		pasta := &Pasta{Raiz: b.raiz, Le: b.le}
		pasta.Varre(ctx)
		b.perguntado = nil
		b.arquivo(t, "dois.mp3", "mudou bastante")
		pasta.Varre(ctx)
		igual(t, []string{"dois.mp3"}, b.perguntado)
		os.Remove(um)
		igual(t, 1, len(pasta.Varre(ctx)))
		igual(t, 1, len(pasta.vistos))
	})
	t.Run("uma pasta que não existe é só uma lista vazia", func(t *testing.T) {
		igual(t, 0, len((&Pasta{Raiz: "/não/existe/mesmo", Le: nova(t).le}).Varre(ctx)))
	})
}

func TestEtiquetasDoFfprobe(t *testing.T) {
	e, ok := LeFfprobe([]byte(`{"format":{"duration":"157.5","tags":{"TITLE":"Envolver","Artist":" Anitta ","encoder":"x"}}}`))
	igual(t, true, ok)
	igual(t, Etiquetas{"Envolver", "Anitta", 157500 * time.Millisecond}, e)
	e, ok = LeFfprobe([]byte(`{"format":{"duration":"12.0"}}`))
	igual(t, true, ok)
	igual(t, Etiquetas{Duracao: 12 * time.Second}, e)
	for _, ruim := range []string{"", "{}", "lixo", `{"format":null}`} {
		if _, ok := LeFfprobe([]byte(ruim)); ok {
			t.Fatalf("devia recusar %q", ruim)
		}
	}
}

func TestFfprobeDeMentira(t *testing.T) {
	pasta := t.TempDir()
	comando, visto := filepath.Join(pasta, "ffprobe"), filepath.Join(pasta, "args")
	corpo := "#!/bin/sh\nprintf '%s\\n' \"$@\" >'" + visto + "'\necho '{\"format\":{\"duration\":\"3.5\",\"tags\":{\"title\":\"Um\",\"artist\":\"Dois\"}}}'\n"
	os.WriteFile(comando, []byte(corpo), 0o755)
	arquivo := filepath.Join(pasta, "a b.mp3")
	e, ok := Ffprobe(comando)(context.Background(), arquivo)
	igual(t, true, ok)
	igual(t, Etiquetas{"Um", "Dois", 3500 * time.Millisecond}, e)
	argumentos, _ := os.ReadFile(visto)
	linhas := strings.Split(strings.TrimSpace(string(argumentos)), "\n")
	igual(t, arquivo, linhas[len(linhas)-1])
	igual(t, true, sort.SearchStrings(linhas, "-v") >= 0 && strings.Contains(string(argumentos), "format=duration:format_tags=title,artist,TITLE,ARTIST"))
	_, ok = Ffprobe(filepath.Join(pasta, "não-existe"))(context.Background(), arquivo)
	igual(t, false, ok)
}
