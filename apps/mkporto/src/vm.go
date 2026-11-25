package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"mkporto/internal/base"
	"mkporto/internal/sshcfg"
	"mkporto/internal/vm"
)

func init() { registrar("vm", maquinas) }

// Um subcomando de `vm`.
var subcomandosDeVm = map[string]comando{
	"requisitos": vmRequisitos,
	"listar":     vmListar,
	"criar":      vmCriar,
	"ligar":      vmLigar,
	"desligar":   vmDesligar,
	"pausar":     func(args []string) int { return vmPausa(args, "pausar", "stop", "pausada") },
	"continuar":  func(args []string) int { return vmPausa(args, "continuar", "cont", "ligada") },
	"uso":        vmUso,
	"snapshot":   vmSnapshot,
	"remover":    vmRemover,
	"host":       vmHost,
}

// Os prazos do qemu-img: criar um disco é rápido; um snapshot de um disco
// grande pode demorar.
const (
	prazoDeCriarDisco = 60 * time.Second
	prazoDeSnapshot   = 5 * time.Minute
)

// maquinas: `vm <subcomando> …`, as máquinas virtuais (QEMU direto, por QMP).
func maquinas(args []string) int {
	if len(args) > 0 {
		if sub, ok := subcomandosDeVm[args[0]]; ok {
			return sub(args[1:])
		}
	}
	nomes := make([]string, 0, len(subcomandosDeVm))
	for n := range subcomandosDeVm {
		nomes = append(nomes, n)
	}
	sort.Strings(nomes)
	return base.Falha(base.Erro, "uso: mkporto-motor vm <%s>", strings.Join(nomes, " | "))
}

// aLoja são as pastas das máquinas, pelas variáveis de ambiente.
func aLoja() vm.Loja {
	return vm.Loja{Raiz: base.Maquinas(), Execucao: base.Execucao()}
}

func oProc() string { return base.Comando("MKPORTO_PROC", "/proc") }

// bandeiras separa os argumentos de um subcomando de `vm`: os posicionais, as
// opções que só ligam algo (`semValor`) e as que levam um valor (`comValor`).
// Depois de "--" tudo é posicional. Opção desconhecida ou repetida é erro.
func bandeiras(args, semValor, comValor []string) (pos []string, ops map[string]string, err error) {
	ops = map[string]string{}
	esta := func(lista []string, a string) bool {
		for _, x := range lista {
			if x == a {
				return true
			}
		}
		return false
	}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if _, repetida := ops[a]; repetida {
			return nil, nil, fmt.Errorf("a opção %s veio duas vezes", a)
		}
		switch {
		case a == "--":
			return append(pos, args[i+1:]...), ops, nil
		case esta(semValor, a):
			ops[a] = "sim"
		case esta(comValor, a):
			if i+1 >= len(args) {
				return nil, nil, fmt.Errorf("a opção %s pede um valor", a)
			}
			i++
			ops[a] = args[i]
		case strings.HasPrefix(a, "-"):
			return nil, nil, fmt.Errorf("não conheço a opção %s", a)
		default:
			pos = append(pos, a)
		}
	}
	return pos, ops, nil
}

// umaMaquina lê os argumentos de um subcomando que recebe só o nome e devolve
// a máquina. codigo != 0: o erro já foi escrito.
func umaMaquina(args []string, uso string, semValor ...string) (l vm.Loja, m vm.Maquina, ops map[string]string, codigo int) {
	l = aLoja()
	pos, ops, err := bandeiras(args, semValor, nil)
	if err != nil {
		return l, m, nil, base.Falha(base.Erro, "%s", err)
	}
	if len(pos) != 1 {
		return l, m, nil, base.Falha(base.Erro, "uso: mkporto-motor vm %s", uso)
	}
	m, err = l.Le(pos[0])
	if err != nil {
		return l, m, nil, base.Falha(base.Erro, "%s", err)
	}
	return l, m, ops, base.Certo
}

