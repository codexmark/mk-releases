#!/usr/bin/env bash
# O daemon do MkTunes passou a ser um binário em Go, que vai pronto em
# apps/mktunes/bin: a máquina virtual que ele pedia não é mais usada. O sqlite3
# serve uma vez, para levar a biblioteca do banco antigo para os arquivos
# novos. Nada é removido daqui: quem não usa mais a máquina virtual em outro
# programa tira o pacote à mão.
# pacotes: sqlite mpv ffmpeg
set -euo pipefail
share=${MK_SHARE:-/usr/share/mk}
R=${MK_ROOT:-} # outra raiz no lugar de /: só os testes trocam (migrations/README.md)

bash "$share/system/mk/mktunes-links" "$R/"
