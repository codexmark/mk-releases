// Package base tem o que todo subcomando do motor usa: a pasta do ssh e os
// comandos (trocáveis por variável de ambiente, para os testes), a saída em
// JSON e os códigos de saída do contrato (../../../CONTRATO.md).
package base

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Códigos de saída do contrato.
const (
	Certo    = 0
	Erro     = 1
	Recusado = 2
)

// Saida é para onde o JSON vai. Os testes trocam por um buffer.
var Saida io.Writer = os.Stdout

var travaDaSaida sync.Mutex

// Home é a home de quem roda.
func Home() string {
	h, _ := os.UserHomeDir()
	return h
}

// PastaDoSsh é ~/.ssh (ou MKPORTO_SSH_DIR).
func PastaDoSsh() string {
	if v := os.Getenv("MKPORTO_SSH_DIR"); v != "" {
		return filepath.Clean(v)
	}
	return filepath.Join(Home(), ".ssh")
}

// Comando é o nome de um comando externo: o da variável, ou o padrão.
func Comando(variavel, padrao string) string {
	if v := os.Getenv(variavel); v != "" {
		return v
	}
	return padrao
}

// Config monta um caminho dentro de ~/.config (ou de XDG_CONFIG_HOME).
func Config(partes ...string) string {
	d := os.Getenv("XDG_CONFIG_HOME")
	if d == "" {
		d = filepath.Join(Home(), ".config")
	}
	return filepath.Join(append([]string{d}, partes...)...)
}

// Til escreve um caminho do jeito que a pessoa lê: a home vira "~".
func Til(caminho string) string {
	h := Home()
	if h == "" || caminho == "" {
		return caminho
	}
	if caminho == h {
		return "~"
	}
	if strings.HasPrefix(caminho, h+string(filepath.Separator)) {
		return "~" + caminho[len(h):]
	}
	return caminho
}

// Escreve solta um valor como uma linha de JSON, sem escapar acentos nem <>.
// Pode ser chamada de mais de uma goroutine: cada linha sai inteira.
func Escreve(v any) error {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return err
	}
	travaDaSaida.Lock()
	defer travaDaSaida.Unlock()
	_, err := Saida.Write(b.Bytes())
	return err
}

// Falha escreve {"erro": "..."} e devolve o código dado: o subcomando faz
// `return base.Falha(base.Recusado, "esse host é do seu ~/.ssh/config")`.
func Falha(codigo int, formato string, args ...any) int {
	_ = Escreve(map[string]string{"erro": fmt.Sprintf(formato, args...)})
	return codigo
}

// FalhaDeCampo é o erro de validação: diz também qual campo está errado.
func FalhaDeCampo(campo, frase string) int {
	_ = Escreve(map[string]string{"erro": frase, "campo": campo})
	return Erro
}

// Maquinas é a pasta das máquinas virtuais (~/.local/share/mk/maquinas, ou
// MKPORTO_MAQUINAS).
func Maquinas() string {
	if v := os.Getenv("MKPORTO_MAQUINAS"); v != "" {
		return absoluto(v)
	}
	return filepath.Join(Home(), ".local", "share", "mk", "maquinas")
}

// Execucao é a pasta em que cada máquina ligada tem o socket do QMP e o pid
// ($XDG_RUNTIME_DIR/mkporto/maquinas, ou MKPORTO_EXECUCAO). "" se não há onde.
func Execucao() string {
	if v := os.Getenv("MKPORTO_EXECUCAO"); v != "" {
		return absoluto(v)
	}
	if d := os.Getenv("XDG_RUNTIME_DIR"); d != "" {
		return filepath.Join(absoluto(d), "mkporto", "maquinas")
	}
	return ""
}

func absoluto(caminho string) string {
	if a, err := filepath.Abs(caminho); err == nil {
		return a
	}
	return filepath.Clean(caminho)
}
