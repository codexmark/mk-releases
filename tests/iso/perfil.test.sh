#!/usr/bin/env bash
# iso/profile: o perfil do ISO, conferido parado (sem montar nem gerar nada).
# Lista de pacotes, profiledef.sh, a entrada de boot UEFI, o que vai no live e
# os arquivos que o iso/build e o README dizem existir.
. "$(dirname "$0")/../lib.sh"

prof=$REPO/iso/profile
# O releng desta máquina, se o archiso estiver instalado (só leitura).
releng=/usr/share/archiso/configs/releng
[[ -f $releng/profiledef.sh ]] || releng=""

# --- packages.mk -------------------------------------------------------------
pk=$prof/packages.mk
lista=$(sed -e 's/#.*//' -e 's/[[:space:]]*$//' "$pk" | grep -v '^$')

t "packages.mk: um pacote por linha; o resto é comentário ou linha vazia"
while IFS= read -r l; do
  like "$l" '^[a-z0-9][a-z0-9@._+-]*$' "linha: $l"
done <<<"$lista"
ok test "$(wc -l <<<"$lista")" -ge 1

t "packages.mk: sem pacote repetido"
eq "" "$(sort <<<"$lista" | uniq -d)"

t "packages.mk: traz o que o instalador usa no live e o releng não tem (gum, lspci)"
ok grep -qx gum <<<"$lista"
ok grep -qx pciutils <<<"$lista"

t "packages.mk: nenhum pacote do AUR (o live só instala dos repositórios)"
while IFS= read -r a; do
  fails grep -qx -- "$a" <<<"$lista"
done < <(sed -e 's/#.*//' -e 's/[[:space:]]*$//' "$REPO/system/packages/aur.packages" | grep -v '^$')

t "packages.mk: o perfil não traz um packages.x86_64 próprio (o build soma ao do releng)"
absent "$prof/packages.x86_64"

# --- profiledef.sh -----------------------------------------------------------
def() { # def <expressão bash>: avalia com o profiledef.sh carregado
  (
    declare -A file_permissions=()
    export SOURCE_DATE_EPOCH=1795132800 # 2026-11-20
    # shellcheck source=/dev/null
    source "$prof/profiledef.sh"
    eval "$1"
  )
}

t "profiledef.sh: sintaxe válida e carrega sem erro"
ok bash -n "$prof/profiledef.sh"
ok def true

t "profiledef.sh: nome mk, rótulo MK, pasta mk"
eq mk "$(def 'echo $iso_name')"
eq MK "$(def 'echo $iso_label')"
eq mk "$(def 'echo $install_dir')"
eq Mk "$(def 'echo $iso_publisher')"

t "profiledef.sh: o rótulo cabe num volume ISO 9660 (até 32, maiúsculas, números e _)"
like "$(def 'echo $iso_label')" '^[A-Z0-9_]{1,32}$'

t "profiledef.sh: a versão é a data AAAA.MM.DD, como no nome do ISO do README"
eq 2026.11.20 "$(def 'echo $iso_version')"

t "profiledef.sh: gera só o ISO, que inicia em UEFI (systemd-boot) e em BIOS (syslinux)"
eq "iso" "$(def 'echo "${buildmodes[*]}"')"
eq "bios.syslinux uefi.systemd-boot" "$(def 'echo "${bootmodes[*]}"')"

t "profiledef.sh: o instalador entra no live como executável do root"
eq "0:0:755" "$(def 'echo "${file_permissions[/usr/local/bin/mk-install]}"')"
eq "0:0:400" "$(def 'echo "${file_permissions[/etc/shadow]}"')"

t "profiledef.sh: toda permissão tem a forma dono:grupo:modo"
while IFS= read -r v; do like "$v" '^[0-9]+:[0-9]+:[0-7]{3,4}$'; done < <(def 'printf "%s\n" "${file_permissions[@]}"')

t "profiledef.sh: cada modo de boot tem os arquivos dele no perfil ou no releng"
exists "$prof/efiboot/loader/entries/01-mk.conf"
exists "$prof/syslinux/splash.png"

t "profiledef.sh: todo caminho com permissão existe no live (releng, perfil ou posto pelo iso/build)"
if [[ -n $releng ]]; then
  while IFS= read -r c; do
    [[ $c == /usr/local/bin/mk-install ]] && continue # o iso/build instala
    [[ -e $releng/airootfs$c || -e $prof/airootfs$c ]] || _fail "não existe no live: $c"
  done < <(def 'printf "%s\n" "${!file_permissions[@]}"')
  ok grep -q 'airootfs/usr/local/bin/mk-install' "$REPO/iso/build"
  exists "$releng/$(def 'echo $pacman_conf')"
else
  echo "  # sem o archiso nesta máquina: conferido só contra o perfil" >&2
  ok grep -q 'airootfs/usr/local/bin/mk-install' "$REPO/iso/build"
fi

