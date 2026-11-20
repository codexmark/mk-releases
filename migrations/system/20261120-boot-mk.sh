#!/usr/bin/env bash
# Boot com a cara do Mk: tema do plymouth e o initramfs refeito com ele.
set -euo pipefail
share=${MK_SHARE:-/usr/share/mk}

rm -rf /usr/share/plymouth/themes/mk
cp -r "$share/system/mk/plymouth/mk" /usr/share/plymouth/themes/mk
mkdir -p /etc/plymouth
printf '[Daemon]\nTheme=mk\nShowDelay=0\n' >/etc/plymouth/plymouthd.conf
# O tema vai dentro do initramfs: sem refazer, o boot continua com o antigo.
mkinitcpio -P
