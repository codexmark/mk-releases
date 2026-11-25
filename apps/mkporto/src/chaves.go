package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"mkporto/internal/base"
)

func init() { registrar("chaves", chaves) }

// Uma chave pública é uma linha; mais que isto não é uma.
const publicaMaxima = 64 * 1024

// chaveJSON é uma chave pública de ~/.ssh.
type chaveJSON struct {
	Nome       string  `json:"nome"`
	Arquivo    string  `json:"arquivo"`
	Tipo       string  `json:"tipo"`
	Bits       *int    `json:"bits"`
	Impressao  *string `json:"impressao"`
	Comentario *string `json:"comentario"`
	Publica    string  `json:"publica"`
}

// chaves: as chaves públicas da pasta do ssh.
func chaves(args []string) int {
	if pos, err := posicionais(args); err != nil {
		return base.Falha(base.Erro, "%s", err)
	} else if len(pos) != 0 {
		return base.Falha(base.Erro, "uso: mkporto-motor chaves")
	}
	_ = base.Escreve(map[string]any{"chaves": publicas(true)})
	return base.Certo
}

// publicas lista os *.pub da pasta do ssh (menos os certificados), em ordem
// de nome. Com `impressoes`, pergunta a impressão digital ao ssh-keygen.
func publicas(impressoes bool) []chaveJSON {
	lista := []chaveJSON{}
	dir := base.PastaDoSsh()
	itens, err := os.ReadDir(dir)
	if err != nil {
		return lista
	}
	nomes := []string{}
	for _, item := range itens {
		if nome := item.Name(); strings.HasSuffix(nome, ".pub") && !strings.HasSuffix(nome, "-cert.pub") {
			nomes = append(nomes, nome)
		}
	}
	sort.Strings(nomes)
	for _, nome := range nomes {
		caminho := filepath.Join(dir, nome)
		linha, ok := lePublica(caminho)
		if !ok {
			continue
		}
		partes := strings.Fields(linha)
		if len(partes) < 2 {
			continue
		}
		miolo, err := base64.StdEncoding.DecodeString(partes[1])
		if err != nil || len(miolo) == 0 {
			continue
		}
		c := chaveJSON{Nome: nome, Arquivo: base.Til(caminho), Tipo: tipoDaChave(partes[0]),
			Comentario: ouNada(strings.Join(partes[2:], " ")), Publica: linha}
		if impressoes {
			c.Bits, c.Impressao = impressao(caminho)
			if c.Impressao == nil {
				soma := sha256.Sum256(miolo)
				c.Impressao = ouNada("SHA256:" + base64.RawStdEncoding.EncodeToString(soma[:]))
			}
		}
		lista = append(lista, c)
	}
	return lista
}

// lePublica lê a primeira linha com conteúdo de um arquivo .pub. É a única
// leitura de chave do motor, e só abre o que termina em ".pub": a chave
// privada, ao lado, nunca é aberta.
func lePublica(caminho string) (string, bool) {
	if !strings.HasSuffix(caminho, ".pub") {
		return "", false
	}
	info, err := os.Stat(caminho)
	if err != nil || !info.Mode().IsRegular() || info.Size() == 0 || info.Size() > publicaMaxima {
		return "", false
	}
	f, err := os.Open(caminho)
	if err != nil {
		return "", false
	}
	defer f.Close()
	leitor := bufio.NewScanner(f)
	leitor.Buffer(make([]byte, 0, 4096), publicaMaxima)
	for leitor.Scan() {
		linha := strings.TrimSpace(leitor.Text())
		if linha == "" || strings.HasPrefix(linha, "#") {
			continue
		}
		for _, c := range linha {
			if c < 0x20 || c == 0x7f {
				return "", false
			}
		}
		return linha, true
	}
	return "", false
}

// tipoDaChave troca o nome do algoritmo pelo nome curto que a pessoa conhece.
func tipoDaChave(algoritmo string) string {
	switch {
	case algoritmo == "ssh-ed25519":
		return "ed25519"
	case algoritmo == "ssh-rsa":
		return "rsa"
	case algoritmo == "ssh-dss":
		return "dsa"
	case strings.HasPrefix(algoritmo, "ecdsa-sha2-"):
		return "ecdsa"
	case strings.HasPrefix(algoritmo, "sk-ssh-ed25519@"):
		return "ed25519-sk"
	case strings.HasPrefix(algoritmo, "sk-ecdsa-sha2-"):
		return "ecdsa-sk"
	}
	return algoritmo
}

// impressao pergunta ao ssh-keygen o tamanho e a impressão digital de uma
// chave pública ("256 SHA256:… comentário (ED25519)"). Sem o ssh-keygen, ou
// se ele não souber, devolve nil.
func impressao(publica string) (bits *int, digital *string) {
	keygen, err := exec.LookPath(base.Comando("MKPORTO_KEYGEN", "ssh-keygen"))
	if err != nil {
		return nil, nil
	}
	ctx, cancela := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancela()
	cmd := exec.CommandContext(ctx, keygen, "-l", "-f", publica)
	cmd.WaitDelay = time.Second
	saida, err := cmd.Output()
	if err != nil {
		return nil, nil
	}
	partes := strings.Fields(strings.SplitN(string(saida), "\n", 2)[0])
	if len(partes) < 2 || !strings.Contains(partes[1], ":") {
		return nil, nil
	}
	if n, err := strconv.Atoi(partes[0]); err == nil && n > 0 {
		bits = &n
	}
	return bits, &partes[1]
}

var nomeDeChaveBom = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9._-]*$`)

// erroDoNomeDaChave confere o nome de uma chave nova: vira um arquivo na
// pasta do ssh e um argumento do ssh-keygen. "" se serve.
func erroDoNomeDaChave(nome string) string {
	switch {
	case nome == "":
		return "falta o nome da chave"
	case strings.HasPrefix(nome, "-"):
		return "o nome não pode começar com \"-\""
	case strings.HasPrefix(nome, "."):
		return "o nome não pode começar com ponto"
	case strings.ContainsAny(nome, "\n\r"):
		return "o nome não pode ter quebra de linha"
	case len(nome) > 64:
		return "o nome é comprido demais (até 64)"
	case !nomeDeChaveBom.MatchString(nome):
		return "o nome só pode ter letras sem acento, algarismos, ponto, hífen e sublinhado"
	case strings.HasSuffix(nome, ".pub"):
		return "o nome não leva .pub: esse é o da metade pública, que nasce junto"
	}
	return ""
}

// erroDoComentario confere o comentário de uma chave nova (pode ser vazio).
func erroDoComentario(comentario string) string {
	if strings.HasPrefix(comentario, "-") {
		return "o comentário não pode começar com \"-\""
	}
	if utf8.RuneCountInString(comentario) > 128 {
		return "o comentário é comprido demais (até 128)"
	}
	for _, c := range comentario {
		switch {
		case c == '\n' || c == '\r':
			return "o comentário não pode ter quebra de linha"
		case c < 0x20 || c == 0x7f:
			return "o comentário não pode ter caractere de controle"
		}
	}
	return ""
}
