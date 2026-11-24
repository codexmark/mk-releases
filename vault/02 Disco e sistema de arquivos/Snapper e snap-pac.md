---
camada: "02 Disco e sistema de arquivos"
tags:
  - mk
  - disco
  - snapshots
  - snapper
fontes:
  - "installer/mk-install"
  - "installer/README.md"
  - "docs/atualizacoes.md"
  - "system/packages/base.packages"
---

# Snapper e snap-pac

O `snapper` guarda snapshots do subvolume `@`, e o `snap-pac` cria um snapshot antes e outro depois de cada transação do `pacman`.

## Para quê

Ter um ponto de volta automático a cada atualização do sistema, sem o usuário precisar lembrar.

## Como funciona

- O instalador roda `snapper --no-dbus -c root create-config /` no chroot e ajusta `/etc/snapper/configs/root`:
  - `TIMELINE_CREATE="no"`: sem snapshots por hora;
  - `NUMBER_LIMIT="5"` e `NUMBER_LIMIT_IMPORTANT="5"`: no máximo 5.
- `snapper-cleanup.timer` é habilitado: apaga os que passam do limite.
- `snap-pac` está em `base.packages`: é um gancho do pacman, não precisa de configuração do Mk.
- Se o `create-config` falha, a instalação segue e avisa como configurar depois (`sudo snapper -c root create-config /`).

Para ver o que existe:

```bash
sudo snapper -c root list
```

## Por quê assim

- **Snapshot por transação, não por relógio**: a etapa do instalador se chama "snapshots (snapper: um a cada atualização)". O que quebra um sistema rolling é a atualização; é nela que o ponto de volta importa.
- **Limite de 5**: o valor exato não tem motivo registrado.
- **A troca do próprio Mk não cria snapshot.** O passo que troca `/usr/share/mk` não é uma transação do pacman. O snapshot "antes" é o do `pacman` que vem depois, já com o Mk novo em disco; uma rodada que só troca o Mk não cria snapshot nenhum. Veja [[mk-update]].
- Como só `@` é coberto, a home fica de fora ([[Btrfs e subvolumes]]).

## Onde está no código

- `installer/mk-install`: `configure_snapper`, `configure_services`
- `system/packages/base.packages` (`snapper`, `snap-pac`, `btrfs-progs`)

## Ligações

Depende de: [[Btrfs e subvolumes]], [[Pacman e AUR]]. Usado por: [[Ganchos do limine]], [[Voltar atrás pelo menu de boot]], [[mk-update]].
