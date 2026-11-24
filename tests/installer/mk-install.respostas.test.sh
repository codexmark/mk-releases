#!/usr/bin/env bash
# installer/mk-install: as respostas (arquivo de respostas, validação, discos
# oferecidos, hardware detectado) e a revisão antes de formatar.
# shellcheck disable=SC2154,SC2034 # as variáveis vêm do instalador (carrega.sh)
. "$(dirname "$0")/../lib.sh"
. "$(dirname "$0")/carrega.sh"

# --- opções da linha de comando (param antes do main) -----------------------
t "-h: mostra o uso e não instala nada"
out=$(bash "$mk_install" -h 2>&1)
eq 0 $?
has "$out" "mk-install --answers <arquivo>"
out=$(bash "$mk_install" --help 2>&1)
has "$out" "Instalador do Mk"

t "opção desconhecida: recusa com mensagem"
out=$(bash "$mk_install" --formatar-tudo 2>&1)
eq 1 $?
has "$out" "opção desconhecida: --formatar-tudo"

t "--answers sem arquivo: recusa com mensagem, não cai no modo de perguntas"
out=$(bash "$mk_install" --answers 2>&1)
eq 1 $?
has "$out" "--answers pede o arquivo"

t "carregado com source, o main não roda"
eq "" "$(<"$SBX/calls")"
eq "$SBX/mnt" "$target"

# --- valid_user / valid_host -------------------------------------------------
t "valid_user: minúsculas, números, - e _, começando por letra ou _"
for u in mark m _svc mark-2 a_b user01 "$(printf 'a%.0s' {1..32})"; do
  ok valid_user "$u"
done

t "valid_user: recusa root, maiúscula, começo com número ou -, espaço, vazio e mais de 32"
for u in root Mark 1mark -mark "ma rk" "" 'mark$' "ção" "$(printf 'a%.0s' {1..33})"; do
  fails valid_user "$u"
done

t "valid_host: letras, números e -"
for h in mk Mk-01 a 9 "mk-live" "$(printf 'a%.0s' {1..63})"; do
  ok valid_host "$h"
done

t "valid_host: recusa -, espaço, ponto, _ e vazio; e mais de 63"
for h in -mk mk- "m k" mk.local mk_1 "" "$(printf 'a%.0s' {1..64})"; do
  fails valid_host "$h"
done

# --- part_name ---------------------------------------------------------------
t "part_name: sda vira sda1; nvme e mmcblk ganham o p"
eq /dev/sda1 "$(part_name /dev/sda 1)"
eq /dev/vda2 "$(part_name /dev/vda 2)"
eq /dev/nvme0n1p1 "$(part_name /dev/nvme0n1 1)"
eq /dev/nvme1n1p2 "$(part_name /dev/nvme1n1 2)"
eq /dev/mmcblk0p2 "$(part_name /dev/mmcblk0 2)"

# --- load_answers ------------------------------------------------------------
mostra() { declare -p DISK HOSTNAME_ USERNAME PASSWORD TIMEZONE KEYMAP LOCALE HARDWARE APPS ENCRYPT CONFIRM POWEROFF; }

t "load_answers: o answers.example é lido inteiro"
roda eval 'load_answers "$REPO/installer/answers.example"; mostra'
eq 0 "$rc"
has "$out" 'DISK="/dev/vda"'
has "$out" 'HOSTNAME_="mk"'
has "$out" 'USERNAME="mark"'
has "$out" 'PASSWORD="troque-esta-senha"'
has "$out" 'TIMEZONE="America/Porto_Velho"'
has "$out" 'KEYMAP="br-abnt2"'
has "$out" 'LOCALE="pt_BR.UTF-8"'
has "$out" 'HARDWARE="vm"'
has "$out" 'APPS="nao"'
has "$out" 'ENCRYPT="nao"'
has "$out" 'CONFIRM="formatar"'
has "$out" 'POWEROFF="sim"'

t "load_answers: o exemplo traz todas as chaves que o instalador aceita"
for k in DISK CONFIRM HOSTNAME USERNAME PASSWORD TIMEZONE KEYMAP LOCALE HARDWARE APPS ENCRYPT POWEROFF; do
  ok grep -q "^$k=" "$REPO/installer/answers.example"
done

t "load_answers: aspas em volta, vários perfis e a última linha sem quebra"
printf 'HARDWARE="amd nvidia"\nPASSWORD=a b=c#d\nUSERNAME=ana' >"$SBX/r.conf"
roda eval 'load_answers "$SBX/r.conf"; mostra'
eq 0 "$rc"
has "$out" 'HARDWARE="amd nvidia"'
has "$out" 'PASSWORD="a b=c#d"'
has "$out" 'USERNAME="ana"'

