package operar

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"

	"mkfiles/internal/base"
	"mkfiles/internal/lixeira"
)

// O que os testes trocam: o rename (para simular outro sistema de arquivos,
// com EXDEV), o tamanho do bloco de cópia e dois ganchos (depois de cada bloco
// copiado; antes de conferir uma cópia entre sistemas de arquivos).
var (
	renomeia       = os.Rename
	tamanhoDoBloco = 1 << 20
	aCadaBloco     func(t *Tarefa, destino string, escritos int64)
	aoConferir     func(de, para string)
)

var errCancelada = errors.New("cancelada")

// Os bits de modo que a cópia preserva.
const bitsDeModo = fs.ModePerm | fs.ModeSetuid | fs.ModeSetgid | fs.ModeSticky

// frase tira o caminho de um erro de disco: "permissão negada", não "open
// /x/y: permission denied".
func frase(err error) string {
	var pe *fs.PathError
	var le *os.LinkError
	var no syscall.Errno
	// Num remoto, rede que caiu ou remoto desmontado no meio: diz isso.
	if errors.As(err, &pe) {
		if f, caiu := base.ConexaoCaiu(pe.Path, err); caiu {
			return f
		}
	} else if errors.As(err, &le) {
		for _, c := range []string{le.Old, le.New} {
			if f, caiu := base.ConexaoCaiu(c, err); caiu {
				return f
			}
		}
	}
	switch {
	case errors.Is(err, fs.ErrPermission):
		return "permissão negada"
	case errors.Is(err, fs.ErrNotExist):
		return "não existe mais"
	case errors.Is(err, fs.ErrExist):
		return "já existe"
	case errors.Is(err, syscall.ENOSPC):
		return "o disco encheu"
	case errors.Is(err, syscall.ENOTEMPTY):
		return "a pasta não está vazia"
	case errors.Is(err, syscall.EROFS):
		return "o disco é só de leitura"
	case errors.As(err, &pe):
		return pe.Err.Error()
	case errors.As(err, &le):
		return le.Err.Error()
	case errors.As(err, &no):
		return no.Error()
	}
	return err.Error()
}

// existe: há algo nesse caminho (um link quebrado também é algo).
func existe(caminho string) bool {
	_, err := os.Lstat(caminho)
	return err == nil
}

// livreEm acha um nome livre na pasta, a partir de nome.
func livreEm(pasta, nome string, ePasta bool) string {
	return lixeira.Numerado(nome, ePasta, func(n string) bool { return existe(filepath.Join(pasta, n)) })
}

// conta soma os itens e os bytes de uma árvore, sem seguir links.
func conta(caminho string) (itens int, bytes int64) {
	info, err := os.Lstat(caminho)
	if err != nil {
		return 0, 0
	}
	itens = 1
	if info.Mode().IsRegular() {
		bytes = info.Size()
	}
	if info.IsDir() {
		entradas, _ := os.ReadDir(caminho)
		for _, e := range entradas {
			i, b := conta(filepath.Join(caminho, e.Name()))
			itens += i
			bytes += b
		}
	}
	return itens, bytes
}

var serie atomic.Int64

// aoLado inventa um nome temporário (oculto) na mesma pasta de um destino.
func aoLado(para string) string {
	nome := filepath.Base(para)
	if len(nome) > 180 {
		nome = strings.ToValidUTF8(nome[:180], "")
	}
	return filepath.Join(filepath.Dir(para), ".mkfiles-"+strconv.Itoa(os.Getpid())+"-"+strconv.FormatInt(serie.Add(1), 10)+"-"+nome)
}

