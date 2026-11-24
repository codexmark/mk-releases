---
camada: "01 Núcleo e boot"
tags:
  - mk
  - boot
  - limine
fontes:
  - "installer/mk-install"
  - "installer/README.md"
  - "system/packages/base.packages"
  - "system/packages/aur.packages"
---

# Limine

O limine é o bootloader do Mk. O instalador copia o binário EFI para a ESP, grava um `limine.conf` com as cores do Mk e deixa os ganchos do limine escreverem as entradas; se eles não estiverem lá, grava entradas fixas.

## Para quê

Iniciar o kernel e, principalmente, oferecer no próprio menu de boot os snapshots do sistema ([[Voltar atrás pelo menu de boot]]).

## Como funciona

1. **Binário.** `/usr/share/limine/BOOTX64.EFI` é copiado para `EFI/limine/BOOTX64.EFI` e para o caminho padrão `EFI/BOOT/BOOTX64.EFI`, na ESP montada em `/boot`.
2. **Cabeçalho do `limine.conf`** (`limine_header`): `timeout: 3`, `interface_branding: Mk` e as cores da [[Direção Folha]] (`0b0b0b`, `cdc4ba`, `ce5e32`).
3. **`/etc/default/limine`**, lido pelo `limine-mkinitcpio-hook`: nome "Mk", ESP em `/boot`, a linha do kernel, `ENABLE_LIMINE_FALLBACK=yes`, `FIND_BOOTLOADERS=no`, `BOOT_ORDER="*, *fallback, Snapshots"` e `MAX_SNAPSHOT_ENTRIES=auto`.
4. **Entradas.** Depois do `mkinitcpio`, se existe `limine-update`, ele escreve as entradas. O instalador então confere (`limine_conf_ok`) que o arquivo ativo tem pelo menos uma entrada `protocol: linux` e que cada `path`/`module_path` aponta para um arquivo que existe em `/boot`.
5. **Rede de segurança.** Se não há `limine-update`, se ele falha ou se o resultado não fecha, o instalador grava entradas fixas: `/Mk` e, havendo o initramfs de fallback, `/Mk (fallback)`. Se nem assim o arquivo aponta para um kernel que existe, a instalação **para**.
6. Por fim roda o [[limine-default]] e cria a [[Entrada UEFI única]].

O arquivo ativo é `EFI/limine/limine.conf` se existir, senão `limine.conf` na raiz da ESP (`active_limine_conf`).

## Por quê assim

- **Cópia no caminho padrão**: "firmware sem entrada na NVRAM ainda acha o Mk".
- **`#resumo` no fim do caminho.** Com a verificação ligada, o `limine-entry-tool` põe `#<resumo>` depois do nome do arquivo. A primeira versão da conferência tratava isso como parte do nome, concluía que o menu não era iniciável e descartava o menu dos ganchos (issue da primeira instalação em máquina real). Hoje o `#…` é cortado antes de conferir. Veja [[D09 Menu de boot dos ganchos]].
- **Exigir `protocol: linux`**: a entrada "EFI fallback" só carrega o limine de novo; sozinha, não inicia sistema nenhum. A migração do boot já fazia a mesma exigência.
- **`MAX_SNAPSHOT_ENTRIES=auto`**: o número fixo (5) gerava um aviso a cada sincronização; `auto` poda os mais antigos pelo espaço da ESP.
- **`FIND_BOOTLOADERS=no`**: o instalador usa o disco inteiro, não há outro sistema para achar.
- A escolha do limine em vez de GRUB ou systemd-boot: **motivo não registrado**. O que o código mostra é o uso que se faz dele: o submenu de snapshots pelo `limine-snapper-sync`.

## Onde está no código

- `installer/mk-install`: `configure_boot`, `limine_header`, `limine_entry`, `active_limine_conf`, `limine_conf_ok`, `finish_boot`
- `system/packages/base.packages` (`limine`, `efibootmgr`), `aur.packages` (os ganchos)

## Ligações

Depende de: [[Particionamento]], [[Kernel e linha de comando]], [[Initramfs]]. Usado por: [[Ganchos do limine]], [[limine-default]], [[Entrada UEFI única]], [[Voltar atrás pelo menu de boot]].
