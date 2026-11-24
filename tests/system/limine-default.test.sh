#!/usr/bin/env bash
# system/mk/boot/limine-default: acha a primeira entrada que inicia o Linux,
# abre os grupos acima dela e grava a posição em default_entry. Sempre com um
# limine.conf de mentira passado por argumento (nunca o de /boot).
. "$(dirname "$0")/../lib.sh"

ld=$REPO/system/mk/boot/limine-default
conf=$SBX/limine.conf
stub limine-enroll-config
padrao() { bash "$ld" "$conf"; }
entrada() { sed -n 's/^default_entry: //p' "$conf"; }
cabecalho='# Menu de boot do Mk (limine).
timeout: 3
interface_branding: Mk'
linux() { # linux <barras+nome>: uma entrada que inicia o Linux
  printf '%s\n    protocol: linux\n    path: boot():/vmlinuz-linux\n    cmdline: root=UUID=1 rw\n    module_path: boot():/initramfs-linux.img\n' "$1"
}
efi() { printf '%s\n    comment: Default EFI loader\n    protocol: efi\n    path: boot():/EFI/BOOT/BOOTX64.EFI\n' "$1"; }

# --- o menu fixo do instalador ----------------------------------------------
t "menu fixo (Mk e Mk (fallback)): a entrada padrão é a 1, gravada depois do timeout"
{
  echo "$cabecalho"
  echo
  linux "/Mk"
  echo
  linux "/Mk (fallback)"
} >"$conf"
out=$(padrao)
eq 0 $?
eq 1 "$(entrada)"
eq "default_entry: 1" "$(sed -n '/^timeout:/{n;p}' "$conf")"
has "$out" "inicia sozinho pela entrada 1"

t "menu fixo: não abre nada nem mexe nas entradas"
eq 0 "$(grep -c '^/+' "$conf")"
eq 2 "$(grep -c '^/Mk' "$conf")"
eq 2 "$(grep -c 'cmdline: root=UUID=1 rw' "$conf")"

t "rodar de novo não muda nada e diz que já está certo"
antes=$(<"$conf")
calls_reset
out=$(padrao)
eq 0 $?
eq "$antes" "$(<"$conf")"
has "$out" "já está certo (entrada 1)"

t "sem mudança, o limine.conf não é assinado de novo"
not_called "limine-enroll-config"

# --- o menu dos ganchos do limine -------------------------------------------
gancho() { # gancho [nome do grupo, com barra e + se aberto]
  echo "$cabecalho"
  echo
  echo "${1:-/Mk}"
  echo "    comment: Mk"
  echo "    comment: machine-id=0123456789abcdef order-priority=50"
  linux "    //linux" | sed '2,$s/^/    /'
  echo
  echo "    //Snapshots"
  echo "    comment: Selecione um snapshot"
  linux "        ///2026-11-20 10:00 | 12 | pacman -Syu" | sed '2,$s/^/        /'
  echo
  efi "/EFI fallback"
}

t "menu dos ganchos: o grupo Mk é aberto (/Mk vira /+Mk) e a padrão é a 2, o linux dentro dele"
gancho >"$conf"
calls_reset
out=$(padrao)
eq 0 $?
ok grep -qx '/+Mk' "$conf"
eq 2 "$(entrada)"
has "$out" "inicia sozinho pela entrada 2"

t "menu dos ganchos: o submenu de snapshots continua fechado"
ok grep -qx '    //Snapshots' "$conf"
eq 1 "$(grep -c '/+' "$conf")"

t "menu dos ganchos: só o nome do grupo e o default_entry mudam"
eq "$(gancho | sed 's|^/Mk$|/+Mk|')" "$(grep -v '^default_entry:' "$conf")"

t "com mudança, o limine.conf é assinado de novo (limine-enroll-config)"
called "limine-enroll-config"

t "menu dos ganchos: rodar de novo é igual (não vira /++Mk)"
antes=$(<"$conf")
calls_reset
padrao >/dev/null
eq "$antes" "$(<"$conf")"
not_called "limine-enroll-config"

t "grupo que já vem aberto: só o default_entry é gravado"
gancho "/+Mk" >"$conf"
padrao >/dev/null
eq 2 "$(entrada)"
eq 1 "$(grep -c '^/+Mk$' "$conf")"

# --- só entradas protocol: linux --------------------------------------------
t "a EFI fallback (protocol: efi) nunca é a padrão, mesmo vindo primeiro"
{
  echo "$cabecalho"
  efi "/EFI fallback"
  linux "/Mk"
} >"$conf"
padrao >/dev/null
eq 2 "$(entrada)"

