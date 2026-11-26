// SPDX-License-Identifier: GPL-3.0-or-later

package daemon

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"mktunes/internal/app"
	"mktunes/internal/dominio"
	"mktunes/internal/laco"
)

// biblioteca é a aba Biblioteca: curtidas, playlists locais, álbuns salvos, as
// listas inteligentes e o resumo de escuta.
type biblioteca struct {
	quadro  *Quadro
	acervo  *app.Acervo
	listas  []msg
	periodo dominio.Periodo
	resumo  msg
}

func novaBiblioteca(q *Quadro, a *app.Acervo) *biblioteca {
	b := &biblioteca{quadro: q, acervo: a, periodo: dominio.PeriodoSemana, resumo: msg{}}
	b.publica()
	a.Mudou.Ouve(b.publica)
	return b
}

func (b *biblioteca) publica() {
	curtidas := b.acervo.Curtidas()
	ids := make([]string, 0, len(curtidas))
	for _, f := range curtidas {
		ids = append(ids, f.ID)
	}
	playlists := []msg{}
	for _, p := range b.acervo.Playlists() {
		capa := ""
		if capas := p.Capas(); len(capas) > 0 {
			capa = capas[0]
		}
		playlists = append(playlists, msg{"id": p.ID, "title": p.Titulo, "artwork": capa, "tracks": faixasNoFio(p.Faixas)})
	}
	salvos := []msg{}
	for _, c := range b.acervo.Salvos() {
		salvos = append(salvos, msg{"kind": tipoNoFio(c.Tipo), "id": c.ID, "title": c.Titulo, "subtitle": c.Dono, "artwork": c.Capa})
	}
	b.quadro.Estado("library", msg{
		"liked": faixasNoFio(curtidas), "likedIds": ids, "playlists": playlists, "saved": salvos,
		"smart": naoNula(b.listas), "period": strings.ToLower(string(b.periodo)), "stats": b.resumo,
	})
}

// atualiza refaz as listas e o resumo: o histórico anda enquanto a música
// toca, então a janela pede de novo sempre que mostra a biblioteca.
func (b *biblioteca) atualiza() {
	listas := b.acervo.Listas()
	b.listas = nil
	for _, l := range []struct {
		titulo string
		faixas []dominio.Faixa
	}{{"No repeat", listas.NoRepeat}, {"Tocadas recentemente", listas.Recentes}, {"Redescobrir", listas.Redescobrir}} {
		if len(l.faixas) > 0 {
			b.listas = append(b.listas, msg{"title": l.titulo, "tracks": faixasNoFio(l.faixas)})
		}
	}
	resumo := b.acervo.Resumo(b.periodo)
	artistas, faixas := []msg{}, []msg{}
	for _, a := range resumo.Artistas {
		artistas = append(artistas, msg{"name": a.Nome, "plays": a.Vezes})
	}
	for _, f := range resumo.Faixas {
		linha := faixaNoFio(f.Faixa)
		linha["plays"] = f.Vezes
		faixas = append(faixas, linha)
	}
	b.resumo = msg{"plays": resumo.Tocadas, "minutes": int64(resumo.Duracao / time.Minute), "artists": artistas, "tracks": faixas}
	b.publica()
}

