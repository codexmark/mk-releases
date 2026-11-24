// Package base tem o que todo subcomando do motor usa: as pastas da pessoa
// (trocáveis pelas variáveis XDG, para os testes), os limites, a saída em
// JSON, os códigos de saída do contrato (../../../CONTRATO.md) e a leitura do
// texto que chega pelo stdin.
package base

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"
)

// Códigos de saída do contrato.
const (
	Certo    = 0
	Erro     = 1
	Recusado = 2
)

// Saida é para onde o JSON vai e Entrada é de onde o texto vem. Os testes
// trocam os dois.
var (
	Saida   io.Writer = os.Stdout
	Entrada io.Reader = os.Stdin
)

var travaDaSaida sync.Mutex

func home() string {
	h, _ := os.UserHomeDir()
	return h
}

func xdg(variavel string, padrao ...string) string {
	if v := os.Getenv(variavel); filepath.IsAbs(v) {
		return v
	}
	return filepath.Join(append([]string{home()}, padrao...)...)
}

// Rascunhos é a pasta dos textos ainda não salvos:
// ${XDG_STATE_HOME:-~/.local/state}/mk/mkescrita/rascunhos.
func Rascunhos() string {
	return filepath.Join(xdg("XDG_STATE_HOME", ".local", "state"), "mk", "mkescrita", "rascunhos")
}

// Preferencias é o arquivo de preferências:
// ${XDG_CONFIG_HOME:-~/.config}/mk/mkescrita.json.
func Preferencias() string {
	return filepath.Join(xdg("XDG_CONFIG_HOME", ".config"), "mk", "mkescrita.json")
}

// Proc é onde se olha se um processo ainda vive (MKESCRITA_PROC, padrão /proc).
func Proc() string {
	if v := os.Getenv("MKESCRITA_PROC"); v != "" {
		return v
	}
	return "/proc"
}

// LimiteBytes é o maior texto que o motor abre ou grava (MKESCRITA_LIMITE_MB,
// padrão 4 MB): acima disso o editor de uma coluna só não é a ferramenta.
func LimiteBytes() int64 {
	mb, err := strconv.ParseInt(strings.TrimSpace(os.Getenv("MKESCRITA_LIMITE_MB")), 10, 64)
	if err != nil || mb <= 0 || mb > 1024 {
		mb = 4
	}
	return mb << 20
}

// Escreve solta um valor como uma linha de JSON, sem escapar acentos nem <>.
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

// Falha escreve {"erro": "..."} e devolve o código dado.
func Falha(codigo int, formato string, args ...any) int {
	_ = Escreve(map[string]string{"erro": fmt.Sprintf(formato, args...)})
	return codigo
}

// recusa é um erro que sai com o código 2: o motor entendeu o pedido e não
// vai fazer (arquivo binário, grande demais, já existe).
type recusa struct{ frase string }

func (r *recusa) Error() string { return r.frase }

// Recusa faz o erro de uma recusa (código de saída 2).
func Recusa(formato string, args ...any) error {
	return &recusa{fmt.Sprintf(formato, args...)}
}

// FoiRecusa diz se o erro é (ou embrulha) uma recusa.
func FoiRecusa(err error) bool {
	var r *recusa
	return errors.As(err, &r)
}

// FalhaDe escreve o erro e devolve o código dele: 2 se foi recusa, 1 se não.
func FalhaDe(err error) int {
	if FoiRecusa(err) {
		return Falha(Recusado, "%s", err.Error())
	}
	return Falha(Erro, "%s", err.Error())
}

// LeTexto lê o texto inteiro do stdin. `bytesEsperados` é o que a janela diz
// que mandou ("" quando não diz): se o que chegou tem outro tamanho, a
// conversa foi cortada no meio e nada deve ser gravado. Texto que não é UTF-8
// ou que passa do limite também é recusado.
func LeTexto(bytesEsperados string) ([]byte, error) {
	limite := LimiteBytes()
	texto, err := io.ReadAll(io.LimitReader(Entrada, limite+1))
	if err != nil {
		return nil, fmt.Errorf("não consegui ler o texto: %v", err)
	}
	if int64(len(texto)) > limite {
		return nil, Recusa("o texto passa de %d MB: não gravo", limite>>20)
	}
	if bytesEsperados != "" {
		n, err := strconv.ParseInt(bytesEsperados, 10, 64)
		if err != nil || n < 0 {
			return nil, fmt.Errorf("--bytes pede um número, veio %q", bytesEsperados)
		}
		if n != int64(len(texto)) {
			return nil, fmt.Errorf("o texto chegou cortado (%d de %d bytes): não gravei nada", len(texto), n)
		}
	}
	if !utf8.Valid(texto) {
		return nil, Recusa("o texto não é UTF-8: não gravo")
	}
	return texto, nil
}

// Nome é o nome de um arquivo para mostrar: a última parte do caminho.
func Nome(caminho string) string {
	if caminho == "" {
		return ""
	}
	return filepath.Base(caminho)
}
