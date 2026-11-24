package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"syscall"
	"testing"
	"time"

	"mkpulso/internal/base"
	"mkpulso/internal/coleta"
	"mkpulso/internal/coleta/falso"
)

// --- o contrato ---

// exemploDoContrato tira de ../CONTRATO.md o bloco de JSON que vem depois do
// título de um subcomando. É a especificação: o teste confere a saída contra
// ele, campo por campo.
func exemploDoContrato(t *testing.T, subcomando string) any {
	t.Helper()
	b, err := os.ReadFile("../CONTRATO.md")
	if err != nil {
		t.Fatal(err)
	}
	texto := string(b)
	titulo := strings.Index(texto, "## `mkpulso-coletor "+subcomando)
	if titulo < 0 {
		t.Fatalf("o CONTRATO.md não tem o subcomando %q", subcomando)
	}
	texto = texto[titulo:]
	abre := strings.Index(texto, "```json\n")
	if abre < 0 {
		t.Fatalf("o subcomando %q não tem exemplo em JSON", subcomando)
	}
	texto = texto[abre+len("```json\n"):]
	fecha := strings.Index(texto, "```")
	var exemplo any
	if err := json.Unmarshal([]byte(texto[:fecha]), &exemplo); err != nil {
		t.Fatalf("o exemplo de %q no CONTRATO.md não é JSON: %v", subcomando, err)
	}
	return exemplo
}

func tipoJSON(v any) string {
	switch v.(type) {
	case nil:
		return "null"
	case float64:
		return "número"
	case string:
		return "texto"
	case bool:
		return "booleano"
	case []any:
		return "lista"
	case map[string]any:
		return "objeto"
	}
	return fmt.Sprintf("%T", v)
}

// confereForma compara a saída com o exemplo do contrato: os mesmos campos
// (nem a mais, nem a menos), os mesmos tipos. Um campo só pode ser null se o
// caminho dele está em podeNull; uma lista só pode vir vazia se está em
// podeVazia. Onde o próprio exemplo traz null (gpu_pct), vale null ou número.
func confereForma(t *testing.T, caminho string, exemplo, veio any, podeNull, podeVazia map[string]bool) {
	t.Helper()
	if veio == nil {
		if exemplo != nil && !podeNull[caminho] {
			t.Errorf("%s: veio null, o contrato pede %s", caminho, tipoJSON(exemplo))
		}
		return
	}
	switch ex := exemplo.(type) {
	case nil:
		if tipoJSON(veio) != "número" {
			t.Errorf("%s: o contrato pede null ou número, veio %s", caminho, tipoJSON(veio))
		}
	case map[string]any:
		obj, ok := veio.(map[string]any)
		if !ok {
			t.Errorf("%s: o contrato pede objeto, veio %s", caminho, tipoJSON(veio))
			return
		}
		for chave := range ex {
			if _, tem := obj[chave]; !tem {
				t.Errorf("%s: falta o campo %q do contrato", caminho, chave)
			}
		}
		for chave := range obj {
			if _, tem := ex[chave]; !tem {
				t.Errorf("%s: o campo %q não está no contrato", caminho, chave)
				continue
			}
			confereForma(t, caminho+"."+chave, ex[chave], obj[chave], podeNull, podeVazia)
		}
	case []any:
		lista, ok := veio.([]any)
		if !ok {
			t.Errorf("%s: o contrato pede lista, veio %s", caminho, tipoJSON(veio))
			return
		}
		if len(lista) == 0 && !podeVazia[caminho] {
			t.Errorf("%s: a lista veio vazia e o teste não conferiu os itens", caminho)
		}
		for _, item := range lista {
			confereForma(t, caminho+"[]", ex[0], item, podeNull, podeVazia)
		}
	default:
		if tipoJSON(exemplo) != tipoJSON(veio) {
			t.Errorf("%s: o contrato pede %s, veio %s (%v)", caminho, tipoJSON(exemplo), tipoJSON(veio), veio)
		}
	}
}

// --- a máquina de mentira ---

const cgFirefox = "/user.slice/user-1000.slice/user@1000.service/app.slice/app-firefox-1234.scope"

