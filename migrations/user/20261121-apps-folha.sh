#!/usr/bin/env bash
# As cores do Mk nos apps Qt (qt6ct), para quem ainda não tem um qt6ct seu.
set -euo pipefail
share=${MK_SHARE:-/usr/share/mk}
dst=${XDG_CONFIG_HOME:-$HOME/.config}/qt6ct

if [[ -e $dst ]]; then
  echo "mantido (já existe): $dst"
else
  mkdir -p "$dst"
  cp -r "$share/system/mk/look/qt6ct/." "$dst/"
fi
