// SPDX-License-Identifier: GPL-3.0-or-later

package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"math/rand/v2"
	"os"
	"strconv"
	"strings"
	"time"

	"mktunes/internal/app"
	"mktunes/internal/dominio"
	"mktunes/internal/laco"
)

// Portas é o que o daemon precisa do mundo de fora; o main liga o YouTube, o
// mpv e a rede, e os testes ligam dublês.
type Portas struct {
	Catalogo  app.Catalogo
	Inicio    app.FonteDoInicio
	Filas     app.FonteDeFilas
	Navegador app.Navegador
	Paginas   app.BuscaDePaginas
	Sugestoes app.FonteDeSugestoes
	Streams   app.Resolvedor
	Tocador   app.Tocador
	Letras    app.FonteDeLetras
	Radio     app.DiretorioDeRadio
	Arquivos  app.Buscador
	// GuardaDoInicio pode ser nil (sem Início guardado).
	GuardaDoInicio app.GuardaDoInicio
	// Volume põe o volume do ouvinte na saída de áudio, de 0 a 1.
	Volume func(nivel float64)
	// Local procura as músicas que já estão no computador.
	Local func(ctx context.Context) []dominio.Faixa
}

// GuardaDeVolume lembra o volume do ouvinte, de 0 a 100, entre uma subida e outra.
type GuardaDeVolume interface {
	Carrega() (nivel int, tem bool)
	Salva(nivel int)
}

// VolumeEmArquivo guarda o nível como uma linha de texto num arquivo da pasta
// de estado do usuário.
type VolumeEmArquivo struct{ Arquivo string }

func (v VolumeEmArquivo) Carrega() (int, bool) {
	bruto, erro := os.ReadFile(v.Arquivo)
	if erro != nil {
		return 0, false
	}
	nivel, erro := strconv.Atoi(strings.TrimSpace(string(bruto)))
	return nivel, erro == nil
}

func (v VolumeEmArquivo) Salva(nivel int) {
	if indice := strings.LastIndex(v.Arquivo, "/"); indice > 0 {
		_ = os.MkdirAll(v.Arquivo[:indice], 0o700)
	}
	_ = os.WriteFile(v.Arquivo, []byte(strconv.Itoa(nivel)+"\n"), 0o600)
}

// Config é o resto do que o App precisa para nascer.
type Config struct {
	Guarda app.Guarda
	Codec  app.Codec
	Laco   laco.Laco
	// PastaDeBackups é onde "exportar backup" escreve (Documentos).
	PastaDeBackups string
	// PastaDeDownloads é onde as músicas baixadas ficam, para a janela mostrar.
	PastaDeDownloads string
	Versao           string
	Volume           GuardaDeVolume
	// AoMudarAjustes é chamada na subida e a cada mudança das preferências.
	AoMudarAjustes func(a dominio.Ajustes)
	// AoSair é chamada pelo comando "quit" (o lançador troca um daemon velho).
	AoSair func()
	// Registro recebe o que vale anotar no log do daemon.
	Registro func(texto string)
}

// area é uma parte do app como a janela a vê: responde aos comandos que são
// dela (false: o comando é de outra área).
type area interface {
	trata(cmd string, a args) bool
}

// App é o MkTunes sem janela: os casos de uso atrás do quadro e dos comandos.
type App struct {
	Quadro *Quadro
	Sessao *app.Sessao
	Acervo *app.Acervo
	// SalvaSessao guarda a fila e a posição agora (o daemon chama ao sair).
	SalvaSessao func()

	areas  []area
	aoSair func()
}

