---
camada: "03 Base do sistema"
tags:
  - mk
  - pacotes
  - pacman
  - aur
fontes:
  - "installer/mk-install"
  - "bin/mk-update"
  - "bin/install-packages"
  - "system/packages/aur.packages"
  - "docs/atualizacoes.md"
  - "docs/desenvolvimento.md"
---

# Pacman e AUR

O Mk usa só os repositórios oficiais do Arch e um punhado de pacotes do AUR, instalados pelo `yay`. Não há repositório próprio de pacotes fora o `mk-aur` local da ISO.

## Para quê

Manter o sistema o mais perto possível de um Arch comum: o que o Mk acrescenta é configuração e o shell, não pacotes binários seus.

## Como funciona

- **Instalação**: `pacstrap -K` com as listas oficiais; depois os do AUR, do repositório local da ISO ([[AUR na instalação]]).
- **Atualização**: o [[mk-update]] roda uma transação só do pacman, sem perguntas no meio (`pacman -Sy`, um ensaio com `pacman -Su --print`, e então `pacman -Su --noconfirm` com o sistema, os pacotes pedidos por migração e os apps aceitos), depois o AUR com `yay -Sua --noconfirm`.
- **Procura sem sudo**: `checkupdates` (do `pacman-contrib`, por isso ele está em `base.packages`) numa base temporária, e `yay -Qua` ([[Atualizações na ilha e nos Ajustes]]).
- **Trava**: se existe `/var/lib/pacman/db.lck` e não há `pacman`, `yay`, `makepkg` nem `pamac` rodando, o `mk-update` oferece remover a trava.
- **Gancho do pacman**: `mk-os-release.hook` ([[Nome do sistema (os-release)]]). E o `snap-pac` ([[Snapper e snap-pac]]).

Pacotes do AUR hoje: `yay-bin`, `limine-mkinitcpio-hook`, `limine-snapper-sync`, `ufw-docker`, `localsend-bin`, `ttf-fraunces-variable`, `blesh-git`, `vercel`.

## Por quê assim

- **Nunca se instala pacote novo sobre um sistema desatualizado.** No Arch isso quebra bibliotecas (atualização parcial). É por isso que os apps novos e os pacotes pedidos pelas migrações entram **na mesma transação** do `-Syu`, e que as migrações não podem instalar nada por conta própria ([[D03 Migração não instala pacote]]).
- **`pacman -Sy` antes, só quando há pedidos**: "um nome que não existe faria o pacman recusar a atualização inteira, hoje e sempre". Com a base nova em mãos, o nome que saiu dos repositórios é tirado do pedido.
- **A trava do pacman**: uma atualização interrompida (terminal fechado, queda de luz) deixa a base travada e tudo dali em diante falharia. Foi achado na VM: a trava de um `yay` interrompido fazia a primeira migração falhar.
- **`yay -Sua --noconfirm --answerclean None --answerdiff None`**: o `yay` compila como o usuário e só pede senha para instalar; as respostas fixas evitam que ele abra o PKGBUILD no meio da rodada.
- **AUR não trava nada**: na instalação, o que falha vira aviso; na atualização, uma falha do AUR deixa o resumo em "parou em: aur" com o sistema já atualizado.

## Onde está no código

- `installer/mk-install`: `install_packages`, `install_aur`, `make_pacman_conf`
- `bin/mk-update` (passos 3 e 4)
- `bin/install-packages`

## Ligações

Depende de: [[Listas de pacotes]]. Usado por: [[mk-update]], [[Migrações]], [[Snapper e snap-pac]], [[Ganchos do limine]].
