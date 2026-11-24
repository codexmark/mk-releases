#!/usr/bin/env bash
# bin/mk-hibernate: o setup recusa, o status descreve e o remove desfaz só o
# que o arquivo de estado anota, numa raiz falsa (MK_HIB_ROOT).
. "$(dirname "$0")/../lib.sh"

hib=$REPO/bin/mk-hibernate
R=$SBX/raiz
estado=$R/var/lib/mk/hibernate/estado
fstab=$R/etc/fstab
limine=$R/etc/default/limine
linha="/swap/swapfile none swap defaults,pri=0 0 0"
nosso="resume=UUID=1234-abcd resume_offset=5678"

# Uma máquina que o setup antigo preparou por inteiro.
preparada() {
  rm -rf "$R"
  mkdir -p "$R"/{etc/default,proc,swap,var/lib/mk/hibernate}
  printf '%s\n' "UUID=1234-abcd / btrfs subvol=/@ 0 0" "UUID=EF00 /boot vfat defaults 0 2" "$linha" >"$fstab"
  printf '%s\n' 'ESP_PATH="/boot"' "KERNEL_CMDLINE[default]=\"quiet splash $nosso rw\"" 'MAX_SNAPSHOT_ENTRIES=auto' >"$limine"
  printf '%s\n' "Filename Type Size Used Priority" "/swap/swapfile file 16777212 0 0" >"$R/proc/swaps"
  echo "quiet splash $nosso rw" >"$R/proc/cmdline"
  echo imagem >"$R/swap/swapfile"
  printf '%s\n' "subvol=1" "swapfile=1" "fstab=1" "cmdline=$nosso" >"$estado"
  calls_reset
}
# O swapoff de verdade tiraria o arquivo de /proc/swaps.
stub swapoff 'sed -i "/swapfile/d" "$MK_HIB_ROOT/proc/swaps"'
stub btrfs
stub limine-update
printf '#!/bin/bash\necho "limine-default" >>"%s"\n' "$SBX/calls" >"$SBX/limine-default"

run() { MK_HIB_ROOT=$R MK_HIB_YES=${SIM-1} MK_HIB_LIMINE_DEFAULT=$SBX/limine-default bash "$hib" "$@" 2>&1; }
cmdline() { sed -n 's/^KERNEL_CMDLINE\[default\]+\{0,1\}="\(.*\)"$/\1/p' "$limine"; }

# -------------------------------------------------------------------- setup

t "setup: recusa, porque a hibernação foi desativada, e não mexe em nada"
preparada
antes=$(cd "$R" && find . -type f -exec md5sum {} + | sort)
out=$(run setup)
eq 1 $?
has "$out" "a hibernação está desativada no Mk"
has "$out" "sudo mk-hibernate remove"
eq "$antes" "$(cd "$R" && find . -type f -exec md5sum {} + | sort)"
eq "" "$(cat "$SBX/calls")"

t "sem argumento, ou com um desconhecido: mostra o uso e sai com 2"
out=$(run)
eq 2 $?
has "$out" "mk-hibernate remove"
has "$out" "mk-hibernate status"
run bobagem >/dev/null
eq 2 $?
eq "" "$(cat "$SBX/calls")"

# ------------------------------------------------------------------- status

t "status: máquina preparada diz o que há e como desfazer"
preparada
out=$(run status)
eq 0 $?
has "$out" "preparado pelo mk-hibernate: sim"
has "$out" "swapfile /swap/swapfile:        existe"
has "$out" "swap ligado:                 sim"
has "$out" "linha no /etc/fstab:         sim"
has "$out" "resume= no boot configurado: $nosso"
has "$out" "kernel subiu com resume=:    sim"
has "$out" "Para desfazer: sudo mk-hibernate remove"

t "status: não mexe em nada e não chama comando nenhum"
eq "" "$(cat "$SBX/calls")"
exists "$estado"

t "status: máquina sem preparo (nem /proc/swaps, nem fstab) diz 'Nada preparado'"
rm -rf "$R"
mkdir -p "$R"
out=$(run status)
eq 0 $?
has "$out" "preparado pelo mk-hibernate: não"
has "$out" "swapfile /swap/swapfile:        não existe"
has "$out" "swap ligado:                 não"
has "$out" "linha no /etc/fstab:         não"
has "$out" "resume= no boot configurado: não"
has "$out" "kernel subiu com resume=:    não"
has "$out" "Nada preparado. O Mk não hiberna."

t "status: preparo pela metade (swap desligado) não conta como preparado"
preparada
echo "Filename Type Size Used Priority" >"$R/proc/swaps"
out=$(run status)
has "$out" "swap ligado:                 não"
has "$out" "Nada preparado."

