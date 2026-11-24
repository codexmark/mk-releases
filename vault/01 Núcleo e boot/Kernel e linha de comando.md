---
camada: "01 Núcleo e boot"
tags:
  - mk
  - boot
  - kernel
fontes:
  - "system/packages/base.packages"
  - "installer/mk-install"
  - "installer/README.md"
---

# Kernel e linha de comando

O Mk usa o kernel `linux` do Arch, com os cabeçalhos e o DKMS instalados, e uma linha de comando curta que aponta para o subvolume `@` do btrfs.

## Para quê

O kernel é o do Arch, sem modificações; o que o Mk decide é **como ele é iniciado** (a linha de comando) e o que vem junto para módulos de fora da árvore.

## Como funciona

- Pacotes em `base.packages`: `linux`, `linux-firmware`, `linux-headers`, `dkms`.
- A linha de comando é montada em `configure_boot` do instalador:
  - sem criptografia: `root=UUID=<uuid do btrfs> rootflags=subvol=@ rw quiet splash`
  - com [[LUKS]]: `rd.luks.name=<uuid da partição>=mkroot root=/dev/mapper/mkroot rootflags=subvol=@ rw quiet splash`
- Ela é gravada em `/etc/default/limine`, na chave `KERNEL_CMDLINE[default]`, que os [[Ganchos do limine]] leem a cada kernel novo. Sem os ganchos, vai direto nas entradas fixas do `limine.conf` ([[Limine]]).
- `quiet splash` é o que deixa o [[Plymouth]] aparecer no lugar das mensagens do kernel.
- O kernel e o initramfs ficam em `/boot`, que é a partição EFI ([[Particionamento]]), **fora** dos snapshots.

## Por quê assim

- `linux-headers` e `dkms`: o comentário da lista diz "módulos DKMS (drivers fora da árvore do kernel)". Na prática servem ao perfil `nvidia` (`nvidia-open-dkms`) e ao driver opcional `xone` da página Controle ([[Páginas dos Ajustes]]).
- `rootflags=subvol=@` em vez de `subvolid`: um rollback de snapshot troca o subvolume `@` e o id muda; o nome continua valendo. A mesma razão tira o `subvolid` do `fstab` ([[Btrfs e subvolumes]]).
- Um kernel só (`linux`), sem `linux-lts` de reserva: **motivo não registrado**.

> [!warning] Kernel fora do snapshot
> Como o kernel mora na ESP, voltar a um snapshot antigo não volta o kernel. O comportamento disso não foi verificado no projeto (veja [[Voltar atrás pelo menu de boot]]).

## Onde está no código

- `system/packages/base.packages` (seção "Núcleo")
- `installer/mk-install`: `configure_boot` (variável `cmdline`, arquivo `/etc/default/limine`)

## Ligações

Depende de: [[Particionamento]], [[Btrfs e subvolumes]]. Usado por: [[Initramfs]], [[Limine]], [[Ganchos do limine]].
