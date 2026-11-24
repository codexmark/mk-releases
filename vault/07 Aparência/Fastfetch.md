---
camada: "07 Aparência"
tags:
  - mk
  - aparência
  - terminal
  - fastfetch
fontes:
  - "config/fastfetch/config.jsonc"
  - "config/bash/mk.bash"
  - "migrations/user/20261121-ficha-compacta.sh"
---

# Fastfetch

A "ficha" é o que o terminal mostra ao abrir: só texto, encostado à esquerda: quem e onde, a linha do Mk aberta por duas setas de cobre (`❯❯`), três seções (máquina, sistema, sessão), cada uma com dois traços, o título e a régua, e a paleta Folha.

## Para quê

Identidade e um resumo útil da máquina na primeira linha de cada janela de terminal.

## Como funciona

- **Config**: `config/fastfetch/config.jsonc`, com o `mark.png` ao lado (instalados em `<raiz>/fastfetch/`).
- **Sem imagem** (`"logo": { "type": "none" }`): a marca à esquerda saiu; a ficha é só texto. A máquina diz a frequência do processador, o tipo da placa, o sistema de arquivos, e os pacotes vêm por gerenciador.
- **Módulos**: uma linha `usuário @ máquina`; a linha do carimbo com a data; `MÁQUINA` (CPU, vídeo, memória, disco); `SISTEMA` (sistema, kernel, janelas, pacotes); `SESSÃO` (ligado há quanto tempo, terminal); os pontos da paleta.
- **Quando aparece** (`mk.bash`): só se o `fastfetch` existe, o terminal é `xterm-ghostty` ou `xterm-kitty`, `SHLVL <= 1`, e não se está dentro do tmux nem do neovim.
- **Desligar**: `MK_NO_FETCH=1` (por exemplo no `seu.bash`).
- **De novo**: `mk ficha`.

## Por quê assim

- **Só em ghostty e kitty**: a condição vem de quando a ficha tinha imagem pelo protocolo do kitty; hoje, só com texto, ela poderia valer em qualquer terminal.
- **Uma vez por janela, nunca dentro do tmux ou do neovim**: um shell aninhado não repete a ficha.
- **Linha própria em vez do módulo `title`**: o `title` do fastfetch sumia quando a imagem estava ligada, em algumas versões (visto na VM).
- **Formato compacto**: a primeira versão tinha um desenho grande; a ficha "compacta, com a marca nova" veio depois, e uma migração de usuário cobriu quem tinha ficado com a pasta antiga.
- O nome do sistema que a ficha mostra vem do [[Nome do sistema (os-release)]].

## Onde está no código

`config/fastfetch/config.jsonc`; `config/bash/mk.bash` (`mk-fetch` e a condição).

## Ligações

Depende de: [[Ghostty]], [[Direção Folha]], [[Arte e geradores]]. Usado por: [[Terminal e bash do Mk]], [[Comando mk]].
