// SPDX-License-Identifier: GPL-3.0-or-later

// Pacote dbus: um cliente mínimo do barramento de sessão do D-Bus, escrito à
// mão a partir da especificação pública (freedesktop), só com o que o MPRIS do
// MkTunes precisa: a conexão por socket Unix com autenticação EXTERNAL, as
// mensagens (chamada, resposta, erro e sinal) e a serialização dos tipos
// y b n q i u x t d s o g a v, structs e dicionários.
package dbus

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
	"unicode/utf8"
)

// Caminho é um caminho de objeto (tipo 'o').
type Caminho string

// Assinatura é uma assinatura de tipos (tipo 'g').
type Assinatura string

// Variante é um valor que carrega o próprio tipo (tipo 'v').
type Variante struct {
	Assinatura string
	Valor      any
}

// Par é uma entrada de dicionário cuja chave não é texto ou cujo valor não é
// uma variante (os a{sv} viram map[string]Variante).
type Par struct{ Chave, Valor any }

const (
	// mensagemMax é a maior mensagem aceita (a especificação permite 128 MiB;
	// nada do MPRIS chega perto de 1).
	mensagemMax      = 16 << 20
	profundidadeMax  = 32
	assinaturaMaxima = 255
)

var errCurta = errors.New("dbus: mensagem cortada")

// umTipo separa o primeiro tipo completo de uma assinatura.
func umTipo(assinatura string) (tipo, resto string, erro error) {
	if assinatura == "" {
		return "", "", errors.New("dbus: assinatura vazia")
	}
	switch c := assinatura[0]; c {
	case 'y', 'b', 'n', 'q', 'i', 'u', 'x', 't', 'd', 's', 'o', 'g', 'v':
		return assinatura[:1], assinatura[1:], nil
	case 'a':
		elemento, resto, erro := umTipo(assinatura[1:])
		if erro != nil {
			return "", "", erro
		}
		return "a" + elemento, resto, nil
	case '(', '{':
		fecha := map[byte]byte{'(': ')', '{': '}'}[c]
		fundo := 0
		for i := 0; i < len(assinatura); i++ {
			switch assinatura[i] {
			case '(', '{':
				fundo++
			case ')', '}':
				if fundo--; fundo == 0 {
					if assinatura[i] != fecha || i == 1 {
						return "", "", errors.New("dbus: assinatura malformada: " + assinatura)
					}
					return assinatura[:i+1], assinatura[i+1:], nil
				}
			}
		}
	}
	return "", "", errors.New("dbus: assinatura malformada: " + assinatura)
}

// alinhamento de um tipo, pelo primeiro caractere.
func alinhamento(tipo byte) int {
	switch tipo {
	case 'y', 'g', 'v':
		return 1
	case 'n', 'q':
		return 2
	case 'x', 't', 'd', '(', '{':
		return 8
	}
	return 4
}

type escritor struct{ b []byte }

func (e *escritor) alinha(n int) {
	for len(e.b)%n != 0 {
		e.b = append(e.b, 0)
	}
}

func (e *escritor) u32(v uint32) {
	e.alinha(4)
	e.b = binary.LittleEndian.AppendUint32(e.b, v)
}

func (e *escritor) texto(s string) error {
	if !utf8.ValidString(s) || strings.ContainsRune(s, 0) {
		return errors.New("dbus: texto inválido")
	}
	e.u32(uint32(len(s)))
	e.b = append(append(e.b, s...), 0)
	return nil
}

func (e *escritor) assinatura(s string) error {
	if len(s) > assinaturaMaxima {
		return errors.New("dbus: assinatura longa demais")
	}
	e.b = append(append(append(e.b, byte(len(s))), s...), 0)
	return nil
}

func tipoErrado(tipo string, v any) error {
	return fmt.Errorf("dbus: o tipo %q não aceita %T", tipo, v)
}

