#!/usr/bin/env bash
# bin/try-session: instala e remove a sessão do Mk na home (de $SBX), a partir
# de uma cópia mínima do repositório; nunca roda nada com sudo. pacman, git e
# Hyprland são stubs, e a raiz do sistema é falsa (MK_TRY_ROOT).
. "$(dirname "$0")/../lib.sh"

R=$SBX/raiz
falso=$SBX/repo
cfg=$XDG_CONFIG_HOME
dados=$XDG_DATA_HOME
feitas=$XDG_STATE_HOME/mk/migrations
marca=.instalado-pelo-mk

# Uma cópia mínima do repositório: o script e o que ele copia.
mkdir -p "$falso"/{bin,config/hypr,config/ghostty,config/fastfetch,config/bash,shell/mare,migrations/user,system/mk/look/themes/Mk}
cp "$REPO/bin/try-session" "$falso/bin/try-session"
echo "#!/bin/bash" >"$falso/bin/mk-session"
for d in config/hypr config/ghostty config/fastfetch config/bash shell/mare system/mk/look/themes/Mk; do echo "v1" >"$falso/$d/arquivo"; done
echo "-- config" >"$falso/config/hypr/hyprland.lua"
echo "# migração" >"$falso/migrations/user/20260101-a.sh"
echo "# migração" >"$falso/migrations/user/20260202-b.sh"
# O make-icons falso faz o que o de verdade faz com o destino: refaz a pasta.
cat >"$falso/system/mk/look/make-icons" <<'FALSO'
#!/bin/bash
[[ -e $SBX/icones-falham ]] && exit 1
rm -rf "${1:?}" && mkdir -p "$1" && echo "tema" >"$1/index.theme"
FALSO

stub pacman '[[ $1 == -Qq ]] && ! grep -qx -- "$2" "$SBX/faltam" 2>/dev/null'
stub git 'case " $* " in *rev-parse*) echo abc123def456 ;; *" log "*) echo 2026-11-20 ;; esac'
stub Hyprland 'echo "config ok"'

zera() {
  rm -rf "${R:?}" "${HOME:?}"
  mkdir -p "$R" "$HOME"
  rm -f "$SBX/faltam" "$SBX/icones-falham"
  calls_reset
}
sessao() { MK_TRY_ROOT=$R bash "$falso/bin/try-session" "$@" 2>&1; }
arvore() { (cd "$HOME" && find . | sort && find . -type f -exec md5sum {} + | sort); }

# ------------------------------------------------------------------ install

t "install: copia a config do Mk para ~/.config/mk, cada pasta com a marca"
zera
out=$(sessao install)
eq 0 $?
for d in hypr ghostty fastfetch bash shell; do
  eq v1 "$(cat "$cfg/mk/$d/arquivo" 2>/dev/null)" "$d"
  exists "$cfg/mk/$d/$marca"
done
has "$out" "copiado: $cfg/mk/hypr"

t "install: não toca na config do desktop atual (~/.config/hypr)"
absent "$cfg/hypr"
absent "$cfg/quickshell"

t "install: o tema GTK e os ícones do Mk vão para ~/.local/share, com a marca"
eq v1 "$(cat "$dados/themes/Mk/arquivo")"
exists "$dados/themes/Mk/$marca"
eq tema "$(cat "$dados/icons/Mk/index.theme")"
exists "$dados/icons/Mk/$marca"

t "install: grava a versão ('commit data') em ~/.config/mk/VERSION"
eq "abc123def456 2026-11-20" "$(cat "$cfg/mk/VERSION")"

t "install: na primeira vez, as migrações de usuário ficam marcadas como feitas"
exists "$feitas/20260101-a.sh"
exists "$feitas/20260202-b.sh"
exists "$feitas/.pelo-try-session"
has "$out" "marcado: $feitas"

t "install: confere a config do Hyprland instalada"
called "Hyprland --verify-config -c $cfg/mk/hypr/hyprland.lua"
has "$out" "config do Hyprland: ok"

t "install: os passos de root são só impressos, nunca rodados"
not_called "sudo"
not_called "pacman -S"
has "$out" "Passos com sudo (rode você mesmo):"
has "$out" "sudo install -Dm755 $falso/bin/mk-session /usr/local/bin/mk-session"
has "$out" "sudo tee /usr/share/wayland-sessions/mk.desktop"
hasnt "$out" "sudo pacman"
hasnt "$out" "meu-os"

