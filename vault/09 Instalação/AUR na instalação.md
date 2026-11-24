---
camada: "09 Instalação"
tags:
  - mk
  - instalação
  - aur
  - iso
fontes:
  - "iso/build"
  - "installer/mk-install"
  - "installer/README.md"
  - "iso/README.md"
  - "bin/mk-update"
  - "docs/desenvolvimento.md"
---

# AUR na instalação

O AUR **não é compilado durante a instalação**. O `iso/build` compila os pacotes de `aur.packages` de antemão num repositório pacman local, `mk-aur`, que vai dentro da ISO; o instalador instala dali.

## Para quê

Instalar pacotes do AUR (os ganchos do limine, o `yay`, a Fraunces, o ble.sh…) sem `makepkg` no live, sem depender do AUR estar no ar na hora e sem que a falha de um trave a instalação.

## Como funciona

**No build**:

- para cada pacote: clona o PKGBUILD, enfileira as dependências que só existem no AUR, e compila com `makepkg --clean --cleanbuild --force --noconfirm --nocheck`;
- **reaproveitamento**: se já existe um pacote compilado **na mesma versão** de uma rodada anterior, copia em vez de compilar (menos os `-git`, que são sempre recompilados);
- sem `--syncdeps`, um pacote ao qual falta dependência **de compilação** falha e vai para o relatório; dependências só de execução não impedem (compila com `--nodeps`; elas chegam no `pacstrap`);
- `repo-add` monta `mk-aur.db.tar.gz`; `packages.txt` lista o que entrou. Sem nenhum pacote, cria um repositório vazio "para o instalador não quebrar".

**Na instalação** (`install_aur`):

1. o `pacman.conf` temporário ganha `[mk-aur]` com `SigLevel = Optional TrustAll` e `Server = file://…`;
2. `pacstrap -G -M` com exatamente a lista de `packages.txt`; se a instalação conjunta falha, tenta um por um;
3. compara com `aur.packages` e grava o que faltou em `/usr/share/mk/aur-pendentes.txt`;
4. a instalação **não para** por causa disso.

**Depois**: o [[mk-update]] vê o `aur-pendentes.txt` e, havendo `yay`, pergunta se instala. Se o `yay` dá certo, apaga o arquivo; se falha, o arquivo fica para a próxima rodada. O arquivo é preservado na troca de `/usr/share/mk`.

## Por quê assim

- **Compilar na ISO, não no live nem no primeiro boot**: o cabeçalho de `aur.packages` diz que o `iso/build` os compila num repositório local que vai no ISO, "e a falha de um não trava nem o build nem a instalação". A compilação sai do caminho do usuário. Decisão registrada em [[D10 AUR compilado na ISO]].
- **Uma falha não para o build nem a instalação**: o pacote fica de fora e vai para o relatório. O sistema inicia sem os ganchos do limine, só perde o submenu de snapshots.
- **Reaproveitar o já compilado**: sem isso, cada rodada pedia de novo as dependências de compilação (o `gradle`) e, sem `--syncdeps`, a ISO saía sem os ganchos do boot.
- **`aur-pendentes.txt` é também o mecanismo para levar um pacote novo do AUR a quem já instalou**: a linha `# pacotes:` das migrações só aceita nomes dos repositórios oficiais, então uma migração de sistema acrescenta o nome ao arquivo (como fazem as do `blesh-git` e do `vercel`).
- **`-G -M` no segundo `pacstrap`**: não copiar de novo o chaveiro e a lista de espelhos do live.

> [!warning] Se a ISO veio sem nenhum pacote do AUR
> O instalador só avisa e **não** grava o `aur-pendentes.txt`; a lista de referência passa a ser `/usr/share/mk/system/packages/aur.packages`. E como o `yay-bin` também é do AUR, sem ele a oferta do `mk-update` não aparece: instala-se o `yay` à mão.

## Onde está no código

- `iso/build` (bloco "1. AUR")
- `installer/mk-install`: `make_pacman_conf`, `install_aur`
- `bin/mk-update` (bloco "AUR que ficou de fora da instalação")

## Ligações

Depende de: [[ISO]], [[Pacman e AUR]], [[Listas de pacotes]]. Usado por: [[Ganchos do limine]], [[mk-update]], [[Migrações]].
