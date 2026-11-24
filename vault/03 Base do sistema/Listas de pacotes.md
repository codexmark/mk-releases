---
camada: "03 Base do sistema"
tags:
  - mk
  - pacotes
  - listas
fontes:
  - "system/packages/base.packages"
  - "system/packages/desktop.packages"
  - "system/packages/dev.packages"
  - "system/packages/apps.packages"
  - "system/packages/hardware.packages"
  - "system/packages/aur.packages"
  - "bin/lib-packages.sh"
  - "docs/desenvolvimento.md"
---

# Listas de pacotes

Tudo o que o Mk instala está em seis arquivos de texto em `system/packages/`, um por função. Um pacote por linha; `#` até o fim da linha e linhas em branco são ignorados.

## Para quê

As listas são a definição do sistema: o instalador, o build da ISO, o atualizador e as ferramentas de checagem leem os mesmos arquivos.

## Como funciona

| Lista | Para quê | Origem |
|---|---|---|
| `base.packages` | o sistema: kernel, boot (limine, btrfs, snapper), rede, impressão, Bluetooth, áudio, energia, fontes | oficiais |
| `desktop.packages` | a sessão: Hyprland, uwsm, sddm, Quickshell e o que o Maré chama | oficiais |
| `dev.packages` | o kit de desenvolvimento: terminal, CLIs, git, editor, linguagens, containers | oficiais |
| `apps.packages` | os apps gráficos do dia a dia; **opcionais** | oficiais |
| `hardware.packages` | drivers e microcódigo em seções `[amd]`, `[intel]`, `[nvidia]`, `[vm]` | oficiais |
| `aur.packages` | o que só existe no AUR, pelo nome do `PackageBase` | AUR |

Quem lê:

- **O instalador** junta `base`, `desktop` e `dev`, mais `apps` se aceito, mais as seções de hardware, num `pacstrap` só.
- **O `iso/build`** lê `aur.packages`, compila e monta o repositório `mk-aur` ([[AUR na instalação]]).
- **O `mk-update`** só olha `apps.packages` ([[Apps oferecidos]]).
- **`bin/install-packages`** lê todas, para um Arch já instalado.
- **`bin/check-packages`** confere cada nome contra os repositórios ([[Checagens]]).

`bin/lib-packages.sh` dá as duas funções de leitura: `read_list <arquivo>` e `read_hardware <perfil>...`.

## Por quê assim

- **Separadas por função** porque cada lista tem um destino diferente: `apps` pode ser pulada na instalação e é a única que o atualizador oferece depois; `hardware` é escolhida por máquina; `aur` segue outro caminho (é compilada, pode falhar sem travar nada). Uma lista única obrigaria todos os leitores a saber classificar cada pacote. A estrutura segue o modelo do Ryoku.
- **Se existe em `core`/`extra`, vai numa lista oficial; o AUR é último recurso.** O `check-packages` reclama de um pacote do AUR que também exista nos oficiais.
- **O que acontece ao acrescentar um pacote** depende da lista, e é a parte que mais engana:
  - em `apps`: o `mk-update` oferece sozinho, uma vez por máquina;
  - em `base`, `desktop` ou `dev`: só chega a instalações novas; para os Mk já instalados, uma migração com `# pacotes:` ([[Migrações]]);
  - em `aur`: só chega por uma ISO nova; para os já instalados, uma migração que acrescente o nome ao `aur-pendentes.txt`.
- **Duas listas à mão, fora daqui.** `bin/try-session` e `bin/new-user` têm cada um a sua lista `needed` de pacotes sem os quais a sessão não abre. Não são lidas das listas e não são iguais entre si (a do `try-session` não tem o `uwsm`). É uma dívida conhecida.

## Onde está no código

`system/packages/*.packages`, `bin/lib-packages.sh`, `bin/check-packages`, `bin/install-packages`. Testes: `tests/system/packages.test.sh`.

## Ligações

Usado por: [[Pacman e AUR]], [[Instalador passo a passo]], [[ISO]], [[mk-update]], [[Microcódigo e perfis de hardware]], [[Apps oferecidos]].
