// Package base tem o que todo subcomando do motor usa: a home e o cache da
// pessoa (trocáveis por MKCORTE_HOME e XDG_CACHE_HOME, para os testes), a
// saída em JSON, os códigos de saída do contrato (../../../CONTRATO.md) e o
// jeito de rodar um comando externo (sempre com lista de argumentos, nunca
// por shell).
package base

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
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

// Home monta um caminho dentro da home de quem roda (ou de MKCORTE_HOME).
func Home(partes ...string) string {
	h := os.Getenv("MKCORTE_HOME")
	if h == "" {
		h, _ = os.UserHomeDir()
	}
	return filepath.Join(append([]string{h}, partes...)...)
}

// Cache monta um caminho dentro do cache do MkCorte:
// ${XDG_CACHE_HOME:-~/.cache}/mk/mkcorte. Um XDG_CACHE_HOME que não é
// caminho absoluto não vale (é o que a norma manda).
func Cache(partes ...string) string {
	raiz := os.Getenv("XDG_CACHE_HOME")
	if !filepath.IsAbs(raiz) {
		raiz = Home(".cache")
	}
	return filepath.Join(append([]string{raiz, "mk", "mkcorte"}, partes...)...)
}

// Til escreve um caminho do jeito que a pessoa lê: a home vira "~".
func Til(caminho string) string {
	h := Home()
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

// Falha escreve {"erro": "..."} e devolve o código dado.
func Falha(codigo int, formato string, args ...any) int {
	_ = Escreve(map[string]string{"erro": fmt.Sprintf(formato, args...)})
	return codigo
}

// recusa é um erro que sai com o código 2: um pedido que o motor não aceita.
type recusa struct{ frase string }

func (r *recusa) Error() string { return r.frase }

// Recusa faz o erro de um pedido recusado (código de saída 2).
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

// Tem diz se um comando externo existe no PATH.
func Tem(nome string) bool {
	_, err := exec.LookPath(nome)
	return err == nil
}

// O máximo que se guarda da saída de um comando de leitura.
const maximoDeSaida = 16 << 20

// Roda executa um comando externo e devolve o stdout (até 16 MB). O erro já
// vem como frase: o comando não existe, demorou demais, ou o que ele disse no
// stderr (resumido numa linha).
func Roda(nome string, prazo time.Duration, args ...string) ([]byte, error) {
	caminho, err := exec.LookPath(nome)
	if err != nil {
		return nil, fmt.Errorf("não achei o %s nesta máquina", nome)
	}
	ctx := context.Background()
	if prazo > 0 {
		var cancela context.CancelFunc
		ctx, cancela = context.WithTimeout(ctx, prazo)
		defer cancela()
	}
	var saida, erros limitado
	saida.max, erros.max = maximoDeSaida, 64<<10
	cmd := exec.CommandContext(ctx, caminho, args...)
	cmd.Stdout, cmd.Stderr = &saida, &erros
	cmd.WaitDelay = time.Second
	err = cmd.Run()
	if err == nil {
		return saida.b.Bytes(), nil
	}
	if ctx.Err() != nil {
		return nil, fmt.Errorf("o %s demorou demais e foi interrompido", nome)
	}
	if frase := Resumo(erros.b.String()); frase != "" {
		return nil, errors.New(frase)
	}
	var saiu *exec.ExitError
	if errors.As(err, &saiu) {
		return nil, fmt.Errorf("o %s saiu com o código %d", nome, saiu.ExitCode())
	}
	return nil, fmt.Errorf("não consegui rodar o %s: %v", nome, err)
}

// limitado guarda até max bytes e joga o resto fora (sem travar quem escreve).
type limitado struct {
	b   bytes.Buffer
	max int
}

func (l *limitado) Write(p []byte) (int, error) {
	if sobra := l.max - l.b.Len(); sobra > 0 {
		if len(p) > sobra {
			l.b.Write(p[:sobra])
		} else {
			l.b.Write(p)
		}
	}
	return len(p), nil
}

// Resumo fica com a última linha não vazia de um stderr (é onde os comandos
// dizem o que deu errado), sem espaços repetidos e sem passar de 300 letras.
func Resumo(stderr string) string {
	linhas := strings.Split(stderr, "\n")
	for i := len(linhas) - 1; i >= 0; i-- {
		linha := strings.Join(strings.Fields(linhas[i]), " ")
		if linha == "" {
			continue
		}
		if letras := []rune(linha); len(letras) > 300 {
			linha = string(letras[:300]) + "…"
		}
		return linha
	}
	return ""
}
