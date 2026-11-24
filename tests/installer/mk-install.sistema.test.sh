#!/usr/bin/env bash
# installer/mk-install: o que vai para dentro do sistema novo ($target):
# pacotes, fstab e região, o desktop Mk, o usuário, snapshots e serviços.
# shellcheck disable=SC2154,SC2034 # as variáveis vêm do instalador (carrega.sh)
. "$(dirname "$0")/../lib.sh"
. "$(dirname "$0")/carrega.sh"

fonte_falsa
raiz_falsa
com() { # com <função> [CHAVE=valor...]: as respostas boas, com as trocas pedidas
  local f=$1 kv
  shift
  respostas_boas
  for kv in "$@"; do printf -v "${kv%%=*}" '%s' "${kv#*=}"; done
  "$f"
}
modo() { stat -c %a "$1"; }

# --- make_pacman_conf --------------------------------------------------------
t "make_pacman_conf: sem o repositório do AUR, é o pacman.conf do live como está"
roda eval 'make_pacman_conf; cat "$pacman_conf"; echo "ARQ=$pacman_conf"'
eq 0 "$rc" "$out"
has "$out" "[core]"
hasnt "$out" "[mk-aur]"

t "make_pacman_conf: o arquivo temporário fica na pasta temporária, não solto em /tmp"
has "$out" "ARQ=$TMPDIR/mk-pacman."

t "make_pacman_conf: com o repositório do AUR, soma o mk-aur apontando para ele"
: >"$aur_repo/mk-aur.db.tar.gz"
roda eval 'make_pacman_conf; cat "$pacman_conf"'
has "$out" "[mk-aur]"
has "$out" "Server = file://$aur_repo"
has "$out" "[core]"

t "make_pacman_conf: o pacman.conf do live não é alterado"
hasnt "$(<"$SBX/live/etc/pacman.conf")" "mk-aur"

# --- install_packages --------------------------------------------------------
stub pacstrap
stub timedatectl
stub systemctl
# shellcheck disable=SC2034
pacman_conf=$SBX/pacman.conf
pacotes() { grep '^pacstrap ' "$SBX/calls" | tail -1 | cut -d' ' -f6- | tr ' ' '\n'; }

t "install_packages: um pacstrap só, no alvo, com o pacman.conf montado"
calls_reset
roda com install_packages
eq 0 "$rc" "$out"
eq 1 "$(grep -c '^pacstrap ' "$SBX/calls")"
called "pacstrap -K -C $SBX/pacman.conf $target "

t "o pacman da instalação roda sem os ganchos do snap-pac (SNAP_PAC_SKIP)"
# No pacstrap ainda não há configuração do snapper no disco novo: o gancho do
# snap-pac escrevia "fatal library error, lookup self" no meio da instalação.
stub pacstrap 'echo "snap=${SNAP_PAC_SKIP:-}" >>"$SBX/env-pacstrap"'
rm -f "$SBX/env-pacstrap"
roda com install_packages
eq "snap=y" "$(sort -u "$SBX/env-pacstrap")"
stub pacstrap

t "install_packages: base, desktop e dev sempre entram"
p=$(pacotes)
for x in base linux limine snapper hyprland quickshell sddm ghostty neovim; do
  ok grep -qx "$x" <<<"$p"
done

t "install_packages: sem apps, nada de apps.packages"
for x in nautilus chromium libreoffice-fresh libreoffice-fresh-pt-br; do
  fails grep -qx "$x" <<<"$p"
done

t "install_packages: só os drivers do perfil escolhido (vm)"
ok grep -qx vulkan-swrast <<<"$p"
for x in amd-ucode intel-ucode nvidia-open-dkms; do fails grep -qx "$x" <<<"$p"; done

t "install_packages: nenhum pacote do AUR vai para o pacstrap dos repositórios"
while read -r a; do fails grep -qx "$a" <<<"$p"; done < <(read_list "$PKG_DIR/aur.packages")

t "install_packages: nenhum comentário, seção ou linha vazia vira pacote"
fails grep -q '[][#[:space:]]' <<<"$p"
fails grep -qx '' <<<"$p"

t "install_packages: com apps e pt_BR, entram os apps e o LibreOffice em português"
roda com install_packages APPS=sim
p=$(pacotes)
for x in nautilus chromium libreoffice-fresh libreoffice-fresh-pt-br; do ok grep -qx "$x" <<<"$p"; done

