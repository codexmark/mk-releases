# Atualizações

Como uma mudança sai do seu repositório e chega a um Mk instalado, e o que
fazer quando a atualização para no meio. Para quem mantém o Mk. Para quem só
usa, basta o [README](../README.md#atualizar).

## Os dois repositórios e o canal

- O **repositório privado** é onde você trabalha no dia a dia.
- O **repositório público de lançamentos** recebe só o que passa pelo
  `bin/mk-release`. É dele que todo Mk instalado se atualiza, sem login no
  GitHub.

O endereço do repositório público fica em `system/mk/channel`: é o **canal**.
Quem procura a versão nova lê o canal nesta ordem e usa a primeira linha que
não seja comentário:

| Arquivo | De quem |
|---|---|
| `~/.config/mk/channel` | seu, para um canal só seu |
| `/usr/share/mk/system/mk/channel` | do Mk |

Canal vazio quer dizer "sem canal": o Mk não se atualiza, o Arch e o AUR sim.

A versão instalada é o arquivo `VERSION`, uma linha `commit data` (12 letras do
commit e a data dele, `AAAA-MM-DD`). Vale `~/.config/mk/VERSION`; sem ele,
`/usr/share/mk/VERSION` (o que o ISO gravou). Há versão nova quando o `VERSION`
do canal tem outro commit **e** a data dele não é anterior à instalada. Um Mk
feito de um commit ainda não lançado não "volta" para o último lançamento.

## Lançar: `bin/mk-release`

```bash
bin/mk-release
```

Com uma frase sua no lugar da lista de commits:

```bash
bin/mk-release "o que mudou"
```

O que ele faz, em ordem:

1. Recusa se houver mudança sem commit em arquivo rastreado. Arquivo novo não
   rastreado não impede, e também não vai no lançamento.
2. Avisa (sem parar) se há commits que ainda não foram para o repositório
   privado.
3. Roda `bin/check-keybinds`, `bin/check-shell` e `tests/run` (todos os
   testes automáticos). Se `config/hypr/mare/binds.lua` e
   `shell/mare/Keybinds.js` divergirem, se o bash do Mk não subir, se um teste
   falhar ou se faltar o `node`, não lança.
4. Clona ou atualiza o repositório público em
   `~/.cache/mk-release/mk-releases`.
5. Se o `VERSION` de lá já é o commit atual, diz "já está lançado" e sai.
6. Troca tudo pelo conteúdo do `HEAD` (`git archive`: os arquivos rastreados,
   sem `.git`) e grava o `VERSION`.
7. Grava o `NOVIDADES`: a sua frase, ou os assuntos dos commits desde o
   lançamento anterior, um por linha. No máximo 12 linhas; passando disso,
   ficam 11 e um "… e mais N". No primeiro lançamento, "Primeiro lançamento.";
   o mesmo texto sai quando o commit do lançamento anterior não existe no seu
   clone (histórico reescrito, clone raso).
8. Faz um commit `Mk <data> (<commit>)`, com as novidades no corpo, e dá
   `git push` para o `main` do repositório público.

O commit público sai com o seu `user.name` do git e um endereço "noreply" do
GitHub no lugar do seu e-mail. Para trocar o endereço, `MK_RELEASE_EMAIL`. Para
lançar em outro repositório, `MK_RELEASES`. O `push` usa as suas credenciais
do git: o script não cuida de login.

### Antes de lançar

- Tudo o que está rastreado vai a público, inteiro. Confira que não há senha,
  chave, e-mail ou caminho pessoal em nenhum arquivo.
- Os **assuntos dos commits** também vão: viram o `NOVIDADES` e o corpo do
  commit público. Se não servem para leitura pública, passe a sua frase.
- Mudou algo fora de `~/.config/mk`? Tem de estar no instalador **e** numa
  migração. Veja [migrations/README.md](../migrations/README.md).
- Pacote novo que uma migração usa está na linha `# pacotes:` dela. App novo
  para todo mundo está em `system/packages/apps.packages` (o `mk-update`
  oferece sozinho).
- Mexeu nas listas de pacotes? Rode `bin/check-packages`, com rede. Sem
  alcançar o AUR ele diz "não alcancei o AUR", deixa os pacotes de
  `aur.packages` sem conferir e sai com 1: isso não quer dizer que algum
  pacote falte. As outras listas são conferidas do mesmo jeito.
- Não lance duas vezes o mesmo commit esperando efeito, nem um commit mais
  antigo que o último lançado: os Mk instalados não aceitam data anterior.
- Depois de lançar não há "desfazer" no script. O conserto é um commit novo e
  outro lançamento.

## Atualizar: o que o `mk-update` faz

```bash
mk-update
```

Só mostrar o que há, sem aplicar nada e sem registro:

```bash
mk-update --check
```

`mk atualizar` e o botão "atualizar tudo" dos Ajustes chamam o mesmo
`/usr/bin/mk-update`.

### Uma confirmação só

Depois do "Atualizar agora?", o `pacman` e o `yay` rodam com `--noconfirm`:
nenhum deles pergunta "Continuar a instalação?". O que é decisão sua, e não
confirmação, continua sendo pergunta, mas vem em seguida, antes de qualquer
passo demorado (passo 3): tirar a trava do pacman, os apps novos e o AUR que
ficou de fora. Respondidas essas, dá para sair de perto: nada mais lê o
teclado até o "Enter fecha" do fim.

A senha do `sudo` é pedida uma vez, no começo da parte demorada (antes do
pacman; numa rodada só de AUR, antes de o `yay` compilar), e renovada ao fundo
a cada 50 segundos (`sudo -n -v`) enquanto a rodada durar. Sem isso, uma
transação mais longa que a validade da senha terminaria noutro pedido de
senha, no meio. Rodada que não precisa de root não chama o `sudo`. Se o
`sudoers` não deixa renovar (`timestamp_timeout=0`), a senha volta a ser
pedida a cada passo.

O `--noconfirm` responde o padrão a cada pergunta do pacman. O que isso dá em
cada decisão de verdade:

| Pergunta do pacman | Padrão | O que o `mk-update` faz |
|---|---|---|
| mais de um pacote serve (provedores) | o primeiro da lista | **não deixa escolher**: um ensaio antes da transação (`pacman -Su --print`, sem root, `LC_ALL=C`) mostra a pergunta; a rodada para em `sistema` com a lista na tela e o comando para você rodar à mão |
| dois pacotes em conflito, remover um? | não | o pacman para sem mexer em nada; a rodada para em `sistema` com o aviso |
| pacote que não dá para instalar, pular? | não | idem |
| trocar um pacote pelo sucessor declarado pelo Arch ("Replace") | sim | aceita, como numa atualização feita à mão |
| importar a chave de um empacotador | sim | aceita |

Nos casos em que para, o conserto é rodar `sudo pacman -Syu` no terminal,
responder ao pacman e rodar `mk-update` de novo. O `yay` com `--noconfirm`
também não resolve conflito sozinho: falha, e a rodada para em `aur`.

O ensaio procura o texto do pacman em inglês ("providers available for"). O
comportamento do pacman e do `yay` de verdade com `--noconfirm` foi deduzido do
manual e das mensagens deles, **não ensaiado**: os testes usam um pacman falso.

### Uma rodada de cada vez

A rodada roda dentro do `flock(1)`, com a trava em
`$XDG_RUNTIME_DIR/mk-update.trava` (sem `XDG_RUNTIME_DIR`, em
`~/.local/state/mk/`). Uma segunda rodada, de outro terminal ou do botão dos
Ajustes, diz "já há uma atualização rodando" e sai com 1 sem tocar em nada.
Conta como rodando desde a procura, inclusive parada no "Atualizar agora?";
o "Enter fecha" do fim já não segura a trava. Ela é do processo, não do
arquivo: se a rodada morre, some junto, e não há o que destravar à mão. O
`--check` não trava nem é travado. A trava é por usuário: não impede outro
usuário da mesma máquina (aí quem segura é a trava do próprio pacman). Sem o
`flock` instalado, a rodada roda sem trava.

A ordem:

### 0. O registro

Num terminal de verdade, o `mk-update` roda de novo dentro do `script(1)`, que
grava tudo o que a tela mostra em
`~/.local/state/mk/atualizacao-AAAAMMDD-HHMMSS.log`. A senha do `sudo` não
aparece na tela, então não vai para o registro. No fim, o arquivo é limpo
(cores e barras de progresso) e ficam os 5 últimos. Se nada foi aplicado (tudo
em dia, ou você disse não), o registro da rodada é apagado. Sem terminal ou
sem o `script`, a atualização roda igual, sem registro.

### 1. Procura

Roda `shell/mare/scripts/mare-updates` (o mesmo dos Ajustes) e mostra: o Mk, o número de
pacotes do sistema e do AUR, as migrações pendentes ("ajustes") e os apps
novos do Mk. Sem nada a fazer, diz "tudo em dia" e sai. Senão pergunta
"Atualizar agora?". É a única confirmação da rodada.

### 2. O próprio Mk

Só quando o canal tem versão nova:

1. Clona o canal (raso) numa pasta temporária (`mk-update.XXXXXX`), tira o
   `.git` e preserva o `/usr/share/mk/aur-pendentes.txt`, se existir. A pasta
   é apagada em qualquer saída: clone que falha, erro no meio da troca,
   janela fechada.
2. Copia para `/usr/share/mk.novo`, apaga `/usr/share/mk` e renomeia.
3. Reinstala `/usr/bin/mk-session`, `/usr/bin/mk-update`,
   `/usr/bin/mk-hibernate` (só serve para desfazer um preparo antigo), o gancho
   `/etc/pacman.d/hooks/mk-os-release.hook` e `/usr/lib/mk/mk-dns` (o ajudante
   da página DNS dos Ajustes, que roda como root).
4. Troca as pastas `hypr`, `shell`, `ghostty`, `fastfetch` e `bash` de
   `~/.config/mk`, cada uma de uma vez (copia ao lado e renomeia). Só troca a
   pasta que tem a marca `.instalado-pelo-mk`, ou que não existe; a que perdeu
   a marca fica como está, com o aviso "mantido (não é do Mk)".
5. Grava `~/.config/mk/VERSION`.
6. Dentro do Hyprland, e se o comando `qs` existe, recarrega a config
   (`hyprctl reload`) e reinicia o Maré (`scripts/mare-start --restart`).
7. Passa a vez para o `mk-update` recém-instalado:
   `exec /usr/bin/mk-update --after-mk`. O resto da rodada já usa a lógica
   nova, no mesmo registro. Ele procura de novo e não repete a pergunta.

A troca de `~/.config/mk` vale para quem rodou o `mk-update`. Outro usuário da
mesma máquina continua com a versão antiga na pasta dele até rodar o
`mk-update` também.

### 3. As decisões

Três perguntas possíveis, nesta ordem, uma atrás da outra. Só aparece a que
tem motivo:

1. **A trava do pacman.** Se existe `/var/lib/pacman/db.lck` e não há
   `pacman`, `yay`, `makepkg` nem `pamac` rodando, a trava é sobra de uma
   atualização interrompida. Ele oferece remover.
2. **Os apps novos do Mk** (veja o passo 5): "Instalar junto com a
   atualização?" e, à parte, o LibreOffice.
3. **O AUR que ficou de fora da instalação** (veja o passo 7): "Instalar no
   fim desta rodada?".

Por que aqui: é o primeiro ponto em que todas podem ser feitas. Com Mk novo,
elas vêm depois da troca (passo 2, segundos), e não antes, porque a lista de
apps chega com o Mk novo e quem pergunta já é o `mk-update` novo. Vêm antes
das migrações, do pacman e do AUR, que são o que demora.

Uma exceção: se `/var/lib/mk/apps-oferecidos` ainda não existe e uma migração
do passo 4 o cria nesta rodada, a oferta de apps só passa a existir depois
dela, e a pergunta é feita ali, ainda antes do pacman.

### 4. Migrações que não pedem pacote

As pendentes de `system/` (com `sudo`) e, em seguida, as de `user/`, em ordem
de nome. Cada uma é marcada como feita logo depois de terminar com 0. Se uma
falha, a rodada para ali.

As de sistema com a linha `# pacotes:` ficam para o passo 6.

### 5. Arch: uma transação só

Entram na mesma transação do `pacman`:

- a atualização do sistema;
- os **pacotes pedidos pelas migrações** (só os que faltam);
- os **apps novos do Mk**: o que está em `system/packages/apps.packages` e
  ainda não consta em `/var/lib/mk/apps-oferecidos`, menos o que já está
  instalado. A pergunta foi feita no passo 3. Aceito o LibreOffice num sistema
  com `LANG=pt_BR` em `/etc/locale.conf`, entra junto o
  `libreoffice-fresh-pt-br`.

A oferta de apps só existe se `/var/lib/mk/apps-oferecidos` existir e não
estiver vazio. Com o arquivo vazio, nada é oferecido nunca.

Nunca se instala pacote novo sobre um sistema desatualizado: é por isso que
tudo vai junto. Os comandos, em ordem:

1. `sudo pacman -Sy`: a lista nova.
2. Sai do pedido das migrações o nome que não existe mais nos repositórios,
   com aviso.
3. O ensaio: `pacman -Su --print --noconfirm --needed <pedidos> <apps>`, sem
   root e sem instalar nada. Se ele mostra uma escolha de provedor, a rodada
   para aqui (veja "Uma confirmação só"). Se o ensaio falha por outro motivo,
   segue: o erro quem dá é a transação.
4. `sudo pacman -Su --noconfirm --needed <pedidos> <apps>`: a transação, sem
   perguntas. Se ela falha, a rodada para com o erro do pacman e a instrução
   de rodar `sudo pacman -Syu` à mão.

Se a rodada para no ensaio ou na transação, a lista já foi baixada (`-Sy`) e o
sistema não foi atualizado: não instale pacote avulso com `pacman -S` nesse
estado; termine a atualização primeiro.

Sem pacotes do sistema a atualizar, sem app aceito e sem pacote pedido, o
pacman é pulado.

Depois que o pacman dá certo (ou é pulado), os apps oferecidos entram em
`/var/lib/mk/apps-oferecidos`, **aceitos ou recusados**: cada app é oferecido
uma vez só, para não trazer de volta o que você tirou. Se o pacman falha, a
oferta volta na próxima rodada. O instalador cria esse arquivo com a lista
inteira; numa máquina mais antiga, quem cria é uma migração
(`20261125-apps-oferecidos.sh`).

### 6. Migrações que esperavam os pacotes

As de sistema com `# pacotes:`, agora com os pacotes instalados (ou não: elas
têm de aguentar a falta).

### 7. AUR

Se há pacotes do AUR a atualizar e o `yay` existe:
`yay -Sua --noconfirm --answerclean None --answerdiff None`. Sem o `yay`, este
passo é pulado em silêncio.

Se existe `/usr/share/mk/aur-pendentes.txt` (pacotes do AUR que não vieram no
ISO) e o `yay` existe, a pergunta foi feita no passo 3. Aceita, é aqui que
roda: `yay -S --needed --noconfirm --answerclean None --answerdiff None
--removemake <pacotes>`. Se o `yay` dá certo, o arquivo é apagado; se falha, o
arquivo fica para a próxima rodada e esta termina como "pronto" mesmo assim.
Roda em seguida `system/mk/boot/limine-default`, com o `yay` dando certo ou
não, porque os ganchos do limine reescrevem o menu de boot.

### 8. Fim

Se o kernel em uso já saiu de `/usr/lib/modules`, avisa para reiniciar. Senão,
"pronto". O terminal espera um Enter para fechar.

### O resumo da rodada

Ao sair, com sucesso ou não, qualquer rodada que passou do "Atualizar agora?"
grava `~/.local/state/mk/ultima-atualizacao.json`:

| Campo | O que é |
|---|---|
| `when` | quando a rodada terminou (epoch) |
| `started` | quando começou |
| `ok` | `true` se foi até o fim |
| `step` | onde parou: `mk`, `ajustes`, `sistema` ou `aur`; vazio se deu certo |
| `reboot` | o kernel em uso já saiu do disco |
| `applied` | a última vez que tudo deu certo |
| `from`, `to` | a versão do Mk antes e depois |
| `log` | o caminho do registro desta rodada |

Fechar a janela no meio também grava o resumo, com o passo em que estava.

## Como os Ajustes e a ilha leem isso

`shell/mare/scripts/mare-updates` procura sem `sudo` e responde em JSON:
pacotes do Arch (`checkupdates`, numa base temporária), do AUR (`yay -Qua`), a
versão do canal (um clone raso que só lê `VERSION` e `NOVIDADES`, e nunca pede
senha), as migrações pendentes, o resumo da última rodada, a data da última
atualização e se falta reiniciar. A data da última atualização é a mais nova
entre o `applied` do resumo e o último "starting full system upgrade" do
`/var/log/pacman.log`: um `pacman -Syu` feito à mão também conta.

`shell/mare/Updates.qml` roda esse script:

- 2 minutos depois do login e depois a cada 3 horas, se "Procurar sozinho"
  (`updates.auto`) estiver ligado;
- ao abrir Ajustes › Atualizações, se a última procura tem mais de 10 minutos;
- no botão "procurar agora";
- sempre que o `ultima-atualizacao.json` muda, ou quando o terminal aberto
  pelo "atualizar tudo" fecha.

**Ajustes › Atualizações** mostra o número de atualizações (o Mk conta 1, mais
as migrações, mais os pacotes), os 12 primeiros pacotes, a linha "última vez:
… tudo certo" ou "parou em: …", o botão "ver registro" (abre o registro no fim,
com `less`; some quando o arquivo já saiu dos 5 últimos), "reinicie para o
kernel novo" e, havendo Mk novo, "o que chega": as linhas do `NOVIDADES`.

