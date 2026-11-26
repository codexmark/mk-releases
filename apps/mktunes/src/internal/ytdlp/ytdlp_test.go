// SPDX-License-Identifier: GPL-3.0-or-later

package ytdlp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"mktunes/internal/app"
	"mktunes/internal/dominio"
)

var _ app.Resolvedor = (*Resolvedor)(nil)

func igual(t *testing.T, esperado, obtido any) {
	t.Helper()
	if !reflect.DeepEqual(esperado, obtido) {
		t.Fatalf("esperado %#v\n  obtido %#v", esperado, obtido)
	}
}

var agora = time.Unix(1795348800, 0)

const video = "n8X9_MgEdCg"

func faixa(id string) dominio.Faixa { return dominio.Faixa{ID: id, Provedor: dominio.YouTube} }

// dublê escreve um yt-dlp de mentira: anota os argumentos de cada chamada (um
// por linha, e uma linha "--" no fim) e roda o corpo dado.
func duble(t *testing.T, corpo string) (comando, chamadas string) {
	t.Helper()
	pasta := t.TempDir()
	comando, chamadas = filepath.Join(pasta, "yt-dlp"), filepath.Join(pasta, "chamadas")
	texto := "#!/bin/sh\n{ for a in \"$@\"; do printf '%s\\n' \"$a\"; done; echo FIM; } >>'" + chamadas + "'\n" + corpo + "\n"
	if erro := os.WriteFile(comando, []byte(texto), 0o755); erro != nil {
		t.Fatal(erro)
	}
	return comando, chamadas
}

func vezes(t *testing.T, chamadas string) int {
	bruto, _ := os.ReadFile(chamadas)
	return strings.Count(string(bruto), "FIM\n")
}

func endereco(vence time.Time) string {
	return fmt.Sprintf("https://rr3---sn-abc.googlevideo.com/videoplayback?expire=%d&itag=140&mime=audio%%2Fmp4", vence.Unix())
}

func TestResolvePeloYtdlp(t *testing.T) {
	comando, chamadas := duble(t, "echo '"+endereco(agora.Add(6*time.Hour))+"'")
	db := 6.68
	r := &Resolvedor{Comando: comando, Agora: func() time.Time { return agora }, Volume: func(_ context.Context, id string) *float64 {
		igual(t, video, id)
		return &db
	}}
	stream, erro := r.Resolve(context.Background(), faixa(video))
	if erro != nil {
		t.Fatal(erro)
	}
	igual(t, endereco(agora.Add(6*time.Hour)), stream.URL)
	igual(t, "140", stream.Variante)
	igual(t, 6.68, *stream.LoudnessDb)
	bruto, _ := os.ReadFile(chamadas)
	igual(t, strings.Join([]string{"-f", SeletorAlta, "-g", "--no-playlist", "--no-warnings", "--", "https://music.youtube.com/watch?v=" + video, "FIM", ""}, "\n"), string(bruto))

	// O endereço fica guardado enquanto vale: a mesma faixa não chama o yt-dlp de novo.
	de, _ := r.Resolve(context.Background(), faixa(video))
	igual(t, stream, de)
	igual(t, 1, vezes(t, chamadas))
	// Perto de vencer (ou vencido), resolve de novo.
	agora = agora.Add(6*time.Hour - time.Minute)
	defer func() { agora = time.Unix(1795348800, 0) }()
	r.Resolve(context.Background(), faixa(video))
	igual(t, 2, vezes(t, chamadas))
}

func TestQualidadeEscolheOSeletor(t *testing.T) {
	comando, chamadas := duble(t, "echo 'https://x.googlevideo.com/v?expire=9999999999'")
	r := &Resolvedor{Comando: comando}
	r.DefineQualidade(dominio.QualidadeEconomia)
	if _, erro := r.Resolve(context.Background(), faixa(video)); erro != nil {
		t.Fatal(erro)
	}
	bruto, _ := os.ReadFile(chamadas)
	igual(t, true, strings.Contains(string(bruto), "\n"+SeletorEconomia+"\n"))
	// A outra qualidade é outro endereço: não reaproveita o guardado.
	r.DefineQualidade(dominio.QualidadeAlta)
	r.Resolve(context.Background(), faixa(video))
	igual(t, 2, vezes(t, chamadas))
	igual(t, true, strings.Contains(SeletorEconomia, "abr<=128") && strings.Contains(SeletorAlta, "ext=m4a"))
}

func TestSoUmYtdlpPorFaixaDeCadaVez(t *testing.T) {
	comando, chamadas := duble(t, "sleep 0.4; echo 'https://x.googlevideo.com/v?expire=9999999999'")
	r := &Resolvedor{Comando: comando}
	var espera sync.WaitGroup
	for range 4 {
		espera.Add(1)
		go func() {
			defer espera.Done()
			if _, erro := r.Resolve(context.Background(), faixa(video)); erro != nil {
				t.Error(erro)
			}
		}()
	}
	espera.Wait()
	igual(t, 1, vezes(t, chamadas))
}

