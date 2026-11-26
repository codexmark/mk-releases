// Package remoto cuida dos lugares de rede. O MkFiles não fala protocolo
// nenhum: quem monta é o gvfs, pelo comando `gio`, e o remoto aparece como
// uma pasta comum em $XDG_RUNTIME_DIR/gvfs. Aqui ficam o tradutor entre o
// endereço (URI) e o nome dessa pasta, a conversa com o `gio mount` (as
// perguntas de usuário e senha viram eventos para a janela) e a leitura dos
// hosts do ~/.ssh/config. Veja ../../../CONTRATO.md.
package remoto

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"mkfiles/internal/base"
)

// Os esquemas que o motor aceita montar.
var esquemas = map[string]bool{"sftp": true, "smb": true, "ftp": true, "ftps": true, "dav": true, "davs": true,
	"mtp": true, "gphoto2": true, "nfs": true, "afp": true}

// Endereco é um URI de remoto aberto nas suas partes.
type Endereco struct {
	Esquema, Usuario, Dominio, Host, Porta string
	Caminho                                string // começa com "/" (ou é "")
}

// Abre confere e abre um endereço: esquema conhecido, host presente, nada de
// byte de controle. A senha nunca vai no endereço: um "usuario:senha@" é
// recusado.
func Abre(uri string) (Endereco, error) {
	for _, r := range uri {
		if r < 0x20 || r == 0x7f {
			return Endereco{}, errors.New("endereço com caractere de controle")
		}
	}
	esquema, _, tem := strings.Cut(uri, "://")
	esquema = strings.ToLower(esquema)
	if !tem || !esquemas[esquema] {
		return Endereco{}, errors.New("o endereço tem de começar com sftp://, smb://, ftp://, ftps://, dav://, davs://, nfs://, afp://, mtp:// ou gphoto2://")
	}
	if esquema == "mtp" || esquema == "gphoto2" {
		// O "host" é o endereço USB ([usb:001,004]), que não é um host de
		// URL: é lido à mão.
		resto := uri[len(esquema)+3:]
		host, caminho, _ := strings.Cut(resto, "/")
		h, err := url.PathUnescape(host)
		c, erro := url.PathUnescape(caminho)
		if err != nil || erro != nil || h == "" || strings.ContainsAny(h, "/@") {
			return Endereco{}, errors.New("endereço mal formado")
		}
		return Endereco{Esquema: esquema, Host: h, Caminho: filepath.Clean("/" + c)}, nil
	}
	u, err := url.Parse(uri)
	if err != nil || u.Host == "" {
		return Endereco{}, errors.New("endereço mal formado")
	}
	if _, temSenha := u.User.Password(); temSenha {
		return Endereco{}, errors.New("a senha não vai no endereço: ela é pedida ao conectar")
	}
	e := Endereco{Esquema: esquema, Host: strings.ToLower(u.Hostname()), Porta: u.Port(), Caminho: u.Path}
	if e.Host == "" {
		return Endereco{}, errors.New("falta o servidor no endereço")
	}
	e.Usuario = u.User.Username()
	if dominio, usuario, ok := strings.Cut(e.Usuario, ";"); ok {
		e.Dominio, e.Usuario = dominio, usuario
	}
	if e.Caminho != "" {
		e.Caminho = filepath.Clean("/" + e.Caminho)
	}
	return e, nil
}

// URI escreve o endereço de volta, com o que não é seguro codificado.
func (e Endereco) URI() string {
	if e.Esquema == "mtp" || e.Esquema == "gphoto2" {
		caminho := e.Caminho
		if caminho == "" {
			caminho = "/"
		}
		return e.Esquema + "://" + url.PathEscape(e.Host) + base.CodificaCaminho(caminho)
	}
	u := url.URL{Scheme: e.Esquema, Host: e.Host, Path: e.Caminho}
	if e.Porta != "" {
		u.Host = e.Host + ":" + e.Porta
		if strings.Contains(e.Host, ":") {
			u.Host = "[" + e.Host + "]:" + e.Porta
		}
	} else if strings.Contains(e.Host, ":") {
		u.Host = "[" + e.Host + "]"
	}
	usuario := e.Usuario
	if e.Dominio != "" {
		usuario = e.Dominio + ";" + usuario
	}
	if usuario != "" {
		u.User = url.User(usuario)
	}
	if u.Path == "" {
		u.Path = "/"
	}
	return u.String()
}

