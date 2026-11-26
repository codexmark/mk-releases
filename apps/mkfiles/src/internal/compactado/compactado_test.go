package compactado

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"mkfiles/internal/mime"
	"mkfiles/internal/operar"
	"mkfiles/internal/prova"
)

// motor monta a sessão de teste: o trilho de tarefas, esta área, um banco de
// tipos pequeno e NENHUMA ferramenta de fora (cada teste põe a de mentira que
// quer). Nada aqui roda xz, zstd, 7z ou unrar de verdade.
func motor(t *testing.T) *prova.Motor {
	t.Helper()
	m := prova.Novo(t, operar.Ligar, Ligar)
	m.Grava(".mime/globs2", "50:text/plain:*.txt\n50:image/png:*.png\n")
	t.Setenv("MKFILES_MIME", m.Casa+"/.mime")
	mime.Zerar()
	t.Cleanup(mime.Zerar)
	for _, v := range []string{"XZ", "ZSTD", "7Z", "UNRAR", "BSDTAR"} {
		t.Setenv("MKFILES_"+v, m.Casa+"/nao-ha")
	}
	t.Setenv("XDG_RUNTIME_DIR", m.Pasta(".run"))
	return m
}

// fim espera a tarefa acabar e devolve o último evento dela.
func fim(m *prova.Motor, r prova.Linha) prova.Linha {
	id := r["tarefa"]
	return m.Evento("tarefa", func(l prova.Linha) bool {
		e := l["estado"]
		return l["tarefa"] == id && (e == "feita" || e == "erro" || e == "cancelada")
	})
}

func pede(t *testing.T, m *prova.Motor, pedido map[string]any) prova.Linha {
	t.Helper()
	return m.Pede(prova.J(pedido))
}

func tarefa(t *testing.T, m *prova.Motor, pedido map[string]any) prova.Linha {
	t.Helper()
	r := pede(t, m, pedido)
	if r["tarefa"] == nil {
		t.Fatalf("%v não virou tarefa: %v", pedido["op"], r)
	}
	return fim(m, r)
}

// arvore descreve uma pasta: caminho -> conteúdo ("<pasta>", "-> alvo").
func arvore(t *testing.T, raiz string) map[string]string {
	t.Helper()
	a := map[string]string{}
	filepath.WalkDir(raiz, func(c string, d fs.DirEntry, err error) error {
		if err != nil || c == raiz {
			return nil
		}
		rel, _ := filepath.Rel(raiz, c)
		switch {
		case d.Type()&fs.ModeSymlink != 0:
			alvo, _ := os.Readlink(c)
			a[rel] = "-> " + alvo
		case d.IsDir():
			a[rel] = "<pasta>"
		default:
			b, _ := os.ReadFile(c)
			a[rel] = string(b)
		}
		return nil
	})
	return a
}

func igual(t *testing.T, oQue string, obtido, esperado map[string]string) {
	t.Helper()
	if fmt.Sprint(obtido) != fmt.Sprint(esperado) {
		t.Errorf("%s:\n obtido   %v\n esperado %v", oQue, obtido, esperado)
	}
}

// semSobras confere que nenhuma pasta ou arquivo temporário do motor ficou.
func semSobras(t *testing.T, pasta string) {
	t.Helper()
	es, _ := os.ReadDir(pasta)
	for _, e := range es {
		if strings.HasPrefix(e.Name(), ".mkfiles-") {
			t.Errorf("sobrou o temporário %s em %s", e.Name(), pasta)
		}
	}
}

func erros(l prova.Linha) string {
	var s []string
	lista, _ := l["erros"].([]any)
	for _, e := range lista {
		s = append(s, e.(map[string]any)["erro"].(string))
	}
	return strings.Join(s, " | ")
}

type peca struct {
	nome, corpo string
	modo        fs.FileMode
	cifrada     bool
}

