package vm

import (
	"path/filepath"
	"strconv"
	"strings"
)

// Ambiente é o que a linha de comando do QEMU precisa saber da máquina de
// fora: os caminhos, se há KVM, a pasta do OVMF e o tipo de tela.
type Ambiente struct {
	Qemu    string // o comando
	KVM     bool   // /dev/kvm abre para escrita
	OVMF    string // a pasta com OVMF_CODE.4m.fd
	Display string // "gtk" ou "none"
	SemISO  bool   // ligar sem o CD
}

// Os arquivos do OVMF dentro da pasta dele.
const (
	OvmfCode = "OVMF_CODE.4m.fd"
	OvmfVars = "OVMF_VARS.4m.fd"
)

// virgula dobra as vírgulas de um caminho que vai dentro de uma opção do
// QEMU (file=…, unix:…): é assim que ele lê uma vírgula que não separa.
func virgula(caminho string) string {
	return strings.ReplaceAll(caminho, ",", ",,")
}

// Argv monta a linha de comando do QEMU para ligar a máquina (o contrato a
// descreve item por item) e os avisos para a pessoa. É uma lista de
// argumentos: nada aqui passa por shell.
func (l Loja) Argv(m Maquina, a Ambiente) (argv []string, avisos []string) {
	avisos = []string{}
	argv = []string{a.Qemu, "-name", m.Nome, "-machine", "q35"}
	if a.KVM {
		argv = append(argv, "-enable-kvm", "-cpu", "host")
	} else {
		argv = append(argv, "-cpu", "max")
		avisos = append(avisos, "sem o KVM a máquina é emulada por inteiro: vai ficar lenta")
	}
	argv = append(argv, "-smp", strconv.Itoa(m.CPUs), "-m", strconv.Itoa(m.MemoriaMB))
	if m.UEFI {
		argv = append(argv,
			"-drive", "if=pflash,format=raw,readonly=on,file="+virgula(filepath.Join(a.OVMF, OvmfCode)),
			"-drive", "if=pflash,format=raw,file="+virgula(l.Vars(m.Nome)))
	}
	argv = append(argv, "-drive", "file="+virgula(l.Disco(m.Nome))+",if=virtio,format=qcow2")
	if m.ISO != "" && !a.SemISO {
		argv = append(argv, "-cdrom", m.ISO, "-boot", "order=cd,menu=on")
	} else {
		argv = append(argv, "-boot", "order=c,menu=on")
	}
	argv = append(argv,
		"-nic", "user,model=virtio-net-pci,hostfwd=tcp:127.0.0.1:"+strconv.Itoa(m.PortaSSH)+"-:22",
		"-device", "qemu-xhci", "-device", "usb-tablet")
	semTela := a.Display == "none"
	if !semTela {
		argv = append(argv, "-audiodev", "pipewire,id=som", "-device", "intel-hda", "-device", "hda-duplex,audiodev=som")
	}
	argv = append(argv, "-vga", "none")
	switch {
	case semTela:
		argv = append(argv, "-device", "virtio-vga", "-display", "none")
	case m.TresD:
		argv = append(argv, "-device", "virtio-vga-gl", "-display", "gtk,gl=on")
	default:
		argv = append(argv, "-device", "virtio-vga", "-display", "gtk")
	}
	argv = append(argv,
		"-qmp", "unix:"+virgula(l.Socket(m.Nome))+",server=on,wait=off",
		"-pidfile", l.Pid(m.Nome),
		"-D", l.Depuracao(m.Nome))
	return argv, avisos
}
