#!/usr/bin/env bash
# O MkTunes entra no Mk: o player de música vem pronto em apps/mktunes e o
# mk-update o troca junto com o resto. Aqui ficam o comando mktunes, o atalho
# no menu e o ícone (links para a pasta do Mk) e o que ele usa do sistema: o
# mpv para tocar e o ffprobe para as tags. O resto do que o daemon de hoje
# pede está na 20261207-mktunes-go.sh.
# pacotes: mpv ffmpeg
set -euo pipefail
share=${MK_SHARE:-/usr/share/mk}
R=${MK_ROOT:-} # outra raiz no lugar de /: só os testes trocam (migrations/README.md)

bash "$share/system/mk/mktunes-links" "$R/"