t "install de novo: a home fica igual, e as migrações não são marcadas outra vez"
antes=$(arvore)
echo "# migração nova" >"$falso/migrations/user/20260303-c.sh"
out=$(sessao install)
eq 0 $?
eq "$antes" "$(arvore)"
absent "$feitas/20260303-c.sh"
hasnt "$out" "marcado:"
rm "$falso/migrations/user/20260303-c.sh"

t "install de novo: a config que o Mk criou é atualizada, e o que saiu do Mk some"
echo "v2" >"$falso/config/hypr/arquivo"
echo "resto" >"$cfg/mk/hypr/arquivo-antigo"
sessao install >/dev/null
eq v2 "$(cat "$cfg/mk/hypr/arquivo")"
absent "$cfg/mk/hypr/arquivo-antigo"
echo "v1" >"$falso/config/hypr/arquivo"

t "install: uma pasta que não foi o Mk que criou é recusada, e fica como estava"
zera
mkdir -p "$cfg/mk/ghostty"
echo "meu" >"$cfg/mk/ghostty/config"
out=$(sessao install)
eq 1 $?
has "$out" "ERRO: $cfg/mk/ghostty já existe e não foi criado pelo Mk"
eq meu "$(cat "$cfg/mk/ghostty/config")"
absent "$cfg/mk/ghostty/$marca"

t "install: ícones 'Mk' que não foram o Mk que criou ficam como estão"
zera
mkdir -p "$dados/icons/Mk"
echo "meus" >"$dados/icons/Mk/index.theme"
sessao install >/dev/null
eq 0 $?
eq meus "$(cat "$dados/icons/Mk/index.theme")"
absent "$dados/icons/Mk/$marca"

t "install: sem o Papirus (o make-icons falha), o resto é instalado mesmo assim"
zera
touch "$SBX/icones-falham"
out=$(sessao install)
eq 0 $?
exists "$cfg/mk/shell/$marca"
absent "$dados/icons/Mk/$marca"
has "$out" "Passos com sudo"

t "install: quem já tinha o Maré (não é a primeira vez) não ganha as marcas de migração"
zera
mkdir -p "$cfg/mk/shell"
touch "$cfg/mk/shell/$marca"
sessao install >/dev/null
eq 0 $?
absent "$feitas"

t "install: a config do Hyprland com erro vira um aviso, não uma falha"
zera
stub Hyprland 'echo "config error: linha 3"; exit 1'
out=$(sessao install)
eq 0 $?
has "$out" "AVISO: a config do Hyprland tem erros"
hasnt "$out" "config do Hyprland: ok"
stub Hyprland 'echo "config ok"'

t "install: fora de um clone git, a versão é 'dev' com a data de hoje"
zera
stub git 'exit 128'
sessao install >/dev/null
eq "dev $(date +%F)" "$(cat "$cfg/mk/VERSION")"
stub git 'case " $* " in *rev-parse*) echo abc123def456 ;; *" log "*) echo 2026-11-20 ;; esac'

t "install: pacotes que faltam aparecem num 'sudo pacman -S --needed' impresso"
zera
printf 'quickshell\ncava\n' >"$SBX/faltam"
out=$(sessao install)
has "$out" "sudo pacman -S --needed quickshell cava"
not_called "pacman -S"

t "install: com o lançador e a entrada do login já no sistema, não manda instalar de novo"
zera
mkdir -p "$R/usr/local/bin" "$R/usr/share/wayland-sessions"
cp "$falso/bin/mk-session" "$R/usr/local/bin/mk-session"
echo "[Desktop Entry]" >"$R/usr/share/wayland-sessions/mk.desktop"
out=$(sessao install)
hasnt "$out" "sudo install"
hasnt "$out" "sudo tee"

t "install: lançador do sistema diferente do deste repositório: manda reinstalar"
echo "# versão velha" >>"$R/usr/local/bin/mk-session"
out=$(sessao install)
has "$out" "sudo install -Dm755 $falso/bin/mk-session /usr/local/bin/mk-session"

t "install: a sessão antiga 'meu-os' no sistema: manda apagar"
touch "$R/usr/local/bin/meu-os-session"
out=$(sessao install)
has "$out" "sudo rm -f /usr/local/bin/meu-os-session /usr/share/wayland-sessions/meu-os.desktop"

