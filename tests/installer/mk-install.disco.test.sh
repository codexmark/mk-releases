#!/usr/bin/env bash
# installer/mk-install: partition. A sequência de comandos que apaga e
# formata o disco, toda por stub: só o disco escolhido pode aparecer.
# shellcheck disable=SC2154,SC2034 # as variáveis vêm do instalador (carrega.sh)
. "$(dirname "$0")/../lib.sh"
. "$(dirname "$0")/carrega.sh"

for c in swapoff umount wipefs sgdisk partprobe udevadm mkfs.fat mkfs.btrfs mount btrfs; do stub "$c"; done
# O cryptsetup recebe a senha pela entrada: guarda o que chegou.
stub cryptsetup 'cat >>"$SBX/cryptsetup.entrada"; echo >>"$SBX/cryptsetup.entrada"'

# Depois do sgdisk as partições novas "aparecem" no lsblk falso.
particoes() { # particoes <disco> <parte1> <parte2>
  printf '%s part 2147483648 2G - %s\n%s part 250000000000 232G - %s\n' "$2" "$1" "$3" "$1" >>"$SBX/discos"
}
particoes /dev/sdb /dev/sdb1 /dev/sdb2
particoes /dev/nvme0n1 /dev/nvme0n1p1 /dev/nvme0n1p2

particiona() { # particiona [CHAVE=valor...]
  local kv
  respostas_boas
  for kv in "$@"; do printf -v "${kv%%=*}" '%s' "${kv#*=}"; done
  partition
  echo "esp=$esp rootpart=$rootpart rootdev=$rootdev"
}
linha() { grep -n -m1 -F -- "$1" "$SBX/calls" | cut -d: -f1; }
# Todo /dev/... que apareceu nos argumentos de algum comando.
dispositivos() { grep -o '/dev/[A-Za-z0-9/_-]*' "$SBX/calls" | sort -u | tr '\n' ' '; }

t "partition: apaga as assinaturas e a tabela do disco escolhido"
calls_reset
roda particiona
eq 0 "$rc" "$out"
called "wipefs -af /dev/sdb"
called "sgdisk --zap-all /dev/sdb"

t "partition: GPT com a ESP de 2 GiB (MK_EFI) e o resto para o Mk"
called "sgdisk -n 1:0:+2G -t 1:ef00 -c 1:MK_EFI -n 2:0:0 -t 2:8304 -c 2:mk /dev/sdb"

t "partition: FAT32 na partição 1 e btrfs com o rótulo mk na 2"
called "mkfs.fat -F 32 -n MK_EFI /dev/sdb1"
called "mkfs.btrfs -f -L mk /dev/sdb2"
has "$out" "esp=/dev/sdb1 rootpart=/dev/sdb2 rootdev=/dev/sdb2"

t "partition: sem LUKS, o cryptsetup não é chamado"
not_called "cryptsetup"
hasnt "$out" "LUKS"

t "partition: desliga o swap e desmonta as partições antigas do disco antes de apagar"
called "swapoff -a"
called "umount -q /dev/sdb1"
ok test "$(linha 'umount -q /dev/sdb1')" -lt "$(linha 'wipefs')"

t "partition: a ordem é apagar, particionar, esperar o kernel, formatar, montar"
ok test "$(linha 'wipefs -af')" -lt "$(linha 'sgdisk --zap-all')"
ok test "$(linha 'sgdisk --zap-all')" -lt "$(linha 'sgdisk -n 1')"
ok test "$(linha 'sgdisk -n 1')" -lt "$(linha 'udevadm settle')"
ok test "$(linha 'udevadm settle')" -lt "$(linha 'mkfs.fat')"
ok test "$(linha 'mkfs.fat')" -lt "$(linha 'mkfs.btrfs')"
ok test "$(linha 'mkfs.btrfs')" -lt "$(linha 'mount /dev/sdb2')"

t "partition: os quatro subvolumes do README"
for s in @ @home @log @pkg; do called "btrfs -q subvolume create $target/$s"; done
eq 4 "$(grep -c '^btrfs ' "$SBX/calls")"

