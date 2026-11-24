---
camada: "07 Aparência"
tags:
  - mk
  - aparência
  - tema
  - cores
fontes:
  - "shell/mare/Theme.qml"
  - "shell/mare/Motion.qml"
  - "config/hypr/mare/look.lua"
  - "docs/desenvolvimento.md"
  - "system/mk/look/themes/Mk/gtk-4.0/gtk.css"
  - "config/ghostty/config"
---

# Direção Folha

"Folha" é o nome da linguagem visual do Mk: preto quente que nunca é preto puro, texto em tons de osso, linhas finas e **um único destaque**, o cobre. O motivo recorrente é a folha de desenho técnico.

## Para quê

Uma paleta e um conjunto de regras que valem em tudo: boot, login, ilha, painéis, janelas, apps GTK e Qt, terminal.

## Como funciona

A paleta-base, igual em todos os lugares:

| Nome | Valor | Onde |
|---|---|---|
| preto quente (`ground`) | `#0b0b0b` | fundo de tudo |
| osso (`text`) | `#cdc4ba` | texto e linhas |
| cobre (`copper`) | `#ce5e32` | o único destaque |

No Maré (`Theme.qml`):

- **Superfícies**, do fundo ao mais elevado: `ground` `#0b0b0b`, `surfaceLow` `#100f0e`, `surface` `#141211`, `raised` `#1b1917`;
- **Linhas**: o osso em transparência: `hairline` 13%, `line` 26%, `wash` 5%;
- **Texto**: `text`, `textSoft` `#b0a9a0`, `textDim` `#958f87`, `textFaint` `#716b69`;
- `accentFor(nome)` devolve sempre o cobre.

Onde a mesma paleta é repetida, à mão, fora do `Theme.qml`:

| Lugar | Arquivo |
|---|---|
| menu de boot | `limine_header` no instalador ([[Limine]]) |
| boot | `mk.script` ([[Plymouth]]) |
| login | `Main.qml` do tema ([[SDDM]]) |
| janelas | `look.lua`: borda `rgba(cdc4ba42)` / `rgba(cdc4ba21)`, fundo `0x0B0B0B` |
| apps GTK e Qt | [[Tema GTK, ícones e Qt]] |
| terminal | [[Ghostty]], [[Fastfetch]], `starship.toml`, as cores do fzf e do ble.sh |
| instalador e `mk-update` | sequências ANSI `38;2;206;94;50` etc. |

Regras de forma: cantos retos nas janelas (`rounding = 0`), sem sombra e sem *blur*; rótulos em mono maiúsculo espaçado; números e títulos em serifa; um grão fino por cima do fundo ("o preto nunca é preto").

Movimento: a curva `cubic-bezier(.215, .61, .355, 1)`, rápida no início e assentando no fim ([[Animações]]).

## Por quê assim

- **Um destaque só**: o `Theme.qml` declara o cobre como "o único destaque". É uma escolha estética da direção; o código não a justifica além disso.
- **Projetos por nome e letra, não por cor**: uma versão anterior dava uma cor a cada projeto; `accentFor` ficou "para os painéis que pedem uma cor por projeto" e passou a devolver sempre o cobre.
- **A paleta é repetida, não importada**, porque cada peça roda num lugar diferente (o limine, o plymouth, o sddm, o GTK, o terminal) e nenhuma lê o `Theme.qml`. Não há script que confira as cópias; mudar a paleta é mudar vários arquivos.
- **No Maré, nunca escrever cor ou fonte direto num painel**: sempre pelos tokens do `Theme`.

> [!note] Fora da paleta
> O `hyprlock.conf` de reserva ainda usa as cores de antes da Folha ([[Ociosidade e bloqueio]]).

## Onde está no código

`shell/mare/Theme.qml`, `shell/mare/Motion.qml`, `config/hypr/mare/look.lua`; a seção "Marca e arte" de `docs/desenvolvimento.md`.

## Ligações

Usado por: [[Maré]], [[Ilha]], [[Plymouth]], [[SDDM]], [[Tema GTK, ícones e Qt]], [[Ghostty]], [[Fastfetch]], [[Papéis de parede]], [[Arte e geradores]], [[Fontes]].
