// SPDX-License-Identifier: GPL-3.0-or-later

// Pacote local: as músicas que já estão neste computador — a pasta de música,
// percorrida atrás de arquivos de áudio, com as etiquetas lidas pelo ffprobe.
package local

import (
	"context"
	"encoding/json"
	"io/fs"
	"net/url"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"mktunes/internal/dominio"
)

// Etiquetas é o que as tags de um arquivo dizem; o que vier em branco é
// completado pelo nome do arquivo.
type Etiquetas struct {
	Titulo, Artista string
	Duracao         time.Duration
}

// Leitor lê as etiquetas de um arquivo; false quando não dá para ler.
type Leitor func(ctx context.Context, arquivo string) (Etiquetas, bool)

const (
	desconhecido = "Artista desconhecido"
	// maxArquivos é o teto de músicas listadas (e lembradas) de uma pasta.
	maxArquivos  = 20000
	prazoFfprobe = 5 * time.Second
	saidaDoProbe = 256 * 1024
)

var audio = map[string]bool{"mp3": true, "flac": true, "ogg": true, "opus": true, "oga": true, "m4a": true, "aac": true, "webm": true, "wav": true, "wma": true, "alac": true, "aiff": true}

type visto struct {
	mudou   time.Time
	tamanho int64
	faixa   dominio.Faixa
}

// Pasta é a pasta de música. As etiquetas são lidas alguns arquivos de cada
// vez e lembradas pelo tamanho e pela data, então olhar de novo só custa o que
// mudou; o que saiu da pasta sai da lembrança.
type Pasta struct {
	Raiz string
	// Pula é uma pasta que fica de fora (a dos downloads do próprio app).
	Pula string
	Le   Leitor
	// Juntos é quantos arquivos têm as etiquetas lidas ao mesmo tempo.
	Juntos int

	mu     sync.Mutex
	vistos map[string]visto
}

// Varre lista as músicas da pasta, por artista e título; as sem artista
// conhecido vão por último.
func (p *Pasta) Varre(ctx context.Context) []dominio.Faixa {
	raiz, pula := filepath.Clean(p.Raiz), ""
	if p.Pula != "" {
		pula = filepath.Clean(p.Pula)
	}
	type achado struct {
		caminho string
		info    fs.FileInfo
	}
	var arquivos []achado
	_ = filepath.WalkDir(raiz, func(caminho string, entrada fs.DirEntry, erro error) error {
		if erro != nil || ctx.Err() != nil {
			if entrada != nil && entrada.IsDir() {
				return fs.SkipDir
			}
			return ctx.Err()
		}
		if entrada.IsDir() {
			if caminho == pula {
				return fs.SkipDir
			}
			return nil
		}
		if !entrada.Type().IsRegular() || !audio[strings.ToLower(strings.TrimPrefix(filepath.Ext(caminho), "."))] {
			return nil
		}
		if info, erro := entrada.Info(); erro == nil {
			arquivos = append(arquivos, achado{caminho, info})
		}
		if len(arquivos) >= maxArquivos {
			return fs.SkipAll
		}
		return nil
	})

	juntos := p.Juntos
	if juntos <= 0 {
		juntos = 6
	}
	faixas := make([]dominio.Faixa, len(arquivos))
	vagas := make(chan struct{}, juntos)
	var espera sync.WaitGroup
	for i, a := range arquivos {
		p.mu.Lock()
		lembrado, tem := p.vistos[a.caminho]
		p.mu.Unlock()
		if tem && lembrado.mudou.Equal(a.info.ModTime()) && lembrado.tamanho == a.info.Size() {
			faixas[i] = lembrado.faixa
			continue
		}
		espera.Add(1)
		vagas <- struct{}{}
		go func() {
			defer func() { <-vagas; espera.Done() }()
			etiquetas, _ := p.Le(ctx, a.caminho)
			faixas[i] = monta(raiz, a.caminho, etiquetas)
		}()
	}
	espera.Wait()

	novos := make(map[string]visto, len(arquivos))
	for i, a := range arquivos {
		novos[a.caminho] = visto{a.info.ModTime(), a.info.Size(), faixas[i]}
	}
	p.mu.Lock()
	p.vistos = novos
	p.mu.Unlock()

	sort.SliceStable(faixas, func(i, j int) bool {
		a, b := faixas[i], faixas[j]
		if (a.Artista == desconhecido) != (b.Artista == desconhecido) {
			return b.Artista == desconhecido
		}
		if x, y := strings.ToLower(a.Artista), strings.ToLower(b.Artista); x != y {
			return x < y
		}
		return strings.ToLower(a.Titulo) < strings.ToLower(b.Titulo)
	})
	return faixas
}

