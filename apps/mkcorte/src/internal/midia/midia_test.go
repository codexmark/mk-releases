package midia

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"mkcorte/internal/base"
	"mkcorte/internal/dubles"
)

// Um vídeo com capa (trilha 0), o vídeo na trilha 1 e duas trilhas de áudio.
const probeVideo = `{"streams":[
 {"index":0,"codec_name":"mjpeg","codec_type":"video","width":600,"height":600,"disposition":{"attached_pic":1}},
 {"index":1,"codec_name":"h264","profile":"High","level":40,"codec_type":"video","width":1920,"height":1080,"pix_fmt":"yuv420p",
  "sample_aspect_ratio":"1:1","r_frame_rate":"30/1","avg_frame_rate":"30000/1001","duration":"62.400000","bit_rate":"6000000","nb_frames":"1870",
  "disposition":{"attached_pic":0}},
 {"index":2,"codec_name":"aac","codec_type":"audio","channels":2,"sample_rate":"48000","bit_rate":"128000"},
 {"index":3,"codec_name":"ac3","codec_type":"audio","channels":6,"sample_rate":"48000","bit_rate":"448000"}],
 "format":{"format_name":"mov,mp4,m4a,3gp,3g2,mj2","duration":"62.400000","size":"48000000","bit_rate":"6153846"}}`

// Um vídeo de celular em pé, sem áudio, a 60 quadros.
const probeCelular = `{"streams":[{"index":0,"codec_name":"hevc","profile":"Main","level":120,"codec_type":"video","width":1920,"height":1080,
  "pix_fmt":"yuv420p","avg_frame_rate":"60/1","side_data_list":[{"side_data_type":"Display Matrix","rotation":-90}]}],
  "format":{"format_name":"mov,mp4","duration":"10.0","bit_rate":"8000000"}}`

// Uma gravação de tela em VP9 com Opus (webm), 4:3.
const probeTela = `{"streams":[{"index":0,"codec_name":"vp9","codec_type":"video","width":1024,"height":768,"pix_fmt":"yuv420p",
  "sample_aspect_ratio":"1:1","avg_frame_rate":"25/1"},
  {"index":1,"codec_name":"vorbis","codec_type":"audio","channels":1,"sample_rate":"44100"}],
  "format":{"format_name":"matroska,webm","duration":"20.5"}}`

func f(v float64) *float64 { return &v }
func n(v int64) *int64     { return &v }

// As fontes que os testes do plano usam, já sondadas.
func fonteHD() *Fonte {
	return &Fonte{Arquivo: "/v/passeio.mp4", Tamanho: 48_000_000, Duracao: 62.4, Largura: 1920, Altura: 1080, FPS: f(29.97), Fracao: "30000/1001",
		Codec: "h264", Perfil: "High", Pixel: "yuv420p", SAR: "1:1", Taxa: n(6_153_846), Trilha: 1, TemAudio: true,
		Audio: &Audio{Codec: "aac", Taxa: n(128000), Canais: 2, Amostragem: 48000, Trilha: 2}, gravado: [2]int{1920, 1080}, nivel: 40}
}

func fonteEmPe() *Fonte {
	return &Fonte{Arquivo: "/v/em pé.mov", Tamanho: 10_000_000, Duracao: 10, Largura: 1080, Altura: 1920, FPS: f(60), Fracao: "60/1",
		Codec: "hevc", Perfil: "Main", Pixel: "yuv420p", SAR: "1:1", Rotacao: 270, Taxa: n(8_000_000), Trilha: 0, gravado: [2]int{1920, 1080}, nivel: 120}
}

func fonteTela() *Fonte {
	return &Fonte{Arquivo: "/v/tela.webm", Tamanho: 5_000_000, Duracao: 20.5, Largura: 1024, Altura: 768, FPS: f(25), Fracao: "25/1",
		Codec: "vp9", Pixel: "yuv420p", SAR: "1:1", Trilha: 0, TemAudio: true, Audio: &Audio{Codec: "vorbis", Canais: 1, Amostragem: 44100, Trilha: 1},
		gravado: [2]int{1024, 768}}
}