func fdinfoDeGPU(ns uint64) string {
	return fmt.Sprintf("pos:\t0\ndrm-driver:\tamdgpu\ndrm-client-id:\t7\ndrm-engine-gfx:\t%d ns\n", ns)
}

// maquinaDeMentira monta um notebook com um pouco de tudo o que o retrato
// mostra, no primeiro instante. passaUmSegundo anda os contadores.
func maquinaDeMentira(t *testing.T) string {
	t.Helper()
	r := falso.Raizes(t)
	e := func(caminho, texto string) { falso.Escreve(t, r, caminho, texto) }
	e("etc/passwd", "root:x:0:0::/root:/bin/bash\nmark:x:1000:1000::/home/mark:/bin/bash\n")
	e("proc/sys/kernel/hostname", "nome-da-maquina\n")
	falso.Stat(t, r, [2]uint64{1000, 9000}, [2]uint64{1000, 9000})
	e("proc/meminfo", "MemTotal: 16000000 kB\nMemAvailable: 7000000 kB\nCached: 3000000 kB\nSwapTotal: 0 kB\nSwapFree: 0 kB\n")
	e("proc/loadavg", "1.20 0.90 0.70 2/1500 999\n")
	e("proc/diskstats", " 259 0 nvme0n1 5000 10 20000 900 3000 20 40000 800 0 500 1700\n 259 1 nvme0n1p1 1 0 8 1 0 0 0 0 0 1 1\n")
	e("proc/self/mounts", "proc /proc proc rw 0 0\n/dev/nvme0n1p2 / btrfs rw 0 0\n")
	e("proc/net/dev", "Inter-|   Receive  |  Transmit\n face |bytes packets|bytes packets\n    lo: 1 1 0 0 0 0 0 0 1 1 0 0 0 0 0 0\n wlan0: 10000 1 0 0 0 0 0 0 2000 1 0 0 0 0 0 0\n")
	e("sys/devices/system/cpu/cpu0/cpufreq/scaling_cur_freq", "2400000\n")
	e("sys/class/hwmon/hwmon1/name", "coretemp\n")
	e("sys/class/hwmon/hwmon1/temp1_input", "61000\n")
	g := "sys/class/drm/card0/device/"
	e(g+"uevent", "DRIVER=amdgpu\nPCI_ID=1002:731F\n")
	e(g+"gpu_busy_percent", "12\n")
	e(g+"mem_info_vram_used", "819200000\n")
	e(g+"mem_info_vram_total", "8192000000\n")
	e(g+"hwmon/hwmon2/temp1_input", "55000\n")
	e("pci.ids", "1002  AMD\n\t731f  Navi 10 [Radeon RX 5700]\n")
	e("sys/class/power_supply/BAT0/capacity", "75\n")
	e("sys/class/power_supply/BAT0/status", "Discharging\n")
	e("sys/class/power_supply/BAT0/power_now", "9400000\n")
	e("apps/firefox.desktop", "[Desktop Entry]\nName=Firefox\nIcon=firefox\n")

	falso.Proc{PID: 1, PPID: 0, Comm: "systemd", Cmd: []string{"/sbin/init"}, Inicio: 1, RSS: 12000, Threads: 1, Cgroup: "/init.scope"}.Grava(t, r)
	falso.Proc{PID: 2, PPID: 0, Comm: "kthreadd", Inicio: 2}.Grava(t, r)
	falso.Proc{PID: 300, PPID: 1, Comm: "sshd", Cmd: []string{"/usr/bin/sshd", "-D"}, Inicio: 300, RSS: 5400, Cgroup: "/system.slice/sshd.service", SemIO: true}.Grava(t, r)
	falso.Proc{PID: 900, PPID: 1, Comm: "systemd", Cmd: []string{"/usr/lib/systemd/systemd", "--user"}, UID: 1000, Inicio: 900, RSS: 9000, Cgroup: "/user.slice/user-1000.slice/user@1000.service/init.scope"}.Grava(t, r)
	falso.Proc{PID: 1234, PPID: 900, Comm: "firefox", Cmd: []string{"/usr/lib/firefox/firefox"}, UID: 1000, Utime: 500, Stime: 100, Threads: 80, Inicio: 7000, RSS: 400000, Cgroup: cgFirefox, Leu: 1000, Gravou: 1000}.Grava(t, r)
	falso.Proc{PID: 1240, PPID: 1234, Comm: "Web Content", Cmd: []string{"/usr/lib/firefox/firefox", "-contentproc"}, UID: 1000, Utime: 100, Threads: 20, Inicio: 7010, RSS: 439000, Cgroup: cgFirefox}.Grava(t, r)
	e("proc/1234/fdinfo/9", fdinfoDeGPU(1_000_000_000))

	falso.Comando(t, r, "hyprctl", `[ "$1" = clients ] || exit 1
echo '[{"address": "0x55d0", "mapped": true, "workspace": {"id": 2, "name": "2"}, "class": "firefox", "title": "Mk — Firefox", "pid": 1234}]'`)

	velho := coleta.Statfs
	t.Cleanup(func() { coleta.Statfs = velho })
	coleta.Statfs = func(onde string) (uint64, uint64, error) {
		if onde == "/" {
			return 500000000, 120000000, nil
		}
		return 0, 0, errors.New("não é da máquina de mentira")
	}
	return r
}

