package vigia

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"mkfiles/internal/prova"
)

func motor(t *testing.T) *prova.Motor {
	t.Helper()
	t.Setenv("MKFILES_MIME", t.TempDir())
	return prova.Novo(t, Ligar)
}

func nomes(e prova.Linha, campo string) string {
	var l []string
	for _, x := range e[campo].([]any) {
		if it, ok := x.(map[string]any); ok {
			l = append(l, it["nome"].(string))
		} else {
			l = append(l, x.(string))
		}
	}
	return strings.Join(l, " ")
}

func daPasta(caminho string) func(prova.Linha) bool {
	return func(l prova.Linha) bool { return l["caminho"] == caminho }
}

func TestCriarRenomearApagar(t *testing.T) {
	m := motor(t)
	pasta := m.Pasta("p")
	m.Grava("p/velho.txt", "x")
	if r := m.Pede(`{"op":"vigiar","caminho":` + prova.J(pasta) + `}`); r["vigiando"] != true {
		t.Fatalf("vigiar: %v", r)
	}
	// Várias mudanças rápidas saem num evento só.
	antes := time.Now()
	for i := 0; i < 20; i++ {
		m.Grava(fmt.Sprintf("p/novo-%02d.txt", i), "conteúdo")
	}
	os.Mkdir(pasta+"/sub", 0o755)
	e := m.Evento("pasta", daPasta(pasta))
	if time.Since(antes) < Intervalo {
		t.Errorf("o evento saiu antes do intervalo: %v", time.Since(antes))
	}
	mudou := e["mudou"].([]any)
	if len(mudou) != 21 || e["sumiu"] != false || e["recarregar"] != false || len(e["saiu"].([]any)) != 0 {
		t.Fatalf("criar: %d itens num evento, %v", len(mudou), e)
	}
	if it := mudou[0].(map[string]any); it["nome"] != "novo-00.txt" || it["tamanho"] != float64(9) || it["caminho"] != pasta+"/novo-00.txt" {
		t.Errorf("o item criado: %v", it)
	}
	if it := mudou[20].(map[string]any); it["nome"] != "sub" || it["tipo"] != "pasta" {
		t.Errorf("a pasta criada: %v", it)
	}
	m.SemEvento("pasta", Intervalo+50*time.Millisecond)

	// Renomear: o nome velho sai, o novo chega, no mesmo evento.
	os.Rename(pasta+"/velho.txt", pasta+"/renomeado.txt")
	e = m.Evento("pasta", daPasta(pasta))
	if nomes(e, "saiu") != "velho.txt" || nomes(e, "mudou") != "renomeado.txt" {
		t.Errorf("renomear: saiu %q, mudou %q", nomes(e, "saiu"), nomes(e, "mudou"))
	}
	// Modificar e mudar a permissão.
	os.WriteFile(pasta+"/renomeado.txt", []byte("maior agora"), 0o644)
	e = m.Evento("pasta", daPasta(pasta))
	if it := e["mudou"].([]any)[0].(map[string]any); it["tamanho"] != float64(11) {
		t.Errorf("modificar: %v", it)
	}
	os.Chmod(pasta+"/renomeado.txt", 0o600)
	if e = m.Evento("pasta", daPasta(pasta)); e["mudou"].([]any)[0].(map[string]any)["permissoes"] != "rw-------" {
		t.Errorf("permissão: %v", e)
	}
	// Apagar; e o que foi criado e apagado no mesmo lote só "saiu".
	os.Remove(pasta + "/renomeado.txt")
	m.Grava("p/relampago", "x")
	os.Remove(pasta + "/relampago")
	e = m.Evento("pasta", daPasta(pasta))
	if nomes(e, "saiu") != "relampago renomeado.txt" || len(e["mudou"].([]any)) != 0 {
		t.Errorf("apagar: saiu %q, mudou %q", nomes(e, "saiu"), nomes(e, "mudou"))
	}
	// Um arquivo que entra vindo de outra pasta, e um que sai para outra.
	fora := m.Grava("fora.txt", "x")
	os.Rename(fora, pasta+"/chegou.txt")
	if e = m.Evento("pasta", daPasta(pasta)); nomes(e, "mudou") != "chegou.txt" {
		t.Errorf("mover para dentro: %v", e)
	}
	os.Rename(pasta+"/chegou.txt", fora)
	if e = m.Evento("pasta", daPasta(pasta)); nomes(e, "saiu") != "chegou.txt" {
		t.Errorf("mover para fora: %v", e)
	}
	// O .hidden muda quem é oculto: a janela lista de novo.
	m.Grava("p/.hidden", "sub\n")
	if e = m.Evento("pasta", daPasta(pasta)); e["recarregar"] != true || len(e["mudou"].([]any)) != 0 {
		t.Errorf(".hidden: %v", e)
	}
}

