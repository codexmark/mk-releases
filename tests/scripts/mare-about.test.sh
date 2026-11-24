#!/usr/bin/env bash
# shell/mare/scripts/mare-about: os detalhes da máquina em JSON, com /proc e
# /etc falsos e uname, uptime, hyprctl e qs por stub.
. "$(dirname "$0")/../lib.sh"

about_sh=$REPO/shell/mare/scripts/mare-about
proc=$SBX/proc etc=$SBX/etc
mkdir -p "$proc" "$etc"

echo "maquina-do-mk" >"$etc/hostname"
cat >"$proc/cpuinfo" <<'EOF'
processor	: 0
vendor_id	: GenuineIntel
model name	: 11th Gen Intel(R) Core(TM) i7-1165G7 @ 2.80GHz
processor	: 1
model name	: Outro Processador
EOF
printf 'MemTotal:       16303428 kB\nMemFree:         1234567 kB\n' >"$proc/meminfo"

stub uname 'case $1 in -r) echo 6.9.1-arch1-1 ;; -n) echo nome-do-uname ;; esac'
stub uptime 'echo "up 2 days, 3 hours, 5 minutes"'
stub hyprctl 'echo "{\"branch\":\"\",\"commit\":\"abc\",\"version\":\"0.57.0\",\"tag\":\"v0.57.0\"}"'
stub qs 'echo "quickshell 0.3.2 (revision 2b4501a2, distributed by Arch Linux)"'

run() { PROC=$proc ETC=$etc bash "$about_sh"; }

t "a saída é uma linha de JSON com as sete chaves"
out=$(run)
eq 1 "$(wc -l <<<"$out")"
eq '["cpu","host","hyprland","kernel","mem","quickshell","uptime"]' "$(jq -cS keys <<<"$out")"

t "máquina comum: cada campo no formato da página Sobre"
eq '{"host":"maquina-do-mk","kernel":"6.9.1-arch1-1","cpu":"11th Gen Intel Core i7-1165G7 @ 2.80GHz","mem":"15,5 GiB","uptime":"2 d 3 h 5 min","hyprland":"0.57.0","quickshell":"0.3.2"}' "$out"

t "só consulta: hyprctl version e qs --version, nada mais"
called "hyprctl version -j"
called "qs --version"
eq 2 "$(grep -c '^hyprctl\|^qs' "$SBX/calls")"

t "cpu: o primeiro 'model name', sem os (R) e (TM) e sem espaço dobrado"
printf 'model name\t: AMD Ryzen 7 5800X  8-Core   Processor\n' >"$proc/cpuinfo"
eq "AMD Ryzen 7 5800X 8-Core Processor" "$(run | jq -r .cpu)"

t "cpu: sem 'model name' (ARM), vem vazio e o JSON continua válido"
printf 'processor\t: 0\nBogoMIPS\t: 48.00\n' >"$proc/cpuinfo"
eq "" "$(run | jq -r .cpu)"

t "mem: em GiB com uma casa e vírgula"
printf 'MemTotal:        8388608 kB\n' >"$proc/meminfo"
eq "8,0 GiB" "$(run | jq -r .mem)"
printf 'MemTotal:       65536000 kB\n' >"$proc/meminfo"
eq "62,5 GiB" "$(run | jq -r .mem)"

t "host: sem /etc/hostname, vale o nome que o kernel diz"
rm -f "$etc/hostname"
eq "nome-do-uname" "$(run | jq -r .host)"

t "host: nome com aspas e barra não quebra o JSON"
printf '%s\n' 'pc"da\sala' >"$etc/hostname"
eq 'pc"da\sala' "$(run | jq -r .host)"
echo "maquina-do-mk" >"$etc/hostname"

t "uptime: singular e plural encurtam igual (1 day, 1 hour, 1 minute)"
stub uptime 'echo "up 1 day, 1 hour, 1 minute"'
eq "1 d 1 h 1 min" "$(run | jq -r .uptime)"
stub uptime 'echo "up 42 minutes"'
eq "42 min" "$(run | jq -r .uptime)"

t "uptime: se o comando falha, vem vazio"
stub uptime 'exit 1'
eq "" "$(run | jq -r .uptime)"

t "hyprland: sem 'version' na resposta, vale a tag"
stub hyprctl 'echo "{\"tag\":\"v0.57.0-12-gabc\"}"'
eq "v0.57.0-12-gabc" "$(run | jq -r .hyprland)"

t "hyprland: fora de uma sessão (hyprctl falha ou não responde JSON), vem vazio"
stub hyprctl 'echo "HYPRLAND_INSTANCE_SIGNATURE not set!"; exit 1'
out=$(run)
ok jq -e . <<<"$out" >/dev/null
eq "" "$(jq -r .hyprland <<<"$out")"

t "quickshell: sem resposta do qs, vem vazio"
stub qs 'exit 1'
out=$(run)
ok jq -e . <<<"$out" >/dev/null
eq "" "$(jq -r .quickshell <<<"$out")"
