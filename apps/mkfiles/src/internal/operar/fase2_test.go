package operar

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// outraUnidade faz de conta que tudo dentro de ponto está em outro disco.
func outraUnidade(ponto string) {
	unidadeDe = func(caminho string, _ fs.FileInfo) uint64 {
		if caminho == ponto || strings.HasPrefix(caminho, ponto+"/") {
			return 99
		}
		return 1
	}
}

func temErro(f map[string]any, trecho string) bool {
	for _, e := range f["erros"].([]any) {
		if strings.Contains(e.(map[string]any)["erro"].(string), trecho) {
			return true
		}
	}
	return false
}

func TestApagarNaoAtravessaPontoDeMontagem(t *testing.T) {
	m := motor(t)
	m.Grava("p/a.txt", "a")
	m.Grava("p/sub/b.txt", "b")
	precioso := m.Grava("p/sub/disco/precioso.txt", "fica")
	m.Grava("p/sub/disco/fundo/outro.txt", "fica")
	outraUnidade(m.Casa + "/p/sub/disco")

	f := faz(t, m, op("apagar", "origens", []string{m.Casa + "/p"}))
	if f["estado"] != "erro" || !temErro(f, "há outro disco montado em "+m.Casa+"/p/sub/disco: desmonte antes") {
		t.Fatalf("apagar por cima de um disco montado: %v", f)
	}
	if le(precioso) != "fica" || le(m.Casa+"/p/sub/disco/fundo/outro.txt") != "fica" {
		t.Fatal("o disco montado foi esvaziado")
	}
	// O que é do disco de partida saiu; o caminho até o ponto ficou.
	if existe(m.Casa+"/p/a.txt") || existe(m.Casa+"/p/sub/b.txt") {
		t.Error("o resto da pasta devia ter saído")
	}
	// Só um erro: a pasta que sobrou por causa do disco não vira "não está vazia".
	if n := len(f["erros"].([]any)); n != 1 {
		t.Errorf("esperava 1 erro, vieram %d: %v", n, f["erros"])
	}

	// O próprio ponto de montagem também não é apagado.
	f = faz(t, m, op("apagar", "origens", []string{m.Casa + "/p/sub/disco"}))
	if f["estado"] != "erro" || !temErro(f, "desmonte antes") || le(precioso) != "fica" {
		t.Fatalf("apagar o ponto de montagem: %v", f)
	}
}

func TestApagarOlhaATabelaDeMontagens(t *testing.T) {
	m := motor(t)
	precioso := m.Grava("q/disco/precioso.txt", "fica")
	m.Grava("q/x.txt", "x")
	m.Montagem(m.Casa+"/q/disco", "/dev/sdz1")
	f := faz(t, m, op("apagar", "origens", []string{m.Casa + "/q"}))
	if f["estado"] != "erro" || !temErro(f, "desmonte antes") || le(precioso) != "fica" {
		t.Fatalf("ponto de montagem da tabela: %v", f)
	}
}

func TestLixeiraRecusaPastaComDiscoMontado(t *testing.T) {
	m := motor(t)
	precioso := m.Grava("p/sub/disco/precioso.txt", "fica")
	m.Grava("p/a.txt", "a")
	outraUnidade(m.Casa + "/p/sub/disco")
	f := faz(t, m, op("lixeira", "origens", []string{m.Casa + "/p"}))
	if f["estado"] != "erro" || !temErro(f, "há outro disco montado em "+m.Casa+"/p/sub/disco: desmonte antes") {
		t.Fatalf("lixeira com disco dentro: %v", f)
	}
	if le(precioso) != "fica" || le(m.Casa+"/p/a.txt") != "a" {
		t.Fatal("nada devia ter sido movido")
	}
	if len(lista(f["sem_lixeira"])) != 0 {
		t.Error("não é caso de sem_lixeira: apagar de vez também não pode")
	}

	// Pela tabela de montagens, sem a unidade dizer nada.
	unidadeDe = unidadeDoDisco
	m.Grava("q/disco/precioso.txt", "fica")
	m.Montagem(m.Casa+"/q/disco", "/dev/sdz1")
	f = faz(t, m, op("lixeira", "origens", []string{m.Casa + "/q"}))
	if f["estado"] != "erro" || !temErro(f, "desmonte antes") || le(m.Casa+"/q/disco/precioso.txt") != "fica" {
		t.Fatalf("lixeira com ponto da tabela dentro: %v", f)
	}
}

