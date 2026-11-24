---
camada: "01 Núcleo e boot"
tags:
  - mk
  - boot
  - criptografia
  - disco
fontes:
  - "installer/mk-install"
  - "installer/README.md"
  - "system/mk/plymouth/mk/mk.script"
  - "docs/limites.md"
---

# LUKS

A criptografia de disco é opcional: se o usuário pedir, a partição do sistema vira um LUKS2 aberto como `/dev/mapper/mkroot`, com o btrfs dentro.

## Para quê

Proteger os dados de uma máquina que pode ser perdida ou roubada. A ESP (`/boot`) fica fora da criptografia.

## Como funciona

1. A pergunta "Criptografar o disco (LUKS)?" tem padrão **não**. A senha do disco é a mesma do usuário.
2. `partition`: `cryptsetup luksFormat --type luks2` na partição 2 e `cryptsetup open` como `mkroot`, com a senha vinda por `--key-file -`. O btrfs é criado em `/dev/mapper/mkroot`.
3. `configure_boot`: o hook `sd-encrypt` entra no [[Initramfs]] e a linha do kernel passa a usar `rd.luks.name=<UUID>=mkroot root=/dev/mapper/mkroot` ([[Kernel e linha de comando]]).
4. No boot, o [[Plymouth]] pede a "senha do disco".
5. O [[SDDM]] entra direto (autologin em `/etc/sddm.conf.d/10-mk.conf`).
6. No fim da instalação, `cryptsetup close mkroot`. A contagem de etapas passa de 15 para 16.

## Por quê assim

- **Autologin com LUKS**: "a senha já foi digitada no boot, então entra direto (como no Omarchy)". Pedir a mesma senha duas vezes seguidas não acrescenta proteção.
- **Senha do disco = senha do usuário**: simplifica a entrevista. Alternativa (senha separada) não é oferecida; motivo não registrado além da simplicidade da tela.
- A senha não vai para o log nem para a linha de comando: passa por *stdin*.

> [!warning] Nunca testado
> O código existe, mas nenhuma instalação com `ENCRYPT=sim` foi feita, nem na VM. Isso inclui o pedido de senha no plymouth e o autologin. Veja [[Limitações conhecidas]].

## Onde está no código

`installer/mk-install`: `interview` (a pergunta), `partition`, `configure_boot`, `configure_services` (o autologin), `main` (o `cryptsetup close`).

## Ligações

Depende de: [[Particionamento]]. Afeta: [[Initramfs]], [[Kernel e linha de comando]], [[Plymouth]], [[SDDM]].
