#!/usr/bin/env bash
# iso/build: o AUR (fila, dependências, reuso do que já foi compilado,
# relatório), o perfil montado e a chamada do mkarchiso. Roda uma cópia do
# script numa árvore falsa dentro de $SBX: o iso/work de verdade não é tocado
# e git, curl, makepkg, pacman, repo-add, sudo e mkarchiso são stubs.
. "$(dirname "$0")/../lib.sh"

[[ $EUID -ne 0 ]] || skip "o iso/build recusa rodar como root"
command -v jq >/dev/null || skip "falta o jq"

# --- a árvore falsa ----------------------------------------------------------
r=$SBX/repo
mkdir -p "$r"/{iso,bin,installer,system/packages,system/mk,config,shell,.git,iso/out}
cp "$REPO/iso/build" "$r/iso/build"
cp -r "$REPO/iso/profile" "$r/iso/profile"
cp "$REPO/bin/lib-packages.sh" "$r/bin/"
cp "$REPO/installer/mk-install" "$r/installer/"
cp "$REPO/system/mk/mk-os-release.hook" "$r/system/mk/"
echo "ref: refs/heads/main" >"$r/.git/HEAD"
: >"$r/iso/out/mk-2020.01.01-x86_64.iso"
work=$r/iso/work
lista() { printf '%s\n' "# Pacotes do AUR de teste" "" "$@" >"$r/system/packages/aur.packages"; }

# Um releng em miniatura, com o que o build mexe.
releng=$SBX/releng
mkdir -p "$releng"/{grub,syslinux,efiboot/loader/entries,airootfs/etc}
printf '%s\n' base linux pciutils zsh >"$releng/packages.x86_64"
echo 'iso_name="archlinux"' >"$releng/profiledef.sh"
echo archiso >"$releng/airootfs/etc/hostname"
printf '%s\n' 'menuentry "Arch Linux install medium (%ARCH%, ${archiso_platform})" --class arch --class gnu-linux --id archlinux {' '}' >"$releng/grub/grub.cfg"
printf '%s\n' 'title    Arch Linux install medium (%ARCH%, UEFI)' 'linux    /%INSTALL_DIR%/boot/%ARCH%/vmlinuz-linux' >"$releng/efiboot/loader/entries/01-archiso-linux.conf"
printf '%s\n' 'title    Arch Linux install medium (%ARCH%, UEFI) with speech' >"$releng/efiboot/loader/entries/02-archiso-speech-linux.conf"
printf '%s\n' 'title    Memtest86+' >"$releng/efiboot/loader/entries/03-archiso-memtest86+.conf"
printf '%s\n' 'MENU TITLE Arch Linux' 'MENU COLOR sel 7;37;40 #e0ffffff #9033ccff all' >"$releng/syslinux/archiso_head.cfg"
printf '%s\n' 'TEXT HELP' 'Boot the Arch Linux install medium on BIOS.' 'It allows you to install Arch Linux or perform system maintenance.' 'ENDTEXT' 'MENU LABEL Arch Linux install medium (%ARCH%, BIOS)' >"$releng/syslinux/archiso_sys-linux.cfg"

# --- o AUR falso -------------------------------------------------------------
# aur <nome> <PackageBase> <versão> <rel> [linhas do .SRCINFO...]
aur() {
  local name=$1 base=$2 ver=$3 rel=$4
  shift 4
  mkdir -p "$SBX/aur/$base" "$SBX/aur-nomes"
  echo "$base" >"$SBX/aur-nomes/$name"
  {
    printf 'pkgbase = %s\n\tpkgver = %s\n\tpkgrel = %s\n' "$base" "$ver" "$rel"
    printf '\t%s\n' "$@"
    printf '\npkgname = %s\n' "$name"
  } >"$SBX/aur/$base/.SRCINFO"
}
stub curl '
n=${*: -1}
n=${n##*=}
if [[ -f $SBX/aur-nomes/$n ]]; then
  printf "{\"resultcount\":1,\"results\":[{\"Name\":\"%s\",\"PackageBase\":\"%s\"}]}\n" "$n" "$(<"$SBX/aur-nomes/$n")"
else
  echo "{\"resultcount\":0,\"results\":[]}"
fi'
stub git '
case "$*" in
  "clone -q --depth 1 "*)
    base=${5##*/}
    base=${base%.git}
    [[ -d $SBX/aur/$base && ! -e $SBX/clone-falha-$base ]] || exit 128
    mkdir -p "$6/.git"
    cp "$SBX/aur/$base/.SRCINFO" "$6/" ;;
  "-C "*" fetch "*) cp "$SBX/aur/${2##*/}/.SRCINFO" "$2/" ;;
  "-C "*" reset "*) ;;
  *"rev-parse --short=12 HEAD") echo abcdef123456 ;;
  *"log -1 --format=%cs") echo 2026-11-20 ;;
  *) echo "git falso: $*" >&2; exit 2 ;;
