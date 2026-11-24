# Mk

Um desktop Arch opinativo, focado em desenvolvimento. Usa só os repositórios
oficiais do Arch, mais um punhado de pacotes do AUR.

É feito de:

- **Hyprland**, com a configuração em Lua (`config/hypr`), um arquivo por assunto;
- **Maré**, o shell em Quickshell/QML (`shell/mare`): a ilha no alto da tela, os
  painéis que descem dela, a busca, as notificações, a tela de bloqueio, o papel
  de parede e os **Ajustes**;
- um **instalador** (`installer/mk-install`) e um **ISO** live (`iso/`): disco
  inteiro, btrfs com snapshots do snapper, boot pelo limine;
- um **atualizador** (`bin/mk-update`), que traz o próprio Mk de um canal git
  público, roda as migrações e atualiza o Arch e o AUR;
- a **cara do Mk** (`system/mk`): boot, tela de login, tema GTK, ícones, fontes;
- as **listas de pacotes** (`system/packages`), separadas por função.

A estrutura segue o modelo do Ryoku (listas separadas por função). O conteúdo
junta a base do Ryoku com o kit de desenvolvimento do Omarchy.

## Estado do projeto

É o desktop de uma pessoa, em uso e mudando rápido. Leia isto antes de instalar
numa máquina que importa.

**Funciona e é usado:** a sessão (Hyprland + Maré), a ilha e os painéis, a
busca, os Ajustes, a tela de bloqueio, o terminal do Mk e a atualização pelo
canal público.

**Testado só em VM (QEMU, UEFI, sem LUKS, perfil de drivers `vm`, rede por
cabo):** o ISO, o instalador, o boot pelo limine, os snapshots e a atualização
de um Mk instalado.

**Em máquina de verdade:** uma instalação pelo ISO, que deu boot e está em uso.
Os defeitos que ela mostrou estão nas issues do repositório.

**Nunca testado:**

- no instalador: disco criptografado (LUKS), o perfil `nvidia` e a escolha de
  rede Wi-Fi;
- em hardware de verdade: mais de um monitor, impressoras, importar VPN e
  parear aparelho Bluetooth. As páginas e painéis existem, mas ninguém os viu
  funcionar fora da VM.

A lista completa, com o que falta para cada item sair dela, está em
[docs/limites.md](docs/limites.md).

**Testes automáticos:** `tests/run` confere a lógica do Mk inteiro (scripts,
Ajustes, configuração, instalador, migrações, atualização) sem tocar na
máquina, e roda antes de cada `git push` e de cada lançamento. O projeto segue
teste primeiro: veja [tests/README.md](tests/README.md).

**Não existe:** instalação em BIOS legado (o instalador para se o live não foi
iniciado em UEFI), dual boot (o instalador usa o disco inteiro) e discos com
menos de 24 GiB.

## Instalar pelo ISO

Gerar o ISO e testar numa VM está em [iso/README.md](iso/README.md). Com o ISO
num pendrive, inicie por ele **em modo UEFI**. O live abre num console; rode:

```bash
mk-install
```

Precisa de internet: os pacotes vêm dos espelhos do Arch. Leva uns 15 minutos.
O que aparece na tela:

1. **Internet** — só se não houver conexão: lista as redes Wi-Fi e pede a senha.
2. **Disco** — em qual disco instalar. O Mk usa o disco inteiro; o pendrive do
   live não aparece na lista.
3. **Você** — nome do computador (padrão `mk`), usuário e senha (duas vezes). A
   senha também vale para o `sudo`; o root fica bloqueado.
4. **Região** — `America/Porto_Velho`, `br-abnt2` e `pt_BR.UTF-8` por padrão;
   dá para trocar fuso, teclado e idioma.
5. **Drivers e apps** — `amd`, `intel`, `nvidia`, `vm`, já marcado o que foi
   detectado; se instala os apps gráficos; se criptografa o disco com LUKS (a
   senha do disco é a do usuário).
6. **Revisão** — o resumo e o que há hoje no disco. Até aqui nada foi alterado
   e `Ctrl+C` cancela. Para seguir, escreva `formatar`. Qualquer outra coisa
   cancela.