func (b *biblioteca) trata(cmd string, a args) bool {
	edita := func(muda func(p dominio.PlaylistLocal) dominio.PlaylistLocal) {
		if id, ok := a.texto("id"); ok {
			b.acervo.EditaPlaylist(id, muda)
		}
	}
	switch cmd {
	case "library":
		b.atualiza()
	case "statsPeriod":
		if periodo, ok := dominio.PeriodoDe(a.textoOu("period", "")); ok {
			b.periodo = periodo
		}
		b.atualiza()
	case "like":
		if f, ok := faixaDoFio(a["track"]); ok {
			if b.acervo.AlternaCurtida(f) {
				b.quadro.Aviso("Removida das curtidas")
			} else {
				b.quadro.Aviso("Curtida")
			}
		}
	case "createPlaylist":
		titulo := a.textoOu("title", "")
		var com *dominio.Faixa
		if f, ok := faixaDoFio(a["track"]); ok {
			com = &f
		}
		if _, ok := b.acervo.CriaPlaylist(titulo, com); ok {
			b.quadro.Aviso("Playlist \"" + strings.TrimSpace(titulo) + "\" criada")
		} else {
			b.quadro.Aviso("Dê um nome à playlist.")
		}
	case "addToPlaylist":
		id, temID := a.texto("id")
		f, temFaixa := faixaDoFio(a["track"])
		if !temID || !temFaixa {
			break
		}
		nome := ""
		for _, p := range b.acervo.Playlists() {
			if p.ID == id {
				nome = p.Titulo
			}
		}
		if b.acervo.AdicionaAPlaylist(id, f) {
			b.quadro.Aviso("Adicionada a " + nome)
		} else {
			b.quadro.Aviso("Já está em " + nome)
		}
	case "removePlaylistTrack":
		if indice, ok := a.inteiro("index"); ok {
			edita(func(p dominio.PlaylistLocal) dominio.PlaylistLocal { return p.Remove(indice) })
		}
	case "movePlaylistTrack":
		de, temDe := a.inteiro("from")
		para, temPara := a.inteiro("to")
		if temDe && temPara {
			edita(func(p dominio.PlaylistLocal) dominio.PlaylistLocal { return p.MoveFaixa(de, para) })
		}
	case "renamePlaylist":
		if titulo, ok := a.texto("title"); ok {
			edita(func(p dominio.PlaylistLocal) dominio.PlaylistLocal { return p.Renomeia(titulo) })
		}
	case "deletePlaylist":
		if id, ok := a.texto("id"); ok {
			b.acervo.ApagaPlaylist(id)
		}
	default:
		return false
	}
	return true
}

// ---- Downloads --------------------------------------------------------------

// ritmoDosDownloads: o progresso anda de 1 % em 1 % para todos os downloads ao
// mesmo tempo; quatro mensagens por segundo bastam.
const ritmoDosDownloads = 250 * time.Millisecond

// Compasso deixa passar no máximo uma chamada por período: a primeira sai na
// hora, depois a última de cada rajada. A última sempre chega, então a janela
// nunca fica num estado velho. Só o laço o usa.
type Compasso struct {
	laco     laco.Laco
	periodo  time.Duration
	faz      func()
	fechado  bool // já saiu uma neste período
	pendente bool // pediram outra enquanto fechado
}

// NovoCompasso chama faz no ritmo dado.
func NovoCompasso(l laco.Laco, periodo time.Duration, faz func()) *Compasso {
	return &Compasso{laco: l, periodo: periodo, faz: faz}
}

// Pede marca que há algo novo a mandar.
func (c *Compasso) Pede() {
	if c.fechado {
		c.pendente = true
		return
	}
	c.fechado = true
	c.faz()
	c.laco.Depois(c.periodo, func() {
		c.fechado = false
		if c.pendente {
			c.pendente = false
			c.Pede()
		}
	})
}

// baixados é a aba Downloads: o que está baixando e o que já está na pasta de
// música.
type baixados struct {
	quadro    *Quadro
	downloads *app.Downloads
	pasta     string
}

func novaAreaDeDownloads(q *Quadro, l laco.Laco, d *app.Downloads, pasta string) *baixados {
	b := &baixados{q, d, pasta}
	b.publica()
	d.Mudou.Ouve(NovoCompasso(l, ritmoDosDownloads, b.publica).Pede)
	return b
}

func (b *baixados) publica() {
	lista := b.downloads.Lista()
	jobs := make([]msg, 0, len(lista))
	for _, d := range lista {
		jobs = append(jobs, downloadNoFio(d))
	}
	b.quadro.Estado("downloads", msg{"folder": b.pasta, "jobs": jobs})
}

