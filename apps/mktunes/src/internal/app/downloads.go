// SPDX-License-Identifier: GPL-3.0-or-later

package app

import (
	"context"
	"errors"
	"net/url"
	"sort"
	"time"

	"mktunes/internal/dominio"
	"mktunes/internal/laco"
)

const (
	extensaoDoDownload = "m4a"
	motivoInterrompido = "Interrompido"
	motivoIndisponivel = "Faixa indisponível para download"
	motivoFalha        = "Falha no download"
	motivoNaoSalvou    = "Não foi possível salvar a música"
)

// Downloads é a fila de downloads: poucos de cada vez, com progresso,
// cancelar, tentar de novo e um histórico que sobrevive a reinícios.
type Downloads struct {
	resolvedor Resolvedor
	buscador   Buscador
	guarda     Guarda
	laco       laco.Laco
	// trabalhadores é lido a cada início, para o ajuste valer no próximo download.
	trabalhadores func() int
	// Mudou avisa a cada mudança da lista (inclusive de progresso).
	Mudou Sinal

	lista   []dominio.Download // o mais novo primeiro
	rodando map[string]laco.Tarefa
}

// NovosDownloads cria a fila vazia; Restaura traz o histórico.
func NovosDownloads(r Resolvedor, b Buscador, g Guarda, l laco.Laco, trabalhadores func() int) *Downloads {
	if trabalhadores == nil {
		trabalhadores = func() int { return 1 }
	}
	return &Downloads{resolvedor: r, buscador: b, guarda: g, laco: l, trabalhadores: trabalhadores, rodando: map[string]laco.Tarefa{}}
}

// Lista é uma cópia da fila, o mais novo primeiro.
func (d *Downloads) Lista() []dominio.Download { return append([]dominio.Download(nil), d.lista...) }

// Restaura carrega o histórico; o que estava no meio quando o daemon fechou
// vira falha.
func (d *Downloads) Restaura() {
	aqui := map[string]bool{}
	for _, j := range d.lista {
		aqui[j.Faixa.ID] = true
	}
	for _, j := range d.guarda.Downloads() {
		if aqui[j.Faixa.ID] {
			continue // um download começado agora é mais novo que o guardado
		}
		if j.Ativo() {
			j.Estado, j.Motivo, j.Progresso = dominio.Falhou, motivoInterrompido, 0
			d.guarda.SalvaDownload(j)
		}
		d.lista = append(d.lista, j)
	}
	sort.SliceStable(d.lista, func(i, k int) bool { return d.lista[i].CriadoEm.After(d.lista[k].CriadoEm) })
	d.Mudou.Avisa()
}

// Enfileira pede o download das faixas, na ordem da lista.
func (d *Downloads) Enfileira(faixas ...dominio.Faixa) {
	agora := d.laco.Agora()
	n := 0
	for _, f := range faixas {
		if j, tem := d.acha(f.ID); f.AoVivo() || (tem && j.Ativo()) || d.CaminhoLocal(f.ID) != "" {
			continue
		}
		// Um milissegundo de diferença, na ordem: um álbum volta na sua ordem
		// depois de reiniciar.
		d.poe(dominio.Download{Faixa: f, Estado: dominio.NaFila, CriadoEm: agora.Add(time.Duration(n) * time.Millisecond)})
		n++
	}
	d.bombeia()
}

// TentaDeNovo recoloca na fila um download que falhou ou foi cancelado.
func (d *Downloads) TentaDeNovo(id string) {
	j, tem := d.acha(id)
	if !tem || (j.Estado != dominio.Falhou && j.Estado != dominio.Cancelado) {
		return
	}
	j.Estado, j.Motivo, j.Progresso = dominio.NaFila, "", 0
	d.poe(j)
	d.bombeia()
}

// Cancela para um download em andamento ou tira um da fila de espera.
func (d *Downloads) Cancela(id string) {
	if tarefa, roda := d.rodando[id]; roda {
		tarefa.Cancela()
		delete(d.rodando, id)
		d.estado(id, dominio.Download{Estado: dominio.Cancelado}, true)
		d.bombeia()
		return
	}
	if j, tem := d.acha(id); tem && j.Estado == dominio.NaFila {
		d.estado(id, dominio.Download{Estado: dominio.Cancelado}, true)
	}
}

