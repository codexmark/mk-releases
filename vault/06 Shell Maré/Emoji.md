---
camada: "06 Shell Maré"
tags:
  - mk
  - maré
  - emoji
fontes:
  - "shell/mare/EmojiPanel.qml"
  - "shell/mare/Emoji.js"
  - "shell/mare/scripts/mare-emoji"
  - "migrations/system/20261125-emoji-wtype.sh"
  - "system/packages/desktop.packages"
---

# Emoji

O painel de emoji (`Super + .`) procura em português ou inglês e, ao escolher, copia o emoji e o digita na janela que estava em foco.

## Para quê

Inserir um emoji em qualquer app sem depender do seletor de cada um.

## Como funciona

- **Dados**: `Emoji.js`, um arquivo gerado (cerca de 2 mil linhas) com os emojis, nomes e palavras de busca.
- **Uso**: digite para procurar; as setas andam pela grade; `Enter` escolhe; `Tab` troca de categoria; `Ctrl + T` troca o tom de pele; `Esc` fecha. `Shift + Enter` (ou o botão direito) só copia.
- **`mare-emoji pick <emoji> <colar|copiar>`**: copia e avisa; com "colar", depois que o painel fecha e o teclado volta para a janela de antes, digita o emoji nela com o `wtype`.
- **Recentes e tom**: `emoji.recent` e `emoji.tone` no `ajustes.json` (chaves `hidden` do esquema).
- **`mare-emoji gerar`** refaz o `Emoji.js`; é para quem mantém o Mk.

## Por quê assim

- **`gerar` não baixa nada**: os nomes e palavras em português vêm do CLDR que o GTK 3 já instala; a ordem e as formas completas vêm do `emoji-test.txt` do Unicode. Só entra o que a Noto Color Emoji instalada desenha, então o painel nunca oferece um emoji que apareceria como quadrado.
- **`wtype` é opcional**: sem ele o painel só copia. A migração `20261125-emoji-wtype.sh` pede o pacote e, se ele não veio, avisa e segue.
- **Digitar depois de fechar**: enquanto o painel tem o teclado exclusivo, o que fosse digitado iria para ele mesmo.
- **Os recentes são lidos uma vez, ao abrir**: ligados ao `Settings`, trocar o tom (que regrava o `ajustes.json`) refazia a lista e levava o cursor de volta ao primeiro.

## Onde está no código

`shell/mare/EmojiPanel.qml`, `shell/mare/Emoji.js`, `shell/mare/scripts/mare-emoji`.

## Ligações

Depende de: [[Painel líquido (Tide)]], [[Clique fora e teclado exclusivo]], [[Fontes]]. Parte de: [[Painéis]].
