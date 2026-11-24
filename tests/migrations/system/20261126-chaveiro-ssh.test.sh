#!/usr/bin/env bash
# migrations/system/20261126-chaveiro-ssh.sh: liga o gcr-ssh-agent.socket para
# todos os usuários, quando o gcr-4 está instalado.
. "$(dirname "$0")/../../lib.sh"
. "$TESTS/migrations/mig.sh" system/20261126-chaveiro-ssh.sh

stub systemctl
unit=$R/usr/lib/systemd/user/gcr-ssh-agent.socket

t "declara o gcr-4 no cabeçalho"
eq "gcr-4" "$(sed -n 's/^# pacotes: //p' "$MIG")"

t "sem o gcr-4 (a unidade não existe): avisa, não liga nada e sai com 0"
sem_mudanca mig
has "$(cat "$ERR")" "aviso"
not_called "systemctl"

t "com a unidade do gcr-4: liga o soquete para todos os usuários (--global)"
mkdir -p "$(dirname "$unit")" && touch "$unit"
ok mig
eq "systemctl --global enable gcr-ssh-agent.socket" "$(cat "$SBX/calls")"

de_novo

t "é o soquete que o env.lua do Hyprland procura para apontar o SSH_AUTH_SOCK"
has "$(cat "$REPO/config/hypr/mare/env.lua")" "sockets.target.wants/gcr-ssh-agent.socket"

t "não instala nada"
not_called "pacman"

t "se o systemctl falhar, a migração falha (fica pendente para a próxima)"
stub systemctl 'exit 1'
fails mig
