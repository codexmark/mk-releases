#!/usr/bin/env bash
# migrations/user/20261120-terminal.sh: a ficha do fastfetch, o bash do Mk e a
# linha do ~/.bashrc, sem tocar no que não é do Mk.
. "$(dirname "$0")/../../lib.sh"
. "$TESTS/migrations/mig.sh" user/20261120-terminal.sh

cfg=$XDG_CONFIG_HOME/mk
printf '# meu bashrc\nexport EDITOR=nvim\n' >"$HOME/.bashrc"

t "home sem o terminal do Mk: copia fastfetch e bash, marcados como do Mk"
ok mig
for d in fastfetch bash; do
  exists "$cfg/$d/.instalado-pelo-mk"
  eq "" "$(diff -r -x .instalado-pelo-mk "$SHARE/config/$d" "$cfg/$d" 2>&1)" "$d"
done

t "o ~/.bashrc ganha a linha que carrega o mk.bash, sem perder o que tinha"
bashrc=$(cat "$HOME/.bashrc")
has "$bashrc" "export EDITOR=nvim"
has "$bashrc" '[[ -f ~/.config/mk/bash/mk.bash ]] && source ~/.config/mk/bash/mk.bash'

de_novo

t "rodar de novo não repete a linha no ~/.bashrc"
eq 1 "$(grep -c 'source ~/.config/mk/bash/mk.bash' "$HOME/.bashrc")"

t "uma pasta do Mk (com a marca) é trocada inteira"
echo velho >"$cfg/fastfetch/sobra"
ok mig
absent "$cfg/fastfetch/sobra"

t "uma pasta que não é do Mk (sem a marca) é mantida, com aviso, e a outra entra"
rm -rf "$cfg/fastfetch" "$cfg/bash"
mkdir -p "$cfg/fastfetch"
echo '{"meu": true}' >"$cfg/fastfetch/config.jsonc"
ok mig
eq '{"meu": true}' "$(cat "$cfg/fastfetch/config.jsonc")"
absent "$cfg/fastfetch/.instalado-pelo-mk"
has "$(cat "$OUT")" "mantido (não é do Mk): $cfg/fastfetch"
exists "$cfg/bash/mk.bash"

t "sem ~/.bashrc: cria com a linha"
rm -f "$HOME/.bashrc"
ok mig
has "$(cat "$HOME/.bashrc")" 'source ~/.config/mk/bash/mk.bash'

t "um ~/.bashrc que já carrega o mk.bash de outro jeito não ganha outra linha"
echo '. ~/.config/mk/bash/mk.bash' >"$HOME/.bashrc"
sem_mudanca mig

t "não sai da home nem chama comando do sistema"
eq "" "$(find "$R" -mindepth 1)"
eq "" "$(cat "$SBX/calls")"
