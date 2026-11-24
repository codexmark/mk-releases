---
camada: "12 Decisões e limites"
tags:
  - mk
  - decisões
  - energia
  - descanso
fontes:
  - "config/hypr/hypridle.conf"
  - "shell/mare/SettingsSchema.js"
  - "shell/mare/ScreenSaver.qml"
  - "shell/mare/scripts/mare-sleep"
  - "docs/limites.md"
---

# D16 Descanso de tela no lugar do suspender

De fábrica o Mk não suspende sozinho. Quando ninguém mexe, entra o descanso de tela; a tela apaga aos 30 minutos; suspender passa a ser uma escolha.

## Contexto

Até aqui o padrão era suspender aos 30 minutos de ociosidade (`power.sleep: "suspender"`), com uma exceção semeada pelo instalador para a AMD Navi 10, que às vezes não volta do S3: a tela fica preta e só reiniciando ([[Energia e suspensão]]). A hibernação já tinha sido retirada ([[D05 Hibernação retirada]]). Sobrava um padrão que, numa placa de risco, podia custar a sessão aberta, e um caminho especial no instalador para contorná-lo.

## Escolha

- `power.sleep` de fábrica passa a `"nada"`: "de fábrica o Mk não dorme: fica o descanso de tela".
- Entra o [[Descanso de tela]] (`ScreenSaver.qml`), ligado pelo hypridle aos 3 minutos: "faz o papel que era do suspender".
- Tempos de fábrica novos: descanso aos 3 minutos, escurecer aos 4, bloqueio aos 5, tela desligada aos 30 (era 6).
- O passo de dormir continua no `hypridle.conf`, mas o `mare-sleep now` só age para quem escolheu suspender em Ajustes › Energia.

## Alternativas

- **Manter suspender como padrão, com a exceção da Navi 10**: era o estado anterior. A lista de placas de risco só tem o que já foi visto falhar; uma placa fora dela cairia no defeito sem aviso.
- **Só desligar a tela, sem descanso** *(não discutida no repositório)*.

O motivo, segundo quem mantém: a Navi 10 que não volta do S3 e a decisão de não dormir. O repositório registra o resultado (o commit, o comentário do esquema e `docs/limites.md`: "Dormir, de fábrica. O Mk não suspende sozinho"); a ligação entre a placa e o padrão novo não está escrita no código.

## Consequência

- A máquina fica ligada quando ociosa; a economia passa a ser a tela apagar aos 30 minutos. O consumo com o descanso horas na tela não foi medido.
- O descanso dura pouco na prática: aos 5 minutos a tela bloqueia, e quem aparece é a tela de bloqueio.
- Deduzido do código, não ensaiado: quem já usava o Mk e nunca mexeu em "Ao dormir" não tinha a chave gravada (só o que sai da fábrica vai para o `ajustes.json`), então passa a não dormir depois de atualizar.
- A semente do instalador para a Navi 10 ficou redundante; o aviso de Ajustes › Energia continua, agora para quem **escolher** suspender numa máquina de risco.
- Um `hypridle.conf` já gerado pelos Ajustes é regerado quando o passo de dormir dele deixa de bater com o que vale (`syncIdle`).

## Onde está no código

`config/hypr/hypridle.conf` (o cabeçalho); `shell/mare/SettingsSchema.js` (as linhas `idle.saver` e `power.sleep`); `shell/mare/ScreenSaver.qml`; `shell/mare/scripts/mare-sleep`.

## Ligações

Afeta: [[Descanso de tela]], [[Ociosidade e bloqueio]], [[Energia e suspensão]], [[Swap e hibernação]]. Índice: [[Registro de decisões]].
