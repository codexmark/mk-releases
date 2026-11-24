---
camada: "07 Aparência"
tags:
  - mk
  - aparência
  - fontes
fontes:
  - "shell/mare/Theme.qml"
  - "system/packages/base.packages"
  - "system/packages/aur.packages"
  - "system/mk/fonts/OFL.txt"
  - "installer/mk-install"
  - "docs/desenvolvimento.md"
  - "system/mk/look/qt6ct/qt6ct.conf"
---

# Fontes

O Mk usa quatro papéis de fonte (serifa, sem serifa, mono e ícones), cada um com uma fonte preferida e uma de reserva.

## Para quê

A tipografia é metade da [[Direção Folha]]: serifa nos números e títulos, mono espaçado nos rótulos, grotesca no texto.

## Como funciona

| Papel | Preferida | De onde vem | Reserva |
|---|---|---|---|
| `serif` | Fraunces | AUR (`ttf-fraunces-variable`) | Noto Serif |
| `sans` | Space Grotesk | **no repositório** (`system/mk/fonts/*.otf`, 4 pesos) | Inter |
| `mono` | Space Mono (Nerd Font) | `ttf-space-mono-nerd` | Geist Mono Nerd, JetBrains Mono Nerd |
| `icons` | Geist Mono Nerd Font | `otf-geist-mono-nerd` | JetBrains Mono Nerd |

- `Theme.qml` consulta `Qt.fontFamilies()` e escolhe a preferida se estiver instalada.
- `base.packages` traz ainda `noto-fonts`, `noto-fonts-cjk`, `noto-fonts-emoji`, `inter-font` e `ttf-jetbrains-mono-nerd`.
- A Space Grotesk é instalada em `/usr/share/fonts/mk/`, com a licença em `/usr/share/licenses/mk-fonts/OFL.txt`, e o instalador roda `fc-cache`.
- Fora do Maré: o `mare-look` põe "Space Grotesk 10" e "SpaceMono Nerd Font 10" no `gsettings`; o `qt6ct.conf` faz o mesmo para o Qt; o [[Ghostty]] usa a Space Mono.

## Por quê assim

- **Reserva para cada papel**: a Fraunces vem do AUR, que pode falhar no build da ISO; "sem ela o Maré usa a Noto Serif". O sistema não fica sem fonte por causa de um pacote.
- **A Space Grotesk mora no repositório**: é a sans do Maré "e não vinha em nenhum pacote das listas". É a única fonte que o Mk carrega consigo; a licença (SIL OFL 1.1) vai ao lado e é instalada junto. A regra para acrescentar outra: trazer o arquivo de licença e instalar os dois.
- **Ícones numa fonte separada**: "quando o mono escolhido não os tem".
- **A Space Grotesk chegou aos Mk já instalados por migração** (`20261121-apps-folha.sh`): fontes ficam em `/usr/share`, fora do que o `mk-update` troca.

## Onde está no código

- `shell/mare/Theme.qml` (`serif`, `sans`, `mono`, `icons`)
- `system/mk/fonts/`, `installer/mk-install` (`install_desktop`)
- `system/packages/base.packages` (seção "Fontes"), `aur.packages`

## Ligações

Depende de: [[Listas de pacotes]], [[Pacman e AUR]]. Usado por: [[Direção Folha]], [[Maré]], [[SDDM]], [[Tema GTK, ícones e Qt]], [[Ghostty]], [[Arte e geradores]].
