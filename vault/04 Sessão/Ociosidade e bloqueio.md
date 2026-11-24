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
  - "shell/mare/ScreenSaver.qml"
  - "tests/config/hypridle.conf.test.sh"
---

# Ociosidade e bloqueio

O `hypridle` vigia a ociosidade e dispara os passos de quem se afasta: descanso de tela, escurecer, bloquear, desligar a tela e, só para quem escolheu, dormir. Quem bloqueia e quem desenha o descanso é o Maré; o `hyprlock` fica de reserva.

## Para quê

Proteger a sessão quando o usuário se afasta e poupar a tela, sem suspender a máquina por conta própria.

## Como funciona

Tempos de fábrica (`config/hypr/hypridle.conf`):

| Tempo | O que acontece | Comando |
|---|---|---|
| 3 min | [[Descanso de tela]] | `qs -p ~/.config/mk/shell ipc call mare saver on` (volta com `saver off`) |
| 4 min | escurece a tela | `brightnessctl -s set 15%` (volta com `-r`) |
| 5 min | bloqueia | `loginctl lock-session` |
| 30 min | desliga a tela | `hyprctl dispatch 'hl.dsp.dpms({ action = "off" })'` |
| 30 min | dorme, **só para quem escolheu suspender** | `~/.config/mk/shell/scripts/mare-sleep now` |

Bloco `general`:

- `lock_cmd = qs -p ~/.config/mk/shell ipc call mare lock || pidof hyprlock || hyprlock -c …/hyprlock.conf`
- `before_sleep_cmd = loginctl lock-session`
- `after_sleep_cmd`: religa a tela (DPMS).

**Qual arquivo vale.** O [[Autostart]] inicia `hypridle -c <raiz>/hypridle.conf` se esse arquivo existir; senão o de fábrica, em `<raiz>/hypr/hypridle.conf`. O da raiz é **gerado** pelos Ajustes (página Energia) por `hypridleConf()` em `SettingsSchema.js`, a partir de `idle.dim`, `idle.saver`, `idle.lock`, `idle.off` e `idle.suspend`: minutos 0 tiram o passo; sem brilho para mexer, o passo de escurecer some; o passo de dormir segue `power.sleep`. Os minutos passam pelo `valueOf` do esquema, então um valor torto no `ajustes.json` vale o de fábrica ([[Ajustes]]).

Ao mudar os tempos, o `Settings.qml` regrava o arquivo e reinicia **só o hypridle do Mk**: `pkill -x -f "hypridle -c (<raiz>/hypridle.conf|<raiz>/hypr/hypridle.conf)"` e sobe outro com `setsid -f`.

## Por quê assim

- **Descanso de tela no lugar de dormir**: de fábrica o Mk não suspende sozinho; o descanso "faz o papel que era do suspender", e a tela apaga aos 30 minutos ([[D16 Descanso de tela no lugar do suspender]]).
- **O descanso cede ao bloqueio**: aos 5 minutos a tela é bloqueada, o descanso desliga, e daí em diante quem aparece é a tela de bloqueio, até a tela apagar. Com "Bloquear" em 0 ele fica até a tela apagar.
- **O bloqueio é do Maré, com o hyprlock de reserva**: a cadeia `||` do `lock_cmd` garante que a tela bloqueia mesmo se o Maré não estiver rodando. O Maré usa a mesma pilha do PAM do hyprlock ([[Tela de bloqueio]]).
- **Arquivo gerado na raiz, não dentro de `hypr/`**: a pasta `hypr` é trocada inteira a cada atualização; o que o usuário escolheu precisa ficar fora ([[Pasta do usuário]]).
- **O passo de dormir de fábrica chama `mare-sleep now`**, e não `systemctl suspend`: o arquivo de fábrica não lê os Ajustes, e o script só dorme se `power.sleep` for "suspender" ([[Energia e suspensão]]).
- **`pkill -x -f` com o padrão completo**: o `-x` exige que a linha de comando inteira case, então o `sh -c` que roda o `pkill` não se mata, e um hypridle de outro desktop na mesma conta não é tocado ([[Convenções]]).

> [!warning] Um hypridle vale para a máquina inteira
> Subir à mão um Hyprland aninhado com o `hyprland.lua` do Mk inicia um segundo `hypridle`, que baixa o brilho e bloqueia a sessão de verdade. O [[dev-shot]] não corre esse risco: não carrega o `autostart.lua`.

> [!note] Visto só por IPC
> O descanso foi visto ligar e desligar por IPC na sessão aninhada; a espera real de 3 minutos pelo hypridle, dois monitores e o consumo com ele horas na tela não foram vistos ([[Limitações conhecidas]]). O `hyprlock.conf` de reserva ainda usa cores e fontes de antes da [[Direção Folha]].

## Onde está no código

- `config/hypr/hypridle.conf`, `config/hypr/hyprlock.conf`
- `shell/mare/SettingsSchema.js`: as linhas `idle.*`, `hypridleConf`, `sleepStep`
- `shell/mare/Settings.qml`: `syncIdle`, `idleRestart`
- `tests/config/hypridle.conf.test.sh`

## Ligações

Depende de: [[Autostart]], [[IPC do Maré]]. Usa: [[Descanso de tela]], [[Tela de bloqueio]], [[Energia e suspensão]]. Configurado em: [[Páginas dos Ajustes]].
