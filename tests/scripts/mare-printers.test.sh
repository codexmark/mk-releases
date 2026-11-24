#!/usr/bin/env bash
# shell/mare/scripts/mare-printers: a leitura do CUPS (status, find), as ações
# (add, default, remove, test, cancel, resume, enable) com e sem a janela de
# senha, o aviso no fim e as frases de erro, com as ferramentas do CUPS, o
# pkexec e o systemctl falsos e uma raiz de mentira (MARE_PRINTERS_ROOT).
. "$(dirname "$0")/../lib.sh"

prn_sh=$REPO/shell/mare/scripts/mare-printers
root=$SBX/raiz
c=$SBX/cups
sd=$XDG_RUNTIME_DIR/mare-printers
dados=$root/usr/share/cups/data
eu=$(id -un)

# As ações esperam em laços de `sleep 0.2`: aqui cada espera dura um instante
# (só nas chamadas de run/acao; o selftest usa o sleep de verdade).
dormir=$(command -v sleep)
mkdir -p "$SBX/rapido"
printf '%s\n' '#!/bin/bash' "exec $dormir 0.02" >"$SBX/rapido/sleep"
chmod +x "$SBX/rapido/sleep"

# O que os stubs respondem sai de $c:
#   r p v o d        saída de lpstat -r, -p, -v, -o, -d
#   ippfind lpinfo   saída do ippfind e do lpinfo (lpinfo.rc: código de saída)
#   habilitado       existe = cups.socket liga no boot;  avahi: existe = descoberta ligada
#   recusa           existe = o CUPS responde Forbidden às ações de administrador
#   erro-<comando>   existe = o comando falha com esse texto
#   pkexec.rc        código com que a janela de senha termina (126 = fechada)
#   pkexec.espera    segundos que a janela de senha demora
#   systemctl.rc     código do `systemctl enable`
#   qs-state         o que `qs ipc call mare state` responde
# Cada chamada fica também em $SBX/args: o nome e um [ ] por argumento.
registra='c=$SBX/cups n=${0##*/}
{ printf "%s" "$n"; printf "[%s]" "$@"; echo; } >>"$SBX/args"
'
falha='if [[ -e $c/erro-$n ]]; then cat "$c/erro-$n" >&2; exit 1; fi
'
recusa='if [[ -z ${COMO_ROOT:-} && -e $c/recusa ]]; then echo "$n: Forbidden" >&2; exit 1; fi
'
stub lpstat "$registra"'
case $1 in
  -r) f=r ;; -p) f=p ;; -v) f=v ;; -o) f=o ;; -d) f=d ;; *) exit 1 ;;
esac
[[ -f $c/$f ]] && cat "$c/$f"
exit 0'
stub lpadmin "$registra$recusa$falha"
stub cupsenable "$registra$recusa$falha"
stub cupsaccept "$registra$recusa$falha"
stub cancel "$registra"'if [[ $1 == -a ]]; then '"$recusa"' fi
'"$falha"
stub lpoptions "$registra$falha"
stub lp "$registra$falha"'echo "request id is $2-99 (1 file(s))"'
stub ippfind "$registra"'[[ -f $c/ippfind ]] && cat "$c/ippfind"; exit 0'
stub lpinfo "$registra"'[[ -f $c/lpinfo ]] && cat "$c/lpinfo"; [[ -f $c/lpinfo.rc ]] && exit "$(<"$c/lpinfo.rc")"; exit 0'
stub systemctl "$registra"'
case $1 in
  is-enabled) [[ -e $c/habilitado && ${!#} == "$(<"$c/habilitado")" ]] ;;
  is-active) [[ -e $c/avahi ]] ;;
  enable) [[ -f $c/systemctl.rc ]] && exit "$(<"$c/systemctl.rc")"; echo "scheduler is running" >"$c/r" ;;
  *) exit 1 ;;
esac'
stub pkexec "$registra"'
[[ -f $c/pkexec.espera ]] && '"$dormir"' "$(<"$c/pkexec.espera")"
[[ -f $c/pkexec.rc ]] && exit "$(<"$c/pkexec.rc")"
COMO_ROOT=1 exec "$@"'
stub notify-send "$registra"
stub qs "$registra"'
case "$*" in
  *"ipc call mare state") [[ -f $c/qs-state ]] && cat "$c/qs-state" ;;
  *"ipc call mare settings impressoras") : ;;
  *) exit 1 ;;
esac'

run() { PATH=$SBX/rapido:$PATH MARE_PRINTERS_ROOT=$root bash "$prn_sh" "$@"; }
args() { cat "$SBX/args"; }
# O JSON do resultado é a última linha do que a ação imprime.
acao() { res=$(run "$@" 2>/dev/null | tail -n1); }
campo() { jq -r "$1" <<<"$res"; }

