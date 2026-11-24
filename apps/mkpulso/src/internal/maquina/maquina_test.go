package maquina

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mkpulso/internal/coleta/falso"
)

const cpuinfoIntel = `processor	: 0
vendor_id	: GenuineIntel
model name	: Intel(R) Core(TM) i5-6300U CPU @ 2.40GHz
physical id	: 0
core id		: 0
cpu cores	: 2

processor	: 1
model name	: Intel(R) Core(TM) i5-6300U CPU @ 2.40GHz
physical id	: 0
core id		: 1
cpu cores	: 2

processor	: 2
model name	: Intel(R) Core(TM) i5-6300U CPU @ 2.40GHz
physical id	: 0
core id		: 0
cpu cores	: 2

processor	: 3
model name	: Intel(R) Core(TM) i5-6300U CPU @ 2.40GHz
physical id	: 0
core id		: 1
cpu cores	: 2
`

// notebook monta a máquina do exemplo do contrato.
func notebook(t *testing.T) string {
	t.Helper()
	r := falso.Raizes(t)
	e := func(caminho, texto string) { falso.Escreve(t, r, caminho, texto) }
	e("proc/sys/kernel/hostname", "nome\n")
	e("proc/sys/kernel/osrelease", "7.3.5-arch1-1\n")
	e("proc/uptime", "3600.42 12000.00\n")
	e("proc/cpuinfo", cpuinfoIntel)
	e("proc/meminfo", "MemTotal:        8000000 kB\nMemAvailable:    4000000 kB\n")
	e("sys/class/dmi/id/sys_vendor", "HP\n")
	e("sys/class/dmi/id/product_name", "HP EliteBook 840 G3\n")
	e("sys/class/dmi/id/board_vendor", "HP\n")
	e("sys/class/dmi/id/board_name", "8079\n")
	e("sys/class/dmi/id/bios_version", "N75 Ver. 01.60\n")
	e("sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_max_freq", "3000000\n")
	e("sys/devices/system/cpu/cpu1/cpufreq/cpuinfo_max_freq", "2800000\n")
	e("sys/class/drm/card1/device/uevent", "DRIVER=i915\nPCI_ID=8086:1916\n")
	e("pci.ids", "8086  Intel Corporation\n\t1916  Skylake GT2 [HD Graphics 520]\n")
	e("sys/block/nvme0n1/size", "1000000000\n")
	e("sys/block/nvme0n1/device/model", "Samsung SSD 970 EVO 500GB            \n")
	e("sys/block/nvme0n1/queue/rotational", "0\n")
	e("sys/block/sda/size", "2000000\n")
	e("sys/block/sda/device/model", "WDC WD10\n")
	e("sys/block/sda/queue/rotational", "1\n")
	e("sys/block/sdb/size", "0\n") // leitor de cartão vazio
	e("sys/block/loop0/size", "100\n")
	e("sys/block/zram0/size", "100\n")
	b := "sys/class/power_supply/BAT0/"
	e(b+"model_name", "Primary\n")
	e(b+"cycle_count", "984\n")
	e(b+"energy_full", "33100000\n")
	e(b+"energy_full_design", "46620000\n")
	e("home/.config/mk/VERSION", "4c4ea04fc26e 2026-11-23\n")
	e("mk/VERSION", "000000000000 2020-01-01\n")
	for _, p := range []string{"bash-5.3-1", "linux-7.3.5-1", "hyprland-0.57.2-1"} {
		e("pacman/"+p+"/desc", "%NAME%\n")
	}
	e("pacman/ALPM_DB_VERSION", "9\n")
	e("etc/machine-id", "abc\n")
	quando := time.Unix(1795000000, 0)
	if err := os.Chtimes(filepath.Join(r, "etc/machine-id"), quando, quando); err != nil {
		t.Fatal(err)
	}
	falso.Comando(t, r, "hyprctl", `case "$1" in
monitors) echo '[{"name": "eDP-1", "description": "BOE 0x0697", "width": 1366, "height": 768, "refreshRate": 60.0, "scale": 1.0}]' ;;
version) echo '{"version": "0.57.2", "tag": "v0.57.2"}' ;;
*) exit 1 ;;
esac`)
	return r
}

