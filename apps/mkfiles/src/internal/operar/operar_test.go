package operar

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"mkfiles/internal/lixeira"
	"mkfiles/internal/prova"
)

func motor(t *testing.T) *prova.Motor {
	t.Helper()
	t.Cleanup(func() {
		renomeia, tamanhoDoBloco, aCadaBloco, aoConferir, intervalo = os.Rename, 1<<20, nil, nil, 100*time.Millisecond
		unidadeDe = unidadeDoDisco
	})
	return prova.Novo(t, lixeira.Ligar, Ligar)
}

func final(l prova.Linha) bool {
	e := l["estado"]
	return e == "feita" || e == "cancelada" || e == "erro"
}

// fim espera o evento final de uma tarefa.
func fim(m *prova.Motor, tarefa any) prova.Linha {
	return m.Evento("tarefa", func(l prova.Linha) bool { return l["tarefa"] == tarefa && final(l) })
}

// pede manda um pedido de tarefa e devolve o número dela.
func pede(t *testing.T, m *prova.Motor, pedido string) any {
	t.Helper()
	r := m.Pede(pedido)
	if r["tarefa"] == nil {
		t.Fatalf("%s não virou tarefa: %v", pedido, r)
	}
	return r["tarefa"]
}

// faz manda um pedido de tarefa e espera ela acabar.
func faz(t *testing.T, m *prova.Motor, pedido string) prova.Linha {
	t.Helper()
	return fim(m, pede(t, m, pedido))
}

func feita(t *testing.T, m *prova.Motor, pedido string) prova.Linha {
	t.Helper()
	f := faz(t, m, pedido)
	if f["estado"] != "feita" {
		t.Fatalf("%s: esperava feita, veio %v", pedido, f)
	}
	return f
}

func op(nome string, campos ...any) string {
	p := map[string]any{"op": nome}
	for i := 0; i+1 < len(campos); i += 2 {
		p[campos[i].(string)] = campos[i+1]
	}
	return prova.J(p)
}

func le(c string) string {
	b, err := os.ReadFile(c)
	if err != nil {
		return "<" + err.Error() + ">"
	}
	return string(b)
}

func responde(m *prova.Motor, c prova.Linha, resposta string, mais ...any) prova.Linha {
	campos := append([]any{"tarefa", c["tarefa"], "conflito", c["conflito"], "resposta", resposta}, mais...)
	return m.Pede(op("responder", campos...))
}

func lista(v any) []string {
	var s []string
	for _, x := range v.([]any) {
		s = append(s, x.(string))
	}
	return s
}

func TestCopiar(t *testing.T) {
	m := motor(t)
	destino := m.Pasta("destino")
	a := m.Grava("origem/a.txt", "conteúdo")
	antigo := time.Date(2001, 2, 3, 4, 5, 6, 0, time.UTC)
	os.Chmod(a, 0o640)
	os.Chtimes(a, antigo, antigo)
	m.Grava("origem/pasta/sub/b.txt", "bê")
	m.Grava("origem/pasta/c", "cê")
	os.Chmod(m.Casa+"/origem/pasta/sub", 0o750)
	os.Chtimes(m.Casa+"/origem/pasta/sub", antigo, antigo)
	os.Symlink("../a.txt", m.Casa+"/origem/pasta/atalho")
	os.Symlink("/nao/existe", m.Casa+"/origem/quebrado")

	f := feita(t, m, op("copiar", "origens", []string{a, m.Casa + "/origem/pasta", m.Casa + "/origem/quebrado"}, "destino", destino))
	if le(destino+"/a.txt") != "conteúdo" || le(destino+"/pasta/sub/b.txt") != "bê" || le(destino+"/pasta/c") != "cê" {
		t.Fatal("o conteúdo não chegou")
	}
	if info, _ := os.Stat(destino + "/a.txt"); info.Mode().Perm() != 0o640 || !info.ModTime().Equal(antigo) {
		t.Errorf("modo e mtime do arquivo: %v %v", info.Mode(), info.ModTime())
	}
	if info, _ := os.Stat(destino + "/pasta/sub"); info.Mode().Perm() != 0o750 || !info.ModTime().Equal(antigo) {
		t.Errorf("modo e mtime da pasta: %v %v", info.Mode(), info.ModTime())
	}
	if alvo, _ := os.Readlink(destino + "/pasta/atalho"); alvo != "../a.txt" {
		t.Errorf("o link é copiado como link: %q", alvo)
	}
	if alvo, _ := os.Readlink(destino + "/quebrado"); alvo != "/nao/existe" {
		t.Errorf("link quebrado: %q", alvo)
	}
	if le(a) != "conteúdo" {
		t.Error("a origem fica")
	}
	// O evento final tem todos os campos do contrato.
	for _, campo := range []string{"tarefa", "acao", "estado", "titulo", "itens", "itens_total", "bytes", "bytes_total", "atual",
		"velocidade", "destino", "criados", "erros", "sem_lixeira", "desfazivel"} {
		if _, tem := f[campo]; !tem {
			t.Errorf("falta %q no evento: %v", campo, f)
		}
	}
	if f["acao"] != "copiar" || f["titulo"] != "Copiando 3 itens para destino" || f["destino"] != destino ||
		f["itens"] != float64(7) || f["itens_total"] != float64(7) || f["bytes"] != float64(15) || f["bytes_total"] != float64(15) ||
		f["desfazivel"] != true || len(f["erros"].([]any)) != 0 || len(f["sem_lixeira"].([]any)) != 0 {
		t.Errorf("o evento final: %v", f)
	}
	if c := lista(f["criados"]); strings.Join(c, " ") != destino+"/a.txt "+destino+"/pasta "+destino+"/quebrado" {
		t.Errorf("criados: %v", c)
	}

	// Copiar para a própria pasta duplica, sem perguntar.
	f = feita(t, m, op("copiar", "origens", []string{a, m.Casa + "/origem/pasta"}, "destino", m.Casa+"/origem"))
	if le(m.Casa+"/origem/a (2).txt") != "conteúdo" || le(m.Casa+"/origem/pasta (2)/c") != "cê" {
		t.Errorf("duplicar: %v", f)
	}
	f = feita(t, m, op("copiar", "origens", []string{m.Casa + "/origem/a (2).txt"}, "destino", m.Casa+"/origem"))
	if c := lista(f["criados"]); c[0] != m.Casa+"/origem/a (3).txt" {
		t.Errorf("duplicar de novo: %v", c)
	}

	// Um especial é pulado, com erro na lista; o resto vai.
	syscall.Mkfifo(m.Casa+"/origem/cano", 0o600)
	f = faz(t, m, op("copiar", "origens", []string{m.Casa + "/origem/cano", m.Casa + "/origem/pasta/c"}, "destino", m.Pasta("outro")))
	erros := f["erros"].([]any)
	if f["estado"] != "erro" || len(erros) != 1 || erros[0].(map[string]any)["caminho"] != m.Casa+"/origem/cano" ||
		le(m.Casa+"/outro/c") != "cê" {
		t.Errorf("especial: %v", f)
	}
	if _, err := os.Lstat(m.Casa + "/outro/cano"); err == nil {
		t.Error("o fifo não é copiado")
	}
	// Origem que sumiu: erro na lista.
	if f = faz(t, m, op("copiar", "origens", []string{m.Casa + "/nao-ha"}, "destino", destino)); f["estado"] != "erro" {
		t.Errorf("origem que não existe: %v", f)
	}
}

