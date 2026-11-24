---
camada: "06 Shell Maré"
tags:
  - mk
  - maré
  - ajustes
  - páginas
fontes:
  - "shell/mare/SettingsSchema.js"
  - "shell/mare/SettingsWindow.qml"
  - "shell/mare/SettingsMonitorsPage.qml"
  - "shell/mare/SettingsAppsPage.qml"
  - "shell/mare/SettingsPrintersPage.qml"
  - "shell/mare/SettingsGamepadPage.qml"
  - "shell/mare/SettingsVpnPage.qml"
  - "shell/mare/SettingsKeybindsPage.qml"
  - "shell/mare/scripts/mare-gamepad"
  - "README.md"
---

# Páginas dos Ajustes

As 17 páginas dos Ajustes, em cinco grupos. A chave de cada uma abre direto nela: `mk ajustes monitores`.

## Para quê

Referência do que se configura onde, e de quais páginas são só linhas do esquema e quais têm arquivo próprio.

## Como funciona

| Nº | Grupo | Página | Chave | Feita de | O que tem |
|---|---|---|---|---|---|
| 01 | Aparência | Ilha | `ilha` | linhas | opacidade em repouso, responder ao som, mostrar sistema e clima |
| 02 | Aparência | Fundo | `fundo` | linhas (+ extra) | letra do workspace, escurecer o papel de parede, grão |
| 03 | Aparência | Movimento | `movimento` | linhas | velocidade: rápido, normal, calmo, sem ([[Animações]]) |
| 04 | Aparência | Notificações | `notificacoes` | linhas (+ extra) | tempo do aviso na tela |
| 05 | Janelas | Janelas | `janelas` | linhas | espaços, borda, cantos, opacidade das inativas |
| 06 | Dispositivos | Tela | `tela` | linhas | tamanho da interface, escala, luz noturna |
| 07 | Dispositivos | Monitores | `monitores` | `SettingsMonitorsPage.qml` | [[Monitores]] |
| 08 | Dispositivos | Som | `som` | conteúdo da `SettingsWindow` | saída e microfone |
| 09 | Dispositivos | Teclado e mouse | `entrada` | linhas | layout, Caps Lock, repetição, sensibilidade, touchpad |
| 10 | Dispositivos | Energia | `energia` | linhas + `SettingsSleepBlock.qml` | descanso de tela, escurecer, bloquear, desligar a tela; bateria fraca; "Ao dormir", de fábrica "não dormir" ([[Ociosidade e bloqueio]], [[Energia e suspensão]]) |
| 11 | Dispositivos | Impressoras | `impressoras` | `SettingsPrintersPage.qml` | [[Impressão]] |
| 12 | Dispositivos | Controle | `controle` | `SettingsGamepadPage.qml` | o controle Xbox e o driver xone |
| 13 | Rede | VPN | `vpn` | `SettingsVpnPage.qml` | WireGuard e OpenVPN ([[Rede]]) |
| 14 | Sistema | Apps padrão | `apps` | `SettingsAppsPage.qml` | [[Apps padrão]] e o terminal |
| 15 | Sistema | Atualizações | `atualizacoes` | linhas + `SettingsUpdatesLast.qml` | [[Atualizações na ilha e nos Ajustes]] |
| 16 | Sistema | Atalhos | `atalhos` | `SettingsKeybindsPage.qml` | [[Atalhos do usuário]] |
| 17 | Sistema | Sobre o Mk | `sobre` | conteúdo da `SettingsWindow` | a máquina (`mare-about`) e onde os ajustes ficam |

**Página com arquivo próprio**: `SettingsWindow.pageFiles` mapeia a chave para o arquivo; o `Loader` `pageLoader` carrega `<nome>.qml` só enquanto a página está aberta. Linhas do esquema com a mesma `page` são desenhadas antes do conteúdo do arquivo. Se a raiz declarar `property string flashKey`, recebe a chave que a busca pediu, para destacar a linha.

**A busca** (`search()` no esquema) procura sem acento e sem caixa em páginas, linhas (incluindo `kw`) e atalhos; devolve até 9 resultados.

Chaves do Hyprland que os Ajustes mudam: `general.gaps_in`, `general.gaps_out`, `general.border_size`, `decoration.rounding`, `decoration.inactive_opacity`, `input.kb_layout`, `input.kb_options`, `input.repeat_rate`, `input.repeat_delay`, `input.sensitivity`, `input.touchpad.natural_scroll`, `input.touchpad.tap_to_click`, as animações e a escala padrão dos monitores.

## Por quê assim

- **Duas escalas diferentes na página Tela**: "Tamanho da interface" (`ui.scale`) é a escala do Qt, vale só para o Maré e reinicia o shell; "Escala da tela" (`hypr.scale`) é a do Hyprland, para telas sem ajuste próprio em Monitores. Separá-las deixa encolher a ilha sem encolher os apps.
- **Controle (xone)**: o kernel traz o `xpad`, que cuida dos botões de um controle Xbox; o fone plugado no controle não vira dispositivo de som. O `xone-dkms` (AUR) resolve, mas é um módulo fora do kernel: recompila a cada kernel novo, marca o kernel como *tainted* e baixa um firmware de licença própria. "Por isso é uma escolha sua, não vem instalado": a página só oferece, num terminal, "com os custos ditos".
- **`kb_options` e animações só entram no Lua quando mexidos** (`ours()`): com tudo de fábrica, o que a pessoa pôs num `user.lua` fica em paz.
- **Alt+Shift troca de layout** só quando há dois layouts: a opção `grp:alt_shift_toggle` é montada a partir da escolha.

> [!warning] Nunca testadas em hardware
> Monitores (mais de um), Impressoras, VPN (importar) e Controle (instalar o xone) existem e nunca foram vistas funcionando numa máquina de verdade ([[Limitações conhecidas]]).

## Onde está no código

`shell/mare/SettingsSchema.js` (`groups`, `blurbs`, `rows`), `shell/mare/SettingsWindow.qml`, `shell/mare/Settings*Page.qml`.

## Ligações

Depende de: [[Ajustes]]. Cada página liga a: [[Monitores]], [[Impressão]], [[Rede]], [[Apps padrão]], [[Atalhos do usuário]], [[Energia e suspensão]], [[Ociosidade e bloqueio]], [[Animações]], [[Atualizações na ilha e nos Ajustes]].
