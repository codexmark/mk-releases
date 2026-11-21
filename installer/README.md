# Instalador do Mk

`installer/mk-install` roda no live do ISO, como root, e instala o Mk num
**disco inteiro**: UEFI, GPT, btrfs com subvolumes, limine, snapper, os pacotes
de `system/packages` e o desktop Mk. Não faz dual boot, não instala em BIOS
legado e não aceita disco com menos de 24 GiB.

Gerar o ISO, gravar no pendrive e testar numa VM está em
[iso/README.md](../iso/README.md).

```bash
mk-install
```

`mk-install -h` (ou `--help`) mostra o uso.

Precisa de internet: os pacotes vêm dos espelhos do Arch. Os do AUR vêm já
compilados no ISO. Leva uns 15 minutos, quase tudo baixando pacotes.

## O que foi testado

**Testado, só em VM** (QEMU, UEFI, sem LUKS, rede por cabo): a instalação
inteira, o boot pelo limine e os snapshots. Na VM o vídeo é virtual: o perfil
`vm` é o único cujo driver de vídeo foi de fato usado.

**Nunca testado:**

- disco criptografado (LUKS);
- o perfil de drivers `nvidia`, e os drivers de vídeo dos perfis `amd` e
  `intel` numa placa de verdade;
- a escolha de rede Wi-Fi;
- instalação em hardware de verdade.

O código desses caminhos existe e está descrito abaixo, mas ninguém o viu
funcionar. Numa máquina que importa, tenha cópia dos seus arquivos e um
pendrive de outro sistema à mão.

## As perguntas

Antes de tudo ele confere: é root, o live está em UEFI (senão para e pede para
reiniciar pela entrada UEFI do pendrive), as ferramentas do live existem e
`/mnt` não está montado.

| Tela | O que pede |
| --- | --- |
| **00 Internet** | só aparece sem conexão: lista as redes Wi-Fi (`iwctl`), pede a senha e repete até conectar. Sem cabo e sem placa Wi-Fi, para. |
| **01 Disco** | em qual disco instalar. O pendrive do live não aparece na lista. |
| **02 Você** | nome do computador (padrão `mk`), usuário e senha, digitada duas vezes. A senha vale para o `sudo`. |
| **03 Região** | `America/Porto_Velho`, `br-abnt2` e `pt_BR.UTF-8`. Se você recusar os padrões, escolhe fuso, teclado e idioma em listas com busca. |
| **04 Drivers e apps** | `amd`, `intel`, `nvidia`, `vm`: já vem marcado o que foi detectado (CPU, placa de vídeo, VM); Espaço marca, Enter segue. Depois: instalar os apps gráficos (padrão sim) e criptografar o disco com LUKS (padrão não; a senha do disco é a do usuário). |
| **05 Revisão** | o resumo das respostas e, em vermelho, o que há hoje no disco. |

Até a revisão nada foi alterado e `Ctrl+C` cancela. Para seguir, escreva
`formatar` e Enter (maiúsculas e espaços são ignorados). Qualquer outra coisa
cancela sem tocar no disco.

O usuário aceita letras minúsculas, números, `-` e `_`, começando por letra ou
`_`, com até 32 caracteres, e não pode ser `root`. O hostname aceita letras,
números e `-`.

As telas usam o `gum`. Sem ele, ou sem terminal, as mesmas perguntas são
feitas em texto simples, uma por linha.

## O que acontece no disco

Depois do `formatar` ele trabalha sozinho, em etapas numeradas com o tempo
decorrido (`[ 5/15]`; 16 etapas com LUKS). Na ordem em que acontecem:

