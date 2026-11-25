package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"mkporto/internal/base"
	"mkporto/internal/sshcfg"
)

func init() { registrar("sondar", sondar) }

// Os prazos da sonda: abrir a porta, ouvir a saudação, e a volta inteira por
// um salto (o ssh de lá tem 4 s para conectar: ConnectTimeout).
const prazoDaSonda = 2 * time.Second

var (
	prazoDaSaudacao = 2 * time.Second
	prazoDoSalto    = 6 * time.Second
)

const (
	sondasDeUmaVez    = 32
	motivoDoSalto     = "por salto"
	motivoDoEstranho  = "não conheço esse apelido"
	motivoDeNaoSerSsh = "a porta abre, mas não é um servidor ssh"
	motivoDoBarrado   = "o salto não deixou passar"
	// Uma saudação é uma linha curta; antes dela o servidor pode dizer outras.
	linhaDeSaudacao  = 512
	linhasAntesDela  = 16
	nomeDeServidor   = 64
	reclamacaoMaxima = 120
)

// sondaJSON é o resultado de uma sonda. `responde` é null quando a máquina
// não foi sondada (não dá para chegar nela sem conversa, ou o apelido não
// existe).
type sondaJSON struct {
	Apelido  string  `json:"apelido"`
	Responde *bool   `json:"responde"`
	Ms       *int64  `json:"ms"`
	Motivo   *string `json:"motivo"`
	Servidor *string `json:"servidor"`
	Via      *string `json:"via"`
}

func (r *sondaJSON) respondeu(sim bool, motivo string) {
	r.Responde, r.Motivo = &sim, ouNada(motivo)
}

// sondar: vê se há um servidor ssh na porta de cada host (direto, ou pelo
// salto dele). Não autentica: ouve a saudação e fecha. Uma linha por
// resultado, conforme chegam.
func sondar(args []string) int {
	pedidos, err := posicionais(args)
	if err != nil {
		return base.Falha(base.Erro, "%s", err)
	}
	todos := catalogo()
	if len(pedidos) == 0 {
		for _, h := range todos {
			pedidos = append(pedidos, h.Apelido)
		}
	}
	// Só os pedidos são perguntados ao ssh, cada um uma vez.
	porApelido := map[string]sshcfg.Host{}
	for _, h := range todos {
		porApelido[h.Apelido] = h
	}
	var alvos []sshcfg.Host
	feito := map[string]bool{}
	for _, apelido := range pedidos {
		if feito[apelido] {
			continue
		}
		feito[apelido] = true
		h, existe := porApelido[apelido]
		if !existe {
			_ = base.Escreve(sondaJSON{Apelido: apelido, Motivo: ouNada(motivoDoEstranho)})
			continue
		}
		alvos = append(alvos, h)
	}
	efetiva(alvos)
	ssh, temSsh := oSsh()

	var espera sync.WaitGroup
	vagas := make(chan struct{}, sondasDeUmaVez)
	for _, h := range alvos {
		var salto []string
		if h.Salto != "" {
			var da bool
			if salto, da = argvDoSalto(ssh, h.Salto, h.Host, h.Porta); !da || !temSsh {
				_ = base.Escreve(sondaJSON{Apelido: h.Apelido, Motivo: ouNada(motivoDoSalto)})
				continue
			}
		}
		espera.Add(1)
		go func(h sshcfg.Host, salto []string) {
			defer espera.Done()
			vagas <- struct{}{}
			defer func() { <-vagas }()
			if salto != nil {
				_ = base.Escreve(sondaPorSalto(h, salto))
			} else {
				_ = base.Escreve(sonda(h))
			}
		}(h, salto)
	}
	espera.Wait()
	return base.Certo
}

