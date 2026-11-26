package lugares

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"mkfiles/internal/base"
)

// As colunas pedidas ao lsblk (o contrato), com o tamanho em bytes.
var colunasDoLsblk = []string{"--json", "--bytes", "-o", "NAME,PATH,LABEL,FSTYPE,SIZE,MOUNTPOINTS,RM,HOTPLUG,TYPE,UUID"}

var dispositivoValido = regexp.MustCompile(`^/dev/[A-Za-z0-9/_.-]+$`)

// Os limites de tempo dos comandos de fora (os testes encurtam).
var (
	prazoDoLsblk  = 5 * time.Second
	prazoDoUdisks = 60 * time.Second
)

// bloco é um dispositivo como o lsblk o descreve. RM, HOTPLUG e SIZE mudam de
// tipo entre versões (true ou "1"; número ou texto): ficam crus.
type bloco struct {
	Path        string          `json:"path"`
	Label       *string         `json:"label"`
	Fstype      *string         `json:"fstype"`
	Size        json.RawMessage `json:"size"`
	Mountpoints []*string       `json:"mountpoints"`
	Rm          json.RawMessage `json:"rm"`
	Hotplug     json.RawMessage `json:"hotplug"`
	Type        string          `json:"type"`
	UUID        *string         `json:"uuid"`
	Children    []bloco         `json:"children"`
}

// disco é um volume que a barra lateral mostra.
type disco struct {
	campos base.Campos
	pai    string // o disco inteiro de que este faz parte ("" se é ele mesmo)
}

func ligado(cru json.RawMessage) bool {
	t := strings.Trim(string(bytes.TrimSpace(cru)), `"`)
	return t == "true" || t == "1"
}

func numero(cru json.RawMessage) int64 {
	n, _ := strconv.ParseInt(strings.Trim(string(bytes.TrimSpace(cru)), `"`), 10, 64)
	return n
}

// doSistema: um ponto de montagem que é do sistema, não um disco da pessoa.
func doSistema(ponto string) bool {
	if ponto == "/" || ponto == "/home" || ponto == "[SWAP]" {
		return true
	}
	for _, p := range []string{"/boot", "/efi", "/var", "/usr"} {
		if base.Dentro(p, ponto) {
			return true
		}
	}
	return false
}

// tamanhoLegivel escreve bytes como "16 GB" (potências de 1000, como na caixa
// do disco).
func tamanhoLegivel(b int64) string {
	unidades := []string{"B", "kB", "MB", "GB", "TB", "PB"}
	v, u := float64(b), 0
	for v >= 1000 && u < len(unidades)-1 {
		v /= 1000
		u++
	}
	texto := strconv.FormatFloat(v, 'f', 0, 64)
	if v < 10 && u > 0 {
		texto = strings.Replace(strings.TrimSuffix(strconv.FormatFloat(v, 'f', 1, 64), ".0"), ".", ",", 1)
	}
	return texto + " " + unidades[u]
}

func texto(p *string) any {
	if p == nil || *p == "" {
		return nil
	}
	return *p
}

