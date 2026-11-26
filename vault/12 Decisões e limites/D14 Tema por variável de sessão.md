---
camada: "12 Decisões e limites"
tags:
  - mk
  - decisões
  - aparência
  - sessão
fontes:
  - "config/hypr/mare/env.lua"
  - "shell/mare/scripts/mare-look"
  - "system/mk/look/themes/Mk/gtk-4.0/gtk.css"
  - "config/ghostty/config"
---

# D14 Tema por variável de sessão

A aparência dos apps é escolhida por coisas que valem **só dentro da sessão do Mk**: variáveis de ambiente e argumentos de linha de comando. O que é da conta inteira só é escrito com cuidado.

## Contexto

O Mk pode ser uma de várias sessões na mesma conta ([[D01 Maré fora da pasta do Quickshell]]). O jeito comum de aplicar um tema GTK é gravar no `gsettings` ou em `~/.config/gtk-*`, que são da conta: o outro desktop herdaria o tema do Mk, e o Mk desfaria a escolha que a pessoa fez lá.

## Escolha

- **Tema GTK**: `GTK_THEME=Mk`, por `hl.env`.
- **Qt**: `QT_QPA_PLATFORMTHEME=qt6ct`.
- **Cursor**: `XCURSOR_THEME`, `XCURSOR_SIZE`.
- **Terminal**: `ghostty --config-default-files=false --config-file=…`, em vez de `~/.config/ghostty`.
- **O que só existe no `gsettings`** (ícones, cursor, fontes, esquema escuro): o `mare-look` escreve cada chave **só se** ela ainda está no padrão de fábrica ou se foi o próprio Mk que escreveu o valor atual. O que ele escreveu fica anotado em `<raiz>/.look`.

## Alternativas

- **Gravar o tema no `gsettings`/`gtk-*`**: é o que versões antigas faziam (o `mare-look` ainda reconhece um `Adwaita-dark` posto por elas "sem anotar").
- **Não tematizar os apps** *(não discutida no repositório)*.

## Consequência

- Outro desktop na mesma conta não é afetado, e a escolha da pessoa lá não é desfeita.
- No `gsettings` fica `adw-gtk3-dark` como tema GTK; dentro da sessão, a variável `GTK_THEME=Mk` vence.
- A variável só chega aos filhos do Hyprland. Para os apps que nascem pelo D-Bus ou pelo systemd, o `mare-env` a exporta pelo `uwsm finalize`, que a retira no fim da sessão: a decisão ("vale só dentro do Mk") continua de pé.
- O `try-session` instala o tema em `~/.local/share/themes`, sem tocar em `~/.config/gtk-*`.

## Onde está no código

`config/hypr/mare/env.lua`; `shell/mare/scripts/mare-look` (o cabeçalho e a função `apply`); `shell/mare/scripts/mare-env`.

## Ligações

Afeta: [[Variáveis de ambiente]], [[Tema GTK, ícones e Qt]], [[Ghostty]], [[Pasta do usuário]]. Índice: [[Registro de decisões]].