// situacao é como uma máquina está agora.
type situacao struct {
	Estado string // desligada | ligada | pausada | travada
	Pid    int
	Desde  int64
}

func (s situacao) viva() bool { return s.Estado != "desligada" }

// situacaoDe olha o arquivo de pid, confere no /proc que aquele processo é o
// QEMU daquela máquina e pergunta o estado pelo QMP. Um pid que sobrou de um
// processo morto (ou que hoje é de outro programa) é limpo.
func situacaoDe(l vm.Loja, nome string) situacao {
	desligada := situacao{Estado: "desligada"}
	if l.Execucao == "" {
		return desligada
	}
	pid := vm.LePid(l.Pid(nome))
	if pid == 0 {
		return desligada
	}
	if !vm.EDaMaquina(oProc(), pid, nome, l.Pid(nome)) {
		l.LimpaCorrida(nome)
		return desligada
	}
	s := situacao{Estado: "travada", Pid: pid}
	if info, err := os.Stat(l.Pid(nome)); err == nil {
		s.Desde = info.ModTime().Unix()
	}
	if estado, err := vm.Estado(l.Socket(nome)); err == nil {
		s.Estado = estado
	}
	return s
}

// --- requisitos -------------------------------------------------------------------

type requisito struct {
	Tem     bool   `json:"tem"`
	Caminho string `json:"caminho"`
}

// comandoDe acha um comando externo (o da variável, ou o padrão).
func comandoDe(variavel, padrao string) requisito {
	nome := base.Comando(variavel, padrao)
	if caminho, err := exec.LookPath(nome); err == nil {
		return requisito{Tem: true, Caminho: caminho}
	}
	return requisito{Caminho: nome}
}

func oOvmf() requisito {
	pasta := base.Comando("MKPORTO_OVMF", "/usr/share/edk2/x64")
	r := requisito{Tem: true, Caminho: pasta}
	for _, arquivo := range []string{vm.OvmfCode, vm.OvmfVars} {
		if info, err := os.Stat(filepath.Join(pasta, arquivo)); err != nil || !info.Mode().IsRegular() {
			r.Tem = false
		}
	}
	return r
}

// oKvm diz se o /dev/kvm abre para leitura e escrita (é o que o QEMU precisa).
func oKvm() requisito {
	caminho := base.Comando("MKPORTO_KVM", "/dev/kvm")
	return requisito{Tem: vm.AbreParaEscrita(caminho), Caminho: caminho}
}

// vmRequisitos: o que há e o que falta para ter máquinas, e o que abrir num
// terminal para instalar. O motor não instala nada.
func vmRequisitos(args []string) int {
	if pos, _, err := bandeiras(args, nil, nil); err != nil || len(pos) != 0 {
		return base.Falha(base.Erro, "uso: mkporto-motor vm requisitos")
	}
	qemu, img := comandoDe("MKPORTO_QEMU", "qemu-system-x86_64"), comandoDe("MKPORTO_QEMU_IMG", "qemu-img")
	ovmf, kvm := oOvmf(), oKvm()
	avisos := []string{}
	if !qemu.Tem || !img.Tem {
		avisos = append(avisos, "falta o QEMU (pacote qemu-desktop)")
	}
	if !ovmf.Tem {
		avisos = append(avisos, "sem o edk2-ovmf não dá para criar máquina com UEFI")
	}
	if !kvm.Tem {
		avisos = append(avisos, "sem acesso ao KVM ("+kvm.Caminho+") as máquinas ligam, mas ficam lentas")
	}
	if base.Execucao() == "" {
		avisos = append(avisos, "sem o XDG_RUNTIME_DIR não há onde pôr o socket de uma máquina ligada")
	}
	// Quem instala é a pessoa, num terminal: é lá que a senha é digitada.
	argv := []string{"sudo", "pacman", "-S", "--needed", "qemu-desktop", "edk2-ovmf"}
	instalar := map[string]any{"argv": argv, "terminal": nil, "aviso": nil}
	if t := terminal(); t != "" {
		instalar["terminal"] = linhaDoTerminal(t, "instalar o QEMU", argv, true)
	} else {
		instalar["aviso"] = "não achei um terminal para abrir (ghostty, kitty, foot, alacritty ou wezterm)"
	}
	_ = base.Escreve(map[string]any{"qemu": qemu, "qemu_img": img, "ovmf": ovmf, "kvm": kvm,
		"pronto": qemu.Tem && img.Tem, "avisos": avisos, "instalar": instalar})
	return base.Certo
}

