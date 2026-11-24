#!/usr/bin/env bash
# migrations/system/20261121-logo-plano.sh: o logo novo no plymouth, no sddm e
# na ficha do skel, e o initramfs refeito.
. "$(dirname "$0")/../../lib.sh"
. "$TESTS/migrations/mig.sh" system/20261121-logo-plano.sh

stub mkinitcpio
ply=$R/usr/share/plymouth/themes/mk
sddm=$R/usr/share/sddm/themes/mk
ficha=$R/etc/skel/.config/mk/fastfetch
mkdir -p "$ply" "$sddm" "$ficha"
echo antigo >"$ply/logo.png"
echo antigo >"$ply/sobra.png"
echo antigo >"$sddm/mark.png"
echo antigo >"$ficha/mark.png"
echo antigo >"$ficha/logo-antigo.txt"

t "os temas do plymouth e do sddm são trocados inteiros pelos novos"
ok mig
eq "" "$(diff -r "$SHARE/system/mk/plymouth/mk" "$ply" 2>&1)"
eq "" "$(diff -r "$SHARE/system/mk/sddm/mk" "$sddm" 2>&1)"

t "a ficha do skel é trocada e marcada como do Mk"
eq "" "$(diff -r -x .instalado-pelo-mk "$SHARE/config/fastfetch" "$ficha" 2>&1)"
exists "$ficha/.instalado-pelo-mk"

t "o initramfs é refeito (o logo do boot vai dentro dele)"
called "mkinitcpio -P"

de_novo

t "skel sem a ficha do fastfetch: não cria uma"
rm -rf "${R:?}"
mkdir -p "$R/usr/share/plymouth/themes" "$R/usr/share/sddm/themes" # as pastas dos pacotes
ok mig
absent "$R/etc/skel/.config/mk/fastfetch"
exists "$ply/logo.png"
exists "$sddm/Main.qml"

t "não instala nada"
not_called "pacman"
