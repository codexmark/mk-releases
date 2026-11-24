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

	"mkutil/internal/base"
	"mkutil/internal/dubles"
)

const probeVideo = `{"streams":[
 {"index":0,"codec_name":"mjpeg","codec_type":"video","width":600,"height":600,"disposition":{"attached_pic":1}},
 {"index":1,"codec_name":"h264","codec_type":"video","width":1920,"height":1080,"r_frame_rate":"30/1","avg_frame_rate":"30000/1001",
  "duration":"62.400000","bit_rate":"6000000","nb_frames":"1870","disposition":{"attached_pic":0}},
 {"index":2,"codec_name":"aac","codec_type":"audio","channels":2,"bit_rate":"128000"},
 {"index":3,"codec_name":"ac3","codec_type":"audio","channels":6,"bit_rate":"448000"}],
 "format":{"format_name":"mov,mp4,m4a,3gp,3g2,mj2","duration":"62.400000","size":"48000000","bit_rate":"6153846"}}`

const probeCelular = `{"streams":[{"codec_name":"hevc","codec_type":"video","width":1920,"height":1080,"avg_frame_rate":"60/1",
  "side_data_list":[{"side_data_type":"Display Matrix","rotation":-90}]}],"format":{"format_name":"mov,mp4","duration":"10.0","bit_rate":"8000000"}}`

const probeFoto = `{"streams":[{"codec_name":"mjpeg","codec_type":"video","width":4001,"height":3001,"avg_frame_rate":"25/1","duration":"0.04","nb_frames":"1"}],
 "format":{"format_name":"image2","duration":"0.040000","bit_rate":"900000000"}}`

const probeGif = `{"streams":[{"codec_name":"gif","codec_type":"video","width":480,"height":270,"avg_frame_rate":"0/0","r_frame_rate":"12/1","nb_frames":"36"}],
 "format":{"format_name":"gif","duration":"3.0"}}`

const codificadoresDoFfmpeg = `Encoders:
 V..... = Video
 A..... = Audio
 ------
 V....D libaom-av1           libaom AV1 (codec av1)
 V..... libsvtav1            SVT-AV1 encoder (codec av1)
 V....D av1_vaapi            AV1 (VAAPI) (codec av1)
 V....D gif                  GIF (Graphics Interchange Format)
 V....D libx264              libx264 H.264 (codec h264)
 V....D h264_nvenc           NVIDIA NVENC H.264 encoder (codec h264)
 V....D h264_vaapi           H.264/AVC (VAAPI) (codec h264)
 V....D libx265              libx265 H.265 / HEVC (codec hevc)
 V....D hevc_vaapi           H.265/HEVC (VAAPI) (codec hevc)
 VFS..D mjpeg                MJPEG (Motion JPEG)
 V....D libwebp_anim         libwebp WebP image (codec webp)
 V....D libwebp              libwebp WebP image (codec webp)
 A....D aac                  AAC (Advanced Audio Coding)
`

func f(v float64) *float64 { return &v }

func TestInterpreta_VideoComCapaEDuasTrilhasDeAudio(t *testing.T) {
	m, err := interpreta([]byte(probeVideo))
	if err != nil {
		t.Fatal(err)
	}
	if m.Tipo != "video" || m.Largura != 1920 || m.Altura != 1080 || m.Codec != "h264" || *m.Duracao != 62.4 || *m.FPS != 29.97 || *m.Taxa != 6000000 {
		t.Errorf("veio %+v", m)
	}
	if m.Audio == nil || m.Audio.Codec != "aac" || *m.Audio.Taxa != 128000 || m.Audio.Canais != 2 {
		t.Errorf("áudio: %+v", m.Audio)
	}
}

func TestInterpreta_CelularEmPeFotoGifEOQueNaoEImagem(t *testing.T) {
	m, err := interpreta([]byte(probeCelular))
	if err != nil || m.Largura != 1080 || m.Altura != 1920 || m.Audio != nil || *m.Taxa != 8000000 || *m.FPS != 60 {
		t.Errorf("celular em pé: %+v, %v", m, err)
	}
	m, err = interpreta([]byte(probeFoto))
	if err != nil || m.Tipo != "imagem" || m.Largura != 4001 || m.Duracao != nil || m.FPS != nil || m.Taxa != nil || m.Audio != nil {
		t.Errorf("foto: %+v, %v", m, err)
	}
	m, err = interpreta([]byte(probeGif))
	if err != nil || m.Tipo != "video" || *m.Duracao != 3 || *m.FPS != 12 || m.Taxa != nil {
		t.Errorf("gif: %+v, %v", m, err)
	}
	for _, pipe := range []string{"png_pipe", "webp_pipe", "jpeg_pipe"} {
		m, err = interpreta([]byte(`{"streams":[{"codec_type":"video","codec_name":"png","width":10,"height":20}],"format":{"format_name":"` + pipe + `"}}`))
		if err != nil || m.Tipo != "imagem" {
			t.Errorf("%s: %+v, %v", pipe, m, err)
		}
	}
	// Uma foto HEIF/AVIF mora num contêiner de vídeo, com um quadro só.
	m, _ = interpreta([]byte(`{"streams":[{"codec_type":"video","codec_name":"av1","width":10,"height":20,"nb_frames":"1"}],"format":{"format_name":"mov,mp4","duration":"0.04"}}`))
	if m == nil || m.Tipo != "imagem" {
		t.Errorf("avif: %+v", m)
	}
	erros := map[string]string{
		`{"streams":[{"codec_type":"audio","codec_name":"mp3"}],"format":{"format_name":"mp3","duration":"200"}}`:                                                            "só tem áudio",
		`{"streams":[{"codec_type":"video","codec_name":"mjpeg","width":5,"height":5,"disposition":{"attached_pic":1}},{"codec_type":"audio"}],"format":{"duration":"200"}}`: "só tem áudio",
		`{"streams":[{"codec_type":"subtitle"}],"format":{}}`:                                                                                                                "não tem imagem",
		`{"streams":[],"format":{}}`: "não tem imagem",
		`{}`:                         "não tem imagem",
		`isto não é json`:            "não entendi",
	}
	for entrada, trecho := range erros {
		if _, err := interpreta([]byte(entrada)); err == nil || !strings.Contains(err.Error(), trecho) {
			t.Errorf("%s: %v", entrada[:20], err)
		}
	}
}