// --- listar -----------------------------------------------------------------------

type maquinaJSON struct {
	Nome       string  `json:"nome"`
	MemoriaMB  int     `json:"memoria_mb"`
	CPUs       int     `json:"cpus"`
	DiscoGB    int     `json:"disco_gb"`
	ISO        *string `json:"iso"`
	UEFI       bool    `json:"uefi"`
	TresD      bool    `json:"tres_d"`
	PortaSSH   int     `json:"porta_ssh"`
	CriadaEm   int64   `json:"criada_em"`
	Estado     string  `json:"estado"`
	Pid        *int    `json:"pid"`
	Desde      *int64  `json:"desde"`
	DiscoBytes *int64  `json:"disco_bytes"`
}

// vmListar: as máquinas, com a configuração e o estado de cada uma.
func vmListar(args []string) int {
	if pos, _, err := bandeiras(args, nil, nil); err != nil || len(pos) != 0 {
		return base.Falha(base.Erro, "uso: mkporto-motor vm listar")
	}
	l := aLoja()
	lista := []maquinaJSON{}
	for _, m := range l.Todas() {
		j := maquinaJSON{Nome: m.Nome, MemoriaMB: m.MemoriaMB, CPUs: m.CPUs, DiscoGB: m.DiscoGB, ISO: ouNada(m.ISO),
			UEFI: m.UEFI, TresD: m.TresD, PortaSSH: m.PortaSSH, CriadaEm: m.CriadaEm}
		s := situacaoDe(l, m.Nome)
		j.Estado = s.Estado
		if s.viva() {
			j.Pid, j.Desde = &s.Pid, &s.Desde
		}
		if n := vm.NoDisco(l.Disco(m.Nome)); n >= 0 {
			j.DiscoBytes = &n
		}
		lista = append(lista, j)
	}
	_ = base.Escreve(map[string]any{"pasta": base.Til(l.Raiz), "maquinas": lista})
	return base.Certo
}

// --- criar ------------------------------------------------------------------------

// pedidoDeMaquina é o JSON que a janela manda para o `vm criar`.
type pedidoDeMaquina struct {
	Nome      string `json:"nome"`
	MemoriaMB int    `json:"memoria_mb"`
	CPUs      int    `json:"cpus"`
	DiscoGB   int    `json:"disco_gb"`
	ISO       string `json:"iso"`
	UEFI      bool   `json:"uefi"`
	TresD     bool   `json:"tres_d"`
}

// portaLivre escolhe a porta local do SSH de uma máquina nova: a primeira, a
// partir de 2222, que não é de outra máquina e em que dá para escutar agora.
func portaLivre(l vm.Loja) (int, bool) {
	usadas := map[int]bool{}
	for _, m := range l.Todas() {
		usadas[m.PortaSSH] = true
	}
	for porta := vm.PrimeiraPorta; porta < vm.PrimeiraPorta+3000; porta++ {
		if usadas[porta] {
			continue
		}
		escuta, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(porta))
		if err != nil {
			continue
		}
		escuta.Close()
		return porta, true
	}
	return 0, false
}