t "install_packages: com apps em outro idioma, o LibreOffice em português não entra"
roda com install_packages APPS=sim LOCALE=en_US.UTF-8
p=$(pacotes)
ok grep -qx libreoffice-fresh <<<"$p"
fails grep -qx libreoffice-fresh-pt-br <<<"$p"

t "install_packages: vários perfis de hardware somam; sem perfil, nenhum driver"
roda com install_packages "HARDWARE=amd nvidia"
p=$(pacotes)
for x in amd-ucode vulkan-radeon nvidia-open-dkms nvidia-utils; do ok grep -qx "$x" <<<"$p"; done
for x in intel-ucode vulkan-swrast; do fails grep -qx "$x" <<<"$p"; done
roda com install_packages HARDWARE=
p=$(pacotes)
for x in amd-ucode intel-ucode nvidia-utils vulkan-swrast; do fails grep -qx "$x" <<<"$p"; done

t "install_packages: se o pacstrap falha, a instalação para"
stub pacstrap 'exit 1'
roda com install_packages
fails test "$rc" -eq 0
stub pacstrap

# --- configure_base ----------------------------------------------------------
stub genfstab 'cat <<FSTAB
# /dev/sdb2 LABEL=mk
UUID=2222-btrfs	/         	btrfs     	rw,noatime,compress=zstd:3,ssd,space_cache=v2,subvolid=256,subvol=/@	0 0

# /dev/sdb2 LABEL=mk
UUID=2222-btrfs	/home     	btrfs     	rw,noatime,compress=zstd:3,ssd,space_cache=v2,subvolid=257,subvol=/@home	0 0

# /dev/sdb1 LABEL=MK_EFI
UUID=AAAA-BBBB      	/boot     	vfat      	rw,relatime,fmask=0077,dmask=0077	0 2
FSTAB'

t "configure_base: fstab por UUID e sem subvolid (um rollback troca o id)"
calls_reset
roda com configure_base
eq 0 "$rc" "$out"
called "genfstab -U $target"
fstab=$(<"$target/etc/fstab")
hasnt "$fstab" "subvolid"
has "$fstab" "space_cache=v2,subvol=/@	0 0"
has "$fstab" "subvol=/@home"
has "$fstab" "UUID=AAAA-BBBB"
hasnt "$fstab" "/dev/sdb2	"

t "configure_base: fuso, idioma e teclado do console"
eq "/usr/share/zoneinfo/America/Porto_Velho" "$(readlink "$target/etc/localtime")"
eq "LANG=pt_BR.UTF-8" "$(<"$target/etc/locale.conf")"
eq "KEYMAP=br-abnt2" "$(<"$target/etc/vconsole.conf")"

t "configure_base: locale.gen com o idioma escolhido e en_US; os outros continuam comentados"
gen=$(<"$target/etc/locale.gen")
ok grep -q '^pt_BR.UTF-8 UTF-8' <<<"$gen"
ok grep -q '^en_US.UTF-8 UTF-8' <<<"$gen"
ok grep -q '^#de_DE.UTF-8 UTF-8' <<<"$gen"
called "arch-chroot $target locale-gen"

t "configure_base: teclado da tela de login: br-abnt2 vira br"
has "$(<"$target/etc/X11/xorg.conf.d/00-keyboard.conf")" 'Option "XkbLayout" "br"'

t "configure_base: hostname e /etc/hosts"
eq "mk" "$(<"$target/etc/hostname")"
has "$(<"$target/etc/hosts")" "127.0.1.1   mk.localdomain mk"
has "$(<"$target/etc/hosts")" "127.0.0.1   localhost"

t "configure_base: nomes .local pela rede (mdns_minimal antes do resolve), uma vez só"
eq "hosts: mymachines mdns_minimal [NOTFOUND=return] resolve [!UNAVAIL=return] files myhostname dns" \
  "$(grep '^hosts:' "$target/etc/nsswitch.conf")"
roda com configure_base
eq 1 "$(grep -o mdns_minimal "$target/etc/nsswitch.conf" | wc -l)"

