---
camada: "08 Aplicativos"
tags:
  - mk
  - aplicativos
  - mkcalc
  - mkcorte
  - mkescrita
fontes:
  - "apps/README.md"
  - "apps/mkcalc/CONTRATO.md"
  - "apps/mkcorte/CONTRATO.md"
  - "apps/mkescrita/CONTRATO.md"
  - "system/mk/apps-links"
  - "migrations/system/20261201-mkcalc-mkcorte-mkescrita.sh"
  - "docs/limites.md"
---
# MkCalc, MkCorte e MkEscrita

Três apps de uso diário, no molde de [[Apps do Mk (motor em Go, janela em QML)]]
(decisão [[D17 Apps próprios em Go e QML]]). O código mora em `apps/mkcalc`,
`apps/mkcorte` e `apps/mkescrita`; cada pasta tem o seu `CONTRATO.md`.

## MkCalc (`mkcalc`)

**Para quê:** fazer contas sem abrir uma planilha.

**Por que um motor em Go para uma calculadora:** para a conta ser exata. O
motor calcula com frações (`math/big`), então `0,1 + 0,2` é `0,3` e um número
grande não perde dígito. O que não tem fim (uma dízima, uma raiz) sai
arredondado e marcado como aproximado. O motor fica de pé (`servir`) e a
janela pede a prévia a cada tecla, sem abrir processo.

O percentual é o de calculadora de mesa: `200 + 10%` é `220`. As contas
confirmadas ficam em `~/.local/state/mk/mkcalc/historico.json`. É o único dos
três que flutua ([[Regras de janela]]).

## MkCorte (`mkcorte`)

**Para quê:** tirar o começo, o fim ou um pedaço de um vídeo, juntar dois, e
exportar, sem um editor de vídeo inteiro.

**Como pensa:** uma edição é uma lista de clipes, cada um um trecho de um
vídeo de origem. Dividir, aparar, remover, mover e juntar são operações sobre
essa lista (em `qml/Corte.js`, testadas sem janela), com desfazer. Nada toca
o vídeo até exportar.

**Dois modos de exportar, e por quê:** recodificar dá o corte no quadro
exato, mas demora; copiar o vídeo como está leva segundos e não perde
qualidade, mas só pode cortar num quadro-chave. O app oferece os dois e diz
a diferença, em vez de escolher calado. A saída é sempre um arquivo novo.

A janela mostra o vídeo pelo QtMultimedia, por isso o Mk passou a trazer o
`qt6-multimedia` e o `qt6-multimedia-ffmpeg`.

## MkEscrita (`mkescrita`)

**Para quê:** escrever em Markdown com a tela limpa.

**O que o motor garante:** não perder texto. Salvar é escrever um arquivo
novo ao lado e trocá-lo de uma vez; se o arquivo mudou por fora desde que foi
aberto, o motor não grava e a janela pergunta. O que ainda não tem nome nunca
é salvo sozinho (o app não escolhe nome por ninguém): vira rascunho em
`~/.local/state/mk/mkescrita/rascunhos/` e é oferecido na abertura seguinte.

A pré-visualização, o sumário e o exportar HTML vêm de um conversor de
Markdown do próprio motor, que escapa tudo o que vem do texto.

## Limites

Em [[Limitações conhecidas]] e em `docs/limites.md`: o que foi visto na tela e
o que só rodou nos testes.
