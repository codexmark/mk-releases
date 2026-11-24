#!/usr/bin/env bash
# migrations/system/20261127-editor-nuvem.sh: code, aws-cli-v2 e wrangler
# declarados (o mk-update instala) e a vercel na fila do AUR.
. "$(dirname "$0")/../../lib.sh"
. "$TESTS/migrations/mig.sh" system/20261127-editor-nuvem.sh

pend=$SHARE/aur-pendentes.txt

t "declara os três pacotes no cabeçalho"
eq "code aws-cli-v2 wrangler" "$(sed -n 's/^# pacotes: //p' "$MIG")"

t "com os três instalados e sem a vercel: a vercel entra na fila do AUR, sem avisos"
instalados code aws-cli-v2 wrangler
ok mig
eq "vercel" "$(cat "$pend")"
eq "" "$(cat "$ERR")"
not_called "pacman -S"

de_novo

t "rodar de novo não repete a vercel na fila"
eq 1 "$(wc -l <"$pend")"

t "a fila que já tinha outros nomes só ganha uma linha"
printf 'blesh-git\n' >"$pend"
ok mig
eq $'blesh-git\nvercel' "$(cat "$pend")"

t "com a vercel já instalada, a fila não é criada"
rm -f "$pend"
instalados code aws-cli-v2 wrangler vercel
sem_mudanca mig
absent "$pend"

t "um pacote que não veio (sem rede): avisa qual, sai com 0 e não tenta instalar"
instalados code vercel
ok mig
has "$(cat "$ERR")" "aviso: aws-cli-v2"
has "$(cat "$ERR")" "aviso: wrangler"
hasnt "$(cat "$ERR")" "aviso: code"
not_called "pacman -S"
