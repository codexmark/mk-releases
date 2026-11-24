// Package disco é a metade do motor que toca o disco: abrir um texto, gravar
// sem perder nada (escrita atômica, conflito quando o arquivo mudou por fora),
// vigiar o arquivo, os rascunhos e as preferências.
package disco

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"unicode/utf8"

	"mkescrita/internal/base"
)

// Impressao é o retrato de um arquivo no disco: com ela a janela sabe se o
// que está lá ainda é o que ela abriu ou gravou.
type Impressao struct {
	Tamanho int64  `json:"tamanho_bytes"`
	Mtime   int64  `json:"mtime_ns"`
	Sha256  string `json:"sha256"`
}

// Resumo é o sha256 de um texto, em hexadecimal.
func Resumo(dados []byte) string {
	s := sha256.Sum256(dados)
	return hex.EncodeToString(s[:])
}

// Aberto é a resposta de `abrir`.
type Aberto struct {
	Arquivo string `json:"arquivo"`
	Nome    string `json:"nome"`
	// Novo: o arquivo ainda não existe (a pasta, sim); o texto é vazio.
	Novo      bool       `json:"novo"`
	Texto     string     `json:"texto"`
	Impressao *Impressao `json:"impressao"`
	// FimDeLinha: "lf", "crlf" ou "misto". Com "crlf" o texto vem com "\n" e
	// o `salvar --fim-de-linha crlf` devolve os "\r\n"; com "misto" o texto
	// vem como está no disco.
	FimDeLinha  string `json:"fim_de_linha"`
	QuebraFinal bool   `json:"quebra_final"`
	SoLeitura   bool   `json:"so_leitura"`
}

// le devolve o conteúdo de um arquivo comum, até o limite. ausente=true quando
// não existe. Pasta, dispositivo e arquivo grande demais são recusados.
func le(caminho string) (dados []byte, info fs.FileInfo, ausente bool, err error) {
	info, err = os.Stat(caminho)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil, true, nil
	}
	if err != nil {
		return nil, nil, false, fmt.Errorf("não consegui olhar %s: %v", base.Nome(caminho), frase(err))
	}
	if info.IsDir() {
		return nil, nil, false, base.Recusa("%s é uma pasta, não um texto", base.Nome(caminho))
	}
	if !info.Mode().IsRegular() {
		return nil, nil, false, base.Recusa("%s não é um arquivo comum", base.Nome(caminho))
	}
	limite := base.LimiteBytes()
	if info.Size() > limite {
		return nil, nil, false, base.Recusa("%s tem %d MB: é grande demais para o MkEscrita (o limite é %d MB)",
			base.Nome(caminho), (info.Size()+(1<<20)-1)>>20, limite>>20)
	}
	f, err := os.Open(caminho)
	if err != nil {
		return nil, nil, false, fmt.Errorf("não consegui abrir %s: %v", base.Nome(caminho), frase(err))
	}
	defer f.Close()
	dados, err = io.ReadAll(io.LimitReader(f, limite+1))
	if err != nil {
		return nil, nil, false, fmt.Errorf("não consegui ler %s: %v", base.Nome(caminho), frase(err))
	}
	if int64(len(dados)) > limite {
		return nil, nil, false, base.Recusa("%s é grande demais para o MkEscrita (o limite é %d MB)", base.Nome(caminho), limite>>20)
	}
	return dados, info, false, nil
}

// frase tira o caminho repetido de um erro do sistema ("permissão negada").
func frase(err error) string {
	var p *fs.PathError
	if errors.As(err, &p) {
		err = p.Err
	}
	switch {
	case errors.Is(err, fs.ErrPermission):
		return "sem permissão"
	case errors.Is(err, fs.ErrNotExist):
		return "não existe"
	case errors.Is(err, syscall.ENOSPC):
		return "o disco está cheio"
	case errors.Is(err, syscall.EROFS):
		return "o disco é só de leitura"
	}
	return err.Error()
}

func impressaoDe(dados []byte, info fs.FileInfo) *Impressao {
	return &Impressao{Tamanho: int64(len(dados)), Mtime: info.ModTime().UnixNano(), Sha256: Resumo(dados)}
}

// Imprime tira a impressão de um arquivo. nil (sem erro) quando ele não existe.
func Imprime(caminho string) (*Impressao, error) {
	dados, info, ausente, err := le(caminho)
	if err != nil || ausente {
		return nil, err
	}
	return impressaoDe(dados, info), nil
}

