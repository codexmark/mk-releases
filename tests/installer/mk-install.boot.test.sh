#!/usr/bin/env bash
# installer/mk-install: o boot. O initramfs, a linha do kernel, o
# /etc/default/limine, o limine.conf e a entrada "Mk" na NVRAM.
# shellcheck disable=SC2154,SC2034 # as variáveis vêm do instalador (carrega.sh)
. "$(dirname "$0")/../lib.sh"
. "$(dirname "$0")/carrega.sh"

raiz_falsa
boot=$target/boot
kernel() { : >"$boot/vmlinuz-linux"; : >"$boot/initramfs-linux.img"; }
# shellcheck disable=SC2034 # lida pelo limine_entry
boot_cmdline="root=UUID=1111-aaaa rootflags=subvol=@ rw quiet splash"

# --- limine_header / limine_entry -------------------------------------------
t "limine_header: 3 segundos de espera, o nome Mk e as cores do Mk"
out=$(limine_header)
has "$out" "timeout: 3"
has "$out" "interface_branding: Mk"
has "$out" "term_background: 0b0b0b"
has "$out" "term_foreground: cdc4ba"
has "$out" "term_foreground_bright: ce5e32"
hasnt "$out" "protocol:"

t "limine_entry: a entrada Mk, protocol linux, com a linha do kernel"
rm -f "$boot"/initramfs-linux-fallback.img
out=$(limine_entry)
has "$out" "/Mk"
has "$out" "    protocol: linux"
has "$out" "    path: boot():/vmlinuz-linux"
has "$out" "    cmdline: root=UUID=1111-aaaa rootflags=subvol=@ rw quiet splash"
has "$out" "    module_path: boot():/initramfs-linux.img"

t "limine_entry: sem o initramfs de fallback, não há Mk (fallback)"
hasnt "$out" "fallback"
eq 1 "$(grep -c '^/' <<<"$out")"

t "limine_entry: com o initramfs de fallback, entra a Mk (fallback)"
: >"$boot/initramfs-linux-fallback.img"
out=$(limine_entry)
has "$out" "/Mk (fallback)"
has "$out" "module_path: boot():/initramfs-linux-fallback.img"
eq 2 "$(grep -c '^/' <<<"$out")"
eq 2 "$(grep -c 'protocol: linux' <<<"$out")"
rm -f "$boot/initramfs-linux-fallback.img"

# --- active_limine_conf ------------------------------------------------------
t "active_limine_conf: o da raiz da ESP quando não há outro"
eq "$boot/limine.conf" "$(active_limine_conf)"

t "active_limine_conf: o da pasta do EFI tem a preferência"
mkdir -p "$boot/EFI/limine"
: >"$boot/EFI/limine/limine.conf"
: >"$boot/limine.conf"
eq "$boot/EFI/limine/limine.conf" "$(active_limine_conf)"
rm -f "$boot/EFI/limine/limine.conf" "$boot/limine.conf"

# --- limine_conf_ok ----------------------------------------------------------
conf=$SBX/limine.conf
kernel

t "limine_conf_ok: arquivo que não existe não serve"
fails limine_conf_ok "$SBX/nao-existe.conf"

t "limine_conf_ok: só o cabeçalho (nenhuma entrada) não serve"
limine_header >"$conf"
fails limine_conf_ok "$conf"

t "limine_conf_ok: as entradas fixas do Mk, com o kernel em /boot, servem"
{ limine_header; limine_entry; } >"$conf"
ok limine_conf_ok "$conf"

t "limine_conf_ok: entrada que aponta para um initramfs que não existe não serve"
rm -f "$boot/initramfs-linux.img"
fails limine_conf_ok "$conf"
kernel

t "limine_conf_ok: entrada que aponta para um kernel que não existe não serve"
rm -f "$boot/vmlinuz-linux"
fails limine_conf_ok "$conf"
kernel

# O que o limine-entry-tool (limine-mkinitcpio-hook) escreve: grupo, kernel
# numa pasta por máquina, "#resumo" no fim de cada caminho e a EFI fallback.
gancho_conf() {
  limine_header
  cat <<'CONF'

/+Mk
    comment: Mk
    comment: machine-id=0123456789abcdef0123456789abcdef order-priority=50
    //linux
    comment: 6.17.8-arch1-1
    protocol: linux
    path: boot():/0123456789abcdef0123456789abcdef/linux/vmlinuz-linux#3b1f9e0c7a5d42e6b8a1c0d9e8f7a6b5c4d3e2f1a0b9c8d7e6f5a4b3c2d1e0f9
    cmdline: root=UUID=1111-aaaa rootflags=subvol=@ rw quiet splash
    module_path: boot():/0123456789abcdef0123456789abcdef/linux/initramfs-linux.img#aa11bb22cc33dd44ee55ff6600112233445566778899aabbccddeeff00112233

/EFI fallback
    comment: Default EFI loader
    protocol: efi
    path: boot():/EFI/BOOT/BOOTX64.EFI
CONF
}
maquina=$boot/0123456789abcdef0123456789abcdef/linux
mkdir -p "$maquina" "$boot/EFI/BOOT"
: >"$maquina/vmlinuz-linux"
: >"$maquina/initramfs-linux.img"
: >"$boot/EFI/BOOT/BOOTX64.EFI"

