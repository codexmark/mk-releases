---
camada: "03 Base do sistema"
tags:
  - mk
  - serviços
  - systemd
fontes:
  - "installer/mk-install"
  - "installer/README.md"
---

# Serviços systemd

O instalador habilita uma lista fixa de unidades do systemd, cada uma só se existir no sistema instalado.

## Para quê

Deixar a máquina utilizável no primeiro boot: rede, login, relógio, firewall e os serviços sob demanda.

## Como funciona

| Unidade | Para quê |
|---|---|
| `NetworkManager.service` | [[Rede]] |
| `bluetooth.service` | [[Bluetooth]] |
| `sddm.service` | [[SDDM]] |
| `avahi-daemon.service` | nomes `.local` ([[Rede]]) |
| `power-profiles-daemon.service` | perfis de energia ([[Energia e suspensão]]) |
| `systemd-timesyncd.service` | relógio |
| `ufw.service` | [[Firewall]] |
| `docker.socket` | Docker, sob demanda |
| `cups.socket` | [[Impressão]], sob demanda |
| `fstrim.timer` | TRIM semanal |
| `plocate-updatedb.timer` | índice do `locate` |
| `snapper-cleanup.timer` | apaga snapshots além do limite ([[Snapper e snap-pac]]) |
| `limine-snapper-sync.service` | snapshots no menu de boot ([[Ganchos do limine]]) |

Além dessas, uma unidade **de usuário** habilitada para todos: `gcr-ssh-agent.socket` (`systemctl --global enable`), descrita em [[Chaveiro e agente SSH]].

O laço procura cada unidade em `/usr/lib/systemd/system` do sistema novo antes de habilitar; se o `enable` falha, avisa e segue.

## Por quê assim

- **"Só se existir"**: a mesma lista serve a instalações com e sem os pacotes do AUR (o `limine-snapper-sync` pode ter ficado fora) e a perfis diferentes. Um `enable` de unidade inexistente não derruba a instalação.
- **Soquetes em vez de serviços** para Docker e CUPS: sobem quando alguém usa, não no boot.
- **`sshd` não está na lista**: o `openssh` é instalado para o cliente e o git por ssh; o servidor "fica desligado até você ligar".
- **O que não aparece aqui**: PipeWire e WirePlumber não são habilitados pelo instalador; o repositório não diz como sobem (veja [[Áudio]]).

## Onde está no código

`installer/mk-install`: `configure_services`.

## Ligações

Depende de: [[Listas de pacotes]]. Usado por: [[Fluxo de boot]], [[Rede]], [[Firewall]], [[Impressão]], [[Bluetooth]], [[SDDM]].
