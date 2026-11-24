package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"mkcorte/internal/base"
	"mkcorte/internal/dubles"
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

const probe = `{"streams":[{"index":0,"codec_name":"h264","profile":"High","level":40,"codec_type":"video","width":1920,"height":1080,"pix_fmt":"yuv420p",
 "sample_aspect_ratio":"1:1","avg_frame_rate":"30/1"},{"index":1,"codec_name":"aac","codec_type":"audio","channels":2,"sample_rate":"48000"}],
 "format":{"format_name":"mov,mp4","duration":"60.0","bit_rate":"4000000"}}`

// mundo monta a home e o cache falsos, o PATH só de dublês e uma pasta de
// vídeos com um nome difícil.
func mundo(t *testing.T) (*dubles.Mesa, string) {
	t.Helper()
	raiz := t.TempDir()
	t.Setenv("MKCORTE_HOME", filepath.Join(raiz, "casa"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(raiz, "cache"))
	t.Setenv("TMPDIR", filepath.Join(raiz, "tmp"))
	for _, p := range []string{"casa", "cache", "tmp", "vídeos"} {
		if err := os.MkdirAll(filepath.Join(raiz, p), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	m := dubles.Nova(t)
	m.Comando("ffprobe", `case " $* " in *" -select_streams "*) printf '0.0,K__\n1.0,___\n2.0,K__\n'; exit 0 ;; esac; printf '%s' `+dubles.Aspas(probe))
	ffmpegDeMentira(m, "")
	return m, filepath.Join(raiz, "vídeos")
}

func ffmpegDeMentira(m *dubles.Mesa, depois string) {
	m.Comando("ffmpeg", `for a in "$@"; do ultimo=$a; done
printf 'feito' >"$ultimo"
printf 'out_time_us=5000000\nspeed=2x\ntotal_size=10\nprogress=continue\n'
`+depois)
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

func querChaves(t *testing.T, oQue string, v map[string]any, quer ...string) {
	t.Helper()
	tem := make([]string, 0, len(v))
	for k := range v {
		tem = append(tem, k)
	}
	sort.Strings(tem)
	sort.Strings(quer)
	if strings.Join(tem, " ") != strings.Join(quer, " ") {
		t.Errorf("%s: as chaves são\n  %v\ne o contrato diz\n  %v", oQue, tem, quer)
	}
}

func querErro(t *testing.T, codigo int, trecho string, args ...string) {
	t.Helper()
	c, saida := roda(t, args...)
	if c != codigo || len(saida) != 1 || len(saida[0]) != 1 {
		t.Errorf("%q: código %d, saída %v", args, c, saida)
		return
	}
	if erro, _ := saida[0]["erro"].(string); !strings.Contains(erro, trecho) {
		t.Errorf("%q: o erro %q não tem %q", args, erro, trecho)
	}
}

func edicaoDeUm(t *testing.T, video, extra string) string {
	t.Helper()
	nome, _ := json.Marshal(video)
	return `{"fontes":[` + string(nome) + `],"clipes":[{"fonte":0,"inicio":2,"fim":12}]` + extra + `}`
}

// As chaves que o contrato promete em cada resposta.
var (
	chavesDaFonte = []string{"arquivo", "tamanho_bytes", "duracao_s", "largura", "altura", "fps", "fps_fracao", "codec", "perfil", "formato_de_pixel",
		"sar", "rotacao", "taxa_bps", "trilha", "tem_audio", "audio"}
	chavesDoPlano = []string{"modo", "duracao_s", "clipes", "largura", "altura", "fps", "tem_audio", "tamanho_estimado_bytes", "sem_recodificar",
		"saida", "comando", "avisos"}
)

func TestUso_SemSubcomandoOuComUmQueNaoExiste(t *testing.T) {
	mundo(t)
	querErro(t, 1, "uso: mkcorte-motor <exportar | miniaturas | plano | quadro | sondar>")
	querErro(t, 1, "uso: mkcorte-motor <", "banana")
}

func TestSondar_RespondeAFonteEAceitaNomeQueComecaComTraco(t *testing.T) {
	_, videos := mundo(t)
	arq := grava(t, filepath.Join(videos, "-férias 'a' \"b\": c.mp4"), make([]byte, 3000))
	c, saida := roda(t, "sondar", "--", arq)
	if c != 0 || len(saida) != 1 {
		t.Fatalf("código %d, %v", c, saida)
	}
	querChaves(t, "sondar", saida[0], chavesDaFonte...)
	if saida[0]["arquivo"] != arq || saida[0]["duracao_s"] != 60.0 || saida[0]["tem_audio"] != true || saida[0]["fps_fracao"] != "30/1" {
		t.Errorf("fonte: %v", saida[0])
	}
	querChaves(t, "áudio", saida[0]["audio"].(map[string]any), "codec", "taxa_bps", "canais", "amostragem_hz", "trilha")
	_, saida = roda(t, "sondar", "--quadros-chave", "--", arq)
	querChaves(t, "sondar com quadros-chave", saida[0], append([]string{"quadros_chave"}, chavesDaFonte...)...)
	if chaves := saida[0]["quadros_chave"].([]any); len(chaves) != 2 || chaves[1] != 2.0 {
		t.Errorf("quadros-chave: %v", saida[0]["quadros_chave"])
	}
	querErro(t, 1, "uso:", "sondar")
	querErro(t, 1, "uso:", "sondar", arq, arq)
	querErro(t, 1, "não conheço a opção", "sondar", filepath.Base(arq))
	querErro(t, 1, "não achei o arquivo", "sondar", filepath.Join(videos, "nao.mp4"))
}

func TestMiniaturas_RespondeATiraEConfereAsOpcoes(t *testing.T) {
	_, videos := mundo(t)
	arq := grava(t, filepath.Join(videos, "a.mp4"), make([]byte, 3000))
	c, saida := roda(t, "miniaturas", arq, "--quantas", "3", "--altura", "40")
	if c != 0 {
		t.Fatalf("código %d, %v", c, saida)
	}
	querChaves(t, "miniaturas", saida[0], "arquivo", "duracao_s", "altura", "pasta", "do_cache", "quadros")
	quadros := saida[0]["quadros"].([]any)
	if len(quadros) != 3 || saida[0]["do_cache"] != false || saida[0]["altura"] != 40.0 {
		t.Fatalf("tira: %v", saida[0])
	}
	querChaves(t, "quadro da tira", quadros[0].(map[string]any), "tempo_s", "arquivo")
	if !strings.HasPrefix(quadros[0].(map[string]any)["arquivo"].(string), filepath.Join(os.Getenv("XDG_CACHE_HOME"), "mk", "mkcorte")+"/") {
		t.Errorf("fora do cache: %v", quadros[0])
	}
	_, saida = roda(t, "miniaturas", arq, "--quantas", "3", "--altura", "40")
	if saida[0]["do_cache"] != true {
		t.Errorf("segunda vez: %v", saida[0])
	}
	querErro(t, 1, "uso:", "miniaturas")
	querErro(t, 1, "--quantas pede um número", "miniaturas", arq, "--quantas", "muitas")
	querErro(t, 1, "--altura pede um número", "miniaturas", arq, "--altura", "0")
	querErro(t, 1, "miniaturas demais", "miniaturas", arq, "--quantas", "500")
}

func TestPlano_LeAEdicaoDoArquivoDoTextoOuDoStdinESoPlaneja(t *testing.T) {
	m, videos := mundo(t)
	arq := grava(t, filepath.Join(videos, "passeio na praia.mp4"), make([]byte, 30_000_000))
	texto := edicaoDeUm(t, arq, "")
	doArquivo := grava(t, filepath.Join(videos, "edicao.json"), []byte(texto))

	c, saida := roda(t, "plano", doArquivo)
	if c != 0 {
		t.Fatalf("código %d, %v", c, saida)
	}
	querChaves(t, "plano", saida[0], chavesDoPlano...)
	querChaves(t, "sem_recodificar", saida[0]["sem_recodificar"].(map[string]any), "pode", "motivo", "aviso")
	quer := filepath.Join(videos, "passeio na praia (corte).mp4")
	if saida[0]["saida"] != quer || saida[0]["duracao_s"] != 10.0 || saida[0]["modo"] != "exato" || saida[0]["comando"].([]any)[0] != "ffmpeg" {
		t.Errorf("plano: %v", saida[0])
	}
	if len(m.Chamadas("ffmpeg")) != 0 {
		t.Error("o plano rodou o ffmpeg")
	}
	if _, err := os.Stat(quer); err == nil {
		t.Error("o plano escreveu a saída")
	}

	// Pelo texto (é assim que a janela manda) e pelo stdin.
	_, pelaOpcao := roda(t, "plano", "--json", texto)
	entrada = strings.NewReader(texto)
	defer func() { entrada = os.Stdin }()
	_, peloStdin := roda(t, "plano", "-")
	for _, outra := range [][]map[string]any{pelaOpcao, peloStdin} {
		a, _ := json.Marshal(saida[0])
		b, _ := json.Marshal(outra[0])
		if string(a) != string(b) {
			t.Errorf("as três entradas deviam dar o mesmo plano:\n%s\n%s", a, b)
		}
	}
	// A saída escolhida, e o modo rápido com o aviso franco.
	_, saida = roda(t, "plano", "--json", edicaoDeUm(t, arq, `,"modo":"rapido"`), filepath.Join(videos, "final"))
	if saida[0]["saida"] != filepath.Join(videos, "final.mp4") || saida[0]["modo"] != "rapido" ||
		!strings.Contains(saida[0]["avisos"].([]any)[0].(string), "quadro-chave") {
		t.Errorf("rápido: %v", saida[0])
	}

	querErro(t, 1, "uso:", "plano")
	querErro(t, 1, "uso:", "plano", doArquivo, "a.mp4", "b.mp4")
	querErro(t, 1, "não consegui ler a edição", "plano", filepath.Join(videos, "nao.json"))
	querErro(t, 1, "não é um JSON", "plano", "--json", "{")
	querErro(t, 1, "está vazia", "plano", "--json", `{"fontes":[],"clipes":[]}`)
	querErro(t, 1, "passa do fim", "plano", "--json", strings.Replace(texto, `"fim":12`, `"fim":120`, 1))
	querErro(t, 1, "não achei o arquivo", "plano", "--json", edicaoDeUm(t, filepath.Join(videos, "sumiu.mp4"), ""))
	// O rápido pedido quando não dá: recusa, com o código 2.
	querErro(t, 2, "não dá para exportar sem recodificar", "plano", "--json", edicaoDeUm(t, arq, `,"modo":"rapido","qualidade":"720"`))
}

func TestExportar_SoltaAsFasesEDeixaASaidaAoLado(t *testing.T) {
	m, videos := mundo(t)
	arq := grava(t, filepath.Join(videos, "-passeio 'na' praia.mp4"), make([]byte, 5000))
	c, saida := roda(t, "exportar", "--json", edicaoDeUm(t, arq, `,"qualidade":"720"`))
	if c != 0 || len(saida) != 3 {
		t.Fatalf("código %d, %v", c, saida)
	}
	querChaves(t, "fase plano", saida[0], append([]string{"fase"}, chavesDoPlano...)...)
	querChaves(t, "fase exportando", saida[1], "fase", "pct", "velocidade", "resta_s", "tamanho_bytes")
	querChaves(t, "fase fim", saida[2], "fase", "ok", "saida", "modo", "duracao_s", "tamanho_bytes")
	feita := filepath.Join(videos, "-passeio 'na' praia (corte).mp4")
	if saida[0]["fase"] != "plano" || saida[0]["largura"] != 1280.0 || saida[1]["fase"] != "exportando" || saida[1]["pct"] != 50.0 || saida[1]["resta_s"] != 3.0 ||
		saida[2]["fase"] != "fim" || saida[2]["ok"] != true || saida[2]["saida"] != feita || saida[2]["tamanho_bytes"] != 5.0 {
		t.Errorf("fases: %v", saida)
	}
	if b, _ := os.ReadFile(feita); string(b) != "feito" {
		t.Errorf("a saída tem %q", b)
	}
	if args := m.Chamadas("ffmpeg")[0]; !strings.Contains(strings.Join(args, " "), "scale=1280:720") {
		t.Errorf("ffmpeg: %q", args)
	}
	// De novo: nunca em cima da anterior.
	_, saida = roda(t, "exportar", "--json", edicaoDeUm(t, arq, ""))
	if saida[2]["saida"] != filepath.Join(videos, "-passeio 'na' praia (corte) (2).mp4") {
		t.Errorf("segunda: %v", saida[2])
	}
	querErro(t, 1, "uso:", "exportar")
	querErro(t, 2, "sem recodificar", "exportar", "--json", edicaoDeUm(t, arq, `,"modo":"rapido","qualidade":"720"`))
	ffmpegDeMentira(m, `echo "No space left on device" >&2; exit 1`)
	c, saida = roda(t, "exportar", "--json", edicaoDeUm(t, arq, ""))
	if c != 1 || !strings.Contains(saida[len(saida)-1]["erro"].(string), "No space left") {
		t.Errorf("falha: %d %v", c, saida)
	}
	if nomes, _ := os.ReadDir(videos); len(nomes) != 3 {
		t.Errorf("sobrou parcial: %v", nomes)
	}
}

func TestExportar_SIGTERMCancelaApagaASaidaESaiCom0(t *testing.T) {
	m, videos := mundo(t)
	arq := grava(t, filepath.Join(videos, "a.mp4"), make([]byte, 5000))
	ffmpegDeMentira(m, "exec /bin/sleep 30")
	antes := base.Saida
	b := &saidaSegura{}
	base.Saida = b
	defer func() { base.Saida = antes }()
	feito := make(chan int, 1)
	go func() { feito <- rodar([]string{"exportar", "--json", edicaoDeUm(t, arq, "")}) }()
	prazo := time.Now().Add(10 * time.Second)
	for !strings.Contains(b.String(), `"exportando"`) {
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
	if nomes, _ := os.ReadDir(videos); len(nomes) != 1 {
		t.Errorf("a saída pela metade ficou: %v", nomes)
	}
}

func TestQuadro_SalvaOPNGEConfereOTempo(t *testing.T) {
	_, videos := mundo(t)
	arq := grava(t, filepath.Join(videos, "a b.mp4"), make([]byte, 5000))
	c, saida := roda(t, "quadro", arq, "12.5")
	if c != 0 {
		t.Fatalf("código %d, %v", c, saida)
	}
	querChaves(t, "quadro", saida[0], "ok", "saida", "tempo_s", "tamanho_bytes")
	if saida[0]["saida"] != filepath.Join(videos, "a b (quadro 0-00-12.500).png") || saida[0]["tempo_s"] != 12.5 {
		t.Errorf("quadro: %v", saida[0])
	}
	_, saida = roda(t, "quadro", arq, "0", filepath.Join(videos, "capa"))
	if saida[0]["saida"] != filepath.Join(videos, "capa.png") {
		t.Errorf("com saída: %v", saida[0])
	}
	querErro(t, 1, "uso:", "quadro", arq)
	querErro(t, 1, "uso:", "quadro", arq, "1", "a", "b")
	querErro(t, 1, "número de segundos", "quadro", arq, "já")
	querErro(t, 1, "número de segundos", "quadro", arq, "NaN")
	querErro(t, 1, "o vídeo acaba em", "quadro", arq, "60")
}
