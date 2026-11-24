---
camada: "12 Decisões e limites"
tags:
  - mk
  - decisões
  - maré
  - monitores
fontes:
  - "shell/mare/Monitor.qml"
  - "shell/mare/Mare.qml"
  - "shell/mare/Bar.qml"
  - "shell/mare/Tide.qml"
---

# D08 Geometria por monitor

A posição e a largura da ilha e do painel são ligadas no `Monitor.qml`, uma vez por tela. O singleton `Mare` não guarda geometria de tela.

## Contexto

Para o painel descer encaixado na ilha, os dois precisam trocar números: a ilha diz onde está, o painel diz de que largura precisa. Na primeira versão, a ilha publicava a posição **no `Mare`** e o painel lia de lá. Funciona com uma tela. Mas o `Mare` é um singleton, um só para o shell inteiro, e a `Bar` e o `Tide` existem uma vez por tela: com duas telas de larguras diferentes, a ilha de uma tela sobrescrevia o valor da outra, e o painel abria com a posição da ilha da outra tela.

## Escolha

- `Bar` expõe `islandX`/`islandW` e recebe `sheetW`; `Tide` expõe `sheetW` e recebe `islandX`/`islandW`.
- As ligações são feitas no `Monitor.qml`, por propriedades, entre a `Bar` e o `Tide` **daquela tela**.
- No `Mare` fica só `originX` (o x do clique), que é do evento.

## Alternativas

- **Guardar no `Mare` um mapa por nome de tela** *(não discutida no repositório)*.
- **Um `Mare` por tela** *(não discutida no repositório)*: o IPC e o `mode` contam com um estado único.

## Consequência

- Regra do shell: "nada de geometria de tela no `Mare`".
- Os painéis só abrem no monitor em foco (`active: focused`), então um estado único de "qual painel" continua bastando.
- A ilha publica **para onde vai**, não onde está a cada quadro, senão as molas do painel recomeçavam sem andar ([[Ilha]]).

> [!warning] Conferido só com tela virtual
> A correção foi verificada com uma segunda tela virtual no [[dev-shot]]; dois monitores físicos de larguras diferentes nunca foram testados.

## Onde está no código

`shell/mare/Monitor.qml`; o comentário em `shell/mare/Mare.qml` ("Nada de geometria de tela mora aqui além de originX").

## Ligações

Afeta: [[Monitor por tela]], [[Ilha]], [[Painel líquido (Tide)]], [[Singletons]]. Índice: [[Registro de decisões]].