t "configure_base: outro idioma e outro teclado"
roda com configure_base LOCALE=de_DE.UTF-8 KEYMAP=de-latin1 HOSTNAME_=Estacao-2
eq "LANG=de_DE.UTF-8" "$(<"$target/etc/locale.conf")"
ok grep -q '^de_DE.UTF-8 UTF-8' "$target/etc/locale.gen"
has "$(<"$target/etc/X11/xorg.conf.d/00-keyboard.conf")" '"XkbLayout" "de"'
eq "Estacao-2" "$(<"$target/etc/hostname")"

# --- place -------------------------------------------------------------------
t "place: copia a pasta e deixa a marca .instalado-pelo-mk"
mkdir -p "$SBX/origem/sub"
echo a >"$SBX/origem/a.conf"
echo b >"$SBX/origem/sub/b.conf"
place "$SBX/origem" "$SBX/destino/x/pasta"
eq a "$(<"$SBX/destino/x/pasta/a.conf")"
eq b "$(<"$SBX/destino/x/pasta/sub/b.conf")"
exists "$SBX/destino/x/pasta/.instalado-pelo-mk"
absent "$SBX/origem/.instalado-pelo-mk"

t "place: troca o que havia no destino, sem misturar com o antigo"
echo velho >"$SBX/destino/x/pasta/velho.conf"
place "$SBX/origem" "$SBX/destino/x/pasta"
absent "$SBX/destino/x/pasta/velho.conf"
absent "$SBX/destino/x/pasta/origem"

# --- install_desktop ---------------------------------------------------------
share=$target/usr/share/mk
skel=$target/etc/skel
# Como no Arch: /etc/os-release é um link para o do pacote filesystem.
printf 'NAME="Arch Linux"\nID=arch\n' >"$target/usr/lib/os-release"
ln -s ../usr/lib/os-release "$target/etc/os-release"
printf 'Arch Linux \\r (\\l)\n' >"$target/etc/issue"
mkdir -p "$target/etc/pacman.d"

t "install_desktop: roda inteiro sem erro e sem perguntar"
calls_reset
: >"$SBX/marca"
roda com install_desktop
eq 0 "$rc" "$out"
eq 0 "$(perguntas)"
hasnt "$out" "AVISO"

t "install_desktop: o repositório vai para /usr/share/mk, com o VERSION do ISO"
for f in bin/mk-session bin/lib-packages.sh system/packages/base.packages config/hypr shell/mare \
  installer/mk-install scripts/user/neovim.sh scripts/system/firewall.sh system/mk/boot/limine-default \
  system/mk/os-release system/mk/issue system/mk/channel apps/mktunes/bin/mktunes iso/build iso/profile/profiledef.sh; do
  exists "$share/$f"
done
eq "abc123def456 2026-11-20" "$(<"$share/VERSION")"

t "install_desktop: sem .git, aur-repo, iso/work e iso/out"
absent "$share/.git"
absent "$share/aur-repo"
absent "$share/iso/work"
absent "$share/iso/out"

t "install_desktop: o nome do sistema é Mk, e o os-release do Arch não é escrito por cima"
has "$(<"$target/etc/os-release")" 'NAME="Mk"'
fails test -L "$target/etc/os-release"
has "$(<"$target/usr/lib/os-release")" "Arch Linux"
eq "$(<"$REPO/system/mk/issue")" "$(<"$target/etc/issue")"
eq 644 "$(modo "$target/etc/os-release")"

t "install_desktop: o gancho do pacman que repõe o nome a cada atualização do filesystem"
eq "$(<"$REPO/system/mk/mk-os-release.hook")" "$(<"$target/etc/pacman.d/hooks/mk-os-release.hook")"

t "install_desktop: a sessão: mk-session, mk-update e mk.desktop"
eq 755 "$(modo "$target/usr/bin/mk-session")"
eq 755 "$(modo "$target/usr/bin/mk-update")"
desk=$(<"$target/usr/share/wayland-sessions/mk.desktop")
has "$desk" "Name=Mk"
has "$desk" "Exec=/usr/bin/mk-session"
has "$desk" "Type=Application"

t "install_desktop: o MkTunes fica ligado por links para /usr/share/mk"
eq /usr/share/mk/apps/mktunes/bin/mktunes "$(readlink "$target/usr/bin/mktunes")"
eq /usr/share/mk/apps/mktunes/share/mktunes.desktop "$(readlink "$target/usr/share/applications/mktunes.desktop")"

