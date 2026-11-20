# ISO e instalador do Mk

`iso/` gera um ISO live (archiso, baseado no perfil `releng`) que inicia num
console com o instalador `mk-install`. O instalador está em `installer/`.

## Gerar o ISO

Precisa do `archiso` (só para gerar, não para usar o Mk):

```bash
sudo pacman -S --needed archiso
```

Rode como o seu usuário, sem sudo. O script compila os pacotes do AUR com
`makepkg` (que não roda como root) e só no fim chama `sudo mkarchiso`, que pede
a sua senha:

```bash
iso/build
```

O ISO sai em `iso/out/mk-AAAA.MM.DD-x86_64.iso`, com o rótulo de volume `MK`.

O que o build faz:

1. **AUR**: lê `system/packages/aur.packages`, baixa cada PKGBUILD do AUR
   (o nome do pacote vira o `PackageBase` pela API do AUR), compila com
   `makepkg` e monta um repositório local, `mk-aur`, em
   `iso/work/aur/repo`. Dependências que também só existem no AUR entram na
   fila sozinhas. Um pacote que não compila fica de fora do ISO, aparece em
   `iso/work/aur/report.txt` e o build continua.
   Por padrão nada é instalado na sua máquina: se faltar uma dependência de
   compilação, o pacote falha e o relatório mostra o que falta. Com
   `iso/build --syncdeps` o makepkg instala o que falta (pede sudo) e remove
   no fim.
2. **Perfil**: copia `/usr/share/archiso/configs/releng` para
   `iso/work/profile` e põe por cima o que está em `iso/profile/`:
   `profiledef.sh` (nome `mk`, rótulo `MK`), `packages.mk` (gum e pciutils,
   somados ao `packages.x86_64` do releng), teclado `br-abnt2` no live, o
   `motd` e o fundo do menu de boot na direção Folha, e os menus de boot
   renomeados para "Mk". O repositório do Mk inteiro (sem `.git`) vai para
   `/usr/share/mk` no live, o repositório do AUR para `/usr/share/mk/aur-repo`
   e o instalador para `/usr/local/bin/mk-install`.
3. **ISO**: `sudo mkarchiso -v -r -w iso/work/mkarchiso -o iso/out iso/work/profile`.

`iso/build --prepare` faz só os passos 1 e 2 (sem sudo) e imprime o comando
do passo 3.

A área do mkarchiso fica com dono root. Para limpar tudo:

```bash
sudo rm -rf iso/work iso/out
```

## Testar no QEMU

Precisa de `qemu-desktop` e `edk2-ovmf` (já instalados nesta máquina). O
instalador só funciona em UEFI, então a VM usa o OVMF:

```bash
mkdir -p ~/.cache/mk-vm && cd ~/.cache/mk-vm
qemu-img create -f qcow2 disk.qcow2 40G
cp /usr/share/edk2/x64/OVMF_VARS.4m.fd vars.fd
```

Iniciar pelo ISO:

```bash
qemu-system-x86_64 -enable-kvm -cpu host -smp 4 -m 6G -machine q35 -drive if=pflash,format=raw,readonly=on,file=/usr/share/edk2/x64/OVMF_CODE.4m.fd -drive if=pflash,format=raw,file=vars.fd -drive file=disk.qcow2,if=virtio -cdrom ~/Work/projetos/meu-os/iso/out/mk-*.iso -nic user,model=virtio-net-pci -device virtio-vga
```

No live, rode `mk-install` e escolha `/dev/vda`. Depois que terminar, inicie
de novo sem o `-cdrom` (o mesmo `vars.fd` guarda a entrada "Mk" do boot).

## O que o instalador faz

`mk-install` roda no live como root e pergunta:

- o disco (o pendrive do live não aparece na lista);
- hostname (padrão `mk`), usuário e senha (digitada duas vezes);
- região: `America/Porto_Velho`, `br-abnt2` e `pt_BR.UTF-8` por padrão, com
  opção de trocar;
- drivers: `[amd] [intel] [nvidia] [vm]` de `hardware.packages`, já
  marcados pelo que foi detectado (CPU, placa de vídeo, VM);
- se instala os apps de `apps.packages`;
- se criptografa o disco com LUKS (a senha do disco é a do usuário).

Antes de apagar, mostra o resumo e um aviso em vermelho com o que há no disco,
e só continua se você **digitar o caminho do disco** (ex.: `/dev/nvme0n1`).
Qualquer outra coisa cancela sem tocar em nada.

Depois:

1. **Disco**: GPT, ESP FAT32 de 2 GiB em `/boot` e o resto em btrfs (zstd,
   noatime) com os subvolumes `@`, `@home`, `@log` e `@pkg`; com LUKS, o
   btrfs fica dentro de um LUKS2.
2. **Pacotes**: `pacstrap` com `base`, `desktop`, `dev`, as seções de
   hardware escolhidas e, se pedido, `apps` (lidas com `bin/lib-packages.sh`).
   Precisa de internet. Depois instala os pacotes do AUR a partir do
   repositório local do ISO; o que não veio no ISO fica anotado em
   `/usr/share/mk/aur-pendentes.txt` para instalar com `yay` depois.
3. **Sistema**: fstab, fuso, locale, teclado (console e tela de login),
   hostname, mDNS (`.local`), usuário no `wheel` com sudo, root bloqueado.
4. **Desktop Mk**: o repositório em `/usr/share/mk`, `/usr/bin/mk-session`,
   `/usr/share/wayland-sessions/mk.desktop` e as configs em
   `/etc/skel/.config/mk/{hypr,shell,ghostty}` (com a marca
   `.instalado-pelo-mk`, como o `bin/new-user`), de onde vão para a home do
   usuário. Também cria `~/Work` e `~/Pictures/Wallpapers` e aplica o LazyVim
   (`scripts/user/neovim.sh`).
5. **Boot**: initramfs com systemd, plymouth e (com LUKS) sd-encrypt; limine
   em `EFI/limine` e na cópia padrão `EFI/BOOT/BOOTX64.EFI`, entrada "Mk" na
   NVRAM, `limine.conf` com as cores do Mk. Se o `limine-mkinitcpio-hook`
   veio do AUR, ele passa a escrever as entradas (`/etc/default/limine`); se o
   resultado não apontar para um kernel que existe, ficam as entradas fixas.
6. **Snapshots**: snapper na raiz (sem timeline, 5 snapshots; o `snap-pac`
   tira um a cada pacman).
7. **Serviços**: NetworkManager, bluetooth, sddm, avahi, power-profiles-daemon,
   timesyncd, ufw (com as regras de `scripts/system/firewall.sh`),
   `docker.socket`, fstrim, plocate e snapper-cleanup. O sddm já vem com o
   usuário e a sessão Mk escolhidos; com LUKS, entra direto (a senha já foi
   digitada no boot).

O log fica em `/var/log/mk-install.log` (no live e no sistema instalado).
A senha não vai para o log.

Para testes sem perguntas existe `mk-install --answers <arquivo>`; veja
`installer/answers.example`. O arquivo precisa repetir o disco em `CONFIRM`.
