// O subcomando `watch`: solta um retrato da máquina a cada intervalo, uma
// linha de JSON por retrato (o formato está em ../CONTRATO.md).
//
// As taxas (CPU, disco, rede, GPU por processo) saem de duas leituras dos
// contadores com o intervalo no meio; por isso o primeiro retrato só sai
// depois de um intervalo, e `--uma-vez` mede duas vezes para soltar um.
//
// O laço não morre por causa de um pedaço: o que não deu para ler vira null
// naquele retrato. Ele acaba, saindo com 0, em três casos: SIGTERM, SIGINT,
// ou o stdout fechou (a janela foi embora). Cada retrato vai numa escrita
// só, então nunca fica JSON pela metade.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"mkpulso/internal/base"
	"mkpulso/internal/coleta"
	"mkpulso/internal/grupos"
	"mkpulso/internal/janelas"
)

func init() {
	registrar("watch", watch)
}

const (
	watchIntervaloPadrao = time.Second
	watchIntervaloMinimo = 250 * time.Millisecond
	watchIntervaloMaximo = time.Hour
)

// watchTotais é o bloco "totais" do retrato.
type watchTotais struct {
	Processos int `json:"processos"`
	Grupos    int `json:"grupos"`
	Threads   int `json:"threads"`
}

// watchRetrato é uma linha do `watch`.
type watchRetrato struct {
	TMs       int64             `json:"t_ms"`
	Maquina   string            `json:"maquina"`
	Nucleos   int               `json:"nucleos"`
	CPU       coleta.CPU        `json:"cpu"`
	Mem       *coleta.Memoria   `json:"mem"`
	Discos    []coleta.Disco    `json:"discos"`
	Montagens []coleta.Montagem `json:"montagens"`
	Rede      []coleta.Rede     `json:"rede"`
	GPUs      []coleta.GPU      `json:"gpus"`
	Bateria   *coleta.Bateria   `json:"bateria"`
	Totais    watchTotais       `json:"totais"`
	Grupos    []grupos.Grupo    `json:"grupos"`
}

// watchRelogio é o tempo visto pelo watch. Os testes trocam por um de
// mentira, que anda sem dormir.
type watchRelogio struct {
	// Mono é o relógio monotônico: só serve para diferenças.
	Mono func() time.Duration
	// Parede é a hora do dia, para o t_ms.
	Parede func() time.Time
	// Espera dorme d, ou até fecharem o canal para. Devolve false se
	// mandaram parar.
	Espera func(d time.Duration, para <-chan struct{}) bool
}

var watchLigouEm = time.Now()

var watchRel = watchRelogio{
	Mono:   func() time.Duration { return time.Since(watchLigouEm) },
	Parede: time.Now,
	Espera: func(d time.Duration, para <-chan struct{}) bool {
		t := time.NewTimer(d)
		defer t.Stop()
		select {
		case <-para:
			return false
		case <-t.C:
			return true
		}
	},
}

// watchOpcoes lê `[--intervalo MS] [--uma-vez]`. Um intervalo abaixo do
// mínimo (250 ms) vira o mínimo.
func watchOpcoes(args []string) (intervalo time.Duration, umaVez bool, err error) {
	intervalo = watchIntervaloPadrao
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--uma-vez":
			umaVez = true
		case a == "--intervalo" || strings.HasPrefix(a, "--intervalo="):
			valor, colado := strings.CutPrefix(a, "--intervalo=")
			if !colado {
				if i+1 >= len(args) {
					return 0, false, errors.New("falta o número de milissegundos depois de --intervalo")
				}
				i++
				valor = args[i]
			}
			ms, erro := strconv.ParseUint(valor, 10, 63)
			if erro != nil {
				return 0, false, fmt.Errorf("o intervalo tem de ser um número de milissegundos, veio %q", valor)
			}
			switch {
			case ms < uint64(watchIntervaloMinimo/time.Millisecond):
				intervalo = watchIntervaloMinimo
			case ms > uint64(watchIntervaloMaximo/time.Millisecond):
				intervalo = watchIntervaloMaximo
			default:
				intervalo = time.Duration(ms) * time.Millisecond
			}
		default:
			return 0, false, fmt.Errorf("não conheço a opção %q; uso: mkpulso-coletor watch [--intervalo MS] [--uma-vez]", a)
		}
	}
	return intervalo, umaVez, nil
}

