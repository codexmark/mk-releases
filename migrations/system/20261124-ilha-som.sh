#!/usr/bin/env bash
# A ilha responde ao som: precisa do cava para ouvir o espectro.
set -euo pipefail
pacman -S --needed --noconfirm cava
