#!/usr/bin/env bash
# migrations/system/20261127-mktunes.sh: o comando mktunes, o atalho no menu e
# o ícone, como links para o Mk instalado; os pacotes quem instala é o mk-update.
. "$(dirname "$0")/../../lib.sh"
. "$TESTS/migrations/mig.sh" system/20261127-mktunes.sh

app=/usr/share/mk/apps/mktunes

t "declara o mpv e o ffmpeg no cabeçalho (o Java saiu com o porte para Go)"
eq "mpv ffmpeg" "$(sed -n 's/^# pacotes: //p' "$MIG")"

t "cria os três links para a pasta do Mk"
ok mig
eq "$app/bin/mktunes" "$(readlink "$R/usr/bin/mktunes")"
eq "$app/share/mktunes.desktop" "$(readlink "$R/usr/share/applications/mktunes.desktop")"
eq "$app/share/mktunes.svg" "$(readlink "$R/usr/share/icons/hicolor/scalable/apps/mktunes.svg")"

t "os alvos dos links existem no repositório (apps/mktunes)"
for f in bin/mktunes share/mktunes.desktop share/mktunes.svg; do
  exists "$REPO/apps/mktunes/$f"
done

t "não avisa nada"
eq "" "$(cat "$ERR")"

t "passa ao mktunes-links a raiz (sem MK_ROOT, é /)"
called "mktunes-links $R/"
has "$(cat "$MIG")" 'mktunes-links" "$R/"'

de_novo

t "um link antigo, apontando para outro lugar, é refeito"
ln -sfn /opt/velho/mktunes "$R/usr/bin/mktunes"
ok mig
eq "$app/bin/mktunes" "$(readlink "$R/usr/bin/mktunes")"

t "não fala mais em Java nem pede a JVM"
eq "" "$(grep -iE 'java|jre' "$MIG" || true)"
not_called "pacman -S"
