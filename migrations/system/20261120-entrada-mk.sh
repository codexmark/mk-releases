#!/usr/bin/env bash
# Tela de entrada (sddm) com a cara do Mk.
set -euo pipefail
share=${MK_SHARE:-/usr/share/mk}

rm -rf /usr/share/sddm/themes/mk
cp -r "$share/system/mk/sddm/mk" /usr/share/sddm/themes/mk
mkdir -p /etc/sddm.conf.d
printf '[Theme]\nCurrent=mk\n' >/etc/sddm.conf.d/20-mk-theme.conf
