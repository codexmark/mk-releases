---
camada: "03 Base do sistema"
tags:
  - mk
  - região
  - locale
  - teclado
fontes:
  - "installer/mk-install"
  - "installer/README.md"
  - "config/hypr/mare/input.lua"
  - "iso/profile/airootfs/etc/vconsole.conf"
  - "system/mk/sddm/mk/theme.conf"
---

# Locale, fuso e teclado

A região tem três valores (fuso, teclado, idioma) com padrões brasileiros, e o teclado precisa ser gravado em quatro lugares diferentes para valer do console à sessão.

## Para quê

Para o sistema falar português e o teclado ABNT2 funcionar no console, na tela de login e dentro do Hyprland.

## Como funciona

A tela "Região" propõe um fuso, `br-abnt2` e `pt_BR.UTF-8`. Recusando, escolhe-se cada um numa lista com busca (`timedatectl list-timezones`, `localectl list-keymaps`, `/usr/share/i18n/SUPPORTED`).

O que o instalador grava (`configure_base`):

- `/etc/localtime` (link para o fuso) e `hwclock --systohc`;
- `locale.gen` com o idioma escolhido **e** `en_US.UTF-8`; `LANG` em `/etc/locale.conf`;
- `KEYMAP` em `/etc/vconsole.conf` (console; o hook `sd-vconsole` leva ao initramfs);
- `/etc/X11/xorg.conf.d/00-keyboard.conf` com `XkbLayout` = a parte antes do primeiro `-` do keymap (`br-abnt2` → `br`);
- `layout=` em `theme.conf.user` do tema do [[SDDM]].

Dentro da sessão, o layout é do Hyprland: `kb_layout = "br"` em `config/hypr/mare/input.lua`, trocável em Ajustes › Teclado e mouse (`br`, `us`, ou os dois com Alt+Shift).

## Por quê assim

- **O layout no tema do SDDM**: "a lista do próprio sddm diz sempre 'us'", mesmo com o X em outro layout. O tema passa a mostrar o do sistema.
- **`en_US.UTF-8` sempre gerado**: o código não comenta; **motivo não registrado**.
- **`pt_BR` tem um efeito colateral**: com apps aceitos, entra o `libreoffice-fresh-pt-br`, que não está em lista nenhuma.
- **O fuso de fábrica** é um valor fixo em `interview()`. **Motivo não registrado.**
- **O layout da sessão não vem do instalador**: o `input.lua` é fixo em `br`. Quem instala com outro teclado precisa trocar nos Ajustes (o código não propaga a escolha do instalador para o Hyprland).

> [!note] Modo desassistido
> No `--answers` não há padrões de região: os três valores têm de vir no arquivo. Um valor vazio ou que não existe no live para o instalador antes de tocar no disco ([[Modo desassistido]]).

## Onde está no código

- `installer/mk-install`: `interview` (seção 03), `validate`, `configure_base`, `install_desktop` (o `theme.conf.user`)
- `config/hypr/mare/input.lua`

## Ligações

Usado por: [[SDDM]], [[Initramfs]], [[Hyprland em Lua]], [[Páginas dos Ajustes]].
