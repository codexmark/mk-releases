#!/usr/bin/env bash
# Aplica o LazyVim (starter oficial) se o usuário ainda não tem config do Neovim.
# Nunca sobrescreve uma config existente.
set -euo pipefail

target="${XDG_CONFIG_HOME:-$HOME/.config}/nvim"
if [[ -e $target ]]; then
  echo "$target já existe, deixando como está"
  exit 0
fi
git clone --depth 1 https://github.com/LazyVim/starter "$target"
rm -rf "$target/.git"