t "load_answers: nada do arquivo é executado"
# shellcheck disable=SC2016
printf '%s\n' 'PASSWORD=$(touch '"$SBX"'/executou)' 'USERNAME=`touch '"$SBX"'/executou`' >"$SBX/r.conf"
roda eval 'load_answers "$SBX/r.conf"; mostra'
eq 0 "$rc"
absent "$SBX/executou"
has "$out" 'touch'

t "load_answers: chave desconhecida para o instalador"
printf 'DISK=/dev/sdb\nSWAP=sim\n' >"$SBX/r.conf"
roda load_answers "$SBX/r.conf"
eq 1 "$rc"
has "$out" "chave desconhecida"
has "$out" "SWAP"

t "load_answers: linha fora do formato CHAVE=valor para o instalador"
for linha in 'disk=/dev/sdb' 'rm -rf /' 'export DISK=/dev/sdb' 'DISK /dev/sdb' ' DISK=/dev/sdb'; do
  printf '%s\n' "$linha" >"$SBX/r.conf"
  roda load_answers "$SBX/r.conf"
  eq 1 "$rc" "linha: $linha"
  has "$out" "linha inválida"
done

t "load_answers: arquivo que não existe para com mensagem"
roda load_answers "$SBX/nao-existe.conf"
eq 1 "$rc"
has "$out" "não consigo ler"

# --- list_disks / live_disk --------------------------------------------------
t "live_disk: o disco inteiro de onde o live foi iniciado"
eq /dev/sdc "$(live_disk)"

t "live_disk: sem live montado (ou live num CD), nada"
stub findmnt 'exit 1'
eq "" "$(live_disk)"
stub findmnt 'echo /dev/sr0'
eq "" "$(live_disk)"
stub findmnt '[[ "$*" == "-no SOURCE /run/archiso/bootmnt" ]] && echo /dev/sdc1'

t "list_disks: só discos inteiros, com tamanho e modelo"
out=$(list_disks)
has "$out" "/dev/sda  465,8G  Samsung SSD 860"
has "$out" "/dev/nvme0n1  931,5G  WD_BLACK SN770"
eq 3 "$(wc -l <<<"$out")"

t "list_disks: nunca oferece o pendrive do live, nem zram, CD, loop ou partição"
hasnt "$out" "/dev/sdc"
hasnt "$out" "zram"
hasnt "$out" "/dev/sr0"
hasnt "$out" "loop"
hasnt "$out" "/dev/sda1"

t "list_disks: com o live num nvme, é ele que some"
stub findmnt 'echo /dev/nvme0n1p1'
out=$(list_disks)
hasnt "$out" "nvme0n1"
has "$out" "/dev/sdc"
stub findmnt '[[ "$*" == "-no SOURCE /run/archiso/bootmnt" ]] && echo /dev/sdc1'

# --- detect_hardware ---------------------------------------------------------
cpu() { printf 'processor\t: 0\nvendor_id\t: %s\n' "$1" >"$SBX/live/proc/cpuinfo"; }
stub systemd-detect-virt 'exit 1'

t "detect_hardware: CPU e vídeo AMD dão um amd só"
cpu AuthenticAMD
stub lspci 'echo "0b:00.0 VGA compatible controller: Advanced Micro Devices, Inc. [AMD/ATI] Navi 10 [Radeon RX 5600 OEM/5600 XT / 5700/5700 XT] (rev c1)"
echo "0b:00.1 Audio device: Advanced Micro Devices, Inc. [AMD/ATI] Navi 10 HDMI Audio"'
eq "amd" "$(detect_hardware)"

t "detect_hardware: CPU Intel com vídeo integrado dá intel"
cpu GenuineIntel
stub lspci 'echo "00:02.0 VGA compatible controller: Intel Corporation TigerLake-LP GT2 [Iris Xe Graphics] (rev 01)"'
eq "intel" "$(detect_hardware)"

t "detect_hardware: notebook híbrido Intel + NVIDIA (3D controller)"
stub lspci 'echo "00:02.0 VGA compatible controller: Intel Corporation UHD Graphics 630"
echo "01:00.0 3D controller: NVIDIA Corporation TU117M [GeForce GTX 1650 Mobile / Max-Q] (rev a1)"'
eq "intel nvidia" "$(detect_hardware)"

