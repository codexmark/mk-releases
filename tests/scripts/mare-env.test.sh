#!/usr/bin/env bash
# shell/mare/scripts/mare-env: leva as variáveis de aparência da sessão ao
# systemd e ao D-Bus (pelo uwsm), só quando o Hyprland que chamou é o da
# sessão. systemctl e uwsm são stubs com o estado em $SBX.
. "$(dirname "$0")/../lib.sh"

env_sh=$REPO/shell/mare/scripts/mare-env

# systemctl --user show-environment: o que há em $SBX/ambiente.
stub systemctl '
case " $* " in
  *" show-environment "*) cat "$SBX/ambiente" 2>/dev/null ;;
  *" try-restart "*) [[ ! -e $SBX/restart-falha ]] || exit 1 ;;
esac'
stub uwsm '[[ ! -e $SBX/uwsm-falha ]] || exit 1'
stub sleep

run() {
  env -u GTK_THEME -u QT_QPA_PLATFORMTHEME -u QT_QPA_PLATFORM -u ELECTRON_OZONE_PLATFORM_HINT -u TERMINAL -u SSH_AUTH_SOCK \
    HYPRLAND_INSTANCE_SIGNATURE=sessao_1 "$@" bash "$env_sh"
}
zera() {
  calls_reset
  rm -f "$SBX/uwsm-falha" "$SBX/restart-falha"
  printf 'HOME=/home/x\nHYPRLAND_INSTANCE_SIGNATURE=sessao_1\nXCURSOR_SIZE=24\n' >"$SBX/ambiente"
}

t "na sessão do Mk: as variáveis de aparência que existem vão para o uwsm finalize"
zera
ok run GTK_THEME=Mk QT_QPA_PLATFORMTHEME=qt6ct "QT_QPA_PLATFORM=wayland;xcb" TERMINAL=ghostty
called "uwsm finalize GTK_THEME QT_QPA_PLATFORMTHEME QT_QPA_PLATFORM TERMINAL"

t "só nomes vão na linha de comando: o valor quem lê é o uwsm, do ambiente"
not_called "Mk"
not_called "qt6ct"

t "variável que a sessão não definiu fica de fora"
not_called "ELECTRON_OZONE_PLATFORM_HINT"
not_called "SSH_AUTH_SOCK"

t "o seletor de arquivos (portal GTK) já pode estar de pé com o ambiente velho: é reiniciado, se estiver"
called "systemctl --user try-restart xdg-desktop-portal-gtk.service"

t "todas as variáveis da lista, na ordem"
zera
ok run GTK_THEME=Mk QT_QPA_PLATFORMTHEME=qt6ct QT_QPA_PLATFORM=wayland ELECTRON_OZONE_PLATFORM_HINT=auto TERMINAL=ghostty SSH_AUTH_SOCK=/run/user/1000/gcr/ssh
called "uwsm finalize GTK_THEME QT_QPA_PLATFORMTHEME QT_QPA_PLATFORM ELECTRON_OZONE_PLATFORM_HINT TERMINAL SSH_AUTH_SOCK"

t "num Hyprland que não é o da sessão (aninhado, de teste): não exporta nada nem reinicia nada"
zera
printf 'HYPRLAND_INSTANCE_SIGNATURE=outra_sessao\n' >"$SBX/ambiente"
ok run GTK_THEME=Mk
not_called "uwsm"
not_called "try-restart"

t "sem o Hyprland da sessão no systemd ainda: espera um pouco (o uwsm finaliza logo no começo) e desiste calado"
zera
printf 'HOME=/home/x\n' >"$SBX/ambiente"
ok run GTK_THEME=Mk
not_called "uwsm"
eq 10 "$(grep -c '^sleep 0.5' "$SBX/calls")"

t "fora de um Hyprland (sem assinatura): não faz nada"
zera
ok env -u HYPRLAND_INSTANCE_SIGNATURE GTK_THEME=Mk bash "$env_sh"
not_called "uwsm"
not_called "systemctl"

t "sem o uwsm instalado: sai com 0 sem exportar (a sessão do Mk sempre o tem; é outro jeito de subir)"
zera
rm -f "$SBX/bin/uwsm"
ok env PATH="$(path_sem uwsm)" HYPRLAND_INSTANCE_SIGNATURE=sessao_1 GTK_THEME=Mk bash "$env_sh"
not_called "try-restart"
stub uwsm '[[ ! -e $SBX/uwsm-falha ]] || exit 1'

t "o uwsm recusando: avisa no erro, não reinicia o portal e não derruba a sessão (sai com 0)"
zera
: >"$SBX/uwsm-falha"
out=$(run GTK_THEME=Mk 2>&1)
eq 0 $?
has "$out" "mare-env"
not_called "try-restart"

t "nenhuma variável de aparência definida: nada a exportar"
zera
ok run
not_called "uwsm"
