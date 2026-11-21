#!/usr/bin/env bash
# Página VPN dos Ajustes: o OpenVPN precisa do plugin do NetworkManager
# (o WireGuard já vem no kernel). Quem instala é o mk-update, junto com a
# atualização do sistema. É opcional: se não veio, a própria página VPN
# oferece instalar depois.
# pacotes: networkmanager-vpn-plugin-openvpn
set -euo pipefail
pacman -Q networkmanager-vpn-plugin-openvpn &>/dev/null ||
  echo "plugin do openvpn ficou para depois: instale pela página VPN dos Ajustes"
