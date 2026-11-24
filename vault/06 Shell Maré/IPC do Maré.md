---
camada: "06 Shell Maré"
tags:
  - mk
  - maré
  - ipc
  - automação
fontes:
  - "shell/mare/Mare.qml"
  - "shell/mare/Wallpaper.qml"
  - "shell/mare/README.md"
  - "config/hypr/mare/binds.lua"
  - "config/bash/mk.bash"
---

# IPC do Maré

Tudo o que os atalhos, o hypridle e o comando `mk` fazem no shell passa por um canal só: o IPC do Quickshell, com o alvo `mare`.

## Para quê

Dar ao mundo de fora (o compositor, scripts, o terminal) uma forma única e estável de mandar no shell.

## Como funciona

```bash
qs -p ~/.config/mk/shell ipc call mare <função> [argumentos]
```

| Função | Argumentos | O que faz |
|---|---|---|
| `toggle` | `panel` | abre o painel; se já é ele, fecha |
| `toggleAt` | `panel`, `x` | o mesmo, descendo embaixo da posição `x`, como um clique na ilha |
| `close` | | volta para `rest` |
| `state` | | devolve o modo atual |
| `settings` | `page` | abre os Ajustes na página; `""` abre na última |
| `set` | `key`, `value` | muda um ajuste, como o painel faria |
| `switcher` | `direction` | Alt+Tab: `prev` volta; outro valor avança |
| `shot` | `mode` | captura sem painel: `copiar`, `area`, `janela`, `tela` |
| `record` | | começa ou para a gravação da tela |
| `lock` | | bloqueia a tela |
| `welcome` | `action` | `abrir`, `continuar`, `fechar`, `fim`, `seguir`, `voltar`, `estado` |
| `here` | `cmd` | roda um programa no diretório do projeto em foco |
| `volume` | `change` | `+5`, `-5` (em %) ou `mute` |
| `brightness` | `change` | `+5` ou `-5` (em %) |
| `dnd` | | liga ou desliga o não perturbe; devolve o novo valor |

Há um segundo alvo, `wallpaper` (em `Wallpaper.qml`): `set <caminho>`, `random` e `get`.

**Sobre o `set`**: o valor passa por `JSON.parse` (`10`, `false`, `0.85`); o que não for JSON entra como texto. Só aceita chaves que têm valor de fábrica no esquema. **Não confere faixa nem opções**: um valor fora do `min`/`max` é gravado como veio. Com o `ajustes.json` ilegível, não muda nada e ainda assim devolve `chave = valor`, com o valor antigo.

## Por quê assim

- **Um canal para tudo**: o `binds.lua` não sabe como um painel abre; só manda `toggle calendar`. Quem quiser automatizar o Mk usa as mesmas chamadas.
- **`qs -p <pasta>`, não `qs -c mare`**: o Maré é achado pelo caminho da instalação, porque não mora em `~/.config/quickshell` ([[D01 Maré fora da pasta do Quickshell]]).
- **`toggle`, `open` e `openAt` ignoram nomes fora de `Mare.panels`**: um nome errado não deixa o shell num estado inexistente.
- **`toggleAt`** reproduz um clique na ilha numa posição `x`; os exemplos do [[dev-shot]] o usam para testar o encaixe do painel sem mouse.
- **O IPC é a base dos testes manuais**: cada argumento do `dev-shot` é o que viria depois de `ipc call mare`.

## Onde está no código

`shell/mare/Mare.qml` (o `IpcHandler` com `target: "mare"`), `shell/mare/Wallpaper.qml` (o alvo `wallpaper`).

## Ligações

Depende de: [[Singletons]]. Usado por: [[Atalhos]], [[Ociosidade e bloqueio]], [[Comando mk]], [[dev-shot]], [[Impressão]].