// passaUmSegundo anda os contadores da máquina de mentira: o firefox gasta
// 30 % de um núcleo, o filho 5 %, lê e grava 4096 bytes, usa 10 % da GPU; o
// disco grava 8192 bytes; a rede recebe 1400 e manda 300.
func passaUmSegundo(t *testing.T, r string) {
	t.Helper()
	falso.Stat(t, r, [2]uint64{1050, 9050}, [2]uint64{1080, 9020})
	falso.Escreve(t, r, "proc/diskstats", " 259 0 nvme0n1 5000 10 20000 900 3000 20 40016 800 0 500 1700\n")
	falso.Escreve(t, r, "proc/net/dev", " wlan0: 11400 1 0 0 0 0 0 0 2300 1 0 0 0 0 0 0\n")
	falso.Proc{PID: 1234, PPID: 900, Comm: "firefox", Cmd: []string{"/usr/lib/firefox/firefox"}, UID: 1000, Utime: 520, Stime: 110, Threads: 80, Inicio: 7000, RSS: 400000, Cgroup: cgFirefox, Leu: 3048, Gravou: 3048}.Grava(t, r)
	falso.Proc{PID: 1240, PPID: 1234, Comm: "Web Content", Cmd: []string{"/usr/lib/firefox/firefox", "-contentproc"}, UID: 1000, Utime: 105, Threads: 20, Inicio: 7010, RSS: 439000, Cgroup: cgFirefox}.Grava(t, r)
	falso.Escreve(t, r, "proc/1234/fdinfo/9", fdinfoDeGPU(1_100_000_000))
}

// relogioDeMentira troca o relógio do watch: cada espera anda o tempo sem
// dormir e chama naEspera (que mexe na máquina de mentira). Depois de
// `voltas` esperas, a próxima manda parar, como um SIGTERM.
func relogioDeMentira(t *testing.T, voltas int, naEspera func(volta int)) *[]time.Duration {
	t.Helper()
	velho := watchRel
	t.Cleanup(func() { watchRel = velho })
	agora := 5 * time.Hour
	esperas := &[]time.Duration{}
	watchRel = watchRelogio{
		Mono:   func() time.Duration { return agora },
		Parede: func() time.Time { return time.UnixMilli(1795480000000) },
		Espera: func(d time.Duration, para <-chan struct{}) bool {
			if len(*esperas) >= voltas {
				return false
			}
			*esperas = append(*esperas, d)
			agora += d
			if naEspera != nil {
				naEspera(len(*esperas))
			}
			return true
		},
	}
	return esperas
}

// rodaComSaida roda um subcomando pelo main e devolve o código e o stdout.
func rodaComSaida(t *testing.T, args ...string) (int, string) {
	t.Helper()
	var saida bytes.Buffer
	velha := base.Saida
	base.Saida = &saida
	defer func() { base.Saida = velha }()
	codigo := rodar(args)
	return codigo, saida.String()
}