func TestQuemDesisteNaoEsperaMasOEnderecoFicaGuardado(t *testing.T) {
	comando, chamadas := duble(t, "sleep 0.3; echo 'https://x.googlevideo.com/v?expire=9999999999'")
	r := &Resolvedor{Comando: comando}
	ctx, cancela := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancela()
	_, erro := r.Resolve(ctx, faixa(video))
	igual(t, true, errors.Is(erro, app.ErrStream))
	stream, erro := r.Resolve(context.Background(), faixa(video))
	igual(t, nil, erro)
	igual(t, "https://x.googlevideo.com/v?expire=9999999999", stream.URL)
	igual(t, 1, vezes(t, chamadas))
}

func TestFalhasViramStreamIndisponivel(t *testing.T) {
	ctx := context.Background()
	falha := func(r *Resolvedor, f dominio.Faixa, trecho string) {
		t.Helper()
		_, erro := r.Resolve(ctx, f)
		if !errors.Is(erro, app.ErrStream) || !strings.Contains(erro.Error(), trecho) {
			t.Fatalf("erro: %v (esperava %q)", erro, trecho)
		}
	}
	// O erro do yt-dlp vem com a última linha do que ele reclamou.
	comando, chamadas := duble(t, "echo 'WARNING: algo' >&2; echo 'ERROR: [youtube] abc: Video unavailable' >&2; exit 1")
	r := &Resolvedor{Comando: comando}
	falha(r, faixa(video), "ERROR: [youtube] abc: Video unavailable")
	// Um id que não é de vídeo nem chega à linha de comando; outro provedor também não.
	for _, id := range []string{"", "curto", "--exec=rm -rf", "n8X9_MgEdCg;x", "../../etc/pw", "n8X9_MgEdCg\n"} {
		falha(r, faixa(id), "id de vídeo inválido")
	}
	falha(r, dominio.Faixa{ID: video, Provedor: dominio.Deezer}, "não é uma faixa do YouTube")
	igual(t, 1, vezes(t, chamadas))

	// Só endereços https do YouTube são aceitos.
	for _, ruim := range []string{"http://x.googlevideo.com/v", "https://evil.example/googlevideo.com", "https://googlevideo.com.evil.example/v", "file:///etc/passwd", "https://user@x.googlevideo.com/v", ""} {
		comando, _ := duble(t, "echo '"+ruim+"'")
		falha(&Resolvedor{Comando: comando}, faixa(video), "yt-dlp")
		igual(t, false, EnderecoAceito(ruim))
	}
	igual(t, true, EnderecoAceito("https://rr1---sn-x.googlevideo.com/videoplayback?a=b") && EnderecoAceito("https://manifest.googlevideo.com/api/manifest/hls_playlist/expire/1795348800/x") && EnderecoAceito("https://www.youtube.com/x"))

	// Quem demora demais é cortado.
	comando, _ = duble(t, "sleep 5")
	falha(&Resolvedor{Comando: comando, Prazo: 100 * time.Millisecond}, faixa(video), "demorou demais")
}

func TestSemYtdlpAvisaOPacoteUmaVez(t *testing.T) {
	var avisos []string
	r := &Resolvedor{Comando: filepath.Join(t.TempDir(), "yt-dlp-que-falta"), Aviso: func(texto string) { avisos = append(avisos, texto) }}
	for range 3 {
		if _, erro := r.Resolve(context.Background(), faixa(video)); !errors.Is(erro, app.ErrStream) {
			t.Fatalf("erro: %v", erro)
		}
	}
	igual(t, []string{AvisoSemYtdlp}, avisos)
	igual(t, true, strings.Contains(AvisoSemYtdlp, "yt-dlp"))
}

func TestOCacheTemTeto(t *testing.T) {
	comando, _ := duble(t, "echo 'https://x.googlevideo.com/v?expire=9999999999'")
	r := &Resolvedor{Comando: comando}
	for i := range guardados + 8 {
		if _, erro := r.Resolve(context.Background(), faixa(fmt.Sprintf("video%06d", i))); erro != nil {
			t.Fatal(erro)
		}
	}
	igual(t, guardados, len(r.cache))
	// Um endereço sem validade não é guardado.
	semValidade, chamadas := duble(t, "echo 'https://x.googlevideo.com/v'")
	r = &Resolvedor{Comando: semValidade}
	r.Resolve(context.Background(), faixa(video))
	r.Resolve(context.Background(), faixa(video))
	igual(t, 2, vezes(t, chamadas))
}