// poe escreve um valor do tipo dado (um tipo completo).
func (e *escritor) poe(tipo string, v any) error {
	switch tipo[0] {
	case 'y':
		valor, ok := v.(byte)
		if !ok {
			return tipoErrado(tipo, v)
		}
		e.b = append(e.b, valor)
	case 'b':
		valor, ok := v.(bool)
		if !ok {
			return tipoErrado(tipo, v)
		}
		if valor {
			e.u32(1)
		} else {
			e.u32(0)
		}
	case 'n':
		valor, ok := v.(int16)
		if !ok {
			return tipoErrado(tipo, v)
		}
		e.alinha(2)
		e.b = binary.LittleEndian.AppendUint16(e.b, uint16(valor))
	case 'q':
		valor, ok := v.(uint16)
		if !ok {
			return tipoErrado(tipo, v)
		}
		e.alinha(2)
		e.b = binary.LittleEndian.AppendUint16(e.b, valor)
	case 'i':
		valor, ok := v.(int32)
		if !ok {
			return tipoErrado(tipo, v)
		}
		e.u32(uint32(valor))
	case 'u':
		valor, ok := v.(uint32)
		if !ok {
			return tipoErrado(tipo, v)
		}
		e.u32(valor)
	case 'x':
		valor, ok := v.(int64)
		if !ok {
			return tipoErrado(tipo, v)
		}
		e.alinha(8)
		e.b = binary.LittleEndian.AppendUint64(e.b, uint64(valor))
	case 't':
		valor, ok := v.(uint64)
		if !ok {
			return tipoErrado(tipo, v)
		}
		e.alinha(8)
		e.b = binary.LittleEndian.AppendUint64(e.b, valor)
	case 'd':
		valor, ok := v.(float64)
		if !ok {
			return tipoErrado(tipo, v)
		}
		e.alinha(8)
		e.b = binary.LittleEndian.AppendUint64(e.b, math.Float64bits(valor))
	case 's':
		valor, ok := v.(string)
		if !ok {
			return tipoErrado(tipo, v)
		}
		return e.texto(valor)
	case 'o':
		valor, ok := v.(Caminho)
		if !ok || !CaminhoValido(string(valor)) {
			return tipoErrado(tipo, v)
		}
		return e.texto(string(valor))
	case 'g':
		valor, ok := v.(Assinatura)
		if !ok {
			return tipoErrado(tipo, v)
		}
		return e.assinatura(string(valor))
	case 'v':
		valor, ok := v.(Variante)
		if !ok {
			return tipoErrado(tipo, v)
		}
		interno, resto, erro := umTipo(valor.Assinatura)
		if erro != nil || resto != "" {
			return errors.New("dbus: a variante tem de ter um tipo só: " + valor.Assinatura)
		}
		if erro := e.assinatura(interno); erro != nil {
			return erro
		}
		return e.poe(interno, valor.Valor)
	case '(':
		campos, ok := v.([]any)
		if !ok {
			return tipoErrado(tipo, v)
		}
		e.alinha(8)
		return e.poeVarios(tipo[1:len(tipo)-1], campos)
	case 'a':
		return e.poeLista(tipo[1:], v)
	default:
		return errors.New("dbus: tipo desconhecido: " + tipo)
	}
	return nil
}

func (e *escritor) poeVarios(assinatura string, valores []any) error {
	for _, v := range valores {
		tipo, resto, erro := umTipo(assinatura)
		if erro != nil {
			return erro
		}
		if erro := e.poe(tipo, v); erro != nil {
			return erro
		}
		assinatura = resto
	}
	if assinatura != "" {
		return errors.New("dbus: faltam valores para " + assinatura)
	}
	return nil
}

func (e *escritor) poeLista(elemento string, v any) error {
	e.u32(0)
	onde := len(e.b) - 4
	e.alinha(alinhamento(elemento[0]))
	comeco := len(e.b)
	var erro error
	if elemento[0] == '{' {
		chave, valor := elemento[1:2], elemento[2:len(elemento)-1]
		pares, ok := v.([]Par)
		if dicionario, eMapa := v.(map[string]Variante); eMapa && chave == "s" && valor == "v" {
			// Em ordem de chave: a mesma mensagem sai sempre com os mesmos bytes.
			chaves := make([]string, 0, len(dicionario))
			for c := range dicionario {
				chaves = append(chaves, c)
			}
			sort.Strings(chaves)
			for _, c := range chaves {
				pares = append(pares, Par{c, dicionario[c]})
			}
		} else if !ok {
			return tipoErrado("a"+elemento, v)
		}
		for _, par := range pares {
			e.alinha(8)
			if erro = e.poe(chave, par.Chave); erro == nil {
				erro = e.poe(valor, par.Valor)
			}
			if erro != nil {
				return erro
			}
		}
	} else {
		switch itens := v.(type) {
		case []string:
			for _, item := range itens {
				if erro = e.poe(elemento, item); erro != nil {
					return erro
				}
			}
		case []byte:
			if elemento != "y" {
				return tipoErrado("a"+elemento, v)
			}
			e.b = append(e.b, itens...)
		case []any:
			for _, item := range itens {
				if erro = e.poe(elemento, item); erro != nil {
					return erro
				}
			}
		default:
			return tipoErrado("a"+elemento, v)
		}
	}
	binary.LittleEndian.PutUint32(e.b[onde:], uint32(len(e.b)-comeco))
	return nil
}