func TestConflitos(t *testing.T) {
	m := motor(t)
	d := m.Pasta("d")
	a := m.Grava("o/a.txt", "novo")
	copiar := op("copiar", "origens", []string{a}, "destino", d)

	// O evento do conflito.
	m.Grava("d/a.txt", "velho!")
	id := pede(t, m, copiar)
	c := m.Evento("conflito", nil)
	origem, destino := c["origem"].(map[string]any), c["destino"].(map[string]any)
	if c["tarefa"] != id || c["conflito"] != float64(1) || c["acao"] != "copiar" || c["sugestao"] != "a (2).txt" || c["restam"] != float64(1) ||
		origem["caminho"] != a || origem["tamanho"] != float64(4) || destino["caminho"] != d+"/a.txt" || destino["tamanho"] != float64(6) ||
		destino["mtime"] == nil {
		t.Errorf("o evento conflito: %v", c)
	}
	m.Evento("tarefa", func(l prova.Linha) bool { return l["estado"] == "conflito" })
	if r := m.Pede(`{"op":"tarefas"}`); len(r["tarefas"].([]any)) != 1 || r["tarefas"].([]any)[0].(map[string]any)["estado"] != "conflito" {
		t.Errorf("tarefas: %v", r)
	}
	// Respostas tortas não destravam.
	if r := responde(m, c, "dançar"); r["erro"] == nil {
		t.Error("resposta desconhecida")
	}
	if r := responde(m, c, "renomear", "nome", "a/b"); r["campo"] != "nome" {
		t.Errorf("renomear com nome torto: %v", r)
	}
	if r := responde(m, c, "renomear"); r["campo"] != "nome" {
		t.Errorf("renomear sem nome: %v", r)
	}
	if r := m.Pede(op("responder", "tarefa", id, "conflito", 99, "resposta", "pular")); r["erro"] == nil {
		t.Error("conflito que não é o de agora")
	}
	// pular: fica como está.
	if r := responde(m, c, "pular"); r["tarefa"] != id {
		t.Errorf("responder: %v", r)
	}
	if f := fim(m, id); f["estado"] != "feita" || le(d+"/a.txt") != "velho!" || len(f["criados"].([]any)) != 0 || f["itens"] != float64(1) {
		t.Errorf("pular: %v", f)
	}
	if r := responde(m, c, "pular"); r["erro"] == nil {
		t.Error("responder duas vezes")
	}

	// substituir: o destino dá lugar à origem, e a tarefa não se desfaz.
	id = pede(t, m, copiar)
	responde(m, m.Evento("conflito", nil), "substituir")
	if f := fim(m, id); f["estado"] != "feita" || le(d+"/a.txt") != "novo" || f["desfazivel"] != false {
		t.Errorf("substituir: %v", f)
	}
	if es, _ := os.ReadDir(d); len(es) != 1 {
		t.Errorf("sobrou temporário: %v", es)
	}

	// manter: grava como a sugestão.
	id = pede(t, m, copiar)
	responde(m, m.Evento("conflito", nil), "manter")
	if f := fim(m, id); le(d+"/a (2).txt") != "novo" || lista(f["criados"])[0] != d+"/a (2).txt" || f["desfazivel"] != true {
		t.Errorf("manter: %v", f)
	}

	// renomear: para um nome ocupado pergunta de novo.
	id = pede(t, m, copiar)
	responde(m, m.Evento("conflito", nil), "renomear", "nome", "a (2).txt")
	c = m.Evento("conflito", nil)
	if c["conflito"] != float64(2) || c["destino"].(map[string]any)["nome"] != "a (2).txt" || c["sugestao"] != "a (3).txt" {
		t.Errorf("renomear para ocupado: %v", c)
	}
	responde(m, c, "renomear", "nome", "outro.txt")
	if fim(m, id); le(d+"/outro.txt") != "novo" {
		t.Error("renomear")
	}

	// cancelar: a tarefa inteira para.
	b := m.Grava("o/b.txt", "bê")
	id = pede(t, m, op("copiar", "origens", []string{a, b}, "destino", d))
	responde(m, m.Evento("conflito", nil), "cancelar")
	if f := fim(m, id); f["estado"] != "cancelada" || le(d+"/b.txt") != "<open "+d+"/b.txt: no such file or directory>" {
		t.Errorf("cancelar: %v", f)
	}
	// Cancelar a tarefa enquanto ela pergunta também destrava.
	id = pede(t, m, copiar)
	m.Evento("conflito", nil)
	m.Pede(op("tarefa-cancelar", "tarefa", id))
	if f := fim(m, id); f["estado"] != "cancelada" {
		t.Errorf("cancelar no conflito: %v", f)
	}

	// todos: a resposta vale para os próximos.
	m.Grava("d/b.txt", "velho b")
	m.Grava("d/c.txt", "velho c")
	cc := m.Grava("o/c.txt", "cê")
	id = pede(t, m, op("copiar", "origens", []string{a, b, cc}, "destino", d))
	c = m.Evento("conflito", nil)
	if c["restam"] != float64(3) {
		t.Errorf("restam: %v", c["restam"])
	}
	responde(m, c, "substituir", "todos", true)
	if f := fim(m, id); f["estado"] != "feita" || le(d+"/b.txt") != "bê" || le(d+"/c.txt") != "cê" {
		t.Errorf("todos: %v", f)
	}
	m.SemEvento("conflito", 0)
}

func TestConflitoDePastas(t *testing.T) {
	m := motor(t)
	d := m.Pasta("d")
	m.Grava("o/p/novo.txt", "n")
	m.Grava("o/p/igual.txt", "da origem")
	m.Grava("o/p/sub/x", "x")
	m.Grava("d/p/igual.txt", "do destino")
	m.Grava("d/p/so-la.txt", "fica")
	p := m.Casa + "/o/p"

	// Pasta sobre pasta mescla, e o conflito de dentro pergunta de novo.
	id := pede(t, m, op("copiar", "origens", []string{p}, "destino", d))
	c := m.Evento("conflito", nil)
	if c["origem"].(map[string]any)["tipo"] != "pasta" || c["sugestao"] != "p (2)" {
		t.Errorf("conflito de pasta: %v", c)
	}
	responde(m, c, "substituir")
	c = m.Evento("conflito", nil)
	if c["destino"].(map[string]any)["caminho"] != d+"/p/igual.txt" {
		t.Errorf("o conflito de dentro: %v", c)
	}
	responde(m, c, "pular")
	f := fim(m, id)
	if f["estado"] != "feita" || le(d+"/p/novo.txt") != "n" || le(d+"/p/igual.txt") != "do destino" || le(d+"/p/so-la.txt") != "fica" ||
		le(d+"/p/sub/x") != "x" || lista(f["criados"])[0] != d+"/p" {
		t.Errorf("mesclar: %v", f)
	}

	// Arquivo sobre pasta troca; pasta sobre arquivo também.
	arq := m.Grava("o2/p", "sou arquivo")
	id = pede(t, m, op("copiar", "origens", []string{arq}, "destino", d))
	responde(m, m.Evento("conflito", nil), "substituir")
	if fim(m, id); le(d+"/p") != "sou arquivo" {
		t.Error("arquivo sobre pasta")
	}
	id = pede(t, m, op("copiar", "origens", []string{p}, "destino", d))
	responde(m, m.Evento("conflito", nil), "substituir")
	if fim(m, id); le(d+"/p/sub/x") != "x" {
		t.Error("pasta sobre arquivo")
	}
	if es, _ := os.ReadDir(d); len(es) != 1 {
		t.Errorf("sobrou temporário: %v", es)
	}

	// Substituir a pasta que contém a origem não apaga a origem.
	m.Grava("z/x/x", "dentro")
	id = pede(t, m, op("copiar", "origens", []string{m.Casa + "/z/x/x"}, "destino", m.Casa+"/z"))
	responde(m, m.Evento("conflito", nil), "substituir")
	if f := fim(m, id); f["estado"] != "erro" || le(m.Casa+"/z/x/x") != "dentro" {
		t.Errorf("destino que contém a origem: %v", f)
	}
}

