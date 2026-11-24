# Arnês dos testes do Mk. Cada tests/**/*.test.sh começa com:
#
#   . "$(dirname "$0")/../lib.sh"
#
# e daí em diante roda numa caixa de areia: $SBX é uma pasta temporária só
# deste arquivo (apagada no fim), HOME e os XDG_* apontam para dentro dela e a
# sessão gráfica de quem roda fica invisível. Nada do sistema de verdade é
# tocado: os comandos que mexem na máquina (sudo, systemctl, pacman, hyprctl,
# nmcli...) estão trocados por guardas que falham e reprovam o teste. Um teste
# que precisa de um deles declara o que ele responde, com `stub`.
#
#   t "o que se espera"        abre um caso; as verificações abaixo são dele
#   eq <esperado> <obtido>     iguais
#   has <texto> <trecho>       o texto contém o trecho
#   hasnt <texto> <trecho>     o texto não contém o trecho
#   like <texto> <regex>       o texto casa com a regex (bash, =~)
#   ok <comando...>            o comando sai com 0
#   fails <comando...>         o comando sai diferente de 0
#   exists <caminho> / absent <caminho>
#   stub <nome> [corpo bash]   põe um comando falso no PATH; cada chamada vira
#                              uma linha "<nome> <args>" em $SBX/calls
#   called <trecho>            alguma chamada registrada contém o trecho
#   not_called <trecho>        nenhuma contém
#   calls_reset                esvazia o registro
#   path_sem <cmd>...          imprime um PATH em que esses comandos não existem
#                              (nem guarda, nem stub, nem o de verdade), para o
#                              ramo "falta a ferramenta": PATH=$(path_sem yay) ...
#   jeq <json> <json>          os dois JSON têm o mesmo valor (60.00 = 60)
#   skip <motivo>              sai deste arquivo sem reprovar (falta ferramenta)
#
# O resultado sai no fim, sozinho: uma linha por caso e o código de saída.
set -uo pipefail

TESTS=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO=$(cd "$TESTS/.." && pwd)
SBX=$(mktemp -d "${TMPDIR:-/tmp}/mk-test.XXXXXX")
export TESTS REPO SBX

mkdir -p "$SBX"/{bin,guard,home,run,tmp}
: >"$SBX/calls"
: >"$SBX/perigo"

export HOME=$SBX/home
export XDG_CONFIG_HOME=$HOME/.config XDG_CACHE_HOME=$HOME/.cache
export XDG_STATE_HOME=$HOME/.local/state XDG_DATA_HOME=$HOME/.local/share
export XDG_RUNTIME_DIR=$SBX/run TMPDIR=$SBX/tmp
export LC_ALL=C.UTF-8 TZ=UTC
unset HYPRLAND_INSTANCE_SIGNATURE WAYLAND_DISPLAY DISPLAY DBUS_SESSION_BUS_ADDRESS \
  SSH_AUTH_SOCK SUDO_USER SUDO_UID NIRI_SOCKET SWAYSOCK

# Os guardas: o que nunca pode rodar de verdade num teste.
for _c in sudo su pkexec doas systemctl loginctl journalctl busctl gdbus dbus-send \
  hyprctl Hyprland hyprlock hypridle hyprpicker qs quickshell wtype wl-copy wl-paste \
  grim slurp wf-recorder notify-send uwsm \
  pacman pacman-key yay paru makepkg pacstrap arch-chroot mkarchiso \
  nmcli iwctl bluetoothctl rfkill ufw ip \
  wpctl pactl playerctl brightnessctl \
  lp lpadmin lpstat lpinfo lpoptions cupsenable cupsaccept cancel ippfind qrencode \
  limine limine-install limine-update limine-mkinitcpio limine-snapper-sync \
  limine-entry-tool limine-enroll-config efibootmgr mkinitcpio bootctl \
  snapper btrfs swapon swapoff mkswap mount umount wipefs sgdisk parted \
  mkfs.btrfs mkfs.fat mkfs.ext4 cryptsetup dd chattr chown \
  useradd usermod userdel passwd chpasswd groupadd \
  reboot poweroff shutdown halt \
  curl wget git gh ssh scp rsync \
  xdg-mime xdg-open xdg-settings gsettings dconf gio \
  mise npm pip docker flatpak \
  checkupdates timedatectl matugen cava pkill killall gpu-screen-recorder ss setpriv \
  qemu-system-x86_64 qemu-img repo-add \
  lsblk findmnt partprobe udevadm genfstab blkid mountpoint lspci systemd-detect-virt \
  localectl hwclock gum update-desktop-database gtk-update-icon-cache; do
  cat >"$SBX/guard/$_c" <<GUARD
