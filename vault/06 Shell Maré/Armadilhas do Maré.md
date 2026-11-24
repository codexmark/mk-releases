---
camada: "06 Shell Maré"
tags:
  - mk
  - maré
  - armadilhas
  - quickshell
fontes:
  - "shell/mare/README.md"
  - "shell/mare/Mare.qml"
  - "shell/mare/scripts/mare-start"
  - "shell/mare/Projects.qml"
---

# Armadilhas do Maré

Coisas que já quebraram no Maré de formas difíceis de adivinhar, e a regra que ficou de cada uma. A lista vem da seção "Armadilhas" do `shell/mare/README.md`.

## Para quê

Para quem vai mexer no shell não repetir o mesmo defeito.

## Como funciona

**Arquivo de singleton novo pede um Maré novo.** O Quickshell recarrega o QML quando um arquivo muda, mas esse recarregamento pode não registrar um arquivo novo com `pragma Singleton`. Depois de criar um, reinicie (`scripts/mare-start --restart`) ou teste com o [[dev-shot]], que sempre sobe do zero. É por isso que o `mk-update` reinicia o Maré inteiro depois de trocar o Mk. (O README marca isto como "observação de uso"; o código não anota.)

**Zona exclusiva com a interface fora de 100%.** Para uma janela ignorar o espaço reservado da ilha, o protocolo pede a zona exclusiva `-1`. Com `QT_SCALE_FACTOR`, o valor é multiplicado pela escala e `-1` virava `0`: os painéis desciam 28 px abaixo da ilha (a 85%) e o fundo parava antes do alto. Use sempre o par:

```qml
exclusionMode: ExclusionMode.Normal
exclusiveZone: Mare.ignoreZone
```

`Mare.ignoreZone` é `-Math.ceil(1 / uiScale)`: o valor que, depois da escala, dá `-1`. Conferido a 75, 85, 100 e 115%.

**Teclado depois de reiniciar o Maré.** Quando as camadas renascem, a janela em uso fica sem o teclado até um clique. O `mare-start --restart` devolve o foco à janela ativa duas vezes. Reiniciando de outro jeito (`qs kill` e `qs -p` à mão), isso não acontece.

**Teclado nos painéis.** O painel pede o teclado ele mesmo; o `HyprlandFocusGrab` foi abandonado ([[Clique fora e teclado exclusivo]]).

**Nada de geometria de tela no `Mare`.** Posição e largura da ilha e do painel moram no `Monitor.qml` ([[Monitor por tela]]).

**O primeiro quadro do painel.** O `Tide` só desenha quando o conteúdo tem tamanho e nasce de uma gota, não de um fio ([[Painel líquido (Tide)]]).

**Comandos ao Hyprland.** O `dispatch` espera código Lua. Do QML, use `Hypr.exec`, `Hypr.workspace` e `Hypr.focusWindow`, nunca `Hyprland.dispatch` direto ([[Singletons]]).

**`shellPath` não sobe da pasta do shell.** `Quickshell.shellPath("../x")` volta como caminho relativo. Para chegar à `<raiz>`, use `Settings.dir`. (Foi assim que o ghostty abria sem a config do Mk.)

**Rodar `qs -p shell/mare` direto do repositório não é um teste isolado.** É um segundo Maré na sua tela, e a `<raiz>` dele passa a ser `shell/`, onde o `ajustes.json` seria gravado.

## Por quê assim

Quase todas vêm da combinação de três escolhas do projeto: a escala da interface pelo Qt, a config do Hyprland em Lua e um shell com várias camadas por tela. Cada uma tem ganho claro, e o preço é esta lista. Registrá-la no README do shell (e aqui) é a forma de não pagar duas vezes.

## Onde está no código

`shell/mare/README.md` (seção "Armadilhas"); `Mare.qml` (`uiScale`, `ignoreZone`); `scripts/mare-start`.

## Ligações

Depende de: [[Maré]]. Relacionado: [[Clique fora e teclado exclusivo]], [[Monitor por tela]], [[Painel líquido (Tide)]], [[dev-shot]], [[Convenções]].