zera() {
  rm -rf "${c:?}" "${sd:?}" "${root:?}"
  mkdir -p "$c" "$root/usr/lib/systemd/system" "$dados"
  : >"$root/usr/lib/systemd/system/cups.socket"
  : >"$dados/default-testpage.pdf"
  : >"$SBX/args"
  calls_reset
}
# Três impressoras: parada, imprimindo e em pausa; nomes com hífen.
tres() {
  zera
  echo "scheduler is running" >"$c/r"
  echo "cups.socket" >"$c/habilitado"
  : >"$c/avahi"
  printf '%s\n' \
    "printer Brother_HL-L2350DW is idle.  enabled since Thu 19 Nov 2026 10:12:44 AM -03" \
    "printer EPSON-ET-2850 now printing EPSON-ET-2850-42.  enabled since Fri 20 Nov 2026 09:01:10 AM -03" \
    $'\tRendering completed' \
    "printer HP_LaserJet_M15w disabled since Wed 18 Nov 2026 05:40:02 PM -03 -" \
    $'\tPaused' >"$c/p"
  echo "system default destination: EPSON-ET-2850" >"$c/d"
  printf '%s\n' \
    "device for Brother_HL-L2350DW: ipp://BRW1234.local:631/ipp/print" \
    "device for EPSON-ET-2850: ipps://EPSON1A2B3C.local:631/ipp/print" \
    "device for HP_LaserJet_M15w: ipp://localhost:60000/ipp/print" >"$c/v"
  printf '%s\n' \
    "EPSON-ET-2850-42        $eu            204800   Fri 20 Nov 2026 09:01:08 AM -03" \
    "EPSON-ET-2850-43        $eu             10240   Fri 20 Nov 2026 09:02:30 AM -03" \
    "HP_LaserJet_M15w-7      outra-pessoa       1024   Wed 18 Nov 2026 05:39:50 PM -03" >"$c/o"
}

# ---------------------------------------------------------------- status

t "status: sem CUPS instalado: installed e running falsos, e o lpstat nem é chamado"
zera
rm "$root/usr/lib/systemd/system/cups.socket"
out=$(run status)
eq 0 $?
eq '{"installed":false,"running":false,"enabled":false,"default":"","pending":"","last":null,"printers":[]}' "$out"
not_called "lpstat"

t "status: o cupsd sem o soquete do systemd também conta como instalado"
mkdir -p "$root/usr/bin"
: >"$root/usr/bin/cupsd"
eq true "$(run status | jq -r .installed)"

t "status: instalado e parado: running falso, sem perguntar pelas impressoras"
zera
echo "scheduler is not running" >"$c/r"
out=$(run status)
eq '{"installed":true,"running":false,"enabled":false,"default":"","pending":"","last":null,"printers":[]}' "$out"
called "lpstat -r"
not_called "lpstat -p"

t "status: o lpstat sem resposta nenhuma conta como parado"
zera
eq false "$(run status | jq -r .running)"

t "status: enabled vem do systemd (cups.socket ou cups.service)"
zera
echo "cups.socket" >"$c/habilitado"
eq true "$(run status | jq -r .enabled)"
echo "cups.service" >"$c/habilitado"
eq true "$(run status | jq -r .enabled)"
called "systemctl is-enabled --quiet cups.socket"
not_called "systemctl enable"

t "status: ligado e sem impressoras: lista vazia e padrão vazio"
zera
echo "scheduler is running" >"$c/r"
echo "no system default destination" >"$c/d"
out=$(run status)
eq '[true,"",[]]' "$(jq -c '[.running, .default, .printers]' <<<"$out")"

t "status: três impressoras, com estado, mensagem, endereço, padrão e fila"
tres
out=$(run status)
ok jq -e . <<<"$out" >/dev/null
eq '{"name":"Brother_HL-L2350DW","state":"idle","message":"","uri":"ipp://BRW1234.local:631/ipp/print","default":false,"jobs":[]}' \
  "$(jq -c '.printers[0]' <<<"$out")"
eq '{"name":"EPSON-ET-2850","state":"printing","message":"Rendering completed","uri":"ipps://EPSON1A2B3C.local:631/ipp/print","default":true,"jobs":[{"id":42,"user":"'"$eu"'","size":204800},{"id":43,"user":"'"$eu"'","size":10240}]}' \
  "$(jq -c '.printers[1]' <<<"$out")"
eq '{"name":"HP_LaserJet_M15w","state":"paused","message":"Paused","uri":"ipp://localhost:60000/ipp/print","default":false,"jobs":[{"id":7,"user":"outra-pessoa","size":1024}]}' \
  "$(jq -c '.printers[2]' <<<"$out")"
eq "EPSON-ET-2850" "$(jq -r .default <<<"$out")"
eq '["installed","running","enabled","default","pending","last","printers"]' "$(jq -c 'keys_unsorted' <<<"$out")"

t "status: só lê (lpstat e systemctl is-enabled), não muda nada"
not_called "lpadmin"
not_called "pkexec"
not_called "systemctl enable"
absent "$sd"

t "status: nome com acento e mensagem com aspas e comando embutido viram texto"
zera
echo "scheduler is running" >"$c/r"
printf '%s\n' "printer Impressora_do_Zé is idle.  enabled since Thu 19 Nov 2026 10:12:44 AM -03" \
  $'\t"Sem papel" $(touch '"$SBX"'/pwn1) `touch '"$SBX"'/pwn2`' \
  'printer $(touch${IFS}'"$SBX"'/pwn3) is idle.  enabled since Thu 19 Nov 2026 10:12:44 AM -03' >"$c/p"
