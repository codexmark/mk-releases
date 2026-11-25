package main

import (
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"
	"time"

	"mksom/internal/base"
	"mksom/internal/eq"
	"mksom/internal/pw"
)

func init() {
	registrar("eq", cmdEq)
}

// Quanto o motor espera o sink aparecer depois de subir a unidade, e de
// quanto em quanto tempo olha.
var (
	esperaDoSink   = 5 * time.Second
	esperaDoPadrao = 1500 * time.Millisecond
	passoDaEspera  = 200 * time.Millisecond
)

const usoDoEq = "uso: mksom-motor eq <ligar | desligar | ajustar | predefinicao <aplicar | guardar | renomear | apagar> -- <nome>>"

func cmdEq(args []string) int {
	if len(args) == 0 {
		return base.Falha(base.Erro, usoDoEq)
	}
	switch args[0] {
	case "ligar", "desligar", "ajustar":
		if len(args) != 1 {
			return base.Falha(base.Erro, usoDoEq)
		}
		switch args[0] {
		case "ligar":
			return eqLigar()
		case "desligar":
			return eqDesligar()
		}
		return eqAjustar()
	case "predefinicao":
		return eqPredefinicao(args[1:])
	}
	return base.Falha(base.Erro, usoDoEq)
}

type ligadoJSON struct {
	Ok            bool          `json:"ok"`
	Ligado        bool          `json:"ligado"`
	NaSaida       bool          `json:"na_saida"`
	SaidaAnterior *anteriorJSON `json:"saida_anterior"`
	Aviso         *string       `json:"aviso"`
}

func systemctl(args ...string) error {
	_, err := oSystemctl.roda(append([]string{"--user"}, args...)...)
	return err
}

// desfaz derruba a unidade e anota que o equalizador está desligado: nada
// fica ligado pela metade.
func desfaz(st guardado) {
	_ = systemctl("disable", "--now", eq.NomeDaUnidade)
	st.Ligado = false
	_ = gravaEstado(st)
}

// esperaOSink olha o pw-dump até o sink do equalizador aparecer, ou o prazo
// acabar.
func esperaOSink() (pw.Grafo, pw.No, error) {
	limite := time.Now().Add(esperaDoSink)
	for {
		g, err := leSom()
		if err == nil {
			if no, ok := g.Saida(eq.NoDoSink); ok {
				return g, no, nil
			}
		}
		if !time.Now().Before(limite) {
			if err != nil {
				return g, pw.No{}, err
			}
			return g, pw.No{}, errors.New("o equalizador não apareceu no PipeWire")
		}
		time.Sleep(passoDaEspera)
	}
}

// esperaVirarPadrao olha o pw-dump até o equalizador ser a saída padrão: o
// WirePlumber aceita o set-default na hora, mas só troca a padrão de verdade
// um instante depois. Devolve o último grafo lido e se virou.
func esperaVirarPadrao(g pw.Grafo) (pw.Grafo, bool) {
	limite := time.Now().Add(esperaDoPadrao)
	for {
		if novo, err := leSom(); err == nil {
			g = novo
			if g.SaidaPadrao == eq.NoDoSink {
				return g, true
			}
		}
		if !time.Now().Before(limite) {
			return g, false
		}
		time.Sleep(passoDaEspera)
	}
}

func eqLigar() int {
	g, err := leSom()
	if err != nil {
		return base.Falha(base.Erro, "%v", err)
	}
	st := leEstado()
	// A saída de verdade: a padrão de agora, a menos que já seja o equalizador.
	if g.SaidaPadrao != "" && g.SaidaPadrao != eq.NoDoSink && eq.AlvoSeguro(g.SaidaPadrao) {
		st.SaidaAnterior = g.SaidaPadrao
	}
	sink, noAr := g.Saida(eq.NoDoSink)
	if !noAr {
		if err := gravaConf(st.ajuste(), st.SaidaAnterior); err != nil {
			return base.Falha(base.Erro, "não consegui escrever a configuração do equalizador: %v", err)
		}
		if err := base.EscreveAtomico(arquivoDaUnidade(), []byte(eq.Unidade(oPipewire.nome(), arquivoDoConf()))); err != nil {
			return base.Falha(base.Erro, "não consegui escrever a unidade do equalizador: %v", err)
		}
		if err := systemctl("daemon-reload"); err != nil {
			return base.Falha(base.Erro, "o equalizador não ligou: %v", err)
		}
		if err := systemctl("enable", "--now", eq.NomeDaUnidade); err != nil {
			desfaz(st)
			return base.Falha(base.Erro, "o equalizador não ligou: %v", err)
		}
		if g, sink, err = esperaOSink(); err != nil {
			desfaz(st)
			return base.Falha(base.Erro, "o equalizador não ligou: %v", err)
		}
	}
	if _, err := oWpctl.roda("set-default", strconv.Itoa(sink.ID)); err != nil {
		desfaz(st)
		return base.Falha(base.Erro, "o equalizador subiu, mas não virou a saída padrão (%v); desliguei de novo", err)
	}
	st.Ligado = true
	if err := gravaEstado(st); err != nil {
		return base.Falha(base.Erro, "não consegui gravar o estado: %v", err)
	}
	// O pedido foi aceito; se o WirePlumber não trocou a padrão (outra regra
	// dele manda), o equalizador fica no ar e a resposta diz a verdade.
	g, virou := esperaVirarPadrao(g)
	aviso := ""
	if !virou {
		aviso = "o equalizador está no ar, mas o WirePlumber não o tornou a saída padrão"
	}
	_ = base.Escreve(ligadoJSON{Ok: true, Ligado: true, NaSaida: virou, SaidaAnterior: anteriorDe(st.SaidaAnterior, g), Aviso: ouNulo(aviso)})
	return base.Certo
}

