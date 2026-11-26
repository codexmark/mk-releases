# Avisos

MkTunes — player de música para Android.
Copyright (C) 2026 codexmark

Este programa é software livre: você pode redistribuí-lo e/ou modificá-lo sob os termos da
GNU General Public License, publicada pela Free Software Foundation, na versão 3 da licença
ou (a seu critério) qualquer versão posterior. Ele é distribuído na esperança de ser útil,
mas SEM NENHUMA GARANTIA. Veja o arquivo `LICENSE`.

SPDX-License-Identifier: GPL-3.0-or-later

## Origem

Esta pasta é o MkTunes do desktop Mk: a janela (`qml/`, Quickshell) e o daemon `mktunesd`
(`src/`, em Go, só com a biblioteca padrão). O daemon é um porte do daemon em Kotlin do
MkTunes, do mesmo autor (GPL-3.0-or-later): o mesmo protocolo com a janela, as mesmas regras
de fila, autoplay, playlists inteligentes, downloads, letras e rádio, reescritos em Go.

O MkTunes, por sua vez, segue o comportamento do [Ryotunes](https://github.com/ryoku-dev/ryotunes)
(GPL-3.0-or-later), que deriva do [LiMusic](https://github.com/SimoHypers/limusic). O código
é próprio: nenhum arquivo do Ryotunes foi copiado para cá.

## O que mudou no porte

- Não usa mais o NewPipe Extractor. A busca, as sugestões, o Início, as páginas e as filas
  vão pelo cliente InnerTube próprio; o endereço do áudio é pedido ao `yt-dlp` instalado no
  sistema, que o daemon só executa (não o inclui nem o modifica).
- A biblioteca saiu do SQLite para arquivos JSON; o banco antigo é migrado uma vez, com o
  `sqlite3` do sistema.
- O MPRIS fala com o barramento de sessão por um cliente D-Bus próprio, escrito a partir da
  especificação pública do freedesktop.

## Programas do sistema que o daemon executa

| Programa | Para quê | Licença |
|---|---|---|
| [mpv](https://mpv.io) | tocar | GPL-2.0-or-later / LGPL-2.1-or-later |
| [yt-dlp](https://github.com/yt-dlp/yt-dlp) | achar o endereço do áudio | Unlicense |
| ffprobe ([FFmpeg](https://ffmpeg.org)) | ler as etiquetas das músicas locais | LGPL-2.1-or-later / GPL-2.0-or-later |
| [SQLite](https://sqlite.org) (`sqlite3`) | migrar o banco antigo, uma vez | domínio público |

Nenhum deles vai dentro deste pacote. O binário `bin/mktunesd` só contém este código e a
biblioteca padrão do Go (BSD-3-Clause).

## Serviços usados sem conta

YouTube Music (InnerTube, por cliente próprio, e `yt-dlp`), [LRCLIB](https://lrclib.net) para
letras e [radio-browser.info](https://www.radio-browser.info) para rádio. O MkTunes não é
afiliado a nenhum deles.
