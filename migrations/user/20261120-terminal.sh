#!/usr/bin/env bash
# O terminal do Mk para quem já tinha o Mk: a ficha do fastfetch, o prompt
# e a linha no ~/.bashrc que carrega os dois.
set -euo pipefail
share=${MK_SHARE:-/usr/share/mk}
cfg=${XDG_CONFIG_HOME:-$HOME/.config}/mk

for d in fastfetch bash; do
  dst=$cfg/$d
  if [[ -e $dst && ! -e $dst/.instalado-pelo-mk ]]; then
    echo "mantido (não é do Mk): $dst"
    continue
  fi
  rm -rf "${dst:?}"
  mkdir -p "$cfg"
  cp -r "$share/config/$d" "$dst"
  touch "$dst/.instalado-pelo-mk"
done
if ! grep -q 'mk/bash/mk.bash' "$HOME/.bashrc" 2>/dev/null; then
  printf '\n# Mk: ficha do fastfetch, prompt e atalhos\n[[ -f ~/.config/mk/bash/mk.bash ]] && source ~/.config/mk/bash/mk.bash\n' >>"$HOME/.bashrc"
fi
