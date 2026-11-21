# ISO do Mk

`iso/` gera um ISO live (archiso, baseado no perfil `releng`) que inicia num
console com o instalador `mk-install`. O que o instalador faz está em
[installer/README.md](../installer/README.md).

| Arquivo | Para quê |
| --- | --- |
| `iso/build` | compila o AUR, monta o perfil e gera o ISO |
| `iso/vm` | VM no QEMU para testar o ISO e o Mk instalado |
| `iso/profile/` | o que o Mk põe por cima do `releng` |
| `iso/work/`, `iso/out/` | saídas do build (fora do git) |

## Gerar o ISO

Precisa de um Arch com `archiso`, `git`, `jq`, `curl` e `base-devel`, e de
internet (AUR e espelhos do Arch). O `archiso` só serve para gerar o ISO, não
para usar o Mk:

```bash
sudo pacman -S --needed archiso git jq curl base-devel
```

Rode como o seu usuário, sem sudo. O script compila os pacotes do AUR com
`makepkg` (que não roda como root) e só no fim chama `sudo mkarchiso`, que pede
a sua senha:

```bash
iso/build
```

O ISO sai em `iso/out/mk-AAAA.MM.DD-x86_64.iso` (cerca de 1,4 GB), com o rótulo
de volume `MK`. O arquivo fica com dono root.

| Opção | O que faz |
| --- | --- |
| `--prepare` | só compila o AUR e monta o perfil (não usa sudo) e imprime o comando do `mkarchiso` |
| `--syncdeps` | deixa o `makepkg` instalar (com sudo) as dependências que faltam na sua máquina e remover no fim |
| `-h`, `--help` | mostra o uso |

As duas primeiras podem ir juntas.

### Dependências de compilação do AUR

Por padrão o build não instala nada na sua máquina. Se faltar uma dependência
de compilação, o pacote falha, fica de fora do ISO e o relatório diz o que
falta. O caso comum é o `gradle`: o `limine-mkinitcpio-hook` e o
`limine-snapper-sync` precisam dele para compilar (e de `git`; o segundo
também de `gettext`). Numa máquina que já tem tudo isso instalado nada falha.
Sem esses dois o Mk instalado ainda inicia (o instalador grava entradas fixas
no menu de boot), mas fica sem o submenu de snapshots.

Duas saídas. Deixar o `makepkg` instalar e remover sozinho tudo o que faltar
(não só o `gradle`):

```bash
iso/build --syncdeps
```

Ou instalar você, uma vez, o que o relatório pedir:

```bash
sudo pacman -S --asdeps gradle
```

Dependências de execução que faltam na sua máquina não impedem o build: o
pacote é compilado com `--nodeps` e elas chegam no `pacstrap` da instalação.

### O que o build faz

1. **AUR**: lê `system/packages/aur.packages`, baixa cada PKGBUILD do AUR
   (o nome do pacote vira o `PackageBase` pela API do AUR), compila com
   `makepkg` e monta um repositório local, `mk-aur`, em `iso/work/aur/repo`.
   Dependências que também só existem no AUR entram na fila sozinhas. Um
   pacote que não compila fica de fora do ISO e o build continua. A lista do
   que entrou vai em `iso/work/aur/repo/packages.txt`; é ela que o instalador
   instala.
2. **Perfil**: copia `/usr/share/archiso/configs/releng` para
   `iso/work/profile` e põe por cima o que está em `iso/profile/`:
   `profiledef.sh` (nome `mk`, rótulo `MK`), `packages.mk` (`gum` e
   `pciutils`, somados ao `packages.x86_64` do releng), teclado `br-abnt2` e
   hostname `mk-live` no live, o `motd`, o fundo do menu de boot em BIOS
   (`syslinux/splash.png`) e os menus de boot renomeados para "Mk". No menu
   UEFI, as entradas `01-archiso-linux` e `02-archiso-speech-linux` do releng
   saem e entra `efiboot/loader/entries/01-mk.conf`: o ISO do Mk não tem a
   entrada de acessibilidade por voz do Arch nesse menu. O repositório do Mk
   inteiro (sem `.git`, `iso/work` e `iso/out`) vai para `/usr/share/mk` no
   live, o repositório do AUR para `/usr/share/mk/aur-repo`, a versão
   (`commit data`) para `/usr/share/mk/VERSION`, o instalador para
   `/usr/local/bin/mk-install` e o gancho do pacman que põe o nome Mk no
   sistema para `/etc/pacman.d/hooks/mk-os-release.hook`.
3. **ISO**: `sudo mkarchiso -v -r -w iso/work/mkarchiso -o iso/out iso/work/profile`.

O `releng` vem de `/usr/share/archiso/configs/releng`; para usar outra cópia,
aponte `RELENG` para ela.

### Saídas

| Caminho | O que é |
| --- | --- |
| `iso/out/mk-AAAA.MM.DD-x86_64.iso` | o ISO |
| `iso/work/aur/report.txt` | uma linha por pacote do AUR: `ok` ou `FALHOU` e o motivo |
| `iso/work/aur/<PackageBase>.log` | a saída do `makepkg` de cada pacote |
| `iso/work/aur/repo/` | o repositório `mk-aur` e o `packages.txt` |
| `iso/work/aur/src/`, `iso/work/aur/pkg/` | PKGBUILDs clonados e pacotes compilados |
| `iso/work/profile/` | o perfil montado, refeito a cada build |
| `iso/work/mkarchiso/` | área de trabalho do `mkarchiso` (dono root) |

