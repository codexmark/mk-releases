// Package base tem o que todo subcomando do coletor usa: as raízes do sistema
// (trocáveis por variável de ambiente, para os testes), a saída em JSON e os
// códigos de saída do contrato (../../../CONTRATO.md).
package base

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Códigos de saída do contrato.
const (
	Certo    = 0
	Erro     = 1
	Recusado = 2
)

// Saida é para onde o JSON vai. Os testes trocam por um buffer.
var Saida io.Writer = os.Stdout

func raiz(variavel, padrao string) string {
	if v := os.Getenv(variavel); v != "" {
		return v
	}
	return padrao
}

// Proc, Sys, Etc e Run montam um caminho dentro de /proc, /sys, /etc e /run
// (ou da raiz que MKPULSO_PROC, MKPULSO_SYS, MKPULSO_ETC e MKPULSO_RUN derem).
func Proc(partes ...string) string { return junta(raiz("MKPULSO_PROC", "/proc"), partes) }
func Sys(partes ...string) string  { return junta(raiz("MKPULSO_SYS", "/sys"), partes) }
func Etc(partes ...string) string  { return junta(raiz("MKPULSO_ETC", "/etc"), partes) }
func Run(partes ...string) string  { return junta(raiz("MKPULSO_RUN", "/run"), partes) }

// Home monta um caminho dentro da home de quem roda (ou de MKPULSO_HOME).
func Home(partes ...string) string {
	h := os.Getenv("MKPULSO_HOME")
	if h == "" {
		h, _ = os.UserHomeDir()
	}
	return junta(h, partes)
}

func junta(raiz string, partes []string) string {
	return filepath.Join(append([]string{raiz}, partes...)...)
}

// Escreve solta um valor como uma linha de JSON, sem escapar acentos nem <>.
func Escreve(v any) error {
	enc := json.NewEncoder(Saida)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// Falha escreve {"erro": "..."} e devolve o código dado: o subcomando faz
// `return base.Falha(base.Recusado, "não mexo no processo 1")`.
func Falha(codigo int, formato string, args ...any) int {
	_ = Escreve(map[string]string{"erro": fmt.Sprintf(formato, args...)})
	return codigo
}