Depois ele trabalha sozinho, em etapas numeradas: formata, instala os pacotes,
configura região, rede, boot, snapshots e serviços, cria o usuário e oferece
reiniciar. O registro fica em `/var/log/mk-install.log`, no live e no sistema
instalado. Se parar no meio, a última linha diz como desmontar e tentar de novo.

No primeiro login o Maré mostra as boas-vindas, um passeio de cinco passos; o
quarto oferece instalar as ferramentas de desenvolvimento do seu usuário
(node, python e as CLIs de IA, pelo mise; opcional, sem sudo, num terminal).
Para fazer isso depois:

```bash
mk ferramentas
```

Já vêm instalados, sem passo extra:

| Ferramenta | Comando | De onde vem | Como atualiza |
|---|---|---|---|
| VS Code (build aberto, "Code - OSS") | `code` | repositório oficial (`code`) | `mk atualizar` (pacman) |
| AWS CLI v2 | `aws` | repositório oficial (`aws-cli-v2`) | `mk atualizar` (pacman) |
| Cloudflare Wrangler | `wrangler` | repositório oficial (`wrangler`) | `mk atualizar` (pacman) |
| Vercel CLI | `vercel` | AUR (`vercel`), compilado no ISO | `mk atualizar` (yay) |

O `code` é o build aberto: usa o Open VSX no lugar da loja de extensões da
Microsoft. Quem quiser o binário da Microsoft instala o
`visual-studio-code-bin` pelo `yay`; o Mk não o traz porque ele não pode ser
redistribuído num ISO. Se a CLI da Vercel não tiver sido compilada no ISO, o
instalador avisa e o primeiro `mk atualizar` oferece instalá-la. Entrar nas
contas (`aws configure`, `vercel login`, `wrangler login`) fica com você.

O que o instalador faz por dentro está em
[installer/README.md](installer/README.md).

## Experimentar num Arch que já existe

Num Arch com Hyprland, o Mk roda como uma sessão separada, escolhida na tela de
login, sem tocar na config do desktop atual (`~/.config/hypr` fica como está).
A partir de um clone deste repositório:

```bash
bin/try-session install
```

Ele copia as configs para `~/.config/mk` (`hypr`, `shell`, `ghostty`,
`fastfetch`, `bash`), o tema GTK e os ícones do Mk para `~/.local/share`,
confere a config do Hyprland e **imprime** os comandos com `sudo` que faltam:
os pacotes, o lançador `/usr/local/bin/mk-session` e a entrada "Mk" no login.
Ele não roda nenhum deles; rode você. Depois saia da sessão e escolha "Mk".

```bash
bin/try-session status
```

Mostra o que está instalado e o que falta.

```bash
bin/try-session uninstall
```

Remove as pastas que o `install` criou, e só elas: uma pasta que não tem a
marca `.instalado-pelo-mk` não é sobrescrita nem apagada. Ficam para trás
`~/.config/mk/VERSION` (e, com ele, a pasta `~/.config/mk`) e o que foi posto
com `sudo`: para o lançador e a entrada "Mk" do login, ele imprime o
`sudo rm -f` e você roda.

O que o `try-session` **não** faz:

- não instala o Hyprland nem o `uwsm` (a sessão precisa dos dois); as listas
  do Mk se instalam com `bin/install-packages <perfil>`, em que o perfil de
  hardware é `amd`, `intel`, `nvidia` e/ou `vm` (sem perfil, ele sai com erro);
- não instala o `mk-update` nem `/usr/share/mk`. Para atualizar, `git pull` e
  `bin/try-session install` de novo;
- não mexe no `~/.bashrc`: o terminal do Mk não é ligado, então o comando `mk`
  inteiro (não só `mk atualizar`) não existe nesse modo;
- não mexe nos apps padrão, no boot, na tela de login nem nas fontes do
  sistema.

Para testar do zero num usuário novo, sem misturar com o seu:

```bash
sudo bin/new-user mare --wallpapers
```

Cria o usuário, instala os pacotes que faltam e a sessão, copia a config para a
home dele e pede a senha. Ao contrário do `try-session`, liga o terminal do Mk:
acrescenta a linha do `mk.bash` ao `~/.bashrc` do usuário novo. `--wallpapers`
copia para ele os papéis de parede de quem chamou o `sudo` (os de
`~/Pictures/Wallpapers`). `--admin` dá acesso a `sudo`. Para apagar depois:
`sudo userdel -r mare`. Se a sessão desse usuário não abrir,
`sudo bin/diagnose-user <nome>` junta o que explica, sem alterar nada.

