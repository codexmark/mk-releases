// mkfiles-motor: a metade do MkFiles que mexe no disco. A janela (../qml) só
// mostra o que sai daqui. O contrato entre as duas está em ../CONTRATO.md:
// mude lá primeiro.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	"mkfiles/internal/abrir"
	"mkfiles/internal/area"
	"mkfiles/internal/base"
	"mkfiles/internal/busca"
	"mkfiles/internal/compactado"
	"mkfiles/internal/item"
	"mkfiles/internal/lixeira"
	"mkfiles/internal/lugares"
	"mkfiles/internal/miniatura"
	"mkfiles/internal/operar"
	"mkfiles/internal/prefs"
	"mkfiles/internal/previa"
	"mkfiles/internal/props"
	"mkfiles/internal/remoto"
	"mkfiles/internal/vigia"
)

const uso = "uso: mkfiles-motor <servir | op [caminho…] [--campo=valor…]> (os pedidos estão no CONTRATO.md; `mkfiles-motor sobre` lista)"

// O tamanho máximo de uma linha de pedido (uma seleção de dezenas de milhares
// de caminhos cabe).
const limiteDaLinha = 32 << 20

// As extensões: a janela só mostra no menu o que estiver
// ligado aqui.
var extensoes = base.Campos{"lote": true, "compactados": true, "previa": true, "remotos": true}

func main() {
	os.Exit(rodar(os.Args[1:], os.Stdin, os.Stdout))
}

// montar liga cada área do motor à sessão. Uma fase nova é um pacote novo e
// uma linha aqui.
func montar(s *base.Sessao) {
	item.Ligar(s)
	vigia.Ligar(s)
	abrir.Ligar(s)
	lixeira.Ligar(s)
	operar.Ligar(s)
	busca.Ligar(s)
	compactado.Ligar(s)
	previa.Ligar(s)
	miniatura.Ligar(s)
	lugares.Ligar(s)
	props.Ligar(s)
	prefs.Ligar(s)
	area.Ligar(s)
	remoto.Ligar(s)
	s.Ligar("sobre", func(p *base.Pedido) {
		// gvfs: onde os remotos montados aparecem como pastas (null se não há).
		var gvfs any
		if g := base.GVFS(); g != "" {
			gvfs = g
		}
		p.Responde(base.Campos{"versao": 1, "ops": s.Ops(), "extensoes": extensoes, "gvfs": gvfs})
	})
}

func rodar(args []string, entrada io.Reader, saida io.Writer) int {
	s := base.NovaSessao(saida)
	montar(s)
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		s.SemID = true
		(&base.Pedido{S: s}).Falha(uso)
		return base.Erro
	}
	if args[0] == "servir" {
		if len(args) != 1 {
			s.SemID = true
			(&base.Pedido{S: s}).Falha(uso)
			return base.Erro
		}
		return servir(s, entrada)
	}
	// Uma chamada só: o pedido montado dos argumentos, e espera tudo acabar.
	s.SemID = true
	linha, err := pedidoDe(args)
	if err != nil {
		(&base.Pedido{S: s}).Falha(err.Error())
		return base.Erro
	}
	s.Atender(linha)
	s.Espera()
	codigo := s.Codigo()
	s.Fecha()
	return codigo
}

// pedidoDe monta o pedido de uma chamada: os caminhos viram "caminhos" (e o
// primeiro, "caminho"); cada --campo=valor vira um campo, lido como JSON se
// for JSON e como texto se não for. Um caminho relativo é resolvido aqui (é a
// pessoa no terminal); o motor em si só aceita absolutos.
func pedidoDe(args []string) ([]byte, error) {
	pedido := map[string]any{"op": args[0]}
	var caminhos []string
	soCaminhos := false
	for _, a := range args[1:] {
		if a == "--" && !soCaminhos {
			soCaminhos = true
			continue
		}
		if !soCaminhos && strings.HasPrefix(a, "--") {
			campo, valor, _ := strings.Cut(a[2:], "=")
			campo = strings.ReplaceAll(campo, "-", "_")
			var v any
			d := json.NewDecoder(strings.NewReader(valor))
			d.UseNumber()
			if d.Decode(&v) != nil || d.More() {
				v = valor
			}
			pedido[campo] = v
			continue
		}
		abs, err := filepath.Abs(a)
		if err != nil {
			return nil, err
		}
		caminhos = append(caminhos, abs)
	}
	if len(caminhos) > 0 {
		pedido["caminhos"] = caminhos
		pedido["caminho"] = caminhos[0]
	}
	return json.Marshal(pedido)
}

// servir lê um pedido por linha até a entrada fechar. Pedido torto vira uma
// resposta de erro; o servidor continua. Ao fechar, o que estava em andamento
// é cancelado.
func servir(s *base.Sessao, entrada io.Reader) int {
	linhas := bufio.NewScanner(entrada)
	linhas.Buffer(make([]byte, 0, 64<<10), limiteDaLinha)
	for linhas.Scan() {
		linha := bytes.TrimSpace(linhas.Bytes())
		if len(linha) == 0 {
			continue
		}
		s.Atender(linha)
	}
	codigo := base.Certo
	if linhas.Err() != nil {
		s.SemID = false
		(&base.Pedido{S: s}).Falha("pedido grande demais")
		codigo = base.Erro
	}
	s.Fecha()
	return codigo
}
