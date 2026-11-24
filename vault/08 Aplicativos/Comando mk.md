---
camada: "08 Aplicativos"
tags:
  - mk
  - aplicativos
  - terminal
  - comando
fontes:
  - "config/bash/mk.bash"
  - "README.md"
  - "bin/try-session"
---

# Comando mk

`mk` é uma função do bash definida em `mk.bash`: um nome só para as coisas do sistema que se fazem pelo terminal.

## Para quê

Não precisar lembrar de `qs -p ~/.config/mk/shell ipc call mare …` nem de `mk-update`.

## Como funciona

| Comando | O que faz | Por baixo |
|---|---|---|
| `mk atualizar` | o Mk, o Arch e o AUR | `mk-update` |
| `mk ajustes [página]` | abre os Ajustes, direto numa página | `ipc call mare settings <página>` |
| `mk papel` | papel de parede | `ipc call mare toggle wallpaper` |
| `mk bloquear` | bloqueia a tela | `ipc call mare lock` |
| `mk boas-vindas` | o passeio pelo Mk, de novo | `ipc call mare welcome abrir` |
| `mk ferramentas` | node, python e CLIs de IA pelo mise | `scripts/mare-devtools` |
| `mk reiniciar-mare` | reinicia a ilha e os painéis | `scripts/mare-start --restart` |
| `mk ficha` | a ficha do fastfetch | `mk-fetch` |
| `mk versao` | a versão do Mk instalada | lê `<raiz>/VERSION` ou `/usr/share/mk/VERSION` |
| `mk seu` | edita o `seu.bash` | `$EDITOR` (padrão `nvim`) |

`mk` sem nada mostra a lista. Há nomes em inglês como sinônimos (`update`, `settings`, `wallpaper`, `lock`, `welcome`, `tools`, `fetch`, `version`) e completação por `Tab`.

## Por quê assim

- **É uma função, não um binário**: mora no `mk.bash` e só existe em shells que o carregaram. Consequência direta: no modo `bin/try-session`, que **não mexe no `~/.bashrc`**, o comando `mk` inteiro não existe ([[Experimentar sem instalar]]).
- **Nomes em português**, como o resto do projeto ([[Convenções]]).
- **Tudo passa pelo IPC ou por um script que já existe**: o `mk` não tem lógica própria; é um índice.
- **`mk-update` continua existindo como binário** em `/usr/bin`: os Ajustes o chamam direto, e ele funciona mesmo sem o `mk.bash`.

## Onde está no código

`config/bash/mk.bash` (a função `mk` e o `complete -W`).

## Ligações

Depende de: [[Terminal e bash do Mk]], [[IPC do Maré]]. Chama: [[mk-update]], [[Ajustes]], [[Boas-vindas]], [[Ferramentas de desenvolvimento]], [[Fastfetch]].
