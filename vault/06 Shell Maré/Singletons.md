---
camada: "06 Shell Maré"
tags:
  - mk
  - maré
  - estado
  - singletons
fontes:
  - "shell/mare/README.md"
  - "shell/mare/Mare.qml"
  - "shell/mare/Hypr.qml"
  - "shell/mare/Theme.qml"
  - "shell/mare/Motion.qml"
  - "shell/mare/Notifs.qml"
  - "shell/mare/Caffeine.qml"
  - "shell/mare/Modes.qml"
  - "shell/mare/Projects.qml"
---

# Singletons

Cada arquivo com `pragma Singleton` é dono de um pedaço do estado do shell. Há um de cada para o shell inteiro, não um por tela.

## Para quê

Painéis são criados e destruídos ao abrir e fechar; o que precisa durar mais que um painel mora num singleton.

## Como funciona

| Singleton | Estado que guarda |
|---|---|
| `Mare` | o modo da ilha, `originX`, o índice do Alt+Tab, o OSD, a página dos Ajustes, `uiScale`/`ignoreZone`; tem o IPC |
| `Settings` | os ajustes (`values` sobre `defaults`), o que a máquina tem (`caps`), o último erro, os arquivos gerados |
| `Theme` | cores, famílias de fonte e a letra grega de cada workspace; só leitura |
| `Motion` | durações e curvas, multiplicadas por `motion.speed`; só leitura |
| `Hypr` | nenhum estado: é o único ponto que manda `dispatch` ao Hyprland |
| `Projects` | a ligação workspace → projeto, os repositórios em `~/Work`, o estado do projeto em foco |
| `Notifs` | o servidor de notificações: a lista, o não perturbe, o aviso na tela |
| `Caffeine` | a cafeína: enquanto ligada, um inibidor de ociosidade do systemd segura o descanso de tela, o bloqueio e a tela apagada |
| `Modes` | o modo apresentação ou jogo ligado e o que ele mudou (não perturbe, cafeína, perfil de energia), para devolver na saída; a conta está no `Modes.js` |
| `Lock` | a tela de bloqueio: `locked`, a senha, a conversa com o PAM, a animação |
| `Welcome` | as boas-vindas: `shown`, `step`, `paused`, o "já vi" |
| `Wallpaper` | o papel de parede em uso, a cor tirada dele, a lista de imagens; tem um IPC próprio |
| `Display` | brilho (`brightnessctl`) e luz noturna (`hyprsunset`), com o horário automático |
| `Monitors` | as telas: o que o Hyprland tem, o rascunho e a contagem de 15 s |
| `Keyboard` | os layouts do teclado principal e qual está ativo |
| `Media` | o player MPRIS em destaque |
| `Sound` | o espectro do som que está tocando, para a onda da ilha |
| `Stats` | CPU, memória, temperatura, rede e disco a cada 2 s, com histórico |
| `Weather` | o clima, renovado a cada 30 min |
| `Updates` | o resultado da última procura por atualizações e se a ilha deve acender |
| `Vpn` | há túnel aberto e qual |
| `Capture` | a gravação da tela em andamento e se dá para gravar |
| `BluetoothPair` | o pareamento em andamento |
| `BatteryWatch` | os avisos de bateria fraca já dados (20% e 8%) |

## Por quê assim

- **`Hypr` como único ponto de contato** com o compositor: com a config em Lua, o `dispatch` espera código Lua. `Hypr.exec`, `Hypr.workspace` e `Hypr.focusWindow` emitem o formato certo conforme `Hyprland.usingLua`, e o resto do shell não precisa saber ([[D02 Um só ponto fala com o Hyprland]]).
- **`Theme` e `Motion` só leitura**: nenhum painel escreve cor, nome de fonte ou duração direto; senão a escolha de velocidade dos Ajustes não valeria para ele.
- **Estado que sobrevive ao painel**: `BluetoothPair` (parear leva de 5 a 20 s), `Capture` (a gravação continua com o painel fechado), `BatteryWatch` (para não repetir o aviso depois de recarregar o shell).
- **Um `nmcli monitor` só** (`Vpn`): a barra existe uma vez por monitor; se cada uma vigiasse a rede, haveria um processo por tela.
- **Armadilha**: um arquivo novo de singleton pode não ser registrado pelo recarregamento a quente do Quickshell; pede um Maré novo ([[Armadilhas do Maré]]).

## Onde está no código

`shell/mare/*.qml` com `pragma Singleton`; a tabela vem do `shell/mare/README.md`.

## Ligações

Depende de: [[Maré]]. Usado por: [[Painéis]], [[Ilha]], [[Ajustes]], [[IPC do Maré]], [[Scripts mare]].
