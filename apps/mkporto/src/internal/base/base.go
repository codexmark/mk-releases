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
