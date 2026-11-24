---
camada: "03 Base do sistema"
tags:
  - mk
  - bluetooth
fontes:
  - "system/packages/base.packages"
  - "installer/mk-install"
  - "shell/mare/BluetoothPanel.qml"
  - "shell/mare/BluetoothPair.qml"
---

# Bluetooth

O Bluetooth é do BlueZ (`bluez`, `bluez-utils`), com o `bluetooth.service` habilitado. O painel do Maré liga o adaptador, lista, pareia e esquece aparelhos.

## Para quê

Fones, mouses, caixas de som e controles, sem abrir um aplicativo à parte.

## Como funciona

- `BluetoothPanel.qml` fala com o BlueZ pelo Quickshell: liga e desliga o adaptador, mostra os pareados (com a bateria quando o aparelho informa), procura aparelhos novos e esquece (com um segundo clique de confirmação).
- A lista de aparelhos novos só mostra os que dizem o nome.
- O pareamento em andamento mora no singleton `BluetoothPair`, não no painel.

## Por quê assim

- **`BluetoothPair` separado do painel**: o painel é destruído ao fechar, e parear leva de 5 a 20 segundos. Fechando no meio, ninguém confiaria no aparelho, conectaria nele ou avisaria da falha. Com o estado num singleton, o pareamento continua com o painel fechado.
- **Sem agente de pareamento**: o Quickshell fala com o BlueZ mas não registra um agente. Aparelhos que pareiam sozinhos funcionam; os que pedem código ou confirmação na tela **falham**, e para eles o painel deixa no rodapé o atalho do `bluetoothctl`.
- **Só aparelhos com nome**: sem isso a lista enche de endereços de sensores e etiquetas que ninguém quer parear.

> [!warning] Não testado em hardware
> Parear um aparelho de verdade nunca foi visto funcionando fora da VM, que não tem Bluetooth.

## Onde está no código

- `system/packages/base.packages` (seção "Bluetooth")
- `shell/mare/BluetoothPanel.qml`, `shell/mare/BluetoothPair.qml`

## Ligações

Depende de: [[Serviços systemd]]. Usado por: [[Painéis]], [[Singletons]].
