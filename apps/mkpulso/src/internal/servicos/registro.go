package servicos

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"syscall"
)

// Como o systemctl kill chama cada sinal do contrato.
var nomeDoSinal = map[syscall.Signal]string{
	syscall.SIGTERM: "SIGTERM",
	syscall.SIGKILL: "SIGKILL",
	syscall.SIGSTOP: "SIGSTOP",
	syscall.SIGCONT: "SIGCONT",
	syscall.SIGHUP:  "SIGHUP",
	syscall.SIGINT:  "SIGINT",
}

// Linha é uma linha do journal, como em `mkpulso-coletor registro`.
type Linha struct {
	TS    *int64 `json:"t_s"`
	Nivel string `json:"nivel"`
	Texto string `json:"texto"`
}

// O máximo de linhas que o contrato deixa pedir, e o padrão.
const (
	MaximoDeLinhas = 500
	PadraoDeLinhas = 80
)

// Registro devolve as últimas n linhas (1 a 500) do journal de uma unidade,
// da mais antiga para a mais nova.
func Registro(unidade string, usuario bool, n int) ([]Linha, error) {
	if n < 1 || n > MaximoDeLinhas {
		return nil, fmt.Errorf("o número de linhas vai de 1 a %d, e veio %d", MaximoDeLinhas, n)
	}
	if err := ValidaUnidade(unidade); err != nil {
		return nil, err
	}
	args := []string{"-u", unidade, "-n", strconv.Itoa(n), "-o", "json", "--no-pager"}
	if usuario {
		args = append([]string{"--user"}, args...)
	}
	saida, err := roda("journalctl", prazoDeLeitura, args...)
	if err != nil {
		return nil, fmt.Errorf("não deu para ler o registro de %s: %v", unidade, err)
	}
	linhas := []Linha{}
	leitor := bufio.NewReader(bytes.NewReader(saida))
	for {
		bruta, err := leitor.ReadBytes('\n')
		if l, ok := leLinha(bruta); ok {
			linhas = append(linhas, l)
		}
		if err != nil {
			break
		}
	}
	return linhas, nil
}

// leLinha traduz um objeto do `journalctl -o json`. O que não é um objeto
// JSON (uma linha vazia, um aviso) fica de fora.
func leLinha(bruta []byte) (Linha, bool) {
	var campos map[string]json.RawMessage
	if json.Unmarshal(bytes.TrimSpace(bruta), &campos) != nil || campos == nil {
		return Linha{}, false
	}
	l := Linha{Nivel: nivel(texto(campos["PRIORITY"])), Texto: texto(campos["MESSAGE"])}
	if us, err := strconv.ParseInt(texto(campos["__REALTIME_TIMESTAMP"]), 10, 64); err == nil && us > 0 {
		s := us / 1_000_000
		l.TS = &s
	}
	return l, true
}

// nivel leva a prioridade do syslog (0 a 7) para o `nivel` do contrato.
func nivel(prioridade string) string {
	switch prioridade {
	case "0", "1", "2", "3":
		return "erro"
	case "4":
		return "aviso"
	case "7":
		return "depuração"
	}
	return "info"
}

// texto lê um campo do journal. Em geral é uma string; quando o conteúdo
// não é UTF-8 limpo, o journald manda a lista dos bytes; um campo repetido
// vem como lista (fica o primeiro); grande demais, vem null.
func texto(campo json.RawMessage) string {
	if len(campo) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(campo, &s) == nil {
		return s
	}
	var numeros []int
	if json.Unmarshal(campo, &numeros) == nil {
		b := make([]byte, len(numeros))
		for i, n := range numeros {
			b[i] = byte(n)
		}
		return strings.ToValidUTF8(string(b), "�")
	}
	var varios []json.RawMessage
	if json.Unmarshal(campo, &varios) == nil && len(varios) > 0 {
		return texto(varios[0])
	}
	return ""
}