func TestCancelarNoMeioLimpaOParcial(t *testing.T) {
	m := motor(t)
	d := m.Pasta("d")
	pequeno := m.Grava("o/1-pequeno", "inteiro")
	grande := m.Grava("o/2-grande", strings.Repeat("x", 10*1024))
	depois := m.Grava("o/3-depois", "não chega")
	tamanhoDoBloco = 1024
	var noMeio int64
	aCadaBloco = func(tf *Tarefa, destino string, escritos int64) {
		if destino == d+"/2-grande" && escritos == 3*1024 {
			if info, err := os.Stat(destino); err == nil {
				noMeio = info.Size()
			}
			m.Manda(op("tarefa-cancelar", "tarefa", tf.ID))
		}
	}
	f := faz(t, m, op("copiar", "origens", []string{pequeno, grande, depois}, "destino", d))
	if f["estado"] != "cancelada" || noMeio != 3*1024 {
		t.Fatalf("cancelar no meio: %v (parcial de %d)", f, noMeio)
	}
	es, _ := os.ReadDir(d)
	if len(es) != 1 || le(d+"/1-pequeno") != "inteiro" {
		t.Errorf("o parcial sai, o que já foi inteiro fica: %v", es)
	}
	if f["bytes"] != float64(7+3*1024) || f["itens"] != float64(1) || f["desfazivel"] != true {
		t.Errorf("o progresso até o cancelamento: %v", f)
	}
	if r := m.Pede(op("tarefa-cancelar", "tarefa", f["tarefa"])); r["erro"] == nil {
		t.Error("cancelar tarefa que já acabou")
	}

	// Cancelar no meio de um substituir: o destino antigo continua inteiro.
	m.Grava("d/2-grande", "antigo")
	aCadaBloco = func(tf *Tarefa, destino string, escritos int64) {
		if escritos == 2*1024 {
			tf.cancela()
		}
	}
	id := pede(t, m, op("copiar", "origens", []string{grande}, "destino", d))
	responde(m, m.Evento("conflito", nil), "substituir")
	if f := fim(m, id); f["estado"] != "cancelada" || le(d+"/2-grande") != "antigo" {
		t.Errorf("cancelar um substituir: %v", f)
	}
	if es, _ := os.ReadDir(d); len(es) != 2 {
		t.Errorf("sobrou temporário: %v", es)
	}
}

func TestPausarERetomarEAndamento(t *testing.T) {
	m := motor(t)
	d := m.Pasta("d")
	grande := m.Grava("o/grande", strings.Repeat("y", 8*1024))
	tamanhoDoBloco = 1024
	intervalo = 0
	aCadaBloco = func(tf *Tarefa, destino string, escritos int64) {
		if escritos == 2*1024 {
			m.Manda(op("tarefa-pausar", "tarefa", tf.ID, "pausar", true))
		}
	}
	id := pede(t, m, op("copiar", "origens", []string{grande}, "destino", d))
	p := m.Evento("tarefa", func(l prova.Linha) bool { return l["estado"] == "pausada" })
	if p["bytes"] != float64(2*1024) || p["bytes_total"] != float64(8*1024) || p["atual"] != grande {
		t.Errorf("pausada: %v", p)
	}
	// Parada: nada anda.
	time.Sleep(60 * time.Millisecond)
	if info, _ := os.Stat(d + "/grande"); info.Size() != 2*1024 {
		t.Errorf("pausada, e andou: %d", info.Size())
	}
	aCadaBloco = nil
	m.Pede(op("tarefa-pausar", "tarefa", id, "pausar", false))
	f := fim(m, id)
	if f["estado"] != "feita" || le(d+"/grande") != strings.Repeat("y", 8*1024) || f["bytes"] != float64(8*1024) {
		t.Errorf("retomar: %v", f)
	}
	if r := m.Pede(op("tarefa-pausar", "tarefa", id, "pausar", true)); r["erro"] == nil {
		t.Error("pausar tarefa que já acabou")
	}
}

func TestFilaUmaPorVez(t *testing.T) {
	m := motor(t)
	d := m.Pasta("d")
	a := m.Grava("o/a", strings.Repeat("a", 4096))
	tamanhoDoBloco = 1024
	parou := make(chan bool, 1)
	aCadaBloco = func(tf *Tarefa, destino string, escritos int64) {
		if tf.ID == 1 && escritos == 1024 {
			m.Manda(op("tarefa-pausar", "tarefa", 1, "pausar", true))
			parou <- true
		}
	}
	um := pede(t, m, op("copiar", "origens", []string{a}, "destino", d))
	<-parou
	dois := pede(t, m, op("criar-pasta", "destino", d, "nome", "segunda"))
	tres := pede(t, m, op("criar-pasta", "destino", d, "nome", "terceira"))
	if e := m.Evento("tarefa", func(l prova.Linha) bool { return l["tarefa"] == dois }); e["estado"] != "fila" {
		t.Errorf("a segunda espera na fila: %v", e)
	}
	if r := m.Pede(`{"op":"tarefas"}`)["tarefas"].([]any); len(r) != 3 || r[0].(map[string]any)["estado"] != "pausada" || r[2].(map[string]any)["estado"] != "fila" {
		t.Errorf("tarefas: %v", r)
	}
	// Cancelar uma da fila: ela acaba cancelada sem rodar.
	m.Pede(op("tarefa-cancelar", "tarefa", tres))
	if _, err := os.Stat(d + "/segunda"); err == nil {
		t.Fatal("a segunda rodou antes de a primeira acabar")
	}
	m.Pede(op("tarefa-pausar", "tarefa", um, "pausar", false))
	if fim(m, um)["estado"] != "feita" || fim(m, dois)["estado"] != "feita" || fim(m, tres)["estado"] != "cancelada" {
		t.Error("a fila")
	}
	if _, err := os.Stat(d + "/terceira"); err == nil {
		t.Error("a cancelada na fila não roda")
	}
	if r := m.Pede(`{"op":"tarefas"}`)["tarefas"].([]any); len(r) != 0 {
		t.Errorf("tarefas no fim: %v", r)
	}
}

func TestFecharCancelaERemoveOParcial(t *testing.T) {
	m := motor(t)
	d := m.Pasta("d")
	a := m.Grava("o/a", strings.Repeat("a", 4096))
	tamanhoDoBloco = 1024
	parou := make(chan bool, 1)
	aCadaBloco = func(tf *Tarefa, destino string, escritos int64) {
		if escritos == 1024 {
			m.Manda(op("tarefa-pausar", "tarefa", tf.ID, "pausar", true))
			parou <- true
		}
	}
	pede(t, m, op("copiar", "origens", []string{a}, "destino", d))
	<-parou
	m.S.Fecha()
	if es, _ := os.ReadDir(d); len(es) != 0 {
		t.Errorf("a janela fechou: o parcial tinha de sair: %v", es)
	}
	if r := m.Pede(op("criar-pasta", "destino", d, "nome", "x")); r["erro"] == nil {
		t.Error("depois de fechar não entra tarefa")
	}
}

// outroDisco faz o rename falhar com EXDEV entre pastas diferentes, como entre
// dois sistemas de arquivos.
func outroDisco() {
	renomeia = func(de, para string) error {
		if filepath.Dir(de) != filepath.Dir(para) {
			return &os.LinkError{Op: "rename", Old: de, New: para, Err: syscall.EXDEV}
		}
		return os.Rename(de, para)
	}
}

