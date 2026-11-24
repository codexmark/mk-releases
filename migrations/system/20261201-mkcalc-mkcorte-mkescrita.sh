#!/usr/bin/env bash
# Mais três apps do Mk, prontos em apps/: o MkCalc (calculadora de contas
# exatas, com histórico), o MkCorte (aparar, dividir, juntar e exportar vídeos)
# e o MkEscrita (escrever em Markdown, com salvamento automático). Aqui ficam
# os comandos, os atalhos no menu e os ícones. O MkCorte mostra o vídeo pelo
# QtMultimedia (com o ffmpeg por trás).
# pacotes: qt6-multimedia qt6-multimedia-ffmpeg
set -euo pipefail
share=${MK_SHARE:-/usr/share/mk}
R=${MK_ROOT:-} # outra raiz no lugar de /: só os testes trocam (migrations/README.md)

bash "$share/system/mk/apps-links" "$R/"