func TestFichaDoNotebook(t *testing.T) {
	notebook(t)
	f := Ler()
	b, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	quer := `{"maquina":"nome","modelo":"HP EliteBook 840 G3","placa":"HP 8079","firmware":"N75 Ver. 01.60",` +
		`"cpu":{"nome":"Intel Core i5-6300U","nucleos":2,"threads":4,"max_mhz":3000},` +
		`"mem_total_kb":8000000,` +
		`"gpus":[{"nome":"Intel HD Graphics 520","driver":"i915"}],` +
		`"discos":[{"nome":"nvme0n1","modelo":"Samsung SSD 970 EVO 500GB","tamanho_kb":500000000,"rotativo":false,"saude":null},` +
		`{"nome":"sda","modelo":"WDC WD10","tamanho_kb":1000000,"rotativo":true,"saude":null}],` +
		`"bateria":{"modelo":"Primary","ciclos":984,"saude_pct":71,"capacidade_wh":33.1},` +
		`"telas":[{"nome":"eDP-1","descricao":"BOE 0x0697","largura":1366,"altura":768,"hz":60,"escala":1}],` +
		`"sistema":{"nome":"Mk","versao_mk":"4c4ea04fc26e","data_mk":"2026-11-23","kernel":"7.3.5-arch1-1",` +
		`"compositor":"Hyprland 0.57.2","ligado_ha_s":3600,"instalado_em_s":1795000000,"pacotes":3}}`
	if string(b) != quer {
		t.Errorf("ficha:\n veio     %s\n esperava %s", b, quer)
	}
}

func TestMaquinaVaziaENull(t *testing.T) {
	falso.Raizes(t)
	t.Setenv("MKPULSO_PACMAN", "/nao/existe")
	b, err := json.Marshal(Ler())
	if err != nil {
		t.Fatal(err)
	}
	quer := `{"maquina":null,"modelo":null,"placa":null,"firmware":null,` +
		`"cpu":{"nome":null,"nucleos":null,"threads":null,"max_mhz":null},` +
		`"mem_total_kb":null,"gpus":[],"discos":[],"bateria":null,"telas":[],` +
		`"sistema":{"nome":"Mk","versao_mk":null,"data_mk":null,"kernel":null,"compositor":null,"ligado_ha_s":null,"instalado_em_s":null,"pacotes":null}}`
	if string(b) != quer {
		t.Errorf("o que não dá para saber é null:\n veio     %s\n esperava %s", b, quer)
	}
}

func TestVersaoDoMk(t *testing.T) {
	r := falso.Raizes(t)
	ver := func() (string, string) {
		s := Ler().Sistema
		v, d := "<null>", "<null>"
		if s.VersaoMk != nil {
			v = *s.VersaoMk
		}
		if s.DataMk != nil {
			d = *s.DataMk
		}
		return v, d
	}
	falso.Escreve(t, r, "mk/VERSION", "aaaaaaaaaaaa 2026-01-02\n")
	if v, d := ver(); v != "aaaaaaaaaaaa" || d != "2026-01-02" {
		t.Errorf("sem o da home vale o do sistema: %s %s", v, d)
	}
	falso.Escreve(t, r, "home/.config/mk/VERSION", "\n  \n")
	if v, d := ver(); v != "aaaaaaaaaaaa" || d != "2026-01-02" {
		t.Errorf("arquivo vazio na home é ignorado: %s %s", v, d)
	}
	falso.Escreve(t, r, "home/.config/mk/VERSION", "bbbbbbbbbbbb\n")
	if v, d := ver(); v != "bbbbbbbbbbbb" || d != "<null>" {
		t.Errorf("só o commit: a data é null: %s %s", v, d)
	}
	falso.Escreve(t, r, "home/.config/mk/VERSION", "cccccccccccc 2026-11-23 sobra\noutra linha\n")
	if v, d := ver(); v != "cccccccccccc" || d != "2026-11-23" {
		t.Errorf("vale a primeira linha: %s %s", v, d)
	}
}

