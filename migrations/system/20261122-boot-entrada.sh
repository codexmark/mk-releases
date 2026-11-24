#!/usr/bin/env bash
# O menu de boot inicia sozinho (com os ganchos do limine a entrada 1 virou
# um grupo e a contagem não iniciava nada), a entrada mostra o teclado certo,
# e o ble.sh 0.3 dá lugar ao 0.4 (blesh-git), que tem as cores do Mk.
set -euo pipefail
share=${MK_SHARE:-/usr/share/mk}
R=${MK_ROOT:-} # outra raiz no lugar de /: só os testes trocam (migrations/README.md)

bash "$share/system/mk/boot/limine-default"

themes=$R/usr/share/sddm/themes
rm -rf "$themes/mk.novo"
cp -r "$share/system/mk/sddm/mk" "$themes/mk.novo"
# Sem o arquivo do teclado (ou sem layout nele), o tema fica com o padrão.
layout=$(sed -n 's/.*XkbLayout" *"\([^"]*\)".*/\1/p' "$R/etc/X11/xorg.conf.d/00-keyboard.conf" 2>/dev/null | head -1) || true
[[ -n $layout ]] && printf '[General]\nlayout=%s\n' "$layout" >"$themes/mk.novo/theme.conf.user"
rm -rf "$themes/mk"
mv "$themes/mk.novo" "$themes/mk"

pend=$share/aur-pendentes.txt
if pacman -Qq blesh >/dev/null 2>&1; then
  pacman -Rns --noconfirm blesh
  grep -qx blesh-git "$pend" 2>/dev/null || echo blesh-git >>"$pend"
fi
# O 0.3 que ainda estava na fila do AUR (não veio no pendrive) também troca.
if grep -qx blesh "$pend" 2>/dev/null; then
  if grep -qx blesh-git "$pend"; then
    sed -i '/^blesh$/d' "$pend"
  else
    sed -i 's/^blesh$/blesh-git/' "$pend"
  fi
fi
