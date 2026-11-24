#!/usr/bin/env bash
# migrations/system/20261126-boot-gerenciado.sh: a entrada fixa do instalador
# antigo sai do limine.conf, o limine-update escreve a dele, e o arquivo
# anterior volta se o menu novo não fechar. Tudo numa ESP falsa em $R/boot.
. "$(dirname "$0")/../../lib.sh"
. "$TESTS/migrations/mig.sh" system/20261126-boot-gerenciado.sh

boot=$R/boot
conf=$boot/limine.conf
defaults=$R/etc/default/limine
id=3f2a9c # a pasta que os ganchos do limine criam na ESP (machine-id)

# A ESP de uma máquina instalada: o kernel na raiz (pacote linux), a cópia que
# os ganchos fazem e o limine de reserva.
esp() {
  rm -rf "${boot:?}"
  mkdir -p "$boot/EFI/BOOT" "$boot/$id/linux" "$boot/$id/snap1"
  touch "$boot/vmlinuz-linux" "$boot/initramfs-linux.img" "$boot/EFI/BOOT/BOOTX64.EFI" \
    "$boot/$id/linux/vmlinuz-linux" "$boot/$id/linux/initramfs-linux" \
    "$boot/$id/snap1/vmlinuz-linux" "$boot/$id/snap1/initramfs-linux"
  calls_reset
}
cabecalho() {
  cat <<'CONF'
# Menu de boot do Mk (limine).
timeout: 3
interface_branding: Mk
term_background: 0b0b0b
CONF
}
# O que o instalador até 20/11 gravava.
fixa() {
  cat <<'CONF'

/Mk
    protocol: linux
    path: boot():/vmlinuz-linux
    cmdline: root=UUID=abc rw quiet splash
    module_path: boot():/initramfs-linux.img
CONF
}
fixa_fallback() {
  cat <<'CONF'

/Mk (fallback)
    protocol: linux
    path: boot():/vmlinuz-linux
    cmdline: root=UUID=abc rw
    module_path: boot():/initramfs-linux-fallback.img
CONF
}
# O que os ganchos escrevem: a subentrada do kernel, com o resumo no fim.
sub_linux() {
  cat <<CONF
    //linux
        protocol: linux
        path: boot():/$id/linux/vmlinuz-linux#b2f1
        cmdline: root=UUID=abc rw quiet splash
        module_path: boot():/$id/linux/initramfs-linux#77aa
CONF
}
sub_snapshots() {
  cat <<CONF
    //Snapshots
        comment: escolha um snapshot
        ///2026-11-20 10:00
            protocol: linux
            path: boot():/$id/snap1/vmlinuz-linux
            cmdline: root=UUID=abc rw rootflags=subvol=/.snapshots/1/snapshot
            module_path: boot():/$id/snap1/initramfs-linux
CONF
}
efi_fallback() {
  cat <<'CONF'

/EFI fallback
    protocol: efi
    path: boot():/EFI/BOOT/BOOTX64.EFI
CONF
}

# O limine-update falso faz o que $SBX/limine-update.sh disser, sobre $LIMINE_CONF.
stub limine-update '. "$SBX/limine-update.sh"'
stub limine-snapper-sync
stub limine-enroll-config
export LIMINE_CONF=$conf
# Como o de verdade: põe a subentrada no grupo Mk que existir, ou cria o grupo.
gancho_normal() {
  sub_linux >"$SBX/sub"
  cat >"$SBX/limine-update.sh" <<'SH'
if grep -q '^/+\?Mk[[:space:]]*$' "$LIMINE_CONF"; then
  sed -i -e "/^\/+\?Mk[[:space:]]*\$/r $SBX/sub" "$LIMINE_CONF"
elif grep -q '^/EFI fallback$' "$LIMINE_CONF"; then
  # O grupo do sistema vem antes da entrada de reserva.
  { printf '/Mk\n'; cat "$SBX/sub"; echo; } >"$SBX/grupo"
  sed -i -e "/^\/EFI fallback\$/e cat $SBX/grupo" "$LIMINE_CONF"
else
  { printf '\n/Mk\n'; cat "$SBX/sub"; } >>"$LIMINE_CONF"
fi
SH
}
# Tira um comando do PATH da migração (o stub e o guarda), e devolve.
esconde() { mv "$SBX/bin/$1" "$SBX/$1.stub" && mv "$SBX/guard/$1" "$SBX/$1.guarda"; }
devolve() { mv "$SBX/$1.stub" "$SBX/bin/$1" && mv "$SBX/$1.guarda" "$SBX/guard/$1"; }