t "profiledef.sh: mantém as permissões do releng (o profiledef do Mk substitui o dele)"
if [[ -n $releng ]]; then
  while IFS= read -r c; do
    eq "$(
      declare -A file_permissions=()
      # shellcheck source=/dev/null
      source "$releng/profiledef.sh" 2>/dev/null
      echo "${file_permissions[$c]}"
    )" "$(def "echo \"\${file_permissions[$c]}\"")" "$c"
  done < <(
    declare -A file_permissions=()
    # shellcheck source=/dev/null
    source "$releng/profiledef.sh" 2>/dev/null
    printf '%s\n' "${!file_permissions[@]}"
  )
fi

# --- a entrada de boot UEFI --------------------------------------------------
entrada=$(<"$prof/efiboot/loader/entries/01-mk.conf")

t "01-mk.conf: chama-se Mk e inicia o kernel e o initramfs do live"
like "$entrada" 'title +Mk '
hasnt "$entrada" "Arch Linux"
like "$entrada" 'linux +/%INSTALL_DIR%/boot/%ARCH%/vmlinuz-linux'
like "$entrada" 'initrd +/%INSTALL_DIR%/boot/%ARCH%/initramfs-linux.img'

t "01-mk.conf: acha o live pela pasta e pelo UUID do ISO (os marcadores do archiso)"
has "$entrada" "archisobasedir=%INSTALL_DIR%"
has "$entrada" "archisosearchuuid=%ARCHISO_UUID%"

t "01-mk.conf: só usa marcadores que o mkarchiso troca"
eq "" "$(grep -o '%[A-Z_]*%' <<<"$entrada" | sort -u | grep -vx -e '%ARCH%' -e '%INSTALL_DIR%' -e '%ARCHISO_UUID%' -e '%ARCHISO_LABEL%' -e '%KERNEL_PARAMS%' || true)"

t "01-mk.conf: fica no lugar da 01-archiso-linux do releng, que o iso/build tira"
has "$(<"$REPO/iso/build")" 'efiboot/loader/entries/01-archiso-linux.conf'
has "$(<"$REPO/iso/build")" 'efiboot/loader/entries/02-archiso-speech-linux.conf'
if [[ -n $releng ]]; then
  exists "$releng/efiboot/loader/entries/01-archiso-linux.conf"
  ok diff <(sed -n 's/^\(linux\|initrd\) *//p' "$releng/efiboot/loader/entries/01-archiso-linux.conf") \
    <(sed -n 's/^\(linux\|initrd\) *//p' "$prof/efiboot/loader/entries/01-mk.conf")
fi

# --- o live ------------------------------------------------------------------
t "live: hostname mk-live e teclado br-abnt2, como no README"
eq mk-live "$(<"$prof/airootfs/etc/hostname")"
eq "KEYMAP=br-abnt2" "$(<"$prof/airootfs/etc/vconsole.conf")"

t "live: o motd manda rodar o mk-install e diz como trocar o teclado"
motd=$(<"$prof/airootfs/etc/motd")
has "$motd" "mk-install"
has "$motd" "loadkeys"
has "$motd" "br-abnt2"

t "live: o fundo do menu de boot em BIOS é um PNG de 640x480"
eq "89504e470d0a1a0a" "$(head -c 8 "$prof/syslinux/splash.png" | od -An -tx1 | tr -d ' \n')"
eq "00000280000001e0" "$(head -c 24 "$prof/syslinux/splash.png" | tail -c 8 | od -An -tx1 | tr -d ' \n')"

# --- o que o build e o README dizem existir ----------------------------------
t "todo arquivo do repositório que o iso/build usa existe"
while IFS= read -r c; do
  c=${c//\$repo/$REPO}
  c=${c//\$iso/$REPO/iso}
  exists "${c%/.}"
done < <(grep -o '\$\(repo\|iso\)/[A-Za-z0-9_./-]*' "$REPO/iso/build" | grep -v -e '^\$iso/work' -e '^\$iso/out' | sort -u)

t "todo arquivo do perfil está descrito no iso/README.md"
readme=$(<"$REPO/iso/README.md")
while IFS= read -r f; do
  case $f in
    airootfs/etc/hostname) has "$readme" "mk-live" ;;
    airootfs/etc/vconsole.conf) has "$readme" "br-abnt2" ;;
    airootfs/etc/motd) has "$readme" "motd" ;;
    *) has "$readme" "${f##*/}" ;;
  esac
done < <(cd "$prof" && find . -type f | sed 's|^\./||')

t "os menus que o iso/build renomeia existem no releng (um sed sem arquivo pararia o build)"
if [[ -n $releng ]]; then
  ok compgen -G "$releng/grub/*.cfg"
  ok compgen -G "$releng/efiboot/loader/entries/*.conf"
  ok compgen -G "$releng/syslinux/*.cfg"
  ok grep -rq 'Arch Linux install medium' "$releng/grub" "$releng/syslinux"
  exists "$releng/packages.x86_64"
fi >/dev/null

t "as saídas do build ficam fora do git"
ok grep -qx '/work/' "$REPO/iso/.gitignore"
ok grep -qx '/out/' "$REPO/iso/.gitignore"

t "iso/build e iso/vm são executáveis"
ok test -x "$REPO/iso/build"
ok test -x "$REPO/iso/vm"
