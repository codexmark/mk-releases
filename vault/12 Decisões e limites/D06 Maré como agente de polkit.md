---
camada: "12 Decisões e limites"
tags:
  - mk
  - decisões
  - sessão
  - polkit
fontes:
  - "shell/mare/PolkitDialog.qml"
  - "shell/mare/shell.qml"
  - "system/packages/desktop.packages"
---

# D06 Maré como agente de polkit

O pedido de senha de administrador é desenhado pelo Maré, não pelo `hyprpolkitagent`.

## Contexto

A sessão iniciava o `hyprpolkitagent` no autostart. Na primeira instalação em máquina real, o diálogo dele mostrou dois problemas: só tem **uma linha** para a mensagem (o texto longo era cortado) e os rótulos são em **inglês**, num sistema inteiro em português.

## Escolha

`PolkitDialog.qml`: o Maré registra o agente de polkit da sessão e mostra o pedido como uma folha no meio da tela em foco, em português, com a mensagem inteira (quebra em linhas, nunca corta). A linha que iniciava o `hyprpolkitagent` saiu do autostart.

## Alternativas

- **Manter o `hyprpolkitagent`**: descartado pelos dois problemas.
- **Outro agente pronto** *(não discutida no repositório)*.

## Consequência

- "Só um agente de polkit registra por sessão": com o do Maré no ar, o `hyprpolkitagent` não pode subir. Ele continua instalado, de reserva (`systemctl --user start hyprpolkitagent`).
- Se o Maré não está rodando, **não há agente** até alguém iniciar o de reserva.
- Com a tela bloqueada, o pedido é cancelado na hora (ficaria escondido atrás do bloqueio).
- A folha dos Ajustes cobre a tela inteira; a página Impressoras precisa fechá-la antes de um pedido de senha ([[Impressão]]).

## Onde está no código

`shell/mare/PolkitDialog.qml` (o cabeçalho), `shell/mare/shell.qml`.

## Ligações

Afeta: [[Polkit]], [[Autostart]], [[Impressão]], [[Tela de bloqueio]]. Índice: [[Registro de decisões]].