t "install: apaga a config antiga do meu-os e o Maré antigo, só as partes com a marca"
zera
mkdir -p "$cfg/meu-os/hypr" "$cfg/meu-os/shell" "$cfg/quickshell/mare"
touch "$cfg/meu-os/hypr/.instalado-pelo-meu-os" "$cfg/quickshell/mare/$marca"
echo "meu" >"$cfg/meu-os/shell/shell.qml"
out=$(sessao install)
eq 0 $?
absent "$cfg/meu-os/hypr"
eq meu "$(cat "$cfg/meu-os/shell/shell.qml")"
absent "$cfg/quickshell/mare"
has "$out" "removido (meu-os antigo): $cfg/meu-os/hypr"

t "install: um ~/.config/quickshell/mare que não é do Mk fica, com aviso"
zera
mkdir -p "$cfg/quickshell/mare"
echo "meu" >"$cfg/quickshell/mare/shell.qml"
out=$(sessao install)
eq meu "$(cat "$cfg/quickshell/mare/shell.qml")"
has "$out" "mantido (não é do Mk): $cfg/quickshell/mare"

# ---------------------------------------------------------------- uninstall

t "uninstall: remove tudo o que o install criou (a home volta a ficar vazia)"
zera
sessao install >/dev/null
out=$(sessao uninstall)
eq 0 $?
eq "" "$(cd "$HOME" && find . -not -type d)"
absent "$cfg/mk"
absent "$dados/themes/Mk"
absent "$dados/icons/Mk"
absent "$feitas"
absent "$XDG_STATE_HOME/mk"
has "$out" "removido: $cfg/mk/hypr"

t "uninstall: nunca roda nada com sudo"
not_called "sudo"

t "uninstall de novo: não há mais nada a remover, e sai com 0"
out=$(sessao uninstall)
eq 0 $?
hasnt "$out" "removido"

t "uninstall: o que não é do Mk fica, com aviso, e a pasta ~/.config/mk também"
zera
sessao install >/dev/null
rm "$cfg/mk/ghostty/$marca"
echo '{"meu":1}' >"$cfg/mk/ajustes.json"
out=$(sessao uninstall)
eq 0 $?
has "$out" "mantido (não é do Mk): $cfg/mk/ghostty"
exists "$cfg/mk/ghostty/arquivo"
eq '{"meu":1}' "$(cat "$cfg/mk/ajustes.json")"
absent "$cfg/mk/hypr"

t "uninstall: as marcas de migração que não foi ele que fez ficam"
zera
mkdir -p "$feitas"
touch "$feitas/20260101-a.sh"
mkdir -p "$cfg/mk/shell"
touch "$cfg/mk/shell/$marca"
sessao install >/dev/null
sessao uninstall >/dev/null
exists "$feitas/20260101-a.sh"

t "uninstall: com a sessão no sistema, diz o comando para tirar (sem rodar)"
zera
mkdir -p "$R/usr/share/wayland-sessions"
touch "$R/usr/share/wayland-sessions/mk.desktop"
sessao install >/dev/null
out=$(sessao uninstall)
has "$out" "sudo rm -f /usr/share/wayland-sessions/mk.desktop /usr/local/bin/mk-session"
not_called "sudo"

t "uninstall: sem a sessão no sistema, não fala em sudo"
zera
sessao install >/dev/null
out=$(sessao uninstall)
hasnt "$out" "sudo"

# ------------------------------------------------------------------- status

t "status: sem nada instalado, diz o que falta"
zera
out=$(sessao status)
eq 0 $?
has "$out" "falta   $cfg/mk/hypr"
has "$out" "falta   $cfg/mk/shell"
has "$out" "falta   /usr/local/bin/mk-session"
has "$out" "falta   /usr/share/wayland-sessions/mk.desktop"
has "$out" "pacotes: ok"

t "status: é o padrão, sem argumento, e não muda nada"
eq "$out" "$(sessao)"
eq "." "$(cd "$HOME" && find .)"

t "status: com tudo instalado, tudo ok; pacotes que faltam são listados"
zera
mkdir -p "$R/usr/local/bin" "$R/usr/share/wayland-sessions"
touch "$R/usr/local/bin/mk-session" "$R/usr/share/wayland-sessions/mk.desktop"
sessao install >/dev/null
echo matugen >"$SBX/faltam"
out=$(sessao status)
has "$out" "ok      $cfg/mk/hypr"
has "$out" "ok      $cfg/mk/shell"
has "$out" "ok      /usr/local/bin/mk-session"
has "$out" "ok      /usr/share/wayland-sessions/mk.desktop"
has "$out" "pacotes que faltam: matugen"

t "argumento desconhecido: mostra o uso e sai com 2"
out=$(sessao instalar)
eq 2 $?
has "$out" "uso: bin/try-session install | uninstall | status"
