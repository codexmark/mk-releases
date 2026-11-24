---
camada: "12 Decisões e limites"
tags:
  - mk
  - decisões
  - atualização
  - migrações
  - pacman
fontes:
  - "tests/migrations/regras.test.sh"
  - "bin/mk-update"
  - "migrations/README.md"
  - "docs/atualizacoes.md"
  - "migrations/system/20261125-impressoras.sh"
---

# D03 Migração não instala pacote

Uma migração de sistema declara os pacotes de que precisa numa linha `# pacotes:`; quem instala é o `mk-update`, dentro da transação do `-Syu`.

## Contexto

As migrações rodam **antes** da atualização do sistema. As primeiras chamavam `pacman -S --needed` por conta própria. Isso é instalar um pacote com a lista de pacotes velha da máquina: o download pode falhar (a versão que a base local conhece já saiu dos espelhos), e mesmo dando certo é uma atualização parcial, que no Arch quebra bibliotecas. Pior: a migração que falha fica pendente, e como ela roda antes do `-Syu`, **toda atualização seguinte** pararia no mesmo ponto.

## Escolha

- A migração não instala nada. Declara no cabeçalho: `# pacotes: a b c`.
- O `mk-update` separa as migrações pendentes em duas filas: as que não pedem pacote rodam antes do pacman; as que pedem, depois.
- Os pacotes pedidos (só os que faltam) entram na **mesma transação** da atualização: `pacman -Sy`, filtra os nomes que não existem mais, `pacman -Su --needed <pedidos> <apps>`.
- A migração tem de aguentar o pacote não estar lá: avisa e sai com 0.

## Alternativas

- **A migração instalar por conta própria** (o estado anterior): descartada pelo contexto acima; é o que o cabeçalho do `mk-update` explica.
- **Rodar todas as migrações depois do `-Syu`** *(não discutida no repositório)*: o código mantém as que não pedem pacote antes do pacman; uma delas pode criar a lista de apps já vistos, que o passo do pacman lê em seguida.

## Consequência

- Um nome de pacote errado ou removido não trava nada: sai do pedido com um aviso. Sem o `-Sy` e o filtro, "um nome que não existe faria o pacman recusar a atualização inteira, hoje e sempre".
- Só servem nomes dos repositórios oficiais. Para o AUR, a migração acrescenta o nome ao `aur-pendentes.txt` ([[AUR na instalação]]).
- O formato do nome é restrito (`^[a-z0-9@._+-]+$`); o que não casa é descartado em silêncio. A linha vira argumento de um `sudo pacman`.
- As quatro migrações anteriores à regra foram convertidas para `# pacotes:`, e `tests/migrations/regras.test.sh` reprova qualquer uma que instale no corpo (`pacman -S`, `yay`, `paru`) ou declare um pacote que não está nas listas ([[Migrações]]).
- O mesmo raciocínio vale para os [[Apps oferecidos]]: entram na mesma transação.

## Onde está no código

`bin/mk-update` (o cabeçalho, `migration_packages`, as filas `pend_now`/`pend_late`, o passo 3); `migrations/README.md`.

## Ligações

Afeta: [[Migrações]], [[mk-update]], [[Pacman e AUR]], [[Catálogo de migrações]]. Índice: [[Registro de decisões]].
