#!/usr/bin/env bash
# O bash novo do Mk (ble.sh, histórico, atalhos, comando mk): cria o
# ~/.config/mk/seu.bash para os atalhos de cada um, se ainda não existir.
# (O mk.bash novo chega com a troca de ~/.config/mk/bash do mk-update.)
set -euo pipefail
share=${MK_SHARE:-/usr/share/mk}
cfg=${XDG_CONFIG_HOME:-$HOME/.config}/mk

[[ -e $cfg/seu.bash ]] || install -Dm644 "$share/config/bash/seu.bash.exemplo" "$cfg/seu.bash"