// sonda abre uma conexão TCP em host:porta, ouve a saudação do servidor e
// fecha. Não manda nada.
func sonda(h sshcfg.Host) sondaJSON {
	r := sondaJSON{Apelido: h.Apelido}
	inicio := time.Now()
	conexao, err := net.DialTimeout("tcp", net.JoinHostPort(h.Host, strconv.Itoa(h.Porta)), prazoDaSonda)
	if err != nil {
		r.respondeu(false, motivo(err))
		return r
	}
	defer conexao.Close()
	ms := time.Since(inicio).Milliseconds()
	_ = conexao.SetReadDeadline(time.Now().Add(prazoDaSaudacao))
	linha, ok := saudacao(conexao)
	if !ok {
		r.respondeu(false, motivoDeNaoSerSsh)
		return r
	}
	r.respondeu(true, "")
	r.Ms, r.Servidor = &ms, ouNada(nomeDoServidor(linha))
	return r
}

// saudacao lê linhas até achar a que começa com "SSH-" (o servidor pode dizer
// outras antes). Devolve essa linha, ou false se não veio.
func saudacao(de io.Reader) (string, bool) {
	leitor := bufio.NewReaderSize(io.LimitReader(de, linhaDeSaudacao*linhasAntesDela), linhaDeSaudacao)
	for i := 0; i < linhasAntesDela; i++ {
		pedaco, err := leitor.ReadSlice('\n')
		linha := strings.TrimRight(string(pedaco), "\r\n")
		if strings.HasPrefix(linha, "SSH-") {
			return linha, true
		}
		// Linha comprida demais para ser uma saudação: o resto dela não conta.
		if errors.Is(err, bufio.ErrBufferFull) {
			for errors.Is(err, bufio.ErrBufferFull) {
				_, err = leitor.ReadSlice('\n')
			}
		}
		if err != nil {
			return "", false
		}
	}
	return "", false
}

// nomeDoServidor tira o nome de uma saudação ("SSH-2.0-OpenSSH_10.0 …"): o
// que vem depois do segundo hífen, só com o que dá para imprimir, até 64.
func nomeDoServidor(linha string) string {
	partes := strings.SplitN(linha, "-", 3)
	if len(partes) < 3 {
		return ""
	}
	return imprimivel(partes[2], nomeDeServidor, false)
}

// imprimivel deixa só os caracteres que dá para mostrar (ASCII; com
// `acentos`, também as letras de fora dele), aparado e cortado em `maximo`.
func imprimivel(texto string, maximo int, acentos bool) string {
	var b strings.Builder
	n := 0
	for _, c := range texto {
		if c < 0x20 || c == 0x7f || (c > 0x7e && (!acentos || c < 0xa0 || c == 0xfffd)) {
			continue
		}
		if n == maximo {
			break
		}
		b.WriteRune(c)
		n++
	}
	return strings.TrimSpace(b.String())
}

// argvDoSalto monta a sonda por salto: `ssh -o BatchMode=yes -o
// ConnectTimeout=4 [-p porta] -W host:porta -- salto`. Só para um salto
// simples ([usuario@]host[:porta]) e um endereço que não tem como virar opção;
// o resto não dá para sondar sem conversa.
func argvDoSalto(ssh, salto, host string, porta int) ([]string, bool) {
	if strings.ContainsAny(salto, ",[]/") || !sshcfg.PalavraSegura(salto) || !sshcfg.PalavraSegura(host) {
		return nil, false
	}
	argv := []string{ssh, "-o", "BatchMode=yes", "-o", "ConnectTimeout=4"}
	if i := strings.LastIndex(salto, ":"); i >= 0 {
		p, err := strconv.Atoi(salto[i+1:])
		if err != nil || p < 1 || p > 65535 || strings.Contains(salto[:i], ":") || i == 0 {
			return nil, false
		}
		argv, salto = append(argv, "-p", strconv.Itoa(p)), salto[:i]
	}
	return append(argv, "-W", net.JoinHostPort(host, strconv.Itoa(porta)), "--", salto), true
}

