// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// O binário de teste faz três papéis: os testes, o daemon (quando
// MKTUNES_PAPEL=daemon: roda o main de verdade num processo próprio) e o mpv
// de mentira (MKTUNES_PAPEL=mpv). Com MKTUNESD_BIN, o daemon da conversa de
// ponta a ponta é esse binário (o que vai no pacote) em vez do de teste.
func TestMain(m *testing.M) {
	switch os.Getenv("MKTUNES_PAPEL") {
	case "daemon":
		os.Exit(roda(os.Args[1:]))
	case "mpv":
		mpvDeMentira()
		return
	}
	os.Exit(m.Run())
}

// mpvDeMentira abre o socket pedido, anota cada comando recebido e finge
// carregar o que mandarem tocar.
func mpvDeMentira() {
	var socket string
	for _, a := range os.Args[1:] {
		if depois, ok := strings.CutPrefix(a, "--input-ipc-server="); ok {
			socket = depois
		}
	}
	diario, _ := os.OpenFile(os.Getenv("MKTUNES_MPV_DIARIO"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	fmt.Fprintln(diario, "args "+strings.Join(os.Args[1:], " "))
	escuta, erro := net.Listen("unix", socket)
	if erro != nil {
		os.Exit(3)
	}
	conexao, erro := escuta.Accept()
	if erro != nil {
		os.Exit(4)
	}
	linhas := bufio.NewScanner(conexao)
	linhas.Buffer(make([]byte, 64*1024), 4<<20)
	for linhas.Scan() {
		fmt.Fprintln(diario, linhas.Text())
		fmt.Fprintln(conexao, `{"error":"success"}`)
		if strings.Contains(linhas.Text(), `"loadfile"`) {
			fmt.Fprintln(conexao, `{"event":"file-loaded"}`)
			fmt.Fprintln(conexao, `{"event":"property-change","id":2,"name":"duration","data":144.0}`)
			fmt.Fprintln(conexao, `{"event":"property-change","id":1,"name":"time-pos","data":7.5}`)
		}
	}
}

func escreve(t *testing.T, caminho, conteudo string, modo os.FileMode) string {
	t.Helper()
	if erro := os.MkdirAll(filepath.Dir(caminho), 0o755); erro != nil {
		t.Fatal(erro)
	}
	if erro := os.WriteFile(caminho, []byte(conteudo), modo); erro != nil {
		t.Fatal(erro)
	}
	return caminho
}

// mundo é a máquina de mentira em volta do daemon: pastas temporárias, os
// comandos trocados por dublês e a rede por um servidor local.
type mundo struct {
	t      *testing.T
	pasta  string
	socket string
	env    []string
	rede   *httptest.Server
}

func novoMundo(t *testing.T) *mundo {
	t.Helper()
	// O socket Unix tem limite de tamanho no caminho: a pasta fica curta.
	pasta, erro := os.MkdirTemp("", "mkt")
	if erro != nil {
		t.Fatal(erro)
	}
	t.Cleanup(func() { os.RemoveAll(pasta) })
	m := &mundo{t: t, pasta: pasta, socket: filepath.Join(pasta, "run", "mktunes", "mktunes.sock")}
	gravada := func(nome string) []byte {
		bruto, erro := os.ReadFile(filepath.Join("internal", "innertube", "testdata", nome))
		if erro != nil {
			t.Fatal(erro)
		}
		return bruto
	}
	m.rede = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		corpo, _ := io.ReadAll(r.Body)
		switch {
		case strings.HasSuffix(r.URL.Path, "/search") && strings.Contains(string(corpo), "EgWKAQIIAWoMEA4QChADEAQQCRAF"):
			w.Write(gravada("search-songs.DEDUZIDO.json"))
		case strings.HasSuffix(r.URL.Path, "/search") && strings.Contains(string(corpo), "EgWKAQIYAWoMEA4QChADEAQQCRAF"):
			w.Write(gravada("search-albums.json"))
		case strings.HasSuffix(r.URL.Path, "/player"):
			w.Write(gravada("player-vr.json"))
		case strings.HasPrefix(r.URL.Path, "/api/get"):
			w.Write([]byte(`{"duration":144,"syncedLyrics":"[00:01.00] até quando"}`))
		case strings.HasPrefix(r.URL.Path, "/json/stations"):
			w.Write([]byte(`[{"stationuuid":"aaa-1","name":"Rádio Cobre","url_resolved":"https://stream.example/cobre.mp3","countrycode":"BR"}]`))
		default:
			w.Write([]byte("{}"))
		}
	}))
	t.Cleanup(m.rede.Close)

	eu, _ := os.Executable()
	mpv := escreve(t, filepath.Join(pasta, "bin", "mpv"), "#!/bin/sh\nMKTUNES_PAPEL=mpv exec '"+eu+"' \"$@\"\n", 0o755)
	ytdlp := escreve(t, filepath.Join(pasta, "bin", "yt-dlp"), "#!/bin/sh\nprintf '%s\\n' \"$*\" >>'"+pasta+"/ytdlp.diario'\necho 'https://rr1---sn-teste.googlevideo.com/videoplayback?expire=9999999999&itag=140'\n", 0o755)
	sqlite := escreve(t, filepath.Join(pasta, "bin", "sqlite3"), `#!/bin/sh
case "$4" in
  *"FROM liked_track"*) echo '[{"track_id":"antiga00001","provider":"YOUTUBE","title":"Do banco antigo","artist":"Alguém","duration_ms":1000,"artwork_url":null,"stream_url":null,"liked_at":5}]' ;;
esac
`, 0o755)
	m.env = append(os.Environ(),
		"HOME="+pasta, "XDG_RUNTIME_DIR="+pasta+"/run", "XDG_DATA_HOME="+pasta+"/dados", "XDG_CACHE_HOME="+pasta+"/cache",
		"XDG_STATE_HOME="+pasta+"/estado", "XDG_CONFIG_HOME="+pasta+"/config", "LANG=pt_BR.UTF-8", "LC_ALL=",
		"MKTUNES_MUSIC_DIR="+pasta+"/Música/MkTunes", "MKTUNES_MPV="+mpv, "MKTUNES_MPV_ARGS=--ao=null", "MKTUNES_MPV_DIARIO="+pasta+"/mpv.diario",
		"MKTUNES_YTDLP="+ytdlp, "MKTUNES_SQLITE3="+sqlite, "MKTUNES_FFPROBE="+pasta+"/bin/ffprobe-que-falta",
		"MKTUNES_INNERTUBE_URL="+m.rede.URL+"/youtubei/v1/", "MKTUNES_LRCLIB_URL="+m.rede.URL, "MKTUNES_RADIO_URL="+m.rede.URL,
		"MKTUNES_LAUNCHER="+pasta+"/bin/mktunes-que-falta", "DBUS_SESSION_BUS_ADDRESS=", "MKTUNES_PAPEL=daemon",
	)
	// O banco da versão antiga, para a migração da primeira subida.
	escreve(t, filepath.Join(pasta, "dados", "mktunes", "mktunes.db"), "SQLite format 3\x00", 0o600)
	return m
}

