#!/usr/bin/env bash
# migrations/system/20261128-shellcheck.sh: só declara o shellcheck (o lint de tests/run); quem instala é o
# mk-update, e a migração aguenta o pacote não ter vindo.
. "$(dirname "$0")/../../lib.sh"
. "$TESTS/migrations/mig.sh" system/20261128-shellcheck.sh

t "declara o pacote no cabeçalho"
eq "shellcheck" "$(sed -n 's/^# pacotes: //p' "$MIG")"

t "com o pacote instalado: sai com 0, calada, sem instalar nada"
instalados shellcheck
sem_mudanca mig
eq "" "$(cat "$OUT" "$ERR")"
not_called "pacman -S"

de_novo

t "sem o pacote (sem rede, por exemplo): avisa, sai com 0 e não tenta instalar"
instalados
sem_mudanca mig
has "$(cat "$OUT" "$ERR")" "shellcheck"
not_called "pacman -S"

t "só pergunta ao pacman, não chama mais nada"
eq "" "$(grep -v '^pacman -Q' "$SBX/calls")"

t "o shellcheck está na lista de desenvolvimento (instalação nova)"
ok grep -qx shellcheck "$REPO/system/packages/dev.packages"
