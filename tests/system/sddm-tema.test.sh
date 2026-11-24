#!/usr/bin/env bash
# system/mk/sddm/mk: a tela de entrada. O desenho não dá para testar aqui (é
# visto no greeter em modo de teste e na VM); o que dá: o tema é uma pasta que
# se basta, só usa as cores do Mk, não perde o que faz o login funcionar, tem
# como desligar o movimento, e a geometria da marca bate com o gerador do logo.
. "$(dirname "$0")/../lib.sh"

tema=$REPO/system/mk/sddm/mk
main=$tema/Main.qml
geo=$tema/geometria.js

t "o tema se basta: todo arquivo que o Main.qml carrega está na pasta"
while read -r f; do
  exists "$tema/$f"
done < <(grep -oE '(source: |import )"[^"]+"' "$main" | sed -E 's/.*"([^"]+)"/\1/' | sort -u)

t "só importa o QtQuick, os componentes do SDDM e arquivos da própria pasta"
out=$(grep -E '^import ' "$main" | grep -vE '^import (QtQuick( [0-9.]+)?|SddmComponents 2\.0|"[A-Za-z0-9_.-]+\.js" as [A-Z][A-Za-z]*)$' || true)
eq "" "$out"

t "metadata: tema do SDDM para Qt 6, com Main.qml e theme.conf"
meta=$(<"$tema/metadata.desktop")
has "$meta" "MainScript=Main.qml"
has "$meta" "ConfigFile=theme.conf"
has "$meta" "QtVersion=6"

t "cores: só os tokens do Mk (fundo, osso e os três cinzas dele, cobre)"
tokens=" #0b0b0b #cdc4ba #b0a9a0 #958f87 #716b69 #ce5e32 "
while read -r c; do
  [[ $tokens == *" ${c,,} "* ]] || _fail "cor fora da paleta: $c"
done < <(grep -oE '#[0-9a-fA-F]{6}\b' "$main" | sort -u)

t "cores em rgba: só o osso (205,196,186) e o cobre (206,94,50), com transparência"
out=$(grep -oE 'Qt\.rgba\([^)]*\)' "$main" | grep -vE 'Qt\.rgba\((205 / 255, 196 / 255, 186 / 255|206 / 255, 94 / 255, 50 / 255),' || true)
eq "" "$out"

t "nenhuma cor com nome (red, white...) nem transparência por cor inventada"
out=$(grep -nE 'color: "(red|green|blue|white|black|gray|grey|yellow|orange)"' "$main" || true)
eq "" "$out"

t "o login continua inteiro: usuário, senha escondida, sessão, falha e as ações de energia"
src=$(<"$main")
has "$src" "sddm.login(user.name, password.text, sessionIndex)"
has "$src" "function onLoginFailed()"
has "$src" "function onLoginSucceeded()"
has "$src" "echoMode: TextInput.Password"
has "$src" "password.forceActiveFocus()"
has "$src" "keyboard.capsLock"
has "$src" "sddm.reboot()"
has "$src" "sddm.powerOff()"
has "$src" "userModel"
has "$src" "sessionModel"

t "senha errada: limpa o campo, devolve o foco e diz que errou"
bloco=$(sed -n '/function onLoginFailed()/,/^        }/p' "$main")
has "$bloco" 'password.text = ""'
has "$bloco" "password.forceActiveFocus()"
has "$bloco" "root.failed = true"
has "$src" "senha errada"

t "o movimento tem interruptor: motion= no theme.conf, lido pelo tema"
has "$(<"$tema/theme.conf")" "motion="
has "$src" "config.motion"

t "nenhuma animação roda sem passar pelo interruptor"
# Toda animação em laço (a que gasta máquina parada) depende de root.motion.
n=0
while IFS= read -r linha; do
  n=$((n + 1))
  bloco=$(sed -n "$((linha)),$((linha + 4))p" "$main")
  [[ $bloco == *"root.motion"* ]] || _fail "laço sem root.motion na linha $linha"
done < <(grep -nE 'loops: Animation\.Infinite|repeat: true' "$main" | grep -v 'clockTick' | cut -d: -f1)
ok test "$n" -ge 1

