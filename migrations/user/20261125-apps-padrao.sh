#!/usr/bin/env bash
# Os apps padrão do Mk (navegador, arquivos, texto, imagens, vídeo, música,
# PDF) para quem ainda não tem um ~/.config/mimeapps.list. Quem já tem fica com
# o seu: a página "Apps padrão" dos Ajustes troca o que for preciso.
set -euo pipefail
share=${MK_SHARE:-/usr/share/mk}
dst=${XDG_CONFIG_HOME:-$HOME/.config}/mimeapps.list

if [[ -e $dst ]]; then
  echo "mantido (já existe): $dst"
else
  install -Dm644 "$share/config/mimeapps.list" "$dst"
fi