# ── instalador antigo ────────────────────────────────────────────────
t "instalador antigo: a entrada fixa sai e a do limine-update entra"
esp
touch "$boot/initramfs-linux-fallback.img"
{ cabecalho; fixa; fixa_fallback; efi_fallback; } >"$conf"
cp "$conf" "$SBX/antes"
gancho_normal
ok mig
called "limine-update"
depois=$(cat "$conf")
hasnt "$depois" "path: boot():/vmlinuz-linux"
hasnt "$depois" "initramfs-linux.img"
hasnt "$depois" "/Mk (fallback)"
has "$depois" "path: boot():/$id/linux/vmlinuz-linux#b2f1"
eq 1 "$(grep -c '^[[:space:]]*protocol: linux[[:space:]]*$' "$conf")"
eq 1 "$(grep -c '^/+\?Mk[[:space:]]*$' "$conf")" "um grupo Mk só"

t "instalador antigo: o cabeçalho e a 'EFI fallback' ficam como estavam"
while IFS= read -r linha; do
  ok grep -qxF -- "$linha" "$conf"
done < <(cabecalho)
eq "# Menu de boot do Mk (limine)." "$(head -1 "$conf")"
has "$depois" "$(efi_fallback)"

t "instalador antigo: o arquivo anterior fica guardado em limine.conf.mk-antes"
ok cmp -s "$SBX/antes" "$conf.mk-antes"

t "instalador antigo: o menu inicia sozinho (grupo aberto e default_entry na subentrada)"
called "limine-default $conf"
has "$depois" "/+Mk"
eq "default_entry: 2" "$(grep '^default_entry:' "$conf")"

t "instalador antigo: os snapshots são sincronizados em seguida"
called "limine-snapper-sync"

t "rodar de novo: nada a fazer, o limine-update não é chamado outra vez"
calls_reset
sem_mudanca mig
not_called "limine-update"
not_called "limine-default"

# ── já gerenciado ────────────────────────────────────────────────────
t "já gerenciado (sem a entrada fixa): não mexe no arquivo nem cria o .mk-antes"
esp
{ cabecalho; printf '\n/+Mk\n'; sub_linux; sub_snapshots; efi_fallback; } >"$conf"
gancho_normal
sem_mudanca mig
absent "$conf.mk-antes"
not_called "limine-update"
not_called "limine-default"

# ── híbrido ──────────────────────────────────────────────────────────
t "híbrido (entrada fixa + //Snapshots): só as linhas fixas saem, os snapshots ficam"
esp
{
  cabecalho
  printf '\n/+Mk\n    comment: Mk\n    protocol: linux\n    path: boot():/vmlinuz-linux\n'
  printf '    cmdline: root=UUID=abc rw quiet splash\n    module_path: boot():/initramfs-linux.img\n'
  sub_snapshots
  efi_fallback
} >"$conf"
gancho_normal
ok mig
depois=$(cat "$conf")
hasnt "$depois" "path: boot():/vmlinuz-linux"
hasnt "$depois" "boot():/initramfs-linux.img"
has "$depois" "$(sub_snapshots)"
has "$depois" "$(sub_linux)"
has "$depois" "$(efi_fallback)"
eq 1 "$(grep -c '^/+\?Mk[[:space:]]*$' "$conf")" "um grupo Mk só"

t "híbrido: o que não é do Mk (outro sistema no menu) não muda"
esp
{
  cabecalho
  fixa
  printf '\n/Windows\n    protocol: efi\n    path: boot():/EFI/Microsoft/bootmgfw.efi\n    comment: não mexa\n'
  efi_fallback
} >"$conf"
mkdir -p "$boot/EFI/Microsoft" && touch "$boot/EFI/Microsoft/bootmgfw.efi"
gancho_normal
ok mig
has "$(cat "$conf")" $'/Windows\n    protocol: efi\n    path: boot():/EFI/Microsoft/bootmgfw.efi\n    comment: não mexa'
has "$(cat "$conf")" "$(sub_linux)"

t "híbrido: uma entrada de outro nome com as mesmas linhas não é tocada"
esp
{
  cabecalho
  fixa
  printf '\n/Arch de reserva\n    protocol: linux\n    path: boot():/vmlinuz-linux\n    module_path: boot():/initramfs-linux.img\n'
} >"$conf"
gancho_normal
ok mig
has "$(cat "$conf")" $'/Arch de reserva\n    protocol: linux\n    path: boot():/vmlinuz-linux\n    module_path: boot():/initramfs-linux.img'

t "sobra só a 'EFI fallback' (o limine-update não escreveu nada): o arquivo anterior volta"
esp
{ cabecalho; fixa; efi_fallback; } >"$conf"
cp "$conf" "$SBX/antes"
: >"$SBX/limine-update.sh"
ok mig
ok cmp -s "$SBX/antes" "$conf"
has "$(cat "$ERR")" "aviso"
has "$(cat "$ERR")" "ficou como estava"

