package main

import (
	"strings"
	"testing"

	"mkpacotes/internal/dubles"
)

func num(v int) *int { return &v }

func join(l []string) string { return strings.Join(l, " ") }

func TestDetalhe_Contrato_Instalado(t *testing.T) {
	maquina(t)
	quer(t, 0, `{"nome":"bash","versao":"5.3.20-1","descricao":"The GNU Bourne Again shell","origem":"core",
	 "instalado":true,"motivo":"pedido","orfao":false,"protegido":false,"atualizacao":"5.3.21-1",
	 "url":"https://www.gnu.org/software/bash/","licencas":["GPL-3.0-or-later"],"mantenedor":"Fulano <fulano@archlinux.org>",
	 "tamanho_bytes":10000000,"baixar_bytes":null,"instalado_em_s":1792000100,"construido_em_s":1789000000,
	 "depende":[{"nome":"glibc","pede":"glibc>=2.38","instalado":true},{"nome":"ncurses","pede":"ncurses","instalado":false}],
	 "opcionais":[{"nome":"bash-completion","para":"for tab completion","instalado":false}],
	 "fornece":["sh"],"conflita":[],
	 "requerido_por":["do-aur","mkinitcpio"],"opcional_para":[],
	 "arquivos":{"total":2,"executaveis":["/usr/bin/bash"]},
	 "aur":null,"erro_aur":null}`, "detalhe", "bash")
}

func TestDetalhe_Contrato_DoRepositorioSemEstarInstalado(t *testing.T) {
	maquina(t)
	quer(t, 0, `{"nome":"fd","versao":"10.2.0-1","descricao":"Simple, fast alternative to find","origem":"extra",
	 "instalado":false,"motivo":null,"orfao":false,"protegido":false,"atualizacao":null,
	 "url":"https://github.com/sharkdp/fd","licencas":["MIT","Apache-2.0"],"mantenedor":"Beltrano <b@archlinux.org>",
	 "tamanho_bytes":4194304,"baixar_bytes":1048576,"instalado_em_s":null,"construido_em_s":null,
	 "depende":[{"nome":"gcc-libs","pede":"gcc-libs","instalado":false},{"nome":"glibc","pede":"glibc","instalado":true}],
	 "opcionais":[],"fornece":[],"conflita":[],"requerido_por":[],"opcional_para":[],
	 "arquivos":null,"aur":null,"erro_aur":null}`, "detalhe", "fd")
}

func TestDetalhe_Contrato_DoAUR(t *testing.T) {
	maquina(t)
	// Instalado e do AUR: a ficha local com o que o AUR diz dele.
	codigo, linhas := roda(t, "detalhe", "do-aur")
	r := objeto(t, linhas[0])
	if codigo != 0 || r["origem"] != "aur" || r["instalado"] != true || r["versao"] != "1.0-1" || r["erro_aur"] != nil {
		t.Fatalf("veio %v", linhas)
	}
	querAur := `{"versao":"1.1-1","votos":12,"popularidade":0.4,"desatualizado_desde_s":1794000000,"mantenedor":"fulano","modificado_em_s":1795000000}`
	if a := campo(r, "aur"); a == nil || campo(a, "votos") != float64(12) || campo(a, "versao") != "1.1-1" || campo(a, "mantenedor") != "fulano" ||
		campo(a, "desatualizado_desde_s") != float64(1794000000) || len(a.(map[string]any)) != 6 {
		t.Errorf("aur: %v, queria %s", a, querAur)
	}
	// Não instalado: tudo vem do AUR.
	quer(t, 0, `{"nome":"top-do-aur","versao":"2-1","descricao":"Outro top","origem":"aur",
	 "instalado":false,"motivo":null,"orfao":false,"protegido":false,"atualizacao":null,
	 "url":"","licencas":[],"mantenedor":null,
	 "tamanho_bytes":null,"baixar_bytes":null,"instalado_em_s":null,"construido_em_s":1795000001,
	 "depende":[{"nome":"glibc","pede":"glibc","instalado":true},{"nome":"libfalta","pede":"libfalta","instalado":false}],
	 "opcionais":[],"fornece":[],"conflita":[],"requerido_por":[],"opcional_para":[],"arquivos":null,
	 "aur":{"versao":"2-1","votos":3,"popularidade":0.1,"desatualizado_desde_s":null,"mantenedor":null,"modificado_em_s":1795000001},
	 "erro_aur":null}`, "detalhe", "top-do-aur")
}

func TestDetalhe_SemRedeAFichaLocalSaiAssimMesmo(t *testing.T) {
	maquina(t)
	semRede(t)
	codigo, linhas := roda(t, "detalhe", "do-aur")
	r := objeto(t, linhas[0])
	if codigo != 0 || r["nome"] != "do-aur" || r["aur"] != nil || r["erro_aur"] != "não consegui falar com o AUR (sem rede?)" {
		t.Errorf("veio %v", linhas)
	}
	// O que não está instalado nem nos repositórios não dá para saber sem rede.
	querErro(t, 1, "não consegui falar com o AUR", "detalhe", "top-do-aur")
}

func TestDetalhe_OrfaoEProtegido(t *testing.T) {
	maquina(t)
	_, linhas := roda(t, "detalhe", "libvelha")
	if r := objeto(t, linhas[0]); r["orfao"] != true || r["motivo"] != "dependencia" {
		t.Errorf("libvelha: %v", linhas)
	}
	_, linhas = roda(t, "detalhe", "glibc")
	if r := objeto(t, linhas[0]); r["protegido"] != true || len(r["requerido_por"].([]any)) != 3 {
		t.Errorf("glibc: %v", linhas)
	}
}

func TestDetalhe_ErrosEUso(t *testing.T) {
	m := maquina(t)
	querErro(t, 1, "não achei nao-existe", "detalhe", "nao-existe")
	antes := len(m.Chamadas("pacman"))
	querErro(t, 1, "uso:", "detalhe")
	querErro(t, 1, "uso:", "detalhe", "a", "b")
	querErro(t, 1, "não conheço a opção", "detalhe", "--help")
	querErro(t, 1, "não é nome de pacote", "detalhe", "--", "-Si")
	querErro(t, 1, "não é nome de pacote", "detalhe", "../../etc/passwd")
	querErro(t, 1, "não é nome de pacote", "detalhe", "a b")
	if len(m.Chamadas("pacman")) != antes {
		t.Errorf("chegou ao pacman: %v", m.Chamadas("pacman"))
	}
}

func TestDetalhe_SemOPacmanOInstaladoSaiSemOrigem(t *testing.T) {
	maquina(t)
	dubles.Nova(t)
	codigo, linhas := roda(t, "detalhe", "bash")
	if r := objeto(t, linhas[0]); codigo != 0 || r["origem"] != nil || r["nome"] != "bash" {
		t.Errorf("veio %v", linhas)
	}
}
