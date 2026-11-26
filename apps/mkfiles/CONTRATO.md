# MkFiles: o contrato entre o motor e a janela

O MkFiles é o gerenciador de arquivos do Mk. Tem duas metades:

- **o motor** (`src/`, em Go, só a biblioteca padrão; um binário:
  `bin/mkfiles-motor`), que lista, descobre o tipo, copia, move, apaga, cuida
  da lixeira, busca, faz miniaturas, vigia pastas e discos. É todo testado em
  pastas temporárias, com bancos e comandos de mentira;
- **a janela** (`qml/`, Quickshell, tokens do Maré), que mostra o que o motor
  diz e pede o que a pessoa fez. A lógica que ela faz sozinha (ordenar,
  filtrar, selecionar, migalhas, textos) mora nos `.js` de `qml/`, testados
  no node.

Este arquivo é o contrato entre as duas. Muda aqui primeiro, depois nos dois
lados. O título da janela é **`MkFiles`** (janela de trabalho: não entra na
regra de flutuar).

## Regras gerais

- **O motor nunca age sobre o caminho vazio, sobre `/`, sobre a pasta pessoal
  inteira nem sobre uma pasta acima dela** (apagar, mandar para a lixeira,
  mover, renomear, mudar permissão com "aplicar ao conteúdo"). O mesmo vale
  para a pasta dos remotos (gvfs), o que a contém e a raiz de cada remoto
  montado. A recusa é `{"erro": "…", "recusado": true}`.
- Todo caminho recebido é **absoluto e limpo** (igual ao seu
  `filepath.Clean`), sem byte nulo; um caminho relativo, com `..` ou `//`
  é recusado, não consertado. Um **nome novo** não é vazio, não tem `/` nem
  byte nulo, não é `.` nem `..` e tem até 255 bytes.
- **O motor nunca passa texto por shell.** Os comandos de fora que ele roda
  são só: o miniaturizador do sistema (argv montado do `.thumbnailer`), o
  `lsblk`, o `udisksctl`, o `wl-copy`, o `wl-paste`, os compactadores (`xz`,
  `zstd`, `7z`, `unrar`, `bsdtar`) e o `gio` (remotos). Os apps que abrem
  arquivos e o terminal **não** são rodados pelo motor: ele devolve o argv e a
  janela executa com `Quickshell.execDetached`, inteiro, sem shell.
- Ao apagar uma pasta, um link simbólico é apagado como link: o motor nunca
  segue um link para fora.
- O que o motor não soube é `null`, nunca `""`. Datas são segundos desde 1970
  (`…_s` ou `mtime`). Tamanhos são bytes.
- Variáveis de ambiente (é por elas que os testes trocam o sistema):

  | Variável | O que troca | Padrão |
  |---|---|---|
  | `HOME`, `XDG_CONFIG_HOME`, `XDG_DATA_HOME`, `XDG_DATA_DIRS`, `XDG_CONFIG_DIRS`, `XDG_CACHE_HOME`, `XDG_STATE_HOME` | as pastas da pessoa | as do XDG |
  | `MKFILES_MIME` | a raiz do banco shared-mime-info | `/usr/share/mime` |
  | `MKFILES_ETC` | onde estão `passwd` e `group` | `/etc` |
  | `MKFILES_MOUNTINFO` | a tabela de montagens | `/proc/self/mountinfo` |
  | `MKFILES_MINIATURIZADORES` | a pasta dos `.thumbnailer` | `/usr/share/thumbnailers` |
  | `MKFILES_LSBLK`, `MKFILES_UDISKSCTL`, `MKFILES_WLCOPY`, `MKFILES_WLPASTE` | os comandos | `lsblk`, `udisksctl`, `wl-copy`, `wl-paste` |
  | `MKFILES_TERMINAL` | o terminal (`ghostty`, `kitty`, `foot`, `alacritty`, `wezterm`) | o dos Ajustes do Mk (`apps.terminal`), ou o primeiro que houver |
  | `MKFILES_LOTE` | itens por linha no `listar` e no `buscar` | `500` |
  | `MKFILES_DISCOS_MS` | de quanto em quanto tempo os discos (e os remotos montados) são reconsultados | `2000` |
  | `MKFILES_GIO` | o comando do gvfs que monta os remotos | `gio` |
  | `MKFILES_GVFS` | a pasta em que o gvfs mostra os remotos montados | `$XDG_RUNTIME_DIR/gvfs` |
  | `MKFILES_SSH_DIR` | a pasta do ssh (só o `config` e os `Include` são lidos) | `~/.ssh` |
  | `MKFILES_REMOTO_MS` | quanto o `gio mount` pode ficar calado antes de ser morto | `30000` |
  | `MKFILES_RELER_MS` | de quanto em quanto tempo uma pasta remota vigiada é relida | `5000` |
  | `MKFILES_XZ`, `MKFILES_ZSTD`, `MKFILES_7Z`, `MKFILES_UNRAR`, `MKFILES_BSDTAR` | os comandos dos compactados | `xz`, `zstd`, `7z` (ou `7zz`, `7za`), `unrar`, `bsdtar` |
  | `MKFILES_EXTRAIR_MAX`, `MKFILES_EXTRAIR_TAXA` | as travas contra bomba ao extrair: bytes no total e vezes o tamanho do arquivo | `21474836480` (20 GiB), `200` |

## Como se fala com o motor

`mkfiles-motor servir` fica de pé: lê **um pedido por linha** no stdin e
escreve **uma linha JSON por resposta ou evento** no stdout.

- Pedido: `{"id": N, "op": "listar", …}`. O `id` é um número ou um texto.
- Resposta: `{"id": N, …}`. Erro: `{"id": N, "erro": "frase em português"}`
  (um erro de formulário traz `"campo"`; uma recusa de segurança,
  `"recusado": true`). Linha que não é JSON: `{"id": null, "erro": "pedido
  que não é JSON"}`; op desconhecido: `não conheço o pedido «x»`.
- **Resposta em fluxo** (`listar`, `buscar`, `propriedades`, `lixeira-listar`):
  várias linhas com o mesmo `id`; todas têm `"fim"`, e só a última tem
  `"fim": true`. `{"op": "cancelar", "pedido": N}` para um fluxo no meio: a
  última linha dele sai com `"fim": true, "cancelada": true`. O `cancelar`
  responde `{"cancelado": true|false}`.
- **Evento**: `{"evento": "nome", …}`, sem `id`. Sai sozinho.
- O servidor continua de pé depois de qualquer erro, e termina (saída 0)
  quando o stdin fecha; as tarefas em andamento são canceladas (o arquivo
  parcial é removido).

Os mesmos pedidos existem como **subcomando de uma chamada só**, para teste e
depuração: `mkfiles-motor <op> [caminho…] [--campo=valor…]`. Os caminhos
viram `"caminhos"` (e o primeiro também `"caminho"`); cada `--campo=valor`
vira um campo (o valor é lido como JSON; se não for JSON, é texto). As linhas
saem sem `id`. A chamada espera os fluxos e as tarefas acabarem. Saída: `0`
certo, `1` erro, `2` recusado. Sem subcomando: `{"erro": "uso: mkfiles-motor
<op> …"}` e saída 1.

