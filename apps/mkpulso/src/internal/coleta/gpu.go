package coleta

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"mkpulso/internal/base"
)

// Placa é uma GPU achada em /sys/class/drm/card*/device.
type Placa struct {
	Cartao string // "card0"
	Dir    string // a pasta device
	Nome   string
	Driver string // "" se não deu para saber
}

var cartao = regexp.MustCompile(`^card(\d+)$`)

var fabricantes = map[string]string{"1002": "AMD", "8086": "Intel", "10de": "NVIDIA", "1af4": "Virtio", "15ad": "VMware", "1234": ""}

// ListarPlacas acha as GPUs (os cardN; os conectores cardN-DP-1 e os
// renderD não contam). O nome vem de product_name quando a placa dá; senão
// do fabricante e do modelo em pci.ids (o trecho entre colchetes, que é o
// nome de vitrine); sem pci.ids, do fabricante e do código vendor:device.
func ListarPlacas() []Placa {
	pastas, _ := filepath.Glob(base.Sys("class", "drm", "card*"))
	type achada struct {
		n int
		p Placa
	}
	var achadas []achada
	for _, pasta := range pastas {
		m := cartao.FindStringSubmatch(filepath.Base(pasta))
		if m == nil {
			continue
		}
		dir := filepath.Join(pasta, "device")
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			continue
		}
		n, _ := strconv.Atoi(m[1])
		p := Placa{Cartao: filepath.Base(pasta), Dir: dir}
		var id string
		if u, err := os.ReadFile(filepath.Join(dir, "uevent")); err == nil {
			for _, linha := range strings.Split(string(u), "\n") {
				chave, valor, _ := strings.Cut(strings.TrimSpace(linha), "=")
				switch chave {
				case "DRIVER":
					p.Driver = valor
				case "PCI_ID":
					id = strings.ToLower(valor)
				}
			}
		}
		if p.Driver == "" {
			if alvo, err := os.Readlink(filepath.Join(dir, "driver")); err == nil {
				p.Driver = filepath.Base(alvo)
			}
		}
		if id == "" {
			v, okV := lerTexto(filepath.Join(dir, "vendor"))
			d, okD := lerTexto(filepath.Join(dir, "device"))
			if okV && okD {
				id = strings.ToLower(strings.TrimPrefix(v, "0x") + ":" + strings.TrimPrefix(d, "0x"))
			}
		}
		p.Nome = nomeDaPlaca(dir, id, p.Cartao)
		achadas = append(achadas, achada{n, p})
	}
	sort.Slice(achadas, func(i, j int) bool { return achadas[i].n < achadas[j].n })
	ps := make([]Placa, len(achadas))
	for i, a := range achadas {
		ps[i] = a.p
	}
	return ps
}

func nomeDaPlaca(dir, id, cartao string) string {
	if nome, ok := lerTexto(filepath.Join(dir, "product_name")); ok && nome != "" {
		return nome
	}
	if id == "" {
		return "GPU " + cartao
	}
	vendor, _, _ := strings.Cut(id, ":")
	fabricante, conhecido := fabricantes[vendor]
	if modelo := modeloPCI(id); modelo != "" {
		if a := strings.LastIndex(modelo, "["); a >= 0 && strings.HasSuffix(modelo, "]") {
			modelo = modelo[a+1 : len(modelo)-1]
		}
		if fabricante == "" || strings.HasPrefix(strings.ToLower(modelo), strings.ToLower(fabricante)) {
			return modelo
		}
		return fabricante + " " + modelo
	}
	if !conhecido || fabricante == "" {
		return "GPU " + id
	}
	return fabricante + " " + id
}

var (
	pciTrava sync.Mutex
	pciNomes = map[string]string{} // arquivo|vendor:device → modelo ("" = não achou)
)

// modeloPCI procura o nome do modelo em pci.ids (/usr/share/hwdata/pci.ids,
// trocável por MKPULSO_PCIIDS). O resultado fica guardado: o arquivo tem
// mais de um mega e o watch pergunta a cada retrato.
func modeloPCI(id string) string {
	arquivo := os.Getenv("MKPULSO_PCIIDS")
	if arquivo == "" {
		arquivo = "/usr/share/hwdata/pci.ids"
	}
	chave := arquivo + "|" + id
	pciTrava.Lock()
	defer pciTrava.Unlock()
	if nome, ok := pciNomes[chave]; ok {
		return nome
	}
	nome := ""
	vendor, device, _ := strings.Cut(id, ":")
	if b, err := os.ReadFile(arquivo); err == nil {
		dentro := false
		for _, linha := range strings.Split(string(b), "\n") {
			if linha == "" || linha[0] == '#' {
				continue
			}
			if linha[0] != '\t' {
				if dentro {
					break
				}
				dentro = strings.HasPrefix(linha, vendor+"  ")
				continue
			}
			if dentro && strings.HasPrefix(linha, "\t"+device+"  ") {
				nome = strings.TrimSpace(linha[len(device)+3:])
				break
			}
		}
	}
	pciNomes[chave] = nome
	return nome
}