// Codifica serializa os valores segundo a assinatura (em little-endian, a
// partir do deslocamento zero).
func Codifica(assinatura string, valores ...any) ([]byte, error) {
	e := &escritor{}
	if erro := e.poeVarios(assinatura, valores); erro != nil {
		return nil, erro
	}
	return e.b, nil
}

type leitor struct {
	b     []byte
	pos   int
	ordem binary.ByteOrder
}

func (l *leitor) alinha(n int) error {
	for l.pos%n != 0 {
		if l.pos >= len(l.b) {
			return errCurta
		}
		l.pos++
	}
	return nil
}

func (l *leitor) pega(n int) ([]byte, error) {
	if n < 0 || l.pos+n > len(l.b) {
		return nil, errCurta
	}
	pedaco := l.b[l.pos : l.pos+n]
	l.pos += n
	return pedaco, nil
}

func (l *leitor) fixo(n int) ([]byte, error) {
	if erro := l.alinha(n); erro != nil {
		return nil, erro
	}
	return l.pega(n)
}

func (l *leitor) u32() (uint32, error) {
	b, erro := l.fixo(4)
	if erro != nil {
		return 0, erro
	}
	return l.ordem.Uint32(b), nil
}

func (l *leitor) texto() (string, error) {
	tamanho, erro := l.u32()
	if erro != nil {
		return "", erro
	}
	b, erro := l.pega(int(tamanho) + 1)
	if erro != nil {
		return "", erro
	}
	if b[tamanho] != 0 || !utf8.Valid(b[:tamanho]) {
		return "", errors.New("dbus: texto inválido")
	}
	return string(b[:tamanho]), nil
}

func (l *leitor) assinatura() (string, error) {
	tamanho, erro := l.pega(1)
	if erro != nil {
		return "", erro
	}
	b, erro := l.pega(int(tamanho[0]) + 1)
	if erro != nil {
		return "", erro
	}
	return string(b[:tamanho[0]]), nil
}

func (l *leitor) le(tipo string, fundo int) (any, error) {
	if fundo > profundidadeMax {
		return nil, errors.New("dbus: valor aninhado demais")
	}
	switch tipo[0] {
	case 'y':
		b, erro := l.pega(1)
		if erro != nil {
			return nil, erro
		}
		return b[0], nil
	case 'b':
		valor, erro := l.u32()
		return valor != 0, erro
	case 'n':
		b, erro := l.fixo(2)
		if erro != nil {
			return nil, erro
		}
		return int16(l.ordem.Uint16(b)), nil
	case 'q':
		b, erro := l.fixo(2)
		if erro != nil {
			return nil, erro
		}
		return l.ordem.Uint16(b), nil
	case 'i':
		valor, erro := l.u32()
		return int32(valor), erro
	case 'u':
		return l.u32()
	case 'x', 't', 'd':
		b, erro := l.fixo(8)
		if erro != nil {
			return nil, erro
		}
		bruto := l.ordem.Uint64(b)
		switch tipo[0] {
		case 'x':
			return int64(bruto), nil
		case 'd':
			return math.Float64frombits(bruto), nil
		}
		return bruto, nil
	case 's':
		return l.texto()
	case 'o':
		valor, erro := l.texto()
		return Caminho(valor), erro
	case 'g':
		valor, erro := l.assinatura()
		return Assinatura(valor), erro
	case 'v':
		assinatura, erro := l.assinatura()
		if erro != nil {
			return nil, erro
		}
		interno, resto, erro := umTipo(assinatura)
		if erro != nil || resto != "" {
			return nil, errors.New("dbus: variante com assinatura inválida")
		}
		valor, erro := l.le(interno, fundo+1)
		return Variante{interno, valor}, erro
	case '(':
		if erro := l.alinha(8); erro != nil {
			return nil, erro
		}
		return l.leVarios(tipo[1:len(tipo)-1], fundo+1)
	case 'a':
		return l.leLista(tipo[1:], fundo+1)
	}
	return nil, errors.New("dbus: tipo desconhecido: " + tipo)
}

