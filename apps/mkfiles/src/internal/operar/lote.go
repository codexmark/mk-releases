package operar

import (
	"errors"
	"path/filepath"
	"strconv"

	"mkfiles/internal/base"
)

// O renomear em lote: a janela faz a conta dos nomes novos e manda os pares
// [de, nome novo]; aqui tudo é conferido antes e feito como uma tarefa só, que
// ou renomeia todos ou deixa tudo como estava.

func init() {
	Registrar("renomear-lote", func(p *base.Pedido) (Trabalho, error) {
		var q struct {
			Pares [][]string `json:"pares"`
		}
		if err := p.Ler(&q); err != nil {
			return nil, errors.New("pedido de renomear em lote mal formado")
		}
		pares, err := confereLote(q.Pares)
		if err != nil {
			return nil, err
		}
		return &loteDeNomes{pares: pares}, nil
	})
}

// confereLote valida o lote inteiro antes de qualquer renomeação e devolve os
// pares que mudam alguma coisa (de → caminho novo). Os conflitos (nome
// inválido, repetido no lote, ou já usado na pasta por quem não sai do lugar)
// voltam todos de uma vez, num *ComCampos.
func confereLote(pedidos [][]string) ([][2]string, error) {
	if len(pedidos) == 0 {
		return nil, errors.New("nenhum item para renomear")
	}
	origens := map[string]bool{}
	for _, par := range pedidos {
		if len(par) != 2 {
			return nil, errors.New("cada par é [caminho, nome novo]")
		}
		if err := base.Caminho(par[0]); err != nil {
			return nil, err
		}
		if origens[par[0]] {
			return nil, errors.New("o mesmo item aparece duas vezes: «" + par[0] + "»")
		}
		origens[par[0]] = true
	}
	for _, par := range pedidos {
		if err := base.Protegido(par[0]); err != nil {
			return nil, &Recusa{err.Error()}
		}
		if !existe(par[0]) {
			return nil, errors.New("isso não existe mais: «" + par[0] + "»")
		}
	}
	conflitos := []base.Campos{}
	conflito := func(par []string, motivo string) {
		conflitos = append(conflitos, base.Campos{"de": par[0], "para": par[1], "motivo": motivo})
	}
	// Quem sai do lugar libera o nome; quem fica (ou não está no lote) ocupa.
	saem, chegam := map[string]bool{}, map[string]int{}
	for _, par := range pedidos {
		if base.Nome(par[1]) != nil {
			continue
		}
		para := filepath.Join(filepath.Dir(par[0]), par[1])
		if para != par[0] {
			saem[par[0]] = true
			chegam[para]++
		}
	}
	var pares [][2]string
	for _, par := range pedidos {
		if base.Nome(par[1]) != nil {
			conflito(par, "invalido")
			continue
		}
		para := filepath.Join(filepath.Dir(par[0]), par[1])
		switch {
		case para == par[0]: // não muda
		case chegam[para] > 1:
			conflito(par, "repetido")
		case existe(para) && !saem[para]:
			conflito(par, "existe")
		default:
			pares = append(pares, [2]string{par[0], para})
		}
	}
	if len(conflitos) > 0 {
		return nil, &ComCampos{Frase: "há " + strconv.Itoa(len(conflitos)) + " nome(s) em conflito: nada foi renomeado",
			Mais: base.Campos{"conflitos": conflitos}}
	}
	if len(pares) == 0 {
		return nil, errors.New("nenhum nome muda")
	}
	return pares, nil
}

type loteDeNomes struct{ pares [][2]string }

func (l *loteDeNomes) descreve() string {
	if len(l.pares) == 1 {
		return "«" + filepath.Base(l.pares[0][0]) + "»"
	}
	return strconv.Itoa(len(l.pares)) + " itens"
}
func (l *loteDeNomes) Titulo() string    { return "Renomeando " + l.descreve() }
func (l *loteDeNomes) Destino() string   { return filepath.Dir(l.pares[0][1]) }
func (l *loteDeNomes) Descricao() string { return "Renomear " + l.descreve() }
func (l *loteDeNomes) Refaz() Trabalho   { n := *l; return &n }
func (l *loteDeNomes) Prepara(t *Tarefa) { t.Total(len(l.pares), 0) }

func (l *loteDeNomes) Roda(t *Tarefa) {
	if t.renomeiaLote(l.pares) {
		t.fez(feito{tipo: fLote, pares: l.pares})
		for _, par := range l.pares {
			t.Criou(par[1])
		}
	}
}

// renomeiaLote faz todas as renomeações ou nenhuma. Uma cadeia (a→b, b→c) é
// feita na ordem que não pisa em ninguém; um ciclo (a→b, b→a) é aberto com um
// nome temporário oculto na mesma pasta. Se uma renomeação falha, se um
// destino aparece ocupado na hora ou se a tarefa é cancelada, as já feitas são
// desfeitas de trás para a frente. Devolve true se o lote inteiro foi feito.
func (t *Tarefa) renomeiaLote(pares [][2]string) bool {
	// O lote conta inteiro no fim: ou foi todo, ou voltou todo.
	defer t.Avanca(len(pares), 0, "")
	for _, par := range pares {
		if !existe(par[0]) {
			t.Erro(par[0], "não existe mais: nada foi renomeado")
			return false
		}
		if err := base.Protegido(par[0]); err != nil {
			t.Erro(par[0], err.Error())
			return false
		}
	}
	type passo struct{ de, para string }
	var feitos []passo
	volta := func(culpado, motivo string) bool {
		t.Erro(culpado, motivo+": nada foi renomeado")
		for i := len(feitos) - 1; i >= 0; i-- {
			if err := renomeia(feitos[i].para, feitos[i].de); err != nil {
				t.Erro(feitos[i].para, "não voltou ao nome antigo («"+filepath.Base(feitos[i].de)+"»): "+frase(err))
			}
		}
		return false
	}
	faz := func(de, para string) error {
		if existe(para) {
			return errors.New("já existe um item com esse nome")
		}
		if err := renomeia(de, para); err != nil {
			return errors.New(frase(err))
		}
		feitos = append(feitos, passo{de, para})
		return nil
	}
	fila := make([]pendente, len(pares))
	for i, par := range pares {
		fila[i] = pendente{par[0], par[0], par[1]}
	}
	for len(fila) > 0 {
		if !t.Segue() {
			return volta(fila[0].origem, "a tarefa foi cancelada")
		}
		andou := false
		for i, p := range fila {
			if existe(p.para) && ocupadoPorPendente(p.para, fila) {
				continue // o destino ainda é de alguém do lote que não saiu
			}
			if err := faz(p.agora, p.para); err != nil {
				return volta(p.origem, err.Error())
			}
			fila = append(fila[:i:i], fila[i+1:]...)
			andou = true
			break
		}
		if andou {
			continue
		}
		// Só sobrou ciclo: o primeiro sai da frente com um nome temporário.
		tmp := aoLado(fila[0].agora)
		if err := faz(fila[0].agora, tmp); err != nil {
			return volta(fila[0].origem, err.Error())
		}
		fila[0].agora = tmp
	}
	return true
}

// ocupadoPorPendente: o caminho é onde está agora um item do lote que ainda
// vai sair.
func ocupadoPorPendente(caminho string, fila []pendente) bool {
	for _, p := range fila {
		if p.agora == caminho {
			return true
		}
	}
	return false
}

// pendente é um item do lote que ainda não chegou ao nome novo: onde está
// agora (muda quando um ciclo é aberto), de onde saiu e para onde vai.
type pendente struct{ agora, origem, para string }
