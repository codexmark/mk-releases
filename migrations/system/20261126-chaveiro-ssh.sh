#!/usr/bin/env bash
# Chaves SSH com senha lembradas pelo chaveiro: liga o gcr-ssh-agent para
# todos os usuários. Vale no próximo login (o Hyprland aponta o SSH_AUTH_SOCK
# para ele quando o soquete está ligado; veja config/hypr/mare/env.lua).
# pacotes: gcr-4
set -euo pipefail
if [[ ! -f /usr/lib/systemd/user/gcr-ssh-agent.socket ]]; then
  echo "aviso: o gcr-4 não foi instalado; o agente SSH do chaveiro fica desligado" >&2
  exit 0
fi
systemctl --global enable gcr-ssh-agent.socket
