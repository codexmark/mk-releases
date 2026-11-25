// Package base tem o que todo subcomando do motor usa: as pastas e os
// comandos (trocáveis por variável de ambiente, para os testes), a escrita
// atômica, a saída em JSON e os códigos de saída do contrato
// (../../../CONTRATO.md).
package base

import (
	"bytes"
	"encoding/json"
	"errors"
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

// Config monta um caminho dentro de ~/.config (ou de XDG_CONFIG_HOME).
func Config(partes ...string) string {
	d := os.Getenv("XDG_CONFIG_HOME")
	if d == "" {
		d = filepath.Join(Home(), ".config")
	}
	return filepath.Join(append([]string{d}, partes...)...)
}

// PastaDoApp é ~/.config/mk/mksom (ou MKSOM_CONFIG): o estado, as
// predefinições e o eq.conf.
func PastaDoApp() string {
	if v := os.Getenv("MKSOM_CONFIG"); v != "" {
		return filepath.Clean(v)
	}
	return Config("mk", "mksom")
}

// PastaDasUnidades é ~/.config/systemd/user (ou MKSOM_UNIDADES).
func PastaDasUnidades() string {
	if v := os.Getenv("MKSOM_UNIDADES"); v != "" {
		return filepath.Clean(v)
	}
	return Config("systemd", "user")
}

// PastaTemporaria é $XDG_RUNTIME_DIR/mksom: memória, não disco. Sem a
// variável não há para onde gravar (nada de /tmp).
func PastaTemporaria() (string, error) {
	d := os.Getenv("XDG_RUNTIME_DIR")
	if d == "" || !filepath.IsAbs(d) {
		return "", errors.New("não há pasta temporária em memória (XDG_RUNTIME_DIR): a gravação de teste não vai para disco")
	}
	return filepath.Join(filepath.Clean(d), "mksom"), nil
}

// Comando é o nome de um comando externo: o da variável, ou o padrão.
func Comando(variavel, padrao string) string {
	if v := os.Getenv(variavel); v != "" {
		return v
	}
	return padrao
}

// dentro diz se o caminho é a pasta ou está abaixo dela.
func dentro(caminho, pasta string) bool {
	rel, err := filepath.Rel(pasta, caminho)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// DoSistemaDeSom diz se o caminho cai na configuração do PipeWire ou do
// WirePlumber da pessoa: ali o MkSom nunca escreve.
func DoSistemaDeSom(caminho string) bool {
	caminho = filepath.Clean(caminho)
	pastas := []string{Config("pipewire"), Config("wireplumber")}
	if h := Home(); h != "" {
		pastas = append(pastas, filepath.Join(h, ".config", "pipewire"), filepath.Join(h, ".config", "wireplumber"))
	}
	for _, p := range pastas {
		if dentro(caminho, p) {
			return true
		}
	}
	return false
}

// EscreveAtomico grava o arquivo inteiro: temporário na mesma pasta (600) e
// rename. A pasta é criada com 700. Recusa a configuração do PipeWire.
func EscreveAtomico(caminho string, dados []byte) error {
	if DoSistemaDeSom(caminho) {
		return fmt.Errorf("o MkSom não escreve na configuração do PipeWire (%s)", caminho)
	}
	pasta := filepath.Dir(caminho)
	if err := os.MkdirAll(pasta, 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(pasta, "."+filepath.Base(caminho)+".*")
	if err != nil {
		return err
	}
	nome := f.Name()
	_, err = f.Write(dados)
	if err == nil {
		err = f.Chmod(0o600)
	}
	if e := f.Close(); err == nil {
		err = e
	}
	if err == nil {
		err = os.Rename(nome, caminho)
	}
	if err != nil {
		_ = os.Remove(nome)
	}
	return err
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
// `return base.Falha(base.Recusado, "essa predefinição é de fábrica")`.
func Falha(codigo int, formato string, args ...any) int {
	_ = Escreve(map[string]string{"erro": fmt.Sprintf(formato, args...)})
	return codigo
}
