---
camada: "11 Desenvolvimento e qualidade"
tags:
  - mk
  - desenvolvimento
  - testes
  - vm
  - qemu
fontes:
  - "iso/vm"
  - "iso/README.md"
  - "docs/desenvolvimento.md"
---

# VM de teste

`iso/vm` sobe uma VM no QEMU para testar a ISO e o Mk instalado, sem tocar em nenhum disco de verdade.

## Para quê

Tudo o que não é a sessão: ISO, instalador, boot, snapshots, tela de login, migrações, atualização.

## Como funciona

| Comando | O que faz |
|---|---|
| `iso/vm iso` | inicia pelo ISO mais novo de `iso/out` (para rodar `mk-install`) |
| `iso/vm`, `iso/vm disk` | inicia pelo disco da VM (o Mk já instalado) |
| `iso/vm reset` | apaga o disco e as variáveis do UEFI da VM |

- **Onde fica**: `~/.cache/mk-vm`, com `disk.qcow2` (40 GB, cresce conforme usa) e `vars.fd` (as variáveis do UEFI, onde fica a entrada "Mk").
- **Máquina**: KVM, `-cpu host`, 4 processadores, 6 GB, `q35`, firmware OVMF, disco virtio (`/dev/vda`), rede por cabo virtual, som pelo PipeWire.
- **Vídeo**: `virtio-vga-gl` com janela SDL e OpenGL; `MK_VM_GL=0` volta ao vídeo sem aceleração.
- **De fora**: a porta 2222 da máquina vai para a 22 da VM; `monitor.sock` é o monitor do QEMU (tira capturas com `screendump`).

O teste completo: `iso/build`, `iso/vm iso`, `mk-install` em `/dev/vda` (ou `--answers`), fechar, `iso/vm`.

## Por quê assim

- **Aceleração gráfica**: sem ela "tudo é desenhado no processador e as animações engasgam". Muito do ajuste fino do painel líquido veio de observar a VM lenta.
- **Janela SDL, não GTK**: a GTK do QEMU não consegue OpenGL nesse Wayland e o QEMU cai.
- **`vars.fd` próprio**: a entrada "Mk" da NVRAM precisa persistir entre os boots da VM; o `reset` apaga junto com o disco.
- **SSH exige dois passos dentro da VM**: ligar o `sshd` e abrir a porta no ufw, porque o Mk instalado vem com os dois fechados ([[Firewall]]). Esse caminho "nunca foi testado de ponta a ponta".
- **`iso/vm iso` não gera a ISO** nem avisa se ela é antiga: pega a mais nova de `iso/out`.

> [!warning] O que a VM não mostra
> Suspensão (a tela pode não voltar), Wi-Fi, os drivers de vídeo dos perfis `amd`/`intel`/`nvidia`, LUKS (dá para escolher, nunca foi feito), BIOS legado, mais de um monitor, impressoras, Bluetooth, bateria e brilho. Nada disso pode ser concluído a partir da VM ([[O que foi testado]]).

## Onde está no código

`iso/vm` (o cabeçalho é a documentação); `iso/README.md` (seção "Testar no QEMU").

## Ligações

Depende de: [[ISO]], [[Modo desassistido]]. Usado por: [[Trabalhar no repositório]], [[Testes e TDD]]. Relacionado: [[Limitações conhecidas]].
