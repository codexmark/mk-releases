package miniatura

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"image"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"mkfiles/internal/base"
	"mkfiles/internal/mime"
)

// Quantos trabalham ao mesmo tempo (dois processos, no máximo) e quanto um
// miniaturizador de fora pode demorar (os testes encurtam).
const Trabalhadores = 2

var Prazo = 10 * time.Second

// Os lados do contrato.
var lados = map[string]int{"normal": 128, "large": 256, "x-large": 512}

// pedido é uma miniatura na fila.
type pedido struct {
	caminho, tamanho, grupo string
	prioridade              int
	ordem                   int64 // quanto maior, mais novo
}

type fabrica struct {
	s        *base.Sessao
	trava    sync.Mutex
	fila     []*pedido
	ordem    int64
	ativos   int
	fechada  bool
	contexto context.Context
	cancela  context.CancelFunc

	umaVez sync.Once
	deFora map[string][]string // mime -> argv do miniaturizador (com os códigos %)
}

// Ligar põe na sessão os pedidos miniatura e miniatura-cancelar.
func Ligar(s *base.Sessao) {
	f := &fabrica{s: s}
	f.contexto, f.cancela = context.WithCancel(context.Background())
	s.Ligar("miniatura", f.pedir)
	s.Ligar("miniatura-cancelar", f.cancelar)
	s.AoFechar(func() {
		f.trava.Lock()
		f.fechada = true
		sobraram := len(f.fila)
		f.fila = nil
		f.trava.Unlock()
		for i := 0; i < sobraram; i++ {
			s.Descontou()
		}
		f.cancela() // derruba o miniaturizador que estiver rodando
	})
}

func (f *fabrica) pedir(p *base.Pedido) {
	var q struct {
		Caminhos   []string `json:"caminhos"`
		Tamanho    string   `json:"tamanho"`
		Grupo      string   `json:"grupo"`
		Prioridade int      `json:"prioridade"`
	}
	_ = p.Ler(&q)
	if q.Tamanho == "" {
		q.Tamanho = "normal"
	}
	if lados[q.Tamanho] == 0 {
		p.FalhaDeCampo("tamanho", "o tamanho é normal, large ou x-large")
		return
	}
	if err := base.Caminhos(q.Caminhos); err != nil {
		p.Falha(err.Error())
		return
	}
	f.trava.Lock()
	if f.fechada {
		f.trava.Unlock()
		p.Responde(base.Campos{"aceitos": 0})
		return
	}
	naFila := map[string]*pedido{}
	for _, ja := range f.fila {
		naFila[ja.tamanho+"\x00"+ja.caminho] = ja
	}
	// O primeiro da lista é o primeiro a sair: ganha a ordem mais alta.
	for i := len(q.Caminhos) - 1; i >= 0; i-- {
		f.ordem++
		c := q.Caminhos[i]
		if ja := naFila[q.Tamanho+"\x00"+c]; ja != nil {
			// Pedido de novo: sobe na fila, e passa a ser do grupo de agora.
			ja.grupo, ja.prioridade, ja.ordem = q.Grupo, q.Prioridade, f.ordem
			continue
		}
		novo := &pedido{caminho: c, tamanho: q.Tamanho, grupo: q.Grupo, prioridade: q.Prioridade, ordem: f.ordem}
		f.fila = append(f.fila, novo)
		naFila[q.Tamanho+"\x00"+c] = novo
		f.s.Conta()
	}
	for f.ativos < Trabalhadores && f.ativos < len(f.fila) {
		f.ativos++
		go f.trabalha()
	}
	f.trava.Unlock()
	p.Responde(base.Campos{"aceitos": len(q.Caminhos)})
}

func (f *fabrica) cancelar(p *base.Pedido) {
	var q struct {
		Grupo string `json:"grupo"`
	}
	_ = p.Ler(&q)
	f.trava.Lock()
	fica := f.fila[:0]
	tirados := 0
	for _, ped := range f.fila {
		if ped.grupo == q.Grupo {
			tirados++
		} else {
			fica = append(fica, ped)
		}
	}
	f.fila = fica
	f.trava.Unlock()
	for i := 0; i < tirados; i++ {
		f.s.Descontou()
	}
	p.Responde(base.Campos{"cancelados": tirados})
}

