#!/usr/bin/env bash
# migrations/system/20261203-mkporto.sh: o comando, o atalho e o ícone do
# MkPorto, como links para o Mk instalado.
. "$(dirname "$0")/../../lib.sh"
. "$TESTS/migrations/mig.sh" system/20261203-mkporto.sh
stub update-desktop-database
stub gtk-update-icon-cache

t "não pede pacote: o openssh já vem na base"
eq "" "$(sed -n 's/^# pacotes: //p' "$MIG")"
ok grep -qx openssh "$REPO/system/packages/base.packages"

t "cria os links do MkPorto"
ok mig
eq /usr/share/mk/apps/mkporto/bin/mkporto "$(readlink "$R/usr/bin/mkporto")"
eq /usr/share/mk/apps/mkporto/share/mkporto.desktop "$(readlink "$R/usr/share/applications/mkporto.desktop")"
eq /usr/share/mk/apps/mkporto/share/mkporto.svg "$(readlink "$R/usr/share/icons/hicolor/scalable/apps/mkporto.svg")"
eq "" "$(cat "$ERR")"

de_novo

t "um link antigo, apontando para outro lugar, é refeito"
ln -sfn /opt/velho "$R/usr/bin/mkporto"
ok mig
eq /usr/share/mk/apps/mkporto/bin/mkporto "$(readlink "$R/usr/bin/mkporto")"
