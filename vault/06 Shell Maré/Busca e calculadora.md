---
camada: "06 Shell Maré"
tags:
  - mk
  - maré
  - busca
  - launcher
fontes:
  - "tests/js/Calc.test.mjs"
  - "shell/mare/LauncherPanel.qml"
  - "shell/mare/Calc.js"
  - "shell/mare/Tide.qml"
  - "config/hypr/mare/binds.lua"
---

# Busca e calculadora

A busca é o launcher do Mk: um toque no `Super` abre um campo que procura, numa lista só, comandos do projeto, painéis do Maré, projetos de `~/Work` e aplicativos. Uma conta digitada ali mostra o resultado na primeira linha.

## Para quê

Um ponto de entrada só para "abrir alguma coisa", sem precisar lembrar se é um app, um painel ou um projeto.

## Como funciona

Fontes da lista, concatenadas nesta ordem:

1. **Comandos do projeto**: `lazygit`, `lazydocker`, `terminal`, no projeto em foco (ou em `~`).
2. **Painéis do Maré que não têm botão na ilha**: ajustes, capturar a tela, papel de parede, área de transferência, emoji, sistema, tela, rede, bluetooth, sessão, boas-vindas.
3. **Projetos**: os repositórios de `~/Work`; Enter vai ao workspace dele ou abre num novo.
4. **Aplicativos**: `DesktopEntries.applications`, menos os `noDisplay`.

- 7 linhas visíveis; `↑/↓` escolhe, `Enter` abre, `Esc` fecha.
- A pontuação favorece, nesta ordem: prefixo, início de palavra, trecho, letras na ordem. O detalhe (`detail`) vale metade do título.
- **Calculadora** (`Calc.js`): se o texto é uma conta, a primeira linha vira `= resultado` e Enter copia (`wl-copy`). Aceita números com vírgula ou ponto, `+ - * x / ^ %`, parênteses, `raiz`/`sqrt` e `pi`. Exemplos do próprio arquivo: `12,5 * 3`, `2^10`, `(1200 - 15%) / 4`, `raiz(81)`.
  - **Só responde quando o texto é claramente uma conta**; na dúvida, `null`. As regras estão no cabeçalho do `Calc.js`: a potência prende mais que o sinal e associa à direita (`-2^2` = −4, `2^3^2` = 2^9); espaços valem só entre um termo e outro e nunca juntam nada (`1 2 + 3` não é conta); zero à esquerda (`007`) não é número de conta; o texto precisa fazer alguma operação (`42`, `-5` e `pi` não são conta; `15%` é); uma data inteira (`2024-11-15`, `15/11/2024`) e um hexadecimal (`0x10`) não são conta; resultado que não é número finito (`1/0`) é `null`.
  - Limite conhecido: `1.000 + 500` dá 501, porque o ponto é lido como decimal.
- Abre por `Super` (um toque) ou `Super + Espaço`, e pela marca "Mk" na ilha.

## Por quê assim

- **Analisador próprio, sem `eval`**: o texto vem de um campo de busca; avaliá-lo como JavaScript executaria o que a pessoa digitasse. O `Calc.js` devolve `null` quando o texto não é uma conta.
- **Sempre carregada**: a busca fica num `Loader` à parte no `Tide`, num canto de 1 px. Criá-la a cada abertura (lista de apps, ícones, textos) levava cerca de um segundo numa VM sem aceleração ([[Painel líquido (Tide)]]).
- **Só mexe no que mudou ao abrir**: limpar o texto e trocar a lista de projetos refazia as linhas a cada abertura, que é o que pesa numa máquina lenta; a lista de projetos só é trocada se mudou.
- **Toque no Super**: bind de soltar a tecla, que não dispara se outra tecla foi apertada junto ([[Atalhos]]).
- **Tudo numa lista**: painéis sem botão na ilha (sistema, rede, Bluetooth, sessão) continuam alcançáveis sem um atalho próprio.

## Onde está no código

`shell/mare/LauncherPanel.qml` (351 linhas), `shell/mare/Calc.js` (92 linhas), o `Loader` `kept` em `Tide.qml`.

## Ligações

Depende de: [[Painel líquido (Tide)]], [[Projetos e workspaces]], [[Atalhos]]. Abre: [[Painéis]], [[Ajustes]], [[Boas-vindas]], [[Emoji]].
