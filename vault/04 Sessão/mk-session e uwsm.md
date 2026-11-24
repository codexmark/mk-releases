---
camada: "04 Sessão"
tags:
  - mk
  - sessão
  - uwsm
  - hyprland
fontes:
  - "bin/mk-session"
  - "installer/mk-install"
  - "system/packages/desktop.packages"
  - "bin/try-session"
  - "bin/new-user"
---

# mk-session e uwsm

`mk-session` é o lançador da sessão: um script de dez linhas que confere se a config existe e se troca pelo `uwsm`, que inicia o Hyprland como uma sessão do systemd.

## Para quê

É o `Exec` da entrada "Mk" da tela de login. Vale para qualquer usuário que tenha a config em `~/.config/mk/hypr`.

## Como funciona

```bash
config="${XDG_CONFIG_HOME:-$HOME/.config}/mk/hypr/hyprland.lua"
exec uwsm start -e -D Hyprland -N mk -- start-hyprland -- -c "$config"
```

- Se o `hyprland.lua` não existe, sai com erro e manda rodar `bin/try-session install`.
- O Hyprland recebe a config **por caminho** (`-c`), não a de `~/.config/hypr`.
- Instalado em `/usr/bin/mk-session` pelo instalador e pelo `mk-update`; em `/usr/local/bin/mk-session` pelo `try-session` e pelo `new-user`.
- A saída da sessão, no painel de sessão do Maré, é `uwsm stop || hyprctl dispatch 'hl.dsp.exit()'`.

## Por quê assim

- **Config por caminho, fora de `~/.config/hypr`**: o Mk pode conviver com outro desktop Hyprland na mesma conta sem tocar na config dele. O `paths.lua` descobre a raiz a partir do próprio caminho ([[Hyprland em Lua]]).
- **`uwsm`**: o comentário da lista diz "inicia o Hyprland como sessão systemd (mesmo esquema do Omarchy)". O que isso traz na prática (alvos de sessão, ambiente para as unidades de usuário) não está descrito no repositório.
- **Sessão separada no login**: é o que permite experimentar o Mk sem instalar pelo ISO ([[Experimentar sem instalar]]).

## Onde está no código

`bin/mk-session`; `installer/mk-install` (`install_desktop`, o `mk.desktop`).

## Ligações

Depende de: [[SDDM]], [[Pasta do usuário]]. Inicia: [[Hyprland em Lua]], [[Ordem de carga]].
