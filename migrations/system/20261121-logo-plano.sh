#!/usr/bin/env bash
# O logo novo (o Mk em serifa reta sobre o plano cartesiano) no boot, na tela
# de entrada e no terminal dos usuários novos.
set -euo pipefail
share=${MK_SHARE:-/usr/share/mk}
R=${MK_ROOT:-} # outra raiz no lugar de /: só os testes trocam (migrations/README.md)

rm -rf "$R/usr/share/plymouth/themes/mk"
cp -r "$share/system/mk/plymouth/mk" "$R/usr/share/plymouth/themes/mk"
rm -rf "$R/usr/share/sddm/themes/mk"
cp -r "$share/system/mk/sddm/mk" "$R/usr/share/sddm/themes/mk"
if [[ -d $R/etc/skel/.config/mk/fastfetch ]]; then
  rm -rf "$R/etc/skel/.config/mk/fastfetch"
  cp -r "$share/config/fastfetch" "$R/etc/skel/.config/mk/fastfetch"
  touch "$R/etc/skel/.config/mk/fastfetch/.instalado-pelo-mk"
fi
# O logo do boot vai dentro do initramfs.
mkinitcpio -P
