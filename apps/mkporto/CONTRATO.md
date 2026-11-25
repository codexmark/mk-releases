# MkPorto: o contrato entre o motor e a janela

O MkPorto é o gerente de conexões SSH do Mk: um **catálogo com estado** dos
hosts que a pessoa tem no `~/.ssh/config`, não um cliente. Conectar é sempre
abrir um terminal com `ssh <apelido>`; quem autentica é o ssh, com o agente ou
o chaveiro. Só SSH: máquinas virtuais ficam para depois.

Tem duas metades:

- **o motor** (`src/`, em Go, um binário só: `bin/mkporto-motor`), que lê o
  `~/.ssh/config`, pergunta ao ssh como ele vê cada host, sonda as portas,
  testa a entrada, grava o arquivo do app e diz que comando abrir. É todo
  testado com pastas `.ssh` de mentira, um `ssh` e um `ssh-keygen` de mentira
  e um listener TCP em `127.0.0.1`;
- **a janela** (`qml/`, Quickshell, com os tokens do Maré), que mostra o que o
  motor diz e pede o que a pessoa clicou. A lógica que a janela faz sozinha (a
  busca, a ordem, a validação do formulário ao digitar, os textos) mora em
  `qml/Porto.js`, testado no node.

Este arquivo é o contrato entre as duas. Muda aqui primeiro, depois nos dois
lados.

## Regras gerais

- O motor escreve JSON em UTF-8 no stdout, um objeto por linha. Um subcomando
  que falha escreve `{"erro": "frase em português"}` e sai com código
  diferente de 0. A janela mostra a frase. Um erro de validação traz também
  `"campo"` (o campo do formulário que está errado).
- O que o motor não soube, ou que o host não tem, é `null`, nunca `""`.
- **O MkPorto nunca guarda nem pede senha**, e nunca lê uma chave privada: da
  pasta `~/.ssh` ele só abre o `config`, os arquivos que o `config` inclui e
  os `*.pub`.
- **O motor nunca abre uma sessão interativa nem pede senha.** Conectar,
  enviar chave e gerar chave são do terminal: o motor só devolve o argv
  (`comando`). Sozinho, ele roda o `ssh` de três jeitos, todos sem conversa
  (nada é perguntado, nada é aceito em nome da pessoa):
  `ssh -G -- <apelido>` (só lê a configuração; `listar` e `sondar`),
  `ssh -o BatchMode=yes -o ConnectTimeout=4 [-p <porta>] -W <host>:<porta> --
  <salto>` (a sonda por salto) e `ssh -o BatchMode=yes -o ConnectTimeout=5 -o
  NumberOfPasswordPrompts=0 -- <apelido> true` (`entrar`). Com `BatchMode` o
  ssh não pede senha nem frase-senha e não aceita sozinho a chave de um host
  desconhecido. O outro comando externo é o `ssh-keygen -l -f <arquivo.pub>`,
  para a impressão digital. Cada um tem tempo limite e é morto ao estourar.
- **A janela não monta comando nenhum com texto da pessoa**: executa o argv
  que o `comando` devolveu, inteiro, sem shell.
- No próprio motor, o que vem depois de `--` é sempre apelido, nome ou
  comentário, nunca opção. Só um apelido seguro (`A-Z a-z 0-9 . _ -`, sem
  começar com `-`) chega à linha de comando do ssh.
- Variáveis de ambiente do motor (é por elas que os testes trocam o sistema):
  `MKPORTO_SSH_DIR` (a pasta do ssh; padrão `~/.ssh`), `MKPORTO_SSH` (o
  comando do ssh, o que vai no argv e o que o motor roda; padrão `ssh`), `MKPORTO_COPY_ID` (padrão
  `ssh-copy-id`), `MKPORTO_KEYGEN` (padrão `ssh-keygen`) e `MKPORTO_TERMINAL`
  (o terminal em que a conexão abre). O `MKPORTO_SSH_DIR` é dos testes: o ssh
  de verdade lê sempre o `~/.ssh/config` dele.
- Os caminhos que o motor mostra vêm com a home abreviada (`~/.ssh/config`).

## O arquivo do app

O MkPorto só escreve em **um** arquivo: `~/.ssh/config.d/mkporto.conf`
(permissão `600`; a pasta `config.d`, `700`). Ele é reescrito inteiro a cada
mudança, de forma atômica (arquivo temporário na mesma pasta e `rename`), e só
tem blocos `Host` com `HostName`, `User`, `Port`, `IdentityFile` e
`ProxyJump`. O que alguém escrever nele à mão além disso se perde na gravação
seguinte (o cabeçalho do arquivo avisa).

