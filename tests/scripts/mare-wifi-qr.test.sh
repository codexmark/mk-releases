#!/usr/bin/env bash
# shell/mare/scripts/mare-wifi-qr: o texto que vai para o QR (formato WIFI:),
# a rede aberta, a rede com senha e os avisos, com ip, nmcli e qrencode falsos.
. "$(dirname "$0")/../lib.sh"

qr_sh=$REPO/shell/mare/scripts/mare-wifi-qr
sys=$SBX/sys nm=$SBX/nm png=$SBX/run/qr.png
mkdir -p "$sys/class/net/wlan0/wireless" "$sys/class/net/enp3s0" "$nm"

# O nmcli de mentira responde como o de verdade: com -g os ":" e as "\" do
# valor saem escapados, a não ser com "-e no". O nome da conexão tem de chegar
# inteiro, senão é "no such connection profile" (saída 10).
#   $nm/conn  nome da conexão do wlan0     $nm/ssid, $nm/keymgmt, $nm/psk
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
  *" GENERAL.CONNECTION dev show "*) val conn; exit 0 ;;
esac
if [[ " $* " == *" con show "* ]]; then
  [[ -n $last && $last == "$(<"$nm/conn")" ]] || { echo "Error: $last - no such connection profile." >&2; exit 10; }
  case " $* " in
    *" 802-11-wireless.ssid "*) val ssid ;;
    *" 802-11-wireless-security.key-mgmt "*) val keymgmt ;;
    *" 802-11-wireless-security.psk "*)
      # Sem -s a senha vem escondida.
      if [[ " $* " == *" -s "* ]]; then val psk; else echo "<hidden>"; fi ;;
  esac
  exit 0
fi
exit 2'
stub ip 'echo "1.1.1.1 via 192.168.1.1 dev wlan0 src 192.168.1.50 uid 1000"; echo "    cache"'
# O qrencode de mentira grava o texto que recebeu (o último argumento) no
# lugar do PNG.
stub qrencode '
out="" prev=""
for a in "$@"; do [[ $prev == -o ]] && out=$a; prev=$a; done
printf "%s" "${!#}" >"$out"'

rede() { # rede <conexão> <ssid> <key-mgmt> <senha>
  printf '%s\n' "$1" >"$nm/conn"
  printf '%s\n' "$2" >"$nm/ssid"
  printf '%s\n' "$3" >"$nm/keymgmt"
  printf '%s\n' "$4" >"$nm/psk"
  rm -f "$png"
  calls_reset
  : >"$SBX/args"
}
run() { SYS=$sys bash "$qr_sh" "$@"; }

t "rede com senha: WIFI:T:WPA com o SSID e a senha, e imprime o nome da rede"
rede "Casa" "Casa" wpa-psk "segredo123"
out=$(run "$png")
eq 0 $?
eq "Casa" "$out"
eq 'WIFI:T:WPA;S:Casa;P:segredo123;;' "$(<"$png")"
called "qrencode -t PNG"
called "-o $png"

t "rede com senha: WPA3 (sae) também sai como WPA"
rede "Casa" "Casa" sae "segredo123"
ok run "$png" >/dev/null
eq 'WIFI:T:WPA;S:Casa;P:segredo123;;' "$(<"$png")"

t "rede aberta (sem key-mgmt): WIFI:T:nopass, sem campo de senha"
rede "Cafe" "Cafe" "" ""
: >"$nm/keymgmt"
out=$(run "$png")
eq "Cafe" "$out"
eq 'WIFI:T:nopass;S:Cafe;;' "$(<"$png")"

t "rede aberta com key-mgmt 'none': também nopass"
rede "Cafe" "Cafe" none ""
ok run "$png" >/dev/null
eq 'WIFI:T:nopass;S:Cafe;;' "$(<"$png")"

t "SSID e senha com ; , : \" e barra ganham a barra do formato WIFI:"
rede "Casa" 'a;b,c:d"e\f' wpa-psk 'p;1,2:3"4\5'
out=$(run "$png")
eq 'a;b,c:d"e\f' "$out"
eq 'WIFI:T:WPA;S:a\;b\,c\:d\"e\\f;P:p\;1\,2\:3\"4\\5;;' "$(<"$png")"

