// SPDX-License-Identifier: GPL-3.0-or-later

// Pacote daemon: o mktunesd visto pela janela. Cada área do app (o player, a
// busca, o Início, a biblioteca…) mantém no quadro o seu estado, como uma
// mensagem JSON com "type", e responde aos comandos que são dela; o servidor
// do socket entrega o quadro a cada janela que conecta. Porte do
// desktop/daemon do MkTunes em Kotlin, com um laço único no lugar das
// corrotinas. O protocolo está em CONTRATO.md.
package daemon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"mktunes/internal/dominio"
)

// msg é uma mensagem do protocolo antes de virar JSON.
type msg = map[string]any

// codifica escreve o JSON de uma linha (sem trocar <, > e & por escapes).
func codifica(valor any) string {
	var saida bytes.Buffer
	codificador := json.NewEncoder(&saida)
	codificador.SetEscapeHTML(false)
	_ = codificador.Encode(valor)
	return strings.TrimRight(saida.String(), "\n")
}

// relogio escreve uma duração como "4:35".
func relogio(d time.Duration) string {
	segundos := max(int64(d/time.Second), 0)
	return fmt.Sprintf("%d:%02d", segundos/60, segundos%60)
}

// faixaNoFio é a faixa como viaja até a janela e volta: inteira, para a janela
// devolver qualquer lista para tocar, curtir ou baixar sem o daemon lembrar do
// que mostrou.
func faixaNoFio(f dominio.Faixa) msg {
	duracao := ""
	switch {
	case f.AoVivo():
		duracao = "AO VIVO"
	case f.Duracao > 0:
		duracao = relogio(f.Duracao)
	}
	m := msg{"id": f.ID, "provider": string(f.Provedor), "title": f.Titulo, "artist": f.Artista, "durationMs": f.Duracao.Milliseconds(), "length": duracao, "artwork": f.Capa}
	if f.Stream != "" {
		m["streamUrl"] = f.Stream
	}
	return m
}

func faixasNoFio(faixas []dominio.Faixa) []msg {
	lista := make([]msg, 0, len(faixas))
	for _, f := range faixas {
		lista = append(lista, faixaNoFio(f))
	}
	return lista
}

// args são os campos de um comando da janela.
type args map[string]any

// texto só aceita um texto de verdade (não um número, nem nulo).
func (a args) texto(chave string) (string, bool) {
	valor, ok := a[chave].(string)
	return valor, ok
}

func (a args) textoOu(chave, reserva string) string {
	if valor, ok := a.texto(chave); ok {
		return valor
	}
	return reserva
}

func (a args) real(chave string) (float64, bool) {
	switch valor := a[chave].(type) {
	case float64:
		return valor, !math.IsNaN(valor) && !math.IsInf(valor, 0)
	case json.Number:
		numero, erro := valor.Float64()
		return numero, erro == nil && !math.IsNaN(numero) && !math.IsInf(numero, 0)
	case string:
		numero, erro := strconv.ParseFloat(valor, 64)
		return numero, erro == nil && !math.IsNaN(numero) && !math.IsInf(numero, 0)
	}
	return 0, false
}

// inteiro aceita um número sem parte fracionária que caiba num int de 32 bits.
func (a args) inteiro(chave string) (int, bool) {
	numero, ok := a.real(chave)
	if !ok || numero != math.Trunc(numero) || math.Abs(numero) > math.MaxInt32 {
		return 0, false
	}
	return int(numero), true
}

func (a args) bool(chave string) (valor, tem bool) {
	valor, tem = a[chave].(bool)
	return valor, tem
}

// faixaDoFio lê uma faixa que a janela devolveu; false para o que não é uma
// faixa que este daemon mandou.
func faixaDoFio(valor any) (dominio.Faixa, bool) {
	objeto, ok := valor.(map[string]any)
	if !ok {
		return dominio.Faixa{}, false
	}
	a := args(objeto)
	id, _ := a.texto("id")
	provedor, conhecido := dominio.ProvedorDe(a.textoOu("provider", ""))
	if strings.TrimSpace(id) == "" || !conhecido {
		return dominio.Faixa{}, false
	}
	f := dominio.Faixa{ID: id, Provedor: provedor, Titulo: a.textoOu("title", ""), Artista: a.textoOu("artist", "")}
	if ms, ok := a.real("durationMs"); ok && ms > 0 && ms < 1e12 {
		f.Duracao = time.Duration(ms) * time.Millisecond
	}
	if capa := a.textoOu("artwork", ""); strings.TrimSpace(capa) != "" {
		f.Capa = capa
	}
	if stream := a.textoOu("streamUrl", ""); strings.TrimSpace(stream) != "" {
		f.Stream = stream
	}
	return f, true
}

func faixasDoFio(valor any) []dominio.Faixa {
	lista, _ := valor.([]any)
	faixas := make([]dominio.Faixa, 0, len(lista))
	for _, item := range lista {
		if f, ok := faixaDoFio(item); ok {
			faixas = append(faixas, f)
		}
	}
	return faixas
}

func tipoNoFio(t dominio.TipoColecao) string {
	if t == dominio.Album {
		return "album"
	}
	return "playlist"
}

// itemNoFio escreve um cartão do feed dizendo o que ele é.
func itemNoFio(i dominio.Item) msg {
	switch i.Tipo {
	case dominio.ItemMusica:
		m := faixaNoFio(i.Faixa)
		m["kind"], m["subtitle"] = "song", i.Faixa.Artista
		return m
	case dominio.ItemColecao:
		return msg{"kind": tipoNoFio(i.Colecao), "id": i.ID, "title": i.Titulo, "subtitle": i.Subtitulo, "artwork": i.Capa, "playlistId": i.PlaylistID}
	}
	return msg{"kind": "artist", "id": i.ID, "title": i.Titulo, "subtitle": i.Subtitulo, "artwork": i.Capa}
}

func itensNoFio(itens []dominio.Item) []msg {
	lista := make([]msg, 0, len(itens))
	for _, i := range itens {
		lista = append(lista, itemNoFio(i))
	}
	return lista
}

func prateleirasNoFio(prateleiras []dominio.Prateleira) []msg {
	lista := make([]msg, 0, len(prateleiras))
	for _, p := range prateleiras {
		lista = append(lista, msg{"title": p.Titulo, "items": itensNoFio(p.Itens)})
	}
	return lista
}

// downloadNoFio leva o estado de um download e o progresso em por cento.
func downloadNoFio(d dominio.Download) msg {
	m := msg{"track": faixaNoFio(d.Faixa), "status": string(d.Estado)}
	switch d.Estado {
	case dominio.Baixando:
		m["percent"] = int(d.Progresso * 100)
	case dominio.Concluido:
		m["path"] = d.Caminho
	case dominio.Falhou:
		m["reason"] = d.Motivo
	}
	return m
}
