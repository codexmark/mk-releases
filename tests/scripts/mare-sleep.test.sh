#!/usr/bin/env bash
# shell/mare/scripts/mare-sleep: o que a máquina pode (status) e o passo de
# dormir do hypridle (now), com /sys e /proc falsos.
. "$(dirname "$0")/../lib.sh"

sleep_sh=$REPO/shell/mare/scripts/mare-sleep
sys=$SBX/sys proc=$SBX/proc
mkdir -p "$sys/power" "$sys/bus/pci/devices" "$proc"
echo "freeze mem disk" >"$sys/power/state"

gpu() { # gpu <slot> <vendor> <device> [classe]
  mkdir -p "$sys/bus/pci/devices/$1"
  echo "${4:-0x030000}" >"$sys/bus/pci/devices/$1/class"
  echo "$2" >"$sys/bus/pci/devices/$1/vendor"
  echo "$3" >"$sys/bus/pci/devices/$1/device"
}
run() { SYS=$sys PROC=$proc AJUSTES=$SBX/ajustes.json bash "$sleep_sh" "$@"; }

t "status: suspende quando o kernel tem mem, e não lista hibernação"
out=$(run status)
eq '{"canSuspend":true,"risky":[]}' "$out"

t "status: sem mem nem freeze, não suspende"
echo "disk" >"$sys/power/state"
has "$(run status)" '"canSuspend":false'
echo "freeze mem disk" >"$sys/power/state"

t "status: a Navi 10 entra em risky; outra placa AMD e uma Intel, não"
gpu 0000:01:00.0 0x1002 0x731F
gpu 0000:02:00.0 0x1002 0x73bf
gpu 0000:00:02.0 0x8086 0x9a49
out=$(run status)
eq '["AMD Navi 10 (1002:731f)"]' "$(jq -c .risky <<<"$out")"

t "status: um dispositivo Navi 10 que não é vídeo (classe de áudio) não conta"
rm -rf "${sys:?}/bus/pci/devices"/*
gpu 0000:01:00.1 0x1002 0x731f 0x040300
eq '[]' "$(jq -c .risky <<<"$(run status)")"

t "now: suspender chama systemctl suspend"
stub systemctl
echo '{"power.sleep":"suspender"}' >"$SBX/ajustes.json"
run now
called "systemctl suspend"

t "now: sem ajustes.json, suspende"
calls_reset
rm -f "$SBX/ajustes.json"
run now
called "systemctl suspend"

t "now: 'nada' não chama nada"
calls_reset
echo '{"power.sleep":"nada"}' >"$SBX/ajustes.json"
ok run now
not_called "systemctl"

t "now: um 'hibernar' antigo não hiberna nem vira suspender"
calls_reset
echo '{"power.sleep":"hibernar"}' >"$SBX/ajustes.json"
ok run now
not_called "systemctl"

t "now: com o suspend indisponível, não chama o systemctl"
calls_reset
echo "disk" >"$sys/power/state"
echo '{"power.sleep":"suspender"}' >"$SBX/ajustes.json"
ok run now
not_called "systemctl"

t "argumento desconhecido: uso e saída 2"
run bobagem 2>/dev/null
eq 2 $?
