#!/usr/bin/env bash
# Mais um app do Mk, pronto em apps/: o MkFiles, o gerenciador de arquivos
# (abas, painel duplo, lixeira, busca, miniaturas). Aqui ficam o comando, o
# atalho no menu e o ícone. Entra ao lado do Nautilus, que continua sendo o
# padrão para abrir pastas. Não precisa instalar nada.
set -euo pipefail
share=${MK_SHARE:-/usr/share/mk}
R=${MK_ROOT:-} # outra raiz no lugar de /: só os testes trocam (migrations/README.md)

bash "$share/system/mk/apps-links" "$R/"
