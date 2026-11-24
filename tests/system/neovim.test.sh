#!/usr/bin/env bash
# scripts/user/neovim.sh: aplica o LazyVim só para quem ainda não tem config do
# Neovim. O git é um stub: nada é baixado.
. "$(dirname "$0")/../lib.sh"

nv=$REPO/scripts/user/neovim.sh
# O clone falso deixa o que o starter traria, com o .git dele.
stub git '
[[ -e $SBX/clone-falha ]] && exit 128
dest=${*: -1}
mkdir -p "$dest/.git" "$dest/lua/config"
echo "ref: refs/heads/main" >"$dest/.git/HEAD"
echo "require(\"config.lazy\")" >"$dest/init.lua"'
nvim=$XDG_CONFIG_HOME/nvim

t "sem config do Neovim: clona o starter do LazyVim em ~/.config/nvim"
out=$(bash "$nv" 2>&1)
eq 0 $? "$out"
called "git clone --depth 1 https://github.com/LazyVim/starter $nvim"
exists "$nvim/init.lua"
exists "$nvim/lua/config"

t "o .git do starter sai: a config passa a ser do usuário"
absent "$nvim/.git"

t "com config existente: deixa como está e não clona nada"
echo "-- a minha config" >"$nvim/init.lua"
calls_reset
out=$(bash "$nv" 2>&1)
eq 0 $?
has "$out" "$nvim já existe, deixando como está"
not_called "git"
eq "-- a minha config" "$(<"$nvim/init.lua")"

t "uma pasta nvim vazia, um arquivo ou um link também contam como config existente"
for forma in pasta arquivo link; do
  rm -rf "$nvim"
  case $forma in
    pasta) mkdir -p "$nvim" ;;
    arquivo) : >"$nvim" ;;
    link) mkdir -p "$SBX/dotfiles/nvim" && ln -s "$SBX/dotfiles/nvim" "$nvim" ;;
  esac
  calls_reset
  ok bash "$nv"
  not_called "git"
done
rm -rf "$nvim"

t "sem XDG_CONFIG_HOME, usa ~/.config/nvim"
calls_reset
out=$(env -u XDG_CONFIG_HOME bash "$nv" 2>&1)
eq 0 $? "$out"
called "git clone --depth 1 https://github.com/LazyVim/starter $HOME/.config/nvim"

t "com XDG_CONFIG_HOME em outro lugar, é lá que a config nasce"
calls_reset
ok env XDG_CONFIG_HOME="$SBX/outra" bash "$nv"
exists "$SBX/outra/nvim/init.lua"
absent "$SBX/outra/nvim/.git"

t "o instalador roda com HOME do usuário novo: a config nasce na home dele"
calls_reset
mkdir -p "$SBX/home-ana"
ok env -u XDG_CONFIG_HOME HOME="$SBX/home-ana" bash "$nv"
exists "$SBX/home-ana/.config/nvim/init.lua"

t "se o clone falha (sem internet), sai com erro para o instalador avisar"
rm -rf "$nvim"
: >"$SBX/clone-falha"
fails bash "$nv"
absent "$nvim/init.lua"

t "só fala com o GitHub do LazyVim e não usa sudo"
eq "" "$(grep -v '^git clone --depth 1 https://github.com/LazyVim/starter ' "$SBX/calls")"
