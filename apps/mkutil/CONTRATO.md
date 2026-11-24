# MkUtil: o contrato entre o motor e as janelas

O MkUtil são dois utilitários pequenos do Mk num app só:

- **MkInstalar** (`mkinstalar [arquivos…]`): instala um AppImage, um pacote do
  Arch ou um arquivo compactado que a pessoa baixou, e desinstala o que ele
  mesmo instalou;
- **MkComprimir** (`mkcomprimir [arquivos…]`): encolhe vídeos e imagens com o
  ffmpeg.

Tem duas metades:

- **o motor** (`src/`, em Go, um binário só: `bin/mkutil-motor`), que olha os
  arquivos e faz o trabalho. É onde mora a lógica que mexe no disco, e é todo
  testado com homes falsas, arquivos gerados no próprio teste e comandos de
  mentira;
- **as janelas** (`qml/`, Quickshell, com os tokens do Maré), que mostram o que
  o motor diz e pedem o que a pessoa clicou. A conta que a janela faz sozinha
  (a estimativa de tamanho, a fila, os textos) mora em `qml/Util.js`, testado
  no node.

Este arquivo é o contrato entre as duas. Muda aqui primeiro, depois nos dois
lados.

## Regras gerais

- O motor escreve JSON em UTF-8 no stdout, um objeto por linha. Um subcomando
  que falha escreve `{"erro": "frase em português"}` e sai com código
  diferente de 0. A janela mostra a frase.
- O que o motor não conseguiu saber é `null`, nunca 0 nem ausente.
- Tamanhos em **bytes** (`_bytes`), taxas em **bits por segundo** (`_bps`),
  tempos em **segundos** (`_s`).
- Um subcomando demorado (`instalar`, `comprimir`) solta uma linha por passo,
  cada uma com `"fase"`; a última é `{"fase": "fim", …}`,
  `{"fase": "cancelado"}` ou `{"erro": …}`.
- **Nome de arquivo nunca passa por shell.** Todo comando externo roda com
  lista de argumentos, o caminho vai absoluto e depois de `--` quando o
  comando aceita. No próprio motor, o que vem depois de `--` é sempre arquivo
  (`mkutil-motor inspecionar -- -nome.zip`); as janelas mandam caminhos
  absolutos. O ffmpeg e o ffprobe nem veem o nome: recebem um atalho
  (`entrada.<ext>`) numa pasta temporária, porque o nome de uma imagem com `%`
  ou `*` é lido por eles como padrão de sequência.
- Comandos externos (`ffmpeg`, `ffprobe`, `pkexec`, `pacman`, `bsdtar`,
  `zstd`, `xz`, `update-desktop-database`) são achados no `PATH`: o teste põe
  dublês na frente.
- O motor nunca pede senha, nunca roda como root e nunca chama o `sudo`. O que
  precisa de root (instalar um pacote) passa pelo `pkexec`: o Maré é o agente
  do polkit e mostra o pedido de senha. O programa que o `pkexec` roda vai por
  extenso (`/usr/bin/pacman`): com o nome solto ele procuraria no `PATH` de
  quem chamou, e rodaria como root o primeiro `pacman` que achasse.
- **O motor nunca executa o arquivo que está inspecionando.** Um AppImage é
  lido por dentro (o squashfs dele), não rodado com `--appimage-extract`.
- Variáveis de ambiente do motor: `MKUTIL_HOME` (a home; dela saem
  `~/Aplicativos`, `~/.local/share/applications` e
  `~/.local/state/mk/instalados.json`), `MKUTIL_LIBS` (onde procurar a
  `libfuse.so.2`, separado por `:`; padrão `/usr/lib:/usr/lib64:/lib`),
  `MKUTIL_LIMITE_MB` (quanto um arquivo compactado pode ocupar depois de
  extraído; padrão 8192) e `MKUTIL_LIMITE_ENTRADAS` (quantas entradas ele pode
  ter; padrão 200000).

## MkInstalar

### `mkutil-motor inspecionar <arquivo>`

Diz o que o arquivo é e o que instalar faria. Não muda nada no disco.

```json
{
  "arquivo": "/home/ana/Downloads/Krita-5.2.2-x86_64.AppImage",
  "tipo": "appimage",
  "id": "krita",
  "nome": "Krita",
  "versao": "5.2.2",
  "descricao": "Pintura digital",
  "tamanho_bytes": 412000000,
  "destino": "/home/ana/Aplicativos/krita.AppImage",
  "atalho": "/home/ana/.local/share/applications/mkinstalar-krita.desktop",
  "ja_instalado": false,
  "precisa_senha": false,
  "plano": ["copiar para ~/Aplicativos/krita.AppImage", "marcar como executável", "criar o atalho no menu"],
  "avisos": [],
  "appimage": { "tipo": 2, "tem_desktop": true, "tem_icone": true, "fuse": true },
  "pacote": null,
  "compactado": null
}
```

