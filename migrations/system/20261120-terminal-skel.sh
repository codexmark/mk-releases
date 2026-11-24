#!/usr/bin/env bash
# Usuários novos nascem com o terminal do Mk (ficha do fastfetch e prompt).
set -euo pipefail
share=${MK_SHARE:-/usr/share/mk}
R=${MK_ROOT:-} # outra raiz no lugar de /: só os testes trocam (migrations/README.md)
skel=$R/etc/skel

for d in fastfetch bash; do
  rm -rf "${skel:?}/.config/mk/$d"
  mkdir -p "$skel/.config/mk"
  cp -r "$share/config/$d" "$skel/.config/mk/$d"
  touch "$skel/.config/mk/$d/.instalado-pelo-mk"
done
if ! grep -q 'mk/bash/mk.bash' "$skel/.bashrc" 2>/dev/null; then
  printf '\n# Mk: ficha do fastfetch, prompt e atalhos\n[[ -f ~/.config/mk/bash/mk.bash ]] && source ~/.config/mk/bash/mk.bash\n' >>"$skel/.bashrc"
fi