t "sem nenhuma entrada protocol: linux, não mexe no arquivo e sai sem erro"
{
  echo "$cabecalho"
  efi "/EFI fallback"
  printf '/Windows\n    protocol: efi_chainload\n    image_path: boot():/EFI/Microsoft/Boot/bootmgfw.efi\n'
} >"$conf"
antes=$(<"$conf")
calls_reset
out=$(padrao)
eq 0 $?
eq "$antes" "$(<"$conf")"
has "$out" "nenhuma entrada que inicie"
not_called "limine-enroll-config"

t "um 'protocol: linux' num comentário ou noutra chave não conta"
{
  echo "$cabecalho"
  printf '/Windows\n    comment: não é protocol: linux\n    protocol: efi\n    path: boot():/x.efi\n'
  printf '/Outro\n    protocol: linuxboot\n'
} >"$conf"
out=$(padrao)
has "$out" "nenhuma entrada que inicie"
eq "" "$(entrada)"

t "um grupo com 'protocol: linux' mas com filhos não é o que inicia: vale o filho"
{
  echo "$cabecalho"
  printf '/Mk\n    protocol: linux\n'
  linux "//linux"
} >"$conf"
padrao >/dev/null
ok grep -qx '/+Mk' "$conf"
eq 2 "$(entrada)"

# --- a posição entre as entradas visíveis -----------------------------------
t "os filhos de um grupo fechado acima não contam na posição"
{
  echo "$cabecalho"
  echo "/Ferramentas"
  efi "//Memtest"
  efi "//Shell UEFI"
  linux "/Mk"
} >"$conf"
padrao >/dev/null
eq 2 "$(entrada)"
ok grep -qx '/Ferramentas' "$conf"

t "os filhos de um grupo aberto acima contam"
sed -i 's|^/Ferramentas$|/+Ferramentas|' "$conf"
padrao >/dev/null
eq 4 "$(entrada)"

t "dois níveis de grupo: os dois são abertos e a posição conta os dois"
{
  echo "$cabecalho"
  echo "/Mk"
  echo "    //Kernels"
  linux "        ///linux"
  linux "        ///linux-lts"
  efi "/EFI fallback"
} >"$conf"
padrao >/dev/null
ok grep -qx '/+Mk' "$conf"
ok grep -qx '    //+Kernels' "$conf"
eq 3 "$(entrada)"
eq 2 "$(grep -c '/+' "$conf")"

t "um grupo irmão, que não está acima da entrada, não é aberto"
{
  echo "$cabecalho"
  echo "/Antigo"
  efi "//Chainload"
  echo "/Mk"
  linux "//linux"
} >"$conf"
padrao >/dev/null
ok grep -qx '/Antigo' "$conf"
ok grep -qx '/+Mk' "$conf"
eq 3 "$(entrada)"

# --- onde o default_entry é gravado -----------------------------------------
t "default_entry errado é corrigido no lugar, sem duplicar"
{
  echo "timeout: 5"
  echo "default_entry: 7"
  linux "/Mk"
} >"$conf"
padrao >/dev/null
eq 1 "$(entrada)"
eq 1 "$(grep -c '^default_entry:' "$conf")"
eq "timeout: 5" "$(head -1 "$conf")"

t "sem a linha timeout, o default_entry vai para a primeira linha"
linux "/Mk" >"$conf"
padrao >/dev/null
eq "default_entry: 1" "$(head -1 "$conf")"
eq "/Mk" "$(sed -n 2p "$conf")"

t "última linha sem quebra de linha também é lida"
printf 'timeout: 3\n/Mk\n    protocol: linux' >"$conf"
padrao >/dev/null
eq 1 "$(entrada)"

# --- arquivo ausente ---------------------------------------------------------
t "arquivo que não existe: avisa e sai sem erro (o instalador segue)"
out=$(bash "$ld" "$SBX/nao-existe.conf")
eq 0 $?
has "$out" "limine.conf não encontrado"

t "o limine.conf que o instalador gera (cabeçalho + entradas fixas) fica com a entrada 1"
{
  sed -n "/^limine_header() {/,/^}/p" "$REPO/installer/mk-install" | sed -n "/<<'EOF'/,/^EOF/p" | sed '1d;$d'
  linux "/Mk"
} >"$conf"
ok grep -q '^timeout: 3' "$conf"
padrao >/dev/null
eq 1 "$(entrada)"

t "sem o limine-enroll-config instalado, grava do mesmo jeito"
# Um PATH só com o que o script usa: o limine desta máquina, se houver, não aparece.
mkdir -p "$SBX/minimo"
for c in bash sed awk grep; do ln -s "$(command -v "$c")" "$SBX/minimo/$c"; done
{ echo "$cabecalho"; linux "/Mk"; } >"$conf"
calls_reset
out=$(PATH=$SBX/minimo "$SBX/minimo/bash" "$ld" "$conf")
eq 0 $?
eq 1 "$(entrada)"
has "$out" "inicia sozinho pela entrada 1"
not_called "limine-enroll-config"