echo "device for Impressora_do_Zé: ipp://zé.local:631/ipp/print?a=1&b=\"2\"" >"$c/v"
out=$(run status)
ok jq -e . <<<"$out" >/dev/null
eq "Impressora_do_Zé" "$(jq -r '.printers[0].name' <<<"$out")"
eq '"Sem papel" $(touch '"$SBX"'/pwn1) `touch '"$SBX"'/pwn2`' "$(jq -r '.printers[0].message' <<<"$out")"
eq 'ipp://zé.local:631/ipp/print?a=1&b="2"' "$(jq -r '.printers[0].uri' <<<"$out")"
eq '$(touch${IFS}'"$SBX"'/pwn3)' "$(jq -r '.printers[1].name' <<<"$out")"
absent "$SBX/pwn1"
absent "$SBX/pwn2"
absent "$SBX/pwn3"

t "status: saída malformada do lpstat não quebra o JSON"
zera
echo "scheduler is running" >"$c/r"
printf '%s\n' "lpstat: algo deu errado" "" $'\tlinha solta' "printer" >"$c/p"
printf '%s\n' "device sem dois pontos" "lixo" >"$c/v"
printf '%s\n' "linha sem numero   maria   10" "HP-x maria 10" "HP-12" >"$c/o"
echo "lixo" >"$c/d"
out=$(run status)
eq 0 $?
ok jq -e . <<<"$out" >/dev/null
eq "" "$(jq -r .default <<<"$out")"

t "status: trabalho com tamanho ilegível entra com size 0"
tres
echo "HP_LaserJet_M15w-8      maria       ???   Wed 18 Nov 2026" >"$c/o"
eq '[{"id":8,"user":"maria","size":0}]' "$(run status | jq -c '.printers[2].jobs')"

t "status: last é o resultado da última ação, se tem menos de 30 s"
tres
mkdir -p "$sd"
agora=$(date +%s%3N)
echo '{"key":"remove:HP","ok":true,"error":"","message":"HP removida","elevated":false,"at":'"$agora"'}' >"$sd/ultimo.json"
eq '{"key":"remove:HP","ok":true,"error":"","message":"HP removida","elevated":false,"at":'"$agora"'}' "$(run status | jq -c .last)"
echo '{"key":"remove:HP","ok":true,"error":"","message":"HP removida","elevated":false,"at":'"$((agora - 31000))"'}' >"$sd/ultimo.json"
eq null "$(run status | jq -c .last)"

t "status: ultimo.json quebrado, vazio ou que não é objeto vira last:null"
for conteudo in '{"key":' '' '[1,2]' '"texto"' '$(touch '"$SBX"'/pwn4)'; do
  printf '%s' "$conteudo" >"$sd/ultimo.json"
  out=$(run status)
  eq null "$(jq -c .last <<<"$out")" "ultimo.json = '$conteudo'"
  eq 3 "$(jq '.printers | length' <<<"$out")" "ultimo.json = '$conteudo'"
done
absent "$SBX/pwn4"

t "status: pending é a chave da ação cujo processo ainda vive"
tres
mkdir -p "$sd"
echo "$$ remove:HP_LaserJet_M15w" >"$sd/pid"
eq "remove:HP_LaserJet_M15w" "$(run status | jq -r .pending)"

t "status: pid de um processo que já morreu (ou lixo no arquivo) não é pending"
(exit 0) &
morto=$!
wait "$morto"
echo "$morto remove:HP_LaserJet_M15w" >"$sd/pid"
eq "" "$(run status | jq -r .pending)"
echo 'lixo; $(touch '"$SBX"'/pwn5)' >"$sd/pid"
eq "" "$(run status | jq -r .pending)"
: >"$sd/pid"
eq "" "$(run status | jq -r .pending)"
absent "$SBX/pwn5"

# ---------------------------------------------------------------- find

t "find: impressoras IPP da rede e do USB (ipp-usb), sem repetir a que anuncia ipp e ipps"
tres
printf '%s\t%s\t%s\n' \
  "Brother HL-L2350DW series" "ipp://BRW1234.local:631/ipp/print" "Brother HL-L2350DW series" \
  "EPSON ET-2850 Series" "ipps://EPSON1A2B3C.local:631/ipp/print" "EPSON ET-2850 Series" \
  "EPSON ET-2850 Series" "ipp://EPSON1A2B3C.local:631/ipp/print" "EPSON ET-2850 Series" \
  "Canon G3010 (USB)" "ipp://localhost:60001/ipp/print" "{txt_ty}" \
  "Sala @ servidor" "http://servidor.local:631/" "" >"$c/ippfind"
printf '%s\n' \
  "Device: uri = usb://Samsung/ML-1660%20Series?serial=Z4WABKBZ500123" \
  "        class = direct" \
  "        info = Samsung ML-1660 Series" \
  "        make-and-model = Samsung ML-1660 Series" \
  "        device-id = MFG:Samsung;CMD:SPL,FWV;MDL:ML-1660 Series;CLS:PRINTER;" \
  "        location =" >"$c/lpinfo"
