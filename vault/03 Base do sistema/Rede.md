---
camada: "03 Base do sistema"
tags:
  - mk
  - rede
  - networkmanager
  - vpn
fontes:
  - "system/packages/base.packages"
  - "installer/mk-install"
  - "shell/mare/scripts/mare-dns"
  - "shell/mare/scripts/mare-vpn"
  - "shell/mare/scripts/mare-wifi-qr"
  - "shell/mare/scripts/mare-weather"
---

# Rede

A rede do Mk instalado é do NetworkManager; o Maré conversa com ele por `nmcli` e por scripts próprios, sem sudo.

## Para quê

Wi-Fi, cabo e VPN com uma interface só (o painel de rede e a página VPN), mais a descoberta de nomes `.local`.

## Como funciona

- **Pacotes** (`base.packages`): `networkmanager`, `iwd`, `wireless-regdb`, `iw`, `networkmanager-vpn-plugin-openvpn`, `avahi`, `nss-mdns`, `openssh`, `ufw`.
- **Serviços**: `NetworkManager.service` e `avahi-daemon.service` ([[Serviços systemd]]).
- **mDNS**: o instalador acrescenta `mdns_minimal [NOTFOUND=return]` antes de `resolve` na linha `hosts:` do `nsswitch.conf`. Com o avahi, isso resolve `impressora.local`, outras máquinas da rede etc.
- **Hostname e `/etc/hosts`**: gravados na etapa "fstab, região, rede".
- **No Maré** ([[Painéis]], [[Scripts mare]]):
  - `mare-net`: interface, IP, gateway, DNS da conexão em uso;
  - `mare-dns`: troca o DNS **só da conexão atual** (`auto`, `cloudflare`, `google`) com `nmcli con mod` + `nmcli dev reapply`;
  - `mare-wifi`: redes guardadas (esquecer, reconectar sozinho);
  - `mare-wifi-qr`: QR code para o celular entrar no Wi-Fi;
  - `mare-vpn`: WireGuard e OpenVPN (listar, ligar, importar `.conf`/`.ovpn`).
- **No live da ISO** a rede é outra: o instalador usa `iwctl` (iwd) para escolher o Wi-Fi quando não há cabo.

## Por quê assim

- **DNS por conexão, sem root**: o cabeçalho do `mare-dns` contrasta com a alternativa do Omarchy, que grava uma config global com root. Aqui só a conexão atual muda e "auto" volta ao DNS do roteador.
- **WireGuard sem pacote extra**: "já vem no kernel e o NetworkManager fala com ele sozinho"; só o OpenVPN precisa do plugin, e a versão escolhida é a sem interface GTK.
- **Senhas de VPN nunca por linha de comando**: o Mk não tem um agente gráfico de segredos, então a VPN que pede usuário e senha conecta num terminal com `nmcli --ask`.
- **O QR do Wi-Fi contém a senha**: o PNG é gravado em `$XDG_RUNTIME_DIR` com `umask 077` e o painel o apaga ao fechar ([[Convenções]]).
- **`iwd` instalado junto com o NetworkManager**: o repositório não configura o iwd como backend do NetworkManager nem diz por que os dois estão na lista. **Motivo não registrado.**

> [!warning] Não testado em hardware
> A escolha de Wi-Fi do instalador, esquecer rede, o QR e a importação de VPN nunca foram exercitados numa placa de verdade: a VM só tem cabo ([[Limitações conhecidas]]).

## Onde está no código

- `installer/mk-install`: `configure_base`, `ensure_network`
- `shell/mare/NetworkPanel.qml`, `shell/mare/Vpn.qml`, `shell/mare/SettingsVpnPage.qml`
- `shell/mare/scripts/mare-net`, `mare-dns`, `mare-wifi`, `mare-wifi-qr`, `mare-vpn`

## Ligações

Depende de: [[Serviços systemd]], [[Listas de pacotes]]. Usado por: [[Painéis]], [[Páginas dos Ajustes]], [[Firewall]], [[Impressão]].