// vmCriar: cria uma máquina (a pasta, o disco, o vars do UEFI e o
// maquina.json). Se algo falha no meio, a pasta criada é apagada.
func vmCriar(args []string) int {
	if pos, _, err := bandeiras(args, nil, nil); err != nil || len(pos) != 0 {
		return base.Falha(base.Erro, "uso: mkporto-motor vm criar (com o pedido em JSON na entrada padrão)")
	}
	var p pedidoDeMaquina
	dec := json.NewDecoder(io.LimitReader(entrada, pedidoMaximo))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&p); err != nil {
		return base.Falha(base.Erro, "o pedido não é o JSON que o vm criar espera")
	}
	if _, err := dec.Token(); err != io.EOF {
		return base.Falha(base.Erro, "o pedido não é o JSON que o vm criar espera")
	}
	m := vm.Maquina{Nome: p.Nome, MemoriaMB: p.MemoriaMB, CPUs: p.CPUs, DiscoGB: p.DiscoGB, ISO: p.ISO, UEFI: p.UEFI, TresD: p.TresD}
	if err := vm.Valida(m); err != nil {
		return base.FalhaDeCampo(err.Campo, err.Frase)
	}
	l := aLoja()
	if l.Existe(m.Nome) {
		return base.FalhaDeCampo("nome", "já existe uma máquina com esse nome")
	}
	if m.ISO != "" {
		if info, err := os.Stat(m.ISO); err != nil {
			return base.FalhaDeCampo("iso", "não achei essa ISO")
		} else if !info.Mode().IsRegular() {
			return base.FalhaDeCampo("iso", "a ISO tem de ser um arquivo")
		}
	}
	img := comandoDe("MKPORTO_QEMU_IMG", "qemu-img")
	if !img.Tem {
		return base.Falha(base.Erro, "falta o qemu-img: instale o QEMU (pacote qemu-desktop)")
	}
	ovmf := oOvmf()
	if m.UEFI && !ovmf.Tem {
		return base.FalhaDeCampo("uefi", "UEFI precisa do OVMF: instale o pacote edk2-ovmf")
	}

	if err := l.CriaPasta(m.Nome); err != nil {
		return base.Falha(base.Erro, "%s", err)
	}
	desfaz := func(formato string, a ...any) int {
		_ = l.Apaga(m.Nome)
		return base.Falha(base.Erro, formato, a...)
	}
	ctx, cancela := context.WithTimeout(context.Background(), prazoDeCriarDisco)
	defer cancela()
	cmd := exec.CommandContext(ctx, img.Caminho, "create", "-f", "qcow2", l.Disco(m.Nome), strconv.Itoa(m.DiscoGB)+"G")
	cmd.WaitDelay = time.Second
	if saida, err := cmd.CombinedOutput(); err != nil {
		return desfaz("o qemu-img não criou o disco: %s", resumo(string(saida), err))
	}
	if m.UEFI {
		if err := l.CopiaVars(m.Nome, filepath.Join(ovmf.Caminho, vm.OvmfVars)); err != nil {
			return desfaz("%s", err)
		}
	}
	porta, ok := portaLivre(l)
	if !ok {
		return desfaz("não achei uma porta local livre para o SSH da máquina")
	}
	m.PortaSSH, m.CriadaEm = porta, time.Now().Unix()
	if err := l.Grava(m); err != nil {
		return desfaz("%s", err)
	}
	_ = base.Escreve(map[string]any{"ok": true, "nome": m.Nome, "porta_ssh": m.PortaSSH, "pasta": base.Til(l.Pasta(m.Nome))})
	return base.Certo
}

// resumo é a última linha do que um comando reclamou, ou o erro dele.
func resumo(saida string, err error) string {
	if ultima := ultimaLinha(saida, 200); ultima != "" {
		return ultima
	}
	return err.Error()
}

// --- snapshots --------------------------------------------------------------------

const usoDoSnapshot = "snapshot <listar | criar | restaurar | apagar> [--] <nome> [rótulo]"