// monta põe os dublês do ffprobe e do ffmpeg. O ffmpeg de mentira escreve
// `saida` no último argumento (a não ser na primeira de duas passadas, que
// vai para /dev/null) depois de soltar dois blocos de progresso.
func monta(t *testing.T, probe, saida string) *dubles.Mesa {
	t.Helper()
	t.Setenv("MKUTIL_HOME", t.TempDir())
	t.Setenv("TMPDIR", t.TempDir())
	m := dubles.Nova(t)
	m.Responde("ffprobe", probe, 0)
	m.Comando("ffmpeg", `case " $* " in *" -encoders "*) printf '%s' `+dubles.Aspas(codificadoresDoFfmpeg)+`; exit 0 ;; esac
for a in "$@"; do ultimo=$a; done
printf 'frame=10\nout_time_us=15600000\nout_time_ms=15600000\nout_time=00:00:15.6\nspeed=N/A\ntotal_size=1000\nprogress=continue\n'
printf 'out_time_us=31200000\nspeed= 2.5x \ntotal_size=2000\nprogress=continue\n'
printf 'out_time_us=62400000\nspeed=3.12x\ntotal_size=4000\nprogress=end\n'
[ "$ultimo" = /dev/null ] || printf '%s' `+dubles.Aspas(saida)+` >"$ultimo"`)
	return m
}

func original(t *testing.T, nome string, bytes int) string {
	t.Helper()
	caminho := filepath.Join(t.TempDir(), nome)
	if err := os.WriteFile(caminho, make([]byte, bytes), 0o644); err != nil {
		t.Fatal(err)
	}
	return caminho
}

func comprime(t *testing.T, arquivo string, p Pedido) (*Fim, []string, error) {
	t.Helper()
	var fases []string
	fim, err := Comprime(arquivo, p, func(v any) error {
		b, _ := json.Marshal(v)
		fases = append(fases, string(b))
		return nil
	}, nil)
	return fim, fases, err
}

// osDoFfmpeg devolve as chamadas que codificam (sem a de -encoders).
func osDoFfmpeg(m *dubles.Mesa) [][]string {
	var out [][]string
	for _, c := range m.Chamadas("ffmpeg") {
		if len(c) > 0 && c[len(c)-1] != "-encoders" {
			out = append(out, c)
		}
	}
	return out
}

func TestSonda_OFfprobeRecebeUmAtalhoNuncaONomeDoArquivo(t *testing.T) {
	m := monta(t, probeVideo, "")
	arq := original(t, "-i %03d *concat: 'aspas\" $(rm) férias.MP4", 1000)
	midia, err := Sonda(arq)
	if err != nil {
		t.Fatal(err)
	}
	if midia.Arquivo != arq || midia.Tamanho != 1000 || midia.Tipo != "video" {
		t.Errorf("veio %+v", midia)
	}
	c := m.Chamadas("ffprobe")
	if len(c) != 1 || len(c[0]) != 8 {
		t.Fatalf("ffprobe: %q", c)
	}
	if quer := []string{"-v", "error", "-of", "json", "-show_format", "-show_streams", "-i"}; !reflect.DeepEqual(c[0][:7], quer) {
		t.Errorf("ffprobe: %q", c[0])
	}
	link := c[0][7]
	if filepath.Base(link) != "entrada.mp4" || !filepath.IsAbs(link) || strings.Contains(strings.Join(c[0], " "), "férias") {
		t.Errorf("o ffprobe viu %q", c[0])
	}
	if _, err := os.Lstat(filepath.Dir(link)); err == nil {
		t.Error("a pasta temporária ficou para trás")
	}
}

func TestSonda_ErrosViramFrase(t *testing.T) {
	m := monta(t, probeVideo, "")
	arq := original(t, "a.mp4", 10)
	if _, err := Sonda(filepath.Join(filepath.Dir(arq), "nao-existe.mp4")); err == nil || !strings.Contains(err.Error(), "não achei") {
		t.Errorf("ausente: %v", err)
	}
	if _, err := Sonda(filepath.Dir(arq)); err == nil || !strings.Contains(err.Error(), "não é um arquivo") {
		t.Errorf("pasta: %v", err)
	}
	m.Comando("ffprobe", `printf '%s\n' "$8: Invalid data found when processing input" >&2; exit 1`)
	if _, err := Sonda(arq); err == nil || !strings.Contains(err.Error(), "não parece vídeo nem imagem") {
		t.Errorf("ffprobe com erro: %v", err)
	}
	os.Remove(filepath.Join(m.Dir, "ffprobe"))
	if _, err := Sonda(arq); err == nil || !strings.Contains(err.Error(), "pacote ffmpeg") {
		t.Errorf("sem ffprobe: %v", err)
	}
}

func TestSonda_SemTaxaNoArquivoFazAContaPeloTamanho(t *testing.T) {
	monta(t, probeGif, "")
	m, err := Sonda(original(t, "a.gif", 3_000_000))
	if err != nil || m.Taxa == nil || *m.Taxa != 8_000_000 {
		t.Errorf("veio %+v, %v", m, err)
	}
}

