# Maré, por dentro

O shell do Mk: a ilha no alto da tela, os painéis que descem dela, o fundo, os
Ajustes, as boas-vindas e a tela de bloqueio. É Quickshell/QML numa pasta só,
sem subpastas de código: cada arquivo `.qml` com letra maiúscula é um tipo que
os outros usam pelo nome.

Este texto é para quem vai mexer no código. Para usar o Mk, veja o `README.md`
da raiz.

## Como ele sobe

O autostart do Hyprland (`config/hypr/mare/autostart.lua`) chama
`scripts/mare-start`. O script lê `"ui.scale"` de `<raiz>/ajustes.json`
(`<raiz>` é a pasta acima do shell, normalmente `~/.config/mk`; sem valor
válido, 0.85) e roda:

```
QT_SCALE_FACTOR=<escala> qs -p <pasta do shell>
```

A saída vai para `~/.cache/mk/mare.log` (ou `$XDG_CACHE_HOME/mk/mare.log`).

O tamanho da interface é a escala do Qt, então só muda num Maré novo: ao trocar
`ui.scale`, o `Settings.qml` chama `mare-start --restart` sozinho. À mão:

```
bash ~/.config/mk/shell/scripts/mare-start --restart
```

`shell.qml` é a raiz. Ele cria:

- um `Monitor` por tela (`Variants` sobre `Quickshell.screens`);
- uma `LockScreen` para todas as telas;
- uma referência ao `BatteryWatch`, só para o singleton ser carregado (um
  singleton que ninguém cita não existe).

`Monitor.qml` monta, para a sua tela, `Backdrop`, `Bar`, `Tide`,
`SettingsWindow` e `WelcomeWindow`, mais uma janela transparente
(`mare-outside`) que fecha o painel aberto quando você clica fora dele e da
ilha. É também ali que a ilha e o painel de cada tela são ligados
(`bar.islandX`/`islandW` → `Tide`, `tide.sheetW` → `Bar`). `Tide`, `SettingsWindow` e
`WelcomeWindow` recebem `active: focused`: os painéis só abrem no monitor em
foco. Com mais de uma tela isso nunca foi testado em hardware.

Instalado, o shell fica em `~/.config/mk/shell`, e não em
`~/.config/quickshell`: outro desktop pode recriar aquela pasta.

## As peças

| Arquivo | Camada (namespace) | O que é |
|---|---|---|
| `Backdrop.qml` | Background (`mare-backdrop`) | O fundo: a folha de desenho, a letra do workspace e, por cima, o papel de parede escurecido. |
| `Bar.qml` | padrão (`mare-island`) | A ilha. Reserva 34 px (`barH`) para as janelas; a janela é mais alta para caber a onda do som. Expõe `islandX`/`islandW` (para onde ela vai) e recebe `sheetW`; abre painéis com `Mare.openAt(painel, x)`. |
| `Tide.qml` | Overlay (`mare-tide`) | O painel líquido que desce da ilha. Carrega o conteúdo do painel em `Mare.mode`. |
| `SettingsWindow.qml` | Overlay (`mare-settings`) | Os Ajustes: uma folha no meio da tela, não um painel da ilha. Aberta quando `Mare.mode === "settings"`. |
| `PolkitDialog.qml` | Overlay (`mare-polkit`) | O pedido de senha de administrador: o Maré é o agente de polkit da sessão (só um registra; o `hyprpolkitagent` fica instalado, de reserva). Cancela o pedido se a tela estiver bloqueada. |
| `WelcomeWindow.qml` | Overlay (`mare-welcome`) | As boas-vindas do primeiro login. Quem decide quando aparece é o `Welcome.qml`. |
| `LockScreen.qml` | bloqueio de sessão | `WlSessionLock` ligado a `Lock.locked`; cada tela ganha uma `LockSurface`. |

Peças pequenas usadas pelos painéis: `PanelHeader`, `PillButton`, `MkSlider`,
`MkToggle`, `Sparkline`, `IslandItem`, `TrayIcon`, `NotifIcon`.

