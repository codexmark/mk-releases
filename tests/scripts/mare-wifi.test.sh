#!/usr/bin/env bash
# shell/mare/scripts/mare-wifi: as redes guardadas (list-saved) e as ações por
# SSID (forget, autoconnect), com um nmcli falso que guarda perfis em arquivos.
. "$(dirname "$0")/../lib.sh"

wifi_sh=$REPO/shell/mare/scripts/mare-wifi
nm=$SBX/nm
mkdir -p "$nm"

# O nmcli de mentira. Cada perfil é um arquivo $nm/p-<uuid> com o que o
# `con show` diz dele (-t -e no: "chave:valor", o valor cru).
#   $nm/parado       existe = NetworkManager parado
#   $nm/cons         saída de `nmcli -t -f UUID,TYPE con show`
#   $nm/falha-<uuid> existe = delete/modify desse perfil falha (o texto é o erro)
stub nmcli '
printf "[%s]" "$@" >>"$SBX/args"; echo >>"$SBX/args"
nm=$SBX/nm
[[ -e $nm/parado ]] && { echo "Error: NetworkManager is not running." >&2; exit 8; }
case "$*" in
  "-t -f RUNNING general") echo running ;;
  "-t -f UUID,TYPE con show") cat "$nm/cons" ;;
  "-t -e no -f connection.uuid,connection.autoconnect,802-11-wireless.ssid con show "*)
    shift 7; first=1
    while (($#)); do
      [[ $1 == uuid ]] || { echo "Error: esperava uuid" >&2; exit 2; }
      [[ -f $nm/p-$2 ]] || { echo "Error: $2 - no such connection profile." >&2; exit 10; }
      ((first)) || echo
      cat "$nm/p-$2"; first=0
      shift 2
    done ;;
  "con delete uuid "* | "con modify uuid "*)
    if [[ -e $nm/falha-$4 ]]; then
      echo "Warning: algo antes do erro" >&2
      cat "$nm/falha-$4" >&2
      exit 1
    fi
    echo "Connection $4 ok." ;;
  *) echo "Error: argumento inesperado: $*" >&2; exit 2 ;;
esac'

u1=11111111-1111-4111-8111-111111111111
u2=22222222-2222-4222-8222-222222222222
u3=33333333-3333-4333-8333-333333333333
u4=44444444-4444-4444-8444-444444444444
ue=eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee

perfil() { # perfil <uuid> <yes|no> <ssid>
  printf 'connection.uuid:%s\nconnection.autoconnect:%s\n802-11-wireless.ssid:%s\n' "$1" "$2" "$3" >"$nm/p-$1"
}
limpa() {
  rm -f "$nm"/p-* "$nm"/falha-* "$nm/parado"
  : >"$nm/cons"
  calls_reset
  : >"$SBX/args"
}
casa() { # a casa de sempre: três Wi-Fi (dois da mesma rede), um cabo e uma vpn
  limpa
  printf '%s\n' "$u1:802-11-wireless" "$ue:802-3-ethernet" "$u2:802-11-wireless" \
    "55555555-5555-4555-8555-555555555555:wireguard" "$u3:802-11-wireless" >"$nm/cons"
  perfil "$u1" yes "Casa"
  perfil "$u2" no "Café da Maré: 5G"
  perfil "$u3" yes "Casa"
}
run() { bash "$wifi_sh" "$@"; }

t "list-saved: só os perfis Wi-Fi, com ssid, uuid e autoconnect"
casa
out=$(run list-saved)
eq 0 $?
eq '{"ok":true,"saved":[{"uuid":"'"$u1"'","autoconnect":true,"ssid":"Casa"},{"uuid":"'"$u2"'","autoconnect":false,"ssid":"Café da Maré: 5G"},{"uuid":"'"$u3"'","autoconnect":true,"ssid":"Casa"}]}' "$out"
hasnt "$(<"$SBX/args")" "[$ue]"

t "list-saved: nenhum perfil Wi-Fi dá lista vazia, sem perguntar pelos detalhes"
limpa
echo "$ue:802-3-ethernet" >"$nm/cons"
out=$(run list-saved)
eq 0 $?
eq '{"ok":true,"saved":[]}' "$out"
not_called "connection.uuid"

