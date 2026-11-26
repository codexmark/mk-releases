---
camada: "06 Shell Maré"
tags:
  - mk
  - maré
  - ilha
  - barra
fontes:
  - "shell/mare/Bar.qml"
  - "shell/mare/IslandItem.qml"
  - "shell/mare/Sound.qml"
  - "shell/mare/Theme.qml"
  - "README.md"
---

# Ilha

A ilha é a barra do Maré: uma peça só, pendurada no meio da borda de cima, com filetes côncavos onde encosta na tela. Tudo nasce dela.

## Para quê

Mostrar o essencial (onde estou, que horas são, o que está tocando, o que pede atenção) e ser o ponto de onde cada painel desce.

## Como funciona

- **Esquerda**: a marca "Mk" (abre a busca), os workspaces por letra grega (α β γ…, `Theme.wsName`) com o atual numa placa de osso, e as leituras do sistema.
- **Centro**: relógio com a data, e o clima ao lado.
- **Direita**: a bandeja (`TrayIcon`) e os indicadores.
- **Altura**: reserva 34 px (`barH`) para as janelas. A janela da ilha é mais alta que isso, para caber a onda do som.
- **Largura**: exatamente a do conteúdo. A ilha se desloca para manter o relógio no meio da tela e cresce quando aparece um item (bandeja, Bluetooth, música, alerta do sistema).
- **Opacidade**: translúcida em repouso (`island.restOpacity`, padrão 82%), sólida com um painel aberto.
- **Clique**: um `IslandItem` chama `Mare.openAt(painel, x)`, e o painel desce embaixo do item clicado.
- **A onda**: a borda de baixo ondula com o espectro do que está tocando (graves à esquerda, agudos à direita), lido do `cava` pelo `Sound.qml`. Em silêncio, ou com um painel aberto, volta a ser a borda reta. Liga e desliga em Ajustes › Ilha.
- **O rastro**: uma segunda linha, mais fraca, que é a mesma curva com os níveis atrasados (`Sound.trail`): vem atrás na subida e demora a cair, então cruza a onda. As contas (`smooth`, `trail`, `curve`) estão no `Wave.js`.
- **Indicadores que só aparecem quando importam**: sistema, rede, Bluetooth, notificações, mídia, gravação em andamento (com o tempo), modo apresentação ou jogo ligado (o ícone do modo, em cobre; um clique desliga), VPN, atualizações.
- A ilha expõe `islandX`/`islandW` (para onde ela vai) e recebe `sheetW` (de que largura o painel precisa); quem liga os dois é o [[Monitor por tela]].

## Por quê assim

- **No molde da ilha do Ryoku**: estrutura, altura e comportamento foram estudados lá, mas "a forma é desenhada pelo próprio Maré", sem depender do plugin de formas daquele projeto.
- **Compacta e centrada**: a primeira versão ocupava quase a largura toda. Passou a mostrar só o essencial; o resto aparece quando importa.
- **Publica o destino, não a posição atual**: a ilha alarga animando quando o painel é mais largo que ela; um painel que seguisse a ilha em movimento via o alvo mudar a cada quadro, e as molas dele recomeçavam sem andar (descia como uma tira estreita). Com o destino publicado, a busca passou de cerca de 440 ms para cerca de 130 ms até ficar larga, na máquina de desenvolvimento.
- **Workspaces por letra grega**, e o atual numa placa: projetos se distinguem "por número e nome, não por cor" ([[Direção Folha]]).
- **Só a ilha recebe cliques**; o resto da faixa deixa passar para as janelas.

## Onde está no código

`shell/mare/Bar.qml` (594 linhas), `IslandItem.qml`, `TrayIcon.qml`, `Sound.qml`, `scripts/mare-cava`.

## Largura estável

As leituras (temperatura, rede, processador, memória, bateria) reservam o lugar do seu texto mais largo: o `IslandItem` tem `reserve`, e os textos vêm do `Island.js` (`rate`, `percent`, `degrees`), que nunca passam de `999K/s`, `100%` e `99°`. Assim a ilha não estica nem encolhe quando um número ganha ou perde um dígito; só muda de tamanho quando algo entra ou sai dela.

## Ligações

Depende de: [[Maré]], [[Singletons]], [[Áudio]]. Usado por: [[Painel líquido (Tide)]], [[Monitor por tela]], [[Painéis]].