- **`tipo`**: `"appimage"`, `"pacote"` (`.pkg.tar.zst`, `.pkg.tar.xz`) ou
  `"compactado"` (`.tar.gz`/`.tgz`, `.tar.xz`, `.tar.zst`, `.tar.bz2`, `.zip`).
  O resto é erro: `.deb`, `.rpm` e `.flatpakref` com a frase que diz por quê.
- **`id`**: o nome do arquivo sem extensão, sem versão e sem arquitetura
  (`Krita-5.2.2-x86_64.AppImage` → `krita`), só com `a-z 0-9 . _ + -`. É o
  nome no disco, a chave no registro e o que o `desinstalar` recebe.
- **`plano`** e **`avisos`**: frases prontas, em português, para mostrar.
- **`ja_instalado`**: o `id` está no registro. Instalar de novo pede
  `--substituir`.
- **`appimage`**: `tipo` 1 ou 2 (a magia `AI\x01`/`AI\x02` no deslocamento 8
  de um ELF). `tem_desktop` e `tem_icone` dizem se deu para ler o atalho e o
  ícone de dentro (só no tipo 2, lendo o squashfs; compressão gzip pela
  biblioteca padrão, zstd e xz pelos comandos `zstd` e `xz` se existirem).
  Quando não deu, o atalho sai mínimo: o nome do arquivo e um ícone genérico.
  `fuse` é `false` quando o AppImage precisa da `libfuse.so.2` e ela não está
  na máquina: o aviso diz o pacote (`fuse2`).
- **`pacote`** (do `.PKGINFO`, lido com o `bsdtar` ou, na falta dele, com o
  `zstd`/`xz`):

  ```json
  { "nome": "htop", "versao": "3.3.0-3", "descricao": "Interactive process viewer", "arquitetura": "x86_64",
    "tamanho_instalado_bytes": 412000, "dependencias": ["libcap", "ncurses"], "instalado": "3.2.2-1" }
  ```

  `instalado` é a versão que o `pacman -Q` diz ter (`null` se não tem).
  `precisa_senha` é `true`, e `destino` e `atalho` são `null`.
- **`compactado`**:

  ```json
  { "formato": "tar.xz", "entradas": 812, "tamanho_total_bytes": 250000000, "pasta_unica": "firefox",
    "executaveis": ["firefox", "firefox-bin"], "atalhos": ["firefox.desktop"], "executavel_sugerido": "firefox" }
  ```

  `pasta_unica` é a pasta do topo quando tudo está dentro de uma só (ela some
  na extração: o conteúdo vai direto para `~/Aplicativos/<id>`). Os caminhos
  de `executaveis` (no máximo 40) e de `atalhos` já são relativos ao destino.
  `executavel_sugerido` é o que tem o nome do app, ou o mais raso; `null` se
  não há executável (aí não há atalho de menu).

