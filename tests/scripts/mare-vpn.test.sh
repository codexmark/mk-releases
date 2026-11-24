#!/usr/bin/env bash
# shell/mare/scripts/mare-vpn: a lista e o estado das VPNs, as ações (up, down,
# delete, autoconnect), a conferência e a importação de arquivos e o `ask`,
# com as respostas do nmcli em arquivos (MARE_VPN_FIXTURES) e, no fim, com um
# nmcli falso no PATH.
. "$(dirname "$0")/../lib.sh"

vpn_sh=$REPO/shell/mare/scripts/mare-vpn
fx=$SBX/fx
arq=$SBX/arquivos
mkdir -p "$arq"
# O `ask` espera 1,5 s depois de conectar e o `watch` dorme para sempre.
stub sleep

wg=aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa
ov=bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb
oc=cccccccc-cccc-4ccc-8ccc-cccccccccccc
w6=dddddddd-dddd-4ddd-8ddd-dddddddddddd
novo=12345678-1234-4234-8234-123456789abc

zera() {
  rm -rf "${fx:?}"
  mkdir -p "$fx/VPN"
  : >"$fx/calls.log"
  calls_reset
}
run() { MARE_VPN_FIXTURES=$fx bash "$vpn_sh" "$@"; }
chamadas() { cat "$fx/calls.log"; }

# A casa de sempre: um WireGuard ligado, um OpenVPN desligado que pede senha,
# um OpenConnect conectando, e o Wi-Fi e o cabo por baixo (que não são VPN).
casa() {
  zera
  : >"$fx/VPN/nm-openvpn-service.name"
  cat >"$fx/con-list" <<FIM
11111111-1111-4111-8111-111111111111:802-11-wireless:wlan0:yes:activated:yes:Casa
$ov:vpn::no::no:Trabalho\: matriz
$wg:wireguard:wg0:yes:activated:yes:wg0
$oc:vpn:wlan0:yes:activating:no:escritório "sul"
22222222-2222-4222-8222-222222222222:802-3-ethernet::no::yes:Conexão cabeada 1
FIM
  cat >"$fx/con-$wg" <<'FIM'
GENERAL.DEVICES:wg0
IP4.ADDRESS[1]:10.8.0.2/24
IP6.ADDRESS[1]:fe80::a1b2/64
FIM
  cat >"$fx/con-$ov" <<'FIM'
vpn.service-type:org.freedesktop.NetworkManager.openvpn
vpn.data:ca = /home/eu/ca.crt, connection-type = password, password-flags = 1, remote = vpn.exemplo.com\:1194, username = maria
FIM
  cat >"$fx/con-$oc" <<'FIM'
vpn.service-type:org.freedesktop.NetworkManager.openconnect
vpn.data:gateway = vpn.exemplo.com, protocol = anyconnect
GENERAL.DEVICES:wlan0
FIM
  cat >"$fx/ip-addr" <<'FIM'
1: lo    inet 127.0.0.1/8 scope host lo\       valid_lft forever preferred_lft forever
3: wlan0    inet 192.168.1.50/24 brd 192.168.1.255 scope global dynamic noprefixroute wlan0\       valid_lft 85000sec preferred_lft 85000sec
7: wg0    inet 10.8.0.2/24 brd 10.8.0.255 scope global noprefixroute wg0\       valid_lft forever preferred_lft forever
8: tun0    inet 10.9.0.6/24 brd 10.9.0.255 scope global noprefixroute tun0\       valid_lft forever preferred_lft forever
9: wg6    inet6 fd00:abcd::2/64 scope global noprefixroute \       valid_lft forever preferred_lft forever
FIM
}

# ---------------------------------------------------------------- list

t "list: só VPNs (vpn e wireguard), as ligadas primeiro e depois por nome"
casa
out=$(run list)
eq 0 $?
ok jq -e . <<<"$out" >/dev/null
eq '["escritório \"sul\"","wg0","Trabalho: matriz"]' "$(jq -c '[.connections[].name]' <<<"$out")"

t "list: cada conexão traz todos os campos do contrato"
eq '["active","autoconnect","device","iface","ip","kind","name","secrets","state","type","username","uuid"]' \
  "$(jq -c '.connections[0] | keys' <<<"$out")"

t "list: o WireGuard ligado: estado on, interface, ip e autoconnect"
eq '{"name":"wg0","uuid":"'"$wg"'","type":"wireguard","kind":"wireguard","device":"wg0","active":true,"state":"on","autoconnect":true,"iface":"wg0","ip":"10.8.0.2/24","secrets":false,"username":""}' \
  "$(jq -c '.connections[] | select(.name == "wg0")' <<<"$out")"

