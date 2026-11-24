// Package dubles monta comandos falsos (ffmpeg, ffprobe, pkexec, pacman,
// bsdtar, zstd…) para os testes do motor: cada um é um script num diretório
// que vira o PATH inteiro do teste, faz o que o teste mandou e anota como foi
// chamado. Com o PATH só nesse diretório, o comando de verdade nunca roda.
package dubles

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Mesa é o diretório dos dublês de um teste.
type Mesa struct {
	t   testing.TB
	Dir string
}

// Nova cria a mesa vazia e faz dela o PATH do teste: enquanto nenhum dublê
// for posto, nenhum comando externo existe.
func Nova(t testing.TB) *Mesa {
	t.Helper()
	m := &Mesa{t: t, Dir: t.TempDir()}
	t.Setenv("PATH", m.Dir)
	return m
}

// Comando põe (ou troca) o dublê de um comando. O corpo é shell (/bin/sh) e
// só pode usar o que o shell tem dentro (printf, read, case, test) ou um
// caminho absoluto: o PATH do teste não tem mais nada. Cada chamada fica
// anotada com os argumentos separados de verdade (veja Chamadas).
func (m *Mesa) Comando(nome, corpo string) {
	m.t.Helper()
	var s strings.Builder
	s.WriteString("#!/bin/sh\n")
	s.WriteString("{ for a in \"$@\"; do printf '%s\\037' \"$a\"; done; printf '\\n'; } >>" + Aspas(m.arquivoDeChamadas(nome)) + "\n")
	s.WriteString(corpo)
	s.WriteString("\n")
	if err := os.WriteFile(filepath.Join(m.Dir, nome), []byte(s.String()), 0o755); err != nil {
		m.t.Fatal(err)
	}
}

// Responde põe um dublê que escreve o texto no stdout e sai com o código.
func (m *Mesa) Responde(nome, texto string, codigo int) {
	m.t.Helper()
	m.Comando(nome, "printf '%s' "+Aspas(texto)+"\nexit "+itoa(codigo))
}

// Chamadas devolve os argumentos de cada chamada ao dublê, na ordem.
func (m *Mesa) Chamadas(nome string) [][]string {
	m.t.Helper()
	b, err := os.ReadFile(m.arquivoDeChamadas(nome))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		m.t.Fatal(err)
	}
	var out [][]string
	for _, linha := range strings.Split(strings.TrimRight(string(b), "\n"), "\n") {
		linha = strings.TrimSuffix(linha, "\037")
		if linha == "" {
			out = append(out, []string{})
			continue
		}
		out = append(out, strings.Split(linha, "\037"))
	}
	return out
}

// Esquece apaga as chamadas anotadas de um dublê.
func (m *Mesa) Esquece(nome string) {
	_ = os.Remove(m.arquivoDeChamadas(nome))
}

func (m *Mesa) arquivoDeChamadas(nome string) string {
	return filepath.Join(m.Dir, "."+nome+".chamadas")
}

// Aspas põe um texto entre aspas simples do shell.
func Aspas(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var d []byte
	for ; n > 0; n /= 10 {
		d = append([]byte{byte('0' + n%10)}, d...)
	}
	return string(d)
}
