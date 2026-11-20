#!/usr/bin/env bash
# A ficha compacta do fastfetch, com a marca nova, para quem já tinha o Mk.
# (O mk-update troca ~/.config/mk/fastfetch quando ela é do Mk; esta migração
# cobre quem ficou com a pasta antiga por ter atualizado antes dela existir.)
set -euo pipefail
share=${MK_SHARE:-/usr/share/mk}
dst=${XDG_CONFIG_HOME:-$HOME/.config}/mk/fastfetch

if [[ -e $dst && ! -e $dst/.instalado-pelo-mk ]]; then
  echo "mantido (não é do Mk): $dst"
  exit 0
fi
rm -rf "${dst:?}"
mkdir -p "$(dirname "$dst")"
cp -r "$share/config/fastfetch" "$dst"
touch "$dst/.instalado-pelo-mk"