### O estado: `Mare.mode`

Tudo gira em torno de uma string no singleton `Mare`:

- `"rest"`: só a ilha;
- um nome da lista `Mare.panels`: aquele painel está aberto;
- `"toast"` ou `"osd"` (`Mare.transients`): passageiros, voltam sozinhos para
  `"rest"` e não atropelam um painel aberto (`Mare.flash`).

`Mare.toggle`, `open`, `openAt` e `close` mudam o modo. `toggle`, `open` e
`openAt` ignoram um nome que não está em `panels`.

Três nomes da lista não seguem o caminho comum:

- `launcher` fica sempre carregado no `Tide` (o `Loader` `kept`), num canto de
  1 px, e só muda de lugar ao abrir. Não entra no `switch`.
- `settings` é desenhado pela `SettingsWindow`; para o `Tide`, é `"rest"`.
- `picker` não recebe teclado nem prende o foco.

Além disso, `project` só abre num workspace ligado a um projeto e que não
esteja marcado como livre (`Projects.free`). Fora disso o `Mare.mode` até vira
`"project"` (é o que o `state` devolve), mas o `Tide` o trata como `"rest"`
e nada aparece.

### Registrar um painel novo

São três lugares, mais o arquivo do painel:

1. Crie `MeuPanel.qml`. A raiz precisa de `implicitWidth` e `implicitHeight`:
   é deles que o `Tide` tira o tamanho da forma.
2. Em `Mare.qml`, ponha o nome na lista `panels`.
3. Em `Tide.qml`, acrescente o `case` no `switch` do `Loader` `content`:

   ```qml
   case "meu": return meuPanel;
   ```

4. Ainda em `Tide.qml`, no fim do arquivo, declare o componente:

   ```qml
   Component { id: meuPanel; MeuPanel {} }
   ```

Pronto: `ipc call mare toggle meu` já abre. Para abrir por tecla, acrescente a
ação em `config/hypr/mare/binds.lua` **e** em `Keybinds.js` (veja "Atalhos").
Para abrir por um item da ilha, chame `Mare.openAt("meu", x)` no `Bar.qml`.

O Esc fecha qualquer painel (`Keys.onEscapePressed` no `Tide`). Um painel com
campo de texto trata o Esc ele mesmo, antes.

## Singletons

Cada arquivo com `pragma Singleton` é dono de um pedaço do estado.

| Singleton | Estado que guarda |
|---|---|
| `Mare` | O modo da ilha (`mode`), a posição de onde o painel desce (`originX`), o índice do Alt+Tab, o OSD, a página dos Ajustes, `uiScale`/`ignoreZone`. Tem o IPC. |
| `Settings` | Os ajustes (`values` sobre `defaults`), o que a máquina tem (`caps`), o último erro (`lastError`) e os arquivos gerados. Veja "Ajustes". |
| `Theme` | Cores, famílias de fonte e a letra grega de cada workspace. Só leitura. |
| `Motion` | Durações e curvas, multiplicadas por `motion.speed`. Só leitura. |
| `Hypr` | Nenhum estado: é o único ponto que manda `dispatch` ao Hyprland (`exec`, `workspace`, `focusWindow`), no formato Lua ou no antigo conforme a sessão. |
| `Projects` | A ligação workspace → projeto (`bindings`, gravada no `workspaces.json` do estado do Quickshell), os repositórios em `~/Work` e o estado do projeto em foco (`info`). |
| `Notifs` | O servidor de notificações: a lista, o não perturbe (`dnd`) e o aviso na tela (`toastItem`). |
| `Lock` | A tela de bloqueio: `locked`, a senha digitada, a conversa com o PAM e os números da animação. |
| `Welcome` | As boas-vindas: `shown`, `step`, `paused` e o "já vi" (`<raiz>/.bem-vindo`). |
| `Wallpaper` | O papel de parede em uso, a cor tirada dele (`primary`, `hue`) e a lista de imagens. Tem um IPC próprio. |
| `Display` | Brilho (`brightnessctl`) e luz noturna (`hyprsunset`), com o horário automático. |
| `Monitors` | As telas para a página Monitores: o que o Hyprland tem (`live`), o rascunho (`draft`) e a contagem de 15 s depois de aplicar. Gera o texto do `<raiz>/monitores.lua`; quem grava é o `scripts/mare-monitors trial`. |
| `Keyboard` | Os layouts do teclado principal e qual está ativo. |
| `Media` | O player MPRIS em destaque. |
| `Sound` | O espectro do som que está tocando (`levels`, `energy`), para a onda da ilha. |
| `Stats` | CPU, memória, temperatura, rede e disco a cada 2 s, com o histórico dos gráficos e o `alert`. |
| `Weather` | O clima (`data`), renovado a cada 30 min. |
| `Updates` | O resultado da última procura por atualizações (`info`) e se a ilha deve acender (`badge`). |
| `Vpn` | Há túnel aberto e qual (`status`). |
| `Capture` | A gravação da tela em andamento (`recording`, `seconds`) e se dá para gravar (`available`, `reason`). |
| `BluetoothPair` | O pareamento em andamento, que precisa sobreviver ao painel fechado. |
| `BatteryWatch` | Os avisos de bateria fraca já dados (20% e 8%). |

