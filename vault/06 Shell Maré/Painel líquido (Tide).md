---
camada: "06 Shell Maré"
tags:
  - mk
  - maré
  - painel
  - animação
fontes:
  - "shell/mare/Tide.qml"
  - "shell/mare/Mare.qml"
  - "shell/mare/Motion.qml"
  - "shell/mare/README.md"
---

# Painel líquido (Tide)

`Tide.qml` é o painel que desce da ilha. Em vez de um retângulo que desliza, é uma forma "líquida": escorre da borda de baixo da ilha, soldada a ela, e carrega dentro o conteúdo do painel em `Mare.mode`.

## Para quê

Dar a todos os painéis o mesmo corpo, o mesmo movimento e o mesmo tratamento de teclado; cada painel só precisa dizer o próprio tamanho e conteúdo.

## Como funciona

- Uma `PanelWindow` por tela, na camada *Overlay* (`mare-tide`), por cima da ilha e colada nela.
- Um `Loader` (`content`) escolhe o componente pelo `switch` sobre `Mare.mode`; o tamanho da forma vem do `implicitWidth`/`implicitHeight` da raiz do painel.
- A busca fica num segundo `Loader` (`kept`), sempre carregada, num canto de 1 px; ao abrir, só muda de lugar.
- **A forma** é desenhada com `QtQuick.Shapes` e cada parte anda numa mola própria: o pescoço (junto da ilha) rápido, o corpo mais mole, o fundo em duas alturas (meio e cantos). Abrindo, nasce como uma gota; trocando de item, o pescoço vai na frente e o corpo vem inclinado atrás; fechando, os cantos sobem primeiro.
- O painel fica sempre dentro do trecho reto da borda de baixo da ilha (depois do ombro e do canto), e a ilha alarga quando ele é mais largo.
- Só o painel recebe cliques (`mask`); o resto da janela é transparente ao mouse.
- **Teclado**: `WlrKeyboardFocus.Exclusive` enquanto um painel está aberto neste monitor ([[Clique fora e teclado exclusivo]]). `Esc` fecha qualquer painel.
- Com o movimento desligado nos Ajustes, vai direto para o lugar.

**Registrar um painel novo** são três lugares mais o arquivo: o nome em `Mare.panels`, o `case` no `switch` do `Tide`, o `Component` no fim do `Tide`, e `MeuPanel.qml`.

## Por quê assim

O desenho de hoje é o resultado de uma série de correções feitas observando o painel numa VM lenta. Cada regra tem um defeito por trás:

- **Só desenha quando o conteúdo tem tamanho** (`sheet.loaded`): numa máquina lenta o conteúdo demora a carregar, a largura-alvo ficava em 0 e aparecia um fio pendurado na ilha por um segundo.
- **Nasce de uma gota, não de um fio**: a altura desenhada acompanha a largura desde zero. O primeiro quadro é justamente aquele em que a máquina trava desenhando o conteúdo pela primeira vez.
- **A busca sempre carregada**: criada a cada toque no Super, levava cerca de um segundo numa VM sem aceleração.
- **Sem espera artificial**: uma versão esperava "os quadros voltarem a fluir" antes de escorrer; com a busca pré-carregada isso só atrasava, e foi retirado.
- **Duas camadas, não uma forma só**: a ilha e o painel são janelas diferentes; a sobreposição cobre só a linha fina da emenda (uma sobra maior cobriria a data do relógio).
- **Zona exclusiva**: usa `Mare.ignoreZone` para ficar por cima da ilha com a interface fora de 100% ([[Armadilhas do Maré]]).

## Onde está no código

`shell/mare/Tide.qml` (370 linhas), `Mare.qml`, `Motion.qml`. A seção "Registrar um painel novo" do `shell/mare/README.md`.

## Ligações

Depende de: [[Ilha]], [[Monitor por tela]], [[Singletons]], [[Animações]]. Carrega: [[Painéis]], [[Busca e calculadora]].