t "detect_hardware: CPU AMD com placa NVIDIA"
cpu AuthenticAMD
stub lspci 'echo "01:00.0 VGA compatible controller: NVIDIA Corporation GA106 [GeForce RTX 3060] (rev a1)"
echo "01:00.1 Audio device: NVIDIA Corporation GA106 High Definition Audio Controller (rev a1)"'
eq "amd nvidia" "$(detect_hardware)"

t "detect_hardware: CPU Intel com placa AMD dá os dois"
cpu GenuineIntel
stub lspci 'echo "03:00.0 Display controller: Advanced Micro Devices, Inc. [AMD/ATI] Navi 23"'
eq "intel amd" "$(detect_hardware)"

t "detect_hardware: a placa de som de outro fabricante não marca driver de vídeo"
cpu GenuineIntel
stub lspci 'echo "00:02.0 VGA compatible controller: Intel Corporation UHD Graphics 630"
echo "01:00.1 Audio device: NVIDIA Corporation High Definition Audio Controller"
echo "02:00.0 Audio device: Advanced Micro Devices, Inc. [AMD] Family 17h HD Audio Controller"'
eq "intel" "$(detect_hardware)"

t "detect_hardware: numa VM entra vm, junto com a CPU"
cpu AuthenticAMD
stub systemd-detect-virt 'exit 0'
stub lspci 'echo "00:01.0 VGA compatible controller: Red Hat, Inc. Virtio 1.0 GPU (rev 01)"'
eq "amd vm" "$(detect_hardware)"
stub systemd-detect-virt 'exit 1'

t "detect_hardware: sem lspci e com CPU desconhecida, nada (e não quebra)"
cpu HygonGenuine
stub lspci 'exit 127'
roda detect_hardware
eq 0 "$rc"
eq "" "$out"
cpu AuthenticAMD

# --- validate ----------------------------------------------------------------
# valida [CHAVE=valor...]: as respostas boas com as trocas pedidas.
_com() { # _com <função> [CHAVE=valor...]
  local f=$1 kv
  shift
  respostas_boas
  for kv in "$@"; do printf -v "${kv%%=*}" '%s' "${kv#*=}"; done
  "$f"
}
valida() { roda _com validate "$@"; }

t "validate: as respostas boas passam"
valida
eq 0 "$rc" "$out"
eq "" "$out"

t "validate: o arquivo de exemplo, com um disco que existe, passa"
roda eval 'load_answers "$(respostas)"; validate'
eq 0 "$rc" "$out"

recusa() { # recusa <mensagem> CHAVE=valor...
  local msg=$1
  shift
  valida "$@"
  eq 1 "$rc" "devia recusar: $*"
  has "$out" "ERRO:" "$*"
  has "$out" "$msg" "$*"
}

t "validate: disco que não existe, partição e disco pequeno são recusados"
recusa "disco inválido: /dev/sdz" DISK=/dev/sdz
recusa "disco inválido" DISK=
recusa "/dev/sda1 não é um disco inteiro" DISK=/dev/sda1
recusa "/dev/sr0 não é um disco inteiro" DISK=/dev/sr0
printf '%s\n' '/dev/sdd disk 25769803775 24G Pequeno -' >>"$SBX/discos"
recusa "/dev/sdd tem menos de 24 GiB" DISK=/dev/sdd
printf '%s\n' '/dev/sde disk 25769803776 24G Exato -' >>"$SBX/discos"
valida DISK=/dev/sde
eq 0 "$rc" "24 GiB exatos servem: $out"

t "validate: o pendrive do live é recusado"
recusa "/dev/sdc é o pendrive do live" DISK=/dev/sdc

t "validate: hostname, usuário e senha"
recusa "hostname inválido: meu pc" "HOSTNAME_=meu pc"
recusa "hostname inválido" HOSTNAME_=
recusa "usuário inválido: Mark" USERNAME=Mark
recusa "usuário inválido: root" USERNAME=root
recusa "usuário inválido" USERNAME=
recusa "senha vazia" PASSWORD=

t "validate: fuso que não existe, vazio, pasta ou caminho para fora"
recusa "fuso inválido: Marte/Olympus" TIMEZONE=Marte/Olympus
recusa "fuso inválido" TIMEZONE=
recusa "fuso inválido: America" TIMEZONE=America
recusa "fuso inválido" TIMEZONE=../i18n/SUPPORTED

