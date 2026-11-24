---
camada: "01 Núcleo e boot"
tags:
  - mk
  - boot
  - limine
fontes:
  - "system/mk/boot/limine-default"
  - "installer/mk-install"
  - "bin/mk-update"
  - "migrations/system/20261122-boot-entrada.sh"
  - "migrations/system/20261126-boot-gerenciado.sh"
---

# limine-default

`system/mk/boot/limine-default` é um script que faz o menu do limine iniciar sozinho depois da contagem, mesmo quando os ganchos reescrevem o menu em grupos.

## Para quê

Sem ele, num Mk com os ganchos do limine, o menu aparecia, a contagem de 3 segundos acabava e **nada iniciava**.

## Como funciona

O limine inicia a entrada 1 por padrão. Com os ganchos, a entrada 1 vira um grupo ("Mk", com "linux" e "Snapshots" dentro), e um grupo só abre e fecha. O script:

1. Lê o `limine.conf` (o de `/boot`, `/boot/EFI/limine` ou `/boot/limine`, ou o caminho dado) e anota, para cada entrada, a profundidade (quantas `/`), se já está aberta (`+`) e se tem `protocol: linux`.
2. Acha a **primeira entrada que inicia de verdade**: tem `protocol: linux` e não tem filhos.
3. Abre os grupos acima dela: `/Mk` vira `/+Mk`, que o limine mostra já aberto.
4. Calcula a posição dela **entre as entradas visíveis** (filhos de grupo fechado não contam) e grava em `default_entry`.
5. Se mudou algo e existe `limine-enroll-config`, roda-o (com a verificação ligada o limine confere o resumo do arquivo).

É idempotente: rodando de novo diz "já está certo".

## Por quê assim

- O limine "só conta o que está visível": com o grupo fechado, a entrada 2 seria a de baixo ("EFI fallback"), não o linux. Por isso abrir o grupo e contar só as visíveis.
- Só entradas de **Linux**: a "EFI fallback" apenas carrega o limine de novo. Esse filtro entrou depois que a VM mostrou um terceiro formato de menu.
- Os ganchos reescrevem o menu sempre que rodam, então o script é chamado em três momentos: pelo instalador, pela migração `20261122-boot-entrada.sh` (e pela `20261126-boot-gerenciado.sh`) e pelo `mk-update`, depois de instalar os pacotes do AUR que ficaram de fora da ISO.

```bash
sudo bash /usr/share/mk/system/mk/boot/limine-default
```

## Onde está no código

`system/mk/boot/limine-default` (o cabeçalho explica o problema inteiro).

## Ligações

Depende de: [[Limine]], [[Ganchos do limine]]. Chamado por: [[Instalador passo a passo]], [[mk-update]], [[Catálogo de migrações]].
