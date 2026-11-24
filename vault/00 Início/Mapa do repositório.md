---
camada: "00 Início"
tags:
  - mk
  - moc
  - repositório
fontes:
  - "README.md"
  - "docs/desenvolvimento.md"
---

# Mapa do repositório

A correspondência entre as pastas do repositório, o destino delas num Mk instalado e as camadas desta vault.

## Para quê

Para achar o arquivo certo a partir de um assunto, e o assunto a partir de um arquivo.

## Como funciona

| Caminho | O que é | Camada |
|---|---|---|
| `system/packages/` | as listas `base`, `desktop`, `dev`, `apps`, `hardware`, `aur` | [[Listas de pacotes]] |
| `system/mk/boot/`, `plymouth/` | o menu de boot e o tema do boot | [[limine-default]], [[Plymouth]] |
| `system/mk/sddm/` | o tema da tela de login | [[SDDM]] |
| `system/mk/look/`, `fonts/`, `art/` | tema GTK, ícones, Qt, fontes e os geradores da marca | [[Tema GTK, ícones e Qt]], [[Fontes]], [[Arte e geradores]] |
| `system/mk/os-release`, `issue`, `mk-os-release.hook`, `channel` | o nome do sistema e o canal | [[Nome do sistema (os-release)]], [[Canal de lançamentos]] |
| `config/hypr/` | Hyprland em Lua, `hypridle.conf`, `hyprlock.conf` | [[Hyprland em Lua]], [[Ociosidade e bloqueio]] |
| `config/bash/`, `config/ghostty/`, `config/fastfetch/` | o terminal do Mk | [[Terminal e bash do Mk]], [[Ghostty]], [[Fastfetch]] |
| `config/mimeapps.list` | os apps padrão de fábrica | [[Apps padrão]] |
| `shell/mare/` | o Maré: QML numa pasta só, `scripts/`, `assets/`, `wallpapers/` | [[Maré]] |
| `apps/` | apps do Mk que vêm prontos (o MkTunes) | [[MkTunes]] |
| `installer/` | `mk-install` e o arquivo de respostas de exemplo | [[Instalador passo a passo]] |
| `iso/` | `build`, `vm`, `profile/` | [[ISO]], [[VM de teste]] |
| `migrations/` | `system/` e `user/` | [[Migrações]] |
| `scripts/` | passos avulsos: firewall, LazyVim, mise | [[Firewall]], [[Ferramentas de desenvolvimento]] |
| `bin/` | `mk-session`, `mk-update`, `mk-release` e as ferramentas de quem desenvolve | [[mk-session e uwsm]], [[mk-update]], [[mk-release]], [[Checagens]] |
| `tests/` | os testes automáticos | [[Testes e TDD]] |
| `docs/` | documentação para quem mantém | [[Trabalhar no repositório]] |

### Onde cada parte vai parar

Num Mk instalado pelo ISO o repositório inteiro fica em `/usr/share/mk` (sem `.git`), com um arquivo `VERSION`. A partir dele:

| Origem | Destino | Quem põe |
|---|---|---|
| `config/hypr`, `shell/mare`, `config/ghostty`, `config/fastfetch`, `config/bash` | `~/.config/mk/{hypr,shell,ghostty,fastfetch,bash}` com a [[Marca instalado-pelo-mk]] | instalador (via `/etc/skel`), `mk-update`, `try-session`, `new-user` |
| `bin/mk-session`, `bin/mk-update` | `/usr/bin/` | instalador, `mk-update` |
| `system/mk/mk-os-release.hook` | `/etc/pacman.d/hooks/` | instalador, `mk-update` |
| tema GTK, ícones, fontes, plymouth, sddm | `/usr/share/...` | instalador; depois, só por migração |

## Por quê assim

A estrutura "listas separadas por função" segue o modelo do Ryoku; o conteúdo junta a base do Ryoku com o kit de desenvolvimento do Omarchy (é o que o `README.md` declara). O `mk-update` só troca `/usr/share/mk`, as cinco pastas de `<raiz>`, os binários e o gancho; todo o resto só chega a um Mk instalado por [[Migrações]]. Essa fronteira é a regra mais importante do repositório: mudou algo fora de `<raiz>`, faça no instalador **e** numa migração.

## Onde está no código

`README.md` (seção "O repositório") e `docs/desenvolvimento.md` (seção "Onde cada parte vai parar").

## Ligações

[[Início]] · [[Pasta do usuário]] · [[Convenções]]
