---
camada: "08 Aplicativos"
tags:
  - mk
  - aplicativos
  - mkporto
  - ssh
fontes:
  - "apps/README.md"
  - "apps/mkporto/CONTRATO.md"
  - "system/mk/apps-links"
  - "migrations/system/20261203-mkporto.sh"
  - "docs/limites.md"
---
# MkPorto

O catálogo das conexões SSH (`mkporto`), no molde de [[Apps do Mk (motor em Go, janela em QML)]]
(decisão [[D17 Apps próprios em Go e QML]]). O código mora em `apps/mkporto`; o
`CONTRATO.md` de lá diz o que o motor responde.

## O que faz

- **Hosts**: um leitor próprio do `~/.ssh/config` e dos `Include` diz quais apelidos existem e de que arquivo vêm; os valores (endereço, usuário, porta, chave, salto) são os do `ssh -G`, então `Match` e o config do sistema valem. Se o `ssh -G` falhar, ficam os do leitor e o detalhe avisa.
- **Estado**: conecta em host:porta e lê a saudação (`SSH-2.0-...`): mostra a versão do servidor, ou "a porta abre, mas não é um servidor ssh". Host com `ProxyJump` é sondado pelo salto (`ssh -W`, em `BatchMode`: nunca pede senha nem aceita chave de host sozinho).
- **Testar entrada** (`Ctrl+T`): `ssh` em `BatchMode` rodando `true`; diz se uma das suas chaves entra, ou por que não.
- **Ações**: conectar (`Enter`), copiar o comando, enviar uma chave pública (`Ctrl+K`), editar (`Ctrl+E`) e remover (`Delete`).
- **Chaves**: as `.pub` de `~/.ssh`, com tipo e impressão digital, "copiar a pública" e "nova chave" (`Ctrl+G`): um `ssh-keygen -t ed25519` num terminal, onde a frase-senha é digitada.

## As regras que não mudam

- **Não guarda nem pede senha, e nunca abre chave privada.** Conectar é abrir um terminal com `ssh -- <apelido>`; quem autentica é o `ssh`, com o agente ([[Chaveiro e agente SSH]]).
- **A janela não monta comando.** Ela roda o argv que o motor devolve em `comando`, e o motor só devolve para apelidos de `[A-Za-z0-9._-]` que não começam com `-`.
- **Só mexe no que é dele.** Os hosts criados pelo app ficam em `~/.ssh/config.d/mkporto.conf` (600, reescrito inteiro e de forma atômica). Na primeira gravação entra `Include config.d/mkporto.conf` no topo do `~/.ssh/config`; o resto fica byte a byte. Host do config da pessoa aparece, mas editar e remover são recusados.
- **Validação contra injeção**: nenhum campo aceita quebra de linha, aspas ou começar com `-`, para não virar opção do `ssh` nem linha nova no config.

## De onde veio

A ideia veio do Ryoport, do Ryoku (SSH, servidores e máquinas virtuais num lugar só). Só a ideia: o código é do Mk. Ficou no SSH; máquinas virtuais dependem de decidir entre libvirt e QEMU direto.

Limites: [[Limitações conhecidas]] e `docs/limites.md`.
