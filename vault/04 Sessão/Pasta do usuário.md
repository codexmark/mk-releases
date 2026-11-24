---
camada: "04 Sessão"
tags:
  - mk
  - sessão
  - usuário
  - configuração
fontes:
  - "README.md"
  - "docs/desenvolvimento.md"
  - "bin/mk-update"
  - "config/hypr/mare/paths.lua"
  - "shell/mare/Settings.qml"
---

# Pasta do usuário

O Mk de cada usuário mora em `~/.config/mk`, chamada `<raiz>` nesta vault. Dentro dela há dois tipos de coisa: pastas **do Mk**, trocadas inteiras a cada atualização, e arquivos **do usuário**, que nenhuma atualização toca.

## Para quê

Separar com clareza o que é código (pode ser substituído) do que é escolha (tem de sobreviver).

## Como funciona

**Do Mk** (têm a [[Marca instalado-pelo-mk]]; não edite):

| Pasta | Origem |
|---|---|
| `hypr/` | `config/hypr` |
| `shell/` | `shell/mare` |
| `ghostty/` | `config/ghostty` |
| `fastfetch/` | `config/fastfetch` |
| `bash/` | `config/bash` |

**Do usuário** (na raiz de `<raiz>`):

| Arquivo | O que é | Quem escreve |
|---|---|---|
| `ajustes.json` | tudo o que foi escolhido nos Ajustes | os [[Ajustes]] |
| `ajustes.lua` | a parte dos ajustes que é do Hyprland | os Ajustes |
| `monitores.lua` | uma regra por tela | Ajustes › [[Monitores]] |
| `atalhos.lua` | os atalhos trocados | Ajustes › [[Atalhos do usuário]] |
| `hypridle.conf` | os tempos de ociosidade | Ajustes › Energia |
| `seu.bash` | aliases, variáveis e `PATH` do usuário | o usuário (`mk seu`) |
| `channel` | outro canal de atualização | o usuário |
| `.bem-vindo` | marca de que as boas-vindas já foram vistas | o Maré |
| `.look` | o que o `mare-look` escreveu no gsettings | `mare-look` |
| `VERSION` | a versão instalada | o `mk-update` (o único que a atualização reescreve) |

Fora de `<raiz>`: `~/.config/mare/city` (a cidade do clima), `~/.config/mimeapps.list`, `~/.config/qt6ct`, `~/.local/state/mk/` (migrações feitas, registros de atualização), `~/.cache/mk/mare.log`, o estado do Quickshell (`workspaces.json`, `wallpaper.json`) e `<Imagens>/Wallpapers`.

## Por quê assim

- **`~/.config/mk`, e não `~/.config/hypr` + `~/.config/quickshell`**: essas pastas podem ser de outro desktop. No primeiro login de um usuário novo, outro desktop instalado na mesma máquina recriava `~/.config/quickshell` e apagava o Maré copiado ali, deixando a sessão sem shell ([[D01 Maré fora da pasta do Quickshell]]).
- **Tudo é achado por caminho relativo**: o `paths.lua` descobre a raiz a partir do próprio arquivo, e o `Settings.qml` tira `dir` de `Quickshell.shellPath("")`. A instalação funciona em qualquer lugar.
- **Arquivos gerados na raiz**: é a única parte que sobrevive à troca das pastas.

> [!warning] `monitors.lua` e `user.lua` se perdem
> O `hyprland.lua` carrega por último `monitors.lua` e `user.lua`, se existirem **ao lado dele**, dentro de `hypr/`. Como essa pasta é trocada inteira, eles somem a cada atualização do Mk. Guarde uma cópia fora, ou tire a marca `hypr/.instalado-pelo-mk` (e deixe de receber as mudanças do Hyprland do Mk).

## Onde está no código

- `README.md` (seção "Onde vai o que é seu")
- `config/hypr/mare/paths.lua`
- `bin/mk-update` (o laço das cinco pastas)

## Ligações

Depende de: [[Usuários e sudo]]. Usado por: [[Ordem de carga]], [[Ajustes]], [[mk-update]], [[Marca instalado-pelo-mk]].
