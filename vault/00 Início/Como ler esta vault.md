---
camada: "00 Início"
tags:
  - mk
  - moc
fontes:
  - "README.md"
  - "docs/desenvolvimento.md"
---

# Como ler esta vault

Esta nota explica como a vault está organizada e que garantias ela dá sobre o que afirma.

## Para quê

Para alguém técnico que nunca viu o Mk conseguir entender o sistema inteiro sem abrir o código, e saber onde abrir quando precisar.

## Como funciona

- **Uma pasta por camada**, numerada de `00` a `12`. A ordem é do núcleo para a superfície: o que está numa pasta depende do que está nas de número menor.
- **Uma nota por assunto**, curta. Cada uma tem o mesmo esqueleto:
  - parágrafo de abertura: o que é;
  - **Para quê**: para que serve e quem usa;
  - **Como funciona**: o mecanismo;
  - **Por quê assim**: a razão da escolha e a alternativa descartada;
  - **Onde está no código**: caminhos reais, relativos à raiz do repositório;
  - **Ligações**: do que depende (camadas de baixo) e quem a usa (camadas de cima).
- **Frontmatter** em cada nota: `camada`, `tags` e `fontes` (os arquivos do repositório que sustentam o que a nota diz).
- **Diagramas Mermaid** onde um fluxo é mais claro que prosa: [[Início]], [[Fluxo de boot]], [[mk-update]], [[Maré]], [[Instalador passo a passo]], [[Monitores]].

## Por quê assim

Tudo o que a vault afirma foi lido no código, na documentação do repositório ou nas mensagens de commit. Quando o código faz uma escolha sem dizer o motivo, a nota descreve o que a escolha faz e marca **motivo não registrado**; essas marcas estão reunidas em [[Motivos não registrados]]. Onde a documentação e o código divergem, a nota segue o código e aponta a divergência (veja [[Limitações conhecidas]]).

> [!note] Caminhos
> `<raiz>` nas notas é a pasta do usuário onde o Mk fica instalado, normalmente `~/.config/mk` (veja [[Pasta do usuário]]). `/usr/share/mk` é a cópia do repositório num sistema instalado.

## Onde está no código

O ponto de partida no repositório é o `README.md`, com `docs/desenvolvimento.md` e `docs/atualizacoes.md` para quem mantém. A correspondência entre pastas e camadas está em [[Mapa do repositório]].

## Ligações

Volte ao mapa em [[Início]]. Termos em [[Glossário]].
