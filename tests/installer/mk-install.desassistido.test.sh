#!/usr/bin/env bash
# installer/mk-install: o main com --answers, do começo ao fim, com tudo o que
# toca a máquina por stub. Ninguém pode ser perguntado de nada.
# shellcheck disable=SC2154,SC2034 # as variáveis vêm do instalador (carrega.sh)
. "$(dirname "$0")/../lib.sh"
. "$(dirname "$0")/carrega.sh"

fonte_falsa
sem_perguntas
mkdir -p "$SBX/live/sys/firmware/efi/efivars"

for c in curl swapoff umount wipefs sgdisk partprobe udevadm mkfs.fat mkfs.btrfs mount btrfs \
  timedatectl systemctl sync; do stub "$c"; done
stub cryptsetup 'cat >/dev/null'
stub mountpoint 'exit 1'
stub genfstab 'echo "UUID=2222-btrfs / btrfs rw,noatime,subvolid=256,subvol=/@ 0 0"'
stub blkid 'echo 2222-btrfs'
stub efibootmgr '[[ $# -eq 0 ]] && printf "BootOrder: 0000\nBoot0000* Mk\tHD(1,GPT,x)/File(\\\\EFI\\\\limine\\\\BOOTX64.EFI)\n"; exit 0'
# O pacstrap de verdade deixaria o sistema base no alvo (e o kernel em /boot).
export -f raiz_falsa
stub pacstrap '
[[ -e $SBX/pacstrap-falha ]] && exit 1
if [[ $1 == -K ]]; then
  raiz_falsa "$4"
  : >"$4/boot/vmlinuz-linux"
  : >"$4/boot/initramfs-linux.img"
fi'
# Depois do sgdisk, as partições novas existem.
printf '%s\n' '/dev/sdb2 part 250000000000 232G - /dev/sdb' >>"$SBX/discos"
printf '%s\n' wheel video audio input >"$SBX/grupos"
read_list "$PKG_DIR/aur.packages" >"$SBX/instalados"
printf '%s\n' yay-bin ufw-docker >"$aur_repo/packages.txt"

instala() { # instala [CHAVE=valor...]: mk-install --answers <arquivo com as trocas>
  rm -rf "${target:?}" "$log" "$SBX/home-antiga"
  mkdir -p "$target"
  calls_reset
  # shellcheck disable=SC2034 # lida pelo main
  answers=$(respostas PASSWORD=segredo-de-teste "$@")
  roda main
  # O main manda a saída por um tee: espera ele terminar de escrever.
  local antes='' agora i
  for i in {1..60}; do
    agora=$(cksum "$SBX/saida" "$log" 2>/dev/null)
    [[ $agora == "$antes" ]] && break
    antes=$agora
    sleep 0.05
  done
  out=$(<"$SBX/saida")
}
destrutivos='^(swapoff|umount|wipefs|sgdisk|mkfs[.a-z]*|pacstrap|mount|cryptsetup|btrfs|arch-chroot|efibootmgr) '
linha() { grep -n -m1 -F -- "$1" "$SBX/calls" | cut -d: -f1; }

t "com respostas: instala do começo ao fim sem erro"
instala
eq 0 "$rc" "$out"
has "$out" "Pronto. O Mk está em /dev/sdb"

t "com respostas: ninguém é perguntado de nada (nem tela do gum)"
eq 0 "$(perguntas)"
eq "" "$(<"$SBX/perguntas")"
eq 0 "$(grep -c '^gum ' "$SBX/calls")"
hasnt "$out" "PERGUNTA"

t "com respostas: a revisão aparece antes de formatar"
has "$out" "TUDO EM /dev/sdb SERÁ APAGADO"
has "$out" "confirmado pelo arquivo de respostas (CONFIRM=formatar)"

t "com respostas: as etapas na ordem do README"
ordem=("curl -fsI" "wipefs -af /dev/sdb" "pacstrap -K" "genfstab -U" "make-icons" "pacstrap -G -M"
  "useradd -m" "snapper --no-dbus" "systemctl enable NetworkManager.service" "/usr/bin/mkinitcpio -P"
  "efibootmgr --create" "neovim.sh" "umount -R $target")