func eqDesligar() int {
	st := leEstado()
	var aviso string
	g, err := leSom()
	switch {
	case err != nil:
		aviso = err.Error()
	case g.SaidaPadrao == eq.NoDoSink:
		// Devolve a saída de antes, se ela ainda existe. Se sumiu, o
		// WirePlumber escolhe: o motor não inventa.
		if anterior, ok := g.Saida(st.SaidaAnterior); ok {
			if _, err := oWpctl.roda("set-default", strconv.Itoa(anterior.ID)); err != nil {
				aviso = "não consegui devolver a saída de antes: " + err.Error()
			}
		}
	}
	if err := systemctl("disable", "--now", eq.NomeDaUnidade); err != nil {
		return base.Falha(base.Erro, "o equalizador não desligou: %v", err)
	}
	st.Ligado = false
	if err := gravaEstado(st); err != nil {
		return base.Falha(base.Erro, "não consegui gravar o estado: %v", err)
	}
	_ = base.Escreve(ligadoJSON{Ok: true, SaidaAnterior: anteriorDe(st.SaidaAnterior, g), Aviso: ouNulo(aviso)})
	return base.Certo
}

// aplica troca os valores do equalizador: ao vivo se o sink está no ar (só o
// que mudou, um pw-cli por banda), e sempre no estado. `nome` é a
// predefinição que está sendo aplicada ("" = descobre pela lista).
func aplica(novo eq.Ajuste, nome string) (aoVivo bool, aplicados []string, st guardado, err error) {
	st = leEstado()
	aplicados = []string{}
	if g, erroDoSom := leSom(); erroDoSom == nil {
		if sink, ok := g.Saida(eq.NoDoSink); ok {
			for _, m := range eq.Mudancas(st.ajuste(), novo) {
				saida, err := oPwCli.roda("set-param", strconv.Itoa(sink.ID), "Props", m.Props())
				if err == nil && strings.HasPrefix(strings.TrimSpace(string(saida)), "Error") {
					err = errors.New("o pw-cli falhou: " + primeiraLinha(saida))
				}
				if err != nil {
					return false, nil, st, err
				}
				aplicados = append(aplicados, m.Nomes()...)
			}
			aoVivo = true
		}
	}
	st.Ganhos, st.Pre = novo.Ganhos, novo.Pre
	if nome == "" {
		nome = eq.QualE(eq.Todas(lePredefinicoes()), novo)
	}
	st.Predefinicao = nome
	if err := gravaEstado(st); err != nil {
		return aoVivo, aplicados, st, err
	}
	// O conf fica pronto para a próxima subida (só se o equalizador já foi
	// ligado alguma vez: antes disso não há o que regravar).
	if aoVivo || existe(arquivoDoConf()) {
		if err := gravaConf(novo, st.SaidaAnterior); err != nil {
			return aoVivo, aplicados, st, err
		}
	}
	return aoVivo, aplicados, st, nil
}

func eqAjustar() int {
	bruto, err := io.ReadAll(io.LimitReader(entrada, 64<<10))
	if err != nil {
		return base.Falha(base.Erro, "não consegui ler o pedido")
	}
	var pedido struct {
		Ganhos []float64 `json:"ganhos"`
		Pre    *float64  `json:"pre"`
	}
	if err := json.Unmarshal(bruto, &pedido); err != nil {
		return base.Falha(base.Erro, "o pedido não é JSON: {\"ganhos\": [10 números], \"pre\": número}")
	}
	if pedido.Pre == nil {
		return base.Falha(base.Erro, "falta o pré-ganho no pedido")
	}
	novo := eq.Ajuste{Ganhos: pedido.Ganhos, Pre: *pedido.Pre}
	if err := novo.Valida(); err != nil {
		return base.Falha(base.Erro, "%v", err)
	}
	aoVivo, aplicados, st, err := aplica(novo, "")
	if err != nil {
		return base.Falha(base.Erro, "%v", err)
	}
	_ = base.Escreve(map[string]any{"ok": true, "ao_vivo": aoVivo, "aplicados": aplicados, "predefinicao": ouNulo(st.Predefinicao)})
	return base.Certo
}
