---
camada: "08 Aplicativos"
tags:
  - mk
  - aplicativos
  - desenvolvimento
  - mise
fontes:
  - "system/packages/dev.packages"
  - "scripts/user/mise-tools.sh"
  - "scripts/user/neovim.sh"
  - "shell/mare/scripts/mare-devtools"
  - "migrations/system/20261127-editor-nuvem.sh"
  - "README.md"
---

# Ferramentas de desenvolvimento

O kit de desenvolvimento é "o coração deste projeto" (comentário de `dev.packages`). Tem duas metades: o que vem como pacote do sistema e o que cada usuário instala para si pelo `mise`.

## Para quê

Sair da instalação com terminal, git, editor, linguagens e containers prontos, e ter um caminho curto para o resto.

## Como funciona

**Do sistema** (`dev.packages`, tudo dos repositórios oficiais):

| Grupo | Pacotes |
|---|---|
| terminal e shell | `ghostty`, `tmux`, `starship`, `zoxide`, `fzf`, `bash-completion`, `atuin` |
| CLI moderna | `ripgrep`, `fd`, `bat`, `eza`, `dua-cli`, `plocate`, `jq`, `yq`, `gum`, `btop`, `fastfetch`, `tldr`, `yazi`, `inotify-tools`, `socat`, `whois` |
| git | `git`, `github-cli`, `lazygit`, `git-delta`, `difftastic` |
| editor | `neovim`, `7zip`, `tree-sitter-cli` |
| linguagens | `mise`, `go`, `rust`, `nodejs`, `npm`, `pnpm`, `python`, `python-pip`, `python-pipx`, `uv`, `ruby`, `lua51`, `luarocks` |
| compilação nativa | `clang`, `llvm`, `cmake`, `postgresql-libs`, `mariadb-libs`, `libyaml` |
| containers | `docker`, `docker-compose`, `docker-buildx`, `lazydocker` |
| projeto | `just`, `direnv`, `httpie`, `pgcli` |
| editor gráfico e nuvem | `code`, `aws-cli-v2`, `wrangler` |

Mais, do AUR: `vercel`.

**Do usuário**:

- `scripts/user/neovim.sh`: clona o *starter* do LazyVim em `~/.config/nvim`, **só se a pasta não existir**. O instalador roda para o usuário criado.
- `scripts/user/mise-tools.sh`: `mise use -g` de node (LTS), python e três CLIs de IA (Claude Code, Codex, Gemini), e cria `~/Work`. Ninguém roda por você: abre-se pelo 4º passo das [[Boas-vindas]] ou com `mk ferramentas`, que chamam `mare-devtools` num terminal.

## Por quê assim

- **Sistema versus usuário**: "versões específicas de linguagem por projeto: use o mise". O que precisa de versão por projeto não vira pacote do sistema.
- **`mise-tools.sh` não roda sozinho**: é opcional, precisa de internet e o próprio script diz "deixe só os que você usa" sobre as CLIs de IA. Roda como o usuário, sem sudo, e a janela fica aberta até o Enter, "para o progresso e os erros ficarem à vista".
- **`code` é o build aberto do VS Code**: aparece como "Code - OSS" e usa o Open VSX no lugar da loja da Microsoft. O binário da Microsoft (`visual-studio-code-bin`, AUR) não vem "porque não pode ser redistribuído num ISO".
- **CLIs de nuvem dos repositórios oficiais**: atualizam com o sistema. A da Vercel só existe no AUR; se não foi compilada na ISO, o instalador avisa e o `mk-update` oferece.
- **LazyVim nunca sobrescreve** uma config existente.
- **Bibliotecas cliente de banco**: "para drivers nativos compilarem" (módulos C de node, python, ruby).
- A lista "mistura as linguagens do Ryoku com o kit de terminal do Omarchy".

> [!warning] Não testado em máquina
> A instalação de VS Code, AWS, Wrangler e Vercel numa máquina real consta como não conferida em `docs/limites.md`. A instalação pelo mise foi conferida na VM.

## Onde está no código

`system/packages/dev.packages`; `scripts/user/`; `shell/mare/scripts/mare-devtools`; `migrations/system/20261127-editor-nuvem.sh`.

## Ligações

Depende de: [[Listas de pacotes]], [[Pacman e AUR]]. Usado por: [[Terminal e bash do Mk]], [[Projetos e workspaces]], [[Boas-vindas]], [[Comando mk]].
