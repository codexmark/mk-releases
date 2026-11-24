---
camada: "12 Decisões e limites"
tags:
  - mk
  - decisões
  - compositor
  - maré
fontes:
  - "shell/mare/Hypr.qml"
  - "shell/mare/README.md"
  - "config/hypr/hyprland.lua"
---

# D02 Um só ponto fala com o Hyprland

No shell, só o singleton `Hypr` manda comandos ao compositor.

## Contexto

O projeto configura o Hyprland em Lua. Com a config em Lua, o `dispatch` do Hyprland espera **código Lua** (`hl.dsp.exec_cmd(…)`, `hl.dsp.focus({ workspace = 2 })`) e ignora os comandos no formato antigo (`exec …`, `workspace 2`). Quando isso passou a valer, pararam de funcionar de uma vez: Super+Enter, escolher projeto, ir para um projeto pela busca, confirmar o Alt+Tab e clicar num workspace. O shell tinha chamadas a `Hyprland.dispatch` espalhadas, todas no formato antigo.

## Escolha

`Hypr.qml` passa a ser "o único ponto que fala com o Hyprland": três funções (`exec`, `workspace`, `focusWindow`) que emitem o formato Lua ou o antigo conforme `Hyprland.usingLua`. Os comandos de DPMS e de sair também foram convertidos.

## Alternativas

- **Converter cada chamada no lugar** *(não discutida no repositório)*: o commit preferiu concentrar as chamadas num ponto só.
- **Voltar para a config no formato `hyprland.conf`** *(não discutida no repositório)*: o resto do projeto depende do Lua para ler dados antes de criar binds e expor funções chamáveis por `hyprctl eval` ([[Hyprland em Lua]]).

## Consequência

- Regra do shell: do QML, nunca `Hyprland.dispatch` direto ([[Armadilhas do Maré]]).
- O shell continua funcionando numa sessão que ainda use o formato antigo (o ramo `else` de cada função).
- Fora do QML, os scripts que falam com o Hyprland escrevem Lua explicitamente: `hyprctl dispatch 'hl.dsp.dpms({ action = "off" })'` no hypridle, `hl.dsp.focus(…)` no `mare-start`.

## Onde está no código

`shell/mare/Hypr.qml` (29 linhas).

## Ligações

Afeta: [[Singletons]], [[Hyprland em Lua]], [[Projetos e workspaces]], [[Ociosidade e bloqueio]]. Índice: [[Registro de decisões]].