func (b *baixados) trata(cmd string, a args) bool {
	id, temID := a.texto("id")
	switch cmd {
	case "download":
		if f, ok := faixaDoFio(a["track"]); ok && !f.AoVivo() {
			b.downloads.Enfileira(f)
			b.quadro.Aviso("Baixando " + f.Titulo)
		}
	case "downloadAll":
		var faixas []dominio.Faixa
		for _, f := range faixasDoFio(a["tracks"]) {
			if !f.AoVivo() {
				faixas = append(faixas, f)
			}
		}
		switch len(faixas) {
		case 0:
			return true
		case 1:
			b.quadro.Aviso("Baixando 1 música")
		default:
			b.quadro.Aviso("Baixando " + strconv.Itoa(len(faixas)) + " músicas")
		}
		b.downloads.Enfileira(faixas...)
	case "cancelDownload":
		if temID {
			b.downloads.Cancela(id)
		}
	case "retryDownload":
		if temID {
			b.downloads.TentaDeNovo(id)
		}
	case "removeDownload":
		if temID {
			b.downloads.Remove(id)
		}
	case "clearDownloadHistory":
		// Só o que nunca chegou ao disco: as músicas concluídas são a biblioteca offline.
		for _, d := range b.downloads.Lista() {
			if d.Estado == dominio.Falhou || d.Estado == dominio.Cancelado {
				b.downloads.Remove(d.Faixa.ID)
			}
		}
	default:
		return false
	}
	return true
}

// ---- Rádio, letras, músicas locais, volume ----------------------------------

// radio é a aba Rádio: as estações mais votadas da região, ou as que casam com
// um nome.
type radio struct {
	quadro     *Quadro
	laco       laco.Laco
	diretorio  app.DiretorioDeRadio
	carregando laco.Tarefa
	mostrada   *string
	pronta     bool
}

func novoRadio(q *Quadro, l laco.Laco, d app.DiretorioDeRadio) *radio {
	r := &radio{quadro: q, laco: l, diretorio: d}
	r.publica("idle", "", nil)
	return r
}

func (r *radio) publica(situacao, consulta string, estacoes []dominio.Faixa) {
	r.pronta = situacao == "ready"
	r.quadro.Estado("radio", msg{"status": situacao, "query": consulta, "stations": faixasNoFio(estacoes)})
}

func (r *radio) trata(cmd string, a args) bool {
	if cmd != "radio" {
		return false
	}
	consulta := strings.TrimSpace(a.textoOu("query", ""))
	// Mostrada de novo sem nada novo a perguntar: fica a lista em vez de recarregar.
	if r.pronta && r.mostrada != nil && *r.mostrada == consulta {
		return true
	}
	laco.Cancela(r.carregando)
	r.publica("loading", consulta, nil)
	r.carregando = r.laco.Vai(func(ctx context.Context) func() {
		estacoes, erro := app.Estacoes(ctx, r.diretorio, consulta)
		return func() {
			switch {
			case erro != nil:
				r.publica("unavailable", consulta, nil)
			case len(estacoes) == 0:
				r.publica("nothing", consulta, nil)
			default:
				r.publica("ready", consulta, estacoes)
			}
			r.mostrada = &consulta
		}
	})
	return true
}

// letras segue a faixa que toca: a letra é buscada de novo só quando a música
// muda.
type letras struct {
	quadro   *Quadro
	laco     laco.Laco
	sessao   *app.Sessao
	acha     *app.Letras
	daFaixa  string
	buscando laco.Tarefa
}

func novasLetras(q *Quadro, l laco.Laco, s *app.Sessao, acha *app.Letras) *letras {
	x := &letras{quadro: q, laco: l, sessao: s, acha: acha}
	q.Estado("lyrics", msg{"status": "none"})
	s.Mudou.Ouve(x.aoMudar)
	x.aoMudar()
	return x
}