// Monta liga tudo. As áreas publicam o estado inicial na ordem em que a
// janela as recebe ao conectar.
func Monta(p Portas, c Config) *App {
	if c.Registro == nil {
		c.Registro = func(string) {}
	}
	if c.AoMudarAjustes == nil {
		c.AoMudarAjustes = func(dominio.Ajustes) {}
	}
	quadro := NovoQuadro()
	acervo := app.NovoAcervo(c.Guarda, c.Codec, c.Laco.Agora)
	downloads := app.NovosDownloads(p.Streams, p.Arquivos, c.Guarda, c.Laco, func() int { return acervo.Ajustes().Trabalhadores })
	sessao := app.NovaSessao(app.OpcoesDaSessao{
		Resolvedor: p.Streams, Tocador: p.Tocador, Laco: c.Laco, Autoplay: p.Filas,
		AutoplayLigado: func() bool { return acervo.Ajustes().Autoplay },
		Normalizar:     func() bool { return acervo.Ajustes().Normalizar },
		Local:          downloads.StreamLocal,
		AoFalhar: func(f dominio.Faixa, erro error) {
			c.Registro("sem stream para " + f.ID + ": " + erro.Error())
		},
	})
	soneca := app.NovaSoneca(sessao, c.Laco)
	a := &App{Quadro: quadro, Sessao: sessao, Acervo: acervo, aoSair: c.AoSair}

	// A fila guardada volta antes de a primeira janela conectar.
	a.SalvaSessao = app.GuardaSessao(sessao, c.Guarda, c.Laco)
	app.OuveHistorico(sessao, c.Guarda, c.Laco)
	downloads.Restaura()
	ajustes := acervo.Ajustes()
	p.Tocador.PularSilencio(ajustes.PularSilencio)
	c.AoMudarAjustes(ajustes)

	a.areas = []area{
		novoPlayer(quadro, c.Laco, sessao, acervo, downloads, soneca, p.Filas),
		novaBusca(quadro, c.Laco, acervo, p.Catalogo, p.Paginas, p.Sugestoes),
		novoInicio(quadro, c.Laco, acervo, app.Inicio{Fonte: p.Inicio, Guarda: p.GuardaDoInicio}),
		&paginas{quadro: quadro, laco: c.Laco, acervo: acervo, navegador: p.Navegador, abertas: map[string]dominio.PaginaColecao{}},
		novaBiblioteca(quadro, acervo),
		novaAreaDeDownloads(quadro, c.Laco, downloads, c.PastaDeDownloads),
		novoRadio(quadro, c.Laco, p.Radio),
		novasLetras(quadro, c.Laco, sessao, app.NovasLetras(p.Letras)),
		novosAjustes(quadro, c.Laco, acervo, c, p.Tocador),
		novoVolume(quadro, p.Volume, c.Volume),
		novoLocal(quadro, c.Laco, p.Local),
	}
	return a
}

// Trata recebe uma linha da janela. Uma linha que não é um comando (JSON
// quebrado, sem "cmd", comando desconhecido) não muda nada.
func (a *App) Trata(linha string) {
	var comando map[string]any
	leitor := json.NewDecoder(bytes.NewReader([]byte(linha)))
	leitor.UseNumber()
	if leitor.Decode(&comando) != nil {
		return
	}
	nome, ok := args(comando).texto("cmd")
	if !ok {
		return
	}
	if nome == "quit" {
		if a.aoSair != nil {
			a.aoSair()
		}
		return
	}
	a.Quadro.SoltaGuardados()
	for _, area := range a.areas {
		if area.trata(nome, comando) {
			return
		}
	}
}

// ---- O player ---------------------------------------------------------------

// player publica "now" só quando algo de verdade muda; a posição viaja à parte,
// num "progress" pequeno, para o tique custar uns bytes e não a fila inteira.
type player struct {
	quadro    *Quadro
	laco      laco.Laco
	sessao    *app.Sessao
	acervo    *app.Acervo
	downloads *app.Downloads
	soneca    *app.Soneca
	filas     app.FonteDeFilas
}

const tique = 500 * time.Millisecond

func novoPlayer(q *Quadro, l laco.Laco, s *app.Sessao, a *app.Acervo, d *app.Downloads, soneca *app.Soneca, filas app.FonteDeFilas) *player {
	p := &player{q, l, s, a, d, soneca, filas}
	p.publica()
	for _, sinal := range []*app.Sinal{&s.Mudou, &a.Mudou, &d.Mudou, &soneca.Mudou} {
		sinal.Ouve(p.publica)
	}
	var bate func()
	bate = func() {
		if s.Estado().Situacao == app.Tocando {
			p.progresso()
		}
		l.Depois(tique, bate)
	}
	l.Depois(tique, bate)
	return p
}

