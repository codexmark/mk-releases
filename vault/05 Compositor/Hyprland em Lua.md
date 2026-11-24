---
camada: "05 Compositor"
tags:
  - mk
  - compositor
  - hyprland
  - lua
fontes:
  - "config/hypr/hyprland.lua"
  - "config/hypr/mare/paths.lua"
  - "config/hypr/mare/look.lua"
  - "config/hypr/mare/input.lua"
  - "shell/mare/Hypr.qml"
  - "docs/desenvolvimento.md"
---

# Hyprland em Lua

O compositor é o Hyprland 0.57, e a configuração dele é **código Lua**, não o formato `hyprland.conf`: um `hyprland.lua` pequeno e um arquivo por assunto em `mare/`.

## Para quê

O Hyprland cuida das janelas, dos workspaces, dos monitores, da entrada e dos atalhos. Tudo o que é interface (barra, painéis, bloqueio, notificações) é do [[Maré]].

## Como funciona

```
config/hypr/
  hyprland.lua        a raiz: carrega os módulos e os arquivos do usuário
  hypridle.conf       tempos de ociosidade de fábrica
  hyprlock.conf       bloqueio de reserva
  mare/
    paths.lua         onde o Mk está instalado
    env.lua           variáveis de ambiente
    display.lua       regra padrão de monitor
    look.lua          aparência das janelas
    motion.lua        curvas e animações
    input.lua         teclado e touchpad
    binds.lua         atalhos
    rules.lua         regras de janela
    autostart.lua     o que sobe com a sessão
```

A API é o objeto global `hl`: `hl.config({...})`, `hl.env`, `hl.monitor`, `hl.bind`, `hl.window_rule`, `hl.curve`, `hl.animation`, `hl.on("hyprland.start", …)`, `hl.exec_cmd`, `hl.timer` e os *dispatchers* em `hl.dsp.*`.

**`paths.lua`** descobre a instalação pelo próprio caminho (`debug.getinfo`) e devolve `root` (`<raiz>/`), `hypr`, `shell`, `ghostty` e `log`. Nada na config aponta para `~/.config/hypr` nem para `~/.config/quickshell`.

**`look.lua`**: `gaps_in = 6`, `gaps_out = 12`, borda de 1 px em osso (26% na janela ativa, 13% nas outras), layout `dwindle`, cantos retos, sem sombra e sem *blur*, fundo `0x0B0B0B`. Também: `no_update_news` e `no_donation_nag`, `allow_session_lock_restore = true`, `xwayland.force_zero_scaling`.

**`input.lua`**: `kb_layout = "br"`, `follow_mouse = 1`, touchpad com rolagem natural e toque para clicar.

## Por quê assim

- **Lua muda como se fala com o Hyprland.** Com a config em Lua, o `dispatch` espera código Lua (`hl.dsp...`) e **ignora** o formato antigo (`exec ...`, `workspace 2`). Quando o projeto adotou isso, Super+Enter, a escolha de projeto e o clique num workspace pararam de funcionar. A saída foi fazer do `Hypr.qml` o único ponto do shell que manda comandos, no formato que a sessão entende ([[D02 Um só ponto fala com o Hyprland]]).
- **Lua permite o que o formato declarativo não permite**: ler um arquivo de dados antes de criar os binds ([[Atalhos do usuário]]), desfazer um teste de monitores que ficou pela metade ([[Recuperação de monitores]]), expor funções (`require("mare.binds").apply`) que o shell chama por `hyprctl eval`.
- **`allow_session_lock_restore`**: se o Maré cair com a tela bloqueada, um Maré novo pode assumir o bloqueio ([[Tela de bloqueio]]).
- **Sem a janela de novidades do Hyprland**: ela tomava o foco no primeiro login.
- **Sem sombra, sem blur, cantos retos**: é a [[Direção Folha]] aplicada às janelas.
- O layout `dwindle`: **motivo não registrado**.

> [!note] Comentário desatualizado
> O cabeçalho do `hyprland.lua` ainda diz "Instale em ~/.config/hypr/". O Mk instala em `<raiz>/hypr`.

## Onde está no código

`config/hypr/hyprland.lua`, `config/hypr/mare/*.lua`, `shell/mare/Hypr.qml`.

## Ligações

Depende de: [[mk-session e uwsm]], [[Pasta do usuário]]. Detalhes: [[Ordem de carga]], [[Atalhos]], [[Monitores]], [[Regras de janela]], [[Animações]], [[Autostart]]. Usado por: [[Maré]], [[Ajustes]].