func (x *letras) aoMudar() {
	faixa, tem := x.sessao.Estado().Faixa()
	if faixa.ID == x.daFaixa {
		return
	}
	x.daFaixa = faixa.ID
	laco.Cancela(x.buscando)
	if !tem || faixa.AoVivo() {
		x.quadro.Estado("lyrics", msg{"status": "none"})
		return
	}
	x.quadro.Estado("lyrics", msg{"status": "loading"})
	x.buscando = x.laco.Vai(func(ctx context.Context) func() {
		desfecho := x.acha.Acha(ctx, faixa)
		return func() {
			switch {
			case desfecho.Desfecho == app.Indisponivel:
				x.quadro.Estado("lyrics", msg{"status": "unavailable"})
			case desfecho.Desfecho != app.Achou:
				x.quadro.Estado("lyrics", msg{"status": "none"})
			case len(desfecho.Letra.Sincronizada) > 0:
				linhas := make([]msg, 0, len(desfecho.Letra.Sincronizada))
				for _, v := range desfecho.Letra.Sincronizada {
					linhas = append(linhas, msg{"ms": v.Tempo.Milliseconds(), "text": v.Texto})
				}
				x.quadro.Estado("lyrics", msg{"status": "synced", "lines": linhas})
			default:
				x.quadro.Estado("lyrics", msg{"status": "plain", "text": desfecho.Letra.Texto})
			}
		}
	})
}

func (x *letras) trata(string, args) bool { return false }

// musicasLocais é "No computador": as músicas da pasta de música, procuradas
// cada vez que a janela pede.
type musicasLocais struct {
	quadro    *Quadro
	laco      laco.Laco
	procura   func(ctx context.Context) []dominio.Faixa
	olhando   laco.Tarefa
	jaMostrou bool
}

func novoLocal(q *Quadro, l laco.Laco, procura func(ctx context.Context) []dominio.Faixa) *musicasLocais {
	if procura == nil {
		procura = func(context.Context) []dominio.Faixa { return nil }
	}
	q.Estado("local", msg{"status": "idle", "tracks": []msg{}})
	return &musicasLocais{quadro: q, laco: l, procura: procura}
}

func (m *musicasLocais) trata(cmd string, _ args) bool {
	if cmd != "local" {
		return false
	}
	if laco.Ativa(m.olhando) {
		return true
	}
	// O que foi achado da última vez fica na tela enquanto a pasta é olhada de novo.
	if !m.jaMostrou {
		m.quadro.Estado("local", msg{"status": "scanning", "tracks": []msg{}})
	}
	m.olhando = m.laco.Vai(func(ctx context.Context) func() {
		faixas := m.procura(ctx)
		return func() {
			m.jaMostrou = true
			situacao := "ready"
			if len(faixas) == 0 {
				situacao = "empty"
			}
			m.quadro.Estado("local", msg{"status": situacao, "tracks": faixasNoFio(faixas)})
		}
	})
	return true
}

// volume é o volume do desktop, por cima da normalização que cada música já traz.
type volume struct {
	quadro *Quadro
	saida  func(nivel float64)
	guarda GuardaDeVolume
}

func novoVolume(q *Quadro, saida func(float64), guarda GuardaDeVolume) *volume {
	if saida == nil {
		saida = func(float64) {}
	}
	v := &volume{q, saida, guarda}
	nivel := 100
	if guarda != nil {
		if guardado, tem := guarda.Carrega(); tem {
			nivel = guardado
		}
	}
	v.aplica(nivel)
	return v
}

func (v *volume) aplica(nivel int) int {
	nivel = min(max(nivel, 0), 100)
	v.saida(float64(nivel) / 100)
	v.quadro.Estado("volume", msg{"level": nivel})
	return nivel
}

func (v *volume) trata(cmd string, a args) bool {
	if cmd != "volume" {
		return false
	}
	if nivel, ok := a.inteiro("level"); ok {
		nivel = v.aplica(nivel)
		if v.guarda != nil {
			v.guarda.Salva(nivel)
		}
	}
	return true
}