Para o ssh enxergar o arquivo, a primeira gravação garante a linha

```
Include config.d/mkporto.conf
```

no **topo** de `~/.ssh/config` (antes de qualquer `Host`, senão o Include
valeria só para aquele host). O resto do arquivo fica como estava, byte a
byte; se o `config` não existe, é criado com `600`; se é um atalho (symlink),
quem é reescrito é o arquivo para onde ele aponta. Se já há, antes do primeiro
`Host`/`Match`, um `Include` que cobre o arquivo do app (o próprio, ou um
padrão como `config.d/*`), o `config` não é tocado.

Tudo o mais no `~/.ssh/config` é da pessoa: o MkPorto lê, mostra, conecta, e
nunca edita nem remove.

## Como o `config` é lido

Um leitor próprio do formato diz **quais** apelidos existem, de que arquivo
vêm e de quem são (`origem`), e dá os valores quando o ssh não responde:

- palavras-chave sem diferenciar maiúsculas; `Chave valor` ou `Chave=valor`;
  valores entre aspas; linhas vazias e comentários (`#`) ignorados;
- `Host a b c` declara três apelidos; padrões com `*`, `?` ou `!` não são
  máquinas e não entram na lista, mas **valem** para completar os hosts que
  casam com eles (um `User` num `Host *.exemplo` aparece nos hosts dali), com
  a regra do ssh: o primeiro valor achado ganha;
- `Include` com glob, relativo a `~/.ssh` (ou absoluto, ou `~/…`), em qualquer
  profundidade (até 16), sem laço;
- `Match` encerra o bloco: o que vem até o próximo `Host` não vale para
  ninguém (o motor não avalia condições), e um `Include` ali dentro não é
  lido;
- `%h` no `HostName` vira o apelido. `ProxyJump none` é não ter salto.
- O `/etc/ssh/ssh_config` não é lido pelo leitor próprio.

Os **valores efetivos** (`host`, `usuario`, `porta`, `chave`, `salto`) vêm do
próprio ssh: `MKPORTO_SSH -G -- <apelido>`, que imprime a configuração já
resolvida (com `Match` e com o `/etc/ssh/ssh_config`), uma linha
`chave valor` por opção. O motor lê `hostname`, `user`, `port`, o primeiro
`identityfile` e `proxyjump` (`none` é não ter). Até 8 de uma vez, 3 s por
host. Só para apelido seguro; para os outros, e quando o `ssh -G` falha,
demora ou não existe, valem os valores do leitor próprio e o host sai com
`"efetivo": false`.

- O `ssh -G` sempre diz um usuário (o local, se o config não tem): `usuario`
  deixa de ser `null` num host efetivo.
- O `ssh -G` também lista as chaves padrão (`~/.ssh/id_rsa`, `id_ecdsa`,
  `id_ecdsa_sk`, `id_ed25519`, `id_ed25519_sk`, `id_xmss`, `id_dsa`) quando o
  config não tem `IdentityFile`. Se o leitor próprio não achou nenhum e a
  primeira do `ssh -G` é uma dessas, `chave` continua `null` (não é uma
  escolha do config).
- O `ssh -G` avalia o config como numa conexão: um `Match exec` da pessoa
  roda, como rodaria ao conectar.

## `mkporto-motor listar`

```json
{
  "pasta": "~/.ssh",
  "arquivo_do_app": "~/.ssh/config.d/mkporto.conf",
  "hosts": [
    {
      "apelido": "atlas",
      "host": "atlas.oficina.example",
      "usuario": "marina",
      "porta": 22,
      "chave": "~/.ssh/id_ed25519",
      "salto": null,
      "origem": "mkporto",
      "arquivo": "~/.ssh/config.d/mkporto.conf",
      "efetivo": true,
      "escrito": {"host": "atlas.oficina.example", "usuario": "marina", "porta": 22,
                  "chave": "~/.ssh/id_ed25519", "salto": null}
    }
  ]
}
```

- `hosts`: na ordem em que aparecem; a janela ordena por apelido.
- `host`, `usuario`, `porta`, `chave`, `salto`: como o ssh os vê (`efetivo:
  true`) ou, se ele não respondeu, como o leitor próprio os leu (`efetivo:
  false`: `host` é o `HostName` ou o próprio apelido; `porta`, padrão 22).