t "list-saved: nenhuma conexão guardada dá lista vazia"
limpa
out=$(run list-saved)
eq 0 $?
eq '{"ok":true,"saved":[]}' "$out"

t "list-saved: NetworkManager parado responde ok:false e sai com 1"
casa
: >"$nm/parado"
out=$(run list-saved)
eq 1 $?
eq '{"ok":false,"saved":[]}' "$out"

t "list-saved: SSID com espaço nas pontas, aspas, tab e dois-pontos sai igual ao que entrou"
limpa
printf '%s\n' "$u1:802-11-wireless" "$u2:802-11-wireless" "$u3:802-11-wireless" >"$nm/cons"
perfil "$u1" yes '  Rede "do" Zé  '
perfil "$u2" no $'a\tb:c::d\\e'
perfil "$u3" yes '$(touch '"$SBX"'/pwn)'
out=$(run list-saved)
ok jq -e . <<<"$out" >/dev/null
eq '  Rede "do" Zé  ' "$(jq -r '.saved[0].ssid' <<<"$out")"
eq $'a\tb:c::d\\e' "$(jq -r '.saved[1].ssid' <<<"$out")"
eq '$(touch '"$SBX"'/pwn)' "$(jq -r '.saved[2].ssid' <<<"$out")"
absent "$SBX/pwn"

t "list-saved: rede escondida (SSID vazio) entra com ssid vazio"
limpa
echo "$u1:802-11-wireless" >"$nm/cons"
perfil "$u1" no ""
eq '{"ok":true,"saved":[{"uuid":"'"$u1"'","autoconnect":false,"ssid":""}]}' "$(run list-saved)"

t "list-saved: linhas malformadas na lista de conexões são ignoradas"
casa
printf '%s\n' "lixo sem dois pontos" "" ":::" "$u1:802-11-wireless" >"$nm/cons"
out=$(run list-saved)
eq 0 $?
eq '["Casa"]' "$(jq -c '[.saved[].ssid]' <<<"$out")"

t "list-saved: o nmcli sem resposta para os detalhes dá lista vazia e sai com 0"
casa
rm -f "$nm"/p-*
out=$(run list-saved 2>/dev/null)
eq 0 $?
eq '{"ok":true,"saved":[]}' "$out"

t "forget: apaga pelo UUID todos os perfis do SSID, e só eles"
casa
out=$(run forget "Casa")
eq 0 $?
eq '{"ok":true,"count":2,"error":""}' "$out"
called "nmcli con delete uuid $u1"
called "nmcli con delete uuid $u3"
not_called "con delete uuid $u2"

t "forget: o SSID nunca vai para o nmcli"
hasnt "$(<"$SBX/args")" "[Casa]"

t "forget: SSID com espaço, acento e dois-pontos acha o perfil certo"
casa
out=$(run forget "Café da Maré: 5G")
eq '{"ok":true,"count":1,"error":""}' "$out"
called "nmcli con delete uuid $u2"
not_called "con delete uuid $u1"

t "forget: o SSID é comparado inteiro (prefixo, maiúsculas e curinga não casam)"
casa
for nome in "Cas" "casa" "Casa " "*" "Ca?a" "[C]asa"; do
  out=$(run forget "$nome")
  eq 1 $? "saída de forget '$nome'"
  eq '{"ok":false,"count":0,"error":"essa rede não está guardada"}' "$out" "forget '$nome'"
done
not_called "con delete"

t "forget: rede que não está guardada responde o erro e sai com 1"
casa
out=$(run forget "Vizinho")
eq 1 $?
eq '{"ok":false,"count":0,"error":"essa rede não está guardada"}' "$out"
not_called "con delete"

t "forget: SSID hostil não vira comando nem opção do nmcli"
casa
for nome in '$(touch '"$SBX"'/pwn1)' '`touch '"$SBX"'/pwn2`' '; touch '"$SBX"'/pwn3' '--help' "uuid $u1" "$u1"; do
  out=$(run forget "$nome")
  eq false "$(jq -r .ok <<<"$out")" "forget '$nome'"
done
absent "$SBX/pwn1"
absent "$SBX/pwn2"
absent "$SBX/pwn3"
not_called "con delete"
hasnt "$(<"$SBX/args")" "[--help]"

