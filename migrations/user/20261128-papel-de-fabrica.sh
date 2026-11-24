#!/usr/bin/env bash
# O Mk passou a nascer com um papel de parede (Maré, dos autorais). Quem já
# usava o Mk escolheu o seu, ou escolheu não ter: marca que o de fábrica não é
# para esta pessoa, para ele não aparecer sozinho depois de uma atualização.
# (O Maré só aplica o de fábrica com esta migração já feita e sem a marca; o
# nome deste arquivo está em shell/mare/Wallpaper.qml, `migration`.)
set -euo pipefail
cfg=${XDG_CONFIG_HOME:-$HOME/.config}/mk

mkdir -p "$cfg"
[[ -e $cfg/.papel-de-fabrica ]] || echo "instalado antes do papel de parede de fábrica" >"$cfg/.papel-de-fabrica"
