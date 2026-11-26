package busca

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"mkfiles/internal/mime"
	"mkfiles/internal/prova"
)

func motor(t *testing.T) *prova.Motor {
	t.Helper()
	m := prova.Novo(t, Ligar)
	m.Grava(".mime/globs2", "50:text/plain:*.txt\n50:image/jpeg:*.jpg\n50:image/png:*.png\n50:video/mp4:*.mp4\n50:audio/flac:*.flac\n50:application/pdf:*.pdf\n")
	m.Grava(".mime/generic-icons", "")
	t.Setenv("MKFILES_MIME", m.Casa+"/.mime")
	mime.Zerar()
	t.Cleanup(mime.Zerar)
	return m
}

// busca roda um buscar na pasta "b" e devolve os caminhos achados (sem a
// pasta), em ordem, e a última linha.
func busca(t *testing.T, m *prova.Motor, campos ...any) ([]string, prova.Linha) {
	t.Helper()
	p := map[string]any{"op": "buscar", "caminho": m.Casa + "/b"}
	for i := 0; i+1 < len(campos); i += 2 {
		p[campos[i].(string)] = campos[i+1]
	}
	linhas := m.PedeFluxo(prova.J(p))
	var achados []string
	for i, l := range linhas {
		if l["erro"] != nil {
			t.Fatalf("buscar %v: %v", campos, l)
		}
		if (l["fim"] == true) != (i == len(linhas)-1) {
			t.Fatalf("só a última linha tem fim: true: %v", linhas)
		}
		for _, it := range l["itens"].([]any) {
			achados = append(achados, strings.TrimPrefix(it.(map[string]any)["caminho"].(string), m.Casa+"/b/"))
		}
	}
	sort.Strings(achados)
	return achados, linhas[len(linhas)-1]
}

func confere(t *testing.T, achados []string, quer ...string) {
	t.Helper()
	sort.Strings(quer)
	if strings.Join(achados, "|") != strings.Join(quer, "|") {
		t.Errorf("achou %v; esperado %v", achados, quer)
	}
}

func TestNormalECasa(t *testing.T) {
	casos := map[string]string{
		"Relatório": "relatorio", "AÇÃO": "acao", "já é": "ja e", "simples.txt": "simples.txt", "Größe": "grosse",
		"café": "cafe", "Ñandú": "nandu", "ŁÓDŹ": "lodz", "日本": "日本", "": "",
	}
	for de, para := range casos {
		if Normal(de) != para {
			t.Errorf("Normal(%q) = %q; esperado %q", de, Normal(de), para)
		}
	}
	for _, c := range []struct {
		padrao, nome string
		casa         bool
	}{
		{"*.jpg", "foto.jpg", true}, {"*.jpg", "foto.jpeg", false}, {"foto?.jpg", "foto1.jpg", true}, {"foto?.jpg", "foto.jpg", false},
		{"foto?.jpg", "fotoç.jpg", true}, {"*", "", true}, {"*", "x", true}, {"a*b*c", "aXXbYYc", true}, {"a*b*c", "aXXbYY", false},
		{"rel*", "relatorio", true}, {"rel*", "o relatorio", false}, {"?", "", false}, {"", "", true}, {"", "x", false},
		{"*a*a*a*b", strings.Repeat("a", 200), false},
	} {
		if Casa(c.padrao, c.nome) != c.casa {
			t.Errorf("Casa(%q, %q) devia ser %v", c.padrao, c.nome, c.casa)
		}
	}
}

func TestBuscaPorNome(t *testing.T) {
	m := motor(t)
	m.Grava("b/Relatório final.txt", "x")
	m.Grava("b/sub/RELATORIO.pdf", "x")
	m.Grava("b/sub/fundo/meu relatório.txt", "x")
	m.Grava("b/outra coisa.txt", "x")
	m.Pasta("b/relatorios")
	m.Grava("fora/relatorio.txt", "x")

	achados, fim := busca(t, m, "nome", "relatorio")
	confere(t, achados, "Relatório final.txt", "sub/RELATORIO.pdf", "sub/fundo/meu relatório.txt", "relatorios")
	if fim["total"] != float64(4) || fim["vistos"] != float64(7) || fim["cancelada"] != nil {
		t.Errorf("a última linha: %v", fim)
	}
	// Com acento e maiúsculas no pedido dá no mesmo.
	achados, _ = busca(t, m, "nome", " RELATÓRIO ")
	if len(achados) != 4 {
		t.Errorf("acentos e maiúsculas no pedido: %v", achados)
	}
	// Com curinga é o nome inteiro.
	achados, _ = busca(t, m, "nome", "relat*")
	confere(t, achados, "Relatório final.txt", "sub/RELATORIO.pdf", "relatorios")
	achados, _ = busca(t, m, "nome", "*.TXT")
	confere(t, achados, "Relatório final.txt", "sub/fundo/meu relatório.txt", "outra coisa.txt")
	achados, _ = busca(t, m, "nome", "relatorio?")
	confere(t, achados, "relatorios")
	// Sem nome: tudo.
	if achados, _ = busca(t, m); len(achados) != 7 {
		t.Errorf("sem nome: %v", achados)
	}
	if achados, _ = busca(t, m, "nome", "nao-ha-nada-assim"); len(achados) != 0 {
		t.Errorf("nada: %v", achados)
	}
}

