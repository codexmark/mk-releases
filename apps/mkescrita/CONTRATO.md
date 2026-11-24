# MkEscrita: o contrato entre o motor e a janela

O MkEscrita (`mkescrita [arquivo…]`) é o editor de Markdown do Mk: uma coluna
de texto e mais nada. Tem duas metades:

- **o motor** (`src/`, em Go, um binário só: `bin/mkescrita-motor`), que é
  quem toca o disco e entende o Markdown: abre, grava sem perder texto, guarda
  os rascunhos, vigia o arquivo, converte para HTML, conta palavras. É todo
  testado com pastas temporárias;
- **a janela** (`qml/`, Quickshell, com os tokens do Maré), que mostra o texto
  e pede ao motor o que a pessoa fez. As decisões que ela toma sozinha (as
  edições de Markdown, a busca, quando salvar, o que fazer ao fechar) moram em
  `qml/Escrita.js`, testado no node.

Este arquivo é o contrato entre as duas. Muda aqui primeiro, depois nos dois
lados.

## Regras gerais

- O motor escreve JSON em UTF-8 no stdout, um objeto por linha. Um subcomando
  que falha escreve `{"erro": "frase em português"}` e sai com código
  diferente de 0. A janela mostra a frase.
- Códigos de saída: **0** deu certo; **1** erro (argumento errado, sem
  permissão, disco cheio); **2** recusa: o motor entendeu e não vai fazer
  (arquivo binário, grande demais, saída que já existe) **ou conflito** no
  `salvar` (aí a resposta não tem `erro`: tem `conflito`).
- O que o motor não sabe é `null`, nunca ausente.
- **O texto vai pelo stdin**, nunca pela linha de comando. Quem manda diz
  quantos bytes mandou (`--bytes N`): se chegou outro tanto, a conversa foi
  cortada e o motor **não grava nada**. Texto que não é UTF-8 é recusado.
- **Nome de arquivo nunca passa por shell**, e o que vem depois de `--` é
  sempre arquivo (`mkescrita-motor abrir -- -nome.md`). A janela manda caminhos
  absolutos.
- O motor não roda nenhum comando externo, não usa a rede e não tem
  dependência além da biblioteca padrão do Go.
- Tamanhos em **bytes** (`_bytes`), tempos em **segundos** (`_s`).
- Variáveis de ambiente do motor: `XDG_STATE_HOME` (os rascunhos ficam em
  `${XDG_STATE_HOME:-~/.local/state}/mk/mkescrita/rascunhos/`),
  `XDG_CONFIG_HOME` (as preferências, em
  `${XDG_CONFIG_HOME:-~/.config}/mk/mkescrita.json`), `MKESCRITA_LIMITE_MB` (o
  maior texto que se abre ou grava; padrão 4) e `MKESCRITA_PROC` (onde olhar
  se a janela dona de um rascunho ainda vive; padrão `/proc`).
- Variáveis da janela: `MKESCRITA_ARQUIVO` (o arquivo a abrir, posto pelo
  lançador), `MKESCRITA_MOTOR` (troca o binário) e `MKESCRITA_EXEMPLO` (uma
  pasta de respostas gravadas, para ver a janela sem o motor:
  `tests/apps/mkescrita-exemplo/`).

## A impressão de um arquivo

É o retrato do arquivo no disco; com ela a janela sabe se o que está lá ainda
é o que ela abriu ou gravou.

```json
{"tamanho_bytes": 624, "mtime_ns": 1795500000000000000, "sha256": "8fe4…"}
```

Quem decide se o arquivo mudou é o **`sha256`** (um `touch` não é mudança). O
`mtime_ns` é informativo (e não cabe num número de JavaScript sem perder os
últimos algarismos: a janela não o manda de volta).

## `mkescrita-motor abrir <arquivo>`

```json
{"arquivo": "/textos/notas.md", "nome": "notas.md", "novo": false,
 "texto": "# Olá\n", "impressao": {…},
 "fim_de_linha": "lf", "quebra_final": true, "so_leitura": false}
```

- `fim_de_linha`: `"lf"`, `"crlf"` ou `"misto"`. Com `crlf` o `texto` vem só
  com `\n` e o `salvar --fim-de-linha crlf` devolve os `\r\n`. Com `misto` o
  texto vem como está no disco e é gravado como veio.
- `quebra_final`: o arquivo terminava com quebra de linha (o `texto` a traz).
- `novo: true`: o arquivo ainda não existe (a pasta, sim): `texto` vazio e
  `impressao` `null`. Se nem a pasta existe, é erro.