func fazZip(t *testing.T, caminho string, pecas ...peca) string {
	t.Helper()
	f, err := os.Create(caminho)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	z := zip.NewWriter(f)
	for _, p := range pecas {
		h := &zip.FileHeader{Name: p.nome, Method: zip.Deflate}
		if p.modo != 0 {
			h.SetMode(p.modo)
		}
		if p.cifrada {
			h.Flags |= 1
		}
		w, err := z.CreateHeader(h)
		if err != nil {
			t.Fatal(err)
		}
		io.WriteString(w, p.corpo)
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	return caminho
}

func fazTar(t *testing.T, cabecas ...*tar.Header) []byte {
	t.Helper()
	var b bytes.Buffer
	tw := tar.NewWriter(&b)
	for _, h := range cabecas {
		corpo := ""
		if h.Typeflag == tar.TypeReg {
			corpo = h.Linkname // os testes guardam o conteúdo aqui
			h.Linkname = ""
			h.Size = int64(len(corpo))
		}
		if h.Mode == 0 {
			h.Mode = 0o644
		}
		if err := tw.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		io.WriteString(tw, corpo)
	}
	tw.Close()
	return b.Bytes()
}

func arq(nome, corpo string) *tar.Header {
	return &tar.Header{Name: nome, Typeflag: tar.TypeReg, Linkname: corpo}
}

func script(t *testing.T, m *prova.Motor, nome, corpo string) string {
	t.Helper()
	c := m.Grava(".bin/"+nome, "#!/bin/sh\n"+corpo)
	os.Chmod(c, 0o755)
	return c
}

func TestNomeLimpoEAlvo(t *testing.T) {
	bons := map[string]string{"a.txt": "a.txt", "./a//b/": "a/b", "pasta/ç.txt": "pasta/ç.txt"}
	for cru, limpo := range bons {
		if l, ruim := nomeLimpo(cru); l != limpo || ruim != "" {
			t.Errorf("nomeLimpo(%q) = %q, %q", cru, l, ruim)
		}
	}
	for _, cru := range []string{"", ".", "/etc/passwd", "../x", "a/../../x", "a/..", "a\\b", "a\x00b", strings.Repeat("x", 256)} {
		if _, ruim := nomeLimpo(cru); ruim == "" {
			t.Errorf("nomeLimpo(%q) devia recusar", cru)
		}
	}
	casos := []struct {
		rel, alvo string
		dentro    bool
	}{
		{"a", "b", true}, {"a", "../b", false}, {"p/a", "../b", true}, {"p/a", "../../b", false},
		{"a", "/etc/passwd", false}, {"a", "", false}, {"p/q/a", "../../x/../../y", false}, {"a", "p/../b", true},
	}
	for _, c := range casos {
		if alvoDentro(c.rel, c.alvo) != c.dentro {
			t.Errorf("alvoDentro(%q, %q) devia ser %v", c.rel, c.alvo, c.dentro)
		}
	}
}

func TestFormato(t *testing.T) {
	m := motor(t)
	nomes := map[string]string{"a.zip": "zip", "A.TAR.GZ": "tar.gz", "a.tgz": "tar.gz", "a.tar.zst": "tar.zst", "a.txz": "tar.xz",
		"a.7z": "7z", "a.rar": "rar", "a.bz2": "bz2", "a.txt": "", ".zip": ""}
	for nome, formato := range nomes {
		if f, _ := peloNome(nome); f != formato {
			t.Errorf("peloNome(%q) = %q", nome, f)
		}
	}
	if _, sem := peloNome("pacote.tar.gz"); sem != "pacote" {
		t.Errorf("nome sem a extensão: %q", sem)
	}
	var gz bytes.Buffer
	w := gzip.NewWriter(&gz)
	w.Write(fazTar(t, arq("a.txt", "oi")))
	w.Close()
	casos := []struct{ nome, dados, formato string }{
		{"a.tar.gz", gz.String(), "tar.gz"},
		{"a.gz", gz.String(), "gz"},
		{"mentira.zip", gz.String(), "gz"}, // os bytes mandam na casca
		{"sem-extensao", "PK\x03\x04....", "zip"},
		{"a.tar", string(fazTar(t, arq("a.txt", "oi"))), "tar"},
		{"a.7z", "7z\xbc\xaf\x27\x1c....", "7z"},
		{"a.rar", "Rar!\x1a\x07\x00...", "rar"},
		{"a.xz", "\xfd7zXZ\x00...", "xz"},
		{"a.tar.zst", "\x28\xb5\x2f\xfd...", "tar.zst"},
		{"falso.zip", "isto é só texto", ""}, // o nome promete, os bytes não cumprem
		{"nota.txt", "texto", ""},
	}
	for _, c := range casos {
		if f := formatoDe(m.Grava("f/"+c.nome, c.dados)); f != c.formato {
			t.Errorf("formatoDe(%s) = %q; esperado %q", c.nome, f, c.formato)
		}
	}
	r := m.Pede(`{"op":"compactado-info","caminho":` + prova.J(m.Casa+"/f/nota.txt") + `}`)
	if r["motivo"] != "formato" {
		t.Errorf("info do que não é compactado: %v", r)
	}
	if r := m.Pede(`{"op":"compactado-info","caminho":"relativo"}`); r["erro"] == nil {
		t.Errorf("caminho relativo: %v", r)
	}
}

func TestFormatosEFalta(t *testing.T) {
	m := motor(t)
	r := m.Pede(`{"op":"formatos"}`)
	faltas := map[string]any{}
	for _, lista := range []string{"extrair", "compactar"} {
		for _, f := range r[lista].([]any) {
			c := f.(map[string]any)
			faltas[lista+":"+c["formato"].(string)] = c["falta"]
			if (c["falta"] == nil) != (c["disponivel"] == true) {
				t.Errorf("disponivel e falta não combinam: %v", c)
			}
		}
	}
	esperado := map[string]any{"extrair:zip": nil, "extrair:tar": nil, "extrair:tar.gz": nil, "extrair:tar.bz2": nil, "extrair:gz": nil,
		"extrair:bz2": nil, "extrair:tar.xz": "xz", "extrair:xz": "xz", "extrair:tar.zst": "zstd", "extrair:zst": "zstd",
		"extrair:7z": "7zip", "extrair:rar": "unrar", "compactar:zip": nil, "compactar:tar.gz": nil, "compactar:tar.zst": "zstd",
		"compactar:tar.xz": "xz", "compactar:7z": "7zip"}
	if fmt.Sprint(faltas) != fmt.Sprint(esperado) {
		t.Errorf("formatos sem ferramenta nenhuma:\n %v\n %v", faltas, esperado)
	}
	// Com a ferramenta, o formato fica disponível.
	t.Setenv("MKFILES_XZ", script(t, m, "xz", "exit 0\n"))
	r = m.Pede(`{"op":"formatos"}`)
	for _, f := range r["compactar"].([]any) {
		if c := f.(map[string]any); c["formato"] == "tar.xz" && c["disponivel"] != true {
			t.Errorf("com o xz: %v", c)
		}
	}
	// Sem ela, extrair diz o pacote que falta, e não cria tarefa.
	xz := m.Grava("a.tar.xz", "\xfd7zXZ\x00....")
	t.Setenv("MKFILES_XZ", m.Casa+"/nao-ha")
	if r := pede(t, m, map[string]any{"op": "extrair", "caminho": xz, "destino": m.Casa, "modo": "pasta"}); !strings.Contains(fmt.Sprint(r["erro"]), "pacote xz") {
		t.Errorf("extrair sem o xz: %v", r)
	}
	if i := pede(t, m, map[string]any{"op": "compactado-info", "caminho": xz}); i["disponivel"] != false || i["falta"] != "xz" || i["formato"] != "tar.xz" && i["formato"] != "xz" {
		t.Errorf("info sem o xz: %v", i)
	}
	if l := m.PedeFluxo(prova.J(map[string]any{"op": "compactado-listar", "caminho": xz, "dentro": ""})); l[0]["motivo"] != "falta" || l[0]["pacote"] != "xz" {
		t.Errorf("listar sem o xz: %v", l)
	}
	if r := pede(t, m, map[string]any{"op": "compactar", "origens": []string{xz}, "destino": m.Casa, "nome": "x.7z", "formato": "7z"}); !strings.Contains(fmt.Sprint(r["erro"]), "7zip") {
		t.Errorf("compactar 7z sem o 7z: %v", r)
	}
}

// origem monta a pasta que os testes de ida e volta compactam.
func origem(t *testing.T, m *prova.Motor) (string, map[string]string) {
	t.Helper()
	p := m.Pasta("origem/projeto")
	m.Grava("origem/projeto/leia.txt", "olá, mundo\n")
	m.Grava("origem/projeto/sub/fundo/dados.bin", strings.Repeat("\x00\x01\x02", 5000))
	m.Grava("origem/projeto/roda.sh", "#!/bin/sh\n")
	os.Chmod(p+"/roda.sh", 0o755)
	m.Pasta("origem/projeto/vazia")
	os.Symlink("leia.txt", p+"/atalho")
	// Um link para fora é guardado como link: o alvo nunca é lido.
	m.Grava("segredo.txt", "não entra")
	os.Symlink(m.Casa+"/segredo.txt", p+"/para-fora")
	return p, map[string]string{"projeto": "<pasta>", "projeto/leia.txt": "olá, mundo\n", "projeto/sub": "<pasta>",
		"projeto/sub/fundo": "<pasta>", "projeto/sub/fundo/dados.bin": strings.Repeat("\x00\x01\x02", 5000),
		"projeto/roda.sh": "#!/bin/sh\n", "projeto/vazia": "<pasta>", "projeto/atalho": "-> leia.txt"}
}

func nomesDe(linhas []prova.Linha) string {
	var ns []string
	for _, l := range linhas {
		lista, _ := l["itens"].([]any)
		for _, i := range lista {
			it := i.(map[string]any)
			ns = append(ns, it["nome"].(string)+":"+it["tipo"].(string))
		}
	}
	sort.Strings(ns)
	return strings.Join(ns, " ")
}

func TestIdaEVolta(t *testing.T) {
	for _, formato := range []string{"zip", "tar.gz"} {
		t.Run(formato, func(t *testing.T) {
			m := motor(t)
			projeto, esperado := origem(t, m)
			solto := m.Grava("origem/solto.txt", "sozinho")
			destino := m.Pasta("saida")
			nome := "pacote." + formato
			e := tarefa(t, m, map[string]any{"op": "compactar", "origens": []string{projeto, solto}, "destino": destino,
				"nome": nome, "formato": formato, "nivel": 9, "senha": nil})
			if e["estado"] != "feita" || fmt.Sprint(e["criados"]) != "["+destino+"/"+nome+"]" || e["acao"] != "compactar" {
				t.Fatalf("compactar: %v", e)
			}
			semSobras(t, destino)
			if info, _ := os.Stat(destino + "/" + nome); info == nil || info.Mode().Perm() != 0o644 {
				t.Errorf("o compactado: %v", info)
			}
			pacote := destino + "/" + nome

			i := pede(t, m, map[string]any{"op": "compactado-info", "caminho": pacote, "senha": nil})
			if i["formato"] != formato || i["disponivel"] != true || i["falta"] != nil || i["cifrado"] != false || i["suspeito"] != false {
				t.Errorf("info: %v", i)
			}
			if formato == "zip" && (i["itens"] != float64(10) || i["tamanho"] != float64(12+15000+10+7)) {
				t.Errorf("info do zip (itens e tamanho do índice): %v", i)
			}

			raiz := m.PedeFluxo(prova.J(map[string]any{"op": "compactado-listar", "caminho": pacote, "dentro": "", "senha": nil}))
			if nomesDe(raiz) != "projeto:pasta solto.txt:arquivo" {
				t.Errorf("a raiz: %s", nomesDe(raiz))
			}
			dentro := m.PedeFluxo(prova.J(map[string]any{"op": "compactado-listar", "caminho": pacote, "dentro": "projeto"}))
			if nomesDe(dentro) != "atalho:link leia.txt:arquivo para-fora:link roda.sh:arquivo sub:pasta vazia:pasta" {
				t.Errorf("dentro de projeto: %s", nomesDe(dentro))
			}
			for _, i := range dentro[len(dentro)-1]["itens"].([]any) {
				it := i.(map[string]any)
				if len(it) != 17 {
					t.Errorf("o item de dentro tem os 16 campos do contrato mais «interno»: %v", it)
				}
				switch it["nome"] {
				case "leia.txt":
					if it["interno"] != "projeto/leia.txt" || it["caminho"] != pacote+"/projeto/leia.txt" || it["tamanho"] != float64(12) ||
						it["mime"] != "text/plain" || it["icone"] != "text-plain" || it["permissoes"] != "rw-r--r--" || it["dono"] == "" {
						t.Errorf("leia.txt: %v", it)
					}
				case "roda.sh":
					if it["permissoes"] != "rwxr-xr-x" {
						t.Errorf("roda.sh: %v", it)
					}
				case "atalho":
					if it["alvo"] != "leia.txt" {
						t.Errorf("atalho: %v", it)
					}
				}
			}
			if l := m.PedeFluxo(prova.J(map[string]any{"op": "compactado-listar", "caminho": pacote, "dentro": "../x"})); l[0]["erro"] == nil {
				t.Errorf("dentro com ..: %v", l)
			}

			// Extrair para uma pasta com o nome do arquivo.
			e = tarefa(t, m, map[string]any{"op": "extrair", "caminho": pacote, "destino": destino, "modo": "pasta"})
			// O link para fora fica de fora, com aviso: a tarefa acaba em "erro", com o resto no lugar.
			if e["estado"] != "erro" || !strings.Contains(erros(e), "para-fora") || !strings.Contains(erros(e), "aponta para fora") {
				t.Fatalf("extrair: %v", e)
			}
			if fmt.Sprint(e["criados"]) != "["+destino+"/pacote]" {
				t.Errorf("criados: %v", e["criados"])
			}
			quer := map[string]string{"solto.txt": "sozinho"}
			for k, v := range esperado {
				quer[k] = v
			}
			igual(t, "o que saiu", arvore(t, destino+"/pacote"), quer)
			if info, _ := os.Stat(destino + "/pacote/projeto/roda.sh"); info.Mode().Perm() != 0o755 {
				t.Errorf("o modo do executável: %v", info.Mode())
			}
			if info, _ := os.Stat(destino + "/pacote/projeto/leia.txt"); info.Mode().Perm() != 0o644 {
				t.Errorf("o modo do texto: %v", info.Mode())
			}
			if info, _ := os.Stat(destino + "/pacote"); info.Mode().Perm() != 0o755 {
				t.Errorf("o modo da pasta criada: %v", info.Mode())
			}
			semSobras(t, destino)
			if b, _ := os.ReadFile(m.Casa + "/segredo.txt"); string(b) != "não entra" {
				t.Error("o alvo do link de fora foi mexido")
			}
			// De novo: o nome ocupado ganha (2), sem perguntar.
			e = tarefa(t, m, map[string]any{"op": "extrair", "caminho": pacote, "destino": destino, "modo": "pasta"})
			if fmt.Sprint(e["criados"]) != "["+destino+"/pacote (2)]" {
				t.Errorf("a segunda extração: %v", e["criados"])
			}

			// Só o pedido (copiar de dentro para fora): um arquivo e uma pasta.
			fora := m.Pasta("fora")
			e = tarefa(t, m, map[string]any{"op": "extrair", "caminho": pacote, "destino": fora, "modo": "pasta",
				"internos": []string{"projeto/sub/fundo", "projeto/leia.txt"}})
			if e["estado"] != "feita" {
				t.Errorf("internos: %v", e)
			}
			igual(t, "internos", arvore(t, fora), map[string]string{"fundo": "<pasta>", "fundo/dados.bin": strings.Repeat("\x00\x01\x02", 5000), "leia.txt": "olá, mundo\n"})
			criados := fmt.Sprint(e["criados"])
			if !strings.Contains(criados, fora+"/fundo") || !strings.Contains(criados, fora+"/leia.txt") {
				t.Errorf("criados dos internos: %s", criados)
			}
			if e := tarefa(t, m, map[string]any{"op": "extrair", "caminho": pacote, "destino": fora, "internos": []string{"nao/ha"}}); e["estado"] != "erro" {
				t.Errorf("interno que não existe: %v", e)
			}
			semSobras(t, fora)

			// Compactar com o nome ocupado é erro de formulário, sem tarefa.
			r := pede(t, m, map[string]any{"op": "compactar", "origens": []string{solto}, "destino": destino, "nome": nome, "formato": formato})
			if r["campo"] != "nome" || r["tarefa"] != nil {
				t.Errorf("nome ocupado: %v", r)
			}
		})
	}
}

func TestExtrairAquiComConflito(t *testing.T) {
	m := motor(t)
	z := fazZip(t, m.Casa+"/a.zip", peca{nome: "novo.txt", corpo: "novo"}, peca{nome: "igual.txt", corpo: "do zip"},
		peca{nome: "p/dentro.txt", corpo: "do zip"}, peca{nome: "p/outro.txt", corpo: "outro"})
	destino := m.Pasta("d")
	m.Grava("d/igual.txt", "de antes")
	m.Grava("d/p/dentro.txt", "de antes")
	responde := func(resposta string, nome any) prova.Linha {
		c := m.Evento("conflito", nil)
		if c["origem"] == nil || c["destino"] == nil || c["sugestao"] == nil {
			t.Errorf("o evento de conflito: %v", c)
		}
		m.Pede(prova.J(map[string]any{"op": "responder", "tarefa": c["tarefa"], "conflito": c["conflito"], "resposta": resposta, "nome": nome, "todos": false}))
		return c
	}
	r := pede(t, m, map[string]any{"op": "extrair", "caminho": z, "destino": destino, "modo": "aqui"})
	if c := responde("manter", nil); c["sugestao"] != "igual (2).txt" {
		t.Errorf("sugestão: %v", c["sugestao"])
	}
	responde("substituir", nil) // pasta sobre pasta: mescla
	responde("substituir", nil) // p/dentro.txt
	e := fim(m, r)
	if e["estado"] != "feita" || e["desfazivel"] != false {
		t.Errorf("extrair aqui: %v", e)
	}
	igual(t, "aqui com conflito", arvore(t, destino), map[string]string{"novo.txt": "novo", "igual.txt": "de antes", "igual (2).txt": "do zip",
		"p": "<pasta>", "p/dentro.txt": "do zip", "p/outro.txt": "outro"})
	semSobras(t, destino)

	// Pular deixa como está; renomear grava com o nome dado; cancelar para tudo.
	r = pede(t, m, map[string]any{"op": "extrair", "caminho": z, "destino": destino, "modo": "aqui"})
	responde("pular", nil)                // igual.txt
	responde("renomear", "mais novo.txt") // novo.txt
	responde("cancelar", nil)             // p
	if e := fim(m, r); e["estado"] != "cancelada" {
		t.Errorf("cancelar no conflito: %v", e)
	}
	a := arvore(t, destino)
	if a["mais novo.txt"] != "novo" || a["igual.txt"] != "de antes" || a["p/dentro.txt"] != "do zip" || len(a) != 7 {
		t.Errorf("depois de pular, renomear e cancelar: %v", a)
	}
	semSobras(t, destino)
}

func TestOutrosFormatosDaBiblioteca(t *testing.T) {
	m := motor(t)
	destino := m.Pasta("d")
	solto, _ := hex.DecodeString("425a68393141592653591b6306c000000259800010400010001e0186102000229801ea10340d05b1dc23300e1772453850901b6306c0")
	tarBz, _ := hex.DecodeString("425a683931415926535910de93e40000b9fb84c19008804001ff80008c7e2a9e40001000183000b80c613134182310c8c2618c262683046219184c11513132a621b481e6a4d1e53ca5de353973eb09b32d015862203d67da031310e59464d07d1822117c6b8049d58bdfbc091f561b0869a5ec84860f462845821a1f35633c9bcbb1cce980568356b1a35a79120a211a04074cdf61cf626ddd877d46398d371d86774379bc5b13c3fc40c37781c8f5443fc5dc914e14240437a4f900")
	var gz bytes.Buffer
	w := gzip.NewWriter(&gz)
	io.WriteString(w, "conteúdo gz\n")
	w.Close()
	casos := []struct {
		nome   string
		dados  []byte
		pasta  string
		espera map[string]string
	}{
		{"t.tar", fazTar(t, arq("a.txt", "um\n"), &tar.Header{Name: "sub/", Typeflag: tar.TypeDir, Mode: 0o755}, arq("sub/b.txt", "dois\n")),
			"t", map[string]string{"a.txt": "um\n", "sub": "<pasta>", "sub/b.txt": "dois\n"}},
		{"u.tar.bz2", tarBz, "u", map[string]string{"a.txt": "um\n", "sub": "<pasta>", "sub/b.txt": "dois\n"}},
		{"solto.txt.bz2", solto, "solto.txt", map[string]string{"solto.txt": "conteudo bz2\n"}},
		{"nota.txt.gz", gz.Bytes(), "nota.txt", map[string]string{"nota.txt": "conteúdo gz\n"}},
	}
	for _, c := range casos {
		caminho := m.Grava(c.nome, string(c.dados))
		e := tarefa(t, m, map[string]any{"op": "extrair", "caminho": caminho, "destino": destino, "modo": "pasta"})
		if e["estado"] != "feita" {
			t.Errorf("%s: %v", c.nome, e)
			continue
		}
		igual(t, c.nome, arvore(t, destino+"/"+c.pasta), c.espera)
		l := m.PedeFluxo(prova.J(map[string]any{"op": "compactado-listar", "caminho": caminho, "dentro": ""}))
		if len(l[len(l)-1]["itens"].([]any)) == 0 {
			t.Errorf("%s: a listagem veio vazia", c.nome)
		}
	}
	// Um arquivo cortado no meio: erro, e nada fica no destino.
	cortado := m.Grava("cortado.tar.gz", string(gz.Bytes()[:10])+"lixo lixo lixo")
	antes := len(arvore(t, destino))
	if e := tarefa(t, m, map[string]any{"op": "extrair", "caminho": cortado, "destino": destino, "modo": "pasta"}); e["estado"] != "erro" {
		t.Errorf("arquivo corrompido: %v", e)
	}
	if len(arvore(t, destino)) != antes {
		t.Error("um arquivo corrompido não pode deixar nada no destino")
	}
	semSobras(t, destino)
}

func TestSegurancaDoZip(t *testing.T) {
	m := motor(t)
	fora := m.Pasta("fora")
	m.Grava("fora/intocado.txt", "intocado")
	destino := m.Pasta("zona/d")
	z := fazZip(t, m.Casa+"/mau.zip",
		peca{nome: "../../fora/escapou.txt", corpo: "x"},
		peca{nome: "/" + strings.TrimPrefix(fora, "/") + "/absoluto.txt", corpo: "x"},
		peca{nome: "a\\b.txt", corpo: "x"},
		peca{nome: "nulo\x00.txt", corpo: "x"},
		peca{nome: "sub/../../acima.txt", corpo: "x"},
		// O link aponta para fora, e a entrada seguinte tenta escrever através dele.
		peca{nome: "ponte", corpo: fora, modo: fs.ModeSymlink | 0o777},
		peca{nome: "ponte/atravessou.txt", corpo: "x"},
		peca{nome: "relativo", corpo: "../../fora", modo: fs.ModeSymlink | 0o777},
		peca{nome: "relativo/atravessou2.txt", corpo: "x"},
		peca{nome: "poderoso", corpo: "#!/bin/sh\n", modo: fs.ModeSetuid | fs.ModeSetgid | fs.ModeSticky | 0o755},
		peca{nome: "cano", modo: fs.ModeNamedPipe | 0o644},
		peca{nome: "bom.txt", corpo: "bom"},
	)
	e := tarefa(t, m, map[string]any{"op": "extrair", "caminho": z, "destino": destino, "modo": "aqui"})
	if e["estado"] != "erro" {
		t.Fatalf("o zip malicioso acaba em erro (com os avisos): %v", e)
	}
	avisos := erros(e)
	for _, trecho := range []string{"sai da pasta", "caminho absoluto", "barra invertida", "byte nulo", "aponta para fora", "dispositivo ou um cano"} {
		if !strings.Contains(avisos, trecho) {
			t.Errorf("faltou o aviso «%s» em: %s", trecho, avisos)
		}
	}
	// Fora do destino, nada mudou.
	igual(t, "a pasta de fora", arvore(t, fora), map[string]string{"intocado.txt": "intocado"})
	igual(t, "a pasta acima do destino", arvore(t, m.Casa+"/zona"), arvore(t, m.Casa+"/zona"))
	if es, _ := os.ReadDir(m.Casa + "/zona"); len(es) != 1 {
		t.Errorf("algo escapou para a pasta de cima: %v", es)
	}
	// Dentro: só o que é seguro. Os links para fora não existem; o que tentou
	// passar por eles virou arquivo numa pasta de verdade, dentro do destino.
	a := arvore(t, destino)
	igual(t, "o destino", a, map[string]string{"bom.txt": "bom", "poderoso": "#!/bin/sh\n", "ponte": "<pasta>", "ponte/atravessou.txt": "x",
		"relativo": "<pasta>", "relativo/atravessou2.txt": "x"})
	info, _ := os.Stat(destino + "/poderoso")
	if info.Mode()&(fs.ModeSetuid|fs.ModeSetgid|fs.ModeSticky) != 0 || info.Mode().Perm() != 0o755 {
		t.Errorf("setuid, setgid e sticky têm de sair: %v", info.Mode())
	}
	semSobras(t, destino)
}

func TestSegurancaDoTar(t *testing.T) {
	m := motor(t)
	fora := m.Pasta("fora")
	m.Grava("fora/intocado.txt", "intocado")
	destino := m.Pasta("zona/d")
	dados := fazTar(t,
		arq("../../fora/escapou.txt", "x"),
		arq("/tmp/mkfiles-nunca-deve-existir.txt", "x"),
		&tar.Header{Name: "ponte", Typeflag: tar.TypeSymlink, Linkname: fora},
		arq("ponte/atravessou.txt", "x"),
		&tar.Header{Name: "sobe", Typeflag: tar.TypeSymlink, Linkname: "../.."},
		arq("sobe/fora/atravessou2.txt", "x"),
		&tar.Header{Name: "duro-fora", Typeflag: tar.TypeLink, Linkname: "../../fora/intocado.txt"},
		&tar.Header{Name: "duro-absoluto", Typeflag: tar.TypeLink, Linkname: fora + "/intocado.txt"},
		arq("bom.txt", "bom"),
		&tar.Header{Name: "duro-dentro", Typeflag: tar.TypeLink, Linkname: "bom.txt"},
		&tar.Header{Name: "dentro", Typeflag: tar.TypeSymlink, Linkname: "bom.txt"},
		&tar.Header{Name: "cano", Typeflag: tar.TypeFifo},
		&tar.Header{Name: "disco", Typeflag: tar.TypeBlock, Devmajor: 8},
		&tar.Header{Name: "tty", Typeflag: tar.TypeChar, Devmajor: 5},
		&tar.Header{Name: "poderoso", Typeflag: tar.TypeReg, Linkname: "s", Mode: 0o6755},
		&tar.Header{Name: "pegajosa/", Typeflag: tar.TypeDir, Mode: 0o1777},
		arq("a\\b", "x"),
	)
	e := tarefa(t, m, map[string]any{"op": "extrair", "caminho": m.Grava("mau.tar", string(dados)), "destino": destino, "modo": "aqui"})
	avisos := erros(e)
	for _, trecho := range []string{"sai da pasta", "caminho absoluto", "aponta para fora", "hardlink para fora", "dispositivo ou um cano", "barra invertida"} {
		if !strings.Contains(avisos, trecho) {
			t.Errorf("faltou o aviso «%s» em: %s", trecho, avisos)
		}
	}
	if strings.Count(avisos, "dispositivo ou um cano") != 3 {
		t.Errorf("fifo, bloco e caractere: os três ficam de fora: %s", avisos)
	}
	igual(t, "a pasta de fora", arvore(t, fora), map[string]string{"intocado.txt": "intocado"})
	if es, _ := os.ReadDir(m.Casa + "/zona"); len(es) != 1 {
		t.Errorf("algo escapou para a pasta de cima: %v", es)
	}
	if _, err := os.Lstat("/tmp/mkfiles-nunca-deve-existir.txt"); err == nil {
		t.Fatal("um caminho absoluto foi escrito")
	}
	igual(t, "o destino", arvore(t, destino), map[string]string{"bom.txt": "bom", "duro-dentro": "bom", "dentro": "-> bom.txt", "poderoso": "s",
		"pegajosa": "<pasta>", "ponte": "<pasta>", "ponte/atravessou.txt": "x", "sobe": "<pasta>", "sobe/fora": "<pasta>", "sobe/fora/atravessou2.txt": "x"})
	for _, nome := range []string{"poderoso", "pegajosa"} {
		if info, _ := os.Stat(destino + "/" + nome); info.Mode()&(fs.ModeSetuid|fs.ModeSetgid|fs.ModeSticky) != 0 {
			t.Errorf("%s: setuid, setgid e sticky têm de sair: %v", nome, info.Mode())
		}
	}
	semSobras(t, destino)
}

func TestBomba(t *testing.T) {
	m := motor(t)
	antigo := pisoDaTaxa
	pisoDaTaxa = 256 << 10
	t.Cleanup(func() { pisoDaTaxa = antigo })
	destino := m.Pasta("d")
	z := fazZip(t, m.Casa+"/bomba.zip", peca{nome: "zeros.bin", corpo: strings.Repeat("\x00", 4<<20)})

	i := pede(t, m, map[string]any{"op": "compactado-info", "caminho": z})
	if i["suspeito"] != true || i["taxa"].(float64) < 200 || i["tamanho"] != float64(4<<20) {
		t.Errorf("a info avisa antes: %v", i)
	}
	e := tarefa(t, m, map[string]any{"op": "extrair", "caminho": z, "destino": destino, "modo": "pasta"})
	if e["estado"] != "erro" || !strings.HasPrefix(erros(e), "parece uma bomba de compressão") {
		t.Errorf("a bomba para a tarefa: %v", e)
	}
	if len(arvore(t, destino)) != 0 {
		t.Errorf("a bomba não deixa nada: %v", arvore(t, destino))
	}
	// A janela perguntou e a pessoa quis: segue.
	e = tarefa(t, m, map[string]any{"op": "extrair", "caminho": z, "destino": destino, "modo": "pasta", "sem_limite": true})
	if info, _ := os.Stat(destino + "/bomba/zeros.bin"); e["estado"] != "feita" || info == nil || info.Size() != 4<<20 {
		t.Errorf("com sem_limite: %v", e)
	}
	// O teto de tamanho total vale mesmo com taxa baixa, e conta o que foi
	// escrito de verdade (um tar não diz o total antes).
	t.Setenv("MKFILES_EXTRAIR_TAXA", "1000000")
	t.Setenv("MKFILES_EXTRAIR_MAX", "100000")
	var gz bytes.Buffer
	w := gzip.NewWriter(&gz)
	w.Write(fazTar(t, arq("a.bin", strings.Repeat("ab", 40000)), arq("b.bin", strings.Repeat("cd", 40000))))
	w.Close()
	e = tarefa(t, m, map[string]any{"op": "extrair", "caminho": m.Grava("grande.tar.gz", gz.String()), "destino": destino, "modo": "aqui"})
	if e["estado"] != "erro" || !strings.Contains(erros(e), "bomba") {
		t.Errorf("o teto total: %v", e)
	}
	if a := arvore(t, destino); a["a.bin"] != "" || a["b.bin"] != "" {
		t.Error("o que já tinha saído antes do teto também não fica")
	}
	semSobras(t, destino)
}

func TestCancelarLimpaOTemporario(t *testing.T) {
	m := motor(t)
	antigo := tamanhoDoBloco
	tamanhoDoBloco = 1024
	t.Cleanup(func() { tamanhoDoBloco, aCadaBloco = antigo, nil })
	destino := m.Pasta("d")
	grande := m.Grava("origem/grande.bin", strings.Repeat("0123456789abcdef", 64<<10)) // 1 MiB
	z := fazZip(t, m.Casa+"/a.zip", peca{nome: "grande.bin", corpo: strings.Repeat("xyz", 300000)})

	for _, pedido := range []map[string]any{
		{"op": "extrair", "caminho": z, "destino": destino, "modo": "pasta"},
		{"op": "extrair", "caminho": z, "destino": destino, "modo": "aqui"},
		{"op": "compactar", "origens": []string{grande}, "destino": destino, "nome": "x.zip", "formato": "zip"},
		{"op": "compactar", "origens": []string{grande}, "destino": destino, "nome": "x.tar.gz", "formato": "tar.gz"},
	} {
		// No terceiro bloco de dados, com o arquivo pela metade, chega o cancelar.
		blocos, id := 0, make(chan any, 1)
		aCadaBloco = func() {
			if blocos++; blocos == 3 {
				m.S.Atender([]byte(prova.J(map[string]any{"id": 9000, "op": "tarefa-cancelar", "tarefa": <-id})))
			}
		}
		r := pede(t, m, pedido)
		id <- r["tarefa"]
		e := fim(m, r)
		if e["estado"] != "cancelada" || blocos > 4 {
			t.Errorf("%v: estado %v depois de %d blocos", pedido["op"], e["estado"], blocos)
		}
		if a := arvore(t, destino); len(a) != 0 {
			t.Errorf("%v %v cancelado deixou: %v", pedido["op"], pedido["modo"], a)
		}
	}
}

func TestAbrirDeDentro(t *testing.T) {
	m := motor(t)
	z := fazZip(t, m.Casa+"/a.zip", peca{nome: "docs/leia.txt", corpo: "de dentro"}, peca{nome: "docs/outro.txt", corpo: "outro"},
		peca{nome: "docs/", modo: fs.ModeDir | 0o755})
	r := pede(t, m, map[string]any{"op": "compactado-abrir", "caminho": z, "interno": "docs/leia.txt", "senha": nil})
	caminho, _ := r["caminho"].(string)
	corrida := m.Casa + "/.run/mkfiles/"
	if !strings.HasPrefix(caminho, corrida+"sessao-") || filepath.Base(caminho) != "leia.txt" {
		t.Fatalf("abrir: %v", r)
	}
	if b, _ := os.ReadFile(caminho); string(b) != "de dentro" {
		t.Errorf("o conteúdo: %q", b)
	}
	// Só aquele arquivo saiu, numa pasta só da pessoa.
	if es, _ := os.ReadDir(filepath.Dir(caminho)); len(es) != 1 {
		t.Errorf("saiu mais do que o pedido: %v", es)
	}
	for _, p := range []string{corrida, filepath.Dir(filepath.Dir(caminho)), filepath.Dir(caminho)} {
		if info, _ := os.Stat(p); info.Mode().Perm() != 0o700 {
			t.Errorf("%s devia ser 0700: %v", p, info.Mode())
		}
	}
	// De um tar.gz também, e o mesmo nome duas vezes não se atropela.
	var gz bytes.Buffer
	w := gzip.NewWriter(&gz)
	w.Write(fazTar(t, arq("x/leia.txt", "do tar")))
	w.Close()
	r2 := pede(t, m, map[string]any{"op": "compactado-abrir", "caminho": m.Grava("b.tgz", gz.String()), "interno": "x/leia.txt"})
	if b, _ := os.ReadFile(r2["caminho"].(string)); string(b) != "do tar" || r2["caminho"] == caminho {
		t.Errorf("abrir do tar.gz: %v", r2)
	}
	for _, ruim := range []string{"docs", "nao/ha.txt", "../a.zip", "/etc/passwd"} {
		if r := pede(t, m, map[string]any{"op": "compactado-abrir", "caminho": z, "interno": ruim}); r["erro"] == nil {
			t.Errorf("abrir %q devia falhar: %v", ruim, r)
		}
	}
	// Grande demais para abrir de dentro.
	antigo := limiteDeAbrir
	limiteDeAbrir = 4
	if r := pede(t, m, map[string]any{"op": "compactado-abrir", "caminho": z, "interno": "docs/leia.txt"}); !strings.Contains(fmt.Sprint(r["erro"]), "grande demais") {
		t.Errorf("o limite de abrir: %v", r)
	}
	limiteDeAbrir = antigo
	// Fechar a sessão apaga os temporários dela, e só eles.
	vizinho := m.Grava(".run/mkfiles/de-outro/fica.txt", "fica")
	m.S.Fecha()
	if _, err := os.Stat(caminho); err == nil {
		t.Error("o temporário devia sumir ao fechar")
	}
	if es, _ := os.ReadDir(corrida); len(es) != 1 {
		t.Errorf("a pasta da sessão devia sumir, e só ela: %v", es)
	}
	if b, _ := os.ReadFile(vizinho); string(b) != "fica" {
		t.Error("o que não é da sessão não pode ser apagado")
	}
}

func TestLimpaSoOQueEDoMotor(t *testing.T) {
	pasta := t.TempDir()
	meu := filepath.Join(pasta, ".mkfiles-extrai-1-1")
	alheio := filepath.Join(pasta, "documentos")
	fora := filepath.Join(pasta, "fora")
	for _, p := range []string{meu + "/sub", alheio, fora} {
		os.MkdirAll(p, 0o755)
	}
	os.WriteFile(fora+"/fica.txt", []byte("fica"), 0o644)
	os.Symlink(fora, meu+"/sub/atalho")
	limpa(alheio, pasta, marcaDeExtrair)       // não tem a marca
	limpa(meu, "/outra/pasta", marcaDeExtrair) // não está onde devia
	limpa("", pasta, marcaDeExtrair)
	limpa(pasta, filepath.Dir(pasta), "")
	for _, p := range []string{meu, alheio, pasta} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("%s não podia ter sido apagado", p)
		}
	}
	limpa(meu, pasta, marcaDeExtrair)
	if _, err := os.Lstat(meu); err == nil {
		t.Error("o temporário do motor devia sumir")
	}
	if _, err := os.Stat(fora + "/fica.txt"); err != nil {
		t.Error("apagar o temporário seguiu um link para fora")
	}
}

