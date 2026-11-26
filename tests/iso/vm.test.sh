#!/usr/bin/env bash
# iso/vm: os argumentos do QEMU (disco e UEFI da VM, vídeo, ISO mais novo) e o
# reset. O QEMU nunca roda: qemu-system-x86_64 e qemu-img são stubs, e roda-se
# uma cópia do script numa árvore falsa, com o firmware (OVMF) de mentira.
. "$(dirname "$0")/../lib.sh"

r=$SBX/repo
mkdir -p "$r/iso/out" "$SBX/ovmf"
cp "$REPO/iso/vm" "$r/iso/vm"
: >"$SBX/ovmf/OVMF_CODE.4m.fd"
echo "variáveis de fábrica" >"$SBX/ovmf/OVMF_VARS.4m.fd"
dir=$XDG_CACHE_HOME/mk-vm

# O QEMU daqui sabe (ou não) o virtio-vga-gl: $SBX/sem-gl decide.
stub qemu-system-x86_64 '
if [[ "$*" == "-device help" ]]; then
  echo "name \"virtio-vga\", bus PCI"
  [[ -e $SBX/sem-gl ]] || echo "name \"virtio-vga-gl\", bus PCI"
fi'
stub qemu-img ': >"$5"'

vm() {
  calls_reset
  OVMF=$SBX/ovmf bash "$r/iso/vm" "$@" >"$SBX/saida" 2>&1
  rc=$?
  out=$(<"$SBX/saida")
  qemu=$(grep '^qemu-system-x86_64 -name' "$SBX/calls")
}

t "sem argumento: inicia pelo disco da VM, criando o disco de 40 GB e as variáveis do UEFI"
vm
eq 0 "$rc" "$out"
called "qemu-img create -q -f qcow2 $dir/disk.qcow2 40G"
eq "variáveis de fábrica" "$(<"$dir/vars.fd")"
has "$qemu" "-drive file=$dir/disk.qcow2,if=virtio"
hasnt "$qemu" "-cdrom"

t "a VM do README: 4 processadores, 6 GB, KVM, UEFI, rede por cabo e som pelo PipeWire"
has "$qemu" "-name Mk"
has "$qemu" "-enable-kvm -cpu host -smp 4 -m 6G -machine q35"
has "$qemu" "-drive if=pflash,format=raw,readonly=on,file=$SBX/ovmf/OVMF_CODE.4m.fd"
has "$qemu" "-drive if=pflash,format=raw,file=$dir/vars.fd"
has "$qemu" "model=virtio-net-pci"
has "$qemu" "-audiodev pipewire,id=som"

t "a porta 2222 daqui vai para o SSH da VM, só em 127.0.0.1"
has "$qemu" "hostfwd=tcp:127.0.0.1:2222-:22"

t "o monitor do QEMU fica em ~/.cache/mk-vm/monitor.sock"
has "$qemu" "-monitor unix:$dir/monitor.sock,server,nowait"

t "nenhum disco de verdade nos argumentos"
hasnt "$qemu" "/dev/"
eq 1 "$(grep -o -- '-drive file=' <<<"$qemu" | wc -l)"

t "o firmware do UEFI (OVMF_CODE) é só leitura; as variáveis são uma cópia da VM"
hasnt "$qemu" "file=$SBX/ovmf/OVMF_VARS.4m.fd"
eq "variáveis de fábrica" "$(<"$SBX/ovmf/OVMF_VARS.4m.fd")"

t "com o QEMU sabendo virtio-vga-gl: vídeo acelerado, em janela SDL com OpenGL"
has "$qemu" "-device virtio-vga-gl -display sdl,gl=on"

t "disk: o mesmo que sem argumento"
antes=$qemu
vm disk
eq 0 "$rc" "$out"
eq "$antes" "$qemu"

t "segunda vez: o disco e as variáveis do UEFI que já existem são mantidos"
echo "entrada Mk gravada" >"$dir/vars.fd"
echo "mk instalado" >"$dir/disk.qcow2"
vm
not_called "qemu-img"
eq "entrada Mk gravada" "$(<"$dir/vars.fd")"
eq "mk instalado" "$(<"$dir/disk.qcow2")"

t "MK_VM_GL=0: vídeo sem OpenGL (virtio-vga), sem janela SDL"
calls_reset
MK_VM_GL=0 OVMF=$SBX/ovmf bash "$r/iso/vm" >"$SBX/saida" 2>&1
eq 0 $?
qemu=$(grep '^qemu-system-x86_64 -name' "$SBX/calls")
has "$qemu" "-device virtio-vga"
hasnt "$qemu" "virtio-vga-gl"
hasnt "$qemu" "gl=on"
hasnt "$qemu" "sdl"