t "sobram só //Snapshots vazio e a 'EFI fallback': o arquivo anterior volta"
esp
{
  cabecalho
  printf '\n/+Mk\n    protocol: linux\n    path: boot():/vmlinuz-linux\n    module_path: boot():/initramfs-linux.img\n'
  printf '    //Snapshots\n        comment: nenhum ainda\n'
  efi_fallback
} >"$conf"
cp "$conf" "$SBX/antes"
: >"$SBX/limine-update.sh"
ok mig
ok cmp -s "$SBX/antes" "$conf"
has "$(cat "$ERR")" "aviso"

# ── falhas ───────────────────────────────────────────────────────────
t "o limine-update falha: o arquivo anterior volta, avisa e sai com 0"
esp
{ cabecalho; fixa; fixa_fallback; efi_fallback; } >"$conf"
touch "$boot/initramfs-linux-fallback.img"
cp "$conf" "$SBX/antes"
echo 'echo "lixo pela metade" >>"$LIMINE_CONF"; exit 1' >"$SBX/limine-update.sh"
ok mig
ok cmp -s "$SBX/antes" "$conf"
has "$(cat "$ERR")" "o limine-update falhou"
not_called "limine-default"
not_called "limine-snapper-sync"

t "depois da falha, uma rodada com o limine-update bom conserta"
gancho_normal
ok mig
hasnt "$(cat "$conf")" "path: boot():/vmlinuz-linux"
has "$(cat "$conf")" "$(sub_linux)"

t "o menu gerado aponta para um kernel que não está na ESP: o anterior volta"
esp
{ cabecalho; fixa; efi_fallback; } >"$conf"
cp "$conf" "$SBX/antes"
gancho_normal
mv "$boot/$id/linux/initramfs-linux" "$SBX/initramfs-sumido"
ok mig
ok cmp -s "$SBX/antes" "$conf"
has "$(cat "$ERR")" "não aponta para um kernel"

t "o limine-snapper-sync falhar não desfaz o menu novo"
esp
{ cabecalho; fixa; efi_fallback; } >"$conf"
gancho_normal
stub limine-snapper-sync 'exit 1'
ok mig
has "$(cat "$conf")" "$(sub_linux)"
hasnt "$(cat "$conf")" "path: boot():/vmlinuz-linux"
stub limine-snapper-sync

# ── onde não há o que fazer ──────────────────────────────────────────
t "o limine.conf da pasta do EFI tem a vez; o da raiz da ESP não é tocado"
esp
mkdir -p "$boot/EFI/limine"
{ cabecalho; fixa; efi_fallback; } >"$boot/EFI/limine/limine.conf"
{ cabecalho; fixa; } >"$conf"
cp "$conf" "$SBX/raiz"
export LIMINE_CONF=$boot/EFI/limine/limine.conf
gancho_normal
ok mig
ok cmp -s "$SBX/raiz" "$conf"
hasnt "$(cat "$boot/EFI/limine/limine.conf")" "path: boot():/vmlinuz-linux"
has "$(cat "$boot/EFI/limine/limine.conf")" "$(sub_linux)"
export LIMINE_CONF=$conf

t "sem limine.conf (outro gerenciador de boot): sai com 0 sem chamar nada"
esp
sem_mudanca mig
eq "" "$(cat "$SBX/calls")"

t "sem o limine-update (os ganchos não foram instalados): o menu fica como está"
esp
{ cabecalho; fixa; efi_fallback; } >"$conf"
esconde limine-update
sem_mudanca mig
absent "$conf.mk-antes"
devolve limine-update

t "sem o limine-snapper-sync: troca a entrada do mesmo jeito"
esp
{ cabecalho; fixa; efi_fallback; } >"$conf"
gancho_normal
esconde limine-snapper-sync
ok mig
has "$(cat "$conf")" "$(sub_linux)"
devolve limine-snapper-sync

# ── o limite de snapshots ────────────────────────────────────────────
t "MAX_SNAPSHOT_ENTRIES=5 vira auto, e o resto do /etc/default/limine fica"
esp
mkdir -p "$(dirname "$defaults")"
printf 'TARGET_OS_NAME="Mk"\nMAX_SNAPSHOT_ENTRIES=5\nENABLE_UKI=no\n' >"$defaults"
ok mig
eq $'TARGET_OS_NAME="Mk"\nMAX_SNAPSHOT_ENTRIES=auto\nENABLE_UKI=no' "$(cat "$defaults")"

t "de novo: o auto fica"
sem_mudanca mig

t "um limite escolhido pela pessoa (10, 50) não é trocado"
printf 'MAX_SNAPSHOT_ENTRIES=10\n' >"$defaults"
sem_mudanca mig
printf 'MAX_SNAPSHOT_ENTRIES=50\n' >"$defaults"
sem_mudanca mig

t "sem /etc/default/limine: não cria um"
mv "$defaults" "$SBX/defaults-sumido"
ok mig
absent "$defaults"

t "não instala nada"
not_called "pacman"