out=$(run find)
eq 0 $?
ok jq -e . <<<"$out" >/dev/null
eq '[{"name":"Brother HL-L2350DW series","uri":"ipp://BRW1234.local:631/ipp/print","model":"Brother HL-L2350DW series","kind":"rede","added":true},{"name":"Canon G3010 (USB)","uri":"ipp://localhost:60001/ipp/print","model":"","kind":"usb","added":false},{"name":"EPSON ET-2850 Series","uri":"ipp://EPSON1A2B3C.local:631/ipp/print","model":"EPSON ET-2850 Series","kind":"rede","added":false}]' \
  "$(jq -c .found <<<"$out")"

t "find: as USB que pedem driver vão em legacy, com o modelo"
eq '[{"uri":"usb://Samsung/ML-1660%20Series?serial=Z4WABKBZ500123","model":"Samsung ML-1660 Series"}]' "$(jq -c .legacy <<<"$out")"
eq '[true,false]' "$(jq -c '[.mdns, .legacyUnknown]' <<<"$out")"

t "find: procura pelo ippfind (ipp e ipps) e pelo lpinfo só no USB, sem pedir senha"
called "ippfind -T 4 _ipp._tcp _ipps._tcp --exec printf"
called "lpinfo --timeout 3 --include-schemes usb -l -v"
not_called "pkexec"
not_called "lpadmin"

t "find: USB sem make-and-model usa o info; 127.0.0.1 e [::1] também são USB"
printf '%s\t%s\t%s\n' "A" "ipp://127.0.0.1:60000/ipp/print" "" "B" "ipps://[::1]:60001/ipp/print" "" \
  "C" "ipp://localhost.exemplo.com/ipp/print" "" >"$c/ippfind"
printf '%s\n' "Device: uri = usb://HP/LaserJet%201020" "        info = HP LaserJet 1020" "        make-and-model = Unknown" \
  "Device: uri = usb://Sem/Nada" >"$c/lpinfo"
out=$(run find)
eq '["usb","usb","rede"]' "$(jq -c '[.found[].kind]' <<<"$out")"
eq '[{"uri":"usb://HP/LaserJet%201020","model":"HP LaserJet 1020"},{"uri":"usb://Sem/Nada","model":""}]' "$(jq -c .legacy <<<"$out")"

t "find: o CUPS recusa o lpinfo: legacyUnknown, e a lista de legacy fica vazia"
echo "lpinfo: Forbidden" >"$c/lpinfo"
echo 1 >"$c/lpinfo.rc"
out=$(run find)
eq '[true,[]]' "$(jq -c '[.legacyUnknown, .legacy]' <<<"$out")"

t "find: sem o avahi, mdns:false"
tres
rm "$c/avahi"
eq false "$(run find | jq -r .mdns)"
called "systemctl is-active --quiet avahi-daemon.service"

t "find: nada achado: listas vazias"
tres
eq '{"found":[],"legacy":[],"mdns":true,"legacyUnknown":false}' "$(run find)"

t "find: com o CUPS parado não procura nada"
zera
echo "scheduler is not running" >"$c/r"
out=$(run find)
eq '{"found":[],"legacy":[],"mdns":false,"legacyUnknown":false}' "$out"
not_called "ippfind"
not_called "lpinfo"

t "find: nome com aspas, acento e comando embutido vira texto; linha malformada é pulada"
tres
printf '%s\t%s\t%s\n' 'Sala "do" Zé $(touch '"$SBX"'/pwn6)' "ipp://sala.local/ipp/print" '`touch '"$SBX"'/pwn7`' >"$c/ippfind"
printf '%s\n' "linha sem tab" "só-nome"$'\t' >>"$c/ippfind"
out=$(run find)
ok jq -e . <<<"$out" >/dev/null
eq '[{"name":"Sala \"do\" Zé $(touch '"$SBX"'/pwn6)","uri":"ipp://sala.local/ipp/print","model":"`touch '"$SBX"'/pwn7`","kind":"rede","added":false}]' "$(jq -c .found <<<"$out")"
absent "$SBX/pwn6"
absent "$SBX/pwn7"

# ---------------------------------------------------------------- add

t "add <uri> <nome>: lpadmin com a fila de nome arrumado, o modo sem driver e o nome como descrição"
tres
acao add "ipp://localhost:60001/ipp/print" "Canon G3010 (USB)"
eq "lpadmin[-p][Canon_G3010_USB][-E][-v][ipp://localhost:60001/ipp/print][-m][everywhere][-D][Canon G3010 (USB)]" "$(args | grep '^lpadmin')"

t "add: a última linha é o status mais o resultado (key, ok, error, message, elevated, at)"
eq '["add:ipp://localhost:60001/ipp/print",true,"","impressora adicionada",false]' "$(campo '[.key, .ok, .error, .message, .elevated] | @json')"
eq true "$(campo '.at > 0')"
eq '[true,true,"",3]' "$(campo '[.installed, .running, .pending, (.printers | length)] | @json')"

t "add: sem recusa do CUPS, não aparece janela de senha nem aviso (a página conta)"
not_called "pkexec"
not_called "notify-send"

t "add: a pasta de estado é só do usuário e não sobra pid nem 'elevando'"
eq 700 "$(stat -c %a "$sd")"
absent "$sd/pid"
absent "$sd/elevando"

