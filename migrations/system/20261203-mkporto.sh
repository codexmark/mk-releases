#!/usr/bin/env bash
# Mais um app do Mk, pronto em apps/: o MkPorto, o catálogo das conexões SSH
# (os hosts do ~/.ssh/config, se respondem, conectar num terminal, as chaves
# públicas). Aqui ficam o comando, o atalho no menu e o ícone. Não precisa
# instalar nada: o openssh já vem na base.
set -euo pipefail
share=${MK_SHARE:-/usr/share/mk}
R=${MK_ROOT:-} # outra raiz no lugar de /: só os testes trocam (migrations/README.md)

bash "$share/system/mk/apps-links" "$R/"