// removeArvore apaga um arquivo ou uma pasta com tudo dentro, de baixo para
// cima, sem nunca seguir um link (o link é apagado; o alvo fica) e sem nunca
// atravessar um ponto de montagem: um disco montado na pasta ou dentro dela
// não é esvaziado (vira erro, e o caminho até ele fica). Recusa o que
// base.Protegido recusa. passo (pode ser nil) é chamado a cada item apagado ou
// que falhou, e devolve false para parar. Devolve false se parou no meio.
func removeArvore(caminho string, passo func(caminho string, info fs.FileInfo, err error) bool) bool {
	if passo == nil {
		passo = func(string, fs.FileInfo, error) bool { return true }
	}
	if err := base.Protegido(caminho); err != nil {
		return passo(caminho, nil, err)
	}
	info, err := os.Lstat(caminho)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return true
		}
		return passo(caminho, nil, err)
	}
	if montadoEm(caminho, info) {
		return passo(caminho, info, outroDiscoEm(caminho))
	}
	// Os pontos da tabela de montagens que ficam dentro: nem se desce neles.
	pontos := map[string]bool{}
	if info.IsDir() {
		for _, m := range base.Montagens() {
			if m.Ponto != caminho && base.Dentro(caminho, m.Ponto) {
				pontos[m.Ponto] = true
			}
		}
	}
	segue, _ := removeDentro(caminho, info, unidadeDe(caminho, info), pontos, passo)
	return segue
}

// removeDentro devolve se é para seguir e se o caminho saiu (uma pasta com um
// disco montado dentro não sai, e isso não é outro erro).
func removeDentro(caminho string, info fs.FileInfo, unidade uint64, pontos map[string]bool,
	passo func(string, fs.FileInfo, error) bool) (segue, saiu bool) {
	if info.IsDir() {
		// Uma pasta sua sem permissão de leitura ou escrita: abre para poder
		// esvaziar (ela vai ser apagada em seguida).
		if info.Mode().Perm()&0o700 != 0o700 {
			_ = os.Chmod(caminho, info.Mode().Perm()|0o700)
		}
		entradas, err := os.ReadDir(caminho)
		if err != nil {
			return passo(caminho, info, err), false
		}
		vazia := true
		for _, e := range entradas {
			filho := filepath.Join(caminho, e.Name())
			la, err := os.Lstat(filho)
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					continue
				}
				vazia = false
				if !passo(filho, nil, err) {
					return false, false
				}
				continue
			}
			if la.IsDir() && (pontos[filho] || unidadeDe(filho, la) != unidade) {
				vazia = false
				if !passo(filho, la, outroDiscoEm(filho)) {
					return false, false
				}
				continue
			}
			segue, saiu := removeDentro(filho, la, unidade, pontos, passo)
			if !segue {
				return false, false
			}
			if !saiu {
				vazia = false
			}
		}
		if !vazia {
			return true, false
		}
	}
	err := os.Remove(caminho)
	return passo(caminho, info, err), err == nil
}

// apaga é o removeArvore de uma tarefa: conta o progresso, anota os erros e
// para se a tarefa foi cancelada. Devolve false se parou.
func (t *Tarefa) apaga(caminho string, contando bool) bool {
	return removeArvore(caminho, func(c string, info fs.FileInfo, err error) bool {
		if err != nil {
			t.Erro(c, frase(err))
		}
		if contando {
			var bytes int64
			if info != nil && info.Mode().IsRegular() {
				bytes = info.Size()
			}
			t.Avanca(1, bytes, c)
		}
		return t.Segue()
	})
}

// copiaArquivo copia um arquivo comum para um destino que não existe
// (O_EXCL), em blocos, preservando modo e mtime. Cancelada no meio, remove o
// parcial e devolve errCancelada.
func (t *Tarefa) copiaArquivo(de, para string, info fs.FileInfo) (err error) {
	origem, err := os.Open(de)
	if err != nil {
		return err
	}
	defer origem.Close()
	destino, err := os.OpenFile(para, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			destino.Close()
			os.Remove(para)
		}
	}()
	bloco := make([]byte, tamanhoDoBloco)
	var escritos int64
	for {
		n, erroDeLeitura := origem.Read(bloco)
		if n > 0 {
			if _, err = destino.Write(bloco[:n]); err != nil {
				return err
			}
			escritos += int64(n)
			t.Avanca(0, int64(n), de)
			if aCadaBloco != nil {
				aCadaBloco(t, para, escritos)
			}
		}
		if erroDeLeitura == io.EOF {
			break
		}
		if erroDeLeitura != nil {
			return erroDeLeitura
		}
		if !t.Segue() {
			return errCancelada
		}
	}
	if err = destino.Chmod(info.Mode() & bitsDeModo); err != nil {
		return err
	}
	if err = destino.Close(); err != nil {
		return err
	}
	_ = os.Chtimes(para, info.ModTime(), info.ModTime())
	return nil
}