## IPC

Os atalhos do Hyprland, o `hypridle` e o comando `mk` falam com o Maré por
aqui. O alvo é `mare`, declarado no `IpcHandler` do `Mare.qml`.

| Função | Argumentos | O que faz |
|---|---|---|
| `toggle` | `panel` | Abre o painel; se já é ele que está aberto, fecha. |
| `toggleAt` | `panel`, `x` | O mesmo, descendo embaixo da posição `x` da tela, como um clique na ilha. |
| `close` | | Volta para `rest`. |
| `state` | | Devolve o modo atual. |
| `settings` | `page` | Abre os Ajustes na página dada; `""` abre na última. |
| `set` | `key`, `value` | Muda um ajuste, como o painel faria. Devolve `chave = valor`, ou `ajuste desconhecido: ...`. |
| `switcher` | `direction` | Alt+Tab: `prev` volta; qualquer outro valor avança. |
| `shot` | `mode` | Captura sem abrir painel: `copiar`, `area`, `janela` ou `tela`. |
| `record` | | Começa ou para a gravação da tela. |
| `lock` | | Bloqueia a tela. |
| `welcome` | `action` | `abrir`, `continuar`, `fechar`, `fim`, `seguir`, `voltar` ou `estado`. Devolve o estado da folha. |
| `here` | `cmd` | Roda um programa no diretório do projeto em foco. |
| `volume` | `change` | `+5`, `-5` (em %) ou `mute`. |
| `brightness` | `change` | `+5` ou `-5` (em %). |
| `dnd` | | Liga ou desliga o não perturbe; devolve o novo valor. |

Os painéis que `toggle` aceita são os de `Mare.panels`: `project`, `volume`,
`power`, `network`, `calendar`, `picker`, `launcher`, `switcher`,
`notifications`, `session`, `media`, `bluetooth`, `display`, `system`,
`clipboard`, `wallpaper`, `settings`, `capture`, `emoji`.

Um exemplo de cada:

```
qs -p ~/.config/mk/shell ipc call mare toggle calendar
```

```
qs -p ~/.config/mk/shell ipc call mare toggleAt volume 800
```

```
qs -p ~/.config/mk/shell ipc call mare close
```

```
qs -p ~/.config/mk/shell ipc call mare state
```

```
qs -p ~/.config/mk/shell ipc call mare settings janelas
```

```
qs -p ~/.config/mk/shell ipc call mare set hypr.gapsIn 10
```

```
qs -p ~/.config/mk/shell ipc call mare switcher next
```