t "MK_VM_TELA=nenhuma: sem janela nenhuma (a tela só existe pelas capturas do monitor) e sem som"
calls_reset
MK_VM_TELA=nenhuma OVMF=$SBX/ovmf bash "$r/iso/vm" >"$SBX/saida" 2>&1
eq 0 $?
qemu=$(grep '^qemu-system-x86_64 -name' "$SBX/calls")
has "$qemu" "-display none"
has "$qemu" "-device virtio-vga"
hasnt "$qemu" "virtio-vga-gl"
hasnt "$qemu" "sdl"
hasnt "$qemu" "audiodev"
has "$qemu" "monitor.sock"

t "QEMU sem virtio-vga-gl: cai para virtio-vga sozinho"
: >"$SBX/sem-gl"
vm
eq 0 "$rc" "$out"
has "$qemu" "-device virtio-vga"
hasnt "$qemu" "virtio-vga-gl"
hasnt "$qemu" "gl=on"
rm -f "$SBX/sem-gl"

t "iso: sem nenhum ISO em iso/out, para e manda rodar o iso/build"
vm iso
eq 1 "$rc"
has "$out" "nenhum ISO em iso/out: rode iso/build primeiro"
eq "" "$qemu"

t "iso: inicia pelo ISO modificado por último, com o disco da VM junto"
: >"$r/iso/out/mk-2026.11.21-x86_64.iso"
: >"$r/iso/out/mk-2026.11.20-x86_64.iso"
: >"$r/iso/out/outro.iso"
touch -d '2026-11-21 02:00' "$r/iso/out/mk-2026.11.21-x86_64.iso"
touch -d '2026-11-22 09:00' "$r/iso/out/mk-2026.11.20-x86_64.iso"
touch -d '2026-11-23 09:00' "$r/iso/out/outro.iso"
vm iso
eq 0 "$rc" "$out"
has "$qemu" "-cdrom $r/iso/out/mk-2026.11.20-x86_64.iso -boot order=d"
has "$qemu" "-drive file=$dir/disk.qcow2,if=virtio"
has "$out" "iniciando pelo ISO: $r/iso/out/mk-2026.11.20-x86_64.iso"
eq 1 "$(grep -o -- '-cdrom' <<<"$qemu" | wc -l)"

t "iso: não gera o ISO (não chama o iso/build nem o mkarchiso)"
eq "" "$(grep -v '^qemu-' "$SBX/calls")"

t "reset: apaga o disco e as variáveis do UEFI da VM, e só isso"
: >"$dir/monitor.sock"
vm reset
eq 0 "$rc" "$out"
absent "$dir/disk.qcow2"
absent "$dir/vars.fd"
exists "$dir/monitor.sock"
has "$out" "disco da VM apagado: $dir"
eq "" "$qemu"
exists "$r/iso/out/mk-2026.11.20-x86_64.iso"
exists "$SBX/ovmf/OVMF_VARS.4m.fd"

t "reset sem VM nenhuma: não é erro"
rm -rf "$dir"
vm reset
eq 0 "$rc" "$out"

t "argumento desconhecido: uso e saída 2, sem criar nem iniciar nada"
vm apagar
eq 2 "$rc"
has "$out" "uso: iso/vm [iso | reset]"
eq "" "$(<"$SBX/calls")"
absent "$dir"

t "sem o firmware UEFI (edk2-ovmf), para dizendo o que instalar"
calls_reset
out=$(OVMF=$SBX/nao-existe bash "$r/iso/vm" 2>&1)
eq 1 $?
has "$out" "falta o UEFI da VM: sudo pacman -S --needed edk2-ovmf"
eq "" "$(<"$SBX/calls")"

t "sem o QEMU, para dizendo o que instalar"
# Um PATH só com o mínimo: o qemu desta máquina, se houver, não aparece.
mkdir -p "$SBX/minimo"
for c in bash dirname; do ln -s "$(command -v "$c")" "$SBX/minimo/$c"; done
out=$(PATH=$SBX/minimo OVMF=$SBX/ovmf "$SBX/minimo/bash" "$r/iso/vm" 2>&1)
eq 1 $?
has "$out" "falta o qemu: sudo pacman -S --needed qemu-desktop edk2-ovmf"
absent "$dir"