`{"op": "sobre"}` responde `{"versao": 1, "ops": ["listar", …], "extensoes":
{"lote": true, "compactados": true, "previa": true, "remotos": true}, "gvfs":
"/run/user/1000/gvfs"}`. `gvfs` é a pasta dos remotos montados (`null` se não
há onde).
As `extensoes` dizem o que este motor já faz (veja o fim): a janela só
mostra no menu o que estiver `true`.

## O item

Tudo que é arquivo aparece como um **item**:

```json
{"nome": "foto.jpg", "caminho": "/home/ana/Imagens/foto.jpg",
 "tipo": "arquivo", "tamanho": 20480, "mtime": 1700000000,
 "modo": 420, "permissoes": "rw-r--r--", "dono": "ana", "grupo": "ana",
 "oculto": false, "mime": "image/jpeg",
 "icone": "image-jpeg", "icone_reserva": "image-x-generic",
 "alvo": null, "quebrado": false, "alvo_tipo": null}
```

- `tipo`: `pasta`, `arquivo`, `link` ou `especial` (soquete, fifo,
  dispositivo). Num `link`, `alvo` é o texto do link, `quebrado` diz se ele
  não leva a nada, e `alvo_tipo` (`pasta`, `arquivo`, `especial`; `null` se
  quebrado) diz o que há do outro lado; `tamanho`, `mime` e os ícones são os
  do alvo (de um link quebrado: tamanho 0, `inode/symlink`).
- `modo` são os 12 bits de permissão (número; `420` = `0644`); `permissoes` é
  o `rwx` de 9 letras (com `s`/`t` quando há setuid/setgid/sticky).
- `dono` e `grupo` são nomes (o número, em texto, se não há nome).
- `oculto`: o nome começa com `.` ou está no arquivo `.hidden` da pasta (um
  nome por linha).
- `mime`: pelo nome, com o banco shared-mime-info (`globs2`: o peso maior
  ganha; no empate, o padrão mais longo; literal, depois `*.ext`, com e sem
  diferenciar maiúsculas). Sem padrão que case, **um farejo de conteúdo**
  (lê até 512 bytes): PNG, JPEG, GIF, PDF, ZIP, gzip, ELF
  (`application/x-executable`), `#!` (`application/x-shellscript`), texto
  (`text/plain`) ou `application/octet-stream`. Arquivo vazio:
  `application/x-zerosize`. Pasta: `inode/directory`. Especial:
  `inode/socket`, `inode/fifo`, `inode/chardevice`, `inode/blockdevice`. No
  `listar`, o farejo só acontece nos arquivos sem extensão.
- `icone` é o nome do ícone do tema: `tipo/subtipo` → `tipo-subtipo`;
  `icone_reserva` é o genérico do banco (`generic-icons`), ou
  `tipo-x-generic`. Pasta: `folder`/`folder`. A janela resolve com
  `Quickshell.iconPath(icone, icone_reserva)`.

## Listar e vigiar

### `listar` (fluxo)

`{"op": "listar", "caminho": "/pasta"}` → linhas `{"itens": [item…], "fim":
false}` com até 500 itens cada, e a última `{"itens": […], "fim": true,
"total": N, "gravavel": true|false}`. A ordem é a do disco: **ordenar e
filtrar são da janela**. Erro: `{"erro": "…", "motivo":
"nao-existe"|"sem-permissao"|"nao-e-pasta"|"tempo"|"conexao"|"erro"}`
(`tempo` e `conexao` são das pastas remotas: veja "Remotos"). Se a leitura parou no
meio, a última linha traz `"aviso"` com a frase.

### `item`

`{"op": "item", "caminho": "/x"}` → `{"item": {…}}` (com o farejo de conteúdo
sempre). É o que a janela usa para `mkfiles <arquivo>` e para completar o
caminho.

### `vigiar` / `desvigiar`

`{"op": "vigiar", "caminho": "/pasta"}` → `{"vigiando": true}`. O motor conta
as referências (duas abas na mesma pasta: dois `vigiar`, dois `desvigiar`).
Vigia por inotify. O evento, no máximo um a cada 150 ms por pasta:

```json
{"evento": "pasta", "caminho": "/pasta", "mudou": [item…], "saiu": ["nome"…],
 "sumiu": false, "recarregar": false}
```

`mudou` traz o item novo de cada nome criado, modificado ou que chegou por
renomeação; `saiu`, os nomes que não existem mais. `sumiu: true`: a própria
pasta foi apagada, movida ou desmontada. `recarregar: true` (lote com mais de
2000 nomes, fila do inotify estourada, ou o `.hidden` da pasta mudou): a
janela lista de novo. `desvigiar` responde `{"vigiando": true|false}` (se
ainda há outra referência).

## Abrir

### `apps`

`{"op": "apps", "mime": "image/png", "todos": false}` → `{"mime": …,
"padrao": "org.app.desktop"|null, "apps": [{"id": "org.app.desktop", "nome":
"Visor", "icone": "nome-ou-caminho"|null, "terminal": false, "padrao": true}…]}`.
Sem `todos`: os apps associados ao MIME (pelo `mimeapps.list`, depois os que o
declaram em `MimeType`), o padrão primeiro. Com `todos`: todo `.desktop` de
aplicativo visível, por nome. Ordem de leitura do padrão XDG:
`$XDG_CONFIG_HOME/mimeapps.list`, `$XDG_CONFIG_DIRS/mimeapps.list`,
`$XDG_DATA_HOME/applications/mimeapps.list`,
`$XDG_DATA_DIRS/applications/mimeapps.list`; `[Default Applications]`,
`[Added Associations]`, `[Removed Associations]`. Ficam de fora: `NoDisplay`,
`Hidden`, sem `Exec`.

### `definir-padrao`

`{"op": "definir-padrao", "mime": "image/png", "app": "org.app.desktop"}` →
`{"padrao": "org.app.desktop"}`. Escreve **só** em
`$XDG_CONFIG_HOME/mimeapps.list`, com escrita atômica, preservando todas as
outras linhas.

### `abrir`

`{"op": "abrir", "caminhos": ["/a.png", "/b.txt"], "app": null}` →

```json
{"lancar": [{"app": "org.app.desktop", "nome": "Visor", "argv": ["visor", "/a.png"]}],
 "sem_app": [{"caminho": "/b.txt", "mime": "text/plain"}]}
```

Os arquivos são agrupados pelo app (o padrão de cada MIME, ou o `app` dado).
`%f`/`%u` dão um argv por arquivo; `%F`/`%U`, um argv com todos (`%u`/`%U`
recebem `file://…`); `%i`, `%c` e `%k` são expandidos ou somem; `%%` é `%`;
sem código de campo, o arquivo vai no fim. `Terminal=true` vira a linha do
terminal do Mk (sem terminal instalado, o argv vai sem ele e o item traz
`"aviso"`). Um arquivo sem app vai em `sem_app`: a janela abre a folha
"abrir com". Pastas não passam por aqui (a janela navega).

