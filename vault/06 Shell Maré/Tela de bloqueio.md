---
camada: "06 Shell Maré"
tags:
  - mk
  - maré
  - bloqueio
  - segurança
  - pam
fontes:
  - "shell/mare/Lock.qml"
  - "shell/mare/LockScreen.qml"
  - "shell/mare/LockSurface.qml"
  - "config/hypr/mare/look.lua"
  - "config/hypr/hypridle.conf"
  - "config/hypr/hyprlock.conf"
---

# Tela de bloqueio

A tela de bloqueio é do Maré: um bloqueio de sessão do Wayland (`WlSessionLock`) com uma superfície por tela, que confere a senha pelo PAM. O `hyprlock` fica de reserva.

## Para quê

Proteger a sessão em `Super + L`, na ociosidade e antes de suspender.

## Como funciona

- **Quem chama**: `Super + L`, o `lock_cmd` do hypridle e o `before_sleep_cmd` (via `loginctl lock-session`) chegam todos a `ipc call mare lock`, que fecha o painel aberto e chama `Lock.lock()`.
- **`LockScreen.qml`**: enquanto `Lock.locked`, cada tela ganha uma `LockSurface` e nada mais recebe teclado ou mouse.
- **`Lock.qml`**: o estado que todas as telas dividem (a senha digitada, a conversa com o PAM, os números da animação). O `PamContext` usa `config: "hyprlock"`.
- **O desenho** (`LockSurface.qml`): dois anéis numerados (minutos e segundos) giram em volta de um ponto fora da tela; o número que passa pela pílula acende em cobre, e o minuto e o segundo exatos aparecem ali ao passar; a hora grande em serifa e a data ao lado; embaixo, quem entra e a senha.
- **Ao desbloquear**: os anéis "dão corda" enquanto o PAM confere; se der certo, um clarão e a tela some; se não, a corda volta e o painel balança.
- **Extras**: indicador de Caps Lock (lido do LED do teclado, só enquanto bloqueado), layout ativo quando há mais de um, texto para conta travada (o `pam_faillock` avisa pela conversa), e o botão "suspender", que some quando `Settings.sleepMode` é "nada" ([[Energia e suspensão]]).
- **Ao desbloquear, o foco volta**: o `Lock.qml` guarda a janela que tinha o teclado ao bloquear e o devolve ao sair.

## Por quê assim

- **A pilha do PAM do hyprlock**: o comentário do código diz "a mesma pilha do hyprlock (auth include login), que vem com o Mk". O repositório não traz um arquivo de PAM próprio; o Maré usa o que o pacote `hyprlock` instala.
- **`allow_session_lock_restore`** no Hyprland: se o Maré cair com a tela bloqueada, o compositor **segura o bloqueio** e a opção deixa um Maré novo assumir.
- **Reserva em cadeia**: `qs … ipc call mare lock || pidof hyprlock || hyprlock -c …`. Se o Maré não responde, o hyprlock bloqueia.
- **Devolver o foco**: ao desbloquear, o Hyprland dá o foco à janela debaixo do ponteiro; com o ponteiro parado em cima da ilha não há nenhuma, e o que se digitava depois se perdia.
- **O `Esc` só larga depois de 3 s** de uma tentativa: para não servir de atalho por cima da espera do PAM numa senha errada.
- **Pedido de polkit com a tela bloqueada é cancelado**: ficaria escondido atrás do bloqueio ([[Polkit]]).
- O desenho segue uma *skin* de relógio do Ryoku, com os tokens do Mk.

## Onde está no código

`shell/mare/Lock.qml` (279 linhas), `LockScreen.qml`, `LockSurface.qml` (570 linhas); `config/hypr/mare/look.lua` (`allow_session_lock_restore`).

## Ligações

Depende de: [[Singletons]], [[IPC do Maré]], [[Hyprland em Lua]]. Chamado por: [[Ociosidade e bloqueio]], [[Atalhos]], [[Painéis]]. Relacionado: [[Polkit]], [[Energia e suspensão]].
