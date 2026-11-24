#!/usr/bin/env bash
# shell/mare/scripts/mare-look: cada chave do gsettings só é escrita se está
# no padrão de fábrica ou se foi o Mk que escreveu o valor atual. gsettings e
# dconf são stubs com o estado em $SBX.
. "$(dirname "$0")/../lib.sh"

look_sh=$REPO/shell/mare/scripts/mare-look
state=$XDG_CONFIG_HOME/mk/.look
gs=$SBX/gs dc=$SBX/dconf

# gsettings: uma chave por arquivo em $SBX/gs (get falha se a chave não
# existe; set falha com gs-trancado). dconf read: o que o usuário (ou alguém)
# escreveu por cima do padrão; vazio = ainda de fábrica.
stub gsettings '
[[ $2 == org.gnome.desktop.interface ]] || exit 1
case $1 in
  get) cat "$SBX/gs/$3" 2>/dev/null || exit 1 ;;
  set) [[ ! -e $SBX/gs-trancado ]] || exit 1; printf "%s\n" "$4" >"$SBX/gs/$3"; printf "%s\n" "$4" >"$SBX/dconf/$3" ;;
esac'
stub dconf '
[[ $1 == read && $2 == /org/gnome/desktop/interface/* ]] || exit 1
cat "$SBX/dconf/${2##*/}" 2>/dev/null; exit 0'

run() { bash "$look_sh"; }
# De fábrica: o gsettings responde o padrão do GNOME e o dconf não tem nada.
fabrica() {
  rm -rf "${gs:?}" "${dc:?}" "$SBX/gs-trancado" "${XDG_CONFIG_HOME:?}/mk"
  mkdir -p "$gs" "$dc" "$XDG_CONFIG_HOME/mk"
  echo "'default'" >"$gs/color-scheme"
  echo "'Adwaita'" >"$gs/gtk-theme"
  echo "'Adwaita'" >"$gs/icon-theme"
  echo "'Adwaita'" >"$gs/cursor-theme"
  echo "24" >"$gs/cursor-size"
  echo "'Cantarell 11'" >"$gs/font-name"
  echo "'Cantarell 11'" >"$gs/document-font-name"
  echo "'Source Code Pro 10'" >"$gs/monospace-font-name"
  calls_reset
}
# O usuário (ou outro desktop) escolheu um valor.
escolheu() {
  echo "$2" >"$gs/$1"
  echo "$2" >"$dc/$1"
}
escritas() { grep -c '^gsettings set ' "$SBX/calls"; }
tudo() { (cd "$gs" && grep -H . ./* | sed 's|^\./||' | sort | tr '\n' ' '); }
quer="color-scheme:'prefer-dark' cursor-size:24 cursor-theme:'Adwaita' document-font-name:'Space Grotesk 10' font-name:'Space Grotesk 10' gtk-theme:'adw-gtk3-dark' icon-theme:'Mk' monospace-font-name:'SpaceMono Nerd Font 10' "

t "sem a pasta ~/.config/mk (não dá para anotar): sai com 0 sem tocar no gsettings"
fabrica
rm -rf "${XDG_CONFIG_HOME:?}/mk"
ok run
eq "" "$(<"$SBX/calls")"

t "conta de fábrica: as oito chaves ficam com a cara do Mk"
fabrica
ok run
eq "$quer" "$(tudo)"

t "conta de fábrica: só escreve o que mudou (cursor-theme e cursor-size já estavam certos)"
eq 6 "$(escritas)"
not_called "gsettings set org.gnome.desktop.interface cursor-theme"
not_called "gsettings set org.gnome.desktop.interface cursor-size"
called "gsettings set org.gnome.desktop.interface icon-theme 'Mk'"
called "gsettings set org.gnome.desktop.interface monospace-font-name 'SpaceMono Nerd Font 10'"

t "o que o Mk deixou fica anotado em .look, uma linha por chave"
eq "color-scheme='prefer-dark'
cursor-size=24
cursor-theme='Adwaita'
document-font-name='Space Grotesk 10'
font-name='Space Grotesk 10'
gtk-theme='adw-gtk3-dark'
icon-theme='Mk'
monospace-font-name='SpaceMono Nerd Font 10'" "$(sort "$state")"
absent "$state.tmp"

t "rodar de novo não escreve nada nem duplica as anotações"
calls_reset
ok run
eq 0 "$(escritas)"
eq 8 "$(wc -l <"$state")"
eq "$quer" "$(tudo)"

t "a escolha de outro desktop (ícones, fonte) não é desfeita; o resto ganha a cara do Mk"
fabrica
escolheu icon-theme "'Papirus-Dark'"
escolheu font-name "'Inter 11'"
ok run
eq "'Papirus-Dark'" "$(<"$gs/icon-theme")"
eq "'Inter 11'" "$(<"$gs/font-name")"
eq "'adw-gtk3-dark'" "$(<"$gs/gtk-theme")"
eq "'Space Grotesk 10'" "$(<"$gs/document-font-name")"
not_called "gsettings set org.gnome.desktop.interface icon-theme"
not_called "gsettings set org.gnome.desktop.interface font-name"

t "a escolha do usuário não é anotada como se fosse do Mk"
eq "" "$(grep -E '^(icon-theme|font-name)=' "$state")"

t "o que o Mk escreveu numa versão anterior é atualizado para o valor novo"
fabrica
escolheu icon-theme "'Mk-antigo'"
escolheu font-name "'Outra Fonte=10'"
printf '%s\n' "icon-theme='Mk-antigo'" "font-name='Outra Fonte=10'" >"$state"
ok run
eq "'Mk'" "$(<"$gs/icon-theme")"
eq "'Space Grotesk 10'" "$(<"$gs/font-name")"
eq "icon-theme='Mk'" "$(grep '^icon-theme=' "$state")"
eq 1 "$(grep -c '^font-name=' "$state")"

t "se o usuário trocou depois do Mk, a troca dele fica"
fabrica
ok run
escolheu icon-theme "'Papirus'"
escolheu cursor-size "32"
calls_reset
ok run
eq "'Papirus'" "$(<"$gs/icon-theme")"
eq "32" "$(<"$gs/cursor-size")"
eq 0 "$(escritas)"

t "a anotação de font-name não vale para document-font-name nem monospace-font-name"
fabrica
escolheu document-font-name "'Do Usuario 12'"
escolheu monospace-font-name "'Do Usuario 12'"
printf '%s\n' "font-name='Do Usuario 12'" >"$state"
ok run
eq "'Do Usuario 12'" "$(<"$gs/document-font-name")"
eq "'Do Usuario 12'" "$(<"$gs/monospace-font-name")"

t "o Adwaita-dark que o Mk antigo punha sem anotar é trocado pelo tema de hoje"
fabrica
escolheu gtk-theme "'Adwaita-dark'"
ok run
eq "'adw-gtk3-dark'" "$(<"$gs/gtk-theme")"
eq "gtk-theme='adw-gtk3-dark'" "$(grep '^gtk-theme=' "$state")"

t "outro tema GTK escolhido pelo usuário fica"
fabrica
escolheu gtk-theme "'Breeze-Dark'"
ok run
eq "'Breeze-Dark'" "$(<"$gs/gtk-theme")"

t "valor que já é o do Mk, mesmo posto por outro: só anota, sem escrever"
fabrica
escolheu icon-theme "'Mk'"
ok run
not_called "gsettings set org.gnome.desktop.interface icon-theme"
eq "icon-theme='Mk'" "$(grep '^icon-theme=' "$state")"

t "sem o esquema do GNOME (gsettings get falha): não escreve nada e sai com 0"
fabrica
rm -f "$gs"/*
ok run
eq 0 "$(escritas)"
eq "" "$(<"$state")"

t "se o gsettings recusa a escrita, a chave não é anotada como do Mk"
fabrica
: >"$SBX/gs-trancado"
run
eq "'Adwaita'" "$(<"$gs/icon-theme")"
eq "" "$(grep '^icon-theme=' "$state")"
eq "cursor-size=24" "$(grep '^cursor-size=' "$state")"

t "só mexe em org.gnome.desktop.interface"
fabrica
run
eq "" "$(grep '^gsettings' "$SBX/calls" | grep -v ' org.gnome.desktop.interface ')"
eq "" "$(grep '^dconf' "$SBX/calls" | grep -v '^dconf read /org/gnome/desktop/interface/')"