// proximo tira da fila o de maior prioridade; no empate, o mais novo.
func (f *fabrica) proximo() *pedido {
	f.trava.Lock()
	defer f.trava.Unlock()
	if len(f.fila) == 0 || f.fechada {
		f.ativos--
		return nil
	}
	melhor := 0
	for i, ped := range f.fila {
		m := f.fila[melhor]
		if ped.prioridade > m.prioridade || (ped.prioridade == m.prioridade && ped.ordem > m.ordem) {
			melhor = i
		}
	}
	ped := f.fila[melhor]
	f.fila = append(f.fila[:melhor], f.fila[melhor+1:]...)
	return ped
}

func (f *fabrica) trabalha() {
	for {
		ped := f.proximo()
		if ped == nil {
			return
		}
		var arquivo any
		if a := f.faz(ped.caminho, ped.tamanho); a != "" {
			arquivo = a
		}
		f.s.Evento("miniatura", base.Campos{"caminho": ped.caminho, "tamanho": ped.tamanho, "arquivo": arquivo})
		f.s.Descontou()
	}
}

func pastaDasMiniaturas() string { return base.Cache("thumbnails") }

func resumo(uri string) string {
	soma := md5.Sum([]byte(uri))
	return hex.EncodeToString(soma[:])
}

// vale diz se um PNG guardado é a miniatura (ou a falha) desse arquivo como
// ele está agora.
func vale(arquivo, uri string, mtime int64) bool {
	dados, err := os.ReadFile(arquivo)
	if err != nil {
		return false
	}
	t := textosDe(dados)
	return t["Thumb::URI"] == uri && t["Thumb::MTime"] == strconv.FormatInt(mtime, 10)
}

// falhou diz se alguém (nós ou outro programa) já anotou que esse arquivo, com
// essa data, não dá miniatura.
func falhou(nome, uri string, mtime int64) bool {
	pastas, _ := os.ReadDir(filepath.Join(pastaDasMiniaturas(), "fail"))
	for _, p := range pastas {
		if p.IsDir() && vale(filepath.Join(pastaDasMiniaturas(), "fail", p.Name(), nome), uri, mtime) {
			return true
		}
	}
	return false
}

// guarda grava um PNG na pasta das miniaturas: pasta 0700, arquivo 0600,
// escrita atômica.
func guarda(arquivo string, dados []byte) error {
	if err := os.MkdirAll(filepath.Dir(arquivo), 0o700); err != nil {
		return err
	}
	return base.GravaAtomico(arquivo, dados, 0o600)
}

func textosPara(uri string, mtime int64) [][2]string {
	return [][2]string{{"Thumb::URI", uri}, {"Thumb::MTime", strconv.FormatInt(mtime, 10)}, {"Software", "MkFiles"}}
}

// faz devolve o arquivo da miniatura de um caminho ("" se não há como): a que
// já está guardada e ainda vale, ou uma nova.
func (f *fabrica) faz(caminho, tamanho string) string {
	info, err := os.Stat(caminho)
	if err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
		return ""
	}
	tipo := mime.DoArquivo(caminho, info.Size(), true)
	// Uma miniatura não ganha miniatura: ela mesma serve.
	if base.Dentro(pastaDasMiniaturas(), caminho) {
		if tipo == "image/png" {
			return caminho
		}
		return ""
	}
	uri, mtime := base.URI(caminho), info.ModTime().Unix()
	nome := resumo(uri) + ".png"
	arquivo := filepath.Join(pastaDasMiniaturas(), tamanho, nome)
	if vale(arquivo, uri, mtime) {
		return arquivo
	}
	if falhou(nome, uri, mtime) {
		return ""
	}
	var dados []byte
	switch tipo {
	case "image/png", "image/jpeg", "image/gif":
		var img image.Image
		if img, err = abre(caminho); err == nil {
			pequena := reduz(img, lados[tamanho])
			if tipo == "image/jpeg" {
				// A foto de celular em pé está gravada deitada: o Exif diz como virar.
				pequena = orienta(pequena, orientacaoDe(caminho))
			}
			dados, err = codifica(pequena, textosPara(uri, mtime))
		}
	default:
		molde := f.miniaturizador(tipo)
		if molde == nil {
			return "" // ninguém sabe fazer: não é uma falha a anotar
		}
		dados, err = f.deForaFaz(molde, caminho, uri, mtime, tamanho)
	}
	if err == nil {
		err = guarda(arquivo, dados)
	}
	if err != nil {
		if f.contexto.Err() == nil {
			// Anota a falha, para não tentar de novo enquanto o arquivo não muda.
			if marca, e := codifica(image.NewGray(image.Rect(0, 0, 1, 1)), textosPara(uri, mtime)); e == nil {
				_ = guarda(filepath.Join(pastaDasMiniaturas(), "fail", "mkfiles", nome), marca)
			}
		}
		return ""
	}
	return arquivo
}

