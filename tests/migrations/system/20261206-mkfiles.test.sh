#!/usr/bin/env bash
# migrations/system/20261206-mkfiles.sh: o comando, o atalho e o ícone do
# MkFiles, como links para o Mk instalado.
. "$(dirname "$0")/../../lib.sh"
. "$TESTS/migrations/mig.sh" system/20261206-mkfiles.sh
stub update-desktop-database
stub gtk-update-icon-cache

t "não pede pacote: não depende de nada além do que o Mk já traz"
eq "" "$(sed -n 's/^# pacotes: //p' "$MIG")"

t "cria os links do MkFiles"
ok mig
eq /usr/share/mk/apps/mkfiles/bin/mkfiles "$(readlink "$R/usr/bin/mkfiles")"
eq /usr/share/mk/apps/mkfiles/share/mkfiles.desktop "$(readlink "$R/usr/share/applications/mkfiles.desktop")"
eq /usr/share/mk/apps/mkfiles/share/mkfiles.svg "$(readlink "$R/usr/share/icons/hicolor/scalable/apps/mkfiles.svg")"
eq "" "$(cat "$ERR")"

de_novo

t "um link antigo, apontando para outro lugar, é refeito"
ln -sfn /opt/velho "$R/usr/bin/mkfiles"
ok mig
eq /usr/share/mk/apps/mkfiles/bin/mkfiles "$(readlink "$R/usr/bin/mkfiles")"