### `terminal`

`{"op": "terminal", "caminho": "/pasta"}` → `{"argv": […]}` (o terminal do Mk
já na pasta) ou `{"erro": "não achei um terminal…"}`.

### `modelos`

`{"op": "modelos"}` → `{"pasta": "/home/ana/Modelos"|null, "modelos":
[item…]}`: os arquivos da pasta de modelos (`XDG_TEMPLATES_DIR` do
`user-dirs.dirs`, senão `~/Templates`), sem os ocultos.

## Operações (tarefas)

Cada pedido abaixo cria uma **tarefa** e responde na hora `{"tarefa": T}` (ou
`{"erro": …}` se o pedido é inválido). As tarefas rodam **uma por vez, em
fila**.

| Pedido | Campos | O que faz |
|---|---|---|
| `copiar` | `origens`, `destino` | copia para a pasta `destino`; preserva modo e mtime; pasta é recursiva; link é copiado como link. Com a origem já na pasta de destino, cria `nome (2).ext` sem perguntar |
| `mover` | `origens`, `destino` | `rename`; entre sistemas de arquivos, copia, confere (tamanho de cada arquivo) e só então apaga a origem |
| `ligar` | `origens`, `destino` | cria links simbólicos (alvo absoluto) |
| `criar-pasta` | `destino`, `nome` | |
| `criar-arquivo` | `destino`, `nome`, `modelo` (caminho ou `null`) | vazio, ou cópia do modelo |
| `renomear` | `caminho`, `nome` | na mesma pasta; nome ocupado é erro (`campo: "nome"`), não conflito |
| `renomear-lote` | `pares` (`[["/pasta/de.txt", "nome novo.txt"]…]`) | renomeia vários de uma vez, tudo ou nada; veja "Renomear em lote" |
| `lixeira` | `origens` | manda para a lixeira |
| `apagar` | `origens` | apaga de vez |
| `restaurar` | `itens` (caminhos em `files/` da lixeira) | devolve ao lugar de origem; lugar ocupado é conflito |
| `lixeira-apagar` | `itens` | apaga de vez itens da lixeira |
| `lixeira-esvaziar` | | esvazia todas as lixeiras da pessoa |
| `desfazer`, `refazer` | | veja "Desfazer" |

Erros na hora (sem tarefa): copiar ou mover uma pasta para dentro dela mesma
(`não dá para copiar uma pasta para dentro dela mesma`; os caminhos são
comparados já sem links), mover para onde já está, destino que não é pasta,
`criar-pasta`/`criar-arquivo` com nome ocupado (`campo: "nome"`), e as recusas
de segurança (só estas com `"recusado": true`).

Na chamada de um subcomando só, os caminhos soltos valem como `origens` (ou
`itens`), e não há quem responda a um conflito: o item é pulado, entra em
`erros` e a saída é 1.

O evento de andamento (a cada mudança de estado, e no máximo um a cada 100 ms
enquanto roda):

```json
{"evento": "tarefa", "tarefa": 3, "acao": "copiar", "estado": "rodando",
 "titulo": "Copiando 3 itens para Documentos",
 "itens": 1, "itens_total": 3, "bytes": 1048576, "bytes_total": 9000000,
 "atual": "/home/ana/a.iso", "velocidade": 52428800, "destino": "/home/ana/Documentos",
 "criados": [], "erros": [], "sem_lixeira": [], "desfazivel": false}
```

- `estado`: `fila`, `rodando`, `pausada`, `conflito`, `feita`, `cancelada`,
  `erro` (acabou com algum erro). Os três últimos são finais.
- `criados`: os caminhos de topo que a tarefa criou (a janela os seleciona).
- `erros`: `[{"caminho": …, "erro": "frase"}]`; a tarefa segue com os outros.
- `sem_lixeira`: os caminhos que não têm lixeira possível (outro disco sem
  `.Trash-$uid` gravável). Nada foi feito com eles: a janela pergunta se apaga
  de vez e manda `apagar`.
- `velocidade`: bytes por segundo (média do último segundo).

`{"op": "tarefa-pausar", "tarefa": T, "pausar": true|false}` e
`{"op": "tarefa-cancelar", "tarefa": T}` → `{"tarefa": T}`. Cancelar no meio de um
arquivo **remove o arquivo parcial**; o que já foi copiado inteiro fica.
`{"op": "tarefas"}` → `{"tarefas": [o último evento de cada uma não final]}`.

### Pontos de montagem

Apagar nunca atravessa um ponto de montagem. Ao apagar uma pasta, uma
subpasta que está em outra unidade (o `st_dev` é outro que o da pasta de
partida) ou que consta da tabela de montagens não é aberta: vira o erro `há
outro disco montado em <ponto>: desmonte antes`, a tarefa acaba em `erro`, o
disco fica intacto e o caminho até ele também (o resto da pasta sai). O
próprio ponto de montagem também não é apagado. Vale o mesmo para a
`lixeira` (a pasta que tem um disco montado dentro não é movida: é erro, não
`sem_lixeira`) e para o `mover` entre sistemas de arquivos (nem começa a
copiar: a origem fica inteira).

### Renomear em lote

`{"op": "renomear-lote", "pares": [["/pasta/a.txt", "b.txt"], ["/pasta/b.txt",
"a.txt"]]}` → `{"tarefa": T}`. Cada par é o caminho de hoje e o **nome** novo
(na mesma pasta); a conta dos nomes é da janela. O motor confere o lote
inteiro antes de tocar em qualquer coisa: caminho e nome válidos, as recusas
de segurança, todo item existe, nenhum item duas vezes. Par cujo nome não
muda é ignorado (`nenhum nome muda` se todos). Havendo conflito, nada vira
tarefa:

```json
{"erro": "há 2 nome(s) em conflito: nada foi renomeado",
 "conflitos": [{"de": "/pasta/a.txt", "para": "x.txt", "motivo": "repetido"}]}
```

`motivo`: `invalido` (o nome não serve), `repetido` (dois itens do lote
chegam ao mesmo nome) ou `existe` (o nome já é de um item da pasta que não
sai do lugar neste lote).

A tarefa é **tudo ou nada**: as cadeias (a→b, b→c) são feitas na ordem que
não pisa em ninguém, os ciclos (a→b, b→a) são abertos com um nome temporário
oculto na mesma pasta (`.mkfiles-…`), e se uma renomeação falha, um destino
aparece ocupado na hora ou a tarefa é cancelada, as já feitas são desfeitas
de trás para a frente (a tarefa acaba em `erro`, com tudo como estava).
`criados` traz os caminhos novos. Entra no histórico como **um passo só**
(`Renomear 3 itens`): desfazer devolve todos os nomes, refazer os põe de novo.

