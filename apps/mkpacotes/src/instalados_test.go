package main

import (
	"path/filepath"
	"testing"

	"mkpacotes/internal/dubles"
)

func TestInstalados_Contrato(t *testing.T) {
	maquina(t)
	quer(t, 0, `{"pacotes":[
	 {"nome":"bash","versao":"5.3.20-1","descricao":"The GNU Bourne Again shell","tamanho_bytes":10000000,"instalado_em_s":1792000100,
	  "motivo":"pedido","origem":"core","orfao":false,"atualizacao":"5.3.21-1","protegido":false},
	 {"nome":"do-aur","versao":"1.0-1","descricao":"Veio do AUR","tamanho_bytes":3000,"instalado_em_s":1792000600,
	  "motivo":"pedido","origem":"aur","orfao":false,"atualizacao":null,"protegido":false},
	 {"nome":"glibc","versao":"2.42-1","descricao":"GNU C Library","tamanho_bytes":50000000,"instalado_em_s":1792000000,
	  "motivo":"dependencia","origem":"core","orfao":false,"atualizacao":null,"protegido":true},
	 {"nome":"htop","versao":"3.5.3-1","descricao":"Interactive process viewer","tamanho_bytes":400000,"instalado_em_s":1792000300,
	  "motivo":"pedido","origem":"extra","orfao":false,"atualizacao":null,"protegido":false},
	 {"nome":"libnl","versao":"3.11.0-1","descricao":"Netlink","tamanho_bytes":1500000,"instalado_em_s":1792000200,
	  "motivo":"dependencia","origem":"core","orfao":false,"atualizacao":null,"protegido":false},
	 {"nome":"libvelha","versao":"0.9-2","descricao":"Sobrou","tamanho_bytes":1000,"instalado_em_s":1792000400,
	  "motivo":"dependencia","origem":"aur","orfao":true,"atualizacao":null,"protegido":false},
	 {"nome":"mkinitcpio","versao":"40-1","descricao":"Initramfs","tamanho_bytes":2000,"instalado_em_s":1792000500,
	  "motivo":"pedido","origem":"core","orfao":false,"atualizacao":null,"protegido":true}],
	 "totais":{"pacotes":7,"pedidos":4,"aur":2,"orfaos":1,"atualizacoes":1,"tamanho_bytes":61906000},
	 "aviso":null}`, "instalados")
}

func TestInstalados_SemOPacmanAListaSaiSemOrigemEComAviso(t *testing.T) {
	maquina(t)
	dubles.Nova(t) // PATH vazio: o pacman sumiu
	codigo, linhas := roda(t, "instalados")
	if codigo != 0 || len(linhas) != 1 {
		t.Fatalf("código %d, %v", codigo, linhas)
	}
	r := objeto(t, linhas[0])
	if campo(r, "pacotes", 0, "origem") != nil || campo(r, "pacotes", 0, "atualizacao") != nil || campo(r, "pacotes", 0, "nome") != "bash" {
		t.Errorf("sem o pacman, origem e atualização são null: %v", campo(r, "pacotes", 0))
	}
	if aviso, _ := r["aviso"].(string); aviso == "" {
		t.Errorf("devia vir o aviso")
	}
	if campo(r, "totais", "aur") != float64(0) || campo(r, "totais", "pacotes") != float64(7) {
		t.Errorf("totais: %v", r["totais"])
	}
}

func TestInstalados_VersaoMaisVelhaNoRepositorioNaoEAtualizacao(t *testing.T) {
	m := maquina(t)
	m.Comando("pacman", dubles.Regra{Contem: " -Sl ", Texto: "core bash 5.3.19-1\ncore glibc 2.42-1\n"})
	_, linhas := roda(t, "instalados")
	if a := campo(objeto(t, linhas[0]), "pacotes", 0, "atualizacao"); a != nil {
		t.Errorf("5.3.19 é mais velha que a instalada: %v", a)
	}
}

func TestInstalados_ErrosEUso(t *testing.T) {
	maquina(t)
	querErro(t, 1, "uso:", "instalados", "sobra")
	querErro(t, 1, "não conheço a opção", "instalados", "--tudo")
	t.Setenv("MKPACOTES_PACMAN_DB", filepath.Join(t.TempDir(), "nada"))
	querErro(t, 1, "não achei a base do pacman", "instalados")
}
