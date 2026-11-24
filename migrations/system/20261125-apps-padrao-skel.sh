#!/usr/bin/env bash
# Os apps padrão do Mk no skel, para os usuários criados depois desta
# atualização (os que já existem recebem pela migração de usuário).
set -euo pipefail
share=${MK_SHARE:-/usr/share/mk}
R=${MK_ROOT:-} # outra raiz no lugar de /: só os testes trocam (migrations/README.md)

[[ -e $R/etc/skel/.config/mimeapps.list ]] || install -Dm644 "$share/config/mimeapps.list" "$R/etc/skel/.config/mimeapps.list"
