#!/usr/bin/env bash
# Firewall padrão: nega toda entrada, libera saída. Idempotente.
# Mesmo princípio do install/config/firewall.sh do Omarchy.
set -euo pipefail

sudo ufw default deny incoming
sudo ufw default allow outgoing
# LocalSend
sudo ufw allow 53317/tcp
sudo ufw allow 53317/udp
# DNS do Docker para os containers
sudo ufw allow in proto udp from 172.16.0.0/12 to 172.17.0.1 port 53 comment 'allow-docker-dns'
sudo ufw --force enable
sudo systemctl enable --now ufw.service

# Impede o Docker de abrir portas por fora do ufw
if command -v ufw-docker >/dev/null; then
  sudo ufw-docker install
fi
