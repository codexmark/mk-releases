---
camada: "11 Desenvolvimento e qualidade"
tags:
  - mk
  - desenvolvimento
  - checagens
  - qualidade
fontes:
  - "bin/check-keybinds"
  - "bin/check-packages"
  - "bin/check-shell"
  - "shell/mare/Keybinds.js"
  - "docs/desenvolvimento.md"
  - "bin/mk-release"
  - "tests/lint.test.sh"
---

# Checagens

Além da suíte de testes, três scripts pequenos em `bin/` conferem coisas pontuais: que as duas listas de atalhos são iguais, que cada pacote existe onde deveria e que o bash do Mk abre sem cair.

## Para quê

São travas baratas, com um alvo cada. Duas delas, mais o `tests/run`, barram o lançamento.

## Como funciona

| O quê | Confere | Barra o lançamento? |
|---|---|---|
| `tests/run` | a suíte inteira, com o lint ([[Testes e TDD]]) | sim |
| `bin/check-keybinds` | `binds.lua` e `Keybinds.js` têm as mesmas ações (id, grupo, descrição, combo, marcas, na mesma ordem), as mesmas famílias e as mesmas teclas de emergência | sim |
| `bin/check-shell` | um bash interativo de verdade, com o `mk.bash` do repositório, sobe e sai sem cair (em `xterm-256color` e `xterm-ghostty`) | sim |
| `bin/check-packages` | cada pacote de `base`, `desktop`, `dev`, `apps` e `hardware` existe em `core`/`extra`/`multilib`; cada um de `aur` existe no AUR **e não** nos oficiais | não |

- **`check-keybinds`**: o comando de verificação mora **no cabeçalho do `Keybinds.js`** (uma linha de `node -e`); o script só o extrai e roda. Sai com 1 se divergirem e com 2 se faltar o `node`.
- **`check-packages`**: usa `pacman -Si` (vale a base sincronizada da máquina) e a API do AUR. Não instala nada.
- **`check-shell`**: usa `script(1)` para dar um pseudo-terminal ao bash; `bash -i -c` não pegaria o erro, porque sem terminal o ble.sh não carrega.

O lint (`tests/lint.test.sh`) cobre o que antes era só convenção: sintaxe de bash, QML, JavaScript, Lua, JSON e Python; shellcheck sem avisos; bit de execução; e `rm -rf` com variável sempre protegido por `${VAR:?}`.

### Antes de lançar

A lista de `docs/desenvolvimento.md`, resumida:

1. `git status` limpo; nada do que a mudança precisa ficou sem `git add`.
2. `tests/run` passa inteiro, e a mudança chegou com o teste dela; `check-keybinds` e `check-shell` passam (o `mk-release` roda os três de novo).
3. Mexeu em listas? `check-packages` passa.
4. Mexeu no Maré? `dev-shot` dá "log do Maré: limpo", e a mudança foi vista numa sessão de verdade.
5. Mexeu fora de `<raiz>`? Está no instalador **e** numa migração `AAAAMMDD-assunto.sh`.
6. A migração precisa de pacote? Está na linha `# pacotes:`, e ela aguenta o pacote faltar.
7. Mexeu no instalador, no boot, no login ou numa migração? Passou por uma VM.
8. Mexeu na marca? Os geradores foram rodados e as imagens estão no commit.
9. Nenhum arquivo rastreado traz senha, chave, e-mail ou caminho pessoal.
10. Os assuntos dos commits servem para leitura pública? Senão, passe uma frase ao `mk-release`.
11. A documentação diz o que o código faz agora.
12. `git push` para o repositório privado.

## Por quê assim

- **Cada checagem nasceu de um defeito ou de uma duplicação assumida**: o `check-shell`, de um `mk.bash` lançado que derrubava todo terminal; o `check-keybinds`, da decisão de manter a lista de atalhos em dois arquivos ([[D12 Atalhos em dois lugares]]).
- **O comando do `check-keybinds` mora no arquivo que ele confere**: quem edita o `Keybinds.js` vê a regra no alto dele.
- **`check-packages` não barra o lançamento**: o `mk-release` não o chama, e o motivo não está registrado. O que o código mostra é que ele precisa de rede (a API do AUR) e depende da base de pacotes sincronizada da máquina de quem roda, e os testes não saem para a rede.
- **Duplicações sem checagem dedicada**: a paleta ([[Direção Folha]]) e as listas `needed` do `try-session` e do `new-user`.

## Onde está no código

`bin/check-keybinds` (11 linhas), `bin/check-packages`, `bin/check-shell`; o cabeçalho de `shell/mare/Keybinds.js`; `tests/lint.test.sh`.

## Ligações

Usado por: [[mk-release]], [[Trabalhar no repositório]]. Confere: [[Atalhos]], [[Listas de pacotes]], [[Terminal e bash do Mk]]. Complementa: [[Testes e TDD]].
