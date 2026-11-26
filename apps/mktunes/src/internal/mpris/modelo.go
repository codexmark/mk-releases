// SPDX-License-Identifier: GPL-3.0-or-later

// Pacote mpris: o MkTunes como player MPRIS no barramento de sessão, para a
// ilha, as teclas de mídia e o playerctl o controlarem. É mais um cliente do
// protocolo do daemon: lê as mesmas mensagens de estado que a janela lê
// ("now", "progress", "volume") e manda os mesmos comandos.
package mpris

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"mktunes/internal/dbus"
)

// SemFaixa é o id de faixa que o MPRIS usa quando nada toca.
const SemFaixa = dbus.Caminho("/org/mpris/MediaPlayer2/TrackList/NoTrack")

const micros = 1_000_000

// estado é uma mensagem do daemon já lida.
type estado map[string]any

func le(linha string) estado {
	var m estado
	_ = json.Unmarshal([]byte(linha), &m)
	if m == nil {
		m = estado{}
	}
	return m
}

func (e estado) texto(chave string) string {
	valor, _ := e[chave].(string)
	return valor
}

func (e estado) numero(chave string, reserva float64) float64 {
	if valor, ok := e[chave].(float64); ok {
		return valor
	}
	return reserva
}

func (e estado) ligado(chave string) bool {
	valor, _ := e[chave].(bool)
	return valor
}

// Jogador são as propriedades de org.mpris.MediaPlayer2.Player, como valores
// simples: texto, bool, float64, int64 (x), dbus.Caminho, []string e, para os
// metadados, map[string]any.
func Jogador(agora, progresso, volume string) map[string]any {
	now := le(agora)
	situacao := now.texto("status")
	temFaixa := situacao != "" && situacao != "idle"
	aoVivo := now.ligado("live")
	tocando := "Stopped"
	switch situacao {
	case "playing", "loading":
		tocando = "Playing"
	case "paused":
		tocando = "Paused"
	}
	repete := "None"
	switch now.texto("repeat") {
	case "one":
		repete = "Track"
	case "all":
		repete = "Playlist"
	}
	return map[string]any{
		"PlaybackStatus": tocando,
		"LoopStatus":     repete,
		"Rate":           now.numero("speed", 1),
		"Shuffle":        false,
		"Metadata":       metadados(now),
		"Volume":         le(volume).numero("level", 100) / 100,
		"Position":       posicao(progresso),
		"MinimumRate":    0.75,
		"MaximumRate":    1.5,
		"CanGoNext":      temFaixa,
		"CanGoPrevious":  temFaixa,
		"CanPlay":        temFaixa,
		"CanPause":       temFaixa,
		"CanSeek":        temFaixa && !aoVivo,
		"CanControl":     true,
	}
}

func posicao(progresso string) int64 {
	return int64(le(progresso).numero("position", 0) * micros)
}

func metadados(now estado) map[string]any {
	faixa, _ := now["track"].(map[string]any)
	id := strings.TrimSpace(estado(faixa).texto("id"))
	if id == "" {
		return map[string]any{"mpris:trackid": SemFaixa}
	}
	m := map[string]any{
		"mpris:trackid": dbus.Caminho("/dev/mk/mktunes/track/" + seguroNoCaminho(estado(faixa).texto("id"))),
		"xesam:title":   estado(faixa).texto("title"),
		"xesam:artist":  []string{estado(faixa).texto("artist")},
	}
	if capa := estado(faixa).texto("artwork"); strings.TrimSpace(capa) != "" {
		m["mpris:artUrl"] = capa
	}
	if duracao := estado(faixa).numero("durationMs", 0); !now.ligado("live") && duracao > 0 {
		m["mpris:length"] = int64(duracao) * 1000
	}
	return m
}

// seguroNoCaminho: os caminhos de objeto só aceitam [A-Za-z0-9_]; o resto é
// escrito como _ e o código em hexadecimal.
func seguroNoCaminho(id string) string {
	var saida strings.Builder
	for _, c := range id {
		if c < 128 && (c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9') {
			saida.WriteRune(c)
		} else {
			fmt.Fprintf(&saida, "_%02X", c)
		}
	}
	return saida.String()
}

func comando(campos map[string]any) string {
	linha, _ := json.Marshal(campos)
	return string(linha)
}

// Os botões do sistema viram os comandos da própria janela. Os que podem não
// querer dizer nada devolvem false.

func TocaPausa() string { return comando(map[string]any{"cmd": "toggle"}) }
func Proxima() string   { return comando(map[string]any{"cmd": "next"}) }
func Anterior() string  { return comando(map[string]any{"cmd": "previous"}) }

// Toca só vale com a música pausada.
func Toca(agora string) (string, bool) { return TocaPausa(), le(agora).texto("status") == "paused" }

// Pausa só vale com a música tocando (ou carregando).
func Pausa(agora string) (string, bool) {
	situacao := le(agora).texto("status")
	return TocaPausa(), situacao == "playing" || situacao == "loading"
}

func vaiPara(microssegundos int64) string {
	return comando(map[string]any{"cmd": "seek", "seconds": float64(max(microssegundos, 0)/1000) / 1000})
}

// PoePosicao pula para uma posição, em microssegundos.
func PoePosicao(microssegundos int64) string { return vaiPara(microssegundos) }

// Pula anda a posição de agora para a frente ou para trás.
func Pula(deslocamento int64, progresso string) string {
	return vaiPara(posicao(progresso) + deslocamento)
}

// Volume põe o volume, de 0 a 1.
func Volume(nivel float64) string {
	return comando(map[string]any{"cmd": "volume", "level": int(math.Round(nivel * 100))})
}

// Repete traduz o LoopStatus do MPRIS no modo de repetir.
func Repete(situacao string) (string, bool) {
	modo, ok := map[string]string{"None": "off", "Track": "one", "Playlist": "all"}[situacao]
	return comando(map[string]any{"cmd": "repeat", "mode": modo}), ok
}

// Embaralha: a fila é embaralhada uma vez; não há um modo para desligar.
func Embaralha(ligado bool) (string, bool) {
	return comando(map[string]any{"cmd": "shuffleUpNext"}), ligado
}