t "SSID com espaço e acento passa inteiro, num argumento só"
rede "Café da Maré 5G" "Café da Maré 5G" wpa-psk "senha com espaço"
out=$(run "$png")
eq "Café da Maré 5G" "$out"
eq 'WIFI:T:WPA;S:Café da Maré 5G;P:senha com espaço;;' "$(<"$png")"
has "$(<"$SBX/args")" "[show][Café da Maré 5G]"

t "conexão com dois-pontos no nome: o nmcli recebe o nome sem barra de escape"
rede "Casa: 5G" "Casa: 5G" wpa-psk "s:1"
out=$(run "$png")
eq 0 $?
eq "Casa: 5G" "$out"
eq 'WIFI:T:WPA;S:Casa\: 5G;P:s\:1;;' "$(<"$png")"
has "$(<"$SBX/args")" "[show][Casa: 5G]"

t "o PNG (tem a senha) nasce legível só pelo dono"
rede "Casa" "Casa" wpa-psk "segredo123"
ok run "$png" >/dev/null
eq 600 "$(stat -c %a "$png")"

t "caminho do PNG com espaço chega inteiro ao qrencode"
rede "Casa" "Casa" wpa-psk "segredo123"
mkdir -p "$SBX/run/pasta com espaço"
ok run "$SBX/run/pasta com espaço/qr.png" >/dev/null
exists "$SBX/run/pasta com espaço/qr.png"

t "a senha é pedida com -s e não aparece na saída"
rede "Casa" "Casa" wpa-psk "segredo123"
out=$(run "$png" 2>&1)
hasnt "$out" "segredo123"
called "nmcli -s "

t "SSID e senha hostis não são executados"
rede "Casa" '$(touch '"$SBX"'/pwn1)`touch '"$SBX"'/pwn2`' wpa-psk '"; touch '"$SBX"'/pwn3; "'
ok run "$png" >/dev/null
absent "$SBX/pwn1"
absent "$SBX/pwn2"
absent "$SBX/pwn3"
has "$(<"$png")" 'S:$(touch '

t "rede com segurança e sem acesso à senha: avisa, sai com 1 e não gera QR"
rede "Casa" "Casa" wpa-psk ""
: >"$nm/psk"
err=$(run "$png" 2>&1 >/dev/null)
eq 1 $?
has "$err" "sem acesso à senha"
not_called "qrencode"
absent "$png"

t "no cabo (interface sem wireless): 'não está num Wi-Fi', sai com 1"
rede "Casa" "Casa" wpa-psk "x"
stub ip 'echo "1.1.1.1 via 192.168.1.1 dev enp3s0 src 192.168.1.60 uid 1000"'
err=$(run "$png" 2>&1 >/dev/null)
eq 1 $?
has "$err" "não está num Wi-Fi"
not_called "nmcli"
not_called "qrencode"

t "sem rota (o ip sai com 2): o mesmo aviso, não um fim mudo"
calls_reset
stub ip 'echo "RTNETLINK answers: Network is unreachable" >&2; exit 2'
err=$(run "$png" 2>&1 >/dev/null)
eq 1 $?
has "$err" "não está num Wi-Fi"
not_called "nmcli"

t "resposta do ip sem 'dev': o mesmo aviso"
stub ip 'echo "bobagem sem interface"'
err=$(run "$png" 2>&1 >/dev/null)
eq 1 $?
has "$err" "não está num Wi-Fi"

t "interface sem conexão do NetworkManager: falha sem gerar QR"
stub ip 'echo "1.1.1.1 via 192.168.1.1 dev wlan0 src 192.168.1.50 uid 1000"'
rede "" "" "" ""
: >"$nm/conn"
fails run "$png" >/dev/null 2>&1
not_called "qrencode"

t "o qrencode falhando: sai diferente de 0 e não imprime o nome"
rede "Casa" "Casa" wpa-psk "segredo123"
stub qrencode 'echo "qrencode: falhou" >&2; exit 1'
out=$(run "$png" 2>/dev/null)
eq 1 $?
eq "" "$out"

t "sem argumento: uso, sai diferente de 0 e não chama nada"
calls_reset
err=$(run 2>&1 >/dev/null)
like "$?" '^[1-9]'
has "$err" "uso: mare-wifi-qr <arquivo.png>"
not_called "ip"
not_called "nmcli"
