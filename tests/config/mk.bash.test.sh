#!/usr/bin/env bash
# config/bash/mk.bash: o comando mk, os atalhos e o que carrega (prompt,
# fastfetch, ble.sh), num bash não interativo com tudo trocado por dublês.
# (O bash interativo de verdade é conferido pelo bin/check-shell.)
. "$(dirname "$0")/../lib.sh"

src=$REPO/config/bash/mk.bash
cfg=$XDG_CONFIG_HOME/mk
guarda='[[ $- == *i* ]] || return'

# O mk.bash só carrega num bash interativo. Aqui ele roda sem a linha da
# guarda e com /usr/share/ apontando para dentro da caixa (ble.sh, fzf,
# bash-completion, a VERSION do Mk instalado): o resto é o arquivo como está.
grep -vxF -- "$guarda" "$src" | sed "s|/usr/share/|$SBX/usr/share/|g" >"$SBX/mk.bash"
mkdir -p "$cfg/shell/scripts" "$cfg/bash" "$cfg/fastfetch" "$SBX/usr/share" "$SBX/base"

# Só o básico no PATH: as ferramentas que o mk.bash procura (starship, eza,
# fzf...) só existem quando o caso as declara com `stub`.
for c in bash sh env cat mkdir grep sed sleep head wc sort tr ls; do
  p=$(PATH=/usr/bin:/bin command -v "$c") && ln -sf "$p" "$SBX/base/$c"
done
P=$SBX/bin:$SBX/guard:$SBX/base