**Recusas (código 2), já na inspeção**: uma entrada com caminho absoluto, com
`..`, com `\` ou vazia; um link (simbólico ou físico) que aponta para fora da
pasta; um arquivo especial (dispositivo, fifo, soquete); mais entradas ou mais
bytes que o limite. O arquivo inteiro é recusado: nada é extraído pela metade.

### `mkutil-motor instalar <arquivo> [--substituir] [--sem-atalho] [--executavel <caminho>]`

Faz o que o `inspecionar` planejou, soltando o progresso:

```json
{"fase": "copiando", "pct": 40.0}
{"fase": "extraindo", "pct": 62.5, "entrada": "firefox/libxul.so"}
{"fase": "pacman", "linha": "(1/1) instalando htop"}
{"fase": "atalho"}
{"fase": "fim", "ok": true, "id": "krita", "tipo": "appimage", "nome": "Krita", "destino": "…", "atalho": "…", "avisos": []}
```

- **AppImage**: cria `~/Aplicativos`, copia para `<id>.AppImage` (por um
  arquivo temporário, trocado no fim; o atalho e o ícone são lidos da cópia,
  que é o que fica instalado), dá `+x`, guarda o ícone em
  `~/Aplicativos/.icones/` e grava `mkinstalar-<id>.desktop` com `Exec` e
  `TryExec` no caminho novo. Do atalho de dentro só passam as chaves que não
  executam nada (`Name`, `GenericName`, `Comment`, `Categories`, `MimeType`,
  `Keywords`, `StartupWMClass`, `Terminal` e as traduções) e os argumentos do
  `Exec` original.
- **Pacote**: `pkexec /usr/bin/pacman -U --needed --noconfirm -- <arquivo>`, com cada
  linha do pacman numa fase `pacman`. A senha recusada (o pkexec sai com 126
  ou 127) vira o erro `"a senha não foi dada: nada foi instalado"`. Pacote não
  entra no registro: quem o tira é o pacman.
- **Compactado**: extrai para uma pasta temporária dentro de `~/Aplicativos`,
  conferindo de novo cada entrada (as recusas acima, contando os bytes de
  verdade) e sem as permissões especiais (setuid, setgid); só no fim a pasta
  ganha o nome `<id>`. `--executavel` escolhe para quem é o atalho (tem de ser
  um arquivo de dentro); sem ele vale o `executavel_sugerido`.
- `--sem-atalho` não cria o atalho. `--substituir` troca o que o MkInstalar já
  tinha instalado com o mesmo `id`; um arquivo ou pasta que está em
  `~/Aplicativos` e não é do registro nunca é trocado (erro).
- No fim roda `update-desktop-database` na pasta dos atalhos, se existir.

### `mkutil-motor instalados`

```json
{ "itens": [
  { "id": "krita", "tipo": "appimage", "nome": "Krita", "versao": "5.2.2", "origem": "Krita-5.2.2-x86_64.AppImage",
    "destino": "/home/ana/Aplicativos/krita.AppImage", "atalho": "/home/ana/.local/share/applications/mkinstalar-krita.desktop",
    "icone": "/home/ana/Aplicativos/.icones/krita.png", "instalado_em_s": 1795470000, "tamanho_bytes": 412000000, "existe": true }
] }
```

O registro é `~/.local/state/mk/instalados.json`. `existe` é `false` quando o
destino sumiu (a pessoa apagou à mão): desinstalar ainda limpa o atalho.
`atalho` e `icone` são `null` quando não há.

### `mkutil-motor desinstalar <id>`

Apaga o destino, o atalho e o ícone do item e o tira do registro. Resposta:
`{"ok": true, "id": "krita", "removidos": ["…", "…"]}`.

O registro é um arquivo que qualquer programa da pessoa pode escrever, então o
que está nele não é confiado. **Recusa (código 2), sem apagar nada**, se: o
destino não é filho direto de `~/Aplicativos` (ou começa com `.`); o atalho
não é `mkinstalar-*.desktop` direto em `~/.local/share/applications`; o ícone
não está direto em `~/Aplicativos/.icones`. Um link simbólico é apagado como
link, nunca seguido.

## MkComprimir

### `mkutil-motor codificadores`

O que o ffmpeg desta máquina sabe fazer, para a janela só oferecer o que
existe:

```json
{ "ffmpeg": true, "codecs": ["h264", "h265", "av1"], "aceleracao": ["vaapi", "nvenc"],
  "imagens": ["jpeg", "webp", "avif"], "animados": ["gif", "webp"] }
```

`aceleracao` lista o que o `ffmpeg -encoders` tem, não o que a placa aguenta:
é a pessoa que pede. Sem o ffmpeg, `"ffmpeg": false` e listas vazias.

### `mkutil-motor midia <arquivo>`

O que o `ffprobe -of json` diz:

```json
{ "arquivo": "/home/ana/Vídeos/férias.mp4", "tipo": "video", "tamanho_bytes": 48000000, "duracao_s": 62.4,
  "largura": 1920, "altura": 1080, "fps": 30.0, "codec": "h264", "taxa_bps": 6153846,
  "audio": { "codec": "aac", "taxa_bps": 128000, "canais": 2 } }
