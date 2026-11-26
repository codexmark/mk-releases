// Package base tem o que toda área do motor usa: as pastas da pessoa e os
// comandos (trocáveis por variável de ambiente, para os testes), a validação
// de caminhos e nomes, as recusas de segurança, a escrita atômica, os URIs e a
// tabela de montagens. O contrato está em ../../../CONTRATO.md.
package base

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// Códigos de saída do contrato.
const (
	Certo    = 0
	Erro     = 1
	Recusado = 2
)

// Home é a pasta pessoal de quem roda ("" se não há).
func Home() string {
	h, _ := os.UserHomeDir()
	if h == "" || !filepath.IsAbs(h) {
		return ""
	}
	return filepath.Clean(h)
}

func xdg(variavel string, padrao ...string) string {
	if v := os.Getenv(variavel); filepath.IsAbs(v) {
		return filepath.Clean(v)
	}
	return filepath.Join(append([]string{Home()}, padrao...)...)
}

// Config monta um caminho dentro de $XDG_CONFIG_HOME (~/.config).
func Config(partes ...string) string {
	return filepath.Join(append([]string{xdg("XDG_CONFIG_HOME", ".config")}, partes...)...)
}

// Dados monta um caminho dentro de $XDG_DATA_HOME (~/.local/share).
func Dados(partes ...string) string {
	return filepath.Join(append([]string{xdg("XDG_DATA_HOME", ".local", "share")}, partes...)...)
}

// Cache monta um caminho dentro de $XDG_CACHE_HOME (~/.cache).
func Cache(partes ...string) string {
	return filepath.Join(append([]string{xdg("XDG_CACHE_HOME", ".cache")}, partes...)...)
}

// Estado monta um caminho dentro de $XDG_STATE_HOME (~/.local/state).
func Estado(partes ...string) string {
	return filepath.Join(append([]string{xdg("XDG_STATE_HOME", ".local", "state")}, partes...)...)
}

func lista(variavel, padrao string) []string {
	v := os.Getenv(variavel)
	if v == "" {
		v = padrao
	}
	var pastas []string
	for _, p := range strings.Split(v, ":") {
		if filepath.IsAbs(p) {
			pastas = append(pastas, filepath.Clean(p))
		}
	}
	return pastas
}

// PastasDeDados são as raízes de dados na ordem do XDG: a da pessoa, depois as
// de $XDG_DATA_DIRS.
func PastasDeDados() []string {
	return append([]string{Dados()}, lista("XDG_DATA_DIRS", "/usr/local/share:/usr/share")...)
}

// PastasDeConfig são as raízes de configuração na ordem do XDG.
func PastasDeConfig() []string {
	return append([]string{Config()}, lista("XDG_CONFIG_DIRS", "/etc/xdg")...)
}

// Comando é o nome de um comando externo: o da variável, ou o padrão.
func Comando(variavel, padrao string) string {
	if v := os.Getenv(variavel); v != "" {
		return v
	}
	return padrao
}

// Raiz é uma pasta ou arquivo do sistema: o da variável (absoluto), ou o padrão.
func Raiz(variavel, padrao string) string {
	if v := os.Getenv(variavel); filepath.IsAbs(v) {
		return filepath.Clean(v)
	}
	return padrao
}

// Caminho confere um caminho recebido: absoluto, limpo, sem byte nulo. Não
// conserta: um caminho torto é recusado.
func Caminho(c string) error {
	switch {
	case c == "":
		return errors.New("caminho vazio")
	case strings.ContainsRune(c, 0):
		return errors.New("caminho com byte nulo")
	case !filepath.IsAbs(c):
		return errors.New("o caminho tem de ser absoluto: «" + c + "»")
	case filepath.Clean(c) != c:
		return errors.New("o caminho não está limpo: «" + c + "»")
	}
	return nil
}

// Caminhos confere uma lista (não vazia) de caminhos.
func Caminhos(cs []string) error {
	if len(cs) == 0 {
		return errors.New("nenhum caminho")
	}
	for _, c := range cs {
		if err := Caminho(c); err != nil {
			return err
		}
	}
	return nil
}

// Nome confere um nome novo de arquivo ou pasta.
func Nome(n string) error {
	switch {
	case n == "":
		return errors.New("o nome não pode ser vazio")
	case n == "." || n == "..":
		return errors.New("«" + n + "» não serve de nome")
	case strings.ContainsRune(n, '/'):
		return errors.New("o nome não pode ter «/»")
	case strings.ContainsRune(n, 0):
		return errors.New("nome com byte nulo")
	case len(n) > 255:
		return errors.New("nome comprido demais")
	}
	return nil
}

// Dentro diz se filho é o próprio pai ou está dentro dele (os dois limpos).
func Dentro(pai, filho string) bool {
	if pai == filho {
		return true
	}
	if pai == "/" {
		return strings.HasPrefix(filho, "/")
	}
	return strings.HasPrefix(filho, pai+"/")
}