esac'
# O makepkg deixa em $PKGDEST um pacote por pkgname, na versão do .SRCINFO
# (com a época na frente, como no nome de arquivo de verdade: nome-2:0.9-1-...).
stub makepkg '
base=${PWD##*/}
[[ -e $SBX/makepkg-falha-$base ]] && { echo "==> ERROR: A failure occurred in build()."; exit 4; }
ver=$(sed -n "s/^[[:space:]]*pkgver = //p" .SRCINFO | head -1)-$(sed -n "s/^[[:space:]]*pkgrel = //p" .SRCINFO | head -1)
epoch=$(sed -n "s/^[[:space:]]*epoch = //p" .SRCINFO | head -1)
[[ -n $epoch ]] && ver=$epoch:$ver
mkdir -p "$PKGDEST"
[[ -e $SBX/makepkg-sem-pacote ]] && exit 0
for n in $(sed -n "s/^pkgname = //p" .SRCINFO); do
  : >"$PKGDEST/$n-$ver-x86_64.pkg.tar.zst"
  : >"$PKGDEST/$n-$ver-x86_64.pkg.tar.zst.sig"
done'
# pacman -Sp: está nos repositórios? (tudo, menos o que estiver em fora-dos-repos)
# pacman -T: o que falta nesta máquina (o que estiver em faltam)
stub pacman '
case $1 in
  -Sp) ! grep -qx -- "${*: -1}" "$SBX/fora-dos-repos" 2>/dev/null ;;
  -T)
    shift
    falta=0
    for d in "$@"; do grep -qx -- "$d" "$SBX/faltam" 2>/dev/null && { echo "$d"; falta=1; }; done
    [[ $falta -eq 0 ]] || exit 127 ;;
  *) echo "pacman falso: $*" >&2; exit 2 ;;
esac'
stub repo-add 'for a in "$@"; do [[ $a == *.db.tar.gz ]] && : >"$a"; done; exit 0'
stub sudo

build() {
  calls_reset
  RELENG=$releng bash "$r/iso/build" "$@" >"$SBX/saida" 2>&1
  rc=$?
  out=$(<"$SBX/saida")
}
makepkgs() { grep -c '^makepkg ' "$SBX/calls"; }

# --- opções ------------------------------------------------------------------
t "-h: mostra o uso e não faz nada"
build -h
eq 0 "$rc"
has "$out" "iso/build --prepare"
has "$out" "iso/build --syncdeps"
eq "" "$(<"$SBX/calls")"
absent "$work"

t "opção desconhecida: sai com 2 e não faz nada"
build --tudo
eq 2 "$rc"
has "$out" "opção desconhecida: --tudo"
absent "$work"

t "sem o archiso (releng), para dizendo o que instalar"
calls_reset
out=$(RELENG=$SBX/nao-existe bash "$r/iso/build" --prepare 2>&1)
eq 1 $?
has "$out" "faltam ferramentas: archiso"
has "$out" "sudo pacman -S --needed archiso git jq curl base-devel"
absent "$work"

# --- AUR ---------------------------------------------------------------------
aur foo foo 1.2 3 "depends = glibc" "makedepends = go"
aur bar-bin bar-bin 0.9 1 "epoch = 2" "depends = libbar>=2"
lista foo "bar-bin  # com comentário"

t "--prepare: compila cada pacote do aur.packages com o makepkg, sem sudo"
build --prepare
eq 0 "$rc" "$out"
eq 2 "$(makepkgs)"
not_called "sudo"

t "--prepare: o makepkg não instala dependência nenhuma por padrão"
mk=$(grep '^makepkg ' "$SBX/calls" | head -1)
eq "makepkg --clean --cleanbuild --force --noconfirm --nocheck --noprogressbar" "$mk"

t "--prepare: o PKGBUILD vem do AUR pelo PackageBase"
called "git clone -q --depth 1 https://aur.archlinux.org/foo.git $work/aur/src/foo"
called "git clone -q --depth 1 https://aur.archlinux.org/bar-bin.git $work/aur/src/bar-bin"

