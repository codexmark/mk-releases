#!/usr/bin/env bash
# bin/mk-session: só inicia o Hyprland (pelo uwsm) se a config do Mk existe.
. "$(dirname "$0")/../lib.sh"

sessao=$REPO/bin/mk-session
config=$XDG_CONFIG_HOME/mk/hypr/hyprland.lua

t "sem ~/.config/mk/hypr/hyprland.lua: avisa qual arquivo falta, sai com 1 e não inicia nada"
stub uwsm
out=$(bash "$sessao" 2>&1)
eq 1 $?
has "$out" "$config não existe"
has "$out" "bin/try-session install"
not_called "uwsm"

t "com a config: passa a vez ao uwsm, com o Hyprland e a config do Mk"
mkdir -p "$(dirname "$config")"
echo "-- config" >"$config"
bash "$sessao"
eq 0 $?
eq "uwsm start -e -D Hyprland -N mk -- start-hyprland -- -c $config" "$(cat "$SBX/calls")"

t "o código de saída é o do uwsm"
stub uwsm 'exit 7'
bash "$sessao"
eq 7 $?

t "sem XDG_CONFIG_HOME, a config é a de ~/.config"
calls_reset
stub uwsm
mkdir -p "$HOME/.config/mk/hypr"
echo "-- config" >"$HOME/.config/mk/hypr/hyprland.lua"
(unset XDG_CONFIG_HOME && bash "$sessao")
called "-c $HOME/.config/mk/hypr/hyprland.lua"

t "uma pasta com o nome da config não conta como config"
calls_reset
rm -f "$config"
mkdir -p "$config"
bash "$sessao" 2>/dev/null
eq 1 $?
not_called "uwsm"
