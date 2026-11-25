// Package vm guarda as máquinas virtuais do MkPorto (uma pasta por máquina,
// com o maquina.json e o disco), monta a linha de comando do QEMU, fala QMP
// com uma máquina ligada e lê o que o /proc diz do processo dela
// (../../../CONTRATO.md, "Máquinas"). Nada aqui roda o QEMU: quem roda é o
// subcomando `vm`, no pacote principal.
package vm

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"syscall"
)

// Maquina é o maquina.json: a configuração de uma máquina.
type Maquina struct {
	Nome      string `json:"nome"`
	MemoriaMB int    `json:"memoria_mb"`
	CPUs      int    `json:"cpus"`
	DiscoGB   int    `json:"disco_gb"`
	ISO       string `json:"iso"`
	UEFI      bool   `json:"uefi"`
	TresD     bool   `json:"tres_d"`
	PortaSSH  int    `json:"porta_ssh"`
	CriadaEm  int64  `json:"criada_em"`
}

// ErroDeCampo é uma recusa da validação: qual campo e por quê.
type ErroDeCampo struct {
	Campo string
	Frase string
}

func (e *ErroDeCampo) Error() string { return e.Frase }

// Os limites de uma máquina.
const (
	MemoriaMinima = 512
	MemoriaMaxima = 65536
	CPUsMaximas   = 32
	DiscoMaximo   = 2000
	PrimeiraPorta = 2222
)