t "install_desktop: o skel recebe as configs do Mk, cada uma de onde o README diz"
ok diff -r -x .instalado-pelo-mk "$REPO/config/hypr" "$skel/.config/mk/hypr"
ok diff -r -x .instalado-pelo-mk "$REPO/config/ghostty" "$skel/.config/mk/ghostty"
ok diff -r -x .instalado-pelo-mk "$REPO/shell/mare" "$skel/.config/mk/shell"
ok diff -r -x .instalado-pelo-mk "$REPO/config/fastfetch" "$skel/.config/mk/fastfetch"
ok diff -r -x .instalado-pelo-mk "$REPO/config/bash" "$skel/.config/mk/bash"
ok diff "$REPO/config/bash/seu.bash.exemplo" "$skel/.config/mk/seu.bash"
ok diff "$REPO/config/mimeapps.list" "$skel/.config/mimeapps.list"
ok diff -r "$REPO/system/mk/look/qt6ct" "$skel/.config/qt6ct"
exists "$skel/Work"

t "install_desktop: as cinco pastas de .config/mk levam a marca; o resto não"
for d in hypr shell ghostty fastfetch bash; do exists "$skel/.config/mk/$d/.instalado-pelo-mk"; done
eq 5 "$(find "$skel" -name .instalado-pelo-mk | wc -l)"

t "install_desktop: o .bashrc do skel ganha a linha que carrega o mk.bash, sem perder o que tinha"
rc_skel=$(<"$skel/.bashrc")
has "$rc_skel" "# Sample .bashrc"
has "$rc_skel" '[[ -f ~/.config/mk/bash/mk.bash ]] && source ~/.config/mk/bash/mk.bash'

t "install_desktop: placa comum não semeia ajustes.json no skel"
absent "$skel/.config/mk/ajustes.json"

