#!/usr/bin/env bash
# Os apps na paleta Folha: o tema GTK do Mk, os ícones com as pastas em osso
# e a fonte Space Grotesk (a sans do Maré, que faltava nos Mk instalados).
# Quem instala os pacotes é o mk-update, junto com a atualização do sistema.
# pacotes: adw-gtk-theme papirus-icon-theme
set -euo pipefail
share=${MK_SHARE:-/usr/share/mk}
R=${MK_ROOT:-} # outra raiz no lugar de /: só os testes trocam (migrations/README.md)

# (a pasta é do adw-gtk-theme e do gnome-themes-extra: pode não existir ainda)
mkdir -p "$R/usr/share/themes"
rm -rf "$R/usr/share/themes/Mk"
cp -r "$share/system/mk/look/themes/Mk" "$R/usr/share/themes/Mk"
# Os ícones herdam do Papirus: sem o pacote (sem rede, por exemplo) não há o
# que gerar, e o resto segue.
bash "$share/system/mk/look/make-icons" "$R/usr/share/icons/Mk" ||
  echo "aviso: os ícones do Mk não foram gerados (falta o papirus-icon-theme?)" >&2
install -Dm644 -t "$R/usr/share/fonts/mk" "$share"/system/mk/fonts/*.otf
install -Dm644 "$share/system/mk/fonts/OFL.txt" "$R/usr/share/licenses/mk-fonts/OFL.txt"
fc-cache -f >/dev/null 2>&1 || true
if [[ ! -e $R/etc/skel/.config/qt6ct ]]; then
  mkdir -p "$R/etc/skel/.config/qt6ct"
  cp -r "$share/system/mk/look/qt6ct/." "$R/etc/skel/.config/qt6ct/"
fi
