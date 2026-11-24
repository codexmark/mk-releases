#!/usr/bin/env bash
# migrations/user/20261121-apps-folha.sh: o qt6ct do Mk para quem ainda não
# tem um.
. "$(dirname "$0")/../../lib.sh"
. "$TESTS/migrations/mig.sh" user/20261121-apps-folha.sh

dst=$XDG_CONFIG_HOME/qt6ct

t "home sem qt6ct: recebe o do Mk"
ok mig
eq "" "$(diff -r "$SHARE/system/mk/look/qt6ct" "$dst" 2>&1)"

de_novo

t "quem já tem um qt6ct fica com o seu, com aviso"
rm -rf "$dst"
mkdir -p "$dst"
echo "[Appearance]" >"$dst/qt6ct.conf"
sem_mudanca mig
eq "[Appearance]" "$(cat "$dst/qt6ct.conf")"
has "$(cat "$OUT")" "mantido (já existe): $dst"

t "sem XDG_CONFIG_HOME, vai para ~/.config"
(unset XDG_CONFIG_HOME; HOME=$SBX/home/outra; mkdir -p "$HOME"; mig)
exists "$SBX/home/outra/.config/qt6ct/qt6ct.conf"

t "não sai da home nem chama comando do sistema"
eq "" "$(find "$R" -mindepth 1)"
eq "" "$(cat "$SBX/calls")"
