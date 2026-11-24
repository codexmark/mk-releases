#!/usr/bin/env bash
# Três apps novos do Mk, prontos em apps/: o MkPacotes (procurar, instalar e
# remover pacotes, com o histórico), o MkInstalar (AppImage, pacote do Arch ou
# arquivo compactado) e o MkComprimir (vídeos e imagens menores, pelo ffmpeg).
# Aqui ficam os comandos, os atalhos no menu e os ícones. Os pacotes que eles
# usam o Mk já tem (ffmpeg, jq, pacman-contrib, zstd, xz, libarchive).
set -euo pipefail
share=${MK_SHARE:-/usr/share/mk}
R=${MK_ROOT:-} # outra raiz no lugar de /: só os testes trocam (migrations/README.md)

bash "$share/system/mk/apps-links" "$R/"
