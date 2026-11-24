package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"mkpacotes/internal/dubles"
)

func TestAtualizacoes_Contrato(t *testing.T) {
	maquina(t)
	quer(t, 0, `{"atualizacoes":[{"nome":"bash","de":"5.3.20-1","para":"5.3.21-1","origem":"core"},
	  {"nome":"do-aur","de":"1.0-1","para":"1.1-1","origem":"aur"}],
	 "totais":{"oficiais":1,"aur":1},"fonte":"pacman","erro_oficial":null,"erro_aur":null}`, "atualizacoes")
}

func TestAtualizacoes_ComOCheckupdatesESemAUR(t *testing.T) {
	m := maquina(t)
	semRede(t)
	m.Comando("checkupdates", dubles.Regra{Contem: "", Texto: "htop 3.5.3-1 -> 3.6-1\n"})
	quer(t, 0, `{"atualizacoes":[{"nome":"htop","de":"3.5.3-1","para":"3.6-1","origem":"extra"}],
	 "totais":{"oficiais":1,"aur":0},"fonte":"checkupdates","erro_oficial":null,"erro_aur":null}`, "atualizacoes", "--sem-aur")
}

func TestAtualizacoes_SemRedeOsOficiaisContinuam(t *testing.T) {
	maquina(t)
	semRede(t)
	codigo, linhas := roda(t, "atualizacoes")
	r := objeto(t, linhas[0])
	if codigo != 0 || campo(r, "atualizacoes", 0, "nome") != "bash" || campo(r, "atualizacoes", 1) != nil ||
		r["erro_aur"] != "não consegui falar com o AUR (sem rede?)" {
		t.Errorf("veio %v", linhas)
	}
}

func TestAtualizacoes_SemOPacmanDizOErro(t *testing.T) {
	maquina(t)
	dubles.Nova(t)
	codigo, linhas := roda(t, "atualizacoes")
	r := objeto(t, linhas[0])
	if codigo != 0 || r["erro_oficial"] != "não achei o pacman nesta máquina" || len(r["atualizacoes"].([]any)) != 0 {
		t.Errorf("veio %v", linhas)
	}
	querErro(t, 1, "uso:", "atualizacoes", "sobra")
}

func TestHistorico_Contrato(t *testing.T) {
	maquina(t)
	quer(t, 0, `{"eventos":[
	  {"t_s":1795341600,"acao":"removido","nome":"fd","de":"10.1-1","para":null},
	  {"t_s":1795255200,"acao":"atualizado","nome":"htop","de":"3.5.2-1","para":"3.5.3-1"},
	  {"t_s":1795168800,"acao":"instalado","nome":"fd","de":null,"para":"10.1-1"}],"total":3}`, "historico")
	quer(t, 0, `{"eventos":[{"t_s":1795341600,"acao":"removido","nome":"fd","de":"10.1-1","para":null}],"total":2}`,
		"historico", "--busca", "FD", "--linhas", "1")
	quer(t, 0, `{"eventos":[],"total":0}`, "historico", "--busca", "nada")
}

func TestHistorico_ErrosEUso(t *testing.T) {
	maquina(t)
	for _, n := range []string{"0", "-1", "5001", "muitas", ""} {
		querErro(t, 1, "--linhas", "historico", "--linhas", n)
	}
	querErro(t, 1, "uso:", "historico", "sobra")
	querErro(t, 1, "não conheço a opção", "historico", "--tudo")
	t.Setenv("MKPACOTES_LOG", filepath.Join(t.TempDir(), "nada.log"))
	querErro(t, 1, "não consegui ler o registro do pacman", "historico")
}

func TestEstado_Contrato(t *testing.T) {
	m := maquina(t)
	raiz := os.Getenv("MKPACOTES_PACMAN_DB")
	quer(t, 0, `{"travado":false,"sincronizado_em_s":null,"cache":{"arquivos":0,"bytes":0},
	 "comandos":{"pacman":true,"pkexec":false,"yay":false,"mk_update":false}}`, "estado")

	os.WriteFile(filepath.Join(raiz, "db.lck"), nil, 0o644)
	for nome, quando := range map[string]int64{"core.db": 1795000000, "extra.db": 1795400000, "extra.files": 1795999999} {
		caminho := filepath.Join(raiz, "sync", nome)
		os.WriteFile(caminho, []byte("x"), 0o644)
		os.Chtimes(caminho, time.Unix(quando, 0), time.Unix(quando, 0))
	}
	os.WriteFile(filepath.Join(os.Getenv("MKPACOTES_CACHE"), "fd-10.2.0-1-x86_64.pkg.tar.zst"), make([]byte, 500), 0o644)
	m.Comando("pkexec", dubles.Regra{Contem: ""})
	m.Comando("yay", dubles.Regra{Contem: ""})
	mk := filepath.Join(os.Getenv("MKPACOTES_MK"), "bin", "mk-update")
	os.MkdirAll(filepath.Dir(mk), 0o755)
	os.WriteFile(mk, []byte("#!/bin/sh\n"), 0o755)
	quer(t, 0, `{"travado":true,"sincronizado_em_s":1795400000,"cache":{"arquivos":1,"bytes":500},
	 "comandos":{"pacman":true,"pkexec":true,"yay":true,"mk_update":true}}`, "estado")

	t.Setenv("MKPACOTES_CACHE", filepath.Join(t.TempDir(), "nada"))
	_, linhas := roda(t, "estado")
	if r := objeto(t, linhas[0]); r["cache"] != nil {
		t.Errorf("sem a pasta, cache é null: %v", linhas)
	}
	querErro(t, 1, "uso:", "estado", "sobra")
	if len(m.Chamadas("pkexec")) != 0 {
		t.Errorf("o estado não roda nada")
	}
}