t "validate: idioma que não existe, vazio, sem UTF-8 ou com curinga"
recusa "idioma inválido: xx_XX.UTF-8" LOCALE=xx_XX.UTF-8
recusa "idioma inválido" LOCALE=
recusa "idioma inválido" LOCALE=pt_BR.UTF
recusa "idioma inválido" "LOCALE=.*"
recusa "idioma inválido" LOCALE=pt_BRxUTF-8

t "validate: teclado que não existe, vazio ou com curinga"
recusa "teclado inválido: klingon" KEYMAP=klingon
recusa "teclado inválido" KEYMAP=
recusa "teclado inválido" "KEYMAP=*"
recusa "teclado inválido" "KEYMAP=br-*"

t "validate: perfil de hardware desconhecido; vazio e vários valem"
recusa "perfil de hardware desconhecido: radeon" "HARDWARE=amd radeon"
recusa "perfil de hardware desconhecido: AMD" HARDWARE=AMD
valida HARDWARE=
eq 0 "$rc" "$out"
valida "HARDWARE=amd intel nvidia vm"
eq 0 "$rc" "$out"

t "validate: APPS, ENCRYPT e POWEROFF só aceitam sim ou nao"
recusa "ENCRYPT deve ser sim ou nao: yes" ENCRYPT=yes
recusa "ENCRYPT deve ser sim ou nao" ENCRYPT=Sim
recusa "ENCRYPT deve ser sim ou nao" ENCRYPT=
recusa "APPS deve ser sim ou nao: true" APPS=true
recusa "POWEROFF deve ser sim ou nao: 1" POWEROFF=1
valida APPS=sim ENCRYPT=sim POWEROFF=sim
eq 0 "$rc" "$out"

t "validate: não pergunta nada nem toca em disco"
eq 0 "$(perguntas)"
hasnt "$(<"$SBX/calls")" "wipefs"

# --- confirm_wipe ------------------------------------------------------------
revisa() { roda _com confirm_wipe "$@"; }

t "confirm_wipe: a revisão mostra as respostas e o que há hoje no disco, sem a senha"
answers=$SBX/respostas.conf
revisa
eq 0 "$rc" "$out"
has "$out" "Disco:      /dev/sdb (238,5G KINGSTON SA400)"
has "$out" "Hostname:   mk"
has "$out" "Usuário:    mark (sudo)"
has "$out" "Região:     America/Porto_Velho · br-abnt2 · pt_BR.UTF-8"
has "$out" "Drivers:    vm"
has "$out" "LUKS:       nao"
has "$out" "TUDO EM /dev/sdb SERÁ APAGADO"
has "$out" "sdb1 238,5G ext4 antigo"
hasnt "$out" "segredo-de-teste"

t "confirm_wipe: com respostas, CONFIRM=formatar ou o caminho do disco autorizam"
has "$out" "confirmado pelo arquivo de respostas"
revisa CONFIRM=/dev/sdb
eq 0 "$rc" "$out"

t "confirm_wipe: com respostas e sem CONFIRM certo, para antes de apagar"
for c in "" sim FORMATAR /dev/sda "formatar "; do
  revisa "CONFIRM=$c"
  eq 1 "$rc" "CONFIRM=$c"
  has "$out" "use CONFIRM=formatar"
done

t "confirm_wipe: com respostas, não lê nada da entrada"
printf 'formatar\n' >"$SBX/entrada"
entrada=$SBX/entrada
revisa CONFIRM=
eq 1 "$rc"
entrada=/dev/null

t "confirm_wipe: na tela, escrever formatar segue (maiúsculas e espaços ignorados)"
answers=''
entrada=$SBX/entrada
for palavra in formatar FORMATAR "  Formatar " "for matar"; do
  printf '%s\n' "$palavra" >"$SBX/entrada"
  revisa CONFIRM=
  eq 0 "$rc" "digitado: [$palavra] $out"
done

t "confirm_wipe: na tela, qualquer outra coisa cancela sem tocar no disco"
for palavra in "" sim s y formata formatar! /dev/sdb; do
  printf '%s\n' "$palavra" >"$SBX/entrada"
  revisa CONFIRM=formatar
  eq 1 "$rc" "digitado: [$palavra]"
  has "$out" "cancelado: nada foi alterado"
done
entrada=/dev/null
revisa CONFIRM=formatar
eq 1 "$rc" "sem ninguém para responder, cancela"

# --- preflight ---------------------------------------------------------------
for c in pacstrap arch-chroot genfstab sgdisk mkfs.fat mkfs.btrfs cryptsetup; do stub "$c"; done
stub mountpoint 'exit 1'
mkdir -p "$SBX/live/sys/firmware/efi"

