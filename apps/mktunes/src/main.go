// SPDX-License-Identifier: GPL-3.0-or-later

// mktunesd é o daemon do MkTunes, o player de música do Mk: toca pelo mpv,
// busca no YouTube Music (InnerTube) e resolve o áudio com o yt-dlp, guarda a
// biblioteca em JSON e aparece na ilha pelo MPRIS. A janela (Quickshell) fala
// com ele por $XDG_RUNTIME_DIR/mktunes/mktunes.sock; o protocolo está em
// CONTRATO.md. Este arquivo é a raiz de composição: o único lugar que sabe
// qual adaptador entra em qual porta.
//
//	mktunesd            sobe o daemon (sai com 0 se já há um respondendo)
//	mktunesd --versao   diz a versão deste binário
//	mktunesd --ping     sai com 0 e diz a versão do daemon que responde no socket; 1 se ninguém responde
//	mktunesd --encerra  pede ao daemon que responde para sair (guardando a sessão) e espera
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"mktunes/internal/app"
	"mktunes/internal/daemon"
	"mktunes/internal/dbus"
	"mktunes/internal/dominio"
	"mktunes/internal/guarda"
	"mktunes/internal/innertube"
	"mktunes/internal/laco"
	"mktunes/internal/local"
	"mktunes/internal/mpris"
	"mktunes/internal/mpv"
	"mktunes/internal/rede"
	"mktunes/internal/ytdlp"
)

// Versao é a deste binário; o lançador troca um daemon que responde outra.
const Versao = "0.2.0"

const prazoDaRede = 10 * time.Second

func main() { os.Exit(roda(os.Args[1:])) }

func roda(argumentos []string) int {
	socket := filepath.Join(pastaDeExecucao(), "mktunes.sock")
	if len(argumentos) == 0 {
		return sobe(socket)
	}
	switch argumentos[0] {
	case "--versao":
		fmt.Println(Versao)
		return 0
	case "--ping":
		versao, responde := versaoViva(socket, 2*time.Second)
		if !responde {
			return 1
		}
		fmt.Println(versao)
		return 0
	case "--encerra":
		return encerra(socket)
	}
	fmt.Fprintln(os.Stderr, "uso: mktunesd [--versao | --ping | --encerra]")
	return 2
}

func avisa(formato string, a ...any) { fmt.Fprintf(os.Stderr, "mktunesd: "+formato+"\n", a...) }

func ambiente(nome, reserva string) string {
	if valor := strings.TrimSpace(os.Getenv(nome)); valor != "" {
		return valor
	}
	return reserva
}

func casa() string {
	if pasta := os.Getenv("HOME"); pasta != "" {
		return pasta
	}
	if eu, erro := user.Current(); erro == nil {
		return eu.HomeDir
	}
	return "/"
}

func xdg(variavel, reserva string) string { return ambiente(variavel, filepath.Join(casa(), reserva)) }

// pastaDeExecucao é $XDG_RUNTIME_DIR/mktunes: só do usuário, esvaziada ao sair da sessão.
func pastaDeExecucao() string {
	base := os.Getenv("XDG_RUNTIME_DIR")
	if base == "" {
		nome := "desconhecido"
		if eu, erro := user.Current(); erro == nil {
			nome = eu.Username
		}
		base = filepath.Join(os.TempDir(), "mktunes-"+nome)
	}
	return filepath.Join(base, "mktunes")
}

var linhaDePasta = regexp.MustCompile(`^\s*(XDG_[A-Z]+_DIR)="(.*)"\s*$`)

// pastaDoUsuario lê uma pasta do usuário (MUSIC, DOCUMENTS) em
// user-dirs.dirs, como o xdg-user-dir faria; "" quando não há (ou é a própria
// pasta pessoal).
func pastaDoUsuario(nome string) string {
	bruto, erro := os.ReadFile(filepath.Join(xdg("XDG_CONFIG_HOME", ".config"), "user-dirs.dirs"))
	if erro != nil {
		return ""
	}
	for _, linha := range strings.Split(string(bruto), "\n") {
		achado := linhaDePasta.FindStringSubmatch(linha)
		if achado == nil || achado[1] != "XDG_"+nome+"_DIR" {
			continue
		}
		pasta := achado[2]
		if resto, tem := strings.CutPrefix(pasta, "$HOME"); tem {
			pasta = casa() + resto
		}
		if pasta = filepath.Clean(pasta); filepath.IsAbs(pasta) && pasta != filepath.Clean(casa()) {
			return pasta
		}
	}
	return ""
}

