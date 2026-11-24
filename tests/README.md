# Testes do Mk

```bash
tests/run              # tudo
tests/run mare-sleep   # só os arquivos com esse trecho no caminho
tests/run -v scripts   # mostra cada caso
```

Não precisa de root, de rede nem de sessão gráfica, e não toca na máquina.
A suíte inteira leva um ou dois minutos (os arquivos rodam em paralelo); um
arquivo só, segundos. É para rodar a cada mudança.

## A regra: teste primeiro

Toda mudança de comportamento começa por um teste que falha.

1. Escreva o caso em `tests/` dizendo o que se espera. Rode: ele falha.
2. Mude o código até passar.
3. Rode tudo (`tests/run`) antes do commit.

Um defeito achado em uso (uma issue) vira primeiro um caso que o reproduz, e
só então a correção. Código novo sem teste não entra; código antigo que for
tocado ganha o teste que faltava.

O `git push` e o `bin/mk-release` rodam `tests/run` e param se algo falhar
(veja "Onde isto roda" abaixo). Não há CI na nuvem: tudo roda aqui.

## Onde fica cada coisa

| Pasta | O que testa |
|---|---|
| `tests/lint.test.sh` | sintaxe e lint de tudo: bash, shellcheck, QML, Lua, JSON, Python |
| `tests/js/` | o JavaScript do shell (`shell/mare/*.js`), com node |
| `tests/scripts/` | `shell/mare/scripts/mare-*` |
| `tests/bin/` | `bin/*` |
| `tests/installer/` | `installer/mk-install` |
| `tests/iso/` | `iso/build`, `iso/vm`, o perfil da ISO |
| `tests/system/` | `system/` e `scripts/` (listas de pacotes, arquivos de sistema) |
| `tests/migrations/` | cada migração de `migrations/` |
| `tests/config/` | `config/` (Lua do Hyprland, `mk.bash`, hypridle) |
| `tests/docs/` | a documentação contra o código |

Um arquivo de teste por arquivo testado, com o mesmo nome:
`shell/mare/scripts/mare-sleep` → `tests/scripts/mare-sleep.test.sh`. As
migrações ficam em `tests/migrations/system/` e `tests/migrations/user/`, como
em `migrations/`; o `tests/migrations/mig.sh` monta a raiz falsa em que cada
uma roda (`MK_ROOT`).

## Como escrever um teste em bash

```bash
#!/usr/bin/env bash
# shell/mare/scripts/mare-sleep: o que este arquivo cobre, em uma linha.
. "$(dirname "$0")/../lib.sh"

t "now: 'nada' não chama nada"
stub systemctl
echo '{"power.sleep":"nada"}' >"$SBX/ajustes.json"
ok env AJUSTES=$SBX/ajustes.json bash "$REPO/shell/mare/scripts/mare-sleep" now
not_called "systemctl"
```

O `tests/lib.sh` explica cada função no cabeçalho. O essencial:

- `$SBX` é uma pasta temporária só daquele arquivo; `HOME` e os `XDG_*`
  apontam para dentro dela. Escreva só ali.
- O nome do caso (`t "..."`) diz o comportamento esperado, em português, do
  jeito que alguém leria numa lista: "status: a Navi 10 entra em risky".
- **Guardas.** `sudo`, `systemctl`, `pacman`, `hyprctl`, `nmcli`, `curl`,
  `git` e os outros comandos que mexem na máquina ou saem para a rede estão
  trocados por guardas: chamar um deles sem `stub` reprova o arquivo. O teste
  declara o que o comando responde:

  ```bash
  stub nmcli 'echo "wlan0:wifi:connected"'
  stub pacman '[[ $1 == -Q ]] && exit 1'
  ```

  Cada chamada fica em `$SBX/calls` (`called`, `not_called`).
- Um script que lê `/proc`, `/sys` ou `/etc` recebe a raiz por variável
  (`PROC`, `SYS`, `MK_HIB_ROOT`...). Se o script ainda não tem essa variável,
  acrescente: é a emenda mínima que o deixa testável.
- **Nunca monte um `PATH` com `/usr/bin`**: os guardas saem do caminho e o
  comando de verdade roda. Para o ramo "a ferramenta não está instalada", use
  `PATH=$(path_sem yay)`. O lint reprova um teste que faça diferente.
- Falta uma ferramenta opcional (node, lua, qmllint)? `skip "motivo"`.

## JavaScript e Lua

`tests/js/*.test.mjs` roda no node. Os `.js` do shell são bibliotecas do QML
(`.pragma library`); o `tests/js/load.mjs` carrega um deles como módulo.

`tests/config/*.test.lua` roda no lua, com um `hl` falso
(`tests/config/hl.lua`) que anota o que a configuração pediu ao Hyprland e
recusa uma função que o Mk não usa.

A lógica pura que mora dentro de um QML (entre os comentários
`// --- puro: início ---` e `puro: fim`, como em `shell/mare/Monitors.qml`) é
tirada de lá pelo `tests/js/qml-puro.mjs` e testada no node.

Os dois imprimem as mesmas linhas `ok - ...` / `not ok - ...` e saem com 1 se
algo falhou.

## O que os testes não cobrem

O que só existe com a sessão no ar: o desenho dos painéis, o foco do teclado,
o Hyprland de verdade. Isso continua sendo conferido à mão com o
`bin/dev-shot` (uma sessão aninhada e escondida) e na VM (`iso/vm`). A
instalação num disco e o boot também: só na VM.

## Onde isto roda

- `tests/run`, à mão, a qualquer hora.
- No `git push`: o gancho `tests/hooks/pre-push`. Ligue uma vez por clone:

  ```bash
  git config core.hooksPath tests/hooks
  ```

- No `bin/mk-release`, antes de publicar.