t "limine_conf_ok: o limine.conf do gancho serve; o #resumo no fim do caminho é tirado"
gancho_conf >"$conf"
ok limine_conf_ok "$conf"

t "limine_conf_ok: no do gancho, kernel que sumiu de /boot não serve"
rm -f "$maquina/vmlinuz-linux"
fails limine_conf_ok "$conf"
: >"$maquina/vmlinuz-linux"

t "limine_conf_ok: só a EFI fallback (nenhuma entrada protocol: linux) não serve"
{
  limine_header
  printf '\n/EFI fallback\n    protocol: efi\n    path: boot():/EFI/BOOT/BOOTX64.EFI\n'
} >"$conf"
fails limine_conf_ok "$conf"

t "limine_conf_ok: kernel_path e module_path com e sem barra também são conferidos"
{
  limine_header
  printf '\n/Mk\n    protocol: linux\n    kernel_path: boot():vmlinuz-linux\n    module_path: boot():/initramfs-linux.img\n'
} >"$conf"
ok limine_conf_ok "$conf"
sed -i 's|boot():vmlinuz-linux|boot():vmlinuz-sumiu|' "$conf"
fails limine_conf_ok "$conf"

# --- configure_boot ----------------------------------------------------------
stub blkid 'case "$*" in
  "-s UUID -o value /dev/sdb2") echo 2222-btrfs-ou-luks ;;
  "-s UUID -o value /dev/mapper/mkroot") echo 3333-btrfs-dentro ;;
  *) echo "blkid falso: $*" >&2; exit 2 ;;
esac'
# shellcheck disable=SC2034
prepara() { respostas_boas; rootpart=/dev/sdb2 rootdev=/dev/sdb2; }
cfgboot() { # cfgboot [CHAVE=valor...]: roda o configure_boot e mostra a linha do kernel
  local kv
  prepara
  for kv in "$@"; do printf -v "${kv%%=*}" '%s' "${kv#*=}"; done
  configure_boot
  echo "CMDLINE=$boot_cmdline"
}

t "configure_boot: os hooks do initramfs, na ordem do README"
calls_reset
roda cfgboot
eq 0 "$rc" "$out"
eq "HOOKS=(base systemd plymouth autodetect microcode modconf kms keyboard sd-vconsole block filesystems fsck)" \
  "$(<"$target/etc/mkinitcpio.conf.d/mk.conf")"

t "configure_boot: a linha do kernel aponta o btrfs pelo UUID, subvolume @"
has "$out" "CMDLINE=root=UUID=2222-btrfs-ou-luks rootflags=subvol=@ rw quiet splash"
hasnt "$out" "rd.luks"

t "configure_boot: /etc/default/limine com o nome Mk, a ESP em /boot e a linha do kernel"
lim=$(<"$target/etc/default/limine")
has "$lim" 'TARGET_OS_NAME="Mk"'
has "$lim" 'ESP_PATH="/boot"'
has "$lim" 'KERNEL_CMDLINE[default]="root=UUID=2222-btrfs-ou-luks rootflags=subvol=@ rw quiet splash"'

t "configure_boot: snapshots no menu em auto e sem procurar outros sistemas"
has "$lim" "MAX_SNAPSHOT_ENTRIES=auto"
has "$lim" "FIND_BOOTLOADERS=no"
eq 7 "$(wc -l <<<"$lim")"

t "configure_boot: o /etc/default/limine é shell válido"
ok bash -n "$target/etc/default/limine"

t "configure_boot: o limine vai para EFI/limine e para o caminho padrão EFI/BOOT"
exists "$boot/EFI/limine/BOOTX64.EFI"
exists "$boot/EFI/BOOT/BOOTX64.EFI"

t "configure_boot: o limine.conf nasce só com o cabeçalho"
eq "$(limine_header)" "$(<"$boot/limine.conf")"

