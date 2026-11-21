# Mk

Um desktop Arch opinativo, focado em desenvolvimento. Usa só os repositórios
oficiais do Arch, mais um punhado de pacotes do AUR.

A estrutura segue o modelo do Ryoku (listas separadas por função). O conteúdo
junta a base do Ryoku com o kit de desenvolvimento do Omarchy.

## Estrutura

```
system/packages/
  base.packages       sistema, boot, rede, áudio, energia, fontes      (53)
  desktop.packages    Hyprland, sessão, shell, launcher, captura       (29)
  dev.packages        terminal, CLIs, git, linguagens, containers      (57)
  apps.packages       apps gráficos do dia a dia, opcional             (11)
  hardware.packages   microcode e drivers por perfil [amd] [intel] [nvidia] [vm]
  aur.packages        só o que não existe nos oficiais                  (5)
bin/
  check-packages      confere cada lista contra os repositórios (não instala nada)
  install-packages    instala as listas num Arch já instalado
  try-session         instala a sessão do Mk no seu usuário
  new-user            cria um usuário novo já com o Mk (sudo)
  diagnose-user       junta, sem alterar nada, o que explica uma sessão que não abriu (sudo)
  mk-session      o lançador da sessão (vai para /usr/local/bin)
scripts/
  system/firewall.sh  ufw: nega entrada, libera saída, integra com Docker
  user/neovim.sh      aplica o LazyVim se não houver config do Neovim
  user/mise-tools.sh  node/python globais e CLIs de IA via mise
shell/mare/           o shell Maré (Quickshell/QML), veja abaixo
config/hypr/          Hyprland (Lua), bloqueio de tela e ociosidade
```

## Maré, o shell

A barra é uma linha fina na borda inferior, na cor do projeto em foco. Tudo
nasce dela: a pílula do centro cresce para cima e vira o painel do projeto,
volume, bateria, rede, calendário ou o seletor de projeto, sempre a mesma forma
se transformando.

- **Workspaces são projetos.** Ao entrar num workspace novo e vazio
  (`Super+5`, por exemplo), o seletor abre sozinho com os repositórios git de
  `~/Work`. Escolher um liga o workspace ao projeto (salvo em
  `~/.local/state/quickshell/.../workspaces.json`) e abre um terminal nele.
- **A cor vem do nome do projeto**: o mesmo projeto tem sempre a mesma cor.
- **Painel do projeto**: branch, arquivos modificados, containers do docker
  compose, portas abertas pelo projeto e versões do mise
  (`shell/mare/scripts/mare-project`, atualizado a cada 5 s).
- **Launcher** (`Super+Space` ou a lupa da barra): uma busca só para comandos
  do projeto em foco (lazygit, lazydocker, terminal), projetos de `~/Work` e
  apps. Escolher um projeto vai ao workspace dele ou cria um novo já ligado.
  ↑/↓ escolhe, Enter abre, Esc fecha; a forma cresce e encolhe com os resultados.
- **Alt+Tab**: cartões com miniatura ao vivo de cada janela, na ordem de uso,
  com a cor e o nome do projeto. Tab avança, soltar o Alt abre, Esc cancela.
- **Notificações**: o aviso nasce da linha e some em 5 s (10 s se for
  crítico); o sino abre o histórico, com "não perturbe" e "limpar tudo".
- **Bandeja do sistema**: ícones dos apps (Discord, Steam...) na cápsula da
  direita; clique abre, botão direito abre o menu do app.
- **Sessão** (botão ⏻ ou `Super+Esc`): bloquear, suspender, sair, reiniciar,
  desligar. Os três últimos pedem um segundo clique.
- **OSD**: volume e brilho pelas teclas viram uma barra de nível na pílula.
- **Tocando agora**: um chip ao lado dos indicadores, com barrinhas que dançam
  enquanto toca (em telas estreitas vira só o ícone 󰎈). O painel tem capa,
  progresso clicável, controles e troca entre players.
- **Bluetooth**: aparelhos pareados com bateria, clique conecta/desconecta,
  "procurar" encontra aparelhos novos para parear.
- **Tela**: brilho (arraste ou role o mouse no ícone) e luz noturna (`hyprsunset`).
- **Sistema**: CPU, memória, temperatura e rede com o último minuto em gráfico,
  mais o disco. O ícone na barra acende e mostra o motivo quando algo passa do
  limite (CPU ou memória > 90%, 90 °C, disco > 95%).
