#!/usr/bin/env bash
# migrations/system/20261205-mksom.sh: o comando, o atalho e o ícone do
# MkSom, como links para o Mk instalado.
. "$(dirname "$0")/../../lib.sh"
. "$TESTS/migrations/mig.sh" system/20261205-mksom.sh
stub update-desktop-database
stub gtk-update-icon-cache

t "não pede pacote: o pipewire e o wireplumber já vêm na base"
eq "" "$(sed -n 's/^# pacotes: //p' "$MIG")"
ok grep -qx pipewire "$REPO/system/packages/base.packages"

t "cria os links do MkSom"
ok mig
eq /usr/share/mk/apps/mksom/bin/mksom "$(readlink "$R/usr/bin/mksom")"
eq /usr/share/mk/apps/mksom/share/mksom.desktop "$(readlink "$R/usr/share/applications/mksom.desktop")"
eq /usr/share/mk/apps/mksom/share/mksom.svg "$(readlink "$R/usr/share/icons/hicolor/scalable/apps/mksom.svg")"
eq "" "$(cat "$ERR")"

de_novo

t "um link antigo, apontando para outro lugar, é refeito"
ln -sfn /opt/velho "$R/usr/bin/mksom"
ok mig
eq /usr/share/mk/apps/mksom/bin/mksom "$(readlink "$R/usr/bin/mksom")"
