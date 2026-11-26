// Package area liga o motor à área de transferência do Wayland pelo wl-copy e
// pelo wl-paste (o QML não expõe tipos MIME arbitrários). O motor só carrega o
// texto: montar e ler a lista de arquivos é da janela. Veja
// ../../../CONTRATO.md.
package area

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"mkfiles/internal/base"
	"mkfiles/internal/item"
)

// Os limites: o texto que vai e vem, o arquivo gravado e a paciência com os
// comandos (os testes encurtam).
const (
	LimiteDoTexto   = 4 << 20
	LimiteDoArquivo = 512 << 20
)

var (
	prazoDeLer    = 5 * time.Second
	prazoDePor    = 3 * time.Second
	prazoDeGravar = 60 * time.Second
)

// Os tipos que o area-por aceita.
var tiposDePor = map[string]bool{"text/uri-list": true, "x-special/gnome-copied-files": true, "text/plain": true}

// Os nomes com que um texto simples aparece na área de transferência.
var nomesDeTexto = []string{"text/plain;charset=utf-8", "text/plain", "UTF8_STRING", "STRING", "TEXT"}

// Ligar põe na sessão os pedidos area-por, area-ler e area-gravar.
func Ligar(s *base.Sessao) {
	s.Ligar("area-por", por)
	s.Ligar("area-ler", ler)
	s.Ligar("area-gravar", gravar)
}

func por(p *base.Pedido) {
	var q struct {
		Tipo  string `json:"tipo"`
		Texto string `json:"texto"`
	}
	_ = p.Ler(&q)
	if !tiposDePor[q.Tipo] {
		p.FalhaDeCampo("tipo", "não ponho esse tipo na área de transferência")
		return
	}
	if len(q.Texto) > LimiteDoTexto {
		p.Falha("isso é grande demais para a área de transferência")
		return
	}
	cmd := exec.Command(base.Comando("MKFILES_WLCOPY", "wl-copy"), "--type", q.Tipo)
	cmd.Stdin = strings.NewReader(q.Texto)
	var dito bytes.Buffer
	cmd.Stderr = &dito
	if err := cmd.Start(); err != nil {
		p.Falha("não achei o wl-copy: " + err.Error())
		return
	}
	// O wl-copy lê a entrada, vai para o fundo e fica lá servindo a seleção:
	// o que se espera é só o processo da frente sair. Se demorar, fica solto.
	acabou := make(chan error, 1)
	go func() { acabou <- cmd.Wait() }()
	select {
	case err := <-acabou:
		if err != nil {
			frase := strings.TrimSpace(dito.String())
			if frase == "" {
				frase = err.Error()
			}
			p.Falha("o wl-copy falhou: " + frase)
			return
		}
	case <-time.After(prazoDePor):
	}
	p.Responde(base.Campos{"posto": true})
}

// colar roda o wl-paste e devolve a saída (até o limite).
func colar(prazo time.Duration, limite int64, args ...string) ([]byte, error) {
	ctx, cancela := context.WithTimeout(context.Background(), prazo)
	defer cancela()
	cmd := exec.CommandContext(ctx, base.Comando("MKFILES_WLPASTE", "wl-paste"), args...)
	cmd.WaitDelay = time.Second
	// A saída vai para um balde com fundo: passou do limite, o comando é morto.
	// (Sem cano lido aqui: com o WaitDelay, um filho que o comando deixou
	// segurando a saída não prende a espera.)
	balde := &limitado{resto: limite, estourou: cancela}
	cmd.Stdout = balde
	err := cmd.Run()
	if balde.cheio {
		return nil, errors.New("o que está na área de transferência é grande demais")
	}
	if ctx.Err() != nil {
		return nil, errors.New("o wl-paste demorou demais")
	}
	if err != nil {
		return nil, err
	}
	return balde.dados.Bytes(), nil
}

// limitado junta o que escrevem nele até o limite.
type limitado struct {
	dados    bytes.Buffer
	resto    int64
	cheio    bool
	estourou func()
}

func (l *limitado) Write(b []byte) (int, error) {
	if int64(len(b)) > l.resto {
		l.cheio = true
		l.estourou()
		return 0, errors.New("limite")
	}
	l.resto -= int64(len(b))
	return l.dados.Write(b)
}

