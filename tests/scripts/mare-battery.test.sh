#!/usr/bin/env bash
# shell/mare/scripts/mare-battery: ciclos, saúde e limite de carga em JSON,
# com um /sys falso.
. "$(dirname "$0")/../lib.sh"

bat_sh=$REPO/shell/mare/scripts/mare-battery
sys=$SBX/sys
ps=$sys/class/power_supply
mkdir -p "$ps"

run() { SYS=$sys bash "$bat_sh"; }
# bateria <nome> <arquivo=valor>...
bateria() {
  local nome=$1 par
  shift
  rm -rf "${ps:?}/$nome"
  mkdir -p "$ps/$nome"
  for par in "$@"; do echo "${par#*=}" >"$ps/$nome/${par%%=*}"; done
}

t "sem bateria: tudo null"
mkdir -p "$ps/AC"
eq '{"cycles":null,"health":null,"limit":null}' "$(run)"

t "bateria em energia (energy_full): ciclos, saúde e limite"
bateria BAT0 cycle_count=312 energy_full=45000000 energy_full_design=50000000 charge_control_end_threshold=80
eq '{"cycles":312,"health":0.9,"limit":80}' "$(run)"

t "bateria em carga (charge_full), sem energy_*: a saúde sai dela"
bateria BAT0 cycle_count=10 charge_full=3000000 charge_full_design=4000000
eq '{"cycles":10,"health":0.75,"limit":null}' "$(run)"

t "com energy_* e charge_*, vale o par de energia"
bateria BAT0 energy_full=50 energy_full_design=100 charge_full=99 charge_full_design=100
eq 0.5 "$(run | jq -c .health)"

t "só metade do par de energia: cai para o par de carga"
bateria BAT0 energy_full=50 charge_full=30 charge_full_design=40
eq 0.75 "$(run | jq -c .health)"

t "ciclos em 0 (o firmware não conta) viram null"
bateria BAT0 cycle_count=0 energy_full=1 energy_full_design=1
eq '{"cycles":null,"health":1,"limit":null}' "$(run)"

t "capacidade de fábrica em 0 não divide por zero: saúde null"
bateria BAT0 cycle_count=5 energy_full=45 energy_full_design=0
eq '{"cycles":5,"health":null,"limit":null}' "$(run)"

t "bateria sem arquivo nenhum: JSON válido, tudo null"
bateria BAT0
eq '{"cycles":null,"health":null,"limit":null}' "$(run)"

t "limite de carga em 100 e em 0 são números, não null"
bateria BAT0 charge_control_end_threshold=100
eq 100 "$(run | jq -c .limit)"
bateria BAT0 charge_control_end_threshold=0
eq 0 "$(run | jq -c .limit)"

t "uma bateria nova pode passar de 1 de saúde"
bateria BAT0 energy_full=51 energy_full_design=50
eq 1.02 "$(run | jq -c .health)"

t "com duas baterias, vale a primeira; a tomada não conta"
bateria BAT0 cycle_count=7
bateria BAT1 cycle_count=99
eq 7 "$(run | jq -c .cycles)"
rm -rf "${ps:?}/BAT0"
eq 99 "$(run | jq -c .cycles)"

t "valor que não é número (leitura torta do firmware) vira null, sem quebrar o JSON"
bateria BAT1 cycle_count=muitos energy_full=cheia energy_full_design=50 charge_control_end_threshold=--
out=$(run)
eq '{"cycles":null,"health":null,"limit":null}' "$out"

t "a saída é uma linha só"
bateria BAT1 cycle_count=3 energy_full=9 energy_full_design=10
eq 1 "$(run | wc -l)"
