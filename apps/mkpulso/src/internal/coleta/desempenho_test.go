package coleta

import (
	"errors"
	"strconv"
	"testing"
	"time"

	"mkpulso/internal/coleta/falso"
)

func TestMemoria(t *testing.T) {
	r := falso.Raizes(t)
	falso.Escreve(t, r, "proc/meminfo", "MemTotal:       16000000 kB\nMemFree:         1000000 kB\nMemAvailable:    7000000 kB\nBuffers:          200000 kB\nCached:          2500000 kB\nSwapCached:            0 kB\nSReclaimable:     300000 kB\nSwapTotal:       4000000 kB\nSwapFree:        3000000 kB\nHugePages_Total:       0\n")
	m := LerMemoria()
	if m == nil {
		t.Fatal("memória veio null")
	}
	if m.TotalKB != 16000000 || m.DisponivelKB != 7000000 || m.UsadoKB != 9000000 || m.CacheKB != 3000000 || m.SwapTotalKB != 4000000 || m.SwapUsadoKB != 1000000 {
		t.Errorf("memória = %+v", *m)
	}
}

func TestMemoriaTorta(t *testing.T) {
	r := falso.Raizes(t)
	if LerMemoria() != nil {
		t.Error("sem /proc/meminfo a memória é null")
	}
	for _, texto := range []string{"", "MemTotal:\n", "lixo sem dois pontos\nMemTotal: abc kB\n"} {
		falso.Escreve(t, r, "proc/meminfo", texto)
		if m := LerMemoria(); m != nil {
			t.Errorf("%q: sem MemTotal a memória é null, veio %+v", texto, *m)
		}
	}
	// kernel antigo, sem MemAvailable: livre + cache
	falso.Escreve(t, r, "proc/meminfo", "MemTotal: 1000 kB\nMemFree: 100 kB\nBuffers: 50 kB\nCached: 250 kB\n")
	if m := LerMemoria(); m == nil || m.DisponivelKB != 400 || m.UsadoKB != 600 {
		t.Errorf("sem MemAvailable: %+v", m)
	}
	// disponível maior que o total não dá uso negativo
	falso.Escreve(t, r, "proc/meminfo", "MemTotal: 1000 kB\nMemAvailable: 99999999999999999999999 kB\nSwapTotal: 10 kB\nSwapFree: 500 kB\n")
	if m := LerMemoria(); m == nil || m.UsadoKB < 0 || m.SwapUsadoKB < 0 {
		t.Errorf("número torto deu uso negativo: %+v", m)
	}
}

const diskstats = `   7       0 loop0 100 0 800 10 0 0 0 0 0 10 10 0 0 0 0 0 0
 259       0 nvme0n1 5000 10 20000 900 3000 20 40000 800 0 500 1700 0 0 0 0 0 0
 259       1 nvme0n1p1 100 0 800 10 0 0 0 0 0 10 10 0 0 0 0 0 0
 259       2 nvme0n1p2 4900 10 19200 890 3000 20 40000 800 0 490 1690 0 0 0 0 0 0
   8       0 sda 10 0 80 1 0 0 0 0 0 1 1 0 0 0 0 0 0
   8       1 sda1 10 0 80 1 0 0 0 0 0 1 1 0 0 0 0 0 0
 253       0 zram0 50 0 400 1 60 0 480 1 0 1 2 0 0 0 0 0 0
   1       0 ram0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0
 254       0 dm-0 4900 0 19200 890 3000 0 40000 800 0 490 1690 0 0 0 0 0 0
 179       0 mmcblk0 1 0 8 1 0 0 0 0 0 1 1 0 0 0 0 0 0
 179       1 mmcblk0p1 1 0 8 1 0 0 0 0 0 1 1 0 0 0 0 0 0
linha torta
 8 16
`

func nomesDeDisco(ds []Disco) []string {
	var n []string
	for _, d := range ds {
		n = append(n, d.Nome)
	}
	return n
}

