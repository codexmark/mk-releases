#!/usr/bin/env bash
# As boas-vindas (a folha do primeiro login) chegaram depois: quem já usava o
# Mk não é novato. Marca como vistas, para a folha não subir sozinha. Dá para
# abri-la quando quiser: busca "boas-vindas", Ajustes › Sobre ou `mk boas-vindas`.
# (O Maré só abre a folha sozinho depois que esta migração consta como feita;
# o nome deste arquivo está em shell/mare/Welcome.qml, `migration`.)
set -euo pipefail
cfg=${XDG_CONFIG_HOME:-$HOME/.config}/mk

mkdir -p "$cfg"
[[ -e $cfg/.bem-vindo ]] || echo "instalado antes das boas-vindas" >"$cfg/.bem-vindo"