t "add: o status seguinte lembra o resultado em last"
eq '["add:ipp://localhost:60001/ipp/print",true]' "$(run status | jq -c '[.last.key, .last.ok]')"

t "add sem nome: a fila leva o nome da máquina no endereço, sem o .local"
tres
acao add "ipps://Nova-Impressora.local:631/ipp/print"
eq "lpadmin[-p][Nova-Impressora][-E][-v][ipps://Nova-Impressora.local:631/ipp/print][-m][everywhere][-D][Nova-Impressora]" "$(args | grep '^lpadmin')"

t "add: nome que já existe (mesmo trocando maiúsculas) ganha _2, _3..."
tres
acao add "ipp://outra.local/ipp/print" "EPSON-ET-2850"
has "$(args | grep '^lpadmin')" "lpadmin[-p][EPSON-ET-2850_2]"
: >"$SBX/args"
acao add "ipp://outra.local/ipp/print" "brother_hl-l2350dw"
has "$(args | grep '^lpadmin')" "lpadmin[-p][brother_hl-l2350dw_2]"
: >"$SBX/args"
echo "device for EPSON-ET-2850_2: ipp://x.local/ipp/print" >>"$c/v"
acao add "ipp://outra.local/ipp/print" "EPSON-ET-2850"
has "$(args | grep '^lpadmin')" "lpadmin[-p][EPSON-ET-2850_3]"

t "add: nome só de símbolos vira 'impressora'; acento e espaço viram '_'"
tres
acao add "ipp://x.local/ipp/print" "(!!!)"
has "$(args | grep '^lpadmin')" "lpadmin[-p][impressora][-E]"
: >"$SBX/args"
acao add "ipp://x.local/ipp/print" "Sala do Zé"
has "$(args | grep '^lpadmin')" "lpadmin[-p][Sala_do_Z][-E]"
has "$(args | grep '^lpadmin')" "[-D][Sala do Zé]"

t "add: nome e endereço hostis vão como argumentos, sem rodar"
tres
acao add 'ipp://x.local/$(touch${IFS}'"$SBX"'/pwn8)' '"; touch '"$SBX"'/pwn9; $(touch '"$SBX"'/pwn10) `touch '"$SBX"'/pwn11` -x HP'
eq true "$(campo .ok)"
has "$(args | grep '^lpadmin')" '[-v][ipp://x.local/$(touch${IFS}'"$SBX"'/pwn8)][-m][everywhere][-D]["; touch '"$SBX"'/pwn9; $(touch '"$SBX"'/pwn10) `touch '"$SBX"'/pwn11` -x HP]'
like "$(args | grep '^lpadmin')" '^lpadmin\[-p\]\[[A-Za-z0-9_-]+\]\[-E\]'
for n in 8 9 10 11; do absent "$SBX/pwn$n"; done

t "add: endereço que não é ipp:// nem ipps:// é recusado sem chamar o lpadmin"
tres
for uri in "usb://Samsung/ML-1660" "http://x.local:631/" "ipp://" "ipp://a b" "-x" "file:///etc/passwd"; do
  acao add "$uri"
  eq '[false,"endereço inválido: só ipp:// ou ipps://"]' "$(campo '[.ok, .error] | @json')" "add '$uri'"
  eq "add:$uri" "$(campo .key)" "add '$uri'"
done
not_called "lpadmin"

t "add: a impressora não responde: a frase diz para conferir a rede, sem janela de senha"
tres
echo 'lpadmin: Unable to connect to "X.local:631": Name or service not known' >"$c/erro-lpadmin"
acao add "ipp://X.local:631/ipp/print"
eq '[false,false]' "$(campo '[.ok, .elevated] | @json')"
has "$(campo .error)" "não deu para falar com a impressora"
eq "$(campo .error)" "$(campo .message)"
not_called "pkexec"

t "add: a impressora não aceita o modo sem driver"
echo 'lpadmin: Printer does not support required IPP attributes or document formats.' >"$c/erro-lpadmin"
acao add "ipp://X.local:631/ipp/print"
eq "a impressora não aceitou o modo sem driver (IPP Everywhere)" "$(campo .error)"

# ------------------------------------------------- a recusa e a janela de senha

t "o CUPS recusa: a ação é refeita uma vez pelo pkexec e sai como elevada"
tres
: >"$c/recusa"
acao remove "HP_LaserJet_M15w"
eq '["remove:HP_LaserJet_M15w",true,"","HP_LaserJet_M15w removida",true]' "$(campo '[.key, .ok, .error, .message, .elevated] | @json')"
eq "lpadmin[-x][HP_LaserJet_M15w]
pkexec[lpadmin][-x][HP_LaserJet_M15w]
lpadmin[-x][HP_LaserJet_M15w]" "$(args | grep '^lpadmin\|^pkexec')"

t "depois da janela de senha, sem o shell por perto, o resultado vai num aviso"
eq "notify-send[-a][Impressoras][-i][printer][Impressoras][HP_LaserJet_M15w removida]" "$(args | grep '^notify-send')"

