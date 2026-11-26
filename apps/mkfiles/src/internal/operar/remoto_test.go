package operar

import (
	"os"
	"strings"
	"syscall"
	"testing"

	"mkfiles/internal/prova"
)

// Um remoto é uma pasta do gvfs: apagar e copiar lá dentro funcionam como em
// qualquer pasta, mas não há lixeira, a raiz do remoto não se apaga e a rede
// que cai tem frase própria.
func TestOperacoesEmRemoto(t *testing.T) {
	m := motor(t)
	gvfs := m.Pasta("fora/gvfs")
	t.Setenv("MKFILES_GVFS", gvfs)
	raiz := m.Pasta("fora/gvfs/sftp:host=servidor,user=ana")
	a := m.Grava(raiz+"/docs/a.txt", "um")
	m.Grava(raiz+"/docs/sub/b.txt", "dois")

	// Lixeira: não há num remoto. Nada é mexido, e a janela pergunta.
	f := faz(t, m, op("lixeira", "origens", []string{a}))
	if sem, _ := f["sem_lixeira"].([]any); len(sem) != 1 || sem[0] != a {
		t.Fatalf("lixeira em remoto: %v", f)
	}
	if _, err := os.Stat(a); err != nil {
		t.Fatal("o arquivo tinha de ficar")
	}
	if _, err := os.Stat(raiz + "/.Trash-" + "0"); err == nil {
		t.Error("nenhuma lixeira é criada no remoto")
	}
	// Apagar de vez lá dentro funciona, pasta inclusive.
	feita(t, m, op("apagar", "origens", []string{raiz + "/docs/sub"}))
	if _, err := os.Stat(raiz + "/docs/sub"); !os.IsNotExist(err) {
		t.Error("apagar dentro do remoto")
	}
	// Copiar de lá para cá também.
	aqui := m.Pasta("aqui")
	feita(t, m, op("copiar", "origens", []string{a}, "destino", aqui))
	if b, _ := os.ReadFile(aqui + "/a.txt"); string(b) != "um" {
		t.Error("copiar do remoto")
	}
	// A raiz do remoto e a pasta do gvfs não se apagam, movem nem renomeiam.
	for _, alvo := range []string{raiz, gvfs, m.Casa + "/fora"} {
		for _, pedido := range []string{op("apagar", "origens", []string{alvo}), op("lixeira", "origens", []string{alvo}),
			op("mover", "origens", []string{alvo}, "destino", aqui), op("renomear", "caminho", alvo, "nome", "x")} {
			if r := m.Pede(pedido); r["recusado"] != true {
				t.Errorf("%s devia ser recusado: %v", pedido, r)
			}
		}
	}
	if _, err := os.Stat(raiz + "/docs/a.txt"); err != nil {
		t.Error("as recusas não tocam em nada")
	}

	// A conexão cai no meio (o remoto some): a tarefa diz isso.
	origem := m.Grava("aqui/novo.txt", "x")
	destino := raiz + "/docs"
	os.Remove(raiz + "/docs/a.txt")
	os.Remove(raiz + "/docs")
	os.Remove(raiz)
	for _, pedido := range []string{op("copiar", "origens", []string{origem}, "destino", destino), op("criar-pasta", "destino", destino, "nome", "nova")} {
		r := m.Pede(pedido)
		texto := prova.J(r)
		if r["tarefa"] != nil {
			texto += prova.J(fim(m, r["tarefa"]))
		}
		if !strings.Contains(texto, "a conexão com servidor caiu") {
			t.Errorf("%s: a frase da conexão caída não apareceu: %s", pedido, texto)
		}
	}
	// E a frase das tarefas, quando o erro vem do meio de uma cópia.
	if f := frase(&os.PathError{Op: "open", Path: destino + "/x", Err: syscall.ENOTCONN}); f != "a conexão com servidor caiu" {
		t.Errorf("frase de um erro de conexão: %q", f)
	}
	if f := frase(&os.PathError{Op: "open", Path: aqui + "/x", Err: syscall.ENOTCONN}); strings.Contains(f, "conexão com") {
		t.Errorf("fora de um remoto não é queda de conexão: %q", f)
	}
}
