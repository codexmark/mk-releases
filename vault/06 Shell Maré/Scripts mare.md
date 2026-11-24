---
camada: "06 Shell Maré"
tags:
  - mk
  - maré
  - scripts
  - bash
fontes:
  - "shell/mare/README.md"
  - "shell/mare/scripts/mare-start"
  - "shell/mare/scripts/mare-updates"
  - "shell/mare/scripts/mare-sleep"
  - "shell/mare/scripts/mare-monitors"
  - "shell/mare/scripts/mare-printers"
  - "shell/mare/scripts/mare-vpn"
  - "shell/mare/scripts/mare-default-apps"
---

# Scripts mare

A pasta `shell/mare/scripts/` tem os `mare-*`: scripts bash que o shell roda por fora. A regra é que o QML não lê `/proc` nem conversa com `nmcli`: chama um script, que responde em JSON (ou em linhas).

## Para quê

Manter no QML só desenho e estado, e no bash tudo o que é leitura do sistema ou ação nele.

## Como funciona

| Script | O que faz | Quem chama |
|---|---|---|
| `mare-start` | inicia o Maré com a escala dos Ajustes; `--restart` | autostart, `Settings.qml`, `mk-update`, `mk reiniciar-mare` |
| `mare-look` | ícones, cursor e fontes por `gsettings` | autostart |
| `mare-caps` | `{"touchpad","bateria","brilho","sono"}`: o que a máquina tem | `Settings.qml` |
| `mare-sleep` | `status` e `now` | `mare-caps`, hypridle, instalador |
| `mare-about` | detalhes da máquina | página Sobre |
| `mare-stats` | CPU, memória, rede, temperatura, disco | `Stats.qml`, a cada 2 s |
| `mare-project` | branch, mudanças, runtime, containers, portas | `Projects.qml`, a cada 5 s |
| `mare-weather` | clima via Open-Meteo, com onde é (`city`, `country`, `lat`, `lon`) para o carimbo do fundo | `Weather.qml`, a cada 30 min |
| `mare-cava` | o espectro do som | `Sound.qml` |
| `mare-palette` | a cor principal de uma imagem (matugen) | `Wallpaper.qml` |
| `mare-battery` | ciclos, saúde, limite de carga | `PowerPanel.qml` |
| `mare-net`, `mare-dns`, `mare-wifi`, `mare-wifi-qr` | a conexão, o DNS, as redes guardadas, o QR | `NetworkPanel.qml` |
| `mare-vpn` | VPNs do NetworkManager | `Vpn.qml`, `SettingsVpnPage.qml` |
| `mare-shot` | captura de tela | `Capture.qml` |
| `mare-emoji` | `pick`, `tem-wtype`, `gerar` | `EmojiPanel.qml` |
| `mare-monitors` | `list`, `trial`, `keep`, `revert` | `Monitors.qml` |
| `mare-default-apps` | `list`, `set`, `terminal`, `open` | `SettingsAppsPage.qml`, `binds.lua` |
| `mare-devtools` | roda o `mise-tools.sh` num terminal | `Welcome.qml`, `mk ferramentas` |
| `mare-gamepad` | o controle Xbox e o xone | `SettingsGamepadPage.qml` |
| `mare-printers` | impressoras do CUPS | `SettingsPrintersPage.qml` |
| `mare-updates` | procura atualizações | `Updates.qml`, `bin/mk-update` |

Do QML, a chamada é sempre `bash` + `Quickshell.shellPath("scripts/mare-...")`. O cabeçalho de cada script traz o formato exato da saída.

## Por quê assim

- **Nenhum usa sudo.** Só o `mare-printers` pede a senha de administrador, pelo `pkexec`, quando o CUPS recusa uma ação. O shell roda como o usuário e não deve ter um caminho silencioso para root.
- **JSON na saída**: o QML faz `JSON.parse` e pronto; o script pode ser testado sozinho, comparando texto.
- **Raízes trocáveis para teste**: `PROC`, `SYS`, `ETC`, `MK_ROOT` (no `mare-updates`), `AJUSTES`, `MARE_HYPRCTL`, `MARE_DRM`, `MARE_VPN_FIXTURES`, `MARE_PRINTERS_FIXTURES`, `--dry-run` no `mare-default-apps`. Um script que lê `/proc` ou `/sys` recebe a raiz por variável; "é a emenda mínima que o deixa testável" ([[Testes e TDD]]).
- **Clima sem conta e sem chave** (Open-Meteo); a cidade vem de `~/.config/mare/city` ou do fuso horário do sistema.
- **Segredos em `$XDG_RUNTIME_DIR`**, só para o dono: o QR do Wi-Fi e o arquivo que o `mare-vpn` importa ([[Convenções]]).
- **`mare-updates` nunca pede senha**: o clone do canal roda com `GIT_TERMINAL_PROMPT=0` e `credential.interactive=never`; num canal sem acesso, só avisa.

> [!note] Testes automáticos
> Cada script desta pasta tem o seu arquivo em `tests/scripts/`. Além disso, `tests/docs/docs.test.sh` exige que todo script desta pasta tenha uma linha no `shell/mare/README.md`, e `tests/qml/estrutura.test.sh`, que todo script chamado pelo QML exista.

## Onde está no código

`shell/mare/scripts/` (24 arquivos) e a tabela "A pasta `scripts/`" do `shell/mare/README.md`.

## Ligações

Depende de: [[Maré]]. Usado por: [[Singletons]], [[Painéis]], [[Páginas dos Ajustes]], [[mk-update]], [[Instalador passo a passo]].