t "list: o OpenVPN desligado: kind do plugin, usuário guardado e secrets (senha do agente)"
eq '{"name":"Trabalho: matriz","uuid":"'"$ov"'","type":"vpn","kind":"openvpn","device":"","active":false,"state":"off","autoconnect":false,"iface":"","ip":"","secrets":true,"username":"maria"}' \
  "$(jq -c '.connections[] | select(.uuid == "'"$ov"'")' <<<"$out")"

t "list: 'activating' vira connecting e 'deactivating' vira disconnecting"
eq connecting "$(jq -r '.connections[] | select(.uuid == "'"$oc"'") | .state' <<<"$out")"
eq openconnect "$(jq -r '.connections[] | select(.uuid == "'"$oc"'") | .kind' <<<"$out")"
sed -i 's/:activating:/:deactivating:/' "$fx/con-list"
eq disconnecting "$(run list | jq -r '.connections[] | select(.uuid == "'"$oc"'") | .state')"

t "list: plugins diz o que está instalado (o wireguard é do kernel, sempre)"
eq '{"wireguard":true,"openvpn":true,"openconnect":false}' "$(jq -c .plugins <<<"$out")"
: >"$fx/VPN/nm-openconnect-service.name"
rm "$fx/VPN/nm-openvpn-service.name"
eq '{"wireguard":true,"openvpn":false,"openconnect":true}' "$(run list | jq -c .plugins)"

