// Package prova é o arnês dos testes do motor: uma sessão com a pasta pessoal
// e os XDG numa pasta temporária, e um jeito de mandar pedidos e ler as
// respostas e os eventos. Só os testes usam.
package prova

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"mkfiles/internal/base"
)

// Linha é uma linha que o motor escreveu.
type Linha = map[string]any

// Motor é uma sessão de teste.
type Motor struct {
	S    *base.Sessao
	Casa string // a pasta pessoal de mentira
	t    testing.TB

	trava  sync.Mutex
	chegou *sync.Cond
	resto  []byte
	linhas []Linha
	id     int
}

// Prazo é quanto um teste espera por uma linha antes de desistir.
var Prazo = 10 * time.Second

// Novo monta a caixa de areia (HOME, XDG_* e a tabela de montagens só com "/")
// e liga as áreas pedidas.
func Novo(t testing.TB, ligar ...func(*base.Sessao)) *Motor {
	t.Helper()
	// EvalSymlinks: num sistema em que a pasta temporária é um link, os
	// caminhos "limpos" têm de ser os de verdade.
	casa, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", casa)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(casa, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(casa, ".local", "share"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(casa, ".cache"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(casa, ".local", "state"))
	t.Setenv("XDG_DATA_DIRS", filepath.Join(casa, ".sistema", "share"))
	t.Setenv("XDG_CONFIG_DIRS", filepath.Join(casa, ".sistema", "xdg"))
	montagens := filepath.Join(casa, ".mountinfo")
	if err := os.WriteFile(montagens, []byte("1 0 8:1 / / rw - ext4 /dev/raiz rw\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MKFILES_MOUNTINFO", montagens)
	m := &Motor{Casa: casa, t: t}
	m.chegou = sync.NewCond(&m.trava)
	m.S = base.NovaSessao(m)
	for _, l := range ligar {
		l(m.S)
	}
	t.Cleanup(m.S.Fecha)
	return m
}

// Montagem acrescenta um ponto de montagem de mentira à tabela.
func (m *Motor) Montagem(ponto, origem string) {
	m.t.Helper()
	f, err := os.OpenFile(os.Getenv("MKFILES_MOUNTINFO"), os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		m.t.Fatal(err)
	}
	defer f.Close()
	f.WriteString("2 1 8:2 / " + ponto + " rw - ext4 " + origem + " rw\n")
}

// Write recebe o que a sessão escreve e separa as linhas.
func (m *Motor) Write(b []byte) (int, error) {
	m.trava.Lock()
	defer m.trava.Unlock()
	m.resto = append(m.resto, b...)
	for {
		i := bytes.IndexByte(m.resto, '\n')
		if i < 0 {
			break
		}
		var l Linha
		if err := json.Unmarshal(m.resto[:i], &l); err == nil {
			m.linhas = append(m.linhas, l)
		}
		m.resto = m.resto[i+1:]
	}
	m.chegou.Broadcast()
	return len(b), nil
}

// tira espera a primeira linha que serve e a tira da fila.
func (m *Motor) tira(oQue string, serve func(Linha) bool) Linha {
	m.t.Helper()
	fim := time.Now().Add(Prazo)
	alarme := time.AfterFunc(Prazo, func() {
		m.trava.Lock()
		m.chegou.Broadcast()
		m.trava.Unlock()
	})
	defer alarme.Stop()
	m.trava.Lock()
	defer m.trava.Unlock()
	for {
		for i, l := range m.linhas {
			if serve(l) {
				m.linhas = append(m.linhas[:i:i], m.linhas[i+1:]...)
				return l
			}
		}
		if !time.Now().Before(fim) {
			m.t.Fatalf("o motor não respondeu a tempo: %s (na fila: %v)", oQue, m.linhas)
			return nil
		}
		m.chegou.Wait()
	}
}

// Manda envia um pedido (um objeto JSON sem "id") e devolve o id que ele ganhou.
func (m *Motor) Manda(pedido string) int {
	m.t.Helper()
	var campos map[string]any
	if err := json.Unmarshal([]byte(pedido), &campos); err != nil {
		m.t.Fatalf("pedido de teste que não é JSON: %s", pedido)
	}
	m.trava.Lock()
	m.id++
	id := m.id
	m.trava.Unlock()
	campos["id"] = id
	linha, _ := json.Marshal(campos)
	m.S.Atender(linha)
	return id
}

func doID(id int) func(Linha) bool {
	return func(l Linha) bool {
		n, ok := l["id"].(float64)
		return ok && int(n) == id
	}
}

// Pede manda um pedido e devolve a (primeira) resposta dele.
func (m *Motor) Pede(pedido string) Linha {
	m.t.Helper()
	return m.Resposta(m.Manda(pedido))
}

// Resposta espera a próxima linha de resposta de um pedido já mandado.
func (m *Motor) Resposta(id int) Linha {
	m.t.Helper()
	return m.tira("resposta", doID(id))
}

// PedeFluxo manda um pedido e junta as linhas dele até a de "fim": true.
func (m *Motor) PedeFluxo(pedido string) []Linha {
	m.t.Helper()
	return m.Fluxo(m.Manda(pedido))
}

// Fluxo junta as linhas de um pedido já mandado até a de "fim": true (ou a de
// erro).
func (m *Motor) Fluxo(id int) []Linha {
	m.t.Helper()
	var todas []Linha
	for {
		l := m.Resposta(id)
		todas = append(todas, l)
		if l["fim"] == true || l["erro"] != nil {
			return todas
		}
	}
}

// Evento espera o próximo evento com esse nome que serve (serve nil: qualquer).
func (m *Motor) Evento(nome string, serve func(Linha) bool) Linha {
	m.t.Helper()
	return m.tira("evento "+nome, func(l Linha) bool {
		return l["evento"] == nome && (serve == nil || serve(l))
	})
}

// SemEvento confere que nenhum evento com esse nome chega durante a espera.
func (m *Motor) SemEvento(nome string, espera time.Duration) {
	m.t.Helper()
	time.Sleep(espera)
	m.trava.Lock()
	defer m.trava.Unlock()
	for _, l := range m.linhas {
		if l["evento"] == nome {
			m.t.Fatalf("não devia ter saído o evento %s: %v", nome, l)
		}
	}
}

// Grava escreve um arquivo (criando as pastas) e devolve o caminho.
func (m *Motor) Grava(caminho, dados string) string {
	m.t.Helper()
	if !filepath.IsAbs(caminho) {
		caminho = filepath.Join(m.Casa, caminho)
	}
	if err := os.MkdirAll(filepath.Dir(caminho), 0o755); err != nil {
		m.t.Fatal(err)
	}
	if err := os.WriteFile(caminho, []byte(dados), 0o644); err != nil {
		m.t.Fatal(err)
	}
	return caminho
}

// Pasta cria uma pasta dentro da casa de mentira e devolve o caminho.
func (m *Motor) Pasta(caminho string) string {
	m.t.Helper()
	if !filepath.IsAbs(caminho) {
		caminho = filepath.Join(m.Casa, caminho)
	}
	if err := os.MkdirAll(caminho, 0o755); err != nil {
		m.t.Fatal(err)
	}
	return caminho
}

// J escreve um valor como JSON, para montar pedidos com caminhos.
func J(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
