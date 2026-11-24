#!/usr/bin/env bash
# migrations/system/20261123-webp.sh: só declara o qt6-imageformats; quem instala é o
# mk-update, e a migração aguenta o pacote não ter vindo.
. "$(dirname "$0")/../../lib.sh"
. "$TESTS/migrations/mig.sh" system/20261123-webp.sh

t "declara o pacote no cabeçalho"
eq "qt6-imageformats" "$(sed -n 's/^# pacotes: //p' "$MIG")"

t "com o pacote instalado: sai com 0, calada, sem instalar nada"
instalados qt6-imageformats
sem_mudanca mig
eq "" "$(cat "$OUT" "$ERR")"
not_called "pacman -S"

de_novo

t "sem o pacote (sem rede, por exemplo): avisa, sai com 0 e não tenta instalar"
instalados
sem_mudanca mig
has "$(cat "$OUT" "$ERR")" "qt6"
not_called "pacman -S"

t "só pergunta ao pacman, não chama mais nada"
eq "" "$(grep -v '^pacman -Q' "$SBX/calls")"
