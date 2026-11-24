#!/usr/bin/env bash
# migrations/system/20261120-terminal-skel.sh: o skel ganha a ficha do
# fastfetch, o bash do Mk e a linha do ~/.bashrc.
. "$(dirname "$0")/../../lib.sh"
. "$TESTS/migrations/mig.sh" system/20261120-terminal-skel.sh

skel=$R/etc/skel
mkdir -p "$skel"
printf '# ~/.bashrc do Arch\nalias ls="ls --color=auto"\n' >"$skel/.bashrc"

t "skel sem o terminal do Mk: copia fastfetch e bash, marcados como do Mk"
ok mig
for d in fastfetch bash; do
  exists "$skel/.config/mk/$d/.instalado-pelo-mk"
  eq "" "$(diff -r -x .instalado-pelo-mk "$SHARE/config/$d" "$skel/.config/mk/$d" 2>&1)" "$d"
done

t "o .bashrc do skel ganha a linha que carrega o mk.bash, sem perder o que tinha"
bashrc=$(cat "$skel/.bashrc")
has "$bashrc" 'alias ls="ls --color=auto"'
has "$bashrc" '[[ -f ~/.config/mk/bash/mk.bash ]] && source ~/.config/mk/bash/mk.bash'

de_novo

t "rodar de novo não repete a linha no .bashrc"
eq 1 "$(grep -c 'source ~/.config/mk/bash/mk.bash' "$skel/.bashrc")"

t "uma pasta antiga do skel é trocada inteira"
echo velho >"$skel/.config/mk/bash/sobra"
ok mig
absent "$skel/.config/mk/bash/sobra"

t "skel sem .bashrc: cria com a linha"
rm -rf "${skel:?}"
ok mig
has "$(cat "$skel/.bashrc")" 'source ~/.config/mk/bash/mk.bash'

t "não chama comando nenhum do sistema"
eq "" "$(cat "$SBX/calls")"