func TestValidacoes(t *testing.T) {
	m := motor(t)
	z := fazZip(t, m.Casa+"/a.zip", peca{nome: "a.txt", corpo: "a"})
	d := m.Pasta("d")
	ruins := []map[string]any{
		{"op": "extrair", "caminho": "a.zip", "destino": d},
		{"op": "extrair", "caminho": z, "destino": "d"},
		{"op": "extrair", "caminho": z, "destino": d + "/../d"},
		{"op": "extrair", "caminho": z, "destino": d + "/nao-ha"},
		{"op": "extrair", "caminho": z, "destino": d, "modo": "outro"},
		{"op": "extrair", "caminho": z, "destino": d, "internos": []string{"../x"}},
		{"op": "extrair", "caminho": z, "destino": d, "senha": "duas\nlinhas"},
		{"op": "extrair", "caminho": m.Casa + "/nao-ha.zip", "destino": d},
		{"op": "extrair", "caminho": m.Grava("nota.txt", "texto"), "destino": d},
		{"op": "compactar", "origens": []string{}, "destino": d, "nome": "x.zip", "formato": "zip"},
		{"op": "compactar", "origens": []string{"relativo"}, "destino": d, "nome": "x.zip", "formato": "zip"},
		{"op": "compactar", "origens": []string{z}, "destino": d, "nome": "a/b.zip", "formato": "zip"},
		{"op": "compactar", "origens": []string{z}, "destino": d, "nome": "..", "formato": "zip"},
		{"op": "compactar", "origens": []string{z}, "destino": d, "nome": "x.lzh", "formato": "lzh"},
		{"op": "compactar", "origens": []string{z}, "destino": d, "nome": "x.zip", "formato": "zip", "nivel": 10},
		{"op": "compactar", "origens": []string{z}, "destino": d, "nome": "x.tar.gz", "formato": "tar.gz", "senha": "s"},
		{"op": "compactar", "origens": []string{z}, "destino": d, "nome": "x.zip", "formato": "zip", "senha": "s"}, // sem o 7z
		{"op": "compactar", "origens": []string{m.Casa + "/nao-ha"}, "destino": d, "nome": "x.zip", "formato": "zip"},
		{"op": "compactar", "origens": []string{"/"}, "destino": d, "nome": "x.zip", "formato": "zip"},
		{"op": "compactar", "origens": []string{z, m.Grava("outra/a.zip", "x")}, "destino": d, "nome": "x.zip", "formato": "zip"},
	}
	for _, pedido := range ruins {
		if r := pede(t, m, pedido); r["erro"] == nil || r["tarefa"] != nil {
			t.Errorf("devia ser recusado: %v -> %v", pedido, r)
		}
	}
	if len(arvore(t, d)) != 0 {
		t.Errorf("um pedido recusado não toca em nada: %v", arvore(t, d))
	}
	// Compactar uma pasta para dentro dela mesma não engole o próprio arquivo.
	p := m.Pasta("eu")
	m.Grava("eu/a.txt", "a")
	if e := tarefa(t, m, map[string]any{"op": "compactar", "origens": []string{p}, "destino": p, "nome": "eu.zip", "formato": "zip"}); e["estado"] != "feita" {
		t.Fatalf("compactar dentro de si: %v", e)
	}
	l := m.PedeFluxo(prova.J(map[string]any{"op": "compactado-listar", "caminho": p + "/eu.zip", "dentro": "eu"}))
	if nomesDe(l) != "a.txt:arquivo" {
		t.Errorf("o zip dentro da pasta compactada: %s", nomesDe(l))
	}
}