func TestMover(t *testing.T) {
	m := motor(t)
	d := m.Pasta("d")
	a := m.Grava("o/a.txt", "á")
	m.Grava("o/p/x/y", "ípsilon")
	f := feita(t, m, op("mover", "origens", []string{a, m.Casa + "/o/p"}, "destino", d))
	if le(d+"/a.txt") != "á" || le(d+"/p/x/y") != "ípsilon" || f["titulo"] != "Movendo 2 itens para d" || f["desfazivel"] != true ||
		f["itens"] != float64(4) || f["itens_total"] != float64(4) {
		t.Errorf("mover: %v", f)
	}
	if es, _ := os.ReadDir(m.Casa + "/o"); len(es) != 0 {
		t.Errorf("a origem sai: %v", es)
	}
	// Mover para onde já está.
	if r := m.Pede(op("mover", "origens", []string{d + "/a.txt"}, "destino", d)); r["erro"] != "«a.txt» já está nessa pasta" {
		t.Errorf("mover para onde já está: %v", r)
	}
	// Conflito: substituir troca, e a tarefa não se desfaz.
	m.Grava("o/a.txt", "nova")
	id := pede(t, m, op("mover", "origens", []string{a}, "destino", d))
	c := m.Evento("conflito", nil)
	if c["acao"] != "mover" {
		t.Errorf("conflito do mover: %v", c)
	}
	responde(m, c, "substituir")
	if f := fim(m, id); le(d+"/a.txt") != "nova" || f["desfazivel"] != false || le(a) == "nova" {
		t.Errorf("mover substituindo: %v", f)
	}
	// Pasta sobre pasta mescla: o que foi pulado fica na origem, que por isso não sai.
	m.Grava("o/p/x/y", "outro")
	m.Grava("o/p/novo", "n")
	id = pede(t, m, op("mover", "origens", []string{m.Casa + "/o/p"}, "destino", d))
	responde(m, m.Evento("conflito", nil), "substituir", "todos", true)
	if f := fim(m, id); f["estado"] != "feita" || le(d+"/p/x/y") != "outro" || le(d+"/p/novo") != "n" {
		t.Errorf("mover mesclando: %v", f)
	}
	if _, err := os.Stat(m.Casa + "/o/p"); err == nil {
		t.Error("a pasta de origem, vazia, sai")
	}
	m.Grava("o/p/x/y", "fica")
	id = pede(t, m, op("mover", "origens", []string{m.Casa + "/o/p"}, "destino", d))
	responde(m, m.Evento("conflito", nil), "substituir")
	responde(m, m.Evento("conflito", nil), "substituir")
	responde(m, m.Evento("conflito", nil), "pular")
	if fim(m, id); le(m.Casa+"/o/p/x/y") != "fica" || le(d+"/p/x/y") != "outro" {
		t.Error("o pulado continua na origem")
	}
	// manter.
	m.Grava("o/a.txt", "terceira")
	id = pede(t, m, op("mover", "origens", []string{a}, "destino", d))
	responde(m, m.Evento("conflito", nil), "manter")
	if f := fim(m, id); le(d+"/a (2).txt") != "terceira" || f["desfazivel"] != true {
		t.Errorf("mover mantendo os dois: %v", f)
	}
}

func TestMoverEntreSistemasDeArquivos(t *testing.T) {
	m := motor(t)
	d := m.Pasta("d")
	m.Grava("o/p/a", "á")
	m.Grava("o/p/sub/b", strings.Repeat("b", 3000))
	os.Symlink("a", m.Casa+"/o/p/atalho")
	p := m.Casa + "/o/p"
	outroDisco()

	// A origem só some depois da conferência.
	conferiu := false
	aoConferir = func(de, para string) {
		conferiu = true
		if le(p+"/a") != "á" || le(p+"/sub/b") == "" || le(para+"/sub/b") != strings.Repeat("b", 3000) {
			t.Error("na conferência a origem ainda está inteira, e a cópia também")
		}
	}
	f := feita(t, m, op("mover", "origens", []string{p}, "destino", d))
	if !conferiu || le(d+"/p/a") != "á" || le(d+"/p/sub/b") != strings.Repeat("b", 3000) || f["bytes"] != float64(3002) || f["desfazivel"] != true {
		t.Errorf("mover com EXDEV: %v", f)
	}
	if alvo, _ := os.Readlink(d + "/p/atalho"); alvo != "a" {
		t.Error("o link vai como link")
	}
	if _, err := os.Lstat(p); err == nil {
		t.Error("depois de conferir, a origem sai")
	}

	// A conferência falha: a origem fica, a cópia sai.
	m.Grava("o/q/a", "inteiro")
	q := m.Casa + "/o/q"
	aoConferir = func(de, para string) { os.WriteFile(para+"/a", []byte("cort"), 0o644) }
	f = faz(t, m, op("mover", "origens", []string{q}, "destino", d))
	if f["estado"] != "erro" || le(q+"/a") != "inteiro" || f["desfazivel"] != false {
		t.Errorf("conferência que falha: %v", f)
	}
	if _, err := os.Lstat(d + "/q"); err == nil {
		t.Error("a cópia que não conferiu sai")
	}
	aoConferir = nil

	// Cancelar no meio: a origem fica, o parcial sai.
	grande := m.Grava("o/grande", strings.Repeat("g", 8192))
	tamanhoDoBloco = 1024
	aCadaBloco = func(tf *Tarefa, destino string, escritos int64) {
		if escritos == 2048 {
			tf.cancela()
		}
	}
	f = faz(t, m, op("mover", "origens", []string{grande}, "destino", d))
	if f["estado"] != "cancelada" || len(le(grande)) != 8192 {
		t.Errorf("cancelar um mover entre discos: %v", f)
	}
	if _, err := os.Lstat(d + "/grande"); err == nil {
		t.Error("o parcial do mover sai")
	}
	aCadaBloco = nil

	// Substituir entre discos: o destino antigo só some com o novo inteiro.
	m.Grava("d/grande", "antigo")
	aoConferir = func(de, para string) {
		if le(d+"/grande") != "antigo" {
			t.Error("o destino antigo sumiu antes da hora")
		}
	}
	id := pede(t, m, op("mover", "origens", []string{grande}, "destino", d))
	responde(m, m.Evento("conflito", nil), "substituir")
	if f := fim(m, id); f["estado"] != "feita" || len(le(d+"/grande")) != 8192 {
		t.Errorf("substituir entre discos: %v", f)
	}
	if es, _ := os.ReadDir(d); len(es) != 2 {
		t.Errorf("sobrou temporário: %v", es)
	}
	if _, err := os.Lstat(grande); err == nil {
		t.Error("a origem sai")
	}

	// A lixeira não atravessa discos: vai em sem_lixeira, e nada é feito.
	x := m.Grava("o/x", "x")
	f = faz(t, m, op("lixeira", "origens", []string{x}))
	if f["estado"] != "feita" || lista(f["sem_lixeira"])[0] != x || le(x) != "x" || lixeira.Contar() != 0 {
		t.Errorf("lixeira em outro sistema de arquivos: %v", f)
	}
}

func TestRecusaCopiarParaDentroDeSi(t *testing.T) {
	m := motor(t)
	m.Grava("a/b/c.txt", "c")
	a := m.Casa + "/a"
	os.Symlink(a+"/b", m.Casa+"/atalho")
	os.Symlink(a, m.Casa+"/a-por-link")
	for _, verbo := range []string{"copiar", "mover"} {
		for _, destino := range []string{a, a + "/b", m.Casa + "/atalho", m.Casa + "/a-por-link"} {
			r := m.Pede(op(verbo, "origens", []string{a}, "destino", destino))
			if r["erro"] != "não dá para "+verbo+" uma pasta para dentro dela mesma" {
				t.Errorf("%s %s → %s: %v", verbo, a, destino, r)
			}
		}
	}
	// Uma pasta com nome parecido não é "dentro".
	m.Pasta("ab")
	feita(t, m, op("copiar", "origens", []string{a}, "destino", m.Casa+"/ab"))
	if le(a+"/b/c.txt") != "c" || le(m.Casa+"/ab/a/b/c.txt") != "c" {
		t.Error("nada mudou na origem, e a cópia vizinha foi")
	}
	if es, _ := os.ReadDir(a + "/b"); len(es) != 1 {
		t.Errorf("a recusa não cria nada: %v", es)
	}
}

