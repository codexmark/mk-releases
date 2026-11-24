#!/usr/bin/env bash
# O conjunto de apps do dia a dia cresceu (editor de texto, calculadora,
# libreoffice, loupe, papers...). Esta migração não instala nada: só anota os
# apps que esta máquina já viu, e o mk-update oferece os que entraram depois
# junto com a atualização do sistema (uma vez, perguntando).
set -euo pipefail
R=${MK_ROOT:-} # outra raiz no lugar de /: só os testes trocam (migrations/README.md)
visto=$R/var/lib/mk/apps-oferecidos
[[ -e $visto ]] && exit 0
lista="${MK_SHARE:-/usr/share/mk}/system/packages/apps.packages"
mkdir -p "$(dirname "$visto")"
if pacman -Q nautilus &>/dev/null; then
  # O que o Mk trazia até aqui: o que você tirou disso não volta.
  printf '%s\n' chromium nautilus sushi ffmpegthumbnailer imv evince mpv mpv-mpris \
    ffmpeg imagemagick yt-dlp >"$visto"
elif [[ -r $lista ]]; then
  # Sem os apps gráficos (pulados na instalação): nada a oferecer.
  sed -e 's/#.*//' -e 's/[[:space:]]*$//' "$lista" | grep -v '^[[:space:]]*$' >"$visto"
else
  : >"$visto"
fi