#!/bin/bash
printf '%s\n' "$_c \$*" >>"$SBX/perigo"
echo "PERIGO: o teste chamou '$_c \$*' sem stub" >&2
exit 97
GUARD
  chmod +x "$SBX/guard/$_c"
done
unset _c
export PATH=$SBX/bin:$SBX/guard:$PATH

_file=${0#"$REPO"/}
_case=""
_n=0
_bad=0
_case_bad=0

_close() {
  [[ -n $_case ]] || return 0
  if ((_case_bad)); then
    printf 'not ok - %s\n' "$_case"
    _bad=$((_bad + 1))
  else
    printf 'ok - %s\n' "$_case"
  fi
  _case=""
}

t() {
  _close
  _case=$1
  _case_bad=0
  _n=$((_n + 1))
}

_fail() {
  [[ -n $_case ]] || t "(fora de um caso)"
  _case_bad=1
  printf '  # %s\n' "$@" | sed 's/^/  /' >&2
}

eq() {
  [[ $1 == "$2" ]] || _fail "${3:-eq}: esperado" "    $1" "  obtido" "    $2"
}
has() {
  [[ $1 == *"$2"* ]] || _fail "${3:-has}: não achei" "    $2" "  em" "    ${1:0:600}"
}
hasnt() {
  [[ $1 != *"$2"* ]] || _fail "${3:-hasnt}: não devia ter" "    $2" "  em" "    ${1:0:600}"
}
like() {
  [[ $1 =~ $2 ]] || _fail "${3:-like}: não casa com /$2/" "    ${1:0:600}"
}
ok() {
  "$@" || _fail "ok: saiu com $? : $*"
}
fails() {
  if "$@"; then _fail "fails: devia falhar: $*"; fi
}
exists() { [[ -e $1 ]] || _fail "exists: falta $1"; }
absent() { [[ ! -e $1 && ! -L $1 ]] || _fail "absent: não devia existir $1"; }

stub() {
  local name=$1 body=${2:-}
  cat >"$SBX/bin/$name" <<STUB
#!/bin/bash
printf '%s\n' "$name \$*" >>"$SBX/calls"
$body
STUB
  chmod +x "$SBX/bin/$name"
}
called() {
  grep -qF -- "$1" "$SBX/calls" || _fail "called: ninguém chamou" "    $1" "  chamadas:" "$(sed 's/^/    /' "$SBX/calls" | head -30)"
}
not_called() {
  if grep -qF -- "$1" "$SBX/calls"; then _fail "not_called: chamaram" "    $(grep -F -- "$1" "$SBX/calls" | head -3)"; fi
}
calls_reset() { : >"$SBX/calls"; }

# Um PATH só de atalhos: tudo o que o PATH de agora tem, menos os pedidos.
# Stubs e guardas continuam valendo para o resto.
path_sem() {
  local dir d f n
  dir=$(mktemp -d "$SBX/sem.XXXXXX")
  local IFS=:
  for d in $PATH; do
    [[ -d $d ]] || continue
    for f in "$d"/*; do
      n=${f##*/}
      [[ -x $f && ! -e $dir/$n ]] || continue
      [[ " $* " == *" $n "* ]] && continue
      ln -s "$f" "$dir/$n"
    done
  done
  for n in "$@"; do rm -f "$dir/$n"; done
  printf '%s\n' "$dir"
}

jeq() {
  local a b
  a=$(jq -cS . <<<"$1" 2>/dev/null) || { _fail "jeq: o esperado não é JSON" "    ${1:0:300}"; return; }
  b=$(jq -cS . <<<"$2" 2>/dev/null) || { _fail "jeq: o obtido não é JSON" "    ${2:0:300}"; return; }
  [[ $(jq -n --argjson a "$a" --argjson b "$b" '$a == $b') == true ]] ||
    _fail "${3:-jeq}: esperado" "    $a" "  obtido" "    $b"
}

skip() {
  _case=""
  printf 'skip - %s: %s\n' "$_file" "$1"
  trap - EXIT
  rm -rf "${SBX:?}"
  exit 0
}

_finish() {
  local rc=$?
  _close
  if [[ -s $SBX/perigo ]]; then
    printf 'not ok - nenhum comando do sistema sem stub\n'
    sed 's/^/    # /' "$SBX/perigo" | sort -u >&2
    _bad=$((_bad + 1))
  fi
  # Um arquivo que morreu no meio (set -e, erro de sintaxe) também reprova.
  if ((rc != 0)); then
    printf 'not ok - %s terminou com %s antes do fim\n' "$_file" "$rc"
    _bad=$((_bad + 1))
  fi
  rm -rf "${SBX:?}"
  ((_n > 0)) || { printf 'not ok - %s não tem nenhum caso\n' "$_file"; exit 1; }
  ((_bad == 0)) || exit 1
  exit 0
}
trap _finish EXIT