func TestApagarNaoSegueLinks(t *testing.T) {
	m := motor(t)
	fora := m.Grava("fora/precioso.txt", "fica")
	m.Grava("p/dentro/x", "x")
	m.Grava("p/y", strings.Repeat("y", 100))
	os.Symlink(m.Casa+"/fora", m.Casa+"/p/para-fora")
	os.Symlink(fora, m.Casa+"/p/dentro/para-arquivo")
	fechada := m.Pasta("p/fechada")
	m.Grava("p/fechada/z", "z")
	os.Chmod(fechada, 0)
	t.Cleanup(func() { os.Chmod(fechada, 0o755) })

	f := feita(t, m, op("apagar", "origens", []string{m.Casa + "/p"}))
	if _, err := os.Lstat(m.Casa + "/p"); err == nil {
		t.Error("a pasta sai inteira (até a subpasta sem permissão)")
	}
	if le(fora) != "fica" {
		t.Fatal("o alvo do link, fora da pasta, tem de ficar")
	}
	if f["titulo"] != "Apagando «p»" || f["itens"] != float64(8) || f["itens_total"].(float64) < 7 || f["bytes"] != float64(102) ||
		f["desfazivel"] != false || f["destino"] != nil {
		t.Errorf("o evento do apagar: %v", f)
	}
	// Apagar um link para pasta apaga só o link.
	os.Symlink(m.Casa+"/fora", m.Casa+"/atalho")
	feita(t, m, op("apagar", "origens", []string{m.Casa + "/atalho"}))
	if _, err := os.Lstat(m.Casa + "/atalho"); err == nil || le(fora) != "fica" {
		t.Error("link para pasta: só o link sai")
	}
	// O que não existe mais não é erro de apagar.
	if f := faz(t, m, op("apagar", "origens", []string{m.Casa + "/nao-ha"})); f["estado"] != "feita" {
		t.Errorf("apagar o que já sumiu: %v", f)
	}
}

func TestCriarERenomear(t *testing.T) {
	m := motor(t)
	d := m.Pasta("d")
	f := feita(t, m, op("criar-pasta", "destino", d, "nome", "Nova pasta"))
	if info, err := os.Stat(d + "/Nova pasta"); err != nil || !info.IsDir() || lista(f["criados"])[0] != d+"/Nova pasta" || f["desfazivel"] != true {
		t.Errorf("criar-pasta: %v", f)
	}
	feita(t, m, op("criar-arquivo", "destino", d, "nome", "vazio.txt", "modelo", nil))
	if le(d+"/vazio.txt") != "" {
		t.Error("criar-arquivo vazio")
	}
	modelo := m.Grava("Modelos/carta.txt", "Prezado,")
	velho := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	os.Chtimes(modelo, velho, velho)
	feita(t, m, op("criar-arquivo", "destino", d, "nome", "minha carta.txt", "modelo", modelo))
	if info, _ := os.Stat(d + "/minha carta.txt"); le(d+"/minha carta.txt") != "Prezado," || info.ModTime().Before(time.Now().Add(-time.Minute)) {
		t.Error("criar-arquivo a partir do modelo (com a data de agora)")
	}
	for _, c := range []struct{ pedido, campo string }{
		{op("criar-pasta", "destino", d, "nome", "Nova pasta"), "nome"},
		{op("criar-arquivo", "destino", d, "nome", "vazio.txt"), "nome"},
		{op("criar-pasta", "destino", d, "nome", "a/b"), "nome"},
		{op("criar-pasta", "destino", d, "nome", ".."), "nome"},
		{op("criar-pasta", "destino", d, "nome", ""), "nome"},
		{op("criar-arquivo", "destino", d, "nome", "a\x00b"), "nome"},
		{op("criar-pasta", "destino", d+"/vazio.txt", "nome", "x"), ""},
		{op("criar-pasta", "destino", "relativo", "nome", "x"), ""},
		{op("criar-arquivo", "destino", d, "nome", "x", "modelo", d), ""},
		{op("criar-arquivo", "destino", d, "nome", "x", "modelo", "relativo"), ""},
		{op("renomear", "caminho", d+"/vazio.txt", "nome", "Nova pasta"), "nome"},
		{op("renomear", "caminho", d+"/vazio.txt", "nome", "vazio.txt"), "nome"},
		{op("renomear", "caminho", d+"/vazio.txt", "nome", "a/b"), "nome"},
		{op("renomear", "caminho", d+"/vazio.txt", "nome", "."), "nome"},
		{op("renomear", "caminho", d+"/nao-ha", "nome", "x"), ""},
		{op("renomear", "caminho", "vazio.txt", "nome", "x"), ""},
		{op("renomear", "caminho", d+"/../d/vazio.txt", "nome", "x"), ""},
	} {
		r := m.Pede(c.pedido)
		if r["erro"] == nil || r["tarefa"] != nil || (c.campo != "" && r["campo"] != c.campo) {
			t.Errorf("%s: %v", c.pedido, r)
		}
	}
	if es, _ := os.ReadDir(d); len(es) != 3 {
		t.Errorf("os pedidos recusados não criam nada: %v", es)
	}
	f = feita(t, m, op("renomear", "caminho", d+"/vazio.txt", "nome", "cheio.txt"))
	if _, err := os.Stat(d + "/cheio.txt"); err != nil || lista(f["criados"])[0] != d+"/cheio.txt" || f["titulo"] != "Renomeando «vazio.txt» para «cheio.txt»" {
		t.Errorf("renomear: %v", f)
	}
}

func TestLigar(t *testing.T) {
	m := motor(t)
	d := m.Pasta("d")
	a := m.Grava("o/a.txt", "á")
	f := feita(t, m, op("ligar", "origens", []string{a, m.Casa + "/o"}, "destino", d))
	if alvo, _ := os.Readlink(d + "/a.txt"); alvo != a || f["desfazivel"] != true {
		t.Errorf("ligar: %q %v", alvo, f)
	}
	if alvo, _ := os.Readlink(d + "/o"); alvo != m.Casa+"/o" {
		t.Error("link para pasta")
	}
	// Na própria pasta ganha um nome novo; com o destino ocupado, pergunta.
	feita(t, m, op("ligar", "origens", []string{a}, "destino", m.Casa+"/o"))
	if alvo, _ := os.Readlink(m.Casa + "/o/a (2).txt"); alvo != a {
		t.Error("link na própria pasta")
	}
	id := pede(t, m, op("ligar", "origens", []string{a}, "destino", d))
	responde(m, m.Evento("conflito", nil), "manter")
	if fim(m, id); le(d+"/a (2).txt") != "á" {
		t.Error("ligar mantendo os dois")
	}
	m.Grava("d/pasta-no-caminho/x", "x")
	m.Pasta("o/pasta-no-caminho")
	id = pede(t, m, op("ligar", "origens", []string{m.Casa + "/o/pasta-no-caminho"}, "destino", d))
	responde(m, m.Evento("conflito", nil), "substituir")
	fim(m, id)
	if alvo, _ := os.Readlink(d + "/pasta-no-caminho"); alvo != m.Casa+"/o/pasta-no-caminho" {
		t.Error("ligar substituindo uma pasta (não mescla)")
	}
}

