#!/usr/bin/env bash
# shell/mare/scripts/mare-caps: o que a máquina tem (touchpad, bateria,
# brilho, sono), com /proc e /sys falsos.
. "$(dirname "$0")/../lib.sh"

caps_sh=$REPO/shell/mare/scripts/mare-caps
sys=$SBX/sys proc=$SBX/proc
dev=$proc/bus/input/devices
mkdir -p "$sys/power" "$sys/class/power_supply" "$sys/class/backlight" "$proc/bus/input"
echo "freeze mem disk" >"$sys/power/state"

run() { SYS=$sys PROC=$proc bash "$caps_sh"; }
tem() { run | jq -c ".$1"; }
# Um aparelho como o kernel lista em /proc/bus/input/devices.
# aparelho <nome> [PROP] [ABS]   (sem ABS, a linha não existe: um mouse comum)
aparelho() {
  printf 'I: Bus=0018 Vendor=04f3 Product=3134 Version=0100\nN: Name="%s"\nP: Phys=i2c\nS: Sysfs=/devices/x\nU: Uniq=\nH: Handlers=mouse0 event5\n' "$1"
  [[ -z ${2:-} ]] || printf 'B: PROP=%s\n' "$2"
  printf 'B: EV=b\nB: KEY=e520 10000 0 0 0 0\n'
  [[ -z ${3:-} ]] || printf 'B: ABS=%s\n' "$3"
  printf '\n'
}
teclado() { aparelho "AT Translated Set 2 keyboard" 0; }
mouse() { aparelho "Logitech USB Optical Mouse" 0; }

t "a saída é uma linha de JSON com as quatro chaves"
teclado >"$dev"
out=$(run)
eq 1 "$(wc -l <<<"$out")"
eq '["bateria","brilho","sono","touchpad"]' "$(jq -cS 'keys' <<<"$out")"

t "desktop sem nada: sem touchpad, sem bateria, sem brilho"
{ teclado; mouse; } >"$dev"
eq '{"touchpad":false,"bateria":false,"brilho":false,"sono":{"canSuspend":true,"risky":[]}}' "$(run)"

t "touchpad: achado pelo nome (touchpad, trackpad, synaptics, glidepoint), sem olhar maiúsculas"
for nome in "ELAN0676:00 04F3:3195 Touchpad" "Apple Internal Keyboard / Trackpad" "SynPS/2 Synaptics TouchPad" "AlpsPS/2 ALPS GlidePoint" "DELL TOUCHPAD"; do
  { teclado; aparelho "$nome"; } >"$dev"
  eq true "$(tem touchpad)" "$nome"
done

t "touchpad: sem nome que ajude, vale um ponteiro (PROP bit 0) com eixos absolutos"
for prop in 1 5 9 d D 101 0000000000000005; do
  { teclado; aparelho "PNP0C50:00 2808:0101" "$prop" "2608000 3"; } >"$dev"
  eq true "$(tem touchpad)" "PROP=$prop"
done

t "touchpad: tela de toque (PROP bit 1) não é touchpad, nem com o bit 0 junto"
for prop in 2 3 6 7 a b e f A F 12; do
  { teclado; aparelho "Wacom HID 5256 Finger" "$prop" "2608000 3"; } >"$dev"
  eq false "$(tem touchpad)" "PROP=$prop"
done

t "touchpad: PROP sem o bit 0 (0, 4, 8, c) não é ponteiro"
for prop in 0 4 8 c 10; do
  { teclado; aparelho "Aparelho qualquer" "$prop" "2608000 3"; } >"$dev"
  eq false "$(tem touchpad)" "PROP=$prop"
done

t "touchpad: um ponteiro sem eixos absolutos (sem ABS, ou ABS=0) não conta"
{ teclado; aparelho "TPPS/2 IBM TrackPoint" 1; } >"$dev"
eq false "$(tem touchpad)"
{ teclado; aparelho "Aparelho estranho" 1 0; } >"$dev"
eq false "$(tem touchpad)"
{ teclado; aparelho "Aparelho estranho" 1 "0000"; } >"$dev"
eq false "$(tem touchpad)"

