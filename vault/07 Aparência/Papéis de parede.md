---
camada: "07 Aparência"
tags:
  - mk
  - aparência
  - papel-de-parede
fontes:
  - "tests/system/papeis-de-parede.test.sh"
  - "shell/mare/scripts/mare-weather"
  - "migrations/user/20261128-papel-de-fabrica.sh"
  - "shell/mare/Place.js"
  - "shell/mare/Wallpaper.qml"
  - "shell/mare/WallpaperPanel.qml"
  - "shell/mare/Backdrop.qml"
  - "shell/mare/scripts/mare-palette"
  - "system/mk/art/make-wallpapers.py"
  - "migrations/user/20261122-papeis-de-parede.sh"
---

# Papéis de parede

O papel de parede é desenhado pelo próprio Maré, por cima da folha de desenho técnico do fundo. O Mk traz dezoito (seis autorais e doze gerados por código), lista primeiro os do usuário, e uma instalação nova já nasce com um aplicado.

## Para quê

Personalizar a área de trabalho sem um programa de papel de parede à parte, e deixar a imagem escolhida tingir a interface.

## Como funciona

- **O fundo** (`Backdrop.qml`, camada *Background*): uma folha de desenho (moldura com marcas de corte e régua, grade de pontos, um arco com colunas e linhas de construção, o sol de cobre, a letra grega do workspace em serifa, o carimbo no canto). Num workspace vazio, a folha convida a começar. Com papel de parede escolhido, a imagem entra por cima, escurecida.
- **Onde ficam**: os do usuário em `<pasta de imagens>/Wallpapers` (achada com `xdg-user-dir PICTURES`; `~/Imagens` em português); os do Mk em `shell/mare/wallpapers/`: seis autorais (Entropia, Núcleo, Origem, Monólito, Singularidade e Maré, em `00-1-entropia` … `00-6-mare`, criados e assinados por quem mantém o Mk, mostrados primeiro) e doze desenhados por código (`01-arcos` … `12-catedral`).
- **O painel** (`Super + W`): 12 por página, os do usuário primeiro. Clique aplica.
- **De fábrica**: uma instalação nova nasce com o Maré (`00-6-mare.jpg`) aplicado. Vale uma vez só, e nunca por cima de uma escolha: o `Wallpaper.qml` só o aplica se a migração de usuário `20261128-papel-de-fabrica.sh` consta como feita e a marca `<raiz>/.papel-de-fabrica` não existe.
- **O lugar na folha**: a cidade no carimbo e as coordenadas junto ao horizonte vêm do que o clima achou (`Place.js`, com `city`, `country`, `lat` e `lon` do `mare-weather`): a cidade de `~/.config/mare/city` ou a do fuso do sistema. Sem clima (sem rede, cidade desconhecida), nada é escrito.
- **A cor**: `mare-palette` roda `matugen image … --dry-run -m dark --prefer saturation` e devolve a cor principal; o `Wallpaper.qml` guarda `primary` e o matiz (`hue`), e a interface toma o matiz da imagem.
- **Persistência**: `wallpaper.json` no estado do Quickshell (caminho e cor).
- **Formatos**: jpg, jpeg, png, webp (o `qt6-imageformats` abre WebP, AVIF e TIFF).
- **Ajustes › Fundo**: letra do workspace, quanto escurecer a imagem (padrão 35%), grão (padrão 6%).
- **IPC**: `qs … ipc call wallpaper set <caminho> | random | get`.

## Por quê assim

- **Desenhado pelo Maré**: "o papel de parede é desenhado pelo próprio Maré" (comentário de `desktop.packages`), então não há um programa de papel de parede nas listas. A imagem entra por cima da folha, na mesma janela de fundo.
- **Pasta de imagens do sistema, não `~/Pictures` fixo**: a primeira versão criava `~/Pictures` ao lado de `~/Imagens`. Uma migração de usuário move o que havia e corrige o caminho gravado.
- **Os doze são desenhados por código**: cada um é um SVG montado no script, renderizado em 2560x1440 e finalizado com grão (que também evita faixas nos degradês). "Fundo sempre calmo no alto (onde fica a ilha) e sem contraste forte onde as janelas ficam."
- **Os seis autorais em 2560x1440**: saem do gerador de quem os fez em 1672x941; foram ampliados duas vezes com o `waifu2x-ncnn-vulkan` (modelo de foto, sem redução de ruído) e reduzidos para o tamanho dos outros doze, porque as linhas finas e a assinatura ficam mais nítidas que com a interpolação que a tela faria. O prefixo `00-` é o que os põe antes dos doze no painel.
- **O de fábrica não aparece para quem já usava o Mk**: essa pessoa "escolheu o seu, ou escolheu não ter"; a migração grava a marca para ela. E como o `mk-update` reinicia o Maré antes de rodar as migrações, a aplicação espera a migração constar como feita, do mesmo jeito que as [[Boas-vindas]].
- **O lugar não é mais escrito no código**: o fundo trazia uma cidade e coordenadas fixas; agora mostra a cidade que o clima achou, a de quem usa.
- **Papéis de parede no repositório são só os do Mk**; os do usuário ficam na pasta de imagens dele, fora do repositório ([[Convenções]]).
- **A camada de cima só some quando a de baixo está pronta**: ao trocar, a tela mostrava o papel anterior por um instante.

## Onde está no código

`shell/mare/Wallpaper.qml`, `WallpaperPanel.qml`, `Backdrop.qml` (458 linhas), `scripts/mare-palette`, `system/mk/art/make-wallpapers.py`.

## Ligações

Depende de: [[Direção Folha]], [[Arte e geradores]], [[Singletons]]. Usado por: [[Painéis]], [[Monitor por tela]], [[Páginas dos Ajustes]].