- `so_leitura`: quem roda não pode escrever nele.
- **Recusa (saída 2)**: pasta, arquivo que não é comum, com byte nulo
  ("parece um arquivo binário"), que não é UTF-8 (abrir e salvar trocaria as
  letras) ou maior que o limite.

## `mkescrita-motor salvar [opções] <arquivo>`

O texto vem pelo stdin.

| Opção | O que faz |
|---|---|
| `--esperado <sha256\|ausente>` | o que a janela conhece do disco: o `sha256` da impressão, ou `ausente` para um arquivo que não existia |
| `--forcar` | grava mesmo que o disco não esteja como o esperado |
| `--fim-de-linha crlf` | grava cada `\n` como `\r\n` (`lf` e `misto`: como veio) |
| `--quebra-final` | garante que um texto não vazio termine com quebra de linha |
| `--bytes N` | quantos bytes vêm pelo stdin |

Sem `--esperado` e sem `--forcar` é erro: o motor não grava sem saber o que
esperar. Quando grava:

```json
{"arquivo": "/textos/notas.md", "nome": "notas.md", "impressao": {…}}
```

- A escrita é **atômica**: um temporário na mesma pasta
  (`.<nome>.mkescrita-*`), `fsync`, e só então a troca de nome. Se algo falha,
  o arquivo antigo fica como estava e o temporário some.
- As permissões do arquivo são mantidas (e o dono, quando dá); um arquivo novo
  nasce com a `umask` de quem roda.
- Um **link simbólico** é seguido: grava-se no alvo e o link continua link
  (mesmo que o alvo ainda não exista).
- Arquivo só de leitura: erro, nada é trocado.

**Conflito (saída 2, sem `erro`)**: o disco não está como o esperado e nada foi
gravado. Quem decide é a pessoa (recarregar, manter a dela com `--forcar`, ou
salvar como):

```json
{"conflito": "mudou", "frase": "o arquivo mudou por fora desde que foi aberto", "impressao": {…}}
```

`conflito` é `"mudou"` (outro conteúdo), `"sumiu"` (não existe mais;
`impressao` `null`) ou `"apareceu"` (era `ausente` e agora existe).

## `mkescrita-motor rascunho <guardar|listar|ler|apagar>`

O texto ainda não salvo, guardado a cada pausa na digitação para o caso de o
app morrer. Um arquivo por rascunho (`<id>.json`, só da pessoa: `0600`), na
pasta dos rascunhos. O `id` é da janela: `[a-z0-9][a-z0-9-]{0,63}`.

- `rascunho guardar <id> [--origem <arquivo>] [--base <sha256>] [--bytes N]`
  (texto pelo stdin): grava ou regrava. `--origem` é o arquivo que estava
  aberto (absoluto); `--base`, a impressão dele naquele momento. Responde o
  item, como no `listar`. O dono do rascunho é o processo que chamou o motor
  (a janela).
- `rascunho listar`: o mais recente primeiro.

  ```json
  {"itens": [{"id": "r-…", "origem": "/textos/notas.md", "nome": "notas.md",
              "guardado_em_s": 1795500000, "tamanho_bytes": 49,
              "trecho": "a primeira linha com texto", "vivo": false}]}
  ```

  `vivo: true` quer dizer que a janela dona ainda está aberta: esse rascunho
  não é oferecido a outra janela. Um arquivo estragado na pasta fica de fora
  da lista (e fica no disco).
- `rascunho ler <id>`:
  `{"id", "origem", "nome", "base_sha256", "guardado_em_s", "dono_pid", "dono_nome", "texto"}`.
- `rascunho apagar <id>`: `{"ok": true, "id": "…"}`. Apagar o que já não
  existe não é erro.

O motor nunca grava um rascunho por cima de um arquivo: recuperar é a janela
ler o rascunho e pôr o texto na tela, como mudança não salva.

## `mkescrita-motor vigiar [--esperado <sha256|ausente>] [--intervalo-ms N] <arquivo>`

Fica rodando e solta uma linha quando o conteúdo do arquivo não é mais o
conhecido (verificação periódica, padrão 1000 ms; só relê o arquivo quando o
tamanho ou a data mudam):

```json
{"evento": "mudou", "impressao": {…}}
{"evento": "sumiu", "impressao": null}
```

