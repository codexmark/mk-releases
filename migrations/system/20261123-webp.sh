#!/usr/bin/env bash
# Papéis de parede em WebP (e AVIF, TIFF) no Maré: faltava o qt6-imageformats.
set -euo pipefail
pacman -S --needed --noconfirm qt6-imageformats
