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
  - "system/mk/mk-dns"
  - "shell/mare/Dns.js"
  - "shell/mare/scripts/mare-vpn"
  - "shell/mare/scripts/mare-wifi-qr"
  - "shell/mare/scripts/mare-weather"
---

# Rede

A rede do Mk instalado é do NetworkManager; o Maré conversa com ele por `nmcli` e por scripts próprios, sem sudo (a troca de DNS pede a senha pelo polkit).

## Para quê

Wi-Fi, cabo, VPN e DNS com uma interface só (o painel de rede e as páginas VPN e DNS), mais a descoberta de nomes `.local`.

## Como funciona

- **Pacotes** (`base.packages`): `networkmanager`, `iwd`, `wireless-regdb`, `iw`, `networkmanager-vpn-plugin-openvpn`, `avahi`, `nss-mdns`, `openssh`, `ufw`.
- **Serviços**: `NetworkManager.service` e `avahi-daemon.service` ([[Serviços systemd]]).
- **mDNS**: o instalador acrescenta `mdns_minimal [NOTFOUND=return]` antes de `resolve` na linha `hosts:` do `nsswitch.conf`. Com o avahi, isso resolve `impressora.local`, outras máquinas da rede etc.
- **Hostname e `/etc/hosts`**: gravados na etapa "fstab, região, rede".
- **No Maré** ([[Painéis]], [[Scripts mare]]):
  - `mare-net`: interface, IP, gateway, DNS da conexão em uso;
  - `mare-dns`: o DNS **de todas as redes de uma vez**, para a página DNS dos Ajustes: `estado`, `aplicar`, `limpar`, `testar`, `soltar`;
  - `mare-wifi`: redes guardadas (esquecer, reconectar sozinho);
  - `mare-wifi-qr`: QR code para o celular entrar no Wi-Fi;
  - `mare-vpn`: WireGuard e OpenVPN (listar, ligar, importar `.conf`/`.ovpn`).
- **No live da ISO** a rede é outra: o instalador usa `iwctl` (iwd) para escolher o Wi-Fi quando não há cabo.

## Por quê assim

- **DNS global, num arquivo só**: Ajustes › DNS grava `/etc/NetworkManager/conf.d/90-mk-dns.conf` com `[global-dns-domain-*]` e `servers=`, o DNS global do NetworkManager, e roda `nmcli general reload conf` e depois `nmcli general reload dns-rc` (o `conf` relê o arquivo, o `dns-rc` reescreve o `resolv.conf`; o nmcli recusa os dois numa chamada só). Vale para todas as conexões e não mexe em nenhuma; "da rede" apaga o arquivo, e de fábrica ele não existe. Antes o Mk trocava o DNS só da conexão em uso, sem root (`nmcli con mod`); ficou global porque a escolha é da máquina, não de cada Wi-Fi. A conexão que ficou com DNS fixo daquele tempo aparece na página, com "soltar".
- **Quem escreve em `/etc` é um ajudante do root**: `/usr/lib/mk/mk-dns` (fonte em `system/mk/mk-dns`), chamado pelo `pkexec` ([[Polkit]]). Não aceita texto livre: só endereços IP, conferidos de novo do lado do root, e só aquele arquivo. Fica em `/usr/lib/mk` porque a troca do `mk-update` deixa o `/usr/share/mk` no nome de quem atualizou; o instalador, o `mk-update` e a migração `20261204-dns-ajudante.sh` o põem lá.
- **Sem DNS cifrado**: DoT/DoH não entram, porque o NetworkManager sozinho não faz (precisaria do systemd-resolved). A página diz isso, e que uma VPN ligada pode impor o DNS dela ([[Limitações conhecidas]]).
- **Testar depois de trocar**: a troca termina resolvendo um nome (`getent hosts archlinux.org`, 5 s); se não resolver, a página oferece voltar para o da rede.
- **WireGuard sem pacote extra**: "já vem no kernel e o NetworkManager fala com ele sozinho"; só o OpenVPN precisa do plugin, e a versão escolhida é a sem interface GTK.
- **Senhas de VPN nunca por linha de comando**: o Mk não tem um agente gráfico de segredos, então a VPN que pede usuário e senha conecta num terminal com `nmcli --ask`.
- **O QR do Wi-Fi contém a senha**: o PNG é gravado em `$XDG_RUNTIME_DIR` com `umask 077` e o painel o apaga ao fechar ([[Convenções]]).
- **`iwd` instalado junto com o NetworkManager**: o repositório não configura o iwd como backend do NetworkManager nem diz por que os dois estão na lista. **Motivo não registrado.**

> [!warning] Não testado em hardware
> A escolha de Wi-Fi do instalador, esquecer rede, o QR e a importação de VPN nunca foram exercitados numa placa de verdade: a VM só tem cabo. A troca de DNS só rodou nos testes, com o `pkexec` e o `nmcli` falsos ([[Limitações conhecidas]]).

## Onde está no código

- `installer/mk-install`: `configure_base`, `ensure_network`
- `shell/mare/NetworkPanel.qml`, `shell/mare/Vpn.qml`, `shell/mare/SettingsVpnPage.qml`, `shell/mare/SettingsDnsPage.qml`, `shell/mare/Dns.js`
- `system/mk/mk-dns`, `migrations/system/20261204-dns-ajudante.sh`
- `shell/mare/scripts/mare-net`, `mare-dns`, `mare-wifi`, `mare-wifi-qr`, `mare-vpn`

## Ligações

Depende de: [[Serviços systemd]], [[Listas de pacotes]], [[Polkit]]. Usado por: [[Painéis]], [[Páginas dos Ajustes]], [[Firewall]], [[Impressão]].
