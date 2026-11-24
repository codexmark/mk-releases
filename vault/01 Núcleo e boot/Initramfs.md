---
camada: "01 Núcleo e boot"
tags:
  - mk
  - boot
  - mkinitcpio
fontes:
  - "installer/mk-install"
  - "installer/README.md"
  - "migrations/system/20261120-boot-mk.sh"
  - "bin/mk-hibernate"
---

# Initramfs

O initramfs é gerado pelo `mkinitcpio` com uma lista de hooks que o instalador grava em `/etc/mkinitcpio.conf.d/mk.conf`, baseada no hook `systemd`.

## Para quê

É a imagem que o kernel carrega antes de montar a raiz: traz os módulos de disco e vídeo, mostra o [[Plymouth]] e, com disco criptografado, pede a senha.

## Como funciona

A lista de fábrica:

```
HOOKS=(base systemd plymouth autodetect microcode modconf kms keyboard sd-vconsole block filesystems fsck)
```

Variações que o instalador aplica:

- com [[LUKS]], entra `sd-encrypt` depois de `block`;
- com o perfil `nvidia`, sai o `kms`, "para o nouveau não carregar antes" (comentário do código).

O arquivo fica em `mkinitcpio.conf.d/`, não no `mkinitcpio.conf` principal: o do pacote continua intacto. O instalador gera as imagens com `/usr/bin/mkinitcpio -P`, pelo caminho inteiro.

O tema do plymouth vai **dentro** do initramfs. Por isso as migrações que trocam o tema ou o logo terminam com `mkinitcpio -P` ([[Catálogo de migrações]]).

## Por quê assim

- **Caminho inteiro do mkinitcpio.** O `limine-mkinitcpio-hook` instala um `/usr/local/bin/mkinitcpio` que fica na frente no `PATH` e pergunta "rodar o limine-mkinitcpio agora? [Y/n]". Na instalação sem perguntas isso travava a etapa à espera da resposta; o instalador passou a chamar `/usr/bin/mkinitcpio` direto e deixa o `limine-update` escrever o menu depois.
- **Hook `systemd` (e `sd-vconsole`, `sd-encrypt`) em vez de `udev`/`encrypt`.** O motivo da escolha não está registrado. O que o código mostra é a consequência: a linha de comando usa `rd.luks.name=` (a sintaxe do `sd-encrypt`), e o `bin/mk-hibernate` recusa mexer em máquinas cujo initramfs não usa o hook `systemd`.
- O initramfs de *fallback* só entra no menu se existir: "o preset padrão do Arch não gera mais o fallback".

## Onde está no código

- `installer/mk-install`: `configure_boot` (a lista `hooks`), `finish_boot` (o `mkinitcpio -P`)
- `migrations/system/20261120-boot-mk.sh`, `20261121-logo-plano.sh`

## Ligações

Depende de: [[Kernel e linha de comando]], [[Microcódigo e perfis de hardware]]. Usado por: [[Plymouth]], [[LUKS]], [[Ganchos do limine]].
