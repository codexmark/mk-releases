#!/usr/bin/env bash
# migrations/user/20261121-ficha-compacta.sh: a ficha nova do fastfetch para
# quem ficou com a pasta antiga do Mk.
. "$(dirname "$0")/../../lib.sh"
. "$TESTS/migrations/mig.sh" user/20261121-ficha-compacta.sh

dst=$XDG_CONFIG_HOME/mk/fastfetch

t "a ficha antiga do Mk (com a marca) é trocada inteira pela nova"
mkdir -p "$dst"
touch "$dst/.instalado-pelo-mk"
echo antiga >"$dst/config.jsonc"
echo antiga >"$dst/logo-grande.txt"
ok mig
eq "" "$(diff -r -x .instalado-pelo-mk "$SHARE/config/fastfetch" "$dst" 2>&1)"
exists "$dst/.instalado-pelo-mk"

de_novo

t "sem a pasta: cria a ficha do Mk, marcada"
rm -rf "$XDG_CONFIG_HOME"
ok mig
exists "$dst/config.jsonc"
exists "$dst/.instalado-pelo-mk"

t "uma ficha que não é do Mk (sem a marca) é mantida, com aviso"
rm -rf "$dst"
mkdir -p "$dst"
echo '{"minha": true}' >"$dst/config.jsonc"
sem_mudanca mig
has "$(cat "$OUT")" "mantido (não é do Mk): $dst"

t "não sai da home nem chama comando do sistema"
eq "" "$(find "$R" -mindepth 1)"
eq "" "$(cat "$SBX/calls")"