func TestMoverEntreDiscosNaoLevaDiscoMontado(t *testing.T) {
	m := motor(t)
	precioso := m.Grava("p/disco/precioso.txt", "fica")
	m.Grava("p/a.txt", "a")
	destino := m.Pasta("destino")
	outroDisco()
	outraUnidade(m.Casa + "/p/disco")
	f := faz(t, m, op("mover", "origens", []string{m.Casa + "/p"}, "destino", destino))
	if f["estado"] != "erro" || !temErro(f, "desmonte antes") {
		t.Fatalf("mover entre discos com um disco dentro: %v", f)
	}
	if le(precioso) != "fica" || le(m.Casa+"/p/a.txt") != "a" || existe(destino+"/p") {
		t.Fatal("a origem tem de ficar inteira e nada chega ao destino")
	}
	if sobras, _ := os.ReadDir(destino); len(sobras) != 0 {
		t.Errorf("sobrou temporário no destino: %v", sobras)
	}
}

func lote(pares ...string) string {
	var ps [][]string
	for i := 0; i+1 < len(pares); i += 2 {
		ps = append(ps, []string{pares[i], pares[i+1]})
	}
	return op("renomear-lote", "pares", ps)
}

func TestRenomearLote(t *testing.T) {
	m := motor(t)
	p := m.Pasta("lote")
	for _, n := range []string{"a", "b", "c", "d", "e"} {
		m.Grava("lote/"+n, n)
	}
	// Simples, com um par que não muda nada no meio.
	f := feita(t, m, lote(p+"/a", "um", p+"/e", "e"))
	if le(p+"/um") != "a" || existe(p+"/a") || f["acao"] != "renomear-lote" || f["desfazivel"] != true ||
		strings.Join(lista(f["criados"]), ",") != p+"/um" {
		t.Fatalf("simples: %v", f)
	}
	if h := historico(m); h["desfazer"] != "Renomear «a»" {
		t.Errorf("histórico: %v", h)
	}
	// Cadeia: b→c, c→d, d→x (a ordem dada é a pior).
	feita(t, m, lote(p+"/b", "c", p+"/c", "d", p+"/d", "x"))
	if le(p+"/c") != "b" || le(p+"/d") != "c" || le(p+"/x") != "d" || existe(p+"/b") {
		t.Fatal("cadeia")
	}
	if h := historico(m); h["desfazer"] != "Renomear 3 itens" {
		t.Errorf("histórico: %v", h)
	}
	// Ciclo de 2 e de 3.
	feita(t, m, lote(p+"/c", "d", p+"/d", "c"))
	if le(p+"/c") != "c" || le(p+"/d") != "b" {
		t.Fatal("ciclo de 2")
	}
	historico(m)
	f = feita(t, m, lote(p+"/c", "d", p+"/d", "x", p+"/x", "c"))
	if le(p+"/d") != "c" || le(p+"/x") != "b" || le(p+"/c") != "d" {
		t.Fatal("ciclo de 3")
	}
	if f["itens"] != float64(3) || f["itens_total"] != float64(3) {
		t.Errorf("andamento: %v", f)
	}
	historico(m)
	// Nenhum temporário sobra.
	if es, _ := os.ReadDir(p); len(es) != 5 {
		t.Errorf("sobrou temporário: %v", es)
	}

	// Desfazer e refazer são um passo só, mesmo com ciclo.
	r := m.Pede(op("desfazer"))
	if r["descricao"] != "Renomear 3 itens" {
		t.Fatalf("desfazer: %v", r)
	}
	if f := fim(m, r["tarefa"]); f["estado"] != "feita" {
		t.Fatalf("desfazer: %v", f)
	}
	if le(p+"/c") != "c" || le(p+"/d") != "b" || le(p+"/x") != "d" {
		t.Fatal("desfazer do ciclo de 3")
	}
	r = m.Pede(op("refazer"))
	if f := fim(m, r["tarefa"]); f["estado"] != "feita" {
		t.Fatalf("refazer: %v", f)
	}
	if le(p+"/d") != "c" || le(p+"/x") != "b" || le(p+"/c") != "d" {
		t.Fatal("refazer do ciclo de 3")
	}
}

func TestRenomearLoteFalhaNoMeioDesfaz(t *testing.T) {
	m := motor(t)
	p := m.Pasta("lote")
	for _, n := range []string{"a", "b", "c"} {
		m.Grava("lote/"+n, n)
	}
	feitas := 0
	renomeia = func(de, para string) error {
		// A volta do temporário do ciclo falha; as de desfazer passam.
		if filepath.Base(para) == "b" && strings.HasPrefix(filepath.Base(de), ".mkfiles-") {
			return &os.LinkError{Op: "rename", Old: de, New: para, Err: errors.New("disco com defeito")}
		}
		feitas++
		return os.Rename(de, para)
	}
	// Um ciclo (a↔b) e mais um: os temporários também têm de voltar.
	f := faz(t, m, lote(p+"/a", "b", p+"/b", "a", p+"/c", "c2"))
	if f["estado"] != "erro" || f["desfazivel"] != false || !temErro(f, "disco com defeito") {
		t.Fatalf("falha no meio: %v", f)
	}
	if feitas == 0 {
		t.Fatal("o teste não chegou a renomear nada antes da falha")
	}
	if le(p+"/a") != "a" || le(p+"/b") != "b" || le(p+"/c") != "c" {
		t.Fatalf("tudo tinha de voltar: a=%s b=%s c=%s", le(p+"/a"), le(p+"/b"), le(p+"/c"))
	}
	if es, _ := os.ReadDir(p); len(es) != 3 {
		t.Errorf("sobrou temporário: %v", es)
	}
	if r := m.Pede(op("desfazer")); r["erro"] != "nada para desfazer" {
		t.Errorf("um lote que falhou não entra no histórico: %v", r)
	}
}