// regiao é o país do ouvinte, pelo idioma do sistema ("pt_BR.UTF-8" → "BR").
func regiao() string {
	for _, variavel := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if achado := regexp.MustCompile(`^[a-z]{2,3}_([A-Z]{2})`).FindStringSubmatch(os.Getenv(variavel)); achado != nil {
			return achado[1]
		}
	}
	return "BR"
}

// versaoViva conecta ao socket e lê a versão que o daemon publica em "settings".
func versaoViva(socket string, prazo time.Duration) (string, bool) {
	conexao, erro := net.DialTimeout("unix", socket, time.Second)
	if erro != nil {
		return "", false
	}
	defer conexao.Close()
	_ = conexao.SetReadDeadline(time.Now().Add(prazo))
	linhas := bufio.NewScanner(conexao)
	linhas.Buffer(make([]byte, 64*1024), 16<<20)
	for linhas.Scan() {
		var m struct {
			Type    string `json:"type"`
			Version string `json:"version"`
		}
		if json.Unmarshal(linhas.Bytes(), &m) == nil && m.Type == "settings" {
			return m.Version, true
		}
	}
	// Alguém responde, mas não diz a versão (um daemon de antes deste).
	return "", true
}

func encerra(socket string) int {
	conexao, erro := net.DialTimeout("unix", socket, time.Second)
	if erro != nil {
		return 0 // ninguém para encerrar
	}
	_, _ = conexao.Write([]byte(`{"cmd":"quit"}` + "\n"))
	_ = conexao.Close()
	for range 50 {
		if !daemon.Responde(socket) {
			return 0
		}
		time.Sleep(100 * time.Millisecond)
	}
	avisa("o daemon em %s não saiu", socket)
	return 1
}

// levanta abre a janela (ou a traz para a frente): é o "Raise" da ilha.
func levanta() {
	comando := exec.Command(ambiente("MKTUNES_LAUNCHER", "mktunes"))
	comando.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if comando.Start() == nil {
		go func() { _ = comando.Wait() }()
	}
}

