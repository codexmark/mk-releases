---
camada: "04 Sessão"
tags:
  - mk
  - sessão
  - polkit
  - segurança
fontes:
  - "shell/mare/PolkitDialog.qml"
  - "shell/mare/shell.qml"
  - "system/packages/desktop.packages"
  - "shell/mare/scripts/mare-printers"
---

# Polkit

Quando um programa pede a senha de administrador pelo polkit (`pkexec`, o CUPS, um app gráfico), quem mostra o pedido é o próprio Maré: ele é o agente de polkit da sessão.

## Para quê

Autorizar ações de administrador numa janela, em português, com a mensagem inteira.

## Como funciona

- `shell.qml` instancia `PolkitDialog {}` uma vez para o shell.
- O pedido aparece como uma folha no meio da tela em foco, na camada *Overlay* (`mare-polkit`), com teclado exclusivo.
- Com a tela bloqueada, o pedido é **cancelado na hora**.
- O `hyprpolkitagent` continua instalado (`desktop.packages`), de reserva: `systemctl --user start hyprpolkitagent`. O autostart não o inicia.

## Por quê assim

- **Por que não o `hyprpolkitagent`**: ele "só tem uma linha para a mensagem e os rótulos em inglês". Foi uma das issues da primeira instalação em máquina real: a mensagem era cortada. O Maré quebra o texto em linhas e nunca corta.
- **Só um agente registra por sessão.** Com o do Maré no ar, o outro não pode subir; por isso ele fica instalado mas parado.
- **Cancelar com a tela bloqueada**: a folha ficaria escondida atrás do bloqueio e o pedido esperaria para sempre.
- **Relação com os Ajustes**: a folha dos Ajustes cobre a tela inteira, e a janela de senha ficaria atrás dela. As páginas Impressoras e DNS fecham a folha antes de pedir a senha ([[Impressão]], [[Rede]]).

Decisão registrada em [[D06 Maré como agente de polkit]].

## Onde está no código

`shell/mare/PolkitDialog.qml` (o cabeçalho explica), `shell/mare/shell.qml`.

## Ligações

Depende de: [[Maré]], [[Usuários e sudo]]. Usado por: [[Impressão]], [[Tela de bloqueio]].