func TestModeloEPlacaSemLixoDoFabricante(t *testing.T) {
	r := falso.Raizes(t)
	d := "sys/class/dmi/id/"
	modelo := func() string {
		f := Ler()
		m, p := "<null>", "<null>"
		if f.Modelo != nil {
			m = *f.Modelo
		}
		if f.Placa != nil {
			p = *f.Placa
		}
		return m + " | " + p
	}
	falso.Escreve(t, r, d+"sys_vendor", "To Be Filled By O.E.M.\n")
	falso.Escreve(t, r, d+"product_name", "System Product Name\n")
	falso.Escreve(t, r, d+"board_vendor", "ASUSTeK COMPUTER INC.\n")
	falso.Escreve(t, r, d+"board_name", "PRIME B450M-A\n")
	if m := modelo(); m != "<null> | ASUS PRIME B450M-A" {
		t.Errorf("montado em casa: %s", m)
	}
	falso.Escreve(t, r, d+"sys_vendor", "LENOVO\n")
	falso.Escreve(t, r, d+"product_name", "20L5CTO1WW\n")
	falso.Escreve(t, r, d+"product_version", "ThinkPad T480\n")
	falso.Escreve(t, r, d+"board_vendor", "LENOVO\n")
	falso.Escreve(t, r, d+"board_name", "20L5CTO1WW\n")
	if m := modelo(); m != "Lenovo ThinkPad T480 | Lenovo 20L5CTO1WW" {
		t.Errorf("a Lenovo põe o nome de vitrine em product_version: %s", m)
	}
	falso.Escreve(t, r, d+"sys_vendor", "Dell Inc.\n")
	falso.Escreve(t, r, d+"product_name", "XPS 13 9310\n")
	falso.Apaga(t, r, d+"product_version")
	falso.Apaga(t, r, d+"board_vendor")
	if m := modelo(); m != "Dell XPS 13 9310 | 20L5CTO1WW" {
		t.Errorf("dell: %s", m)
	}
}

func TestNomeDaCpu(t *testing.T) {
	casos := map[string]string{
		"Intel(R) Core(TM) i5-6300U CPU @ 2.40GHz":       "Intel Core i5-6300U",
		"AMD Ryzen 7 5800X 8-Core Processor":             "AMD Ryzen 7 5800X",
		"AMD Ryzen 5 PRO 4650U with Radeon Graphics":     "AMD Ryzen 5 PRO 4650U",
		"12th Gen Intel(R) Core(TM) i7-1260P":            "Intel Core i7-1260P",
		"Intel(R) Celeron(R) N4020 CPU @ 1.10GHz":        "Intel Celeron N4020",
		"  ARMv8   Processor  ":                          "ARMv8",
		"AMD Ryzen Threadripper 3990X 64-Core Processor": "AMD Ryzen Threadripper 3990X",
	}
	for bruto, quer := range casos {
		if veio := nomeDaCPU(bruto); veio != quer {
			t.Errorf("%q: veio %q, esperava %q", bruto, veio, quer)
		}
	}
}