```
qs -p ~/.config/mk/shell ipc call mare shot area
```

```
qs -p ~/.config/mk/shell ipc call mare record
```

```
qs -p ~/.config/mk/shell ipc call mare lock
```

```
qs -p ~/.config/mk/shell ipc call mare welcome estado
```

```
qs -p ~/.config/mk/shell ipc call mare here lazygit
```

```
qs -p ~/.config/mk/shell ipc call mare volume +5
```

```
qs -p ~/.config/mk/shell ipc call mare brightness -5
```

```
qs -p ~/.config/mk/shell ipc call mare dnd
```

Sobre o `set`: o valor passa por `JSON.parse` (`10`, `false`, `0.85`); o que
não for JSON entra como texto (`set hypr.kbLayout us`). Só aceita chaves que
têm valor de fábrica no `SettingsSchema.js`. Não confere faixa nem opções: um
valor fora do `min`/`max` da linha é gravado como veio. Com o `ajustes.json`
ilegível (`Settings.broken`), o `set` não muda nada e ainda assim devolve
`chave = valor`, com o valor antigo.

Há um segundo alvo, `wallpaper`, no `Wallpaper.qml`: `set <caminho>`, `random`
e `get`.

```
qs -p ~/.config/mk/shell ipc call wallpaper random
```

## A pasta `scripts/`

O QML não lê `/proc` nem conversa com `nmcli`: chama um script, que responde em
JSON (ou em linhas). Nenhum usa sudo; só o `mare-printers` pede a senha de
administrador, pelo `pkexec`, quando o CUPS recusa uma ação.

| Script | O que faz | Quem chama |
|---|---|---|
| `mare-start` | Inicia o Maré com a escala dos Ajustes; `--restart` fecha o que está rodando, inicia outro e devolve o teclado à janela em uso. | `autostart.lua`, `Settings.qml` (ao trocar `ui.scale`), `bin/mk-update`, `mk reiniciar-mare` |
| `mare-look` | Aplica ícones, cursor e fontes por `gsettings`, sem desfazer a escolha de outro desktop. | `autostart.lua` |
| `mare-caps` | Imprime `{"touchpad", "bateria", "brilho"}`: o que a máquina tem. | `Settings.qml`, uma vez ao subir |
| `mare-about` | JSON com os detalhes da máquina. | `SettingsWindow.qml` (página Sobre) |
| `mare-stats` | Leituras cruas de CPU, memória, rede, temperatura e disco da raiz, uma por linha. | `Stats.qml`, a cada 2 s |
| `mare-project` | JSON do projeto: branch, arquivos mexidos, runtime, containers, portas. | `Projects.qml`, a cada 5 s |
| `mare-weather` | JSON do clima, via Open-Meteo. A cidade vem de `~/.config/mare/city` ou do fuso horário. | `Weather.qml`, a cada 30 min |
| `mare-cava` | Roda o `cava` e imprime uma linha de faixas (0 a 100) por quadro. | `Sound.qml` |
| `mare-palette` | Imprime a cor principal de uma imagem. | `Wallpaper.qml` |
| `mare-battery` | JSON com ciclos, saúde e limite de carga da bateria. | `PowerPanel.qml` |
| `mare-net` | JSON da conexão em uso: interface, IP, gateway, DNS. | `NetworkPanel.qml` |
| `mare-dns` | Troca o DNS da conexão em uso: `auto`, `cloudflare` ou `google`. | `NetworkPanel.qml` |
| `mare-wifi` | Redes Wi-Fi guardadas: `list-saved`, `forget`, `autoconnect`. | `NetworkPanel.qml` |
| `mare-wifi-qr` | Gera o PNG do QR code do Wi-Fi atual e imprime o nome da rede. O PNG contém a senha. | `NetworkPanel.qml` |
| `mare-vpn` | VPNs do NetworkManager: `list`, `status`, `up`, `down`, `delete`, `autoconnect`, `check`, `import`, `ask`, `watch`. | `Vpn.qml` (`status`, `watch`), `SettingsVpnPage.qml` |
| `mare-shot` | Captura de tela (`copiar`, `area`, `janela`, `tela`); `pasta` e `gravacao` só respondem. | `Capture.qml` |
| `mare-emoji` | `pick` copia (e digita, com `wtype`) o emoji; `tem-wtype`; `gerar` refaz o `Emoji.js`. | `EmojiPanel.qml`; `gerar` é para quem mantém o Mk |
| `mare-monitors` | `list` lê as telas; `trial` grava o `monitores.lua`, recarrega o Hyprland e deixa um vigia; `keep` e `revert` respondem ao vigia. | `Monitors.qml` |
| `mare-default-apps` | Apps padrão: `list`, `set`, `terminal`, `open`. | `SettingsAppsPage.qml`; `binds.lua` (`open navegador`) |
| `mare-sleep` | Como a máquina pode dormir: `status` (JSON: suspender e hibernar permitidos, hibernação pronta ou pendente de reinício, placas com defeito conhecido ao voltar da suspensão); `now` dorme do jeito escolhido em `power.sleep`. | `mare-caps`, `config/hypr/hypridle.conf` |
| `mare-devtools` | Roda o `scripts/user/mise-tools.sh` num terminal e espera o Enter. | `Welcome.qml`, `mk ferramentas` |
| `mare-gamepad` | O controle Xbox e o driver xone: `status` (JSON), `install` e `remove` (interativos, num terminal). | `SettingsGamepadPage.qml` |
| `mare-printers` | Impressoras do CUPS: `status`, `find`, `add` e as outras ações da página. | `SettingsPrintersPage.qml` |
| `mare-updates` | Procura atualizações do Mk, do Arch e do AUR e responde em JSON. | `Updates.qml`, `bin/mk-update` |

