#!/usr/bin/env bash
# Os papéis de parede passam a ficar na pasta de imagens do sistema (~/Imagens
# em português). Quem tinha ~/Pictures/Wallpapers criada pelo instalador: as
# imagens vão para a pasta nova e a ~/Pictures vazia some.
set -euo pipefail

pics=$(xdg-user-dir PICTURES 2>/dev/null || true)
old=$HOME/Pictures
[[ -n $pics && $pics != "$HOME" && $pics != "$old" && -d $old/Wallpapers ]] || exit 0

mkdir -p "$pics/Wallpapers"
find "$old/Wallpapers" -maxdepth 1 -type f -exec mv -n {} "$pics/Wallpapers/" \;
# O papel de parede escolhido aponta para o caminho antigo.
find "${XDG_STATE_HOME:-$HOME/.local/state}/quickshell" -name wallpaper.json 2>/dev/null |
  while IFS= read -r f; do sed -i "s|$old/Wallpapers/|$pics/Wallpapers/|" "$f"; done
rmdir "$old/Wallpapers" "$old" 2>/dev/null || true