func TestCodificadores_LeAListaDoFfmpegEResumeOQueOferece(t *testing.T) {
	m := monta(t, probeVideo, "")
	tem, err := Codificadores()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []string{"libx264", "libx265", "libsvtav1", "h264_vaapi", "h264_nvenc", "mjpeg", "libwebp", "gif", "aac"} {
		if !tem[c] {
			t.Errorf("faltou %s", c)
		}
	}
	if tem["Video"] || tem["="] || tem["hevc_nvenc"] {
		t.Errorf("leu o que não é codificador: %v", tem)
	}
	quer := Capacidades{Ffmpeg: true, Codecs: []string{"h264", "h265", "av1"}, Aceleracao: []string{"vaapi", "nvenc"},
		Imagens: []string{"jpeg", "webp", "avif"}, Animados: []string{"gif", "webp"}}
	if got := Resume(tem); !reflect.DeepEqual(got, quer) {
		t.Errorf("resumo: %+v", got)
	}
	if got := Resume(map[string]bool{"libx264": true}); !reflect.DeepEqual(got.Codecs, []string{"h264"}) || len(got.Aceleracao) != 0 || got.Imagens == nil {
		t.Errorf("só x264: %+v", got)
	}
	if got := Resume(nil); got.Ffmpeg || got.Codecs == nil {
		t.Errorf("sem ffmpeg: %+v", got)
	}
	os.Remove(filepath.Join(m.Dir, "ffmpeg"))
	if _, err := Codificadores(); err == nil || !strings.Contains(err.Error(), "não achei o ffmpeg") {
		t.Errorf("sem ffmpeg: %v", err)
	}
}

// Os mesmos casos que o teste do Util.js (tests/apps/mkutil-Util.test.mjs) usa.
type vetores struct {
	Caber []struct {
		AlvoMB       float64  `json:"alvo_mb"`
		Duracao      float64  `json:"duracao_s"`
		Audio        bool     `json:"audio"`
		Largura      int      `json:"largura"`
		Altura       int      `json:"altura"`
		FPS          *float64 `json:"fps"`
		Cabe         bool     `json:"cabe"`
		VideoBps     int64    `json:"video_bps"`
		AudioBps     int64    `json:"audio_bps"`
		SaidaLargura int      `json:"saida_largura"`
		SaidaAltura  int      `json:"saida_altura"`
	} `json:"caber"`
	Dimensoes []struct {
		Largura      int `json:"largura"`
		Altura       int `json:"altura"`
		Lado         int `json:"lado"`
		SaidaLargura int `json:"saida_largura"`
		SaidaAltura  int `json:"saida_altura"`
	} `json:"dimensoes"`
}

func leVetores(t *testing.T) vetores {
	t.Helper()
	b, err := os.ReadFile("testdata/caber.json")
	if err != nil {
		t.Fatal(err)
	}
	var v vetores
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}
	if len(v.Caber) < 10 || len(v.Dimensoes) < 8 {
		t.Fatal("faltam casos em testdata/caber.json")
	}
	return v
}

func TestCabe_AContaDoContrato(t *testing.T) {
	for _, c := range leVetores(t).Caber {
		v, a, l, alt, err := Cabe(c.AlvoMB, c.Duracao, c.Audio, c.Largura, c.Altura, c.FPS)
		if !c.Cabe {
			if err == nil || !strings.Contains(err.Error(), "não cabe") {
				t.Errorf("%v MB em %v s devia não caber: %v", c.AlvoMB, c.Duracao, err)
			}
			continue
		}
		if err != nil || v != c.VideoBps || a != c.AudioBps || l != c.SaidaLargura || alt != c.SaidaAltura {
			t.Errorf("%v MB, %v s, %dx%d: veio %d/%d %dx%d (%v), queria %d/%d %dx%d", c.AlvoMB, c.Duracao, c.Largura, c.Altura,
				v, a, l, alt, err, c.VideoBps, c.AudioBps, c.SaidaLargura, c.SaidaAltura)
		}
	}
	for _, ruim := range [][2]float64{{0, 10}, {-1, 10}, {8, 0}} {
		if _, _, _, _, err := Cabe(ruim[0], ruim[1], true, 1920, 1080, nil); err == nil {
			t.Errorf("Cabe(%v) devia falhar", ruim)
		}
	}
}

func TestDimensoes_LimitaOLadoMenorSemAmpliarESemprePar(t *testing.T) {
	for _, c := range leVetores(t).Dimensoes {
		if l, a := Dimensoes(c.Largura, c.Altura, c.Lado); l != c.SaidaLargura || a != c.SaidaAltura {
			t.Errorf("Dimensoes(%d, %d, %d) = %dx%d, queria %dx%d", c.Largura, c.Altura, c.Lado, l, a, c.SaidaLargura, c.SaidaAltura)
		}
	}
	casos := [][5]int{{1920, 1080, 480, 480, 270}, {1920, 1080, 4000, 1920, 1080}, {1081, 1921, 480, 480, 852}, {1921, 1081, 0, 1920, 1080}, {1000, 3, 500, 500, 2}}
	for _, c := range casos {
		if l, a := porLargura(c[0], c[1], c[2]); l != c[3] || a != c[4] {
			t.Errorf("porLargura(%d, %d, %d) = %dx%d, queria %dx%d", c[0], c[1], c[2], l, a, c[3], c[4])
		}
	}
}

func video() *Midia {
	taxa, ataxa := int64(6000000), int64(128000)
	return &Midia{Arquivo: "/v/férias.mp4", Tipo: "video", Tamanho: 48000000, Duracao: f(62.4), Largura: 1920, Altura: 1080, FPS: f(30),
		Codec: "h264", Taxa: &taxa, Audio: &Audio{Codec: "aac", Taxa: &ataxa, Canais: 2}}
}

var temTudo = map[string]bool{"libx264": true, "libx265": true, "libsvtav1": true, "libaom-av1": true, "h264_vaapi": true, "hevc_vaapi": true,
	"h264_nvenc": true, "mjpeg": true, "libwebp": true, "libwebp_anim": true, "gif": true}

