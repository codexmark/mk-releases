---
camada: "08 Aplicativos"
tags:
  - mk
  - aplicativos
  - mkutil
  - appimage
  - ffmpeg
fontes:
  - "apps/mkutil/CONTRATO.md"
  - "apps/README.md"
  - "apps/mkutil/bin/mkinstalar"
  - "apps/mkutil/bin/mkcomprimir"
  - "apps/mkutil/qml/shell.qml"
  - "system/mk/apps-links"
  - "migrations/system/20261130-mkpacotes-mkutil.sh"
  - "docs/limites.md"
---

# MkInstalar e MkComprimir

Dois utilitários pequenos num app só (`apps/mkutil`), com um motor comum em Go: o **MkInstalar** instala um arquivo que a pessoa baixou, e o **MkComprimir** encolhe vídeos e imagens.

## Para quê

Dois gestos comuns que no Arch pedem terminal: "baixei um AppImage, e agora?" e "este vídeo é grande demais para mandar".

## Como funciona

### MkInstalar (`mkinstalar [arquivos…]`)

É também o alvo de "abrir com" desses arquivos.

| Tipo | O que faz |
|---|---|
| AppImage | copia para `~/Aplicativos/<id>.AppImage`, dá `+x`, guarda o ícone e grava o atalho `mkinstalar-<id>.desktop` |
| pacote do Arch (`.pkg.tar.zst`, `.pkg.tar.xz`) | `pkexec /usr/bin/pacman -U --needed --noconfirm -- <arquivo>` |
| compactado (`.tar.gz`, `.tar.xz`, `.tar.zst`, `.tar.bz2`, `.zip`) | extrai para `~/Aplicativos/<id>` e cria o atalho do executável sugerido |

- `inspecionar` diz o que o arquivo é e o que instalar faria, sem mudar nada no disco; `instalar` faz, soltando o progresso; `instalados` e `desinstalar` cuidam do que ele mesmo instalou (registro em `~/.local/state/mk/instalados.json`).
- `.deb`, `.rpm` e `.flatpakref` são erro, com a frase que diz por quê.

**Recusas**:

- **O AppImage nunca é executado para ser inspecionado**: é lido por dentro (o squashfs dele), não rodado com `--appimage-extract`. Do atalho de dentro só passam as chaves que não executam nada.
- **Caminhos que saem da pasta (*zip slip*)**: uma entrada com caminho absoluto, com `..`, com `\` ou vazia; um link que aponta para fora; um arquivo especial (dispositivo, fifo, soquete). O arquivo **inteiro** é recusado: nada é extraído pela metade.
- **Bombas**: mais entradas ou mais bytes que o limite (padrão 200 mil entradas e 8 GiB), contando os bytes de verdade na extração.
- **Nunca troca o que não é dele**: um arquivo ou pasta em `~/Aplicativos` que não está no registro nunca é substituído; instalar de novo o que é dele pede `--substituir`.
- **O registro não é confiado**: `desinstalar` recusa se o destino não é filho direto de `~/Aplicativos`, se o atalho não é `mkinstalar-*.desktop` na pasta certa, ou se o ícone não está em `~/Aplicativos/.icones`. Um link simbólico é apagado como link, nunca seguido.
- Permissões especiais (setuid, setgid) saem na extração.

### MkComprimir (`mkcomprimir [arquivos…]`)

Encolhe com o `ffmpeg`, por predefinições:

| Predefinição | O que faz |
|---|---|
| `menor`, `equilibrio`, `alta` | três níveis de qualidade e resolução |
| `caber` | cabe em N MB: a taxa sai da duração, em duas passadas |
| `resolucao` | só reduz para 1080, 720 ou 480 |
| `sem-audio` | tira o áudio sem recodificar o vídeo |
| `gif`, `webp` | animação |
| `imagem` | JPEG, WebP ou AVIF, com qualidade e largura |

- A janela mostra a **estimativa antes** de começar, o progresso (do `-progress` do ffmpeg), cancelar e fila.
- Codec `h264` por padrão, "pela compatibilidade"; `h265` e `av1` a pedido; aceleração (`vaapi`, `nvenc`) só se a pessoa pedir e o ffmpeg listar o codificador.
- **Nunca sobrescreve**: a saída fica ao lado do original, com um sufixo; se o nome já existe, ganha `-2`. O ffmpeg escreve num arquivo escondido (`.mkcomprimir-….parcial.<ext>`) que só ganha o nome no fim; cancelado ou com erro, é apagado.

## Por quê assim

- **Nome de arquivo nunca passa por shell**: todo comando roda com lista de argumentos, com o caminho absoluto e depois de `--`. O ffmpeg e o ffprobe nem veem o nome: recebem um atalho numa pasta temporária, porque um nome de imagem com `%` ou `*` é lido por eles como padrão de sequência.
- **`pkexec /usr/bin/pacman`, por extenso**: com o nome solto, o `pkexec` procuraria no `PATH` de quem chamou e rodaria como root o primeiro `pacman` que achasse.
- **Não executar o AppImage**: o contrato põe em negrito que "o motor nunca executa o arquivo que está inspecionando".
- **A mesma conta em dois lugares, com os mesmos casos**: a estimativa do "caber em N MB" é feita pelo motor e pela janela (`Util.js`), e os dois testes usam o mesmo arquivo de casos (`testdata/caber.json`).
- **Pacote instalado não entra no registro**: quem o tira é o pacman.
- **Não pedem pacote novo**: usam `ffmpeg`, `jq`, `zstd`, `xz` e `libarchive`, que o Mk já tem.

> [!warning] Defeitos abertos e o que não foi visto
> Fechar a janela do MkComprimir no meio de uma codificação pode deixar um `.mkcomprimir-*.parcial.*` escondido ao lado do vídeo. Comprimir um vídeo foi exercitado na VM pelo motor; o seletor de arquivo e o arrastar das duas janelas nunca foram usados de verdade ([[Limitações conhecidas]]).

## Onde está no código

`apps/mkutil/` (`src/internal/instala/` com `appimage.go`, `compactado.go`, `pacote.go`, `registro.go`; `src/internal/squash/`; `src/internal/midia/`); `qml/Instalar.qml`, `Comprimir.qml`, `Util.js`; `CONTRATO.md`. `system/mk/apps-links`.

## Ligações

Molde: [[Apps do Mk (motor em Go, janela em QML)]]. Depende de: [[Polkit]], [[Pacman e AUR]], [[Apps oferecidos]] (o `ffmpeg`). Relacionado: [[MkPacotes]], [[Apps padrão]].