// miniaturizador acha o comando do sistema que faz a miniatura de um tipo (nil
// se nenhum). Os .thumbnailer são lidos uma vez por sessão.
func (f *fabrica) miniaturizador(tipo string) []string {
	f.umaVez.Do(func() {
		f.deFora = map[string][]string{}
		pasta := base.Raiz("MKFILES_MINIATURIZADORES", "/usr/share/thumbnailers")
		arquivos, _ := filepath.Glob(filepath.Join(pasta, "*.thumbnailer"))
		sort.Strings(arquivos)
		for _, arq := range arquivos {
			b, err := os.ReadFile(arq)
			if err != nil || len(b) > 1<<20 {
				continue
			}
			campos := map[string]string{}
			dentro := false
			for _, linha := range strings.Split(string(b), "\n") {
				linha = strings.TrimSpace(linha)
				if strings.HasPrefix(linha, "[") {
					dentro = linha == "[Thumbnailer Entry]"
					continue
				}
				if chave, valor, ok := strings.Cut(linha, "="); ok && dentro {
					campos[strings.TrimSpace(chave)] = strings.TrimSpace(valor)
				}
			}
			molde := palavras(campos["Exec"])
			if len(molde) == 0 {
				continue
			}
			if t := campos["TryExec"]; t != "" {
				if _, err := exec.LookPath(t); err != nil {
					continue
				}
			}
			for _, m := range strings.Split(campos["MimeType"], ";") {
				if m = strings.TrimSpace(m); m != "" && f.deFora[m] == nil {
					f.deFora[m] = molde
				}
			}
		}
	})
	return f.deFora[tipo]
}

// palavras separa a linha Exec em argumentos (espaços; aspas duplas juntam).
// Aspas sem par: nenhum argumento.
func palavras(linha string) []string {
	var args []string
	var atual strings.Builder
	tem, aspas := false, false
	for i := 0; i < len(linha); i++ {
		c := linha[i]
		switch {
		case aspas && c == '\\' && i+1 < len(linha):
			i++
			atual.WriteByte(linha[i])
		case c == '"':
			aspas, tem = !aspas, true
		case !aspas && (c == ' ' || c == '\t'):
			if tem {
				args = append(args, atual.String())
				atual.Reset()
				tem = false
			}
		default:
			atual.WriteByte(c)
			tem = true
		}
	}
	if aspas {
		return nil
	}
	if tem {
		args = append(args, atual.String())
	}
	return args
}

// deForaFaz roda o miniaturizador do sistema (sem shell, com prazo, num grupo
// de processos próprio para morrer inteiro) e devolve o PNG já com os blocos
// tEXt.
func (f *fabrica) deForaFaz(molde []string, caminho, uri string, mtime int64, tamanho string) ([]byte, error) {
	pasta := filepath.Join(pastaDasMiniaturas(), tamanho)
	if err := os.MkdirAll(pasta, 0o700); err != nil {
		return nil, err
	}
	tmp, err := os.CreateTemp(pasta, ".mkfiles-*.png")
	if err != nil {
		return nil, err
	}
	saida := tmp.Name()
	tmp.Close()
	defer os.Remove(saida) // o temporário criado aqui mesmo
	troca := strings.NewReplacer("%%", "%", "%i", caminho, "%u", uri, "%o", saida, "%s", strconv.Itoa(lados[tamanho]))
	argv := make([]string, len(molde))
	for i, parte := range molde {
		argv[i] = troca.Replace(parte)
	}
	ctx, cancela := context.WithTimeout(f.contexto, Prazo)
	defer cancela()
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = time.Second
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, errors.New("o miniaturizador demorou demais")
		}
		return nil, err
	}
	info, err := os.Stat(saida)
	if err != nil || info.Size() == 0 || info.Size() > LimiteDoArquivo {
		return nil, errors.New("o miniaturizador não fez a miniatura")
	}
	dados, err := os.ReadFile(saida)
	if err != nil {
		return nil, err
	}
	return comTextos(dados, textosPara(uri, mtime))
}