- **Papel de parede** (`Super+W` ou pelo launcher): o próprio Maré desenha o
  fundo, com crossfade ao trocar. As imagens vêm de `~/Pictures/Wallpapers`.
  O `matugen` tira a cor principal da imagem e a interface acompanha:
  superfícies com um leve tom dela e as cores dos projetos espalhadas pela roda
  a partir desse matiz, todas com o mesmo brilho.
- **Área de transferência** (`Super+Shift+V`): histórico do `cliphist` com
  busca; Enter copia de volta, Del apaga, imagens aparecem como "imagem png 1366×768".
- **Clima**: no painel do calendário (agora, mínima/máxima, sensação e as
  próximas horas) e ao lado do relógio, via Open-Meteo (sem conta nem chave).
  A cidade vem do fuso horário; para outra, escreva o nome em `~/.config/mare/city`.
- **Layout do teclado**: só aparece com mais de um layout (`kb_layout = "br,us"`);
  clique troca.
- **Movimento**: uma curva só (`cubic-bezier(.2, .9, .1, 1)`), forma em 240 ms,
  conteúdo entra 80 ms depois. Os tokens ficam em `Motion.qml`; cores e fontes
  em `Theme.qml`.

| Arquivo | O que é |
|---|---|
| `shell.qml`, `Monitor.qml` | uma barra e uma forma por monitor |
| `Bar.qml`, `WorkspaceChip.qml`, `IndicatorButton.qml` | a linha, os workspaces, os indicadores |
| `Tide.qml`, `RestPill.qml` | a forma que cresce e a pílula em repouso |
| `*Panel.qml` | projeto, volume, bateria, rede, calendário, seletor, launcher, Alt+Tab, notificações, sessão |
| `ToastView.qml`, `OsdView.qml` | estados passageiros: aviso de notificação e nível de volume/brilho |
| `Notifs.qml`, `TrayIcon.qml` | servidor de notificações e ícone da bandeja |
| `Media.qml`, `MediaChip.qml` | player em destaque e o chip da barra |
| `Display.qml`, `Stats.qml`, `Keyboard.qml` | brilho e luz noturna, leituras do sistema (`scripts/mare-stats`), layout do teclado |
| `Sparkline.qml` | gráfico de linha dos blocos do sistema |
| `Wallpaper.qml`, `Backdrop.qml`, `WallpaperPanel.qml` | papel de parede, a cor tirada dele (`scripts/mare-palette`) e o seletor |
| `Weather.qml`, `ClipboardPanel.qml` | clima (`scripts/mare-weather`) e histórico da área de transferência |
| `Mare.qml`, `Projects.qml` | estado, IPC, ligação workspace → projeto |

Para testar sem instalar (mostra uma segunda barra enquanto estiver rodando):

```bash
qs -p shell/mare
```

Instalado (veja "Experimentar"), o Maré fica em `~/.config/mk/shell` e a
sessão o inicia com `qs -p ~/.config/mk/shell`, gravando a saída em
`~/.cache/mk/mare.log`. Os atalhos abrem os painéis pelo IPC
(`qs -p ~/.config/mk/shell ipc call mare toggle calendar`). Ele não vai
para `~/.config/quickshell`: outro desktop (o Ryoku, por exemplo) recria essa
pasta e apaga o que não conhece.

O Maré substitui o `mako`: só um programa pode ser o servidor de
notificações por vez, e o Maré só assume quando nenhum outro estiver rodando.

## Hyprland

`config/hypr/` é a configuração completa do Hyprland, em Lua, com um arquivo
por assunto em `mare/`:

| Arquivo | O que faz |
|---|---|
| `look.lua` | bordas de 1 px, cantos de 10 px, sem blur, fundo grafite |
| `motion.lua` | a curva do Maré nas janelas (entram de 95%) e nos workspaces (deslizam) |
| `binds.lua` | atalhos (tabela abaixo) e teclas de mídia passando pelo OSD |
| `rules.lua` | diálogos e utilitários flutuam centralizados |
| `autostart.lua` | sobe o Maré, o hypridle, o polkit e o chaveiro |
| `hyprlock.conf`, `hypridle.conf` | bloqueio na linguagem do Maré; escurece em 4 min, bloqueia em 5, desliga a tela em 6, suspende em 30 |