func umaLinhaDeJSON(t *testing.T, saida string) map[string]any {
	t.Helper()
	if !strings.HasSuffix(saida, "\n") || strings.Count(saida, "\n") != 1 {
		t.Fatalf("esperava uma linha só de JSON, veio %q", saida)
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(saida), &obj); err != nil {
		t.Fatalf("a saída não é JSON: %v\n%s", err, saida)
	}
	return obj
}

// cava desce por um caminho ("grupos", 0, "processos", 1, "cpu_pct").
func cava(t *testing.T, v any, caminho ...any) any {
	t.Helper()
	for _, passo := range caminho {
		switch p := passo.(type) {
		case string:
			obj, ok := v.(map[string]any)
			if !ok {
				t.Fatalf("cava %v: %q não é objeto", caminho, p)
			}
			v = obj[p]
		case int:
			lista, ok := v.([]any)
			if !ok || p >= len(lista) {
				t.Fatalf("cava %v: não há item %d", caminho, p)
			}
			v = lista[p]
		}
	}
	return v
}

func grupoPorID(t *testing.T, retrato map[string]any, id string) map[string]any {
	t.Helper()
	for _, g := range retrato["grupos"].([]any) {
		if g.(map[string]any)["id"] == id {
			return g.(map[string]any)
		}
	}
	t.Fatalf("o retrato não tem o grupo %s", id)
	return nil
}

// --- os testes ---

func TestWatchUmaVezSegueOContrato(t *testing.T) {
	r := maquinaDeMentira(t)
	esperas := relogioDeMentira(t, 9, func(int) { passaUmSegundo(t, r) })
	codigo, saida := rodaComSaida(t, "watch", "--uma-vez")
	if codigo != base.Certo {
		t.Fatalf("código %d, saída %s", codigo, saida)
	}
	retrato := umaLinhaDeJSON(t, saida)
	if len(*esperas) != 1 || (*esperas)[0] != time.Second {
		t.Errorf("--uma-vez mede duas vezes com um intervalo (de 1 s) no meio: esperas = %v", *esperas)
	}
	// só o que o contrato deixa ser null: a GPU e o disco de um processo
	// (o disco de quem é de outro usuário, como o sshd daqui)
	podeNull := map[string]bool{
		"retrato.grupos[].gpu_pct": true, "retrato.grupos[].processos[].gpu_pct": true,
		"retrato.grupos[].disco_bps": true, "retrato.grupos[].processos[].disco_bps": true,
	}
	podeVazia := map[string]bool{"retrato.grupos[].janelas": true}
	confereForma(t, "retrato", exemploDoContrato(t, "watch"), retrato, podeNull, podeVazia)

	// e os valores do exemplo do contrato, que a máquina de mentira imita
	for caminho, quer := range map[string]any{
		"t_ms": 1795480000000.0, "maquina": "nome-da-maquina", "nucleos": 2.0,
	} {
		if retrato[caminho] != quer {
			t.Errorf("%s = %v, esperava %v", caminho, retrato[caminho], quer)
		}
	}
	confere := func(quer string, caminho ...any) {
		t.Helper()
		b, _ := json.Marshal(cava(t, retrato, caminho...))
		if string(b) != quer {
			t.Errorf("%v:\n veio     %s\n esperava %s", caminho, b, quer)
		}
	}
	confere(`{"carga":[1.2,0.9,0.7],"freq_mhz":2400,"pct":65,"por_nucleo":[50,80],"temp_c":61}`, "cpu")
	confere(`{"cache_kb":3000000,"disponivel_kb":7000000,"swap_total_kb":0,"swap_usado_kb":0,"total_kb":16000000,"usado_kb":9000000}`, "mem")
	confere(`[{"escrita_bps":8192,"leitura_bps":0,"nome":"nvme0n1"}]`, "discos")
	confere(`[{"fs":"btrfs","onde":"/","total_kb":500000000,"usado_kb":120000000}]`, "montagens")
	confere(`[{"nome":"wlan0","rx_bps":1400,"tx_bps":300}]`, "rede")
	confere(`[{"mem_total_kb":8000000,"mem_usada_kb":800000,"nome":"AMD Radeon RX 5700","temp_c":55,"uso_pct":12}]`, "gpus")
	confere(`{"carregando":false,"pct":75,"watts":9.4}`, "bateria")
	confere(`{"grupos":5,"processos":6,"threads":104}`, "totais")

	f := cava(t, retrato, "grupos", 0).(map[string]any)
	processos := f["processos"]
	delete(f, "processos")
	b, _ := json.Marshal(f)
	quer := `{"cpu_pct":35,"disco_bps":4096,"energia":"moderada","escopo":"usuario","gpu_pct":10,"icone":"firefox","id":"app:firefox",` +
		`"janelas":[{"endereco":"0x55d0","titulo":"Mk — Firefox","workspace":"2"}],` +
		`"media":{"cpu_pct_1m":35,"cpu_pct_3m":35,"disco_bps_1m":4096,"disco_bps_3m":4096,"mem_kb_1m":839000,"mem_kb_3m":839000},` +
		`"mem_kb":839000,"n":2,"nome":"Firefox","pid":1234,"tipo":"app","unidade":"app-firefox-1234.scope"}`
	if string(b) != quer {
		t.Errorf("o grupo que mais gasta:\n veio     %s\n esperava %s", b, quer)
	}
	b, _ = json.Marshal(cava(t, processos, 0))
	quer = `{"cmd":"/usr/lib/firefox/firefox","cpu_pct":30,"disco_bps":4096,"estado":"dormindo","gpu_pct":10,"inicio_s":1795000070,"mem_kb":400000,"nice":0,"nome":"firefox","pid":1234,"ppid":900,"threads":80,"usuario":"mark"}`
	if string(b) != quer {
		t.Errorf("o processo que mais gasta:\n veio     %s\n esperava %s", b, quer)
	}
	if s := grupoPorID(t, retrato, "servico:sshd.service"); s["disco_bps"] != nil || cava(t, s, "processos", 0, "disco_bps") != nil {
		t.Errorf("io ilegível: disco null no processo e no grupo: %v", s)
	}
	if k := grupoPorID(t, retrato, "sistema:kernel"); k["tipo"] != "sistema" || k["escopo"] != "" {
		t.Errorf("kernel = %v", k)
	}
}