// compartilhamento separa o primeiro pedaço do caminho (o compartilhamento do
// SMB, o volume do AFP) do resto.
func compartilhamento(caminho string) (primeiro, resto string) {
	primeiro, resto, _ = strings.Cut(strings.TrimPrefix(caminho, "/"), "/")
	if resto != "" {
		resto = "/" + resto
	}
	return primeiro, resto
}

// escapa escreve um valor como o gvfs escreve no nome da pasta: o que não é
// letra, número ou um dos poucos sinais seguros vira %XX.
func escapa(valor string) string {
	const hex = "0123456789ABCDEF"
	var b strings.Builder
	for i := 0; i < len(valor); i++ {
		c := valor[i]
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.IndexByte("-_.~!$&'()*+;:@[]", c) >= 0 {
			b.WriteByte(c)
		} else {
			b.WriteByte('%')
			b.WriteByte(hex[c>>4])
			b.WriteByte(hex[c&15])
		}
	}
	return b.String()
}

// Pasta é o nome que o gvfs dá à pasta deste endereço ("sftp:host=x,user=y"),
// e o que sobra do caminho para andar dentro dela. A ordem das chaves do gvfs
// de verdade pode ser outra: para achar um remoto montado use Casa, não a
// comparação de nomes.
func (e Endereco) Pasta() (nome, dentro string) {
	pares := []string{}
	junta := func(chave, valor string) {
		if valor != "" {
			pares = append(pares, chave+"="+escapa(valor))
		}
	}
	protocolo, dentro := e.Esquema, e.Caminho
	switch e.Esquema {
	case "smb":
		protocolo = "smb-share"
		parte, resto := compartilhamento(e.Caminho)
		junta("domain", e.Dominio)
		junta("server", e.Host)
		junta("share", strings.ToLower(parte))
		junta("user", e.Usuario)
		junta("port", e.Porta)
		dentro = resto
	case "afp":
		protocolo = "afp-volume"
		parte, resto := compartilhamento(e.Caminho)
		junta("host", e.Host)
		junta("user", e.Usuario)
		junta("volume", parte)
		dentro = resto
	case "dav", "davs":
		protocolo = "dav"
		junta("host", e.Host)
		junta("port", e.Porta)
		if e.Esquema == "davs" {
			junta("ssl", "true")
		} else {
			junta("ssl", "false")
		}
		junta("user", e.Usuario)
		// O prefixo é o caminho montado: o que vem depois já é dentro dele.
		if e.Caminho != "" && e.Caminho != "/" {
			junta("prefix", e.Caminho)
		}
		dentro = ""
	case "nfs":
		junta("host", e.Host)
		if e.Caminho != "" && e.Caminho != "/" {
			junta("prefix", e.Caminho)
		}
		dentro = ""
	default:
		junta("host", e.Host)
		junta("port", e.Porta)
		junta("user", e.Usuario)
	}
	if dentro == "/" {
		dentro = ""
	}
	return protocolo + ":" + strings.Join(pares, ","), dentro
}

// DaPasta é o caminho inverso: o endereço de uma pasta do gvfs. ok é false
// para um protocolo que o motor não conhece.
func DaPasta(nome string) (e Endereco, ok bool) {
	protocolo, pares := base.PartesDoRemoto(nome)
	e = Endereco{Usuario: pares["user"], Porta: pares["port"], Host: strings.ToLower(pares["host"])}
	switch protocolo {
	case "smb-share":
		e.Esquema, e.Host, e.Dominio = "smb", strings.ToLower(pares["server"]), pares["domain"]
		e.Caminho = "/" + pares["share"]
	case "afp-volume":
		e.Esquema, e.Caminho = "afp", "/"+pares["volume"]
	case "dav":
		e.Esquema = "dav"
		if pares["ssl"] == "true" {
			e.Esquema = "davs"
		}
		e.Caminho = pares["prefix"]
	case "nfs":
		e.Esquema, e.Caminho = "nfs", pares["prefix"]
	case "mtp", "gphoto2":
		e.Esquema, e.Host, e.Porta = protocolo, pares["host"], ""
	case "sftp", "ftp", "ftps":
		e.Esquema = protocolo
	default:
		return Endereco{}, false
	}
	if e.Host == "" {
		return Endereco{}, false
	}
	if e.Caminho != "" {
		e.Caminho = filepath.Clean("/" + e.Caminho)
	}
	return e, true
}

