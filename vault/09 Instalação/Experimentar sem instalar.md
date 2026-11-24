---
camada: "09 Instalação"
tags:
  - mk
  - instalação
  - desenvolvimento
  - sessão
fontes:
  - "bin/try-session"
  - "bin/new-user"
  - "bin/diagnose-user"
  - "bin/install-packages"
  - "README.md"
  - "docs/desenvolvimento.md"
---

# Experimentar sem instalar

Num Arch que já tem Hyprland, o Mk pode rodar como uma **sessão separada**, escolhida na tela de login, sem tocar na config do desktop atual. Há dois caminhos: no seu usuário (`try-session`) ou num usuário novo (`new-user`).

## Para quê

Experimentar o Mk, ou desenvolver nele, sem formatar nada.

## Como funciona

**`bin/try-session install | uninstall | status`** (no seu usuário):

- copia `config/hypr`, `config/ghostty`, `config/fastfetch`, `config/bash` e `shell/mare` para `~/.config/mk`, com a marca;
- põe o tema GTK e gera os ícones em `~/.local/share`;
- grava `~/.config/mk/VERSION` e roda `Hyprland --verify-config`;
- na primeira vez, marca as migrações de usuário como feitas;
- **imprime** (não roda) os comandos com `sudo`: pacotes que faltam, `/usr/local/bin/mk-session`, a entrada "Mk" do login.

**`sudo bin/new-user <nome> [--admin] [--wallpapers]`** (usuário de teste):

- instala os pacotes que faltam, o lançador e a entrada do login;
- cria o usuário, copia a config para a home dele e **liga o `mk.bash` no `~/.bashrc`**;
- marca as migrações de usuário (só quando ele mesmo criou o usuário) e pede a senha;
- com um usuário que já existe, só atualiza a config dele.

**`sudo bin/diagnose-user <nome>`**: junta em `/tmp/mk-diagnostico` o que explica uma sessão que não abriu (listagens, o fim do `mare.log`, relatórios de crash do Quickshell, processos), sem alterar nada na home do usuário.

**`bin/install-packages <perfil…> [--no-apps]`**: instala as listas num Arch que já existe.

## Por quê assim

- **O `try-session` não roda sudo**: mostra os comandos e a pessoa decide. É a regra "root só onde precisa" ([[Convenções]]).
- **Uma pasta sem a marca não é sobrescrita nem apagada**, nem no `install` nem no `uninstall` ([[Marca instalado-pelo-mk]]).
- **O que o `try-session` não faz**, e por isso o modo é limitado:
  - não instala o Hyprland nem o `uwsm`;
  - não instala o `mk-update` nem `/usr/share/mk` (para atualizar: `git pull` e `install` de novo);
  - não mexe no `~/.bashrc`, então o [[Comando mk]] não existe;
  - não mexe nos apps padrão, no boot, na tela de login nem nas fontes do sistema.
- **`new-user` nasce "como numa instalação nova"**, com as boas-vindas no primeiro login: é o jeito de testar o primeiro uso.
- **`diagnose-user` existe por um acidente**: no primeiro login de um usuário novo, outro desktop recriava `~/.config/quickshell` e a sessão ficava sem shell ([[D01 Maré fora da pasta do Quickshell]]).
- O `try-session` ainda limpa restos de quando o projeto se chamava `meu-os`.

> [!note] Listas `needed` à mão
> Os dois scripts têm cada um a sua lista de pacotes sem os quais a sessão não abre, fora de `system/packages/`. Não são iguais (a do `try-session` não tem o `uwsm`).

## Onde está no código

`bin/try-session`, `bin/new-user`, `bin/diagnose-user`, `bin/install-packages`.

## Ligações

Depende de: [[mk-session e uwsm]], [[Pasta do usuário]], [[Listas de pacotes]]. Usado por: [[Trabalhar no repositório]]. Alternativa a: [[Instalador passo a passo]].
