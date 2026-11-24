#!/usr/bin/env bash
# shell/mare/scripts/mare-stats: as leituras cruas (cpu, memória, rede,
# temperatura, disco), com /proc e /sys falsos e o df por stub.
. "$(dirname "$0")/../lib.sh"

stats_sh=$REPO/shell/mare/scripts/mare-stats
sys=$SBX/sys proc=$SBX/proc
mkdir -p "$proc/net" "$sys/class/hwmon"

stub df 'printf "%s\n" "  1B-blocks         Used" "500000000000 123456789012"'

cat >"$proc/stat" <<'EOF'
cpu  100 20 30 4000 500 6 7 8 9 10
cpu0 50 10 15 2000 250 3 3 4 4 5
cpu1 50 10 15 2000 250 3 4 4 5 5
intr 123456 0 0
ctxt 987654
EOF
cat >"$proc/meminfo" <<'EOF'
MemTotal:       16303428 kB
MemFree:         1234567 kB
MemAvailable:    8151714 kB
Buffers:          123456 kB
EOF
cat >"$proc/net/dev" <<'EOF'
Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
    lo: 9999999   10000    0    0    0     0          0         0  9999999   10000    0    0    0     0       0          0
  eth0: 1000000    2000    0    0    0     0          0         0   300000    1500    0    0    0     0       0          0
enp3s0: 2500000    4000    0    0    0     0          0         0   700000    2500    0    0    0     0       0          0
EOF
sensor() { # sensor <hwmonN> <nome> [milésimos de grau]
  mkdir -p "$sys/class/hwmon/$1"
  echo "$2" >"$sys/class/hwmon/$1/name"
  [[ -z ${3:-} ]] || echo "$3" >"$sys/class/hwmon/$1/temp1_input"
}
run() { SYS=$sys PROC=$proc bash "$stats_sh" 2>/dev/null; }
val() { run | awk -v k="$1" '$1 == k { print $2 }'; }

t "cpu: o total soma todos os campos da linha 'cpu ' e o ocioso é idle + iowait"
eq 4690 "$(val cpu_total)"
eq 4500 "$(val cpu_idle)"

t "cpu: as linhas por núcleo (cpu0, cpu1) não entram"
eq 1 "$(run | grep -c '^cpu_total ')"

t "memória: MemTotal e MemAvailable em kB (MemFree não é o disponível)"
eq 16303428 "$(val mem_total)"
eq 8151714 "$(val mem_avail)"

t "rede: soma os bytes recebidos e enviados de todas as interfaces, menos a lo"
eq 3500000 "$(val net_rx)"
eq 1000000 "$(val net_tx)"

t "rede: nome comprido, sem espaço antes (wlp0s20f3:), conta os bytes e não os pacotes"
cp "$proc/net/dev" "$SBX/dev.bak"
printf '%s\n' "wlp0s20f3: 40000000   30000    0    0    0     0          0         0  5000000   20000    0    0    0     0       0          0" >>"$proc/net/dev"
eq 43500000 "$(val net_rx)"
eq 6000000 "$(val net_tx)"
cp "$SBX/dev.bak" "$proc/net/dev"

t "rede: só com a lo, os contadores vêm em 0, não vazios"
head -3 "$SBX/dev.bak" >"$proc/net/dev"
eq 0 "$(val net_rx)"
eq 0 "$(val net_tx)"
cp "$SBX/dev.bak" "$proc/net/dev"

t "temperatura: sem sensor de CPU conhecido, a linha temp não sai"
sensor hwmon0 nvme 35000
sensor hwmon1 iwlwifi_1 40000
eq 0 "$(run | grep -c '^temp ')"

t "temperatura: o primeiro sensor conhecido (coretemp, k10temp, zenpower, cpu_thermal)"
for nome in coretemp k10temp zenpower cpu_thermal; do
  sensor hwmon2 "$nome" 47000
  eq 47000 "$(val temp)" "$nome"
done
sensor hwmon3 k10temp 99000
eq 1 "$(run | grep -c '^temp ')" "um sensor só"
eq 47000 "$(val temp)"

t "temperatura: sensor conhecido sem leitura dá 0"
rm -f "$sys/class/hwmon/hwmon2/temp1_input" "$sys/class/hwmon/hwmon3/temp1_input"
eq 0 "$(val temp)"

t "disco: tamanho e uso da raiz em bytes, pedidos ao df"
calls_reset
eq 500000000000 "$(val disk_total)"
eq 123456789012 "$(val disk_used)"
called "df -B1 --output=size,used /"

t "a saída é 'chave valor', uma por linha, na ordem que o painel espera"
sensor hwmon2 coretemp 51000
eq "cpu_total cpu_idle mem_total mem_avail net_rx net_tx temp disk_total disk_used" "$(run | awk '{ printf "%s%s", (NR > 1 ? " " : ""), $1 }')"
eq 0 "$(run | grep -vcE '^[a-z_]+ [0-9]+$')" "todo valor é um número inteiro"

t "sem /proc legível, o que dá para ler ainda sai"
out=$(SYS=$sys PROC=$SBX/nao-existe bash "$stats_sh" 2>/dev/null)
has "$out" "disk_total 500000000000"
has "$out" "temp 51000"
hasnt "$out" "cpu_total"