for ((i = 1; i < ${#ordem[@]}; i++)); do
  a=$(linha "${ordem[i - 1]}") b=$(linha "${ordem[i]}")
  ok test "${a:-9999}" -lt "${b:-0}"
done

t "com respostas: 15 etapas numeradas, a última é desmontar"
has "$out" "[ 1/15]"
has "$out" "[15/15]"
has "$(sed 's/\x1b\[[0-9;]*m//g' <<<"$out")" "[15/15] desmontando"
hasnt "$out" "/16]"

t "com respostas: só o disco das respostas aparece nos comandos"
# (o lsblk só lê: é com ele que o instalador acha o pendrive do live)
eq "/dev/sdb /dev/sdb1 /dev/sdb2 " "$(grep -v '^lsblk ' "$SBX/calls" | grep -o '/dev/[A-Za-z0-9/_-]*' | sort -u | tr '\n' ' ')"

t "com respostas: tudo o que é montado, instalado ou desmontado fica no alvo"
while read -r l; do has "$l" "$target" "$l"; done < <(grep -E '^(mount|pacstrap|arch-chroot|genfstab|btrfs) ' "$SBX/calls")
eq "umount -R $target" "$(grep '^umount -R' "$SBX/calls")"

t "com respostas: o sistema novo tem o usuário, o boot e o desktop"
exists "$target/home/mark/.config/mk/hypr/.instalado-pelo-mk"
exists "$target/etc/sudoers.d/10-mk-wheel"
has "$(<"$target/boot/limine.conf")" "protocol: linux"
has "$(<"$target/etc/default/limine")" 'KERNEL_CMDLINE[default]="root=UUID=2222-btrfs'
eq mk "$(<"$target/etc/hostname")"
has "$(<"$target/etc/os-release")" 'NAME="Mk"'
absent "$target/etc/sddm.conf.d/10-mk.conf"

t "com respostas: o log começa depois da revisão e uma cópia fica no sistema novo"
has "$(<"$log")" "=== mk-install"
has "$(<"$log")" "-> /dev/sdb ==="
hasnt "$(<"$log")" "SERÁ APAGADO"
exists "$target/var/log/mk-install.log"
has "$(<"$target/var/log/mk-install.log")" "=== mk-install"

t "com respostas: a senha não aparece na tela, no log nem nos argumentos"
hasnt "$out" "segredo-de-teste"
hasnt "$(<"$log")" "segredo-de-teste"
hasnt "$(<"$SBX/calls")" "segredo-de-teste"
eq "mark:segredo-de-teste" "$(<"$SBX/chpasswd.entrada")"

t "com respostas: POWEROFF=sim desliga no fim, sem oferecer reiniciar"
called "systemctl poweroff"
not_called "systemctl reboot"
ok test "$(linha "umount -R $target")" -lt "$(linha 'systemctl poweroff')"

t "com respostas: POWEROFF=nao volta ao prompt, sem desligar nem reiniciar"
instala POWEROFF=nao
eq 0 "$rc" "$out"
not_called "systemctl poweroff"
not_called "systemctl reboot"
eq 0 "$(perguntas)"

t "com respostas: ISO sem pacotes do AUR termina em [14/15] e avisa"
: >"$aur_repo/packages.txt"
instala
eq 0 "$rc" "$out"
has "$out" "nenhum pacote do AUR no ISO"
has "$out" "[14/15]"
hasnt "$out" "[15/15]"
not_called "pacstrap -G"
printf '%s\n' yay-bin ufw-docker >"$aur_repo/packages.txt"

t "com respostas e LUKS: 16 etapas, o LUKS fecha no fim e o sddm entra direto"
instala ENCRYPT=sim
eq 0 "$rc" "$out"
has "$out" "[16/16]"
called "cryptsetup luksFormat --type luks2 --batch-mode --key-file - /dev/sdb2"
called "mkfs.btrfs -f -L mk /dev/mapper/mkroot"
ok test "$(linha "umount -R $target")" -lt "$(linha 'cryptsetup close mkroot')"
has "$(<"$target/etc/default/limine")" "rd.luks.name=2222-btrfs=mkroot root=/dev/mapper/mkroot"
has "$(<"$target/etc/mkinitcpio.conf.d/mk.conf")" "sd-encrypt"
has "$(<"$target/etc/sddm.conf.d/10-mk.conf")" "User=mark"
eq 0 "$(perguntas)"
hasnt "$(<"$SBX/calls")" "segredo-de-teste"

t "com respostas e apps: os apps entram no pacstrap"
instala APPS=sim "HARDWARE=amd nvidia"
eq 0 "$rc" "$out"
has "$(grep '^pacstrap -K' "$SBX/calls")" " nautilus "
has "$(grep '^pacstrap -K' "$SBX/calls")" " nvidia-open-dkms "
hasnt "$(<"$target/etc/mkinitcpio.conf.d/mk.conf")" " kms "

t "sem CONFIRM=formatar: para na revisão, sem apagar nem perguntar"
instala CONFIRM=
eq 1 "$rc"
has "$out" "use CONFIRM=formatar"
eq "" "$(grep -E "$destrutivos" "$SBX/calls")"
eq 0 "$(perguntas)"
absent "$log"

t "resposta inválida: para antes da revisão, sem apagar nem perguntar"
for troca in DISK=/dev/sdc DISK=/dev/sdz USERNAME=root ENCRYPT=yes TIMEZONE= LOCALE= KEYMAP= HARDWARE=radeon PASSWORD=; do
  instala "$troca"
  eq 1 "$rc" "$troca"
  has "$out" "ERRO:" "$troca"
  hasnt "$out" "SERÁ APAGADO" "$troca"
  eq "" "$(grep -E "$destrutivos" "$SBX/calls")" "$troca"
done
eq 0 "$(perguntas)"

t "chave desconhecida no arquivo: para antes de qualquer coisa"
instala SWAP=sim
eq 1 "$rc"
has "$out" "chave desconhecida"
eq "" "$(<"$SBX/calls")"

t "arquivo de respostas que não existe: para com mensagem"
calls_reset
# shellcheck disable=SC2034
answers=$SBX/nao-existe.conf
roda main
eq 1 "$rc"
has "$out" "não consigo ler"
eq "" "$(<"$SBX/calls")"

t "sem internet: para antes de tocar no disco"
stub curl 'exit 7'
stub sleep
instala
eq 1 "$rc"
has "$out" "sem internet"
eq "" "$(grep -E "$destrutivos" "$SBX/calls")"
rm -f "$SBX/bin/sleep"
stub curl

t "falha no meio (pacstrap): para, diz o código, o log e como tentar de novo"
: >"$SBX/pacstrap-falha"
instala
fails test "$rc" -eq 0
has "$out" "A instalação parou (código"
has "$out" "Log: $log"
has "$out" "umount -R $target"
eq 0 "$(grep -c -e '^efibootmgr' -e 'useradd' -e '^systemctl poweroff' "$SBX/calls")"
eq 0 "$(perguntas)"
rm -f "$SBX/pacstrap-falha"

t "sem respostas: aí sim pergunta (o detector de perguntas funciona) e nada é apagado"
calls_reset
answers=''
roda main
fails test "$rc" -eq 0
ok test "$(perguntas)" -gt 0
eq "" "$(grep -E "$destrutivos" "$SBX/calls")"
