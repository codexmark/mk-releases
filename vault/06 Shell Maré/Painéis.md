---
camada: "06 Shell Maré"
tags:
  - mk
  - maré
  - painéis
fontes:
  - "shell/mare/Mare.qml"
  - "shell/mare/Tide.qml"
  - "shell/mare/README.md"
  - "shell/mare/SessionPanel.qml"
  - "shell/mare/NetworkPanel.qml"
  - "shell/mare/CapturePanel.qml"
  - "shell/mare/Capture.qml"
  - "shell/mare/scripts/mare-shot"
---

# Painéis

O catálogo dos painéis que descem da ilha. Cada nome de `Mare.panels` é um arquivo `<Nome>Panel.qml`, carregado pelo `Tide` quando `Mare.mode` vira esse nome.

## Para quê

Cada painel é uma função do sistema ao alcance de um clique na ilha, de um atalho ou da busca.

## Como funciona

| Painel | Arquivo | O que faz |
|---|---|---|
| `project` | `ProjectPanel.qml` | o projeto em foco: branch, mudanças, containers, portas e atalhos ([[Projetos e workspaces]]) |
| `picker` | `PickerPanel.qml` | escolher um projeto de `~/Work` para o workspace, ou deixá-lo livre |
| `launcher` | `LauncherPanel.qml` | [[Busca e calculadora]] |
| `switcher` | `SwitcherPanel.qml` | Alt+Tab: cartões com miniatura ao vivo, na ordem de uso |
| `volume` | `VolumePanel.qml` | volume da saída padrão e de cada app que está tocando ([[Áudio]]) |
| `media` | `MediaPanel.qml` | tocando agora: capa, faixa, progresso, controles; troca entre players |
| `power` | `PowerPanel.qml` | bateria (saúde, ciclos, limite de carga) e perfil de energia |
| `network` | `NetworkPanel.qml` | conexão, IP/gateway/DNS, tráfego, ping, escolha de DNS, QR do Wi-Fi, redes ao redor ([[Rede]]) |
| `bluetooth` | `BluetoothPanel.qml` | adaptador, aparelhos, pareamento ([[Bluetooth]]) |
| `calendar` | `CalendarPanel.qml` | o mês (semana começando no domingo) e o clima |
| `notifications` | `NotificationsPanel.qml` | histórico e não perturbe ([[Notificações e OSD]]) |
| `session` | `SessionPanel.qml` | bloquear, suspender, sair, reiniciar, desligar |
| `display` | `DisplayPanel.qml` | brilho e luz noturna |
| `system` | `SystemPanel.qml` | CPU, memória, temperatura, rede, disco, com o último minuto em gráfico; atalho para o [[MkPulso]], o monitor do Mk (antes era o btop) |
| `clipboard` | `ClipboardPanel.qml` | histórico da área de transferência (`cliphist`) |
| `wallpaper` | `WallpaperPanel.qml` | [[Papéis de parede]], 12 por página |
| `capture` | `CapturePanel.qml` | capturar área, janela, tela, ou gravar um vídeo |
| `emoji` | `EmojiPanel.qml` | [[Emoji]] |
| `settings` | (`SettingsWindow.qml`) | os [[Ajustes]]: não desce da ilha |

Detalhes que valem registro:

- **Sessão**: bloquear e suspender agem na hora; sair, reiniciar e desligar pedem um segundo clique (o botão vira "confirmar" por 3 segundos). O botão de suspender some quando Ajustes › Energia diz "não dormir".
- **Captura**: as capturas rodam `mare-shot` **depois** de o painel fechar, para ele não sair na foto. Salvam em `<Imagens>/Capturas`, copiam e avisam. A gravação usa o `gpu-screen-recorder` com o som do sistema; enquanto grava, a ilha mostra o tempo e um clique nela para. Antes, uma sondagem (só a ajuda do gravador) diz se dá para gravar.
- **Projeto**: só abre num workspace ligado a um projeto e não marcado como livre; fora disso `Mare.mode` vira `"project"` mas o `Tide` o trata como `"rest"`.
- **Seletor (`picker`)**: só abre quando chamado ([[Projetos e workspaces]]).

## Por quê assim

- **Um arquivo por painel, com `implicitWidth`/`implicitHeight` na raiz**: é só disso que o `Tide` precisa para desenhar a forma.
- **Confirmação em dois cliques** no que não tem volta (sair, desligar, esquecer um aparelho, remover uma impressora), em vez de um diálogo.
- **Peças comuns**: `PanelHeader` (número em cobre, título em serifa, linha guia, informação em mono), `PillButton`, `MkSlider`, `MkToggle`, `Sparkline`, `NotifIcon`.
- Ideias dos painéis de rede e bateria vieram da barra do Omarchy, "refeitas no estilo do Maré".

> [!note] Teste automático da estrutura
> `tests/qml/estrutura.test.sh` confere que cada nome de `Mare.panels` tem `case` no `Tide` e arquivo próprio.

## Onde está no código

`shell/mare/*Panel.qml`, `Mare.qml` (a lista `panels`), `Tide.qml` (o `switch` e os `Component`).

## Ligações

Depende de: [[Painel líquido (Tide)]], [[Singletons]], [[Scripts mare]]. Abertos por: [[Ilha]], [[Atalhos]], [[Busca e calculadora]], [[IPC do Maré]].
