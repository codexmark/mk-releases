package sshcfg

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// LinhaDoInclude é a linha que faz o ssh ler o arquivo do app.
const LinhaDoInclude = "Include config.d/mkporto.conf"

const cabecalho = `# Hosts guardados pelo MkPorto (o gerente de conexões do Mk).
# Este arquivo é reescrito inteiro a cada mudança feita por lá: o que for
# escrito aqui à mão se perde. Os seus hosts ficam no ~/.ssh/config.
`

// Entrada é um host do arquivo do app, do jeito que vai escrito.
type Entrada struct {
	Apelido string
	Host    string
	Usuario string
	Porta   int
	Chave   string
	Salto   string
}

// ErroDeCampo é uma recusa da validação: qual campo e por quê.
type ErroDeCampo struct {
	Campo string
	Frase string
}

func (e *ErroDeCampo) Error() string { return e.Frase }

var apelidoBom = regexp.MustCompile(`^[A-Za-z0-9._][A-Za-z0-9._-]*$`)

// ApelidoSeguro diz se o apelido pode ir para uma linha de comando e para o
// config: só letras sem acento, algarismos, ponto, sublinhado e hífen, sem
// começar com hífen, até 64.
func ApelidoSeguro(apelido string) bool {
	return len(apelido) <= 64 && apelidoBom.MatchString(apelido)
}

// palavra confere um valor que vai numa linha do config como uma palavra só:
// sem espaço, sem caractere de controle (nem quebra de linha), sem aspas, sem
// # e sem começar com "-" (viraria opção na linha de comando do ssh).
func palavra(valor, oQue string, maximo int) string {
	switch {
	case strings.HasPrefix(valor, "-"):
		return oQue + " não pode começar com \"-\""
	case len(valor) > maximo:
		return oQue + " é comprido demais (até " + strconv.Itoa(maximo) + ")"
	}
	for _, c := range valor {
		switch {
		case c == '\n' || c == '\r':
			return oQue + " não pode ter quebra de linha"
		case c < 0x20 || c == 0x7f:
			return oQue + " não pode ter caractere de controle"
		case c == ' ' || c == 0xa0:
			return oQue + " não pode ter espaço"
		case c == '"' || c == '#':
			return oQue + " não pode ter " + string(c)
		}
	}
	return ""
}

// Valida confere uma entrada com as regras do contrato. nil se está boa.
func Valida(e Entrada) *ErroDeCampo {
	erro := func(campo, frase string) *ErroDeCampo { return &ErroDeCampo{Campo: campo, Frase: frase} }
	switch {
	case e.Apelido == "":
		return erro("apelido", "falta o apelido")
	case strings.HasPrefix(e.Apelido, "-"):
		return erro("apelido", "o apelido não pode começar com \"-\"")
	case strings.ContainsAny(e.Apelido, "\n\r"):
		return erro("apelido", "o apelido não pode ter quebra de linha")
	case len(e.Apelido) > 64:
		return erro("apelido", "o apelido é comprido demais (até 64)")
	case !ApelidoSeguro(e.Apelido):
		return erro("apelido", "o apelido só pode ter letras sem acento, algarismos, ponto, hífen e sublinhado")
	case e.Host == "":
		return erro("host", "falta o endereço")
	}
	if f := palavra(e.Host, "o endereço", 253); f != "" {
		return erro("host", f)
	}
	if f := palavra(e.Usuario, "o usuário", 64); f != "" {
		return erro("usuario", f)
	}
	if e.Porta < 1 || e.Porta > 65535 {
		return erro("porta", "a porta vai de 1 a 65535")
	}
	switch {
	case strings.HasPrefix(e.Chave, "-"):
		return erro("chave", "a chave não pode começar com \"-\"")
	case len(e.Chave) > 4096:
		return erro("chave", "o caminho da chave é comprido demais")
	}
	for _, c := range e.Chave {
		switch {
		case c == '\n' || c == '\r':
			return erro("chave", "a chave não pode ter quebra de linha")
		case c < 0x20 || c == 0x7f:
			return erro("chave", "a chave não pode ter caractere de controle")
		case c == '"' || c == '\\':
			return erro("chave", "a chave não pode ter "+string(c))
		}
	}
	if f := palavra(e.Salto, "o salto", 1024); f != "" {
		return erro("salto", f)
	}
	return nil
}

// Texto escreve o arquivo do app inteiro. As entradas já têm de estar
// validadas (Grava confere de novo).
func Texto(entradas []Entrada) []byte {
	var b bytes.Buffer
	b.WriteString(cabecalho)
	for _, e := range entradas {
		fmt.Fprintf(&b, "\nHost %s\n    HostName %s\n", e.Apelido, e.Host)
		if e.Usuario != "" {
			fmt.Fprintf(&b, "    User %s\n", e.Usuario)
		}
		fmt.Fprintf(&b, "    Port %d\n", e.Porta)
		if e.Chave != "" {
			fmt.Fprintf(&b, "    IdentityFile \"%s\"\n", e.Chave)
		}
		if e.Salto != "" {
			fmt.Fprintf(&b, "    ProxyJump %s\n", e.Salto)
		}
	}
	return b.Bytes()
}

