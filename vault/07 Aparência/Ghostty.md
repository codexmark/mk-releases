---
camada: "07 Aparência"
tags:
  - mk
  - aparência
  - terminal
  - ghostty
fontes:
  - "config/ghostty/config"
  - "shell/mare/Projects.qml"
  - "config/hypr/mare/binds.lua"
  - "config/hypr/mare/paths.lua"
---

# Ghostty

O terminal do Mk é o Ghostty, com uma config própria carregada **só pela sessão do Mk**: fundo quase preto e quente, texto em osso, cursor e seleção de cobre.

## Para quê

Um terminal que combina com o resto e que não mexe no Ghostty de outros desktops na mesma conta.

## Como funciona

- **Config**: `config/ghostty/config`, instalada em `<raiz>/ghostty/config`.
- **Como é carregada**: todo terminal que o Maré abre usa

  ```
  ghostty --config-default-files=false --config-file=?<raiz>/ghostty/config
  ```

  O `?` deixa a config opcional; `--config-default-files=false` ignora o `~/.config/ghostty`.
- **Fontes**: SpaceMono Nerd Font, com Geist Mono e JetBrains Mono de reserva, tamanho 8, altura de célula +12%.
- **Cores**: fundo `#0f0e0d`, texto `#cdc4ba`, cursor em barra `#ce5e32` sem piscar, seleção `#3a2a22`. A paleta de 16 cores é "quente e contida": o vermelho é o cobre e as outras são versões apagadas.
- **Janela**: respiro de 16 x 14, sem decoração e sem barra de título ("a moldura é do Hyprland").
- **Comportamento**: `copy-on-select = clipboard`, sem confirmação ao fechar, mouse some ao digitar.

## Por quê assim

- **Config por argumento, não em `~/.config/ghostty`**: "o `~/.config/ghostty` de outros desktops fica como está". É o mesmo princípio de [[Pasta do usuário]].
- **Tamanho 8**: a versão com Space Mono 11 "parecia lupa ao lado do resto".
- **Paleta apagada**: "para o terminal não brigar com a interface".
- **O atalho de emergência usa a mesma linha**, mas com reserva: `ghostty … || kitty || foot || alacritty` ([[Atalhos]]).
- **A escolha de outro terminal** (kitty, foot, alacritty, wezterm) fica em Ajustes › Apps padrão; só o ghostty leva a config do Mk ([[Projetos e workspaces]]).

## Onde está no código

`config/ghostty/config`; `shell/mare/Projects.qml` (`terminalLineFor`); `config/hypr/mare/binds.lua` (`emergency.terminal`).

## Ligações

Depende de: [[Direção Folha]], [[Fontes]], [[Pasta do usuário]]. Usado por: [[Terminal e bash do Mk]], [[Projetos e workspaces]], [[Fastfetch]].
