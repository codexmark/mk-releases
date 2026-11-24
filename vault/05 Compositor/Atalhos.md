---
camada: "05 Compositor"
tags:
  - mk
  - compositor
  - atalhos
  - teclado
fontes:
  - "config/hypr/mare/binds.lua"
  - "shell/mare/Keybinds.js"
  - "bin/check-keybinds"
  - "README.md"
---

# Atalhos

Os atalhos são **uma tabela de ações** em `binds.lua`: cada linha tem um id, um grupo, uma descrição, o combo de fábrica e o que faz. A função `apply()` transforma a tabela em `hl.bind`.

## Para quê

Uma lista única serve para criar os binds, para os Ajustes mostrarem e trocarem cada um, para a busca achar atalhos e para as boas-vindas ensinarem as teclas certas.

## Como funciona

Uma linha da tabela:

```lua
{ id = "mare.calendar", group = "MARÉ", desc = "calendário",
  combo = "SUPER + C", run = shell("toggle calendar") },
```

- **Grupos**: EMERGÊNCIA, MARÉ, APPS, JANELAS, WORKSPACES, TELA, TECLAS DE MÍDIA.
- **`run`** é um *dispatcher*: `shell(args)` chama o [[IPC do Maré]] (`qs -p <shell> ipc call mare …`); `exec(cmd)` roda um comando; `hl.dsp.window.close()` etc. agem no Hyprland.
- **Marcas**:
  - `required`: não pode ficar sem atalho (os dois de emergência);
  - `fixed`: os Ajustes não trocam (mouse, teclas de mídia, toque no Super, Alt+Tab);
  - `family`: a ação vale para várias teclas; o combo guarda só os modificadores e a tecla vem de `families` (`super`, `arrows`, `digits`), com `each(arg)` gerando o dispatcher.
- **`apply(overrides)`** remove os binds do módulo e recria todos, em quatro voltas: as fixas, as de emergência, quem está no combo de fábrica e, por último, quem trocou.
- **`canon(combo)`** põe o combo numa forma só (minúsculas, modificadores em ordem) para comparar.
- **`suspend(seconds)` / `resume()`**: desligam e religam os binds do Mk; servem à captura de teclas dos Ajustes.

Os atalhos de fábrica estão no `README.md`. Os que importam para entender o desenho:

| Atalho | Ação |
|---|---|
| `Super` (um toque) | a busca |
| `Super + Shift + Enter` | terminal de emergência |
| `Super + Shift + Esc` | sair do Hyprland |
| `Super + Enter`, `G`, `D`, `E` | terminal, lazygit, lazydocker, yazi **na pasta do projeto** |
| `Super + 1…9` | workspaces α β γ… |

## Por quê assim

- **Emergência sem o Maré**: os dois atalhos `required` não passam pelo IPC. O terminal de emergência tenta `ghostty`, depois `kitty`, `foot`, `alacritty`. Existem desde que uma sessão ficou sem shell (tela preta) e não havia como sair.
- **Toque no Super**: o bind é de **soltar** a tecla (`release = true`), e o Hyprland não o dispara se outra tecla foi apertada junto; assim `Super+Enter` continua funcionando sem abrir a busca.
- **Quatro voltas**: "uma troca nunca tira o atalho de quem não foi mexido: se ela bate em alguém, é ela que volta para o de fábrica".
- **`canon`**: sem ele, `SHIFT + SUPER + RETURN` passaria por livre ao lado de `SUPER + SHIFT + RETURN` e as duas ações rodariam juntas.
- **`reachable`**: uma ação de emergência só muda para uma tecla que todo teclado tem (Enter, Esc, espaço, Tab, letras, números, F1–F12), junto de Super, Ctrl ou Alt. Numa tecla que o teclado não tem, o atalho existiria e ninguém conseguiria apertá-lo.
- **`suspend` com relógio de segurança**: o Hyprland engole os atalhos que já existem antes de a janela vê-los, então para capturar um combo os binds do Mk "dormem"; voltam sozinhos em 8 s, "para a sessão nunca ficar sem atalhos se o Maré cair no meio da captura". Sem `hl.timer`, a suspensão é recusada.
- **Teclas de mídia passam pelo Maré**: é ele que mostra o OSD.
- **A lista existe duas vezes** (`binds.lua` e `Keybinds.js`), e um script confere: veja [[D12 Atalhos em dois lugares]].

## Onde está no código

- `config/hypr/mare/binds.lua` (a tabela `actions`, `apply`, `try`, `suspend`, `saved`)
- `shell/mare/Keybinds.js` (o espelho, sem o `run`)
- `bin/check-keybinds`

## Ligações

Depende de: [[Hyprland em Lua]], [[IPC do Maré]]. Usado por: [[Atalhos do usuário]], [[Páginas dos Ajustes]], [[Busca e calculadora]], [[Boas-vindas]], [[Checagens]].
