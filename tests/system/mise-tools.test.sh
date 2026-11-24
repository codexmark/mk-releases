#!/usr/bin/env bash
# scripts/user/mise-tools.sh: as ferramentas por usuário, com o mise por stub
# (nada é baixado nem instalado).
. "$(dirname "$0")/../lib.sh"

mt=$REPO/scripts/user/mise-tools.sh
stub mise

t "instala node (LTS) e python no mise global do usuário"
out=$(bash "$mt" 2>&1)
eq 0 $? "$out"
called "mise use -g node@lts"
called "mise use -g python@latest"

t "instala as CLIs de IA do README pelo npm do mise"
called "mise use -g npm:@anthropic-ai/claude-code"
called "mise use -g npm:@openai/codex"
called "mise use -g npm:@google/gemini-cli"

t "o node vem antes das CLIs que dependem do npm"
eq "mise use -g node@lts" "$(head -1 "$SBX/calls")"

t "tudo é global do usuário (-g) e nada usa sudo nem pacman"
eq "" "$(grep -v '^mise use -g ' "$SBX/calls")"
eq 5 "$(wc -l <"$SBX/calls")"

t "cria a pasta ~/Work"
exists "$HOME/Work"

t "pode rodar de novo com a pasta já lá"
echo guardado >"$HOME/Work/arquivo"
ok bash "$mt"
eq guardado "$(<"$HOME/Work/arquivo")"

t "se o mise falha, para (não finge que instalou)"
rm -rf "$HOME/Work"
calls_reset
stub mise '[[ "$*" == "use -g python@latest" ]] && exit 1; exit 0'
fails bash "$mt"
eq 2 "$(wc -l <"$SBX/calls")"
absent "$HOME/Work"

t "sem o mise instalado, falha em vez de seguir em silêncio"
mkdir -p "$SBX/minimo"
for c in bash env mkdir; do ln -s "$(command -v "$c")" "$SBX/minimo/$c"; done
out=$(PATH=$SBX/minimo "$SBX/minimo/bash" "$mt" 2>&1)
fails test $? -eq 0
absent "$HOME/Work"