// Casa diz se a pasta do gvfs é a deste endereço: mesmo protocolo e servidor,
// e, quando o endereço diz, o mesmo usuário, porta e compartilhamento. Se
// casa, dentro é o que sobra do caminho do endereço dentro da pasta.
func (e Endereco) Casa(pasta string) (dentro string, casa bool) {
	da, ok := DaPasta(pasta)
	if !ok || da.Esquema != e.Esquema || da.Host != e.Host {
		return "", false
	}
	if e.Usuario != "" && da.Usuario != "" && e.Usuario != da.Usuario {
		return "", false
	}
	if e.Porta != "" && da.Porta != "" && e.Porta != da.Porta {
		return "", false
	}
	switch e.Esquema {
	case "smb", "afp":
		meu, resto := compartilhamento(e.Caminho)
		dele, _ := compartilhamento(da.Caminho)
		if !strings.EqualFold(meu, dele) {
			return "", false
		}
		return resto, true
	case "dav", "davs", "nfs":
		prefixo := da.Caminho
		if prefixo == "" {
			prefixo = "/"
		}
		caminho := e.Caminho
		if caminho == "" {
			caminho = "/"
		}
		if !base.Dentro(prefixo, caminho) {
			return "", false
		}
		resto := strings.TrimPrefix(caminho, strings.TrimSuffix(prefixo, "/"))
		if resto == "/" {
			resto = ""
		}
		return resto, true
	}
	if e.Caminho == "/" {
		return "", true
	}
	return e.Caminho, true
}

// Montado é um remoto que o gvfs tem montado.
type Montado struct {
	URI       *string `json:"uri"`
	Raiz      string  `json:"raiz"`
	Nome      string  `json:"nome"`
	Protocolo string  `json:"protocolo"`
	Host      *string `json:"host"`
}

// Montados lista as pastas do gvfs, em ordem de nome.
func Montados() []Montado {
	lista := []Montado{}
	g := base.GVFS()
	if g == "" {
		return lista
	}
	entradas, err := os.ReadDir(g)
	if err != nil {
		return lista
	}
	for _, en := range entradas {
		nome := en.Name()
		protocolo, _ := base.PartesDoRemoto(nome)
		m := Montado{Raiz: filepath.Join(g, nome), Nome: base.NomeDoRemoto(nome), Protocolo: protocolo}
		if e, ok := DaPasta(nome); ok {
			uri, host := e.URI(), e.Host
			m.URI, m.Host, m.Protocolo = &uri, &host, e.Esquema
			if parte, _ := compartilhamento(e.Caminho); parte != "" && (e.Esquema == "smb" || e.Esquema == "afp") {
				m.Nome = parte + " em " + e.Host
			}
		}
		lista = append(lista, m)
	}
	sort.Slice(lista, func(i, j int) bool { return lista[i].Raiz < lista[j].Raiz })
	return lista
}

// Onde acha a pasta montada de um endereço: a raiz dela e o caminho completo
// (a raiz mais o que o endereço pede lá dentro).
func Onde(e Endereco) (raiz, caminho string, achou bool) {
	g := base.GVFS()
	if g == "" {
		return "", "", false
	}
	entradas, _ := os.ReadDir(g)
	for _, en := range entradas {
		if dentro, casa := e.Casa(en.Name()); casa {
			raiz = filepath.Join(g, en.Name())
			return raiz, filepath.Join(raiz, dentro), true
		}
	}
	return "", "", false
}