func TestDiscosSoInteiros(t *testing.T) {
	r := falso.Raizes(t)
	falso.Escreve(t, r, "proc/diskstats", diskstats)
	antes := Colher(0)
	falso.Escreve(t, r, "proc/diskstats", " 259       0 nvme0n1 5000 10 20016 900 3000 20 40032 800 0 500 1700\n   8       0 sda 10 0 80 1 0 0 0 0 0 1 1\n 179       0 mmcblk0 1 0 2 1 0 0 0 0 0 1 1\n")
	depois := Colher(2 * time.Second)
	ds := MedirDiscos(antes, depois)
	n := nomesDeDisco(ds)
	if len(n) != 3 || n[0] != "mmcblk0" || n[1] != "nvme0n1" || n[2] != "sda" {
		t.Fatalf("só discos inteiros, em ordem de nome: %v", n)
	}
	if ds[1].LeituraBps != 4096 || ds[1].EscritaBps != 8192 { // 16 e 32 setores de 512 em 2 s
		t.Errorf("nvme0n1 = %+v", ds[1])
	}
	if ds[0].LeituraBps != 0 {
		t.Errorf("contador que voltou deveria dar 0: %+v", ds[0])
	}
	if d := MedirDiscos(nil, depois); len(d) != 3 || d[1].LeituraBps != 0 {
		t.Errorf("sem leitura anterior: os discos aparecem com taxa 0: %+v", d)
	}
}

func TestDiscoInteiroPeloSysBlock(t *testing.T) {
	r := falso.Raizes(t)
	// com /sys/block, vale o que está lá (um nome estranho entra, a partição não)
	falso.Escreve(t, r, "sys/block/estranho0/size", "100\n")
	falso.Escreve(t, r, "sys/block/loop0/size", "100\n")
	falso.Escreve(t, r, "sys/block/dm-0/size", "100\n")
	for nome, quer := range map[string]bool{"estranho0": true, "sda": false, "loop0": false, "dm-0": false, "zram0": false, "sr0": false} {
		if DiscoInteiro(nome) != quer {
			t.Errorf("DiscoInteiro(%q) deveria ser %v", nome, quer)
		}
	}
}

func TestMontagens(t *testing.T) {
	r := falso.Raizes(t)
	falso.Escreve(t, r, "proc/self/mounts", `proc /proc proc rw 0 0
sysfs /sys sysfs rw 0 0
tmpfs /tmp tmpfs rw 0 0
/dev/nvme0n1p2 / btrfs rw,subvol=/@ 0 0
/dev/nvme0n1p2 /home btrfs rw,subvol=/@home 0 0
/dev/nvme0n1p1 /boot vfat rw 0 0
/dev/sda1 /mnt/Meu\040Disco ext4 rw 0 0
/dev/sdb1 /mnt/quebrado ext4 rw 0 0
overlay /var/lib/docker/overlay2/x overlay rw 0 0
/dev/loop3 /var/lib/snapd/snap/x squashfs ro 0 0
torta
`)
	velho := Statfs
	t.Cleanup(func() { Statfs = velho })
	Statfs = func(onde string) (uint64, uint64, error) {
		switch onde {
		case "/":
			return 500000000, 120000000, nil
		case "/boot":
			return 1000000, 100, nil
		case "/mnt/Meu Disco":
			return 2000, 1, nil
		}
		return 0, 0, errors.New("não deu")
	}
	ms := LerMontagens()
	if len(ms) != 4 {
		t.Fatalf("esperava /, /boot e os dois de /mnt, vieram %d: %+v", len(ms), ms)
	}
	if ms[0].Onde != "/" || ms[0].FS != "btrfs" || *ms[0].TotalKB != 500000000 || *ms[0].UsadoKB != 120000000 {
		t.Errorf("raiz = %+v", ms[0])
	}
	if ms[2].Onde != "/mnt/Meu Disco" {
		t.Errorf("o \\040 do mounts é um espaço: %q", ms[2].Onde)
	}
	if ms[3].Onde != "/mnt/quebrado" || ms[3].TotalKB != nil || ms[3].UsadoKB != nil {
		t.Errorf("statfs que falha vira null: %+v", ms[3])
	}
	falso.Apaga(t, r, "proc/self/mounts")
	if ms := LerMontagens(); ms == nil || len(ms) != 0 {
		t.Errorf("sem mounts: lista vazia, não null: %#v", ms)
	}
}

