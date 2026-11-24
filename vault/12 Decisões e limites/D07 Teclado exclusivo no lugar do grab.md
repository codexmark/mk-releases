---
camada: "12 Decisões e limites"
tags:
  - mk
  - decisões
  - maré
  - teclado
fontes:
  - "shell/mare/Monitor.qml"
  - "shell/mare/Tide.qml"
  - "shell/mare/README.md"
---

# D07 Teclado exclusivo no lugar do grab

Cada janela do Maré que precisa do teclado pede o teclado **ela mesma** (`WlrKeyboardFocus.Exclusive`), e o clique fora é pego por uma janela transparente. O `HyprlandFocusGrab` saiu.

## Contexto

O mecanismo original era um `HyprlandFocusGrab`: uma lista de janelas do shell; clicar fora delas desfazia o grab (fechando o painel), e o grab entregava o teclado. Deu três defeitos, um depois do outro:

1. Na VM, o que se digitava com a busca aberta ia para a janela de baixo: o painel pedia o teclado no mesmo instante em que o grab era armado, e quando o grab chegava antes não havia o que focar.
2. Trocar de um painel com teclado para outro fechava tudo: soltar o teclado desfazia o grab, e isso era lido como clique fora.
3. O Hyprland entrega o foco a **uma janela qualquer** da lista do grab. Quando caía na ilha, o que se digitava não chegava a ninguém.

Os dois primeiros foram remendados. O terceiro é do desenho do grab.

## Escolha

- O `Tide` fica em `WlrKeyboardFocus.Exclusive` enquanto um painel está aberto no seu monitor; os Ajustes e as boas-vindas fazem o mesmo enquanto estão na frente.
- O clique fora é de uma `PanelWindow` transparente (`mare-outside`), na camada Top, sem teclado, com um buraco na máscara onde está a ilha.

## Alternativas

- **Continuar com o grab**: foi o que os dois primeiros remendos fizeram; o terceiro defeito é do próprio mecanismo.
- **`OnDemand` em vez de `Exclusive`**: foi um dos remendos ("a janela agora aceita teclado (OnDemand) desde sempre"), anterior à troca; a versão final pede `Exclusive`.

## Consequência

- Todo painel aberto recebe o teclado, e o `Esc` fecha qualquer um.
- Os passageiros (aviso, OSD) e o seletor não pedem teclado.
- Com um app em tela cheia, a janela de clique fora fica por baixo do app; quem fecha é o `Esc`.
- Foi preciso `Mare.switchedAt`, para a troca de painel não parecer clique fora.
- Uma camada com teclado exclusivo leva o teclado embora de quem estava digitando: daí os cuidados das [[Boas-vindas]] (recolher, não subir no meio do `mk-update`).

## Onde está no código

`shell/mare/Monitor.qml` (o comentário da janela `outside`), `shell/mare/Tide.qml`.

## Ligações

Afeta: [[Clique fora e teclado exclusivo]], [[Painel líquido (Tide)]], [[Boas-vindas]], [[Armadilhas do Maré]]. Índice: [[Registro de decisões]].
