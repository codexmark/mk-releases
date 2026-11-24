// Package maquina monta a ficha do `sistema`: o que a máquina é (modelo,
// processador, discos, bateria, telas) e o que roda nela (o Mk, o kernel, o
// compositor). É lida uma vez só, sem taxas.
//
// De onde vem cada coisa: o modelo, a placa e o firmware, do DMI
// (/sys/class/dmi/id); o processador, de /proc/cpuinfo e do cpufreq; os
// discos, de /sys/block; a bateria, de /sys/class/power_supply; as GPUs, de
// /sys/class/drm (pacote coleta); as telas e o compositor, do hyprctl
// (pacote janelas); a versão do Mk, do arquivo VERSION ("commit data").
//
// O que não dá para saber é nil (null no JSON). A saúde de um disco é sempre
// null: o SMART só responde a root, e o coletor nunca roda como root.
package maquina

import (
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"mkpulso/internal/base"
	"mkpulso/internal/coleta"
	"mkpulso/internal/janelas"
)

// CPU é o bloco "cpu" da ficha.
type CPU struct {
	Nome    *string  `json:"nome"`
	Nucleos *int     `json:"nucleos"` // núcleos físicos
	Threads *int     `json:"threads"` // núcleos lógicos
	MaxMHz  *float64 `json:"max_mhz"`
}

// GPU é um item de "gpus" da ficha.
type GPU struct {
	Nome   string  `json:"nome"`
	Driver *string `json:"driver"`
}

// Disco é um item de "discos" da ficha.
type Disco struct {
	Nome      string  `json:"nome"`
	Modelo    *string `json:"modelo"`
	TamanhoKB *int64  `json:"tamanho_kb"`
	Rotativo  *bool   `json:"rotativo"`
	Saude     *string `json:"saude"`
}

// Bateria é o bloco "bateria" da ficha.
type Bateria struct {
	Modelo       *string  `json:"modelo"`
	Ciclos       *int     `json:"ciclos"`
	SaudePct     *int     `json:"saude_pct"`
	CapacidadeWh *float64 `json:"capacidade_wh"`
}

// Sistema é o bloco "sistema" da ficha.
type Sistema struct {
	Nome         string  `json:"nome"`
	VersaoMk     *string `json:"versao_mk"`
	DataMk       *string `json:"data_mk"`
	Kernel       *string `json:"kernel"`
	Compositor   *string `json:"compositor"`
	LigadoHaS    *int64  `json:"ligado_ha_s"`
	InstaladoEmS *int64  `json:"instalado_em_s"`
	Pacotes      *int    `json:"pacotes"`
}

// Ficha é a resposta do subcomando `sistema`.
type Ficha struct {
	Maquina    *string        `json:"maquina"`
	Modelo     *string        `json:"modelo"`
	Placa      *string        `json:"placa"`
	Firmware   *string        `json:"firmware"`
	CPU        CPU            `json:"cpu"`
	MemTotalKB *int64         `json:"mem_total_kb"`
	GPUs       []GPU          `json:"gpus"`
	Discos     []Disco        `json:"discos"`
	Bateria    *Bateria       `json:"bateria"`
	Telas      []janelas.Tela `json:"telas"`
	Sistema    Sistema        `json:"sistema"`
}

// Ler monta a ficha inteira. Cada pedaço é independente: o que falha fica
// null e o resto sai.
func Ler() Ficha {
	f := Ficha{GPUs: []GPU{}, Discos: lerDiscos(), Bateria: lerBateria(), Telas: janelas.Telas()}
	f.Maquina = texto(coleta.Maquina())
	f.Modelo, f.Placa, f.Firmware = lerDMI()
	f.CPU = lerCPU()
	if m := coleta.LerMemoria(); m != nil {
		f.MemTotalKB = &m.TotalKB
	}
	for _, p := range coleta.ListarPlacas() {
		f.GPUs = append(f.GPUs, GPU{Nome: p.Nome, Driver: texto(p.Driver)})
	}
	f.Sistema = lerSistema()
	return f
}

