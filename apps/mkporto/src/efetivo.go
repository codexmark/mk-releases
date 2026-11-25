package main

import (
	"context"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"mkporto/internal/base"
	"mkporto/internal/sshcfg"
)

// Quanto se espera pelo `ssh -G` de um host, e quantos se pergunta de uma vez.
var prazoDoG = 3 * time.Second

const perguntasDeUmaVez = 8

// As chaves que o ssh tenta quando o config não diz nenhuma: o `ssh -G` as
// lista sempre, e elas não são uma escolha do config.
var chavesPadrao = []string{"id_rsa", "id_ecdsa", "id_ecdsa_sk", "id_ed25519", "id_ed25519_sk", "id_xmss", "id_dsa"}

// oSsh acha o ssh (MKPORTO_SSH, ou o do PATH).
func oSsh() (string, bool) {
	caminho, err := exec.LookPath(base.Comando("MKPORTO_SSH", "ssh"))
	return caminho, err == nil
}

// efetiva troca os valores de cada host pelos que o próprio ssh vê
// (`ssh -G -- apelido`: com Match e com o config do sistema). Devolve, para
// cada host, se o ssh respondeu; quem ficou sem resposta continua com os
// valores do leitor próprio. Só um apelido seguro vai para a linha de comando.
func efetiva(hosts []sshcfg.Host) []bool {
	respondeu := make([]bool, len(hosts))
	ssh, tem := oSsh()
	if !tem {
		return respondeu
	}
	var espera sync.WaitGroup
	vagas := make(chan struct{}, perguntasDeUmaVez)
	for i := range hosts {
		if !sshcfg.ApelidoSeguro(hosts[i].Apelido) {
			continue
		}
		espera.Add(1)
		go func(i int) {
			defer espera.Done()
			vagas <- struct{}{}
			defer func() { <-vagas }()
			saida, ok := perguntaAoSsh(ssh, hosts[i].Apelido)
			if !ok {
				return
			}
			visto, ok := aplicaG(hosts[i].Apelido, hosts[i].Chave, saida)
			if !ok {
				return
			}
			hosts[i].Host, hosts[i].Usuario, hosts[i].Porta = visto.Host, visto.Usuario, visto.Porta
			hosts[i].Chave, hosts[i].Salto = visto.Chave, visto.Salto
			respondeu[i] = true
		}(i)
	}
	espera.Wait()
	return respondeu
}

// perguntaAoSsh roda `ssh -G -- apelido`: só lê a configuração, não conecta.
func perguntaAoSsh(ssh, apelido string) (string, bool) {
	ctx, cancela := context.WithTimeout(context.Background(), prazoDoG)
	defer cancela()
	cmd := exec.CommandContext(ctx, ssh, "-G", "--", apelido)
	cmd.WaitDelay = 200 * time.Millisecond
	saida, err := cmd.Output()
	if err != nil {
		return "", false
	}
	return string(saida), true
}

// aplicaG lê a saída do `ssh -G` ("chave valor" por linha; a primeira de cada
// chave ganha) e devolve os valores que o MkPorto mostra. `chaveDoLeitor` é o
// IdentityFile que o leitor próprio achou: sem ele, uma chave padrão na saída
// não conta. Sem `hostname`, aquilo não é a saída do ssh -G.
func aplicaG(apelido, chaveDoLeitor, saida string) (sshcfg.Host, bool) {
	achado := map[string]string{}
	for _, linha := range strings.Split(saida, "\n") {
		partes := strings.Fields(linha)
		if len(partes) < 2 {
			continue
		}
		chave := strings.ToLower(partes[0])
		if _, ja := achado[chave]; !ja {
			achado[chave] = strings.Join(partes[1:], " ")
		}
	}
	h := sshcfg.Host{Apelido: apelido, Host: achado["hostname"], Usuario: achado["user"], Porta: 22}
	if h.Host == "" {
		return h, false
	}
	if p, err := strconv.Atoi(achado["port"]); err == nil && p >= 1 && p <= 65535 {
		h.Porta = p
	}
	if chave := achado["identityfile"]; chaveDoLeitor != "" || !chavePadrao(chave) {
		h.Chave = chave
	}
	if salto := achado["proxyjump"]; !strings.EqualFold(salto, "none") {
		h.Salto = salto
	}
	return h, true
}

// chavePadrao diz se o caminho é uma das chaves que o ssh tenta por conta
// própria (~/.ssh/id_ed25519 e as irmãs), escrito com til ou por extenso.
func chavePadrao(caminho string) bool {
	for _, nome := range chavesPadrao {
		if caminho == "~/.ssh/"+nome || caminho == filepath.Join(base.Home(), ".ssh", nome) {
			return true
		}
	}
	return false
}