// Remove tira o download do histórico e apaga o arquivo, se já baixou.
func (d *Downloads) Remove(id string) {
	if tarefa, roda := d.rodando[id]; roda {
		tarefa.Cancela()
		delete(d.rodando, id)
	}
	if j, tem := d.acha(id); tem && j.Estado == dominio.Concluido {
		d.buscador.Apaga(j.Caminho)
	}
	resto := d.lista[:0:0]
	for _, j := range d.lista {
		if j.Faixa.ID != id {
			resto = append(resto, j)
		}
	}
	d.lista = resto
	d.guarda.RemoveDownload(id)
	d.Mudou.Avisa()
	d.bombeia()
}

// CaminhoLocal é o arquivo de um download concluído, se ainda está no disco.
func (d *Downloads) CaminhoLocal(id string) string {
	if j, tem := d.acha(id); tem && j.Estado == dominio.Concluido && d.buscador.Existe(j.Caminho) {
		return j.Caminho
	}
	return ""
}

// StreamLocal toca a cópia baixada quando há uma (é o Local da sessão).
func (d *Downloads) StreamLocal(f dominio.Faixa) (Stream, bool) {
	caminho := d.CaminhoLocal(f.ID)
	if caminho == "" {
		return Stream{}, false
	}
	return Stream{URL: (&url.URL{Scheme: "file", Path: caminho}).String(), Volume: 1}, true
}

func (d *Downloads) bombeia() {
	for len(d.rodando) < d.trabalhadores() {
		// O mais antigo da espera primeiro (a lista tem o mais novo na frente).
		achou := false
		for i := len(d.lista) - 1; i >= 0; i-- {
			if d.lista[i].Estado == dominio.NaFila {
				d.comeca(d.lista[i].Faixa)
				achou = true
				break
			}
		}
		if !achou {
			return
		}
	}
}

func (d *Downloads) comeca(f dominio.Faixa) {
	id := f.ID
	d.estado(id, dominio.Download{Estado: dominio.Baixando}, true)
	var tarefa laco.Tarefa
	tarefa = d.laco.Vai(func(ctx context.Context) func() {
		fim := dominio.Download{Estado: dominio.Concluido, Progresso: 1}
		stream, erro := d.resolvedor.Resolve(ctx, f)
		if erro == nil {
			fim.Caminho, erro = d.buscador.Busca(ctx, stream.URL, dominio.CaminhoDoDownload(f, extensaoDoDownload), func(p float64) {
				d.laco.Posta(func() {
					// Um progresso atrasado não ressuscita um download já encerrado.
					if d.rodando[id] == tarefa {
						d.estado(id, dominio.Download{Estado: dominio.Baixando, Progresso: p}, false)
					}
				})
			})
		} else if !errors.Is(erro, ErrStream) {
			erro = errors.Join(ErrStream, erro)
		}
		var falha *ErroDownload
		switch {
		case erro == nil:
		case errors.Is(erro, ErrStream):
			fim = dominio.Download{Estado: dominio.Falhou, Motivo: motivoIndisponivel}
		case errors.As(erro, &falha):
			motivo := falha.Motivo
			if motivo == "" {
				motivo = motivoFalha
			}
			fim = dominio.Download{Estado: dominio.Falhou, Motivo: motivo}
		default:
			// Disco cheio e companhia: uma falha que o ouvinte pode tentar de novo.
			fim = dominio.Download{Estado: dominio.Falhou, Motivo: motivoNaoSalvou}
		}
		return func() {
			delete(d.rodando, id)
			d.estado(id, fim, true)
			d.bombeia()
		}
	})
	d.rodando[id] = tarefa
}

func (d *Downloads) acha(id string) (dominio.Download, bool) {
	for _, j := range d.lista {
		if j.Faixa.ID == id {
			return j, true
		}
	}
	return dominio.Download{}, false
}

func (d *Downloads) poe(j dominio.Download) {
	nova := []dominio.Download{j}
	for _, velho := range d.lista {
		if velho.Faixa.ID != j.Faixa.ID {
			nova = append(nova, velho)
		}
	}
	d.lista = nova
	d.guarda.SalvaDownload(j)
	d.Mudou.Avisa()
}

// estado troca o estado de um download (os campos de novo, menos a faixa e a data).
func (d *Downloads) estado(id string, novo dominio.Download, guarda bool) {
	for i := range d.lista {
		if d.lista[i].Faixa.ID != id {
			continue
		}
		novo.Faixa, novo.CriadoEm = d.lista[i].Faixa, d.lista[i].CriadoEm
		d.lista[i] = novo
		if guarda {
			d.guarda.SalvaDownload(novo)
		}
		d.Mudou.Avisa()
		return
	}
}
