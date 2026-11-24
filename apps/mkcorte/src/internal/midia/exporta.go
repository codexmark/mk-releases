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

	"mkcorte/internal/base"
)

// ErrCancelado é o que Exporta devolve quando mandaram parar.
var ErrCancelado = errors.New("cancelado")

// FaseExportando é uma linha de progresso do `exportar`.
type FaseExportando struct {
	Fase       string   `json:"fase"`
	Pct        float64  `json:"pct"`
	Velocidade *float64 `json:"velocidade"`
	Resta      *int64   `json:"resta_s"`
	Tamanho    int64    `json:"tamanho_bytes"`
}

// Fim é a última linha do `exportar`.
type Fim struct {
	Fase    string  `json:"fase"`
	Ok      bool    `json:"ok"`
	Saida   string  `json:"saida"`
	Modo    string  `json:"modo"`
	Duracao float64 `json:"duracao_s"`
	Tamanho int64   `json:"tamanho_bytes"`
}

// progresso junta as linhas "chave=valor" do `-progress pipe:1` e, a cada
// bloco (que acaba em "progress=…"), devolve uma fase.
type progresso struct {
	duracao    float64
	tempo      float64 // segundos já escritos
	velocidade *float64
	tamanho    int64
}

func (p *progresso) linha(l string) *FaseExportando {
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

func (p *progresso) fase() *FaseExportando {
	feito := 0.0
	if p.duracao > 0 {
		feito = math.Min(1, p.tempo/p.duracao)
	}
	f := &FaseExportando{Fase: "exportando", Pct: math.Round(math.Min(feito*100, 99.9)*10) / 10, Velocidade: p.velocidade, Tamanho: p.tamanho}
	if p.velocidade != nil && p.duracao > 0 {
		r := int64(math.Ceil(p.duracao * (1 - feito) / *p.velocidade))
		f.Resta = &r
	}
	return f
}

// SaidaPadrao é o nome que a exportação ganha quando ninguém escolheu um: ao
// lado do primeiro vídeo, "nome (corte).mp4".
func SaidaPadrao(primeiroVideo string) string {
	pasta, nome := filepath.Split(primeiroVideo)
	raiz := strings.TrimSuffix(nome, filepath.Ext(nome))
	if raiz == "" {
		raiz = nome
	}
	return filepath.Join(pasta, raiz+" (corte).mp4")
}

// comExtensao troca a extensão do caminho pela dada (a saída é sempre MP4; o
// quadro, sempre PNG), sem mexer num nome que já a tem.
func comExtensao(caminho, ext string) string {
	atual := filepath.Ext(caminho)
	if strings.EqualFold(atual, ext) {
		return caminho
	}
	if extensaoSimples.MatchString(strings.ToLower(atual)) && len(atual) < len(filepath.Base(caminho)) {
		caminho = strings.TrimSuffix(caminho, atual)
	}
	return caminho + ext
}

// NomeLivre devolve o caminho pedido se ele está livre, ou o primeiro
// "nome (2).ext", "nome (3).ext"… que estiver.
func NomeLivre(desejado string) string {
	ext := filepath.Ext(desejado)
	raiz := strings.TrimSuffix(desejado, ext)
	for n := 1; ; n++ {
		candidato := desejado
		if n > 1 {
			candidato = raiz + " (" + strconv.Itoa(n) + ")" + ext
		}
		if _, err := os.Lstat(candidato); err != nil {
			return candidato
		}
	}
}

// daNome põe o arquivo pronto no nome livre, sem nunca cair em cima de um
// que existe: o link físico falha se o nome já foi tomado, e aí tenta o
// número seguinte.
func daNome(parcial, desejado string) (string, error) {
	for tentativa := 0; tentativa < 50; tentativa++ {
		destino := NomeLivre(desejado)
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

// destinoDe resolve onde a exportação vai parar: o caminho pedido (ou o
// padrão, ao lado do primeiro vídeo da edição), absoluto e terminando em .mp4.
func destinoDe(e *Edicao, saida string) (string, error) {
	if saida == "" {
		if len(e.Clipes) == 0 || e.Clipes[0].Fonte < 0 || e.Clipes[0].Fonte >= len(e.Fontes) {
			return "", errors.New("a edição está vazia: não há o que exportar")
		}
		return SaidaPadrao(e.Fontes[e.Clipes[0].Fonte]), nil
	}
	abs, err := filepath.Abs(saida)
	if err != nil {
		return "", fmt.Errorf("não entendi o caminho %s", saida)
	}
	return comExtensao(abs, ".mp4"), nil
}

// prepara sonda os vídeos da edição (pelos atalhos) e monta o plano. `f`
// roda com a pasta temporária ainda no lugar.
func prepara(e *Edicao, saida string, f func(plano *Plano, pasta string, links []string) error) error {
	desejado, err := destinoDe(e, saida)
	if err != nil {
		return err
	}
	return comAtalhos(e.Fontes, func(pasta string, links []string) error {
		fontes := make([]*Fonte, len(e.Fontes))
		for k, arquivo := range e.Fontes {
			if fontes[k], err = sonda(arquivo, links[k]); err != nil {
				return err
			}
		}
		plano, err := Planeja(e, fontes, NomeLivre(desejado))
		if err != nil {
			return err
		}
		return f(plano, pasta, links)
	})
}

// Planejar faz o `plano`: sonda os vídeos e diz como a edição sairia, sem
// rodar o ffmpeg nem escrever nada ao lado dos vídeos.
func Planejar(e *Edicao, saida string) (*Plano, error) {
	var plano *Plano
	err := prepara(e, saida, func(p *Plano, _ string, _ []string) error {
		plano = p
		return nil
	})
	return plano, err
}

// Exporta faz o `exportar` inteiro. `solta` recebe cada fase (menos a
// última, que é o Fim devolvido); fechar `cancela` mata o ffmpeg e apaga a
// saída pela metade (aí o erro é ErrCancelado).
func Exporta(e *Edicao, saida string, solta func(any) error, cancela <-chan struct{}) (*Fim, error) {
	var fim *Fim
	err := prepara(e, saida, func(plano *Plano, pasta string, links []string) error {
		var err error
		fim, err = roda(plano, pasta, links, solta, cancela)
		return err
	})
	return fim, err
}

func roda(plano *Plano, pasta string, links []string, solta func(any) error, cancela <-chan struct{}) (*Fim, error) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, errors.New("não achei o ffmpeg nesta máquina: instale o pacote ffmpeg")
	}
	destino := filepath.Dir(plano.Saida)
	if fi, err := os.Stat(destino); err != nil || !fi.IsDir() {
		return nil, fmt.Errorf("a pasta da saída não existe: %s", base.Til(destino))
	}
	// O ffmpeg escreve num arquivo escondido na pasta da saída; só no fim ele
	// ganha o nome.
	parcial, err := os.CreateTemp(destino, ".mkcorte-*.parcial")
	if err != nil {
		return nil, fmt.Errorf("não dá para escrever em %s: %v", base.Til(destino), err)
	}
	parcial.Close()
	pronto := false
	defer func() {
		if !pronto {
			_ = os.Remove(parcial.Name())
		}
	}()

	lista := filepath.Join(pasta, nomeDaLista)
	if plano.Lista != "" {
		if err := os.WriteFile(lista, []byte(plano.Lista), 0o600); err != nil {
			return nil, fmt.Errorf("não consegui preparar a lista de clipes: %v", err)
		}
	}
	if solta(struct {
		Fase string `json:"fase"`
		*Plano
	}{"plano", plano}) != nil {
		return nil, ErrCancelado
	}

	troca := []string{"{saida}", parcial.Name(), "{lista}", lista}
	for k, link := range links {
		troca = append(troca, "{fonte"+strconv.Itoa(k)+"}", link)
	}
	r := strings.NewReplacer(troca...)
	args := make([]string, len(plano.Args))
	for i, a := range plano.Args {
		args[i] = r.Replace(a)
	}
	if err := umaPassada(ffmpeg, args, pasta, &progresso{duracao: plano.Duracao}, solta, cancela); err != nil {
		return nil, err
	}

	fi, err := os.Stat(parcial.Name())
	if err != nil || fi.Size() == 0 {
		return nil, errors.New("o ffmpeg terminou sem escrever a saída")
	}
	_ = os.Chmod(parcial.Name(), 0o644)
	final, err := daNome(parcial.Name(), plano.Saida)
	if err != nil {
		return nil, err
	}
	pronto = true
	return &Fim{Fase: "fim", Ok: true, Saida: final, Modo: plano.Modo, Duracao: plano.Duracao, Tamanho: fi.Size()}, nil
}

// umaPassada roda o ffmpeg uma vez, lendo o progresso.
func umaPassada(ffmpeg string, args []string, pasta string, p *progresso, solta func(any) error, cancela <-chan struct{}) error {
	cmd := exec.Command(ffmpeg, args...)
	cmd.Dir = pasta // a lista do modo rápido aponta os atalhos pelo nome
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
		if f := p.linha(l); f != nil {
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
