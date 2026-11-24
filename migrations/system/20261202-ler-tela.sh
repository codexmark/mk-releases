#!/usr/bin/env bash
# A captura ganhou o "ler" (SUPER + SHIFT + T): escolhe uma área e copia o que
# está escrito nela, ou o que um QR code diz. O texto é lido pelo tesseract (com
# o português e o inglês) e o código pelo zbar. Quem instala é o mk-update,
# junto com a atualização do sistema.
# pacotes: tesseract tesseract-data-por tesseract-data-eng zbar
set -euo pipefail
for p in tesseract tesseract-data-por tesseract-data-eng zbar; do
  pacman -Q "$p" &>/dev/null ||
    echo "aviso: o $p não foi instalado; o \"ler\" da captura fica sem ele" >&2
done