t "--prepare: o repositório mk-aur recebe os pacotes (sem as assinaturas) e o packages.txt"
exists "$work/aur/repo/foo-1.2-3-x86_64.pkg.tar.zst"
exists "$work/aur/repo/bar-bin-2:0.9-1-x86_64.pkg.tar.zst"
absent "$work/aur/repo/foo-1.2-3-x86_64.pkg.tar.zst.sig"
eq "foo
bar-bin" "$(<"$work/aur/repo/packages.txt")"
called "repo-add -q $work/aur/repo/mk-aur.db.tar.gz $work/aur/repo/"

t "--prepare: o relatório tem uma linha ok por pacote"
rep=$(<"$work/aur/report.txt")
like "$rep" 'ok +foo +foo-1.2-3-x86_64.pkg.tar.zst'
like "$rep" 'ok +bar-bin +bar-bin-2:0.9-1-x86_64.pkg.tar.zst'
hasnt "$rep" "FALHOU"
exists "$work/aur/foo.log"

t "--prepare: imprime o comando do mkarchiso em vez de rodar"
has "$out" "sudo mkarchiso -v -r -w $work/mkarchiso -o $r/iso/out $work/profile"
not_called "mkarchiso"

t "segunda rodada: pacote já compilado na mesma versão é reaproveitado, sem makepkg"
build --prepare
eq 0 "$rc" "$out"
eq 0 "$(makepkgs)"
like "$(<"$work/aur/report.txt")" 'ok +foo +foo-1.2-3-x86_64.pkg.tar.zst \(já compilado\)'
like "$(<"$work/aur/report.txt")" 'ok +bar-bin +bar-bin-2:0.9-1-x86_64.pkg.tar.zst \(já compilado\)'
exists "$work/aur/repo/foo-1.2-3-x86_64.pkg.tar.zst"
eq "foo
bar-bin" "$(<"$work/aur/repo/packages.txt")"

t "segunda rodada: o PKGBUILD já clonado é atualizado, não clonado de novo"
not_called "git clone"
called "git -C $work/aur/src/foo fetch -q --depth 1 origin"

t "versão nova no AUR: só esse pacote é recompilado, e o repositório leva a nova"
aur foo foo 1.3 1 "depends = glibc" "makedepends = go"
build --prepare
eq 1 "$(makepkgs)"
exists "$work/aur/repo/foo-1.3-1-x86_64.pkg.tar.zst"
absent "$work/aur/repo/foo-1.2-3-x86_64.pkg.tar.zst"
like "$(<"$work/aur/report.txt")" 'ok +bar-bin +bar-bin-2:0.9-1-x86_64.pkg.tar.zst \(já compilado\)'

t "pacote -git é sempre recompilado, mesmo com a versão igual"
aur blesh-git blesh-git 0.4.0 1
lista blesh-git
build --prepare
eq 1 "$(makepkgs)"
build --prepare
eq 1 "$(makepkgs)"
hasnt "$(<"$work/aur/report.txt")" "já compilado"

t "o nome do pacote vira o PackageBase do AUR (pacote dividido)"
aur libfoo foo-suite 2.0 1
printf 'pkgname = foo-suite-docs\n' >>"$SBX/aur/foo-suite/.SRCINFO"
lista libfoo
build --prepare
eq 0 "$rc" "$out"
called "git clone -q --depth 1 https://aur.archlinux.org/foo-suite.git $work/aur/src/foo-suite"
exists "$work/aur/repo/libfoo-2.0-1-x86_64.pkg.tar.zst"
absent "$work/aur/repo/foo-suite-docs-2.0-1-x86_64.pkg.tar.zst"
eq "libfoo" "$(<"$work/aur/repo/packages.txt")"

t "dependência que só existe no AUR entra na fila sozinha; a dos repositórios, não"
aur app app 1.0 1 "depends = glibc" "depends = libso-aur>=1.0" "depends = app-data"
aur libso-aur libso-aur 3.1 2
aur app-data app-data 1.0 1
printf '%s\n' libso-aur app-data >"$SBX/fora-dos-repos"
lista app
build --prepare
eq 0 "$rc" "$out"
eq "app
app-data
libso-aur" "$(<"$work/aur/repo/packages.txt")"
fails grep -q 'glibc' "$work/aur/report.txt"

t "dependência do AUR pedida por dois pacotes é compilada uma vez só"
aur app2 app2 1.0 1 "depends = libso-aur"
lista app app2 libso-aur
rm -rf "$work/aur/pkg"
build --prepare
eq 4 "$(makepkgs)"
eq 1 "$(grep -c '^libso-aur$' "$work/aur/repo/packages.txt")"
eq "app app2 libso-aur app-data" "$(tr '\n' ' ' <"$work/aur/repo/packages.txt" | sed 's/ $//')"
: >"$SBX/fora-dos-repos"

