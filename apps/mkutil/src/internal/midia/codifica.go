package midia

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"mkutil/internal/base"
)

// ErrCancelado é o que Comprime devolve quando mandaram parar.
var ErrCancelado = errors.New("cancelado")

// FasePlano é a primeira linha do `comprimir`.
type FasePlano struct {
	Fase     string `json:"fase"`
	Saida    string `json:"saida"`
	Passadas int    `json:"passadas"`
	VideoBps *int64 `json:"video_bps,omitempty"`
	AudioBps *int64 `json:"audio_bps,omitempty"`
	Largura  int    `json:"largura"`
	Altura   int    `json:"altura"`
}

// FaseCodificando é uma linha de progresso.
type FaseCodificando struct {
	Fase       string   `json:"fase"`
	Passada    int      `json:"passada"`
	Passadas   int      `json:"passadas"`
	Pct        float64  `json:"pct"`
	Velocidade *float64 `json:"velocidade"`
	Resta      *int64   `json:"resta_s"`
	Tamanho    int64    `json:"tamanho_bytes"`
}

// Fim é a última linha do `comprimir`.
type Fim struct {
	Fase     string  `json:"fase"`
	Ok       bool    `json:"ok"`
	Saida    string  `json:"saida"`
	Antes    int64   `json:"antes_bytes"`
	Depois   int64   `json:"depois_bytes"`
	Economia float64 `json:"economia_pct"`
	Maior    bool    `json:"maior"`
}

// progresso junta as linhas "chave=valor" do `-progress pipe:1` e, a cada
// bloco (que acaba em "progress=…"), devolve uma fase.
type progresso struct {
	duracao           float64
	passada, passadas int
	tempo             float64 // segundos já codificados
	velocidade        *float64
	tamanho           int64
}

func (p *progresso) linha(l string) *FaseCodificando {
	chave, valor, ok := strings.Cut(strings.TrimSpace(l), "=")
	if !ok {
		return nil
	}
	valor = strings.TrimSpace(valor)
	switch chave {
	case "out_time_us", "out_time_ms": // os dois vêm em microssegundos
		if v, err := strconv.ParseFloat(valor, 64); err == nil && v >= 0 {
			p.tempo = v / 1e6
		}
	case "speed":
		// "N/A" no meio do caminho não apaga a velocidade que já se sabia.
		if v, err := strconv.ParseFloat(strings.TrimSuffix(valor, "x"), 64); err == nil && v > 0 && !math.IsInf(v, 0) {
			v = math.Round(v*100) / 100
			p.velocidade = &v
		}
	case "total_size":
		if v, err := strconv.ParseInt(valor, 10, 64); err == nil && v >= 0 {
			p.tamanho = v
		}
	case "progress":
		return p.fase()
	}
	return nil
}

func (p *progresso) fase() *FaseCodificando {
	feito := 0.0
	if p.duracao > 0 {
		feito = math.Min(1, p.tempo/p.duracao)
	}
	pct := (float64(p.passada-1) + feito) / float64(p.passadas) * 100
	f := &FaseCodificando{Fase: "codificando", Passada: p.passada, Passadas: p.passadas,
		Pct: math.Round(math.Min(pct, 99.9)*10) / 10, Velocidade: p.velocidade, Tamanho: p.tamanho}
	if p.velocidade != nil && p.duracao > 0 {
		falta := (p.duracao*(1-feito) + p.duracao*float64(p.passadas-p.passada)) / *p.velocidade
		r := int64(math.Ceil(falta))
		f.Resta = &r
	}
	return f
}

// Comprime faz o `comprimir` inteiro. `solta` recebe cada fase (menos a
// última, que é o Fim devolvido); fechar `cancela` mata o ffmpeg e apaga a
// saída pela metade (aí o erro é ErrCancelado).
func Comprime(arquivo string, pedido Pedido, solta func(any) error, cancela <-chan struct{}) (*Fim, error) {
	var fim *Fim
	err := comAtalho(arquivo, func(link, pasta string) error {
		m, err := sonda(arquivo, link)
		if err != nil {
			return err
		}
		tem, err := Codificadores()
		if err != nil {
			return err
		}
		plano, err := Planeja(m, pedido, tem)
		if err != nil {
			return err
		}
		fim, err = roda(m, plano, link, pasta, solta, cancela)
		return err
	})
	return fim, err
}