// fimDeLinha diz como as linhas do texto acabam.
func fimDeLinha(dados []byte) string {
	crlf := bytes.Count(dados, []byte("\r\n"))
	if crlf == 0 {
		return "lf"
	}
	if crlf == bytes.Count(dados, []byte("\n")) && crlf == bytes.Count(dados, []byte("\r")) {
		return "crlf"
	}
	return "misto"
}

// Abrir lê um texto para a janela. Recusa (código 2) o que não é texto:
// binário (tem byte nulo), outra codificação que não UTF-8, grande demais.
func Abrir(caminho string) (*Aberto, error) {
	abs, err := filepath.Abs(caminho)
	if err != nil {
		return nil, fmt.Errorf("caminho inválido: %s", caminho)
	}
	a := &Aberto{Arquivo: abs, Nome: base.Nome(abs), FimDeLinha: "lf", QuebraFinal: true}
	dados, info, ausente, err := le(abs)
	if err != nil {
		return nil, err
	}
	if ausente {
		pasta, err := os.Stat(filepath.Dir(abs))
		if err != nil || !pasta.IsDir() {
			return nil, fmt.Errorf("a pasta %s não existe", filepath.Dir(abs))
		}
		a.Novo = true
		return a, nil
	}
	if bytes.IndexByte(dados, 0) >= 0 {
		return nil, base.Recusa("%s parece um arquivo binário, não um texto: não abro", a.Nome)
	}
	if !utf8.Valid(dados) {
		return nil, base.Recusa("%s não está em UTF-8 (talvez Latin-1): abrir e salvar trocaria as letras acentuadas, então não abro", a.Nome)
	}
	a.Impressao = impressaoDe(dados, info)
	a.FimDeLinha = fimDeLinha(dados)
	a.QuebraFinal = len(dados) == 0 || dados[len(dados)-1] == '\n'
	a.SoLeitura = syscall.Access(abs, 2) != nil
	if a.FimDeLinha == "crlf" {
		dados = bytes.ReplaceAll(dados, []byte("\r\n"), []byte("\n"))
	}
	a.Texto = string(dados)
	return a, nil
}

// gravaAtomico põe `dados` em `destino` sem nunca deixar um arquivo pela
// metade: escreve num temporário na mesma pasta, manda para o disco (fsync),
// e só então troca o nome. Se algo falha no caminho, o arquivo antigo fica
// como estava e o temporário some.
func gravaAtomico(destino string, dados []byte, modo fs.FileMode, dono *syscall.Stat_t) (err error) {
	pasta := filepath.Dir(destino)
	tmp, err := os.CreateTemp(pasta, "."+filepath.Base(destino)+".mkescrita-*")
	if err != nil {
		return fmt.Errorf("não consigo gravar na pasta %s: %v", pasta, frase(err))
	}
	nome := tmp.Name()
	defer func() {
		if err != nil {
			tmp.Close()
			os.Remove(nome)
		}
	}()
	if _, err = tmp.Write(dados); err != nil {
		return fmt.Errorf("não consegui gravar: %v", frase(err))
	}
	if err = tmp.Chmod(modo); err != nil {
		return fmt.Errorf("não consegui acertar as permissões: %v", frase(err))
	}
	if dono != nil && (int(dono.Uid) != os.Getuid() || int(dono.Gid) != os.Getgid()) {
		// O arquivo é de outro dono ou grupo: tenta manter. Sem privilégio
		// para isso, segue (o conteúdo importa mais que o grupo).
		_ = tmp.Chown(int(dono.Uid), int(dono.Gid))
		_ = tmp.Chmod(modo) // o chown pode ter derrubado bits
	}
	if err = tmp.Sync(); err != nil {
		return fmt.Errorf("não consegui mandar o texto para o disco: %v", frase(err))
	}
	if err = tmp.Close(); err != nil {
		return fmt.Errorf("não consegui gravar: %v", frase(err))
	}
	if err = os.Rename(nome, destino); err != nil {
		return fmt.Errorf("não consegui pôr o texto no lugar: %v", frase(err))
	}
	// A troca de nome também vai para o disco.
	if d, e := os.Open(pasta); e == nil {
		_ = d.Sync()
		d.Close()
	}
	return nil
}