t "forget: um SSID com aspas e \$() guardado de verdade é apagado pelo UUID"
limpa
echo "$u4:802-11-wireless" >"$nm/cons"
perfil "$u4" yes '"; $(touch '"$SBX"'/pwn4) #'
out=$(run forget '"; $(touch '"$SBX"'/pwn4) #')
eq '{"ok":true,"count":1,"error":""}' "$out"
called "nmcli con delete uuid $u4"
absent "$SBX/pwn4"

t "forget: se o nmcli recusa um dos perfis, conta os que saíram e traz a última linha do erro"
casa
echo "Error: Connection deletion failed: Insufficient privileges" >"$nm/falha-$u3"
out=$(run forget "Casa")
eq 1 $?
eq '{"ok":false,"count":1,"error":"Error: Connection deletion failed: Insufficient privileges"}' "$out"
called "nmcli con delete uuid $u1"

t "forget: sem SSID, SSID vazio ou argumento a mais: uso em JSON, sai com 2"
casa
for args in "forget" "forget|" "forget|a|b"; do
  IFS='|' read -r -a argv <<<"$args"
  [[ $args == "forget|" ]] && argv=(forget "")
  out=$(run "${argv[@]}")
  eq 2 $? "saída de '$args'"
  eq '{"ok":false,"count":0,"error":"uso: mare-wifi forget <ssid>"}' "$out" "'$args'"
done
not_called "nmcli"

t "autoconnect on: connection.autoconnect yes em todos os perfis do SSID"
casa
out=$(run autoconnect "Casa" on)
eq 0 $?
eq '{"ok":true,"count":2,"error":""}' "$out"
called "nmcli con modify uuid $u1 connection.autoconnect yes"
called "nmcli con modify uuid $u3 connection.autoconnect yes"
not_called "uuid $u2 connection.autoconnect"

t "autoconnect off: connection.autoconnect no"
casa
out=$(run autoconnect "Café da Maré: 5G" off)
eq '{"ok":true,"count":1,"error":""}' "$out"
called "nmcli con modify uuid $u2 connection.autoconnect no"

t "autoconnect: rede que não está guardada responde o erro e sai com 1"
casa
out=$(run autoconnect "Vizinho" on)
eq 1 $?
eq '{"ok":false,"count":0,"error":"essa rede não está guardada"}' "$out"
not_called "con modify"

t "autoconnect: o nmcli recusando vira ok:false com o erro dele"
casa
echo "Error: Failed to modify connection: Insufficient privileges" >"$nm/falha-$u2"
out=$(run autoconnect "Café da Maré: 5G" on)
eq 1 $?
eq '{"ok":false,"count":0,"error":"Error: Failed to modify connection: Insufficient privileges"}' "$out"

t "autoconnect: valor que não é on|off, valor faltando ou SSID vazio: uso em JSON, sai com 2"
casa
uso='{"ok":false,"count":0,"error":"uso: mare-wifi autoconnect <ssid> on|off"}'
out=$(run autoconnect "Casa" yes); eq 2 $?; eq "$uso" "$out"
out=$(run autoconnect "Casa"); eq 2 $?; eq "$uso" "$out"
out=$(run autoconnect "" on); eq 2 $?; eq "$uso" "$out"
out=$(run autoconnect "Casa" on extra); eq 2 $?; eq "$uso" "$out"
out=$(run autoconnect "Casa" '$(touch '"$SBX"'/pwn5)'); eq 2 $?; eq "$uso" "$out"
absent "$SBX/pwn5"
not_called "nmcli"

t "forget com o NetworkManager parado: 'não está guardada', sai com 1"
casa
: >"$nm/parado"
out=$(run forget "Casa")
eq 1 $?
eq false "$(jq -r .ok <<<"$out")"
eq 0 "$(jq -r .count <<<"$out")"

t "sem subcomando ou subcomando desconhecido: uso no stderr, sai com 2"
casa
err=$(run 2>&1 >/dev/null)
eq 2 $?
has "$err" "uso: mare-wifi list-saved | forget <ssid> | autoconnect <ssid> on|off"
run bobagem 2>/dev/null
eq 2 $?
not_called "nmcli"
