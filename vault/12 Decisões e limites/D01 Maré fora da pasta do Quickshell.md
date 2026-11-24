---
camada: "12 Decisões e limites"
tags:
  - mk
  - decisões
  - sessão
  - maré
fontes:
  - "config/hypr/mare/paths.lua"
  - "config/hypr/mare/autostart.lua"
  - "bin/mk-session"
  - "bin/try-session"
  - "bin/new-user"
  - "shell/mare/README.md"
---

# D01 Maré fora da pasta do Quickshell

O Mk inteiro mora em `~/.config/mk` e é achado por caminho; nada dele fica em `~/.config/hypr` nem em `~/.config/quickshell`.

## Contexto

A primeira versão copiava o shell para `~/.config/quickshell/mare`, o lugar convencional de uma config do Quickshell. No primeiro login de um usuário novo, numa máquina que também tinha outro desktop baseado em Quickshell, esse outro desktop **recriava** `~/.config/quickshell` e apagava o que não reconhecia. A sessão do Mk ficava sem shell: tela preta, sem como sair.

## Escolha

- O shell fica ao lado da config do Hyprland, em `<raiz>/shell`, e é iniciado e chamado **pelo caminho**: `qs -p <pasta>`.
- O Hyprland recebe a config por `-c <raiz>/hypr/hyprland.lua`.
- `paths.lua` descobre a raiz a partir do próprio arquivo; o autostart, os binds e o hypridle usam esses caminhos.
- No mesmo commit entraram os dois atalhos de emergência (terminal e sair) que não dependem do shell, o log em `~/.cache/…/mare.log` e o `bin/diagnose-user`.

## Alternativas

- **Ficar em `~/.config/quickshell/mare`** (`qs -c mare`): descartada pelo defeito acima. A pasta "pode pertencer a outro desktop, que a recria e apaga o que não reconhece".
- **Usar `~/.config/hypr`**: descartada pela mesma razão; é a config do desktop que a pessoa já tem.

## Consequência

- O Mk convive com outro desktop na mesma conta, como uma sessão separada no login ([[Experimentar sem instalar]]).
- Toda chamada de IPC leva o caminho: `qs -p ~/.config/mk/shell ipc call mare …`. O [[Comando mk]] existe em parte para esconder isso.
- O mesmo princípio foi estendido: o Ghostty por `--config-file`, o tema GTK por variável ([[D14 Tema por variável de sessão]]).
- Os scripts `try-session` e `new-user` ainda removem a pasta antiga em `~/.config/quickshell/mare`, se tiver a marca.

## Onde está no código

`config/hypr/mare/paths.lua` (o comentário), `config/hypr/mare/autostart.lua`, `bin/mk-session`.

## Ligações

Afeta: [[Pasta do usuário]], [[mk-session e uwsm]], [[Autostart]], [[IPC do Maré]], [[Maré]]. Índice: [[Registro de decisões]].
