# MkPacotes: o contrato entre o coletor e a janela

O MkPacotes é o gerenciador de pacotes do Mk: o que está instalado, o que há
nos repositórios e no AUR, a ficha de cada pacote, instalar, remover, limpar,
e o histórico do pacman. Tem duas metades:

- **o coletor** (`src/`, em Go, um binário só: `bin/mkpacotes-coletor`), que lê
  a base do pacman, fala com o AUR e executa as ações. É onde mora toda a
  lógica, e é todo testado com bases de pacman falsas, dublês de comando e um
  AUR de mentira;
- **a janela** (`qml/`, Quickshell, com os tokens do Maré), que só mostra o que
  o coletor diz e pede o que a pessoa clicou.

Este arquivo é o contrato entre as duas. Muda aqui primeiro, depois nos dois
lados. Toda frase de erro e de aviso vem pronta em português do coletor;
rótulos de coluna e de botão são da janela.

## Regras gerais

- O coletor escreve JSON em UTF-8 no stdout, um objeto por linha (sem quebra
  de linha dentro do objeto).
- Um subcomando que falha escreve no stdout `{"erro": "frase em português"}`
  e sai com código diferente de 0. A janela mostra a frase.
- O que o coletor não conseguiu saber é `null`, nunca 0 nem ausente.
- Tamanhos em **bytes** (`_bytes`), instantes em **segundos desde a época**
  (`_s`).
- Raízes trocáveis para teste, por variável de ambiente:
  `MKPACOTES_PACMAN_DB` (`/var/lib/pacman`: a pasta com `local/`, `sync/` e a
  trava `db.lck`), `MKPACOTES_LOG` (`/var/log/pacman.log`), `MKPACOTES_MK`
  (`/usr/share/mk`), `MKPACOTES_CACHE` (`/var/cache/pacman/pkg`),
  `MKPACOTES_AUR` (`https://aur.archlinux.org`, o endereço do AUR),
  `MKPACOTES_TRAVA` (o arquivo da trava do próprio coletor; padrão
  `$XDG_RUNTIME_DIR/mkpacotes.trava`) e `MKPACOTES_TERMINAL` (o terminal que
  abre o `mk-update`). Os comandos externos (`pacman`, `yay`, `pkexec`,
  `mk-update`, `checkupdates`, o terminal) são achados no `PATH`, então o teste
  põe dublês na frente.
- **Os instalados são lidos direto de `local/`** (um `desc` por pacote), sem
  chamar o pacman: é uma leitura só de arquivos (milissegundos para mil e
  tantos pacotes), não depende do idioma da saída do `pacman -Qi` e funciona
  com o pacman travado. O que é dos repositórios (de onde é cada pacote, a
  versão nova, a busca) vem do pacman (`-Sl`, `-Ss`, `-Si`, com `LC_ALL=C`),
  porque as bases sincronizadas podem estar em zstd, que a biblioteca padrão
  do Go não lê.
- O coletor nunca pede senha, nunca roda como root (recusa) e nunca chama o
  `sudo`. O que precisa de root passa pelo `pkexec`: o Maré é o agente do
  polkit e mostra o pedido de senha. O `yay` roda como a pessoa, com
  `--sudo pkexec`.
- Nome de pacote só entra num comando depois de validado
  (`^[a-z0-9@._+-]+$`, sem começar com `-` nem com `.`) e sempre depois de
  `--`. O que não passa é erro de uso (código 1) e nenhum comando roda.
- Opção que o subcomando não conhece (qualquer coisa começando com `-`) é
  erro de uso. Depois de um `--` tudo é texto de posição: a janela sempre
  manda o que a pessoa escreveu (o nome, a busca) depois dele.

## `mkpacotes-coletor instalados`

Todos os pacotes instalados, em ordem de nome:

```json
{
  "pacotes": [
    { "nome": "bash", "versao": "5.3.20-1", "descricao": "The GNU Bourne Again shell",
      "tamanho_bytes": 10055840, "instalado_em_s": 1792797774, "motivo": "pedido",
      "origem": "core", "orfao": false, "atualizacao": null, "protegido": true }
  ],
  "totais": { "pacotes": 812, "pedidos": 210, "aur": 9, "orfaos": 3, "atualizacoes": 12, "tamanho_bytes": 9000000000 },
  "aviso": null
}
```

- **`motivo`**: `"pedido"` (a pessoa pediu) ou `"dependencia"` (veio por causa
  de outro).