t "pacote que não existe no AUR: fica de fora, vai para o relatório e o build continua"
lista naoexiste foo
build --prepare
eq 0 "$rc" "$out"
like "$(<"$work/aur/report.txt")" 'FALHOU +naoexiste: não encontrado no AUR'
eq "foo" "$(<"$work/aur/repo/packages.txt")"
has "$out" "1 compilados, 1 com falha"

t "git clone que falha: fica de fora e o build continua"
aur novo novo 1.0 1
: >"$SBX/clone-falha-novo"
lista novo foo
build --prepare
eq 0 "$rc" "$out"
like "$(<"$work/aur/report.txt")" 'FALHOU +novo: git clone de https://aur.archlinux.org/novo.git falhou'
eq "foo" "$(<"$work/aur/repo/packages.txt")"
rm -f "$SBX/clone-falha-novo"

t "makepkg que falha: fica de fora, o relatório aponta o log e o build continua"
aur quebra quebra 1.0 1
: >"$SBX/makepkg-falha-quebra"
lista quebra foo
build --prepare
eq 0 "$rc" "$out"
has "$(<"$work/aur/report.txt")" "FALHOU  quebra: makepkg falhou, veja $work/aur/quebra.log"
has "$(<"$work/aur/quebra.log")" "A failure occurred in build()"
eq "foo" "$(<"$work/aur/repo/packages.txt")"
absent "$work/aur/repo/quebra-1.0-1-x86_64.pkg.tar.zst"
rm -f "$SBX/makepkg-falha-quebra"

t "makepkg que termina sem deixar o pacote: falha no relatório"
aur vazio vazio 1.0 1
: >"$SBX/makepkg-sem-pacote"
lista vazio
build --prepare
eq 0 "$rc" "$out"
has "$(<"$work/aur/report.txt")" "FALHOU  vazio: makepkg terminou, mas o pacote vazio não apareceu"
rm -f "$SBX/makepkg-sem-pacote"

t "nenhum pacote compilado: repositório vazio e packages.txt vazio (nenhuma linha em branco)"
exists "$work/aur/repo/mk-aur.db.tar.gz"
eq "repo-add -q $work/aur/repo/mk-aur.db.tar.gz" "$(grep '^repo-add' "$SBX/calls")"
exists "$work/aur/repo/packages.txt"
eq 0 "$(wc -c <"$work/aur/repo/packages.txt")"
has "$out" "0 compilados, 1 com falha"

t "falta dependência de compilação: por padrão o pacote falha e o relatório diz o que instalar"
aur gancho gancho 1.0 1 "makedepends = gradle" "makedepends = git" "depends = limine"
printf '%s\n' gradle limine >"$SBX/faltam"
lista gancho foo
rm -rf "$work/aur/pkg"
build --prepare
eq 0 "$rc" "$out"
rep=$(<"$work/aur/report.txt")
has "$rep" "FALHOU  gancho: faltam para compilar: gradle "
has "$rep" "iso/build --syncdeps"
has "$rep" "sudo pacman -S --asdeps gradle"
hasnt "$(grep gancho <<<"$rep")" "limine)"
eq "foo" "$(<"$work/aur/repo/packages.txt")"
not_called "sudo"
hasnt "$(<"$SBX/calls")" "--syncdeps"

t "--syncdeps: o makepkg instala e remove o que falta"
build --prepare --syncdeps
eq 0 "$rc" "$out"
ok grep -qx 'makepkg --clean --cleanbuild --force --noconfirm --nocheck --noprogressbar --syncdeps --rmdeps' "$SBX/calls"
ok grep -qx gancho "$work/aur/repo/packages.txt"

t "--syncdeps: quem não tem dependência faltando é compilado sem --syncdeps"
eq 1 "$(grep -c -- '--syncdeps' "$SBX/calls")"

t "só falta dependência de execução: compila com --nodeps (ela chega no pacstrap)"
echo limine >"$SBX/faltam"
rm -rf "$work/aur/pkg"
build --prepare
eq 0 "$rc" "$out"
ok grep -qx 'makepkg --clean --cleanbuild --force --noconfirm --nocheck --noprogressbar --nodeps' "$SBX/calls"
ok grep -qx gancho "$work/aur/repo/packages.txt"
: >"$SBX/faltam"

# --- o perfil montado --------------------------------------------------------
lista foo
build --prepare
p=$work/profile
share=$p/airootfs/usr/share/mk

t "perfil: é o releng com o iso/profile por cima"
eq 0 "$rc" "$out"
has "$(<"$p/profiledef.sh")" 'iso_name="mk"'
eq mk-live "$(<"$p/airootfs/etc/hostname")"
eq "KEYMAP=br-abnt2" "$(<"$p/airootfs/etc/vconsole.conf")"
exists "$p/airootfs/etc/motd"
exists "$p/syslinux/splash.png"

