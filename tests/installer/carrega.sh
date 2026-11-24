# Carrega o installer/mk-install como biblioteca, para testar uma função de
# cada vez. Use logo depois do tests/lib.sh:
#
#   . "$(dirname "$0")/../lib.sh"
#   . "$(dirname "$0")/carrega.sh"
#
# O instalador formata discos. Aqui ele nunca roda inteiro numa máquina de
# verdade: o `main` não é chamado (o arquivo só o chama quando é executado),
# $target, $log e $live_root apontam para dentro de $SBX, o disco é um nome
# que só existe num lsblk falso e todo comando que toca a máquina é guarda
# (reprova) ou stub.
#
#   roda <função> [args]     roda numa subcasca com `set -e`, como no
#                            instalador; a saída fica em $out, o código em $rc.
#                            A entrada é $entrada (padrão: /dev/null).
#   discos <<EOF             o que o lsblk falso enxerga, uma linha por
#                            dispositivo: nome tipo bytes tamanho modelo pai
#   raiz_falsa [dir]         o mínimo que o pacstrap deixaria em $target
#   fonte_falsa              copia o repositório para $SBX/src (sem tests/ e
#                            iso/), com um .git, um aur-repo e saídas do build
#                            de mentira, e aponta $src para lá
#   respostas [CHAVE=valor]  grava $SBX/respostas.conf a partir do
#                            installer/answers.example, trocando as chaves dadas
#   perguntas                quantas perguntas foram feitas a alguém

# O que o tests/lib.sh ainda não guarda e o instalador chama.
for _c in lsblk findmnt partprobe udevadm genfstab blkid mountpoint sync lspci \
  systemd-detect-virt timedatectl localectl hwclock gum clear; do
  [[ -e $SBX/guard/$_c ]] && continue
  cat >"$SBX/guard/$_c" <<GUARD
#!/bin/bash
printf '%s\n' "$_c \$*" >>"$SBX/perigo"
echo "PERIGO: o teste chamou '$_c \$*' sem stub" >&2
exit 97
GUARD
  chmod +x "$SBX/guard/$_c"
done
unset _c

mk_install=$REPO/installer/mk-install
export MK_SRC=${MK_SRC:-$REPO}
# O mare-sleep (chamado pelo install_desktop) olha estas raízes, não a máquina.
export SYS=$SBX/live/sys PROC=$SBX/live/proc
mkdir -p "$SBX/live"/{proc,sys/power,sys/bus/pci/devices,etc,usr/share/zoneinfo/America,usr/share/i18n,usr/share/kbd/keymaps/i386/qwerty,run/archiso}
echo "freeze mem disk" >"$SBX/live/sys/power/state"
printf 'processor\t: 0\nvendor_id\t: AuthenticAMD\nmodel name\t: AMD Ryzen 5\n' >"$SBX/live/proc/cpuinfo"
: >"$SBX/live/usr/share/zoneinfo/America/Porto_Velho"
: >"$SBX/live/usr/share/zoneinfo/UTC"
printf '%s\n' 'en_US.UTF-8 UTF-8  ' 'pt_BR.UTF-8 UTF-8  ' 'pt_BR ISO-8859-1  ' >"$SBX/live/usr/share/i18n/SUPPORTED"
: >"$SBX/live/usr/share/kbd/keymaps/i386/qwerty/br-abnt2.map.gz"
: >"$SBX/live/usr/share/kbd/keymaps/i386/qwerty/us.map.gz"
printf '[options]\nArchitecture = auto\n\n[core]\nInclude = /etc/pacman.d/mirrorlist\n' >"$SBX/live/etc/pacman.conf"

set --
# shellcheck source=/dev/null
source "$mk_install"
set +e

# Daqui para baixo, nada do instalador aponta para fora de $SBX.
# shellcheck disable=SC2034 # lidas pelas funções do instalador
{
  target=$SBX/mnt
  log=$SBX/mk-install.log
  live_root=$SBX/live
  aur_repo=$SBX/aur-repo
  use_gum=0
  answers=''
}
mkdir -p "$target" "$aur_repo"

# Bloco é o que o lsblk falso conhece (e o que o LUKS abriria).
is_block() { [[ $1 == /dev/mapper/mkroot ]] || awk -v d="$1" '$1 == d { ok = 1 } END { exit !ok }' "$SBX/discos"; }
is_root() { return 0; }

# As perguntas: num teste ninguém responde. Cada uma fica anotada.
: >"$SBX/perguntas"
_pergunta() {
  printf '%s\n' "$*" >>"$SBX/perguntas"
  echo "PERGUNTA sem resposta: $*" >&2
  exit 96
}
perguntas() { wc -l <"$SBX/perguntas"; }