func TestWatchNumaMaquinaQueNaoDizNada(t *testing.T) {
	falso.Raizes(t)
	relogioDeMentira(t, 9, nil)
	codigo, saida := rodaComSaida(t, "watch", "--uma-vez")
	if codigo != base.Certo {
		t.Fatalf("código %d, saída %s", codigo, saida)
	}
	retrato := umaLinhaDeJSON(t, saida)
	// todos os campos de primeiro nível do contrato continuam lá
	exemplo := exemploDoContrato(t, "watch").(map[string]any)
	for chave := range exemplo {
		if _, tem := retrato[chave]; !tem {
			t.Errorf("falta o campo %q", chave)
		}
	}
	delete(retrato, "t_ms")
	if n, ok := retrato["nucleos"].(float64); !ok || n < 1 {
		t.Errorf("núcleos nunca é zero: %v", retrato["nucleos"])
	}
	delete(retrato, "nucleos")
	b, _ := json.Marshal(retrato)
	quer := `{"bateria":null,"cpu":{"carga":null,"freq_mhz":null,"pct":null,"por_nucleo":[],"temp_c":null},"discos":[],"gpus":[],"grupos":[],"maquina":"","mem":null,"montagens":[],"rede":[],"totais":{"grupos":0,"processos":0,"threads":0}}`
	if string(b) != quer {
		t.Errorf("o que não dá para ler é null ou lista vazia:\n veio     %s\n esperava %s", b, quer)
	}
}