t "perfil: packages.x86_64 = releng + packages.mk, ordenado e sem repetir"
eq "base
gum
linux
pciutils
zsh" "$(<"$p/packages.x86_64")"
absent "$p/packages.mk"

t "perfil: no menu UEFI saem as entradas do Arch (e a de voz) e entra a 01-mk"
absent "$p/efiboot/loader/entries/01-archiso-linux.conf"
absent "$p/efiboot/loader/entries/02-archiso-speech-linux.conf"
exists "$p/efiboot/loader/entries/01-mk.conf"
exists "$p/efiboot/loader/entries/03-archiso-memtest86+.conf"

t "perfil: os menus de boot dizem Mk, não Arch Linux install medium"
fails grep -rq 'Arch Linux install medium' "$p/grub" "$p/syslinux" "$p/efiboot"
has "$(<"$p/grub/grub.cfg")" 'menuentry "Mk (%ARCH%, ${archiso_platform})" --class mk --class gnu-linux'
has "$(<"$p/syslinux/archiso_head.cfg")" "MENU TITLE Mk"
has "$(<"$p/syslinux/archiso_head.cfg")" "#ffce5e32"
has "$(<"$p/syslinux/archiso_sys-linux.cfg")" "Boot the Mk install medium on BIOS."
has "$(<"$p/syslinux/archiso_sys-linux.cfg")" "It allows you to install Mk or perform"

t "perfil: o releng de origem não é alterado"
has "$(<"$releng/grub/grub.cfg")" "Arch Linux install medium"
exists "$releng/efiboot/loader/entries/01-archiso-linux.conf"
absent "$releng/airootfs/usr"

t "perfil: o repositório do Mk vai para /usr/share/mk, sem .git e sem as saídas do build"
exists "$share/installer/mk-install"
exists "$share/system/packages/aur.packages"
exists "$share/bin/lib-packages.sh"
exists "$share/iso/build"
absent "$share/.git"
absent "$share/iso/work"
absent "$share/iso/out"

t "perfil: o AUR compilado vai junto, em /usr/share/mk/aur-repo"
exists "$share/aur-repo/mk-aur.db.tar.gz"
eq "foo" "$(<"$share/aur-repo/packages.txt")"
exists "$share/aur-repo/foo-1.3-1-x86_64.pkg.tar.zst"

t "perfil: VERSION é 'commit data'"
eq "abcdef123456 2026-11-20" "$(<"$share/VERSION")"

t "perfil: o instalador em /usr/local/bin/mk-install (executável) e o gancho do os-release"
eq 755 "$(stat -c %a "$p/airootfs/usr/local/bin/mk-install")"
ok cmp -s "$REPO/installer/mk-install" "$p/airootfs/usr/local/bin/mk-install"
ok cmp -s "$REPO/system/mk/mk-os-release.hook" "$p/airootfs/etc/pacman.d/hooks/mk-os-release.hook"

t "perfil: é refeito a cada build (nada de uma rodada anterior sobra)"
: >"$p/sobra-de-antes"
build --prepare
absent "$p/sobra-de-antes"

t "perfil: avisa se ainda houver 'meu-os' nas configs"
hasnt "$out" "AVISO"
echo "source ~/meu-os/x" >"$r/config/velho.conf"
build --prepare
has "$out" "AVISO: ainda há 'meu-os'"
rm -f "$r/config/velho.conf"

# --- o ISO -------------------------------------------------------------------
t "sem --prepare: o único sudo é o mkarchiso, com a área de trabalho e a saída em iso/"
build
eq 0 "$rc" "$out"
eq "sudo mkarchiso -v -r -w $work/mkarchiso -o $r/iso/out $work/profile" "$(grep '^sudo' "$SBX/calls")"
eq "sudo" "$(tail -1 "$SBX/calls" | cut -d' ' -f1)"

t "sem --prepare: no fim lembra os pacotes do AUR que ficaram de fora"
lista naoexiste foo
build
eq 0 "$rc" "$out"
has "$out" "Pacotes do AUR que ficaram de fora: naoexiste"

t "se o mkarchiso falha, o build falha"
stub sudo 'exit 1'
build
fails test "$rc" -eq 0
stub sudo

t "nada sai da árvore do build: nenhum caminho de fora de \$SBX nos comandos que escrevem"
eq "" "$(grep -E '^(makepkg|repo-add|sudo|git clone)' "$SBX/calls" | grep -o ' /[^ ]*' | grep -v " $SBX/" || true)"