- **`origem`**: o repositório sincronizado que tem o pacote (`core`, `extra`…;
  com o mesmo nome em dois, vale o primeiro do `pacman.conf`) ou `"aur"`
  quando nenhum tem (veio do AUR ou foi instalado à mão). `null` quando o
  pacman não respondeu; aí `aviso` traz a frase e `atualizacao` é `null` em
  todos.
- **`orfao`**: é dependência e nenhum instalado depende dele (nem pelo que ele
  fornece, nem como opcional). É a conta do `pacman -Qdt`.
- **`atualizacao`**: a versão do repositório, quando é mais nova que a
  instalada (comparação do pacman, `vercmp`), segundo a última sincronização.
  Pacote do AUR é sempre `null` aqui: quem pergunta ao AUR é o `atualizacoes`.
- **`protegido`**: remover derruba o sistema (veja `plano`).

## `mkpacotes-coletor procurar <texto…> [--sem-aur]`

Procura nos repositórios (`pacman -Ss`) e no AUR. As palavras viram uma busca
só; menos de 2 letras é erro. No máximo 200 resultados (`cortado: true` se
havia mais).

```json
{
  "resultados": [
    { "nome": "htop", "versao": "3.5.3-1", "descricao": "Interactive process viewer", "origem": "extra",
      "instalado": true, "versao_instalada": "3.5.2-1", "votos": null, "desatualizado": null }
  ],
  "cortado": false, "erro_oficial": null, "erro_aur": null
}
```

- Do AUR, `origem` é `"aur"`, `votos` é o número de votos e `desatualizado`
  diz se está marcado como desatualizado. Dos repositórios, os dois são `null`.
- Sem rede (ou com o AUR fora do ar), `erro_aur` traz a frase e os oficiais
  continuam. Sem o pacman, `erro_oficial` traz a frase e o AUR continua. Só
  quando os dois falham o subcomando falha.
- Ordem: o nome igual ao texto, depois os nomes que começam com ele, depois o
  resto; em cada faixa, os oficiais antes do AUR, e o AUR por votos.

## `mkpacotes-coletor detalhe <nome>`

A ficha de um pacote: instalado (de `local/`), dos repositórios (`pacman -Si`)
ou do AUR, nessa ordem de procura.

```json
{
  "nome": "bash", "versao": "5.3.20-1", "descricao": "The GNU Bourne Again shell", "origem": "core",
  "instalado": true, "motivo": "pedido", "orfao": false, "protegido": true, "atualizacao": null,
  "url": "https://www.gnu.org/software/bash/bash.html", "licencas": ["GPL-3.0-or-later"],
  "mantenedor": "Tobias Powalowski <tpowa@archlinux.org>",
  "tamanho_bytes": 10055840, "baixar_bytes": null, "instalado_em_s": 1792797774, "construido_em_s": 1789501639,
  "depende": [ { "nome": "glibc", "pede": "glibc>=2.38", "instalado": true } ],
  "opcionais": [ { "nome": "bash-completion", "para": "for tab completion", "instalado": false } ],
  "fornece": ["sh"], "conflita": [],
  "requerido_por": ["base", "git"], "opcional_para": [],
  "arquivos": { "total": 120, "executaveis": ["/usr/bin/bash", "/usr/bin/bashbug"] },
  "aur": null, "erro_aur": null
}
```

- **`depende[].instalado`**: há um instalado com esse nome ou que o fornece.
- **`requerido_por`** e **`opcional_para`**: só entre os instalados; vazios
  para quem não está instalado.
- **`arquivos`**: só de pacote instalado (`null` nos outros): quantos arquivos
  (pastas não contam) e os que ficam em `usr/bin/`.
- **`mantenedor`**: quem empacotou (repositórios e instalados) ou o mantenedor
  no AUR.
- **`aur`**: só para pacote do AUR (instalado ou não):
  `{ "versao": "1.2-1", "votos": 12, "popularidade": 0.4, "desatualizado_desde_s": null,
  "mantenedor": "fulano", "modificado_em_s": 1795000000 }`. `mantenedor: null`
  é pacote órfão no AUR. Sem rede, `aur` é `null` e `erro_aur` traz a frase (a
  ficha local de um instalado sai assim mesmo).
- Um pacote do AUR não instalado tem `tamanho_bytes`, `baixar_bytes` e as
  datas de instalação `null`; `construido_em_s` vale a última modificação.