t "configure_boot: não roda mkinitcpio, limine nem efibootmgr (isso é do finish_boot)"
hasnt "$(<"$SBX/calls")" "mkinitcpio"
hasnt "$(<"$SBX/calls")" "efibootmgr"
hasnt "$(<"$SBX/calls")" "limine"

t "configure_boot com LUKS: sd-encrypt antes de filesystems"
# shellcheck disable=SC2034
roda cfgboot ENCRYPT=sim rootdev=/dev/mapper/mkroot
eq 0 "$rc" "$out"
eq "HOOKS=(base systemd plymouth autodetect microcode modconf kms keyboard sd-vconsole block sd-encrypt filesystems fsck)" \
  "$(<"$target/etc/mkinitcpio.conf.d/mk.conf")"

t "configure_boot com LUKS: rd.luks.name com o UUID da partição (não o do btrfs) e root no mapper"
has "$out" "CMDLINE=rd.luks.name=2222-btrfs-ou-luks=mkroot root=/dev/mapper/mkroot rootflags=subvol=@ rw quiet splash"
has "$(<"$target/etc/default/limine")" 'KERNEL_CMDLINE[default]="rd.luks.name=2222-btrfs-ou-luks=mkroot root=/dev/mapper/mkroot rootflags=subvol=@ rw quiet splash"'
hasnt "$out" "3333-btrfs-dentro"

t "configure_boot com NVIDIA: sem kms no initramfs, o resto igual"
roda cfgboot "HARDWARE=intel nvidia"
eq 0 "$rc" "$out"
eq "HOOKS=(base systemd plymouth autodetect microcode modconf keyboard sd-vconsole block filesystems fsck)" \
  "$(<"$target/etc/mkinitcpio.conf.d/mk.conf")"

t "configure_boot com NVIDIA e LUKS: sem kms e com sd-encrypt"
roda cfgboot HARDWARE=nvidia ENCRYPT=sim rootdev=/dev/mapper/mkroot
eq "HOOKS=(base systemd plymouth autodetect microcode modconf keyboard sd-vconsole block sd-encrypt filesystems fsck)" \
  "$(<"$target/etc/mkinitcpio.conf.d/mk.conf")"

t "configure_boot: um perfil que só parece nvidia não tira o kms"
roda cfgboot "HARDWARE=amd vm"
has "$(<"$target/etc/mkinitcpio.conf.d/mk.conf")" " kms "

# --- finish_boot -------------------------------------------------------------
# O que o efibootmgr mostra numa máquina que já teve Windows, um Mk antigo e
# o limine-install do gancho.
efi_lista='BootCurrent: 0004
Timeout: 1 seconds
BootOrder: 0003,0001,0000,0004,0002,000A,0005,0006
Boot0000* Windows Boot Manager	HD(1,GPT,6a1c0c2e-0f3b-4a52-9a44-0d5a3f1b2c3d,0x800,0x32000)/File(\EFI\Microsoft\Boot\bootmgfw.efi)57494e444f5753
Boot0001* Mk	HD(1,GPT,9d2b7c11-52aa-4c0e-8f6e-1b2a3c4d5e6f,0x800,0x400000)/File(\EFI\limine\BOOTX64.EFI)
Boot0002* UEFI OS	HD(1,GPT,9d2b7c11-52aa-4c0e-8f6e-1b2a3c4d5e6f,0x800,0x400000)/File(\EFI\BOOT\BOOTX64.EFI)0000424f
Boot0003* Limine	HD(1,GPT,9d2b7c11-52aa-4c0e-8f6e-1b2a3c4d5e6f,0x800,0x400000)/File(\EFI\limine\limine_x64.efi)
Boot0004* UEFI: SanDisk, Partition 1	PciRoot(0x0)/Pci(0x14,0x0)/USB(16,0)/HD(1,MBR,0x1a2b3c4d,0x800,0x1c9f800)0000424f
Boot0005* MkTunes	HD(1,GPT,6a1c0c2e-0f3b-4a52-9a44-0d5a3f1b2c3d,0x800,0x32000)/File(\EFI\mktunes\x.efi)
Boot0006* Limine	HD(1,GPT,6a1c0c2e-0f3b-4a52-9a44-0d5a3f1b2c3d,0x800,0x32000)/File(\EFI\outro\BOOTX64.EFI)
Boot000A  Mk	HD(1,GPT,11111111-52aa-4c0e-8f6e-1b2a3c4d5e6f,0x800,0x400000)/File(\EFI\limine\limine_x64.efi)'
printf '%s\n' "$efi_lista" >"$SBX/efi.txt"
stub_efi() { stub efibootmgr '[[ $# -eq 0 ]] && cat "$SBX/efi.txt"; [[ -e $SBX/efi-falha && $1 == --create ]] && exit 5; exit 0'; }
stub_efi
mkdir -p "$SBX/live/sys/firmware/efi/efivars"

