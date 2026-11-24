---
camada: "08 Aplicativos"
tags:
  - mk
  - aplicativos
  - mkpacotes
  - pacotes
  - pacman
fontes:
  - "apps/mkpacotes/CONTRATO.md"
  - "apps/README.md"
  - "apps/mkpacotes/bin/mkpacotes"
  - "apps/mkpacotes/qml/App.qml"
  - "system/mk/apps-links"
  - "migrations/system/20261130-mkpacotes-mkutil.sh"
  - "docs/limites.md"
---

# MkPacotes

O MkPacotes é o gerenciador de pacotes gráfico do Mk: o que está instalado, a busca nos repositórios e no AUR, a ficha de cada pacote, instalar, remover, limpar, e o histórico do pacman.

## Para quê

Ver e cuidar dos pacotes sem decorar as opções do `pacman` e do `yay`, com a garantia de ver **antes** o que cada ação vai fazer.

## Como funciona

Quatro áreas; `mkpacotes <área>` abre direto numa.

- **Instalados**, com filtros: pedidos por você, do AUR, órfãos, com atualização. Cada pacote traz o `motivo` (pedido ou dependência), a `origem` (o repositório, ou `aur` quando nenhum o tem), se é órfão (a conta do `pacman -Qdt`) e se é **protegido**.
- **Procurar**: nos repositórios (`pacman -Ss`) e no AUR, numa lista só, até 200 resultados; o nome igual ao texto vem primeiro, os oficiais antes do AUR, o AUR por votos. Sem rede, os oficiais continuam.
- **Atualizações**: o que há para atualizar, pelo `checkupdates` (ou `pacman -Qu`) e por uma pergunta ao AUR. **Só mostra.**
- **Histórico**: o `pacman.log` resumido (instalado, atualizado, removido, reinstalado, rebaixado), com busca.

**Plano antes de agir.** Toda ação tem dois subcomandos com os mesmos argumentos:

1. `plano <acao>`: o ensaio, sem fazer nada e sem root (`pacman -S --needed --print`, `pacman -Rs --print`). Devolve o que entra, o que sai (inclusive as dependências que ficariam órfãs), quanto baixa ou libera, os avisos e **os comandos exatos** que serão rodados.
2. `agir <acao>`: depois da confirmação na janela. Refaz o ensaio e as recusas e transmite o progresso linha a linha.

Ações: `instalar`, `remover`, `orfaos`, `cache`, `atualizar`.

**Os comandos**: `pkexec pacman -S --needed --noconfirm -- <oficiais>`; `yay -S … --sudo pkexec -- <do AUR>` (o yay nunca como root); `pkexec pacman -Rns --noconfirm -- <pacotes>`; `pkexec pacman -Sc --noconfirm`.

**Recusas** (código 2, antes de qualquer comando):

- remover um pacote **protegido**: `base`, `linux`, `linux-firmware`, `systemd`, `pacman`, `glibc`, `sudo`, `limine`, `hyprland`, `quickshell`, `sddm` e tudo o que está em `base.packages`. Com `--mesmo-assim` o plano sai, com um aviso por protegido;
- o pacman travado (`db.lck`);
- outra ação do MkPacotes em andamento (uma de cada vez, por `flock`);
- o coletor rodando como root.

## Por quê assim

- **Atualizar tudo é do `mk atualizar`**: a ação `atualizar` só abre o `mk-update` num terminal e espera a janela dele fechar. O atualizador tem a ordem certa (o Mk, as migrações, o pacman, o AUR) e sabe lidar com a trava; "o coletor não a refaz" ([[mk-update]]).
- **Senha pelo polkit, não pelo sudo**: o coletor nunca pede senha; o `pkexec` chama o polkit e o Maré mostra o pedido ([[Polkit]]).
- **Os instalados são lidos direto de `local/`**, sem chamar o pacman: é só leitura de arquivos, não depende do idioma da saída do `pacman -Qi` e funciona com o pacman travado. O que é dos repositórios vem do pacman, com `LC_ALL=C`, porque as bases sincronizadas podem estar em zstd, que a biblioteca padrão do Go não lê.
- **Nome de pacote validado e sempre depois de `--`**: só `^[a-z0-9@._+-]+$`, sem começar com `-` nem `.`. Um nome nunca vira uma opção de comando.
- **O `agir` refaz tudo**: "a janela não é de confiança".
- **Os protegidos vêm também da lista do Mk**: o que o Mk declara como base não sai por um clique.

> [!warning] Defeitos abertos
> Segundo `docs/limites.md`: o MkPacotes chama `pkexec pacman` **pelo nome** (o certo é o caminho fixo `/usr/bin/pacman`, como o MkInstalar já faz), e não remove pacote do AUR com letra maiúscula no nome (a validação só aceita minúsculas). Instalar e remover um pacote com a senha pedida pelo Maré foi exercitado na VM pelo motor; a folha de confirmação da janela nunca foi clicada.

## Onde está no código

`apps/mkpacotes/` (`src/internal/acoes/` tem o plano, a ação e a segurança; `src/internal/alpm`, `aur`, `pacman`, `historico`); `qml/`; `CONTRATO.md`. `system/mk/apps-links`.

## Ligações

Molde: [[Apps do Mk (motor em Go, janela em QML)]]. Depende de: [[Pacman e AUR]], [[Listas de pacotes]], [[Polkit]], [[mk-update]]. Relacionado: [[Atualizações na ilha e nos Ajustes]], [[MkInstalar e MkComprimir]].