func TestPlaneja_CadaPredefinicaoDeVideo(t *testing.T) {
	inicio := "-hide_banner -nostdin -loglevel error -y -progress pipe:1 -nostats -i {entrada} -map 0:v:0 "
	casos := []struct {
		pedido       Pedido
		sufixo, ext  string
		largura, alt int
		args         []string
	}{
		{Pedido{Predefinicao: "menor"}, "-menor", ".mp4", 1280, 720, []string{inicio +
			"-vf scale=1280:720:flags=lanczos -c:v libx264 -preset medium -pix_fmt yuv420p -crf 30 -map 0:a:0 -c:a aac -b:a 64k -ac 2 -sn -dn -movflags +faststart {saida}"}},
		{Pedido{Predefinicao: "equilibrio"}, "-equilibrio", ".mp4", 1920, 1080, []string{inicio +
			"-c:v libx264 -preset medium -pix_fmt yuv420p -crf 24 -map 0:a:0 -c:a aac -b:a 128k -sn -dn -movflags +faststart {saida}"}},
		{Pedido{Predefinicao: "alta", Codec: "h265"}, "-alta", ".mp4", 1920, 1080, []string{inicio +
			"-c:v libx265 -preset medium -pix_fmt yuv420p -tag:v hvc1 -crf 23 -x265-params log-level=error -map 0:a:0 -c:a aac -b:a 160k -sn -dn -movflags +faststart {saida}"}},
		{Pedido{Predefinicao: "alta", Codec: "av1", SemAudio: true}, "-alta-sem-audio", ".mp4", 1920, 1080, []string{inicio +
			"-c:v libsvtav1 -preset 8 -pix_fmt yuv420p -crf 27 -an -sn -dn -movflags +faststart {saida}"}},
		{Pedido{Predefinicao: "resolucao", Lado: 480}, "-480p", ".mp4", 854, 480, []string{inicio +
			"-vf scale=854:480:flags=lanczos -c:v libx264 -preset medium -pix_fmt yuv420p -crf 22 -map 0:a:0 -c:a aac -b:a 128k -sn -dn -movflags +faststart {saida}"}},
		{Pedido{Predefinicao: "caber", AlvoMB: 8}, "-8mb", ".mp4", 854, 480, []string{
			inicio + "-vf scale=854:480:flags=lanczos -c:v libx264 -preset medium -pix_fmt yuv420p -b:v 898k -pass 1 -passlogfile {passada} -an -sn -dn -f null /dev/null",
			inicio + "-vf scale=854:480:flags=lanczos -c:v libx264 -preset medium -pix_fmt yuv420p -b:v 898k -pass 2 -passlogfile {passada} -map 0:a:0 -c:a aac -b:a 96k -ac 2 -sn -dn -movflags +faststart {saida}"}},
		{Pedido{Predefinicao: "caber", AlvoMB: 25.5, Codec: "h265"}, "-25.5mb", ".mp4", 1920, 1080, []string{
			inicio + "-c:v libx265 -preset medium -pix_fmt yuv420p -tag:v hvc1 -b:v 3075k -x265-params log-level=error:pass=1:stats={passada}.x265 -an -sn -dn -f null /dev/null",
			inicio + "-c:v libx265 -preset medium -pix_fmt yuv420p -tag:v hvc1 -b:v 3075k -x265-params log-level=error:pass=2:stats={passada}.x265 -map 0:a:0 -c:a aac -b:a 96k -ac 2 -sn -dn -movflags +faststart {saida}"}},
		{Pedido{Predefinicao: "caber", AlvoMB: 8, Codec: "av1"}, "-8mb", ".mp4", 854, 480, []string{
			inicio + "-vf scale=854:480:flags=lanczos -c:v libsvtav1 -preset 8 -pix_fmt yuv420p -b:v 898k -map 0:a:0 -c:a aac -b:a 96k -ac 2 -sn -dn -movflags +faststart {saida}"}},
		{Pedido{Predefinicao: "menor", Acelerar: "vaapi"}, "-menor", ".mp4", 1280, 720, []string{
			"-hide_banner -nostdin -loglevel error -y -progress pipe:1 -nostats -vaapi_device /dev/dri/renderD128 -i {entrada} -map 0:v:0 " +
				"-vf scale=1280:720:flags=lanczos,format=nv12,hwupload -c:v h264_vaapi -qp 30 -map 0:a:0 -c:a aac -b:a 64k -ac 2 -sn -dn -movflags +faststart {saida}"}},
		{Pedido{Predefinicao: "equilibrio", Acelerar: "nvenc"}, "-equilibrio", ".mp4", 1920, 1080, []string{inicio +
			"-c:v h264_nvenc -preset p5 -rc vbr -cq 24 -b:v 0 -map 0:a:0 -c:a aac -b:a 128k -sn -dn -movflags +faststart {saida}"}},
		{Pedido{Predefinicao: "sem-audio"}, "-sem-audio", ".mp4", 1920, 1080, []string{inicio + "-c:v copy -an -sn -dn {saida}"}},
		{Pedido{Predefinicao: "gif"}, "", ".gif", 480, 270, []string{inicio +
			"-vf fps=12,scale=480:270:flags=lanczos,split[a][b];[a]palettegen=stats_mode=diff[p];[b][p]paletteuse=dither=bayer:bayer_scale=5 -c:v gif -loop 0 -an -sn -dn {saida}"}},
		{Pedido{Predefinicao: "webp", FPS: 20, Largura: 320}, "", ".webp", 320, 180, []string{inicio +
			"-vf fps=20,scale=320:180:flags=lanczos -c:v libwebp_anim -lossless 0 -q:v 60 -compression_level 4 -loop 0 -an -sn -dn {saida}"}},
	}
	for _, c := range casos {
		p, err := Planeja(video(), c.pedido, temTudo)
		if err != nil {
			t.Errorf("%+v: %v", c.pedido, err)
			continue
		}
		var got []string
		for _, a := range p.Passadas {
			got = append(got, strings.Join(a, " "))
		}
		if !reflect.DeepEqual(got, c.args) {
			t.Errorf("%+v:\n veio   %q\n queria %q", c.pedido, got, c.args)
		}
		if p.Sufixo != c.sufixo || p.Ext != c.ext || p.Largura != c.largura || p.Altura != c.alt {
			t.Errorf("%+v: sufixo %q ext %q %dx%d", c.pedido, p.Sufixo, p.Ext, p.Largura, p.Altura)
		}
	}
}