t "preflight: root, UEFI, ferramentas e /mnt livre passam"
roda preflight
eq 0 "$rc" "$out"

t "preflight: sem root, para"
roda eval 'is_root() { return 1; }; preflight'
eq 1 "$rc"
has "$out" "rode como root"

t "preflight: live em BIOS legado, para e manda reiniciar pela entrada UEFI"
rmdir "$SBX/live/sys/firmware/efi"
roda preflight
eq 1 "$rc"
has "$out" "modo UEFI"
mkdir -p "$SBX/live/sys/firmware/efi"

t "preflight: alvo já montado, para e diz como desmontar"
stub mountpoint 'exit 0'
roda preflight
eq 1 "$rc"
has "$out" "já está montado"
has "$out" "umount -R $target"
stub mountpoint 'exit 1'

t "preflight: falta uma ferramenta do live, para dizendo qual"
# Um PATH só com os stubs: o genfstab desta máquina, se houver, não aparece.
rm -f "$SBX/bin/genfstab"
so_stubs() {
  PATH=$SBX/bin
  preflight
}
roda so_stubs
eq 1 "$rc"
has "$out" "falta genfstab no live"
stub genfstab

t "nenhuma destas funções formata, monta ou instala"
for c in wipefs sgdisk mkfs mount pacstrap; do not_called "$c "; done

# --- interview (sem gum: as mesmas perguntas em texto simples) --------------
stub lspci 'echo "00:01.0 VGA compatible controller: Red Hat, Inc. Virtio 1.0 GPU (rev 01)"'
entrevista() {
  interview
  mostra
}

t "interview: Enter aceita os padrões (hostname mk, região do Brasil, drivers detectados, apps sim, LUKS não)"
printf '%s\n' 2 "" mark senha1 senha1 "" "" "" "" >"$SBX/entrada"
entrada=$SBX/entrada
roda entrevista
eq 0 "$rc" "$out"
has "$out" 'DISK="/dev/sdb"'
has "$out" 'HOSTNAME_="mk"'
has "$out" 'USERNAME="mark"'
has "$out" 'PASSWORD="senha1"'
has "$out" 'TIMEZONE="America/Porto_Velho"'
has "$out" 'KEYMAP="br-abnt2"'
has "$out" 'LOCALE="pt_BR.UTF-8"'
has "$out" 'HARDWARE="amd"'
has "$out" 'APPS="sim"'
has "$out" 'ENCRYPT="nao"'

t "interview: a lista de discos não traz o pendrive do live"
hasnt "$out" "/dev/sdc"
has "$out" "1) /dev/sda"

t "interview: repete até o usuário e o hostname serem válidos e as senhas baterem"
printf '%s\n' 1 "meu pc" estacao Root ana "" s1 s2 s3 s3 "" "amd nvidia" n s >"$SBX/entrada"
roda entrevista
eq 0 "$rc" "$out"
has "$out" "use letras, números e -, sem espaços"
has "$out" "use letras minúsculas"
has "$out" "a senha não pode ser vazia"
has "$out" "as senhas não são iguais"
has "$out" 'DISK="/dev/sda"'
has "$out" 'HOSTNAME_="estacao"'
has "$out" 'USERNAME="ana"'
has "$out" 'PASSWORD="s3"'
has "$out" 'HARDWARE="amd nvidia"'
has "$out" 'APPS="nao"'
has "$out" 'ENCRYPT="sim"'

t "interview: opção de disco fora da lista para"
printf '%s\n' 9 >"$SBX/entrada"
roda entrevista
fails test "$rc" -eq 0
has "$out" "opção inválida"

t "interview: se a entrada acaba no meio, para em vez de repetir a pergunta para sempre"
printf '%s\n' 1 mk >"$SBX/entrada"
roda entrevista
eq 1 "$rc"
has "$out" "a entrada acabou no meio das perguntas"
printf '%s\n' 1 mk ana s1 >"$SBX/entrada"
roda entrevista
eq 1 "$rc"
has "$out" "a entrada acabou no meio das perguntas"

t "interview: sem nenhum disco além do pendrive, para"
printf '%s\n' '/dev/sdc disk 15376318464 14,3G SanDisk -' '/dev/sdc1 part 1509949440 1,4G - /dev/sdc' >"$SBX/discos"
roda entrevista
eq 1 "$rc"
has "$out" "nenhum disco encontrado"
entrada=/dev/null