t "install_desktop: todas as migrações nascem marcadas como feitas"
for m in "$REPO"/migrations/system/*.sh; do exists "$target/var/lib/mk/migrations/$(basename "$m")"; done
for m in "$REPO"/migrations/user/*.sh; do exists "$skel/.local/state/mk/migrations/$(basename "$m")"; done
eq "$(find "$REPO/migrations/system" -name '*.sh' | wc -l)" "$(find "$target/var/lib/mk/migrations" -type f | wc -l)"
eq "$(find "$REPO/migrations/user" -name '*.sh' | wc -l)" "$(find "$skel/.local/state/mk/migrations" -type f | wc -l)"

t "install_desktop: os apps oferecidos são gravados, tenha aceitado ou não"
eq "$(read_list "$REPO/system/packages/apps.packages")" "$(<"$target/var/lib/mk/apps-oferecidos")"

t "install_desktop: tema GTK, fontes e a licença das fontes"
exists "$target/usr/share/themes/Mk"
ok diff -r "$REPO/system/mk/look/themes/Mk" "$target/usr/share/themes/Mk"
for f in "$REPO"/system/mk/fonts/*.otf; do exists "$target/usr/share/fonts/mk/$(basename "$f")"; done
exists "$target/usr/share/licenses/mk-fonts/OFL.txt"

t "install_desktop: os ícones são gerados dentro do sistema novo"
called "arch-chroot $target bash /usr/share/mk/system/mk/look/make-icons /usr/share/icons/Mk"
exists "$share/system/mk/look/make-icons"

t "install_desktop: plymouth e sddm com o tema mk e o teclado escolhido"
exists "$target/usr/share/plymouth/themes/mk/mk.plymouth"
has "$(<"$target/etc/plymouth/plymouthd.conf")" "Theme=mk"
exists "$target/usr/share/sddm/themes/mk/Main.qml"
eq "[General]
layout=br" "$(<"$target/usr/share/sddm/themes/mk/theme.conf.user")"
eq "[Theme]
Current=mk" "$(<"$target/etc/sddm.conf.d/20-mk-theme.conf")"

t "install_desktop: nada é escrito fora do alvo"
eq "" "$(grep -v "^arch-chroot $target " "$SBX/calls")"
eq "" "$(find "$SBX/src" -newer "$SBX/marca")" "a cópia de origem não pode mudar"

t "install_desktop: numa Navi 10 o usuário novo nasce com power.sleep em 'nada'"
mkdir -p "$SYS/bus/pci/devices/0000:0b:00.0"
echo 0x030000 >"$SYS/bus/pci/devices/0000:0b:00.0/class"
echo 0x1002 >"$SYS/bus/pci/devices/0000:0b:00.0/vendor"
echo 0x731f >"$SYS/bus/pci/devices/0000:0b:00.0/device"
roda com install_desktop
eq 0 "$rc" "$out"
eq nada "$(jq -r '."power.sleep"' "$skel/.config/mk/ajustes.json")"

t "install_desktop: rodar de novo não repete a linha do .bashrc"
eq 1 "$(grep -c 'mk/bash/mk.bash' "$skel/.bashrc")"

t "install_desktop: outra placa AMD (não Navi 10) não semeia nada"
echo 0x73bf >"$SYS/bus/pci/devices/0000:0b:00.0/device"
rm -f "$skel/.config/mk/ajustes.json"
roda com install_desktop
absent "$skel/.config/mk/ajustes.json"
rm -rf "$SYS/bus/pci/devices/0000:0b:00.0"

t "install_desktop: se os ícones não saem, avisa como gerar depois e segue"
: >"$SBX/falha-bash"
roda com install_desktop
eq 0 "$rc" "$out"
has "$out" "AVISO:"
has "$out" "make-icons /usr/share/icons/Mk"
rm -f "$SBX/falha-bash"

# --- create_user -------------------------------------------------------------
t "create_user: bash, home vinda do skel, wheel e os grupos que existem"
calls_reset
printf '%s\n' wheel video audio input >"$SBX/grupos"
roda com create_user
eq 0 "$rc" "$out"
called "arch-chroot $target useradd -m -s /bin/bash -G wheel,video,audio,input mark"
exists "$target/home/mark/.config/mk/hypr/.instalado-pelo-mk"

t "create_user: com o docker instalado, entra no grupo docker também"
rm -rf "$target/home/mark"
printf '%s\n' wheel video audio input docker >"$SBX/grupos"
roda com create_user
called "useradd -m -s /bin/bash -G wheel,video,audio,input,docker mark"

t "create_user: sem nenhum grupo extra, só wheel"
rm -rf "$target/home/mark"
: >"$SBX/grupos"
calls_reset
roda com create_user
called "useradd -m -s /bin/bash -G wheel mark"

t "create_user: a senha vai para o chpasswd pela entrada, nunca nos argumentos nem na tela"
eq "mark:segredo-de-teste" "$(<"$SBX/chpasswd.entrada")"
hasnt "$(<"$SBX/calls")" "segredo-de-teste"
hasnt "$out" "segredo-de-teste"

t "create_user: senha com dois-pontos, espaço e cifrão chega inteira"
rm -rf "$target/home/mark"
roda com create_user 'PASSWORD=a:b c$d\e'
eq 'mark:a:b c$d\e' "$(<"$SBX/chpasswd.entrada")"

t "create_user: o root fica bloqueado"
called "arch-chroot $target passwd -q -l root"

t "create_user: o wheel tem sudo com senha"
eq "%wheel ALL=(ALL:ALL) ALL" "$(<"$target/etc/sudoers.d/10-mk-wheel")"
eq 440 "$(modo "$target/etc/sudoers.d/10-mk-wheel")"
hasnt "$(<"$target/etc/sudoers.d/10-mk-wheel")" "NOPASSWD"

t "create_user: se as configs do Mk não chegaram na home, a instalação para"
rm -rf "$target/home/mark"
: >"$SBX/useradd-sem-skel"
roda com create_user
eq 1 "$rc"
has "$out" "a config hypr não chegou em $target/home/mark/.config/mk"
rm -f "$SBX/useradd-sem-skel"

t "create_user: se o useradd falha, a instalação para"
rm -rf "$target/home/mark"
: >"$SBX/falha-useradd"
roda com create_user
fails test "$rc" -eq 0
rm -f "$SBX/falha-useradd"

# --- configure_snapper -------------------------------------------------------
t "configure_snapper: config root em /, sem timeline e com limite de 5"
calls_reset
roda com configure_snapper
eq 0 "$rc" "$out"
called "arch-chroot $target snapper --no-dbus -c root create-config /"
cfg=$(<"$target/etc/snapper/configs/root")
has "$cfg" 'TIMELINE_CREATE="no"'
has "$cfg" 'NUMBER_LIMIT="5"'
has "$cfg" 'NUMBER_LIMIT_IMPORTANT="5"'
has "$cfg" 'SUBVOLUME="/"'
has "$cfg" 'TIMELINE_CLEANUP="yes"'

t "configure_snapper: se o create-config falha, avisa como fazer depois e segue"
rm -f "$target/etc/snapper/configs/root"
: >"$SBX/snapper-falha"
roda com configure_snapper
eq 0 "$rc" "$out"
has "$out" "AVISO:"
has "$out" "sudo snapper -c root create-config /"
rm -f "$SBX/snapper-falha"

# --- configure_services ------------------------------------------------------
habilitados() { sed -n "s|^arch-chroot $target systemctl enable ||p" "$SBX/calls" | sort | tr '\n' ' '; }

t "configure_services: habilita as unidades do README que existem no sistema novo"
calls_reset
roda com configure_services
eq 0 "$rc" "$out"
eq "NetworkManager.service avahi-daemon.service bluetooth.service cups.socket docker.socket fstrim.timer plocate-updatedb.timer power-profiles-daemon.service sddm.service snapper-cleanup.timer systemd-timesyncd.service ufw.service " "$(habilitados)"

t "configure_services: o sshd existe mas fica desligado"
not_called "sshd"

t "configure_services: unidade que não existe não é habilitada (limine-snapper-sync sem o AUR)"
not_called "limine-snapper-sync"
rm -f "$target/usr/lib/systemd/system/bluetooth.service" "$target/usr/lib/systemd/system/docker.socket"
: >"$target/usr/lib/systemd/system/limine-snapper-sync.service"
calls_reset
roda com configure_services
hasnt "$(habilitados)" "bluetooth"
hasnt "$(habilitados)" "docker"
has "$(habilitados)" "limine-snapper-sync.service"

t "configure_services: o agente SSH do chaveiro é ligado para todos os usuários"
called "arch-chroot $target systemctl --global enable gcr-ssh-agent.socket"
rm -f "$target/usr/lib/systemd/user/gcr-ssh-agent.socket"
calls_reset
roda com configure_services
not_called "gcr-ssh-agent"

t "configure_services: firewall nega entrada, libera saída, LocalSend e DNS do Docker; liga no primeiro boot"
# O sh -c ocupa várias linhas do registro: tudo o que não é systemctl nem chown.
regras=$(grep -v -e 'systemctl' -e 'chown' "$SBX/calls")
has "$regras" "arch-chroot $target sh -c"
has "$regras" "ufw default deny incoming"
has "$regras" "ufw default allow outgoing"
has "$regras" "ufw allow 53317/tcp"
has "$regras" "ufw allow 53317/udp"
has "$regras" "ufw allow in proto udp from 172.16.0.0/12 to 172.17.0.1 port 53"
has "$(<"$target/etc/ufw/ufw.conf")" "ENABLED=yes"
has "$(<"$target/etc/ufw/ufw.conf")" "LOGLEVEL=low"

t "configure_services: as regras são as mesmas do scripts/system/firewall.sh"
while read -r regra; do
  has "$regras" "$regra"
done < <(sed -n "s/^sudo \(ufw \(default\|allow\) [^']*[^' ]\).*/\1/p" "$REPO/scripts/system/firewall.sh" | sed 's/ comment$//')
ok test "$(grep -c '^sudo ufw \(default\|allow\)' "$REPO/scripts/system/firewall.sh")" -eq 5