// copiaArvore copia de para um destino que não existe: arquivo, link (como
// link) ou pasta com tudo dentro. Os erros de cada item vão para a lista da
// tarefa e a cópia segue. Com grava, anota o que criou para o desfazer.
// Devolve true se a tarefa foi cancelada no meio.
func (t *Tarefa) copiaArvore(de, para string, grava bool) (cancelada bool) {
	info, err := os.Lstat(de)
	if err != nil {
		t.Erro(de, frase(err))
		return false
	}
	switch {
	case info.Mode()&fs.ModeSymlink != 0:
		alvo, err := os.Readlink(de)
		if err == nil {
			err = os.Symlink(alvo, para)
		}
		if err != nil {
			t.Erro(de, frase(err))
		} else if grava {
			t.fezArquivo(para)
		}
	case info.IsDir():
		if err := os.Mkdir(para, 0o700); err != nil {
			n, b := conta(de)
			t.Erro(de, frase(err))
			t.Avanca(n, b, de)
			return false
		}
		if grava {
			t.fez(feito{tipo: fPasta, caminho: para})
		}
		entradas, err := os.ReadDir(de)
		if err != nil {
			t.Erro(de, frase(err))
		}
		for _, e := range entradas {
			if !t.Segue() || t.copiaArvore(filepath.Join(de, e.Name()), filepath.Join(para, e.Name()), grava) {
				return true
			}
		}
		_ = os.Chmod(para, info.Mode()&bitsDeModo)
		_ = os.Chtimes(para, info.ModTime(), info.ModTime())
	case info.Mode().IsRegular():
		err := t.copiaArquivo(de, para, info)
		if err == errCancelada {
			return true
		}
		if err != nil {
			t.Erro(de, frase(err))
		} else if grava {
			t.fezArquivo(para)
		}
	default:
		t.Erro(de, "arquivo especial (soquete, fifo ou dispositivo) não é copiado")
	}
	t.Avanca(1, 0, de)
	return false
}

