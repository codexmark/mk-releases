#!/usr/bin/env bash
# migrations/system/20261121-bash-mk.sh: o seu.bash e o bash do Mk no skel, e
# o ble.sh na fila do AUR; o 7zip fica declarado, quem instala é o mk-update.
. "$(dirname "$0")/../../lib.sh"
. "$TESTS/migrations/mig.sh" system/20261121-bash-mk.sh

skel=$R/etc/skel/.config/mk
pend=$SHARE/aur-pendentes.txt

t "declara o 7zip no cabeçalho e não instala nada"
eq "7zip" "$(sed -n 's/^# pacotes: //p' "$MIG")"
ok mig
not_called "pacman -S"

t "o skel ganha o seu.bash e o bash do Mk, marcado como do Mk"
ok cmp -s "$SHARE/config/bash/seu.bash.exemplo" "$skel/seu.bash"
exists "$skel/bash/.instalado-pelo-mk"
eq "" "$(diff -r -x .instalado-pelo-mk "$SHARE/config/bash" "$skel/bash" 2>&1)"

t "sem ble.sh instalado, o blesh-git (o 0.4, o de aur.packages) entra na fila do AUR"
eq "blesh-git" "$(cat "$pend")"

de_novo

t "rodar de novo não repete o nome na fila"
eq 1 "$(wc -l <"$pend")"

t "um seu.bash que já estava no skel fica; a pasta bash antiga é trocada"
echo "alias k=kubectl" >"$skel/seu.bash"
echo velho >"$skel/bash/sobra"
ok mig
eq "alias k=kubectl" "$(cat "$skel/seu.bash")"
absent "$skel/bash/sobra"

t "a fila do AUR que já tinha outros nomes só ganha uma linha"
printf 'vercel\n' >"$pend"
ok mig
eq $'vercel\nblesh-git' "$(cat "$pend")"

t "com o blesh-git instalado, a fila não é criada"
rm -f "$pend"
instalados blesh-git
ok mig
absent "$pend"

t "com o blesh 0.3 instalado, a fila não é criada (a troca é da 20261122-boot-entrada)"
instalados blesh
ok mig
absent "$pend"

t "com o blesh 0.3 já na fila (do instalador), não acrescenta outro"
instalados
echo blesh >"$pend"
ok mig
eq "blesh" "$(cat "$pend")"