func TestCpuinfoTorto(t *testing.T) {
	r := falso.Raizes(t)
	// sem physical id nem core id (ARM, VM): vale "cpu cores"; sem ele, os threads
	falso.Escreve(t, r, "proc/cpuinfo", "processor\t: 0\nmodel name\t: Coisa\ncpu cores\t: 2\n\nprocessor\t: 1\nmodel name\t: Coisa\ncpu cores\t: 2\n\nprocessor\t: 2\n\nprocessor\t: 3\n")
	c := Ler().CPU
	if c.Nome == nil || *c.Nome != "Coisa" || c.Nucleos == nil || *c.Nucleos != 2 || c.Threads == nil || *c.Threads != 4 || c.MaxMHz != nil {
		t.Errorf("cpu = %s", jsonDe(c))
	}
	falso.Escreve(t, r, "proc/cpuinfo", "processor\t: 0\nBogoMIPS\t: 48.00\n\nprocessor\t: 1\n")
	c = Ler().CPU
	if c.Nome != nil || c.Nucleos == nil || *c.Nucleos != 2 || *c.Threads != 2 {
		t.Errorf("cpu sem nome = %s", jsonDe(c))
	}
	for _, lixo := range []string{"", "\x00\xff\n:::\n", "processor\nmodel name\n", strings.Repeat("x", 100000)} {
		falso.Escreve(t, r, "proc/cpuinfo", lixo)
		c = Ler().CPU
		if c.Nucleos != nil || c.Threads != nil {
			t.Errorf("cpuinfo de lixo deveria dar null: %s", jsonDe(c))
		}
	}
}

func jsonDe(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func TestBateriaPorCargaETorta(t *testing.T) {
	r := falso.Raizes(t)
	b := "sys/class/power_supply/BAT1/"
	falso.Escreve(t, r, b+"charge_full", "3000000\n")
	falso.Escreve(t, r, b+"charge_full_design", "4000000\n")
	falso.Escreve(t, r, b+"voltage_min_design", "11400000\n")
	bat := Ler().Bateria
	if bat == nil || bat.Modelo != nil || bat.Ciclos != nil || bat.SaudePct == nil || *bat.SaudePct != 75 || bat.CapacidadeWh == nil || *bat.CapacidadeWh != 34.2 {
		t.Fatalf("bateria por carga = %s", jsonDe(bat))
	}
	// bateria nova que carrega mais que o projeto: a saúde para em 100
	falso.Escreve(t, r, b+"energy_full", "50000000\n")
	falso.Escreve(t, r, b+"energy_full_design", "40000000\n")
	if bat := Ler().Bateria; *bat.SaudePct != 100 || *bat.CapacidadeWh != 50 {
		t.Errorf("saúde acima de 100: %s", jsonDe(bat))
	}
	falso.Escreve(t, r, b+"energy_full_design", "0\n")
	falso.Escreve(t, r, b+"cycle_count", "lixo\n")
	if bat := Ler().Bateria; bat.SaudePct != nil || bat.Ciclos != nil || *bat.CapacidadeWh != 50 {
		t.Errorf("projeto zero não divide por zero: %s", jsonDe(bat))
	}
}

func TestDiscosTortos(t *testing.T) {
	r := falso.Raizes(t)
	falso.Escreve(t, r, "sys/block/sda/size", "lixo\n")
	falso.Escreve(t, r, "sys/block/sdb/size", "2048\n")
	falso.Escreve(t, r, "sys/block/sdb/queue/rotational", "talvez\n")
	ds := Ler().Discos
	if len(ds) != 1 || ds[0].Nome != "sdb" || *ds[0].TamanhoKB != 1024 || ds[0].Modelo != nil || ds[0].Rotativo != nil || ds[0].Saude != nil {
		t.Errorf("discos = %s", jsonDe(ds))
	}
}

func TestUptimeEKernelTortos(t *testing.T) {
	r := falso.Raizes(t)
	falso.Escreve(t, r, "proc/uptime", "lixo\n")
	falso.Escreve(t, r, "proc/version", "Linux version 6.1.0-teste (builder@x) #1 SMP\n")
	s := Ler().Sistema
	if s.LigadoHaS != nil || s.Kernel == nil || *s.Kernel != "6.1.0-teste" {
		t.Errorf("sistema = %s", jsonDe(s))
	}
}
