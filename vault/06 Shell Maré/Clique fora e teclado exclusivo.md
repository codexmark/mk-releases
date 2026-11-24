---
camada: "06 Shell Maré"
tags:
  - mk
  - maré
  - teclado
  - foco
  - wayland
fontes:
  - "shell/mare/Monitor.qml"
  - "shell/mare/Tide.qml"
  - "shell/mare/Mare.qml"
  - "shell/mare/SettingsWindow.qml"
  - "shell/mare/WelcomeWindow.qml"
  - "shell/mare/PolkitDialog.qml"
  - "shell/mare/README.md"
---

# Clique fora e teclado exclusivo

Duas coisas que parecem simples e deram trabalho: fechar o painel quando se clica fora dele, e garantir que o que se digita chega ao painel aberto. O Maré resolve com uma janela transparente (`mare-outside`) e com cada janela pedindo o teclado para si.

## Para quê

Sem isso, o `Esc` não fecha o painel, a busca não recebe o que se digita, ou um clique na área de trabalho deixa o painel aberto para sempre.

## Como funciona

**A janela `mare-outside`** (no `Monitor.qml`):

- uma `PanelWindow` transparente, ancorada nos quatro lados, na camada **Top**, com `keyboardFocus: None`;
- visível só quando há um painel de verdade aberto (não em `rest`, `picker`, passageiros nem nos Ajustes);
- a máscara de entrada é a tela inteira **menos** o retângulo da ilha;
- um `TapHandler` com qualquer botão chama `Mare.close()`.

O painel (`Tide`) está na camada **Overlay**, por cima dela: clicar no painel não bate na `mare-outside`. Clicar na ilha também não (o buraco na máscara): ali quem decide é o item clicado, que troca de painel ou fecha.

**O teclado**:

| Janela | Quando pede `WlrKeyboardFocus.Exclusive` |
|---|---|
| `Tide` | `tide.open && Mare.wantsKeyboard` |
| `SettingsWindow` | enquanto a folha está aberta |
| `WelcomeWindow` | enquanto a folha está na frente (não recolhida) |
| `PolkitDialog` | sempre que existe |

`Mare.wantsKeyboard` vale para todo painel aberto, menos `picker`, os passageiros e os Ajustes (que pedem por conta própria).

## Por quê assim

- **Antes era um `HyprlandFocusGrab`**, que fechava no clique fora e também entregava o teclado. O Hyprland dá o foco a **uma janela qualquer** da lista do grab; quando caía na ilha, o que se digitava não chegava a ninguém. A correção separou as duas funções: o clique fora virou uma janela, e o teclado, um pedido da própria janela do painel ([[D07 Teclado exclusivo no lugar do grab]]).
- **Todo painel recebe teclado**, não só a busca: antes só a busca, o Alt+Tab e a área de transferência recebiam, e nos outros o `Esc` ia para a janela de baixo.
- **Com um app em tela cheia** a `mare-outside` fica por baixo do app; aí quem fecha é o `Esc`.
- **`Mare.switchedAt`**: guarda quando um painel deu lugar a outro, para a troca não ser confundida com um clique fora.
- **As boas-vindas soltam o teclado** na hora em que dão a vez a um painel, e os atalhos do Hyprland continuam valendo.
- **Depois de reiniciar o Maré**, a janela em uso fica sem o teclado até um clique; o `mare-start --restart` devolve o foco ([[Autostart]]).

> [!note] Conferido só na VM
> `docs/limites.md` lista "o teclado e os cliques fora dos painéis da ilha" entre o que foi conferido na VM.

## Onde está no código

- `shell/mare/Monitor.qml` (a `PanelWindow` `outside`, com o comentário que conta a história)
- `shell/mare/Tide.qml` (a linha do `keyboardFocus`)
- `shell/mare/Mare.qml` (`wantsKeyboard`, `switchedAt`)

## Ligações

Depende de: [[Monitor por tela]], [[Painel líquido (Tide)]], [[Hyprland em Lua]]. Relacionado: [[Armadilhas do Maré]], [[Polkit]], [[Boas-vindas]].
