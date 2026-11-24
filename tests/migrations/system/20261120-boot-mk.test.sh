#!/usr/bin/env bash
# migrations/system/20261120-boot-mk.sh: o tema do plymouth no lugar, o
# plymouthd.conf apontando para ele e o initramfs refeito.
. "$(dirname "$0")/../../lib.sh"
. "$TESTS/migrations/mig.sh" system/20261120-boot-mk.sh

tema=$R/usr/share/plymouth/themes/mk
stub mkinitcpio
mkdir -p "$R/usr/share/plymouth/themes" # a pasta é do pacote plymouth

t "sistema sem o tema: copia o tema, escreve o plymouthd.conf e refaz o initramfs"
ok mig
eq "" "$(diff -r "$SHARE/system/mk/plymouth/mk" "$tema" 2>&1)"
eq $'[Daemon]\nTheme=mk\nShowDelay=0' "$(cat "$R/etc/plymouth/plymouthd.conf")"
called "mkinitcpio -P"

t "um tema antigo no lugar é trocado inteiro (não sobra arquivo velho)"
echo velho >"$tema/sobra.png"
echo velho >"$tema/mk.script"
printf '[Daemon]\nTheme=spinner\n' >"$R/etc/plymouth/plymouthd.conf"
ok mig
absent "$tema/sobra.png"
eq "" "$(diff -r "$SHARE/system/mk/plymouth/mk" "$tema" 2>&1)"
has "$(cat "$R/etc/plymouth/plymouthd.conf")" "Theme=mk"

de_novo

t "se o mkinitcpio falhar, a migração falha (fica pendente para a próxima)"
stub mkinitcpio 'exit 1'
fails mig

t "não instala nada"
not_called "pacman"
