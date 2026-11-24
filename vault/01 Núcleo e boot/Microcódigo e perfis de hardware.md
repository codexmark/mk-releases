---
camada: "01 Núcleo e boot"
tags:
  - mk
  - boot
  - hardware
  - drivers
fontes:
  - "system/packages/hardware.packages"
  - "installer/mk-install"
  - "bin/lib-packages.sh"
  - "iso/README.md"
---

# Microcódigo e perfis de hardware

O microcódigo da CPU e os drivers de vídeo não estão na lista base: ficam em `hardware.packages`, em seções por perfil (`amd`, `intel`, `nvidia`, `vm`), e o instalador instala só as que a máquina pede.

## Para quê

Para a mesma ISO servir a máquinas diferentes sem instalar drivers de todo fabricante em todas.

## Como funciona

| Perfil | Pacotes |
|---|---|
| `amd` | `amd-ucode`, `vulkan-radeon` |
| `intel` | `intel-ucode`, `vulkan-intel`, `intel-media-driver` |
| `nvidia` | `nvidia-open-dkms`, `nvidia-utils`, `libva-nvidia-driver`, `egl-wayland` |
| `vm` | `vulkan-swrast` |

- `detect_hardware` marca o perfil pela CPU (`vendor_id` em `/proc/cpuinfo`), pela placa de vídeo (`lspci`, por isso `pciutils` entra na ISO) e por `systemd-detect-virt`. Mais de um perfil pode valer (por exemplo `amd` + `nvidia`).
- A tela "Drivers e apps" já vem com o detectado marcado; o usuário pode mudar.
- `read_hardware` (em `bin/lib-packages.sh`) lê só as seções pedidas.
- O microcódigo entra no boot pelo hook `microcode` do [[Initramfs]].
- Com `nvidia`, o hook `kms` sai do initramfs.

## Por quê assim

- Seções num arquivo só: "formato igual ao do Ryoku" (comentário da lista).
- `nvidia-open-dkms`: o comentário diz que serve para placas Turing (GTX 16xx / RTX 20xx) em diante. Placas mais antigas não têm perfil.
- Na VM o instalador marca `vm` e também `amd` ou `intel` (a VM roda com `-cpu host`), mas o vídeo é o virtual: os drivers de vídeo desses perfis são instalados e não usados.

> [!warning] Não testado
> O perfil `nvidia` nunca foi exercitado. O `installer/README.md` também lista como não verificados os drivers de vídeo de `amd` e `intel` "numa placa de verdade"; o `README.md` da raiz registra uma instalação em máquina real, sem dizer com que perfil ([[O que foi testado]]).

## Onde está no código

- `system/packages/hardware.packages`
- `installer/mk-install`: `detect_hardware`, `install_packages`, `configure_boot`
- `bin/lib-packages.sh`: `read_hardware`

## Ligações

Depende de: [[Listas de pacotes]]. Usado por: [[Initramfs]], [[Instalador passo a passo]]. Relacionado: [[Energia e suspensão]] (o aviso da AMD Navi 10).
