// SPDX-License-Identifier: GPL-3.0-or-later

package dominio

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// PlaylistLocal é uma playlist feita neste computador: cada faixa uma vez só.
type PlaylistLocal struct {
	ID     string
	Titulo string
	Faixas []Faixa
}

// Contem diz se a faixa já está na playlist.
func (p PlaylistLocal) Contem(id string) bool {
	for _, f := range p.Faixas {
		if f.ID == id {
			return true
		}
	}
	return false
}

// Adiciona põe a faixa no fim, se ainda não está.
func (p PlaylistLocal) Adiciona(f Faixa) PlaylistLocal {
	if p.Contem(f.ID) {
		return p
	}
	p.Faixas = append(copia(p.Faixas), f)
	return p
}

// Remove tira a faixa da posição dada.
func (p PlaylistLocal) Remove(indice int) PlaylistLocal {
	if indice < 0 || indice >= len(p.Faixas) {
		return p
	}
	novo := copia(p.Faixas[:indice])
	p.Faixas = append(novo, p.Faixas[indice+1:]...)
	return p
}

// MoveFaixa troca uma faixa de lugar.
func (p PlaylistLocal) MoveFaixa(de, para int) PlaylistLocal {
	if de < 0 || para < 0 || de >= len(p.Faixas) || para >= len(p.Faixas) {
		return p
	}
	p.Faixas = Move(p.Faixas, de, para)
	return p
}

// Renomeia troca o título; um título em branco não muda nada.
func (p PlaylistLocal) Renomeia(titulo string) PlaylistLocal {
	if limpo, ok := TituloDePlaylist(titulo); ok {
		p.Titulo = limpo
	}
	return p
}

// Capas dá até quatro capas diferentes, para a colagem.
func (p PlaylistLocal) Capas() []string {
	var capas []string
	visto := map[string]bool{}
	for _, f := range p.Faixas {
		if f.Capa == "" || visto[f.Capa] {
			continue
		}
		visto[f.Capa] = true
		if capas = append(capas, f.Capa); len(capas) == 4 {
			break
		}
	}
	return capas
}

// TamanhoMaxTitulo é o maior título de playlist.
const TamanhoMaxTitulo = 100

// TituloDePlaylist limpa um título: sem espaços nas pontas, obrigatório e curto.
func TituloDePlaylist(bruto string) (string, bool) {
	letras := []rune(strings.TrimSpace(bruto))
	if len(letras) > TamanhoMaxTitulo {
		letras = letras[:TamanhoMaxTitulo]
	}
	return string(letras), len(letras) > 0
}

// ColecaoSalva é um álbum ou playlist do catálogo guardado na biblioteca: um
// marcador, reaberto online.
type ColecaoSalva struct {
	ID      string
	Tipo    TipoColecao
	Titulo  string
	Dono    string // o artista do álbum, ou "Playlist" quando a página não diz
	Capa    string
	SalvaEm time.Time
}

// VersaoBackup é a versão do formato de backup que esta versão lê e escreve.
const VersaoBackup = 1

// Backup é tudo que o ouvinte montou, para levar a outra instalação.
type Backup struct {
	CriadoEm  time.Time
	Curtidas  []Faixa
	Playlists []PlaylistLocal
	Tocadas   []Tocada
	Ajustes   Ajustes
	Buscas    []string
	Salvos    []ColecaoSalva
}

// EstadoDownload é em que pé um download está.
type EstadoDownload string

const (
	NaFila    EstadoDownload = "queued"
	Baixando  EstadoDownload = "downloading"
	Concluido EstadoDownload = "completed"
	Falhou    EstadoDownload = "failed"
	Cancelado EstadoDownload = "cancelled"
)

// Download é uma faixa pedida para baixar. Progresso (0 a 1) vale enquanto
// baixa, Caminho quando concluiu e Motivo quando falhou.
type Download struct {
	Faixa     Faixa
	Estado    EstadoDownload
	Progresso float64
	Caminho   string
	Motivo    string
	CriadoEm  time.Time
}

// Ativo: na fila ou baixando.
func (d Download) Ativo() bool { return d.Estado == NaFila || d.Estado == Baixando }

var inseguro = regexp.MustCompile(`[/\\:*?"<>|\p{Cc}]`)

const artistaDesconhecido = "Artista desconhecido"

// CaminhoDoDownload é onde uma faixa baixada fica, dentro da pasta de música:
// uma pasta por artista, "Título [id].ext", cada parte com até 120 bytes.
func CaminhoDoDownload(f Faixa, extensao string) string {
	pasta := limpaParte(f.Artista)
	if pasta == "" {
		pasta = artistaDesconhecido
	}
	return pasta + "/" + limpaParte(f.Titulo) + " [" + inseguro.ReplaceAllString(f.ID, "_") + "]." + extensao
}

func limpaParte(parte string) string {
	seguro := strings.TrimSpace(cabeEm(strings.TrimSpace(inseguro.ReplaceAllString(parte, "_")), 120))
	// "." e ".." sairiam da pasta do artista; um ponto na frente esconde a pasta.
	if strings.HasPrefix(seguro, ".") {
		return "_" + strings.TrimLeft(seguro, ".")
	}
	return seguro
}

// cabeEm corta letras inteiras (emoji inclusive) para o texto caber no limite.
func cabeEm(texto string, limite int) string {
	fim := 0
	for fim < len(texto) {
		_, tamanho := utf8.DecodeRuneInString(texto[fim:])
		if fim+tamanho > limite {
			break
		}
		fim += tamanho
	}
	return texto[:fim]
}

// Verso é uma linha da letra e quando ela é cantada.
type Verso struct {
	Tempo time.Duration
	Texto string
}

// Letra é a letra sincronizada, em ordem de tempo.
type Letra []Verso

// VersoAtivo é o índice do verso cantado na posição, ou -1 antes do primeiro.
func (l Letra) VersoAtivo(posicao time.Duration) int {
	ativo := -1
	for i, v := range l {
		if v.Tempo <= posicao {
			ativo = i
		}
	}
	return ativo
}

var carimbo = regexp.MustCompile(`\[(\d{1,3}):(\d{2})(?:[.:](\d{1,3}))?\]`)

// LeLRC lê o formato LRC; nil quando nada nele tem tempo.
func LeLRC(texto string) Letra {
	var versos Letra
	for _, linha := range strings.Split(texto, "\n") {
		marcas := carimbo.FindAllStringSubmatch(linha, -1)
		if len(marcas) == 0 {
			continue
		}
		palavras := strings.TrimFunc(carimbo.ReplaceAllString(linha, ""), unicode.IsSpace)
		for _, m := range marcas {
			minutos, _ := strconv.Atoi(m[1])
			segundos, _ := strconv.Atoi(m[2])
			milis, _ := strconv.Atoi((m[3] + "000")[:3])
			tempo := time.Duration(minutos)*time.Minute + time.Duration(segundos)*time.Second + time.Duration(milis)*time.Millisecond
			versos = append(versos, Verso{Tempo: tempo, Texto: palavras})
		}
	}
	sort.SliceStable(versos, func(i, j int) bool { return versos[i].Tempo < versos[j].Tempo })
	return versos
}

// LetraDaFaixa é o que uma fonte de letras achou: Sincronizada, ou só Texto.
type LetraDaFaixa struct {
	Sincronizada Letra
	Texto        string
}
