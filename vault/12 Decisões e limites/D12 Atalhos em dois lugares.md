---
camada: "12 Decisões e limites"
tags:
  - mk
  - decisões
  - atalhos
  - compositor
fontes:
  - "config/hypr/mare/binds.lua"
  - "shell/mare/Keybinds.js"
  - "bin/check-keybinds"
  - "bin/mk-release"
---

# D12 Atalhos em dois lugares

A lista de atalhos existe duas vezes, em `binds.lua` e em `Keybinds.js`, e um script barra o lançamento se elas divergirem.

## Contexto

Dois programas precisam da mesma lista: o Hyprland (Lua), para criar os binds, e o Maré (JavaScript/QML), para a página Atalhos, a busca dos Ajustes e as boas-vindas. O Lua da config roda dentro do compositor; o QML não consegue importá-lo, e o Hyprland não lê JavaScript.

## Escolha

- Manter as duas listas, com a mesma forma: mesmos `id`, na mesma ordem, com o mesmo `group`, `desc`, `combo` e marcas; as mesmas `families`; a mesma lista `everywhere` (teclas válidas para a emergência).
- A do Lua tem a mais o `run` (o que o atalho faz).
- `bin/check-keybinds` compara as duas; o comando de comparação mora no cabeçalho do `Keybinds.js`.
- `bin/mk-release` roda a checagem e não lança se ela falhar.
- Regras de validação também são duplicadas e apontadas uma para a outra nos comentários: `reachable` (Lua) e `validate` (JS); a ordem de quem escolhe primeiro (Lua) e `clean` (JS).

## Alternativas

- **Uma fonte só, gerando a outra** (um JSON lido pelos dois, ou um gerador): não foi feito; o repositório não registra por quê. Seria um passo de build num projeto que não tem nenhum.
- **O Maré ler os binds do Hyprland em tempo de execução** (`hyprctl -j binds`): a página faz isso para **conferir** o resultado, mas a saída do Hyprland não traz grupo, marcas nem família.

## Consequência

- Mexer num atalho é mexer em dois arquivos; esquecer um é pego antes do lançamento, não depois.
- A checagem precisa do `node`.
- O formato de cada linha do `binds.lua` é rígido (a checagem lê com expressão regular): uma ação fora do formato "de uma linha" é acusada.
- É o modelo citado nas convenções: "dois lugares para a mesma lista, um script que confere". Outras duplicações do projeto não têm esse script ([[Checagens]]).

## Onde está no código

`config/hypr/mare/binds.lua`, `shell/mare/Keybinds.js` (o cabeçalho com o comando), `bin/check-keybinds`.

## Ligações

Afeta: [[Atalhos]], [[Atalhos do usuário]], [[Checagens]], [[mk-release]]. Índice: [[Registro de decisões]].
