#!/usr/bin/env bash
# Tela de entrada (sddm) com a cara do Mk.
set -euo pipefail
share=${MK_SHARE:-/usr/share/mk}
R=${MK_ROOT:-} # outra raiz no lugar de /: só os testes trocam (migrations/README.md)

rm -rf "$R/usr/share/sddm/themes/mk"
cp -r "$share/system/mk/sddm/mk" "$R/usr/share/sddm/themes/mk"
mkdir -p "$R/etc/sddm.conf.d"
printf '[Theme]\nCurrent=mk\n' >"$R/etc/sddm.conf.d/20-mk-theme.conf"
