#!/usr/bin/env bash
# migrations/user/20261128-papel-de-fabrica.sh: quem já usava o Mk não ganha
# um papel de parede sozinho; só uma instalação nova nasce com o de fábrica.
. "$(dirname "$0")/../../lib.sh"
. "$TESTS/migrations/mig.sh" user/20261128-papel-de-fabrica.sh

marca=$XDG_CONFIG_HOME/mk/.papel-de-fabrica

t "home sem a marca: fica marcado que o papel de fábrica não é para esta pessoa"
ok mig
exists "$marca"

t "é a marca e a migração que o Maré lê (shell/mare/Wallpaper.qml)"
w=$(cat "$REPO/shell/mare/Wallpaper.qml")
has "$w" ".papel-de-fabrica"
has "$w" '"20261128-papel-de-fabrica.sh"'

de_novo

t "uma marca que já existe não é reescrita"
echo "aplicado em 2026-11-23" >"$marca"
sem_mudanca mig
eq "aplicado em 2026-11-23" "$(cat "$marca")"

t "sem ~/.config/mk: cria a pasta e a marca"
rm -rf "$XDG_CONFIG_HOME"
ok mig
exists "$marca"

t "não mexe no papel de parede que a pessoa já escolheu, nem sai da home"
eq "" "$(find "$R" -mindepth 1)"
eq "" "$(cat "$SBX/calls")"