// sondaPorSalto pede ao ssh um túnel até a porta do host, pelo salto, e ouve a
// saudação que vem por ele. BatchMode: o ssh não pergunta nada (nem senha, nem
// se aceita a chave de um host novo). O processo é morto em seguida.
func sondaPorSalto(h sshcfg.Host, argv []string) sondaJSON {
	r := sondaJSON{Apelido: h.Apelido}
	ctx, cancela := context.WithTimeout(context.Background(), prazoDoSalto)
	defer cancela()
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.WaitDelay = 200 * time.Millisecond
	var reclamou bytes.Buffer
	cmd.Stderr = &limitado{b: &reclamou, max: 16 * 1024}
	// A entrada fica aberta e calada: com ela fechada o ssh encerraria o
	// túnel antes de a saudação chegar.
	entradaDele, err := cmd.StdinPipe()
	if err != nil {
		r.respondeu(false, motivoDoBarrado)
		return r
	}
	saida, err := cmd.StdoutPipe()
	if err != nil {
		r.respondeu(false, motivoDoBarrado)
		return r
	}
	inicio := time.Now()
	if err := cmd.Start(); err != nil {
		r.respondeu(false, motivoDoBarrado)
		return r
	}
	// Quem ouve é outra linha de execução: se um filho do ssh (um ProxyCommand)
	// ficar com o cano aberto depois de o ssh morrer, a sonda não fica presa.
	type ouvido struct {
		linha string
		ok    bool
	}
	veio := make(chan ouvido, 1)
	go func() {
		linha, ok := saudacao(saida)
		veio <- ouvido{linha, ok}
	}()
	var linha string
	var ok bool
	select {
	case o := <-veio:
		linha, ok = o.linha, o.ok
	case <-time.After(prazoDoSalto + 300*time.Millisecond):
	}
	ms := time.Since(inicio).Milliseconds()
	cancela()
	_ = entradaDele.Close()
	_ = cmd.Wait()
	if !ok {
		frase := motivoDoBarrado
		if ultima := ultimaLinha(reclamou.String(), reclamacaoMaxima); ultima != "" {
			frase += ": " + ultima
		}
		r.respondeu(false, frase)
		return r
	}
	r.respondeu(true, "")
	r.Ms, r.Servidor, r.Via = &ms, ouNada(nomeDoServidor(linha)), ouNada(h.Salto)
	return r
}

// limitado guarda até max bytes e joga o resto fora (sem travar quem escreve).
type limitado struct {
	mu  sync.Mutex
	b   *bytes.Buffer
	max int
}

func (l *limitado) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if sobra := l.max - l.b.Len(); sobra > 0 {
		if len(p) < sobra {
			sobra = len(p)
		}
		l.b.Write(p[:sobra])
	}
	return len(p), nil
}

// ultimaLinha devolve a última linha com conteúdo de um texto, só com o que
// dá para mostrar e cortada em `maximo` (com reticências).
func ultimaLinha(texto string, maximo int) string {
	linhas := strings.Split(texto, "\n")
	for i := len(linhas) - 1; i >= 0; i-- {
		limpa := imprimivel(linhas[i], maximo+1, true)
		if limpa == "" {
			continue
		}
		if letras := []rune(limpa); len(letras) > maximo {
			return string(letras[:maximo]) + "…"
		}
		return limpa
	}
	return ""
}

// motivo diz, numa frase curta, por que a porta não respondeu.
func motivo(err error) string {
	var dns *net.DNSError
	var deRede net.Error
	switch {
	case errors.Is(err, syscall.ECONNREFUSED):
		return "recusou a conexão"
	case errors.As(err, &dns):
		return "não achei esse nome"
	case errors.Is(err, syscall.EHOSTUNREACH), errors.Is(err, syscall.ENETUNREACH):
		return "sem rota até lá"
	case errors.As(err, &deRede) && deRede.Timeout():
		return "não respondeu em " + strconv.Itoa(int(prazoDaSonda/time.Second)) + " s"
	}
	return "não deu para conectar"
}
