---
camada: "07 Aparência"
tags:
  - mk
  - aparência
  - marca
  - arte
  - python
fontes:
  - "system/mk/art/make-logo.py"
  - "system/mk/art/make-art.py"
  - "system/mk/art/make-wallpapers.py"
  - "docs/desenvolvimento.md"
---

# Arte e geradores

As imagens da marca (o logo, a marca compacta, o emblema, o grão e os doze papéis de parede) são geradas por três scripts Python em `system/mk/art/`. O que eles geram **está no git**.

## Para quê

Uma origem só para todas as cópias do logo espalhadas pelo sistema, e papéis de parede que não dependem de imagens de terceiros.

## Como funciona

| Script | O que gera |
|---|---|
| `make-logo.py` | o logo em vetor (o "Mk" em serifa reta sobre um plano cartesiano), com as letras tiradas do contorno da própria fonte; SVG e PNG em `art/logo/`, e as cópias que o resto usa |
| `make-art.py` | `emblem.png` (arco, colunas, sol de cobre) e `grain.png`; usa a marca do `make-logo.py` |
| `make-wallpapers.py` | os doze papéis de parede em `shell/mare/wallpapers/*.jpg` |

As cópias que o `make-logo.py` grava:

| Arquivo | Quem usa |
|---|---|
| `shell/mare/assets/mark.png`, `mark.svg` | o Maré (ilha, bloqueio, Ajustes, boas-vindas) |
| `config/fastfetch/mark.png` | a ficha do terminal |
| `system/mk/sddm/mk/letra-m.png`, `letra-k.png`, `geometria.js` | a tela de login (as letras em camadas e a geometria da marca) |
| `system/mk/plymouth/mk/logo.png` | o boot |
| `system/mk/art/mark.png` | o `make-art.py` |

**Dependências**: `python` com `fontTools`, Pillow e `numpy`, e o `rsvg-convert`, que **não estão em lista nenhuma do Mk**; e as fontes instaladas (Fraunces e Space Mono Nerd, abertas por caminho fixo em `/usr/share/fonts/TTF/`).

## Por quê assim

- **O resultado no git**: "ninguém precisa rodá-los para instalar ou lançar". Os geradores só rodam quando o desenho muda, e o commit leva o que saiu. Assim a instalação não depende de Pillow, numpy nem das fontes.
- **Gerar por código**: o logo sai dos pontos reais do contorno da fonte, e os papéis de parede falam a mesma língua (traço fino, formas de desenho técnico, um ponto de cobre).
- **Ordem**: `make-logo.py` antes de `make-art.py`.
- **Cópias que o gerador não atualiza**: o `grain.png` também existe em `shell/mare/assets/` e em `system/mk/sddm/mk/`, e o `make-art.py` não as regrava. É uma dívida registrada em `docs/desenvolvimento.md`.
- **Mudou a marca num Mk instalado?** O que está em `<raiz>` chega pelo `mk-update`; o boot e o login, por migração ([[Catálogo de migrações]], `20261121-logo-plano.sh`).

> [!note] `__pycache__`
> Há uma pasta `system/mk/art/__pycache__/` no diretório de trabalho; não é rastreada pelo git.

## Onde está no código

`system/mk/art/` e a seção "Marca e arte" de `docs/desenvolvimento.md`.

## Ligações

Depende de: [[Fontes]], [[Direção Folha]]. Usado por: [[Plymouth]], [[SDDM]], [[Fastfetch]], [[Papéis de parede]], [[Maré]].