func TestLixeira(t *testing.T) {
	m := motor(t)
	raiz := m.Casa + "/.local/share/Trash"
	a := m.Grava("docs/meu arquivo é.txt", "um")
	m.Grava("docs/p/x", "x")
	f := feita(t, m, op("lixeira", "origens", []string{a, m.Casa + "/docs/p"}))
	if f["titulo"] != "Mandando 2 itens para a lixeira" || f["desfazivel"] != true || f["itens"] != float64(2) {
		t.Errorf("lixeira: %v", f)
	}
	if e := m.Evento("lixeira", nil); e["itens"] != float64(2) {
		t.Errorf("evento lixeira: %v", e)
	}
	if !strings.Contains(le(raiz+"/info/meu arquivo é.txt.trashinfo"), "Path="+m.Casa+"/docs/meu%20arquivo%20%C3%A9.txt\n") {
		t.Errorf("o .trashinfo: %q", le(raiz+"/info/meu arquivo é.txt.trashinfo"))
	}
	if _, err := os.Lstat(a); err == nil || le(raiz+"/files/p/x") != "x" {
		t.Error("os arquivos foram para a lixeira")
	}

	// Restaurar: volta ao lugar, e a ficha sai.
	naLixeira := raiz + "/files/meu arquivo é.txt"
	f = feita(t, m, op("restaurar", "itens", []string{naLixeira}))
	if le(a) != "um" || lista(f["criados"])[0] != a || f["titulo"] != "Restaurando «meu arquivo é.txt»" {
		t.Errorf("restaurar: %v", f)
	}
	if _, err := os.Lstat(raiz + "/info/meu arquivo é.txt.trashinfo"); err == nil {
		t.Error("a ficha sai ao restaurar")
	}
	if e := m.Evento("lixeira", nil); e["itens"] != float64(1) {
		t.Errorf("evento lixeira depois de restaurar: %v", e)
	}
	// Restaurar para uma pasta que não existe mais: ela é recriada.
	feita(t, m, op("lixeira", "origens", []string{a}))
	os.Remove(m.Casa + "/docs")
	feita(t, m, op("restaurar", "itens", []string{naLixeira}))
	if le(a) != "um" {
		t.Error("restaurar recria a pasta de origem")
	}

	// Conflito ao restaurar: o lugar voltou a ser ocupado.
	feita(t, m, op("lixeira", "origens", []string{a}))
	m.Grava("docs/meu arquivo é.txt", "o novo")
	id := pede(t, m, op("restaurar", "itens", []string{naLixeira}))
	c := m.Evento("conflito", nil)
	if c["acao"] != "restaurar" || c["origem"].(map[string]any)["caminho"] != naLixeira || c["destino"].(map[string]any)["caminho"] != a ||
		c["sugestao"] != "meu arquivo é (2).txt" {
		t.Errorf("conflito ao restaurar: %v", c)
	}
	responde(m, c, "pular")
	if fim(m, id); le(a) != "o novo" || le(naLixeira) != "um" {
		t.Error("pular: os dois ficam onde estão")
	}
	id = pede(t, m, op("restaurar", "itens", []string{naLixeira}))
	responde(m, m.Evento("conflito", nil), "manter")
	if fim(m, id); le(m.Casa+"/docs/meu arquivo é (2).txt") != "um" || le(a) != "o novo" {
		t.Error("manter os dois ao restaurar")
	}
	feita(t, m, op("lixeira", "origens", []string{m.Casa + "/docs/meu arquivo é (2).txt"}))
	os.WriteFile(m.Casa+"/docs/meu arquivo é (2).txt", []byte("ocupou"), 0o644)
	id = pede(t, m, op("restaurar", "itens", []string{raiz + "/files/meu arquivo é (2).txt"}))
	responde(m, m.Evento("conflito", nil), "substituir")
	if fim(m, id); le(m.Casa+"/docs/meu arquivo é (2).txt") != "um" {
		t.Error("substituir ao restaurar")
	}

	// Só restaura o que está numa lixeira.
	for _, ruim := range [][]string{{a}, {raiz + "/files/nao-ha"}, {"relativo"}, {}, {raiz + "/info/p.trashinfo"}} {
		for _, verbo := range []string{"restaurar", "lixeira-apagar"} {
			if r := m.Pede(op(verbo, "itens", ruim)); r["erro"] == nil {
				t.Errorf("%s %v devia falhar: %v", verbo, ruim, r)
			}
		}
	}
	if le(a) != "o novo" {
		t.Fatal("lixeira-apagar de um arquivo de fora não apaga nada")
	}

	// Apagar um item da lixeira.
	f = feita(t, m, op("lixeira-apagar", "itens", []string{raiz + "/files/p"}))
	if _, err := os.Lstat(raiz + "/files/p"); err == nil || f["desfazivel"] != false {
		t.Errorf("lixeira-apagar: %v", f)
	}
	if _, err := os.Lstat(raiz + "/info/p.trashinfo"); err == nil {
		t.Error("a ficha sai junto")
	}
	if lixeira.Contar() != 0 {
		t.Errorf("a lixeira ficou com %d", lixeira.Contar())
	}

	// Esvaziar: tudo, com os órfãos, das duas lixeiras; as pastas da lixeira ficam.
	disco := m.Pasta("disco")
	m.Montagem(disco, "/dev/sdz1")
	feita(t, m, op("lixeira", "origens", []string{a, m.Grava("disco/no-disco", "d")}))
	doDisco := disco + "/.Trash-" + strconv.Itoa(os.Getuid())
	if le(doDisco+"/files/no-disco") != "d" {
		t.Fatal("a lixeira do outro disco")
	}
	m.Grava(".local/share/Trash/files/orfao/x", "x")
	m.Grava(".local/share/Trash/info/fantasma.trashinfo", "[Trash Info]\nPath=/x\n")
	fora := m.Grava("fora", "fica")
	os.Symlink(fora, raiz+"/files/orfao/link")
	f = feita(t, m, op("lixeira-esvaziar"))
	for _, pasta := range []string{raiz + "/files", raiz + "/info", doDisco + "/files", doDisco + "/info"} {
		if es, err := os.ReadDir(pasta); err != nil || len(es) != 0 {
			t.Errorf("esvaziar: %s ficou com %v (%v)", pasta, es, err)
		}
	}
	if f["titulo"] != "Esvaziando a lixeira" || le(fora) != "fica" || le(m.Casa+"/docs/meu arquivo é (2).txt") != "um" {
		t.Errorf("esvaziar: %v", f)
	}
	var ultimo prova.Linha
	for range 8 {
		ultimo = m.Evento("lixeira", nil)
		if ultimo["itens"] == float64(0) {
			break
		}
	}
	if ultimo["itens"] != float64(0) {
		t.Errorf("evento lixeira no fim: %v", ultimo)
	}
}

func TestSemLixeira(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("o root grava em qualquer lugar")
	}
	m := motor(t)
	disco := m.Pasta("disco")
	m.Montagem(disco, "/dev/sdz1")
	b := m.Grava("disco/b.txt", "b")
	a := m.Grava("a.txt", "a")
	os.Chmod(disco, 0o555)
	t.Cleanup(func() { os.Chmod(disco, 0o755) })
	f := faz(t, m, op("lixeira", "origens", []string{b, a}))
	if f["estado"] != "feita" || len(lista(f["sem_lixeira"])) != 1 || lista(f["sem_lixeira"])[0] != b || le(b) != "b" {
		t.Errorf("sem lixeira possível: %v", f)
	}
	if _, err := os.Lstat(a); err == nil {
		t.Error("o que tem lixeira vai")
	}
}

func historico(m *prova.Motor) prova.Linha { return m.Evento("historico", nil) }

