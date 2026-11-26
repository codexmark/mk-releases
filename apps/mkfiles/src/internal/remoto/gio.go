package remoto

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"mkfiles/internal/base"
)

// Prazo é quanto o `gio mount` pode ficar calado (sem acabar e sem fazer uma
// pergunta que o motor reconheça) antes de ser morto (MKFILES_REMOTO_MS;
// padrão 30 s). O tempo em que a pessoa pensa na resposta não conta.
func Prazo() time.Duration {
	if n, err := strconv.Atoi(os.Getenv("MKFILES_REMOTO_MS")); err == nil && n >= 50 {
		return time.Duration(n) * time.Millisecond
	}
	return 30 * time.Second
}

// sossego é quanto o motor espera a saída do gio parar antes de tratar um
// resto sem quebra de linha como pergunta de tipo desconhecido.
var sossego = 400 * time.Millisecond

// Os tipos de pergunta.
const (
	pUsuario   = "usuario"
	pSenha     = "senha"
	pDominio   = "dominio"
	pConfirmar = "confirmar"
	pOutro     = "outro"
)

var (
	// As palavras que denunciam cada pergunta, em inglês e em português. A
	// senha vem antes: "Password for user ana" é pergunta de senha.
	palavras = []struct {
		tipo  string
		deles []string
	}{
		{pSenha, []string{"password", "passphrase", "senha", "palavra-passe", "palavra passe", "frase secreta", "frase-senha"}},
		{pDominio, []string{"domain", "domínio", "dominio"}},
		{pUsuario, []string{"user", "usuário", "usuario", "utilizador", "login"}},
		{pConfirmar, []string{"choice", "escolha", "opção", "opcao", "yes/no", "y/n", "sim/não", "sim/nao", "s/n"}},
	}
	entreColchetes = regexp.MustCompile(`\[([^\[\]]*)\]\s*:?\s*$`)
)

// classifica olha o fim do que o gio escreveu: se é uma pergunta esperando
// resposta (um resto de linha, sem quebra, acabando em ":" ou "?"), diz o
// tipo, o texto para mostrar e o valor padrão ("User [ana]: " → ana). quieto:
// a saída parou há um tempo, então um resto qualquer também conta.
func classifica(pendente string, quieto bool) (tipo, texto, padrao string, ePergunta bool) {
	i := strings.LastIndexByte(pendente, '\n')
	resto := strings.TrimRight(pendente[i+1:], " \t\r")
	if resto == "" {
		return "", "", "", false
	}
	fim := resto[len(resto)-1]
	if fim != ':' && fim != '?' && !quieto {
		return "", "", "", false
	}
	minusculo := strings.ToLower(resto)
	tipo = pOutro
	for _, p := range palavras {
		for _, palavra := range p.deles {
			if strings.Contains(minusculo, palavra) {
				tipo = p.tipo
				break
			}
		}
		if tipo != pOutro {
			break
		}
	}
	if tipo == pOutro && fim != ':' && fim != '?' {
		return "", "", "", false
	}
	texto = resto
	if tipo == pConfirmar || tipo == pOutro {
		// A explicação e as escolhas vêm nas linhas de cima: vão junto.
		texto = strings.TrimSpace(pendente)
	} else if m := entreColchetes.FindStringSubmatch(resto); m != nil {
		padrao = m[1]
	}
	return tipo, texto, padrao, true
}

// pergunta é uma pergunta do gio esperando a pessoa.
type pergunta struct {
	numero   int
	resposta chan *string // nil: a pessoa desistiu
}

// montagem é um `gio mount` em andamento.
type montagem struct {
	trava   sync.Mutex
	aberta  *pergunta
	numero  int
	segredo []string // o que foi respondido às perguntas de senha
}

func (m *montagem) abre() *pergunta {
	m.trava.Lock()
	defer m.trava.Unlock()
	m.numero++
	m.aberta = &pergunta{numero: m.numero, resposta: make(chan *string, 1)}
	return m.aberta
}

// responde entrega a resposta à pergunta aberta, se o número bate.
func (m *montagem) responde(numero int, resposta *string) bool {
	m.trava.Lock()
	defer m.trava.Unlock()
	if m.aberta == nil || m.aberta.numero != numero {
		return false
	}
	m.aberta.resposta <- resposta
	m.aberta = nil
	return true
}

// semSegredo tira do texto qualquer senha respondida: o que o gio escreveu vai
// para a janela, e nenhuma senha pode ir junto.
func (m *montagem) semSegredo(texto string) string {
	m.trava.Lock()
	defer m.trava.Unlock()
	for _, s := range m.segredo {
		if s != "" {
			texto = strings.ReplaceAll(texto, s, "•••")
		}
	}
	return texto
}

// resultado de uma montagem.
type resultado struct {
	saida  string
	motivo string // "" se montou
	frase  string
}