func (m *mundo) comando(argumentos ...string) *exec.Cmd {
	binario := os.Getenv("MKTUNESD_BIN")
	if binario == "" {
		binario, _ = os.Executable()
	}
	comando := exec.Command(binario, argumentos...)
	comando.Env = m.env
	comando.Stderr = os.Stderr
	return comando
}

// sobe o daemon e espera o socket responder.
func (m *mundo) sobe() *exec.Cmd {
	m.t.Helper()
	daemon := m.comando()
	if erro := daemon.Start(); erro != nil {
		m.t.Fatal(erro)
	}
	m.t.Cleanup(func() { _ = daemon.Process.Kill() })
	for range 100 {
		if conexao, erro := net.Dial("unix", m.socket); erro == nil {
			conexao.Close()
			return daemon
		}
		time.Sleep(50 * time.Millisecond)
	}
	m.t.Fatal("o daemon não abriu o socket")
	return nil
}

// janela é uma janela de mentira: conecta no socket e lembra o último estado
// de cada área, como a Daemon.qml.
type janela struct {
	t        *testing.T
	conexao  net.Conn
	chegadas chan map[string]any
	estados  map[string]map[string]any
	avisos   []string
}

func (m *mundo) conecta() *janela {
	m.t.Helper()
	conexao, erro := net.Dial("unix", m.socket)
	if erro != nil {
		m.t.Fatal(erro)
	}
	m.t.Cleanup(func() { conexao.Close() })
	j := &janela{t: m.t, conexao: conexao, chegadas: make(chan map[string]any, 256), estados: map[string]map[string]any{}}
	go func() {
		linhas := bufio.NewScanner(conexao)
		linhas.Buffer(make([]byte, 64*1024), 16<<20)
		for linhas.Scan() {
			var mensagem map[string]any
			if json.Unmarshal(linhas.Bytes(), &mensagem) == nil {
				j.chegadas <- mensagem
			}
		}
		close(j.chegadas)
	}()
	return j
}

