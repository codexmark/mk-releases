---
camada: "12 Decisões e limites"
tags:
  - mk
  - decisões
  - adr
fontes:
  - "docs/desenvolvimento.md"
  - "docs/atualizacoes.md"
  - "shell/mare/README.md"
  - "migrations/README.md"
---

# Registro de decisões

O índice das decisões de projeto que moldaram o Mk. Cada uma tem a sua nota, no mesmo formato: **contexto**, **escolha**, **alternativas**, **consequência**.

## Para quê

As notas das camadas dizem como cada peça funciona. Estas dizem por que o sistema tomou um caminho e não outro, com a fonte: um comentário do código, um trecho da documentação ou a mensagem do commit que fez a mudança.

## Como funciona

| Nº | Decisão | Camada |
|---|---|---|
| D01 | [[D01 Maré fora da pasta do Quickshell]] | sessão, shell |
| D02 | [[D02 Um só ponto fala com o Hyprland]] | compositor, shell |
| D03 | [[D03 Migração não instala pacote]] | atualização |
| D04 | [[D04 Canal público separado]] | atualização |
| D05 | [[D05 Hibernação retirada]] | disco, energia |
| D06 | [[D06 Maré como agente de polkit]] | sessão |
| D07 | [[D07 Teclado exclusivo no lugar do grab]] | shell |
| D08 | [[D08 Geometria por monitor]] | shell |
| D09 | [[D09 Menu de boot dos ganchos]] | boot |
| D10 | [[D10 AUR compilado na ISO]] | instalação |
| D11 | [[D11 Ajustes como dados]] | shell |
| D12 | [[D12 Atalhos em dois lugares]] | compositor, shell |
| D13 | [[D13 Monitores por arquivo com vigia]] | compositor |
| D14 | [[D14 Tema por variável de sessão]] | aparência |
| D15 | [[D15 Teste primeiro e CI local]] | desenvolvimento |
| D16 | [[D16 Descanso de tela no lugar do suspender]] | sessão, energia |
| D17 | [[D17 Apps próprios em Go e QML]] | aplicativos |

O que **não** virou nota de decisão, porque o repositório não registra o motivo, está em [[Motivos não registrados]]. O que se sabe que não funciona ou não foi testado está em [[Limitações conhecidas]].

## Por quê assim

- **Uma nota por decisão**, e não uma seção numa nota grande: cada decisão é ligada a partir das notas de camada que ela afeta, e o grafo mostra quais decisões pesam em mais lugares.
- **Como ler a seção "Alternativas".** O que o repositório registra vem citado ou atribuído (um comentário, um commit, o estado anterior do código). O que está marcado *(não discutida no repositório)* é a alternativa que a escolha exclui, apontada por esta vault para dar o contraste; o projeto não a debateu por escrito.
- **Só entra o que tem fonte.** Várias escolhas grandes do Mk (o limine, o btrfs, o sddm) não têm justificativa escrita em lugar nenhum do repositório; inventar uma seria pior que dizer que não há.
- **Um padrão que se repete**: quase todas nasceram de um defeito observado (numa VM, na primeira instalação em máquina real, no uso diário), não de um plano prévio. O histórico de commits do projeto é curto e denso, e as mensagens costumam dizer o que quebrou.

## Onde está no código

As fontes de cada decisão estão no frontmatter da nota dela. As mais ricas: os comentários de `bin/mk-update`, `shell/mare/Monitor.qml`, `shell/mare/scripts/mare-monitors`, `config/hypr/mare/paths.lua` e a seção "Armadilhas" do `shell/mare/README.md`.

## Ligações

[[Início]] · [[Motivos não registrados]] · [[Limitações conhecidas]] · [[Convenções]]