t "status: um swap de outro arquivo não é confundido com o do Mk"
preparada
printf '%s\n' "Filename Type Size Used Priority" "/dev/zram0 partition 8388604 0 100" >"$R/proc/swaps"
has "$(run status)" "swap ligado:                 não"

# ------------------------------------------------- remove, com estado completo

t "remove: sem arquivo de estado não há o que desfazer, e nada é tocado"
preparada
rm "$estado"
antes=$(cd "$R" && find . -type f -exec md5sum {} + | sort)
out=$(run remove)
eq 0 $?
has "$out" "Nada a desfazer"
eq "$antes" "$(cd "$R" && find . -type f -exec md5sum {} + | sort)"
eq "" "$(cat "$SBX/calls")"

t "remove: desliga o swap do swapfile"
preparada
out=$(run remove)
eq 0 $?
called "swapoff $R/swap/swapfile"

t "remove: tira do /etc/fstab só a linha do setup, e guarda uma cópia de antes"
eq "UUID=1234-abcd / btrfs subvol=/@ 0 0
UUID=EF00 /boot vfat defaults 0 2" "$(cat "$fstab")"
copia=$(find "$R/var/lib/mk/hibernate" -name 'fstab.*' | head -1)
has "$(cat "$copia" 2>/dev/null)" "$linha"
has "$out" "cópia: /var/lib/mk/hibernate/fstab."

t "remove: tira o resume= do boot, deixa o resto da linha e do arquivo, e guarda uma cópia"
eq "quiet splash rw" "$(cmdline)"
has "$(cat "$limine")" 'ESP_PATH="/boot"'
has "$(cat "$limine")" 'MAX_SNAPSHOT_ENTRIES=auto'
eq 3 "$(grep -c '' "$limine")"
copia=$(find "$R/var/lib/mk/hibernate" -name 'limine.*' | head -1)
has "$(cat "$copia" 2>/dev/null)" "$nosso"

t "remove: refaz o menu de boot (limine-update) e acerta a entrada padrão"
called "limine-update"
called "limine-default"

t "remove: apaga o swapfile e o subvolume /swap"
absent "$R/swap/swapfile"
called "btrfs subvolume delete $R/swap"

t "remove: no fim apaga o estado e avisa que o resume= some no próximo boot"
absent "$estado"
has "$out" "Pronto."
has "$out" "some no próximo boot"
hasnt "$out" "aviso:"

t "remove: rodar de novo não faz mais nada"
calls_reset
antes=$(cd "$R" && find . -type f -exec md5sum {} + | sort)
out=$(run remove)
eq 0 $?
has "$out" "Nada a desfazer"
eq "$antes" "$(cd "$R" && find . -type f -exec md5sum {} + | sort)"
eq "" "$(cat "$SBX/calls")"

t "remove: depois dele o status diz 'Nada preparado'"
has "$(run status)" "Nada preparado."

t "remove: o /etc/default/limine mantém as permissões"
preparada
chmod 600 "$limine"
run remove >/dev/null
eq 600 "$(stat -c %a "$limine")"

t "remove: a forma KERNEL_CMDLINE[default]+= é mantida"
preparada
printf '%s\n' "KERNEL_CMDLINE[default]+=\"quiet $nosso\"" >"$limine"
run remove >/dev/null
eq 'KERNEL_CMDLINE[default]+="quiet"' "$(cat "$limine")"

t "remove: com o swap já desligado, não chama o swapoff"
preparada
echo "Filename Type Size Used Priority" >"$R/proc/swaps"
out=$(run remove)
eq 0 $?
not_called "swapoff"
absent "$R/swap/swapfile"

# ------------------------------------------------------- remove: a confirmação

t "remove: pergunta antes; 'n' cancela e nada é mexido"
preparada
antes=$(cd "$R" && find . -type f -exec md5sum {} + | sort)
out=$(SIM="" run remove <<<"n")
eq 1 $?
has "$out" "cancelado; nada foi mexido"
eq "$antes" "$(cd "$R" && find . -type f -exec md5sum {} + | sort)"
eq "" "$(cat "$SBX/calls")"

t "remove: sem resposta (entrada fechada) também cancela"
SIM="" run remove </dev/null >/dev/null
eq 1 $?
exists "$estado"
exists "$R/swap/swapfile"

t "remove: 's' continua"
SIM="" run remove <<<"s" >/dev/null
eq 0 $?
absent "$estado"
absent "$R/swap/swapfile"

