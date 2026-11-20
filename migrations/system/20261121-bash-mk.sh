#!/usr/bin/env bash
# O bash novo do Mk: o ble.sh (do AUR, já compilado no pendrive ou pelo yay
# depois) e o 7zip; e o seu.bash no skel dos usuários novos.
set -euo pipefail
share=${MK_SHARE:-/usr/share/mk}

pacman -S --needed --noconfirm 7zip || true
[[ -e /etc/skel/.config/mk/seu.bash ]] || install -Dm644 "$share/config/bash/seu.bash.exemplo" /etc/skel/.config/mk/seu.bash
rm -rf /etc/skel/.config/mk/bash
cp -r "$share/config/bash" /etc/skel/.config/mk/bash
touch /etc/skel/.config/mk/bash/.instalado-pelo-mk
# O ble.sh vem do AUR: o mk-update instala depois das migrações (é um pacote
# do AUR que ainda não está instalado).
if ! pacman -Qq blesh >/dev/null 2>&1 && ! grep -qx blesh "$share/aur-pendentes.txt" 2>/dev/null; then
  echo blesh >>"$share/aur-pendentes.txt"
fi
