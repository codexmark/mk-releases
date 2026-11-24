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
---

# Energia e suspensão

O Mk tem dois jeitos de dormir: **suspender** ou **não dormir**. Uma escolha só (`power.sleep`) vale no hypridle, no painel de sessão e na tela de bloqueio, e uma placa de vídeo conhecida por não voltar da suspensão muda o padrão.

## Para quê

Evitar que a máquina durma de um jeito do qual não volta.

## Como funciona

- **`mare-sleep status`** responde `{"canSuspend": bool, "risky": [...]}`:
  - `canSuspend`: o que o logind aceita (`CanSuspend` por `busctl`; respeita o `sleep.conf`); sem logind, olha `/sys/power/state`;
  - `risky`: placas de vídeo (classe PCI `0x03`) que sabidamente não voltam do S3. Hoje, a **AMD Navi 10** (RX 5600/5700), reconhecida pelo vendor `0x1002` e uma lista de device ids.
- **`mare-sleep now`** lê `power.sleep` do `ajustes.json`: com `nada` (ou um `hibernar` antigo) não faz nada; senão, se `canSuspend`, `systemctl suspend`.
- **`mare-caps`** inclui a resposta inteira do `mare-sleep` na chave `sono`; o `Settings.qml` a guarda em `caps`.
- **`sleepMode()`** (em `SettingsSchema.js`) resolve o que vale de fato: suspender com o suspend proibido vira "nada".
- **Quem segue `sleepMode`**: o passo de dormir do [[Ociosidade e bloqueio|hypridle]], o botão "suspender" do painel de sessão (some com "não dormir") e a tela de bloqueio.
- **Ajustes › Energia**: a linha "Ao dormir" e, abaixo, o `SettingsSleepBlock.qml` com os avisos: placa de risco (só com suspender escolhido) e suspend proibido no sistema.
- **Instalação nova com Navi 10**: o instalador roda `mare-sleep status` e, se houver item em `risky`, semeia `/etc/skel/.config/mk/ajustes.json` com `"power.sleep": "nada"`.

Perfis de energia (economia, equilibrado, desempenho) são do `power-profiles-daemon`, mostrados no painel de bateria; `upower` e `brightnessctl` completam a lista.

## Por quê assim

- **A Navi 10**: "a SMU da amdgpu falha no resume, às vezes" (com a referência ao issue do driver no cabeçalho do script). A tela fica preta e só reiniciando. É um defeito do driver, não do Mk; o que o Mk faz é não cair nele por padrão.
- **Semear só na instalação nova**: "quem já usa o Mk vê o aviso em Ajustes › Energia e escolhe". O sistema não muda a escolha de ninguém por baixo.
- **O aviso não muda a escolha**: o `SettingsSleepBlock` "só diz a saída".
- **Uma escolha, três lugares**: o esquema declara "uma escolha só, usada em todo lugar". Com `sleepMode` num ponto só, o painel de sessão não oferece o que o hypridle não faria.
- **Sem hibernar**: veja [[Swap e hibernação]].
- **`SYS`, `PROC`, `AJUSTES`, `LOGIND`** trocam as raízes: o script foi feito para ser testado sem tocar na máquina ([[Testes e TDD]]).

> [!warning] Suspensão e VM
> A tela da VM pode não voltar depois de suspender. Nada sobre suspensão deve ser concluído a partir da VM.

## Onde está no código

- `shell/mare/scripts/mare-sleep`, `mare-caps`
- `shell/mare/SettingsSchema.js`: `sleepMode`, `sleepCommand`
- `shell/mare/SettingsSleepBlock.qml`, `SessionPanel.qml`
- `installer/mk-install`: `install_desktop` (o bloco da Navi 10)

## Ligações

Depende de: [[Serviços systemd]], [[Microcódigo e perfis de hardware]]. Usado por: [[Ociosidade e bloqueio]], [[Painéis]], [[Tela de bloqueio]], [[Páginas dos Ajustes]]. Relacionado: [[Swap e hibernação]].