func watch(args []string) int {
	intervalo, umaVez, err := watchOpcoes(args)
	if err != nil {
		return base.Falha(base.Erro, "%s", err.Error())
	}
	// sem isto, escrever num stdout fechado mata o processo com SIGPIPE; com
	// isto a escrita devolve EPIPE e o laço sai limpo
	signal.Ignore(syscall.SIGPIPE)
	sinais := make(chan os.Signal, 1)
	signal.Notify(sinais, syscall.SIGTERM, syscall.SIGINT)
	para := make(chan struct{})
	acabou := make(chan struct{})
	go func() {
		select {
		case <-sinais:
			close(para)
		case <-acabou:
		}
	}()
	defer func() {
		signal.Stop(sinais)
		close(acabou)
	}()
	return watchLaco(intervalo, umaVez, para)
}

// watchLaco é o laço: colhe, espera, colhe, solta o retrato. Sempre devolve
// base.Certo: parar o watch não é erro.
func watchLaco(intervalo time.Duration, umaVez bool, para <-chan struct{}) int {
	var historico *grupos.Historico
	if !umaVez {
		historico = grupos.NovoHistorico() // em --uma-vez a média é o valor de agora
	}
	atalhos := grupos.NovosAtalhos()
	antes := coleta.Colher(watchRel.Mono())
	proximo := antes.Quando + intervalo
	for {
		falta := proximo - watchRel.Mono()
		if falta < 0 {
			// a leitura levou mais que o intervalo: não acumula atraso
			falta = 0
			proximo = watchRel.Mono()
		}
		if !watchRel.Espera(falta, para) {
			return base.Certo
		}
		proximo += intervalo
		depois := coleta.Colher(watchRel.Mono())
		if retrato, ok := watchMonta(antes, depois, historico, atalhos); ok {
			if err := base.Escreve(retrato); err != nil {
				// o stdout fechou (EPIPE): quem lia foi embora
				return base.Certo
			}
		}
		if umaVez {
			return base.Certo
		}
		antes = depois
	}
}

// watchMonta faz um retrato de duas amostras. Se alguma leitura quebrar de
// um jeito que ninguém previu, o retrato é pulado (ok false) e o laço segue:
// melhor um retrato a menos que um JSON torto ou o watch morto.
func watchMonta(antes, depois *coleta.Amostra, historico *grupos.Historico, atalhos *grupos.Atalhos) (r watchRetrato, ok bool) {
	defer func() {
		if p := recover(); p != nil {
			fmt.Fprintf(os.Stderr, "mkpulso-coletor: retrato pulado: %v\n", p)
			ok = false
		}
	}()
	nucleos := depois.Nucleos()
	medidos := coleta.MedirProcessos(antes, depois)
	gs := grupos.Agrupar(medidos, janelas.Listar(), nucleos, atalhos)
	if historico != nil {
		historico.Registrar(depois.Quando, gs)
	}
	r = watchRetrato{
		TMs:       watchRel.Parede().UnixMilli(),
		Maquina:   coleta.Maquina(),
		Nucleos:   nucleos,
		CPU:       coleta.MedirCPU(antes, depois),
		Mem:       coleta.LerMemoria(),
		Discos:    coleta.MedirDiscos(antes, depois),
		Montagens: coleta.LerMontagens(),
		Rede:      coleta.MedirRede(antes, depois),
		GPUs:      coleta.LerGPUs(),
		Bateria:   coleta.LerBateria(),
		Grupos:    gs,
	}
	r.CPU.FreqMHz = coleta.Frequencia()
	r.CPU.TempC = coleta.Temperatura()
	r.CPU.Carga = coleta.Carga()
	r.Totais.Processos = len(medidos)
	r.Totais.Grupos = len(gs)
	for _, m := range medidos {
		r.Totais.Threads += m.Threads
	}
	return r, true
}