// Protegido é a recusa de segurança: o motor nunca apaga, move nem renomeia o
// caminho vazio, a raiz, a pasta pessoal nem uma pasta acima dela. Devolve nil
// quando o caminho pode ser mexido.
func Protegido(c string) error {
	if err := Caminho(c); err != nil {
		return err
	}
	if c == "/" {
		return errors.New("a raiz do sistema não pode ser mexida")
	}
	h := Home()
	if h == "" {
		return errors.New("não sei qual é a pasta pessoal: não mexo em nada")
	}
	if c == h {
		return errors.New("a pasta pessoal inteira não pode ser mexida")
	}
	if Dentro(c, h) {
		return errors.New("«" + c + "» contém a pasta pessoal: não pode ser mexida")
	}
	// A pasta dos remotos e a raiz de cada remoto montado não se apagam nem se
	// movem: um remoto se desmonta.
	if g := GVFS(); g != "" {
		if Dentro(c, g) {
			return errors.New("«" + c + "» contém os remotos montados: não pode ser mexida")
		}
		if raiz, nome, ok := Remoto(c); ok && raiz == c {
			return errors.New("«" + nome + "» é um remoto montado: desmonte em vez de apagar")
		}
	}
	return nil
}

// GravaAtomico escreve um arquivo inteiro sem deixar um meio escrito: grava num
// temporário na mesma pasta e troca no fim. Cria a pasta se falta.
func GravaAtomico(caminho string, dados []byte, modo os.FileMode) error {
	if err := Caminho(caminho); err != nil {
		return err
	}
	pasta := filepath.Dir(caminho)
	if err := os.MkdirAll(pasta, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(pasta, "."+filepath.Base(caminho)+".*")
	if err != nil {
		return err
	}
	nome := tmp.Name()
	_, err = tmp.Write(dados)
	if err == nil {
		err = tmp.Chmod(modo)
	}
	if e := tmp.Close(); err == nil {
		err = e
	}
	if err == nil {
		err = os.Rename(nome, caminho)
	}
	if err != nil {
		os.Remove(nome)
	}
	return err
}

// URI escreve um caminho como file://…, com o que não é seguro codificado.
func URI(caminho string) string {
	return "file://" + CodificaCaminho(caminho)
}

// CodificaCaminho codifica um caminho como numa URI (as barras ficam).
func CodificaCaminho(caminho string) string {
	partes := strings.Split(caminho, "/")
	for i, p := range partes {
		partes[i] = url.PathEscape(p)
	}
	return strings.Join(partes, "/")
}

// DeURI devolve o caminho de um file://… (false se não é um, ou se é de
// outra máquina).
func DeURI(uri string) (string, bool) {
	resto, ok := strings.CutPrefix(uri, "file://")
	if !ok {
		return "", false
	}
	if !strings.HasPrefix(resto, "/") {
		// file://máquina/caminho: só serve a daqui.
		maquina, caminho, tem := strings.Cut(resto, "/")
		if !tem || (maquina != "localhost") {
			return "", false
		}
		resto = "/" + caminho
	}
	c, err := url.PathUnescape(resto)
	if err != nil || strings.ContainsRune(c, 0) {
		return "", false
	}
	return filepath.Clean(c), true
}

// Montagem é uma linha da tabela de montagens.
type Montagem struct {
	Ponto  string // onde está montado
	Tipo   string // ext4, vfat…
	Origem string // /dev/sda1…
}

// Montagens lê a tabela de montagens (/proc/self/mountinfo, ou
// MKFILES_MOUNTINFO).
func Montagens() []Montagem {
	b, err := os.ReadFile(Raiz("MKFILES_MOUNTINFO", "/proc/self/mountinfo"))
	if err != nil {
		return nil
	}
	var ms []Montagem
	for _, linha := range strings.Split(string(b), "\n") {
		antes, depois, ok := strings.Cut(linha, " - ")
		campos := strings.Fields(antes)
		if !ok || len(campos) < 5 {
			continue
		}
		m := Montagem{Ponto: filepath.Clean(semOctal(campos[4]))}
		if resto := strings.Fields(depois); len(resto) >= 2 {
			m.Tipo, m.Origem = resto[0], semOctal(resto[1])
		}
		if filepath.IsAbs(m.Ponto) {
			ms = append(ms, m)
		}
	}
	return ms
}

// PontoDe diz em que ponto de montagem um caminho está: o mais longo da
// tabela que o contém ("/" se nenhum).
func PontoDe(caminho string) string {
	melhor := "/"
	for _, m := range Montagens() {
		if Dentro(m.Ponto, caminho) && len(m.Ponto) > len(melhor) {
			melhor = m.Ponto
		}
	}
	return melhor
}

// semOctal desfaz os \040 do mountinfo.
func semOctal(s string) string {
	if !strings.ContainsRune(s, '\\') {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+3 < len(s) {
			n := 0
			certo := true
			for _, c := range s[i+1 : i+4] {
				if c < '0' || c > '7' {
					certo = false
				}
				n = n*8 + int(c-'0')
			}
			if certo {
				b.WriteByte(byte(n))
				i += 3
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