# mkb [VAR=valor...] '<comandos>': os comandos num bash novo, não interativo,
# com o mk.bash carregado. TERM de terminal comum, a não ser que o caso diga.
mkb() {
  local vars=()
  while (($# > 1)); do
    vars+=("$1")
    shift
  done
  env -i HOME="$HOME" XDG_CONFIG_HOME="$XDG_CONFIG_HOME" PATH="$P" SBX="$SBX" TERM=xterm-256color "${vars[@]}" \
    bash --norc --noprofile -c 'source "$SBX/mk.bash" || exit 90
'"$1"
}
sem() { rm -f "${SBX:?}/bin/$1"; }
# Os scripts do Maré que o `mk` chama, como dublês que só anotam.
for s in mare-devtools mare-start; do
  printf '#!/bin/bash\nprintf "%%s\\n" "%s $*" >>"%s/calls"\n' "$s" "$SBX" >"$cfg/shell/scripts/$s"
done

# ── carregar ──────────────────────────────────────────────────────────
t "a guarda do bash interativo está no arquivo, uma vez"
eq 1 "$(grep -cxF -- "$guarda" "$src")"

t "num bash não interativo o arquivo de verdade não faz nada: nenhuma função, nenhuma saída"
out=$(env -i HOME="$HOME" PATH="$P" SRC="$src" bash --norc --noprofile -c 'source "$SRC"; declare -F mk mkcd extract; alias; echo "fim:$?"' 2>&1)
eq "fim:0" "$out"

t "carrega sem nenhuma ferramenta instalada: sem erro e sem escrever nada"
out=$(mkb 'echo "fim"' 2>&1)
eq "fim" "$out"
eq "" "$(cat "$SBX/calls")"

t "define o comando mk e as funções de atalho"
out=$(mkb 'declare -F mk mkcd extract porta servir mk-fetch mk_precmd | sort | tr "\n" " "')
eq "extract mk mk-fetch mk_precmd mkcd porta servir " "$out"

t "o que é de arrumação interna não sobra no ambiente (mk_ble, mk_face, mk_name)"
out=$(mkb 'declare -F mk_face; echo "${mk_ble-sem}/${mk_name-sem}"')
eq "sem/sem" "$out"

t "histórico grande, sem repetidos, com data, gravado em modo de acréscimo"
out=$(mkb 'echo "$HISTSIZE $HISTFILESIZE $HISTCONTROL"; shopt -q histappend && echo acrescenta')
eq $'50000 100000 ignoreboth:erasedups\nacrescenta' "$out"

t "o seu.bash é carregado por último e vale sobre o do Mk"
stub eza
echo "alias ls='ls -1'; MEU=sim" >"$cfg/seu.bash"
out=$(mkb 'echo "$MEU"; alias ls')
eq $'sim\nalias ls=\'ls -1\'' "$out"
rm -f "${cfg:?}/seu.bash"
sem eza

t "sem seu.bash, carrega do mesmo jeito"
ok mkb 'true'

# ── ficha do fastfetch ────────────────────────────────────────────────
t "fastfetch: mostra a ficha ao abrir o ghostty (e o kitty), com a config do Mk"
stub fastfetch
calls_reset
mkb TERM=xterm-ghostty 'true'
eq "fastfetch --config $cfg/fastfetch/config.jsonc --pipe false" "$(cat "$SBX/calls")"
calls_reset
mkb TERM=xterm-kitty 'true'
called "fastfetch --config"

t "fastfetch: não mostra em outro terminal, no tmux, no neovim, num shell aninhado nem com MK_NO_FETCH"
calls_reset
mkb TERM=xterm-256color 'true'
mkb TERM=linux MK_NO_BLE=1 'true'
mkb TERM=xterm-ghostty TMUX=/tmp/tmux-1000/default,1,0 'true'
mkb TERM=xterm-ghostty NVIM=/run/user/1000/nvim.1.0 'true'
mkb TERM=xterm-ghostty MK_NO_FETCH=1 'true'
mkb TERM=xterm-ghostty SHLVL=1 'true' # o bash soma 1: é o segundo nível
not_called "fastfetch"

t "fastfetch: sem o programa instalado, não tenta"
sem fastfetch
out=$(mkb TERM=xterm-ghostty 'echo fim' 2>&1)
eq "fim" "$out"

t "mk ficha: chama o fastfetch com a config do Mk, em qualquer terminal"
stub fastfetch
calls_reset
mkb 'mk ficha'
eq "fastfetch --config $cfg/fastfetch/config.jsonc --pipe false" "$(cat "$SBX/calls")"
calls_reset
mkb 'mk fetch'
called "fastfetch --config"
sem fastfetch

# ── prompt ────────────────────────────────────────────────────────────
t "prompt: com o starship, usa o starship.toml do Mk e grava o histórico a cada comando"
stub starship 'echo "STARSHIP_LIGADO=sim"'
calls_reset
out=$(mkb 'echo "$STARSHIP_CONFIG|$starship_precmd_user_func|$STARSHIP_LIGADO"')
eq "$cfg/bash/starship.toml|mk_precmd|sim" "$out"
called "starship init bash"
exists "$REPO/config/bash/starship.toml"

t "prompt: sem o starship, um prompt simples nas cores Folha e o histórico a cada comando"
sem starship
out=$(mkb 'echo "$PROMPT_COMMAND"; echo "$PS1"')
has "$out" "mk_precmd"
has "$out" '38;2;206;94;50m' "cobre"
has "$out" '\w'

t "prompt: sem o starship, o PROMPT_COMMAND que já existia continua"
echo 'PROMPT_COMMAND="meu_gancho"' >"$SBX/antes.bash"
out=$(env -i HOME="$HOME" XDG_CONFIG_HOME="$XDG_CONFIG_HOME" PATH="$P" SBX="$SBX" TERM=xterm-256color \
  bash --norc --noprofile -c 'source "$SBX/antes.bash"; source "$SBX/mk.bash"; echo "$PROMPT_COMMAND"')
eq "mk_precmd; meu_gancho" "$out"

# ── ferramentas ───────────────────────────────────────────────────────
t "zoxide: ligado quando existe"
stub zoxide 'echo "ZOXIDE_LIGADO=sim"'
calls_reset
eq "sim" "$(mkb 'echo "$ZOXIDE_LIGADO"')"
called "zoxide init bash"
sem zoxide

t "fzf nas cores Folha (cobre no realce, osso no texto)"
out=$(mkb 'echo "$FZF_DEFAULT_OPTS"')
has "$out" "hl:#ce5e32"
has "$out" "fg+:#cdc4ba"
has "$out" "--layout=reverse"

t "fzf procura com o fd quando ele existe, e não inventa comando sem ele"
eq "" "$(mkb 'echo "${FZF_DEFAULT_COMMAND-}${FZF_CTRL_T_COMMAND-}${FZF_ALT_C_COMMAND-}"')"
stub fd
out=$(mkb 'echo "$FZF_DEFAULT_COMMAND|$FZF_CTRL_T_COMMAND|$FZF_ALT_C_COMMAND"')
eq "fd --type f --hidden --exclude .git|fd --type f --hidden --exclude .git|fd --type d --hidden --exclude .git" "$out"
sem fd

t "fzf sem o ble.sh: carrega os atalhos e a completação do pacote fzf"
stub fzf
mkdir -p "$SBX/usr/share/fzf"
echo 'FZF_TECLAS=sim' >"$SBX/usr/share/fzf/key-bindings.bash"
echo 'FZF_COMPLETA=sim' >"$SBX/usr/share/fzf/completion.bash"
eq "sim/sim" "$(mkb 'echo "$FZF_TECLAS/$FZF_COMPLETA"')"
sem fzf
eq "/" "$(mkb 'echo "${FZF_TECLAS-}/${FZF_COMPLETA-}"')" "sem o fzf não carrega"

t "bash-completion: carregado quando existe"
mkdir -p "$SBX/usr/share/bash-completion"
echo 'COMPLETA=sim' >"$SBX/usr/share/bash-completion/bash_completion"
eq "sim" "$(mkb 'echo "$COMPLETA"')"

# ── atalhos ───────────────────────────────────────────────────────────
t "ls, ll, la e lt viram eza só quando o eza existe"
eq "" "$(mkb 'alias ls ll la lt 2>/dev/null')"
stub eza
out=$(mkb 'alias ls ll la lt')
has "$out" "alias ls='eza --group-directories-first'"
has "$out" "alias ll='eza -l "
has "$out" "alias la='eza -la "
has "$out" "alias lt='eza --tree --level=2 "
sem eza

t "cat, v e lg só existem com o bat, o nvim e o lazygit"
eq "" "$(mkb 'alias cat v lg 2>/dev/null')"
stub bat
stub nvim
stub lazygit
out=$(mkb 'alias cat v lg')
has "$out" "alias cat='bat --paging=never --style=plain'"
has "$out" "alias v='nvim'"
has "$out" "alias lg='lazygit'"
sem bat
sem nvim
sem lazygit

t "os atalhos do git e de pasta existem sempre"
out=$(mkb 'alias g gs gd ga gc gp gl .. ... ....')
has "$out" "alias g='git'"
has "$out" "alias gs='git status -sb'"
has "$out" "alias gl='git log --oneline --graph --decorate -20'"
has "$out" "alias ..='cd ..'"
has "$out" "alias ....='cd ../../..'"

t "mkcd: cria a pasta (com os pais) e entra nela"
out=$(mkb 'cd "$SBX" && mkcd "a b/c d" && pwd')
eq "$SBX/a b/c d" "$out"

t "mkcd: uma pasta que começa com traço não vira opção"
out=$(mkb 'cd "$SBX" && mkcd -p-x && pwd')
eq "$SBX/-p-x" "$out"

t "extract: tar.zst pelo tar, 7z pelo 7z, o resto pelo bsdtar"
stub tar
stub 7z
stub bsdtar
touch "$SBX/a.tar.zst" "$SBX/b.7z" "$SBX/c.zip" "$SBX/d.tar.gz" "$SBX/e.rar"
calls_reset
mkb 'cd "$SBX" && extract a.tar.zst b.7z c.zip d.tar.gz e.rar'
eq $'tar --zstd -xf a.tar.zst\n7z x b.7z\nbsdtar -xf c.zip\nbsdtar -xf d.tar.gz\nbsdtar -xf e.rar' "$(cat "$SBX/calls")"

t "extract: um arquivo que não existe é avisado e os outros seguem"
calls_reset
out=$(mkb 'cd "$SBX" && extract sumiu.zip c.zip' 2>&1)
has "$out" "extract: sumiu.zip não existe"
eq "bsdtar -xf c.zip" "$(cat "$SBX/calls")"

t "porta: com número, só quem ouve nela; sem número, todas"
stub ss 'printf "%s\n" "tcp LISTEN 0 4096 0.0.0.0:3000 users:((node))" "tcp LISTEN 0 4096 0.0.0.0:30001 users:((x))" "udp UNCONN 0 0 0.0.0.0:5353 users:((avahi))"'
eq "tcp LISTEN 0 4096 0.0.0.0:3000 users:((node))" "$(mkb 'porta 3000')"
eq 3 "$(mkb 'porta' | wc -l)"
called "ss -ltnup"

t "servir: a pasta atual na porta 8000, ou na que for dita"
stub python
calls_reset
mkb 'servir; servir 9090'
eq $'python -m http.server 8000\npython -m http.server 9090' "$(cat "$SBX/calls")"

# ── o comando mk ──────────────────────────────────────────────────────
ipc="qs -p $cfg/shell ipc call mare"

t "mk (sem argumento), mk ajuda e um comando desconhecido mostram a ajuda"
ajuda=$(mkb 'mk')
has "$ajuda" "mk atualizar"
eq "$ajuda" "$(mkb 'mk ajuda')"
eq "$ajuda" "$(mkb 'mk bobagem')"
eq "" "$(cat "$SBX/calls" | grep -v '^python\|^ss' || true)"

t "todo comando da ajuda existe, e toda palavra da completação está na ajuda"
stub qs
stub mk-update
stub fastfetch
stub nvim
palavras=$(mkb 'complete -p mk' | sed -n "s/^complete -W [\"']\(.*\)[\"'] mk\$/\1/p")
like "$palavras" 'atualizar'
for w in $palavras; do
  [[ $w == ajuda ]] || has "$ajuda" "mk $w"
done
while read -r w; do
  has " $palavras " " $w " "na completação"
  # Um comando que existe não cai na ajuda.
  hasnt "$(mkb "mk $w" 2>&1)" "mk atualizar" "mk $w caiu na ajuda"
done < <(sed -n 's/^  mk \([a-z-]*\).*/\1/p' <<<"$ajuda")
# O reiniciar-mare roda em segundo plano: espera ele anotar antes de seguir.
for _ in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20; do grep -q mare-start "$SBX/calls" && break; sleep 0.1; done
sem fastfetch
sem nvim

t "mk atualizar (e update): chama o mk-update"
calls_reset
mkb 'mk atualizar; mk update'
eq $'mk-update \nmk-update ' "$(cat "$SBX/calls")"

t "mk atualizar devolve o código do mk-update"
stub mk-update 'exit 3'
mkb 'mk atualizar'
eq 3 $?
stub mk-update

t "mk ajustes [página]: abre os Ajustes pelo IPC do Maré, calado"
calls_reset
out=$(mkb 'mk ajustes; mk ajustes som; mk settings energia' 2>&1)
eq "$ipc settings "$'\n'"$ipc settings som"$'\n'"$ipc settings energia" "$(cat "$SBX/calls")"
eq "" "$out"

t "mk papel, mk bloquear e mk boas-vindas: cada um o seu pedido ao Maré"
calls_reset
mkb 'mk papel; mk wallpaper; mk bloquear; mk lock; mk boas-vindas; mk welcome'
eq "$ipc toggle wallpaper"$'\n'"$ipc toggle wallpaper"$'\n'"$ipc lock"$'\n'"$ipc lock"$'\n'"$ipc welcome abrir"$'\n'"$ipc welcome abrir" "$(cat "$SBX/calls")"

t "o que o qs responde não suja o terminal"
stub qs 'echo "ok do ipc"'
eq "" "$(mkb 'mk papel; mk bloquear; mk boas-vindas; mk ajustes')"
stub qs

t "mk ferramentas (e tools): roda o scripts/mare-devtools da instalação"
calls_reset
mkb 'mk ferramentas; mk tools'
eq $'mare-devtools \nmare-devtools ' "$(cat "$SBX/calls")"
exists "$REPO/shell/mare/scripts/mare-devtools"

t "mk reiniciar-mare: chama o scripts/mare-start --restart em segundo plano, sem prender o terminal"
calls_reset
out=$(mkb 'mk reiniciar-mare; echo "voltou"' 2>&1)
eq "voltou" "$out"
for _ in 1 2 3 4 5 6 7 8 9 10; do grep -q mare-start "$SBX/calls" && break; sleep 0.1; done
eq "mare-start --restart" "$(cat "$SBX/calls")"
exists "$REPO/shell/mare/scripts/mare-start"

t "mk versao: a VERSION de ~/.config/mk"
echo "2026.11.23" >"$cfg/VERSION"
eq "2026.11.23" "$(mkb 'mk versao')"
eq "2026.11.23" "$(mkb 'mk version')"

t "mk versao: sem ela, a do Mk instalado no sistema; sem nenhuma, diz que não há"
rm -f "${cfg:?}/VERSION"
mkdir -p "$SBX/usr/share/mk"
echo "2026.11.20" >"$SBX/usr/share/mk/VERSION"
eq "2026.11.20" "$(mkb 'mk versao' 2>&1)"
rm -f "${SBX:?}/usr/share/mk/VERSION"
eq "sem versão" "$(mkb 'mk versao' 2>&1)"

t "mk seu: abre o seu.bash no \$EDITOR, ou no nvim sem ele"
stub nvim
stub meu-editor
calls_reset
mkb 'mk seu; EDITOR=meu-editor; mk seu'
eq "nvim $cfg/seu.bash"$'\n'"meu-editor $cfg/seu.bash" "$(cat "$SBX/calls")"
sem nvim

t "sem XDG_CONFIG_HOME, o mk usa ~/.config/mk"
calls_reset
env -i HOME="$HOME" PATH="$P" SBX="$SBX" TERM=xterm-256color bash --norc --noprofile -c 'source "$SBX/mk.bash"; mk bloquear'
eq "qs -p $HOME/.config/mk/shell ipc call mare lock" "$(cat "$SBX/calls")"

# ── ble.sh ────────────────────────────────────────────────────────────
# Um ble.sh falso: anota como foi carregado e o que lhe pedem. O 0.4 tem o
# ble-face; o 0.3, só o ble-color-setface.
ble() { # ble 0.4|0.3
  mkdir -p "$SBX/usr/share/blesh"
  {
    echo 'printf "%s\n" "ble.sh $*" >>"$SBX/calls"'
    for f in bleopt ble-import ble-attach ble-color-setface; do
      echo "$f() { printf '%s\n' \"$f \$*\" >>\"\$SBX/calls\"; }"
    done
    [[ $1 == 0.3 ]] || echo "ble-face() { printf '%s\n' \"ble-face \$*\" >>\"\$SBX/calls\"; }"
  } >"$SBX/usr/share/blesh/ble.sh"
}

t "ble.sh: carregado cedo com --noattach e ligado no fim, depois do seu.bash"
ble 0.4
echo 'printf "%s\n" "seu.bash" >>"$SBX/calls"' >"$cfg/seu.bash"
calls_reset
ok mkb 'true'
eq "ble.sh --noattach" "$(head -1 "$SBX/calls")"
eq $'seu.bash\nble-attach ' "$(tail -2 "$SBX/calls")"
rm -f "${cfg:?}/seu.bash"

t "ble.sh 0.4: as cores Folha pelo ble-face, incluindo as variáveis por tipo"
called "ble-face syntax_command=fg=#ce5e32"
called "ble-face syntax_default=fg=#cdc4ba"
called "ble-face syntax_error=fg=#ff5f40,bold"
called "ble-face auto_complete=fg=#716b69"
called "ble-face varname_export=fg=#958f87"
called "ble-face varname_unset=fg=#958f87"
not_called "ble-color-setface"

t "ble.sh: sem as marcas de erro e de tempo (o starship já mostra)"
called "bleopt exec_errexit_mark="
called "bleopt exec_elapsed_mark="

t "ble.sh 0.3 (sem ble-face): as cores pelo ble-color-setface, sem as faces que ele não tem"
ble 0.3
calls_reset
out=$(mkb 'echo fim' 2>&1)
eq "fim" "$out"
called "ble-color-setface syntax_command fg=#ce5e32"
called "ble-color-setface auto_complete fg=#716b69"
not_called "varname_"
not_called "ble-face"
called "ble-attach"

t "ble.sh com o fzf: a integração é a do ble.sh, não os arquivos do pacote fzf"
ble 0.4
stub fzf
calls_reset
eq "/" "$(mkb 'echo "${FZF_TECLAS-}/${FZF_COMPLETA-}"')"
called "ble-import -d integration/fzf-completion"
called "ble-import -d integration/fzf-key-bindings"
sem fzf

t "MK_NO_BLE=1 desliga o ble.sh"
calls_reset
ok mkb MK_NO_BLE=1 'true'
eq "" "$(cat "$SBX/calls")"

t "no console (TERM=linux) o ble.sh não é carregado"
calls_reset
ok mkb TERM=linux 'true'
eq "" "$(cat "$SBX/calls")"

t "sem o ble.sh instalado, nada dele é chamado"
rm -rf "${SBX:?}/usr/share/blesh"
calls_reset
out=$(mkb 'echo fim' 2>&1)
eq "fim" "$out"
eq "" "$(cat "$SBX/calls")"

# ── os arquivos ao lado ───────────────────────────────────────────────
t "seu.bash.exemplo: só comentários (carregá-lo não muda nada) e ensina o 'mk seu'"
exemplo=$REPO/config/bash/seu.bash.exemplo
eq "" "$(grep -v '^#' "$exemplo" | grep -v '^[[:space:]]*$' || true)"
has "$(cat "$exemplo")" "mk seu"

t "starship.toml: uma linha de prompt com as cores Folha, e é um TOML válido"
toml=$REPO/config/bash/starship.toml
has "$(cat "$toml")" 'success_symbol = "[›](#ce5e32)"'
if command -v python3 >/dev/null; then
  ok python3 -c 'import sys, tomllib; tomllib.load(open(sys.argv[1], "rb"))' "$toml"
fi
