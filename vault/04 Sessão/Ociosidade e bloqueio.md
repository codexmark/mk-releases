---
camada: "04 Sessão"
tags:
  - mk
  - sessão
  - idle
  - hypridle
  - bloqueio
fontes:
  - "config/hypr/hypridle.conf"
  - "config/hypr/hyprlock.conf"
  - "config/hypr/mare/autostart.lua"
  - "shell/mare/SettingsSchema.js"
  - "shell/mare/Settings.qml"
  - "shell/mare/Lock.qml"
---

# Ociosidade e bloqueio

O `hypridle` vigia a ociosidade e dispara quatro passos: escurecer, bloquear, desligar a tela e dormir. Quem bloqueia é o Maré; o `hyprlock` fica de reserva.

## Para quê

Proteger a sessão quando o usuário se afasta e economizar tela e energia.

## Como funciona

Tempos de fábrica (`config/hypr/hypridle.conf`):

| Tempo | O que acontece | Comando |
|---|---|---|
| 4 min | escurece a tela | `brightnessctl -s set 15%` (volta com `-r`) |
| 5 min | bloqueia | `loginctl lock-session` |
| 6 min | desliga a tela | `hyprctl dispatch 'hl.dsp.dpms({ action = "off" })'` |
| 30 min | dorme | `~/.config/mk/shell/scripts/mare-sleep now` |

Bloco `general`:

- `lock_cmd = qs -p ~/.config/mk/shell ipc call mare lock || pidof hyprlock || hyprlock -c …/hyprlock.conf`
- `before_sleep_cmd = loginctl lock-session`
- `after_sleep_cmd`: religa a tela (DPMS).

**Qual arquivo vale.** O [[Autostart]] inicia `hypridle -c <raiz>/hypridle.conf` se esse arquivo existir; senão o de fábrica, em `<raiz>/hypr/hypridle.conf`. O da raiz é **gerado** pelos Ajustes (página Energia) por `hypridleConf()` em `SettingsSchema.js`: minutos 0 tiram o passo; sem brilho para mexer, o passo de escurecer some; o passo de dormir segue `power.sleep`.

Ao mudar os tempos, o `Settings.qml` regrava o arquivo e reinicia **só o hypridle do Mk**: `pkill -x -f "hypridle -c (<raiz>/hypridle.conf|<raiz>/hypr/hypridle.conf)"` e sobe outro com `setsid -f`.

## Por quê assim

- **O bloqueio é do Maré, com o hyprlock de reserva**: a cadeia `||` do `lock_cmd` garante que a tela bloqueia mesmo se o Maré não estiver rodando. O Maré usa a mesma pilha do PAM do hyprlock ([[Tela de bloqueio]]).
- **Arquivo gerado na raiz, não dentro de `hypr/`**: a pasta `hypr` é trocada inteira a cada atualização; o que o usuário escolheu precisa ficar fora ([[Pasta do usuário]]).
- **O passo de dormir de fábrica chama `mare-sleep now`**, e não `systemctl suspend`: o arquivo de fábrica não lê os Ajustes, e o instalador pode ter semeado "não dormir" ([[Energia e suspensão]]).
- **`pkill -x -f` com o padrão completo**: o `-x` exige que a linha de comando inteira case, então o `sh -c` que roda o `pkill` não se mata, e um hypridle de outro desktop na mesma conta não é tocado ([[Convenções]]).

> [!warning] Um hypridle vale para a máquina inteira
> Subir à mão um Hyprland aninhado com o `hyprland.lua` do Mk inicia um segundo `hypridle`, que baixa o brilho, bloqueia a sessão e suspende a máquina de verdade. O [[dev-shot]] não corre esse risco: não carrega o `autostart.lua`.

> [!note] hyprlock.conf fora da paleta
> O `hyprlock.conf` de reserva ainda usa cores e fontes de antes da [[Direção Folha]] (fundo grafite, Inter, JetBrains Mono).

## Onde está no código

- `config/hypr/hypridle.conf`, `config/hypr/hyprlock.conf`
- `shell/mare/SettingsSchema.js`: `hypridleConf`, `sleepStep`
- `shell/mare/Settings.qml`: `syncIdle`, `idleRestart`

## Ligações

Depende de: [[Autostart]], [[IPC do Maré]]. Usa: [[Tela de bloqueio]], [[Energia e suspensão]]. Configurado em: [[Páginas dos Ajustes]].