## Atualizar

Num Mk instalado pelo ISO:

```bash
mk atualizar
```

É o mesmo que `mk-update`, e o mesmo que Ajustes › Atualizações abre. Ele
mostra o que há, pergunta "Atualizar agora?" e segue nesta ordem:

1. **O próprio Mk.** Baixa o canal, troca `/usr/share/mk` e as pastas que o Mk
   criou em `~/.config/mk`, recarrega o Hyprland e reinicia o Maré.
2. **Migrações.** O que mudou no Mk fora de `~/.config/mk` (em `/etc`, no boot,
   no `~/.bashrc`...). Cada uma roda uma vez. Veja
   [migrations/README.md](migrations/README.md).
3. **Arch.** `pacman -Syu`, junto com os apps que entraram na lista do Mk desde
   a última vez (cada um é oferecido uma vez só).
4. **AUR**, com o `yay`.

No caminho ele ainda oferece duas coisas, quando é o caso: destravar o `pacman`
se uma atualização anterior foi interrompida, e instalar pelo `yay` os pacotes
do AUR que ficaram fora do ISO.

Para só ver o que há, sem aplicar nada:

```bash
mk-update --check
```

O **canal** é o repositório público de lançamentos,
`https://github.com/codexmark/mk-releases.git` (em `system/mk/channel`). Não
precisa de login no GitHub. Para usar outro, escreva o endereço em
`~/.config/mk/channel`. Sem canal, o Arch e o AUR atualizam do mesmo jeito.

Com o `snap-pac`, cada `pacman` cria um snapshot antes e depois, e o menu do
limine lista os snapshots: se algo quebrar, dá para voltar pelo boot. A saída de
cada rodada fica em `~/.local/state/mk/atualizacao-*.log` (as cinco últimas).

Como uma mudança chega ao canal e o que fazer quando a atualização para no meio
está em [docs/atualizacoes.md](docs/atualizacoes.md).

### O comando `mk`

No terminal do Mk (`config/bash/mk.bash`, carregado pelo `~/.bashrc`):

| Comando | O que faz |
|---|---|
| `mk atualizar` | o Mk, o Arch e o AUR (`mk-update`) |
| `mk ajustes [página]` | abre os Ajustes, direto numa página se você der a chave dela |
| `mk papel` | papel de parede |
| `mk bloquear` | bloqueia a tela |
| `mk boas-vindas` | o passeio pelo Mk, de novo |
| `mk reiniciar-mare` | reinicia a ilha e os painéis |
| `mk ficha` | a ficha do fastfetch |
| `mk versao` | a versão do Mk instalada |
| `mk seu` | edita o `seu.bash` (seus atalhos, que o Mk não toca) |

`mk` sem nada mostra essa lista. O mesmo arquivo traz o ble.sh (sugestões
enquanto digita), o prompt do starship, zoxide, fzf, histórico dividido entre
terminais e atalhos curtos (`ll`, `lg`, `gs`, `mkcd`, `extract`, `porta`,
`servir`).

## A ilha e os painéis

A ilha é uma peça só, pendurada no meio da borda de cima. À esquerda, a marca
(abre a busca), os workspaces por letra grega (α β γ…) e as leituras do
sistema; no centro, relógio e clima; à direita, a bandeja e os indicadores.
Clicar num item faz o painel dele descer da ilha, embaixo do clique. `Esc`
fecha qualquer painel. A borda de baixo ondula com o som que está tocando.

- **Workspaces são projetos.** Um workspace novo e vazio mostra a folha que
  convida a começar; o seletor, com os repositórios git de `~/Work`, abre
  quando chamado (`qs ipc call mare toggle picker`) e não mais sozinho.
  Escolher um projeto, por ele ou pela busca, liga o workspace ao projeto; terminal, lazygit, lazydocker e yazi passam a abrir na pasta dele.
- **Busca** (um toque no `Super`): apps, projetos e comandos numa lista só.
- **Painéis:** projeto, volume, energia, rede, calendário e clima,
  notificações, sessão, tocando agora, Bluetooth, tela, sistema, área de
  transferência, papel de parede, captura de tela e emoji. Mais o Alt+Tab e os
  avisos passageiros (notificação que chega, nível de volume e brilho).
