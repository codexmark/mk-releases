# MkCorte: o contrato entre o motor e a janela

O MkCorte é o editor de cortes de vídeo do Mk (`mkcorte [arquivos…]`): abre um
vídeo, apara as pontas, divide em clipes, remove trechos, junta outros vídeos,
reordena, pré-visualiza e exporta um MP4.

Tem duas metades:

- **o motor** (`src/`, em Go, um binário só: `bin/mkcorte-motor`), que
  pergunta ao ffprobe o que cada vídeo é, gera a tira de quadros da linha do
  tempo e chama o ffmpeg para exportar. É todo testado com pastas falsas e com
  um ffmpeg e um ffprobe de mentira que só anotam os argumentos;
- **a janela** (`qml/`, Quickshell, com os tokens do Maré), que guarda a
  edição, toca a pré-visualização (QtMultimedia) e pede ao motor o que a
  pessoa clicou. O modelo da edição (clipes, dividir, aparar, desfazer…) mora
  em `qml/Corte.js`, testado no node.

Este arquivo é o contrato entre as duas. Muda aqui primeiro, depois nos dois
lados.

## Regras gerais

- O motor escreve JSON em UTF-8 no stdout, um objeto por linha. Um subcomando
  que falha escreve `{"erro": "frase em português"}` e sai com código
  diferente de 0: **1** é erro (arquivo que não existe, ffmpeg que falhou),
  **2** é um pedido que o motor recusa (exportar sem recodificar quando não
  dá). **0** é certo. A janela mostra a frase.
- O que o motor não conseguiu saber é `null`, nunca 0 nem ausente.
- Tamanhos em **bytes** (`_bytes`), taxas em **bits por segundo** (`_bps`),
  tempos em **segundos** (`_s`, com três casas).
- `exportar` solta uma linha por passo, cada uma com `"fase"`; a última é
  `{"fase": "fim", …}`, `{"fase": "cancelado"}` ou `{"erro": …}`.
- **Nome de arquivo nunca passa por shell**, e o ffmpeg e o ffprobe nem veem o
  nome: recebem um atalho de nome neutro (`fonte0.mp4`, `fonte1.mkv`…) numa
  pasta temporária, porque um nome pode começar com `-`, ter aspas, `'`, `%`
  ou `:` (que eles leem como opção, padrão de sequência ou protocolo). No
  próprio motor, o que vem depois de `--` é sempre arquivo
  (`mkcorte-motor sondar -- -nome.mp4`); a janela manda caminhos absolutos.
- Comandos externos (`ffmpeg`, `ffprobe`) são achados no `PATH`: o teste põe
  dublês na frente. O motor não chama mais nada.
- **O motor nunca escreve em cima de um arquivo que existe**, nem do original:
  a saída vai para um arquivo escondido `.mkcorte-….parcial` na pasta de
  destino e só no fim ganha o nome; se o nome está tomado, vira
  `nome (2).mp4`, `nome (3).mp4`…
- Variáveis de ambiente do motor: `MKCORTE_HOME` (a home, para escrever os
  caminhos com `~` e achar o cache) e `XDG_CACHE_HOME` (o cache das miniaturas
  fica em `${XDG_CACHE_HOME:-~/.cache}/mk/mkcorte/`).

## `mkcorte-motor sondar <arquivo> [--quadros-chave]`

O que o vídeo é. Uma linha:

```json
{ "arquivo": "/home/ana/Vídeos/passeio.mp4", "tamanho_bytes": 48000000, "duracao_s": 62.4,
  "largura": 1920, "altura": 1080, "fps": 29.97, "fps_fracao": "30000/1001",
  "codec": "h264", "perfil": "High", "formato_de_pixel": "yuv420p", "sar": "1:1", "rotacao": 0,
  "taxa_bps": 6153846, "trilha": 0, "tem_audio": true,
  "audio": { "codec": "aac", "taxa_bps": 128000, "canais": 2, "amostragem_hz": 48000, "trilha": 1 } }
```

- `largura` e `altura` são como o vídeo aparece na tela: com a `rotacao` (0,
  90, 180 ou 270; um vídeo de celular em pé vem gravado deitado) e o pixel não
  quadrado (`sar`) já aplicados.
- `trilha` é o índice da trilha de vídeo no arquivo (a capa de um arquivo de
  música não conta); `audio` é a primeira trilha de áudio, ou `null`.
- `taxa_bps` é a do arquivo inteiro.
- `--quadros-chave` acrescenta `"quadros_chave": [0, 2.002, 4.004, …]`, os
  instantes dos quadros-chave. Lê o arquivo inteiro (sem decodificar), então
  demora num vídeo longo: por isso não vem por padrão, e a janela não pede.
- Erros: o arquivo não existe, só tem áudio, é uma imagem parada, ou o ffprobe
  não o entende.