func TestPlano_Contrato(t *testing.T) {
	m := maquina(t)
	quer(t, 0, `{"acao":"remover","pedidos":["htop"],"entra":[],
	 "sai":[{"nome":"htop","versao":"3.5.3-1","tamanho_bytes":400000,"pedido":true},{"nome":"libnl","versao":"3.11.0-1","tamanho_bytes":1500000,"pedido":false}],
	 "baixar_bytes":null,"liberar_bytes":1900000,"comandos":[["pkexec","pacman","-Rns","--noconfirm","--","htop"]],
	 "avisos":[],"travado":false,"nada_a_fazer":false,"cache":null}`, "plano", "remover", "htop")
	quer(t, 0, `{"acao":"instalar","pedidos":["fd"],
	 "entra":[{"nome":"fd","versao":"10.2.0-1","origem":"extra","baixar_bytes":1048576,"pedido":true}],"sai":[],
	 "baixar_bytes":1048576,"liberar_bytes":null,"comandos":[["pkexec","pacman","-S","--needed","--noconfirm","--","fd"]],
	 "avisos":[],"travado":false,"nada_a_fazer":false,"cache":null}`, "plano", "instalar", "fd")
	quer(t, 0, `{"acao":"orfaos","pedidos":["libvelha"],"entra":[],
	 "sai":[{"nome":"libvelha","versao":"0.9-2","tamanho_bytes":1000,"pedido":true}],
	 "baixar_bytes":null,"liberar_bytes":1000,"comandos":[["pkexec","pacman","-Rns","--noconfirm","--","libvelha"]],
	 "avisos":[],"travado":false,"nada_a_fazer":false,"cache":null}`, "plano", "orfaos")
	quer(t, 0, `{"acao":"cache","pedidos":[],"entra":[],"sai":[],"baixar_bytes":null,"liberar_bytes":0,"comandos":[],
	 "avisos":[],"travado":false,"nada_a_fazer":true,"cache":{"arquivos":0,"bytes":0,"sobram":0}}`, "plano", "cache")
	// O plano não muda nada: nem pkexec, nem yay.
	if len(m.Chamadas("pkexec")) != 0 || len(m.Chamadas("yay")) != 0 {
		t.Errorf("o plano rodou comando de ação")
	}
}

func TestPlano_RecusasEErros(t *testing.T) {
	m := maquina(t)
	// Protegido: código 2; com --mesmo-assim sai, com aviso.
	querErro(t, 2, "não removo mkinitcpio", "plano", "remover", "mkinitcpio")
	codigo, linhas := roda(t, "plano", "remover", "mkinitcpio", "--mesmo-assim")
	if r := objeto(t, linhas[0]); codigo != 0 || len(r["avisos"].([]any)) != 1 {
		t.Errorf("--mesmo-assim: %d %v", codigo, linhas)
	}
	// O que nem chega ao pacman: código 1.
	antes := len(m.Chamadas("pacman"))
	querErro(t, 1, "não conheço a opção", "plano", "remover", "-Rdd")
	querErro(t, 1, "não conheço a opção", "plano", "instalar", "--overwrite", "*")
	querErro(t, 1, "não é nome de pacote", "plano", "remover", "--", "-Rdd")
	querErro(t, 1, "não é nome de pacote", "plano", "instalar", "fd;reboot")
	querErro(t, 1, "não é nome de pacote", "plano", "instalar", "FD")
	querErro(t, 1, "não conheço a ação", "plano", "explodir")
	querErro(t, 1, "uso:", "plano")
	querErro(t, 1, "faltou dizer o pacote", "plano", "instalar")
	querErro(t, 1, "htop depende dele", "plano", "remover", "libnl")
	querErro(t, 1, "fd não está instalado", "plano", "remover", "fd")
	if len(m.Chamadas("pacman")) != antes {
		t.Errorf("chegou ao pacman: %v", m.Chamadas("pacman")[antes:])
	}
}

