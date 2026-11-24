---
camada: "11 Desenvolvimento e qualidade"
tags:
  - mk
  - desenvolvimento
  - testes
  - maré
fontes:
  - "bin/dev-shot"
  - "docs/desenvolvimento.md"
  - "shell/mare/README.md"
---

# dev-shot

`bin/dev-shot` testa o Maré sem mexer na sua tela: sobe um Hyprland aninhado, roda nele uma **cópia** do shell, manda os comandos de IPC pedidos, tenta uma captura e mostra os erros do log.

## Para quê

É o teste mais rápido para "o que eu escrevi carrega?", e o jeito de ver um painel sem arriscar a sessão em uso.

## Como funciona

```bash
bin/dev-shot <saida.png> "settings monitores"
bin/dev-shot <saida.png> "toggle launcher" "close" "toggleAt volume 800"
MK_DEV_HEADLESS=1080x1920 bin/dev-shot <saida.png> "toggle launcher" "tecla -k Escape" "state"
```

1. Copia o shell para uma pasta temporária em `$XDG_RUNTIME_DIR`.
2. Escreve uma config mínima do Hyprland (um monitor, as curvas de `motion.lua`, o Maré com `QT_SCALE_FACTOR=0.85`) e sobe o Hyprland aninhado.
3. Acha a instância por `hyprctl -j instances` e espera o Quickshell dizer "Configuration Loaded".
4. Para cada argumento:
   - texto comum → `qs -p <cópia> ipc call mare <texto>`;
   - `tecla …` → vai para o `wtype` **dentro** do aninhado;
   - `hypr <lua>` → um *dispatch* ao Hyprland aninhado.
5. Tenta a captura com o `grim`.
6. Filtra o log do Maré atrás de erros de QML (`TypeError`, `ReferenceError`, `not defined`…).
7. Mata o aninhado e apaga a pasta temporária.

**Saída**: 0 se o log está limpo, 1 se há avisos, 2 se o aninhado não subiu.

| Variável | Para quê | Padrão |
|---|---|---|
| `MK_DEV_SHELL` | a pasta do shell a copiar | `shell/mare` |
| `MK_DEV_WAIT` | segundos depois de cada comando | `1.5` |
| `MK_DEV_SIZE` | tamanho da tela aninhada | `1600x900` |
| `MK_DEV_HEADLESS` | cria uma segunda tela, virtual, desse tamanho | |
| `MK_DEV_EXEC` | abre um app dentro do aninhado | |
| `MK_DEV_WELCOME` | mostra as boas-vindas | vazio |

## Por quê assim

- **Uma cópia do shell**: os ajustes mexidos no teste ficam na cópia e somem no fim; o `ajustes.json` de quem testa não é tocado.
- **Sempre do zero**: como sobe um Quickshell novo, pega o caso do singleton novo que o recarregamento a quente não registra ([[Armadilhas do Maré]]).
- **A janela do aninhado pode ser escondida** com uma regra temporária no Hyprland de quem testa. Escondida, o compositor não desenha quadros novos e a captura não sai ("captura: indisponível"); o log continua valendo. Para ver a imagem, roda-se com a janela à vista.
- **A tela virtual desenha mesmo escondida**: foi acrescentada para conferir o encaixe do painel em telas de larguras diferentes ([[Monitor por tela]]).
- **Teclas e *dispatch* dentro do aninhado**, nunca na sessão de quem testa: para testar o teclado dos painéis sem digitar na própria sessão.
- **O que não cobre**: a config do Hyprland do Mk. Atalhos, regras de janela e o `autostart.lua` não são carregados. Por isso também não inicia o `hypridle`.
- **Precisa rodar dentro de uma sessão Hyprland** (usa `hyprctl instances`), com `Hyprland`, `qs`, `jq` e `grim`.

## Onde está no código

`bin/dev-shot` (107 linhas; o cabeçalho lista as variáveis).

## Ligações

Depende de: [[IPC do Maré]], [[Hyprland em Lua]], [[Animações]]. Usado por: [[Trabalhar no repositório]], [[Testes e TDD]]. Relacionado: [[Armadilhas do Maré]].