### Conflito

Quando o destino já existe, a tarefa para (`estado: "conflito"`) e pergunta:

```json
{"evento": "conflito", "tarefa": 3, "conflito": 1, "acao": "copiar",
 "origem": {item}, "destino": {item}, "sugestao": "foto (2).jpg", "restam": 4}
```

A janela responde `{"op": "responder", "tarefa": 3, "conflito": 1, "resposta":
"substituir", "nome": null, "todos": false}` → `{"tarefa": 3}`.

| `resposta` | O que faz |
|---|---|
| `substituir` | o destino dá lugar à origem. Pasta sobre pasta **mescla** (os conflitos de dentro perguntam de novo); arquivo sobre pasta, ou o contrário, troca |
| `pular` | deixa como está |
| `manter` | mantém os dois: grava como a `sugestao` (`nome (2).ext`, `nome (3).ext`…; em `.tar.gz` e parentes o número vai antes do `.tar`) |
| `renomear` | grava como `nome` (ocupado também: pergunta de novo) |
| `cancelar` | cancela a tarefa inteira |

`todos: true` vale a resposta para os próximos conflitos da tarefa (menos
`renomear`). Substituir grava num arquivo temporário ao lado e troca no fim:
o destino antigo só some quando o novo está inteiro.

### Desfazer

O histórico é da sessão do motor (some com a janela). `desfazer` e `refazer`
criam uma tarefa (`{"tarefa": T, "descricao": "Mover 3 itens"}`) ou respondem
`{"erro": "nada para desfazer"}`. A cada mudança sai:

```json
{"evento": "historico", "desfazer": "Mover 3 itens"|null, "refazer": null}
```

O último `historico` que saiu é sempre o estado de agora (os avisos nunca saem
fora de ordem).

| Tarefa | Desfazer |
|---|---|
| `renomear` | renomeia de volta |
| `renomear-lote` | devolve todos os nomes de uma vez (tudo ou nada, como a ida) |
| `mover` | move de volta (só o que foi movido sem substituir nada) |
| `criar-pasta`, `criar-arquivo`, `ligar` | apaga o que criou (pasta só se continua vazia; arquivo só se não mudou) |
| `lixeira` | restaura |
| `copiar` | apaga o que copiou, **só o que não mudou desde então** (mesmo tamanho e mtime) |

`apagar`, `restaurar`, `lixeira-apagar`, `lixeira-esvaziar` e uma tarefa que
substituiu algum arquivo não se desfazem (`desfazivel: false`). Um desfazer
que acaba em erro não vira refazer.

## Lixeira

Especificação do freedesktop: `$XDG_DATA_HOME/Trash/{files,info}`, com
`info/<nome>.trashinfo` (`[Trash Info]`, `Path=` com o caminho codificado como
URI, `DeletionDate=AAAA-MM-DDTHH:MM:SS` na hora local). Um arquivo de outro
sistema de arquivos (pelo ponto de montagem mais longo do `mountinfo`) vai
para `<ponto>/.Trash-<uid>/{files,info}`, com o `Path=` relativo ao ponto; se
não dá para criar (ou o mover daria em outro disco), vai em `sem_lixeira`: a
lixeira nunca copia entre discos. O `.trashinfo` é criado antes (`O_EXCL`;
nome ocupado ganha ` (2)`), depois o arquivo é movido. `lixeira-esvaziar`
leva também o arquivo sem ficha e a ficha sem arquivo.

`{"op": "lixeira-listar"}` (fluxo) → `{"itens": [item + {"nome_original":
"a.txt", "origem": "/de/onde/veio/a.txt", "apagado_s": 1700000000,
"lixeira": "/home/ana/.local/share/Trash"}…], "fim": …}`. O `caminho` de cada
um é o do arquivo dentro de `files/`: é ele que vai em `restaurar` e
`lixeira-apagar`. `{"op": "lixeira-estado"}` → `{"itens": N}`; o evento
`{"evento": "lixeira", "itens": N}` sai quando uma tarefa muda a lixeira.

## Busca

`buscar` (fluxo):

```json
{"op": "buscar", "caminho": "/pasta", "nome": "relat*", "tipo": "qualquer",
 "tamanho_min": null, "tamanho_max": null, "depois_s": null, "antes_s": null,
 "conteudo": null, "ocultos": false, "outros_discos": false}
```

→ `{"itens": [item…], "fim": false}` (a cada 100 achados ou 200 ms) e
`{"itens": […], "fim": true, "total": N, "vistos": M}`.

- `nome`: sem diferenciar maiúsculas nem acentos. Com `*` ou `?` é um padrão
  do nome inteiro; sem eles, um trecho do nome. Vazio: qualquer nome.
- `tipo`: `qualquer`, `pasta`, `arquivo` (só arquivo comum), `imagem`,
  `video`, `audio`, `texto`, `documento` (PDF, texto e os de escritório). Um
  filtro de tamanho ou de conteúdo deixa as pastas de fora.
- `conteudo`: procura o texto (sem diferenciar maiúsculas nem acentos) dentro
  dos arquivos de texto de até 8 MiB; binário (byte nulo nos primeiros 8 KiB)
  é pulado.
- Não segue links. Não entra em outro sistema de arquivos (pelo `st_dev`) sem
  `outros_discos`. Não entra em pasta oculta nem lista oculto sem `ocultos`.

## Miniaturas

Especificação do freedesktop: `$XDG_CACHE_HOME/thumbnails/{normal,large,x-large}/<md5
do URI>.png` (128, 256 e 512 px), com os blocos `tEXt` `Thumb::URI` e
`Thumb::MTime`; uma miniatura vale se os dois batem com o arquivo. Falha
gravada em `thumbnails/fail/<quem>/<md5>.png` (de qualquer programa) é
respeitada: não tenta de novo enquanto a mtime não muda; o motor grava as
dele em `fail/mkfiles/`. Tipo sem miniaturizador dá `arquivo: null` sem anotar
falha. A orientação do EXIF de um JPEG é aplicada (só a tag `0x0112` é lida:
a foto de celular em pé sai em pé).

`{"op": "miniatura", "caminhos": […], "tamanho": "normal"|"large"|"x-large", "grupo":
"texto", "prioridade": 0}` → `{"aceitos": N}`. Para cada caminho sai um
evento:

```json
{"evento": "miniatura", "caminho": "/x/a.png", "tamanho": "normal",
 "arquivo": "/home/ana/.cache/thumbnails/normal/….png"|null}
```

PNG, JPEG e GIF o motor faz sozinho (imagem de até 64 MiB e 100 megapixels).
O resto, pelo `.thumbnailer` do sistema que declara o MIME (`Exec` com `%i`,
`%o`, `%s`, `%u`; grava num temporário e o motor põe os blocos `tEXt`), com
limite de 10 s e **dois** processos por vez. Fila: prioridade maior primeiro;
no empate, o pedido mais novo. `{"op": "miniatura-cancelar", "grupo":
"texto"}` → `{"cancelados": N}` tira da fila o que ainda não começou.