func TestPlaneja_SoAceleraSePediremESeOFfmpegTiver(t *testing.T) {
	// Sem pedir, nunca: mesmo com a VAAPI e a NVENC na lista.
	for _, pre := range []string{"menor", "equilibrio", "alta", "caber", "resolucao"} {
		p, err := Planeja(video(), Pedido{Predefinicao: pre, AlvoMB: 20, Lado: 720}, temTudo)
		if err != nil {
			t.Fatal(err)
		}
		if junto := strings.Join(p.Passadas[0], " "); strings.Contains(junto, "vaapi") || strings.Contains(junto, "nvenc") || !strings.Contains(junto, "-c:v libx264") {
			t.Errorf("%s: %s", pre, junto)
		}
	}
	erros := []struct {
		pedido Pedido
		tem    map[string]bool
		trecho string
	}{
		{Pedido{Predefinicao: "menor", Acelerar: "vaapi"}, map[string]bool{"libx264": true}, "não tem o codificador h264_vaapi"},
		{Pedido{Predefinicao: "menor", Codec: "h265", Acelerar: "nvenc"}, temTudo, "não tem o codificador hevc_nvenc"},
		{Pedido{Predefinicao: "menor", Codec: "h265"}, map[string]bool{"libx264": true}, "não tem o codificador libx265"},
		{Pedido{Predefinicao: "menor", Codec: "vp9"}, temTudo, "não conheço o codec"},
		{Pedido{Predefinicao: "menor", Acelerar: "qsv"}, temTudo, "não conheço a aceleração"},
		{Pedido{Predefinicao: "banana"}, temTudo, "não conheço a predefinição"},
		{Pedido{Predefinicao: ""}, temTudo, "não conheço a predefinição"},
		{Pedido{Predefinicao: "imagem"}, temTudo, "isto é um vídeo"},
		{Pedido{Predefinicao: "resolucao", Lado: 500}, temTudo, "1080, 720 ou 480"},
		{Pedido{Predefinicao: "caber"}, temTudo, "é preciso o tamanho"},
		{Pedido{Predefinicao: "caber", AlvoMB: 0.1}, temTudo, "não cabe"},
		{Pedido{Predefinicao: "gif", FPS: 500}, temTudo, "--fps"},
		{Pedido{Predefinicao: "gif"}, map[string]bool{}, "não tem o codificador gif"},
	}
	for _, c := range erros {
		if _, err := Planeja(video(), c.pedido, c.tem); err == nil || !strings.Contains(err.Error(), c.trecho) {
			t.Errorf("%+v: %v", c.pedido, err)
		}
	}
	mudo := video()
	mudo.Audio = nil
	if _, err := Planeja(mudo, Pedido{Predefinicao: "sem-audio"}, temTudo); err == nil || !strings.Contains(err.Error(), "já não tem áudio") {
		t.Errorf("sem-audio num vídeo mudo: %v", err)
	}
	p, _ := Planeja(mudo, Pedido{Predefinicao: "caber", AlvoMB: 8}, temTudo)
	if p == nil || p.AudioBps != 0 || !strings.Contains(strings.Join(p.Passadas[1], " "), " -an ") || p.Sufixo != "-8mb" {
		t.Errorf("caber sem áudio: %+v", p)
	}
	semDuracao := video()
	semDuracao.Duracao = nil
	if _, err := Planeja(semDuracao, Pedido{Predefinicao: "menor"}, temTudo); err == nil {
		t.Error("sem duração devia falhar")
	}
	// Sem o SVT-AV1, o AV1 sai pelo libaom.
	p, err := Planeja(video(), Pedido{Predefinicao: "alta", Codec: "av1"}, map[string]bool{"libaom-av1": true})
	if err != nil || !strings.Contains(strings.Join(p.Passadas[0], " "), "-c:v libaom-av1 -cpu-used 6 -row-mt 1 -pix_fmt yuv420p -crf 27 -b:v 0") {
		t.Errorf("libaom: %v", err)
	}
}

func TestPlaneja_Imagem(t *testing.T) {
	foto := &Midia{Arquivo: "/f/foto.PNG", Tipo: "imagem", Tamanho: 9000000, Largura: 4001, Altura: 3001, Codec: "png"}
	inicio := "-hide_banner -nostdin -loglevel error -y -progress pipe:1 -nostats -i {entrada} -map 0:v:0 -frames:v 1 "
	casos := []struct {
		pedido       Pedido
		ext          string
		largura, alt int
		args         string
	}{
		{Pedido{Predefinicao: "imagem"}, ".jpg", 4001, 3001, inicio + "-c:v mjpeg -q:v 8 -pix_fmt yuvj420p -update 1 {saida}"},
		{Pedido{Predefinicao: "imagem", Formato: "jpeg", Qualidade: 100}, ".jpg", 4001, 3001, inicio + "-c:v mjpeg -q:v 2 -pix_fmt yuvj420p -update 1 {saida}"},
		{Pedido{Predefinicao: "imagem", Formato: "jpeg", Qualidade: 1, Largura: 1600}, ".jpg", 1600, 1200, inicio + "-vf scale=1600:1200:flags=lanczos -c:v mjpeg -q:v 31 -pix_fmt yuvj420p -update 1 {saida}"},
		{Pedido{Predefinicao: "imagem", Formato: "webp", Qualidade: 75, Largura: 9000}, ".webp", 4001, 3001, inicio + "-c:v libwebp -quality 75 -compression_level 6 {saida}"},
		{Pedido{Predefinicao: "imagem", Formato: "avif", Qualidade: 60}, ".avif", 4000, 3000, inicio + "-vf scale=4000:3000:flags=lanczos -c:v libaom-av1 -still-picture 1 -crf 27 -b:v 0 -cpu-used 6 {saida}"},
	}
	for _, c := range casos {
		p, err := Planeja(foto, c.pedido, temTudo)
		if err != nil {
			t.Errorf("%+v: %v", c.pedido, err)
			continue
		}
		if got := strings.Join(p.Passadas[0], " "); len(p.Passadas) != 1 || got != c.args || p.Ext != c.ext || p.Sufixo != "-menor" || p.Largura != c.largura || p.Altura != c.alt {
			t.Errorf("%+v:\n veio   %s (%s %dx%d)\n queria %s", c.pedido, got, p.Ext, p.Largura, p.Altura, c.args)
		}
	}
	erros := map[string]Pedido{
		"predefinição de vídeo": {Predefinicao: "menor"},
		"--qualidade":           {Predefinicao: "imagem", Qualidade: 101},
		"--largura":             {Predefinicao: "imagem", Largura: 3},
		"não conheço o formato": {Predefinicao: "imagem", Formato: "bmp"},
	}
	for trecho, pedido := range erros {
		if _, err := Planeja(foto, pedido, temTudo); err == nil || !strings.Contains(err.Error(), trecho) {
			t.Errorf("%+v: %v", pedido, err)
		}
	}
	if _, err := Planeja(foto, Pedido{Predefinicao: "imagem", Formato: "avif"}, map[string]bool{"mjpeg": true}); err == nil || !strings.Contains(err.Error(), "AVIF") {
		t.Errorf("avif sem AV1: %v", err)
	}
}

