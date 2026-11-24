---
camada: "03 Base do sistema"
tags:
  - mk
  - impressão
  - cups
fontes:
  - "installer/mk-install"
  - "system/packages/base.packages"
  - "shell/mare/scripts/mare-printers"
  - "migrations/system/20261125-impressoras.sh"
  - "shell/mare/SettingsPrintersPage.qml"
---

# Impressão

A impressão é do CUPS, ligado sob demanda pelo `cups.socket`, com o `ipp-usb` para impressoras USB sem driver. A página Impressoras dos Ajustes fala com ele pelo script `mare-printers`.

## Para quê

Achar, adicionar e usar impressoras de rede e USB sem abrir a interface web do CUPS.

## Como funciona

- **Pacotes**: `cups`, `ipp-usb`, `jq` (os scripts montam JSON com ele); `avahi` e `nss-mdns` para a descoberta ([[Rede]]).
- **Serviço**: `cups.socket` habilitado pelo instalador.
- **Permissão**: o instalador garante que o grupo `wheel` está no `SystemGroup` de `/etc/cups/cups-files.conf` (acrescenta se faltar).
- **`mare-printers`**: `status`, `find` (procura cerca de 5 s por impressoras IPP Everywhere / AirPrint na rede e no USB), `add` (`lpadmin ... -m everywhere`), `default`, `remove`, `test`, `cancel`, `resume`, `enable`. Tudo responde em JSON.
- **Senha de administrador**: a ação é tentada direto; se o CUPS recusa, é refeita **uma vez** pelo `pkexec`, e quem pede a senha é o Maré ([[Polkit]]).

## Por quê assim

- **`wheel` no `SystemGroup`** é o padrão do Arch; o instalador "só garante". O cabeçalho do script admite que o padrão "não se confirmou em toda máquina", daí a segunda tentativa pelo `pkexec`.
- **A ação roda num processo à parte, em outra sessão.** Os Ajustes são uma folha por cima de tudo, e a janela de senha ficaria atrás dela. A página fecha a folha ao ler `{"elevating":true}`; a ação segue sozinha, grava o resultado em `$XDG_RUNTIME_DIR/mare-printers` e reabre a folha na página.
- **Só impressoras sem driver** entram pelo botão; as USB que pedem driver do fabricante aparecem numa lista "legacy", sem ação.
- A migração `20261125-impressoras.sh` leva o mesmo arranjo a quem instalou antes, com `# pacotes: cups ipp-usb jq`.

> [!warning] Não testado em hardware
> Nenhuma impressora de verdade foi usada com o `mare-printers`. O script tem um modo de fixtures (`MARE_PRINTERS_FIXTURES`) e um `selftest`, mas isso só confere a leitura das saídas.

## Onde está no código

- `installer/mk-install`: `configure_services` (bloco "Impressoras")
- `shell/mare/scripts/mare-printers`, `shell/mare/SettingsPrintersPage.qml`
- `migrations/system/20261125-impressoras.sh`

## Ligações

Depende de: [[Serviços systemd]], [[Rede]], [[Polkit]], [[Usuários e sudo]]. Usado por: [[Páginas dos Ajustes]].