O cabeçalho de cada script traz o formato exato da saída. Do QML, chame sempre
por `Quickshell.shellPath("scripts/mare-...")`, com `bash` na frente.

Nunca testados em hardware de verdade: impressoras reais (`mare-printers`),
importar VPN (`mare-vpn import`), parear Bluetooth (`BluetoothPair.qml`) e mais
de um monitor (`mare-monitors`).

## Ajustes

### Onde os valores ficam

Tudo fica em `<raiz>` (a pasta acima do shell, normalmente `~/.config/mk`),
fora das pastas que a atualização troca.

| Arquivo | Quem escreve | O que tem |
|---|---|---|
| `ajustes.json` | `Settings.qml` | Só o que saiu da fábrica, como `{ "chave": valor }`. É a fonte. |
| `ajustes.lua` | `Settings.qml` | O Lua das linhas com `hypr`. O `hyprland.lua` carrega no login. Gerado: não edite. |
| `hypridle.conf` | `Settings.qml` | Os tempos da página Energia (chaves `idle.*`). Gerado. |
| `atalhos.lua` | `SettingsKeybindsPage.qml` | As trocas de atalho. O `binds.lua` lê. |
| `monitores.lua` | `scripts/mare-monitors trial`, com o texto que o `Monitors.qml` gera | Uma regra `hl.monitor` por saída. Carregado depois do `ajustes.lua`. |
| `.bem-vindo` | `Welcome.qml` | Existe = as boas-vindas já foram vistas. |

`Settings.get(chave)` devolve o valor gravado ou o de fábrica.
`Settings.set(chave, valor)` grava (400 ms depois do último toque, para um
slider arrastado não escrever a cada pixel); um valor igual ao de fábrica some
do arquivo.

Numa linha com `hypr`, o `set` também manda o Lua ao Hyprland com `hyprctl
eval`. O `ajustes.lua` só é regravado se o Hyprland aceitar. Se recusar, o
valor volta ao que era e o motivo aparece em `Settings.lastError`.