- **Ajustes** não descem da ilha: são uma folha no meio da tela.

Tudo o que os atalhos fazem passa pelo IPC do Maré, que você também pode chamar:

```bash
qs -p ~/.config/mk/shell ipc call mare toggle calendar
```

A lista dos arquivos e do IPC está em [shell/mare/README.md](shell/mare/README.md).

## Ajustes

`Super + ,` abre. À esquerda, as páginas; em cima delas, uma busca que acha
qualquer opção. O que você muda vale na hora.

| Grupo | Página | Chave | O que tem |
|---|---|---|---|
| Aparência | Ilha | `ilha` | o que a ilha mostra e quanto do fundo deixa passar |
| Aparência | Fundo | `fundo` | a folha atrás das janelas e como o papel de parede pousa nela |
| Aparência | Movimento | `movimento` | a velocidade de painéis, ilha, janelas e workspaces |
| Aparência | Notificações | `notificacoes` | os avisos que descem da ilha |
| Janelas | Janelas | `janelas` | espaços, bordas e cantos |
| Dispositivos | Tela | `tela` | tamanho da interface e luz noturna |
| Dispositivos | Monitores | `monitores` | resolução, taxa, escala, posição e rotação de cada tela |
| Dispositivos | Som | `som` | saída e microfone |
| Dispositivos | Teclado e mouse | `entrada` | layout, repetição de teclas, mouse e touchpad |
| Dispositivos | Energia | `energia` | o que acontece quando você se afasta: descanso de tela aos 3 min, bloqueio aos 5, tela desligada aos 30. De fábrica o Mk não dorme; suspender é uma escolha |
| Dispositivos | Impressoras | `impressoras` | as deste computador e as da rede |
| Dispositivos | Controle | `controle` | o controle Xbox e o driver xone, para o fone plugado nele |
| Rede | VPN | `vpn` | WireGuard e OpenVPN, pelo NetworkManager |
| Sistema | Apps padrão | `apps` | que app abre cada tipo de coisa, e o terminal |
| Sistema | Atualizações | `atualizacoes` | o Mk, o Arch e o AUR |
| Sistema | Atalhos | `atalhos` | trocar a tecla de cada atalho |
| Sistema | Sobre o Mk | `sobre` | esta máquina e onde os ajustes ficam guardados |

A chave serve para abrir direto numa página: `mk ajustes monitores`.

## Atalhos

Os atalhos abaixo são os de fábrica. Dá para trocar cada um em Ajustes
(`Super + ,`) › Atalhos: as trocas ficam em `~/.config/mk/atalhos.lua`, que o
`binds.lua` lê ao criar os atalhos, e valem na hora. A lista mora em
`config/hypr/mare/binds.lua` e é espelhada em `shell/mare/Keybinds.js`
(`bin/check-keybinds` confere os dois).

Os marcados com **\*** são fixos: os Ajustes não trocam. Os dois de emergência
podem mudar de tecla, mas nunca ficam sem atalho.

**Emergência** (funcionam mesmo se o Maré não abrir)

| Atalho | Ação |
|---|---|
| `Super + Shift + Enter` | terminal de emergência |
| `Super + Shift + Esc` | sair do Hyprland |

**Maré**

| Atalho | Ação |
|---|---|
| `Super` (um toque) **\*** | busca: apps, projetos e comandos |
| `Super + Espaço` | busca: apps, projetos e comandos |
| `Alt + Tab` **\*** | trocar de janela |
| `Alt + Shift + Tab` **\*** | trocar de janela, para trás |
| `Super + P` | projeto do workspace |
| `Super + C` | calendário |
| `Super + N` | notificações |
| `Super + Shift + N` | não perturbe |
| `Super + Esc` | controles rápidos: leituras, Wi-Fi, Bluetooth, energia, volume, brilho, sessão e alternadores |
| `Super + Ctrl + Esc` | sair, bloquear, desligar |
| `Super + W` | papel de parede |
| `Super + Shift + V` | área de transferência |
| `Super + .` | emoji |
| `Super + ,` | ajustes |
| `Super + L` | bloquear a tela |

**Apps** (abrem na pasta do projeto em foco)

| Atalho | Ação |
|---|---|
| `Super + Enter` | terminal no projeto |
| `Super + G` | lazygit |
| `Super + D` | lazydocker |
| `Super + E` | arquivos (yazi) |
| `Super + B` | navegador |

