#!/usr/bin/env bash
# migrations/system/20261207-mktunes-go.sh: o MkTunes com o daemon em Go. Pede
# o yt-dlp e o sqlite, refaz os links e não mexe no Java de ninguém.
. "$(dirname "$0")/../../lib.sh"
. "$TESTS/migrations/mig.sh" system/20261207-mktunes-go.sh
stub update-desktop-database
stub gtk-update-icon-cache

app=/usr/share/mk/apps/mktunes

t "declara o que o daemon em Go usa do sistema"
eq "yt-dlp sqlite mpv ffmpeg" "$(sed -n 's/^# pacotes: //p' "$MIG")"

t "os mesmos pacotes estão na lista do desktop, e a JVM saiu dela"
lista=$(<"$REPO/system/packages/desktop.packages")
for p in yt-dlp sqlite mpv ffmpeg; do
  has "$lista" $'\n'"$p"$'\n'
done
eq "" "$(grep -iE 'jre|jdk|java' "$REPO/system/packages/desktop.packages" || true)"

t "refaz os links do MkTunes"
ok mig
eq "$app/bin/mktunes" "$(readlink "$R/usr/bin/mktunes")"
eq "$app/share/mktunes.desktop" "$(readlink "$R/usr/share/applications/mktunes.desktop")"
eq "" "$(cat "$ERR")"

t "o daemon que os links alcançam é o binário em Go, sem a pasta dos jars"
exists "$REPO/apps/mktunes/bin/mktunesd"
ok test ! -e "$REPO/apps/mktunes/daemon"

t "não remove pacote nenhum: o Java pode ser de outro programa"
not_called "pacman"
eq "" "$(grep -nE 'pacman|-Rs?n?\b' "$MIG" | grep -v '^[0-9]*:#' || true)"

de_novo

t "rodar duas vezes dá no mesmo"
ok mig
ok mig
eq "$app/bin/mktunes" "$(readlink "$R/usr/bin/mktunes")"