**A ilha** só mostra o ícone com o número quando importa (e se "Avisar na
ilha", `updates.island`, estiver ligado):

- há versão nova do Mk (o ícone fica em cobre), ou
- há migração pendente, ou
- há pacotes esperando e a última atualização foi há 7 dias ou mais, ou
  nunca.

Clicar nele abre Ajustes › Atualizações.

Erros de procura aparecem na página: sem internet ou espelho fora, falta do
`pacman-contrib`, ou "sem acesso ao canal do Mk".

## Snapshots e voltar atrás

O que o instalador deixa pronto:

- A raiz é um btrfs com os subvolumes `@` (`/`), `@home`, `@log` (`/var/log`)
  e `@pkg` (cache do pacman). O `/boot` é a partição EFI, fora do btrfs.
- `snapper` com a configuração `root` para `/`: sem snapshots por hora
  (`TIMELINE_CREATE="no"`), limite de 5 (`NUMBER_LIMIT` e
  `NUMBER_LIMIT_IMPORTANT`), e o `snapper-cleanup.timer` ligado.
- `snap-pac`: cada transação do `pacman` cria um snapshot antes e outro depois.
- `/etc/default/limine` com `MAX_SNAPSHOT_ENTRIES=auto` e "Snapshots" na ordem do
  menu, e o `limine-snapper-sync.service` ligado, se o pacote estiver
  instalado.
- O `fstab` sem `subvolid`, para a raiz continuar montando depois de uma troca
  de subvolume.

O que isso quer dizer na prática:

- O snapshot cobre `/` (e portanto `/usr/share/mk`, `/etc`, `/var/lib/mk`).
  **Não cobre** a sua home: `~/.config/mk` e `~/.local/state/mk` não voltam
  junto.
- A troca do Mk (passo 2) não é uma transação do pacman e não cria snapshot
  por si. O snapshot "antes" é o do `pacman` que vem depois, já com o Mk novo
  em `/usr/share/mk`. Rodada que só troca o Mk não cria snapshot nenhum.
- O submenu "Snapshots" no boot vem de dois pacotes do AUR,
  `limine-mkinitcpio-hook` e `limine-snapper-sync`. Se o ISO trouxe parte dos
  pacotes do AUR e esses faltaram, estão em `aur-pendentes.txt` e o submenu só
  existe depois que o `mk-update` os instala (passo 7). Essa oferta depende do
  `yay`, e o `yay-bin` também é do AUR: se ele próprio ficou de fora, ela não
  aparece. Se o ISO veio sem nenhum pacote do AUR, o instalador só avisa e não
  grava o `aur-pendentes.txt`. Nesses dois casos, instale à mão o `yay` e os
  pacotes de `/usr/share/mk/system/packages/aur.packages`.

Para ver os snapshots que existem:

```bash
sudo snapper -c root list
```

Para voltar: reinicie, abra "Snapshots" no menu do limine e escolha um
anterior à atualização. O Mk só configura isso; quem lista, inicia e restaura
o snapshot é o `limine-snapper-sync`, e o Mk não tem comando próprio de
restauração. **Não foi verificado** neste projeto: iniciar por um snapshot,
tornar a volta permanente, nem o comportamento com o kernel (que mora na
partição EFI, fora do snapshot). Consulte a documentação do
`limine-snapper-sync` antes de depender disso. A instalação com LUKS, com
NVIDIA e em BIOS antigo também nunca foi exercitada.

Voltar **só o Mk** para uma versão anterior não existe: o `mk-update` sempre
instala o que está na ponta do canal e recusa data mais antiga. O conserto de
um lançamento ruim é outro lançamento.

## Quando a atualização para no meio

Primeiro veja onde parou. Em Ajustes › Atualizações, a linha "parou em: …" e o
botão "ver registro". Ou no terminal:

```bash
jq . ~/.local/state/mk/ultima-atualizacao.json
```

```bash
less -R +G "$(jq -r .log ~/.local/state/mk/ultima-atualizacao.json)"
```

Depois, quase sempre, **rode `mk-update` de novo**. Cada passo aguenta
repetição: a troca do Mk se refaz inteira, migração feita não roda outra vez,
a trava do pacman é oferecida para remoção, e o pacman continua de onde dá.
Se a resposta for "já há uma atualização rodando", há outra rodada viva (outra
janela, talvez parada numa pergunta): termine ou feche aquela.

| Parou em | O que costuma ser | O que fazer |
|---|---|---|
| `mk` | sem rede ou sem acesso ao canal durante o clone; `sudo` recusado | `mk-update` de novo. Enquanto `~/.config/mk/VERSION` não muda, a versão nova continua sendo oferecida |
| `ajustes` | uma migração saiu com erro | o registro diz qual (a última linha `system/…` ou `user/…`). Ela continua pendente. Se ela não tem `# pacotes:`, os pacotes e o AUR não rodaram; se tem, o pacman já rodou e só o AUR ficou por fazer. Corrija a causa e rode de novo. Se o defeito é da migração, o conserto é um lançamento novo |
| `sistema` | o pacman falhou: espelho, chave, conflito de arquivo, disco cheio; ou precisava de uma escolha sua (provedor, conflito entre pacotes), que o `mk-update` não faz por você | leia o erro do pacman no registro, resolva e rode de novo. Se era uma escolha, rode `sudo pacman -Syu` no terminal, responda e rode `mk-update` de novo. Os apps oferecidos voltam a ser oferecidos |
| `aur` | um pacote do AUR não compilou | o sistema já está atualizado. Tente de novo mais tarde, ou trate o pacote com o `yay` |
| `início` | parou antes do primeiro passo | `mk-update` de novo |

Casos à parte, deduzidos do código e não ensaiados:

- **`/usr/share/mk` sumiu** (interrompido entre apagar a pasta e renomear a
  nova). A cópia nova está ao lado:

  ```bash
  sudo mv /usr/share/mk.novo /usr/share/mk
  ```

  Se `/usr/share/mk` ainda existe pela metade (interrompido durante o
  apagar), esse `mv` poria a pasta nova dentro da velha. Confira que
  `/usr/share/mk.novo` existe e apague o resto antes:

  ```bash
  sudo rm -rf /usr/share/mk
  ```

  Depois, `mk-update`. O `mv` é obrigatório: sem `/usr/share/mk` o `mk-update`
  até roda (mora em `/usr/bin` e usa o `mare-updates` de
  `~/.config/mk/shell/scripts`), mas só atualiza o Arch e o AUR. O canal padrão
  e as migrações são lidos de `/usr/share/mk`, então o Mk aparece como "sem
  canal" (a não ser que exista `~/.config/mk/channel`) e nenhuma migração é
  vista.
- **O Mk trocou mas o `mk-update` novo quebra.** As migrações e os pacotes
  ficam por fazer; o Arch ainda atualiza à mão com `sudo pacman -Syu`. Se ele
  quebra num passo adiante, o conserto vem por outro lançamento: a procura e a
  troca do Mk rodam antes de todo o resto. Se ele nem começa, rode o de um
  clone do canal já consertado, que troca o Mk e passa a vez ao novo:

  ```bash
  git clone --depth 1 https://github.com/codexmark/mk-releases.git /tmp/mk
  ```

  ```bash
  bash /tmp/mk/bin/mk-update
  ```
- **O Maré não voltou depois da troca.** Saia da sessão e entre de novo: o
  Hyprland e o Maré sobem com a config nova no login.
- **O sistema não inicia depois da atualização.** É o caso dos snapshots,
  acima, com as ressalvas de lá.