## Lugares, favoritos e discos

`{"op": "lugares"}` → `{"lugares": [{"id": "pessoal", "nome": "Pasta
pessoal", "caminho": "/home/ana", "icone": "user-home"}…], "favoritos":
[favorito…]}`. Os ids fixos: `pessoal`, `documentos`, `downloads`, `imagens`,
`musicas`, `videos` (do `user-dirs.dirs`; o que não existe ou é a própria
pasta pessoal não aparece), `lixeira` (`caminho: null`) e `raiz`.

Um favorito: `{"caminho": "/home/ana/Work"|null, "uri": "file:///home/ana/Work",
"nome": "Work", "existe": true, "remoto": false}`. O arquivo é
`$XDG_CONFIG_HOME/gtk-3.0/bookmarks` (uma linha `URI [nome]`). As linhas que
não são `file://` ficam como estão (`remoto: true`, `caminho: null`). Toda
escrita é atômica e preserva as outras linhas.

`favorito-por` (`caminho`, `nome` ou `null`, `posicao` ou `null`),
`favorito-tirar` (`uri`), `favorito-mover` (`de`, `para`: posições),
`favorito-renomear` (`uri`, `nome`) → `{"favoritos": […]}`. Só uma pasta que
existe vira favorito; a que já é não entra duas vezes.

`{"op": "discos"}` → `{"discos": [{"dispositivo": "/dev/sdb1", "nome":
"PENDRIVE", "rotulo": "PENDRIVE"|null, "sistema": "vfat", "tamanho":
16000000000, "montado": "/run/media/ana/PENDRIVE"|null, "livre": 1000|null,
"total": 16000|null, "removivel": true, "uuid": …|null}…]}`. Entram as
partições e discos com sistema de arquivos; ficam de fora o swap, os
contêineres (`crypto_LUKS`, `LVM2_member`, `linux_raid_member`), `squashfs`,
os `loop` e o que está montado em `/`, `/boot…`, `/efi…`, `/home`, `/var…`,
`/usr…`. Sem rótulo, o `nome` é `Volume de 16 GB`. O motor roda `lsblk --json
--bytes -o NAME,PATH,LABEL,FSTYPE,SIZE,MOUNTPOINTS,RM,HOTPLUG,TYPE,UUID`.

`montar`, `desmontar`, `ejetar` (`dispositivo`) → `{"discos": […]}` (o
`montar` traz também `"montado": "/onde/montou"`) ou `{"erro": "o que o
udisksctl disse"}`. Rodam
`udisksctl mount|unmount|power-off -b <dispositivo>` (o `ejetar` desmonta e
desliga o disco pai). O dispositivo tem de estar na lista do `lsblk`.

`{"op": "vigiar-discos", "ligado": true}` → `{"ligado": true}`. Ligado, o
motor relê o `mountinfo` e o `lsblk` a cada 2 s e solta `{"evento": "discos",
"discos": […]}` quando a lista muda (o `livre` sozinho não conta: para a
barra de uso a janela pede `discos` de novo, ao fim de uma tarefa).

`{"op": "espaco", "caminho": "/pasta"}` → `{"livre": N, "total": N}`.

## Propriedades e permissões

`propriedades` (fluxo): `{"op": "propriedades", "caminhos": […]}` → primeiro
`{"itens": [item…], "onde": "/pasta", "sou_dono": true, "grupos": ["ana",
"wheel"], "contagem": {"arquivos": 0, "pastas": 0, "tamanho": 0}, "fim":
false}`, depois `{"contagem": {…}, "fim": false}` a cada 200 ms enquanto conta
o conteúdo das pastas, e por fim `{"contagem": {…}, "fim": true}`. A contagem
não segue links.

`{"op": "permissoes", "caminhos": […], "modo": 420|null, "executavel":
true|false|null, "grupo": "nome"|null, "recursivo": false}` → `{"itens":
[item…], "erros": […]}`. `executavel` liga ou desliga o `x` de quem já tem o
`r`. Com `recursivo`, o modo vale para o conteúdo: nas pastas de dentro, quem
tem `r` ganha `x`; cada arquivo de dentro fica com o `x` que já tinha (o modo
de uma pasta não faz de cada arquivo um executável). Links não são seguidos
(um link pedido diretamente entra em `erros`). No `propriedades`, `onde` é
`null` quando os itens vêm de pastas diferentes.

## Preferências e estado

`{"op": "prefs"}` → `{"prefs": {…}, "estado": {…}}`. `{"op": "prefs-gravar",
"prefs": {só o que mudou}}` → `{"prefs": {tudo}}`. `{"op": "estado-gravar",
"estado": {só o que mudou}}` → `{"estado": {tudo}}`. Chave desconhecida ou
valor fora do lugar é ignorado; arquivo corrompido volta ao padrão.

`$XDG_CONFIG_HOME/mk/mkfiles/preferencias.json`:

| Chave | Valores | Padrão |
|---|---|---|
| `visao` | `icones`, `lista`, `compacta` | `icones` |
| `visao_por_pasta` | `{"/pasta": "lista"}` (até 500; a mais antiga sai; `null` esquece a pasta) | `{}` |
| `ordem` | `{"campo": "nome"\|"tamanho"\|"tipo"\|"mtime", "inversa": bool}` | nome, `false` |
| `ocultos` | bool | `false` |
| `icone_px` | 32 a 256 | `64` |
| `pastas_antes` | bool | `true` |
| `confirmar_lixeira` | bool | `false` |
| `clique` | `duplo`, `simples` | `duplo` |
| `miniaturas_remotas` | bool (miniaturas em pasta remota; quem aplica é a janela) | `true` |
| `miniatura_remota_mb` | 1 a 100 (em remoto, só arquivos até este tamanho ganham miniatura) | `5` |
| `colunas` | `{"tamanho": px, "tipo": px, "mtime": px, "permissoes": px}` (40 a 600) | `90, 150, 150, 100` |

`$XDG_STATE_HOME/mk/mkfiles/estado.json`:

| Chave | Valores | Padrão |
|---|---|---|
| `paineis` | `[{"abas": ["/pasta"…], "atual": 0}]` (1 ou 2; até 30 abas cada; a aba da lixeira é o texto `lixeira:`) | `[]` |
| `ativo` | 0 ou 1 | `0` |
| `duplo` | bool | `false` |
| `lateral_px` | 120 a 480 | `200` |

## Área de transferência

O QML não expõe tipos MIME arbitrários: o motor chama o `wl-copy` e o
`wl-paste`. A montagem e a leitura do texto são da janela (`.js`).

