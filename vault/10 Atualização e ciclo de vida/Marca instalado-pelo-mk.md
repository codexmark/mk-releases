---
camada: "10 Atualização e ciclo de vida"
tags:
  - mk
  - atualização
  - convenção
  - segurança
fontes:
  - "bin/mk-update"
  - "installer/mk-install"
  - "bin/try-session"
  - "bin/new-user"
  - "migrations/user/20261120-terminal.sh"
  - "docs/desenvolvimento.md"
---

# Marca instalado-pelo-mk

`.instalado-pelo-mk` é um arquivo vazio que o Mk põe dentro de cada pasta que ele cria na home. É o que autoriza o Mk a trocar ou apagar aquela pasta mais tarde.

## Para quê

Responder com segurança à pergunta "esta pasta é minha ou do usuário?" antes de um `rm -rf`.

## Como funciona

- **Quem põe**: o instalador (no skel), o `mk-update`, o `try-session`, o `new-user` e as migrações que trocam pastas.
- **Onde**: nas cinco pastas de `<raiz>` (`hypr`, `shell`, `ghostty`, `fastfetch`, `bash`); o `try-session` marca também o tema GTK e os ícones em `~/.local/share`.
- **A regra**, igual em todos:
  - a pasta tem a marca, ou não existe → pode ser trocada;
  - a pasta existe **sem** a marca → fica como está. O `mk-update` diz "mantido (não é do Mk)"; o `try-session` e o `new-user` param com erro.
- O instalador **confere** a marca em `hypr`, `ghostty` e `shell` na home do usuário recém-criado; se falta, a instalação para.

## Por quê assim

- **"Só se apaga o que o Mk criou."** O Mk convive com outros desktops na mesma conta e troca pastas inteiras a cada atualização; sem uma prova de autoria, uma pasta `~/.config/mk/hypr` posta ali pelo usuário seria apagada.
- **Tirar a marca é uma escolha suportada**: quem quer manter um `user.lua` ou um `monitors.lua` dentro de `hypr/` pode apagar `hypr/.instalado-pelo-mk`. O `mk-update` passa a deixar a pasta como está, e a pessoa deixa de receber as mudanças do Hyprland do Mk ([[Pasta do usuário]]).
- O nome antigo, `.instalado-pelo-meu-os`, ainda é reconhecido pelo `try-session` para limpar instalações de quando o projeto tinha outro nome.

## Onde está no código

- `bin/mk-update` (o laço `for pair in hypr:config/hypr …`)
- `installer/mk-install` (`place`, e a conferência em `create_user`)
- `bin/try-session`, `bin/new-user` (`place`, `remove`)

## Ligações

Usado por: [[mk-update]], [[Pasta do usuário]], [[Experimentar sem instalar]], [[Catálogo de migrações]], [[Convenções]].