var (
	nomeBom   = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,39}$`)
	rotuloBom = regexp.MustCompile(`^[A-Za-z0-9._][A-Za-z0-9._-]{0,39}$`)
)

// NomeBom diz se o texto serve de nome de máquina: vira pasta, argumento do
// QEMU e apelido de host.
func NomeBom(nome string) bool { return nomeBom.MatchString(nome) }

// RotuloBom diz se o texto serve de rótulo de snapshot (um argumento do
// qemu-img: não pode começar com "-").
func RotuloBom(rotulo string) bool { return rotuloBom.MatchString(rotulo) }

// ErroDoNome é a frase de um nome que não serve ("" se serve).
func ErroDoNome(nome string) string {
	switch {
	case nome == "":
		return "falta o nome da máquina"
	case len(nome) > 40:
		return "o nome é comprido demais (até 40)"
	case !NomeBom(nome):
		return "o nome só pode ter letras minúsculas sem acento, algarismos, ponto, hífen e sublinhado, começando por letra ou algarismo"
	}
	return ""
}

// ErroDaISO confere o caminho de uma ISO, sem olhar o disco ("" se serve; o
// vazio serve: máquina sem ISO).
func ErroDaISO(iso string) string {
	if iso == "" {
		return ""
	}
	for _, c := range iso {
		switch {
		case c == '\n' || c == '\r':
			return "o caminho da ISO não pode ter quebra de linha"
		case c < 0x20 || c == 0x7f:
			return "o caminho da ISO não pode ter caractere de controle"
		case c == ',':
			return "o caminho da ISO não pode ter vírgula (o QEMU a lê como separador de opções): renomeie o arquivo"
		}
	}
	if !strings.HasPrefix(iso, "/") {
		return "o caminho da ISO tem de ser inteiro, começando por /"
	}
	if len(iso) > 4096 {
		return "o caminho da ISO é comprido demais"
	}
	return ""
}

// Valida confere o pedido de uma máquina nova com as regras do contrato (o
// que não depende do disco). nil se está bom.
func Valida(m Maquina) *ErroDeCampo {
	erro := func(campo, frase string) *ErroDeCampo { return &ErroDeCampo{Campo: campo, Frase: frase} }
	if f := ErroDoNome(m.Nome); f != "" {
		return erro("nome", f)
	}
	if m.MemoriaMB < MemoriaMinima || m.MemoriaMB > MemoriaMaxima {
		return erro("memoria_mb", "a memória vai de 512 a 65536 MB")
	}
	if m.CPUs < 1 || m.CPUs > CPUsMaximas {
		return erro("cpus", "as CPUs vão de 1 a 32")
	}
	if m.DiscoGB < 1 || m.DiscoGB > DiscoMaximo {
		return erro("disco_gb", "o disco vai de 1 a 2000 GB")
	}
	if f := ErroDaISO(m.ISO); f != "" {
		return erro("iso", f)
	}
	return nil
}

// Loja são as duas pastas: a das máquinas e a de execução (que pode faltar).
type Loja struct {
	Raiz     string
	Execucao string
}

// Pasta é a pasta de uma máquina; Config, Disco e Vars, os arquivos dela.
func (l Loja) Pasta(nome string) string  { return filepath.Join(l.Raiz, nome) }
func (l Loja) Config(nome string) string { return filepath.Join(l.Raiz, nome, "maquina.json") }
func (l Loja) Disco(nome string) string  { return filepath.Join(l.Raiz, nome, "disco.qcow2") }
func (l Loja) Vars(nome string) string   { return filepath.Join(l.Raiz, nome, "vars.fd") }

// Onde ficam, com a máquina ligada, o socket do QMP, o pid e os registros.
func (l Loja) Corrida(nome string) string { return filepath.Join(l.Execucao, nome) }
func (l Loja) Socket(nome string) string  { return filepath.Join(l.Execucao, nome, "qmp.sock") }
func (l Loja) Pid(nome string) string     { return filepath.Join(l.Execucao, nome, "pid") }
func (l Loja) Log(nome string) string     { return filepath.Join(l.Execucao, nome, "qemu.log") }
func (l Loja) Depuracao(nome string) string {
	return filepath.Join(l.Execucao, nome, "qemu-depuracao.log")
}

// Le lê a configuração de uma máquina.
func (l Loja) Le(nome string) (Maquina, error) {
	var m Maquina
	if !NomeBom(nome) {
		return m, fmt.Errorf("%q não é nome de máquina", nome)
	}
	b, err := os.ReadFile(l.Config(nome))
	if errors.Is(err, os.ErrNotExist) {
		return m, fmt.Errorf("não conheço a máquina %s", nome)
	}
	if err != nil {
		return m, fmt.Errorf("não consegui ler a máquina %s: %v", nome, err)
	}
	if err := json.Unmarshal(b, &m); err != nil || m.Nome != nome {
		return m, fmt.Errorf("o maquina.json de %s está estragado", nome)
	}
	return m, nil
}

// Todas devolve as máquinas, em ordem de nome. Pasta sem um maquina.json
// legível não entra.
func (l Loja) Todas() []Maquina {
	maquinas := []Maquina{}
	itens, err := os.ReadDir(l.Raiz)
	if err != nil {
		return maquinas
	}
	for _, item := range itens {
		if !item.IsDir() {
			continue
		}
		if m, err := l.Le(item.Name()); err == nil {
			maquinas = append(maquinas, m)
		}
	}
	sort.Slice(maquinas, func(i, j int) bool { return maquinas[i].Nome < maquinas[j].Nome })
	return maquinas
}

// Existe diz se já há alguma coisa com esse nome na pasta das máquinas.
func (l Loja) Existe(nome string) bool {
	_, err := os.Lstat(l.Pasta(nome))
	return err == nil
}

// CriaPasta cria a pasta de uma máquina nova (700), e a raiz se faltar.
func (l Loja) CriaPasta(nome string) error {
	if !NomeBom(nome) {
		return fmt.Errorf("%q não é nome de máquina", nome)
	}
	if err := os.MkdirAll(l.Raiz, 0o700); err != nil {
		return fmt.Errorf("não consegui criar %s: %v", l.Raiz, err)
	}
	if err := os.Mkdir(l.Pasta(nome), 0o700); err != nil {
		return fmt.Errorf("não consegui criar a pasta da máquina: %v", err)
	}
	return os.Chmod(l.Pasta(nome), 0o700)
}

// Grava escreve o maquina.json de forma atômica (temporário e rename).
func (l Loja) Grava(m Maquina) error {
	if !NomeBom(m.Nome) {
		return fmt.Errorf("%q não é nome de máquina", m.Nome)
	}
	dados, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(l.Pasta(m.Nome), ".maquina-*.tmp")
	if err != nil {
		return fmt.Errorf("não consegui gravar a máquina: %v", err)
	}
	nome := tmp.Name()
	falha := func(err error) error {
		tmp.Close()
		os.Remove(nome)
		return fmt.Errorf("não consegui gravar a máquina: %v", err)
	}
	if _, err := tmp.Write(append(dados, '\n')); err != nil {
		return falha(err)
	}
	if err := tmp.Sync(); err != nil {
		return falha(err)
	}
	if err := tmp.Close(); err != nil {
		return falha(err)
	}
	if err := os.Rename(nome, l.Config(m.Nome)); err != nil {
		return falha(err)
	}
	return nil
}

// CopiaVars copia o OVMF_VARS do sistema para a máquina: cada uma tem o seu.
func (l Loja) CopiaVars(nome, modelo string) error {
	de, err := os.Open(modelo)
	if err != nil {
		return fmt.Errorf("não consegui ler %s: %v", modelo, err)
	}
	defer de.Close()
	para, err := os.OpenFile(l.Vars(nome), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("não consegui criar o vars.fd da máquina: %v", err)
	}
	if _, err := io.Copy(para, de); err != nil {
		para.Close()
		return fmt.Errorf("não consegui copiar o OVMF_VARS: %v", err)
	}
	return para.Close()
}

// filhoDireto confere que `pasta` é uma pasta de verdade (não um atalho) e um
// filho direto de `raiz`, depois de resolver os atalhos do caminho.
func filhoDireto(raiz, pasta string) error {
	if raiz == "" || pasta == "" {
		return errors.New("caminho vazio")
	}
	info, err := os.Lstat(pasta)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("%s não é uma pasta de verdade", pasta)
	}
	raizReal, err := filepath.EvalSymlinks(raiz)
	if err != nil {
		return err
	}
	real, err := filepath.EvalSymlinks(pasta)
	if err != nil {
		return err
	}
	if filepath.Dir(real) != raizReal || real == raizReal || real == "/" {
		return fmt.Errorf("%s não está dentro de %s", pasta, raiz)
	}
	return nil
}

// Apaga remove a pasta de uma máquina e a de execução dela, e nada fora
// delas: o nome tem de ser um nome de máquina, e cada pasta, um filho direto
// da sua raiz (um atalho no lugar da pasta é recusado).
func (l Loja) Apaga(nome string) error {
	if !NomeBom(nome) {
		return fmt.Errorf("%q não é nome de máquina", nome)
	}
	if err := filhoDireto(l.Raiz, l.Pasta(nome)); err != nil {
		return fmt.Errorf("não apago: %v", err)
	}
	if l.Execucao != "" {
		if err := filhoDireto(l.Execucao, l.Corrida(nome)); err == nil {
			_ = os.RemoveAll(l.Corrida(nome))
		}
	}
	if err := os.RemoveAll(l.Pasta(nome)); err != nil {
		return fmt.Errorf("não consegui apagar a máquina: %v", err)
	}
	return nil
}

// PreparaCorrida cria a pasta de execução da máquina (700) e tira dela o
// socket e o pid de uma corrida anterior.
func (l Loja) PreparaCorrida(nome string) error {
	if l.Execucao == "" || !NomeBom(nome) {
		return errors.New("não há onde pôr o socket da máquina (falta o XDG_RUNTIME_DIR)")
	}
	if err := os.MkdirAll(l.Corrida(nome), 0o700); err != nil {
		return fmt.Errorf("não consegui criar %s: %v", l.Corrida(nome), err)
	}
	l.LimpaCorrida(nome)
	return nil
}

// LimpaCorrida tira o socket e o pid que sobraram de uma máquina que morreu.
func (l Loja) LimpaCorrida(nome string) {
	if l.Execucao == "" || !NomeBom(nome) {
		return
	}
	_ = os.Remove(l.Socket(nome))
	_ = os.Remove(l.Pid(nome))
}

// AbreLog abre (zerando) o arquivo que recebe o que o QEMU reclama.
func (l Loja) AbreLog(nome string) (*os.File, error) {
	return os.OpenFile(l.Log(nome), os.O_WRONLY|os.O_CREATE|os.O_TRUNC|os.O_APPEND, 0o600)
}

// FimDoLog devolve as últimas linhas com conteúdo do qemu.log (até `linhas`,
// cada uma cortada), numa frase só.
func (l Loja) FimDoLog(nome string, linhas int) string {
	b, err := os.ReadFile(l.Log(nome))
	if err != nil {
		return ""
	}
	if len(b) > 16*1024 {
		b = b[len(b)-16*1024:]
	}
	var boas []string
	for _, linha := range strings.Split(string(b), "\n") {
		linha = strings.TrimSpace(strings.Map(func(c rune) rune {
			if c < 0x20 || c == 0x7f {
				return -1
			}
			return c
		}, linha))
		if linha == "" {
			continue
		}
		if letras := []rune(linha); len(letras) > 200 {
			linha = string(letras[:200]) + "…"
		}
		boas = append(boas, linha)
	}
	if len(boas) > linhas {
		boas = boas[len(boas)-linhas:]
	}
	return strings.Join(boas, " / ")
}

// NoDisco diz quanto um arquivo ocupa de verdade (um qcow2 novo é quase
// vazio, por maior que seja o disco que ele representa). -1 se não deu.
func NoDisco(caminho string) int64 {
	info, err := os.Stat(caminho)
	if err != nil {
		return -1
	}
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		return st.Blocks * 512
	}
	return info.Size()
}

// AbreParaEscrita diz se um arquivo que já existe abre para leitura e escrita
// (é o que o QEMU precisa do /dev/kvm). Não cria nem escreve nada.
func AbreParaEscrita(caminho string) bool {
	f, err := os.OpenFile(caminho, os.O_RDWR, 0)
	if err != nil {
		return false
	}
	f.Close()
	return true
}
