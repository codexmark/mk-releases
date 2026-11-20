#!/usr/bin/env bash
# O logo novo (o Mk em serifa reta sobre o plano cartesiano) no boot, na tela
# de entrada e no terminal dos usuários novos.
set -euo pipefail
share=${MK_SHARE:-/usr/share/mk}

rm -rf /usr/share/plymouth/themes/mk
cp -r "$share/system/mk/plymouth/mk" /usr/share/plymouth/themes/mk
rm -rf /usr/share/sddm/themes/mk
cp -r "$share/system/mk/sddm/mk" /usr/share/sddm/themes/mk
if [[ -d /etc/skel/.config/mk/fastfetch ]]; then
  cp "$share/config/fastfetch/emblem.png" /etc/skel/.config/mk/fastfetch/emblem.png
fi
# O logo do boot vai dentro do initramfs.
mkinitcpio -P
