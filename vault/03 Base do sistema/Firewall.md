---
camada: "03 Base do sistema"
tags:
  - mk
  - rede
  - firewall
  - ufw
  - docker
fontes:
  - "installer/mk-install"
  - "scripts/system/firewall.sh"
  - "installer/README.md"
  - "iso/README.md"
---

# Firewall

O firewall é o `ufw`, com a política "nega toda entrada, libera toda saída" e duas exceções.

## Para quê

Uma máquina de desenvolvimento roda servidores locais e containers o tempo todo; o padrão seguro é que nada disso fique exposto à rede sem uma decisão explícita.

## Como funciona

Regras gravadas pelo instalador (no chroot só se grava; o ufw liga no primeiro boot, com `ENABLED=yes` em `/etc/ufw/ufw.conf` e `ufw.service` habilitado):

- `default deny incoming`, `default allow outgoing`;
- `53317/tcp` e `53317/udp` abertas: o LocalSend (compartilhamento na rede local, `localsend-bin` do AUR);
- UDP 53 de `172.16.0.0/12` para `172.17.0.1`: o DNS do Docker para os containers.

`scripts/system/firewall.sh` grava as mesmas regras num sistema em uso e, se houver `ufw-docker`, roda `ufw-docker install`.

## Por quê assim

- **`ufw-docker`** (AUR): "firewall que o Docker não fura". O Docker escreve no iptables por fora do ufw, então uma porta publicada por um container ficaria aberta apesar do `deny incoming`.
- **O instalador não roda o `ufw-docker install`**: ele instala o pacote, mas o passo que amarra o Docker ao ufw só acontece se o usuário rodar `/usr/share/mk/scripts/system/firewall.sh` depois do primeiro boot. O `installer/README.md` documenta essa lacuna; o motivo de não fazê-lo no chroot não está registrado.
- **Porta 22 fechada**: o `sshd` vem desligado e o ufw nega a 22. Para entrar numa VM de teste por SSH é preciso ligar o `sshd` e abrir a porta ([[VM de teste]]).
- A política segue "o mesmo princípio do `install/config/firewall.sh` do Omarchy" (comentário do script).

## Onde está no código

- `installer/mk-install`: `configure_services` (bloco "Firewall")
- `scripts/system/firewall.sh`

## Ligações

Depende de: [[Rede]], [[Serviços systemd]], [[Pacman e AUR]].
