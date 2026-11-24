---
camada: "12 Decisões e limites"
tags:
  - mk
  - decisões
  - boot
  - limine
fontes:
  - "installer/mk-install"
  - "migrations/system/20261126-boot-gerenciado.sh"
  - "system/mk/boot/limine-default"
  - "installer/README.md"
---

# D09 Menu de boot dos ganchos

O menu de boot é escrito pelos ganchos do limine (`limine-update`); as entradas fixas do instalador são só a rede de segurança.

## Contexto

O instalador sempre teve uma conferência: depois de gerar o menu, verificar se cada `path` do `limine.conf` aponta para um arquivo que existe em `/boot`; se não, gravar entradas fixas (`/Mk` → `vmlinuz-linux`). A conferência tinha um defeito: com a verificação ligada, o `limine-entry-tool` põe `#<resumo>` no fim de cada caminho, e a conferência tratava isso como parte do nome do arquivo. Concluía que o menu dos ganchos estava quebrado e o **descartava sempre**.

Na primeira instalação em máquina real isso apareceu como: sem submenu de snapshots; num kernel novo, o menu podia continuar apontando para o antigo; três entradas na NVRAM; um aviso a cada sincronização de snapshots.

## Escolha

- A conferência corta o `#…` antes de testar; o menu dos ganchos passa a ser mantido.
- Uma entrada só na NVRAM, "Mk", apontando para o binário que os ganchos atualizam ([[Entrada UEFI única]]).
- `MAX_SNAPSHOT_ENTRIES=auto`; sem a entrada extra "EFI fallback" no menu.
- O [[limine-default]] resolve o efeito colateral: com os ganchos, a entrada 1 é um grupo e o menu não iniciava sozinho.
- Para quem já instalou: a migração `20261126-boot-gerenciado.sh` tira só as linhas da entrada fixa, deixa o `limine-update` escrever, e volta ao arquivo anterior se algo não fechar.

## Alternativas

- **Ficar só com as entradas fixas**: era o que acontecia de fato, por causa do defeito; sem submenu de snapshots e sem acompanhar kernels novos.
- **Depender só dos ganchos, sem rede de segurança** *(não discutida no repositório)*: os ganchos são do AUR e podem não ter sido compilados na ISO; o instalador mantém as entradas fixas para esse caso.

## Consequência

- O boot depende de dois pacotes do AUR para a parte boa (snapshots, kernels novos) e funciona sem eles para o essencial.
- O menu tem três formatos possíveis (fixo, dos ganchos, misto), e a migração e o `limine-default` precisam entender os três.
- A instalação para se, depois de tudo, o `limine.conf` não aponta para um kernel que existe: é a única verificação de boot que é fatal.

## Onde está no código

`installer/mk-install` (`limine_conf_ok`, `finish_boot`); `migrations/system/20261126-boot-gerenciado.sh`; `system/mk/boot/limine-default`.

## Ligações

Afeta: [[Limine]], [[Ganchos do limine]], [[limine-default]], [[Entrada UEFI única]], [[Voltar atrás pelo menu de boot]]. Índice: [[Registro de decisões]].
