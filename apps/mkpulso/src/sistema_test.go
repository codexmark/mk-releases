package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"mkpulso/internal/base"
	"mkpulso/internal/coleta/falso"
)

// fichaDeMentira monta a máquina do exemplo do contrato.
func fichaDeMentira(t *testing.T) {
	t.Helper()
	r := falso.Raizes(t)
	e := func(caminho, texto string) { falso.Escreve(t, r, caminho, texto) }
	e("proc/sys/kernel/hostname", "nome\n")
	e("proc/sys/kernel/osrelease", "7.3.5-arch1-1\n")
	e("proc/uptime", "3600.42 12000.00\n")
	e("proc/cpuinfo", "processor\t: 0\nmodel name\t: Intel(R) Core(TM) i5-6300U CPU @ 2.40GHz\nphysical id\t: 0\ncore id\t\t: 0\n\nprocessor\t: 1\nmodel name\t: Intel(R) Core(TM) i5-6300U CPU @ 2.40GHz\nphysical id\t: 0\ncore id\t\t: 0\n")
	e("proc/meminfo", "MemTotal:        8000000 kB\n")
	e("sys/class/dmi/id/sys_vendor", "HP\n")
	e("sys/class/dmi/id/product_name", "HP EliteBook 840 G3\n")
	e("sys/class/dmi/id/board_vendor", "HP\n")
	e("sys/class/dmi/id/board_name", "8079\n")
	e("sys/class/dmi/id/bios_version", "N75 Ver. 01.60\n")
	e("sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_max_freq", "3000000\n")
	e("sys/class/drm/card1/device/uevent", "DRIVER=i915\nPCI_ID=8086:1916\n")
	e("sys/block/nvme0n1/size", "1000000000\n")
	e("sys/block/nvme0n1/device/model", "Samsung 970\n")
	e("sys/block/nvme0n1/queue/rotational", "0\n")
	e("sys/class/power_supply/BAT0/model_name", "Primary\n")
	e("sys/class/power_supply/BAT0/cycle_count", "984\n")
	e("sys/class/power_supply/BAT0/energy_full", "33100000\n")
	e("sys/class/power_supply/BAT0/energy_full_design", "46620000\n")
	e("home/.config/mk/VERSION", "4c4ea04fc26e 2026-11-23\n")
	e("pacman/bash-5.3-1/desc", "x\n")
	e("etc/machine-id", "abc\n")
	quando := time.Unix(1795000000, 0)
	if err := os.Chtimes(filepath.Join(r, "etc/machine-id"), quando, quando); err != nil {
		t.Fatal(err)
	}
	falso.Comando(t, r, "hyprctl", `case "$1" in
monitors) echo '[{"name": "eDP-1", "description": "BOE 0x0697", "width": 1366, "height": 768, "refreshRate": 60.012, "scale": 1.0}]' ;;
version) echo '{"version": "0.57.2", "tag": "v0.57.2"}' ;;
*) exit 1 ;;
esac`)
}

func TestSistemaSegueOContrato(t *testing.T) {
	fichaDeMentira(t)
	codigo, saida := rodaComSaida(t, "sistema")
	if codigo != base.Certo {
		t.Fatalf("código %d, saída %s", codigo, saida)
	}
	ficha := umaLinhaDeJSON(t, saida)
	// a saúde do disco é o único campo que o contrato deixa null aqui
	confereForma(t, "ficha", exemploDoContrato(t, "sistema"), ficha, map[string]bool{"ficha.discos[].saude": true}, nil)
	for caminho, quer := range map[string]any{"maquina": "nome", "modelo": "HP EliteBook 840 G3", "placa": "HP 8079", "firmware": "N75 Ver. 01.60", "mem_total_kb": 8000000.0} {
		if ficha[caminho] != quer {
			t.Errorf("%s = %v, esperava %v", caminho, ficha[caminho], quer)
		}
	}
	if v := cava(t, ficha, "sistema", "compositor"); v != "Hyprland 0.57.2" {
		t.Errorf("compositor = %v", v)
	}
	if v := cava(t, ficha, "discos", 0, "saude"); v != nil {
		t.Errorf("sem root não há como saber a saúde do disco: %v", v)
	}
}

func TestSistemaNumaMaquinaQueNaoDizNada(t *testing.T) {
	falso.Raizes(t)
	t.Setenv("MKPULSO_PACMAN", "/nao/existe")
	codigo, saida := rodaComSaida(t, "sistema")
	if codigo != base.Certo {
		t.Fatalf("código %d, saída %s", codigo, saida)
	}
	ficha := umaLinhaDeJSON(t, saida)
	// os mesmos campos do contrato, com null ou lista vazia em tudo
	tudo := map[string]bool{}
	for _, c := range []string{"maquina", "modelo", "placa", "firmware", "mem_total_kb", "bateria", "cpu.nome", "cpu.nucleos", "cpu.threads", "cpu.max_mhz",
		"sistema.versao_mk", "sistema.data_mk", "sistema.kernel", "sistema.compositor", "sistema.ligado_ha_s", "sistema.instalado_em_s", "sistema.pacotes"} {
		tudo["ficha."+c] = true
	}
	confereForma(t, "ficha", exemploDoContrato(t, "sistema"), ficha, tudo, map[string]bool{"ficha.gpus": true, "ficha.discos": true, "ficha.telas": true})
	if ficha["maquina"] != nil || ficha["bateria"] != nil || cava(t, ficha, "cpu", "nucleos") != nil || cava(t, ficha, "sistema", "pacotes") != nil {
		t.Errorf("o que não dá para saber é null: %s", saida)
	}
	if cava(t, ficha, "sistema", "nome") != "Mk" {
		t.Errorf("o nome do sistema é sempre Mk: %s", saida)
	}
}

func TestSistemaNaoAceitaArgumento(t *testing.T) {
	falso.Raizes(t)
	codigo, saida := rodaComSaida(t, "sistema", "--tudo")
	if obj := umaLinhaDeJSON(t, saida); codigo != base.Erro || obj["erro"] == nil {
		t.Errorf("argumento a mais é erro de uso: código %d, saída %s", codigo, saida)
	}
}