func (j *janela) manda(comando string) {
	j.t.Helper()
	if _, erro := j.conexao.Write([]byte(comando + "\n")); erro != nil {
		j.t.Fatal(erro)
	}
}

// ate lê mensagens até a condição valer sobre o que a janela já sabe.
func (j *janela) ate(oQue string, vale func() bool) {
	j.t.Helper()
	prazo := time.After(15 * time.Second)
	for !vale() {
		select {
		case mensagem, aberta := <-j.chegadas:
			if !aberta {
				j.t.Fatalf("o daemon fechou a conexão esperando: %s", oQue)
			}
			tipo, _ := mensagem["type"].(string)
			if tipo == "notice" {
				j.avisos = append(j.avisos, mensagem["text"].(string))
			} else {
				j.estados[tipo] = mensagem
			}
		case <-prazo:
			j.t.Fatalf("não chegou: %s\n  estados: %v\n  avisos: %v", oQue, j.estados["now"], j.avisos)
		}
	}
}

func (j *janela) campo(tipo, chave string) any { return j.estados[tipo][chave] }

func TestConversaDePontaAPonta(t *testing.T) {
	m := novoMundo(t)
	daemon := m.sobe()
	j := m.conecta()

	// Ao conectar, a janela recebe o estado de todas as áreas.
	areas := []string{"now", "progress", "search", "home", "library", "downloads", "radio", "lyrics", "settings", "volume", "local"}
	j.ate("os onze estados", func() bool { return len(j.estados) == len(areas) })
	for _, area := range areas {
		if j.estados[area] == nil {
			t.Fatalf("faltou o estado %q", area)
		}
	}
	if j.campo("now", "status") != "idle" || j.campo("settings", "version") != Versao || j.campo("volume", "level") != 100.0 {
		t.Fatalf("estado inicial: %v %v %v", j.estados["now"], j.estados["settings"], j.estados["volume"])
	}
	if pasta := j.campo("downloads", "folder"); pasta != m.pasta+"/Música/MkTunes" {
		t.Fatalf("pasta de downloads: %v", pasta)
	}
	// A biblioteca do banco antigo veio na primeira subida, e a janela é avisada.
	curtidas := j.campo("library", "liked").([]any)
	if len(curtidas) != 1 || curtidas[0].(map[string]any)["title"] != "Do banco antigo" {
		t.Fatalf("curtidas migradas: %v", curtidas)
	}
	j.manda(`{"cmd":"home"}`)
	j.ate("o aviso da migração", func() bool { return len(j.avisos) > 0 })
	if !strings.Contains(j.avisos[0], "Biblioteca trazida da versão anterior: 1 curtidas") {
		t.Fatalf("aviso: %v", j.avisos)
	}
	if _, erro := os.Stat(filepath.Join(m.pasta, "dados", "mktunes", "mktunes.db.migrado")); erro != nil {
		t.Fatal("o banco antigo devia ter virado mktunes.db.migrado")
	}

	// search: as músicas e os álbuns vêm do InnerTube (de mentira).
	j.manda(`{"cmd":"search","query":"ate quando"}`)
	j.ate("o resultado da busca", func() bool { return j.campo("search", "status") == "results" })
	faixas := j.campo("search", "tracks").([]any)
	if len(faixas) != 2 || faixas[0].(map[string]any)["title"] != "Até Quando" || j.campo("search", "top").(map[string]any)["kind"] != "song" {
		t.Fatalf("busca: %v", j.estados["search"])
	}
	if secoes := j.campo("search", "sections").([]any); len(secoes) != 1 || secoes[0].(map[string]any)["title"] != "Álbuns" {
		t.Fatalf("seções: %v", secoes)
	}

	// play: a faixa volta inteira; o yt-dlp resolve o áudio e o mpv o carrega.
	lista, _ := json.Marshal(faixas)
	j.manda(`{"cmd":"play","tracks":` + string(lista) + `,"index":0}`)
	j.ate("a música tocando", func() bool { return j.campo("now", "status") == "playing" })
	if j.campo("now", "title") != "Até Quando" || len(j.campo("now", "upNext").([]any)) != 1 {
		t.Fatalf("now: %v", j.estados["now"])
	}
	j.ate("a letra sincronizada", func() bool { return j.campo("lyrics", "status") == "synced" })
	j.ate("a posição que o mpv informa", func() bool { return j.campo("progress", "position") == 7.5 && j.campo("progress", "duration") == 144.0 })
	chamada, _ := os.ReadFile(filepath.Join(m.pasta, "ytdlp.diario"))
	if !strings.Contains(string(chamada), "-g --no-playlist --no-warnings -- https://music.youtube.com/watch?v=W40W7L1fhUM") {
		t.Fatalf("yt-dlp chamado com: %s", chamada)
	}
	diario := func() string {
		bruto, _ := os.ReadFile(filepath.Join(m.pasta, "mpv.diario"))
		return string(bruto)
	}
	for _, trecho := range []string{"--config=no --load-scripts=no --idle=yes --no-video", "--ao=null", `["loadfile","https://rr1---sn-teste.googlevideo.com/videoplayback?expire=9999999999&itag=140","replace"]`,
		// O volume vem normalizado pelo loudness que o InnerTube informou (6,68 dB → 46 %).
		`["set_property","volume",46]`, `["set_property","force-media-title","Natanzinho Lima e Ronaldinho Gaúcho — Até Quando"]`} {
		if !strings.Contains(diario(), trecho) {
			t.Fatalf("o mpv não recebeu %s:\n%s", trecho, diario())
		}
	}

	// toggle e like.
	j.manda(`{"cmd":"toggle"}`)
	j.ate("a música pausada", func() bool { return j.campo("now", "status") == "paused" })
	faixa, _ := json.Marshal(faixas[0])
	j.manda(`{"cmd":"like","track":` + string(faixa) + `}`)
	j.ate("a curtida", func() bool { return j.campo("now", "liked") == true && len(j.campo("library", "liked").([]any)) == 2 })
	j.ate("o aviso da curtida", func() bool { return j.avisos[len(j.avisos)-1] == "Curtida" })
	j.manda(`{"cmd":"radio"}`)
	j.ate("as rádios", func() bool { return j.campo("radio", "status") == "ready" })
	j.manda(`{"cmd":"volume","level":30}`)
	j.ate("o volume", func() bool { return j.campo("volume", "level") == 30.0 })

	// O motivo do porte: o daemon cabe em pouca memória. (Só o binário do pacote é
	// medido: o de teste carrega o detector de corridas.)
	if estado, erro := os.ReadFile(fmt.Sprintf("/proc/%d/status", daemon.Process.Pid)); erro == nil {
		for _, linha := range strings.Split(string(estado), "\n") {
			var kb int
			if n, _ := fmt.Sscanf(linha, "VmRSS: %d kB", &kb); n == 1 {
				t.Logf("memória do daemon tocando: %d MB", kb/1024)
				if os.Getenv("MKTUNESD_BIN") != "" && kb > 48*1024 {
					t.Fatalf("o daemon está usando %d MB", kb/1024)
				}
			}
		}
	}

	// --ping e --versao: é assim que o lançador sabe quem responde no socket.
	saida, erro := m.comando("--ping").Output()
	if erro != nil || strings.TrimSpace(string(saida)) != Versao {
		t.Fatalf("--ping: %q %v", saida, erro)
	}
	saida, _ = m.comando("--versao").Output()
	if strings.TrimSpace(string(saida)) != Versao {
		t.Fatalf("--versao: %q", saida)
	}
	// Um segundo daemon vê que já há um e sai sem atrapalhar.
	if erro := m.comando().Run(); erro != nil {
		t.Fatalf("o segundo daemon devia sair com 0: %v", erro)
	}

	// --encerra: o daemon guarda a sessão, solta o socket e sai com 0.
	if erro := m.comando("--encerra").Run(); erro != nil {
		t.Fatalf("--encerra: %v", erro)
	}
	if erro := daemon.Wait(); erro != nil {
		t.Fatalf("o daemon devia sair com 0: %v", erro)
	}
	if erro := m.comando("--ping").Run(); erro == nil {
		t.Fatal("--ping devia falhar sem daemon")
	}
	if _, erro := os.Lstat(m.socket); erro == nil {
		t.Fatal("o socket devia ter sido apagado")
	}
	sessao, _ := os.ReadFile(filepath.Join(m.pasta, "dados", "mktunes", "sessao.json"))
	if !strings.Contains(string(sessao), `"W40W7L1fhUM"`) || !strings.Contains(string(sessao), `"positionMs":7500`) {
		t.Fatalf("sessão guardada: %s", sessao)
	}

	// A fila, a posição, as curtidas e o volume voltam na subida seguinte; e o
	// SIGTERM também encerra direito.
	daemon = m.sobe()
	j = m.conecta()
	j.ate("a sessão restaurada", func() bool { return len(j.estados) == len(areas) })
	if j.campo("now", "status") != "paused" || j.campo("now", "title") != "Até Quando" || j.campo("progress", "position") != 7.5 || j.campo("volume", "level") != 30.0 || len(j.campo("library", "liked").([]any)) != 2 {
		t.Fatalf("depois de reiniciar: %v %v %v", j.estados["now"], j.estados["progress"], j.estados["volume"])
	}
	if len(j.avisos) != 0 {
		t.Fatalf("a migração não se repete: %v", j.avisos)
	}
	_ = daemon.Process.Signal(syscall.SIGTERM)
	if erro := daemon.Wait(); erro != nil {
		t.Fatalf("o SIGTERM devia encerrar com 0: %v", erro)
	}
}

