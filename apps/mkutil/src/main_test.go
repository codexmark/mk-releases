package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"mkutil/internal/base"
	"mkutil/internal/dubles"
	"mkutil/internal/instala/falso"
	sq "mkutil/internal/squash/falso"
)

// saidaSegura é um buffer que aguenta escrita e leitura ao mesmo tempo.
type saidaSegura struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *saidaSegura) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *saidaSegura) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// mundo monta a home falsa e o PATH só de dublês.
func mundo(t *testing.T) (*dubles.Mesa, string) {
	t.Helper()
	raiz := t.TempDir()
	t.Setenv("MKUTIL_HOME", filepath.Join(raiz, "casa"))
	t.Setenv("MKUTIL_LIBS", filepath.Join(raiz, "libs"))
	t.Setenv("MKUTIL_LIMITE_MB", "")
	t.Setenv("MKUTIL_LIMITE_ENTRADAS", "")
	t.Setenv("TMPDIR", filepath.Join(raiz, "tmp"))
	for _, p := range []string{"casa", "libs", "tmp", "baixados"} {
		if err := os.MkdirAll(filepath.Join(raiz, p), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return dubles.Nova(t), filepath.Join(raiz, "baixados")
}

func grava(t *testing.T, caminho string, conteudo []byte) string {
	t.Helper()
	if err := os.WriteFile(caminho, conteudo, 0o644); err != nil {
		t.Fatal(err)
	}
	return caminho
}

// roda chama um subcomando e devolve o código e as linhas de JSON que saíram.
func roda(t *testing.T, args ...string) (int, []map[string]any) {
	t.Helper()
	antes := base.Saida
	b := &saidaSegura{}
	base.Saida = b
	defer func() { base.Saida = antes }()
	codigo := rodar(args)
	return codigo, linhas(t, b.String())
}

func linhas(t *testing.T, bruto string) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, l := range strings.Split(strings.TrimRight(bruto, "\n"), "\n") {
		var v map[string]any
		if err := json.Unmarshal([]byte(l), &v); err != nil {
			t.Fatalf("linha que não é um objeto JSON: %q", l)
		}
		out = append(out, v)
	}
	return out
}

func chaves(v map[string]any) []string {
	out := make([]string, 0, len(v))
	for k := range v {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func querChaves(t *testing.T, oQue string, v map[string]any, quer ...string) {
	t.Helper()
	sort.Strings(quer)
	if got := chaves(v); !reflect.DeepEqual(got, quer) {
		t.Errorf("%s: chaves\n veio   %v\n queria %v", oQue, got, quer)
	}
}

// querErro confere a resposta de erro do contrato: uma linha {"erro": …}.
func querErro(t *testing.T, codigo int, trecho string, args ...string) {
	t.Helper()
	c, saida := roda(t, args...)
	ultima := saida[len(saida)-1]
	frase, _ := ultima["erro"].(string)
	if c != codigo || len(ultima) != 1 || !strings.Contains(frase, trecho) {
		t.Errorf("%v: código %d e %v; queria código %d com %q", args, c, ultima, codigo, trecho)
	}
}

func krita() []byte {
	return falso.AppImage(false, []sq.Arquivo{
		{Caminho: "krita.desktop", Conteudo: []byte("[Desktop Entry]\nName=Krita\nExec=krita %F\nIcon=krita\n")},
		{Caminho: ".DirIcon", Conteudo: falso.PNG},
	})
}

var chavesDaFicha = []string{"arquivo", "tipo", "id", "nome", "versao", "descricao", "tamanho_bytes", "destino", "atalho", "ja_instalado",
	"precisa_senha", "plano", "avisos", "appimage", "pacote", "compactado"}

func TestSemSubcomandoOuComUmQueNaoExiste_UsoEmJSONESaida1(t *testing.T) {
	for _, args := range [][]string{nil, {"banana"}, {"--help"}} {
		c, saida := roda(t, args...)
		frase, _ := saida[0]["erro"].(string)
		if c != base.Erro || len(saida) != 1 || !strings.HasPrefix(frase, "uso: mkutil-motor <") {
			t.Errorf("%v: %d %v", args, c, saida)
		}
		for _, sub := range []string{"inspecionar", "instalar", "instalados", "desinstalar", "codificadores", "midia", "comprimir", "descartar"} {
			if !strings.Contains(frase, sub) {
				t.Errorf("o uso não lista %s", sub)
			}
		}
	}
	if len(comandos) != 8 {
		t.Errorf("há %d subcomandos; o contrato tem 8", len(comandos))
	}
}

func TestOpcoes(t *testing.T) {
	pos, ops, err := opcoes([]string{"a", "--lado", "720", "--sem-audio", "b", "--", "-c", "--lado"}, []string{"--lado"}, []string{"--sem-audio"})
	if err != nil || !reflect.DeepEqual(pos, []string{"a", "b", "-c", "--lado"}) || ops["--lado"] != "720" || ops["--sem-audio"] == "" {
		t.Errorf("veio %v %v %v", pos, ops, err)
	}
	for _, ruim := range [][]string{{"--nao-existe"}, {"-x"}, {"--lado"}, {"--lado", "1", "--lado", "2"}} {
		if _, _, err := opcoes(ruim, []string{"--lado"}, nil); err == nil {
			t.Errorf("%v devia falhar", ruim)
		}
	}
	if pos, _, err := opcoes([]string{"-"}, nil, nil); err != nil || len(pos) != 1 {
		t.Errorf("um traço só é posicional: %v %v", pos, err)
	}
}

func TestInspecionar_Contrato(t *testing.T) {
	m, baixados := mundo(t)
	arq := grava(t, filepath.Join(baixados, "Krita-5.2.2-x86_64.AppImage"), krita())
	c, saida := roda(t, "inspecionar", arq)
	if c != 0 || len(saida) != 1 {
		t.Fatalf("código %d, %v", c, saida)
	}
	f := saida[0]
	querChaves(t, "ficha", f, chavesDaFicha...)
	querChaves(t, "appimage", f["appimage"].(map[string]any), "tipo", "tem_desktop", "tem_icone", "fuse")
	if f["tipo"] != "appimage" || f["id"] != "krita" || f["nome"] != "Krita" || f["versao"] != "5.2.2" || f["descricao"] != nil ||
		f["pacote"] != nil || f["compactado"] != nil || f["ja_instalado"] != false || f["arquivo"] != arq {
		t.Errorf("ficha: %v", f)
	}

	// Pacote.
	m.Responde("bsdtar", falso.PKGINFO, 0)
	pkg := grava(t, filepath.Join(baixados, "htop-3.3.0-3-x86_64.pkg.tar.zst"), []byte("x"))
	_, saida = roda(t, "inspecionar", pkg)
	f = saida[0]
	querChaves(t, "ficha de pacote", f, chavesDaFicha...)
	querChaves(t, "pacote", f["pacote"].(map[string]any), "nome", "versao", "descricao", "arquitetura", "tamanho_instalado_bytes", "dependencias", "instalado")
	if f["precisa_senha"] != true || f["destino"] != nil || f["atalho"] != nil || f["pacote"].(map[string]any)["instalado"] != nil {
		t.Errorf("pacote: %v", f)
	}

	// Compactado.
	tgz := grava(t, filepath.Join(baixados, "app-1.0.tar.gz"), falso.TarGz([]falso.Entrada{{Nome: "app/app", Conteudo: "\x7fELF....", Modo: 0o755}}))
	_, saida = roda(t, "inspecionar", tgz)
	f = saida[0]
	querChaves(t, "ficha de compactado", f, chavesDaFicha...)
	querChaves(t, "compactado", f["compactado"].(map[string]any), "formato", "entradas", "tamanho_total_bytes", "pasta_unica", "executaveis", "atalhos", "executavel_sugerido")

	// Um nome que começa com "-" entra depois de "--".
	traco := grava(t, filepath.Join(baixados, "-app.AppImage"), krita())
	t.Chdir(baixados)
	querErro(t, 1, "não conheço a opção", "inspecionar", "-app.AppImage")
	if c, saida := roda(t, "inspecionar", "--", "-app.AppImage"); c != 0 || saida[0]["arquivo"] != traco {
		t.Errorf("com --: %d %v", c, saida)
	}

	querErro(t, 1, "uso:", "inspecionar")
	querErro(t, 1, "uso:", "inspecionar", arq, pkg)
	querErro(t, 1, "não achei o arquivo", "inspecionar", filepath.Join(baixados, "nada.AppImage"))
	querErro(t, 1, "não sei instalar", "inspecionar", grava(t, filepath.Join(baixados, "a.txt"), []byte("texto qualquer aqui")))
	// Recusa por segurança: código 2.
	mau := grava(t, filepath.Join(baixados, "mau.tar.gz"), falso.TarGz([]falso.Entrada{{Nome: "../../fora", Conteudo: "x"}}))
	querErro(t, 2, "sai da pasta", "inspecionar", mau)
	querErro(t, 2, "sai da pasta", "instalar", mau)
}

func TestInstalar_SoltaAsFasesEAcabaNoFim(t *testing.T) {
	_, baixados := mundo(t)
	arq := grava(t, filepath.Join(baixados, "Krita-5.2.2-x86_64.AppImage"), krita())
	c, saida := roda(t, "instalar", arq)
	if c != 0 || len(saida) < 3 {
		t.Fatalf("código %d, %v", c, saida)
	}
	for i, l := range saida {
		if _, ok := l["fase"].(string); !ok {
			t.Errorf("linha %d sem fase: %v", i, l)
		}
	}
	if saida[0]["fase"] != "copiando" || saida[0]["pct"] == nil || saida[len(saida)-2]["fase"] != "atalho" {
		t.Errorf("fases: %v", saida)
	}
	fim := saida[len(saida)-1]
	querChaves(t, "fim", fim, "fase", "ok", "id", "tipo", "nome", "destino", "atalho", "avisos")
	if fim["fase"] != "fim" || fim["ok"] != true || fim["id"] != "krita" || fim["destino"] != base.Aplicativos("krita.AppImage") {
		t.Errorf("fim: %v", fim)
	}

	querErro(t, 1, "já está instalado", "instalar", arq)
	if c, saida := roda(t, "instalar", arq, "--substituir", "--sem-atalho"); c != 0 || saida[len(saida)-1]["atalho"] != nil {
		t.Errorf("--substituir --sem-atalho: %d %v", c, saida)
	}
	querErro(t, 1, "uso:", "instalar")
	querErro(t, 1, "não conheço a opção", "instalar", arq, "--forcar")
	querErro(t, 1, "pede um valor", "instalar", arq, "--executavel")

	// instalados e desinstalar.
	c, saida = roda(t, "instalados")
	itens := saida[0]["itens"].([]any)
	if c != 0 || len(itens) != 1 {
		t.Fatalf("instalados: %d %v", c, saida)
	}
	item := itens[0].(map[string]any)
	querChaves(t, "item", item, "id", "tipo", "nome", "versao", "origem", "destino", "atalho", "icone", "instalado_em_s", "tamanho_bytes", "existe")
	if item["existe"] != true || item["atalho"] != nil || item["origem"] != "Krita-5.2.2-x86_64.AppImage" {
		t.Errorf("item: %v", item)
	}
	_ = os.Remove(base.Aplicativos("krita.AppImage"))
	if _, saida = roda(t, "instalados"); saida[0]["itens"].([]any)[0].(map[string]any)["existe"] != false {
		t.Error("existe devia ser false com o destino apagado")
	}
	querErro(t, 1, "uso:", "instalados", "x")
	querErro(t, 1, "uso:", "desinstalar")
	querErro(t, 1, "não foi instalado", "desinstalar", "gimp")
	querErro(t, 1, "não é nome", "desinstalar", "../krita")
	c, saida = roda(t, "desinstalar", "krita")
	querChaves(t, "desinstalar", saida[0], "ok", "id", "removidos")
	if c != 0 || saida[0]["ok"] != true || saida[0]["id"] != "krita" {
		t.Errorf("desinstalar: %d %v", c, saida)
	}
	if _, saida = roda(t, "instalados"); len(saida[0]["itens"].([]any)) != 0 {
		t.Errorf("depois de desinstalar: %v", saida)
	}
	// Registro adulterado: recusa com código 2.
	grava(t, base.Registro(), []byte(`{"versao":1,"itens":[{"id":"x","tipo":"appimage","destino":"`+base.Home("Documentos")+`"}]}`))
	_ = os.MkdirAll(base.Home("Documentos"), 0o755)
	querErro(t, 2, "não apago", "desinstalar", "x")
	if _, err := os.Stat(base.Home("Documentos")); err != nil {
		t.Error("apagou os Documentos")
	}
}

const probe = `{"streams":[{"codec_name":"h264","codec_type":"video","width":1920,"height":1080,"avg_frame_rate":"30/1","bit_rate":"6000000","nb_frames":"1870"},
 {"codec_name":"aac","codec_type":"audio","channels":2,"bit_rate":"128000"}],"format":{"format_name":"mov,mp4","duration":"62.4"}}`

const encoders = " ------\n V....D libx264  x\n V....D gif  x\n VFS..D mjpeg  x\n"

func ffmpegDeMentira(m *dubles.Mesa, depois string) {
	m.Responde("ffprobe", probe, 0)
	m.Comando("ffmpeg", `case " $* " in *" -encoders "*) printf '%s' `+dubles.Aspas(encoders)+`; exit 0 ;; esac
for a in "$@"; do ultimo=$a; done
printf 'pela metade' >"$ultimo"
printf 'out_time_us=31200000\nspeed=2x\ntotal_size=5\nprogress=continue\n'
`+depois)
}

func TestMidiaECodificadores_Contrato(t *testing.T) {
	m, baixados := mundo(t)
	arq := grava(t, filepath.Join(baixados, "férias.mp4"), make([]byte, 5000))
	c, saida := roda(t, "codificadores")
	querChaves(t, "codificadores", saida[0], "ffmpeg", "codecs", "aceleracao", "imagens", "animados")
	if c != 0 || saida[0]["ffmpeg"] != false || len(saida[0]["codecs"].([]any)) != 0 {
		t.Errorf("sem ffmpeg: %d %v", c, saida)
	}
	querErro(t, 1, "ffprobe", "midia", arq)
	ffmpegDeMentira(m, "")
	if _, saida = roda(t, "codificadores"); saida[0]["ffmpeg"] != true || !reflect.DeepEqual(saida[0]["codecs"], []any{"h264"}) ||
		!reflect.DeepEqual(saida[0]["imagens"], []any{"jpeg"}) || !reflect.DeepEqual(saida[0]["animados"], []any{"gif"}) || len(saida[0]["aceleracao"].([]any)) != 0 {
		t.Errorf("codificadores: %v", saida)
	}
	c, saida = roda(t, "midia", arq)
	querChaves(t, "midia", saida[0], "arquivo", "tipo", "tamanho_bytes", "duracao_s", "largura", "altura", "fps", "codec", "taxa_bps", "audio")
	querChaves(t, "audio", saida[0]["audio"].(map[string]any), "codec", "taxa_bps", "canais")
	if c != 0 || saida[0]["tipo"] != "video" || saida[0]["tamanho_bytes"] != 5000.0 || saida[0]["duracao_s"] != 62.4 || saida[0]["arquivo"] != arq {
		t.Errorf("midia: %d %v", c, saida)
	}
	querErro(t, 1, "uso:", "midia")
	querErro(t, 1, "uso:", "codificadores", "x")
	querErro(t, 1, "não achei o arquivo", "midia", filepath.Join(baixados, "nada.mp4"))
}

func TestComprimir_Contrato(t *testing.T) {
	m, baixados := mundo(t)
	arq := grava(t, filepath.Join(baixados, "férias.mp4"), make([]byte, 5000))
	ffmpegDeMentira(m, "")
	c, saida := roda(t, "comprimir", arq, "--predefinicao", "caber", "--alvo-mb", "8")
	if c != 0 || len(saida) != 4 {
		t.Fatalf("código %d, %v", c, saida)
	}
	querChaves(t, "plano", saida[0], "fase", "saida", "passadas", "video_bps", "audio_bps", "largura", "altura")
	querChaves(t, "codificando", saida[1], "fase", "passada", "passadas", "pct", "velocidade", "resta_s", "tamanho_bytes")
	querChaves(t, "fim", saida[3], "fase", "ok", "saida", "antes_bytes", "depois_bytes", "economia_pct", "maior")
	if saida[0]["fase"] != "plano" || saida[1]["fase"] != "codificando" || saida[1]["pct"] != 25.0 || saida[2]["pct"] != 75.0 || saida[3]["fase"] != "fim" {
		t.Errorf("fases: %v", saida)
	}
	feita := filepath.Join(baixados, "férias-8mb.mp4")
	if saida[3]["saida"] != feita || saida[3]["depois_bytes"] != 11.0 || saida[3]["maior"] != false {
		t.Errorf("fim: %v", saida[3])
	}
	// Fora do caber, o plano não traz as taxas.
	_, saida = roda(t, "comprimir", arq, "--predefinicao", "menor", "--sem-audio")
	querChaves(t, "plano sem taxa", saida[0], "fase", "saida", "passadas", "largura", "altura")
	if !strings.HasSuffix(saida[0]["saida"].(string), "férias-menor-sem-audio.mp4") {
		t.Errorf("saída: %v", saida[0])
	}

	querErro(t, 1, "uso:", "comprimir", arq)
	querErro(t, 1, "uso:", "comprimir", "--predefinicao", "menor")
	querErro(t, 1, "não conheço a predefinição", "comprimir", arq, "--predefinicao", "banana")
	querErro(t, 1, "--lado pede um número", "comprimir", arq, "--predefinicao", "resolucao", "--lado", "grande")
	querErro(t, 1, "--alvo-mb pede um número", "comprimir", arq, "--predefinicao", "caber", "--alvo-mb", "-3")
	querErro(t, 1, "--alvo-mb pede um número", "comprimir", arq, "--predefinicao", "caber", "--alvo-mb", "NaN")
	querErro(t, 1, "não tem o codificador h264_vaapi", "comprimir", arq, "--predefinicao", "menor", "--acelerar", "vaapi")
	querErro(t, 1, "não conheço a opção", "comprimir", arq, "--predefinicao", "menor", "--crf", "1")

	// descartar.
	querErro(t, 2, "não é uma saída", "descartar", arq, "--original", arq)
	querErro(t, 1, "uso:", "descartar", feita)
	c, saida = roda(t, "descartar", feita, "--original", arq)
	querChaves(t, "descartar", saida[0], "ok", "apagado")
	if c != 0 || saida[0]["apagado"] != feita {
		t.Errorf("descartar: %d %v", c, saida)
	}
	if _, err := os.Stat(arq); err != nil {
		t.Error("o original sumiu")
	}
}

func TestComprimir_SIGTERMCancelaApagaASaidaESaiCom0(t *testing.T) {
	m, baixados := mundo(t)
	arq := grava(t, filepath.Join(baixados, "férias.mp4"), make([]byte, 5000))
	ffmpegDeMentira(m, "exec /bin/sleep 30")
	antes := base.Saida
	b := &saidaSegura{}
	base.Saida = b
	defer func() { base.Saida = antes }()
	feito := make(chan int, 1)
	go func() { feito <- rodar([]string{"comprimir", arq, "--predefinicao", "menor"}) }()
	prazo := time.Now().Add(10 * time.Second)
	for !strings.Contains(b.String(), `"codificando"`) {
		if time.Now().After(prazo) {
			t.Fatalf("o progresso não chegou: %q", b.String())
		}
		time.Sleep(5 * time.Millisecond)
	}
	// O motor já está escutando o sinal: o teste manda para si mesmo.
	if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	select {
	case c := <-feito:
		if c != base.Certo {
			t.Errorf("código %d", c)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("o SIGTERM não cancelou")
	}
	saida := linhas(t, b.String())
	if ultima := saida[len(saida)-1]; len(ultima) != 1 || ultima["fase"] != "cancelado" {
		t.Errorf("última linha: %v", ultima)
	}
	if nomes, _ := os.ReadDir(baixados); len(nomes) != 1 {
		t.Errorf("a saída pela metade ficou: %v", nomes)
	}
}
