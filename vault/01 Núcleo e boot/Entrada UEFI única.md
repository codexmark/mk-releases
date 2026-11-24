---
camada: "01 Núcleo e boot"
tags:
  - mk
  - boot
  - uefi
fontes:
  - "installer/mk-install"
  - "installer/README.md"
---

# Entrada UEFI única

O instalador deixa uma só entrada de boot na NVRAM do firmware, chamada "Mk", e apaga as que competiriam com ela.

## Para quê

Para o menu do firmware mostrar um item só com o nome do sistema, apontando para o binário do limine que continua sendo atualizado.

## Como funciona

No fim de `finish_boot`, se `/sys/firmware/efi/efivars` existe:

1. Escolhe o *loader*: `\EFI\limine\limine_x64.efi` se os ganchos do limine o instalaram (e então apaga a cópia `EFI/limine/BOOTX64.EFI`); senão `\EFI\limine\BOOTX64.EFI`.
2. Lê a saída do `efibootmgr` e remove a entrada "Limine" que o gancho registrou para `limine_x64.efi` e qualquer entrada "Mk" de uma instalação anterior.
3. Cria a entrada: `efibootmgr --create --disk <disco> --part 1 --label Mk --loader <loader>`.
4. Se a criação falha, só avisa: o firmware ainda acha `EFI/BOOT/BOOTX64.EFI`.

## Por quê assim

- Com os ganchos, quem cuida do binário é o `limine-install`, que mantém `limine_x64.efi` atualizado a cada versão. A entrada "Mk" aponta para **esse** arquivo; o `limine-install` não recria a entrada dele "enquanto houver entrada para esse caminho".
- Uma entrada só, e não "Limine" + "Mk" + restos: veio das issues da primeira instalação em máquina real.
- O firmware pode criar a própria entrada ("UEFI OS") para o caminho padrão; isso não é do instalador e está registrado em [[Limitações conhecidas]].

> [!note] UEFI obrigatório
> O instalador para logo no começo se `/sys/firmware/efi` não existe. Não há instalação em BIOS legado.

## Onde está no código

`installer/mk-install`: `preflight` (a exigência de UEFI) e `finish_boot` (o bloco do `efibootmgr`).

## Ligações

Depende de: [[Limine]], [[Ganchos do limine]], [[Particionamento]]. Parte do [[Fluxo de boot]].