func TestSemMpvODaemonNaoSobe(t *testing.T) {
	m := novoMundo(t)
	for i, variavel := range m.env {
		if strings.HasPrefix(variavel, "MKTUNES_MPV=") {
			m.env[i] = "MKTUNES_MPV=" + m.pasta + "/bin/mpv-que-falta"
		}
	}
	daemon := m.comando()
	daemon.Stderr = nil
	saida, erro := daemon.CombinedOutput()
	if erro == nil || !strings.Contains(string(saida), "não deu para subir o mpv") {
		t.Fatalf("saída: %s (%v)", saida, erro)
	}
}

func TestPastasDoUsuarioERegiao(t *testing.T) {
	pasta := t.TempDir()
	t.Setenv("HOME", pasta)
	t.Setenv("XDG_CONFIG_HOME", pasta+"/config")
	if achada := pastaDoUsuario("MUSIC"); achada != "" {
		t.Fatalf("sem user-dirs.dirs: %q", achada)
	}
	escreve(t, pasta+"/config/user-dirs.dirs", "# comentário\nXDG_DOCUMENTS_DIR=\"$HOME/Documentos\"\nXDG_MUSIC_DIR=\"$HOME/Música\"\nXDG_VIDEOS_DIR=\"$HOME/\"\nXDG_DESKTOP_DIR=\"relativa\"\n", 0o600)
	for nome, esperada := range map[string]string{"MUSIC": pasta + "/Música", "DOCUMENTS": pasta + "/Documentos", "VIDEOS": "", "DESKTOP": "", "PICTURES": ""} {
		if achada := pastaDoUsuario(nome); achada != esperada {
			t.Fatalf("%s: esperada %q, achada %q", nome, esperada, achada)
		}
	}
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_MESSAGES", "")
	t.Setenv("LANG", "pt_PT.UTF-8")
	if regiao() != "PT" {
		t.Fatalf("região: %s", regiao())
	}
	t.Setenv("LANG", "C")
	if regiao() != "BR" {
		t.Fatalf("região padrão: %s", regiao())
	}
	t.Setenv("XDG_RUNTIME_DIR", "/run/user/1000")
	if pastaDeExecucao() != "/run/user/1000/mktunes" {
		t.Fatalf("pasta de execução: %s", pastaDeExecucao())
	}
	if codigo := roda([]string{"--inventada"}); codigo != 2 {
		t.Fatalf("opção desconhecida: %d", codigo)
	}
}
