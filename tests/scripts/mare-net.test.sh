#!/usr/bin/env bash
# shell/mare/scripts/mare-net: o JSON da conexão em uso (interface, tipo, ip,
# gateway, dns e o modo do dns), com ip e nmcli falsos e um /sys de mentira.
. "$(dirname "$0")/../lib.sh"

net_sh=$REPO/shell/mare/scripts/mare-net
sys=$SBX/sys nm=$SBX/nm
mkdir -p "$sys/class/net/wlteste0/wireless" "$sys/class/net/enp3s0" "$nm"

# O nmcli de mentira: com -g os ":" e as "\" saem escapados (a não ser com
# "-e no"), como no de verdade, e `con show` só acha a conexão pelo nome exato.
#   $nm/conn nome da conexão    $nm/addr, $nm/gw  valores de IP4.ADDRESS e IP4.GATEWAY
#   $nm/dns  linhas "IP4.DNS[n]:..." do -t -f     $nm/ignore, $nm/fixed  ipv4.* da conexão
stub nmcli '
printf "[%s]" "$@" >>"$SBX/args"; echo >>"$SBX/args"
nm=$SBX/nm esc=1 prev=""
for a in "$@"; do [[ $prev == -e && $a == no ]] && esc=0; prev=$a; done
val() {
  [[ -f $nm/$1 ]] || return 0
  if ((esc)); then sed '"'"'s/[\\:]/\\&/g'"'"' "$nm/$1"; else cat "$nm/$1"; fi
}
last=${!#}
case " $* " in
  *" GENERAL.CONNECTION dev show "*) val conn ;;
  *" IP4.ADDRESS dev show "*) val addr ;;
  *" IP4.GATEWAY dev show "*) val gw ;;
  *" -t -f IP4.DNS dev show "*) [[ -f $nm/dns ]] && cat "$nm/dns" ;;
  *" con show "*)
    [[ -n $last && $last == "$(<"$nm/conn")" ]] || { echo "Error: $last - no such connection profile." >&2; exit 10; }
    case " $* " in
      *" ipv4.ignore-auto-dns "*) val ignore ;;
      *" ipv4.dns "*) val fixed ;;
    esac ;;
  *) exit 2 ;;
esac
exit 0'

rota() { stub ip "echo '1.1.1.1 via 192.168.1.1 dev $1 src 192.168.1.50 uid 1000'; echo '    cache'"; }
rede() { # rede <conexão> <ignore-auto-dns> <ipv4.dns>
  printf '%s\n' "$1" >"$nm/conn"
  echo "192.168.1.50/24" >"$nm/addr"
  echo "192.168.1.1" >"$nm/gw"
  printf 'IP4.DNS[1]:192.168.1.1\n' >"$nm/dns"
  printf '%s\n' "$2" >"$nm/ignore"
  printf '%s\n' "$3" >"$nm/fixed"
  calls_reset
  : >"$SBX/args"
}
run() { SYS=$sys bash "$net_sh"; }

t "wi-fi: interface, tipo, conexão, ip, gateway e dns num JSON só"
rota wlteste0
rede "Casa" no ""
out=$(run)
eq 0 $?
eq '{"iface":"wlteste0","type":"wifi","conn":"Casa","ip":"192.168.1.50/24","gateway":"192.168.1.1","dns":["192.168.1.1"],"dnsMode":"auto"}' "$out"
ok jq -e . <<<"$out" >/dev/null

t "cabo: a interface sem pasta wireless é ethernet"
rota enp3s0
rede "Conexão cabeada 1" no ""
out=$(run)
eq ethernet "$(jq -r .type <<<"$out")"
eq enp3s0 "$(jq -r .iface <<<"$out")"
eq "Conexão cabeada 1" "$(jq -r .conn <<<"$out")"

t "rota sem 'via' (túnel): a interface ainda é achada"
stub ip 'echo "1.1.1.1 dev wlteste0 src 10.8.0.2 uid 1000"'
rede "Casa" no ""
eq wlteste0 "$(run | jq -r .iface)"