entrada=/dev/null
out='' rc=0
roda() {
  (
    set -e
    "$@"
  ) >"$SBX/saida" 2>&1 <"$entrada"
  rc=$?
  out=$(<"$SBX/saida")
}

discos() { cat >"$SBX/discos"; }
# Uma máquina com dois discos internos, o pendrive do live e a zram.
discos <<'EOF'
/dev/sda disk 500107862016 465,8G Samsung\x20SSD\x20860 -
/dev/sda1 part 536870912 512M - /dev/sda
/dev/sda2 part 499570991104 465,3G - /dev/sda
/dev/sdb disk 256060514304 238,5G KINGSTON\x20SA400 -
/dev/sdb1 part 256059465728 238,5G - /dev/sdb
/dev/sdc disk 15376318464 14,3G SanDisk\x20Cruzer -
/dev/sdc1 part 1509949440 1,4G - /dev/sdc
/dev/nvme0n1 disk 1000204886016 931,5G WD_BLACK\x20SN770 -
/dev/nvme0n1p1 part 1000203837440 931,5G - /dev/nvme0n1
/dev/zram0 disk 8589934592 8G - -
/dev/sr0 rom 1073741824 1024M QEMU\x20DVD-ROM -
/dev/loop0 loop 900000000 858M - -
EOF

# lsblk falso: responde só as formas que o instalador usa, a partir de $SBX/discos.
stub_lsblk() {
  stub lsblk '
d=$SBX/discos
dev=${*: -1}
case "$*" in
  "-dpnr -o NAME,TYPE,SIZE,MODEL") awk "\$2 != \"part\" { m = \$5; if (m == \"-\") m = \"\"; print \$1, \$2, \$4, m }" "$d" ;;
  "-npdo PKNAME "*) awk -v d="$dev" "\$1 == d && \$6 != \"-\" { print \$6 }" "$d" ;;
  "-dno TYPE "*) awk -v d="$dev" "\$1 == d { print \$2 }" "$d" ;;
  "-dbno SIZE "*) awk -v d="$dev" "\$1 == d { print \$3 }" "$d" ;;
  "-dno SIZE "*) awk -v d="$dev" "\$1 == d { print \$4 }" "$d" ;;
  "-dno MODEL "*) awk -v d="$dev" "\$1 == d { gsub(/\\\\x20/, \" \", \$5); print \$5 }" "$d" ;;
  "-no NAME,SIZE,FSTYPE,LABEL,MOUNTPOINTS "*) awk -v d="$dev" "\$1 == d || \$6 == d { sub(\"/dev/\", \"\", \$1); print \$1, \$4, \"ext4\", \"antigo\" }" "$d" ;;
  "-lnpo NAME "*) awk -v d="$dev" "\$1 == d || \$6 == d { print \$1 }" "$d" ;;
  *) echo "lsblk falso: não sei responder: $*" >&2; exit 64 ;;
esac'
}
stub_lsblk
# O live veio do pendrive /dev/sdc.
stub findmnt '[[ "$*" == "-no SOURCE /run/archiso/bootmnt" ]] && echo /dev/sdc1'

raiz_falsa() {
  local r=${1:-$target} u
  mkdir -p "$r"/{boot,home,etc/default,etc/skel,etc/ufw,etc/cups,etc/snapper/configs,var/log,var/lib} \
    "$r"/usr/{bin,share/limine,share/themes,lib/systemd/system,lib/systemd/user}
  : >"$r/usr/share/limine/BOOTX64.EFI"
  printf '%s\n' '#en_US.UTF-8 UTF-8  ' '#pt_BR.UTF-8 UTF-8  ' '#de_DE.UTF-8 UTF-8  ' >"$r/etc/locale.gen"
  printf '%s\n' 'passwd: files systemd' 'hosts: mymachines resolve [!UNAVAIL=return] files myhostname dns' >"$r/etc/nsswitch.conf"
  printf '# Static information about the filesystems.\n' >"$r/etc/fstab"
  printf 'ENABLED=no\nLOGLEVEL=low\n' >"$r/etc/ufw/ufw.conf"
  printf 'SystemGroup sys root\n' >"$r/etc/cups/cups-files.conf"
  printf '# Sample .bashrc\n' >"$r/etc/skel/.bashrc"
  printf '#!/bin/sh\n' >"$r/usr/bin/ufw"
  chmod +x "$r/usr/bin/ufw"
  for u in NetworkManager.service bluetooth.service sddm.service avahi-daemon.service \
    power-profiles-daemon.service systemd-timesyncd.service ufw.service docker.socket cups.socket \
    fstrim.timer plocate-updatedb.timer snapper-cleanup.timer sshd.service; do
    : >"$r/usr/lib/systemd/system/$u"
  done
  : >"$r/usr/lib/systemd/user/gcr-ssh-agent.socket"
}

