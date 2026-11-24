#!/usr/bin/env bash
# system/mk/mkpulso-links: os links do MkPulso (comando, atalho, ícone) para
# /usr/share/mk. Sempre com uma raiz de mentira: sem argumento ele mexeria no
# /usr desta máquina, então esse modo só é conferido por leitura.
. "$(dirname "$0")/../lib.sh"

mk=$REPO/system/mk/mkpulso-links
raiz=$SBX/raiz
stub update-desktop-database
stub gtk-update-icon-cache
app=/usr/share/mk/apps/mkpulso

t "com uma raiz: cria o comando, o atalho do menu e o ícone dentro dela"
mkdir -p "$raiz"
ok bash "$mk" "$raiz"
eq "$app/bin/mkpulso" "$(readlink "$raiz/usr/bin/mkpulso")"
eq "$app/share/mkpulso.desktop" "$(readlink "$raiz/usr/share/applications/mkpulso.desktop")"
eq "$app/share/mkpulso.svg" "$(readlink "$raiz/usr/share/icons/hicolor/scalable/apps/mkpulso.svg")"

t "são três links e mais nada"
eq 3 "$(find "$raiz" -type l | wc -l)"
eq 0 "$(find "$raiz" -type f | wc -l)"

t "os links apontam para /usr/share/mk do sistema novo, não para dentro da raiz nem do repositório"
while IFS= read -r l; do
  like "$(readlink "$l")" "^$app/"
done < <(find "$raiz" -type l)

t "tudo o que os links apontam existe no repositório (que vira /usr/share/mk)"
while IFS= read -r l; do
  alvo=$(readlink "$l")
  exists "$REPO/apps/mkpulso/${alvo#"$app"/}"
done < <(find "$raiz" -type l)
ok test -x "$REPO/apps/mkpulso/bin/mkpulso"

t "com uma raiz: não atualiza o menu nem os ícones desta máquina"
not_called "update-desktop-database"
not_called "gtk-update-icon-cache"

t "raiz com barra no fim dá no mesmo"
rm -rf "$raiz"
mkdir -p "$raiz"
ok bash "$mk" "$raiz/"
eq "$app/bin/mkpulso" "$(readlink "$raiz/usr/bin/mkpulso")"
eq 3 "$(find "$raiz" -type l | wc -l)"
not_called "update-desktop-database"

t "rodar de novo troca os links no lugar (não cria link dentro de link nem falha)"
ok bash "$mk" "$raiz"
eq 3 "$(find "$raiz" -type l | wc -l)"
eq "$app/bin/mkpulso" "$(readlink "$raiz/usr/bin/mkpulso")"

t "um link velho, para outro lugar, é corrigido"
ln -sfn /opt/antigo/mkpulso "$raiz/usr/bin/mkpulso"
ok bash "$mk" "$raiz"
eq "$app/bin/mkpulso" "$(readlink "$raiz/usr/bin/mkpulso")"

t "as pastas que faltam na raiz são criadas"
rm -rf "$raiz"
mkdir -p "$raiz"
ok bash "$mk" "$raiz"
exists "$raiz/usr/share/icons/hicolor/scalable/apps"

t "sem argumento a raiz é o sistema em uso, e só aí o menu e os ícones são avisados"
src=$(<"$mk")
has "$src" 'root=${1:-/}'
has "$src" 'if [[ -z $root ]]; then'
ok test "$(grep -n 'update-desktop-database' "$mk" | cut -d: -f1)" -gt "$(grep -n 'if \[\[ -z $root \]\]' "$mk" | cut -d: -f1)"

t "o atalho do menu chama o comando mkpulso e usa o ícone mkpulso"
desk=$(<"$REPO/apps/mkpulso/share/mkpulso.desktop")
like "$desk" 'Exec=(/usr/bin/)?mkpulso'
like "$desk" 'Icon=mkpulso'

t "o instalador chama este script com a raiz do sistema novo"
has "$(<"$REPO/installer/mk-install")" 'bash "$src/system/mk/mkpulso-links" "$target"'