Um `ajustes.json` ilegível não é sobrescrito: fica onde está, uma cópia vai
para `ajustes.json.quebrado`, `Settings.broken` fica verdadeiro e nada é
gravado até a pessoa escolher começar do zero (aí o `ajustes.json` é regravado
e a cópia fica).

### Uma linha

As páginas são dados. Em `SettingsSchema.js`, `groups` lista as páginas,
`blurbs` tem o texto de cada uma e `rows` tem as linhas. O painel desenha todas
do mesmo jeito e a busca procura em todas.

| Campo | Para quê |
|---|---|
| `page` | A chave da página (como em `groups`). |
| `group` | O título do bloco dentro da página, em maiúsculas. |
| `key` | Onde o valor fica no `ajustes.json`. |
| `label`, `desc` | O nome e a linha de explicação. |
| `ctl` | `"switch"`, `"slider"` ou `"choice"`. |
| `def` | O valor de fábrica. Tem de ser o que o Mk usa sem `ajustes.json`. |
| `min`, `max`, `step`, `unit` | Para `slider`. Unidades que o painel formata: `%`, `px`, `ms`, `/s`, `s`, `K`, `h`, `min` (onde 0 vira "nunca"). |
| `options` | Para `choice`: `[ { v: valor, label: "texto" } ]`. |
| `hypr` | O campo do Hyprland que a linha muda, como `"general.gaps_in"`. Sem ele, a linha é só do Maré. |
| `kw` | Palavras para a busca que o texto não usa. |
| `quando` | `{ "outra.chave": valor ou [valores] }`: a linha só aparece assim. |
| `precisa` | `"touchpad"`, `"bateria"` ou `"brilho"`: a linha some se o `mare-caps` disser que a máquina não tem. |

Um `hypr` que começa com `@` (`@animations`, `@kb_options`, `@monitor.scale`)
não vira um campo de `hl.config` direto: entra no Lua por código próprio em
`hyprCode`, `hyprTable` e `animationCode`. Se criar outro, escreva esse código.

Chaves sem linha desenhada (as que uma página própria ou um painel gravam)
ficam em `hidden`, só com o valor de fábrica: hoje `apps.terminal`,
`emoji.recent` e `emoji.tone`. Sem isso o `ipc call mare set` recusa a chave.

### Acrescentar uma linha

1. Ponha o objeto em `rows`, junto das outras da página:

   ```js
   { page: "ilha", group: "CONTEÚDO", key: "island.showAlgo", label: "Algo",
     desc: "Uma frase curta", ctl: "switch", def: true, kw: "sinônimos" },
   ```

2. Leia o valor onde ele importa:

   ```qml
   visible: Settings.get("island.showAlgo")
   ```

Não há terceiro passo: a linha aparece na página, entra na busca, ganha a
marca de "mudado" e entra no voltar à fábrica da página (`Settings.reset`). Com `hypr: "secao.campo"`, o valor também vai para o
Hyprland e para o `ajustes.lua`, sem mais código.

As animações em `SettingsSchema.js` (`animations`) repetem as de
`config/hypr/mare/motion.lua`. Mudou lá, mude aqui.

### Acrescentar uma página

Uma página só de linhas:

1. Em `groups`, acrescente `{ key: "minha", name: "Minha página", num: "17" }`
   ao grupo certo.
2. Em `blurbs`, a frase que fica no alto dela.
3. As linhas em `rows`, com `page: "minha"`.

Uma página com conteúdo próprio (uma lista, um editor, um mapa das telas):

1. Os passos 1 e 2 acima.
2. Crie `SettingsMinhaPage.qml`. A raiz recebe a largura da folha e pede a
   altura que o conteúdo precisar (as que existem são `ColumnLayout`).
3. Em `SettingsWindow.qml`, acrescente a página a `pageFiles`:

   ```qml
   "minha": "SettingsMinhaPage",
   ```