func sobe(socket string) int {
	if daemon.Responde(socket) {
		avisa("já está rodando em %s", socket)
		return 0
	}
	// O motivo de este daemon existir em Go é gastar pouca memória: o coletor
	// trabalha um pouco mais cedo do que o padrão.
	debug.SetGCPercent(50)
	execucao := filepath.Dir(socket)
	if erro := os.MkdirAll(execucao, 0o700); erro != nil {
		avisa("%v", erro)
		return 1
	}
	l := laco.Novo()

	dados := filepath.Join(xdg("XDG_DATA_HOME", ".local/share"), "mktunes")
	cofre, erro := guarda.Abre(dados)
	if erro != nil {
		avisa("%v", erro)
		return 1
	}
	var avisosDaSubida []string
	switch migrado, migrou, erro := cofre.Migra(context.Background(), ambiente("MKTUNES_SQLITE3", "sqlite3")); {
	case errors.Is(erro, guarda.ErrSemSqlite):
		avisosDaSubida = append(avisosDaSubida, "Para trazer a biblioteca da versão anterior, instale o pacote sqlite (sudo pacman -S sqlite) e abra o MkTunes de novo.")
	case erro != nil:
		avisa("a migração do banco antigo falhou: %v", erro)
		avisosDaSubida = append(avisosDaSubida, "Não deu para trazer a biblioteca da versão anterior; o banco antigo continua guardado.")
	case migrou:
		avisa("biblioteca migrada do banco antigo: %+v", migrado)
		avisosDaSubida = append(avisosDaSubida, fmt.Sprintf("Biblioteca trazida da versão anterior: %d curtidas, %d playlists.", migrado.Curtidas, migrado.Playlists))
	}

	transporte := &http.Transport{
		Proxy: http.ProxyFromEnvironment, MaxIdleConns: 8, MaxIdleConnsPerHost: 2, IdleConnTimeout: 60 * time.Second,
		TLSHandshakeTimeout: prazoDaRede, ResponseHeaderTimeout: prazoDaRede,
		DialContext: (&net.Dialer{Timeout: prazoDaRede}).DialContext, ForceAttemptHTTP2: true,
	}
	rapida := &http.Client{Transport: transporte, Timeout: prazoDaRede}
	// Os downloads não têm prazo total (o arquivo pode ser grande): quem corta
	// uma transferência parada é o próprio buscador.
	lenta := &http.Client{Transport: transporte}

	pais := regiao()
	visitante := &innertube.Visitante{}
	base := os.Getenv("MKTUNES_INNERTUBE_URL")
	feed := innertube.Feed{Cliente: &innertube.Cliente{HTTP: rapida, Perfil: innertube.WebRemix, Base: base, Idioma: "pt", Regiao: pais, Visitante: visitante}}
	volume := innertube.Volume{Cliente: &innertube.Cliente{HTTP: rapida, Perfil: innertube.AndroidVR, Base: base, Idioma: "pt", Regiao: pais, Visitante: visitante}}

	musica := os.Getenv("MKTUNES_MUSIC_DIR")
	if musica == "" {
		pasta := pastaDoUsuario("MUSIC")
		if pasta == "" {
			pasta = filepath.Join(casa(), "Music")
		}
		musica = filepath.Join(pasta, "MkTunes")
	}
	_ = os.MkdirAll(musica, 0o755)
	documentos := pastaDoUsuario("DOCUMENTS")
	if documentos == "" {
		documentos = casa()
	}
	arquivos := rede.Arquivos{HTTP: lenta, Raiz: musica}
	arquivos.LimpaIncompletos()
	pasta := &local.Pasta{Raiz: filepath.Dir(musica), Pula: musica, Le: local.Ffprobe(ambiente("MKTUNES_FFPROBE", "ffprobe"))}

	var (
		aplicativo *daemon.App
		tocador    *mpv.Tocador
		processo   *mpv.Processo
		saindo     bool
		quedas     []time.Time
	)
	extras := strings.Fields(os.Getenv("MKTUNES_MPV_ARGS"))
	socketDoMpv := filepath.Join(execucao, "mpv.sock")
	var ligaMpv func() error
	ligaMpv = func() error {
		novo, erro := mpv.Inicia(ambiente("MKTUNES_MPV", "mpv"), socketDoMpv, extras,
			func(linha string) { l.Posta(func() { tocador.AoLinha(linha) }) },
			func() {
				l.Posta(func() {
					if saindo {
						return
					}
					// O mpv morreu: sobe outro, a não ser que esteja morrendo sem parar.
					agora := time.Now()
					quedas = append(quedas, agora)
					for len(quedas) > 0 && agora.Sub(quedas[0]) > time.Minute {
						quedas = quedas[1:]
					}
					avisa("o mpv saiu; subindo outro")
					if len(quedas) > 5 || ligaMpv() != nil {
						avisa("o mpv não fica de pé; o daemon segue sem som")
						return
					}
					tocador.Religa(processo.Envia)
					aplicativo.Sessao.AoEvento(app.Evento{Tipo: app.FalhouAoTocar, Motivo: "o mpv saiu"})
				})
			})
		if erro == nil {
			processo = novo
		}
		return erro
	}
	if erro := ligaMpv(); erro != nil {
		avisa("não deu para subir o mpv: %v", erro)
		return 1
	}
	tocador = mpv.NovoTocador(processo.Envia, func(e app.Evento) { aplicativo.Sessao.AoEvento(e) })

	resolvedor := &ytdlp.Resolvedor{
		Comando: ambiente("MKTUNES_YTDLP", "yt-dlp"),
		Volume:  volume.LoudnessDb,
		Aviso:   func(texto string) { l.Posta(func() { aplicativo.Quadro.Aviso(texto) }) },
	}

	var servidor *daemon.Servidor
	var barramento *dbus.Conexao
	sai := func() {
		if saindo {
			return
		}
		saindo = true
		aplicativo.SalvaSessao()
		if servidor != nil {
			servidor.Fecha()
		}
		if barramento != nil {
			go barramento.Fecha()
		}
		processo.Fecha()
		l.Fecha()
	}

	radios := rede.RadioBrowser{HTTP: rapida, Pais: pais}
	if espelhos := os.Getenv("MKTUNES_RADIO_URL"); espelhos != "" {
		radios.Servidores = strings.Split(espelhos, ",")
	}
	aplicativo = daemon.Monta(daemon.Portas{
		Catalogo: feed, Inicio: feed, Filas: feed, Navegador: feed, Paginas: feed, Sugestoes: feed,
		Streams:        app.PorProvedor{dominio.YouTube: resolvedor},
		Tocador:        tocador,
		Letras:         rede.Lrclib{HTTP: rapida, Base: os.Getenv("MKTUNES_LRCLIB_URL")},
		Radio:          radios,
		Arquivos:       arquivos,
		GuardaDoInicio: guarda.FeedGuardado{Arquivo: filepath.Join(xdg("XDG_CACHE_HOME", ".cache"), "mktunes", "home-feed.json")},
		Volume:         tocador.VolumeDoOuvinte,
		Local:          pasta.Varre,
	}, daemon.Config{
		Guarda: cofre, Codec: guarda.Codec{}, Laco: l,
		PastaDeBackups: documentos, PastaDeDownloads: musica, Versao: Versao,
		Volume:         daemon.VolumeEmArquivo{Arquivo: filepath.Join(xdg("XDG_STATE_HOME", ".local/state"), "mktunes", "volume")},
		AoMudarAjustes: func(a dominio.Ajustes) { resolvedor.DefineQualidade(a.Qualidade) },
		AoSair:         sai,
		Registro:       func(texto string) { avisa("%s", texto) },
	})
	for _, texto := range avisosDaSubida {
		aplicativo.Quadro.Aviso(texto)
	}

	servidor, erro = daemon.Serve(socket, aplicativo.Quadro, l, aplicativo.Trata)
	if erro != nil {
		avisa("%v", erro)
		processo.Fecha()
		if errors.Is(erro, daemon.ErrJaRodando) {
			return 0
		}
		return 1
	}

	// MPRIS: a ilha e as teclas de mídia. Sem barramento de sessão o daemon
	// funciona igual; MKTUNES_MPRIS=0 o deixa fora do barramento.
	if os.Getenv("MKTUNES_MPRIS") != "0" {
		go func() {
			var servico *mpris.Servico
			conexao, erro := dbus.Liga(func(m *dbus.Mensagem) {
				l.Posta(func() {
					if servico != nil {
						servico.Chamada(m)
					}
				})
			})
			if erro != nil {
				if !errors.Is(erro, dbus.ErrSemBarramento) {
					avisa("MPRIS indisponível: %v", erro)
				}
				return
			}
			pronto := make(chan struct{})
			l.Posta(func() {
				barramento = conexao
				servico = mpris.NovoServico(conexao, l, aplicativo.Quadro.Linha, aplicativo.Trata, levanta)
				aplicativo.Quadro.Ouve(func(tipo, _ string) {
					if tipo == "now" || tipo == "progress" || tipo == "volume" {
						servico.Mudou()
					}
				})
				close(pronto)
			})
			<-pronto
			// O nome só é pedido com o serviço já pronto para responder.
			if dono, erro := conexao.PedeNome(mpris.Nome); erro != nil || !dono {
				avisa("MPRIS indisponível: o nome %s não ficou com este daemon (%v)", mpris.Nome, erro)
			}
		}()
	}

	sinais := make(chan os.Signal, 2)
	signal.Notify(sinais, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	go func() {
		<-sinais
		l.Posta(sai)
	}()
	l.Roda()
	return 0
}
