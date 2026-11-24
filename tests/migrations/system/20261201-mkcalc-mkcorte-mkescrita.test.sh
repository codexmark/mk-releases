#!/usr/bin/env bash
# migrations/system/20261201-mkcalc-mkcorte-mkescrita.sh: os comandos, atalhos
# e ícones do MkCalc, do MkCorte e do MkEscrita, como links para o Mk instalado.
. "$(dirname "$0")/../../lib.sh"
. "$TESTS/migrations/mig.sh" system/20261201-mkcalc-mkcorte-mkescrita.sh
stub update-desktop-database
stub gtk-update-icon-cache

t "pede o qt6-multimedia, que o MkCorte usa para mostrar o vídeo"
eq "qt6-multimedia qt6-multimedia-ffmpeg" "$(sed -n 's/^# pacotes: //p' "$MIG")"

t "cria os links dos três apps"
ok mig
for nome in mkcalc mkcorte mkescrita; do
  eq "/usr/share/mk/apps/$nome/bin/$nome" "$(readlink "$R/usr/bin/$nome")"
  eq "/usr/share/mk/apps/$nome/share/$nome.desktop" "$(readlink "$R/usr/share/applications/$nome.desktop")"
  eq "/usr/share/mk/apps/$nome/share/$nome.svg" "$(readlink "$R/usr/share/icons/hicolor/scalable/apps/$nome.svg")"
done
eq "" "$(cat "$ERR")"

de_novo

t "um link antigo, apontando para outro lugar, é refeito"
ln -sfn /opt/velho "$R/usr/bin/mkcorte"
ok mig
eq /usr/share/mk/apps/mkcorte/bin/mkcorte "$(readlink "$R/usr/bin/mkcorte")"