# Deixa o alvo como o configure_boot deixaria, com o kernel já em /boot.
alvo_pronto() {
  rm -rf "${boot:?}"
  mkdir -p "$boot/EFI/limine" "$boot/EFI/BOOT"
  : >"$boot/EFI/limine/BOOTX64.EFI"
  : >"$boot/EFI/BOOT/BOOTX64.EFI"
  limine_header >"$boot/limine.conf"
  kernel
  rm -f "$SBX/limine-update"
  calls_reset
}
fim() { respostas_boas; finish_boot; }

t "finish_boot: chama o /usr/bin/mkinitcpio -P, não o wrapper que pergunta Y/n"
alvo_pronto
roda fim
eq 0 "$rc" "$out"
called "arch-chroot $target /usr/bin/mkinitcpio -P"
not_called "arch-chroot $target mkinitcpio"
eq 0 "$(perguntas)"

t "finish_boot sem os ganchos: grava as entradas fixas no limine.conf da ESP"
conf=$(<"$boot/limine.conf")
has "$conf" "timeout: 3"
has "$conf" "/Mk"
has "$conf" "protocol: linux"
has "$conf" "cmdline: $boot_cmdline"
not_called "arch-chroot $target limine-update"
has "$out" "limine: /boot/limine.conf"

t "finish_boot sem os ganchos: a entrada Mk aponta para a cópia EFI/limine/BOOTX64.EFI, que fica"
called 'efibootmgr --create --disk /dev/sdb --part 1 --label Mk --loader \EFI\limine\BOOTX64.EFI'
exists "$boot/EFI/limine/BOOTX64.EFI"
exists "$boot/EFI/BOOT/BOOTX64.EFI"

t "finish_boot: cria uma única entrada na NVRAM"
eq 1 "$(grep -c '^efibootmgr --create' "$SBX/calls")"

t "finish_boot: apaga a Mk antiga (ativa ou não) e a Limine do gancho"
called "efibootmgr --bootnum 0001 --delete-bootnum"
called "efibootmgr --bootnum 0003 --delete-bootnum"
called "efibootmgr --bootnum 000A --delete-bootnum"

t "finish_boot: não apaga Windows, UEFI OS, o pendrive, MkTunes nem uma Limine de outro caminho"
eq 3 "$(grep -c -- '--delete-bootnum' "$SBX/calls")"
for n in 0000 0002 0004 0005 0006; do not_called "--bootnum $n "; done

t "finish_boot: apaga as velhas antes de criar a nova"
eq "efibootmgr --create" "$(grep -E '^efibootmgr --(create|bootnum)' "$SBX/calls" | tail -1 | cut -d' ' -f1-2)"

t "finish_boot: acerta a entrada padrão do menu com o limine-default"
called "arch-chroot $target bash /usr/share/mk/system/mk/boot/limine-default"

t "finish_boot: com o initramfs de fallback, o menu fixo ganha a Mk (fallback)"
alvo_pronto
: >"$boot/initramfs-linux-fallback.img"
roda fim
has "$(<"$boot/limine.conf")" "/Mk (fallback)"

# O limine-update do gancho: escreve o limine.conf e instala o limine_x64.efi.
gancho() { # gancho <onde escreve, relativo a /boot> [kernel que ele aponta existe? sim|nao]
  cat >"$SBX/limine-update" <<SH
#!/bin/bash
root=\$1
mkdir -p "\$root/boot/EFI/limine" "\$root/boot/0123456789abcdef0123456789abcdef/linux"
: >"\$root/boot/EFI/limine/limine_x64.efi"
cp "$SBX/gancho.conf" "\$root/boot/$1"
[[ ${2:-sim} == sim ]] && : >"\$root/boot/0123456789abcdef0123456789abcdef/linux/vmlinuz-linux" && : >"\$root/boot/0123456789abcdef0123456789abcdef/linux/initramfs-linux.img"
exit ${3:-0}
SH
  chmod +x "$SBX/limine-update"
}
gancho_conf >"$SBX/gancho.conf"

t "finish_boot com os ganchos: o limine-update escreve as entradas e elas ficam"
alvo_pronto
gancho limine.conf
roda fim
eq 0 "$rc" "$out"
called "arch-chroot $target limine-update"
eq "$(<"$SBX/gancho.conf")" "$(<"$boot/limine.conf")"