- `escrito`: só nos hosts de `origem: "mkporto"` (nos outros, `null`): o que
  está escrito no arquivo do app, sem completar com nada. É daí que o
  formulário de edição parte, para não gravar como do host o que veio de um
  `Host *` ou do usuário local.
- `origem`: `"mkporto"` se o host está no arquivo do app (dá para editar e
  remover), senão `"config"`. `arquivo`: onde o `Host` foi declarado.
- Sem `~/.ssh/config`, `hosts` é `[]` (não é erro).

## `mkporto-motor sondar [--] [apelido…]`

Vê se há um servidor ssh em `host:porta` (os valores efetivos) de cada apelido
(de todos, se não vier nenhum), em paralelo. Não autentica: lê a saudação do
servidor e fecha, sem mandar nada. Solta uma linha por resultado, conforme
chegam:

```json
{"apelido": "atlas", "responde": true, "ms": 12, "motivo": null, "servidor": "OpenSSH_10.0", "via": null}
{"apelido": "farol", "responde": false, "ms": null, "motivo": "recusou a conexão", "servidor": null, "via": null}
{"apelido": "cofre", "responde": true, "ms": 180, "motivo": null, "servidor": "OpenSSH_9.9", "via": "farol"}
```

- **Direto**: abre a conexão TCP (até 2 s; `ms` é esse tempo) e espera a
  primeira linha por até 2 s. Começa com `SSH-`: `responde: true` e
  `servidor` é o que vem depois de `SSH-2.0-` (até 64 caracteres imprimíveis).
  Outra coisa, ou silêncio: `responde: false`, motivo
  `"a porta abre, mas não é um servidor ssh"`.
- **Por salto**: roda `MKPORTO_SSH -o BatchMode=yes -o ConnectTimeout=4 -W
  <host>:<porta> -- <salto>`, lê a primeira linha da saída (até 6 s) e mata o
  processo. `SSH-`: `responde: true`, `via` é o salto e `ms`, o tempo até a
  saudação. Senão `responde: false`, motivo `"o salto não deixou passar"`
  seguido da última linha do que o ssh reclamou (cortada). Um salto escrito
  como `usuario:porta` vira `-p <porta>` e `usuario`.
- Continua `responde: null`, motivo `"por salto"` (não dá para sondar sem
  conversa): sem o ssh; salto em cadeia (com vírgula), com colchetes ou barra;
  salto ou endereço que não é uma palavra segura.
- Apelido que não está na lista: `responde: null`, motivo
  `"não conheço esse apelido"`.
- Motivos de quem não responde direto: `"recusou a conexão"`,
  `"não respondeu em 2 s"`, `"não achei esse nome"`, `"sem rota até lá"`,
  `"não deu para conectar"`, `"a porta abre, mas não é um servidor ssh"`.
- O processo termina quando o último respondeu; a saída é 0.

## `mkporto-motor entrar [--] <apelido>`

Testa se a pessoa **entra** na máquina com o que já tem (agente, chaves), sem
abrir sessão: roda `MKPORTO_SSH -o BatchMode=yes -o ConnectTimeout=5 -o
NumberOfPasswordPrompts=0 -- <apelido> true`, com limite de 10 s. Nunca pede
senha.

```json
{"apelido": "atlas", "entra": true, "motivo": null, "ms": 340}
{"apelido": "estufa", "entra": false, "motivo": "a máquina não aceitou nenhuma das suas chaves (envie uma com enviar chave)", "ms": 410}
```

A saída é 0 nos dois casos (não entrar é uma resposta, não um erro). Os
motivos, pelo que o ssh reclamou:

| O ssh disse | Motivo |
|---|---|
| `REMOTE HOST IDENTIFICATION HAS CHANGED` | `a chave do host mudou: confira antes de conectar` |
| `Host key verification failed` | `a chave do host ainda não foi aceita: conecte uma vez pelo terminal` |
| `Permission denied` | `a máquina não aceitou nenhuma das suas chaves (envie uma com enviar chave)` |
| `timed out`, ou os 10 s estouraram | `a máquina não respondeu a tempo` |
| `Connection refused` | `a máquina recusou a conexão` |
| `Could not resolve hostname` | `não achei esse nome` |
| `No route to host`, `Network is unreachable` | `sem rota até lá` |
| outra coisa | a última linha do que ele reclamou, cortada |

