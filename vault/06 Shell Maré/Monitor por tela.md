---
camada: "06 Shell Maré"
tags:
  - mk
  - maré
  - monitores
  - arquitetura
fontes:
  - "shell/mare/Monitor.qml"
  - "shell/mare/shell.qml"
  - "shell/mare/Mare.qml"
  - "shell/mare/README.md"
---

# Monitor por tela

`Monitor.qml` é o que existe uma vez **por tela**: o `shell.qml` cria um para cada item de `Quickshell.screens`, e ele monta o fundo, a ilha, o painel, os Ajustes, as boas-vindas e a janela de clique fora daquela tela.

## Para quê

Separar o que é da tela (geometria, janelas) do que é do shell (estado).

## Como funciona

- `shell.qml`: `Variants { model: Quickshell.screens; Monitor {} }`.
- Cada `Monitor` cria `Backdrop`, `Bar`, `Tide`, `SettingsWindow`, `WelcomeWindow` e a `PanelWindow` `mare-outside`.
- **O encaixe ilha ↔ painel é ligado aqui**, por propriedades:
  - `Bar.sheetW: tide.sheetW` (de que largura o painel precisa);
  - `Tide.islandX: bar.islandX` e `Tide.islandW: bar.islandW` (onde a ilha vai estar).
- **`focused`**: verdadeiro se há uma tela só, ou se `Hyprland.focusedMonitor` é esta. `Tide`, `SettingsWindow` e `WelcomeWindow` recebem `active: focused`: os painéis só abrem no monitor em foco.
- A ilha e o fundo existem em todas as telas.

## Por quê assim

**Nada de geometria de tela no singleton `Mare`.** O singleton é um só para todas as telas; a `Bar` e o `Tide` são um por tela. Antes, a ilha publicava a posição no `Mare` e o painel lia de lá; com telas de larguras diferentes, o painel abria com a posição da ilha **da outra tela**. A correção foi tirar posição e largura do singleton e ligá-las no `Monitor.qml`. No `Mare` ficou só `originX` (o x do clique), que é do evento e não da tela. Decisão registrada em [[D08 Geometria por monitor]].

O que continua único, de propósito: a tela de bloqueio (uma `LockScreen` que dá uma `LockSurface` a cada tela), o diálogo de polkit e todos os singletons.

> [!warning] Não testado com duas telas físicas
> A correção foi conferida com uma segunda tela **virtual** (`MK_DEV_HEADLESS` no [[dev-shot]]), não com dois monitores de verdade.

## Onde está no código

`shell/mare/Monitor.qml` (98 linhas), `shell/mare/shell.qml`.

## Ligações

Depende de: [[Maré]], [[Monitores]]. Monta: [[Ilha]], [[Painel líquido (Tide)]], [[Ajustes]], [[Boas-vindas]], [[Clique fora e teclado exclusivo]].