// GPU é um item de "gpus" do retrato.
type GPU struct {
	Nome       string   `json:"nome"`
	UsoPct     *float64 `json:"uso_pct"`
	MemUsadaKB *int64   `json:"mem_usada_kb"`
	MemTotalKB *int64   `json:"mem_total_kb"`
	TempC      *float64 `json:"temp_c"`
}

// LerGPUs lê o uso de cada GPU. A AMD dá gpu_busy_percent e
// mem_info_vram_*; a Intel e a NVIDIA não dão nada pelo sysfs, e aí os
// números são null (a placa aparece do mesmo jeito).
func LerGPUs() []GPU {
	gs := []GPU{}
	for _, p := range ListarPlacas() {
		g := GPU{Nome: p.Nome}
		if v, ok := lerNumero(filepath.Join(p.Dir, "gpu_busy_percent")); ok && v <= 100 {
			g.UsoPct = Ptr(float64(v))
		}
		if v, ok := lerNumero(filepath.Join(p.Dir, "mem_info_vram_used")); ok {
			g.MemUsadaKB = Ptr(int64(v / 1024))
		}
		if v, ok := lerNumero(filepath.Join(p.Dir, "mem_info_vram_total")); ok {
			g.MemTotalKB = Ptr(int64(v / 1024))
		}
		sensores, _ := filepath.Glob(filepath.Join(p.Dir, "hwmon", "hwmon*", "temp1_input"))
		for _, s := range sensores {
			if g.TempC = miligraus(s); g.TempC != nil {
				break
			}
		}
		gs = append(gs, g)
	}
	return gs
}

// LerGPUProcesso soma, por motor (gfx, render, dec...), os nanossegundos de
// GPU que um processo gastou, lidos de /proc/<pid>/fdinfo/* (as linhas
// "drm-engine-<motor>: N ns"). O mesmo cliente DRM aberto por dois fds conta
// uma vez. Sem nenhuma linha dessas (processo sem GPU, driver que não conta,
// fdinfo ilegível), o segundo valor é false e o gpu_pct sai null.
func LerGPUProcesso(pid int) (map[string]uint64, bool) {
	n := strconv.Itoa(pid)
	var fds []string
	if entradas, err := os.ReadDir(base.Proc(n, "fd")); err == nil {
		// o caminho rápido: só os fds que apontam para uma placa
		for _, e := range entradas {
			if alvo, err := os.Readlink(base.Proc(n, "fd", e.Name())); err == nil && strings.HasPrefix(alvo, "/dev/dri/") {
				fds = append(fds, e.Name())
			}
		}
	} else if entradas, err := os.ReadDir(base.Proc(n, "fdinfo")); err == nil {
		for _, e := range entradas {
			fds = append(fds, e.Name())
		}
	}
	var motores map[string]uint64
	clientes := map[string]bool{}
	for _, fd := range fds {
		b, err := os.ReadFile(base.Proc(n, "fdinfo", fd))
		if err != nil || !strings.Contains(string(b), "drm-engine-") {
			continue
		}
		cliente := "fd" + fd
		deste := map[string]uint64{}
		for _, linha := range strings.Split(string(b), "\n") {
			chave, valor, ok := strings.Cut(linha, ":")
			if !ok {
				continue
			}
			c := strings.Fields(valor)
			if len(c) == 0 {
				continue
			}
			switch {
			case chave == "drm-client-id":
				cliente = "c" + c[0]
			case strings.HasPrefix(chave, "drm-engine-capacity-"):
			case strings.HasPrefix(chave, "drm-engine-"):
				if len(c) >= 2 && c[1] == "ns" {
					deste[strings.TrimPrefix(chave, "drm-engine-")] = numero(c[0])
				}
			}
		}
		if len(deste) == 0 || clientes[cliente] {
			continue
		}
		clientes[cliente] = true
		if motores == nil {
			motores = map[string]uint64{}
		}
		for m, ns := range deste {
			motores[m] += ns
		}
	}
	return motores, motores != nil
}
