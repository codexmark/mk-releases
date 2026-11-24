#!/usr/bin/env bash
# Boot com a cara do Mk: tema do plymouth e o initramfs refeito com ele.
set -euo pipefail
share=${MK_SHARE:-/usr/share/mk}
R=${MK_ROOT:-} # outra raiz no lugar de /: só os testes trocam (migrations/README.md)

rm -rf "$R/usr/share/plymouth/themes/mk"
cp -r "$share/system/mk/plymouth/mk" "$R/usr/share/plymouth/themes/mk"
mkdir -p "$R/etc/plymouth"
printf '[Daemon]\nTheme=mk\nShowDelay=0\n' >"$R/etc/plymouth/plymouthd.conf"
# O tema vai dentro do initramfs: sem refazer, o boot continua com o antigo.
mkinitcpio -P
