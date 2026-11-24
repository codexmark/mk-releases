#!/usr/bin/env bash
# migrations/system/20261129-mkpulso.sh: o comando mkpulso, o atalho no menu e
# o ícone, como links para o Mk instalado.
. "$(dirname "$0")/../../lib.sh"
. "$TESTS/migrations/mig.sh" system/20261129-mkpulso.sh

app=/usr/share/mk/apps/mkpulso
mkdir -p "$SHARE/apps/mkpulso/bin"
printf '#!/bin/sh\n' >"$SHARE/apps/mkpulso/bin/mkpulso-coletor"
chmod +x "$SHARE/apps/mkpulso/bin/mkpulso-coletor"
stub update-desktop-database
stub gtk-update-icon-cache

t "não pede pacote nenhum"
eq "" "$(sed -n 's/^# pacotes: //p' "$MIG")"

t "cria os três links para a pasta do Mk"
ok mig
eq "$app/bin/mkpulso" "$(readlink "$R/usr/bin/mkpulso")"
eq "$app/share/mkpulso.desktop" "$(readlink "$R/usr/share/applications/mkpulso.desktop")"
eq "$app/share/mkpulso.svg" "$(readlink "$R/usr/share/icons/hicolor/scalable/apps/mkpulso.svg")"
eq "" "$(cat "$ERR")"

t "os alvos dos links e o coletor existem no repositório (apps/mkpulso)"
for f in bin/mkpulso bin/mkpulso-coletor share/mkpulso.desktop share/mkpulso.svg qml/shell.qml; do
  exists "$REPO/apps/mkpulso/$f"
done
ok test -x "$REPO/apps/mkpulso/bin/mkpulso-coletor"

de_novo

t "um link antigo, apontando para outro lugar, é refeito"
ln -sfn /opt/velho/mkpulso "$R/usr/bin/mkpulso"
ok mig
eq "$app/bin/mkpulso" "$(readlink "$R/usr/bin/mkpulso")"

t "sem o coletor no lançamento: avisa, cria os links e sai com 0"
rm -f "$SHARE/apps/mkpulso/bin/mkpulso-coletor"
rm -rf "${R:?}/usr"
ok mig
has "$(cat "$ERR")" "aviso: o coletor do MkPulso não veio"
ok test -L "$R/usr/share/applications/mkpulso.desktop"
