#!/usr/bin/env bash
# migrations/user/20261121-bash-mk.sh: cria o ~/.config/mk/seu.bash, se ainda
# não existir.
. "$(dirname "$0")/../../lib.sh"
. "$TESTS/migrations/mig.sh" user/20261121-bash-mk.sh

seu=$XDG_CONFIG_HOME/mk/seu.bash

t "home sem seu.bash: recebe o exemplo do Mk"
ok mig
ok cmp -s "$SHARE/config/bash/seu.bash.exemplo" "$seu"

t "o seu.bash novo é só comentário: carregá-lo não muda nada"
eq "" "$(grep -v '^#' "$seu" | grep -v '^[[:space:]]*$' || true)"

de_novo

t "um seu.bash que já existe (até vazio) nunca é trocado"
echo "alias k=kubectl" >"$seu"
sem_mudanca mig
: >"$seu"
sem_mudanca mig

t "não sai da home nem chama comando do sistema"
eq "" "$(find "$R" -mindepth 1)"
eq "" "$(cat "$SBX/calls")"