// lerDiscos roda o lsblk e monta a lista do contrato.
func lerDiscos() ([]disco, error) {
	ctx, cancela := context.WithTimeout(context.Background(), prazoDoLsblk)
	defer cancela()
	cmd := exec.CommandContext(ctx, base.Comando("MKFILES_LSBLK", "lsblk"), colunasDoLsblk...)
	cmd.WaitDelay = time.Second
	saida, err := cmd.Output()
	if err != nil {
		return nil, errors.New("o lsblk não respondeu: " + err.Error())
	}
	var arvore struct {
		Blockdevices []bloco `json:"blockdevices"`
	}
	if err := json.Unmarshal(saida, &arvore); err != nil {
		return nil, errors.New("não entendi o que o lsblk disse")
	}
	var lista []disco
	var desce func(b bloco, pai string, removivel bool)
	desce = func(b bloco, pai string, removivel bool) {
		removivel = removivel || ligado(b.Rm) || ligado(b.Hotplug)
		raiz := pai
		if raiz == "" {
			raiz = b.Path
		}
		for _, f := range b.Children {
			desce(f, raiz, removivel)
		}
		sistema := ""
		if b.Fstype != nil {
			sistema = *b.Fstype
		}
		switch sistema {
		case "", "swap", "crypto_LUKS", "LVM2_member", "linux_raid_member", "squashfs":
			return
		}
		if b.Type == "loop" || !dispositivoValido.MatchString(b.Path) {
			return
		}
		var montado any
		for _, m := range b.Mountpoints {
			if m == nil || *m == "" {
				continue
			}
			if doSistema(*m) {
				return
			}
			if montado == nil {
				montado = *m
			}
		}
		tamanho := numero(b.Size)
		c := base.Campos{"dispositivo": b.Path, "rotulo": texto(b.Label), "sistema": sistema, "tamanho": tamanho,
			"montado": montado, "livre": nil, "total": nil, "removivel": removivel, "uuid": texto(b.UUID)}
		if b.Label != nil && *b.Label != "" {
			c["nome"] = *b.Label
		} else {
			c["nome"] = "Volume de " + tamanhoLegivel(tamanho)
		}
		if ponto, ok := montado.(string); ok {
			if livre, total, ok := espaco(ponto); ok {
				c["livre"], c["total"] = livre, total
			}
		}
		lista = append(lista, disco{campos: c, pai: pai})
	}
	for _, b := range arvore.Blockdevices {
		desce(b, "", false)
	}
	return lista, nil
}

func camposDe(ds []disco) []base.Campos {
	lista := []base.Campos{}
	for _, d := range ds {
		lista = append(lista, d.campos)
	}
	return lista
}

func (a *area) pedirDiscos(p *base.Pedido) {
	ds, err := lerDiscos()
	if err != nil {
		p.Falha(err.Error())
		return
	}
	p.Responde(base.Campos{"discos": camposDe(ds)})
}

// udisksctl roda um comando do udisksctl e devolve o que ele disse.
func udisksctl(args ...string) (string, error) {
	ctx, cancela := context.WithTimeout(context.Background(), prazoDoUdisks)
	defer cancela()
	cmd := exec.CommandContext(ctx, base.Comando("MKFILES_UDISKSCTL", "udisksctl"), args...)
	cmd.WaitDelay = time.Second
	saida, err := cmd.CombinedOutput()
	dito := strings.TrimSpace(string(saida))
	if err != nil {
		if ctx.Err() != nil {
			return dito, errors.New("o udisksctl demorou demais")
		}
		if dito == "" {
			dito = "o udisksctl falhou: " + err.Error()
		}
		return dito, errors.New(dito)
	}
	return dito, nil
}