func edicao(t *testing.T, texto string) *Edicao {
	t.Helper()
	e, err := LeEdicao([]byte(texto))
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func TestInterpreta_VideoComCapaEDuasTrilhasDeAudio(t *testing.T) {
	m, err := interpreta([]byte(probeVideo))
	if err != nil {
		t.Fatal(err)
	}
	quer := fonteHD()
	quer.Arquivo, quer.Tamanho = "", 0
	if !reflect.DeepEqual(m, quer) {
		t.Errorf("veio  %+v\nquer %+v\náudio %+v", m, quer, m.Audio)
	}
}

func TestInterpreta_CelularEmPePixelNaoQuadradoEOQueNaoEVideo(t *testing.T) {
	m, err := interpreta([]byte(probeCelular))
	if err != nil || m.Largura != 1080 || m.Altura != 1920 || m.Rotacao != 270 || m.Audio != nil || m.TemAudio || *m.Taxa != 8000000 || *m.FPS != 60 ||
		m.Fracao != "60/1" || m.SAR != "1:1" || m.gravado != [2]int{1920, 1080} {
		t.Errorf("celular em pé: %+v, %v", m, err)
	}
	// A rotação antiga, numa marca; 180 não troca os lados.
	m, _ = interpreta([]byte(`{"streams":[{"codec_type":"video","codec_name":"h264","width":640,"height":480,"avg_frame_rate":"0/0","r_frame_rate":"24/1",
	  "tags":{"rotate":"180"}}],"format":{"duration":"3"}}`))
	if m == nil || m.Rotacao != 180 || m.Largura != 640 || *m.FPS != 24 || m.Fracao != "24/1" {
		t.Errorf("rotate=180: %+v", m)
	}
	// DVD: 720x576 com pixel 16:15 aparece com 768 de largura.
	m, _ = interpreta([]byte(`{"streams":[{"codec_type":"video","codec_name":"mpeg2video","width":720,"height":576,"sample_aspect_ratio":"16:15",
	  "avg_frame_rate":"25/1"}],"format":{"duration":"3"}}`))
	if m == nil || m.Largura != 768 || m.Altura != 576 || m.SAR != "16:15" {
		t.Errorf("pixel não quadrado: %+v", m)
	}
	for _, sar := range []string{"0:1", "N/A", "", "1:0", "banana", "5:5", "1:1000"} {
		if v, texto := proporcaoDoPixel(sar); v != 1 || texto != "1:1" {
			t.Errorf("SAR %q: %v %q", sar, v, texto)
		}
	}
	erros := map[string]string{
		`{"streams":[{"codec_type":"audio","codec_name":"mp3"}],"format":{"format_name":"mp3","duration":"200"}}`: "só tem áudio",
		`{"streams":[{"codec_type":"subtitle"}],"format":{}}`:                                                     "não tem imagem",
		`{}`:              "não tem imagem",
		`isto não é json`: "não entendi",
		`{"streams":[{"codec_type":"video","codec_name":"png","width":10,"height":20}],"format":{"format_name":"png_pipe"}}`:                                   "imagem parada",
		`{"streams":[{"codec_type":"video","codec_name":"mjpeg","width":10,"height":20,"nb_frames":"1"}],"format":{"format_name":"image2","duration":"0.04"}}`: "imagem parada",
	}
	for entrada, trecho := range erros {
		if _, err := interpreta([]byte(entrada)); err == nil || !strings.Contains(err.Error(), trecho) {
			t.Errorf("%s: %v", entrada[:15], err)
		}
	}
}

// monta põe os dublês. O ffprobe responde pelo atalho que recebeu (fonte1.* é
// o celular, fonte2.* a gravação de tela, o resto o vídeo HD); o ffmpeg solta
// três blocos de progresso e escreve `saida` no último argumento.
func monta(t *testing.T, saida string) *dubles.Mesa {
	t.Helper()
	t.Setenv("MKCORTE_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("TMPDIR", t.TempDir())
	m := dubles.Nova(t)
	m.Comando("ffprobe", `for a in "$@"; do ultimo=$a; done
case " $* " in *" -select_streams "*) printf '0.000000,K__\n0.033000,___\n2.002000,K__\nlixo\n4.004000,K_D\nN/A,K__\n'; exit 0 ;; esac
case "$ultimo" in
  */fonte1|*/fonte1.*) printf '%s' `+dubles.Aspas(probeCelular)+` ;;
  */fonte2|*/fonte2.*) printf '%s' `+dubles.Aspas(probeTela)+` ;;
  *) printf '%s' `+dubles.Aspas(probeVideo)+` ;;
esac`)
	ffmpegDeMentira(m, saida)
	return m
}

func ffmpegDeMentira(m *dubles.Mesa, saida string) {
	m.Comando("ffmpeg", `for a in "$@"; do ultimo=$a; done
if [ -f lista.ffconcat ]; then while IFS= read -r l; do printf '%s\n' "$l"; done <lista.ffconcat >`+dubles.Aspas(filepath.Join(m.Dir, "lista-vista"))+`; fi
printf 'frame=10\nout_time_us=1000000\nout_time=00:00:01.0\nspeed=N/A\ntotal_size=1000\nprogress=continue\n'
printf 'out_time_us=2000000\nspeed= 2.5x \ntotal_size=2000\nprogress=continue\n'
printf 'out_time_us=4000000\nspeed=4x\ntotal_size=4000\nprogress=end\n'
printf '%s' `+dubles.Aspas(saida)+` >"$ultimo"`)
}

func original(t *testing.T, pasta, nome string, bytes int) string {
	t.Helper()
	if pasta == "" {
		pasta = t.TempDir()
	}
	caminho := filepath.Join(pasta, nome)
	if err := os.WriteFile(caminho, make([]byte, bytes), 0o644); err != nil {
		t.Fatal(err)
	}
	return caminho
}

func nomesDe(t *testing.T, pasta string) []string {
	t.Helper()
	entradas, err := os.ReadDir(pasta)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entradas {
		out = append(out, e.Name())
	}
	return out
}

// Um nome com tudo que costuma quebrar uma linha de comando.
const nomeRuim = "-i %03d *concat: 'aspas\" $(rm) `x` férias.MP4"

func TestSonda_OFfprobeRecebeUmAtalhoNuncaONomeDoArquivo(t *testing.T) {
	m := monta(t, "")
	arq := original(t, "", nomeRuim, 1000)
	fonte, err := Sonda(arq, false)
	if err != nil {
		t.Fatal(err)
	}
	if fonte.Arquivo != arq || fonte.Tamanho != 1000 || fonte.Duracao != 62.4 || !fonte.TemAudio || fonte.QuadrosChave != nil {
		t.Errorf("veio %+v", fonte)
	}
	c := m.Chamadas("ffprobe")
	if len(c) != 1 || len(c[0]) != 8 {
		t.Fatalf("ffprobe: %q", c)
	}
	if quer := []string{"-v", "error", "-of", "json", "-show_format", "-show_streams", "-i"}; !reflect.DeepEqual(c[0][:7], quer) {
		t.Errorf("ffprobe: %q", c[0])
	}
	link := c[0][7]
	if filepath.Base(link) != "fonte0.mp4" || !filepath.IsAbs(link) || strings.Contains(strings.Join(c[0], " "), "férias") {
		t.Errorf("o ffprobe viu %q", c[0])
	}
	if _, err := os.Lstat(filepath.Dir(link)); err == nil {
		t.Error("a pasta temporária ficou para trás")
	}
	// O JSON tem os campos do contrato.
	var campos map[string]any
	b, _ := json.Marshal(fonte)
	_ = json.Unmarshal(b, &campos)
	for _, k := range []string{"arquivo", "tamanho_bytes", "duracao_s", "largura", "altura", "fps", "fps_fracao", "codec", "perfil", "formato_de_pixel",
		"sar", "rotacao", "taxa_bps", "trilha", "tem_audio", "audio"} {
		if _, tem := campos[k]; !tem {
			t.Errorf("falta %q em %s", k, b)
		}
	}
	if _, tem := campos["quadros_chave"]; tem || len(campos) != 16 {
		t.Errorf("campos a mais: %s", b)
	}
}

func TestSonda_QuadrosChaveSoQuandoPedem(t *testing.T) {
	m := monta(t, "")
	fonte, err := Sonda(original(t, "", "a.mkv", 10), true)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(fonte.QuadrosChave, []float64{0, 2.002, 4.004}) {
		t.Errorf("quadros-chave: %v", fonte.QuadrosChave)
	}
	c := m.Chamadas("ffprobe")
	if len(c) != 2 || !reflect.DeepEqual(c[1][:8], []string{"-v", "error", "-select_streams", "1", "-show_entries", "packet=pts_time,flags", "-of", "csv=p=0"}) ||
		filepath.Base(c[1][len(c[1])-1]) != "fonte0.mkv" {
		t.Errorf("ffprobe: %q", c)
	}
}

func TestSonda_ErrosViramFrase(t *testing.T) {
	m := monta(t, "")
	arq := original(t, "", "a.mp4", 10)
	if _, err := Sonda(filepath.Join(filepath.Dir(arq), "nao-existe.mp4"), false); err == nil || !strings.Contains(err.Error(), "não achei") {
		t.Errorf("ausente: %v", err)
	}
	if _, err := Sonda(filepath.Dir(arq), false); err == nil || !strings.Contains(err.Error(), "não é um arquivo") {
		t.Errorf("pasta: %v", err)
	}
	m.Comando("ffprobe", `printf '%s\n' "$8: Invalid data found when processing input" >&2; exit 1`)
	if _, err := Sonda(arq, false); err == nil || !strings.Contains(err.Error(), "não parece um vídeo") || !strings.Contains(err.Error(), "a.mp4") {
		t.Errorf("ffprobe com erro: %v", err)
	}
	os.Remove(filepath.Join(m.Dir, "ffprobe"))
	if _, err := Sonda(arq, false); err == nil || !strings.Contains(err.Error(), "pacote ffmpeg") {
		t.Errorf("sem ffprobe: %v", err)
	}
}

func TestNomeDoAtalho_SoLetrasAlgarismosEPonto(t *testing.T) {
	casos := map[string]string{"/v/a.MP4": "fonte3.mp4", "/v/sem extensão": "fonte3", "/v/a.tar.gz": "fonte3.gz", "/v/a.m'p4": "fonte3", "/v/a.": "fonte3",
		"/v/-x.%03d": "fonte3", "/v/a.extensaolonga": "fonte3"}
	for arquivo, quer := range casos {
		if got := nomeDoAtalho(3, arquivo); got != quer {
			t.Errorf("%s: %q", arquivo, got)
		}
	}
}

func TestLeEdicao_ConfereOQueNaoDependeDosArquivos(t *testing.T) {
	e := edicao(t, `{"fontes":["/v/a.mp4"],"clipes":[{"fonte":0,"inicio":1.5,"fim":4}]}`)
	if e.Qualidade != "original" || e.Modo != "exato" || len(e.Clipes) != 1 || e.Clipes[0] != (Clipe{0, 1.5, 4}) {
		t.Errorf("veio %+v", e)
	}
	muitos := `{"fontes":["/a"],"clipes":[` + strings.Repeat(`{"fonte":0,"inicio":0,"fim":1},`, maximoDeClipes) + `{"fonte":0,"inicio":0,"fim":1}]}`
	erros := map[string]string{
		`não é json`:                    "não é um JSON",
		`{"fontes":[],"clipes":[]}`:     "está vazia",
		`{"fontes":["/a"],"clipes":[]}`: "está vazia",
		`{"fontes":["a.mp4"],"clipes":[{"fonte":0,"inicio":0,"fim":1}]}`:               "caminho absoluto",
		`{"fontes":["/a"],"clipes":[{"fonte":1,"inicio":0,"fim":1}]}`:                  "não está na edição",
		`{"fontes":["/a"],"clipes":[{"fonte":-1,"inicio":0,"fim":1}]}`:                 "não está na edição",
		`{"fontes":["/a"],"clipes":[{"fonte":0,"inicio":2,"fim":1}]}`:                  "trecho que não existe",
		`{"fontes":["/a"],"clipes":[{"fonte":0,"inicio":-1,"fim":1}]}`:                 "trecho que não existe",
		`{"fontes":["/a"],"clipes":[{"fonte":0,"inicio":1,"fim":1.001}]}`:              "trecho que não existe",
		`{"fontes":["/a"],"clipes":[{"fonte":0,"inicio":0,"fim":1}],"qualidade":"4k"}`: "não conheço a qualidade",
		`{"fontes":["/a"],"clipes":[{"fonte":0,"inicio":0,"fim":1}],"modo":"turbo"}`:   "não conheço o modo",
		muitos: "clipes demais",
	}
	for texto, trecho := range erros {
		if _, err := LeEdicao([]byte(texto)); err == nil || !strings.Contains(err.Error(), trecho) {
			t.Errorf("%.40s: %v", texto, err)
		}
	}
}

func TestDimensoes_NuncaAumentaPreservaAProporcaoEDeixaPar(t *testing.T) {
	casos := [][5]int{
		{1920, 1080, 0, 1920, 1080}, {1920, 1080, 1080, 1920, 1080}, {1920, 1080, 720, 1280, 720}, {3840, 2160, 1080, 1920, 1080},
		{1280, 720, 1080, 1280, 720}, {1080, 1920, 720, 720, 1280}, {1081, 1921, 0, 1080, 1920}, {641, 361, 0, 640, 360},
		{2560, 1080, 720, 1706, 720}, {1000, 1000, 720, 720, 720}, {1, 1, 0, 2, 2},
	}
	for _, c := range casos {
		if l, a := Dimensoes(c[0], c[1], c[2]); l != c[3] || a != c[4] || l%2 != 0 || a%2 != 0 {
			t.Errorf("%dx%d em %d: %dx%d", c[0], c[1], c[2], l, a)
		}
	}
}

func TestEncaixe_FaixasPretasSemDeformar(t *testing.T) {
	casos := [][8]int{
		{1080, 1920, 1920, 1080, 608, 1080, 656, 0}, // em pé num quadro deitado: faixas dos lados
		{1024, 768, 1920, 1080, 1440, 1080, 240, 0}, // 4:3 em 16:9
		{1920, 1080, 1080, 1920, 1080, 608, 0, 656}, // deitado num quadro em pé: faixas em cima e embaixo
		{1280, 720, 1920, 1080, 1920, 1080, 0, 0},   // mesma proporção: só escala
		{3840, 2160, 1280, 720, 1280, 720, 0, 0},
		{1921, 1080, 1920, 1080, 1920, 1080, 0, 0},
	}
	for _, c := range casos {
		sl, sa, x, y := encaixe(c[0], c[1], c[2], c[3])
		if sl != c[4] || sa != c[5] || x != c[6] || y != c[7] || sl > c[2] || sa > c[3] {
			t.Errorf("%dx%d em %dx%d: %dx%d +%d+%d", c[0], c[1], c[2], c[3], sl, sa, x, y)
		}
	}
}

// O começo de toda linha de comando do ffmpeg.
var inicio = []string{"-hide_banner", "-nostdin", "-loglevel", "error", "-y", "-progress", "pipe:1", "-nostats"}

// O fim de toda exportação recodificada com áudio.
var fimComAudio = []string{"-map", "[v]", "-map", "[a]", "-c:v", "libx264", "-preset", "medium", "-crf", "20", "-pix_fmt", "yuv420p",
	"-c:a", "aac", "-b:a", "160k", "-ar", "48000", "-ac", "2", "-movflags", "+faststart", "-f", "mp4", "{saida}"}

func junta(partes ...[]string) []string {
	var out []string
	for _, p := range partes {
		out = append(out, p...)
	}
	return out
}

func TestPlaneja_UmClipeSoAparaEExportaExatoAoQuadro(t *testing.T) {
	e := edicao(t, `{"fontes":["/v/passeio.mp4"],"clipes":[{"fonte":0,"inicio":1.5,"fim":11.75}]}`)
	p, err := Planeja(e, []*Fonte{fonteHD()}, "/v/passeio (corte).mp4")
	if err != nil {
		t.Fatal(err)
	}
	quer := junta(inicio, []string{"-ss", "1.500", "-t", "10.250", "-i", "{fonte0}", "-filter_complex",
		"[0:1]trim=duration=10.250,setpts=PTS-STARTPTS,setsar=1,format=yuv420p[v];" +
			"[0:2]atrim=duration=10.250,asetpts=PTS-STARTPTS,aresample=48000,aformat=sample_fmts=fltp:channel_layouts=stereo[a]"}, fimComAudio)
	if !reflect.DeepEqual(p.Args, quer) {
		t.Errorf("args:\n%q\nquer:\n%q", p.Args, quer)
	}
	if p.Modo != "exato" || p.Duracao != 10.25 || p.Clipes != 1 || p.Largura != 1920 || p.Altura != 1080 || *p.FPS != 29.97 || !p.TemAudio || p.Lista != "" ||
		len(p.Avisos) != 0 {
		t.Errorf("plano: %+v", p)
	}
	// A linha de comando mostrada: o atalho pelo nome e a saída por extenso.
	if p.Comando[0] != "ffmpeg" || p.Comando[len(p.Comando)-1] != "/v/passeio (corte).mp4" || p.Comando[14] != "fonte0.mp4" || len(p.Comando) != len(p.Args)+1 {
		t.Errorf("comando: %q", p.Comando)
	}
	// A estimativa: o vídeo não passa da taxa do original, mais o áudio.
	bits := 6_153_846*10.25 + 160_000*10.25
	if quer := int64(bits / 8); p.Tamanho != quer {
		t.Errorf("estimativa: %d, quer %d", p.Tamanho, quer)
	}
	if !p.SemRecodificar.Pode || p.SemRecodificar.Aviso != AvisoDoRapido || p.SemRecodificar.Motivo != "" {
		t.Errorf("sem recodificar: %+v", p.SemRecodificar)
	}
}

func TestPlaneja_VariosClipesDaMesmaFonteForaDeOrdemViramUmConcat(t *testing.T) {
	e := edicao(t, `{"fontes":["/v/passeio.mp4"],"clipes":[{"fonte":0,"inicio":30,"fim":40},{"fonte":0,"inicio":0,"fim":5.5},{"fonte":0,"inicio":40,"fim":62.4}]}`)
	p, err := Planeja(e, []*Fonte{fonteHD()}, "/s.mp4")
	if err != nil {
		t.Fatal(err)
	}
	a := "asetpts=PTS-STARTPTS,aresample=48000,aformat=sample_fmts=fltp:channel_layouts=stereo"
	grafo := "[0:1]trim=duration=10.000,setpts=PTS-STARTPTS,setsar=1,format=yuv420p[v0];[0:2]atrim=duration=10.000," + a + "[a0];" +
		"[1:1]trim=duration=5.500,setpts=PTS-STARTPTS,setsar=1,format=yuv420p[v1];[1:2]atrim=duration=5.500," + a + "[a1];" +
		"[2:1]trim=duration=22.400,setpts=PTS-STARTPTS,setsar=1,format=yuv420p[v2];[2:2]atrim=duration=22.400," + a + "[a2];" +
		"[v0][a0][v1][a1][v2][a2]concat=n=3:v=1:a=1[v][a]"
	quer := junta(inicio, []string{"-ss", "30.000", "-t", "10.000", "-i", "{fonte0}", "-ss", "0.000", "-t", "5.500", "-i", "{fonte0}",
		"-ss", "40.000", "-t", "22.400", "-i", "{fonte0}", "-filter_complex", grafo}, fimComAudio)
	if !reflect.DeepEqual(p.Args, quer) {
		t.Errorf("args:\n%q\nquer:\n%q", p.Args, quer)
	}
	if p.Duracao != 37.9 || len(p.Avisos) != 0 {
		t.Errorf("plano: %+v", p)
	}
}

func TestPlaneja_FontesDiferentesFaixasPretasSilencioETaxaDoPrimeiro(t *testing.T) {
	e := edicao(t, `{"fontes":["/v/passeio.mp4","/v/em pé.mov","/v/tela.webm"],
	  "clipes":[{"fonte":0,"inicio":0,"fim":4},{"fonte":1,"inicio":2,"fim":5},{"fonte":2,"inicio":1,"fim":2.5}],"qualidade":"720"}`)
	p, err := Planeja(e, []*Fonte{fonteHD(), fonteEmPe(), fonteTela()}, "/s.mp4")
	if err != nil {
		t.Fatal(err)
	}
	a := "asetpts=PTS-STARTPTS,aresample=48000,aformat=sample_fmts=fltp:channel_layouts=stereo"
	grafo := "[0:1]trim=duration=4.000,setpts=PTS-STARTPTS,scale=1280:720,setsar=1,fps=30000/1001,format=yuv420p[v0];[0:2]atrim=duration=4.000," + a + "[a0];" +
		"[1:0]trim=duration=3.000,setpts=PTS-STARTPTS,scale=406:720,pad=1280:720:437:0:color=black,setsar=1,fps=30000/1001,format=yuv420p[v1];" +
		"anullsrc=channel_layout=stereo:sample_rate=48000,atrim=duration=3.000[a1];" +
		"[2:0]trim=duration=1.500,setpts=PTS-STARTPTS,scale=960:720,pad=1280:720:160:0:color=black,setsar=1,fps=30000/1001,format=yuv420p[v2];" +
		"[2:1]atrim=duration=1.500," + a + "[a2];" +
		"[v0][a0][v1][a1][v2][a2]concat=n=3:v=1:a=1[v][a]"
	quer := junta(inicio, []string{"-ss", "0.000", "-t", "4.000", "-i", "{fonte0}", "-ss", "2.000", "-t", "3.000", "-i", "{fonte1}",
		"-ss", "1.000", "-t", "1.500", "-i", "{fonte2}", "-filter_complex", grafo}, fimComAudio)
	if !reflect.DeepEqual(p.Args, quer) {
		t.Errorf("args:\n%q\nquer:\n%q", p.Args, quer)
	}
	if p.Largura != 1280 || p.Altura != 720 || p.Duracao != 8.5 || !p.TemAudio {
		t.Errorf("plano: %+v", p)
	}
	if p.SemRecodificar.Pode || !strings.Contains(p.SemRecodificar.Motivo, "reduzir para 720p") {
		t.Errorf("sem recodificar: %+v", p.SemRecodificar)
	}
	avisos := strings.Join(p.Avisos, " | ")
	for _, trecho := range []string{"faixas pretas", "silêncio", "taxa de quadros do primeiro"} {
		if !strings.Contains(avisos, trecho) {
			t.Errorf("faltou o aviso %q em %q", trecho, avisos)
		}
	}
	if strings.Join(p.Comando[11:17], " ") != "-t 4.000 -i fonte0.mp4 -ss 2.000" || !strings.Contains(strings.Join(p.Comando, " "), "-i fonte1.mov -ss 1.000 -t 1.500 -i fonte2.webm") {
		t.Errorf("comando: %q", p.Comando)
	}
}

func TestPlaneja_OQuadroSegueOPrimeiroClipeENenhumaFonteComAudioSaiSemAudio(t *testing.T) {
	// O primeiro clipe é o vídeo em pé: o quadro é o dele, e o HD ganha
	// faixas em cima e embaixo.
	semAudio := fonteHD()
	semAudio.Audio, semAudio.TemAudio = nil, false
	e := edicao(t, `{"fontes":["/v/passeio.mp4","/v/em pé.mov"],"clipes":[{"fonte":1,"inicio":0,"fim":2},{"fonte":0,"inicio":0,"fim":2}],"qualidade":"1080"}`)
	p, err := Planeja(e, []*Fonte{semAudio, fonteEmPe()}, "/s.mp4")
	if err != nil {
		t.Fatal(err)
	}
	grafo := "[0:0]trim=duration=2.000,setpts=PTS-STARTPTS,setsar=1,fps=60/1,format=yuv420p[v0];" +
		"[1:1]trim=duration=2.000,setpts=PTS-STARTPTS,scale=1080:608,pad=1080:1920:0:656:color=black,setsar=1,fps=60/1,format=yuv420p[v1];" +
		"[v0][v1]concat=n=2:v=1:a=0[v]"
	quer := junta(inicio, []string{"-ss", "0.000", "-t", "2.000", "-i", "{fonte1}", "-ss", "0.000", "-t", "2.000", "-i", "{fonte0}", "-filter_complex", grafo,
		"-map", "[v]", "-c:v", "libx264", "-preset", "medium", "-crf", "20", "-pix_fmt", "yuv420p", "-movflags", "+faststart", "-f", "mp4", "{saida}"})
	if !reflect.DeepEqual(p.Args, quer) {
		t.Errorf("args:\n%q\nquer:\n%q", p.Args, quer)
	}
	if p.Largura != 1080 || p.Altura != 1920 || p.TemAudio || *p.FPS != 60 {
		t.Errorf("plano: %+v", p)
	}
	for _, a := range p.Args {
		if strings.Contains(a, "anullsrc") || a == "-c:a" {
			t.Errorf("áudio num plano sem áudio: %q", a)
		}
	}
}

func TestPlaneja_PixelNaoQuadradoEDimensaoImparPassamPelaEscala(t *testing.T) {
	dvd := &Fonte{Arquivo: "/v/dvd.mpg", Duracao: 100, Largura: 768, Altura: 576, SAR: "16:15", Codec: "mpeg2video", Trilha: 0, gravado: [2]int{720, 576}}
	p, err := Planeja(edicao(t, `{"fontes":["/v/dvd.mpg"],"clipes":[{"fonte":0,"inicio":0,"fim":10}]}`), []*Fonte{dvd}, "/s.mp4")
	if err != nil || !strings.Contains(strings.Join(p.Args, " "), "[0:0]trim=duration=10.000,setpts=PTS-STARTPTS,scale=768:576,setsar=1,format=yuv420p[v]") {
		t.Errorf("dvd: %v %+v", err, p)
	}
	// Sem taxa de quadros nem de bits conhecidas: a estimativa usa 30 quadros.
	bitsDoDvd := 768 * 576 * 30 * bitsPorPx * 10.0
	if quer := int64(bitsDoDvd / 8); p.Tamanho != quer {
		t.Errorf("estimativa: %d, quer %d", p.Tamanho, quer)
	}
	impar := fonteHD()
	impar.Largura, impar.Altura = 1921, 1081
	p, err = Planeja(edicao(t, `{"fontes":["/v/a.mp4"],"clipes":[{"fonte":0,"inicio":0,"fim":10}]}`), []*Fonte{impar}, "/s.mp4")
	if err != nil || p.Largura != 1920 || p.Altura != 1080 || !strings.Contains(strings.Join(p.Args, " "), ",scale=1920:1080,setsar=1,") {
		t.Errorf("ímpar: %v %+v", err, p)
	}
	if p.SemRecodificar.Pode || !strings.Contains(p.SemRecodificar.Motivo, "dimensão ímpar") {
		t.Errorf("ímpar sem recodificar: %+v", p.SemRecodificar)
	}
}

func TestPlaneja_ClipeQuePassaDoFimDoVideo(t *testing.T) {
	// Um fio além do fim (arredondamento da janela): corta no fim.
	p, err := Planeja(edicao(t, `{"fontes":["/v/a.mp4"],"clipes":[{"fonte":0,"inicio":60,"fim":62.45}]}`), []*Fonte{fonteHD()}, "/s.mp4")
	if err != nil || p.Duracao != 2.4 || p.Args[11] != "2.400" {
		t.Errorf("folga: %v %+v", err, p)
	}
	for texto, trecho := range map[string]string{
		`{"fontes":["/v/a.mp4"],"clipes":[{"fonte":0,"inicio":10,"fim":70}]}`:      "passa do fim",
		`{"fontes":["/v/a.mp4"],"clipes":[{"fonte":0,"inicio":62.4,"fim":62.8}]}`:  "passa do fim",
		`{"fontes":["/v/a.mp4"],"clipes":[{"fonte":0,"inicio":62.39,"fim":62.8}]}`: "curto demais",
	} {
		if _, err := Planeja(edicao(t, texto), []*Fonte{fonteHD()}, "/s.mp4"); err == nil || !strings.Contains(err.Error(), trecho) {
			t.Errorf("%s: %v", texto, err)
		}
	}
	if _, err := Planeja(edicao(t, `{"fontes":["/a","/b"],"clipes":[{"fonte":0,"inicio":0,"fim":1}]}`), []*Fonte{fonteHD()}, "/s.mp4"); err == nil {
		t.Error("faltou uma sondagem e passou")
	}
}

func TestPlaneja_RapidoCopiaOsPacotesPelaListaDoConcat(t *testing.T) {
	e := edicao(t, `{"fontes":["/v/passeio.mp4"],"clipes":[{"fonte":0,"inicio":30,"fim":40},{"fonte":0,"inicio":0,"fim":5.5}],"modo":"rapido","qualidade":"1080"}`)
	p, err := Planeja(e, []*Fonte{fonteHD()}, "/v/s.mp4")
	if err != nil {
		t.Fatal(err)
	}
	quer := junta(inicio, []string{"-f", "concat", "-safe", "0", "-i", "{lista}", "-map", "0:1", "-map", "0:2", "-codec", "copy",
		"-avoid_negative_ts", "make_zero", "-movflags", "+faststart", "-f", "mp4", "{saida}"})
	if !reflect.DeepEqual(p.Args, quer) {
		t.Errorf("args:\n%q\nquer:\n%q", p.Args, quer)
	}
	if quer := "ffconcat version 1.0\nfile 'fonte0.mp4'\ninpoint 30.000\noutpoint 40.000\nfile 'fonte0.mp4'\noutpoint 5.500\n"; p.Lista != quer {
		t.Errorf("lista:\n%s", p.Lista)
	}
	if p.Modo != "rapido" || p.Duracao != 15.5 || !p.TemAudio || len(p.Avisos) != 1 || p.Avisos[0] != AvisoDoRapido {
		t.Errorf("plano: %+v", p)
	}
	// A estimativa é a fatia do arquivo: 15,5 s de 62,4 s.
	fatia := 48_000_000 * 15.5 / 62.4
	if quer := int64(fatia); p.Tamanho < quer-1 || p.Tamanho > quer+1 {
		t.Errorf("estimativa: %d, quer %d", p.Tamanho, quer)
	}
	if !strings.Contains(strings.Join(p.Comando, " "), "-i lista.ffconcat -map 0:1") {
		t.Errorf("comando: %q", p.Comando)
	}
	// A franqueza: o aviso diz que o corte cai no quadro-chave.
	for _, trecho := range []string{"quadro-chave", "antes do ponto marcado", "modo exato"} {
		if !strings.Contains(AvisoDoRapido, trecho) {
			t.Errorf("o aviso não diz %q", trecho)
		}
	}
}

func TestPlaneja_RapidoSoQuandoOsParametrosSaoIdenticos(t *testing.T) {
	doisIguais := `{"fontes":["/v/a.mp4","/v/b.mp4"],"clipes":[{"fonte":0,"inicio":0,"fim":5},{"fonte":1,"inicio":1,"fim":3}],"modo":"rapido"}`
	gemea := fonteHD()
	gemea.Arquivo, gemea.Tamanho, gemea.Duracao = "/v/b.mp4", 1_000_000, 8
	p, err := Planeja(edicao(t, doisIguais), []*Fonte{fonteHD(), gemea}, "/s.mp4")
	if err != nil || p.Modo != "rapido" || !strings.Contains(p.Lista, "file 'fonte1.mp4'\ninpoint 1.000\noutpoint 3.000\n") || len(p.Avisos) != 2 ||
		!strings.Contains(p.Avisos[1], "arquivos diferentes") {
		t.Errorf("gêmeas: %v %+v", err, p)
	}

	// Cada diferença tira o modo rápido, e pedir assim mesmo é recusado.
	muda := map[string]func(*Fonte){
		"codec":      func(f *Fonte) { f.Codec = "hevc" },
		"perfil":     func(f *Fonte) { f.Perfil = "Main" },
		"nível":      func(f *Fonte) { f.nivel = 41 },
		"resolução":  func(f *Fonte) { f.Largura, f.Altura, f.gravado = 1280, 720, [2]int{1280, 720} },
		"pixel":      func(f *Fonte) { f.Pixel = "yuv420p10le" },
		"sar":        func(f *Fonte) { f.SAR = "4:3" },
		"quadros":    func(f *Fonte) { f.Fracao = "25/1" },
		"rotação":    func(f *Fonte) { f.Rotacao = 180 },
		"trilha":     func(f *Fonte) { f.Trilha = 0 },
		"sem áudio":  func(f *Fonte) { f.Audio, f.TemAudio = nil, false },
		"áudio":      func(f *Fonte) { f.Audio = &Audio{Codec: "mp3", Canais: 2, Amostragem: 48000, Trilha: 2} },
		"canais":     func(f *Fonte) { f.Audio = &Audio{Codec: "aac", Canais: 6, Amostragem: 48000, Trilha: 2} },
		"amostragem": func(f *Fonte) { f.Audio = &Audio{Codec: "aac", Canais: 2, Amostragem: 44100, Trilha: 2} },
	}
	for nome, f := range muda {
		outra := fonteHD()
		f(outra)
		_, err := Planeja(edicao(t, doisIguais), []*Fonte{fonteHD(), outra}, "/s.mp4")
		if err == nil || !base.FoiRecusa(err) || !strings.Contains(err.Error(), "parâmetros diferentes") {
			t.Errorf("%s: %v", nome, err)
		}
		// No modo exato o plano sai, dizendo por que o rápido não dá.
		p, err := Planeja(edicao(t, strings.Replace(doisIguais, `"rapido"`, `"exato"`, 1)), []*Fonte{fonteHD(), outra}, "/s.mp4")
		if err != nil || p.Modo != "exato" || p.SemRecodificar.Pode || p.SemRecodificar.Aviso != "" {
			t.Errorf("%s no exato: %v %+v", nome, err, p)
		}
	}
	// Uma fonte que a edição lista mas nenhum clipe usa não atrapalha.
	p, err = Planeja(edicao(t, `{"fontes":["/v/a.mp4","/v/b.mov"],"clipes":[{"fonte":0,"inicio":0,"fim":5}],"modo":"rapido"}`), []*Fonte{fonteHD(), fonteEmPe()}, "/s.mp4")
	if err != nil || p.Modo != "rapido" {
		t.Errorf("fonte sem uso: %v", err)
	}
}

func TestPlaneja_RapidoSoComCodecQueEntraNumMP4(t *testing.T) {
	um := `{"fontes":["/v/a"],"clipes":[{"fonte":0,"inicio":0,"fim":5}],"modo":"rapido"}`
	if _, err := Planeja(edicao(t, um), []*Fonte{fonteTela()}, "/s.mp4"); !base.FoiRecusa(err) || !strings.Contains(err.Error(), "o áudio está em vorbis") {
		t.Errorf("vorbis: %v", err)
	}
	velho := fonteHD()
	velho.Codec = "wmv3"
	if _, err := Planeja(edicao(t, um), []*Fonte{velho}, "/s.mp4"); !base.FoiRecusa(err) || !strings.Contains(err.Error(), "o vídeo está em wmv3") {
		t.Errorf("wmv: %v", err)
	}
	// Reduzir pede recodificar; pedir 1080 de um vídeo que já é menor, não.
	if _, err := Planeja(edicao(t, strings.Replace(um, `"modo"`, `"qualidade":"720","modo"`, 1)), []*Fonte{fonteHD()}, "/s.mp4"); !base.FoiRecusa(err) ||
		!strings.Contains(err.Error(), "reduzir para 720p") {
		t.Errorf("720: %v", err)
	}
	// HEVC sem áudio (o vídeo em pé): só o vídeo, com a marca que a Apple pede; a rotação vai junto na cópia.
	p, err := Planeja(edicao(t, strings.Replace(um, `"modo"`, `"qualidade":"1080","modo"`, 1)), []*Fonte{fonteEmPe()}, "/s.mp4")
	quer := junta(inicio, []string{"-f", "concat", "-safe", "0", "-i", "{lista}", "-map", "0:0", "-codec", "copy", "-tag:v", "hvc1",
		"-avoid_negative_ts", "make_zero", "-movflags", "+faststart", "-f", "mp4", "{saida}"})
	if err != nil || !reflect.DeepEqual(p.Args, quer) || p.TemAudio || p.Largura != 1080 || p.Altura != 1920 {
		t.Errorf("hevc: %v\n%q", err, p.Args)
	}
}

func TestNomes_SaidaPadraoNomeLivreEExtensao(t *testing.T) {
	casos := map[string]string{"/v/passeio.mp4": "/v/passeio (corte).mp4", "/v/a b.tar.MKV": "/v/a b.tar (corte).mp4", "/v/sem": "/v/sem (corte).mp4",
		"/v/.oculto": "/v/.oculto (corte).mp4"}
	for arquivo, quer := range casos {
		if got := SaidaPadrao(arquivo); got != quer {
			t.Errorf("%s: %q", arquivo, got)
		}
	}
	ext := map[string]string{"/v/a.mp4": "/v/a.mp4", "/v/a.MP4": "/v/a.MP4", "/v/a.mkv": "/v/a.mp4", "/v/a": "/v/a.mp4", "/v/a.b c": "/v/a.b c.mp4",
		"/v/.mkv": "/v/.mkv.mp4"}
	for caminho, quer := range ext {
		if got := comExtensao(caminho, ".mp4"); got != quer {
			t.Errorf("%s: %q", caminho, got)
		}
	}
	pasta := t.TempDir()
	livre := filepath.Join(pasta, "a (corte).mp4")
	if got := NomeLivre(livre); got != livre {
		t.Errorf("livre: %q", got)
	}
	original(t, pasta, "a (corte).mp4", 1)
	original(t, pasta, "a (corte) (2).mp4", 1)
	// Um link quebrado também ocupa o nome.
	if err := os.Symlink("/nao/existe", filepath.Join(pasta, "a (corte) (3).mp4")); err != nil {
		t.Fatal(err)
	}
	if got := NomeLivre(livre); got != filepath.Join(pasta, "a (corte) (4).mp4") {
		t.Errorf("ocupado: %q", got)
	}
	for tempo, quer := range map[float64]string{0: "0-00-00.000", 83.45: "0-01-23.450", 3725.0004: "1-02-05.000", 59.9996: "0-01-00.000"} {
		if got := carimbo(tempo); got != quer {
			t.Errorf("carimbo de %v: %q", tempo, got)
		}
	}
	if got := NomeDoQuadro("/v/a b.mkv", 83.45); got != "/v/a b (quadro 0-01-23.450).png" {
		t.Errorf("nome do quadro: %q", got)
	}
}

// exporta roda Exporta guardando as fases como JSON.
func exporta(t *testing.T, e *Edicao, saida string) (*Fim, []map[string]any, error) {
	t.Helper()
	var fases []map[string]any
	fim, err := Exporta(e, saida, func(v any) error {
		b, _ := json.Marshal(v)
		var m map[string]any
		_ = json.Unmarshal(b, &m)
		fases = append(fases, m)
		return nil
	}, nil)
	return fim, fases, err
}

func TestExporta_NomesRuinsNuncaChegamAoFfmpegEASaidaFicaAoLadoDoPrimeiro(t *testing.T) {
	m := monta(t, "vídeo pronto")
	// A pasta também tem espaço, aspas e dois-pontos.
	pasta := filepath.Join(t.TempDir(), "meus: 'vídeos' \"de\" férias")
	if err := os.MkdirAll(pasta, 0o755); err != nil {
		t.Fatal(err)
	}
	a := original(t, pasta, nomeRuim, 48000)
	b := original(t, pasta, "-outro: 'b'.mov", 1000)
	texto, _ := json.Marshal(Edicao{Fontes: []string{a, b}, Clipes: []Clipe{{0, 1, 3}, {1, 0, 2}}})
	fim, fases, err := exporta(t, edicao(t, string(texto)), "")
	if err != nil {
		t.Fatal(err)
	}
	querSaida := filepath.Join(pasta, "-i %03d *concat: 'aspas\" $(rm) `x` férias (corte).mp4")
	if !fim.Ok || fim.Fase != "fim" || fim.Saida != querSaida || fim.Modo != "exato" || fim.Duracao != 4 || fim.Tamanho != int64(len("vídeo pronto")) {
		t.Errorf("fim: %+v", fim)
	}
	if got, _ := os.ReadFile(querSaida); string(got) != "vídeo pronto" {
		t.Errorf("a saída tem %q", got)
	}
	if nomes := nomesDe(t, pasta); len(nomes) != 3 {
		t.Errorf("sobrou algo na pasta: %q", nomes)
	}
	if fi, _ := os.Stat(a); fi == nil || fi.Size() != 48000 {
		t.Error("o original mudou")
	}

	// O ffmpeg rodou uma vez, com os atalhos e a saída parcial: nenhum
	// argumento tem o nome de verdade dos vídeos.
	c := m.Chamadas("ffmpeg")
	if len(c) != 1 {
		t.Fatalf("ffmpeg: %d chamadas", len(c))
	}
	args := c[0]
	for _, x := range args {
		if strings.Contains(x, "férias.MP4") || strings.Contains(x, "outro") || strings.Contains(x, "$(rm)") {
			t.Errorf("o ffmpeg viu um nome de verdade: %q", x)
		}
	}
	if filepath.Base(args[13]) != "fonte0.mp4" || filepath.Base(args[19]) != "fonte1.mov" || !filepath.IsAbs(args[13]) || args[12] != "-i" || args[18] != "-i" {
		t.Errorf("entradas: %q", args[8:20])
	}
	parcial := args[len(args)-1]
	if filepath.Dir(parcial) != pasta || !strings.HasPrefix(filepath.Base(parcial), ".mkcorte-") || !strings.HasSuffix(parcial, ".parcial") {
		t.Errorf("parcial: %q", parcial)
	}
	if !strings.Contains(args[21], "[1:0]trim=duration=2.000") || !strings.Contains(args[21], "anullsrc") || !strings.Contains(args[21], "pad=1920:1080:656:0:color=black") {
		t.Errorf("grafo: %q", args[21])
	}
	// Cada vídeo foi sondado uma vez, também pelo atalho.
	if p := m.Chamadas("ffprobe"); len(p) != 2 || filepath.Base(p[0][7]) != "fonte0.mp4" || filepath.Base(p[1][7]) != "fonte1.mov" {
		t.Errorf("ffprobe: %q", p)
	}

	// As fases: o plano, o progresso (com o tempo que resta) e nada mais.
	if len(fases) != 4 || fases[0]["fase"] != "plano" || fases[0]["saida"] != querSaida || fases[0]["modo"] != "exato" || fases[0]["duracao_s"] != 4.0 {
		t.Fatalf("fases: %v", fases)
	}
	for _, k := range []string{"clipes", "largura", "altura", "fps", "tem_audio", "tamanho_estimado_bytes", "sem_recodificar", "comando", "avisos"} {
		if _, tem := fases[0][k]; !tem {
			t.Errorf("o plano não tem %q", k)
		}
	}
	if fases[1]["pct"] != 25.0 || fases[1]["velocidade"] != nil || fases[1]["resta_s"] != nil || fases[2]["pct"] != 50.0 || fases[2]["velocidade"] != 2.5 ||
		fases[2]["resta_s"] != 1.0 || fases[3]["pct"] != 99.9 || fases[3]["tamanho_bytes"] != 4000.0 || fases[3]["fase"] != "exportando" {
		t.Errorf("progresso: %v", fases[1:])
	}
}

func TestExporta_NuncaSobrescreveNemOOriginalNemUmaSaidaAnterior(t *testing.T) {
	monta(t, "novo")
	pasta := t.TempDir()
	a := original(t, pasta, "a.mp4", 100)
	velha := original(t, pasta, "a (corte).mp4", 7)
	um := `{"fontes":[` + string(must(json.Marshal(a))) + `],"clipes":[{"fonte":0,"inicio":0,"fim":4}]}`
	fim, fases, err := exporta(t, edicao(t, um), "")
	if err != nil || fim.Saida != filepath.Join(pasta, "a (corte) (2).mp4") || fases[0]["saida"] != fim.Saida {
		t.Fatalf("segunda: %v %+v", err, fim)
	}
	if fi, _ := os.Stat(velha); fi.Size() != 7 {
		t.Error("a saída anterior foi sobrescrita")
	}
	// Pedir a saída em cima do original (ou com outra extensão): ganha outro nome.
	fim, _, err = exporta(t, edicao(t, um), a)
	if err != nil || fim.Saida != filepath.Join(pasta, "a (2).mp4") {
		t.Fatalf("em cima do original: %v %+v", err, fim)
	}
	if fi, _ := os.Stat(a); fi.Size() != 100 {
		t.Error("o original foi sobrescrito")
	}
	outra := t.TempDir()
	fim, _, err = exporta(t, edicao(t, um), filepath.Join(outra, "final.mkv"))
	if err != nil || fim.Saida != filepath.Join(outra, "final.mp4") {
		t.Fatalf("outra pasta: %v %+v", err, fim)
	}
	if _, _, err = exporta(t, edicao(t, um), filepath.Join(outra, "nao-existe", "x.mp4")); err == nil || !strings.Contains(err.Error(), "a pasta da saída não existe") {
		t.Errorf("pasta ausente: %v", err)
	}
	if nomes := nomesDe(t, pasta); len(nomes) != 4 {
		t.Errorf("pasta: %q", nomes)
	}
}

func must(b []byte, err error) []byte {
	if err != nil {
		panic(err)
	}
	return b
}

func TestExporta_RapidoEscreveAListaNaPastaDosAtalhos(t *testing.T) {
	m := monta(t, "cópia")
	a := original(t, "", "de 'aspas'.mp4", 48000)
	texto, _ := json.Marshal(Edicao{Fontes: []string{a}, Clipes: []Clipe{{0, 10, 20}, {0, 0, 5}}, Modo: "rapido"})
	fim, fases, err := exporta(t, edicao(t, string(texto)), "")
	if err != nil || fim.Modo != "rapido" || fim.Duracao != 15 {
		t.Fatalf("rápido: %v %+v", err, fim)
	}
	vista, _ := os.ReadFile(filepath.Join(m.Dir, "lista-vista"))
	if quer := "ffconcat version 1.0\nfile 'fonte0.mp4'\ninpoint 10.000\noutpoint 20.000\nfile 'fonte0.mp4'\noutpoint 5.000\n"; string(vista) != quer {
		t.Errorf("a lista que o ffmpeg leu:\n%s", vista)
	}
	args := m.Chamadas("ffmpeg")[0]
	if args[8] != "-f" || args[9] != "concat" || filepath.Base(args[13]) != "lista.ffconcat" || !filepath.IsAbs(args[13]) || args[18] != "-codec" || args[19] != "copy" {
		t.Errorf("args: %q", args)
	}
	if fases[0]["modo"] != "rapido" || len(fases[0]["avisos"].([]any)) != 1 {
		t.Errorf("plano: %v", fases[0])
	}
	// Pedir o rápido quando não dá é recusado antes de o ffmpeg rodar.
	m.Esquece("ffmpeg")
	texto, _ = json.Marshal(Edicao{Fontes: []string{a}, Clipes: []Clipe{{0, 10, 20}}, Modo: "rapido", Qualidade: "720"})
	if _, _, err := exporta(t, edicao(t, string(texto)), ""); !base.FoiRecusa(err) {
		t.Errorf("devia recusar: %v", err)
	}
	if len(m.Chamadas("ffmpeg")) != 0 || len(nomesDe(t, filepath.Dir(a))) != 2 {
		t.Error("o ffmpeg rodou ou sobrou arquivo")
	}
}

func TestExporta_FalhaDoFfmpegViraFraseEApagaOParcial(t *testing.T) {
	m := monta(t, "")
	a := original(t, "", "a.mp4", 100)
	um := edicao(t, `{"fontes":[`+string(must(json.Marshal(a)))+`],"clipes":[{"fonte":0,"inicio":0,"fim":4}]}`)
	m.Comando("ffmpeg", `for a in "$@"; do ultimo=$a; done; printf 'meio' >"$ultimo"; printf 'aviso\nConversion failed!\n' >&2; exit 1`)
	if _, _, err := exporta(t, um, ""); err == nil || err.Error() != "o ffmpeg não conseguiu: Conversion failed!" {
		t.Errorf("falha: %v", err)
	}
	m.Comando("ffmpeg", `exit 0`)
	if _, _, err := exporta(t, um, ""); err == nil || !strings.Contains(err.Error(), "sem escrever a saída") {
		t.Errorf("saída vazia: %v", err)
	}
	if nomes := nomesDe(t, filepath.Dir(a)); len(nomes) != 1 {
		t.Errorf("sobrou: %q", nomes)
	}
	os.Remove(filepath.Join(m.Dir, "ffmpeg"))
	if _, _, err := exporta(t, um, ""); err == nil || !strings.Contains(err.Error(), "não achei o ffmpeg") {
		t.Errorf("sem ffmpeg: %v", err)
	}
	// Um vídeo da edição que sumiu.
	os.Remove(a)
	if _, _, err := exporta(t, um, ""); err == nil || !strings.Contains(err.Error(), "não achei o arquivo") {
		t.Errorf("vídeo ausente: %v", err)
	}
}

func TestExporta_CancelarMataOFfmpegEApagaASaidaPelaMetade(t *testing.T) {
	m := monta(t, "")
	a := original(t, "", "a.mp4", 48000)
	um := edicao(t, `{"fontes":[`+string(must(json.Marshal(a)))+`],"clipes":[{"fonte":0,"inicio":0,"fim":4}]}`)
	m.Comando("ffmpeg", `for a in "$@"; do ultimo=$a; done
printf 'pela metade' >"$ultimo"
printf 'out_time_us=1000000\nspeed=1x\nprogress=continue\n'
exec /bin/sleep 30`)

	// Cancelado de fora (o SIGTERM do motor fecha este canal).
	cancela := make(chan struct{})
	viu := make(chan struct{}, 8)
	feito := make(chan error, 1)
	go func() {
		_, err := Exporta(um, "", func(v any) error {
			if _, ok := v.(*FaseExportando); ok {
				viu <- struct{}{}
			}
			return nil
		}, cancela)
		feito <- err
	}()
	select {
	case <-viu:
	case <-time.After(10 * time.Second):
		t.Fatal("o progresso não chegou")
	}
	comeco := time.Now()
	close(cancela)
	select {
	case err := <-feito:
		if !errors.Is(err, ErrCancelado) {
			t.Errorf("devia ser ErrCancelado: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("cancelar não parou o ffmpeg")
	}
	if time.Since(comeco) > 5*time.Second {
		t.Errorf("cancelar demorou %v", time.Since(comeco))
	}
	if nomes := nomesDe(t, filepath.Dir(a)); len(nomes) != 1 {
		t.Errorf("a saída pela metade ficou: %v", nomes)
	}

	// Ninguém mais lê o que o motor escreve (a janela fechou): para também.
	_, err := Exporta(um, "", func(v any) error {
		if _, ok := v.(*FaseExportando); ok {
			return errors.New("cano fechado")
		}
		return nil
	}, nil)
	if !errors.Is(err, ErrCancelado) {
		t.Errorf("stdout fechado: %v", err)
	}
	if nomes := nomesDe(t, filepath.Dir(a)); len(nomes) != 1 {
		t.Errorf("a saída pela metade ficou: %v", nomes)
	}
}

func TestPlanejar_NaoRodaOFfmpegNemEscreveNada(t *testing.T) {
	m := monta(t, "x")
	pasta := t.TempDir()
	a := original(t, pasta, "a.mp4", 48000)
	original(t, pasta, "a (corte).mp4", 1)
	p, err := Planejar(edicao(t, `{"fontes":[`+string(must(json.Marshal(a)))+`],"clipes":[{"fonte":0,"inicio":0,"fim":4}]}`), "")
	if err != nil {
		t.Fatal(err)
	}
	if p.Saida != filepath.Join(pasta, "a (corte) (2).mp4") || p.Comando[len(p.Comando)-1] != p.Saida || p.Duracao != 4 {
		t.Errorf("plano: %+v", p)
	}
	if len(m.Chamadas("ffmpeg")) != 0 || len(nomesDe(t, pasta)) != 2 {
		t.Error("o plano rodou o ffmpeg ou escreveu na pasta")
	}
}

func TestProgresso_LeAsLinhasDoFfmpeg(t *testing.T) {
	p := &progresso{duracao: 100}
	for _, l := range []string{"frame=3", "lixo sem igual", "", "out_time_us=N/A", "speed=N/A", "total_size=N/A", "out_time_us=-5"} {
		if f := p.linha(l); f != nil {
			t.Errorf("%q não fecha um bloco", l)
		}
	}
	if f := p.linha("progress=continue"); f == nil || f.Pct != 0 || f.Velocidade != nil || f.Resta != nil || f.Tamanho != 0 {
		t.Errorf("bloco vazio: %+v", f)
	}
	p.linha("out_time_ms=40000000\n")
	p.linha("speed=0.5x")
	p.linha("total_size=123")
	f := p.linha("progress=continue\n")
	if f == nil || f.Fase != "exportando" || f.Pct != 40 || *f.Velocidade != 0.5 || *f.Resta != 120 || f.Tamanho != 123 {
		t.Errorf("bloco: %+v", f)
	}
	// Passou da duração (no modo rápido acontece): não passa de 99,9 antes do fim.
	p.linha("out_time_us=500000000")
	if f := p.linha("progress=end"); f.Pct != 99.9 || *f.Resta != 0 {
		t.Errorf("além do fim: %+v", f)
	}
	// Sem duração conhecida: 0%, sem previsão.
	q := &progresso{}
	q.linha("out_time_us=5000000")
	q.linha("speed=2x")
	if f := q.linha("progress=continue"); f.Pct != 0 || f.Resta != nil {
		t.Errorf("sem duração: %+v", f)
	}
}

// --- miniaturas ------------------------------------------------------------

func TestMiniaturas_GeraATiraNoCacheEDepoisSoLe(t *testing.T) {
	m := monta(t, "jpeg")
	emParalelo = 1
	a := original(t, "", nomeRuim, 48000)
	tira, err := Miniaturas(a, 4, 0)
	if err != nil {
		t.Fatal(err)
	}
	cache := filepath.Join(os.Getenv("XDG_CACHE_HOME"), "mk", "mkcorte")
	if tira.Arquivo != a || tira.Duracao != 62.4 || tira.Altura != 54 || tira.DoCache || filepath.Dir(tira.Pasta) != cache || len(tira.Quadros) != 4 {
		t.Fatalf("tira: %+v", tira)
	}
	for i, quer := range []float64{7.8, 23.4, 39, 54.6} {
		q := tira.Quadros[i]
		if q.Tempo != quer || filepath.Dir(q.Arquivo) != tira.Pasta || !strings.HasSuffix(q.Arquivo, ".jpg") {
			t.Errorf("quadro %d: %+v", i, q)
		}
		if b, _ := os.ReadFile(q.Arquivo); string(b) != "jpeg" {
			t.Errorf("quadro %d não foi escrito", i)
		}
	}
	c := m.Chamadas("ffmpeg")
	if len(c) != 4 {
		t.Fatalf("ffmpeg: %d chamadas", len(c))
	}
	quer := []string{"-hide_banner", "-nostdin", "-loglevel", "error", "-y", "-ss", "23.400", "-i", "?", "-map", "0:1", "-frames:v", "1", "-vf", "scale=-2:54",
		"-c:v", "mjpeg", "-q:v", "5", "-f", "image2", "-update", "1", "?"}
	if len(c[1]) != len(quer) || filepath.Base(c[1][8]) != "fonte0.mp4" || !strings.HasSuffix(c[1][23], "/q02.jpg") {
		t.Fatalf("ffmpeg: %q", c[1])
	}
	for i, x := range quer {
		if x != "?" && c[1][i] != x {
			t.Errorf("argumento %d: %q, quer %q", i, c[1][i], x)
		}
	}
	// Nada pela metade ficou no cache: só a pasta da tira.
	if nomes := nomesDe(t, cache); len(nomes) != 1 || nomes[0] != filepath.Base(tira.Pasta) {
		t.Errorf("cache: %q", nomes)
	}

	// De novo: vem do cache, sem ffmpeg nem ffprobe.
	m.Esquece("ffmpeg")
	m.Esquece("ffprobe")
	de, err := Miniaturas(a, 4, 54)
	if err != nil || !de.DoCache || !reflect.DeepEqual(de.Quadros, tira.Quadros) || de.Pasta != tira.Pasta {
		t.Errorf("do cache: %v %+v", err, de)
	}
	if len(m.Chamadas("ffmpeg")) != 0 || len(m.Chamadas("ffprobe")) != 0 {
		t.Error("com a tira no cache, não era para rodar nada")
	}

	// Outro pedido (mais quadros, outra altura) é outra tira.
	outra, err := Miniaturas(a, 2, 91)
	if err != nil || outra.Pasta == tira.Pasta || outra.Altura != 90 || len(outra.Quadros) != 2 || outra.DoCache {
		t.Errorf("outro pedido: %v %+v", err, outra)
	}
	// O arquivo mudou (a data de modificação): a chave muda e a tira é refeita.
	m.Esquece("ffmpeg")
	depois := time.Now().Add(time.Hour)
	if err := os.Chtimes(a, depois, depois); err != nil {
		t.Fatal(err)
	}
	nova, err := Miniaturas(a, 4, 54)
	if err != nil || nova.Pasta == tira.Pasta || nova.DoCache || len(m.Chamadas("ffmpeg")) != 4 {
		t.Errorf("arquivo mudado: %v %+v", err, nova)
	}
	// Faltou um quadro numa tira guardada: não serve, é refeita.
	m.Esquece("ffmpeg")
	os.Remove(nova.Quadros[2].Arquivo)
	refeita, err := Miniaturas(a, 4, 54)
	if err != nil || refeita.DoCache || len(refeita.Quadros) != 4 || len(m.Chamadas("ffmpeg")) != 4 {
		t.Errorf("tira incompleta: %v %+v", err, refeita)
	}
}

func TestMiniaturas_ChaveLimitesEQuadrosQueFalham(t *testing.T) {
	m := monta(t, "jpeg")
	emParalelo = 1
	agora := time.Unix(1700000000, 5)
	k := chaveDaTira("/v/a.mp4", 10, agora, 12, 54)
	if len(k) != 32 || k != chaveDaTira("/v/a.mp4", 10, agora, 12, 54) {
		t.Errorf("chave: %q", k)
	}
	for nome, outra := range map[string]string{"caminho": chaveDaTira("/v/b.mp4", 10, agora, 12, 54), "tamanho": chaveDaTira("/v/a.mp4", 11, agora, 12, 54),
		"data": chaveDaTira("/v/a.mp4", 10, agora.Add(time.Nanosecond), 12, 54), "quantas": chaveDaTira("/v/a.mp4", 10, agora, 13, 54),
		"altura": chaveDaTira("/v/a.mp4", 10, agora, 12, 56)} {
		if outra == k {
			t.Errorf("a chave não muda com: %s", nome)
		}
	}
	a := original(t, "", "a.mp4", 100)
	if _, err := Miniaturas(a, 61, 0); err == nil || !strings.Contains(err.Error(), "miniaturas demais") {
		t.Errorf("demais: %v", err)
	}
	if _, err := Miniaturas(a, 4, 4000); err == nil || !strings.Contains(err.Error(), "altura") {
		t.Errorf("altura: %v", err)
	}
	if _, err := Miniaturas(filepath.Join(filepath.Dir(a), "nao.mp4"), 4, 0); err == nil || !strings.Contains(err.Error(), "não achei") {
		t.Errorf("ausente: %v", err)
	}
	// Um quadro que o ffmpeg não consegue tirar (o do fim) fica de fora; os outros valem.
	m.Comando("ffmpeg", `for a in "$@"; do ultimo=$a; done
case "$ultimo" in */q03.jpg) echo "nada" >&2; exit 1 ;; */q02.jpg) exit 0 ;; esac
printf 'jpeg' >"$ultimo"`)
	tira, err := Miniaturas(a, 3, 0)
	if err != nil || len(tira.Quadros) != 1 || !strings.HasSuffix(tira.Quadros[0].Arquivo, "q01.jpg") || tira.Quadros[0].Tempo != 10.4 {
		t.Errorf("com falhas: %v %+v", err, tira)
	}
	// Nenhum quadro: erro, e nada fica no cache.
	m.Comando("ffmpeg", `exit 1`)
	cache := base.Cache()
	antes := len(nomesDe(t, cache))
	if _, err := Miniaturas(a, 5, 0); err == nil || !strings.Contains(err.Error(), "nenhum quadro") {
		t.Errorf("nenhum: %v", err)
	}
	if len(nomesDe(t, cache)) != antes {
		t.Errorf("o cache ficou com restos: %q", nomesDe(t, cache))
	}
	os.Remove(filepath.Join(m.Dir, "ffmpeg"))
	if _, err := Miniaturas(a, 6, 0); err == nil || !strings.Contains(err.Error(), "não achei o ffmpeg") {
		t.Errorf("sem ffmpeg: %v", err)
	}
}

func TestMiniaturas_ACacheVelhaSaiEAFichaNaoMandaLerForaDaPasta(t *testing.T) {
	monta(t, "jpeg")
	emParalelo = 1
	cache := base.Cache()
	velha, nova := filepath.Join(cache, "velha"), filepath.Join(cache, "nova")
	for _, p := range []string{velha, nova} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	faz40dias := time.Now().Add(-40 * 24 * time.Hour)
	if err := os.Chtimes(velha, faz40dias, faz40dias); err != nil {
		t.Fatal(err)
	}
	solto := original(t, cache, "arquivo-solto", 1)
	_ = os.Chtimes(solto, faz40dias, faz40dias)
	a := original(t, "", "a.mp4", 100)
	tira, err := Miniaturas(a, 2, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(velha); err == nil {
		t.Error("a tira de 40 dias ficou")
	}
	if _, err := os.Stat(nova); err != nil {
		t.Error("a tira recente saiu")
	}
	if _, err := os.Stat(solto); err != nil {
		t.Error("um arquivo que não é tira foi apagado")
	}
	// Uma ficha adulterada apontando para fora: o caminho é refeito dentro da pasta.
	segredo := original(t, "", "segredo.jpg", 10)
	ficha, _ := json.Marshal(Tira{Arquivo: a, Duracao: 62.4, Altura: 54, Quadros: []Mini{{Tempo: 1, Arquivo: segredo}}})
	if err := os.WriteFile(filepath.Join(tira.Pasta, nomeDaTira), ficha, 0o644); err != nil {
		t.Fatal(err)
	}
	if pronta := tiraPronta(tira.Pasta, a); pronta != nil {
		t.Errorf("a ficha adulterada valeu: %+v", pronta)
	}
	// E a ficha de outro arquivo não serve para este.
	if pronta := tiraPronta(tira.Pasta, "/outro.mp4"); pronta != nil {
		t.Error("a ficha de outro arquivo valeu")
	}
}

// --- quadro ----------------------------------------------------------------

func TestQuadro_SalvaOPNGAoLadoDoVideoSemSobrescrever(t *testing.T) {
	m := monta(t, "png")
	pasta := t.TempDir()
	a := original(t, pasta, nomeRuim, 48000)
	foto, err := Quadro(a, 83.4504, "")
	if err == nil {
		t.Fatalf("além do fim (62,4 s) devia falhar: %+v", foto)
	}
	if !strings.Contains(err.Error(), "o vídeo acaba em 62.400 s") || len(m.Chamadas("ffmpeg")) != 0 {
		t.Errorf("além do fim: %v", err)
	}
	foto, err = Quadro(a, 23.4504, "")
	if err != nil {
		t.Fatal(err)
	}
	quer := filepath.Join(pasta, "-i %03d *concat: 'aspas\" $(rm) `x` férias (quadro 0-00-23.450).png")
	if !foto.Ok || foto.Saida != quer || foto.Tempo != 23.45 || foto.Tamanho != 3 {
		t.Errorf("foto: %+v", foto)
	}
	c := m.Chamadas("ffmpeg")[0]
	querArgs := []string{"-hide_banner", "-nostdin", "-loglevel", "error", "-y", "-ss", "23.450", "-i", "?", "-map", "0:1", "-frames:v", "1",
		"-c:v", "png", "-f", "image2", "-update", "1", "?"}
	if len(c) != len(querArgs) || filepath.Base(c[8]) != "fonte0.mp4" || filepath.Dir(c[19]) != pasta || !strings.HasSuffix(c[19], ".parcial") {
		t.Fatalf("ffmpeg: %q", c)
	}
	for i, x := range querArgs {
		if x != "?" && c[i] != x {
			t.Errorf("argumento %d: %q, quer %q", i, c[i], x)
		}
	}
	// O mesmo quadro de novo: outro nome.
	foto, err = Quadro(a, 23.45, "")
	if err != nil || foto.Saida != strings.TrimSuffix(quer, ".png")+" (2).png" {
		t.Errorf("de novo: %v %+v", err, foto)
	}
	// Com a saída escolhida: vira .png e também não sobrescreve.
	escolhida := filepath.Join(pasta, "capa.jpg")
	foto, err = Quadro(a, 0, escolhida)
	if err != nil || foto.Saida != filepath.Join(pasta, "capa.png") {
		t.Errorf("escolhida: %v %+v", err, foto)
	}
	foto, err = Quadro(a, 0, a)
	if err != nil || foto.Saida == a || !strings.HasSuffix(foto.Saida, ".png") {
		t.Errorf("em cima do vídeo: %v %+v", err, foto)
	}
	if fi, _ := os.Stat(a); fi.Size() != 48000 {
		t.Error("o vídeo foi sobrescrito")
	}
	if nomes := nomesDe(t, pasta); len(nomes) != 5 {
		t.Errorf("pasta: %q", nomes)
	}

	for _, tempo := range []float64{-1, 62.4} {
		if _, err := Quadro(a, tempo, ""); err == nil {
			t.Errorf("tempo %v devia falhar", tempo)
		}
	}
	m.Comando("ffmpeg", `exit 0`)
	if _, err := Quadro(a, 1, ""); err == nil || !strings.Contains(err.Error(), "não há quadro") {
		t.Errorf("sem quadro: %v", err)
	}
	m.Comando("ffmpeg", `echo "quebrou" >&2; exit 1`)
	if _, err := Quadro(a, 1, ""); err == nil || !strings.Contains(err.Error(), "quebrou") {
		t.Errorf("ffmpeg com erro: %v", err)
	}
	if nomes := nomesDe(t, pasta); len(nomes) != 5 {
		t.Errorf("sobrou parcial: %q", nomes)
	}
}
