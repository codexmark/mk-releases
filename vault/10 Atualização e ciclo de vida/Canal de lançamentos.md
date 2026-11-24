---
camada: "10 Atualização e ciclo de vida"
tags:
  - mk
  - atualização
  - canal
  - versão
fontes:
  - "system/mk/channel"
  - "docs/atualizacoes.md"
  - "shell/mare/scripts/mare-updates"
  - "bin/mk-release"
  - "bin/mk-update"
---

# Canal de lançamentos

O **canal** é o repositório git público de onde todo Mk instalado se atualiza, sem login. O código do dia a dia fica num repositório privado; só vai a público o que passa pelo `bin/mk-release`.

## Para quê

Separar "trabalhando" de "lançado": um Mk instalado só vê versões que alguém decidiu publicar.

## Como funciona

**Onde o canal está escrito** (vale a primeira linha que não é comentário, nesta ordem):

| Arquivo | De quem |
|---|---|
| `~/.config/mk/channel` | do usuário, para um canal só dele |
| `/usr/share/mk/system/mk/channel` | do Mk |

Canal vazio quer dizer "sem canal": o Mk não se atualiza; o Arch e o AUR, sim.

**A versão** é o arquivo `VERSION`, uma linha `commit data`: 12 letras do commit e a data dele (`AAAA-MM-DD`). Vale `~/.config/mk/VERSION`; sem ele, `/usr/share/mk/VERSION` (o que a ISO gravou).

**Há versão nova quando** o `VERSION` do canal tem outro commit **e** a data dele não é anterior à instalada.

**`NOVIDADES`**: o que chega com o lançamento, em até 12 linhas; os Ajustes mostram antes de atualizar.

**Como a procura lê o canal** (`mare-updates`): um clone raso, sem *checkout* e sem baixar arquivos (`--depth 1 --filter=blob:none --no-checkout`), e `git show HEAD:VERSION` e `HEAD:NOVIDADES`. Com prazo de 20 segundos, e nunca pedindo senha (`GIT_TERMINAL_PROMPT=0`, `GIT_ASKPASS=/bin/false`, `credential.interactive=never`).

## Por quê assim

- **Público e sem login**: a primeira versão do canal era o próprio repositório privado, e a procura precisava "nunca pedir senha; avisar quando não tem acesso". Um repositório público só de lançamentos dispensa credenciais em toda máquina instalada ([[D04 Canal público separado]]).
- **Git como transporte**: não há servidor, pacote nem API do Mk; um `git clone` de um repositório público basta, e o canal pode ser trocado por qualquer outro endereço git.
- **A regra da data**: "um Mk feito de um commit ainda não lançado não 'volta' para o último lançamento". Quem instalou de uma ISO gerada de um commit mais novo que o último lançado não é rebaixado.
- **Não existe voltar só o Mk**: o `mk-update` sempre instala a ponta do canal e recusa data mais antiga. O conserto de um lançamento ruim é outro lançamento.
- **`VERSION` em dois lugares**: o do usuário é o que o `mk-update` reescreve; o de `/usr/share/mk` é o da ISO. Cada usuário da máquina tem a sua versão das pastas de `<raiz>`.

## Onde está no código

- `system/mk/channel` (o endereço e o comentário)
- `shell/mare/scripts/mare-updates` (a leitura do canal e a comparação)
- `bin/mk-release` (quem escreve `VERSION` e `NOVIDADES`)

## Ligações

Usado por: [[mk-update]], [[mk-release]], [[Atualizações na ilha e nos Ajustes]], [[Voltar atrás pelo menu de boot]]. Decisão: [[D04 Canal público separado]].
