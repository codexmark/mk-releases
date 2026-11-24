---
camada: "05 Compositor"
tags:
  - mk
  - compositor
  - movimento
fontes:
  - "tests/config/motion.test.lua"
  - "config/hypr/mare/motion.lua"
  - "shell/mare/SettingsSchema.js"
  - "shell/mare/Motion.qml"
---

# Animações

As animações das janelas e dos workspaces usam a mesma curva do shell, e uma escolha só nos Ajustes ("Velocidade") escala as duas coisas juntas.

## Para quê

Para o movimento das janelas e o dos painéis parecerem do mesmo sistema.

## Como funciona

`motion.lua` define duas curvas e nove animações:

- `mare`: `cubic-bezier(.215, .61, .355, 1)`, rápida no início e assentando no fim;
- `mareOut`: `(0.4, 0) (1, 1)`, para a saída.

| Folha | speed | Curva | Estilo |
|---|---|---|---|
| `global`, `windows` | 2.4 | mare | |
| `windowsIn` | 2.0 | mare | `popin 95%` |
| `windowsOut` | 1.6 | mareOut | `popin 94%` |
| `border` | 2.0 | mare | |
| `fade` | 1.7 | mare | |
| `layers` | 2.0 | mare | `fade` |
| `workspaces` | 2.8 | mare | `slide` |
| `specialWorkspace` | 2.4 | mare | `slidevert` |

A unidade de `speed` do Hyprland é 100 ms: 2.4 = 240 ms.

**A Velocidade dos Ajustes** (`motion.speed`: 0.7 rápido, 1 normal, 1.4 calmo, 0 sem):

- no Maré, `Motion.qml` multiplica todas as durações por ela;
- no Hyprland, `animationCode()` em `SettingsSchema.js` reemite a lista inteira com os `speed` multiplicados, e ela vai para o `ajustes.lua`. Com "sem", só `animations.enabled = false`.

## Por quê assim

- **A mesma curva nos dois lados**: é a curva da [[Direção Folha]], declarada em `motion.lua` e em `Motion.qml`.
- **A lista está duplicada** em `motion.lua` e em `SettingsSchema.js` (`animations`). O comentário dos dois arquivos é o mesmo aviso: "mudou uma folha ou um número aqui, mude lá". Quem confere é um teste, `tests/config/motion.test.lua`, que lê a lista do esquema e a compara com o que o `motion.lua` pede ao Hyprland.
- **Só fala do assunto quem mexeu** (`ours()`): com a velocidade de fábrica, o `ajustes.lua` não emite animações, e as de um `user.lua` da pessoa "ficam em paz".
- O [[dev-shot]] carrega o `motion.lua` na config mínima do aninhado, porque os Ajustes mandam animações que usam os nomes dessas curvas.

## Onde está no código

- `config/hypr/mare/motion.lua`
- `shell/mare/SettingsSchema.js`: `animations`, `animationCode`, `ours`
- `shell/mare/Motion.qml`

## Ligações

Depende de: [[Hyprland em Lua]], [[Direção Folha]]. Usado por: [[Ajustes]], [[Painel líquido (Tide)]].
