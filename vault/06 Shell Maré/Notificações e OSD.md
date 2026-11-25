---
camada: "06 Shell Maré"
tags:
  - mk
  - maré
  - notificações
  - osd
fontes:
  - "shell/mare/Notifs.qml"
  - "shell/mare/ToastView.qml"
  - "shell/mare/OsdView.qml"
  - "shell/mare/NotificationsPanel.qml"
  - "shell/mare/Mare.qml"
  - "shell/mare/BatteryWatch.qml"
---

# Notificações e OSD

O Maré é o servidor de notificações da sessão. Uma notificação nova aparece como um aviso que cresce da ilha e volta sozinho; todas ficam no histórico. O mesmo mecanismo de "passageiro" mostra o nível de volume e de brilho.

## Para quê

Avisar sem interromper, e dar retorno visual às teclas de mídia.

## Como funciona

- **`Notifs.qml`** usa o `NotificationServer` do Quickshell (com corpo, ações, imagem e persistência) e marca toda notificação como `tracked`.
- **Aviso (`toast`)**: `Mare.flash("toast", ms)`, com `ms` = `notify.duration` (padrão 5 s; as críticas ficam o dobro). `ToastView.qml` desenha: clique abre a ação padrão do app, o × descarta.
- **Não perturbe** (`Super + Shift + N`, ou `ipc call mare dnd`): segura os avisos, menos os críticos; o histórico continua registrando tudo. O modo apresentação (`Super + Shift + P`) e o modo jogo (`Super + Shift + G`) o ligam enquanto duram e, ao sair, o devolvem ao que era (`shell/mare/Modes.qml`).
- **Histórico** (`Super + N`): a mais nova em cima.
- **OSD**: `Mare.osd(tipo, valor, mudo)` mostra por 1,3 s uma barra de nível na ilha (`OsdView.qml`) quando o volume ou o brilho mudam pelas teclas.
- **Bateria fraca** (`BatteryWatch`): dois degraus, 20% (normal) e 8% (crítico, que fura o não perturbe). Cada um avisa uma vez por descarga e se rearma na tomada.

## Por quê assim

- **`flash` não atropela um painel aberto**: se há um painel de verdade na frente, o passageiro não entra; a notificação vai só para o histórico.
- **Passageiros não pedem teclado** nem armam a janela de clique fora: somem sozinhos.
- **Hora de chegada guardada pelo Maré**: o protocolo de notificações não traz o horário.
- **`keepOnReload`**: recarregar o QML não perde a lista.
- **Sem `mako` nem `fuzzel`**: o Maré "também é o launcher e o servidor de notificações" (comentário de `desktop.packages`); só o `libnotify` (o `notify-send`) está na lista.

> [!warning] Não testado
> O aviso de bateria fraca nunca foi visto num notebook descarregando ([[Limitações conhecidas]]).

## Onde está no código

`shell/mare/Notifs.qml`, `ToastView.qml`, `OsdView.qml`, `NotificationsPanel.qml`, `BatteryWatch.qml`; `Mare.qml` (`flash`, `osd`, `transients`).

## Ligações

Depende de: [[Singletons]], [[Ilha]], [[Áudio]]. Usado por: [[Atalhos]], [[Painéis]].
