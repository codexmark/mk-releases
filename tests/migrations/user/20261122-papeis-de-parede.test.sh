#!/usr/bin/env bash
# migrations/user/20261122-papeis-de-parede.sh: ~/Pictures/Wallpapers vai para
# a pasta de imagens do sistema (~/Imagens) e o papel escolhido acompanha.
. "$(dirname "$0")/../../lib.sh"
. "$TESTS/migrations/mig.sh" user/20261122-papeis-de-parede.sh

old=$HOME/Pictures
new=$HOME/Imagens
state=$XDG_STATE_HOME/quickshell/by-shell/abc
stub xdg-user-dir 'echo "$HOME/Imagens"'
antiga() {
  rm -rf "$old" "$new" "$XDG_STATE_HOME/quickshell"
  mkdir -p "$old/Wallpapers"
  echo a >"$old/Wallpapers/folha.webp"
  echo b >"$old/Wallpapers/mar aberto.jpg"
}

t "as imagens vão para ~/Imagens/Wallpapers e a ~/Pictures vazia some"
antiga
mkdir -p "$state"
printf '{"path":"%s/Wallpapers/folha.webp","fit":"cover"}\n' "$old" >"$state/wallpaper.json"
ok mig
eq a "$(cat "$new/Wallpapers/folha.webp")"
eq b "$(cat "$new/Wallpapers/mar aberto.jpg")"
absent "$old"

t "o papel de parede escolhido passa a apontar para a pasta nova"
eq "{\"path\":\"$new/Wallpapers/folha.webp\",\"fit\":\"cover\"}" "$(cat "$state/wallpaper.json")"

de_novo

t "quem nunca escolheu papel de parede (sem o estado do quickshell): move e sai com 0"
antiga
ok mig
eq a "$(cat "$new/Wallpapers/folha.webp")"
absent "$old"

t "a ~/Pictures com outras coisas fica; só a Wallpapers sai"
antiga
echo foto >"$old/ferias.jpg"
ok mig
eq foto "$(cat "$old/ferias.jpg")"
absent "$old/Wallpapers"
exists "$new/Wallpapers/folha.webp"

t "uma imagem de mesmo nome que já estava na pasta nova não é sobrescrita"
antiga
mkdir -p "$new/Wallpapers"
echo minha >"$new/Wallpapers/folha.webp"
ok mig
eq minha "$(cat "$new/Wallpapers/folha.webp")"
eq a "$(cat "$old/Wallpapers/folha.webp")"
eq b "$(cat "$new/Wallpapers/mar aberto.jpg")"

t "sistema em inglês (a pasta de imagens é a própria ~/Pictures): nada a fazer"
antiga
stub xdg-user-dir 'echo "$HOME/Pictures"'
sem_mudanca mig

t "pasta de imagens apontando para a home (xdg-user-dirs sem configurar): nada a fazer"
stub xdg-user-dir 'echo "$HOME"'
sem_mudanca mig

t "sem o xdg-user-dir: nada a fazer, sai com 0"
rm -f "$SBX/bin/xdg-user-dir"
sem_mudanca mig

t "sem ~/Pictures/Wallpapers: nada a fazer"
stub xdg-user-dir 'echo "$HOME/Imagens"'
rm -rf "$old"
sem_mudanca mig

t "não sai da home"
eq "" "$(find "$R" -mindepth 1)"
