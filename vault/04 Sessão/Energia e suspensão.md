---
camada: "04 Sessão"
tags:
  - mk
  - sessão
  - energia
  - suspensão
  - hardware
fontes:
  - "shell/mare/scripts/mare-sleep"
  - "shell/mare/scripts/mare-caps"
  - "shell/mare/SettingsSchema.js"
  - "shell/mare/SettingsSleepBlock.qml"
  - "shell/mare/SessionPanel.qml"
  - "installer/mk-install"
  - "docs/limites.md"
  - "config/hypr/hypridle.conf"
---

# Energia e suspensão

De fábrica o Mk **não dorme**: no lugar fica o descanso de tela, e a tela apaga aos 30 minutos. Suspender é uma escolha em Ajustes › Energia (`power.sleep`), e a mesma escolha vale no hypridle, no painel de sessão e na tela de bloqueio.

## Para quê

Evitar que a máquina durma de um jeito do qual não volta, sem tirar a suspensão de quem a quer.

## Como funciona

- **`power.sleep`**: `"nada"` (de fábrica, "não dormir") ou `"suspender"`. Com suspender, aparece a linha "Dormir" (`idle.suspend`, 30 minutos de fábrica).
- **`mare-sleep now`** lê `power.sleep` do `ajustes.json`: só com "suspender", e se a máquina puder, roda `systemctl suspend`. Com "nada", sem a chave, ou com um "hibernar" antigo, não faz nada.
- **`mare-sleep status`** responde `{"canSuspend": bool, "risky": [...]}`:
  - `canSuspend`: o que o logind aceita (`CanSuspend` por `busctl`; respeita o `sleep.conf`); sem logind, olha `/sys/power/state`;
  - `risky`: placas de vídeo (classe PCI `0x03`) que sabidamente não voltam do S3. Hoje, a **AMD Navi 10** (RX 5600/5700), reconhecida pelo vendor `0x1002` e uma lista de device ids.
- **`mare-caps`** inclui a resposta inteira do `mare-sleep` na chave `sono`; o `Settings.qml` a guarda em `caps`.
- **`sleepMode()`** (em `SettingsSchema.js`) resolve o que vale de fato: suspender com o suspend proibido vira "nada"; um valor que não é nenhuma das opções (`orphan`) também.
- **Quem segue `sleepMode`**: o passo de dormir do [[Ociosidade e bloqueio|hypridle]], o botão "suspender" do painel de sessão (só existe com suspender escolhido) e o da tela de bloqueio.
- **Ajustes › Energia**: a linha "Ao dormir" ("De fábrica o Mk não dorme: fica o descanso de tela") e, abaixo, o `SettingsSleepBlock.qml` com os avisos: placa de risco (só com suspender escolhido) e suspend proibido no sistema.
- **Instalação nova com Navi 10**: o instalador ainda semeia `"power.sleep": "nada"` no skel quando o `mare-sleep status` traz um item em `risky`; com o padrão novo, isso só repete o valor de fábrica.

Perfis de energia (economia, equilibrado, desempenho) são do `power-profiles-daemon`, mostrados no painel de bateria; `upower` e `brightnessctl` completam a lista.

## Por quê assim

- **Não dormir é o padrão**: decisão de quem mantém, com a Navi 10 por trás. Detalhes em [[D16 Descanso de tela no lugar do suspender]].
- **A Navi 10**: "a SMU da amdgpu falha no resume, às vezes" (com a referência ao issue do driver no cabeçalho do script). A tela fica preta e só reiniciando. É um defeito do driver, não do Mk.
- **O aviso não muda a escolha**: o `SettingsSleepBlock` "só diz a saída" a quem escolher suspender numa máquina de risco.
- **Uma escolha, três lugares**: o esquema declara "uma escolha só, usada em todo lugar". Com `sleepMode` num ponto só, o painel de sessão não oferece o que o hypridle não faria.
- **Sem hibernar**: veja [[Swap e hibernação]].
- **`SYS`, `PROC`, `AJUSTES`, `LOGIND`** trocam as raízes: o script foi feito para ser testado sem tocar na máquina ([[Testes e TDD]]).

> [!warning] Suspensão e VM
> A tela da VM pode não voltar depois de suspender. Nada sobre suspensão deve ser concluído a partir da VM.

## Onde está no código

- `shell/mare/scripts/mare-sleep`, `mare-caps`
- `shell/mare/SettingsSchema.js`: a linha `power.sleep`, `sleepMode`, `sleepCommand`, `orphan`
- `shell/mare/SettingsSleepBlock.qml`, `SessionPanel.qml`
- `installer/mk-install`: `install_desktop` (o bloco da Navi 10)

## Ligações

Depende de: [[Serviços systemd]], [[Microcódigo e perfis de hardware]]. Usado por: [[Ociosidade e bloqueio]], [[Painéis]], [[Tela de bloqueio]], [[Páginas dos Ajustes]]. Relacionado: [[Descanso de tela]], [[Swap e hibernação]].