t "configure_services: não roda o ufw-docker install nem liga o ufw no chroot"
hasnt "$regras" "ufw-docker"
hasnt "$regras" "enable"

t "configure_services: sem o ufw instalado, não grava regra nenhuma"
rm -f "$target/usr/bin/ufw"
calls_reset
roda com configure_services
eq 0 "$rc" "$out"
not_called "ufw default"
printf '#!/bin/sh\n' >"$target/usr/bin/ufw"
chmod +x "$target/usr/bin/ufw"

t "configure_services: se as regras não gravam, avisa qual script rodar depois"
: >"$SBX/falha-sh"
roda com configure_services
eq 0 "$rc" "$out"
has "$out" "regras do ufw não gravadas"
has "$out" "/usr/share/mk/scripts/system/firewall.sh"
rm -f "$SBX/falha-sh"

t "configure_services: se uma unidade não habilita, avisa e segue"
: >"$SBX/falha-systemctl"
roda com configure_services
eq 0 "$rc" "$out"
has "$out" "não habilitou sddm.service"
rm -f "$SBX/falha-systemctl"

t "configure_services: o wheel entra no SystemGroup do CUPS, uma vez só"
eq "SystemGroup sys root wheel" "$(<"$target/etc/cups/cups-files.conf")"
roda com configure_services
eq "SystemGroup sys root wheel" "$(<"$target/etc/cups/cups-files.conf")"

