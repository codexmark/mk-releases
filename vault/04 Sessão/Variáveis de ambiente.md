---
camada: "04 Sessão"
tags:
  - mk
  - sessão
  - ambiente
fontes:
  - "config/hypr/mare/env.lua"
  - "shell/mare/scripts/mare-start"
  - "shell/mare/scripts/mare-look"
---

# Variáveis de ambiente

As variáveis de ambiente da sessão são definidas em Lua, em `config/hypr/mare/env.lua`, e valem só dentro do Mk.

## Para quê

Escolher o tema dos apps, o cursor, a plataforma do Qt e do Electron, o terminal e o agente SSH, sem gravar nada na conta que outro desktop leria.

## Como funciona

| Variável | Valor | Efeito |
|---|---|---|
| `GTK_THEME` | `Mk` | os apps GTK usam o tema do Mk ([[Tema GTK, ícones e Qt]]) |
| `XCURSOR_THEME`, `XCURSOR_SIZE`, `HYPRCURSOR_SIZE` | `Adwaita`, `24`, `24` | cursor |
| `QT_QPA_PLATFORMTHEME` | `qt6ct` | os apps Qt leem as cores do `qt6ct` |
| `QT_QPA_PLATFORM` | `wayland;xcb` | Qt em Wayland, com X de reserva |
| `ELECTRON_OZONE_PLATFORM_HINT` | `auto` | Electron em Wayland quando dá |
| `TERMINAL` | `ghostty` | terminal padrão |
| `SSH_AUTH_SOCK` | `$XDG_RUNTIME_DIR/gcr/ssh` | só se o soquete do `gcr-ssh-agent` está ligado para todos ([[Chaveiro e agente SSH]]) |

Fora do `env.lua`, uma variável é posta por processo: `QT_SCALE_FACTOR`, que o `mare-start` define só para o Maré, com o valor de `ui.scale` ([[Maré]]).

## Por quê assim

- **Por variável de sessão, não por arquivo de config**: "por variável de ambiente, vale só dentro do Mk". Outro desktop na mesma conta não é afetado. O que precisa ir para o `gsettings` (ícones, cursor, fontes), que é da conta inteira, passa pelo `mare-look`, que só escreve uma chave se ela ainda está no padrão ou se foi o próprio Mk que a escreveu ([[D14 Tema por variável de sessão]]).
- **`SSH_AUTH_SOCK` condicional**: o `env.lua` abre `/etc/systemd/user/sockets.target.wants/gcr-ssh-agent.socket`; só aponta a variável se o arquivo existe. Assim uma sessão de teste (`try-session`) num Arch sem o agente não fica com um soquete morto.
- **`QT_SCALE_FACTOR` só no Maré**: se fosse da sessão, todos os apps Qt encolheriam junto.

## Onde está no código

`config/hypr/mare/env.lua`; `shell/mare/scripts/mare-start`; `shell/mare/scripts/mare-look`.

## Ligações

Depende de: [[mk-session e uwsm]]. Usado por: [[Tema GTK, ícones e Qt]], [[Chaveiro e agente SSH]], [[Ghostty]], [[Ordem de carga]].