// --- ferramentas de fora, de mentira ---

// A casca de mentira: os bytes mágicos de verdade na frente e o conteúdo sem
// comprimir atrás. "-dc" tira a frente; "-c" põe.
func cascaDeMentira(t *testing.T, m *prova.Motor, nome, magicoOctal string, pular int) string {
	return script(t, m, nome, fmt.Sprintf(`echo "$@" >>%q
case "$1" in
  -dc) tail -c +%d ;;
  -c) printf '%s'; cat ;;
  *) exit 2 ;;
esac
`, m.Casa+"/."+nome+"-args", pular+1, magicoOctal))
}

func TestXzEZstdPorComando(t *testing.T) {
	m := motor(t)
	t.Setenv("MKFILES_XZ", cascaDeMentira(t, m, "xz", `\3757zXZ\000`, 6))
	t.Setenv("MKFILES_ZSTD", cascaDeMentira(t, m, "zstd", `\050\265\057\375`, 4))
	projeto, esperado := origem(t, m)
	destino := m.Pasta("saida")
	for _, formato := range []string{"tar.xz", "tar.zst"} {
		nome := "p." + formato
		e := tarefa(t, m, map[string]any{"op": "compactar", "origens": []string{projeto}, "destino": destino, "nome": nome, "formato": formato, "nivel": 3})
		if e["estado"] != "feita" {
			t.Fatalf("compactar %s: %v", formato, e)
		}
		if f := formatoDe(destino + "/" + nome); f != formato {
			t.Fatalf("o formato do que saiu: %q", f)
		}
		l := m.PedeFluxo(prova.J(map[string]any{"op": "compactado-listar", "caminho": destino + "/" + nome, "dentro": "projeto/sub"}))
		if nomesDe(l) != "fundo:pasta" {
			t.Errorf("listar %s: %s (%v)", formato, nomesDe(l), l)
		}
		e = tarefa(t, m, map[string]any{"op": "extrair", "caminho": destino + "/" + nome, "destino": destino, "modo": "pasta"})
		igual(t, formato, arvore(t, destino+"/p"), esperado)
		apagaArvore(destino + "/p")
		if !strings.Contains(erros(e), "aponta para fora") {
			t.Errorf("o tar de dentro passa pelas nossas travas: %v", e)
		}
	}
	args, _ := os.ReadFile(m.Casa + "/.xz-args")
	if !strings.Contains(string(args), "-c -3") || !strings.Contains(string(args), "-dc") {
		t.Errorf("o xz foi chamado com: %q", args)
	}
	// Um xz sozinho vira um arquivo com o nome sem a extensão.
	solto := m.Grava("nota.txt.xz", "\xfd7zXZ\x00o texto")
	e := tarefa(t, m, map[string]any{"op": "extrair", "caminho": solto, "destino": destino, "modo": "aqui"})
	if b, _ := os.ReadFile(destino + "/nota.txt"); string(b) != "o texto" {
		t.Errorf("xz sozinho: %v %q", e, b)
	}
	// O comando que falha vira erro, sem nada no destino.
	t.Setenv("MKFILES_XZ", script(t, m, "xz-ruim", "echo 'xz: arquivo corrompido' >&2\nexit 1\n"))
	e = tarefa(t, m, map[string]any{"op": "extrair", "caminho": destino + "/p.tar.xz", "destino": destino, "modo": "pasta"})
	if e["estado"] != "erro" {
		t.Errorf("xz que falha: %v", e)
	}
	if _, err := os.Stat(destino + "/p"); err == nil {
		t.Error("o xz falhou e mesmo assim ficou algo")
	}
	semSobras(t, destino)
}