func TestRede(t *testing.T) {
	r := falso.Raizes(t)
	cab := "Inter-|   Receive                                                |  Transmit\n face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed\n"
	falso.Escreve(t, r, "proc/net/dev", cab+
		"    lo: 5000 10 0 0 0 0 0 0 5000 10 0 0 0 0 0 0\n"+
		" wlan0: 10000 10 0 0 0 0 0 0 2000 10 0 0 0 0 0 0\n"+
		"enp0s31f6u1u2:999999 10 0 0 0 0 0 0 1 10 0 0 0 0 0 0\n"+
		"torta sem dois pontos\n"+
		"curta: 1 2\n")
	antes := Colher(0)
	falso.Escreve(t, r, "proc/net/dev", cab+
		" wlan0: 12800 10 0 0 0 0 0 0 2600 10 0 0 0 0 0 0\n"+
		"enp0s31f6u1u2:1000999 10 0 0 0 0 0 0 1 10 0 0 0 0 0 0\n")
	depois := Colher(2 * time.Second)
	rs := MedirRede(antes, depois)
	if len(rs) != 2 || rs[0].Nome != "enp0s31f6u1u2" || rs[1].Nome != "wlan0" {
		t.Fatalf("rede sem lo, em ordem de nome: %+v", rs)
	}
	if rs[0].RxBps != 500 || rs[0].TxBps != 0 {
		t.Errorf("interface de nome comprido (colado no ':'): %+v", rs[0])
	}
	if rs[1].RxBps != 1400 || rs[1].TxBps != 300 {
		t.Errorf("wlan0 = %+v", rs[1])
	}
}

func TestTemperatura(t *testing.T) {
	r := falso.Raizes(t)
	if Temperatura() != nil {
		t.Error("sem sensor a temperatura é null")
	}
	// só zona térmica
	falso.Escreve(t, r, "sys/class/thermal/thermal_zone0/type", "acpitz\n")
	falso.Escreve(t, r, "sys/class/thermal/thermal_zone0/temp", "48000\n")
	if v := Temperatura(); v == nil || *v != 48 {
		t.Errorf("thermal_zone = %v", v)
	}
	falso.Escreve(t, r, "sys/class/thermal/thermal_zone1/type", "x86_pkg_temp\n")
	falso.Escreve(t, r, "sys/class/thermal/thermal_zone1/temp", "52500\n")
	if v := Temperatura(); v == nil || *v != 52.5 {
		t.Errorf("a zona x86_pkg_temp ganha da acpitz: %v", v)
	}
	// o hwmon da CPU ganha de qualquer zona
	falso.Escreve(t, r, "sys/class/hwmon/hwmon0/name", "acpitz\n")
	falso.Escreve(t, r, "sys/class/hwmon/hwmon0/temp1_input", "30000\n")
	falso.Escreve(t, r, "sys/class/hwmon/hwmon6/name", "coretemp\n")
	falso.Escreve(t, r, "sys/class/hwmon/hwmon6/temp2_label", "Core 0\n")
	falso.Escreve(t, r, "sys/class/hwmon/hwmon6/temp2_input", "70000\n")
	falso.Escreve(t, r, "sys/class/hwmon/hwmon6/temp1_label", "Package id 0\n")
	falso.Escreve(t, r, "sys/class/hwmon/hwmon6/temp1_input", "61000\n")
	if v := Temperatura(); v == nil || *v != 61 {
		t.Errorf("coretemp: vale o pacote, veio %v", v)
	}
}

func TestTemperaturaAmd(t *testing.T) {
	for _, nome := range []string{"k10temp", "zenpower"} {
		r := falso.Raizes(t)
		falso.Escreve(t, r, "sys/class/hwmon/hwmon2/name", nome+"\n")
		falso.Escreve(t, r, "sys/class/hwmon/hwmon2/temp1_label", "Tctl\n")
		falso.Escreve(t, r, "sys/class/hwmon/hwmon2/temp1_input", "55125\n")
		if v := Temperatura(); v == nil || *v != 55.1 {
			t.Errorf("%s = %v", nome, v)
		}
		// sensor que devolve lixo não vale
		falso.Escreve(t, r, "sys/class/hwmon/hwmon2/temp1_input", "lixo\n")
		if v := Temperatura(); v != nil {
			t.Errorf("%s com lixo = %v", nome, *v)
		}
	}
}

func TestFrequenciaECarga(t *testing.T) {
	r := falso.Raizes(t)
	if Frequencia() != nil || Carga() != nil {
		t.Error("sem arquivos, frequência e carga são null")
	}
	falso.Escreve(t, r, "proc/cpuinfo", "processor\t: 0\ncpu MHz\t\t: 1000.000\n\nprocessor\t: 1\ncpu MHz\t\t: 3000.000\n")
	if v := Frequencia(); v == nil || *v != 2000 {
		t.Errorf("frequência pelo cpuinfo = %v", v)
	}
	falso.Escreve(t, r, "sys/devices/system/cpu/cpu0/cpufreq/scaling_cur_freq", "2400000\n")
	falso.Escreve(t, r, "sys/devices/system/cpu/cpu1/cpufreq/scaling_cur_freq", "2600000\n")
	falso.Escreve(t, r, "sys/devices/system/cpu/cpufreq/x", "1\n")
	if v := Frequencia(); v == nil || *v != 2500 {
		t.Errorf("frequência pelo cpufreq = %v", v)
	}
	falso.Escreve(t, r, "proc/loadavg", "1.20 0.90 0.70 2/1500 12345\n")
	if c := Carga(); len(c) != 3 || c[0] != 1.2 || c[1] != 0.9 || c[2] != 0.7 {
		t.Errorf("carga = %v", c)
	}
	falso.Escreve(t, r, "proc/loadavg", "1.20 x\n")
	if c := Carga(); c != nil {
		t.Errorf("loadavg torto = %v", c)
	}
}