t "as letras da marca: M e k em camadas separadas, do mesmo tamanho, com transparência"
tamanho() { python3 -I -c '
import struct, sys
d = open(sys.argv[1], "rb").read(26)
assert d[:8] == b"\x89PNG\r\n\x1a\n", "não é PNG"
w, h, bits, tipo = struct.unpack(">IIBB", d[16:26])
print(w, h, tipo)' "$1"; }
m=$(tamanho "$tema/letra-m.png")
k=$(tamanho "$tema/letra-k.png")
eq "$m" "$k"
like "$m" ' 6$' "PNG com canal alfa (tipo 6)"

t "geometria.js: gerado pelo make-logo.py, com os 8 nós, as 4 diagonais e o quadro"
exists "$geo"
g=$(<"$geo")
has "$g" "GERADO"
has "$g" "make-logo.py"
node=$(command -v node) || node=""
if [[ -n $node ]]; then
  json=$(grep -v '^\.pragma' "$geo" | "$node" -e '
    let s = ""; process.stdin.on("data", d => s += d).on("end", () => {
      const G = new Function(s + "; return { vb, heights, verts, diags, nodes, ax, ay, top, right, kx };")();
      console.log(JSON.stringify(G));
    });')
  eq 8 "$(jq '.nodes | length' <<<"$json")"
  eq 4 "$(jq '.diags | length' <<<"$json")"
  eq 5 "$(jq '.heights | length' <<<"$json")"
  eq 4 "$(jq '.vb | length' <<<"$json")"
  # Os nós do M são osso (0) e os do k cobre (1): três e cinco.
  eq 3 "$(jq '[.nodes[] | select(.[2] == 0)] | length' <<<"$json")"
  eq 5 "$(jq '[.nodes[] | select(.[2] == 1)] | length' <<<"$json")"

  t "geometria.js: a proporção do quadro é a das imagens das letras"
  read -r w h _ <<<"$m"
  prop=$(jq -r '.vb[2] / .vb[3]' <<<"$json")
  ok python3 -I -c "import sys; a, b = float(sys.argv[1]), $w / $h; sys.exit(0 if abs(a - b) / b < 0.005 else 1)" "$prop"

  t "geometria.js: os números são os do make-logo.py (alturas, diagonais, eixos)"
  py=$REPO/system/mk/art/make-logo.py
  eq "$(grep -oE '^HEIGHTS = \[[^]]*\]' "$py" | grep -oE '[0-9]+' | tr '\n' ' ')" "$(jq -r '.heights | join(" ")' <<<"$json") "
  eq "$(grep -oE '^AX, AY = -?[0-9]+, -?[0-9]+' "$py" | grep -oE -- '-?[0-9]+' | tr '\n' ' ')" "$(jq -r '[.ax, .ay] | join(" ")' <<<"$json") "
else
  echo "  # node ausente: a leitura do geometria.js foi pulada" >&2
fi

t "todo nó e toda diagonal cabem no quadro"
if [[ -n $node ]]; then
  fora=$(jq '[.nodes[] | select(.[0] < $v[0] or .[0] > $v[0] + $v[2] or -.[1] < $v[1] or -.[1] > $v[1] + $v[3])] | length' --argjson v "$(jq -c .vb <<<"$json")" <<<"$json")
  eq 0 "$fora"
fi

t "toda chave do theme.conf é lida pelo tema, e todo arquivo da pasta é usado"
while read -r chave; do
  grep -q "config\.$chave\b" "$main" || _fail "o theme.conf tem '$chave', que o Main.qml não lê"
done < <(grep -oE '^[a-zA-Z]+=' "$tema/theme.conf" | tr -d '=')
for f in "$tema"/*; do
  n=$(basename "$f")
  case $n in Main.qml | theme.conf | metadata.desktop) continue ;; esac
  grep -qF "\"$n\"" "$main" || _fail "arquivo sem uso na pasta do tema: $n"
done

# --- o relógio de anéis (o mesmo desenho do bloqueio) ---------------------------
lock=$REPO/shell/mare/LockSurface.qml

t "o relógio é o de anéis: dois anéis de 60 marcas, a pílula e a hora em serifa"
has "$src" "component Ring: Item"
eq 2 "$(grep -cE '^\s+Ring \{' "$main")"
has "$src" "id: minuteRing"
has "$src" "id: secondRing"
has "$src" "model: 60"
has "$src" "id: pill"
has "$src" 'String(root.curH).padStart(2, "0")'

t "o relógio grande em serifa (HH:mm) saiu"
hasnt "$src" 'Qt.formatTime(now, "HH:mm")'
hasnt "$src" "root.height * 0.2"

t "os anéis seguem a mesma conta do bloqueio (onde o número acende, as marcas de 5 em 5)"
for conta in 'Math.max(0, 1 - Math.abs(rel) / 4)' 'index % 5 === 0' '(base + ring.angle) % 360' \
  'secTarget = -((d.getSeconds() * 1000 + ms) / 60000) * 360' \
  'minTarget = -((d.getMinutes() * 60000 + d.getSeconds() * 1000 + ms) / 3600000) * 360'; do
  has "$(<"$lock")" "$conta" "o bloqueio mudou a conta"
  has "$src" "$conta" "a entrada não tem a conta do bloqueio"
done

t "com o movimento desligado os anéis não giram entre um segundo e outro"
bloco=$(sed -n '/Behavior on rotation {/,/}/p' "$main")
has "$bloco" "enabled: root.motion"

t "quem entra e a senha continuam na tela, fora do caminho dos anéis"
has "$src" "id: who"
has "$src" "id: field"
# O painel fica do lado da marca: a esquerda inteira é do relógio.
like "$(sed -n '/id: panel/,/spacing:/p' "$main")" 'x: root\.planeX'