func TestFiltros(t *testing.T) {
	m := motor(t)
	m.Grava("b/foto.jpg", strings.Repeat("j", 3000))
	m.Grava("b/desenho.png", "p")
	m.Grava("b/filme.mp4", strings.Repeat("v", 5000))
	m.Grava("b/musica.flac", "f")
	m.Grava("b/nota.txt", "texto")
	m.Grava("b/manual.pdf", "%PDF-")
	m.Grava("b/script", "#!/bin/sh\n")
	m.Pasta("b/pasta.jpg")
	os.Symlink("foto.jpg", m.Casa+"/b/atalho.jpg")
	velho := time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC)
	os.Chtimes(m.Casa+"/b/nota.txt", velho, velho)

	por := func(tipo string, quer ...string) {
		t.Helper()
		achados, _ := busca(t, m, "tipo", tipo)
		confere(t, achados, quer...)
	}
	por("imagem", "foto.jpg", "desenho.png", "atalho.jpg")
	por("video", "filme.mp4")
	por("audio", "musica.flac")
	por("texto", "nota.txt")
	por("documento", "nota.txt", "manual.pdf")
	por("pasta", "pasta.jpg")
	por("arquivo", "foto.jpg", "desenho.png", "filme.mp4", "musica.flac", "nota.txt", "manual.pdf", "script")
	if achados, _ := busca(t, m, "tipo", "qualquer"); len(achados) != 9 {
		t.Errorf("qualquer: %v", achados)
	}

	achados, _ := busca(t, m, "tamanho_min", 3000)
	confere(t, achados, "foto.jpg", "filme.mp4")
	achados, _ = busca(t, m, "tamanho_max", 1)
	confere(t, achados, "desenho.png", "musica.flac")
	achados, _ = busca(t, m, "tamanho_min", 2, "tamanho_max", 3000)
	confere(t, achados, "foto.jpg", "nota.txt", "manual.pdf", "script")
	achados, _ = busca(t, m, "antes_s", velho.Unix())
	confere(t, achados, "nota.txt")
	achados, _ = busca(t, m, "depois_s", velho.Unix()+1, "nome", "*.txt")
	confere(t, achados)
	achados, _ = busca(t, m, "depois_s", velho.Unix()-1, "antes_s", velho.Unix()+1)
	confere(t, achados, "nota.txt")
	// Os filtros se somam.
	achados, _ = busca(t, m, "nome", "*.jpg", "tipo", "imagem", "tamanho_min", 100)
	confere(t, achados, "foto.jpg")
}

func TestConteudo(t *testing.T) {
	m := motor(t)
	m.Grava("b/carta.txt", "Prezada Conceição,\nsegue o RELATÓRIO.")
	m.Grava("b/sem-extensao", "aqui também tem relatorio")
	m.Grava("b/outra.txt", "nada a ver")
	m.Grava("b/binario.bin", "relatorio\x00\x01\x02")
	m.Grava("b/relatorio no nome.txt", "mas não dentro")
	m.Pasta("b/relatorio")
	grande := strings.Repeat("a", limiteDoConteudo) + " relatorio"
	m.Grava("b/grande.txt", grande)
	m.Grava("b/no-limite.txt", grande[len(" relatorio"):])
	os.Symlink("carta.txt", m.Casa+"/b/atalho.txt")

	achados, _ := busca(t, m, "conteudo", "relatório")
	confere(t, achados, "carta.txt", "sem-extensao", "no-limite.txt")
	achados, _ = busca(t, m, "conteudo", "CONCEICAO")
	confere(t, achados, "carta.txt")
	achados, _ = busca(t, m, "conteudo", "relatorio", "nome", "*.txt", "tamanho_max", 1000)
	confere(t, achados, "carta.txt")
	// Conteúdo vazio não filtra.
	if achados, _ = busca(t, m, "conteudo", ""); len(achados) != 9 {
		t.Errorf("conteúdo vazio: %v", achados)
	}
}