// Um 7z de mentira: anota os argumentos e o que chegou pelo stdin; lista um
// índice gravado; ao extrair, cria um arquivo bom, um setuid, um link para
// fora e um cano; ao compactar, grava o arquivo pedido. Senha certa: "certa".
func seteZip(t *testing.T, m *prova.Motor, fora string) string {
	return script(t, m, "7z", fmt.Sprintf(`log=%q
echo "ARGS: $*" >>"$log"
senha=$(head -n 1)
echo "STDIN: $senha" >>"$log"
if grep -q cifrado "$log.modo" 2>/dev/null && [ "$senha" != certa ]; then
  if [ -z "$senha" ]; then echo "Enter password (will not be echoed):"; else echo "ERROR: Wrong password : x"; fi
  exit 2
fi
case "$1" in
  l)
    printf 'Path = docs\nFolder = +\nSize = 0\nModified = 2024-05-06 07:08:09\nAttributes = D\nEncrypted = -\n\n'
    printf 'Path = docs/leia.txt\nFolder = -\nSize = 9\nModified = 2024-05-06 07:08:09\nAttributes = A\nEncrypted = %%s\n\n' "$(cat "$log.cifra" 2>/dev/null || echo -)"
    printf 'Path = poderoso\nFolder = -\nSize = 2\nModified = 2024-05-06 07:08:09\nAttributes = A\nEncrypted = -\n'
    ;;
  x)
    for a in "$@"; do case "$a" in -o*) d=${a#-o} ;; esac; done
    mkdir -p "$d/docs"
    printf 'de dentro' >"$d/docs/leia.txt"
    printf 'su' >"$d/poderoso"; chmod 6755 "$d/poderoso"
    ln -s %q "$d/ponte"
    ln -s docs/leia.txt "$d/dentro"
    mkfifo "$d/cano"
    ;;
  a)
    while [ "$1" != "--" ]; do shift; done
    printf '7z\274\257\047\034feito' >"$2"
    ;;
esac
`, m.Casa+"/.7z-log", fora))
}

