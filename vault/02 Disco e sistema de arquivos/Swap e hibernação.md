---
camada: "02 Disco e sistema de arquivos"
tags:
  - mk
  - disco
  - swap
  - energia
fontes:
  - "installer/mk-install"
  - "installer/README.md"
  - "bin/mk-hibernate"
  - "shell/mare/scripts/mare-sleep"
  - "shell/mare/SettingsSchema.js"
  - "docs/limites.md"
---

# Swap e hibernação

O Mk instalado não tem swap: nem partição, nem arquivo, nem zram. A hibernação, que chegou a existir com um swapfile, foi retirada.

## Para quê

Esta nota registra uma ausência, porque ela surpreende: quem procura `zram` ou `resume=` não vai achar.

## Como funciona

- O instalador não cria swap. O `installer/README.md` diz: "Não há swap em partição nem em arquivo".
- **zram**: nenhum pacote das listas o configura. A única menção no repositório é o instalador ignorar `/dev/zram*` na lista de discos (o live do Arch tem um).
- **Hibernação**: por um curto período, `bin/mk-hibernate setup` criava um subvolume `/swap` com um swapfile, uma linha no `fstab` e `resume=` na linha do kernel. Hoje:
  - `mk-hibernate setup` recusa: "a hibernação está desativada no Mk";
  - `mk-hibernate remove` desfaz exatamente o que o setup anotou em `/var/lib/mk/hibernate/estado` (swap, swapfile, subvolume, linha do `fstab`, `resume=`) e roda `limine-update` e o [[limine-default]];
  - `mk-hibernate status` mostra como está, sem root;
  - a opção "hibernar" saiu de Ajustes › Energia, do painel de sessão e da tela de bloqueio;
  - um `power.sleep: "hibernar"` antigo no `ajustes.json` passa a valer como "não dormir" (`sleepMode` em `SettingsSchema.js`; `mare-sleep now` também não faz nada com ele).

## Por quê assim

**Por que a hibernação saiu.** Na VM a sessão era gravada e não voltava no boot seguinte (`systemd-hibernate-resume: Unable to resume from device`), e a causa não foi achada. O Mk passou a só suspender ou não dormir. O `remove` ficou para quem já tinha preparado a máquina poder desfazer. Detalhes em [[D05 Hibernação retirada]].

**Por que um "hibernar" antigo não vira suspender sozinho.** O código só afirma a regra ("não vira suspender sozinho"); o motivo não está registrado. O efeito é que a escolha antiga cai em "não dormir".

**Por que não há swap nem zram.** **Motivo não registrado.** O código apenas não os cria.

> [!note] mk-hibernate ainda é instalado
> O `mk-update` instala `/usr/bin/mk-hibernate` quando o arquivo existe no lançamento, para o `remove` estar à mão. O instalador não o instala.

## Onde está no código

- `bin/mk-hibernate` (o cabeçalho conta a história)
- `shell/mare/SettingsSchema.js`: `sleepMode`, `sleepCommands`, `sleepStep`
- `shell/mare/scripts/mare-sleep`

## Ligações

Depende de: [[Btrfs e subvolumes]], [[Kernel e linha de comando]]. Usado por: [[Energia e suspensão]]. Decisão: [[D05 Hibernação retirada]].
