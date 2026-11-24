---
camada: "07 Aparência"
tags:
  - mk
  - aparência
  - gtk
  - qt
  - ícones
  - cursor
fontes:
  - "system/mk/look/themes/Mk/gtk-4.0/gtk.css"
  - "system/mk/look/themes/Mk/index.theme"
  - "system/mk/look/make-icons"
  - "system/mk/look/qt6ct/qt6ct.conf"
  - "shell/mare/scripts/mare-look"
  - "config/hypr/mare/env.lua"
  - "migrations/system/20261121-apps-folha.sh"
---

# Tema GTK, ícones e Qt

Os aplicativos vestem a paleta do Mk por três peças pequenas: um tema GTK que só troca cores, um tema de ícones que só troca as pastas, e uma paleta para o Qt. O cursor é o Adwaita.

## Para quê

Para o Nautilus, o editor de texto e os apps Qt não parecerem de outro sistema ao lado do Maré.

## Como funciona

- **Tema GTK "Mk"** (`system/mk/look/themes/Mk/`): `gtk-3.0/gtk.css` e `gtk-4.0/gtk.css` importam o `adw-gtk3-dark` (pacote `adw-gtk-theme`) e redefinem as cores (variáveis do libadwaita e cores nomeadas) e os cantos. Instalado em `/usr/share/themes/Mk`. Escolhido por `GTK_THEME=Mk` ([[Variáveis de ambiente]]).
- **Ícones "Mk"** (`make-icons <destino>`): gera um tema que herda `Papirus-Dark,Papirus,hicolor` e só tem **links simbólicos** para as pastas na cor `palebrown` do Papirus. Apaga e recria o destino. Precisa do `papirus-icon-theme` instalado.
- **Qt**: `qt6ct.conf` (estilo Fusion, paleta `colors/Mk.conf`, ícones Mk, Space Grotesk e Space Mono), copiado para `~/.config/qt6ct` via skel; ativado por `QT_QPA_PLATFORMTHEME=qt6ct`.
- **Cursor**: `XCURSOR_THEME=Adwaita`, tamanho 24.
- **`mare-look`** (no autostart) aplica por `gsettings`: `color-scheme prefer-dark`, `gtk-theme adw-gtk3-dark`, `icon-theme Mk`, cursor, e as fontes.

## Por quê assim

- **Sem passo de build no tema GTK**: é só CSS por cima de um tema mantido por outros; "edite o CSS".
- **Ícones por links, sem copiar ícone nenhum**: "do jeito que o papirus-folders faz, mas sem mexer no pacote". Por não conter ícones, o tema não fica no git e é gerado na instalação.
- **`GTK_THEME` por variável**: "a sessão do Mk escolhe o tema com GTK_THEME=Mk, então outro desktop na mesma conta não é afetado" ([[D14 Tema por variável de sessão]]).
- **O `gsettings` é da conta inteira**, não da sessão. Por isso o `mare-look` só escreve uma chave se ela ainda está no padrão de fábrica ou se foi o próprio Mk que escreveu o valor atual (ele anota o que escreveu em `<raiz>/.look`). Assim não desfaz a escolha de outro desktop nem a do usuário.
- **`gtk-theme adw-gtk3-dark` no gsettings e `Mk` na variável**: fora da sessão do Mk, a conta fica com o tema-base; dentro, vale o do Mk.
- **O `qt6ct` do usuário não é sobrescrito**: a migração de usuário só copia se `~/.config/qt6ct` não existe.
- **A preferência por apps GTK4/libadwaita** em `apps.packages` vem daqui: é "o que veste melhor o tema GTK Mk".

## Onde está no código

- `system/mk/look/` (`themes/Mk`, `make-icons`, `qt6ct/`)
- `shell/mare/scripts/mare-look`
- `installer/mk-install` (`install_desktop`, bloco "Os apps na paleta Folha")
- `migrations/system/20261121-apps-folha.sh`, `migrations/user/20261121-apps-folha.sh`

## Ligações

Depende de: [[Direção Folha]], [[Fontes]], [[Variáveis de ambiente]], [[Autostart]]. Usado por: [[Apps padrão]], [[Apps oferecidos]].