func TestWatchOpcoes(t *testing.T) {
	bons := []struct {
		args      []string
		intervalo time.Duration
		umaVez    bool
	}{
		{nil, time.Second, false},
		{[]string{"--uma-vez"}, time.Second, true},
		{[]string{"--intervalo", "500"}, 500 * time.Millisecond, false},
		{[]string{"--intervalo=2000", "--uma-vez"}, 2 * time.Second, true},
		{[]string{"--uma-vez", "--intervalo", "250"}, 250 * time.Millisecond, true},
		{[]string{"--intervalo", "249"}, 250 * time.Millisecond, false},
		{[]string{"--intervalo", "1"}, 250 * time.Millisecond, false},
		{[]string{"--intervalo", "0"}, 250 * time.Millisecond, false},
		{[]string{"--intervalo", "99999999999"}, time.Hour, false},
	}
	for _, c := range bons {
		intervalo, umaVez, err := watchOpcoes(c.args)
		if err != nil || intervalo != c.intervalo || umaVez != c.umaVez {
			t.Errorf("%v: veio %v %v %v; esperava %v %v", c.args, intervalo, umaVez, err, c.intervalo, c.umaVez)
		}
	}
	for _, ruins := range [][]string{{"--intervalo"}, {"--intervalo", "abc"}, {"--intervalo", "-5"}, {"--intervalo", "1.5"}, {"--intervalo="}, {"--outra"}, {"solto"}, {"--uma-vez", "--intervalo", ""}} {
		if _, _, err := watchOpcoes(ruins); err == nil {
			t.Errorf("%v deveria ser erro de uso", ruins)
		}
		falso.Raizes(t)
		relogioDeMentira(t, 0, nil)
		codigo, saida := rodaComSaida(t, append([]string{"watch"}, ruins...)...)
		if obj := umaLinhaDeJSON(t, saida); codigo != base.Erro || obj["erro"] == nil || obj["erro"] == "" {
			t.Errorf("%v: código %d, saída %s", ruins, codigo, saida)
		}
	}
}

func TestWatchSoltaUmRetratoPorIntervaloEGuardaAsMedias(t *testing.T) {
	r := maquinaDeMentira(t)
	utime := uint64(500)
	esperas := relogioDeMentira(t, 4, func(volta int) {
		// 30 %, 60 %, 90 % e 0 % de um núcleo
		utime += []uint64{30, 60, 90, 0}[volta-1]
		falso.Proc{PID: 1234, PPID: 900, Comm: "firefox", Cmd: []string{"/usr/lib/firefox/firefox"}, UID: 1000, Utime: utime, Stime: 100, Inicio: 7000, RSS: int64(400000 + 1000*volta), Cgroup: cgFirefox}.Grava(t, r)
		if volta == 3 {
			falso.Apaga(t, r, "proc/300") // o sshd morreu
			falso.Apaga(t, r, "proc/meminfo")
		}
	})
	codigo, saida := rodaComSaida(t, "watch", "--intervalo", "1000")
	if codigo != base.Certo {
		t.Fatalf("parar o watch é sair certo: código %d", codigo)
	}
	linhas := strings.Split(strings.TrimSuffix(saida, "\n"), "\n")
	if len(linhas) != 4 || len(*esperas) != 4 {
		t.Fatalf("quatro intervalos, quatro retratos: vieram %d linhas e %d esperas", len(linhas), len(*esperas))
	}
	var cpus, medias, mems []float64
	for i, linha := range linhas {
		var retrato map[string]any
		if err := json.Unmarshal([]byte(linha), &retrato); err != nil {
			t.Fatalf("a linha %d não é um JSON inteiro: %v", i+1, err)
		}
		f := grupoPorID(t, retrato, "app:firefox")
		cpus = append(cpus, f["cpu_pct"].(float64))
		medias = append(medias, cava(t, f, "media", "cpu_pct_1m").(float64))
		mems = append(mems, cava(t, f, "media", "mem_kb_3m").(float64))
		temSshd := false
		for _, g := range retrato["grupos"].([]any) {
			temSshd = temSshd || g.(map[string]any)["id"] == "servico:sshd.service"
		}
		if temSshd != (i < 2) {
			t.Errorf("retrato %d: o sshd some quando o processo some (tem=%v)", i+1, temSshd)
		}
		if (retrato["mem"] == nil) != (i >= 2) {
			t.Errorf("retrato %d: um pedaço que não dá para ler vira null e o laço segue: mem=%v", i+1, retrato["mem"])
		}
	}
	if fmt.Sprint(cpus) != "[30 60 90 0]" {
		t.Errorf("cpu a cada retrato = %v", cpus)
	}
	if fmt.Sprint(medias) != "[30 45 60 45]" {
		t.Errorf("média de 1 minuto a cada retrato = %v", medias)
	}
	if fmt.Sprint(mems) != "[840000 840500 841000 841500]" {
		t.Errorf("média de memória a cada retrato = %v", mems)
	}
	for i, d := range *esperas {
		if d != time.Second {
			t.Errorf("espera %d = %v", i+1, d)
		}
	}
}

