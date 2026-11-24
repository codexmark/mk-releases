---
camada: "05 Compositor"
tags:
  - mk
  - compositor
  - sessão
  - autostart
fontes:
  - "config/hypr/mare/autostart.lua"
  - "shell/mare/scripts/mare-start"
  - "shell/mare/scripts/mare-look"
---

# Autostart

`autostart.lua` diz o que sobe junto com a sessão: sete processos independentes, iniciados no evento `hyprland.start`.

## Para quê

É a ponte entre o compositor e tudo o que roda em volta dele.

## Como funciona

| Comando | Para quê |
|---|---|
| `bash '<shell>/scripts/mare-start'` | o [[Maré]], na escala escolhida nos Ajustes |
| `hypridle -c '<arquivo>'` | ociosidade; `<raiz>/hypridle.conf` se existir, senão o de fábrica ([[Ociosidade e bloqueio]]) |
| `wl-clip-persist --clipboard regular` | o que foi copiado sobrevive ao fechar o app de origem |
| `wl-paste --type text --watch cliphist store` | histórico da área de transferência (texto) |
| `wl-paste --type image --watch cliphist store` | idem (imagens) |
| `bash '<shell>/scripts/mare-look'` | ícones, cursor e fontes dos apps por `gsettings` ([[Tema GTK, ícones e Qt]]) |
| `udiskie --automount --no-tray` | monta pendrives e discos removíveis |

**`mare-start`** lê `"ui.scale"` de `<raiz>/ajustes.json` (sem valor válido, 0.85) e roda `QT_SCALE_FACTOR=<escala> qs -p <shell>`, com a saída em `~/.cache/mk/mare.log`. Com `--restart`, fecha o Maré em execução (`qs kill -p`), inicia outro e devolve o foco à janela ativa duas vezes, pouco depois de subir.

## Por quê assim

- **Tudo pelo caminho da instalação** (`paths.shell`, `paths.root`): "para a sessão não depender de `~/.config/hypr` nem de `~/.config/quickshell`, que podem ser de outro desktop".
- **`mare-start` em vez de `qs` direto**: o tamanho da interface é a escala do Qt, que só vale num processo novo; o script é o único lugar que a aplica, e o autostart, os Ajustes, o `mk-update` e o `mk reiniciar-mare` passam por ele.
- **Devolver o foco no `--restart`**: quando as camadas do Maré renascem, a janela em uso fica sem o teclado até um clique. Foi visto no meio do `mk-update`, na hora da senha do sudo, que não chegava ao terminal.
- **O que saiu daqui**: duas linhas foram removidas na correção das issues da primeira instalação em máquina real, `systemctl --user start hyprpolkitagent` e `gnome-keyring-daemon --start --components=secrets,pkcs11`. Hoje o Maré é o agente de polkit ([[Polkit]]) e o chaveiro não é iniciado pelo Mk ([[Chaveiro e agente SSH]]).
- **Sem barra, launcher ou daemon de notificações**: o Maré é os três.

## Onde está no código

`config/hypr/mare/autostart.lua` (22 linhas); `shell/mare/scripts/mare-start`.

## Ligações

Depende de: [[Hyprland em Lua]], [[Pasta do usuário]]. Inicia: [[Maré]], [[Ociosidade e bloqueio]]. Parte do [[Fluxo de boot]].
