#!/usr/bin/env bash
# Os modos do Maré (Modes.qml + Modes.js): apresentação e jogo. A conta do
# "estado de antes" é testada no node (tests/js/Modes.test.mjs); aqui: as
# ligações. O singleton, o IPC, os atalhos, a linha nos Controles, o sinal na
# ilha e o caminho até o Hyprland.
. "$(dirname "$0")/../lib.sh"

mare=$REPO/shell/mare
src=$(cat "$mare/Modes.qml" 2>/dev/null)
lua=$(<"$REPO/config/hypr/mare/binds.lua")
kb=$(<"$mare/Keybinds.js")
ipc=$(sed -n '/IpcHandler {/,$p' "$mare/Mare.qml")

t "Modes é um singleton, e a conta fica no Modes.js (biblioteca pura)"
has "$src" "pragma Singleton"
has "$src" 'import "Modes.js" as Logic'
has "$(head -n 1 "$mare/Modes.js" 2>/dev/null)" ".pragma library"
for f in toggle leave reconcile hyprOff hyprBack optionOn; do
  has "$src" "Logic.$f(" "o QML não usa a função pura"
done

t "o modo mexe nos mesmos interruptores de sempre: Notifs.dnd, a cafeína e o perfil de energia"
has "$src" "Notifs.dnd = "
has "$src" "Caffeine.on = "
has "$src" "PowerProfiles.profile = "
# Nada de inibidor próprio: é o da cafeína.
hasnt "$src" "systemd-inhibit"

t "desempenho só se o power-profiles-daemon oferecer"
has "$src" "PowerProfiles.hasPerformanceProfile ? PowerProfile.Performance : null"

t "o que a pessoa mexe à mão durante o modo é avisado à conta (reconcile)"
has "$src" "function onDndChanged()"
has "$src" "function onOnChanged()"
has "$src" "function onProfileChanged()"

t "não é guardado: sessão nova começa desligado"
hasnt "$src" "Settings.set("
hasnt "$src" "FileView"
hasnt "$(<"$mare/SettingsSchema.js")" '"modes.'

t "modo jogo: o Lua vai ao Hyprland por 'hyprctl eval', como o dos Ajustes, e fica nos evals dos Ajustes enquanto durar"
has "$src" '"hyprctl", "eval"'
has "$src" "Settings.overlay = "
set_qml=$(<"$mare/Settings.qml")
has "$set_qml" 'property string overlay: ""'
has "$set_qml" 'Schema.hyprCode(root.values, root.touched) + root.overlay'
# O ajustes.lua (o que vale no próximo login) não leva o modo.
has "$set_qml" 'root.flightLua = Schema.hyprLua(root.values);'

t "modo jogo: na saída, as animações voltam ao que os Ajustes dizem"
has "$src" 'Settings.get("motion.speed") > 0'

t "modo jogo: desfoque e sombra são lidos antes de desligar (para voltar só o que estava ligado)"
has "$src" '"hyprctl", "-j", "getoption", "decoration:blur:enabled"'
has "$src" '"hyprctl", "-j", "getoption", "decoration:shadow:enabled"'

t "IPC: present e game alternam e devolvem o estado, como o dnd"
like "$ipc" 'function present\(\): bool \{[[:space:]]+return Modes\.toggle\("present"\);'
like "$ipc" 'function game\(\): bool \{[[:space:]]+return Modes\.toggle\("game"\);'

t "atalhos: Super + Shift + P e Super + Shift + G, nas duas tabelas"
has "$lua" '{ id = "mare.present", group = "MARÉ", desc = "modo apresentação", combo = "SUPER + SHIFT + P",'
has "$lua" 'run = shell("present") },'
has "$lua" '{ id = "mare.game", group = "MARÉ", desc = "modo jogo", combo = "SUPER + SHIFT + G",'
has "$lua" 'run = shell("game") },'
has "$kb" '{ id: "mare.present", group: "MARÉ", desc: "modo apresentação", combo: "SUPER + SHIFT + P" },'
has "$kb" '{ id: "mare.game", group: "MARÉ", desc: "modo jogo", combo: "SUPER + SHIFT + G" },'
ok "$REPO/bin/check-keybinds" >/dev/null

t "atalhos: nenhum outro de fábrica usa os dois combos"
eq "1" "$(grep -c 'combo = "SUPER + SHIFT + P"' "$REPO/config/hypr/mare/binds.lua")"
eq "1" "$(grep -c 'combo = "SUPER + SHIFT + G"' "$REPO/config/hypr/mare/binds.lua")"

t "Controles: uma linha com os dois modos, que liga e desliga em Ajustes › Ilha"
painel=$(<"$mare/ControlsPanel.qml")
has "$painel" 'Settings.get("controls.showModes")'
has "$painel" 'Modes.toggle("present")'
has "$painel" 'Modes.toggle("game")'
has "$painel" "selected: Modes.present"
has "$painel" "selected: Modes.game"
has "$(<"$mare/SettingsSchema.js")" 'key: "controls.showModes"'

t "ilha: um sinal em cobre enquanto um modo está ligado; um clique desliga"
bar=$(<"$mare/Bar.qml")
has "$bar" "visible: Modes.on"
has "$bar" "onClicked: Modes.off()"

t "cores: só as do tema (nenhuma escrita à mão)"
eq "" "$(grep -oE '#[0-9a-fA-F]{6}\b' "$mare/Modes.qml" 2>/dev/null | sort -u | tr '\n' ' ')"