Apelido que não existe ou sem o ssh: erro (código 1). Apelido que não é
seguro para uma linha de comando: recusa (código 2).

## `mkporto-motor guardar`

Lê **um** objeto JSON da entrada padrão:

```json
{"apelido": "atlas", "host": "atlas.oficina.example", "usuario": "marina",
 "porta": 22, "chave": "~/.ssh/id_ed25519", "salto": "", "anterior": ""}
```

`usuario`, `chave`, `salto` e `anterior` podem faltar ou vir vazios; `porta`
que falta é 22. `anterior` é o apelido de antes, numa edição que troca o
apelido (o bloco antigo é substituído no mesmo lugar). Guardar um apelido que
já está no arquivo do app substitui aquele host (é a edição sem troca).

Validação (código 1, com `campo`):

| Campo | Regra |
|---|---|
| `apelido` | obrigatório; só `A-Z a-z 0-9 . _ -`; não começa com `-`; até 64 |
| `host` | obrigatório; sem espaço, caractere de controle, `"` nem `#`; não começa com `-`; até 253 |
| `usuario` | as mesmas do host; até 64 |
| `porta` | inteiro de 1 a 65535 |
| `chave` | sem caractere de controle, `"` nem `\`; não começa com `-` |
| `salto` | as mesmas do host (`usuario@host:porta`, vários com vírgula) |

Nenhum campo aceita quebra de linha: é o que impediria alguém de enfiar outra
linha (outra opção) no `config`. Começar com `-` é o que viraria opção na
linha de comando do ssh.

Recusas (código 2): o apelido (ou o `anterior`) já está declarado em outro
arquivo da pessoa (`"… já está no seu ~/.ssh/config …"`): o MkPorto não mexe
no que não é dele.

Resposta:

```json
{"ok": true, "apelido": "atlas", "arquivo": "~/.ssh/config.d/mkporto.conf", "include": "posto"}
```

`include`: `"posto"` (a linha entrou no `config` agora) ou `"ja_havia"`.

## `mkporto-motor remover [--] <apelido>`

Tira o host do arquivo do app e responde `{"ok": true, "apelido": "atlas"}`.
Host de `origem: "config"`: recusa (código 2) dizendo que ele é do seu
`~/.ssh/config`. Apelido que não existe: erro (código 1).

## `mkporto-motor chaves`

As chaves **públicas** de `~/.ssh` (`*.pub`, menos os certificados
`*-cert.pub`), em ordem de nome:

```json
{
  "chaves": [
    {
      "nome": "id_ed25519.pub",
      "arquivo": "~/.ssh/id_ed25519.pub",
      "tipo": "ed25519",
      "bits": 256,
      "impressao": "SHA256:…",
      "comentario": "marina@oficina",
      "publica": "ssh-ed25519 AAAA… marina@oficina"
    }
  ]
}
```

- `bits` e `impressao` vêm do `ssh-keygen -l -f`. Sem o `ssh-keygen` (ou se
  ele falhar), `bits` é `null` e a impressão é calculada pelo motor (SHA256
  do conteúdo da chave pública); se nem isso der, `null`.
- `publica`: a linha do `.pub`, para copiar. `nome` é o que o
  `comando chave` recebe.
- Um `.pub` que não é uma chave (vazio, grande demais, ilegível) não entra.

## `mkporto-motor comando <conectar | chave | gerar> [--] …`

`comando conectar <apelido>`, `comando chave <apelido> <nome.pub>` e
`comando gerar <nome> [comentário]`. Diz o que a janela abre num terminal,
depois de conferir que o apelido existe (ou que a chave ainda não):

```json
{
  "acao": "conectar",
  "apelido": "atlas",
  "argv": ["ssh", "--", "atlas"],
  "texto": "ssh atlas",
  "terminal": ["foot", "--title=ssh · atlas", "ssh", "--", "atlas"],
  "aviso": null
}
```

- `conectar`: `argv` é `["ssh", "--", "<apelido>"]`.
- `chave`: `argv` é `["ssh-copy-id", "-i", "<pasta do ssh>/<nome.pub>", "--",
  "<apelido>"]`. O `nome.pub` tem de ser uma das chaves do `chaves` (um nome,
  não um caminho); a senha da máquina, se for pedida, é digitada no terminal,
  para o ssh.
- `gerar`: `argv` é `["ssh-keygen", "-t", "ed25519", "-f", "<pasta do
  ssh>/<nome>", "-C", "<comentário>"]`; `apelido` e `texto` são `null`. A
  Depois de `chave` e de `gerar` o terminal fica aberto até a pessoa fechar
  (`--wait-after-command` no ghostty, `--hold` no kitty, no foot e no
  alacritty), para dar para ler o que o comando disse; no `conectar` ele
  fecha com a sessão.
  frase-senha da chave é digitada no terminal, para o ssh-keygen: o MkPorto
  não a vê. Validação (código 1, com `campo`): `nome` obrigatório, só
  `A-Z a-z 0-9 . _ -`, sem começar com `-` nem `.`, sem terminar em `.pub`,
  até 64, e nem `<nome>` nem `<nome>.pub` podem existir na pasta do ssh;
  `comentario` sem caractere de controle (nem quebra de linha), sem começar
  com `-`, até 128. Comentário vazio vira `usuario@maquina` daqui. Quando o
  terminal fecha ninguém avisa a janela: a área Chaves tem um "atualizar".
- `texto`: o que o "copiar comando" põe na área de transferência.
- `terminal`: o argv inteiro, já com o terminal na frente. O terminal é o de
  `MKPORTO_TERMINAL`, o dos Ajustes do Mk (`apps.terminal` em
  `~/.config/mk/ajustes.json`) ou o primeiro que houver entre `ghostty`,
  `kitty`, `foot`, `alacritty` e `wezterm`. Cada um recebe o título
  (`ssh · <apelido>`, ou `ssh-keygen · <nome>`) e o comando do seu jeito,
  sempre como lista de argumentos, um argumento do comando por item: os cinco
  executam essa lista direto, sem shell (no `ghostty`, o `-e` desliga a
  expansão de shell desde a 1.2), então um caminho com espaço, aspas ou `$`
  chega inteiro. O `ghostty` abre com a config do Mk, como no Maré. Sem nenhum,
  `terminal` é `null` e `aviso` diz por quê (o `texto` continua servindo para
  copiar).
- Recusa (código 2): um apelido do `config` da pessoa com caracteres fora de
  `A-Z a-z 0-9 . _ -` (ou começando com `-`) não vai para linha de comando
  nenhuma.
- Apelido ou chave que não existe: erro (código 1).

## Códigos de saída

`0` certo; `1` erro de uso, de validação ou de sistema (com `{"erro": …}`);
`2` recusado (o host não é do MkPorto, ou o apelido não é seguro para uma
linha de comando), também com `{"erro": …}`.

## A janela

- Um processo do motor por pedido (não há `servir`): `listar` e `chaves` ao
  abrir; `sondar` ao abrir, no "sondar de novo" (F5) e, só para aquele host,
  depois de guardar. Nunca em laço. `entrar` só quando a pessoa pede ("testar
  entrada"); o resultado fica no detalhe e some ao trocar de host.
- A bolinha de um host por salto diz se ele respondeu pelo salto ("responde
  por farol"); o detalhe mostra o `servidor` e se os valores são os do ssh.
- A área Chaves tem "nova chave" (uma folha com nome e comentário, que abre o
  `comando gerar` num terminal) e "atualizar" (relê as chaves; F5 ali também).
- Título e classe: `MkPorto`. É uma janela de trabalho (não flutua): não há
  regra de flutuar para ela.
- O lançador (`bin/mkporto`) não recebe argumento; `MKPORTO_RAIZ` diz onde
  estão `bin/` e `qml/`.
- `MKPORTO_MOTOR` troca o binário. `MKPORTO_EXEMPLO=pasta` faz a janela usar
  respostas gravadas (`listar.json`, `sondar.json` — uma lista, uma linha por
  item —, `chaves.json`, `guardar.json`, `remover.json`,
  `comando-conectar.json`, `comando-chave.json`, `comando-gerar.json`,
  `entrar.json`) em vez do motor; nesse modo
  ela **não abre terminal nenhum**, só diz o que abriria.
- Atalhos: `Ctrl+F` busca, `Ctrl+N` novo host, `Enter` conecta, `F5` sonda
  (nas Chaves, relê), `Ctrl+T` testa a entrada, `Ctrl+G` nova chave,
  `Ctrl+E` edita, `Ctrl+K` envia uma chave, `Delete` remove (com a pergunta),
  `Ctrl+Shift+C` copia o comando, `Ctrl+1`/`Ctrl+2` trocam
  entre Hosts e Chaves, `↑`/`↓` andam na lista, `Esc` fecha o que estiver
  aberto, `F1` mostra a folha de atalhos.