**Janelas**

| Atalho | Ação |
|---|---|
| `Super + Q` | fechar |
| `Super + F` | tela cheia |
| `Super + V` | flutuar |
| `Super + setas` | mover o foco |
| `Super + Shift + setas` | mover a janela |
| `Super + Ctrl + setas` | redimensionar |
| `Super + arrastar` (botão esquerdo) **\*** | mover com o mouse |
| `Super + arrastar` (botão direito) **\*** | redimensionar com o mouse |

**Workspaces**

| Atalho | Ação |
|---|---|
| `Super + 1…9` | ir para α β γ… |
| `Super + Shift + 1…9` | levar a janela |
| `Super + Tab` | workspace anterior |
| `Super + roda do mouse` **\*** | workspace seguinte / anterior |

**Tela**

| Atalho | Ação |
|---|---|
| `Print` | recortar e copiar |
| `Shift + Print` | salvar a tela inteira |
| `Super + Print` | painel de captura (área, janela, vídeo) |
| `Super + Shift + R` | gravar a tela / parar |

**Teclas de mídia** (todas fixas **\***)

| Tecla | Ação |
|---|---|
| volume + / − | aumentar / diminuir o volume |
| mudo | silenciar |
| mudo do microfone | silenciar o microfone |
| brilho + / − | aumentar / diminuir o brilho |
| tocar, próxima, anterior | tocar / pausar, próxima faixa, faixa anterior |

## Onde vai o que é seu

O Mk instalado fica em `~/.config/mk`. Uma atualização troca inteiras as pastas
que o Mk criou (as que têm a marca `.instalado-pelo-mk`): `hypr`, `shell`,
`ghostty`, `fastfetch` e `bash`. **Não edite nada dentro delas**: some na
próxima atualização. O que é seu fica na raiz de `~/.config/mk`. Ali a
atualização só reescreve o `VERSION`; nos seus arquivos ela não toca:

| Arquivo | O que é | Quem escreve |
|---|---|---|
| `~/.config/mk/ajustes.json` | tudo o que você escolheu nos Ajustes | os Ajustes |
| `~/.config/mk/ajustes.lua` | a parte dos ajustes que é do Hyprland, carregada no login | os Ajustes |
| `~/.config/mk/monitores.lua` | uma regra por tela | Ajustes › Monitores |
| `~/.config/mk/atalhos.lua` | os atalhos que você trocou | Ajustes › Atalhos |
| `~/.config/mk/hypridle.conf` | os tempos de ociosidade | Ajustes › Energia |
| `~/.config/mk/seu.bash` | seus aliases, variáveis e `PATH`; carrega no fim do bash do Mk | você (`mk seu`) |
| `~/.config/mk/channel` | outro canal de atualização | você |
| `~/.config/mare/city` | a cidade do clima, se não for a do fuso horário | você |
| `~/.config/mk/.bem-vindo` | marca de que você já viu as boas-vindas; apagar faz a folha voltar no próximo login | o Maré |
| `~/.config/mk/VERSION` | a versão do Mk instalada, que `mk versao` mostra | o `mk-update` |

Um modelo do `seu.bash` está em `config/bash/seu.bash.exemplo`; o instalador já
o põe no lugar. Os papéis de parede são seus também: ficam em `Wallpapers`,
dentro da pasta de imagens do sistema (`~/Imagens` em português).

**Hyprland à mão.** O `hyprland.lua` carrega por último, se existirem ao lado
dele, `monitors.lua` e `user.lua` (em `~/.config/mk/hypr/`). Funcionam, mas
ficam dentro da pasta `hypr`, que a atualização troca inteira: hoje eles **se
perdem a cada atualização do Mk**. Guarde uma cópia fora dali e recoloque
depois, ou tire a marca `~/.config/mk/hypr/.instalado-pelo-mk`: o `mk-update`
passa a deixar a pasta como está, e você deixa de receber as mudanças do
Hyprland do Mk.

## O repositório

