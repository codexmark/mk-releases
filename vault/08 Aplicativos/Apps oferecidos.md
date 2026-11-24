---
camada: "08 Aplicativos"
tags:
  - mk
  - aplicativos
  - pacotes
  - atualização
fontes:
  - "system/packages/apps.packages"
  - "bin/mk-update"
  - "installer/mk-install"
  - "migrations/system/20261125-apps-oferecidos.sh"
  - "docs/atualizacoes.md"
---

# Apps oferecidos

`apps.packages` é a lista dos aplicativos gráficos do dia a dia. É a única lista **opcional** na instalação, e a única que o `mk-update` olha depois: um app que entrar nela é oferecido **uma vez** em cada Mk instalado.

## Para quê

Deixar o conjunto de apps crescer sem reinstalar ninguém e sem trazer de volta o que a pessoa desinstalou.

## Como funciona

**O conjunto** (um app por função):

| Função | Apps |
|---|---|
| navegador | `chromium` |
| arquivos | `nautilus`, `sushi`, `file-roller`, `unrar`, `gvfs-gphoto2`, `gvfs-dnssd`, `ffmpegthumbnailer`, `libheif` |
| texto e documentos | `gnome-text-editor`, `gnome-calculator`, `papers`, `libreoffice-fresh` |
| imagens | `loupe`, `pinta` |
| vídeo e música | `mpv`, `mpv-mpris`, `amberol`, `gst-libav`, `snapshot` |
| sistema | `gnome-disk-utility`, `baobab`, `seahorse`, `gnome-font-viewer` |
| scanner | `simple-scan`, `sane-airscan` |
| torrent | `fragments` |
| linha de comando | `ffmpeg`, `imagemagick`, `yt-dlp` |

**O mecanismo da oferta**:

1. O instalador grava a lista do dia em `/var/lib/mk/apps-oferecidos`, **tenha o usuário aceitado os apps ou não**.
2. A cada rodada, o `mk-update` calcula o que está em `apps.packages` e não consta nesse arquivo, e tira o que já está instalado (`pacman -T`).
3. Se sobra algo, pergunta "Instalar junto com a atualização?" e, à parte, pelo LibreOffice (~150 MB). Com LibreOffice aceito e `LANG=pt_BR`, entra o `libreoffice-fresh-pt-br`.
4. Os aceitos entram na **mesma transação** do `pacman -Syu`.
5. Depois que o pacman dá certo, os oferecidos entram no arquivo, **aceitos ou recusados**.

## Por quê assim

- **Oferecido uma vez só**: "para não trazer de volta o que você tirou".
- **Na mesma transação do `-Syu`**: "instalar app novo sobre um sistema desatualizado quebra bibliotecas" ([[Pacman e AUR]]).
- **Só anota depois do pacman dar certo**: se ele falha, a oferta volta na próxima rodada.
- **O arquivo vazio desliga a oferta**: a condição é `-s` (existe e não é vazio). Numa máquina antiga, a migração `20261125-apps-oferecidos.sh` cria o arquivo: com o Nautilus instalado, anota o conjunto antigo (e os apps novos são oferecidos); sem os apps gráficos, anota a lista inteira (nada a oferecer).
- **GTK4/libadwaita de preferência**: é o que veste melhor o tema GTK do Mk ([[Tema GTK, ícones e Qt]]).
- **O que ficou de fora de propósito** (comentário da lista): monitor de sistema ("a ilha e o btop já cobrem", diz o comentário; hoje há também o [[MkPulso]]), app de notas, anotador de capturas (entra quando o painel de captura tiver a ação "anotar"), gimp, kdenlive, obs-studio.
- **`mpv-mpris`**: é o que faz o mpv aparecer na ilha e obedecer às teclas de mídia.

## Onde está no código

- `system/packages/apps.packages` (os comentários explicam cada app)
- `bin/mk-update` (`find_new_apps` e o passo 3)
- `installer/mk-install` (`install_desktop`, a linha do `apps-oferecidos`)

## Ligações

Depende de: [[Listas de pacotes]], [[Pacman e AUR]]. Usado por: [[mk-update]], [[Apps padrão]], [[Instalador passo a passo]].