func TestLoteGrandeMandaRecarregar(t *testing.T) {
	m := motor(t)
	LimiteDoLote = 50
	t.Cleanup(func() { LimiteDoLote = 2000 })
	pasta := m.Pasta("p")
	m.Pede(`{"op":"vigiar","caminho":` + prova.J(pasta) + `}`)
	for i := 0; i < 60; i++ {
		m.Grava(fmt.Sprintf("p/%03d", i), "")
	}
	e := m.Evento("pasta", daPasta(pasta))
	if e["recarregar"] != true || len(e["mudou"].([]any)) != 0 || len(e["saiu"].([]any)) != 0 {
		t.Errorf("lote grande: recarregar %v, %d itens", e["recarregar"], len(e["mudou"].([]any)))
	}
}

func TestPastaQueSome(t *testing.T) {
	m := motor(t)
	pasta := m.Pasta("vai-sumir")
	outra := m.Pasta("vai-mudar")
	m.Pede(`{"op":"vigiar","caminho":` + prova.J(pasta) + `}`)
	m.Pede(`{"op":"vigiar","caminho":` + prova.J(outra) + `}`)
	os.Remove(pasta)
	if e := m.Evento("pasta", daPasta(pasta)); e["sumiu"] != true {
		t.Errorf("apagada: %v", e)
	}
	os.Rename(outra, outra+"-2")
	if e := m.Evento("pasta", daPasta(outra)); e["sumiu"] != true {
		t.Errorf("movida: %v", e)
	}
	// Depois de sumir, desvigiar não é erro, e vigiar de novo (a pasta voltou) funciona.
	if r := m.Pede(`{"op":"desvigiar","caminho":` + prova.J(pasta) + `}`); r["vigiando"] != false {
		t.Errorf("desvigiar a que sumiu: %v", r)
	}
	m.Pasta("vai-sumir")
	m.Pede(`{"op":"vigiar","caminho":` + prova.J(pasta) + `}`)
	m.Grava("vai-sumir/a", "x")
	if e := m.Evento("pasta", daPasta(pasta)); nomes(e, "mudou") != "a" {
		t.Errorf("vigiar de novo: %v", e)
	}
}

func TestDesvigiarERefcount(t *testing.T) {
	m := motor(t)
	pasta := m.Pasta("p")
	// Duas abas na mesma pasta: dois vigiar.
	m.Pede(`{"op":"vigiar","caminho":` + prova.J(pasta) + `}`)
	m.Pede(`{"op":"vigiar","caminho":` + prova.J(pasta) + `}`)
	if r := m.Pede(`{"op":"desvigiar","caminho":` + prova.J(pasta) + `}`); r["vigiando"] != true {
		t.Errorf("uma aba fechou, a outra ainda vê: %v", r)
	}
	m.Grava("p/a", "x")
	if e := m.Evento("pasta", daPasta(pasta)); nomes(e, "mudou") != "a" {
		t.Errorf("ainda vigiada: %v", e)
	}
	// Um evento só, mesmo com duas referências.
	m.SemEvento("pasta", Intervalo+50*time.Millisecond)
	if r := m.Pede(`{"op":"desvigiar","caminho":` + prova.J(pasta) + `}`); r["vigiando"] != false {
		t.Errorf("a última aba fechou: %v", r)
	}
	m.Grava("p/b", "x")
	m.SemEvento("pasta", Intervalo+100*time.Millisecond)
	// Desvigiar o que não se vigia não é erro.
	if r := m.Pede(`{"op":"desvigiar","caminho":` + prova.J(pasta) + `}`); r["vigiando"] != false || r["erro"] != nil {
		t.Errorf("desvigiar de novo: %v", r)
	}
}

