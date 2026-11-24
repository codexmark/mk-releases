#!/usr/bin/env bash
# migrations/system/20261202-ler-tela.sh: só declara o tesseract (com o
# português e o inglês) e o zbar, que o "ler" da captura usa; quem instala é o
# mk-update, e a migração aguenta o pacote não ter vindo.
. "$(dirname "$0")/../../lib.sh"
. "$TESTS/migrations/mig.sh" system/20261202-ler-tela.sh
pacotes="tesseract tesseract-data-por tesseract-data-eng zbar"

t "declara os pacotes no cabeçalho"
eq "$pacotes" "$(sed -n 's/^# pacotes: //p' "$MIG")"

t "com os pacotes instalados: sai com 0, calada, sem instalar nada"
# shellcheck disable=SC2086
instalados $pacotes
sem_mudanca mig
eq "" "$(cat "$OUT" "$ERR")"
not_called "pacman -S"

de_novo

t "sem os pacotes (sem rede, por exemplo): avisa de cada um, sai com 0 e não tenta instalar"
instalados zbar
sem_mudanca mig
for p in tesseract tesseract-data-por tesseract-data-eng; do
  has "$(cat "$ERR")" "$p"
done
not_called "pacman -S"

t "só pergunta ao pacman, não chama mais nada"
eq "" "$(grep -v '^pacman -Q' "$SBX/calls")"

t "os pacotes estão na lista do desktop (instalação nova)"
for p in $pacotes; do
  ok grep -qx "$p" "$REPO/system/packages/desktop.packages"
done
