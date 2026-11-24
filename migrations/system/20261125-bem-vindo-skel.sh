#!/usr/bin/env bash
# As boas-vindas para os usuários criados depois desta atualização: no skel, a
# migração de usuário delas já consta como feita, como numa instalação nova.
# Sem a marca, o Maré não mostra a folha sozinho (shell/mare/Welcome.qml) e o
# primeiro mk-update do usuário novo a daria por vista. Só esta marca: as
# outras migrações de usuário ainda têm o que fazer numa home saída deste skel.
set -euo pipefail
R=${MK_ROOT:-} # outra raiz no lugar de /: só os testes trocam (migrations/README.md)
state=$R/etc/skel/.local/state/mk/migrations

mkdir -p "$state"
touch "$state/20261123-bem-vindo.sh"