func TestOMesmoLugarPorDoisCaminhos(t *testing.T) {
	m := motor(t)
	pasta := m.Pasta("real")
	link := m.Casa + "/atalho"
	os.Symlink(pasta, link)
	m.Pede(`{"op":"vigiar","caminho":` + prova.J(pasta) + `}`)
	m.Pede(`{"op":"vigiar","caminho":` + prova.J(link) + `}`)
	m.Grava("real/a", "x")
	e1 := m.Evento("pasta", daPasta(pasta))
	e2 := m.Evento("pasta", daPasta(link))
	if nomes(e1, "mudou") != "a" || e2["mudou"].([]any)[0].(map[string]any)["caminho"] != link+"/a" {
		t.Errorf("os dois caminhos avisam, cada um com o seu: %v / %v", e1, e2)
	}
	// Fechar um não cala o outro.
	m.Pede(`{"op":"desvigiar","caminho":` + prova.J(link) + `}`)
	m.Grava("real/b", "x")
	if e := m.Evento("pasta", daPasta(pasta)); nomes(e, "mudou") != "b" {
		t.Errorf("depois de fechar o outro caminho: %v", e)
	}
}

func TestErros(t *testing.T) {
	m := motor(t)
	arquivo := m.Grava("a.txt", "x")
	for _, ruim := range []string{"relativa", "/a/../b", m.Casa + "/nao-ha", arquivo, ""} {
		for _, op := range []string{"vigiar", "desvigiar"} {
			r := m.Pede(`{"op":"` + op + `","caminho":` + prova.J(ruim) + `}`)
			if r["erro"] == nil && !(op == "desvigiar" && filepath.IsAbs(ruim) && filepath.Clean(ruim) == ruim) {
				t.Errorf("%s %q devia falhar: %v", op, ruim, r)
			}
		}
	}
}

func TestFecharParaALeitura(t *testing.T) {
	m := motor(t)
	pasta := m.Pasta("p")
	m.Pede(`{"op":"vigiar","caminho":` + prova.J(pasta) + `}`)
	m.S.Fecha()
	m.Grava("p/a", "x")
	m.SemEvento("pasta", Intervalo+100*time.Millisecond)
	if r := m.Pede(`{"op":"vigiar","caminho":` + prova.J(pasta) + `}`); r["erro"] == nil {
		t.Errorf("vigiar depois de fechar: %v", r)
	}
}

func TestFilaEstourada(t *testing.T) {
	m := motor(t)
	a, b := m.Pasta("a"), m.Pasta("b")
	v := &vigia{s: m.S, pastas: map[string]*pasta{}, porWd: map[int32][]*pasta{}}
	for i, c := range []string{a, b} {
		ps := &pasta{caminho: c, wd: int32(i + 1), refs: 1, mudou: map[string]bool{}, saiu: map[string]bool{}}
		v.pastas[c] = ps
		v.porWd[ps.wd] = []*pasta{ps}
	}
	v.trava.Lock()
	v.anota(-1, syscall.IN_Q_OVERFLOW, "")
	v.trava.Unlock()
	for _, c := range []string{a, b} {
		if e := m.Evento("pasta", daPasta(c)); e["recarregar"] != true {
			t.Errorf("fila estourada em %s: %v", c, e)
		}
	}
}
