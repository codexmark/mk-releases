#!/usr/bin/env bash
# scripts/system/firewall.sh: as regras do ufw, na ordem, com sudo e ufw por
# stub (nada é aplicado nesta máquina).
. "$(dirname "$0")/../lib.sh"

fw=$REPO/scripts/system/firewall.sh
stub sudo
# Um PATH sem o ufw-docker desta máquina, se houver: só stubs, guardas e o básico.
mkdir -p "$SBX/minimo"
for c in bash env; do ln -s "$(command -v "$c")" "$SBX/minimo/$c"; done
roda() {
  calls_reset
  out=$(PATH=$SBX/bin:$SBX/guard:$SBX/minimo "$SBX/minimo/bash" "$fw" 2>&1)
  rc=$?
}

t "nega toda entrada e libera a saída"
roda
eq 0 "$rc" "$out"
called "sudo ufw default deny incoming"
called "sudo ufw default allow outgoing"

t "abre o LocalSend (53317, TCP e UDP) e o DNS do Docker para os containers"
called "sudo ufw allow 53317/tcp"
called "sudo ufw allow 53317/udp"
called "sudo ufw allow in proto udp from 172.16.0.0/12 to 172.17.0.1 port 53 comment allow-docker-dns"

t "não abre mais nenhuma porta (nem o SSH)"
eq 3 "$(grep -c '^sudo ufw allow' "$SBX/calls")"
hasnt "$(<"$SBX/calls")" "22"

t "liga o ufw sem perguntar e habilita o serviço"
called "sudo ufw --force enable"
called "sudo systemctl enable --now ufw.service"

t "a política vem antes das portas, e ligar vem por último"
eq "sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow 53317/tcp
sudo ufw allow 53317/udp
sudo ufw allow in proto udp from 172.16.0.0/12 to 172.17.0.1 port 53 comment allow-docker-dns
sudo ufw --force enable
sudo systemctl enable --now ufw.service" "$(<"$SBX/calls")"

t "tudo o que mexe no sistema passa pelo sudo"
eq "" "$(grep -v '^sudo ' "$SBX/calls")"

t "sem o ufw-docker instalado, não tenta rodá-lo"
not_called "ufw-docker"

t "com o ufw-docker instalado, impede o Docker de abrir portas por fora do ufw"
stub ufw-docker
roda
eq 0 "$rc" "$out"
eq "sudo ufw-docker install" "$(tail -1 "$SBX/calls")"
eq 8 "$(wc -l <"$SBX/calls")"

t "pode rodar de novo: as mesmas chamadas (idempotente)"
antes=$(<"$SBX/calls")
roda
eq "$antes" "$(<"$SBX/calls")"

t "se uma regra falha, para em vez de ligar o firewall pela metade"
stub sudo '[[ "$*" == "ufw allow 53317/tcp" ]] && exit 1; exit 0'
roda
fails test "$rc" -eq 0
not_called "ufw --force enable"
not_called "systemctl"