| Caminho | O que é |
|---|---|
| `config/hypr/` | Hyprland em Lua: `hyprland.lua` e um arquivo por assunto em `mare/` |
| `config/bash/` | o terminal do Mk: `mk.bash`, o prompt (`starship.toml`), `seu.bash.exemplo` |
| `config/ghostty/`, `config/fastfetch/` | terminal e a ficha que ele mostra ao abrir |
| `config/mimeapps.list` | os apps padrão de fábrica |
| `tests/` | os testes automáticos (`tests/run`) e o gancho de `git push` |
| `apps/` | os apps que vêm com o Mk: o MkTunes (player de música, gerado no repositório dele) e, com o código aqui, em Go por trás e Quickshell na frente: o MkPulso (monitor de sistema), o MkPacotes (gerenciador de pacotes), o MkInstalar, o MkComprimir, o MkCalc (calculadora), o MkCorte (cortes de vídeo) e o MkEscrita (Markdown) |
| `vault/` | o Mk explicado por camadas, do kernel aos aplicativos, com os porquês: uma vault do Obsidian (abra a pasta nele e comece por `00 Início`) |
| `shell/mare/` | o Maré: QML numa pasta só, mais `scripts/` (o que ele roda por fora) |
| `installer/` | `mk-install` e um arquivo de respostas de exemplo, para testes |
| `iso/` | `build` (gera o ISO), `vm` (VM de teste no QEMU), `profile/` (o que muda no archiso) |
| `migrations/` | `system/` e `user/`: o que o `mk-update` faz uma vez em cada Mk já instalado |
| `system/packages/` | as listas: `base`, `desktop`, `dev`, `apps`, `hardware` (por perfil), `aur` |
| `system/mk/` | a cara do Mk: boot (plymouth, limine), login (sddm), tema GTK, ícones, Qt, fontes, logo, `os-release` e o `channel` |
| `scripts/system/firewall.sh` | ufw: nega entrada, libera saída, integra com Docker |
| `scripts/user/neovim.sh` | aplica o LazyVim se não houver config do Neovim |
| `scripts/user/mise-tools.sh` | node/python globais e CLIs de IA via mise |
| `bin/mk-session` | o lançador da sessão (Hyprland pelo `uwsm`) |
| `bin/mk-update` | o atualizador |
| `bin/mk-release` | publica o commit atual no repositório público de lançamentos |
| `bin/try-session`, `bin/new-user`, `bin/diagnose-user` | experimentar sem instalar pelo ISO |
| `bin/check-packages` | confere cada lista contra os repositórios (não instala nada) |
| `bin/install-packages` | instala as listas num Arch já instalado; pede o perfil de hardware |
| `bin/lib-packages.sh` | as funções que leem as listas, usadas por `install-packages`, `check-packages` e pelo instalador |
| `bin/check-keybinds` | confere que `binds.lua` e `Keybinds.js` estão iguais |
| `bin/mk-hibernate` | desfaz um preparo antigo de hibernação (`remove`); o Mk não hiberna mais e o `setup` recusa |
| `bin/check-shell` | abre um bash interativo com o `mk.bash` e confere que ele não cai |
| `bin/dev-shot` | testa o Maré num Hyprland aninhado e tira uma captura |
| `docs/` | `atualizacoes.md` e `desenvolvimento.md`: para quem mantém o Mk |

Regras das listas de pacotes:

- Um pacote por linha; `#` e linhas em branco são ignorados.
- Se existe em `core`/`extra`, vai numa lista oficial; o AUR é último recurso.
- Rode `bin/check-packages` depois de editar qualquer lista.
- Todo script é idempotente: rodar duas vezes dá o mesmo resultado.

## Mais documentação

| Arquivo | Assunto |
|---|---|
| [iso/README.md](iso/README.md) | gerar o ISO e testar numa VM |
| [installer/README.md](installer/README.md) | o que o instalador faz, passo a passo |
| [migrations/README.md](migrations/README.md) | como escrever uma migração |
| [shell/mare/README.md](shell/mare/README.md) | o Maré por dentro: arquivos, IPC, scripts |
| [docs/atualizacoes.md](docs/atualizacoes.md) | do repositório ao Mk instalado: lançar, o canal, e o que fazer se a atualização parar |
| [docs/desenvolvimento.md](docs/desenvolvimento.md) | trabalhar no Mk: testar, conferir, lançar |
| [docs/limites.md](docs/limites.md) | o que nunca foi testado em máquina de verdade e o que se sabe que não funciona |
| [tests/README.md](tests/README.md) | os testes automáticos e a regra de teste primeiro |