// DoApp lê as entradas do arquivo do app: só o que está escrito nele, sem
// completar com os curingas do config da pessoa.
func DoApp(arquivo string) []Entrada {
	entradas := []Entrada{}
	visto := map[string]bool{}
	l := LeSo(arquivo)
	for i := range l.Blocos {
		b := &l.Blocos[i]
		if !b.Declarado {
			continue
		}
		valor := func(chave string) string {
			for _, o := range b.Opcoes {
				if o.Chave == chave && len(o.Args) > 0 {
					return o.Args[0]
				}
			}
			return ""
		}
		for _, apelido := range b.Padroes {
			if Curinga(apelido) || visto[apelido] {
				continue
			}
			visto[apelido] = true
			e := Entrada{Apelido: apelido, Host: valor("hostname"), Usuario: valor("user"), Porta: 22,
				Chave: valor("identityfile"), Salto: valor("proxyjump")}
			if e.Host == "" {
				e.Host = apelido
			}
			if p, err := strconv.Atoi(valor("port")); err == nil && p >= 1 && p <= 65535 {
				e.Porta = p
			}
			entradas = append(entradas, e)
		}
	}
	return entradas
}

// Grava reescreve o arquivo do app com estas entradas: de forma atômica, com
// permissão 600, numa pasta config.d com 700 (e o ~/.ssh com 700, se for
// preciso criá-lo). Uma entrada inválida não é escrita: nada muda.
func Grava(dir string, entradas []Entrada) error {
	for _, e := range entradas {
		if err := Valida(e); err != nil {
			return fmt.Errorf("o host %q não pode ser gravado: %s", e.Apelido, err.Frase)
		}
	}
	if err := pastaFechada(dir); err != nil {
		return err
	}
	if err := pastaFechada(filepath.Join(dir, "config.d")); err != nil {
		return err
	}
	return escreveAtomico(ArquivoDoApp(dir), Texto(entradas), 0o600)
}

// pastaFechada cria a pasta com 700 se ela não existe. Uma que já existe fica
// como a pessoa deixou.
func pastaFechada(caminho string) error {
	info, err := os.Stat(caminho)
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("%s existe e não é uma pasta", caminho)
		}
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("não consegui olhar %s: %v", caminho, err)
	}
	if err := os.Mkdir(caminho, 0o700); err != nil {
		return fmt.Errorf("não consegui criar %s: %v", caminho, err)
	}
	// O Mkdir obedece à umask: o 700 é posto de novo, por extenso.
	return os.Chmod(caminho, 0o700)
}

// escreveAtomico grava num temporário na mesma pasta e troca de nome: quem lê
// vê o arquivo antigo ou o novo, nunca a metade.
func escreveAtomico(caminho string, dados []byte, modo os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(caminho), ".mkporto-*.tmp")
	if err != nil {
		return fmt.Errorf("não consegui gravar em %s: %v", filepath.Dir(caminho), err)
	}
	nome := tmp.Name()
	limpa := func(err error) error {
		tmp.Close()
		os.Remove(nome)
		return fmt.Errorf("não consegui gravar %s: %v", caminho, err)
	}
	if err := tmp.Chmod(modo); err != nil {
		return limpa(err)
	}
	if _, err := tmp.Write(dados); err != nil {
		return limpa(err)
	}
	if err := tmp.Sync(); err != nil {
		return limpa(err)
	}
	if err := tmp.Close(); err != nil {
		return limpa(err)
	}
	if err := os.Rename(nome, caminho); err != nil {
		return limpa(err)
	}
	return nil
}

// GaranteInclude põe a linha do Include no topo de dir/config, se nenhum
// Include de antes do primeiro Host já cobre o arquivo do app. O resto do
// arquivo fica como estava, byte a byte. Devolve se a linha foi posta agora.
func GaranteInclude(dir, casa string) (posto bool, err error) {
	config := filepath.Join(dir, "config")
	// Um config que é atalho (dotfiles) continua atalho: reescreve-se o alvo.
	if info, err := os.Lstat(config); err == nil && info.Mode()&os.ModeSymlink != 0 {
		alvo, err := filepath.EvalSymlinks(config)
		if err != nil {
			return false, fmt.Errorf("o %s é um atalho para um arquivo que não existe", config)
		}
		config = alvo
	}
	antes, err := os.ReadFile(config)
	modo := os.FileMode(0o600)
	switch {
	case errors.Is(err, os.ErrNotExist):
		antes = nil
	case err != nil:
		return false, fmt.Errorf("não consegui ler %s: %v", config, err)
	default:
		if info, err := os.Stat(config); err == nil {
			modo = info.Mode().Perm()
		}
	}
	if cobre(antes, dir, casa) {
		return false, nil
	}
	novo := []byte(LinhaDoInclude + "\n")
	if len(antes) > 0 {
		novo = append(append(novo, '\n'), antes...)
	}
	if err := pastaFechada(dir); err != nil {
		return false, err
	}
	if err := escreveAtomico(config, novo, modo); err != nil {
		return false, err
	}
	return true, nil
}

// cobre diz se, antes do primeiro Host ou Match, há um Include que alcança o
// arquivo do app (um Include depois disso só valeria para aquele bloco).
func cobre(config []byte, dir, casa string) bool {
	app := ArquivoDoApp(dir)
	for _, linha := range strings.Split(string(config), "\n") {
		chave, args := palavras(linha)
		switch chave {
		case "host", "match":
			return false
		case "include":
			for _, a := range args {
				padrao := Resolve(a, dir, casa)
				if padrao == "" {
					continue
				}
				if ok, err := filepath.Match(padrao, app); err == nil && ok {
					return true
				}
			}
		}
	}
	return false
}

// PalavraSegura diz se um valor (um endereço, um salto) pode ir como um
// argumento só numa linha de comando do ssh: não é vazio, não começa com "-"
// e não tem espaço, caractere de controle, aspas nem #.
func PalavraSegura(valor string) bool {
	return valor != "" && palavra(valor, "", 1024) == ""
}
