#!/usr/bin/env bash
# migrations/system/20261120-entrada-mk.sh: o tema do sddm no lugar e o
# sddm.conf.d apontando para ele.
. "$(dirname "$0")/../../lib.sh"
. "$TESTS/migrations/mig.sh" system/20261120-entrada-mk.sh

tema=$R/usr/share/sddm/themes/mk
mkdir -p "$R/usr/share/sddm/themes" # a pasta é do pacote sddm

t "sistema sem o tema: copia o tema e escreve o 20-mk-theme.conf"
ok mig
eq "" "$(diff -r "$SHARE/system/mk/sddm/mk" "$tema" 2>&1)"
eq $'[Theme]\nCurrent=mk' "$(cat "$R/etc/sddm.conf.d/20-mk-theme.conf")"

t "um tema antigo no lugar é trocado inteiro, e os outros .conf do sddm ficam"
echo velho >"$tema/sobra.qml"
echo velho >"$tema/Main.qml"
echo "[Autologin]" >"$R/etc/sddm.conf.d/10-outro.conf"
ok mig
absent "$tema/sobra.qml"
eq "" "$(diff -r "$SHARE/system/mk/sddm/mk" "$tema" 2>&1)"
eq "[Autologin]" "$(cat "$R/etc/sddm.conf.d/10-outro.conf")"

de_novo

t "não chama comando nenhum do sistema"
eq "" "$(cat "$SBX/calls")"