- `{"op": "area-por", "tipo": "text/uri-list"|"x-special/gnome-copied-files"|"text/plain",
  "texto": "…"}` → `{"posto": true}`. Roda `wl-copy --type <tipo>` com o texto
  no stdin. O `wl-copy` oferece **um tipo por vez**, então a janela põe
  **sempre** `x-special/gnome-copied-files` (`copy\n` ou `cut\n` + um URI por
  linha), para copiar e para recortar: é o único tipo que carrega o
  "recortar" (veja "Quem lê o quê"). `text/uri-list` e `text/plain` continuam
  aceitos (copiar caminho usa `text/plain`). O `wl-copy` fica vivo servindo a seleção: o motor espera até
  3 s por um erro e o deixa solto.
- `{"op": "area-ler"}` → `{"tipos": ["text/uri-list", …], "tipo":
  "x-special/gnome-copied-files"|"text/uri-list"|"image/png"|"text/plain"|null,
  "texto": "…"|null}`. `wl-paste --list-types`, e o texto do primeiro tipo
  que houver, nessa ordem de preferência (de `image/png` não vem texto). Até
  4 MiB. Um `text/plain` pode ser uma lista de caminhos ou de `file://…`, um
  por linha: a janela o trata como arquivos copiados se todos existirem.
- `{"op": "area-gravar", "destino": "/pasta", "nome": "Imagem colada.png",
  "tipo": "image/png"|"text/plain"}` → `{"item": {…}}`. Grava o que está na
  área de transferência num arquivo novo (`O_EXCL`: nome ocupado é erro com
  `campo: "nome"`).

### Quem lê o quê (dedução, não conferida em cada programa)

Pelo que se sabe do comportamento público de cada gerenciador, sem ler o
código de nenhum:

| Gerenciador | Lê ao colar | O que acontece com o que o MkFiles põe |
|---|---|---|
| Nautilus (GNOME), Nemo, Caja | `x-special/gnome-copied-files` | cola, e respeita o recortar |
| Thunar, PCManFM (GTK) | `x-special/gnome-copied-files` | cola, e respeita o recortar |
| Dolphin, PCManFM-Qt e os de Qt/KDE | `text/uri-list` (o recortar vai em `application/x-kde-cutselection`) | **não vê** o que o MkFiles copiou; no outro sentido funciona (o MkFiles lê `text/uri-list`, sempre como cópia) |
| Editores e terminais | `text/plain` | não recebem os caminhos de um copiar de arquivos; "copiar caminho" põe `text/plain` |

É a troca aceita enquanto a área de transferência for um `wl-copy`: um tipo
só, e o escolhido é o que os gerenciadores GTK entendem por inteiro.

## Prévia

`{"op": "previa", "caminho": "/x"}` → o que a prévia rápida (Espaço) mostra:

```json
{"item": {item}, "classe": "texto", "texto": "…"|null, "cortado": false,
 "binario": false, "itens": null, "total": null}
```

- `classe`: `imagem` (`image/*`), `audio`, `video`, `pdf`, `compactado`
  (zip, tar e os comprimidos, 7z, rar), `texto` (`text/*`, JSON, XML, scripts
  e o que o farejo diz ser texto), `pasta` ou `outro`. Um link vale pelo que
  há do outro lado; link quebrado, fifo, soquete e dispositivo são `outro` e
  **nunca são abertos**.
- `texto`: os primeiros 200 KiB, em UTF-8, cortados numa fronteira de
  caractere (`cortado: true` se havia mais). Arquivo com nome de texto mas
  com byte nulo ou que não é UTF-8: `binario: true` e `texto: null` (a janela
  diz "arquivo binário", com tamanho e tipo).
- `itens` e `total` (só na `pasta`): os primeiros 50 itens, na ordem do
  disco, e quantos há ao todo; `null` os dois se a pasta não abre. A contagem
  e o tamanho recursivos vêm do `propriedades`.
- Imagem, áudio e vídeo a janela mostra sozinha, do arquivo. PDF: a miniatura
  `x-large` (pedido `miniatura`). Compactado: a listagem vem do pedido dos
  compactados.
- Erro: `{"erro": …, "motivo": "nao-existe"|"sem-permissao"|"erro"}`.

## Compactados

Extensão `compactados`. O motor lê e grava sozinho **zip, tar, gzip** e lê
**bzip2** (biblioteca padrão). **xz, zstd, 7z e rar** passam por um comando de
fora, quando existe: `xz -dc`/`-c`, `zstd -dc`/`-c` (só a casca, em pipe; o
tar de dentro é lido pelo motor, com as travas dele), `7z` (ou `7zz`, `7za`)
para 7z, rar e zip com senha, e `unrar` ou `bsdtar` para rar sem o 7z. Sem a
ferramenta, o erro diz o pacote (`xz`, `zstd`, `7zip`, `unrar`). Sempre argv,
sem shell, com prazo, e parando quando a tarefa é cancelada.

Os formatos: `zip`, `tar`, `tar.gz`, `tar.bz2`, `tar.xz`, `tar.zst`, `gz`,
`bz2`, `xz`, `zst`, `7z`, `rar`. Quem decide é o começo do arquivo (os bytes
mágicos); o nome só diz se há um tar dentro de uma casca (`.tar.gz`, `.tgz`,
`.tbz2`, `.txz`, `.tzst`). Um nome que promete e bytes que não cumprem: erro
com `motivo: "formato"`.

Os erros destes pedidos trazem `"motivo"`: `senha` (o arquivo pede senha, ou
a senha não confere), `falta` (com `"pacote"`), `formato`, ou `erro`.

**A senha** vem no campo `"senha"` do pedido (ou `null`) e vai à ferramenta
pelo stdin: nunca na linha de comando, nunca no ambiente, nunca numa frase de
erro. Não pode ser vazia nem ter quebra de linha. Zip com senha sem o 7z:
`zip com senha precisa do 7zip` (`motivo: "falta"`, `pacote: "7zip"`).

### `formatos`

`{"op": "formatos"}` → `{"extrair": [{"formato": "zip", "disponivel": true,
"falta": null}, {"formato": "7z", "disponivel": false, "falta": "7zip"}…],
"compactar": [idem para zip, tar.gz, tar.zst, tar.xz, 7z]}`.

### `compactado-info`

`{"op": "compactado-info", "caminho": "/a.zip", "senha": null}` →
`{"formato": "zip", "disponivel": true, "falta": null, "cifrado": false,
"itens": 12|null, "tamanho": 34567|null, "taxa": 3.2|null, "suspeito":
false}`. `itens`, `tamanho` (bytes descompactados, pelo índice) e `taxa`
(tamanho ÷ tamanho do arquivo) só vêm de quem tem índice (zip, tar, 7z, rar):
num tar com casca são `null`, porque saber custaria descompactar tudo.
`suspeito`: o índice já diz que passa dos limites da bomba (abaixo). Num zip
cifrado sem o 7z, `disponivel` é `false` e `falta` é `7zip`.

### `compactado-listar` (fluxo)

