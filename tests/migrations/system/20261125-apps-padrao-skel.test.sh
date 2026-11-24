#!/usr/bin/env bash
# migrations/system/20261125-apps-padrao-skel.sh: o mimeapps.list do Mk no
# skel, para os usuários criados depois.
. "$(dirname "$0")/../../lib.sh"
. "$TESTS/migrations/mig.sh" system/20261125-apps-padrao-skel.sh

dst=$R/etc/skel/.config/mimeapps.list

t "skel sem mimeapps.list: recebe o do Mk, legível por todos"
ok mig
ok cmp -s "$SHARE/config/mimeapps.list" "$dst"
eq 644 "$(stat -c %a "$dst")"

de_novo

t "um mimeapps.list que já estava no skel fica como está"
echo "[Default Applications]" >"$dst"
sem_mudanca mig
eq "[Default Applications]" "$(cat "$dst")"

t "não chama comando nenhum do sistema"
eq "" "$(cat "$SBX/calls")"