1. **Internet**: confere de novo (15 tentativas) e para se não houver.
2. **Disco**: desliga swap, desmonta o que estiver montado do disco, apaga as
   assinaturas (`wipefs`) e a tabela de partições e cria uma GPT com duas
   partições:

   | Partição | Tamanho | Sistema | Montagem |
   | --- | --- | --- | --- |
   | 1, `MK_EFI` | 2 GiB | FAT32 (ESP) | `/boot` |
   | 2, `mk` | o resto | btrfs, rótulo `mk` | veja os subvolumes |

   Com LUKS, a partição 2 vira um LUKS2 aberto como `/dev/mapper/mkroot`, e o
   btrfs fica dentro dele.
3. **Subvolumes btrfs**, montados com `noatime,compress=zstd`:

   | Subvolume | Montagem |
   | --- | --- |
   | `@` | `/` |
   | `@home` | `/home` |
   | `@log` | `/var/log` |
   | `@pkg` | `/var/cache/pacman/pkg` |

   Não há swap em partição nem em arquivo.
4. **Pacotes**: acerta o relógio e as chaves do pacman e roda um `pacstrap`
   com `base.packages`, `desktop.packages`, `dev.packages`, as seções de
   `hardware.packages` escolhidas e, se pedido, `apps.packages` (mais o
   `libreoffice-fresh-pt-br` quando o idioma é `pt_BR`). É a parte demorada.
5. **Fstab, região, rede**: `fstab` por UUID e sem `subvolid` (um rollback de
   snapshot troca o `@` e o id muda); fuso, `locale.gen` (o idioma escolhido e
   `en_US.UTF-8`), `LANG`, teclado do console e da tela de login, hostname,
   `/etc/hosts`, e nomes `.local` pela rede (`mdns_minimal` no `nsswitch.conf`).