`monitors.lua` e `user.lua`, se existirem ao lado de `hyprland.lua`, carregam
por último e são seus. Antes de instalar, valide:

```bash
Hyprland --verify-config -c "$PWD/config/hypr/hyprland.lua"
```

Os atalhos abaixo são os de fábrica. Dá para trocar cada um em Ajustes
(`Super+,`) › Atalhos: as trocas ficam em `~/.config/mk/atalhos.lua`, que o
`binds.lua` lê ao criar os atalhos, e valem na hora. A lista mora em
`config/hypr/mare/binds.lua` e é espelhada em `shell/mare/Keybinds.js`
(`bin/check-keybinds` confere os dois).

| Atalho | Ação |
|---|---|
| `Super+Espaço` | launcher |
| `Alt+Tab` / `Alt+Shift+Tab` | alternar janelas |
| `Super+1…9` / `Super+Shift+1…9` | ir para o workspace / mandar a janela para ele |
| `Super+Tab` | workspace anterior |
| `Super+Enter` / `Super+G` / `Super+D` / `Super+E` | terminal / lazygit / lazydocker / yazi no projeto |
| `Super+P` / `Super+C` / `Super+N` | painel do projeto / calendário / notificações |
| `Super+Shift+N` | não perturbe |
| `Super+Esc` / `Super+L` | sessão / bloquear |
| `Super+Shift+Enter` / `Super+Shift+Esc` | emergência: terminal / sair, mesmo sem o Maré |
| `Super+W` / `Super+Shift+V` | papel de parede / área de transferência |
| `Super+Q` / `Super+F` / `Super+V` | fechar / tela cheia / flutuar |
| `Super+setas` (+`Shift` move, +`Ctrl` redimensiona) | foco entre janelas |
| `Print` / `Shift+Print` | captura de área / tela inteira |

O plano de três níveis está completo. Configurações (cidade do clima, pasta de
papéis de parede, limites de alerta) ainda são arquivos ou constantes no código,
não uma tela de ajustes.

## Experimentar

O Mk roda como uma sessão separada, escolhida na tela de login, sem tocar
na config do desktop atual.

No seu próprio usuário:

```bash
bin/try-session install
```

Ele copia a config para `~/.config/mk/hypr` e o Maré para `~/.config/mk/shell` e
mostra os comandos com `sudo` que faltam (pacotes, o lançador
`/usr/local/bin/mk-session` e a entrada "Mk" no login).
`bin/try-session uninstall` desfaz.

Num usuário novo, para testar do zero sem misturar com o seu:

```bash
sudo bin/new-user mare --wallpapers
```

Cria o usuário, instala pacotes e sessão, copia a config para a home dele e
pede a senha. `--admin` dá acesso a `sudo`. Para apagar depois:
`sudo userdel -r mare`.

## Regras

- Um pacote por linha; `#` e linhas em branco são ignorados.
- Se existe em `core`/`extra`, vai numa lista oficial; o AUR é último recurso.
- Rode `bin/check-packages` depois de editar qualquer lista.
- Todo script é idempotente: rodar duas vezes dá o mesmo resultado.

## Uso

```bash
bin/check-packages
bin/install-packages amd nvidia     # perfis: amd, intel, nvidia, vm
scripts/system/firewall.sh
scripts/user/neovim.sh
scripts/user/mise-tools.sh
```

## De onde veio cada decisão

| Lista | Base | Principais mudanças |
|---|---|---|
| base | Ryoku `base.packages` | sem gaming, VMs, shell próprio do Ryoku e pacotes do repositório `[ryoku]`; com `man-db`, `ufw`, `avahi` do Omarchy |
| desktop | Hyprland puro | `waybar` + `fuzzel` + `mako` no lugar do shell Quickshell do Ryoku |
| dev | Omarchy + Ryoku `dev.packages` | adiciona `atuin`, `git-delta`, `difftastic`, `uv`, `pnpm`, `just`, `direnv`, `pgcli` |
| apps | Omarchy | sem kdenlive, obs, libreoffice, pinta, obsidian |

## Próximos passos

1. Configs padrão (`config/` com hypr, waybar, ghostty, starship) aplicadas sem
   sobrescrever o que o usuário já mudou.
2. Um meta-pacote (`PKGBUILD` com `depends=(...)` gerado das listas) e um
   repositório pacman próprio, assinado.
3. ISO com `archiso`, usando `base + desktop + dev` como lista do pacstrap.
