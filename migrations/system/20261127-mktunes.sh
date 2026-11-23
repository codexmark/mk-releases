#!/usr/bin/env bash
# O MkTunes entra no Mk: o player de música vem pronto em apps/mktunes e o
# mk-update o troca junto com o resto. Aqui ficam o comando mktunes, o atalho
# no menu e o ícone (links para a pasta do Mk) e o que ele usa do sistema: o
# Java só de linha de comando, o mpv para tocar e o ffprobe para as tags.
# pacotes: jre-openjdk-headless mpv ffmpeg
set -euo pipefail
share=${MK_SHARE:-/usr/share/mk}

bash "$share/system/mk/mktunes-links" /
command -v java >/dev/null ||
  echo "aviso: sem Java; o MkTunes abre depois de: sudo pacman -S jre-openjdk-headless" >&2