// udisks atende montar, desmontar e ejetar: roda por fora (o udisksctl pode
// esperar a pessoa autorizar) e responde a lista nova.
func (a *area) udisks(p *base.Pedido, acao string) {
	var q struct {
		Dispositivo string `json:"dispositivo"`
	}
	_ = p.Ler(&q)
	if !dispositivoValido.MatchString(q.Dispositivo) || strings.Contains(q.Dispositivo, "..") {
		p.FalhaDeCampo("dispositivo", "dispositivo inválido")
		return
	}
	a.s.Vai(func() {
		a.montando.Lock()
		defer a.montando.Unlock()
		ds, err := lerDiscos()
		if err != nil {
			p.Falha(err.Error())
			return
		}
		var alvo *disco
		for i := range ds {
			if ds[i].campos["dispositivo"] == q.Dispositivo {
				alvo = &ds[i]
			}
		}
		if alvo == nil {
			p.FalhaDeCampo("dispositivo", "esse disco não está mais aqui")
			return
		}
		montado := alvo.campos["montado"] != nil
		switch acao {
		case "montar":
			if !montado {
				_, err = udisksctl("mount", "-b", q.Dispositivo)
			}
		case "desmontar":
			if montado {
				_, err = udisksctl("unmount", "-b", q.Dispositivo)
			}
		case "ejetar":
			inteiro := alvo.pai
			if inteiro == "" {
				inteiro = q.Dispositivo
			}
			// Tudo que é desse disco e está montado sai antes de desligar.
			for _, d := range ds {
				daqui := d.campos["dispositivo"] == q.Dispositivo || (d.pai != "" && d.pai == alvo.pai)
				if daqui && d.campos["montado"] != nil && err == nil {
					_, err = udisksctl("unmount", "-b", d.campos["dispositivo"].(string))
				}
			}
			if err == nil {
				_, err = udisksctl("power-off", "-b", inteiro)
			}
		}
		if err != nil {
			p.Falha(err.Error())
			return
		}
		resposta := base.Campos{"discos": []base.Campos{}}
		if ds, err := lerDiscos(); err == nil {
			resposta["discos"] = camposDe(ds)
			if acao == "montar" {
				resposta["montado"] = nil
				for _, d := range ds {
					if d.campos["dispositivo"] == q.Dispositivo {
						resposta["montado"] = d.campos["montado"]
					}
				}
			}
		}
		p.Responde(resposta)
	})
}

// intervalo é de quanto em quanto tempo os discos são reconsultados
// (MKFILES_DISCOS_MS).
func intervalo() time.Duration {
	if n, err := strconv.Atoi(os.Getenv("MKFILES_DISCOS_MS")); err == nil && n >= 10 && n <= 600000 {
		return time.Duration(n) * time.Millisecond
	}
	return 2 * time.Second
}

// retrato é a lista de discos e um resumo dela para comparar. O espaço livre
// fica fora do resumo: muda o tempo todo e não é um disco que entrou ou saiu.
func retrato() (string, []base.Campos) {
	ds, err := lerDiscos()
	if err != nil {
		return "", nil
	}
	campos := camposDe(ds)
	var resumo []base.Campos
	for _, c := range campos {
		r := base.Campos{}
		for k, v := range c {
			if k != "livre" {
				r[k] = v
			}
		}
		resumo = append(resumo, r)
	}
	b, _ := json.Marshal(resumo)
	return string(b), campos
}

func (a *area) pararVigia() {
	a.trava.Lock()
	if a.parar != nil {
		close(a.parar)
		a.parar = nil
	}
	a.trava.Unlock()
}

func (a *area) vigiarDiscos(p *base.Pedido) {
	var q struct {
		Ligado bool `json:"ligado"`
	}
	_ = p.Ler(&q)
	a.pararVigia()
	if q.Ligado {
		parar := make(chan struct{})
		a.trava.Lock()
		a.parar = parar
		a.trava.Unlock()
		// Fora do Vai: a vigia não tem fim, e a chamada de um subcomando só
		// não pode ficar esperando por ela.
		go func() {
			montagens, _ := os.ReadFile(base.Raiz("MKFILES_MOUNTINFO", "/proc/self/mountinfo"))
			antes, _ := retrato()
			relogio := time.NewTicker(intervalo())
			defer relogio.Stop()
			for {
				select {
				case <-parar:
					return
				case <-relogio.C:
				}
				agora, campos := retrato()
				novas, _ := os.ReadFile(base.Raiz("MKFILES_MOUNTINFO", "/proc/self/mountinfo"))
				if campos == nil || (agora == antes && bytes.Equal(novas, montagens)) {
					continue
				}
				antes, montagens = agora, novas
				select {
				case <-parar:
					return
				default:
					a.s.Evento("discos", base.Campos{"discos": campos})
				}
			}
		}()
	}
	p.Responde(base.Campos{"ligado": q.Ligado})
}
