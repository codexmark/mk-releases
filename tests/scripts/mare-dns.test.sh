#!/usr/bin/env bash
# shell/mare/scripts/mare-dns: o que cada modo (auto, cloudflare, google) pede
# ao nmcli para a conexão em uso, e os avisos, com ip e nmcli falsos.
. "$(dirname "$0")/../lib.sh"

dns_sh=$REPO/shell/mare/scripts/mare-dns
nm=$SBX/nm
mkdir -p "$nm"

# O nmcli de mentira: com -g os ":" e as "\" saem escapados (a não ser com
# "-e no"), e `con mod` só aceita o nome exato da conexão. Cada chamada fica
# também em $SBX/args, com um [ ] por argumento.
#   $nm/conn nome da conexão    $nm/mod.rc, $nm/reapply.rc  código de saída
stub nmcli '
printf "[%s]" "$@" >>"$SBX/args"; echo >>"$SBX/args"
nm=$SBX/nm esc=1 prev=""
for a in "$@"; do [[ $prev == -e && $a == no ]] && esc=0; prev=$a; done
case " $* " in
  *" GENERAL.CONNECTION dev show "*)
    if ((esc)); then sed '"'"'s/[\\:]/\\&/g'"'"' "$nm/conn"; else cat "$nm/conn"; fi ;;
  " con mod "*)
    [[ $3 == "$(<"$nm/conn")" ]] || { echo "Error: unknown connection '"'"'$3'"'"'." >&2; exit 10; }
    [[ -f $nm/mod.rc ]] && { echo "Error: Failed to modify connection: Insufficient privileges" >&2; exit "$(<"$nm/mod.rc")"; } ;;
  " dev reapply "*)
    [[ -f $nm/reapply.rc ]] && { echo "Error: Reapply failed" >&2; exit "$(<"$nm/reapply.rc")"; }
    echo "Connection successfully reapplied to device." ;;
  *) exit 2 ;;
esac
exit 0'

rede() { # rede <conexão> [interface]
  printf '%s\n' "$1" >"$nm/conn"
  stub ip "echo '1.1.1.1 via 192.168.1.1 dev ${2:-wlan0} src 192.168.1.50 uid 1000'; echo '    cache'"
  rm -f "$nm/mod.rc" "$nm/reapply.rc"
  calls_reset
  : >"$SBX/args"
}
run() { bash "$dns_sh" "$@"; }

t "cloudflare: grava 1.1.1.1 no ipv4 e no ipv6, ignora o dns automático e reaplica"
rede "Casa"
out=$(run cloudflare)
eq 0 $?
eq "Casa: dns cloudflare" "$out"
has "$(<"$SBX/args")" '[con][mod][Casa][ipv4.dns][1.1.1.1 1.0.0.1][ipv4.ignore-auto-dns][yes][ipv6.dns][2606:4700:4700::1111 2606:4700:4700::1001][ipv6.ignore-auto-dns][yes]'
has "$(<"$SBX/args")" '[dev][reapply][wlan0]'

t "google: grava 8.8.8.8 no ipv4 e no ipv6 e ignora o dns automático"
rede "Casa"
out=$(run google)
eq "Casa: dns google" "$out"
has "$(<"$SBX/args")" '[con][mod][Casa][ipv4.dns][8.8.8.8 8.8.4.4][ipv4.ignore-auto-dns][yes][ipv6.dns][2001:4860:4860::8888 2001:4860:4860::8844][ipv6.ignore-auto-dns][yes]'

t "auto: esvazia o dns fixo (argumento vazio) e volta a aceitar o do roteador"
rede "Casa"
out=$(run auto)
eq "Casa: dns auto" "$out"
has "$(<"$SBX/args")" '[con][mod][Casa][ipv4.dns][][ipv4.ignore-auto-dns][no][ipv6.dns][][ipv6.ignore-auto-dns][no]'

t "a ordem: primeiro muda a conexão, depois reaplica na interface dela"
rede "Cabo" enp3s0
ok run google >/dev/null
eq "nmcli con mod Cabo" "$(grep -n 'con mod\|dev reapply' "$SBX/calls" | head -1 | cut -d: -f2 | cut -c1-18)"
called "nmcli dev reapply enp3s0"

t "só a conexão em uso muda, sem sudo"
not_called "sudo"
eq 1 "$(grep -c 'con mod' "$SBX/calls")"

t "a saída do reapply não suja a resposta"
rede "Casa"
eq "Casa: dns auto" "$(run auto 2>&1)"

t "conexão com espaço, acento e aspas: um argumento só"
rede 'Café "do" Zé 5G'
out=$(run cloudflare)
eq 0 $?
eq 'Café "do" Zé 5G: dns cloudflare' "$out"
has "$(<"$SBX/args")" '[con][mod][Café "do" Zé 5G][ipv4.dns]'

t "conexão com dois-pontos e barra no nome: o nmcli recebe o nome sem barra de escape"
rede 'Casa: 5G \ sala'
out=$(run google)
eq 0 $?
eq 'Casa: 5G \ sala: dns google' "$out"
has "$(<"$SBX/args")" '[con][mod][Casa: 5G \ sala][ipv4.dns]'

t "conexão com comando embutido no nome: nada roda"
rede '$(touch '"$SBX"'/pwn)`touch '"$SBX"'/pwn2`; touch '"$SBX"'/pwn3'
ok run auto >/dev/null
absent "$SBX/pwn"
absent "$SBX/pwn2"
absent "$SBX/pwn3"

t "sem argumento: uso, sai com 2 e não chama nada"
rede "Casa"
err=$(run 2>&1 >/dev/null)
eq 2 $?
has "$err" "uso: mare-dns auto | cloudflare | google"
not_called "ip"
not_called "nmcli"

t "modo desconhecido (e um endereço solto): uso, sai com 2 e não chama nada"
run quad9 2>/dev/null
eq 2 $?
run 9.9.9.9 2>/dev/null
eq 2 $?
run '$(touch '"$SBX"'/pwn4)' 2>/dev/null
eq 2 $?
absent "$SBX/pwn4"
not_called "nmcli"

t "sem rota (o ip sai com 2): 'sem conexão ativa', sai com 1 e não muda nada"
rede "Casa"
stub ip 'echo "RTNETLINK answers: Network is unreachable" >&2; exit 2'
err=$(run cloudflare 2>&1 >/dev/null)
eq 1 $?
has "$err" "sem conexão ativa"
not_called "nmcli"

t "resposta do ip sem 'dev': 'sem conexão ativa'"
stub ip 'echo "bobagem"'
err=$(run cloudflare 2>&1 >/dev/null)
eq 1 $?
has "$err" "sem conexão ativa"

t "interface fora do NetworkManager: avisa qual, sai com 1 e não muda nada"
rede "" docker0
: >"$nm/conn"
err=$(run cloudflare 2>&1 >/dev/null)
eq 1 $?
has "$err" "docker0 não está ligado a uma conexão do NetworkManager"
not_called "con mod"
not_called "reapply"

t "o nmcli recusa a mudança: sai diferente de 0, não reaplica e não diz que trocou"
rede "Casa"
echo 4 >"$nm/mod.rc"
out=$(run cloudflare 2>/dev/null)
eq 4 $?
eq "" "$out"
not_called "reapply"

t "o reapply falha: sai diferente de 0 e não diz que trocou"
rede "Casa"
echo 1 >"$nm/reapply.rc"
out=$(run cloudflare 2>/dev/null)
eq 1 $?
eq "" "$out"