func TestAgir_Contrato(t *testing.T) {
	m := maquina(t)
	m.Comando("pkexec", dubles.Regra{Contem: "", Texto: "removendo htop...\nremovendo libnl...\n"})
	codigo, linhas := roda(t, "agir", "remover", "htop")
	quer := []string{`{"linha":"$ pkexec pacman -Rns --noconfirm -- htop"}`, `{"linha":"removendo htop..."}`, `{"linha":"removendo libnl..."}`,
		`{"ok":true,"acao":"remover","pacotes":["htop"]}`}
	if codigo != 0 || !reflect.DeepEqual(linhas, quer) {
		t.Errorf("código %d\n veio %v\n quer %v", codigo, linhas, quer)
	}
	if c := m.Chamadas("pkexec"); !reflect.DeepEqual(c, []string{"pacman -Rns --noconfirm -- htop"}) {
		t.Errorf("pkexec: %v", c)
	}
	codigo, linhas = roda(t, "agir", "instalar", "fd")
	if codigo != 0 || linhas[len(linhas)-1] != `{"ok":true,"acao":"instalar","pacotes":["fd"]}` {
		t.Errorf("instalar: %d %v", codigo, linhas)
	}
	codigo, linhas = roda(t, "agir", "orfaos")
	if codigo != 0 || linhas[len(linhas)-1] != `{"ok":true,"acao":"orfaos","pacotes":["libvelha"]}` {
		t.Errorf("orfaos: %d %v", codigo, linhas)
	}
	codigo, linhas = roda(t, "agir", "cache")
	if codigo != 0 || !reflect.DeepEqual(linhas, []string{`{"linha":"nada a fazer"}`, `{"ok":true,"acao":"cache","pacotes":[]}`}) {
		t.Errorf("cache: %d %v", codigo, linhas)
	}
}

func TestAgir_ErroNoMeioSaiComoUltimaLinha(t *testing.T) {
	m := maquina(t)
	m.Comando("pkexec", dubles.Regra{Contem: "", Codigo: 1, Texto: "conferindo...\n", Erro: "error: target not found: htop\n"})
	codigo, linhas := roda(t, "agir", "remover", "htop")
	if codigo != 1 || len(linhas) != 4 || linhas[3] != `{"erro":"não deu para remover: error: target not found: htop"}` {
		t.Errorf("código %d, %v", codigo, linhas)
	}
	m.Comando("pkexec", dubles.Regra{Contem: "", Codigo: 126})
	querErro(t, 1, "a senha não foi dada", "agir", "remover", "htop")
}

func TestAgir_RecusasDeSeguranca(t *testing.T) {
	m := maquina(t)
	m.Comando("pkexec", dubles.Regra{Contem: ""})
	m.Comando("yay", dubles.Regra{Contem: ""})
	// Pacote protegido.
	querErro(t, 2, "não removo mkinitcpio", "agir", "remover", "mkinitcpio")
	// Nome hostil: nunca chega a um comando.
	for _, nome := range []string{"-Rdd", "--noconfirm", "htop;reboot", "$(reboot)", "a b", "HTOP", "../x"} {
		codigo, linhas := roda(t, "agir", "remover", "--", nome)
		if codigo != 1 || !strings.Contains(linhas[len(linhas)-1], "não é nome de pacote") {
			t.Errorf("%q: %d %v", nome, codigo, linhas)
		}
		codigo, _ = roda(t, "agir", "instalar", nome)
		if codigo != 1 {
			t.Errorf("instalar %q: código %d", nome, codigo)
		}
	}
	// O pacman travado.
	trava := filepath.Join(os.Getenv("MKPACOTES_PACMAN_DB"), "db.lck")
	os.WriteFile(trava, nil, 0o644)
	querErro(t, 2, "o pacman está travado", "agir", "remover", "htop")
	querErro(t, 2, "o pacman está travado", "agir", "instalar", "fd")
	os.Remove(trava)
	if len(m.Chamadas("pkexec")) != 0 || len(m.Chamadas("yay")) != 0 {
		t.Errorf("uma recusa deixou rodar: pkexec %v, yay %v", m.Chamadas("pkexec"), m.Chamadas("yay"))
	}
	// Com --mesmo-assim o protegido sai (a janela pergunta duas vezes antes).
	codigo, linhas := roda(t, "agir", "remover", "mkinitcpio", "--mesmo-assim")
	if codigo != 0 || !reflect.DeepEqual(m.Chamadas("pkexec"), []string{"pacman -Rns --noconfirm -- mkinitcpio"}) {
		t.Errorf("--mesmo-assim: %d %v", codigo, linhas)
	}
	querErro(t, 1, "uso:", "agir")
	querErro(t, 1, "não conheço a ação", "agir", "explodir")
}
