#!/usr/bin/env bash
# O painel Controles do Maré (ControlsPanel.qml), aberto por Super + Esc: tudo
# de um relance num lugar só. O desenho se vê no bin/dev-shot; aqui: o atalho,
# os blocos e os ajustes deles, a cafeína e o reaproveitamento do que os
# outros painéis já fazem.
. "$(dirname "$0")/../lib.sh"

mare=$REPO/shell/mare
src=$(cat "$mare/ControlsPanel.qml" 2>/dev/null)
lua=$(<"$REPO/config/hypr/mare/binds.lua")

t "Super + Esc abre os Controles; o painel Sessão passa para Super + Ctrl + Esc"
has "$lua" '{ id = "mare.controls", group = "MARÉ", desc = "controles rápidos", combo = "SUPER + ESCAPE",'
has "$lua" 'run = shell("toggle controls") },'
has "$lua" '{ id = "mare.session", group = "MARÉ", desc = "sair, bloquear, desligar", combo = "SUPER + CTRL + ESCAPE",'
ok "$REPO/bin/check-keybinds" >/dev/null

t "os blocos: leituras, Wi-Fi e Bluetooth, perfil de energia, volume e brilho, sessão e alternadores"
for peca in "Stats.cpu" "Networking.wifiEnabled" "Bluetooth.defaultAdapter" "PowerProfiles.profile" \
  "Pipewire.defaultAudioSink" "Display.brightness" "SessionActions.actions(Settings.sleepMode)" \
  "Display.toggleNight()" "Notifs.dnd" "Caffeine.on" "Pipewire.defaultAudioSource"; do
  has "$src" "$peca" "falta no painel"
done

t "os modos (apresentação e jogo) são do singleton Modes: o painel só alterna (veja modos.test.sh)"
has "$src" 'Modes.toggle("present")'
has "$src" 'Modes.toggle("game")'
hasnt "$src" "systemd-inhibit"
hasnt "$src" "hyprctl"

t "cada bloco some quando a máquina não tem (brilho, bateria, Bluetooth, perfis de energia)"
has "$src" "Display.hasBacklight"
has "$src" "UPower.displayDevice"
ok grep -q 'visible: panel.adapter !== null' "$mare/ControlsPanel.qml"

t "os blocos ligam e desligam em Ajustes › Ilha, e o painel obedece"
for chave in controls.showStats controls.showRadios controls.showPower controls.showModes controls.showLevels; do
  has "$(<"$mare/SettingsSchema.js")" "key: \"$chave\""
  has "$src" "Settings.get(\"$chave\")"
done

t "um botão leva ao MkPulso e outro aos Ajustes"
has "$src" 'Hypr.exec("mkpulso")'
has "$src" 'Mare.mode = "settings"'

t "sair, reiniciar e desligar pedem o segundo clique, como no painel Sessão (a mesma lista)"
has "$src" "armed"
has "$(<"$mare/SessionPanel.qml")" "SessionActions.actions(Settings.sleepMode)"

t "cafeína: segura a ociosidade (descanso de tela, bloqueio, tela) enquanto ligada, e solta ao desligar"
caf=$(cat "$mare/Caffeine.qml" 2>/dev/null)
has "$caf" "pragma Singleton"
has "$caf" '"systemd-inhibit", "--what=idle"'
has "$caf" "running: root.on"
# O hypridle do Mk respeita o inibidor do systemd (não pode estar mandado ignorar).
hasnt "$(<"$REPO/config/hypr/hypridle.conf")" "ignore_systemd_inhibit"
hasnt "$(<"$mare/SettingsSchema.js")" "ignore_systemd_inhibit"

t "cores: só as do tema (nenhuma escrita à mão)"
eq "" "$(grep -oE '#[0-9a-fA-F]{6}\b' "$mare/ControlsPanel.qml" "$mare/Caffeine.qml" 2>/dev/null | sort -u | tr '\n' ' ')"