func TestSeteZipPorComando(t *testing.T) {
	m := motor(t)
	fora := m.Pasta("fora")
	m.Grava("fora/intocado.txt", "intocado")
	t.Setenv("MKFILES_7Z", seteZip(t, m, fora))
	log := m.Casa + "/.7z-log"
	arquivo := m.Grava("a.7z", "7z\xbc\xaf\x27\x1c........")
	destino := m.Pasta("d")

	i := pede(t, m, map[string]any{"op": "compactado-info", "caminho": arquivo})
	if i["formato"] != "7z" || i["disponivel"] != true || i["itens"] != float64(3) || i["tamanho"] != float64(11) || i["cifrado"] != false {
		t.Errorf("info do 7z: %v", i)
	}
	l := m.PedeFluxo(prova.J(map[string]any{"op": "compactado-listar", "caminho": arquivo, "dentro": "docs"}))
	if nomesDe(l) != "leia.txt:arquivo" || l[0]["itens"].([]any)[0].(map[string]any)["mtime"] == float64(0) {
		t.Errorf("listar o 7z: %v", l)
	}
	e := tarefa(t, m, map[string]any{"op": "extrair", "caminho": arquivo, "destino": destino, "modo": "pasta"})
	// O que a ferramenta extraiu passa pela varredura antes de ir para o lugar.
	igual(t, "o 7z extraído", arvore(t, destino+"/a"), map[string]string{"docs": "<pasta>", "docs/leia.txt": "de dentro", "poderoso": "su", "dentro": "-> docs/leia.txt"})
	if !strings.Contains(erros(e), "aponta para fora") || !strings.Contains(erros(e), "dispositivo ou um cano") {
		t.Errorf("os avisos da varredura: %v", e)
	}
	if info, _ := os.Stat(destino + "/a/poderoso"); info.Mode()&(fs.ModeSetuid|fs.ModeSetgid) != 0 {
		t.Errorf("setuid do 7z tem de sair: %v", info.Mode())
	}
	igual(t, "a pasta de fora", arvore(t, fora), map[string]string{"intocado.txt": "intocado"})
	// Só o pedido, e abrir um arquivo de dentro.
	so := m.Pasta("so")
	e = tarefa(t, m, map[string]any{"op": "extrair", "caminho": arquivo, "destino": so, "internos": []string{"docs/leia.txt"}})
	igual(t, "internos do 7z", arvore(t, so), map[string]string{"leia.txt": "de dentro"})
	r := pede(t, m, map[string]any{"op": "compactado-abrir", "caminho": arquivo, "interno": "docs/leia.txt"})
	if b, _ := os.ReadFile(fmt.Sprint(r["caminho"])); string(b) != "de dentro" {
		t.Errorf("abrir do 7z: %v", r)
	}
	semSobras(t, destino)
	semSobras(t, so)

	// Com senha: a senha chega pelo stdin e nunca aparece nos argumentos.
	os.WriteFile(log+".modo", []byte("cifrado"), 0o644)
	os.WriteFile(log+".cifra", []byte("+"), 0o644)
	cifrado := m.Grava("c.7z", "7z\xbc\xaf\x27\x1c.cifrado.")
	if i := pede(t, m, map[string]any{"op": "compactado-info", "caminho": cifrado}); i["cifrado"] != true {
		t.Errorf("info do cifrado sem senha: %v", i)
	}
	if l := m.PedeFluxo(prova.J(map[string]any{"op": "compactado-listar", "caminho": cifrado, "dentro": ""})); l[0]["motivo"] != "senha" {
		t.Errorf("listar sem senha: %v", l)
	}
	if l := m.PedeFluxo(prova.J(map[string]any{"op": "compactado-listar", "caminho": cifrado, "dentro": "", "senha": "errada"})); l[0]["motivo"] != "senha" || !strings.Contains(fmt.Sprint(l[0]["erro"]), "não confere") {
		t.Errorf("listar com a senha errada: %v", l)
	}
	e = tarefa(t, m, map[string]any{"op": "extrair", "caminho": cifrado, "destino": destino, "modo": "pasta", "senha": "errada"})
	if e["estado"] != "erro" || !strings.Contains(erros(e), "senha") || strings.Contains(erros(e), "errada") {
		t.Errorf("extrair com a senha errada (e a senha não aparece no erro): %v", e)
	}
	if _, err := os.Stat(destino + "/c"); err == nil {
		t.Error("senha errada e mesmo assim extraiu")
	}
	e = tarefa(t, m, map[string]any{"op": "extrair", "caminho": cifrado, "destino": destino, "modo": "pasta", "senha": "certa"})
	if b, _ := os.ReadFile(destino + "/c/docs/leia.txt"); string(b) != "de dentro" {
		t.Errorf("extrair com a senha certa: %v", e)
	}
	// Compactar em 7z com senha, e zip com senha (que também vai pelo 7z).
	origem := m.Grava("o/a.txt", "a")
	for _, f := range []string{"7z", "zip"} {
		e = tarefa(t, m, map[string]any{"op": "compactar", "origens": []string{origem}, "destino": destino, "nome": "novo." + f, "formato": f, "nivel": 5, "senha": "certa"})
		if _, err := os.Stat(destino + "/novo." + f); e["estado"] != "feita" || err != nil {
			t.Errorf("compactar %s com senha: %v", f, e)
		}
	}
	tudo, _ := os.ReadFile(log)
	args := ""
	for _, linha := range strings.Split(string(tudo), "\n") {
		if strings.HasPrefix(linha, "ARGS: ") {
			args += linha + "\n"
		}
	}
	if strings.Contains(args, "certa") || strings.Contains(args, "errada") {
		t.Errorf("a senha foi parar na linha de comando:\n%s", args)
	}
	if !strings.Contains(string(tudo), "STDIN: certa") || !strings.Contains(string(tudo), "STDIN: errada") {
		t.Errorf("a senha tem de chegar pelo stdin:\n%s", tudo)
	}
	for _, trecho := range []string{"a -t7z -y -bd -snl -sccUTF-8 -mx=5 -p -mhe=on -- ", "a -tzip -y -bd -snl -sccUTF-8 -mx=5 -p -- ", "x -y -bd -sccUTF-8 -o", "l -slt -ba -sccUTF-8 -- "} {
		if !strings.Contains(args, trecho) {
			t.Errorf("faltou a chamada «%s» em:\n%s", trecho, args)
		}
	}
	semSobras(t, destino)
}