// monta a faixa: as etiquetas ganham; sem elas o nome diz "artista - título",
// ou a pasta é o artista.
func monta(raiz, caminho string, e Etiquetas) dominio.Faixa {
	nome := strings.TrimSuffix(filepath.Base(caminho), filepath.Ext(caminho))
	titulo, artista := nome, desconhecido
	if pasta := filepath.Dir(caminho); pasta != raiz {
		artista = filepath.Base(pasta)
	}
	if antes, depois, tem := strings.Cut(nome, " - "); tem && strings.TrimSpace(antes) != "" && strings.TrimSpace(depois) != "" {
		artista, titulo = strings.TrimSpace(antes), strings.TrimSpace(depois)
	}
	if strings.TrimSpace(e.Titulo) != "" {
		titulo = e.Titulo
	}
	if strings.TrimSpace(e.Artista) != "" {
		artista = e.Artista
	}
	return dominio.Faixa{
		ID: "local:" + caminho, Provedor: dominio.Local, Titulo: titulo, Artista: artista, Duracao: e.Duracao,
		// O mpv quer "file:///" e o caminho com os acentos e espaços escapados.
		Stream: (&url.URL{Scheme: "file", Path: caminho}).String(),
	}
}

// Ffprobe lê as etiquetas com o ffprobe, que vem com o ffmpeg do mpv.
func Ffprobe(comando string) Leitor {
	return func(ctx context.Context, arquivo string) (Etiquetas, bool) {
		ctx, cancela := context.WithTimeout(ctx, prazoFfprobe)
		defer cancela()
		// O caminho é sempre absoluto: nunca é lido como opção.
		saida, erro := exec.CommandContext(ctx, comando, "-v", "quiet", "-of", "json",
			"-show_entries", "format=duration:format_tags=title,artist,TITLE,ARTIST", arquivo).Output()
		if erro != nil || len(saida) > saidaDoProbe {
			return Etiquetas{}, false
		}
		return LeFfprobe(saida)
	}
}

// LeFfprobe lê o JSON do ffprobe. O nome das tags muda de caixa entre os
// formatos (ID3, Vorbis): as duas são olhadas.
func LeFfprobe(saida []byte) (Etiquetas, bool) {
	var resposta struct {
		Format *struct {
			Duration string         `json:"duration"`
			Tags     map[string]any `json:"tags"`
		} `json:"format"`
	}
	if json.Unmarshal(saida, &resposta) != nil || resposta.Format == nil {
		return Etiquetas{}, false
	}
	tag := func(nome string) string {
		// Em ordem de nome, para duas grafias da mesma tag darem sempre a mesma.
		var chaves []string
		for chave := range resposta.Format.Tags {
			if strings.EqualFold(chave, nome) {
				chaves = append(chaves, chave)
			}
		}
		sort.Strings(chaves)
		for _, chave := range chaves {
			if texto, ok := resposta.Format.Tags[chave].(string); ok {
				return strings.TrimSpace(texto)
			}
		}
		return ""
	}
	e := Etiquetas{Titulo: tag("title"), Artista: tag("artist")}
	if segundos, erro := strconv.ParseFloat(resposta.Format.Duration, 64); erro == nil && segundos > 0 {
		e.Duracao = time.Duration(segundos * float64(time.Second))
	}
	return e, true
}
