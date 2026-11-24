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
	"strconv"
	"strings"
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

// PacmanDB monta um caminho dentro de /var/lib/pacman (ou de
// MKPACOTES_PACMAN_DB): local/, sync/ e a trava db.lck.
func PacmanDB(partes ...string) string {
	return junta(raiz("MKPACOTES_PACMAN_DB", "/var/lib/pacman"), partes)
}

// Mk monta um caminho dentro de /usr/share/mk (ou de MKPACOTES_MK).
func Mk(partes ...string) string { return junta(raiz("MKPACOTES_MK", "/usr/share/mk"), partes) }

// Log é o registro do pacman (ou MKPACOTES_LOG).
func Log() string { return raiz("MKPACOTES_LOG", "/var/log/pacman.log") }

// Cache é a pasta dos pacotes baixados (ou MKPACOTES_CACHE).
func Cache() string { return raiz("MKPACOTES_CACHE", "/var/cache/pacman/pkg") }

// AUR é o endereço do AUR (ou MKPACOTES_AUR), sem a barra do fim.
func AUR() string {
	return strings.TrimRight(raiz("MKPACOTES_AUR", "https://aur.archlinux.org"), "/")
}

// Trava é o arquivo em que o coletor segura o flock de "uma ação de cada vez"
// (ou MKPACOTES_TRAVA): na pasta da sessão, ou na temporária com o uid no nome.
func Trava() string {
	if v := os.Getenv("MKPACOTES_TRAVA"); v != "" {
		return v
	}
	if d := os.Getenv("XDG_RUNTIME_DIR"); d != "" {
		return filepath.Join(d, "mkpacotes.trava")
	}
	return filepath.Join(os.TempDir(), "mkpacotes-"+strconv.Itoa(os.Getuid())+".trava")
}

// Config monta um caminho dentro de ~/.config (ou de XDG_CONFIG_HOME).
func Config(partes ...string) string {
	d := os.Getenv("XDG_CONFIG_HOME")
	if d == "" {
		h, _ := os.UserHomeDir()
		d = filepath.Join(h, ".config")
	}
	return junta(d, partes)
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
// `return base.Falha(base.Recusado, "o pacman está travado")`.
func Falha(codigo int, formato string, args ...any) int {
	_ = Escreve(map[string]string{"erro": fmt.Sprintf(formato, args...)})
	return codigo
}
