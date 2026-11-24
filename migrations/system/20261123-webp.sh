#!/usr/bin/env bash
# Papéis de parede em WebP (e AVIF, TIFF) no Maré: faltava o qt6-imageformats.
# Quem instala é o mk-update, junto com a atualização do sistema.
# pacotes: qt6-imageformats
set -euo pipefail
pacman -Q qt6-imageformats &>/dev/null ||
  echo "aviso: o qt6-imageformats não foi instalado; papéis de parede em WebP não abrem" >&2
