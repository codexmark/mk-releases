#!/usr/bin/env bash
# migrations/system/20261128-entrada-marca.sh: a tela de entrada nova (a marca
# construída em movimento) chega aos Mk já instalados, sem perder o que a
# máquina tinha no theme.conf.user.
. "$(dirname "$0")/../../lib.sh"
. "$TESTS/migrations/mig.sh" system/20261128-entrada-marca.sh

temas=$R/usr/share/sddm/themes
tema=$temas/mk

t "troca o tema pelo novo, inteiro: as letras, a geometria e nenhuma sobra do antigo"
mkdir -p "$tema"
echo velho >"$tema/Main.qml"
echo sobra >"$tema/arquivo-que-saiu.qml"
ok mig
eq "" "$(diff -r "$SHARE/system/mk/sddm/mk" "$tema" 2>&1)"
exists "$tema/letra-m.png"
exists "$tema/letra-k.png"
exists "$tema/geometria.js"
absent "$tema/arquivo-que-saiu.qml"
absent "$temas/mk.novo"
absent "$temas/mk.velho"

de_novo

t "o theme.conf.user da máquina (teclado, movimento desligado) fica como estava"
printf '[General]\nlayout=br\nmotion=nao\n' >"$tema/theme.conf.user"
ok mig
eq $'[General]\nlayout=br\nmotion=nao' "$(cat "$tema/theme.conf.user")"
eq "" "$(diff -r -x theme.conf.user "$SHARE/system/mk/sddm/mk" "$tema" 2>&1)"

t "sem tema instalado (o sddm foi trocado por outro): não cria nada e sai com 0"
rm -rf "${temas:?}"
sem_mudanca mig
absent "$tema"

t "uma sobra de mk.novo (rodada interrompida) não atrapalha"
mkdir -p "$tema" "$temas/mk.novo"
echo lixo >"$temas/mk.novo/lixo"
ok mig
absent "$tema/lixo"
absent "$temas/mk.novo"
exists "$tema/Main.qml"

t "o Mk instalado sem o tema novo (lançamento incompleto): avisa e não apaga o que há"
rm -rf "${SHARE:?}/system/mk/sddm/mk"
echo atual >"$tema/Main.qml"
ok mig
eq atual "$(cat "$tema/Main.qml")"
has "$(cat "$ERR" "$OUT")" "aviso"
