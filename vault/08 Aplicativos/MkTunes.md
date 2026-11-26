---
camada: "08 Aplicativos"
tags:
  - mk
  - aplicativos
  - música
  - mktunes
fontes:
  - "docs/limites.md"
  - "apps/README.md"
  - "apps/mktunes/VERSION"
  - "apps/mktunes/CONTRATO.md"
  - "apps/mktunes/NOTICE.md"
  - "migrations/system/20261207-mktunes-go.sh"
  - "apps/mktunes/bin/mktunes"
  - "system/mk/mktunes-links"
  - "migrations/system/20261127-mktunes.sh"
  - "system/packages/desktop.packages"
---

# MkTunes

O MkTunes é o player de música do Mk: músicas, álbuns, rádios e letras, sem conta. Foi o primeiro app "do Mk" a vir pronto dentro do repositório, em `apps/mktunes`. O núcleo era um daemon em Kotlin, gerado em outro repositório; hoje é um daemon em Go cujo código mora aqui, como nos outros apps.

## Para quê

Tocar música com uma janela na mesma linguagem do Maré e aparecendo na ilha.

## Como funciona

- **Duas partes**: a janela é do Quickshell (QML, com os tokens do Maré) e o núcleo é um *daemon* em Go (`mktunesd`, só com a biblioteca padrão) que toca pelo `mpv` e aparece na ilha pelo MPRIS. Os dois conversam por um soquete (`$XDG_RUNTIME_DIR/mktunes/mktunes.sock`), uma linha JSON por mensagem; o protocolo está em `apps/mktunes/CONTRATO.md`.
- **`apps/mktunes/`**: `bin/mktunes` (o lançador), `bin/mktunesd` (o daemon compilado) e `bin/mktunesd.de` (a assinatura do código de que ele veio), `src/` (o código em Go), `qml/`, `share/` (o `.desktop` e o ícone), `pack`, `assinatura`, `CONTRATO.md`, `VERSION`, `LICENSE`, `NOTICE.md`.
- **De onde vem a música**: a busca, as sugestões, o Início, as páginas de álbum e artista e a fila automática vêm do YouTube Music por um cliente próprio (InnerTube); o endereço do áudio é pedido ao YouTube pelo mesmo cliente (perfil visionOS, o tipo de endereço que o YouTube serve inteiro) e vai direto ao mpv; as letras vêm do LRCLIB e as rádios do radio-browser. As tags das músicas locais vêm do `ffprobe`.
- **A biblioteca**: arquivos JSON em `~/.local/share/mktunes` (`biblioteca.json`, `preferencias.json`, `sessao.json`, `downloads.json`, `historico.jsonl`), escritos num temporário e trocados de nome.
- **O MPRIS**: `org.mpris.MediaPlayer2.mktunes`, por um cliente D-Bus próprio (`src/internal/dbus`), escrito a partir da especificação pública.
- **O lançador**: sobe o `mktunesd` se ninguém responde no soquete e abre a janela; se já está aberta, só a traz para a frente. Se falta o `mpv` ou o Quickshell, avisa por notificação.
- **Ligação ao sistema** (`system/mk/mktunes-links`): três links simbólicos para `/usr/share/mk/apps/mktunes`: `/usr/bin/mktunes`, o atalho em `/usr/share/applications` e o ícone em `hicolor`.
- **Dependências**: `mpv`, `ffmpeg`, `sqlite`, em `desktop.packages`.

## Por quê assim

- **Porte, não reescrita livre**: o daemon em Go segue o daemon em Kotlin campo por campo, e os testes dele foram portados junto. Por isso a janela não mudou.
- **Sem a máquina virtual**: o daemon em Kotlin pedia uma JVM e ocupava por volta de 100 MB; o binário em Go ocupa menos de 20 MB tocando e sobe na hora. O pacote `jre-openjdk-headless` saiu de `desktop.packages` (ninguém o remove de uma máquina já instalada: pode ser de outro programa).
- **Só a biblioteca padrão**: as três peças que dependiam de bibliotecas foram trocadas. O SQLite virou arquivos JSON; o extrator do YouTube virou o cliente InnerTube próprio; a biblioteca de D-Bus virou um cliente mínimo escrito à mão.
- **Quem vinha da versão em Kotlin não faz nada**: o lançador para o daemon antigo que ficou no ar (achado em `/proc` pelo nome da classe) e, na primeira subida, o daemon novo lê o banco `mktunes.db` com o `sqlite3`, junta à biblioteca e o renomeia para `mktunes.db.migrado`. Daí em diante, um `mktunesd` de versão diferente da instalada é encerrado (guardando a sessão) para o novo subir.
- **Licença própria**: o MkTunes é GPL-3.0-or-later, e o porte continua sendo. É a única pasta do repositório com licença própria (`LICENSE`, `NOTICE.md`).
- **Vem compilado dentro do Mk**: o `mk-update` troca `/usr/share/mk` inteiro, então "uma versão nova de um app chega como qualquer outra mudança do Mk", sem pacote, sem AUR, sem repositório de binários.
- **Links em vez de cópias**: "a versão que o mk-update traz vale sozinha, sem rodar isto de novo".
- **Em `desktop.packages`, não em `apps.packages`**: `mpv` e `ffmpeg` também estão na lista de apps; estão aqui "porque o MkTunes não é opcional".
- **Num Mk já instalado**: a migração `20261127-mktunes.sh` cria os links e a `20261207-mktunes-go.sh` pede o `sqlite`.

> [!warning] Depende de serviços de fora
> O YouTube muda as respostas sem avisar. A busca, o Início e o endereço do áudio dependem dos leitores próprios. A primeira versão do porte pedia o endereço ao `yt-dlp` e o YouTube passou a recusá-lo (403 depois de 1 MiB); por isso o daemon voltou a pedir sozinho, como a versão em Kotlin fazia. O que foi e o que não foi conferido está em `docs/limites.md`.

## Onde está no código

`apps/README.md`, `apps/mktunes/`, `system/mk/mktunes-links`, `migrations/system/20261127-mktunes.sh`, `migrations/system/20261207-mktunes-go.sh`, `tests/apps/mktunes-go.test.sh`, `tests/apps/mktunes-janela.test.sh`.

## Ligações

Depende de: [[Áudio]], [[Listas de pacotes]], [[Direção Folha]], [[Apps do Mk (motor em Go, janela em QML)]]. Atualizado por: [[mk-update]]. Aparece em: [[Ilha]], [[Painéis]] (mídia).
