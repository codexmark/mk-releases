---
camada: "02 Disco e sistema de arquivos"
tags:
  - mk
  - disco
  - gpt
  - instalador
fontes:
  - "installer/mk-install"
  - "installer/README.md"
---

# Particionamento

O Mk usa o disco inteiro com uma tabela GPT de duas partições: uma ESP de 2 GiB montada em `/boot` e o resto para o btrfs.

## Para quê

Dar ao instalador um desenho de disco único e previsível, sobre o qual o boot, os snapshots e as migrações podem contar.

## Como funciona

Na etapa "formatando", depois de o usuário escrever `formatar`:

1. `swapoff -a`, desmonta o que estiver montado do disco, `wipefs -af` e `sgdisk --zap-all`.
2. Cria as partições com `sgdisk`:

| Partição | Tamanho | Tipo GPT | Rótulo | Sistema | Montagem |
|---|---|---|---|---|---|
| 1 | 2 GiB | `ef00` (ESP) | `MK_EFI` | FAT32 | `/boot` (com `umask=0077`) |
| 2 | o resto | `8304` (raiz Linux x86-64) | `mk` | btrfs, rótulo `mk` (ou LUKS2 com o btrfs dentro) | veja [[Btrfs e subvolumes]] |

3. `partprobe` e `udevadm settle`; se as duas partições não aparecem, para.
4. `part_name` cuida dos nomes: discos que terminam em número (`nvme0n1`) ganham `p1`, `p2`.

Validações antes de tocar no disco (`validate`): é um dispositivo de bloco do tipo `disk`, não é o pendrive do live e tem pelo menos 24 GiB. O pendrive e os `zram` nem aparecem na lista de escolha.

## Por quê assim

- **Disco inteiro, sem dual boot**: o instalador não redimensiona nem preserva partições. É uma limitação declarada, não um descuido ([[Limitações conhecidas]]).
- **ESP em `/boot`**: o kernel e o initramfs ficam na partição EFI, onde o limine os lê (`boot():/vmlinuz-linux`). Consequência: o kernel fica fora dos snapshots.
- **2 GiB de ESP**: o tamanho exato não tem motivo registrado. O que o código mostra é que a ESP também guarda entradas de snapshot, e `MAX_SNAPSHOT_ENTRIES=auto` "poda os mais antigos pelo espaço da ESP".
- **Mínimo de 24 GiB**: valor fixo em `validate`; **motivo não registrado**.
- **`umask=0077` na ESP**: só root lê a partição de boot.

> [!warning] Sem volta
> A confirmação pede a palavra `formatar` justamente porque tudo o que há no disco é apagado. Qualquer outra resposta cancela sem tocar em nada.

## Onde está no código

`installer/mk-install`: `list_disks`, `live_disk`, `validate`, `confirm_wipe`, `partition`, `part_name`.

## Ligações

Usado por: [[Btrfs e subvolumes]], [[LUKS]], [[Limine]], [[Entrada UEFI única]]. Parte do [[Instalador passo a passo]].