func TestComprime_DoComecoAoFimComProgressoDeVerdade(t *testing.T) {
	m := monta(t, probeVideo, strings.Repeat("x", 12000))
	arq := original(t, "-i %03d *concat: 'aspas\" $(rm) férias.MP4", 48000)
	fim, fases, err := comprime(t, arq, Pedido{Predefinicao: "menor"})
	if err != nil {
		t.Fatal(err)
	}
	pasta := filepath.Dir(arq)
	saida := filepath.Join(pasta, "-i %03d *concat: 'aspas\" $(rm) férias-menor.mp4")
	if !fim.Ok || fim.Fase != "fim" || fim.Saida != saida || fim.Antes != 48000 || fim.Depois != 12000 || fim.Economia != 75 || fim.Maior {
		t.Errorf("fim: %+v", fim)
	}
	if b, _ := os.ReadFile(saida); len(b) != 12000 {
		t.Error("a saída não está no nome final")
	}
	if fi, _ := os.Stat(saida); fi.Mode().Perm() != 0o644 {
		t.Errorf("modo da saída: %v", fi.Mode())
	}
	if fi, _ := os.Stat(arq); fi.Size() != 48000 {
		t.Error("o original foi tocado")
	}
	nomes, _ := os.ReadDir(pasta)
	if len(nomes) != 2 {
		t.Errorf("sobrou arquivo pela metade: %v", nomes)
	}
	quer := []string{
		`{"fase":"plano","saida":` + jsonDe(saida) + `,"passadas":1,"largura":1280,"altura":720}`,
		`{"fase":"codificando","passada":1,"passadas":1,"pct":25,"velocidade":null,"resta_s":null,"tamanho_bytes":1000}`,
		`{"fase":"codificando","passada":1,"passadas":1,"pct":50,"velocidade":2.5,"resta_s":13,"tamanho_bytes":2000}`,
		`{"fase":"codificando","passada":1,"passadas":1,"pct":99.9,"velocidade":3.12,"resta_s":0,"tamanho_bytes":4000}`,
	}
	if !reflect.DeepEqual(fases, quer) {
		t.Errorf("fases:\n %s\nqueria:\n %s", strings.Join(fases, "\n "), strings.Join(quer, "\n "))
	}
	// O ffmpeg nunca viu o nome do arquivo: entrou pelo atalho, saiu num arquivo escondido.
	c := osDoFfmpeg(m)
	if len(c) != 1 {
		t.Fatalf("ffmpeg: %q", c)
	}
	args := c[0]
	entrada, parcial := "", args[len(args)-1]
	for i, a := range args {
		if a == "-i" {
			entrada = args[i+1]
		}
		if strings.Contains(a, "férias") && a != parcial {
			t.Errorf("o ffmpeg viu o nome: %q", a)
		}
	}
	if filepath.Base(entrada) != "entrada.mp4" || filepath.Dir(parcial) != pasta || !strings.HasPrefix(filepath.Base(parcial), ".mkcomprimir-") ||
		!strings.HasSuffix(parcial, ".parcial.mp4") {
		t.Errorf("entrada %q, saída %q", entrada, parcial)
	}
	if args[0] != "-hide_banner" || args[1] != "-nostdin" {
		t.Errorf("args: %q", args)
	}
}

func jsonDe(s string) string {
	b, _ := json.Marshal(s)
	// O motor não escapa <, > e & (base.Escreve); aqui o Marshal do teste escapa.
	r := strings.NewReplacer(`\u003c`, "<", `\u003e`, ">", `\u0026`, "&")
	return r.Replace(string(b))
}

