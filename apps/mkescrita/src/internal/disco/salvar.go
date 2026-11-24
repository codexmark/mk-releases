package disco

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"

	"mkescrita/internal/base"
)

// Ausente é o valor de Esperado que diz "quando comecei, o arquivo não existia".
const Ausente = "ausente"

// Pedido é o que `salvar` recebe.
type Pedido struct {
	Caminho string
	Texto   []byte
	// Esperado: o sha256 do arquivo como a janela o conhece, ou Ausente.
	// Vazio só vale com Forcar.
	Esperado string
	// Forcar grava mesmo que o arquivo tenha mudado por fora.
	Forcar bool
	// FimDeLinha "crlf" troca cada "\n" por "\r\n" (o texto da janela é
	// sempre "\n"); qualquer outro valor grava como veio.
	FimDeLinha string
	// QuebraFinal garante que um texto não vazio termine com quebra de linha.
	QuebraFinal bool
}

// Salvo é a resposta de `salvar` quando gravou.
type Salvo struct {
	Arquivo   string     `json:"arquivo"`
	Nome      string     `json:"nome"`
	Impressao *Impressao `json:"impressao"`
}

// Conflito é a resposta de `salvar` quando o arquivo no disco não é mais o
// que a janela conhecia: nada foi gravado.
type Conflito struct {
	// Motivo: "mudou" (outro conteúdo), "sumiu" (não existe mais) ou
	// "apareceu" (não existia e agora existe).
	Motivo string
	// Impressao do que está no disco agora (nil quando sumiu).
	Impressao *Impressao
}

func (c *Conflito) Error() string {
	switch c.Motivo {
	case "sumiu":
		return "o arquivo sumiu do disco desde que foi aberto"
	case "apareceu":
		return "já existe um arquivo com esse nome, criado por fora"
	}
	return "o arquivo mudou por fora desde que foi aberto"
}

// destinoReal segue os links simbólicos de um caminho até o arquivo de
// verdade, mesmo que ele ainda não exista (link para um arquivo a criar):
// gravar "no link" troca o conteúdo do alvo, nunca o link por um arquivo.
func destinoReal(caminho string) (string, error) {
	for i := 0; i < 40; i++ {
		info, err := os.Lstat(caminho)
		if err != nil || info.Mode()&fs.ModeSymlink == 0 {
			return caminho, nil
		}
		alvo, err := os.Readlink(caminho)
		if err != nil {
			return "", fmt.Errorf("não consegui seguir o link %s: %v", base.Nome(caminho), frase(err))
		}
		if !filepath.IsAbs(alvo) {
			alvo = filepath.Join(filepath.Dir(caminho), alvo)
		}
		caminho = filepath.Clean(alvo)
	}
	return "", errors.New("o link simbólico dá voltas e não chega a arquivo nenhum")
}

// umask lê a máscara de permissões do processo (o sistema só deixa ler trocando).
func umask() fs.FileMode {
	m := syscall.Umask(0)
	syscall.Umask(m)
	return fs.FileMode(m)
}

// Preparar deixa o texto como vai para o disco: a quebra final e o fim de linha.
func Preparar(texto []byte, fimDeLinha string, quebraFinal bool) []byte {
	if quebraFinal && len(texto) > 0 && texto[len(texto)-1] != '\n' {
		texto = append(append([]byte{}, texto...), '\n')
	}
	if fimDeLinha == "crlf" {
		texto = bytes.ReplaceAll(texto, []byte("\r\n"), []byte("\n"))
		texto = bytes.ReplaceAll(texto, []byte("\n"), []byte("\r\n"))
	}
	return texto
}