t "com a janela de senha demorando, a primeira linha avisa: {\"elevating\":true}"
tres
: >"$c/recusa"
echo 0.6 >"$c/pkexec.espera"
out=$(run remove "HP_LaserJet_M15w" 2>/dev/null)
eq '{"elevating":true}' "$(head -n1 <<<"$out")"
eq 2 "$(wc -l <<<"$out")"
eq true "$(tail -n1 <<<"$out" | jq -r '.ok and .elevated')"

t "janela de senha fechada (126): nada mudou, e a frase diz isso"
tres
: >"$c/recusa"
echo 126 >"$c/pkexec.rc"
acao add "ipp://nova.local/ipp/print"
eq '[false,true,"a senha não foi dada: nada mudou"]' "$(campo '[.ok, .elevated, .error] | @json')"
eq "$(campo .error)" "$(campo .message)"
eq 1 "$(args | grep -c '^pkexec')"
eq 1 "$(args | grep -c '^lpadmin')"

t "autorização falhou (127): nada mudou"
echo 127 >"$c/pkexec.rc"
acao remove "HP_LaserJet_M15w"
eq "a autorização falhou: nada mudou" "$(campo .error)"

t "pela janela de senha o comando falha por outro motivo: a frase é a do comando"
tres
: >"$c/recusa"
echo 'lpadmin: Unable to connect to "X.local:631": Connection refused' >"$c/erro-lpadmin"
acao add "ipp://X.local:631/ipp/print"
eq '[false,true]' "$(campo '[.ok, .elevated] | @json')"
has "$(campo .error)" "não deu para falar com a impressora"

t "com o shell em repouso, depois da senha a folha dos Ajustes volta na página, sem aviso"
tres
: >"$c/recusa"
echo rest >"$c/qs-state"
res=$(MARE_PRINTERS_SHELL=$SBX/shell run remove "HP_LaserJet_M15w" 2>/dev/null | tail -n1)
eq true "$(campo .ok)"
has "$(args)" "qs[-p][$SBX/shell][ipc][call][mare][settings][impressoras]"
not_called "notify-send"

t "com os Ajustes já abertos, não reabre nem avisa"
tres
: >"$c/recusa"
echo settings >"$c/qs-state"
res=$(MARE_PRINTERS_SHELL=$SBX/shell run remove "HP_LaserJet_M15w" 2>/dev/null | tail -n1)
hasnt "$(args)" "[settings][impressoras]"
not_called "notify-send"

t "com outra coisa aberta no shell (ou o shell mudo), manda o aviso"
for estado in launcher ""; do
  tres
  : >"$c/recusa"
  printf '%s' "$estado" >"$c/qs-state"
  res=$(MARE_PRINTERS_SHELL=$SBX/shell run remove "HP_LaserJet_M15w" 2>/dev/null | tail -n1)
  hasnt "$(args)" "[settings][impressoras]" "estado '$estado'"
  called "notify-send -a Impressoras"
done

# ---------------------------------------------------------------- default, remove, test

t "default: lpoptions -d, sem pedir senha"
tres
acao default "Brother_HL-L2350DW"
eq '["default:Brother_HL-L2350DW",true,"Brother_HL-L2350DW é a padrão agora"]' "$(campo '[.key, .ok, .message] | @json')"
eq "lpoptions[-d][Brother_HL-L2350DW]" "$(args | grep '^lpoptions')"
not_called "pkexec"

t "remove: lpadmin -x na fila"
tres
acao remove "EPSON-ET-2850"
eq '[true,"EPSON-ET-2850 removida"]' "$(campo '[.ok, .message] | @json')"
eq "lpadmin[-x][EPSON-ET-2850]" "$(args | grep '^lpadmin')"

t "fila que não existe (ou hostil): 'não há impressora chamada', e nenhum comando roda"
tres
for fila in "nao-existe" "epson-et-2850" "EPSON" '$(touch '"$SBX"'/pwn12)' "-x" "EPSON-ET-2850 -x Brother_HL-L2350DW" ".*"; do
  for verbo in remove default test cancel resume; do
    [[ $verbo == remove || $fila == "nao-existe" ]] || continue
    acao "$verbo" "$fila"
    eq '[false,"não há impressora chamada '"${fila//\"/\\\"}"'"]' "$(campo '[.ok, .error] | @json')" "$verbo '$fila'"
    eq 3 "$(campo '.printers | length')" "$verbo '$fila'"
  done
done
absent "$SBX/pwn12"
for cmd in lpadmin lpoptions "lp " cancel cupsenable cupsaccept pkexec; do not_called "$cmd"; done

t "fila com nome hostil que existe de verdade: vai como um argumento só, até pela janela de senha"
tres
ruim='$(touch${IFS}'"$SBX"'/pwn13);x'
echo "device for $ruim: ipp://x.local/ipp/print" >>"$c/v"
: >"$c/recusa"
acao remove "$ruim"
eq true "$(campo .ok)"
has "$(args)" "pkexec[lpadmin][-x][$ruim]"
: >"$SBX/args"
acao resume "$ruim"
eq true "$(campo .ok)"
has "$(args)" "cupsenable[$ruim]"
has "$(args)" "cupsaccept[$ruim]"
absent "$SBX/pwn13"

