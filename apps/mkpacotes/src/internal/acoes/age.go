package acoes

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// Quanto esperar pelo resto da saída depois de o comando terminar.
var esperaDoCano = 2 * time.Second

var verbos = map[string]string{"instalar": "instalar", "remover": "remover", "orfaos": "remover os órfãos",
	"cache": "limpar o cache", "atualizar": "atualizar"}

// Age executa uma ação: confere as recusas, refaz o ensaio (quem chama não é
// de confiança) e roda os comandos do plano, na ordem, passando a `emite`
// cada linha que eles escrevem. Para no primeiro comando que falhar.
func Age(acao string, nomes []string, mesmoAssim bool, emite func(linha string)) (*Plano, error) {
	if _, err := limpa(nomes); err != nil {
		return nil, err
	}
	if verbos[acao] == "" {
		return nil, fmt.Errorf("não conheço a ação %q (as que há: %s)", acao, strings.Join(Acoes(), ", "))
	}
	if Euid() == 0 {
		return nil, recusa("o MkPacotes não roda como root: abra como você, e a senha é pedida na hora")
	}
	// A atualização é do mk-update, que sabe lidar com a trava do pacman
	// (ele oferece tirá-la). O resto não roda com ela.
	if acao != "atualizar" && Travado() {
		return nil, recusaTravado()
	}
	solta, err := Tranca()
	if err != nil {
		return nil, err
	}
	defer solta()

	p, err := Planeja(acao, nomes, mesmoAssim)
	if err != nil {
		return nil, err
	}
	if p.NadaAFazer {
		for _, a := range p.Avisos {
			emite(a)
		}
		emite("nada a fazer")
		return p, nil
	}
	// Tudo o que vai rodar tem de existir antes do primeiro comando.
	for _, c := range p.Comandos {
		if _, err := exec.LookPath(c[0]); err != nil {
			return nil, fmt.Errorf("não achei o %s nesta máquina", c[0])
		}
	}
	for _, c := range p.Comandos {
		emite("$ " + strings.Join(c, " "))
		if acao == "atualizar" {
			if err := abre(c); err != nil {
				return nil, err
			}
			emite("o mk-update terminou")
			continue
		}
		ditas, codigo, err := transmite(c, emite)
		if err != nil {
			return nil, err
		}
		if codigo != 0 {
			return nil, errors.New(fraseDeFalha(acao, c[0], codigo, ditas))
		}
	}
	return p, nil
}

func fraseDeFalha(acao, comando string, codigo int, ditas []string) string {
	// O pkexec sai com 126 quando o pedido de senha é fechado e com 127
	// quando a senha não confere ou a pessoa não pode.
	if comando == "pkexec" && (codigo == 126 || codigo == 127) {
		return "a senha não foi dada: nada foi mudado"
	}
	motivo := ""
	for _, l := range ditas {
		if strings.HasPrefix(l, "error:") || strings.HasPrefix(l, "erro:") {
			motivo = l
			break
		}
	}
	if motivo == "" && len(ditas) > 0 {
		motivo = ditas[len(ditas)-1]
	}
	if motivo == "" {
		motivo = fmt.Sprintf("o %s saiu com o código %d", comando, codigo)
	}
	return "não deu para " + verbos[acao] + ": " + motivo
}

var corDeTerminal = regexp.MustCompile("\x1b\\[[0-9;?]*[A-Za-z]")

// transmite roda um comando com o stdout e o stderr no mesmo cano e entrega
// cada linha (quebrada em \n ou em \r, sem as cores) a `emite`. Devolve as
// linhas ditas e o código de saída.
func transmite(args []string, emite func(string)) (ditas []string, codigo int, err error) {
	leitura, escrita, err := os.Pipe()
	if err != nil {
		return nil, 0, fmt.Errorf("não consegui rodar o %s: %v", args[0], err)
	}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout, cmd.Stderr = escrita, escrita
	if err := cmd.Start(); err != nil {
		leitura.Close()
		escrita.Close()
		return nil, 0, fmt.Errorf("não consegui rodar o %s: %v", args[0], err)
	}
	escrita.Close()
	// Quem lê é outra linha de execução: se um neto do comando ficar com o
	// cano aberto (um daemon que um gancho subiu), a leitura não prende a ação
	// depois de o comando terminar.
	lido := make(chan struct{})
	go func() {
		defer close(lido)
		leitor := bufio.NewReaderSize(leitura, 64*1024)
		var linha strings.Builder
		solta := func() {
			l := strings.TrimSpace(corDeTerminal.ReplaceAllString(linha.String(), ""))
			linha.Reset()
			if l == "" {
				return
			}
			if letras := []rune(l); len(letras) > 2000 {
				l = string(letras[:2000]) + "…"
			}
			ditas = append(ditas, l)
			emite(l)
		}
		for {
			c, _, err := leitor.ReadRune()
			if err != nil {
				break
			}
			if c == '\n' || c == '\r' {
				solta()
				continue
			}
			if linha.Len() < 16*1024 {
				linha.WriteRune(c)
			}
		}
		solta()
	}()
	err = cmd.Wait()
	select {
	case <-lido:
	case <-time.After(esperaDoCano):
	}
	leitura.Close()
	<-lido
	var saiu *exec.ExitError
	if errors.As(err, &saiu) {
		return ditas, saiu.ExitCode(), nil
	}
	if err != nil {
		return ditas, 0, fmt.Errorf("não consegui rodar o %s: %v", args[0], err)
	}
	return ditas, 0, nil
}

// abre roda o terminal da atualização e espera a janela dele fechar. O que o
// mk-update escreve fica no terminal, não aqui.
func abre(args []string) error {
	cmd := exec.Command(args[0], args[1:]...)
	err := cmd.Run()
	var saiu *exec.ExitError
	if errors.As(err, &saiu) {
		return fmt.Errorf("o terminal da atualização fechou com erro (código %d)", saiu.ExitCode())
	}
	if err != nil {
		return fmt.Errorf("não consegui abrir o %s: %v", args[0], err)
	}
	return nil
}