func TestZipCifrado(t *testing.T) {
	m := motor(t)
	destino := m.Pasta("d")
	z := fazZip(t, m.Casa+"/c.zip", peca{nome: "docs/leia.txt", corpo: "xxxxxxxxx", cifrada: true})
	// A lista sai do índice, sem senha; sem o 7z, extrair e abrir dizem o que falta.
	if l := m.PedeFluxo(prova.J(map[string]any{"op": "compactado-listar", "caminho": z, "dentro": "docs"})); nomesDe(l) != "leia.txt:arquivo" || l[0]["cifrado"] != true {
		t.Errorf("listar o zip cifrado: %v", l)
	}
	if i := pede(t, m, map[string]any{"op": "compactado-info", "caminho": z}); i["cifrado"] != true || i["disponivel"] != false || i["falta"] != "7zip" {
		t.Errorf("info do zip cifrado sem o 7z: %v", i)
	}
	e := tarefa(t, m, map[string]any{"op": "extrair", "caminho": z, "destino": destino, "modo": "pasta", "senha": "certa"})
	if e["estado"] != "erro" || !strings.Contains(erros(e), "precisa do 7zip") || len(arvore(t, destino)) != 0 {
		t.Errorf("zip cifrado sem o 7z: %v", e)
	}
	if r := pede(t, m, map[string]any{"op": "compactado-abrir", "caminho": z, "interno": "docs/leia.txt", "senha": "certa"}); r["motivo"] != "falta" || r["pacote"] != "7zip" {
		t.Errorf("abrir de zip cifrado sem o 7z: %v", r)
	}
	// Com o 7z, vai por ele, com a senha no stdin.
	t.Setenv("MKFILES_7Z", seteZip(t, m, m.Pasta("fora")))
	e = tarefa(t, m, map[string]any{"op": "extrair", "caminho": z, "destino": destino, "modo": "pasta", "senha": "certa"})
	if b, _ := os.ReadFile(destino + "/c/docs/leia.txt"); string(b) != "de dentro" {
		t.Errorf("zip cifrado pelo 7z: %v", e)
	}
	if i := pede(t, m, map[string]any{"op": "compactado-info", "caminho": z}); i["disponivel"] != true {
		t.Errorf("info do zip cifrado com o 7z: %v", i)
	}
	if r := pede(t, m, map[string]any{"op": "compactado-abrir", "caminho": z, "interno": "docs/leia.txt"}); r["motivo"] != "senha" {
		t.Errorf("abrir de zip cifrado sem senha: %v", r)
	}
}