func TestRenomearLoteConflitosERecusas(t *testing.T) {
	m := motor(t)
	p := m.Pasta("lote")
	for _, n := range []string{"a", "b", "c", "fica"} {
		m.Grava("lote/"+n, n)
	}
	r := m.Pede(lote(p+"/a", "x", p+"/b", "x", p+"/c", "fica", p+"/fica", "fica"))
	if r["erro"] == nil || r["tarefa"] != nil {
		t.Fatalf("conflitos: %v", r)
	}
	motivos := map[string]string{}
	for _, c := range r["conflitos"].([]any) {
		c := c.(map[string]any)
		motivos[c["de"].(string)] = c["motivo"].(string) + ":" + c["para"].(string)
	}
	if len(motivos) != 3 || motivos[p+"/a"] != "repetido:x" || motivos[p+"/b"] != "repetido:x" || motivos[p+"/c"] != "existe:fica" {
		t.Errorf("os conflitos: %v", motivos)
	}
	r = m.Pede(lote(p+"/a", "a/b", p+"/b", ".."))
	if cs, _ := r["conflitos"].([]any); len(cs) != 2 || cs[0].(map[string]any)["motivo"] != "invalido" {
		t.Errorf("nome inválido: %v", r)
	}
	for _, ruim := range []string{
		op("renomear-lote", "pares", [][]string{}),
		op("renomear-lote"),
		op("renomear-lote", "pares", [][]string{{p + "/a"}}),
		lote("relativo", "x"),
		lote(p+"/../lote/a", "x"),
		lote(p+"/nao-ha", "x"),
		lote(p+"/a", "x", p+"/a", "y"),
		lote(p+"/a", "a"),
	} {
		if r := m.Pede(ruim); r["erro"] == nil || r["recusado"] != nil {
			t.Errorf("%s devia ser erro comum: %v", ruim, r)
		}
	}
	for _, protegido := range []string{"/", m.Casa, filepath.Dir(m.Casa)} {
		if r := m.Pede(lote(p+"/a", "x", protegido, "outro")); r["recusado"] != true {
			t.Errorf("renomear %s em lote devia ser recusado: %v", protegido, r)
		}
	}
	for _, n := range []string{"a", "b", "c", "fica"} {
		if le(p+"/"+n) != n {
			t.Fatalf("nada podia ter sido tocado: %s", n)
		}
	}
	if es, _ := os.ReadDir(p); len(es) != 4 {
		t.Errorf("a pasta mudou: %v", es)
	}
}

func TestRenomearLoteOcupadoNaHora(t *testing.T) {
	m := motor(t)
	p := m.Pasta("lote")
	m.Grava("lote/a", "a")
	m.Grava("lote/b", "b")
	// Alguém cria o destino entre o pedido e a vez da tarefa: nada é sobrescrito.
	primeira := true
	renomeia = func(de, para string) error {
		if primeira {
			primeira = false
			os.WriteFile(p+"/b2", []byte("intruso"), 0o644)
		}
		return os.Rename(de, para)
	}
	f := faz(t, m, lote(p+"/a", "a2", p+"/b", "b2"))
	if f["estado"] != "erro" || le(p+"/b2") != "intruso" || le(p+"/a") != "a" || le(p+"/b") != "b" || existe(p+"/a2") {
		t.Fatalf("destino ocupado na hora: %v", f)
	}
}

// O último evento "historico" é sempre o estado de agora, mesmo quando a
// tarefa do desfazer acaba antes de o pedido terminar de avisar (era uma
// corrida: o aviso velho, sem "refazer", podia sair depois do novo).
func TestHistoricoNuncaSaiForaDeOrdem(t *testing.T) {
	m := motor(t)
	d := m.Pasta("h")
	for i := 0; i < 150; i++ {
		feita(t, m, op("criar-pasta", "destino", d, "nome", "p"))
		historico(m)
		r := m.Pede(`{"op":"desfazer"}`)
		fim(m, r["tarefa"])
		// Dois avisos: o do pedido e o do fim da tarefa. O segundo a sair
		// tem de ser o que diz que dá para refazer.
		historico(m)
		if h := historico(m); h["refazer"] != "Criar a pasta «p»" || h["desfazer"] != nil {
			t.Fatalf("rodada %d: o último aviso é velho: %v", i, h)
		}
	}
}
