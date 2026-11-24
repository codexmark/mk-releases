#!/usr/bin/env bash
# system/mk/apps-links: os links dos apps do Mk que vieram depois do MkPulso
# (MkPacotes, MkInstalar, MkComprimir): comando, atalho no menu e ícone, todos
# para /usr/share/mk. Sempre com uma raiz de mentira.
. "$(dirname "$0")/../lib.sh"

mk=$REPO/system/mk/apps-links
raiz=$SBX/raiz
stub update-desktop-database
stub gtk-update-icon-cache
mkdir -p "$raiz"

t "cria comando, atalho e ícone de cada app dentro da raiz"
ok bash "$mk" "$raiz"
for par in mkpacotes:mkpacotes mkutil:mkinstalar mkutil:mkcomprimir; do
  pasta=${par%%:*} nome=${par#*:}
  eq "/usr/share/mk/apps/$pasta/bin/$nome" "$(readlink "$raiz/usr/bin/$nome")"
  eq "/usr/share/mk/apps/$pasta/share/$nome.desktop" "$(readlink "$raiz/usr/share/applications/$nome.desktop")"
  eq "/usr/share/mk/apps/$pasta/share/$nome.svg" "$(readlink "$raiz/usr/share/icons/hicolor/scalable/apps/$nome.svg")"
done

t "são nove links e mais nada"
eq 9 "$(find "$raiz" -type l | wc -l)"
eq 0 "$(find "$raiz" -type f | wc -l)"

t "tudo o que os links apontam existe no repositório, e os comandos são executáveis"
while IFS= read -r l; do
  alvo=$(readlink "$l")
  exists "$REPO/${alvo#/usr/share/mk/}"
done < <(find "$raiz" -type l)
for f in apps/mkpacotes/bin/mkpacotes apps/mkpacotes/bin/mkpacotes-coletor apps/mkutil/bin/mkinstalar apps/mkutil/bin/mkcomprimir apps/mkutil/bin/mkutil-motor; do
  ok test -x "$REPO/$f"
done

t "rodar de novo dá no mesmo, e um link velho é refeito"
ln -sfn /opt/velho "$raiz/usr/bin/mkpacotes"
ok bash "$mk" "$raiz"
eq "/usr/share/mk/apps/mkpacotes/bin/mkpacotes" "$(readlink "$raiz/usr/bin/mkpacotes")"
eq 9 "$(find "$raiz" -type l | wc -l)"

t "com uma raiz, não mexe nos caches de menu e de ícones desta máquina"
not_called "update-desktop-database"
not_called "gtk-update-icon-cache"

t "o instalador liga os apps e a migração também"
has "$(<"$REPO/installer/mk-install")" 'system/mk/apps-links" "$target"'
has "$(<"$REPO/migrations/system/20261130-mkpacotes-mkutil.sh")" 'apps-links" "$R/"'

t "as janelas dos apps flutuam no centro (regra do Hyprland)"
regras=$(<"$REPO/config/hypr/mare/rules.lua")
has "$regras" '^(MkPulso|MkPacotes|MkInstalar|MkComprimir)$'
