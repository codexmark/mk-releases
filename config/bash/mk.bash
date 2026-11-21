# Bash do Mk. Carregado pelo ~/.bashrc (o instalador põe a linha lá).
#
# O que tem: o ble.sh (sugestões em cinza enquanto digita, Tab completa, e
# cores de sintaxe), histórico grande e dividido entre terminais, o prompt do
# starship, zoxide, fzf, completação, atalhos curtos e o comando `mk`. Tudo
# nas cores Folha. É do Mk e o mk-update troca; o que for só seu vai em
# ~/.config/mk/seu.bash, que nenhuma atualização toca.
[[ $- == *i* ]] || return

mk_cfg=${XDG_CONFIG_HOME:-$HOME/.config}/mk

# ── ble.sh: carrega cedo e liga no fim (é como ele pede) ──────────────
# MK_NO_BLE=1 no ambiente desliga, para comparar ou se algo estranhar.
mk_ble=0
if [[ -r /usr/share/blesh/ble.sh && -z ${MK_NO_BLE:-} && $TERM != linux ]]; then
  source /usr/share/blesh/ble.sh --noattach && mk_ble=1
fi

# ── histórico ─────────────────────────────────────────────────────────
HISTSIZE=50000
HISTFILESIZE=100000
HISTCONTROL=ignoreboth:erasedups
HISTTIMEFORMAT='%F %T  '
HISTIGNORE='ls:ll:la:cd:pwd:exit:clear'
shopt -s histappend checkwinsize autocd cdspell dirspell globstar extglob

# ── a ficha do fastfetch ──────────────────────────────────────────────
# Só no terminal gráfico (a imagem usa o protocolo do kitty, que o ghostty e
# o kitty entendem), uma vez por janela, nunca dentro do tmux ou do neovim.
mk-fetch() {
  fastfetch --config "$mk_cfg/fastfetch/config.jsonc" --pipe false "$@"
}
if command -v fastfetch >/dev/null && [[ -z ${TMUX:-}${NVIM:-}${MK_NO_FETCH:-} ]] &&
  [[ $TERM == xterm-ghostty || $TERM == xterm-kitty ]] && [[ $SHLVL -le 1 ]]; then
  mk-fetch
fi

# ── prompt ────────────────────────────────────────────────────────────
# Cada comando entra no histórico na hora (outro terminal já vê no Ctrl+R).
mk_precmd() { history -a; }
if command -v starship >/dev/null; then
  export STARSHIP_CONFIG=$mk_cfg/bash/starship.toml
  starship_precmd_user_func=mk_precmd
  eval "$(starship init bash)"
else
  PROMPT_COMMAND="mk_precmd${PROMPT_COMMAND:+; $PROMPT_COMMAND}"
  PS1='\[\e[38;2;206;94;50m\]›\[\e[0m\] \[\e[38;2;205;196;186m\]\w\[\e[0m\] '
fi

# ── ferramentas ───────────────────────────────────────────────────────
[[ -r /usr/share/bash-completion/bash_completion ]] && source /usr/share/bash-completion/bash_completion
command -v zoxide >/dev/null && eval "$(zoxide init bash)"

# fzf nas cores Folha: Ctrl+R (histórico), Ctrl+T (arquivo), Alt+C (pasta).
export FZF_DEFAULT_OPTS="--layout=reverse --height=45% --border=none --info=inline-right \
--prompt='› ' --pointer='›' --marker='·' --separator='─' \
--color=bg:-1,bg+:#1b1917,fg:#b0a9a0,fg+:#cdc4ba,hl:#ce5e32,hl+:#ce5e32 \
--color=info:#716b69,prompt:#ce5e32,pointer:#ce5e32,marker:#ce5e32,spinner:#ce5e32 \
--color=header:#716b69,separator:#3a3632,scrollbar:#3a3632,border:#3a3632,query:#cdc4ba"
command -v fd >/dev/null && export FZF_DEFAULT_COMMAND='fd --type f --hidden --exclude .git' \
  FZF_CTRL_T_COMMAND='fd --type f --hidden --exclude .git' FZF_ALT_C_COMMAND='fd --type d --hidden --exclude .git'
if command -v fzf >/dev/null; then
  if ((mk_ble)); then
    ble-import -d integration/fzf-completion 2>/dev/null || ble-import integration/fzf-completion 2>/dev/null
    ble-import -d integration/fzf-key-bindings 2>/dev/null || ble-import integration/fzf-key-bindings 2>/dev/null
  else
    [[ -r /usr/share/fzf/key-bindings.bash ]] && source /usr/share/fzf/key-bindings.bash
    [[ -r /usr/share/fzf/completion.bash ]] && source /usr/share/fzf/completion.bash
  fi
fi

# ── atalhos ───────────────────────────────────────────────────────────
if command -v eza >/dev/null; then
  alias ls='eza --group-directories-first'
  alias ll='eza -l --group-directories-first --git --time-style=relative'
  alias la='eza -la --group-directories-first --git --time-style=relative'
  alias lt='eza --tree --level=2 --group-directories-first'
fi
command -v bat >/dev/null && alias cat='bat --paging=never --style=plain'
command -v nvim >/dev/null && alias v='nvim'
command -v lazygit >/dev/null && alias lg='lazygit'
alias ..='cd ..' ...='cd ../..' ....='cd ../../..'
alias g='git' gs='git status -sb' gd='git diff' ga='git add' gc='git commit' gp='git push'
alias gl='git log --oneline --graph --decorate -20'
alias grep='grep --color=auto' ip='ip -color=auto' df='df -h' free='free -h'