O `Loader` `pageLoader` carrega `<nome>.qml` só enquanto a página está aberta.
Se a raiz declarar `property string flashKey`, ela recebe a chave que a busca
pediu, para destacar a linha (veja `SettingsKeybindsPage.qml`).

Uma página própria pode ter linhas em `rows` também: elas são desenhadas antes
do conteúdo do arquivo. Para guardar um valor sem linha, ponha a chave em
`hidden`.

Para abrir direto nela:

```
qs -p ~/.config/mk/shell ipc call mare settings minha
```

## Atalhos

A lista de atalhos existe em dois lugares, e os dois têm de ficar iguais:

- `config/hypr/mare/binds.lua`: a tabela `actions`, com o `run` (o que o
  atalho faz). É quem cria os `hl.bind`.
- `shell/mare/Keybinds.js`: a mesma tabela sem o `run`, para a página Atalhos,
  a busca dos Ajustes e as boas-vindas.

Mesmos `id`, na mesma ordem, com o mesmo `group`, `desc`, `combo` e marcas
(`required`, `fixed`, `family`), as mesmas `families` e a mesma lista
`everywhere`. Para conferir, na raiz do repositório:

```
bin/check-keybinds
```

Precisa do `node`. Sai com 1 e diz o que diverge.

O que o usuário troca fica no `ajustes.json`, na chave `keybinds`, como
`{ "id": "COMBO" }` (`""` desliga o atalho), e é gravado em
`<raiz>/atalhos.lua`. Esse arquivo é só dados: o `binds.lua` o lê ao criar os
atalhos, e o `hyprland.lua` não o carrega. Na sessão aberta, a página aplica
com `hyprctl eval 'require("mare.binds").apply({...})'` e depois confere com
`hyprctl -j binds` se o Hyprland ficou com o que foi pedido. Se o `eval` não
responde `ok` (uma sessão cuja config ainda não tem o `apply()`), a página roda
`hyprctl reload`; se isso também falhar, o estado fica "gravado; vale no
próximo login".

## Cores e movimento

`Theme.qml` tem as cores (superfícies de `ground` a `raised`, linhas
`hairline`/`line`/`wash`, texto de `text` a `textFaint`, e `copper`, o único
destaque), as famílias (`serif`, `sans`, `mono`, `icons`, cada uma com uma
reserva se a fonte do Mk faltar) e `wsName(id)`, a letra grega do workspace.
Não escreva cor nem nome de fonte direto num painel.

`Motion.qml` tem as durações (`fast`, `base`, `slow`, `enter`, `exit`,
`spatial`, `content`) e as curvas (`curve`, `enterCurve`, `exitCurve`,
`spatialCurve`). Todas as durações são multiplicadas por `motion.speed` dos
Ajustes; com "sem" (0), viram zero. Use-as em vez de números soltos, senão a
escolha de velocidade não vale para o que você fez.

## Testar sem mexer na sua tela

`bin/dev-shot` sobe um Hyprland aninhado, roda nele uma **cópia** do shell,
manda os comandos de IPC que você pedir, tira uma captura e mostra os avisos do
log.

```
bin/dev-shot <saida.png> "settings monitores"
```

```
bin/dev-shot <saida.png> "toggle launcher" "close" "toggleAt volume 800"
```

Cada argumento depois do arquivo é o que viria depois de `ipc call mare`.

Antes, esconda a janela do Hyprland aninhado com uma regra no seu Hyprland
(some no próximo reload):

```
hyprctl eval 'hl.window_rule({ match = { class = "^aquamarine$" }, workspace = "special:mkdev silent" })'
```

O que saber:

- Sai com 1 se o log do Maré tiver erro de QML, e os lista. Sai com 2 se o
  Hyprland aninhado não subir.
- Com a janela escondida, o compositor não desenha quadros novos e a captura
  não sai ("captura: indisponível"). O log ainda vale. Para ver a imagem, deixe
  a janela visível.
- Como o shell é uma cópia numa pasta temporária, os ajustes mexidos no teste
  ficam nela e somem no fim: o seu `ajustes.json` não é tocado.