`{"op": "compactado-listar", "caminho": "/a.zip", "dentro": "", "senha":
null}` → linhas `{"itens": [item…], "fim": false}` e a última com `"fim":
true, "formato": …, "cifrado": …`. Só os filhos diretos de `dentro` (uma
pasta de dentro; `""` é a raiz), em ordem de nome; uma pasta que o arquivo não
declara mas tem algo dentro aparece assim mesmo. Cada item tem os 16 campos do
item mais `"interno": "pasta/a.txt"`; o `caminho` é o do compactado + `/` +
`interno`; `dono` e `grupo` são os do arquivo compactado; `mime` e ícones
saem do nome; um link traz `alvo` (`quebrado: false`, `alvo_tipo: null`). Lê
do índice, sem extrair (um tar é percorrido uma vez e lembrado enquanto o
arquivo não muda). As entradas de nome inválido (abaixo) não aparecem.

### `compactado-abrir`

`{"op": "compactado-abrir", "caminho": "/a.zip", "interno": "docs/a.txt",
"senha": null}` → `{"caminho":
"/run/user/1000/mkfiles/sessao-…/…/a.txt"}`: extrai **só** aquele arquivo
para um temporário em `$XDG_RUNTIME_DIR/mkfiles` (pastas `0700`; sem
`XDG_RUNTIME_DIR`, em `$XDG_CACHE_HOME/mkfiles/abertos`). Até 2 GiB (`o
arquivo é grande demais para abrir de dentro do compactado: extraia antes`).
Os temporários da sessão são apagados quando o motor fecha. Com 7z, rar e zip
cifrado a ferramenta extrai tudo num temporário e só o pedido fica.

### `extrair` (tarefa)

`{"op": "extrair", "caminho": "/a.zip", "destino": "/pasta", "modo":
"aqui"|"pasta", "internos": null, "senha": null, "sem_limite": false}` →
`{"tarefa": T}`.

- `pasta` cria `destino/<nome do arquivo sem a extensão>` (ocupado: `nome
  (2)`, sem perguntar); `aqui` põe o conteúdo direto em `destino`, com o
  conflito das tarefas (pasta sobre pasta, com `substituir`, mescla; uma pasta
  nunca é trocada por um arquivo).
- `internos` (`["a/b.txt", "docs"]`): extrai só o pedido, cada um pelo último
  nome, direto em `destino` (é o copiar/arrastar de dentro para fora).
- Sempre extrai para uma pasta temporária irmã (`destino/.mkfiles-extrai-…`,
  `0700`) e só então move para o lugar. Cancelar ou falhar não deixa nada.
- **Nada sai da pasta de destino.** Fica de fora, com um aviso em `erros` (a
  tarefa acaba em `erro`, com o resto no lugar): nome com `..`, caminho
  absoluto, byte nulo ou barra invertida; link simbólico que aponta para fora
  (alvo absoluto ou que sobe além da pasta); hardlink para fora; dispositivo,
  fifo e soquete. O motor nunca escreve através de um link: cada pedaço do
  caminho tem de ser uma pasta de verdade, o arquivo nasce com `O_EXCL` e
  `O_NOFOLLOW`, e os links só são criados no fim. `setuid`, `setgid` e
  `sticky` são tirados; o dono sempre lê e escreve. O que o 7z/unrar extraiu
  passa pela mesma varredura antes de ir para o lugar.
- **Bomba**: a tarefa para, limpa e acaba em `erro` com a frase começando por
  `parece uma bomba de compressão` quando o que foi **escrito de verdade**
  passa de `MKFILES_EXTRAIR_MAX` bytes (padrão 20 GiB) ou de
  `MKFILES_EXTRAIR_TAXA` vezes o tamanho do arquivo (padrão 200; a taxa só
  conta acima de 100 MiB escritos). A janela pergunta e manda de novo com
  `"sem_limite": true`.
- Andamento: num zip, `itens_total` e `bytes_total` vêm do índice; nos outros
  ficam em 0 e só `itens`/`bytes` andam.
- Desfazer: apaga o que criou (o arquivo que não mudou, a pasta que ficou
  vazia); uma extração que substituiu algo não se desfaz.

### `compactar` (tarefa)

`{"op": "compactar", "origens": […], "destino": "/pasta", "nome": "x.zip",
"formato": "zip"|"tar.gz"|"tar.zst"|"tar.xz"|"7z", "nivel": null|0…9,
"senha": null}` → `{"tarefa": T}`. Cada origem entra pelo nome dela (dois
itens com o mesmo nome: erro). Links são guardados como links, nunca
seguidos; dispositivos e fifos ficam de fora, com aviso. Grava num temporário
ao lado (`.mkfiles-compacta-…`) e põe no lugar no fim, sem passar por cima de
nada; cancelar remove o parcial. Nome ocupado, nome inválido, formato
desconhecido, nível fora de 0–9 e senha em formato que não cifra são erro na
hora, com `"campo"`. `nivel: 0` num zip guarda sem comprimir. Senha: só `zip`
e `7z`, e os dois pelo 7z (`-p`, com a senha no stdin; o 7z cifra também os
nomes). Desfazer apaga o arquivo criado, se não mudou.

## Remotos

O MkFiles **não implementa protocolo nenhum**. Quem fala sftp, smb, ftp, dav e
os outros é o **gvfs**: o motor monta com o `gio` e daí em diante o remoto é
uma pasta comum dentro de `gvfs` (o campo do `sobre`), com um nome como
`sftp:host=servidor,user=ana` ou `smb-share:server=nas,share=docs`. Listar,
copiar, mover, apagar, buscar, miniaturas, propriedades e prévia funcionam ali
sem nada próprio de rede. Os comandos do `gio` que o motor roda são só: `gio
mount [-a] <uri>`, `gio mount -u <uri>` e `gio list -u network:///`.

O que muda numa pasta remota:

- **`vigiar`**: o inotify não vê o FUSE do gvfs. A pasta é relida a cada 5 s
  e, se os nomes, tamanhos ou datas mudaram, sai o evento `pasta` com
  `"recarregar": true` (ou `"sumiu": true` se ela deixou de existir: o remoto
  caiu ou foi desmontado). A janela relista também ao fim de cada tarefa.
- **`listar` e `item`** têm prazo de 20 s para a pasta responder: estourou,
  `{"erro": "servidor não respondeu a tempo", "motivo": "tempo"}`. Se a raiz
  do remoto sumiu ou o erro é de rede, `{"erro": "a conexão com servidor
  caiu", "motivo": "conexao"}`. Um `cancelar` solta um `listar` que espera.
- **Tarefas**: um erro de rede num caminho remoto entra em `erros` (ou na
  resposta do pedido) como `a conexão com <nome> caiu`.
- **Lixeira**: não há num remoto. `lixeira` devolve os caminhos em
  `sem_lixeira`; a janela pergunta e manda `apagar`.
- **Segurança**: a pasta do gvfs, o que a contém e a raiz de cada remoto
  montado não podem ser apagadas, movidas nem renomeadas (`"recusado":
  true`): um remoto se desmonta. Lá dentro, vale tudo o que vale numa pasta.

### `remoto-hosts`