func (l *leitor) leVarios(assinatura string, fundo int) ([]any, error) {
	valores := []any{}
	for assinatura != "" {
		tipo, resto, erro := umTipo(assinatura)
		if erro != nil {
			return nil, erro
		}
		valor, erro := l.le(tipo, fundo)
		if erro != nil {
			return nil, erro
		}
		valores, assinatura = append(valores, valor), resto
	}
	return valores, nil
}

func (l *leitor) leLista(elemento string, fundo int) (any, error) {
	tamanho, erro := l.u32()
	if erro != nil {
		return nil, erro
	}
	if erro := l.alinha(alinhamento(elemento[0])); erro != nil {
		return nil, erro
	}
	fim := l.pos + int(tamanho)
	if tamanho > mensagemMax || fim > len(l.b) {
		return nil, errCurta
	}
	switch {
	case elemento == "{sv}":
		dicionario := map[string]Variante{}
		for l.pos < fim {
			par, erro := l.lePar(elemento, fundo)
			if erro != nil {
				return nil, erro
			}
			dicionario[par.Chave.(string)] = par.Valor.(Variante)
		}
		return dicionario, nil
	case elemento[0] == '{':
		pares := []Par{}
		for l.pos < fim {
			par, erro := l.lePar(elemento, fundo)
			if erro != nil {
				return nil, erro
			}
			pares = append(pares, par)
		}
		return pares, nil
	case elemento == "s":
		textos := []string{}
		for l.pos < fim {
			valor, erro := l.texto()
			if erro != nil {
				return nil, erro
			}
			textos = append(textos, valor)
		}
		return textos, nil
	case elemento == "y":
		b, erro := l.pega(int(tamanho))
		return append([]byte{}, b...), erro
	}
	itens := []any{}
	for l.pos < fim {
		valor, erro := l.le(elemento, fundo)
		if erro != nil {
			return nil, erro
		}
		itens = append(itens, valor)
	}
	return itens, nil
}

func (l *leitor) lePar(elemento string, fundo int) (Par, error) {
	if erro := l.alinha(8); erro != nil {
		return Par{}, erro
	}
	chave, erro := l.le(elemento[1:2], fundo)
	if erro != nil {
		return Par{}, erro
	}
	valor, erro := l.le(elemento[2:len(elemento)-1], fundo)
	return Par{chave, valor}, erro
}

// Decodifica lê os valores de uma assinatura (little-endian, do deslocamento zero).
func Decodifica(assinatura string, dados []byte) ([]any, error) {
	l := &leitor{b: dados, ordem: binary.LittleEndian}
	return l.leVarios(assinatura, 0)
}

// CaminhoValido confere um caminho de objeto: "/" ou partes de [A-Za-z0-9_]
// separadas por "/".
func CaminhoValido(caminho string) bool {
	if caminho == "/" {
		return true
	}
	if !strings.HasPrefix(caminho, "/") || strings.HasSuffix(caminho, "/") {
		return false
	}
	for _, parte := range strings.Split(caminho[1:], "/") {
		if parte == "" {
			return false
		}
		for _, c := range parte {
			if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_') {
				return false
			}
		}
	}
	return true
}

// Tipos de mensagem.
const (
	Chamada  byte = 1
	Resposta byte = 2
	Erro     byte = 3
	Sinal    byte = 4
)

// SemResposta é a marca de "não precisa responder" de uma chamada.
const SemResposta byte = 1

// Mensagem é uma mensagem do barramento, com os campos do cabeçalho abertos.
type Mensagem struct {
	Tipo       byte
	Marcas     byte
	Serial     uint32
	Caminho    Caminho
	Interface  string
	Membro     string
	NomeDoErro string
	RespostaA  uint32
	Destino    string
	Remetente  string
	Assinatura string
	Corpo      []any
}