t "sem rota (o ip falha): tipo none, tudo vazio, sai com 0 e não chama o nmcli"
stub ip 'echo "RTNETLINK answers: Network is unreachable" >&2; exit 2'
rede "Casa" no ""
out=$(run)
eq 0 $?
eq '{"iface":"","type":"none","conn":"","ip":"","gateway":"","dns":[],"dnsMode":"auto"}' "$out"
not_called "nmcli"

t "resposta do ip sem 'dev': tipo none"
stub ip 'echo "bobagem"'
eq none "$(run | jq -r .type)"

t "vários endereços e vários dns: o primeiro ip, todos os dns"
rota wlteste0
rede "Casa" no ""
echo "192.168.1.50/24 | 10.0.0.7/8" >"$nm/addr"
printf 'IP4.DNS[1]:192.168.1.1\nIP4.DNS[2]:9.9.9.9\n' >"$nm/dns"
out=$(run)
eq "192.168.1.50/24" "$(jq -r .ip <<<"$out")"
eq '["192.168.1.1","9.9.9.9"]' "$(jq -c .dns <<<"$out")"

t "sem dns nenhum: lista vazia, não um JSON quebrado"
rede "Casa" no ""
: >"$nm/dns"
out=$(run)
eq '[]' "$(jq -c .dns <<<"$out")"

t "dnsMode: ignora o automático e usa 1.1.1.1 = cloudflare"
rede "Casa" yes "1.1.1.1,1.0.0.1"
eq cloudflare "$(run | jq -r .dnsMode)"

t "dnsMode: ignora o automático e usa 8.8.8.8 = google"
rede "Casa" yes "8.8.8.8,8.8.4.4"
eq google "$(run | jq -r .dnsMode)"

t "dnsMode: outro dns fixo = custom"
rede "Casa" yes "9.9.9.9"
eq custom "$(run | jq -r .dnsMode)"

t "dnsMode: um dns fixo que só contém '1.1.1.1' no meio (10.1.1.1, 8.8.8.80) é custom"
rede "Casa" yes "10.1.1.1"
eq custom "$(run | jq -r .dnsMode)"
rede "Casa" yes "21.1.1.10,8.8.8.80"
eq custom "$(run | jq -r .dnsMode)"

t "dnsMode: dns fixo sem ignorar o automático continua auto"
rede "Casa" no "1.1.1.1"
eq auto "$(run | jq -r .dnsMode)"

t "dnsMode: ignorar o automático sem dns fixo continua auto"
rede "Casa" yes ""
eq auto "$(run | jq -r .dnsMode)"

t "interface sem conexão do NetworkManager: conn vazia, modo auto, sem perguntar pela conexão"
rede "" no ""
: >"$nm/conn"
out=$(run)
eq 0 $?
eq "" "$(jq -r .conn <<<"$out")"
eq auto "$(jq -r .dnsMode <<<"$out")"
not_called "con show"

t "conexão com dois-pontos e barra no nome: sai inteira e o modo do dns é lido"
rede 'Casa: 5G \ sala' yes "8.8.8.8,8.8.4.4"
out=$(run)
eq 'Casa: 5G \ sala' "$(jq -r .conn <<<"$out")"
eq google "$(jq -r .dnsMode <<<"$out")"
has "$(<"$SBX/args")" '[show][Casa: 5G \ sala]'

t "conexão com aspas, acento e comando embutido: vira texto no JSON, nada roda"
rede 'Café "do" Zé $(touch '"$SBX"'/pwn)`touch '"$SBX"'/pwn2`' no ""
out=$(run)
ok jq -e . <<<"$out" >/dev/null
eq 'Café "do" Zé $(touch '"$SBX"'/pwn)`touch '"$SBX"'/pwn2`' "$(jq -r .conn <<<"$out")"
absent "$SBX/pwn"
absent "$SBX/pwn2"

t "o nmcli falhando em tudo: JSON válido, com a interface e o resto vazio"
rota wlteste0
stub nmcli 'echo "Error: NetworkManager is not running." >&2; exit 8'
out=$(run)
eq 0 $?
eq '{"iface":"wlteste0","type":"wifi","conn":"","ip":"","gateway":"","dns":[],"dnsMode":"auto"}' "$out"