// texto devolve nil para o vazio.
func texto(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func arquivo(caminho string) string {
	b, err := os.ReadFile(caminho)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(strings.ToValidUTF8(string(b), ""))
}

func inteiro(caminho string) (int64, bool) {
	n, err := strconv.ParseInt(arquivo(caminho), 10, 64)
	return n, err == nil
}

// --- DMI ---

// O que os fabricantes deixam nos campos que não preencheram.
var lixoDoDMI = map[string]bool{
	"": true, "to be filled by o.e.m.": true, "default string": true, "system product name": true,
	"system manufacturer": true, "system version": true, "not specified": true, "not applicable": true,
	"none": true, "o.e.m.": true, "unknown": true, "n/a": true, "invalid": true, "type1productconfigid": true,
}

var apelidos = map[string]string{
	"hewlett-packard": "HP", "lenovo": "Lenovo", "asustek computer inc.": "ASUS", "dell inc.": "Dell",
	"acer": "Acer", "micro-star international co., ltd.": "MSI", "gigabyte technology co., ltd.": "Gigabyte",
	"samsung electronics co., ltd.": "Samsung",
}

func dmi(campo string) string {
	v := arquivo(base.Sys("class", "dmi", "id", campo))
	if lixoDoDMI[strings.ToLower(v)] {
		return ""
	}
	return v
}

func fabricante(campo string) string {
	v := dmi(campo)
	if apelido, ok := apelidos[strings.ToLower(v)]; ok {
		return apelido
	}
	return v
}

// junta põe o fabricante na frente do nome, se o nome já não começa com ele.
func junta(fabricante, nome string) *string {
	if nome == "" {
		return nil
	}
	if fabricante == "" || strings.HasPrefix(strings.ToLower(nome), strings.ToLower(fabricante)) {
		return &nome
	}
	s := fabricante + " " + nome
	return &s
}

func lerDMI() (modelo, placa, firmware *string) {
	marca := fabricante("sys_vendor")
	produto := dmi("product_name")
	// a Lenovo põe o código em product_name e o nome de vitrine em product_version
	if marca == "Lenovo" {
		if vitrine := dmi("product_version"); vitrine != "" {
			produto = vitrine
		}
	}
	return junta(marca, produto), junta(fabricante("board_vendor"), dmi("board_name")), texto(dmi("bios_version"))
}

// --- processador ---

var (
	marcasRegistradas = regexp.MustCompile(`(?i)\((r|tm|c)\)`)
	geracao           = regexp.MustCompile(`^\d+(st|nd|rd|th) Gen `)
	rabos             = regexp.MustCompile(`(?i)( CPU)?( @ [\d.]+ ?[GM]Hz| \d+-Core Processor| with Radeon.*| Processor| CPU)$`)
	espacos           = regexp.MustCompile(`\s+`)
)

// nomeDaCPU limpa o "model name": tira (R), (TM), a geração, o "CPU @ 2.40GHz"
// e o "8-Core Processor".
func nomeDaCPU(bruto string) string {
	s := marcasRegistradas.ReplaceAllString(bruto, " ")
	s = strings.TrimSpace(espacos.ReplaceAllString(s, " "))
	s = geracao.ReplaceAllString(s, "")
	for {
		limpo := strings.TrimSpace(rabos.ReplaceAllString(s, ""))
		if limpo == s {
			return s
		}
		s = limpo
	}
}

func lerCPU() CPU {
	var c CPU
	b, err := os.ReadFile(base.Proc("cpuinfo"))
	if err != nil {
		return c
	}
	threads, declarados := 0, 0
	fisicos := map[string]bool{}
	pacote := ""
	for _, linha := range strings.Split(string(b), "\n") {
		chave, valor, ok := strings.Cut(linha, ":")
		if !ok {
			continue
		}
		chave, valor = strings.TrimSpace(chave), strings.TrimSpace(valor)
		switch chave {
		case "processor":
			threads++
			pacote = ""
		case "model name":
			if c.Nome == nil {
				c.Nome = texto(nomeDaCPU(strings.ToValidUTF8(valor, "")))
			}
		case "physical id":
			pacote = valor
		case "core id":
			fisicos[pacote+"/"+valor] = true
		case "cpu cores":
			if n, err := strconv.Atoi(valor); err == nil && n > declarados {
				declarados = n
			}
		}
	}
	if threads == 0 {
		return c
	}
	c.Threads = &threads
	nucleos := threads
	if len(fisicos) > 0 {
		nucleos = len(fisicos)
	} else if declarados > 0 {
		nucleos = declarados
	}
	c.Nucleos = &nucleos

	maior := int64(0)
	arquivos, _ := filepath.Glob(base.Sys("devices", "system", "cpu", "cpu[0-9]*", "cpufreq", "cpuinfo_max_freq"))
	for _, a := range arquivos {
		if khz, ok := inteiro(a); ok && khz > maior {
			maior = khz
		}
	}
	if maior > 0 {
		c.MaxMHz = coleta.Ptr(math.Round(float64(maior) / 1000))
	}
	return c
}

// --- discos ---

func lerDiscos() []Disco {
	ds := []Disco{}
	entradas, err := os.ReadDir(base.Sys("block"))
	if err != nil {
		return ds
	}
	for _, e := range entradas {
		nome := e.Name()
		if !coleta.DiscoInteiro(nome) {
			continue
		}
		// size é em setores de 512 bytes; zero é um leitor de cartão vazio
		setores, ok := inteiro(base.Sys("block", nome, "size"))
		if !ok || setores <= 0 {
			continue
		}
		d := Disco{Nome: nome, TamanhoKB: coleta.Ptr(setores / 2)}
		d.Modelo = texto(arquivo(base.Sys("block", nome, "device", "model")))
		switch arquivo(base.Sys("block", nome, "queue", "rotational")) {
		case "0":
			d.Rotativo = coleta.Ptr(false)
		case "1":
			d.Rotativo = coleta.Ptr(true)
		}
		ds = append(ds, d)
	}
	return ds
}

// --- bateria ---

func lerBateria() *Bateria {
	pasta := coleta.PastaDaBateria()
	if pasta == "" {
		return nil
	}
	b := &Bateria{Modelo: texto(arquivo(filepath.Join(pasta, "model_name")))}
	if n, ok := inteiro(filepath.Join(pasta, "cycle_count")); ok && n >= 0 {
		b.Ciclos = coleta.Ptr(int(n))
	}
	// energia em µWh; nas que só dão carga (µAh), vezes a tensão de projeto
	cheia, okCheia := inteiro(filepath.Join(pasta, "energy_full"))
	projeto, okProjeto := inteiro(filepath.Join(pasta, "energy_full_design"))
	fator := 1e-6
	if !okCheia {
		cheia, okCheia = inteiro(filepath.Join(pasta, "charge_full"))
		projeto, okProjeto = inteiro(filepath.Join(pasta, "charge_full_design"))
		tensao, okTensao := inteiro(filepath.Join(pasta, "voltage_min_design"))
		if !okTensao {
			tensao, okTensao = inteiro(filepath.Join(pasta, "voltage_now"))
		}
		fator = float64(tensao) * 1e-12
		okCheia = okCheia && okTensao && tensao > 0
	}
	if okCheia && cheia > 0 {
		b.CapacidadeWh = coleta.Ptr(coleta.Arred(float64(cheia)*fator, 1))
		if okProjeto && projeto > 0 {
			saude := int(math.Round(float64(cheia) / float64(projeto) * 100))
			if saude > 100 {
				saude = 100
			}
			b.SaudePct = &saude
		}
	}
	return b
}

// --- o sistema ---

func lerSistema() Sistema {
	s := Sistema{Nome: "Mk", Compositor: janelas.Compositor()}
	s.VersaoMk, s.DataMk = versaoDoMk()

	s.Kernel = texto(arquivo(base.Proc("sys", "kernel", "osrelease")))
	if s.Kernel == nil {
		// "Linux version 7.3.5-arch1-1 (...)"
		if c := strings.Fields(arquivo(base.Proc("version"))); len(c) >= 3 && c[1] == "version" {
			s.Kernel = &c[2]
		}
	}
	if c := strings.Fields(arquivo(base.Proc("uptime"))); len(c) > 0 {
		if seg, err := strconv.ParseFloat(c[0], 64); err == nil && seg >= 0 && !math.IsInf(seg, 0) {
			s.LigadoHaS = coleta.Ptr(int64(seg))
		}
	}
	// o machine-id é escrito na instalação e não muda mais
	if info, err := os.Stat(base.Etc("machine-id")); err == nil {
		s.InstaladoEmS = coleta.Ptr(info.ModTime().Unix())
	}
	pacman := os.Getenv("MKPULSO_PACMAN")
	if pacman == "" {
		pacman = "/var/lib/pacman/local"
	}
	if entradas, err := os.ReadDir(pacman); err == nil {
		n := 0
		for _, e := range entradas {
			if e.IsDir() {
				n++
			}
		}
		s.Pacotes = &n
	}
	return s
}

// versaoDoMk lê o VERSION ("commit data", uma linha): o de ~/.config/mk, que
// o mk-update grava; sem ele (ou com ele vazio), o de /usr/share/mk, que veio
// na ISO (pasta trocável por MKPULSO_MK).
func versaoDoMk() (versao, data *string) {
	doSistema := os.Getenv("MKPULSO_MK")
	if doSistema == "" {
		doSistema = "/usr/share/mk"
	}
	for _, caminho := range []string{base.Home(".config", "mk", "VERSION"), filepath.Join(doSistema, "VERSION")} {
		linha, _, _ := strings.Cut(arquivo(caminho), "\n")
		c := strings.Fields(linha)
		if len(c) == 0 {
			continue
		}
		versao = &c[0]
		if len(c) > 1 {
			data = &c[1]
		}
		return versao, data
	}
	return nil, nil
}