fonte_falsa() {
  local e
  mkdir -p "$SBX/src/iso"/{work/aur/repo,out} "$SBX/src/.git" "$SBX/src/aur-repo"
  for e in "$REPO"/*; do
    case ${e##*/} in tests | iso) continue ;; esac
    cp -r "$e" "$SBX/src/"
  done
  cp -r "$REPO/iso/profile" "$REPO/iso/build" "$REPO/iso/vm" "$SBX/src/iso/"
  echo "ref: refs/heads/main" >"$SBX/src/.git/HEAD"
  : >"$SBX/src/aur-repo/yay-bin-1-1-x86_64.pkg.tar.zst"
  : >"$SBX/src/iso/work/aur/repo/lixo"
  : >"$SBX/src/iso/out/mk-2026.01.01-x86_64.iso"
  echo "abc123def456 2026-11-20" >"$SBX/src/VERSION"
  # shellcheck disable=SC2034
  src=$SBX/src PKG_DIR=$SBX/src/system/packages
}

respostas() {
  local kv f=$SBX/respostas.conf
  cp "$REPO/installer/answers.example" "$f"
  sed -i 's|^DISK=.*|DISK=/dev/sdb|' "$f"
  for kv in "$@"; do
    if grep -q "^${kv%%=*}=" "$f"; then
      sed -i "s|^${kv%%=*}=.*|${kv//|/\\|}|" "$f"
    else
      printf '%s\n' "$kv" >>"$f"
    fi
  done
  printf '%s\n' "$f"
}

# As respostas de um arquivo bom, já nas variáveis (sem passar pelo load_answers).
# shellcheck disable=SC2034
respostas_boas() {
  DISK=/dev/sdb HOSTNAME_=mk USERNAME=mark PASSWORD=segredo-de-teste
  TIMEZONE=America/Porto_Velho KEYMAP=br-abnt2 LOCALE=pt_BR.UTF-8
  HARDWARE=vm APPS=nao ENCRYPT=nao CONFIRM=formatar POWEROFF=nao
}

# Troca tudo o que pergunta algo a alguém por uma anotação em $SBX/perguntas.
sem_perguntas() {
  local f
  for f in ask ask_secret confirm choose filter welcome interview ensure_network; do
    eval "$f() { _pergunta $f \"\$@\"; }"
  done
}

# arch-chroot falso: anota a chamada e imita o pouco de que o instalador
# depende. O que cada teste quer diferente, liga por arquivo em $SBX:
#   grupos             os grupos que existem no sistema novo (um por linha)
#   instalados         os pacotes que o pacman -Qq acha
#   limine-update      se existir (executável), é o limine-update do sistema novo
#   snapper-falha      o snapper create-config falha
#   useradd-sem-skel   o useradd não leva o skel para a home
#   falha-<palavra>    o comando cuja primeira palavra é <palavra> falha
stub_chroot() {
  stub arch-chroot '
[[ $1 == -u ]] && shift 2
root=$1
shift
[[ -e $SBX/falha-${1##*/} ]] && exit 1
case "$*" in
  "getent group "*) grep -qx "$3" "$SBX/grupos" 2>/dev/null ;;
  useradd\ *)
    mkdir -p "$root/home"
    [[ -e $SBX/useradd-sem-skel ]] && { mkdir -p "$root/home/${*: -1}"; exit 0; }
    cp -a "$root/etc/skel" "$root/home/${*: -1}" ;;
  chpasswd) cat >"$SBX/chpasswd.entrada" ;;
  "sh -c command -v limine-update") [[ -x $SBX/limine-update ]] ;;
  limine-update) "$SBX/limine-update" "$root" ;;
  "pacman -Qq "*) grep -qx "$3" "$SBX/instalados" 2>/dev/null ;;
  "snapper --no-dbus -c root create-config /")
    [[ -e $SBX/snapper-falha ]] && exit 1
    printf "%s\n" "SUBVOLUME=\"/\"" "TIMELINE_CREATE=\"yes\"" "NUMBER_LIMIT=\"50\"" "NUMBER_LIMIT_IMPORTANT=\"10\"" "TIMELINE_CLEANUP=\"yes\"" >"$root/etc/snapper/configs/root" ;;
esac'
}
stub_chroot
