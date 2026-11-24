---
camada: "06 Shell Maré"
tags:
  - mk
  - maré
  - descanso
  - energia
fontes:
  - "shell/mare/ScreenSaver.qml"
  - "shell/mare/Mare.qml"
  - "config/hypr/hypridle.conf"
  - "shell/mare/SettingsSchema.js"
  - "tests/qml/descanso.test.sh"
  - "docs/limites.md"
---

# Descanso de tela

O descanso de tela é o que o Mk faz quando ninguém mexe: em vez de dormir, toma a tela com a própria marca, construída numa folha preta, com a hora embaixo.

## Para quê

Ocupar o lugar que seria do suspender: a máquina continua ligada e acessível, e a tela não fica parada numa imagem.

## Como funciona

- **O desenho** (`ScreenSaver.qml`): a folha fica preta e a marca "Mk" é construída como no logo e na tela de entrada (eixos, guias, diagonais, as letras, os nós). Fica um tempo, some, e volta a ser construída **noutro canto**. Embaixo dela, a hora.
- **Quem liga**: o hypridle, aos 3 minutos de fábrica (`idle.saver`, em Ajustes › Energia › Descanso de tela; 0 desliga), ou o IPC:

  ```bash
  qs -p ~/.config/mk/shell ipc call mare saver on | off | toggle
  ```

- **Quem desliga**: qualquer tecla, clique ou movimento do mouse; o `on-resume` do hypridle (`saver off`); o bloqueio da tela.
- **Estado**: `Mare.saver` (um booleano à parte do `Mare.mode`). Ligar fecha o painel aberto. `ipc call mare state` devolve `saver` enquanto ele está na tela.
- A janela cobre a tela inteira, também por cima da ilha (`Mare.ignoreZone`).

## Por quê assim

- **No lugar de dormir**: veja [[D16 Descanso de tela no lugar do suspender]].
- **A marca muda de canto a cada ciclo** "para não marcar a tela".
- **Com a tela bloqueada não aparece**: "o bloqueio tem o desenho dele". O `saver on` não liga com `Lock.locked`, e bloquear desliga o descanso. Com os tempos de fábrica, isso quer dizer dois minutos de descanso (dos 3 aos 5) e depois a tela de bloqueio até a tela apagar aos 30.
- **O mesmo desenho do logo e da entrada**: as peças (as letras em camadas e a geometria) vêm do mesmo gerador ([[Arte e geradores]]).

> [!warning] Pouco visto
> Ligar e desligar por IPC foi visto na sessão aninhada. A espera real de 3 minutos pelo hypridle, o comportamento com dois monitores e o consumo com o descanso horas na tela não foram vistos ([[Limitações conhecidas]]).

## Onde está no código

`shell/mare/ScreenSaver.qml` (264 linhas); `shell/mare/Mare.qml` (a propriedade `saver` e a função `saver` do IPC); `config/hypr/hypridle.conf`; `tests/qml/descanso.test.sh`.

## Ligações

Depende de: [[Maré]], [[IPC do Maré]], [[Ociosidade e bloqueio]]. Relacionado: [[Tela de bloqueio]], [[Energia e suspensão]], [[SDDM]].
