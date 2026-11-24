# Trabalhar no Mk

Para quem muda o Mk: onde fica cada coisa, como testar sem quebrar a própria
sessão, o que cada ferramenta faz e o que conferir antes de lançar.

Os assuntos que têm documento próprio não se repetem aqui:

| Arquivo | Assunto |
|---|---|
| [shell/mare/README.md](../shell/mare/README.md) | o Maré por dentro: arquivos, IPC, scripts, Ajustes |
| [installer/README.md](../installer/README.md) | o que o instalador faz, passo a passo |
| [iso/README.md](../iso/README.md) | gerar o ISO e testar numa VM |
| [migrations/README.md](../migrations/README.md) | como escrever uma migração |
| [atualizacoes.md](atualizacoes.md) | o canal, o `mk-release` e o `mk-update` em detalhe |

## O repositório

```
bin/         ferramentas de quem desenvolve, mais mk-session e mk-update
config/      o que vai para ~/.config/mk: hypr, bash, ghostty, fastfetch
shell/mare/  o Maré (Quickshell/QML), vai para ~/.config/mk/shell
installer/   mk-install e o arquivo de respostas de exemplo
iso/         build, vm e o que o Mk põe por cima do archiso
migrations/  system/ e user/: mudanças que o mk-update aplica uma vez
scripts/     system/ e user/: passos avulsos, fora da atualização
system/      packages/ (as listas) e mk/ (marca, boot, login, tema, fontes)
docs/        este arquivo e atualizacoes.md
```

**`config/hypr/`** — `hyprland.lua` e um arquivo por assunto em `mare/`:
`env`, `look`, `motion`, `input`, `rules`, `display`, `autostart`, `paths`,
`binds`. O `paths.lua` descobre onde o Mk está instalado a partir do próprio
caminho; nada ali aponta para `~/.config/hypr` nem para `~/.config/quickshell`,
que podem ser de outro desktop. `hyprlock.conf` e `hypridle.conf` ficam ao lado
do `hyprland.lua`.

**`config/bash/`** — `mk.bash` (o terminal do Mk e o comando `mk`),
`starship.toml` e `seu.bash.exemplo`. `config/ghostty/config`,
`config/fastfetch/` (a ficha e o `mark.png` dela) e `config/mimeapps.list` (os
apps padrão de fábrica) completam a pasta.

**`shell/mare/`** — todo o QML numa pasta só, sem subpastas de código. Ao lado:
`scripts/` (os `mare-*` que o shell roda por fora), `assets/` (a marca e o
grão) e `wallpapers/` (os papéis de parede do Mk: seis autorais e doze desenhados por código).

