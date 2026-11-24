---
camada: "12 Decisões e limites"
tags:
  - mk
  - decisões
  - atualização
  - canal
fontes:
  - "bin/mk-release"
  - "system/mk/channel"
  - "shell/mare/scripts/mare-updates"
  - "docs/atualizacoes.md"
---

# D04 Canal público separado

O Mk instalado se atualiza de um repositório git **público, só de lançamentos**, diferente do repositório privado onde o trabalho acontece.

## Contexto

O atualizador nasceu apontando para o próprio repositório de desenvolvimento, que é privado. Isso obrigava toda máquina instalada a ter credenciais do GitHub, e a procura automática (que roda sozinha, em segundo plano) corria o risco de pedir senha. A primeira correção foi fazer a procura "nunca pedir senha; avisar quando não tem acesso". Não bastava: sem acesso, não havia atualização.

## Escolha

- Um segundo repositório, público, que recebe só o que passa pelo `bin/mk-release`.
- Cada lançamento é o conteúdo de um commit (`git archive`), mais um `VERSION` (`commit data`) e um `NOVIDADES`.
- A procura lê o `VERSION` com um clone raso que não baixa arquivos, e só aponta versão nova quando ela não é mais antiga que a instalada.
- Os commits públicos usam um e-mail "noreply".

## Alternativas

- **Tornar público o repositório de trabalho**: exporia o histórico inteiro e todo commit intermediário viraria atualização para quem instalou.
- **Empacotar o Mk** (um pacote pacman, um repositório próprio): exigiria infraestrutura de build e assinatura. O Mk não tem pacotes próprios.
- **Manter o privado com credenciais**: descartada pelo contexto acima.

O repositório não discute essas alternativas por escrito; o que está registrado é a sequência de três commits (canal privado, "nunca pede senha", canal público) e o resultado.

## Consequência

- Atualizar não precisa de login.
- Lançar é um ato deliberado; "trabalhando" e "lançado" são coisas diferentes.
- **Tudo o que está rastreado vai a público a cada lançamento**, inclusive a documentação: daí a regra de nenhum dado pessoal em arquivo rastreado ([[Convenções]]).
- Não há desfazer nem voltar de versão: o conserto é outro lançamento.
- O canal é só um endereço git: trocá-lo é escrever outro em `~/.config/mk/channel`.

## Onde está no código

`bin/mk-release`, `system/mk/channel`, `shell/mare/scripts/mare-updates`.

## Ligações

Afeta: [[Canal de lançamentos]], [[mk-release]], [[mk-update]]. Índice: [[Registro de decisões]].
