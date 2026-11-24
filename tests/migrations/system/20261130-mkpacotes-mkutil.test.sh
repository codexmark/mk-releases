#!/usr/bin/env bash
# migrations/system/20261130-mkpacotes-mkutil.sh: os comandos, atalhos e ícones
# do MkPacotes, do MkInstalar e do MkComprimir, como links para o Mk instalado.
. "$(dirname "$0")/../../lib.sh"
. "$TESTS/migrations/mig.sh" system/20261130-mkpacotes-mkutil.sh
stub update-desktop-database
stub gtk-update-icon-cache

t "não pede pacote nenhum"
eq "" "$(sed -n 's/^# pacotes: //p' "$MIG")"

t "cria os links dos três apps"
ok mig
eq /usr/share/mk/apps/mkpacotes/bin/mkpacotes "$(readlink "$R/usr/bin/mkpacotes")"
eq /usr/share/mk/apps/mkutil/bin/mkinstalar "$(readlink "$R/usr/bin/mkinstalar")"
eq /usr/share/mk/apps/mkutil/bin/mkcomprimir "$(readlink "$R/usr/bin/mkcomprimir")"
eq /usr/share/mk/apps/mkutil/share/mkcomprimir.desktop "$(readlink "$R/usr/share/applications/mkcomprimir.desktop")"
eq 18 "$(find "$R" -type l | wc -l)" # o apps-links de hoje liga também os apps que vieram depois
eq "" "$(cat "$ERR")"

de_novo

t "um link antigo, apontando para outro lugar, é refeito"
ln -sfn /opt/velho "$R/usr/bin/mkinstalar"
ok mig
eq /usr/share/mk/apps/mkutil/bin/mkinstalar "$(readlink "$R/usr/bin/mkinstalar")"