func TestOcultosLinksEDiscos(t *testing.T) {
	m := motor(t)
	m.Grava("b/visivel.txt", "x")
	m.Grava("b/.oculto.txt", "x")
	m.Grava("b/.pasta-oculta/dentro.txt", "x")
	m.Grava("b/escondida/dentro.txt", "x")
	m.Grava("b/escondido.txt", "x")
	m.Grava("b/.hidden", "escondida\nescondido.txt\n")
	m.Grava("fora/la-fora.txt", "x")
	os.Symlink(m.Casa+"/fora", m.Casa+"/b/link-para-fora")
	m.Grava("b/outro-disco/no-disco.txt", "x")
	m.Grava("b/sub/outro-disco/tambem.txt", "x")

	achados, _ := busca(t, m, "nome", "*.txt")
	confere(t, achados, "visivel.txt", "outro-disco/no-disco.txt", "sub/outro-disco/tambem.txt")
	achados, _ = busca(t, m, "nome", "*.txt", "ocultos", true)
	confere(t, achados, "visivel.txt", ".oculto.txt", ".pasta-oculta/dentro.txt", "escondida/dentro.txt", "escondido.txt",
		"outro-disco/no-disco.txt", "sub/outro-disco/tambem.txt")
	if achados, _ = busca(t, m, "nome", "la-fora", "ocultos", true); len(achados) != 0 {
		t.Errorf("a busca não segue links: %v", achados)
	}
	if achados, _ = busca(t, m, "nome", "link-para"); len(achados) != 1 {
		t.Errorf("mas o link em si aparece: %v", achados)
	}

	// Outro sistema de arquivos: a pasta aparece, mas a busca não entra sem pedir.
	antigo := discoDe
	t.Cleanup(func() { discoDe = antigo })
	discoDe = func(caminho string, info fs.FileInfo) uint64 {
		if filepath.Base(caminho) == "outro-disco" {
			return 99
		}
		return 1
	}
	achados, _ = busca(t, m)
	confere(t, achados, "visivel.txt", "link-para-fora", "outro-disco", "sub", "sub/outro-disco")
	achados, _ = busca(t, m, "nome", "*.txt", "outros_discos", true)
	confere(t, achados, "visivel.txt", "outro-disco/no-disco.txt", "sub/outro-disco/tambem.txt")
}

func TestLotesECancelar(t *testing.T) {
	m := motor(t)
	for i := 0; i < 250; i++ {
		m.Grava(fmt.Sprintf("b/p%02d/arquivo-%03d.txt", i%10, i), "x")
	}
	p := prova.J(map[string]any{"op": "buscar", "caminho": m.Casa + "/b", "nome": "arquivo"})
	antigo := esperaDoLote
	t.Cleanup(func() { esperaDoLote = antigo })
	esperaDoLote = time.Hour
	linhas := m.PedeFluxo(p)
	if len(linhas) != 3 || len(linhas[0]["itens"].([]any)) != 100 || len(linhas[2]["itens"].([]any)) != 50 || linhas[2]["total"] != float64(250) {
		t.Errorf("lotes de 100: %d linhas", len(linhas))
	}
	// Com o tempo do lote zerado, cada achado sai na hora: dá para cancelar no meio.
	esperaDoLote = 0
	id := m.Manda(p)
	if n := len(m.Resposta(id)["itens"].([]any)); n != 1 {
		t.Errorf("em fluxo: %d", n)
	}
	if r := m.Pede(fmt.Sprintf(`{"op":"cancelar","pedido":%d}`, id)); r["cancelado"] != true {
		// A busca pode ter acabado antes: aí não há o que conferir.
		t.Skipf("a busca acabou antes do cancelar: %v", r)
	}
	resto := m.Fluxo(id)
	if ultimo := resto[len(resto)-1]; ultimo["cancelada"] != true || ultimo["fim"] != true {
		t.Errorf("cancelar: %v", ultimo)
	}
}

func TestErros(t *testing.T) {
	m := motor(t)
	arq := m.Grava("b/a.txt", "x")
	for _, c := range []struct{ pedido, motivo string }{
		{`{"op":"buscar","caminho":"relativo"}`, ""},
		{`{"op":"buscar","caminho":` + prova.J(m.Casa+"/b/../b") + `}`, ""},
		{`{"op":"buscar","caminho":"/a\u0000b"}`, ""},
		{`{"op":"buscar"}`, ""},
		{`{"op":"buscar","caminho":` + prova.J(m.Casa+"/nao-ha") + `}`, "nao-existe"},
		{`{"op":"buscar","caminho":` + prova.J(arq) + `}`, "nao-e-pasta"},
		{`{"op":"buscar","caminho":` + prova.J(m.Casa+"/b") + `,"tipo":"planilha"}`, ""},
	} {
		r := m.Pede(c.pedido)
		if r["erro"] == nil || (c.motivo != "" && r["motivo"] != c.motivo) {
			t.Errorf("%s: %v", c.pedido, r)
		}
	}
	// Pasta sem permissão no meio: a busca segue com o resto.
	if os.Getuid() != 0 {
		fechada := m.Pasta("b/fechada")
		m.Grava("b/fechada/x.txt", "x")
		os.Chmod(fechada, 0)
		t.Cleanup(func() { os.Chmod(fechada, 0o755) })
		achados, _ := busca(t, m, "nome", "*.txt")
		confere(t, achados, "a.txt")
	}
}
