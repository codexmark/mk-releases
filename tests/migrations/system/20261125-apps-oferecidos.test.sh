#!/usr/bin/env bash
# migrations/system/20261125-apps-oferecidos.sh: anota em
# /var/lib/mk/apps-oferecidos os apps que a máquina já viu, sem instalar nada.
. "$(dirname "$0")/../../lib.sh"
. "$TESTS/migrations/mig.sh" system/20261125-apps-oferecidos.sh

visto=$R/var/lib/mk/apps-oferecidos
lista=$SHARE/system/packages/apps.packages

t "máquina com os apps gráficos (tem nautilus): anota o conjunto antigo do Mk"
instalados nautilus chromium
ok mig
eq "chromium nautilus sushi ffmpegthumbnailer imv evince mpv mpv-mpris ffmpeg imagemagick yt-dlp" "$(tr '\n' ' ' <"$visto" | sed 's/ $//')"

t "os apps que entraram depois (loupe, papers...) ficam de fora, para serem oferecidos"
for p in loupe papers gnome-text-editor libreoffice-fresh; do
  ok grep -qx "$p" <(sed -e 's/#.*//' -e 's/[[:space:]]*$//' "$lista")
  fails grep -qx "$p" "$visto"
done

t "não instala nada"
not_called "pacman -S"

de_novo

t "uma lista que já existe (até vazia) não é reescrita"
: >"$visto"
sem_mudanca mig
echo "so-este" >"$visto"
sem_mudanca mig

t "máquina sem os apps gráficos: anota a lista inteira de hoje (nada a oferecer)"
rm -rf "${R:?}/var"
instalados
ok mig
eq "$(sed -e 's/#.*//' -e 's/[[:space:]]*$//' "$lista" | grep -v '^[[:space:]]*$')" "$(cat "$visto")"
hasnt "$(cat "$visto")" "#"

t "sem os apps gráficos e sem a lista do Mk: cria a anotação vazia e sai com 0"
rm -rf "${R:?}/var" "$lista"
ok mig
exists "$visto"
eq "" "$(cat "$visto")"
