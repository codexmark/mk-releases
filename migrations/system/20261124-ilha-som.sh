#!/usr/bin/env bash
# A ilha responde ao som: precisa do cava para ouvir o espectro.
# Quem instala é o mk-update, junto com a atualização do sistema.
# pacotes: cava
set -euo pipefail
pacman -Q cava &>/dev/null || echo "aviso: o cava não foi instalado; a ilha não responde ao som" >&2
