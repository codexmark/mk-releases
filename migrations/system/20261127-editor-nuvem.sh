#!/usr/bin/env bash
# VS Code (o build aberto, `code`) e as CLIs da AWS e da Cloudflare entram na
# instalação padrão; a da Vercel é do AUR e fica na fila do yay, que o
# mk-update oferece no fim da rodada.
# pacotes: code aws-cli-v2 wrangler
set -euo pipefail
share=${MK_SHARE:-/usr/share/mk}

if ! pacman -Qq vercel >/dev/null 2>&1; then
  grep -qx vercel "$share/aur-pendentes.txt" 2>/dev/null || echo vercel >>"$share/aur-pendentes.txt"
fi
for p in code aws-cli-v2 wrangler; do
  pacman -Qq "$p" >/dev/null 2>&1 || echo "aviso: $p não foi instalado; depois: sudo pacman -S --needed $p" >&2
done