t "finish_boot com os ganchos: a Mk aponta para o limine_x64.efi e a cópia EFI/limine/BOOTX64.EFI sai"
called 'efibootmgr --create --disk /dev/sdb --part 1 --label Mk --loader \EFI\limine\limine_x64.efi'
eq 1 "$(grep -c '^efibootmgr --create' "$SBX/calls")"
absent "$boot/EFI/limine/BOOTX64.EFI"
exists "$boot/EFI/limine/limine_x64.efi"

t "finish_boot com os ganchos: a cópia do caminho padrão EFI/BOOT continua"
exists "$boot/EFI/BOOT/BOOTX64.EFI"

t "finish_boot com os ganchos: o limine.conf na pasta do EFI é o que vale"
alvo_pronto
gancho EFI/limine/limine.conf
roda fim
eq 0 "$rc" "$out"
has "$out" "limine: /boot/EFI/limine/limine.conf"
eq "$(<"$SBX/gancho.conf")" "$(<"$boot/EFI/limine/limine.conf")"

t "finish_boot: se o gancho aponta para um kernel que não está em /boot, voltam as entradas fixas"
alvo_pronto
gancho EFI/limine/limine.conf nao
roda fim
eq 0 "$rc" "$out"
absent "$boot/EFI/limine/limine.conf"
conf=$(<"$boot/limine.conf")
has "$conf" "path: boot():/vmlinuz-linux"
hasnt "$conf" "0123456789abcdef"
has "$out" "limine: /boot/limine.conf"

t "finish_boot: se o limine-update falha, avisa e usa as entradas fixas"
alvo_pronto
printf '#!/bin/bash\nexit 1\n' >"$SBX/limine-update"
chmod +x "$SBX/limine-update"
roda fim
eq 0 "$rc" "$out"
has "$out" "limine-update falhou"
has "$(<"$boot/limine.conf")" "path: boot():/vmlinuz-linux"

t "finish_boot: se o gancho só deixou a EFI fallback, voltam as entradas fixas"
alvo_pronto
{
  limine_header
  printf '\n/EFI fallback\n    protocol: efi\n    path: boot():/EFI/BOOT/BOOTX64.EFI\n'
} >"$SBX/gancho.conf"
gancho limine.conf
roda fim
eq 0 "$rc" "$out"
has "$(<"$boot/limine.conf")" "/Mk"
has "$(<"$boot/limine.conf")" "protocol: linux"
gancho_conf >"$SBX/gancho.conf"

t "finish_boot: sem kernel em /boot, a instalação para"
alvo_pronto
rm -f "$boot/vmlinuz-linux"
roda fim
eq 1 "$rc"
has "$out" "não aponta para um kernel que exista em /boot"
not_called "efibootmgr"

t "finish_boot: se o mkinitcpio falha, a instalação para"
alvo_pronto
: >"$SBX/falha-mkinitcpio"
roda fim
fails test "$rc" -eq 0
not_called "efibootmgr"
rm -f "$SBX/falha-mkinitcpio"

t "finish_boot: se a entrada na NVRAM não puder ser criada, avisa e segue"
alvo_pronto
: >"$SBX/efi-falha"
roda fim
eq 0 "$rc" "$out"
has "$out" "AVISO:"
has "$out" "não consegui criar a entrada Mk na NVRAM"
rm -f "$SBX/efi-falha"

t "finish_boot: NVRAM vazia (nenhuma entrada antiga), só cria a Mk"
alvo_pronto
printf 'BootCurrent: 0000\nTimeout: 0 seconds\nBootOrder: 0000\nBoot0000* UEFI QEMU DVD-ROM QM00001 	PciRoot(0x0)/Pci(0x1f,0x2)/Sata(0,65535,0)N.....YM....R,Y.\n' >"$SBX/efi.txt"
roda fim
eq 0 "$rc" "$out"
not_called "--delete-bootnum"
eq 1 "$(grep -c '^efibootmgr --create' "$SBX/calls")"
printf '%s\n' "$efi_lista" >"$SBX/efi.txt"

t "finish_boot: sem efivars, não chama o efibootmgr"
alvo_pronto
rmdir "$SBX/live/sys/firmware/efi/efivars"
roda fim
eq 0 "$rc" "$out"
not_called "efibootmgr"
mkdir -p "$SBX/live/sys/firmware/efi/efivars"

t "finish_boot: o efibootmgr só recebe o disco escolhido"
alvo_pronto
roda fim
hasnt "$(grep '^efibootmgr' "$SBX/calls")" "/dev/sda"
hasnt "$(grep '^efibootmgr' "$SBX/calls")" "nvme"