// confere uma cópia com a origem: tudo que há na origem existe no destino, do
// mesmo tipo; arquivo com o mesmo tamanho, link com o mesmo alvo.
func confere(de, para string) error {
	a, err := os.Lstat(de)
	if err != nil {
		return err
	}
	b, err := os.Lstat(para)
	if err != nil {
		return err
	}
	if a.Mode().Type() != b.Mode().Type() {
		return errors.New("tipo diferente: " + para)
	}
	switch {
	case a.Mode().IsRegular():
		if a.Size() != b.Size() {
			return errors.New("tamanho diferente: " + para)
		}
	case a.Mode()&fs.ModeSymlink != 0:
		x, _ := os.Readlink(de)
		y, _ := os.Readlink(para)
		if x != y {
			return errors.New("link diferente: " + para)
		}
	case a.IsDir():
		entradas, err := os.ReadDir(de)
		if err != nil {
			return err
		}
		for _, e := range entradas {
			if err := confere(filepath.Join(de, e.Name()), filepath.Join(para, e.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

// troca põe tmp no lugar de para. Arquivo sobre arquivo é um rename só (o
// antigo some quando o novo já está lá); quando um dos dois é pasta, o destino
// antigo é apagado antes.
func troca(tmp, para string) error {
	a, err := os.Lstat(tmp)
	if err != nil {
		return err
	}
	if b, err := os.Lstat(para); err == nil && (a.IsDir() || b.IsDir()) {
		var primeiro error
		removeArvore(para, func(_ string, _ fs.FileInfo, err error) bool {
			if err != nil && primeiro == nil {
				primeiro = err
			}
			return true
		})
		if primeiro != nil {
			return primeiro
		}
	}
	return renomeia(tmp, para)
}

// move leva de para para. Sem substituir, para não pode existir. Tenta o
// rename; em outro sistema de arquivos (EXDEV), copia para um temporário ao
// lado do destino, confere e só então apaga a origem. Devolve true se moveu.
func (t *Tarefa) move(de, para string, substitui bool) bool {
	itens, bytes := conta(de)
	if !substitui && existe(para) {
		t.Erro(de, "o destino já existe")
		t.Avanca(itens, bytes, de)
		return false
	}
	// Com substituição o rename vai primeiro para um temporário ao lado: é ele
	// que diz se é outro sistema de arquivos, sem tocar no destino antigo.
	tmp := para
	if substitui {
		tmp = aoLado(para)
	}
	err := renomeia(de, tmp)
	if err == nil {
		if substitui {
			if err := troca(tmp, para); err != nil {
				t.Erro(de, frase(err))
				_ = renomeia(tmp, de) // devolve
				t.Avanca(itens, bytes, de)
				return false
			}
		}
		t.Avanca(itens, bytes, de)
		return true
	}
	if !errors.Is(err, syscall.EXDEV) {
		t.Erro(de, frase(err))
		t.Avanca(itens, bytes, de)
		return false
	}
	// Levar para outro disco é copiar e apagar a origem: com um disco montado
	// dentro dela, nem começa (a cópia levaria o disco junto, e o apagar não
	// pode esvaziá-lo).
	if ponto := montadoDentro(de); ponto != "" {
		t.Erro(de, outroDiscoEm(ponto).Error())
		t.Avanca(itens, bytes, de)
		return false
	}
	antes := t.nErros()
	cancelada := t.copiaArvore(de, tmp, false)
	if cancelada || t.nErros() > antes {
		removeArvore(tmp, nil)
		return false
	}
	if aoConferir != nil {
		aoConferir(de, tmp)
	}
	if err := confere(de, tmp); err != nil {
		removeArvore(tmp, nil)
		t.Erro(de, "a cópia não conferiu com a origem: a origem ficou onde estava")
		return false
	}
	if substitui {
		if err := troca(tmp, para); err != nil {
			removeArvore(tmp, nil)
			t.Erro(de, frase(err))
			return false
		}
	}
	removeArvore(de, func(c string, _ fs.FileInfo, err error) bool {
		if err != nil {
			t.Erro(c, "foi copiado, mas a origem não saiu: "+frase(err))
		}
		return true
	})
	return true
}

// Os jeitos de seguir depois de um conflito.
const (
	livre = iota
	substitui
	mescla
	pula
)

// resolve decide onde e como gravar quando o destino pode já existir:
// pergunta à janela (Conflito) até ter um destino livre ou uma decisão.
// podeMesclar: pasta sobre pasta com "substituir" vira mescla.
func (t *Tarefa) resolve(de, para string, info fs.FileInfo, podeMesclar bool) (string, int) {
	for {
		la, err := os.Lstat(para)
		if err != nil {
			return para, livre
		}
		pasta := filepath.Dir(para)
		r := t.Conflito(de, para, livreEm(pasta, filepath.Base(para), info.IsDir()))
		switch r.Qual {
		case "manter":
			return filepath.Join(pasta, livreEm(pasta, filepath.Base(para), info.IsDir())), livre
		case "renomear":
			para = filepath.Join(pasta, r.Nome)
		case "substituir":
			if podeMesclar && info.IsDir() && la.IsDir() {
				return para, mescla
			}
			if base.Dentro(para, de) || base.Dentro(de, para) {
				t.Erro(de, "o destino e a origem estão um dentro do outro: nada foi substituído")
				return "", pula
			}
			if la.IsDir() {
				if err := base.Protegido(para); err != nil {
					t.Erro(para, err.Error())
					return "", pula
				}
			}
			return para, substitui
		default: // pular e cancelar
			return "", pula
		}
	}
}

// Os tipos do que uma tarefa fez (para o desfazer).
const (
	fArquivo = iota // criou um arquivo ou link: sai se não mudou
	fPasta          // criou uma pasta: sai se continua vazia
	fMovido         // moveu ou renomeou de → caminho
	fLixado         // mandou de para a lixeira, onde é caminho
	fLote           // renomeou em lote: pares de → para, desfeitos todos juntos
)

type feito struct {
	tipo    int
	caminho string
	de      string
	tamanho int64
	mtime   int64 // nanossegundos
	pares   [][2]string
}

// fezArquivo anota um arquivo (ou link) criado, com o tamanho e a mtime de
// agora: o desfazer só o apaga se continuarem os mesmos.
func (t *Tarefa) fezArquivo(caminho string) {
	if info, err := os.Lstat(caminho); err == nil {
		t.fez(feito{tipo: fArquivo, caminho: caminho, tamanho: info.Size(), mtime: info.ModTime().UnixNano()})
	}
}
