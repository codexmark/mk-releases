---
camada: "06 Shell Maré"
tags:
  - mk
  - maré
  - projetos
  - workspaces
fontes:
  - "shell/mare/Projects.qml"
  - "shell/mare/PickerPanel.qml"
  - "shell/mare/ProjectPanel.qml"
  - "shell/mare/scripts/mare-project"
  - "config/hypr/mare/binds.lua"
  - "README.md"
---

# Projetos e workspaces

No Mk, um workspace é um projeto: cada workspace pode ser ligado a um repositório git de `~/Work`, e a partir daí terminal, lazygit, lazydocker e yazi abrem na pasta dele.

## Para quê

É a ideia central do Mk como desktop de desenvolvimento: trocar de workspace é trocar de contexto de trabalho, e as ferramentas seguem.

## Como funciona

- **Descoberta**: `Projects.refreshDirs` roda `find ~/Work -mindepth 2 -maxdepth 4 -name .git` e lista as pastas que contêm um `.git` (até 3 níveis).
- **Ligação**: `Projects.bind(id, nome, dir)` grava em `workspaces.json`, no estado do Quickshell. Um workspace "livre" é ligado com `dir` vazio.
- **Projeto em foco**: `Projects.current` é a ligação do workspace focado.
- **Estado**: `scripts/mare-project <dir>` responde em JSON (`branch`, `dirty`, `runtime`, `containers`, `ports`), relido a cada 5 s enquanto há projeto em foco.
- **Abrir aqui**: os atalhos `Super + Enter/G/D/E` chamam `ipc call mare here <cmd>`, que vira `Projects.openHere(cmd)`: a linha do terminal escolhido, na pasta do projeto (ou na home, sem projeto).
- **Terminais**: `ghostty` (padrão, com a config do Mk), `kitty`, `foot`, `alacritty`, `wezterm`; a escolha fica em `apps.terminal` no `ajustes.json`. Se o escolhido foi desinstalado, abre o ghostty.
- **Letras gregas**: `Theme.wsName(id)` dá α, β, γ… para 1 a 10.
- A busca lista os projetos e leva ao workspace de cada um, ou abre num workspace novo.

## Por quê assim

- **`~/Work` como convenção**: o instalador cria a pasta vazia no skel; nada mais é configurado.
- **O terminal não tem padrão no XDG**: por isso a escolha fica no `ajustes.json` e quem a segue é o `Projects.qml`, com as flags de cada terminal escritas à mão. Só o ghostty leva a config do Mk.
- **`Super + Enter` nunca fica sem terminal**: a linha gerada testa `command -v` e cai no ghostty.
- **`wezterm start --always-new-process`**: sem isso a janela nasce num wezterm já aberto e o comando volta na hora; quem espera o fim (as atualizações) acharia que acabou.
- **A config do ghostty sai de `Settings.dir`**, não de `shellPath("../ghostty")`: o `shellPath` não sobe acima da pasta do shell.
- **Cor por projeto foi abandonada**: `Theme.accentFor()` devolve sempre o cobre; projetos se distinguem por nome e letra.

> [!note] O seletor só abre quando chamado
> Num workspace novo e vazio, quem convida a começar é a folha do fundo (`Backdrop.qml`). O seletor de projeto abre por `ipc call mare toggle picker`; o temporizador de `Projects.qml` só **fecha** um seletor que ficou aberto num workspace que ganhou janela. Uma versão anterior o abria sozinho ("o seletor não abre mais sozinho", no commit da direção Folha); o `README.md` e o cabeçalho do `PickerPanel.qml` já dizem o comportamento atual.

## Onde está no código

- `shell/mare/Projects.qml` (`bind`, `terminalLineFor`, `terminalLine`, `openHere`, `needsPicker`)
- `shell/mare/PickerPanel.qml`, `ProjectPanel.qml`, `Backdrop.qml` (o convite do workspace vazio)
- `shell/mare/scripts/mare-project`

## Ligações

Depende de: [[Singletons]], [[Atalhos]], [[Ghostty]]. Usado por: [[Painéis]], [[Busca e calculadora]], [[Ilha]], [[Apps padrão]].