func roda(m *Midia, plano *Plano, link, pasta string, solta func(any) error, cancela <-chan struct{}) (*Fim, error) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, errors.New("não achei o ffmpeg nesta máquina: instale o pacote ffmpeg")
	}
	// O ffmpeg escreve num arquivo escondido ao lado do original; só no fim
	// ele ganha o nome. A extensão fica no fim para o ffmpeg saber o formato.
	parcial, err := os.CreateTemp(filepath.Dir(m.Arquivo), ".mkcomprimir-*.parcial"+plano.Ext)
	if err != nil {
		return nil, fmt.Errorf("não dá para escrever na pasta do original (%s): %v", base.Til(filepath.Dir(m.Arquivo)), err)
	}
	parcial.Close()
	pronto := false
	defer func() {
		if !pronto {
			_ = os.Remove(parcial.Name())
		}
	}()

	primeira := FasePlano{Fase: "plano", Saida: nomeLivre(m.Arquivo, plano.Sufixo, plano.Ext), Passadas: len(plano.Passadas),
		Largura: plano.Largura, Altura: plano.Altura}
	if plano.VideoBps > 0 {
		primeira.VideoBps, primeira.AudioBps = &plano.VideoBps, &plano.AudioBps
	}
	if solta(primeira) != nil {
		return nil, ErrCancelado
	}

	troca := strings.NewReplacer("{entrada}", link, "{saida}", parcial.Name(), "{passada}", filepath.Join(pasta, "passada"))
	for i, modelo := range plano.Passadas {
		args := make([]string, len(modelo))
		for j, a := range modelo {
			args[j] = troca.Replace(a)
		}
		p := &progresso{passada: i + 1, passadas: len(plano.Passadas)}
		if m.Duracao != nil {
			p.duracao = *m.Duracao
		}
		if err := umaPassada(ffmpeg, args, pasta, p, m.Tipo == "video", solta, cancela); err != nil {
			return nil, err
		}
	}

	fi, err := os.Stat(parcial.Name())
	if err != nil || fi.Size() == 0 {
		return nil, errors.New("o ffmpeg terminou sem escrever a saída")
	}
	_ = os.Chmod(parcial.Name(), 0o644)
	saida, err := daNome(parcial.Name(), m.Arquivo, plano.Sufixo, plano.Ext)
	if err != nil {
		return nil, err
	}
	pronto = true
	f := &Fim{Fase: "fim", Ok: true, Saida: saida, Antes: m.Tamanho, Depois: fi.Size(), Maior: fi.Size() > m.Tamanho}
	if m.Tamanho > 0 {
		f.Economia = math.Round((1-float64(fi.Size())/float64(m.Tamanho))*1000) / 10
	}
	return f, nil
}

// daNome põe o arquivo pronto no nome livre, sem nunca cair em cima de um
// que existe: o link físico falha se o nome já foi tomado, e aí tenta o
// número seguinte.
func daNome(parcial, original, sufixo, ext string) (string, error) {
	for tentativa := 0; tentativa < 50; tentativa++ {
		destino := nomeLivre(original, sufixo, ext)
		err := os.Link(parcial, destino)
		if err == nil {
			_ = os.Remove(parcial)
			return destino, nil
		}
		if errors.Is(err, os.ErrExist) {
			continue
		}
		// Sistema de arquivos sem link físico (um pendrive em FAT): cria o
		// nome, só se não existir, e move para cima dele.
		f, err := os.OpenFile(destino, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("não consegui dar o nome à saída: %v", err)
		}
		f.Close()
		if err := os.Rename(parcial, destino); err != nil {
			_ = os.Remove(destino)
			return "", fmt.Errorf("não consegui dar o nome à saída: %v", err)
		}
		return destino, nil
	}
	return "", errors.New("não consegui achar um nome livre para a saída")
}

