---
camada: "08 Aplicativos"
tags:
  - mk
  - aplicativos
  - terminal
  - bash
  - blesh
fontes:
  - "config/bash/mk.bash"
  - "config/bash/starship.toml"
  - "config/bash/seu.bash.exemplo"
  - "bin/check-shell"
  - "migrations/user/20261120-terminal.sh"
  - "migrations/system/20261121-bash-mk.sh"
---

# Terminal e bash do Mk

O "terminal do Mk" é uma camada sobre o bash, num arquivo só (`mk.bash`), carregada por uma linha no `~/.bashrc`: ble.sh, histórico, prompt, fzf, atalhos e o comando `mk`.

## Para quê

Um bash confortável logo depois da instalação, sem trocar de shell e sem um framework por cima.

## Como funciona

A linha no `~/.bashrc` (posta pelo instalador, via skel):

```bash
[[ -f ~/.config/mk/bash/mk.bash ]] && source ~/.config/mk/bash/mk.bash
```

O que o `mk.bash` faz, em ordem:

1. Sai se o shell não é interativo.
2. **ble.sh**: carrega cedo (`--noattach`) e liga no fim (`ble-attach`); dá sugestões em cinza enquanto se digita e cores de sintaxe. Não carrega com `TERM=linux`; `MK_NO_BLE=1` desliga.
3. **Histórico**: 50 mil linhas em memória, 100 mil em arquivo, sem repetidos, com data; `history -a` a cada comando.
4. **A ficha**: [[Fastfetch]].
5. **Prompt**: starship com `starship.toml` (a pasta, o git, o tempo do comando quando passa de 2 s); sem starship, um `PS1` simples nas mesmas cores.
6. **Ferramentas**: bash-completion, zoxide, fzf (com `fd`) nas cores Folha.
7. **Atalhos**: `ls`/`ll`/`la`/`lt` (eza), `cat` (bat), `v`, `lg`, `g`, `gs`, `gd`…, `mkcd`, `extract`, `porta`, `servir`.
8. **O [[Comando mk]]**.
9. **Cores do ble.sh** na paleta Folha.
10. **`seu.bash`**: carregado no fim, se existir.

`seu.bash` é do usuário (`<raiz>/seu.bash`, editado com `mk seu`); nenhuma atualização o toca.

## Por quê assim

- **Uma camada só do Mk, no lugar de um oh-my-bash**: "leve e nas cores Folha" (commit que a introduziu).
- **Bash, não zsh ou fish**: o código não registra a razão; o usuário é criado com `/bin/bash` e a camada é construída sobre ele. **Motivo não registrado.**
- **`blesh-git` (0.4), não `blesh` (0.3)**: a 0.4 tem as cores por tipo que o Mk usa. Enquanto os dois existiram, a função `mk_face` escolhia entre `ble-face` (0.4) e `ble-color-setface` (0.3), em silêncio; o pacote 0.3 abria todo terminal com dezenas de erros.
- **Cada comando entra no histórico na hora**: outro terminal já o vê no `Ctrl+R`.
- **`mk.bash` é do Mk e o `mk-update` o troca**; por isso o que é pessoal vai em `seu.bash`.
- **`bin/check-shell`** existe por um acidente: foi lançado um `mk.bash` em que `mk_face` chamava a si mesma com o ble.sh 0.4 e **derrubava todo bash interativo**. O script abre um bash interativo de verdade, num pseudo-terminal (`script`), com `TERM=xterm-256color` e `xterm-ghostty`, e confere que ele sobe e sai. `bash -i -c` não pega esses erros: sem terminal o ble.sh não carrega. O `mk-release` não lança se ele falhar.

> [!warning] `ls` é o eza
> No Mk, `ls -t` não ordena por data como no `ls` do GNU. O `iso/README.md` avisa para não montar nomes de arquivo com `$(ls -t … | head -1)`.

## Onde está no código

`config/bash/mk.bash` (196 linhas), `starship.toml`, `seu.bash.exemplo`; `bin/check-shell`.

## Ligações

Depende de: [[Ghostty]], [[Ferramentas de desenvolvimento]], [[Pasta do usuário]], [[Direção Folha]]. Contém: [[Comando mk]], [[Fastfetch]]. Conferido por: [[Checagens]].