t "test: manda a página de teste do CUPS para a fila"
tres
acao test "Brother_HL-L2350DW"
eq '[true,"página de teste enviada para Brother_HL-L2350DW"]' "$(campo '[.ok, .message] | @json')"
eq "lp[-d][Brother_HL-L2350DW][-t][página de teste][$dados/default-testpage.pdf]" "$(args | grep '^lp\[')"

t "test: sem o PDF, usa o testprint; sem nenhum dos dois, avisa que não está instalada"
tres
rm "$dados/default-testpage.pdf"
: >"$dados/testprint"
acao test "Brother_HL-L2350DW"
has "$(args | grep '^lp\[')" "[$dados/testprint]"
rm "$dados/testprint"
: >"$SBX/args"
acao test "Brother_HL-L2350DW"
eq '[false,"a página de teste do CUPS não está instalada"]' "$(campo '[.ok, .error] | @json')"
eq "" "$(args | grep '^lp\[')"

t "test: o lp falha com algo novo: 'não deu certo:' e a última linha dele"
tres
printf '%s\n' "lp: aviso qualquer" "" "lp: algo novo" >"$c/erro-lp"
acao test "Brother_HL-L2350DW"
eq "não deu certo: lp: algo novo" "$(campo .error)"

# ---------------------------------------------------------------- cancel e resume

t "cancel: os trabalhos de quem clicou saem um a um, sem administrador"
tres
acao cancel "EPSON-ET-2850"
eq '[true,"fila de EPSON-ET-2850 cancelada"]' "$(campo '[.ok, .message] | @json')"
eq "cancel[EPSON-ET-2850-42]
cancel[EPSON-ET-2850-43]" "$(args | grep '^cancel')"
not_called "pkexec"

t "cancel: trabalho de outra pessoa pede 'cancel -a', que é de administrador"
tres
acao cancel "HP_LaserJet_M15w"
eq "cancel[-a][HP_LaserJet_M15w]" "$(args | grep '^cancel')"
eq false "$(campo .elevated)"

t "cancel: o CUPS recusa o 'cancel -a': vai pela janela de senha"
tres
: >"$c/recusa"
acao cancel "HP_LaserJet_M15w"
eq '[true,true]' "$(campo '[.ok, .elevated] | @json')"
has "$(args)" "pkexec[cancel][-a][HP_LaserJet_M15w]"

t "cancel: fila vazia não chama nada e dá certo"
tres
acao cancel "Brother_HL-L2350DW"
eq true "$(campo .ok)"
eq "" "$(args | grep '^cancel')"

t "cancel: se um dos próprios trabalhos falha, para e conta o erro"
tres
echo "cancel: cancel-job failed: Job #42 is already completed - can't cancel." >"$c/erro-cancel"
acao cancel "EPSON-ET-2850"
eq false "$(campo .ok)"
has "$(campo .error)" "não deu certo: cancel: cancel-job failed"
eq 1 "$(args | grep -c '^cancel')"

t "resume: cupsenable e cupsaccept na fila"
tres
acao resume "HP_LaserJet_M15w"
eq '[true,"HP_LaserJet_M15w voltou a imprimir",false]' "$(campo '[.ok, .message, .elevated] | @json')"
eq "cupsenable[HP_LaserJet_M15w]
cupsaccept[HP_LaserJet_M15w]" "$(args | grep '^cups')"

t "resume: recusado, os dois comandos passam por uma janela de senha só"
tres
: >"$c/recusa"
acao resume "HP_LaserJet_M15w"
eq '[true,true]' "$(campo '[.ok, .elevated] | @json')"
eq 1 "$(args | grep -c '^pkexec')"
eq 2 "$(args | grep -c '^cupsenable')"
eq 1 "$(args | grep -c '^cupsaccept')"

t "resume: falha que não é de permissão não abre a janela de senha"
tres
echo "cupsenable: Operation failed: client-error-not-possible" >"$c/erro-cupsenable"
acao resume "HP_LaserJet_M15w"
eq false "$(campo .ok)"
not_called "pkexec"
not_called "cupsaccept"

# ---------------------------------------------------------------- enable

t "enable: liga o cups.socket pelo pkexec, numa janela só, e avisa da janela antes"
zera
echo "scheduler is not running" >"$c/r"
echo 0.6 >"$c/pkexec.espera"
out=$(run enable 2>/dev/null)
eq '{"elevating":true}' "$(head -n1 <<<"$out")"
res=$(tail -n1 <<<"$out")
eq '["enable",true,"","impressão ligada",true,true]' "$(campo '[.key, .ok, .error, .message, .elevated, .running] | @json')"
eq "pkexec[systemctl][enable][--now][cups.socket]" "$(args | grep '^pkexec')"
eq 1 "$(args | grep -c '^pkexec')"

t "enable: janela de senha fechada: nada mudou"
zera
echo "scheduler is not running" >"$c/r"
echo 126 >"$c/pkexec.rc"
acao enable
eq '[false,"a senha não foi dada: nada mudou",false]' "$(campo '[.ok, .error, .running] | @json')"
eq "" "$(args | grep '^systemctl\[enable')"

