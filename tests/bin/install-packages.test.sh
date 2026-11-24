#!/usr/bin/env bash
# bin/install-packages: as listas certas vão para o pacman e para o yay, com
# listas falsas (PKG_DIR) e sudo, pacman, yay, git e makepkg por stub.
. "$(dirname "$0")/../lib.sh"

export PKG_DIR=$SBX/pacotes
mkdir -p "$PKG_DIR"
listas() {
  printf '# base\nbase\nlinux   # o kernel\n\n' >"$PKG_DIR/base.packages"
  printf 'hyprland\nquickshell\n' >"$PKG_DIR/desktop.packages"
  printf 'git\nneovim\n' >"$PKG_DIR/dev.packages"
  printf '# apps\nchromium\nnautilus\n' >"$PKG_DIR/apps.packages"
  printf '# AUR\nyay-bin\nlimine-snapper-sync  # snapshots no boot\n' >"$PKG_DIR/aur.packages"
  printf '[amd]\nvulkan-radeon\n[intel]\nvulkan-intel\n[nvidia]\nnvidia-open-dkms\n[vm]\nqemu-guest-agent\n' >"$PKG_DIR/hardware.packages"
}
listas

# O sudo falso só executa o que vem depois (o pacman falso).
stub sudo 'exec "$@"'
stub pacman 'exit "$(cat "$SBX/pacman-rc" 2>/dev/null || echo 0)"'
stub yay 'grep -qxF -- "${@: -1}" "$SBX/yay-falha" 2>/dev/null && exit 1; exit 0'
instala() { bash "$REPO/bin/install-packages" "$@" 2>&1; }

t "sem perfil de hardware: pede um e sai com 2, sem instalar nada"
out=$(instala)
eq 2 $?
has "$out" "informe o perfil de hardware"
eq "" "$(cat "$SBX/calls")"

t "só --no-apps, sem perfil: também sai com 2"
instala --no-apps >/dev/null
eq 2 $?
eq "" "$(cat "$SBX/calls")"

t "argumento desconhecido: diz qual e sai com 2, sem instalar nada"
out=$(instala amd --tudo)
eq 2 $?
has "$out" "argumento desconhecido: --tudo"
eq "" "$(cat "$SBX/calls")"

t "um perfil: base, desktop, dev, apps e o hardware dele, numa chamada só do pacman"
instala amd >/dev/null
eq 0 $?
eq "sudo pacman -S --needed --noconfirm base linux hyprland quickshell git neovim chromium nautilus vulkan-radeon" \
  "$(grep '^sudo' "$SBX/calls")"

t "os pacotes dos outros perfis não entram"
not_called "vulkan-intel"
not_called "nvidia-open-dkms"
not_called "qemu-guest-agent"

t "comentários e linhas vazias das listas não viram argumento"
not_called "#"
not_called "kernel"

t "AUR: depois dos oficiais, um yay por pacote, com --needed"
eq "sudo pacman -S --needed --noconfirm base linux hyprland quickshell git neovim chromium nautilus vulkan-radeon
pacman -S --needed --noconfirm base linux hyprland quickshell git neovim chromium nautilus vulkan-radeon
yay -S --needed --noconfirm yay-bin
yay -S --needed --noconfirm limine-snapper-sync" "$(cat "$SBX/calls")"
not_called "git clone"
not_called "makepkg"

t "idempotente: rodar de novo faz as mesmas chamadas (o --needed pula o que já existe)"
primeira=$(cat "$SBX/calls")
calls_reset
instala amd >/dev/null
eq 0 $?
eq "$primeira" "$(cat "$SBX/calls")"

t "dois perfis: o hardware dos dois"
calls_reset
instala amd nvidia >/dev/null
called "nautilus vulkan-radeon nvidia-open-dkms"

t "--no-apps: a lista de apps fica de fora, em qualquer posição do argumento"
calls_reset
instala --no-apps vm >/dev/null
eq 0 $?
called "sudo pacman -S --needed --noconfirm base linux hyprland quickshell git neovim qemu-guest-agent"
not_called "chromium"
calls_reset
instala intel --no-apps >/dev/null
called "git neovim vulkan-intel"
not_called "chromium"

t "um pacote do AUR que falha só avisa: os seguintes são tentados e a saída é 0"
calls_reset
echo yay-bin >"$SBX/yay-falha"
out=$(instala amd)
eq 0 $?
has "$out" "AVISO: yay-bin (AUR) falhou, siga em frente"
called "yay -S --needed --noconfirm limine-snapper-sync"
rm "$SBX/yay-falha"

t "o pacman falha: para ali, com erro, e o AUR não é tentado"
calls_reset
echo 1 >"$SBX/pacman-rc"
instala amd >/dev/null
[[ $? -ne 0 ]] || _fail "devia sair com erro"
not_called "yay"
rm "$SBX/pacman-rc"

t "uma lista só com comentários não derruba as listas seguintes"
calls_reset
printf '# nenhum app por enquanto\n' >"$PKG_DIR/apps.packages"
instala amd >/dev/null
eq 0 $?
called "sudo pacman -S --needed --noconfirm base linux hyprland quickshell git neovim vulkan-radeon"
listas

t "aur.packages só com comentários: nenhum yay"
calls_reset
printf '# nada do AUR\n' >"$PKG_DIR/aur.packages"
instala amd >/dev/null
eq 0 $?
not_called "yay"
listas

# Sem o yay: um PATH só com os stubs e as ferramentas de que o script precisa
# (o yay de verdade da máquina não pode ficar ao alcance).
mkdir -p "$SBX/min"
for c in bash env sed grep mktemp rm dirname mkdir chmod cat; do
  ln -s "$(PATH=/usr/bin:/bin command -v "$c")" "$SBX/min/$c"
done

t "sem o yay: compila o yay-bin do AUR (git clone + makepkg) e depois instala o AUR com ele"
calls_reset
rm "$SBX/bin/yay"
stub git 'mkdir -p "${@: -1}"'
# O makepkg -si "instala" o yay: daqui em diante ele existe.
stub makepkg 'printf "#!/bin/bash\necho \"yay \$*\" >>\"$SBX/calls\"\n" >"$SBX/bin/yay"; chmod +x "$SBX/bin/yay"'
out=$(PATH=$SBX/bin:$SBX/min instala amd)
eq 0 $?
eq "git clone --depth 1 https://aur.archlinux.org/yay-bin.git
makepkg -si --noconfirm
yay -S --needed --noconfirm yay-bin
yay -S --needed --noconfirm limine-snapper-sync" "$(grep -v pacman "$SBX/calls" | sed 's| /[^ ]*/yay-bin$||')"

t "sem o yay: a pasta temporária do yay-bin é apagada no fim"
eq 0 "$(find "$TMPDIR" -mindepth 1 -maxdepth 1 | wc -l)"