- Roda sempre com `QT_SCALE_FACTOR=0.85`.
- As boas-vindas não aparecem, a não ser com `MK_DEV_WELCOME=1`.

| Variável | Para quê | Padrão |
|---|---|---|
| `MK_DEV_SHELL` | A pasta do shell a copiar | `shell/mare` do repositório |
| `MK_DEV_WAIT` | Segundos de espera depois de cada comando | `1.5` |
| `MK_DEV_SIZE` | Tamanho da tela aninhada | `1600x900` |
| `MK_DEV_WELCOME` | Qualquer valor mostra as boas-vindas | vazio |

Rodar `qs -p shell/mare` direto do repositório, dentro da sua sessão, não é um
teste isolado: ele é um segundo Maré na sua tela, e a `<raiz>` dele passa a ser
`shell/`, onde o `ajustes.json` seria gravado.

## Armadilhas

**Arquivo de singleton novo pede um Maré novo.** O Quickshell recarrega o QML
quando um arquivo muda, mas esse recarregamento pode não registrar um arquivo
novo com `pragma Singleton` (observação de uso; o código não anota isso).
Depois de criar um, reinicie
(`scripts/mare-start --restart`) ou teste com o `bin/dev-shot`, que sempre sobe
do zero. E lembre: um singleton só é carregado quando alguém o cita (veja o
`BatteryWatch` em `shell.qml`).

**Zona exclusiva com a interface fora de 100%.** Para uma janela ignorar o
espaço reservado da ilha (ficar por cima ou por trás dela), o protocolo pede a
zona exclusiva -1. Com `QT_SCALE_FACTOR`, o valor é multiplicado pela escala e
-1 virava 0: os painéis desciam e o fundo parava antes do alto. Use sempre o
par abaixo, como `Tide`, `Backdrop` e `SettingsWindow` fazem:

```qml
exclusionMode: ExclusionMode.Normal
exclusiveZone: Mare.ignoreZone
```

**Teclado depois de reiniciar o Maré.** Quando as camadas do Maré renascem, a
janela em uso fica sem o teclado até um clique. Por isso o `mare-start
--restart` devolve o foco à janela ativa duas vezes, pouco depois de subir. Se
você reiniciar o Maré de outro jeito (`qs kill` e `qs -p` à mão), isso não
acontece.

**Teclado nos painéis.** O painel pede o teclado ele mesmo: o `Tide` fica em
`WlrKeyboardFocus.Exclusive` enquanto um painel está aberto no seu monitor
(`tide.open && Mare.wantsKeyboard`), e os Ajustes, enquanto a folha está
aberta. Antes quem entregava era um `HyprlandFocusGrab`, mas o Hyprland dá o
foco a uma janela qualquer da lista do grab, e quando caía na ilha o que se
digitava não chegava a ninguém.

**Nada de geometria de tela no `Mare`.** O singleton é um só para todas as
telas; a `Bar` e o `Tide` são um por tela. Posição e largura da ilha e do
painel moram no `Monitor.qml`. Com isso no singleton, em telas de larguras
diferentes o painel abria com a posição da ilha da outra tela.

**O primeiro quadro do painel.** O conteúdo leva um instante para carregar, e
até lá a largura-alvo é 0. O `Tide` só desenha quando o conteúdo tem tamanho
(`sheet.loaded`) e nasce de uma gota pequena, não de um fio: numa máquina
lenta o fio ficava parado na tela.

**Comandos ao Hyprland.** A config é Lua, e o `dispatch` espera código Lua
(`hl.dsp...`), não o formato antigo. Do QML, use `Hypr.exec`, `Hypr.workspace`
e `Hypr.focusWindow` em vez de chamar `Hyprland.dispatch` direto.

**`shellPath` não sobe da pasta do shell.** `Quickshell.shellPath("../x")`
volta como caminho relativo. Para chegar à `<raiz>`, use `Settings.dir`.