# Cria a pasta e entra nela.
mkcd() { mkdir -p -- "$1" && cd -- "$1" || return; }

# Abre qualquer arquivo compactado (tar, zip, 7z, rar...) na pasta atual.
extract() {
  local f
  for f in "$@"; do
    [[ -f $f ]] || { echo "extract: $f não existe" >&2; continue; }
    case $f in
      *.tar.zst) tar --zstd -xf "$f" ;;
      *.7z) 7z x "$f" ;;
      *) bsdtar -xf "$f" ;;
    esac
  done
}

# Quem está ouvindo numa porta: porta 3000 (sem número, todas).
porta() { ss -ltnup | { [[ -n ${1:-} ]] && grep -E ":$1\b" || cat; }; }

# Serve a pasta atual na rede local: servir [porta]
servir() { python -m http.server "${1:-8000}"; }

# ── o comando mk ──────────────────────────────────────────────────────
mk() {
  local shell=$mk_cfg/shell
  case ${1:-ajuda} in
    atualizar | update) mk-update ;;
    ajustes | settings) qs -p "$shell" ipc call mare settings "${2:-}" >/dev/null ;;
    papel | wallpaper) qs -p "$shell" ipc call mare toggle wallpaper >/dev/null ;;
    bloquear | lock) qs -p "$shell" ipc call mare lock >/dev/null ;;
    boas-vindas | welcome) qs -p "$shell" ipc call mare welcome abrir >/dev/null ;;
    reiniciar-mare) bash "$shell/scripts/mare-start" --restart >/dev/null 2>&1 & disown ;;
    ficha | fetch) mk-fetch ;;
    versao | version)
      cat "$mk_cfg/VERSION" 2>/dev/null || cat /usr/share/mk/VERSION 2>/dev/null || echo "sem versão"
      ;;
    seu) "${EDITOR:-nvim}" "$mk_cfg/seu.bash" ;;
    *)
      printf '  %-21s %s\n' \
        "mk atualizar" "o Mk, o Arch e o AUR (mk-update)" \
        "mk ajustes [página]" "abre os Ajustes (ilha, tela, som, energia...)" \
        "mk papel" "papel de parede" \
        "mk bloquear" "bloqueia a tela" \
        "mk boas-vindas" "o passeio pelo Mk, de novo" \
        "mk reiniciar-mare" "reinicia a ilha e os painéis" \
        "mk ficha" "a ficha do fastfetch" \
        "mk versao" "a versão do Mk instalada" \
        "mk seu" "edita o seu.bash (seus atalhos, que o Mk não toca)"
      ;;
  esac
}
complete -W "atualizar ajustes papel bloquear boas-vindas reiniciar-mare ficha versao seu ajuda" mk

# ── cores do ble.sh na paleta Folha ───────────────────────────────────
# Vale para o ble.sh 0.3 (o pacote blesh do AUR) e o 0.4: o 0.3 não tem o
# comando ble-face nem algumas opções, então cada ajuste é tentado em silêncio.
if ((mk_ble)); then
  mk_face() {
    if declare -F ble-face >/dev/null; then
      mk_face "$1" "$2"
    else
      ble-color-setface "$1" "$2"
    fi 2>/dev/null
  }
  bleopt complete_auto_delay=80 2>/dev/null
  # O starship já mostra erro (› em vermelho) e tempo do comando.
  bleopt exec_errexit_mark= 2>/dev/null
  bleopt exec_elapsed_mark= 2>/dev/null
  mk_face syntax_default 'fg=#cdc4ba'
  mk_face syntax_command 'fg=#ce5e32'
  mk_face command_builtin 'fg=#ce5e32'
  mk_face command_alias 'fg=#ce5e32'
  mk_face command_function 'fg=#ce5e32'
  mk_face command_file 'fg=#ce5e32'
  mk_face command_keyword 'fg=#b0a9a0,bold'
  mk_face command_jobs 'fg=#ce5e32'
  mk_face syntax_function_name 'fg=#ce5e32,bold'
  mk_face syntax_quoted 'fg=#b0a9a0'
  mk_face syntax_quotation 'fg=#b0a9a0'
  mk_face syntax_varname 'fg=#958f87'
  # Variáveis de todo tipo (exportada, vetor, número, não definida...): só
  # no 0.4; o 0.3 não tem essas faces e reclama delas mais tarde.
  declare -F ble-face >/dev/null && for mk_name in varname_array varname_empty varname_export varname_expr varname_hash \
    varname_number varname_readonly varname_transform varname_unset; do
    mk_face "$mk_name" 'fg=#958f87'
  done
  unset mk_name
  mk_face syntax_param_expansion 'fg=#958f87'
  mk_face syntax_expr 'fg=#958f87'
  mk_face syntax_glob 'fg=#ce5e32'
  mk_face syntax_brace 'fg=#958f87'
  mk_face syntax_delimiter 'fg=#807a73'
  mk_face syntax_comment 'fg=#716b69,italic'
  mk_face syntax_error 'fg=#ff5f40,bold'
  mk_face filename_directory 'fg=#cdc4ba,underline'
  mk_face filename_executable 'fg=#ce5e32'
  mk_face filename_other 'fg=#b0a9a0'
  mk_face auto_complete 'fg=#716b69'
  mk_face region 'bg=#3a3632'
  mk_face region_target 'bg=#3a3632'
fi

# ── o que é só seu ────────────────────────────────────────────────────
[[ -r $mk_cfg/seu.bash ]] && source "$mk_cfg/seu.bash"

((mk_ble)) && ble-attach
unset mk_ble
unset -f mk_face 2>/dev/null
