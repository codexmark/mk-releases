---
camada: "10 Atualização e ciclo de vida"
tags:
  - mk
  - atualização
  - migrações
  - catálogo
fontes:
  - "migrations/system/20261130-mkpacotes-mkutil.sh"
  - "migrations/system/20261129-mkpulso.sh"
  - "migrations/user/20261128-papel-de-fabrica.sh"
  - "migrations/system/20261128-entrada-marca.sh"
  - "migrations/system/20261120-boot-mk.sh"
  - "migrations/system/20261122-boot-entrada.sh"
  - "migrations/system/20261125-apps-oferecidos.sh"
  - "migrations/system/20261126-boot-gerenciado.sh"
  - "migrations/system/20261127-mktunes.sh"
  - "migrations/user/20261120-terminal.sh"
  - "migrations/user/20261122-papeis-de-parede.sh"
  - "migrations/user/20261123-bem-vindo.sh"
---

# Catálogo de migrações

Todas as migrações que existem no repositório, com o que cada uma faz. Servem também de exemplos de como escrever a próxima.

## Para quê

Ver a história do sistema pelo que mudou fora de `<raiz>`, e achar um modelo parecido com o que se precisa fazer.

## Como funciona

### De sistema (`migrations/system/`)

| Arquivo | O que faz | `# pacotes:` |
|---|---|---|
| `20261120-boot-mk.sh` | tema do plymouth e `mkinitcpio -P` | |
| `20261120-entrada-mk.sh` | tema do sddm | |
| `20261120-terminal-skel.sh` | `fastfetch` e `bash` no skel, e a linha do `.bashrc` | |
| `20261121-apps-folha.sh` | tema GTK, ícones, Space Grotesk, qt6ct no skel | `adw-gtk-theme papirus-icon-theme` |
| `20261121-bash-mk.sh` | `seu.bash` e `bash` no skel; `blesh` no `aur-pendentes.txt` | `7zip` |
| `20261121-logo-plano.sh` | logo novo no plymouth e no sddm, ficha no skel, `mkinitcpio -P` | |
| `20261122-boot-entrada.sh` | roda o `limine-default`; layout no tema do sddm; `blesh` → `blesh-git` | |
| `20261123-webp.sh` | só declara o pacote (papéis de parede em WebP) | `qt6-imageformats` |
| `20261124-ilha-som.sh` | só declara o pacote (a onda da ilha) | `cava` |
| `20261125-apps-oferecidos.sh` | cria `/var/lib/mk/apps-oferecidos` | |
| `20261125-apps-padrao-skel.sh` | `mimeapps.list` no skel | |
| `20261125-bem-vindo-skel.sh` | marca a migração das boas-vindas no skel | |
| `20261125-emoji-wtype.sh` | só avisa se o `wtype` não veio | `wtype` |
| `20261125-impressoras.sh` | `wheel` no `SystemGroup` do CUPS; `cups.socket` | `cups ipp-usb jq` |
| `20261125-vpn-openvpn.sh` | só avisa se o plugin não veio | `networkmanager-vpn-plugin-openvpn` |
| `20261126-boot-gerenciado.sh` | tira a entrada fixa do menu e deixa o `limine-update` escrever | |
| `20261126-chaveiro-ssh.sh` | `gcr-ssh-agent.socket` para todos | `gcr-4` |
| `20261127-editor-nuvem.sh` | `vercel` no `aur-pendentes.txt` | `code aws-cli-v2 wrangler` |
| `20261127-mktunes.sh` | os links do MkTunes | `jre-openjdk-headless mpv ffmpeg` |
| `20261128-entrada-marca.sh` | o tema novo do sddm (a marca construída), mantendo o `theme.conf.user` | |
| `20261128-shellcheck.sh` | só avisa se o `shellcheck` não veio | `shellcheck` |
| `20261129-mkpulso.sh` | os links do MkPulso (comando, atalho, ícone); avisa se o coletor não veio | |
| `20261130-mkpacotes-mkutil.sh` | os links do MkPacotes, do MkInstalar e do MkComprimir (`apps-links`) | |

### De usuário (`migrations/user/`)

| Arquivo | O que faz |
|---|---|
| `20261120-terminal.sh` | `fastfetch` e `bash` em `<raiz>`, e a linha no `~/.bashrc` |
| `20261121-apps-folha.sh` | `~/.config/qt6ct`, se ainda não existe |
| `20261121-bash-mk.sh` | cria `<raiz>/seu.bash`, se não existe |
| `20261121-ficha-compacta.sh` | troca a pasta `fastfetch` de quem ficou com a antiga |
| `20261122-papeis-de-parede.sh` | move `~/Pictures/Wallpapers` para a pasta de imagens do sistema |
| `20261123-bem-vindo.sh` | grava `<raiz>/.bem-vindo` para quem já usava o Mk |
| `20261125-apps-padrao.sh` | `~/.config/mimeapps.list`, se não existe |
| `20261128-papel-de-fabrica.sh` | grava `<raiz>/.papel-de-fabrica`: quem já usava o Mk não recebe o papel de parede de fábrica |

## Por quê assim

Três padrões se repetem, e vale conhecê-los:

- **"Só se ainda não existe"** (`[[ -e … ]] ||`): `qt6ct`, `seu.bash`, `mimeapps.list`. O que já é do usuário não é tocado; a migração só diz "mantido (já existe)".
- **"Só se é do Mk"**: as que trocam uma pasta de `<raiz>` conferem a [[Marca instalado-pelo-mk]] antes.
- **"Guarda o anterior e volta se não fechar"**: a `20261126-boot-gerenciado.sh` é o exemplo mais cuidadoso. Copia o `limine.conf` para `limine.conf.mk-antes`, tira com `awk` só as linhas da entrada fixa do instalador (as subentradas dos ganchos ficam), roda o `limine-update` e confere que o menu aponta para arquivos que existem **e** tem uma entrada de Linux. Se qualquer passo não fecha, devolve o arquivo anterior, avisa e sai com 0. Só age se a entrada fixa está lá.

Essa migração existe porque o instalador, até certa versão, **descartava** o menu que os ganchos geravam e gravava entradas fixas: o submenu de snapshots nunca entrava e, num kernel novo, o menu podia ficar no antigo ([[D09 Menu de boot dos ganchos]]). Como as outras de sistema, lê e escreve por baixo da raiz `$R` (`MK_ROOT`), o que permite testá-la fora de `/boot`; cada migração tem o seu teste em `tests/migrations/`.

> [!note] Bit de execução
> O `mk-update` chama `bash <arquivo>`; o bit de execução não importa (a `20261127-mktunes.sh` não o tem).

## Onde está no código

`migrations/system/` (23 arquivos) e `migrations/user/` (8 arquivos).

## Ligações

Depende de: [[Migrações]]. Toca em: [[Plymouth]], [[SDDM]], [[limine-default]], [[Ganchos do limine]], [[Impressão]], [[Chaveiro e agente SSH]], [[MkTunes]], [[Apps oferecidos]], [[Boas-vindas]], [[Papéis de parede]].
