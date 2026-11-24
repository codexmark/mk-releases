#!/usr/bin/env bash
# migrations/system/20261121-apps-folha.sh: tema GTK, ícones, fontes e o qt6ct
# do skel; os pacotes ficam declarados, quem instala é o mk-update.
. "$(dirname "$0")/../../lib.sh"
. "$TESTS/migrations/mig.sh" system/20261121-apps-folha.sh

stub fc-cache

t "declara os pacotes no cabeçalho e não instala nenhum"
eq "adw-gtk-theme papirus-icon-theme" "$(sed -n 's/^# pacotes: //p' "$MIG")"
instalados adw-gtk-theme papirus-icon-theme
ok mig
not_called "pacman -S"

t "o tema GTK, os ícones, as fontes e a licença ficam no lugar"
eq "" "$(diff -r "$SHARE/system/mk/look/themes/Mk" "$R/usr/share/themes/Mk" 2>&1)"
called "make-icons $R/usr/share/icons/Mk"
exists "$R/usr/share/icons/Mk/index.theme"
for f in "$SHARE"/system/mk/fonts/*.otf; do
  ok cmp -s "$f" "$R/usr/share/fonts/mk/$(basename "$f")"
done
ok cmp -s "$SHARE/system/mk/fonts/OFL.txt" "$R/usr/share/licenses/mk-fonts/OFL.txt"
called "fc-cache -f"

t "o skel ganha o qt6ct do Mk"
eq "" "$(diff -r "$SHARE/system/mk/look/qt6ct" "$R/etc/skel/.config/qt6ct" 2>&1)"

de_novo

t "um qt6ct que já estava no skel não é trocado"
rm -rf "$R/etc/skel/.config/qt6ct"
mkdir -p "$R/etc/skel/.config/qt6ct"
echo "meu" >"$R/etc/skel/.config/qt6ct/qt6ct.conf"
ok mig
eq "meu" "$(cat "$R/etc/skel/.config/qt6ct/qt6ct.conf")"
absent "$R/etc/skel/.config/qt6ct/colors"

t "um tema antigo é trocado inteiro"
echo velho >"$R/usr/share/themes/Mk/sobra.css"
ok mig
absent "$R/usr/share/themes/Mk/sobra.css"

t "sem o papirus (pacote não veio): avisa, faz o resto e sai com 0"
rm -rf "${R:?}"
instalados
touch "$SBX/sem-papirus"
ok mig
has "$(cat "$ERR")" "aviso"
exists "$R/usr/share/themes/Mk/index.theme"
exists "$R/usr/share/fonts/mk/SpaceGrotesk-Regular.otf"
not_called "pacman -S"

t "sem o fc-cache no sistema, não quebra"
rm -f "$SBX/bin/fc-cache" "$SBX/sem-papirus"
ok mig