**`system/mk/`** — a cara do Mk: `art/`, `plymouth/`, `sddm/`, `look/`,
`fonts/`, `boot/limine-default`, `os-release`, `issue`, `mk-os-release.hook` e
`channel`. Veja [Marca e arte](#marca-e-arte).

**`iso/work/` e `iso/out/`** são saídas do build e ficam fora do git
(`iso/.gitignore`). É o único `.gitignore` do repositório.

### Onde cada parte vai parar

Num Mk instalado pelo ISO, o repositório inteiro fica em `/usr/share/mk`
(sem `.git`), com um `VERSION` (`commit data`). A partir dele:

| Origem | Destino | Quem põe |
|---|---|---|
| `config/hypr`, `shell/mare`, `config/ghostty`, `config/fastfetch`, `config/bash` | `~/.config/mk/{hypr,shell,ghostty,fastfetch,bash}`, com a marca `.instalado-pelo-mk` | instalador (via `/etc/skel`), `mk-update`, `try-session`, `new-user` |
| `config/bash/seu.bash.exemplo` | `~/.config/mk/seu.bash` | instalador (via `/etc/skel`) |
| `config/mimeapps.list` | `~/.config/mimeapps.list` | instalador (via `/etc/skel`) |
| `bin/mk-session`, `bin/mk-update` | `/usr/bin/` | instalador, `mk-update` |
| `system/mk/mk-os-release.hook` | `/etc/pacman.d/hooks/` | instalador, `mk-update` |
| `system/mk/os-release`, `issue` | `/etc/os-release`, `/etc/issue` | instalador; depois o gancho, a cada atualização do pacote `filesystem` |
| `system/mk/look/themes/Mk` | `/usr/share/themes/Mk` | instalador |
| ícones gerados por `system/mk/look/make-icons` | `/usr/share/icons/Mk` | instalador |
| `system/mk/look/qt6ct/` | `/etc/skel/.config/qt6ct/` | instalador |
| `system/mk/fonts/*.otf`, `OFL.txt` | `/usr/share/fonts/mk/`, `/usr/share/licenses/mk-fonts/OFL.txt` | instalador |
| `system/mk/plymouth/mk`, `system/mk/sddm/mk` | `/usr/share/plymouth/themes/mk`, `/usr/share/sddm/themes/mk` | instalador |

O instalador não escreve na home: enche `/etc/skel/.config/`, e a home recebe a
cópia quando o usuário é criado (`useradd -m`).

O `mk-update` só troca `/usr/share/mk`, as cinco pastas de `~/.config/mk`, os
dois binários e o gancho. **Tudo o mais da tabela só chega a um Mk já instalado
por migração.** Mudou o tema GTK, o plymouth, o sddm, uma fonte? Faça no
instalador e escreva a migração (veja as de `migrations/system/` que já fazem
isso, como `20261121-logo-plano.sh` e `20261121-apps-folha.sh`).

## O ciclo de trabalho

Do mais barato ao mais caro. Suba um degrau só quando o anterior passou.

### 0. Teste primeiro

Toda mudança de comportamento começa por um caso em `tests/` que falha, e só
então o código. As regras e o jeito de escrever estão em
[tests/README.md](../tests/README.md).

```bash
tests/run
```

Leva um ou dois minutos, sem root, sem rede e sem sessão gráfica: os comandos que
mexem na máquina são dublês. Ligue uma vez por clone o gancho que roda isto
antes de cada `git push`:

```bash
git config core.hooksPath tests/hooks
```

### 1. Editar e conferir

Sempre que mexer no que cada um cobre:

```bash
bin/check-keybinds
```

```bash
bin/check-packages
```

O primeiro, depois de mudar `config/hypr/mare/binds.lua` ou
`shell/mare/Keybinds.js`. O segundo, depois de mudar qualquer lista de
`system/packages/`.

### 2. O Maré num Hyprland aninhado: `bin/dev-shot`

Testa o QML sem tocar na sua sessão: sobe um Hyprland aninhado, roda nele uma
cópia do shell, manda os comandos de IPC que você der e mostra os erros do log.

```bash
bin/dev-shot <saida.png> "settings monitores"
# duas telas de larguras diferentes, uma tecla dentro do aninhado, e a captura da tela virtual
MK_DEV_HEADLESS=1080x1920 bin/dev-shot <saida.png> "toggle launcher" "tecla -k Escape" "state"
```

Cada argumento depois do arquivo é o que viria depois de `ipc call mare`. Sai
com 1 se o log do Maré tiver erro de QML: é o teste mais rápido para "o que eu
escrevi carrega?".

Para a janela do aninhado não aparecer na sua frente, mande-a antes para um
workspace escondido (a regra some no próximo reload do seu Hyprland):

```bash
hyprctl eval 'hl.window_rule({ match = { class = "^aquamarine$" }, workspace = "special:mkdev silent" })'
```

Escondida, a janela não desenha quadros novos e a captura não sai ("captura:
indisponível"); o log continua valendo. **Para ver a imagem, rode sem a regra**,
com a janela à vista. Variáveis e limites estão em
[shell/mare/README.md](../shell/mare/README.md#testar-sem-mexer-na-sua-tela).

O que o `dev-shot` não cobre: a config do Hyprland do Mk. O aninhado sobe com
uma config mínima, escrita na hora (um monitor, as curvas de
`config/hypr/mare/motion.lua` e o Maré). Atalhos, regras de janela e o
`autostart.lua` não são carregados. Ele também fecha tudo ao terminar: não é
uma sessão para usar.

### 3. A sessão de verdade

Não há no repositório um script que deixe um Hyprland aninhado aberto com a
config inteira do Mk. Para usar o Mk de fato, com atalhos e autostart, entre
numa sessão "Mk" pela tela de login. Dois jeitos, a partir do clone:

```bash
bin/try-session install
```

Instala no seu usuário, ao lado do seu desktop.

```bash
sudo bin/new-user mare --wallpapers
```

Cria um usuário só para isso, que nasce como numa instalação nova (com as
boas-vindas no primeiro login). Rodar de novo com o mesmo nome só atualiza a
config dele (e repõe o lançador e a entrada do login): é o jeito de levar a sua
mudança para o usuário de teste.

Se a sessão não abrir:

```bash
sudo bin/diagnose-user mare
```

Se um dia você subir à mão um Hyprland aninhado com o `hyprland.lua` do Mk,
saiba que o `autostart.lua` inicia um `hypridle` também ali, e os tempos dele
(`config/hypr/hypridle.conf`) baixam o brilho, bloqueiam a sessão e chamam
`systemctl suspend`, que valem para a máquina inteira. Tire essa linha da
cópia que for usar. (O `dev-shot` não corre esse risco: não carrega o
`autostart.lua`, e os Ajustes só reiniciam um `hypridle` que o próprio Mk
iniciou.)

### 4. Uma VM

Para tudo o que não é a sessão: ISO, instalador, boot, snapshots, tela de
login, migrações, atualização.

```bash
iso/build
```

```bash
iso/vm iso
```

No live, `mk-install` e o disco `/dev/vda`. Depois, `iso/vm` inicia pelo disco
instalado e `iso/vm reset` apaga a VM. Para instalar sem responder perguntas,
`mk-install --answers <arquivo>`, a partir de `installer/answers.example`.
Tudo isso está em [iso/README.md](../iso/README.md) e
[installer/README.md](../installer/README.md).

A VM não mostra Wi-Fi, os perfis `amd`/`intel`/`nvidia`, mais de um monitor,
impressoras, Bluetooth nem bateria. O que depende disso segue sem teste (veja
"Estado do projeto" no [README](../README.md)).

### 5. Lançar

```bash
bin/mk-release
```

Antes, passe pela [lista no fim deste arquivo](#antes-de-lançar).

## As ferramentas de `bin/`

### `bin/dev-shot`

```
bin/dev-shot <saida.png> [comando ipc]...
```

Copia o shell (`shell/mare`, ou a pasta em `MK_DEV_SHELL`) para uma pasta
temporária em `$XDG_RUNTIME_DIR`, sobe um Hyprland aninhado com uma config
mínima e roda a cópia nele com `QT_SCALE_FACTOR=0.85`. Espera o Quickshell
dizer que carregou, manda cada comando de IPC (com `MK_DEV_WAIT` segundos entre
eles, padrão 1,5), tenta a captura com o `grim` e filtra o log atrás de erros
de QML. Sai com 0 se o log está limpo, 1 se há avisos, 2 se o Hyprland aninhado
não subiu. No fim mata o aninhado e apaga a pasta temporária: como o shell é
uma cópia, o seu `ajustes.json` não é tocado. Precisa de `Hyprland`, `qs`,
`jq` e `grim`, e de rodar dentro de uma sessão Hyprland (usa `hyprctl
instances` para achar o aninhado).

### `bin/check-keybinds`

A lista de atalhos existe duas vezes: em `config/hypr/mare/binds.lua` (o que o
Hyprland executa) e em `shell/mare/Keybinds.js` (o que os Ajustes e a busca
mostram). O script confere que as duas têm as mesmas ações (id, grupo,
descrição, combo e marcas, na mesma ordem), as mesmas famílias e as mesmas
teclas de emergência. O comando de verificação mora no cabeçalho do
`Keybinds.js`; o script só o extrai e roda. Sai com 1 se divergirem e com 2 se
faltar o `node` ou o comando não for achado. O `mk-release` chama este script e
não lança se ele falhar.

### `bin/lib-packages.sh`

Não se roda: carrega-se com `source`. Dá `PKG_DIR` (o `system/packages` ao lado,
se a variável já não vier definida) e duas funções. `read_list <arquivo>`
imprime os pacotes de uma lista, sem comentários nem linhas em branco.
`read_hardware <perfil>...` imprime os pacotes das seções pedidas de
`hardware.packages`. Quem usa: `bin/check-packages`, `bin/install-packages`,
`installer/mk-install` e `iso/build`.

### `bin/check-packages`

Confere cada pacote das listas contra os repositórios, sem instalar nada. Os de
`base`, `desktop`, `dev`, `apps` e de todas as seções de `hardware` precisam
existir em `core`, `extra` ou `multilib` (consulta o `pacman -Si`, então vale a
base de pacotes sincronizada da sua máquina). Os de `aur` precisam existir no
AUR (consulta a API do AUR com `curl` e `jq`) e **não** existir nos oficiais.
Imprime `FALTA` ou `AVISO` para cada problema e sai com 1 se houver algum.

### `bin/install-packages`

```
bin/install-packages <perfil-de-hardware...> [--no-apps]
```

Instala as listas num Arch que já existe: `base`, `desktop`, `dev`, `apps` (a
não ser com `--no-apps`) e as seções de hardware pedidas (`amd`, `intel`,
`nvidia`, `vm`; ao menos uma), num `sudo pacman -S --needed --noconfirm` só.
Depois o AUR: instala o `yay-bin` se não houver `yay` e passa a lista
`aur.packages` um pacote por vez; o que falha vira aviso e não interrompe. Não é
usado pelo instalador (que usa `pacstrap`); serve para pôr os pacotes do Mk num
Arch instalado de outro jeito.

### `bin/try-session`

```
bin/try-session install | uninstall | status
```

Instala o Mk como uma sessão separada no **seu** usuário. O `install` copia
`config/hypr`, `config/ghostty`, `config/fastfetch`, `config/bash` e
`shell/mare` para `~/.config/mk`, o tema GTK para `~/.local/share/themes/Mk`,
gera os ícones em `~/.local/share/icons/Mk`, grava `~/.config/mk/VERSION` e
roda `Hyprland --verify-config` na config copiada. Na primeira vez num usuário,
marca as migrações de usuário como feitas. Os passos que precisam de root
(pacotes que faltam, `/usr/local/bin/mk-session` e
`/usr/share/wayland-sessions/mk.desktop`) ele só **imprime**. O `uninstall`
remove o que o `install` criou. Uma pasta sem a marca `.instalado-pelo-mk` não
é sobrescrita nem apagada, em nenhum dos dois. Sem argumento, faz o `status`.
O script ainda limpa restos de quando o projeto se chamava `meu-os`.

### `bin/new-user`

```
sudo bin/new-user <nome> [--admin] [--wallpapers]
```

Cria um usuário já com o Mk, para testar do zero. Em ordem: instala com
`pacman` os pacotes que faltam para a sessão, instala `/usr/local/bin/mk-session`
e a entrada "Mk" do login, cria o usuário (grupos `video`, `audio`, `input`;
`wheel` com `--admin`), copia a config para `~/.config/mk` dele, liga o
`mk.bash` no `~/.bashrc`, marca as migrações de usuário como feitas e pede a
senha. `--wallpapers` copia as imagens de `~/Pictures/Wallpapers` de quem
chamou o `sudo`. Com um usuário que já existe, não cria nem pede senha:
atualiza a config dele, e ainda instala os pacotes que faltam e regrava o
lançador e a entrada do login, que vêm antes no script. As migrações de
usuário só são marcadas quando o usuário é criado. Diferente do `try-session`,
este roda os passos de root ele mesmo.

### `bin/diagnose-user`

```
sudo bin/diagnose-user <nome>
```

Junta em `/tmp/mk-diagnostico` o que explica uma sessão que não abriu, sem
alterar nada na home do usuário: a listagem de `~/.config/mk/shell` e
`~/.config/mk/hypr`, o fim de `~/.cache/mk/mare.log`, o estado e os relatórios
de crash do Quickshell, os processos do usuário e as capturas de tela das
últimas quatro horas. A pasta é apagada e recriada a cada rodada e fica com
dono de quem chamou o `sudo`.

### `bin/mk-session`

O lançador da sessão: confere que `~/.config/mk/hypr/hyprland.lua` existe e
troca a si mesmo por `uwsm start ... start-hyprland -- -c <config>`. É o `Exec`
da entrada "Mk" da tela de login. O instalador o põe em `/usr/bin`; o
`try-session` e o `new-user`, em `/usr/local/bin`.

### `bin/mk-update`

```
mk-update [--check]
```

O atualizador de um Mk instalado: o próprio Mk pelo canal git, as migrações, o
Arch e o AUR, nessa ordem, com registro em `~/.local/state/mk/`. Lê tudo de
`/usr/share/mk` (ou de `MK_SHARE`), não do seu clone: no repositório ele é
código-fonte, e o que roda é a cópia em `/usr/bin`. Depois de trocar o Mk, ele
chama a versão nova de si mesmo com `--after-mk`, uso interno. O passo a passo
está em [atualizacoes.md](atualizacoes.md).

### `bin/mk-release`

```
bin/mk-release ["o que mudou"]
```

Publica o `HEAD` no repositório público de lançamentos, de onde todo Mk
instalado se atualiza. Recusa se houver mudança sem commit em arquivo
rastreado, roda o `check-keybinds`, troca o conteúdo do repositório público
pelo do commit (`git archive`), grava `VERSION` e `NOVIDADES`, faz o commit e o
`push`. Se o `VERSION` de lá já é o `HEAD`, diz "já está lançado" e sai com 0.
`MK_RELEASES` aponta para outro repositório; `MK_RELEASE_EMAIL` troca o e-mail
de autor do commit público (o padrão é um endereço "noreply" do GitHub). Os
detalhes estão em
[atualizacoes.md](atualizacoes.md#lançar-binmk-release).

## `scripts/`

Passos avulsos, fora da atualização. Num Mk instalado ficam em
`/usr/share/mk/scripts/`.

**`scripts/system/firewall.sh`** — ufw: nega toda entrada, libera a saída, abre
a porta 53317 (LocalSend) e o DNS do Docker para os containers, liga o serviço
e, se houver `ufw-docker`, roda `ufw-docker install`. Chama `sudo` por dentro.
O instalador grava as mesmas regras por conta própria e só cita este script
quando não consegue.

**`scripts/user/neovim.sh`** — clona o starter do LazyVim em `~/.config/nvim`,
só se a pasta não existir. O instalador roda para o usuário criado.

**`scripts/user/mise-tools.sh`** — `mise use -g` de node (LTS), python e três
CLIs de IA (Claude Code, Codex, Gemini), e cria `~/Work`. Ninguém roda por
você: o instalador só lembra dele no fim.

## Listas de pacotes

Ficam em `system/packages/`. Um pacote por linha; `#` até o fim da linha e
linhas em branco são ignorados.

| Lista | Para quê | De onde vêm |
|---|---|---|
| `base.packages` | o sistema: kernel, boot (limine, btrfs, snapper), rede, impressão, Bluetooth, áudio, energia, fontes | oficiais |
| `desktop.packages` | a sessão: Hyprland, uwsm, sddm, Quickshell e o que o Maré chama (captura, área de transferência, tema) | oficiais |
| `dev.packages` | o kit de desenvolvimento: terminal, CLIs, git, editor, linguagens, containers | oficiais |
| `apps.packages` | os apps gráficos do dia a dia; opcionais | oficiais |
| `hardware.packages` | drivers e microcode, em seções `[amd]`, `[intel]`, `[nvidia]`, `[vm]` | oficiais |
| `aur.packages` | o que só existe no AUR, pelo nome do `PackageBase` | AUR |

Quem lê as listas:

- **O instalador** junta `base`, `desktop` e `dev`, mais `apps` se você disse
  sim, mais as seções de hardware escolhidas, e instala tudo num `pacstrap`.
  Com apps e idioma `pt_BR`, acrescenta `libreoffice-fresh-pt-br`, que não está
  em lista nenhuma.
- **O `iso/build`** lê `aur.packages`, compila cada um e monta um repositório
  dentro do ISO. O instalador instala dali o que compilou; o que ficou de fora
  vai para `/usr/share/mk/aur-pendentes.txt`, e o `mk-update` oferece instalar
  com o `yay`.
- **O `mk-update`** só olha `apps.packages`. O instalador grava a lista do dia
  em `/var/lib/mk/apps-oferecidos`; um app que entrar depois é oferecido uma
  vez em cada Mk instalado e entra na mesma transação do `pacman -Syu`.
- **O `bin/install-packages`** lê todas.

O que isso quer dizer na prática:

- **App novo para todo mundo:** uma linha em `apps.packages`. O `mk-update`
  cuida do resto.
- **Pacote novo em `base`, `desktop` ou `dev`:** só chega a instalações novas.
  Para os Mk já instalados, escreva uma migração de sistema com a linha
  `# pacotes: nome` (veja [migrations/README.md](../migrations/README.md)).
- **Pacote novo em `aur.packages`:** só chega a quem instalar por um ISO novo.
  A linha `# pacotes:` não serve aqui: o `mk-update` deixa de fora o nome que
  não está nos repositórios oficiais. Para os Mk já instalados, escreva uma
  migração de sistema que acrescente o nome a `/usr/share/mk/aur-pendentes.txt`
  (como faz a `20261122-boot-entrada.sh` com o `blesh-git`); o `mk-update`
  então pergunta se instala com o `yay`.
- **Se existe em `core`/`extra`, vai numa lista oficial.** O AUR é último
  recurso, e o `check-packages` reclama de um pacote do AUR que também exista
  nos oficiais.
- **Pacotes sem os quais a sessão não abre** também estão, à mão, na lista
  `needed` do `bin/try-session` e do `bin/new-user`. As duas não são lidas das
  listas e não são iguais entre si: ao acrescentar algo de que o Maré depende,
  atualize as duas. Hoje a do `try-session` não tem o `uwsm`, que o
  `mk-session` chama: num Arch sem ele, instale-o à mão.

Depois de editar qualquer lista:

```bash
bin/check-packages
```

## Marca e arte

A paleta é a mesma em tudo: preto quente `#0b0b0b`, osso `#cdc4ba`, cobre
`#ce5e32`.

### Os geradores (`system/mk/art/`)

As imagens da marca são geradas por código, e o que os scripts geram **está no
git**. Ninguém precisa rodá-los para instalar ou lançar: rode só quando mudar o
desenho, e faça o commit do que saiu.

Dependências: o `python` (está em `dev.packages`) com os módulos `fontTools`,
Pillow e `numpy`, e o `rsvg-convert`, que não estão em lista nenhuma do Mk; e
as fontes instaladas no sistema. O `make-logo.py` e o `make-art.py` abrem as
fontes por caminho fixo em `/usr/share/fonts/TTF/` (a Fraunces variável, normal
e itálica, do `ttf-fraunces-variable` de `aur.packages`, e a Space Mono Nerd
Font, do `ttf-space-mono-nerd` de `base.packages`); o `make-wallpapers.py` pede
a Space Mono pelo nome da família.

```bash
python3 system/mk/art/make-logo.py
```

O logo em vetor, com as letras tiradas do contorno da própria fonte. Grava em
`system/mk/art/logo/` os SVG e PNG do logo inteiro (com e sem fundo) e da marca
compacta, e depois as cópias que o resto do Mk usa:

| Arquivo | Quem usa |
|---|---|
| `system/mk/art/mark.png` | o `make-art.py` |
| `shell/mare/assets/mark.png`, `mark.svg` | o Maré (ilha, bloqueio, Ajustes, boas-vindas) |
| `config/fastfetch/mark.png` | a ficha do terminal |
| `system/mk/sddm/mk/letra-m.png`, `letra-k.png`, `geometria.js` | a tela de login: as letras em camadas e a geometria da marca, para ela ser construída em movimento (`make-logo.py entrada` gera só estes) |
| `system/mk/plymouth/mk/logo.png` | o boot |

```bash
python3 system/mk/art/make-art.py
```

Gera `system/mk/art/emblem.png` (o desenho com arco, colunas e o sol de cobre)
e `grain.png` (o grão). Usa a marca feita pelo `make-logo.py`: rode aquele
antes. O `grain.png` também existe em `shell/mare/assets/` e em
`system/mk/sddm/mk/`; este script não atualiza essas cópias.

```bash
python3 system/mk/art/make-wallpapers.py
```

Os doze papéis de parede desenhados por código (`01-arcos` a `12-catedral`):
cada um é um SVG montado no script, renderizado em 2560x1440 e finalizado com
grão. Saem em `shell/mare/wallpapers/*.jpg`, que o painel de papel de parede
lista depois dos seus.

Na mesma pasta ficam os seis **autorais**, criados por quem mantém o Mk e
assinados "Mk_ by codexmark": Entropia (`00-1-entropia`), Núcleo
(`00-2-nucleo`), Origem (`00-3-origem`), Monólito (`00-4-monolito`),
Singularidade (`00-5-singularidade`) e Maré (`00-6-mare`). Não saem de script:
o arquivo no repositório é o original, em JPEG, sem metadados. O `00-` no nome
faz o painel mostrá-los antes dos desenhados por código, e o gerador não toca
neles. Estão em 1672x941; uma exportação maior entra no lugar com o mesmo nome. Com argumentos, refaz só os
que tiverem o texto no nome:

```bash
python3 system/mk/art/make-wallpapers.py duna catedral
```

### Boot, login e nome do sistema

**`system/mk/plymouth/mk/`** — o tema do boot: `mk.plymouth`, `mk.script` (o
logo no centro e uma linha que avança; com LUKS, a senha é pedida no mesmo
lugar) e as imagens. O instalador copia para `/usr/share/plymouth/themes/mk` e
grava `/etc/plymouth/plymouthd.conf`. O pedido de senha do LUKS nunca foi
visto funcionando: a instalação com disco criptografado não foi testada.

**`system/mk/sddm/mk/`** — o tema da tela de login, em QML (`Main.qml`,
`theme.conf`, `metadata.desktop`). O instalador copia para
`/usr/share/sddm/themes/mk`, grava o layout do teclado em `theme.conf.user` e
escolhe o tema em `/etc/sddm.conf.d/20-mk-theme.conf`.

À esquerda fica o relógio de anéis da tela de bloqueio (os minutos e os
segundos giram em volta de um ponto fora da tela; o número que passa pela
pílula acende em cobre), com a mesma conta do `shell/mare/LockSurface.qml`:
um teste reprova se as duas divergirem. À direita, a marca, com quem entra e
a senha embaixo dela. A entrada constrói a marca em vez de mostrá-la pronta: os eixos, as guias, as
diagonais, as letras e os nós do logo aparecem nessa ordem, em pouco mais de
dois segundos. Depois um cursor de cobre anda pela cota com os segundos, um nó
pisca de cada vez, cada tecla da senha acende o próximo e, enquanto a senha é
conferida, uma guia varre a marca. Só as cores do Mk. `motion=nao` no
`theme.conf.user` da máquina desliga tudo e mostra a marca pronta.

Para ver o tema sem sair da sessão:

```bash
sddm-greeter-qt6 --test-mode --theme system/mk/sddm/mk
```

**`system/mk/boot/limine-default`** — faz o menu do limine iniciar sozinho
depois da contagem, mesmo quando os ganchos do limine reescrevem o menu em
grupos. Roda como root e pode rodar quantas vezes quiser. Chamam: o
instalador, a migração `20261122-boot-entrada.sh` e o `mk-update`, depois de
instalar os pacotes do AUR que ficaram de fora do ISO.

**`system/mk/os-release`, `issue`, `mk-os-release.hook`** — o nome do sistema.
O pacote `filesystem` traz o `/etc/os-release` do Arch; o gancho do pacman põe
o do Mk de volta a cada instalação ou atualização dele.

**`system/mk/channel`** — o endereço do canal de atualização.

### Tema dos apps e fontes

**`system/mk/look/themes/Mk/`** — o tema GTK: `gtk-3.0/gtk.css` e
`gtk-4.0/gtk.css` importam o `adw-gtk3-dark` (pacote `adw-gtk-theme`) e só
trocam as cores e os cantos. Não há passo de build: edite o CSS.

**`system/mk/look/make-icons <destino>`** — gera o tema de ícones "Mk": herda
do Papirus-Dark e só tem atalhos (links simbólicos) para as pastas na cor
`palebrown` do Papirus. Não copia ícone nenhum, por isso não fica no git e
precisa do `papirus-icon-theme` instalado. Apaga e recria o destino.

```bash
sudo bash /usr/share/mk/system/mk/look/make-icons /usr/share/icons/Mk
```

**`system/mk/look/qt6ct/`** — `qt6ct.conf` e as cores `colors/Mk.conf`, para os
apps Qt.

**`system/mk/fonts/`** — a Space Grotesk (quatro pesos, `.otf`), a única fonte
que o Mk carrega no próprio repositório, com a licença dela ao lado
(`OFL.txt`, SIL Open Font License 1.1). O instalador instala a licença junto
com a fonte. As outras fontes vêm de pacotes: a Geist Mono e a Space Mono
(Nerd Font) de `base.packages`, a Fraunces de `aur.packages` (sem ela o Maré
usa a Noto Serif). Se acrescentar uma fonte aqui, traga o arquivo de licença
junto e instale os dois.

## Convenções

Só o que o código de hoje segue de fato.

- **Português.** Comentários, mensagens, nomes de migração e assuntos de commit.
  Todo script abre com um cabeçalho que diz o que faz e como se usa; o
  `iso/build --help` imprime o próprio cabeçalho.
- **`rm -rf` com `"${VAR:?}"`.** Se a variável estiver vazia, o comando falha em
  vez de apagar outra coisa. Exemplos: `rm -rf "${tmp:?}"` no `dev-shot`,
  `sudo rm -rf "${share:?}"` no `mk-update`. O código mais novo segue; ainda há
  `rm -rf "$var"` sem a guarda (em `new-user`, `try-session`, `diagnose-user`,
  `install-packages`, `mk-install`, `iso/build`). Em código novo, use a guarda.
- **Só se apaga o que o Mk criou.** As pastas que o Mk põe em `~/.config/mk`
  levam a marca `.instalado-pelo-mk`. Sem ela, `try-session`, `new-user` e
  `mk-update` não sobrescrevem nem apagam.
- **Root só onde precisa.** O `try-session` imprime os comandos com `sudo` em
  vez de rodar; o `iso/build` recusa rodar como root e só chama `sudo` no
  último passo, o `mkarchiso` (com `--syncdeps`, o `makepkg` também usa `sudo`
  antes, para instalar o que falta para compilar o AUR).
- **Idempotência.** Scripts e migrações podem rodar duas vezes com o mesmo
  resultado (`--needed`, `[[ -e ... ]] ||`, `grep -q ... ||`).
- **Troca de pasta de uma vez.** O Hyprland e o Maré vigiam os próprios
  arquivos. O `mk-update` copia a pasta nova ao lado (`.novo`) e troca com
  `mv`, para ninguém ver a pasta vazia no meio.
- **`pkill -f` com padrão que não casa com quem chama.** Há dois no
  repositório. Em `shell/mare/scripts/mare-cava` o padrão é `cava -p
  <arquivo de config>`, que a linha de comando do próprio script não contém.
  Em `shell/mare/Settings.qml` é `pkill -x -f "hypridle -c <config do Mk>"`,
  rodado por `sh -c`: o `-x` exige que a linha de comando inteira case, e a do
  `sh` não casa. Onde dá, o código prefere o nome exato (`pgrep -x` no
  `mk-update`) ou a sessão do processo (`pkill -s` no `mare-printers`).
- **Segredo fica em `$XDG_RUNTIME_DIR`, só para o dono.** O QR do Wi-Fi contém
  a senha: o `mare-wifi-qr` grava com `umask 077` e o painel de rede apaga o
  arquivo ao fechar. O `mare-vpn` faz o mesmo com o arquivo que importa.
- **Papéis de parede são do Mk.** Os doze que vão no repositório são desenhados
  por código (`make-wallpapers.py`). Os do usuário ficam na pasta de imagens
  dele, fora do repositório.
- **Dois lugares para a mesma lista, um script que confere.** Os atalhos
  (`binds.lua` e `Keybinds.js`) são o caso; o `check-keybinds` barra o
  lançamento se divergirem.
- **O que muda fora de `~/.config/mk` vai no instalador e numa migração.**
  Sempre os dois.

## Antes de lançar

O `mk-release` publica **tudo o que está rastreado** no git, inteiro, e os
assuntos dos commits. Não há "desfazer": o conserto é outro commit e outro
lançamento.

1. `git status` limpo. Arquivo novo sem `git add` não impede o lançamento, mas
   também não vai nele: confira que nada do que a mudança precisa ficou de
   fora.
2. `tests/run` passa inteiro, e a mudança chegou com o teste dela.
   `bin/check-keybinds` e `bin/check-shell` passam. (O `mk-release` roda os
   três de novo. Os testes e o `check-keybinds` precisam do `node`; o
   `check-shell` abre um bash interativo de verdade com o
   `config/bash/mk.bash` e falha se ele cair.)
3. Mexeu em `system/packages/`? `bin/check-packages` passa.
4. Mexeu no Maré? `bin/dev-shot` com os painéis e páginas tocados dá "log do
   Maré: limpo", e você viu a mudança numa sessão de verdade.
5. Mexeu em algo fora de `~/.config/mk` (em `/etc`, `/usr/share`, boot, tema,
   fontes, `~/.bashrc`)? Está no instalador **e** numa migração, com o nome
   `AAAAMMDD-assunto.sh`. Uma migração já lançada não se renomeia.
6. A migração precisa de pacote? Está na linha `# pacotes:` dela, e ela
   aguenta o pacote faltar.
7. Mexeu no instalador, no boot, no login ou numa migração? Passou por uma VM:
   instalação nova pelo ISO e, para migração, a atualização de um Mk instalado
   antes da mudança.
8. Mexeu no desenho da marca? Os geradores foram rodados e as imagens geradas
   estão no commit.
9. Nenhum arquivo rastreado traz senha, chave, e-mail ou caminho da sua home.
   Vale para a documentação também: ela vai junto.
10. Os assuntos dos commits desde o último lançamento servem para leitura
    pública? Eles viram o `NOVIDADES` que os Ajustes mostram. Se não servem,
    passe uma frase: `bin/mk-release "o que mudou"`.
11. A documentação diz o que o código faz agora: o README (atalhos, páginas dos
    Ajustes, comando `mk`) e os arquivos da tabela no alto deste.
12. `git push` para o repositório privado. O `mk-release` só avisa se faltar.

Depois de lançar, confira num Mk instalado que a versão nova aparece:

```bash
mk-update --check
```
