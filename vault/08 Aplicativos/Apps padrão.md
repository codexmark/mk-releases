---
camada: "08 Aplicativos"
tags:
  - mk
  - aplicativos
  - mime
  - xdg
fontes:
  - "config/mimeapps.list"
  - "shell/mare/scripts/mare-default-apps"
  - "shell/mare/SettingsAppsPage.qml"
  - "migrations/user/20261125-apps-padrao.sh"
  - "migrations/system/20261125-apps-padrao-skel.sh"
  - "config/hypr/mare/binds.lua"
---

# Apps padrão

Que app abre cada tipo de coisa é decidido pelo `~/.config/mimeapps.list`, o arquivo padrão do XDG. O Mk entrega um de fábrica e uma página dos Ajustes para trocar.

## Para quê

Duplo clique num PDF abre o leitor de PDF; um link abre o navegador; `Super + B` abre o navegador escolhido, seja qual for.

## Como funciona

**De fábrica** (`config/mimeapps.list`):

| Tipo | App | Reserva na mesma linha |
|---|---|---|
| páginas (`http`, `https`, `text/html`) | Chromium | |
| pastas | Nautilus | |
| texto, markdown, shell script | Editor de texto do GNOME | |
| PDF | Papers | Evince |
| imagens | Loupe | imv |
| vídeo | mpv | |
| música | Amberol | mpv |

Onde há dois apps na linha, vale o primeiro que estiver instalado.

**A página** (Ajustes › Apps padrão) usa `mare-default-apps`:

- `list`: os papéis (`navegador`, `arquivos`, `editor`, `imagens`, `video`, `musica`, `pdf`, `email`), o app de hoje e os instalados que servem;
- `set <papel> <app.desktop>`: `xdg-mime default` para o grupo de tipos do papel;
- `terminal <nome>`: põe o terminal no alto de `xdg-terminals.list`;
- `open <papel>`: abre o app padrão do papel, sem arquivo. É o que `Super + B` chama (`open navegador || chromium`).

Cada papel tem tipos "core", que sempre mudam, e "extra", que só mudam se o app declarar no `.desktop` que os abre.

**O terminal** é a exceção: a escolha fica em `apps.terminal` no `ajustes.json`, e quem a segue é o `Projects.qml` ([[Projetos e workspaces]]).

## Por quê assim

- **`mimeapps.list`, não um formato do Mk**: é "onde todo app procura". Depois de instalado, o arquivo é do usuário.
- **Quem já tem um `mimeapps.list` fica com o seu**: o instalador o põe no skel; a migração de usuário só instala se o arquivo não existe.
- **O segundo app da linha** é o que o Mk trazia antes (imv, evince) ou o mpv, "que toca tudo": quem atualizou sem aceitar os apps novos continua com um padrão que funciona.
- **Tipos "extra" condicionais**: "escolher o imv para imagens não entrega a ele um formato que ele não lê".
- **O terminal fora do XDG**: "não há padrão do XDG para ele". O `xdg-terminals.list` é ajustado para o `xdg-terminal-exec`; sem este, um app de terminal (Neovim, Yazi) não abre a partir do gerenciador de arquivos; o `list` devolve `terminalExec` para a página saber.
- **`--dry-run`**: o script pode ser exercitado sem tocar nos padrões de ninguém.

## Onde está no código

`config/mimeapps.list`; `shell/mare/scripts/mare-default-apps` (336 linhas); `shell/mare/SettingsAppsPage.qml`.

## Ligações

Depende de: [[Apps oferecidos]], [[Usuários e sudo]]. Usado por: [[Páginas dos Ajustes]], [[Atalhos]], [[Projetos e workspaces]].
