---
camada: "01 Núcleo e boot"
tags:
  - mk
  - boot
  - limine
  - aur
  - snapshots
fontes:
  - "system/packages/aur.packages"
  - "installer/mk-install"
  - "installer/README.md"
  - "docs/atualizacoes.md"
  - "iso/README.md"
---

# Ganchos do limine

Dois pacotes do AUR, `limine-mkinitcpio-hook` e `limine-snapper-sync`, mantêm o menu de boot em dia: o primeiro regenera as entradas a cada kernel novo, o segundo acrescenta o submenu de snapshots.

## Para quê

- `limine-mkinitcpio-hook`: lê `/etc/default/limine` e reescreve o `limine.conf` (comando `limine-update`) sempre que o initramfs é refeito. Sem ele, o menu fica com as entradas fixas e não se atualiza sozinho.
- `limine-snapper-sync`: lista os snapshots do snapper no menu e os inicia. Sem ele, não há o submenu "Snapshots".

## Como funciona

- Vêm compilados na ISO ([[AUR na instalação]]) e são instalados na etapa de AUR do instalador.
- O instalador habilita `limine-snapper-sync.service`, se a unidade existir.
- A configuração é a de `/etc/default/limine` ([[Limine]]).
- Se algum dos dois não compilou no build da ISO, o nome vai para `/usr/share/mk/aur-pendentes.txt` e o `mk-update` oferece instalar pelo `yay`; depois roda o [[limine-default]], "porque os ganchos do limine reescrevem o menu de boot".

## Por quê assim

- São do AUR porque não existem nos repositórios oficiais (regra das listas: AUR é último recurso; veja [[Listas de pacotes]]).
- O caso comum de falha é a falta do `gradle` na máquina que gera a ISO: os dois precisam dele para compilar. Por isso o `iso/build` reaproveita o pacote já compilado na mesma versão ([[ISO]]).
- O sistema continua iniciando sem eles: o instalador grava entradas fixas. Perde-se o submenu de snapshots e a atualização automática do menu.

> [!warning] Dependência frágil
> O `yay` também é do AUR. Se ele próprio ficou de fora da ISO, a oferta do `mk-update` não aparece, e os pacotes precisam ser instalados à mão.

## Onde está no código

- `system/packages/aur.packages`
- `installer/mk-install`: `install_aur`, `finish_boot`, `configure_services`
- `bin/mk-update` (o bloco "AUR que ficou de fora da instalação")

## Ligações

Depende de: [[Limine]], [[Snapper e snap-pac]], [[Pacman e AUR]]. Usado por: [[Voltar atrás pelo menu de boot]], [[limine-default]]. Decisão: [[D09 Menu de boot dos ganchos]].