t "enable: o systemctl falha: 'não deu para ligar o serviço de impressão'"
zera
echo "scheduler is not running" >"$c/r"
echo 1 >"$c/systemctl.rc"
acao enable
eq '[false,"não deu para ligar o serviço de impressão"]' "$(campo '[.ok, .error] | @json')"

t "enable: o serviço liga mas o CUPS não responde: avisa disso"
zera
echo "scheduler is not running" >"$c/r"
stub systemctl "$registra"'[[ $1 == enable ]]'
acao enable
eq '[false,"o serviço foi ligado, mas o CUPS não respondeu"]' "$(campo '[.ok, .error] | @json')"

# ---------------------------------------------------------------- o resto

t "outra ação em curso: responde na hora que está ocupado, sem rodar nada"
tres
mkdir -p "$sd"
echo "$$ add:ipp://x.local/ipp/print" >"$sd/pid"
acao remove "HP_LaserJet_M15w"
eq '["remove:HP_LaserJet_M15w",false,"outra ação ainda está em curso","outra ação ainda está em curso",false,0,"add:ipp://x.local/ipp/print"]' \
  "$(campo '[.key, .ok, .error, .message, .elevated, .at, .pending] | @json')"
eq 3 "$(campo '.printers | length')"
not_called "lpadmin"
rm "$sd/pid"

t "o pid de uma ação que morreu não trava a próxima"
echo "$morto add:ipp://x.local/ipp/print" >"$sd/pid"
acao remove "HP_LaserJet_M15w"
eq true "$(campo .ok)"
called "lpadmin -x HP_LaserJet_M15w"

t "sem pkexec no sistema: o CUPS recusou e não há como pedir a senha"
tres
: >"$c/recusa"
mkdir -p "$SBX/limpo"
for cmd in bash jq date sed awk grep head tail cut timeout setsid sleep rm mv mkdir chmod mktemp cat id tr; do
  ln -sf "$(command -v "$cmd")" "$SBX/limpo/$cmd"
done
mv "$SBX/bin/pkexec" "$SBX/pkexec.guardado"
res=$(PATH=$SBX/rapido:$SBX/bin:$SBX/limpo MARE_PRINTERS_ROOT=$root bash "$prn_sh" remove "HP_LaserJet_M15w" 2>/dev/null | tail -n1)
mv "$SBX/pkexec.guardado" "$SBX/bin/pkexec"
eq '[false,false,"o CUPS recusou e não há pkexec para pedir a senha"]' "$(campo '[.ok, .elevated, .error] | @json')"
eq 1 "$(args | grep -c '^lpadmin')"

t "sem as ferramentas do CUPS no PATH: status diz que não está rodando, com JSON válido"
mkdir "$SBX/so-systemctl"
cp "$SBX/bin/systemctl" "$SBX/so-systemctl/"
out=$(PATH=$SBX/so-systemctl:$SBX/limpo MARE_PRINTERS_ROOT=$root bash "$prn_sh" status)
eq 0 $?
eq '[true,false,[]]' "$(jq -c '[.installed, .running, .printers]' <<<"$out")"

t "frases de erro: cada resposta das ferramentas do CUPS tem a sua"
while IFS='|' read -r rc texto frase; do
  eq "$frase" "$(run __explain "$rc" "$texto")" "$texto"
done <<'FIM'
124||demorou demais e foi interrompido
1|cancel: cancel-jobs failed: Forbidden|o CUPS recusou: sem permissão para administrar as impressoras
1|lpadmin: Unauthorized|o CUPS recusou: sem permissão para administrar as impressoras
1|Password for root on localhost?|o CUPS recusou: sem permissão para administrar as impressoras
1|lpstat: Scheduler is not running.|o serviço de impressão não respondeu
1|lpadmin: Bad file descriptor|o serviço de impressão não respondeu
1|lpadmin: Unable to connect to "X.local:631": Name or service not known|não deu para falar com a impressora: ela está ligada e na mesma rede?
1|lpadmin: Unable to connect to host: No route to host|não deu para falar com a impressora: ela está ligada e na mesma rede?
1|lpadmin: Printer does not support required IPP attributes or document formats.|a impressora não aceitou o modo sem driver (IPP Everywhere)
1|lpadmin: Unable to create PPD file|a impressora não aceitou o modo sem driver (IPP Everywhere)
1|lp: algo novo|não deu certo: lp: algo novo
1||o comando comando falhou
FIM

t "ação sem o argumento dela, sem subcomando ou subcomando desconhecido: uso, sai com 2"
tres
for linha in "" "add" "remove" "default" "test" "cancel" "resume" "bobagem" "remove|"; do
  IFS='|' read -r -a argv <<<"$linha"
  [[ $linha == "remove|" ]] && argv=(remove "")
  err=$(run "${argv[@]}" 2>&1 >/dev/null)
  eq 2 $? "saída de '$linha'"
  has "$err" "uso: mare-printers status | find | add <uri> [nome]" "'$linha'"
done
eq "" "$(args)"
absent "$sd"

t "selftest: a leitura com as saídas de exemplo (MARE_PRINTERS_FIXTURES) passa inteira"
zera
out=$(bash "$prn_sh" selftest 2>&1)
eq 0 $?
has "$out" "tudo certo"
hasnt "$out" "FALHA"
eq "" "$(args)"
