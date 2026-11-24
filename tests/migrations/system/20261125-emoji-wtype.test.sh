#!/usr/bin/env bash
# migrations/system/20261125-emoji-wtype.sh: só declara o wtype; quem instala é o
# mk-update, e a migração aguenta o pacote não ter vindo.
. "$(dirname "$0")/../../lib.sh"
. "$TESTS/migrations/mig.sh" system/20261125-emoji-wtype.sh

t "declara o pacote no cabeçalho"
eq "wtype" "$(sed -n 's/^# pacotes: //p' "$MIG")"

t "com o pacote instalado: sai com 0, calada, sem instalar nada"
instalados wtype
sem_mudanca mig
eq "" "$(cat "$OUT" "$ERR")"
not_called "pacman -S"

de_novo

t "sem o pacote (sem rede, por exemplo): avisa, sai com 0 e não tenta instalar"
instalados
sem_mudanca mig
has "$(cat "$OUT" "$ERR")" "wtype"
not_called "pacman -S"

t "só pergunta ao pacman, não chama mais nada"
eq "" "$(grep -v '^pacman -Q' "$SBX/calls")"