func TestDesfazerERefazer(t *testing.T) {
	m := motor(t)
	d := m.Pasta("d")
	if r := m.Pede(`{"op":"desfazer"}`); r["erro"] != "nada para desfazer" {
		t.Errorf("histórico vazio: %v", r)
	}
	if r := m.Pede(`{"op":"refazer"}`); r["erro"] != "nada para refazer" {
		t.Errorf("histórico vazio: %v", r)
	}
	desfaz := func(descricao string) prova.Linha {
		t.Helper()
		r := m.Pede(`{"op":"desfazer"}`)
		if r["descricao"] != descricao || r["tarefa"] == nil {
			t.Fatalf("desfazer: esperava %q, veio %v", descricao, r)
		}
		return fim(m, r["tarefa"])
	}
	refaz := func(descricao string) prova.Linha {
		t.Helper()
		r := m.Pede(`{"op":"refazer"}`)
		if r["descricao"] != descricao || r["tarefa"] == nil {
			t.Fatalf("refazer: esperava %q, veio %v", descricao, r)
		}
		return fim(m, r["tarefa"])
	}
	existe := func(c string) bool { _, err := os.Lstat(c); return err == nil }

	// criar-pasta
	feita(t, m, op("criar-pasta", "destino", d, "nome", "nova"))
	if h := historico(m); h["desfazer"] != "Criar a pasta «nova»" || h["refazer"] != nil {
		t.Errorf("evento historico: %v", h)
	}
	if f := desfaz("Criar a pasta «nova»"); f["estado"] != "feita" || existe(d+"/nova") || f["acao"] != "desfazer" || f["desfazivel"] != false {
		t.Errorf("desfazer criar-pasta: %v", f)
	}
	if h := historico(m); h["desfazer"] != nil {
		t.Errorf("ao pedir, sai da pilha: %v", h)
	}
	if h := historico(m); h["refazer"] != "Criar a pasta «nova»" {
		t.Errorf("ao acabar, pode refazer: %v", h)
	}
	if f := refaz("Criar a pasta «nova»"); !existe(d+"/nova") || f["acao"] != "refazer" || f["desfazivel"] != true {
		t.Errorf("refazer criar-pasta: %v", f)
	}
	// A pasta que ganhou conteúdo não sai.
	m.Grava("d/nova/x", "x")
	if f := desfaz("Criar a pasta «nova»"); f["estado"] != "erro" || !existe(d+"/nova/x") {
		t.Errorf("pasta que não está mais vazia: %v", f)
	}
	if r := m.Pede(`{"op":"refazer"}`); r["erro"] == nil {
		t.Error("um desfazer que falhou não vira refazer")
	}

	// criar-arquivo: só sai se não mudou.
	feita(t, m, op("criar-arquivo", "destino", d, "nome", "a.txt"))
	desfaz("Criar «a.txt»")
	if existe(d + "/a.txt") {
		t.Error("desfazer criar-arquivo")
	}
	refaz("Criar «a.txt»")
	os.WriteFile(d+"/a.txt", []byte("escrevi"), 0o644)
	if f := desfaz("Criar «a.txt»"); f["estado"] != "erro" || le(d+"/a.txt") != "escrevi" {
		t.Errorf("arquivo criado e depois escrito: %v", f)
	}

	// renomear
	feita(t, m, op("renomear", "caminho", d+"/a.txt", "nome", "b.txt"))
	desfaz("Renomear «a.txt»")
	if le(d+"/a.txt") != "escrevi" || existe(d+"/b.txt") {
		t.Error("desfazer renomear")
	}
	refaz("Renomear «a.txt»")
	if le(d+"/b.txt") != "escrevi" {
		t.Error("refazer renomear")
	}
	// O nome antigo voltou a ser ocupado: não passa por cima.
	m.Grava("d/a.txt", "outro")
	if f := desfaz("Renomear «a.txt»"); f["estado"] != "erro" || le(d+"/a.txt") != "outro" || le(d+"/b.txt") != "escrevi" {
		t.Errorf("desfazer renomear com o lugar ocupado: %v", f)
	}

	// mover (inclusive quando a pasta de origem sumiu)
	o := m.Pasta("o")
	feita(t, m, op("mover", "origens", []string{d + "/a.txt", d + "/b.txt"}, "destino", o))
	os.Remove(d + "/nova/x")
	desfaz("Mover 2 itens")
	if le(d+"/a.txt") != "outro" || le(d+"/b.txt") != "escrevi" || existe(o+"/a.txt") {
		t.Error("desfazer mover")
	}
	refaz("Mover 2 itens")
	if le(o+"/a.txt") != "outro" || existe(d+"/a.txt") {
		t.Error("refazer mover")
	}

	// lixeira: desfaz restaurando.
	feita(t, m, op("lixeira", "origens", []string{o + "/a.txt"}))
	if existe(o + "/a.txt") {
		t.Fatal("foi para a lixeira")
	}
	desfaz("Mandar «a.txt» para a lixeira")
	if le(o+"/a.txt") != "outro" || lixeira.Contar() != 0 {
		t.Error("desfazer lixeira restaura")
	}
	refaz("Mandar «a.txt» para a lixeira")
	if existe(o+"/a.txt") || lixeira.Contar() != 1 {
		t.Error("refazer lixeira")
	}
	desfaz("Mandar «a.txt» para a lixeira")

	// ligar
	feita(t, m, op("ligar", "origens", []string{o + "/a.txt"}, "destino", d))
	desfaz("Criar link para «a.txt»")
	if existe(d+"/a.txt") || le(o+"/a.txt") != "outro" {
		t.Error("desfazer ligar apaga só o link")
	}

	// copiar: desfaz apagando o que copiou, menos o que mudou desde então.
	m.Grava("fonte/p/um", "1")
	m.Grava("fonte/p/dois", "2")
	m.Grava("fonte/solto", "s")
	feita(t, m, op("copiar", "origens", []string{m.Casa + "/fonte/p", m.Casa + "/fonte/solto"}, "destino", d))
	desfaz("Copiar 2 itens")
	if existe(d+"/p") || existe(d+"/solto") || le(m.Casa+"/fonte/p/um") != "1" {
		t.Error("desfazer copiar apaga a cópia inteira, e só ela")
	}
	refaz("Copiar 2 itens")
	os.WriteFile(d+"/p/dois", []byte("mexi aqui"), 0o644)
	f := desfaz("Copiar 2 itens")
	if f["estado"] != "erro" || le(d+"/p/dois") != "mexi aqui" || existe(d+"/p/um") || existe(d+"/solto") {
		t.Errorf("a cópia modificada fica: %v", f)
	}
	// Mesmo tamanho, outra data: também mudou.
	m.Grava("fonte/t", "abc")
	feita(t, m, op("copiar", "origens", []string{m.Casa + "/fonte/t"}, "destino", d))
	os.WriteFile(d+"/t", []byte("xyz"), 0o644)
	os.Chtimes(d+"/t", time.Now().Add(time.Hour), time.Now().Add(time.Hour))
	if f := desfaz("Copiar «t»"); f["estado"] != "erro" || le(d+"/t") != "xyz" {
		t.Errorf("mesma tamanho, outra mtime: %v", f)
	}

	// Uma tarefa nova esvazia o refazer; apagar e substituir não entram no histórico.
	feita(t, m, op("criar-pasta", "destino", d, "nome", "um"))
	desfaz("Criar a pasta «um»")
	feita(t, m, op("criar-pasta", "destino", d, "nome", "dois"))
	if r := m.Pede(`{"op":"refazer"}`); r["erro"] != "nada para refazer" {
		t.Errorf("tarefa nova esvazia o refazer: %v", r)
	}
	feita(t, m, op("apagar", "origens", []string{d + "/t"}))
	id := pede(t, m, op("copiar", "origens", []string{m.Casa + "/fonte/solto"}, "destino", m.Casa+"/fonte/p"))
	_ = id
	feita(t, m, op("copiar", "origens", []string{m.Casa + "/fonte/t"}, "destino", o))
	m.Grava("fonte/t", "novo")
	id = pede(t, m, op("copiar", "origens", []string{m.Casa + "/fonte/t"}, "destino", o))
	responde(m, m.Evento("conflito", nil), "substituir")
	fim(m, id)
	if r := m.Pede(`{"op":"desfazer"}`); r["descricao"] != "Copiar «t»" {
		t.Errorf("o topo é a cópia de antes do substituir: %v", r)
	} else if f := fim(m, r["tarefa"]); f["estado"] != "erro" || le(o+"/t") != "novo" {
		t.Errorf("e o arquivo substituído depois não é apagado por ela: %v", f)
	}
}

