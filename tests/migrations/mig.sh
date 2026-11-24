# Ajudante dos testes das migrações. Carregue depois do tests/lib.sh, com M já
# dito (a migração deste arquivo, "system/AAAAMMDD-assunto.sh"):
#
#   M=system/20261120-boot-mk.sh
#   . "$(dirname "$0")/../../lib.sh"
#   . "$TESTS/migrations/mig.sh"
#
# Monta o sistema falso em $SBX: a raiz em $R (a migração recebe MK_ROOT=$R,
# veja migrations/README.md) e o Mk instalado em $SHARE (cópia de system/ e
# config/ do repositório, com MK_SHARE=$SHARE). A migração roda com um PATH
# só de ferramentas básicas: fora elas, só existe o que o teste declarar com
# `stub` (e os guardas do lib.sh, que reprovam).
#
#   mig              roda a migração; a saída fica em $OUT e $ERR, o código em $rc
#   instalados a b   os pacotes que o pacman falso diz estarem instalados
#   estado           retrato de $R, $HOME e $SHARE (caminhos, tipos, conteúdo)
#   de_novo          caso pronto: rodar outra vez sai com 0 e não muda nada
#   sem_mudanca cmd  o comando sai com 0 e o retrato fica igual
# shellcheck shell=bash
[[ -n ${SBX:-} && -d $SBX && -n ${1:-} ]] || { echo "mig.sh: carregue o lib.sh antes e diga a migração" >&2; exit 1; }

M=$1
MIG=$REPO/migrations/$M
R=$SBX/root
SHARE=$SBX/share
OUT=$SBX/out ERR=$SBX/err
mkdir -p "$R" "$SHARE"
cp -r "$REPO/system" "$REPO/config" "$SHARE/"

# Os ajudantes do Mk que as migrações chamam. O limine-default sem argumento
# procura o limine.conf em /boot de verdade: aqui ele só roda com o arquivo
# dito. O make-icons lê /usr/share/icons/Papirus: aqui é um dublê, que falha
# como o de verdade quando $SBX/sem-papirus existe. O mktunes-links só roda
# com uma raiz que não seja /.
mv "$SHARE/system/mk/boot/limine-default" "$SHARE/system/mk/boot/limine-default.real"
cat >"$SHARE/system/mk/boot/limine-default" <<DUBLE
#!/usr/bin/env bash
printf '%s\n' "limine-default \$*" >>"$SBX/calls"
[[ \$# -gt 0 && \$1 == "$SBX"/* ]] || exit 0
exec bash "$SHARE/system/mk/boot/limine-default.real" "\$@"
DUBLE
cat >"$SHARE/system/mk/look/make-icons" <<DUBLE
#!/usr/bin/env bash
printf '%s\n' "make-icons \$*" >>"$SBX/calls"
[[ \${1:-} == "$SBX"/* ]] || exit 96
[[ ! -e "$SBX/sem-papirus" ]] || { echo "falta o papirus-icon-theme" >&2; exit 1; }
rm -rf "\$1" && mkdir -p "\$1" && echo "[Icon Theme]" >"\$1/index.theme"
DUBLE
mv "$SHARE/system/mk/mktunes-links" "$SHARE/system/mk/mktunes-links.real"
cat >"$SHARE/system/mk/mktunes-links" <<DUBLE
#!/usr/bin/env bash
printf '%s\n' "mktunes-links \$*" >>"$SBX/calls"
[[ \${1:-} == "$SBX"/* ]] || { echo "mktunes-links fora da caixa: '\${1:-}'" >>"$SBX/perigo"; exit 96; }
exec bash "$SHARE/system/mk/mktunes-links.real" "\$@"
DUBLE

# Só o básico no PATH da migração: o resto é stub ou guarda.
mkdir -p "$SBX/base"
for _c in bash sh env sed grep awk gawk cp mv rm mkdir rmdir cat head tail touch install find \
  dirname basename ln tr sort uniq chmod readlink sha256sum wc cut tee mktemp ls stat date \
  cmp diff true false xargs; do
  _p=$(PATH=/usr/bin:/bin command -v "$_c") && ln -sf "$_p" "$SBX/base/$_c"
done
unset _c _p
MIG_PATH=$SBX/bin:$SBX/guard:$SBX/base

mig() {
  [[ $R == "$SBX"/* && $SHARE == "$SBX"/* && $HOME == "$SBX"/* ]] || { echo "mig: raiz fora da caixa" >&2; exit 1; }
  env PATH="$MIG_PATH" MK_ROOT="$R" MK_SHARE="$SHARE" bash "$MIG" >"$OUT" 2>"$ERR"
}

# O pacman falso: -Q/-T respondem pela lista de `instalados`, -R tira dela, e
# o resto (que nenhuma migração deveria pedir) só fica registrado.
: >"$SBX/instalados"
instalados() { : >"$SBX/instalados"; (($# == 0)) || printf '%s\n' "$@" >"$SBX/instalados"; }
stub pacman 'case $1 in
  -Q* | -T*) shift; for p in "$@"; do grep -qx -- "$p" "$SBX/instalados" || exit 1; done ;;
  -R*) shift; for p in "$@"; do [[ $p == -* ]] || sed -i "/^$p\$/d" "$SBX/instalados"; done ;;
esac
exit 0'

estado() {
  (
    cd "$SBX" || exit 1
    find root home share -mindepth 1 -printf '%p %y %m %l\n' | sort
    find root home share -type f -print0 | sort -z | xargs -0 -r sha256sum
  )
}

sem_mudanca() {
  local antes
  antes=$(estado)
  ok "$@"
  eq "$antes" "$(estado)" "o sistema mudou"
}

de_novo() {
  t "rodar de novo sai com 0 e não muda nada"
  sem_mudanca mig
}
