---
camada: "02 Disco e sistema de arquivos"
tags:
  - mk
  - disco
  - btrfs
fontes:
  - "installer/mk-install"
  - "installer/README.md"
  - "docs/atualizacoes.md"
---

# Btrfs e subvolumes

A raiz do Mk é um btrfs com quatro subvolumes (`@`, `@home`, `@log`, `@pkg`), montados com `noatime,compress=zstd`.

## Para quê

Separar o que deve voltar atrás num rollback (o sistema) do que não deve (os arquivos do usuário, os logs, o cache de pacotes).

## Como funciona

| Subvolume | Montagem | Entra no snapshot de `/`? |
|---|---|---|
| `@` | `/` | sim |
| `@home` | `/home` | não |
| `@log` | `/var/log` | não |
| `@pkg` | `/var/cache/pacman/pkg` | não |

- Criados com `btrfs subvolume create` no topo do volume; depois o topo é desmontado e cada um é montado com `subvol=`.
- O `fstab` é gerado por `genfstab -U` (por UUID) e em seguida o instalador **remove o `subvolid=`** de cada linha.
- A ESP fica fora: é FAT32 ([[Particionamento]]).
- O `fstrim.timer` é habilitado (TRIM semanal).

## Por quê assim

- **Sem `subvolid` no fstab**: "um rollback de snapshot troca o `@` e o id muda". Com o id fixo, a raiz deixaria de montar depois de uma volta atrás; pelo nome (`subvol=@`), continua montando. A mesma lógica vale para `rootflags=subvol=@` na linha do kernel.
- **O que o snapshot cobre**: `/` inteiro, e portanto `/usr/share/mk`, `/etc` e `/var/lib/mk`. **Não cobre** a home: `~/.config/mk` e `~/.local/state/mk` não voltam junto. É uma consequência prática importante para [[Voltar atrás pelo menu de boot]].
- **`@log` e `@pkg` separados**: o código não comenta o porquê (**motivo não registrado**). O efeito é que logs e o cache de pacotes ficam fora dos snapshots: não crescem neles e não são revertidos.
- **`compress=zstd` e `noatime`**: escolha fixa em `partition`; **motivo não registrado**.
- Só btrfs: os snapshots e a volta atrás dependem dele; não há opção de outro sistema de arquivos.

## Onde está no código

`installer/mk-install`: `partition` (subvolumes e montagem), `configure_base` (o `genfstab` e o `sed` que tira o `subvolid`).

## Ligações

Depende de: [[Particionamento]], [[LUKS]]. Usado por: [[Snapper e snap-pac]], [[Kernel e linha de comando]], [[Voltar atrás pelo menu de boot]].
