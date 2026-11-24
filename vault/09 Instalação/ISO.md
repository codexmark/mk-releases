---
camada: "09 Instalação"
tags:
  - mk
  - instalação
  - iso
  - archiso
fontes:
  - "iso/build"
  - "iso/README.md"
  - "iso/profile/profiledef.sh"
  - "iso/profile/packages.mk"
  - "iso/profile/efiboot/loader/entries/01-mk.conf"
  - "iso/profile/airootfs/etc/motd"
---

# ISO

A ISO do Mk é um live do archiso, baseado no perfil `releng`, que inicia num console com o instalador `mk-install` à mão. Quem a gera é `iso/build`.

## Para quê

Levar o Mk a uma máquina vazia: o live traz o repositório inteiro, o AUR já compilado e o instalador.

## Como funciona

`iso/build` roda como o usuário, **sem sudo**, em três passos:

1. **AUR.** Lê `system/packages/aur.packages`, baixa cada PKGBUILD (o nome vira `PackageBase` pela API do AUR), compila com `makepkg` e monta o repositório local `mk-aur` em `iso/work/aur/repo`. Dependências que também só existem no AUR entram na fila sozinhas. A lista do que entrou vai em `packages.txt`; o relatório, em `iso/work/aur/report.txt`. Detalhes em [[AUR na instalação]].
2. **Perfil.** Copia `/usr/share/archiso/configs/releng` para `iso/work/profile` e põe por cima `iso/profile/`:
   - `profiledef.sh`: nome `mk`, rótulo `MK`, modos de boot `bios.syslinux` e `uefi.systemd-boot`;
   - `packages.mk`: `gum` (as telas do instalador) e `pciutils` (o `lspci`), somados ao `packages.x86_64` do releng;
   - no live: teclado `br-abnt2`, hostname `mk-live`, o `motd` que aponta para `mk-install`;
   - menus de boot renomeados para "Mk"; no UEFI, as entradas do releng saem e entra `01-mk.conf`;
   - o repositório inteiro (sem `.git`, `iso/work`, `iso/out`) em `/usr/share/mk`; o `aur-repo` ao lado; o `VERSION` (`commit data`); o instalador em `/usr/local/bin/mk-install`; o gancho do os-release.
3. **ISO.** `sudo mkarchiso -v -r -w iso/work/mkarchiso -o iso/out iso/work/profile`.

Saída: `iso/out/mk-AAAA.MM.DD-x86_64.iso` (cerca de 1,4 GB).

Opções: `--prepare` (só os passos 1 e 2, sem sudo, e imprime o comando do `mkarchiso`), `--syncdeps` (deixa o `makepkg` instalar e remover as dependências de compilação que faltam).

## Por quê assim

- **Sem sudo até o último passo**: o `makepkg` não roda como root; o script recusa `EUID 0` e só o `mkarchiso` pede a senha ([[Convenções]], "root só onde precisa").
- **Em cima do `releng`, não um perfil do zero**: `iso/profile/` tem só o que muda; o resto do releng é copiado como está a cada build.
- **O live não é o desktop**: abre num console. O ISO não traz o Hyprland rodando; traz o que é preciso para instalar.
- **Sem a entrada de acessibilidade por voz** do Arch no menu UEFI: o build remove `02-archiso-speech-linux.conf`. O motivo não está registrado.
- **`archiso` não está em lista nenhuma do Mk**: "só serve para gerar o ISO, não para usar o Mk".
- **Aviso de nome antigo**: o build avisa se ainda houver `meu-os` (o nome anterior do projeto) em `config/`, `shell/` ou `bin/mk-session`.
- **A ISO inicia em BIOS legado** (o releng traz o syslinux), mas o instalador para se o live não estiver em UEFI.

> [!warning] Leia o relatório antes de gravar
> Um `FALHOU` em `iso/work/aur/report.txt` é um pacote que o Mk instalado não vai ter. O caso comum são os ganchos do limine, que precisam do `gradle` para compilar.

## Onde está no código

`iso/build` (o cabeçalho é o `--help`), `iso/profile/`, `iso/README.md`.

## Ligações

Depende de: [[Listas de pacotes]], [[Pacman e AUR]], [[Nome do sistema (os-release)]]. Usado por: [[Instalador passo a passo]], [[VM de teste]]. Relacionado: [[AUR na instalação]].
