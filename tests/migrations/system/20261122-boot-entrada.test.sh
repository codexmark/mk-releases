#!/usr/bin/env bash
# migrations/system/20261122-boot-entrada.sh: o menu de boot inicia sozinho, o
# sddm mostra o teclado certo e o ble.sh 0.3 dá lugar ao blesh-git.
. "$(dirname "$0")/../../lib.sh"
. "$TESTS/migrations/mig.sh" system/20261122-boot-entrada.sh

tema=$R/usr/share/sddm/themes/mk
pend=$SHARE/aur-pendentes.txt
teclado() {
  mkdir -p "$R/etc/X11/xorg.conf.d"
  printf 'Section "InputClass"\n    Identifier "system-keyboard"\n    Option "XkbLayout" "%s"\nEndSection\n' "$1" \
    >"$R/etc/X11/xorg.conf.d/00-keyboard.conf"
}

t "chama o limine-default do Mk instalado"
teclado br
mkdir -p "$tema"
echo velho >"$tema/sobra.qml"
ok mig
called "limine-default"

t "o tema do sddm é o novo, com o layout do teclado do sistema no theme.conf.user"
eq "" "$(diff -r -x theme.conf.user "$SHARE/system/mk/sddm/mk" "$tema" 2>&1)"
eq $'[General]\nlayout=br' "$(cat "$tema/theme.conf.user")"
absent "$R/usr/share/sddm/themes/mk.novo"

t "sem ble.sh instalado nem na fila, não mexe em pacote nem cria a fila"
not_called "pacman -R"
not_called "pacman -S"
absent "$pend"

de_novo

t "o layout segue o do sistema (us)"
teclado us
ok mig
eq $'[General]\nlayout=us' "$(cat "$tema/theme.conf.user")"

t "sem o 00-keyboard.conf: o tema entra sem theme.conf.user e a migração sai com 0"
rm -rf "$R/etc/X11"
ok mig
exists "$tema/Main.qml"
absent "$tema/theme.conf.user"
absent "$R/usr/share/sddm/themes/mk.novo"

t "um 00-keyboard.conf sem XkbLayout: idem"
mkdir -p "$R/etc/X11/xorg.conf.d"
echo 'Section "InputClass"' >"$R/etc/X11/xorg.conf.d/00-keyboard.conf"
ok mig
absent "$tema/theme.conf.user"

t "uma sobra de mk.novo (rodada interrompida) não atrapalha"
mkdir -p "$R/usr/share/sddm/themes/mk.novo"
echo lixo >"$R/usr/share/sddm/themes/mk.novo/lixo"
ok mig
absent "$tema/lixo"
absent "$R/usr/share/sddm/themes/mk.novo"

t "com o blesh 0.3 instalado: remove e põe o blesh-git na fila do AUR"
instalados blesh
calls_reset
ok mig
called "pacman -Rns --noconfirm blesh"
eq "blesh-git" "$(cat "$pend")"
not_called "pacman -S"

t "de novo, já sem o blesh: não remove outra vez nem repete a fila"
calls_reset
sem_mudanca mig
not_called "pacman -R"
eq 1 "$(wc -l <"$pend")"

t "o blesh 0.3 que estava na fila do AUR vira blesh-git, e o resto da fila fica"
printf 'vercel\nblesh\nlocalsend-bin\n' >"$pend"
ok mig
eq $'vercel\nblesh-git\nlocalsend-bin' "$(cat "$pend")"

t "com os dois na fila, sai só o 0.3"
printf 'blesh\nblesh-git\n' >"$pend"
ok mig
eq "blesh-git" "$(cat "$pend")"