func TestValidacoes(t *testing.T) {
	m := motor(t)
	d := m.Pasta("d")
	a := m.Grava("a.txt", "a")
	for _, pedido := range []string{
		op("copiar", "origens", []string{"relativo"}, "destino", d),
		op("copiar", "origens", []string{d + "/../a.txt"}, "destino", d),
		op("copiar", "origens", []string{"/a\x00b"}, "destino", d),
		op("copiar", "origens", []string{a + "/"}, "destino", d),
		op("copiar", "origens", []string{}, "destino", d),
		op("copiar", "destino", d),
		op("copiar", "origens", []string{a}, "destino", "d"),
		op("copiar", "origens", []string{a}, "destino", ""),
		op("copiar", "origens", []string{a}, "destino", a),
		op("copiar", "origens", []string{a}, "destino", d+"/nao-ha"),
		op("mover", "origens", []string{"./a.txt"}, "destino", d),
		op("mover", "origens", []string{a}, "destino", d+"//x"),
		op("ligar", "origens", []string{"a.txt"}, "destino", d),
		op("ligar", "origens", []string{a}, "destino", "../d"),
		op("lixeira", "origens", []string{"a.txt"}),
		op("lixeira", "origens", []string{a, "/x/../y"}),
		op("apagar", "origens", []string{"a.txt"}),
		op("apagar", "origens", []string{a, "/x\x00"}),
		op("apagar", "origens", []string{d + "/."}),
	} {
		r := m.Pede(pedido)
		if r["erro"] == nil || r["tarefa"] != nil || r["recusado"] != nil {
			t.Errorf("%s: %v", pedido, r)
		}
	}
	if le(a) != "a" {
		t.Fatal("nenhum pedido torto mexe em nada")
	}
	if es, _ := os.ReadDir(d); len(es) != 0 {
		t.Errorf("nada foi criado: %v", es)
	}
}

func TestRecusasDeSeguranca(t *testing.T) {
	m := motor(t)
	precioso := m.Grava("Documentos/precioso.txt", "fica")
	d := m.Pasta("d")
	acima := filepath.Dir(m.Casa)
	for _, alvo := range []string{"/", m.Casa, acima, filepath.Dir(acima)} {
		for _, pedido := range []string{
			op("apagar", "origens", []string{alvo}),
			op("apagar", "origens", []string{d, alvo}),
			op("lixeira", "origens", []string{alvo}),
			op("lixeira", "origens", []string{d, alvo}),
			op("mover", "origens", []string{alvo}, "destino", d),
			op("renomear", "caminho", alvo, "nome", "outro"),
		} {
			r := m.Pede(pedido)
			if r["erro"] == nil || r["recusado"] != true || r["tarefa"] != nil {
				t.Errorf("%s devia ser recusado: %v", pedido, r)
			}
		}
	}
	// Caminho vazio e lista vazia: erro, sem tarefa.
	for _, pedido := range []string{
		op("apagar", "origens", []string{""}), op("apagar", "origens", []string{}), op("apagar"),
		op("lixeira", "origens", []string{""}), op("lixeira", "origens", []string{}), op("lixeira"),
		op("mover", "origens", []string{""}, "destino", d), op("mover", "origens", []string{}, "destino", d),
		op("renomear", "caminho", "", "nome", "x"), op("renomear", "nome", "x"),
	} {
		if r := m.Pede(pedido); r["erro"] == nil || r["tarefa"] != nil {
			t.Errorf("%s devia falhar: %v", pedido, r)
		}
	}
	if m.S.Codigo() != 2 {
		t.Errorf("uma recusa deixa o código de saída 2: %d", m.S.Codigo())
	}
	// Nada foi tocado.
	if le(precioso) != "fica" {
		t.Fatal("a pasta pessoal foi mexida")
	}
	if info, err := os.Stat(d); err != nil || !info.IsDir() {
		t.Error("a pasta que ia junto no pedido recusado também fica")
	}
	if lixeira.Contar() != 0 {
		t.Error("nada foi para a lixeira")
	}
	// A trava de baixo também recusa, mesmo se alguém passar da de cima.
	for _, alvo := range []string{"", "/", m.Casa, acima} {
		chamou := false
		removeArvore(alvo, func(_ string, _ os.FileInfo, err error) bool {
			chamou = true
			if err == nil {
				t.Errorf("removeArvore(%q) devia recusar", alvo)
			}
			return true
		})
		if !chamou || le(precioso) != "fica" {
			t.Errorf("removeArvore(%q)", alvo)
		}
	}
	// Copiar a pasta pessoal (não mexe nela) pode; para dentro dela mesma, não.
	if r := m.Pede(op("copiar", "origens", []string{m.Casa}, "destino", d)); r["erro"] != "não dá para copiar uma pasta para dentro dela mesma" {
		t.Errorf("copiar a pasta pessoal para dentro dela: %v", r)
	}
}

func TestChamadaSoPulaOConflito(t *testing.T) {
	m := motor(t)
	m.S.SemID = true
	d := m.Pasta("d")
	a := m.Grava("a.txt", "novo")
	m.Grava("d/a.txt", "velho")
	// Sem id não dá para usar o Pede: manda direto e espera o evento final.
	m.S.Atender([]byte(prova.J(map[string]any{"op": "copiar", "caminhos": []string{a}, "destino": d})))
	f := m.Evento("tarefa", final)
	if f["estado"] != "erro" || le(d+"/a.txt") != "velho" {
		t.Errorf("sem janela para responder, o conflito é pulado e vira erro: %v", f)
	}
	if m.S.Codigo() != 1 {
		t.Errorf("tarefa com erro marca o código de saída: %d", m.S.Codigo())
	}
	m.S.Atender([]byte(prova.J(map[string]any{"op": "apagar", "caminhos": []string{d + "/a.txt"}})))
	if f := m.Evento("tarefa", final); f["estado"] != "feita" || le(d+"/a.txt") == "velho" {
		t.Errorf("os caminhos soltos valem de origens: %v", f)
	}
}

func TestNomes(t *testing.T) {
	if quantos([]string{"/a/b.txt"}) != "«b.txt»" || quantos([]string{"/a", "/b"}) != "2 itens" || nomeDaPasta("/") != "a raiz" {
		t.Error("textos")
	}
	if a, b := aoLado("/x/y"), aoLado("/x/y"); a == b || filepath.Dir(a) != "/x" || !strings.HasPrefix(filepath.Base(a), ".mkfiles-") {
		t.Errorf("aoLado: %s %s", a, b)
	}
	if n := filepath.Base(aoLado("/x/" + strings.Repeat("ç", 127))); len(n) > 255 {
		t.Errorf("aoLado de nome comprido: %d", len(n))
	}
	casos := map[error]string{
		syscall.ENOSPC: "o disco encheu", syscall.EROFS: "o disco é só de leitura", os.ErrPermission: "permissão negada",
		fmt.Errorf("x: %w", os.ErrNotExist):                              "não existe mais",
		&os.PathError{Op: "open", Path: "/segredo", Err: syscall.EISDIR}: "is a directory",
	}
	for err, quer := range casos {
		if frase(err) != quer {
			t.Errorf("frase(%v) = %q; esperado %q", err, frase(err), quer)
		}
	}
}
