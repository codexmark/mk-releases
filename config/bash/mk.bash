# Bash do Mk: a ficha do fastfetch ao abrir o terminal, o prompt e alguns
# atalhos. Carregado pelo ~/.bashrc (o instalador põe a linha lá).
[[ $- == *i* ]] || return

mk_cfg=${XDG_CONFIG_HOME:-$HOME/.config}/mk

# A ficha: só no terminal gráfico (a imagem usa o protocolo do kitty, que o
# ghostty e o kitty entendem), uma vez por janela, e nunca dentro do tmux ou
# do neovim. --pipe false: sem ele o fastfetch às vezes troca a imagem por texto.
mk-fetch() {
  fastfetch --config "$mk_cfg/fastfetch/config.jsonc" --pipe false "$@"
}
if command -v fastfetch >/dev/null && [[ -z ${TMUX:-}${NVIM:-}${MK_NO_FETCH:-} ]] &&
  [[ $TERM == xterm-ghostty || $TERM == xterm-kitty ]] && [[ $SHLVL -le 1 ]]; then
  mk-fetch
fi

# Prompt: starship com a config do Mk; sem starship, um prompt simples nas cores Folha.
if command -v starship >/dev/null; then
  export STARSHIP_CONFIG=$mk_cfg/bash/starship.toml
  eval "$(starship init bash)"
else
  PS1='\[\e[38;2;206;94;50m\]›\[\e[0m\] \[\e[38;2;205;196;186m\]\w\[\e[0m\] '
fi

command -v zoxide >/dev/null && eval "$(zoxide init bash)"
[[ -r /usr/share/fzf/key-bindings.bash ]] && source /usr/share/fzf/key-bindings.bash
[[ -r /usr/share/bash-completion/bash_completion ]] && source /usr/share/bash-completion/bash_completion

if command -v eza >/dev/null; then
  alias ls='eza --group-directories-first'
  alias ll='eza -l --group-directories-first --git'
  alias la='eza -la --group-directories-first --git'
fi
command -v bat >/dev/null && alias cat='bat --paging=never --style=plain'