- Pacote que não existe em lugar nenhum: erro, código 1.

## `mkpacotes-coletor atualizacoes [--sem-aur]`

O que há para atualizar:

```json
{
  "atualizacoes": [ { "nome": "htop", "de": "3.5.2-1", "para": "3.5.3-1", "origem": "extra" } ],
  "totais": { "oficiais": 1, "aur": 0 },
  "fonte": "checkupdates", "erro_oficial": null, "erro_aur": null
}
```

- **`fonte`**: `"checkupdates"` (o do `pacman-contrib`: sincroniza numa base à
  parte, sem root, e por isso vê o que saiu agora) ou `"pacman"` (`pacman -Qu`:
  só o que a última sincronização já sabia), quando o `checkupdates` não existe
  ou falhou.
- Os do AUR vêm de uma pergunta ao AUR sobre os instalados de `origem` `"aur"`;
  entra o que tem versão mais nova lá.
- Esta lista é só para mostrar. Quem atualiza é o `mk-update` (`agir atualizar`).

## `mkpacotes-coletor historico [--linhas N] [--busca texto]`

O `pacman.log` resumido, do mais novo para o mais velho:

```json
{ "eventos": [
    { "t_s": 1795480000, "acao": "atualizado", "nome": "htop", "de": "3.5.2-1", "para": "3.5.3-1" },
    { "t_s": 1795470000, "acao": "instalado", "nome": "fd", "de": null, "para": "10.2.0-1" },
    { "t_s": 1795460000, "acao": "removido", "nome": "fuse2", "de": "2.9.9-6", "para": null }
  ],
  "total": 4312 }
```

- **`acao`**: `"instalado"`, `"atualizado"`, `"removido"`, `"reinstalado"` ou
  `"rebaixado"` (voltou para uma versão anterior).
- `--linhas`: quantos eventos devolver (padrão 300, de 1 a 5000; fora disso é
  erro). `--busca`: só os eventos cujo nome contém o texto (sem diferenciar
  maiúsculas). `total` conta os que casam com a busca, antes do corte.
- Linha do log que não é uma dessas (ganchos, avisos) é ignorada; `t_s` de uma
  data ilegível é `null`.

## `mkpacotes-coletor estado`

```json
{ "travado": false, "sincronizado_em_s": 1795400000,
  "cache": { "arquivos": 4227, "bytes": 12000000000 },
  "comandos": { "pacman": true, "pkexec": true, "yay": true, "mk_update": true } }
```

`travado`: existe a trava do pacman (`db.lck`): outro pacman está rodando, ou
um morreu no meio. `sincronizado_em_s`: a base sincronizada mais recente
(`null` sem nenhuma). `cache`: `null` se a pasta não pôde ser lida.

## `mkpacotes-coletor plano <acao> [pacotes…] [--mesmo-assim]`

O ensaio: o que a ação vai fazer, sem fazer nada e sem root. A janela mostra
o plano e só depois da confirmação chama o `agir` com os mesmos argumentos.

`<acao>`: `instalar <pacotes…>`, `remover <pacotes…>`, `orfaos`, `cache`,
`atualizar`.

```json
{
  "acao": "remover", "pedidos": ["htop"],
  "entra": [],
  "sai": [ { "nome": "htop", "versao": "3.5.3-1", "tamanho_bytes": 400000, "pedido": true },
           { "nome": "libnl", "versao": "3.11.0-1", "tamanho_bytes": 1500000, "pedido": false } ],
  "baixar_bytes": null, "liberar_bytes": 1900000,
  "comandos": [ ["pkexec", "pacman", "-Rns", "--noconfirm", "--", "htop"] ],
  "avisos": [], "travado": false, "nada_a_fazer": false, "cache": null
}
```

- **`instalar`**: cada pacote que os repositórios têm vai pelo pacman (ensaio:
  `pacman -S --needed --print`); o resto é procurado no AUR e vai pelo `yay`.
  `entra[]` tem `{ "nome", "versao", "origem", "baixar_bytes", "pedido" }`: os
  pedidos e as dependências que vêm junto. Do AUR, `baixar_bytes` é `null` e
  as dependências que faltam (as de uso e as de construção) entram com
  `versao: null` e `origem: null`. Pacote que não existe em lugar nenhum é
  erro. O que já está instalado e em dia não entra (sai um aviso).