// umaPassada roda o ffmpeg uma vez, lendo o progresso.
func umaPassada(ffmpeg string, args []string, pasta string, p *progresso, mostra bool, solta func(any) error, cancela <-chan struct{}) error {
	cmd := exec.Command(ffmpeg, args...)
	cmd.Dir = pasta // os arquivos soltos da primeira passada caem na pasta temporária
	saida, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	erros := &rabo{max: 16 << 10}
	cmd.Stderr = erros
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("não consegui rodar o ffmpeg: %v", err)
	}
	acabou := make(chan struct{})
	parou := make(chan bool, 1)
	// Quem mata: mandaram cancelar, ou ninguém mais lê o que o motor escreve.
	mata := make(chan struct{}, 1)
	go func() {
		select {
		case <-cancela:
		case <-mata:
		case <-acabou:
			parou <- false
			return
		}
		_ = cmd.Process.Signal(syscall.SIGTERM)
		select {
		case <-acabou:
		case <-time.After(2 * time.Second):
			_ = cmd.Process.Kill()
		}
		parou <- true
	}()
	linhas := bufio.NewReader(saida)
	for {
		l, err := linhas.ReadString('\n')
		if f := p.linha(l); f != nil && mostra {
			if solta(f) != nil {
				select {
				case mata <- struct{}{}:
				default:
				}
			}
		}
		if err != nil {
			break
		}
	}
	_, _ = io.Copy(io.Discard, saida)
	err = cmd.Wait()
	close(acabou)
	if <-parou {
		return ErrCancelado
	}
	if err != nil {
		if frase := base.Resumo(erros.String()); frase != "" {
			return fmt.Errorf("o ffmpeg não conseguiu: %s", frase)
		}
		return fmt.Errorf("o ffmpeg não conseguiu (%v)", err)
	}
	return nil
}

// rabo guarda só o fim do que escrevem nele: o erro do ffmpeg vem por último.
type rabo struct {
	b   []byte
	max int
}

func (r *rabo) Write(p []byte) (int, error) {
	r.b = append(r.b, p...)
	if len(r.b) > r.max {
		r.b = r.b[len(r.b)-r.max:]
	}
	return len(p), nil
}

func (r *rabo) String() string { return string(r.b) }

// Descarta apaga uma saída do `comprimir` (a que ficou maior que o original).
// Só apaga o que tem cara de saída: um arquivo comum, na pasta do original,
// com o nome começando pelo do original, e que não é o original.
func Descarta(saida, original string) (string, error) {
	s, err := filepath.Abs(saida)
	if err != nil {
		return "", fmt.Errorf("não entendi o caminho %s", saida)
	}
	o, err := filepath.Abs(original)
	if err != nil {
		return "", fmt.Errorf("não entendi o caminho %s", original)
	}
	nomeO := filepath.Base(o)
	raiz := strings.TrimSuffix(nomeO, filepath.Ext(nomeO))
	if raiz == "" {
		raiz = nomeO
	}
	nomeS := filepath.Base(s)
	if s == o || filepath.Dir(s) != filepath.Dir(o) || !strings.HasPrefix(nomeS, raiz) || nomeS == nomeO || len(nomeS) <= len(raiz) {
		return "", base.Recusa("%s não é uma saída de %s: não apago", base.Til(s), nomeO)
	}
	fo, err := os.Stat(o)
	if err != nil {
		return "", base.Recusa("não achei o original (%s): não apago nada", base.Til(o))
	}
	fs, err := os.Lstat(s)
	if err != nil {
		return "", fmt.Errorf("não achei %s", base.Til(s))
	}
	if !fs.Mode().IsRegular() || os.SameFile(fo, fs) {
		return "", base.Recusa("%s não é uma saída de %s: não apago", base.Til(s), nomeO)
	}
	if err := os.Remove(s); err != nil {
		return "", fmt.Errorf("não consegui apagar %s: %v", base.Til(s), err)
	}
	return s, nil
}
