#!/usr/bin/env bash
# O descanso de tela do Maré (ScreenSaver.qml): o que a tela mostra não dá
# para testar aqui (vê-se no bin/dev-shot); o que dá: ele existe em cada
# monitor, liga e desliga por IPC, sai com qualquer tecla ou com o mouse, não
# passa por cima do bloqueio, só usa as cores do Mk e desenha a marca com a
# mesma geometria da tela de entrada.
. "$(dirname "$0")/../lib.sh"

mare=$REPO/shell/mare
saver=$mare/ScreenSaver.qml
src=$(cat "$saver" 2>/dev/null)

t "existe, e cada monitor tem o seu"
exists "$saver"
has "$(<"$mare/Monitor.qml")" "ScreenSaver {"
like "$(sed -n '/ScreenSaver {/,/}/p' "$mare/Monitor.qml")" 'screen: scope\.modelData'

t "liga, desliga e alterna por IPC: saver on | off | toggle"
ipc=$(sed -n '/function saver(state: string)/,/^        }/p' "$mare/Mare.qml")
has "$ipc" '"on"'
has "$ipc" '"off"'
has "$ipc" '"toggle"'
has "$(<"$mare/Mare.qml")" "property bool saver"

t "o estado (state) diz quando o descanso está na tela"
has "$(sed -n '/function state(): string/,/^        }/p' "$mare/Mare.qml")" "saver"

t "fica por cima de tudo, na tela inteira, e pega o teclado enquanto está ligado"
has "$src" "WlrLayershell.layer: WlrLayer.Overlay"
has "$src" 'WlrLayershell.namespace: "mare-saver"'
has "$src" "WlrKeyboardFocus.Exclusive"
for lado in top bottom left right; do like "$src" "$lado: true"; done

t "qualquer tecla, clique ou movimento do mouse devolve a tela"
has "$src" "Keys.onPressed"
has "$src" "onPositionChanged"
has "$src" "onPressed"
like "$src" 'Mare\.saver = false'

t "o mouse parado onde estava não desliga: só um movimento de verdade"
# Ao aparecer, a janela recebe a posição do ponteiro sem ele ter se mexido.
like "$src" 'Math\.abs\(.*\) > [0-9]+'

t "não aparece por cima do bloqueio nem com a tela bloqueada"
like "$src" 'visible: Mare\.saver && !Lock\.locked'

t "bloquear a tela desliga o descanso (o bloqueio tem o desenho dele)"
has "$(sed -n '/function lock(): void/,/^        }/p' "$mare/Mare.qml")" "root.saver = false"

t "cores: só as do tema (nenhuma cor escrita à mão, fora osso e cobre em rgba)"
eq "" "$(grep -oE '#[0-9a-fA-F]{6}\b' "$saver" | sort -u | tr '\n' ' ')"
out=$(grep -oE 'Qt\.rgba\([^)]*\)' "$saver" | grep -vE 'Qt\.rgba\((205 / 255, 196 / 255, 186 / 255|206 / 255, 94 / 255, 50 / 255),' || true)
eq "" "$out"

t "a marca é desenhada com a geometria da tela de entrada (o mesmo arquivo, gerado)"
ok cmp -s "$mare/assets/geometria.js" "$REPO/system/mk/sddm/mk/geometria.js"
ok cmp -s "$mare/assets/letra-m.png" "$REPO/system/mk/sddm/mk/letra-m.png"
ok cmp -s "$mare/assets/letra-k.png" "$REPO/system/mk/sddm/mk/letra-k.png"
has "$src" 'import "assets/geometria.js" as Geo'

t "a marca muda de lugar a cada ciclo, para não marcar a tela"
has "$src" "property int cycle"
like "$src" 'cycle'

t "o movimento respeita Ajustes › Movimento (Motion), como o resto do Maré"
has "$src" "Motion."

t "o hypridle de fábrica liga o descanso aos 3 min e desliga quando alguém volta"
conf=$(<"$REPO/config/hypr/hypridle.conf")
bloco=$(grep -B2 -A2 'ipc call mare saver on' <<<"$conf")
has "$bloco" "timeout = 180"
has "$bloco" "on-resume = qs -p ~/.config/mk/shell ipc call mare saver off"

t "cobre a ilha também (a zona exclusiva que o resto do Maré usa para a tela inteira)"
has "$src" "exclusiveZone: Mare.ignoreZone"
hasnt "$src" "ExclusionMode.Ignore"
