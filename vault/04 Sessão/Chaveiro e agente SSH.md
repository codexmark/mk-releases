---
camada: "04 Sessão"
tags:
  - mk
  - sessão
  - chaveiro
  - ssh
fontes:
  - "config/hypr/mare/env.lua"
  - "installer/mk-install"
  - "migrations/system/20261126-chaveiro-ssh.sh"
  - "system/packages/desktop.packages"
  - "system/packages/apps.packages"
---

# Chaveiro e agente SSH

As senhas dos apps ficam no `gnome-keyring`, e as chaves SSH com senha são lembradas pelo `gcr-ssh-agent`, ligado para todos os usuários.

## Para quê

Não redigitar a senha de uma chave SSH a cada `git push`, e dar aos apps um lugar padrão para guardar segredos.

## Como funciona

- **Pacotes** (`desktop.packages`): `gnome-keyring`, `gcr-4`, `polkit`. O `seahorse` (ver e apagar o que está guardado) está em `apps.packages`.
- **Agente SSH**: o instalador roda `systemctl --global enable gcr-ssh-agent.socket` se a unidade existe.
- **`SSH_AUTH_SOCK`**: o `env.lua` aponta para `$XDG_RUNTIME_DIR/gcr/ssh` quando o soquete está habilitado globalmente ([[Variáveis de ambiente]]).
- **Mk já instalado**: a migração `20261126-chaveiro-ssh.sh` (`# pacotes: gcr-4`) faz o mesmo `enable --global`; vale no login seguinte.

## Por quê assim

- **`gcr-ssh-agent` e não o agente do gnome-keyring**: "o gnome-keyring não traz mais o seu" (comentário do `env.lua`).
- **Habilitado para todos, e a variável condicional**: o `env.lua` só define `SSH_AUTH_SOCK` se o soquete está ligado, para não apontar para um agente que não existe.
- **O daemon do chaveiro não é iniciado pelo Mk.** O commit que corrigiu a issue diz "o daemon não é mais iniciado duas vezes": a linha `gnome-keyring-daemon --start --components=secrets,pkcs11` saiu do `autostart.lua`. Quem o inicia hoje (a outra das "duas vezes") não está no repositório: não há config de PAM nem de autostart para ele. **Não confirmado no código.**

> [!note] Conferido só na VM
> `docs/limites.md` lista o chaveiro e o agente SSH entre o que foi conferido na VM.

## Onde está no código

- `config/hypr/mare/env.lua`
- `installer/mk-install`: `configure_services`
- `migrations/system/20261126-chaveiro-ssh.sh`

## Ligações

Depende de: [[Serviços systemd]], [[Variáveis de ambiente]]. Usado por: [[Terminal e bash do Mk]], [[Ferramentas de desenvolvimento]].