t "touchpad: o PROP de um aparelho e o ABS de outro não se somam"
{ aparelho "Ponteiro sem eixos" 1; aparelho "Joystick" 0 "3001f"; } >"$dev"
eq false "$(tem touchpad)"

t "touchpad: vale em qualquer posição da lista (primeiro, no meio, último)"
{ aparelho "Coisa" 1 "2608000 3"; teclado; mouse; } >"$dev"
eq true "$(tem touchpad)" "primeiro"
{ teclado; aparelho "Coisa" 1 "2608000 3"; mouse; } >"$dev"
eq true "$(tem touchpad)" "no meio"
{ teclado; mouse; aparelho "Coisa" 1 "2608000 3"; } | sed '$d' >"$dev"
eq true "$(tem touchpad)" "último, sem linha em branco no fim"

t "touchpad: lista de aparelhos vazia, não tem"
: >"$dev"
eq false "$(tem touchpad)"

t "touchpad: sem conseguir ler a lista, não esconde os ajustes (verdadeiro)"
rm -f "$dev"
out=$(run)
ok jq -e . <<<"$out" >/dev/null
eq true "$(jq -c .touchpad <<<"$out")"
teclado >"$dev"

t "bateria: qualquer BAT* em power_supply; só a tomada (AC, ADP1) não conta"
mkdir -p "$sys/class/power_supply/AC" "$sys/class/power_supply/ADP1"
eq false "$(tem bateria)"
mkdir -p "$sys/class/power_supply/BAT1"
eq true "$(tem bateria)"
rmdir "$sys/class/power_supply/BAT1"
mkdir -p "$sys/class/power_supply/BATT"
eq true "$(tem bateria)"

t "brilho: qualquer coisa em /sys/class/backlight"
eq false "$(tem brilho)"
mkdir -p "$sys/class/backlight/intel_backlight"
eq true "$(tem brilho)"

t "notebook completo: tudo verdadeiro"
{ teclado; aparelho "ELAN Touchpad" 1 "2608000 3"; } >"$dev"
eq '[true,true,true]' "$(run | jq -c '[.touchpad, .bateria, .brilho]')"

t "sono: é a resposta do mare-sleep status, inteira"
mkdir -p "$sys/bus/pci/devices/0000:01:00.0"
echo 0x030000 >"$sys/bus/pci/devices/0000:01:00.0/class"
echo 0x1002 >"$sys/bus/pci/devices/0000:01:00.0/vendor"
echo 0x731f >"$sys/bus/pci/devices/0000:01:00.0/device"
eq "$(SYS=$sys PROC=$proc bash "$REPO/shell/mare/scripts/mare-sleep" status)" "$(tem sono)"
eq '["AMD Navi 10 (1002:731f)"]' "$(run | jq -c .sono.risky)"

t "sono: sem suspend no kernel, canSuspend falso chega aos Ajustes"
echo "disk" >"$sys/power/state"
eq false "$(tem sono.canSuspend)"
echo "freeze mem disk" >"$sys/power/state"

t "SYS e PROC com espaço no caminho funcionam"
mkdir -p "$SBX/com espaco"
cp -r "$sys" "$SBX/com espaco/sys"
cp -r "$proc" "$SBX/com espaco/proc"
eq '[true,true,true]' "$(SYS="$SBX/com espaco/sys" PROC="$SBX/com espaco/proc" bash "$caps_sh" | jq -c '[.touchpad, .bateria, .brilho]')"

# Os dois casos abaixo trocam o vizinho mare-sleep: rodam numa cópia.
copia=$SBX/copia/scripts
mkdir -p "$copia"
cp "$caps_sh" "$copia/mare-caps"

t "sono: sem resposta do mare-sleep, suspende como sempre e sem avisos"
eq '{"canSuspend":true,"risky":[]}' "$(SYS=$sys PROC=$proc bash "$copia/mare-caps" | jq -c .sono)"

t "sono: resposta do mare-sleep que não é um objeto não estraga o JSON"
printf '#!/usr/bin/env bash\necho "erro: algo deu errado"\n' >"$copia/mare-sleep"
out=$(SYS=$sys PROC=$proc bash "$copia/mare-caps")
ok jq -e . <<<"$out" >/dev/null
eq '{"canSuspend":true,"risky":[]}' "$(jq -c .sono <<<"$out")"