// monta roda `gio mount <uri>` conversando: lê a saída sem esperar quebra de
// linha, transforma cada pergunta em evento (pelo avisa) e escreve a resposta
// no stdin. O endereço já foi conferido (começa com um esquema conhecido: não
// tem como virar opção do gio). A senha só passa pelo stdin.
func (m *montagem) monta(uri string, anonimo bool, parou <-chan struct{}, avisa func(numero int, tipo, texto, padrao string, deNovo bool)) resultado {
	argv := []string{"mount"}
	if anonimo {
		argv = append(argv, "-a")
	}
	argv = append(argv, uri)
	ctx, mata := context.WithCancel(context.Background())
	defer mata()
	cmd := exec.CommandContext(ctx, base.Comando("MKFILES_GIO", "gio"), argv...)
	// As perguntas saem na língua do sistema; o motor entende inglês e português.
	cmd.WaitDelay = 2 * time.Second
	entrada, err := cmd.StdinPipe()
	if err != nil {
		return resultado{motivo: "erro", frase: err.Error()}
	}
	leitura, escrita, err := os.Pipe()
	if err != nil {
		return resultado{motivo: "erro", frase: err.Error()}
	}
	cmd.Stdout, cmd.Stderr = escrita, escrita
	if err := cmd.Start(); err != nil {
		leitura.Close()
		escrita.Close()
		if errors.Is(err, exec.ErrNotFound) || errors.Is(err, os.ErrNotExist) {
			return resultado{motivo: "falta", frase: "falta o gio (pacotes glib2 e gvfs) para conectar a servidores"}
		}
		return resultado{motivo: "erro", frase: "não deu para rodar o gio: " + err.Error()}
	}
	escrita.Close()

	pedacos := make(chan []byte)
	go func() {
		defer close(pedacos)
		defer leitura.Close()
		buf := make([]byte, 4096)
		for {
			n, err := leitura.Read(buf)
			if n > 0 {
				pedacos <- append([]byte(nil), buf[:n]...)
			}
			if err != nil {
				return
			}
		}
	}()

	var tudo, pendente bytes.Buffer
	senhas, desistiu, estourou := 0, false, false
	prazo := time.NewTimer(Prazo())
	defer prazo.Stop()
	calma := time.NewTimer(time.Hour)
	calma.Stop()
	defer calma.Stop()

	// pergunta à janela e espera; devolve false se é para parar.
	pergunta := func(tipo, texto, padrao string) bool {
		prazo.Stop()
		p := m.abre()
		if tipo == pSenha {
			senhas++
		}
		avisa(p.numero, tipo, m.semSegredo(texto), padrao, tipo == pSenha && senhas > 1)
		var resposta *string
		select {
		case resposta = <-p.resposta:
		case <-parou:
			desistiu = true
			return false
		}
		if resposta == nil {
			desistiu = true
			return false
		}
		if tipo == pSenha {
			m.trava.Lock()
			m.segredo = append(m.segredo, *resposta)
			m.trava.Unlock()
		}
		// Uma linha só: uma quebra no meio da resposta responderia à pergunta seguinte.
		linha := strings.NewReplacer("\n", "", "\r", "").Replace(*resposta)
		if tipo == pConfirmar {
			var segue bool
			if linha, segue = escolha(texto, linha); !segue {
				desistiu = true
				return false
			}
		}
		if _, err := io.WriteString(entrada, linha+"\n"); err != nil {
			return false
		}
		pendente.Reset()
		prazo.Reset(Prazo())
		return true
	}

laco:
	for {
		select {
		case pedaco, aberto := <-pedacos:
			if !aberto {
				break laco
			}
			tudo.Write(pedaco)
			pendente.Write(pedaco)
			calma.Stop()
			if tipo, texto, padrao, ok := classifica(pendente.String(), false); ok && tipo != pOutro {
				if !pergunta(tipo, texto, padrao) {
					break laco
				}
			} else {
				calma.Reset(sossego)
			}
		case <-calma.C:
			// A saída parou num resto sem quebra de linha: é pergunta, mesmo
			// que o motor não saiba de quê.
			if tipo, texto, padrao, ok := classifica(pendente.String(), true); ok {
				if !pergunta(tipo, texto, padrao) {
					break laco
				}
			}
		case <-prazo.C:
			estourou = true
			break laco
		case <-parou:
			desistiu = true
			break laco
		}
	}
	entrada.Close()
	if desistiu || estourou {
		mata()
	}
	erro := cmd.Wait()
	// O que sobrou no cano depois do fim.
	// (Com limite: um filho do gio que herdou o cano não segura o motor.)
	sobra := time.NewTimer(2 * time.Second)
	defer sobra.Stop()
resto:
	for {
		select {
		case pedaco, aberto := <-pedacos:
			if !aberto {
				break resto
			}
			tudo.Write(pedaco)
		case <-sobra.C:
			break resto
		}
	}
	saida := m.semSegredo(strings.TrimSpace(tudo.String()))
	if len(saida) > 4000 {
		saida = saida[len(saida)-4000:]
	}
	switch {
	case desistiu:
		return resultado{saida: saida, motivo: "recusado", frase: "a conexão foi cancelada"}
	case estourou:
		return resultado{saida: saida, motivo: "tempo", frase: "o servidor não respondeu a tempo"}
	case erro == nil:
		return resultado{saida: saida}
	}
	minusculo := strings.ToLower(saida)
	for _, pista := range []string{"password", "senha", "authentication", "autentica", "permission denied", "permissão negada", "acesso negado", "access denied", "login"} {
		if senhas > 0 && strings.Contains(minusculo, pista) || senhas > 1 {
			return resultado{saida: saida, motivo: "senha", frase: "o servidor não aceitou o usuário ou a senha"}
		}
	}
	frase := "não deu para conectar"
	if saida != "" {
		ultima := saida[strings.LastIndexByte(saida, '\n')+1:]
		frase = fmt.Sprintf("não deu para conectar: %s", strings.TrimSpace(ultima))
	}
	return resultado{saida: saida, motivo: "erro", frase: frase}
}

// roda um `gio` curto (listar a rede, desmontar), com prazo, sem conversa.
func roda(prazo time.Duration, args ...string) (string, error) {
	ctx, solta := context.WithTimeout(context.Background(), prazo)
	defer solta()
	cmd := exec.CommandContext(ctx, base.Comando("MKFILES_GIO", "gio"), args...)
	cmd.WaitDelay = time.Second
	var saida bytes.Buffer
	cmd.Stdout, cmd.Stderr = &saida, &saida
	err := cmd.Run()
	return strings.TrimSpace(saida.String()), err
}