func TestRarSemO7z(t *testing.T) {
	m := motor(t)
	t.Setenv("MKFILES_UNRAR", script(t, m, "unrar", fmt.Sprintf(`echo "$*" >>%q
case "$1" in
  lb) printf 'docs\ndocs/leia.txt\nsolto.txt\n' ;;
  x) for d in "$@"; do :; done; mkdir -p "$d/docs"; printf 'do rar' >"$d/docs/leia.txt"; printf 's' >"$d/solto.txt" ;;
esac
`, m.Casa+"/.unrar-args")))
	arquivo := m.Grava("a.rar", "Rar!\x1a\x07\x00.....")
	l := m.PedeFluxo(prova.J(map[string]any{"op": "compactado-listar", "caminho": arquivo, "dentro": ""}))
	if nomesDe(l) != "docs:pasta solto.txt:arquivo" {
		t.Errorf("listar o rar: %s", nomesDe(l))
	}
	destino := m.Pasta("d")
	e := tarefa(t, m, map[string]any{"op": "extrair", "caminho": arquivo, "destino": destino, "modo": "pasta"})
	if e["estado"] != "feita" {
		t.Errorf("extrair o rar: %v", e)
	}
	igual(t, "o rar", arvore(t, destino+"/a"), map[string]string{"docs": "<pasta>", "docs/leia.txt": "do rar", "solto.txt": "s"})
	args, _ := os.ReadFile(m.Casa + "/.unrar-args")
	if !strings.Contains(string(args), "x -o+ -y -p- -- "+arquivo) {
		t.Errorf("o unrar foi chamado com: %s", args)
	}
}

func TestDesfazerExtrairECompactar(t *testing.T) {
	m := motor(t)
	pasta := m.Pasta("d")
	z := fazZip(t, pasta+"/pacote.zip", peca{nome: "a.txt", corpo: "um"}, peca{nome: "sub/b.txt", corpo: "dois"})
	e := tarefa(t, m, map[string]any{"op": "extrair", "caminho": z, "destino": pasta, "modo": "pasta"})
	if e["estado"] != "feita" || e["desfazivel"] != true {
		t.Fatalf("extrair tem de ser desfazível: %v", e)
	}
	if _, err := os.Stat(pasta + "/pacote/sub/b.txt"); err != nil {
		t.Fatal(err)
	}
	// Um arquivo mexido depois de extraído não é apagado pelo desfazer.
	os.WriteFile(pasta+"/pacote/a.txt", []byte("mudei aqui"), 0o644)
	fim(m, m.Pede(`{"op":"desfazer"}`))
	if _, err := os.Stat(pasta + "/pacote/sub"); !os.IsNotExist(err) {
		t.Errorf("o desfazer tira o que foi extraído e não mudou: %v", err)
	}
	if b, _ := os.ReadFile(pasta + "/pacote/a.txt"); string(b) != "mudei aqui" {
		t.Errorf("o arquivo mudado fica: %q", b)
	}
	if _, err := os.Stat(z); err != nil {
		t.Error("o compactado de origem não é tocado")
	}

	fonte := m.Grava("d/nota.txt", "texto")
	e = tarefa(t, m, map[string]any{"op": "compactar", "origens": []string{fonte}, "destino": pasta, "nome": "novo.zip", "formato": "zip"})
	if e["estado"] != "feita" || e["desfazivel"] != true {
		t.Fatalf("compactar tem de ser desfazível: %v", e)
	}
	fim(m, m.Pede(`{"op":"desfazer"}`))
	if _, err := os.Stat(pasta + "/novo.zip"); !os.IsNotExist(err) {
		t.Errorf("o desfazer tira o compactado criado: %v", err)
	}
	if _, err := os.Stat(fonte); err != nil {
		t.Error("a origem fica")
	}
}
