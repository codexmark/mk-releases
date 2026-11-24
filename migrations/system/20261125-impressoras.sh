#!/usr/bin/env bash
# Impressoras: o CUPS (com o ipp-usb, para as USB sem driver) instalado e
# ligado, e o grupo wheel como administrador dele, que é o padrão do Arch.
# Quem instala é o mk-update, junto com a atualização do sistema; aqui fica
# só a configuração, que roda depois dela.
# pacotes: cups ipp-usb jq
set -euo pipefail
if ! pacman -Q cups &>/dev/null; then
  echo "aviso: o cups não foi instalado; a página Impressoras fica sem ele" >&2
  exit 0
fi

R=${MK_ROOT:-} # outra raiz no lugar de /: só os testes trocam (migrations/README.md)
conf=$R/etc/cups/cups-files.conf
if [[ -f $conf ]] && ! grep -Eq '^SystemGroup .*\bwheel\b' "$conf"; then
  if grep -q '^SystemGroup ' "$conf"; then
    sed -i 's/^SystemGroup .*/& wheel/' "$conf"
  else
    echo 'SystemGroup sys root wheel' >>"$conf"
  fi
  systemctl try-restart cups.service
fi
# Sem o PeerCred o CUPS não reconhece o usuário pelo soquete local, e a
# página Impressoras vai pedir a senha (polkit) a cada mudança.
if [[ -f $conf ]] && grep -Eiq '^PeerCred[[:space:]]+off' "$conf"; then
  echo "aviso: $conf tem 'PeerCred off'; administrar impressoras vai pedir senha" >&2
fi
systemctl enable --now cups.socket
