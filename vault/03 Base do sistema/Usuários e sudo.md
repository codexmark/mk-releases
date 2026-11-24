---
camada: "03 Base do sistema"
tags:
  - mk
  - usuários
  - sudo
  - skel
fontes:
  - "installer/mk-install"
  - "installer/README.md"
  - "bin/new-user"
---

# Usuários e sudo

O instalador cria um usuário com `bash`, no grupo `wheel`, com sudo por senha; o root fica bloqueado. A home nasce de `/etc/skel`, que o instalador enche com as configs do Mk.

## Para quê

Uma conta só, com poderes de administrador por senha, e a garantia de que **qualquer** usuário criado depois nasce com o Mk.

## Como funciona

- `useradd -m -s /bin/bash -G wheel[,video,audio,input,docker]` (os grupos extras só se existirem).
- A senha é a digitada na entrevista (`chpasswd`); `passwd -l root` bloqueia o root.
- `/etc/sudoers.d/10-mk-wheel`: `%wheel ALL=(ALL:ALL) ALL`, com permissão `440`.
- **O skel**: o instalador não escreve na home. Ele põe em `/etc/skel`:

| No skel | De onde vem |
|---|---|
| `.config/mk/{hypr,shell,ghostty,fastfetch,bash}` (com a [[Marca instalado-pelo-mk]]) | `config/hypr`, `shell/mare`, `config/ghostty`, `config/fastfetch`, `config/bash` |
| `.config/mk/seu.bash` | `config/bash/seu.bash.exemplo` |
| `.config/mk/ajustes.json` (só com placa de risco) | semeado com `"power.sleep": "nada"`, que hoje é também o valor de fábrica ([[Energia e suspensão]]) |
| `.config/mimeapps.list` | `config/mimeapps.list` |
| `.config/qt6ct` | `system/mk/look/qt6ct` |
| `.bashrc` | ganha a linha que carrega o `mk.bash` |
| `.local/state/mk/migrations/` | as migrações de usuário, marcadas como feitas |
| `Work/` | pasta vazia |

- Depois do `useradd`, o instalador **confere** que `hypr`, `ghostty` e `shell` chegaram na home com a marca; se não, para.

## Por quê assim

- **Via skel, não direto na home**: "todo usuário criado depois nasce igual". Consequência: o `/etc/skel` de uma máquina já instalada só muda por migração de sistema ([[Migrações]]).
- **Root bloqueado, sudo com senha**: escolha do instalador; o motivo não está registrado além do comentário "a senha também vale para o sudo".
- **`docker` no grupo**: para usar o Docker sem sudo; o grupo só existe se o pacote foi instalado.
- `bin/new-user` cria um usuário de teste com o mesmo desenho, mas escreve direto na home dele e usa só `video`, `audio`, `input` (e `wheel` com `--admin`).

## Onde está no código

- `installer/mk-install`: `install_desktop` (o skel), `create_user`
- `bin/new-user`

## Ligações

Depende de: [[Listas de pacotes]]. Usado por: [[Pasta do usuário]], [[Impressão]], [[Polkit]], [[Migrações]].