func (p *player) publica() {
	estado := p.sessao.Estado()
	faixa, tem := estado.Faixa()
	situacao := "idle"
	if tem {
		situacao = [...]string{"idle", "loading", "playing", "paused", "failed", "failed"}[estado.Situacao]
	}
	baixado, soneca := "", ""
	if tem {
		for _, d := range p.downloads.Lista() {
			if d.Faixa.ID != faixa.ID {
				continue
			}
			switch d.Estado {
			case dominio.NaFila, dominio.Baixando:
				baixado = "downloading"
			case dominio.Concluido:
				baixado = "completed"
			}
		}
	}
	switch estadoDaSoneca, resta := p.soneca.Estado(); estadoDaSoneca {
	case app.SonecaFimDaFaixa:
		soneca = "fim da faixa"
	case app.SonecaContando:
		soneca = relogio(resta)
	}
	agora := msg{
		"status": situacao, "track": msg{}, "title": faixa.Titulo, "artist": faixa.Artista, "artwork": faixa.Capa,
		"live": tem && faixa.AoVivo(), "liked": tem && p.acervo.Curtida(faixa.ID), "download": baixado,
		"upNext": faixasNoFio(estado.Fila.ASeguir()), "repeat": strings.ToLower(string(estado.Fila.Repetir())),
		"stopAfterCurrent": estado.Fila.PararAposAtual(), "speed": p.sessao.Velocidade(), "sleep": soneca,
	}
	if tem {
		agora["track"] = faixaNoFio(faixa)
	}
	p.quadro.Estado("now", agora)
	p.progresso()
}

func (p *player) progresso() {
	progresso := p.sessao.Progresso()
	p.quadro.Estado("progress", msg{"position": float64(progresso.Posicao.Milliseconds()) / 1000, "duration": float64(progresso.Duracao.Milliseconds()) / 1000})
}

func (p *player) trata(cmd string, a args) bool {
	s := p.sessao
	switch cmd {
	case "play":
		faixas := faixasDoFio(a["tracks"])
		indice, tem := a.inteiro("index")
		if !tem {
			indice = 0
		}
		if indice < 0 || indice >= len(faixas) {
			break
		}
		if mistura, _ := a.bool("shuffle"); mistura {
			faixas, indice = append([]dominio.Faixa(nil), faixas...), 0
			rand.Shuffle(len(faixas), func(i, j int) { faixas[i], faixas[j] = faixas[j], faixas[i] })
		}
		s.TocaDe(faixas, indice)
	case "playCollection":
		if id, ok := a.texto("playlistId"); ok {
			s.TocaColecao(p.filas, id, func(deu bool) {
				if !deu {
					p.quadro.Aviso("Não deu para abrir esta coleção agora.")
				}
			})
		}
	case "toggle":
		s.AlternaPausa()
	case "next":
		s.Proxima()
	case "previous":
		s.Anterior()
	case "seek":
		if segundos, ok := a.real("seconds"); ok && segundos >= 0 && segundos < 1e7 {
			s.VaiPara(time.Duration(segundos*1000+0.5) * time.Millisecond)
			// Pausado não há tique: a posição nova vai agora.
			p.progresso()
		}
	case "playUpNext":
		if indice, ok := a.inteiro("index"); ok {
			s.TocaDaFila(indice)
		}
	case "removeUpNext":
		if indice, ok := a.inteiro("index"); ok {
			s.RemoveDaFila(indice)
		}
	case "moveUpNext":
		de, temDe := a.inteiro("from")
		para, temPara := a.inteiro("to")
		if temDe && temPara {
			s.MoveNaFila(de, para)
		}
	case "clearUpNext":
		s.LimpaFila()
	case "shuffleUpNext":
		s.EmbaralhaFila()
	case "cycleRepeat":
		s.CicloRepetir()
	case "repeat":
		if modo, ok := dominio.RepetirDe(strings.ToUpper(a.textoOu("mode", ""))); ok {
			s.DefineRepetir(modo)
		}
	case "stopAfterCurrent":
		s.AlternaPararAposAtual()
	case "speed":
		if v, ok := a.real("value"); ok {
			s.DefineVelocidade(v)
		}
	case "playNext":
		if f, ok := faixaDoFio(a["track"]); ok {
			s.TocaASeguir(f)
			p.quadro.Aviso("Toca a seguir: " + f.Titulo)
		}
	case "addToQueue":
		if f, ok := faixaDoFio(a["track"]); ok {
			s.PoeNaFila(f)
			p.quadro.Aviso("Na fila: " + f.Titulo)
		}
	case "sleep":
		cancela, _ := a.bool("cancel")
		fim, _ := a.bool("endOfTrack")
		switch minutos, ok := a.inteiro("minutes"); {
		case cancela:
			p.soneca.Cancela()
		case fim:
			p.soneca.FimDaFaixa()
		case ok && minutos > 0 && minutos <= 24*60:
			p.soneca.Comeca(time.Duration(minutos) * time.Minute)
		}
	default:
		return false
	}
	return true
}