Com `--esperado`, a primeira olhada já compara com o que a janela conhece; sem
ele, o ponto de partida é o que estiver no disco. O vigia não sabe quem gravou:
a janela compara a impressão do evento com a que ela tem e ignora o que ela
mesma gravou. Para sozinho quando a janela que o abriu morre.

## `mkescrita-motor html [--pagina] [--titulo <título>] [--bytes N]`

Markdown pelo stdin, `{"html": "…"}` na saída. O conversor é próprio e cobre o
subconjunto que importa ao escrever: títulos `#`, parágrafos, quebra forçada
(dois espaços ou `\` no fim da linha), `**negrito**`, `*itálico*`,
`~~tachado~~`, `` `código` ``, blocos de código cercados (a linguagem vai em
`class="language-…"`), citações, listas (ordenadas, não ordenadas, aninhadas e
de tarefas: `- [ ]` vira `<li class="tarefa">☐ …`, `- [x]` vira
`<li class="tarefa feita">☑ …`), links, imagens, linhas horizontais e tabelas
simples (com o alinhamento em `align`).

- **Tudo que vem do texto sai escapado**: HTML digitado (`<script>`) aparece
  como texto, nunca como marcação.
- Um link ou imagem só leva o endereço se ele for relativo, `http`, `https`
  ou (em link) `mailto`. `javascript:`, `data:`, `file:` e o resto: fica só o
  texto, sem link.
- `--pagina` devolve um documento completo, com um CSS sóbrio embutido (nada
  vem de fora: abre sem rede e imprime igual). O `<title>` é `--titulo` ou o
  primeiro título do texto.

## `mkescrita-motor contar [--bytes N]` e `mkescrita-motor estrutura [--bytes N]`

```json
{"palavras": 91, "caracteres": 613, "caracteres_sem_espacos": 471,
 "linhas": 29, "paragrafos": 11, "leitura_s": 28}
```

Palavra é um trecho entre espaços com ao menos uma letra ou algarismo (`#` e
`**` soltos não contam). `leitura_s` é a 200 palavras por minuto.

```json
{"titulos": [{"nivel": 1, "texto": "Caderno de viagem", "linha": 1}]}
```

Os títulos `#`, na ordem, com o texto sem marcação e a linha contada do 1. O
que está dentro de um bloco de código não é título.

## `mkescrita-motor exportar html [--forcar] [--titulo <título>] [--bytes N] <saída>`

Grava a página do `html --pagina` em `<saída>` (atômico). **Nunca passa por
cima** de um arquivo que já existe sem `--forcar`: recusa com saída 2. Responde
como o `salvar`.

## `mkescrita-motor preferencias <ler|gravar>`

```json
{"fonte": "", "zoom": 100, "previa": false}
```

`fonte` é a família escolhida (`""` = a monoespaçada do tema), `zoom` o tamanho
do texto em porcento (sempre entre 50 e 300) e `previa` se a pré-visualização
fica aberta. `gravar` recebe pelo stdin um objeto só com o que mudou, junta ao
que já existe e responde o resultado. Arquivo ausente ou estragado: os padrões
(preferência ruim nunca impede ninguém de escrever).

## O que a janela faz com isso

- **Salvamento automático**: depois que o documento tem nome, grava pouco
  depois de a pessoa parar de digitar, sempre com `--esperado`. Documento sem
  nome nunca é salvo sozinho (só vira rascunho) e o app nunca escolhe um nome.
  Em conflito, o salvamento automático para até a pessoa decidir.
- **Rascunho**: a cada pausa, o texto que não está no disco vai para
  `rascunho guardar`; quando o salvar dá certo (ou a pessoa descarta), o
  rascunho é apagado. Ao abrir, um rascunho que não é de janela viva é
  oferecido; recuperado, ele entra como mudança não salva e **o salvamento
  automático não o grava**: só a pessoa, com Ctrl+S (e aí o `--esperado` é a
  `base_sha256` do rascunho: se o arquivo mudou desde então, é conflito).
- **Mudou por fora** (`vigiar`): sem mudança na janela, o texto é recarregado;
  com mudança, é conflito.
- **Fechar**: sem mudança, fecha; documento com nome e sem conflito, grava e
  fecha; o resto pergunta (Salvar / Descartar / Cancelar).
- O título da janela nasce `MkEscrita` e, depois de assentada, passa a dizer o
  documento (`● notas.md — MkEscrita`).
- A janela não roda nada além do motor (e do próprio lançador, para abrir
  outra janela).
