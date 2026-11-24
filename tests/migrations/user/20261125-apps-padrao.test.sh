#!/usr/bin/env bash
# migrations/user/20261125-apps-padrao.sh: o mimeapps.list do Mk para quem
# ainda não tem um.
. "$(dirname "$0")/../../lib.sh"
. "$TESTS/migrations/mig.sh" user/20261125-apps-padrao.sh

dst=$XDG_CONFIG_HOME/mimeapps.list

t "home sem mimeapps.list: recebe o do Mk"
ok mig
ok cmp -s "$SHARE/config/mimeapps.list" "$dst"

de_novo

t "quem já tem um mimeapps.list fica com o seu, com aviso"
echo "[Default Applications]" >"$dst"
sem_mudanca mig
eq "[Default Applications]" "$(cat "$dst")"
has "$(cat "$OUT")" "mantido (já existe): $dst"

t "um link quebrado no lugar (dotfiles de outra máquina) não derruba a migração"
rm -f "$dst"
ln -s "$HOME/dotfiles/mimeapps.list" "$dst"
ok mig

t "não sai da home nem chama comando do sistema"
eq "" "$(find "$R" -mindepth 1)"
eq "" "$(cat "$SBX/calls")"