t "partition: cada subvolume no seu lugar, com noatime e zstd; a ESP em /boot"
called "mount -o noatime,compress=zstd,subvol=@ /dev/sdb2 $target"
called "mount -o noatime,compress=zstd,subvol=@home /dev/sdb2 $target/home"
called "mount -o noatime,compress=zstd,subvol=@log /dev/sdb2 $target/var/log"
called "mount -o noatime,compress=zstd,subvol=@pkg /dev/sdb2 $target/var/cache/pacman/pkg"
called "mount -o umask=0077 /dev/sdb1 $target/boot"
for d in boot home var/log var/cache/pacman/pkg; do exists "$target/$d"; done

t "partition: a raiz é montada antes do que fica dentro dela, e a ESP por último"
ok test "$(linha 'subvol=@ ')" -lt "$(linha 'subvol=@home')"
eq "mount -o umask=0077 /dev/sdb1 $target/boot" "$(tail -1 "$SBX/calls")"

t "partition: não cria swap"
not_called "mkswap"
not_called "swapon"

t "partition: só o disco escolhido e as partições dele aparecem nos argumentos"
eq "/dev/sdb /dev/sdb1 /dev/sdb2 " "$(dispositivos)"
hasnt "$(<"$SBX/calls")" "/dev/sda"
hasnt "$(<"$SBX/calls")" "nvme"
hasnt "$(<"$SBX/calls")" "/dev/sdc"

t "partition: todo destino de montagem fica dentro do alvo"
while read -r l; do
  has "${l##* }" "$target" "$l"
done < <(grep '^mount ' "$SBX/calls")

t "partition num nvme: as partições são p1 e p2, e só o nvme aparece"
calls_reset
roda particiona DISK=/dev/nvme0n1
eq 0 "$rc" "$out"
called "wipefs -af /dev/nvme0n1"
called "mkfs.fat -F 32 -n MK_EFI /dev/nvme0n1p1"
called "mkfs.btrfs -f -L mk /dev/nvme0n1p2"
eq "/dev/nvme0n1 /dev/nvme0n1p1 /dev/nvme0n1p2 " "$(dispositivos)"

t "partition: se as partições não aparecem, para antes de formatar"
calls_reset
sed -i '\|^/dev/sdb[12] |d' "$SBX/discos"
roda particiona
eq 1 "$rc"
has "$out" "as partições /dev/sdb1 e /dev/sdb2 não apareceram"
not_called "mkfs"
eq 0 "$(grep -c '^mount ' "$SBX/calls")"
particoes /dev/sdb /dev/sdb1 /dev/sdb2

t "partition: se o wipefs falha, para sem particionar"
calls_reset
stub wipefs 'exit 1'
roda particiona
fails test "$rc" -eq 0
not_called "sgdisk"
not_called "mkfs"
stub wipefs

t "partition com LUKS: LUKS2 na partição 2, aberto como mkroot"
calls_reset
: >"$SBX/cryptsetup.entrada"
roda particiona ENCRYPT=sim
eq 0 "$rc" "$out"
called "cryptsetup luksFormat --type luks2 --batch-mode --key-file - /dev/sdb2"
called "cryptsetup open --key-file - /dev/sdb2 mkroot"
has "$out" "esp=/dev/sdb1 rootpart=/dev/sdb2 rootdev=/dev/mapper/mkroot"

t "partition com LUKS: o btrfs e as montagens ficam dentro do LUKS; a ESP não"
called "mkfs.btrfs -f -L mk /dev/mapper/mkroot"
not_called "mkfs.btrfs -f -L mk /dev/sdb2"
called "mount -o noatime,compress=zstd,subvol=@ /dev/mapper/mkroot $target"
called "mkfs.fat -F 32 -n MK_EFI /dev/sdb1"
called "mount -o umask=0077 /dev/sdb1 $target/boot"
ok test "$(linha 'cryptsetup open')" -lt "$(linha 'mkfs.btrfs')"

t "partition com LUKS: a senha vai pela entrada, nunca nos argumentos nem na tela"
eq "segredo-de-teste
segredo-de-teste" "$(<"$SBX/cryptsetup.entrada")"
hasnt "$(<"$SBX/calls")" "segredo-de-teste"
hasnt "$out" "segredo-de-teste"

t "partition com LUKS: só o disco escolhido e o mapper aparecem"
eq "/dev/mapper/mkroot /dev/sdb /dev/sdb1 /dev/sdb2 " "$(dispositivos)"

t "partition: não pergunta nada"
eq 0 "$(perguntas)"
