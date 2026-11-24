---
camada: "12 Decisões e limites"
tags:
  - mk
  - decisões
  - aplicativos
  - go
fontes:
  - "apps/README.md"
  - "apps/mkpulso/CONTRATO.md"
  - "apps/mkpacotes/CONTRATO.md"
  - "apps/mkutil/CONTRATO.md"
  - "apps/mkpulso/pack"
  - "shell/mare/SystemPanel.qml"
  - "docs/limites.md"
---

# D17 Apps próprios em Go e QML

O Mk passa a ter os próprios utilitários (monitor de sistema, pacotes, instalar, comprimir), escritos num molde só: motor em Go, janela em QML. Entram no lugar de ferramentas de terminal e de apps de terceiros.

## Contexto

Até aqui o Mk cobria essas funções com o que já existia: o `btop` num terminal como monitor (o painel Sistema tinha um atalho para ele), o `pacman` e o `yay` na linha de comando, nada para um AppImage baixado, o `ffmpeg` à mão. O único app próprio era o MkTunes, feito em outro repositório, com um núcleo em Java que exige uma JVM instalada ([[MkTunes]]). E a lista de apps opcionais deixava "monitor de sistema" de fora de propósito.

## Escolha

- **Apps próprios, dentro do repositório do Mk**: `apps/mkpulso`, `apps/mkpacotes`, `apps/mkutil`. "Ao contrário do MkTunes, o código mora aqui."
- **Motor em Go, só com a biblioteca padrão**, compilado em binário estático que vai pronto com o Mk.
- **Janela do Quickshell**, com os tokens do Maré, que só mostra o que o motor diz.
- **Contrato escrito** entre os dois, mudado primeiro.
- **Segurança no motor**: recusas com código de saída próprio; root só pelo polkit.
- O painel Sistema do Maré passa a abrir o MkPulso.

## Alternativas

- **Continuar com ferramentas de terminal e apps de terceiros**: era o estado anterior.
- **Python, Java ou Rust para o motor**: segundo quem mantém, o Go foi escolhido pelo binário único, pela pouca memória, pelo build em segundos, e pela decisão de evitar Python. Esse raciocínio **não está escrito no repositório**; o que está registrado é o resultado (binário estático, sem dependências, sem pacote novo).
- **Aproveitar código de outro desktop**: também segundo quem mantém, as ideias vieram do Ryoku, sem copiar código, que é GPL-3 e de outra pessoa. O repositório cita o Ryoku como inspiração em outros pontos (os Ajustes, a ilha); para estes apps, a ressalva de licença não está escrita no código.
- **Lógica na janela (QML/JavaScript)** *(não discutida no repositório)*: os contratos põem "toda a lógica" no motor, o lado testável sem tela.

## Consequência

- Quatro apps novos chegam a um Mk instalado sem pacote novo: links e uma pasta que o `mk-update` já troca.
- **Binários no git** (alguns megabytes cada), e a obrigação de recompilar e fazer o commit do binário junto com o código; o `tests/run` reprova um binário velho. O `go` já está em `dev.packages`.
- O instalador precisou passar a devolver o bit de execução também a binários compilados (`fix_exec` reconhece ELF), porque o `mkarchiso` grava tudo como 644 ([[Instalador passo a passo]]).
- A suíte ganhou `tests/apps/`: os testes em Go de cada motor, a estrutura das janelas, os lançadores e as contas em JavaScript.
- O MkTunes fica como exceção: continua em Java, e o porte para Go espera o repositório dele.
- As janelas são código novo e pouco visto: os botões nunca foram clicados de verdade ([[Limitações conhecidas]]).
- Mais superfície para manter: três contratos, três motores, cinco janelas.

## Onde está no código

`apps/README.md`; `apps/*/CONTRATO.md`; `apps/*/pack`; `shell/mare/SystemPanel.qml`.

## Ligações

Afeta: [[Apps do Mk (motor em Go, janela em QML)]], [[MkPulso]], [[MkPacotes]], [[MkInstalar e MkComprimir]], [[MkTunes]], [[Painéis]], [[Testes e TDD]]. Índice: [[Registro de decisões]].