6. **O desktop Mk**: veja [Marca e desktop](#marca-e-desktop).
7. **Boot**: veja [Boot](#boot-limine).
8. **Pacotes do AUR**: veja [AUR](#pacotes-do-aur).
9. **Usuário**: veja [Usuário](#usuário-e-o-que-nasce-na-home).
10. **Snapshots**: `snapper` com a config `root` em `/`, sem timeline e com
    limite de 5 snapshots. O `snap-pac` (de `base.packages`) tira um snapshot
    antes e outro depois de cada transação do pacman. Se o `create-config`
    falhar, a instalação segue e avisa como configurar depois.
11. **Serviços e firewall**: veja [Serviços](#serviços-e-firewall).
12. **mkinitcpio** e o `limine.conf` final.
13. **LazyVim** para o usuário (`scripts/user/neovim.sh`). Se falhar, avisa e
    segue.
14. **Desmonta** tudo (e fecha o LUKS) e oferece reiniciar. No modo
    `--answers` não oferece: desliga se `POWEROFF=sim`, senão volta ao prompt.

### Pacotes do AUR

O AUR não é compilado na instalação. O `iso/build` já compilou os pacotes de
`system/packages/aur.packages` num repositório local, `mk-aur`, que vai no ISO
em `/usr/share/mk/aur-repo`. O instalador soma esse repositório ao
`pacman.conf` do live e instala exatamente a lista de
`/usr/share/mk/aur-repo/packages.txt`.

Quando falta algum pacote:

- se a instalação conjunta falha, ele tenta um por um e avisa
  `não instalou <pacote>` para cada um que não entrar;
- no fim compara com `aur.packages` e grava o que ficou faltando em
  `/usr/share/mk/aur-pendentes.txt`, no sistema instalado;
- a instalação **não para** por causa disso.

Depois do primeiro login, instale o que faltou (o `yay` precisa ter vindo no
ISO; ele também é um pacote do AUR):

```bash
yay -S $(cat /usr/share/mk/aur-pendentes.txt)
```

Se o ISO veio sem nenhum pacote do AUR, ele só avisa e não grava esse arquivo;
a lista é a de `/usr/share/mk/system/packages/aur.packages`.

O que muda sem cada um: sem `limine-mkinitcpio-hook` o menu de boot fica com as
entradas fixas e não se atualiza sozinho com kernels novos; sem
`limine-snapper-sync` não há o submenu de snapshots no boot.

### Boot (limine)

- **Initramfs**: `/etc/mkinitcpio.conf.d/mk.conf` com os hooks `base systemd
  plymouth autodetect microcode modconf kms keyboard sd-vconsole block
  filesystems fsck`. Com LUKS entra `sd-encrypt`; com o perfil `nvidia` sai o
  `kms`, para o nouveau não carregar antes.
- **Linha do kernel**: `root=UUID=... rootflags=subvol=@ rw quiet splash`; com
  LUKS, `rd.luks.name=<UUID>=mkroot root=/dev/mapper/mkroot ...`.
- **Limine na ESP**: `EFI/limine/BOOTX64.EFI` e uma cópia no caminho padrão
  `EFI/BOOT/BOOTX64.EFI`, para firmware sem entrada na NVRAM ainda achar o Mk.
- **Entrada na NVRAM**: uma só, chamada "Mk", criada com o `efibootmgr`; se
  não der, avisa e segue. Com os ganchos do limine instalados ela aponta para
  `EFI/limine/limine_x64.efi` (o binário que eles mantêm atualizado), a
  entrada "Limine" que o gancho registrou sai e a cópia `EFI/limine/BOOTX64.EFI`
  é apagada. Uma "Mk" de instalação anterior também sai. O firmware ainda pode
  criar a sua ("UEFI OS") para o caminho padrão; isso não é do instalador.
- **`/etc/default/limine`**: lido pelo `limine-mkinitcpio-hook` a cada kernel
  novo (nome "Mk", ESP em `/boot`, a linha do kernel, snapshots no menu no modo
  `auto`, que poda os mais antigos pelo espaço da ESP, e sem a entrada extra
  "EFI fallback").
- **`limine.conf`**: cabeçalho com as cores do Mk e 3 segundos de espera. Se o
  `limine-update` (do `limine-mkinitcpio-hook`) existir, ele escreve as
  entradas. Se não existir, ou se o resultado apontar para um kernel que não
  está em `/boot`, o instalador grava entradas fixas: "Mk" e, se o initramfs
  de fallback existir, "Mk (fallback)". Se nem assim o arquivo apontar para um
  kernel que existe, a instalação para.
- **`system/mk/boot/limine-default`**: roda no fim. Com os ganchos do limine a
  entrada 1 do menu é um grupo ("Mk", com "linux" e "Snapshots" dentro), e um
  grupo só abre e fecha: a contagem acabava e nada iniciava. O script acha a
  primeira entrada que inicia de verdade, abre os grupos acima dela e grava a
  posição em `default_entry`. Pode rodar de novo quando quiser:

```bash
sudo bash /usr/share/mk/system/mk/boot/limine-default
```

### Marca e desktop

- O repositório inteiro vai para `/usr/share/mk` (sem `.git`, `aur-repo`,
  `iso/work` e `iso/out`), com o `VERSION` do ISO.
- **Nome do sistema**: `/etc/os-release` e `/etc/issue` do Mk, e o gancho
  `/etc/pacman.d/hooks/mk-os-release.hook`, que os repõe a cada atualização do
  pacote `filesystem`.
- **Sessão**: `/usr/bin/mk-session`, `/usr/bin/mk-update` e
  `/usr/share/wayland-sessions/mk.desktop`.
- **Aparência**: tema GTK em `/usr/share/themes/Mk`, ícones gerados em
  `/usr/share/icons/Mk` (`system/mk/look/make-icons`), fontes em
  `/usr/share/fonts/mk`, tema do plymouth `mk` e tema do sddm `mk`, com o
  teclado escolhido.
- **Migrações**: todas as de `migrations/system` são marcadas como feitas em
  `/var/lib/mk/migrations`, e as de `migrations/user` em
  `/etc/skel/.local/state/mk/migrations`. Uma instalação nova já tem o que
  elas fariam; o `mk-update` só roda as que chegarem depois.
- **Apps oferecidos**: a lista de `apps.packages` é gravada em
  `/var/lib/mk/apps-oferecidos`, tenha você aceitado os apps ou não. O
  `mk-update` só oferece os que entrarem na lista depois.

### Usuário e o que nasce na home

As configs do Mk vão para `/etc/skel`, de onde o `useradd -m` as leva para a
home. Todo usuário criado depois nasce igual.

| No skel | De onde vem |
| --- | --- |
| `.config/mk/hypr` | `config/hypr` |
| `.config/mk/shell` | `shell/mare` |
| `.config/mk/ghostty` | `config/ghostty` |
| `.config/mk/fastfetch` | `config/fastfetch` |
| `.config/mk/bash` | `config/bash` |
| `.config/mk/seu.bash` | `config/bash/seu.bash.exemplo` |
| `.config/mimeapps.list` | `config/mimeapps.list` (apps padrão) |
| `.config/qt6ct` | `system/mk/look/qt6ct` |
| `.bashrc` | ganha a linha que carrega `~/.config/mk/bash/mk.bash` |
| `Work/` | pasta vazia |

As cinco pastas de `.config/mk` (`hypr`, `shell`, `ghostty`, `fastfetch`,
`bash`) levam o arquivo `.instalado-pelo-mk`, a mesma marca do `bin/new-user`.
O resto da tabela não leva.

O usuário é criado com `bash`, no grupo `wheel` e, se existirem, em `video`,
`audio`, `input` e `docker`. O `wheel` tem sudo com senha
(`/etc/sudoers.d/10-mk-wheel`). **O root fica bloqueado.** Se as configs
`hypr`, `ghostty` e `shell` não chegarem na home, a instalação para.

### Serviços e firewall

Habilitados, cada um só se a unidade existir no sistema instalado:

| Unidade | Para quê |
| --- | --- |
| `NetworkManager.service` | rede |
| `bluetooth.service` | Bluetooth |
| `sddm.service` | tela de login |
| `avahi-daemon.service` | nomes `.local` |
| `power-profiles-daemon.service` | perfis de energia |
| `systemd-timesyncd.service` | relógio |
| `ufw.service` | firewall |
| `docker.socket` | Docker, sob demanda |
| `cups.socket` | impressão, sob demanda |
| `fstrim.timer` | TRIM semanal |
| `plocate-updatedb.timer` | índice do `locate` |
| `snapper-cleanup.timer` | apaga snapshots além do limite |
| `limine-snapper-sync.service` | snapshots no menu de boot (AUR) |

**Firewall (ufw)**: nega entrada, libera saída, abre a porta 53317 (TCP e UDP,
LocalSend) e o DNS do Docker (UDP 53 de `172.16.0.0/12` para `172.17.0.1`).
São as mesmas regras de `scripts/system/firewall.sh`. No chroot só se grava a
regra; o ufw liga no primeiro boot.

O instalador **não** roda o `ufw-docker install`, o passo daquele script que
impede o Docker de abrir portas por fora do ufw (o pacote `ufw-docker`, do
AUR, é instalado). Para ter esse passo, rode o script depois do primeiro boot:

```bash
/usr/share/mk/scripts/system/firewall.sh
```

**Impressão**: o grupo `wheel` entra no `SystemGroup` do CUPS, se ainda não
estiver.

**Tela de login**: o sddm já vem com o seu usuário e a sessão Mk escolhidos.
Com LUKS ele entra direto (`/etc/sddm.conf.d/10-mk.conf`), porque a senha já
foi digitada no boot.

## Depois de instalar

Tire o pendrive e reinicie. Para instalar node, python e as CLIs de IA no seu
usuário, com internet:

```bash
/usr/share/mk/scripts/user/mise-tools.sh
```

## Instalar sem perguntas

Feito para testes em VM. Copie `installer/answers.example`, ajuste e passe o
arquivo:

```bash
mk-install --answers respostas.conf
```

| Chave | Valor |
| --- | --- |
| `DISK` | o disco inteiro, ex. `/dev/vda` |
| `CONFIRM` | `formatar` (ou o mesmo caminho de `DISK`). Sem isso, para antes de apagar. |
| `HOSTNAME`, `USERNAME`, `PASSWORD` | como nas perguntas |
| `TIMEZONE`, `KEYMAP`, `LOCALE` | ex. `America/Porto_Velho`, `br-abnt2`, `pt_BR.UTF-8` |
| `HARDWARE` | `amd intel nvidia vm`, separados por espaço; vazio para nenhum |
| `APPS`, `ENCRYPT` | `sim` ou `nao` |
| `POWEROFF` | `sim` desliga a máquina no fim |

O arquivo é lido como `CHAVE=valor`, uma por linha, sem executar nada; uma
chave desconhecida ou uma linha fora desse formato para o instalador. Preencha
todas as chaves do exemplo: neste modo não há valores padrão de região. Com
`LOCALE` ou `KEYMAP` vazios o instalador para; com `TIMEZONE` vazio ele **não
para** e instala com o fuso quebrado.

Neste modo não há tela de Wi-Fi: a rede já precisa estar no ar. As respostas
passam pelas mesmas validações, e a revisão é mostrada antes de formatar.

A senha fica em texto puro no arquivo. Não use uma senha de verdade e não
deixe o arquivo em lugar compartilhado.

### Rodar de fora de `/usr/share/mk`

O instalador procura o Mk em `/usr/share/mk` e, se não achar, no repositório
de onde ele próprio está rodando. Duas variáveis trocam isso:

| Variável | O que aponta |
| --- | --- |
| `MK_SRC` | a cópia do repositório do Mk a instalar |
| `MK_AUR_REPO` | o repositório `mk-aur` (padrão `$MK_SRC/aur-repo`; se não existir, `iso/work/aur/repo` dentro da cópia) |

## Quando falha

Se parar depois do `formatar`, a última linha mostra o código de saída, o
caminho do log e como tentar de novo:

```bash
umount -R /mnt
```

```bash
cryptsetup close mkroot
```

(o segundo só se você escolheu LUKS) e então `mk-install` outra vez. Ele
recomeça do zero e formata o disco de novo.

### O que juntar para relatar

| O quê | Onde |
| --- | --- |
| Log da instalação | `/var/log/mk-install.log` no live |
| Cópia do log | `/var/log/mk-install.log` no sistema instalado (só se chegou ao fim) |
| Pacotes do AUR que faltaram | `/usr/share/mk/aur-pendentes.txt` no sistema instalado |
| Versão do Mk no ISO | `/usr/share/mk/VERSION` |
| Como o disco ficou | saída de `lsblk -f` |
| Do lado do build | `iso/work/aur/report.txt` e `iso/work/aur/<PackageBase>.log` |

Sobre o log:

- A etapa dos pacotes do AUR (a 9; a 10 com LUKS) só é numerada quando o ISO
  traz pacotes do AUR. Sem eles a contagem termina em `[14/15]` (`[15/16]` com
  LUKS): não é uma etapa que falhou.
- Ele começa **depois** da revisão. As perguntas, e um erro antes do
  `formatar` (sem UEFI, disco pequeno, sem internet no começo), só aparecem na
  tela: fotografe.
- O live roda da memória. **O log do live some ao desligar**; copie antes de
  reiniciar, para outro pendrive ou pela rede.
- A cópia no sistema instalado é feita antes de desmontar, então as últimas
  linhas (desmontagem) não estão nela.
- A senha não é escrita no log. O usuário, o disco e a saída inteira dos
  comandos são; leia antes de publicar.

Avisos em amarelo (`AVISO:`) não param a instalação: procure por eles no log,
cada um diz o que rodar depois.