func TestGpuAmd(t *testing.T) {
	r := falso.Raizes(t)
	d := "sys/class/drm/card0/device/"
	falso.Escreve(t, r, d+"uevent", "DRIVER=amdgpu\nPCI_CLASS=30000\nPCI_ID=1002:731F\n")
	falso.Escreve(t, r, d+"vendor", "0x1002\n")
	falso.Escreve(t, r, d+"device", "0x731f\n")
	falso.Escreve(t, r, d+"gpu_busy_percent", "12\n")
	falso.Escreve(t, r, d+"mem_info_vram_used", "819200000\n")
	falso.Escreve(t, r, d+"mem_info_vram_total", "8192000000\n")
	falso.Escreve(t, r, d+"hwmon/hwmon3/temp1_input", "55000\n")
	falso.Escreve(t, r, "sys/class/drm/card0-DP-1/status", "connected\n")
	falso.Escreve(t, r, "sys/class/drm/renderD128/dev", "226:128\n")
	falso.Escreve(t, r, "pci.ids", "# comentário\n1002  Advanced Micro Devices, Inc. [AMD/ATI]\n\t731e  Outra\n\t731f  Navi 10 [Radeon RX 5700]\n\t\t1002 0b36  Subsistema\n8086  Intel Corporation\n\t1916  Skylake GT2 [HD Graphics 520]\n")
	gs := LerGPUs()
	if len(gs) != 1 {
		t.Fatalf("esperava uma GPU (conectores e renderD não contam): %+v", gs)
	}
	g := gs[0]
	if g.Nome != "AMD Radeon RX 5700" {
		t.Errorf("nome = %q", g.Nome)
	}
	if g.UsoPct == nil || *g.UsoPct != 12 || g.MemUsadaKB == nil || *g.MemUsadaKB != 800000 || g.MemTotalKB == nil || *g.MemTotalKB != 8000000 || g.TempC == nil || *g.TempC != 55 {
		t.Errorf("gpu = %+v", g)
	}
	if ps := ListarPlacas(); len(ps) != 1 || ps[0].Driver != "amdgpu" {
		t.Errorf("placas = %+v", ps)
	}
}

func TestGpuIntelENomePorCodigo(t *testing.T) {
	r := falso.Raizes(t)
	d := "sys/class/drm/card1/device/"
	falso.Escreve(t, r, d+"uevent", "DRIVER=i915\nPCI_ID=8086:1916\n")
	gs := LerGPUs()
	if len(gs) != 1 || gs[0].Nome != "Intel 8086:1916" {
		t.Fatalf("sem pci.ids o nome sai do fabricante e do código: %+v", gs)
	}
	if gs[0].UsoPct != nil || gs[0].MemUsadaKB != nil || gs[0].MemTotalKB != nil || gs[0].TempC != nil {
		t.Errorf("Intel sem medida de uso: tudo null, veio %+v", gs[0])
	}
	// sem uevent, vale vendor e device; fabricante desconhecido sai só o código
	falso.Escreve(t, r, "sys/class/drm/card2/device/vendor", "0x1234\n")
	falso.Escreve(t, r, "sys/class/drm/card2/device/device", "0xABCD\n")
	falso.Escreve(t, r, "sys/class/drm/card3/device/product_name", "Placa Com Nome\n")
	gs = LerGPUs()
	if len(gs) != 3 || gs[1].Nome != "GPU 1234:abcd" || gs[2].Nome != "Placa Com Nome" {
		t.Errorf("gpus = %+v", gs)
	}
	falso.Apaga(t, r, "sys/class/drm")
	if gs := LerGPUs(); gs == nil || len(gs) != 0 {
		t.Errorf("sem GPU: lista vazia, não null: %#v", gs)
	}
}

