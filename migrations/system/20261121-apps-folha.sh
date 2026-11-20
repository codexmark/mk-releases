#!/usr/bin/env bash
# Os apps na paleta Folha: o tema GTK do Mk, os ícones com as pastas em osso
# e a fonte Space Grotesk (a sans do Maré, que faltava nos Mk instalados).
set -euo pipefail
share=${MK_SHARE:-/usr/share/mk}

pacman -S --needed --noconfirm adw-gtk-theme papirus-icon-theme
rm -rf /usr/share/themes/Mk
cp -r "$share/system/mk/look/themes/Mk" /usr/share/themes/Mk
bash "$share/system/mk/look/make-icons" /usr/share/icons/Mk
install -Dm644 -t /usr/share/fonts/mk "$share"/system/mk/fonts/*.otf
install -Dm644 "$share/system/mk/fonts/OFL.txt" /usr/share/licenses/mk-fonts/OFL.txt
fc-cache -f >/dev/null 2>&1 || true
if [[ ! -e /etc/skel/.config/qt6ct ]]; then
  mkdir -p /etc/skel/.config/qt6ct
  cp -r "$share/system/mk/look/qt6ct/." /etc/skel/.config/qt6ct/
fi