func TestComprime_NuncaSobrescreveAcrescentaNumero(t *testing.T) {
	monta(t, probeVideo, "novo")
	arq := original(t, "férias.mp4", 48000)
	pasta := filepath.Dir(arq)
	for _, n := range []string{"férias-menor.mp4", "férias-menor-2.mp4"} {
		if err := os.WriteFile(filepath.Join(pasta, n), []byte("o que já estava"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	fim, fases, err := comprime(t, arq, Pedido{Predefinicao: "menor"})
	if err != nil {
		t.Fatal(err)
	}
	if fim.Saida != filepath.Join(pasta, "férias-menor-3.mp4") || !strings.Contains(fases[0], "férias-menor-3.mp4") {
		t.Errorf("saída: %q", fim.Saida)
	}
	for _, n := range []string{"férias-menor.mp4", "férias-menor-2.mp4"} {
		if b, _ := os.ReadFile(filepath.Join(pasta, n)); string(b) != "o que já estava" {
			t.Errorf("%s foi sobrescrito", n)
		}
	}
	// Nomes sem extensão, com dois pontos, escondidos.
	casos := map[string]string{"a.b.mkv": "a.b-menor.mp4", "semext": "semext-menor.mp4", ".oculto": ".oculto-menor.mp4", "x.mp4": "x-menor.mp4"}
	for nome, quer := range casos {
		if got := filepath.Base(nomeLivre(filepath.Join(pasta, nome), "-menor", ".mp4")); got != quer {
			t.Errorf("nomeLivre(%q) = %q, queria %q", nome, got, quer)
		}
	}
	// Um link quebrado com o nome da saída também conta como "já existe".
	_ = os.Symlink("/nao/existe", filepath.Join(pasta, "x-menor.mp4"))
	if got := filepath.Base(nomeLivre(filepath.Join(pasta, "x.mp4"), "-menor", ".mp4")); got != "x-menor-2.mp4" {
		t.Errorf("link quebrado: %q", got)
	}
}

func TestComprime_CaberFazDuasPassadasEOProgressoEDoTrabalhoInteiro(t *testing.T) {
	m := monta(t, probeVideo, strings.Repeat("x", 7000))
	arq := original(t, "férias.mp4", 48000)
	fim, fases, err := comprime(t, arq, Pedido{Predefinicao: "caber", AlvoMB: 8})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(fim.Saida) != "férias-8mb.mp4" {
		t.Errorf("saída: %q", fim.Saida)
	}
	if !strings.HasSuffix(fases[0], `"passadas":2,"video_bps":898871,"audio_bps":96000,"largura":854,"altura":480}`) {
		t.Errorf("plano: %s", fases[0])
	}
	var pcts []float64
	for _, f := range fases[1:] {
		var v struct {
			Passada int
			Pct     float64
			Resta   *int64 `json:"resta_s"`
		}
		if err := json.Unmarshal([]byte(f), &v); err != nil {
			t.Fatal(err)
		}
		pcts = append(pcts, v.Pct)
		// O que resta na 1ª passada conta a 2ª inteira (62,4 s a 2,5x: 25 s).
		if v.Passada == 1 && v.Resta != nil && *v.Resta < 20 {
			t.Errorf("resta pouco demais na 1ª passada: %s", f)
		}
	}
	if quer := []float64{12.5, 25, 50, 62.5, 75, 99.9}; !reflect.DeepEqual(pcts, quer) {
		t.Errorf("porcentagens: %v", pcts)
	}
	c := osDoFfmpeg(m)
	if len(c) != 2 || c[0][len(c[0])-1] != "/dev/null" || !strings.HasSuffix(c[1][len(c[1])-1], ".parcial.mp4") {
		t.Fatalf("ffmpeg: %q", c)
	}
	// As duas passadas usam o mesmo arquivo de estatísticas, na pasta temporária.
	achou := func(args []string) string {
		for i, a := range args {
			if a == "-passlogfile" {
				return args[i+1]
			}
		}
		return ""
	}
	if achou(c[0]) == "" || achou(c[0]) != achou(c[1]) || filepath.Dir(achou(c[0])) != filepath.Dir(c[0][indice(c[0], "-i")+1]) {
		t.Errorf("passlogfile: %q e %q", achou(c[0]), achou(c[1]))
	}
}

func indice(args []string, qual string) int {
	for i, a := range args {
		if a == qual {
			return i
		}
	}
	return -1
}

func TestComprime_SaidaMaiorQueOOriginalAvisa(t *testing.T) {
	monta(t, probeVideo, strings.Repeat("x", 60000))
	fim, _, err := comprime(t, original(t, "a.mp4", 48000), Pedido{Predefinicao: "alta"})
	if err != nil {
		t.Fatal(err)
	}
	if !fim.Maior || fim.Economia != -25 || fim.Depois != 60000 {
		t.Errorf("fim: %+v", fim)
	}
	if _, err := os.Stat(fim.Saida); err != nil {
		t.Error("a saída maior fica lá: quem decide apagar é a pessoa")
	}
}

func TestComprime_ImagemNaoTemProgresso(t *testing.T) {
	m := monta(t, probeFoto, "jpeg pequeno")
	arq := original(t, "foto 100%d [1].PNG", 9000)
	fim, fases, err := comprime(t, arq, Pedido{Predefinicao: "imagem", Formato: "webp", Qualidade: 70, Largura: 2000})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(fim.Saida) != "foto 100%d [1]-menor.webp" || fim.Depois != 12 {
		t.Errorf("fim: %+v", fim)
	}
	if len(fases) != 1 || !strings.Contains(fases[0], `"fase":"plano"`) || !strings.Contains(fases[0], `"largura":2000,"altura":1500`) {
		t.Errorf("fases: %v", fases)
	}
	c := osDoFfmpeg(m)
	if filepath.Base(c[0][indice(c[0], "-i")+1]) != "entrada.png" {
		t.Errorf("entrada: %q", c[0])
	}
}

func TestComprime_ErroDoFfmpegApagaASaidaPelaMetade(t *testing.T) {
	m := monta(t, probeVideo, "")
	arq := original(t, "a.mp4", 48000)
	codificadores := `case " $* " in *" -encoders "*) printf '%s' ` + dubles.Aspas(codificadoresDoFfmpeg) + `; exit 0 ;; esac; for a in "$@"; do ultimo=$a; done; `
	m.Comando("ffmpeg", codificadores+`printf 'pela metade' >"$ultimo"; printf 'aviso\n[libx264] Error: sem memória\n' >&2; exit 1`)
	if _, _, err := comprime(t, arq, Pedido{Predefinicao: "menor"}); err == nil || err.Error() != "o ffmpeg não conseguiu: [libx264] Error: sem memória" {
		t.Errorf("erro: %v", err)
	}
	m.Comando("ffmpeg", codificadores+`exit 0`)
	if _, _, err := comprime(t, arq, Pedido{Predefinicao: "menor"}); err == nil || !strings.Contains(err.Error(), "sem escrever a saída") {
		t.Errorf("sem saída: %v", err)
	}
	m.Comando("ffmpeg", codificadores+`exit 7`)
	if _, _, err := comprime(t, arq, Pedido{Predefinicao: "menor"}); err == nil || !strings.Contains(err.Error(), "não conseguiu") {
		t.Errorf("mudo: %v", err)
	}
	if _, _, err := comprime(t, arq, Pedido{Predefinicao: "banana"}); err == nil {
		t.Error("predefinição inválida devia falhar")
	}
	if nomes, _ := os.ReadDir(filepath.Dir(arq)); len(nomes) != 1 {
		t.Errorf("sobrou arquivo: %v", nomes)
	}
	// Pasta sem permissão de escrita: frase, e nada de ffmpeg.
	m.Esquece("ffmpeg")
	_ = os.Chmod(filepath.Dir(arq), 0o500)
	defer os.Chmod(filepath.Dir(arq), 0o700)
	if _, _, err := comprime(t, arq, Pedido{Predefinicao: "menor"}); err == nil || !strings.Contains(err.Error(), "não dá para escrever na pasta") {
		t.Errorf("pasta só de leitura: %v", err)
	}
	if len(osDoFfmpeg(m)) != 0 {
		t.Error("rodou o ffmpeg sem ter onde escrever")
	}
}

func TestComprime_CancelarMataOFfmpegEApagaASaidaPelaMetade(t *testing.T) {
	m := monta(t, probeVideo, "")
	arq := original(t, "a.mp4", 48000)
	lento := `case " $* " in *" -encoders "*) printf '%s' ` + dubles.Aspas(codificadoresDoFfmpeg) + `; exit 0 ;; esac; for a in "$@"; do ultimo=$a; done
printf 'pela metade' >"$ultimo"
printf 'out_time_us=1000000\nspeed=1x\nprogress=continue\n'
exec /bin/sleep 30`
	m.Comando("ffmpeg", lento)

	// Cancelado de fora (o SIGTERM do motor fecha este canal).
	cancela := make(chan struct{})
	viu := make(chan struct{}, 8)
	feito := make(chan error, 1)
	go func() {
		_, err := Comprime(arq, Pedido{Predefinicao: "menor"}, func(v any) error {
			if _, ok := v.(*FaseCodificando); ok {
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
	if nomes, _ := os.ReadDir(filepath.Dir(arq)); len(nomes) != 1 {
		t.Errorf("a saída pela metade ficou: %v", nomes)
	}

	// Ninguém mais lê o que o motor escreve (a janela fechou): para também.
	_, err := Comprime(arq, Pedido{Predefinicao: "menor"}, func(v any) error {
		if _, ok := v.(*FaseCodificando); ok {
			return errors.New("cano fechado")
		}
		return nil
	}, nil)
	if !errors.Is(err, ErrCancelado) {
		t.Errorf("stdout fechado: %v", err)
	}
	if nomes, _ := os.ReadDir(filepath.Dir(arq)); len(nomes) != 1 {
		t.Errorf("a saída pela metade ficou: %v", nomes)
	}
}

func TestProgresso_LeAsLinhasDoFfmpeg(t *testing.T) {
	p := &progresso{duracao: 100, passada: 1, passadas: 1}
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
	if f == nil || f.Pct != 40 || *f.Velocidade != 0.5 || *f.Resta != 120 || f.Tamanho != 123 {
		t.Errorf("bloco: %+v", f)
	}
	// Passou da duração (acontece): não passa de 99,9 antes do fim.
	p.linha("out_time_us=500000000")
	if f := p.linha("progress=end"); f.Pct != 99.9 || *f.Resta != 0 {
		t.Errorf("além do fim: %+v", f)
	}
	// Sem duração conhecida: 0%, sem previsão.
	s := &progresso{passada: 1, passadas: 1}
	s.linha("out_time_us=5000000")
	s.linha("speed=2x")
	if f := s.linha("progress=continue"); f.Pct != 0 || f.Resta != nil {
		t.Errorf("sem duração: %+v", f)
	}
}

func TestDescarta_SoApagaOQueTemCaraDeSaida(t *testing.T) {
	t.Setenv("MKUTIL_HOME", t.TempDir())
	pasta := t.TempDir()
	poe := func(nome string) string {
		c := filepath.Join(pasta, nome)
		_ = os.MkdirAll(filepath.Dir(c), 0o755)
		if err := os.WriteFile(c, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		return c
	}
	orig := poe("férias.mp4")
	saida := poe("férias-alta.mp4")
	outro := poe("trabalho.mp4")
	emOutra := poe("sub/férias-alta.mp4")
	_ = os.Mkdir(filepath.Join(pasta, "férias-pasta"), 0o755)
	_ = os.Symlink(outro, filepath.Join(pasta, "férias-link.mp4"))
	_ = os.Link(orig, filepath.Join(pasta, "férias-duro.mp4"))
	recusas := [][2]string{
		{orig, orig}, {outro, orig}, {emOutra, orig}, {filepath.Join(pasta, "férias-pasta"), orig}, {filepath.Join(pasta, "férias-link.mp4"), orig},
		{filepath.Join(pasta, "férias-duro.mp4"), orig}, {filepath.Join(pasta, "sub", "..", "trabalho.mp4"), orig}, {saida, filepath.Join(pasta, "nao-existe.mp4")},
		{filepath.Join(pasta, "férias"), orig}, {pasta, orig}, {"/", orig},
	}
	for _, c := range recusas {
		if _, err := Descarta(c[0], c[1]); err == nil || !base.FoiRecusa(err) {
			t.Errorf("Descarta(%q) devia recusar: %v", c[0], err)
		}
	}
	for _, c := range []string{orig, outro, emOutra, saida} {
		if _, err := os.Stat(c); err != nil {
			t.Errorf("sumiu %s", c)
		}
	}
	if _, err := Descarta(filepath.Join(pasta, "férias-nao-existe.mp4"), orig); err == nil || base.FoiRecusa(err) {
		t.Errorf("ausente é erro comum: %v", err)
	}
	apagado, err := Descarta(saida, orig)
	if err != nil || apagado != saida {
		t.Fatalf("Descarta: %q, %v", apagado, err)
	}
	if _, err := os.Stat(saida); err == nil {
		t.Error("não apagou")
	}
	if _, err := os.Stat(orig); err != nil {
		t.Error("apagou o original")
	}
}
