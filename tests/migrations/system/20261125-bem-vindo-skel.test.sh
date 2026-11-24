#!/usr/bin/env bash
# migrations/system/20261125-bem-vindo-skel.sh: no skel, a migração de usuário
# das boas-vindas já consta como feita (e só ela).
. "$(dirname "$0")/../../lib.sh"
. "$TESTS/migrations/mig.sh" system/20261125-bem-vindo-skel.sh

state=$R/etc/skel/.local/state/mk/migrations

t "skel sem marcas: ganha a marca das boas-vindas"
ok mig
exists "$state/20261123-bem-vindo.sh"

t "a marca tem o nome de uma migração de usuário que existe"
exists "$REPO/migrations/user/20261123-bem-vindo.sh"

t "é o nome que o Maré espera (shell/mare/Welcome.qml)"
has "$(cat "$REPO/shell/mare/Welcome.qml")" '20261123-bem-vindo.sh'

t "só essa marca: as outras migrações de usuário ainda rodam numa home nova"
eq "20261123-bem-vindo.sh" "$(ls -A "$state")"

de_novo

t "as marcas que o instalador já tinha posto no skel ficam"
touch "$state/20261120-terminal.sh"
ok mig
exists "$state/20261120-terminal.sh"
exists "$state/20261123-bem-vindo.sh"

t "não chama comando nenhum do sistema"
eq "" "$(cat "$SBX/calls")"