Leia o `report.txt` antes de gravar o pendrive: um `FALHOU` ali é um pacote
que o Mk instalado não vai ter.

Para limpar tudo:

```bash
sudo rm -rf iso/work iso/out
```

## Gravar num pendrive

**O `dd` apaga o pendrive inteiro, sem perguntar. Se você errar o destino, ele
apaga outro disco.** Por isso use o caminho em `/dev/disk/by-id`, que traz
marca, modelo e número de série, e não `/dev/sdX`, que muda de um boot para o
outro.

Com o pendrive espetado, ache o nome dele (a linha com `usb` na coluna `TRAN`):

```bash
lsblk -dpo NAME,SIZE,TRAN,MODEL,ID-LINK
```

Grave, trocando a data do ISO e o `usb-...` pelo que apareceu na coluna
`ID-LINK` (o disco inteiro, sem `-part1` no fim):

```bash
sudo dd if=iso/out/mk-AAAA.MM.DD-x86_64.iso of=/dev/disk/by-id/usb-MARCA_MODELO_SERIE-0:0 bs=4M conv=fsync oflag=direct status=progress
```

Escreva o nome do arquivo por extenso. No Mk o `ls` é um apelido do `eza`
(`config/bash/mk.bash`), e nele `-t` não é "ordenar por data" como no `ls` do
GNU: não monte o nome com `$(ls -t iso/out | head -1)`.

Inicie o computador pelo pendrive **em modo UEFI**. O ISO também inicia em
BIOS legado (o `releng` traz o syslinux), mas o `mk-install` para logo no
começo se o live não estiver em UEFI.

## Testar no QEMU

Precisa de `qemu-desktop` e `edk2-ovmf`, e de KVM:

```bash
sudo pacman -S --needed qemu-desktop edk2-ovmf
```

| Comando | O que faz |
| --- | --- |
| `iso/vm iso` | inicia pelo ISO mais novo de `iso/out` (para rodar `mk-install`) |
| `iso/vm`, `iso/vm disk` | inicia pelo disco da VM (o Mk já instalado) |
| `iso/vm reset` | apaga o disco e as variáveis do UEFI da VM |

`iso/vm iso` não gera o ISO nem avisa se ele é antigo: pega o `mk-*.iso`
modificado por último em `iso/out` e, se não houver nenhum, para e manda rodar
`iso/build`.

A VM fica em `~/.cache/mk-vm`: um disco qcow2 de 40 GB (cresce conforme usa)
e `vars.fd`, as variáveis do UEFI, onde fica a entrada "Mk" do boot. Ela tem
4 processadores, 6 GB de memória, rede por cabo virtual e som pelo PipeWire.
Nenhum disco de verdade é tocado.

O caminho de um teste completo:

```bash
iso/vm iso
```

No live, rode `mk-install` e escolha `/dev/vda`. Quando terminar, feche a VM
e inicie pelo disco:

```bash
iso/vm
```

Para repetir a instalação do zero:

```bash
iso/vm reset
```

Para instalar sem responder perguntas, veja o modo `--answers` em
[installer/README.md](../installer/README.md).

### Vídeo

A VM usa a placa de vídeo da sua máquina (`virtio-vga-gl`, janela SDL com
OpenGL), para o Mk rodar com aceleração. Sem isso tudo é desenhado no
processador e as animações engasgam. Se a janela não abrir ou ficar preta:

```bash
MK_VM_GL=0 iso/vm
```

### Entrar na VM de fora

A porta 2222 da sua máquina vai para a porta 22 da VM. No Mk instalado o
`sshd` vem desligado e o firewall (ufw) nega toda entrada, a porta 22
inclusive. Dentro da VM, ligue o `sshd`:

```bash
sudo systemctl start sshd
```

E abra a porta no firewall:

```bash
sudo ufw allow 22/tcp
```

Então, da sua máquina:

```bash
ssh -p 2222 usuario@localhost
```

Esse caminho nunca foi testado de ponta a ponta.

`~/.cache/mk-vm/monitor.sock` é o monitor do QEMU. Com o `socat`, ele tira uma
captura da tela da VM:

```bash
echo "screendump /tmp/vm.ppm" | socat - UNIX:$HOME/.cache/mk-vm/monitor.sock
```

### O que a VM não mostra

- **Suspender**: a tela da VM pode não voltar depois de suspender. Não conclua
  nada sobre suspensão a partir da VM.
- **Wi-Fi**: a VM só tem rede por cabo. A escolha de rede Wi-Fi do instalador
  nunca aparece nela (e nunca foi testada).
- **Drivers**: na VM o instalador marca `vm` e também `amd` ou `intel`, pela
  CPU da sua máquina (a VM roda com `-cpu host`). O vídeo, porém, é o virtual:
  os drivers de vídeo desses dois perfis (`vulkan-radeon`, `vulkan-intel`) são
  instalados mas não são usados, e o perfil `nvidia` nunca aparece.
- **LUKS e BIOS legado**: dá para escolher LUKS na VM, mas isso nunca foi
  feito; BIOS legado a VM não tem (ela é sempre UEFI).
- **Hardware de verdade**: mais de um monitor, impressoras, Bluetooth, bateria
  e brilho da tela.
