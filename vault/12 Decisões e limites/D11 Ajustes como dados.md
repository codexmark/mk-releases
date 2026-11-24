---
camada: "12 Decisões e limites"
tags:
  - mk
  - decisões
  - maré
  - ajustes
fontes:
  - "shell/mare/SettingsSchema.js"
  - "shell/mare/Settings.qml"
  - "shell/mare/README.md"
---

# D11 Ajustes como dados

As páginas dos Ajustes são descritas num arquivo de dados (`SettingsSchema.js`); o painel desenha todas as linhas do mesmo jeito.

## Contexto

Os Ajustes nasceram num commit só, já neste formato: "folha de configurações inspirada no Hub do Ryoku", com "busca em todas as opções (com sinônimos)" e "linhas desenhadas a partir de `SettingsSchema.js`". Depois ganharam páginas (tela, som, energia, notificações, monitores, atalhos…) sem mudar de desenho. O repositório não descreve um problema anterior que a escolha tenha resolvido; registra a origem da ideia.

## Escolha

"A página é dado, não layout" (a ideia vem do Hub do Ryoku, segundo o comentário do esquema). Uma linha é um objeto com chave, rótulo, tipo de controle, valor de fábrica e, se for do Hyprland, o campo que muda. A partir dela saem, sem código por opção:

- o desenho na página;
- a entrada na busca (com as palavras de `kw`);
- a marca de "mudado" e o "voltar à fábrica";
- o Lua do `ajustes.lua` e o `hyprctl eval`;
- a aceitação da chave pelo `ipc call mare set`.

Páginas que não cabem em linhas (monitores, atalhos, impressoras, VPN, apps, controle) têm um arquivo QML próprio, carregado só enquanto abertas.

## Alternativas

- **Uma página QML por assunto, escrita à mão**: é como ficaram as seis páginas próprias, cujo conteúdo não é uma lista de opções.
- **Só arquivos de config, sem painel** *(não discutida no repositório)*.

## Consequência

- Acrescentar uma opção é uma linha no esquema e uma leitura com `Settings.get`.
- O valor de fábrica mora em dois lugares e precisa bater: o `def` da linha e o que o Mk usa sem `ajustes.json` (por exemplo, `gaps_in = 6` em `look.lua`).
- A lista de animações é repetida no esquema e em `motion.lua` ([[Animações]]); um teste (`tests/config/motion.test.lua`) confere as duas.
- O `set` por IPC aceita qualquer valor para uma chave conhecida: não confere faixa nem opções.
- O que não é uma chave simples (`@animations`, `@kb_options`, `@monitor.scale`) precisa de código próprio no esquema.

## Onde está no código

`shell/mare/SettingsSchema.js` (o cabeçalho e `rows`); `shell/mare/Settings.qml`.

## Ligações

Afeta: [[Ajustes]], [[Páginas dos Ajustes]], [[IPC do Maré]]. Índice: [[Registro de decisões]].
