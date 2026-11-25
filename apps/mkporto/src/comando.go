package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"

	"mkporto/internal/base"
)

func init() { registrar("comando", comandoDeTerminal) }

const usoDoComando = "uso: mkporto-motor comando <conectar <apelido> | chave <apelido> <nome.pub> | gerar <nome> [comentário]>"

// Os terminais que o Mk conhece, na ordem de preferência.
var terminais = []string{"ghostty", "kitty", "foot", "alacritty", "wezterm"}

// comandoDeTerminal: diz o argv que a janela abre num terminal para conectar,
// para enviar uma chave ou para gerar uma. O motor não roda nada disto: quem
// fala com a máquina (e pergunta o que tiver de perguntar) é o ssh, no
// terminal.
func comandoDeTerminal(args []string) int {
	pos, err := posicionais(args)
	if err != nil {
		return base.Falha(base.Erro, "%s", err)
	}
	if len(pos) == 0 {
		return base.Falha(base.Erro, "%s", usoDoComando)
	}
	switch acao := pos[0]; {
	case acao == "conectar" && len(pos) == 2:
		apelido := pos[1]
		if codigo := confereApelido(apelido); codigo != base.Certo {
			return codigo
		}
		return respondeComando(acao, apelido, "ssh · "+apelido, []string{base.Comando("MKPORTO_SSH", "ssh"), "--", apelido})
	case acao == "chave" && len(pos) == 3:
		apelido := pos[1]
		if codigo := confereApelido(apelido); codigo != base.Certo {
			return codigo
		}
		publica, ok := chaveListada(pos[2])
		if !ok {
			return base.Falha(base.Erro, "não há uma chave pública %s em %s", pos[2], base.Til(base.PastaDoSsh()))
		}
		return respondeComando(acao, apelido, "ssh · "+apelido,
			[]string{base.Comando("MKPORTO_COPY_ID", "ssh-copy-id"), "-i", publica, "--", apelido})
	case acao == "gerar" && (len(pos) == 2 || len(pos) == 3):
		nome, comentario := pos[1], ""
		if len(pos) == 3 {
			comentario = pos[2]
		}
		if frase := erroDoNomeDaChave(nome); frase != "" {
			return base.FalhaDeCampo("nome", frase)
		}
		if frase := erroDoComentario(comentario); frase != "" {
			return base.FalhaDeCampo("comentario", frase)
		}
		destino := filepath.Join(base.PastaDoSsh(), nome)
		for _, caminho := range []string{destino, destino + ".pub"} {
			if _, err := os.Lstat(caminho); err == nil {
				return base.FalhaDeCampo("nome", "já existe uma chave com esse nome em "+base.Til(base.PastaDoSsh()))
			}
		}
		if comentario == "" {
			comentario = daqui()
		}
		return respondeComando(acao, "", "ssh-keygen · "+nome,
			[]string{base.Comando("MKPORTO_KEYGEN", "ssh-keygen"), "-t", "ed25519", "-f", destino, "-C", comentario})
	}
	return base.Falha(base.Erro, "%s", usoDoComando)
}

// respondeComando escreve a resposta do `comando`: o argv, o texto de copiar
// (só quando há um apelido) e a linha do terminal.
func respondeComando(acao, apelido, titulo string, argv []string) int {
	// Conectar fica aberto enquanto a sessão durar; os outros acabam sozinhos
	// e o terminal espera, para dar para ler o que o comando disse.
	espera := acao != "conectar"
	resposta := map[string]any{"acao": acao, "apelido": nil, "argv": argv, "texto": nil, "terminal": nil, "aviso": nil}
	if apelido != "" {
		resposta["apelido"], resposta["texto"] = apelido, "ssh "+apelido
	}
	if t := terminal(); t != "" {
		resposta["terminal"] = linhaDoTerminal(t, titulo, argv, espera)
	} else {
		resposta["aviso"] = "não achei um terminal para abrir (ghostty, kitty, foot, alacritty ou wezterm)"
	}
	_ = base.Escreve(resposta)
	return base.Certo
}

// daqui é o comentário padrão de uma chave nova: usuario@maquina.
func daqui() string {
	quem := os.Getenv("USER")
	if u, err := user.Current(); err == nil && u.Username != "" {
		quem = u.Username
	}
	maquina, _ := os.Hostname()
	if comentario := quem + "@" + maquina; quem != "" && maquina != "" && erroDoComentario(comentario) == "" {
		return comentario
	}
	return "mkporto"
}

// chaveListada devolve o caminho de uma chave pública da pasta do ssh, dado o
// nome dela. Só serve um nome (sem pasta) que esteja na lista do `chaves`.
func chaveListada(nome string) (string, bool) {
	if nome == "" || filepath.Base(nome) != nome {
		return "", false
	}
	for _, c := range publicas(false) {
		if c.Nome == nome {
			return filepath.Join(base.PastaDoSsh(), nome), true
		}
	}
	return "", false
}

// terminal escolhe o terminal: o de MKPORTO_TERMINAL, o dos Ajustes do Mk
// ("apps.terminal") ou o primeiro que houver. Só entram os que o Mk conhece.
func terminal() string {
	var ajustes map[string]any
	if b, err := os.ReadFile(base.Config("mk", "ajustes.json")); err == nil {
		_ = json.Unmarshal(b, &ajustes)
	}
	doAjuste, _ := ajustes["apps.terminal"].(string)
	for _, pedido := range append([]string{os.Getenv("MKPORTO_TERMINAL"), doAjuste}, terminais...) {
		for _, t := range terminais {
			if t != pedido {
				continue
			}
			if _, err := exec.LookPath(t); err == nil {
				return t
			}
		}
	}
	return ""
}

// linhaDoTerminal monta o argv de cada terminal: como ele recebe o título e o
// comando. O comando vai sempre como lista, um argumento por item, e os cinco
// terminais a executam direto, sem shell (no ghostty, o -e desliga a expansão
// de shell): um caminho com espaço, aspas ou $ chega inteiro. Embrulhar em
// `sh -c` não ajudaria: num terminal que juntasse os argumentos numa linha, o
// `sh -c exec 'ssh' …` rodaria um `exec` vazio.
func linhaDoTerminal(nome, titulo string, argv []string, espera bool) []string {
	var linha []string
	fica := func(opcao string) []string {
		if espera {
			return []string{opcao}
		}
		return nil
	}
	switch nome {
	case "kitty", "foot":
		linha = append([]string{nome, "--title=" + titulo}, fica("--hold")...)
	case "alacritty":
		linha = append(append([]string{nome, "--title", titulo}, fica("--hold")...), "-e")
	case "wezterm":
		// Sem --always-new-process o comando volta na hora, com a janela
		// nascendo num wezterm já aberto.
		linha = []string{nome, "start", "--always-new-process", "--"}
	default:
		// O ghostty abre com a config do Mk, como no Maré.
		linha = append(append([]string{"ghostty", "--config-default-files=false", "--config-file=?" + base.Config("mk", "ghostty", "config"),
			"--title=" + titulo}, fica("--wait-after-command=true")...), "-e")
	}
	return append(linha, argv...)
}