// ---- Ajustes ----------------------------------------------------------------

// backupMax é o maior arquivo de backup lido.
const backupMax = 256 << 20

// ajustes é a aba Ajustes: as preferências de reprodução e o backup da
// biblioteca como arquivo.
type ajustes struct {
	quadro  *Quadro
	laco    laco.Laco
	acervo  *app.Acervo
	config  Config
	tocador app.Tocador
	ultimos dominio.Ajustes
}

func novosAjustes(q *Quadro, l laco.Laco, a *app.Acervo, c Config, t app.Tocador) *ajustes {
	x := &ajustes{quadro: q, laco: l, acervo: a, config: c, tocador: t, ultimos: a.Ajustes()}
	x.publica()
	a.Mudou.Ouve(func() {
		novos := a.Ajustes()
		if novos == x.ultimos {
			return
		}
		if novos.PularSilencio != x.ultimos.PularSilencio {
			t.PularSilencio(novos.PularSilencio)
		}
		x.ultimos = novos
		c.AoMudarAjustes(novos)
		x.publica()
	})
	return x
}

func (x *ajustes) publica() {
	s := x.ultimos
	x.quadro.Estado("settings", msg{
		"autoplay": s.Autoplay, "normalizeVolume": s.Normalizar, "audioQuality": strings.ToLower(string(s.Qualidade)),
		"downloadWorkers": s.Trabalhadores, "maxWorkers": dominio.MaxTrabalhadores,
		"backupsDir": x.config.PastaDeBackups, "version": x.config.Versao,
	})
}

func (x *ajustes) trata(cmd string, a args) bool {
	switch cmd {
	case "setting":
		chave, _ := a.texto("key")
		ligado, eBool := a.bool("value")
		x.acervo.MudaAjustes(func(s dominio.Ajustes) dominio.Ajustes {
			switch chave {
			case "autoplay":
				if eBool {
					s.Autoplay = ligado
				}
			case "normalizeVolume":
				if eBool {
					s.Normalizar = ligado
				}
			case "downloadWorkers":
				if n, ok := a.inteiro("value"); ok {
					s = s.ComTrabalhadores(n)
				}
			case "audioQuality":
				if q, ok := dominio.QualidadeDe(a.textoOu("value", "")); ok {
					s.Qualidade = q
				}
			}
			return s
		})
	case "exportBackup":
		arquivo := filepath.Join(x.config.PastaDeBackups, "mktunes-backup-"+x.laco.Agora().Local().Format("2006-01-02")+".json")
		texto := x.acervo.Exporta()
		x.laco.Vai(func(context.Context) func() {
			erro := os.MkdirAll(x.config.PastaDeBackups, 0o755)
			if erro == nil {
				erro = os.WriteFile(arquivo, []byte(texto), 0o600)
			}
			return func() {
				if erro != nil {
					x.quadro.Aviso("Não deu para salvar o backup em " + x.config.PastaDeBackups + ".")
					return
				}
				x.quadro.Aviso("Backup salvo em " + arquivo)
			}
		})
	case "importBackup":
		caminho, ok := a.texto("path")
		if !ok {
			break
		}
		x.laco.Vai(func(context.Context) func() {
			var bruto []byte
			arquivo, erro := os.Open(caminho)
			if erro == nil {
				bruto, erro = io.ReadAll(io.LimitReader(arquivo, backupMax))
				_ = arquivo.Close()
			}
			return func() {
				if erro != nil {
					x.quadro.Aviso("Não deu para ler " + caminho + ".")
					return
				}
				if r, ok := x.acervo.Restaura(string(bruto)); ok {
					x.quadro.Aviso("Restaurado: " + strconv.Itoa(r.Curtidas) + " curtidas, " + strconv.Itoa(r.Playlists) + " playlists")
				} else {
					x.quadro.Aviso("Este arquivo não é um backup do MkTunes.")
				}
			}
		})
	default:
		return false
	}
	return true
}
