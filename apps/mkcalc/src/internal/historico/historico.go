// Package historico guarda as contas confirmadas do MkCalc num arquivo JSON
// (../../../CONTRATO.md, "O histórico"): o mais novo primeiro, com limite de
// itens, escrita atômica, e um arquivo corrompido vale como histórico vazio.
package historico

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// LimitePadrao é quantas contas o histórico guarda.
const LimitePadrao = 200

// Um arquivo maior que isto não é um histórico nosso: nem tenta ler.
const tamanhoMaximo = 8 << 20

// Agora é o relógio. Os testes trocam.
var Agora = time.Now

// Item é uma conta guardada, com os nomes do contrato.
type Item struct {
	ID         int64  `json:"id"`
	Expressao  string `json:"expressao"`
	Resultado  string `json:"resultado"`
	Cru        string `json:"cru"`
	Aproximado bool   `json:"aproximado"`
	QuandoS    int64  `json:"quando_s"`
}

type arquivo struct {
	Versao int    `json:"versao"`
	Itens  []Item `json:"itens"`
}

// ErroSemItem é o apagar de uma conta que não está lá.
var ErroSemItem = errors.New("essa conta não está mais no histórico")

// Caminho é onde o histórico mora:
// ${XDG_STATE_HOME:-~/.local/state}/mk/mkcalc/historico.json. Vazio quando
// não dá para saber (sem XDG_STATE_HOME e sem home).
func Caminho() string {
	estado := os.Getenv("XDG_STATE_HOME")
	if !filepath.IsAbs(estado) {
		casa, err := os.UserHomeDir()
		if err != nil || !filepath.IsAbs(casa) {
			return ""
		}
		estado = filepath.Join(casa, ".local", "state")
	}
	return filepath.Join(estado, "mk", "mkcalc", "historico.json")
}

// Limite é quantas contas ficam (MKCALC_HISTORICO_LIMITE, padrão 200).
func Limite() int {
	n, err := strconv.Atoi(strings.TrimSpace(os.Getenv("MKCALC_HISTORICO_LIMITE")))
	if err != nil || n <= 0 {
		return LimitePadrao
	}
	return n
}

// Ler devolve o histórico, o mais novo primeiro. Arquivo que não existe, que
// não é JSON ou que é de outra versão: histórico vazio, nunca erro.
func Ler() []Item {
	itens := []Item{}
	caminho := Caminho()
	if caminho == "" {
		return itens
	}
	if info, err := os.Stat(caminho); err != nil || !info.Mode().IsRegular() || info.Size() > tamanhoMaximo {
		return itens
	}
	dados, err := os.ReadFile(caminho)
	if err != nil {
		return itens
	}
	var a arquivo
	if json.Unmarshal(dados, &a) != nil || a.Versao != 1 {
		return itens
	}
	vistos := map[int64]bool{}
	for _, it := range a.Itens {
		// Só o que tem cara de conta nossa, e cada id uma vez.
		if it.ID <= 0 || vistos[it.ID] || it.Expressao == "" || it.Resultado == "" || it.Cru == "" {
			continue
		}
		vistos[it.ID] = true
		itens = append(itens, it)
	}
	return aparar(itens)
}

func aparar(itens []Item) []Item {
	if limite := Limite(); len(itens) > limite {
		itens = itens[:limite]
	}
	return itens
}

// Guardar põe uma conta na frente do histórico e o devolve como ficou.
// Confirmar de novo a conta que já é a mais nova só atualiza a hora.
func Guardar(expressao, resultado, cru string, aproximado bool) ([]Item, error) {
	itens := Ler()
	agora := Agora().Unix()
	if len(itens) > 0 && itens[0].Expressao == expressao && itens[0].Cru == cru {
		itens[0].QuandoS = agora
	} else {
		var maior int64
		for _, it := range itens {
			maior = max(maior, it.ID)
		}
		novo := Item{ID: maior + 1, Expressao: expressao, Resultado: resultado, Cru: cru, Aproximado: aproximado, QuandoS: agora}
		itens = aparar(append([]Item{novo}, itens...))
	}
	return itens, gravar(itens)
}

// Apagar tira uma conta. ErroSemItem se ela não está lá.
func Apagar(id int64) ([]Item, error) {
	itens := Ler()
	for i, it := range itens {
		if it.ID == id {
			itens = append(itens[:i], itens[i+1:]...)
			return itens, gravar(itens)
		}
	}
	return itens, ErroSemItem
}

// Limpar esvazia o histórico.
func Limpar() ([]Item, error) {
	itens := []Item{}
	return itens, gravar(itens)
}

// gravar escreve o arquivo inteiro de uma vez: um temporário na mesma pasta
// e um rename por cima, para quem lê nunca pegar um arquivo pela metade.
func gravar(itens []Item) error {
	caminho := Caminho()
	if caminho == "" {
		return errors.New("não achei a pasta de estado (XDG_STATE_HOME ou a home)")
	}
	dados, err := json.MarshalIndent(arquivo{Versao: 1, Itens: itens}, "", "  ")
	if err != nil {
		return err
	}
	pasta := filepath.Dir(caminho)
	if err := os.MkdirAll(pasta, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(pasta, ".historico-*.tmp")
	if err != nil {
		return err
	}
	_, err = tmp.Write(append(dados, '\n'))
	if err == nil {
		err = tmp.Sync()
	}
	if fechou := tmp.Close(); err == nil {
		err = fechou
	}
	if err == nil {
		err = os.Rename(tmp.Name(), caminho)
	}
	if err != nil {
		os.Remove(tmp.Name())
	}
	return err
}