`{"op": "remoto-hosts"}` → `{"hosts": [{"apelido": "casa", "host":
"192.168.0.10"|null, "usuario": "ana"|null, "porta": "2222"|null}…]}`: os
`Host` do `~/.ssh/config` e dos arquivos dos `Include` (relativos a `~/.ssh`,
com curingas), sem os apelidos com curinga. Conectar a um deles é montar
`sftp://apelido/`: o gvfs usa o ssh do sistema, então o config, a chave e o
agente valem. Nenhuma chave é lida pelo motor.

### `remoto-montar`

`{"op": "remoto-montar", "uri": "sftp://ana@servidor:2222/home/ana",
"anonimo": false}` → responde **quando acaba**:

```json
{"uri": "sftp://ana@servidor:2222/home/ana", "nome": "servidor",
 "raiz": "/run/user/1000/gvfs/sftp:host=servidor,port=2222,user=ana",
 "caminho": "/run/user/1000/gvfs/sftp:host=servidor,port=2222,user=ana/home/ana"}
```

ou `{"erro": "frase", "motivo":
"senha"|"recusado"|"tempo"|"falta"|"endereco"|"erro", "saida": "o que o gio
escreveu"|null}`. `caminho` é a pasta pedida dentro do remoto (a raiz, se ela
não existe lá). Já montado: responde o lugar sem chamar o `gio`.

- O endereço tem de começar por `sftp://`, `smb://`, `ftp://`, `ftps://`,
  `dav://`, `davs://`, `nfs://`, `afp://`, `mtp://` ou `gphoto2://`, ter
  servidor e não ter caractere de controle (`motivo: "endereco"`). **Um
  endereço com senha (`usuario:senha@`) é recusado.** No SMB o domínio vai
  como `smb://DOMINIO;usuario@servidor/compartilhamento`.
- O `gio mount` pergunta usuário, domínio e senha no terminal. O motor o roda
  com o stdin num cano e lê a saída sem esperar quebra de linha; cada pergunta
  vira um evento e a montagem espera a resposta:

  ```json
  {"evento": "pergunta-remota", "pedido": 12, "pergunta": 1, "tipo": "senha",
   "texto": "Password: ", "padrao": null, "de_novo": false}
  ```

  `pedido` é o `id` do `remoto-montar`. `tipo`: `usuario`, `senha`, `dominio`,
  `confirmar` (a identidade do servidor não pôde ser conferida: `texto` traz a
  explicação e as escolhas numeradas, e a resposta é o número) ou `outro`
  (uma pergunta que o motor não conhece: `texto` é o que o `gio` escreveu).
  `padrao` é o valor entre colchetes (`User [ana]:`). `de_novo: true`: a
  senha já foi pedida nesta montagem (a anterior não serviu).
- A janela responde `{"op": "remoto-responder", "pedido": 12, "pergunta": 1,
  "resposta": "texto"|null}` → `{"respondida": true}`. `null` desiste: o
  `gio` é morto e a montagem acaba com `motivo: "recusado"`. `{"op":
  "cancelar", "pedido": 12}` faz o mesmo a qualquer momento. Numa pergunta
  `confirmar` a janela manda `"yes"` ou `"no"`: se a pergunta veio com opções
  numeradas (`[1] …`, `[2] …`), o motor responde ao `gio` o número da opção
  que quer dizer sim ou não (um "não" sem opção que o diga desiste); sem
  opções numeradas, vai como veio.
- O `gio` calado por 30 s (sem acabar e sem pergunta; o tempo da pessoa
  pensando não conta) é morto: `motivo: "tempo"`, com a `saida`.
- **A senha só passa pelo stdin do `gio`.** Nunca vai para argumento,
  variável de ambiente, log, `erro`, `saida` (uma senha ecoada é trocada por
  `•••`), preferências nem estado. "Lembrar a senha" é do chaveiro do
  sistema, que o gvfs usa sozinho.
- **É dedução, não foi conferido com o `gio` de verdade**: o texto exato das
  perguntas (o motor reconhece por palavras, em inglês e português: `user`,
  `usuário`, `utilizador`, `login`; `password`, `passphrase`, `senha`,
  `palavra-passe`; `domain`, `domínio`; `choice`, `escolha`, `yes/no`), e que
  o `gio` aceite as respostas por um cano, sem terminal. Um resto de linha
  que para sem quebra por 0,4 s é tratado como pergunta `outro`.

### `remotos` e `remoto-desmontar`

`{"op": "remotos", "rede": false}` → `{"remotos": [{"uri":
"sftp://ana@servidor/"|null, "raiz": "/run/user/1000/gvfs/sftp:host=…",
"nome": "servidor", "protocolo": "sftp", "host": "servidor"|null}…], "rede":
null}`: as pastas do gvfs. Num protocolo que o motor não conhece, `uri` e
`host` são `null` e `protocolo` é o texto antes dos dois pontos; num
compartilhamento, `nome` é `docs em nas`. Celulares e câmeras (MTP, gphoto2)
que o gvfs monta sozinho aparecem aqui. Com `"rede": true`, `rede` é
`[{"uri": "smb://nas/", "nome": "nas"}…]`, o que o `gio list -u network:///`
mostrar em 4 s (lista vazia se não der; **dedução**).

Depois do primeiro `remotos`, o motor confere a pasta do gvfs a cada 2 s e
solta `{"evento": "remotos", "remotos": […]}` quando a lista muda (e sempre
que monta ou desmonta).

`{"op": "remoto-desmontar", "uri": "sftp://ana@servidor/"}` → `{"remotos":
[…]}` ou `{"erro": …}`. Só desmonta um `uri` que está na lista (`gio mount -u
<uri>`).

## O que a janela espera

- Uma janela por chamada do `mkfiles`. `MKFILES_ABRIR` traz os caminhos
  pedidos, um por linha: pasta abre numa aba; arquivo abre a pasta dele com
  ele selecionado.
- `MKFILES_MOTOR` troca o binário. `MKFILES_EXEMPLO=pasta` usa respostas
  gravadas (`tests/apps/mkfiles-exemplo/<op>.json`: a lista das linhas que o
  motor daria, respostas sem `id` e eventos) em vez do motor.
- A janela só manda pedidos que estão neste arquivo e só executa (com
  `Quickshell.execDetached`) um argv que o motor devolveu (`abrir`,
  `terminal`), ou o próprio `mkfiles` para uma janela nova.
- Ordenar, filtrar, selecionar e formatar são dela; apagar, mover e gravar,
  nunca.

## Pontos de extensão

Renomear em lote, compactados, prévia rápida e remotos existem: as quatro
`extensoes` do `sobre` são `true`.

No motor, cada área é um pacote de `src/internal/` que se liga à sessão
(`Ligar`), e as tarefas passam todas pelo mesmo trilho (fila, progresso,
conflito, desfazer): um trabalho novo é um `operar.Registrar` (é assim que os
compactados e o lote entram), e uma área nova é um pacote e uma linha em
`main.go`. Na janela, o menu de contexto é uma lista montada no `.js`.