t "configure_services: CUPS sem SystemGroup ganha a linha; sem CUPS, nada é criado"
printf '#SystemGroup sys root\nUser cups\n' >"$target/etc/cups/cups-files.conf"
roda com configure_services
eq "SystemGroup sys root wheel" "$(tail -1 "$target/etc/cups/cups-files.conf")"
rm -f "$target/etc/cups/cups-files.conf"
roda com configure_services
absent "$target/etc/cups/cups-files.conf"

t "configure_services: o sddm já vem com o usuário e a sessão Mk escolhidos, sem entrar sozinho"
eq "[Last]
Session=/usr/share/wayland-sessions/mk.desktop
User=mark" "$(<"$target/var/lib/sddm/state.conf")"
absent "$target/etc/sddm.conf.d/10-mk.conf"

t "configure_services com LUKS: o sddm entra direto no usuário, na sessão mk"
roda com configure_services ENCRYPT=sim USERNAME=ana
eq "[Autologin]
User=ana
Session=mk" "$(<"$target/etc/sddm.conf.d/10-mk.conf")"
has "$(<"$target/var/lib/sddm/state.conf")" "User=ana"

# --- install_aur -------------------------------------------------------------
lista=$aur_repo/packages.txt
mkdir -p "$share"

t "install_aur: ISO sem pacotes do AUR só avisa; não chama o pacstrap nem grava pendentes"
calls_reset
rm -f "$lista" "$share/aur-pendentes.txt"
roda com install_aur
eq 0 "$rc" "$out"
has "$out" "nenhum pacote do AUR no ISO"
not_called "pacstrap"
absent "$share/aur-pendentes.txt"
: >"$lista"
roda com install_aur
not_called "pacstrap"

t "install_aur: instala exatamente a lista do packages.txt, sem copiar chaveiro e espelhos de novo"
printf '%s\n' yay-bin limine-mkinitcpio-hook limine-snapper-sync >"$lista"
read_list "$PKG_DIR/aur.packages" >"$SBX/instalados"
calls_reset
roda com install_aur
eq 0 "$rc" "$out"
called "pacstrap -G -M -C $SBX/pacman.conf $target yay-bin limine-mkinitcpio-hook limine-snapper-sync"
eq 1 "$(grep -c '^pacstrap' "$SBX/calls")"

t "install_aur: com tudo instalado, não há arquivo de pendentes"
absent "$share/aur-pendentes.txt"
hasnt "$out" "ficaram faltando"

t "install_aur: o que não veio no ISO fica anotado em aur-pendentes.txt, e a instalação segue"
printf '%s\n' yay-bin limine-mkinitcpio-hook >"$SBX/instalados"
roda com install_aur
eq 0 "$rc" "$out"
has "$out" "ficaram faltando"
has "$out" "yay -S "
pend=$(<"$share/aur-pendentes.txt")
ok grep -qx limine-snapper-sync <<<"$pend"
ok grep -qx localsend-bin <<<"$pend"
fails grep -qx yay-bin <<<"$pend"
eq "$(($(read_list "$PKG_DIR/aur.packages" | wc -l) - 2))" "$(wc -l <<<"$pend")"

t "install_aur: se a instalação conjunta falha, tenta um por um e avisa de cada um que não entrou"
stub pacstrap '[[ $# -gt 6 || ${*: -1} == limine-snapper-sync ]] && exit 1; exit 0'
calls_reset
roda com install_aur
eq 0 "$rc" "$out"
has "$out" "a instalação conjunta falhou"
has "$out" "não instalou limine-snapper-sync"
hasnt "$out" "não instalou yay-bin"
eq 4 "$(grep -c '^pacstrap' "$SBX/calls")"
called "pacstrap -G -M -C $SBX/pacman.conf $target yay-bin"
stub pacstrap

# --- user_extras -------------------------------------------------------------
t "user_extras: o LazyVim é aplicado como o usuário, na home dele"
calls_reset
roda com user_extras
eq 0 "$rc" "$out"
called "arch-chroot -u mark $target env HOME=/home/mark bash /usr/share/mk/scripts/user/neovim.sh"