// tipos lista o que a área de transferência oferece (vazio se não há nada: o
// wl-paste sai com erro).
func tipos() []string {
	lista := []string{}
	saida, err := colar(prazoDeLer, 1<<20, "--list-types")
	if err != nil {
		return lista
	}
	for _, linha := range strings.Split(string(saida), "\n") {
		if t := strings.TrimSpace(linha); t != "" {
			lista = append(lista, t)
		}
	}
	return lista
}

func tem(lista []string, tipo string) bool {
	for _, t := range lista {
		if t == tipo {
			return true
		}
	}
	return false
}

// textoDe diz com que nome o texto simples está na lista ("" se não está).
func textoDe(lista []string) string {
	for _, nome := range nomesDeTexto {
		if tem(lista, nome) {
			return nome
		}
	}
	return ""
}

func ler(p *base.Pedido) {
	lista := tipos()
	resposta := base.Campos{"tipos": lista, "tipo": nil, "texto": nil}
	pedir := ""
	switch {
	case tem(lista, "x-special/gnome-copied-files"):
		resposta["tipo"], pedir = "x-special/gnome-copied-files", "x-special/gnome-copied-files"
	case tem(lista, "text/uri-list"):
		resposta["tipo"], pedir = "text/uri-list", "text/uri-list"
	case tem(lista, "image/png"):
		resposta["tipo"] = "image/png"
	case textoDe(lista) != "":
		resposta["tipo"], pedir = "text/plain", textoDe(lista)
	}
	if pedir != "" {
		dados, err := colar(prazoDeLer, LimiteDoTexto, "--no-newline", "--type", pedir)
		if err != nil {
			p.Falha("não deu para ler a área de transferência: " + err.Error())
			return
		}
		resposta["texto"] = string(dados)
	}
	p.Responde(resposta)
}

func gravar(p *base.Pedido) {
	var q struct {
		Destino string `json:"destino"`
		Nome    string `json:"nome"`
		Tipo    string `json:"tipo"`
	}
	_ = p.Ler(&q)
	if err := base.Caminho(q.Destino); err != nil {
		p.Falha(err.Error())
		return
	}
	if err := base.Nome(q.Nome); err != nil {
		p.FalhaDeCampo("nome", err.Error())
		return
	}
	lista := tipos()
	pedir := ""
	switch q.Tipo {
	case "image/png":
		if tem(lista, "image/png") {
			pedir = "image/png"
		}
	case "text/plain":
		pedir = textoDe(lista)
	default:
		p.FalhaDeCampo("tipo", "só sei colar imagem (image/png) ou texto (text/plain)")
		return
	}
	if pedir == "" {
		p.Falha("não há isso na área de transferência")
		return
	}
	if info, err := os.Stat(q.Destino); err != nil || !info.IsDir() {
		p.Falha("a pasta de destino não existe mais")
		return
	}
	caminho := filepath.Join(q.Destino, q.Nome)
	// O_EXCL: nunca por cima de um arquivo que já existe.
	arq, err := os.OpenFile(caminho, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			p.FalhaDeCampo("nome", "já existe «"+q.Nome+"» nessa pasta")
		} else if errors.Is(err, fs.ErrPermission) {
			p.Falha("você não tem permissão para gravar nessa pasta")
		} else {
			p.Falha(err.Error())
		}
		return
	}
	args := []string{"--type", pedir}
	if q.Tipo == "text/plain" {
		args = append([]string{"--no-newline"}, args...)
	}
	dados, err := colar(prazoDeGravar, LimiteDoArquivo, args...)
	if err == nil && len(dados) == 0 {
		err = errors.New("a área de transferência está vazia")
	}
	if err == nil {
		_, err = arq.Write(dados)
	}
	if e := arq.Close(); err == nil {
		err = e
	}
	if err != nil {
		os.Remove(caminho) // o arquivo é o que acabou de ser criado aqui
		p.Falha("não deu para colar: " + err.Error())
		return
	}
	it, err := item.De(caminho, true)
	if err != nil {
		p.Falha(err.Error())
		return
	}
	p.Responde(base.Campos{"item": it})
}