// saidaFechada imita o stdout depois que a janela fechou o cano.
type saidaFechada struct{ escritas int }

func (s *saidaFechada) Write(p []byte) (int, error) {
	s.escritas++
	return 0, syscall.EPIPE
}

func TestWatchParaQuandoOStdoutFecha(t *testing.T) {
	maquinaDeMentira(t)
	esperas := relogioDeMentira(t, 50, nil)
	fechada := &saidaFechada{}
	velha := base.Saida
	base.Saida = fechada
	defer func() { base.Saida = velha }()
	if codigo := rodar([]string{"watch", "--intervalo", "250"}); codigo != base.Certo {
		t.Errorf("stdout fechado é sair limpo: código %d", codigo)
	}
	if fechada.escritas != 1 || len(*esperas) != 1 {
		t.Errorf("deveria parar na primeira escrita que falha: %d escritas, %d esperas", fechada.escritas, len(*esperas))
	}
}

// saidaEmPedacos anota cada Write: um retrato tem de chegar num Write só.
type saidaEmPedacos struct{ pedacos []string }

func (s *saidaEmPedacos) Write(p []byte) (int, error) {
	s.pedacos = append(s.pedacos, string(p))
	return len(p), nil
}

func TestWatchNuncaEscreveJSONPelaMetade(t *testing.T) {
	maquinaDeMentira(t)
	relogioDeMentira(t, 3, nil)
	saida := &saidaEmPedacos{}
	velha := base.Saida
	base.Saida = saida
	defer func() { base.Saida = velha }()
	rodar([]string{"watch"})
	if len(saida.pedacos) != 3 {
		t.Fatalf("três retratos, três escritas: vieram %d", len(saida.pedacos))
	}
	for i, p := range saida.pedacos {
		var v map[string]any
		if !strings.HasSuffix(p, "\n") || strings.Count(p, "\n") != 1 || json.Unmarshal([]byte(p), &v) != nil {
			t.Errorf("a escrita %d não é uma linha inteira de JSON", i+1)
		}
	}
}

func TestWatchParaNoMeioDaEspera(t *testing.T) {
	// o relógio de verdade: a espera acaba quando mandam parar, sem esperar
	// o intervalo inteiro
	para := make(chan struct{})
	close(para)
	inicio := time.Now()
	if watchRel.Espera(time.Hour, para) {
		t.Error("com o pedido de parar, a espera devolve false")
	}
	if time.Since(inicio) > 2*time.Second {
		t.Error("a espera não largou quando mandaram parar")
	}
	if !watchRel.Espera(time.Millisecond, make(chan struct{})) {
		t.Error("sem pedido de parar, a espera devolve true no fim do tempo")
	}
	if a, b := watchRel.Mono(), watchRel.Mono(); b < a {
		t.Error("o relógio monotônico andou para trás")
	}
}

func TestWatchOrdenaOsGruposPorCpu(t *testing.T) {
	r := maquinaDeMentira(t)
	relogioDeMentira(t, 9, func(int) { passaUmSegundo(t, r) })
	_, saida := rodaComSaida(t, "watch", "--uma-vez")
	retrato := umaLinhaDeJSON(t, saida)
	var cpus []float64
	var ids []string
	for _, g := range retrato["grupos"].([]any) {
		cpus = append(cpus, g.(map[string]any)["cpu_pct"].(float64))
		ids = append(ids, g.(map[string]any)["id"].(string))
	}
	if !sort.SliceIsSorted(cpus, func(i, j int) bool { return cpus[i] > cpus[j] }) {
		t.Errorf("grupos fora de ordem: %v %v", ids, cpus)
	}
	if strings.Join(ids, " ") != "app:firefox servico:init.scope servico:sshd.service servico:user@1000.service sistema:kernel" {
		t.Errorf("ordem dos grupos (cpu, depois id) = %v", ids)
	}
}
