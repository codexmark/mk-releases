package base

import (
	"errors"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// GVFS é a pasta em que o gvfs mostra os remotos montados como pastas comuns
// (MKFILES_GVFS, senão $XDG_RUNTIME_DIR/gvfs). "" se não há onde.
func GVFS() string {
	if v := os.Getenv("MKFILES_GVFS"); filepath.IsAbs(v) {
		return filepath.Clean(v)
	}
	if d := os.Getenv("XDG_RUNTIME_DIR"); filepath.IsAbs(d) {
		return filepath.Join(filepath.Clean(d), "gvfs")
	}
	return ""
}

// Remoto diz se um caminho está dentro de um remoto montado pelo gvfs: devolve
// a raiz do remoto (a pasta dele em GVFS) e o nome para mostrar (o host).
func Remoto(caminho string) (raiz, nome string, ok bool) {
	g := GVFS()
	if g == "" || caminho == g || !Dentro(g, caminho) {
		return "", "", false
	}
	pasta, _, _ := strings.Cut(caminho[len(g)+1:], "/")
	if pasta == "" {
		return "", "", false
	}
	return filepath.Join(g, pasta), NomeDoRemoto(pasta), true
}

// PartesDoRemoto abre o nome de uma pasta do gvfs ("sftp:host=x,user=y") no
// protocolo e nos pares chave=valor, já sem os escapes %XX.
func PartesDoRemoto(pasta string) (protocolo string, pares map[string]string) {
	pares = map[string]string{}
	protocolo, resto, tem := strings.Cut(pasta, ":")
	if !tem {
		return pasta, pares
	}
	for _, par := range strings.Split(resto, ",") {
		chave, valor, ok := strings.Cut(par, "=")
		if !ok || chave == "" {
			continue
		}
		if v, err := url.PathUnescape(valor); err == nil {
			valor = v
		}
		pares[chave] = valor
	}
	return protocolo, pares
}

// NomeDoRemoto é o nome de mostrar de uma pasta do gvfs: o host (ou o
// servidor); sem nenhum dos dois, o nome da pasta como está.
func NomeDoRemoto(pasta string) string {
	_, pares := PartesDoRemoto(pasta)
	for _, chave := range []string{"host", "server"} {
		if v := pares[chave]; v != "" {
			return v
		}
	}
	return pasta
}

// ConexaoCaiu olha um erro de disco num caminho remoto: se é de conexão
// perdida (o remoto foi desmontado, a rede caiu), devolve a frase para a
// pessoa. Num caminho que não é remoto, nunca.
func ConexaoCaiu(caminho string, err error) (frase string, caiu bool) {
	raiz, nome, ok := Remoto(caminho)
	if !ok || err == nil {
		return "", false
	}
	caiu = errors.Is(err, syscall.ENOTCONN) || errors.Is(err, syscall.EIO) || errors.Is(err, syscall.EHOSTUNREACH) ||
		errors.Is(err, syscall.ENETUNREACH) || errors.Is(err, syscall.ETIMEDOUT) || errors.Is(err, syscall.ECONNRESET)
	if !caiu && errors.Is(err, fs.ErrNotExist) {
		// "Não existe" só é queda se a própria raiz do remoto sumiu.
		_, e := os.Stat(raiz)
		caiu = e != nil
	}
	if !caiu {
		return "", false
	}
	return "a conexão com " + nome + " caiu", true
}