// vmSnapshot: os snapshots do disco, pelo qemu-img. Só com a máquina
// desligada: mexer no qcow2 de uma máquina ligada estraga o disco.
func vmSnapshot(args []string) int {
	pos, _, err := bandeiras(args, nil, nil)
	if err != nil {
		return base.Falha(base.Erro, "%s", err)
	}
	opcao := map[string]string{"listar": "-l", "criar": "-c", "restaurar": "-a", "apagar": "-d"}
	if len(pos) < 2 || opcao[pos[0]] == "" || (pos[0] == "listar") != (len(pos) == 2) || len(pos) > 3 {
		return base.Falha(base.Erro, "uso: mkporto-motor vm %s", usoDoSnapshot)
	}
	acao, l := pos[0], aLoja()
	m, err := l.Le(pos[1])
	if err != nil {
		return base.Falha(base.Erro, "%s", err)
	}
	rotulo := ""
	if acao != "listar" {
		if rotulo = pos[2]; !vm.RotuloBom(rotulo) {
			return base.FalhaDeCampo("rotulo", "o rótulo só pode ter letras sem acento, algarismos, ponto, hífen e sublinhado (até 40), sem começar com hífen")
		}
	}
	if situacaoDe(l, m.Nome).viva() {
		return base.Falha(base.Recusado, "snapshots só com a máquina desligada: desligue a %s antes", m.Nome)
	}
	img := comandoDe("MKPORTO_QEMU_IMG", "qemu-img")
	if !img.Tem {
		return base.Falha(base.Erro, "falta o qemu-img: instale o QEMU (pacote qemu-desktop)")
	}
	argv := []string{"snapshot", opcao[acao]}
	if rotulo != "" {
		argv = append(argv, rotulo)
	}
	ctx, cancela := context.WithTimeout(context.Background(), prazoDeSnapshot)
	defer cancela()
	cmd := exec.CommandContext(ctx, img.Caminho, append(argv, l.Disco(m.Nome))...)
	cmd.WaitDelay = time.Second
	saida, err := cmd.CombinedOutput()
	if err != nil {
		return base.Falha(base.Erro, "o qemu-img não fez o snapshot: %s", resumo(string(saida), err))
	}
	if acao == "listar" {
		_ = base.Escreve(map[string]any{"nome": m.Nome, "snapshots": vm.LeSnapshots(string(saida))})
	} else {
		_ = base.Escreve(map[string]any{"ok": true, "nome": m.Nome, "rotulo": rotulo})
	}
	return base.Certo
}

// --- remover e host ---------------------------------------------------------------

// vmRemover: apaga uma máquina desligada, com o nome repetido em --confirmo.
func vmRemover(args []string) int {
	l := aLoja()
	pos, ops, err := bandeiras(args, nil, []string{"--confirmo"})
	if err != nil {
		return base.Falha(base.Erro, "%s", err)
	}
	if len(pos) != 1 {
		return base.Falha(base.Erro, "uso: mkporto-motor vm remover --confirmo <nome> [--] <nome>")
	}
	m, err := l.Le(pos[0])
	if err != nil {
		return base.Falha(base.Erro, "%s", err)
	}
	if ops["--confirmo"] != m.Nome {
		return base.Falha(base.Recusado, "para remover a %s, repita o nome dela em --confirmo: o disco vai junto e não volta", m.Nome)
	}
	if situacaoDe(l, m.Nome).viva() {
		return base.Falha(base.Recusado, "a %s está ligada: desligue antes de remover", m.Nome)
	}
	if err := l.Apaga(m.Nome); err != nil {
		return base.Falha(base.Erro, "%s", err)
	}
	_ = base.Escreve(map[string]any{"ok": true, "nome": m.Nome})
	return base.Certo
}

// vmHost: põe a máquina na área Hosts (127.0.0.1 na porta dela), pelo mesmo
// caminho do `guardar`. O usuário e a chave de um host que já existia ficam.
func vmHost(args []string) int {
	_, m, _, codigo := umaMaquina(args, "host [--] <nome>")
	if codigo != base.Certo {
		return codigo
	}
	porta := m.PortaSSH
	p := &pedidoDeGuardar{Apelido: m.Nome, Host: "127.0.0.1", Porta: &porta}
	for _, e := range sshcfg.DoApp(sshcfg.ArquivoDoApp(base.PastaDoSsh())) {
		if e.Apelido == m.Nome {
			p.Usuario, p.Chave = e.Usuario, e.Chave
		}
	}
	return gravaHost(p)
}
