#!/usr/bin/env bash
# shell/mare/scripts/mare-gamepad: o estado do controle Xbox e do xone (status)
# e o que install/remove pedem ao yay e ao pacman, com um /sys de mentira.
. "$(dirname "$0")/../lib.sh"

pad_sh=$REPO/shell/mare/scripts/mare-gamepad
sys=$SBX/sys
usb=$sys/bus/usb/devices

# Um PATH sem os guardas: só os stubs e o jq, para o caso "não há yay".
bash_bin=$(command -v bash)
mkdir -p "$SBX/limpo"
ln -s "$(command -v jq)" "$SBX/limpo/jq"

dev() { # dev <porta> <idVendor> <produto>
  mkdir -p "$usb/$1"
  echo "$2" >"$usb/$1/idVendor"
  printf '%s\n' "$3" >"$usb/$1/product"
}
zera() {
  rm -rf "${sys:?}"
  mkdir -p "$usb" "$sys/module"
  calls_reset
}
run() { SYS=$sys bash "$pad_sh" "$@"; }

t "status: sem xone, sem controle, com yay"
zera
stub pacman 'exit 1'
out=$(run status)
eq 0 $?
eq '{"installed":false,"loaded":false,"yay":true,"controllers":[]}' "$out"
called "pacman -Qq xone-dkms"

t "sem subcomando é status"
eq '{"installed":false,"loaded":false,"yay":true,"controllers":[]}' "$(run)"

t "status: o pacote instalado vira installed, sem mudar nada"
zera
stub pacman 'echo xone-dkms'
out=$(run status)
eq true "$(jq -r .installed <<<"$out")"
eq false "$(jq -r .loaded <<<"$out")"
not_called "pacman -S"
not_called "pacman -R"
not_called "sudo"

t "status: qualquer módulo do xone carregado (gip, wired ou dongle) vira loaded"
for mod in xone_gip xone_wired xone_dongle; do
  zera
  mkdir -p "$sys/module/$mod"
  eq true "$(run status | jq -r .loaded)" "$mod"
done
zera
mkdir -p "$sys/module/xpad"
eq false "$(run status | jq -r .loaded)" "só o xpad"

t "status: lista os controles da Microsoft (045e), pelo nome do produto"
zera
dev 1-1 045e "Controller"
dev 1-2 045e "Xbox Wireless Controller"
dev 1-3 045e "XBOX ACC"
out=$(run status)
eq '["Controller","Xbox Wireless Controller","XBOX ACC"]' "$(jq -c .controllers <<<"$out")"

t "status: controle de outra marca e mouse da Microsoft não entram"
zera
dev 1-1 054c "DualSense Wireless Controller"
dev 1-2 046d "USB Receiver"
dev 1-3 045e "Microsoft Wireless Optical Mouse"
out=$(run status)
eq 0 $?
eq '[]' "$(jq -c .controllers <<<"$out")"

t "status: sai com 0 mesmo quando o último dispositivo USB é da Microsoft e não é controle"
zera
dev 1-1 045e "Controller"
dev 9-9 045e "Microsoft Wireless Optical Mouse"
out=$(run status)
eq 0 $?
eq '["Controller"]' "$(jq -c .controllers <<<"$out")"

t "status: dispositivo sem idVendor ou sem product (hub, interface) é pulado"
zera
mkdir -p "$usb/usb1" "$usb/1-0:1.0"
echo 045e >"$usb/usb1/idVendor"
echo "Xbox Controller" >"$usb/1-0:1.0/product"
dev 1-4 045e "Controller"
eq '["Controller"]' "$(run status | jq -c .controllers)"

t "status: nome de produto com aspas, acento e comando embutido vira texto no JSON"
zera
dev 1-1 045e 'Xbox "Elite" Sério $(touch '"$SBX"'/pwn)`touch '"$SBX"'/pwn2`'
out=$(run status)
ok jq -e . <<<"$out" >/dev/null
eq 'Xbox "Elite" Sério $(touch '"$SBX"'/pwn)`touch '"$SBX"'/pwn2`' "$(jq -r '.controllers[0]' <<<"$out")"
absent "$SBX/pwn"
absent "$SBX/pwn2"