```

`tipo`: `"video"` (inclusive GIF animado) ou `"imagem"`. Um arquivo só de
áudio, ou que o ffprobe não entende, é erro. `largura` e `altura` já vêm como
a pessoa vê (um vídeo de celular em pé sai 1080 × 1920). Numa imagem,
`duracao_s`, `fps`, `taxa_bps` e `audio` são `null`.

### `mkutil-motor comprimir <arquivo> --predefinicao <qual> [opções]`

| `--predefinicao` | O que faz | Sufixo da saída |
|---|---|---|
| `menor` | qualidade 30, no máximo 720p, áudio a 64 kb/s | `-menor.mp4` |
| `equilibrio` | qualidade 24, no máximo 1080p, áudio a 128 kb/s | `-equilibrio.mp4` |
| `alta` | qualidade 19, na resolução original, áudio a 160 kb/s | `-alta.mp4` |
| `caber` | cabe em `--alvo-mb N`: a taxa sai da duração, em duas passadas | `-<N>mb.mp4` |
| `resolucao` | só reduz para `--lado 1080\|720\|480`, qualidade 22 | `-<lado>p.mp4` |
| `sem-audio` | tira o áudio sem recodificar o vídeo | `-sem-audio.<ext original>` |
| `gif` | GIF animado, `--fps` (padrão 12), `--largura` (padrão 480) | `.gif` |
| `webp` | WebP animado, `--fps` (padrão 15), `--largura` (padrão 640) | `.webp` |
| `imagem` | imagem: `--formato jpeg\|webp\|avif`, `--qualidade 1-100` (padrão 80), `--largura` | `-menor.<jpg\|webp\|avif>` |

Opções: `--codec h264|h265|av1` (padrão `h264`, pela compatibilidade;
"qualidade" é o CRF do H.264, e vira +4 no H.265 e +8 no AV1);
`--acelerar vaapi|nvenc` (só se a pessoa pedir, e recusado se o ffmpeg não
lista o codificador); `--sem-audio` (junto de qualquer predefinição de vídeo).
`--lado` é o lado menor do quadro; nada é ampliado, e as dimensões saem pares.

**A conta do `caber`** (a janela faz a mesma em `Util.js`; os dois testes usam
os mesmos casos, de `src/internal/midia/testdata/caber.json`):

    total_bps = alvo_mb × 8.000.000 × 0,97 ÷ duração        (MB de 1.000.000 bytes; 3% de folga para o contêiner; arredonda para baixo)
    áudio_bps = 0 sem áudio; 96.000 se total ≥ 500.000; 64.000 se ≥ 200.000; senão 32.000
    vídeo_bps = total_bps − áudio_bps                        (abaixo de 50.000 é erro: não cabe)
    lado      = o maior entre o original, 1080, 720, 480, 360 e 240 (sem ampliar) em que
                vídeo_bps ÷ (largura × altura × fps) ≥ 0,045; se nenhum, 240

Duas passadas no H.264 e no H.265 por software; nos outros, uma passada com a
taxa média.

**A saída** fica ao lado do original. O original nunca é tocado e nenhum
arquivo que já existe é sobrescrito: se `férias-menor.mp4` existe, sai
`férias-menor-2.mp4`. O ffmpeg escreve num arquivo escondido
(`.mkcomprimir-….parcial.<ext>`), que só ganha o nome no fim; cancelado ou com
erro, ele é apagado.

O progresso vem do `-progress pipe:1` do ffmpeg:

```json
{"fase": "plano", "saida": "/home/ana/Vídeos/férias-8mb.mp4", "passadas": 2, "video_bps": 898871, "audio_bps": 96000, "largura": 854, "altura": 480}
{"fase": "codificando", "passada": 1, "passadas": 2, "pct": 18.4, "velocidade": 3.1, "resta_s": 41, "tamanho_bytes": 0}
{"fase": "fim", "ok": true, "saida": "…", "antes_bytes": 48000000, "depois_bytes": 7600000, "economia_pct": 84.2, "maior": false}
```

- `pct` é do trabalho inteiro (as duas passadas juntas), pela duração.
  `velocidade` é quantas vezes o tempo real; `resta_s` é `null` enquanto o
  ffmpeg não disse a velocidade. Numa imagem não há `codificando`.
- `video_bps`, `audio_bps` só vêm no `caber`; `largura` e `altura` são as da
  saída.
- `maior` é `true` quando a saída ficou maior que o original (`economia_pct`
  negativa): a janela avisa e oferece apagar (`descartar`).
- **Cancelar** é mandar SIGTERM (ou SIGINT) ao motor, ou fechar o stdout dele:
  ele mata o ffmpeg, apaga a saída pela metade, escreve
  `{"fase": "cancelado"}` e sai com 0.

A fila de vários arquivos é da janela: um `comprimir` por vez.

### `mkutil-motor descartar <saida> --original <arquivo>`

Apaga uma saída do `comprimir`. Recusa (código 2) o que não é uma: tem de ser
um arquivo comum, na mesma pasta do original, com o nome começando pelo nome
do original (sem extensão) e não ser o próprio original. Resposta:
`{"ok": true, "apagado": "…"}`.

## As janelas

- `bin/mkinstalar [arquivos…]` e `bin/mkcomprimir [arquivos…]` abrem a janela
  (títulos `MkInstalar` e `MkComprimir`) já nos arquivos dados. Sem arquivo e
  com a janela aberta, só a trazem para a frente.
- Os lançadores passam `MKUTIL_JANELA` (`instalar` ou `comprimir`) e
  `MKUTIL_ARQUIVOS` (uma lista JSON de caminhos absolutos).
- `MKUTIL_MOTOR` troca o binário. `MKUTIL_EXEMPLO` aponta uma pasta de
  respostas gravadas (`inspecionar-appimage.json`, `midia-video.json`,
  `instalar.json`…: o nome sai de `Util.arquivoDeExemplo`), para ver as
  janelas sem o motor; as respostas em fases são uma lista JSON, uma fase por
  item.

## Códigos de saída

`0` certo (inclusive cancelado); `1` erro de uso ou de sistema (com
`{"erro": …}`); `2` recusado por segurança (caminho que sai da pasta, bomba,
arquivo especial, apagar fora das pastas do MkInstalar), também com
`{"erro": …}`.