if ((EUID != 0)); then
  t "remove: fora da raiz de teste, sem root, recusa antes de qualquer coisa"
  calls_reset
  out=$(MK_HIB_YES="" bash "$hib" remove 2>&1 </dev/null)
  eq 1 $?
  has "$out" "precisa de root: sudo mk-hibernate remove"
  eq "" "$(cat "$SBX/calls")"
fi

# --------------------------------------------------- remove, com estado parcial

t "estado só com o swapfile: apaga o arquivo e não toca em fstab, boot nem subvolume"
preparada
echo "swapfile=1" >"$estado"
f_antes=$(cat "$fstab") l_antes=$(cat "$limine")
out=$(run remove)
eq 0 $?
absent "$R/swap/swapfile"
eq "$f_antes" "$(cat "$fstab")"
eq "$l_antes" "$(cat "$limine")"
not_called "limine-update"
not_called "limine-default"
not_called "btrfs"
exists "$R/swap"
absent "$estado"

t "estado só com o swapfile: avisa que a linha do fstab (que não é dele) ficou sem o arquivo"
has "$out" "aviso: o /etc/fstab ainda tem uma linha do /swap/swapfile"

t "estado só com o fstab: tira a linha e deixa o swapfile e o boot como estão"
preparada
echo "fstab=1" >"$estado"
l_antes=$(cat "$limine")
run remove >/dev/null
eq 0 $?
hasnt "$(cat "$fstab")" "swapfile"
exists "$R/swap/swapfile"
eq "$l_antes" "$(cat "$limine")"
not_called "limine-update"
not_called "btrfs"

t "estado só com o cmdline: tira o resume= e refaz o menu; fstab, swapfile e subvolume ficam"
preparada
echo "cmdline=$nosso" >"$estado"
f_antes=$(cat "$fstab")
run remove >/dev/null
eq 0 $?
eq "quiet splash rw" "$(cmdline)"
called "limine-update"
eq "$f_antes" "$(cat "$fstab")"
exists "$R/swap/swapfile"
not_called "btrfs"

t "estado só com o subvolume: só o subvolume sai"
preparada
echo "subvol=1" >"$estado"
run remove >/dev/null
eq 0 $?
called "btrfs subvolume delete $R/swap"
has "$(cat "$fstab")" "$linha"
has "$(cmdline)" "resume="
not_called "limine-update"

t "estado com o subvolume anotado, mas a pasta /swap já não existe: não chama o btrfs"
preparada
rm -rf "$R/swap"
run remove >/dev/null
eq 0 $?
not_called "btrfs"
absent "$estado"

t "estado vazio: nada é desfeito, e o arquivo de estado sai"
preparada
: >"$estado"
f_antes=$(cat "$fstab") l_antes=$(cat "$limine")
run remove >/dev/null
eq 0 $?
eq "$f_antes" "$(cat "$fstab")"
eq "$l_antes" "$(cat "$limine")"
exists "$R/swap/swapfile"
not_called "btrfs"
not_called "limine-update"
absent "$estado"

t "estado com a chave repetida: vale a última"
preparada
printf '%s\n' "cmdline=resume=UUID=velho" "cmdline=$nosso" >"$estado"
run remove >/dev/null
eq "quiet splash rw" "$(cmdline)"

# ------------------------------------- fstab e limine editados à mão depois

t "fstab: a linha já tirada à mão: não mexe no arquivo nem faz cópia, e o resto é desfeito"
preparada
printf '%s\n' "UUID=1234-abcd / btrfs subvol=/@ 0 0" "# meu comentário" >"$fstab"
f_antes=$(cat "$fstab")
out=$(run remove)
eq 0 $?
has "$out" "a linha não está lá"
eq "$f_antes" "$(cat "$fstab")"
eq 0 "$(find "$R/var/lib/mk/hibernate" -name 'fstab.*' | wc -l)"
absent "$R/swap/swapfile"
eq "quiet splash rw" "$(cmdline)"
hasnt "$out" "aviso:"

t "fstab: a linha trocada à mão (outras opções) não é do setup: fica, com aviso"
preparada
printf '%s\n' "UUID=1234-abcd / btrfs subvol=/@ 0 0" "/swap/swapfile none swap defaults,pri=10 0 0" >"$fstab"
f_antes=$(cat "$fstab")
out=$(run remove)
eq 0 $?
eq "$f_antes" "$(cat "$fstab")"
has "$out" "aviso: o /etc/fstab ainda tem uma linha do /swap/swapfile"
absent "$R/swap/swapfile"