t "status: sem a pasta de USB no /sys, lista vazia e JSON válido"
rm -rf "${sys:?}"
out=$(run status)
eq 0 $?
eq '[]' "$(jq -c .controllers <<<"$out")"

t "status: sem yay e sem pacman no PATH, yay:false e installed:false"
zera
dev 1-1 045e "Controller"
out=$(SYS=$sys PATH=$SBX/limpo "$bash_bin" "$pad_sh" status)
eq 0 $?
eq '{"installed":false,"loaded":false,"yay":false,"controllers":["Controller"]}' "$out"

t "install: pede ao yay o xone-dkms com os cabeçalhos e o dkms, sem reinstalar"
zera
stub yay
out=$(run install <<<"")
called "yay -S --needed linux-headers dkms xone-dkms"
not_called "sudo"
not_called "pacman"

t "install: sem yay avisa que falta, sai com 1 e não instala nada"
zera
out=$(SYS=$sys PATH=$SBX/limpo "$bash_bin" "$pad_sh" install <<<"" 2>&1)
eq 1 $?
has "$out" "falta o yay"
not_called "yay"
not_called "pacman"

t "remove: tira o xone-dkms pelo pacman, com sudo"
zera
stub sudo
out=$(run remove <<<"")
called "sudo pacman -Rns xone-dkms"
not_called "yay"

# Roda install/remove com o Enter preso num fifo: o que estiver em $SBX/tela
# quando o aviso aparece foi escrito antes de o script parar no "Enter fecha".
# Só depois o Enter é dado. Deixa a saída do script em $rc.
com_enter_preso() { # com_enter_preso <subcomando> <trecho esperado antes do Enter>
  local pid antes=0
  rm -f "$SBX/enter" "$SBX/tela"
  mkfifo "$SBX/enter"
  run "$1" <"$SBX/enter" >"$SBX/tela" 2>&1 &
  pid=$!
  exec 9>"$SBX/enter"
  for _ in $(seq 1 50); do
    grep -qF -- "$2" "$SBX/tela" 2>/dev/null && { antes=1; break; }
    kill -0 "$pid" 2>/dev/null || break
    sleep 0.1
  done
  eq 1 "$antes" "'$2' na tela antes do Enter"
  echo >&9
  exec 9>&-
  wait "$pid"
  rc=$?
}

t "install: o yay falhando sai com o código do yay, não com o do Enter"
zera
stub yay 'exit 7'
out=$(run install <<<"" 2>&1)
eq 7 $?
has "$out" "falhou"
hasnt "$out" "driver novo"

t "install: a falha aparece na tela antes do 'Enter fecha'"
zera
stub yay 'exit 7'
com_enter_preso install "falhou"
eq 7 "$rc"
has "$(<"$SBX/tela")" "código 7"

t "install: o yay dando certo sai com 0 e não fala em falha"
zera
stub yay
out=$(run install <<<"" 2>&1)
eq 0 $?
hasnt "$out" "falhou"

t "install: sem ninguém para dar o Enter (entrada fechada), a saída ainda é a do yay"
zera
stub yay
run install </dev/null >/dev/null 2>&1
eq 0 $?
stub yay 'exit 5'
run install </dev/null >/dev/null 2>&1
eq 5 $?

t "remove: o pacman falhando (ou a senha do sudo recusada) sai com o código dele e avisa antes do Enter"
zera
stub sudo 'exit 3'
com_enter_preso remove "falhou"
eq 3 "$rc"
has "$(<"$SBX/tela")" "código 3"
stub sudo
out=$(run remove <<<"" 2>&1)
eq 0 $?
hasnt "$out" "falhou"
run remove </dev/null >/dev/null 2>&1
eq 0 $?

t "install: sem yay e sem ninguém para dar o Enter, ainda sai com 1"
zera
out=$(SYS=$sys PATH=$SBX/limpo "$bash_bin" "$pad_sh" install </dev/null 2>&1)
eq 1 $?
has "$out" "falta o yay"

t "subcomando desconhecido: uso, sai com 2 e não chama nada"
zera
err=$(run bobagem 2>&1 >/dev/null)
eq 2 $?
has "$err" "uso: mare-gamepad status | install | remove"
not_called "pacman"
not_called "yay"
not_called "sudo"