## `mkcorte-motor miniaturas <arquivo> [--quantas N] [--altura N]`

A tira de quadros que a linha do tempo desenha em cima dos clipes: `N`
miniaturas JPEG (padrão 12, máximo 60) com a altura dada (padrão 54, de 16 a
360), uma do meio de cada fatia do vídeo.

```json
{ "arquivo": "/home/ana/Vídeos/passeio.mp4", "duracao_s": 62.4, "altura": 54,
  "pasta": "/home/ana/.cache/mk/mkcorte/3f9a1c…", "do_cache": false,
  "quadros": [ { "tempo_s": 2.6, "arquivo": "/home/ana/.cache/mk/mkcorte/3f9a1c…/q01.jpg" }, … ] }
```

- O cache é uma pasta por tira; a chave é um resumo do caminho, do tamanho, da
  data de modificação do arquivo e do pedido (quantas, altura). Mexeu no
  vídeo, a tira é outra. Na segunda vez (`do_cache: true`) nada roda.
- Um quadro que o ffmpeg não conseguiu tirar fica de fora da lista; nenhum é
  erro. Tiras que ninguém usa há 30 dias são apagadas na geração seguinte.

## A edição

É o que a janela manda para o `plano` e o `exportar`:

```json
{ "fontes": ["/home/ana/Vídeos/passeio.mp4", "/home/ana/Vídeos/em pé.mov"],
  "clipes": [ { "fonte": 0, "inicio": 12.5, "fim": 20 },
              { "fonte": 1, "inicio": 0, "fim": 4.2 },
              { "fonte": 0, "inicio": 0, "fim": 6 } ],
  "qualidade": "original", "modo": "exato" }
```

- `clipes` sai na ordem dada; cada um é o trecho `[inicio, fim)` do vídeo
  `fontes[fonte]`. Até 64 vídeos e 400 clipes.
- `qualidade`: `original`, `1080` ou `720`: o limite do lado menor do quadro.
  Nunca aumenta, preserva a proporção e as dimensões saem pares.
- `modo`: `exato` (recodifica; o corte é exato ao quadro) ou `rapido` (copia
  os pacotes, sem recodificar; veja abaixo).

A edição chega de três jeitos: um arquivo (`plano edicao.json`), o stdin
(`plano -`) ou o texto na linha de comando (`plano --json '<texto>'`, que é o
que a janela usa).

## `mkcorte-motor plano (<edicao.json> | - | --json <texto>) [saida]`

Sem rodar o ffmpeg nem escrever nada, diz como a edição seria exportada. Sonda
os vídeos e responde uma linha:

```json
{ "modo": "exato", "duracao_s": 18.2, "clipes": 3, "largura": 1920, "altura": 1080, "fps": 29.97,
  "tem_audio": true, "tamanho_estimado_bytes": 14300000,
  "sem_recodificar": { "pode": false, "motivo": "os clipes vêm de vídeos com parâmetros diferentes (…)", "aviso": "" },
  "saida": "/home/ana/Vídeos/passeio (corte).mp4",
  "comando": ["ffmpeg", "-hide_banner", "…", "-i", "fonte0.mp4", "…", "/home/ana/Vídeos/passeio (corte).mp4"],
  "avisos": ["Há clipes com outra proporção: entram com faixas pretas, no quadro do primeiro clipe."] }
```

- `saida` é o nome que a exportação teria agora: o caminho pedido (a extensão
  vira `.mp4`) ou, sem ele, `nome (corte).mp4` ao lado do vídeo do primeiro
  clipe; se já existe, `… (2).mp4`.
- `comando` é a linha de comando do ffmpeg, com os atalhos pelo nome
  (`fonte0.mp4`) no lugar dos vídeos.
- `tamanho_estimado_bytes` é uma estimativa: no modo rápido, a fatia de cada
  arquivo; no exato, o que o CRF costuma gastar nesse quadro, sem passar da
  taxa do original.
- `sem_recodificar.pode` diz se o modo rápido serve para esta edição; se não,
  `motivo` diz por quê. Se sim, `aviso` traz a ressalva do modo (abaixo), que a
  janela mostra antes de a pessoa escolher.
- Pedir `"modo": "rapido"` quando não dá é **recusado** (código 2): o motor não
  troca de modo calado.
- Um clipe que passa do fim do vídeo por menos de meio segundo (arredondamento)
  é cortado no fim; por mais que isso, é erro.

### Modo exato (recodifica)

Uma entrada do ffmpeg por clipe, já posicionada no início dele (`-ss` e `-t`
antes do `-i`, que com decodificação é exato ao quadro), e um grafo de filtros:

- vídeo de cada clipe: `trim`, `setpts`, `scale` (se o tamanho não é o do
  quadro), `pad` com preto (se a proporção é outra), `setsar=1`, `fps` (se há
  mais de um vídeo: todos passam para a taxa do primeiro) e `format=yuv420p`;