// Codifica monta os bytes da mensagem (little-endian, protocolo 1).
func (m *Mensagem) Codifica() ([]byte, error) {
	corpo, erro := Codifica(m.Assinatura, m.Corpo...)
	if erro != nil {
		return nil, erro
	}
	var campos []any
	campo := func(codigo byte, assinatura string, valor any) {
		campos = append(campos, []any{codigo, Variante{assinatura, valor}})
	}
	if m.Caminho != "" {
		campo(1, "o", m.Caminho)
	}
	if m.Interface != "" {
		campo(2, "s", m.Interface)
	}
	if m.Membro != "" {
		campo(3, "s", m.Membro)
	}
	if m.NomeDoErro != "" {
		campo(4, "s", m.NomeDoErro)
	}
	if m.RespostaA != 0 {
		campo(5, "u", m.RespostaA)
	}
	if m.Destino != "" {
		campo(6, "s", m.Destino)
	}
	if m.Remetente != "" {
		campo(7, "s", m.Remetente)
	}
	if m.Assinatura != "" {
		campo(8, "g", Assinatura(m.Assinatura))
	}
	e := &escritor{b: []byte{'l', m.Tipo, m.Marcas, 1}}
	e.u32(uint32(len(corpo)))
	e.u32(m.Serial)
	if erro := e.poe("a(yv)", campos); erro != nil {
		return nil, erro
	}
	e.alinha(8)
	if len(e.b)+len(corpo) > mensagemMax {
		return nil, errors.New("dbus: mensagem grande demais")
	}
	return append(e.b, corpo...), nil
}

// LeMensagem lê uma mensagem inteira do fio.
func LeMensagem(r io.Reader) (*Mensagem, error) {
	cabeca := make([]byte, 16)
	if _, erro := io.ReadFull(r, cabeca); erro != nil {
		return nil, erro
	}
	var ordem binary.ByteOrder = binary.LittleEndian
	switch cabeca[0] {
	case 'l':
	case 'B':
		ordem = binary.BigEndian
	default:
		return nil, errors.New("dbus: ordem de bytes desconhecida")
	}
	if cabeca[3] != 1 {
		return nil, errors.New("dbus: versão de protocolo desconhecida")
	}
	tamanhoDoCorpo, tamanhoDosCampos := ordem.Uint32(cabeca[4:]), ordem.Uint32(cabeca[12:])
	if tamanhoDoCorpo > mensagemMax || tamanhoDosCampos > mensagemMax {
		return nil, errors.New("dbus: mensagem grande demais")
	}
	resto := int(tamanhoDosCampos)
	if sobra := (16 + resto) % 8; sobra != 0 {
		resto += 8 - sobra
	}
	dados := make([]byte, 16+resto+int(tamanhoDoCorpo))
	copy(dados, cabeca)
	if _, erro := io.ReadFull(r, dados[16:]); erro != nil {
		return nil, erro
	}
	m := &Mensagem{Tipo: cabeca[1], Marcas: cabeca[2], Serial: ordem.Uint32(cabeca[8:])}
	l := &leitor{b: dados[:16+resto], pos: 12, ordem: ordem}
	campos, erro := l.le("a(yv)", 0)
	if erro != nil {
		return nil, erro
	}
	for _, bruto := range campos.([]any) {
		campo := bruto.([]any)
		valor := campo[1].(Variante).Valor
		switch campo[0].(byte) {
		case 1:
			m.Caminho, _ = valor.(Caminho)
		case 2:
			m.Interface, _ = valor.(string)
		case 3:
			m.Membro, _ = valor.(string)
		case 4:
			m.NomeDoErro, _ = valor.(string)
		case 5:
			m.RespostaA, _ = valor.(uint32)
		case 6:
			m.Destino, _ = valor.(string)
		case 7:
			m.Remetente, _ = valor.(string)
		case 8:
			assinatura, _ := valor.(Assinatura)
			m.Assinatura = string(assinatura)
		}
	}
	// O corpo começa num múltiplo de 8: o alinhamento dentro dele conta dali.
	corpo := &leitor{b: dados[16+resto:], ordem: ordem}
	if m.Corpo, erro = corpo.leVarios(m.Assinatura, 0); erro != nil {
		return nil, erro
	}
	return m, nil
}