t "fstab: outras linhas, comentários e outro swap ficam como estavam"
preparada
printf '%s\n' "# /etc/fstab" "UUID=1234-abcd / btrfs subvol=/@ 0 0" "" "$linha" "/dev/sdb2 none swap defaults 0 0" "# $linha" >"$fstab"
run remove >/dev/null
eq "# /etc/fstab
UUID=1234-abcd / btrfs subvol=/@ 0 0

/dev/sdb2 none swap defaults 0 0
# $linha" "$(cat "$fstab")"

t "limine: o resume= já tirado à mão: não regrava nem faz cópia, mas refaz o menu"
preparada
printf '%s\n' 'KERNEL_CMDLINE[default]="quiet splash rw"' >"$limine"
out=$(run remove)
eq 0 $?
has "$out" "já não estava"
eq 'KERNEL_CMDLINE[default]="quiet splash rw"' "$(cat "$limine")"
eq 0 "$(find "$R/var/lib/mk/hibernate" -name 'limine.*' | wc -l)"
called "limine-update"
absent "$estado"

t "limine: parâmetros postos à mão depois (e fora de ordem) ficam; só os do setup saem"
preparada
printf '%s\n' "KERNEL_CMDLINE[default]=\"resume_offset=5678 quiet mitigations=off resume=UUID=1234-abcd nowatchdog\"" >"$limine"
run remove >/dev/null
eq "quiet mitigations=off nowatchdog" "$(cmdline)"

t "limine: um resume= diferente do anotado (trocado à mão) não é do setup: fica"
preparada
printf '%s\n' 'KERNEL_CMDLINE[default]="quiet resume=UUID=outro resume_offset=5678"' >"$limine"
run remove >/dev/null
eq "quiet resume=UUID=outro" "$(cmdline)"

t "limine: sem o arquivo, para dizendo qual, e mantém o estado para tentar de novo"
preparada
rm "$limine"
out=$(run remove)
eq 1 $?
has "$out" "não achei /etc/default/limine"
exists "$estado"
exists "$R/swap/swapfile"
not_called "btrfs"

t "limine: duas linhas KERNEL_CMDLINE[default] (editado à mão): para sem escolher uma"
preparada
printf '%s\n' "KERNEL_CMDLINE[default]=\"quiet $nosso\"" 'KERNEL_CMDLINE[default]+="rw"' >"$limine"
l_antes=$(cat "$limine")
out=$(run remove)
eq 1 $?
has "$out" "e só uma"
eq "$l_antes" "$(cat "$limine")"
exists "$estado"
exists "$R/swap/swapfile"
not_called "limine-update"

t "limine: linha sem aspas (formato que ele não entende): para sem regravar"
preparada
printf '%s\n' "KERNEL_CMDLINE[default]=quiet" >"$limine"
out=$(run remove)
eq 1 $?
has "$out" "não entendi a linha 1"
eq "KERNEL_CMDLINE[default]=quiet" "$(cat "$limine")"
exists "$estado"

# --------------------------------------------------- quando um passo falha

t "o swapoff falha: para ali, sem tocar em fstab, boot, swapfile nem estado"
preparada
stub swapoff 'exit 1'
f_antes=$(cat "$fstab") l_antes=$(cat "$limine")
run remove >/dev/null
[[ $? -ne 0 ]] || _fail "devia sair com erro"
eq "$f_antes" "$(cat "$fstab")"
eq "$l_antes" "$(cat "$limine")"
exists "$R/swap/swapfile"
exists "$estado"
stub swapoff 'sed -i "/swapfile/d" "$MK_HIB_ROOT/proc/swaps"'

t "o limine-update falha: diz para rodar de novo e mantém o estado e o swapfile"
preparada
stub limine-update 'exit 1'
out=$(run remove)
eq 1 $?
has "$out" "o limine-update falhou"
has "$out" "rode de novo: sudo mk-hibernate remove"
exists "$estado"
exists "$R/swap/swapfile"
not_called "btrfs"

t "depois do limine-update consertado, rodar de novo termina o serviço"
stub limine-update
calls_reset
out=$(run remove)
eq 0 $?
absent "$estado"
absent "$R/swap/swapfile"
called "btrfs subvolume delete $R/swap"
called "limine-update"
not_called "swapoff"
eq "quiet splash rw" "$(cmdline)"
hasnt "$(cat "$fstab")" "swapfile"

t "o limine-default falha: só avisa, e o remove termina"
preparada
printf '#!/bin/bash\nexit 1\n' >"$SBX/limine-default"
out=$(run remove)
eq 0 $?
has "$out" "aviso: o limine-default falhou"
absent "$estado"