t "user_extras: se falhar, avisa como aplicar depois e segue"
: >"$SBX/falha-env"
roda com user_extras
eq 0 "$rc" "$out"
has "$out" "LazyVim não aplicado"
rm -f "$SBX/falha-env"

# --- o que o instalador espera do repositório -------------------------------
t "todo arquivo do repositório que o instalador usa existe"
n=0
while IFS= read -r c; do
  case $c in aur-repo | iso/work/aur/repo) continue ;; esac # saídas do iso/build
  n=$((n + 1))
  ok compgen -G "$REPO/${c%/.}" >/dev/null
done < <(grep -o -e '\$src/[A-Za-z0-9_./*-]*' -e '"\$src"/[A-Za-z0-9_./*-]*' "$REPO/installer/mk-install" |
  sed -e 's|^"\$src"/||' -e 's|^\$src/||' | sort -u)
ok test "$n" -ge 20

t "os scripts que o instalador roda dentro do sistema novo existem no repositório"
while IFS= read -r c; do
  exists "$REPO/$c"
done < <(grep -o '/usr/share/mk/[a-z][A-Za-z0-9_./-]*' "$REPO/installer/mk-install" |
  sed 's|^/usr/share/mk/||' | grep -v -e '^aur-' -e '^VERSION$' -e '^system/packages$' | sort -u)

# --- as permissões que o ISO tira ----------------------------------------------
# Visto na VM: o mkarchiso grava tudo em /usr/share/mk do live como 644 (só o
# que o profiledef lista ganha outro modo). O instalador copiava assim, e o Mk
# instalado ficava sem nenhum script executável: `mktunes` (um link para lá)
# não rodava e `tests/run` dava "Permissão negada".
t "install_desktop: com a fonte sem bits de execução (como no ISO), os scripts voltam a ser executáveis"
find "$SBX/src" -type f -exec chmod a-x {} +
rm -rf "${share:?}" "${skel:?}/.config/mk"
roda com install_desktop
eq 0 "$rc" "$out"
for f in bin/mk-update bin/mk-session tests/run tests/hooks/pre-push shell/mare/scripts/mare-caps \
  system/mk/boot/limine-default installer/mk-install apps/mktunes/bin/mktunes iso/build; do
  [[ -e $share/$f ]] || continue
  [[ -x $share/$f ]] || _fail "sem +x em /usr/share/mk/$f"
done
ok test -x "$share/bin/mk-update"
ok test -x "$share/shell/mare/scripts/mare-caps"

t "install_desktop: um binário compilado (o coletor do MkPulso) também volta a ser executável"
# O fix_exec olhava só o #! dos scripts: um ELF copiado do ISO ficaria 644.
mkdir -p "$SBX/src/apps/elf"
printf '\177ELF\002\001\001\000binario' >"$SBX/src/apps/elf/programa"
printf 'ELF não: texto que começa parecido' >"$SBX/src/apps/elf/texto"
chmod a-x "$SBX/src/apps/elf/programa" "$SBX/src/apps/elf/texto"
rm -rf "${share:?}" "${skel:?}/.config/mk"
roda com install_desktop
eq 0 "$rc" "$out"
ok test -x "$share/apps/elf/programa"
fails test -x "$share/apps/elf/texto"
[[ ! -e $share/apps/mkpulso/bin/mkpulso-coletor ]] || ok test -x "$share/apps/mkpulso/bin/mkpulso-coletor"

t "install_desktop: o MkPulso fica ligado por links para /usr/share/mk"
eq /usr/share/mk/apps/mkpulso/bin/mkpulso "$(readlink "$target/usr/bin/mkpulso")"
eq /usr/share/mk/apps/mkpulso/share/mkpulso.desktop "$(readlink "$target/usr/share/applications/mkpulso.desktop")"

t "install_desktop: o que não é script continua sem bit de execução"
for f in README.md system/packages/base.packages config/hypr/hyprland.lua shell/mare/Mare.qml bin/lib-packages.sh; do
  [[ ! -x $share/$f ]] || _fail "+x indevido em /usr/share/mk/$f"
done

t "install_desktop: os scripts do Maré no skel (o ~/.config/mk de quem é criado) são executáveis"
ok test -x "$skel/.config/mk/shell/scripts/mare-caps"
ok test -x "$skel/.config/mk/shell/scripts/mare-start"