t "list: nome com dois-pontos, barra, aspas, espaço e acento sai como é"
zera
cat >"$fx/con-list" <<FIM
$wg:wireguard:wg0:no::no:a\:b\\\\c\\\\\:d
$ov:vpn::no::no:  Café "do" Zé  
$oc:vpn::no::no:fim\:
$w6:wireguard::no::no:\$(touch $SBX/pwn)\`touch $SBX/pwn2\`
FIM
out=$(run list)
ok jq -e . <<<"$out" >/dev/null
eq 'a:b\c\:d' "$(jq -r '.connections[] | select(.uuid == "'"$wg"'") | .name' <<<"$out")"
eq '  Café "do" Zé  ' "$(jq -r '.connections[] | select(.uuid == "'"$ov"'") | .name' <<<"$out")"
eq 'fim:' "$(jq -r '.connections[] | select(.uuid == "'"$oc"'") | .name' <<<"$out")"
eq '$(touch '"$SBX"'/pwn)`touch '"$SBX"'/pwn2`' "$(jq -r '.connections[] | select(.uuid == "'"$w6"'") | .name' <<<"$out")"
absent "$SBX/pwn"
absent "$SBX/pwn2"

t "list: secrets só quando a senha não está guardada no sistema (flags 1 ou 2)"
zera
echo "$ov:vpn::no::no:x" >"$fx/con-list"
for caso in "password:0:false" "password:1:true" "password:2:true" "password-tls:2:true" "tls:1:false" "password::false"; do
  IFS=: read -r ctype flags esperado <<<"$caso"
  printf 'vpn.service-type:org.freedesktop.NetworkManager.openvpn\nvpn.data:connection-type = %s, password-flags = %s, remote = h\n' "$ctype" "$flags" >"$fx/con-$ov"
  eq "$esperado" "$(run list | jq -r '.connections[0].secrets')" "$caso"
done

t "list: VPN de plugin ligada: a interface é a do túnel (quem tem o IP), não a rede por baixo"
casa
sed -i "s/^$ov:.*/$ov:vpn:wlan0:yes:activated:no:Trabalho/" "$fx/con-list"
cat >>"$fx/con-$ov" <<'FIM'
GENERAL.DEVICES:wlan0
IP4.ADDRESS[1]:10.9.0.6/24
FIM
out=$(run list | jq -c '.connections[] | select(.uuid == "'"$ov"'") | [.device, .iface, .ip, .state]')
eq '["wlan0","tun0","10.9.0.6/24","on"]' "$out"

t "list: túnel só com IPv6: o primeiro endereço que não é fe80"
zera
echo "$w6:wireguard:wg6:yes:activated:no:seis" >"$fx/con-list"
printf '%s\n' 'GENERAL.DEVICES:wg6' 'IP6.ADDRESS[1]:fe80::1/64' 'IP6.ADDRESS[2]:fd00:abcd::2/64' >"$fx/con-$w6"
echo '9: wg6    inet6 fd00:abcd::2/64 scope global noprefixroute \       valid_lft forever preferred_lft forever' >"$fx/ip-addr"
eq '["wg6","fd00:abcd::2/64"]' "$(run list | jq -c '.connections[0] | [.iface, .ip]')"

t "list: ligada e ainda sem IP: vale o device da conexão"
zera
echo "$wg:wireguard:wg0:yes:activated:no:wg0" >"$fx/con-list"
eq '["wg0",""]' "$(run list | jq -c '.connections[0] | [.iface, .ip]')"

t "list: sem nenhuma VPN: lista vazia e nm:true"
zera
echo "11111111-1111-4111-8111-111111111111:802-11-wireless:wlan0:yes:activated:yes:Casa" >"$fx/con-list"
eq '{"plugins":{"wireguard":true,"openvpn":false,"openconnect":false},"nm":true,"connections":[]}' "$(run list)"

t "list: NetworkManager parado: nm:false, que não é o mesmo que 'nenhuma vpn'"
zera
echo 8 >"$fx/con-list.rc"
out=$(run list)
eq 0 $?
eq false "$(jq -r .nm <<<"$out")"
eq '[]' "$(jq -c .connections <<<"$out")"

t "list: linhas malformadas do nmcli são puladas"
zera
printf '%s\n' "lixo" "a:b:c" "" "$wg:wireguard:wg0:no::no:wg0" ":::::::" >"$fx/con-list"
out=$(run list)
ok jq -e . <<<"$out" >/dev/null
eq '["wg0"]' "$(jq -c '[.connections[].name]' <<<"$out")"

# ---------------------------------------------------------------- status

t "status: nenhuma ligada: up:false, e uma chamada só ao nmcli"
casa
sed -i 's/:yes:activated:yes:wg0/:no::yes:wg0/' "$fx/con-list"
: >"$fx/calls.log"
out=$(run status)
eq 0 $?
eq '{"up":false,"count":0,"name":"","kind":"","iface":"","ip":""}' "$out"
eq 1 "$(chamadas | grep -c '^nmcli')"

t "status: uma ligada: nome, tipo, interface e ip, em duas chamadas ao nmcli"
casa
out=$(run status)
eq '{"up":true,"count":1,"name":"wg0","kind":"wireguard","iface":"wg0","ip":"10.8.0.2/24"}' "$out"
eq 2 "$(chamadas | grep -c '^nmcli')"

t "status: quem está conectando ainda não conta"
eq 1 "$(jq -r .count <<<"$out")"

t "status: duas ligadas: conta as duas e mostra a primeira"
casa
sed -i "s/^$ov:.*/$ov:vpn:wlan0:yes:activated:no:Trabalho\\\\: matriz/" "$fx/con-list"
printf '%s\n' 'GENERAL.DEVICES:wlan0' 'IP4.ADDRESS[1]:10.9.0.6/24' >>"$fx/con-$ov"
out=$(run status)
eq '{"up":true,"count":2,"name":"Trabalho: matriz","kind":"openvpn","iface":"tun0","ip":"10.9.0.6/24"}' "$out"

t "status: NetworkManager parado: up:false, JSON válido"
zera
echo 8 >"$fx/con-list.rc"
eq '{"up":false,"count":0,"name":"","kind":"","iface":"","ip":""}' "$(run status)"

# ---------------------------------------------------------------- ações

t "up: liga pelo UUID, com espera, e responde ok"
casa
: >"$fx/calls.log"
out=$(run up "$wg")
eq 0 $?
eq '{"ok":true,"error":"","needsSecrets":false}' "$out"
eq "nmcli --wait 45 con up uuid $wg" "$(chamadas)"

t "down: desliga pelo UUID"
: >"$fx/calls.log"
eq '{"ok":true,"error":"","needsSecrets":false}' "$(run down "$wg")"
eq "nmcli --wait 20 con down uuid $wg" "$(chamadas)"

t "delete: apaga pelo UUID"
: >"$fx/calls.log"
eq '{"ok":true,"error":"","needsSecrets":false}' "$(run delete "$wg")"
eq "nmcli con delete uuid $wg" "$(chamadas)"

t "autoconnect yes|no: muda connection.autoconnect"
: >"$fx/calls.log"
eq '{"ok":true,"error":"","needsSecrets":false}' "$(run autoconnect "$wg" yes)"
ok run autoconnect "$wg" no >/dev/null
eq "nmcli con mod uuid $wg connection.autoconnect yes
nmcli con mod uuid $wg connection.autoconnect no" "$(chamadas)"

t "autoconnect: valor que não é yes|no: uso em JSON, sai com 2, sem chamar o nmcli"
: >"$fx/calls.log"
for v in "" on true 'yes; touch '"$SBX"'/pwn3'; do
  out=$(run autoconnect "$wg" "$v")
  eq 2 $? "saída com '$v'"
  eq '{"ok":false,"error":"uso: mare-vpn autoconnect <uuid> yes|no","needsSecrets":false}' "$out" "'$v'"
done
eq "" "$(chamadas)"
absent "$SBX/pwn3"

t "up/down/delete/autoconnect: sem UUID ou com algo que não é UUID: 'conexão inválida', sai com 2"
: >"$fx/calls.log"
for verbo in up down delete autoconnect; do
  for id in "" "wg0" "Trabalho: matriz" "$wg extra" "${wg}0" '$(touch '"$SBX"'/pwn4)' "--help" "../../../etc/passwd" 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa;'; do
    out=$(run "$verbo" "$id" yes)
    eq 2 $? "saída de $verbo '$id'"
    eq '{"ok":false,"error":"conexão inválida","needsSecrets":false}' "$out" "$verbo '$id'"
  done
done
eq "" "$(chamadas)"
absent "$SBX/pwn4"

t "up: o nmcli pede segredos: needsSecrets e a frase em português"
casa
echo 4 >"$fx/up.rc"
echo "Error: Connection activation failed: Secrets were required, but not provided." >"$fx/up.out"
out=$(run up "$ov")
eq '{"ok":false,"error":"precisa de usuário e senha","needsSecrets":true}' "$out"

t "up: login recusado: 'usuário ou senha recusados'"
echo "Error: Connection activation failed: Invalid secrets" >"$fx/up.out"
eq '{"ok":false,"error":"usuário ou senha recusados","needsSecrets":true}' "$(run up "$ov")"

t "ações: cada erro conhecido do nmcli vira uma frase curta em português"
casa
echo 4 >"$fx/up.rc"
while IFS='|' read -r msg frase; do
  printf '%s\n' "$msg" >"$fx/up.out"
  out=$(run up "$ov")
  eq "$frase" "$(jq -r .error <<<"$out")" "$msg"
  eq false "$(jq -r .ok <<<"$out")" "$msg"
done <<'FIM'
Error: Connection activation failed: The VPN service 'org.freedesktop.NetworkManager.openvpn' was not installed.|falta o plugin dessa vpn no networkmanager
Error: Connection activation failed: The VPN service did not start in time|o serviço da vpn não subiu
Error: Connection activation failed: The VPN service returned invalid configuration|o servidor devolveu uma configuração inválida
Error: Connection activation failed: IP configuration could not be reserved (no available address, timeout, etc.)|o servidor não entregou um endereço ip
Error: Connection activation failed: The base network connection was interrupted|a rede por baixo caiu
Error: Timeout 45 sec expired.|o servidor não respondeu a tempo
Error: Connection activation failed: Not authorized to control networking.|sem permissão para mexer nas conexões
Error: NetworkManager is not running.|o networkmanager não está rodando
Error: unknown connection 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'.|essa conexão não existe mais
Error: Connection activation failed: Could not find source connection, no active connection|sem rede por baixo: conecte o wi-fi ou o cabo antes
Error: Connection activation failed: Algo que ninguém previu.|Algo que ninguém previu
FIM

t "ações: o nmcli falha calado: diz que falhou, não uma frase vazia"
rm "$fx/up.out"
eq '{"ok":false,"error":"o nmcli falhou sem dizer por quê","needsSecrets":false}' "$(run up "$ov")"

t "ações: erro com aspas e comando embutido vira texto no JSON"
echo 'Error: falhou "assim" $(touch '"$SBX"'/pwn5)' >"$fx/up.out"
out=$(run up "$ov")
ok jq -e . <<<"$out" >/dev/null
has "$(jq -r .error <<<"$out")" 'falhou "assim" $(touch'
absent "$SBX/pwn5"

t "down e delete: a falha do nmcli vira ok:false"
casa
echo 10 >"$fx/down.rc"
echo "Error: unknown connection '$wg'." >"$fx/down.out"
eq '{"ok":false,"error":"essa conexão não existe mais","needsSecrets":false}' "$(run down "$wg")"
echo 1 >"$fx/delete.rc"
echo "Error: Connection deletion failed: Insufficient privileges" >"$fx/delete.out"
eq '{"ok":false,"error":"sem permissão para mexer nas conexões","needsSecrets":false}' "$(run delete "$wg")"

# ---------------------------------------------------------------- check

wgconf() { # wgconf <caminho>
  printf '%s\n' '[Interface]' 'PrivateKey = cOFA0000000000000000000000000000000000000000=' 'Address = 10.8.0.2/24' \
    '' '[Peer]' 'PublicKey = dQw40000000000000000000000000000000000000000=' 'Endpoint = vpn.exemplo.com:51820' \
    'AllowedIPs = 0.0.0.0/0' >"$1"
}
ovpn() { printf '%s\n' 'client' 'dev tun' 'proto udp' 'remote vpn.exemplo.com 1194' 'auth-user-pass' >"$1"; }

t "check: um .conf do WireGuard passa, com kind, nome e caminho"
zera
wgconf "$arq/wg0.conf"
out=$(run check "$arq/wg0.conf")
eq 0 $?
eq '{"ok":true,"kind":"wireguard","name":"wg0","path":"'"$arq"'/wg0.conf","error":""}' "$out"

t "check: só confere, não chama o nmcli"
eq "" "$(chamadas)"

t "check: um .ovpn passa quando o plugin do openvpn está instalado"
: >"$fx/VPN/nm-openvpn-service.name"
ovpn "$arq/trabalho.ovpn"
eq '{"ok":true,"kind":"openvpn","name":"trabalho","path":"'"$arq"'/trabalho.ovpn","error":""}' "$(run check "$arq/trabalho.ovpn")"

t "check: OpenVPN num .conf é reconhecido pelo conteúdo"
ovpn "$arq/empresa.conf"
eq openvpn "$(run check "$arq/empresa.conf" | jq -r .kind)"

t "check: WireGuard num .txt é reconhecido pelo conteúdo"
wgconf "$arq/wg.txt"
eq '[true,"wireguard","wg"]' "$(run check "$arq/wg.txt" | jq -c '[.ok, .kind, .name]')"

t "check: .ovpn sem o plugin: diz qual pacote falta"
rm "$fx/VPN/nm-openvpn-service.name"
out=$(run check "$arq/trabalho.ovpn")
eq false "$(jq -r .ok <<<"$out")"
eq openvpn "$(jq -r .kind <<<"$out")"
has "$(jq -r .error <<<"$out")" "falta o plugin do openvpn (pacote networkmanager-vpn-plugin-openvpn)"

t "check: caminho vazio, pasta, arquivo que não existe e arquivo sem leitura: cada um com a sua frase"
eq "escreva o caminho de um arquivo .conf ou .ovpn" "$(run check | jq -r .error)"
eq "escreva o caminho de um arquivo .conf ou .ovpn" "$(run check "" | jq -r .error)"
eq "isso é uma pasta, não um arquivo" "$(run check "$arq" | jq -r .error)"
eq "esse arquivo não existe" "$(run check "$arq/sumiu.conf" | jq -r .error)"
wgconf "$arq/fechado.conf"
chmod 000 "$arq/fechado.conf"
if [[ -r $arq/fechado.conf ]]; then
  : # rodando como root: todo arquivo é legível, este pedaço não se aplica
else
  eq "não dá para ler esse arquivo" "$(run check "$arq/fechado.conf" | jq -r .error)"
fi
chmod 600 "$arq/fechado.conf"
eq false "$(run check "$arq/sumiu.conf" | jq -r .ok)"

t "check: conteúdo que não é VPN: a frase depende da extensão"
echo "olá" >"$arq/nada.conf"
echo "olá" >"$arq/nada.ovpn"
echo "olá" >"$arq/nada.txt"
has "$(run check "$arq/nada.conf" | jq -r .error)" "não parece uma configuração do wireguard"
has "$(run check "$arq/nada.ovpn" | jq -r .error)" "não parece uma configuração do openvpn"
eq "só arquivos .conf (wireguard) ou .ovpn (openvpn)" "$(run check "$arq/nada.txt" | jq -r .error)"

t "check: arquivo vazio não passa"
: >"$arq/vazio.conf"
eq false "$(run check "$arq/vazio.conf" | jq -r .ok)"

t "check: [Interface] sem PrivateKey não é WireGuard"
printf '%s\n' '[Interface]' 'Address = 10.8.0.2/24' >"$arq/semchave.conf"
eq false "$(run check "$arq/semchave.conf" | jq -r .ok)"

t "check: até 1 MiB passa; maior que isso é 'grande demais'"
wgconf "$arq/grande.conf"
truncate -s 1048576 "$arq/grande.conf"
eq true "$(run check "$arq/grande.conf" | jq -r .ok)"
truncate -s 1048577 "$arq/grande.conf"
eq "grande demais para ser uma configuração de vpn" "$(run check "$arq/grande.conf" | jq -r .error)"

t "check: aceita file:// e ~/ na frente do caminho"
wgconf "$HOME/casa.conf"
eq '[true,"'"$arq"'/wg0.conf"]' "$(run check "file://$arq/wg0.conf" | jq -c '[.ok, .path]')"
# shellcheck disable=SC2088
eq '[true,"'"$HOME"'/casa.conf"]' "$(run check '~/casa.conf' | jq -c '[.ok, .path]')"

t "check: nome de arquivo com espaço, aspas, acento e \$() é só um nome"
mau=$arq/'Minha "VPN" (casa) $(touch '"${SBX//\//_}"'_pwn6) çã.conf'
wgconf "$mau"
out=$(run check "$mau")
ok jq -e . <<<"$out" >/dev/null
eq true "$(jq -r .ok <<<"$out")"
eq "$mau" "$(jq -r .path <<<"$out")"
eq 'Minha "VPN" (casa) $(touch '"${SBX//\//_}"'_pwn6) çã' "$(jq -r .name <<<"$out")"
absent "${SBX//\//_}_pwn6"
absent "$arq/${SBX//\//_}_pwn6"

# ---------------------------------------------------------------- import

t "import: WireGuard de nome simples entra pelo próprio arquivo e não fica ligado"
zera
echo "Connection 'wg0' ($novo) successfully added." >"$fx/import"
out=$(run import "$arq/wg0.conf")
eq 0 $?
eq '{"ok":true,"kind":"wireguard","name":"wg0","uuid":"'"$novo"'","error":""}' "$out"
eq "nmcli con import type wireguard file $arq/wg0.conf
nmcli con mod uuid $novo connection.autoconnect no
nmcli con down uuid $novo" "$(chamadas)"

t "import: OpenVPN: tira o 'conectar sozinha' e não precisa desligar"
zera
: >"$fx/VPN/nm-openvpn-service.name"
echo "Connection 'trabalho' ($novo) successfully added." >"$fx/import"
out=$(run import "$arq/trabalho.ovpn")
eq '{"ok":true,"kind":"openvpn","name":"trabalho","uuid":"'"$novo"'","error":""}' "$out"
eq "nmcli con import type openvpn file $arq/trabalho.ovpn
nmcli con mod uuid $novo connection.autoconnect no" "$(chamadas)"

t "import: WireGuard de nome comprido ou com espaço entra por uma cópia de nome arrumado, que some depois"
zera
wgconf "$arq/Minha VPN (casa) de verdade.conf"
echo "Connection 'Minha-VPN-casa' ($novo) successfully added." >"$fx/import"
out=$(run import "$arq/Minha VPN (casa) de verdade.conf")
eq true "$(jq -r .ok <<<"$out")"
eq "Minha-VPN-casa" "$(jq -r .name <<<"$out")"
like "$(chamadas | head -1)" "^nmcli con import type wireguard file $XDG_RUNTIME_DIR/mare-vpn\.[A-Za-z0-9]+/Minha-VPN-casa\.conf$"
eq "" "$(find "$XDG_RUNTIME_DIR" -name 'mare-vpn.*')"
exists "$arq/Minha VPN (casa) de verdade.conf"

t "import: o nome da cópia cabe numa interface (até 15 caracteres, sem símbolos)"
for caso in "0123456789abcdefghij.conf|0123456789abcde" "wg.txt|wg" "wg0.conf.txt|wg0" "(   ).conf|wg0" "a  b.conf|a-b"; do
  zera
  wgconf "$arq/${caso%%|*}"
  echo "Connection 'x' ($novo) successfully added." >"$fx/import"
  ok run import "$arq/${caso%%|*}" >/dev/null
  like "$(chamadas | head -1)" "/mare-vpn\.[A-Za-z0-9]+/${caso##*|}\.conf$" "${caso%%|*}"
done

t "import: arquivo hostil no nome não roda nada e entra pela cópia"
zera
echo "Connection 'x' ($novo) successfully added." >"$fx/import"
out=$(run import "$mau")
eq true "$(jq -r .ok <<<"$out")"
hasnt "$(chamadas)" '$('
absent "${SBX//\//_}_pwn6"

t "import: arquivo que não passa na conferência: ok:false, sai com 1, sem chamar o nmcli"
zera
out=$(run import "$arq/nada.txt")
eq 1 $?
eq '{"ok":false,"kind":"","name":"","uuid":"","error":"só arquivos .conf (wireguard) ou .ovpn (openvpn)"}' "$out"
out=$(run import)
eq 1 $?
eq false "$(jq -r .ok <<<"$out")"
eq "" "$(chamadas)"

t "import: o nmcli recusa: ok:false com a frase, sai com 1, e a cópia some"
zera
echo 1 >"$fx/import.rc"
echo "Error: failed to import '/x/y.conf': Insufficient privileges" >"$fx/import"
out=$(run import "$arq/Minha VPN (casa) de verdade.conf")
eq 1 $?
eq '{"ok":false,"kind":"wireguard","name":"","uuid":"","error":"sem permissão para mexer nas conexões"}' "$out"
eq "" "$(find "$XDG_RUNTIME_DIR" -name 'mare-vpn.*')"
eq 1 "$(chamadas | grep -c '^nmcli')"

t "import: resposta do nmcli sem UUID: ok com o nome do arquivo e uuid vazio, sem mexer em mais nada"
zera
echo "resposta estranha" >"$fx/import"
out=$(run import "$arq/wg0.conf")
eq '{"ok":true,"kind":"wireguard","name":"wg0","uuid":"","error":""}' "$out"
eq 1 "$(chamadas | grep -c '^nmcli')"

t "import: nome com aspas simples na resposta do nmcli sai inteiro"
zera
echo "Connection 'd'Ávila 'x'' ($novo) successfully added." >"$fx/import"
eq "d'Ávila 'x'" "$(run import "$arq/wg0.conf" | jq -r .name)"

# ---------------------------------------------------------------- ask

t "ask: sem UUID válido: uso, sai com 2"
zera
err=$(run ask "wg0" 2>&1 >/dev/null)
eq 2 $?
has "$err" "uso: mare-vpn ask <uuid>"
eq "" "$(chamadas)"

t "ask: pede o usuário, guarda na conexão e liga com --ask (a senha é do nmcli)"
casa
sed -i 's/, username = maria//' "$fx/con-$ov"
echo "Trabalho: matriz" >"$fx/name-$ov"
: >"$fx/calls.log"
out=$(run ask "$ov" <<<"joao")
eq 0 $?
has "$out" "vpn: Trabalho: matriz"
has "$out" "conectada"
has "$(chamadas)" "nmcli con mod uuid $ov +vpn.data username=joao"
eq "nmcli --ask con up uuid $ov" "$(chamadas | tail -1)"

t "ask: com o usuário já guardado, não pergunta nem regrava"
casa
: >"$fx/calls.log"
out=$(run ask "$ov" </dev/null)
eq 0 $?
hasnt "$(chamadas)" "+vpn.data"
eq "nmcli --ask con up uuid $ov" "$(chamadas | tail -1)"

t "ask: usuário com espaço ou vírgula: avisa, sai com 1 e não tenta ligar"
casa
sed -i 's/, username = maria//' "$fx/con-$ov"
for u in "joão silva" "a,b"; do
  : >"$fx/calls.log"
  out=$(run ask "$ov" <<<"$u"$'\n'"x")
  eq 1 $? "saída com '$u'"
  has "$out" "usuário com espaço ou vírgula não dá para guardar por aqui" "'$u'"
  hasnt "$(chamadas)" "con mod" "'$u'"
  hasnt "$(chamadas)" "con up" "'$u'"
done

t "ask: usuário hostil vai como um valor só, sem rodar"
: >"$fx/calls.log"
out=$(run ask "$ov" <<<'x;$(touch${IFS}'"$SBX"'/pwn7)')
has "$(chamadas)" '+vpn.data username=x;$(touch${IFS}'
absent "$SBX/pwn7"

t "ask: usuário em branco não grava nada e segue para o nmcli"
: >"$fx/calls.log"
out=$(run ask "$ov" <<<"")
hasnt "$(chamadas)" "+vpn.data"
has "$(chamadas)" "nmcli --ask con up uuid $ov"

# (O "não conectou" é o texto de um `read -p`, que só aparece num terminal.)
t "ask: se não conectar, não diz 'conectada' e sai com o código do nmcli"
echo 4 >"$fx/up.rc"
out=$(run ask "$ov" <<<"joao"$'\n'"x" 2>/dev/null)
eq 4 $?
hasnt "$out" "conectada"

t "ask: sem nome conhecido, mostra o UUID"
casa
out=$(run ask "$wg" </dev/null)
has "$out" "vpn: $wg"

# ---------------------------------------------------------------- watch e uso

t "watch (com fixtures): fica aberto sem falar com o nmcli"
zera
ok run watch
called "sleep infinity"
eq "" "$(chamadas)"

t "sem subcomando ou subcomando desconhecido: uso, sai com 2"
err=$(run 2>&1 >/dev/null)
eq 2 $?
has "$err" "uso: mare-vpn list | status"
run bobagem 2>/dev/null
eq 2 $?
eq "" "$(chamadas)"

# ------------------------------------------------- sem fixtures: nmcli no PATH

# Daqui em diante o script fala com o nmcli e o ip do PATH (stubs).
real() { env -u MARE_VPN_FIXTURES bash "$vpn_sh" "$@"; }
stub ip 'cat <<FIM
3: wlan0    inet 192.168.1.50/24 brd 192.168.1.255 scope global dynamic noprefixroute wlan0\       valid_lft 85000sec preferred_lft 85000sec
8: tun0    inet 10.9.0.6/24 brd 10.9.0.255 scope global noprefixroute tun0\       valid_lft forever preferred_lft forever
FIM'
stub nmcli '
printf "[%s]" "$@" >>"$SBX/args"; echo >>"$SBX/args"
[[ $LC_ALL == C ]] || { echo "Erro: mensagem traduzida" >&2; exit 99; }
case "$*" in
  "-t -f UUID,TYPE,DEVICE,ACTIVE,STATE,AUTOCONNECT,NAME con show")
    echo "'"$ov"':vpn:wlan0:yes:activated:no:Trabalho\: matriz"
    echo "11111111-1111-4111-8111-111111111111:802-11-wireless:wlan0:yes:activated:yes:Casa" ;;
  "-t -f vpn.service-type,vpn.data,GENERAL.DEVICES,IP4.ADDRESS,IP6.ADDRESS con show uuid '"$ov"'")
    echo "vpn.service-type:org.freedesktop.NetworkManager.openvpn"
    echo "vpn.data:connection-type = password, password-flags = 1, username = maria"
    echo "GENERAL.DEVICES:wlan0"
    echo "IP4.ADDRESS[1]:10.9.0.6/24" ;;
  "--wait 45 con up uuid "*)
    [[ -f $SBX/nm-falha ]] && { cat "$SBX/nm-falha" >&2; exit 4; }
    echo "Connection successfully activated" ;;
  "--wait 20 con down uuid "* | "con delete uuid "* | "con mod uuid "*) : ;;
  "con import type wireguard file "*) echo "Connection '"'"'wg0'"'"' ('"$novo"') successfully added." ;;
  "con down uuid "*) : ;;
  monitor) echo "NetworkManager is running" ;;
  *) echo "Error: argumento inesperado: $*" >&2; exit 2 ;;
esac'

t "status com o nmcli do PATH: lê as conexões e acha a interface do túnel pelo ip"
calls_reset
out=$(real status)
eq '{"up":true,"count":1,"name":"Trabalho: matriz","kind":"openvpn","iface":"tun0","ip":"10.9.0.6/24"}' "$out"
called "ip -o addr show"

t "up/down/delete/autoconnect com o nmcli do PATH: o UUID é um argumento só e o inglês é forçado"
: >"$SBX/args"
eq '{"ok":true,"error":"","needsSecrets":false}' "$(real up "$ov")"
eq '{"ok":true,"error":"","needsSecrets":false}' "$(real down "$ov")"
eq '{"ok":true,"error":"","needsSecrets":false}' "$(real delete "$ov")"
eq '{"ok":true,"error":"","needsSecrets":false}' "$(real autoconnect "$ov" no)"
eq "[--wait][45][con][up][uuid][$ov]
[--wait][20][con][down][uuid][$ov]
[con][delete][uuid][$ov]
[con][mod][uuid][$ov][connection.autoconnect][no]" "$(<"$SBX/args")"

t "up com o nmcli do PATH falhando: o erro do stderr vira a frase"
echo "Error: Connection activation failed: Secrets were required, but not provided." >"$SBX/nm-falha"
eq '{"ok":false,"error":"precisa de usuário e senha","needsSecrets":true}' "$(real up "$ov")"
rm "$SBX/nm-falha"

t "import com o nmcli do PATH: importa, tira o autoconnect e desliga"
calls_reset
out=$(real import "$arq/wg0.conf")
eq '{"ok":true,"kind":"wireguard","name":"wg0","uuid":"'"$novo"'","error":""}' "$out"
called "nmcli con import type wireguard file $arq/wg0.conf"
called "nmcli con mod uuid $novo connection.autoconnect no"
called "nmcli con down uuid $novo"

t "UUID inválido com o nmcli do PATH: nada é chamado"
calls_reset
out=$(real delete 'x; touch '"$SBX"'/pwn8')
eq 2 $?
not_called "nmcli"
absent "$SBX/pwn8"

t "watch: vira o nmcli monitor, preso à vida do Maré (setpriv --pdeathsig)"
calls_reset
stub setpriv '[[ "$1 $2" == "--pdeathsig TERM" ]] || exit 64; shift 2; exec "$@"'
out=$(real watch)
eq "NetworkManager is running" "$out"
called "setpriv --pdeathsig TERM nmcli monitor"
called "nmcli monitor"
