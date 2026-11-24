package instala

import (
	"archive/tar"
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"mkutil/internal/base"
)

// FichaPacote é o bloco "pacote" do contrato: o .PKGINFO de um pacote do Arch.
type FichaPacote struct {
	Nome             string   `json:"nome"`
	Versao           string   `json:"versao"`
	Descricao        *string  `json:"descricao"`
	Arquitetura      *string  `json:"arquitetura"`
	TamanhoInstalado *int64   `json:"tamanho_instalado_bytes"`
	Dependencias     []string `json:"dependencias"`
	Instalado        *string  `json:"instalado"`
}

var (
	nomeDePacote   = regexp.MustCompile(`^[a-z0-9@_+][a-z0-9@._+-]*$`)
	versaoDePacote = regexp.MustCompile(`^[A-Za-z0-9.:_+~-]+$`)
)

const maximoDoPKGINFO = 1 << 20

// interpretaPKGINFO lê as linhas "chave = valor" de um .PKGINFO.
func interpretaPKGINFO(b []byte) (FichaPacote, error) {
	p := FichaPacote{Dependencias: []string{}}
	for _, linha := range strings.Split(string(b), "\n") {
		chave, valor, ok := strings.Cut(linha, " = ")
		if !ok || strings.HasPrefix(linha, "#") {
			continue
		}
		valor = limpaValor(valor)
		switch strings.TrimSpace(chave) {
		case "pkgname":
			p.Nome = valor
		case "pkgver":
			p.Versao = valor
		case "pkgdesc":
			if valor != "" {
				p.Descricao = &valor
			}
		case "arch":
			if valor != "" {
				p.Arquitetura = &valor
			}
		case "size":
			if n, err := strconv.ParseInt(valor, 10, 64); err == nil && n >= 0 {
				p.TamanhoInstalado = &n
			}
		case "depend":
			if valor != "" && len(p.Dependencias) < 500 {
				p.Dependencias = append(p.Dependencias, valor)
			}
		}
	}
	if !nomeDePacote.MatchString(p.Nome) || len(p.Nome) > 200 {
		return p, errors.New("o .PKGINFO não tem um nome de pacote válido: não é um pacote do Arch")
	}
	if !versaoDePacote.MatchString(p.Versao) {
		return p, errors.New("o .PKGINFO não tem uma versão válida: não é um pacote do Arch")
	}
	return p, nil
}

// lePKGINFO tira o .PKGINFO de dentro de um pacote. Com o bsdtar, que abre
// qualquer compressão; sem ele, com o descompressor (zstd ou xz) e o tar da
// biblioteca padrão. O caminho vai absoluto, como valor de -f.
func lePKGINFO(arquivo, formato string) ([]byte, error) {
	if base.Tem("bsdtar") {
		b, err := base.Roda("bsdtar", 30*time.Second, nil, "-xOqf", arquivo, ".PKGINFO")
		if err != nil {
			return nil, fmt.Errorf("não consegui ler o pacote: %v", err)
		}
		if len(b) == 0 {
			return nil, errors.New("o arquivo não tem .PKGINFO: não é um pacote do Arch")
		}
		if len(b) > maximoDoPKGINFO {
			return nil, errors.New("o .PKGINFO é grande demais: não é um pacote do Arch")
		}
		return b, nil
	}
	fluxo, err := abreFluxo(arquivo, formato)
	if err != nil {
		return nil, err
	}
	defer fluxo.Close()
	tr := tar.NewReader(bufio.NewReader(fluxo))
	for n := 0; n < 100000; n++ {
		h, err := tr.Next()
		if err == io.EOF {
			if e := fluxo.fim(); e != nil {
				return nil, fmt.Errorf("não consegui ler o pacote: %v", e)
			}
			break
		}
		if err != nil {
			return nil, fmt.Errorf("não consegui ler o pacote: %v", fluxo.erro(err))
		}
		if h.Name == ".PKGINFO" && h.Typeflag == tar.TypeReg {
			if h.Size > maximoDoPKGINFO {
				return nil, errors.New("o .PKGINFO é grande demais: não é um pacote do Arch")
			}
			return io.ReadAll(io.LimitReader(tr, maximoDoPKGINFO))
		}
	}
	return nil, errors.New("o arquivo não tem .PKGINFO: não é um pacote do Arch")
}

// versaoInstalada pergunta ao pacman (sem root, só leitura) que versão do
// pacote a máquina tem. nil se não tem, ou se não há pacman.
func versaoInstalada(nome string) *string {
	if !base.Tem("pacman") {
		return nil
	}
	b, err := base.Roda("pacman", 15*time.Second, nil, "-Q", "--", nome)
	if err != nil {
		return nil
	}
	campos := strings.Fields(string(bytes.TrimSpace(b)))
	if len(campos) != 2 || campos[0] != nome {
		return nil
	}
	return &campos[1]
}
