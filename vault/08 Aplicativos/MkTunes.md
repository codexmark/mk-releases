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
  - "apps/mktunes/bin/mktunes"
  - "system/mk/mktunes-links"
  - "migrations/system/20261127-mktunes.sh"
  - "system/packages/desktop.packages"
---

# MkTunes

O MkTunes é o player de música do Mk: músicas, álbuns, rádios e letras, sem conta. Foi o primeiro app "do Mk" a vir pronto dentro do repositório, em `apps/mktunes`, e é o único cujo código mora fora dele e cujo núcleo é em Java.

## Para quê

Tocar música com uma janela na mesma linguagem do Maré e aparecendo na ilha.

## Como funciona

- **Duas partes**: a janela é do Quickshell (QML, com os tokens do Maré) e o núcleo é um *daemon* Java (`mktunesd`) que toca pelo `mpv` e aparece na ilha pelo MPRIS. As tags das músicas locais vêm do `ffprobe`.
- **`apps/mktunes/`**: `bin/mktunes` (o lançador), `daemon/` (`bin`, `lib`), `qml/`, `share/` (o `.desktop` e o ícone), `VERSION`, `LICENSE`.
- **O lançador**: sobe o `mktunesd` se ninguém responde no soquete (`$XDG_RUNTIME_DIR/mktunes/mktunes.sock`) e abre a janela; se já está aberta, só a traz para a frente. Se falta o `mpv` ou o Quickshell, avisa por notificação.
- **Ligação ao sistema** (`system/mk/mktunes-links`): três links simbólicos para `/usr/share/mk/apps/mktunes`: `/usr/bin/mktunes`, o atalho em `/usr/share/applications` e o ícone em `hicolor`.
- **Dependências**: `jre-openjdk-headless`, `mpv`, `ffmpeg`, em `desktop.packages`.

## Por quê assim

- **A pasta é gerada, nunca editada aqui**: o código mora no repositório do MkTunes, que tem um comando para testar o pacote e outro para empacotar para dentro de `apps/mktunes`. O `VERSION` diz de qual commit veio.
- **Vem compilado dentro do Mk**: o `mk-update` troca `/usr/share/mk` inteiro, então "uma versão nova de um app chega como qualquer outra mudança do Mk", sem pacote, sem AUR, sem repositório de binários.
- **Links em vez de cópias**: "a versão que o mk-update traz vale sozinha, sem rodar isto de novo".
- **Em `desktop.packages`, não em `apps.packages`**: `mpv` e `ffmpeg` também estão na lista de apps; estão aqui "porque o MkTunes não é opcional".
- **Continua em Java**: os apps que vieram depois têm o motor em Go ([[Apps do Mk (motor em Go, janela em QML)]]). O porte do MkTunes para Go espera o repositório dele, que não estava na máquina onde os outros apps foram feitos (`docs/limites.md`). Até lá, ele é o único app do Mk que pede uma JVM (`jre-openjdk-headless`).
- **Java só de linha de comando** (`headless`): o núcleo não tem parte gráfica.
- **Daemon velho depois de um `mk-update`**: o lançador detecta um daemon que segura os jars que a atualização apagou (arquivos `(deleted)` em `/proc/<pid>/fd`), para-o e sobe o novo.
- **Num Mk já instalado**: a migração `20261127-mktunes.sh` cria os links e pede os três pacotes.

> [!note] Fora do lint
> `tests/lint.test.sh` deixa `apps/` de fora: é código gerado em outro repositório.

## Onde está no código

`apps/README.md`, `apps/mktunes/`, `system/mk/mktunes-links`, `migrations/system/20261127-mktunes.sh`.

## Ligações

Depende de: [[Áudio]], [[Listas de pacotes]], [[Direção Folha]]. Atualizado por: [[mk-update]]. Aparece em: [[Ilha]], [[Painéis]] (mídia).
