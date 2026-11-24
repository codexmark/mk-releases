---
camada: "03 Base do sistema"
tags:
  - mk
  - marca
  - os-release
  - pacman
fontes:
  - "system/mk/os-release"
  - "system/mk/issue"
  - "system/mk/mk-os-release.hook"
  - "installer/mk-install"
  - "iso/build"
---

# Nome do sistema (os-release)

O sistema se apresenta como "Mk", não como "Arch Linux": no boot, no login do console, no `hostnamectl` e na ficha do terminal. Um gancho do pacman mantém isso depois das atualizações.

## Para quê

Coerência: quem instalou o Mk vê o nome Mk em todo lugar. Para ferramentas, `ID_LIKE=arch` continua dizendo de onde ele vem.

## Como funciona

- `system/mk/os-release`: `NAME="Mk"`, `PRETTY_NAME="Mk"`, `ID=mk`, `ID_LIKE=arch`, `BUILD_ID=rolling`, a cor ANSI do cobre e `LOGO=archlinux-logo`.
- `system/mk/issue`: `Mk \r (\l)`.
- O instalador copia os dois para `/etc/` e instala `/etc/pacman.d/hooks/mk-os-release.hook`.
- **O gancho**: dispara em `Install` e `Upgrade` do pacote `filesystem`, depois da transação, e copia de novo os arquivos de `/usr/share/mk/system/mk/`.
- No live da ISO o mesmo gancho já está instalado: quando o `pacstrap` instala o `filesystem` no disco novo, o os-release do Mk entra no lugar.
- O `mk-update` reinstala o gancho a cada troca do Mk.

## Por quê assim

O pacote `filesystem` é dono do `/etc/os-release` e o reescreve a cada atualização. Sem o gancho, o nome voltaria a "Arch Linux" na primeira atualização do `filesystem`. O código não discute alternativas; o Mk não tem pacotes próprios nem repositório de pacotes ([[Pacman e AUR]]), então um gancho é o que cabe.

Deduzido do código, não ensaiado: o gancho lê de `/usr/share/mk` e o `Exec` faz `rm -f` antes de copiar, então, se essa pasta sumir, o `/etc/os-release` fica ausente até ela voltar. Veja [[Quando a atualização para]].

## Onde está no código

- `system/mk/os-release`, `system/mk/issue`, `system/mk/mk-os-release.hook`
- `installer/mk-install`: `install_desktop`
- `iso/build` (instala o gancho no live), `bin/mk-update`

## Ligações

Depende de: [[Pacman e AUR]]. Usado por: [[Fastfetch]], [[ISO]], [[mk-update]].
