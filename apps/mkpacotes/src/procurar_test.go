package main

import (
	"testing"

	"mkpacotes/internal/dubles"
)

func TestProcurar_Contrato(t *testing.T) {
	maquina(t)
	// "top-do-aur" começa com o texto: vem antes dos que só o contêm.
	quer(t, 0, `{"resultados":[
	 {"nome":"top-do-aur","versao":"2-1","descricao":"Outro top","origem":"aur","instalado":false,"versao_instalada":null,"votos":3,"desatualizado":false},
	 {"nome":"btop","versao":"1.4.5-1","descricao":"A monitor of resources on top","origem":"extra","instalado":false,"versao_instalada":null,"votos":null,"desatualizado":null},
	 {"nome":"htop","versao":"3.5.3-1","descricao":"Interactive process viewer","origem":"extra","instalado":true,"versao_instalada":"3.5.3-1","votos":null,"desatualizado":null}],
	 "cortado":false,"erro_oficial":null,"erro_aur":null}`, "procurar", "top")
}

func TestProcurar_SemRedeOsOficiaisContinuam(t *testing.T) {
	maquina(t)
	semRede(t)
	codigo, linhas := roda(t, "procurar", "top")
	r := objeto(t, linhas[0])
	if codigo != 0 || campo(r, "resultados", 0, "nome") != "btop" || campo(r, "resultados", 1, "nome") != "htop" || campo(r, "resultados", 2) != nil {
		t.Errorf("os oficiais tinham de vir: %d %v", codigo, linhas)
	}
	if r["erro_aur"] != "não consegui falar com o AUR (sem rede?)" || r["erro_oficial"] != nil {
		t.Errorf("erro_aur: %v", r["erro_aur"])
	}
}

func TestProcurar_SemOPacmanOAURContinuaESemOsDoisFalha(t *testing.T) {
	maquina(t)
	dubles.Nova(t)
	codigo, linhas := roda(t, "procurar", "top")
	r := objeto(t, linhas[0])
	if codigo != 0 || campo(r, "resultados", 0, "nome") != "top-do-aur" || r["erro_oficial"] != "não achei o pacman nesta máquina" {
		t.Errorf("veio %d %v", codigo, linhas)
	}
	semRede(t)
	querErro(t, 1, "não achei o pacman nesta máquina", "procurar", "top")
}

func TestProcurar_SemAurNaoSaiDaMaquina(t *testing.T) {
	maquina(t)
	semRede(t)
	codigo, linhas := roda(t, "procurar", "top", "--sem-aur")
	r := objeto(t, linhas[0])
	if codigo != 0 || r["erro_aur"] != nil || campo(r, "resultados", 0, "nome") != "btop" || campo(r, "resultados", 2) != nil {
		t.Errorf("veio %d %v", codigo, linhas)
	}
}

func TestProcurar_OrdemONomeExatoNaFrenteEOAURPorVotos(t *testing.T) {
	achados := []achado{
		{Nome: "zz-top", Origem: "aur", Votos: num(1)}, {Nome: "top-b", Origem: "aur", Votos: num(5)}, {Nome: "top-a", Origem: "aur", Votos: num(50)},
		{Nome: "top-z", Origem: "extra"}, {Nome: "atop", Origem: "extra"}, {Nome: "top", Origem: "aur", Votos: num(0)}, {Nome: "top-c", Origem: "core"},
	}
	ordena(achados, "top")
	var nomes []string
	for _, a := range achados {
		nomes = append(nomes, a.Nome)
	}
	if quer := "top top-c top-z top-a top-b atop zz-top"; join(nomes) != quer {
		t.Errorf("veio %s\nquer %s", join(nomes), quer)
	}
}

func TestProcurar_NadaAchadoEListaVazia(t *testing.T) {
	maquina(t)
	quer(t, 0, `{"resultados":[],"cortado":false,"erro_oficial":null,"erro_aur":null}`, "procurar", "xyzzy")
}

func TestProcurar_VariasPalavrasFiltramTambemOAUR(t *testing.T) {
	maquina(t)
	// O AUR é perguntado pela palavra mais comprida; a outra filtra aqui.
	codigo, linhas := roda(t, "procurar", "outro", "top")
	r := objeto(t, linhas[0])
	if codigo != 0 || campo(r, "resultados", 0, "nome") != "top-do-aur" || campo(r, "resultados", 1) != nil {
		t.Errorf("veio %v", linhas)
	}
	_, linhas = roda(t, "procurar", "banana", "top")
	if campo(objeto(t, linhas[0]), "resultados", 0) != nil {
		t.Errorf("'banana top' não casa com nada: %v", linhas)
	}
}

func TestProcurar_ErrosEUso(t *testing.T) {
	m := maquina(t)
	antes := len(m.Chamadas("pacman"))
	querErro(t, 1, "pelo menos 2 letras", "procurar")
	querErro(t, 1, "pelo menos 2 letras", "procurar", "a")
	querErro(t, 1, "não conheço a opção", "procurar", "-Syu")
	querErro(t, 1, "não conheço a opção", "procurar", "top", "--noconfirm")
	querErro(t, 1, "comprida demais", "procurar", string(make([]byte, 300)))
	if len(m.Chamadas("pacman")) != antes {
		t.Errorf("chegou ao pacman: %v", m.Chamadas("pacman"))
	}
}

func TestProcurar_CortaEm200(t *testing.T) {
	m := maquina(t)
	texto := ""
	for i := 0; i < 250; i++ {
		texto += "extra/pacote-top-" + string(rune('a'+i%26)) + string(rune('a'+i/26)) + " 1-1\n    desc\n"
	}
	m.Comando("pacman", dubles.Regra{Contem: " -Ss ", Texto: texto}, dubles.Regra{Contem: " -Sl ", Texto: ""})
	_, linhas := roda(t, "procurar", "top", "--sem-aur")
	r := objeto(t, linhas[0])
	if n := len(r["resultados"].([]any)); n != 200 || r["cortado"] != true {
		t.Errorf("vieram %d, cortado %v", n, r["cortado"])
	}
}