- **`remover`**: `sai[]` é tudo o que sai (ensaio: `pacman -Rs --print`): os
  pedidos (`pedido: true`) e **as dependências que ficariam órfãs**
  (`pedido: false`), para a pessoa ver antes o que mais sai. Remover um pacote
  de que outro instalado depende é erro, com a frase do pacman.
- **`orfaos`**: remover todos os órfãos; `pedidos` são eles. Os protegidos
  ficam de fora (com aviso). Sem órfão nenhum, erro.
- **`cache`**: `pacman -Sc`: tira do cache os pacotes que não estão mais
  instalados. `liberar_bytes` é quanto isso solta e o plano traz
  `"cache": { "arquivos": 4227, "bytes": 12000000000, "sobram": 1300 }`
  (nas outras ações, `cache` é `null`).
- **`atualizar`**: abre o `mk-update` num terminal (`comandos` mostra a
  linha). A atualização é do `mk atualizar`; o coletor não a refaz.
- **`comandos`**: exatamente o que o `agir` vai rodar, na ordem.
- **`avisos`**: frases para a pessoa ler antes de confirmar (um pacote que já
  está em dia, o AUR que compila aqui, um protegido que sai com
  `--mesmo-assim`).
- **`nada_a_fazer`**: não sobrou comando (tudo já instalado, cache sem sobra):
  a janela não oferece confirmar, e o `agir` responde `ok` sem pedir senha.
- **`travado`**: o pacman está travado agora. O plano sai, mas o `agir` vai
  recusar.
- **Recusa** (código 2): `remover` (e `orfaos`) recusa quando entre o que sai
  há pacote **protegido**: `base`, `linux`, `linux-firmware`, `systemd`,
  `pacman`, `glibc`, `sudo`, `limine`, `hyprland`, `quickshell`, `sddm` e tudo
  o que o Mk declara como base
  (`$MKPACOTES_MK/system/packages/base.packages`). Com `--mesmo-assim` o plano
  sai, com um aviso por protegido.

## `mkpacotes-coletor agir <acao> [pacotes…] [--mesmo-assim]`

A ação de verdade. As mesmas ações e os mesmos argumentos do `plano`; refaz o
ensaio e as recusas (a janela não é de confiança). Transmite o progresso, uma
linha de JSON por linha do comando:

```json
{"linha": "$ pkexec pacman -Rns --noconfirm -- htop"}
{"linha": "removendo htop..."}
{"ok": true, "acao": "remover", "pacotes": ["htop"]}
```

ou, no fim, `{"erro": "frase"}` e código diferente de 0. A primeira linha de
cada comando é o próprio comando, com `$ ` na frente.

- `instalar`: `pkexec pacman -S --needed --noconfirm -- <oficiais>` e, se há
  do AUR, `yay -S --needed --noconfirm --sudo pkexec -- <do AUR>` (o yay nunca
  como root). Sem o `yay`, pedir um pacote do AUR é erro.
- `remover` e `orfaos`: `pkexec pacman -Rns --noconfirm -- <pacotes>`.
- `cache`: `pkexec pacman -Sc --noconfirm`.
- `atualizar`: abre `mk-update` num terminal (`MKPACOTES_TERMINAL`, ou o
  primeiro que houver entre `ghostty`, `kitty`, `foot`, `alacritty` e
  `wezterm`), espera a janela dele fechar e responde
  `{"ok": true, "acao": "atualizar", "pacotes": []}`. O `mk-update` é o do
  `PATH` ou o de `$MKPACOTES_MK/bin/mk-update`.
- **Recusas** (código 2, antes de qualquer comando): as do `plano`; o pacman
  travado (`db.lck` existe; menos no `atualizar`, porque o `mk-update` sabe
  lidar com a trava e oferece tirá-la); outra ação do MkPacotes em andamento
  (uma de cada vez, por `flock` em `MKPACOTES_TRAVA`); o coletor rodando como
  root.
- Com `nada_a_fazer`, as linhas são os avisos e `nada a fazer`, e a resposta é
  `ok`.
- A senha recusada ou o pedido fechado no polkit vira a frase
  `"a senha não foi dada: nada foi mudado"`.

## Códigos de saída

`0` certo; `1` erro de uso ou de sistema (com `{"erro": …}`); `2` recusado por
segurança (pacote protegido, pacman travado, outra ação em andamento, root),
também com `{"erro": …}`.
