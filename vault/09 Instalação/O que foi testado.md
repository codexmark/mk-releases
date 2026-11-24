---
camada: "09 Instalação"
tags:
  - mk
  - instalação
  - testes
  - limites
fontes:
  - "README.md"
  - "installer/README.md"
  - "iso/README.md"
  - "docs/limites.md"
  - "tests/README.md"
  - "tests/installer/carrega.sh"
---

# O que foi testado

O Mk é o desktop de uma pessoa, em uso e mudando rápido. Esta nota junta o que o repositório declara como testado e como não testado na instalação, e por qual dos três meios.

## Para quê

Decidir com os olhos abertos se dá para instalar numa máquina que importa.

## Como funciona

O Mk é testado de três jeitos (`docs/limites.md`): os testes automáticos, a sessão aninhada e uma VM.

**Pelos testes automáticos** (`tests/installer/`, `tests/iso/`, `tests/system/`, sem máquina nenhuma): a lógica do instalador, função por função, com o disco e os comandos trocados por dublês: a leitura e a validação das respostas, o particionamento, a configuração do sistema, o boot (inclusive `limine_conf_ok` e a entrada UEFI) e o modo desassistido; o `iso/build`, o `iso/vm` e o perfil da ISO; as listas de pacotes e o `limine-default`. O instalador **nunca roda inteiro** num teste.

**Na VM** (QEMU, UEFI/OVMF, sem LUKS, perfil `vm`, rede por cabo, um disco virtio, uma tela):

- a instalação do começo ao fim, com o `limine.conf` gerenciado pelos ganchos, uma entrada UEFI só ("Mk") e o submenu de snapshots;
- a atualização por `mk atualizar`, com migrações de sistema e de usuário, e o reinício depois da migração do boot;
- o pedido de senha de administrador, o chaveiro e o agente SSH;
- as boas-vindas e a instalação das ferramentas pelo mise.

**Em máquina de verdade**: uma instalação pelo ISO, que deu boot e está em uso. Os defeitos que ela mostrou viraram issues, e várias correções descritas nesta vault vêm delas (o `mk.bash` que derrubava o terminal, o menu de boot descartado, a entrada UEFI duplicada, o polkit em inglês).

**Nunca testado**:

| O quê | Onde está o código |
|---|---|
| disco criptografado ([[LUKS]]): uma instalação com `ENCRYPT=sim` e o boot dela | `partition`, `configure_boot` |
| perfil `nvidia` | `hardware.packages`, `configure_boot` |
| escolha de rede Wi-Fi no instalador | `ensure_network` |
| firmwares UEFI de placas variadas | `finish_boot` |

**Não existe**: instalação em BIOS legado, dual boot, discos com menos de 24 GiB, sistema de arquivos que não seja btrfs.

## Por quê assim

- **Três meios, três alcances**: os testes conferem a lógica e rodam a cada mudança; a VM confere o disco e o boot de verdade; nenhum dos dois alcança Wi-Fi, placas de vídeo reais, mais de um monitor, impressoras, Bluetooth, bateria e suspensão ([[VM de teste]]).
- **Um caminho coberto por teste não é um caminho visto funcionando**: o código do LUKS tem casos em `tests/installer/`, e a instalação criptografada continua na lista do nunca testado.
- **Por que declarar em vez de esconder**: o `README.md` abre com "Estado do projeto" e pede: "leia isto antes de instalar numa máquina que importa". O `installer/README.md` recomenda cópia dos arquivos e um pendrive de outro sistema à mão.
- **A regra de saída** (`docs/limites.md`): uma coisa sai da lista quando alguém a confere numa máquina de verdade e registra onde.

## Onde está no código

`README.md` (seção "Estado do projeto"), `docs/limites.md`, `installer/README.md` ("O que foi testado"), `iso/README.md` ("O que a VM não mostra"), `tests/installer/`.

## Ligações

Relacionado: [[Limitações conhecidas]], [[VM de teste]], [[Testes e TDD]], [[Instalador passo a passo]], [[Microcódigo e perfis de hardware]].
