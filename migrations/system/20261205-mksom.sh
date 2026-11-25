#!/usr/bin/env bash
# Mais um app do Mk, pronto em apps/: o MkSom, o equalizador da saída de som
# (dez bandas, com predefinições) e o teste de microfone. Aqui ficam o
# comando, o atalho no menu e o ícone. Não precisa instalar nada: o pipewire
# e o wireplumber já vêm na base.
set -euo pipefail
share=${MK_SHARE:-/usr/share/mk}
R=${MK_ROOT:-} # outra raiz no lugar de /: só os testes trocam (migrations/README.md)

bash "$share/system/mk/apps-links" "$R/"