func TestGpuPorProcesso(t *testing.T) {
	r := falso.Raizes(t)
	falso.Stat(t, r, [2]uint64{1, 1})
	fd := func(cliente int, gfx, dec uint64) string {
		return "pos:\t0\nflags:\t02100002\nmnt_id:\t26\nino:\t500\ndrm-driver:\tamdgpu\ndrm-client-id:\t" + strconv.Itoa(cliente) + "\ndrm-pdev:\t0000:03:00.0\ndrm-engine-gfx:\t" + strconv.FormatUint(gfx, 10) + " ns\ndrm-engine-dec:\t" + strconv.FormatUint(dec, 10) + " ns\ndrm-memory-vram:\t100 KiB\n"
	}
	falso.Proc{PID: 10, PPID: 1, Comm: "jogo", Cmd: []string{"jogo"}, Cgroup: "/a.service"}.Grava(t, r)
	falso.Escreve(t, r, "proc/10/fdinfo/0", "pos:\t0\nflags:\t0100000\n")
	falso.Escreve(t, r, "proc/10/fdinfo/5", fd(7, 1000000000, 0))
	falso.Escreve(t, r, "proc/10/fdinfo/6", fd(7, 1000000000, 0)) // o mesmo cliente por outro fd: conta uma vez
	falso.Escreve(t, r, "proc/10/fdinfo/7", fd(8, 0, 500000000))
	falso.Proc{PID: 11, PPID: 1, Comm: "semgpu", Cmd: []string{"semgpu"}, Cgroup: "/a.service"}.Grava(t, r)
	falso.Escreve(t, r, "proc/11/fdinfo/0", "pos:\t0\n")
	antes := Colher(0)
	falso.Escreve(t, r, "proc/10/fdinfo/5", fd(7, 1500000000, 0))
	falso.Escreve(t, r, "proc/10/fdinfo/6", fd(7, 1500000000, 0))
	falso.Escreve(t, r, "proc/10/fdinfo/7", fd(8, 0, 700000000))
	depois := Colher(2 * time.Second)
	ms := MedirProcessos(antes, depois)
	if g := medido(t, ms, 10).GPUPct; g == nil || *g != 25 { // 0,5 s de gfx em 2 s; o motor mais ocupado
		t.Errorf("gpu do processo = %v", g)
	}
	if g := medido(t, ms, 11).GPUPct; g != nil {
		t.Errorf("processo sem drm-engine: null, veio %v", *g)
	}
	if g := medido(t, MedirProcessos(nil, depois), 10).GPUPct; g == nil || *g != 0 {
		t.Errorf("primeira leitura de quem usa GPU: 0, veio %v", g)
	}
}

func TestBateria(t *testing.T) {
	r := falso.Raizes(t)
	if LerBateria() != nil {
		t.Error("sem bateria é null")
	}
	falso.Escreve(t, r, "sys/class/power_supply/AC/online", "1\n")
	if LerBateria() != nil {
		t.Error("a fonte não é bateria")
	}
	b := "sys/class/power_supply/BAT0/"
	falso.Escreve(t, r, b+"capacity", "75\n")
	falso.Escreve(t, r, b+"status", "Discharging\n")
	falso.Escreve(t, r, b+"power_now", "9400000\n")
	bat := LerBateria()
	if bat == nil || bat.Pct == nil || *bat.Pct != 75 || bat.Carregando || bat.Watts == nil || *bat.Watts != 9.4 {
		t.Fatalf("bateria = %+v", bat)
	}
	falso.Apaga(t, r, b+"power_now")
	falso.Escreve(t, r, b+"status", "Charging\n")
	falso.Escreve(t, r, b+"current_now", "1000000\n")
	falso.Escreve(t, r, b+"voltage_now", "11500000\n")
	bat = LerBateria()
	if bat == nil || !bat.Carregando || bat.Watts == nil || *bat.Watts != 11.5 {
		t.Errorf("corrente x tensão = %+v", bat)
	}
	falso.Apaga(t, r, b+"current_now")
	falso.Escreve(t, r, b+"capacity", "\n")
	bat = LerBateria()
	if bat == nil || bat.Pct != nil || bat.Watts != nil {
		t.Errorf("o que não dá para ler é null: %+v", bat)
	}
}

func TestMaquina(t *testing.T) {
	r := falso.Raizes(t)
	if Maquina() != "" {
		t.Errorf("sem nome: vazio, veio %q", Maquina())
	}
	falso.Escreve(t, r, "etc/hostname", "da-etc\n")
	if Maquina() != "da-etc" {
		t.Errorf("nome pelo /etc/hostname = %q", Maquina())
	}
	falso.Escreve(t, r, "proc/sys/kernel/hostname", "do-kernel\n")
	if Maquina() != "do-kernel" {
		t.Errorf("nome pelo kernel = %q", Maquina())
	}
}
