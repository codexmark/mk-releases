#!/usr/bin/env bash
# config/hypr/hypridle.conf: os passos da ociosidade de fábrica, em ordem, e o
# de dormir passando pelo scripts/mare-sleep now (que segue os Ajustes).
. "$(dirname "$0")/../lib.sh"

conf=$REPO/config/hypr/hypridle.conf
# Os listeners, um por linha: "<timeout>|<on-timeout>|<on-resume>".
passos=$(awk '
  /^[[:space:]]*listener[[:space:]]*\{/ { in_l = 1; t = ""; on = ""; re = ""; next }
  in_l && /^[[:space:]]*\}/ { print t "|" on "|" re; in_l = 0; next }
  in_l && /^[[:space:]]*timeout[[:space:]]*=/ { sub(/^[^=]*=[[:space:]]*/, ""); t = $0 }
  in_l && /^[[:space:]]*on-timeout[[:space:]]*=/ { sub(/^[^=]*=[[:space:]]*/, ""); on = $0 }
  in_l && /^[[:space:]]*on-resume[[:space:]]*=/ { sub(/^[^=]*=[[:space:]]*/, ""); re = $0 }
' "$conf")
campo() { sed -n "s/^[[:space:]]*$1[[:space:]]*=[[:space:]]*//p" "$conf"; }
passo() { grep -F -- "$1" <<<"$passos" | head -1; }

t "formato: chaves fechadas, só as seções general e listener"
eq "$(grep -c '^[a-z-]*[[:space:]]*{[[:space:]]*$' "$conf")" "$(grep -c '^[[:space:]]*}[[:space:]]*$' "$conf")"
eq $'general\nlistener' "$(sed -n 's/^\([a-z-]*\)[[:space:]]*{.*/\1/p' "$conf" | sort -u)"
eq 1 "$(grep -c '^general' "$conf")"

t "cinco passos, cada um com tempo e ação, em ordem crescente"
eq 5 "$(wc -l <<<"$passos")"
while IFS='|' read -r tempo acao _; do
  like "$tempo" '^[0-9]+$' "tempo"
  [[ -n $acao ]] || _fail "passo de $tempo s sem on-timeout"
done <<<"$passos"
tempos=$(cut -d'|' -f1 <<<"$passos")
eq "$(sort -n <<<"$tempos")" "$tempos"

t "os tempos são os do cabeçalho: descanso de tela aos 3 min, escurece aos 4, bloqueia aos 5, apaga aos 30"
has "$(head -4 "$conf")" "descanso de tela aos 3 min, escurece aos 4, bloqueia aos 5"
has "$(head -4 "$conf")" "desliga a tela aos 30"
eq "180" "$(passo 'ipc call mare saver on' | cut -d'|' -f1)"
eq "240" "$(passo 'brightnessctl -s set' | cut -d'|' -f1)"
eq "300" "$(passo 'loginctl lock-session' | cut -d'|' -f1)"
eq "1800" "$(passo 'action = "off"' | cut -d'|' -f1)"
eq "1800" "$(passo 'mare-sleep' | cut -d'|' -f1)"

t "o descanso de tela liga pelo Maré e desliga quando alguém volta"
eq "180|qs -p ~/.config/mk/shell ipc call mare saver on|qs -p ~/.config/mk/shell ipc call mare saver off" "$(passo 'saver on')"

t "o passo de dormir é o último e chama o scripts/mare-sleep now da instalação"
eq "1800|~/.config/mk/shell/scripts/mare-sleep now|" "$(tail -1 <<<"$passos")"

t "nenhum passo suspende por conta própria (systemctl suspend/hibernate): quem decide são os Ajustes"
hasnt "$(grep -v '^[[:space:]]*#' "$conf")" "systemctl"
hasnt "$(grep -v '^[[:space:]]*#' "$conf")" "suspend"
hasnt "$(grep -v '^[[:space:]]*#' "$conf")" "hibernate"

t "o mare-sleep existe, é executável (o hypridle o chama direto) e entende 'now'"
script=$REPO/shell/mare/scripts/mare-sleep
exists "$script"
[[ -x $script ]] || _fail "sem o bit de execução: $script"
has "$(grep -v '^#' "$script")" '$action == now'

t "o passo de dormir, rodado como o hypridle roda (sh -c), não dorme quando os Ajustes não dizem outra coisa"
mkdir -p "$HOME/.config/mk/shell/scripts" "$SBX/sys/power" "$SBX/sys/bus/pci/devices" "$SBX/proc"
cp "$script" "$HOME/.config/mk/shell/scripts/mare-sleep"
echo "freeze mem disk" >"$SBX/sys/power/state"
stub systemctl
dormir=$(tail -1 <<<"$passos" | cut -d'|' -f2)
ok env SYS="$SBX/sys" PROC="$SBX/proc" sh -c "$dormir"
not_called "systemctl"

t "o passo de dormir suspende para quem escolheu 'Ao dormir: suspender' nos Ajustes"
calls_reset
echo '{"power.sleep":"suspender"}' >"$HOME/.config/mk/ajustes.json"
ok env SYS="$SBX/sys" PROC="$SBX/proc" sh -c "$dormir"
called "systemctl suspend"

t "bloqueio: a tela do Maré primeiro; sem o Maré, o hyprlock com a config do Mk"
lock=$(campo lock_cmd)
like "$lock" '^qs -p ~/\.config/mk/shell ipc call mare lock \|\| '
has "$lock" "hyprlock -c ~/.config/mk/hypr/hyprlock.conf"
has "$lock" "pidof hyprlock ||" "não abre dois hyprlock"
exists "$REPO/config/hypr/hyprlock.conf"

t "antes de dormir a sessão é bloqueada, e ao voltar a tela acende"
eq "loginctl lock-session" "$(campo before_sleep_cmd)"
has "$(campo after_sleep_cmd)" 'action = "on"'

t "quem escurece devolve o brilho, e quem apaga a tela acende ao voltar"
eq "240|brightnessctl -s set 15%|brightnessctl -r" "$(passo 'brightnessctl -s set')"
has "$(passo 'action = "off"' | cut -d'|' -f3)" 'action = "on"'

t "os despachos do hyprctl são Lua que o Hyprland 0.57 entende (hl.dsp.dpms)"
command -v lua >/dev/null || skip "sem lua"
n=0
while IFS= read -r codigo; do
  n=$((n + 1))
  ok env CODIGO="$codigo" lua -e 'package.path = os.getenv("TESTS") .. "/config/?.lua;" .. package.path
    hl = require("hl").new()
    local d = assert(load("return " .. os.getenv("CODIGO")))()
    assert(d.dsp == "dpms" and (d.args[1].action == "on" or d.args[1].action == "off"))'
done < <(grep -v '^[[:space:]]*#' "$conf" | sed -n "s/.*hyprctl dispatch '\(.*\)'[[:space:]]*$/\1/p")
eq 3 "$n" "despachos conferidos"
hasnt "$(grep -v '^[[:space:]]*#' "$conf")" "dispatch dpms" "sintaxe antiga"

t "só chama programas que vêm nas listas de pacotes do Mk"
pkgs=$(sed -e 's/#.*//' "$REPO"/system/packages/{base,desktop}.packages)
for p in brightnessctl hyprlock hypridle quickshell hyprland; do
  ok grep -qx "$p" <<<"$pkgs"
done
