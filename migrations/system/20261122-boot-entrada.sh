#!/usr/bin/env bash
# O menu de boot inicia sozinho (com os ganchos do limine a entrada 1 virou
# um grupo e a contagem não iniciava nada), a entrada mostra o teclado certo,
# e o ble.sh 0.3 dá lugar ao 0.4 (blesh-git), que tem as cores do Mk.
set -euo pipefail
share=${MK_SHARE:-/usr/share/mk}

bash "$share/system/mk/boot/limine-default"

rm -rf /usr/share/sddm/themes/mk.novo
cp -r "$share/system/mk/sddm/mk" /usr/share/sddm/themes/mk.novo
layout=$(sed -n 's/.*XkbLayout" *"\([^"]*\)".*/\1/p' /etc/X11/xorg.conf.d/00-keyboard.conf 2>/dev/null | head -1)
[[ -n $layout ]] && printf '[General]\nlayout=%s\n' "$layout" >/usr/share/sddm/themes/mk.novo/theme.conf.user
rm -rf /usr/share/sddm/themes/mk
mv /usr/share/sddm/themes/mk.novo /usr/share/sddm/themes/mk

if pacman -Qq blesh >/dev/null 2>&1; then
  pacman -Rns --noconfirm blesh
  grep -qx blesh-git "$share/aur-pendentes.txt" 2>/dev/null || echo blesh-git >>"$share/aur-pendentes.txt"
fi