// Salvar grava o texto. Devolve *Conflito (como erro) quando o disco não está
// como a janela esperava: aí nada é gravado e quem decide é a pessoa.
func Salvar(p Pedido) (*Salvo, error) {
	if p.Caminho == "" {
		return nil, errors.New("salvar onde? falta o arquivo")
	}
	if p.Esperado == "" && !p.Forcar {
		return nil, errors.New("salvar pede --esperado (a impressão do arquivo aberto) ou --forcar")
	}
	abs, err := filepath.Abs(p.Caminho)
	if err != nil {
		return nil, fmt.Errorf("caminho inválido: %s", p.Caminho)
	}
	destino, err := destinoReal(abs)
	if err != nil {
		return nil, err
	}
	atual, info, ausente, err := le(destino)
	if err != nil {
		// Forçar não transforma pasta, dispositivo nem arquivo enorme em texto.
		return nil, err
	}
	if !p.Forcar {
		switch {
		case p.Esperado == Ausente && !ausente:
			return nil, &Conflito{Motivo: "apareceu", Impressao: impressaoDe(atual, info)}
		case p.Esperado != Ausente && ausente:
			return nil, &Conflito{Motivo: "sumiu"}
		case p.Esperado != Ausente && Resumo(atual) != p.Esperado:
			return nil, &Conflito{Motivo: "mudou", Impressao: impressaoDe(atual, info)}
		}
	}
	modo := 0o666 &^ umask()
	var dono *syscall.Stat_t
	if !ausente {
		if syscall.Access(destino, 2) != nil {
			return nil, fmt.Errorf("%s é só para leitura: não gravei", base.Nome(abs))
		}
		modo = info.Mode().Perm()
		dono, _ = info.Sys().(*syscall.Stat_t)
	} else if pasta, e := os.Stat(filepath.Dir(destino)); e != nil || !pasta.IsDir() {
		return nil, fmt.Errorf("a pasta %s não existe", filepath.Dir(destino))
	}
	dados := Preparar(p.Texto, p.FimDeLinha, p.QuebraFinal)
	if err := gravaAtomico(destino, dados, modo, dono); err != nil {
		return nil, err
	}
	novo, err := os.Stat(destino)
	if err != nil {
		return nil, fmt.Errorf("gravei, mas não consegui conferir: %v", frase(err))
	}
	return &Salvo{Arquivo: abs, Nome: base.Nome(abs), Impressao: impressaoDe(dados, novo)}, nil
}

// Exportar grava um arquivo novo (o HTML exportado). Nunca passa por cima de
// um que já existe sem Forcar; é atômico como o salvar.
func Exportar(caminho string, dados []byte, forcar bool) (*Salvo, error) {
	if caminho == "" {
		return nil, errors.New("exportar para onde? falta o arquivo de saída")
	}
	abs, err := filepath.Abs(caminho)
	if err != nil {
		return nil, fmt.Errorf("caminho inválido: %s", caminho)
	}
	destino, err := destinoReal(abs)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(destino)
	modo := 0o666 &^ umask()
	var dono *syscall.Stat_t
	switch {
	case err == nil && info.IsDir():
		return nil, base.Recusa("%s é uma pasta", base.Nome(abs))
	case err == nil && !forcar:
		return nil, base.Recusa("%s já existe: não passo por cima", base.Nome(abs))
	case err == nil:
		if !info.Mode().IsRegular() {
			return nil, base.Recusa("%s não é um arquivo comum", base.Nome(abs))
		}
		modo = info.Mode().Perm()
		dono, _ = info.Sys().(*syscall.Stat_t)
	case !errors.Is(err, fs.ErrNotExist):
		return nil, fmt.Errorf("não consegui olhar %s: %v", base.Nome(abs), frase(err))
	}
	if err := gravaAtomico(destino, dados, modo, dono); err != nil {
		return nil, err
	}
	novo, err := os.Stat(destino)
	if err != nil {
		return nil, fmt.Errorf("gravei, mas não consegui conferir: %v", frase(err))
	}
	return &Salvo{Arquivo: abs, Nome: base.Nome(abs), Impressao: impressaoDe(dados, novo)}, nil
}