- áudio de cada clipe: `atrim`, `asetpts`, `aresample=48000` e `aformat`
  (estéreo); um clipe de um vídeo sem áudio ganha silêncio (`anullsrc`). Se
  nenhum vídeo tem áudio, a saída não tem áudio;
- tudo emendado pelo filtro `concat`.

O quadro (tamanho e taxa de quadros) é o do vídeo do **primeiro clipe**. A
saída é MP4 com H.264 (`libx264`, CRF 20) e AAC (160 kbps), com `+faststart`.

### Modo rápido (sem recodificar)

Os pacotes são copiados como estão (`-codec copy`), pelo demuxer `concat` do
ffmpeg, com uma lista que diz o trecho de cada clipe. Leva segundos e não perde
qualidade. Só é oferecido quando:

- a qualidade pedida não reduz o quadro;
- todos os clipes vêm do mesmo vídeo, ou de vídeos com parâmetros idênticos
  (codec, perfil, resolução, formato de pixel, taxa de quadros, rotação e o
  mesmo áudio);
- o codec entra num MP4 (vídeo H.264, HEVC, AV1, MPEG-4 ou VP9; áudio AAC,
  MP3, AC-3, E-AC-3, Opus, ALAC ou FLAC).

**A ressalva, dita com franqueza:** sem recodificar, um corte só pode começar
num quadro-chave. Cada clipe começa no quadro-chave **anterior** ao ponto
marcado, então pode começar um pouco antes (até alguns segundos, conforme o
vídeo) e a duração final pode passar da prevista. O fim de cada clipe cai no
ponto marcado. Quem precisa do corte exato ao quadro usa o modo exato. Quando
os clipes vêm de arquivos diferentes, o plano avisa também que a emenda pode
sair com defeito em alguns casos.

## `mkcorte-motor exportar (<edicao.json> | - | --json <texto>) [saida]`

Exporta. Solta o plano, o progresso e o fim:

```json
{ "fase": "plano", "modo": "exato", "duracao_s": 18.2, "saida": "/home/ana/Vídeos/passeio (corte).mp4", "…": "os campos do plano" }
{ "fase": "exportando", "pct": 42.5, "velocidade": 2.8, "resta_s": 4, "tamanho_bytes": 6100000 }
{ "fase": "fim", "ok": true, "saida": "/home/ana/Vídeos/passeio (corte).mp4", "modo": "exato", "duracao_s": 18.2, "tamanho_bytes": 13900000 }
```

- `"fase": "plano"` traz todos os campos do `plano`.
- `"fase": "exportando"`: `pct` de 0 a 99,9; `velocidade` (quantas vezes o
  tempo real) e `resta_s` são `null` enquanto o ffmpeg não diz.
- `"fase": "fim"`: `saida` é o nome que o arquivo ganhou de verdade (pode ter
  mudado desde o plano, se outro arquivo tomou o nome no meio).
- **Cancelar** é mandar SIGTERM ao motor (a janela faz isso ao parar o
  processo): ele mata o ffmpeg, apaga o `.parcial`, escreve
  `{"fase": "cancelado"}` e sai com 0. Fechar a janela no meio também cancela.
- Se o ffmpeg falha, o `.parcial` é apagado e a última linha é o erro.

## `mkcorte-motor quadro <arquivo> <tempo> [saida]`

Salva o quadro do instante dado (em segundos) como PNG, do tamanho do vídeo:

```json
{ "ok": true, "saida": "/home/ana/Vídeos/passeio (quadro 0-00-12.500).png", "tempo_s": 12.5, "tamanho_bytes": 2400000 }
```

Sem `saida`, fica ao lado do vídeo, com o instante no nome. Nunca sobrescreve
(`… (2).png`). Um tempo depois do fim do vídeo é erro.

## A janela

- Título e classe da janela: `MkCorte`. `mkcorte [arquivos…]` abre no primeiro
  vídeo e junta os outros na ordem; sem arquivo e com a janela já aberta, só a
  traz para a frente. Os arquivos chegam em `MKCORTE_ARQUIVOS` (uma lista JSON
  de caminhos absolutos).
- `MKCORTE_MOTOR` troca o binário; `MKCORTE_EXEMPLO=pasta` faz a janela usar
  respostas gravadas (`sondar.json`, `miniaturas.json`, `plano.json`,
  `exportar.json`, `quadro.json`, como em `tests/apps/mkcorte-exemplo/`) em vez
  do motor.
- A janela só roda o motor (e o `jq`, para os exemplos). "Mostrar na pasta"
  abre a pasta da saída pelo próprio Qt.
- A edição mora na janela (`qml/Corte.js`): o motor só a recebe pronta.
  Desfazer e refazer são da janela.
