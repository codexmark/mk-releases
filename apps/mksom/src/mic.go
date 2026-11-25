package main

import (
	"fmt"
	"regexp"
	"strconv"

	"mksom/internal/base"
	"mksom/internal/pw"
)

func init() {
	registrar("mic", cmdMic)
}

const usoDoMic = "uso: mksom-motor mic <listar | padrao <id> | volume <id> <0-150> | mudo <id> <sim|nao> | nivel <id> | gravar <id> | ouvir | limpar>"

var (
	soNumero  = regexp.MustCompile(`^[0-9]{1,7}$`)
	soVolume  = regexp.MustCompile(`^[0-9]{1,3}$`)
	volumeMax = 150
)

func cmdMic(args []string) int {
	if len(args) == 0 {
		return base.Falha(base.Erro, usoDoMic)
	}
	resto, err := posicionais(args[1:])
	if err != nil {
		return base.Falha(base.Erro, "%v", err)
	}
	quantos := map[string]int{"listar": 0, "padrao": 1, "volume": 2, "mudo": 2, "nivel": 1, "gravar": 1, "ouvir": 0, "limpar": 0}
	if n, ok := quantos[args[0]]; !ok || len(resto) != n {
		return base.Falha(base.Erro, usoDoMic)
	}
	switch args[0] {
	case "listar":
		return micListar()
	case "padrao":
		return micPadrao(resto[0])
	case "volume":
		return micVolume(resto[0], resto[1])
	case "mudo":
		return micMudo(resto[0], resto[1])
	case "nivel":
		return micNivel(resto[0])
	case "gravar":
		return micGravar(resto[0])
	case "ouvir":
		return micOuvir()
	}
	return micLimpar()
}

// entradaDe confere que o texto é um id (só algarismos) e que esse id é de
// uma entrada de áudio que o PipeWire tem agora. Devolve o nó, ou o código
// de saída depois de já ter escrito o erro.
func entradaDe(texto string) (pw.No, int) {
	if !soNumero.MatchString(texto) {
		return pw.No{}, base.Falha(base.Erro, "o id da entrada tem de ser um número")
	}
	id, _ := strconv.Atoi(texto)
	g, err := leSom()
	if err != nil {
		return pw.No{}, base.Falha(base.Erro, "%v", err)
	}
	no, ok := g.Entrada(id)
	if !ok {
		return pw.No{}, base.Falha(base.Recusado, "essa entrada não existe mais")
	}
	return no, base.Certo
}

func micListar() int {
	g, err := leSom()
	if err != nil {
		return base.Falha(base.Erro, "%v", err)
	}
	_ = base.Escreve(map[string]any{"entradas": entradasDe(g)})
	return base.Certo
}

// wpctl roda um ajuste do WirePlumber numa entrada já conferida.
func wpctl(args ...string) int {
	if _, err := oWpctl.roda(args...); err != nil {
		return base.Falha(base.Erro, "%v", err)
	}
	_ = base.Escreve(map[string]bool{"ok": true})
	return base.Certo
}

func micPadrao(id string) int {
	no, codigo := entradaDe(id)
	if codigo != base.Certo {
		return codigo
	}
	return wpctl("set-default", strconv.Itoa(no.ID))
}

func micVolume(id, volume string) int {
	// O volume é conferido antes de qualquer conversa com o PipeWire.
	pct := -1
	if soVolume.MatchString(volume) {
		pct, _ = strconv.Atoi(volume)
	}
	if pct < 0 || pct > volumeMax {
		return base.Falha(base.Erro, "o volume vai de 0 a %d", volumeMax)
	}
	no, codigo := entradaDe(id)
	if codigo != base.Certo {
		return codigo
	}
	return wpctl("set-volume", strconv.Itoa(no.ID), fmt.Sprintf("%d.%02d", pct/100, pct%100))
}

func micMudo(id, mudo string) int {
	valor := map[string]string{"sim": "1", "nao": "0"}[mudo]
	if valor == "" {
		return base.Falha(base.Erro, "o mudo é sim ou nao")
	}
	no, codigo := entradaDe(id)
	if codigo != base.Certo {
		return codigo
	}
	return wpctl("set-mute", strconv.Itoa(no.ID), valor)
}
