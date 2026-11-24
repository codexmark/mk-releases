#!/usr/bin/env bash
# migrations/user/20261123-bem-vindo.sh: quem já usava o Mk tem as boas-vindas
# marcadas como vistas.
. "$(dirname "$0")/../../lib.sh"
. "$TESTS/migrations/mig.sh" user/20261123-bem-vindo.sh

marca=$XDG_CONFIG_HOME/mk/.bem-vindo

t "home sem a marca: as boas-vindas ficam como vistas"
ok mig
exists "$marca"

t "é a marca que o Maré lê (shell/mare/Welcome.qml)"
has "$(cat "$REPO/shell/mare/Welcome.qml")" ".bem-vindo"

de_novo

t "uma marca que já existe (quem já fez o passeio) não é reescrita"
echo "visto em 2026-11-23" >"$marca"
sem_mudanca mig
eq "visto em 2026-11-23" "$(cat "$marca")"

t "sem ~/.config/mk: cria a pasta e a marca"
rm -rf "$XDG_CONFIG_HOME"
ok mig
exists "$marca"

t "não sai da home nem chama comando do sistema"
eq "" "$(find "$R" -mindepth 1)"
eq "" "$(cat "$SBX/calls")"
