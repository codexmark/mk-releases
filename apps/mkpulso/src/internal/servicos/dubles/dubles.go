// Package dubles monta comandos falsos (systemctl, journalctl) para os testes
// do coletor: cada um é um script num diretório que vira o PATH inteiro do
// teste, responde o que o teste mandou e anota como foi chamado. Com o PATH
// só nesse diretório, o comando de verdade nunca roda.
package dubles

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Regra diz o que o dublê responde a uma chamada. Vale a primeira regra cujo
// Contem aparece nos argumentos (separados por espaço); Contem vazio casa
// com tudo. A saída vem de Arquivo (um caminho, em geral em testdata) ou de
// Texto; Erro vai para o stderr e Codigo é o código de saída.
type Regra struct {
	Contem  string
	Arquivo string
	Texto   string
	Erro    string
	Codigo  int
}

// Mesa é o diretório dos dublês de um teste.
type Mesa struct {
	t   testing.TB
	dir string
}

// Nova cria a mesa vazia e faz dela o PATH do teste: enquanto nenhum dublê
// for posto, nenhum comando externo existe.
func Nova(t testing.TB) *Mesa {
	t.Helper()
	m := &Mesa{t: t, dir: t.TempDir()}
	t.Setenv("PATH", m.dir)
	return m
}

// Comando põe (ou troca) o dublê de um comando. Uma chamada que não casa com
// regra nenhuma reclama no stderr e sai com 97.
func (m *Mesa) Comando(nome string, regras ...Regra) {
	m.t.Helper()
	var s strings.Builder
	s.WriteString("#!/bin/sh\n")
	fmt.Fprintf(&s, "printf '%%s\\n' \"$*\" >>%s\n", aspas(m.arquivoDeChamadas(nome)))
	s.WriteString("solta() { while IFS= read -r l || [ -n \"$l\" ]; do printf '%s\\n' \"$l\"; done <\"$1\"; }\n")
	s.WriteString("case \" $* \" in\n")
	for i, r := range regras {
		fmt.Fprintf(&s, "*%s*)\n", aspas(r.Contem))
		arquivo := r.Arquivo
		if arquivo != "" {
			abs, err := filepath.Abs(arquivo)
			if err != nil {
				m.t.Fatal(err)
			}
			if _, err := os.Stat(abs); err != nil {
				m.t.Fatalf("dublê %s: %v", nome, err)
			}
			arquivo = abs
		} else if r.Texto != "" {
			arquivo = m.guarda(fmt.Sprintf("%s.saida.%d", nome, i), r.Texto)
		}
		if arquivo != "" {
			fmt.Fprintf(&s, "  solta %s\n", aspas(arquivo))
		}
		if r.Erro != "" {
			fmt.Fprintf(&s, "  solta %s >&2\n", aspas(m.guarda(fmt.Sprintf("%s.erro.%d", nome, i), r.Erro)))
		}
		fmt.Fprintf(&s, "  exit %d ;;\n", r.Codigo)
	}
	fmt.Fprintf(&s, "*)\n  printf '%%s\\n' \"dublê de %s: chamada inesperada: $*\" >&2\n  exit 97 ;;\nesac\n", nome)
	if err := os.WriteFile(filepath.Join(m.dir, nome), []byte(s.String()), 0o755); err != nil {
		m.t.Fatal(err)
	}
}

// Chamadas devolve os argumentos de cada chamada ao dublê, na ordem, cada
// uma como uma linha com os argumentos separados por espaço.
func (m *Mesa) Chamadas(nome string) []string {
	m.t.Helper()
	b, err := os.ReadFile(m.arquivoDeChamadas(nome))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		m.t.Fatal(err)
	}
	return strings.Split(strings.TrimRight(string(b), "\n"), "\n")
}

func (m *Mesa) arquivoDeChamadas(nome string) string {
	return filepath.Join(m.dir, "."+nome+".chamadas")
}

func (m *Mesa) guarda(nome, texto string) string {
	caminho := filepath.Join(m.dir, "."+nome)
	if err := os.WriteFile(caminho, []byte(texto), 0o644); err != nil {
		m.t.Fatal(err)
	}
	return caminho
}

// aspas põe um texto entre aspas simples do shell.
func aspas(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
